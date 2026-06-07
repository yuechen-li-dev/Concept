// ─────────────────────────────────────────────────────────────────────────────
// Public API / SemanticModule
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const ast = @import("ast.zig");
const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const interner = @import("interner.zig");
const source = @import("source.zig");
const types = @import("types.zig");

pub const DiagnosticBag = diagnostics.DiagnosticBag;
pub const DiagnosticCode = diagnostics.DiagnosticCode;
pub const SemanticError = error{InvalidSemanticModule};

pub const SemanticModule = struct {
    interner: interner.Interner,
    hir: hir.HirStore,
    types: types.TypeStore,

    pub fn init(allocator: std.mem.Allocator) !SemanticModule {
        return .{
            .interner = interner.Interner.init(allocator),
            .hir = hir.HirStore.init(allocator),
            .types = try types.TypeStore.init(allocator),
        };
    }

    pub fn deinit(self: *SemanticModule) void {
        self.types.deinit();
        self.hir.deinit();
        self.interner.deinit();
        self.* = undefined;
    }

    pub fn resultShapeForType(self: *const SemanticModule, type_id: types.TypeId) ?hir.HirResultShape {
        return switch (self.types.kind(type_id)) {
            .enum_type => |enum_id| self.hir.getResultShape(enum_id),
            else => null,
        };
    }
};

// ─────────────────────────────────────────────────────────────────────────────
// Top-level collection
// ─────────────────────────────────────────────────────────────────────────────

pub fn collectTopLevelDeclarations(
    allocator: std.mem.Allocator,
    unit: ast.CompilationUnit,
    diagnostic_bag: *DiagnosticBag,
) !SemanticModule {
    var module = try SemanticModule.init(allocator);
    errdefer module.deinit();

    var collector = Collector.init(allocator, &module, diagnostic_bag);
    defer collector.deinit();

    try collector.collect(unit);
    if (diagnostic_bag.count() != 0) return error.InvalidSemanticModule;
    return module;
}

