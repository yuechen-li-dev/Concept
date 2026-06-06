const std = @import("std");
const source = @import("source.zig");

pub const SourceSpan = source.SourceSpan;

/// Syntactic token categories recognized by the Concept front end.
///
/// This is only the token model: literal spelling is preserved on `Token`, but
/// semantic literal values are intentionally not represented yet.
pub const TokenKind = enum {
    eof,
    invalid,
    identifier,
    int_literal,
    float_literal,
    string_literal,
    char_literal,

    module,
    import,
    @"export",
    @"struct",
    @"enum",
    concept,
    interface,
    impl,
    marker,
    unsafe,
    mut,
    @"const",
    @"return",
    match,
    when,
    @"if",
    @"else",
    @"while",
    @"for",
    machine,
    state,
    transition,
    yield,
    run,
    true,
    false,

    left_paren,
    right_paren,
    left_brace,
    right_brace,
    left_bracket,
    right_bracket,
    less,
    greater,
    semicolon,
    comma,
    dot,
    colon,
    colon_colon,
    arrow,
    fat_arrow,
    ampersand,
    plus,
    minus,
    star,
    slash,
    percent,
    equal,
    equal_equal,
    bang,
    bang_equal,
    less_equal,
    greater_equal,
    ampersand_ampersand,
    pipe_pipe,

    /// Stable, human-readable token kind name for tests and diagnostics.
    pub fn name(self: TokenKind) []const u8 {
        return switch (self) {
            .eof => "eof",
            .invalid => "invalid",
            .identifier => "identifier",
            .int_literal => "int_literal",
            .float_literal => "float_literal",
            .string_literal => "string_literal",
            .char_literal => "char_literal",

            .module => "module",
            .import => "import",
            .@"export" => "export",
            .@"struct" => "struct",
            .@"enum" => "enum",
            .concept => "concept",
            .interface => "interface",
            .impl => "impl",
            .marker => "marker",
            .unsafe => "unsafe",
            .mut => "mut",
            .@"const" => "const",
            .@"return" => "return",
            .match => "match",
            .when => "when",
            .@"if" => "if",
            .@"else" => "else",
            .@"while" => "while",
            .@"for" => "for",
            .machine => "machine",
            .state => "state",
            .transition => "transition",
            .yield => "yield",
            .run => "run",
            .true => "true",
            .false => "false",

            .left_paren => "left_paren",
            .right_paren => "right_paren",
            .left_brace => "left_brace",
            .right_brace => "right_brace",
            .left_bracket => "left_bracket",
            .right_bracket => "right_bracket",
            .less => "less",
            .greater => "greater",
            .semicolon => "semicolon",
            .comma => "comma",
            .dot => "dot",
            .colon => "colon",
            .colon_colon => "colon_colon",
            .arrow => "arrow",
            .fat_arrow => "fat_arrow",
            .ampersand => "ampersand",
            .plus => "plus",
            .minus => "minus",
            .star => "star",
            .slash => "slash",
            .percent => "percent",
            .equal => "equal",
            .equal_equal => "equal_equal",
            .bang => "bang",
            .bang_equal => "bang_equal",
            .less_equal => "less_equal",
            .greater_equal => "greater_equal",
            .ampersand_ampersand => "ampersand_ampersand",
            .pipe_pipe => "pipe_pipe",
        };
    }

    /// Fixed source spelling for reserved words and punctuation/operators.
    ///
    /// Structural token kinds such as identifiers and literals return null
    /// because their lexeme is source-dependent and stored on `Token`.
    pub fn lexeme(self: TokenKind) ?[]const u8 {
        return switch (self) {
            .eof,
            .invalid,
            .identifier,
            .int_literal,
            .float_literal,
            .string_literal,
            .char_literal,
            => null,

            .module => "module",
            .import => "import",
            .@"export" => "export",
            .@"struct" => "struct",
            .@"enum" => "enum",
            .concept => "concept",
            .interface => "interface",
            .impl => "impl",
            .marker => "marker",
            .unsafe => "unsafe",
            .mut => "mut",
            .@"const" => "const",
            .@"return" => "return",
            .match => "match",
            .when => "when",
            .@"if" => "if",
            .@"else" => "else",
            .@"while" => "while",
            .@"for" => "for",
            .machine => "machine",
            .state => "state",
            .transition => "transition",
            .yield => "yield",
            .run => "run",
            .true => "true",
            .false => "false",

            .left_paren => "(",
            .right_paren => ")",
            .left_brace => "{",
            .right_brace => "}",
            .left_bracket => "[",
            .right_bracket => "]",
            .less => "<",
            .greater => ">",
            .semicolon => ";",
            .comma => ",",
            .dot => ".",
            .colon => ":",
            .colon_colon => "::",
            .arrow => "->",
            .fat_arrow => "=>",
            .ampersand => "&",
            .plus => "+",
            .minus => "-",
            .star => "*",
            .slash => "/",
            .percent => "%",
            .equal => "=",
            .equal_equal => "==",
            .bang => "!",
            .bang_equal => "!=",
            .less_equal => "<=",
            .greater_equal => ">=",
            .ampersand_ampersand => "&&",
            .pipe_pipe => "||",
        };
    }
};

