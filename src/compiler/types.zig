const std = @import("std");

const hir = @import("hir.zig");
const interner = @import("interner.zig");

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

pub const TypeStoreError = std.mem.Allocator.Error || error{TooManyTypes};

pub const TypeParamOwner = struct {
    kind: enum { generic_function, concept },
    index: u32,
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
    type_param: struct {
        owner: TypeParamOwner,
        index: u32,
        name: interner.SymbolId,
    },

    pub fn format(self: TypeKind, writer: *std.Io.Writer) std.Io.Writer.Error!void {
        switch (self) {
            .void => try writer.writeAll("void"),
            .int => try writer.writeAll("int"),
            .bool => try writer.writeAll("bool"),
            .struct_type => |id| try writer.print("struct {f}", .{id}),
            .enum_type => |id| try writer.print("enum {f}", .{id}),
            .pointer => |pointer| try writer.print("{f}*", .{pointer.pointee}),
            .type_param => |param| try writer.print("type_param({s}:{d}/{d} {f})", .{ @tagName(param.owner.kind), param.owner.index, param.index, param.name }),
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

    pub fn addStructType(self: *TypeStore, struct_id: hir.StructId) TypeStoreError!TypeId {
        if (self.findNominal(.{ .struct_type = struct_id })) |existing| {
            return existing;
        }

        return try self.append(.{ .struct_type = struct_id }, error.TooManyTypes);
    }

    pub fn addEnumType(self: *TypeStore, enum_id: hir.EnumId) TypeStoreError!TypeId {
        if (self.findNominal(.{ .enum_type = enum_id })) |existing| {
            return existing;
        }

        return try self.append(.{ .enum_type = enum_id }, error.TooManyTypes);
    }

    pub fn addTypeParam(self: *TypeStore, owner: TypeParamOwner, index: u32, name: interner.SymbolId) TypeStoreError!TypeId {
        if (self.findTypeParam(owner, index, name)) |existing| {
            return existing;
        }

        return try self.append(.{ .type_param = .{ .owner = owner, .index = index, .name = name } }, error.TooManyTypes);
    }

    pub fn addPointerType(self: *TypeStore, pointee: TypeId) TypeStoreError!TypeId {
        std.debug.assert(self.contains(pointee));
        if (self.findPointer(pointee)) |existing| {
            return existing;
        }

        return try self.append(.{ .pointer = .{ .pointee = pointee } }, error.TooManyTypes);
    }

    pub fn pointerType(self: TypeStore, pointee: TypeId) ?TypeId {
        std.debug.assert(self.contains(pointee));
        return self.findPointer(pointee);
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

    /// P10-M3 copyability model v0.
    ///
    /// Integer and boolean values are intrinsic Copy. Raw pointers and enums are
    /// Copy for now because existing runtime semantics treat them as ordinary
    /// scalar/tag value flow. Structs are non-Copy by default until marker
    /// concept `Copy<T>` satisfaction is integrated deliberately.
    pub fn isCopyType(self: TypeStore, hir_store: ?*const hir.HirStore, id: TypeId) bool {
        return switch (self.kind(id)) {
            .int, .bool, .pointer, .enum_type => true,
            .struct_type => self.hasCopyMarkerImpl(hir_store, id),
            .void, .type_param => false,
        };
    }

    fn hasCopyMarkerImpl(self: TypeStore, hir_store: ?*const hir.HirStore, id: TypeId) bool {
        _ = self;
        const store = hir_store orelse return false;
        for (store.concepts.items, 0..) |concept, index| {
            if (!concept.is_marker or concept.known_marker_kind != .copy) continue;
            if (store.hasConceptImpl(.{ .index = @intCast(index) }, id)) return true;
        }
        return false;
    }

    pub fn contains(self: TypeStore, id: TypeId) bool {
        return id.index < self.types.items.len;
    }

    pub fn count(self: TypeStore) usize {
        return self.types.items.len;
    }

    fn append(self: *TypeStore, type_kind: TypeKind, overflow_error: TypeStoreError) TypeStoreError!TypeId {
        const id = TypeId{ .index = try nextIndex(self.types.items.len, overflow_error) };
        try self.types.append(self.allocator, type_kind);
        return id;
    }

    fn findTypeParam(self: TypeStore, owner: TypeParamOwner, param_index: u32, name: interner.SymbolId) ?TypeId {
        for (self.types.items, 0..) |candidate, index| {
            switch (candidate) {
                .type_param => |param| if (param.owner.kind == owner.kind and param.owner.index == owner.index and param.index == param_index and param.name.index == name.index) {
                    return .{ .index = @intCast(index) };
                },
                else => {},
            }
        }

        return null;
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
                .pointer, .type_param => unreachable,
                else => unreachable,
            }
        }

        return null;
    }
};

fn nextIndex(len: usize, overflow_error: TypeStoreError) TypeStoreError!u32 {
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

test "copyability model v0 marks scalars pointers and enums Copy but structs non-Copy" {
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const struct_type = try store.addStructType(.{ .index = 0 });
    const enum_type = try store.addEnumType(.{ .index = 0 });
    const pointer_type = try store.addPointerType(store.intType());

    try std.testing.expect(store.isCopyType(null, store.intType()));
    try std.testing.expect(store.isCopyType(null, store.boolType()));
    try std.testing.expect(store.isCopyType(null, pointer_type));
    try std.testing.expect(store.isCopyType(null, enum_type));
    try std.testing.expect(!store.isCopyType(null, struct_type));
    try std.testing.expect(!store.isCopyType(null, store.voidType()));
}

test "copyability model recognizes explicit marker Copy impl for structs" {
    var symbols = interner.Interner.init(std.testing.allocator);
    defer symbols.deinit();
    var hir_module = hir.HirModule.init(std.testing.allocator);
    defer hir_module.deinit();
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const copy_concept = try hir_module.store.addConcept(try symbols.intern("Copy"), true, false, hir.synthetic_span);
    hir_module.store.setConceptKnownMarkerKind(copy_concept, .copy);
    const struct_id = try hir_module.store.addStruct(try symbols.intern("Vec2"));
    const struct_type = try store.addStructType(struct_id);

    try std.testing.expect(!store.isCopyType(&hir_module.store, struct_type));
    _ = try hir_module.store.addConceptImpl(copy_concept, struct_type, &.{}, false, hir.synthetic_span);
    try std.testing.expect(store.isCopyType(&hir_module.store, struct_type));
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

test "type parameter TypeIds are stable and distinct by owner and index" {
    var symbols = interner.Interner.init(std.testing.allocator);
    defer symbols.deinit();
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const t = try symbols.intern("T");
    const owner0 = TypeParamOwner{ .kind = .generic_function, .index = 0 };
    const owner1 = TypeParamOwner{ .kind = .generic_function, .index = 1 };
    const concept_owner = TypeParamOwner{ .kind = .concept, .index = 0 };

    const first = try store.addTypeParam(owner0, 0, t);
    const again = try store.addTypeParam(owner0, 0, t);
    const other_owner = try store.addTypeParam(owner1, 0, t);
    const other_kind = try store.addTypeParam(concept_owner, 0, t);
    const other_index = try store.addTypeParam(owner0, 1, t);

    try std.testing.expectEqual(first, again);
    try std.testing.expect(first.index != other_owner.index);
    try std.testing.expect(first.index != other_kind.index);
    try std.testing.expect(first.index != other_index.index);
    try std.testing.expectEqual(TypeKind{ .type_param = .{ .owner = owner0, .index = 0, .name = t } }, store.kind(first));
}

test "pointer to type parameter is interned" {
    var symbols = interner.Interner.init(std.testing.allocator);
    defer symbols.deinit();
    var store = try TypeStore.init(std.testing.allocator);
    defer store.deinit();

    const param = try store.addTypeParam(.{ .kind = .generic_function, .index = 0 }, 0, try symbols.intern("T"));
    const first = try store.addPointerType(param);
    const second = try store.addPointerType(param);

    try std.testing.expectEqual(first, second);
    try std.testing.expectEqual(TypeKind{ .pointer = .{ .pointee = param } }, store.kind(first));
}

test "type parameter debug formatting is stable" {
    var symbols = interner.Interner.init(std.testing.allocator);
    defer symbols.deinit();
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f}", .{TypeKind{ .type_param = .{
        .owner = .{ .kind = .generic_function, .index = 3 },
        .index = 2,
        .name = try symbols.intern("T"),
    } }});
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("type_param(generic_function:3/2 SymbolId(0))", rendered);
}
