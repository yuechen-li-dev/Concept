const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const hir_checker = @import("hir_checker.zig");
const mir = @import("mir.zig");
const mir_lowering = @import("mir_lowering.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");

pub const ValidationError = error{InvalidMirModule} || std.mem.Allocator.Error;

const synthetic_span = hir.synthetic_span;

pub fn validateModule(
    allocator: std.mem.Allocator,
    semantic_module: *const semantics.SemanticModule,
    mir_module: *const mir.MirModule,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
) ValidationError!void {
    var validator = Validator{
        .allocator = allocator,
        .semantic_module = semantic_module,
        .mir_module = mir_module,
        .diagnostics = diagnostic_bag,
        .had_error = false,
    };
    try validator.validateModule();
    if (validator.had_error) return error.InvalidMirModule;
}

const Validator = struct {
    allocator: std.mem.Allocator,
    semantic_module: *const semantics.SemanticModule,
    mir_module: *const mir.MirModule,
    diagnostics: ?*diagnostics.DiagnosticBag,
    had_error: bool,

    fn validateModule(self: *Validator) ValidationError!void {
        for (self.mir_module.store.functions.items, 0..) |mir_function, index| {
            try self.validateFunction(.{ .index = @intCast(index) }, mir_function);
        }
        for (self.mir_module.store.locals.items, 0..) |mir_local, index| {
            try self.validateLocal(.{ .index = @intCast(index) }, mir_local);
        }
        for (self.mir_module.store.blocks.items, 0..) |mir_block_value, index| {
            try self.validateBlock(.{ .index = @intCast(index) }, mir_block_value);
        }
    }

    fn validateFunction(self: *Validator, function_id: mir.MirFunctionId, mir_function: mir.MirFunction) ValidationError!void {
        if (!self.containsHirFunction(mir_function.hir_function)) {
            try self.report(.InvalidMirOperand, mir_function.source_span, diagnostics.invalidMirOperand);
        } else {
            const hir_function = self.semantic_module.hir.getFunction(mir_function.hir_function);
            if (hir_function.is_extern) {
                try self.report(.InvalidMirOperand, mir_function.source_span, diagnostics.invalidMirOperand);
            }
            switch (mir_function.linkage) {
                .internal => {
                    if (hir_function.is_exported and hir_function.extern_abi == .c) {
                        try self.report(.InvalidMirOperand, mir_function.source_span, diagnostics.invalidMirOperand);
                    }
                },
                .export_c => |export_c| {
                    if (!hir_function.is_exported or hir_function.extern_abi == null or hir_function.extern_abi.? != .c) {
                        try self.report(.InvalidMirOperand, mir_function.source_span, diagnostics.invalidMirOperand);
                    }
                    if (hir_function.c_symbol_name == null or hir_function.c_symbol_name.?.index != export_c.symbol.index or self.semantic_module.interner.text(export_c.symbol).len == 0) {
                        try self.report(.InvalidMirOperand, mir_function.source_span, diagnostics.invalidMirOperand);
                    }
                },
            }
        }
        try self.requireValidType(mir_function.return_type, mir_function.source_span);
        if (mir_function.blocks.len == 0) {
            try self.report(.InvalidMirBlock, mir_function.source_span, diagnostics.invalidMirBlock);
        }

        for (mir_function.params) |local_id| {
            if (!self.containsLocalInFunction(function_id, local_id)) {
                try self.report(.InvalidMirLocal, mir_function.source_span, diagnostics.invalidMirLocal);
                continue;
            }
            const local_value = self.local(local_id);
            if (local_value.kind != .param) try self.report(.InvalidMirLocal, local_value.source_span, diagnostics.invalidMirLocal);
        }
        for (mir_function.locals) |local_id| {
            if (!self.containsLocalInFunction(function_id, local_id)) {
                try self.report(.InvalidMirLocal, mir_function.source_span, diagnostics.invalidMirLocal);
                continue;
            }
            const local_value = self.local(local_id);
            if (local_value.kind == .param) try self.report(.InvalidMirLocal, local_value.source_span, diagnostics.invalidMirLocal);
        }
        for (mir_function.blocks) |block_id| {
            if (!self.containsBlockInFunction(function_id, block_id)) {
                try self.report(.InvalidMirBlock, mir_function.source_span, diagnostics.invalidMirBlock);
            }
        }
    }

    fn validateLocal(self: *Validator, local_id: mir.MirLocalId, local_value: mir.MirLocal) ValidationError!void {
        if (!self.containsFunction(local_value.parent)) {
            try self.report(.InvalidMirLocal, local_value.source_span, diagnostics.invalidMirLocal);
            return;
        }
        try self.requireValidType(local_value.type_id, local_value.source_span);
        const parent_function = self.function(local_value.parent);
        const in_params = containsId(mir.MirLocalId, parent_function.params, local_id);
        const in_locals = containsId(mir.MirLocalId, parent_function.locals, local_id);
        switch (local_value.kind) {
            .param => if (!in_params or in_locals) try self.report(.InvalidMirLocal, local_value.source_span, diagnostics.invalidMirLocal),
            .user, .temp => if (!in_locals or in_params) try self.report(.InvalidMirLocal, local_value.source_span, diagnostics.invalidMirLocal),
        }
    }

    fn validateBlock(self: *Validator, block_id: mir.MirBlockId, mir_block: mir.MirBlock) ValidationError!void {
        if (!self.containsFunction(mir_block.parent)) {
            try self.report(.InvalidMirBlock, mir_block.source_span, diagnostics.invalidMirBlock);
            return;
        }
        const parent_function = self.function(mir_block.parent);
        if (!containsId(mir.MirBlockId, parent_function.blocks, block_id)) {
            try self.report(.InvalidMirBlock, mir_block.source_span, diagnostics.invalidMirBlock);
        }
        for (mir_block.statements) |statement| {
            try self.validateStatement(mir_block.parent, statement);
        }
        if (mir_block.terminator) |terminator| {
            try self.validateTerminator(mir_block.parent, parent_function, terminator);
        } else {
            try self.report(.MissingTerminator, mir_block.source_span, diagnostics.missingMirTerminator);
        }
    }

    fn validateStatement(self: *Validator, function_id: mir.MirFunctionId, statement: mir.MirStatement) ValidationError!void {
        switch (statement.kind) {
            .assign => |assignment| {
                const place_type = try self.placeType(function_id, assignment.place, statement.span);
                const rvalue_type = try self.rvalueType(function_id, assignment.rvalue, statement.span);
                if (place_type != null and rvalue_type != null and !sameType(place_type.?, rvalue_type.?)) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
            },
            .call => |call| {
                const call_type = try self.validateCall(function_id, call, statement.span);
                if (call_type != null and !sameType(call_type.?, self.semantic_module.types.voidType())) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
            },
            .interface_call => |call| {
                const call_type = try self.rvalueType(function_id, .{ .interface_call = call }, statement.span);
                if (call_type != null and !sameType(call_type.?, self.semantic_module.types.voidType())) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
            },
            .drop => |drop| {
                const place_type = try self.placeType(function_id, drop.place, statement.span);
                if (!self.containsHirFunction(drop.function)) {
                    try self.report(.InvalidMirOperand, statement.span, diagnostics.invalidMirOperand);
                    return;
                }
                const drop_function = self.semantic_module.hir.getFunction(drop.function);
                if (!sameType(drop_function.return_type, self.semantic_module.types.voidType())) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
                if (drop_function.params.len != 1) {
                    try self.report(.InvalidMirOperand, statement.span, diagnostics.invalidMirOperand);
                    return;
                }
                const param_type = self.semantic_module.hir.getParam(drop_function.params[0]).type_id;
                if (place_type != null and !sameType(place_type.?, param_type)) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
            },
            .arena_reset => |arena_operand| try self.validateArenaStorageOperand(function_id, arena_operand, statement.span),
            .arena_destroy => |arena_operand| try self.validateArenaStorageOperand(function_id, arena_operand, statement.span),
            .machine_step => |machine_operand| {
                const machine_type = try self.operandType(function_id, machine_operand, statement.span);
                if (machine_type == null or self.semantic_module.types.kind(machine_type.?) != .machine_type) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
            },
            .panic => |panic_stmt| {
                if (panic_stmt.reason.len == 0) try self.report(.InvalidMirOperand, statement.span, diagnostics.invalidMirOperand);
            },
            .assert_stmt => |assert_stmt| {
                const condition_type = try self.operandType(function_id, assert_stmt.condition, statement.span);
                if (condition_type != null and !sameType(condition_type.?, self.semantic_module.types.boolType())) {
                    try self.report(.InvalidMirType, statement.span, diagnostics.invalidMirType);
                }
                if (assert_stmt.reason.len == 0) try self.report(.InvalidMirOperand, statement.span, diagnostics.invalidMirOperand);
            },
        }
    }

    fn validateArenaStorageOperand(self: *Validator, function_id: mir.MirFunctionId, arena_operand: mir.MirOperand, span: ?diagnostics.SourceSpan) ValidationError!void {
        const arena_operand_type = try self.operandType(function_id, arena_operand, span);
        const expected_arena_pointer = self.semantic_module.types.pointerType(self.semantic_module.types.arenaType()) orelse {
            try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
            return;
        };
        if (arena_operand_type != null and !sameType(arena_operand_type.?, expected_arena_pointer)) {
            try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
        }
    }

    fn validateTerminator(self: *Validator, function_id: mir.MirFunctionId, mir_function: *const mir.MirFunction, terminator: mir.MirTerminator) ValidationError!void {
        switch (terminator.kind) {
            .goto => |target| try self.requireBlockInFunction(function_id, target, terminator.span),
            .return_ => |maybe_operand| {
                if (sameType(mir_function.return_type, self.semantic_module.types.voidType())) {
                    if (maybe_operand) |_| try self.report(.InvalidMirType, terminator.span, diagnostics.invalidMirType);
                    return;
                }
                const operand = maybe_operand orelse {
                    try self.report(.InvalidMirType, terminator.span, diagnostics.invalidMirType);
                    return;
                };
                const operand_type = try self.operandType(function_id, operand, terminator.span);
                if (operand_type != null and !sameType(operand_type.?, mir_function.return_type)) {
                    try self.report(.InvalidMirType, terminator.span, diagnostics.invalidMirType);
                }
            },
            .switch_bool => |switch_bool| {
                const discriminant_type = try self.operandType(function_id, switch_bool.discriminant, terminator.span);
                if (discriminant_type != null and !sameType(discriminant_type.?, self.semantic_module.types.boolType())) {
                    try self.report(.InvalidMirType, terminator.span, diagnostics.invalidMirType);
                }
                try self.requireBlockInFunction(function_id, switch_bool.true_target, terminator.span);
                try self.requireBlockInFunction(function_id, switch_bool.false_target, terminator.span);
            },
            .switch_int => |switch_int| {
                const discriminant_type = try self.operandType(function_id, switch_int.discriminant, terminator.span);
                if (discriminant_type != null and !sameType(discriminant_type.?, self.semantic_module.types.intType())) {
                    try self.report(.InvalidMirType, terminator.span, diagnostics.invalidMirType);
                }
                var seen_cases = std.StringHashMap(void).init(self.allocator);
                defer seen_cases.deinit();
                for (switch_int.cases) |case| {
                    if (seen_cases.contains(case.value)) {
                        try self.report(.InvalidMirOperand, terminator.span, diagnostics.invalidMirOperand);
                    } else {
                        try seen_cases.put(case.value, {});
                    }
                    try self.requireBlockInFunction(function_id, case.target, terminator.span);
                }
                try self.requireBlockInFunction(function_id, switch_int.default_target, terminator.span);
            },
            .@"unreachable" => {},
        }
    }

    fn placeType(self: *Validator, function_id: mir.MirFunctionId, place: mir.MirPlace, span: ?diagnostics.SourceSpan) ValidationError!?types.TypeId {
        return switch (place) {
            .local => |local_id| blk: {
                if (!self.containsLocalInFunction(function_id, local_id)) {
                    try self.report(.InvalidMirLocal, span, diagnostics.invalidMirLocal);
                    break :blk null;
                }
                const local_value = self.local(local_id);
                try self.requireValidType(local_value.type_id, span orelse local_value.source_span);
                break :blk local_value.type_id;
            },
            .field => |field_place| blk: {
                const base_type = try self.placeType(function_id, mir.MirPlace.localPlace(field_place.base), span);
                if (field_place.field_id.index >= self.semantic_module.hir.fields.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const field = self.semantic_module.hir.getField(field_place.field_id);
                if (base_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    if (kind != .struct_type or kind.struct_type.index != field.parent.index) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk field.type_id;
            },
            .index => |index_place| blk: {
                const base_type = try self.placeType(function_id, index_place.base.*, span);
                const index_type = try self.operandType(function_id, index_place.index.*, span);
                if (index_type) |type_id| if (self.semantic_module.types.kind(type_id) != .int) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                };
                if (base_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    if (kind != .array) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    } else if (kind.array.element.index != index_place.result_type.index or kind.array.length != index_place.length) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk index_place.result_type;
            },
        };
    }

    fn operandType(self: *Validator, function_id: mir.MirFunctionId, operand: mir.MirOperand, span: ?diagnostics.SourceSpan) ValidationError!?types.TypeId {
        return switch (operand) {
            .int_literal => self.semantic_module.types.intType(),
            .bool_literal => self.semantic_module.types.boolType(),
            .copy => |place| try self.placeType(function_id, place, span),
        };
    }

    fn rvalueType(self: *Validator, function_id: mir.MirFunctionId, rvalue: mir.MirRvalue, span: ?diagnostics.SourceSpan) ValidationError!?types.TypeId {
        return switch (rvalue) {
            .use => |operand| try self.operandType(function_id, operand, span),
            .move => |place| try self.placeType(function_id, place, span),
            .array_index => |array_index| blk: {
                const base_type = try self.operandType(function_id, array_index.base, span);
                const index_type = try self.operandType(function_id, array_index.index, span);
                if (index_type) |type_id| if (self.semantic_module.types.kind(type_id) != .int) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                };
                if (base_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    if (kind != .array) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    } else if (kind.array.element.index != array_index.result_type.index or kind.array.length != array_index.length) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk array_index.result_type;
            },
            .slice_from_array => |slice| blk: {
                const array_type = try self.operandType(function_id, slice.array, span);
                if (array_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    const slice_kind = self.semantic_module.types.kind(slice.result_type);
                    if (kind != .array or slice_kind != .slice or kind.array.length != slice.length or kind.array.element.index != slice_kind.slice.element.index) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk slice.result_type;
            },
            .slice_index => |slice_index| blk: {
                const base_type = try self.operandType(function_id, slice_index.base, span);
                const index_type = try self.operandType(function_id, slice_index.index, span);
                if (index_type) |type_id| if (self.semantic_module.types.kind(type_id) != .int) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                if (base_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    if (kind != .slice or kind.slice.element.index != slice_index.result_type.index) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                break :blk slice_index.result_type;
            },
            .slice_len => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type) |type_id| if (self.semantic_module.types.kind(type_id) != .slice) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                break :blk self.semantic_module.types.intType();
            },
            .manual_init_assume => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type == null) break :blk null;
                break :blk self.semantic_module.types.manualInitPayload(operand_type.?) orelse {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    break :blk null;
                };
            },
            .unary => |unary| blk: {
                const operand_type = try self.operandType(function_id, unary.operand, span);
                if (operand_type == null) break :blk null;
                switch (unary.op) {
                    .negate => {
                        if (!sameType(operand_type.?, self.semantic_module.types.intType())) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk self.semantic_module.types.intType();
                    },
                    .logical_not => {
                        if (!sameType(operand_type.?, self.semantic_module.types.boolType())) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk self.semantic_module.types.boolType();
                    },
                }
            },
            .address_of => |place| blk: {
                const place_type = try self.placeType(function_id, place, span);
                if (place_type == null) break :blk null;
                break :blk self.semantic_module.types.pointerType(place_type.?) orelse {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    break :blk null;
                };
            },
            .deref => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type == null) break :blk null;
                switch (self.semantic_module.types.kind(operand_type.?)) {
                    .pointer => |pointer| break :blk pointer.pointee,
                    else => {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk null;
                    },
                }
            },
            .dyn_coerce => |coerce| blk: {
                try self.requireValidType(coerce.result_type, span);
                const dyn = switch (self.semantic_module.types.kind(coerce.result_type)) {
                    .dyn_interface => |dyn| dyn,
                    else => {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk null;
                    },
                };
                if (dyn.interface_id.index != coerce.interface_id.index) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                const source_type = try self.placeType(function_id, coerce.source, span);
                if (coerce.impl_id.index >= self.semantic_module.hir.interface_impls.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk coerce.result_type;
                }
                const interface_impl = self.semantic_module.hir.getInterfaceImpl(coerce.impl_id);
                if (interface_impl.interface_id.index != coerce.interface_id.index) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
                if (source_type != null and !sameType(source_type.?, interface_impl.target_type)) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                break :blk coerce.result_type;
            },
            .binary => |binary| blk: {
                const left_type = try self.operandType(function_id, binary.left, span);
                const right_type = try self.operandType(function_id, binary.right, span);
                if (left_type == null or right_type == null) break :blk null;
                switch (binary.op) {
                    .multiply, .divide, .modulo, .add, .subtract => {
                        if (!self.isIntPair(left_type.?, right_type.?)) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk self.semantic_module.types.intType();
                    },
                    .less, .less_equal, .greater, .greater_equal => {
                        if (!self.isIntPair(left_type.?, right_type.?)) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk self.semantic_module.types.boolType();
                    },
                    .equal_equal, .bang_equal => {
                        if (!sameType(left_type.?, right_type.?) or (!sameType(left_type.?, self.semantic_module.types.intType()) and !sameType(left_type.?, self.semantic_module.types.boolType()))) {
                            try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        }
                        break :blk self.semantic_module.types.boolType();
                    },
                    .logical_and, .logical_or => {
                        if (!self.isBoolPair(left_type.?, right_type.?)) try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk self.semantic_module.types.boolType();
                    },
                }
            },
            .call => |call| blk: {
                break :blk try self.validateCall(function_id, call, span);
            },
            .interface_call => |call| blk: {
                try self.requireValidType(call.result_type, span);
                const receiver_type = try self.operandType(function_id, call.receiver, span);
                if (receiver_type == null) break :blk call.result_type;
                const dyn = switch (self.semantic_module.types.kind(receiver_type.?)) {
                    .dyn_interface => |dyn| dyn,
                    else => {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk call.result_type;
                    },
                };
                if (dyn.interface_id.index != call.interface_id.index) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                if (call.interface_id.index >= self.semantic_module.hir.interfaces.items.len or call.requirement_id.index >= self.semantic_module.hir.interface_requirements.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk call.result_type;
                }
                const interface_decl = self.semantic_module.hir.getInterface(call.interface_id);
                if (call.requirement_index >= interface_decl.requirements.len or interface_decl.requirements[call.requirement_index].index != call.requirement_id.index) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk call.result_type;
                }
                const requirement = self.semantic_module.hir.getInterfaceRequirement(call.requirement_id);
                if (requirement.parent.index != call.interface_id.index) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
                if (call.args.len != requirement.params.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk call.result_type;
                }
                for (call.args, requirement.params) |arg, param_id| {
                    const arg_type = try self.operandType(function_id, arg, span);
                    const param_type = self.semantic_module.hir.getInterfaceParam(param_id).type_id;
                    try self.requireValidType(param_type, span);
                    if (arg_type != null and !sameType(arg_type.?, param_type)) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                if (!sameType(call.result_type, requirement.return_type)) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                break :blk call.result_type;
            },
            .arena_alloc => |arena_alloc| blk: {
                try self.requireValidType(arena_alloc.allocated_type, span);
                try self.requireValidType(arena_alloc.result_type, span);
                const arena_operand_type = try self.operandType(function_id, arena_alloc.arena_operand, span);
                const expected_arena_pointer = self.semantic_module.types.pointerType(self.semantic_module.types.arenaType()) orelse {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    break :blk null;
                };
                if (arena_operand_type != null and !sameType(arena_operand_type.?, expected_arena_pointer)) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                const expected_result = self.semantic_module.types.pointerType(arena_alloc.allocated_type) orelse {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    break :blk null;
                };
                if (!sameType(arena_alloc.result_type, expected_result)) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                break :blk arena_alloc.result_type;
            },
            .machine_construct => |construct| blk: {
                if (construct.machine.index >= self.semantic_module.hir.machines.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const machine = self.semantic_module.hir.getMachine(construct.machine);
                if (construct.args.len != machine.params.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
                for (construct.args, machine.params) |arg, param_id| {
                    const arg_type = try self.operandType(function_id, arg, span);
                    const param_type = self.semantic_module.hir.getMachineParam(param_id).type_id;
                    if (arg_type != null and !sameType(arg_type.?, param_type)) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk self.semantic_module.types.machineType(construct.machine) orelse {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    break :blk null;
                };
            },
            .machine_complete => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type == null or self.semantic_module.types.kind(operand_type.?) != .machine_type) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                break :blk self.semantic_module.types.boolType();
            },
            .machine_state => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type == null or self.semantic_module.types.kind(operand_type.?) != .machine_type) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
                break :blk self.semantic_module.types.intType();
            },
            .machine_result => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type == null) break :blk null;
                break :blk switch (self.semantic_module.types.kind(operand_type.?)) {
                    .machine_type => |machine_id| self.semantic_module.hir.getMachine(machine_id).return_type,
                    else => {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                        break :blk null;
                    },
                };
            },
            .struct_constructor => |constructor| blk: {
                if (constructor.struct_id.index >= self.semantic_module.hir.structs.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const struct_decl = self.semantic_module.hir.getStruct(constructor.struct_id);
                var seen = std.AutoHashMap(hir.FieldId, void).init(self.allocator);
                defer seen.deinit();
                for (constructor.fields) |field_value| {
                    if (field_value.field_id.index >= self.semantic_module.hir.fields.items.len) {
                        try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                        continue;
                    }
                    const field = self.semantic_module.hir.getField(field_value.field_id);
                    if (field.parent.index != constructor.struct_id.index or seen.contains(field_value.field_id)) {
                        try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                        continue;
                    }
                    try seen.put(field_value.field_id, {});
                    const value_type = try self.operandType(function_id, field_value.value, span);
                    if (value_type != null and !sameType(value_type.?, field.type_id)) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                for (struct_decl.fields) |field_id| {
                    if (!seen.contains(field_id)) try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
                break :blk self.structType(constructor.struct_id);
            },
            .array_constructor => |elements| blk: {
                if (elements.len == 0) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const element_type = (try self.operandType(function_id, elements[0], span)) orelse break :blk null;
                for (elements[1..]) |element| {
                    const actual = try self.operandType(function_id, element, span);
                    if (actual != null and !sameType(actual.?, element_type)) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk self.arrayType(element_type, @intCast(elements.len)) orelse {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    break :blk null;
                };
            },
            .enum_constructor => |constructor| blk: {
                if (constructor.enum_id.index >= self.semantic_module.hir.enums.items.len or constructor.variant_id.index >= self.semantic_module.hir.variants.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const variant = self.semantic_module.hir.getVariant(constructor.variant_id);
                if (variant.parent.index != constructor.enum_id.index or constructor.args.len != variant.payload_fields.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk self.enumType(constructor.enum_id);
                }
                for (constructor.args, variant.payload_fields) |arg, payload_id| {
                    const arg_type = try self.operandType(function_id, arg, span);
                    const payload_type = self.semantic_module.hir.getEnumPayloadField(payload_id).type_id;
                    if (arg_type != null and !sameType(arg_type.?, payload_type)) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk self.enumType(constructor.enum_id);
            },
            .enum_tag => |operand| blk: {
                const operand_type = try self.operandType(function_id, operand, span);
                if (operand_type) |type_id| {
                    if (self.semantic_module.types.kind(type_id) != .enum_type) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk self.semantic_module.types.intType();
            },
            .field_access => |field_access| blk: {
                if (field_access.field_id.index >= self.semantic_module.hir.fields.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const field = self.semantic_module.hir.getField(field_access.field_id);
                const receiver_type = try self.operandType(function_id, field_access.receiver, span);
                if (receiver_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    if (kind != .struct_type or kind.struct_type.index != field.parent.index) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk field.type_id;
            },
            .enum_payload_field => |payload| blk: {
                if (payload.payload_field.index >= self.semantic_module.hir.enum_payload_fields.items.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const field = self.semantic_module.hir.getEnumPayloadField(payload.payload_field);
                const variant = self.semantic_module.hir.getVariant(field.parent);
                const operand_type = try self.operandType(function_id, payload.enum_operand, span);
                if (operand_type) |type_id| {
                    const kind = self.semantic_module.types.kind(type_id);
                    if (kind != .enum_type or kind.enum_type.index != variant.parent.index) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk field.type_id;
            },
        };
    }

    fn structType(self: *Validator, struct_id: hir.StructId) ?types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .struct_type and kind.struct_type.index == struct_id.index) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn enumType(self: *Validator, enum_id: hir.EnumId) ?types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .enum_type and kind.enum_type.index == enum_id.index) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn arrayType(self: *Validator, element: types.TypeId, length: u64) ?types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .array and kind.array.element.index == element.index and kind.array.length == length) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn validateCall(self: *Validator, function_id: mir.MirFunctionId, call: mir.MirCall, span: ?diagnostics.SourceSpan) ValidationError!?types.TypeId {
        const callee_function_id = call.callee.functionId();
        if (!self.containsHirFunction(callee_function_id)) {
            try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
            return null;
        }

        const callee = self.semantic_module.hir.getFunction(callee_function_id);
        switch (call.callee) {
            .internal => {
                if (callee.is_extern) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
            },
            .extern_c => |extern_c| {
                if (!callee.is_extern or callee.extern_abi == null or callee.extern_abi.? != .c) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
                if (callee.c_symbol_name == null or callee.c_symbol_name.?.index != extern_c.symbol.index or self.semantic_module.interner.text(extern_c.symbol).len == 0) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                }
                if (!sameType(callee.return_type, extern_c.result_type)) {
                    try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                }
            },
        }

        try self.requireValidType(callee.return_type, span);
        if (call.args.len != callee.params.len) {
            try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
            return callee.return_type;
        }
        for (call.args, callee.params) |arg, param_id| {
            const arg_type = try self.operandType(function_id, arg, span);
            const param_type = self.semantic_module.hir.getParam(param_id).type_id;
            try self.requireValidType(param_type, span);
            if (arg_type != null and !sameType(arg_type.?, param_type)) {
                try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
            }
        }
        return callee.return_type;
    }

    fn requireValidType(self: *Validator, type_id: types.TypeId, span: ?diagnostics.SourceSpan) ValidationError!void {
        if (!self.semantic_module.types.contains(type_id)) {
            try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
            return;
        }
        if (self.containsTypeParam(type_id)) {
            try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
        }
    }

    fn containsTypeParam(self: *Validator, type_id: types.TypeId) bool {
        return switch (self.semantic_module.types.kind(type_id)) {
            .type_param => true,
            .pointer => |pointer| self.containsTypeParam(pointer.pointee),
            else => false,
        };
    }

    fn requireBlockInFunction(self: *Validator, function_id: mir.MirFunctionId, block_id: mir.MirBlockId, span: ?diagnostics.SourceSpan) ValidationError!void {
        if (!self.containsBlockInFunction(function_id, block_id)) {
            try self.report(.InvalidMirBlock, span, diagnostics.invalidMirBlock);
        }
    }

    fn isIntPair(self: *Validator, left: types.TypeId, right: types.TypeId) bool {
        return sameType(left, self.semantic_module.types.intType()) and sameType(right, self.semantic_module.types.intType());
    }

    fn isBoolPair(self: *Validator, left: types.TypeId, right: types.TypeId) bool {
        return sameType(left, self.semantic_module.types.boolType()) and sameType(right, self.semantic_module.types.boolType());
    }

    fn containsHirFunction(self: *Validator, function_id: hir.FunctionId) bool {
        return function_id.index < self.semantic_module.hir.functions.items.len;
    }

    fn containsFunction(self: *Validator, function_id: mir.MirFunctionId) bool {
        return function_id.index < self.mir_module.store.functions.items.len;
    }

    fn containsLocal(self: *Validator, local_id: mir.MirLocalId) bool {
        return local_id.index < self.mir_module.store.locals.items.len;
    }

    fn containsBlock(self: *Validator, block_id: mir.MirBlockId) bool {
        return block_id.index < self.mir_module.store.blocks.items.len;
    }

    fn containsLocalInFunction(self: *Validator, function_id: mir.MirFunctionId, local_id: mir.MirLocalId) bool {
        if (!self.containsFunction(function_id) or !self.containsLocal(local_id)) return false;
        return sameMirFunction(self.local(local_id).parent, function_id);
    }

    fn containsBlockInFunction(self: *Validator, function_id: mir.MirFunctionId, block_id: mir.MirBlockId) bool {
        if (!self.containsFunction(function_id) or !self.containsBlock(block_id)) return false;
        return sameMirFunction(self.block(block_id).parent, function_id);
    }

    fn function(self: *Validator, function_id: mir.MirFunctionId) *const mir.MirFunction {
        return &self.mir_module.store.functions.items[function_id.index];
    }

    fn local(self: *Validator, local_id: mir.MirLocalId) *const mir.MirLocal {
        return &self.mir_module.store.locals.items[local_id.index];
    }

    fn block(self: *Validator, block_id: mir.MirBlockId) *const mir.MirBlock {
        return &self.mir_module.store.blocks.items[block_id.index];
    }

    fn report(
        self: *Validator,
        comptime code: diagnostics.DiagnosticCode,
        maybe_span: ?diagnostics.SourceSpan,
        comptime make: fn (diagnostics.SourceSpan) diagnostics.Diagnostic,
    ) ValidationError!void {
        _ = code;
        self.had_error = true;
        if (self.diagnostics) |bag| {
            try bag.append(make(maybe_span orelse synthetic_span));
        }
    }
};

fn containsId(comptime Id: type, ids: []const Id, needle: Id) bool {
    for (ids) |id| {
        if (id.index == needle.index) return true;
    }
    return false;
}

fn sameType(left: types.TypeId, right: types.TypeId) bool {
    return left.index == right.index;
}

fn sameMirFunction(left: mir.MirFunctionId, right: mir.MirFunctionId) bool {
    return left.index == right.index;
}

const TestContext = struct {
    module: semantics.SemanticModule,

    fn init() !TestContext {
        return .{ .module = try semantics.SemanticModule.init(std.testing.allocator) };
    }

    fn deinit(self: *TestContext) void {
        self.module.deinit();
    }

    fn function(self: *TestContext, name: []const u8, return_type: types.TypeId, has_body: bool) !hir.FunctionId {
        const name_id = try self.module.interner.intern(name);
        const function_id = try self.module.hir.addFunction(name_id, return_type, synthetic_span);
        if (has_body) self.module.hir.setFunctionBody(function_id, try self.module.hir.addStmt(.{ .block = &.{} }, synthetic_span));
        return function_id;
    }

    fn externFunction(self: *TestContext, name: []const u8, return_type: types.TypeId) !hir.FunctionId {
        const name_id = try self.module.interner.intern(name);
        return self.module.hir.addExternFunction(name_id, return_type, .c, name_id, synthetic_span, synthetic_span);
    }

    fn param(self: *TestContext, function_id: hir.FunctionId, name: []const u8, type_id: types.TypeId) !hir.ParamId {
        return self.module.hir.addParam(function_id, try self.module.interner.intern(name), type_id, synthetic_span);
    }

    fn validateOk(self: *TestContext, mir_module: *const mir.MirModule) !void {
        var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try validateModule(std.testing.allocator, &self.module, mir_module, &bag);
        try std.testing.expectEqual(@as(usize, 0), bag.count());
    }

    fn validateFail(self: *TestContext, mir_module: *const mir.MirModule, code: diagnostics.DiagnosticCode) !void {
        var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try std.testing.expectError(error.InvalidMirModule, validateModule(std.testing.allocator, &self.module, mir_module, &bag));
        try std.testing.expect(hasDiagnostic(&bag, code));
    }
};

fn hasDiagnostic(bag: *const diagnostics.DiagnosticBag, code: diagnostics.DiagnosticCode) bool {
    for (bag.diagnostics.items) |diagnostic| {
        if (diagnostic.code == code) return true;
    }
    return false;
}

fn validMirFunction(ctx: *TestContext, return_type: types.TypeId) !struct { module: mir.MirModule, function: mir.MirFunctionId, block: mir.MirBlockId } {
    const hir_function = try ctx.function("main", return_type, true);
    var mir_module = mir.MirModule.init(std.testing.allocator);
    errdefer mir_module.deinit();
    const function_id = try mir_module.store.addFunction(hir_function, ctx.module.hir.getFunction(hir_function).name, return_type, synthetic_span);
    const block_id = try mir_module.store.addBlock(function_id, synthetic_span);
    return .{ .module = mir_module, .function = function_id, .block = block_id };
}

test "MIR validator accepts valid return literal MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "1")) });
    try ctx.validateOk(&built.module);
}

test "MIR validator accepts valid local assignment MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    const mir_local_id = try built.module.store.addLocal(built.function, try ctx.module.interner.intern("x"), .user, ctx.module.types.intType(), synthetic_span);
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = mir_local_id }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = mir_local_id })) });
    try ctx.validateOk(&built.module);
}

