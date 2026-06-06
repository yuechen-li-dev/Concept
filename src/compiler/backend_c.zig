const std = @import("std");

const ast_model = @import("ast.zig");
const checker = @import("checker.zig");
const diagnostics_model = @import("diagnostics.zig");

pub const EmitError = error{InvalidExecutable} || std.mem.Allocator.Error;

pub fn emitExecutable(allocator: std.mem.Allocator, unit: ast_model.CompilationUnit, diagnostics: ?*diagnostics_model.DiagnosticBag) EmitError![]const u8 {
    const executable = checker.validateExecutable(unit, diagnostics) catch |err| switch (err) {
        error.InvalidExecutable => return error.InvalidExecutable,
        error.OutOfMemory => return error.OutOfMemory,
    };

    var output = std.ArrayList(u8).init(allocator);
    errdefer output.deinit();
    try emitMain(output.writer(), executable.main.*);
    return output.toOwnedSlice();
}

fn emitMain(writer: anytype, main: ast_model.FunctionDecl) !void {
    try writer.writeAll("int main(void) {");
    try writer.writeByte('\n');

    const block = main.body.?.block.?;
    for (block.statements) |stmt| {
        try emitStmt(writer, stmt);
    }

    try writer.writeAll("}\n");
}

fn emitStmt(writer: anytype, stmt: ast_model.Stmt) !void {
    switch (stmt) {
        .return_stmt => |return_stmt| {
            try writer.writeAll("    return ");
            try emitExpr(writer, return_stmt.value.?.*);
            try writer.writeAll(";\n");
        },
    }
}

fn emitExpr(writer: anytype, expr: ast_model.Expr) !void {
    switch (expr) {
        .int_literal => |literal| try writer.writeAll(literal.text),
        .bool_literal => |literal| try writer.writeAll(if (literal.value) "1" else "0"),
        .group => |group| {
            try writer.writeByte('(');
            try emitExpr(writer, group.inner.*);
            try writer.writeByte(')');
        },
        .unary => |unary| {
            try writer.writeAll(unary.op.lexeme());
            try emitExpr(writer, unary.operand.*);
        },
        .binary => |binary| {
            try writer.writeByte('(');
            try emitExpr(writer, binary.left.*);
            try writer.writeByte(' ');
            try writer.writeAll(binary.op.lexeme());
            try writer.writeByte(' ');
            try emitExpr(writer, binary.right.*);
            try writer.writeByte(')');
        },
    }
}

const parser_model = @import("parser.zig");
const source_model = @import("source.zig");

fn emitForTest(source_text: []const u8) ![]const u8 {
    var parse_diagnostics = diagnostics_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = diagnostics_model.DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, "test.concept", source_text);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    const c_source = try emitExecutable(std.testing.allocator, unit, &check_diagnostics);
    errdefer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 0), check_diagnostics.count());
    return c_source;
}

fn expectEmit(source_text: []const u8, expected: []const u8) !void {
    const c_source = try emitForTest(source_text);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqualStrings(expected, c_source);
}

test "C backend emits return zero" {
    try expectEmit(
        "module Main; int main() { return 0; }",
        "int main(void) {\n    return 0;\n}\n",
    );
}

test "C backend emits arithmetic with precedence parentheses" {
    try expectEmit(
        "module Main; int main() { return 1 + 2 * 3; }",
        "int main(void) {\n    return (1 + (2 * 3));\n}\n",
    );
}

test "C backend emits parenthesized expression" {
    try expectEmit(
        "module Main; int main() { return (1 + 2) * 3; }",
        "int main(void) {\n    return (((1 + 2)) * 3);\n}\n",
    );
}

test "C backend emits bool literals as integers" {
    try expectEmit(
        "module Main; int main() { return true == false; }",
        "int main(void) {\n    return (1 == 0);\n}\n",
    );
}

test "C backend emits unary expressions" {
    try expectEmit(
        "module Main; int main() { return -1 + !false; }",
        "int main(void) {\n    return (-1 + !0);\n}\n",
    );
}

test "C backend reports checker errors" {
    var parse_diagnostics = diagnostics_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = diagnostics_model.DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, "test.concept", "module Main; int main() { return; }");
    defer source_file.deinit(std.testing.allocator);
    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectError(error.InvalidExecutable, emitExecutable(std.testing.allocator, unit, &check_diagnostics));
    try std.testing.expectEqual(@as(usize, 1), check_diagnostics.count());
}

fn expectCorpusC(comptime source_path: []const u8, comptime expected_path: []const u8) !void {
    var parse_diagnostics = diagnostics_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = diagnostics_model.DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const source_text = @embedFile(source_path);
    const source_file = try source_model.SourceFile.init(std.testing.allocator, source_path, source_text);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    const c_source = try emitExecutable(std.testing.allocator, unit, &check_diagnostics);
    defer std.testing.allocator.free(c_source);
    try std.testing.expectEqual(@as(usize, 0), check_diagnostics.count());
    try std.testing.expectEqualStrings(@embedFile(expected_path), c_source);
}

test "Phase 2 C snapshot: return zero" {
    try expectCorpusC(
        "../../tests/corpus/phase2/return_zero.concept",
        "../../tests/corpus/phase2/return_zero.c.expected",
    );
}

test "Phase 2 C snapshot: arithmetic return" {
    try expectCorpusC(
        "../../tests/corpus/phase2/arithmetic_return.concept",
        "../../tests/corpus/phase2/arithmetic_return.c.expected",
    );
}
