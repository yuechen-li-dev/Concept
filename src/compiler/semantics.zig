const std = @import("std");

const ast = @import("ast.zig");
const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const interner = @import("interner.zig");
const source = @import("source.zig");
const types = @import("types.zig");

pub const DiagnosticBag = diagnostics.DiagnosticBag;
pub const DiagnosticCode = diagnostics.DiagnosticCode;
pub const SemanticError = error{InvalidSemanticModule};

pub const SemanticModule = struct {
    interner: interner.Interner,
    hir: hir.HirStore,
    types: types.TypeStore,

    pub fn init(allocator: std.mem.Allocator) !SemanticModule {
        return .{
            .interner = interner.Interner.init(allocator),
            .hir = hir.HirStore.init(allocator),
            .types = try types.TypeStore.init(allocator),
        };
    }

    pub fn deinit(self: *SemanticModule) void {
        self.types.deinit();
        self.hir.deinit();
        self.interner.deinit();
        self.* = undefined;
    }
};

pub fn collectTopLevelDeclarations(
    allocator: std.mem.Allocator,
    unit: ast.CompilationUnit,
    diagnostic_bag: *DiagnosticBag,
) !SemanticModule {
    var module = try SemanticModule.init(allocator);
    errdefer module.deinit();

    var collector = Collector.init(allocator, &module, diagnostic_bag);
    defer collector.deinit();

    try collector.collect(unit);
    if (diagnostic_bag.count() != 0) return error.InvalidSemanticModule;
    return module;
}

