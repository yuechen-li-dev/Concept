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
    source_text: ?[]const u8 = null,
    index: usize = 0,

    pub fn init(tokens: []const Token, diagnostics: *DiagnosticBag) Parser {
        return .{
            .tokens = tokens,
            .diagnostics = diagnostics,
        };
    }

    pub fn initWithSource(tokens: []const Token, diagnostics: *DiagnosticBag, source_text: []const u8) Parser {
        return .{
            .tokens = tokens,
            .diagnostics = diagnostics,
            .source_text = source_text,
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
            var attributes = try self.parseAttributes(allocator);
            errdefer deinitAttributes(attributes, allocator);

            switch (self.current().kind) {
                .module => {
                    try self.rejectAttributesBeforeUnsupportedItem(attributes, allocator);
                    attributes = &.{};
                    try self.reportDuplicateModule();
                    const duplicate = try self.parseModuleDecl(allocator);
                    duplicate.deinit(allocator);
                },
                .@"struct" => {
                    try items.append(try self.parseStructItem(allocator, attributes));
                    attributes = &.{};
                },
                .identifier, .mut, .unsafe => {
                    if (self.current().kind == .unsafe and !(self.peek(1).kind == .identifier or self.peek(1).kind == .mut or self.peek(1).kind == .unsafe)) {
                        try self.rejectAttributesBeforeUnsupportedItem(attributes, allocator);
                        attributes = &.{};
                        try self.reportExpectedItem("unsafe modifier is only supported on functions", self.current().span);
                        self.advance();
                        continue;
                    }
                    if (try self.parseFunctionItem(allocator, attributes)) |item| {
                        try items.append(item);
                        attributes = &.{};
                    }
                },
                .@"enum", .must_use => {
                    if (self.current().kind == .must_use and self.peek(1).kind == .@"struct") {
                        try self.rejectAttributesBeforeUnsupportedItem(attributes, allocator);
                        attributes = &.{};
                        try self.report(.UnexpectedToken, "must_use is only supported on enum declarations", self.current().span);
                        self.advance();
                        continue;
                    }
                    try items.append(try self.parseEnumItem(allocator, attributes));
                    attributes = &.{};
                },
                .concept => {
                    try items.append(try self.parseConceptItem(allocator, attributes));
                    attributes = &.{};
                },
                .interface => {
                    try items.append(try self.parseInterfaceItem(allocator, attributes));
                    attributes = &.{};
                },
                .impl => {
                    try items.append(try self.parseImplItem(allocator, attributes));
                    attributes = &.{};
                },
                .@"export" => {
                    if (self.peek(1).kind == .@"struct") {
                        try items.append(try self.parseStructItem(allocator, attributes));
                        attributes = &.{};
                    } else if (self.peek(1).kind == .@"enum" or self.peek(1).kind == .must_use) {
                        try items.append(try self.parseEnumItem(allocator, attributes));
                        attributes = &.{};
                    } else if (self.peek(1).kind == .identifier or self.peek(1).kind == .mut or self.peek(1).kind == .unsafe) {
                        if (try self.parseFunctionItem(allocator, attributes)) |item| {
                            try items.append(item);
                            attributes = &.{};
                        }
                    } else {
                        try self.rejectAttributesBeforeUnsupportedItem(attributes, allocator);
                        attributes = &.{};
                        try self.reportExpectedItem("expected item declaration", self.current().span);
                        self.advance();
                    }
                },
                .import => {
                    try self.rejectAttributesBeforeUnsupportedItem(attributes, allocator);
                    attributes = &.{};
                    try self.reportExpectedItem("expected item declaration", self.current().span);
                    self.advance();
                },
                else => {
                    if (attributes.len != 0) {
                        try self.reportExpectedItem("expected item declaration after attribute", self.current().span);
                        deinitAttributes(attributes, allocator);
                        attributes = &.{};
                    } else {
                        try self.reportExpectedItem("expected item declaration", self.current().span);
                    }
                    self.advance();
                },
            }
            deinitAttributes(attributes, allocator);
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

    fn parseAttributes(self: *Parser, allocator: std.mem.Allocator) ![]ast.Attribute {
        var attributes = std.ArrayList(ast.Attribute).init(allocator);
        errdefer {
            for (attributes.items) |attribute| attribute.deinit(allocator);
            attributes.deinit();
        }

        while (self.current().kind == .left_bracket) {
            try attributes.append(try self.parseAttribute(allocator));
        }

        return attributes.toOwnedSlice();
    }

    fn parseAttribute(self: *Parser, allocator: std.mem.Allocator) !ast.Attribute {
        const left_bracket = self.advance();
        const name_start_span = self.current().span;
        var name = try self.parseAttributeName(allocator);
        errdefer name.deinit(allocator);

        var arguments: ?ast.AttributeArguments = null;
        errdefer if (arguments) |argument_list| argument_list.deinit(allocator);

        if (self.match(.left_paren)) |left_paren| {
            arguments = try self.parseAttributeArguments(allocator, left_paren.span);
        }

        const end_span = if (self.match(.right_bracket)) |right_bracket| right_bracket.span else blk: {
            try self.report(.UnexpectedToken, "expected ']' after attribute", self.current().span);
            self.recoverAttributeAfterMissingBracket();
            if (self.match(.right_bracket)) |right_bracket| break :blk right_bracket.span;
            break :blk if (arguments) |argument_list| argument_list.span else name_start_span;
        };

        return .{
            .name = name,
            .arguments = arguments,
            .span = ast.spanFromBounds(left_bracket.span.start, spanEnd(end_span)),
        };
    }

    fn parseAttributeName(self: *Parser, allocator: std.mem.Allocator) !ast.QualifiedName {
        if (self.current().kind != .identifier) {
            try self.report(.UnexpectedToken, "expected attribute name", self.current().span);
            const parts = try allocator.alloc(ast.NameSegment, 0);
            return .{ .parts = parts, .span = self.current().span };
        }
        return self.parseDottedName(allocator);
    }

    fn parseAttributeArguments(self: *Parser, allocator: std.mem.Allocator, left_paren_span: SourceSpan) !ast.AttributeArguments {
        const args_start = self.current().span.start;
        var args_end = args_start;
        var depth: usize = 1;
        var last_span = left_paren_span;
        while (self.current().kind != .eof and self.current().kind != .right_bracket) {
            const token = self.advance();
            last_span = token.span;
            switch (token.kind) {
                .left_paren => depth += 1,
                .right_paren => {
                    depth -= 1;
                    if (depth == 0) {
                        args_end = token.span.start;
                        const raw_text = try self.attributeArgumentText(allocator, args_start, args_end);
                        return .{
                            .text = raw_text,
                            .span = ast.spanFromBounds(left_paren_span.start, spanEnd(token.span)),
                        };
                    }
                },
                else => {},
            }
        }

        try self.report(.UnexpectedToken, "unterminated attribute argument list", self.current().span);
        const raw_end = if (last_span.start >= args_start) spanEnd(last_span) else args_start;
        const raw_text = try self.attributeArgumentText(allocator, args_start, raw_end);
        return .{
            .text = raw_text,
            .span = ast.spanFromBounds(left_paren_span.start, raw_end),
        };
    }

    fn attributeArgumentText(self: Parser, allocator: std.mem.Allocator, start: usize, end: usize) ![]const u8 {
        if (self.source_text) |source_text| {
            if (start <= end and end <= source_text.len) return allocator.dupe(u8, source_text[start..end]);
        }
        return allocator.dupe(u8, "");
    }

    fn recoverAttributeAfterMissingBracket(self: *Parser) void {
        while (self.current().kind != .eof) {
            switch (self.current().kind) {
                .right_bracket => return,
                .@"struct", .@"enum", .concept, .interface, .impl, .@"export", .identifier, .mut, .module, .import => return,
                else => self.advance(),
            }
        }
    }

    fn rejectAttributesBeforeUnsupportedItem(self: *Parser, attributes: []ast.Attribute, allocator: std.mem.Allocator) !void {
        if (attributes.len == 0) return;
        try self.reportExpectedItem("expected item declaration after attribute", self.current().span);
        deinitAttributes(attributes, allocator);
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

    fn parseFunctionItem(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !?ast.Item {
        if (try self.parseFunctionDecl(allocator, attributes)) |function_decl| {
            return .{ .function_decl = function_decl };
        }
        return null;
    }

    fn parseStructItem(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !ast.Item {
        var decl = try self.parseStructDecl(allocator);
        decl.attributes = attributes;
        decl.span = ast.spanFromBounds(itemStartWithAttributes(attributes, decl.span), spanEnd(decl.span));
        return .{ .struct_decl = decl };
    }

    fn parseEnumItem(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !ast.Item {
        var decl = try self.parseEnumDecl(allocator);
        decl.attributes = attributes;
        decl.span = ast.spanFromBounds(itemStartWithAttributes(attributes, decl.span), spanEnd(decl.span));
        return .{ .enum_decl = decl };
    }

    fn parseConceptItem(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !ast.Item {
        var decl = try self.parseConceptDecl(allocator);
        decl.attributes = attributes;
        decl.span = ast.spanFromBounds(itemStartWithAttributes(attributes, decl.span), spanEnd(decl.span));
        return .{ .concept_decl = decl };
    }

    fn parseInterfaceItem(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !ast.Item {
        var decl = try self.parseInterfaceDecl(allocator);
        decl.attributes = attributes;
        decl.span = ast.spanFromBounds(itemStartWithAttributes(attributes, decl.span), spanEnd(decl.span));
        return .{ .interface_decl = decl };
    }

    fn parseImplItem(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !ast.Item {
        var decl = try self.parseImplDecl(allocator);
        decl.attributes = attributes;
        decl.span = ast.spanFromBounds(itemStartWithAttributes(attributes, decl.span), spanEnd(decl.span));
        return .{ .impl_decl = decl };
    }

    fn parseFunctionDecl(self: *Parser, allocator: std.mem.Allocator, attributes: []ast.Attribute) !?ast.FunctionDecl {
        const export_token = self.match(.@"export");
        const first_unsafe_token = self.match(.unsafe);
        const start_span = if (export_token) |token| token.span else if (first_unsafe_token) |token| token.span else self.current().span;
        while (self.match(.unsafe)) |duplicate| {
            try self.report(.UnexpectedToken, "duplicate unsafe function modifier", duplicate.span);
        }

        var return_type = self.parseTypeName(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            else => unreachable,
        };
        errdefer return_type.deinit(allocator);

        const name = try self.parseSignatureName();
        if (name == null) {
            try self.report(.UnexpectedToken, "expected function name", self.current().span);
            try self.recoverFunctionDecl();
            return_type.deinit(allocator);
            return null;
        }

        if (self.match(.left_paren) == null) {
            try self.report(.UnexpectedToken, "expected '(' after function name", self.current().span);
            try self.recoverFunctionDecl();
            return_type.deinit(allocator);
            return null;
        }

        var params = std.ArrayList(ast.ParamDecl).init(allocator);
        errdefer {
            for (params.items) |param| param.deinit(allocator);
            params.deinit();
        }
        var last_span = name.?.span;
        while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .left_brace and self.current().kind != .semicolon) {
            if (try self.parseFunctionParamDecl(allocator)) |param| {
                last_span = param.span;
                try params.append(param);
            }
            if (self.match(.comma)) |comma| {
                last_span = comma.span;
                continue;
            }
            if (self.current().kind == .right_paren or self.current().kind == .left_brace or self.current().kind == .semicolon) break;
            try self.report(.UnexpectedToken, "expected ',' between function parameters", self.current().span);
            self.recoverFunctionParam();
            _ = self.match(.comma);
        }

        if (self.match(.right_paren)) |right_paren| {
            last_span = right_paren.span;
        } else {
            try self.report(.UnexpectedToken, "expected ')' after function parameter list", self.current().span);
            self.recoverFunctionAfterMissingRightParen();
        }

        var body: ?ast.FunctionBody = null;
        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            if (self.current().kind == .left_brace) {
                const parsed_body = try self.parseFunctionBody(allocator);
                const body_span = parsed_body.span;
                body = parsed_body;
                break :blk body_span;
            }
            try self.report(.UnexpectedToken, "expected ';' or function body after function declaration", self.current().span);
            try self.recoverFunctionDecl();
            break :blk last_span;
        };

        return .{
            .attributes = attributes,
            .is_export = export_token != null,
            .is_unsafe = first_unsafe_token != null,
            .signature = .{
                .return_type = return_type,
                .name = name.?,
                .params = try params.toOwnedSlice(),
                .span = ast.spanFromBounds(start_span.start, spanEnd(last_span)),
            },
            .body = body,
            .span = ast.spanFromBounds(itemStartWithAttributes(attributes, start_span), spanEnd(end_span)),
        };
    }

    fn parseFunctionParamDecl(self: *Parser, allocator: std.mem.Allocator) !?ast.ParamDecl {
        if (self.current().kind != .identifier and self.current().kind != .mut) {
            try self.report(.UnexpectedToken, "malformed function parameter", self.current().span);
            self.recoverFunctionParam();
            return null;
        }
        var type_name = try self.parseTypeName(allocator);
        errdefer type_name.deinit(allocator);
        const name_token = try self.expect(.identifier, "expected parameter name", .UnexpectedToken);
        if (name_token == null) {
            type_name.deinit(allocator);
            self.recoverFunctionParam();
            return null;
        }
        return .{
            .type_name = type_name,
            .name = .{ .text = name_token.?.lexeme, .span = name_token.?.span },
            .span = ast.spanFromBounds(type_name.span.start, spanEnd(name_token.?.span)),
        };
    }

    fn parseFunctionBody(self: *Parser, allocator: std.mem.Allocator) !ast.FunctionBody {
        const block = try self.parseRequiredBlock(allocator, "function body");
        return .{
            .span = block.span,
            .block = block,
        };
    }

    fn parseRequiredBlock(self: *Parser, allocator: std.mem.Allocator, context: []const u8) !ast.BlockStmt {
        _ = context;
        if (self.current().kind != .left_brace) {
            try self.report(.UnexpectedToken, "expected braced block", self.current().span);
            return .{ .statements = try allocator.alloc(ast.Stmt, 0), .span = self.current().span };
        }
        return self.parseBlockAfterOpenBrace(allocator);
    }

    fn parseBlockAfterOpenBrace(self: *Parser, allocator: std.mem.Allocator) !ast.BlockStmt {
        const open_brace = self.advance();
        var statements = std.ArrayList(ast.Stmt).init(allocator);
        errdefer {
            for (statements.items) |stmt| stmt.deinit(allocator);
            statements.deinit();
        }

        var last_span = open_brace.span;
        while (self.current().kind != .eof and self.current().kind != .right_brace) {
            const stmt = (try self.parseStmt(allocator)) orelse continue;
            last_span = stmtSpan(stmt);
            try statements.append(stmt);
        }

        const close_span = if (self.match(.right_brace)) |right_brace| right_brace.span else blk: {
            try self.report(.UnexpectedToken, "unterminated block", self.current().span);
            break :blk last_span;
        };

        return .{
            .statements = try statements.toOwnedSlice(),
            .span = ast.spanFromBounds(open_brace.span.start, spanEnd(close_span)),
        };
    }

    fn parseStmt(self: *Parser, allocator: std.mem.Allocator) !?ast.Stmt {
        if (self.current().kind == .@"return") return try self.parseReturnStmt(allocator);
        if (self.current().kind == .@"if") return try self.parseIfStmt(allocator);
        if (self.current().kind == .@"while") return try self.parseWhileStmt(allocator);
        if (self.current().kind == .unsafe) return try self.parseUnsafeBlockStmt(allocator);
        if (self.current().kind == .match) return try self.parseMatchStmt(allocator);
        if (self.current().kind == .discard) return try self.parseDiscardStmt(allocator);
        if (self.current().kind == .left_brace) return .{ .block_stmt = try self.parseBlockAfterOpenBrace(allocator) };
        if (self.isLocalDeclStart()) return try self.parseLocalDeclStmt(allocator);
        if (self.isAssignmentStmtStart()) return try self.parseAssignmentStmt(allocator);
        if (self.isExprStmtStart()) return try self.parseExprStmt(allocator);

        try self.report(.UnexpectedToken, "unsupported statement in function body", self.current().span);
        self.recoverStatement();
        return null;
    }

    fn parseDiscardStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const discard_token = self.advance();
        var value: *ast.Expr = undefined;
        var have_value = true;
        if (self.current().kind == .semicolon or self.current().kind == .right_brace or self.current().kind == .eof) {
            try self.report(.UnexpectedToken, "expected expression after discard", self.current().span);
            value = try allocator.create(ast.Expr);
            value.* = .{ .int_literal = .{ .text = "0", .span = discard_token.span } };
            have_value = false;
        } else {
            value = self.parseExpr(allocator) catch |err| switch (err) {
                error.OutOfMemory => return err,
                error.ParseFailed => blk: {
                    try self.report(.UnexpectedToken, "expected expression after discard", self.current().span);
                    const fallback = try allocator.create(ast.Expr);
                    fallback.* = .{ .int_literal = .{ .text = "0", .span = discard_token.span } };
                    have_value = false;
                    break :blk fallback;
                },
            };
        }
        errdefer {
            value.deinit(allocator);
            allocator.destroy(value);
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            try self.report(.UnexpectedToken, "expected ';' after discard statement", self.current().span);
            self.recoverStatement();
            break :blk if (have_value) value.span() else discard_token.span;
        };

        return .{ .discard_stmt = .{
            .value = value,
            .span = ast.spanFromBounds(discard_token.span.start, spanEnd(end_span)),
        } };
    }

    fn parseExprStmt(self: *Parser, allocator: std.mem.Allocator) !?ast.Stmt {
        var value = self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => {
                try self.report(.UnexpectedToken, "expected expression", self.current().span);
                self.recoverStatement();
                return null;
            },
        };
        errdefer {
            value.deinit(allocator);
            allocator.destroy(value);
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            try self.report(.UnexpectedToken, "expected ';' after expression statement", self.current().span);
            self.recoverStatement();
            break :blk value.span();
        };

        return .{ .expr_stmt = .{
            .value = value,
            .span = ast.spanFromBounds(value.span().start, spanEnd(end_span)),
        } };
    }

    fn parseIfStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const if_token = self.advance();

        if (self.match(.left_paren) == null) {
            try self.report(.UnexpectedToken, "expected '(' after 'if'", self.current().span);
        }

        var condition = self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => blk: {
                const fallback = try allocator.create(ast.Expr);
                fallback.* = .{ .bool_literal = .{ .value = false, .span = self.current().span } };
                break :blk fallback;
            },
        };
        errdefer {
            condition.deinit(allocator);
            allocator.destroy(condition);
        }

        if (self.match(.right_paren) == null) {
            try self.report(.UnexpectedToken, "expected ')' after if condition", self.current().span);
            self.recoverIfCondition();
            _ = self.match(.right_paren);
        }

        var then_block = if (self.current().kind == .left_brace)
            try self.parseBlockAfterOpenBrace(allocator)
        else blk: {
            try self.report(.UnexpectedToken, "expected braced block after if condition", self.current().span);
            break :blk ast.BlockStmt{ .statements = try allocator.alloc(ast.Stmt, 0), .span = self.current().span };
        };
        errdefer then_block.deinit(allocator);

        var else_block: ?ast.BlockStmt = null;
        errdefer if (else_block) |block| block.deinit(allocator);

        var end_span = then_block.span;
        if (self.match(.@"else")) |else_token| {
            if (self.current().kind == .@"if") {
                try self.report(.UnexpectedToken, "else if ladders are not supported; use match for multi-way branching", self.current().span);
                end_span = else_token.span;
            } else if (self.current().kind == .left_brace) {
                else_block = try self.parseBlockAfterOpenBrace(allocator);
                end_span = else_block.?.span;
            } else {
                try self.report(.UnexpectedToken, "expected braced block after else", self.current().span);
                end_span = else_token.span;
            }
        }

        return .{ .if_stmt = .{
            .condition = condition,
            .then_block = then_block,
            .else_block = else_block,
            .span = ast.spanFromBounds(if_token.span.start, spanEnd(end_span)),
        } };
    }

    fn parseWhileStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const while_token = self.advance();

        if (self.match(.left_paren) == null) {
            try self.report(.UnexpectedToken, "expected '(' after 'while'", self.current().span);
        }

        var condition = if (self.current().kind == .right_paren or self.current().kind == .left_brace) blk: {
            try self.report(.UnexpectedToken, "expected while condition", self.current().span);
            const fallback = try allocator.create(ast.Expr);
            fallback.* = .{ .bool_literal = .{ .value = false, .span = self.current().span } };
            break :blk fallback;
        } else self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => blk: {
                const fallback = try allocator.create(ast.Expr);
                fallback.* = .{ .bool_literal = .{ .value = false, .span = self.current().span } };
                break :blk fallback;
            },
        };
        errdefer {
            condition.deinit(allocator);
            allocator.destroy(condition);
        }

        if (self.match(.right_paren) == null) {
            try self.report(.UnexpectedToken, "expected ')' after while condition", self.current().span);
            self.recoverWhileCondition();
            _ = self.match(.right_paren);
        }

        var body = if (self.current().kind == .left_brace)
            try self.parseBlockAfterOpenBrace(allocator)
        else blk: {
            try self.report(.UnexpectedToken, "expected braced block after while condition", self.current().span);
            break :blk ast.BlockStmt{ .statements = try allocator.alloc(ast.Stmt, 0), .span = self.current().span };
        };
        errdefer body.deinit(allocator);

        return .{ .while_stmt = .{
            .condition = condition,
            .body = body,
            .span = ast.spanFromBounds(while_token.span.start, spanEnd(body.span)),
        } };
    }

    fn parseUnsafeBlockStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const unsafe_token = self.advance();
        var body = if (self.current().kind == .left_brace)
            try self.parseBlockAfterOpenBrace(allocator)
        else blk: {
            try self.report(.UnexpectedToken, "expected '{' after unsafe", self.current().span);
            break :blk ast.BlockStmt{ .statements = try allocator.alloc(ast.Stmt, 0), .span = self.current().span };
        };
        errdefer body.deinit(allocator);

        return .{ .unsafe_block = .{
            .body = body,
            .span = ast.spanFromBounds(unsafe_token.span.start, spanEnd(body.span)),
        } };
    }

    fn parseMatchStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const match_token = self.advance();

        if (self.match(.left_paren) == null) {
            try self.report(.UnexpectedToken, "expected '(' after 'match'", self.current().span);
        }

        var scrutinee = self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => blk: {
                const fallback = try allocator.create(ast.Expr);
                fallback.* = .{ .int_literal = .{ .text = "0", .span = self.current().span } };
                break :blk fallback;
            },
        };
        errdefer {
            scrutinee.deinit(allocator);
            allocator.destroy(scrutinee);
        }

        if (self.match(.right_paren) == null) {
            try self.report(.UnexpectedToken, "expected ')' after match scrutinee", self.current().span);
            self.recoverMatchHeader();
            _ = self.match(.right_paren);
        }

        if (self.match(.left_brace) == null) {
            try self.report(.UnexpectedToken, "expected '{' after match scrutinee", self.current().span);
            return .{ .match_stmt = .{
                .scrutinee = scrutinee,
                .arms = try allocator.alloc(ast.MatchArm, 0),
                .span = ast.spanFromBounds(match_token.span.start, spanEnd(scrutinee.span())),
            } };
        }

        var arms = std.ArrayList(ast.MatchArm).init(allocator);
        errdefer {
            for (arms.items) |arm| arm.deinit(allocator);
            arms.deinit();
        }

        var last_span = scrutinee.span();
        while (self.current().kind != .eof and self.current().kind != .right_brace) {
            if (try self.parseMatchArm(allocator)) |arm| {
                last_span = arm.span;
                try arms.append(arm);
            }
        }

        const close_span = if (self.match(.right_brace)) |right_brace| right_brace.span else blk: {
            try self.report(.UnexpectedToken, "unterminated match statement", self.current().span);
            break :blk last_span;
        };

        return .{ .match_stmt = .{
            .scrutinee = scrutinee,
            .arms = try arms.toOwnedSlice(),
            .span = ast.spanFromBounds(match_token.span.start, spanEnd(close_span)),
        } };
    }

    fn parseMatchArm(self: *Parser, allocator: std.mem.Allocator) !?ast.MatchArm {
        const pattern = (try self.parseMatchPattern(allocator)) orelse {
            self.recoverMatchArm();
            return null;
        };

        if (self.match(.fat_arrow) == null) {
            try self.report(.UnexpectedToken, "expected '=>' after match pattern", self.current().span);
            self.recoverMatchArm();
            return null;
        }

        const body = (try self.parseStmt(allocator)) orelse {
            try self.report(.UnexpectedToken, "expected statement after match arm '=>'", self.current().span);
            self.recoverMatchArm();
            return null;
        };

        return .{ .pattern = pattern, .body = body, .span = ast.spanFromBounds(pattern.span().start, spanEnd(stmtSpan(body))) };
    }

    fn parseMatchPattern(self: *Parser, allocator: std.mem.Allocator) !?ast.MatchPattern {
        const token = self.current();
        switch (token.kind) {
            .int_literal => {
                _ = self.advance();
                return .{ .int_literal = .{ .text = token.lexeme, .span = token.span } };
            },
            .true, .false => {
                _ = self.advance();
                return .{ .bool_literal = .{ .value = token.kind == .true, .span = token.span } };
            },
            .identifier => {
                _ = self.advance();
                if (std.mem.eql(u8, token.lexeme, "_")) {
                    return .{ .wildcard = token.span };
                }
                if (self.match(.colon_colon) != null) {
                    const variant = (try self.expect(.identifier, "expected enum variant after '::'", .UnexpectedToken)) orelse return null;
                    var bindings = std.ArrayList(ast.PatternBinding).init(allocator);
                    errdefer bindings.deinit();
                    var end_span = variant.span;
                    if (self.match(.left_paren)) |left_paren| {
                        end_span = left_paren.span;
                        while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .fat_arrow) {
                            const binding = self.current();
                            if (binding.kind != .identifier or std.mem.eql(u8, binding.lexeme, "_")) {
                                try self.report(.UnexpectedToken, "expected payload binding identifier", binding.span);
                                self.recoverPatternBindingList();
                                break;
                            }
                            _ = self.advance();
                            try bindings.append(.{ .name = .{ .text = binding.lexeme, .span = binding.span } });
                            end_span = binding.span;
                            if (self.match(.comma)) |comma| {
                                end_span = comma.span;
                                if (self.current().kind == .right_paren) break;
                                continue;
                            }
                            break;
                        }
                        if (self.match(.right_paren)) |right_paren| {
                            end_span = right_paren.span;
                        } else {
                            try self.report(.UnexpectedToken, "expected ')' after payload binding list", self.current().span);
                        }
                    }
                    return .{ .enum_variant = .{
                        .enum_name = .{ .text = token.lexeme, .span = token.span },
                        .variant_name = .{ .text = variant.lexeme, .span = variant.span },
                        .bindings = try bindings.toOwnedSlice(),
                        .span = ast.spanFromBounds(token.span.start, spanEnd(end_span)),
                    } };
                }
            },
            else => {},
        }
        try self.report(.UnexpectedToken, "expected match pattern", token.span);
        return null;
    }

    fn recoverPatternBindingList(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .fat_arrow) {
            _ = self.advance();
        }
    }

    fn skipBalancedParens(self: *Parser) void {
        var depth: usize = 0;
        while (self.current().kind != .eof) {
            const kind = self.current().kind;
            if (kind == .left_paren) depth += 1;
            _ = self.advance();
            if (kind == .right_paren) {
                if (depth == 0) return;
                depth -= 1;
                if (depth == 0) return;
            }
        }
    }

    fn parseLocalDeclStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        var type_name = try self.parseTypeName(allocator);
        errdefer type_name.deinit(allocator);

        const name_token = try self.expect(.identifier, "expected local variable name", .UnexpectedToken);
        const name = if (name_token) |identifier|
            ast.NameSegment{ .text = identifier.lexeme, .span = identifier.span }
        else
            ast.NameSegment{ .text = "", .span = self.current().span };

        if (self.match(.equal) == null) {
            try self.report(.UnexpectedToken, "expected '=' and initializer in local declaration", self.current().span);
            self.recoverStatement();
            const fallback = try allocator.create(ast.Expr);
            fallback.* = .{ .int_literal = .{ .text = "0", .span = name.span } };
            return .{ .local_decl = .{
                .type_name = type_name,
                .name = name,
                .initializer = fallback,
                .span = ast.spanFromBounds(type_name.span.start, spanEnd(name.span)),
            } };
        }

        var initializer = self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => blk: {
                const fallback = try allocator.create(ast.Expr);
                fallback.* = .{ .int_literal = .{ .text = "0", .span = self.current().span } };
                break :blk fallback;
            },
        };
        errdefer {
            initializer.deinit(allocator);
            allocator.destroy(initializer);
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            try self.report(.UnexpectedToken, "expected ';' after local declaration", self.current().span);
            self.recoverStatement();
            break :blk initializer.span();
        };

        return .{ .local_decl = .{
            .type_name = type_name,
            .name = name,
            .initializer = initializer,
            .span = ast.spanFromBounds(type_name.span.start, spanEnd(end_span)),
        } };
    }

    fn parseAssignmentStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const target_token = self.advance();
        const target = ast.NameSegment{ .text = target_token.lexeme, .span = target_token.span };

        if (self.match(.equal) == null) {
            try self.report(.UnexpectedToken, "expected '=' in assignment statement", self.current().span);
            self.recoverStatement();
            const fallback = try allocator.create(ast.Expr);
            fallback.* = .{ .int_literal = .{ .text = "0", .span = target.span } };
            return .{ .assignment = .{
                .target = target,
                .value = fallback,
                .span = ast.spanFromBounds(target.span.start, spanEnd(target.span)),
            } };
        }

        var value = self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => blk: {
                const fallback = try allocator.create(ast.Expr);
                fallback.* = .{ .int_literal = .{ .text = "0", .span = self.current().span } };
                break :blk fallback;
            },
        };
        errdefer {
            value.deinit(allocator);
            allocator.destroy(value);
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            try self.report(.UnexpectedToken, "expected ';' after assignment statement", self.current().span);
            self.recoverStatement();
            break :blk value.span();
        };

        return .{ .assignment = .{
            .target = target,
            .value = value,
            .span = ast.spanFromBounds(target.span.start, spanEnd(end_span)),
        } };
    }

    fn parseReturnStmt(self: *Parser, allocator: std.mem.Allocator) !ast.Stmt {
        const return_token = self.advance();
        var value: ?*ast.Expr = null;
        errdefer if (value) |expr| {
            expr.deinit(allocator);
            allocator.destroy(expr);
        };

        if (self.current().kind != .semicolon and self.current().kind != .right_brace and self.current().kind != .eof) {
            value = self.parseExpr(allocator) catch |err| switch (err) {
                error.OutOfMemory => return err,
                error.ParseFailed => null,
            };
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            try self.report(.UnexpectedToken, "expected ';' after return statement", self.current().span);
            self.recoverStatement();
            break :blk if (value) |expr| expr.span() else return_token.span;
        };

        return .{ .return_stmt = .{
            .value = value,
            .span = ast.spanFromBounds(return_token.span.start, spanEnd(end_span)),
        } };
    }

    const ParseExprError = error{ OutOfMemory, ParseFailed };

    fn parseExpr(self: *Parser, allocator: std.mem.Allocator) ParseExprError!*ast.Expr {
        return self.parseBinaryExpr(allocator, 1);
    }

    fn parseBinaryExpr(self: *Parser, allocator: std.mem.Allocator, min_precedence: u8) ParseExprError!*ast.Expr {
        var left = try self.parseUnaryExpr(allocator);
        errdefer {
            left.deinit(allocator);
            allocator.destroy(left);
        }

        while (binaryOp(self.current().kind)) |op_info| {
            if (op_info.precedence < min_precedence) break;
            _ = self.advance();
            var right = try self.parseBinaryExpr(allocator, op_info.precedence + 1);
            errdefer {
                right.deinit(allocator);
                allocator.destroy(right);
            }
            const node = try allocator.create(ast.Expr);
            node.* = .{ .binary = .{
                .op = op_info.op,
                .left = left,
                .right = right,
                .span = ast.spanFromBounds(left.span().start, spanEnd(right.span())),
            } };
            left = node;
        }

        return left;
    }

    fn parseUnaryExpr(self: *Parser, allocator: std.mem.Allocator) ParseExprError!*ast.Expr {
        if (self.current().kind == .@"try") {
            const try_token = self.advance();
            var operand = self.parseUnaryExpr(allocator) catch |err| switch (err) {
                error.OutOfMemory => return err,
                error.ParseFailed => {
                    self.report(.UnexpectedToken, "expected expression after try", self.current().span) catch return error.OutOfMemory;
                    return error.ParseFailed;
                },
            };
            errdefer {
                operand.deinit(allocator);
                allocator.destroy(operand);
            }
            const node = try allocator.create(ast.Expr);
            node.* = .{ .try_expr = .{
                .operand = operand,
                .span = ast.spanFromBounds(try_token.span.start, spanEnd(operand.span())),
            } };
            return node;
        }
        if (unaryOp(self.current().kind)) |op| {
            const op_token = self.advance();
            var operand = try self.parseUnaryExpr(allocator);
            errdefer {
                operand.deinit(allocator);
                allocator.destroy(operand);
            }
            const node = try allocator.create(ast.Expr);
            node.* = .{ .unary = .{
                .op = op,
                .operand = operand,
                .span = ast.spanFromBounds(op_token.span.start, spanEnd(operand.span())),
            } };
            return node;
        }
        return self.parsePostfixExpr(allocator);
    }

    fn parsePostfixExpr(self: *Parser, allocator: std.mem.Allocator) ParseExprError!*ast.Expr {
        var expr = try self.parsePrimaryExpr(allocator);
        errdefer {
            expr.deinit(allocator);
            allocator.destroy(expr);
        }

        while (self.current().kind == .left_paren) {
            const identifier = switch (expr.*) {
                .identifier => |identifier| identifier,
                else => break,
            };
            expr = try self.finishCallExpr(allocator, expr, identifier);
        }
        return expr;
    }

    fn finishCallExpr(self: *Parser, allocator: std.mem.Allocator, callee_expr: *ast.Expr, identifier: ast.Expr.IdentifierExpr) ParseExprError!*ast.Expr {
        const left_paren = self.advance();
        _ = left_paren;
        var args = std.ArrayList(*ast.Expr).init(allocator);
        errdefer {
            for (args.items) |arg| {
                arg.deinit(allocator);
                allocator.destroy(arg);
            }
            args.deinit();
        }

        var last_span = identifier.span;
        if (self.current().kind != .right_paren) {
            while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .semicolon) {
                const arg = self.parseExpr(allocator) catch |err| switch (err) {
                    error.OutOfMemory => return err,
                    error.ParseFailed => {
                        self.recoverCallArgument();
                        if (self.match(.comma) != null) continue;
                        break;
                    },
                };
                last_span = arg.span();
                try args.append(arg);
                if (self.match(.comma) == null) break;
                if (self.current().kind == .right_paren) {
                    self.report(.UnexpectedToken, "expected expression", self.current().span) catch return error.OutOfMemory;
                    break;
                }
            }
        }

        const end_span = if (self.match(.right_paren)) |right_paren| right_paren.span else blk: {
            self.report(.UnexpectedToken, "expected ')' after call arguments", self.current().span) catch return error.OutOfMemory;
            self.recoverCallArgument();
            break :blk last_span;
        };

        const owned_args = try args.toOwnedSlice();
        const node = try allocator.create(ast.Expr);
        callee_expr.deinit(allocator);
        allocator.destroy(callee_expr);
        node.* = .{ .call = .{
            .callee = identifier.name,
            .args = owned_args,
            .span = ast.spanFromBounds(identifier.span.start, spanEnd(end_span)),
        } };
        return node;
    }

    fn finishEnumConstructorExpr(self: *Parser, allocator: std.mem.Allocator, enum_name: ast.NameSegment) ParseExprError!*ast.Expr {
        _ = self.advance();
        const variant_token = if (self.current().kind == .identifier) self.advance() else {
            self.report(.UnexpectedToken, "expected enum variant after '::'", self.current().span) catch return error.OutOfMemory;
            return error.ParseFailed;
        };
        const variant_name = ast.NameSegment{ .text = variant_token.lexeme, .span = variant_token.span };
        var args = std.ArrayList(*ast.Expr).init(allocator);
        errdefer {
            for (args.items) |arg| {
                arg.deinit(allocator);
                allocator.destroy(arg);
            }
            args.deinit();
        }

        var end_span = variant_name.span;
        if (self.match(.left_paren)) |_| {
            if (self.current().kind != .right_paren) {
                while (self.current().kind != .eof and self.current().kind != .right_paren and self.current().kind != .semicolon) {
                    const arg = self.parseExpr(allocator) catch |err| switch (err) {
                        error.OutOfMemory => return err,
                        error.ParseFailed => {
                            self.recoverCallArgument();
                            if (self.match(.comma) != null) continue;
                            break;
                        },
                    };
                    end_span = arg.span();
                    try args.append(arg);
                    if (self.match(.comma) == null) break;
                    if (self.current().kind == .right_paren) {
                        self.report(.UnexpectedToken, "expected expression", self.current().span) catch return error.OutOfMemory;
                        break;
                    }
                }
            }
            end_span = if (self.match(.right_paren)) |right_paren| right_paren.span else blk: {
                self.report(.UnexpectedToken, "expected ')' after enum constructor arguments", self.current().span) catch return error.OutOfMemory;
                self.recoverCallArgument();
                break :blk end_span;
            };
        }

        const node = try allocator.create(ast.Expr);
        node.* = .{ .enum_constructor = .{
            .enum_name = enum_name,
            .variant_name = variant_name,
            .args = try args.toOwnedSlice(),
            .span = ast.spanFromBounds(enum_name.span.start, spanEnd(end_span)),
        } };
        return node;
    }

    fn parseDecideExpr(self: *Parser, allocator: std.mem.Allocator) ParseExprError!*ast.Expr {
        const decide_token = self.advance();
        var type_name = if (self.current().kind == .identifier or self.current().kind == .mut)
            self.parseTypeName(allocator) catch |err| switch (err) {
                error.OutOfMemory => return err,
            }
        else blk: {
            self.report(.UnexpectedToken, "expected decide target type", self.current().span) catch return error.OutOfMemory;
            break :blk self.emptyTypeName(allocator, self.current().span) catch return error.OutOfMemory;
        };
        errdefer type_name.deinit(allocator);

        var arms = std.ArrayList(ast.Expr.DecideArm).init(allocator);
        errdefer {
            for (arms.items) |arm| arm.deinit(allocator);
            arms.deinit();
        }

        if (self.match(.left_brace) == null) {
            self.report(.UnexpectedToken, "expected '{' after decide target type", self.current().span) catch return error.OutOfMemory;
            const node = try allocator.create(ast.Expr);
            node.* = .{ .decide = .{
                .type_name = type_name,
                .arms = try arms.toOwnedSlice(),
                .span = ast.spanFromBounds(decide_token.span.start, spanEnd(type_name.span)),
            } };
            return node;
        }

        var end_span = type_name.span;
        while (self.current().kind != .eof and self.current().kind != .right_brace) {
            if (try self.parseDecideArm(allocator)) |arm| {
                end_span = arm.span;
                try arms.append(arm);
            }
        }

        if (self.match(.right_brace)) |right_brace| {
            end_span = right_brace.span;
        } else {
            self.report(.UnexpectedToken, "expected '}' after decide arms", self.current().span) catch return error.OutOfMemory;
        }

        const node = try allocator.create(ast.Expr);
        node.* = .{ .decide = .{
            .type_name = type_name,
            .arms = try arms.toOwnedSlice(),
            .span = ast.spanFromBounds(decide_token.span.start, spanEnd(end_span)),
        } };
        return node;
    }

    fn parseDecideArm(self: *Parser, allocator: std.mem.Allocator) ParseExprError!?ast.Expr.DecideArm {
        const variant_token = if (self.current().kind == .identifier) self.advance() else blk: {
            self.report(.UnexpectedToken, "expected decide arm variant name", self.current().span) catch return error.OutOfMemory;
            self.recoverDecideArm();
            break :blk null;
        };
        const token = variant_token orelse return null;
        const variant_name = ast.NameSegment{ .text = token.lexeme, .span = token.span };

        var condition: ?*ast.Expr = null;
        errdefer if (condition) |condition_expr| {
            condition_expr.deinit(allocator);
            allocator.destroy(condition_expr);
        };

        if (self.match(.when) != null) {
            if (self.isContextualIdentifier("score") or self.current().kind == .semicolon or self.current().kind == .right_brace or self.current().kind == .eof) {
                self.report(.UnexpectedToken, "expected decide arm condition after when", self.current().span) catch return error.OutOfMemory;
            } else {
                condition = self.parseExpr(allocator) catch |err| switch (err) {
                    error.OutOfMemory => return err,
                    error.ParseFailed => null,
                };
            }
        }

        if (self.expectContextualIdentifier("score", "expected 'score' in decide arm") == null) {
            self.recoverDecideArm();
            return null;
        }

        var score = if (self.current().kind == .semicolon or self.current().kind == .right_brace or self.current().kind == .eof) blk: {
            self.report(.UnexpectedToken, "expected decide arm score expression", self.current().span) catch return error.OutOfMemory;
            const fallback = try allocator.create(ast.Expr);
            fallback.* = .{ .int_literal = .{ .text = "0", .span = self.current().span } };
            break :blk fallback;
        } else self.parseExpr(allocator) catch |err| switch (err) {
            error.OutOfMemory => return err,
            error.ParseFailed => blk: {
                self.report(.UnexpectedToken, "expected decide arm score expression", self.current().span) catch return error.OutOfMemory;
                const fallback = try allocator.create(ast.Expr);
                fallback.* = .{ .int_literal = .{ .text = "0", .span = self.current().span } };
                break :blk fallback;
            },
        };
        errdefer {
            score.deinit(allocator);
            allocator.destroy(score);
        }

        const end_span = if (self.match(.semicolon)) |semicolon| semicolon.span else blk: {
            self.report(.UnexpectedToken, "expected ';' after decide arm", self.current().span) catch return error.OutOfMemory;
            if (self.current().kind != .identifier and self.current().kind != .right_brace and self.current().kind != .eof) self.recoverDecideArm();
            break :blk score.span();
        };

        return .{
            .variant_name = variant_name,
            .condition = condition,
            .score = score,
            .span = ast.spanFromBounds(variant_name.span.start, spanEnd(end_span)),
        };
    }

    fn isContextualIdentifier(self: Parser, text: []const u8) bool {
        return self.current().kind == .identifier and std.mem.eql(u8, self.current().lexeme, text);
    }

    fn expectContextualIdentifier(self: *Parser, text: []const u8, message: []const u8) !?Token {
        if (self.isContextualIdentifier(text)) return self.advance();
        try self.report(.UnexpectedToken, message, self.current().span);
        return null;
    }

    fn recoverDecideArm(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .right_brace) {
            self.advance();
        }
        _ = self.match(.semicolon);
    }

    fn parsePrimaryExpr(self: *Parser, allocator: std.mem.Allocator) ParseExprError!*ast.Expr {
        const token = self.current();
        switch (token.kind) {
            .int_literal => {
                _ = self.advance();
                const node = try allocator.create(ast.Expr);
                node.* = .{ .int_literal = .{ .text = token.lexeme, .span = token.span } };
                return node;
            },
            .true, .false => {
                _ = self.advance();
                const node = try allocator.create(ast.Expr);
                node.* = .{ .bool_literal = .{ .value = token.kind == .true, .span = token.span } };
                return node;
            },
            .identifier => {
                _ = self.advance();
                const name = ast.NameSegment{ .text = token.lexeme, .span = token.span };
                if (self.current().kind == .colon_colon) {
                    return self.finishEnumConstructorExpr(allocator, name);
                }
                const node = try allocator.create(ast.Expr);
                node.* = .{ .identifier = .{
                    .name = name,
                    .span = token.span,
                } };
                return node;
            },
            .decide => return self.parseDecideExpr(allocator),
            .left_paren => {
                const left_paren = self.advance();
                var inner = self.parseExpr(allocator) catch |err| switch (err) {
                    error.OutOfMemory => return err,
                    error.ParseFailed => {
                        self.report(.UnexpectedToken, "expected expression", self.current().span) catch return error.OutOfMemory;
                        return error.ParseFailed;
                    },
                };
                errdefer {
                    inner.deinit(allocator);
                    allocator.destroy(inner);
                }
                const right_span = if (self.match(.right_paren)) |right_paren| right_paren.span else blk: {
                    self.report(.UnexpectedToken, "expected ')' after parenthesized expression", self.current().span) catch return error.OutOfMemory;
                    break :blk inner.span();
                };
                const node = try allocator.create(ast.Expr);
                node.* = .{ .group = .{
                    .inner = inner,
                    .span = ast.spanFromBounds(left_paren.span.start, spanEnd(right_span)),
                } };
                return node;
            },
            else => {
                self.report(.UnexpectedToken, "expected expression", token.span) catch return error.OutOfMemory;
                return error.ParseFailed;
            },
        }
    }

    fn recoverStatement(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .right_brace) {
            self.advance();
        }
        _ = self.match(.semicolon);
    }

    fn recoverMatchHeader(self: *Parser) void {
        while (self.current().kind != .eof and
            self.current().kind != .right_paren and
            self.current().kind != .left_brace and
            self.current().kind != .right_brace and
            self.current().kind != .semicolon)
        {
            self.advance();
        }
    }

    fn recoverMatchArm(self: *Parser) void {
        while (self.current().kind != .eof and
            self.current().kind != .semicolon and
            self.current().kind != .right_brace)
        {
            self.advance();
        }
        _ = self.match(.semicolon);
    }

    fn recoverWhileCondition(self: *Parser) void {
        self.recoverIfCondition();
    }

    fn recoverIfCondition(self: *Parser) void {
        while (self.current().kind != .eof and
            self.current().kind != .right_paren and
            self.current().kind != .left_brace and
            self.current().kind != .right_brace and
            self.current().kind != .semicolon)
        {
            self.advance();
        }
    }

    fn recoverCallArgument(self: *Parser) void {
        while (self.current().kind != .eof and
            self.current().kind != .comma and
            self.current().kind != .right_paren and
            self.current().kind != .semicolon)
        {
            self.advance();
        }
    }

    fn skipFunctionBody(self: *Parser) void {
        const open_brace = self.advance();
        _ = open_brace;
        var depth: usize = 1;
        while (self.current().kind != .eof) {
            const token = self.advance();
            switch (token.kind) {
                .left_brace => depth += 1,
                .right_brace => {
                    depth -= 1;
                    if (depth == 0) return;
                },
                else => {},
            }
        }
    }

    fn recoverFunctionDecl(self: *Parser) !void {
        while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .left_brace) self.advance();
        if (self.match(.semicolon) != null) return;
        if (self.current().kind == .left_brace) self.skipFunctionBody();
    }

    fn recoverFunctionParam(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .comma and self.current().kind != .right_paren and self.current().kind != .left_brace and self.current().kind != .semicolon) self.advance();
    }

    fn recoverFunctionAfterMissingRightParen(self: *Parser) void {
        while (self.current().kind != .eof and self.current().kind != .semicolon and self.current().kind != .left_brace) self.advance();
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
        const export_token = if (self.match(.@"export")) |token| token else null;
        const must_use_token = if (self.match(.must_use)) |token| token else null;
        if (must_use_token != null and self.current().kind == .must_use) {
            try self.report(.UnexpectedToken, "duplicate must_use modifier", self.current().span);
            _ = self.advance();
        }
        const enum_token = (try self.expect(.@"enum", if (must_use_token != null) "expected 'enum' after must_use" else "expected 'enum' after export", .ExpectedItem)) orelse (must_use_token orelse export_token).?;
        const start_span = if (export_token) |token| token.span else if (must_use_token) |token| token.span else enum_token.span;

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
                    .is_export = export_token != null,
                    .is_must_use = must_use_token != null,
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
            .is_export = export_token != null,
            .is_must_use = must_use_token != null,
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

    fn isAssignmentStmtStart(self: Parser) bool {
        return self.current().kind == .identifier and
            (self.peek(1).kind == .equal or self.peek(1).kind == .semicolon);
    }

    fn isExprStmtStart(self: Parser) bool {
        return switch (self.current().kind) {
            .identifier, .int_literal, .true, .false, .left_paren, .minus, .bang, .decide => true,
            else => false,
        };
    }

    fn isLocalDeclStart(self: Parser) bool {
        if (self.current().kind != .identifier) return false;
        var offset: usize = 1;
        while (self.peek(offset).kind == .dot and self.peek(offset + 1).kind == .identifier) {
            offset += 2;
        }
        if (self.peek(offset).kind != .identifier) return false;
        const after_name = self.peek(offset + 1).kind;
        return after_name == .equal or after_name == .semicolon;
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

fn stmtSpan(stmt: ast.Stmt) SourceSpan {
    return switch (stmt) {
        .local_decl => |local_decl| local_decl.span,
        .assignment => |assignment| assignment.span,
        .expr_stmt => |expr_stmt| expr_stmt.span,
        .discard_stmt => |discard_stmt| discard_stmt.span,
        .return_stmt => |return_stmt| return_stmt.span,
        .if_stmt => |if_stmt| if_stmt.span,
        .while_stmt => |while_stmt| while_stmt.span,
        .unsafe_block => |unsafe_block| unsafe_block.span,
        .match_stmt => |match_stmt| match_stmt.span,
        .block_stmt => |block_stmt| block_stmt.span,
    };
}

fn deinitAttributes(attributes: []ast.Attribute, allocator: std.mem.Allocator) void {
    for (attributes) |attribute| attribute.deinit(allocator);
    allocator.free(attributes);
}

const BinaryOpInfo = struct {
    op: ast.BinaryOp,
    precedence: u8,
};

fn binaryOp(kind: TokenKind) ?BinaryOpInfo {
    return switch (kind) {
        .pipe_pipe => .{ .op = .logical_or, .precedence = 1 },
        .ampersand_ampersand => .{ .op = .logical_and, .precedence = 2 },
        .equal_equal => .{ .op = .equal_equal, .precedence = 3 },
        .bang_equal => .{ .op = .bang_equal, .precedence = 3 },
        .less => .{ .op = .less, .precedence = 4 },
        .less_equal => .{ .op = .less_equal, .precedence = 4 },
        .greater => .{ .op = .greater, .precedence = 4 },
        .greater_equal => .{ .op = .greater_equal, .precedence = 4 },
        .plus => .{ .op = .add, .precedence = 5 },
        .minus => .{ .op = .subtract, .precedence = 5 },
        .star => .{ .op = .multiply, .precedence = 6 },
        .slash => .{ .op = .divide, .precedence = 6 },
        .percent => .{ .op = .modulo, .precedence = 6 },
        else => null,
    };
}

fn unaryOp(kind: TokenKind) ?ast.UnaryOp {
    return switch (kind) {
        .minus => .negate,
        .bang => .logical_not,
        else => null,
    };
}

fn itemStartWithAttributes(attributes: []const ast.Attribute, item_span: SourceSpan) usize {
    if (attributes.len == 0) return item_span.start;
    return attributes[0].span.start;
}

pub fn parseTokens(allocator: std.mem.Allocator, tokens: []const Token, diagnostics: *DiagnosticBag) !ast.CompilationUnit {
    var parser = Parser.init(tokens, diagnostics);
    return parser.parseCompilationUnit(allocator);
}

pub fn parseSource(allocator: std.mem.Allocator, source_file: SourceFile, diagnostics: *DiagnosticBag) !ast.CompilationUnit {
    const tokens = try lexer_model.lexAll(allocator, source_file, diagnostics);
    defer allocator.free(tokens);
    var parser = Parser.initWithSource(tokens, diagnostics, source_file.text);
    return parser.parseCompilationUnit(allocator);
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

fn expectCorpusSnapshot(comptime source_path: []const u8, comptime expected_path: []const u8) !void {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const source_text = @embedFile(source_path);
    const source_file = try SourceFile.init(std.testing.allocator, source_path, source_text);
    defer source_file.deinit(std.testing.allocator);

    const unit = try parseSource(std.testing.allocator, source_file, &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(@embedFile(expected_path), snapshot);
}

test "Phase 1 corpus snapshot: basic module" {
    try expectCorpusSnapshot(
        "../../tests/corpus/phase1/basic_module.concept",
        "../../tests/corpus/phase1/basic_module.ast.expected",
    );
}

test "Phase 1 corpus snapshot: type surface" {
    try expectCorpusSnapshot(
        "../../tests/corpus/phase1/types.concept",
        "../../tests/corpus/phase1/types.ast.expected",
    );
}

test "Phase 1 corpus snapshot: compiler shape" {
    try expectCorpusSnapshot(
        "../../tests/corpus/phase1/compiler_shape.concept",
        "../../tests/corpus/phase1/compiler_shape.ast.expected",
    );
}

test "Phase 1 corpus snapshot: test surface" {
    try expectCorpusSnapshot(
        "../../tests/corpus/phase1/test_surface.con_test",
        "../../tests/corpus/phase1/test_surface.ast.expected",
    );
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
        else => unreachable,
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
        else => unreachable,
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
        else => unreachable,
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
        else => unreachable,
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
        else => unreachable,
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
        else => unreachable,
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
        else => unreachable,
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

fn expectFunctionDecl(unit: ast.CompilationUnit, index: usize) ast.FunctionDecl {
    return switch (unit.items[index]) {
        .function_decl => |function_decl| function_decl,
        else => unreachable,
    };
}

test "parses top-level function declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int add(int a, int b);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(!function_decl.is_export);
    try std.testing.expect(function_decl.body == null);
    try std.testing.expectEqualStrings("int", function_decl.signature.return_type.name.parts[0].text);
    try std.testing.expectEqualStrings("add", function_decl.signature.name.base.text);
    try std.testing.expectEqual(@as(usize, 2), function_decl.signature.params.len);
}

test "parses exported function declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; export Result<Token, LexError> nextToken(mut Lexer& lexer);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.is_export);
    try std.testing.expectEqualStrings("Result", function_decl.signature.return_type.name.parts[0].text);
    try std.testing.expectEqual(@as(usize, 2), function_decl.signature.return_type.generic_args.len);
    try std.testing.expect(function_decl.signature.params[0].type_name.is_mut);
    try std.testing.expect(function_decl.signature.params[0].type_name.is_reference);
}

test "parses function with no parameters" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int main();", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 0), function_decl.signature.params.len);
}

test "parses function with multiple parameters" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int add(int a, int b);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 2), function_decl.signature.params.len);
    try std.testing.expectEqualStrings("a", function_decl.signature.params[0].name.text);
    try std.testing.expectEqualStrings("b", function_decl.signature.params[1].name.text);
}

