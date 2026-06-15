// ─────────────────────────────────────────────────────────────────────────────
// Imports and public API
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const hir_checker = @import("hir_checker.zig");
const mir = @import("mir.zig");
const mir_lowering = @import("mir_lowering.zig");
const mir_storage = @import("mir_storage.zig");
const mir_validator = @import("mir_validator.zig");
const parser_model = @import("parser.zig");
const semantics = @import("semantics.zig");
const source_model = @import("source.zig");
const types = @import("types.zig");

pub const EmitError = error{InvalidExecutable} || std.mem.Allocator.Error || std.Io.Writer.Error;

// ─────────────────────────────────────────────────────────────────────────────
// Backend context
// ─────────────────────────────────────────────────────────────────────────────

const BackendContext = struct {
    module: *const semantics.SemanticModule,
    mir_module: *mir.MirModule,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
};

pub fn emitExecutableFromMir(
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
    mir_module: *mir.MirModule,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
) EmitError![]const u8 {
    mir_storage.analyzeModule(allocator, semantic_module, mir_module, diagnostic_bag) catch |err| switch (err) {
        error.InvalidStorageState => return error.InvalidExecutable,
        error.OutOfMemory => return error.OutOfMemory,
    };
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
    if (emitted_enum_layouts and ctx.module.hir.structs.items.len > 0) try writer.writeByte('\n');

    const emitted_struct_layouts = try emitStructLayouts(writer, &ctx);
    if ((emitted_enum_layouts or emitted_struct_layouts) and ctx.module.hir.machines.items.len > 0) try writer.writeByte('\n');

    const emitted_dyn_interface_layouts = try emitDynInterfaceLayouts(writer, &ctx);
    if ((emitted_enum_layouts or emitted_struct_layouts or emitted_dyn_interface_layouts) and ctx.module.hir.machines.items.len > 0) try writer.writeByte('\n');

    const emitted_machine_layouts = try emitMachineLayouts(writer, &ctx);
    if ((emitted_enum_layouts or emitted_struct_layouts or emitted_machine_layouts) and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    const emitted_opaque_allocation_decls = try emitOpaqueAllocationForwardDeclarations(writer, &ctx);
    if (emitted_opaque_allocation_decls and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    const emitted_arena_alloc_size_header = try emitArenaAllocSizeHeader(writer, &ctx);
    if (emitted_arena_alloc_size_header and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    const emitted_arena_alloc_decl = try emitArenaAllocHelperDeclaration(writer, &ctx);
    if (emitted_arena_alloc_decl and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    const emitted_arena_storage_decl = try emitArenaStorageHelperDeclarations(writer, &ctx);
    if (emitted_arena_storage_decl and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    const emitted_panic_helper = try emitPanicHelper(writer, &ctx);
    if (emitted_panic_helper and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    const emitted_extern_c_prototypes = try emitExternCPrototypes(writer, &ctx);
    if (emitted_extern_c_prototypes and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

    if (mir_module.store.functions.items.len > 1) {
        for (mir_module.store.functions.items, 0..) |function, index| {
            try emitPrototype(writer, &ctx, .{ .index = @intCast(index) }, function);
        }
        try writer.writeByte('\n');
    }

    const emitted_interface_impl_tables = try emitInterfaceImplWrappersAndTables(writer, &ctx);
    if (emitted_interface_impl_tables and mir_module.store.functions.items.len > 0) try writer.writeByte('\n');

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
        .int, .bool, .alloc_error => true,
        .void, .arena, .allocator, .struct_type, .enum_type, .machine_type, .interface_type, .dyn_interface, .pointer, .manual_init, .type_param => false,
    };
}

// ─────────────────────────────────────────────────────────────────────────────
// Struct layout emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitStructLayouts(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var emitted_any = false;
    for (ctx.module.hir.structs.items, 0..) |struct_decl, index| {
        if (emitted_any) try writer.writeByte('\n');
        try emitStructLayout(writer, ctx, .{ .index = @intCast(index) }, struct_decl);
        emitted_any = true;
    }
    return emitted_any;
}

/// P7-M1 struct layout emission is backend-local and intentionally not ABI-stable.
/// Empty structs, by-value nested structs, invalid TypeIds, `void` fields, and
/// pointers to structs report CON0045 until later Phase 7 layout ordering and
/// recursive/reference rules are defined.
fn emitStructLayout(writer: anytype, ctx: *const BackendContext, struct_id: hir.StructId, struct_decl: hir.HirStruct) EmitError!void {
    try requireSupportedStructLayout(ctx, struct_id, null);

    if (struct_decl.repr_abi == .c) {
        try writer.writeAll("typedef struct ");
        try emitStructTypeName(writer, ctx, struct_id);
        try writer.writeAll(" {\n");
    } else {
        try writer.writeAll("typedef struct {\n");
    }
    for (struct_decl.fields, 0..) |field_id, field_index| {
        const field = ctx.module.hir.getField(field_id);
        try writer.writeAll("    ");
        try emitStructFieldCType(writer, ctx, field.type_id, field.span);
        try writer.writeByte(' ');
        try emitStructFieldName(writer, ctx.module, field.name, field_index);
        try writer.writeAll(";\n");
    }
    try writer.writeAll("} ");
    try emitStructTypeName(writer, ctx, struct_id);
    try writer.writeAll(";\n");
}

fn requireSupportedStructLayout(ctx: *const BackendContext, struct_id: hir.StructId, span: ?diagnostics.SourceSpan) EmitError!void {
    if (struct_id.index >= ctx.module.hir.structs.items.len) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }

    const struct_decl = ctx.module.hir.getStruct(struct_id);
    if (struct_decl.fields.len == 0) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }

    for (struct_decl.fields) |field_id| {
        const field = ctx.module.hir.getField(field_id);
        if (!isSupportedStructFieldType(ctx, field.type_id)) {
            try reportUnsupportedCType(ctx, field.span);
            return error.InvalidExecutable;
        }
    }
}

fn isSupportedStructFieldType(ctx: *const BackendContext, type_id: types.TypeId) bool {
    if (!ctx.module.types.contains(type_id)) return false;
    return switch (ctx.module.types.kind(type_id)) {
        .int, .bool, .alloc_error => true,
        .enum_type => |enum_id| isSupportedEnumLayout(ctx, enum_id),
        .pointer => |pointer| isSupportedStructFieldPointerType(ctx, pointer.pointee),
        .void, .arena, .allocator, .struct_type, .machine_type, .interface_type, .dyn_interface, .manual_init, .type_param => false,
    };
}

fn isSupportedStructFieldPointerType(ctx: *const BackendContext, pointee: types.TypeId) bool {
    if (!ctx.module.types.contains(pointee)) return false;
    return switch (ctx.module.types.kind(pointee)) {
        .void, .int, .bool, .arena, .allocator, .alloc_error => true,
        .enum_type => |enum_id| isSupportedEnumLayout(ctx, enum_id),
        .pointer => |nested| isSupportedStructFieldPointerType(ctx, nested.pointee),
        .struct_type => |struct_id| struct_id.index < ctx.module.hir.structs.items.len and ctx.module.hir.getStruct(struct_id).repr_abi == .c,
        .machine_type, .interface_type, .dyn_interface, .manual_init, .type_param => false,
    };
}

// ─────────────────────────────────────────────────────────────────────────────
// Interface dyn/vtable layout emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitDynInterfaceLayouts(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var emitted_any = false;
    for (ctx.module.hir.interfaces.items, 0..) |interface_decl, index| {
        const interface_id = hir.InterfaceId{ .index = @intCast(index) };
        if (!usesDynInterface(ctx, interface_id)) continue;
        if (emitted_any) try writer.writeByte('\n');
        try emitInterfaceVTableLayout(writer, ctx, interface_id, interface_decl);
        try writer.writeByte('\n');
        try emitDynInterfaceLayout(writer, ctx, interface_decl);
        emitted_any = true;
    }
    return emitted_any;
}

fn emitInterfaceVTableLayout(writer: anytype, ctx: *const BackendContext, interface_id: hir.InterfaceId, interface_decl: hir.HirInterface) EmitError!void {
    try writer.writeAll("typedef struct {\n");
    for (interface_decl.requirements) |requirement_id| {
        const requirement = ctx.module.hir.getInterfaceRequirement(requirement_id);
        try writer.writeAll("    ");
        try emitCType(writer, ctx, requirement.return_type, requirement.span);
        try writer.writeAll(" (*");
        try emitInterfaceRequirementSlotName(writer, ctx.module, requirement.name);
        try writer.writeAll(")(void* self");
        for (requirement.params) |param_id| {
            const param = ctx.module.hir.getInterfaceParam(param_id);
            try writer.writeAll(", ");
            try emitCType(writer, ctx, param.type_id, param.span);
        }
        try writer.writeAll(");\n");
    }
    try writer.writeAll("} ");
    try emitInterfaceVTableTypeName(writer, ctx.module, ctx.module.hir.getInterface(interface_id).name);
    try writer.writeAll(";\n");
}

fn emitDynInterfaceLayout(writer: anytype, ctx: *const BackendContext, interface_decl: hir.HirInterface) EmitError!void {
    try writer.writeAll("typedef struct {\n");
    try writer.writeAll("    void* data;\n");
    try writer.writeAll("    const ");
    try emitInterfaceVTableTypeName(writer, ctx.module, interface_decl.name);
    try writer.writeAll("* vtable;\n");
    try writer.writeAll("} ");
    try emitDynInterfaceTypeName(writer, ctx.module, interface_decl.name);
    try writer.writeAll(";\n");
}

fn usesDynInterface(ctx: *const BackendContext, interface_id: hir.InterfaceId) bool {
    for (ctx.mir_module.store.functions.items) |function| {
        if (typeIsDynInterface(ctx, function.return_type, interface_id)) return true;
        for (function.params) |local_id| {
            if (typeIsDynInterface(ctx, ctx.mir_module.store.getLocal(local_id).type_id, interface_id)) return true;
        }
        for (function.locals) |local_id| {
            if (typeIsDynInterface(ctx, ctx.mir_module.store.getLocal(local_id).type_id, interface_id)) return true;
        }
    }
    return false;
}

fn typeIsDynInterface(ctx: *const BackendContext, type_id: types.TypeId, interface_id: hir.InterfaceId) bool {
    if (!ctx.module.types.contains(type_id)) return false;
    return switch (ctx.module.types.kind(type_id)) {
        .dyn_interface => |dyn| dyn.interface_id.index == interface_id.index,
        else => false,
    };
}

// ─────────────────────────────────────────────────────────────────────────────
// Machine layout and step emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitMachineLayouts(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var emitted_any = false;
    for (ctx.module.hir.machines.items, 0..) |machine, index| {
        if (emitted_any) try writer.writeByte('\n');
        try emitMachineLayout(writer, ctx, .{ .index = @intCast(index) }, machine);
        emitted_any = true;
    }
    return emitted_any;
}

fn emitMachineLayout(writer: anytype, ctx: *const BackendContext, machine_id: hir.MachineId, machine: hir.HirMachine) EmitError!void {
    if (!isSupportedMachineScalarType(ctx, machine.return_type)) {
        try reportUnsupportedCType(ctx, machine.span);
        return error.InvalidExecutable;
    }

    try writer.writeAll("typedef enum {\n");
    for (machine.states, 0..) |state, index| {
        try writer.writeAll("    ");
        try emitMachineStateName(writer, ctx.module, machine.name, state.name);
        try writer.print(" = {d},\n", .{index});
    }
    try writer.writeAll("} ");
    try emitMachineStateTypeName(writer, ctx.module, machine.name);
    try writer.writeAll(";\n\n");

    try writer.writeAll("typedef struct {\n    ");
    try emitMachineStateTypeName(writer, ctx.module, machine.name);
    try writer.writeAll(" state;\n    int complete;\n    ");
    try emitCType(writer, ctx, machine.return_type, machine.span);
    try writer.writeAll(" result;\n");
    for (machine.params) |param_id| {
        const param = ctx.module.hir.getMachineParam(param_id);
        if (!isSupportedMachineScalarType(ctx, param.type_id)) {
            try reportUnsupportedCType(ctx, param.span);
            return error.InvalidExecutable;
        }
        try writer.writeAll("    ");
        try emitCType(writer, ctx, param.type_id, param.span);
        try writer.writeByte(' ');
        try emitMachineParamFieldName(writer, ctx.module, param.name);
        try writer.writeAll(";\n");
    }
    for (machine.fields, 0..) |field_id, field_index| {
        const field = ctx.module.hir.getMachineField(field_id);
        switch (ctx.module.types.kind(field.type_id)) {
            .machine_type => |child_id| {
                const child = ctx.module.hir.getMachine(child_id);
                if (child.params.len != 0) {
                    if (ctx.diagnostic_bag) |bag| try bag.append(diagnostics.nestedMachineFieldRequiresDefaultConstruction(field.type_span));
                    return error.InvalidExecutable;
                }
                try writer.writeAll("    ");
                try emitMachineTypeName(writer, ctx.module, child.name);
                try writer.writeByte(' ');
                try emitMachineFieldName(writer, ctx.module, field.name, field_index);
                try writer.writeAll(";\n");
            },
            else => {
                try reportUnsupportedCType(ctx, field.span);
                return error.InvalidExecutable;
            },
        }
    }
    try writer.writeAll("} ");
    try emitMachineTypeName(writer, ctx.module, machine.name);
    try writer.writeAll(";\n\n");

    try writer.writeAll("static ");
    try emitMachineTypeName(writer, ctx.module, machine.name);
    try writer.writeByte(' ');
    try emitMachineNewFunctionName(writer, ctx.module, machine.name);
    try writer.writeByte('(');
    for (machine.params, 0..) |param_id, index| {
        if (index != 0) try writer.writeAll(", ");
        const param = ctx.module.hir.getMachineParam(param_id);
        try emitCType(writer, ctx, param.type_id, param.span);
        try writer.writeByte(' ');
        try emitMachineParamName(writer, ctx.module, param.name);
    }
    try writer.writeAll(") {\n    ");
    try emitMachineTypeName(writer, ctx.module, machine.name);
    try writer.writeAll(" m;\n    m.state = ");
    try emitMachineStateName(writer, ctx.module, machine.name, machine.states[machine.initial_state_index].name);
    try writer.writeAll(";\n    m.complete = 0;\n    m.result = 0;\n");
    for (machine.params) |param_id| {
        const param = ctx.module.hir.getMachineParam(param_id);
        try writer.writeAll("    m.");
        try emitMachineParamFieldName(writer, ctx.module, param.name);
        try writer.writeAll(" = ");
        try emitMachineParamName(writer, ctx.module, param.name);
        try writer.writeAll(";\n");
    }
    for (machine.fields, 0..) |field_id, field_index| {
        const field = ctx.module.hir.getMachineField(field_id);
        switch (ctx.module.types.kind(field.type_id)) {
            .machine_type => |child_id| {
                const child = ctx.module.hir.getMachine(child_id);
                try writer.writeAll("    m.");
                try emitMachineFieldName(writer, ctx.module, field.name, field_index);
                try writer.writeAll(" = ");
                try emitMachineNewFunctionName(writer, ctx.module, child.name);
                try writer.writeAll("();\n");
            },
            else => return error.InvalidExecutable,
        }
    }
    try writer.writeAll("    return m;\n}\n\n");

    try writer.writeAll("static void ");
    try emitMachineStepFunctionName(writer, ctx.module, machine.name);
    try writer.writeByte('(');
    try emitMachineTypeName(writer, ctx.module, machine.name);
    try writer.writeAll("* m) {\n    if (m->complete) return;\n    switch (m->state) {\n");
    for (machine.states) |state| {
        try writer.writeAll("        case ");
        try emitMachineStateName(writer, ctx.module, machine.name, state.name);
        try writer.writeAll(":\n");
        if (state.body) |body| {
            try emitMachineStateStmt(writer, ctx, machine_id, machine, body);
        }
        try writer.writeAll("            return;\n");
    }
    try writer.writeAll("    }\n}\n");
}

fn isSupportedMachineScalarType(ctx: *const BackendContext, type_id: types.TypeId) bool {
    if (!ctx.module.types.contains(type_id)) return false;
    return switch (ctx.module.types.kind(type_id)) {
        .int, .bool => true,
        else => false,
    };
}

fn emitMachineStateStmt(writer: anytype, ctx: *const BackendContext, machine_id: hir.MachineId, machine: hir.HirMachine, stmt_id: hir.StmtId) EmitError!void {
    const stmt = ctx.module.hir.getStmt(stmt_id).*;
    switch (stmt.kind) {
        .block => |children| for (children) |child| try emitMachineStateStmt(writer, ctx, machine_id, machine, child),
        .transition_stmt => |target| switch (target) {
            .literal_state => |literal| {
                try writer.writeAll("            m->state = ");
                try emitMachineStateName(writer, ctx.module, machine.name, machine.states[literal.state_index].name);
                try writer.writeAll(";\n            return;\n");
            },
            .match_state => |match_target| try emitMachineTransitionMatch(writer, ctx, machine_id, machine, match_target),
            .decide_state => {
                if (ctx.diagnostic_bag) |bag| try bag.append(diagnostics.machineSemanticsNotImplemented(stmt.span));
                return error.InvalidExecutable;
            },
        },
        .return_stmt => |maybe_expr| {
            const expr_id = maybe_expr orelse {
                if (ctx.diagnostic_bag) |bag| try bag.append(diagnostics.machineSemanticsNotImplemented(stmt.span));
                return error.InvalidExecutable;
            };
            try writer.writeAll("            m->result = ");
            try emitMachineStateExpr(writer, ctx, machine_id, expr_id);
            try writer.writeAll(";\n            m->complete = 1;\n            return;\n");
        },
        .expr_stmt => |expr_id| {
            const expr = ctx.module.hir.getExpr(expr_id).*;
            switch (expr.kind) {
                .machine_step => |machine_expr| {
                    const field_id = try machineFieldRef(ctx, machine_id, machine_expr);
                    const field = ctx.module.hir.getMachineField(field_id);
                    const child_id = switch (ctx.module.types.kind(field.type_id)) {
                        .machine_type => |id| id,
                        else => return error.InvalidExecutable,
                    };
                    try writer.writeAll("            ");
                    try emitMachineStepFunctionName(writer, ctx.module, ctx.module.hir.getMachine(child_id).name);
                    try writer.writeAll("(&m->");
                    try emitMachineFieldName(writer, ctx.module, field.name, machineFieldIndex(ctx, machine_id, field_id) orelse return error.InvalidExecutable);
                    try writer.writeAll(");\n");
                },
                else => {
                    if (ctx.diagnostic_bag) |bag| try bag.append(diagnostics.machineSemanticsNotImplemented(expr.span));
                    return error.InvalidExecutable;
                },
            }
        },
        .if_stmt => |if_stmt| {
            try writer.writeAll("            if (");
            try emitMachineStateExpr(writer, ctx, machine_id, if_stmt.condition);
            try writer.writeAll(") {\n");
            try emitMachineStateStmt(writer, ctx, machine_id, machine, if_stmt.then_block);
            try writer.writeAll("            }");
            if (if_stmt.else_block) |else_block| {
                try writer.writeAll(" else {\n");
                try emitMachineStateStmt(writer, ctx, machine_id, machine, else_block);
                try writer.writeAll("            }");
            }
            try writer.writeByte('\n');
        },
        else => {
            if (ctx.diagnostic_bag) |bag| try bag.append(diagnostics.machineSemanticsNotImplemented(stmt.span));
            return error.InvalidExecutable;
        },
    }
}

fn emitMachineTransitionMatch(writer: anytype, ctx: *const BackendContext, machine_id: hir.MachineId, machine: hir.HirMachine, match_target: anytype) EmitError!void {
    var true_target: ?hir.HirTransitionStateTarget = null;
    var false_target: ?hir.HirTransitionStateTarget = null;
    var default_target: ?hir.HirTransitionStateTarget = null;
    for (match_target.arms) |arm| {
        switch (arm.pattern) {
            .bool_literal => |value| {
                if (value) true_target = arm.target else false_target = arm.target;
            },
            .wildcard => default_target = arm.target,
            else => return error.InvalidExecutable,
        }
    }
    const on_true = true_target orelse default_target orelse return error.InvalidExecutable;
    const on_false = false_target orelse default_target orelse return error.InvalidExecutable;

    try writer.writeAll("            if (");
    try emitMachineStateExpr(writer, ctx, machine_id, match_target.scrutinee);
    try writer.writeAll(") {\n                m->state = ");
    try emitMachineStateName(writer, ctx.module, machine.name, machine.states[on_true.state_index].name);
    try writer.writeAll(";\n            } else {\n                m->state = ");
    try emitMachineStateName(writer, ctx.module, machine.name, machine.states[on_false.state_index].name);
    try writer.writeAll(";\n            }\n            return;\n");
}

fn emitMachineStateExpr(writer: anytype, ctx: *const BackendContext, machine_id: hir.MachineId, expr_id: hir.ExprId) EmitError!void {
    const expr = ctx.module.hir.getExpr(expr_id).*;
    switch (expr.kind) {
        .int_literal => |text| try writer.writeAll(text),
        .bool_literal => |value| try writer.writeAll(if (value) "1" else "0"),
        .machine_param_ref => |param_id| {
            const param = ctx.module.hir.getMachineParam(param_id);
            if (param.parent.index != machine_id.index) return error.InvalidExecutable;
            try writer.writeAll("m->");
            try emitMachineParamFieldName(writer, ctx.module, param.name);
        },
        .machine_field_ref => |field_id| {
            const field = ctx.module.hir.getMachineField(field_id);
            if (field.parent.index != machine_id.index) return error.InvalidExecutable;
            try writer.writeAll("m->");
            try emitMachineFieldName(writer, ctx.module, field.name, machineFieldIndex(ctx, machine_id, field_id) orelse return error.InvalidExecutable);
        },
        .machine_complete => |machine_expr| {
            const field_id = try machineFieldRef(ctx, machine_id, machine_expr);
            const field = ctx.module.hir.getMachineField(field_id);
            try writer.writeAll("m->");
            try emitMachineFieldName(writer, ctx.module, field.name, machineFieldIndex(ctx, machine_id, field_id) orelse return error.InvalidExecutable);
            try writer.writeAll(".complete");
        },
        .machine_result => |machine_expr| {
            const field_id = try machineFieldRef(ctx, machine_id, machine_expr);
            const field = ctx.module.hir.getMachineField(field_id);
            try writer.writeByte('(');
            try writer.writeAll("m->");
            try emitMachineFieldName(writer, ctx.module, field.name, machineFieldIndex(ctx, machine_id, field_id) orelse return error.InvalidExecutable);
            try writer.writeAll(".complete ? m->");
            try emitMachineFieldName(writer, ctx.module, field.name, machineFieldIndex(ctx, machine_id, field_id) orelse return error.InvalidExecutable);
            try writer.writeAll(".result : (cpt_panic(\"machine result cannot be read before completion\"), 0))");
        },
        .group => |inner| {
            try writer.writeByte('(');
            try emitMachineStateExpr(writer, ctx, machine_id, inner);
            try writer.writeByte(')');
        },
        .unary => |unary| {
            try writer.writeAll(unary.op.lexeme());
            try emitMachineStateExpr(writer, ctx, machine_id, unary.operand);
        },
        .binary => |binary| {
            try writer.writeByte('(');
            try emitMachineStateExpr(writer, ctx, machine_id, binary.left);
            try writer.writeByte(' ');
            try writer.writeAll(binary.op.lexeme());
            try writer.writeByte(' ');
            try emitMachineStateExpr(writer, ctx, machine_id, binary.right);
            try writer.writeByte(')');
        },
        else => {
            if (ctx.diagnostic_bag) |bag| try bag.append(diagnostics.machineSemanticsNotImplemented(expr.span));
            return error.InvalidExecutable;
        },
    }
}

fn machineFieldRef(ctx: *const BackendContext, machine_id: hir.MachineId, expr_id: hir.ExprId) EmitError!hir.MachineFieldId {
    const expr = ctx.module.hir.getExpr(expr_id).*;
    return switch (expr.kind) {
        .machine_field_ref => |field_id| blk: {
            const field = ctx.module.hir.getMachineField(field_id);
            if (field.parent.index != machine_id.index) return error.InvalidExecutable;
            break :blk field_id;
        },
        else => error.InvalidExecutable,
    };
}

fn machineFieldIndex(ctx: *const BackendContext, machine_id: hir.MachineId, field_id: hir.MachineFieldId) ?usize {
    const machine = ctx.module.hir.getMachine(machine_id);
    for (machine.fields, 0..) |candidate, index| {
        if (candidate.index == field_id.index) return index;
    }
    return null;
}

fn emitOpaqueAllocationForwardDeclarations(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var needs_arena = false;
    var needs_allocator = false;

    for (ctx.mir_module.store.functions.items) |function| {
        noteOpaqueAllocationPointerType(ctx, function.return_type, &needs_arena, &needs_allocator);
        for (function.params) |local_id| {
            noteOpaqueAllocationPointerType(ctx, ctx.mir_module.store.getLocal(local_id).type_id, &needs_arena, &needs_allocator);
        }
        for (function.locals) |local_id| {
            noteOpaqueAllocationPointerType(ctx, ctx.mir_module.store.getLocal(local_id).type_id, &needs_arena, &needs_allocator);
        }
    }

    if (needs_arena) try writer.writeAll("struct cpt_Arena;\n");
    if (needs_allocator) try writer.writeAll("struct cpt_Allocator;\n");
    return needs_arena or needs_allocator;
}

fn noteOpaqueAllocationPointerType(ctx: *const BackendContext, type_id: types.TypeId, needs_arena: *bool, needs_allocator: *bool) void {
    if (!ctx.module.types.contains(type_id)) return;
    switch (ctx.module.types.kind(type_id)) {
        .pointer => |pointer| noteOpaqueAllocationPointee(ctx, pointer.pointee, needs_arena, needs_allocator),
        else => {},
    }
}

fn noteOpaqueAllocationPointee(ctx: *const BackendContext, type_id: types.TypeId, needs_arena: *bool, needs_allocator: *bool) void {
    if (!ctx.module.types.contains(type_id)) return;
    switch (ctx.module.types.kind(type_id)) {
        .arena => needs_arena.* = true,
        .allocator => needs_allocator.* = true,
        .pointer => |pointer| noteOpaqueAllocationPointee(ctx, pointer.pointee, needs_arena, needs_allocator),
        else => {},
    }
}

fn emitArenaAllocHelperDeclaration(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    if (!mirContainsArenaAlloc(ctx)) return false;
    try writer.writeAll("void* cpt_arena_alloc(struct cpt_Arena* arena, size_t size, size_t align);\n");
    return true;
}

fn emitArenaAllocSizeHeader(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    if (!mirContainsArenaAlloc(ctx)) return false;
    try writer.writeAll("#include <stddef.h>\n");
    return true;
}

fn mirContainsArenaAlloc(ctx: *const BackendContext) bool {
    for (ctx.mir_module.store.blocks.items) |block| {
        for (block.statements) |statement| {
            switch (statement.kind) {
                .assign => |assignment| if (rvalueContainsArenaAlloc(assignment.rvalue)) return true,
                .drop, .machine_step, .panic, .assert_stmt => {},
            }
        }
    }
    return false;
}

fn rvalueContainsArenaAlloc(rvalue: mir.MirRvalue) bool {
    return switch (rvalue) {
        .arena_alloc => true,
        .machine_construct, .machine_complete, .machine_result => false,
        else => false,
    };
}

fn emitArenaStorageHelperDeclarations(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    const needs = mirContainsArenaStorageOps(ctx);
    if (needs.reset) try writer.writeAll("void cpt_arena_reset(struct cpt_Arena* arena);\n");
    if (needs.destroy) try writer.writeAll("void cpt_arena_destroy(struct cpt_Arena* arena);\n");
    return needs.reset or needs.destroy;
}

fn mirContainsArenaStorageOps(ctx: *const BackendContext) struct { reset: bool, destroy: bool } {
    var needs_reset = false;
    var needs_destroy = false;
    for (ctx.mir_module.store.blocks.items) |block| {
        for (block.statements) |statement| {
            switch (statement.kind) {
                .arena_reset => needs_reset = true,
                .arena_destroy => needs_destroy = true,
                else => {},
            }
        }
    }
    return .{ .reset = needs_reset, .destroy = needs_destroy };
}

fn emitPanicHelper(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    if (!mirContainsPanic(ctx)) return false;
    try writer.writeAll(
        \\#include <stdio.h>
        \\#include <stdlib.h>
        \\
        \\static void cpt_panic(const char* reason) {
        \\    fprintf(stderr, "panic: %s\n", reason);
        \\    exit(101);
        \\}
        \\
    );
    return true;
}

fn mirContainsPanic(ctx: *const BackendContext) bool {
    for (ctx.mir_module.store.blocks.items) |block| {
        for (block.statements) |statement| {
            switch (statement.kind) {
                .panic, .assert_stmt => return true,
                .assign => |assignment| if (rvalueContainsRuntimePanic(assignment.rvalue)) return true,
                else => {},
            }
        }
    }
    return false;
}

fn rvalueContainsRuntimePanic(rvalue: mir.MirRvalue) bool {
    return switch (rvalue) {
        .machine_result => true,
        else => false,
    };
}

fn isSupportedEnumLayout(ctx: *const BackendContext, enum_id: hir.EnumId) bool {
    if (enum_id.index >= ctx.module.hir.enums.items.len) return false;
    const enum_decl = ctx.module.hir.getEnum(enum_id);
    if (enum_decl.variants.len == 0) return false;
    for (enum_decl.variants) |variant_id| {
        const variant = ctx.module.hir.getVariant(variant_id);
        for (variant.payload_fields) |payload_id| {
            const payload_field = ctx.module.hir.getEnumPayloadField(payload_id);
            if (!isSupportedEnumPayloadType(ctx, payload_field.type_id)) return false;
        }
    }
    return true;
}

fn emitStructFieldCType(writer: anytype, ctx: *const BackendContext, type_id: types.TypeId, span: ?diagnostics.SourceSpan) EmitError!void {
    if (!ctx.module.types.contains(type_id) or !isSupportedStructFieldType(ctx, type_id)) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }
    try emitCType(writer, ctx, type_id, span);
}

