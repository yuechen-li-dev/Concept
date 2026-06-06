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
    generic_args: []TypeName = &.{},
    is_mut: bool = false,
    is_reference: bool = false,
    is_pointer: bool = false,
    span: SourceSpan,

    pub fn deinit(self: TypeName, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
        for (self.generic_args) |generic_arg| {
            generic_arg.deinit(allocator);
        }
        allocator.free(self.generic_args);
    }

    pub fn write(self: TypeName, writer: anytype) !void {
        if (self.is_mut) try writer.writeAll("mut ");
        try self.name.write(writer);
        if (self.generic_args.len != 0) {
            try writer.writeByte('<');
            for (self.generic_args, 0..) |generic_arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try generic_arg.write(writer);
            }
            try writer.writeByte('>');
        }
        if (self.is_reference) try writer.writeByte('&');
        if (self.is_pointer) try writer.writeByte('*');
    }
};

pub const GenericParam = NameSegment;

pub const SignatureName = struct {
    base: NameSegment,
    operator_suffix: ?NameSegment = null,
    span: SourceSpan,

    pub fn write(self: SignatureName, writer: anytype) !void {
        try writer.writeAll(self.base.text);
        if (self.operator_suffix) |suffix| try writer.writeAll(suffix.text);
    }
};

pub const ParamDecl = struct {
    type_name: TypeName,
    name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: ParamDecl, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
    }
};

pub const SignatureDecl = struct {
    return_type: TypeName,
    name: SignatureName,
    params: []ParamDecl,
    span: SourceSpan,

    pub fn deinit(self: SignatureDecl, allocator: std.mem.Allocator) void {
        self.return_type.deinit(allocator);
        for (self.params) |param| {
            param.deinit(allocator);
        }
        allocator.free(self.params);
    }

    pub fn writeDebug(self: SignatureDecl, writer: anytype) !void {
        try writer.writeAll("    Signature ");
        try self.return_type.write(writer);
        try writer.writeByte(' ');
        try self.name.write(writer);
        try writer.writeByte('(');
        for (self.params, 0..) |param, index| {
            if (index != 0) try writer.writeAll(", ");
            try param.type_name.write(writer);
            try writer.writeByte(' ');
            try writer.writeAll(param.name.text);
        }
        try writer.writeAll(")\n");
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

pub const ConceptDecl = struct {
    name: NameSegment,
    generic_params: []GenericParam,
    signatures: []SignatureDecl,
    span: SourceSpan,

    pub fn deinit(self: ConceptDecl, allocator: std.mem.Allocator) void {
        allocator.free(self.generic_params);
        for (self.signatures) |signature| {
            signature.deinit(allocator);
        }
        allocator.free(self.signatures);
    }
};

pub const InterfaceDecl = struct {
    name: NameSegment,
    signatures: []SignatureDecl,
    span: SourceSpan,

    pub fn deinit(self: InterfaceDecl, allocator: std.mem.Allocator) void {
        for (self.signatures) |signature| {
            signature.deinit(allocator);
        }
        allocator.free(self.signatures);
    }
};

pub const ImplDecl = struct {
    target: TypeName,
    signatures: []SignatureDecl,
    span: SourceSpan,

    pub fn deinit(self: ImplDecl, allocator: std.mem.Allocator) void {
        self.target.deinit(allocator);
        for (self.signatures) |signature| {
            signature.deinit(allocator);
        }
        allocator.free(self.signatures);
    }
};

pub const FunctionBody = struct {
    span: SourceSpan,
};

pub const FunctionDecl = struct {
    is_export: bool,
    signature: SignatureDecl,
    body: ?FunctionBody,
    span: SourceSpan,

    pub fn deinit(self: FunctionDecl, allocator: std.mem.Allocator) void {
        self.signature.deinit(allocator);
    }
};

pub const Item = union(enum) {
    function_decl: FunctionDecl,
    struct_decl: StructDecl,
    enum_decl: EnumDecl,
    concept_decl: ConceptDecl,
    interface_decl: InterfaceDecl,
    impl_decl: ImplDecl,

    pub fn deinit(self: Item, allocator: std.mem.Allocator) void {
        switch (self) {
            .function_decl => |function_decl| function_decl.deinit(allocator),
            .struct_decl => |struct_decl| struct_decl.deinit(allocator),
            .enum_decl => |enum_decl| enum_decl.deinit(allocator),
            .concept_decl => |concept_decl| concept_decl.deinit(allocator),
            .interface_decl => |interface_decl| interface_decl.deinit(allocator),
            .impl_decl => |impl_decl| impl_decl.deinit(allocator),
        }
    }

    pub fn writeDebug(self: Item, writer: anytype) !void {
        switch (self) {
            .function_decl => |function_decl| {
                if (function_decl.is_export) {
                    try writer.writeAll("  Export Function ");
                } else {
                    try writer.writeAll("  Function ");
                }
                try function_decl.signature.return_type.write(writer);
                try writer.writeByte(' ');
                try function_decl.signature.name.write(writer);
                try writer.writeByte('(');
                for (function_decl.signature.params, 0..) |param, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try param.type_name.write(writer);
                    try writer.writeByte(' ');
                    try writer.writeAll(param.name.text);
                }
                try writer.writeAll(")\n");
                if (function_decl.body != null) {
                    try writer.writeAll("    Body\n");
                }
            },
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
            .concept_decl => |concept_decl| {
                try writer.writeAll("  Concept ");
                try writer.writeAll(concept_decl.name.text);
                if (concept_decl.generic_params.len != 0) {
                    try writer.writeByte('<');
                    for (concept_decl.generic_params, 0..) |generic_param, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try writer.writeAll(generic_param.text);
                    }
                    try writer.writeByte('>');
                }
                try writer.writeByte('\n');
                for (concept_decl.signatures) |signature| {
                    try signature.writeDebug(writer);
                }
            },
            .interface_decl => |interface_decl| {
                try writer.writeAll("  Interface ");
                try writer.writeAll(interface_decl.name.text);
                try writer.writeByte('\n');
                for (interface_decl.signatures) |signature| {
                    try signature.writeDebug(writer);
                }
            },
            .impl_decl => |impl_decl| {
                try writer.writeAll("  Impl ");
                try impl_decl.target.write(writer);
                try writer.writeByte('\n');
                for (impl_decl.signatures) |signature| {
                    try signature.writeDebug(writer);
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
