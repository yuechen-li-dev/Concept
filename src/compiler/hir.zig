const std = @import("std");

const interner_module = @import("interner.zig");
pub const Interner = interner_module.Interner;
pub const SymbolId = interner_module.SymbolId;

pub const ItemId = SemanticId("ItemId");
pub const FunctionId = SemanticId("FunctionId");
pub const StructId = SemanticId("StructId");
pub const EnumId = SemanticId("EnumId");
pub const FieldId = SemanticId("FieldId");
pub const VariantId = SemanticId("VariantId");
pub const LocalId = SemanticId("LocalId");
pub const ParamId = SemanticId("ParamId");

fn SemanticId(comptime label: []const u8) type {
    return struct {
        index: u32,

        pub fn format(self: @This(), writer: *std.Io.Writer) std.Io.Writer.Error!void {
            try writer.print(label ++ "({d})", .{self.index});
        }
    };
}

pub const HirItem = union(enum) {
    function: FunctionId,
    struct_: StructId,
    enum_: EnumId,
};

pub const HirFunction = struct {
    item: ItemId,
    name: SymbolId,
    params: []ParamId,
    locals: []LocalId,
};

pub const HirStruct = struct {
    item: ItemId,
    name: SymbolId,
    fields: []FieldId,
};

pub const HirEnum = struct {
    item: ItemId,
    name: SymbolId,
    variants: []VariantId,
};

pub const HirField = struct {
    parent: StructId,
    name: SymbolId,
};

pub const HirVariant = struct {
    parent: EnumId,
    name: SymbolId,
};

pub const HirModule = struct {
    store: HirStore,

    pub fn init(allocator: std.mem.Allocator) HirModule {
        return .{ .store = HirStore.init(allocator) };
    }

    pub fn deinit(self: *HirModule) void {
        self.store.deinit();
        self.* = undefined;
    }
};

