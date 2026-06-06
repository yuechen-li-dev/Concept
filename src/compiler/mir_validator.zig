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
                if (!self.containsHirFunction(call.function)) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk null;
                }
                const callee = self.semantic_module.hir.getFunction(call.function);
                if (call.args.len != callee.params.len) {
                    try self.report(.InvalidMirOperand, span, diagnostics.invalidMirOperand);
                    break :blk callee.return_type;
                }
                for (call.args, callee.params) |arg, param_id| {
                    const arg_type = try self.operandType(function_id, arg, span);
                    const param_type = self.semantic_module.hir.getParam(param_id).type_id;
                    if (arg_type != null and !sameType(arg_type.?, param_type)) {
                        try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
                    }
                }
                break :blk callee.return_type;
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
        };
    }

    fn enumType(self: *Validator, enum_id: hir.EnumId) ?types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .enum_type and kind.enum_type.index == enum_id.index) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn requireValidType(self: *Validator, type_id: types.TypeId, span: ?diagnostics.SourceSpan) ValidationError!void {
        if (!self.semantic_module.types.contains(type_id)) {
            try self.report(.InvalidMirType, span, diagnostics.invalidMirType);
        }
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
    try built.module.store.appendStatement(built.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, .{ .call = .{ .function = callee, .args = args } }) });
    try built.module.store.setTerminator(built.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = temp })) });
    try ctx.validateOk(&built.module);
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
    try call_count.module.store.appendStatement(call_count.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = count_temp }, .{ .call = .{ .function = callee, .args = &.{} } }) });
    try call_count.module.store.setTerminator(call_count.block, .{ .span = synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = count_temp })) });
    try ctx.validateFail(&call_count.module, .InvalidMirOperand);

    var call_type = try validMirFunction(&ctx, ctx.module.types.intType());
    defer call_type.module.deinit();
    const type_temp = try call_type.module.store.addLocal(call_type.function, null, .temp, ctx.module.types.intType(), synthetic_span);
    const args = try std.testing.allocator.alloc(mir.MirOperand, 1);
    args[0] = mir.MirOperand.boolLiteral(false);
    try call_type.module.store.appendStatement(call_type.block, .{ .span = synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = type_temp }, .{ .call = .{ .function = callee, .args = args } }) });
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
}

test "MIR validator integration covers checked HIR lowering programs" {
    try validateLowered(.arithmetic);
    try validateLowered(.while_loop);
    try validateLowered(.match_int);
}

const LoweredCase = enum { arithmetic, if_else, while_loop, match_int };

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
