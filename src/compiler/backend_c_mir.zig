// ─────────────────────────────────────────────────────────────────────────────
// Imports and public API
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const hir_checker = @import("hir_checker.zig");
const mir = @import("mir.zig");
const mir_lowering = @import("mir_lowering.zig");
const mir_validator = @import("mir_validator.zig");
const parser_model = @import("parser.zig");
const semantics = @import("semantics.zig");
const source_model = @import("source.zig");
const types = @import("types.zig");

pub const EmitError = error{InvalidExecutable} || std.mem.Allocator.Error;

// ─────────────────────────────────────────────────────────────────────────────
// Backend context
// ─────────────────────────────────────────────────────────────────────────────

const BackendContext = struct {
    module: *const semantics.SemanticModule,
    mir_module: *const mir.MirModule,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
};

pub fn emitExecutableFromMir(
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
    mir_module: *const mir.MirModule,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
) EmitError![]const u8 {
    mir_validator.validateModule(allocator, semantic_module, mir_module, diagnostic_bag) catch |err| switch (err) {
        error.InvalidMirModule => return error.InvalidExecutable,
        error.OutOfMemory => return error.OutOfMemory,
    };

    var output = std.Io.Writer.Allocating.init(allocator);
    errdefer output.deinit();
    const writer = &output.writer;
    const ctx = BackendContext{
        .module = semantic_module,
        .mir_module = mir_module,
        .diagnostic_bag = diagnostic_bag,
    };

    const emitted_enum_layouts = try emitEnumLayouts(writer, &ctx);
    if (emitted_enum_layouts and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    if (mir_module.store.functions.items.len > 1) {
        for (mir_module.store.functions.items, 0..) |function, index| {
            try emitPrototype(writer, &ctx, .{ .index = @intCast(index) }, function);
        }
        try writer.writeByte('\n');
    }

    for (mir_module.store.functions.items, 0..) |function, index| {
        if (index != 0) try writer.writeByte('\n');
        try emitFunction(writer, &ctx, .{ .index = @intCast(index) }, function);
    }
    return output.toOwnedSlice();
}

// ─────────────────────────────────────────────────────────────────────────────
// Enum layout emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitEnumLayouts(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var emitted_any = false;
    for (ctx.module.hir.enums.items, 0..) |enum_decl, index| {
        if (emitted_any) try writer.writeByte('\n');
        try emitEnumLayout(writer, ctx, .{ .index = @intCast(index) }, enum_decl);
        emitted_any = true;
    }
    return emitted_any;
}

/// P5-M1 enum layout emission is backend-local and intentionally not ABI-stable.
/// The MIR C backend renders every collected enum as `tag` plus payload union only
/// when the enum is non-empty and all payload fields are backend v0 scalar types
/// (`int` or `bool`, both emitted as C `int`). Struct payloads, invalid TypeIds,
/// empty enums, and other nominal payloads report CON0045 when layout is attempted.
fn emitEnumLayout(writer: anytype, ctx: *const BackendContext, enum_id: hir.EnumId, enum_decl: hir.HirEnum) EmitError!void {
    try requireSupportedEnumLayout(ctx, enum_id, null);

    const has_payload = enumHasPayload(ctx, enum_decl);
    try writer.writeAll("typedef struct {\n");
    try writer.writeAll("    int tag;\n");
    if (has_payload) {
        try writer.writeAll("    union {\n");
        for (enum_decl.variants, 0..) |variant_id, variant_index| {
            const variant = ctx.module.hir.getVariant(variant_id);
            if (variant.payload_fields.len == 0) continue;
            try writer.writeAll("        struct {\n");
            for (variant.payload_fields, 0..) |payload_id, payload_index| {
                const payload_field = ctx.module.hir.getEnumPayloadField(payload_id);
                try writer.writeAll("            ");
                try emitCType(writer, ctx, payload_field.type_id, payload_field.span);
                try writer.writeByte(' ');
                try emitEnumPayloadFieldName(writer, ctx.module, payload_field.name, payload_index);
                try writer.writeAll(";\n");
            }
            try writer.writeAll("        } ");
            try emitEnumVariantPayloadName(writer, ctx.module, variant.name, variant_index);
            try writer.writeAll(";\n");
        }
        try writer.writeAll("    } payload;\n");
    }
    try writer.writeAll("} ");
    try emitEnumTypeName(writer, ctx.module, enum_decl.name);
    try writer.writeAll(";\n");
}

fn enumHasPayload(ctx: *const BackendContext, enum_decl: hir.HirEnum) bool {
    for (enum_decl.variants) |variant_id| {
        if (ctx.module.hir.getVariant(variant_id).payload_fields.len != 0) return true;
    }
    return false;
}

fn requireSupportedEnumLayout(ctx: *const BackendContext, enum_id: hir.EnumId, span: ?diagnostics.SourceSpan) EmitError!void {
    if (enum_id.index >= ctx.module.hir.enums.items.len) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }

    const enum_decl = ctx.module.hir.getEnum(enum_id);
    if (enum_decl.variants.len == 0) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }

    for (enum_decl.variants) |variant_id| {
        const variant = ctx.module.hir.getVariant(variant_id);
        for (variant.payload_fields) |payload_id| {
            const payload_field = ctx.module.hir.getEnumPayloadField(payload_id);
            if (!isSupportedEnumPayloadType(ctx, payload_field.type_id)) {
                try reportUnsupportedCType(ctx, payload_field.span);
                return error.InvalidExecutable;
            }
        }
    }
}

