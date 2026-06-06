const std = @import("std");

const hir = @import("hir.zig");
const interner_module = @import("interner.zig");
const source = @import("source.zig");
const types = @import("types.zig");

pub const Interner = interner_module.Interner;
pub const SourceSpan = source.SourceSpan;

pub const MirFunctionId = MirId("MirFunctionId");
pub const MirLocalId = MirId("MirLocalId");
pub const MirBlockId = MirId("MirBlockId");

fn MirId(comptime label: []const u8) type {
    return struct {
        index: u32,

        pub fn format(self: @This(), writer: *std.Io.Writer) std.Io.Writer.Error!void {
            try writer.print(label ++ "({d})", .{self.index});
        }
    };
}

pub const MirLocalKind = enum {
    param,
    user,
    temp,
};

pub const MirModule = struct {
    store: MirStore,

    pub fn init(allocator: std.mem.Allocator) MirModule {
        return .{ .store = MirStore.init(allocator) };
    }

    pub fn deinit(self: *MirModule) void {
        self.store.deinit();
        self.* = undefined;
    }
};

pub const MirFunction = struct {
    hir_function: hir.FunctionId,
    name: hir.SymbolId,
    return_type: types.TypeId,
    params: []MirLocalId,
    locals: []MirLocalId,
    blocks: []MirBlockId,
    source_span: ?SourceSpan,
};

pub const MirLocal = struct {
    parent: MirFunctionId,
    name: ?hir.SymbolId,
    kind: MirLocalKind,
    type_id: types.TypeId,
    source_span: ?SourceSpan,
};

pub const MirBlock = struct {
    parent: MirFunctionId,
    source_span: ?SourceSpan,
};

pub const MirStore = struct {
    allocator: std.mem.Allocator,
    functions: std.ArrayList(MirFunction),
    locals: std.ArrayList(MirLocal),
    blocks: std.ArrayList(MirBlock),

    pub fn init(allocator: std.mem.Allocator) MirStore {
        return .{
            .allocator = allocator,
            .functions = std.ArrayList(MirFunction).empty,
            .locals = std.ArrayList(MirLocal).empty,
            .blocks = std.ArrayList(MirBlock).empty,
        };
    }

    pub fn deinit(self: *MirStore) void {
        for (self.functions.items) |function| {
            if (function.params.len > 0) self.allocator.free(function.params);
            if (function.locals.len > 0) self.allocator.free(function.locals);
            if (function.blocks.len > 0) self.allocator.free(function.blocks);
        }

        self.blocks.deinit(self.allocator);
        self.locals.deinit(self.allocator);
        self.functions.deinit(self.allocator);
        self.* = undefined;
    }

    pub fn addFunction(
        self: *MirStore,
        hir_function: hir.FunctionId,
        name: hir.SymbolId,
        return_type: types.TypeId,
        source_span: ?SourceSpan,
    ) !MirFunctionId {
        const id = MirFunctionId{ .index = try nextIndex(self.functions.items.len, error.TooManyMirFunctions) };
        try self.functions.append(self.allocator, .{
            .hir_function = hir_function,
            .name = name,
            .return_type = return_type,
            .params = &.{},
            .locals = &.{},
            .blocks = &.{},
            .source_span = source_span,
        });
        return id;
    }

    pub fn addLocal(
        self: *MirStore,
        parent: MirFunctionId,
        name: ?hir.SymbolId,
        kind: MirLocalKind,
        type_id: types.TypeId,
        source_span: ?SourceSpan,
    ) !MirLocalId {
        _ = self.getFunction(parent);
        const id = MirLocalId{ .index = try nextIndex(self.locals.items.len, error.TooManyMirLocals) };
        try self.locals.append(self.allocator, .{
            .parent = parent,
            .name = name,
            .kind = kind,
            .type_id = type_id,
            .source_span = source_span,
        });
        errdefer _ = self.locals.pop();

        const function = self.getFunctionMut(parent);
        switch (kind) {
            .param => function.params = try appendId(self.allocator, MirLocalId, function.params, id),
            .user, .temp => function.locals = try appendId(self.allocator, MirLocalId, function.locals, id),
        }
        return id;
    }

    pub fn addBlock(
        self: *MirStore,
        parent: MirFunctionId,
        source_span: ?SourceSpan,
    ) !MirBlockId {
        _ = self.getFunction(parent);
        const id = MirBlockId{ .index = try nextIndex(self.blocks.items.len, error.TooManyMirBlocks) };
        try self.blocks.append(self.allocator, .{
            .parent = parent,
            .source_span = source_span,
        });
        errdefer _ = self.blocks.pop();

        const function = self.getFunctionMut(parent);
        function.blocks = try appendId(self.allocator, MirBlockId, function.blocks, id);
        return id;
    }

    pub fn getFunction(self: *const MirStore, id: MirFunctionId) *const MirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    pub fn getFunctionMut(self: *MirStore, id: MirFunctionId) *MirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    pub fn getLocal(self: *const MirStore, id: MirLocalId) *const MirLocal {
        const index: usize = id.index;
        std.debug.assert(index < self.locals.items.len);
        return &self.locals.items[index];
    }

    pub fn getBlock(self: *const MirStore, id: MirBlockId) *const MirBlock {
        const index: usize = id.index;
        std.debug.assert(index < self.blocks.items.len);
        return &self.blocks.items[index];
    }

    pub fn debugString(self: *const MirStore, allocator: std.mem.Allocator, interner: Interner) ![]u8 {
        var buffer: std.Io.Writer.Allocating = .init(allocator);
        errdefer buffer.deinit();
        const writer = &buffer.writer;

        try writer.writeAll("MirModule\n");
        for (self.functions.items) |function| {
            try writer.print("  Function {s} -> {f}\n", .{ interner.text(function.name), function.return_type });
            if (function.params.len != 0) {
                try writer.writeAll("    Params\n");
                for (function.params) |local_id| {
                    try self.writeLocalDebug(writer, local_id, interner);
                }
            }
            if (function.locals.len != 0) {
                try writer.writeAll("    Locals\n");
                for (function.locals) |local_id| {
                    try self.writeLocalDebug(writer, local_id, interner);
                }
            }
            if (function.blocks.len != 0) {
                try writer.writeAll("    Blocks\n");
                for (function.blocks) |block_id| {
                    try writer.print("      {f}\n", .{block_id});
                }
            }
        }

        return try buffer.toOwnedSlice();
    }

    fn writeLocalDebug(self: *const MirStore, writer: *std.Io.Writer, id: MirLocalId, interner: Interner) !void {
        const local = self.getLocal(id);
        const name = if (local.name) |symbol| interner.text(symbol) else "<temp>";
        try writer.print("      {f} {s} {s}: {f}\n", .{ id, @tagName(local.kind), name, local.type_id });
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

const testing_span = SourceSpan{ .start = 3, .length = 5 };

test "MIR ID formatting and equality" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f} {f} {f}", .{
        MirFunctionId{ .index = 1 },
        MirLocalId{ .index = 2 },
        MirBlockId{ .index = 3 },
    });
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("MirFunctionId(1) MirLocalId(2) MirBlockId(3)", rendered);
    try std.testing.expectEqual(MirFunctionId{ .index = 7 }, MirFunctionId{ .index = 7 });
    try std.testing.expectEqual(MirLocalId{ .index = 8 }, MirLocalId{ .index = 8 });
    try std.testing.expectEqual(MirBlockId{ .index = 9 }, MirBlockId{ .index = 9 });
}

