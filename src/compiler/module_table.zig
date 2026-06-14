const std = @import("std");

const ast = @import("ast.zig");
const diagnostics = @import("diagnostics.zig");
const source = @import("source.zig");

pub const ModuleId = struct { index: usize };

pub const ParsedSource = struct {
    path: []const u8,
    unit: *const ast.CompilationUnit,
};

pub const RawImport = struct {
    name: []const u8,
    import_span: source.SourceSpan,
    path_span: source.SourceSpan,
};

pub const ModuleUnit = struct {
    id: ModuleId,
    name: []const u8,
    source_index: usize,
    source_path: []const u8,
    module_decl_span: source.SourceSpan,
    imports: []RawImport,
    resolved_imports: []ModuleId,
};

pub const ModuleTable = struct {
    modules: []ModuleUnit,
    name_to_module: std.StringHashMapUnmanaged(ModuleId) = .empty,

    pub fn deinit(self: *ModuleTable, allocator: std.mem.Allocator) void {
        for (self.modules) |module| {
            allocator.free(module.name);
            for (module.imports) |import_decl| allocator.free(import_decl.name);
            allocator.free(module.imports);
            allocator.free(module.resolved_imports);
        }
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
        for (modules.items) |module| {
            allocator.free(module.name);
            for (module.imports) |import_decl| allocator.free(import_decl.name);
            allocator.free(module.imports);
            allocator.free(module.resolved_imports);
        }
        modules.deinit(allocator);
    }

    var name_to_module: std.StringHashMapUnmanaged(ModuleId) = .empty;
    errdefer name_to_module.deinit(allocator);
    var structural_error = false;

    for (parsed_sources, 0..) |parsed_source, source_index| {
        const module_decl = parsed_source.unit.module orelse {
            try diagnostic_bag.append(try diagnostics.moduleDeclarationRequired(allocator, parsed_source.path, parsed_source.unit.span));
            structural_error = true;
            continue;
        };

        const name = try qualifiedNameString(allocator, module_decl.name);
        errdefer allocator.free(name);

        if (name_to_module.get(name)) |previous_id| {
            const previous = modules.items[previous_id.index];
            try diagnostic_bag.append(try diagnostics.duplicateModule(allocator, name, module_decl.span, previous.source_path));
            structural_error = true;
            allocator.free(name);
            continue;
        }

        var imports = std.ArrayList(RawImport).empty;
        errdefer {
            for (imports.items) |import_decl| allocator.free(import_decl.name);
            imports.deinit(allocator);
        }
        for (parsed_source.unit.imports) |import_decl| {
            try imports.append(allocator, .{
                .name = try qualifiedNameString(allocator, import_decl.name),
                .import_span = import_decl.span,
                .path_span = import_decl.name.span,
            });
        }

        const id: ModuleId = .{ .index = modules.items.len };
        try modules.append(allocator, .{
            .id = id,
            .name = name,
            .source_index = source_index,
            .source_path = parsed_source.path,
            .module_decl_span = module_decl.span,
            .imports = try imports.toOwnedSlice(allocator),
            .resolved_imports = &.{},
        });
        try name_to_module.put(allocator, name, id);
    }

    var table = ModuleTable{
        .modules = try modules.toOwnedSlice(allocator),
        .name_to_module = name_to_module,
    };

    if (!structural_error) {
        try resolveImports(allocator, &table, diagnostic_bag);
    }

    return table;
}

fn resolveImports(allocator: std.mem.Allocator, table: *ModuleTable, diagnostic_bag: *diagnostics.DiagnosticBag) !void {
    var had_resolution_error = false;

    for (table.modules) |*module| {
        var seen: std.StringHashMapUnmanaged(void) = .empty;
        defer seen.deinit(allocator);

        var resolved = std.ArrayList(ModuleId).empty;
        errdefer resolved.deinit(allocator);

        for (module.imports) |import_decl| {
            if (seen.contains(import_decl.name)) {
                try diagnostic_bag.append(try diagnostics.duplicateImport(allocator, import_decl.name, import_decl.path_span));
                had_resolution_error = true;
                continue;
            }
            try seen.put(allocator, import_decl.name, {});

            const imported_id = table.name_to_module.get(import_decl.name) orelse {
                try diagnostic_bag.append(try diagnostics.unknownImport(allocator, import_decl.name, module.source_path, import_decl.path_span));
                had_resolution_error = true;
                continue;
            };
            try resolved.append(allocator, imported_id);
        }

        module.resolved_imports = try resolved.toOwnedSlice(allocator);
    }

    if (!had_resolution_error) {
        try detectCycles(allocator, table, diagnostic_bag);
    }
}

const VisitState = enum { unvisited, visiting, visited };

fn detectCycles(allocator: std.mem.Allocator, table: *ModuleTable, diagnostic_bag: *diagnostics.DiagnosticBag) !void {
    const states = try allocator.alloc(VisitState, table.modules.len);
    defer allocator.free(states);
    @memset(states, .unvisited);

    var stack = std.ArrayList(ModuleId).empty;
    defer stack.deinit(allocator);

    for (table.modules) |module| {
        if (states[module.id.index] == .unvisited) {
            try visitModule(allocator, table, diagnostic_bag, module.id, states, &stack);
        }
    }
}