fn isSupportedEnumPayloadType(ctx: *const BackendContext, type_id: types.TypeId) bool {
    if (!ctx.module.types.contains(type_id)) return false;
    return switch (ctx.module.types.kind(type_id)) {
        .int, .bool => true,
        .void, .struct_type, .enum_type => false,
    };
}

// ─────────────────────────────────────────────────────────────────────────────
// Function/prototype emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitPrototype(writer: anytype, ctx: *const BackendContext, function_id: mir.MirFunctionId, function: mir.MirFunction) EmitError!void {
    _ = function_id;
    try emitCType(writer, ctx, function.return_type, function.source_span);
    try writer.writeByte(' ');
    try emitFunctionName(writer, ctx.module, function.name);
    try writer.writeByte('(');
    try emitParamList(writer, ctx, function, function.params);
    try writer.writeAll(");\n");
}

fn emitFunction(writer: anytype, ctx: *const BackendContext, function_id: mir.MirFunctionId, function: mir.MirFunction) EmitError!void {
    _ = function_id;
    try emitCType(writer, ctx, function.return_type, function.source_span);
    try writer.writeByte(' ');
    try emitFunctionName(writer, ctx.module, function.name);
    try writer.writeByte('(');
    try emitParamList(writer, ctx, function, function.params);
    try writer.writeAll(") {\n");

    for (function.locals) |local_id| {
        const local = ctx.mir_module.store.getLocal(local_id);
        try writer.writeAll("    ");
        try emitCType(writer, ctx, local.type_id, local.source_span);
        try writer.writeByte(' ');
        try emitLocalName(writer, ctx.module, ctx.mir_module, local_id);
        try writer.writeAll(";\n");
    }
    if (function.locals.len != 0) try writer.writeByte('\n');

    for (function.blocks) |block_id| {
        try emitBlock(writer, ctx, block_id);
    }

    try writer.writeAll("}\n");
}

// ─────────────────────────────────────────────────────────────────────────────
// Block/statement/terminator emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitBlock(writer: anytype, ctx: *const BackendContext, block_id: mir.MirBlockId) EmitError!void {
    try emitBlockLabel(writer, block_id);
    try writer.writeAll(":\n");

    const block = ctx.mir_module.store.getBlock(block_id);
    for (block.statements) |statement| {
        try emitStatement(writer, ctx, statement);
    }
    try emitTerminator(writer, ctx, block.terminator.?);
}