pub const HirStore = struct {
    allocator: std.mem.Allocator,
    items: std.ArrayList(HirItem),
    functions: std.ArrayList(HirFunction),
    structs: std.ArrayList(HirStruct),
    enums: std.ArrayList(HirEnum),
    fields: std.ArrayList(HirField),
    variants: std.ArrayList(HirVariant),

    pub fn init(allocator: std.mem.Allocator) HirStore {
        return .{
            .allocator = allocator,
            .items = std.ArrayList(HirItem).empty,
            .functions = std.ArrayList(HirFunction).empty,
            .structs = std.ArrayList(HirStruct).empty,
            .enums = std.ArrayList(HirEnum).empty,
            .fields = std.ArrayList(HirField).empty,
            .variants = std.ArrayList(HirVariant).empty,
        };
    }

    pub fn deinit(self: *HirStore) void {
        for (self.functions.items) |function| {
            if (function.params.len > 0) self.allocator.free(function.params);
            if (function.locals.len > 0) self.allocator.free(function.locals);
        }
        for (self.structs.items) |struct_decl| {
            if (struct_decl.fields.len > 0) self.allocator.free(struct_decl.fields);
        }
        for (self.enums.items) |enum_decl| {
            if (enum_decl.variants.len > 0) self.allocator.free(enum_decl.variants);
        }

        self.variants.deinit(self.allocator);
        self.fields.deinit(self.allocator);
        self.enums.deinit(self.allocator);
        self.structs.deinit(self.allocator);
        self.functions.deinit(self.allocator);
        self.items.deinit(self.allocator);
        self.* = undefined;
    }

    pub fn addFunction(self: *HirStore, name: SymbolId) !FunctionId {
        const id = FunctionId{ .index = try nextIndex(self.functions.items.len, error.TooManyFunctions) };
        const item = try self.addItem(.{ .function = id });
        errdefer _ = self.items.pop();
        try self.functions.append(self.allocator, .{
            .item = item,
            .name = name,
            .params = &.{},
            .locals = &.{},
        });
        return id;
    }

    pub fn addStruct(self: *HirStore, name: SymbolId) !StructId {
        const id = StructId{ .index = try nextIndex(self.structs.items.len, error.TooManyStructs) };
        const item = try self.addItem(.{ .struct_ = id });
        errdefer _ = self.items.pop();
        try self.structs.append(self.allocator, .{
            .item = item,
            .name = name,
            .fields = &.{},
        });
        return id;
    }

    pub fn addEnum(self: *HirStore, name: SymbolId) !EnumId {
        const id = EnumId{ .index = try nextIndex(self.enums.items.len, error.TooManyEnums) };
        const item = try self.addItem(.{ .enum_ = id });
        errdefer _ = self.items.pop();
        try self.enums.append(self.allocator, .{
            .item = item,
            .name = name,
            .variants = &.{},
        });
        return id;
    }

    pub fn addField(self: *HirStore, parent: StructId, name: SymbolId) !FieldId {
        _ = self.getStruct(parent);
        const id = FieldId{ .index = try nextIndex(self.fields.items.len, error.TooManyFields) };
        try self.fields.append(self.allocator, .{ .parent = parent, .name = name });
        errdefer _ = self.fields.pop();

        const struct_decl = self.getStructMut(parent);
        struct_decl.fields = try appendId(self.allocator, FieldId, struct_decl.fields, id);
        return id;
    }

    pub fn addVariant(self: *HirStore, parent: EnumId, name: SymbolId) !VariantId {
        _ = self.getEnum(parent);
        const id = VariantId{ .index = try nextIndex(self.variants.items.len, error.TooManyVariants) };
        try self.variants.append(self.allocator, .{ .parent = parent, .name = name });
        errdefer _ = self.variants.pop();

        const enum_decl = self.getEnumMut(parent);
        enum_decl.variants = try appendId(self.allocator, VariantId, enum_decl.variants, id);
        return id;
    }

    pub fn getItem(self: *const HirStore, id: ItemId) *const HirItem {
        const index: usize = id.index;
        std.debug.assert(index < self.items.items.len);
        return &self.items.items[index];
    }

    pub fn getFunction(self: *const HirStore, id: FunctionId) *const HirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    pub fn getStruct(self: *const HirStore, id: StructId) *const HirStruct {
        const index: usize = id.index;
        std.debug.assert(index < self.structs.items.len);
        return &self.structs.items[index];
    }

    pub fn getEnum(self: *const HirStore, id: EnumId) *const HirEnum {
        const index: usize = id.index;
        std.debug.assert(index < self.enums.items.len);
        return &self.enums.items[index];
    }

    pub fn getField(self: *const HirStore, id: FieldId) *const HirField {
        const index: usize = id.index;
        std.debug.assert(index < self.fields.items.len);
        return &self.fields.items[index];
    }

    pub fn getVariant(self: *const HirStore, id: VariantId) *const HirVariant {
        const index: usize = id.index;
        std.debug.assert(index < self.variants.items.len);
        return &self.variants.items[index];
    }

    fn getStructMut(self: *HirStore, id: StructId) *HirStruct {
        const index: usize = id.index;
        std.debug.assert(index < self.structs.items.len);
        return &self.structs.items[index];
    }

    fn getEnumMut(self: *HirStore, id: EnumId) *HirEnum {
        const index: usize = id.index;
        std.debug.assert(index < self.enums.items.len);
        return &self.enums.items[index];
    }

    pub fn debugString(self: *const HirStore, allocator: std.mem.Allocator, interner: Interner) ![]u8 {
        var buffer: std.Io.Writer.Allocating = .init(allocator);
        errdefer buffer.deinit();
        const writer = &buffer.writer;

        try writer.writeAll("HirModule\n");
        for (self.items.items) |item| {
            switch (item) {
                .function => |id| {
                    const function = self.getFunction(id);
                    try writer.print("  Function {s}\n", .{interner.text(function.name)});
                },
                .struct_ => |id| {
                    const struct_decl = self.getStruct(id);
                    try writer.print("  Struct {s}\n", .{interner.text(struct_decl.name)});
                    for (struct_decl.fields) |field_id| {
                        const field = self.getField(field_id);
                        try writer.print("    Field {s}\n", .{interner.text(field.name)});
                    }
                },
                .enum_ => |id| {
                    const enum_decl = self.getEnum(id);
                    try writer.print("  Enum {s}\n", .{interner.text(enum_decl.name)});
                    for (enum_decl.variants) |variant_id| {
                        const variant = self.getVariant(variant_id);
                        try writer.print("    Variant {s}\n", .{interner.text(variant.name)});
                    }
                },
            }
        }

        return try buffer.toOwnedSlice();
    }

    fn addItem(self: *HirStore, item: HirItem) !ItemId {
        const id = ItemId{ .index = try nextIndex(self.items.items.len, error.TooManyItems) };
        try self.items.append(self.allocator, item);
        return id;
    }
};

