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
        if (function.body) |body| try function_lowerer.lowerStmt(body, entry);

        const block = self.mir_module.store.getBlock(entry);
        if (block.terminator == null) {
            try self.mir_module.store.setTerminator(entry, .{ .span = function.span, .kind = .@"unreachable" });
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

    fn lowerStmt(self: *FunctionLowerer, stmt_id: hir.StmtId, block_id: mir.MirBlockId) LoweringError!void {
        if (self.isTerminated(block_id)) return;

        const stmt = self.semantic_module.hir.getStmt(stmt_id).*;
        switch (stmt.kind) {
            .block => |children| {
                for (children) |child| {
                    if (self.isTerminated(block_id)) return;
                    try self.lowerStmt(child, block_id);
                }
            },
            .local_decl => |decl| {
                const local_id = try self.ensureLocal(decl.local);
                const value = try self.lowerExprToOperand(decl.initializer, block_id);
                try self.store.appendStatement(block_id, .{
                    .span = stmt.span,
                    .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(local_id), mir.MirRvalue.use_(value)),
                });
            },
            .assignment => |assignment| {
                const local_id = try self.resolveAssignTarget(assignment.target);
                const value = try self.lowerExprToOperand(assignment.value, block_id);
                try self.store.appendStatement(block_id, .{
                    .span = stmt.span,
                    .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(local_id), mir.MirRvalue.use_(value)),
                });
            },
            .return_stmt => |maybe_value| {
                const value = if (maybe_value) |expr_id| try self.lowerExprToOperand(expr_id, block_id) else null;
                try self.store.setTerminator(block_id, .{ .span = stmt.span, .kind = mir.MirTerminatorKind.returnValue(value) });
            },
            .if_stmt, .while_stmt, .match_stmt => return error.UnsupportedControlFlow,
        }
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
        };
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

test "MIR lowering rejects unsupported control flow" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const then_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = then_block, .else_block = null } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span);
    try setBody(&module, main, &.{ if_stmt, ret });

    try std.testing.expectError(error.UnsupportedControlFlow, lowerModule(std.testing.allocator, &module));

    var while_module = try newModule();
    defer while_module.deinit();
    const while_main = try addFunction(&while_module, "main", while_module.types.intType(), false);
    const while_body = try blockStmt(&while_module, &.{});
    const while_stmt = try while_module.hir.addStmt(.{ .while_stmt = .{ .condition = try boolExpr(&while_module, true), .body = while_body } }, hir.synthetic_span);
    try setBody(&while_module, while_main, &.{while_stmt});
    try std.testing.expectError(error.UnsupportedControlFlow, lowerModule(std.testing.allocator, &while_module));

    var match_module = try newModule();
    defer match_module.deinit();
    const match_main = try addFunction(&match_module, "main", match_module.types.intType(), false);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 1);
    arms[0] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&match_module, &.{}) };
    const match_stmt = try match_module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try intExpr(&match_module, "1"), .arms = arms } }, hir.synthetic_span);
    try setBody(&match_module, match_main, &.{match_stmt});
    try std.testing.expectError(error.UnsupportedControlFlow, lowerModule(std.testing.allocator, &match_module));
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