const Collector = struct {
    allocator: std.mem.Allocator,
    module: *SemanticModule,
    diagnostics: *DiagnosticBag,
    top_level_decls: std.AutoHashMap(interner.SymbolId, TopLevelDecl),

    const TopLevelDecl = union(enum) {
        function: hir.FunctionId,
        generic_function: hir.GenericFunctionId,
        struct_: struct {
            id: hir.StructId,
            type_id: types.TypeId,
        },
        enum_: struct {
            id: hir.EnumId,
            type_id: types.TypeId,
        },
    };

    fn init(allocator: std.mem.Allocator, module: *SemanticModule, diagnostic_bag: *DiagnosticBag) Collector {
        return .{
            .allocator = allocator,
            .module = module,
            .diagnostics = diagnostic_bag,
            .top_level_decls = std.AutoHashMap(interner.SymbolId, TopLevelDecl).init(allocator),
        };
    }

    fn deinit(self: *Collector) void {
        self.top_level_decls.deinit();
    }

    fn collect(self: *Collector, unit: ast.CompilationUnit) !void {
        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.declareFunction(function_decl),
                .template_decl => |template_decl| try self.declareGenericFunction(template_decl),
                .struct_decl => |struct_decl| try self.declareStruct(struct_decl),
                .enum_decl => |enum_decl| try self.declareEnum(enum_decl),
                .concept_decl, .interface_decl, .impl_decl => {},
            }
        }

        if (self.diagnostics.count() != 0) return;

        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.resolveFunction(function_decl),
                .template_decl => |template_decl| try self.resolveGenericFunction(template_decl),
                .struct_decl => |struct_decl| try self.resolveStruct(struct_decl),
                .enum_decl => |enum_decl| try self.resolveEnum(enum_decl),
                .concept_decl, .interface_decl, .impl_decl => {},
            }
        }

        if (self.diagnostics.count() != 0) return;

        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.lowerFunctionBody(function_decl),
                .template_decl => |template_decl| try self.lowerGenericFunctionBody(template_decl),
                .struct_decl, .enum_decl, .concept_decl, .interface_decl, .impl_decl => {},
            }
        }
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Declaration resolution
    // ─────────────────────────────────────────────────────────────────────────────

    fn declareFunction(self: *Collector, function_decl: ast.FunctionDecl) !void {
        const name = try self.internFreshTopLevelName(
            function_decl.signature.name.base.text,
            function_decl.signature.name.base.span,
        ) orelse return;
        const function_id = try self.module.hir.addFunctionWithSafety(name, self.module.types.voidType(), function_decl.is_unsafe, function_decl.span);
        try self.top_level_decls.put(name, .{ .function = function_id });
    }

    fn declareGenericFunction(self: *Collector, template_decl: ast.TemplateDecl) !void {
        const name = try self.internFreshTopLevelName(
            template_decl.body.signature.name.base.text,
            template_decl.body.signature.name.base.span,
        ) orelse return;
        const generic_id = try self.module.hir.addGenericFunction(name, template_decl.span);
        try self.top_level_decls.put(name, .{ .generic_function = generic_id });
    }

    fn declareStruct(self: *Collector, struct_decl: ast.StructDecl) !void {
        const name = try self.internFreshTopLevelName(struct_decl.name.text, struct_decl.name.span) orelse return;
        const struct_id = try self.module.hir.addStruct(name);
        const type_id = try self.module.types.addStructType(struct_id);
        try self.top_level_decls.put(name, .{ .struct_ = .{ .id = struct_id, .type_id = type_id } });
    }

    fn declareEnum(self: *Collector, enum_decl: ast.EnumDecl) !void {
        const name = try self.internFreshTopLevelName(enum_decl.name.text, enum_decl.name.span) orelse return;
        const enum_id = try self.module.hir.addEnum(name, enum_decl.is_must_use);
        const type_id = try self.module.types.addEnumType(enum_id);
        try self.top_level_decls.put(name, .{ .enum_ = .{ .id = enum_id, .type_id = type_id } });
    }

    fn resolveFunction(self: *Collector, function_decl: ast.FunctionDecl) !void {
        const function_symbol = try self.module.interner.intern(function_decl.signature.name.base.text);
        const function_id = switch (self.top_level_decls.get(function_symbol).?) {
            .function => |id| id,
            else => unreachable,
        };

        if (try self.resolveTypeName(function_decl.signature.return_type)) |return_type| {
            self.module.hir.setFunctionReturnType(function_id, return_type);
        }

        var param_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer param_names.deinit();

        for (function_decl.signature.params) |param| {
            const param_symbol = try self.module.interner.intern(param.name.text);
            if (param_names.contains(param_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateParameterName(param.name.span));
                continue;
            }
            try param_names.put(param_symbol, param.name.span);

            if (try self.resolveTypeName(param.type_name)) |type_id| {
                _ = try self.module.hir.addParam(function_id, param_symbol, type_id, param.span);
            }
        }
    }

    fn resolveGenericFunction(self: *Collector, template_decl: ast.TemplateDecl) !void {
        const function_symbol = try self.module.interner.intern(template_decl.body.signature.name.base.text);
        const generic_id = switch (self.top_level_decls.get(function_symbol).?) {
            .generic_function => |id| id,
            else => unreachable,
        };
        const generic = self.module.hir.getGenericFunction(generic_id);
        const function_id = generic.function;
        const owner = types.TypeParamOwner{ .kind = .generic_function, .index = generic_id.index };

        var type_scope = TypeParamScope.init(self.allocator);
        defer type_scope.deinit();
        var hir_params = std.ArrayList(hir.HirTypeParam).empty;
        var transferred_type_params = false;
        defer hir_params.deinit(self.allocator);
        errdefer if (!transferred_type_params) {
            for (hir_params.items) |param| if (param.constraint) |constraint| self.allocator.free(constraint.text);
        };

        for (template_decl.params, 0..) |param, index| {
            const symbol = try self.module.interner.intern(param.name.text);
            const type_id = try self.module.types.addTypeParam(owner, @intCast(index), symbol);
            try type_scope.put(symbol, type_id);
            const constraint = if (param.constraint) |constraint_type| blk: {
                break :blk hir.HirTypeConstraint{ .text = try self.renderTypeName(constraint_type), .span = constraint_type.span };
            } else null;
            try hir_params.append(self.allocator, .{ .name = symbol, .span = param.span, .type_id = type_id, .constraint = constraint });
        }

        const owned_type_params = try self.allocator.alloc(hir.HirTypeParam, hir_params.items.len);
        @memcpy(owned_type_params, hir_params.items);
        hir_params.clearRetainingCapacity();
        self.module.hir.setGenericFunctionTypeParams(generic_id, owned_type_params);
        transferred_type_params = true;

        if (try self.resolveTypeNameScoped(template_decl.body.signature.return_type, &type_scope)) |return_type| {
            self.module.hir.setFunctionReturnType(function_id, return_type);
        }

        var param_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer param_names.deinit();

        for (template_decl.body.signature.params) |param| {
            const param_symbol = try self.module.interner.intern(param.name.text);
            if (param_names.contains(param_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateParameterName(param.name.span));
                continue;
            }
            try param_names.put(param_symbol, param.name.span);

            if (try self.resolveTypeNameScoped(param.type_name, &type_scope)) |type_id| {
                _ = try self.module.hir.addParam(function_id, param_symbol, type_id, param.span);
            }
        }
    }

    fn resolveStruct(self: *Collector, struct_decl: ast.StructDecl) !void {
        const struct_symbol = try self.module.interner.intern(struct_decl.name.text);
        const struct_id = switch (self.top_level_decls.get(struct_symbol).?) {
            .struct_ => |entry| entry.id,
            else => unreachable,
        };

        var field_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer field_names.deinit();

        for (struct_decl.fields) |field| {
            const field_symbol = try self.module.interner.intern(field.name.text);
            if (field_names.contains(field_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateStructField(field.name.span));
                continue;
            }
            try field_names.put(field_symbol, field.name.span);

            if (try self.resolveTypeName(field.type_name)) |type_id| {
                _ = try self.module.hir.addField(struct_id, field_symbol, type_id, field.span);
            }
        }
    }

    fn resolveEnum(self: *Collector, enum_decl: ast.EnumDecl) !void {
        const enum_symbol = try self.module.interner.intern(enum_decl.name.text);
        const enum_id = switch (self.top_level_decls.get(enum_symbol).?) {
            .enum_ => |entry| entry.id,
            else => unreachable,
        };

        var variant_names = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer variant_names.deinit();

        for (enum_decl.variants) |variant| {
            const variant_symbol = try self.module.interner.intern(variant.name.text);
            if (variant_names.contains(variant_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateEnumVariant(variant.name.span));
                continue;
            }
            try variant_names.put(variant_symbol, variant.name.span);

            const variant_id = try self.module.hir.addVariant(enum_id, variant_symbol, variant.span);
            for (variant.payload_fields) |payload_field| {
                const payload_symbol = try self.module.interner.intern(payload_field.name.text);
                if (try self.resolveTypeName(payload_field.type_name)) |type_id| {
                    _ = try self.module.hir.addEnumPayloadField(variant_id, payload_symbol, type_id, payload_field.span);
                }
            }
        }

        self.module.hir.setEnumResultShape(enum_id, try self.detectResultShape(enum_id));
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Result-shape detection
    // ─────────────────────────────────────────────────────────────────────────────

    fn detectResultShape(self: *Collector, enum_id: hir.EnumId) !?hir.HirResultShape {
        const enum_decl = self.module.hir.getEnum(enum_id);
        if (enum_decl.variants.len != 2) return null;

        const ok_symbol = try self.module.interner.intern("Ok");
        const err_symbol = try self.module.interner.intern("Err");

        var ok_variant_id: ?hir.VariantId = null;
        var err_variant_id: ?hir.VariantId = null;
        for (enum_decl.variants) |variant_id| {
            const variant = self.module.hir.getVariant(variant_id);
            if (variant.name.index == ok_symbol.index) {
                ok_variant_id = variant_id;
            } else if (variant.name.index == err_symbol.index) {
                err_variant_id = variant_id;
            } else {
                return null;
            }
        }

        const ok_id = ok_variant_id orelse return null;
        const err_id = err_variant_id orelse return null;
        const ok_variant = self.module.hir.getVariant(ok_id);
        const err_variant = self.module.hir.getVariant(err_id);
        if (ok_variant.payload_fields.len != 1 or err_variant.payload_fields.len != 1) return null;

        const ok_payload_id = ok_variant.payload_fields[0];
        const err_payload_id = err_variant.payload_fields[0];
        const ok_payload = self.module.hir.getEnumPayloadField(ok_payload_id);
        const err_payload = self.module.hir.getEnumPayloadField(err_payload_id);

        return .{
            .ok_variant = ok_id,
            .err_variant = err_id,
            .ok_payload = ok_payload_id,
            .err_payload = err_payload_id,
            .ok_type = ok_payload.type_id,
            .err_type = err_payload.type_id,
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Type-name resolution
    // ─────────────────────────────────────────────────────────────────────────────

    fn resolveTypeName(self: *Collector, type_name: ast.TypeName) !?types.TypeId {
        return self.resolveTypeNameScoped(type_name, null);
    }

    fn resolveTypeNameScoped(self: *Collector, type_name: ast.TypeName, type_scope: ?*TypeParamScope) !?types.TypeId {
        if (type_name.is_mut or type_name.is_reference or type_name.generic_args.len != 0 or type_name.name.parts.len != 1) {
            try self.diagnostics.append(diagnostics.unsupportedTypeSyntax(type_name.span));
            return null;
        }

        const pointee = try self.resolveBaseTypeNameScoped(type_name, type_scope) orelse return null;
        if (type_name.is_pointer) return try self.module.types.addPointerType(pointee);
        return pointee;
    }

    fn resolveBaseTypeNameScoped(self: *Collector, type_name: ast.TypeName, type_scope: ?*TypeParamScope) !?types.TypeId {
        const part = type_name.name.parts[0];
        if (std.mem.eql(u8, part.text, "void")) return self.module.types.voidType();
        if (std.mem.eql(u8, part.text, "int")) return self.module.types.intType();
        if (std.mem.eql(u8, part.text, "bool")) return self.module.types.boolType();

        const symbol = try self.module.interner.intern(part.text);
        if (type_scope) |scope| {
            if (scope.get(symbol)) |type_id| return type_id;
        }
        const decl = self.top_level_decls.get(symbol) orelse {
            try self.diagnostics.append(diagnostics.unknownTypeName(part.span));
            return null;
        };

        return switch (decl) {
            .struct_ => |entry| entry.type_id,
            .enum_ => |entry| entry.type_id,
            .function, .generic_function => blk: {
                try self.diagnostics.append(diagnostics.unknownTypeName(part.span));
                break :blk null;
            },
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Diagnostic helpers
    // ─────────────────────────────────────────────────────────────────────────────

    fn internFreshTopLevelName(self: *Collector, text: []const u8, span: source.SourceSpan) !?interner.SymbolId {
        const symbol = try self.module.interner.intern(text);
        if (self.top_level_decls.contains(symbol)) {
            try self.diagnostics.append(diagnostics.duplicateTopLevelName(span));
            return null;
        }
        return symbol;
    }

    fn renderTypeName(self: *Collector, type_name: ast.TypeName) ![]const u8 {
        var buffer: std.Io.Writer.Allocating = .init(self.allocator);
        errdefer buffer.deinit();
        try type_name.write(&buffer.writer);
        return buffer.toOwnedSlice();
    }

    fn lowerGenericFunctionBody(self: *Collector, template_decl: ast.TemplateDecl) !void {
        const body = template_decl.body.body orelse return;
        const block = body.block orelse return;
        const function_symbol = try self.module.interner.intern(template_decl.body.signature.name.base.text);
        const generic_id = switch (self.top_level_decls.get(function_symbol).?) {
            .generic_function => |id| id,
            else => unreachable,
        };
        const function_id = self.module.hir.getGenericFunction(generic_id).function;

        var lowerer = BodyLowerer.init(self, function_id);
        defer lowerer.deinit();
        try lowerer.seedParams();
        if (try lowerer.lowerBlock(block)) |body_id| {
            self.module.hir.setFunctionBody(function_id, body_id);
        }
    }

    fn lowerFunctionBody(self: *Collector, function_decl: ast.FunctionDecl) !void {
        const body = function_decl.body orelse return;
        const block = body.block orelse return;
        const function_symbol = try self.module.interner.intern(function_decl.signature.name.base.text);
        const function_id = switch (self.top_level_decls.get(function_symbol).?) {
            .function => |id| id,
            else => unreachable,
        };

        var lowerer = BodyLowerer.init(self, function_id);
        defer lowerer.deinit();
        try lowerer.seedParams();
        if (try lowerer.lowerBlock(block)) |body_id| {
            self.module.hir.setFunctionBody(function_id, body_id);
        }
    }
};

const TypeParamScope = struct {
    map: std.AutoHashMap(interner.SymbolId, types.TypeId),

    fn init(allocator: std.mem.Allocator) TypeParamScope {
        return .{ .map = std.AutoHashMap(interner.SymbolId, types.TypeId).init(allocator) };
    }

    fn deinit(self: *TypeParamScope) void {
        self.map.deinit();
    }

    fn put(self: *TypeParamScope, name: interner.SymbolId, type_id: types.TypeId) !void {
        try self.map.put(name, type_id);
    }

    fn get(self: *TypeParamScope, name: interner.SymbolId) ?types.TypeId {
        return self.map.get(name);
    }
};

const Binding = hir.AssignBase;

const ScopedBinding = struct {
    name: interner.SymbolId,
    binding: Binding,
    depth: usize,
};

// ─────────────────────────────────────────────────────────────────────────────
// Body lowering state
// ─────────────────────────────────────────────────────────────────────────────

const BodyLowerer = struct {
    collector: *Collector,
    function_id: hir.FunctionId,
    bindings: std.ArrayList(ScopedBinding),
    depth: usize = 0,

    fn init(collector: *Collector, function_id: hir.FunctionId) BodyLowerer {
        return .{
            .collector = collector,
            .function_id = function_id,
            .bindings = std.ArrayList(ScopedBinding).empty,
        };
    }

    fn deinit(self: *BodyLowerer) void {
        self.bindings.deinit(self.collector.allocator);
    }

    fn seedParams(self: *BodyLowerer) !void {
        const function = self.collector.module.hir.getFunction(self.function_id);
        for (function.params) |param_id| {
            const param = self.collector.module.hir.getParam(param_id);
            try self.bindings.append(self.collector.allocator, .{ .name = param.name, .binding = .{ .param = param_id }, .depth = 0 });
        }
    }

    fn pushScope(self: *BodyLowerer) void {
        self.depth += 1;
    }

    fn popScope(self: *BodyLowerer) void {
        while (self.bindings.items.len > 0 and self.bindings.items[self.bindings.items.len - 1].depth == self.depth) {
            _ = self.bindings.pop();
        }
        self.depth -= 1;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Statement lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerBlock(self: *BodyLowerer, block: ast.BlockStmt) anyerror!?hir.StmtId {
        self.pushScope();
        defer self.popScope();

        var stmt_ids = std.ArrayList(hir.StmtId).empty;
        defer stmt_ids.deinit(self.collector.allocator);
        for (block.statements) |stmt| {
            if (try self.lowerStmt(stmt)) |stmt_id| try stmt_ids.append(self.collector.allocator, stmt_id);
        }
        const owned = try self.collector.allocator.alloc(hir.StmtId, stmt_ids.items.len);
        @memcpy(owned, stmt_ids.items);
        return try self.collector.module.hir.addStmt(.{ .block = owned }, block.span);
    }

    fn lowerArm(self: *BodyLowerer, arm: ast.MatchArm) anyerror!?hir.HirMatchArm {
        self.pushScope();
        defer self.popScope();
        const pattern = (try self.lowerPattern(arm.pattern)) orelse return null;
        const body = (try self.lowerStmt(arm.body)) orelse return null;
        return .{ .pattern = pattern, .pattern_span = arm.pattern.span(), .body = body };
    }

    fn lowerStmt(self: *BodyLowerer, stmt: ast.Stmt) anyerror!?hir.StmtId {
        switch (stmt) {
            .block_stmt => |block| return self.lowerBlock(block),
            .return_stmt => |ret| {
                const value = if (ret.value) |expr| try self.lowerExpr(expr.*) else null;
                if (ret.value != null and value == null) return null;
                return try self.collector.module.hir.addStmt(.{ .return_stmt = value }, ret.span);
            },
            .local_decl => |local_decl| {
                const initializer = (try self.lowerExpr(local_decl.initializer.*)) orelse return null;
                const local_symbol = try self.collector.module.interner.intern(local_decl.name.text);
                if (self.lookup(local_symbol) != null) {
                    try self.collector.diagnostics.append(diagnostics.duplicateLocalName(local_decl.name.span));
                    return null;
                }
                const type_id = (try self.collector.resolveTypeName(local_decl.type_name)) orelse return null;
                const local_id = try self.collector.module.hir.addLocal(self.function_id, local_symbol, type_id, local_decl.span);
                try self.bindings.append(self.collector.allocator, .{ .name = local_symbol, .binding = .{ .local = local_id }, .depth = self.depth });
                return try self.collector.module.hir.addStmt(.{ .local_decl = .{ .local = local_id, .initializer = initializer } }, local_decl.span);
            },
            .expr_stmt => |expr_stmt| {
                const value = (try self.lowerExpr(expr_stmt.value.*)) orelse return null;
                return try self.collector.module.hir.addStmt(.{ .expr_stmt = value }, expr_stmt.span);
            },
            .discard_stmt => |discard_stmt| {
                const value = (try self.lowerExpr(discard_stmt.value.*)) orelse return null;
                return try self.collector.module.hir.addStmt(.{ .discard_stmt = value }, discard_stmt.span);
            },
            .assignment => |assignment| {
                const target = (try self.lowerAssignTarget(assignment.target.*)) orelse return null;
                const value = (try self.lowerExpr(assignment.value.*)) orelse return null;
                return try self.collector.module.hir.addStmt(.{ .assignment = .{ .target = target, .value = value } }, assignment.span);
            },
            .if_stmt => |if_stmt| {
                const condition = (try self.lowerExpr(if_stmt.condition.*)) orelse return null;
                const then_block = (try self.lowerBlock(if_stmt.then_block)) orelse return null;
                const else_block = if (if_stmt.else_block) |else_block| try self.lowerBlock(else_block) else null;
                if (if_stmt.else_block != null and else_block == null) return null;
                return try self.collector.module.hir.addStmt(.{ .if_stmt = .{ .condition = condition, .then_block = then_block, .else_block = else_block } }, if_stmt.span);
            },
            .while_stmt => |while_stmt| {
                const condition = (try self.lowerExpr(while_stmt.condition.*)) orelse return null;
                const body = (try self.lowerBlock(while_stmt.body)) orelse return null;
                return try self.collector.module.hir.addStmt(.{ .while_stmt = .{ .condition = condition, .body = body } }, while_stmt.span);
            },
            .unsafe_block => |unsafe_block| {
                const body = (try self.lowerBlock(unsafe_block.body)) orelse return null;
                return try self.collector.module.hir.addStmt(.{ .unsafe_block = body }, unsafe_block.span);
            },
            .match_stmt => |match_stmt| {
                const scrutinee = (try self.lowerExpr(match_stmt.scrutinee.*)) orelse return null;
                var arms = std.ArrayList(hir.HirMatchArm).empty;
                defer arms.deinit(self.collector.allocator);
                for (match_stmt.arms) |arm| {
                    if (try self.lowerArm(arm)) |lowered_arm| {
                        try arms.append(self.collector.allocator, lowered_arm);
                    } else return null;
                }
                const owned = try self.collector.allocator.alloc(hir.HirMatchArm, arms.items.len);
                @memcpy(owned, arms.items);
                return try self.collector.module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = scrutinee, .arms = owned } }, match_stmt.span);
            },
        }
    }

    fn lowerAssignTarget(self: *BodyLowerer, target: ast.Expr) !?hir.AssignTarget {
        return switch (target) {
            .identifier => |ident| blk: {
                const target_symbol = try self.collector.module.interner.intern(ident.name.text);
                const base = self.lookup(target_symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownIdentifier(ident.name.span));
                    return null;
                };
                break :blk switch (base) {
                    .local => |id| hir.AssignTarget{ .local = id },
                    .param => |id| hir.AssignTarget{ .param = id },
                };
            },
            .field_access => |field_access| blk: {
                const base = (try self.lowerAssignBase(field_access.receiver.*)) orelse return null;
                const base_type = self.assignBaseType(base);
                const base_kind = self.collector.module.types.kind(base_type);
                if (base_kind != .struct_type) {
                    try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.FieldAccessNonStruct, .@"error", "field assignment receiver must be a struct place", field_access.receiver.span()));
                    return null;
                }
                const field_symbol = try self.collector.module.interner.intern(field_access.field_name.text);
                const field_id = self.findField(base_kind.struct_type, field_symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.UnknownFieldAccess, .@"error", "unknown field on struct value", field_access.field_name.span));
                    return null;
                };
                break :blk hir.AssignTarget{ .field = .{ .base = base, .field_id = field_id, .field_span = field_access.field_name.span } };
            },
            else => {
                try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.FieldAssignmentNonPlace, .@"error", "field assignment target must be an assignable place", target.span()));
                return null;
            },
        };
    }

    fn lowerAssignBase(self: *BodyLowerer, target: ast.Expr) !?hir.AssignBase {
        return switch (target) {
            .identifier => |ident| blk: {
                const symbol = try self.collector.module.interner.intern(ident.name.text);
                break :blk self.lookup(symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownIdentifier(ident.name.span));
                    return null;
                };
            },
            else => {
                try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.FieldAssignmentNonPlace, .@"error", "field assignment receiver must be an assignable place", target.span()));
                return null;
            },
        };
    }

    fn assignBaseType(self: *BodyLowerer, base: hir.AssignBase) types.TypeId {
        return switch (base) {
            .local => |id| self.collector.module.hir.getLocal(id).type_id,
            .param => |id| self.collector.module.hir.getParam(id).type_id,
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Expression lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerExpr(self: *BodyLowerer, expr: ast.Expr) anyerror!?hir.ExprId {
        switch (expr) {
            .int_literal => |lit| return try self.collector.module.hir.addExpr(.{ .int_literal = try self.collector.allocator.dupe(u8, lit.text) }, lit.span),
            .bool_literal => |lit| return try self.collector.module.hir.addExpr(.{ .bool_literal = lit.value }, lit.span),
            .identifier => |ident| {
                const symbol = try self.collector.module.interner.intern(ident.name.text);
                const binding = self.lookup(symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownIdentifier(ident.name.span));
                    return null;
                };
                return switch (binding) {
                    .local => |id| try self.collector.module.hir.addExpr(.{ .local_ref = id }, ident.span),
                    .param => |id| try self.collector.module.hir.addExpr(.{ .param_ref = id }, ident.span),
                };
            },
            .call => |call| {
                const symbol = try self.collector.module.interner.intern(call.callee.text);
                const function_id = switch (self.collector.top_level_decls.get(symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownFunction(call.callee.span));
                    return null;
                }) {
                    .function => |id| id,
                    .generic_function => |id| self.collector.module.hir.getGenericFunction(id).function,
                    else => {
                        try self.collector.diagnostics.append(diagnostics.unknownFunction(call.callee.span));
                        return null;
                    },
                };
                var args = std.ArrayList(hir.ExprId).empty;
                defer args.deinit(self.collector.allocator);
                for (call.args) |arg| {
                    const arg_id = (try self.lowerExpr(arg.*)) orelse return null;
                    try args.append(self.collector.allocator, arg_id);
                }
                const owned = try self.collector.allocator.alloc(hir.ExprId, args.items.len);
                @memcpy(owned, args.items);
                return try self.collector.module.hir.addExpr(.{ .call = .{ .function = function_id, .args = owned } }, call.span);
            },
            .struct_literal => |literal| {
                const struct_symbol = try self.collector.module.interner.intern(literal.type_name.text);
                const target = switch (self.collector.top_level_decls.get(struct_symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownStructLiteralType(literal.type_name.span));
                    return null;
                }) {
                    .struct_ => |entry| entry,
                    else => {
                        try self.collector.diagnostics.append(diagnostics.unknownStructLiteralType(literal.type_name.span));
                        return null;
                    },
                };
                const struct_decl = self.collector.module.hir.getStruct(target.id);
                var seen = std.AutoHashMap(hir.FieldId, source.SourceSpan).init(self.collector.allocator);
                defer seen.deinit();
                var fields = std.ArrayList(hir.HirStructLiteralField).empty;
                defer fields.deinit(self.collector.allocator);
                for (literal.fields) |field| {
                    const field_symbol = try self.collector.module.interner.intern(field.name.text);
                    const field_id = self.findField(target.id, field_symbol) orelse {
                        try self.collector.diagnostics.append(diagnostics.unknownStructLiteralField(field.name.span));
                        return null;
                    };
                    if (seen.contains(field_id)) {
                        try self.collector.diagnostics.append(diagnostics.duplicateStructLiteralField(field.name.span));
                        return null;
                    }
                    try seen.put(field_id, field.name.span);
                    const value = (try self.lowerExpr(field.value.*)) orelse return null;
                    try fields.append(self.collector.allocator, .{ .field_id = field_id, .value = value, .span = field.span });
                }
                for (struct_decl.fields) |field_id| {
                    if (!seen.contains(field_id)) {
                        const field = self.collector.module.hir.getField(field_id);
                        try self.collector.diagnostics.append(diagnostics.missingStructLiteralField(field.span));
                        return null;
                    }
                }
                const owned = try self.collector.allocator.alloc(hir.HirStructLiteralField, fields.items.len);
                @memcpy(owned, fields.items);
                return try self.collector.module.hir.addExpr(.{ .struct_literal = .{ .struct_id = target.id, .type_id = target.type_id, .fields = owned } }, literal.span);
            },
            .enum_constructor => |constructor| {
                const enum_symbol = try self.collector.module.interner.intern(constructor.enum_name.text);
                const enum_id = switch (self.collector.top_level_decls.get(enum_symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownEnumConstructor(constructor.enum_name.span));
                    return null;
                }) {
                    .enum_ => |entry| entry.id,
                    else => {
                        try self.collector.diagnostics.append(diagnostics.unknownEnumConstructor(constructor.enum_name.span));
                        return null;
                    },
                };
                const variant_symbol = try self.collector.module.interner.intern(constructor.variant_name.text);
                const variant_id = self.findVariant(enum_id, variant_symbol) orelse {
                    try self.collector.diagnostics.append(diagnostics.unknownEnumVariant(constructor.variant_name.span));
                    return null;
                };
                const variant = self.collector.module.hir.getVariant(variant_id);
                if (constructor.args.len != variant.payload_fields.len) {
                    try self.collector.diagnostics.append(diagnostics.enumConstructorArityMismatch(constructor.span));
                    return null;
                }
                var args = std.ArrayList(hir.ExprId).empty;
                defer args.deinit(self.collector.allocator);
                for (constructor.args) |arg| {
                    const arg_id = (try self.lowerExpr(arg.*)) orelse return null;
                    try args.append(self.collector.allocator, arg_id);
                }
                const owned = try self.collector.allocator.alloc(hir.ExprId, args.items.len);
                @memcpy(owned, args.items);
                return try self.collector.module.hir.addExpr(.{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = variant_id, .args = owned } }, constructor.span);
            },
            .group => |group| {
                const inner = (try self.lowerExpr(group.inner.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .group = inner }, group.span);
            },
            .unary => |unary| {
                const operand = (try self.lowerExpr(unary.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .unary = .{ .op = lowerUnaryOp(unary.op), .operand = operand } }, unary.span);
            },
            .address_of => |address_of| {
                const operand = (try self.lowerExpr(address_of.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .address_of = operand }, address_of.span);
            },
            .deref => |deref| {
                const operand = (try self.lowerExpr(deref.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .deref = operand }, deref.span);
            },
            .try_expr => |try_expr| {
                const operand = (try self.lowerExpr(try_expr.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .try_expr = operand }, try_expr.span);
            },
            .field_access => |field_access| {
                const receiver = (try self.lowerExpr(field_access.receiver.*)) orelse return null;
                const field_symbol = try self.collector.module.interner.intern(field_access.field_name.text);
                return try self.collector.module.hir.addExpr(.{ .field_access = .{
                    .receiver = receiver,
                    .field_name = field_symbol,
                    .field_span = field_access.field_name.span,
                } }, field_access.span);
            },
            .decide => |decide| {
                const target = (try self.resolveDecideEnum(decide.type_name)) orelse return null;
                var arms = std.ArrayList(hir.HirDecideArm).empty;
                defer arms.deinit(self.collector.allocator);
                for (decide.arms) |arm| {
                    const variant_symbol = try self.collector.module.interner.intern(arm.variant_name.text);
                    const variant_id = self.findVariant(target.enum_id, variant_symbol) orelse {
                        try self.collector.diagnostics.append(diagnostics.unknownDecideVariant(arm.variant_name.span));
                        return null;
                    };
                    const condition = if (arm.condition) |condition_expr| (try self.lowerExpr(condition_expr.*)) orelse return null else null;
                    const score = (try self.lowerExpr(arm.score.*)) orelse return null;
                    try arms.append(self.collector.allocator, .{
                        .variant_id = variant_id,
                        .condition = condition,
                        .score = score,
                        .span = arm.span,
                    });
                }
                const owned = try self.collector.allocator.alloc(hir.HirDecideArm, arms.items.len);
                @memcpy(owned, arms.items);
                return try self.collector.module.hir.addExpr(.{ .decide = .{ .enum_type = target.type_id, .enum_id = target.enum_id, .arms = owned } }, decide.span);
            },
            .binary => |binary| {
                const left = (try self.lowerExpr(binary.left.*)) orelse return null;
                const right = (try self.lowerExpr(binary.right.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .binary = .{ .op = lowerBinaryOp(binary.op), .left = left, .right = right } }, binary.span);
            },
        }
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Decide lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn resolveDecideEnum(self: *BodyLowerer, type_name: ast.TypeName) !?struct { type_id: types.TypeId, enum_id: hir.EnumId } {
        if (type_name.is_mut or type_name.is_reference or type_name.is_pointer or type_name.generic_args.len != 0 or type_name.name.parts.len != 1) {
            try self.collector.diagnostics.append(diagnostics.unknownDecideEnum(type_name.span));
            return null;
        }

        const part = type_name.name.parts[0];
        const symbol = try self.collector.module.interner.intern(part.text);
        return switch (self.collector.top_level_decls.get(symbol) orelse {
            try self.collector.diagnostics.append(diagnostics.unknownDecideEnum(part.span));
            return null;
        }) {
            .enum_ => |entry| .{ .type_id = entry.type_id, .enum_id = entry.id },
            else => blk: {
                try self.collector.diagnostics.append(diagnostics.unknownDecideEnum(part.span));
                break :blk null;
            },
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Pattern lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerPattern(self: *BodyLowerer, pattern: ast.MatchPattern) !?hir.HirMatchPattern {
        return switch (pattern) {
            .int_literal => |lit| .{ .int_literal = try self.collector.allocator.dupe(u8, lit.text) },
            .bool_literal => |lit| .{ .bool_literal = lit.value },
            .wildcard => .wildcard,
            .enum_variant => |variant_pattern| try self.lowerEnumVariantPattern(variant_pattern),
        };
    }

    fn lowerEnumVariantPattern(self: *BodyLowerer, variant_pattern: ast.EnumVariantPattern) !?hir.HirMatchPattern {
        const enum_symbol = try self.collector.module.interner.intern(variant_pattern.enum_name.text);
        const enum_id = switch (self.collector.top_level_decls.get(enum_symbol) orelse {
            try self.collector.diagnostics.append(diagnostics.unknownEnumPattern(variant_pattern.enum_name.span));
            return null;
        }) {
            .enum_ => |entry| entry.id,
            else => {
                try self.collector.diagnostics.append(diagnostics.unknownEnumPattern(variant_pattern.enum_name.span));
                return null;
            },
        };
        const variant_symbol = try self.collector.module.interner.intern(variant_pattern.variant_name.text);
        const variant_id = self.findVariant(enum_id, variant_symbol) orelse {
            try self.collector.diagnostics.append(diagnostics.unknownEnumPattern(variant_pattern.variant_name.span));
            return null;
        };
        const variant = self.collector.module.hir.getVariant(variant_id);
        if (variant_pattern.bindings.len != 0 and variant_pattern.bindings.len != variant.payload_fields.len) {
            try self.collector.diagnostics.append(diagnostics.enumPayloadBindingArityMismatch(variant_pattern.span));
            return null;
        }
        var bindings = std.ArrayList(hir.HirPatternBinding).empty;
        errdefer bindings.deinit(self.collector.allocator);
        for (variant_pattern.bindings, 0..) |binding, index| {
            const symbol = try self.collector.module.interner.intern(binding.name.text);
            for (bindings.items) |existing| {
                if (existing.name.index == symbol.index) {
                    try self.collector.diagnostics.append(diagnostics.duplicatePatternBinding(binding.name.span));
                    return null;
                }
            }
            if (self.lookup(symbol) != null) {
                try self.collector.diagnostics.append(diagnostics.duplicateLocalName(binding.name.span));
                return null;
            }
            const payload_id = variant.payload_fields[index];
            const payload = self.collector.module.hir.getEnumPayloadField(payload_id);
            const local_id = try self.collector.module.hir.addLocal(self.function_id, symbol, payload.type_id, binding.name.span);
            try self.bindings.append(self.collector.allocator, .{ .name = symbol, .binding = .{ .local = local_id }, .depth = self.depth });
            try bindings.append(self.collector.allocator, .{ .name = symbol, .local = local_id, .payload_field = payload_id, .type_id = payload.type_id, .span = binding.name.span });
        }
        return .{ .enum_variant = .{ .enum_id = enum_id, .variant_id = variant_id, .bindings = try bindings.toOwnedSlice(self.collector.allocator) } };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Scope/binding helpers
    // ─────────────────────────────────────────────────────────────────────────────

    fn findVariant(self: *BodyLowerer, enum_id: hir.EnumId, variant_symbol: interner.SymbolId) ?hir.VariantId {
        const enum_decl = self.collector.module.hir.getEnum(enum_id);
        for (enum_decl.variants) |variant_id| {
            const variant = self.collector.module.hir.getVariant(variant_id);
            if (variant.name.index == variant_symbol.index) return variant_id;
        }
        return null;
    }

    fn findField(self: *BodyLowerer, struct_id: hir.StructId, field_symbol: interner.SymbolId) ?hir.FieldId {
        const struct_decl = self.collector.module.hir.getStruct(struct_id);
        for (struct_decl.fields) |field_id| {
            const field = self.collector.module.hir.getField(field_id);
            if (field.name.index == field_symbol.index) return field_id;
        }
        return null;
    }

    fn lookup(self: *BodyLowerer, name: interner.SymbolId) ?hir.AssignBase {
        var index = self.bindings.items.len;
        while (index > 0) {
            index -= 1;
            const candidate = self.bindings.items[index];
            if (candidate.name.index == name.index) return candidate.binding;
        }
        return null;
    }
};

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

fn lowerUnaryOp(op: ast.UnaryOp) hir.UnaryOp {
    return switch (op) {
        .negate => .negate,
        .logical_not => .logical_not,
    };
}

fn lowerBinaryOp(op: ast.BinaryOp) hir.BinaryOp {
    return switch (op) {
        .multiply => .multiply,
        .divide => .divide,
        .modulo => .modulo,
        .add => .add,
        .subtract => .subtract,
        .less => .less,
        .less_equal => .less_equal,
        .greater => .greater,
        .greater_equal => .greater_equal,
        .equal_equal => .equal_equal,
        .bang_equal => .bang_equal,
        .logical_and => .logical_and,
        .logical_or => .logical_or,
    };
}

fn collectItems(items: []const ast.Item, diagnostic_bag: *DiagnosticBag) !SemanticModule {
    return collectTopLevelDeclarations(std.testing.allocator, unitFromItems(items), diagnostic_bag);
}

fn unitFromItems(items: []const ast.Item) ast.CompilationUnit {
    return .{
        .module = null,
        .imports = &.{},
        .items = @constCast(items),
        .span = .{ .start = 0, .length = 0 },
    };
}

fn functionItem(name: []const u8, start: usize) ast.Item {
    return .{ .function_decl = .{
        .is_export = false,
        .signature = .{
            .return_type = typeName("int", start),
            .name = .{
                .base = nameSegment(name, start + 4),
                .span = .{ .start = start + 4, .length = name.len },
            },
            .params = &.{},
            .span = .{ .start = start, .length = name.len + 6 },
        },
        .body = null,
        .span = .{ .start = start, .length = name.len + 6 },
    } };
}

fn structItem(name: []const u8, start: usize) ast.Item {
    return .{ .struct_decl = .{
        .is_export = false,
        .name = nameSegment(name, start + 7),
        .fields = &.{},
        .span = .{ .start = start, .length = name.len + 10 },
    } };
}

fn enumItem(name: []const u8, start: usize) ast.Item {
    return .{ .enum_decl = .{
        .is_export = false,
        .name = nameSegment(name, start + 5),
        .variants = &.{},
        .span = .{ .start = start, .length = name.len + 8 },
    } };
}

fn enumItemWithVariants(name: []const u8, variants: []ast.EnumVariant, is_must_use: bool, start: usize) ast.Item {
    return .{ .enum_decl = .{
        .is_export = false,
        .is_must_use = is_must_use,
        .name = nameSegment(name, start + 5),
        .variants = variants,
        .span = .{ .start = start, .length = name.len + 8 },
    } };
}

fn enumVariant(name: []const u8, payload_fields: []ast.EnumPayloadField, start: usize) ast.EnumVariant {
    return .{
        .name = nameSegment(name, start),
        .payload_fields = payload_fields,
        .span = .{ .start = start, .length = name.len },
    };
}

fn enumPayload(type_name: []const u8, name: []const u8, start: usize) ast.EnumPayloadField {
    return .{
        .type_name = typeName(type_name, start),
        .name = nameSegment(name, start + type_name.len + 1),
        .span = .{ .start = start, .length = type_name.len + name.len + 1 },
    };
}

fn collectSingleEnum(item: ast.Item) !SemanticModule {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{item}, &diagnostics_bag);
    errdefer module.deinit();
    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    return module;
}

fn conceptItem(name: []const u8, start: usize) ast.Item {
    return .{ .concept_decl = .{
        .name = nameSegment(name, start + 8),
        .generic_params = &.{},
        .signatures = &.{},
        .span = .{ .start = start, .length = name.len + 11 },
    } };
}

fn interfaceItem(name: []const u8, start: usize) ast.Item {
    return .{ .interface_decl = .{
        .name = nameSegment(name, start + 10),
        .signatures = &.{},
        .span = .{ .start = start, .length = name.len + 13 },
    } };
}

fn implItem(start: usize) ast.Item {
    return .{ .impl_decl = .{
        .target = typeName("Drop", start + 5),
        .signatures = &.{},
        .span = .{ .start = start, .length = 12 },
    } };
}

const test_int_type_parts = [_]ast.NameSegment{.{ .text = "int", .span = .{ .start = 0, .length = 3 } }};
const test_drop_type_parts = [_]ast.NameSegment{.{ .text = "Drop", .span = .{ .start = 0, .length = 4 } }};
const test_bool_type_parts = [_]ast.NameSegment{.{ .text = "bool", .span = .{ .start = 0, .length = 4 } }};
const test_void_type_parts = [_]ast.NameSegment{.{ .text = "void", .span = .{ .start = 0, .length = 4 } }};

fn typeName(name: []const u8, start: usize) ast.TypeName {
    const parts: []ast.NameSegment = if (std.mem.eql(u8, name, "Drop"))
        @constCast(test_drop_type_parts[0..])
    else if (std.mem.eql(u8, name, "bool"))
        @constCast(test_bool_type_parts[0..])
    else if (std.mem.eql(u8, name, "void"))
        @constCast(test_void_type_parts[0..])
    else
        @constCast(test_int_type_parts[0..]);
    return .{
        .name = .{
            .parts = parts,
            .span = .{ .start = start, .length = name.len },
        },
        .span = .{ .start = start, .length = name.len },
    };
}

fn nameSegment(text: []const u8, start: usize) ast.NameSegment {
    return .{ .text = text, .span = .{ .start = start, .length = text.len } };
}

fn typeNameFromSegment(part: *[1]ast.NameSegment) ast.TypeName {
    return .{
        .name = .{ .parts = part[0..], .span = part[0].span },
        .span = part[0].span,
    };
}

fn genericFunctionItem(template_params: []ast.TypeParamDecl, name: []const u8, return_type: ast.TypeName, params: []ast.ParamDecl, statements: []ast.Stmt) ast.Item {
    return .{ .template_decl = .{
        .params = template_params,
        .body = .{
            .is_export = false,
            .signature = .{
                .return_type = return_type,
                .name = .{ .base = nameSegment(name, 0), .span = .{ .start = 0, .length = name.len } },
                .params = params,
                .span = .{ .start = 0, .length = name.len },
            },
            .body = .{ .span = .{ .start = 0, .length = 1 }, .block = .{ .statements = statements, .span = .{ .start = 0, .length = 1 } } },
            .span = .{ .start = 0, .length = name.len },
        },
        .span = .{ .start = 0, .length = name.len },
    } };
}

test "semantic collection lowers generic identity declaration" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var t_parts = [_]ast.NameSegment{nameSegment("T", 0)};
    const t_type = typeNameFromSegment(&t_parts);
    var template_params = [_]ast.TypeParamDecl{.{ .name = nameSegment("T", 0), .span = .{ .start = 0, .length = 1 } }};
    var params = [_]ast.ParamDecl{.{ .type_name = t_type, .name = nameSegment("value", 0), .span = .{ .start = 0, .length = 1 } }};
    var value_expr = identExpr("value");
    var statements = [_]ast.Stmt{.{ .return_stmt = .{ .value = &value_expr, .span = .{ .start = 0, .length = 1 } } }};

    var module = try collectItems(&.{genericFunctionItem(template_params[0..], "identity", t_type, params[0..], statements[0..])}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    try std.testing.expectEqual(@as(usize, 1), module.hir.generic_functions.items.len);
    try std.testing.expectEqual(@as(usize, 0), module.hir.items.items.len);

    const generic = module.hir.getGenericFunction(.{ .index = 0 });
    const function = module.hir.getFunction(generic.function);
    const type_param = generic.type_params[0];
    try std.testing.expectEqual(type_param.type_id, function.return_type);
    try std.testing.expectEqual(type_param.type_id, module.hir.getParam(function.params[0]).type_id);
    try std.testing.expect(function.body != null);
}

test "semantic collection lowers multiple pointer and constrained generic parameters" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var k_parts = [_]ast.NameSegment{nameSegment("K", 0)};
    var v_parts = [_]ast.NameSegment{nameSegment("V", 0)};
    var equatable_parts = [_]ast.NameSegment{nameSegment("Equatable", 0)};
    const k_type = typeNameFromSegment(&k_parts);
    const v_type = typeNameFromSegment(&v_parts);
    var k_ptr_type = k_type;
    k_ptr_type.is_pointer = true;
    var constraint_args = [_]ast.TypeName{v_type};
    const constraint = ast.TypeName{ .name = .{ .parts = equatable_parts[0..], .span = equatable_parts[0].span }, .generic_args = constraint_args[0..], .span = .{ .start = 0, .length = 12 } };
    var template_params = [_]ast.TypeParamDecl{
        .{ .name = nameSegment("K", 0), .span = .{ .start = 0, .length = 1 } },
        .{ .name = nameSegment("V", 0), .constraint = constraint, .span = .{ .start = 0, .length = 1 } },
    };
    var params = [_]ast.ParamDecl{
        .{ .type_name = k_ptr_type, .name = nameSegment("key", 0), .span = .{ .start = 0, .length = 1 } },
        .{ .type_name = v_type, .name = nameSegment("value", 0), .span = .{ .start = 0, .length = 1 } },
    };
    var key_expr = identExpr("key");
    var deref_expr = ast.Expr{ .deref = .{ .operand = &key_expr, .span = .{ .start = 0, .length = 1 } } };
    var statements = [_]ast.Stmt{.{ .return_stmt = .{ .value = &deref_expr, .span = .{ .start = 0, .length = 1 } } }};

    var module = try collectItems(&.{genericFunctionItem(template_params[0..], "first", k_type, params[0..], statements[0..])}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const generic = module.hir.getGenericFunction(.{ .index = 0 });
    const function = module.hir.getFunction(generic.function);
    try std.testing.expectEqual(@as(usize, 2), generic.type_params.len);
    try std.testing.expectEqualStrings("Equatable<V>", generic.type_params[1].constraint.?.text);
    try std.testing.expectEqual(generic.type_params[0].type_id, function.return_type);
    try std.testing.expectEqual(types.TypeKind{ .pointer = .{ .pointee = generic.type_params[0].type_id } }, module.types.kind(module.hir.getParam(function.params[0]).type_id));
    try std.testing.expectEqual(generic.type_params[1].type_id, module.hir.getParam(function.params[1]).type_id);
}

test "semantic collection rejects duplicate generic and concrete names and hides type params" {
    var t_parts = [_]ast.NameSegment{nameSegment("T", 0)};
    const t_type = typeNameFromSegment(&t_parts);
    var template_params = [_]ast.TypeParamDecl{.{ .name = nameSegment("T", 0), .span = .{ .start = 0, .length = 1 } }};
    var params = [_]ast.ParamDecl{.{ .type_name = t_type, .name = nameSegment("value", 0), .span = .{ .start = 0, .length = 1 } }};
    var value_expr = identExpr("value");
    var statements = [_]ast.Stmt{.{ .return_stmt = .{ .value = &value_expr, .span = .{ .start = 0, .length = 1 } } }};

    var duplicate_diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer duplicate_diagnostics.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, collectItems(&.{ genericFunctionItem(template_params[0..], "identity", t_type, params[0..], statements[0..]), functionItem("identity", 20) }, &duplicate_diagnostics));
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, duplicate_diagnostics.diagnostics.items[0].code);

    var outside_diagnostics = DiagnosticBag.init(std.testing.allocator);
    defer outside_diagnostics.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, collectItems(&.{functionWithBody("outside", params[0..], statements[0..])}, &outside_diagnostics));
    try std.testing.expectEqual(DiagnosticCode.UnknownTypeName, outside_diagnostics.diagnostics.items[0].code);
}

