const std = @import("std");

const hir = @import("hir.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");
const compile_time_target = @import("compile_time_target.zig");

/// P9-M1 compile-time values are deliberately limited to hermetic scalar
/// constants. The evaluator below has no access to filesystem, environment,
/// network, time, random, process spawning, host calls, or arbitrary externs.
pub const CompileTimeValue = union(enum) {
    int: i64,
    bool: bool,

    pub fn typeOf(self: CompileTimeValue, type_store: types.TypeStore) types.TypeId {
        return switch (self) {
            .int => type_store.intType(),
            .bool => type_store.boolType(),
        };
    }

    pub fn eql(self: CompileTimeValue, other: CompileTimeValue) bool {
        return switch (self) {
            .int => |left| switch (other) {
                .int => |right| left == right,
                .bool => false,
            },
            .bool => |left| switch (other) {
                .int => false,
                .bool => |right| left == right,
            },
        };
    }

    pub fn toIntLiteral(self: CompileTimeValue, allocator: std.mem.Allocator) ![]const u8 {
        return switch (self) {
            .int => |value| try std.fmt.allocPrint(allocator, "{d}", .{value}),
            .bool => error.NotInt,
        };
    }

    pub fn format(self: CompileTimeValue, writer: *std.Io.Writer) std.Io.Writer.Error!void {
        switch (self) {
            .int => |value| try writer.print("{d}", .{value}),
            .bool => |value| try writer.writeAll(if (value) "true" else "false"),
        }
    }
};

pub const CompileTimeBudget = struct {
    /// P9-M6 default step budget. This is intentionally fixed for now so
    /// compile-time execution is deterministic and cannot hang the compiler.
    pub const default_step_limit: usize = 100_000;

    remaining_steps: usize = default_step_limit,
};

pub const CompileTimeError = error{
    UnsupportedExpression,
    UnsupportedStatement,
    TypeMismatch,
    DivisionByZero,
    Overflow,
    EvaluationFailed,
    FunctionRequired,
    UnsupportedBody,
    UnsupportedSignature,
    RecursionLimit,
    ArgumentTypeMismatch,
    UnsupportedLocalType,
    MissingReturn,
    IfRequiresBool,
    UnboundLocal,
    AssignmentTypeMismatch,
    FuelExhausted,
    WhileRequiresBool,
    CapabilityNotGranted,
    TargetMetadataUnavailable,
};

pub const CompileTimeResult = CompileTimeError!CompileTimeValue;

pub const CompileTimeFrame = struct {
    parameters: []const ParamBinding = &.{},
    locals: std.ArrayList(LocalBinding) = .empty,
    scope_marks: std.ArrayList(usize) = .empty,

    pub const ParamBinding = struct {
        param: hir.ParamId,
        value: CompileTimeValue,
    };

    pub const LocalBinding = struct {
        local: hir.LocalId,
        value: CompileTimeValue,
    };

    pub fn deinit(self: *CompileTimeFrame, allocator: std.mem.Allocator) void {
        self.locals.deinit(allocator);
        self.scope_marks.deinit(allocator);
    }

    pub fn pushScope(self: *CompileTimeFrame, allocator: std.mem.Allocator) CompileTimeError!void {
        self.scope_marks.append(allocator, self.locals.items.len) catch return error.EvaluationFailed;
    }

    pub fn popScope(self: *CompileTimeFrame) void {
        const mark = self.scope_marks.pop() orelse return;
        self.locals.shrinkRetainingCapacity(mark);
    }

    pub fn bindLocal(self: *CompileTimeFrame, allocator: std.mem.Allocator, local: hir.LocalId, value: CompileTimeValue) CompileTimeError!void {
        if (self.lookupLocalIndex(local)) |_| return error.EvaluationFailed;
        self.locals.append(allocator, .{ .local = local, .value = value }) catch return error.EvaluationFailed;
    }

    pub fn assignLocal(self: *CompileTimeFrame, local: hir.LocalId, value: CompileTimeValue) CompileTimeError!void {
        const index = self.lookupLocalIndex(local) orelse return error.UnboundLocal;
        self.locals.items[index].value = value;
    }

    fn lookupParam(self: CompileTimeFrame, param: hir.ParamId) ?CompileTimeValue {
        for (self.parameters) |binding| {
            if (binding.param.index == param.index) return binding.value;
        }
        return null;
    }

    fn lookupLocal(self: CompileTimeFrame, local: hir.LocalId) ?CompileTimeValue {
        const index = self.lookupLocalIndex(local) orelse return null;
        return self.locals.items[index].value;
    }

    fn lookupLocalIndex(self: CompileTimeFrame, local: hir.LocalId) ?usize {
        var index = self.locals.items.len;
        while (index > 0) {
            index -= 1;
            if (self.locals.items[index].local.index == local.index) return index;
        }
        return null;
    }
};

