const std = @import("std");

const hir = @import("hir.zig");
const mir = @import("mir.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");

pub const LoweringError = anyerror;

pub fn lowerModule(
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
) LoweringError!mir.MirModule {
    var lowerer = ModuleLowerer.init(allocator, semantic_module);
    return lowerer.lower();
}

const ModuleLowerer = struct {
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
    mir_module: mir.MirModule,

    fn init(allocator: std.mem.Allocator, semantic_module: *semantics.SemanticModule) ModuleLowerer {
        return .{
            .allocator = allocator,
            .semantic_module = semantic_module,
            .mir_module = mir.MirModule.init(allocator),
        };
    }

    fn lower(self: *ModuleLowerer) LoweringError!mir.MirModule {
        errdefer self.mir_module.deinit();

        for (self.semantic_module.hir.functions.items, 0..) |function, index| {
            if (function.body == null) continue;
            const hir_function_id = hir.FunctionId{ .index = @intCast(index) };
            try self.lowerFunction(hir_function_id, function);
        }

        const lowered = self.mir_module;
        self.mir_module = undefined;
        return lowered;
    }

    fn lowerFunction(self: *ModuleLowerer, hir_function_id: hir.FunctionId, function: hir.HirFunction) LoweringError!void {
        const function_id = try self.mir_module.store.addFunction(hir_function_id, function.name, function.return_type, function.span);
        var function_lowerer = FunctionLowerer.init(self.allocator, self.semantic_module, &self.mir_module.store, function_id);
        defer function_lowerer.deinit();

        for (function.params) |param_id| {
            const param = self.semantic_module.hir.getParam(param_id);
            const local_id = try self.mir_module.store.addLocal(function_id, param.name, .param, param.type_id, param.span);
            try function_lowerer.param_map.put(param_id, local_id);
        }

        const entry = try self.mir_module.store.addBlock(function_id, function.span);
        const exit = if (function.body) |body| try function_lowerer.lowerStmt(body, entry) else entry;

        const block = self.mir_module.store.getBlock(exit);
        if (block.terminator == null) {
            try self.mir_module.store.setTerminator(exit, .{ .span = function.span, .kind = .@"unreachable" });
        }
    }
};