fn visitModule(
    allocator: std.mem.Allocator,
    table: *ModuleTable,
    diagnostic_bag: *diagnostics.DiagnosticBag,
    id: ModuleId,
    states: []VisitState,
    stack: *std.ArrayList(ModuleId),
) !void {
    states[id.index] = .visiting;
    try stack.append(allocator, id);

    const module = table.modules[id.index];
    for (module.resolved_imports, 0..) |next_id, import_index| {
        switch (states[next_id.index]) {
            .unvisited => try visitModule(allocator, table, diagnostic_bag, next_id, states, stack),
            .visiting => {
                const cycle = try cycleString(allocator, table, stack.items, next_id);
                defer allocator.free(cycle);
                const span = module.imports[import_index].path_span;
                try diagnostic_bag.append(try diagnostics.importCycle(allocator, cycle, span));
            },
            .visited => {},
        }
    }

    _ = stack.pop();
    states[id.index] = .visited;
}

fn cycleString(allocator: std.mem.Allocator, table: *const ModuleTable, stack: []const ModuleId, repeated: ModuleId) ![]const u8 {
    var start: usize = 0;
    for (stack, 0..) |id, index| {
        if (id.index == repeated.index) {
            start = index;
            break;
        }
    }

    var output = std.Io.Writer.Allocating.init(allocator);
    errdefer output.deinit();
    for (stack[start..]) |id| {
        if (output.written().len != 0) try output.writer.writeAll(" -> ");
        try output.writer.writeAll(table.modules[id.index].name);
    }
    if (output.written().len != 0) try output.writer.writeAll(" -> ");
    try output.writer.writeAll(table.modules[repeated.index].name);
    return output.toOwnedSlice();
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

test "module table preserves raw imports" {
    const parser = @import("parser.zig");

    var diagnostics_bag = diagnostics.DiagnosticBag.init(testing.allocator);
    defer diagnostics_bag.deinit();

    var source_main = try source.SourceFile.init(testing.allocator, "Main.concept", "module Main;\nimport Math;\nimport Compiler.Lexer;\n");
    defer source_main.deinit(testing.allocator);
    var unit_main = try parser.parseSource(testing.allocator, source_main, &diagnostics_bag);
    defer unit_main.deinit(testing.allocator);

    var source_math = try source.SourceFile.init(testing.allocator, "Math.concept", "module Math;\n");
    defer source_math.deinit(testing.allocator);
    var unit_math = try parser.parseSource(testing.allocator, source_math, &diagnostics_bag);
    defer unit_math.deinit(testing.allocator);

    var source_lexer = try source.SourceFile.init(testing.allocator, "Lexer.concept", "module Compiler.Lexer;\n");
    defer source_lexer.deinit(testing.allocator);
    var unit_lexer = try parser.parseSource(testing.allocator, source_lexer, &diagnostics_bag);
    defer unit_lexer.deinit(testing.allocator);

    var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "Main.concept", .unit = &unit_main }, .{ .path = "Math.concept", .unit = &unit_math }, .{ .path = "Lexer.concept", .unit = &unit_lexer } }, &diagnostics_bag);
    defer table.deinit(testing.allocator);

    try testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    try testing.expectEqual(@as(usize, 3), table.modules.len);
    try testing.expectEqual(@as(usize, 2), table.modules[0].imports.len);
    try testing.expectEqualStrings("Math", table.modules[0].imports[0].name);
    try testing.expectEqualStrings("Compiler.Lexer", table.modules[0].imports[1].name);
    try testing.expectEqual(@as(usize, 13), table.modules[0].imports[0].import_span.start);
    try testing.expectEqual(@as(usize, 20), table.modules[0].imports[0].path_span.start);
}

fn parseUnitForTest(allocator: std.mem.Allocator, path: []const u8, text: []const u8, diagnostic_bag: *diagnostics.DiagnosticBag) !ast.CompilationUnit {
    const parser = @import("parser.zig");
    var source_file = try source.SourceFile.init(allocator, path, text);
    defer source_file.deinit(allocator);
    return parser.parseSource(allocator, source_file, diagnostic_bag);
}

test "module table resolves a single import to module id" {
    var bag = diagnostics.DiagnosticBag.init(testing.allocator);
    defer bag.deinit();
    var main = try parseUnitForTest(testing.allocator, "Main.concept", "module Main;\nimport Math;\n", &bag);
    defer main.deinit(testing.allocator);
    var math = try parseUnitForTest(testing.allocator, "Math.concept", "module Math;\n", &bag);
    defer math.deinit(testing.allocator);

    var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "Main.concept", .unit = &main }, .{ .path = "Math.concept", .unit = &math } }, &bag);
    defer table.deinit(testing.allocator);

    try testing.expectEqual(@as(usize, 0), bag.count());
    try testing.expectEqual(@as(usize, 1), table.modules[0].resolved_imports.len);
    try testing.expectEqual(@as(usize, 1), table.modules[0].resolved_imports[0].index);
}

