const std = @import("std");

const ast_model = @import("ast.zig");
const diagnostics_model = @import("diagnostics.zig");

pub const ast = ast_model;
pub const DiagnosticBag = diagnostics_model.DiagnosticBag;
pub const CheckError = error{ InvalidExecutable, OutOfMemory };

pub const ExprType = enum { int, bool };

const LocalSymbol = struct {
    name: []const u8,
    type: ExprType,
    span: ast.SourceSpan,
};

pub const FunctionInfo = struct {
    decl: *const ast.FunctionDecl,
    name: []const u8,
    return_type: ExprType,
    param_types: []ExprType,
    has_body: bool,
};

pub const Executable = struct {
    main: *const ast.FunctionDecl,
    functions: []FunctionInfo,
};

pub fn validateExecutable(unit: ast.CompilationUnit, diagnostics: ?*DiagnosticBag) !Executable {
    var functions = std.ArrayList(FunctionInfo).init(std.heap.page_allocator);
    errdefer {
        for (functions.items) |function| std.heap.page_allocator.free(function.param_types);
        functions.deinit();
    }

    try collectFunctions(unit, &functions, diagnostics);

    const main_info = findFunction(functions.items, "main") orelse {
        try report(diagnostics, "expected top-level 'main' function", unit.span);
        return error.InvalidExecutable;
    };

    if (main_info.return_type != .int) {
        try report(diagnostics, "P2-M2 requires 'main' to return exactly 'int'", main_info.decl.signature.return_type.span);
        return error.InvalidExecutable;
    }

    if (main_info.param_types.len != 0) {
        try report(diagnostics, "P2-M5 C backend requires 'main' to have no parameters", main_info.decl.signature.span);
        return error.InvalidExecutable;
    }

    if (!main_info.has_body) {
        try report(diagnostics, "P2-M2 requires 'main' to have a parsed body", main_info.decl.signature.span);
        return error.InvalidExecutable;
    }

    for (functions.items) |function| {
        if (!function.has_body) continue;
        try validateFunction(function, functions.items, diagnostics);
    }

    return .{ .main = main_info.decl, .functions = try functions.toOwnedSlice() };
}

fn collectFunctions(unit: ast.CompilationUnit, functions: *std.ArrayList(FunctionInfo), diagnostics: ?*DiagnosticBag) !void {
    for (unit.items) |*item| switch (item.*) {
        .function_decl => |*function_decl| {
            if (function_decl.signature.name.operator_suffix != null) continue;
            const name = function_decl.signature.name.base.text;
            if (findFunction(functions.items, name) != null) {
                try report(diagnostics, "duplicate top-level function name", function_decl.signature.name.base.span);
                return error.InvalidExecutable;
            }
            const return_type = executableType(function_decl.signature.return_type) orelse {
                try report(diagnostics, "unsupported function return type in executable subset", function_decl.signature.return_type.span);
                return error.InvalidExecutable;
            };
            const param_types = try std.heap.page_allocator.alloc(ExprType, function_decl.signature.params.len);
            errdefer std.heap.page_allocator.free(param_types);
            for (function_decl.signature.params, 0..) |param, index| {
                param_types[index] = executableType(param.type_name) orelse {
                    try report(diagnostics, "unsupported parameter type in executable subset", param.type_name.span);
                    return error.InvalidExecutable;
                };
            }
            try functions.append(.{
                .decl = function_decl,
                .name = name,
                .return_type = return_type,
                .param_types = param_types,
                .has_body = function_decl.body != null,
            });
        },
        else => {},
    };
}

fn validateFunction(function: FunctionInfo, functions: []const FunctionInfo, diagnostics: ?*DiagnosticBag) !void {
    const body = function.decl.body orelse return;
    const block = body.block orelse {
        try report(diagnostics, "P2-M2 requires function to have a block body", body.span);
        return error.InvalidExecutable;
    };

    var locals = std.ArrayList(LocalSymbol).init(std.heap.page_allocator);
    defer locals.deinit();

    for (function.decl.signature.params, 0..) |param, index| {
        for (locals.items) |local| {
            if (std.mem.eql(u8, local.name, param.name.text)) {
                try report(diagnostics, "duplicate parameter name", param.name.span);
                return error.InvalidExecutable;
            }
        }
        try locals.append(.{ .name = param.name.text, .type = function.param_types[index], .span = param.name.span });
    }

    try validateBlock(block, function.return_type, &locals, functions, diagnostics, false);
}

