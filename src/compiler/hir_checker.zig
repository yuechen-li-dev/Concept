// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Checker state
// ─────────────────────────────────────────────────────────────────────────────

const Checker = struct {
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    diagnostics: ?*DiagnosticBag,
    unsafe_depth: usize = 0,

    // ─────────────────────────────────────────────────────────────────────────────
    // Function/main validation
    // ─────────────────────────────────────────────────────────────────────────────

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
                if (function.is_unsafe) self.unsafe_depth += 1;
                defer {
                    if (function.is_unsafe) self.unsafe_depth -= 1;
                }
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

    // ─────────────────────────────────────────────────────────────────────────────
    // Statement checking
    // ─────────────────────────────────────────────────────────────────────────────

    fn checkStmt(self: *Checker, function_id: hir.FunctionId, stmt_id: hir.StmtId, return_type: types.TypeId) CheckError!void {
        const stmt = self.module.hir.getStmt(stmt_id).*;
        switch (stmt.kind) {
            .block => |stmts| for (stmts) |child| try self.checkStmt(function_id, child, return_type),
            .return_stmt => |maybe_value| {
                const value = maybe_value orelse {
                    try self.reportAt(.TypeMismatch, "return statements must have an expression", stmt.span);
                    return error.InvalidSemanticModule;
                };
                const value_type = try self.checkExpr(return_type, value);
                try self.requireSame(value_type, return_type, "return expression type does not match function return type", self.exprSpan(value));
            },
            .local_decl => |decl| {
                const local = self.module.hir.getLocal(decl.local);
                const init_type = try self.checkExpr(return_type, decl.initializer);
                try self.requireSame(init_type, local.type_id, "local initializer type does not match declared type", self.exprSpan(decl.initializer));
            },
            .expr_stmt => |expr_id| {
                const value_type = try self.checkExpr(return_type, expr_id);
                if (self.isMustUseType(value_type)) {
                    try self.reportAt(.IgnoredMustUseValue, "value must be used; use discard to ignore intentionally", self.exprSpan(expr_id));
                    return error.InvalidSemanticModule;
                }
            },
            .discard_stmt => |expr_id| {
                _ = try self.checkExpr(return_type, expr_id);
            },
            .assignment => |assignment| {
                const target_type = switch (assignment.target) {
                    .local => |id| self.module.hir.getLocal(id).type_id,
                    .param => |id| self.module.hir.getParam(id).type_id,
                };
                const value_type = try self.checkExpr(return_type, assignment.value);
                try self.requireSame(value_type, target_type, "assignment expression type does not match target type", self.exprSpan(assignment.value));
            },
            .if_stmt => |if_stmt| {
                const condition_type = try self.checkExpr(return_type, if_stmt.condition);
                try self.requireBool(condition_type, "if condition must be bool", self.exprSpan(if_stmt.condition));
                try self.checkStmt(function_id, if_stmt.then_block, return_type);
                if (if_stmt.else_block) |else_block| try self.checkStmt(function_id, else_block, return_type);
            },
            .while_stmt => |while_stmt| {
                const condition_type = try self.checkExpr(return_type, while_stmt.condition);
                try self.requireBool(condition_type, "while condition must be bool", self.exprSpan(while_stmt.condition));
                try self.checkStmt(function_id, while_stmt.body, return_type);
            },
            .unsafe_block => |body| {
                self.unsafe_depth += 1;
                defer self.unsafe_depth -= 1;
                try self.checkStmt(function_id, body, return_type);
            },
            .match_stmt => |match_stmt| {
                const scrutinee_type = try self.checkExpr(return_type, match_stmt.scrutinee);
                const scrutinee_kind = self.module.types.kind(scrutinee_type);
                if (!self.isInt(scrutinee_type) and !self.isBool(scrutinee_type) and scrutinee_kind != .enum_type) {
                    try self.reportAt(.TypeMismatch, "match scrutinee must be int, bool, or enum", self.exprSpan(match_stmt.scrutinee));
                    return error.InvalidSemanticModule;
                }
                var seen_ints = std.StringHashMap(void).init(self.allocator);
                defer seen_ints.deinit();
                var seen_variants = std.AutoHashMap(hir.VariantId, void).init(self.allocator);
                defer seen_variants.deinit();
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
                        .enum_variant => |pattern| {
                            if (scrutinee_kind != .enum_type or scrutinee_kind.enum_type.index != pattern.enum_id.index) {
                                try self.reportAt(.EnumPatternTypeMismatch, "enum pattern type does not match match scrutinee", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            if (seen_variants.contains(pattern.variant_id)) {
                                try self.reportAt(.EnumPatternTypeMismatch, "duplicate enum variant match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            try seen_variants.put(pattern.variant_id, {});
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

    // ─────────────────────────────────────────────────────────────────────────────
    // Expression checking
    // ─────────────────────────────────────────────────────────────────────────────

    fn checkExpr(self: *Checker, return_type: types.TypeId, expr_id: hir.ExprId) CheckError!types.TypeId {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => self.module.types.intType(),
            .bool_literal => self.module.types.boolType(),
            .local_ref => |id| self.module.hir.getLocal(id).type_id,
            .param_ref => |id| self.module.hir.getParam(id).type_id,
            .group => |inner| try self.checkExpr(return_type, inner),
            .call => |call| blk: {
                const callee = self.module.hir.getFunction(call.function);
                if (callee.body == null) {
                    try self.reportAt(.InvalidCall, "cannot call function without body", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (callee.is_unsafe and self.unsafe_depth == 0) {
                    try self.reportAt(.UnsafeCallRequiresUnsafe, "unsafe function call requires unsafe context", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (call.args.len != callee.params.len) {
                    try self.reportAt(.InvalidCall, "function call argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (call.args, callee.params) |arg, param_id| {
                    const arg_type = try self.checkExpr(return_type, arg);
                    const param_type = self.module.hir.getParam(param_id).type_id;
                    try self.requireCallSame(arg_type, param_type, "function call argument type mismatch", self.exprSpan(arg));
                }
                break :blk callee.return_type;
            },
            // ─────────────────────────────────────────────────────────────────────────────
            // Match/enum checking
            // ─────────────────────────────────────────────────────────────────────────────

            .enum_constructor => |constructor| blk: {
                const variant = self.module.hir.getVariant(constructor.variant_id);
                if (variant.parent.index != constructor.enum_id.index or constructor.args.len != variant.payload_fields.len) {
                    try self.reportAt(.EnumConstructorArityMismatch, "enum constructor argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (constructor.args, variant.payload_fields) |arg, payload_id| {
                    const arg_type = try self.checkExpr(return_type, arg);
                    const payload_type = self.module.hir.getEnumPayloadField(payload_id).type_id;
                    if (!sameType(arg_type, payload_type)) {
                        try self.reportAt(.EnumConstructorTypeMismatch, "enum constructor argument type mismatch", self.exprSpan(arg));
                        return error.InvalidSemanticModule;
                    }
                }
                break :blk try self.enumType(constructor.enum_id);
            },
            // ─────────────────────────────────────────────────────────────────────────────
            // Decide checking
            // ─────────────────────────────────────────────────────────────────────────────

            .decide => |decide| blk: {
                if (!sameType(decide.enum_type, try self.enumType(decide.enum_id))) {
                    try self.reportAt(.UnknownDecideEnum, "decide target type must be an enum", expr.span);
                    return error.InvalidSemanticModule;
                }
                var has_unconditional = false;
                for (decide.arms) |arm| {
                    const variant = self.module.hir.getVariant(arm.variant_id);
                    if (variant.parent.index != decide.enum_id.index) {
                        try self.reportAt(.UnknownDecideVariant, "unknown decide variant", arm.span);
                        return error.InvalidSemanticModule;
                    }
                    if (variant.payload_fields.len != 0) {
                        try self.reportAt(.DecideVariantHasPayload, "decide candidate variant must not have payload fields", arm.span);
                        return error.InvalidSemanticModule;
                    }
                    if (arm.condition) |condition| {
                        const condition_type = try self.checkExpr(return_type, condition);
                        if (!self.isBool(condition_type)) {
                            try self.reportAt(.DecideConditionNotBool, "decide arm condition must be bool", self.exprSpan(condition));
                            return error.InvalidSemanticModule;
                        }
                    } else {
                        has_unconditional = true;
                    }
                    const score_type = try self.checkExpr(return_type, arm.score);
                    if (!self.isInt(score_type)) {
                        try self.reportAt(.DecideScoreNotInt, "decide arm score must be int", self.exprSpan(arm.score));
                        return error.InvalidSemanticModule;
                    }
                }
                if (!has_unconditional) {
                    try self.reportAt(.DecideMissingUnconditionalArm, "decide expression requires at least one unconditional arm", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk decide.enum_type;
            },
            .unary => |unary| blk: {
                const operand_type = try self.checkExpr(return_type, unary.operand);
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
            .address_of => |operand| blk: {
                const place_type = try self.addressableExprType(operand, expr.span);
                break :blk try self.module.types.addPointerType(place_type);
            },
            .deref => |operand| blk: {
                const operand_type = try self.checkExpr(return_type, operand);
                const pointee = switch (self.module.types.kind(operand_type)) {
                    .pointer => |pointer| pointer.pointee,
                    else => {
                        try self.reportAt(.DerefRequiresPointer, "dereference requires raw pointer operand", expr.span);
                        return error.InvalidSemanticModule;
                    },
                };
                if (self.unsafe_depth == 0) {
                    try self.reportAt(.DerefRequiresUnsafe, "raw pointer dereference requires unsafe context", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk pointee;
            },
            // ─────────────────────────────────────────────────────────────────────────────
            // Result/try checking
            // ─────────────────────────────────────────────────────────────────────────────

            .try_expr => |operand| blk: {
                const operand_type = try self.checkExpr(return_type, operand);
                const operand_shape = self.module.resultShapeForType(operand_type) orelse {
                    try self.reportAt(.TryOperandNotResult, "try operand must be a Result-shaped enum", expr.span);
                    return error.InvalidSemanticModule;
                };
                if (self.module.resultShapeForType(return_type) == null) {
                    try self.reportAt(.TryOutsideResultFunction, "try requires the enclosing function to return a Result-shaped enum", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (!sameType(operand_type, return_type)) {
                    try self.reportAt(.TryResultTypeMismatch, "try operand Result type must match the enclosing function return type", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk operand_shape.ok_type;
            },
            .binary => |binary| blk: {
                const left_type = try self.checkExpr(return_type, binary.left);
                const right_type = try self.checkExpr(return_type, binary.right);
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

    // ─────────────────────────────────────────────────────────────────────────────
    // Type helper functions
    // ─────────────────────────────────────────────────────────────────────────────


    fn addressableExprType(self: *Checker, expr_id: hir.ExprId, span: diagnostics.SourceSpan) CheckError!types.TypeId {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| self.module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.module.hir.getParam(param_id).type_id,
            .group => |inner| try self.addressableExprType(inner, span),
            else => {
                try self.reportAt(.AddressOfRequiresPlace, "address-of requires a local or parameter place", span);
                return error.InvalidSemanticModule;
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

    fn isMustUseType(self: *Checker, actual: types.TypeId) bool {
        const kind = self.module.types.kind(actual);
        if (kind != .enum_type) return false;
        return self.module.hir.getEnum(kind.enum_type).is_must_use;
    }

    fn exprSpan(self: *Checker, expr_id: hir.ExprId) diagnostics.SourceSpan {
        return self.module.hir.getExpr(expr_id).span;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Diagnostic helpers
    // ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

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

    fn unsafeFunction(self: *TestModule, name_text: []const u8, return_type: types.TypeId) !hir.FunctionId {
        return self.module.hir.addFunctionWithSafety(try self.name(name_text), return_type, true, synthetic_span);
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

test "HIR checker accepts pointer return local copy and call argument" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const int_ptr = try tm.module.types.addPointerType(tm.module.types.intType());

    const identity_id = try tm.function("identity", int_ptr);
    const p = try tm.param(identity_id, "p", int_ptr);
    const q = try tm.local(identity_id, "q", int_ptr);
    const init_q = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = q, .initializer = try addTestExpr(&tm.module.hir, .{ .param_ref = p }) } });
    const ret_q = try tm.ret(try addTestExpr(&tm.module.hir, .{ .local_ref = q }));
    tm.setBody(identity_id, try tm.block(&.{ init_q, ret_q }));

    const use_id = try tm.function("use", tm.module.types.intType());
    const input = try tm.param(use_id, "input", int_ptr);
    const output = try tm.local(use_id, "output", int_ptr);
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{try addTestExpr(&tm.module.hir, .{ .param_ref = input })});
    const init_output = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = output, .initializer = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = identity_id, .args = args } }) } });
    tm.setBody(use_id, try tm.block(&.{ init_output, try tm.ret(try tm.int("0")) }));
    _ = try addMainReturnInt(&tm, "0");

    try tm.checkPass();
}



test "HIR checker accepts address-of local and param" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const int_ptr = try tm.module.types.addPointerType(tm.module.types.intType());
    const helper_id = try tm.function("helper", int_ptr);
    const param_id = try tm.param(helper_id, "p", tm.module.types.intType());
    tm.setBody(helper_id, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .address_of = try addTestExpr(&tm.module.hir, .{ .param_ref = param_id }) }))}));

    const main_id = try tm.function("main", tm.module.types.intType());
    const local_id = try tm.local(main_id, "x", tm.module.types.intType());
    const ptr_local = try tm.local(main_id, "p", int_ptr);
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local_id, .initializer = try tm.int("1") } });
    const addr = try addTestExpr(&tm.module.hir, .{ .address_of = try addTestExpr(&tm.module.hir, .{ .local_ref = local_id }) });
    const init_p = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = ptr_local, .initializer = addr } });
    tm.setBody(main_id, try tm.block(&.{ init_x, init_p, try tm.ret(try tm.int("0")) }));
    try tm.checkPass();
}

test "HIR checker accepts deref inside unsafe contexts" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const int_ptr = try tm.module.types.addPointerType(tm.module.types.intType());
    const read_id = try tm.unsafeFunction("read", tm.module.types.intType());
    const param_id = try tm.param(read_id, "p", int_ptr);
    const deref_param = try addTestExpr(&tm.module.hir, .{ .deref = try addTestExpr(&tm.module.hir, .{ .param_ref = param_id }) });
    tm.setBody(read_id, try tm.block(&.{try tm.ret(deref_param)}));

    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const p_local = try tm.local(main_id, "p", int_ptr);
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("1") } });
    const init_p = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = p_local, .initializer = try addTestExpr(&tm.module.hir, .{ .address_of = try addTestExpr(&tm.module.hir, .{ .local_ref = x }) }) } });
    const deref_local = try addTestExpr(&tm.module.hir, .{ .deref = try addTestExpr(&tm.module.hir, .{ .local_ref = p_local }) });
    const unsafe_body = try tm.block(&.{try tm.ret(deref_local)});
    const unsafe_stmt = try addTestStmt(&tm.module.hir, .{ .unsafe_block = unsafe_body });
    tm.setBody(main_id, try tm.block(&.{ init_x, init_p, unsafe_stmt }));
    try tm.checkPass();
}

