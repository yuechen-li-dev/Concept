const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");

pub const DiagnosticBag = diagnostics.DiagnosticBag;
pub const CheckError = error{InvalidSemanticModule} || std.mem.Allocator.Error;

const synthetic_span = hir.synthetic_span;

pub fn checkExecutable(
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    diagnostic_bag: ?*DiagnosticBag,
) CheckError!void {
    var checker = Checker{ .allocator = allocator, .module = module, .diagnostics = diagnostic_bag };
    try checker.checkModule();
}

const Checker = struct {
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    diagnostics: ?*DiagnosticBag,

    fn checkModule(self: *Checker) CheckError!void {
        const main_id = self.findMain() orelse {
            try self.report(.MissingMain, "expected top-level 'main' function");
            return error.InvalidSemanticModule;
        };
        const main = self.module.hir.getFunction(main_id);
        if (!sameType(main.return_type, self.module.types.intType())) {
            try self.reportAt(.InvalidMainSignature, "main must return int", main.span);
            return error.InvalidSemanticModule;
        }
        if (main.params.len != 0) {
            try self.reportAt(.InvalidMainSignature, "main must not have parameters", main.span);
            return error.InvalidSemanticModule;
        }
        if (main.body == null) {
            try self.reportAt(.InvalidMainSignature, "main must have a body", main.span);
            return error.InvalidSemanticModule;
        }

        for (self.module.hir.functions.items, 0..) |function, index| {
            if (function.body) |body| {
                try self.checkStmt(.{ .index = @intCast(index) }, body, function.return_type);
            }
        }
    }

    fn findMain(self: *Checker) ?hir.FunctionId {
        for (self.module.hir.functions.items, 0..) |function, index| {
            if (std.mem.eql(u8, self.module.interner.text(function.name), "main")) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn checkStmt(self: *Checker, function_id: hir.FunctionId, stmt_id: hir.StmtId, return_type: types.TypeId) CheckError!void {
        const stmt = self.module.hir.getStmt(stmt_id).*;
        switch (stmt.kind) {
            .block => |stmts| for (stmts) |child| try self.checkStmt(function_id, child, return_type),
            .return_stmt => |maybe_value| {
                const value = maybe_value orelse {
                    try self.reportAt(.TypeMismatch, "return statements must have an expression", stmt.span);
                    return error.InvalidSemanticModule;
                };
                const value_type = try self.checkExpr(value);
                try self.requireSame(value_type, return_type, "return expression type does not match function return type", self.exprSpan(value));
            },
            .local_decl => |decl| {
                const local = self.module.hir.getLocal(decl.local);
                const init_type = try self.checkExpr(decl.initializer);
                try self.requireSame(init_type, local.type_id, "local initializer type does not match declared type", self.exprSpan(decl.initializer));
            },
            .assignment => |assignment| {
                const target_type = switch (assignment.target) {
                    .local => |id| self.module.hir.getLocal(id).type_id,
                    .param => |id| self.module.hir.getParam(id).type_id,
                };
                const value_type = try self.checkExpr(assignment.value);
                try self.requireSame(value_type, target_type, "assignment expression type does not match target type", self.exprSpan(assignment.value));
            },
            .if_stmt => |if_stmt| {
                const condition_type = try self.checkExpr(if_stmt.condition);
                try self.requireBool(condition_type, "if condition must be bool", self.exprSpan(if_stmt.condition));
                try self.checkStmt(function_id, if_stmt.then_block, return_type);
                if (if_stmt.else_block) |else_block| try self.checkStmt(function_id, else_block, return_type);
            },
            .while_stmt => |while_stmt| {
                const condition_type = try self.checkExpr(while_stmt.condition);
                try self.requireBool(condition_type, "while condition must be bool", self.exprSpan(while_stmt.condition));
                try self.checkStmt(function_id, while_stmt.body, return_type);
            },
            .match_stmt => |match_stmt| {
                const scrutinee_type = try self.checkExpr(match_stmt.scrutinee);
                if (!self.isInt(scrutinee_type) and !self.isBool(scrutinee_type)) {
                    try self.reportAt(.TypeMismatch, "match scrutinee must be int or bool", self.exprSpan(match_stmt.scrutinee));
                    return error.InvalidSemanticModule;
                }
                var seen_ints = std.StringHashMap(void).init(self.allocator);
                defer seen_ints.deinit();
                var seen_true = false;
                var seen_false = false;
                var seen_wildcard = false;
                for (match_stmt.arms) |arm| {
                    switch (arm.pattern) {
                        .int_literal => |text| {
                            try self.requireSame(self.module.types.intType(), scrutinee_type, "match pattern type does not match scrutinee type", arm.pattern_span);
                            if (seen_ints.contains(text)) {
                                try self.reportAt(.TypeMismatch, "duplicate literal match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            try seen_ints.put(text, {});
                        },
                        .bool_literal => |value| {
                            try self.requireSame(self.module.types.boolType(), scrutinee_type, "match pattern type does not match scrutinee type", arm.pattern_span);
                            if ((value and seen_true) or (!value and seen_false)) {
                                try self.reportAt(.TypeMismatch, "duplicate literal match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            if (value) seen_true = true else seen_false = true;
                        },
                        .wildcard => {
                            if (seen_wildcard) {
                                try self.reportAt(.TypeMismatch, "duplicate wildcard match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            seen_wildcard = true;
                        },
                    }
                    try self.checkStmt(function_id, arm.body, return_type);
                }
            },
        }
    }

    fn checkExpr(self: *Checker, expr_id: hir.ExprId) CheckError!types.TypeId {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => self.module.types.intType(),
            .bool_literal => self.module.types.boolType(),
            .local_ref => |id| self.module.hir.getLocal(id).type_id,
            .param_ref => |id| self.module.hir.getParam(id).type_id,
            .group => |inner| try self.checkExpr(inner),
            .call => |call| blk: {
                const callee = self.module.hir.getFunction(call.function);
                if (callee.body == null) {
                    try self.reportAt(.InvalidCall, "cannot call function without body", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (call.args.len != callee.params.len) {
                    try self.reportAt(.InvalidCall, "function call argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (call.args, callee.params) |arg, param_id| {
                    const arg_type = try self.checkExpr(arg);
                    const param_type = self.module.hir.getParam(param_id).type_id;
                    try self.requireCallSame(arg_type, param_type, "function call argument type mismatch", self.exprSpan(arg));
                }
                break :blk callee.return_type;
            },
            .enum_constructor => |constructor| blk: {
                const variant = self.module.hir.getVariant(constructor.variant_id);
                if (variant.parent.index != constructor.enum_id.index or constructor.args.len != variant.payload_fields.len) {
                    try self.reportAt(.EnumConstructorArityMismatch, "enum constructor argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (constructor.args, variant.payload_fields) |arg, payload_id| {
                    const arg_type = try self.checkExpr(arg);
                    const payload_type = self.module.hir.getEnumPayloadField(payload_id).type_id;
                    if (!sameType(arg_type, payload_type)) {
                        try self.reportAt(.EnumConstructorTypeMismatch, "enum constructor argument type mismatch", self.exprSpan(arg));
                        return error.InvalidSemanticModule;
                    }
                }
                break :blk try self.enumType(constructor.enum_id);
            },
            .unary => |unary| blk: {
                const operand_type = try self.checkExpr(unary.operand);
                switch (unary.op) {
                    .negate => {
                        try self.requireInt(operand_type, "arithmetic unary operator requires int operand", expr.span);
                        break :blk self.module.types.intType();
                    },
                    .logical_not => {
                        try self.requireBool(operand_type, "logical unary operator requires bool operand", expr.span);
                        break :blk self.module.types.boolType();
                    },
                }
            },
            .binary => |binary| blk: {
                const left_type = try self.checkExpr(binary.left);
                const right_type = try self.checkExpr(binary.right);
                switch (binary.op) {
                    .add, .subtract, .multiply, .divide, .modulo => {
                        try self.requireIntPair(left_type, right_type, "arithmetic binary operator requires int operands", expr.span);
                        break :blk self.module.types.intType();
                    },
                    .less, .less_equal, .greater, .greater_equal => {
                        try self.requireIntPair(left_type, right_type, "comparison operator requires int operands", expr.span);
                        break :blk self.module.types.boolType();
                    },
                    .equal_equal, .bang_equal => {
                        if (!sameType(left_type, right_type) or (!self.isInt(left_type) and !self.isBool(left_type))) {
                            try self.reportAt(.TypeMismatch, "equality operator requires matching int or bool operands", expr.span);
                            return error.InvalidSemanticModule;
                        }
                        break :blk self.module.types.boolType();
                    },
                    .logical_and, .logical_or => {
                        try self.requireBoolPair(left_type, right_type, "logical binary operator requires bool operands", expr.span);
                        break :blk self.module.types.boolType();
                    },
                }
            },
        };
    }

    fn enumType(self: *Checker, enum_id: hir.EnumId) CheckError!types.TypeId {
        for (self.module.types.types.items, 0..) |kind, index| {
            if (kind == .enum_type and kind.enum_type.index == enum_id.index) return .{ .index = @intCast(index) };
        }
        try self.reportAt(.TypeMismatch, "unknown enum constructor type", synthetic_span);
        return error.InvalidSemanticModule;
    }

    fn requireSame(self: *Checker, actual: types.TypeId, expected: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!sameType(actual, expected)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireCallSame(self: *Checker, actual: types.TypeId, expected: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!sameType(actual, expected)) {
            try self.reportAt(.InvalidCall, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireInt(self: *Checker, actual: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isInt(actual)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireBool(self: *Checker, actual: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isBool(actual)) {
            try self.reportAt(.InvalidConditionType, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireIntPair(self: *Checker, left: types.TypeId, right: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isInt(left) or !self.isInt(right)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireBoolPair(self: *Checker, left: types.TypeId, right: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isBool(left) or !self.isBool(right)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn isInt(self: *Checker, actual: types.TypeId) bool {
        return sameType(actual, self.module.types.intType());
    }

    fn isBool(self: *Checker, actual: types.TypeId) bool {
        return sameType(actual, self.module.types.boolType());
    }

    fn exprSpan(self: *Checker, expr_id: hir.ExprId) diagnostics.SourceSpan {
        return self.module.hir.getExpr(expr_id).span;
    }

    fn report(self: *Checker, code: diagnostics.DiagnosticCode, message: []const u8) !void {
        try self.reportAt(code, message, synthetic_span);
    }

    fn reportAt(self: *Checker, code: diagnostics.DiagnosticCode, message: []const u8, span: diagnostics.SourceSpan) !void {
        if (self.diagnostics) |bag| try bag.append(diagnostics.makeDiagnostic(code, .@"error", message, span));
    }
};

fn sameType(a: types.TypeId, b: types.TypeId) bool {
    return a.index == b.index;
}

const TestModule = struct {
    module: semantics.SemanticModule,

    fn init() !TestModule {
        return .{ .module = try semantics.SemanticModule.init(std.testing.allocator) };
    }

    fn deinit(self: *TestModule) void {
        self.module.deinit();
    }

    fn name(self: *TestModule, text: []const u8) !hir.SymbolId {
        return self.module.interner.intern(text);
    }

    fn function(self: *TestModule, name_text: []const u8, return_type: types.TypeId) !hir.FunctionId {
        return self.module.hir.addFunction(try self.name(name_text), return_type, synthetic_span);
    }

    fn param(self: *TestModule, function_id: hir.FunctionId, name_text: []const u8, type_id: types.TypeId) !hir.ParamId {
        return self.module.hir.addParam(function_id, try self.name(name_text), type_id, synthetic_span);
    }

    fn local(self: *TestModule, function_id: hir.FunctionId, name_text: []const u8, type_id: types.TypeId) !hir.LocalId {
        return self.module.hir.addLocal(function_id, try self.name(name_text), type_id, synthetic_span);
    }

    fn block(self: *TestModule, stmts: []const hir.StmtId) !hir.StmtId {
        const owned = try std.testing.allocator.dupe(hir.StmtId, stmts);
        return self.module.hir.addStmt(.{ .block = owned }, synthetic_span);
    }

    fn setBody(self: *TestModule, function_id: hir.FunctionId, stmt_id: hir.StmtId) void {
        self.module.hir.setFunctionBody(function_id, stmt_id);
    }

    fn int(self: *TestModule, text: []const u8) !hir.ExprId {
        return self.module.hir.addExpr(.{ .int_literal = try std.testing.allocator.dupe(u8, text) }, synthetic_span);
    }

    fn boolLit(self: *TestModule, value: bool) !hir.ExprId {
        return self.module.hir.addExpr(.{ .bool_literal = value }, synthetic_span);
    }

    fn ret(self: *TestModule, expr: ?hir.ExprId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .return_stmt = expr }, synthetic_span);
    }

    fn checkPass(self: *TestModule) !void {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try checkExecutable(std.testing.allocator, &self.module, &bag);
        try std.testing.expectEqual(@as(usize, 0), bag.count());
    }

    fn checkFail(self: *TestModule, expected: diagnostics.DiagnosticCode) !void {
        _ = try self.checkFailSpan(expected);
    }

    fn checkFailSpan(self: *TestModule, expected: diagnostics.DiagnosticCode) !diagnostics.SourceSpan {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try std.testing.expectError(error.InvalidSemanticModule, checkExecutable(std.testing.allocator, &self.module, &bag));
        try std.testing.expectEqual(@as(usize, 1), bag.count());
        try std.testing.expectEqual(expected, bag.diagnostics.items[0].code);
        return bag.diagnostics.items[0].primary_span;
    }
};

fn testSpan(start: usize) diagnostics.SourceSpan {
    return .{ .start = start, .length = 1 };
}

fn addTestStmtAt(store: *hir.HirStore, kind: hir.HirStmtKind, span: diagnostics.SourceSpan) !hir.StmtId {
    return store.addStmt(kind, span);
}

fn addTestExprAt(store: *hir.HirStore, kind: hir.HirExprKind, span: diagnostics.SourceSpan) !hir.ExprId {
    return store.addExpr(kind, span);
}

fn addTestStmt(store: *hir.HirStore, kind: hir.HirStmtKind) !hir.StmtId {
    return store.addStmt(kind, synthetic_span);
}

fn addTestExpr(store: *hir.HirStore, kind: hir.HirExprKind) !hir.ExprId {
    return store.addExpr(kind, synthetic_span);
}

fn addMainReturnInt(tm: *TestModule, value: []const u8) !hir.FunctionId {
    const main_id = try tm.function("main", tm.module.types.intType());
    const ret_stmt = try tm.ret(try tm.int(value));
    tm.setBody(main_id, try tm.block(&.{ret_stmt}));
    return main_id;
}

test "HIR checker accepts return int main" {
    var tm = try TestModule.init();
    defer tm.deinit();
    _ = try addMainReturnInt(&tm, "0");
    try tm.checkPass();
}

test "HIR checker accepts locals and assignments" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const b = try tm.local(main_id, "b", tm.module.types.boolType());
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("1") } });
    const init_b = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = b, .initializer = try tm.boolLit(true) } });
    const add = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .add, .left = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .right = try tm.int("2") } });
    const assign_x = try addTestStmt(&tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = add } });
    const ret_stmt = try tm.ret(try addTestExpr(&tm.module.hir, .{ .local_ref = x }));
    tm.setBody(main_id, try tm.block(&.{ init_x, init_b, assign_x, ret_stmt }));
    try tm.checkPass();
}

test "HIR checker accepts function calls" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const add_id = try tm.function("add", tm.module.types.intType());
    const a = try tm.param(add_id, "a", tm.module.types.intType());
    const b = try tm.param(add_id, "b", tm.module.types.intType());
    const sum = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .add, .left = try addTestExpr(&tm.module.hir, .{ .param_ref = a }), .right = try addTestExpr(&tm.module.hir, .{ .param_ref = b }) } });
    tm.setBody(add_id, try tm.block(&.{try tm.ret(sum)}));
    const main_id = try tm.function("main", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{ try tm.int("1"), try tm.int("2") });
    tm.setBody(main_id, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .call = .{ .function = add_id, .args = args } }))}));
    try tm.checkPass();
}

