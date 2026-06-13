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
const compile_time = @import("compile_time.zig");
const compile_time_target = @import("compile_time_target.zig");

pub const DiagnosticBag = diagnostics.DiagnosticBag;
pub const DiagnosticCode = diagnostics.DiagnosticCode;
pub const SemanticError = error{InvalidSemanticModule};

pub const CollectOptions = struct {
    source_file_kind: source.SourceFileKind = .normal,
};

pub const SemanticModule = struct {
    interner: interner.Interner,
    hir: hir.HirStore,
    types: types.TypeStore,
    compile_time_values: std.AutoHashMap(hir.ExprId, compile_time.CompileTimeValue),
    compile_time_target_info: compile_time_target.CompileTimeTargetInfo,

    pub fn init(allocator: std.mem.Allocator) !SemanticModule {
        return .{
            .interner = interner.Interner.init(allocator),
            .hir = hir.HirStore.init(allocator),
            .types = try types.TypeStore.init(allocator),
            .compile_time_values = std.AutoHashMap(hir.ExprId, compile_time.CompileTimeValue).init(allocator),
            .compile_time_target_info = compile_time_target.defaultTargetInfo(),
        };
    }

    pub fn deinit(self: *SemanticModule) void {
        self.compile_time_values.deinit();
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

    pub const DropInfo = struct {
        type_id: types.TypeId,
        impl_id: hir.ConceptImplId,
        function: hir.FunctionId,
        span: source.SourceSpan,
    };

    pub fn hasDrop(self: *const SemanticModule, type_id: types.TypeId) ?DropInfo {
        const drop_concept = self.findIntrinsicDropConcept() orelse return null;
        const impl_id = self.hir.findConceptImpl(drop_concept, type_id) orelse return null;
        const concept_impl = self.hir.getConceptImpl(impl_id);
        if (concept_impl.functions.len != 1) return null;
        return .{
            .type_id = type_id,
            .impl_id = impl_id,
            .function = concept_impl.functions[0],
            .span = concept_impl.span,
        };
    }

    pub const ArenaAllocationTypeEligibility = enum {
        ok,
        contains_type_param,
        void_type,
        opaque_handle,
        drop_required,
        manual_init_drop_payload,
    };

    pub fn arenaAllocationTypeEligibility(self: *const SemanticModule, type_id: types.TypeId) ArenaAllocationTypeEligibility {
        if (self.typeContainsTypeParam(type_id)) return .contains_type_param;
        return switch (self.types.kind(type_id)) {
            .void => .void_type,
            .arena, .allocator => .opaque_handle,
            .manual_init => |manual_init| if (self.typeRequiresDropForArena(manual_init.payload, 0))
                .manual_init_drop_payload
            else
                .ok,
            else => if (self.typeRequiresDropForArena(type_id, 0)) .drop_required else .ok,
        };
    }

    fn typeRequiresDropForArena(self: *const SemanticModule, type_id: types.TypeId, depth: usize) bool {
        if (depth > self.types.count()) return false;
        if (self.hasDrop(type_id) != null) return true;
        return switch (self.types.kind(type_id)) {
            .struct_type => |struct_id| self.structRequiresDropForArena(struct_id, depth + 1),
            else => false,
        };
    }

    fn structRequiresDropForArena(self: *const SemanticModule, struct_id: hir.StructId, depth: usize) bool {
        if (struct_id.index >= self.hir.structs.items.len) return false;
        const struct_decl = self.hir.getStruct(struct_id);
        for (struct_decl.fields) |field_id| {
            const field = self.hir.getField(field_id);
            if (self.typeRequiresDropForArena(field.type_id, depth)) return true;
        }
        return false;
    }

    fn typeContainsTypeParam(self: *const SemanticModule, type_id: types.TypeId) bool {
        return switch (self.types.kind(type_id)) {
            .type_param => true,
            .pointer => |pointer| self.typeContainsTypeParam(pointer.pointee),
            .manual_init => |manual_init| self.typeContainsTypeParam(manual_init.payload),
            else => false,
        };
    }

    fn findIntrinsicDropConcept(self: *const SemanticModule) ?hir.ConceptId {
        for (self.hir.concepts.items, 0..) |concept, index| {
            if (std.mem.eql(u8, self.interner.text(concept.name), "Drop") and
                concept.type_params.len == 1 and
                !concept.is_marker)
            {
                return .{ .index = @intCast(index) };
            }
        }
        return null;
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
    return collectTopLevelDeclarationsWithOptions(allocator, unit, diagnostic_bag, .{});
}

pub fn collectTopLevelDeclarationsWithOptions(
    allocator: std.mem.Allocator,
    unit: ast.CompilationUnit,
    diagnostic_bag: *DiagnosticBag,
    options: CollectOptions,
) !SemanticModule {
    var module = try SemanticModule.init(allocator);
    errdefer module.deinit();

    var collector = Collector.init(allocator, &module, diagnostic_bag, options);
    defer collector.deinit();

    try collector.collect(unit);
    if (diagnostic_bag.count() != 0) return error.InvalidSemanticModule;
    return module;
}

const Collector = struct {
    allocator: std.mem.Allocator,
    module: *SemanticModule,
    diagnostics: *DiagnosticBag,
    options: CollectOptions,
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
        concept: hir.ConceptId,
    };

    fn init(allocator: std.mem.Allocator, module: *SemanticModule, diagnostic_bag: *DiagnosticBag, options: CollectOptions) Collector {
        return .{
            .allocator = allocator,
            .module = module,
            .diagnostics = diagnostic_bag,
            .options = options,
            .top_level_decls = std.AutoHashMap(interner.SymbolId, TopLevelDecl).init(allocator),
        };
    }

    fn deinit(self: *Collector) void {
        self.top_level_decls.deinit();
    }

    fn collect(self: *Collector, unit: ast.CompilationUnit) !void {
        try self.validateTestAttributePlacement(unit);
        if (self.diagnostics.count() != 0) return;

        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.declareFunction(function_decl),
                .template_decl => |template_decl| try self.declareGenericFunction(template_decl),
                .struct_decl => |struct_decl| try self.declareStruct(struct_decl),
                .enum_decl => |enum_decl| try self.declareEnum(enum_decl),
                .concept_decl => |concept_decl| try self.declareConcept(concept_decl),
                .interface_decl, .impl_decl, .static_assert_decl => {},
            }
        }

        if (self.diagnostics.count() != 0) return;

        for (unit.items) |item| {
            switch (item) {
                .concept_decl => |concept_decl| try self.resolveConcept(concept_decl),
                else => {},
            }
        }

        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.resolveFunction(function_decl),
                .template_decl => |template_decl| try self.resolveGenericFunction(template_decl),
                .struct_decl => |struct_decl| try self.resolveStruct(struct_decl),
                .enum_decl => |enum_decl| try self.resolveEnum(enum_decl),
                .concept_decl, .interface_decl, .static_assert_decl => {},
                .impl_decl => |impl_decl| try self.resolveImpl(impl_decl),
            }
        }

        if (self.diagnostics.count() == 0) {
            try self.validateTestAttributeSemantics();
        }
        if (self.diagnostics.count() != 0) return;

        for (unit.items) |item| {
            switch (item) {
                .function_decl => |function_decl| try self.lowerFunctionBody(function_decl),
                .template_decl => |template_decl| try self.lowerGenericFunctionBody(template_decl),
                .struct_decl, .enum_decl, .concept_decl, .interface_decl => {},
                .impl_decl => |impl_decl| try self.lowerImplFunctionBodies(impl_decl),
                .static_assert_decl => |static_assert_decl| try self.lowerStaticAssert(static_assert_decl),
            }
        }
    }

    fn validateTestAttributeSemantics(self: *Collector) !void {
        if (self.options.source_file_kind != .@"test") return;

        for (self.module.hir.functions.items) |function| {
            const summary = self.testAttributeSummary(function.attributes);
            if (summary.fact_count == 0 and summary.theory_count == 0 and summary.inline_data_count == 0) continue;

            if (summary.fact_count > 1) {
                try self.diagnostics.append(diagnostics.duplicateTestAttribute(summary.second_fact_span.?));
                return;
            }
            if (summary.theory_count > 1) {
                try self.diagnostics.append(diagnostics.duplicateTestAttribute(summary.second_theory_span.?));
                return;
            }
            if (summary.fact_count != 0 and summary.theory_count != 0) {
                try self.diagnostics.append(diagnostics.conflictingTestAttributes(summary.theory_span.?));
                return;
            }
            if (summary.inline_data_count != 0 and summary.theory_count == 0) {
                try self.diagnostics.append(diagnostics.inlineDataRequiresTheory(summary.first_inline_data_span.?));
                return;
            }
            if (summary.fact_count != 0) {
                if (function.params.len != 0) {
                    try self.diagnostics.append(diagnostics.factRequiresZeroArgFunction(summary.fact_span.?));
                    return;
                }
                if (!sameType(function.return_type, self.module.types.voidType())) {
                    try self.diagnostics.append(diagnostics.testFunctionReturnTypeInvalid(summary.fact_span.?));
                    return;
                }
            }
            if (summary.theory_count != 0) {
                if (summary.inline_data_count == 0) {
                    try self.diagnostics.append(diagnostics.theoryRequiresInlineData(summary.theory_span.?));
                    return;
                }
                if (!sameType(function.return_type, self.module.types.voidType())) {
                    try self.diagnostics.append(diagnostics.testFunctionReturnTypeInvalid(summary.theory_span.?));
                    return;
                }
                for (function.attributes) |attribute| {
                    const name = self.module.interner.text(attribute.name);
                    if (!std.mem.eql(u8, name, "InlineData")) continue;
                    if (attribute.args.len != function.params.len) {
                        try self.diagnostics.append(diagnostics.inlineDataArityMismatch(attribute.span));
                        return;
                    }
                    for (attribute.args, function.params) |arg, param_id| {
                        const param = self.module.hir.getParam(param_id);
                        if (!self.inlineDataArgMatchesParam(arg, param.type_id)) {
                            try self.diagnostics.append(diagnostics.inlineDataTypeMismatch(attribute.span));
                            return;
                        }
                    }
                }
            }
        }
    }

    const TestAttributeSummary = struct {
        fact_count: usize = 0,
        theory_count: usize = 0,
        inline_data_count: usize = 0,
        fact_span: ?source.SourceSpan = null,
        second_fact_span: ?source.SourceSpan = null,
        theory_span: ?source.SourceSpan = null,
        second_theory_span: ?source.SourceSpan = null,
        first_inline_data_span: ?source.SourceSpan = null,
    };

    fn testAttributeSummary(self: *Collector, attributes: []const hir.HirAttribute) TestAttributeSummary {
        var summary = TestAttributeSummary{};
        for (attributes) |attribute| {
            const name = self.module.interner.text(attribute.name);
            if (std.mem.eql(u8, name, "Fact")) {
                summary.fact_count += 1;
                if (summary.fact_span == null) summary.fact_span = attribute.span else if (summary.second_fact_span == null) summary.second_fact_span = attribute.span;
            } else if (std.mem.eql(u8, name, "Theory")) {
                summary.theory_count += 1;
                if (summary.theory_span == null) summary.theory_span = attribute.span else if (summary.second_theory_span == null) summary.second_theory_span = attribute.span;
            } else if (std.mem.eql(u8, name, "InlineData")) {
                summary.inline_data_count += 1;
                if (summary.first_inline_data_span == null) summary.first_inline_data_span = attribute.span;
            }
        }
        return summary;
    }

    fn inlineDataArgMatchesParam(self: *Collector, arg: hir.HirAttributeArg, param_type: types.TypeId) bool {
        return switch (arg) {
            .int_literal => sameType(param_type, self.module.types.intType()),
            .bool_literal => sameType(param_type, self.module.types.boolType()),
            .string_literal => false,
        };
    }

    fn validateTestAttributePlacement(self: *Collector, unit: ast.CompilationUnit) !void {
        if (self.options.source_file_kind == .@"test") return;

        for (unit.items) |item| {
            const attributes = itemAttributes(item);
            for (attributes) |attribute| {
                if (isTestAttributeName(attribute.name.parts)) {
                    try self.diagnostics.append(diagnostics.testAttributeOutsideTestFile(attribute.span));
                }
            }
        }
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Declaration resolution
    // ─────────────────────────────────────────────────────────────────────────────

    fn declareIntrinsicDropConcept(self: *Collector) !hir.ConceptId {
        const drop_name = try self.module.interner.intern("Drop");
        if (self.top_level_decls.get(drop_name)) |decl| switch (decl) {
            .concept => |id| return id,
            else => {},
        };
        const concept_id = try self.module.hir.addConcept(drop_name, false, false, hir.synthetic_span);
        try self.top_level_decls.put(drop_name, .{ .concept = concept_id });

        const type_param_name = try self.module.interner.intern("T");
        const type_param = try self.module.types.addTypeParam(.{ .kind = .concept, .index = concept_id.index }, 0, type_param_name);
        const type_params = try self.allocator.alloc(hir.HirTypeParam, 1);
        type_params[0] = .{
            .name = type_param_name,
            .span = hir.synthetic_span,
            .type_id = type_param,
        };
        self.module.hir.setConceptTypeParams(concept_id, type_params);

        const value_name = try self.module.interner.intern("value");
        const drop_name_symbol = try self.module.interner.intern("drop");
        const params = try self.allocator.alloc(hir.HirConceptParam, 1);
        params[0] = .{
            .name = value_name,
            .span = hir.synthetic_span,
            .type_id = type_param,
        };
        const requirements = try self.allocator.alloc(hir.HirConceptRequirement, 1);
        requirements[0] = .{
            .name = drop_name_symbol,
            .return_type = self.module.types.voidType(),
            .params = params,
            .span = hir.synthetic_span,
        };
        self.module.hir.setConceptRequirements(concept_id, requirements);
        return concept_id;
    }

    fn declareFunction(self: *Collector, function_decl: ast.FunctionDecl) !void {
        const name = try self.internFreshTopLevelName(
            function_decl.signature.name.base.text,
            function_decl.signature.name.base.span,
        ) orelse return;
        const function_id = try self.module.hir.addFunctionWithSafety(name, self.module.types.voidType(), function_decl.is_unsafe, function_decl.span);
        self.module.hir.setFunctionAttributes(function_id, try self.copyAttributes(function_decl.attributes));
        self.module.hir.setFunctionAllocationEffect(function_id, lowerAllocationEffect(function_decl.allocation_effect));
        if (function_decl.is_compile_time) {
            self.module.hir.markFunctionCompileTime(function_id);
            try self.copyCompileTimeCapabilities(function_id, function_decl.compile_time_capabilities);
        }
        try self.top_level_decls.put(name, .{ .function = function_id });
    }

    fn copyCompileTimeCapabilities(self: *Collector, function_id: hir.FunctionId, capabilities: []const ast.CompileTimeCapabilitySyntax) !void {
        if (capabilities.len == 0) return;
        var owned = try self.allocator.alloc(hir.CompileTimeCapabilityRequired, capabilities.len);
        errdefer self.allocator.free(owned);
        var initialized: usize = 0;
        errdefer for (owned[0..initialized]) |capability| self.allocator.free(capability.name);
        for (capabilities, 0..) |capability, index| {
            owned[index] = .{
                .name = try self.allocator.dupe(u8, capability.name.text),
                .span = capability.name.span,
            };
            initialized += 1;
        }
        self.module.hir.setFunctionCompileTimeCapabilities(function_id, owned);
    }

    fn declareGenericFunction(self: *Collector, template_decl: ast.TemplateDecl) !void {
        const name = try self.internFreshTopLevelName(
            template_decl.body.signature.name.base.text,
            template_decl.body.signature.name.base.span,
        ) orelse return;
        const generic_id = try self.module.hir.addGenericFunction(name, template_decl.span);
        self.module.hir.setGenericFunctionAttributes(generic_id, try self.copyAttributes(template_decl.attributes));
        self.module.hir.setFunctionAllocationEffect(self.module.hir.getGenericFunction(generic_id).function, lowerAllocationEffect(template_decl.body.allocation_effect));
        try self.top_level_decls.put(name, .{ .generic_function = generic_id });
    }

    fn declareStruct(self: *Collector, struct_decl: ast.StructDecl) !void {
        const name = try self.internFreshTopLevelName(struct_decl.name.text, struct_decl.name.span) orelse return;
        const struct_id = try self.module.hir.addStruct(name);
        self.module.hir.setStructAttributes(struct_id, try self.copyAttributes(struct_decl.attributes));
        const type_id = try self.module.types.addStructType(struct_id);
        try self.top_level_decls.put(name, .{ .struct_ = .{ .id = struct_id, .type_id = type_id } });
    }

    fn declareEnum(self: *Collector, enum_decl: ast.EnumDecl) !void {
        const name = try self.internFreshTopLevelName(enum_decl.name.text, enum_decl.name.span) orelse return;
        const enum_id = try self.module.hir.addEnum(name, enum_decl.is_must_use);
        self.module.hir.setEnumAttributes(enum_id, try self.copyAttributes(enum_decl.attributes));
        const type_id = try self.module.types.addEnumType(enum_id);
        try self.top_level_decls.put(name, .{ .enum_ = .{ .id = enum_id, .type_id = type_id } });
    }

    fn declareConcept(self: *Collector, concept_decl: ast.ConceptDecl) !void {
        const name = try self.internFreshTopLevelName(concept_decl.name.text, concept_decl.name.span) orelse return;
        const concept_id = try self.module.hir.addConcept(name, concept_decl.is_marker, concept_decl.is_unsafe, concept_decl.span);
        self.module.hir.setConceptAttributes(concept_id, try self.copyAttributes(concept_decl.attributes));
        try self.top_level_decls.put(name, .{ .concept = concept_id });
    }

    fn copyAttributes(self: *Collector, attributes: []const ast.Attribute) ![]hir.HirAttribute {
        if (attributes.len == 0) return &.{};
        var owned = try self.allocator.alloc(hir.HirAttribute, attributes.len);
        var initialized: usize = 0;
        errdefer {
            for (owned[0..initialized]) |attribute| {
                for (attribute.args) |arg| switch (arg) {
                    .int_literal => |text| self.allocator.free(text),
                    .string_literal => |text| self.allocator.free(text),
                    .bool_literal => {},
                };
                if (attribute.args.len > 0) self.allocator.free(attribute.args);
            }
            self.allocator.free(owned);
        }

        for (attributes, 0..) |attribute, index| {
            const name = try self.module.interner.intern(attribute.name.parts[0].text);
            const args = if (attribute.arguments) |arguments| try self.copyAttributeArgs(arguments.args) else @constCast(&.{});
            owned[index] = .{
                .name = name,
                .args = args,
                .has_arguments = attribute.arguments != null,
                .span = attribute.span,
            };
            initialized += 1;
        }
        return owned;
    }

    fn copyAttributeArgs(self: *Collector, args: []const ast.AttributeArg) ![]hir.HirAttributeArg {
        if (args.len == 0) return &.{};
        var owned = try self.allocator.alloc(hir.HirAttributeArg, args.len);
        var initialized: usize = 0;
        errdefer {
            for (owned[0..initialized]) |arg| switch (arg) {
                .int_literal => |text| self.allocator.free(text),
                .string_literal => |text| self.allocator.free(text),
                .bool_literal => {},
            };
            self.allocator.free(owned);
        }

        for (args, 0..) |arg, index| {
            owned[index] = switch (arg) {
                .int_literal => |literal| .{ .int_literal = try self.allocator.dupe(u8, literal.text) },
                .bool_literal => |literal| .{ .bool_literal = literal.value },
                .string_literal => |literal| .{ .string_literal = try self.allocator.dupe(u8, literal.text) },
            };
            initialized += 1;
        }
        return owned;
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
            const constraint = if (param.constraint) |constraint_type| try self.resolveTypeConstraint(constraint_type, &type_scope) else null;
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

    fn resolveTypeConstraint(self: *Collector, type_name: ast.TypeName, type_scope: *TypeParamScope) !?hir.HirTypeConstraint {
        const text = try self.renderTypeName(type_name);
        var text_owned = true;
        defer if (text_owned) self.allocator.free(text);
        if (type_name.is_mut or type_name.is_reference or type_name.is_pointer or type_name.name.parts.len != 1) {
            try self.diagnostics.append(diagnostics.unsupportedConceptConstraint(type_name.span));
            return null;
        }
        const concept_symbol = try self.module.interner.intern(type_name.name.parts[0].text);
        const concept_id = switch (self.top_level_decls.get(concept_symbol) orelse {
            try self.diagnostics.append(diagnostics.unknownConceptConstraint(type_name.name.parts[0].span));
            return null;
        }) {
            .concept => |id| id,
            else => {
                try self.diagnostics.append(diagnostics.unknownConceptConstraint(type_name.name.parts[0].span));
                return null;
            },
        };
        const concept = self.module.hir.getConcept(concept_id);
        if (type_name.generic_args.len != concept.type_params.len) {
            try self.diagnostics.append(diagnostics.conceptConstraintArityMismatch(type_name.span));
            return null;
        }
        if (type_name.generic_args.len != 1) {
            try self.diagnostics.append(diagnostics.unsupportedConceptConstraint(type_name.span));
            return null;
        }
        var type_args = try self.allocator.alloc(types.TypeId, type_name.generic_args.len);
        var type_args_owned = true;
        defer if (type_args_owned) self.allocator.free(type_args);
        for (type_name.generic_args, 0..) |arg, index| {
            type_args[index] = (try self.resolveTypeNameScoped(arg, type_scope)) orelse return null;
        }
        text_owned = false;
        type_args_owned = false;
        return hir.HirTypeConstraint{ .text = text, .span = type_name.span, .concept_id = concept_id, .type_args = type_args };
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

    fn resolveConcept(self: *Collector, concept_decl: ast.ConceptDecl) !void {
        const concept_symbol = try self.module.interner.intern(concept_decl.name.text);
        const concept_id = switch (self.top_level_decls.get(concept_symbol).?) {
            .concept => |id| id,
            else => unreachable,
        };
        const owner = types.TypeParamOwner{ .kind = .concept, .index = concept_id.index };

        var type_scope = TypeParamScope.init(self.allocator);
        defer type_scope.deinit();
        var hir_params = std.ArrayList(hir.HirTypeParam).empty;
        defer hir_params.deinit(self.allocator);

        for (concept_decl.generic_params, 0..) |param, index| {
            const symbol = try self.module.interner.intern(param.text);
            const type_id = try self.module.types.addTypeParam(owner, @intCast(index), symbol);
            try type_scope.put(symbol, type_id);
            try hir_params.append(self.allocator, .{ .name = symbol, .span = param.span, .type_id = type_id });
        }
        const owned_type_params = try self.allocator.alloc(hir.HirTypeParam, hir_params.items.len);
        @memcpy(owned_type_params, hir_params.items);
        hir_params.clearRetainingCapacity();
        self.module.hir.setConceptTypeParams(concept_id, owned_type_params);
        if (concept_decl.is_marker) {
            self.module.hir.setConceptKnownMarkerKind(concept_id, hir.MarkerKind.fromDeclaredName(concept_decl.name.text, owned_type_params.len));
        }

        var requirement_keys = std.AutoHashMap(RequirementKey, source.SourceSpan).init(self.allocator);
        defer requirement_keys.deinit();
        var requirements = std.ArrayList(hir.HirConceptRequirement).empty;
        defer requirements.deinit(self.allocator);
        errdefer {
            for (requirements.items) |requirement| if (requirement.params.len > 0) self.allocator.free(requirement.params);
        }

        for (concept_decl.signatures) |signature| {
            const requirement_symbol = try self.module.interner.intern(signature.name.base.text);
            const key = RequirementKey{ .name = requirement_symbol, .arity = @intCast(signature.params.len) };
            if (requirement_keys.contains(key)) {
                try self.diagnostics.append(diagnostics.duplicateConceptRequirement(signature.name.span));
                continue;
            }
            try requirement_keys.put(key, signature.name.span);

            const return_type = (try self.resolveTypeNameScoped(signature.return_type, &type_scope)) orelse continue;
            var params = std.ArrayList(hir.HirConceptParam).empty;
            errdefer params.deinit(self.allocator);
            for (signature.params) |param| {
                const param_symbol = try self.module.interner.intern(param.name.text);
                if (try self.resolveTypeNameScoped(param.type_name, &type_scope)) |type_id| {
                    try params.append(self.allocator, .{ .name = param_symbol, .span = param.span, .type_id = type_id });
                }
            }
            try requirements.append(self.allocator, .{
                .name = requirement_symbol,
                .return_type = return_type,
                .params = try params.toOwnedSlice(self.allocator),
                .span = signature.span,
            });
        }

        const owned_requirements = try self.allocator.alloc(hir.HirConceptRequirement, requirements.items.len);
        @memcpy(owned_requirements, requirements.items);
        requirements.clearRetainingCapacity();
        self.module.hir.setConceptRequirements(concept_id, owned_requirements);
    }

    fn resolveImpl(self: *Collector, impl_decl: ast.ImplDecl) !void {
        const concept_part = if (impl_decl.concept_name.name.parts.len == 1) impl_decl.concept_name.name.parts[0] else {
            try self.diagnostics.append(diagnostics.unknownConcept(impl_decl.concept_name.span));
            return;
        };
        if (impl_decl.concept_name.is_mut or impl_decl.concept_name.is_reference or impl_decl.concept_name.is_pointer) {
            try self.diagnostics.append(diagnostics.unknownConcept(impl_decl.concept_name.span));
            return;
        }
        const concept_symbol = try self.module.interner.intern(concept_part.text);
        const maybe_decl = self.top_level_decls.get(concept_symbol) orelse blk: {
            if (std.mem.eql(u8, concept_part.text, "Drop")) {
                break :blk TopLevelDecl{ .concept = try self.declareIntrinsicDropConcept() };
            }
            try self.diagnostics.append(diagnostics.unknownConcept(concept_part.span));
            return;
        };
        const concept_id = switch (maybe_decl) {
            .concept => |id| id,
            else => {
                try self.diagnostics.append(diagnostics.unknownConcept(concept_part.span));
                return;
            },
        };
        const concept = self.module.hir.getConcept(concept_id);

        if (impl_decl.target_types.len != concept.type_params.len or impl_decl.target_types.len != 1) {
            try self.diagnostics.append(diagnostics.conceptArityMismatch(impl_decl.concept_name.span));
            return;
        }

        const target_type = (try self.resolveTypeName(impl_decl.target_types[0])) orelse return;
        if (self.typeContainsTypeParam(target_type)) {
            try self.diagnostics.append(diagnostics.invalidImplTarget(impl_decl.target_types[0].span));
            return;
        }

        if (self.module.hir.hasConceptImpl(concept_id, target_type)) {
            try self.diagnostics.append(diagnostics.duplicateImpl(impl_decl.span));
            return;
        }

        if (impl_decl.is_unsafe and !concept.is_unsafe) {
            try self.diagnostics.append(diagnostics.unsafeImplNotAllowed(impl_decl.span));
            return;
        }
        if (concept.is_unsafe and !impl_decl.is_unsafe) {
            try self.diagnostics.append(diagnostics.unsafeImplRequired(impl_decl.span));
            return;
        }

        if (concept.is_marker) {
            if (impl_decl.functions.len != 0) {
                try self.diagnostics.append(diagnostics.markerConceptImplCannotHaveFunctions(impl_decl.functions[0].span));
                return;
            }
            const impl_id = try self.module.hir.addConceptImpl(concept_id, target_type, &.{}, impl_decl.is_unsafe, impl_decl.span);
            self.module.hir.setConceptImplAttributes(impl_id, try self.copyAttributes(impl_decl.attributes));
            return;
        }

        try self.validateOrdinaryConceptImpl(concept_id, concept.*, target_type, impl_decl);
    }

    fn validateOrdinaryConceptImpl(self: *Collector, concept_id: hir.ConceptId, concept: hir.HirConcept, target_type: types.TypeId, impl_decl: ast.ImplDecl) !void {
        var seen = std.AutoHashMap(interner.SymbolId, source.SourceSpan).init(self.allocator);
        defer seen.deinit();
        var witness_functions = std.ArrayList(hir.FunctionId).empty;
        defer witness_functions.deinit(self.allocator);

        for (impl_decl.functions) |function_decl| {
            if (self.isIntrinsicDropConcept(concept) and function_decl.is_compile_time) {
                try self.diagnostics.append(diagnostics.invalidDropImpl(function_decl.span));
                continue;
            }
            const function_symbol = try self.module.interner.intern(function_decl.signature.name.base.text);
            if (seen.contains(function_symbol)) {
                try self.diagnostics.append(diagnostics.duplicateConceptImplFunction(function_decl.signature.name.span));
                continue;
            }
            try seen.put(function_symbol, function_decl.signature.name.span);

            const requirement = self.findRequirement(concept, function_symbol) orelse {
                try self.diagnostics.append(diagnostics.extraConceptImplFunction(function_decl.signature.name.span));
                continue;
            };
            if (function_decl.signature.params.len != requirement.params.len) {
                try self.diagnostics.append(diagnostics.invalidConceptRequirementImplSignature(function_decl.signature.span));
                continue;
            }
            const return_type = (try self.resolveTypeName(function_decl.signature.return_type)) orelse continue;
            if (!sameType(return_type, try self.substituteConceptType(requirement.return_type, concept, target_type))) {
                try self.diagnostics.append(diagnostics.invalidConceptRequirementImplSignature(function_decl.signature.return_type.span));
                continue;
            }
            var param_types_ok = true;
            for (function_decl.signature.params, 0..) |param, index| {
                const param_type = (try self.resolveTypeName(param.type_name)) orelse {
                    param_types_ok = false;
                    continue;
                };
                const expected = try self.substituteConceptType(requirement.params[index].type_id, concept, target_type);
                if (!sameType(param_type, expected)) {
                    try self.diagnostics.append(diagnostics.invalidConceptRequirementImplSignature(param.type_name.span));
                    param_types_ok = false;
                }
            }
            if (!param_types_ok) continue;

            const function_id = try self.module.hir.addConceptWitnessFunction(function_symbol, return_type, function_decl.is_unsafe, function_decl.span);
            self.module.hir.setFunctionAllocationEffect(function_id, lowerAllocationEffect(function_decl.allocation_effect));
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
            try witness_functions.append(self.allocator, function_id);
        }

        for (concept.requirements) |requirement| {
            if (!seen.contains(requirement.name)) {
                try self.diagnostics.append(diagnostics.missingConceptRequirementImpl(impl_decl.span));
            }
        }
        if (self.diagnostics.count() != 0) return;

        const owned = try self.allocator.alloc(hir.FunctionId, witness_functions.items.len);
        @memcpy(owned, witness_functions.items);
        witness_functions.clearRetainingCapacity();
        const impl_id = try self.module.hir.addConceptImpl(concept_id, target_type, owned, impl_decl.is_unsafe, impl_decl.span);
        self.module.hir.setConceptImplAttributes(impl_id, try self.copyAttributes(impl_decl.attributes));
        if (self.isIntrinsicDropConcept(concept)) {
            for (owned) |function_id| self.module.hir.markConceptWitnessReferenced(function_id);
        }
    }

    fn isIntrinsicDropConcept(self: *Collector, concept: hir.HirConcept) bool {
        return !concept.is_marker and
            concept.type_params.len == 1 and
            std.mem.eql(u8, self.module.interner.text(concept.name), "Drop");
    }

    fn findRequirement(self: *Collector, concept: hir.HirConcept, name: interner.SymbolId) ?hir.HirConceptRequirement {
        _ = self;
        for (concept.requirements) |requirement| {
            if (requirement.name.index == name.index) return requirement;
        }
        return null;
    }

    fn substituteConceptType(self: *Collector, type_id: types.TypeId, concept: hir.HirConcept, target_type: types.TypeId) !types.TypeId {
        return switch (self.module.types.kind(type_id)) {
            .type_param => blk: {
                for (concept.type_params) |param| {
                    if (param.type_id.index == type_id.index) break :blk target_type;
                }
                break :blk type_id;
            },
            .pointer => |pointer| try self.module.types.addPointerType(try self.substituteConceptType(pointer.pointee, concept, target_type)),
            else => type_id,
        };
    }

    fn typeContainsTypeParam(self: *Collector, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .type_param => true,
            .pointer => |pointer| self.typeContainsTypeParam(pointer.pointee),
            else => false,
        };
    }

    fn lowerImplFunctionBodies(self: *Collector, impl_decl: ast.ImplDecl) !void {
        if (impl_decl.functions.len == 0) return;
        const concept_part = if (impl_decl.concept_name.name.parts.len == 1) impl_decl.concept_name.name.parts[0] else return;
        const concept_symbol = try self.module.interner.intern(concept_part.text);
        const concept_id = switch (self.top_level_decls.get(concept_symbol) orelse return) {
            .concept => |id| id,
            else => return,
        };
        if (impl_decl.target_types.len != 1) return;
        const target_type = (try self.resolveTypeName(impl_decl.target_types[0])) orelse return;
        const impl_id = self.module.hir.findConceptImpl(concept_id, target_type) orelse return;
        const concept_impl = self.module.hir.getConceptImpl(impl_id);
        for (impl_decl.functions, concept_impl.functions) |function_decl, function_id| {
            const body = function_decl.body orelse continue;
            const block = body.block orelse continue;
            var lowerer = BodyLowerer.init(self, function_id);
            defer lowerer.deinit();
            if (try lowerer.lowerBlock(block)) |body_id| self.module.hir.setFunctionBody(function_id, body_id);
        }
    }

    const RequirementKey = struct {
        name: interner.SymbolId,
        arity: u32,
    };

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

    fn resolveTypeName(self: *Collector, type_name: ast.TypeName) anyerror!?types.TypeId {
        return self.resolveTypeNameScoped(type_name, null);
    }

    fn resolveTypeNameScoped(self: *Collector, type_name: ast.TypeName, type_scope: ?*TypeParamScope) anyerror!?types.TypeId {
        if (type_name.is_mut or type_name.is_reference or type_name.name.parts.len != 1) {
            try self.diagnostics.append(diagnostics.unsupportedTypeSyntax(type_name.span));
            return null;
        }

        const pointee = try self.resolveBaseTypeNameScoped(type_name, type_scope) orelse return null;
        if (type_name.is_pointer) return try self.module.types.addPointerType(pointee);
        return pointee;
    }

    fn resolveBaseTypeNameScoped(self: *Collector, type_name: ast.TypeName, type_scope: ?*TypeParamScope) anyerror!?types.TypeId {
        const part = type_name.name.parts[0];
        if (std.mem.eql(u8, part.text, "void")) return self.module.types.voidType();
        if (std.mem.eql(u8, part.text, "int")) return self.module.types.intType();
        if (std.mem.eql(u8, part.text, "bool")) return self.module.types.boolType();
        if (std.mem.eql(u8, part.text, "Arena")) return self.module.types.arenaType();
        if (std.mem.eql(u8, part.text, "Allocator")) return self.module.types.allocatorType();
        if (std.mem.eql(u8, part.text, "AllocError")) return self.module.types.allocErrorType();
        if (std.mem.eql(u8, part.text, "ManualInit")) {
            if (type_name.generic_args.len != 1) {
                try self.diagnostics.append(diagnostics.manualInitRequiresTypeArgument(type_name.span));
                return null;
            }
            const payload = (try self.resolveTypeNameScoped(type_name.generic_args[0], type_scope)) orelse return null;
            return try self.module.types.addManualInitType(payload);
        }
        if (type_name.generic_args.len != 0) {
            try self.diagnostics.append(diagnostics.unsupportedTypeSyntax(type_name.span));
            return null;
        }

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
            .function, .generic_function, .concept => blk: {
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
        if (self.top_level_decls.contains(symbol) or isCompilerKnownTypeName(text)) {
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

    fn lowerStaticAssert(self: *Collector, static_assert_decl: ast.StaticAssertDecl) !void {
        var lowerer = BodyLowerer.initStatic(self);
        defer lowerer.deinit();
        const expr = (try lowerer.lowerExpr(static_assert_decl.expr.*)) orelse return;
        try self.module.hir.addStaticAssert(expr, static_assert_decl.span);
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

fn itemAttributes(item: ast.Item) []const ast.Attribute {
    return switch (item) {
        .function_decl => |decl| decl.attributes,
        .template_decl => |decl| decl.attributes,
        .struct_decl => |decl| decl.attributes,
        .enum_decl => |decl| decl.attributes,
        .concept_decl => |decl| decl.attributes,
        .interface_decl => |decl| decl.attributes,
        .impl_decl => |decl| decl.attributes,
        .static_assert_decl => &.{},
    };
}

fn lowerAllocationEffect(effect: ast.AllocationEffect) hir.AllocationEffect {
    return switch (effect) {
        .unspecified => .unspecified,
        .noalloc => .noalloc,
        .alloc => .alloc,
    };
}

fn isTestAttributeName(parts: []const ast.NameSegment) bool {
    if (parts.len != 1) return false;
    const text = parts[0].text;
    return std.mem.eql(u8, text, "Fact") or
        std.mem.eql(u8, text, "Theory") or
        std.mem.eql(u8, text, "InlineData");
}

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
    function_id: ?hir.FunctionId,
    bindings: std.ArrayList(ScopedBinding),
    depth: usize = 0,

    fn init(collector: *Collector, function_id: hir.FunctionId) BodyLowerer {
        return .{
            .collector = collector,
            .function_id = function_id,
            .bindings = std.ArrayList(ScopedBinding).empty,
        };
    }

    fn initStatic(collector: *Collector) BodyLowerer {
        return .{
            .collector = collector,
            .function_id = null,
            .bindings = std.ArrayList(ScopedBinding).empty,
        };
    }

    fn deinit(self: *BodyLowerer) void {
        self.bindings.deinit(self.collector.allocator);
    }

    fn seedParams(self: *BodyLowerer) !void {
        const function = self.collector.module.hir.getFunction(self.function_id.?);
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
                const local_id = try self.collector.module.hir.addLocal(self.function_id.?, local_symbol, type_id, local_decl.span);
                try self.bindings.append(self.collector.allocator, .{ .name = local_symbol, .binding = .{ .local = local_id }, .depth = self.depth });
                return try self.collector.module.hir.addStmt(.{ .local_decl = .{ .local = local_id, .initializer = initializer } }, local_decl.span);
            },
            .expr_stmt => |expr_stmt| {
                if (try self.lowerArenaStorageOpStmt(expr_stmt.value.*)) |stmt_id| return stmt_id;
                const value = (try self.lowerExpr(expr_stmt.value.*)) orelse return null;
                return try self.collector.module.hir.addStmt(.{ .expr_stmt = value }, expr_stmt.span);
            },
            .discard_stmt => |discard_stmt| {
                if (try self.lowerArenaStorageOpStmt(discard_stmt.value.*)) |stmt_id| return stmt_id;
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
            .string_literal => |lit| {
                try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.TypeMismatch, .@"error", "string literals are only supported as test intrinsic reasons in Phase 11 v0", lit.span));
                return null;
            },
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
                if (self.isArenaAllocCall(call)) {
                    return try self.lowerArenaAlloc(call);
                }
                if (self.arenaStorageOpKind(call) != null) {
                    try self.collector.diagnostics.append(diagnostics.Diagnostic.init(
                        .InvalidCall,
                        .@"error",
                        "Arena.reset and Arena.destroy are statement-only operations",
                        call.span,
                    ));
                    return null;
                }
                if (self.maybeTestIntrinsicCall(call)) |intrinsic| {
                    return try self.lowerTestIntrinsic(call, intrinsic);
                }
                if (self.maybeTestRelationConstructorCall(call)) {
                    const diagnostic = if (self.collector.options.source_file_kind == .@"test")
                        diagnostics.testRelationOutsideExpectThat(call.span)
                    else
                        diagnostics.testIntrinsicOutsideTestFile(call.span);
                    try self.collector.diagnostics.append(diagnostic);
                    return null;
                }
                if (call.qualifier) |qualifier| {
                    try self.collector.diagnostics.append(diagnostics.unknownFunction(qualifier.span));
                    return null;
                }
                const symbol = try self.collector.module.interner.intern(call.callee.text);
                var args = std.ArrayList(hir.ExprId).empty;
                defer args.deinit(self.collector.allocator);
                for (call.args) |arg| {
                    const arg_id = (try self.lowerExpr(arg.*)) orelse return null;
                    try args.append(self.collector.allocator, arg_id);
                }
                const owned = try self.collector.allocator.alloc(hir.ExprId, args.items.len);
                var owned_args_transferred = false;
                defer if (!owned_args_transferred and owned.len > 0) self.collector.allocator.free(owned);
                @memcpy(owned, args.items);

                if (std.mem.eql(u8, call.callee.text, "manualAssumeInit")) {
                    if (call.type_args.len != 0) {
                        try self.collector.diagnostics.append(diagnostics.manualInitInvalidOperation(call.span));
                        return null;
                    }
                    if (owned.len != 1) {
                        try self.collector.diagnostics.append(diagnostics.manualInitInvalidOperation(call.span));
                        return null;
                    }
                    const expr_id = try self.collector.module.hir.addExpr(.{ .manual_init_assume = owned[0] }, call.span);
                    return expr_id;
                }

                if (self.collector.top_level_decls.get(symbol)) |decl| {
                    const function_id = switch (decl) {
                        .function => |id| id,
                        .generic_function => |id| self.collector.module.hir.getGenericFunction(id).function,
                        else => {
                            try self.collector.diagnostics.append(diagnostics.unknownFunction(call.callee.span));
                            return null;
                        },
                    };
                    const expr_id = try self.collector.module.hir.addExpr(.{ .call = .{ .function = function_id, .args = owned } }, call.span);
                    owned_args_transferred = true;
                    return expr_id;
                }

                const diagnostic_count_before_requirement_lookup = self.collector.diagnostics.count();
                if (try self.resolveConceptRequirementCall(symbol, @intCast(call.args.len), call.callee.span)) |requirement_call| {
                    const expr_id = try self.collector.module.hir.addExpr(.{ .concept_requirement_call = .{ .concept_id = requirement_call.concept_id, .requirement_index = requirement_call.requirement_index, .args = owned } }, call.span);
                    owned_args_transferred = true;
                    return expr_id;
                }
                if (self.collector.diagnostics.count() != diagnostic_count_before_requirement_lookup) return null;

                try self.collector.diagnostics.append(diagnostics.unknownFunction(call.callee.span));
                return null;
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
            .move_expr => |move_expr| {
                const operand = (try self.lowerExpr(move_expr.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .move_expr = operand }, move_expr.span);
            },
            .try_expr => |try_expr| {
                const operand = (try self.lowerExpr(try_expr.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .try_expr = operand }, try_expr.span);
            },
            .compile_time => |compile_time_expr| {
                const operand = (try self.lowerExpr(compile_time_expr.operand.*)) orelse return null;
                return try self.collector.module.hir.addExpr(.{ .compile_time = .{ .operand = operand, .span = compile_time_expr.span } }, compile_time_expr.span);
            },
            .field_access => |field_access| {
                if (self.isUnboundTargetRoot(field_access.receiver.*)) {
                    const query = compile_time_target.lookupTargetQuery(field_access.field_name.text) orelse {
                        try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.CompileTimeUnknownTargetField, .@"error", "unknown compile-time target metadata field", field_access.field_name.span));
                        return null;
                    };
                    return try self.collector.module.hir.addExpr(.{ .target_metadata = .{
                        .query = query,
                        .field_span = field_access.field_name.span,
                    } }, field_access.span);
                }
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

    const ParsedTestIntrinsic = struct {
        family: enum { assert, expect },
        name: []const u8,
    };

    const ParsedTestRelation = struct {
        kind: enum { is_true, is_false, equal },
        expected: ?hir.ExprId = null,
        span: source.SourceSpan,
    };

    fn maybeTestIntrinsicCall(self: *BodyLowerer, call: ast.Expr.CallExpr) ?ParsedTestIntrinsic {
        _ = self;
        const qualifier = call.qualifier orelse return null;
        if (std.mem.eql(u8, qualifier.text, "Assert")) return .{ .family = .assert, .name = call.callee.text };
        if (std.mem.eql(u8, qualifier.text, "Expect")) return .{ .family = .expect, .name = call.callee.text };
        return null;
    }

    fn maybeTestRelationConstructorCall(self: *BodyLowerer, call: ast.Expr.CallExpr) bool {
        _ = self;
        const qualifier = call.qualifier orelse return false;
        return std.mem.eql(u8, qualifier.text, "Is");
    }

    fn lowerTestIntrinsic(self: *BodyLowerer, call: ast.Expr.CallExpr, intrinsic: ParsedTestIntrinsic) !?hir.ExprId {
        if (self.collector.options.source_file_kind != .@"test") {
            try self.collector.diagnostics.append(diagnostics.testIntrinsicOutsideTestFile(call.span));
            return null;
        }

        const expected_arity: usize = if (std.mem.eql(u8, intrinsic.name, "True") or std.mem.eql(u8, intrinsic.name, "False"))
            2
        else if (intrinsic.family == .expect and std.mem.eql(u8, intrinsic.name, "Equal"))
            3
        else if (intrinsic.family == .expect and std.mem.eql(u8, intrinsic.name, "That"))
            3
        else {
            try self.collector.diagnostics.append(diagnostics.unknownFunction(call.callee.span));
            return null;
        };

        if (call.args.len < expected_arity) {
            try self.collector.diagnostics.append(diagnostics.testExpectationRequiresReason(call.span));
            return null;
        }
        if (call.args.len != expected_arity) {
            try self.collector.diagnostics.append(diagnostics.testIntrinsicArityMismatch(call.span));
            return null;
        }

        const reason_arg = call.args[call.args.len - 1].*;
        const reason = switch (reason_arg) {
            .string_literal => |literal| literal,
            else => {
                try self.collector.diagnostics.append(diagnostics.testExpectationRequiresReason(reason_arg.span()));
                return null;
            },
        };
        const reason_text = stringLiteralContents(reason.text);
        if (std.mem.trim(u8, reason_text, " \t\r\n").len == 0) {
            try self.collector.diagnostics.append(diagnostics.testReasonMustBeNonEmpty(reason.span));
            return null;
        }

        if (intrinsic.family == .expect and std.mem.eql(u8, intrinsic.name, "That")) {
            return try self.lowerExpectThat(call, reason_text, reason.span);
        }

        const operand_count = expected_arity - 1;
        var operands = try self.collector.allocator.alloc(hir.ExprId, operand_count);
        var operands_transferred = false;
        defer if (!operands_transferred) self.collector.allocator.free(operands);

        for (call.args[0..operand_count], 0..) |arg, index| {
            operands[index] = (try self.lowerExpr(arg.*)) orelse return null;
        }

        const kind = (try self.resolveTestIntrinsicKind(call, intrinsic, operands)) orelse return null;
        const owned_reason = try self.collector.allocator.dupe(u8, reason_text);
        errdefer self.collector.allocator.free(owned_reason);

        const expr_id = try self.collector.module.hir.addExpr(.{ .test_intrinsic = .{
            .kind = kind,
            .operands = operands,
            .reason = owned_reason,
            .reason_span = reason.span,
        } }, call.span);
        operands_transferred = true;
        return expr_id;
    }

    fn resolveTestIntrinsicKind(self: *BodyLowerer, call: ast.Expr.CallExpr, intrinsic: ParsedTestIntrinsic, operands: []const hir.ExprId) !?hir.HirTestIntrinsicKind {
        if (std.mem.eql(u8, intrinsic.name, "True") or std.mem.eql(u8, intrinsic.name, "False")) {
            const condition_type = (try self.inferExprType(operands[0])) orelse return null;
            if (!sameType(condition_type, self.collector.module.types.boolType())) {
                try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(self.collector.module.hir.getExpr(operands[0]).span));
                return null;
            }
            if (intrinsic.family == .assert and std.mem.eql(u8, intrinsic.name, "True")) return .assert_true;
            if (intrinsic.family == .assert and std.mem.eql(u8, intrinsic.name, "False")) return .assert_false;
            if (intrinsic.family == .expect and std.mem.eql(u8, intrinsic.name, "True")) return .expect_true;
            if (intrinsic.family == .expect and std.mem.eql(u8, intrinsic.name, "False")) return .expect_false;
        }

        if (intrinsic.family == .expect and std.mem.eql(u8, intrinsic.name, "Equal")) {
            const expected_type = (try self.inferExprType(operands[0])) orelse return null;
            const actual_type = (try self.inferExprType(operands[1])) orelse return null;
            if (!sameType(expected_type, actual_type)) {
                try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(call.span));
                return null;
            }
            if (sameType(expected_type, self.collector.module.types.intType())) return .expect_equal_int;
            if (sameType(expected_type, self.collector.module.types.boolType())) return .expect_equal_bool;
            try self.collector.diagnostics.append(diagnostics.expectEqualUnsupportedType(call.span));
            return null;
        }

        try self.collector.diagnostics.append(diagnostics.unknownFunction(call.callee.span));
        return null;
    }

    fn lowerExpectThat(self: *BodyLowerer, call: ast.Expr.CallExpr, reason_text: []const u8, reason_span: source.SourceSpan) !?hir.ExprId {
        const actual = (try self.lowerExpr(call.args[0].*)) orelse return null;
        const relation = (try self.lowerTestRelation(call.args[1].*)) orelse return null;

        const kind = (try self.resolveExpectThatKind(call, actual, relation)) orelse return null;
        const operand_count: usize = if (relation.expected == null) 1 else 2;
        var operands = try self.collector.allocator.alloc(hir.ExprId, operand_count);
        var operands_transferred = false;
        defer if (!operands_transferred) self.collector.allocator.free(operands);

        if (relation.expected) |expected| {
            operands[0] = expected;
            operands[1] = actual;
        } else {
            operands[0] = actual;
        }

        const owned_reason = try self.collector.allocator.dupe(u8, reason_text);
        errdefer self.collector.allocator.free(owned_reason);

        const expr_id = try self.collector.module.hir.addExpr(.{ .test_intrinsic = .{
            .kind = kind,
            .operands = operands,
            .reason = owned_reason,
            .reason_span = reason_span,
        } }, call.span);
        operands_transferred = true;
        return expr_id;
    }

    fn lowerTestRelation(self: *BodyLowerer, expr: ast.Expr) !?ParsedTestRelation {
        const call = switch (expr) {
            .call => |call| call,
            else => {
                try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(expr.span()));
                return null;
            },
        };
        const qualifier = call.qualifier orelse {
            try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(call.span));
            return null;
        };
        if (!std.mem.eql(u8, qualifier.text, "Is")) {
            try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(call.span));
            return null;
        }
        if (std.mem.eql(u8, call.callee.text, "True")) {
            if (call.args.len != 0) {
                try self.collector.diagnostics.append(diagnostics.testIntrinsicArityMismatch(call.span));
                return null;
            }
            return .{ .kind = .is_true, .span = call.span };
        }
        if (std.mem.eql(u8, call.callee.text, "False")) {
            if (call.args.len != 0) {
                try self.collector.diagnostics.append(diagnostics.testIntrinsicArityMismatch(call.span));
                return null;
            }
            return .{ .kind = .is_false, .span = call.span };
        }
        if (std.mem.eql(u8, call.callee.text, "EqualTo")) {
            if (call.args.len != 1) {
                try self.collector.diagnostics.append(diagnostics.testIntrinsicArityMismatch(call.span));
                return null;
            }
            switch (call.args[0].*) {
                .string_literal, .struct_literal, .enum_constructor => {
                    try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(call.span));
                    return null;
                },
                else => {},
            }
            const expected = (try self.lowerExpr(call.args[0].*)) orelse return null;
            return .{ .kind = .equal, .expected = expected, .span = call.span };
        }

        try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(call.span));
        return null;
    }

    fn resolveExpectThatKind(self: *BodyLowerer, call: ast.Expr.CallExpr, actual: hir.ExprId, relation: ParsedTestRelation) !?hir.HirTestIntrinsicKind {
        const actual_type = (try self.inferExprType(actual)) orelse return null;
        switch (relation.kind) {
            .is_true => {
                if (!sameType(actual_type, self.collector.module.types.boolType())) {
                    try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(self.collector.module.hir.getExpr(actual).span));
                    return null;
                }
                return .expect_that_true;
            },
            .is_false => {
                if (!sameType(actual_type, self.collector.module.types.boolType())) {
                    try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(self.collector.module.hir.getExpr(actual).span));
                    return null;
                }
                return .expect_that_false;
            },
            .equal => {
                const expected = relation.expected orelse {
                    try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(relation.span));
                    return null;
                };
                const expected_type = (try self.inferExprType(expected)) orelse return null;
                if (!sameType(expected_type, actual_type)) {
                    try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(call.span));
                    return null;
                }
                if (sameType(expected_type, self.collector.module.types.intType())) return .expect_that_equal_int;
                if (sameType(expected_type, self.collector.module.types.boolType())) return .expect_that_equal_bool;
                try self.collector.diagnostics.append(diagnostics.testRelationUnsupported(relation.span));
                return null;
            },
        }
    }

    fn inferExprType(self: *BodyLowerer, expr_id: hir.ExprId) !?types.TypeId {
        const expr = self.collector.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => self.collector.module.types.intType(),
            .bool_literal => self.collector.module.types.boolType(),
            .local_ref => |local_id| self.collector.module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.collector.module.hir.getParam(param_id).type_id,
            .group => |inner| try self.inferExprType(inner),
            .unary => |unary| blk: {
                const operand_type = (try self.inferExprType(unary.operand)) orelse return null;
                switch (unary.op) {
                    .logical_not => {
                        if (!sameType(operand_type, self.collector.module.types.boolType())) {
                            try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                            return null;
                        }
                        break :blk self.collector.module.types.boolType();
                    },
                    .negate => {
                        if (!sameType(operand_type, self.collector.module.types.intType())) {
                            try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                            return null;
                        }
                        break :blk self.collector.module.types.intType();
                    },
                }
            },
            .binary => |binary| blk: {
                const left_type = (try self.inferExprType(binary.left)) orelse return null;
                const right_type = (try self.inferExprType(binary.right)) orelse return null;
                switch (binary.op) {
                    .add, .subtract, .multiply, .divide, .modulo => {
                        if (!sameType(left_type, self.collector.module.types.intType()) or !sameType(right_type, self.collector.module.types.intType())) {
                            try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                            return null;
                        }
                        break :blk self.collector.module.types.intType();
                    },
                    .less, .less_equal, .greater, .greater_equal => {
                        if (!sameType(left_type, self.collector.module.types.intType()) or !sameType(right_type, self.collector.module.types.intType())) {
                            try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                            return null;
                        }
                        break :blk self.collector.module.types.boolType();
                    },
                    .equal_equal, .bang_equal => {
                        if (!sameType(left_type, right_type)) {
                            try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                            return null;
                        }
                        break :blk self.collector.module.types.boolType();
                    },
                    .logical_and, .logical_or => {
                        if (!sameType(left_type, self.collector.module.types.boolType()) or !sameType(right_type, self.collector.module.types.boolType())) {
                            try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                            return null;
                        }
                        break :blk self.collector.module.types.boolType();
                    },
                }
            },
            .call => |call| self.collector.module.hir.getFunction(call.function).return_type,
            .arena_alloc => |arena_alloc| arena_alloc.result_type,
            .field_access => |field_access| blk: {
                const receiver_type = (try self.inferExprType(field_access.receiver)) orelse return null;
                const receiver_kind = self.collector.module.types.kind(receiver_type);
                if (receiver_kind != .struct_type) {
                    try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                    return null;
                }
                const field_id = self.findField(receiver_kind.struct_type, field_access.field_name) orelse {
                    try self.collector.diagnostics.append(diagnostics.Diagnostic.init(.UnknownFieldAccess, .@"error", "unknown field on struct value", field_access.field_span));
                    return null;
                };
                break :blk self.collector.module.hir.getField(field_id).type_id;
            },
            .struct_literal => |literal| literal.type_id,
            .enum_constructor => |constructor| blk: {
                for (self.collector.module.types.types.items, 0..) |kind, index| {
                    if (kind == .enum_type and kind.enum_type.index == constructor.enum_id.index) break :blk types.TypeId{ .index = @intCast(index) };
                }
                try self.collector.diagnostics.append(diagnostics.unknownEnumConstructor(expr.span));
                return null;
            },
            .address_of => |operand| blk: {
                const operand_type = (try self.inferExprType(operand)) orelse return null;
                break :blk try self.collector.module.types.addPointerType(operand_type);
            },
            .deref => |operand| blk: {
                const operand_type = (try self.inferExprType(operand)) orelse return null;
                const pointee = switch (self.collector.module.types.kind(operand_type)) {
                    .pointer => |pointer| pointer.pointee,
                    else => {
                        try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                        return null;
                    },
                };
                break :blk pointee;
            },
            .move_expr, .manual_init_assume, .try_expr, .compile_time, .target_metadata, .decide, .concept_requirement_call, .test_intrinsic => {
                try self.collector.diagnostics.append(diagnostics.testIntrinsicTypeMismatch(expr.span));
                return null;
            },
        };
    }

    fn stringLiteralContents(text: []const u8) []const u8 {
        if (text.len >= 2 and text[0] == '"' and text[text.len - 1] == '"') return text[1 .. text.len - 1];
        return text;
    }

    fn isUnboundTargetRoot(self: *BodyLowerer, expr: ast.Expr) bool {
        return switch (expr) {
            .identifier => |ident| blk: {
                if (!std.mem.eql(u8, ident.name.text, "target")) break :blk false;
                const symbol = self.collector.module.interner.intern(ident.name.text) catch break :blk false;
                break :blk self.lookup(symbol) == null;
            },
            else => false,
        };
    }

    fn isArenaAllocCall(self: *BodyLowerer, call: ast.Expr.CallExpr) bool {
        _ = self;
        const qualifier = call.qualifier orelse return false;
        return std.mem.eql(u8, qualifier.text, "Arena") and std.mem.eql(u8, call.callee.text, "alloc");
    }

    const ArenaStorageOpKind = enum { reset, destroy };

    fn arenaStorageOpKind(self: *BodyLowerer, call: ast.Expr.CallExpr) ?ArenaStorageOpKind {
        _ = self;
        const qualifier = call.qualifier orelse return null;
        if (!std.mem.eql(u8, qualifier.text, "Arena")) return null;
        if (std.mem.eql(u8, call.callee.text, "reset")) return .reset;
        if (std.mem.eql(u8, call.callee.text, "destroy")) return .destroy;
        return null;
    }

    fn lowerArenaStorageOpStmt(self: *BodyLowerer, expr: ast.Expr) !?hir.StmtId {
        const call = switch (expr) {
            .call => |call| call,
            else => return null,
        };
        const kind = self.arenaStorageOpKind(call) orelse return null;
        if (call.type_args.len != 0) {
            try self.collector.diagnostics.append(diagnostics.Diagnostic.init(
                .ArenaResetDestroyTypeArgsUnsupported,
                .@"error",
                "Arena.reset and Arena.destroy do not accept type arguments",
                call.span,
            ));
            return null;
        }
        if (call.args.len != 1) {
            try self.collector.diagnostics.append(diagnostics.Diagnostic.init(
                .ArenaResetDestroyArityMismatch,
                .@"error",
                "Arena.reset and Arena.destroy require exactly one arena argument",
                call.span,
            ));
            return null;
        }

        const arena_expr = (try self.lowerExpr(call.args[0].*)) orelse return null;
        const arena_type = try self.collector.module.types.addPointerType(self.collector.module.types.arenaType());
        return try self.collector.module.hir.addStmt(switch (kind) {
            .reset => .{ .arena_reset = .{ .arena_expr = arena_expr, .arena_type = arena_type } },
            .destroy => .{ .arena_destroy = .{ .arena_expr = arena_expr, .arena_type = arena_type } },
        }, call.span);
    }

    fn lowerArenaAlloc(self: *BodyLowerer, call: ast.Expr.CallExpr) !?hir.ExprId {
        if (call.type_args.len != 1) {
            try self.collector.diagnostics.append(diagnostics.Diagnostic.init(
                .ArenaAllocArityMismatch,
                .@"error",
                "Arena.alloc requires exactly one type argument",
                call.span,
            ));
            return null;
        }
        if (call.args.len != 1) {
            try self.collector.diagnostics.append(diagnostics.Diagnostic.init(
                .ArenaAllocArityMismatch,
                .@"error",
                "Arena.alloc requires exactly one arena argument",
                call.span,
            ));
            return null;
        }

        const allocated_type = (try self.resolveTypeNameInCurrentFunction(call.type_args[0])) orelse return null;
        const arena_expr = (try self.lowerExpr(call.args[0].*)) orelse return null;
        const result_type = try self.collector.module.types.addPointerType(allocated_type);
        return try self.collector.module.hir.addExpr(.{ .arena_alloc = .{
            .arena_expr = arena_expr,
            .allocated_type = allocated_type,
            .result_type = result_type,
        } }, call.span);
    }

    fn resolveTypeNameInCurrentFunction(self: *BodyLowerer, type_name: ast.TypeName) !?types.TypeId {
        var scope = TypeParamScope.init(self.collector.allocator);
        defer scope.deinit();
        if (self.function_id) |function_id| {
            if (self.collector.module.hir.genericFunctionFor(function_id)) |generic_id| {
                const generic = self.collector.module.hir.getGenericFunction(generic_id);
                for (generic.type_params) |type_param| {
                    try scope.put(type_param.name, type_param.type_id);
                }
            }
        }
        return self.collector.resolveTypeNameScoped(type_name, &scope);
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
            const local_id = try self.collector.module.hir.addLocal(self.function_id.?, symbol, payload.type_id, binding.name.span);
            try self.bindings.append(self.collector.allocator, .{ .name = symbol, .binding = .{ .local = local_id }, .depth = self.depth });
            try bindings.append(self.collector.allocator, .{ .name = symbol, .local = local_id, .payload_field = payload_id, .type_id = payload.type_id, .span = binding.name.span });
        }
        return .{ .enum_variant = .{ .enum_id = enum_id, .variant_id = variant_id, .bindings = try bindings.toOwnedSlice(self.collector.allocator) } };
    }

    const RequirementCallResolution = struct {
        concept_id: hir.ConceptId,
        requirement_index: u32,
    };

    fn resolveConceptRequirementCall(self: *BodyLowerer, name: interner.SymbolId, arity: u32, span: source.SourceSpan) !?RequirementCallResolution {
        const generic_id = self.collector.module.hir.genericFunctionFor(self.function_id orelse return null) orelse return null;
        const generic = self.collector.module.hir.getGenericFunction(generic_id);
        var found: ?RequirementCallResolution = null;
        var found_name = false;
        for (generic.type_params) |type_param| {
            const constraint = type_param.constraint orelse continue;
            const concept_id = constraint.concept_id orelse continue;
            const concept = self.collector.module.hir.getConcept(concept_id);
            for (concept.requirements, 0..) |requirement, index| {
                if (requirement.name.index != name.index) continue;
                found_name = true;
                if (requirement.params.len == arity) {
                    if (found != null) {
                        try self.collector.diagnostics.append(diagnostics.ambiguousConceptRequirementCall(span));
                        return null;
                    }
                    found = .{ .concept_id = concept_id, .requirement_index = @intCast(index) };
                }
            }
        }
        if (found == null and found_name) {
            try self.collector.diagnostics.append(diagnostics.invalidConceptRequirementCall(span));
        }
        return found;
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

fn collectTestItems(items: []const ast.Item, diagnostic_bag: *DiagnosticBag) !SemanticModule {
    return collectTopLevelDeclarationsWithOptions(std.testing.allocator, unitFromItems(items), diagnostic_bag, .{ .source_file_kind = .@"test" });
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

fn functionItemWithSignature(name: []const u8, return_type: []const u8, params: []ast.ParamDecl, attributes: []ast.Attribute, start: usize) ast.Item {
    return .{ .function_decl = .{
        .is_export = false,
        .signature = .{
            .return_type = typeName(return_type, start),
            .name = .{
                .base = nameSegment(name, start + return_type.len + 1),
                .span = .{ .start = start + return_type.len + 1, .length = name.len },
            },
            .params = params,
            .span = .{ .start = start, .length = return_type.len + name.len + 3 },
        },
        .attributes = attributes,
        .body = null,
        .span = .{ .start = start, .length = return_type.len + name.len + 3 },
    } };
}

fn paramDecl(type_name: []const u8, name: []const u8, start: usize) ast.ParamDecl {
    return .{
        .type_name = typeName(type_name, start),
        .name = nameSegment(name, start + type_name.len + 1),
        .span = .{ .start = start, .length = type_name.len + name.len + 1 },
    };
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
        .concept_name = typeName("Drop", start + 5),
        .target_types = &.{},
        .functions = &.{},
        .span = .{ .start = start, .length = 12 },
    } };
}