const FunctionLowerer = struct {
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
    store: *mir.MirStore,
    function_id: mir.MirFunctionId,
    local_map: std.AutoHashMap(hir.LocalId, mir.MirLocalId),
    param_map: std.AutoHashMap(hir.ParamId, mir.MirLocalId),

    fn init(
        allocator: std.mem.Allocator,
        semantic_module: *semantics.SemanticModule,
        store: *mir.MirStore,
        function_id: mir.MirFunctionId,
    ) FunctionLowerer {
        return .{
            .allocator = allocator,
            .semantic_module = semantic_module,
            .store = store,
            .function_id = function_id,
            .local_map = std.AutoHashMap(hir.LocalId, mir.MirLocalId).init(allocator),
            .param_map = std.AutoHashMap(hir.ParamId, mir.MirLocalId).init(allocator),
        };
    }

    fn deinit(self: *FunctionLowerer) void {
        self.param_map.deinit();
        self.local_map.deinit();
    }

    fn lowerStmt(self: *FunctionLowerer, stmt_id: hir.StmtId, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        if (self.isTerminated(block_id)) return block_id;

        const stmt = self.semantic_module.hir.getStmt(stmt_id).*;
        switch (stmt.kind) {
            .block => |children| {
                var current = block_id;
                for (children) |child| {
                    if (self.isTerminated(current)) return current;
                    current = try self.lowerStmt(child, current);
                }
                return current;
            },
            .local_decl => |decl| {
                const local_id = try self.ensureLocal(decl.local);
                const value = try self.lowerExprToOperand(decl.initializer, block_id);
                try self.store.appendStatement(block_id, .{
                    .span = stmt.span,
                    .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(local_id), mir.MirRvalue.use_(value)),
                });
                return block_id;
            },
            .assignment => |assignment| {
                const local_id = try self.resolveAssignTarget(assignment.target);
                const value = try self.lowerExprToOperand(assignment.value, block_id);
                try self.store.appendStatement(block_id, .{
                    .span = stmt.span,
                    .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(local_id), mir.MirRvalue.use_(value)),
                });
                return block_id;
            },
            .return_stmt => |maybe_value| {
                const value = if (maybe_value) |expr_id| try self.lowerExprToOperand(expr_id, block_id) else null;
                try self.terminateBlock(block_id, stmt.span, mir.MirTerminatorKind.returnValue(value));
                return block_id;
            },
            .if_stmt => |if_stmt| return self.lowerIf(stmt, if_stmt, block_id),
            .while_stmt => |while_stmt| return self.lowerWhile(stmt, while_stmt, block_id),
            .match_stmt => |match_stmt| return self.lowerMatch(stmt, match_stmt, block_id),
        }
    }

    fn lowerIf(self: *FunctionLowerer, stmt: hir.HirStmt, if_stmt: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        const condition = try self.lowerExprToOperand(if_stmt.condition, block_id);
        const then_block = try self.newBlock(stmt.span);
        const join_block = try self.newBlock(stmt.span);
        const else_block = if (if_stmt.else_block != null) try self.newBlock(stmt.span) else join_block;

        try self.terminateBlock(block_id, stmt.span, mir.MirTerminatorKind.switchBool(condition, then_block, else_block));

        const then_end = try self.lowerStmt(if_stmt.then_block, then_block);
        const then_falls_through = !self.isTerminated(then_end);
        try self.ensureGotoTo(then_end, join_block, stmt.span);

        var has_fallthrough = then_falls_through or if_stmt.else_block == null;
        if (if_stmt.else_block) |else_stmt_id| {
            const else_end = try self.lowerStmt(else_stmt_id, else_block);
            const else_falls_through = !self.isTerminated(else_end);
            try self.ensureGotoTo(else_end, join_block, stmt.span);
            has_fallthrough = has_fallthrough or else_falls_through;
        }

        if (!has_fallthrough) {
            try self.terminateBlock(join_block, stmt.span, .@"unreachable");
        }
        return join_block;
    }

    fn lowerWhile(self: *FunctionLowerer, stmt: hir.HirStmt, while_stmt: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        const condition_block = try self.newBlock(stmt.span);
        const body_block = try self.newBlock(stmt.span);
        const exit_block = try self.newBlock(stmt.span);

        try self.terminateBlock(block_id, stmt.span, mir.MirTerminatorKind.gotoBlock(condition_block));

        const condition = try self.lowerExprToOperand(while_stmt.condition, condition_block);
        try self.terminateBlock(condition_block, stmt.span, mir.MirTerminatorKind.switchBool(condition, body_block, exit_block));

        const body_end = try self.lowerStmt(while_stmt.body, body_block);
        try self.ensureGotoTo(body_end, condition_block, stmt.span);

        return exit_block;
    }

    fn lowerMatch(self: *FunctionLowerer, stmt: hir.HirStmt, match_stmt: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        const scrutinee = try self.lowerExprToOperand(match_stmt.scrutinee, block_id);
        const scrutinee_type = try self.inferExprType(match_stmt.scrutinee);
        const join_block = try self.newBlock(stmt.span);
        const arm_blocks = try self.allocator.alloc(mir.MirBlockId, match_stmt.arms.len);
        defer self.allocator.free(arm_blocks);

        for (match_stmt.arms, 0..) |_, index| {
            arm_blocks[index] = try self.newBlock(stmt.span);
        }

        if (sameType(scrutinee_type, self.semantic_module.types.boolType())) {
            try self.terminateBlock(block_id, stmt.span, try self.lowerBoolMatchTerminator(scrutinee, match_stmt.arms, arm_blocks, join_block));
        } else if (sameType(scrutinee_type, self.semantic_module.types.intType())) {
            try self.terminateBlock(block_id, stmt.span, try self.lowerIntMatchTerminator(scrutinee, match_stmt.arms, arm_blocks, join_block));
        } else {
            return error.UnsupportedControlFlow;
        }

        var has_fallthrough = false;
        for (match_stmt.arms, 0..) |arm, index| {
            const arm_end = try self.lowerStmt(arm.body, arm_blocks[index]);
            if (!self.isTerminated(arm_end)) {
                has_fallthrough = true;
                try self.ensureGotoTo(arm_end, join_block, stmt.span);
            }
        }

        if (!has_fallthrough and !self.joinIsDefaultTarget(block_id, join_block)) {
            try self.terminateBlock(join_block, stmt.span, .@"unreachable");
        }
        return join_block;
    }

    fn lowerBoolMatchTerminator(
        self: *FunctionLowerer,
        scrutinee: mir.MirOperand,
        arms: []const hir.HirMatchArm,
        arm_blocks: []const mir.MirBlockId,
        default_target: mir.MirBlockId,
    ) LoweringError!mir.MirTerminatorKind {
        _ = self;
        var true_target: ?mir.MirBlockId = null;
        var false_target: ?mir.MirBlockId = null;
        var wildcard_target: ?mir.MirBlockId = null;

        for (arms, 0..) |arm, index| {
            switch (arm.pattern) {
                .bool_literal => |value| if (value) {
                    true_target = arm_blocks[index];
                } else {
                    false_target = arm_blocks[index];
                },
                .wildcard => wildcard_target = arm_blocks[index],
                .int_literal => return error.UnsupportedControlFlow,
            }
        }

        const fallback = wildcard_target orelse default_target;
        return mir.MirTerminatorKind.switchBool(scrutinee, true_target orelse fallback, false_target orelse fallback);
    }

    fn lowerIntMatchTerminator(
        self: *FunctionLowerer,
        scrutinee: mir.MirOperand,
        arms: []const hir.HirMatchArm,
        arm_blocks: []const mir.MirBlockId,
        default_target: mir.MirBlockId,
    ) LoweringError!mir.MirTerminatorKind {
        var cases = std.ArrayList(mir.MirSwitchIntCase).empty;
        defer cases.deinit(self.allocator);
        var fallback = default_target;

        for (arms, 0..) |arm, index| {
            switch (arm.pattern) {
                .int_literal => |text| try cases.append(self.allocator, .{ .value = text, .target = arm_blocks[index] }),
                .wildcard => fallback = arm_blocks[index],
                .bool_literal => return error.UnsupportedControlFlow,
            }
        }

        return try mir.MirTerminatorKind.switchInt(self.allocator, scrutinee, cases.items, fallback);
    }

    fn joinIsDefaultTarget(self: *FunctionLowerer, switch_block: mir.MirBlockId, join_block: mir.MirBlockId) bool {
        const terminator = self.store.getBlock(switch_block).terminator orelse return false;
        return switch (terminator.kind) {
            .switch_bool => |switch_bool| switch_bool.true_target.index == join_block.index or switch_bool.false_target.index == join_block.index,
            .switch_int => |switch_int| switch_int.default_target.index == join_block.index,
            else => false,
        };
    }

    fn lowerExprToOperand(self: *FunctionLowerer, expr_id: hir.ExprId, block_id: mir.MirBlockId) LoweringError!mir.MirOperand {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => |text| try mir.MirOperand.intLiteral(self.allocator, text),
            .bool_literal => |value| mir.MirOperand.boolLiteral(value),
            .local_ref => |local_id| mir.MirOperand.copyPlace(mir.MirPlace.localPlace(try self.resolveLocal(local_id))),
            .param_ref => |param_id| mir.MirOperand.copyPlace(mir.MirPlace.localPlace(try self.resolveParam(param_id))),
            .group => |inner| try self.lowerExprToOperand(inner, block_id),
            .unary => |unary| try self.lowerUnary(expr, unary, block_id),
            .binary => |binary| try self.lowerBinary(expr, binary, block_id),
            .call => |call| try self.lowerCall(expr, call, block_id),
            .enum_constructor => |constructor| try self.lowerEnumConstructor(expr, constructor, block_id),
        };
    }

    fn lowerUnary(self: *FunctionLowerer, expr: hir.HirExpr, unary: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirOperand {
        const operand = try self.lowerExprToOperand(unary.operand, block_id);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.unaryOp(lowerUnaryOp(unary.op), operand),
            ),
        });
        return mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp));
    }

    fn lowerBinary(self: *FunctionLowerer, expr: hir.HirExpr, binary: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirOperand {
        const left = try self.lowerExprToOperand(binary.left, block_id);
        const right = try self.lowerExprToOperand(binary.right, block_id);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.binaryOp(lowerBinaryOp(binary.op), left, right),
            ),
        });
        return mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp));
    }

    fn lowerCall(self: *FunctionLowerer, expr: hir.HirExpr, call: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirOperand {
        const args = try self.allocator.alloc(mir.MirOperand, call.args.len);
        var args_owned = true;
        var initialized: usize = 0;
        errdefer if (args_owned) deinitInitializedOperands(self.allocator, args, initialized);

        for (call.args) |arg_expr| {
            args[initialized] = try self.lowerExprToOperand(arg_expr, block_id);
            initialized += 1;
        }

        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        args_owned = false;
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                .{ .call = .{ .function = call.function, .args = args } },
            ),
        });
        return mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp));
    }

    fn lowerEnumConstructor(self: *FunctionLowerer, expr: hir.HirExpr, constructor: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirOperand {
        const args = try self.allocator.alloc(mir.MirOperand, constructor.args.len);
        var args_owned = true;
        var initialized: usize = 0;
        errdefer if (args_owned) deinitInitializedOperands(self.allocator, args, initialized);

        for (constructor.args) |arg_expr| {
            args[initialized] = try self.lowerExprToOperand(arg_expr, block_id);
            initialized += 1;
        }

        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        args_owned = false;
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                .{ .enum_constructor = .{ .enum_id = constructor.enum_id, .variant_id = constructor.variant_id, .args = args } },
            ),
        });
        return mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp));
    }

    fn inferExprType(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!types.TypeId {
        return self.inferExprTypeFrom(self.semantic_module.hir.getExpr(expr_id).*);
    }

    fn inferExprTypeFrom(self: *FunctionLowerer, expr: hir.HirExpr) LoweringError!types.TypeId {
        return switch (expr.kind) {
            .int_literal => self.semantic_module.types.intType(),
            .bool_literal => self.semantic_module.types.boolType(),
            .local_ref => |local_id| self.semantic_module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.semantic_module.hir.getParam(param_id).type_id,
            .group => |inner| try self.inferExprType(inner),
            .unary => |unary| switch (unary.op) {
                .negate => self.semantic_module.types.intType(),
                .logical_not => self.semantic_module.types.boolType(),
            },
            .binary => |binary| switch (binary.op) {
                .multiply, .divide, .modulo, .add, .subtract => self.semantic_module.types.intType(),
                .less, .less_equal, .greater, .greater_equal, .equal_equal, .bang_equal, .logical_and, .logical_or => self.semantic_module.types.boolType(),
            },
            .call => |call| self.semantic_module.hir.getFunction(call.function).return_type,
            .enum_constructor => |constructor| try self.enumType(constructor.enum_id),
        };
    }

    fn enumType(self: *FunctionLowerer, enum_id: hir.EnumId) LoweringError!types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .enum_type and kind.enum_type.index == enum_id.index) return .{ .index = @intCast(index) };
        }
        return error.MissingEnumType;
    }

    fn ensureLocal(self: *FunctionLowerer, local_id: hir.LocalId) LoweringError!mir.MirLocalId {
        if (self.local_map.get(local_id)) |mapped| return mapped;
        const local = self.semantic_module.hir.getLocal(local_id);
        const mapped = try self.store.addLocal(self.function_id, local.name, .user, local.type_id, local.span);
        try self.local_map.put(local_id, mapped);
        return mapped;
    }

    fn resolveLocal(self: *FunctionLowerer, local_id: hir.LocalId) LoweringError!mir.MirLocalId {
        return self.local_map.get(local_id) orelse error.MissingLocalMapping;
    }

    fn resolveParam(self: *FunctionLowerer, param_id: hir.ParamId) LoweringError!mir.MirLocalId {
        return self.param_map.get(param_id) orelse error.MissingParamMapping;
    }

    fn resolveAssignTarget(self: *FunctionLowerer, target: hir.AssignTarget) LoweringError!mir.MirLocalId {
        return switch (target) {
            .local => |local_id| try self.resolveLocal(local_id),
            .param => |param_id| try self.resolveParam(param_id),
        };
    }

    fn addTemp(self: *FunctionLowerer, type_id: types.TypeId) LoweringError!mir.MirLocalId {
        return self.store.addLocal(self.function_id, null, .temp, type_id, null);
    }

    fn newBlock(self: *FunctionLowerer, span: ?hir.SourceSpan) LoweringError!mir.MirBlockId {
        return self.store.addBlock(self.function_id, span);
    }

    fn terminateBlock(self: *FunctionLowerer, block_id: mir.MirBlockId, span: ?hir.SourceSpan, kind: mir.MirTerminatorKind) LoweringError!void {
        try self.store.setTerminator(block_id, .{ .span = span, .kind = kind });
    }

    fn ensureGotoTo(self: *FunctionLowerer, block_id: mir.MirBlockId, target: mir.MirBlockId, span: ?hir.SourceSpan) LoweringError!void {
        if (!self.isTerminated(block_id)) {
            try self.terminateBlock(block_id, span, mir.MirTerminatorKind.gotoBlock(target));
        }
    }

    fn isTerminated(self: *FunctionLowerer, block_id: mir.MirBlockId) bool {
        return self.store.getBlock(block_id).terminator != null;
    }
};

