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

pub const ModuleDecl = struct {
    name: []const u8,
    root: []const u8,

    pub fn deinit(self: ModuleDecl, allocator: std.mem.Allocator) void {
        allocator.free(self.name);
        allocator.free(self.root);
    }
};

pub const TargetKind = enum {
    executable,
    library,
    tests,

    fn parse(value: []const u8) ?TargetKind {
        if (std.mem.eql(u8, value, "executable")) return .executable;
        if (std.mem.eql(u8, value, "library")) return .library;
        if (std.mem.eql(u8, value, "tests")) return .tests;
        return null;
    }
};

pub const TargetDecl = struct {
    name: []const u8,
    kind: TargetKind,
    module: ?[]const u8 = null,
    modules: []const []const u8,

    pub fn deinit(self: TargetDecl, allocator: std.mem.Allocator) void {
        allocator.free(self.name);
        if (self.module) |module| allocator.free(module);
        for (self.modules) |module| allocator.free(module);
        allocator.free(self.modules);
    }
};

pub const Manifest = struct {
    kind: ManifestKind,
    format: ManifestFormat,
    package: ?PackageInfo,
    modules: []ModuleDecl,
    targets: []TargetDecl,
    sections: []Section,

    pub fn deinit(self: Manifest, allocator: std.mem.Allocator) void {
        if (self.package) |package| package.deinit(allocator);
        for (self.modules) |module| module.deinit(allocator);
        allocator.free(self.modules);
        for (self.targets) |target| target.deinit(allocator);
        allocator.free(self.targets);
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
        .modules = &.{},
        .targets = &.{},
        .sections = owned_sections,
    };
    errdefer manifest.deinit(allocator);

    switch (manifest.kind) {
        .package => {
            const package_section = manifest.section("package") orelse return error.MissingPackageSection;
            manifest.package = try parsePackageSection(allocator, package_section.body);
            if (manifest.section("modules")) |modules_section| {
                manifest.modules = try parseModulesSection(allocator, modules_section.body);
            }
            if (manifest.section("targets")) |targets_section| {
                manifest.targets = try parseTargetsSection(allocator, targets_section.body, manifest.modules);
            }
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

fn parseModulesSection(allocator: std.mem.Allocator, body: []const u8) ![]ModuleDecl {
    var parser = ManifestBodyParser{ .text = body };
    var modules = std.ArrayList(ModuleDecl).empty;
    errdefer {
        for (modules.items) |module| module.deinit(allocator);
        modules.deinit(allocator);
    }

    while (true) {
        parser.skipWhitespace();
        if (parser.atEnd()) break;
        try parser.expectKeyword("module", error.InvalidModuleDeclaration);
        parser.skipWhitespace();
        const name_source = try parser.readDottedName();
        for (modules.items) |existing| {
            if (std.mem.eql(u8, existing.name, name_source)) return error.DuplicateModule;
        }
        const name = try allocator.dupe(u8, name_source);
        errdefer allocator.free(name);

        parser.skipWhitespace();
        try parser.expectByte('{', error.InvalidModuleDeclaration);
        var root: ?[]const u8 = null;
        errdefer if (root) |owned_root| allocator.free(owned_root);

        while (true) {
            parser.skipWhitespace();
            if (parser.consumeByte('}')) break;
            try parser.expectKeyword("root", error.InvalidModuleDeclaration);
            if (root != null) return error.DuplicateModuleRoot;
            parser.skipWhitespace();
            root = try parser.readStringLiteral(allocator);
            parser.skipWhitespace();
            try parser.expectByte(';', error.ExpectedSemicolon);
        }

        try modules.append(allocator, .{
            .name = name,
            .root = root orelse return error.MissingModuleRoot,
        });
    }

    return modules.toOwnedSlice(allocator);
}

fn parseTargetsSection(allocator: std.mem.Allocator, body: []const u8, modules: []const ModuleDecl) ![]TargetDecl {
    var parser = ManifestBodyParser{ .text = body };
    var targets = std.ArrayList(TargetDecl).empty;
    errdefer {
        for (targets.items) |target| target.deinit(allocator);
        targets.deinit(allocator);
    }

    while (true) {
        parser.skipWhitespace();
        if (parser.atEnd()) break;
        try parser.expectKeyword("target", error.InvalidTargetDeclaration);
        parser.skipWhitespace();
        const name_source = try parser.readIdentifier(error.InvalidTargetDeclaration);
        for (targets.items) |existing| {
            if (std.mem.eql(u8, existing.name, name_source)) return error.DuplicateTarget;
        }
        const name = try allocator.dupe(u8, name_source);
        errdefer allocator.free(name);

        parser.skipWhitespace();
        try parser.expectByte('{', error.InvalidTargetDeclaration);
        var kind: ?TargetKind = null;
        var module: ?[]const u8 = null;
        var module_list: ?std.ArrayList([]const u8) = null;
        errdefer {
            if (module) |owned_module| allocator.free(owned_module);
            if (module_list) |*list| {
                for (list.items) |owned_module| allocator.free(owned_module);
                list.deinit(allocator);
            }
        }

        while (true) {
            parser.skipWhitespace();
            if (parser.consumeByte('}')) break;
            const statement = try parser.readIdentifier(error.InvalidTargetDeclaration);
            parser.skipWhitespace();

            if (std.mem.eql(u8, statement, "kind")) {
                if (kind != null) return error.DuplicateTargetKind;
                const kind_name = try parser.readIdentifier(error.InvalidTargetKind);
                kind = TargetKind.parse(kind_name) orelse return error.InvalidTargetKind;
                parser.skipWhitespace();
                try parser.expectByte(';', error.ExpectedSemicolon);
            } else if (std.mem.eql(u8, statement, "module")) {
                if (module != null or module_list != null) return error.DuplicateTargetModule;
                const module_source = try parser.readDottedName();
                module = try allocator.dupe(u8, module_source);
                parser.skipWhitespace();
                try parser.expectByte(';', error.ExpectedSemicolon);
            } else if (std.mem.eql(u8, statement, "modules")) {
                if (module != null or module_list != null) return error.DuplicateTargetModule;
                module_list = try parser.readModuleList(allocator);
                parser.skipWhitespace();
                try parser.expectByte(';', error.ExpectedSemicolon);
            } else {
                return error.InvalidTargetDeclaration;
            }
        }

        const target_kind = kind orelse return error.MissingTargetKind;
        switch (target_kind) {
            .executable, .library => {
                if (module == null or module_list != null) return error.MissingTargetModule;
                try validateKnownModule(module.?, modules);
                try targets.append(allocator, .{
                    .name = name,
                    .kind = target_kind,
                    .module = module,
                    .modules = &.{},
                });
                module = null;
            },
            .tests => {
                if (module != null or module_list == null) return error.MissingTargetModule;
                if (module_list.?.items.len == 0) return error.InvalidModuleList;
                for (module_list.?.items) |module_name| try validateKnownModule(module_name, modules);
                const owned_modules = try module_list.?.toOwnedSlice(allocator);
                errdefer {
                    for (owned_modules) |owned_module| allocator.free(owned_module);
                    allocator.free(owned_modules);
                }
                module_list = null;
                try targets.append(allocator, .{
                    .name = name,
                    .kind = target_kind,
                    .module = null,
                    .modules = owned_modules,
                });
            },
        }
    }

    return targets.toOwnedSlice(allocator);
}

fn validateKnownModule(name: []const u8, modules: []const ModuleDecl) !void {
    for (modules) |module| {
        if (std.mem.eql(u8, module.name, name)) return;
    }
    return error.UnknownTargetModule;
}

const ManifestBodyParser = struct {
    text: []const u8,
    cursor: usize = 0,

    fn atEnd(self: ManifestBodyParser) bool {
        return self.cursor >= self.text.len;
    }

    fn skipWhitespace(self: *ManifestBodyParser) void {
        while (!self.atEnd()) {
            switch (self.text[self.cursor]) {
                ' ', '\t', '\r', '\n' => self.cursor += 1,
                else => return,
            }
        }
    }

    fn consumeByte(self: *ManifestBodyParser, byte: u8) bool {
        if (!self.atEnd() and self.text[self.cursor] == byte) {
            self.cursor += 1;
            return true;
        }
        return false;
    }

    fn expectByte(self: *ManifestBodyParser, byte: u8, parse_error: anyerror) !void {
        if (!self.consumeByte(byte)) return parse_error;
    }

    fn expectKeyword(self: *ManifestBodyParser, keyword: []const u8, parse_error: anyerror) !void {
        const identifier = try self.readIdentifier(parse_error);
        if (!std.mem.eql(u8, identifier, keyword)) return parse_error;
    }

    fn readIdentifier(self: *ManifestBodyParser, parse_error: anyerror) ![]const u8 {
        if (self.atEnd() or !isIdentifierStart(self.text[self.cursor])) return parse_error;
        const start = self.cursor;
        self.cursor += 1;
        while (!self.atEnd() and isIdentifierContinue(self.text[self.cursor])) self.cursor += 1;
        return self.text[start..self.cursor];
    }

    fn readDottedName(self: *ManifestBodyParser) ![]const u8 {
        const start = self.cursor;
        _ = try self.readIdentifier(error.InvalidDottedName);
        while (self.consumeByte('.')) {
            _ = try self.readIdentifier(error.InvalidDottedName);
        }
        if (!self.atEnd() and !isDottedNameDelimiter(self.text[self.cursor])) return error.InvalidDottedName;
        return self.text[start..self.cursor];
    }

    fn readStringLiteral(self: *ManifestBodyParser, allocator: std.mem.Allocator) ![]const u8 {
        if (self.atEnd() or self.text[self.cursor] != '"') return error.InvalidStringLiteral;
        const start = self.cursor;
        self.cursor += 1;
        while (!self.atEnd() and self.text[self.cursor] != '"') {
            if (self.text[self.cursor] == '\\' or self.text[self.cursor] == '\n' or self.text[self.cursor] == '\r') return error.InvalidStringLiteral;
            self.cursor += 1;
        }
        if (self.atEnd()) return error.InvalidStringLiteral;
        self.cursor += 1;
        return parseStringLiteral(allocator, self.text[start..self.cursor]);
    }

    fn readModuleList(self: *ManifestBodyParser, allocator: std.mem.Allocator) !std.ArrayList([]const u8) {
        var modules = std.ArrayList([]const u8).empty;
        errdefer {
            for (modules.items) |module| allocator.free(module);
            modules.deinit(allocator);
        }

        self.skipWhitespace();
        try self.expectByte('[', error.InvalidModuleList);
        while (true) {
            self.skipWhitespace();
            if (self.consumeByte(']')) break;
            const module_source = self.readDottedName() catch return error.InvalidModuleList;
            for (modules.items) |existing| {
                if (std.mem.eql(u8, existing, module_source)) return error.DuplicateTargetModule;
            }
            try modules.append(allocator, try allocator.dupe(u8, module_source));
            self.skipWhitespace();
            if (self.consumeByte(',')) continue;
            if (self.consumeByte(']')) break;
            return error.InvalidModuleList;
        }

        return modules;
    }
};

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

fn isDottedNameDelimiter(byte: u8) bool {
    return switch (byte) {
        ' ', '\t', '\r', '\n', ';', '{', '}', '[', ']', ',' => true,
        else => false,
    };
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
        \\module Compiler.Main {
        \\    root "src/compiler/main";
        \\}
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

test "manifest parser parses one module" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root "src/compiler/source";
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), manifest.modules.len);
    try std.testing.expectEqualStrings("Compiler.Source", manifest.modules[0].name);
    try std.testing.expectEqualStrings("src/compiler/source", manifest.modules[0].root);
}

test "manifest parser parses dotted module name" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source.Tests {
        \\    root "src/compiler/source/tests";
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqualStrings("Compiler.Source.Tests", manifest.modules[0].name);
}

test "manifest parser parses multiple modules" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root "src/compiler/source";
        \\}
        \\
        \\module Compiler.Main {
        \\    root "src/compiler/main";
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), manifest.modules.len);
    try std.testing.expectEqualStrings("Compiler.Source", manifest.modules[0].name);
    try std.testing.expectEqualStrings("Compiler.Main", manifest.modules[1].name);
}