test "parses function with generic return type" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; Result<Token, LexError> nextToken(Lexer lexer);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 2), function_decl.signature.return_type.generic_args.len);
}

test "parses function with reference parameter" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; void draw(Mesh& mesh);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.signature.params[0].type_name.is_reference);
}

test "parses function with mutable reference parameter" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; Token nextToken(mut Lexer& lexer);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.signature.params[0].type_name.is_mut);
    try std.testing.expect(function_decl.signature.params[0].type_name.is_reference);
}

test "parses function with raw pointer parameter" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; void fill(byte* buffer);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.signature.params[0].type_name.is_pointer);
}

test "rejects nested raw pointer syntax in type positions for now" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; void fill(int** buffer);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() != 0);
}

test "parses function with parsed body" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int main() { return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.body != null);
}

test "unsupported nested body statement diagnoses and recovers" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const source = "module Example; int main() { if true { return 1; } return 0; } int next();";
    const unit = try parseTestSource(source, &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() > 0);
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.body != null);
}

test "AST snapshot debug output for function declaration is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int add(int a, int b);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Function int add(int a, int b)
        \\
    , snapshot);
}

test "AST snapshot debug output for exported function declaration is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; export Result<Token, LexError> nextToken(mut Lexer& lexer);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Export Function Result<Token, LexError> nextToken(mut Lexer& lexer)
        \\
    , snapshot);
}