/// Small typed-HIR constant evaluator for P9-M1.
///
/// The evaluator assumes expressions have already passed HIR checking, but it
/// still rejects impossible value/operator pairs instead of coercing them. It is
/// intentionally not a constant-propagation engine: aggregate values, field
/// access, pointers, `try`, `decide`, and unsupported statement forms are
/// future work.
pub const CompileTimeEvaluator = struct {
    module: *const semantics.SemanticModule,
    allocator: std.mem.Allocator,
    frame: CompileTimeFrame = .{},
    call_depth: usize = 0,
    budget: CompileTimeBudget = .{},

    const max_call_depth = 32;

    pub fn init(module: *const semantics.SemanticModule, allocator: std.mem.Allocator) CompileTimeEvaluator {
        return .{ .module = module, .allocator = allocator };
    }

    pub fn initWithBudget(module: *const semantics.SemanticModule, allocator: std.mem.Allocator, budget: CompileTimeBudget) CompileTimeEvaluator {
        return .{ .module = module, .allocator = allocator, .budget = budget };
    }

    fn consumeFuel(self: *CompileTimeEvaluator) CompileTimeError!void {
        if (self.budget.remaining_steps == 0) return error.FuelExhausted;
        self.budget.remaining_steps -= 1;
    }

    pub fn evaluateExpr(self: *CompileTimeEvaluator, expr_id: hir.ExprId) CompileTimeResult {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => |text| .{ .int = std.fmt.parseInt(i64, text, 10) catch return error.Overflow },
            .bool_literal => |value| .{ .bool = value },
            .group => |inner| try self.evaluateExpr(inner),
            .compile_time => |compile_time_expr| try self.evaluateExpr(compile_time_expr.operand),
            .unary => |unary| try self.evaluateUnary(unary.op, try self.evaluateExpr(unary.operand)),
            .binary => |binary| switch (binary.op) {
                .logical_and => try self.evaluateLogicalAnd(binary.left, binary.right),
                .logical_or => try self.evaluateLogicalOr(binary.left, binary.right),
                else => try self.evaluateBinary(binary.op, try self.evaluateExpr(binary.left), try self.evaluateExpr(binary.right)),
            },
            .param_ref => |param| self.frame.lookupParam(param) orelse error.UnsupportedExpression,
            .local_ref => |local| self.frame.lookupLocal(local) orelse error.UnboundLocal,
            .call => |call| try self.evaluateCall(call.function, call.args),
            .target_metadata => |metadata| self.evaluateTargetMetadata(metadata.query),
            .arena_alloc,
            .machine_param_ref,
            .concept_requirement_call,
            .enum_constructor,
            .struct_literal,
            .field_access,
            .decide,
            .address_of,
            .deref,
            .move_expr,
            .manual_init_assume,
            .try_expr,
            .test_intrinsic,
            => error.UnsupportedExpression,
        };
    }

    fn evaluateCall(self: *CompileTimeEvaluator, function_id: hir.FunctionId, args: []const hir.ExprId) CompileTimeResult {
        try self.consumeFuel();
        if (self.call_depth >= max_call_depth) return error.RecursionLimit;

        const function = self.module.hir.getFunction(function_id);
        if (!function.is_compile_time) return error.FunctionRequired;
        if (function.compile_time_capabilities.len != 0) return error.CapabilityNotGranted;
        try self.requireSupportedSignature(function);
        if (args.len != function.params.len) return error.ArgumentTypeMismatch;

        const bindings = self.allocator.alloc(CompileTimeFrame.ParamBinding, args.len) catch return error.EvaluationFailed;
        defer self.allocator.free(bindings);
        for (args, function.params, 0..) |arg, param_id, index| {
            const value = try self.evaluateExpr(arg);
            const param_type = self.module.hir.getParam(param_id).type_id;
            if (!sameType(value.typeOf(self.module.types), param_type)) return error.ArgumentTypeMismatch;
            bindings[index] = .{ .param = param_id, .value = value };
        }

        var nested = CompileTimeEvaluator{
            .module = self.module,
            .allocator = self.allocator,
            .frame = .{ .parameters = bindings },
            .call_depth = self.call_depth + 1,
            .budget = self.budget,
        };
        defer {
            self.budget = nested.budget;
            nested.frame.deinit(self.allocator);
        }
        return nested.evaluateFunctionBody(function);
    }

    fn evaluateTargetMetadata(self: *CompileTimeEvaluator, query: compile_time_target.CompileTimeTargetQuery) CompileTimeValue {
        return switch (compile_time_target.queryTargetInfo(self.module.compile_time_target_info, query)) {
            .int => |value| .{ .int = value },
            .bool => |value| .{ .bool = value },
        };
    }

    fn requireSupportedSignature(self: *CompileTimeEvaluator, function: *const hir.HirFunction) CompileTimeError!void {
        if (!self.isSupportedScalarType(function.return_type)) return error.UnsupportedSignature;
        for (function.params) |param_id| {
            if (!self.isSupportedScalarType(self.module.hir.getParam(param_id).type_id)) return error.UnsupportedSignature;
        }
    }

    fn evaluateFunctionBody(self: *CompileTimeEvaluator, function: *const hir.HirFunction) CompileTimeResult {
        const body = function.body orelse return error.UnsupportedBody;
        return switch (try self.executeStmt(body)) {
            .continue_ => error.MissingReturn,
            .return_value => |value| blk: {
                if (!sameType(value.typeOf(self.module.types), function.return_type)) return error.TypeMismatch;
                break :blk value;
            },
        };
    }

    const ExecResult = union(enum) {
        continue_,
        return_value: CompileTimeValue,
    };

    fn executeStmt(self: *CompileTimeEvaluator, stmt_id: hir.StmtId) CompileTimeError!ExecResult {
        try self.consumeFuel();
        const stmt = self.module.hir.getStmt(stmt_id);
        return switch (stmt.kind) {
            .block => |children| try self.executeBlock(children),
            .return_stmt => |maybe_value| .{ .return_value = try self.evaluateExpr(maybe_value orelse return error.UnsupportedBody) },
            .local_decl => |decl| blk: {
                const local = self.module.hir.getLocal(decl.local);
                if (!self.isSupportedScalarType(local.type_id)) return error.UnsupportedLocalType;
                const value = try self.evaluateExpr(decl.initializer);
                if (!sameType(value.typeOf(self.module.types), local.type_id)) return error.TypeMismatch;
                try self.frame.bindLocal(self.allocator, decl.local, value);
                break :blk .continue_;
            },
            .assignment => |assignment| blk: {
                const local_id = switch (assignment.target) {
                    .local => |local| local,
                    .param, .field => return error.UnsupportedStatement,
                };
                const local = self.module.hir.getLocal(local_id);
                const value = try self.evaluateExpr(assignment.value);
                if (!sameType(value.typeOf(self.module.types), local.type_id)) return error.AssignmentTypeMismatch;
                try self.frame.assignLocal(local_id, value);
                break :blk .continue_;
            },
            .if_stmt => |if_stmt| blk: {
                const condition = try self.evaluateExpr(if_stmt.condition);
                const selected = switch (condition) {
                    .bool => |value| if (value) if_stmt.then_block else if_stmt.else_block,
                    .int => return error.IfRequiresBool,
                };
                if (selected) |branch| break :blk try self.executeStmt(branch);
                break :blk .continue_;
            },
            .while_stmt => |while_stmt| try self.executeWhile(while_stmt.condition, while_stmt.body),
            .expr_stmt, .discard_stmt, .transition_stmt, .arena_reset, .arena_destroy, .unsafe_block, .match_stmt => error.UnsupportedStatement,
        };
    }

    fn executeWhile(self: *CompileTimeEvaluator, condition_id: hir.ExprId, body_id: hir.StmtId) CompileTimeError!ExecResult {
        while (true) {
            const condition = try self.evaluateExpr(condition_id);
            const should_enter = switch (condition) {
                .bool => |value| value,
                .int => return error.WhileRequiresBool,
            };
            if (!should_enter) return .continue_;
            try self.consumeFuel();
            const result = try self.executeStmt(body_id);
            switch (result) {
                .continue_ => {},
                .return_value => return result,
            }
        }
    }

    fn executeBlock(self: *CompileTimeEvaluator, children: []const hir.StmtId) CompileTimeError!ExecResult {
        try self.frame.pushScope(self.allocator);
        defer self.frame.popScope();
        for (children) |child| {
            const result = try self.executeStmt(child);
            switch (result) {
                .continue_ => {},
                .return_value => return result,
            }
        }
        return .continue_;
    }

    fn isSupportedScalarType(self: *CompileTimeEvaluator, type_id: types.TypeId) bool {
        return sameType(type_id, self.module.types.intType()) or sameType(type_id, self.module.types.boolType());
    }

    fn evaluateUnary(self: *CompileTimeEvaluator, op: hir.UnaryOp, operand: CompileTimeValue) CompileTimeResult {
        _ = self;
        return switch (op) {
            .negate => switch (operand) {
                .int => |value| .{ .int = std.math.negate(value) catch return error.Overflow },
                .bool => error.TypeMismatch,
            },
            .logical_not => switch (operand) {
                .int => error.TypeMismatch,
                .bool => |value| .{ .bool = !value },
            },
        };
    }

    fn evaluateLogicalAnd(self: *CompileTimeEvaluator, left_id: hir.ExprId, right_id: hir.ExprId) CompileTimeResult {
        const left = try self.evaluateExpr(left_id);
        const left_value = switch (left) {
            .bool => |value| value,
            .int => return error.TypeMismatch,
        };
        if (!left_value) return .{ .bool = false };

        const right = try self.evaluateExpr(right_id);
        const right_value = switch (right) {
            .bool => |value| value,
            .int => return error.TypeMismatch,
        };
        return .{ .bool = right_value };
    }

    fn evaluateLogicalOr(self: *CompileTimeEvaluator, left_id: hir.ExprId, right_id: hir.ExprId) CompileTimeResult {
        const left = try self.evaluateExpr(left_id);
        const left_value = switch (left) {
            .bool => |value| value,
            .int => return error.TypeMismatch,
        };
        if (left_value) return .{ .bool = true };

        const right = try self.evaluateExpr(right_id);
        const right_value = switch (right) {
            .bool => |value| value,
            .int => return error.TypeMismatch,
        };
        return .{ .bool = right_value };
    }

    fn evaluateBinary(self: *CompileTimeEvaluator, op: hir.BinaryOp, left: CompileTimeValue, right: CompileTimeValue) CompileTimeResult {
        _ = self;
        return switch (op) {
            .add => try intArithmetic(left, right, checkedAdd),
            .subtract => try intArithmetic(left, right, checkedSub),
            .multiply => try intArithmetic(left, right, checkedMul),
            .divide => try intArithmetic(left, right, checkedDiv),
            .modulo => error.UnsupportedExpression,
            .less => try intComparison(left, right, less),
            .less_equal => try intComparison(left, right, lessEqual),
            .greater => try intComparison(left, right, greater),
            .greater_equal => try intComparison(left, right, greaterEqual),
            .equal_equal => try equality(left, right, true),
            .bang_equal => try equality(left, right, false),
            .logical_and => switch (left) {
                .bool => |left_value| switch (right) {
                    .bool => |right_value| .{ .bool = left_value and right_value },
                    .int => error.TypeMismatch,
                },
                .int => error.TypeMismatch,
            },
            .logical_or => switch (left) {
                .bool => |left_value| switch (right) {
                    .bool => |right_value| .{ .bool = left_value or right_value },
                    .int => error.TypeMismatch,
                },
                .int => error.TypeMismatch,
            },
        };
    }
};

