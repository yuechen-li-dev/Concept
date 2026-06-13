const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const hir_checker = @import("hir_checker.zig");
const semantics = @import("semantics.zig");
const source = @import("source.zig");
const test_discovery = @import("test_discovery.zig");

pub const RunError = error{
    CheckFailed,
    TestFailed,
    UnsupportedConstruct,
    MissingFunctionBody,
    InvalidValue,
    LoopFuelExhausted,
} || std.mem.Allocator.Error;

pub const Value = union(enum) {
    void,
    int: i64,
    bool: bool,
};

pub const TestFailure = struct {
    module_name: []const u8,
    function_name: []const u8,
    source_span: source.SourceSpan,
    intrinsic_kind: hir.HirTestIntrinsicKind,
    reason: []const u8,
    expected: ?[]const u8 = null,
    actual: ?[]const u8 = null,
    message: []const u8,

    pub fn deinit(self: TestFailure, allocator: std.mem.Allocator) void {
        allocator.free(self.module_name);
        allocator.free(self.function_name);
        allocator.free(self.reason);
        if (self.expected) |text| allocator.free(text);
        if (self.actual) |text| allocator.free(text);
        allocator.free(self.message);
    }
};

pub const TestRunResult = struct {
    passed_count: usize = 0,
    failed_count: usize = 0,
    skipped_count: usize = 0,
    failures: []TestFailure = &.{},

    pub fn deinit(self: TestRunResult, allocator: std.mem.Allocator) void {
        for (self.failures) |failure| failure.deinit(allocator);
        if (self.failures.len > 0) allocator.free(self.failures);
    }
};

pub fn runModule(
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    module_name: []const u8,
) RunError!TestRunResult {
    var check_diagnostics = diagnostics.DiagnosticBag.init(allocator);
    defer check_diagnostics.deinit();

    hir_checker.checkTestModule(allocator, module, &check_diagnostics) catch |err| switch (err) {
        error.InvalidSemanticModule => return error.CheckFailed,
        error.OutOfMemory => return error.OutOfMemory,
        else => return error.CheckFailed,
    };
    if (check_diagnostics.count() != 0) return error.CheckFailed;

    const discovered = try test_discovery.discoverTests(allocator, module, module_name, .@"test");
    defer allocator.free(discovered);

    var runner = Runner.init(allocator, module);
    defer runner.deinit();
    return runner.run(discovered);
}

pub fn formatFailure(allocator: std.mem.Allocator, failure: TestFailure) ![]const u8 {
    var output = std.Io.Writer.Allocating.init(allocator);
    errdefer output.deinit();
    const writer = &output.writer;

    try writer.print(
        \\FAILED {s}.{s}
        \\
        \\{s} failed
        \\
        \\Because:
        \\  {s}
        \\
    , .{
        failure.module_name,
        failure.function_name,
        failureMessageName(failure.intrinsic_kind),
        failure.reason,
    });

    if (failure.expected) |expected| {
        try writer.print(
            \\
            \\Expected:
            \\  {s}
            \\
        , .{expected});
    }
    if (failure.actual) |actual| {
        try writer.print(
            \\
            \\Actual:
            \\  {s}
            \\
        , .{actual});
    }

    return output.toOwnedSlice();
}

