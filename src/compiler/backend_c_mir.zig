const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const mir = @import("mir.zig");
const mir_validator = @import("mir_validator.zig");
const parser_model = @import("parser.zig");
const semantics = @import("semantics.zig");
const source_model = @import("source.zig");
const types = @import("types.zig");

pub const EmitError = error{InvalidExecutable} || std.mem.Allocator.Error;

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

    if (mir_module.store.functions.items.len > 1) {
        for (mir_module.store.functions.items, 0..) |function, index| {
            try emitPrototype(writer, semantic_module, mir_module, .{ .index = @intCast(index) }, function);
        }
        try writer.writeByte('\n');
    }

    for (mir_module.store.functions.items, 0..) |function, index| {
        if (index != 0) try writer.writeByte('\n');
        try emitFunction(writer, semantic_module, mir_module, .{ .index = @intCast(index) }, function);
    }
    return output.toOwnedSlice();
}

fn emitPrototype(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function_id: mir.MirFunctionId, function: mir.MirFunction) !void {
    _ = function_id;
    try emitCType(writer, module, function.return_type);
    try writer.writeByte(' ');
    try emitSymbolName(writer, module, function.name);
    try writer.writeByte('(');
    try emitParamList(writer, module, mir_module, function, function.params);
    try writer.writeAll(");\n");
}

fn emitFunction(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function_id: mir.MirFunctionId, function: mir.MirFunction) !void {
    _ = function_id;
    try emitCType(writer, module, function.return_type);
    try writer.writeByte(' ');
    try emitSymbolName(writer, module, function.name);
    try writer.writeByte('(');
    try emitParamList(writer, module, mir_module, function, function.params);
    try writer.writeAll(") {\n");

    for (function.locals) |local_id| {
        const local = mir_module.store.getLocal(local_id);
        try writer.writeAll("    ");
        try emitCType(writer, module, local.type_id);
        try writer.writeByte(' ');
        try emitLocalName(writer, module, mir_module, function, local_id);
        try writer.writeAll(";\n");
    }
    if (function.locals.len != 0) try writer.writeByte('\n');

    for (function.blocks) |block_id| {
        try emitBlock(writer, module, mir_module, function, block_id);
    }

    try writer.writeAll("}\n");
}

fn emitBlock(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, block_id: mir.MirBlockId) !void {
    try emitBlockLabel(writer, function, block_id);
    try writer.writeAll(":\n");

    const block = mir_module.store.getBlock(block_id);
    for (block.statements) |statement| {
        try emitStatement(writer, module, mir_module, function, statement);
    }
    try emitTerminator(writer, module, mir_module, function, block.terminator.?);
}

fn emitStatement(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, statement: mir.MirStatement) !void {
    switch (statement.kind) {
        .assign => |assignment| {
            try writer.writeAll("    ");
            try emitPlace(writer, module, mir_module, function, assignment.place);
            try writer.writeAll(" = ");
            try emitRvalue(writer, module, mir_module, function, assignment.rvalue);
            try writer.writeAll(";\n");
        },
    }
}

fn emitTerminator(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, terminator: mir.MirTerminator) !void {
    switch (terminator.kind) {
        .goto => |target| {
            try writer.writeAll("    goto ");
            try emitBlockLabel(writer, function, target);
            try writer.writeAll(";\n");
        },
        .return_ => |maybe_operand| {
            try writer.writeAll("    return");
            if (maybe_operand) |operand| {
                try writer.writeByte(' ');
                try emitOperand(writer, module, mir_module, function, operand);
            }
            try writer.writeAll(";\n");
        },
        .switch_bool => |switch_bool| {
            try writer.writeAll("    if (");
            try emitOperand(writer, module, mir_module, function, switch_bool.discriminant);
            try writer.writeAll(") goto ");
            try emitBlockLabel(writer, function, switch_bool.true_target);
            try writer.writeAll("; else goto ");
            try emitBlockLabel(writer, function, switch_bool.false_target);
            try writer.writeAll(";\n");
        },
        .switch_int => |switch_int| {
            try writer.writeAll("    switch (");
            try emitOperand(writer, module, mir_module, function, switch_int.discriminant);
            try writer.writeAll(") {\n");
            for (switch_int.cases) |case| {
                try writer.writeAll("        case ");
                try writer.writeAll(case.value);
                try writer.writeAll(": goto ");
                try emitBlockLabel(writer, function, case.target);
                try writer.writeAll(";\n");
            }
            try writer.writeAll("        default: goto ");
            try emitBlockLabel(writer, function, switch_int.default_target);
            try writer.writeAll(";\n");
            try writer.writeAll("    }\n");
        },
        .@"unreachable" => try writer.writeAll("    __builtin_unreachable();\n"),
    }
}

