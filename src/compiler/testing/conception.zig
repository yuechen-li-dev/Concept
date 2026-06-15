// ─────────────────────────────────────────────────────────────────────────────
// Fixture metadata model
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const parser_model = @import("../parser.zig");
const module_table_model = @import("../module_table.zig");
const source_model = @import("../source.zig");
const checker_model = @import("../checker.zig");
const semantics_model = @import("../semantics.zig");
const hir_checker_model = @import("../hir_checker.zig");
const mir_lowering_model = @import("../mir_lowering.zig");
const mir_validator_model = @import("../mir_validator.zig");
const backend_c_mir_model = @import("../backend_c_mir.zig");
const run_harness = @import("run_harness.zig");

pub const Phase = enum {
    lex,
    parse,
    run,
    check,
    mir,
    backend_c,

    pub fn fromString(value: []const u8) ?Phase {
        if (std.mem.eql(u8, value, "lex")) return .lex;
        if (std.mem.eql(u8, value, "parse")) return .parse;
        if (std.mem.eql(u8, value, "run")) return .run;
        if (std.mem.eql(u8, value, "check")) return .check;
        if (std.mem.eql(u8, value, "mir")) return .mir;
        if (std.mem.eql(u8, value, "backend-c")) return .backend_c;
        return null;
    }
};

pub const CheckMode = enum {
    declarations,
    hir,

    pub fn parse(value: []const u8) ?CheckMode {
        if (std.mem.eql(u8, value, "declarations")) return .declarations;
        if (std.mem.eql(u8, value, "hir")) return .hir;
        return null;
    }
};

pub const Expectation = enum {
    pass,
    fail,

    pub fn parse(value: []const u8) ?Expectation {
        if (std.mem.eql(u8, value, "pass")) return .pass;
        if (std.mem.eql(u8, value, "fail")) return .fail;
        return null;
    }
};

pub const Section = struct {
    name: []const u8,
    body: []const u8,
};

pub const FixtureSource = struct {
    path: []const u8,
    text: []const u8,
};

pub const ConceptionFixture = struct {
    name: []const u8,
    phase: Phase,
    expect: Expectation,
    check: ?CheckMode = null,
    sections: []const Section,
    sources: []const FixtureSource,

    pub fn checkMode(self: ConceptionFixture) CheckMode {
        return self.check orelse .declarations;
    }

    pub fn source(self: ConceptionFixture) ?[]const u8 {
        if (self.sources.len == 1) return self.sources[0].text;
        return null;
    }

    pub fn isMultiSource(self: ConceptionFixture) bool {
        return self.sources.len > 1;
    }

    pub fn singleSourceText(self: ConceptionFixture) ![]const u8 {
        if (self.sources.len != 1) return error.MultiSourceCompilationDeferred;
        return self.sources[0].text;
    }

    pub fn ast(self: ConceptionFixture) ?[]const u8 {
        return self.section("ast");
    }

    pub fn diagnostics(self: ConceptionFixture) ?[]const u8 {
        return self.section("diagnostics");
    }

    pub fn run(self: ConceptionFixture) ?[]const u8 {
        return self.section("run");
    }

    pub fn mir(self: ConceptionFixture) ?[]const u8 {
        return self.section("mir");
    }

    pub fn c(self: ConceptionFixture) ?[]const u8 {
        return self.section("c");
    }

    pub fn expectedExitCode(self: ConceptionFixture) !u8 {
        const text = self.run() orelse return error.MissingRun;
        var lines = std.mem.splitScalar(u8, text, '\n');
        while (lines.next()) |line| {
            const trimmed = std.mem.trim(u8, line, " \t\r");
            if (trimmed.len == 0) continue;
            if (trimmed[0] == '#') continue;
            const colon = std.mem.indexOfScalar(u8, trimmed, ':') orelse return error.InvalidRunExpectation;
            const key = std.mem.trim(u8, trimmed[0..colon], " \t");
            const value = std.mem.trim(u8, trimmed[colon + 1 ..], " \t");
            if (!std.mem.eql(u8, key, "exit_code")) return error.InvalidRunExpectation;
            return std.fmt.parseInt(u8, value, 10) catch return error.InvalidRunExpectation;
        }
        return error.MissingRunExitCode;
    }

    pub fn section(self: ConceptionFixture, name: []const u8) ?[]const u8 {
        for (self.sections) |candidate| {
            if (std.mem.eql(u8, candidate.name, name)) return candidate.body;
        }
        return null;
    }

    pub fn diagnosticCodes(self: ConceptionFixture, allocator: std.mem.Allocator) ![][]const u8 {
        const text = self.diagnostics() orelse return error.MissingDiagnostics;
        var codes = std.ArrayList([]const u8).init(allocator);
        errdefer codes.deinit();

        var lines = std.mem.splitScalar(u8, text, '\n');
        while (lines.next()) |line| {
            const trimmed = std.mem.trim(u8, line, " \t\r");
            if (trimmed.len == 0) continue;
            if (trimmed[0] == '#') continue;
            const code = firstWord(trimmed);
            try codes.append(code);
        }

        return codes.toOwnedSlice();
    }

    pub fn deinit(self: ConceptionFixture, allocator: std.mem.Allocator) void {
        allocator.free(self.sections);
        allocator.free(self.sources);
    }
};

pub const ParseOptions = struct {
    path: ?[]const u8 = null,
};

// ─────────────────────────────────────────────────────────────────────────────
// Fixture parser
// ─────────────────────────────────────────────────────────────────────────────

pub fn parse(allocator: std.mem.Allocator, text: []const u8, options: ParseOptions) !ConceptionFixture {
    var name: ?[]const u8 = null;
    var phase: ?Phase = null;
    var expect: ?Expectation = null;
    var check: ?CheckMode = null;
    var sections = std.ArrayList(Section).init(allocator);
    errdefer sections.deinit();
    var sources = std.ArrayList(FixtureSource).init(allocator);
    errdefer sources.deinit();

    var cursor: usize = 0;
    var current_section_name: ?[]const u8 = null;
    var current_section_start: usize = 0;
    var saw_section = false;

    while (cursor < text.len) {
        const line_start = cursor;
        const newline_index = std.mem.indexOfScalarPos(u8, text, cursor, '\n') orelse text.len;
        const line_end_without_cr = if (newline_index > line_start and text[newline_index - 1] == '\r') newline_index - 1 else newline_index;
        const line = text[line_start..line_end_without_cr];
        cursor = if (newline_index == text.len) text.len else newline_index + 1;

        if (sectionName(line)) |section_name| {
            if (current_section_name) |previous_name| {
                try sections.append(.{ .name = previous_name, .body = trimTrailingLineEnding(text[current_section_start..line_start]) });
            }
            current_section_name = section_name;
            current_section_start = cursor;
            saw_section = true;
            continue;
        }

        if (!saw_section) {
            const trimmed = std.mem.trim(u8, line, " \t");
            if (trimmed.len == 0) continue;
            if (!std.mem.startsWith(u8, trimmed, "#")) return error.ExpectedHeaderOrSection;
            try parseHeader(trimmed, &name, &phase, &expect, &check);
        }
    }

    if (current_section_name) |previous_name| {
        try sections.append(.{ .name = previous_name, .body = trimTrailingLineEnding(text[current_section_start..text.len]) });
    }

    try collectSources(sections.items, &sources, options);

    const parsed = ConceptionFixture{
        .name = name orelse return error.MissingName,
        .phase = phase orelse return error.MissingPhase,
        .expect = expect orelse return error.MissingExpectation,
        .check = check,
        .sections = try sections.toOwnedSlice(),
        .sources = try sources.toOwnedSlice(),
    };
    errdefer parsed.deinit(allocator);

    try validate(parsed, options);
    return parsed;
}

// ─────────────────────────────────────────────────────────────────────────────
// Header parsing
// ─────────────────────────────────────────────────────────────────────────────

fn parseHeader(line: []const u8, name: *?[]const u8, phase: *?Phase, expect: *?Expectation, check: *?CheckMode) !void {
    const without_hash = std.mem.trim(u8, line[1..], " \t");
    const colon = std.mem.indexOfScalar(u8, without_hash, ':') orelse return;
    const key = std.mem.trim(u8, without_hash[0..colon], " \t");
    const value = std.mem.trim(u8, without_hash[colon + 1 ..], " \t");

    if (std.mem.eql(u8, key, "name")) {
        if (name.* != null) return error.DuplicateHeader;
        if (value.len == 0) return error.EmptyName;
        name.* = value;
    } else if (std.mem.eql(u8, key, "phase")) {
        if (phase.* != null) return error.DuplicateHeader;
        phase.* = Phase.fromString(value) orelse return error.InvalidPhase;
    } else if (std.mem.eql(u8, key, "expect")) {
        if (expect.* != null) return error.DuplicateHeader;
        expect.* = Expectation.parse(value) orelse return error.InvalidExpectation;
    } else if (std.mem.eql(u8, key, "check")) {
        if (check.* != null) return error.DuplicateHeader;
        check.* = CheckMode.parse(value) orelse return error.InvalidCheckMode;
    } else {
        return error.UnknownHeader;
    }
}

fn collectSources(
    sections: []const Section,
    sources: *std.ArrayList(FixtureSource),
    options: ParseOptions,
) !void {
    var saw_file_section = false;
    for (sections) |section| {
        if (std.mem.startsWith(u8, section.name, "file:")) {
            const path = fileSectionPath(section.name) orelse return error.MissingVirtualFilePath;
            saw_file_section = true;
            for (sources.items) |existing| {
                if (std.mem.eql(u8, existing.path, path)) return error.DuplicateVirtualFilePath;
            }
            try sources.append(.{ .path = path, .text = section.body });
        }
    }

    if (saw_file_section) return;

    for (sections) |section| {
        if (std.mem.eql(u8, section.name, "source")) {
            try sources.append(.{
                .path = options.path orelse "<source>",
                .text = section.body,
            });
            return;
        }
    }
}

fn fileSectionPath(name: []const u8) ?[]const u8 {
    if (!std.mem.startsWith(u8, name, "file:")) return null;
    const path = std.mem.trim(u8, name[5..], " \t");
    if (path.len == 0) return null;
    return path;
}

// ─────────────────────────────────────────────────────────────────────────────
// Validation
// ─────────────────────────────────────────────────────────────────────────────

fn validate(fixture: ConceptionFixture, options: ParseOptions) !void {
    if (fixture.sources.len == 0) return error.MissingSource;

    if (fixture.check != null and fixture.phase != .check) return error.CheckModeOnNonCheckPhase;

    switch (fixture.phase) {
        .parse => switch (fixture.expect) {
            .pass => if (!fixture.isMultiSource() and fixture.ast() == null) return error.MissingAst,
            .fail => if (fixture.diagnostics() == null) return error.MissingDiagnostics,
        },
        .run => {
            if (fixture.expect == .pass) {
                _ = try fixture.expectedExitCode();
            }
        },
        else => {},
    }

    if (options.path) |path| {
        if (std.mem.endsWith(u8, path, ".valid.conception") and fixture.expect != .pass) return error.ExtensionExpectationMismatch;
        if (std.mem.endsWith(u8, path, ".invalid.conception") and fixture.expect != .fail) return error.ExtensionExpectationMismatch;
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Section parsing
// ─────────────────────────────────────────────────────────────────────────────

fn sectionName(line: []const u8) ?[]const u8 {
    const trimmed = std.mem.trim(u8, line, " \t\r");
    if (!std.mem.startsWith(u8, trimmed, "===")) return null;
    if (!std.mem.endsWith(u8, trimmed, "===")) return null;
    const inner = std.mem.trim(u8, trimmed[3 .. trimmed.len - 3], " \t");
    if (inner.len == 0) return null;
    return inner;
}

fn trimTrailingLineEnding(text: []const u8) []const u8 {
    var end = text.len;
    if (end > 0 and text[end - 1] == '\n') end -= 1;
    if (end > 0 and text[end - 1] == '\r') end -= 1;
    return text[0..end];
}

fn firstWord(text: []const u8) []const u8 {
    const index = std.mem.indexOfAny(u8, text, " \t\r") orelse text.len;
    return text[0..index];
}

// ─────────────────────────────────────────────────────────────────────────────
// Parser tests
// ─────────────────────────────────────────────────────────────────────────────

fn expectFixture(text: []const u8) !ConceptionFixture {
    return parse(std.testing.allocator, text, .{});
}

const sample_fixture =
    \\# name: payload enum surface
    \\# phase: parse
    \\# expect: pass
    \\
    \\=== source ===
    \\module Example;
    \\
    \\=== ast ===
    \\CompilationUnit
    \\  Module Example
;

test "conception parser parses metadata" {
    const fixture = try expectFixture(sample_fixture);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("payload enum surface", fixture.name);
    try std.testing.expectEqual(Phase.parse, fixture.phase);
    try std.testing.expectEqual(Expectation.pass, fixture.expect);
}

test "conception parser rejects unknown header" {
    const text =
        \\# name: unknown header
        \\# phase: parse
        \\# expectation: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.UnknownHeader, expectFixture(text));
}

test "conception parser rejects duplicate name header" {
    const text =
        \\# name: duplicate name
        \\# name: duplicate name again
        \\# phase: parse
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.DuplicateHeader, expectFixture(text));
}

test "conception parser rejects duplicate phase header" {
    const text =
        \\# name: duplicate phase
        \\# phase: parse
        \\# phase: check
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.DuplicateHeader, expectFixture(text));
}

test "conception parser rejects duplicate expect header" {
    const text =
        \\# name: duplicate expect
        \\# phase: parse
        \\# expect: pass
        \\# expect: fail
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.DuplicateHeader, expectFixture(text));
}

test "conception parser rejects duplicate check header" {
    const text =
        \\# name: duplicate check
        \\# phase: check
        \\# check: declarations
        \\# check: hir
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.DuplicateHeader, expectFixture(text));
}

test "conception parser accepts valid headers and leading comments" {
    const text =
        \\# parser smoke comment
        \\
        \\# name: valid strict headers
        \\# phase: check
        \\# check: hir
        \\# expect: pass
        \\# another comment before sections
        \\
        \\=== source ===
        \\module Example;
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("valid strict headers", fixture.name);
    try std.testing.expectEqual(Phase.check, fixture.phase);
    try std.testing.expectEqual(CheckMode.hir, fixture.checkMode());
    try std.testing.expectEqual(Expectation.pass, fixture.expect);
}

test "conception parser parses source section" {
    const fixture = try expectFixture(sample_fixture);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("module Example;", try fixture.singleSourceText());
}

test "conception parser maps legacy source section to one fixture source" {
    const fixture = try expectFixture(sample_fixture);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), fixture.sources.len);
    try std.testing.expectEqualStrings("<source>", fixture.sources[0].path);
    try std.testing.expectEqualStrings("module Example;", fixture.sources[0].text);
}