/// A borrowed token view over source text.
pub const Token = struct {
    kind: TokenKind,
    span: SourceSpan,
    lexeme: []const u8,

    pub fn init(kind: TokenKind, span: SourceSpan, lexeme: []const u8) Token {
        return .{
            .kind = kind,
            .span = span,
            .lexeme = lexeme,
        };
    }
};

pub fn keywordKind(identifier_text: []const u8) ?TokenKind {
    if (std.mem.eql(u8, identifier_text, "module")) return .module;
    if (std.mem.eql(u8, identifier_text, "import")) return .import;
    if (std.mem.eql(u8, identifier_text, "export")) return .@"export";
    if (std.mem.eql(u8, identifier_text, "struct")) return .@"struct";
    if (std.mem.eql(u8, identifier_text, "enum")) return .@"enum";
    if (std.mem.eql(u8, identifier_text, "concept")) return .concept;
    if (std.mem.eql(u8, identifier_text, "interface")) return .interface;
    if (std.mem.eql(u8, identifier_text, "impl")) return .impl;
    if (std.mem.eql(u8, identifier_text, "marker")) return .marker;
    if (std.mem.eql(u8, identifier_text, "unsafe")) return .unsafe;
    if (std.mem.eql(u8, identifier_text, "mut")) return .mut;
    if (std.mem.eql(u8, identifier_text, "const")) return .@"const";
    if (std.mem.eql(u8, identifier_text, "return")) return .@"return";
    if (std.mem.eql(u8, identifier_text, "match")) return .match;
    if (std.mem.eql(u8, identifier_text, "when")) return .when;
    if (std.mem.eql(u8, identifier_text, "if")) return .@"if";
    if (std.mem.eql(u8, identifier_text, "else")) return .@"else";
    if (std.mem.eql(u8, identifier_text, "while")) return .@"while";
    if (std.mem.eql(u8, identifier_text, "for")) return .@"for";
    if (std.mem.eql(u8, identifier_text, "machine")) return .machine;
    if (std.mem.eql(u8, identifier_text, "state")) return .state;
    if (std.mem.eql(u8, identifier_text, "transition")) return .transition;
    if (std.mem.eql(u8, identifier_text, "yield")) return .yield;
    if (std.mem.eql(u8, identifier_text, "run")) return .run;
    if (std.mem.eql(u8, identifier_text, "true")) return .true;
    if (std.mem.eql(u8, identifier_text, "false")) return .false;

    return null;
}