fn emitRvalue(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, rvalue: mir.MirRvalue) !void {
    switch (rvalue) {
        .use => |operand| try emitOperand(writer, module, mir_module, function, operand),
        .unary => |unary| {
            try writer.writeAll(unary.op.lexeme());
            try emitOperand(writer, module, mir_module, function, unary.operand);
        },
        .binary => |binary| {
            try writer.writeByte('(');
            try emitOperand(writer, module, mir_module, function, binary.left);
            try writer.writeByte(' ');
            try writer.writeAll(binary.op.lexeme());
            try writer.writeByte(' ');
            try emitOperand(writer, module, mir_module, function, binary.right);
            try writer.writeByte(')');
        },
        .call => |call| {
            const callee = module.hir.getFunction(call.function);
            try emitSymbolName(writer, module, callee.name);
            try writer.writeByte('(');
            for (call.args, 0..) |arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try emitOperand(writer, module, mir_module, function, arg);
            }
            try writer.writeByte(')');
        },
    }
}

fn emitOperand(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, operand: mir.MirOperand) !void {
    switch (operand) {
        .int_literal => |text| try writer.writeAll(text),
        .bool_literal => |value| try writer.writeAll(if (value) "1" else "0"),
        .copy => |place| try emitPlace(writer, module, mir_module, function, place),
    }
}

fn emitPlace(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, place: mir.MirPlace) !void {
    switch (place) {
        .local => |local_id| try emitLocalName(writer, module, mir_module, function, local_id),
    }
}

fn emitParamList(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, params: []const mir.MirLocalId) !void {
    if (params.len == 0) {
        try writer.writeAll("void");
        return;
    }
    for (params, 0..) |local_id, index| {
        const local = mir_module.store.getLocal(local_id);
        if (index != 0) try writer.writeAll(", ");
        try emitCType(writer, module, local.type_id);
        try writer.writeByte(' ');
        try emitLocalName(writer, module, mir_module, function, local_id);
    }
}

fn emitCType(writer: anytype, module: *const semantics.SemanticModule, type_id: types.TypeId) !void {
    switch (module.types.kind(type_id)) {
        .void => try writer.writeAll("void"),
        .int, .bool => try writer.writeAll("int"),
        // TODO(P4): richer C type lowering and diagnostics for non-executable MIR types.
        .struct_type, .enum_type => try writer.writeAll("int"),
    }
}

fn emitSymbolName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    // TODO(P3): add proper C identifier mangling. Current fixtures use simple C-compatible names.
    try writer.writeAll(module.interner.text(symbol));
}

fn emitLocalName(writer: anytype, module: *const semantics.SemanticModule, mir_module: *const mir.MirModule, function: mir.MirFunction, local_id: mir.MirLocalId) !void {
    const local = mir_module.store.getLocal(local_id);
    if (local.name) |name| {
        try emitSymbolName(writer, module, name);
        return;
    }
    try writer.print("tmp{d}", .{tempIndex(mir_module, function, local_id)});
}

fn tempIndex(mir_module: *const mir.MirModule, function: mir.MirFunction, local_id: mir.MirLocalId) usize {
    var index: usize = 0;
    for (function.locals) |candidate_id| {
        const candidate = mir_module.store.getLocal(candidate_id);
        if (candidate.kind != .temp) continue;
        if (candidate_id.index == local_id.index) return index;
        index += 1;
    }
    return index;
}