test "HIR checker accepts if match while" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("0") } });
    const while_body = try tm.block(&.{try addTestStmt(&tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = try tm.int("1") } })});
    const while_stmt = try addTestStmt(&tm.module.hir, .{ .while_stmt = .{ .condition = try tm.boolLit(false), .body = while_body } });
    const match_body = try addTestStmt(&tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = try tm.int("2") } });
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{.{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = match_body }});
    const match_stmt = try addTestStmt(&tm.module.hir, .{ .match_stmt = .{ .scrutinee = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .arms = arms } });
    const then_block = try tm.block(&.{match_stmt});
    const if_stmt = try addTestStmt(&tm.module.hir, .{ .if_stmt = .{ .condition = try tm.boolLit(true), .then_block = then_block, .else_block = null } });
    const ret_stmt = try tm.ret(try addTestExpr(&tm.module.hir, .{ .local_ref = x }));
    tm.setBody(main_id, try tm.block(&.{ init_x, while_stmt, if_stmt, ret_stmt }));
    try tm.checkPass();
}

test "HIR checker rejects missing main" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const helper = try tm.function("helper", tm.module.types.intType());
    tm.setBody(helper, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    try tm.checkFail(.MissingMain);
}

test "HIR checker rejects bad main return type and params" {
    var bad_return = try TestModule.init();
    defer bad_return.deinit();
    const main_bool = try bad_return.function("main", bad_return.module.types.boolType());
    bad_return.setBody(main_bool, try bad_return.block(&.{try bad_return.ret(try bad_return.boolLit(true))}));
    try bad_return.checkFail(.InvalidMainSignature);

    var bad_params = try TestModule.init();
    defer bad_params.deinit();
    const main_param = try bad_params.function("main", bad_params.module.types.intType());
    _ = try bad_params.param(main_param, "argc", bad_params.module.types.intType());
    bad_params.setBody(main_param, try bad_params.block(&.{try bad_params.ret(try bad_params.int("0"))}));
    try bad_params.checkFail(.InvalidMainSignature);
}