fn findFunction(functions: []const FunctionInfo, name: []const u8) ?FunctionInfo {
    for (functions) |function| {
        if (std.mem.eql(u8, function.name, name)) return function;
    }
    return null;
}

fn findLocal(locals: []const LocalSymbol, name: []const u8) ?ExprType {
    var index = locals.len;
    while (index != 0) {
        index -= 1;
        if (std.mem.eql(u8, locals[index].name, name)) return locals[index].type;
    }
    return null;
}

fn validateBlock(block: ast.BlockStmt, return_type: ExprType, locals: *std.ArrayList(LocalSymbol), functions: []const FunctionInfo, diagnostics: ?*DiagnosticBag, creates_scope: bool) !void {
    const local_start = locals.items.len;
    defer if (creates_scope) locals.shrinkRetainingCapacity(local_start);

    for (block.statements) |stmt| {
        try validateStmt(stmt, return_type, locals, functions, diagnostics);
    }
}

fn validateStmt(stmt: ast.Stmt, return_type: ExprType, locals: *std.ArrayList(LocalSymbol), functions: []const FunctionInfo, diagnostics: ?*DiagnosticBag) !void {
    switch (stmt) {
        .local_decl => |local_decl| {
            const local_type = executableType(local_decl.type_name) orelse {
                try report(diagnostics, "unsupported local type in executable subset", local_decl.type_name.span);
                return error.InvalidExecutable;
            };
            for (locals.items) |local| {
                if (std.mem.eql(u8, local.name, local_decl.name.text)) {
                    try report(diagnostics, "duplicate local variable name", local_decl.name.span);
                    return error.InvalidExecutable;
                }
            }
            const initializer_type = try validateExpr(local_decl.initializer.*, locals, functions, diagnostics);
            if (initializer_type != local_type) {
                try report(diagnostics, "local initializer type does not match declared type", local_decl.initializer.span());
                return error.InvalidExecutable;
            }
            try locals.append(.{ .name = local_decl.name.text, .type = local_type, .span = local_decl.name.span });
        },
        .assignment => |assignment| {
            const target_type = findLocal(locals.items, assignment.target.text) orelse {
                if (findFunction(functions, assignment.target.text) != null) {
                    try report(diagnostics, "cannot assign to function name", assignment.target.span);
                } else {
                    try report(diagnostics, "assignment target is not a visible local or parameter", assignment.target.span);
                }
                return error.InvalidExecutable;
            };
            const value_type = try validateExpr(assignment.value.*, locals, functions, diagnostics);
            if (value_type != target_type) {
                try report(diagnostics, "assignment expression type does not match target type", assignment.value.span());
                return error.InvalidExecutable;
            }
        },
        .return_stmt => |return_stmt| {
            const value = return_stmt.value orelse {
                try report(diagnostics, "P2-M2 C backend requires return statements to have an expression", return_stmt.span);
                return error.InvalidExecutable;
            };
            const value_type = try validateExpr(value.*, locals, functions, diagnostics);
            if (value_type != return_type) {
                try report(diagnostics, "return expression type does not match function return type", value.span());
                return error.InvalidExecutable;
            }
        },
        .if_stmt => |if_stmt| {
            const condition_type = try validateExpr(if_stmt.condition.*, locals, functions, diagnostics);
            if (condition_type != .bool) {
                try report(diagnostics, "if condition must be bool", if_stmt.condition.span());
                return error.InvalidExecutable;
            }
            try validateBlock(if_stmt.then_block, return_type, locals, functions, diagnostics, true);
            if (if_stmt.else_block) |else_block| {
                try validateBlock(else_block, return_type, locals, functions, diagnostics, true);
            }
        },
        .while_stmt => |while_stmt| {
            const condition_type = try validateExpr(while_stmt.condition.*, locals, functions, diagnostics);
            if (condition_type != .bool) {
                try report(diagnostics, "while condition must be bool", while_stmt.condition.span());
                return error.InvalidExecutable;
            }
            try validateBlock(while_stmt.body, return_type, locals, functions, diagnostics, true);
        },
        .match_stmt => |match_stmt| {
            const scrutinee_type = try validateExpr(match_stmt.scrutinee.*, locals, functions, diagnostics);
            var seen_wildcard = false;
            var int_patterns = std.ArrayList([]const u8).init(std.heap.page_allocator);
            defer int_patterns.deinit();
            var seen_true = false;
            var seen_false = false;

            for (match_stmt.arms) |arm| {
                switch (arm.pattern) {
                    .wildcard => |span| {
                        if (seen_wildcard) {
                            try report(diagnostics, "duplicate wildcard match pattern", span);
                            return error.InvalidExecutable;
                        }
                        seen_wildcard = true;
                    },
                    .int_literal => |literal| {
                        if (scrutinee_type != .int) {
                            try report(diagnostics, "match pattern type does not match scrutinee type", literal.span);
                            return error.InvalidExecutable;
                        }
                        for (int_patterns.items) |seen| {
                            if (std.mem.eql(u8, seen, literal.text)) {
                                try report(diagnostics, "duplicate literal match pattern", literal.span);
                                return error.InvalidExecutable;
                            }
                        }
                        try int_patterns.append(literal.text);
                    },
                    .bool_literal => |literal| {
                        if (scrutinee_type != .bool) {
                            try report(diagnostics, "match pattern type does not match scrutinee type", literal.span);
                            return error.InvalidExecutable;
                        }
                        if (literal.value) {
                            if (seen_true) {
                                try report(diagnostics, "duplicate literal match pattern", literal.span);
                                return error.InvalidExecutable;
                            }
                            seen_true = true;
                        } else {
                            if (seen_false) {
                                try report(diagnostics, "duplicate literal match pattern", literal.span);
                                return error.InvalidExecutable;
                            }
                            seen_false = true;
                        }
                    },
                }
                {
                    const arm_local_start = locals.items.len;
                    defer locals.shrinkRetainingCapacity(arm_local_start);
                    try validateStmt(arm.body, return_type, locals, functions, diagnostics);
                }
            }
        },
        .block_stmt => |block_stmt| try validateBlock(block_stmt, return_type, locals, functions, diagnostics, true),
    }
}

