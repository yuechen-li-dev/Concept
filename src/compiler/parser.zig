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
            for (items.items) |item| item.deinit(allocator);
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
                .@"struct" => {
                    try items.append(try self.parseStructItem(allocator));
                },
                .@"enum" => {
                    try items.append(try self.parseEnumItem(allocator));
                },
                .concept => {
                    try items.append(try self.parseConceptItem(allocator));
                },
                .interface => {
                    try items.append(try self.parseInterfaceItem(allocator));
                },
                .impl => {
                    try items.append(try self.parseImplItem(allocator));
                },
                .@"export" => {
                    if (self.peek(1).kind == .@"struct") {
                        try items.append(try self.parseStructItem(allocator));
                    } else if (self.peek(1).kind == .@"enum") {
                        try items.append(try self.parseEnumItem(allocator));
                    } else {
                        try self.reportExpectedItem("expected item declaration", self.current().span);
                        self.advance();
                    }
                },
                .import => {
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
        const owned_items = try items.toOwnedSlice();
        errdefer {
            for (owned_items) |item| item.deinit(allocator);
            allocator.free(owned_items);
        }

        return .{
            .span = ast.spanFromBounds(start, eof_span.start),
            .module = module_decl,
            .imports = owned_imports,
            .items = owned_items,
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

    fn parseStructItem(self: *Parser, allocator: std.mem.Allocator) !ast.Item {
        return .{ .struct_decl = try self.parseStructDecl(allocator) };
    }

    fn parseEnumItem(self: *Parser, allocator: std.mem.Allocator) !ast.Item {
        return .{ .enum_decl = try self.parseEnumDecl(allocator) };
    }

    fn parseConceptItem(self: *Parser, allocator: std.mem.Allocator) !ast.Item {
        return .{ .concept_decl = try self.parseConceptDecl(allocator) };
    }

    fn parseInterfaceItem(self: *Parser, allocator: std.mem.Allocator) !ast.Item {
        return .{ .interface_decl = try self.parseInterfaceDecl(allocator) };
    }

    fn parseImplItem(self: *Parser, allocator: std.mem.Allocator) !ast.Item {
        return .{ .impl_decl = try self.parseImplDecl(allocator) };
    }

    fn parseStructDecl(self: *Parser, allocator: std.mem.Allocator) !ast.StructDecl {
        const start_token = if (self.match(.@"export")) |export_token| export_token else null;
        const struct_token = (try self.expect(.@"struct", "expected 'struct' after export", .ExpectedItem)) orelse start_token.?;
        const start_span = if (start_token) |export_token| export_token.span else struct_token.span;

        const name_token = try self.expect(.identifier, "expected struct name", .UnexpectedToken);
        const name = if (name_token) |identifier|
            ast.NameSegment{ .text = identifier.lexeme, .span = identifier.span }
        else
            ast.NameSegment{ .text = "", .span = self.current().span };

        var fields = std.ArrayList(ast.FieldDecl).init(allocator);
        errdefer {
            for (fields.items) |field| field.deinit(allocator);
            fields.deinit();
        }

        if (self.match(.left_brace) == null) {
            try self.report(.UnexpectedToken, "expected '{' in struct declaration", self.current().span);
            while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .left_brace) {
                self.advance();
            }
            if (self.match(.left_brace) == null) {
                const end_span = try self.expectTrailingStructSemicolon();
                return .{
                    .is_export = start_token != null,
                    .name = name,
                    .fields = try fields.toOwnedSlice(),
                    .span = ast.spanFromBounds(start_span.start, spanEnd(end_span)),
                };
            }
        }

        const close_span = try self.parseStructBody(allocator, &fields);
        const end_span = if (self.match(.right_brace)) |right_brace| blk: {
            _ = close_span;
            break :blk try self.expectTrailingStructSemicolonAfterBrace(right_brace.span);
        } else blk: {
            const diagnostic_span = self.current().span;
            try self.report(.UnexpectedToken, "expected '}' to close struct declaration", diagnostic_span);
            if (self.match(.semicolon)) |semicolon| {
                break :blk semicolon.span;
            }
            break :blk diagnostic_span;
        };

        return .{
            .is_export = start_token != null,
            .name = name,
            .fields = try fields.toOwnedSlice(),
            .span = ast.spanFromBounds(start_span.start, spanEnd(end_span)),
        };
    }

    fn parseStructBody(self: *Parser, allocator: std.mem.Allocator, fields: *std.ArrayList(ast.FieldDecl)) !SourceSpan {
        while (self.current().kind != .eof and self.current().kind != .right_brace) {
            switch (self.current().kind) {
                .identifier => try fields.append(try self.parseFieldDecl(allocator)),
                .semicolon => return self.current().span,
                else => {
                    try self.report(.UnexpectedToken, "unexpected token in struct body", self.current().span);
                    self.advance();
                },
            }
        }
        return self.current().span;
    }

    fn parseFieldDecl(self: *Parser, allocator: std.mem.Allocator) !ast.FieldDecl {
        var type_name = try self.parseTypeName(allocator);
        errdefer type_name.deinit(allocator);

        const name_token = try self.expect(.identifier, "expected field name", .UnexpectedToken);
        const name = if (name_token) |identifier|
            ast.NameSegment{ .text = identifier.lexeme, .span = identifier.span }
        else
            ast.NameSegment{ .text = "", .span = self.current().span };

        const end_span = try self.expectFieldSemicolon("expected ';' after field declaration");
        return .{
            .type_name = type_name,
            .name = name,
            .span = ast.spanFromBounds(type_name.span.start, spanEnd(end_span)),
        };
    }

    fn expectFieldSemicolon(self: *Parser, message: []const u8) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;

        const diagnostic_span = self.current().span;
        try self.report(.UnexpectedToken, message, diagnostic_span);
        return diagnostic_span;
    }

    fn expectTrailingStructSemicolonAfterBrace(self: *Parser, right_brace_span: SourceSpan) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;

        try self.report(.UnexpectedToken, "expected ';' after struct declaration", self.current().span);
        return right_brace_span;
    }

    fn expectTrailingStructSemicolon(self: *Parser) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;
        try self.report(.UnexpectedToken, "expected ';' after struct declaration", self.current().span);
        return self.current().span;
    }

    fn parseEnumDecl(self: *Parser, allocator: std.mem.Allocator) !ast.EnumDecl {
        const start_token = if (self.match(.@"export")) |export_token| export_token else null;
        const enum_token = (try self.expect(.@"enum", "expected 'enum' after export", .ExpectedItem)) orelse start_token.?;
        const start_span = if (start_token) |export_token| export_token.span else enum_token.span;

        const name_token = try self.expect(.identifier, "expected enum name", .UnexpectedToken);
        const name = if (name_token) |identifier|
            ast.NameSegment{ .text = identifier.lexeme, .span = identifier.span }
        else
            ast.NameSegment{ .text = "", .span = self.current().span };

        var variants = std.ArrayList(ast.EnumVariant).init(allocator);
        errdefer {
            for (variants.items) |variant| variant.deinit(allocator);
            variants.deinit();
        }

        if (self.match(.left_brace) == null) {
            try self.report(.UnexpectedToken, "expected '{' in enum declaration", self.current().span);
            while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .left_brace) {
                self.advance();
            }
            if (self.match(.left_brace) == null) {
                const end_span = try self.expectTrailingEnumSemicolon();
                return .{
                    .is_export = start_token != null,
                    .name = name,
                    .variants = try variants.toOwnedSlice(),
                    .span = ast.spanFromBounds(start_span.start, spanEnd(end_span)),
                };
            }
        }

        const close_span = try self.parseEnumBody(allocator, &variants);
        const end_span = if (self.match(.right_brace)) |right_brace| blk: {
            _ = close_span;
            break :blk try self.expectTrailingEnumSemicolonAfterBrace(right_brace.span);
        } else blk: {
            const diagnostic_span = self.current().span;
            try self.report(.UnexpectedToken, "expected '}' to close enum declaration", diagnostic_span);
            if (self.match(.semicolon)) |semicolon| {
                break :blk semicolon.span;
            }
            break :blk diagnostic_span;
        };

        return .{
            .is_export = start_token != null,
            .name = name,
            .variants = try variants.toOwnedSlice(),
            .span = ast.spanFromBounds(start_span.start, spanEnd(end_span)),
        };
    }

    fn parseEnumBody(self: *Parser, allocator: std.mem.Allocator, variants: *std.ArrayList(ast.EnumVariant)) !SourceSpan {
        while (self.current().kind != .eof and self.current().kind != .right_brace) {
            switch (self.current().kind) {
                .identifier => {
                    try variants.append(try self.parseEnumVariant(allocator));
                    switch (self.current().kind) {
                        .comma => {
                            _ = self.advance();
                        },
                        .right_brace, .eof, .semicolon => {},
                        .identifier => try self.report(.UnexpectedToken, "expected ',' between enum variants", self.current().span),
                        else => {
                            try self.report(.UnexpectedToken, "unexpected token in enum body", self.current().span);
                            self.advance();
                            _ = self.match(.comma);
                        },
                    }
                },
                .comma => {
                    try self.report(.UnexpectedToken, "unexpected token in enum body", self.current().span);
                    self.advance();
                },
                .semicolon => return self.current().span,
                else => {
                    try self.report(.UnexpectedToken, "unexpected token in enum body", self.current().span);
                    self.advance();
                    _ = self.match(.comma);
                },
            }
        }
        return self.current().span;
    }

    fn parseEnumVariant(self: *Parser, allocator: std.mem.Allocator) !ast.EnumVariant {
        const name_token = self.advance();
        const name = ast.NameSegment{ .text = name_token.lexeme, .span = name_token.span };
        var payload_fields = std.ArrayList(ast.EnumPayloadField).init(allocator);
        errdefer {
            for (payload_fields.items) |field| field.deinit(allocator);
            payload_fields.deinit();
        }

        var end_span = name_token.span;
        if (self.match(.left_paren)) |left_paren| {
            end_span = try self.parseEnumPayloadList(allocator, &payload_fields, left_paren.span);
        }

        return .{
            .name = name,
            .payload_fields = try payload_fields.toOwnedSlice(),
            .span = ast.spanFromBounds(name.span.start, spanEnd(end_span)),
        };
    }

    fn parseEnumPayloadList(self: *Parser, allocator: std.mem.Allocator, payload_fields: *std.ArrayList(ast.EnumPayloadField), left_paren_span: SourceSpan) !SourceSpan {
        var last_span = left_paren_span;
        while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .right_brace) {
            if (self.current().kind == .comma) {
                try self.report(.UnexpectedToken, "malformed enum payload field", self.current().span);
                last_span = self.advance().span;
                continue;
            }

            if (try self.parseEnumPayloadField(allocator)) |field| {
                errdefer field.deinit(allocator);
                last_span = field.span;
                try payload_fields.append(field);
            }

            switch (self.current().kind) {
                .comma => last_span = self.advance().span,
                .right_paren, .right_brace, .eof => {},
                else => {
                    try self.report(.UnexpectedToken, "expected ',' between enum payload fields", self.current().span);
                    self.recoverEnumPayloadField();
                    if (self.current().kind == .comma) last_span = self.advance().span;
                },
            }
        }

        if (self.match(.right_paren)) |right_paren| return right_paren.span;

        try self.report(.UnexpectedToken, "expected ')' after enum payload fields", self.current().span);
        return last_span;
    }

    fn parseEnumPayloadField(self: *Parser, allocator: std.mem.Allocator) !?ast.EnumPayloadField {
        if (self.current().kind != .identifier) {
            try self.report(.UnexpectedToken, "malformed enum payload field", self.current().span);
            self.recoverEnumPayloadField();
            return null;
        }

        const start_span = self.current().span;
        var parts = std.ArrayList(ast.NameSegment).init(allocator);
        errdefer parts.deinit();

        const first = self.advance();
        try parts.append(.{ .text = first.lexeme, .span = first.span });
        var last_type_span = first.span;

        while (self.current().kind == .dot) {
            _ = self.advance();
            if (self.current().kind != .identifier) {
                try self.report(.UnexpectedToken, "malformed enum payload field", self.current().span);
                self.recoverEnumPayloadField();
                return null;
            }
            const part = self.advance();
            try parts.append(.{ .text = part.lexeme, .span = part.span });
            last_type_span = part.span;
        }

        if (self.current().kind != .identifier) {
            try self.report(.UnexpectedToken, "malformed enum payload field", self.current().span);
            self.recoverEnumPayloadField();
            return null;
        }

        const name_token = self.advance();
        const type_name = ast.TypeName{
            .name = .{
                .parts = try parts.toOwnedSlice(),
                .span = ast.spanFromBounds(start_span.start, spanEnd(last_type_span)),
            },
            .span = ast.spanFromBounds(start_span.start, spanEnd(last_type_span)),
        };

        return .{
            .type_name = type_name,
            .name = .{ .text = name_token.lexeme, .span = name_token.span },
            .span = ast.spanFromBounds(start_span.start, spanEnd(name_token.span)),
        };
    }

    fn recoverEnumPayloadField(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .comma and self.current().kind != .right_paren and self.current().kind != .right_brace) {
            self.advance();
        }
    }

    fn expectTrailingEnumSemicolonAfterBrace(self: *Parser, right_brace_span: SourceSpan) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;

        try self.report(.UnexpectedToken, "expected ';' after enum declaration", self.current().span);
        return right_brace_span;
    }

    fn expectTrailingEnumSemicolon(self: *Parser) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;
        try self.report(.UnexpectedToken, "expected ';' after enum declaration", self.current().span);
        return self.current().span;
    }

    fn parseConceptDecl(self: *Parser, allocator: std.mem.Allocator) !ast.ConceptDecl {
        const concept_token = self.advance();
        const name = try self.parseRequiredName("expected concept name");
        const generic_params = try self.parseOptionalGenericParams(allocator);
        errdefer allocator.free(generic_params);
        var signatures = std.ArrayList(ast.SignatureDecl).init(allocator);
        errdefer {
            for (signatures.items) |signature| signature.deinit(allocator);
            signatures.deinit();
        }

        const end_span = try self.parseSignatureBlock(allocator, &signatures, "concept");
        return .{
            .name = name,
            .generic_params = generic_params,
            .signatures = try signatures.toOwnedSlice(),
            .span = ast.spanFromBounds(concept_token.span.start, spanEnd(end_span)),
        };
    }

    fn parseInterfaceDecl(self: *Parser, allocator: std.mem.Allocator) !ast.InterfaceDecl {
        const interface_token = self.advance();
        const name = try self.parseRequiredName("expected interface name");
        var signatures = std.ArrayList(ast.SignatureDecl).init(allocator);
        errdefer {
            for (signatures.items) |signature| signature.deinit(allocator);
            signatures.deinit();
        }

        const end_span = try self.parseSignatureBlock(allocator, &signatures, "interface");
        return .{
            .name = name,
            .signatures = try signatures.toOwnedSlice(),
            .span = ast.spanFromBounds(interface_token.span.start, spanEnd(end_span)),
        };
    }

    fn parseImplDecl(self: *Parser, allocator: std.mem.Allocator) !ast.ImplDecl {
        const impl_token = self.advance();
        var target = if (self.current().kind == .identifier)
            try self.parseTypeName(allocator)
        else blk: {
            try self.report(.UnexpectedToken, "expected impl target name", self.current().span);
            break :blk try self.emptyTypeName(allocator, self.current().span);
        };
        errdefer target.deinit(allocator);

        var signatures = std.ArrayList(ast.SignatureDecl).init(allocator);
        errdefer {
            for (signatures.items) |signature| signature.deinit(allocator);
            signatures.deinit();
        }

        const end_span = try self.parseSignatureBlock(allocator, &signatures, "impl");
        return .{
            .target = target,
            .signatures = try signatures.toOwnedSlice(),
            .span = ast.spanFromBounds(impl_token.span.start, spanEnd(end_span)),
        };
    }

    fn parseRequiredName(self: *Parser, message: []const u8) !ast.NameSegment {
        const name_token = try self.expect(.identifier, message, .UnexpectedToken);
        return if (name_token) |identifier|
            ast.NameSegment{ .text = identifier.lexeme, .span = identifier.span }
        else
            ast.NameSegment{ .text = "", .span = self.current().span };
    }

    fn parseOptionalGenericParams(self: *Parser, allocator: std.mem.Allocator) ![]ast.GenericParam {
        var params = std.ArrayList(ast.GenericParam).init(allocator);
        errdefer params.deinit();

        if (self.match(.less) == null) return params.toOwnedSlice();
        while (self.current().kind != .eof and self.current().kind != .greater and self.current().kind != .left_brace and self.current().kind != .semicolon) {
            if (self.current().kind == .identifier) {
                const param = self.advance();
                try params.append(.{ .text = param.lexeme, .span = param.span });
                if (self.match(.comma) != null) continue;
                if (self.current().kind == .greater) break;
                try self.report(.UnexpectedToken, "expected ',' between generic parameters", self.current().span);
                self.advance();
                continue;
            }
            try self.report(.UnexpectedToken, "expected generic parameter name", self.current().span);
            self.advance();
        }

        if (self.match(.greater) == null) {
            try self.report(.UnexpectedToken, "expected '>' after generic parameters", self.current().span);
        }
        return params.toOwnedSlice();
    }

    fn parseSignatureBlock(self: *Parser, allocator: std.mem.Allocator, signatures: *std.ArrayList(ast.SignatureDecl), decl_kind: []const u8) !SourceSpan {
        if (self.match(.left_brace) == null) {
            try self.report(.UnexpectedToken, "expected '{' in declaration", self.current().span);
            while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .left_brace) self.advance();
            if (self.match(.left_brace) == null) return try self.expectTrailingBlockSemicolon(decl_kind);
        }

        while (self.current().kind != .eof and self.current().kind != .right_brace) {
            if (self.current().kind == .semicolon) {
                try self.report(.UnexpectedToken, "unexpected token inside declaration block", self.current().span);
                self.advance();
                continue;
            }
            if (self.current().kind == .identifier or self.current().kind == .mut) {
                if (try self.parseSignatureDecl(allocator)) |signature| {
                    try signatures.append(signature);
                }
            } else {
                try self.report(.UnexpectedToken, "unexpected token inside declaration block", self.current().span);
                self.advance();
            }
        }

        if (self.match(.right_brace)) |right_brace| return try self.expectTrailingBlockSemicolonAfterBrace(decl_kind, right_brace.span);

        const diagnostic_span = self.current().span;
        try self.report(.UnexpectedToken, "expected '}' to close declaration", diagnostic_span);
        if (self.match(.semicolon)) |semicolon| return semicolon.span;
        return diagnostic_span;
    }

    fn parseSignatureDecl(self: *Parser, allocator: std.mem.Allocator) !?ast.SignatureDecl {
        const start_span = self.current().span;
        var return_type = self.parseTypeName(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            else => unreachable,
        };
        errdefer return_type.deinit(allocator);

        const name = try self.parseSignatureName();
        if (name == null) {
            return_type.deinit(allocator);
            try self.report(.UnexpectedToken, "malformed signature", self.current().span);
            self.recoverSignature();
            return null;
        }

        if (self.match(.left_paren) == null) {
            return_type.deinit(allocator);
            try self.report(.UnexpectedToken, "malformed signature", self.current().span);
            self.recoverSignature();
            return null;
        }

        var params = std.ArrayList(ast.ParamDecl).init(allocator);
        errdefer {
            for (params.items) |param| param.deinit(allocator);
            params.deinit();
        }
        var last_span = name.?.span;
        while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .right_brace and self.current().kind != .semicolon) {
            if (try self.parseParamDecl(allocator)) |param| {
                last_span = param.span;
                try params.append(param);
            }
            if (self.match(.comma)) |comma| {
                last_span = comma.span;
                continue;
            }
            if (self.current().kind == .right_paren) break;
            try self.report(.UnexpectedToken, "expected ',' between signature parameters", self.current().span);
            self.recoverSignatureParam();
            _ = self.match(.comma);
        }

        if (self.match(.right_paren)) |right_paren| {
            last_span = right_paren.span;
        } else {
            try self.report(.UnexpectedToken, "malformed signature", self.current().span);
            self.recoverSignature();
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            try self.report(.UnexpectedToken, "expected ';' after signature declaration", self.current().span);
            break :blk last_span;
        };

        return .{
            .return_type = return_type,
            .name = name.?,
            .params = try params.toOwnedSlice(),
            .span = ast.spanFromBounds(start_span.start, spanEnd(end_span)),
        };
    }

    fn parseSignatureName(self: *Parser) !?ast.SignatureName {
        if (self.current().kind != .identifier) return null;
        const base = self.advance();
        if (std.mem.eql(u8, base.lexeme, "operator")) {
            switch (self.current().kind) {
                .equal_equal, .bang_equal, .less, .greater, .less_equal, .greater_equal, .plus, .minus, .star, .slash, .percent => {
                    const suffix = self.advance();
                    return .{
                        .base = .{ .text = base.lexeme, .span = base.span },
                        .operator_suffix = .{ .text = suffix.lexeme, .span = suffix.span },
                        .span = ast.spanFromBounds(base.span.start, spanEnd(suffix.span)),
                    };
                },
                else => {},
            }
        }
        return .{
            .base = .{ .text = base.lexeme, .span = base.span },
            .span = base.span,
        };
    }

    fn parseParamDecl(self: *Parser, allocator: std.mem.Allocator) !?ast.ParamDecl {
        if (self.current().kind != .identifier and self.current().kind != .mut) {
            try self.report(.UnexpectedToken, "malformed signature", self.current().span);
            self.recoverSignatureParam();
            return null;
        }
        var type_name = try self.parseTypeName(allocator);
        errdefer type_name.deinit(allocator);
        const name_token = try self.expect(.identifier, "expected parameter name", .UnexpectedToken);
        const name = if (name_token) |identifier|
            ast.NameSegment{ .text = identifier.lexeme, .span = identifier.span }
        else
            ast.NameSegment{ .text = "", .span = self.current().span };
        return .{
            .type_name = type_name,
            .name = name,
            .span = ast.spanFromBounds(type_name.span.start, spanEnd(name.span)),
        };
    }

    fn recoverSignature(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .right_brace) self.advance();
        _ = self.match(.semicolon);
    }

    fn recoverSignatureParam(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .comma and self.current().kind != .right_paren and self.current().kind != .semicolon and self.current().kind != .right_brace) self.advance();
    }

    fn parseTypeName(self: *Parser, allocator: std.mem.Allocator) !ast.TypeName {
        const mut_token = self.match(.mut);
        const start_span = if (mut_token) |token| token.span else self.current().span;
        var name = try self.parseDottedName(allocator);
        errdefer name.deinit(allocator);
        const generic_args = try self.parseOptionalGenericTypeArgs(allocator);
        errdefer {
            for (generic_args) |arg| arg.deinit(allocator);
            allocator.free(generic_args);
        }
        var last_span = if (generic_args.len != 0) generic_args[generic_args.len - 1].span else name.span;
        const is_reference = self.match(.ampersand) != null;
        if (is_reference) last_span = self.peekBack().span;
        const is_pointer = self.match(.star) != null;
        if (is_pointer) last_span = self.peekBack().span;
        return .{
            .name = name,
            .generic_args = generic_args,
            .is_mut = mut_token != null,
            .is_reference = is_reference,
            .is_pointer = is_pointer,
            .span = ast.spanFromBounds(start_span.start, spanEnd(last_span)),
        };
    }

    fn parseOptionalGenericTypeArgs(self: *Parser, allocator: std.mem.Allocator) ![]ast.TypeName {
        var args = std.ArrayList(ast.TypeName).init(allocator);
        errdefer {
            for (args.items) |arg| arg.deinit(allocator);
            args.deinit();
        }
        if (self.match(.less) == null) return args.toOwnedSlice();
        while (self.current().kind != .eof and self.current().kind != .greater and self.current().kind != .right_paren and self.current().kind != .semicolon and self.current().kind != .left_brace and self.current().kind != .right_brace) {
            if (self.current().kind == .identifier or self.current().kind == .mut) {
                try args.append(try self.parseTypeName(allocator));
                if (self.match(.comma) != null) continue;
                if (self.current().kind == .greater) break;
                try self.report(.UnexpectedToken, "expected ',' between generic type arguments", self.current().span);
                self.advance();
                continue;
            }
            try self.report(.UnexpectedToken, "expected generic type argument", self.current().span);
            self.advance();
        }
        if (self.match(.greater) == null) {
            try self.report(.UnexpectedToken, "expected '>' after generic type arguments", self.current().span);
        }
        return args.toOwnedSlice();
    }

    fn emptyTypeName(self: *Parser, allocator: std.mem.Allocator, span: SourceSpan) !ast.TypeName {
        const parts = try allocator.alloc(ast.NameSegment, 0);
        _ = self;
        return .{ .name = .{ .parts = parts, .span = span }, .generic_args = try allocator.alloc(ast.TypeName, 0), .span = span };
    }

    fn expectTrailingBlockSemicolonAfterBrace(self: *Parser, decl_kind: []const u8, right_brace_span: SourceSpan) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;
        _ = decl_kind;
        try self.report(.UnexpectedToken, "expected ';' after declaration", self.current().span);
        return right_brace_span;
    }

    fn expectTrailingBlockSemicolon(self: *Parser, decl_kind: []const u8) !SourceSpan {
        if (self.match(.semicolon)) |semicolon| return semicolon.span;
        _ = decl_kind;
        try self.report(.UnexpectedToken, "expected ';' after declaration", self.current().span);
        return self.current().span;
    }

    fn peekBack(self: Parser) Token {
        if (self.index == 0) return self.current();
        return self.tokens[self.index - 1];
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

test "parses empty struct" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; struct Empty {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 1), unit.items.len);
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expectEqualStrings("Empty", struct_decl.name.text);
    try std.testing.expectEqual(@as(usize, 0), struct_decl.fields.len);
}