test "MIR validator accepts address-of and deref MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    const int_ptr = try ctx.module.types.addPointerType(ctx.module.types.intType());
    const x = try built.module.store.addLocal(built.function, try ctx.module.interner.intern("x"), .user, ctx.module.types.intType(), synthetic_span);
    const p = try built.module.store.addLocal(built.function, try ctx.module.interner.intern("p"), .user, int_ptr, synthetic_span);
    const temp = try built.module.store.addLocal(built.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = p }, mir.MirRvalue.addressOf(.{ .local = x })) });
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, mir.MirRvalue.dereference(mir.MirOperand.copyPlace(.{ .local = p }))) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp })) });
    try ctx.validateOk(&built.module);
}

test "MIR validator accepts address-of field place MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    const vec_id = try ctx.module.hir.addStruct(try ctx.module.interner.intern("Vec2"));
    const vec_type = try ctx.module.types.addStructType(vec_id);
    const x_field = try ctx.module.hir.addField(vec_id, try ctx.module.interner.intern("x"), ctx.module.types.intType(), synthetic_span);
    _ = try ctx.module.hir.addField(vec_id, try ctx.module.interner.intern("y"), ctx.module.types.intType(), synthetic_span);
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    const int_ptr = try ctx.module.types.addPointerType(ctx.module.types.intType());
    const v = try built.module.store.addLocal(built.function, try ctx.module.interner.intern("v"), .user, vec_type, synthetic_span);
    const p = try built.module.store.addLocal(built.function, try ctx.module.interner.intern("p"), .user, int_ptr, synthetic_span);
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = p }, mir.MirRvalue.addressOf(mir.MirPlace.fieldPlace(v, x_field))) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateOk(&built.module);
}