// ─────────────────────────────────────────────────────────────────────────────
// Interface impl wrapper and vtable constant emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitInterfaceImplWrappersAndTables(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var emitted_any = false;
    for (ctx.module.hir.interface_impls.items, 0..) |interface_impl, index| {
        const impl_id = hir.InterfaceImplId{ .index = @intCast(index) };
        if (!usesInterfaceImpl(ctx, impl_id)) continue;
        if (emitted_any) try writer.writeByte('\n');
        try emitInterfaceImplWrappers(writer, ctx, impl_id, interface_impl);
        try writer.writeByte('\n');
        try emitInterfaceImplVTableConstant(writer, ctx, impl_id, interface_impl);
        emitted_any = true;
    }
    return emitted_any;
}

fn emitInterfaceImplWrappers(writer: anytype, ctx: *const BackendContext, impl_id: hir.InterfaceImplId, interface_impl: hir.HirInterfaceImpl) EmitError!void {
    const interface_decl = ctx.module.hir.getInterface(interface_impl.interface_id);
    for (interface_decl.requirements) |requirement_id| {
        const requirement = ctx.module.hir.getInterfaceRequirement(requirement_id);
        const function_id = findInterfaceImplFunction(ctx, interface_impl, requirement.name) orelse return error.InvalidExecutable;
        try emitInterfaceImplWrapper(writer, ctx, impl_id, interface_impl, function_id, requirement.*);
    }
}

