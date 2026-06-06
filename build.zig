const std = @import("std");

pub fn build(b: *std.Build) void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const parser_tests = b.addTest(.{
        .root_module = b.createModule(.{
            .root_source_file = b.path("src/compiler/parser.zig"),
            .target = target,
            .optimize = optimize,
        }),
    });

    const run_parser_tests = b.addRunArtifact(parser_tests);

    const test_step = b.step("test", "Run compiler tests");
    test_step.dependOn(&run_parser_tests.step);
}
