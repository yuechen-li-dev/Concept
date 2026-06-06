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
