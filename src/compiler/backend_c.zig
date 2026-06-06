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
    const writer = output.writer();

    var body_count: usize = 0;
    for (executable.functions) |function| {
        if (function.has_body) body_count += 1;
    }
    if (body_count > 1) {
        for (executable.functions) |function| {
            if (!function.has_body) continue;
            try emitPrototype(writer, function.decl.*);
        }
        try writer.writeByte('\n');
    }

    var emitted_any = false;
    for (executable.functions) |function| {
        if (!function.has_body) continue;
        if (emitted_any) try writer.writeByte('\n');
        try emitFunction(writer, function.decl.*);
        emitted_any = true;
    }
    return output.toOwnedSlice();
}

fn emitPrototype(writer: anytype, function: ast_model.FunctionDecl) !void {
    try emitCType(writer, function.signature.return_type);
    try writer.writeByte(' ');
    try writer.writeAll(function.signature.name.base.text);
    try writer.writeByte('(');
    try emitParamList(writer, function.signature.params, std.mem.eql(u8, function.signature.name.base.text, "main"));
    try writer.writeAll(");\n");
}

fn emitFunction(writer: anytype, function: ast_model.FunctionDecl) !void {
    try emitCType(writer, function.signature.return_type);
    try writer.writeByte(' ');
    try writer.writeAll(function.signature.name.base.text);
    try writer.writeByte('(');
    try emitParamList(writer, function.signature.params, std.mem.eql(u8, function.signature.name.base.text, "main"));
    try writer.writeAll(") {");
    try writer.writeByte('\n');

    const block = function.body.?.block.?;
    try emitBlockContents(writer, block, 1);

    try writer.writeAll("}\n");
}

fn emitIndent(writer: anytype, depth: usize) !void {
    for (0..depth) |_| try writer.writeAll("    ");
}

fn emitBlockContents(writer: anytype, block: ast_model.BlockStmt, depth: usize) !void {
    for (block.statements) |stmt| {
        try emitStmt(writer, stmt, depth);
    }
}

fn emitStmt(writer: anytype, stmt: ast_model.Stmt, depth: usize) !void {
    switch (stmt) {
        .local_decl => |local_decl| {
            try emitIndent(writer, depth);
            try emitCType(writer, local_decl.type_name);
            try writer.writeByte(' ');
            try writer.writeAll(local_decl.name.text);
            try writer.writeAll(" = ");
            try emitExpr(writer, local_decl.initializer.*);
            try writer.writeAll(";\n");
        },
        .assignment => |assignment| {
            try emitIndent(writer, depth);
            try writer.writeAll(assignment.target.text);
            try writer.writeAll(" = ");
            try emitExpr(writer, assignment.value.*);
            try writer.writeAll(";\n");
        },
        .return_stmt => |return_stmt| {
            try emitIndent(writer, depth);
            try writer.writeAll("return ");
            try emitExpr(writer, return_stmt.value.?.*);
            try writer.writeAll(";\n");
        },
        .if_stmt => |if_stmt| {
            try emitIndent(writer, depth);
            try writer.writeAll("if (");
            try emitExpr(writer, if_stmt.condition.*);
            try writer.writeAll(") {\n");
            try emitBlockContents(writer, if_stmt.then_block, depth + 1);
            try emitIndent(writer, depth);
            try writer.writeByte('}');
            if (if_stmt.else_block) |else_block| {
                try writer.writeAll(" else {\n");
                try emitBlockContents(writer, else_block, depth + 1);
                try emitIndent(writer, depth);
                try writer.writeAll("}\n");
            } else {
                try writer.writeByte('\n');
            }
        },
        .while_stmt => |while_stmt| {
            try emitIndent(writer, depth);
            try writer.writeAll("while (");
            try emitExpr(writer, while_stmt.condition.*);
            try writer.writeAll(") {\n");
            try emitBlockContents(writer, while_stmt.body, depth + 1);
            try emitIndent(writer, depth);
            try writer.writeAll("}\n");
        },
        .match_stmt => |match_stmt| {
            try emitIndent(writer, depth);
            try writer.writeAll("switch (");
            try emitExpr(writer, match_stmt.scrutinee.*);
            try writer.writeAll(") {\n");
            for (match_stmt.arms) |arm| {
                try emitIndent(writer, depth + 1);
                switch (arm.pattern) {
                    .int_literal => |literal| {
                        try writer.writeAll("case ");
                        try writer.writeAll(literal.text);
                        try writer.writeAll(":\n");
                    },
                    .bool_literal => |literal| {
                        try writer.writeAll("case ");
                        try writer.writeAll(if (literal.value) "1" else "0");
                        try writer.writeAll(":\n");
                    },
                    .wildcard => {
                        try writer.writeAll("default:\n");
                    },
                }
                try emitStmt(writer, arm.body, depth + 2);
                if (!stmtAlwaysExits(arm.body)) {
                    try emitIndent(writer, depth + 2);
                    try writer.writeAll("break;\n");
                }
            }
            try emitIndent(writer, depth);
            try writer.writeAll("}\n");
        },
        .block_stmt => |block_stmt| {
            try emitIndent(writer, depth);
            try writer.writeAll("{\n");
            try emitBlockContents(writer, block_stmt, depth + 1);
            try emitIndent(writer, depth);
            try writer.writeAll("}\n");
        },
    }
}