test "conception parser parses multi-file sources in fixture order" {
    const text =
        \\# name: multi source
        \\# phase: parse
        \\# expect: pass
        \\
        \\=== file: Math.concept ===
        \\module Math;
        \\
        \\int Add() {
        \\    return 1;
        \\}
        \\
        \\=== file: app/Main.concept ===
        \\module Main;
        \\
        \\int main() {
        \\    return 0;
        \\}
        \\
        \\=== run ===
        \\exit_code: 0
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expect(fixture.isMultiSource());
    try std.testing.expectEqual(@as(usize, 2), fixture.sources.len);
    try std.testing.expectEqualStrings("Math.concept", fixture.sources[0].path);
    try std.testing.expectEqualStrings("module Math;\n\nint Add() {\n    return 1;\n}", fixture.sources[0].text);
    try std.testing.expectEqualStrings("app/Main.concept", fixture.sources[1].path);
    try std.testing.expect(std.mem.indexOf(u8, fixture.sources[1].text, "module Main;") != null);
    try std.testing.expectEqual(@as(u8, 0), try fixture.expectedExitCode());
}

test "conception parser rejects duplicate virtual file paths" {
    const text =
        \\# name: duplicate virtual path
        \\# phase: parse
        \\# expect: pass
        \\
        \\=== file: A.concept ===
        \\module A;
        \\
        \\=== file: A.concept ===
        \\module B;
    ;
    try std.testing.expectError(error.DuplicateVirtualFilePath, expectFixture(text));
}

test "conception parser rejects missing virtual file path" {
    const text =
        \\# name: missing virtual path
        \\# phase: parse
        \\# expect: pass
        \\
        \\=== file: ===
        \\module A;
    ;
    try std.testing.expectError(error.MissingVirtualFilePath, expectFixture(text));
}

test "conception parser parses ast section" {
    const fixture = try expectFixture(sample_fixture);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("CompilationUnit\n  Module Example", fixture.ast().?);
}

test "conception parser parses diagnostics section" {
    const text =
        \\# name: invalid sample
        \\# phase: parse
        \\# expect: fail
        \\
        \\=== source ===
        \\import Core.Memory;
        \\
        \\=== diagnostics ===
        \\CON0004
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);

    const codes = try fixture.diagnosticCodes(std.testing.allocator);
    defer std.testing.allocator.free(codes);
    try std.testing.expectEqual(@as(usize, 1), codes.len);
    try std.testing.expectEqualStrings("CON0004", codes[0]);
}

test "conception parser parses run exit code" {
    const text =
        \\# name: arithmetic return
        \\# phase: run
        \\# expect: pass
        \\
        \\=== source ===
        \\module Main;
        \\
        \\int main() {
        \\    return 1 + 2 * 3;
        \\}
        \\
        \\=== run ===
        \\exit_code: 7
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.run, fixture.phase);
    try std.testing.expectEqual(@as(u8, 7), try fixture.expectedExitCode());
}

test "conception parser parses declarations check mode" {
    const text =
        \\# name: declarations check
        \\# phase: check
        \\# check: declarations
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);
    try std.testing.expectEqual(CheckMode.declarations, fixture.checkMode());
}

test "conception parser parses hir check mode" {
    const text =
        \\# name: hir check
        \\# phase: check
        \\# check: hir
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);
    try std.testing.expectEqual(CheckMode.hir, fixture.checkMode());
}

test "conception parser rejects invalid check mode" {
    const text =
        \\# name: invalid check
        \\# phase: check
        \\# check: runtime
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.InvalidCheckMode, expectFixture(text));
}

test "conception parser rejects check mode on non-check phase" {
    const text =
        \\# name: parse with check
        \\# phase: parse
        \\# check: declarations
        \\# expect: fail
        \\
        \\=== source ===
        \\module Example;
        \\
        \\=== diagnostics ===
        \\CON0004
    ;
    try std.testing.expectError(error.CheckModeOnNonCheckPhase, expectFixture(text));
}

test "conception parser defaults check phase to declarations" {
    const text =
        \\# name: default check
        \\# phase: check
        \\# expect: pass
        \\
        \\=== source ===
        \\module Example;
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);
    try std.testing.expectEqual(CheckMode.declarations, fixture.checkMode());
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase/corpus tests
// ─────────────────────────────────────────────────────────────────────────────

test "language check fixture execution uses check metadata not filename" {
    const text =
        \\# name: metadata selected hir check
        \\# phase: check
        \\# check: hir
        \\# expect: fail
        \\
        \\=== source ===
        \\module Example;
        \\
        \\int helper() {
        \\    return 0;
        \\}
        \\
        \\=== diagnostics ===
        \\CON0031 error: expected top-level 'main' function
    ;
    const fixture = try expectFixture(text);
    defer fixture.deinit(std.testing.allocator);
    try expectSemanticCheckFixture("metadata_selected.invalid.conception", fixture);
}

test "conception parser rejects missing source" {
    const text =
        \\# name: no source
        \\# phase: parse
        \\# expect: pass
        \\
        \\=== ast ===
        \\CompilationUnit
    ;
    try std.testing.expectError(error.MissingSource, expectFixture(text));
}

test "conception parser rejects invalid expect value" {
    const text =
        \\# name: bad expect
        \\# phase: parse
        \\# expect: maybe
        \\
        \\=== source ===
        \\module Example;
    ;
    try std.testing.expectError(error.InvalidExpectation, expectFixture(text));
}

test "conception parser detects extension and metadata mismatch" {
    try std.testing.expectError(
        error.ExtensionExpectationMismatch,
        parse(std.testing.allocator, sample_fixture, .{ .path = "language/phase1-surface/invalid/sample.invalid.conception" }),
    );
}

// ─────────────────────────────────────────────────────────────────────────────
// Diagnostic/run helpers
// ─────────────────────────────────────────────────────────────────────────────

fn expectParseFixture(comptime path: []const u8) !void {
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.parse, fixture.phase);

    if (fixture.isMultiSource()) {
        var diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
        defer diagnostics.deinit();

        const units = try std.testing.allocator.alloc(parser_model.ast.CompilationUnit, fixture.sources.len);
        defer std.testing.allocator.free(units);
        var parsed_sources = try std.testing.allocator.alloc(module_table_model.ParsedSource, fixture.sources.len);
        defer std.testing.allocator.free(parsed_sources);

        var parsed_count: usize = 0;
        defer {
            for (units[0..parsed_count]) |unit| unit.deinit(std.testing.allocator);
        }

        for (fixture.sources, 0..) |fixture_source, source_index| {
            const source_file = try source_model.SourceFile.init(std.testing.allocator, fixture_source.path, fixture_source.text);
            defer source_file.deinit(std.testing.allocator);

            units[source_index] = try parser_model.parseSource(std.testing.allocator, source_file, &diagnostics);
            parsed_count += 1;
            parsed_sources[source_index] = .{ .path = fixture_source.path, .unit = &units[source_index] };
        }

        var module_table = try module_table_model.buildFromParsedSources(std.testing.allocator, parsed_sources, &diagnostics);
        defer module_table.deinit(std.testing.allocator);

        switch (fixture.expect) {
            .pass => try std.testing.expectEqual(@as(usize, 0), diagnostics.count()),
            .fail => {
                try std.testing.expect(diagnostics.count() > 0);
                const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
                defer std.testing.allocator.free(expected_codes);
                try std.testing.expectEqual(expected_codes.len, diagnostics.count());
                for (expected_codes, diagnostics.diagnostics.items) |expected_code, actual| {
                    try std.testing.expectEqualStrings(expected_code, actual.code.format());
                }
            },
        }
        return;
    }

    var diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, fixture.sources[0].path, fixture.sources[0].text);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &diagnostics);
    defer unit.deinit(std.testing.allocator);

    switch (fixture.expect) {
        .pass => {
            try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
            const snapshot = try unit.debugString(std.testing.allocator);
            defer std.testing.allocator.free(snapshot);
            try std.testing.expectEqualStrings(fixture.ast().?, snapshot);
        },
        .fail => {
            try std.testing.expect(diagnostics.count() > 0);
            const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
            defer std.testing.allocator.free(expected_codes);
            try std.testing.expectEqual(expected_codes.len, diagnostics.count());
            for (expected_codes, diagnostics.diagnostics.items) |expected_code, actual| {
                try std.testing.expectEqualStrings(expected_code, actual.code.format());
            }
        },
    }
}

// ─────────────────────────────────────────────────────────────────────────────
// Fixture registration
// ─────────────────────────────────────────────────────────────────────────────

test "language parse fixture: basic module" {
    try expectParseFixture("../../../language/phase1-surface/valid/basic_module.valid.conception");
}

test "language parse fixture: payload enum" {
    try expectParseFixture("../../../language/phase1-surface/valid/payload_enum.valid.conception");
}

test "language parse fixture: test surface" {
    try expectParseFixture("../../../language/phase1-surface/valid/test_surface.valid.conception");
}

test "language parse fixture: missing module" {
    try expectParseFixture("../../../language/phase1-surface/invalid/missing_module.invalid.conception");
}

test "language parse fixture: malformed enum" {
    try expectParseFixture("../../../language/phase1-surface/invalid/malformed_enum.invalid.conception");
}

test "language parse fixture: phase2 return zero" {
    try expectParseFixture("../../../language/phase2-execution/valid/return_zero.valid.conception");
}

test "language parse fixture: phase2 arithmetic return" {
    try expectParseFixture("../../../language/phase2-execution/valid/arithmetic_return.valid.conception");
}

test "language parse fixture: phase2 parenthesized return" {
    try expectParseFixture("../../../language/phase2-execution/valid/parenthesized_return.valid.conception");
}

test "language parse fixture: phase2 boolean return" {
    try expectParseFixture("../../../language/phase2-execution/valid/boolean_return.valid.conception");
}

test "language parse fixture: phase2 missing return semicolon" {
    try expectParseFixture("../../../language/phase2-execution/invalid/missing_return_semicolon.invalid.conception");
}

test "language parse fixture: phase2 bad expression" {
    try expectParseFixture("../../../language/phase2-execution/invalid/bad_expression.invalid.conception");
}

test "language parse fixture: phase2 missing paren" {
    try expectParseFixture("../../../language/phase2-execution/invalid/missing_paren.invalid.conception");
}

test "language parse fixture: phase2 unsupported statement" {
    try expectParseFixture("../../../language/phase2-execution/invalid/unsupported_statement.invalid.conception");
}

test "language parse fixture: phase8 template identity" {
    try expectParseFixture("../../../language/phase8-concepts-templates/valid/template_identity.valid.conception");
}

test "language parse fixture: phase8 template map get or default" {
    try expectParseFixture("../../../language/phase8-concepts-templates/valid/template_map_get_or_default.valid.conception");
}

test "language parse fixture: phase8 constrained template equal" {
    try expectParseFixture("../../../language/phase8-concepts-templates/valid/template_constrained_equal.valid.conception");
}

test "language parse fixture: phase8 template missing angle" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/template_missing_angle.invalid.conception");
}

test "language parse fixture: phase8 template missing close angle" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/template_missing_close_angle.invalid.conception");
}

test "language parse fixture: phase8 template missing parameter identifier" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/template_missing_param_identifier.invalid.conception");
}

test "language parse fixture: phase8 template duplicate parameter" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/template_duplicate_param.invalid.conception");
}

test "language parse fixture: phase8 template invalid constraint" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/template_invalid_constraint.invalid.conception");
}

test "language check fixture: phase8 concept equatable declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/concept_equatable_decl.valid.conception");
}

test "language check fixture: phase8 concept multiple requirements" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/concept_multiple_requirements.valid.conception");
}

test "language check fixture: phase8 marker concept declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/concept_marker_decl.valid.conception");
}

test "language check fixture: phase8 unsafe marker concept declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/concept_unsafe_marker_decl.valid.conception");
}

test "language check fixture: phase8 concept pointer requirement" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/concept_pointer_requirement.valid.conception");
}

test "language parse fixture: phase8 concept duplicate type parameter" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/concept_duplicate_type_param.invalid.conception");
}

test "language parse fixture: phase8 concept duplicate requirement" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/concept_duplicate_requirement.invalid.conception");
}

test "language parse fixture: phase8 marker concept with body" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/concept_marker_with_body.invalid.conception");
}

test "language parse fixture: phase8 non-marker concept without body" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/concept_non_marker_without_body.invalid.conception");
}

test "language parse fixture: phase8 concept requirement body" {
    try expectParseFixture("../../../language/phase8-concepts-templates/invalid/concept_requirement_body.invalid.conception");
}

test "language check fixture: phase8 concept unknown type in requirement" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/concept_unknown_type_in_requirement.invalid.conception");
}

test "language check fixture: phase8 concept duplicate top level name" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/concept_duplicate_top_level_name.invalid.conception");
}

test "language check fixture: phase8 impl equatable vec2" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/impl_equatable_vec2.valid.conception");
}

test "language check fixture: phase8 impl multiple requirements" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/impl_multiple_requirements.valid.conception");
}

test "language check fixture: phase8 impl marker copy" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/impl_marker_copy.valid.conception");
}

test "language check fixture: phase8 impl unsafe marker" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/impl_unsafe_marker.valid.conception");
}

test "language check fixture: phase8 marker trivial decl" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/marker_trivial_decl.valid.conception");
}

test "language check fixture: phase8 marker known kinds decl" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/marker_known_kinds_decl.valid.conception");
}

test "language check fixture: phase8 unsafe marker impl decl" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/unsafe_marker_impl_decl.valid.conception");
}

test "language check fixture: phase8 impl unknown concept" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_unknown_concept.invalid.conception");
}

test "language check fixture: phase8 impl missing requirement" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_missing_requirement.invalid.conception");
}

test "language check fixture: phase8 impl wrong return type" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_wrong_return_type.invalid.conception");
}

test "language check fixture: phase8 impl wrong param type" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_wrong_param_type.invalid.conception");
}

test "language check fixture: phase8 impl extra function" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_extra_function.invalid.conception");
}

test "language check fixture: phase8 impl duplicate" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_duplicate.invalid.conception");
}

test "language check fixture: phase8 impl marker with function" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_marker_with_function.invalid.conception");
}

test "language check fixture: phase8 impl safe impl for unsafe marker" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_safe_impl_for_unsafe_marker.invalid.conception");
}

test "language check fixture: phase8 impl unsafe impl for safe concept" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/impl_unsafe_impl_for_safe_concept.invalid.conception");
}

test "language check fixture: phase8 marker safe impl for unsafe marker alias" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/unsafe_marker_safe_impl.invalid.conception");
}

test "language check fixture: phase8 safe marker unsafe impl alias" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/safe_marker_unsafe_impl.invalid.conception");
}

test "language check fixture: phase8 marker impl with function alias" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/marker_impl_with_function.invalid.conception");
}

test "language check fixture: phase8 template identity declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/template_identity_decl.valid.conception");
}

test "language check fixture: phase8 template multi parameter declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/template_multi_param_decl.valid.conception");
}

test "language check fixture: phase8 template pointer parameter declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/template_pointer_param_decl.valid.conception");
}

test "language check fixture: phase8 template constrained declaration" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/template_constrained_decl.valid.conception");
}

test "language check fixture: phase8 template declaration with concrete main" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/valid/template_decl_with_concrete_main.valid.conception");
}

test "language check fixture: phase8 template call unsupported" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/template_call_unsupported.invalid.conception");
}

test "language check fixture: phase8 template duplicate concrete name" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/template_duplicate_concrete_name.invalid.conception");
}

test "language check fixture: phase8 template type parameter outside scope" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/template_type_param_outside_scope.invalid.conception");
}

test "language check fixture: phase8 generic main not concrete" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/template_generic_main_not_concrete.invalid.conception");
}