fn emitStatement(writer: anytype, ctx: *const BackendContext, statement: mir.MirStatement) EmitError!void {
    switch (statement.kind) {
        .assign => |assignment| {
            switch (assignment.rvalue) {
                .enum_constructor => |constructor| try emitEnumConstructorAssignment(writer, ctx, assignment.place, constructor),
                else => {
                    try writer.writeAll("    ");
                    try emitPlace(writer, ctx, assignment.place);
                    try writer.writeAll(" = ");
                    try emitRvalue(writer, ctx, assignment.rvalue);
                    try writer.writeAll(";\n");
                },
            }
        },
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Enum constructor/tag/payload emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitEnumConstructorAssignment(writer: anytype, ctx: *const BackendContext, place: mir.MirPlace, constructor: anytype) EmitError!void {
    const enum_decl = ctx.module.hir.getEnum(constructor.enum_id);
    const variant = ctx.module.hir.getVariant(constructor.variant_id);
    const variant_index = enumVariantIndex(ctx, enum_decl, constructor.variant_id) orelse return error.InvalidExecutable;

    try writer.writeAll("    ");
    try emitPlace(writer, ctx, place);
    try writer.writeAll(".tag = ");
    try writer.print("{d};\n", .{variant_index});

    for (constructor.args, variant.payload_fields, 0..) |arg, payload_id, payload_index| {
        const payload_field = ctx.module.hir.getEnumPayloadField(payload_id);
        try writer.writeAll("    ");
        try emitPlace(writer, ctx, place);
        try writer.writeAll(".payload.");
        try emitEnumVariantPayloadName(writer, ctx.module, variant.name, variant_index);
        try writer.writeByte('.');
        try emitEnumPayloadFieldName(writer, ctx.module, payload_field.name, payload_index);
        try writer.writeAll(" = ");
        try emitOperand(writer, ctx, arg);
        try writer.writeAll(";\n");
    }
}

fn enumVariantIndex(ctx: *const BackendContext, enum_decl: hir.HirEnum, variant_id: hir.VariantId) ?usize {
    _ = ctx;
    for (enum_decl.variants, 0..) |candidate, index| {
        if (candidate.index == variant_id.index) return index;
    }
    return null;
}

fn enumPayloadFieldIndex(ctx: *const BackendContext, variant: hir.HirVariant, payload_id: hir.EnumPayloadFieldId) ?usize {
    _ = ctx;
    for (variant.payload_fields, 0..) |candidate, index| {
        if (candidate.index == payload_id.index) return index;
    }
    return null;
}

fn emitTerminator(writer: anytype, ctx: *const BackendContext, terminator: mir.MirTerminator) EmitError!void {
    switch (terminator.kind) {
        .goto => |target| {
            try writer.writeAll("    goto ");
            try emitBlockLabel(writer, target);
            try writer.writeAll(";\n");
        },
        .return_ => |maybe_operand| {
            try writer.writeAll("    return");
            if (maybe_operand) |operand| {
                try writer.writeByte(' ');
                try emitOperand(writer, ctx, operand);
            }
            try writer.writeAll(";\n");
        },
        .switch_bool => |switch_bool| {
            try writer.writeAll("    if (");
            try emitOperand(writer, ctx, switch_bool.discriminant);
            try writer.writeAll(") goto ");
            try emitBlockLabel(writer, switch_bool.true_target);
            try writer.writeAll("; else goto ");
            try emitBlockLabel(writer, switch_bool.false_target);
            try writer.writeAll(";\n");
        },
        .switch_int => |switch_int| {
            try writer.writeAll("    switch (");
            try emitOperand(writer, ctx, switch_int.discriminant);
            try writer.writeAll(") {\n");
            for (switch_int.cases) |case| {
                try writer.writeAll("        case ");
                try writer.writeAll(case.value);
                try writer.writeAll(": goto ");
                try emitBlockLabel(writer, case.target);
                try writer.writeAll(";\n");
            }
            try writer.writeAll("        default: goto ");
            try emitBlockLabel(writer, switch_int.default_target);
            try writer.writeAll(";\n");
            try writer.writeAll("    }\n");
        },
        .@"unreachable" => try writer.writeAll("    __builtin_unreachable();\n"),
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Rvalue/operand/place emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitRvalue(writer: anytype, ctx: *const BackendContext, rvalue: mir.MirRvalue) EmitError!void {
    switch (rvalue) {
        .use => |operand| try emitOperand(writer, ctx, operand),
        .unary => |unary| {
            try writer.writeAll(unary.op.lexeme());
            try emitOperand(writer, ctx, unary.operand);
        },
        .address_of => |place| {
            try writer.writeByte('&');
            try emitPlace(writer, ctx, place);
        },
        .deref => |operand| {
            try writer.writeByte('*');
            try emitOperand(writer, ctx, operand);
        },
        .binary => |binary| {
            try writer.writeByte('(');
            try emitOperand(writer, ctx, binary.left);
            try writer.writeByte(' ');
            try writer.writeAll(binary.op.lexeme());
            try writer.writeByte(' ');
            try emitOperand(writer, ctx, binary.right);
            try writer.writeByte(')');
        },
        .call => |call| {
            const callee = ctx.module.hir.getFunction(call.function);
            try emitFunctionName(writer, ctx.module, callee.name);
            try writer.writeByte('(');
            for (call.args, 0..) |arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try emitOperand(writer, ctx, arg);
            }
            try writer.writeByte(')');
        },
        .enum_constructor => unreachable,
        .enum_tag => |operand| {
            try emitOperand(writer, ctx, operand);
            try writer.writeAll(".tag");
        },
        .enum_payload_field => |payload| {
            const payload_field = ctx.module.hir.getEnumPayloadField(payload.payload_field);
            const variant = ctx.module.hir.getVariant(payload_field.parent);
            const enum_decl = ctx.module.hir.getEnum(variant.parent);
            const variant_index = enumVariantIndex(ctx, enum_decl.*, payload_field.parent) orelse return error.InvalidExecutable;
            const payload_index = enumPayloadFieldIndex(ctx, variant.*, payload.payload_field) orelse return error.InvalidExecutable;
            try emitOperand(writer, ctx, payload.enum_operand);
            try writer.writeAll(".payload.");
            try emitEnumVariantPayloadName(writer, ctx.module, variant.name, variant_index);
            try writer.writeByte('.');
            try emitEnumPayloadFieldName(writer, ctx.module, payload_field.name, payload_index);
        },
    }
}

fn emitOperand(writer: anytype, ctx: *const BackendContext, operand: mir.MirOperand) EmitError!void {
    switch (operand) {
        .int_literal => |text| try writer.writeAll(text),
        .bool_literal => |value| try writer.writeAll(if (value) "1" else "0"),
        .copy => |place| try emitPlace(writer, ctx, place),
    }
}

fn emitPlace(writer: anytype, ctx: *const BackendContext, place: mir.MirPlace) EmitError!void {
    switch (place) {
        .local => |local_id| try emitLocalName(writer, ctx.module, ctx.mir_module, local_id),
    }
}

fn emitParamList(writer: anytype, ctx: *const BackendContext, function: mir.MirFunction, params: []const mir.MirLocalId) EmitError!void {
    _ = function;
    if (params.len == 0) {
        try writer.writeAll("void");
        return;
    }
    for (params, 0..) |local_id, index| {
        const local = ctx.mir_module.store.getLocal(local_id);
        if (index != 0) try writer.writeAll(", ");
        try emitCType(writer, ctx, local.type_id, local.source_span);
        try writer.writeByte(' ');
        try emitLocalName(writer, ctx.module, ctx.mir_module, local_id);
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// C type rendering
// ─────────────────────────────────────────────────────────────────────────────

fn emitCType(writer: anytype, ctx: *const BackendContext, type_id: types.TypeId, span: ?diagnostics.SourceSpan) EmitError!void {
    if (!ctx.module.types.contains(type_id)) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }

    switch (ctx.module.types.kind(type_id)) {
        .void => try writer.writeAll("void"),
        .int, .bool => try writer.writeAll("int"),
        .enum_type => |enum_id| {
            try requireSupportedEnumLayout(ctx, enum_id, span);
            try emitEnumTypeName(writer, ctx.module, ctx.module.hir.getEnum(enum_id).name);
        },
        .pointer => |pointer| {
            try emitCType(writer, ctx, pointer.pointee, span);
            try writer.writeByte('*');
        },
        .struct_type => {
            try reportUnsupportedCType(ctx, span);
            return error.InvalidExecutable;
        },
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Diagnostics
// ─────────────────────────────────────────────────────────────────────────────

fn reportUnsupportedCType(ctx: *const BackendContext, span: ?diagnostics.SourceSpan) !void {
    if (ctx.diagnostic_bag) |bag| {
        try bag.append(diagnostics.unsupportedCBackendType(span orelse hir.synthetic_span));
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// C name rendering / escaping
// ─────────────────────────────────────────────────────────────────────────────

fn emitEnumTypeName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_enum_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitEnumVariantPayloadName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId, variant_index: usize) !void {
    try writer.writeAll("cpt_v_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
    try writer.print("_{d}", .{variant_index});
}

fn emitEnumPayloadFieldName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId, payload_index: usize) !void {
    try writer.writeAll("cpt_pf_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
    try writer.print("_{d}", .{payload_index});
}

fn emitFunctionName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    const source_name = module.interner.text(symbol);
    if (std.mem.eql(u8, source_name, "main")) {
        try writer.writeAll("main");
        return;
    }
    try writer.writeAll("cpt_f_");
    try emitEscapedIdentifierComponent(writer, source_name);
}

fn emitLocalName(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, local_id: mir.MirLocalId) !void {
    const local = mir_module.store.getLocal(local_id);
    switch (local.kind) {
        .param => {
            try writer.writeAll("cpt_p_");
            try emitLocalSymbolComponent(writer, module, local.name);
            try writer.print("_{d}", .{local_id.index});
        },
        .user => {
            try writer.writeAll("cpt_l_");
            try emitLocalSymbolComponent(writer, module, local.name);
            try writer.print("_{d}", .{local_id.index});
        },
        .temp => try writer.print("cpt_t_{d}", .{local_id.index}),
    }
}

fn emitLocalSymbolComponent(writer: anytype, module: *const semantics.SemanticModule, maybe_symbol: ?hir.SymbolId) !void {
    if (maybe_symbol) |symbol| {
        try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
    } else {
        try writer.writeAll("anon");
    }
}

fn emitBlockLabel(writer: anytype, block_id: mir.MirBlockId) !void {
    try writer.print("cpt_bb_{d}", .{block_id.index});
}

fn emitEscapedIdentifierComponent(writer: anytype, source_name: []const u8) !void {
    const needs_keyword_suffix = isCKeyword(source_name);
    if (source_name.len == 0) {
        try writer.writeAll("_empty");
    } else {
        const first = source_name[0];
        if (isAsciiDigit(first)) try writer.writeByte('_');
        for (source_name) |byte| {
            if (isCIdentifierComponentByte(byte)) {
                try writer.writeByte(byte);
            } else {
                try writer.print("_x{X:0>2}", .{byte});
            }
        }
    }
    if (needs_keyword_suffix) try writer.writeByte('_');
}

fn isCIdentifierComponentByte(byte: u8) bool {
    return isAsciiLetter(byte) or isAsciiDigit(byte) or byte == '_';
}

fn isAsciiLetter(byte: u8) bool {
    return (byte >= 'a' and byte <= 'z') or (byte >= 'A' and byte <= 'Z');
}

fn isAsciiDigit(byte: u8) bool {
    return byte >= '0' and byte <= '9';
}

fn isCKeyword(text: []const u8) bool {
    const keywords = [_][]const u8{
        "auto",     "break",  "case",   "char",           "const",    "continue", "default",  "do",
        "double",   "else",   "enum",   "extern",         "float",    "for",      "goto",     "if",
        "inline",   "int",    "long",   "register",       "restrict", "return",   "short",    "signed",
        "sizeof",   "static", "struct", "switch",         "typedef",  "union",    "unsigned", "void",
        "volatile", "while",  "_Bool",  "_Static_assert",
    };
    for (keywords) |keyword| {
        if (std.mem.eql(u8, text, keyword)) return true;
    }
    return false;
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

fn emitForTest(source_text: []const u8) ![]const u8 {
    var parse_diagnostics = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, "test.concept", source_text);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics.collectTopLevelDeclarations(std.testing.allocator, unit, &check_diagnostics);
    defer module.deinit();

    try hir_checker.checkExecutable(std.testing.allocator, &module, &check_diagnostics);

    var mir_module = try mir_lowering.lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const c_source = try emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &check_diagnostics);
    errdefer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 0), check_diagnostics.count());
    return c_source;
}

fn expectEmit(source_text: []const u8, expected: []const u8) !void {
    const c_source = try emitForTest(source_text);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(expected, c_source);
}

fn expectEmitCorpus(comptime source_path: []const u8, comptime expected_path: []const u8) !void {
    try expectEmit(@embedFile(source_path), @embedFile(expected_path));
}

test "MIR C backend emits return literal" {
    try expectEmit(
        "module Main; int main() { return 0; }",
        "int main(void) {\ncpt_bb_0:\n    return 0;\n}\n",
    );
}

test "MIR C backend emits arithmetic" {
    try expectEmit(
        "module Main; int main() { return 1 + 2 * 3; }",
        "int main(void) {\n    int cpt_t_0;\n    int cpt_t_1;\n\ncpt_bb_0:\n    cpt_t_0 = (2 * 3);\n    cpt_t_1 = (1 + cpt_t_0);\n    return cpt_t_1;\n}\n",
    );
}

test "MIR C backend emits locals and assignment" {
    try expectEmit(
        "module Main; int main() { int x = 1; x = x + 2; return x; }",
        "int main(void) {\n    int cpt_l_x_0;\n    int cpt_t_1;\n\ncpt_bb_0:\n    cpt_l_x_0 = 1;\n    cpt_t_1 = (cpt_l_x_0 + 2);\n    cpt_l_x_0 = cpt_t_1;\n    return cpt_l_x_0;\n}\n",
    );
}

test "MIR C backend emits function call" {
    try expectEmit(
        "module Main; int add(int a, int b) { return a + b; } int main() { return add(1, 2); }",
        "int cpt_f_add(int cpt_p_a_0, int cpt_p_b_1);\nint main(void);\n\nint cpt_f_add(int cpt_p_a_0, int cpt_p_b_1) {\n    int cpt_t_2;\n\ncpt_bb_0:\n    cpt_t_2 = (cpt_p_a_0 + cpt_p_b_1);\n    return cpt_t_2;\n}\n\nint main(void) {\n    int cpt_t_3;\n\ncpt_bb_1:\n    cpt_t_3 = cpt_f_add(1, 2);\n    return cpt_t_3;\n}\n",
    );
}

test "MIR C backend emits if else labels" {
    try expectEmit(
        "module Main; int main() { if (false) { return 1; } else { return 7; } }",
        "int main(void) {\ncpt_bb_0:\n    if (0) goto cpt_bb_1; else goto cpt_bb_3;\ncpt_bb_1:\n    return 1;\ncpt_bb_2:\n    __builtin_unreachable();\ncpt_bb_3:\n    return 7;\n}\n",
    );
}

test "MIR C backend emits while count loop" {
    try expectEmit(
        "module Main; int main() { int x = 0; while (x < 7) { x = x + 1; } return x; }",
        "int main(void) {\n    int cpt_l_x_0;\n    int cpt_t_1;\n    int cpt_t_2;\n\ncpt_bb_0:\n    cpt_l_x_0 = 0;\n    goto cpt_bb_1;\ncpt_bb_1:\n    cpt_t_1 = (cpt_l_x_0 < 7);\n    if (cpt_t_1) goto cpt_bb_2; else goto cpt_bb_3;\ncpt_bb_2:\n    cpt_t_2 = (cpt_l_x_0 + 1);\n    cpt_l_x_0 = cpt_t_2;\n    goto cpt_bb_1;\ncpt_bb_3:\n    return cpt_l_x_0;\n}\n",
    );
}

test "MIR C backend emits int match default" {
    try expectEmit(
        "module Main; int main() { int x = 3; match (x) { 1 => return 10; 2 => return 7; _ => return 0; } }",
        "int main(void) {\n    int cpt_l_x_0;\n\ncpt_bb_0:\n    cpt_l_x_0 = 3;\n    switch (cpt_l_x_0) {\n        case 1: goto cpt_bb_2;\n        case 2: goto cpt_bb_3;\n        default: goto cpt_bb_4;\n    }\ncpt_bb_1:\n    __builtin_unreachable();\ncpt_bb_2:\n    return 10;\ncpt_bb_3:\n    return 7;\ncpt_bb_4:\n    return 0;\n}\n",
    );
}

test "MIR C backend emits bool match" {
    try expectEmit(
        "module Main; int main() { bool ok = true; match (ok) { true => return 7; false => return 0; } return 0; }",
        "int main(void) {\n    int cpt_l_ok_0;\n\ncpt_bb_0:\n    cpt_l_ok_0 = 1;\n    if (cpt_l_ok_0) goto cpt_bb_2; else goto cpt_bb_3;\ncpt_bb_1:\n    return 0;\ncpt_bb_2:\n    return 7;\ncpt_bb_3:\n    return 0;\n}\n",
    );
}

fn newTestModule() !semantics.SemanticModule {
    return semantics.SemanticModule.init(std.testing.allocator);
}

fn internForTest(module: *semantics.SemanticModule, text: []const u8) !hir.SymbolId {
    return module.interner.intern(text);
}

fn addHirFunctionForTest(module: *semantics.SemanticModule, name: []const u8, return_type: types.TypeId) !hir.FunctionId {
    return module.hir.addFunction(try internForTest(module, name), return_type, hir.synthetic_span);
}

fn addVoidMainMirForTest(module: *semantics.SemanticModule, mir_module: *mir.MirModule) !void {
    const main_hir = try addHirFunctionForTest(module, "main", module.types.voidType());
    const main_mir = try mir_module.store.addFunction(main_hir, try internForTest(module, "main"), module.types.voidType(), hir.synthetic_span);
    const block = try mir_module.store.addBlock(main_mir, hir.synthetic_span);
    try mir_module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(null) });
}

fn emitManualModuleForTest(module: *semantics.SemanticModule, mir_module: *const mir.MirModule) ![]const u8 {
    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    const c_source = try emitExecutableFromMir(std.testing.allocator, module, mir_module, &diagnostic_bag);
    errdefer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 0), diagnostic_bag.count());
    return c_source;
}