fn stmtAlwaysExits(stmt: ast_model.Stmt) bool {
    return switch (stmt) {
        .return_stmt => true,
        else => false,
    };
}

fn emitExpr(writer: anytype, expr: ast_model.Expr) !void {
    switch (expr) {
        .int_literal => |literal| try writer.writeAll(literal.text),
        .bool_literal => |literal| try writer.writeAll(if (literal.value) "1" else "0"),
        .identifier => |identifier| try writer.writeAll(identifier.name.text),
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
        .call => |call| {
            try writer.writeAll(call.callee.text);
            try writer.writeByte('(');
            for (call.args, 0..) |arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try emitExpr(writer, arg.*);
            }
            try writer.writeByte(')');
        },
    }
}

fn emitParamList(writer: anytype, params: []const ast_model.ParamDecl, is_main: bool) !void {
    _ = is_main;
    if (params.len == 0) {
        try writer.writeAll("void");
        return;
    }
    for (params, 0..) |param, index| {
        if (index != 0) try writer.writeAll(", ");
        try emitCType(writer, param.type_name);
        try writer.writeByte(' ');
        try writer.writeAll(param.name.text);
    }
}

fn emitCType(writer: anytype, type_name: ast_model.TypeName) !void {
    _ = type_name;
    try writer.writeAll("int");
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

test "C backend emits assignment statement" {
    try expectEmit(
        "module Main; int main() { int x = 1; x = 2; return x; }",
        "int main(void) {\n    int x = 1;\n    x = 2;\n    return x;\n}\n",
    );
}

test "C backend emits assignment with expression" {
    try expectEmit(
        "module Main; int main() { int x = 1; x = x + 2; return x; }",
        "int main(void) {\n    int x = 1;\n    x = (x + 2);\n    return x;\n}\n",
    );
}

test "C backend emits assignment inside if block" {
    try expectEmit(
        "module Main; int main() { int x = 1; if (true) { x = 3; } return x; }",
        "int main(void) {\n    int x = 1;\n    if (1) {\n        x = 3;\n    }\n    return x;\n}\n",
    );
}

test "C backend emits assignment inside match arm" {
    try expectEmit(
        "module Main; int main() { int x = 1; match (x) { 1 => x = 3; _ => x = 0; } return x; }",
        "int main(void) {\n    int x = 1;\n    switch (x) {\n        case 1:\n            x = 3;\n            break;\n        default:\n            x = 0;\n            break;\n    }\n    return x;\n}\n",
    );
}

test "Phase 2 C snapshot: local arithmetic return" {
    try expectCorpusC(
        "../../tests/corpus/phase2/local_arithmetic_return.concept",
        "../../tests/corpus/phase2/local_arithmetic_return.c.expected",
    );
}

test "Phase 2 C snapshot: assignment local int" {
    try expectCorpusC(
        "../../tests/corpus/phase2/assignment_local_int.concept",
        "../../tests/corpus/phase2/assignment_local_int.c.expected",
    );
}

test "C backend emits int local declaration" {
    try expectEmit(
        "module Main; int main() { int x = 1; return x; }",
        "int main(void) {\n    int x = 1;\n    return x;\n}\n",
    );
}

test "C backend emits bool local as int" {
    try expectEmit(
        "module Main; int main() { bool ok = true; return ok; }",
        "int main(void) {\n    int ok = 1;\n    return ok;\n}\n",
    );
}

test "C backend emits identifier expression" {
    try expectEmit(
        "module Main; int main() { int x = 4; return x + 1; }",
        "int main(void) {\n    int x = 4;\n    return (x + 1);\n}\n",
    );
}

test "C backend emits local arithmetic return" {
    try expectEmit(
        "module Main; int main() { int x = 1 + 2; return x * 3; }",
        "int main(void) {\n    int x = (1 + 2);\n    return (x * 3);\n}\n",
    );
}

test "C backend emits multiple functions parameters calls and prototypes" {
    try expectEmit(
        "module Main; int add(int a, int b) { return a + b; } int main() { return add(1, 2); }",
        "int add(int a, int b);\nint main(void);\n\nint add(int a, int b) {\n    return (a + b);\n}\n\nint main(void) {\n    return add(1, 2);\n}\n",
    );
}

test "C backend emits bool parameters and returns as int" {
    try expectEmit(
        "module Main; bool same(bool a, bool b) { return a == b; } int main() { return same(true, true); }",
        "int same(int a, int b);\nint main(void);\n\nint same(int a, int b) {\n    return (a == b);\n}\n\nint main(void) {\n    return same(1, 1);\n}\n",
    );
}

test "C backend emits forward call prototypes" {
    try expectEmit(
        "module Main; int main() { return later(); } int later() { return 3; }",
        "int main(void);\nint later(void);\n\nint main(void) {\n    return later();\n}\n\nint later(void) {\n    return 3;\n}\n",
    );
}

test "Phase 2 C snapshot: function call add" {
    try expectCorpusC(
        "../../tests/corpus/phase2/function_call_add.concept",
        "../../tests/corpus/phase2/function_call_add.c.expected",
    );
}

test "C backend emits if without else" {
    try expectEmit(
        "module Main; int main() { if (true) { return 7; } return 0; }",
        "int main(void) {\n    if (1) {\n        return 7;\n    }\n    return 0;\n}\n",
    );
}

test "C backend emits if with else" {
    try expectEmit(
        "module Main; int main() { if (false) { return 1; } else { return 7; } }",
        "int main(void) {\n    if (0) {\n        return 1;\n    } else {\n        return 7;\n    }\n}\n",
    );
}

test "C backend emits nested blocks" {
    try expectEmit(
        "module Main; int main() { { int x = 7; if (true) { return x; } } return 0; }",
        "int main(void) {\n    {\n        int x = 7;\n        if (1) {\n            return x;\n        }\n    }\n    return 0;\n}\n",
    );
}

test "C backend emits if condition expression" {
    try expectEmit(
        "module Main; int max(int a, int b) { if (a > b) { return a; } else { return b; } } int main() { return max(3, 7); }",
        "int max(int a, int b);\nint main(void);\n\nint max(int a, int b) {\n    if ((a > b)) {\n        return a;\n    } else {\n        return b;\n    }\n}\n\nint main(void) {\n    return max(3, 7);\n}\n",
    );
}

test "Phase 2 C snapshot: if compare function" {
    try expectCorpusC(
        "../../tests/corpus/phase2/if_compare_function.concept",
        "../../tests/corpus/phase2/if_compare_function.c.expected",
    );
}

test "Phase 2 C snapshot: if else return" {
    try expectCorpusC(
        "../../tests/corpus/phase2/if_else_return.concept",
        "../../tests/corpus/phase2/if_else_return.c.expected",
    );
}

test "C backend emits switch for int match" {
    try expectEmit(
        "module Main; int main() { int x = 2; match (x) { 1 => return 10; 2 => return 7; _ => return 0; } }",
        "int main(void) {\n    int x = 2;\n    switch (x) {\n        case 1:\n            return 10;\n        case 2:\n            return 7;\n        default:\n            return 0;\n    }\n}\n",
    );
}

test "C backend emits switch for bool match" {
    try expectEmit(
        "module Main; int main() { bool ok = false; match (ok) { true => return 1; false => return 7; _ => return 0; } }",
        "int main(void) {\n    int ok = 0;\n    switch (ok) {\n        case 1:\n            return 1;\n        case 0:\n            return 7;\n        default:\n            return 0;\n    }\n}\n",
    );
}

test "Phase 2 C snapshot: match int return" {
    try expectCorpusC(
        "../../tests/corpus/phase2/match_int_return.concept",
        "../../tests/corpus/phase2/match_int_return.c.expected",
    );
}

test "C backend emits while loop" {
    try expectEmit(
        "module Main; int main() { int x = 0; while (x < 7) { x = x + 1; } return x; }",
        "int main(void) {\n    int x = 0;\n    while ((x < 7)) {\n        x = (x + 1);\n    }\n    return x;\n}\n",
    );
}

test "C backend emits nested while loops" {
    try expectEmit(
        "module Main; int main() { int x = 0; int y = 0; while (x < 2) { while (y < 2) { y = y + 1; } x = x + 1; } return x + y; }",
        "int main(void) {\n    int x = 0;\n    int y = 0;\n    while ((x < 2)) {\n        while ((y < 2)) {\n            y = (y + 1);\n        }\n        x = (x + 1);\n    }\n    return (x + y);\n}\n",
    );
}

test "Phase 2 C snapshot: while count to seven" {
    try expectCorpusC(
        "../../tests/corpus/phase2/while_count_to_seven.concept",
        "../../tests/corpus/phase2/while_count_to_seven.c.expected",
    );
}
