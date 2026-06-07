// ─────────────────────────────────────────────────────────────────────────────
// Fixture metadata model
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const parser_model = @import("../parser.zig");
const source_model = @import("../source.zig");
const checker_model = @import("../checker.zig");
const semantics_model = @import("../semantics.zig");
const hir_checker_model = @import("../hir_checker.zig");
const mir_lowering_model = @import("../mir_lowering.zig");
const mir_validator_model = @import("../mir_validator.zig");
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

pub const ConceptionFixture = struct {
    name: []const u8,
    phase: Phase,
    expect: Expectation,
    check: ?CheckMode = null,
    sections: []const Section,

    pub fn checkMode(self: ConceptionFixture) CheckMode {
        return self.check orelse .declarations;
    }

    pub fn source(self: ConceptionFixture) ?[]const u8 {
        return self.section("source");
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

    const parsed = ConceptionFixture{
        .name = name orelse return error.MissingName,
        .phase = phase orelse return error.MissingPhase,
        .expect = expect orelse return error.MissingExpectation,
        .check = check,
        .sections = try sections.toOwnedSlice(),
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

// ─────────────────────────────────────────────────────────────────────────────
// Validation
// ─────────────────────────────────────────────────────────────────────────────

fn validate(fixture: ConceptionFixture, options: ParseOptions) !void {
    if (fixture.source() == null) return error.MissingSource;

    if (fixture.check != null and fixture.phase != .check) return error.CheckModeOnNonCheckPhase;

    switch (fixture.phase) {
        .parse => switch (fixture.expect) {
            .pass => if (fixture.ast() == null) return error.MissingAst,
            .fail => if (fixture.diagnostics() == null) return error.MissingDiagnostics,
        },
        .run => {
            if (fixture.expect != .pass) return error.UnsupportedRunExpectation;
            _ = try fixture.expectedExitCode();
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

    try std.testing.expectEqualStrings("module Example;", fixture.source().?);
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

    var diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, fixture.source().?);
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

test "language check fixture: phase8 generic constrained unsupported" {
    try expectCheckFixture("../../../language/phase8-concepts-templates/invalid/generic_constrained_call_unsupported.invalid.conception");
}

// ─────────────────────────────────────────────────────────────────────────────
// Phase execution helpers
// ─────────────────────────────────────────────────────────────────────────────

fn expectCheckFixture(comptime path: []const u8) !void {
    const text = @embedFile(path);
    const fixture = try parse(std.testing.allocator, text, .{ .path = path });
    defer fixture.deinit(std.testing.allocator);

    try std.testing.expectEqual(Phase.check, fixture.phase);

    if (std.mem.indexOf(u8, path, "phase3-semantics") != null or std.mem.indexOf(u8, path, "phase8-concepts-templates") != null or std.mem.indexOf(u8, path, "phase5-sum-types") != null or std.mem.indexOf(u8, path, "phase5a-judgment") != null or std.mem.indexOf(u8, path, "phase6-unsafe-ownership") != null or std.mem.indexOf(u8, path, "phase7-runtime-structs") != null) {
        try expectSemanticCheckFixture(path, fixture);
    } else {
        try expectPhase2CheckFixture(path, fixture);
    }
}

fn expectSemanticCheckFixture(comptime path: []const u8, fixture: ConceptionFixture) !void {
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var semantic_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer semantic_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, fixture.source().?);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    const use_hir_checker = fixture.checkMode() == .hir;

    switch (fixture.expect) {
        .pass => {
            var module = try semantics_model.collectTopLevelDeclarations(std.testing.allocator, unit, &semantic_diagnostics);
            defer module.deinit();
            if (use_hir_checker) {
                try hir_checker_model.checkExecutable(std.testing.allocator, &module, &semantic_diagnostics);
            }
            try std.testing.expectEqual(@as(usize, 0), semantic_diagnostics.count());
        },
        .fail => {
            var maybe_module = semantics_model.collectTopLevelDeclarations(std.testing.allocator, unit, &semantic_diagnostics) catch |err| switch (err) {
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

fn expectPhase2CheckFixture(comptime path: []const u8, fixture: ConceptionFixture) !void {
    var parse_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer parse_diagnostics.deinit();
    var check_diagnostics = parser_model.DiagnosticBag.init(std.testing.allocator);
    defer check_diagnostics.deinit();

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, fixture.source().?);
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

    const source_file = try source_model.SourceFile.init(std.testing.allocator, path, fixture.source().?);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parser_model.parseSource(std.testing.allocator, source_file, &parse_diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), parse_diagnostics.count());

    var module = try semantics_model.collectTopLevelDeclarations(std.testing.allocator, unit, &semantic_diagnostics);
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

    var module = try semantics_model.collectTopLevelDeclarations(std.testing.allocator, unit, &semantic_diagnostics);
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
    try std.testing.expectEqual(Expectation.pass, fixture.expect);
    _ = try run_harness.expectExitCode(std.testing.allocator, fixture.source().?, try fixture.expectedExitCode());
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
}
