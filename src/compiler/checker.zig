const std = @import("std");

const ast_model = @import("ast.zig");
const diagnostics_model = @import("diagnostics.zig");

pub const ast = ast_model;
pub const DiagnosticBag = diagnostics_model.DiagnosticBag;
pub const CheckError = error{InvalidExecutable};

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

    for (block.statements) |stmt| {
        try validateStmt(stmt, diagnostics);
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

fn validateStmt(stmt: ast.Stmt, diagnostics: ?*DiagnosticBag) !void {
    switch (stmt) {
        .return_stmt => |return_stmt| {
            const value = return_stmt.value orelse {
                try report(diagnostics, "P2-M2 C backend requires return statements to have an expression", return_stmt.span);
                return error.InvalidExecutable;
            };
            try validateExpr(value.*, diagnostics);
        },
    }
}

fn validateExpr(expr: ast.Expr, diagnostics: ?*DiagnosticBag) !void {
    switch (expr) {
        .int_literal, .bool_literal => {},
        .group => |group| try validateExpr(group.inner.*, diagnostics),
        .unary => |unary| {
            switch (unary.op) {
                .negate, .logical_not => {},
            }
            try validateExpr(unary.operand.*, diagnostics);
        },
        .binary => |binary| {
            try validateExpr(binary.left.*, diagnostics);
            try validateExpr(binary.right.*, diagnostics);
        },
    }
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