test "MIR C backend corpus snapshot: phase4 sum loop" {
    try expectEmitCorpus("../../tests/corpus/phase4/mir_c_sum_loop.concept", "../../tests/corpus/phase4/mir_c_sum_loop.c.expected");
}
test "MIR C backend corpus snapshot: phase4 if match" {
    try expectEmitCorpus("../../tests/corpus/phase4/mir_c_if_match.concept", "../../tests/corpus/phase4/mir_c_if_match.c.expected");
}

test "MIR C backend corpus snapshot: phase5 enum payload layout" {
    try expectEmitCorpus("../../tests/corpus/phase5/enum_constructor.concept", "../../tests/corpus/phase5/enum_payload_layout.c.expected");
}

test "MIR C backend corpus snapshot: phase5 enum match payload" {
    try expectEmitCorpus("../../tests/corpus/phase5/enum_match_payload.concept", "../../tests/corpus/phase5/enum_match_payload.c.expected");
}

test "MIR C backend corpus snapshot: phase5 result try" {
    try expectEmitCorpus("../../tests/corpus/phase5/result_try_failure.concept", "../../tests/corpus/phase5/result_try.c.expected");
}

test "MIR C backend emits void helper prototype with backend-owned function name" {
    try expectEmit(
        "module Main; void helper() { return; } int main() { return 0; }",
        "void cpt_f_helper(void);\nint main(void);\n\nvoid cpt_f_helper(void) {\ncpt_bb_0:\n    return;\n}\n\nint main(void) {\ncpt_bb_1:\n    return 0;\n}\n",
    );
}