const Runner = struct {
    allocator: std.mem.Allocator,
    module: *const semantics.SemanticModule,
    failures: std.ArrayList(TestFailure),
    current_module_name: []const u8 = "",
    current_function_name: []const u8 = "",

    fn init(allocator: std.mem.Allocator, module: *const semantics.SemanticModule) Runner {
        return .{
            .allocator = allocator,
            .module = module,
            .failures = std.ArrayList(TestFailure).empty,
        };
    }

    fn deinit(self: *Runner) void {
        self.failures.deinit(self.allocator);
    }

    fn run(self: *Runner, discovered: []const test_discovery.DiscoveredTest) RunError!TestRunResult {
        var result = TestRunResult{};

        for (discovered) |test_case| {
            if (test_case.attribute_kind != .fact) continue;

            self.current_module_name = test_case.module_name;
            self.current_function_name = test_case.function_name;

            const function_id = self.findFunction(test_case.function_name) orelse return error.UnsupportedConstruct;
            const before_failures = self.failures.items.len;
            _ = self.executeFunction(function_id, &.{}) catch |err| switch (err) {
                error.TestFailed => {},
                else => |other| return other,
            };

            if (self.failures.items.len == before_failures) {
                result.passed_count += 1;
            } else {
                result.failed_count += 1;
            }
        }

        result.failures = try self.failures.toOwnedSlice(self.allocator);
        return result;
    }

    fn findFunction(self: *Runner, name: []const u8) ?hir.FunctionId {
        for (self.module.hir.functions.items, 0..) |function, index| {
            if (std.mem.eql(u8, self.module.interner.text(function.name), name)) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn executeFunction(self: *Runner, function_id: hir.FunctionId, args: []const Value) RunError!Value {
        const function = self.module.hir.getFunction(function_id);
        const body = function.body orelse return error.MissingFunctionBody;
        if (args.len != function.params.len) return error.UnsupportedConstruct;

        var frame = try Frame.init(self.allocator, self.module, function, args);
        defer frame.deinit(self.allocator);

        const result = try self.executeStmt(body, &frame, function.return_type);
        return result orelse .void;
    }

    fn executeStmt(self: *Runner, stmt_id: hir.StmtId, frame: *Frame, return_type: anytype) RunError!?Value {
        const stmt = self.module.hir.getStmt(stmt_id).*;
        return switch (stmt.kind) {
            .block => |children| blk: {
                for (children) |child| {
                    if (try self.executeStmt(child, frame, return_type)) |value| break :blk value;
                }
                break :blk null;
            },
            .local_decl => |decl| {
                frame.locals[decl.local.index] = try self.evaluateExpr(decl.initializer, frame);
                return null;
            },
            .assignment => |assignment| {
                const value = try self.evaluateExpr(assignment.value, frame);
                switch (assignment.target) {
                    .local => |local| frame.locals[local.index] = value,
                    .param => |param| frame.params[param.index] = value,
                    .field => return error.UnsupportedConstruct,
                }
                return null;
            },
            .expr_stmt => |expr_id| {
                _ = try self.evaluateExpr(expr_id, frame);
                return null;
            },
            .discard_stmt => |expr_id| {
                _ = try self.evaluateExpr(expr_id, frame);
                return null;
            },
            .return_stmt => |maybe_expr| if (maybe_expr) |expr_id| try self.evaluateExpr(expr_id, frame) else .void,
            .if_stmt => |if_stmt| blk: {
                const condition = try self.evaluateExpr(if_stmt.condition, frame);
                const branch = if (try valueAsBool(condition)) if_stmt.then_block else if_stmt.else_block;
                if (branch) |branch_id| break :blk try self.executeStmt(branch_id, frame, return_type);
                break :blk null;
            },
            .while_stmt => |while_stmt| blk: {
                var fuel: usize = 100000;
                while (try valueAsBool(try self.evaluateExpr(while_stmt.condition, frame))) {
                    if (fuel == 0) return error.LoopFuelExhausted;
                    fuel -= 1;
                    if (try self.executeStmt(while_stmt.body, frame, return_type)) |value| break :blk value;
                }
                break :blk null;
            },
            .unsafe_block => |body| try self.executeStmt(body, frame, return_type),
            .match_stmt => error.UnsupportedConstruct,
        };
    }

    fn evaluateExpr(self: *Runner, expr_id: hir.ExprId, frame: *Frame) RunError!Value {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => |text| .{ .int = std.fmt.parseInt(i64, text, 10) catch return error.InvalidValue },
            .bool_literal => |value| .{ .bool = value },
            .local_ref => |local| frame.locals[local.index],
            .param_ref => |param| frame.params[param.index],
            .group => |inner| try self.evaluateExpr(inner, frame),
            .unary => |unary| try self.evaluateUnary(unary, frame),
            .binary => |binary| try self.evaluateBinary(binary, frame),
            .call => |call| try self.evaluateCall(call, frame),
            .test_intrinsic => |test_intrinsic| {
                try self.evaluateTestIntrinsic(expr.span, test_intrinsic, frame);
                return .void;
            },
            .compile_time => |compile_time_expr| try self.evaluateExpr(compile_time_expr.operand, frame),
            .address_of,
            .deref,
            .move_expr,
            .manual_init_assume,
            .try_expr,
            .concept_requirement_call,
            .enum_constructor,
            .struct_literal,
            .field_access,
            .target_metadata,
            .decide,
            => error.UnsupportedConstruct,
        };
    }

    fn evaluateUnary(self: *Runner, unary: anytype, frame: *Frame) RunError!Value {
        const operand = try self.evaluateExpr(unary.operand, frame);
        return switch (unary.op) {
            .negate => .{ .int = -(try valueAsInt(operand)) },
            .logical_not => .{ .bool = !(try valueAsBool(operand)) },
        };
    }

    fn evaluateBinary(self: *Runner, binary: anytype, frame: *Frame) RunError!Value {
        const left = try self.evaluateExpr(binary.left, frame);
        const right = try self.evaluateExpr(binary.right, frame);
        return switch (binary.op) {
            .add => .{ .int = (try valueAsInt(left)) + (try valueAsInt(right)) },
            .subtract => .{ .int = (try valueAsInt(left)) - (try valueAsInt(right)) },
            .multiply => .{ .int = (try valueAsInt(left)) * (try valueAsInt(right)) },
            .divide => .{ .int = @divTrunc(try valueAsInt(left), try valueAsInt(right)) },
            .modulo => .{ .int = @rem(try valueAsInt(left), try valueAsInt(right)) },
            .less => .{ .bool = (try valueAsInt(left)) < (try valueAsInt(right)) },
            .less_equal => .{ .bool = (try valueAsInt(left)) <= (try valueAsInt(right)) },
            .greater => .{ .bool = (try valueAsInt(left)) > (try valueAsInt(right)) },
            .greater_equal => .{ .bool = (try valueAsInt(left)) >= (try valueAsInt(right)) },
            .equal_equal => .{ .bool = valuesEqual(left, right) },
            .bang_equal => .{ .bool = !valuesEqual(left, right) },
            .logical_and => .{ .bool = (try valueAsBool(left)) and (try valueAsBool(right)) },
            .logical_or => .{ .bool = (try valueAsBool(left)) or (try valueAsBool(right)) },
        };
    }

    fn evaluateCall(self: *Runner, call: anytype, frame: *Frame) RunError!Value {
        var args = try self.allocator.alloc(Value, call.args.len);
        defer self.allocator.free(args);
        for (call.args, 0..) |arg_id, index| {
            args[index] = try self.evaluateExpr(arg_id, frame);
        }
        return self.executeFunction(call.function, args);
    }

    fn evaluateTestIntrinsic(self: *Runner, span: source.SourceSpan, test_intrinsic: hir.HirTestIntrinsic, frame: *Frame) RunError!void {
        switch (test_intrinsic.kind) {
            .assert_true, .expect_true => {
                const actual = try self.evaluateExpr(test_intrinsic.operands[0], frame);
                const actual_bool = try valueAsBool(actual);
                if (!actual_bool) try self.recordFailure(span, test_intrinsic, .{ .bool = true }, actual);
            },
            .assert_false, .expect_false => {
                const actual = try self.evaluateExpr(test_intrinsic.operands[0], frame);
                const actual_bool = try valueAsBool(actual);
                if (actual_bool) try self.recordFailure(span, test_intrinsic, .{ .bool = false }, actual);
            },
            .expect_equal_int, .expect_equal_bool => {
                const expected = try self.evaluateExpr(test_intrinsic.operands[0], frame);
                const actual = try self.evaluateExpr(test_intrinsic.operands[1], frame);
                if (!valuesEqual(expected, actual)) try self.recordFailure(span, test_intrinsic, expected, actual);
            },
        }
    }

    fn recordFailure(self: *Runner, span: source.SourceSpan, test_intrinsic: hir.HirTestIntrinsic, expected: Value, actual: Value) RunError!void {
        try self.failures.append(self.allocator, .{
            .module_name = try self.allocator.dupe(u8, self.current_module_name),
            .function_name = try self.allocator.dupe(u8, self.current_function_name),
            .source_span = span,
            .intrinsic_kind = test_intrinsic.kind,
            .reason = try self.allocator.dupe(u8, test_intrinsic.reason),
            .expected = try formatValue(self.allocator, expected),
            .actual = try formatValue(self.allocator, actual),
            .message = try std.fmt.allocPrint(self.allocator, "{s} failed", .{failureMessageName(test_intrinsic.kind)}),
        });
        return error.TestFailed;
    }
};

const Frame = struct {
    locals: []Value,
    params: []Value,

    fn init(
        allocator: std.mem.Allocator,
        module: *const semantics.SemanticModule,
        function: *const hir.HirFunction,
        args: []const Value,
    ) !Frame {
        const locals = try allocator.alloc(Value, module.hir.locals.items.len);
        errdefer allocator.free(locals);
        var params = try allocator.alloc(Value, module.hir.params.items.len);
        errdefer allocator.free(params);
        @memset(locals, .void);
        @memset(params, .void);
        for (function.params, args) |param_id, value| params[param_id.index] = value;
        return .{ .locals = locals, .params = params };
    }

    fn deinit(self: *Frame, allocator: std.mem.Allocator) void {
        allocator.free(self.locals);
        allocator.free(self.params);
        self.* = undefined;
    }
};

fn valueAsInt(value: Value) RunError!i64 {
    return switch (value) {
        .int => |int| int,
        else => error.InvalidValue,
    };
}

fn valueAsBool(value: Value) RunError!bool {
    return switch (value) {
        .bool => |bool_value| bool_value,
        else => error.InvalidValue,
    };
}

fn valuesEqual(left: Value, right: Value) bool {
    return switch (left) {
        .int => |left_int| switch (right) {
            .int => |right_int| left_int == right_int,
            else => false,
        },
        .bool => |left_bool| switch (right) {
            .bool => |right_bool| left_bool == right_bool,
            else => false,
        },
        .void => switch (right) {
            .void => true,
            else => false,
        },
    };
}

fn formatValue(allocator: std.mem.Allocator, value: Value) ![]const u8 {
    return switch (value) {
        .void => allocator.dupe(u8, "void"),
        .int => |int| std.fmt.allocPrint(allocator, "{d}", .{int}),
        .bool => |bool_value| allocator.dupe(u8, if (bool_value) "true" else "false"),
    };
}

fn failureMessageName(kind: hir.HirTestIntrinsicKind) []const u8 {
    return switch (kind) {
        .assert_true => "Assert.True",
        .assert_false => "Assert.False",
        .expect_true => "Expect.True",
        .expect_false => "Expect.False",
        .expect_equal_int, .expect_equal_bool => "Expect.Equal",
    };
}

fn newTestModule() !semantics.SemanticModule {
    return semantics.SemanticModule.init(std.testing.allocator);
}

fn intern(module: *semantics.SemanticModule, name: []const u8) !hir.SymbolId {
    return module.interner.intern(name);
}

fn addFunction(module: *semantics.SemanticModule, name: []const u8) !hir.FunctionId {
    return module.hir.addFunction(try intern(module, name), module.types.voidType(), .{ .start = 0, .length = name.len });
}

fn setAttributes(module: *semantics.SemanticModule, function_id: hir.FunctionId, names: []const []const u8) !void {
    const attributes = try std.testing.allocator.alloc(hir.HirAttribute, names.len);
    for (names, 0..) |name, index| {
        attributes[index] = .{
            .name = try intern(module, name),
            .args = &.{},
            .has_arguments = false,
            .span = .{ .start = index, .length = name.len },
        };
    }
    module.hir.setFunctionAttributes(function_id, attributes);
}

fn setTheoryInlineData(module: *semantics.SemanticModule, function_id: hir.FunctionId) !void {
    const args = try std.testing.allocator.alloc(hir.HirAttributeArg, 1);
    args[0] = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") };
    const attributes = try std.testing.allocator.alloc(hir.HirAttribute, 2);
    attributes[0] = .{ .name = try intern(module, "Theory"), .args = &.{}, .has_arguments = false, .span = .{ .start = 0, .length = 6 } };
    attributes[1] = .{ .name = try intern(module, "InlineData"), .args = args, .has_arguments = true, .span = .{ .start = 8, .length = 10 } };
    module.hir.setFunctionAttributes(function_id, attributes);
}

fn setBody(module: *semantics.SemanticModule, function_id: hir.FunctionId, stmts: []const hir.StmtId) !void {
    const owned = try std.testing.allocator.dupe(hir.StmtId, stmts);
    module.hir.setFunctionBody(function_id, try module.hir.addStmt(.{ .block = owned }, hir.synthetic_span));
}

fn exprStmt(module: *semantics.SemanticModule, expr: hir.ExprId) !hir.StmtId {
    return module.hir.addStmt(.{ .expr_stmt = expr }, hir.synthetic_span);
}

fn intExpr(module: *semantics.SemanticModule, text: []const u8) !hir.ExprId {
    return module.hir.addExpr(.{ .int_literal = try std.testing.allocator.dupe(u8, text) }, hir.synthetic_span);
}

fn boolExpr(module: *semantics.SemanticModule, value: bool) !hir.ExprId {
    return module.hir.addExpr(.{ .bool_literal = value }, hir.synthetic_span);
}

fn binaryExpr(module: *semantics.SemanticModule, op: hir.BinaryOp, left: hir.ExprId, right: hir.ExprId) !hir.ExprId {
    return module.hir.addExpr(.{ .binary = .{ .op = op, .left = left, .right = right } }, hir.synthetic_span);
}

fn unaryExpr(module: *semantics.SemanticModule, op: hir.UnaryOp, operand: hir.ExprId) !hir.ExprId {
    return module.hir.addExpr(.{ .unary = .{ .op = op, .operand = operand } }, hir.synthetic_span);
}

fn intrinsic(module: *semantics.SemanticModule, kind: hir.HirTestIntrinsicKind, operands: []const hir.ExprId, reason: []const u8) !hir.ExprId {
    return module.hir.addExpr(.{
        .test_intrinsic = .{
            .kind = kind,
            .operands = try std.testing.allocator.dupe(hir.ExprId, operands),
            .reason = try std.testing.allocator.dupe(u8, reason),
            .reason_span = hir.synthetic_span,
        },
    }, hir.synthetic_span);
}

test "Fact runner executes passing Assert and Expect intrinsics" {
    var module = try newTestModule();
    defer module.deinit();
    const fact = try addFunction(&module, "Passes");
    try setAttributes(&module, fact, &.{"Fact"});

    const sum = try binaryExpr(&module, .add, try intExpr(&module, "2"), try intExpr(&module, "2"));
    const not_false = try unaryExpr(&module, .logical_not, try boolExpr(&module, false));
    try setBody(&module, fact, &.{
        try exprStmt(&module, try intrinsic(&module, .assert_true, &.{try boolExpr(&module, true)}, "true should pass")),
        try exprStmt(&module, try intrinsic(&module, .assert_false, &.{try boolExpr(&module, false)}, "false should pass")),
        try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try binaryExpr(&module, .less, try intExpr(&module, "1"), try intExpr(&module, "2"))}, "comparison should be true")),
        try exprStmt(&module, try intrinsic(&module, .expect_false, &.{try binaryExpr(&module, .less, try intExpr(&module, "2"), try intExpr(&module, "1"))}, "comparison should be false")),
        try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try intExpr(&module, "4"), sum }, "integer equality should compare evaluated operands")),
        try exprStmt(&module, try intrinsic(&module, .expect_equal_bool, &.{ try boolExpr(&module, true), not_false }, "boolean equality should compare evaluated operands")),
    });

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failures.len);
}