test "language run fixture: phase8 generic identity int" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/generic_identity_int_run.valid.conception");
}

test "language run fixture: phase8 generic identity bool" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/generic_identity_bool_run.valid.conception");
}

test "language run fixture: phase8 generic choose int" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/generic_choose_int_run.valid.conception");
}

test "language run fixture: phase8 generic identity struct" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/generic_identity_struct_run.valid.conception");
}

test "language run fixture: phase8 generic reuse instantiation" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/generic_reuse_instantiation_run.valid.conception");
}

test "language check fixture: phase8 generic call arity mismatch" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/generic_call_arity_mismatch.invalid.conception");
}

test "language check fixture: phase8 generic inference conflict" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/generic_inference_conflict.invalid.conception");
}

test "language check fixture: phase8 generic uninferred type param" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/generic_uninferred_type_param.invalid.conception");
}

test "language run fixture: phase8 constrained equatable vec2" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/constrained_equatable_vec2_run.valid.conception");
}

test "language run fixture: phase8 constrained equatable vec2 false" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/constrained_equatable_vec2_false_run.valid.conception");
}

test "language run fixture: phase8 constrained marker copy" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/constrained_marker_copy_run.valid.conception");
}

test "language run fixture: phase8 marker copy constraint run" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/marker_copy_constraint_run.valid.conception");
}

test "language run fixture: phase8 unsafe marker constraint check" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/unsafe_marker_constraint_check.valid.conception");
}

test "language run fixture: phase8 concepts templates pipeline closeout" {
    try expectRunFixture("../../../language/phase8-concepts-templates/valid/phase8_concepts_templates_pipeline_run.valid.conception");
}

test "language check fixture: phase8 constrained missing impl" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_missing_impl.invalid.conception");
}

test "language check fixture: phase8 constrained unknown concept" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_unknown_concept.invalid.conception");
}

test "language check fixture: phase8 constrained arity mismatch" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_arity_mismatch.invalid.conception");
}

test "language check fixture: phase8 constrained requirement wrong arity" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_requirement_wrong_arity.invalid.conception");
}

test "language check fixture: phase8 constrained requirement wrong arg type" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_requirement_wrong_arg_type.invalid.conception");
}

test "language check fixture: phase8 constrained requirement ambiguous" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_requirement_ambiguous.invalid.conception");
}

test "language check fixture: phase8 constrained marker missing impl" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/constrained_marker_missing_impl.invalid.conception");
}

test "language check fixture: phase8 marker constraint missing impl alias" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/marker_constraint_missing_impl.invalid.conception");
}

test "language check fixture: phase8 unsafe marker constraint missing impl" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/unsafe_marker_constraint_missing_impl.invalid.conception");
}

test "language check fixture: phase8 generic constrained unsupported" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/generic_constrained_call_unsupported.invalid.conception");
}

test "language MIR fixture: phase8 concepts templates pipeline uses concrete MIR" {
    const path = "../../../language/phase8-concepts-templates/valid/phase8_concepts_templates_pipeline_run.valid.conception";
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, try fixture.singleSourceText());
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics_model.collectTopLevelDeclarationsWithOptions(
        std.testing.allocator,
        unit,
        &semantic_diagnostics,
        .{ .source_file_kind = source_file.kind },
    );
    defer module.deinit();
    try hir_checker_model.checkExecutable(std.testing.allocator, &module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    var mir_module = try mir_lowering_model.lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    try mir_validator_model.validateModule(std.testing.allocator, &module, &mir_module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function identity__int -> TypeId(1)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function identity__struct_Vec2 -> TypeId(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function areEqual__struct_Vec2 -> TypeId(2)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call equals(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "type_param") == null);
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase execution helpers
// ─────────────────────────────────────────────────────────────────────────────

fn expectCheckFixture(comptime path: []const u8) !void {
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.check, fixture.phase);

    if (std.mem.indexOf(u8, path, "phase12-allocation") != null) {
        try expectSemanticCheckFixtureAllowNoMain(path, fixture);
    } else if (std.mem.indexOf(u8, path, "phase3-semantics") != null or std.mem.indexOf(u8, path, "phase8-concepts-templates") != null or std.mem.indexOf(u8, path, "phase5-sum-types") != null or std.mem.indexOf(u8, path, "phase5a-judgment") != null or std.mem.indexOf(u8, path, "phase6-unsafe-ownership") != null or std.mem.indexOf(u8, path, "phase7-runtime-structs") != null or std.mem.indexOf(u8, path, "phase10-ownership") != null or std.mem.indexOf(u8, path, "phase11-testing") != null or std.mem.indexOf(u8, path, "phase13-machines") != null or std.mem.indexOf(u8, path, "phase14-interfaces") != null or std.mem.indexOf(u8, path, "phase15-c-abi") != null or std.mem.indexOf(u8, path, "phase16-imports") != null or std.mem.indexOf(u8, path, "phase17-runtime-failure") != null or std.mem.indexOf(u8, path, "phase18-machines") != null or std.mem.indexOf(u8, path, "phase19-yielding-machines") != null) {
        try expectSemanticCheckFixture(path, fixture);
    } else {
        try expectPhase2CheckFixture(path, fixture);
    }
}

fn expectSemanticMultiSourceCheckFixture(fixture: ConceptionFixture) !void {
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const units = try std.testing.allocator.alloc(parser_model.ast.CompilationUnit, fixture.sources.len);
    defer std.testing.allocator.free(units);
    var parsed_sources = try std.testing.allocator.alloc(module_table_model.ParsedSource, fixture.sources.len);
    defer std.testing.allocator.free(parsed_sources);

    var parsed_count: usize = 0;
    defer for (units[0..parsed_count]) |unit| unit.deinit(std.testing.allocator);

    for (fixture.sources, 0..) |fixture_source, source_index| {
        const source_file = try source_model.SourceFile.init(std.testing.allocator, fixture_source.path, fixture_source.text);
        defer source_file.deinit(std.testing.allocator);
        units[source_index] = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
        parsed_count += 1;
        parsed_sources[source_index] = .{ .path = fixture_source.path, .unit = &units[source_index] };
    }
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module_table = try module_table_model.buildFromParsedSources(std.testing.allocator, parsed_sources, &semantic_diagnostics);
    defer module_table.deinit(std.testing.allocator);

    const maybe_module = semantics_model.collectModuleTableDeclarations(
        std.testing.allocator,
        parsed_sources,
        module_table,
        &semantic_diagnostics,
        .{},
    ) catch |err| switch (err) {
        error.InvalidSemanticModule => null,
        else => return err,
    };
    defer if (maybe_module) |*module| module.deinit();

    switch (fixture.expect) {
        .pass => {
            try std.testing.expect(maybe_module != null);
            try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());
        },
        .fail => {
            try std.testing.expect(semantic_diagnostics.count() > 0);
            const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
            defer std.testing.allocator.free(expected_codes);
            try std.testing.expectEqual(expected_codes.len, semantic_diagnostics.count());
            for (expected_codes, semantic_diagnostics.diagnostics.items) |expected_code, actual| {
                try std.testing.expectEqualStrings(expected_code, actual.code.format());
            }
        },
    }
}

fn expectSemanticCheckFixture(comptime path: []const u8, fixture: ConceptionFixture) !void {
    if (fixture.isMultiSource()) return expectSemanticMultiSourceCheckFixture(fixture);
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, try fixture.singleSourceText());
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    const use_hir_checker = fixture.checkMode() == .hir;

    switch (fixture.expect) {
        .pass => {
            var module = try semantics_model.collectTopLevelDeclarationsWithOptions(
                std.testing.allocator,
                unit,
                &semantic_diagnostics,
                .{ .source_file_kind = source_file.kind },
            );
            defer module.deinit();
            if (use_hir_checker) {
                try hir_checker_model.checkExecutable(std.testing.allocator, &module, &semantic_diagnostics);
            }
            try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());
        },
        .fail => {
            var maybe_module = semantics_model.collectTopLevelDeclarationsWithOptions(
                std.testing.allocator,
                unit,
                &semantic_diagnostics,
                .{ .source_file_kind = source_file.kind },
            ) catch |err| switch (err) {
                error.InvalidSemanticModule => null,
                else => return err,
            };
            defer if (maybe_module) |*module| module.deinit();
            if (maybe_module) |*module| {
                if (use_hir_checker) {
                    try std.testing.expectError(
                        error.InvalidSemanticModule,
                        hir_checker_model.checkExecutable(std.testing.allocator, module, &semantic_diagnostics),
                    );
                } else {
                    return error.ExpectedSemanticDiagnostic;
                }
            }
            const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
            defer std.testing.allocator.free(expected_codes);
            try std.testing.expectEqual(expected_codes.len, semantic_diagnostics.count());
            for (expected_codes, semantic_diagnostics.diagnostics.items) |expected_code, actual| {
                try std.testing.expectEqualStrings(expected_code, actual.code.format());
            }
        },
    }
}

fn expectSemanticCheckFixtureAllowNoMain(comptime path: []const u8, fixture: ConceptionFixture) !void {
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, try fixture.singleSourceText());
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    const use_hir_checker = fixture.checkMode() == .hir;

    switch (fixture.expect) {
        .pass => {
            var module = try semantics_model.collectTopLevelDeclarationsWithOptions(
                std.testing.allocator,
                unit,
                &semantic_diagnostics,
                .{ .source_file_kind = source_file.kind },
            );
            defer module.deinit();
            if (use_hir_checker) {
                try hir_checker_model.checkTestModule(std.testing.allocator, &module, &semantic_diagnostics);
            }
            try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());
        },
        .fail => {
            var maybe_module = semantics_model.collectTopLevelDeclarationsWithOptions(
                std.testing.allocator,
                unit,
                &semantic_diagnostics,
                .{ .source_file_kind = source_file.kind },
            ) catch |err| switch (err) {
                error.InvalidSemanticModule => null,
                else => return err,
            };
            defer if (maybe_module) |*module| module.deinit();
            if (maybe_module) |*module| {
                if (use_hir_checker) {
                    try std.testing.expectError(
                        error.InvalidSemanticModule,
                        hir_checker_model.checkTestModule(std.testing.allocator, module, &semantic_diagnostics),
                    );
                } else {
                    return error.ExpectedSemanticDiagnostic;
                }
            }
            const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
            defer std.testing.allocator.free(expected_codes);
            try std.testing.expectEqual(expected_codes.len, semantic_diagnostics.count());
            for (expected_codes, semantic_diagnostics.diagnostics.items) |expected_code, actual| {
                try std.testing.expectEqualStrings(expected_code, actual.code.format());
            }
        },
    }
}

fn expectPhase2CheckFixture(comptime path: []const u8, fixture: ConceptionFixture) !void {
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, try fixture.singleSourceText());
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    switch (fixture.expect) {
        .pass => {
            const executable = try checker_model.validateExecutable(std.testing.allocator, unit, &check_diagnostics);
            defer executable.deinit(std.testing.allocator);
            try std.testing.expectEqual(@as(usize, 0), check_diagnostics.count());
        },
        .fail => {
            try std.testing.expectError(error.InvalidExecutable, checker_model.validateExecutable(std.testing.allocator, unit, &check_diagnostics));
            const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
            defer std.testing.allocator.free(expected_codes);
            try std.testing.expectEqual(expected_codes.len, check_diagnostics.count());
            for (expected_codes, check_diagnostics.diagnostics.items) |expected_code, actual| {
                try std.testing.expectEqualStrings(expected_code, actual.code.format());
            }
        },
    }
}

test "language check fixture: phase3 top-level declarations" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/top_level_decls.valid.conception");
}

test "language check fixture: phase3 parsed nonsemantic items" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/parsed_nonsemantic_items.valid.conception");
}

test "language check fixture: phase3 duplicate function" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/duplicate_function.invalid.conception");
}

test "language check fixture: phase3 duplicate function struct" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/duplicate_function_struct.invalid.conception");
}

test "language check fixture: phase3 duplicate struct enum" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/duplicate_struct_enum.invalid.conception");
}

test "language check fixture: phase3 builtin function types" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/builtin_function_types.valid.conception");
}

test "language check fixture: phase3 struct field types" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/struct_field_types.valid.conception");
}

test "language check fixture: phase3 enum payload types" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/enum_payload_types.valid.conception");
}

test "language check fixture: phase3 user type references" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/user_type_references.valid.conception");
}

test "language check fixture: phase3 unknown function return type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/unknown_function_return_type.invalid.conception");
}

test "language check fixture: phase3 unknown parameter type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/unknown_parameter_type.invalid.conception");
}

test "language check fixture: phase3 unknown struct field type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/unknown_struct_field_type.invalid.conception");
}

test "language check fixture: phase3 unknown enum payload type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/unknown_enum_payload_type.invalid.conception");
}

test "language check fixture: phase3 duplicate struct field" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/duplicate_struct_field.invalid.conception");
}

test "language check fixture: phase3 duplicate enum variant" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/duplicate_enum_variant.invalid.conception");
}

test "language check fixture: phase3 function name not type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/function_name_not_type.invalid.conception");
}

test "language check fixture: phase3 unsupported reference type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/unsupported_pointer_type.invalid.conception");
}

test "language check fixture: phase3 HIR body local return" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_body_local_return.valid.conception");
}

test "language check fixture: phase3 HIR body function call" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_body_function_call.valid.conception");
}

test "language check fixture: phase3 HIR body if match while" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_body_if_match_while.valid.conception");
}

test "language check fixture: phase3 HIR body unknown identifier" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_body_unknown_identifier.invalid.conception");
}

test "language check fixture: phase3 HIR body unknown function" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_body_unknown_function.invalid.conception");
}

test "language check fixture: phase3 HIR body duplicate local" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_body_duplicate_local.invalid.conception");
}

test "language check fixture: phase3 HIR body inner scope leak" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_body_inner_scope_leak.invalid.conception");
}

test "language check fixture: phase3 closeout struct enum type graph" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/phase3_struct_enum_type_graph.valid.conception");
}

test "language check fixture: phase3 closeout HIR control flow" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/phase3_hir_control_flow.valid.conception");
}

test "language check fixture: phase3 closeout HIR function graph" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/phase3_hir_function_graph.valid.conception");
}

test "language check fixture: phase3 closeout duplicate top level" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/phase3_duplicate_top_level.invalid.conception");
}

test "language check fixture: phase3 closeout unknown type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/phase3_unknown_type.invalid.conception");
}

test "language check fixture: phase3 closeout duplicate field" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/phase3_duplicate_field.invalid.conception");
}

test "language check fixture: phase3 closeout duplicate variant" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/phase3_duplicate_variant.invalid.conception");
}

test "language check fixture: phase3 closeout HIR unknown identifier" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/phase3_hir_unknown_identifier.invalid.conception");
}

test "language check fixture: phase3 closeout HIR call type mismatch" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/phase3_hir_call_type_mismatch.invalid.conception");
}

test "language check fixture: phase3 HIR checker return int" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_check_return_int.valid.conception");
}

test "language check fixture: phase3 HIR checker locals assignments" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_check_locals_assignments.valid.conception");
}

test "language check fixture: phase3 HIR checker function calls" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_check_function_calls.valid.conception");
}

test "language check fixture: phase3 HIR checker if match while" {
    try expectCheckFixture("../../../language/phase3-semantics/valid/hir_check_if_match_while.valid.conception");
}

