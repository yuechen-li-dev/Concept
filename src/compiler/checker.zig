const std = @import("std");

const ast_model = @import("ast.zig");
const diagnostics_model = @import("diagnostics.zig");

pub const ast = ast_model;
pub const DiagnosticBag = diagnostics_model.DiagnosticBag;
pub const CheckError = error{ InvalidExecutable, OutOfMemory };

const ExprType = enum { int, bool };

const LocalSymbol = struct {
    name: []const u8,
    type: ExprType,
    span: ast.SourceSpan,
};

pub const Executable = struct {
    main: *const ast.FunctionDecl,
};

pub fn validateExecutable(unit: ast.CompilationUnit, diagnostics: ?*DiagnosticBag) !Executable {
    const main = findMain(unit) orelse {
        try report(diagnostics, "expected top-level 'main' function", unit.span);
        return error.InvalidExecutable;
    };

    if (!isExactSimpleType(main.signature.return_type, "int")) {
        try report(diagnostics, "P2-M2 requires 'main' to return exactly 'int'", main.signature.return_type.span);
        return error.InvalidExecutable;
    }

    if (main.signature.params.len != 0) {
        try report(diagnostics, "P2-M2 C backend requires 'main' to have no parameters", main.signature.span);
        return error.InvalidExecutable;
    }

    const body = main.body orelse {
        try report(diagnostics, "P2-M2 requires 'main' to have a parsed body", main.signature.span);
        return error.InvalidExecutable;
    };

    const block = body.block orelse {
        try report(diagnostics, "P2-M2 requires 'main' to have a block body", body.span);
        return error.InvalidExecutable;
    };

    var locals = std.ArrayList(LocalSymbol).init(std.heap.page_allocator);
    defer locals.deinit();
    for (block.statements) |stmt| {
        try validateStmt(stmt, &locals, diagnostics);
    }

    return .{ .main = main };
}

fn findMain(unit: ast.CompilationUnit) ?*const ast.FunctionDecl {
    for (unit.items) |*item| {
        switch (item.*) {
            .function_decl => |*function_decl| {
                if (function_decl.signature.name.operator_suffix == null and
                    std.mem.eql(u8, function_decl.signature.name.base.text, "main"))
                {
                    return function_decl;
                }
            },
            else => {},
        }
    }
    return null;
}

fn validateStmt(stmt: ast.Stmt, locals: *std.ArrayList(LocalSymbol), diagnostics: ?*DiagnosticBag) !void {
    switch (stmt) {
        .local_decl => |local_decl| {
            const local_type = executableLocalType(local_decl.type_name) orelse {
                try report(diagnostics, "unsupported local type in executable subset", local_decl.type_name.span);
                return error.InvalidExecutable;
            };
            for (locals.items) |local| {
                if (std.mem.eql(u8, local.name, local_decl.name.text)) {
                    try report(diagnostics, "duplicate local variable name", local_decl.name.span);
                    return error.InvalidExecutable;
                }
            }
            const initializer_type = try validateExpr(local_decl.initializer.*, locals, diagnostics);
            if (initializer_type != local_type) {
                try report(diagnostics, "local initializer type does not match declared type", local_decl.initializer.span());
                return error.InvalidExecutable;
            }
            try locals.append(.{ .name = local_decl.name.text, .type = local_type, .span = local_decl.name.span });
        },
        .return_stmt => |return_stmt| {
            const value = return_stmt.value orelse {
                try report(diagnostics, "P2-M2 C backend requires return statements to have an expression", return_stmt.span);
                return error.InvalidExecutable;
            };
            _ = try validateExpr(value.*, locals, diagnostics);
        },
    }
}

fn validateExpr(expr: ast.Expr, locals: *const std.ArrayList(LocalSymbol), diagnostics: ?*DiagnosticBag) !ExprType {
    switch (expr) {
        .int_literal => return .int,
        .bool_literal => return .bool,
        .identifier => |identifier| {
            for (locals.items) |local| {
                if (std.mem.eql(u8, local.name, identifier.name.text)) return local.type;
            }
            try report(diagnostics, "unknown identifier in executable subset", identifier.span);
            return error.InvalidExecutable;
        },
        .group => |group| return validateExpr(group.inner.*, locals, diagnostics),
        .unary => |unary| {
            const operand_type = try validateExpr(unary.operand.*, locals, diagnostics);
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
            const left_type = try validateExpr(binary.left.*, locals, diagnostics);
            const right_type = try validateExpr(binary.right.*, locals, diagnostics);
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

fn executableLocalType(type_name: ast.TypeName) ?ExprType {
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
        "P2-M2 C backend requires 'main' to have no parameters",
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
