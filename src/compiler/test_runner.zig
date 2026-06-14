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

pub const TestFailureKind = enum {
    assertion_invariant,
    expectation,
};

pub const TestFailure = struct {
    module_name: []const u8,
    function_name: []const u8,
    inline_data_row_index: ?usize = null,
    source_span: source.SourceSpan,
    intrinsic_kind: hir.HirTestIntrinsicKind,
    failure_kind: TestFailureKind,
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
    total_count: usize = 0,
    passed_count: usize = 0,
    failed_count: usize = 0,
    skipped_count: usize = 0,
    failures: []TestFailure = &.{},

    pub fn deinit(self: TestRunResult, allocator: std.mem.Allocator) void {
        for (self.failures) |failure| failure.deinit(allocator);
        if (self.failures.len > 0) allocator.free(self.failures);
    }
};

pub fn formatResult(allocator: std.mem.Allocator, result: TestRunResult) ![]const u8 {
    var output = std.Io.Writer.Allocating.init(allocator);
    errdefer output.deinit();
    const writer = &output.writer;

    if (result.failed_count == 0) {
        try writer.print("PASS tests={d} failures=0\n", .{result.total_count});
    } else {
        try writer.print("FAIL tests={d} failures={d}\n", .{ result.total_count, result.failed_count });
        for (result.failures, 0..) |failure, index| {
            try writer.writeByte('\n');
            const rendered = try formatFailure(allocator, failure);
            defer allocator.free(rendered);
            try writer.writeAll(rendered);
            if (index + 1 < result.failures.len) try writer.writeByte('\n');
        }
    }

    return output.toOwnedSlice();
}

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
    const display_name = try testCaseName(allocator, failure.module_name, failure.function_name, failure.inline_data_row_index);
    defer allocator.free(display_name);

    try writer.print(
        \\FAILED {s}
        \\
        \\{s} failed
        \\
        \\Source:
        \\  span start={d} length={d}
        \\
        \\Because:
        \\  {s}
        \\
    , .{
        display_name,
        failureMessageName(failure.intrinsic_kind),
        failure.source_span.start,
        failure.source_span.length,
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
    current_inline_data_row_index: ?usize = null,

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
            result.total_count += 1;
            self.current_module_name = test_case.module_name;
            self.current_function_name = test_case.function_name;
            self.current_inline_data_row_index = test_case.inline_data_row_index;

            const function_id = self.findFunction(test_case.function_name) orelse return error.UnsupportedConstruct;
            const function = self.module.hir.getFunction(function_id);
            const args = switch (test_case.attribute_kind) {
                .fact => &.{},
                .theory => try self.inlineDataArgs(function, test_case.inline_data_args),
            };
            defer if (test_case.attribute_kind == .theory) self.allocator.free(args);

            const before_failures = self.failures.items.len;
            _ = self.executeFunction(function_id, args) catch |err| switch (err) {
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

    fn inlineDataArgs(self: *Runner, function: *const hir.HirFunction, row_args: []const hir.HirAttributeArg) RunError![]Value {
        if (row_args.len != function.params.len) return error.InvalidValue;

        const args = try self.allocator.alloc(Value, row_args.len);
        errdefer self.allocator.free(args);
        for (row_args, function.params, 0..) |row_arg, param_id, index| {
            const param = self.module.hir.getParam(param_id);
            args[index] = try self.inlineDataValue(row_arg, param.type_id);
        }
        return args;
    }

    fn inlineDataValue(self: *Runner, row_arg: hir.HirAttributeArg, param_type: anytype) RunError!Value {
        return switch (row_arg) {
            .identifier => return error.InvalidValue,
            .int_literal => |text| {
                if (param_type.index != self.module.types.intType().index) return error.InvalidValue;
                return .{ .int = std.fmt.parseInt(i64, text, 10) catch return error.InvalidValue };
            },
            .bool_literal => |value| {
                if (param_type.index != self.module.types.boolType().index) return error.InvalidValue;
                return .{ .bool = value };
            },
            .string_literal => error.InvalidValue,
        };
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
            .panic_stmt, .assert_stmt => error.UnsupportedConstruct,
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
            .transition_stmt, .arena_reset, .arena_destroy, .match_stmt => error.UnsupportedConstruct,
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
            .machine_param_ref,
            .machine_construct,
            .machine_step,
            .machine_complete,
            .machine_result,
            .address_of,
            .deref,
            .dyn_coerce,
            .move_expr,
            .manual_init_assume,
            .arena_alloc,
            .try_expr,
            .concept_requirement_call,
            .interface_call,
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
            .assert_true, .expect_true, .expect_that_true => {
                const actual = try self.evaluateExpr(test_intrinsic.operands[0], frame);
                const actual_bool = try valueAsBool(actual);
                if (!actual_bool) try self.recordFailure(span, test_intrinsic, .{ .bool = true }, actual);
            },
            .assert_false, .expect_false, .expect_that_false => {
                const actual = try self.evaluateExpr(test_intrinsic.operands[0], frame);
                const actual_bool = try valueAsBool(actual);
                if (actual_bool) try self.recordFailure(span, test_intrinsic, .{ .bool = false }, actual);
            },
            .expect_equal_int, .expect_equal_bool, .expect_that_equal_int, .expect_that_equal_bool => {
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
            .inline_data_row_index = self.current_inline_data_row_index,
            .source_span = span,
            .intrinsic_kind = test_intrinsic.kind,
            .failure_kind = failureKind(test_intrinsic.kind),
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

fn testCaseName(allocator: std.mem.Allocator, module_name: []const u8, function_name: []const u8, inline_data_row_index: ?usize) ![]const u8 {
    if (inline_data_row_index) |row_index| {
        return std.fmt.allocPrint(allocator, "{s}.{s}#{d}", .{ module_name, function_name, row_index });
    }
    return std.fmt.allocPrint(allocator, "{s}.{s}", .{ module_name, function_name });
}

fn failureKind(kind: hir.HirTestIntrinsicKind) TestFailureKind {
    return switch (kind) {
        .assert_true, .assert_false => .assertion_invariant,
        .expect_true, .expect_false, .expect_equal_int, .expect_equal_bool, .expect_that_true, .expect_that_false, .expect_that_equal_int, .expect_that_equal_bool => .expectation,
    };
}

fn failureMessageName(kind: hir.HirTestIntrinsicKind) []const u8 {
    return switch (kind) {
        .assert_true => "Assert.True",
        .assert_false => "Assert.False",
        .expect_true => "Expect.True",
        .expect_false => "Expect.False",
        .expect_equal_int, .expect_equal_bool => "Expect.Equal",
        .expect_that_true, .expect_that_false, .expect_that_equal_int, .expect_that_equal_bool => "Expect.That",
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

fn setTheoryInlineDataRows(module: *semantics.SemanticModule, function_id: hir.FunctionId, rows: []const []const hir.HirAttributeArg) !void {
    const attributes = try std.testing.allocator.alloc(hir.HirAttribute, rows.len + 1);
    attributes[0] = .{ .name = try intern(module, "Theory"), .args = &.{}, .has_arguments = false, .span = .{ .start = 0, .length = 6 } };
    for (rows, 0..) |row, index| {
        const args = try std.testing.allocator.alloc(hir.HirAttributeArg, row.len);
        for (row, 0..) |arg, arg_index| {
            args[arg_index] = switch (arg) {
                .identifier => |text| .{ .identifier = try std.testing.allocator.dupe(u8, text) },
                .int_literal => |text| .{ .int_literal = try std.testing.allocator.dupe(u8, text) },
                .bool_literal => |value| .{ .bool_literal = value },
                .string_literal => |text| .{ .string_literal = try std.testing.allocator.dupe(u8, text) },
            };
        }
        attributes[index + 1] = .{ .name = try intern(module, "InlineData"), .args = args, .has_arguments = true, .span = .{ .start = 8 + index, .length = 10 } };
    }
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

fn paramExpr(module: *semantics.SemanticModule, param: hir.ParamId) !hir.ExprId {
    return module.hir.addExpr(.{ .param_ref = param }, hir.synthetic_span);
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

fn callExpr(module: *semantics.SemanticModule, function_id: hir.FunctionId, args: []const hir.ExprId) !hir.ExprId {
    return module.hir.addExpr(.{
        .call = .{
            .function = function_id,
            .args = try std.testing.allocator.dupe(hir.ExprId, args),
        },
    }, hir.synthetic_span);
}

fn returnStmt(module: *semantics.SemanticModule, maybe_expr: ?hir.ExprId) !hir.StmtId {
    return module.hir.addStmt(.{ .return_stmt = maybe_expr }, hir.synthetic_span);
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

test "Fact runner executes passing Expect.That relation intrinsics" {
    var module = try newTestModule();
    defer module.deinit();
    const fact = try addFunction(&module, "RelationPasses");
    try setAttributes(&module, fact, &.{"Fact"});

    const sum = try binaryExpr(&module, .add, try intExpr(&module, "2"), try intExpr(&module, "2"));
    const not_false = try unaryExpr(&module, .logical_not, try boolExpr(&module, false));
    try setBody(&module, fact, &.{
        try exprStmt(&module, try intrinsic(&module, .expect_that_true, &.{try boolExpr(&module, true)}, "true relation should pass")),
        try exprStmt(&module, try intrinsic(&module, .expect_that_false, &.{try boolExpr(&module, false)}, "false relation should pass")),
        try exprStmt(&module, try intrinsic(&module, .expect_that_equal_int, &.{ try intExpr(&module, "4"), sum }, "integer EqualTo relation should compare evaluated operands")),
        try exprStmt(&module, try intrinsic(&module, .expect_that_equal_bool, &.{ try boolExpr(&module, true), not_false }, "boolean EqualTo relation should compare evaluated operands")),
    });

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failures.len);
}

test "runner discovers facts and theories but does not execute helpers directly" {
    var module = try newTestModule();
    defer module.deinit();

    const helper = try addFunction(&module, "HelperNotRun");
    try setBody(&module, helper, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, false)}, "helper should not be an entrypoint"))});

    const theory = try addFunction(&module, "TheoryRuns");
    _ = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{&.{.{ .int_literal = "1" }}});
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "theory row should run"))});

    const fact = try addFunction(&module, "FactRuns");
    try setAttributes(&module, fact, &.{"Fact"});
    try setBody(&module, fact, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "fact should run"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
}