const Collector = struct {
    allocator: std.mem.Allocator,
    module: *SemanticModule,
    diagnostics: *DiagnosticBag,
    top_level_decls: std.AutoHashMap(interner.SymbolId, TopLevelDecl),

    const TopLevelDecl = union(enum) {
        function: hir.FunctionId,
        struct_: struct {
            id: hir.StructId,
            type_id: types.TypeId,
        },
        enum_: struct {
            id: hir.EnumId,
            type_id: types.TypeId,
        },
    };

    fn init(allocator: std.mem.Allocator, module: *SemanticModule, diagnostic_bag: *DiagnosticBag) Collector {
        return .{
            .allocator = allocator,
            .module = module,
            .diagnostics = diagnostic_bag,
            .top_level_decls = std.AutoHashMap(interner.SymbolId, TopLevelDecl).init(allocator),
        };
    }

    fn deinit(self: *Collector) void {
        self.top_level_decls.deinit();
    }

    fn collect(self: *Collector, unit: ast.CompilationUnit) !void {
        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.declareFunction(function_decl),
                .struct_decl => |struct_decl| try self.declareStruct(struct_decl),
                .enum_decl => |enum_decl| try self.declareEnum(enum_decl),
                .concept_decl, .interface_decl, .impl_decl => {},
            }
        }

        if (self.diagnostics.count() != 0) return;

        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.resolveFunction(function_decl),
                .struct_decl => |struct_decl| try self.resolveStruct(struct_decl),
                .enum_decl => |enum_decl| try self.resolveEnum(enum_decl),
                .concept_decl, .interface_decl, .impl_decl => {},
            }
        }
    }

    fn declareFunction(self: *Collector, function_decl: ast.FunctionDecl) !void {
        const name = try self.internFreshTopLevelName(
            function_decl.signature.name.base.text,
            function_decl.signature.name.base.span,
        ) orelse return;
        const function_id = try self.module.hir.addFunction(name, self.module.types.voidType());
        try self.top_level_decls.put(name, .{ .function = function_id });
    }

    fn declareStruct(self: *Collector, struct_decl: ast.StructDecl) !void {
        const name = try self.internFreshTopLevelName(struct_decl.name.text, struct_decl.name.span) orelse return;
        const struct_id = try self.module.hir.addStruct(name);
        const type_id = try self.module.types.addStructType(struct_id);
        try self.top_level_decls.put(name, .{ .struct_ = .{ .id = struct_id, .type_id = type_id } });
    }

    fn declareEnum(self: *Collector, enum_decl: ast.EnumDecl) !void {
        const name = try self.internFreshTopLevelName(enum_decl.name.text, enum_decl.name.span) orelse return;
        const enum_id = try self.module.hir.addEnum(name);
        const type_id = try self.module.types.addEnumType(enum_id);
        try self.top_level_decls.put(name, .{ .enum_ = .{ .id = enum_id, .type_id = type_id } });
    }

    fn resolveFunction(self: *Collector, function_decl: ast.FunctionDecl) !void {
        const function_symbol = try self.module.interner.intern(function_decl.signature.name.base.text);
        const function_id = switch (self.top_level_decls.get(function_symbol).?) {
            .function => |id| id,
            else => unreachable,
        };

        if (try self.resolveTypeName(function_decl.signature.return_type)) |return_type| {
            self.module.hir.setFunctionReturnType(function_id, return_type);
        }

        var param_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer param_names.deinit();

        for (function_decl.signature.params) |param| {
            const param_symbol = try self.module.interner.intern(param.name.text);
            if (param_names.contains(param_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateParameterName(param.name.span));
                continue;
            }
            try param_names.put(param_symbol, param.name.span);

            if (try self.resolveTypeName(param.type_name)) |type_id| {
                _ = try self.module.hir.addParam(function_id, param_symbol, type_id);
            }
        }
    }

    fn resolveStruct(self: *Collector, struct_decl: ast.StructDecl) !void {
        const struct_symbol = try self.module.interner.intern(struct_decl.name.text);
        const struct_id = switch (self.top_level_decls.get(struct_symbol).?) {
            .struct_ => |entry| entry.id,
            else => unreachable,
        };

        var field_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer field_names.deinit();

        for (struct_decl.fields) |field| {
            const field_symbol = try self.module.interner.intern(field.name.text);
            if (field_names.contains(field_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateStructField(field.name.span));
                continue;
            }
            try field_names.put(field_symbol, field.name.span);

            if (try self.resolveTypeName(field.type_name)) |type_id| {
                _ = try self.module.hir.addField(struct_id, field_symbol, type_id);
            }
        }
    }

    fn resolveEnum(self: *Collector, enum_decl: ast.EnumDecl) !void {
        const enum_symbol = try self.module.interner.intern(enum_decl.name.text);
        const enum_id = switch (self.top_level_decls.get(enum_symbol).?) {
            .enum_ => |entry| entry.id,
            else => unreachable,
        };

        var variant_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer variant_names.deinit();

        for (enum_decl.variants) |variant| {
            const variant_symbol = try self.module.interner.intern(variant.name.text);
            if (variant_names.contains(variant_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateEnumVariant(variant.name.span));
                continue;
            }
            try variant_names.put(variant_symbol, variant.name.span);

            const variant_id = try self.module.hir.addVariant(enum_id, variant_symbol);
            for (variant.payload_fields) |payload_field| {
                const payload_symbol = try self.module.interner.intern(payload_field.name.text);
                if (try self.resolveTypeName(payload_field.type_name)) |type_id| {
                    _ = try self.module.hir.addEnumPayloadField(variant_id, payload_symbol, type_id);
                }
            }
        }
    }

    fn resolveTypeName(self: *Collector, type_name: ast.TypeName) !?types.TypeId {
        if (type_name.is_mut or type_name.is_reference or type_name.is_pointer or type_name.generic_args.len != 0 or type_name.name.parts.len != 1) {
            try self.diagnostics.append(diagnostics.unsupportedTypeSyntax(type_name.span));
            return null;
        }

        const part = type_name.name.parts[0];
        if (std.mem.eql(u8, part.text, "void")) return self.module.types.voidType();
        if (std.mem.eql(u8, part.text, "int")) return self.module.types.intType();
        if (std.mem.eql(u8, part.text, "bool")) return self.module.types.boolType();

        const symbol = try self.module.interner.intern(part.text);
        const decl = self.top_level_decls.get(symbol) orelse {
            try self.diagnostics.append(diagnostics.unknownTypeName(part.span));
            return null;
        };

        return switch (decl) {
            .struct_ => |entry| entry.type_id,
            .enum_ => |entry| entry.type_id,
            .function => blk: {
                try self.diagnostics.append(diagnostics.unknownTypeName(part.span));
                break :blk null;
            },
        };
    }

    fn internFreshTopLevelName(self: *Collector, text: []const u8, span: source.SourceSpan) !?interner.SymbolId {
        const symbol = try self.module.interner.intern(text);
        if (self.top_level_decls.contains(symbol)) {
            try self.diagnostics.append(diagnostics.duplicateTopLevelName(span));
            return null;
        }
        return symbol;
    }
};

fn collectItems(items: []const ast.Item, diagnostic_bag: *DiagnosticBag) !SemanticModule {
    return collectTopLevelDeclarations(std.testing.allocator, unitFromItems(items), diagnostic_bag);
}

fn unitFromItems(items: []const ast.Item) ast.CompilationUnit {
    return .{
        .module = null,
        .imports = &.{},
        .items = @constCast(items),
        .span = .{ .start = 0, .length = 0 },
    };
}

fn functionItem(name: []const u8, start: usize) ast.Item {
    return .{ .function_decl = .{
        .is_export = false,
        .signature = .{
            .return_type = typeName("int", start),
            .name = .{
                .base = nameSegment(name, start + 4),
                .span = .{ .start = start + 4, .length = name.len },
            },
            .params = &.{},
            .span = .{ .start = start, .length = name.len + 6 },
        },
        .body = null,
        .span = .{ .start = start, .length = name.len + 6 },
    } };
}