fn emitInterfaceImplWrapper(writer: anytype, ctx: *const BackendContext, impl_id: hir.InterfaceImplId, interface_impl: hir.HirInterfaceImpl, function_id: hir.FunctionId, requirement: hir.HirInterfaceRequirement) EmitError!void {
    const function = ctx.module.hir.getFunction(function_id);
    try writer.writeAll("static ");
    try emitCType(writer, ctx, requirement.return_type, requirement.span);
    try writer.writeByte(' ');
    try emitInterfaceImplWrapperName(writer, ctx.module, impl_id, interface_impl, requirement.name);
    try writer.writeAll("(void* self");
    for (requirement.params, 0..) |param_id, index| {
        const param = ctx.module.hir.getInterfaceParam(param_id);
        try writer.writeAll(", ");
        try emitCType(writer, ctx, param.type_id, param.span);
        try writer.writeByte(' ');
        try emitInterfaceWrapperParamName(writer, index);
    }
    try writer.writeAll(") {\n");
    try writer.writeAll("    ");
    try emitCType(writer, ctx, interface_impl.target_type, interface_impl.span);
    try writer.writeAll("* typed = (");
    try emitCType(writer, ctx, interface_impl.target_type, interface_impl.span);
    try writer.writeAll("*)self;\n");
    try writer.writeAll("    ");
    if (!sameType(requirement.return_type, ctx.module.types.voidType())) try writer.writeAll("return ");
    try emitHirFunctionName(writer, ctx, function_id, function.*);
    try writer.writeAll("(*typed");
    for (requirement.params, 0..) |_, index| {
        try writer.writeAll(", ");
        try emitInterfaceWrapperParamName(writer, index);
    }
    try writer.writeAll(");\n");
    try writer.writeAll("}\n");
}