test "semantic collection detects must_use Result-shaped enum" {
    var ok_payloads = [_]ast.EnumPayloadField{enumPayload("int", "value", 0)};
    var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 10)};
    var variants = [_]ast.EnumVariant{
        enumVariant("Ok", ok_payloads[0..], 0),
        enumVariant("Err", err_payloads[0..], 10),
    };

    var module = try collectSingleEnum(enumItemWithVariants("ParseIntResult", variants[0..], true, 0));
    defer module.deinit();

    const shape = module.hir.getResultShape(.{ .index = 0 }).?;
    try std.testing.expect(module.hir.isResultShapedEnum(.{ .index = 0 }));
    try std.testing.expectEqual(hir.VariantId{ .index = 0 }, shape.ok_variant);
    try std.testing.expectEqual(hir.VariantId{ .index = 1 }, shape.err_variant);
    try std.testing.expectEqual(hir.EnumPayloadFieldId{ .index = 0 }, shape.ok_payload);
    try std.testing.expectEqual(hir.EnumPayloadFieldId{ .index = 1 }, shape.err_payload);
    try std.testing.expectEqual(module.types.intType(), shape.ok_type);
    try std.testing.expectEqual(module.types.intType(), shape.err_type);
    try std.testing.expectEqual(shape, module.resultShapeForType(.{ .index = 3 }).?);
}