pub fn punctuationKind(lexeme: []const u8) ?TokenKind {
    if (std.mem.eql(u8, lexeme, "(")) return .left_paren;
    if (std.mem.eql(u8, lexeme, ")")) return .right_paren;
    if (std.mem.eql(u8, lexeme, "{")) return .left_brace;
    if (std.mem.eql(u8, lexeme, "}")) return .right_brace;
    if (std.mem.eql(u8, lexeme, "[")) return .left_bracket;
    if (std.mem.eql(u8, lexeme, "]")) return .right_bracket;
    if (std.mem.eql(u8, lexeme, "<")) return .less;
    if (std.mem.eql(u8, lexeme, ">")) return .greater;
    if (std.mem.eql(u8, lexeme, ";")) return .semicolon;
    if (std.mem.eql(u8, lexeme, ",")) return .comma;
    if (std.mem.eql(u8, lexeme, ".")) return .dot;
    if (std.mem.eql(u8, lexeme, ":")) return .colon;
    if (std.mem.eql(u8, lexeme, "::")) return .colon_colon;
    if (std.mem.eql(u8, lexeme, "->")) return .arrow;
    if (std.mem.eql(u8, lexeme, "=>")) return .fat_arrow;
    if (std.mem.eql(u8, lexeme, "&")) return .ampersand;
    if (std.mem.eql(u8, lexeme, "+")) return .plus;
    if (std.mem.eql(u8, lexeme, "-")) return .minus;
    if (std.mem.eql(u8, lexeme, "*")) return .star;
    if (std.mem.eql(u8, lexeme, "/")) return .slash;
    if (std.mem.eql(u8, lexeme, "%")) return .percent;
    if (std.mem.eql(u8, lexeme, "=")) return .equal;
    if (std.mem.eql(u8, lexeme, "==")) return .equal_equal;
    if (std.mem.eql(u8, lexeme, "!")) return .bang;
    if (std.mem.eql(u8, lexeme, "!=")) return .bang_equal;
    if (std.mem.eql(u8, lexeme, "<=")) return .less_equal;
    if (std.mem.eql(u8, lexeme, ">=")) return .greater_equal;
    if (std.mem.eql(u8, lexeme, "&&")) return .ampersand_ampersand;
    if (std.mem.eql(u8, lexeme, "||")) return .pipe_pipe;

    return null;
}

const KeywordCase = struct {
    text: []const u8,
    kind: TokenKind,
};

const keyword_cases = [_]KeywordCase{
    .{ .text = "module", .kind = .module },
    .{ .text = "import", .kind = .import },
    .{ .text = "export", .kind = .@"export" },
    .{ .text = "struct", .kind = .@"struct" },
    .{ .text = "enum", .kind = .@"enum" },
    .{ .text = "concept", .kind = .concept },
    .{ .text = "interface", .kind = .interface },
    .{ .text = "impl", .kind = .impl },
    .{ .text = "marker", .kind = .marker },
    .{ .text = "unsafe", .kind = .unsafe },
    .{ .text = "mut", .kind = .mut },
    .{ .text = "const", .kind = .@"const" },
    .{ .text = "return", .kind = .@"return" },
    .{ .text = "match", .kind = .match },
    .{ .text = "when", .kind = .when },
    .{ .text = "if", .kind = .@"if" },
    .{ .text = "else", .kind = .@"else" },
    .{ .text = "while", .kind = .@"while" },
    .{ .text = "for", .kind = .@"for" },
    .{ .text = "machine", .kind = .machine },
    .{ .text = "state", .kind = .state },
    .{ .text = "transition", .kind = .transition },
    .{ .text = "yield", .kind = .yield },
    .{ .text = "run", .kind = .run },
    .{ .text = "true", .kind = .true },
    .{ .text = "false", .kind = .false },
};

const PunctuationCase = struct {
    text: []const u8,
    kind: TokenKind,
};

const punctuation_cases = [_]PunctuationCase{
    .{ .text = "(", .kind = .left_paren },
    .{ .text = ")", .kind = .right_paren },
    .{ .text = "{", .kind = .left_brace },
    .{ .text = "}", .kind = .right_brace },
    .{ .text = "[", .kind = .left_bracket },
    .{ .text = "]", .kind = .right_bracket },
    .{ .text = "<", .kind = .less },
    .{ .text = ">", .kind = .greater },
    .{ .text = ";", .kind = .semicolon },
    .{ .text = ",", .kind = .comma },
    .{ .text = ".", .kind = .dot },
    .{ .text = ":", .kind = .colon },
    .{ .text = "::", .kind = .colon_colon },
    .{ .text = "->", .kind = .arrow },
    .{ .text = "=>", .kind = .fat_arrow },
    .{ .text = "&", .kind = .ampersand },
    .{ .text = "+", .kind = .plus },
    .{ .text = "-", .kind = .minus },
    .{ .text = "*", .kind = .star },
    .{ .text = "/", .kind = .slash },
    .{ .text = "%", .kind = .percent },
    .{ .text = "=", .kind = .equal },
    .{ .text = "==", .kind = .equal_equal },
    .{ .text = "!", .kind = .bang },
    .{ .text = "!=", .kind = .bang_equal },
    .{ .text = "<=", .kind = .less_equal },
    .{ .text = ">=", .kind = .greater_equal },
    .{ .text = "&&", .kind = .ampersand_ampersand },
    .{ .text = "||", .kind = .pipe_pipe },
};