test "parses struct with one field" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; struct Vec1 { float x; };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expectEqual(@as(usize, 1), struct_decl.fields.len);
    try std.testing.expectEqualStrings("float", struct_decl.fields[0].type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("x", struct_decl.fields[0].name.text);
}

test "parses struct with multiple fields" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\struct Vec3 {
        \\    float x;
        \\    float y;
        \\    float z;
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expectEqual(@as(usize, 3), struct_decl.fields.len);
    try std.testing.expectEqualStrings("z", struct_decl.fields[2].name.text);
}

test "parses export struct" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; export struct SourceSpan { int start; int length; };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expect(struct_decl.is_export);
    try std.testing.expectEqualStrings("SourceSpan", struct_decl.name.text);
    try std.testing.expectEqual(@as(usize, 2), struct_decl.fields.len);
}

test "parses dotted field type" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; struct Holder { Qualified.Type value; };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expectEqual(@as(usize, 2), struct_decl.fields[0].type_name.name.parts.len);
    try std.testing.expectEqualStrings("Qualified", struct_decl.fields[0].type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("Type", struct_decl.fields[0].type_name.name.parts[1].text);
}

test "AST snapshot debug output includes structs" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\struct Vec3 {
        \\    float x;
        \\    float y;
        \\    float z;
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Struct Vec3
        \\    Field float x
        \\    Field float y
        \\    Field float z
        \\
    , snapshot);
}