test "manifest parser parses executable target with module" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main {
        \\    root "src/compiler/main";
        \\}
        \\
        \\=== targets ===
        \\target Cathedral {
        \\    kind executable;
        \\    module Compiler.Main;
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), manifest.targets.len);
    try std.testing.expectEqualStrings("Cathedral", manifest.targets[0].name);
    try std.testing.expectEqual(TargetKind.executable, manifest.targets[0].kind);
    try std.testing.expectEqualStrings("Compiler.Main", manifest.targets[0].module.?);
}

test "manifest parser parses library target with module" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root "src/compiler/source";
        \\}
        \\
        \\=== targets ===
        \\target CompilerLib {
        \\    kind library;
        \\    module Compiler.Source;
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(TargetKind.library, manifest.targets[0].kind);
    try std.testing.expectEqualStrings("Compiler.Source", manifest.targets[0].module.?);
}

test "manifest parser parses tests target with module list" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source.Tests {
        \\    root "src/compiler/source/tests";
        \\}
        \\module Compiler.Parser.Tests {
        \\    root "src/compiler/parser/tests";
        \\}
        \\
        \\=== targets ===
        \\target CompilerTests {
        \\    kind tests;
        \\    modules [
        \\        Compiler.Source.Tests,
        \\        Compiler.Parser.Tests
        \\    ];
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(TargetKind.tests, manifest.targets[0].kind);
    try std.testing.expectEqual(@as(?[]const u8, null), manifest.targets[0].module);
    try std.testing.expectEqual(@as(usize, 2), manifest.targets[0].modules.len);
    try std.testing.expectEqualStrings("Compiler.Source.Tests", manifest.targets[0].modules[0]);
    try std.testing.expectEqualStrings("Compiler.Parser.Tests", manifest.targets[0].modules[1]);
}