test "MIR validator rejects deref non-pointer MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    const x = try built.module.store.addLocal(built.function, try ctx.module.interner.intern("x"), .user, ctx.module.types.intType(), synthetic_span);
    const temp = try built.module.store.addLocal(built.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, mir.MirRvalue.dereference(mir.MirOperand.copyPlace(.{ .local = x }))) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp })) });
    try ctx.validateFail(&built.module, .InvalidMirType);
}

test "MIR validator accepts valid unary and binary MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    const temp_int = try built.module.store.addLocal(built.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const temp_bool = try built.module.store.addLocal(built.function, null, .temp, ctx.module.types.boolType(), synthetic_span);
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp_int }, mir.MirRvalue.unaryOp(.negate, try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp_bool }, mir.MirRvalue.binaryOp(.less, mir.MirOperand.copyPlace(.{ .local = temp_int }), try mir.MirOperand.intLiteral(std.testing.allocator, "2"))) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp_int })) });
    try ctx.validateOk(&built.module);
}

test "MIR validator accepts valid call MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    const callee = try ctx.function("id", ctx.module.types.intType(), true);
    _ = try ctx.param(callee, "x", ctx.module.types.intType());
    var built = try validMirFunction(&ctx, ctx.module.types.intType());
    defer built.module.deinit();
    const temp = try built.module.store.addLocal(built.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    args[0] = try mir.MirOperand.intLiteral(std.testing.allocator, "1");
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, .{ .call = .{ .callee = .{ .internal = callee }, .args = args } }) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp })) });
    try ctx.validateOk(&built.module);
}