test "semantic collection detects non-must_use Result-shaped enum" {
    var ok_payloads = [_]ast.EnumPayloadField{enumPayload("int", "value", 0)};
    var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 10)};
    var variants = [_]ast.EnumVariant{
        enumVariant("Ok", ok_payloads[0..], 0),
        enumVariant("Err", err_payloads[0..], 10),
    };

    var module = try collectSingleEnum(enumItemWithVariants("ParseIntResult", variants[0..], false, 0));
    defer module.deinit();

    try std.testing.expect(module.hir.getResultShape(.{ .index = 0 }) != null);
}

test "semantic collection stores Result-shaped payload types" {
    var ok_payloads = [_]ast.EnumPayloadField{enumPayload("bool", "value", 0)};
    var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 10)};
    var variants = [_]ast.EnumVariant{
        enumVariant("Ok", ok_payloads[0..], 0),
        enumVariant("Err", err_payloads[0..], 10),
    };

    var module = try collectSingleEnum(enumItemWithVariants("BoolResult", variants[0..], false, 0));
    defer module.deinit();

    const shape = module.hir.getResultShape(.{ .index = 0 }).?;
    try std.testing.expectEqual(module.types.boolType(), shape.ok_type);
    try std.testing.expectEqual(module.types.intType(), shape.err_type);
}