test "MIR C backend escapes keywords and invalid identifier bytes in backend names" {
    var module = try newTestModule();
    defer module.deinit();

    const int_type = module.types.intType();
    const callee_hir = try addHirFunctionForTest(&module, "int", int_type);
    _ = try module.hir.addParam(callee_hir, try internForTest(&module, "a-b"), int_type, hir.synthetic_span);
    const main_hir = try addHirFunctionForTest(&module, "main", int_type);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();

    const callee_mir = try mir_module.store.addFunction(callee_hir, try internForTest(&module, "int"), int_type, hir.synthetic_span);
    const param = try mir_module.store.addLocal(callee_mir, try internForTest(&module, "a-b"), .param, int_type, hir.synthetic_span);
    const user = try mir_module.store.addLocal(callee_mir, try internForTest(&module, "return"), .user, int_type, hir.synthetic_span);
    const temp = try mir_module.store.addLocal(callee_mir, null, .temp, int_type, hir.synthetic_span);
    const callee_block = try mir_module.store.addBlock(callee_mir, hir.synthetic_span);
    try mir_module.store.appendStatement(callee_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = user }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(.{ .local = param }))) });
    try mir_module.store.appendStatement(callee_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, mir.MirRvalue.binaryOp(.add, mir.MirOperand.copyPlace(.{ .local = user }), try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try mir_module.store.setTerminator(callee_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp })) });

    const main_mir = try mir_module.store.addFunction(main_hir, try internForTest(&module, "main"), int_type, hir.synthetic_span);
    const call_temp = try mir_module.store.addLocal(main_mir, null, .temp, int_type, hir.synthetic_span);
    const main_block = try mir_module.store.addBlock(main_mir, hir.synthetic_span);
    const args = [_]mir.MirOperand{try mir.MirOperand.intLiteral(std.testing.allocator, "2")};
    try mir_module.store.appendStatement(main_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = call_temp }, try mir.MirRvalue.callFunction(std.testing.allocator, callee_hir, &args)) });
    try mir_module.store.setTerminator(main_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = call_temp })) });

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    const c_source = try emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 0), diagnostic_bag.count());
    try std.testing.expectEqualStrings(
        "int cpt_f_int_(int cpt_p_a_x2D_b_0);\nint main(void);\n\nint cpt_f_int_(int cpt_p_a_x2D_b_0) {\n    int cpt_l_return__1;\n    int cpt_t_2;\n\ncpt_bb_0:\n    cpt_l_return__1 = cpt_p_a_x2D_b_0;\n    cpt_t_2 = (cpt_l_return__1 + 1);\n    return cpt_t_2;\n}\n\nint main(void) {\n    int cpt_t_3;\n\ncpt_bb_1:\n    cpt_t_3 = cpt_f_int_(2);\n    return cpt_t_3;\n}\n",
        c_source,
    );
}

