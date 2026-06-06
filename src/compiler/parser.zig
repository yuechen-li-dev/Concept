const std = @import("std");

const ast_model = @import("ast.zig");
const diagnostics_model = @import("diagnostics.zig");
const lexer_model = @import("lexer.zig");
const source_model = @import("source.zig");
const token_model = @import("token.zig");

pub const ast = ast_model;
pub const DiagnosticBag = diagnostics_model.DiagnosticBag;
pub const DiagnosticCode = diagnostics_model.DiagnosticCode;
pub const SourceFile = source_model.SourceFile;
pub const SourceSpan = source_model.SourceSpan;
pub const Token = token_model.Token;
pub const TokenKind = token_model.TokenKind;

pub const Parser = struct {
    tokens: []const Token,
    diagnostics: *DiagnosticBag,
    index: usize = 0,

    pub fn init(tokens: []const Token, diagnostics: *DiagnosticBag) Parser {
        return .{
            .tokens = tokens,
            .diagnostics = diagnostics,
        };
    }

    pub fn parseCompilationUnit(self: *Parser, allocator: std.mem.Allocator) !ast.CompilationUnit {
        const start = self.current().span.start;
        var module_decl: ?ast.ModuleDecl = null;
        var imports = std.ArrayList(ast.ImportDecl).init(allocator);
        var items = std.ArrayList(ast.Item).init(allocator);
        errdefer {
            if (module_decl) |module| module.deinit(allocator);
            for (imports.items) |import_decl| import_decl.deinit(allocator);
            imports.deinit();
            items.deinit();
        }

        if (self.current().kind == .module) {
            module_decl = try self.parseModuleDecl(allocator);
        } else {
            try self.reportExpectedItem("expected module declaration at start of file", self.current().span);
        }

        while (self.current().kind == .import) {
            try imports.append(try self.parseImportDecl(allocator));
        }

        while (self.current().kind != .eof) {
            switch (self.current().kind) {
                .module => {
                    try self.reportDuplicateModule();
                    const duplicate = try self.parseModuleDecl(allocator);
                    duplicate.deinit(allocator);
                },
                .import => {
                    // M6 only accepts imports in the initial import block. Once
                    // future item parsing exists this can become an item-ordering
                    // diagnostic instead of treating the token as unexpected.
                    try self.reportExpectedItem("expected item declaration", self.current().span);
                    self.advance();
                },
                else => {
                    try self.reportExpectedItem("expected item declaration", self.current().span);
                    self.advance();
                },
            }
        }

        const eof_span = self.current().span;
        const owned_imports = try imports.toOwnedSlice();
        errdefer {
            for (owned_imports) |import_decl| import_decl.deinit(allocator);
            allocator.free(owned_imports);
        }

        return .{
            .span = ast.spanFromBounds(start, eof_span.start),
            .module = module_decl,
            .imports = owned_imports,
            .items = try items.toOwnedSlice(),
        };
    }

    pub fn current(self: Parser) Token {
        return self.peek(0);
    }

    pub fn peek(self: Parser, offset: usize) Token {
        if (self.tokens.len == 0) {
            return Token.init(.eof, .{ .start = 0, .length = 0 }, "");
        }
        const peek_index = self.index + offset;
        if (peek_index >= self.tokens.len) return self.tokens[self.tokens.len - 1];
        return self.tokens[peek_index];
    }

    pub fn advance(self: *Parser) Token {
        const token = self.current();
        if (self.current().kind != .eof) self.index += 1;
        return token;
    }

    pub fn match(self: *Parser, kind: TokenKind) ?Token {
        if (self.current().kind != kind) return null;
        return self.advance();
    }

    pub fn consume(self: *Parser, kind: TokenKind) ?Token {
        return self.match(kind);
    }

    pub fn expect(self: *Parser, kind: TokenKind, message: []const u8, code: DiagnosticCode) !?Token {
        if (self.current().kind == kind) return self.advance();

        try self.report(code, message, self.current().span);
        return null;
    }

    fn parseModuleDecl(self: *Parser, allocator: std.mem.Allocator) !ast.ModuleDecl {
        const module_token = self.advance();
        var name = try self.parseDottedName(allocator);
        errdefer name.deinit(allocator);
        const end_span = try self.expectSemicolon("expected ';' after module declaration");
        return .{
            .name = name,
            .span = ast.spanFromBounds(module_token.span.start, spanEnd(end_span)),
        };
    }

    fn parseImportDecl(self: *Parser, allocator: std.mem.Allocator) !ast.ImportDecl {
        const import_token = self.advance();
        var name = try self.parseDottedName(allocator);
        errdefer name.deinit(allocator);
        const end_span = try self.expectSemicolon("expected ';' after import declaration");
        return .{
            .name = name,
            .span = ast.spanFromBounds(import_token.span.start, spanEnd(end_span)),
        };
    }

    fn parseDottedName(self: *Parser, allocator: std.mem.Allocator) !ast.QualifiedName {
        var parts = std.ArrayList(ast.NameSegment).init(allocator);
        errdefer parts.deinit();

        const start_span = self.current().span;
        const first = try self.expect(.identifier, "expected identifier in dotted name", .UnexpectedToken);
        if (first) |identifier| {
            try parts.append(.{ .text = identifier.lexeme, .span = identifier.span });
        } else {
            return .{
                .parts = try parts.toOwnedSlice(),
                .span = start_span,
            };
        }

        var last_span = parts.items[parts.items.len - 1].span;
        while (self.current().kind == .dot) {
            _ = self.advance();
            const part = try self.expect(.identifier, "expected identifier after '.' in dotted name", .UnexpectedToken);
            if (part) |identifier| {
                try parts.append(.{ .text = identifier.lexeme, .span = identifier.span });
                last_span = identifier.span;
            } else {
                break;
            }
        }

        return .{
            .parts = try parts.toOwnedSlice(),
            .span = ast.spanFromBounds(start_span.start, spanEnd(last_span)),
        };
    }

    fn expectSemicolon(self: *Parser, message: []const u8) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;

        const diagnostic_span = self.current().span;
        try self.report(.UnexpectedToken, message, diagnostic_span);

        while (self.current().kind != .eof) {
            switch (self.current().kind) {
                .semicolon => return self.advance().span,
                .module, .import => return diagnostic_span,
                else => _ = self.advance(),
            }
        }

        return diagnostic_span;
    }

    fn reportDuplicateModule(self: *Parser) !void {
        try self.report(.DuplicateModuleDeclaration, "duplicate module declaration", self.current().span);
    }

    fn reportExpectedItem(self: *Parser, message: []const u8, span: SourceSpan) !void {
        try self.report(.ExpectedItem, message, span);
    }

    fn report(self: *Parser, code: DiagnosticCode, message: []const u8, span: SourceSpan) !void {
        try self.diagnostics.append(diagnostics_model.makeDiagnostic(code, .@"error", message, span));
    }
};