fn lowerUnaryOp(op: hir.UnaryOp) mir.MirUnaryOp {
    return switch (op) {
        .negate => .negate,
        .logical_not => .logical_not,
    };
}

fn lowerBinaryOp(op: hir.BinaryOp) mir.MirBinaryOp {
    return switch (op) {
        .multiply => .multiply,
        .divide => .divide,
        .modulo => .modulo,
        .add => .add,
        .subtract => .subtract,
        .less => .less,
        .less_equal => .less_equal,
        .greater => .greater,
        .greater_equal => .greater_equal,
        .equal_equal => .equal_equal,
        .bang_equal => .bang_equal,
        .logical_and => .logical_and,
        .logical_or => .logical_or,
    };
}

fn sameType(left: types.TypeId, right: types.TypeId) bool {
    return left.index == right.index;
}

fn deinitOperand(allocator: std.mem.Allocator, operand: mir.MirOperand) void {
    switch (operand) {
        .int_literal => |text| allocator.free(text),
        else => {},
    }
}

fn deinitInitializedOperands(allocator: std.mem.Allocator, operands: []mir.MirOperand, initialized: usize) void {
    for (operands[0..initialized]) |operand| deinitOperand(allocator, operand);
    if (operands.len > 0) allocator.free(operands);
}

fn newModule() !semantics.SemanticModule {
    return semantics.SemanticModule.init(std.testing.allocator);
}