fn validateExpr(expr: ast.Expr, locals: *const std.ArrayList(LocalSymbol), functions: []const FunctionInfo, diagnostics: ?*DiagnosticBag) !ExprType {
    switch (expr) {
        .int_literal => return .int,
        .bool_literal => return .bool,
        .identifier => |identifier| {
            if (findLocal(locals.items, identifier.name.text)) |local_type| return local_type;
            try report(diagnostics, "unknown identifier in executable subset", identifier.span);
            return error.InvalidExecutable;
        },
        .call => |call| {
            const callee = findFunction(functions, call.callee.text) orelse {
                try report(diagnostics, "unknown function in executable subset", call.callee.span);
                return error.InvalidExecutable;
            };
            if (!callee.has_body) {
                try report(diagnostics, "cannot call function without body in executable subset", call.callee.span);
                return error.InvalidExecutable;
            }
            if (call.args.len != callee.param_types.len) {
                try report(diagnostics, "function call argument count mismatch", call.span);
                return error.InvalidExecutable;
            }
            for (call.args, 0..) |arg, index| {
                const arg_type = try validateExpr(arg.*, locals, functions, diagnostics);
                if (arg_type != callee.param_types[index]) {
                    try report(diagnostics, "function call argument type mismatch", arg.span());
                    return error.InvalidExecutable;
                }
            }
            return callee.return_type;
        },
        .group => |group| return validateExpr(group.inner.*, locals, functions, diagnostics),
        .unary => |unary| {
            const operand_type = try validateExpr(unary.operand.*, locals, functions, diagnostics);
            switch (unary.op) {
                .negate => if (operand_type == .int) return .int else {
                    try report(diagnostics, "arithmetic unary operator requires int operand", unary.span);
                    return error.InvalidExecutable;
                },
                .logical_not => if (operand_type == .bool) return .bool else {
                    try report(diagnostics, "logical unary operator requires bool operand", unary.span);
                    return error.InvalidExecutable;
                },
            }
        },
        .binary => |binary| {
            const left_type = try validateExpr(binary.left.*, locals, functions, diagnostics);
            const right_type = try validateExpr(binary.right.*, locals, functions, diagnostics);
            switch (binary.op) {
                .add, .subtract, .multiply, .divide, .modulo => {
                    if (left_type == .int and right_type == .int) return .int;
                    try report(diagnostics, "arithmetic binary operator requires int operands", binary.span);
                    return error.InvalidExecutable;
                },
                .less, .less_equal, .greater, .greater_equal => {
                    if (left_type == .int and right_type == .int) return .bool;
                    try report(diagnostics, "comparison operator requires int operands", binary.span);
                    return error.InvalidExecutable;
                },
                .equal_equal, .bang_equal => {
                    if (left_type == right_type) return .bool;
                    try report(diagnostics, "equality operator requires matching operand types", binary.span);
                    return error.InvalidExecutable;
                },
                .logical_and, .logical_or => {
                    if (left_type == .bool and right_type == .bool) return .bool;
                    try report(diagnostics, "logical binary operator requires bool operands", binary.span);
                    return error.InvalidExecutable;
                },
            }
        },
    }
}