test "module table preserves resolved import order and dotted imports" {
    var bag = diagnostics.DiagnosticBag.init(testing.allocator);
    defer bag.deinit();
    var main = try parseUnitForTest(testing.allocator, "Main.concept", "module Main;\nimport Geometry;\nimport Compiler.Lexer;\n", &bag);
    defer main.deinit(testing.allocator);
    var geometry = try parseUnitForTest(testing.allocator, "Geometry.concept", "module Geometry;\n", &bag);
    defer geometry.deinit(testing.allocator);
    var lexer = try parseUnitForTest(testing.allocator, "Lexer.concept", "module Compiler.Lexer;\n", &bag);
    defer lexer.deinit(testing.allocator);

    var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "Main.concept", .unit = &main }, .{ .path = "Geometry.concept", .unit = &geometry }, .{ .path = "Lexer.concept", .unit = &lexer } }, &bag);
    defer table.deinit(testing.allocator);

    try testing.expectEqual(@as(usize, 0), bag.count());
    try testing.expectEqual(@as(usize, 2), table.modules[0].resolved_imports.len);
    try testing.expectEqual(@as(usize, 1), table.modules[0].resolved_imports[0].index);
    try testing.expectEqual(@as(usize, 2), table.modules[0].resolved_imports[1].index);
}

test "module table rejects duplicate and unknown imports" {
    var bag = diagnostics.DiagnosticBag.init(testing.allocator);
    defer bag.deinit();
    var main = try parseUnitForTest(testing.allocator, "Main.concept", "module Main;\nimport Math;\nimport Math;\nimport Missing;\n", &bag);
    defer main.deinit(testing.allocator);
    var math = try parseUnitForTest(testing.allocator, "Math.concept", "module Math;\n", &bag);
    defer math.deinit(testing.allocator);

    var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "Main.concept", .unit = &main }, .{ .path = "Math.concept", .unit = &math } }, &bag);
    defer table.deinit(testing.allocator);

    try testing.expectEqual(@as(usize, 2), bag.count());
    try testing.expectEqual(.DuplicateImport, bag.diagnostics.items[0].code);
    try testing.expectEqual(.UnknownImport, bag.diagnostics.items[1].code);
}

test "module table detects self direct and longer cycles" {
    const cases = [_]struct { a: []const u8, b: []const u8, c: []const u8, count: usize }{
        .{ .a = "module A;\nimport A;\n", .b = "module B;\n", .c = "module C;\n", .count = 1 },
        .{ .a = "module A;\nimport B;\n", .b = "module B;\nimport A;\n", .c = "module C;\n", .count = 1 },
        .{ .a = "module A;\nimport B;\n", .b = "module B;\nimport C;\n", .c = "module C;\nimport A;\n", .count = 1 },
    };
    for (cases) |case| {
        var bag = diagnostics.DiagnosticBag.init(testing.allocator);
        defer bag.deinit();
        var a_unit = try parseUnitForTest(testing.allocator, "A.concept", case.a, &bag);
        defer a_unit.deinit(testing.allocator);
        var b_unit = try parseUnitForTest(testing.allocator, "B.concept", case.b, &bag);
        defer b_unit.deinit(testing.allocator);
        var c_unit = try parseUnitForTest(testing.allocator, "C.concept", case.c, &bag);
        defer c_unit.deinit(testing.allocator);
        var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "A.concept", .unit = &a_unit }, .{ .path = "B.concept", .unit = &b_unit }, .{ .path = "C.concept", .unit = &c_unit } }, &bag);
        defer table.deinit(testing.allocator);
        try testing.expectEqual(case.count, bag.count());
        try testing.expectEqual(.ImportCycle, bag.diagnostics.items[0].code);
    }
}

test "module table allows shared dependencies and duplicate imports in different modules" {
    var bag = diagnostics.DiagnosticBag.init(testing.allocator);
    defer bag.deinit();
    var main = try parseUnitForTest(testing.allocator, "Main.concept", "module Main;\nimport Math;\n", &bag);
    defer main.deinit(testing.allocator);
    var tests = try parseUnitForTest(testing.allocator, "Tests.concept", "module Tests;\nimport Math;\n", &bag);
    defer tests.deinit(testing.allocator);
    var math = try parseUnitForTest(testing.allocator, "Math.concept", "module Math;\nimport Common;\n", &bag);
    defer math.deinit(testing.allocator);
    var common = try parseUnitForTest(testing.allocator, "Common.concept", "module Common;\n", &bag);
    defer common.deinit(testing.allocator);

    var table = try buildFromParsedSources(testing.allocator, &.{ .{ .path = "Main.concept", .unit = &main }, .{ .path = "Tests.concept", .unit = &tests }, .{ .path = "Math.concept", .unit = &math }, .{ .path = "Common.concept", .unit = &common } }, &bag);
    defer table.deinit(testing.allocator);
    try testing.expectEqual(@as(usize, 0), bag.count());
}