test "HIR checker rejects return local and assignment mismatches" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    tm.setBody(main_id, try tm.block(&.{try tm.ret(try tm.boolLit(true))}));
    try tm.checkFail(.TypeMismatch);

    var local_tm = try TestModule.init();
    defer local_tm.deinit();
    const local_main = try local_tm.function("main", local_tm.module.types.intType());
    const x = try local_tm.local(local_main, "x", local_tm.module.types.intType());
    local_tm.setBody(local_main, try local_tm.block(&.{try addTestStmt(&local_tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try local_tm.boolLit(false) } })}));
    try local_tm.checkFail(.TypeMismatch);

    var assign_tm = try TestModule.init();
    defer assign_tm.deinit();
    const assign_main = try assign_tm.function("main", assign_tm.module.types.intType());
    const y = try assign_tm.local(assign_main, "y", assign_tm.module.types.intType());
    const decl_y = try addTestStmt(&assign_tm.module.hir, .{ .local_decl = .{ .local = y, .initializer = try assign_tm.int("0") } });
    const assign_y = try addTestStmt(&assign_tm.module.hir, .{ .assignment = .{ .target = .{ .local = y }, .value = try assign_tm.boolLit(true) } });
    assign_tm.setBody(assign_main, try assign_tm.block(&.{ decl_y, assign_y }));
    try assign_tm.checkFail(.TypeMismatch);
}