test "HIR checker rejects invalid pointer operations" {
    var address_tm = try TestModule.init();
    defer address_tm.deinit();
    const int_ptr = try address_tm.module.types.addPointerType(address_tm.module.types.intType());
    const main_a = try address_tm.function("main", address_tm.module.types.intType());
    const ptr_local = try address_tm.local(main_a, "p", int_ptr);
    const bad_addr = try addTestExpr(&address_tm.module.hir, .{ .address_of = try address_tm.int("1") });
    const init_p = try addTestStmt(&address_tm.module.hir, .{ .local_decl = .{ .local = ptr_local, .initializer = bad_addr } });
    address_tm.setBody(main_a, try address_tm.block(&.{ init_p, try address_tm.ret(try address_tm.int("0")) }));
    try address_tm.checkFail(.AddressOfRequiresPlace);

    var nonptr_tm = try TestModule.init();
    defer nonptr_tm.deinit();
    const main_b = try nonptr_tm.function("main", nonptr_tm.module.types.intType());
    const bad_deref = try addTestExpr(&nonptr_tm.module.hir, .{ .deref = try nonptr_tm.int("1") });
    nonptr_tm.setBody(main_b, try nonptr_tm.block(&.{try nonptr_tm.ret(bad_deref)}));
    try nonptr_tm.checkFail(.DerefRequiresPointer);

    var unsafe_tm = try TestModule.init();
    defer unsafe_tm.deinit();
    const unsafe_ptr = try unsafe_tm.module.types.addPointerType(unsafe_tm.module.types.intType());
    const main_c = try unsafe_tm.function("main", unsafe_tm.module.types.intType());
    const p = try unsafe_tm.local(main_c, "p", unsafe_ptr);
    const bad_unsafe = try addTestExpr(&unsafe_tm.module.hir, .{ .deref = try addTestExpr(&unsafe_tm.module.hir, .{ .local_ref = p }) });
    const init_ptr = try addTestStmt(&unsafe_tm.module.hir, .{ .local_decl = .{ .local = p, .initializer = try addTestExpr(&unsafe_tm.module.hir, .{ .address_of = try addTestExpr(&unsafe_tm.module.hir, .{ .local_ref = try unsafe_tm.local(main_c, "x", unsafe_tm.module.types.intType()) }) }) } });
    unsafe_tm.setBody(main_c, try unsafe_tm.block(&.{ init_ptr, try unsafe_tm.ret(bad_unsafe) }));
    try unsafe_tm.checkFail(.DerefRequiresUnsafe);
}