test "language check fixture: phase3 HIR checker missing main" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_missing_main.invalid.conception");
}

test "language check fixture: phase3 HIR checker bad main signature" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_bad_main_signature.invalid.conception");
}

test "language check fixture: phase3 HIR checker return mismatch" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_return_type_mismatch.invalid.conception");
}

test "language check fixture: phase3 HIR checker local initializer mismatch" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_local_initializer_type_mismatch.invalid.conception");
}

test "language check fixture: phase3 HIR checker assignment mismatch" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_assignment_type_mismatch.invalid.conception");
}

test "language check fixture: phase3 HIR checker if condition" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_if_condition_not_bool.invalid.conception");
}

test "language check fixture: phase3 HIR checker while condition" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_while_condition_not_bool.invalid.conception");
}

test "language check fixture: phase3 HIR checker call arg count" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_call_arg_count.invalid.conception");
}

test "language check fixture: phase3 HIR checker call arg type" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_call_arg_type.invalid.conception");
}

test "language check fixture: phase3 HIR checker match pattern mismatch" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_match_pattern_type_mismatch.invalid.conception");
}

test "language check fixture: phase3 HIR checker match duplicate pattern" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_match_duplicate_pattern.invalid.conception");
}

test "language check fixture: phase3 HIR checker match duplicate wildcard" {
    try expectCheckFixture("../../../language/phase3-semantics/invalid/hir_check_match_duplicate_wildcard.invalid.conception");
}

fn expectMirFixture(comptime path: []const u8) !void {
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.mir, fixture.phase);
    try std.testing.expectEqual(Expectation.pass, fixture.expect);

    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, try fixture.singleSourceText());
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics_model.collectTopLevelDeclarationsWithOptions(
        std.testing.allocator,
        unit,
        &semantic_diagnostics,
        .{ .source_file_kind = source_file.kind },
    );
    defer module.deinit();
    try hir_checker_model.checkExecutable(std.testing.allocator, &module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    var mir_module = try mir_lowering_model.lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    try mir_validator_model.validateModule(std.testing.allocator, &module, &mir_module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expectEqualStrings(fixture.mir().?, snapshot);
}

fn expectBackendCFixture(comptime path: []const u8) !void {
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.backend_c, fixture.phase);

    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, try fixture.singleSourceText());
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics_model.collectTopLevelDeclarationsWithOptions(
        std.testing.allocator,
        unit,
        &diagnostics,
        .{ .source_file_kind = source_file.kind },
    );
    defer module.deinit();
    try hir_checker_model.checkExecutable(std.testing.allocator, &module, &diagnostics);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());

    var mir_module = try mir_lowering_model.lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    switch (fixture.expect) {
        .pass => {
            const c_source = try backend_c_mir_model.emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostics);
            defer std.testing.allocator.free(c_source);
            try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
            if (fixture.c()) |expected_fragments| {
                var lines = std.mem.splitScalar(u8, expected_fragments, '\n');
                while (lines.next()) |line| {
                    const trimmed = std.mem.trim(u8, line, " \t\r");
                    if (trimmed.len == 0) continue;
                    if (trimmed[0] == '#') continue;
                    if (std.mem.startsWith(u8, trimmed, "not_contains:")) {
                        const needle = std.mem.trim(u8, trimmed["not_contains:".len..], " \t");
                        try std.testing.expect(std.mem.indexOf(u8, c_source, needle) == null);
                    } else if (std.mem.startsWith(u8, trimmed, "count:")) {
                        const rest = std.mem.trim(u8, trimmed["count:".len..], " \t");
                        const separator = std.mem.indexOfScalar(u8, rest, '|') orelse return error.InvalidFixture;
                        const needle = std.mem.trim(u8, rest[0..separator], " \t");
                        const expected_count = try std.fmt.parseInt(usize, std.mem.trim(u8, rest[separator + 1 ..], " \t"), 10);
                        var actual_count: usize = 0;
                        var search_start: usize = 0;
                        while (std.mem.indexOf(u8, c_source[search_start..], needle)) |relative_index| {
                            actual_count += 1;
                            search_start += relative_index + needle.len;
                        }
                        try std.testing.expectEqual(expected_count, actual_count);
                    } else if (std.mem.startsWith(u8, trimmed, "before:")) {
                        const rest = std.mem.trim(u8, trimmed["before:".len..], " \t");
                        const separator = std.mem.indexOfScalar(u8, rest, '|') orelse return error.InvalidFixture;
                        const first = std.mem.trim(u8, rest[0..separator], " \t");
                        const second = std.mem.trim(u8, rest[separator + 1 ..], " \t");
                        const first_index = std.mem.indexOf(u8, c_source, first) orelse return error.TestExpectedEqual;
                        const second_index = std.mem.indexOf(u8, c_source, second) orelse return error.TestExpectedEqual;
                        try std.testing.expect(first_index < second_index);
                    } else {
                        const needle = if (std.mem.startsWith(u8, trimmed, "contains:"))
                            std.mem.trim(u8, trimmed["contains:".len..], " \t")
                        else
                            trimmed;
                        try std.testing.expect(std.mem.indexOf(u8, c_source, needle) != null);
                    }
                }
            }
        },
        .fail => {
            try std.testing.expectError(error.InvalidExecutable, backend_c_mir_model.emitExecutableFromMir(std.testing.allocator, &module, &mir_module, &diagnostics));
            const expected_codes = try fixture.diagnosticCodes(std.testing.allocator);
            defer std.testing.allocator.free(expected_codes);
            try std.testing.expectEqual(expected_codes.len, diagnostics.count());
            for (expected_codes, diagnostics.diagnostics.items) |expected_code, actual| {
                try std.testing.expectEqualStrings(expected_code, actual.code.format());
            }
        },
    }
}

fn expectMirCorpus(comptime source_path: []const u8, comptime expected_path: []const u8) !void {
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, source_path, @embedFile(source_path));
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics_model.collectTopLevelDeclarationsWithOptions(
        std.testing.allocator,
        unit,
        &semantic_diagnostics,
        .{ .source_file_kind = source_file.kind },
    );
    defer module.deinit();
    try hir_checker_model.checkExecutable(std.testing.allocator, &module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    var mir_module = try mir_lowering_model.lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    try mir_validator_model.validateModule(std.testing.allocator, &module, &mir_module, &semantic_diagnostics);
    try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expectEqualStrings(@embedFile(expected_path), snapshot);
}

test "MIR corpus snapshot: phase4 return literal" {
    try expectMirCorpus("../../../tests/corpus/phase4/mir_return_literal.concept", "../../../tests/corpus/phase4/mir_return_literal.mir.expected");
}

test "MIR corpus snapshot: phase4 sum loop" {
    try expectMirCorpus("../../../tests/corpus/phase4/mir_sum_loop.concept", "../../../tests/corpus/phase4/mir_sum_loop.mir.expected");
}

test "MIR corpus snapshot: phase4 if match" {
    try expectMirCorpus("../../../tests/corpus/phase4/mir_if_match.concept", "../../../tests/corpus/phase4/mir_if_match.mir.expected");
}

test "MIR corpus snapshot: phase5 enum constructor" {
    try expectMirCorpus("../../../tests/corpus/phase5/enum_constructor.concept", "../../../tests/corpus/phase5/enum_constructor.mir.expected");
}

test "MIR corpus snapshot: phase5 enum match payload" {
    try expectMirCorpus("../../../tests/corpus/phase5/enum_match_payload.concept", "../../../tests/corpus/phase5/enum_match_payload.mir.expected");
}

test "MIR corpus snapshot: phase5 result try success" {
    try expectMirCorpus("../../../tests/corpus/phase5/result_try_success.concept", "../../../tests/corpus/phase5/result_try_success.mir.expected");
}

test "MIR corpus snapshot: phase5 result try failure" {
    try expectMirCorpus("../../../tests/corpus/phase5/result_try_failure.concept", "../../../tests/corpus/phase5/result_try_failure.mir.expected");
}

test "language MIR fixture: phase4 return literal" {
    try expectMirFixture("../../../language/phase4-mir/valid/mir_return_literal.valid.conception");
}

test "language MIR fixture: phase4 sum loop" {
    try expectMirFixture("../../../language/phase4-mir/valid/mir_sum_loop.valid.conception");
}

test "language MIR fixture: phase4 if match" {
    try expectMirFixture("../../../language/phase4-mir/valid/mir_if_match.valid.conception");
}

test "language MIR fixture: phase7 struct places snapshot" {
    try expectMirFixture("../../../language/phase7-runtime-structs/valid/struct_places_mir_snapshot.valid.conception");
}

fn expectRunFixture(comptime path: []const u8) !void {
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.run, fixture.phase);
    switch (fixture.expect) {
        .pass => {
            if (fixture.isMultiSource()) {
                const inputs = try std.testing.allocator.alloc(run_harness.SourceInput, fixture.sources.len);
                defer std.testing.allocator.free(inputs);
                for (fixture.sources, 0..) |fixture_source, index| {
                    inputs[index] = .{ .path = fixture_source.path, .text = fixture_source.text };
                }
                _ = try run_harness.expectExitCodeMulti(std.testing.allocator, inputs, try fixture.expectedExitCode());
            } else {
                _ = try run_harness.expectExitCode(std.testing.allocator, try fixture.singleSourceText(), try fixture.expectedExitCode());
            }
        },
        .fail => try run_harness.expectRuntimeFailure(std.testing.allocator, try fixture.singleSourceText()),
    }
}

test "language run fixture: phase3 HIR sum loop" {
    try expectRunFixture("../../../language/phase3-semantics/valid/phase3_hir_sum_loop_run.valid.conception");
}

test "language run fixture: phase2 return zero" {
    try expectRunFixture("../../../language/phase2-execution/valid/return_zero_run.valid.conception");
}

test "language run fixture: phase2 arithmetic return" {
    try expectRunFixture("../../../language/phase2-execution/valid/arithmetic_return_run.valid.conception");
}

test "language run fixture: phase2 bool return" {
    try expectRunFixture("../../../language/phase2-execution/valid/bool_return_run.valid.conception");
}

test "language run fixture: phase2 local int return" {
    try expectRunFixture("../../../language/phase2-execution/valid/local_int_return.valid.conception");
}

test "language run fixture: phase2 local arithmetic return" {
    try expectRunFixture("../../../language/phase2-execution/valid/local_arithmetic_return.valid.conception");
}

test "language run fixture: phase2 local bool return" {
    try expectRunFixture("../../../language/phase2-execution/valid/local_bool_return.valid.conception");
}

test "language run fixture: phase2 function call add" {
    try expectRunFixture("../../../language/phase2-execution/valid/function_call_add.valid.conception");
}

test "language run fixture: phase2 no-argument function call" {
    try expectRunFixture("../../../language/phase2-execution/valid/function_call_no_args.valid.conception");
}

test "language run fixture: phase2 function call with local" {
    try expectRunFixture("../../../language/phase2-execution/valid/function_call_with_local.valid.conception");
}

test "language run fixture: phase2 forward function call" {
    try expectRunFixture("../../../language/phase2-execution/valid/forward_function_call.valid.conception");
}

test "language run fixture: phase2 if true return" {
    try expectRunFixture("../../../language/phase2-execution/valid/if_true_return.valid.conception");
}

test "language run fixture: phase2 if false else return" {
    try expectRunFixture("../../../language/phase2-execution/valid/if_false_else_return.valid.conception");
}

test "language run fixture: phase2 if else return" {
    try expectRunFixture("../../../language/phase2-execution/valid/if_else_return.valid.conception");
}

test "language run fixture: phase2 if compare function" {
    try expectRunFixture("../../../language/phase2-execution/valid/if_compare_function.valid.conception");
}

test "language run fixture: phase2 if with outer local" {
    try expectRunFixture("../../../language/phase2-execution/valid/if_with_outer_local.valid.conception");
}

test "language run fixture: phase2 if outer local" {
    try expectRunFixture("../../../language/phase2-execution/valid/if_outer_local.valid.conception");
}

test "language parse fixture: phase2 else if ladder rejected" {
    try expectParseFixture("../../../language/phase2-execution/invalid/else_if_ladder.invalid.conception");
}

test "language check fixture: phase2 if condition not bool" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/if_condition_not_bool.invalid.conception");
}

test "language check fixture: phase2 if branch local not visible" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/if_branch_local_not_visible.invalid.conception");
}

test "language parse fixture: phase2 match missing open brace" {
    try expectParseFixture("../../../language/phase2-execution/invalid/match_missing_open_brace.invalid.conception");
}

test "language parse fixture: phase2 match malformed pattern" {
    try expectParseFixture("../../../language/phase2-execution/invalid/match_malformed_pattern.invalid.conception");
}

test "language parse fixture: phase2 match missing fat arrow" {
    try expectParseFixture("../../../language/phase2-execution/invalid/match_missing_fat_arrow.invalid.conception");
}

test "language check fixture: phase2 match pattern type mismatch" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/match_pattern_type_mismatch.invalid.conception");
}

test "language check fixture: phase2 match duplicate pattern" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/match_duplicate_pattern.invalid.conception");
}

test "language check fixture: phase2 match duplicate wildcard" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/match_duplicate_wildcard.invalid.conception");
}

test "language run fixture: phase2 assignment local int" {
    try expectRunFixture("../../../language/phase2-execution/valid/assignment_local_int.valid.conception");
}

test "language run fixture: phase2 assignment local bool" {
    try expectRunFixture("../../../language/phase2-execution/valid/assignment_local_bool.valid.conception");
}

test "language run fixture: phase2 assignment parameter" {
    try expectRunFixture("../../../language/phase2-execution/valid/assignment_parameter.valid.conception");
}

test "language run fixture: phase2 assignment in if" {
    try expectRunFixture("../../../language/phase2-execution/valid/assignment_in_if.valid.conception");
}

test "language run fixture: phase2 assignment in match" {
    try expectRunFixture("../../../language/phase2-execution/valid/assignment_in_match.valid.conception");
}

test "language check fixture: phase2 assignment unknown identifier" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/assignment_unknown_identifier.invalid.conception");
}

test "language check fixture: phase2 assignment type mismatch" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/assignment_type_mismatch.invalid.conception");
}

test "language check fixture: phase2 assignment inner scope local" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/assignment_inner_scope_local.invalid.conception");
}

test "language parse fixture: phase2 assignment missing expression" {
    try expectParseFixture("../../../language/phase2-execution/invalid/assignment_missing_expression.invalid.conception");
}

test "language parse fixture: phase2 assignment missing semicolon" {
    try expectParseFixture("../../../language/phase2-execution/invalid/assignment_missing_semicolon.invalid.conception");
}

test "language run fixture: phase2 match int return" {
    try expectRunFixture("../../../language/phase2-execution/valid/match_int_return.valid.conception");
}

test "language run fixture: phase2 match bool return" {
    try expectRunFixture("../../../language/phase2-execution/valid/match_bool_return.valid.conception");
}

test "language run fixture: phase2 match default return" {
    try expectRunFixture("../../../language/phase2-execution/valid/match_default_return.valid.conception");
}

test "language run fixture: phase2 nested if explicit else block" {
    try expectRunFixture("../../../language/phase2-execution/valid/nested_if_explicit_else_block.valid.conception");
}