test "Theory runner executes one passing int row" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "BindsInt");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{&.{.{ .int_literal = "7" }}});
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try intExpr(&module, "7"), try paramExpr(&module, value) }, "InlineData int should initialize the theory parameter"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
}

test "Theory runner executes multiple passing rows as separate cases" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "MultipleRows");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .int_literal = "1" }}, &.{.{ .int_literal = "2" }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try binaryExpr(&module, .greater, try paramExpr(&module, value), try intExpr(&module, "0"))}, "each InlineData row should be positive"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
}

test "Theory runner records failing row reason and row index" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "FailsRow");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{&.{.{ .int_literal = "5" }}});
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try intExpr(&module, "6"), try paramExpr(&module, value) }, "the current row should bind the expected integer value"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), result.passed_count);
    try std.testing.expectEqual(@as(usize, 1), result.failed_count);
    try std.testing.expectEqual(@as(?usize, 0), result.failures[0].inline_data_row_index);
    try std.testing.expectEqualStrings("the current row should bind the expected integer value", result.failures[0].reason);
    try std.testing.expectEqualStrings("6", result.failures[0].expected.?);
    try std.testing.expectEqualStrings("5", result.failures[0].actual.?);

    const rendered = try formatFailure(std.testing.allocator, result.failures[0]);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "FAILED Test.FailsRow#0") != null);
}