fn nextIndex(len: usize, overflow_error: anyerror) !u32 {
    if (len > std.math.maxInt(u32)) return overflow_error;
    return @intCast(len);
}

fn appendId(allocator: std.mem.Allocator, comptime Id: type, existing: []Id, id: Id) ![]Id {
    const expanded = try allocator.alloc(Id, existing.len + 1);
    @memcpy(expanded[0..existing.len], existing);
    expanded[existing.len] = id;
    if (existing.len > 0) allocator.free(existing);
    return expanded;
}

test "semantic ID debug formatting" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f} {f} {f} {f}", .{
        ItemId{ .index = 1 },
        FunctionId{ .index = 2 },
        FieldId{ .index = 3 },
        ParamId{ .index = 4 },
    });
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("ItemId(1) FunctionId(2) FieldId(3) ParamId(4)", rendered);
}

test "semantic IDs compare by value" {
    try std.testing.expectEqual(FunctionId{ .index = 7 }, FunctionId{ .index = 7 });
    try std.testing.expectEqual(StructId{ .index = 8 }, StructId{ .index = 8 });
    try std.testing.expectEqual(EnumId{ .index = 9 }, EnumId{ .index = 9 });
}

test "empty HIR store" {
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(usize, 0), store.items.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.functions.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.structs.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.enums.items.len);
}

test "add function with interned name and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const main_name = try interner.intern("main");
    const function_id = try store.addFunction(main_name);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(u32, 0), function_id.index);
    try std.testing.expectEqual(main_name, function.name);
    try std.testing.expectEqualStrings("main", interner.text(function.name));
    try std.testing.expectEqual(@as(usize, 0), function.params.len);
    try std.testing.expectEqual(@as(usize, 0), function.locals.len);
    try std.testing.expectEqual(HirItem{ .function = function_id }, store.getItem(function.item).*);
}

test "add struct with fields and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const vec3_id = try store.addStruct(try interner.intern("Vec3"));
    const x_id = try store.addField(vec3_id, try interner.intern("x"));
    const y_id = try store.addField(vec3_id, try interner.intern("y"));
    const z_id = try store.addField(vec3_id, try interner.intern("z"));
    const vec3 = store.getStruct(vec3_id);

    try std.testing.expectEqual(@as(usize, 3), vec3.fields.len);
    try std.testing.expectEqual(x_id, vec3.fields[0]);
    try std.testing.expectEqual(y_id, vec3.fields[1]);
    try std.testing.expectEqual(z_id, vec3.fields[2]);
    try std.testing.expectEqual(vec3_id, store.getField(x_id).parent);
    try std.testing.expectEqualStrings("x", interner.text(store.getField(x_id).name));
}

test "add enum with variants and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const token_id = try store.addEnum(try interner.intern("Token"));
    const identifier_id = try store.addVariant(token_id, try interner.intern("Identifier"));
    const end_id = try store.addVariant(token_id, try interner.intern("End"));
    const token = store.getEnum(token_id);

    try std.testing.expectEqual(@as(usize, 2), token.variants.len);
    try std.testing.expectEqual(identifier_id, token.variants[0]);
    try std.testing.expectEqual(end_id, token.variants[1]);
    try std.testing.expectEqual(token_id, store.getVariant(end_id).parent);
    try std.testing.expectEqualStrings("End", interner.text(store.getVariant(end_id).name));
}

test "HIR debug formatting uses interned names" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var module = HirModule.init(std.testing.allocator);
    defer module.deinit();

    _ = try module.store.addFunction(try interner.intern("main"));
    const vec3_id = try module.store.addStruct(try interner.intern("Vec3"));
    _ = try module.store.addField(vec3_id, try interner.intern("x"));
    _ = try module.store.addField(vec3_id, try interner.intern("y"));
    _ = try module.store.addField(vec3_id, try interner.intern("z"));
    const token_id = try module.store.addEnum(try interner.intern("Token"));
    _ = try module.store.addVariant(token_id, try interner.intern("Identifier"));
    _ = try module.store.addVariant(token_id, try interner.intern("End"));

    const rendered = try module.store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Function main
        \\  Struct Vec3
        \\    Field x
        \\    Field y
        \\    Field z
        \\  Enum Token
        \\    Variant Identifier
        \\    Variant End
        \\
    , rendered);
}