test "manifest parser parses combined package modules and targets manifest" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\version "0.1.0";
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root "src/compiler/source";
        \\}
        \\module Compiler.Main {
        \\    root "src/compiler/main";
        \\}
        \\module Compiler.Source.Tests {
        \\    root "src/compiler/source/tests";
        \\}
        \\
        \\=== targets ===
        \\target Cathedral {
        \\    kind executable;
        \\    module Compiler.Main;
        \\}
        \\target CompilerTests {
        \\    kind tests;
        \\    modules [
        \\        Compiler.Source.Tests,
        \\    ];
        \\}
    ;
    const manifest = try expectManifest(text);
    defer manifest.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 3), manifest.modules.len);
    try std.testing.expectEqual(@as(usize, 2), manifest.targets.len);
    try std.testing.expectEqualStrings("0.1.0", manifest.package.?.version.?);
}

test "manifest parser rejects duplicate module name" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source { root "src/compiler/source"; }
        \\module Compiler.Source { root "src/compiler/source2"; }
    ;
    try std.testing.expectError(error.DuplicateModule, expectManifest(text));
}

test "manifest parser rejects missing module root" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\}
    ;
    try std.testing.expectError(error.MissingModuleRoot, expectManifest(text));
}

test "manifest parser rejects duplicate module root" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root "src/compiler/source";
        \\    root "src/compiler/other";
        \\}
    ;
    try std.testing.expectError(error.DuplicateModuleRoot, expectManifest(text));
}