test "language run fixture: phase2 explicit nested if in else" {
    try expectRunFixture("../../../language/phase2-execution/valid/explicit_nested_if_in_else.valid.conception");
}

test "language run fixture: phase2 while count to seven" {
    try expectRunFixture("../../../language/phase2-execution/valid/while_count_to_seven.valid.conception");
}

test "language run fixture: phase2 while zero iterations" {
    try expectRunFixture("../../../language/phase2-execution/valid/while_zero_iterations.valid.conception");
}

test "language run fixture: phase2 while with if" {
    try expectRunFixture("../../../language/phase2-execution/valid/while_with_if.valid.conception");
}

test "language run fixture: phase2 nested while" {
    try expectRunFixture("../../../language/phase2-execution/valid/nested_while.valid.conception");
}

test "language check fixture: phase2 while condition not bool" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/while_condition_not_bool.invalid.conception");
}

test "language check fixture: phase2 while inner local not visible" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/while_inner_local_not_visible.invalid.conception");
}

test "language parse fixture: phase2 while missing condition" {
    try expectParseFixture("../../../language/phase2-execution/invalid/while_missing_condition.invalid.conception");
}

test "language parse fixture: phase2 while missing body" {
    try expectParseFixture("../../../language/phase2-execution/invalid/while_missing_body.invalid.conception");
}

test "language run fixture: phase2 closeout sum loop" {
    try expectRunFixture("../../../language/phase2-execution/valid/phase2_sum_loop.valid.conception");
}

test "language run fixture: phase2 closeout function call loop" {
    try expectRunFixture("../../../language/phase2-execution/valid/phase2_function_call_loop.valid.conception");
}

test "language run fixture: phase2 closeout if match mix" {
    try expectRunFixture("../../../language/phase2-execution/valid/phase2_if_match_mix.valid.conception");
}

test "language run fixture: phase2 closeout nested control flow" {
    try expectRunFixture("../../../language/phase2-execution/valid/phase2_nested_control_flow.valid.conception");
}

test "language parse fixture: phase2 closeout else if rejected" {
    try expectParseFixture("../../../language/phase2-execution/invalid/phase2_else_if_rejected.invalid.conception");
}

test "language check fixture: phase2 closeout unknown identifier" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/phase2_unknown_identifier.invalid.conception");
}

test "language check fixture: phase2 closeout bad call" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/phase2_bad_call.invalid.conception");
}

test "language check fixture: phase2 closeout bad while condition" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/phase2_bad_while_condition.invalid.conception");
}

test "language check fixture: phase2 closeout match duplicate pattern" {
    try expectCheckFixture("../../../language/phase2-execution/invalid/phase2_match_duplicate_pattern.invalid.conception");
}

test "language run fixture: phase5 enum constructor tag only run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_constructor_tag_only_run.valid.conception");
}

test "language run fixture: phase5 enum constructor payload run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_constructor_payload_run.valid.conception");
}

test "language run fixture: phase5 enum match tag only" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_tag_only.valid.conception");
}

test "language run fixture: phase5 enum match payload without binding" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_payload_no_binding.valid.conception");
}

test "language run fixture: phase5 enum match wildcard" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_wildcard.valid.conception");
}

test "language run fixture: phase5 enum match function return" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_function_return.valid.conception");
}

test "language check fixture: phase5 enum constructor unknown enum" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_constructor_unknown_enum.invalid.conception");
}

test "language check fixture: phase5 enum constructor unknown variant" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_constructor_unknown_variant.invalid.conception");
}

test "language check fixture: phase5 enum constructor arity mismatch" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_constructor_arity_mismatch.invalid.conception");
}

test "language check fixture: phase5 enum constructor payload type mismatch" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_constructor_payload_type_mismatch.invalid.conception");
}

test "language check fixture: phase5 enum match unknown variant" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_unknown_variant.invalid.conception");
}

test "language check fixture: phase5 enum match duplicate variant" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_duplicate_variant.invalid.conception");
}

test "language check fixture: phase5 enum match enum mismatch" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_enum_mismatch.invalid.conception");
}

test "language check fixture: phase5 enum match int pattern" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_int_pattern.invalid.conception");
}

test "language check fixture: phase5 enum match variant against int" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_variant_against_int.invalid.conception");
}

test "language run fixture: phase5 enum match payload bind int" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_payload_bind_int.valid.conception");
}

test "language run fixture: phase5 enum match payload bind bool" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_payload_bind_bool.valid.conception");
}

test "language run fixture: phase5 enum match payload bind two" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/enum_match_payload_bind_two.valid.conception");
}

test "language check fixture: phase5 enum match payload too few bindings" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_too_few_bindings.invalid.conception");
}

test "language check fixture: phase5 enum match payload too many bindings" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_too_many_bindings.invalid.conception");
}

test "language check fixture: phase5 enum match payload duplicate binding" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_duplicate_binding.invalid.conception");
}

test "language check fixture: phase5 enum match payload binding shadow" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_binding_shadow.invalid.conception");
}

test "language check fixture: phase5 enum match payload binding leak" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_binding_leak.invalid.conception");
}

test "language check fixture: phase5 enum match payload zero variant binding" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_zero_variant_binding.invalid.conception");
}

test "language parse fixture: phase5 enum match payload expression binding" {
    try expectParseFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_expression_binding.invalid.conception");
}

test "language parse fixture: phase5 enum match payload named binding" {
    try expectParseFixture("../../../language/phase5-sum-types/invalid/enum_match_payload_named_binding.invalid.conception");
}

test "language check fixture: phase5 enum match unknown enum" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/enum_match_unknown_enum.invalid.conception");
}

test "language run fixture: phase5 must_use enum local init" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/must_use_enum_local_init.valid.conception");
}

test "language run fixture: phase5 discard must_use constructor" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/discard_must_use_constructor.valid.conception");
}

test "language run fixture: phase5 discard must_use call" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/discard_must_use_call.valid.conception");
}

test "language check fixture: phase5 ignored must_use constructor" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/ignored_must_use_constructor.invalid.conception");
}

test "language check fixture: phase5 ignored must_use call" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/ignored_must_use_call.invalid.conception");
}

test "language check fixture: phase5 result shape must_use" {
    try expectCheckFixture("../../../language/phase5-sum-types/valid/result_shape_must_use.valid.conception");
}

test "language check fixture: phase5 result shape not must_use" {
    try expectCheckFixture("../../../language/phase5-sum-types/valid/result_shape_not_must_use.valid.conception");
}

test "language check fixture: phase5 result shape payload types" {
    try expectCheckFixture("../../../language/phase5-sum-types/valid/result_shape_payload_types.valid.conception");
}

test "language check fixture: phase5 not result shape extra variant" {
    try expectCheckFixture("../../../language/phase5-sum-types/valid/not_result_shape_extra_variant.valid.conception");
}

test "language check fixture: phase5 not result shape bad ok arity" {
    try expectCheckFixture("../../../language/phase5-sum-types/valid/not_result_shape_bad_ok_arity.valid.conception");
}

test "language run fixture: phase5 try success add" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/try_success_add.valid.conception");
}

test "language run fixture: phase5 try failure propagates" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/try_failure_propagates.valid.conception");
}

test "language run fixture: phase5 try inside binary" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/try_inside_binary.valid.conception");
}

test "language run fixture: phase5 result try failure run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/result_try_failure_run.valid.conception");
}

test "language run fixture: phase5 result try two success run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/result_try_two_success_run.valid.conception");
}

test "language run fixture: phase5 result try binary lhs run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/result_try_binary_lhs_run.valid.conception");
}

test "language run fixture: phase5 result try constructor argument run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/result_try_constructor_argument_run.valid.conception");
}

test "language run fixture: phase5 result try call argument run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/result_try_call_argument_run.valid.conception");
}

test "language run fixture: phase5 must use match result run" {
    try expectRunFixture("../../../language/phase5-sum-types/valid/must_use_match_result_run.valid.conception");
}

test "language check fixture: phase5 try operand not result" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/try_operand_not_result.invalid.conception");
}

test "language check fixture: phase5 try in int function" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/try_in_int_function.invalid.conception");
}

test "language check fixture: phase5 try result mismatch" {
    try expectCheckFixture("../../../language/phase5-sum-types/invalid/try_result_mismatch.invalid.conception");
}

test "language check fixture: phase5a decide basic" {
    try expectCheckFixture("../../../language/phase5a-judgment/valid/decide_basic.valid.conception");
}

test "language check fixture: phase5a decide duplicate variant" {
    try expectCheckFixture("../../../language/phase5a-judgment/valid/decide_duplicate_variant.valid.conception");
}

test "language check fixture: phase5a decide negative score" {
    try expectCheckFixture("../../../language/phase5a-judgment/valid/decide_negative_score.valid.conception");
}

test "language run fixture: phase5a decide highest score" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_highest_score_run.valid.conception");
}

test "language run fixture: phase5a decide fallback" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_fallback_run.valid.conception");
}

test "language run fixture: phase5a decide tie break first" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_tie_break_first_run.valid.conception");
}

test "language run fixture: phase5a decide duplicate variant" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_duplicate_variant_run.valid.conception");
}

test "language run fixture: phase5a decide negative score" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_negative_score_run.valid.conception");
}

test "language run fixture: phase5a decide condition gates score" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_condition_gates_score_run.valid.conception");
}

test "language run fixture: phase5a decide return expression" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_return_expression_run.valid.conception");
}

test "language run fixture: phase5a decide call argument" {
    try expectRunFixture("../../../language/phase5a-judgment/valid/decide_call_argument_run.valid.conception");
}

test "language check fixture: phase5a decide unknown enum" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_unknown_enum.invalid.conception");
}

test "language check fixture: phase5a decide target not enum" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_target_not_enum.invalid.conception");
}

test "language check fixture: phase5a decide unknown variant" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_unknown_variant.invalid.conception");
}

test "language check fixture: phase5a decide payload variant" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_payload_variant.invalid.conception");
}

test "language check fixture: phase5a decide condition not bool" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_condition_not_bool.invalid.conception");
}

test "language check fixture: phase5a decide score not int" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_score_not_int.invalid.conception");
}

test "language check fixture: phase5a decide missing unconditional" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_missing_unconditional.invalid.conception");
}

test "language check fixture: phase5a decide empty arms" {
    try expectCheckFixture("../../../language/phase5a-judgment/invalid/decide_empty_arms.invalid.conception");
}

test "language run fixture: phase6 unsafe block return" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/unsafe_block_return.valid.conception");
}

test "language run fixture: phase6 unsafe block normal statements" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/unsafe_block_normal_statements.valid.conception");
}

test "language run fixture: phase6 unsafe function call inside unsafe" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/unsafe_function_call_inside_unsafe.valid.conception");
}

test "language check fixture: phase6 unsafe function call outside unsafe" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/unsafe_function_call_outside_unsafe.invalid.conception");
}

test "language check fixture: phase6 pointer identity" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/valid/pointer_identity.valid.conception");
}

test "language check fixture: phase6 pointer local copy" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/valid/pointer_local_copy.valid.conception");
}

test "language check fixture: phase6 pointer call argument" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/valid/pointer_call_argument.valid.conception");
}

test "language check fixture: phase6 pointer enum type" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/valid/pointer_enum_type.valid.conception");
}

test "language check fixture: phase6 pointer return mismatch" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/pointer_return_mismatch.invalid.conception");
}

test "language check fixture: phase6 pointer local mismatch" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/pointer_local_mismatch.invalid.conception");
}

test "language check fixture: phase6 pointer call mismatch" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/pointer_call_mismatch.invalid.conception");
}

test "language run fixture: phase6 pointer address deref" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/pointer_address_deref_run.valid.conception");
}

test "language run fixture: phase6 pointer deref in unsafe function" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/pointer_deref_in_unsafe_function.valid.conception");
}

test "language run fixture: phase6 pointer address of param" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/pointer_address_of_param.valid.conception");
}

test "language run fixture: phase6 pointer unsafe call deref interaction" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/pointer_unsafe_call_deref_interaction.valid.conception");
}

test "language run fixture: phase6 pointer local copy then deref" {
    try expectRunFixture("../../../language/phase6-unsafe-ownership/valid/pointer_local_copy_then_deref.valid.conception");
}

test "language check fixture: phase6 pointer deref outside unsafe" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/pointer_deref_outside_unsafe.invalid.conception");
}

test "language check fixture: phase6 pointer deref non pointer" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/pointer_deref_non_pointer.invalid.conception");
}

test "language check fixture: phase6 pointer address of temporary" {
    try expectCheckFixture("../../../language/phase6-unsafe-ownership/invalid/pointer_address_of_temporary.invalid.conception");
}

test "language check fixture: phase7 struct literal basic" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/struct_literal_basic.valid.conception");
}

test "language check fixture: phase7 struct literal reordered" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/struct_literal_reordered.valid.conception");
}

test "language run fixture: phase7 struct literal run" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/struct_literal_run.valid.conception");
}

test "language check fixture: phase7 struct literal unknown type" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_literal_unknown_type.invalid.conception");
}

test "language check fixture: phase7 struct literal non struct type" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_literal_non_struct_type.invalid.conception");
}

test "language check fixture: phase7 struct literal unknown field" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_literal_unknown_field.invalid.conception");
}

test "language check fixture: phase7 struct literal duplicate field" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_literal_duplicate_field.invalid.conception");
}

test "language check fixture: phase7 struct literal missing field" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_literal_missing_field.invalid.conception");
}

test "language check fixture: phase7 struct literal type mismatch" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_literal_type_mismatch.invalid.conception");
}

test "language run fixture: phase7 field access run" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/field_access_run.valid.conception");
}

test "language check fixture: phase7 field access multiple" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/field_access_multiple.valid.conception");
}

test "language check fixture: phase7 field access bool enum" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/field_access_bool_enum.valid.conception");
}

test "language check fixture: phase7 field access non struct" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_access_non_struct.invalid.conception");
}

test "language check fixture: phase7 field access pointer" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_access_pointer.invalid.conception");
}

test "language check fixture: phase7 field access unknown field" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_access_unknown_field.invalid.conception");
}

test "language check fixture: phase7 field assignment basic" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/field_assignment_basic.valid.conception");
}

test "language run fixture: phase7 field assignment run" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/field_assignment_run.valid.conception");
}

test "language check fixture: phase7 field assignment bool" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/field_assignment_bool.valid.conception");
}

test "language check fixture: phase7 field assignment enum" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/field_assignment_enum.valid.conception");
}

test "language check fixture: phase7 field assignment type mismatch" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_assignment_type_mismatch.invalid.conception");
}

test "language check fixture: phase7 field assignment unknown field" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_assignment_unknown_field.invalid.conception");
}

test "language check fixture: phase7 field assignment non struct" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_assignment_non_struct.invalid.conception");
}

test "language check fixture: phase7 field assignment non place" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_assignment_non_place.invalid.conception");
}

test "language run fixture: phase7 field address-of run" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/field_address_of_run.valid.conception");
}

test "language check fixture: phase7 field address-of param" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/field_address_of_param.valid.conception");
}

test "language run fixture: phase7 field address-of after assignment" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/field_address_of_after_assignment.valid.conception");
}

test "language check fixture: phase7 field address-of temporary" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_address_of_temporary.invalid.conception");
}

test "language check fixture: phase7 field address-of unknown" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_address_of_unknown.invalid.conception");
}