test "MIR validator validates extern C call linkage" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    const extern_abs = try ctx.externFunction("abs", ctx.module.types.intType());
    _ = try ctx.param(extern_abs, "value", ctx.module.types.intType());

    var valid = try validMirFunction(&ctx, ctx.module.types.intType());
    defer valid.module.deinit();
    const valid_temp = try valid.module.store.addLocal(valid.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const valid_args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    valid_args[0] = try mir.MirOperand.intLiteral(std.testing.allocator, "1");
    const symbol = ctx.module.hir.getFunction(extern_abs).c_symbol_name.?;
    try valid.module.store.appendStatement(valid.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = valid_temp }, .{ .call = .{ .callee = .{ .extern_c = .{ .function = extern_abs, .symbol = symbol, .result_type = ctx.module.types.intType() } }, .args = valid_args } }) });
    try valid.module.store.setTerminator(valid.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = valid_temp })) });
    try ctx.validateOk(&valid.module);

    var internal_to_extern = try validMirFunction(&ctx, ctx.module.types.intType());
    defer internal_to_extern.module.deinit();
    const internal_temp = try internal_to_extern.module.store.addLocal(internal_to_extern.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const internal_args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    internal_args[0] = try mir.MirOperand.intLiteral(std.testing.allocator, "1");
    try internal_to_extern.module.store.appendStatement(internal_to_extern.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = internal_temp }, .{ .call = .{ .callee = .{ .internal = extern_abs }, .args = internal_args } }) });
    try internal_to_extern.module.store.setTerminator(internal_to_extern.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = internal_temp })) });
    try ctx.validateFail(&internal_to_extern.module, .InvalidMirOperand);

    const ordinary = try ctx.function("ordinary", ctx.module.types.intType(), true);
    _ = try ctx.param(ordinary, "value", ctx.module.types.intType());
    var extern_to_internal = try validMirFunction(&ctx, ctx.module.types.intType());
    defer extern_to_internal.module.deinit();
    const extern_temp = try extern_to_internal.module.store.addLocal(extern_to_internal.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const extern_args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    extern_args[0] = try mir.MirOperand.intLiteral(std.testing.allocator, "1");
    try extern_to_internal.module.store.appendStatement(extern_to_internal.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = extern_temp }, .{ .call = .{ .callee = .{ .extern_c = .{ .function = ordinary, .symbol = symbol, .result_type = ctx.module.types.intType() } }, .args = extern_args } }) });
    try extern_to_internal.module.store.setTerminator(extern_to_internal.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = extern_temp })) });
    try ctx.validateFail(&extern_to_internal.module, .InvalidMirOperand);
}