fn executableType(type_name: ast.TypeName) ?ExprType {
    if (isExactSimpleType(type_name, "int")) return .int;
    if (isExactSimpleType(type_name, "bool")) return .bool;
    return null;
}

fn isExactSimpleType(type_name: ast.TypeName, expected: []const u8) bool {
    return !type_name.is_mut and
        !type_name.is_reference and
        !type_name.is_pointer and
        type_name.generic_args.len == 0 and
        type_name.name.parts.len == 1 and
        std.mem.eql(u8, type_name.name.parts[0].text, expected);
}

fn report(diagnostics: ?*DiagnosticBag, message: []const u8, span: ast.SourceSpan) !void {
    if (diagnostics) |bag| {
        try bag.append(diagnostics_model.makeDiagnostic(
            .InvalidExecutableSubset,
            .@"error",
            message,
            span,
        ));
    }
}

const parser_model = @import("parser.zig");
const source_model = @import("source.zig");

fn parseForTest(source_text: []const u8, parse_diagnostics: *DiagnosticBag) !ast.CompilationUnit {
    const source_file = try source_model.SourceFile.init(std.testing.allocator, "test.concept", source_text);
    defer source_file.deinit(std.testing.allocator);
    return parser_model.parseSource(std.testing.allocator, source_file, parse_diagnostics);
}

fn expectValid(source_text: []const u8) !void {
    var parse_diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const unit = try parseForTest(source_text, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    const executable = try validateExecutable(unit, &check_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), check_diagnostics.count());
    try std.testing.expectEqualStrings("main", executable.main.signature.name.base.text);
}