test "MIR C backend emits tag-only enum layout" {
    var module = try newTestModule();
    defer module.deinit();

    const enum_id = try module.hir.addEnum(try internForTest(&module, "TokenKind"), false);
    _ = try module.types.addEnumType(enum_id);
    _ = try module.hir.addVariant(enum_id, try internForTest(&module, "Identifier"), hir.synthetic_span);
    _ = try module.hir.addVariant(enum_id, try internForTest(&module, "End"), hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int tag;\n} cpt_enum_TokenKind;\n\nvoid main(void) {\ncpt_bb_0:\n    return;\n}\n",
        c_source,
    );
}

test "MIR C backend emits payload enum layout with int fields" {
    var module = try newTestModule();
    defer module.deinit();

    const enum_id = try module.hir.addEnum(try internForTest(&module, "ParseResult"), false);
    _ = try module.types.addEnumType(enum_id);
    const ok = try module.hir.addVariant(enum_id, try internForTest(&module, "Ok"), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(ok, try internForTest(&module, "value"), module.types.intType(), hir.synthetic_span);
    const err = try module.hir.addVariant(enum_id, try internForTest(&module, "Err"), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(err, try internForTest(&module, "code"), module.types.intType(), hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int tag;\n    union {\n        struct {\n            int cpt_pf_value_0;\n        } cpt_v_Ok_0;\n        struct {\n            int cpt_pf_code_0;\n        } cpt_v_Err_1;\n    } payload;\n} cpt_enum_ParseResult;\n\nvoid main(void) {\ncpt_bb_0:\n    return;\n}\n",
        c_source,
    );
}

test "MIR C backend emits payload enum layout with bool fields" {
    var module = try newTestModule();
    defer module.deinit();

    const enum_id = try module.hir.addEnum(try internForTest(&module, "Check"), false);
    _ = try module.types.addEnumType(enum_id);
    const done = try module.hir.addVariant(enum_id, try internForTest(&module, "Done"), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(done, try internForTest(&module, "ok"), module.types.boolType(), hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int tag;\n    union {\n        struct {\n            int cpt_pf_ok_0;\n        } cpt_v_Done_0;\n    } payload;\n} cpt_enum_Check;\n\nvoid main(void) {\ncpt_bb_0:\n    return;\n}\n",
        c_source,
    );
}

test "MIR C backend emits mixed enum layout with backend-owned escaped names" {
    var module = try newTestModule();
    defer module.deinit();

    const enum_id = try module.hir.addEnum(try internForTest(&module, "enum"), false);
    _ = try module.types.addEnumType(enum_id);
    _ = try module.hir.addVariant(enum_id, try internForTest(&module, "None"), hir.synthetic_span);
    const some = try module.hir.addVariant(enum_id, try internForTest(&module, "some-value"), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(some, try internForTest(&module, "return"), module.types.intType(), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(some, try internForTest(&module, "flag?"), module.types.boolType(), hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int tag;\n    union {\n        struct {\n            int cpt_pf_return__0;\n            int cpt_pf_flag_x3F_1;\n        } cpt_v_some_x2D_value_1;\n    } payload;\n} cpt_enum_enum_;\n\nvoid main(void) {\ncpt_bb_0:\n    return;\n}\n",
        c_source,
    );
}

test "MIR C backend renders enum TypeId in function signatures and copies" {
    var module = try newTestModule();
    defer module.deinit();

    const enum_id = try module.hir.addEnum(try internForTest(&module, "Status"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    _ = try module.hir.addVariant(enum_id, try internForTest(&module, "Ok"), hir.synthetic_span);
    _ = try module.hir.addVariant(enum_id, try internForTest(&module, "Err"), hir.synthetic_span);

    const identity_hir = try addHirFunctionForTest(&module, "identity", enum_type);
    _ = try module.hir.addParam(identity_hir, try internForTest(&module, "value"), enum_type, hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    const identity_mir = try mir_module.store.addFunction(identity_hir, try internForTest(&module, "identity"), enum_type, hir.synthetic_span);
    const value = try mir_module.store.addLocal(identity_mir, try internForTest(&module, "value"), .param, enum_type, hir.synthetic_span);
    const block = try mir_module.store.addBlock(identity_mir, hir.synthetic_span);
    try mir_module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = value })) });

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int tag;\n} cpt_enum_Status;\n\ncpt_enum_Status cpt_f_identity(cpt_enum_Status cpt_p_value_0) {\ncpt_bb_0:\n    return cpt_p_value_0;\n}\n",
        c_source,
    );
}

