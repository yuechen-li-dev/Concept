const std = @import("std");

/// Stable identity for interned identifier text.
///
/// Values are small, copyable IDs. Callers can compare them by value, but the
/// backing text remains owned by `Interner`.
pub const SymbolId = struct {
    index: u32,

    pub fn format(self: SymbolId, writer: *std.Io.Writer) std.Io.Writer.Error!void {
        try writer.print("SymbolId({d})", .{self.index});
    }
};

pub const Interner = struct {
    allocator: std.mem.Allocator,
    lookup: std.StringHashMap(SymbolId),
    symbols: std.ArrayList([]const u8),

    pub fn init(allocator: std.mem.Allocator) Interner {
        return .{
            .allocator = allocator,
            .lookup = std.StringHashMap(SymbolId).init(allocator),
            .symbols = std.ArrayList([]const u8).empty,
        };
    }

    pub fn deinit(self: *Interner) void {
        for (self.symbols.items) |symbol_text| {
            self.allocator.free(symbol_text);
        }
        self.symbols.deinit(self.allocator);
        self.lookup.deinit();
        self.* = undefined;
    }

    pub fn intern(self: *Interner, name: []const u8) !SymbolId {
        if (self.lookup.get(name)) |existing| {
            return existing;
        }

        if (self.symbols.items.len > std.math.maxInt(u32)) {
            return error.TooManySymbols;
        }

        const owned_text = try self.allocator.dupe(u8, name);
        errdefer self.allocator.free(owned_text);

        const id = SymbolId{ .index = @intCast(self.symbols.items.len) };
        try self.lookup.put(owned_text, id);
        errdefer _ = self.lookup.remove(owned_text);

        try self.symbols.append(self.allocator, owned_text);
        return id;
    }

    /// Returns the text for a valid symbol ID.
    ///
    /// Invalid IDs assert. Semantic callers should only carry IDs previously
    /// produced by this interner instance.
    pub fn text(self: Interner, id: SymbolId) []const u8 {
        const index: usize = id.index;
        std.debug.assert(index < self.symbols.items.len);
        return self.symbols.items[index];
    }

    pub fn count(self: Interner) usize {
        return self.symbols.items.len;
    }
};

test "interning same string returns same SymbolId" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    const first = try interner.intern("name");
    const second = try interner.intern("name");

    try std.testing.expectEqual(first, second);
    try std.testing.expectEqual(@as(u32, 0), first.index);
    try std.testing.expectEqual(@as(usize, 1), interner.count());
}

test "interning different strings returns different SymbolIds" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    const first = try interner.intern("first");
    const second = try interner.intern("second");

    try std.testing.expect(first.index != second.index);
    try std.testing.expectEqual(@as(usize, 2), interner.count());
}

test "lookup text by SymbolId" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    const id = try interner.intern("lookup_target");

    try std.testing.expectEqualStrings("lookup_target", interner.text(id));
}

test "interner owns copied text from temporary caller buffer" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    var buffer = [_]u8{ 'a', 'l', 'p', 'h', 'a' };
    const id = try interner.intern(buffer[0..]);

    @memcpy(buffer[0..], "omega");

    try std.testing.expectEqualStrings("alpha", interner.text(id));
    try std.testing.expectEqual(@as(usize, 1), interner.count());
}

test "count increases only for new strings" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    _ = try interner.intern("one");
    _ = try interner.intern("two");
    _ = try interner.intern("one");
    _ = try interner.intern("two");
    _ = try interner.intern("three");

    try std.testing.expectEqual(@as(usize, 3), interner.count());
}

test "empty string can be interned and looked up" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    const empty = try interner.intern("");
    const repeated = try interner.intern("");

    try std.testing.expectEqual(empty, repeated);
    try std.testing.expectEqualStrings("", interner.text(empty));
    try std.testing.expectEqual(@as(usize, 1), interner.count());
}

test "many inserts keep stable lookup results" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();

    const total = 256;
    var ids: [total]SymbolId = undefined;
    var expected: [total][32]u8 = undefined;
    var expected_lens: [total]usize = undefined;

    for (&ids, 0..) |*slot, index| {
        const text_buffer = try std.fmt.bufPrint(&expected[index], "symbol_{d}", .{index});
        expected_lens[index] = text_buffer.len;
        slot.* = try interner.intern(text_buffer);
    }

    try std.testing.expectEqual(@as(usize, total), interner.count());

    for (ids, 0..) |id, index| {
        try std.testing.expectEqual(@as(u32, @intCast(index)), id.index);
        try std.testing.expectEqualStrings(expected[index][0..expected_lens[index]], interner.text(id));
    }

    for (ids, 0..) |id, index| {
        const text_buffer = try std.fmt.bufPrint(&expected[index], "symbol_{d}", .{index});
        try std.testing.expectEqual(id, try interner.intern(text_buffer));
    }
}

test "SymbolId debug formatting" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f}", .{SymbolId{ .index = 42 }});
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("SymbolId(42)", rendered);
}