test "missing struct name produces diagnostic" {
    try expectSingleDiagnostic("module Example; struct { int x; };", .UnexpectedToken);
}

test "missing opening brace produces diagnostic" {
    try expectSingleDiagnostic("module Example; struct Vec3 int x;", .UnexpectedToken);
}

test "missing field semicolon produces diagnostic and continues parsing fields" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; struct Vec2 { float x float y; };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expectEqual(@as(usize, 2), struct_decl.fields.len);
    try std.testing.expectEqualStrings("y", struct_decl.fields[1].name.text);
}

test "missing closing brace produces diagnostic" {
    try expectSingleDiagnostic("module Example; struct Vec1 { float x;", .UnexpectedToken);
}

test "missing trailing struct semicolon produces diagnostic" {
    try expectSingleDiagnostic("module Example; struct Vec1 { float x; }", .UnexpectedToken);
}

test "unexpected token inside struct body recovers" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; struct Vec1 { 123 float x; };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
    };
    try std.testing.expectEqual(@as(usize, 1), struct_decl.fields.len);
    try std.testing.expectEqualStrings("x", struct_decl.fields[0].name.text);
}

test "parses empty enum" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Empty {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 1), unit.items.len);
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqualStrings("Empty", enum_decl.name.text);
    try std.testing.expectEqual(@as(usize, 0), enum_decl.variants.len);
}

