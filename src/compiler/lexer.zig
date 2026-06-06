const std = @import("std");

const diagnostics_model = @import("diagnostics.zig");
const source_model = @import("source.zig");
const token_model = @import("token.zig");

pub const DiagnosticBag = diagnostics_model.DiagnosticBag;
pub const SourceFile = source_model.SourceFile;
pub const SourceSpan = source_model.SourceSpan;
pub const Token = token_model.Token;
pub const TokenKind = token_model.TokenKind;

pub const Lexer = struct {
    source_file: SourceFile,
    diagnostics: *DiagnosticBag,
    offset: usize = 0,

    pub fn init(source_file: SourceFile, diagnostics: *DiagnosticBag) Lexer {
        return .{
            .source_file = source_file,
            .diagnostics = diagnostics,
        };
    }

    pub fn nextToken(self: *Lexer) !Token {
        try self.skipTrivia();

        const start = self.offset;
        if (self.isAtEnd()) {
            return self.makeToken(.eof, start, start);
        }

        if (isIdentifierStart(self.peek().?)) {
            return self.scanIdentifier();
        }

        if (self.matchPunctuation()) |punctuation| {
            return punctuation;
        }

        // Recovery for lexer v1 is intentionally simple: report the invalid
        // byte, consume exactly that byte, and return an invalid token so
        // callers can keep token stream position aligned with diagnostics.
        const invalid_byte = self.advance().?;
        const invalid_span = SourceSpan{ .start = start, .length = 1 };
        try self.diagnostics.append(try diagnostics_model.invalidCharacter(
            self.diagnostics.diagnostics.allocator,
            invalid_span,
            invalid_byte,
        ));
        return Token.init(.invalid, invalid_span, self.source_file.text[start..self.offset]);
    }

    fn skipTrivia(self: *Lexer) !void {
        while (!self.isAtEnd()) {
            if (isWhitespace(self.peek().?)) {
                _ = self.advance();
                continue;
            }

            if (self.startsWith("//")) {
                self.offset += 2;
                while (!self.isAtEnd() and self.peek().? != '\n') {
                    _ = self.advance();
                }
                continue;
            }

            if (self.startsWith("/*")) {
                try self.skipBlockComment();
                continue;
            }

            return;
        }
    }

    fn skipBlockComment(self: *Lexer) !void {
        const start = self.offset;
        self.offset += 2;

        while (!self.isAtEnd()) {
            if (self.startsWith("*/")) {
                self.offset += 2;
                return;
            }
            _ = self.advance();
        }

        try self.diagnostics.append(diagnostics_model.unterminatedBlockComment(.{
            .start = start,
            .length = self.source_file.len() - start,
        }));
    }

    fn scanIdentifier(self: *Lexer) Token {
        const start = self.offset;
        _ = self.advance();

        while (!self.isAtEnd() and isIdentifierContinue(self.peek().?)) {
            _ = self.advance();
        }

        const lexeme = self.source_file.text[start..self.offset];
        const kind = token_model.keywordKind(lexeme) orelse .identifier;
        return Token.init(kind, .{ .start = start, .length = self.offset - start }, lexeme);
    }

    fn matchPunctuation(self: *Lexer) ?Token {
        const two_char_kinds = [_]struct { lexeme: []const u8, kind: TokenKind }{
            .{ .lexeme = "::", .kind = .colon_colon },
            .{ .lexeme = "->", .kind = .arrow },
            .{ .lexeme = "==", .kind = .equal_equal },
            .{ .lexeme = "!=", .kind = .bang_equal },
            .{ .lexeme = "<=", .kind = .less_equal },
            .{ .lexeme = ">=", .kind = .greater_equal },
            .{ .lexeme = "&&", .kind = .ampersand_ampersand },
            .{ .lexeme = "||", .kind = .pipe_pipe },
        };

        const start = self.offset;
        for (two_char_kinds) |case| {
            if (self.startsWith(case.lexeme)) {
                self.offset += case.lexeme.len;
                return self.makeToken(case.kind, start, self.offset);
            }
        }

        const kind: TokenKind = switch (self.peek().?) {
            '(' => .left_paren,
            ')' => .right_paren,
            '{' => .left_brace,
            '}' => .right_brace,
            '[' => .left_bracket,
            ']' => .right_bracket,
            '<' => .less,
            '>' => .greater,
            ';' => .semicolon,
            ',' => .comma,
            '.' => .dot,
            ':' => .colon,
            '&' => .ampersand,
            '+' => .plus,
            '-' => .minus,
            '*' => .star,
            '/' => .slash,
            '%' => .percent,
            '=' => .equal,
            '!' => .bang,
            else => return null,
        };
        _ = self.advance();
        return self.makeToken(kind, start, self.offset);
    }

    fn makeToken(self: Lexer, kind: TokenKind, start: usize, end: usize) Token {
        return Token.init(kind, .{ .start = start, .length = end - start }, self.source_file.text[start..end]);
    }

    fn startsWith(self: Lexer, text: []const u8) bool {
        return std.mem.startsWith(u8, self.source_file.text[self.offset..], text);
    }

    fn isAtEnd(self: Lexer) bool {
        return self.offset >= self.source_file.text.len;
    }

    fn peek(self: Lexer) ?u8 {
        if (self.isAtEnd()) return null;
        return self.source_file.text[self.offset];
    }

    fn advance(self: *Lexer) ?u8 {
        if (self.isAtEnd()) return null;
        const byte = self.source_file.text[self.offset];
        self.offset += 1;
        return byte;
    }
};