const test_int_type_parts = [_]ast.NameSegment{.{ .text = "int", .span = .{ .start = 0, .length = 3 } }};
const test_drop_type_parts = [_]ast.NameSegment{.{ .text = "Drop", .span = .{ .start = 0, .length = 4 } }};
const test_bool_type_parts = [_]ast.NameSegment{.{ .text = "bool", .span = .{ .start = 0, .length = 4 } }};
const test_void_type_parts = [_]ast.NameSegment{.{ .text = "void", .span = .{ .start = 0, .length = 4 } }};
const test_arena_type_parts = [_]ast.NameSegment{.{ .text = "Arena", .span = .{ .start = 0, .length = 5 } }};
const test_allocator_type_parts = [_]ast.NameSegment{.{ .text = "Allocator", .span = .{ .start = 0, .length = 9 } }};
const test_alloc_error_type_parts = [_]ast.NameSegment{.{ .text = "AllocError", .span = .{ .start = 0, .length = 10 } }};

fn typeName(name: []const u8, start: usize) ast.TypeName {
    const is_pointer = std.mem.endsWith(u8, name, "*");
    const base_name = if (is_pointer) name[0 .. name.len - 1] else name;
    const parts: []ast.NameSegment = if (std.mem.eql(u8, base_name, "Drop"))
        @constCast(test_drop_type_parts[0..])
    else if (std.mem.eql(u8, base_name, "bool"))
        @constCast(test_bool_type_parts[0..])
    else if (std.mem.eql(u8, base_name, "void"))
        @constCast(test_void_type_parts[0..])
    else if (std.mem.eql(u8, base_name, "Arena"))
        @constCast(test_arena_type_parts[0..])
    else if (std.mem.eql(u8, base_name, "Allocator"))
        @constCast(test_allocator_type_parts[0..])
    else if (std.mem.eql(u8, base_name, "AllocError"))
        @constCast(test_alloc_error_type_parts[0..])
    else
        @constCast(test_int_type_parts[0..]);
    return .{
        .name = .{
            .parts = parts,
            .span = .{ .start = start, .length = base_name.len },
        },
        .is_pointer = is_pointer,
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
    var concept_params = [_]ast.NameSegment{nameSegment("T", 0)};
    const equatable_concept = ast.Item{ .concept_decl = .{
        .name = nameSegment("Equatable", 0),
        .generic_params = concept_params[0..],
        .signatures = &.{},
        .is_marker = true,
        .span = .{ .start = 0, .length = 9 },
    } };

    var module = try collectItems(&.{ equatable_concept, genericFunctionItem(template_params[0..], "first", k_type, params[0..], statements[0..]) }, &diagnostics_bag);
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
    try std.testing.expectEqual(shape, module.resultShapeForType(.{ .index = 6 }).?);
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

    try std.testing.expectEqual(@as(usize, 8), module.types.count());
    try std.testing.expectEqual(types.TypeKind{ .struct_type = .{ .index = 0 } }, module.types.kind(.{ .index = 6 }));
    try std.testing.expectEqual(types.TypeKind{ .enum_type = .{ .index = 0 } }, module.types.kind(.{ .index = 7 }));
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
    try std.testing.expectEqual(types.TypeKind{ .pointer = .{ .pointee = .{ .index = 6 } } }, module.types.kind(pointer_type));
}

test "semantic collection resolves compiler-known allocation surface types" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var params = [_]ast.ParamDecl{
        paramDecl("Arena*", "arena", 20),
        paramDecl("Allocator*", "allocator", 40),
        paramDecl("AllocError", "err", 70),
    };
    var module = try collectItems(&.{functionItemWithSignature("usesAllocationSurface", "AllocError", params[0..], &.{}, 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const function = module.hir.getFunction(.{ .index = 0 });
    try std.testing.expectEqual(module.types.allocErrorType(), function.return_type);
    try std.testing.expectEqual(types.TypeKind{ .pointer = .{ .pointee = module.types.arenaType() } }, module.types.kind(module.hir.getParam(function.params[0]).type_id));
    try std.testing.expectEqual(types.TypeKind{ .pointer = .{ .pointee = module.types.allocatorType() } }, module.types.kind(module.hir.getParam(function.params[1]).type_id));
    try std.testing.expectEqual(module.types.allocErrorType(), module.hir.getParam(function.params[2]).type_id);
}

test "semantic collection reserves allocation surface type names" {
    try expectOneSemanticDiagnostic(&.{structItem("Arena", 0)}, .DuplicateTopLevelName);
    try expectOneSemanticDiagnostic(&.{structItem("Allocator", 0)}, .DuplicateTopLevelName);
    try expectOneSemanticDiagnostic(&.{enumItem("AllocError", 0)}, .DuplicateTopLevelName);
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

test "semantic collection stores concept and ignores interface executable items" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectItems(&.{ conceptItem("Hashable", 0), interfaceItem("Renderer", 40), structItem("Texture", 100) }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 1), module.hir.items.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.structs.items.len);
    try std.testing.expectEqual(@as(usize, 1), module.hir.concepts.items.len);
    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
}

test "semantic collection lowers concept requirements into HIR debug" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var t_parts = [_]ast.NameSegment{nameSegment("T", 0)};
    const t_type = typeNameFromSegment(&t_parts);
    var concept_params = [_]ast.GenericParam{nameSegment("T", 0)};
    var requirement_params = [_]ast.ParamDecl{
        .{ .type_name = t_type, .name = nameSegment("left", 0), .span = .{ .start = 0, .length = 1 } },
        .{ .type_name = t_type, .name = nameSegment("right", 0), .span = .{ .start = 0, .length = 1 } },
    };
    var signatures = [_]ast.SignatureDecl{.{
        .return_type = typeName("bool", 0),
        .name = .{ .base = nameSegment("equals", 0), .span = .{ .start = 0, .length = 6 } },
        .params = requirement_params[0..],
        .span = .{ .start = 0, .length = 1 },
    }};
    var module = try collectItems(&.{.{ .concept_decl = .{
        .name = nameSegment("Equatable", 0),
        .generic_params = concept_params[0..],
        .signatures = signatures[0..],
        .span = .{ .start = 0, .length = 1 },
    } }}, &diagnostics_bag);
    defer module.deinit();

    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Concept Equatable
        \\    TypeParams
        \\      #0 T: TypeId(6)
        \\    Requirements
        \\      equals -> TypeId(2)
        \\        left: TypeId(6)
        \\        right: TypeId(6)
        \\
    , snapshot);
}