test "AST snapshot debug output for body capture is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int main() { return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example
        \\  Function int main()
        \\    Body
        \\      Return
        \\        Int 0
        \\
    , snapshot);
}

test "missing function name diagnostic" {
    try expectSingleDiagnostic("module Example; int ();", .UnexpectedToken);
}

test "missing function opening paren diagnostic" {
    try expectSingleDiagnostic("module Example; int add;", .UnexpectedToken);
}

test "malformed function parameter recovery" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int add(int, int b);", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 1), function_decl.signature.params.len);
    try std.testing.expectEqualStrings("b", function_decl.signature.params[0].name.text);
}

test "missing function closing paren diagnostic" {
    try expectSingleDiagnostic("module Example; int add(int a;", .UnexpectedToken);
}

test "missing function semicolon or body diagnostic" {
    try expectSingleDiagnostic("module Example; int add() export", .UnexpectedToken);
}

test "unterminated function body diagnostic" {
    try expectSingleDiagnostic("module Example; int main() { return 0;", .UnexpectedToken);
}

test "unknown top-level identifier recovery still works" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; unknown; struct StillParsed {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 1), unit.items.len);
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
        else => unreachable,
    };
    try std.testing.expectEqualStrings("StillParsed", struct_decl.name.text);
}

test "parses single attribute before function" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; [Fact] void AddsIntegers() { return; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 1), function_decl.attributes.len);
    try std.testing.expectEqualStrings("Fact", function_decl.attributes[0].name.parts[0].text);
    try std.testing.expect(function_decl.attributes[0].arguments == null);
}

