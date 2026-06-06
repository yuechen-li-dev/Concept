const std = @import("std");
const source_model = @import("source.zig");

pub const SourceSpan = source_model.SourceSpan;

pub const NameSegment = struct {
    text: []const u8,
    span: SourceSpan,
};

pub const QualifiedName = struct {
    parts: []NameSegment,
    span: SourceSpan,

    pub fn deinit(self: QualifiedName, allocator: std.mem.Allocator) void {
        allocator.free(self.parts);
    }

    pub fn write(self: QualifiedName, writer: anytype) !void {
        for (self.parts, 0..) |part, index| {
            if (index != 0) try writer.writeByte('.');
            try writer.writeAll(part.text);
        }
    }
};

pub const ModuleDecl = struct {
    name: QualifiedName,
    span: SourceSpan,

    pub fn deinit(self: ModuleDecl, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
    }
};

pub const ImportDecl = struct {
    name: QualifiedName,
    span: SourceSpan,

    pub fn deinit(self: ImportDecl, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
    }
};

/// Placeholder for future top-level declarations.
pub const Item = union(enum) {};

pub const CompilationUnit = struct {
    span: SourceSpan,
    module: ?ModuleDecl,
    imports: []ImportDecl,
    items: []Item,

    pub fn deinit(self: CompilationUnit, allocator: std.mem.Allocator) void {
        if (self.module) |module| module.deinit(allocator);
        for (self.imports) |import_decl| {
            import_decl.deinit(allocator);
        }
        allocator.free(self.imports);
        allocator.free(self.items);
    }

    pub fn writeDebug(self: CompilationUnit, writer: anytype) !void {
        try writer.writeAll("CompilationUnit\n");
        if (self.module) |module| {
            try writer.writeAll("  Module ");
            try module.name.write(writer);
            try writer.writeByte('\n');
        }
        for (self.imports) |import_decl| {
            try writer.writeAll("  Import ");
            try import_decl.name.write(writer);
            try writer.writeByte('\n');
        }
    }

    pub fn debugString(self: CompilationUnit, allocator: std.mem.Allocator) ![]u8 {
        var output = std.ArrayList(u8).init(allocator);
        errdefer output.deinit();
        try self.writeDebug(output.writer());
        return output.toOwnedSlice();
    }
};

pub fn spanFromBounds(start: usize, end: usize) SourceSpan {
    return .{ .start = start, .length = end - start };
}

pub fn mergeSpans(first: SourceSpan, second: SourceSpan) SourceSpan {
    const first_end = first.end() orelse first.start;
    const second_end = second.end() orelse second.start;
    return spanFromBounds(@min(first.start, second.start), @max(first_end, second_end));
}

test "qualified name debug writing joins segments with dots" {
    const parts = try std.testing.allocator.dupe(NameSegment, &.{
        .{ .text = "Example", .span = .{ .start = 0, .length = 7 } },
        .{ .text = "Compiler", .span = .{ .start = 8, .length = 8 } },
    });
    defer std.testing.allocator.free(parts);

    const name = QualifiedName{ .parts = parts, .span = .{ .start = 0, .length = 16 } };
    var output = std.ArrayList(u8).init(std.testing.allocator);
    defer output.deinit();

    try name.write(output.writer());
    try std.testing.expectEqualStrings("Example.Compiler", output.items);
}