test "semantic collection leaves ordinary enums non-Result-shaped" {
    {
        var variants = [_]ast.EnumVariant{enumVariant("Ready", &.{}, 0)};
        var module = try collectSingleEnum(enumItemWithVariants("Status", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expectEqual(@as(?hir.HirResultShape, null), module.hir.getResultShape(.{ .index = 0 }));
    }
    {
        var ok_payloads = [_]ast.EnumPayloadField{enumPayload("int", "value", 0)};
        var variants = [_]ast.EnumVariant{enumVariant("Ok", ok_payloads[0..], 0)};
        var module = try collectSingleEnum(enumItemWithVariants("OnlyOk", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
    }
    {
        var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 0)};
        var variants = [_]ast.EnumVariant{enumVariant("Err", err_payloads[0..], 0)};
        var module = try collectSingleEnum(enumItemWithVariants("OnlyErr", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
    }
}

test "semantic collection requires exact Result-shaped arity" {
    {
        var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 10)};
        var variants = [_]ast.EnumVariant{
            enumVariant("Ok", &.{}, 0),
            enumVariant("Err", err_payloads[0..], 10),
        };
        var module = try collectSingleEnum(enumItemWithVariants("BadOkZero", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
    }
    {
        var ok_payloads = [_]ast.EnumPayloadField{enumPayload("int", "value", 0)};
        var variants = [_]ast.EnumVariant{
            enumVariant("Ok", ok_payloads[0..], 0),
            enumVariant("Err", &.{}, 10),
        };
        var module = try collectSingleEnum(enumItemWithVariants("BadErrZero", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
    }
    {
        var ok_payloads = [_]ast.EnumPayloadField{
            enumPayload("int", "value", 0),
            enumPayload("bool", "rounded", 10),
        };
        var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 20)};
        var variants = [_]ast.EnumVariant{
            enumVariant("Ok", ok_payloads[0..], 0),
            enumVariant("Err", err_payloads[0..], 20),
        };
        var module = try collectSingleEnum(enumItemWithVariants("BadOkTwo", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
    }
    {
        var ok_payloads = [_]ast.EnumPayloadField{enumPayload("int", "value", 0)};
        var err_payloads = [_]ast.EnumPayloadField{
            enumPayload("int", "code", 10),
            enumPayload("bool", "fatal", 20),
        };
        var variants = [_]ast.EnumVariant{
            enumVariant("Ok", ok_payloads[0..], 0),
            enumVariant("Err", err_payloads[0..], 10),
        };
        var module = try collectSingleEnum(enumItemWithVariants("BadErrTwo", variants[0..], false, 0));
        defer module.deinit();
        try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
    }
}

test "semantic collection requires exactly Ok and Err variants for Result shape" {
    var ok_payloads = [_]ast.EnumPayloadField{enumPayload("int", "value", 0)};
    var err_payloads = [_]ast.EnumPayloadField{enumPayload("int", "code", 10)};
    var variants = [_]ast.EnumVariant{
        enumVariant("Ok", ok_payloads[0..], 0),
        enumVariant("Err", err_payloads[0..], 10),
        enumVariant("Pending", &.{}, 20),
    };

    var module = try collectSingleEnum(enumItemWithVariants("ExtraVariant", variants[0..], false, 0));
    defer module.deinit();

    try std.testing.expect(!module.hir.isResultShapedEnum(.{ .index = 0 }));
}

test "semantic collection collects one function" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{functionItem("main", 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.functions.items.len);
    try std.testing.expectEqualStrings("main", module.interner.text(module.hir.getFunction(.{ .index = 0 }).name));
}

test "semantic collection collects one struct" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{structItem("Vec2", 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqualStrings("Vec2", module.interner.text(module.hir.getStruct(.{ .index = 0 }).name));
}

test "semantic collection collects one enum" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{enumItem("Token", 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.enums.items.len);
    try std.testing.expectEqualStrings("Token", module.interner.text(module.hir.getEnum(.{ .index = 0 }).name));
}

test "semantic collection collects function struct and enum together" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ functionItem("make", 0), structItem("Vec2", 20), enumItem("Token", 40) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 3), module.hir.items.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.functions.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.enums.items.len);
}

test "semantic collection adds struct and enum nominal types" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ structItem("Vec2", 0), enumItem("Token", 20) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 5), module.types.count());
    try std.testing.expectEqual(types.TypeKind{ .struct_type = .{ .index = 0 } }, module.types.kind(.{ .index = 3 }));
    try std.testing.expectEqual(types.TypeKind{ .enum_type = .{ .index = 0 } }, module.types.kind(.{ .index = 4 }));
}

test "semantic collection resolves raw pointer function return and parameter types" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    const int_ptr = ast.TypeName{
        .name = .{ .parts = @constCast(test_int_type_parts[0..]), .span = .{ .start = 0, .length = 3 } },
        .is_pointer = true,
        .span = .{ .start = 0, .length = 4 },
    };
    var params = [_]ast.ParamDecl{.{
        .type_name = int_ptr,
        .name = nameSegment("p", 5),
        .span = .{ .start = 0, .length = 6 },
    }};
    const item = ast.Item{ .function_decl = .{
        .is_export = false,
        .signature = .{
            .return_type = int_ptr,
            .name = .{ .base = nameSegment("identity", 7), .span = .{ .start = 7, .length = 8 } },
            .params = params[0..],
            .span = .{ .start = 0, .length = 18 },
        },
        .body = null,
        .span = .{ .start = 0, .length = 18 },
    } };

    var module = try collectItems(&.{item}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const function = module.hir.getFunction(.{ .index = 0 });
    const pointer_type = function.return_type;
    try std.testing.expectEqual(pointer_type, module.hir.getParam(function.params[0]).type_id);
    try std.testing.expectEqual(types.TypeKind{ .pointer = .{ .pointee = module.types.intType() } }, module.types.kind(pointer_type));
}

test "semantic collection resolves raw pointer to enum type" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    const enum_decl = enumItem("Status", 0);
    const status_parts = [_]ast.NameSegment{.{ .text = "Status", .span = .{ .start = 20, .length = 6 } }};
    const status_ptr = ast.TypeName{
        .name = .{ .parts = @constCast(status_parts[0..]), .span = .{ .start = 20, .length = 6 } },
        .is_pointer = true,
        .span = .{ .start = 20, .length = 7 },
    };
    const function_decl = ast.Item{ .function_decl = .{
        .is_export = false,
        .signature = .{
            .return_type = status_ptr,
            .name = .{ .base = nameSegment("current", 28), .span = .{ .start = 28, .length = 7 } },
            .params = &.{},
            .span = .{ .start = 20, .length = 17 },
        },
        .body = null,
        .span = .{ .start = 20, .length = 17 },
    } };

    var module = try collectItems(&.{ enum_decl, function_decl }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const pointer_type = module.hir.getFunction(.{ .index = 0 }).return_type;
    try std.testing.expectEqual(types.TypeKind{ .pointer = .{ .pointee = .{ .index = 3 } } }, module.types.kind(pointer_type));
}

test "semantic collection rejects duplicate function" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(
        error.InvalidSemanticModule,
        collectItems(&.{ functionItem("f", 0), functionItem("f", 20) }, &diagnostics_bag),
    );
    try std.testing.expectEqual(@as(usize, 1), diagnostics_bag.count());
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, diagnostics_bag.diagnostics.items[0].code);
}

