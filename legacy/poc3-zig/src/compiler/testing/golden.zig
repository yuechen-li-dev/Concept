const std = @import("std");

pub const Comparison = union(enum) {
    equal,
    mismatch: []const u8,

    pub fn passed(self: Comparison) bool {
        return switch (self) {
            .equal => true,
            .mismatch => false,
        };
    }

    pub fn deinit(self: Comparison, allocator: std.mem.Allocator) void {
        switch (self) {
            .equal => {},
            .mismatch => |message| allocator.free(message),
        }
    }
};

pub fn compare(allocator: std.mem.Allocator, expected: []const u8, actual: []const u8) !Comparison {
    if (std.mem.eql(u8, expected, actual)) {
        return .equal;
    }

    const index = firstDifference(expected, actual);
    const message = try std.fmt.allocPrint(
        allocator,
        "golden mismatch at byte {d}: expected length {d}, actual length {d}",
        .{ index, expected.len, actual.len },
    );

    return .{ .mismatch = message };
}

fn firstDifference(expected: []const u8, actual: []const u8) usize {
    const shared_len = @min(expected.len, actual.len);
    for (expected[0..shared_len], actual[0..shared_len], 0..) |expected_byte, actual_byte, index| {
        if (expected_byte != actual_byte) {
            return index;
        }
    }

    return shared_len;
}
