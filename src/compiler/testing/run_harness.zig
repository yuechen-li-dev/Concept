const std = @import("std");
const builtin = @import("builtin");

const backend_c = @import("../backend_c.zig");
const diagnostics_model = @import("../diagnostics.zig");
const parser_model = @import("../parser.zig");
const source_model = @import("../source.zig");

pub const RunError = error{
    ParseFailed,
    CheckOrBackendFailed,
    CCompileFailed,
    ExecutableTerminated,
    UnexpectedExitCode,
} || std.mem.Allocator.Error || std.fs.File.OpenError || std.fs.File.WriteError || std.fs.Dir.MakeError || std.fs.Dir.OpenError || std.process.Child.SpawnError || std.process.Child.WaitError;

pub const RunResult = struct {
    expected_exit_code: u8,
    actual_exit_code: u8,
};

pub fn expectExitCode(allocator: std.mem.Allocator, source_text: []const u8, expected_exit_code: u8) RunError!RunResult {
    const actual_exit_code = try runSource(allocator, source_text);
    if (actual_exit_code != expected_exit_code) return error.UnexpectedExitCode;
    return .{
        .expected_exit_code = expected_exit_code,
        .actual_exit_code = actual_exit_code,
    };
}

pub fn runSource(allocator: std.mem.Allocator, source_text: []const u8) RunError!u8 {
    var parse_diagnostics = diagnostics_model.DiagnosticBag.init(allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = diagnostics_model.DiagnosticBag.init(allocator);
    defer check_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(allocator, "run-harness.concept", source_text);
    defer source_file.deinit(allocator);

    const unit = try parser_model.parseSource(allocator, source_file, &parse_diagnostics);
    defer unit.deinit(allocator);
    if (parse_diagnostics.count() != 0) return error.ParseFailed;

    const c_source = backend_c.emitExecutable(allocator, unit, &check_diagnostics) catch |err| switch (err) {
        error.InvalidExecutable => return error.CheckOrBackendFailed,
        error.OutOfMemory => return error.OutOfMemory,
    };
    defer allocator.free(c_source);
    if (check_diagnostics.count() != 0) return error.CheckOrBackendFailed;

    var tmp = std.testing.tmpDir(.{});
    defer tmp.cleanup();

    try tmp.dir.writeFile(.{ .sub_path = "main.c", .data = c_source });

    const exe_name = if (builtin.os.tag == .windows) "main.exe" else "main";
    try compileC(&tmp.dir, exe_name);
    return runExecutable(&tmp.dir, exe_name);
}

fn compileC(cwd: *std.fs.Dir, exe_name: []const u8) RunError!void {
    const argv = [_][]const u8{ "zig", "cc", "main.c", "-o", exe_name };
    var child = std.process.Child.init(&argv, std.testing.allocator);
    child.cwd_dir = cwd.*;
    child.stdin_behavior = .Ignore;
    child.stdout_behavior = .Ignore;
    child.stderr_behavior = .Ignore;

    const term = try child.spawnAndWait();
    switch (term) {
        .Exited => |code| if (code != 0) return error.CCompileFailed,
        else => return error.CCompileFailed,
    }
}

fn runExecutable(cwd: *std.fs.Dir, exe_name: []const u8) RunError!u8 {
    const exe_path = try std.fmt.allocPrint(std.testing.allocator, ".{c}{s}", .{ std.fs.path.sep, exe_name });
    defer std.testing.allocator.free(exe_path);

    const argv = [_][]const u8{exe_path};
    var child = std.process.Child.init(&argv, std.testing.allocator);
    child.cwd_dir = cwd.*;
    child.stdin_behavior = .Ignore;
    child.stdout_behavior = .Ignore;
    child.stderr_behavior = .Ignore;

    const term = try child.spawnAndWait();
    switch (term) {
        .Exited => |code| return code,
        else => return error.ExecutableTerminated,
    }
}

test "run harness compiles and runs return zero" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { return 0; }",
        0,
    );
    try std.testing.expectEqual(@as(u8, 0), result.actual_exit_code);
}