test "Theory runner aggregates multiple rows with one failure" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "OneBadRow");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .int_literal = "1" }}, &.{.{ .int_literal = "-1" }}, &.{.{ .int_literal = "2" }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try binaryExpr(&module, .greater, try paramExpr(&module, value), try intExpr(&module, "0"))}, "theory rows should continue after one row fails"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), result.passed_count);
    try std.testing.expectEqual(@as(usize, 1), result.failed_count);
    try std.testing.expectEqual(@as(?usize, 1), result.failures[0].inline_data_row_index);
}

test "Theory runner binds bool parameters" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "BindsBool");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.boolType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .bool_literal = true }}, &.{.{ .bool_literal = false }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_bool, &.{ try paramExpr(&module, value), try paramExpr(&module, value) }, "InlineData bool should initialize the theory parameter"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
}

test "Theory runner can call helper functions without discovering them directly" {
    var module = try newTestModule();
    defer module.deinit();
    const helper = try addFunction(&module, "AddOne");
    const helper_value = try module.hir.addParam(helper, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    module.hir.setFunctionReturnType(helper, module.types.intType());
    try setBody(&module, helper, &.{try returnStmt(&module, try binaryExpr(&module, .add, try paramExpr(&module, helper_value), try intExpr(&module, "1")))});

    const theory = try addFunction(&module, "UsesHelper");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    const expected = try module.hir.addParam(theory, try intern(&module, "expected"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{&.{ .{ .int_literal = "2" }, .{ .int_literal = "3" } }});
    const call = try callExpr(&module, helper, &.{try paramExpr(&module, value)});
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try paramExpr(&module, expected), call }, "helper result should be available inside theory rows"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
}

test "runner aggregates Facts and Theory rows in the same test module" {
    var module = try newTestModule();
    defer module.deinit();
    const fact = try addFunction(&module, "FactPasses");
    try setAttributes(&module, fact, &.{"Fact"});
    try setBody(&module, fact, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "fact should pass"))});

    const theory = try addFunction(&module, "TheoryRows");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .int_literal = "1" }}, &.{.{ .int_literal = "2" }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try binaryExpr(&module, .greater, try paramExpr(&module, value), try intExpr(&module, "0"))}, "theory row should pass"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 3), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
    try std.testing.expectEqual(@as(usize, 3), result.total_count);
}