test "semantic collection marks compiler-known marker concepts in HIR debug" {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var t_parts = [_]ast.GenericParam{nameSegment("T", 0)};
    var module = try collectItems(&.{.{ .concept_decl = .{
        .name = nameSegment("Copy", 0),
        .generic_params = t_parts[0..],
        .signatures = &.{},
        .is_marker = true,
        .span = .{ .start = 0, .length = 1 },
    } }}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    try std.testing.expectEqual(hir.MarkerKind.copy, module.hir.getConcept(.{ .index = 0 }).known_marker_kind);

    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Marker Concept Copy known_marker=Copy
        \\    TypeParams
        \\      #0 T: TypeId(6)
        \\
    , snapshot);
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

test "semantic collection preserves compile-time function metadata" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const statements = try allocator.alloc(ast.Stmt, 1);
    statements[0] = try returnStmt(allocator, intExpr("42"));
    var item = functionWithBody("answer", &.{}, statements);
    item.function_decl.is_compile_time = true;

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectItems(&.{item}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expect(module.hir.getFunction(.{ .index = 0 }).is_compile_time);
    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "CompileTime Function answer") != null);
}

test "semantic collection preserves allocation effect function metadata" {
    var noalloc_item = functionItem("add", 0);
    noalloc_item.function_decl.allocation_effect = .noalloc;
    var alloc_item = functionItem("build", 20);
    alloc_item.function_decl.allocation_effect = .alloc;
    const unspecified_item = functionItem("plain", 40);

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectItems(&.{ noalloc_item, alloc_item, unspecified_item }, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    try std.testing.expectEqual(hir.AllocationEffect.noalloc, module.hir.getFunction(.{ .index = 0 }).allocation_effect);
    try std.testing.expectEqual(hir.AllocationEffect.alloc, module.hir.getFunction(.{ .index = 1 }).allocation_effect);
    try std.testing.expectEqual(hir.AllocationEffect.unspecified, module.hir.getFunction(.{ .index = 2 }).allocation_effect);
    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function add effect=NoAlloc") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function build effect=Alloc") != null);
}

