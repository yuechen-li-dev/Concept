const std = @import("std");

pub const ManifestKind = enum {
    package,
    registry,

    fn parse(value: []const u8) ?ManifestKind {
        if (std.mem.eql(u8, value, "package")) return .package;
        if (std.mem.eql(u8, value, "registry")) return .registry;
        return null;
    }
};

pub const ManifestFormat = enum {
    concept_manifest_v0,
    concept_registry_v0,

    fn parse(value: []const u8) ?ManifestFormat {
        if (std.mem.eql(u8, value, "concept-manifest-v0")) return .concept_manifest_v0;
        if (std.mem.eql(u8, value, "concept-registry-v0")) return .concept_registry_v0;
        return null;
    }
};

pub const PackageInfo = struct {
    name: []const u8,
    version: ?[]const u8 = null,
    license: ?[]const u8 = null,

    pub fn deinit(self: PackageInfo, allocator: std.mem.Allocator) void {
        allocator.free(self.name);
        if (self.version) |version| allocator.free(version);
        if (self.license) |license| allocator.free(license);
    }
};

pub const Section = struct {
    /// Section names and bodies borrow from the original manifest text passed to parse().
    /// The caller must keep that text alive for as long as the Manifest is used.
    name: []const u8,
    body: []const u8,
};

pub const Manifest = struct {
    kind: ManifestKind,
    format: ManifestFormat,
    package: ?PackageInfo,
    sections: []Section,

    pub fn deinit(self: Manifest, allocator: std.mem.Allocator) void {
        if (self.package) |package| package.deinit(allocator);
        allocator.free(self.sections);
    }

    pub fn section(self: Manifest, name: []const u8) ?Section {
        for (self.sections) |candidate| {
            if (std.mem.eql(u8, candidate.name, name)) return candidate;
        }
        return null;
    }
};

pub fn parse(allocator: std.mem.Allocator, text: []const u8) !Manifest {
    var kind: ?ManifestKind = null;
    var format: ?ManifestFormat = null;
    var sections = std.ArrayList(Section).empty;
    errdefer sections.deinit(allocator);

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

        if (sectionName(line)) |name| {
            if (current_section_name) |previous_name| {
                try appendSection(allocator, &sections, previous_name, trimTrailingLineEnding(text[current_section_start..line_start]));
            }
            current_section_name = name;
            current_section_start = cursor;
            saw_section = true;
            continue;
        }

        if (!saw_section) {
            const trimmed = std.mem.trim(u8, line, " \t\r");
            if (trimmed.len == 0) continue;
            if (!std.mem.startsWith(u8, trimmed, "#")) return error.ExpectedHeaderOrSection;
            try parseHeader(trimmed, &kind, &format);
        }
    }

    if (current_section_name) |previous_name| {
        try appendSection(allocator, &sections, previous_name, trimTrailingLineEnding(text[current_section_start..text.len]));
    }

    const manifest_kind = kind orelse return error.MissingKind;
    const manifest_format = format orelse return error.MissingFormat;
    try validateKindFormat(manifest_kind, manifest_format);

    const owned_sections = try sections.toOwnedSlice(allocator);
    var manifest = Manifest{
        .kind = manifest_kind,
        .format = manifest_format,
        .package = null,
        .sections = owned_sections,
    };
    errdefer manifest.deinit(allocator);

    switch (manifest.kind) {
        .package => {
            const package_section = manifest.section("package") orelse return error.MissingPackageSection;
            manifest.package = try parsePackageSection(allocator, package_section.body);
        },
        .registry => {
            if (manifest.section("registry") == null) return error.MissingRegistrySection;
        },
    }

    return manifest;
}

fn parseHeader(line: []const u8, kind: *?ManifestKind, format: *?ManifestFormat) !void {
    const without_hash = std.mem.trim(u8, line[1..], " \t");
    const colon = std.mem.indexOfScalar(u8, without_hash, ':') orelse return error.InvalidHeader;
    const key = std.mem.trim(u8, without_hash[0..colon], " \t");
    const value = std.mem.trim(u8, without_hash[colon + 1 ..], " \t");

    if (std.mem.eql(u8, key, "kind")) {
        if (kind.* != null) return error.DuplicateKind;
        kind.* = ManifestKind.parse(value) orelse return error.InvalidKind;
    } else if (std.mem.eql(u8, key, "format")) {
        if (format.* != null) return error.DuplicateFormat;
        format.* = ManifestFormat.parse(value) orelse return error.InvalidFormat;
    } else {
        return error.UnknownHeader;
    }
}

