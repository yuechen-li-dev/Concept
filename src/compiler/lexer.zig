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

        if (std.ascii.isDigit(self.peek().?)) {
            return self.scanNumber();
        }

        if (self.peek().? == '"') {
            return self.scanString();
        }

        if (self.peek().? == '\'') {
            return self.scanChar();
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
            self.diagnostics.allocator,
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

    fn scanNumber(self: *Lexer) Token {
        const start = self.offset;
        while (!self.isAtEnd() and std.ascii.isDigit(self.peek().?)) {
            _ = self.advance();
        }

        if (self.peek() == '.') {
            if (self.peekNext()) |next| {
                if (std.ascii.isDigit(next)) {
                    _ = self.advance();
                    while (!self.isAtEnd() and std.ascii.isDigit(self.peek().?)) {
                        _ = self.advance();
                    }
                    return self.makeToken(.float_literal, start, self.offset);
                }
            }
        }

        return self.makeToken(.int_literal, start, self.offset);
    }

    fn scanString(self: *Lexer) !Token {
        const start = self.offset;
        _ = self.advance();

        while (!self.isAtEnd()) {
            const byte = self.peek().?;
            if (byte == '"') {
                _ = self.advance();
                return self.makeToken(.string_literal, start, self.offset);
            }

            if (isLineBreak(byte)) {
                try self.diagnostics.append(diagnostics_model.unterminatedString(.{
                    .start = start,
                    .length = self.offset - start,
                }));
                return self.makeToken(.string_literal, start, self.offset);
            }

            if (byte == '\\') {
                try self.scanEscape(false);
                continue;
            }

            _ = self.advance();
        }

        try self.diagnostics.append(diagnostics_model.unterminatedString(.{
            .start = start,
            .length = self.offset - start,
        }));
        return self.makeToken(.string_literal, start, self.offset);
    }

    fn scanChar(self: *Lexer) !Token {
        const start = self.offset;
        _ = self.advance();

        if (self.peek() == '\'') {
            _ = self.advance();
            try self.diagnostics.append(diagnostics_model.emptyCharLiteral(.{
                .start = start,
                .length = self.offset - start,
            }));
            return self.makeToken(.char_literal, start, self.offset);
        }

        while (!self.isAtEnd()) {
            const byte = self.peek().?;
            if (byte == '\'') {
                _ = self.advance();
                return self.makeToken(.char_literal, start, self.offset);
            }

            if (isLineBreak(byte)) {
                try self.diagnostics.append(diagnostics_model.unterminatedChar(.{
                    .start = start,
                    .length = self.offset - start,
                }));
                return self.makeToken(.char_literal, start, self.offset);
            }

            if (byte == '\\') {
                try self.scanEscape(true);
                continue;
            }

            _ = self.advance();
        }

        try self.diagnostics.append(diagnostics_model.unterminatedChar(.{
            .start = start,
            .length = self.offset - start,
        }));
        return self.makeToken(.char_literal, start, self.offset);
    }

    fn scanEscape(self: *Lexer, allow_single_quote: bool) !void {
        const escape_start = self.offset;
        _ = self.advance();

        if (self.isAtEnd()) {
            return;
        }

        const escaped = self.peek().?;
        if (isLineBreak(escaped)) {
            return;
        }

        _ = self.advance();
        if (!isRecognizedEscape(escaped, allow_single_quote)) {
            try self.diagnostics.append(diagnostics_model.invalidEscapeSequence(.{
                .start = escape_start,
                .length = self.offset - escape_start,
            }));
        }
    }

    fn matchPunctuation(self: *Lexer) ?Token {
        const two_char_kinds = [_]struct { lexeme: []const u8, kind: TokenKind }{
            .{ .lexeme = "::", .kind = .colon_colon },
            .{ .lexeme = "->", .kind = .arrow },
            .{ .lexeme = "=>", .kind = .fat_arrow },
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

    fn peekNext(self: Lexer) ?u8 {
        if (self.offset + 1 >= self.source_file.text.len) return null;
        return self.source_file.text[self.offset + 1];
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
    var tokens = std.ArrayList(Token).empty;
    errdefer tokens.deinit(allocator);

    while (true) {
        const token = try lexer.nextToken();
        try tokens.append(allocator, token);
        if (token.kind == .eof) break;
    }

    return tokens.toOwnedSlice(allocator);
}

fn isWhitespace(byte: u8) bool {
    return switch (byte) {
        ' ', '\t', '\n', '\r' => true,
        else => false,
    };
}

fn isLineBreak(byte: u8) bool {
    return byte == '\n' or byte == '\r';
}

fn isRecognizedEscape(byte: u8, allow_single_quote: bool) bool {
    return switch (byte) {
        '\\', '"', 'n', 'r', 't', '0' => true,
        '\'' => allow_single_quote,
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
    const source_text = "module import export struct enum concept interface impl marker unsafe mut const return comptime must_use discard match decide when if else while for machine state transition yield run true false";
    const expected = [_]TokenKind{ .module, .import, .@"export", .@"struct", .@"enum", .concept, .interface, .impl, .marker, .unsafe, .mut, .@"const", .@"return", .@"comptime", .must_use, .discard, .match, .decide, .when, .@"if", .@"else", .@"while", .@"for", .machine, .state, .transition, .yield, .run, .true, .false, .eof };
    try expectLexedKinds(source_text, &expected);
}

test "library and attribute names tokenize as identifiers" {
    const expected = [_]TokenKind{ .identifier, .identifier, .identifier, .identifier, .eof };
    try expectLexedKinds("Result Drop Fact InlineData", &expected);
}

test "comptime tokenizes as keyword without consuming nearby identifiers" {
    const expected = [_]TokenKind{ .@"comptime", .identifier, .identifier, .identifier, .identifier, .eof };
    try expectLexedKinds("comptime comptimeValue compileTime comptimer comptime_mode", &expected);
}

test "must_use and discard keywords do not consume nearby identifiers" {
    const expected = [_]TokenKind{ .must_use, .discard, .identifier, .identifier, .eof };
    try expectLexedKinds("must_use discard must_useful discarded", &expected);
}

test "unsafe keyword does not consume nearby identifiers" {
    const expected = [_]TokenKind{ .unsafe, .identifier, .identifier, .identifier, .eof };
    try expectLexedKinds("unsafe unsafeValue unsafely unsafe_mode", &expected);
}

test "decide and when tokenize as keywords while score remains contextual identifier" {
    const expected = [_]TokenKind{ .decide, .when, .identifier, .identifier, .identifier, .identifier, .eof };
    try expectLexedKinds("decide when score scoreValue score_thing highScore", &expected);
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
    const source_text = ":: : -> - => = == = != ! <= < >= > && & || ( ) { } [ ] ; , . + * / %";
    const expected = [_]TokenKind{ .colon_colon, .colon, .arrow, .minus, .fat_arrow, .equal, .equal_equal, .equal, .bang_equal, .bang, .less_equal, .less, .greater_equal, .greater, .ampersand_ampersand, .ampersand, .pipe_pipe, .left_paren, .right_paren, .left_brace, .right_brace, .left_bracket, .right_bracket, .semicolon, .comma, .dot, .plus, .star, .slash, .percent, .eof };
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

test "decimal integer literals tokenize with raw lexemes" {
    const source_file = try SourceFile.init(std.testing.allocator, "ints.con", "0 123 987654");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    const expected_kinds = [_]TokenKind{ .int_literal, .int_literal, .int_literal, .eof };
    const expected_lexemes = [_][]const u8{ "0", "123", "987654", "" };
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(expected_kinds.len, tokens.len);
    for (expected_kinds, expected_lexemes, tokens) |expected_kind, expected_lexeme, actual| {
        try std.testing.expectEqual(expected_kind, actual.kind);
        try std.testing.expectEqualStrings(expected_lexeme, actual.lexeme);
        try std.testing.expectEqualStrings(try source_file.slice(actual.span), actual.lexeme);
    }
}

test "simple float literals tokenize with raw lexemes" {
    const source_file = try SourceFile.init(std.testing.allocator, "floats.con", "1.0 0.5 123.456");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    const expected_lexemes = [_][]const u8{ "1.0", "0.5", "123.456", "" };
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(expected_lexemes.len, tokens.len);
    for (expected_lexemes, tokens[0..3]) |expected_lexeme, actual| {
        try std.testing.expectEqual(TokenKind.float_literal, actual.kind);
        try std.testing.expectEqualStrings(expected_lexeme, actual.lexeme);
        try std.testing.expectEqualStrings(try source_file.slice(actual.span), actual.lexeme);
    }
    try std.testing.expectEqual(TokenKind.eof, tokens[3].kind);
}

test "number before member dot does not consume dot as float" {
    const source_file = try SourceFile.init(std.testing.allocator, "dot.con", "1.foo");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    const expected_kinds = [_]TokenKind{ .int_literal, .dot, .identifier, .eof };
    const expected_lexemes = [_][]const u8{ "1", ".", "foo", "" };
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(expected_kinds.len, tokens.len);
    for (expected_kinds, expected_lexemes, tokens) |expected_kind, expected_lexeme, actual| {
        try std.testing.expectEqual(expected_kind, actual.kind);
        try std.testing.expectEqualStrings(expected_lexeme, actual.lexeme);
    }
}

test "string literal raw lexeme includes quotes" {
    const source_file = try SourceFile.init(std.testing.allocator, "string.con", "\"hello\"");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(TokenKind.string_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("\"hello\"", tokens[0].lexeme);
    try std.testing.expectEqualStrings(try source_file.slice(tokens[0].span), tokens[0].lexeme);
    try std.testing.expectEqual(TokenKind.eof, tokens[1].kind);
}

test "string literal recognizes common escapes and preserves raw lexeme" {
    const source_text = "\"\\\\ \\\" \\n \\r \\t \\0\"";
    const source_file = try SourceFile.init(std.testing.allocator, "escapes.con", source_text);
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(TokenKind.string_literal, tokens[0].kind);
    try std.testing.expectEqualStrings(source_text, tokens[0].lexeme);
    try std.testing.expectEqualStrings(try source_file.slice(tokens[0].span), tokens[0].lexeme);
}

test "unterminated string literal produces diagnostic" {
    const source_file = try SourceFile.init(std.testing.allocator, "unterminated_string.con", "\"hello");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.UnterminatedString, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("CON0002", diagnostics.diagnostics.items[0].code.format());
    try std.testing.expectEqual(TokenKind.string_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("\"hello", tokens[0].lexeme);
    try std.testing.expectEqual(TokenKind.eof, tokens[1].kind);
}

test "unescaped newline terminates string literal with diagnostic" {
    const source_file = try SourceFile.init(std.testing.allocator, "newline_string.con", "\"hello\nworld\"");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.UnterminatedString, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqual(TokenKind.string_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("\"hello", tokens[0].lexeme);
}

test "char literal raw lexeme includes quotes" {
    const source_file = try SourceFile.init(std.testing.allocator, "char.con", "'a'");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(TokenKind.char_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("'a'", tokens[0].lexeme);
    try std.testing.expectEqualStrings(try source_file.slice(tokens[0].span), tokens[0].lexeme);
}

test "escaped char literal raw lexemes are preserved" {
    const source_file = try SourceFile.init(std.testing.allocator, "escaped_char.con", "'\\n' '\\''");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(TokenKind.char_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("'\\n'", tokens[0].lexeme);
    try std.testing.expectEqual(TokenKind.char_literal, tokens[1].kind);
    try std.testing.expectEqualStrings("'\\''", tokens[1].lexeme);
}

test "unterminated char literal produces diagnostic" {
    const source_file = try SourceFile.init(std.testing.allocator, "unterminated_char.con", "'a");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.UnterminatedChar, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("CON0007", diagnostics.diagnostics.items[0].code.format());
    try std.testing.expectEqual(TokenKind.char_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("'a", tokens[0].lexeme);
}

test "empty char literal produces diagnostic" {
    const source_file = try SourceFile.init(std.testing.allocator, "empty_char.con", "''");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.EmptyCharLiteral, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("CON0008", diagnostics.diagnostics.items[0].code.format());
    try std.testing.expectEqual(TokenKind.char_literal, tokens[0].kind);
    try std.testing.expectEqualStrings("''", tokens[0].lexeme);
}

test "invalid escape sequence produces diagnostic" {
    const source_file = try SourceFile.init(std.testing.allocator, "invalid_escape.con", "\"bad\\x\" '\\q'");
    defer source_file.deinit(std.testing.allocator);

    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 2), diagnostics.count());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.InvalidEscapeSequence, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("CON0009", diagnostics.diagnostics.items[0].code.format());
    try std.testing.expectEqual(diagnostics_model.DiagnosticCode.InvalidEscapeSequence, diagnostics.diagnostics.items[1].code);
    try std.testing.expectEqual(TokenKind.string_literal, tokens[0].kind);
    try std.testing.expectEqual(TokenKind.char_literal, tokens[1].kind);
}

test "fat arrow tokenizes" {
    const expected = [_]TokenKind{ .fat_arrow, .eof };
    try expectLexedKinds("=>", &expected);
}

test "fat arrow uses longest match before equal" {
    const expected = [_]TokenKind{ .fat_arrow, .equal, .eof };
    try expectLexedKinds("=> =", &expected);
}

test "try tokenizes as keyword and trying as identifier" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const source_file = try SourceFile.init(std.testing.allocator, "try.con", "try trying");
    defer source_file.deinit(std.testing.allocator);
    const tokens = try lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(TokenKind.@"try", tokens[0].kind);
    try std.testing.expectEqual(TokenKind.identifier, tokens[1].kind);
}