test "Fact runner discovers only Facts and does not execute helpers or Theory" {
    var module = try newTestModule();
    defer module.deinit();

    const helper = try addFunction(&module, "HelperNotRun");
    try setBody(&module, helper, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, false)}, "helper should not be an entrypoint"))});

    const theory = try addFunction(&module, "TheoryNotRun");
    _ = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineData(&module, theory);
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, false)}, "theory execution is deferred"))});

    const fact = try addFunction(&module, "FactRuns");
    try setAttributes(&module, fact, &.{"Fact"});
    try setBody(&module, fact, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "fact should run"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
}

test "Fact runner records failing True and False reasons" {
    var true_module = try newTestModule();
    defer true_module.deinit();
    const true_fact = try addFunction(&true_module, "Fails");
    try setAttributes(&true_module, true_fact, &.{"Fact"});
    try setBody(&true_module, true_fact, &.{try exprStmt(&true_module, try intrinsic(&true_module, .assert_true, &.{try boolExpr(&true_module, false)}, "setup condition should hold"))});
    const true_result = try runModule(std.testing.allocator, &true_module, "Test");
    defer true_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), true_result.passed_count);
    try std.testing.expectEqual(@as(usize, 1), true_result.failed_count);
    try std.testing.expectEqual(hir.HirTestIntrinsicKind.assert_true, true_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("setup condition should hold", true_result.failures[0].reason);
    try std.testing.expectEqualStrings("true", true_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("false", true_result.failures[0].actual.?);

    var false_module = try newTestModule();
    defer false_module.deinit();
    const false_fact = try addFunction(&false_module, "Fails");
    try setAttributes(&false_module, false_fact, &.{"Fact"});
    try setBody(&false_module, false_fact, &.{try exprStmt(&false_module, try intrinsic(&false_module, .expect_false, &.{try boolExpr(&false_module, true)}, "condition should stay false"))});
    const false_result = try runModule(std.testing.allocator, &false_module, "Test");
    defer false_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_false, false_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("false", false_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("true", false_result.failures[0].actual.?);

    var assert_false_module = try newTestModule();
    defer assert_false_module.deinit();
    const assert_false_fact = try addFunction(&assert_false_module, "Fails");
    try setAttributes(&assert_false_module, assert_false_fact, &.{"Fact"});
    try setBody(&assert_false_module, assert_false_fact, &.{try exprStmt(&assert_false_module, try intrinsic(&assert_false_module, .assert_false, &.{try boolExpr(&assert_false_module, true)}, "assert false should fail on true"))});
    const assert_false_result = try runModule(std.testing.allocator, &assert_false_module, "Test");
    defer assert_false_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.assert_false, assert_false_result.failures[0].intrinsic_kind);
    try std.testing.expectEqual(hir.synthetic_span, assert_false_result.failures[0].source_span);
    try std.testing.expectEqualStrings("assert false should fail on true", assert_false_result.failures[0].reason);
}

test "Fact runner records Equal int and bool expected actual values" {
    var int_module = try newTestModule();
    defer int_module.deinit();
    const int_fact = try addFunction(&int_module, "Fails");
    try setAttributes(&int_module, int_fact, &.{"Fact"});
    try setBody(&int_module, int_fact, &.{try exprStmt(&int_module, try intrinsic(&int_module, .expect_equal_int, &.{ try intExpr(&int_module, "4"), try intExpr(&int_module, "5") }, "sum should match expected arithmetic result"))});
    const int_result = try runModule(std.testing.allocator, &int_module, "Test");
    defer int_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_equal_int, int_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("4", int_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("5", int_result.failures[0].actual.?);

    var bool_module = try newTestModule();
    defer bool_module.deinit();
    const bool_fact = try addFunction(&bool_module, "Fails");
    try setAttributes(&bool_module, bool_fact, &.{"Fact"});
    try setBody(&bool_module, bool_fact, &.{try exprStmt(&bool_module, try intrinsic(&bool_module, .expect_equal_bool, &.{ try boolExpr(&bool_module, true), try boolExpr(&bool_module, false) }, "boolean result should match expected branch"))});
    const bool_result = try runModule(std.testing.allocator, &bool_module, "Test");
    defer bool_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_equal_bool, bool_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("true", bool_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("false", bool_result.failures[0].actual.?);
}

test "Fact runner aggregates multiple Facts and aborts each failed Fact on first failure" {
    var module = try newTestModule();
    defer module.deinit();
    const failing = try addFunction(&module, "FailsOnce");
    try setAttributes(&module, failing, &.{"Fact"});
    try setBody(&module, failing, &.{
        try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, false)}, "first failure should fail this fact immediately")),
        try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, false)}, "second failure should not be reached in v0")),
    });

    const passing = try addFunction(&module, "Passes");
    try setAttributes(&module, passing, &.{"Fact"});
    try setBody(&module, passing, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "second fact should still run"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 1), result.failed_count);
    try std.testing.expectEqual(@as(usize, 1), result.failures.len);
    try std.testing.expectEqualStrings("first failure should fail this fact immediately", result.failures[0].reason);
}

test "Fact runner formats failure with reason and expected actual" {
    var module = try newTestModule();
    defer module.deinit();
    const fact = try addFunction(&module, "Fails");
    try setAttributes(&module, fact, &.{"Fact"});
    try setBody(&module, fact, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try intExpr(&module, "4"), try intExpr(&module, "5") }, "addition should return the arithmetic sum of both operands"))});
    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    const rendered = try formatFailure(std.testing.allocator, result.failures[0]);
    defer std.testing.allocator.free(rendered);

    try std.testing.expect(std.mem.indexOf(u8, rendered, "FAILED Test.Fails") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Because:\n  addition should return the arithmetic sum of both operands") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Expected:\n  4") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Actual:\n  5") != null);
}