fn sameType(left: types.TypeId, right: types.TypeId) bool {
    return left.index == right.index;
}

fn intArithmetic(left: CompileTimeValue, right: CompileTimeValue, operation: fn (i64, i64) CompileTimeError!i64) CompileTimeResult {
    return switch (left) {
        .int => |left_value| switch (right) {
            .int => |right_value| .{ .int = try operation(left_value, right_value) },
            .bool => error.TypeMismatch,
        },
        .bool => error.TypeMismatch,
    };
}

fn intComparison(left: CompileTimeValue, right: CompileTimeValue, predicate: fn (i64, i64) bool) CompileTimeResult {
    return switch (left) {
        .int => |left_value| switch (right) {
            .int => |right_value| .{ .bool = predicate(left_value, right_value) },
            .bool => error.TypeMismatch,
        },
        .bool => error.TypeMismatch,
    };
}

fn equality(left: CompileTimeValue, right: CompileTimeValue, equal: bool) CompileTimeResult {
    return switch (left) {
        .int => |left_value| switch (right) {
            .int => |right_value| .{ .bool = if (equal) left_value == right_value else left_value != right_value },
            .bool => error.TypeMismatch,
        },
        .bool => |left_value| switch (right) {
            .int => error.TypeMismatch,
            .bool => |right_value| .{ .bool = if (equal) left_value == right_value else left_value != right_value },
        },
    };
}