test "parses multiple attributes before function" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; [Theory] [InlineData] void AddsValues();", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 2), function_decl.attributes.len);
    try std.testing.expectEqualStrings("Theory", function_decl.attributes[0].name.parts[0].text);
    try std.testing.expectEqualStrings("InlineData", function_decl.attributes[1].name.parts[0].text);
}

test "parses attribute with argument tokens" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; [InlineData(1, value, (2 + 3))] void AddsValues();", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 1), function_decl.attributes.len);
    try std.testing.expect(function_decl.attributes[0].arguments != null);
    try std.testing.expectEqualStrings("1, value, (2 + 3)", function_decl.attributes[0].arguments.?.text);
}

test "parses multiple InlineData attributes" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\[Theory]
        \\[InlineData(1, 2, 3)]
        \\[InlineData(10, 20, 30)]
        \\void AddsValues(int a, int b, int expected) { return; }
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 3), function_decl.attributes.len);
    try std.testing.expectEqualStrings("1, 2, 3", function_decl.attributes[1].arguments.?.text);
    try std.testing.expectEqualStrings("10, 20, 30", function_decl.attributes[2].arguments.?.text);
}

test "Fact Theory and InlineData remain identifiers" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const source_file = try SourceFile.init(std.testing.allocator, "test.con", "Fact Theory InlineData");
    defer source_file.deinit(std.testing.allocator);
    const tokens = try lexer_model.lexAll(std.testing.allocator, source_file, &diagnostics);
    defer std.testing.allocator.free(tokens);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(TokenKind.identifier, tokens[0].kind);
    try std.testing.expectEqual(TokenKind.identifier, tokens[1].kind);
    try std.testing.expectEqual(TokenKind.identifier, tokens[2].kind);
}