test "semantic collection rejects duplicate function and struct" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(
        error.InvalidSemanticModule,
        collectItems(&.{ functionItem("Foo", 0), structItem("Foo", 30) }, &diagnostics_bag),
    );
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, diagnostics_bag.diagnostics.items[0].code);
}

test "semantic collection rejects duplicate struct and enum" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(
        error.InvalidSemanticModule,
        collectItems(&.{ structItem("Token", 0), enumItem("Token", 30) }, &diagnostics_bag),
    );
    try std.testing.expectEqual(DiagnosticCode.DuplicateTopLevelName, diagnostics_bag.diagnostics.items[0].code);
}

test "semantic collection ignores concept interface and impl items" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ conceptItem("Hashable", 0), interfaceItem("Renderer", 40), implItem(80), structItem("Texture", 100) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.items.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
}

test "semantic collection HIR debug snapshot is stable" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ functionItem("make", 0), structItem("Vec2", 20), enumItem("Token", 40) }, &diagnostics_bag);
    defer module.deinit();

    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Function make -> TypeId(1)
        \\  Struct Vec2
        \\  Enum Token
        \\
    , snapshot);
}

fn allocExpr(allocator: std.mem.Allocator, expr: ast.Expr) !*ast.Expr {
    const ptr = try allocator.create(ast.Expr);
    ptr.* = expr;
    return ptr;
}