fn intern(module: *semantics.SemanticModule, text: []const u8) !hir.SymbolId {
    return module.interner.intern(text);
}

fn addFunction(module: *semantics.SemanticModule, name: []const u8, return_type: types.TypeId, has_body: bool) !hir.FunctionId {
    const function_id = try module.hir.addFunction(try intern(module, name), return_type, hir.synthetic_span);
    if (has_body) {
        const body = try module.hir.addStmt(.{ .block = &.{} }, hir.synthetic_span);
        module.hir.setFunctionBody(function_id, body);
    }
    return function_id;
}

fn setBody(module: *semantics.SemanticModule, function_id: hir.FunctionId, statements: []const hir.StmtId) !void {
    const owned = try std.testing.allocator.alloc(hir.StmtId, statements.len);
    @memcpy(owned, statements);
    const body = try module.hir.addStmt(.{ .block = owned }, hir.synthetic_span);
    module.hir.setFunctionBody(function_id, body);
}

fn addParam(module: *semantics.SemanticModule, function_id: hir.FunctionId, name: []const u8, type_id: types.TypeId) !hir.ParamId {
    return module.hir.addParam(function_id, try intern(module, name), type_id, hir.synthetic_span);
}

fn addLocal(module: *semantics.SemanticModule, function_id: hir.FunctionId, name: []const u8, type_id: types.TypeId) !hir.LocalId {
    return module.hir.addLocal(function_id, try intern(module, name), type_id, hir.synthetic_span);
}