fn checkedAdd(left: i64, right: i64) CompileTimeError!i64 {
    return std.math.add(i64, left, right) catch return error.Overflow;
}

fn checkedSub(left: i64, right: i64) CompileTimeError!i64 {
    return std.math.sub(i64, left, right) catch return error.Overflow;
}

fn checkedMul(left: i64, right: i64) CompileTimeError!i64 {
    return std.math.mul(i64, left, right) catch return error.Overflow;
}

fn checkedDiv(left: i64, right: i64) CompileTimeError!i64 {
    if (right == 0) return error.DivisionByZero;
    return std.math.divTrunc(i64, left, right) catch return error.Overflow;
}

fn less(left: i64, right: i64) bool {
    return left < right;
}

fn lessEqual(left: i64, right: i64) bool {
    return left <= right;
}

fn greater(left: i64, right: i64) bool {
    return left > right;
}

fn greaterEqual(left: i64, right: i64) bool {
    return left >= right;
}

const TestModule = struct {
    module: semantics.SemanticModule,

    fn init() !TestModule {
        return .{ .module = try semantics.SemanticModule.init(std.testing.allocator) };
    }

    fn deinit(self: *TestModule) void {
        self.module.deinit();
    }

    fn evaluator(self: *const TestModule) CompileTimeEvaluator {
        return CompileTimeEvaluator.init(&self.module, std.testing.allocator);
    }

    fn name(self: *TestModule, text: []const u8) !hir.SymbolId {
        return try self.module.interner.intern(text);
    }

    fn expr(self: *TestModule, kind: hir.HirExprKind) !hir.ExprId {
        return try self.module.hir.addExpr(kind, hir.synthetic_span);
    }

    fn int(self: *TestModule, text: []const u8) !hir.ExprId {
        return try self.expr(.{ .int_literal = try self.module.hir.allocator.dupe(u8, text) });
    }

    fn boolExpr(self: *TestModule, value: bool) !hir.ExprId {
        return try self.expr(.{ .bool_literal = value });
    }

    fn unary(self: *TestModule, op: hir.UnaryOp, operand: hir.ExprId) !hir.ExprId {
        return try self.expr(.{ .unary = .{ .op = op, .operand = operand } });
    }

    fn binary(self: *TestModule, op: hir.BinaryOp, left: hir.ExprId, right: hir.ExprId) !hir.ExprId {
        return try self.expr(.{ .binary = .{ .op = op, .left = left, .right = right } });
    }

    fn compileTimeFunction(self: *TestModule, name_text: []const u8, return_type: types.TypeId) !hir.FunctionId {
        const function_id = try self.module.hir.addFunction(try self.name(name_text), return_type, hir.synthetic_span);
        self.module.hir.markFunctionCompileTime(function_id);
        return function_id;
    }

    fn param(self: *TestModule, function_id: hir.FunctionId, name_text: []const u8, type_id: types.TypeId) !hir.ParamId {
        return self.module.hir.addParam(function_id, try self.name(name_text), type_id, hir.synthetic_span);
    }

    fn local(self: *TestModule, function_id: hir.FunctionId, name_text: []const u8, type_id: types.TypeId) !hir.LocalId {
        return self.module.hir.addLocal(function_id, try self.name(name_text), type_id, hir.synthetic_span);
    }

    fn block(self: *TestModule, stmts: []const hir.StmtId) !hir.StmtId {
        const owned = try std.testing.allocator.dupe(hir.StmtId, stmts);
        return self.module.hir.addStmt(.{ .block = owned }, hir.synthetic_span);
    }

    fn ret(self: *TestModule, expr_id: hir.ExprId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .return_stmt = expr_id }, hir.synthetic_span);
    }

    fn localDecl(self: *TestModule, local_id: hir.LocalId, initializer: hir.ExprId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .local_decl = .{ .local = local_id, .initializer = initializer } }, hir.synthetic_span);
    }

    fn assignLocal(self: *TestModule, local_id: hir.LocalId, value: hir.ExprId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = local_id }, .value = value } }, hir.synthetic_span);
    }

    fn ifStmt(self: *TestModule, condition: hir.ExprId, then_block: hir.StmtId, else_block: ?hir.StmtId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .if_stmt = .{ .condition = condition, .then_block = then_block, .else_block = else_block } }, hir.synthetic_span);
    }

    fn whileStmt(self: *TestModule, condition: hir.ExprId, body: hir.StmtId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .while_stmt = .{ .condition = condition, .body = body } }, hir.synthetic_span);
    }

    fn setBody(self: *TestModule, function_id: hir.FunctionId, stmt_id: hir.StmtId) void {
        self.module.hir.setFunctionBody(function_id, stmt_id);
    }
};