fn emitInterfaceImplVTableConstant(writer: anytype, ctx: *const BackendContext, impl_id: hir.InterfaceImplId, interface_impl: hir.HirInterfaceImpl) EmitError!void {
    const interface_decl = ctx.module.hir.getInterface(interface_impl.interface_id);
    try writer.writeAll("static const ");
    try emitInterfaceVTableTypeName(writer, ctx.module, interface_decl.name);
    try writer.writeByte(' ');
    try emitInterfaceImplVTableName(writer, ctx.module, interface_impl);
    try writer.writeAll(" = {\n");
    for (interface_decl.requirements) |requirement_id| {
        const requirement = ctx.module.hir.getInterfaceRequirement(requirement_id);
        try writer.writeAll("    .");
        try emitInterfaceRequirementSlotName(writer, ctx.module, requirement.name);
        try writer.writeAll(" = ");
        try emitInterfaceImplWrapperName(writer, ctx.module, impl_id, interface_impl, requirement.name);
        try writer.writeAll(",\n");
    }
    try writer.writeAll("};\n");
}

fn usesInterfaceImpl(ctx: *const BackendContext, impl_id: hir.InterfaceImplId) bool {
    for (ctx.mir_module.store.functions.items) |function| {
        for (function.blocks) |block_id| {
            const block = ctx.mir_module.store.getBlock(block_id);
            for (block.statements) |statement| {
                switch (statement.kind) {
                    .assign => |assignment| if (rvalueUsesInterfaceImpl(assignment.rvalue, impl_id)) return true,
                    else => {},
                }
            }
        }
    }
    return false;
}

fn rvalueUsesInterfaceImpl(rvalue: mir.MirRvalue, impl_id: hir.InterfaceImplId) bool {
    return switch (rvalue) {
        .dyn_coerce => |coerce| coerce.impl_id.index == impl_id.index,
        else => false,
    };
}

fn findInterfaceImplFunction(ctx: *const BackendContext, interface_impl: hir.HirInterfaceImpl, name: hir.SymbolId) ?hir.FunctionId {
    for (interface_impl.functions) |function_id| {
        const function = ctx.module.hir.getFunction(function_id);
        if (function.name.index == name.index) return function_id;
    }
    return null;
}

// ─────────────────────────────────────────────────────────────────────────────
// Function/prototype emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitPrototype(writer: anytype, ctx: *const BackendContext, function_id: mir.MirFunctionId, function: mir.MirFunction) EmitError!void {
    try emitCType(writer, ctx, function.return_type, function.source_span);
    try writer.writeByte(' ');
    try emitMirFunctionName(writer, ctx, function_id, function);
    try writer.writeByte('(');
    try emitParamList(writer, ctx, function, function.params);
    try writer.writeAll(");\n");
}

fn emitExternCPrototypes(writer: anytype, ctx: *const BackendContext) EmitError!bool {
    var emitted_any = false;
    for (ctx.module.hir.functions.items) |function| {
        if (!function.is_extern) continue;
        if (function.extern_abi == null or function.extern_abi.? != .c or function.c_symbol_name == null or ctx.module.interner.text(function.c_symbol_name.?).len == 0) {
            return error.InvalidExecutable;
        }
        try emitCType(writer, ctx, function.return_type, function.span);
        try writer.writeByte(' ');
        try emitCSymbolName(writer, ctx.module, function.c_symbol_name.?);
        try writer.writeByte('(');
        try emitExternCParamList(writer, ctx, function);
        try writer.writeAll(");\n");
        emitted_any = true;
    }
    return emitted_any;
}

fn emitExternCParamList(writer: anytype, ctx: *const BackendContext, function: hir.HirFunction) EmitError!void {
    if (function.params.len == 0) {
        try writer.writeAll("void");
        return;
    }
    for (function.params, 0..) |param_id, index| {
        const param = ctx.module.hir.getParam(param_id);
        if (index != 0) try writer.writeAll(", ");
        try emitCType(writer, ctx, param.type_id, param.span);
        try writer.writeByte(' ');
        try emitCSymbolName(writer, ctx.module, param.name);
    }
}

fn emitFunction(writer: anytype, ctx: *const BackendContext, function_id: mir.MirFunctionId, function: mir.MirFunction) EmitError!void {
    const hir_function = ctx.module.hir.getFunction(function.hir_function);
    if (hir_function.is_extern) return error.InvalidExecutable;
    switch (function.linkage) {
        .internal => if (hir_function.is_exported and hir_function.extern_abi == .c) return error.InvalidExecutable,
        .export_c => |export_c| {
            if (!hir_function.is_exported or hir_function.extern_abi == null or hir_function.extern_abi.? != .c) return error.InvalidExecutable;
            if (hir_function.c_symbol_name == null or hir_function.c_symbol_name.?.index != export_c.symbol.index or ctx.module.interner.text(export_c.symbol).len == 0) return error.InvalidExecutable;
        },
    }
    try emitCType(writer, ctx, function.return_type, function.source_span);
    try writer.writeByte(' ');
    try emitMirFunctionName(writer, ctx, function_id, function);
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
                .struct_constructor => |constructor| try emitStructConstructorAssignment(writer, ctx, assignment.place, constructor),
                else => {
                    try writer.writeAll("    ");
                    try emitPlace(writer, ctx, assignment.place);
                    try writer.writeAll(" = ");
                    try emitRvalue(writer, ctx, assignment.rvalue);
                    try writer.writeAll(";\n");
                },
            }
        },
        .call => |call| {
            try writer.writeAll("    ");
            try emitCall(writer, ctx, call);
            try writer.writeAll(";\n");
        },
        .interface_call => |call| {
            try writer.writeAll("    ");
            try emitInterfaceCall(writer, ctx, call);
            try writer.writeAll(";\n");
        },
        .drop => |drop| {
            const drop_function = ctx.module.hir.getFunction(drop.function);
            try writer.writeAll("    ");
            try emitHirFunctionName(writer, ctx, drop.function, drop_function.*);
            try writer.writeByte('(');
            try emitPlace(writer, ctx, drop.place);
            try writer.writeAll(");\n");
        },
        .arena_reset => |arena_operand| {
            try writer.writeAll("    cpt_arena_reset(");
            try emitOperand(writer, ctx, arena_operand);
            try writer.writeAll(");\n");
        },
        .arena_destroy => |arena_operand| {
            try writer.writeAll("    cpt_arena_destroy(");
            try emitOperand(writer, ctx, arena_operand);
            try writer.writeAll(");\n");
        },
        .machine_step => |machine_operand| {
            try writer.writeAll("    ");
            const machine_id = try machineIdFromOperand(ctx, machine_operand);
            try emitMachineStepFunctionName(writer, ctx.module, ctx.module.hir.getMachine(machine_id).name);
            try writer.writeAll("(&");
            try emitOperand(writer, ctx, machine_operand);
            try writer.writeAll(");\n");
        },
        .panic => |panic_stmt| {
            try writer.writeAll("    cpt_panic(");
            try emitCStringLiteral(writer, panic_stmt.reason);
            try writer.writeAll(");\n");
        },
        .assert_stmt => |assert_stmt| {
            try writer.writeAll("    if (!(");
            try emitOperand(writer, ctx, assert_stmt.condition);
            try writer.writeAll(")) {\n");
            try writer.writeAll("        cpt_panic(");
            try emitCStringLiteral(writer, assert_stmt.reason);
            try writer.writeAll(");\n");
            try writer.writeAll("    }\n");
        },
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Struct/enum constructor/tag/payload emission
// ─────────────────────────────────────────────────────────────────────────────

fn emitStructConstructorAssignment(writer: anytype, ctx: *const BackendContext, place: mir.MirPlace, constructor: anytype) EmitError!void {
    const struct_decl = ctx.module.hir.getStruct(constructor.struct_id);
    for (struct_decl.fields, 0..) |decl_field_id, field_index| {
        const value = structConstructorFieldValue(constructor.fields, decl_field_id) orelse return error.InvalidExecutable;
        const field = ctx.module.hir.getField(decl_field_id);
        try writer.writeAll("    ");
        try emitPlace(writer, ctx, place);
        try writer.writeByte('.');
        try emitStructFieldName(writer, ctx.module, field.name, field_index);
        try writer.writeAll(" = ");
        try emitOperand(writer, ctx, value);
        try writer.writeAll(";\n");
    }
}

fn structConstructorFieldValue(fields: []const mir.MirStructFieldValue, field_id: hir.FieldId) ?mir.MirOperand {
    for (fields) |field| {
        if (field.field_id.index == field_id.index) return field.value;
    }
    return null;
}