fn structItem(name: []const u8, start: usize) ast.Item {
    return .{ .struct_decl = .{
        .is_export = false,
        .name = nameSegment(name, start + 7),
        .fields = &.{},
        .span = .{ .start = start, .length = name.len + 10 },
    } };
}

fn enumItem(name: []const u8, start: usize) ast.Item {
    return .{ .enum_decl = .{
        .is_export = false,
        .name = nameSegment(name, start + 5),
        .variants = &.{},
        .span = .{ .start = start, .length = name.len + 8 },
    } };
}

fn conceptItem(name: []const u8, start: usize) ast.Item {
    return .{ .concept_decl = .{
        .name = nameSegment(name, start + 8),
        .generic_params = &.{},
        .signatures = &.{},
        .span = .{ .start = start, .length = name.len + 11 },
    } };
}

fn interfaceItem(name: []const u8, start: usize) ast.Item {
    return .{ .interface_decl = .{
        .name = nameSegment(name, start + 10),
        .signatures = &.{},
        .span = .{ .start = start, .length = name.len + 13 },
    } };
}

fn implItem(start: usize) ast.Item {
    return .{ .impl_decl = .{
        .target = typeName("Drop", start + 5),
        .signatures = &.{},
        .span = .{ .start = start, .length = 12 },
    } };
}

const test_int_type_parts = [_]ast.NameSegment{.{ .text = "int", .span = .{ .start = 0, .length = 3 } }};
const test_drop_type_parts = [_]ast.NameSegment{.{ .text = "Drop", .span = .{ .start = 0, .length = 4 } }};

fn typeName(name: []const u8, start: usize) ast.TypeName {
    const parts: []ast.NameSegment = if (std.mem.eql(u8, name, "Drop"))
        @constCast(test_drop_type_parts[0..])
    else
        @constCast(test_int_type_parts[0..]);
    return .{
        .name = .{
            .parts = parts,
            .span = .{ .start = start, .length = name.len },
        },
        .span = .{ .start = start, .length = name.len },
    };
}

fn nameSegment(text: []const u8, start: usize) ast.NameSegment {
    return .{ .text = text, .span = .{ .start = start, .length = text.len } };
}

test "semantic collection collects one function" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{functionItem("main", 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.functions.items.len);
    try std.testing.expectEqualStrings("main", module.interner.text(module.hir.getFunction(.{ .index = 0 }).name));
}

test "semantic collection collects one struct" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{structItem("Vec2", 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqualStrings("Vec2", module.interner.text(module.hir.getStruct(.{ .index = 0 }).name));
}

test "semantic collection collects one enum" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{enumItem("Token", 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.enums.items.len);
    try std.testing.expectEqualStrings("Token", module.interner.text(module.hir.getEnum(.{ .index = 0 }).name));
}

test "semantic collection collects function struct and enum together" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ functionItem("make", 0), structItem("Vec2", 20), enumItem("Token", 40) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 3), module.hir.items.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.functions.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.enums.items.len);
}

test "semantic collection adds struct and enum nominal types" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ structItem("Vec2", 0), enumItem("Token", 20) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 5), module.types.count());
    try std.testing.expectEqual(types.TypeKind{ .struct_type = .{ .index = 0 } }, module.types.kind(.{ .index = 3 }));
    try std.testing.expectEqual(types.TypeKind{ .enum_type = .{ .index = 0 } }, module.types.kind(.{ .index = 4 }));
}

test "semantic collection rejects duplicate function" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(
        error.InvalidSemanticModule,
        collectItems(&.{ functionItem("f", 0), functionItem("f", 20) }, &diagnostics_bag),
    );
    try std.testing.expectEqual(@as(usize, 1), diagnostics_bag.count());
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, diagnostics_bag.diagnostics.items[0].code);
}

test "semantic collection rejects duplicate function and struct" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(
        error.InvalidSemanticModule,
        collectItems(&.{ functionItem("Foo", 0), structItem("Foo", 30) }, &diagnostics_bag),
    );
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, diagnostics_bag.diagnostics.items[0].code);
}

test "semantic collection rejects duplicate struct and enum" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(
        error.InvalidSemanticModule,
        collectItems(&.{ structItem("Token", 0), enumItem("Token", 30) }, &diagnostics_bag),
    );
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, diagnostics_bag.diagnostics.items[0].code);
}

test "semantic collection ignores concept interface and impl items" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ conceptItem("Hashable", 0), interfaceItem("Renderer", 40), implItem(80), structItem("Texture", 100) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.items.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
}

test "semantic collection HIR debug snapshot is stable" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ functionItem("make", 0), structItem("Vec2", 20), enumItem("Token", 40) }, &diagnostics_bag);
    defer module.deinit();

    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Function make -> TypeId(1)
        \\  Struct Vec2
        \\  Enum Token
        \\
    , snapshot);
}