test "runner formats Fact pass summary" {
    var module = try newTestModule();
    defer module.deinit();
    const fact = try addFunction(&module, "Passes");
    try setAttributes(&module, fact, &.{"Fact"});
    try setBody(&module, fact, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "fact should pass"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    const rendered = try formatResult(std.testing.allocator, result);
    defer std.testing.allocator.free(rendered);
    try std.testing.expectEqualStrings("PASS tests=1 failures=0\n", rendered);
}

test "runner formats Theory pass summary" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "RowsPass");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .int_literal = "1" }}, &.{.{ .int_literal = "2" }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try binaryExpr(&module, .greater, try paramExpr(&module, value), try intExpr(&module, "0"))}, "theory row should pass"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    const rendered = try formatResult(std.testing.allocator, result);
    defer std.testing.allocator.free(rendered);
    try std.testing.expectEqualStrings("PASS tests=2 failures=0\n", rendered);
}

test "runner formats mixed Fact and Theory pass summary" {
    var module = try newTestModule();
    defer module.deinit();
    const fact = try addFunction(&module, "FactPasses");
    try setAttributes(&module, fact, &.{"Fact"});
    try setBody(&module, fact, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "fact should pass"))});

    const theory = try addFunction(&module, "RowsPass");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .int_literal = "1" }}, &.{.{ .int_literal = "2" }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try paramExpr(&module, value), try paramExpr(&module, value) }, "theory value should equal itself"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    const rendered = try formatResult(std.testing.allocator, result);
    defer std.testing.allocator.free(rendered);
    try std.testing.expectEqualStrings("PASS tests=3 failures=0\n", rendered);
}

test "Theory runner defensively rejects malformed InlineData rows" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "MalformedRow");
    _ = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{&.{}});
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_true, &.{try boolExpr(&module, true)}, "malformed rows should not execute"))});

    try std.testing.expectError(error.InvalidValue, runModule(std.testing.allocator, &module, "Test"));
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
    try std.testing.expectEqual(TestFailureKind.assertion_invariant, true_result.failures[0].failure_kind);
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
    try std.testing.expectEqual(TestFailureKind.expectation, false_result.failures[0].failure_kind);
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