test "parses enum with one variant" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { End };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), enum_decl.variants.len);
    try std.testing.expectEqualStrings("End", enum_decl.variants[0].name.text);
}

test "parses enum with multiple variants" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier, Number, End };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 3), enum_decl.variants.len);
    try std.testing.expectEqualStrings("Number", enum_decl.variants[1].name.text);
}

test "parses enum with trailing comma" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier, Number, End, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 3), enum_decl.variants.len);
    try std.testing.expectEqualStrings("End", enum_decl.variants[2].name.text);
}

test "parses exported enum" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; export enum ParseError { ExpectedExpression(SourceSpan span), UnexpectedToken(Token token), };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expect(enum_decl.is_export);
    try std.testing.expectEqualStrings("ParseError", enum_decl.name.text);
    try std.testing.expectEqual(@as(usize, 2), enum_decl.variants.len);
}

test "parses enum with one payload variant" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier(StringView name), End };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), enum_decl.variants[0].payload_fields.len);
    try std.testing.expectEqualStrings("StringView", enum_decl.variants[0].payload_fields[0].type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("name", enum_decl.variants[0].payload_fields[0].name.text);
}

test "parses enum with multiple payload variants" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier(StringView name), Number(double value), Plus, Minus, End, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 5), enum_decl.variants.len);
    try std.testing.expectEqual(@as(usize, 1), enum_decl.variants[1].payload_fields.len);
    try std.testing.expectEqualStrings("double", enum_decl.variants[1].payload_fields[0].type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("value", enum_decl.variants[1].payload_fields[0].name.text);
}