test "parses attribute before struct" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; [SomeMetadata] struct Thing {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
        else => unreachable,
    };
    try std.testing.expectEqual(@as(usize, 1), struct_decl.attributes.len);
    try std.testing.expectEqualStrings("SomeMetadata", struct_decl.attributes[0].name.parts[0].text);
}

test "AST snapshot debug output for Fact function is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example.Tests; [Fact] void AddsIntegers() { return; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example.Tests
        \\  Attribute Fact
        \\  Function void AddsIntegers()
        \\    Body
        \\      Return
        \\
    , snapshot);
}

test "AST snapshot debug output for Theory InlineData function is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example.Tests;
        \\[Theory]
        \\[InlineData(1, 2, 3)]
        \\[InlineData(10, 20, 30)]
        \\void AddsValues(int a, int b, int expected) { return; }
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Example.Tests
        \\  Attribute Theory
        \\  Attribute InlineData(1, 2, 3)
        \\  Attribute InlineData(10, 20, 30)
        \\  Function void AddsValues(int a, int b, int expected)
        \\    Body
        \\      Return
        \\
    , snapshot);
}

test "missing attribute name diagnostic" {
    try expectSingleDiagnostic("module Example; [] void Test();", .UnexpectedToken);
}

test "missing closing attribute bracket diagnostic" {
    try expectSingleDiagnostic("module Example; [Fact void Test();", .UnexpectedToken);
}