fn intExpr(module: *semantics.SemanticModule, text: []const u8) !hir.ExprId {
    return module.hir.addExpr(.{ .int_literal = try std.testing.allocator.dupe(u8, text) }, hir.synthetic_span);
}

fn boolExpr(module: *semantics.SemanticModule, value: bool) !hir.ExprId {
    return module.hir.addExpr(.{ .bool_literal = value }, hir.synthetic_span);
}

fn blockStmt(module: *semantics.SemanticModule, statements: []const hir.StmtId) !hir.StmtId {
    const owned = try std.testing.allocator.alloc(hir.StmtId, statements.len);
    @memcpy(owned, statements);
    return module.hir.addStmt(.{ .block = owned }, hir.synthetic_span);
}

test "MIR lowering skips function declarations without bodies" {
    var module = try newModule();
    defer module.deinit();
    _ = try addFunction(&module, "declared", module.types.intType(), false);
    const main = try addFunction(&module, "main", module.types.intType(), true);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    try std.testing.expectEqual(@as(usize, 1), mir_module.store.functions.items.len);
    try std.testing.expectEqualStrings("main", module.interner.text(mir_module.store.functions.items[0].name));
}

test "MIR lowering lowers return int" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "42") }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(@as(usize, 0), block.statements.len);
    try std.testing.expectEqualStrings("42", block.terminator.?.kind.return_.?.int_literal);
}