test "parses enum payload field with dotted type" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Qualified(Qualified.Type value), End };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    const field = enum_decl.variants[0].payload_fields[0];
    try std.testing.expectEqual(@as(usize, 2), field.type_name.name.parts.len);
    try std.testing.expectEqualStrings("Qualified", field.type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("Type", field.type_name.name.parts[1].text);
    try std.testing.expectEqualStrings("value", field.name.text);
}

test "AST snapshot debug output includes plain enum" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\enum Token {
        \\    Identifier,
        \\    Number,
        \\    End,
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Enum Token
        \\    Variant Identifier
        \\    Variant Number
        \\    Variant End
        \\
    , snapshot);
}

test "AST snapshot debug output includes payload enum" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\enum Token {
        \\    Identifier(StringView name),
        \\    Number(double value),
        \\    Plus,
        \\    Minus,
        \\    End,
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Enum Token
        \\    Variant Identifier
        \\      Payload StringView name
        \\    Variant Number
        \\      Payload double value
        \\    Variant Plus
        \\    Variant Minus
        \\    Variant End
        \\
    , snapshot);
}

test "missing enum name produces diagnostic" {
    try expectSingleDiagnostic("module Example; enum { End };", .UnexpectedToken);
}

test "missing enum opening brace produces diagnostic" {
    try expectSingleDiagnostic("module Example; enum Token End;", .UnexpectedToken);
}

