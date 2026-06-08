const std = @import("std");

const hir = @import("hir.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");

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

pub const CompileTimeError = error{
    UnsupportedExpression,
    TypeMismatch,
    DivisionByZero,
    Overflow,
    EvaluationFailed,
};

pub const CompileTimeResult = CompileTimeError!CompileTimeValue;

/// Small typed-HIR constant evaluator for P9-M1.
///
/// The evaluator assumes expressions have already passed HIR checking, but it
/// still rejects impossible value/operator pairs instead of coercing them. It is
/// intentionally not a constant-propagation engine: calls, locals, parameters,
/// aggregate values, field access, pointers, `try`, and `decide` are future work.
pub const CompileTimeEvaluator = struct {
    module: *const semantics.SemanticModule,

    pub fn init(module: *const semantics.SemanticModule) CompileTimeEvaluator {
        return .{ .module = module };
    }

    pub fn evaluateExpr(self: CompileTimeEvaluator, expr_id: hir.ExprId) CompileTimeResult {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => |text| .{ .int = std.fmt.parseInt(i64, text, 10) catch return error.Overflow },
            .bool_literal => |value| .{ .bool = value },
            .group => |inner| try self.evaluateExpr(inner),
            .unary => |unary| try self.evaluateUnary(unary.op, try self.evaluateExpr(unary.operand)),
            .binary => |binary| try self.evaluateBinary(binary.op, try self.evaluateExpr(binary.left), try self.evaluateExpr(binary.right)),
            .local_ref,
            .param_ref,
            .call,
            .concept_requirement_call,
            .enum_constructor,
            .struct_literal,
            .field_access,
            .decide,
            .address_of,
            .deref,
            .try_expr,
            .compile_time,
            => error.UnsupportedExpression,
        };
    }

    fn evaluateUnary(self: CompileTimeEvaluator, op: hir.UnaryOp, operand: CompileTimeValue) CompileTimeResult {
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

    fn evaluateBinary(self: CompileTimeEvaluator, op: hir.BinaryOp, left: CompileTimeValue, right: CompileTimeValue) CompileTimeResult {
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
        return CompileTimeEvaluator.init(&self.module);
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
};

fn expectValue(actual: CompileTimeResult, expected: CompileTimeValue) !void {
    try std.testing.expect((try actual).eql(expected));
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
    try expectValue(tm.evaluator().evaluateExpr(int_literal), .{ .int = 42 });

    const bool_literal = try tm.boolExpr(true);
    try expectValue(tm.evaluator().evaluateExpr(bool_literal), .{ .bool = true });

    const grouped = try tm.expr(.{ .group = try tm.int("7") });
    try expectValue(tm.evaluator().evaluateExpr(grouped), .{ .int = 7 });

    const addition = try tm.binary(.add, try tm.int("40"), try tm.int("2"));
    try expectValue(tm.evaluator().evaluateExpr(addition), .{ .int = 42 });

    const subtraction = try tm.binary(.subtract, try tm.int("50"), try tm.int("8"));
    try expectValue(tm.evaluator().evaluateExpr(subtraction), .{ .int = 42 });

    const multiplication = try tm.binary(.multiply, try tm.int("6"), try tm.int("7"));
    try expectValue(tm.evaluator().evaluateExpr(multiplication), .{ .int = 42 });

    const division = try tm.binary(.divide, try tm.int("84"), try tm.int("2"));
    try expectValue(tm.evaluator().evaluateExpr(division), .{ .int = 42 });

    const negation = try tm.unary(.negate, try tm.int("42"));
    try expectValue(tm.evaluator().evaluateExpr(negation), .{ .int = -42 });
}

test "compile-time evaluator evaluates comparisons and boolean operators" {
    var tm = try TestModule.init();
    defer tm.deinit();

    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.less, try tm.int("1"), try tm.int("2"))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.less_equal, try tm.int("2"), try tm.int("2"))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.greater, try tm.int("3"), try tm.int("2"))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.greater_equal, try tm.int("2"), try tm.int("2"))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.equal_equal, try tm.int("2"), try tm.int("2"))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.bang_equal, try tm.int("2"), try tm.int("3"))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.unary(.logical_not, try tm.boolExpr(false))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.equal_equal, try tm.boolExpr(true), try tm.boolExpr(true))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.bang_equal, try tm.boolExpr(true), try tm.boolExpr(false))), .{ .bool = true });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.logical_and, try tm.boolExpr(true), try tm.boolExpr(false))), .{ .bool = false });
    try expectValue(tm.evaluator().evaluateExpr(try tm.binary(.logical_or, try tm.boolExpr(true), try tm.boolExpr(false))), .{ .bool = true });
}

test "compile-time evaluator rejects unsupported expression forms" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const callee = try tm.module.hir.addFunction(try tm.name("callee"), tm.module.types.intType(), hir.synthetic_span);
    const no_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try tm.expr(.{ .call = .{ .function = callee, .args = no_args } });
    try std.testing.expectError(error.UnsupportedExpression, tm.evaluator().evaluateExpr(call));

    const struct_id = try tm.module.hir.addStruct(try tm.name("Pair"));
    const struct_type = try tm.module.types.addStructType(struct_id);
    const field_id = try tm.module.hir.addField(struct_id, try tm.name("value"), tm.module.types.intType(), hir.synthetic_span);
    const fields = try std.testing.allocator.dupe(hir.HirStructLiteralField, &.{.{ .field_id = field_id, .value = try tm.int("1"), .span = hir.synthetic_span }});
    const struct_literal = try tm.expr(.{ .struct_literal = .{ .struct_id = struct_id, .type_id = struct_type, .fields = fields } });
    try std.testing.expectError(error.UnsupportedExpression, tm.evaluator().evaluateExpr(struct_literal));

    const field_access = try tm.expr(.{ .field_access = .{ .receiver = struct_literal, .field_name = try tm.name("value"), .field_span = hir.synthetic_span } });
    try std.testing.expectError(error.UnsupportedExpression, tm.evaluator().evaluateExpr(field_access));
}

test "compile-time evaluator rejects mismatches division by zero overflow and modulo" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const mismatch = try tm.binary(.add, try tm.int("1"), try tm.boolExpr(true));
    try std.testing.expectError(error.TypeMismatch, tm.evaluator().evaluateExpr(mismatch));

    const divide_by_zero = try tm.binary(.divide, try tm.int("1"), try tm.int("0"));
    try std.testing.expectError(error.DivisionByZero, tm.evaluator().evaluateExpr(divide_by_zero));

    const add_overflow = try tm.binary(.add, try tm.int("9223372036854775807"), try tm.int("1"));
    try std.testing.expectError(error.Overflow, tm.evaluator().evaluateExpr(add_overflow));

    const negate_overflow = try tm.unary(.negate, try tm.int("-9223372036854775808"));
    try std.testing.expectError(error.Overflow, tm.evaluator().evaluateExpr(negate_overflow));

    const modulo = try tm.binary(.modulo, try tm.int("3"), try tm.int("2"));
    try std.testing.expectError(error.UnsupportedExpression, tm.evaluator().evaluateExpr(modulo));
}