test "HIR checker rejects pointer type mismatches" {
    var return_mismatch = try TestModule.init();
    defer return_mismatch.deinit();
    const int_ptr = try return_mismatch.module.types.addPointerType(return_mismatch.module.types.intType());
    const bool_ptr = try return_mismatch.module.types.addPointerType(return_mismatch.module.types.boolType());
    const bad_return = try return_mismatch.function("bad", int_ptr);
    const bp = try return_mismatch.param(bad_return, "bp", bool_ptr);
    return_mismatch.setBody(bad_return, try return_mismatch.block(&.{try return_mismatch.ret(try addTestExpr(&return_mismatch.module.hir, .{ .param_ref = bp }))}));
    _ = try addMainReturnInt(&return_mismatch, "0");
    try return_mismatch.checkFail(.TypeMismatch);

    var local_mismatch = try TestModule.init();
    defer local_mismatch.deinit();
    const local_int_ptr = try local_mismatch.module.types.addPointerType(local_mismatch.module.types.intType());
    const local_bool_ptr = try local_mismatch.module.types.addPointerType(local_mismatch.module.types.boolType());
    const local_helper = try local_mismatch.function("localMismatch", local_mismatch.module.types.intType());
    const source = try local_mismatch.param(local_helper, "source", local_bool_ptr);
    const target = try local_mismatch.local(local_helper, "target", local_int_ptr);
    const bad_init = try addTestStmt(&local_mismatch.module.hir, .{ .local_decl = .{ .local = target, .initializer = try addTestExpr(&local_mismatch.module.hir, .{ .param_ref = source }) } });
    local_mismatch.setBody(local_helper, try local_mismatch.block(&.{ bad_init, try local_mismatch.ret(try local_mismatch.int("0")) }));
    _ = try addMainReturnInt(&local_mismatch, "0");
    try local_mismatch.checkFail(.TypeMismatch);

    var call_mismatch = try TestModule.init();
    defer call_mismatch.deinit();
    const call_int_ptr = try call_mismatch.module.types.addPointerType(call_mismatch.module.types.intType());
    const call_bool_ptr = try call_mismatch.module.types.addPointerType(call_mismatch.module.types.boolType());
    const callee = try call_mismatch.function("takesIntPtr", call_mismatch.module.types.intType());
    _ = try call_mismatch.param(callee, "p", call_int_ptr);
    call_mismatch.setBody(callee, try call_mismatch.block(&.{try call_mismatch.ret(try call_mismatch.int("1"))}));
    const caller = try call_mismatch.function("callMismatch", call_mismatch.module.types.intType());
    const bool_param = try call_mismatch.param(caller, "bp", call_bool_ptr);
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{try addTestExpr(&call_mismatch.module.hir, .{ .param_ref = bool_param })});
    call_mismatch.setBody(caller, try call_mismatch.block(&.{try call_mismatch.ret(try addTestExpr(&call_mismatch.module.hir, .{ .call = .{ .function = callee, .args = call_args } }))}));
    _ = try addMainReturnInt(&call_mismatch, "0");
    try call_mismatch.checkFail(.InvalidCall);
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