fn validateKindFormat(kind: ManifestKind, format: ManifestFormat) !void {
    switch (kind) {
        .package => if (format != .concept_manifest_v0) return error.ManifestKindFormatMismatch,
        .registry => if (format != .concept_registry_v0) return error.ManifestKindFormatMismatch,
    }
}

fn appendSection(allocator: std.mem.Allocator, sections: *std.ArrayList(Section), name: []const u8, body: []const u8) !void {
    for (sections.items) |existing| {
        if (std.mem.eql(u8, existing.name, name)) return error.DuplicateSection;
    }
    try sections.append(allocator, .{ .name = name, .body = body });
}

fn parsePackageSection(allocator: std.mem.Allocator, body: []const u8) !PackageInfo {
    var package = PackageBuilder{};
    errdefer package.deinit(allocator);

    var lines = std.mem.splitScalar(u8, body, '\n');
    while (lines.next()) |raw_line| {
        const line = std.mem.trim(u8, raw_line, " \t\r");
        if (line.len == 0) continue;
        try parsePackageStatement(allocator, line, &package);
    }

    return .{
        .name = package.name orelse return error.MissingPackageName,
        .version = package.version,
        .license = package.license,
    };
}

const PackageBuilder = struct {
    name: ?[]const u8 = null,
    version: ?[]const u8 = null,
    license: ?[]const u8 = null,

    fn deinit(self: PackageBuilder, allocator: std.mem.Allocator) void {
        if (self.name) |name| allocator.free(name);
        if (self.version) |version| allocator.free(version);
        if (self.license) |license| allocator.free(license);
    }
};

fn parsePackageStatement(allocator: std.mem.Allocator, line: []const u8, package: *PackageBuilder) !void {
    if (!std.mem.endsWith(u8, line, ";")) return error.ExpectedSemicolon;
    const statement = std.mem.trim(u8, line[0 .. line.len - 1], " \t");

    if (std.mem.startsWith(u8, statement, "package")) {
        const name = std.mem.trim(u8, statement["package".len..], " \t");
        if (name.len == 0 or !isIdentifier(name)) return error.InvalidPackageStatement;
        if (package.name != null) return error.DuplicatePackageName;
        package.name = try allocator.dupe(u8, name);
    } else if (std.mem.startsWith(u8, statement, "version")) {
        const value = std.mem.trim(u8, statement["version".len..], " \t");
        if (package.version != null) return error.DuplicateVersion;
        package.version = try parseStringLiteral(allocator, value);
    } else if (std.mem.startsWith(u8, statement, "license")) {
        const value = std.mem.trim(u8, statement["license".len..], " \t");
        if (package.license != null) return error.DuplicateLicense;
        package.license = try parseStringLiteral(allocator, value);
    } else {
        return error.InvalidPackageStatement;
    }
}

fn parseStringLiteral(allocator: std.mem.Allocator, value: []const u8) ![]const u8 {
    if (value.len < 2 or value[0] != '"' or value[value.len - 1] != '"') return error.InvalidStringLiteral;
    const inner = value[1 .. value.len - 1];
    if (std.mem.indexOfScalar(u8, inner, '"') != null) return error.InvalidStringLiteral;
    if (std.mem.indexOfScalar(u8, inner, '\\') != null) return error.InvalidStringLiteral;
    return allocator.dupe(u8, inner);
}

fn isIdentifier(value: []const u8) bool {
    if (value.len == 0) return false;
    if (!isIdentifierStart(value[0])) return false;
    for (value[1..]) |byte| {
        if (!isIdentifierContinue(byte)) return false;
    }
    return true;
}

fn isIdentifierStart(byte: u8) bool {
    return (byte >= 'A' and byte <= 'Z') or (byte >= 'a' and byte <= 'z') or byte == '_';
}

fn isIdentifierContinue(byte: u8) bool {
    return isIdentifierStart(byte) or (byte >= '0' and byte <= '9');
}

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

fn expectManifest(text: []const u8) !Manifest {
    return parse(std.testing.allocator, text);
}

const minimal_package_manifest =
    \\# kind: package
    \\# format: concept-manifest-v0
    \\
    \\=== package ===
    \\package Compiler;
;