test "semantic collection preserves function attributes in HIR" {
    var parts = [_]ast.NameSegment{nameSegment("Fact", 0)};
    const attr = attributeNoArgs("Fact", parts[0..]);
    var attrs = [_]ast.Attribute{attr};
    const item = functionItemWithSignature("ParsesIdentifier", "void", &.{}, attrs[0..], 0);

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectTestItems(&.{item}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const function = module.hir.getFunction(.{ .index = 0 });
    try std.testing.expectEqual(@as(usize, 1), function.attributes.len);
    try std.testing.expectEqualStrings("Fact", module.interner.text(function.attributes[0].name));

    const snapshot = try module.hir.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Attribute Fact\n  Function ParsesIdentifier") != null);
}

test "semantic collection preserves valid InlineData literal args" {
    var args = [_]ast.AttributeArg{
        .{ .int_literal = .{ .text = "1", .span = .{ .start = 0, .length = 1 } } },
        .{ .bool_literal = .{ .value = true, .span = .{ .start = 3, .length = 4 } } },
    };
    var theory_parts = [_]ast.NameSegment{nameSegment("Theory", 0)};
    var inline_parts = [_]ast.NameSegment{nameSegment("InlineData", 8)};
    var attrs = [_]ast.Attribute{
        attributeNoArgs("Theory", theory_parts[0..]),
        attributeWithArgs("InlineData", inline_parts[0..], args[0..]),
    };
    var params = [_]ast.ParamDecl{
        paramDecl("int", "value", 20),
        paramDecl("bool", "flag", 30),
    };
    const item = functionItemWithSignature("main", "void", params[0..], attrs[0..], 0);

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectTestItems(&.{item}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const function = module.hir.getFunction(.{ .index = 0 });
    try std.testing.expectEqual(@as(usize, 2), function.attributes[1].args.len);
    try std.testing.expectEqualStrings("1", function.attributes[1].args[0].int_literal);
    try std.testing.expect(function.attributes[1].args[1].bool_literal);
}

test "semantic collection rejects test attributes in normal source files" {
    var parts = [_]ast.NameSegment{nameSegment("Fact", 0)};
    const attr = attributeNoArgs("Fact", parts[0..]);
    var item = functionItem("ParsesIdentifier", 0);
    var attrs = [_]ast.Attribute{attr};
    item.function_decl.attributes = attrs[0..];

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    try std.testing.expectError(error.InvalidSemanticModule, collectItems(&.{item}, &diagnostics_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostics_bag.count());
    try std.testing.expectEqual(DiagnosticCode.TestAttributeOutsideTestFile, diagnostics_bag.diagnostics.items[0].code);
    try std.testing.expectEqualStrings("test attributes are only valid in .con_test files", diagnostics_bag.diagnostics.items[0].message);
}

fn allocExpr(allocator: std.mem.Allocator, expr: ast.Expr) !*ast.Expr {
    const ptr = try allocator.create(ast.Expr);
    ptr.* = expr;
    return ptr;
}

fn intExpr(text: []const u8) ast.Expr {
    return .{ .int_literal = .{ .text = text, .span = .{ .start = 0, .length = text.len } } };
}

fn boolExpr(value: bool) ast.Expr {
    return .{ .bool_literal = .{ .value = value, .span = .{ .start = 0, .length = if (value) 4 else 5 } } };
}

fn stringExpr(text: []const u8) ast.Expr {
    return .{ .string_literal = .{ .text = text, .span = .{ .start = 0, .length = text.len } } };
}

fn identExpr(name: []const u8) ast.Expr {
    return .{ .identifier = .{ .name = nameSegment(name, 0), .span = .{ .start = 0, .length = name.len } } };
}

fn callExpr(allocator: std.mem.Allocator, name: []const u8, args: []const ast.Expr) !ast.Expr {
    const arg_ptrs = try allocator.alloc(*ast.Expr, args.len);
    for (args, 0..) |arg, index| arg_ptrs[index] = try allocExpr(allocator, arg);
    return .{ .call = .{ .callee = nameSegment(name, 0), .args = arg_ptrs, .span = .{ .start = 0, .length = name.len } } };
}

fn qualifiedCallExpr(allocator: std.mem.Allocator, qualifier: []const u8, name: []const u8, args: []const ast.Expr) !ast.Expr {
    const arg_ptrs = try allocator.alloc(*ast.Expr, args.len);
    for (args, 0..) |arg, index| arg_ptrs[index] = try allocExpr(allocator, arg);
    return .{ .call = .{
        .qualifier = nameSegment(qualifier, 0),
        .callee = nameSegment(name, qualifier.len + 1),
        .args = arg_ptrs,
        .span = .{ .start = 0, .length = qualifier.len + name.len + 1 },
    } };
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

fn localStmtTyped(allocator: std.mem.Allocator, type_name: []const u8, name: []const u8, expr: ast.Expr) !ast.Stmt {
    return .{ .local_decl = .{ .type_name = typeName(type_name, 0), .name = nameSegment(name, 0), .initializer = try allocExpr(allocator, expr), .span = .{ .start = 0, .length = 1 } } };
}

fn exprStmt(allocator: std.mem.Allocator, expr: ast.Expr) !ast.Stmt {
    return .{ .expr_stmt = .{ .value = try allocExpr(allocator, expr), .span = expr.span() } };
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

fn attributeNoArgs(name: []const u8, parts: []ast.NameSegment) ast.Attribute {
    return .{
        .name = .{ .parts = parts, .span = .{ .start = 0, .length = name.len } },
        .arguments = null,
        .span = .{ .start = 0, .length = name.len + 2 },
    };
}

fn attributeWithArgs(name: []const u8, parts: []ast.NameSegment, args: []ast.AttributeArg) ast.Attribute {
    return .{
        .name = .{ .parts = parts, .span = .{ .start = 0, .length = name.len } },
        .arguments = .{ .args = args, .span = .{ .start = 0, .length = 0 } },
        .span = .{ .start = 0, .length = name.len + 2 },
    };
}

fn expectOneSemanticDiagnostic(items: []const ast.Item, code: DiagnosticCode) !void {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, collectItems(items, &diagnostics_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostics_bag.count());
    try std.testing.expectEqual(code, diagnostics_bag.diagnostics.items[0].code);
}

fn expectOneTestSemanticDiagnostic(items: []const ast.Item, code: DiagnosticCode) !void {
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, collectTestItems(items, &diagnostics_bag));
    try std.testing.expectEqual(@as(usize, 1), diagnostics_bag.count());
    try std.testing.expectEqual(code, diagnostics_bag.diagnostics.items[0].code);
}

test "test attribute semantics accept zero-param Fact" {
    var fact_parts = [_]ast.NameSegment{nameSegment("Fact", 0)};
    var attrs = [_]ast.Attribute{attributeNoArgs("Fact", fact_parts[0..])};
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectTestItems(&.{functionItemWithSignature("ParsesIdentifier", "void", &.{}, attrs[0..], 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
}

test "test attribute semantics reject Fact with parameter" {
    var fact_parts = [_]ast.NameSegment{nameSegment("Fact", 0)};
    var attrs = [_]ast.Attribute{attributeNoArgs("Fact", fact_parts[0..])};
    var params = [_]ast.ParamDecl{paramDecl("int", "value", 20)};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("HasParam", "void", params[0..], attrs[0..], 0)}, .FactRequiresZeroArgFunction);
}

test "test attribute semantics reject Fact returning non-void" {
    var fact_parts = [_]ast.NameSegment{nameSegment("Fact", 0)};
    var attrs = [_]ast.Attribute{attributeNoArgs("Fact", fact_parts[0..])};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("ReturnsInt", "int", &.{}, attrs[0..], 0)}, .TestFunctionReturnTypeInvalid);
}

test "test attribute semantics reject Theory without InlineData" {
    var theory_parts = [_]ast.NameSegment{nameSegment("Theory", 0)};
    var attrs = [_]ast.Attribute{attributeNoArgs("Theory", theory_parts[0..])};
    var params = [_]ast.ParamDecl{paramDecl("int", "value", 20)};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("NoData", "void", params[0..], attrs[0..], 0)}, .TheoryRequiresInlineData);
}

test "test attribute semantics reject InlineData without Theory" {
    var inline_parts = [_]ast.NameSegment{nameSegment("InlineData", 0)};
    var args = [_]ast.AttributeArg{.{ .int_literal = .{ .text = "1", .span = .{ .start = 0, .length = 1 } } }};
    var attrs = [_]ast.Attribute{attributeWithArgs("InlineData", inline_parts[0..], args[0..])};
    var params = [_]ast.ParamDecl{paramDecl("int", "value", 20)};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("NotTheory", "void", params[0..], attrs[0..], 0)}, .InlineDataRequiresTheory);
}