fn blockIndex(function: mir.MirFunction, block_id: mir.MirBlockId) usize {
    for (function.blocks, 0..) |candidate_id, index| {
        if (candidate_id.index == block_id.index) return index;
    }
    return block_id.index;
}

fn emitBlockLabel(writer: anytype, function: mir.MirFunction, block_id: mir.MirBlockId) !void {
    try writer.print("bb{d}", .{blockIndex(function, block_id)});
}

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

    const hir_checker = @import("hir_checker.zig");
    try hir_checker.checkExecutable(std.testing.allocator, &module, &check_diagnostics);

    const mir_lowering = @import("mir_lowering.zig");
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

test "MIR C backend emits return literal" {
    try expectEmit(
        "module Main; int main() { return 0; }",
        "int main(void) {\nbb0:\n    return 0;\n}\n",
    );
}

test "MIR C backend emits arithmetic" {
    try expectEmit(
        "module Main; int main() { return 1 + 2 * 3; }",
        "int main(void) {\n    int tmp0;\n    int tmp1;\n\nbb0:\n    tmp0 = (2 * 3);\n    tmp1 = (1 + tmp0);\n    return tmp1;\n}\n",
    );
}

test "MIR C backend emits locals and assignment" {
    try expectEmit(
        "module Main; int main() { int x = 1; x = x + 2; return x; }",
        "int main(void) {\n    int x;\n    int tmp0;\n\nbb0:\n    x = 1;\n    tmp0 = (x + 2);\n    x = tmp0;\n    return x;\n}\n",
    );
}

test "MIR C backend emits function call" {
    try expectEmit(
        "module Main; int add(int a, int b) { return a + b; } int main() { return add(1, 2); }",
        "int add(int a, int b);\nint main(void);\n\nint add(int a, int b) {\n    int tmp0;\n\nbb0:\n    tmp0 = (a + b);\n    return tmp0;\n}\n\nint main(void) {\n    int tmp0;\n\nbb0:\n    tmp0 = add(1, 2);\n    return tmp0;\n}\n",
    );
}

test "MIR C backend emits if else labels" {
    try expectEmit(
        "module Main; int main() { if (false) { return 1; } else { return 7; } }",
        "int main(void) {\nbb0:\n    if (0) goto bb1; else goto bb3;\nbb1:\n    return 1;\nbb2:\n    __builtin_unreachable();\nbb3:\n    return 7;\n}\n",
    );
}

test "MIR C backend emits while count loop" {
    try expectEmit(
        "module Main; int main() { int x = 0; while (x < 7) { x = x + 1; } return x; }",
        "int main(void) {\n    int x;\n    int tmp0;\n    int tmp1;\n\nbb0:\n    x = 0;\n    goto bb1;\nbb1:\n    tmp0 = (x < 7);\n    if (tmp0) goto bb2; else goto bb3;\nbb2:\n    tmp1 = (x + 1);\n    x = tmp1;\n    goto bb1;\nbb3:\n    return x;\n}\n",
    );
}

test "MIR C backend emits int match default" {
    try expectEmit(
        "module Main; int main() { int x = 3; match (x) { 1 => return 10; 2 => return 7; _ => return 0; } }",
        "int main(void) {\n    int x;\n\nbb0:\n    x = 3;\n    switch (x) {\n        case 1: goto bb2;\n        case 2: goto bb3;\n        default: goto bb4;\n    }\nbb1:\n    __builtin_unreachable();\nbb2:\n    return 10;\nbb3:\n    return 7;\nbb4:\n    return 0;\n}\n",
    );
}

test "MIR C backend emits bool match" {
    try expectEmit(
        "module Main; int main() { bool ok = true; match (ok) { true => return 7; false => return 0; } return 0; }",
        "int main(void) {\n    int ok;\n\nbb0:\n    ok = 1;\n    if (ok) goto bb2; else goto bb3;\nbb1:\n    return 0;\nbb2:\n    return 7;\nbb3:\n    return 0;\n}\n",
    );
}