test "missing comma between enum variants produces diagnostic and continues parsing variants" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier Number, End, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 3), enum_decl.variants.len);
    try std.testing.expectEqualStrings("Number", enum_decl.variants[1].name.text);
}

test "malformed enum payload field produces diagnostic and recovers" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier(StringView, bad), Number(double value), End, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 2), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 3), enum_decl.variants.len);
    try std.testing.expectEqual(@as(usize, 0), enum_decl.variants[0].payload_fields.len);
    try std.testing.expectEqual(@as(usize, 1), enum_decl.variants[1].payload_fields.len);
}

test "missing closing enum payload paren produces diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier(StringView name, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), enum_decl.variants.len);
    try std.testing.expectEqual(@as(usize, 1), enum_decl.variants[0].payload_fields.len);
}

test "missing closing enum brace produces diagnostic" {
    try expectSingleDiagnostic("module Example; enum Token { Identifier, Number;", .UnexpectedToken);
}

test "missing trailing enum semicolon produces diagnostic" {
    try expectSingleDiagnostic("module Example; enum Token { End }", .UnexpectedToken);
}

test "unexpected token inside enum body recovers" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; enum Token { Identifier, 123, End, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 2), enum_decl.variants.len);
    try std.testing.expectEqualStrings("End", enum_decl.variants[1].name.text);
}