test "MIR C backend rejects empty enum layout" {
    var module = try newTestModule();
    defer module.deinit();

    const enum_id = try module.hir.addEnum(try internForTest(&module, "Never"), false);
    _ = try module.types.addEnumType(enum_id);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostic_bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UnsupportedCBackendType, diagnostic_bag.diagnostics.items[0].code);
}

test "MIR C backend rejects enum payload fields with unsupported struct type" {
    var module = try newTestModule();
    defer module.deinit();

    const struct_id = try module.hir.addStruct(try internForTest(&module, "Payload"));
    const struct_type = try module.types.addStructType(struct_id);
    const enum_id = try module.hir.addEnum(try internForTest(&module, "Box"), false);
    _ = try module.types.addEnumType(enum_id);
    const boxed = try module.hir.addVariant(enum_id, try internForTest(&module, "Boxed"), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(boxed, try internForTest(&module, "payload"), struct_type, hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostic_bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UnsupportedCBackendType, diagnostic_bag.diagnostics.items[0].code);
}

test "MIR C backend rejects nominal types with backend diagnostic" {
    var module = try newTestModule();
    defer module.deinit();

    const struct_id = try module.hir.addStruct(try internForTest(&module, "Vec3"));
    const struct_type = try module.types.addStructType(struct_id);
    const main_hir = try addHirFunctionForTest(&module, "main", module.types.voidType());

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    const main_mir = try mir_module.store.addFunction(main_hir, try internForTest(&module, "main"), module.types.voidType(), hir.synthetic_span);
    _ = try mir_module.store.addLocal(main_mir, try internForTest(&module, "value"), .user, struct_type, hir.synthetic_span);
    const block = try mir_module.store.addBlock(main_mir, hir.synthetic_span);
    try mir_module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(null) });

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostic_bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UnsupportedCBackendType, diagnostic_bag.diagnostics.items[0].code);
}