fn eval(tm: *const TestModule, expr_id: hir.ExprId) CompileTimeResult {
    var evaluator = tm.evaluator();
    return evaluator.evaluateExpr(expr_id);
}

fn evalWithBudget(tm: *const TestModule, expr_id: hir.ExprId, remaining_steps: usize) CompileTimeResult {
    var evaluator = CompileTimeEvaluator.initWithBudget(&tm.module, std.testing.allocator, .{ .remaining_steps = remaining_steps });
    return evaluator.evaluateExpr(expr_id);
}

fn expectValue(actual: CompileTimeResult, expected: CompileTimeValue) !void {
    try std.testing.expect((try actual).eql(expected));
}

test "CompileTimeFrame supports parameter local lookup assignment and scopes" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const function_id = try tm.compileTimeFunction("f", tm.module.types.intType());
    const param_id = try tm.param(function_id, "x", tm.module.types.intType());
    const local_id = try tm.local(function_id, "y", tm.module.types.intType());

    var frame = CompileTimeFrame{
        .parameters = try std.testing.allocator.dupe(CompileTimeFrame.ParamBinding, &.{.{ .param = param_id, .value = .{ .int = 1 } }}),
    };
    defer {
        std.testing.allocator.free(frame.parameters);
        frame.deinit(std.testing.allocator);
    }

    try std.testing.expectEqual(CompileTimeValue{ .int = 1 }, frame.lookupParam(param_id).?);
    try std.testing.expect(frame.lookupLocal(local_id) == null);
    try frame.pushScope(std.testing.allocator);
    try frame.bindLocal(std.testing.allocator, local_id, .{ .int = 2 });
    try std.testing.expectEqual(CompileTimeValue{ .int = 2 }, frame.lookupLocal(local_id).?);
    try frame.assignLocal(local_id, .{ .int = 3 });
    try std.testing.expectEqual(CompileTimeValue{ .int = 3 }, frame.lookupLocal(local_id).?);
    frame.popScope();
    try std.testing.expect(frame.lookupLocal(local_id) == null);
}

test "CompileTimeValue reports scalar Concept types" {
    var tm = try TestModule.init();
    defer tm.deinit();

    try std.testing.expectEqual(tm.module.types.intType(), (CompileTimeValue{ .int = 42 }).typeOf(tm.module.types));
    try std.testing.expectEqual(tm.module.types.boolType(), (CompileTimeValue{ .bool = true }).typeOf(tm.module.types));
}

test "compile-time evaluator evaluates literals grouping unary and arithmetic" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const int_literal = try tm.int("42");
    try expectValue(eval(&tm, int_literal), .{ .int = 42 });

    const bool_literal = try tm.boolExpr(true);
    try expectValue(eval(&tm, bool_literal), .{ .bool = true });

    const grouped = try tm.expr(.{ .group = try tm.int("7") });
    try expectValue(eval(&tm, grouped), .{ .int = 7 });

    const addition = try tm.binary(.add, try tm.int("40"), try tm.int("2"));
    try expectValue(eval(&tm, addition), .{ .int = 42 });

    const subtraction = try tm.binary(.subtract, try tm.int("50"), try tm.int("8"));
    try expectValue(eval(&tm, subtraction), .{ .int = 42 });

    const multiplication = try tm.binary(.multiply, try tm.int("6"), try tm.int("7"));
    try expectValue(eval(&tm, multiplication), .{ .int = 42 });

    const division = try tm.binary(.divide, try tm.int("84"), try tm.int("2"));
    try expectValue(eval(&tm, division), .{ .int = 42 });

    const negation = try tm.unary(.negate, try tm.int("42"));
    try expectValue(eval(&tm, negation), .{ .int = -42 });
}