test "MIR lowering lowers empty body to unreachable" {
    var module = try newModule();
    defer module.deinit();
    _ = try addFunction(&module, "main", module.types.intType(), true);

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(@as(usize, 0), block.statements.len);
    try std.testing.expectEqual(mir.MirTerminatorKind.@"unreachable", block.terminator.?.kind);
}

test "MIR lowering lowers local declaration and local reference return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "1") } }, hir.synthetic_span);
    const ret_expr = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = ret_expr }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 1), function.locals.len);
    const local = mir_module.store.getLocal(function.locals[0]);
    try std.testing.expectEqual(mir.MirLocalKind.user, local.kind);
    try std.testing.expectEqualStrings("x", module.interner.text(local.name.?));

    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    try std.testing.expectEqualStrings("1", block.statements[0].kind.assign.rvalue.use.int_literal);
    try std.testing.expectEqual(function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers assignment" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "1") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span);
    const ret_expr = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = ret_expr }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, assign, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 2), block.statements.len);
    try std.testing.expectEqual(function.locals[0], block.statements[1].kind.assign.place.local);
    try std.testing.expectEqualStrings("2", block.statements[1].kind.assign.rvalue.use.int_literal);
}

test "MIR lowering lowers parameter assignment and grouped bool return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.boolType(), false);
    const flag = try addParam(&module, main, "flag", module.types.boolType());
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .param = flag }, .value = try boolExpr(&module, true) } }, hir.synthetic_span);
    const ref = try module.hir.addExpr(.{ .param_ref = flag }, hir.synthetic_span);
    const group = try module.hir.addExpr(.{ .group = ref }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = group }, hir.synthetic_span);
    try setBody(&module, main, &.{ assign, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 1), function.params.len);
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(function.params[0], block.statements[0].kind.assign.place.local);
    try std.testing.expectEqual(true, block.statements[0].kind.assign.rvalue.use.bool_literal);
    try std.testing.expectEqual(function.params[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers unary expression temp" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const one = try intExpr(&module, "1");
    const unary = try module.hir.addExpr(.{ .unary = .{ .op = .negate, .operand = one } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = unary }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    try std.testing.expectEqual(mir.MirLocalKind.temp, mir_module.store.getLocal(function.locals[0]).kind);
    try std.testing.expectEqual(mir.MirUnaryOp.negate, block.statements[0].kind.assign.rvalue.unary.op);
    try std.testing.expectEqual(function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers binary expression temp" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const binary = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try intExpr(&module, "1"), .right = try intExpr(&module, "2") } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = binary }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    try std.testing.expectEqual(mir.MirBinaryOp.add, block.statements[0].kind.assign.rvalue.binary.op);
    try std.testing.expectEqualStrings("1", block.statements[0].kind.assign.rvalue.binary.left.int_literal);
    try std.testing.expectEqualStrings("2", block.statements[0].kind.assign.rvalue.binary.right.int_literal);
    try std.testing.expectEqual(function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers function call temp" {
    var module = try newModule();
    defer module.deinit();
    const add = try addFunction(&module, "add", module.types.intType(), false);
    const a = try addParam(&module, add, "a", module.types.intType());
    const b = try addParam(&module, add, "b", module.types.intType());
    const add_expr = try module.hir.addExpr(.{ .binary = .{
        .op = .add,
        .left = try module.hir.addExpr(.{ .param_ref = a }, hir.synthetic_span),
        .right = try module.hir.addExpr(.{ .param_ref = b }, hir.synthetic_span),
    } }, hir.synthetic_span);
    try setBody(&module, add, &.{try module.hir.addStmt(.{ .return_stmt = add_expr }, hir.synthetic_span)});

    const main = try addFunction(&module, "main", module.types.intType(), false);
    const args = try std.testing.allocator.alloc(hir.ExprId, 2);
    args[0] = try intExpr(&module, "1");
    args[1] = try intExpr(&module, "2");
    const call = try module.hir.addExpr(.{ .call = .{ .function = add, .args = args } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = call }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const add_function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 2), add_function.params.len);

    const main_function = mir_module.store.functions.items[1];
    const block = mir_module.store.getBlock(main_function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    const lowered_call = block.statements[0].kind.assign.rvalue.call;
    try std.testing.expectEqual(hir.FunctionId{ .index = 0 }, lowered_call.function);
    try std.testing.expectEqual(@as(usize, 2), lowered_call.args.len);
    try std.testing.expectEqualStrings("1", lowered_call.args[0].int_literal);
    try std.testing.expectEqualStrings("2", lowered_call.args[1].int_literal);
    try std.testing.expectEqual(main_function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers if without else" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span);
    const then_block = try blockStmt(&module, &.{assign});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = then_block, .else_block = null } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 3), function.blocks.len);
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_bool, std.meta.activeTag(mir_module.store.getBlock(function.blocks[0]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.goto, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.return_, std.meta.activeTag(mir_module.store.getBlock(function.blocks[2]).terminator.?.kind));
}

test "MIR lowering lowers if else with branch return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const then_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)});
    const assign_else = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span);
    const else_block = try blockStmt(&module, &.{assign_else});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, false), .then_block = then_block, .else_block = else_block } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 4), function.blocks.len);
    try std.testing.expectEqual(mir.MirTerminatorKind.return_, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.goto, std.meta.activeTag(mir_module.store.getBlock(function.blocks[3]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.return_, std.meta.activeTag(mir_module.store.getBlock(function.blocks[2]).terminator.?.kind));
}

test "MIR lowering lowers if else when both branches return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const then_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)});
    const else_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "2") }, hir.synthetic_span)});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = then_block, .else_block = else_block } }, hir.synthetic_span);
    const dead = try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "3") }, hir.synthetic_span);
    try setBody(&module, main, &.{ if_stmt, dead });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(mir.MirTerminatorKind.@"unreachable", mir_module.store.getBlock(function.blocks[2]).terminator.?.kind);
}