fn intExpr(text: []const u8) ast.Expr {
    return .{ .int_literal = .{ .text = text, .span = .{ .start = 0, .length = text.len } } };
}

fn identExpr(name: []const u8) ast.Expr {
    return .{ .identifier = .{ .name = nameSegment(name, 0), .span = .{ .start = 0, .length = name.len } } };
}

fn callExpr(allocator: std.mem.Allocator, name: []const u8, args: []const ast.Expr) !ast.Expr {
    const arg_ptrs = try allocator.alloc(*ast.Expr, args.len);
    for (args, 0..) |arg, index| arg_ptrs[index] = try allocExpr(allocator, arg);
    return .{ .call = .{ .callee = nameSegment(name, 0), .args = arg_ptrs, .span = .{ .start = 0, .length = name.len } } };
}

fn binaryExpr(allocator: std.mem.Allocator, op: ast.BinaryOp, left: ast.Expr, right: ast.Expr) !ast.Expr {
    return .{ .binary = .{ .op = op, .left = try allocExpr(allocator, left), .right = try allocExpr(allocator, right), .span = .{ .start = 0, .length = 1 } } };
}

fn returnStmt(allocator: std.mem.Allocator, expr: ast.Expr) !ast.Stmt {
    return .{ .return_stmt = .{ .value = try allocExpr(allocator, expr), .span = .{ .start = 0, .length = 1 } } };
}