test "manifest parser rejects invalid dotted module name" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler..Source {
        \\    root "src/compiler/source";
        \\}
    ;
    try std.testing.expectError(error.InvalidDottedName, expectManifest(text));
}

test "manifest parser rejects unknown module-section statement" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    source "src/compiler/source";
        \\}
    ;
    try std.testing.expectError(error.InvalidModuleDeclaration, expectManifest(text));
}

test "manifest parser rejects invalid module root string literal" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root src/compiler/source;
        \\}
    ;
    try std.testing.expectError(error.InvalidStringLiteral, expectManifest(text));
}

test "manifest parser rejects missing module root semicolon" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source {
        \\    root "src/compiler/source"
        \\}
    ;
    try std.testing.expectError(error.ExpectedSemicolon, expectManifest(text));
}

test "manifest parser rejects duplicate target name" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral { kind executable; module Compiler.Main; }
        \\target Cathedral { kind executable; module Compiler.Main; }
    ;
    try std.testing.expectError(error.DuplicateTarget, expectManifest(text));
}

test "manifest parser rejects missing target kind" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral { module Compiler.Main; }
    ;
    try std.testing.expectError(error.MissingTargetKind, expectManifest(text));
}

test "manifest parser rejects invalid target kind" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral { kind binary; module Compiler.Main; }
    ;
    try std.testing.expectError(error.InvalidTargetKind, expectManifest(text));
}