test "MIR validator checks arena reset and destroy operands" {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    const arena_ptr = try ctx.module.types.addPointerType(ctx.module.types.arenaType());

    var reset_ok = try validMirFunction(&ctx, ctx.module.types.intType());
    defer reset_ok.module.deinit();
    const reset_arena = try reset_ok.module.store.addLocal(reset_ok.function, try ctx.module.interner.intern("reset_arena"), .user, arena_ptr, synthetic_span);
    try reset_ok.module.store.appendStatement(reset_ok.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.arenaReset(mir.MirOperand.copyPlace(.{ .local = reset_arena })) });
    try reset_ok.module.store.setTerminator(reset_ok.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateOk(&reset_ok.module);

    var destroy_ok = try validMirFunction(&ctx, ctx.module.types.intType());
    defer destroy_ok.module.deinit();
    const destroy_arena = try destroy_ok.module.store.addLocal(destroy_ok.function, try ctx.module.interner.intern("destroy_arena"), .user, arena_ptr, synthetic_span);
    try destroy_ok.module.store.appendStatement(destroy_ok.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.arenaDestroy(mir.MirOperand.copyPlace(.{ .local = destroy_arena })) });
    try destroy_ok.module.store.setTerminator(destroy_ok.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateOk(&destroy_ok.module);

    var reset_bad = try validMirFunction(&ctx, ctx.module.types.intType());
    defer reset_bad.module.deinit();
    const not_arena = try reset_bad.module.store.addLocal(reset_bad.function, try ctx.module.interner.intern("not_arena"), .user, ctx.module.types.intType(), synthetic_span);
    try reset_bad.module.store.appendStatement(reset_bad.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.arenaReset(mir.MirOperand.copyPlace(.{ .local = not_arena })) });
    try reset_bad.module.store.setTerminator(reset_bad.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateFail(&reset_bad.module, .InvalidMirType);
}