fn localStmt(allocator: std.mem.Allocator, name: []const u8, expr: ast.Expr) !ast.Stmt {
    return .{ .local_decl = .{ .type_name = typeName("int", 0), .name = nameSegment(name, 0), .initializer = try allocExpr(allocator, expr), .span = .{ .start = 0, .length = 1 } } };
}

fn functionWithBody(name: []const u8, params: []ast.ParamDecl, statements: []ast.Stmt) ast.Item {
    return .{ .function_decl = .{
        .is_export = false,
        .signature = .{
            .return_type = typeName("int", 0),
            .name = .{ .base = nameSegment(name, 0), .span = .{ .start = 0, .length = name.len } },
            .params = params,
            .span = .{ .start = 0, .length = name.len },
        },
        .body = .{ .span = .{ .start = 0, .length = 1 }, .block = .{ .statements = statements, .span = .{ .start = 0, .length = 1 } } },
        .span = .{ .start = 0, .length = name.len },
    } };
}

fn expectOneSemanticDiagnostic(items: []const ast.Item, code: DiagnosticCode) !void {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, collectItems(items, &diagnostics_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostics_bag.count());
    try std.testing.expectEqual(code, diagnostics_bag.diagnostics.items[0].code);
}

test "body lowering lowers return integer" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const statements = try allocator.alloc(ast.Stmt, 1);
    statements[0] = try returnStmt(allocator, intExpr("42"));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectItems(&.{functionWithBody("main", &.{}, statements)}, &diagnostics_bag);
    defer module.deinit();

    const block = module.hir.getStmt(module.hir.getFunction(.{ .index = 0 }).body.?).kind.block;
    const ret = module.hir.getStmt(block[0]).kind.return_stmt.?;
    try std.testing.expectEqualStrings("42", module.hir.getExpr(ret).kind.int_literal);
}

test "body lowering lowers local declaration and local reference" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const statements = try allocator.alloc(ast.Stmt, 2);
    statements[0] = try localStmt(allocator, "x", intExpr("1"));
    statements[1] = try returnStmt(allocator, identExpr("x"));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectItems(&.{functionWithBody("main", &.{}, statements)}, &diagnostics_bag);
    defer module.deinit();

    const function = module.hir.getFunction(.{ .index = 0 });
    try std.testing.expectEqual(@as(usize, 1), function.locals.len);
    const block = module.hir.getStmt(function.body.?).kind.block;
    try std.testing.expectEqual(hir.LocalId{ .index = 0 }, module.hir.getStmt(block[0]).kind.local_decl.local);
    try std.testing.expectEqual(hir.HirExprKind{ .local_ref = .{ .index = 0 } }, module.hir.getExpr(module.hir.getStmt(block[1]).kind.return_stmt.?).kind);
}

test "body lowering resolves parameters and function calls" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const add_params = try allocator.alloc(ast.ParamDecl, 2);
    add_params[0] = .{ .type_name = typeName("int", 0), .name = nameSegment("a", 0), .span = .{ .start = 0, .length = 1 } };
    add_params[1] = .{ .type_name = typeName("int", 0), .name = nameSegment("b", 0), .span = .{ .start = 0, .length = 1 } };
    const add_stmts = try allocator.alloc(ast.Stmt, 1);
    add_stmts[0] = try returnStmt(allocator, try binaryExpr(allocator, .add, identExpr("a"), identExpr("b")));
    const main_stmts = try allocator.alloc(ast.Stmt, 1);
    main_stmts[0] = try returnStmt(allocator, try callExpr(allocator, "add", &.{ intExpr("1"), intExpr("2") }));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectItems(&.{ functionWithBody("add", add_params, add_stmts), functionWithBody("main", &.{}, main_stmts) }, &diagnostics_bag);
    defer module.deinit();

    const add_block = module.hir.getStmt(module.hir.getFunction(.{ .index = 0 }).body.?).kind.block;
    const add_ret = module.hir.getExpr(module.hir.getStmt(add_block[0]).kind.return_stmt.?).kind.binary;
    try std.testing.expectEqual(hir.HirExprKind{ .param_ref = .{ .index = 0 } }, module.hir.getExpr(add_ret.left).kind);
    const main_block = module.hir.getStmt(module.hir.getFunction(.{ .index = 1 }).body.?).kind.block;
    try std.testing.expectEqual(hir.FunctionId{ .index = 0 }, module.hir.getExpr(module.hir.getStmt(main_block[0]).kind.return_stmt.?).kind.call.function);
}

test "body lowering reports unknown and duplicate body names" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const unknown_stmts = try allocator.alloc(ast.Stmt, 1);
    unknown_stmts[0] = try returnStmt(allocator, identExpr("missing"));
    try expectOneSemanticDiagnostic(&.{functionWithBody("main", &.{}, unknown_stmts)}, .UnknownIdentifier);

    const dup_stmts = try allocator.alloc(ast.Stmt, 2);
    dup_stmts[0] = try localStmt(allocator, "x", intExpr("1"));
    dup_stmts[1] = try localStmt(allocator, "x", intExpr("2"));
    try expectOneSemanticDiagnostic(&.{functionWithBody("main", &.{}, dup_stmts)}, .DuplicateLocalName);
}

test "body lowering HIR debug snapshot with locals is stable" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const statements = try allocator.alloc(ast.Stmt, 2);
    statements[0] = try localStmt(allocator, "x", try binaryExpr(allocator, .add, intExpr("1"), intExpr("2")));
    statements[1] = try returnStmt(allocator, identExpr("x"));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectItems(&.{functionWithBody("main", &.{}, statements)}, &diagnostics_bag);
    defer module.deinit();
    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      LocalId(0) x: TypeId(1)
        \\    Body
        \\      Block
        \\        LocalDecl LocalId(0)
        \\          Binary +
        \\            Int 1
        \\            Int 2
        \\        Return
        \\          LocalRef LocalId(0)
        \\
    , snapshot);
}

test "body lowering lowers empty body and reports unknown function" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    const empty_statements = try allocator.alloc(ast.Stmt, 0);
    var module = try collectItems(&.{functionWithBody("main", &.{}, empty_statements)}, &diagnostics_bag);
    defer module.deinit();
    try std.testing.expect(module.hir.getFunction(.{ .index = 0 }).body != null);
    try std.testing.expectEqual(@as(usize, 0), module.hir.getStmt(module.hir.getFunction(.{ .index = 0 }).body.?).kind.block.len);

    const bad_statements = try allocator.alloc(ast.Stmt, 1);
    bad_statements[0] = try returnStmt(allocator, try callExpr(allocator, "missing", &.{}));
    try expectOneSemanticDiagnostic(&.{functionWithBody("bad", &.{}, bad_statements)}, .UnknownFunction);
}
