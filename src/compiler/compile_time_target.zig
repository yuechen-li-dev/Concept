const std = @import("std");

const types = @import("types.zig");

/// Compiler-owned target metadata visible to compile-time evaluation.
///
/// P9-M8 intentionally treats this as configured compiler metadata, not as a
/// query against the ambient host machine. The default below is a deterministic
/// scaffold matching the current Stage 0 C-backend assumptions until a real
/// build-target configuration object exists.
pub const CompileTimeTargetInfo = struct {
    pointer_size: i64,
    is_little_endian: bool,
    is_big_endian: bool,
};

pub const CompileTimeTargetQuery = enum {
    pointer_size,
    is_little_endian,
    is_big_endian,

    pub fn typeOf(self: CompileTimeTargetQuery, type_store: types.TypeStore) types.TypeId {
        return switch (self) {
            .pointer_size => type_store.intType(),
            .is_little_endian, .is_big_endian => type_store.boolType(),
        };
    }

    pub fn sourceName(self: CompileTimeTargetQuery) []const u8 {
        return switch (self) {
            .pointer_size => "pointerSize",
            .is_little_endian => "isLittleEndian",
            .is_big_endian => "isBigEndian",
        };
    }
};

pub const CompileTimeTargetMetadata = union(enum) {
    int: i64,
    bool: bool,
};

pub const CompileTimeTargetUnsupportedQuery = error{UnsupportedTargetQuery};

pub fn defaultTargetInfo() CompileTimeTargetInfo {
    return .{
        .pointer_size = 8,
        .is_little_endian = true,
        .is_big_endian = false,
    };
}

pub fn lookupTargetQuery(source_name: []const u8) ?CompileTimeTargetQuery {
    if (std.mem.eql(u8, source_name, "pointerSize")) return .pointer_size;
    if (std.mem.eql(u8, source_name, "isLittleEndian")) return .is_little_endian;
    if (std.mem.eql(u8, source_name, "isBigEndian")) return .is_big_endian;
    return null;
}

pub fn queryTargetInfo(info: CompileTimeTargetInfo, query: CompileTimeTargetQuery) CompileTimeTargetMetadata {
    return switch (query) {
        .pointer_size => .{ .int = info.pointer_size },
        .is_little_endian => .{ .bool = info.is_little_endian },
        .is_big_endian => .{ .bool = info.is_big_endian },
    };
}

test "default target info is deterministic scaffold" {
    const info = defaultTargetInfo();
    try std.testing.expectEqual(@as(i64, 8), info.pointer_size);
    try std.testing.expect(info.is_little_endian);
    try std.testing.expect(!info.is_big_endian);
}

test "target field lookup accepts supported source names" {
    try std.testing.expectEqual(CompileTimeTargetQuery.pointer_size, lookupTargetQuery("pointerSize").?);
    try std.testing.expectEqual(CompileTimeTargetQuery.is_little_endian, lookupTargetQuery("isLittleEndian").?);
    try std.testing.expectEqual(CompileTimeTargetQuery.is_big_endian, lookupTargetQuery("isBigEndian").?);
}

test "target field lookup rejects unknown source names" {
    try std.testing.expect(lookupTargetQuery("arch") == null);
    try std.testing.expect(lookupTargetQuery("nope") == null);
}