test "MIR validator rejects manually malformed MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();

    var missing = try validMirFunction(&ctx, ctx.module.types.intType());
    defer missing.module.deinit();
    try ctx.validateFail(&missing.module, .MissingTerminator);

    var bad_target = try validMirFunction(&ctx, ctx.module.types.intType());
    defer bad_target.module.deinit();
    try bad_target.module.store.setTerminator(bad_target.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(.{ .index = 999 }) });
    try ctx.validateFail(&bad_target.module, .InvalidMirBlock);

    var bad_local = try validMirFunction(&ctx, ctx.module.types.intType());
    defer bad_local.module.deinit();
    try bad_local.module.store.setTerminator(bad_local.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = .{ .index = 999 } })) });
    try ctx.validateFail(&bad_local.module, .InvalidMirLocal);

    var assign_mismatch = try validMirFunction(&ctx, ctx.module.types.intType());
    defer assign_mismatch.module.deinit();
    const int_local = try assign_mismatch.module.store.addLocal(assign_mismatch.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    try assign_mismatch.module.store.appendStatement(assign_mismatch.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = int_local }, mir.MirRvalue.use_(mir.MirOperand.boolLiteral(true))) });
    try assign_mismatch.module.store.setTerminator(assign_mismatch.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateFail(&assign_mismatch.module, .InvalidMirType);

    var return_mismatch = try validMirFunction(&ctx, ctx.module.types.intType());
    defer return_mismatch.module.deinit();
    try return_mismatch.module.store.setTerminator(return_mismatch.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.boolLiteral(true)) });
    try ctx.validateFail(&return_mismatch.module, .InvalidMirType);
}

