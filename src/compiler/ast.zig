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

pub const TypeName = struct {
    name: QualifiedName,
    span: SourceSpan,

    pub fn deinit(self: TypeName, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
    }

    pub fn write(self: TypeName, writer: anytype) !void {
        try self.name.write(writer);
    }
};

pub const FieldDecl = struct {
    type_name: TypeName,
    name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: FieldDecl, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
    }
};

pub const StructDecl = struct {
    is_export: bool,
    name: NameSegment,
    fields: []FieldDecl,
    span: SourceSpan,

    pub fn deinit(self: StructDecl, allocator: std.mem.Allocator) void {
        for (self.fields) |field| {
            field.deinit(allocator);
        }
        allocator.free(self.fields);
    }
};

pub const EnumPayloadField = struct {
    type_name: TypeName,
    name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: EnumPayloadField, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
    }
};

pub const EnumVariant = struct {
    name: NameSegment,
    payload_fields: []EnumPayloadField,
    span: SourceSpan,

    pub fn deinit(self: EnumVariant, allocator: std.mem.Allocator) void {
        for (self.payload_fields) |field| {
            field.deinit(allocator);
        }
        allocator.free(self.payload_fields);
    }
};

pub const EnumDecl = struct {
    is_export: bool,
    name: NameSegment,
    variants: []EnumVariant,
    span: SourceSpan,

    pub fn deinit(self: EnumDecl, allocator: std.mem.Allocator) void {
        for (self.variants) |variant| {
            variant.deinit(allocator);
        }
        allocator.free(self.variants);
    }
};

pub const Item = union(enum) {
    struct_decl: StructDecl,
    enum_decl: EnumDecl,

    pub fn deinit(self: Item, allocator: std.mem.Allocator) void {
        switch (self) {
            .struct_decl => |struct_decl| struct_decl.deinit(allocator),
            .enum_decl => |enum_decl| enum_decl.deinit(allocator),
        }
    }

    pub fn writeDebug(self: Item, writer: anytype) !void {
        switch (self) {
            .struct_decl => |struct_decl| {
                if (struct_decl.is_export) {
                    try writer.writeAll("  Export Struct ");
                } else {
                    try writer.writeAll("  Struct ");
                }
                try writer.writeAll(struct_decl.name.text);
                try writer.writeByte('\n');

                for (struct_decl.fields) |field| {
                    try writer.writeAll("    Field ");
                    try field.type_name.write(writer);
                    try writer.writeByte(' ');
                    try writer.writeAll(field.name.text);
                    try writer.writeByte('\n');
                }
            },
            .enum_decl => |enum_decl| {
                if (enum_decl.is_export) {
                    try writer.writeAll("  Export Enum ");
                } else {
                    try writer.writeAll("  Enum ");
                }
                try writer.writeAll(enum_decl.name.text);
                try writer.writeByte('\n');

                for (enum_decl.variants) |variant| {
                    try writer.writeAll("    Variant ");
                    try writer.writeAll(variant.name.text);
                    try writer.writeByte('\n');

                    for (variant.payload_fields) |field| {
                        try writer.writeAll("      Payload ");
                        try field.type_name.write(writer);
                        try writer.writeByte(' ');
                        try writer.writeAll(field.name.text);
                        try writer.writeByte('\n');
                    }
                }
            },
        }
    }
};

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
        for (self.items) |item| {
            item.deinit(allocator);
        }
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
        for (self.items) |item| {
            try item.writeDebug(writer);
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