test "compile-time evaluator evaluates comparisons and boolean operators" {
    var tm = try TestModule.init();
    defer tm.deinit();

    try expectValue(eval(&tm, try tm.binary(.less, try tm.int("1"), try tm.int("2"))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.less_equal, try tm.int("2"), try tm.int("2"))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.greater, try tm.int("3"), try tm.int("2"))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.greater_equal, try tm.int("2"), try tm.int("2"))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.equal_equal, try tm.int("2"), try tm.int("2"))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.bang_equal, try tm.int("2"), try tm.int("3"))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.unary(.logical_not, try tm.boolExpr(false))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.equal_equal, try tm.boolExpr(true), try tm.boolExpr(true))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.bang_equal, try tm.boolExpr(true), try tm.boolExpr(false))), .{ .bool = true });
    try expectValue(eval(&tm, try tm.binary(.logical_and, try tm.boolExpr(true), try tm.boolExpr(false))), .{ .bool = false });
    try expectValue(eval(&tm, try tm.binary(.logical_or, try tm.boolExpr(true), try tm.boolExpr(false))), .{ .bool = true });
}

test "compile-time evaluator short-circuits logical operators" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const runtime_only = try tm.module.hir.addFunction(try tm.name("runtimeOnly"), tm.module.types.boolType(), hir.synthetic_span);

    const runtime_call_for_or = try tm.expr(.{ .call = .{ .function = runtime_only, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } });
    try expectValue(eval(&tm, try tm.binary(.logical_or, try tm.boolExpr(true), runtime_call_for_or)), .{ .bool = true });

    const runtime_call_for_and = try tm.expr(.{ .call = .{ .function = runtime_only, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } });
    try expectValue(eval(&tm, try tm.binary(.logical_and, try tm.boolExpr(false), runtime_call_for_and)), .{ .bool = false });

    const evaluated_or_rhs = try tm.expr(.{ .call = .{ .function = runtime_only, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } });
    try std.testing.expectError(error.FunctionRequired, eval(&tm, try tm.binary(.logical_or, try tm.boolExpr(false), evaluated_or_rhs)));

    const evaluated_and_rhs = try tm.expr(.{ .call = .{ .function = runtime_only, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } });
    try std.testing.expectError(error.FunctionRequired, eval(&tm, try tm.binary(.logical_and, try tm.boolExpr(true), evaluated_and_rhs)));
}

test "compile-time evaluator rejects unsupported expression forms" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const callee = try tm.module.hir.addFunction(try tm.name("callee"), tm.module.types.intType(), hir.synthetic_span);
    const no_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try tm.expr(.{ .call = .{ .function = callee, .args = no_args } });
    try std.testing.expectError(error.FunctionRequired, eval(&tm, call));

    const struct_id = try tm.module.hir.addStruct(try tm.name("Pair"));
    const struct_type = try tm.module.types.addStructType(struct_id);
    const field_id = try tm.module.hir.addField(struct_id, try tm.name("value"), tm.module.types.intType(), hir.synthetic_span);
    const fields = try std.testing.allocator.dupe(hir.HirStructLiteralField, &.{.{ .field_id = field_id, .value = try tm.int("1"), .span = hir.synthetic_span }});
    const struct_literal = try tm.expr(.{ .struct_literal = .{ .struct_id = struct_id, .type_id = struct_type, .fields = fields } });
    try std.testing.expectError(error.UnsupportedExpression, eval(&tm, struct_literal));

    const field_access = try tm.expr(.{ .field_access = .{ .receiver = struct_literal, .field_name = try tm.name("value"), .field_span = hir.synthetic_span } });
    try std.testing.expectError(error.UnsupportedExpression, eval(&tm, field_access));
}

test "compile-time evaluator calls compile-time function with parameter bindings" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const add = try tm.compileTimeFunction("add", tm.module.types.intType());
    const a = try tm.param(add, "a", tm.module.types.intType());
    const b = try tm.param(add, "b", tm.module.types.intType());
    const a_ref = try tm.expr(.{ .param_ref = a });
    const b_ref = try tm.expr(.{ .param_ref = b });
    const sum = try tm.binary(.add, a_ref, b_ref);
    tm.setBody(add, try tm.block(&.{try tm.ret(sum)}));

    const args = try std.testing.allocator.dupe(hir.ExprId, &.{ try tm.int("20"), try tm.int("22") });
    const call = try tm.expr(.{ .call = .{ .function = add, .args = args } });
    try expectValue(eval(&tm, call), .{ .int = 42 });
}

test "compile-time evaluator supports nested function calls and recursion guard" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const one = try tm.compileTimeFunction("one", tm.module.types.intType());
    tm.setBody(one, try tm.block(&.{try tm.ret(try tm.int("1"))}));

    const two = try tm.compileTimeFunction("two", tm.module.types.intType());
    const no_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const one_call = try tm.expr(.{ .call = .{ .function = one, .args = no_args } });
    tm.setBody(two, try tm.block(&.{try tm.ret(try tm.binary(.add, one_call, try tm.int("1")))}));

    const two_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const two_call = try tm.expr(.{ .call = .{ .function = two, .args = two_args } });
    try expectValue(eval(&tm, two_call), .{ .int = 2 });

    const recur = try tm.compileTimeFunction("recur", tm.module.types.intType());
    const recur_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const recur_call = try tm.expr(.{ .call = .{ .function = recur, .args = recur_args } });
    tm.setBody(recur, try tm.block(&.{try tm.ret(recur_call)}));
    try std.testing.expectError(error.RecursionLimit, eval(&tm, recur_call));
}