test "MIR validator rejects type params in executable MIR" {
    var ctx = try TestContext.init();
    defer ctx.deinit();

    const type_param = try ctx.module.types.addTypeParam(.{ .kind = .generic_function, .index = 0 }, 0, try ctx.module.interner.intern("T"));
    const type_param_ptr = try ctx.module.types.addPointerType(type_param);

    var return_type_param = try validMirFunction(&ctx, type_param);
    defer return_type_param.module.deinit();
    try return_type_param.module.store.setTerminator(return_type_param.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(null) });
    try ctx.validateFail(&return_type_param.module, .InvalidMirType);

    var local_type_param = try validMirFunction(&ctx, ctx.module.types.intType());
    defer local_type_param.module.deinit();
    _ = try local_type_param.module.store.addLocal(local_type_param.function, try ctx.module.interner.intern("value"), .user, type_param, synthetic_span);
    try local_type_param.module.store.setTerminator(local_type_param.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateFail(&local_type_param.module, .InvalidMirType);

    var pointer_to_type_param = try validMirFunction(&ctx, ctx.module.types.intType());
    defer pointer_to_type_param.module.deinit();
    _ = try pointer_to_type_param.module.store.addLocal(pointer_to_type_param.function, try ctx.module.interner.intern("ptr"), .user, type_param_ptr, synthetic_span);
    try pointer_to_type_param.module.store.setTerminator(pointer_to_type_param.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try ctx.validateFail(&pointer_to_type_param.module, .InvalidMirType);

    const callee = try ctx.function("take_type_param", ctx.module.types.intType(), true);
    _ = try ctx.param(callee, "value", type_param);
    var call_type_param = try validMirFunction(&ctx, ctx.module.types.intType());
    defer call_type_param.module.deinit();
    const temp = try call_type_param.module.store.addLocal(call_type_param.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    args[0] = try mir.MirOperand.intLiteral(std.testing.allocator, "1");
    try call_type_param.module.store.appendStatement(call_type_param.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, .{ .call = .{ .callee = .{ .internal = callee }, .args = args } }) });
    try call_type_param.module.store.setTerminator(call_type_param.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp })) });
    try ctx.validateFail(&call_type_param.module, .InvalidMirType);
}

test "MIR validator rejects switch and call mismatches" {
    var ctx = try TestContext.init();
    defer ctx.deinit();

    var bool_switch = try validMirFunction(&ctx, ctx.module.types.intType());
    defer bool_switch.module.deinit();
    const bool_exit = try bool_switch.module.store.addBlock(bool_switch.function, synthetic_span);
    try bool_switch.module.store.setTerminator(bool_exit, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
    try bool_switch.module.store.setTerminator(bool_switch.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.switchBool(try mir.MirOperand.intLiteral(std.testing.allocator, "1"), bool_exit, bool_exit) });
    try ctx.validateFail(&bool_switch.module, .InvalidMirType);

    var int_switch = try validMirFunction(&ctx, ctx.module.types.intType());
    defer int_switch.module.deinit();
    const int_exit = try int_switch.module.store.addBlock(int_switch.function, synthetic_span);
    try int_exitTerm(&int_switch.module, int_exit);
    const cases = try std.testing.allocator.alloc(mir.MirSwitchIntCase, 1);
    cases[0] = .{ .value = try std.testing.allocator.dupe(u8, "1"), .target = int_exit };
    try int_switch.module.store.setTerminator(int_switch.block, .{ .span = synthetic_span, .kind = .{ .switch_int = .{ .discriminant = mir.MirOperand.boolLiteral(true), .cases = cases, .default_target = int_exit } } });
    try ctx.validateFail(&int_switch.module, .InvalidMirType);

    const callee = try ctx.function("take_int", ctx.module.types.intType(), true);
    _ = try ctx.param(callee, "x", ctx.module.types.intType());

    var call_count = try validMirFunction(&ctx, ctx.module.types.intType());
    defer call_count.module.deinit();
    const count_temp = try call_count.module.store.addLocal(call_count.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    try call_count.module.store.appendStatement(call_count.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = count_temp }, .{ .call = .{ .callee = .{ .internal = callee }, .args = &.{} } }) });
    try call_count.module.store.setTerminator(call_count.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = count_temp })) });
    try ctx.validateFail(&call_count.module, .InvalidMirOperand);

    var call_type = try validMirFunction(&ctx, ctx.module.types.intType());
    defer call_type.module.deinit();
    const type_temp = try call_type.module.store.addLocal(call_type.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    args[0] = mir.MirOperand.boolLiteral(false);
    try call_type.module.store.appendStatement(call_type.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = type_temp }, .{ .call = .{ .callee = .{ .internal = callee }, .args = args } }) });
    try call_type.module.store.setTerminator(call_type.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = type_temp })) });
    try ctx.validateFail(&call_type.module, .InvalidMirType);
}

fn int_exitTerm(module: *mir.MirModule, block_id: mir.MirBlockId) !void {
    try module.store.setTerminator(block_id, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });
}

test "MIR validator accepts lowered if while and match MIR" {
    try validateLowered(.if_else);
    try validateLowered(.while_loop);
    try validateLowered(.match_int);
    try validateLowered(.decide);
}

test "MIR validator integration covers checked HIR lowering programs" {
    try validateLowered(.arithmetic);
    try validateLowered(.while_loop);
    try validateLowered(.match_int);
    try validateLowered(.decide);
}

const LoweredCase = enum { arithmetic, if_else, while_loop, match_int, decide };