test "language check fixture: phase7 field address-of non struct" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_address_of_non_struct.invalid.conception");
}

test "language check fixture: phase7 field address-of pointer receiver" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/field_address_of_pointer_receiver.invalid.conception");
}

test "language check fixture: phase7 struct return literal" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/struct_return_literal.valid.conception");
}

test "language check fixture: phase7 struct param call" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/struct_param_call.valid.conception");
}

test "language check fixture: phase7 struct local from call" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/struct_local_from_call.valid.conception");
}

test "language check fixture: phase7 struct return local" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/valid/struct_return_local.valid.conception");
}

test "language check fixture: phase7 struct return mismatch" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_return_mismatch.invalid.conception");
}

test "language check fixture: phase7 struct call arg mismatch" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_call_arg_mismatch.invalid.conception");
}

test "language check fixture: phase7 struct local initializer mismatch" {
    try expectCheckFixture("../../../language/phase7-runtime-structs/invalid/struct_local_initializer_mismatch.invalid.conception");
}

test "language run fixture: phase7 struct return and call" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/struct_return_and_call_run.valid.conception");
}

test "language run fixture: phase7 struct return local" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/struct_return_local_run.valid.conception");
}

test "language run fixture: phase7 struct param field assignment" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/struct_param_field_assignment_run.valid.conception");
}

test "language run fixture: phase7 struct pipeline closeout" {
    try expectRunFixture("../../../language/phase7-runtime-structs/valid/phase7_struct_pipeline_closeout_run.valid.conception");

    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_int_arithmetic_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_bool_logic_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_comparison_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_return_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_grouping_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_zero_arg_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_args_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_bool_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_nested_function_call.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_local_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_bool_local_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_if_then_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_if_else_run.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_function_abs_static_assert.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_nested_if.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_local_assignment_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_while_sum_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_while_count_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_while_return.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_compile_time_while.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_function_while_false_fallthrough.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_true.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_arithmetic.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_bool_logic.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_compile_time_function.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/static_assert_with_main_run.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_capability_declaration_valid.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_multiple_capabilities_declaration_valid.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_plain_function_still_works.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_target_pointer_size_run.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_target_endian_bool_run.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_target_pointer_size.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/static_assert_target_endian.valid.conception");
    try expectRunFixture("../../../language/phase9-compile-time/valid/compile_time_target_in_if.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_logical_or_short_circuit_runtime_call.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_logical_and_short_circuit_runtime_call.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_logical_or_short_circuit_capability_denied.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/valid/compile_time_logical_and_short_circuit_capability_denied.valid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_call_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_unknown_capability.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_duplicate_capability.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_capability_not_granted_expr.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_capability_not_granted_static_assert.invalid.conception");
    try expectParseFixture("../../../language/phase9-compile-time/invalid/compile_time_empty_capability_list.invalid.conception");
    try expectParseFixture("../../../language/phase9-compile-time/invalid/compile_time_capability_on_non_function.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_call_runtime_function.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_local_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_struct_literal_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_field_access_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_target_unknown_field.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_target_runtime_use.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_target_string_field_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_division_by_zero.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_overflow.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_runtime_call.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_missing_return.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_if_non_bool.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_while_condition_non_bool.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_while_fuel_exhausted.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_while_missing_return.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_while_runtime_call.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_while_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_local_struct_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_if_unsupported_executed_branch.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_runtime_call_in_branch.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_unsupported_signature_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_function_recursion.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_logical_or_evaluates_rhs_runtime_call.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/compile_time_logical_and_evaluates_rhs_runtime_call.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/static_assert_false.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/static_assert_requires_bool.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/static_assert_call_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/static_assert_compile_time_function_false.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/static_assert_division_by_zero.invalid.conception");
    try expectCheckFixture("../../../language/phase9-compile-time/invalid/static_assert_overflow.invalid.conception");

    try expectParseFixture("../../../language/phase11-testing/valid/attribute_fact_function.valid.conception");
    try expectParseFixture("../../../language/phase11-testing/valid/attribute_theory_inline_data.valid.conception");
    try expectParseFixture("../../../language/phase11-testing/valid/attribute_multiple_inline_data.valid.conception");
    try expectCheckFixture("../../../language/phase11-testing/valid/normal_file_without_test_attributes_ok.valid.conception");
    try expectCheckFixture("../../../language/phase11-testing/valid/fact_zero_arg_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_inline_data_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_multiple_inline_data_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_int_bool_inline_data_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_expect_equal_int_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_expect_equal_bool_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_multiple_rows_all_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_with_helper_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/assert_true_reason_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/assert_false_reason_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/assert_true_reason_alignment.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/assert_false_reason_alignment.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/assert_runtime_and_test_alignment.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_true_reason_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_false_reason_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_equal_int_reason_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_equal_bool_reason_valid.valid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_that_true_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_that_false_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_that_equal_int_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/expect_that_equal_bool_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_expect_that_equal_int_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/theory_expect_that_true_pass.con_test");
    try expectCheckFixture("../../../language/phase11-testing/valid/phase11_closeout_smoke.con_test");
    try expectParseFixture("../../../language/phase11-testing/invalid/attribute_unknown.invalid.conception");
    try expectParseFixture("../../../language/phase11-testing/invalid/attribute_orphan.invalid.conception");
    try expectParseFixture("../../../language/phase11-testing/invalid/attribute_statement_position.invalid.conception");
    try expectParseFixture("../../../language/phase11-testing/invalid/attribute_arg_expression.invalid.conception");
    try expectCheckFixture("../../../language/phase11-testing/invalid/fact_in_normal_source.invalid.conception");
    try expectCheckFixture("../../../language/phase11-testing/invalid/theory_in_normal_source.invalid.conception");
    try expectCheckFixture("../../../language/phase11-testing/invalid/fact_with_parameter_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/theory_missing_inline_data_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/inline_data_without_theory_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/fact_with_inline_data_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/fact_theory_conflict_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/duplicate_fact_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/duplicate_theory_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/inline_data_arity_mismatch_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/inline_data_type_mismatch_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/fact_wrong_return_type_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_true_missing_reason_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_true_missing_reason_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_equal_missing_reason_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_reason_empty_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_reason_whitespace_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_true_empty_reason.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_true_whitespace_reason.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_false_empty_reason.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_false_whitespace_reason.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_true_non_bool_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_true_non_bool.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/assert_false_non_bool_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_equal_type_mismatch_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_equal_unsupported_type_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_wrong_arity_invalid.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/test_intrinsic_normal_source_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_missing_reason.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_empty_reason.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_wrong_arity.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_true_non_bool.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_equal_type_mismatch.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_equal_unsupported_type.invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/test_relation_standalone_invalid.con_test");
    try expectCheckFixture("../../../language/phase11-testing/invalid/expect_that_normal_source_invalid.concept");
    try expectParseFixture("../../../language/phase11-testing/invalid/inline_test_block_rejected.invalid.conception");

    try expectParseFixture("../../../language/phase12-allocation/valid/function_noalloc.valid.conception");
    try expectParseFixture("../../../language/phase12-allocation/valid/function_alloc.valid.conception");
    try expectParseFixture("../../../language/phase12-allocation/valid/function_effect_unspecified.valid.conception");
    try expectParseFixture("../../../language/phase12-allocation/valid/unsafe_noalloc_function.valid.conception");
    try expectParseFixture("../../../language/phase12-allocation/valid/comptime_noalloc_function.valid.conception");
    try expectParseFixture("../../../language/phase12-allocation/valid/template_noalloc_function.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/noalloc_calls_noalloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/alloc_calls_alloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/alloc_calls_unspecified.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/unspecified_calls_alloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/noalloc_recursive_call.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/noalloc_generic_call.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/comptime_noalloc_calls_noalloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/unsafe_noalloc_effect_call.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_type_pointer_param.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/allocator_type_pointer_param.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/alloc_error_type_position.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/noalloc_uses_arena_pointer.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/alloc_uses_allocator_pointer.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/generic_function_with_arena_pointer.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_alloc_int.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_alloc_bool.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_alloc_struct.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_alloc_in_unspecified_function.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_alloc_generic.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_alloc_generic_non_drop.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_reset.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_destroy.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_reset_in_noalloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_destroy_in_noalloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_reset_after_alloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_destroy_after_alloc.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_reset_in_unspecified_function.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/arena_destroy_in_unspecified_function.valid.conception");
    try expectBackendCFixture("../../../language/phase12-allocation/valid/arena_helper_alloc_output.valid.conception");
    try expectBackendCFixture("../../../language/phase12-allocation/valid/arena_helper_reset_output.valid.conception");
    try expectBackendCFixture("../../../language/phase12-allocation/valid/arena_helper_destroy_output.valid.conception");
    try expectBackendCFixture("../../../language/phase12-allocation/valid/arena_helper_struct_size_align.valid.conception");
    try expectBackendCFixture("../../../language/phase12-allocation/valid/arena_helper_multiple_ops.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/typed_id_structs.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/expr_store_single_entry.valid.conception");
    try expectRunFixture("../../../language/phase12-allocation/valid/store_add_get_pattern.valid.conception");
    try expectRunFixture("../../../language/phase12-allocation/valid/store_id_copy_value.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/store_id_with_noalloc_functions.valid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/valid/phase12_closeout_smoke.valid.conception");
    try expectParseFixture("../../../language/phase12-allocation/invalid/function_conflicting_alloc_effects.invalid.conception");
    try expectParseFixture("../../../language/phase12-allocation/invalid/function_duplicate_noalloc.invalid.conception");
    try expectParseFixture("../../../language/phase12-allocation/invalid/function_duplicate_alloc.invalid.conception");
    try expectParseFixture("../../../language/phase12-allocation/invalid/allocation_effect_invalid_target.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/noalloc_calls_alloc.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/noalloc_calls_unspecified.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/noalloc_calls_alloc_generic.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/noalloc_calls_unspecified_generic.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/comptime_noalloc_calls_alloc.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_struct_literal.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/allocator_struct_literal.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_field_access.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/allocator_field_access.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_by_value_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/allocator_by_value_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_in_noalloc.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_requires_arena_pointer.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_missing_type_arg.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_wrong_arity.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_drop_type.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_generic_drop_type.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_struct_contains_drop.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_manualinit_drop.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_opaque_arena.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_opaque_allocator.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_opaque_arena_regression.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_opaque_allocator_regression.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_alloc_comptime.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_reset_requires_arena_pointer.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_destroy_requires_arena_pointer.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_reset_wrong_arity.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_destroy_wrong_arity.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_reset_type_args.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_destroy_type_args.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_reset_comptime.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/arena_destroy_comptime.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/store_id_type_mismatch.invalid.conception");
    try expectCheckFixture("../../../language/phase12-allocation/invalid/store_get_wrong_id_type.invalid.conception");

    try expectParseFixture("../../../language/phase13-machines/valid/machine_empty_state.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_single_state_valid.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_two_states.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_two_states_same_names_in_different_machines.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_with_return.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_with_params.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_noalloc_effect.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_with_attribute.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_state_ordinary_statements.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_literal.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_self.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_initial.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_later_state.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_in_nested_if.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_match_basic.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_match_self.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_match_initial.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_match_later_state.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_match_nested_if.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_basic.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_when_conditions.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_self.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_initial.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_later_state.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_nested_if.valid.conception");
    try expectParseFixture("../../../language/phase13-machines/valid/machine_transition_decide_zero_score_fallback.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_step_expression_statement.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_step_literal_two_steps.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_complete_result.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_extra_step_after_complete.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_param_return.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_bool_result.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_two_literal_transitions.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_complete_false_before_return.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/machine_step_completed_noop.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/valid/phase13_closeout_literal_machine.valid.conception");
    try expectRunFixture("../../../language/phase13-machines/invalid/machine_result_before_complete.invalid.conception");
    try expectBackendCFixture("../../../language/phase13-machines/valid/machine_backend_c_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase13-machines/invalid/machine_match_runtime_unsupported.invalid.conception");
    try expectBackendCFixture("../../../language/phase13-machines/invalid/machine_decide_runtime_unsupported.invalid.conception");
    try expectBackendCFixture("../../../language/phase13-machines/invalid/machine_non_scalar_result_unsupported.invalid.conception");
    try expectBackendCFixture("../../../language/phase13-machines/invalid/machine_non_scalar_param_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_step_assignment.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_step_return_value.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_step_condition.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_step_call_argument.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_step_binary_expr.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_step_discard.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/state_outside_machine.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/transition_outside_machine.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/transition_inside_function.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_missing_name.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_missing_return_type.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_missing_body.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_unclosed_state.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_missing_target.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_missing_semicolon.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_match_bad_pattern.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_match_missing_semicolon.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_match_non_state_result.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_decide_missing_score.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_decide_missing_semicolon.invalid.conception");
    try expectParseFixture("../../../language/phase13-machines/invalid/machine_transition_decide_non_state_result.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_zero_states.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_duplicate_state.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_transition_unknown_state.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_transition_cross_machine_state.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_transition_match_unknown_state.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_transition_match_cross_machine_state.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_transition_decide_unknown_state.invalid.conception");
    try expectCheckFixture("../../../language/phase13-machines/invalid/machine_transition_decide_cross_machine_state.invalid.conception");

    try expectRunFixture("../../../language/phase18-machines/valid/machine_local_step_result_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_local_complete_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_multiple_instances_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_copy_assignment_behavior_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_result_before_completion_panics_run.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/machine_frame_backend_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/machine_result_panic_backend_shape.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/nested_machine_field_parent_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/nested_machine_step_result_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/nested_machine_complete_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/nested_machine_result_before_completion_panics_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/nested_machine_explicit_stepping_only_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/nested_machine_multiple_child_results_run.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/nested_machine_field_basic_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/nested_machine_field_multiple_children_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/nested_machine_operations_backend.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_state_initial_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_state_after_step_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_state_nested_child_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/machine_state_no_completion_requirement_run.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/machine_state_backend.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/integration_hierarchical_child_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/integration_match_child_completion_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/integration_decide_utility_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/integration_nested_decide_child_run.valid.conception");
    try expectRunFixture("../../../language/phase18-machines/valid/integration_state_observation_run.valid.conception");
    try expectBackendCFixture("../../../language/phase18-machines/valid/integration_phase18_backend_shape.valid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_step_non_machine.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_step_temporary.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_complete_non_machine.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_result_non_machine.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/nested_machine_field_parameterized_child.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/nested_machine_field_duplicate_name.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/nested_machine_step_unknown_field.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/nested_machine_complete_unknown_field.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/nested_machine_result_unknown_field.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_state_non_machine.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_state_too_few_args.invalid.conception");
    try expectCheckFixture("../../../language/phase18-machines/invalid/machine_state_too_many_args.invalid.conception");

    try expectParseFixture("../../../language/phase15-c-abi/valid/extern_c_block_basic.valid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/valid/extern_c_block_multiple_functions.valid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/valid/extern_c_pointer_param.valid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/valid/extern_c_empty_block.valid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_abi_cpp.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_missing_abi_string.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_missing_block.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_function_body.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_missing_semicolon.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_non_function_item.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_global.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_varargs.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_hir_basic.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_hir_multiple_functions.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_hir_empty_block.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_hir_pointer_params.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_bool_param_return.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_arena_pointer_param.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_allocator_pointer_param.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/extern_c_call_resolves_semantic.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_call_abs_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_prototype_multiple.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_call_bool_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_call_pointer_param_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_empty_block_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_prototype_emitted_once.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_prototype_order.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_multiple_blocks_order.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_bool_abi_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/extern_c_alloc_error_abi_backend.valid.conception");
    try expectRunFixture("../../../language/phase15-c-abi/valid/extern_c_call_abs_run.valid.conception");
    try expectRunFixture("../../../language/phase15-c-abi/valid/export_c_add_run.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/repr_c_struct_marker.valid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/valid/repr_c_struct_marker_debug.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/valid/repr_c_struct_with_multiple_fields.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_struct_marker_no_abi_use.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_struct_int_fields_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_struct_bool_field_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_struct_pointer_field_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_extern_struct_param_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_extern_struct_pointer_param_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_typedef_before_extern_prototype.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_typedef_before_export_definition.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_typedef_emitted_once.valid.conception");
    try expectRunFixture("../../../language/phase15-c-abi/valid/repr_c_export_struct_param_run.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_invalid_target_function.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_invalid_target_enum.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_invalid_target_machine.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_missing_arg.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_unsupported_abi.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_too_many_args.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_duplicate_marker.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_empty_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_field_non_repr_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_nested_struct_by_value_still_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_empty_struct_abi_invalid.invalid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/export_c_backend_symbol.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/export_c_internal_helper_naming_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/export_c_void_backend.valid.conception");
    try expectRunFixture("../../../language/phase15-c-abi/valid/export_c_void_call_run.valid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_missing_body.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/export_c_unsupported_abi_cpp.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/export_c_missing_abi_string.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/export_c_block_form_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_duplicate_extern_symbol.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_duplicate_symbol.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_unsupported_return_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_unsupported_param_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/non_repr_struct_export_param_still_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/non_repr_struct_pointer_export_param_still_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_duplicate_symbol_same_block.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_duplicate_symbol_across_blocks.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_duplicate_top_level_name.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_return_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_param_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_param_struct_pointer.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/non_repr_struct_extern_param_still_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/non_repr_struct_pointer_extern_param_still_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_param_interface.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_param_dyn.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_param_manual_init.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_void_param_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_void_call_used_as_value.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_void_call_used_as_value.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_void_param_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_call_wrong_arg_type.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_call_wrong_arity.invalid.conception");

    // P15-M8 representative examples and final surface coverage.
    try expectRunFixture("../../../language/phase15-c-abi/valid/extern_c_call_abs_example.valid.conception");
    try expectRunFixture("../../../language/phase15-c-abi/valid/export_c_add_example.valid.conception");
    try expectRunFixture("../../../language/phase15-c-abi/valid/repr_c_export_point_sum_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_extern_point_prototype_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_pointer_prototype_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/c_abi_bool_signature_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/c_abi_alloc_error_signature_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/c_abi_no_headers_includes_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/export_c_add_backend_example.valid.conception");
    try expectBackendCFixture("../../../language/phase15-c-abi/valid/repr_c_export_point_sum_backend_example.valid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_varargs_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_struct_without_repr_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/extern_c_struct_without_repr_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_bad_field_dyn_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_empty_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/c_abi_dyn_param_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/c_abi_machine_param_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_missing_body_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/export_c_duplicate_symbol_example.invalid.conception");
    try expectParseFixture("../../../language/phase15-c-abi/invalid/extern_c_unsupported_abi_cpp_example.invalid.conception");
    try expectCheckFixture("../../../language/phase15-c-abi/invalid/repr_c_nested_by_value_example.invalid.conception");

    try expectParseFixture("../../../language/phase14-interfaces/valid/interface_single_requirement.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_multiple_requirements.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_requirement_params.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_requirement_void_return.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_requirement_scalar_params.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_requirement_pointer_param.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_requirement_struct_param.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_requirement_enum_return.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_basic.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_multiple_requirements.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_void_return.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_scalar_params.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_struct_target.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_enum_target.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_builtin_target.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/interface_impl_body_checked.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_param_read_ref.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_param_mut_ref.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_type_hir_surface.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_multiple_interfaces.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_coerce_arg_basic.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_coerce_arg_mut.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_coerce_arg_multiple_params.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_coerce_arg_exact_dyn_passthrough.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_coerce_arg_impl_separation.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_call_void_requirement.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_call_return_int_requirement.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_call_multiple_args.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/valid/dyn_call_after_coercion_hir.valid.conception");
    try expectBackendCFixture("../../../language/phase14-interfaces/valid/interface_backend_c_hir_only.valid.conception");
    try expectBackendCFixture("../../../language/phase14-interfaces/valid/interface_impl_backend_c_no_vtable.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_return_int.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_void_call.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_multiple_args.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_bool_return.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_two_impls_same_interface.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_two_interfaces_same_type.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_call_boundary_coercion.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_exact_dyn_passthrough.valid.conception");
    try expectRunFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_mut_receiver_required_valid.valid.conception");
    try expectBackendCFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_backend_c_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase14-interfaces/valid/dyn_dispatch_receiver_abi_shape.valid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_duplicate_top_level.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_empty.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_unknown_return_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_unknown_param_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_duplicate_requirement.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_requirement_overload.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_duplicate_requirement_param.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_by_value_interface_return.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_by_value_interface_param.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_pointer_interface_param.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_value_runtime_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_missing_requirement.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_extra_function.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_wrong_return.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_wrong_param_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_missing_receiver.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_wrong_receiver_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_duplicate_function.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_duplicate_impl.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_unknown_interface.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_invalid_target_interface_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_target_void.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/interface_impl_does_not_satisfy_concept.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_non_interface_struct.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_non_interface_enum.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_non_interface_concept.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_builtin_int.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_unknown_interface.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_by_value.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_raw_pointer.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_local_initializer_requires_coercion.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_return_type_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_struct_field_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_local_initializer_still_deferred.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_return_still_deferred.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_field_still_deferred.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_missing_impl.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_dispatch_missing_impl.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_wrong_interface_impl.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_wrong_concrete_impl.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_concept_impl_not_enough.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_temporary.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_dispatch_temporary_still_invalid.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_call_result.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_binary_expr.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_coerce_arg_move_expr.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_unknown_method.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_unknown_method.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_wrong_arity_too_few.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_wrong_arity_too_many.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_wrong_arg_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_wrong_arg_type.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_requires_mut_dyn.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_non_dyn_receiver.invalid.conception");
    try expectCheckFixture("../../../language/phase14-interfaces/invalid/dyn_call_void_used_as_value.invalid.conception");

    try expectRunFixture("../../../language/phase10-ownership/valid/move_struct_local_run.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_struct_argument_run.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_int_copy_run.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_bool_copy_run.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_int_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_bool_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/assign_copy_int_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/assign_copy_bool_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_param_forward_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_enum_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_pointer_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_marker_struct_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/assign_copy_marker_struct_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/reinitialize_after_move_struct_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_branch_return_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_if_else_both_return_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_scalar_after_branch_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/move_in_one_branch_no_later_use_valid.valid.conception");
    try expectRunFixture("../../../language/phase10-ownership/valid/copy_field_read_valid.valid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/valid/drop_local_reverse_order.valid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/valid/drop_moved_local_skipped.valid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/valid/drop_early_return.valid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/valid/drop_return_move.valid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/valid/reinitialize_after_move_drop_valid.valid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/valid/manual_init_assume_init_check.valid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/use_after_move_local.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/use_after_move_argument.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/move_temporary_call.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/move_struct_literal.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/move_field_unsupported.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/move_after_move.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/drop_invalid_signature.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/drop_compile_time_function_invalid.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_struct_local.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_struct_argument.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_struct_return.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_struct_assignment.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_param_forward.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_drop_field.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_drop_field_return.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/implicit_copy_noncopy_field_argument.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/drop_param_deferred.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/assign_noncopy_struct_replacement.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/assign_drop_replacement.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/assign_maybe_moved_replacement.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/drop_maybe_moved_cleanup.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/maybe_moved_after_if.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/maybe_moved_move_after_if.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/maybe_moved_argument_after_if.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/maybe_moved_return_after_if.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/manual_init_assume_init_requires_unsafe.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/manual_init_missing_type_arg.invalid.conception");
    try expectCheckFixture("../../../language/phase10-ownership/invalid/manual_init_implicit_to_t.invalid.conception");
    try expectBackendCFixture("../../../language/phase10-ownership/invalid/manual_init_copy_rejected.invalid.conception");
}