test "compile-time evaluator rejects mismatches division by zero overflow and modulo" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const mismatch = try tm.binary(.add, try tm.int("1"), try tm.boolExpr(true));
    try std.testing.expectError(error.TypeMismatch, eval(&tm, mismatch));

    const divide_by_zero = try tm.binary(.divide, try tm.int("1"), try tm.int("0"));
    try std.testing.expectError(error.DivisionByZero, eval(&tm, divide_by_zero));

    const add_overflow = try tm.binary(.add, try tm.int("9223372036854775807"), try tm.int("1"));
    try std.testing.expectError(error.Overflow, eval(&tm, add_overflow));

    const negate_overflow = try tm.unary(.negate, try tm.int("-9223372036854775808"));
    try std.testing.expectError(error.Overflow, eval(&tm, negate_overflow));

    const modulo = try tm.binary(.modulo, try tm.int("3"), try tm.int("2"));
    try std.testing.expectError(error.UnsupportedExpression, eval(&tm, modulo));
}

test "compile-time evaluator supports local declarations references and assignment" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const f = try tm.compileTimeFunction("f", tm.module.types.intType());
    const x = try tm.param(f, "x", tm.module.types.intType());
    const y = try tm.local(f, "y", tm.module.types.intType());
    const y_ref = try tm.expr(.{ .local_ref = y });
    const x_ref = try tm.expr(.{ .param_ref = x });
    tm.setBody(f, try tm.block(&.{
        try tm.localDecl(y, x_ref),
        try tm.assignLocal(y, try tm.binary(.add, y_ref, try tm.int("1"))),
        try tm.ret(y_ref),
    }));

    const args = try std.testing.allocator.dupe(hir.ExprId, &.{try tm.int("41")});
    const call = try tm.expr(.{ .call = .{ .function = f, .args = args } });
    try expectValue(eval(&tm, call), .{ .int = 42 });
}

test "compile-time evaluator supports if branches and fallthrough" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const choose = try tm.compileTimeFunction("choose", tm.module.types.intType());
    const condition = try tm.param(choose, "condition", tm.module.types.boolType());
    const a = try tm.param(choose, "a", tm.module.types.intType());
    const b = try tm.param(choose, "b", tm.module.types.intType());
    const then_block = try tm.block(&.{try tm.ret(try tm.expr(.{ .param_ref = a }))});
    const else_block = try tm.block(&.{try tm.ret(try tm.expr(.{ .param_ref = b }))});
    tm.setBody(choose, try tm.block(&.{try tm.ifStmt(try tm.expr(.{ .param_ref = condition }), then_block, else_block)}));

    const true_args = try std.testing.allocator.dupe(hir.ExprId, &.{ try tm.boolExpr(true), try tm.int("42"), try tm.int("1") });
    try expectValue(eval(&tm, try tm.expr(.{ .call = .{ .function = choose, .args = true_args } })), .{ .int = 42 });

    const false_args = try std.testing.allocator.dupe(hir.ExprId, &.{ try tm.boolExpr(false), try tm.int("1"), try tm.int("42") });
    try expectValue(eval(&tm, try tm.expr(.{ .call = .{ .function = choose, .args = false_args } })), .{ .int = 42 });

    const abs = try tm.compileTimeFunction("abs", tm.module.types.intType());
    const x = try tm.param(abs, "x", tm.module.types.intType());
    const x_ref = try tm.expr(.{ .param_ref = x });
    const abs_then = try tm.block(&.{try tm.ret(try tm.unary(.negate, x_ref))});
    tm.setBody(abs, try tm.block(&.{
        try tm.ifStmt(try tm.binary(.less, x_ref, try tm.int("0")), abs_then, null),
        try tm.ret(x_ref),
    }));
    const abs_args = try std.testing.allocator.dupe(hir.ExprId, &.{try tm.int("-42")});
    try expectValue(eval(&tm, try tm.expr(.{ .call = .{ .function = abs, .args = abs_args } })), .{ .int = 42 });
}