fn emitEnumConstructorAssignment(writer: anytype, ctx: *const BackendContext, place: mir.MirPlace, constructor: anytype) EmitError!void {
    const enum_decl = ctx.module.hir.getEnum(constructor.enum_id);
    const variant = ctx.module.hir.getVariant(constructor.variant_id);
    const variant_index = enumVariantIndex(ctx, enum_decl.*, constructor.variant_id) orelse return error.InvalidExecutable;

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

fn structFieldIndex(ctx: *const BackendContext, struct_decl: hir.HirStruct, field_id: hir.FieldId) ?usize {
    _ = ctx;
    for (struct_decl.fields, 0..) |candidate, index| {
        if (candidate.index == field_id.index) return index;
    }
    return null;
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
        .move => |place| try emitPlace(writer, ctx, place),
        .manual_init_assume => {
            try reportUnsupportedCType(ctx, null);
            return error.InvalidExecutable;
        },
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
        .dyn_coerce => |coerce| try emitDynCoerce(writer, ctx, coerce),
        .interface_call => |call| try emitInterfaceCall(writer, ctx, call),
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
            try emitCall(writer, ctx, call);
        },
        .arena_alloc => |arena_alloc| {
            try writer.writeByte('(');
            try emitCType(writer, ctx, arena_alloc.result_type, null);
            try writer.writeAll(")cpt_arena_alloc(");
            try emitOperand(writer, ctx, arena_alloc.arena_operand);
            try writer.writeAll(", sizeof(");
            try emitCType(writer, ctx, arena_alloc.allocated_type, null);
            try writer.writeAll("), _Alignof(");
            try emitCType(writer, ctx, arena_alloc.allocated_type, null);
            try writer.writeAll("))");
        },
        .machine_construct => |construct| {
            try emitMachineNewFunctionName(writer, ctx.module, ctx.module.hir.getMachine(construct.machine).name);
            try writer.writeByte('(');
            for (construct.args, 0..) |arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try emitOperand(writer, ctx, arg);
            }
            try writer.writeByte(')');
        },
        .machine_complete => |operand| {
            try emitOperand(writer, ctx, operand);
            try writer.writeAll(".complete");
        },
        .machine_result => |operand| {
            try writer.writeByte('(');
            try emitOperand(writer, ctx, operand);
            try writer.writeAll(".complete ? ");
            try emitOperand(writer, ctx, operand);
            try writer.writeAll(".result : (cpt_panic(\"machine result cannot be read before completion\"), 0))");
        },
        .enum_constructor, .struct_constructor => unreachable,
        .enum_tag => |operand| {
            try emitOperand(writer, ctx, operand);
            try writer.writeAll(".tag");
        },
        .field_access => |field_access| {
            const field = ctx.module.hir.getField(field_access.field_id);
            const struct_decl = ctx.module.hir.getStruct(field.parent);
            const field_index = structFieldIndex(ctx, struct_decl.*, field_access.field_id) orelse return error.InvalidExecutable;
            try emitOperand(writer, ctx, field_access.receiver);
            try writer.writeByte('.');
            try emitStructFieldName(writer, ctx.module, field.name, field_index);
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

fn emitCall(writer: anytype, ctx: *const BackendContext, call: mir.MirCall) EmitError!void {
    switch (call.callee) {
        .internal => |function_id| {
            const callee = ctx.module.hir.getFunction(function_id);
            try emitHirFunctionName(writer, ctx, function_id, callee.*);
        },
        .extern_c => |extern_c| try emitCSymbolName(writer, ctx.module, extern_c.symbol),
    }
    try writer.writeByte('(');
    for (call.args, 0..) |arg, index| {
        if (index != 0) try writer.writeAll(", ");
        try emitOperand(writer, ctx, arg);
    }
    try writer.writeByte(')');
}

fn emitDynCoerce(writer: anytype, ctx: *const BackendContext, coerce: anytype) EmitError!void {
    const interface_decl = ctx.module.hir.getInterface(coerce.interface_id);
    const interface_impl = ctx.module.hir.getInterfaceImpl(coerce.impl_id);
    try writer.writeByte('(');
    try emitDynInterfaceTypeName(writer, ctx.module, interface_decl.name);
    try writer.writeAll("){ .data = &");
    try emitPlace(writer, ctx, coerce.source);
    try writer.writeAll(", .vtable = &");
    try emitInterfaceImplVTableName(writer, ctx.module, interface_impl.*);
    try writer.writeAll(" }");
}

fn emitInterfaceCall(writer: anytype, ctx: *const BackendContext, call: mir.MirInterfaceCall) EmitError!void {
    const requirement = ctx.module.hir.getInterfaceRequirement(call.requirement_id);
    try writer.writeByte('(');
    try emitOperand(writer, ctx, call.receiver);
    try writer.writeAll(").vtable->");
    try emitInterfaceRequirementSlotName(writer, ctx.module, requirement.name);
    try writer.writeByte('(');
    try writer.writeByte('(');
    try emitOperand(writer, ctx, call.receiver);
    try writer.writeAll(").data");
    for (call.args) |arg| {
        try writer.writeAll(", ");
        try emitOperand(writer, ctx, arg);
    }
    try writer.writeByte(')');
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
        .field => |field_place| {
            const field = ctx.module.hir.getField(field_place.field_id);
            const struct_decl = ctx.module.hir.getStruct(field.parent);
            const field_index = structFieldIndex(ctx, struct_decl.*, field_place.field_id) orelse return error.InvalidExecutable;
            try emitLocalName(writer, ctx.module, ctx.mir_module, field_place.base);
            try writer.writeByte('.');
            try emitStructFieldName(writer, ctx.module, field.name, field_index);
        },
    }
}

fn machineIdFromOperand(ctx: *const BackendContext, operand: mir.MirOperand) EmitError!hir.MachineId {
    const place = switch (operand) {
        .copy => |place| place,
        else => return error.InvalidExecutable,
    };
    const local_id = switch (place) {
        .local => |local_id| local_id,
        else => return error.InvalidExecutable,
    };
    return switch (ctx.module.types.kind(ctx.mir_module.store.getLocal(local_id).type_id)) {
        .machine_type => |machine_id| machine_id,
        else => error.InvalidExecutable,
    };
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

const CTypePosition = enum {
    value,
    pointer_pointee,
};

fn emitCType(writer: anytype, ctx: *const BackendContext, type_id: types.TypeId, span: ?diagnostics.SourceSpan) EmitError!void {
    try emitCTypeAt(writer, ctx, type_id, span, .value);
}

fn emitCTypeAt(writer: anytype, ctx: *const BackendContext, type_id: types.TypeId, span: ?diagnostics.SourceSpan, position: CTypePosition) EmitError!void {
    if (!ctx.module.types.contains(type_id)) {
        try reportUnsupportedCType(ctx, span);
        return error.InvalidExecutable;
    }

    switch (ctx.module.types.kind(type_id)) {
        .void => try writer.writeAll("void"),
        .int, .bool, .alloc_error => try writer.writeAll("int"),
        .arena => {
            if (position != .pointer_pointee) {
                try reportUnsupportedCType(ctx, span);
                return error.InvalidExecutable;
            }
            try writer.writeAll("struct cpt_Arena");
        },
        .allocator => {
            if (position != .pointer_pointee) {
                try reportUnsupportedCType(ctx, span);
                return error.InvalidExecutable;
            }
            try writer.writeAll("struct cpt_Allocator");
        },
        .enum_type => |enum_id| {
            try requireSupportedEnumLayout(ctx, enum_id, span);
            try emitEnumTypeName(writer, ctx.module, ctx.module.hir.getEnum(enum_id).name);
        },
        .pointer => |pointer| {
            try emitCTypeAt(writer, ctx, pointer.pointee, span, .pointer_pointee);
            try writer.writeByte('*');
        },
        .struct_type => |struct_id| {
            try requireSupportedStructLayout(ctx, struct_id, span);
            if (position == .pointer_pointee) {
                try writer.writeAll("struct ");
            }
            try emitStructTypeName(writer, ctx, struct_id);
        },
        .machine_type => |machine_id| {
            try emitMachineTypeName(writer, ctx.module, ctx.module.hir.getMachine(machine_id).name);
        },
        .dyn_interface => |dyn| {
            try emitDynInterfaceTypeName(writer, ctx.module, ctx.module.hir.getInterface(dyn.interface_id).name);
        },
        .interface_type => {
            try reportInterfaceRuntimeUnsupported(ctx, span);
            return error.InvalidExecutable;
        },
        .manual_init, .type_param => {
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

fn reportInterfaceRuntimeUnsupported(ctx: *const BackendContext, span: ?diagnostics.SourceSpan) !void {
    if (ctx.diagnostic_bag) |bag| {
        try bag.append(diagnostics.interfaceRuntimeUnsupported(span orelse hir.synthetic_span));
    }
}

fn sameType(left: types.TypeId, right: types.TypeId) bool {
    return left.index == right.index;
}

// ─────────────────────────────────────────────────────────────────────────────
// C name rendering / escaping
// ─────────────────────────────────────────────────────────────────────────────

fn emitEnumTypeName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_enum_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitStructTypeName(writer: anytype, ctx: *const BackendContext, struct_id: hir.StructId) !void {
    const struct_decl = ctx.module.hir.getStruct(struct_id);
    try writer.writeAll("cpt_struct_");
    try emitEscapedIdentifierComponent(writer, ctx.module.interner.text(struct_decl.name));
    if (structNameCollides(ctx, struct_id, struct_decl.*)) {
        try writer.print("_{d}", .{struct_id.index});
    }
}

fn structNameCollides(ctx: *const BackendContext, struct_id: hir.StructId, struct_decl: hir.HirStruct) bool {
    for (ctx.module.hir.structs.items, 0..) |candidate, index| {
        if (index == struct_id.index) continue;
        if (candidate.name.index == struct_decl.name.index) return true;
    }
    return false;
}

fn emitDynInterfaceTypeName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_dyn_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitInterfaceVTableTypeName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_itf_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
    try writer.writeAll("_vtable");
}

fn emitMachineTypeName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_m_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitMachineStateTypeName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try emitMachineTypeName(writer, module, symbol);
    try writer.writeAll("_state");
}

fn emitMachineStateName(writer: anytype, module: *const semantics.SemanticModule, machine_symbol: hir.SymbolId, state_symbol: hir.SymbolId) !void {
    try emitMachineTypeName(writer, module, machine_symbol);
    try writer.writeAll("_s_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(state_symbol));
}

fn emitMachineNewFunctionName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try emitMachineTypeName(writer, module, symbol);
    try writer.writeAll("_new");
}

fn emitMachineStepFunctionName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try emitMachineTypeName(writer, module, symbol);
    try writer.writeAll("_step");
}

fn emitMachineParamName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_mp_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitMachineParamFieldName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll("cpt_mpf_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitMachineFieldName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId, field_index: usize) !void {
    try writer.writeAll("cpt_mf_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
    try writer.print("_{d}", .{field_index});
}

fn emitStructFieldName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId, field_index: usize) !void {
    try writer.writeAll("cpt_f_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
    try writer.print("_{d}", .{field_index});
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

fn emitCStringLiteral(writer: anytype, text: []const u8) !void {
    try writer.writeByte('"');
    for (text) |byte| {
        switch (byte) {
            '\\' => try writer.writeAll("\\\\"),
            '"' => try writer.writeAll("\\\""),
            '\n' => try writer.writeAll("\\n"),
            '\r' => try writer.writeAll("\\r"),
            '\t' => try writer.writeAll("\\t"),
            else => try writer.writeByte(byte),
        }
    }
    try writer.writeByte('"');
}

fn emitInterfaceRequirementSlotName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try emitEscapedIdentifierComponent(writer, module.interner.text(symbol));
}

fn emitInterfaceWrapperParamName(writer: anytype, index: usize) !void {
    try writer.print("arg_{d}", .{index});
}

fn emitInterfaceImplVTableName(writer: anytype, module: *const semantics.SemanticModule, interface_impl: hir.HirInterfaceImpl) !void {
    try writer.writeAll("cpt_impl_");
    try emitTypeNameComponent(writer, module, interface_impl.target_type);
    try writer.writeAll("_as_");
    try emitEscapedIdentifierComponent(writer, module.interner.text(module.hir.getInterface(interface_impl.interface_id).name));
}

fn emitInterfaceImplWrapperName(writer: anytype, module: *const semantics.SemanticModule, impl_id: hir.InterfaceImplId, interface_impl: hir.HirInterfaceImpl, requirement_name: hir.SymbolId) !void {
    try emitInterfaceImplVTableName(writer, module, interface_impl);
    try writer.writeByte('_');
    try emitEscapedIdentifierComponent(writer, module.interner.text(requirement_name));
    try writer.print("_{d}", .{impl_id.index});
}

fn emitMirFunctionName(writer: anytype, ctx: *const BackendContext, function_id: mir.MirFunctionId, function: mir.MirFunction) !void {
    switch (function.linkage) {
        .internal => try emitHirFunctionName(writer, ctx, function.hir_function, ctx.module.hir.getFunction(function.hir_function).*),
        .export_c => |export_c| try emitCSymbolName(writer, ctx.module, export_c.symbol),
    }
    _ = function_id;
}

fn emitHirFunctionName(writer: anytype, ctx: *const BackendContext, function_id: hir.FunctionId, function: hir.HirFunction) !void {
    if (findInterfaceImplForFunction(ctx, function_id)) |interface_impl| {
        try writer.writeAll("cpt_ifn_");
        try emitTypeNameComponent(writer, ctx.module, interface_impl.target_type);
        try writer.writeAll("_as_");
        try emitEscapedIdentifierComponent(writer, ctx.module.interner.text(ctx.module.hir.getInterface(interface_impl.interface_id).name));
        try writer.writeByte('_');
        try emitEscapedIdentifierComponent(writer, ctx.module.interner.text(function.name));
        try writer.print("_{d}", .{function_id.index});
        return;
    }
    if (function.is_exported and function.extern_abi == .c) {
        if (function.c_symbol_name == null or ctx.module.interner.text(function.c_symbol_name.?).len == 0) return error.InvalidExecutable;
        try emitCSymbolName(writer, ctx.module, function.c_symbol_name.?);
        return;
    }
    try emitFunctionName(writer, ctx.module, function.name);
    if (functionNameCollides(ctx, function_id, function)) {
        try writer.print("_{d}", .{function_id.index});
    }
}

fn functionNameCollides(ctx: *const BackendContext, function_id: hir.FunctionId, function: hir.HirFunction) bool {
    for (ctx.module.hir.functions.items, 0..) |candidate, index| {
        if (index == function_id.index) continue;
        if (candidate.is_extern) continue;
        if (candidate.is_exported and candidate.extern_abi == .c) continue;
        if (candidate.name.index == function.name.index) return true;
    }
    return false;
}

fn findInterfaceImplForFunction(ctx: *const BackendContext, function_id: hir.FunctionId) ?hir.HirInterfaceImpl {
    for (ctx.module.hir.interface_impls.items) |interface_impl| {
        for (interface_impl.functions) |candidate| {
            if (candidate.index == function_id.index) return interface_impl;
        }
    }
    return null;
}

fn emitTypeNameComponent(writer: anytype, module: *const semantics.SemanticModule, type_id: types.TypeId) !void {
    switch (module.types.kind(type_id)) {
        .int => try writer.writeAll("int"),
        .bool => try writer.writeAll("bool"),
        .struct_type => |struct_id| try emitEscapedIdentifierComponent(writer, module.interner.text(module.hir.getStruct(struct_id).name)),
        .enum_type => |enum_id| try emitEscapedIdentifierComponent(writer, module.interner.text(module.hir.getEnum(enum_id).name)),
        .machine_type => |machine_id| try emitEscapedIdentifierComponent(writer, module.interner.text(module.hir.getMachine(machine_id).name)),
        .pointer => |pointer| {
            try emitTypeNameComponent(writer, module, pointer.pointee);
            try writer.writeAll("_ptr");
        },
        .dyn_interface => |dyn| {
            try writer.writeAll("dyn_");
            try emitEscapedIdentifierComponent(writer, module.interner.text(module.hir.getInterface(dyn.interface_id).name));
        },
        .void, .arena, .allocator, .alloc_error, .interface_type, .manual_init, .type_param => try writer.writeAll("unsupported"),
    }
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

fn emitCSymbolName(writer: anytype, module: *const semantics.SemanticModule, symbol: hir.SymbolId) !void {
    try writer.writeAll(module.interner.text(symbol));
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

test "MIR C backend omits static assertions" {
    const c_source = try emitForTest("module Main; static_assert(1 + 1 == 2); int main() { return 0; }");
    defer std.testing.allocator.free(c_source);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "static_assert") == null);
    try std.testing.expectEqualStrings("int main(void) {\ncpt_bb_0:\n    return 0;\n}\n", c_source);
}

test "MIR C backend lowers target metadata only as constants" {
    const c_source = try emitForTest("module Main; static_assert(target.pointerSize == 8); int main() { return comptime target.pointerSize; }");
    defer std.testing.allocator.free(c_source);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "target") == null);
    try std.testing.expectEqualStrings("int main(void) {\ncpt_bb_0:\n    return 8;\n}\n", c_source);
}

