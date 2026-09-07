const std = @import("std");

const hir = @import("hir.zig");
const semantics = @import("semantics.zig");
const source = @import("source.zig");

pub const SourceFileKind = source.SourceFileKind;
pub const SourceSpan = source.SourceSpan;

pub const TestAttributeKind = enum {
    fact,
    theory,
};

pub const DiscoveredTest = struct {
    module_name: []const u8,
    function_name: []const u8,
    attribute_kind: TestAttributeKind,
    inline_data_row_index: ?usize = null,
    inline_data_args: []const hir.HirAttributeArg = &.{},
    source_span: SourceSpan,
    inline_data_count: usize,
    source_file_kind: SourceFileKind,
};

pub fn discoverTests(
    allocator: std.mem.Allocator,
    module: *const semantics.SemanticModule,
    module_name: []const u8,
    source_file_kind: SourceFileKind,
) ![]DiscoveredTest {
    if (source_file_kind != .@"test") return &.{};

    var discovered = std.ArrayList(DiscoveredTest).empty;
    errdefer discovered.deinit(allocator);

    for (module.hir.functions.items) |function| {
        var maybe_kind: ?TestAttributeKind = null;
        var inline_data_count: usize = 0;

        for (function.attributes) |attribute| {
            const name = module.interner.text(attribute.name);
            if (std.mem.eql(u8, name, "Fact")) {
                maybe_kind = .fact;
            } else if (std.mem.eql(u8, name, "Theory")) {
                maybe_kind = .theory;
            } else if (std.mem.eql(u8, name, "InlineData")) {
                inline_data_count += 1;
            }
        }

        if (maybe_kind) |kind| {
            switch (kind) {
                .fact => try discovered.append(allocator, .{
                    .module_name = module_name,
                    .function_name = module.interner.text(function.name),
                    .attribute_kind = kind,
                    .source_span = function.span,
                    .inline_data_count = inline_data_count,
                    .source_file_kind = source_file_kind,
                }),
                .theory => {
                    var row_index: usize = 0;
                    for (function.attributes) |attribute| {
                        const name = module.interner.text(attribute.name);
                        if (!std.mem.eql(u8, name, "InlineData")) continue;
                        try discovered.append(allocator, .{
                            .module_name = module_name,
                            .function_name = module.interner.text(function.name),
                            .attribute_kind = kind,
                            .inline_data_row_index = row_index,
                            .inline_data_args = attribute.args,
                            .source_span = attribute.span,
                            .inline_data_count = inline_data_count,
                            .source_file_kind = source_file_kind,
                        });
                        row_index += 1;
                    }
                },
            }
        }
    }

    return discovered.toOwnedSlice(allocator);
}

fn addFunction(module: *semantics.SemanticModule, name: []const u8) !hir.FunctionId {
    return module.hir.addFunction(
        try module.interner.intern(name),
        module.types.voidType(),
        .{ .start = 0, .length = name.len },
    );
}

fn setAttributes(module: *semantics.SemanticModule, function_id: hir.FunctionId, names: []const []const u8) !void {
    const attributes = try std.testing.allocator.alloc(hir.HirAttribute, names.len);
    for (names, 0..) |name, index| {
        attributes[index] = .{
            .name = try module.interner.intern(name),
            .args = &.{},
            .has_arguments = std.mem.eql(u8, name, "InlineData"),
            .span = .{ .start = index, .length = name.len },
        };
    }
    module.hir.setFunctionAttributes(function_id, attributes);
}

test "discovers Fact function in test source" {
    var module = try semantics.SemanticModule.init(std.testing.allocator);
    defer module.deinit();
    const function_id = try addFunction(&module, "AddsIntegers");
    try setAttributes(&module, function_id, &.{"Fact"});

    const tests = try discoverTests(std.testing.allocator, &module, "Math.Tests", .@"test");
    defer std.testing.allocator.free(tests);

    try std.testing.expectEqual(@as(usize, 1), tests.len);
    try std.testing.expectEqual(TestAttributeKind.fact, tests[0].attribute_kind);
    try std.testing.expectEqualStrings("Math.Tests", tests[0].module_name);
    try std.testing.expectEqualStrings("AddsIntegers", tests[0].function_name);
    try std.testing.expectEqual(@as(usize, 0), tests[0].inline_data_count);
    try std.testing.expectEqual(SourceFileKind.@"test", tests[0].source_file_kind);
}

test "discovers Theory function as one case per InlineData row" {
    var module = try semantics.SemanticModule.init(std.testing.allocator);
    defer module.deinit();
    const function_id = try addFunction(&module, "AddsIntegers");
    try setAttributes(&module, function_id, &.{ "Theory", "InlineData", "InlineData" });

    const tests = try discoverTests(std.testing.allocator, &module, "Math.Tests", .@"test");
    defer std.testing.allocator.free(tests);

    try std.testing.expectEqual(@as(usize, 2), tests.len);
    try std.testing.expectEqual(TestAttributeKind.theory, tests[0].attribute_kind);
    try std.testing.expectEqualStrings("AddsIntegers", tests[0].function_name);
    try std.testing.expectEqual(@as(usize, 2), tests[0].inline_data_count);
    try std.testing.expectEqual(@as(?usize, 0), tests[0].inline_data_row_index);
    try std.testing.expectEqual(@as(?usize, 1), tests[1].inline_data_row_index);
}

test "helper functions in test source are not discovered" {
    var module = try semantics.SemanticModule.init(std.testing.allocator);
    defer module.deinit();
    _ = try addFunction(&module, "helper");
    const test_id = try addFunction(&module, "UsesHelper");
    try setAttributes(&module, test_id, &.{"Fact"});

    const tests = try discoverTests(std.testing.allocator, &module, "Helper.Tests", .@"test");
    defer std.testing.allocator.free(tests);

    try std.testing.expectEqual(@as(usize, 1), tests.len);
    try std.testing.expectEqualStrings("UsesHelper", tests[0].function_name);
}

test "normal source is ignored by discovery" {
    var module = try semantics.SemanticModule.init(std.testing.allocator);
    defer module.deinit();
    _ = try addFunction(&module, "helper");

    const tests = try discoverTests(std.testing.allocator, &module, "Main", .normal);
    defer std.testing.allocator.free(tests);

    try std.testing.expectEqual(@as(usize, 0), tests.len);
}