test "test attribute semantics reject Fact Theory conflict" {
    var fact_parts = [_]ast.NameSegment{nameSegment("Fact", 0)};
    var theory_parts = [_]ast.NameSegment{nameSegment("Theory", 8)};
    var inline_parts = [_]ast.NameSegment{nameSegment("InlineData", 16)};
    var args = [_]ast.AttributeArg{.{ .int_literal = .{ .text = "1", .span = .{ .start = 16, .length = 1 } } }};
    var attrs = [_]ast.Attribute{
        attributeNoArgs("Fact", fact_parts[0..]),
        attributeNoArgs("Theory", theory_parts[0..]),
        attributeWithArgs("InlineData", inline_parts[0..], args[0..]),
    };
    var params = [_]ast.ParamDecl{paramDecl("int", "value", 40)};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("Bad", "void", params[0..], attrs[0..], 0)}, .ConflictingTestAttributes);
}

test "test attribute semantics reject duplicate Fact and Theory" {
    var fact_parts_a = [_]ast.NameSegment{nameSegment("Fact", 0)};
    var fact_parts_b = [_]ast.NameSegment{nameSegment("Fact", 8)};
    var fact_attrs = [_]ast.Attribute{
        attributeNoArgs("Fact", fact_parts_a[0..]),
        attributeNoArgs("Fact", fact_parts_b[0..]),
    };
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("DuplicateFact", "void", &.{}, fact_attrs[0..], 0)}, .DuplicateTestAttribute);

    var theory_parts_a = [_]ast.NameSegment{nameSegment("Theory", 0)};
    var theory_parts_b = [_]ast.NameSegment{nameSegment("Theory", 8)};
    var inline_parts = [_]ast.NameSegment{nameSegment("InlineData", 16)};
    var args = [_]ast.AttributeArg{.{ .int_literal = .{ .text = "1", .span = .{ .start = 16, .length = 1 } } }};
    var theory_attrs = [_]ast.Attribute{
        attributeNoArgs("Theory", theory_parts_a[0..]),
        attributeNoArgs("Theory", theory_parts_b[0..]),
        attributeWithArgs("InlineData", inline_parts[0..], args[0..]),
    };
    var params = [_]ast.ParamDecl{paramDecl("int", "value", 40)};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("DuplicateTheory", "void", params[0..], theory_attrs[0..], 0)}, .DuplicateTestAttribute);
}