test "MIR lowering lowers while with assignment back edge" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const condition = try module.hir.addExpr(.{ .binary = .{ .op = .less, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "3") } }, hir.synthetic_span);
    const increment = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "1") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = increment } }, hir.synthetic_span);
    const while_stmt = try module.hir.addStmt(.{ .while_stmt = .{ .condition = condition, .body = try blockStmt(&module, &.{assign}) } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), while_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 4), function.blocks.len);
    try std.testing.expectEqual(function.blocks[1], mir_module.store.getBlock(function.blocks[0]).terminator.?.kind.goto);
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_bool, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
    try std.testing.expectEqual(function.blocks[1], mir_module.store.getBlock(function.blocks[2]).terminator.?.kind.goto);
}

test "MIR lowering lowers match int default and bool" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const int_arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    int_arms[0] = .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span)}) };
    int_arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "3") } }, hir.synthetic_span)}) };
    const int_match = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try intExpr(&module, "1"), .arms = int_arms } }, hir.synthetic_span);
    const bool_arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    bool_arms[0] = .{ .pattern = .{ .bool_literal = true }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "4") } }, hir.synthetic_span)}) };
    bool_arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "5") }, hir.synthetic_span)}) };
    const bool_match = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try boolExpr(&module, false), .arms = bool_arms } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), int_match, bool_match, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_int, std.meta.activeTag(mir_module.store.getBlock(function.blocks[0]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_bool, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
}

test "MIR lowering adds unreachable for unterminated straight-line body" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "1") } }, hir.synthetic_span);
    try setBody(&module, main, &.{decl});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(mir.MirTerminatorKind.@"unreachable", block.terminator.?.kind);
}