test "MIR C backend reports invalid TypeId before type rendering" {
    var module = try newTestModule();
    defer module.deinit();

    const main_hir = try addHirFunctionForTest(&module, "main", module.types.voidType());

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    const main_mir = try mir_module.store.addFunction(main_hir, try internForTest(&module, "main"), module.types.voidType(), hir.synthetic_span);
    _ = try mir_module.store.addLocal(main_mir, try internForTest(&module, "bad"), .user, .{ .index = 999 }, hir.synthetic_span);
    const block = try mir_module.store.addBlock(main_mir, hir.synthetic_span);
    try mir_module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(null) });

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostic_bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.InvalidMirType, diagnostic_bag.diagnostics.items[0].code);
}

test "MIR C backend emits raw pointer params returns and locals" {
    const c_source = try emitForTest(
        \\module Main;
        \\int* identity(int* p) { int* q = p; return q; }
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int* cpt_f_identity(int* cpt_p_p_0);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int* cpt_l_q_1;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_l_q_1 = cpt_p_p_0;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_l_q_1;") != null);
}

test "MIR C backend emits address-of and deref" {
    const c_source = try emitForTest(
        \\module Main;
        \\int readParamAddress(int x) {
        \\    int* p = &x;
        \\    unsafe { return *p; }
        \\}
        \\int main() {
        \\    int x = 7;
        \\    int* p = &x;
        \\    unsafe { return *p; }
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int* cpt_l_p_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, " = &cpt_l_x_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, " = &cpt_p_x_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, " = *cpt_l_p_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "unsafe") == null);
}

test "MIR C backend emits enum raw pointer params and returns" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum Status { Ok, Err, };
        \\Status* choose(bool useLeft, Status* left, Status* right) {
        \\    if (useLeft) { return left; }
        \\    return right;
        \\}
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_enum_Status* cpt_f_choose(int cpt_p_useLeft_0, cpt_enum_Status* cpt_p_left_1, cpt_enum_Status* cpt_p_right_2);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_p_left_1;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_p_right_2;") != null);
}

test "MIR C backend rejects struct raw pointer rendering until struct layout is supported" {
    var parse_diagnostics = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, "test.concept",
        \\module Main;
        \\struct Box {};
        \\Box* id(Box* p) { return p; }
        \\int main() { return 0; }
    );
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics.collectTopLevelDeclarations(std.testing.allocator, unit, &diagnostic_bag);
    defer module.deinit();
    try hir_checker.checkExecutable(std.testing.allocator, &module, &diagnostic_bag);
    var mir_module = try mir_lowering.lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag));
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UnsupportedCBackendType, diagnostic_bag.diagnostics.items[0].code);
}

test "MIR C backend emits enum constructor assignments" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum Status { Ok, Err, };
        \\int main() { Status status = Status::Ok; status = Status::Err; return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef struct {\n    int tag;\n} cpt_enum_Status;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".tag = 0;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".tag = 1;") != null);
}

test "MIR C backend emits payload enum constructor assignment" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum ParseResult { Ok(int value), Err(int code), };
        \\int main() { ParseResult result = ParseResult::Ok(7); return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_enum_ParseResult") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".tag = 0;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".payload.cpt_v_Ok_0.cpt_pf_value_0 = 7;") != null);
}

test "MIR C backend returns enum constructor through temp" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum Status { Ok, Err, };
        \\Status make() { return Status::Ok; }
        \\int main() { Status status = make(); return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_enum_Status cpt_f_make(void)") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".tag = 0;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_t_") != null);
}

test "MIR C backend emits enum match tag switch" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum Status { Ok, Err, };
        \\int main() { Status status = Status::Ok; match (status) { Status::Ok => return 7; Status::Err => return 1; } }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, ".tag;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "switch (cpt_t_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "case 0: goto") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "case 1: goto") != null);
}

test "MIR C backend emits enum match payload extraction" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum ParseResult { Ok(int value), Err(int code), };
        \\int main() { ParseResult result = ParseResult::Ok(7); match (result) { ParseResult::Ok(value) => return value; _ => return 0; } }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, ".payload.cpt_v_Ok_0.cpt_pf_value_0") != null);
}

test "MIR C backend emits enum match multiple payload extraction" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum PairResult { Ok(int left, int right), Err(int code), };
        \\int main() { PairResult pair = PairResult::Ok(4, 5); match (pair) { PairResult::Ok(left, right) => return left + right; _ => return 0; } }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, ".payload.cpt_v_Ok_0.cpt_pf_left_0") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".payload.cpt_v_Ok_0.cpt_pf_right_1") != null);
}

test "MIR C backend emits payload enum match without binding" {
    const c_source = try emitForTest(
        \\module Main;
        \\enum ParseResult { Ok(int value), Err(int code), };
        \\int main() { ParseResult result = ParseResult::Ok(7); match (result) { ParseResult::Ok => return 7; _ => return 0; } }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_enum_ParseResult") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".tag;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "case 0: goto") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "default: goto") != null);
}