test "parses empty concept" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Empty {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqualStrings("Empty", concept_decl.name.text);
    try std.testing.expectEqual(@as(usize, 0), concept_decl.signatures.len);
}

test "parses concept with one generic parameter" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Hashable<T> {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), concept_decl.generic_params.len);
    try std.testing.expectEqualStrings("T", concept_decl.generic_params[0].text);
}

test "parses concept with multiple generic parameters" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Mapper<K, V> {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 2), concept_decl.generic_params.len);
    try std.testing.expectEqualStrings("K", concept_decl.generic_params[0].text);
    try std.testing.expectEqualStrings("V", concept_decl.generic_params[1].text);
}

test "parses concept with one signature" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Hashable<T> { uint64 hash(T& value); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), concept_decl.signatures.len);
    try std.testing.expectEqualStrings("hash", concept_decl.signatures[0].name.base.text);
    try std.testing.expect(concept_decl.signatures[0].params[0].type_name.is_reference);
}

test "parses concept with multiple signatures and operator signature" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Equatable<T> { bool equals(T& left, T& right); bool operator==(T& left, T& right); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 2), concept_decl.signatures.len);
    try std.testing.expectEqualStrings("equals", concept_decl.signatures[0].name.base.text);
    try std.testing.expectEqualStrings("operator", concept_decl.signatures[1].name.base.text);
    try std.testing.expectEqualStrings("==", concept_decl.signatures[1].name.operator_suffix.?.text);
}

test "parses interface with signatures" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; interface Renderer { void drawMesh(Mesh& mesh); void drawTexture(Texture& texture); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const interface_decl = switch (unit.items[0]) {
        .interface_decl => |interface_decl| interface_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 2), interface_decl.signatures.len);
    try std.testing.expectEqualStrings("drawTexture", interface_decl.signatures[1].name.base.text);
}

test "parses empty impl" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; impl Drop<Texture> {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const impl_decl = switch (unit.items[0]) {
        .impl_decl => |impl_decl| impl_decl,
        else => unreachable,
    };
    try std.testing.expectEqualStrings("Drop", impl_decl.target.name.parts[0].text);
    try std.testing.expectEqual(@as(usize, 1), impl_decl.target.generic_args.len);
    try std.testing.expectEqual(@as(usize, 0), impl_decl.signatures.len);
}