fn validateLowered(case: LoweredCase) !void {
    var ctx = try TestContext.init();
    defer ctx.deinit();
    const main = try ctx.function("main", ctx.module.types.intType(), false);
    switch (case) {
        .arithmetic => {
            const expr = try binaryExpr(&ctx, .add, try intExpr(&ctx, "1"), try binaryExpr(&ctx, .multiply, try intExpr(&ctx, "2"), try intExpr(&ctx, "3")));
            try setBody(&ctx, main, &.{try returnStmt(&ctx, expr)});
        },
        .if_else => {
            const x = try hirLocal(&ctx, main, "x", ctx.module.types.intType());
            const decl = try localDecl(&ctx, x, try intExpr(&ctx, "0"));
            const then_assign = try assignLocal(&ctx, x, try intExpr(&ctx, "1"));
            const else_assign = try assignLocal(&ctx, x, try intExpr(&ctx, "2"));
            const if_stmt = try addStmt(&ctx, .{ .if_stmt = .{ .condition = try boolExpr(&ctx, true), .then_block = try blockStmt(&ctx, &.{then_assign}), .else_block = try blockStmt(&ctx, &.{else_assign}) } });
            const ret = try returnStmt(&ctx, try localRef(&ctx, x));
            try setBody(&ctx, main, &.{ decl, if_stmt, ret });
        },
        .while_loop => {
            const i = try hirLocal(&ctx, main, "i", ctx.module.types.intType());
            const decl = try localDecl(&ctx, i, try intExpr(&ctx, "0"));
            const condition = try binaryExpr(&ctx, .less, try localRef(&ctx, i), try intExpr(&ctx, "3"));
            const increment = try binaryExpr(&ctx, .add, try localRef(&ctx, i), try intExpr(&ctx, "1"));
            const assign = try assignLocal(&ctx, i, increment);
            const while_stmt = try addStmt(&ctx, .{ .while_stmt = .{ .condition = condition, .body = try blockStmt(&ctx, &.{assign}) } });
            const ret = try returnStmt(&ctx, try localRef(&ctx, i));
            try setBody(&ctx, main, &.{ decl, while_stmt, ret });
        },
        .match_int => {
            const x = try hirLocal(&ctx, main, "x", ctx.module.types.intType());
            const decl = try localDecl(&ctx, x, try intExpr(&ctx, "2"));
            const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 3);
            arms[0] = .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = try blockStmt(&ctx, &.{try returnStmt(&ctx, try intExpr(&ctx, "10"))}) };
            arms[1] = .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "2") }, .pattern_span = synthetic_span, .body = try blockStmt(&ctx, &.{try returnStmt(&ctx, try intExpr(&ctx, "7"))}) };
            arms[2] = .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = try blockStmt(&ctx, &.{try returnStmt(&ctx, try intExpr(&ctx, "0"))}) };
            const match_stmt = try addStmt(&ctx, .{ .match_stmt = .{ .scrutinee = try localRef(&ctx, x), .arms = arms } });
            try setBody(&ctx, main, &.{ decl, match_stmt });
        },
        .decide => {
            const enum_id = try ctx.module.hir.addEnum(try ctx.module.interner.intern("Choice"), false);
            const enum_type = try ctx.module.types.addEnumType(enum_id);
            const a = try ctx.module.hir.addVariant(enum_id, try ctx.module.interner.intern("A"), synthetic_span);
            const b = try ctx.module.hir.addVariant(enum_id, try ctx.module.interner.intern("B"), synthetic_span);
            const selected = try hirLocal(&ctx, main, "selected", enum_type);
            const arms = try std.testing.allocator.alloc(hir.HirDecideArm, 2);
            arms[0] = .{ .variant_id = a, .condition = try boolExpr(&ctx, true), .score = try intExpr(&ctx, "4"), .span = synthetic_span };
            arms[1] = .{ .variant_id = b, .condition = null, .score = try intExpr(&ctx, "0"), .span = synthetic_span };
            const decide = try addExpr(&ctx, .{ .decide = .{ .enum_type = enum_type, .enum_id = enum_id, .arms = arms } });
            const decl = try localDecl(&ctx, selected, decide);
            try setBody(&ctx, main, &.{ decl, try returnStmt(&ctx, try intExpr(&ctx, "0")) });
        },
    }

    var semantic_diagnostics = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();
    try hir_checker.checkExecutable(std.testing.allocator, &ctx.module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    var mir_module = try mir_lowering.lowerModule(std.testing.allocator, &ctx.module);
    defer mir_module.deinit();
    try ctx.validateOk(&mir_module);
}

fn addStmt(ctx: *TestContext, kind: hir.HirStmtKind) !hir.StmtId {
    return ctx.module.hir.addStmt(kind, synthetic_span);
}

fn addExpr(ctx: *TestContext, kind: hir.HirExprKind) !hir.ExprId {
    return ctx.module.hir.addExpr(kind, synthetic_span);
}

fn setBody(ctx: *TestContext, function_id: hir.FunctionId, stmts: []const hir.StmtId) !void {
    ctx.module.hir.setFunctionBody(function_id, try blockStmt(ctx, stmts));
}

fn blockStmt(ctx: *TestContext, stmts: []const hir.StmtId) !hir.StmtId {
    return addStmt(ctx, .{ .block = try std.testing.allocator.dupe(hir.StmtId, stmts) });
}

fn returnStmt(ctx: *TestContext, expr: hir.ExprId) !hir.StmtId {
    return addStmt(ctx, .{ .return_stmt = expr });
}

fn localDecl(ctx: *TestContext, local_id: hir.LocalId, initializer: hir.ExprId) !hir.StmtId {
    return addStmt(ctx, .{ .local_decl = .{ .local = local_id, .initializer = initializer } });
}

fn assignLocal(ctx: *TestContext, local_id: hir.LocalId, value: hir.ExprId) !hir.StmtId {
    return addStmt(ctx, .{ .assignment = .{ .target = .{ .local = local_id }, .value = value } });
}

fn hirLocal(ctx: *TestContext, function_id: hir.FunctionId, name: []const u8, type_id: types.TypeId) !hir.LocalId {
    return ctx.module.hir.addLocal(function_id, try ctx.module.interner.intern(name), type_id, synthetic_span);
}

fn intExpr(ctx: *TestContext, text: []const u8) !hir.ExprId {
    return addExpr(ctx, .{ .int_literal = try std.testing.allocator.dupe(u8, text) });
}

fn boolExpr(ctx: *TestContext, value: bool) !hir.ExprId {
    return addExpr(ctx, .{ .bool_literal = value });
}

fn localRef(ctx: *TestContext, local_id: hir.LocalId) !hir.ExprId {
    return addExpr(ctx, .{ .local_ref = local_id });
}

fn binaryExpr(ctx: *TestContext, op: hir.BinaryOp, left: hir.ExprId, right: hir.ExprId) !hir.ExprId {
    return addExpr(ctx, .{ .binary = .{ .op = op, .left = left, .right = right } });
}