pub fn lexAll(allocator: std.mem.Allocator, source_file: SourceFile, diagnostics: *DiagnosticBag) ![]Token {
    var lexer = Lexer.init(source_file, diagnostics);
    var tokens = std.ArrayList(Token).init(allocator);
    errdefer tokens.deinit();

    while (true) {
        const token = try lexer.nextToken();
        try tokens.append(token);
        if (token.kind == .eof) break;
    }

    return tokens.toOwnedSlice();
}

fn isWhitespace(byte: u8) bool {
    return switch (byte) {
        ' ', '\t', '\n', '\r' => true,
        else => false,
    };
}

fn isIdentifierStart(byte: u8) bool {
    return std.ascii.isAlphabetic(byte) or byte == '_';
}

fn isIdentifierContinue(byte: u8) bool {
    return isIdentifierStart(byte) or std.ascii.isDigit(byte);
}

fn expectLexedKinds(source_text: []const u8, expected: []const TokenKind) !void {
    const source_file = try SourceFile.init(std.testing.allocator, "test.con", source_text);
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(expected.len, tokens.len);
    for (expected, tokens) |expected_kind, actual| {
        try std.testing.expectEqual(expected_kind, actual.kind);
    }
}

test "empty source produces EOF" {
    const source_file = try SourceFile.init(std.testing.allocator, "empty.con", "");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    var lexer = Lexer.init(source_file, &diagnostics);
    const eof = try lexer.nextToken();

    try std.testing.expectEqual(TokenKind.eof, eof.kind);
    try std.testing.expectEqual(@as(usize, 0), eof.span.start);
    try std.testing.expectEqual(@as(usize, 0), eof.span.length);
    try std.testing.expectEqualStrings("", eof.lexeme);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
}

test "module declaration tokenizes correctly" {
    const expected = [_]TokenKind{ .module, .identifier, .semicolon, .eof };
    try expectLexedKinds("module Test;", &expected);
}

test "keywords tokenize as keyword kinds" {
    const source_text = "module import export struct enum concept interface impl marker unsafe mut const return match when if else while for machine state transition yield run true false";
    const expected = [_]TokenKind{ .module, .import, .@"export", .@"struct", .@"enum", .concept, .interface, .impl, .marker, .unsafe, .mut, .@"const", .@"return", .match, .when, .@"if", .@"else", .@"while", .@"for", .machine, .state, .transition, .yield, .run, .true, .false, .eof };
    try expectLexedKinds(source_text, &expected);
}

test "library and attribute names tokenize as identifiers" {
    const expected = [_]TokenKind{ .identifier, .identifier, .identifier, .identifier, .eof };
    try expectLexedKinds("Result Drop Fact InlineData", &expected);
}

test "line comments are skipped" {
    const expected = [_]TokenKind{ .module, .identifier, .semicolon, .eof };
    try expectLexedKinds("// comment with tokens :: ->\nmodule Test; // tail", &expected);
}

test "block comments are skipped" {
    const expected = [_]TokenKind{ .module, .identifier, .semicolon, .eof };
    try expectLexedKinds("/* comment :: -> */ module /* middle */ Test;", &expected);
}

test "punctuation operators tokenize with longest match" {
    const source_text = ":: : -> - == = != ! <= < >= > && & || ( ) { } [ ] ; , . + * / %";
    const expected = [_]TokenKind{ .colon_colon, .colon, .arrow, .minus, .equal_equal, .equal, .bang_equal, .bang, .less_equal, .less, .greater_equal, .greater, .ampersand_ampersand, .ampersand, .pipe_pipe, .left_paren, .right_paren, .left_brace, .right_brace, .left_bracket, .right_bracket, .semicolon, .comma, .dot, .plus, .star, .slash, .percent, .eof };
    try expectLexedKinds(source_text, &expected);
}

test "invalid character produces invalid-character diagnostic" {
    const source_file = try SourceFile.init(std.testing.allocator, "invalid.con", "@");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.InvalidCharacter, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("CON0001", diagnostics.diagnostics.items[0].code.format());
    try std.testing.expectEqual(TokenKind.invalid, tokens[0].kind);
    try std.testing.expectEqual(TokenKind.eof, tokens[1].kind);
}

test "single pipe is invalid while double pipe is punctuation" {
    const source_file = try SourceFile.init(std.testing.allocator, "pipe.con", "| ||");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.InvalidCharacter, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqual(TokenKind.invalid, tokens[0].kind);
    try std.testing.expectEqual(TokenKind.pipe_pipe, tokens[1].kind);
    try std.testing.expectEqual(TokenKind.eof, tokens[2].kind);
}

test "unterminated block comment produces diagnostic and EOF" {
    const source_file = try SourceFile.init(std.testing.allocator, "comment.con", "module /* unterminated");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.UnterminatedBlockComment, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqual(TokenKind.module, tokens[0].kind);
    try std.testing.expectEqual(TokenKind.eof, tokens[1].kind);
}

test "EOF token span is zero-length at source length" {
    const source_file = try SourceFile.init(std.testing.allocator, "eof.con", "module Test;");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    const eof = tokens[tokens.len - 1];
    try std.testing.expectEqual(TokenKind.eof, eof.kind);
    try std.testing.expectEqual(source_file.len(), eof.span.start);
    try std.testing.expectEqual(@as(usize, 0), eof.span.length);
    try std.testing.expectEqualStrings("", eof.lexeme);
}

test "token raw lexemes match exact source slices" {
    const source_file = try SourceFile.init(std.testing.allocator, "lexeme.con", "module Test::Name -> value");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    const expected = [_][]const u8{ "module", "Test", "::", "Name", "->", "value", "" };
    try std.testing.expectEqual(expected.len, tokens.len);
    for (expected, tokens) |expected_lexeme, actual| {
        try std.testing.expectEqualStrings(expected_lexeme, actual.lexeme);
        try std.testing.expectEqualStrings(try source_file.slice(actual.span), actual.lexeme);
    }
}