test "parses impl with generic target" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; impl Drop<Texture> { void drop(mut Texture& tex); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const impl_decl = switch (unit.items[0]) {
        .impl_decl => |impl_decl| impl_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), impl_decl.signatures.len);
    try std.testing.expect(impl_decl.signatures[0].params[0].type_name.is_mut);
    try std.testing.expect(impl_decl.signatures[0].params[0].type_name.is_reference);
}

test "parses impl with concept target" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; impl Hashable<Symbol> { uint64 hash(Symbol& symbol); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const impl_decl = switch (unit.items[0]) {
        .impl_decl => |impl_decl| impl_decl,
        else => unreachable,
    };
    try std.testing.expectEqualStrings("Hashable", impl_decl.target.name.parts[0].text);
    try std.testing.expectEqualStrings("Symbol", impl_decl.target.generic_args[0].name.parts[0].text);
}

test "parses signature with dotted type" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; interface AllocatorUser { void use(Core.Memory.Allocator allocator); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const interface_decl = switch (unit.items[0]) {
        .interface_decl => |interface_decl| interface_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 3), interface_decl.signatures[0].params[0].type_name.name.parts.len);
}

test "parses signature with generic return type" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; interface Parser { Result<Token, LexError> next(); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const interface_decl = switch (unit.items[0]) {
        .interface_decl => |interface_decl| interface_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 2), interface_decl.signatures[0].return_type.generic_args.len);
}

test "parses signature with reference parameter" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; interface Reader { void read(Texture& texture); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const interface_decl = switch (unit.items[0]) {
        .interface_decl => |interface_decl| interface_decl,
        else => unreachable,
    };
    try std.testing.expect(interface_decl.signatures[0].params[0].type_name.is_reference);
}

test "parses signature with mutable reference parameter" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; interface Writer { void write(mut Texture& texture); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const interface_decl = switch (unit.items[0]) {
        .interface_decl => |interface_decl| interface_decl,
        else => unreachable,
    };
    try std.testing.expect(interface_decl.signatures[0].params[0].type_name.is_mut);
    try std.testing.expect(interface_decl.signatures[0].params[0].type_name.is_reference);
}

test "AST snapshot debug output includes concept" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\concept Hashable<T> {
        \\    uint64 hash(T& value);
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Concept Hashable<T>
        \\    Signature uint64 hash(T& value)
        \\
    , snapshot);
}

test "AST snapshot debug output includes interface" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\interface Renderer {
        \\    void drawMesh(Mesh& mesh);
        \\    void drawTexture(Texture& texture);
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Interface Renderer
        \\    Signature void drawMesh(Mesh& mesh)
        \\    Signature void drawTexture(Texture& texture)
        \\
    , snapshot);
}

test "AST snapshot debug output includes impl" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\impl Drop<Texture> {
        \\    void drop(mut Texture& tex);
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Impl Drop<Texture>
        \\    Signature void drop(mut Texture& tex)
        \\
    , snapshot);
}

test "missing concept interface and impl name diagnostics" {
    try expectSingleDiagnostic("module Example; concept {};", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; interface {};", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; impl {};", .UnexpectedToken);
}

test "missing generic closing greater diagnostic" {
    try expectSingleDiagnostic("module Example; concept Hashable<T {};", .UnexpectedToken);
}

test "missing concept interface and impl opening brace diagnostics" {
    try expectSingleDiagnostic("module Example; concept Hashable<T> uint64 hash(T& value);", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; interface Renderer void drawMesh(Mesh& mesh);", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; impl Drop<Texture> void drop(mut Texture& tex);", .UnexpectedToken);
}

test "missing signature semicolon diagnostics" {
    try expectSingleDiagnostic("module Example; concept Hashable<T> { uint64 hash(T& value) };", .UnexpectedToken);
}

test "malformed signature recovery" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Hashable<T> { uint64 ; uint64 hash(T& value); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), concept_decl.signatures.len);
    try std.testing.expectEqualStrings("hash", concept_decl.signatures[0].name.base.text);
}

test "missing declaration closing brace diagnostics" {
    try expectSingleDiagnostic("module Example; concept Hashable<T> { uint64 hash(T& value);", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; interface Renderer { void drawMesh(Mesh& mesh);", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; impl Drop<Texture> { void drop(mut Texture& tex);", .UnexpectedToken);
}

test "missing trailing declaration semicolon diagnostics" {
    try expectSingleDiagnostic("module Example; concept Empty {}", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; interface Empty {}", .UnexpectedToken);
    try expectSingleDiagnostic("module Example; impl Empty {}", .UnexpectedToken);
}

test "unexpected token inside declaration block recovers" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; concept Hashable<T> { 123 uint64 hash(T& value); };", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    const concept_decl = switch (unit.items[0]) {
        .concept_decl => |concept_decl| concept_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), concept_decl.signatures.len);
    try std.testing.expectEqualStrings("hash", concept_decl.signatures[0].name.base.text);
}
