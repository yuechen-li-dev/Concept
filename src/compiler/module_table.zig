const std = @import("std");

const ast = @import("ast.zig");
const diagnostics = @import("diagnostics.zig");
const source = @import("source.zig");

pub const ModuleId = struct { index: usize };

pub const ParsedSource = struct {
    path: []const u8,
    unit: *const ast.CompilationUnit,
};

pub const ModuleUnit = struct {
    id: ModuleId,
    name: []const u8,
    source_index: usize,
    source_path: []const u8,
    module_decl_span: source.SourceSpan,
};

pub const ModuleTable = struct {
    modules: []ModuleUnit,
    name_to_module: std.StringHashMapUnmanaged(ModuleId) = .empty,

    pub fn deinit(self: *ModuleTable, allocator: std.mem.Allocator) void {
        for (self.modules) |module| allocator.free(module.name);
        allocator.free(self.modules);
        self.name_to_module.deinit(allocator);
        self.* = .{ .modules = &.{} };
    }

    pub fn find(self: ModuleTable, name: []const u8) ?ModuleUnit {
        const id = self.name_to_module.get(name) orelse return null;
        return self.modules[id.index];
    }
};

pub fn buildFromParsedSources(
    allocator: std.mem.Allocator,
    parsed_sources: []const ParsedSource,
    diagnostic_bag: *diagnostics.DiagnosticBag,
) !ModuleTable {
    var modules = std.ArrayList(ModuleUnit).empty;
    errdefer {
        for (modules.items) |module| allocator.free(module.name);
        modules.deinit(allocator);
    }

    var name_to_module: std.StringHashMapUnmanaged(ModuleId) = .empty;
    errdefer name_to_module.deinit(allocator);

    for (parsed_sources, 0..) |parsed_source, source_index| {
        const module_decl = parsed_source.unit.module orelse {
            try diagnostic_bag.append(try diagnostics.moduleDeclarationRequired(allocator, parsed_source.path, parsed_source.unit.span));
            continue;
        };

        const name = try qualifiedNameString(allocator, module_decl.name);
        errdefer allocator.free(name);

        if (name_to_module.get(name)) |previous_id| {
            const previous = modules.items[previous_id.index];
            try diagnostic_bag.append(try diagnostics.duplicateModule(allocator, name, module_decl.span, previous.source_path));
            allocator.free(name);
            continue;
        }

        const id: ModuleId = .{ .index = modules.items.len };
        try modules.append(allocator, .{
            .id = id,
            .name = name,
            .source_index = source_index,
            .source_path = parsed_source.path,
            .module_decl_span = module_decl.span,
        });
        try name_to_module.put(allocator, name, id);
    }

    return .{
        .modules = try modules.toOwnedSlice(allocator),
        .name_to_module = name_to_module,
    };
}

pub fn qualifiedNameString(allocator: std.mem.Allocator, name: ast.QualifiedName) ![]const u8 {
    var output = std.Io.Writer.Allocating.init(allocator);
    errdefer output.deinit();
    try name.write(&output.writer);
    return output.toOwnedSlice();
}

const testing = std.testing;

test "module table preserves source order paths ids names and spans" {
    const parser = @import("parser.zig");

    var diagnostics_bag = diagnostics.DiagnosticBag.init(testing.allocator);
    defer diagnostics_bag.deinit();

    var source_a = try source.SourceFile.init(testing.allocator, "A.concept", "module A;\n");
    defer source_a.deinit(testing.allocator);
    var unit_a = try parser.parseSource(testing.allocator, source_a, &diagnostics_bag);
    defer unit_a.deinit(testing.allocator);

    var source_b = try source.SourceFile.init(testing.allocator, "B.concept", "module Compiler.Lexer;\n");
    defer source_b.deinit(testing.allocator);
    var unit_b = try parser.parseSource(testing.allocator, source_b, &diagnostics_bag);
    defer unit_b.deinit(testing.allocator);

    var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "A.concept", .unit = &unit_a }, .{ .path = "B.concept", .unit = &unit_b } }, &diagnostics_bag);
    defer table.deinit(testing.allocator);

    try testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    try testing.expectEqual(@as(usize, 2), table.modules.len);
    try testing.expectEqual(@as(usize, 0), table.modules[0].id.index);
    try testing.expectEqual(@as(usize, 1), table.modules[1].id.index);
    try testing.expectEqualStrings("A", table.modules[0].name);
    try testing.expectEqualStrings("Compiler.Lexer", table.modules[1].name);
    try testing.expectEqualStrings("B.concept", table.modules[1].source_path);
    try testing.expectEqual(@as(usize, 0), table.modules[0].module_decl_span.start);
    try testing.expect(table.find("Compiler.Lexer") != null);
}