test "manifest parser rejects missing module for executable target" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== targets ===
        \\target Cathedral { kind executable; }
    ;
    try std.testing.expectError(error.MissingTargetModule, expectManifest(text));
}

test "manifest parser rejects missing modules list for tests target" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== targets ===
        \\target CompilerTests { kind tests; }
    ;
    try std.testing.expectError(error.MissingTargetModule, expectManifest(text));
}

test "manifest parser rejects duplicate target kind" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral {
        \\    kind executable;
        \\    kind library;
        \\    module Compiler.Main;
        \\}
    ;
    try std.testing.expectError(error.DuplicateTargetKind, expectManifest(text));
}

test "manifest parser rejects duplicate target module field" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral {
        \\    kind executable;
        \\    module Compiler.Main;
        \\    module Compiler.Main;
        \\}
    ;
    try std.testing.expectError(error.DuplicateTargetModule, expectManifest(text));
}

test "manifest parser rejects invalid target module list" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source.Tests { root "src/compiler/source/tests"; }
        \\
        \\=== targets ===
        \\target CompilerTests {
        \\    kind tests;
        \\    modules [Compiler.Source.Tests Compiler.Parser.Tests];
        \\}
    ;
    try std.testing.expectError(error.InvalidModuleList, expectManifest(text));
}

test "manifest parser rejects unknown referenced target module" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Source { root "src/compiler/source"; }
        \\
        \\=== targets ===
        \\target Cathedral {
        \\    kind executable;
        \\    module Compiler.Main;
        \\}
    ;
    try std.testing.expectError(error.UnknownTargetModule, expectManifest(text));
}

test "manifest parser rejects unknown target-section statement" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral { kind executable; entry Compiler.Main; }
    ;
    try std.testing.expectError(error.InvalidTargetDeclaration, expectManifest(text));
}

test "manifest parser rejects missing target semicolon" {
    const text =
        \\# kind: package
        \\# format: concept-manifest-v0
        \\
        \\=== package ===
        \\package Compiler;
        \\
        \\=== modules ===
        \\module Compiler.Main { root "src/compiler/main"; }
        \\
        \\=== targets ===
        \\target Cathedral {
        \\    kind executable
        \\    module Compiler.Main;
        \\}
    ;
    try std.testing.expectError(error.ExpectedSemicolon, expectManifest(text));
}
