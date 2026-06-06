const std = @import("std");

pub const golden = @import("golden.zig");

pub const TestKind = enum {
    lexer_pass,
    lexer_fail,
    parser_pass,
    parser_fail,
    diagnostic,
    ast_snapshot,

    pub fn asString(self: TestKind) []const u8 {
        return switch (self) {
            .lexer_pass => "lexer_pass",
            .lexer_fail => "lexer_fail",
            .parser_pass => "parser_pass",
            .parser_fail => "parser_fail",
            .diagnostic => "diagnostic",
            .ast_snapshot => "ast_snapshot",
        };
    }

    pub fn format(
        self: TestKind,
        comptime fmt: []const u8,
        options: std.fmt.FormatOptions,
        writer: anytype,
    ) !void {
        _ = fmt;
        _ = options;
        try writer.writeAll(self.asString());
    }
};

pub const TestCase = struct {
    path: []const u8,
    kind: TestKind,
    expected_path: ?[]const u8 = null,
    expected_text: ?[]const u8 = null,

    pub fn init(path: []const u8, kind: TestKind) TestCase {
        return .{
            .path = path,
            .kind = kind,
        };
    }
};

pub const TestStatus = enum {
    pass,
    fail,
    skipped,
};

pub const TestResult = struct {
    status: TestStatus,
    message: ?[]const u8 = null,

    pub fn passed() TestResult {
        return .{ .status = .pass };
    }

    pub fn failed(message: []const u8) TestResult {
        return .{ .status = .fail, .message = message };
    }

    pub fn skipped(message: []const u8) TestResult {
        return .{ .status = .skipped, .message = message };
    }
};

test "golden comparison accepts equal strings" {
    const comparison = try golden.compare(std.testing.allocator, "same", "same");
    defer comparison.deinit(std.testing.allocator);

    try std.testing.expect(comparison.passed());
}

test "golden comparison reports mismatched strings" {
    const comparison = try golden.compare(std.testing.allocator, "expected", "actual");
    defer comparison.deinit(std.testing.allocator);

    try std.testing.expect(!comparison.passed());
    switch (comparison) {
        .equal => return error.ExpectedMismatch,
        .mismatch => |message| try std.testing.expect(std.mem.indexOf(u8, message, "golden mismatch") != null),
    }
}

test "test kind has stable string and formatting" {
    try std.testing.expectEqualStrings("parser_pass", TestKind.parser_pass.asString());

    var buffer: [32]u8 = undefined;
    const formatted = try std.fmt.bufPrint(&buffer, "{}", .{TestKind.diagnostic});
    try std.testing.expectEqualStrings("diagnostic", formatted);
}

test "test case can be constructed with path and kind" {
    const case = TestCase.init("tests/parser/pass/empty.con", .parser_pass);

    try std.testing.expectEqualStrings("tests/parser/pass/empty.con", case.path);
    try std.testing.expectEqual(TestKind.parser_pass, case.kind);
    try std.testing.expect(case.expected_path == null);
    try std.testing.expect(case.expected_text == null);
}
