const std = @import("std");

const hir = @import("hir.zig");

/// Stable identity for semantic types owned by a TypeStore.
///
/// Values are compact, copyable IDs. Callers can compare them by value, but the
/// backing type data remains owned by `TypeStore`.
pub const TypeId = struct {
    index: u32,

    pub fn format(self: TypeId, writer: *std.Io.Writer) std.Io.Writer.Error!void {
        try writer.print("TypeId({d})", .{self.index});
    }
};

pub const TypeKind = union(enum) {
    void,
    int,
    bool,
    struct_type: hir.StructId,
    enum_type: hir.EnumId,
    pointer: struct {
        pointee: TypeId,
    },

    pub fn format(self: TypeKind, writer: *std.Io.Writer) std.Io.Writer.Error!void {
        switch (self) {
            .void => try writer.writeAll("void"),
            .int => try writer.writeAll("int"),
            .bool => try writer.writeAll("bool"),
            .struct_type => |id| try writer.print("struct {f}", .{id}),
            .enum_type => |id| try writer.print("enum {f}", .{id}),
            .pointer => |pointer| try writer.print("{f}*", .{pointer.pointee}),
        }
    }
};

pub const TypeStore = struct {
    allocator: std.mem.Allocator,
    types: std.ArrayList(TypeKind),

    const void_type_id = TypeId{ .index = 0 };
    const int_type_id = TypeId{ .index = 1 };
    const bool_type_id = TypeId{ .index = 2 };

    pub fn init(allocator: std.mem.Allocator) !TypeStore {
        var store = TypeStore{
            .allocator = allocator,
            .types = std.ArrayList(TypeKind).empty,
        };
        errdefer store.deinit();

        try store.types.append(allocator, .void);
        try store.types.append(allocator, .int);
        try store.types.append(allocator, .bool);

        return store;
    }

    pub fn deinit(self: *TypeStore) void {
        self.types.deinit(self.allocator);
        self.* = undefined;
    }

    pub fn voidType(self: TypeStore) TypeId {
        _ = self;
        return void_type_id;
    }

    pub fn intType(self: TypeStore) TypeId {
        _ = self;
        return int_type_id;
    }

    pub fn boolType(self: TypeStore) TypeId {
        _ = self;
        return bool_type_id;
    }

    pub fn addStructType(self: *TypeStore, struct_id: hir.StructId) !TypeId {
        if (self.findNominal(.{ .struct_type = struct_id })) |existing| {
            return existing;
        }

        return try self.append(.{ .struct_type = struct_id }, error.TooManyTypes);
    }

    pub fn addEnumType(self: *TypeStore, enum_id: hir.EnumId) !TypeId {
        if (self.findNominal(.{ .enum_type = enum_id })) |existing| {
            return existing;
        }

        return try self.append(.{ .enum_type = enum_id }, error.TooManyTypes);
    }

    pub fn addPointerType(self: *TypeStore, pointee: TypeId) !TypeId {
        std.debug.assert(self.contains(pointee));
        if (self.findPointer(pointee)) |existing| {
            return existing;
        }

        return try self.append(.{ .pointer = .{ .pointee = pointee } }, error.TooManyTypes);
    }

    /// Returns the type kind for a valid type ID.
    ///
    /// Invalid IDs assert. Semantic callers should only carry IDs previously
    /// produced by this store instance.
    pub fn kind(self: TypeStore, id: TypeId) TypeKind {
        const index: usize = id.index;
        std.debug.assert(index < self.types.items.len);
        return self.types.items[index];
    }

    pub fn contains(self: TypeStore, id: TypeId) bool {
        return id.index < self.types.items.len;
    }

    pub fn count(self: TypeStore) usize {
        return self.types.items.len;
    }

    fn append(self: *TypeStore, type_kind: TypeKind, overflow_error: anyerror) !TypeId {
        const id = TypeId{ .index = try nextIndex(self.types.items.len, overflow_error) };
        try self.types.append(self.allocator, type_kind);
        return id;
    }

    fn findPointer(self: TypeStore, pointee: TypeId) ?TypeId {
        for (self.types.items, 0..) |candidate, index| {
            switch (candidate) {
                .pointer => |pointer| if (pointer.pointee.index == pointee.index) {
                    return .{ .index = @intCast(index) };
                },
                else => {},
            }
        }

        return null;
    }

    fn findNominal(self: TypeStore, needle: TypeKind) ?TypeId {
        for (self.types.items, 0..) |candidate, index| {
            switch (needle) {
                .struct_type => |needle_id| switch (candidate) {
                    .struct_type => |candidate_id| if (candidate_id.index == needle_id.index) {
                        return .{ .index = @intCast(index) };
                    },
                    else => {},
                },
                .enum_type => |needle_id| switch (candidate) {
                    .enum_type => |candidate_id| if (candidate_id.index == needle_id.index) {
                        return .{ .index = @intCast(index) };
                    },
                    else => {},
                },
                .pointer => unreachable,
                else => unreachable,
            }
        }

        return null;
    }
};

fn nextIndex(len: usize, overflow_error: anyerror) !u32 {
    if (len > std.math.maxInt(u32)) return overflow_error;
    return @intCast(len);
}