test "test attribute semantics validate InlineData arity and type" {
    var theory_parts = [_]ast.NameSegment{nameSegment("Theory", 0)};
    var inline_parts = [_]ast.NameSegment{nameSegment("InlineData", 8)};
    var one_arg = [_]ast.AttributeArg{.{ .int_literal = .{ .text = "1", .span = .{ .start = 8, .length = 1 } } }};
    var arity_attrs = [_]ast.Attribute{
        attributeNoArgs("Theory", theory_parts[0..]),
        attributeWithArgs("InlineData", inline_parts[0..], one_arg[0..]),
    };
    var two_params = [_]ast.ParamDecl{ paramDecl("int", "a", 20), paramDecl("int", "b", 26) };
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("Adds", "void", two_params[0..], arity_attrs[0..], 0)}, .InlineDataArityMismatch);

    var mismatch_arg = [_]ast.AttributeArg{.{ .bool_literal = .{ .value = true, .span = .{ .start = 8, .length = 4 } } }};
    var mismatch_attrs = [_]ast.Attribute{
        attributeNoArgs("Theory", theory_parts[0..]),
        attributeWithArgs("InlineData", inline_parts[0..], mismatch_arg[0..]),
    };
    var int_param = [_]ast.ParamDecl{paramDecl("int", "value", 20)};
    try expectOneTestSemanticDiagnostic(&.{functionItemWithSignature("NeedsInt", "void", int_param[0..], mismatch_attrs[0..], 0)}, .InlineDataTypeMismatch);
}