test "MIR C backend omits compile-time-only functions" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\comptime int answer() {
        \\    return 42;
        \\}
        \\
        \\int main() {
        \\    return comptime answer();
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "answer") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "comptime") == null);
    try std.testing.expectEqualStrings("int main(void) {\ncpt_bb_0:\n    return 42;\n}\n", c_source);
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

fn emitManualModuleForTest(module: *semantics.SemanticModule, mir_module: *mir.MirModule) ![]const u8 {
    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    const c_source = try emitExecutableFromMir(std.testing.allocator, module, mir_module, &diagnostic_bag);
    errdefer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 0), diagnostic_bag.count());
    return c_source;
}

fn expectUnsupportedManualModule(module: *semantics.SemanticModule, mir_module: *mir.MirModule) !void {
    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, module, mir_module, &diagnostic_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostic_bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UnsupportedCBackendType, diagnostic_bag.diagnostics.items[0].code);
}

test "MIR C backend emits simple struct layout" {
    try expectEmit(
        "module Main; struct Vec2 { int x; int y; }; int main() { return 0; }",
        "typedef struct {\n    int cpt_f_x_0;\n    int cpt_f_y_1;\n} cpt_struct_Vec2;\n\nint main(void) {\ncpt_bb_0:\n    return 0;\n}\n",
    );
}

test "MIR C backend emits Drop cleanup before return in reverse order" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\struct File {
        \\    int fd;
        \\};
        \\
        \\impl Drop<File> {
        \\    void drop(File f) {
        \\        return;
        \\    }
        \\}
        \\
        \\int main() {
        \\    File a = File { fd: 1 };
        \\    File b = File { fd: 2 };
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    const drop_b = std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_b_2);").?;
    const drop_a = std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_a_1);").?;
    const ret = std.mem.indexOf(u8, c_source, "return 0;").?;
    try std.testing.expect(drop_b < drop_a);
    try std.testing.expect(drop_a < ret);
}

test "MIR C backend skips moved-from Drop local cleanup" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\struct File {
        \\    int fd;
        \\};
        \\
        \\impl Drop<File> {
        \\    void drop(File f) {
        \\        return;
        \\    }
        \\}
        \\
        \\int main() {
        \\    File f = File { fd: 1 };
        \\    File g = move f;
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_g_2);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_f_1);") == null);
}

test "MIR C backend return move does not drop returned source in callee" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\struct File {
        \\    int fd;
        \\};
        \\
        \\impl Drop<File> {
        \\    void drop(File f) {
        \\        return;
        \\    }
        \\}
        \\
        \\File makeFile() {
        \\    File f = File { fd: 7 };
        \\    return move f;
        \\}
        \\
        \\int main() {
        \\    File x = makeFile();
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_f_1);") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_x_3);") != null);
}

test "MIR C backend emits address-of field place" {
    const c_source = try emitForTest("module Main; struct Vec2 { int x; int y; }; int main() { Vec2 v = Vec2 { x: 7, y: 4, }; int* px = &v.x; unsafe { return *px; } }");
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "&cpt_l_v_0.cpt_f_x_0") != null);
}

test "MIR C backend emits struct layout with bool enum and pointer fields" {
    var module = try newTestModule();
    defer module.deinit();

    const status_id = try module.hir.addEnum(try internForTest(&module, "Status"), false);
    const status_type = try module.types.addEnumType(status_id);
    _ = try module.hir.addVariant(status_id, try internForTest(&module, "Ok"), hir.synthetic_span);
    _ = try module.hir.addVariant(status_id, try internForTest(&module, "Err"), hir.synthetic_span);

    const int_ptr = try module.types.addPointerType(module.types.intType());
    const bool_ptr = try module.types.addPointerType(module.types.boolType());
    const status_ptr = try module.types.addPointerType(status_type);

    const header_id = try module.hir.addStruct(try internForTest(&module, "Header"));
    _ = try module.types.addStructType(header_id);
    _ = try module.hir.addField(header_id, try internForTest(&module, "valid"), module.types.boolType(), hir.synthetic_span);
    _ = try module.hir.addField(header_id, try internForTest(&module, "status"), status_type, hir.synthetic_span);
    _ = try module.hir.addField(header_id, try internForTest(&module, "count"), int_ptr, hir.synthetic_span);
    _ = try module.hir.addField(header_id, try internForTest(&module, "flag"), bool_ptr, hir.synthetic_span);
    _ = try module.hir.addField(header_id, try internForTest(&module, "next"), status_ptr, hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int tag;\n} cpt_enum_Status;\n\ntypedef struct {\n    int cpt_f_valid_0;\n    cpt_enum_Status cpt_f_status_1;\n    int* cpt_f_count_2;\n    int* cpt_f_flag_3;\n    cpt_enum_Status* cpt_f_next_4;\n} cpt_struct_Header;\n\nvoid main(void) {\ncpt_bb_0:\n    return;\n}\n",
        c_source,
    );
}