test "unterminated attribute argument list diagnostic" {
    try expectSingleDiagnostic("module Example; [InlineData(1, 2] void Test();", .UnexpectedToken);
}

test "attribute with no following item diagnostic" {
    try expectSingleDiagnostic("module Example; [Fact]", .ExpectedItem);
}

test "malformed attribute recovery followed by valid item" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; [Fact struct StillParsed {};", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 1), unit.items.len);
    const struct_decl = switch (unit.items[0]) {
        .struct_decl => |struct_decl| struct_decl,
        else => unreachable,
    };
    try std.testing.expectEqualStrings("StillParsed", struct_decl.name.text);
    try std.testing.expectEqual(@as(usize, 1), struct_decl.attributes.len);
}

test "parses dotted attribute name" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; [Test.Fact] void Test();", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expectEqual(@as(usize, 2), function_decl.attributes[0].name.parts.len);
    try std.testing.expectEqualStrings("Test", function_decl.attributes[0].name.parts[0].text);
    try std.testing.expectEqualStrings("Fact", function_decl.attributes[0].name.parts[1].text);
}

test "parses return integer literal body" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const function_decl = expectFunctionDecl(unit, 0);
    try std.testing.expect(function_decl.body != null);
    try std.testing.expectEqual(@as(usize, 1), function_decl.body.?.block.?.statements.len);
}

test "parses return without expression" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; void main() { return; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const return_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .return_stmt => |return_stmt| return_stmt,
    };
    try std.testing.expect(return_stmt.value == null);
}

test "parses empty function body" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; void f() { }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 0), expectFunctionDecl(unit, 0).body.?.block.?.statements.len);
}