test "language parse fixture: phase16 import single module" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_single_module_parse.valid.conception");
}

test "language parse fixture: phase16 import dotted module" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_dotted_module_parse.valid.conception");
}

test "language parse fixture: phase16 import multiple modules" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_multiple_modules_parse.valid.conception");
}

test "language parse fixture: phase16 import graph unknown import" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_unknown_import.invalid.conception");
}

test "language parse fixture: phase16 import graph single edge" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_graph_single_edge.valid.conception");
}

test "language parse fixture: phase16 import graph dotted edge" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_graph_dotted_edge.valid.conception");
}

test "language parse fixture: phase16 import graph multiple edges" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_graph_multiple_edges.valid.conception");
}

test "language parse fixture: phase16 import graph shared dependency" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_graph_shared_dependency.valid.conception");
}

test "language parse fixture: phase16 import graph duplicate across modules valid" {
    try expectParseFixture("../../../language/phase16-imports/valid/import_graph_duplicate_import_across_modules_valid.valid.conception");
}

test "language parse fixture: phase16 import graph unknown dotted import" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_unknown_dotted_import.invalid.conception");
}

test "language parse fixture: phase16 import graph duplicate import" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_duplicate_import.invalid.conception");
}

test "language parse fixture: phase16 import graph duplicate dotted import" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_duplicate_dotted_import.invalid.conception");
}

test "language parse fixture: phase16 import graph self cycle" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_self_import_cycle.invalid.conception");
}

test "language parse fixture: phase16 import graph direct cycle" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_direct_cycle.invalid.conception");
}

test "language parse fixture: phase16 import graph long cycle" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_graph_long_cycle.invalid.conception");
}

test "language parse fixture: phase16 multifile import syntax" {
    try expectParseFixture("../../../language/phase16-imports/valid/multifile_import_syntax_parse.valid.conception");
}

test "language parse fixture: phase16 import before module" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_before_module.invalid.conception");
}

test "language parse fixture: phase16 import after declaration" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_after_declaration.invalid.conception");
}

test "language parse fixture: phase16 import missing semicolon" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_missing_semicolon.invalid.conception");
}

test "language parse fixture: phase16 import string path invalid" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_string_path_invalid.invalid.conception");
}

test "language parse fixture: phase16 import wildcard invalid" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_wildcard_invalid.invalid.conception");
}

test "language parse fixture: phase16 import alias invalid" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_alias_invalid.invalid.conception");
}

test "language parse fixture: phase16 import list invalid" {
    try expectParseFixture("../../../language/phase16-imports/invalid/import_list_invalid.invalid.conception");
}

test "language parse fixture: phase16 export import invalid" {
    try expectParseFixture("../../../language/phase16-imports/invalid/export_import_invalid.invalid.conception");
}

test "language parse fixture: phase16 multi-file two sources" {
    try expectParseFixture("../../../language/phase16-imports/valid/multifile_two_sources_parse.valid.conception");
}

test "language parse fixture: phase16 multi-file three sources" {
    try expectParseFixture("../../../language/phase16-imports/valid/multifile_three_sources_parse.valid.conception");
}

test "language parse fixture: phase16 multi-file virtual paths" {
    try expectParseFixture("../../../language/phase16-imports/valid/multifile_virtual_paths_parse.valid.conception");
}

test "language parse fixture: phase16 multi-file empty second source" {
    try expectParseFixture("../../../language/phase16-imports/valid/multifile_empty_second_source_parse.valid.conception");
}

test "language parse fixture: phase16 module table two modules" {
    try expectParseFixture("../../../language/phase16-imports/valid/module_table_two_modules.valid.conception");
}

test "language parse fixture: phase16 module table three modules" {
    try expectParseFixture("../../../language/phase16-imports/valid/module_table_three_modules.valid.conception");
}

test "language parse fixture: phase16 module table dotted module names" {
    try expectParseFixture("../../../language/phase16-imports/valid/module_table_dotted_module_names.valid.conception");
}

test "language parse fixture: phase16 module table same item names different modules" {
    try expectParseFixture("../../../language/phase16-imports/valid/module_table_same_item_names_different_modules.valid.conception");
}

test "language parse fixture: phase16 duplicate module name" {
    try expectParseFixture("../../../language/phase16-imports/invalid/module_table_duplicate_module_name.invalid.conception");
}

test "language parse fixture: phase16 missing module in multifile" {
    try expectParseFixture("../../../language/phase16-imports/invalid/module_table_missing_module_in_multifile.invalid.conception");
}

test "language parse fixture: phase16 multiple modules in file" {
    try expectParseFixture("../../../language/phase16-imports/invalid/module_table_multiple_modules_in_file.invalid.conception");
}

test "language parse fixture: phase16 duplicate module name different paths" {
    try expectParseFixture("../../../language/phase16-imports/invalid/module_table_duplicate_module_name_different_paths.invalid.conception");
}

test "language parse fixture: phase16 empty module file" {
    try expectParseFixture("../../../language/phase16-imports/invalid/module_table_empty_module_file.invalid.conception");
}

test "language fixture format: phase16 duplicate virtual path rejected" {
    const path = "../../../language/phase16-imports/invalid/multifile_duplicate_virtual_path.invalid.conception";
    try std.testing.expectError(error.DuplicateVirtualFilePath, parse(std.testing.allocator, @embedFile(path), .{ .path = path }));
}

test "language fixture format: phase16 missing virtual path rejected" {
    const path = "../../../language/phase16-imports/invalid/multifile_missing_virtual_path.invalid.conception";
    try std.testing.expectError(error.MissingVirtualFilePath, parse(std.testing.allocator, @embedFile(path), .{ .path = path }));
}

test "language check fixture: phase16 hir same function names across modules" {
    try expectCheckFixture("../../../language/phase16-imports/valid/hir_modules_same_function_name_different_modules.valid.conception");
}

test "language check fixture: phase16 hir same struct names across modules" {
    try expectCheckFixture("../../../language/phase16-imports/valid/hir_modules_same_struct_name_different_modules.valid.conception");
}

test "language check fixture: phase16 hir import edges preserved" {
    try expectCheckFixture("../../../language/phase16-imports/valid/hir_modules_import_edges_preserved.valid.conception");
}

test "language check fixture: phase16 hir duplicate function same module" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/hir_modules_duplicate_function_same_module.invalid.conception");
}

test "language check fixture: phase16 hir duplicate struct same module" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/hir_modules_duplicate_struct_same_module.invalid.conception");
}

test "language check fixture: phase16 hir duplicate export c symbol across modules" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/hir_modules_duplicate_export_c_symbol_across_modules.invalid.conception");
}

test "language check fixture: phase16 hir duplicate extern c symbol across modules" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/hir_modules_duplicate_extern_c_symbol_across_modules.invalid.conception");
}

test "language check fixture: phase16 qualified imported struct type param" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_struct_param.valid.conception");
}

test "language check fixture: phase16 qualified imported enum type param" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_enum_param.valid.conception");
}