test "compile-time evaluator supports while loops fuel and scoped bodies" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const sum = try tm.compileTimeFunction("sum", tm.module.types.intType());
    const n = try tm.param(sum, "n", tm.module.types.intType());
    const i = try tm.local(sum, "i", tm.module.types.intType());
    const total = try tm.local(sum, "total", tm.module.types.intType());
    const n_ref = try tm.expr(.{ .param_ref = n });
    const i_ref = try tm.expr(.{ .local_ref = i });
    const total_ref = try tm.expr(.{ .local_ref = total });
    tm.setBody(sum, try tm.block(&.{
        try tm.localDecl(i, try tm.int("0")),
        try tm.localDecl(total, try tm.int("0")),
        try tm.whileStmt(try tm.binary(.less_equal, i_ref, n_ref), try tm.block(&.{
            try tm.assignLocal(total, try tm.binary(.add, total_ref, i_ref)),
            try tm.assignLocal(i, try tm.binary(.add, i_ref, try tm.int("1"))),
        })),
        try tm.ret(total_ref),
    }));
    const sum_args = try std.testing.allocator.dupe(hir.ExprId, &.{try tm.int("9")});
    try expectValue(eval(&tm, try tm.expr(.{ .call = .{ .function = sum, .args = sum_args } })), .{ .int = 45 });

    const fallthrough = try tm.compileTimeFunction("fallthrough", tm.module.types.intType());
    tm.setBody(fallthrough, try tm.block(&.{
        try tm.whileStmt(try tm.boolExpr(false), try tm.block(&.{try tm.ret(try tm.int("1"))})),
        try tm.ret(try tm.int("3")),
    }));
    const fallthrough_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try expectValue(eval(&tm, try tm.expr(.{ .call = .{ .function = fallthrough, .args = fallthrough_args } })), .{ .int = 3 });

    const first = try tm.compileTimeFunction("first", tm.module.types.intType());
    const x = try tm.local(first, "x", tm.module.types.intType());
    const x_ref = try tm.expr(.{ .local_ref = x });
    tm.setBody(first, try tm.block(&.{
        try tm.localDecl(x, try tm.int("0")),
        try tm.whileStmt(try tm.boolExpr(true), try tm.block(&.{
            try tm.ifStmt(try tm.binary(.equal_equal, x_ref, try tm.int("3")), try tm.block(&.{try tm.ret(x_ref)}), null),
            try tm.assignLocal(x, try tm.binary(.add, x_ref, try tm.int("1"))),
        })),
        try tm.ret(try tm.int("0")),
    }));
    const first_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try expectValue(eval(&tm, try tm.expr(.{ .call = .{ .function = first, .args = first_args } })), .{ .int = 3 });

    const leak = try tm.compileTimeFunction("leak", tm.module.types.intType());
    const inner = try tm.local(leak, "inner", tm.module.types.intType());
    tm.setBody(leak, try tm.block(&.{
        try tm.whileStmt(try tm.boolExpr(false), try tm.block(&.{try tm.localDecl(inner, try tm.int("1"))})),
        try tm.ret(try tm.expr(.{ .local_ref = inner })),
    }));
    const leak_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try std.testing.expectError(error.UnboundLocal, eval(&tm, try tm.expr(.{ .call = .{ .function = leak, .args = leak_args } })));

    const infinite = try tm.compileTimeFunction("infinite", tm.module.types.intType());
    tm.setBody(infinite, try tm.block(&.{
        try tm.whileStmt(try tm.boolExpr(true), try tm.block(&.{})),
        try tm.ret(try tm.int("0")),
    }));
    const infinite_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try std.testing.expectError(error.FuelExhausted, evalWithBudget(&tm, try tm.expr(.{ .call = .{ .function = infinite, .args = infinite_args } }), 8));
}

test "compile-time evaluator rejects missing return unsupported statements and non-bool conditions" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const missing = try tm.compileTimeFunction("missing", tm.module.types.intType());
    tm.setBody(missing, try tm.block(&.{}));
    const no_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try std.testing.expectError(error.MissingReturn, eval(&tm, try tm.expr(.{ .call = .{ .function = missing, .args = no_args } })));

    const unsupported = try tm.compileTimeFunction("unsupported", tm.module.types.intType());
    tm.setBody(unsupported, try tm.block(&.{try tm.module.hir.addStmt(.{ .expr_stmt = try tm.int("0") }, hir.synthetic_span)}));
    const unsupported_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try std.testing.expectError(error.UnsupportedStatement, eval(&tm, try tm.expr(.{ .call = .{ .function = unsupported, .args = unsupported_args } })));

    const bad_if = try tm.compileTimeFunction("bad_if", tm.module.types.intType());
    tm.setBody(bad_if, try tm.block(&.{try tm.ifStmt(try tm.int("1"), try tm.block(&.{try tm.ret(try tm.int("1"))}), null)}));
    const bad_if_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try std.testing.expectError(error.IfRequiresBool, eval(&tm, try tm.expr(.{ .call = .{ .function = bad_if, .args = bad_if_args } })));

    const bad_while = try tm.compileTimeFunction("bad_while", tm.module.types.intType());
    tm.setBody(bad_while, try tm.block(&.{try tm.whileStmt(try tm.int("1"), try tm.block(&.{try tm.ret(try tm.int("1"))}))}));
    const bad_while_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    try std.testing.expectError(error.WhileRequiresBool, eval(&tm, try tm.expr(.{ .call = .{ .function = bad_while, .args = bad_while_args } })));
}

test "compile-time evaluator returns configured target metadata values" {
    var tm = try TestModule.init();
    defer tm.deinit();

    try expectValue(eval(&tm, try tm.expr(.{ .target_metadata = .{ .query = .pointer_size, .field_span = hir.synthetic_span } })), .{ .int = 8 });
    try expectValue(eval(&tm, try tm.expr(.{ .target_metadata = .{ .query = .is_little_endian, .field_span = hir.synthetic_span } })), .{ .bool = true });
    try expectValue(eval(&tm, try tm.expr(.{ .target_metadata = .{ .query = .is_big_endian, .field_span = hir.synthetic_span } })), .{ .bool = false });
}