test "HIR checker accepts discard and explicit uses of must_use enum values" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const enum_id = try tm.module.hir.addEnum(try tm.name("ParseResult"), true);
    const enum_type = try tm.module.types.addEnumType(enum_id);
    const ok_variant = try tm.module.hir.addVariant(enum_id, try tm.name("Ok"), synthetic_span);
    _ = try tm.module.hir.addEnumPayloadField(ok_variant, try tm.name("value"), tm.module.types.intType(), synthetic_span);

    const make = try tm.function("make", enum_type);
    _ = try tm.module.hir.addParam(make, try tm.name("value"), tm.module.types.intType(), synthetic_span);
    const make_return_arg = try tm.int("1");
    const make_args = try std.testing.allocator.dupe(hir.ExprId, &.{make_return_arg});
    const make_value = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok_variant, .args = make_args } });
    const make_return = try tm.ret(make_value);
    tm.setBody(make, try tm.block(&.{make_return}));

    const main = try tm.function("main", tm.module.types.intType());
    const constructor_arg = try tm.int("2");
    const constructor_args = try std.testing.allocator.dupe(hir.ExprId, &.{constructor_arg});
    const constructor = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok_variant, .args = constructor_args } });
    const local = try tm.local(main, "result", enum_type);
    const local_decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local, .initializer = constructor } });
    const call_arg = try tm.int("3");
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{call_arg});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = make, .args = call_args } });
    const discard_call = try addTestStmt(&tm.module.hir, .{ .discard_stmt = call });
    const non_must_use = try tm.int("4");
    const discard_int = try addTestStmt(&tm.module.hir, .{ .discard_stmt = non_must_use });
    const ret = try tm.ret(try tm.int("0"));
    tm.setBody(main, try tm.block(&.{ local_decl, discard_call, discard_int, ret }));

    try tm.checkPass();
}