test "TypeId debug formatting" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f}", .{TypeId{ .index = 42 }});
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("TypeId(42)", rendered);
}

test "TypeIds compare by value" {
    try std.testing.expectEqual(TypeId{ .index = 7 }, TypeId{ .index = 7 });
    try std.testing.expect((TypeId{ .index = 7 }).index != (TypeId{ .index = 8 }).index);
}

test "builtin TypeIds are stable and distinct" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(u32, 0), store.voidType().index);
    try std.testing.expectEqual(@as(u32, 1), store.intType().index);
    try std.testing.expectEqual(@as(u32, 2), store.boolType().index);
    try std.testing.expect(store.intType().index != store.boolType().index);
    try std.testing.expectEqual(@as(usize, 3), store.count());
    try std.testing.expect(store.contains(store.voidType()));
    try std.testing.expect(store.contains(store.intType()));
    try std.testing.expect(store.contains(store.boolType()));
    try std.testing.expect(!store.contains(.{ .index = 99 }));
}

test "builtin kind lookup works" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(TypeKind, .void), store.kind(store.voidType()));
    try std.testing.expectEqual(@as(TypeKind, .int), store.kind(store.intType()));
    try std.testing.expectEqual(@as(TypeKind, .bool), store.kind(store.boolType()));
}

test "adding struct type stores nominal StructId" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const vec3_struct_id = hir.StructId{ .index = 0 };
    const vec3_type = try store.addStructType(vec3_struct_id);

    try std.testing.expectEqual(@as(u32, 3), vec3_type.index);
    try std.testing.expectEqual(TypeKind{ .struct_type = vec3_struct_id }, store.kind(vec3_type));
    try std.testing.expectEqual(@as(usize, 4), store.count());
}

test "adding enum type stores nominal EnumId" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const token_enum_id = hir.EnumId{ .index = 0 };
    const token_type = try store.addEnumType(token_enum_id);

    try std.testing.expectEqual(@as(u32, 3), token_type.index);
    try std.testing.expectEqual(TypeKind{ .enum_type = token_enum_id }, store.kind(token_type));
    try std.testing.expectEqual(@as(usize, 4), store.count());
}

test "repeated nominal adds return same TypeId" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const vec3_struct_id = hir.StructId{ .index = 0 };
    const token_enum_id = hir.EnumId{ .index = 0 };

    const vec3_first = try store.addStructType(vec3_struct_id);
    const token_first = try store.addEnumType(token_enum_id);
    const vec3_second = try store.addStructType(vec3_struct_id);
    const token_second = try store.addEnumType(token_enum_id);

    try std.testing.expectEqual(vec3_first, vec3_second);
    try std.testing.expectEqual(token_first, token_second);
    try std.testing.expectEqual(@as(usize, 5), store.count());
}

test "distinct nominal declarations get distinct TypeIds" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const vec3_type = try store.addStructType(.{ .index = 0 });
    const vec4_type = try store.addStructType(.{ .index = 1 });
    const token_type = try store.addEnumType(.{ .index = 0 });
    const keyword_type = try store.addEnumType(.{ .index = 1 });

    try std.testing.expect(vec3_type.index != vec4_type.index);
    try std.testing.expect(token_type.index != keyword_type.index);
    try std.testing.expectEqual(@as(usize, 7), store.count());
}

test "pointer TypeIds are interned by pointee" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const int_ptr_first = try store.addPointerType(store.intType());
    const int_ptr_second = try store.addPointerType(store.intType());
    const bool_ptr = try store.addPointerType(store.boolType());
    const void_ptr = try store.addPointerType(store.voidType());
    const int_ptr_ptr = try store.addPointerType(int_ptr_first);

    try std.testing.expectEqual(int_ptr_first, int_ptr_second);
    try std.testing.expect(int_ptr_first.index != bool_ptr.index);
    try std.testing.expectEqual(TypeKind{ .pointer = .{ .pointee = store.intType() } }, store.kind(int_ptr_first));
    try std.testing.expectEqual(TypeKind{ .pointer = .{ .pointee = store.voidType() } }, store.kind(void_ptr));
    try std.testing.expectEqual(TypeKind{ .pointer = .{ .pointee = int_ptr_first } }, store.kind(int_ptr_ptr));
}

test "pointer TypeKind debug formatting is stable" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f}", .{TypeKind{ .pointer = .{ .pointee = .{ .index = 1 } } }});
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("TypeId(1)*", rendered);
}

test "invalid TypeIds assert by contract" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(usize, 3), store.count());
    // `kind` documents that invalid IDs assert; this test keeps that contract
    // visible without intentionally triggering a process abort.
}

test "TypeKind debug formatting is stable" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f} {f} {f} {f} {f} {f}", .{
        @as(TypeKind, .void),
        @as(TypeKind, .int),
        @as(TypeKind, .bool),
        TypeKind{ .struct_type = .{ .index = 0 } },
        TypeKind{ .enum_type = .{ .index = 0 } },
        TypeKind{ .pointer = .{ .pointee = .{ .index = 1 } } },
    });
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("void int bool struct StructId(0) enum EnumId(0) TypeId(1)*", rendered);
}