test "AST snapshot debug output for arithmetic precedence is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return 1 + 2 * 3; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Main
        \\  Function int main()
        \\    Body
        \\      Return
        \\        Binary +
        \\          Int 1
        \\          Binary *
        \\            Int 2
        \\            Int 3
        \\
    , snapshot);
}

test "AST snapshot debug output for parenthesized expression is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return (1 + 2) * 3; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Main
        \\  Function int main()
        \\    Body
        \\      Return
        \\        Binary *
        \\          Group
        \\            Binary +
        \\              Int 1
        \\              Int 2
        \\          Int 3
        \\
    , snapshot);
}

test "parses unary minus" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return -1; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Unary -") != null);
}

test "parses unary bang" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; bool main() { return !true; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Unary !") != null);
}

test "parses comparison operators" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; bool main() { return 1 < 2 <= 3 > 4 >= 5; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary >=") != null);
}

test "parses equality operators" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; bool main() { return true == false != true; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
}

test "AST snapshot debug output for logical precedence is stable" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; bool main() { return true || false && true; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Main
        \\  Function bool main()
        \\    Body
        \\      Return
        \\        Binary ||
        \\          Bool true
        \\          Binary &&
        \\            Bool false
        \\            Bool true
        \\
    , snapshot);
}

test "missing return semicolon diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return 0 }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqualStrings("expected ';' after return statement", diagnostics.diagnostics.items[0].message);
}

test "expected expression diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return +; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected expression", diagnostics.diagnostics.items[0].message);
}

test "missing closing parenthesis diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return (1 + 2; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqualStrings("expected ')' after parenthesized expression", diagnostics.diagnostics.items[0].message);
}

test "unsupported statement recovery" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { foo(); return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(@as(usize, 1), expectFunctionDecl(unit, 0).body.?.block.?.statements.len);
}

test "parses assignment statement" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1; x = 2; return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const assignment = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[1]) {
        .assignment => |assignment| assignment,
        else => return error.ExpectedAssignment,
    };
    try std.testing.expectEqualStrings("x", assignment.target.text);
    _ = switch (assignment.value.*) {
        .int_literal => |literal| literal,
        else => return error.ExpectedIntLiteral,
    };
}

test "parses assignment using binary expression" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1; x = x + 2; return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const assignment = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[1]) {
        .assignment => |assignment| assignment,
        else => return error.ExpectedAssignment,
    };
    _ = switch (assignment.value.*) {
        .binary => |binary| binary,
        else => return error.ExpectedBinaryExpr,
    };
}

test "parses assignment to bool local" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { bool ok = true; ok = !ok; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const assignment = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[1]) {
        .assignment => |assignment| assignment,
        else => return error.ExpectedAssignment,
    };
    try std.testing.expectEqualStrings("ok", assignment.target.text);
    _ = switch (assignment.value.*) {
        .unary => |unary| unary,
        else => return error.ExpectedUnaryExpr,
    };
}

test "assignment missing equal diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1; x; return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqualStrings("expected '=' in assignment statement", diagnostics.diagnostics.items[0].message);
}

test "assignment missing expression diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1; x = ; return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqualStrings("expected expression", diagnostics.diagnostics.items[0].message);
}

test "assignment missing semicolon diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1; x = 2 return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqualStrings("expected ';' after assignment statement", diagnostics.diagnostics.items[0].message);
}

test "parses local int declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1 + 2; return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const stmt = expectFunctionDecl(unit, 0).body.?.block.?.statements[0];
    const local_decl = switch (stmt) {
        .local_decl => |local_decl| local_decl,
        else => return error.ExpectedLocalDecl,
    };
    try std.testing.expectEqualStrings("int", local_decl.type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("x", local_decl.name.text);
}

test "parses local bool declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { bool ok = true; return ok; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const local_decl = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .local_decl => |local_decl| local_decl,
        else => return error.ExpectedLocalDecl,
    };
    try std.testing.expectEqualStrings("bool", local_decl.type_name.name.parts[0].text);
    try std.testing.expectEqualStrings("ok", local_decl.name.text);
}

test "parses return identifier" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return x + 1; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const return_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .return_stmt => |return_stmt| return_stmt,
        else => return error.ExpectedReturnStmt,
    };
    const binary = switch (return_stmt.value.?.*) {
        .binary => |binary| binary,
        else => return error.ExpectedBinaryExpr,
    };
    const identifier = switch (binary.left.*) {
        .identifier => |identifier| identifier,
        else => return error.ExpectedIdentifierExpr,
    };
    try std.testing.expectEqualStrings("x", identifier.name.text);
}

test "parses if without else" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if (true) { return 1; } return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const if_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .if_stmt => |if_stmt| if_stmt,
        else => return error.ExpectedIfStmt,
    };
    try std.testing.expect(if_stmt.else_block == null);
    try std.testing.expectEqual(@as(usize, 1), if_stmt.then_block.statements.len);
}

test "parses if with else" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if (false) { return 1; } else { return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const if_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .if_stmt => |if_stmt| if_stmt,
        else => return error.ExpectedIfStmt,
    };
    try std.testing.expect(if_stmt.else_block != null);
    try std.testing.expectEqual(@as(usize, 1), if_stmt.else_block.?.statements.len);
}

test "parses nested if" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if (true) { if (false) { return 1; } } return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const outer = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .if_stmt => |if_stmt| if_stmt,
        else => return error.ExpectedIfStmt,
    };
    _ = switch (outer.then_block.statements[0]) {
        .if_stmt => |inner| inner,
        else => return error.ExpectedIfStmt,
    };
}

test "if missing paren diagnostics" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if true) { return 1; } return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected '(' after 'if'", diagnostics.diagnostics.items[0].message);
}

test "if missing closing paren diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if (true { return 1; } return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected ')' after if condition", diagnostics.diagnostics.items[0].message);
}

test "if missing then block diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if (true) return 1; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected braced block after if condition", diagnostics.diagnostics.items[0].message);
}

test "malformed else recovery" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { if (false) { return 1; } else return 0; int next(); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected braced block after else", diagnostics.diagnostics.items[0].message);
}

test "local declaration missing initializer diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
}

test "local declaration missing semicolon diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int x = 1 return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
}

test "parses call expression with no args" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return add(); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call add") != null);
}

test "parses call expression with one arg" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return add(1); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const call = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .return_stmt => |return_stmt| switch (return_stmt.value.?.*) {
            .call => |call| call,
            else => return error.ExpectedCallExpr,
        },
        else => return error.ExpectedReturnStmt,
    };
    try std.testing.expectEqualStrings("add", call.callee.text);
    try std.testing.expectEqual(@as(usize, 1), call.args.len);
}

test "parses call expression with multiple args" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return add(1, 2); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const call = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .return_stmt => |return_stmt| switch (return_stmt.value.?.*) {
            .call => |call| call,
            else => return error.ExpectedCallExpr,
        },
        else => return error.ExpectedReturnStmt,
    };
    try std.testing.expectEqual(@as(usize, 2), call.args.len);
}

test "parses call nested in binary expression" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return add(1, 2) + 3; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary +") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call add") != null);
}

test "call expression missing closing paren diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return add(1, 2; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected ')' after call arguments", diagnostics.diagnostics.items[0].message);
}

test "call expression malformed argument recovery" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return add(1, +, 2); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected expression", diagnostics.diagnostics.items[0].message);
    try std.testing.expectEqual(@as(usize, 1), expectFunctionDecl(unit, 0).body.?.block.?.statements.len);
}

test "parses match with int literal arms" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (1) { 1 => return 10; 2 => return 7; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const match_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .match_stmt => |match_stmt| match_stmt,
        else => return error.ExpectedMatchStmt,
    };
    try std.testing.expectEqual(@as(usize, 2), match_stmt.arms.len);
    _ = switch (match_stmt.arms[0].pattern) {
        .int_literal => |literal| literal,
        else => return error.ExpectedIntPattern,
    };
}

test "parses match with bool literal arms" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (true) { true => return 1; false => return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const match_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .match_stmt => |match_stmt| match_stmt,
        else => return error.ExpectedMatchStmt,
    };
    try std.testing.expectEqual(@as(usize, 2), match_stmt.arms.len);
    _ = switch (match_stmt.arms[0].pattern) {
        .bool_literal => |literal| literal,
        else => return error.ExpectedBoolPattern,
    };
}

test "parses match with wildcard arm" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (2) { _ => return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const match_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .match_stmt => |match_stmt| match_stmt,
        else => return error.ExpectedMatchStmt,
    };
    _ = switch (match_stmt.arms[0].pattern) {
        .wildcard => |span| span,
        else => return error.ExpectedWildcardPattern,
    };
}

test "parses enum variant match pattern" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; enum Status { Ok, Err, }; int main() { Status status = Status::Ok; match (status) { Status::Ok => return 7; Status::Err => return 1; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const match_stmt = switch (expectFunctionDecl(unit, 1).body.?.block.?.statements[1]) {
        .match_stmt => |match_stmt| match_stmt,
        else => return error.ExpectedMatchStmt,
    };
    const pattern = switch (match_stmt.arms[0].pattern) {
        .enum_variant => |pattern| pattern,
        else => return error.ExpectedEnumVariantPattern,
    };
    try std.testing.expectEqualStrings("Status", pattern.enum_name.text);
    try std.testing.expectEqualStrings("Ok", pattern.variant_name.text);
}

test "enum variant match pattern missing variant diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (0) { Status:: => return 0; _ => return 1; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected enum variant after '::'", diagnostics.diagnostics.items[0].message);
}

test "parses enum variant match payload bindings" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (0) { ParseResult::Ok(value, code) => return 0; _ => return 1; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const match_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .match_stmt => |match_stmt| match_stmt,
        else => return error.ExpectedMatchStmt,
    };
    const pattern = switch (match_stmt.arms[0].pattern) {
        .enum_variant => |pattern| pattern,
        else => return error.ExpectedEnumVariantPattern,
    };
    try std.testing.expectEqual(@as(usize, 2), pattern.bindings.len);
    try std.testing.expectEqualStrings("value", pattern.bindings[0].name.text);
}

test "enum variant match payload binding diagnostics" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (0) { ParseResult::Ok(1) => return 0; _ => return 1; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected payload binding identifier", diagnostics.diagnostics.items[0].message);
}

test "match missing opening brace diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (1) 1 => return 0; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected '{' after match scrutinee", diagnostics.diagnostics.items[0].message);
}

test "match missing fat arrow diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (1) { 1 return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected '=>' after match pattern", diagnostics.diagnostics.items[0].message);
}

test "match malformed pattern diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { match (1) { + => return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected match pattern", diagnostics.diagnostics.items[0].message);
}