pub fn parseTokens(allocator: std.mem.Allocator, tokens: []const Token, diagnostics: *DiagnosticBag) !ast.CompilationUnit {
    var parser = Parser.init(tokens, diagnostics);
    return parser.parseCompilationUnit(allocator);
}

pub fn parseSource(allocator: std.mem.Allocator, source_file: SourceFile, diagnostics: *DiagnosticBag) !ast.CompilationUnit {
    const tokens = try lexer_model.lexAll(allocator, source_file, diagnostics);
    defer allocator.free(tokens);
    return parseTokens(allocator, tokens, diagnostics);
}

fn spanEnd(span: SourceSpan) usize {
    return span.end() orelse span.start;
}

fn parseTestSource(source_text: []const u8, diagnostics: *DiagnosticBag) !ast.CompilationUnit {
    const source_file = try SourceFile.init(std.testing.allocator, "test.con", source_text);
    defer source_file.deinit(std.testing.allocator);
    return parseSource(std.testing.allocator, source_file, diagnostics);
}

fn expectSingleDiagnostic(source_text: []const u8, code: DiagnosticCode) !void {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(source_text, &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(code, diagnostics.diagnostics.items[0].code);
}

test "parses module declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example;", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expect(unit.module != null);
    try std.testing.expectEqualStrings("Example", unit.module.?.name.parts[0].text);
    try std.testing.expectEqual(@as(usize, 7), unit.module.?.span.start);
    try std.testing.expectEqual(@as(usize, 7), unit.module.?.name.span.start);
}

test "parses dotted module name" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example.Compiler;", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 2), unit.module.?.name.parts.len);
    try std.testing.expectEqualStrings("Example", unit.module.?.name.parts[0].text);
    try std.testing.expectEqualStrings("Compiler", unit.module.?.name.parts[1].text);
}

test "parses one import" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; import Core.Memory;", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 1), unit.imports.len);
    try std.testing.expectEqualStrings("Core", unit.imports[0].name.parts[0].text);
    try std.testing.expectEqualStrings("Memory", unit.imports[0].name.parts[1].text);
}

test "parses multiple imports" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example.Compiler;
        \\import Core.Memory;
        \\import Core.Diagnostics;
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 2), unit.imports.len);
    try std.testing.expectEqualStrings("Diagnostics", unit.imports[1].name.parts[1].text);
}

test "missing module declaration produces diagnostic" {
    try expectSingleDiagnostic("import Core.Memory;", .ExpectedItem);
}

test "missing semicolon after module produces diagnostic" {
    try expectSingleDiagnostic("module Example import Core.Memory;", .UnexpectedToken);
}

test "malformed import name produces diagnostic for trailing dot" {
    try expectSingleDiagnostic("module Example; import Core.;", .UnexpectedToken);
}

test "malformed import name produces diagnostic for missing name" {
    try expectSingleDiagnostic("module Example; import ;", .UnexpectedToken);
}

test "duplicate module declaration produces diagnostic" {
    try expectSingleDiagnostic("module Example; module Other;", .DuplicateModuleDeclaration);
}

test "AST snapshot debug output is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example.Compiler;
        \\import Core.Memory;
        \\import Core.Diagnostics;
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example.Compiler
        \\  Import Core.Memory
        \\  Import Core.Diagnostics
        \\
    , snapshot);
}