fn expectInvalid(source_text: []const u8, message: []const u8) !void {
    var parse_diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const unit = try parseForTest(source_text, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    try std.testing.expectError(error.InvalidExecutable, validateExecutable(unit, &check_diagnostics));
    try std.testing.expectEqual(@as(usize, 1), check_diagnostics.count());
    try std.testing.expectEqualStrings(message, check_diagnostics.diagnostics.items[0].message);
}

test "checker validates minimal int main" {
    try expectValid("module Main; int main() { return 0; }");
}

test "checker rejects missing main" {
    try expectInvalid(
        "module Main; int helper() { return 0; }",
        "expected top-level 'main' function",
    );
}

test "checker rejects main without body" {
    try expectInvalid(
        "module Main; int main();",
        "P2-M2 requires 'main' to have a parsed body",
    );
}

test "checker rejects non-int main" {
    try expectInvalid(
        "module Main; bool main() { return true; }",
        "P2-M2 requires 'main' to return exactly 'int'",
    );
}

test "checker rejects unsupported return without expression" {
    try expectInvalid(
        "module Main; int main() { return; }",
        "P2-M2 C backend requires return statements to have an expression",
    );
}

test "checker rejects main parameters for C backend v0" {
    try expectInvalid(
        "module Main; int main(int argc) { return 0; }",
        "P2-M5 C backend requires 'main' to have no parameters",
    );
}

test "checker validates local int usage" {
    try expectValid("module Main; int main() { int x = 1 + 2; return x * 3; }");
}

test "checker rejects unknown identifier" {
    try expectInvalid(
        "module Main; int main() { return x; }",
        "unknown identifier in executable subset",
    );
}

test "checker rejects duplicate local" {
    try expectInvalid(
        "module Main; int main() { int x = 1; int x = 2; return x; }",
        "duplicate local variable name",
    );
}

test "checker rejects unsupported local type" {
    try expectInvalid(
        "module Main; int main() { String x = 1; return 0; }",
        "unsupported local type in executable subset",
    );
}

test "checker rejects bad initializer type" {
    try expectInvalid(
        "module Main; int main() { int x = true; return x; }",
        "local initializer type does not match declared type",
    );
}

test "checker rejects arithmetic operand type mismatch" {
    try expectInvalid(
        "module Main; int main() { return true + 1; }",
        "arithmetic binary operator requires int operands",
    );
}

test "checker rejects logical operand type mismatch" {
    try expectInvalid(
        "module Main; int main() { return 1 && false; }",
        "logical binary operator requires bool operands",
    );
}

test "checker accepts assignment to int local" {
    try expectValid("module Main; int main() { int x = 1; x = x + 2; return x; }");
}

test "checker accepts assignment to bool local" {
    try expectValid("module Main; int main() { bool ok = true; ok = !ok; if (ok) { return 1; } return 0; }");
}

test "checker rejects assignment to unknown identifier" {
    try expectInvalid(
        "module Main; int main() { x = 1; return 0; }",
        "assignment target is not a visible local or parameter",
    );
}

test "checker rejects assignment type mismatch" {
    try expectInvalid(
        "module Main; int main() { int x = 1; x = true; return x; }",
        "assignment expression type does not match target type",
    );
}

test "checker rejects assignment to out-of-scope branch local" {
    try expectInvalid(
        "module Main; int main() { if (true) { int x = 1; } x = 2; return 0; }",
        "assignment target is not a visible local or parameter",
    );
}

test "checker allows assignment to parameter" {
    try expectValid("module Main; int bump(int x) { x = x + 1; return x; } int main() { return bump(2); }");
}

test "checker rejects assignment to function name" {
    try expectInvalid(
        "module Main; int f() { return 1; } int main() { f = 2; return 0; }",
        "cannot assign to function name",
    );
}

test "checker validates call to known function" {
    try expectValid("module Main; int add(int a, int b) { return a + b; } int main() { return add(1, 2); }");
}

test "checker validates parameter identifiers in function body" {
    try expectValid("module Main; int id(int x) { return x; } int main() { return id(3); }");
}

test "checker validates locals and parameters in function body" {
    try expectValid("module Main; int add_local(int x) { int y = 2; return x + y; } int main() { return add_local(1); }");
}

test "checker rejects duplicate top-level function" {
    try expectInvalid(
        "module Main; int f() { return 0; } int f() { return 1; } int main() { return f(); }",
        "duplicate top-level function name",
    );
}

test "checker rejects unknown function call" {
    try expectInvalid(
        "module Main; int main() { return missing(); }",
        "unknown function in executable subset",
    );
}

test "checker rejects wrong argument count" {
    try expectInvalid(
        "module Main; int add(int a, int b) { return a + b; } int main() { return add(1); }",
        "function call argument count mismatch",
    );
}

test "checker rejects wrong argument type" {
    try expectInvalid(
        "module Main; int id(int x) { return x; } int main() { return id(true); }",
        "function call argument type mismatch",
    );
}

test "checker rejects duplicate local colliding with parameter" {
    try expectInvalid(
        "module Main; int f(int x) { int x = 1; return x; } int main() { return f(0); }",
        "duplicate local variable name",
    );
}

test "checker accepts bool if condition" {
    try expectValid("module Main; int main() { if (true) { return 1; } return 0; }");
}

test "checker accepts comparison if condition" {
    try expectValid("module Main; int main() { int a = 3; int b = 7; if (a < b) { return b; } return a; }");
}

test "checker rejects int if condition" {
    try expectInvalid(
        "module Main; int main() { if (1) { return 1; } return 0; }",
        "if condition must be bool",
    );
}

test "checker allows outer local inside if" {
    try expectValid("module Main; int main() { int x = 7; if (true) { return x; } return 0; }");
}

test "checker rejects branch local after branch" {
    try expectInvalid(
        "module Main; int main() { if (true) { int x = 7; } return x; }",
        "unknown identifier in executable subset",
    );
}

test "checker rejects duplicate local in visible outer scope" {
    try expectInvalid(
        "module Main; int main() { int x = 1; if (true) { int x = 2; } return x; }",
        "duplicate local variable name",
    );
}

test "checker rejects match arm local after arm" {
    try expectInvalid(
        "module Main; int main() { int x = 1; match (x) { 1 => { int y = 7; } _ => return 0; } return y; }",
        "unknown identifier in executable subset",
    );
}

test "checker accepts int match" {
    try expectValid("module Main; int main() { int x = 2; match (x) { 1 => return 10; 2 => return 7; _ => return 0; } return 0; }");
}

test "checker accepts bool match" {
    try expectValid("module Main; int main() { bool ok = true; match (ok) { true => return 7; false => return 0; } return 0; }");
}

test "checker rejects match pattern type mismatch" {
    try expectInvalid(
        "module Main; int main() { int x = 1; match (x) { true => return 1; } return 0; }",
        "match pattern type does not match scrutinee type",
    );
}

test "checker rejects duplicate literal match pattern" {
    try expectInvalid(
        "module Main; int main() { int x = 1; match (x) { 1 => return 1; 1 => return 2; } return 0; }",
        "duplicate literal match pattern",
    );
}

test "checker rejects duplicate wildcard match pattern" {
    try expectInvalid(
        "module Main; int main() { int x = 1; match (x) { _ => return 1; _ => return 2; } return 0; }",
        "duplicate wildcard match pattern",
    );
}

test "checker checks match arm statements" {
    try expectInvalid(
        "module Main; int main() { int x = 1; match (x) { 1 => return true; } return 0; }",
        "return expression type does not match function return type",
    );
}

test "checker accepts bool while condition" {
    try expectValid("module Main; int main() { while (true) { return 0; } return 1; }");
}

test "checker accepts comparison while condition" {
    try expectValid("module Main; int main() { int x = 0; while (x < 7) { x = x + 1; } return x; }");
}

test "checker rejects int while condition" {
    try expectInvalid(
        "module Main; int main() { int x = 0; while (x) { x = x + 1; } return x; }",
        "while condition must be bool",
    );
}

test "checker allows outer local inside while" {
    try expectValid("module Main; int main() { int x = 0; while (x < 1) { return x; } return 1; }");
}

test "checker allows assignment to outer local inside while" {
    try expectValid("module Main; int main() { int x = 0; while (x < 1) { x = 1; } return x; }");
}

test "checker rejects while body local after loop" {
    try expectInvalid(
        "module Main; int main() { while (true) { int x = 1; } return x; }",
        "unknown identifier in executable subset",
    );
}

test "checker rejects duplicate local in visible while scope" {
    try expectInvalid(
        "module Main; int main() { int x = 0; while (true) { int x = 1; } return x; }",
        "duplicate local variable name",
    );
}