test "direct else-if is rejected" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { if (true) { return 1; } else if (false) { return 2; } return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("else if ladders are not supported; use match for multi-way branching", diagnostics.diagnostics.items[0].message);
}

test "explicit nested if in else block remains valid" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { if (true) { return 1; } else { if (false) { return 2; } } return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
}

test "parses while loop" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { while (true) { return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const while_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .while_stmt => |while_stmt| while_stmt,
        else => return error.ExpectedWhileStmt,
    };
    try std.testing.expectEqual(@as(usize, 1), while_stmt.body.statements.len);
}

test "parses while loop with assignment body" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { int x = 0; while (x < 7) { x = x + 1; } return x; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const while_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[1]) {
        .while_stmt => |while_stmt| while_stmt,
        else => return error.ExpectedWhileStmt,
    };
    _ = switch (while_stmt.body.statements[0]) {
        .assignment => |assignment| assignment,
        else => return error.ExpectedAssignment,
    };
}

test "parses nested while loops" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { while (true) { while (false) { return 0; } } return 1; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const outer = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .while_stmt => |while_stmt| while_stmt,
        else => return error.ExpectedWhileStmt,
    };
    _ = switch (outer.body.statements[0]) {
        .while_stmt => |while_stmt| while_stmt,
        else => return error.ExpectedWhileStmt,
    };
}

test "while missing opening paren diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { while true) { return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected '(' after 'while'", diagnostics.diagnostics.items[0].message);
}

test "while missing condition diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { while () { return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected while condition", diagnostics.diagnostics.items[0].message);
}

test "while missing closing paren diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { while (true { return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected ')' after while condition", diagnostics.diagnostics.items[0].message);
}

test "while missing body diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();
    const unit = try parseTestSource("module Main; int main() { while (true) return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected braced block after while condition", diagnostics.diagnostics.items[0].message);
}

test "parses enum constructor expressions" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; enum Status { Ok, Err, }; Status main() { return Status::Ok; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor Status::Ok") != null);
}

test "parses enum constructor arguments" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; enum ParseResult { Ok(int value), Err(int code), }; ParseResult main() { return ParseResult::Ok(7); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor ParseResult::Ok") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Int 7") != null);
}

test "enum constructor missing variant diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return Status::; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected enum variant after '::'", diagnostics.diagnostics.items[0].message);
}

test "enum constructor malformed arguments diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { return Status::Ok(1, ); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected expression", diagnostics.diagnostics.items[0].message);
}

test "parses must_use enum declaration and debug output" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Example;
        \\must_use enum ParseResult {
        \\    Ok(int value),
        \\    Err(int code),
        \\};
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const enum_decl = switch (unit.items[0]) {
        .enum_decl => |enum_decl| enum_decl,
        else => return error.ExpectedEnumDecl,
    };
    try std.testing.expect(enum_decl.is_must_use);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "MustUse Enum ParseResult") != null);
}

test "must_use struct is rejected" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; must_use struct Box { int value; };", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("must_use is only supported on enum declarations", diagnostics.diagnostics.items[0].message);
}

test "duplicate must_use enum modifier is rejected" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; must_use must_use enum ParseResult { Ok, };", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("duplicate must_use modifier", diagnostics.diagnostics.items[0].message);
}

test "parses discard statements" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; enum Status { Ok, }; int make(); int main() { discard make(); discard Status::Ok; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const statements = expectFunctionDecl(unit, 2).body.?.block.?.statements;
    _ = switch (statements[0]) {
        .discard_stmt => |discard_stmt| discard_stmt,
        else => return error.ExpectedDiscardStmt,
    };
    _ = switch (statements[1]) {
        .discard_stmt => |discard_stmt| discard_stmt,
        else => return error.ExpectedDiscardStmt,
    };
}

test "discard missing expression diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { discard; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected expression after discard", diagnostics.diagnostics.items[0].message);
}

test "discard missing semicolon diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { discard 1 return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected ';' after discard statement", diagnostics.diagnostics.items[0].message);
}

test "discarded remains an identifier in local declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int discarded = 1; return discarded; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);
    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
}

test "parses try expression forms" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Main;
        \\int main() {
        \\    int value = try parseInt(1);
        \\    return ParseIntResult::Ok(try parseInt(1 + try parseInt(2)));
        \\}
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Try\n") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary +") != null);
}

test "missing try operand diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int value = try; return value; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected expression after try", diagnostics.diagnostics.items[0].message);
}

test "trying remains an identifier" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int trying = 1; return trying; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
}

test "parses decide expression with one unconditional arm" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main() { return decide AlertChannel { Nominal score 0; }; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const stmt = expectFunctionDecl(unit, 0).body.?.block.?.statements[0];
    const return_stmt = switch (stmt) {
        .return_stmt => |return_stmt| return_stmt,
        else => return error.ExpectedReturnStmt,
    };
    const decide = switch (return_stmt.value.?.*) {
        .decide => |decide| decide,
        else => return error.ExpectedDecideExpr,
    };
    try std.testing.expectEqualStrings("AlertChannel", decide.type_name.name.parts[0].text);
    try std.testing.expectEqual(@as(usize, 1), decide.arms.len);
    try std.testing.expectEqualStrings("Nominal", decide.arms[0].variant_name.text);
    try std.testing.expect(decide.arms[0].condition == null);
}

test "parses decide expression with conditional arm" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main(bool fault, int temperature) { return decide AlertChannel { Critical when fault && temperature > 900 score 120; Nominal score 0; }; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Decide AlertChannel") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Arm Critical") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "When") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary &&") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Score") != null);
}

test "parses decide expression with multiple arms" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Main;
        \\AlertChannel main(bool fault, int temperature, int pressure, int threshold) {
        \\    return decide AlertChannel {
        \\        Critical when fault && temperature > 900 score 120;
        \\        Warning when temperature > 750 score 85;
        \\        Advisory when pressure > threshold score 70;
        \\        Nominal score 0;
        \\    };
        \\}
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const return_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .return_stmt => |return_stmt| return_stmt,
        else => return error.ExpectedReturnStmt,
    };
    const decide = switch (return_stmt.value.?.*) {
        .decide => |decide| decide,
        else => return error.ExpectedDecideExpr,
    };
    try std.testing.expectEqual(@as(usize, 4), decide.arms.len);
    try std.testing.expectEqualStrings("Warning", decide.arms[1].variant_name.text);
    try std.testing.expectEqualStrings("Nominal", decide.arms[3].variant_name.text);
}

test "parses decide expression in local initializer" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { AlertChannel channel = decide AlertChannel { Nominal score 0; }; return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const local = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .local_decl => |local| local,
        else => return error.ExpectedLocalDecl,
    };
    _ = switch (local.initializer.*) {
        .decide => |decide| decide,
        else => return error.ExpectedDecideExpr,
    };
}

test "parses decide expression as function call argument" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int use(AlertChannel channel); int main() { return use(decide AlertChannel { Nominal score 0; }); }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call use") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Decide AlertChannel") != null);
}

test "score remains valid identifier outside decide context" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; int main() { int score = 0; return score; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
}

test "decide missing score marker diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main(bool fault) { return decide AlertChannel { Critical when fault 120; Nominal score 0; }; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected 'score' in decide arm", diagnostics.diagnostics.items[0].message);
}

test "decide missing score expression diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main() { return decide AlertChannel { Nominal score; }; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected decide arm score expression", diagnostics.diagnostics.items[0].message);
}

test "decide missing semicolon diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main() { return decide AlertChannel { Nominal score 0 Critical score 1; }; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected ';' after decide arm", diagnostics.diagnostics.items[0].message);
}

test "decide malformed arm recovers to following arm" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main() { return decide AlertChannel { 123 score 9; Nominal score 0; }; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    const return_stmt = switch (expectFunctionDecl(unit, 0).body.?.block.?.statements[0]) {
        .return_stmt => |return_stmt| return_stmt,
        else => return error.ExpectedReturnStmt,
    };
    const decide = switch (return_stmt.value.?.*) {
        .decide => |decide| decide,
        else => return error.ExpectedDecideExpr,
    };
    try std.testing.expectEqual(@as(usize, 1), decide.arms.len);
    try std.testing.expectEqualStrings("Nominal", decide.arms[0].variant_name.text);
}

test "decide missing closing brace diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Main; AlertChannel main() { return decide AlertChannel { Nominal score 0; ", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expect(diagnostics.count() >= 1);
    try std.testing.expectEqualStrings("expected '}' after decide arms", diagnostics.diagnostics.items[0].message);
}

test "AST snapshot debug output includes decide expression" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource(
        \\module Main;
        \\AlertChannel main(bool fault) {
        \\    return decide AlertChannel {
        \\        Critical when fault score 120;
        \\        Nominal score 0;
        \\    };
        \\}
    , &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\CompilationUnit
        \\  Module Main
        \\  Function AlertChannel main
        \\    Param bool fault
        \\    Body
        \\      Return
        \\        Decide AlertChannel
        \\          Arm Critical
        \\            When
        \\              Identifier fault
        \\            Score
        \\              Int 120
        \\          Arm Nominal
        \\            Score
        \\              Int 0
        \\
    , snapshot);
}

test "parses unsafe block statement" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int main() { unsafe { return 0; } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    const stmt = expectFunctionDecl(unit, 0).body.?.block.?.statements[0];
    const unsafe_block = switch (stmt) {
        .unsafe_block => |unsafe_block| unsafe_block,
        else => return error.ExpectedUnsafeBlock,
    };
    try std.testing.expectEqual(@as(usize, 1), unsafe_block.body.statements.len);
}

test "parses nested unsafe blocks and debug output" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int main() { unsafe { unsafe { return 0; } } }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    const snapshot = try unit.debugString(std.testing.allocator);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "UnsafeBlock") != null);
}

test "unsafe block requires braced block" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; int main() { unsafe return 0; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqual(DiagnosticCode.UnexpectedToken, diagnostics.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("expected '{' after unsafe", diagnostics.diagnostics.items[0].message);
}

test "parses unsafe function declaration" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; unsafe int helper() { return 1; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 0), diagnostics.count());
    try std.testing.expect(expectFunctionDecl(unit, 0).is_unsafe);
}

test "duplicate unsafe function modifier diagnostic" {
    var diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics.deinit();

    const unit = try parseTestSource("module Example; unsafe unsafe int helper() { return 1; }", &diagnostics);
    defer unit.deinit(std.testing.allocator);

    try std.testing.expectEqual(@as(usize, 1), diagnostics.count());
    try std.testing.expectEqualStrings("duplicate unsafe function modifier", diagnostics.diagnostics.items[0].message);
}