test "HIR checker rejects ignored must_use expression statement" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const enum_id = try tm.module.hir.addEnum(try tm.name("ParseResult"), true);
    _ = try tm.module.types.addEnumType(enum_id);
    const ok_variant = try tm.module.hir.addVariant(enum_id, try tm.name("Ok"), synthetic_span);

    const main = try tm.function("main", tm.module.types.intType());
    const empty_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const constructor = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok_variant, .args = empty_args } });
    const ignored = try addTestStmt(&tm.module.hir, .{ .expr_stmt = constructor });
    const ret = try tm.ret(try tm.int("0"));
    tm.setBody(main, try tm.block(&.{ ignored, ret }));

    try tm.checkFail(.IgnoredMustUseValue);
}

fn addResultEnum(tm: *TestModule, name: []const u8, must_use: bool) !struct { enum_id: hir.EnumId, type_id: types.TypeId, ok: hir.VariantId, err: hir.VariantId, ok_field: hir.EnumPayloadFieldId, err_field: hir.EnumPayloadFieldId } {
    const enum_id = try tm.module.hir.addEnum(try tm.name(name), must_use);
    const type_id = try tm.module.types.addEnumType(enum_id);
    const ok = try tm.module.hir.addVariant(enum_id, try tm.name("Ok"), synthetic_span);
    const ok_field = try tm.module.hir.addEnumPayloadField(ok, try tm.name("value"), tm.module.types.intType(), synthetic_span);
    const err = try tm.module.hir.addVariant(enum_id, try tm.name("Err"), synthetic_span);
    const err_field = try tm.module.hir.addEnumPayloadField(err, try tm.name("code"), tm.module.types.intType(), synthetic_span);
    tm.module.hir.setEnumResultShape(enum_id, .{ .ok_variant = ok, .err_variant = err, .ok_payload = ok_field, .err_payload = err_field, .ok_type = tm.module.types.intType(), .err_type = tm.module.types.intType() });
    return .{ .enum_id = enum_id, .type_id = type_id, .ok = ok, .err = err, .ok_field = ok_field, .err_field = err_field };
}