test "MIR lowering ignores statements after return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const ret = try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span);
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "2") } }, hir.synthetic_span);
    try setBody(&module, main, &.{ ret, decl });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(@as(usize, 0), block.statements.len);
    try std.testing.expectEqualStrings("1", block.terminator.?.kind.return_.?.int_literal);
}

test "MIR lowering debug snapshot is stable for straight-line function" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const binary = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try intExpr(&module, "1"), .right = try intExpr(&module, "2") } }, hir.synthetic_span);
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = binary } }, hir.synthetic_span);
    const ret_expr = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = ret_expr }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\      MirLocalId(1) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(1) = Binary + Int 1, Int 2
        \\        MirLocalId(0) = Use(Copy(MirLocalId(1)))
        \\        Return Copy(MirLocalId(0))
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for if without else" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span);
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = try blockStmt(&module, &.{assign}), .else_block = null } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchBool Bool true true: MirBlockId(1), false: MirBlockId(2)
        \\      MirBlockId(1)
        \\        MirLocalId(0) = Use(Int 1)
        \\        Goto MirBlockId(2)
        \\      MirBlockId(2)
        \\        Return Copy(MirLocalId(0))
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for while count loop" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const condition = try module.hir.addExpr(.{ .binary = .{ .op = .less, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "3") } }, hir.synthetic_span);
    const increment = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "1") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = increment } }, hir.synthetic_span);
    const while_stmt = try module.hir.addStmt(.{ .while_stmt = .{ .condition = condition, .body = try blockStmt(&module, &.{assign}) } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, while_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\      MirLocalId(1) temp <temp>: TypeId(2)
        \\      MirLocalId(2) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        Goto MirBlockId(1)
        \\      MirBlockId(1)
        \\        MirLocalId(1) = Binary < Copy(MirLocalId(0)), Int 3
        \\        SwitchBool Copy(MirLocalId(1)) true: MirBlockId(2), false: MirBlockId(3)
        \\      MirBlockId(2)
        \\        MirLocalId(2) = Binary + Copy(MirLocalId(0)), Int 1
        \\        MirLocalId(0) = Use(Copy(MirLocalId(2)))
        \\        Goto MirBlockId(1)
        \\      MirBlockId(3)
        \\        Return Copy(MirLocalId(0))
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for match int default" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "3") } }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try intExpr(&module, "1"), .arms = arms } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchInt Int 1 [1: MirBlockId(2)] default: MirBlockId(3)
        \\      MirBlockId(1)
        \\        Return Copy(MirLocalId(0))
        \\      MirBlockId(2)
        \\        MirLocalId(0) = Use(Int 2)
        \\        Goto MirBlockId(1)
        \\      MirBlockId(3)
        \\        MirLocalId(0) = Use(Int 3)
        \\        Goto MirBlockId(1)
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for if else join" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const then_assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span);
    const else_assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span);
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, false), .then_block = try blockStmt(&module, &.{then_assign}), .else_block = try blockStmt(&module, &.{else_assign}) } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchBool Bool false true: MirBlockId(1), false: MirBlockId(3)
        \\      MirBlockId(1)
        \\        MirLocalId(0) = Use(Int 1)
        \\        Goto MirBlockId(2)
        \\      MirBlockId(2)
        \\        Return Copy(MirLocalId(0))
        \\      MirBlockId(3)
        \\        MirLocalId(0) = Use(Int 2)
        \\        Goto MirBlockId(2)
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for match bool" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .bool_literal = true }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try boolExpr(&module, true), .arms = arms } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchBool Bool true true: MirBlockId(2), false: MirBlockId(3)
        \\      MirBlockId(1)
        \\        Return Copy(MirLocalId(0))
        \\      MirBlockId(2)
        \\        MirLocalId(0) = Use(Int 1)
        \\        Goto MirBlockId(1)
        \\      MirBlockId(3)
        \\        MirLocalId(0) = Use(Int 2)
        \\        Goto MirBlockId(1)
        \\
    , snapshot);
}