test "HIR checker rejects bad conditions calls and match patterns" {
    var if_tm = try TestModule.init();
    defer if_tm.deinit();
    const if_main = try if_tm.function("main", if_tm.module.types.intType());
    if_tm.setBody(if_main, try if_tm.block(&.{try addTestStmt(&if_tm.module.hir, .{ .if_stmt = .{ .condition = try if_tm.int("1"), .then_block = try if_tm.block(&.{}), .else_block = null } })}));
    try if_tm.checkFail(.InvalidConditionType);

    var while_tm = try TestModule.init();
    defer while_tm.deinit();
    const while_main = try while_tm.function("main", while_tm.module.types.intType());
    while_tm.setBody(while_main, try while_tm.block(&.{try addTestStmt(&while_tm.module.hir, .{ .while_stmt = .{ .condition = try while_tm.int("1"), .body = try while_tm.block(&.{}) } })}));
    try while_tm.checkFail(.InvalidConditionType);

    var call_count = try TestModule.init();
    defer call_count.deinit();
    const callee = try call_count.function("callee", call_count.module.types.intType());
    _ = try call_count.param(callee, "x", call_count.module.types.intType());
    call_count.setBody(callee, try call_count.block(&.{try call_count.ret(try call_count.int("0"))}));
    const call_main = try call_count.function("main", call_count.module.types.intType());
    const no_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    call_count.setBody(call_main, try call_count.block(&.{try call_count.ret(try addTestExpr(&call_count.module.hir, .{ .call = .{ .function = callee, .args = no_args } }))}));
    try call_count.checkFail(.InvalidCall);

    var call_type = try TestModule.init();
    defer call_type.deinit();
    const id = try call_type.function("id", call_type.module.types.intType());
    _ = try call_type.param(id, "x", call_type.module.types.intType());
    call_type.setBody(id, try call_type.block(&.{try call_type.ret(try call_type.int("0"))}));
    const main = try call_type.function("main", call_type.module.types.intType());
    const bad_args = try std.testing.allocator.dupe(hir.ExprId, &.{try call_type.boolLit(false)});
    call_type.setBody(main, try call_type.block(&.{try call_type.ret(try addTestExpr(&call_type.module.hir, .{ .call = .{ .function = id, .args = bad_args } }))}));
    try call_type.checkFail(.InvalidCall);

    var match_tm = try TestModule.init();
    defer match_tm.deinit();
    const match_main = try match_tm.function("main", match_tm.module.types.intType());
    const body = try match_tm.block(&.{});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{.{ .pattern = .{ .bool_literal = true }, .pattern_span = synthetic_span, .body = body }});
    match_tm.setBody(match_main, try match_tm.block(&.{try addTestStmt(&match_tm.module.hir, .{ .match_stmt = .{ .scrutinee = try match_tm.int("0"), .arms = arms } })}));
    try match_tm.checkFail(.TypeMismatch);
}