test "HIR checker accepts try over same Result-shaped enum" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    const parse = try tm.function("parse", result.type_id);
    const arg_expr = try tm.int("1");
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{arg_expr});
    const ok_expr = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = args } });
    tm.setBody(parse, try tm.block(&.{try tm.ret(ok_expr)}));
    const add = try tm.function("add", result.type_id);
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = parse, .args = call_args } });
    const tried = try addTestExpr(&tm.module.hir, .{ .try_expr = call });
    const local = try tm.local(add, "value", tm.module.types.intType());
    const decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local, .initializer = tried } });
    const local_ref = try addTestExpr(&tm.module.hir, .{ .local_ref = local });
    const ret_args = try std.testing.allocator.dupe(hir.ExprId, &.{local_ref});
    const ret_ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = ret_args } });
    tm.setBody(add, try tm.block(&.{ decl, try tm.ret(ret_ok) }));
    try tm.checkPass();
}

test "HIR checker rejects try operand that is not Result-shaped" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    const f = try tm.function("bad", result.type_id);
    tm.setBody(f, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .try_expr = try tm.int("1") }))}));
    try tm.checkFail(.TryOperandNotResult);
}

test "HIR checker rejects try in non-Result-returning function" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    const ok_arg = try tm.int("1");
    const ok_args = try std.testing.allocator.dupe(hir.ExprId, &.{ok_arg});
    const ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = ok_args } });
    const tried = try addTestExpr(&tm.module.hir, .{ .try_expr = ok });
    tm.setBody(main, try tm.block(&.{ try addTestStmt(&tm.module.hir, .{ .discard_stmt = tried }), try tm.ret(try tm.int("0")) }));
    try tm.checkFail(.TryOutsideResultFunction);
}