test "MIR C backend emits escaped struct and field names" {
    var module = try newTestModule();
    defer module.deinit();

    const struct_id = try module.hir.addStruct(try internForTest(&module, "struct"));
    _ = try module.types.addStructType(struct_id);
    _ = try module.hir.addField(struct_id, try internForTest(&module, "return"), module.types.intType(), hir.synthetic_span);
    _ = try module.hir.addField(struct_id, try internForTest(&module, "field-name?"), module.types.boolType(), hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(
        "typedef struct {\n    int cpt_f_return__0;\n    int cpt_f_field_x2D_name_x3F_1;\n} cpt_struct_struct_;\n\nvoid main(void) {\ncpt_bb_0:\n    return;\n}\n",
        c_source,
    );
}

test "MIR C backend rejects empty struct layout" {
    var module = try newTestModule();
    defer module.deinit();

    const struct_id = try module.hir.addStruct(try internForTest(&module, "Empty"));
    _ = try module.types.addStructType(struct_id);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    try expectUnsupportedManualModule(&module, &mir_module);
}

test "MIR C backend rejects void struct field" {
    var module = try newTestModule();
    defer module.deinit();

    const struct_id = try module.hir.addStruct(try internForTest(&module, "Bad"));
    _ = try module.types.addStructType(struct_id);
    _ = try module.hir.addField(struct_id, try internForTest(&module, "nothing"), module.types.voidType(), hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    try expectUnsupportedManualModule(&module, &mir_module);
}

test "MIR C backend rejects struct by value field" {
    var module = try newTestModule();
    defer module.deinit();

    const vec_id = try module.hir.addStruct(try internForTest(&module, "Vec2"));
    const vec_type = try module.types.addStructType(vec_id);
    _ = try module.hir.addField(vec_id, try internForTest(&module, "x"), module.types.intType(), hir.synthetic_span);

    const box_id = try module.hir.addStruct(try internForTest(&module, "Box"));
    _ = try module.types.addStructType(box_id);
    _ = try module.hir.addField(box_id, try internForTest(&module, "value"), vec_type, hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    try expectUnsupportedManualModule(&module, &mir_module);
}

test "MIR C backend rejects struct pointer fields for P7-M1" {
    var module = try newTestModule();
    defer module.deinit();

    const node_id = try module.hir.addStruct(try internForTest(&module, "Node"));
    const node_type = try module.types.addStructType(node_id);
    const node_ptr = try module.types.addPointerType(node_type);
    _ = try module.hir.addField(node_id, try internForTest(&module, "next"), node_ptr, hir.synthetic_span);

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    try addVoidMainMirForTest(&module, &mir_module);

    try expectUnsupportedManualModule(&module, &mir_module);
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

test "MIR C backend emits extern C prototypes and symbol calls" {
    const c_source = try emitForTest(
        \\module Main;
        \\extern "C" {
        \\    int abs(int value);
        \\}
        \\int main() {
        \\    return abs(-3);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int abs(int value);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return abs(") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int abs(int value) {") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_abs") == null);
}

test "MIR C backend emits extern C prototype once for multiple calls" {
    const c_source = try emitForTest(
        \\module Main;
        \\extern "C" {
        \\    int abs(int value);
        \\}
        \\int main() {
        \\    return abs(-3) + abs(-4);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "int abs(int value);"));
    try std.testing.expectEqual(@as(usize, 2), countOccurrences(c_source, "abs(") - 1);
}

test "MIR C backend keeps ordinary function mangling distinct from extern symbols" {
    const c_source = try emitForTest(
        \\module Main;
        \\extern "C" {
        \\    int c_abs(int value);
        \\}
        \\int abs(int value) {
        \\    return value;
        \\}
        \\int main() {
        \\    return abs(1) + c_abs(2);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int c_abs(int value);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_abs(int cpt_p_value_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_abs(1)") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "c_abs(2)") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_c_abs") == null);
}

test "MIR C backend emits void extern C calls as statements" {
    const c_source = try emitForTest(
        \\module Main;
        \\extern "C" {
        \\    void observe(int value);
        \\}
        \\int main() {
        \\    observe(3);
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "void observe(int value);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "    observe(3);\n") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "void cpt_t_") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_observe") == null);
}

test "MIR C backend emits no extern C prototypes for empty extern block" {
    const c_source = try emitForTest(
        \\module Main;
        \\extern "C" {
        \\}
        \\int main() {
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "extern") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int main(void)") != null);
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

test "MIR C backend renders supported struct TypeId in local declarations" {
    var module = try newTestModule();
    defer module.deinit();

    const struct_id = try module.hir.addStruct(try internForTest(&module, "Vec3"));
    const struct_type = try module.types.addStructType(struct_id);
    _ = try module.hir.addField(struct_id, try internForTest(&module, "x"), module.types.intType(), hir.synthetic_span);
    const main_hir = try addHirFunctionForTest(&module, "main", module.types.voidType());

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    const main_mir = try mir_module.store.addFunction(main_hir, try internForTest(&module, "main"), module.types.voidType(), hir.synthetic_span);
    _ = try mir_module.store.addLocal(main_mir, try internForTest(&module, "value"), .user, struct_type, hir.synthetic_span);
    const block = try mir_module.store.addBlock(main_mir, hir.synthetic_span);
    try mir_module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(null) });

    const c_source = try emitManualModuleForTest(&module, &mir_module);
    defer std.testing.allocator.free(c_source);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec3 cpt_l_value_0;") != null);
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

test "MIR C backend emits opaque allocation handle pointer params" {
    const c_source = try emitForTest(
        \\module Main;
        \\noalloc int usesArena(Arena* arena) { return 0; }
        \\alloc int usesAllocator(Allocator* allocator) { return 0; }
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "struct cpt_Arena;\nstruct cpt_Allocator;\n") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_usesArena(struct cpt_Arena* cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_usesAllocator(struct cpt_Allocator* cpt_p_allocator_") != null);
}

test "MIR C backend emits explicit arena allocation helper call" {
    const c_source = try emitForTest(
        \\module Main;
        \\alloc int* make(Arena* arena) {
        \\    return Arena.alloc<int>(arena);
        \\}
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "#include <stddef.h>\n"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "void* cpt_arena_alloc(struct cpt_Arena* arena, size_t size, size_t align);"));
    try std.testing.expect(std.mem.indexOf(u8, c_source, "(int*)cpt_arena_alloc(cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ", sizeof(int), _Alignof(int))") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "malloc(") == null);
}

test "MIR C backend emits arena helper declarations once for multiple operations" {
    const c_source = try emitForTest(
        \\module Main;
        \\alloc int useArena(Arena* arena) {
        \\    int* first = Arena.alloc<int>(arena);
        \\    bool* second = Arena.alloc<bool>(arena);
        \\    Arena.reset(arena);
        \\    int* third = Arena.alloc<int>(arena);
        \\    Arena.destroy(arena);
        \\    return 0;
        \\}
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "#include <stddef.h>\n"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "void* cpt_arena_alloc(struct cpt_Arena* arena, size_t size, size_t align);"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "void cpt_arena_reset(struct cpt_Arena* arena);"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "void cpt_arena_destroy(struct cpt_Arena* arena);"));
    try std.testing.expectEqual(@as(usize, 3), countOccurrences(c_source, "cpt_arena_alloc(cpt_p_arena_"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "cpt_arena_reset(cpt_p_arena_"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "cpt_arena_destroy(cpt_p_arena_"));
    try std.testing.expect(std.mem.indexOf(u8, c_source, "malloc(") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "realloc(") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "free(") == null);
}

test "MIR C backend emits arena allocation size and alignment for bool and struct" {
    const c_source = try emitForTest(
        \\module Main;
        \\struct Vec2 {
        \\    int x;
        \\    bool ok;
        \\};
        \\alloc int useArena(Arena* arena) {
        \\    bool* flag = Arena.alloc<bool>(arena);
        \\    Vec2* vec = Arena.alloc<Vec2>(arena);
        \\    return 0;
        \\}
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "(int*)cpt_arena_alloc(cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ", sizeof(int), _Alignof(int))") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "(cpt_struct_Vec2*)cpt_arena_alloc(cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ", sizeof(cpt_struct_Vec2), _Alignof(cpt_struct_Vec2))") != null);
}

test "MIR C backend emits explicit arena reset and destroy helper calls" {
    const c_source = try emitForTest(
        \\module Main;
        \\noalloc int resetOnly(Arena* arena) {
        \\    Arena.reset(arena);
        \\    return 0;
        \\}
        \\alloc int destroyAfterAlloc(Arena* arena) {
        \\    int* value = Arena.alloc<int>(arena);
        \\    Arena.destroy(arena);
        \\    return 0;
        \\}
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "void cpt_arena_reset(struct cpt_Arena* arena);"));
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "void cpt_arena_destroy(struct cpt_Arena* arena);"));
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_arena_reset(cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_arena_destroy(cpt_p_arena_") != null);
}

test "MIR C backend does not emit Drop cleanup for arena pointer locals or reset destroy" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\struct File {
        \\    int handle;
        \\};
        \\
        \\impl Drop<File> {
        \\    void drop(File f) {
        \\        return;
        \\    }
        \\}
        \\
        \\alloc int useArena(Arena* arena) {
        \\    int* value = Arena.alloc<int>(arena);
        \\    Arena.reset(arena);
        \\    Arena.destroy(arena);
        \\    return 0;
        \\}
        \\
        \\int main() {
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_arena_reset(cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_arena_destroy(cpt_p_arena_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_drop(cpt_l_value_") == null);
}

test "MIR C backend emits AllocError as value placeholder" {
    const c_source = try emitForTest(
        \\module Main;
        \\noalloc AllocError id(AllocError err) { return err; }
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_id(int cpt_p_err_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_p_err_") != null);
}

test "MIR C backend rejects by-value opaque allocation handles" {
    var module = try newTestModule();
    defer module.deinit();

    const bad_hir = try addHirFunctionForTest(&module, "bad", module.types.arenaType());

    var mir_module = mir.MirModule.init(std.testing.allocator);
    defer mir_module.deinit();
    _ = try mir_module.store.addFunction(bad_hir, try internForTest(&module, "bad"), module.types.arenaType(), hir.synthetic_span);

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try std.testing.expectError(error.InvalidExecutable, emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostic_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostic_bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UnsupportedCBackendType, diagnostic_bag.diagnostics.items[0].code);
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

test "MIR C backend emits struct value params returns and calls" {
    const c_source = try emitForTest(
        \\module Main;
        \\struct Vec2 { int x; int y; };
        \\Vec2 makeVec(int x, int y) { return Vec2 { x: x, y: y, }; }
        \\int sum(Vec2 v) { return v.x + v.y; }
        \\int main() { Vec2 v = makeVec(3, 4); return sum(v); }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec2 cpt_f_makeVec(int cpt_p_x_0, int cpt_p_y_1);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_sum(cpt_struct_Vec2 cpt_p_v_2);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec2 cpt_l_v_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, " = cpt_f_makeVec(3, 4);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_f_sum(cpt_l_v_") != null);
}

test "MIR C backend emits supported struct raw pointer params and returns" {
    const c_source = try emitForTest(
        \\module Main;
        \\struct Box { int value; };
        \\Box* id(Box* p) { return p; }
        \\int main() { return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef struct {\n    int cpt_f_value_0;\n} cpt_struct_Box;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Box* cpt_f_id(cpt_struct_Box* cpt_p_p_0);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_p_p_0;") != null);
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

test "MIR C backend Phase 7 closeout struct runtime snapshot" {
    const c_source = try emitForTest(
        \\module Main;
        \\struct Vec2 { int x; int y; };
        \\Vec2 makeVec(int x, int y) { return Vec2 { x: x, y: y, }; }
        \\int sum(Vec2 v) { return v.x + v.y; }
        \\int main() {
        \\    Vec2 v = makeVec(3, 4);
        \\    v.x = 11;
        \\    int* px = &v.x;
        \\    return sum(v);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef struct {\n    int cpt_f_x_0;\n    int cpt_f_y_1;\n} cpt_struct_Vec2;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec2 cpt_f_makeVec(int cpt_p_x_0, int cpt_p_y_1);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_sum(cpt_struct_Vec2 cpt_p_v_2);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec2 cpt_l_v_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".cpt_f_x_0 = 11;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "&cpt_l_v_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".cpt_f_x_0") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, " = cpt_f_makeVec(3, 4);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_f_sum(cpt_l_v_") != null);
}

test "MIR C backend stabilizes Phase 8 concrete generic pipeline" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\concept Equatable<T> { bool equals(T left, T right); };
        \\concept Hashable<T> { int hash(T value); };
        \\marker concept Copy<T>;
        \\
        \\struct Vec2 { int x; int y; };
        \\
        \\impl Equatable<Vec2> { bool equals(Vec2 left, Vec2 right) { return left.x == right.x && left.y == right.y; } }
        \\impl Hashable<Vec2> { int hash(Vec2 value) { return value.x; } }
        \\impl Copy<Vec2>;
        \\
        \\template<T> T identity(T value) { return value; }
        \\template<T: Copy<T>> T pass(T value) { return value; }
        \\template<T: Equatable<T>> bool areEqual(T left, T right) { return equals(left, right); }
        \\
        \\int main() {
        \\    int a = identity(3);
        \\    int b = identity(4);
        \\    Vec2 left = identity(Vec2 { x: a, y: 2, });
        \\    Vec2 right = pass(Vec2 { x: b + -1, y: 2, });
        \\    if (areEqual(left, right)) { return 8; }
        \\    return 1;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_identity__int(int cpt_p_value_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec2 cpt_f_identity__struct_Vec2(cpt_struct_Vec2 cpt_p_value_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Vec2 cpt_f_pass__struct_Vec2(cpt_struct_Vec2 cpt_p_value_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_areEqual__struct_Vec2(cpt_struct_Vec2 cpt_p_left_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_equals(cpt_struct_Vec2 cpt_p_left_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_f_equals(cpt_p_left_") != null);

    try std.testing.expectEqual(@as(usize, 3), countOccurrences(c_source, "cpt_f_identity__int("));
    try std.testing.expectEqual(@as(usize, 3), countOccurrences(c_source, "cpt_f_identity__struct_Vec2("));
    try std.testing.expectEqual(@as(usize, 2), countOccurrences(c_source, "cpt_f_areEqual__struct_Vec2("));
    try std.testing.expectEqual(@as(usize, 2), countOccurrences(c_source, "cpt_f_equals("));
    try std.testing.expectEqual(@as(usize, 0), countOccurrences(c_source, "cpt_f_hash("));
    try std.testing.expect(std.mem.indexOf(u8, c_source, "concept") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "template") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "type_param") == null);
}

test "MIR C backend emits deterministic generic instantiation names" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\template<T>
        \\T identity(T value) {
        \\    return value;
        \\}
        \\
        \\int main() {
        \\    int a = identity(3);
        \\    int b = identity(4);
        \\    if (identity(true)) {
        \\        return a + b;
        \\    }
        \\    return 0;
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_identity__int(int cpt_p_value_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "int cpt_f_identity__bool(int cpt_p_value_") != null);
    try std.testing.expectEqual(@as(usize, 2), countOccurrences(c_source, "cpt_f_identity__int("));
    try std.testing.expectEqual(@as(usize, 2), countOccurrences(c_source, "cpt_f_identity__bool("));
}

test "MIR C backend emits machine frame and step runtime" {
    const c_source = try emitForTest(
        \\module Main;
        \\
        \\machine Door() -> int {
        \\    state Closed {
        \\        transition Open;
        \\    }
        \\
        \\    state Open {
        \\        return 1;
        \\    }
        \\}
        \\
        \\int main() {
        \\    Door m = Door();
        \\    Step(m);
        \\    Step(m);
        \\    return Result(m);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef enum {\n    cpt_m_Door_s_Closed = 0,\n    cpt_m_Door_s_Open = 1,") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef struct {\n    cpt_m_Door_state state;\n    int complete;\n    int result;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "static void cpt_m_Door_step(cpt_m_Door* m)") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "    cpt_m_Door_step(&cpt_l_m_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "= cpt_m_Door_step(") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "m->state = cpt_m_Door_s_Open;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "m->result = 1;\n            m->complete = 1;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "malloc") == null);
}

test "MIR C backend lowers dyn dispatch to vtable and fat reference" {
    const c_source = try emitForTest(
        \\module Main;
        \\interface Counter { int Value(); };
        \\struct Box { int value; };
        \\impl Counter<Box> {
        \\    int Value(mut Box& self) { return self.value; }
        \\}
        \\int Read(mut dyn Counter& counter) { return counter.Value(); }
        \\int main() { Box box = Box { value: 7 }; return Read(box); }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef struct {\n    int (*Value)(void* self);\n} cpt_itf_Counter_vtable;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "typedef struct {\n    void* data;\n    const cpt_itf_Counter_vtable* vtable;\n} cpt_dyn_Counter;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "static int cpt_impl_Box_as_Counter_Value_0(void* self)") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_struct_Box* typed = (cpt_struct_Box*)self;") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "return cpt_ifn_Box_as_Counter_Value_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "static const cpt_itf_Counter_vtable cpt_impl_Box_as_Counter") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".Value = cpt_impl_Box_as_Counter_Value_0") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".data = &cpt_l_box_") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ".vtable = &cpt_impl_Box_as_Counter") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ").vtable->Value((") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ").data") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "malloc") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "calloc") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "realloc") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "free(") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "RTTI") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "dynamic_cast") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "reflection") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_scheduler") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_async") == null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "class ") == null);
}

test "MIR C backend lowers void dyn interface calls as statements" {
    const c_source = try emitForTest(
        \\module Main;
        \\interface Sink { void Put(int value); };
        \\struct Box { int value; };
        \\impl Sink<Box> {
        \\    void Put(mut Box& self, int value) { return; }
        \\}
        \\void Emit(mut dyn Sink& sink) { sink.Put(7); }
        \\int main() { Box box = Box { value: 0 }; Emit(box); return 0; }
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "void (*Put)(void* self, int);") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "static void cpt_impl_Box_as_Sink_Put_0(void* self, int arg_0)") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, ").vtable->Put((") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "void cpt_t_") == null);
}

test "MIR C backend emits distinct dyn vtables for repeated interface shapes" {
    const c_source = try emitForTest(
        \\module Main;
        \\interface Counter { int Value(); };
        \\interface Other { int Value(); };
        \\struct Left { int value; };
        \\struct Right { int value; };
        \\impl Counter<Left> { int Value(mut Left& self) { return self.value; } }
        \\impl Counter<Right> { int Value(mut Right& self) { return self.value; } }
        \\impl Other<Left> { int Value(mut Left& self) { return self.value + 1; } }
        \\int ReadCounter(mut dyn Counter& counter) { return counter.Value(); }
        \\int ReadOther(mut dyn Other& other) { return other.Value(); }
        \\int main() {
        \\    Left left = Left { value: 1 };
        \\    Right right = Right { value: 2 };
        \\    return ReadCounter(left) + ReadCounter(right) + ReadOther(left);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_impl_Left_as_Counter") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_impl_Right_as_Counter") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_impl_Left_as_Other") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_itf_Counter_vtable") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_itf_Other_vtable") != null);
}

test "MIR C backend preserves interface requirement slot order" {
    const c_source = try emitForTest(
        \\module Main;
        \\interface Pair {
        \\    int First();
        \\    int Second(int value);
        \\};
        \\struct Box { int value; };
        \\impl Pair<Box> {
        \\    int First(mut Box& self) { return self.value; }
        \\    int Second(mut Box& self, int value) { return self.value + value; }
        \\}
        \\int Read(mut dyn Pair& pair) { return pair.First() + pair.Second(3); }
        \\int main() { Box box = Box { value: 2 }; return Read(box); }
    );
    defer std.testing.allocator.free(c_source);

    const slot_first = std.mem.indexOf(u8, c_source, "int (*First)(void* self);") orelse return error.TestExpectedEqual;
    const slot_second = std.mem.indexOf(u8, c_source, "int (*Second)(void* self, int);") orelse return error.TestExpectedEqual;
    try std.testing.expect(slot_first < slot_second);

    const table_first = std.mem.indexOf(u8, c_source, ".First = cpt_impl_Box_as_Pair_First_0") orelse return error.TestExpectedEqual;
    const table_second = std.mem.indexOf(u8, c_source, ".Second = cpt_impl_Box_as_Pair_Second_0") orelse return error.TestExpectedEqual;
    try std.testing.expect(table_first < table_second);
}

test "MIR C backend reuses one vtable constant for repeated dyn coercions" {
    const c_source = try emitForTest(
        \\module Main;
        \\interface Counter { int Value(); };
        \\struct Box { int value; };
        \\impl Counter<Box> {
        \\    int Value(mut Box& self) { return self.value; }
        \\}
        \\int Read(mut dyn Counter& counter) { return counter.Value(); }
        \\int main() {
        \\    Box box = Box { value: 3 };
        \\    return Read(box) + Read(box);
        \\}
    );
    defer std.testing.allocator.free(c_source);

    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "static const cpt_itf_Counter_vtable cpt_impl_Box_as_Counter"));
    try std.testing.expectEqual(@as(usize, 2), countOccurrences(c_source, ".vtable = &cpt_impl_Box_as_Counter"));
}

fn countOccurrences(haystack: []const u8, needle: []const u8) usize {
    var count: usize = 0;
    var start: usize = 0;
    while (std.mem.indexOf(u8, haystack[start..], needle)) |offset| {
        count += 1;
        start += offset + needle.len;
    }
    return count;
}

test "MIR C backend lowers assert to conditional panic" {
    const c_source = try emitForTest("module Main; int main() { bool ok = true; assert(ok, \"ok reason\"); return 0; }");
    defer std.testing.allocator.free(c_source);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "if (!(cpt_l_ok_0)) {") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_panic(\"ok reason\");") != null);
}

test "MIR C backend emits panic helper once for multiple asserts" {
    const c_source = try emitForTest("module Main; int main() { assert(true, \"one\"); assert(true, \"two\"); return 0; }");
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "static void cpt_panic(const char* reason)"));
}

test "MIR C backend emits panic helper once for panic and assert" {
    const c_source = try emitForTest("module Main; int main() { assert(true, \"assert reason\"); panic(\"panic reason\"); return 0; }");
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 1), countOccurrences(c_source, "static void cpt_panic(const char* reason)"));
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_panic(\"assert reason\");") != null);
    try std.testing.expect(std.mem.indexOf(u8, c_source, "cpt_panic(\"panic reason\");") != null);
}