test "test attribute semantics accept multiple int bool InlineData rows" {
    var theory_parts = [_]ast.NameSegment{nameSegment("Theory", 0)};
    var inline_parts_a = [_]ast.NameSegment{nameSegment("InlineData", 8)};
    var inline_parts_b = [_]ast.NameSegment{nameSegment("InlineData", 24)};
    var args_a = [_]ast.AttributeArg{
        .{ .int_literal = .{ .text = "1", .span = .{ .start = 8, .length = 1 } } },
        .{ .bool_literal = .{ .value = true, .span = .{ .start = 11, .length = 4 } } },
    };
    var args_b = [_]ast.AttributeArg{
        .{ .int_literal = .{ .text = "2", .span = .{ .start = 24, .length = 1 } } },
        .{ .bool_literal = .{ .value = false, .span = .{ .start = 27, .length = 5 } } },
    };
    var attrs = [_]ast.Attribute{
        attributeNoArgs("Theory", theory_parts[0..]),
        attributeWithArgs("InlineData", inline_parts_a[0..], args_a[0..]),
        attributeWithArgs("InlineData", inline_parts_b[0..], args_b[0..]),
    };
    var params = [_]ast.ParamDecl{ paramDecl("int", "value", 40), paramDecl("bool", "flag", 50) };
    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();

    var module = try collectTestItems(&.{functionItemWithSignature("Works", "void", params[0..], attrs[0..], 0)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
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

test "test intrinsic semantics recognize valid Assert and Expect spellings" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    const statements = try allocator.alloc(ast.Stmt, 7);
    statements[0] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Assert", "True", &.{ boolExpr(true), stringExpr("\"setup should be valid\"") }));
    statements[1] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Assert", "False", &.{ boolExpr(false), stringExpr("\"setup should reject false path\"") }));
    statements[2] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "True", &.{ boolExpr(true), stringExpr("\"condition should hold\"") }));
    statements[3] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "False", &.{ boolExpr(false), stringExpr("\"condition should not hold\"") }));
    statements[4] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "Equal", &.{ intExpr("1"), intExpr("1"), stringExpr("\"integers should match\"") }));
    statements[5] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "Equal", &.{ boolExpr(true), boolExpr(true), stringExpr("\"booleans should match\"") }));
    statements[6] = try returnStmt(allocator, intExpr("0"));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectTestItems(&.{functionWithBody("main", &.{}, statements)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const block = module.hir.getStmt(module.hir.getFunction(.{ .index = 0 }).body.?).kind.block;
    const expected_kinds = [_]hir.HirTestIntrinsicKind{
        .assert_true,
        .assert_false,
        .expect_true,
        .expect_false,
        .expect_equal_int,
        .expect_equal_bool,
    };
    for (expected_kinds, 0..) |expected_kind, index| {
        const expr_id = module.hir.getStmt(block[index]).kind.expr_stmt;
        const test_intrinsic = module.hir.getExpr(expr_id).kind.test_intrinsic;
        try std.testing.expectEqual(expected_kind, test_intrinsic.kind);
        try std.testing.expect(test_intrinsic.reason.len != 0);
    }
}

test "test intrinsic metadata preserves reason and Assert versus Expect kind" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    const statements = try allocator.alloc(ast.Stmt, 3);
    statements[0] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Assert", "True", &.{ boolExpr(true), stringExpr("\"assert reason\"") }));
    statements[1] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "True", &.{ boolExpr(true), stringExpr("\"expect reason\"") }));
    statements[2] = try returnStmt(allocator, intExpr("0"));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectTestItems(&.{functionWithBody("main", &.{}, statements)}, &diagnostics_bag);
    defer module.deinit();

    const block = module.hir.getStmt(module.hir.getFunction(.{ .index = 0 }).body.?).kind.block;
    const assert_intrinsic = module.hir.getExpr(module.hir.getStmt(block[0]).kind.expr_stmt).kind.test_intrinsic;
    const expect_intrinsic = module.hir.getExpr(module.hir.getStmt(block[1]).kind.expr_stmt).kind.test_intrinsic;

    try std.testing.expectEqual(hir.HirTestIntrinsicKind.assert_true, assert_intrinsic.kind);
    try std.testing.expectEqual(hir.HirTestIntrinsicKind.expect_true, expect_intrinsic.kind);
    try std.testing.expectEqualStrings("assert reason", assert_intrinsic.reason);
    try std.testing.expectEqualStrings("expect reason", expect_intrinsic.reason);
}