test "Fact runner records Expect.That failures with relation-shaped names and values" {
    var true_module = try newTestModule();
    defer true_module.deinit();
    const true_fact = try addFunction(&true_module, "FailsTrue");
    try setAttributes(&true_module, true_fact, &.{"Fact"});
    try setBody(&true_module, true_fact, &.{try exprStmt(&true_module, try intrinsic(&true_module, .expect_that_true, &.{try boolExpr(&true_module, false)}, "relation true should fail on false"))});
    const true_result = try runModule(std.testing.allocator, &true_module, "Test");
    defer true_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_that_true, true_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("relation true should fail on false", true_result.failures[0].reason);
    try std.testing.expectEqualStrings("true", true_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("false", true_result.failures[0].actual.?);
    const rendered = try formatFailure(std.testing.allocator, true_result.failures[0]);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Expect.That failed") != null);

    var false_module = try newTestModule();
    defer false_module.deinit();
    const false_fact = try addFunction(&false_module, "FailsFalse");
    try setAttributes(&false_module, false_fact, &.{"Fact"});
    try setBody(&false_module, false_fact, &.{try exprStmt(&false_module, try intrinsic(&false_module, .expect_that_false, &.{try boolExpr(&false_module, true)}, "relation false should fail on true"))});
    const false_result = try runModule(std.testing.allocator, &false_module, "Test");
    defer false_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_that_false, false_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("false", false_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("true", false_result.failures[0].actual.?);

    var int_module = try newTestModule();
    defer int_module.deinit();
    const int_fact = try addFunction(&int_module, "FailsIntEqual");
    try setAttributes(&int_module, int_fact, &.{"Fact"});
    try setBody(&int_module, int_fact, &.{try exprStmt(&int_module, try intrinsic(&int_module, .expect_that_equal_int, &.{ try intExpr(&int_module, "4"), try intExpr(&int_module, "5") }, "integer EqualTo relation should report values"))});
    const int_result = try runModule(std.testing.allocator, &int_module, "Test");
    defer int_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_that_equal_int, int_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("4", int_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("5", int_result.failures[0].actual.?);

    var bool_module = try newTestModule();
    defer bool_module.deinit();
    const bool_fact = try addFunction(&bool_module, "FailsBoolEqual");
    try setAttributes(&bool_module, bool_fact, &.{"Fact"});
    try setBody(&bool_module, bool_fact, &.{try exprStmt(&bool_module, try intrinsic(&bool_module, .expect_that_equal_bool, &.{ try boolExpr(&bool_module, true), try boolExpr(&bool_module, false) }, "boolean EqualTo relation should report values"))});
    const bool_result = try runModule(std.testing.allocator, &bool_module, "Test");
    defer bool_result.deinit(std.testing.allocator);

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_that_equal_bool, bool_result.failures[0].intrinsic_kind);
    try std.testing.expectEqualStrings("true", bool_result.failures[0].expected.?);
    try std.testing.expectEqualStrings("false", bool_result.failures[0].actual.?);
}

test "Theory runner executes Expect.That relation intrinsics" {
    var module = try newTestModule();
    defer module.deinit();
    const theory = try addFunction(&module, "RelationTheoryRows");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    const expected = try module.hir.addParam(theory, try intern(&module, "expected"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{ .{ .int_literal = "1" }, .{ .int_literal = "1" } }, &.{ .{ .int_literal = "2" }, .{ .int_literal = "2" } } });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_that_equal_int, &.{ try paramExpr(&module, expected), try paramExpr(&module, value) }, "Expect.That should run for each theory row"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), result.passed_count);
    try std.testing.expectEqual(@as(usize, 0), result.failed_count);
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
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Source:\n  span start=0 length=0") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Because:\n  addition should return the arithmetic sum of both operands") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Expected:\n  4") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Actual:\n  5") != null);
}

test "runner formats failure summary with ordered failure blocks" {
    var module = try newTestModule();
    defer module.deinit();
    const first = try addFunction(&module, "FirstFails");
    try setAttributes(&module, first, &.{"Fact"});
    try setBody(&module, first, &.{try exprStmt(&module, try intrinsic(&module, .expect_equal_int, &.{ try intExpr(&module, "10"), try intExpr(&module, "9") }, "first failure should appear first"))});

    const theory = try addFunction(&module, "TheoryFails");
    const value = try module.hir.addParam(theory, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    try setTheoryInlineDataRows(&module, theory, &.{ &.{.{ .int_literal = "1" }}, &.{.{ .int_literal = "-1" }} });
    try setBody(&module, theory, &.{try exprStmt(&module, try intrinsic(&module, .expect_that_true, &.{try binaryExpr(&module, .greater, try paramExpr(&module, value), try intExpr(&module, "0"))}, "row value should be positive"))});

    const result = try runModule(std.testing.allocator, &module, "Test");
    defer result.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 3), result.total_count);
    try std.testing.expectEqual(@as(usize, 1), result.passed_count);
    try std.testing.expectEqual(@as(usize, 2), result.failed_count);
    try std.testing.expectEqualStrings("FirstFails", result.failures[0].function_name);
    try std.testing.expectEqualStrings("TheoryFails", result.failures[1].function_name);
    try std.testing.expectEqual(@as(?usize, 1), result.failures[1].inline_data_row_index);

    const rendered = try formatResult(std.testing.allocator, result);
    defer std.testing.allocator.free(rendered);

    try std.testing.expect(std.mem.startsWith(u8, rendered, "FAIL tests=3 failures=2\n\nFAILED Test.FirstFails"));
    const first_index = std.mem.indexOf(u8, rendered, "FAILED Test.FirstFails").?;
    const second_index = std.mem.indexOf(u8, rendered, "FAILED Test.TheoryFails#1").?;
    try std.testing.expect(first_index < second_index);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Expect.That failed") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Because:\n  row value should be positive") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Expected:\n  true") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Actual:\n  false") != null);
}