test "HIR checker rejects duplicate match pattern and wildcard" {
    var dup_lit = try TestModule.init();
    defer dup_lit.deinit();
    const lit_main = try dup_lit.function("main", dup_lit.module.types.intType());
    const body = try dup_lit.block(&.{});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{ .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = body }, .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = body } });
    dup_lit.setBody(lit_main, try dup_lit.block(&.{try addTestStmt(&dup_lit.module.hir, .{ .match_stmt = .{ .scrutinee = try dup_lit.int("0"), .arms = arms } })}));
    try dup_lit.checkFail(.TypeMismatch);

    var dup_wild = try TestModule.init();
    defer dup_wild.deinit();
    const wild_main = try dup_wild.function("main", dup_wild.module.types.intType());
    const wild_body = try dup_wild.block(&.{});
    const wild_arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{ .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = wild_body }, .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = wild_body } });
    dup_wild.setBody(wild_main, try dup_wild.block(&.{try addTestStmt(&dup_wild.module.hir, .{ .match_stmt = .{ .scrutinee = try dup_wild.boolLit(false), .arms = wild_arms } })}));
    try dup_wild.checkFail(.TypeMismatch);
}

test "HIR checker diagnostics use HIR source spans" {
    var if_tm = try TestModule.init();
    defer if_tm.deinit();
    const if_main = try if_tm.function("main", if_tm.module.types.intType());
    const if_cond = try addTestExprAt(&if_tm.module.hir, .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, testSpan(11));
    if_tm.setBody(if_main, try if_tm.block(&.{try addTestStmtAt(&if_tm.module.hir, .{ .if_stmt = .{ .condition = if_cond, .then_block = try if_tm.block(&.{}), .else_block = null } }, testSpan(10))}));
    try std.testing.expectEqual(testSpan(11), try if_tm.checkFailSpan(.InvalidConditionType));

    var assign_tm = try TestModule.init();
    defer assign_tm.deinit();
    const assign_main = try assign_tm.function("main", assign_tm.module.types.intType());
    const x = try assign_tm.local(assign_main, "x", assign_tm.module.types.intType());
    const decl_x = try addTestStmt(&assign_tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try assign_tm.int("0") } });
    const bad_value = try addTestExprAt(&assign_tm.module.hir, .{ .bool_literal = true }, testSpan(22));
    const assign_x = try addTestStmtAt(&assign_tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = bad_value } }, testSpan(20));
    assign_tm.setBody(assign_main, try assign_tm.block(&.{ decl_x, assign_x }));
    try std.testing.expectEqual(testSpan(22), try assign_tm.checkFailSpan(.TypeMismatch));

    var ret_tm = try TestModule.init();
    defer ret_tm.deinit();
    const ret_main = try ret_tm.function("main", ret_tm.module.types.intType());
    const bad_ret = try addTestExprAt(&ret_tm.module.hir, .{ .bool_literal = false }, testSpan(33));
    ret_tm.setBody(ret_main, try ret_tm.block(&.{try addTestStmtAt(&ret_tm.module.hir, .{ .return_stmt = bad_ret }, testSpan(30))}));
    try std.testing.expectEqual(testSpan(33), try ret_tm.checkFailSpan(.TypeMismatch));

    var call_tm = try TestModule.init();
    defer call_tm.deinit();
    const callee = try call_tm.function("id", call_tm.module.types.intType());
    _ = try call_tm.param(callee, "x", call_tm.module.types.intType());
    call_tm.setBody(callee, try call_tm.block(&.{try call_tm.ret(try call_tm.int("0"))}));
    const call_main = try call_tm.function("main", call_tm.module.types.intType());
    const bad_arg = try addTestExprAt(&call_tm.module.hir, .{ .bool_literal = false }, testSpan(44));
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{bad_arg});
    const bad_call = try addTestExprAt(&call_tm.module.hir, .{ .call = .{ .function = callee, .args = args } }, testSpan(40));
    call_tm.setBody(call_main, try call_tm.block(&.{try call_tm.ret(bad_call)}));
    try std.testing.expectEqual(testSpan(44), try call_tm.checkFailSpan(.InvalidCall));

    var match_tm = try TestModule.init();
    defer match_tm.deinit();
    const match_main = try match_tm.function("main", match_tm.module.types.intType());
    const arm_body = try match_tm.block(&.{});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{.{ .pattern = .{ .bool_literal = true }, .pattern_span = testSpan(55), .body = arm_body }});
    match_tm.setBody(match_main, try match_tm.block(&.{try addTestStmtAt(&match_tm.module.hir, .{ .match_stmt = .{ .scrutinee = try match_tm.int("0"), .arms = arms } }, testSpan(50))}));
    try std.testing.expectEqual(testSpan(55), try match_tm.checkFailSpan(.TypeMismatch));
}

test "HIR checker accepts Phase 2-style lowered program" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("4") } });
    const then_body = try tm.block(&.{try tm.ret(try tm.int("9"))});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{ .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "4") }, .pattern_span = synthetic_span, .body = then_body }, .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = try tm.block(&.{try tm.ret(try tm.int("1"))}) } });
    const match_stmt = try addTestStmt(&tm.module.hir, .{ .match_stmt = .{ .scrutinee = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .arms = arms } });
    const condition = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .greater_equal, .left = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .right = try tm.int("4") } });
    const if_stmt = try addTestStmt(&tm.module.hir, .{ .if_stmt = .{ .condition = condition, .then_block = try tm.block(&.{match_stmt}), .else_block = try tm.block(&.{try tm.ret(try tm.int("0"))}) } });
    tm.setBody(main_id, try tm.block(&.{ decl, if_stmt }));
    try tm.checkPass();
}