test "manifest parser parses minimal package manifest" {
    const manifest = try expectManifest(minimal_package_manifest);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(ManifestKind.package, manifest.kind);
    try std.testing.expectEqual(ManifestFormat.concept_manifest_v0, manifest.format);
    try std.testing.expectEqualStrings("Compiler", manifest.package.?.name);
    try std.testing.expectEqual(@as(?[]const u8, null), manifest.package.?.version);
    try std.testing.expectEqual(@as(?[]const u8, null), manifest.package.?.license);
}

test "manifest parser parses package version and license" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\version "0.1.0";
        \\license "MIT";
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("Compiler", manifest.package.?.name);
    try std.testing.expectEqualStrings("0.1.0", manifest.package.?.version.?);
    try std.testing.expectEqualStrings("MIT", manifest.package.?.license.?);
}

test "manifest parser stores sections in order" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main;
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), manifest.sections.len);
    try std.testing.expectEqualStrings("package", manifest.sections[0].name);
    try std.testing.expectEqualStrings("modules", manifest.sections[1].name);
}

test "manifest parser represents registry header and section" {
    const text =
        \\# kind: registry
        \\# format: concept-registry-v0
        \\
        \\=== registry ===
        \\name Yuechen;
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(ManifestKind.registry, manifest.kind);
    try std.testing.expectEqual(ManifestFormat.concept_registry_v0, manifest.format);
    try std.testing.expectEqual(@as(?PackageInfo, null), manifest.package);
    try std.testing.expectEqualStrings("registry", manifest.sections[0].name);
}

test "manifest parser owns package fields" {
    const text = try std.testing.allocator.dupe(u8, minimal_package_manifest);
    const manifest = try parse(std.testing.allocator, text);
    std.testing.allocator.free(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("Compiler", manifest.package.?.name);
}

test "manifest parser rejects missing kind" {
    const text =
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
    ;
    try std.testing.expectError(error.MissingKind, expectManifest(text));
}

test "manifest parser rejects missing format" {
    const text =
        \\# kind: package
        \\
        \\=== package ===
        \\package Compiler;
    ;
    try std.testing.expectError(error.MissingFormat, expectManifest(text));
}

test "manifest parser rejects invalid kind" {
    const text =
        \\# kind: workspace
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
    ;
    try std.testing.expectError(error.InvalidKind, expectManifest(text));
}

test "manifest parser rejects invalid format" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v1
        \\
        \\=== package ===
        \\package Compiler;
    ;
    try std.testing.expectError(error.InvalidFormat, expectManifest(text));
}

test "manifest parser rejects kind and format mismatch" {
    const text =
        \\# kind: package
        \\# format: concept-registry-v0
        \\
        \\=== package ===
        \\package Compiler;
    ;
    try std.testing.expectError(error.ManifestKindFormatMismatch, expectManifest(text));
}

test "manifest parser rejects missing package section" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== modules ===
        \\module Compiler.Main;
    ;
    try std.testing.expectError(error.MissingPackageSection, expectManifest(text));
}

test "manifest parser rejects missing package name" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\version "0.1.0";
    ;
    try std.testing.expectError(error.MissingPackageName, expectManifest(text));
}

test "manifest parser rejects duplicate section" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== package ===
        \\package Other;
    ;
    try std.testing.expectError(error.DuplicateSection, expectManifest(text));
}

test "manifest parser rejects duplicate package name" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\package Other;
    ;
    try std.testing.expectError(error.DuplicatePackageName, expectManifest(text));
}

test "manifest parser rejects duplicate version" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\version "0.1.0";
        \\version "0.2.0";
    ;
    try std.testing.expectError(error.DuplicateVersion, expectManifest(text));
}

test "manifest parser rejects duplicate license" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\license "MIT";
        \\license "Apache-2.0";
    ;
    try std.testing.expectError(error.DuplicateLicense, expectManifest(text));
}

test "manifest parser rejects invalid package statement" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\author "Yuechen";
    ;
    try std.testing.expectError(error.InvalidPackageStatement, expectManifest(text));
}

test "manifest parser rejects invalid string literal" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\version "0.1.0;
    ;
    try std.testing.expectError(error.InvalidStringLiteral, expectManifest(text));
}

test "manifest parser rejects missing semicolon" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler
    ;
    try std.testing.expectError(error.ExpectedSemicolon, expectManifest(text));
}