test "HIR checker rejects mismatched try Result type" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const other = try addResultEnum(&tm, "OtherResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    const f = try tm.function("bad", other.type_id);
    const ok_arg = try tm.int("1");
    const ok_args = try std.testing.allocator.dupe(hir.ExprId, &.{ok_arg});
    const ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = ok_args } });
    const tried = try addTestExpr(&tm.module.hir, .{ .try_expr = ok });
    const ret_arg = try tm.int("0");
    const ret_args = try std.testing.allocator.dupe(hir.ExprId, &.{ret_arg});
    const ret_ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = other.enum_id, .variant_id = other.ok, .args = ret_args } });
    tm.setBody(f, try tm.block(&.{ try addTestStmt(&tm.module.hir, .{ .discard_stmt = tried }), try tm.ret(ret_ok) }));
    try tm.checkFail(.TryResultTypeMismatch);
}

test "HIR checker accepts unsafe blocks" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const body = try tm.block(&.{try tm.ret(try tm.int("0"))});
    const unsafe_stmt = try addTestStmt(&tm.module.hir, .{ .unsafe_block = body });
    tm.setBody(main_id, try tm.block(&.{unsafe_stmt}));
    try tm.checkPass();
}

test "HIR checker enforces unsafe function calls" {
    var outside = try TestModule.init();
    defer outside.deinit();
    const helper = try outside.unsafeFunction("helper", outside.module.types.intType());
    outside.setBody(helper, try outside.block(&.{try outside.ret(try outside.int("1"))}));
    const outside_main = try outside.function("main", outside.module.types.intType());
    const outside_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    outside.setBody(outside_main, try outside.block(&.{try outside.ret(try addTestExpr(&outside.module.hir, .{ .call = .{ .function = helper, .args = outside_args } }))}));
    try outside.checkFail(.UnsafeCallRequiresUnsafe);

    var inside = try TestModule.init();
    defer inside.deinit();
    const inside_helper = try inside.unsafeFunction("helper", inside.module.types.intType());
    inside.setBody(inside_helper, try inside.block(&.{try inside.ret(try inside.int("1"))}));
    const inside_main = try inside.function("main", inside.module.types.intType());
    const inside_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&inside.module.hir, .{ .call = .{ .function = inside_helper, .args = inside_args } });
    const unsafe_body = try inside.block(&.{try inside.ret(call)});
    inside.setBody(inside_main, try inside.block(&.{try addTestStmt(&inside.module.hir, .{ .unsafe_block = unsafe_body })}));
    try inside.checkPass();
}