test "each keyword maps to its token kind" {
    for (keyword_cases) |case| {
        try std.testing.expectEqual(case.kind, keywordKind(case.text).?);
        try std.testing.expectEqualStrings(case.text, case.kind.lexeme().?);
    }
}

test "non-keyword identifiers return null" {
    const identifiers = [_][]const u8{
        "",
        "Module",
        "module_name",
        "identifier",
        "imports",
        "unsafe_mode",
    };

    for (identifiers) |identifier| {
        try std.testing.expectEqual(@as(?TokenKind, null), keywordKind(identifier));
    }
}

test "language library and attribute names are identifiers, not keywords" {
    const identifiers = [_][]const u8{
        "Result",
        "Vector",
        "Assert",
        "Fact",
        "Theory",
        "InlineData",
        "Drop",
    };

    for (identifiers) |identifier| {
        try std.testing.expectEqual(@as(?TokenKind, null), keywordKind(identifier));
    }
}

test "punctuation token kinds have stable names and lexemes" {
    for (punctuation_cases) |case| {
        try std.testing.expectEqual(case.kind, punctuationKind(case.text).?);
        try std.testing.expectEqualStrings(case.text, case.kind.lexeme().?);
    }

    try std.testing.expectEqualStrings("left_paren", TokenKind.left_paren.name());
    try std.testing.expectEqualStrings("colon_colon", TokenKind.colon_colon.name());
    try std.testing.expectEqualStrings("ampersand_ampersand", TokenKind.ampersand_ampersand.name());
}

test "unknown punctuation lookup returns null" {
    const unknown = [_][]const u8{ "", "@", "|", "===", "..." };

    for (unknown) |text| {
        try std.testing.expectEqual(@as(?TokenKind, null), punctuationKind(text));
    }
}

test "token can be constructed with span and raw lexeme" {
    const text = "answer";
    const token = Token.init(.identifier, .{ .start = 4, .length = text.len }, text);

    try std.testing.expectEqual(TokenKind.identifier, token.kind);
    try std.testing.expectEqual(@as(usize, 4), token.span.start);
    try std.testing.expectEqual(@as(usize, 6), token.span.length);
    try std.testing.expectEqualStrings(text, token.lexeme);
}

test "token kind name output is stable" {
    try std.testing.expectEqualStrings("eof", TokenKind.eof.name());
    try std.testing.expectEqualStrings("invalid", TokenKind.invalid.name());
    try std.testing.expectEqualStrings("identifier", TokenKind.identifier.name());
    try std.testing.expectEqualStrings("int_literal", TokenKind.int_literal.name());
    try std.testing.expectEqualStrings("float_literal", TokenKind.float_literal.name());
    try std.testing.expectEqualStrings("string_literal", TokenKind.string_literal.name());
    try std.testing.expectEqualStrings("char_literal", TokenKind.char_literal.name());
    try std.testing.expectEqualStrings("module", TokenKind.module.name());
    try std.testing.expectEqualStrings("struct", TokenKind.@"struct".name());
    try std.testing.expectEqualStrings("return", TokenKind.@"return".name());
    try std.testing.expectEqualStrings("true", TokenKind.true.name());
    try std.testing.expectEqualStrings("false", TokenKind.false.name());
    try std.testing.expectEqualStrings("pipe_pipe", TokenKind.pipe_pipe.name());
}

test "structural token kinds do not have fixed lexemes" {
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.eof.lexeme());
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.invalid.lexeme());
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.identifier.lexeme());
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.int_literal.lexeme());
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.float_literal.lexeme());
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.string_literal.lexeme());
    try std.testing.expectEqual(@as(?[]const u8, null), TokenKind.char_literal.lexeme());
}