test "run harness compiles and runs arithmetic return" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { return 1 + 2 * 3; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs bool literal return" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { return true; }",
        1,
    );
    try std.testing.expectEqual(@as(u8, 1), result.actual_exit_code);
}

test "run harness reports checker backend failures" {
    try std.testing.expectError(
        error.CheckOrBackendFailed,
        expectExitCode(std.testing.allocator, "module Main; int main() { return; }", 0),
    );
}

test "run harness compiles and runs local arithmetic return" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 1 + 2; return x * 3; }",
        9,
    );
    try std.testing.expectEqual(@as(u8, 9), result.actual_exit_code);
}

test "run harness compiles and runs bool local return" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { bool ok = true; return ok; }",
        1,
    );
    try std.testing.expectEqual(@as(u8, 1), result.actual_exit_code);
}

test "run harness compiles and runs local int assignment" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 1; x = x + 2; return x; }",
        3,
    );
    try std.testing.expectEqual(@as(u8, 3), result.exit_code);
}

test "run harness compiles and runs local bool assignment" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { bool ok = false; ok = !ok; if (ok) { return 7; } return 0; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.exit_code);
}

test "run harness compiles and runs assignment inside if" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 1; if (true) { x = 5; } return x; }",
        5,
    );
    try std.testing.expectEqual(@as(u8, 5), result.exit_code);
}

test "run harness compiles and runs parameter assignment" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int bump(int x) { x = x + 1; return x; } int main() { return bump(2); }",
        3,
    );
    try std.testing.expectEqual(@as(u8, 3), result.exit_code);
}

test "run harness compiles and runs function call add" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int add(int a, int b) { return a + b; } int main() { return add(1, 2); }",
        3,
    );
    try std.testing.expectEqual(@as(u8, 3), result.actual_exit_code);
}

test "run harness compiles and runs function call using local" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int add_one(int x) { return x + 1; } int main() { int y = 2; return add_one(y); }",
        3,
    );
    try std.testing.expectEqual(@as(u8, 3), result.actual_exit_code);
}

test "run harness compiles and runs no-argument function call" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int three() { return 3; } int main() { return three(); }",
        3,
    );
    try std.testing.expectEqual(@as(u8, 3), result.actual_exit_code);
}

test "run harness compiles and runs forward function call" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { return later(); } int later() { return 3; }",
        3,
    );
    try std.testing.expectEqual(@as(u8, 3), result.actual_exit_code);
}

test "run harness compiles and runs if true return" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { if (true) { return 7; } return 0; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs if false else return" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { if (false) { return 1; } else { return 7; } }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs if comparison function" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int max(int a, int b) { if (a > b) { return a; } return b; } int main() { return max(3, 7); }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs outer local in if" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 7; if (true) { return x; } return 0; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs int match" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 2; match (x) { 1 => return 10; 2 => return 7; _ => return 0; } }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs bool match" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { bool ok = true; match (ok) { true => return 7; false => return 0; } return 0; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.actual_exit_code);
}

test "run harness compiles and runs default match arm" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 3; match (x) { 1 => return 10; 2 => return 7; _ => return 0; } }",
        0,
    );
    try std.testing.expectEqual(@as(u8, 0), result.actual_exit_code);
}

test "run harness compiles and runs while count to seven" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 0; while (x < 7) { x = x + 1; } return x; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.exit_code);
}

test "run harness compiles and runs while zero iterations" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 7; while (x < 7) { x = x + 1; } return x; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.exit_code);
}

test "run harness compiles and runs while with if" {
    const result = try expectExitCode(
        std.testing.allocator,
        "module Main; int main() { int x = 0; int y = 0; while (x < 4) { if (x == 2) { y = 7; } x = x + 1; } return y; }",
        7,
    );
    try std.testing.expectEqual(@as(u8, 7), result.exit_code);
}