test "empty MIR store" {
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(usize, 0), store.functions.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.locals.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.blocks.items.len);
}

test "add MIR function and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    const main_name = try interner.intern("main");
    const function_id = try store.addFunction(.{ .index = 4 }, main_name, .{ .index = 1 }, testing_span);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(u32, 0), function_id.index);
    try std.testing.expectEqual(hir.FunctionId{ .index = 4 }, function.hir_function);
    try std.testing.expectEqual(main_name, function.name);
    try std.testing.expectEqual(types.TypeId{ .index = 1 }, function.return_type);
    try std.testing.expectEqual(testing_span, function.source_span.?);
    try std.testing.expectEqualStrings("main", interner.text(function.name));
    try std.testing.expectEqual(@as(usize, 0), function.params.len);
    try std.testing.expectEqual(@as(usize, 0), function.locals.len);
    try std.testing.expectEqual(@as(usize, 0), function.blocks.len);
}

test "add MIR locals and associate with function" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const argc_id = try store.addLocal(function_id, try interner.intern("argc"), .param, .{ .index = 1 }, testing_span);
    const x_id = try store.addLocal(function_id, try interner.intern("x"), .user, .{ .index = 1 }, testing_span);
    const temp_id = try store.addLocal(function_id, null, .temp, .{ .index = 1 }, null);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(usize, 1), function.params.len);
    try std.testing.expectEqual(argc_id, function.params[0]);
    try std.testing.expectEqual(@as(usize, 2), function.locals.len);
    try std.testing.expectEqual(x_id, function.locals[0]);
    try std.testing.expectEqual(temp_id, function.locals[1]);

    try std.testing.expectEqual(function_id, store.getLocal(argc_id).parent);
    try std.testing.expectEqual(MirLocalKind.param, store.getLocal(argc_id).kind);
    try std.testing.expectEqualStrings("argc", interner.text(store.getLocal(argc_id).name.?));
    try std.testing.expectEqual(MirLocalKind.user, store.getLocal(x_id).kind);
    try std.testing.expectEqualStrings("x", interner.text(store.getLocal(x_id).name.?));
    try std.testing.expectEqual(MirLocalKind.temp, store.getLocal(temp_id).kind);
    try std.testing.expectEqual(@as(?hir.SymbolId, null), store.getLocal(temp_id).name);
}

test "add MIR block and associate with function" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const block_id = try store.addBlock(function_id, testing_span);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(u32, 0), block_id.index);
    try std.testing.expectEqual(@as(usize, 1), function.blocks.len);
    try std.testing.expectEqual(block_id, function.blocks[0]);
    try std.testing.expectEqual(function_id, store.getBlock(block_id).parent);
    try std.testing.expectEqual(testing_span, store.getBlock(block_id).source_span.?);
}

test "MIR debug snapshot formatting uses interned names" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var module = MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function_id = try module.store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, testing_span);
    _ = try module.store.addLocal(function_id, try interner.intern("argc"), .param, .{ .index = 1 }, testing_span);
    _ = try module.store.addLocal(function_id, try interner.intern("x"), .user, .{ .index = 1 }, testing_span);
    _ = try module.store.addLocal(function_id, null, .temp, .{ .index = 1 }, null);
    _ = try module.store.addBlock(function_id, testing_span);

    const rendered = try module.store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Params
        \\      MirLocalId(0) param argc: TypeId(1)
        \\    Locals
        \\      MirLocalId(1) user x: TypeId(1)
        \\      MirLocalId(2) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\
    , rendered);
}