test "Expect.That semantics recognize built-in relation spellings" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    const statements = try allocator.alloc(ast.Stmt, 5);
    statements[0] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(true), try qualifiedCallExpr(allocator, "Is", "True", &.{}), stringExpr("\"true relation should check a bool actual\"") }));
    statements[1] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(false), try qualifiedCallExpr(allocator, "Is", "False", &.{}), stringExpr("\"false relation should check a bool actual\"") }));
    statements[2] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("4"), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{intExpr("4")}), stringExpr("\"integer relation should preserve expected value\"") }));
    statements[3] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(true), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{boolExpr(true)}), stringExpr("\"boolean relation should preserve expected value\"") }));
    statements[4] = try returnStmt(allocator, intExpr("0"));

    var diagnostics_bag = DiagnosticBag.init(std.testing.allocator);
    defer diagnostics_bag.deinit();
    var module = try collectTestItems(&.{functionWithBody("main", &.{}, statements)}, &diagnostics_bag);
    defer module.deinit();

    try std.testing.expectEqual(@as(usize, 0), diagnostics_bag.count());
    const block = module.hir.getStmt(module.hir.getFunction(.{ .index = 0 }).body.?).kind.block;
    const expected_kinds = [_]hir.HirTestIntrinsicKind{
        .expect_that_true,
        .expect_that_false,
        .expect_that_equal_int,
        .expect_that_equal_bool,
    };
    for (expected_kinds, 0..) |expected_kind, index| {
        const expr_id = module.hir.getStmt(block[index]).kind.expr_stmt;
        const test_intrinsic = module.hir.getExpr(expr_id).kind.test_intrinsic;
        try std.testing.expectEqual(expected_kind, test_intrinsic.kind);
        try std.testing.expect(test_intrinsic.reason.len != 0);
    }
}

fn expectOneTestIntrinsicDiagnostic(expr: ast.Expr, code: DiagnosticCode) !void {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();
    const statements = try allocator.alloc(ast.Stmt, 2);
    statements[0] = try exprStmt(allocator, expr);
    statements[1] = try returnStmt(allocator, intExpr("0"));
    try expectOneTestSemanticDiagnostic(&.{functionWithBody("main", &.{}, statements)}, code);
}

test "test intrinsic semantics reject missing empty and whitespace reasons" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "True", &.{boolExpr(true)}), .TestExpectationRequiresReason);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Assert", "True", &.{boolExpr(true)}), .TestExpectationRequiresReason);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "Equal", &.{ intExpr("1"), intExpr("1") }), .TestExpectationRequiresReason);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("1"), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{intExpr("1")}) }), .TestExpectationRequiresReason);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "True", &.{ boolExpr(true), stringExpr("\"\"") }), .TestReasonMustBeNonEmpty);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "True", &.{ boolExpr(true), stringExpr("\"   \"") }), .TestReasonMustBeNonEmpty);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("1"), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{intExpr("1")}), stringExpr("\"\"") }), .TestReasonMustBeNonEmpty);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("1"), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{intExpr("1")}), stringExpr("\"   \"") }), .TestReasonMustBeNonEmpty);
}

test "test intrinsic semantics validate primitive operand types" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "True", &.{ intExpr("1"), stringExpr("\"needs bool\"") }), .TestIntrinsicTypeMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Assert", "False", &.{ intExpr("1"), stringExpr("\"needs bool\"") }), .TestIntrinsicTypeMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "Equal", &.{ intExpr("1"), boolExpr(true), stringExpr("\"types should match\"") }), .TestIntrinsicTypeMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("1"), try qualifiedCallExpr(allocator, "Is", "True", &.{}), stringExpr("\"true relation needs bool actual\"") }), .TestIntrinsicTypeMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(true), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{intExpr("1")}), stringExpr("\"relation expected type should match actual\"") }), .TestIntrinsicTypeMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("1"), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{boolExpr(true)}), stringExpr("\"relation expected type should match actual\"") }), .TestIntrinsicTypeMismatch);

    const statements = try allocator.alloc(ast.Stmt, 3);
    statements[0] = try localStmt(allocator, "x", intExpr("1"));
    statements[1] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "Equal", &.{ .{ .address_of = .{ .operand = try allocExpr(allocator, identExpr("x")), .span = .{ .start = 0, .length = 1 } } }, .{ .address_of = .{ .operand = try allocExpr(allocator, identExpr("x")), .span = .{ .start = 0, .length = 1 } } }, stringExpr("\"pointer equality is not v0\"") }));
    statements[2] = try returnStmt(allocator, intExpr("0"));
    try expectOneTestSemanticDiagnostic(&.{functionWithBody("main", &.{}, statements)}, .ExpectEqualUnsupportedType);

    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ intExpr("1"), try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{stringExpr("\"text\"")}), stringExpr("\"string equality is not v0\"") }), .TestRelationUnsupported);
}

test "test intrinsic semantics reject wrong arity and normal source usage" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "True", &.{ boolExpr(true), stringExpr("\"reason\""), boolExpr(false) }), .TestIntrinsicArityMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(true), try qualifiedCallExpr(allocator, "Is", "True", &.{}), stringExpr("\"reason\""), boolExpr(false) }), .TestIntrinsicArityMismatch);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(true), try qualifiedCallExpr(allocator, "Unknown", "Relation", &.{}), stringExpr("\"reason\"") }), .TestRelationUnsupported);
    try expectOneTestIntrinsicDiagnostic(try qualifiedCallExpr(allocator, "Is", "EqualTo", &.{intExpr("1")}), .TestRelationOutsideExpectThat);

    const statements = try allocator.alloc(ast.Stmt, 2);
    statements[0] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "True", &.{ boolExpr(true), stringExpr("\"reason\"") }));
    statements[1] = try returnStmt(allocator, intExpr("0"));
    try expectOneSemanticDiagnostic(&.{functionWithBody("main", &.{}, statements)}, .TestIntrinsicOutsideTestFile);

    const that_statements = try allocator.alloc(ast.Stmt, 2);
    that_statements[0] = try exprStmt(allocator, try qualifiedCallExpr(allocator, "Expect", "That", &.{ boolExpr(true), try qualifiedCallExpr(allocator, "Is", "True", &.{}), stringExpr("\"reason\"") }));
    that_statements[1] = try returnStmt(allocator, intExpr("0"));
    try expectOneSemanticDiagnostic(&.{functionWithBody("main", &.{}, that_statements)}, .TestIntrinsicOutsideTestFile);
}

fn sameType(left: types.TypeId, right: types.TypeId) bool {
    return left.index == right.index;
}

fn isCompilerKnownTypeName(text: []const u8) bool {
    return std.mem.eql(u8, text, "Arena") or
        std.mem.eql(u8, text, "Allocator") or
        std.mem.eql(u8, text, "AllocError");
}