test "language check fixture: phase16 qualified same struct name different modules" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_same_struct_name_different_modules.valid.conception");
}

test "language check fixture: phase16 qualified current module struct type" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_current_module_struct.valid.conception");
}

test "language check fixture: phase16 qualified repr c export param" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_repr_c_export_param.valid.conception");
}

test "language check fixture: phase16 qualified repr c extern param" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_repr_c_extern_param.valid.conception");
}

test "language check fixture: phase16 qualified repr c pointer extern param" {
    try expectCheckFixture("../../../language/phase16-imports/valid/qualified_type_repr_c_pointer_extern_param.valid.conception");
}

test "language check fixture: phase16 qualified type module not imported" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_module_not_imported.invalid.conception");
}

test "language check fixture: phase16 qualified type unknown module" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_unknown_module.invalid.conception");
}

test "language check fixture: phase16 qualified type missing item" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_missing_item.invalid.conception");
}

test "language check fixture: phase16 qualified type item not type" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_item_not_type.invalid.conception");
}

test "language check fixture: phase16 unqualified imported type remains unknown" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_unqualified_imported_type.invalid.conception");
}

test "language check fixture: phase16 qualified non repr export param" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_non_repr_export_param.invalid.conception");
}

test "language check fixture: phase16 qualified non repr pointer extern param" {
    try expectCheckFixture("../../../language/phase16-imports/invalid/qualified_type_non_repr_pointer_extern_param.invalid.conception");
}

test "language run fixture: phase16 cross module add" {
    try expectRunFixture("../../../language/phase16-imports/valid/cross_module_add_run.valid.conception");
}

test "language run fixture: phase16 same function names qualified" {
    try expectRunFixture("../../../language/phase16-imports/valid/same_function_names_qualified_run.valid.conception");
}

test "language run fixture: phase16 imported struct type use" {
    try expectRunFixture("../../../language/phase16-imports/valid/imported_struct_type_run.valid.conception");
}

test "language run fixture: phase16 imported extern C abs" {
    try expectRunFixture("../../../language/phase16-imports/valid/imported_extern_c_abs_run.valid.conception");
}

test "language run fixture: phase16 same struct names qualified" {
    try expectRunFixture("../../../language/phase16-imports/valid/same_struct_names_qualified_run.valid.conception");
}

test "language check fixture: phase17 panic statement basic" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/panic_statement_basic.valid.conception");
}

test "language check fixture: phase17 panic statement preserves reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/panic_statement_preserves_reason.valid.conception");
}

test "language check fixture: phase17 panic statement after code" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/panic_statement_after_code.valid.conception");
}

test "language check fixture: phase17 panic non empty reason still valid" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/panic_non_empty_reason_still_valid.valid.conception");
}

test "language check fixture: phase17 panic missing reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_missing_reason.invalid.conception");
}

test "language check fixture: phase17 panic too many args" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_too_many_args.invalid.conception");
}

test "language check fixture: phase17 panic non string reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_non_string_reason.invalid.conception");
}

test "language check fixture: phase17 panic empty reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_empty_reason.invalid.conception");
}

test "language check fixture: phase17 panic whitespace reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_whitespace_reason.invalid.conception");
}

test "language check fixture: phase17 panic tab reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_tab_reason.invalid.conception");
}

test "language check fixture: phase17 panic expression initializer" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_expression_initializer.invalid.conception");
}

test "language check fixture: phase17 panic return expression" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_return_expression.invalid.conception");
}

test "language run fixture: phase17 panic exits 101" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/panic_run_exits_101.valid.conception");
}

test "language backend-c fixture: phase17 panic helper emitted" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/panic_backend_helper_emitted.valid.conception");
}

test "language backend-c fixture: phase17 panic helper emitted once" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/panic_backend_helper_emitted_once.valid.conception");
}

test "language backend-c fixture: phase17 panic reason literal" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/panic_backend_reason_literal.valid.conception");
}

test "language backend-c fixture: phase17 panic has no test runner dependency" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/panic_backend_no_test_runner_dependency.valid.conception");
}

test "language check fixture: phase17 assert statement basic" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/assert_statement_basic.valid.conception");
}

test "language check fixture: phase17 assert statement preserves reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/assert_statement_preserves_reason.valid.conception");
}

test "language check fixture: phase17 assert statement condition expression" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/assert_statement_condition_expression.valid.conception");
}

test "language check fixture: phase17 assert statement after code" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/assert_statement_after_code.valid.conception");
}

test "language check fixture: phase17 assert non empty reason still valid" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/valid/assert_non_empty_reason_still_valid.valid.conception");
}

test "language check fixture: phase17 assert missing all args" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_missing_all_args.invalid.conception");
}

test "language check fixture: phase17 assert missing reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_missing_reason.invalid.conception");
}

test "language check fixture: phase17 assert too many args" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_too_many_args.invalid.conception");
}

test "language check fixture: phase17 assert non string reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_non_string_reason.invalid.conception");
}

test "language check fixture: phase17 assert empty reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_empty_reason.invalid.conception");
}

test "language check fixture: phase17 assert whitespace reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_whitespace_reason.invalid.conception");
}

test "language check fixture: phase17 assert tab reason" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_tab_reason.invalid.conception");
}

test "language check fixture: phase17 assert non bool condition" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_non_bool_condition.invalid.conception");
}

test "language check fixture: phase17 assert expression initializer" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_expression_initializer.invalid.conception");
}

test "language check fixture: phase17 assert return expression" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_return_expression.invalid.conception");
}

test "language run fixture: phase17 assert true continues" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/assert_true_run.valid.conception");
}

test "language run fixture: phase17 assert false exits 101" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/assert_false_exits_101.valid.conception");
}

test "language run fixture: phase17 assert condition expression continues" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/assert_condition_expression_run.valid.conception");
}

test "language run fixture: phase17 assert after code continues" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/assert_after_code_run.valid.conception");
}

test "language backend-c fixture: phase17 assert lowers to if panic" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/assert_backend_lowers_to_if_panic.valid.conception");
}

test "language backend-c fixture: phase17 assert helper emitted" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/assert_backend_helper_emitted.valid.conception");
}

test "language backend-c fixture: phase17 assert helper emitted once" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/assert_backend_helper_emitted_once.valid.conception");
}

test "language backend-c fixture: phase17 assert helper once with panic" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/assert_backend_helper_once_with_panic.valid.conception");
}

test "language backend-c fixture: phase17 assert reason literal" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/assert_backend_reason_literal.valid.conception");
}

test "language backend-c fixture: phase17 assert no test runner dependency" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/assert_backend_no_test_runner_dependency.valid.conception");
}

test "language run fixture: phase17 machine result before completion panics" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/machine_result_before_completion_panics.valid.conception");
}

test "language run fixture: phase17 machine result after completion still works" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/machine_result_after_completion_still_works.valid.conception");
}

test "language backend-c fixture: phase17 runtime traps share panic helper" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/runtime_traps_share_panic_helper.valid.conception");
}

test "language run fixture: phase17 M8 panic example exits 101" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/panic_example_exits_101.valid.conception");
}

test "language run fixture: phase17 M8 assert true example runs" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/assert_true_example_runs.valid.conception");
}

test "language run fixture: phase17 M8 assert false example exits 101" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/assert_false_example_exits_101.valid.conception");
}

test "language run fixture: phase17 M8 machine result before completion example exits 101" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/machine_result_before_completion_example_exits_101.valid.conception");
}

test "language run fixture: phase17 M8 machine result after completion example runs" {
    try expectRunFixture("../../../language/phase17-runtime-failure/valid/machine_result_after_completion_example_runs.valid.conception");
}

test "language backend-c fixture: phase17 M8 panic assert machine helper shared" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/panic_assert_machine_helper_shared_backend.valid.conception");
}

test "language backend-c fixture: phase17 M8 runtime assert without Core.Test" {
    try expectBackendCFixture("../../../language/phase17-runtime-failure/valid/runtime_assert_without_core_test_example.valid.conception");
}

test "language check fixture: phase17 M8 panic blank reason example" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_blank_reason_example.invalid.conception");
}

test "language check fixture: phase17 M8 assert blank reason example" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_blank_reason_example.invalid.conception");
}

test "language check fixture: phase17 M8 panic expression use example" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/panic_expression_use_example.invalid.conception");
}

test "language check fixture: phase17 M8 assert expression use example" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_expression_use_example.invalid.conception");
}

test "language check fixture: phase17 M8 assert non bool condition example" {
    try expectCheckFixture("../../../language/phase17-runtime-failure/invalid/assert_non_bool_condition_example.invalid.conception");
}

test "language run fixture: phase18 transition match bool true" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_match_bool_true_run.valid.conception");
}

test "language run fixture: phase18 transition match bool false" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_match_bool_false_run.valid.conception");
}

test "language run fixture: phase18 transition match complete child" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_match_complete_child_run.valid.conception");
}

test "language backend-c fixture: phase18 transition match bool branch" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/transition_match_bool_backend.valid.conception");
}

test "language check fixture: phase18 transition match non bool scrutinee" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_non_bool_scrutinee.invalid.conception");
}

test "language check fixture: phase18 transition match bool missing false" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_bool_missing_false.invalid.conception");
}

test "language check fixture: phase18 transition match bool missing true" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_bool_missing_true.invalid.conception");
}

test "language check fixture: phase18 transition match duplicate true" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_duplicate_true.invalid.conception");
}

test "language check fixture: phase18 transition match case type mismatch" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_case_type_mismatch.invalid.conception");
}

test "language check fixture: phase18 transition match unknown target state" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_unknown_target_state.invalid.conception");
}

test "language check fixture: phase18 transition match empty" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_match_empty.invalid.conception");
}

test "language run fixture: phase18 transition decide highest score" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_decide_highest_score_run.valid.conception");
}

test "language run fixture: phase18 transition decide guard false ignored" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_decide_guard_false_ignored_run.valid.conception");
}

test "language run fixture: phase18 transition decide tie source order" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_decide_tie_source_order_run.valid.conception");
}

test "language run fixture: phase18 transition decide unconditional candidate" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_decide_unconditional_candidate_run.valid.conception");
}

test "language run fixture: phase18 transition decide no enabled panics" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_decide_no_enabled_panics_run.valid.conception");
}

test "language run fixture: phase18 transition decide nested child score" {
    try expectRunFixture("../../../language/phase18-machines/valid/transition_decide_nested_child_score_run.valid.conception");
}

test "language backend-c fixture: phase18 transition decide" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/transition_decide_backend.valid.conception");
}

test "language backend-c fixture: phase18 machine result panic reason" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/machine_result_before_completion_panic_reason_backend.valid.conception");
}

test "language backend-c fixture: phase18 nested machine result panic reason" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/nested_machine_result_before_completion_panic_reason_backend.valid.conception");
}

test "language backend-c fixture: phase18 transition decide no enabled panic reason" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/transition_decide_no_enabled_panic_backend.valid.conception");
}

test "language backend-c fixture: phase18 machine multiple panic sites" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/machine_multiple_panic_sites_backend.valid.conception");
}

test "language backend-c fixture: phase18 invalid machine state defensive panic" {
    try expectBackendCFixture("../../../language/phase18-machines/valid/machine_invalid_state_defensive_backend.valid.conception");
}

test "language check fixture: phase18 transition decide empty" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_decide_empty.invalid.conception");
}

test "language check fixture: phase18 transition decide guard not bool" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_decide_guard_not_bool.invalid.conception");
}

test "language check fixture: phase18 transition decide score not int" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_decide_score_not_int.invalid.conception");
}

test "language check fixture: phase18 transition decide unknown target state" {
    try expectCheckFixture("../../../language/phase18-machines/invalid/transition_decide_unknown_target_state.invalid.conception");
}

test "language fixtures: phase19 yield validation" {
    try expectCheckFixture("../../../language/phase19-yielding-machines/valid/machine_yield_statement.valid.conception");
    try expectCheckFixture("../../../language/phase19-yielding-machines/valid/machine_yield_in_if.valid.conception");
    try expectCheckFixture("../../../language/phase19-yielding-machines/valid/machine_yield_in_nested_block.valid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_outside_machine.invalid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_inside_function_block.invalid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_top_level.invalid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_value.invalid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_return.invalid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_expression.invalid.conception");
    try expectParseFixture("../../../language/phase19-yielding-machines/invalid/yield_missing_semicolon.invalid.conception");
}

test "language run fixtures: phase19 yield runtime lowering" {
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_keeps_machine_incomplete_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_preserves_state_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_reenters_same_state_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_in_if_branch_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_result_after_yield_panics_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_child_state_readable_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_transition_match_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_transition_decide_run.valid.conception");
    try expectRunFixture("../../../language/phase19-yielding-machines/valid/yield_multi_step_child_poll_run.valid.conception");
}

test "language backend-c fixture: phase19 yield lowers to step return" {
    try expectBackendCFixture("../../../language/phase19-yielding-machines/valid/yield_backend_step_return.valid.conception");
    try expectBackendCFixture("../../../language/phase19-yielding-machines/valid/yield_backend_transition_decide.valid.conception");
    try expectBackendCFixture("../../../language/phase19-yielding-machines/valid/yield_backend_no_runtime_failure.valid.conception");
    try expectBackendCFixture("../../../language/phase19-yielding-machines/valid/yield_result_after_yield_panic_reason_backend.valid.conception");
}

test "language run fixtures: phase20 DragonGod Kernel core seed" {
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_core_ids.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_automata_signal.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_shell.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_agent_world.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_cross_module_imports.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_succeed_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_fail_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_goto_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_yield_then_succeed_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_match_true_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_match_false_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_decide_high_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_decide_low_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_act_smoke_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_signal_await_actuation_smoke_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_write_read_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_read_fallback_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_has_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_revision_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_update_existing_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_signal_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_record_succeed_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_record_fail_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_record_goto_deferred_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_static_succeed_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_static_yield_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_memory_signal_machine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_highest_score_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_disabled_option_ignored_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_source_order_tie_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_fallback_no_enabled_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_first_selection_records_current_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_keep_current_same_option_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_min_commit_blocks_switch_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_hysteresis_blocks_switch_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_tie_epsilon_prefers_current_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_switch_after_policy_allows_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_memory_integration_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_mind_goto_integration_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_transition_decide_doctrine_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_allow_dispatch_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_deny_dispatch_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_complete_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_fail_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_status_missing_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_signal_act_match_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_signal_await_match_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_mind_record_act_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_push_read_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_read_fallback_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_tail_cursor_future_only_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_order_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_multi_type_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_cursor_advances_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_memory_integration_run.valid.conception");
    try expectRunFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_signal_machine_run.valid.conception");
}

test "language backend-c fixture: phase20 DragonGod automata signal machines" {
    try expectBackendCFixture("../../../language/phase20-dragongod-kernel/valid/kernel_automata_signal_machine_backend.valid.conception");
    try expectBackendCFixture("../../../language/phase20-dragongod-kernel/valid/kernel_memory_backend_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase20-dragongod-kernel/valid/kernel_mind_backend_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase20-dragongod-kernel/valid/kernel_decision_backend_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase20-dragongod-kernel/valid/kernel_actuation_backend_shape.valid.conception");
    try expectBackendCFixture("../../../language/phase20-dragongod-kernel/valid/kernel_events_backend_shape.valid.conception");
}
