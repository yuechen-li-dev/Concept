const std = @import("std");

const interner_module = @import("interner.zig");
const types = @import("types.zig");
const compile_time_capability = @import("compile_time_capability.zig");
const compile_time_target = @import("compile_time_target.zig");
const source = @import("source.zig");

pub const SourceSpan = source.SourceSpan;
pub const synthetic_span = SourceSpan{ .start = 0, .length = 0 };
pub const Interner = interner_module.Interner;
pub const SymbolId = interner_module.SymbolId;

pub const ItemId = SemanticId("ItemId");
pub const FunctionId = SemanticId("FunctionId");
pub const GenericFunctionId = SemanticId("GenericFunctionId");
pub const ConceptId = SemanticId("ConceptId");
pub const ConceptImplId = SemanticId("ConceptImplId");
pub const StructId = SemanticId("StructId");
pub const EnumId = SemanticId("EnumId");
pub const FieldId = SemanticId("FieldId");
pub const VariantId = SemanticId("VariantId");
pub const LocalId = SemanticId("LocalId");
pub const ParamId = SemanticId("ParamId");
pub const StmtId = SemanticId("StmtId");
pub const ExprId = SemanticId("ExprId");
pub const EnumPayloadFieldId = SemanticId("EnumPayloadFieldId");

fn SemanticId(comptime label: []const u8) type {
    return struct {
        index: u32,

        pub fn format(self: @This(), writer: *std.Io.Writer) std.Io.Writer.Error!void {
            try writer.print(label ++ "({d})", .{self.index});
        }
    };
}

pub const HirItem = union(enum) {
    function: FunctionId,
    struct_: StructId,
    enum_: EnumId,
};

pub const HirTypeConstraint = struct {
    text: []const u8,
    span: SourceSpan,
    concept_id: ?ConceptId = null,
    type_args: []types.TypeId = &.{},
};

pub const HirTypeParam = struct {
    name: SymbolId,
    span: SourceSpan,
    type_id: types.TypeId,
    constraint: ?HirTypeConstraint = null,
};

pub const HirGenericFunction = struct {
    name: SymbolId,
    span: SourceSpan,
    type_params: []HirTypeParam,
    function: FunctionId,
};

pub const HirConceptParam = struct {
    name: SymbolId,
    span: SourceSpan,
    type_id: types.TypeId,
};

pub const HirConceptRequirement = struct {
    name: SymbolId,
    return_type: types.TypeId,
    params: []HirConceptParam,
    span: SourceSpan,
};

pub const MarkerKind = enum {
    user,
    copy,
    move,
    trivial,
    relocatable,
    pod,

    pub fn fromDeclaredName(name: []const u8, arity: usize) MarkerKind {
        if (arity != 1) return .user;
        if (std.mem.eql(u8, name, "Copy")) return .copy;
        if (std.mem.eql(u8, name, "Move")) return .move;
        if (std.mem.eql(u8, name, "Trivial")) return .trivial;
        if (std.mem.eql(u8, name, "Relocatable")) return .relocatable;
        if (std.mem.eql(u8, name, "Pod")) return .pod;
        return .user;
    }

    pub fn displayName(self: MarkerKind) []const u8 {
        return switch (self) {
            .user => "User",
            .copy => "Copy",
            .move => "Move",
            .trivial => "Trivial",
            .relocatable => "Relocatable",
            .pod => "Pod",
        };
    }
};

pub const HirConcept = struct {
    name: SymbolId,
    span: SourceSpan,
    type_params: []HirTypeParam,
    requirements: []HirConceptRequirement,
    is_marker: bool = false,
    is_unsafe: bool = false,
    known_marker_kind: MarkerKind = .user,
};

pub const HirConceptImpl = struct {
    concept_id: ConceptId,
    target_type: types.TypeId,
    functions: []FunctionId,
    is_unsafe: bool = false,
    span: SourceSpan,
};

pub const HirStaticAssert = struct {
    expr: ExprId,
    span: SourceSpan,
};

pub const CompileTimeCapability = compile_time_capability.CompileTimeCapability;
pub const CompileTimeCapabilitySet = compile_time_capability.CompileTimeCapabilitySet;
pub const CompileTimeCapabilityList = compile_time_capability.CompileTimeCapabilityList;
pub const CompileTimeCapabilityRequired = compile_time_capability.CompileTimeCapabilityRequired;
pub const CompileTimeTargetQuery = compile_time_target.CompileTimeTargetQuery;

pub const HirFunction = struct {
    item: ItemId,
    name: SymbolId,
    span: SourceSpan,
    return_type: types.TypeId,
    is_unsafe: bool = false,
    is_compile_time: bool = false,
    compile_time_capabilities: []CompileTimeCapabilityRequired = &.{},
    params: []ParamId,
    locals: []LocalId,
    body: ?StmtId = null,
    is_instantiation: bool = false,
    is_concept_witness: bool = false,
    is_referenced_concept_witness: bool = false,
};

pub const HirLocal = struct {
    parent: FunctionId,
    name: SymbolId,
    span: SourceSpan,
    type_id: types.TypeId,
};

pub const AssignBase = union(enum) {
    local: LocalId,
    param: ParamId,
};

pub const AssignTarget = union(enum) {
    local: LocalId,
    param: ParamId,
    field: struct { base: AssignBase, field_id: FieldId, field_span: SourceSpan },
};

pub const HirStmt = struct {
    span: SourceSpan,
    kind: HirStmtKind,
};

pub const HirStmtKind = union(enum) {
    block: []StmtId,
    return_stmt: ?ExprId,
    local_decl: struct { local: LocalId, initializer: ExprId },
    assignment: struct { target: AssignTarget, value: ExprId },
    expr_stmt: ExprId,
    discard_stmt: ExprId,
    if_stmt: struct { condition: ExprId, then_block: StmtId, else_block: ?StmtId },
    while_stmt: struct { condition: ExprId, body: StmtId },
    unsafe_block: StmtId,
    match_stmt: struct { scrutinee: ExprId, arms: []HirMatchArm },
};

pub const HirPatternBinding = struct {
    name: SymbolId,
    local: LocalId,
    payload_field: EnumPayloadFieldId,
    type_id: types.TypeId,
    span: SourceSpan,
};

pub const HirMatchPattern = union(enum) {
    int_literal: []const u8,
    bool_literal: bool,
    wildcard,
    enum_variant: struct {
        enum_id: EnumId,
        variant_id: VariantId,
        bindings: []HirPatternBinding,
    },
};

pub const HirMatchArm = struct {
    pattern: HirMatchPattern,
    pattern_span: SourceSpan,
    body: StmtId,
};

pub const HirExpr = struct {
    span: SourceSpan,
    kind: HirExprKind,
};

pub const HirStructLiteralField = struct {
    field_id: FieldId,
    value: ExprId,
    span: SourceSpan,
};

pub const HirDecideArm = struct {
    variant_id: VariantId,
    condition: ?ExprId,
    score: ExprId,
    span: SourceSpan,
};

pub const HirExprKind = union(enum) {
    int_literal: []const u8,
    bool_literal: bool,
    local_ref: LocalId,
    param_ref: ParamId,
    call: struct { function: FunctionId, args: []ExprId },
    concept_requirement_call: struct { concept_id: ConceptId, requirement_index: u32, args: []ExprId },
    enum_constructor: struct { enum_id: EnumId, variant_id: VariantId, args: []ExprId },
    struct_literal: struct { struct_id: StructId, type_id: types.TypeId, fields: []HirStructLiteralField },
    field_access: struct { receiver: ExprId, field_name: SymbolId, field_span: SourceSpan },
    target_metadata: struct { query: CompileTimeTargetQuery, field_span: SourceSpan },
    decide: struct { enum_type: types.TypeId, enum_id: EnumId, arms: []HirDecideArm },
    group: ExprId,
    unary: struct { op: UnaryOp, operand: ExprId },
    address_of: ExprId,
    deref: ExprId,
    move_expr: ExprId,
    try_expr: ExprId,
    compile_time: struct { operand: ExprId, span: SourceSpan },
    binary: struct { op: BinaryOp, left: ExprId, right: ExprId },
};

pub const UnaryOp = enum {
    negate,
    logical_not,

    pub fn lexeme(self: UnaryOp) []const u8 {
        return switch (self) {
            .negate => "-",
            .logical_not => "!",
        };
    }
};

pub const BinaryOp = enum {
    multiply,
    divide,
    modulo,
    add,
    subtract,
    less,
    less_equal,
    greater,
    greater_equal,
    equal_equal,
    bang_equal,
    logical_and,
    logical_or,

    pub fn lexeme(self: BinaryOp) []const u8 {
        return switch (self) {
            .multiply => "*",
            .divide => "/",
            .modulo => "%",
            .add => "+",
            .subtract => "-",
            .less => "<",
            .less_equal => "<=",
            .greater => ">",
            .greater_equal => ">=",
            .equal_equal => "==",
            .bang_equal => "!=",
            .logical_and => "&&",
            .logical_or => "||",
        };
    }
};

pub const HirParam = struct {
    parent: FunctionId,
    name: SymbolId,
    span: SourceSpan,
    type_id: types.TypeId,
};

pub const HirStruct = struct {
    item: ItemId,
    name: SymbolId,
    fields: []FieldId,
};

pub const HirResultShape = struct {
    ok_variant: VariantId,
    err_variant: VariantId,
    ok_payload: EnumPayloadFieldId,
    err_payload: EnumPayloadFieldId,
    ok_type: types.TypeId,
    err_type: types.TypeId,
};

pub const HirEnum = struct {
    item: ItemId,
    name: SymbolId,
    variants: []VariantId,
    is_must_use: bool = false,
    result_shape: ?HirResultShape = null,
};

pub const HirField = struct {
    parent: StructId,
    name: SymbolId,
    span: SourceSpan,
    type_id: types.TypeId,
};

pub const HirVariant = struct {
    parent: EnumId,
    name: SymbolId,
    span: SourceSpan,
    payload_fields: []EnumPayloadFieldId,
};

pub const HirEnumPayloadField = struct {
    parent: VariantId,
    name: SymbolId,
    span: SourceSpan,
    type_id: types.TypeId,
};

pub const HirModule = struct {
    store: HirStore,

    pub fn init(allocator: std.mem.Allocator) HirModule {
        return .{ .store = HirStore.init(allocator) };
    }

    pub fn deinit(self: *HirModule) void {
        self.store.deinit();
        self.* = undefined;
    }
};

pub const HirStore = struct {
    allocator: std.mem.Allocator,
    items: std.ArrayList(HirItem),
    functions: std.ArrayList(HirFunction),
    generic_functions: std.ArrayList(HirGenericFunction),
    concepts: std.ArrayList(HirConcept),
    concept_impls: std.ArrayList(HirConceptImpl),
    params: std.ArrayList(HirParam),
    locals: std.ArrayList(HirLocal),
    stmts: std.ArrayList(HirStmt),
    exprs: std.ArrayList(HirExpr),
    static_asserts: std.ArrayList(HirStaticAssert),
    structs: std.ArrayList(HirStruct),
    enums: std.ArrayList(HirEnum),
    fields: std.ArrayList(HirField),
    variants: std.ArrayList(HirVariant),
    enum_payload_fields: std.ArrayList(HirEnumPayloadField),

    pub fn init(allocator: std.mem.Allocator) HirStore {
        return .{
            .allocator = allocator,
            .items = std.ArrayList(HirItem).empty,
            .functions = std.ArrayList(HirFunction).empty,
            .generic_functions = std.ArrayList(HirGenericFunction).empty,
            .concepts = std.ArrayList(HirConcept).empty,
            .concept_impls = std.ArrayList(HirConceptImpl).empty,
            .params = std.ArrayList(HirParam).empty,
            .locals = std.ArrayList(HirLocal).empty,
            .stmts = std.ArrayList(HirStmt).empty,
            .exprs = std.ArrayList(HirExpr).empty,
            .static_asserts = std.ArrayList(HirStaticAssert).empty,
            .structs = std.ArrayList(HirStruct).empty,
            .enums = std.ArrayList(HirEnum).empty,
            .fields = std.ArrayList(HirField).empty,
            .variants = std.ArrayList(HirVariant).empty,
            .enum_payload_fields = std.ArrayList(HirEnumPayloadField).empty,
        };
    }

    pub fn deinit(self: *HirStore) void {
        for (self.generic_functions.items) |generic_function| {
            for (generic_function.type_params) |type_param| {
                if (type_param.constraint) |constraint| {
                    self.allocator.free(constraint.text);
                    if (constraint.type_args.len > 0) self.allocator.free(constraint.type_args);
                }
            }
            if (generic_function.type_params.len > 0) self.allocator.free(generic_function.type_params);
        }
        for (self.concepts.items) |concept| {
            for (concept.type_params) |type_param| {
                if (type_param.constraint) |constraint| {
                    self.allocator.free(constraint.text);
                    if (constraint.type_args.len > 0) self.allocator.free(constraint.type_args);
                }
            }
            if (concept.type_params.len > 0) self.allocator.free(concept.type_params);
            for (concept.requirements) |requirement| {
                if (requirement.params.len > 0) self.allocator.free(requirement.params);
            }
            if (concept.requirements.len > 0) self.allocator.free(concept.requirements);
        }
        for (self.concept_impls.items) |concept_impl| {
            if (concept_impl.functions.len > 0) self.allocator.free(concept_impl.functions);
        }
        for (self.functions.items) |function| {
            for (function.compile_time_capabilities) |capability| self.allocator.free(capability.name);
            if (function.compile_time_capabilities.len > 0) self.allocator.free(function.compile_time_capabilities);
            if (function.params.len > 0) self.allocator.free(function.params);
            if (function.locals.len > 0) self.allocator.free(function.locals);
        }
        for (self.stmts.items) |stmt| {
            switch (stmt.kind) {
                .block => |children| if (children.len > 0) self.allocator.free(children),
                .match_stmt => |match_stmt| if (match_stmt.arms.len > 0) {
                    for (match_stmt.arms) |arm| {
                        switch (arm.pattern) {
                            .int_literal => |text| self.allocator.free(text),
                            .enum_variant => |pattern| if (pattern.bindings.len > 0) self.allocator.free(pattern.bindings),
                            else => {},
                        }
                    }
                    self.allocator.free(match_stmt.arms);
                },
                else => {},
            }
        }
        for (self.exprs.items) |expr| {
            switch (expr.kind) {
                .int_literal => |text| self.allocator.free(text),
                .call => |call| if (call.args.len > 0) self.allocator.free(call.args),
                .concept_requirement_call => |call| if (call.args.len > 0) self.allocator.free(call.args),
                .enum_constructor => |constructor| if (constructor.args.len > 0) self.allocator.free(constructor.args),
                .struct_literal => |literal| if (literal.fields.len > 0) self.allocator.free(literal.fields),
                .decide => |decide| if (decide.arms.len > 0) self.allocator.free(decide.arms),
                else => {},
            }
        }
        for (self.structs.items) |struct_decl| {
            if (struct_decl.fields.len > 0) self.allocator.free(struct_decl.fields);
        }
        for (self.enums.items) |enum_decl| {
            if (enum_decl.variants.len > 0) self.allocator.free(enum_decl.variants);
        }

        for (self.variants.items) |variant| {
            if (variant.payload_fields.len > 0) self.allocator.free(variant.payload_fields);
        }
        self.enum_payload_fields.deinit(self.allocator);
        self.variants.deinit(self.allocator);
        self.fields.deinit(self.allocator);
        self.enums.deinit(self.allocator);
        self.structs.deinit(self.allocator);
        self.static_asserts.deinit(self.allocator);
        self.exprs.deinit(self.allocator);
        self.stmts.deinit(self.allocator);
        self.locals.deinit(self.allocator);
        self.params.deinit(self.allocator);
        self.concept_impls.deinit(self.allocator);
        self.concepts.deinit(self.allocator);
        self.generic_functions.deinit(self.allocator);
        self.functions.deinit(self.allocator);
        self.items.deinit(self.allocator);
        self.* = undefined;
    }

    pub fn addGenericFunction(self: *HirStore, name: SymbolId, span: SourceSpan) !GenericFunctionId {
        const function = try self.addFunctionStorage(name, types.TypeId{ .index = 0 }, false, span);
        errdefer _ = self.functions.pop();
        const id = GenericFunctionId{ .index = try nextIndex(self.generic_functions.items.len, error.TooManyFunctions) };
        try self.generic_functions.append(self.allocator, .{
            .name = name,
            .span = span,
            .type_params = &.{},
            .function = function,
        });
        return id;
    }

    pub fn setGenericFunctionTypeParams(self: *HirStore, id: GenericFunctionId, type_params: []HirTypeParam) void {
        const generic = self.getGenericFunctionMut(id);
        if (generic.type_params.len > 0) self.allocator.free(generic.type_params);
        generic.type_params = type_params;
    }

    pub fn addConcept(self: *HirStore, name: SymbolId, is_marker: bool, is_unsafe: bool, span: SourceSpan) !ConceptId {
        const id = ConceptId{ .index = try nextIndex(self.concepts.items.len, error.TooManyConcepts) };
        try self.concepts.append(self.allocator, .{
            .name = name,
            .span = span,
            .type_params = &.{},
            .requirements = &.{},
            .is_marker = is_marker,
            .is_unsafe = is_unsafe,
        });
        return id;
    }

    pub fn setConceptTypeParams(self: *HirStore, id: ConceptId, type_params: []HirTypeParam) void {
        const concept = self.getConceptMut(id);
        if (concept.type_params.len > 0) self.allocator.free(concept.type_params);
        concept.type_params = type_params;
    }

    pub fn setConceptKnownMarkerKind(self: *HirStore, id: ConceptId, kind: MarkerKind) void {
        self.getConceptMut(id).known_marker_kind = kind;
    }

    pub fn setConceptRequirements(self: *HirStore, id: ConceptId, requirements: []HirConceptRequirement) void {
        const concept = self.getConceptMut(id);
        for (concept.requirements) |requirement| {
            if (requirement.params.len > 0) self.allocator.free(requirement.params);
        }
        if (concept.requirements.len > 0) self.allocator.free(concept.requirements);
        concept.requirements = requirements;
    }

    pub fn genericFunctionFor(self: *const HirStore, function_id: FunctionId) ?GenericFunctionId {
        for (self.generic_functions.items, 0..) |generic, index| {
            if (generic.function.index == function_id.index) return .{ .index = @intCast(index) };
        }
        return null;
    }

    pub fn isGenericFunction(self: *const HirStore, function_id: FunctionId) bool {
        return self.genericFunctionFor(function_id) != null;
    }

    pub fn isConceptWitnessFunction(self: *const HirStore, function_id: FunctionId) bool {
        return self.getFunction(function_id).is_concept_witness;
    }

    pub fn isReferencedConceptWitnessFunction(self: *const HirStore, function_id: FunctionId) bool {
        const function = self.getFunction(function_id);
        return function.is_concept_witness and function.is_referenced_concept_witness;
    }

    pub fn markConceptWitnessReferenced(self: *HirStore, function_id: FunctionId) void {
        const function = self.getFunctionMut(function_id);
        std.debug.assert(function.is_concept_witness);
        function.is_referenced_concept_witness = true;
    }

    pub fn hasConceptImpl(self: *const HirStore, concept_id: ConceptId, target_type: types.TypeId) bool {
        return self.findConceptImpl(concept_id, target_type) != null;
    }

    pub fn findConceptImpl(self: *const HirStore, concept_id: ConceptId, target_type: types.TypeId) ?ConceptImplId {
        for (self.concept_impls.items, 0..) |concept_impl, index| {
            if (concept_impl.concept_id.index == concept_id.index and concept_impl.target_type.index == target_type.index) {
                return .{ .index = @intCast(index) };
            }
        }
        return null;
    }

    pub fn addFunction(self: *HirStore, name: SymbolId, return_type: types.TypeId, span: SourceSpan) !FunctionId {
        return self.addFunctionWithSafety(name, return_type, false, span);
    }

    pub fn addFunctionWithSafety(self: *HirStore, name: SymbolId, return_type: types.TypeId, is_unsafe: bool, span: SourceSpan) !FunctionId {
        const id = try self.addFunctionStorage(name, return_type, is_unsafe, span);
        const item = try self.addItem(.{ .function = id });
        errdefer _ = self.items.pop();
        self.getFunctionMut(id).item = item;
        return id;
    }

    pub fn addConceptWitnessFunction(self: *HirStore, name: SymbolId, return_type: types.TypeId, is_unsafe: bool, span: SourceSpan) !FunctionId {
        const id = try self.addFunctionStorage(name, return_type, is_unsafe, span);
        self.getFunctionMut(id).is_concept_witness = true;
        return id;
    }

    pub fn addConceptImpl(self: *HirStore, concept_id: ConceptId, target_type: types.TypeId, functions: []FunctionId, is_unsafe: bool, span: SourceSpan) !ConceptImplId {
        const id = ConceptImplId{ .index = try nextIndex(self.concept_impls.items.len, error.TooManyConceptImpls) };
        try self.concept_impls.append(self.allocator, .{
            .concept_id = concept_id,
            .target_type = target_type,
            .functions = functions,
            .is_unsafe = is_unsafe,
            .span = span,
        });
        return id;
    }

    fn addFunctionStorage(self: *HirStore, name: SymbolId, return_type: types.TypeId, is_unsafe: bool, span: SourceSpan) !FunctionId {
        const id = FunctionId{ .index = try nextIndex(self.functions.items.len, error.TooManyFunctions) };
        try self.functions.append(self.allocator, .{
            .item = ItemId{ .index = std.math.maxInt(u32) },
            .name = name,
            .span = span,
            .return_type = return_type,
            .is_unsafe = is_unsafe,
            .is_compile_time = false,
            .compile_time_capabilities = &.{},
            .params = &.{},
            .locals = &.{},
            .body = null,
            .is_instantiation = false,
            .is_concept_witness = false,
            .is_referenced_concept_witness = false,
        });
        return id;
    }

    pub fn markFunctionCompileTime(self: *HirStore, function_id: FunctionId) void {
        self.getFunctionMut(function_id).is_compile_time = true;
    }

    pub fn setFunctionCompileTimeCapabilities(self: *HirStore, function_id: FunctionId, capabilities: []CompileTimeCapabilityRequired) void {
        const function = self.getFunctionMut(function_id);
        for (function.compile_time_capabilities) |capability| self.allocator.free(capability.name);
        if (function.compile_time_capabilities.len > 0) self.allocator.free(function.compile_time_capabilities);
        function.compile_time_capabilities = capabilities;
    }

    pub fn addStruct(self: *HirStore, name: SymbolId) !StructId {
        const id = StructId{ .index = try nextIndex(self.structs.items.len, error.TooManyStructs) };
        const item = try self.addItem(.{ .struct_ = id });
        errdefer _ = self.items.pop();
        try self.structs.append(self.allocator, .{
            .item = item,
            .name = name,
            .fields = &.{},
        });
        return id;
    }

    pub fn addEnum(self: *HirStore, name: SymbolId, is_must_use: bool) !EnumId {
        const id = EnumId{ .index = try nextIndex(self.enums.items.len, error.TooManyEnums) };
        const item = try self.addItem(.{ .enum_ = id });
        errdefer _ = self.items.pop();
        try self.enums.append(self.allocator, .{
            .item = item,
            .name = name,
            .variants = &.{},
            .is_must_use = is_must_use,
            .result_shape = null,
        });
        return id;
    }

    pub fn addParam(self: *HirStore, parent: FunctionId, name: SymbolId, type_id: types.TypeId, span: SourceSpan) !ParamId {
        _ = self.getFunction(parent);
        const id = ParamId{ .index = try nextIndex(self.params.items.len, error.TooManyParams) };
        try self.params.append(self.allocator, .{ .parent = parent, .name = name, .span = span, .type_id = type_id });
        errdefer _ = self.params.pop();

        const function_decl = self.getFunctionMut(parent);
        function_decl.params = try appendId(self.allocator, ParamId, function_decl.params, id);
        return id;
    }

    pub fn addLocal(self: *HirStore, parent: FunctionId, name: SymbolId, type_id: types.TypeId, span: SourceSpan) !LocalId {
        _ = self.getFunction(parent);
        const id = LocalId{ .index = try nextIndex(self.locals.items.len, error.TooManyLocals) };
        try self.locals.append(self.allocator, .{ .parent = parent, .name = name, .span = span, .type_id = type_id });
        errdefer _ = self.locals.pop();

        const function_decl = self.getFunctionMut(parent);
        function_decl.locals = try appendId(self.allocator, LocalId, function_decl.locals, id);
        return id;
    }

    pub fn addStmt(self: *HirStore, kind: HirStmtKind, span: SourceSpan) !StmtId {
        const id = StmtId{ .index = try nextIndex(self.stmts.items.len, error.TooManyStmts) };
        try self.stmts.append(self.allocator, .{ .span = span, .kind = kind });
        return id;
    }

    pub fn addExpr(self: *HirStore, kind: HirExprKind, span: SourceSpan) !ExprId {
        const id = ExprId{ .index = try nextIndex(self.exprs.items.len, error.TooManyExprs) };
        try self.exprs.append(self.allocator, .{ .span = span, .kind = kind });
        return id;
    }

    pub fn addStaticAssert(self: *HirStore, expr: ExprId, span: SourceSpan) !void {
        try self.static_asserts.append(self.allocator, .{ .expr = expr, .span = span });
    }

    pub fn setFunctionBody(self: *HirStore, id: FunctionId, body: StmtId) void {
        self.getFunctionMut(id).body = body;
    }

    pub fn markFunctionInstantiation(self: *HirStore, id: FunctionId) void {
        self.getFunctionMut(id).is_instantiation = true;
    }

    pub fn setFunctionReturnType(self: *HirStore, id: FunctionId, type_id: types.TypeId) void {
        self.getFunctionMut(id).return_type = type_id;
    }

    pub fn setEnumResultShape(self: *HirStore, id: EnumId, result_shape: ?HirResultShape) void {
        self.getEnumMut(id).result_shape = result_shape;
    }

    pub fn addField(self: *HirStore, parent: StructId, name: SymbolId, type_id: types.TypeId, span: SourceSpan) !FieldId {
        _ = self.getStruct(parent);
        const id = FieldId{ .index = try nextIndex(self.fields.items.len, error.TooManyFields) };
        try self.fields.append(self.allocator, .{ .parent = parent, .name = name, .span = span, .type_id = type_id });
        errdefer _ = self.fields.pop();

        const struct_decl = self.getStructMut(parent);
        struct_decl.fields = try appendId(self.allocator, FieldId, struct_decl.fields, id);
        return id;
    }

    pub fn addVariant(self: *HirStore, parent: EnumId, name: SymbolId, span: SourceSpan) !VariantId {
        _ = self.getEnum(parent);
        const id = VariantId{ .index = try nextIndex(self.variants.items.len, error.TooManyVariants) };
        try self.variants.append(self.allocator, .{ .parent = parent, .name = name, .span = span, .payload_fields = &.{} });
        errdefer _ = self.variants.pop();

        const enum_decl = self.getEnumMut(parent);
        enum_decl.variants = try appendId(self.allocator, VariantId, enum_decl.variants, id);
        return id;
    }

    pub fn addEnumPayloadField(self: *HirStore, parent: VariantId, name: SymbolId, type_id: types.TypeId, span: SourceSpan) !EnumPayloadFieldId {
        _ = self.getVariant(parent);
        const id = EnumPayloadFieldId{ .index = try nextIndex(self.enum_payload_fields.items.len, error.TooManyEnumPayloadFields) };
        try self.enum_payload_fields.append(self.allocator, .{ .parent = parent, .name = name, .span = span, .type_id = type_id });
        errdefer _ = self.enum_payload_fields.pop();

        const variant = self.getVariantMut(parent);
        variant.payload_fields = try appendId(self.allocator, EnumPayloadFieldId, variant.payload_fields, id);
        return id;
    }

    pub fn getItem(self: *const HirStore, id: ItemId) *const HirItem {
        const index: usize = id.index;
        std.debug.assert(index < self.items.items.len);
        return &self.items.items[index];
    }

    pub fn getGenericFunction(self: *const HirStore, id: GenericFunctionId) *const HirGenericFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.generic_functions.items.len);
        return &self.generic_functions.items[index];
    }

    pub fn getConcept(self: *const HirStore, id: ConceptId) *const HirConcept {
        const index: usize = id.index;
        std.debug.assert(index < self.concepts.items.len);
        return &self.concepts.items[index];
    }

    pub fn getConceptImpl(self: *const HirStore, id: ConceptImplId) *const HirConceptImpl {
        const index: usize = id.index;
        std.debug.assert(index < self.concept_impls.items.len);
        return &self.concept_impls.items[index];
    }

    pub fn getFunction(self: *const HirStore, id: FunctionId) *const HirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    pub fn getParam(self: *const HirStore, id: ParamId) *const HirParam {
        const index: usize = id.index;
        std.debug.assert(index < self.params.items.len);
        return &self.params.items[index];
    }

    pub fn getLocal(self: *const HirStore, id: LocalId) *const HirLocal {
        const index: usize = id.index;
        std.debug.assert(index < self.locals.items.len);
        return &self.locals.items[index];
    }

    pub fn getStmt(self: *const HirStore, id: StmtId) *const HirStmt {
        const index: usize = id.index;
        std.debug.assert(index < self.stmts.items.len);
        return &self.stmts.items[index];
    }

    pub fn getExpr(self: *const HirStore, id: ExprId) *const HirExpr {
        const index: usize = id.index;
        std.debug.assert(index < self.exprs.items.len);
        return &self.exprs.items[index];
    }

    pub fn getExprMut(self: *HirStore, id: ExprId) *HirExpr {
        const index: usize = id.index;
        std.debug.assert(index < self.exprs.items.len);
        return &self.exprs.items[index];
    }

    pub fn getStruct(self: *const HirStore, id: StructId) *const HirStruct {
        const index: usize = id.index;
        std.debug.assert(index < self.structs.items.len);
        return &self.structs.items[index];
    }

    pub fn getEnum(self: *const HirStore, id: EnumId) *const HirEnum {
        const index: usize = id.index;
        std.debug.assert(index < self.enums.items.len);
        return &self.enums.items[index];
    }

    pub fn getResultShape(self: *const HirStore, id: EnumId) ?HirResultShape {
        return self.getEnum(id).result_shape;
    }

    pub fn isResultShapedEnum(self: *const HirStore, id: EnumId) bool {
        return self.getResultShape(id) != null;
    }

    pub fn getField(self: *const HirStore, id: FieldId) *const HirField {
        const index: usize = id.index;
        std.debug.assert(index < self.fields.items.len);
        return &self.fields.items[index];
    }

    pub fn getVariant(self: *const HirStore, id: VariantId) *const HirVariant {
        const index: usize = id.index;
        std.debug.assert(index < self.variants.items.len);
        return &self.variants.items[index];
    }

    pub fn getEnumPayloadField(self: *const HirStore, id: EnumPayloadFieldId) *const HirEnumPayloadField {
        const index: usize = id.index;
        std.debug.assert(index < self.enum_payload_fields.items.len);
        return &self.enum_payload_fields.items[index];
    }

    fn getGenericFunctionMut(self: *HirStore, id: GenericFunctionId) *HirGenericFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.generic_functions.items.len);
        return &self.generic_functions.items[index];
    }

    fn getConceptMut(self: *HirStore, id: ConceptId) *HirConcept {
        const index: usize = id.index;
        std.debug.assert(index < self.concepts.items.len);
        return &self.concepts.items[index];
    }

    fn getFunctionMut(self: *HirStore, id: FunctionId) *HirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    fn getStructMut(self: *HirStore, id: StructId) *HirStruct {
        const index: usize = id.index;
        std.debug.assert(index < self.structs.items.len);
        return &self.structs.items[index];
    }

    fn getVariantMut(self: *HirStore, id: VariantId) *HirVariant {
        const index: usize = id.index;
        std.debug.assert(index < self.variants.items.len);
        return &self.variants.items[index];
    }

    fn getEnumMut(self: *HirStore, id: EnumId) *HirEnum {
        const index: usize = id.index;
        std.debug.assert(index < self.enums.items.len);
        return &self.enums.items[index];
    }

    pub fn debugString(self: *const HirStore, allocator: std.mem.Allocator, interner: Interner) ![]u8 {
        var buffer: std.Io.Writer.Allocating = .init(allocator);
        errdefer buffer.deinit();
        const writer = &buffer.writer;

        try writer.writeAll("HirModule\n");
        for (self.static_asserts.items) |static_assert| {
            try writer.writeAll("  StaticAssert\n");
            try self.writeExprDebug(writer, static_assert.expr, 2);
        }
        for (self.generic_functions.items) |generic_function| {
            const function = self.getFunction(generic_function.function);
            try writer.print("  GenericFunction {s} {f} -> {f}\n", .{ interner.text(generic_function.name), generic_function.function, function.return_type });
            if (generic_function.type_params.len != 0) {
                try writer.writeAll("    TypeParams\n");
                for (generic_function.type_params, 0..) |type_param, type_param_index| {
                    try writer.print("      #{d} {s}: {f}", .{ type_param_index, interner.text(type_param.name), type_param.type_id });
                    if (type_param.constraint) |constraint| {
                        try writer.print(" constraint {s}", .{constraint.text});
                        if (constraint.concept_id) |concept_id| try writer.print(" resolved {f}", .{concept_id});
                    }
                    try writer.writeByte('\n');
                }
            }
            if (function.params.len != 0) {
                try writer.writeAll("    Params\n");
                for (function.params) |param_id| {
                    const param = self.getParam(param_id);
                    try writer.print("      {f} {s}: {f}\n", .{ param_id, interner.text(param.name), param.type_id });
                }
            }
            if (function.body) |body| {
                try writer.writeAll("    Body\n");
                try self.writeStmtDebug(writer, body, 3);
            }
        }
        for (self.concepts.items) |concept| {
            try writer.print("  {s}{s}Concept {s}", .{ if (concept.is_unsafe) "Unsafe " else "", if (concept.is_marker) "Marker " else "", interner.text(concept.name) });
            if (concept.known_marker_kind != .user) try writer.print(" known_marker={s}", .{concept.known_marker_kind.displayName()});
            try writer.writeByte('\n');
            if (concept.type_params.len != 0) {
                try writer.writeAll("    TypeParams\n");
                for (concept.type_params, 0..) |type_param, type_param_index| {
                    try writer.print("      #{d} {s}: {f}\n", .{ type_param_index, interner.text(type_param.name), type_param.type_id });
                }
            }
            if (concept.requirements.len != 0) {
                try writer.writeAll("    Requirements\n");
                for (concept.requirements) |requirement| {
                    try writer.print("      {s} -> {f}\n", .{ interner.text(requirement.name), requirement.return_type });
                    for (requirement.params) |param| {
                        try writer.print("        {s}: {f}\n", .{ interner.text(param.name), param.type_id });
                    }
                }
            }
        }
        for (self.concept_impls.items, 0..) |concept_impl, index| {
            const concept = self.getConcept(concept_impl.concept_id);
            try writer.print("  {s}ConceptImpl #{d} {s} for {f}\n", .{ if (concept_impl.is_unsafe) "Unsafe " else "", index, interner.text(concept.name), concept_impl.target_type });
            if (concept_impl.functions.len != 0) {
                try writer.writeAll("    Witnesses\n");
                for (concept_impl.functions) |function_id| {
                    const function = self.getFunction(function_id);
                    try writer.print("      {f} {s} -> {f}\n", .{ function_id, interner.text(function.name), function.return_type });
                }
            }
        }
        for (self.items.items) |item| {
            switch (item) {
                .function => |id| {
                    const function = self.getFunction(id);
                    try writer.print("  {s}{s}Function {s}", .{ if (function.is_compile_time) "CompileTime " else "", if (function.is_unsafe) "Unsafe " else "", interner.text(function.name) });
                    if (function.compile_time_capabilities.len != 0) {
                        try writer.writeAll(" capabilities=[");
                        for (function.compile_time_capabilities, 0..) |capability, capability_index| {
                            if (capability_index != 0) try writer.writeAll(", ");
                            try writer.writeAll(capability.name);
                        }
                        try writer.writeByte(']');
                    }
                    try writer.print(" -> {f}\n", .{function.return_type});
                    if (function.params.len != 0) {
                        try writer.writeAll("    Params\n");
                        for (function.params) |param_id| {
                            const param = self.getParam(param_id);
                            try writer.print("      {f} {s}: {f}\n", .{ param_id, interner.text(param.name), param.type_id });
                        }
                    }
                    if (function.locals.len != 0) {
                        try writer.writeAll("    Locals\n");
                        for (function.locals) |local_id| {
                            const local = self.getLocal(local_id);
                            try writer.print("      {f} {s}: {f}\n", .{ local_id, interner.text(local.name), local.type_id });
                        }
                    }
                    if (function.body) |body| {
                        try writer.writeAll("    Body\n");
                        try self.writeStmtDebug(writer, body, 3);
                    }
                },
                .struct_ => |id| {
                    const struct_decl = self.getStruct(id);
                    try writer.print("  Struct {s}\n", .{interner.text(struct_decl.name)});
                    for (struct_decl.fields) |field_id| {
                        const field = self.getField(field_id);
                        try writer.print("    Field {s}: {f}\n", .{ interner.text(field.name), field.type_id });
                    }
                },
                .enum_ => |id| {
                    const enum_decl = self.getEnum(id);
                    try writer.print("  {s}Enum {s}{s}\n", .{ if (enum_decl.is_must_use) "MustUse " else "", interner.text(enum_decl.name), if (enum_decl.result_shape != null) " result_shape" else "" });
                    for (enum_decl.variants) |variant_id| {
                        const variant = self.getVariant(variant_id);
                        try writer.print("    Variant {s}\n", .{interner.text(variant.name)});
                        for (variant.payload_fields) |payload_id| {
                            const payload = self.getEnumPayloadField(payload_id);
                            try writer.print("      Payload {s}: {f}\n", .{ interner.text(payload.name), payload.type_id });
                        }
                    }
                },
            }
        }

        return try buffer.toOwnedSlice();
    }

    fn writeStmtDebug(self: *const HirStore, writer: *std.Io.Writer, id: StmtId, depth: usize) !void {
        try writeIndent(writer, depth);
        switch (self.getStmt(id).kind) {
            .block => |children| {
                try writer.writeAll("Block\n");
                for (children) |child| try self.writeStmtDebug(writer, child, depth + 1);
            },
            .return_stmt => |maybe_value| {
                try writer.writeAll("Return\n");
                if (maybe_value) |value| try self.writeExprDebug(writer, value, depth + 1);
            },
            .local_decl => |decl| {
                try writer.print("LocalDecl {f}\n", .{decl.local});
                try self.writeExprDebug(writer, decl.initializer, depth + 1);
            },
            .assignment => |assignment| {
                try writer.writeAll("Assignment ");
                try writeAssignTarget(writer, assignment.target);
                try writer.writeByte('\n');
                try self.writeExprDebug(writer, assignment.value, depth + 1);
            },
            .expr_stmt => |expr_id| {
                try writer.writeAll("ExprStmt\n");
                try self.writeExprDebug(writer, expr_id, depth + 1);
            },
            .discard_stmt => |expr_id| {
                try writer.writeAll("Discard\n");
                try self.writeExprDebug(writer, expr_id, depth + 1);
            },
            .if_stmt => |stmt| {
                try writer.writeAll("If\n");
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Condition\n");
                try self.writeExprDebug(writer, stmt.condition, depth + 2);
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Then\n");
                try self.writeStmtDebug(writer, stmt.then_block, depth + 2);
                if (stmt.else_block) |else_block| {
                    try writeIndent(writer, depth + 1);
                    try writer.writeAll("Else\n");
                    try self.writeStmtDebug(writer, else_block, depth + 2);
                }
            },
            .while_stmt => |stmt| {
                try writer.writeAll("While\n");
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Condition\n");
                try self.writeExprDebug(writer, stmt.condition, depth + 2);
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Body\n");
                try self.writeStmtDebug(writer, stmt.body, depth + 2);
            },
            .unsafe_block => |body| {
                try writer.writeAll("UnsafeBlock\n");
                try self.writeStmtDebug(writer, body, depth + 1);
            },
            .match_stmt => |stmt| {
                try writer.writeAll("Match\n");
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Scrutinee\n");
                try self.writeExprDebug(writer, stmt.scrutinee, depth + 2);
                for (stmt.arms) |arm| {
                    try writeIndent(writer, depth + 1);
                    try writer.writeAll("Arm ");
                    try writePattern(writer, arm.pattern);
                    try writer.writeByte('\n');
                    try self.writeStmtDebug(writer, arm.body, depth + 2);
                }
            },
        }
    }

    fn writeExprDebug(self: *const HirStore, writer: *std.Io.Writer, id: ExprId, depth: usize) !void {
        try writeIndent(writer, depth);
        switch (self.getExpr(id).kind) {
            .int_literal => |text| try writer.print("Int {s}\n", .{text}),
            .bool_literal => |value| try writer.print("Bool {s}\n", .{if (value) "true" else "false"}),
            .local_ref => |local| try writer.print("LocalRef {f}\n", .{local}),
            .param_ref => |param| try writer.print("ParamRef {f}\n", .{param}),
            .call => |call| {
                try writer.print("Call {f}\n", .{call.function});
                for (call.args) |arg| try self.writeExprDebug(writer, arg, depth + 1);
            },
            .concept_requirement_call => |call| {
                try writer.print("ConceptRequirementCall {f} #{d}\n", .{ call.concept_id, call.requirement_index });
                for (call.args) |arg| try self.writeExprDebug(writer, arg, depth + 1);
            },
            .enum_constructor => |constructor| {
                try writer.print("EnumConstructor {f}::{f}\n", .{ constructor.enum_id, constructor.variant_id });
                for (constructor.args) |arg| try self.writeExprDebug(writer, arg, depth + 1);
            },
            .struct_literal => |literal| {
                try writer.print("StructLiteral {f} {f}\n", .{ literal.type_id, literal.struct_id });
                for (literal.fields) |field| {
                    try writeIndent(writer, depth + 1);
                    try writer.print("Field {f}\n", .{field.field_id});
                    try self.writeExprDebug(writer, field.value, depth + 2);
                }
            },
            .field_access => |field_access| {
                try writer.print("FieldAccess {f}\n", .{field_access.field_name});
                try self.writeExprDebug(writer, field_access.receiver, depth + 1);
            },
            .target_metadata => |metadata| {
                try writer.print("TargetMetadata {s}\n", .{metadata.query.sourceName()});
            },
            .decide => |decide| {
                try writer.print("Decide {f} {f}\n", .{ decide.enum_type, decide.enum_id });
                for (decide.arms) |arm| {
                    try writeIndent(writer, depth + 1);
                    try writer.print("Arm {f}{s}\n", .{ arm.variant_id, if (arm.condition != null) " when" else "" });
                    if (arm.condition) |condition| {
                        try writeIndent(writer, depth + 2);
                        try writer.writeAll("Condition\n");
                        try self.writeExprDebug(writer, condition, depth + 3);
                    }
                    try writeIndent(writer, depth + 2);
                    try writer.writeAll("Score\n");
                    try self.writeExprDebug(writer, arm.score, depth + 3);
                }
            },
            .group => |inner| {
                try writer.writeAll("Group\n");
                try self.writeExprDebug(writer, inner, depth + 1);
            },
            .unary => |unary| {
                try writer.print("Unary {s}\n", .{unary.op.lexeme()});
                try self.writeExprDebug(writer, unary.operand, depth + 1);
            },
            .address_of => |operand| {
                try writer.writeAll("AddressOf\n");
                try self.writeExprDebug(writer, operand, depth + 1);
            },
            .deref => |operand| {
                try writer.writeAll("Deref\n");
                try self.writeExprDebug(writer, operand, depth + 1);
            },
            .move_expr => |operand| {
                try writer.writeAll("Move\n");
                try self.writeExprDebug(writer, operand, depth + 1);
            },
            .try_expr => |operand| {
                try writer.writeAll("Try\n");
                try self.writeExprDebug(writer, operand, depth + 1);
            },
            .compile_time => |compile_time| {
                try writer.writeAll("CompileTime\n");
                try self.writeExprDebug(writer, compile_time.operand, depth + 1);
            },
            .binary => |binary| {
                try writer.print("Binary {s}\n", .{binary.op.lexeme()});
                try self.writeExprDebug(writer, binary.left, depth + 1);
                try self.writeExprDebug(writer, binary.right, depth + 1);
            },
        }
    }

    fn addItem(self: *HirStore, item: HirItem) !ItemId {
        const id = ItemId{ .index = try nextIndex(self.items.items.len, error.TooManyItems) };
        try self.items.append(self.allocator, item);
        return id;
    }
};

fn nextIndex(len: usize, overflow_error: anyerror) !u32 {
    if (len > std.math.maxInt(u32)) return overflow_error;
    return @intCast(len);
}

fn appendId(allocator: std.mem.Allocator, comptime Id: type, existing: []Id, id: Id) ![]Id {
    const expanded = try allocator.alloc(Id, existing.len + 1);
    @memcpy(expanded[0..existing.len], existing);
    expanded[existing.len] = id;
    if (existing.len > 0) allocator.free(existing);
    return expanded;
}

test "semantic ID debug formatting" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f} {f} {f} {f}", .{
        ItemId{ .index = 1 },
        FunctionId{ .index = 2 },
        FieldId{ .index = 3 },
        ParamId{ .index = 4 },
    });
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("ItemId(1) FunctionId(2) FieldId(3) ParamId(4)", rendered);
}

test "semantic IDs compare by value" {
    try std.testing.expectEqual(FunctionId{ .index = 7 }, FunctionId{ .index = 7 });
    try std.testing.expectEqual(StructId{ .index = 8 }, StructId{ .index = 8 });
    try std.testing.expectEqual(EnumId{ .index = 9 }, EnumId{ .index = 9 });
}

test "empty HIR store" {
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(usize, 0), store.items.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.functions.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.generic_functions.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.structs.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.enums.items.len);
}

test "add function with interned name and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const main_name = try interner.intern("main");
    const function_id = try store.addFunction(main_name, .{ .index = 1 }, synthetic_span);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(u32, 0), function_id.index);
    try std.testing.expectEqual(main_name, function.name);
    try std.testing.expectEqual(types.TypeId{ .index = 1 }, function.return_type);
    try std.testing.expectEqualStrings("main", interner.text(function.name));
    try std.testing.expectEqual(@as(usize, 0), function.params.len);
    try std.testing.expectEqual(@as(usize, 0), function.locals.len);
    try std.testing.expectEqual(HirItem{ .function = function_id }, store.getItem(function.item).*);
}

test "add struct with fields and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const vec3_id = try store.addStruct(try interner.intern("Vec3"));
    const x_id = try store.addField(vec3_id, try interner.intern("x"), .{ .index = 1 }, synthetic_span);
    const y_id = try store.addField(vec3_id, try interner.intern("y"), .{ .index = 1 }, synthetic_span);
    const z_id = try store.addField(vec3_id, try interner.intern("z"), .{ .index = 1 }, synthetic_span);
    const vec3 = store.getStruct(vec3_id);

    try std.testing.expectEqual(@as(usize, 3), vec3.fields.len);
    try std.testing.expectEqual(x_id, vec3.fields[0]);
    try std.testing.expectEqual(y_id, vec3.fields[1]);
    try std.testing.expectEqual(z_id, vec3.fields[2]);
    try std.testing.expectEqual(vec3_id, store.getField(x_id).parent);
    try std.testing.expectEqualStrings("x", interner.text(store.getField(x_id).name));
}

test "add enum with variants and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const token_id = try store.addEnum(try interner.intern("Token"), false);
    const identifier_id = try store.addVariant(token_id, try interner.intern("Identifier"), synthetic_span);
    const end_id = try store.addVariant(token_id, try interner.intern("End"), synthetic_span);
    const token = store.getEnum(token_id);

    try std.testing.expectEqual(@as(usize, 2), token.variants.len);
    try std.testing.expectEqual(identifier_id, token.variants[0]);
    try std.testing.expectEqual(end_id, token.variants[1]);
    try std.testing.expectEqual(token_id, store.getVariant(end_id).parent);
    try std.testing.expectEqualStrings("End", interner.text(store.getVariant(end_id).name));
}

test "HIR debug formatting uses interned names" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var module = HirModule.init(std.testing.allocator);
    defer module.deinit();

    _ = try module.store.addFunction(try interner.intern("main"), .{ .index = 0 }, synthetic_span);
    const vec3_id = try module.store.addStruct(try interner.intern("Vec3"));
    _ = try module.store.addField(vec3_id, try interner.intern("x"), .{ .index = 1 }, synthetic_span);
    _ = try module.store.addField(vec3_id, try interner.intern("y"), .{ .index = 1 }, synthetic_span);
    _ = try module.store.addField(vec3_id, try interner.intern("z"), .{ .index = 1 }, synthetic_span);
    const token_id = try module.store.addEnum(try interner.intern("Token"), false);
    _ = try module.store.addVariant(token_id, try interner.intern("Identifier"), synthetic_span);
    _ = try module.store.addVariant(token_id, try interner.intern("End"), synthetic_span);

    const rendered = try module.store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings(
        \\HirModule
        \\  Function main -> TypeId(0)
        \\  Struct Vec3
        \\    Field x: TypeId(1)
        \\    Field y: TypeId(1)
        \\    Field z: TypeId(1)
        \\  Enum Token
        \\    Variant Identifier
        \\    Variant End
        \\
    , rendered);
}

fn writeIndent(writer: *std.Io.Writer, depth: usize) !void {
    for (0..depth) |_| try writer.writeAll("  ");
}

fn writeAssignTarget(writer: *std.Io.Writer, target: AssignTarget) !void {
    switch (target) {
        .local => |id| try writer.print("LocalRef {f}", .{id}),
        .param => |id| try writer.print("ParamRef {f}", .{id}),
        .field => |field| {
            try writer.writeAll("FieldPlace ");
            switch (field.base) {
                .local => |id| try writer.print("LocalRef {f}", .{id}),
                .param => |id| try writer.print("ParamRef {f}", .{id}),
            }
            try writer.print(".{f}", .{field.field_id});
        },
    }
}

fn writePattern(writer: *std.Io.Writer, pattern: HirMatchPattern) !void {
    switch (pattern) {
        .int_literal => |text| try writer.writeAll(text),
        .bool_literal => |value| try writer.writeAll(if (value) "true" else "false"),
        .wildcard => try writer.writeByte('_'),
        .enum_variant => |enum_variant| {
            try writer.print("EnumVariant {f}::{f}", .{ enum_variant.enum_id, enum_variant.variant_id });
            if (enum_variant.bindings.len != 0) {
                try writer.writeAll("(");
                for (enum_variant.bindings, 0..) |binding, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try writer.print("{f}", .{binding.local});
                }
                try writer.writeByte(')');
            }
        },
    }
}

test "HIR debug formatting includes generic functions" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var module = HirModule.init(std.testing.allocator);
    defer module.deinit();

    const identity_name = try interner.intern("identity");
    const t_name = try interner.intern("T");
    const value_name = try interner.intern("value");
    const generic_id = try module.store.addGenericFunction(identity_name, synthetic_span);
    const function_id = module.store.getGenericFunction(generic_id).function;
    const type_param_id = types.TypeId{ .index = 3 };
    const params = try std.testing.allocator.dupe(HirTypeParam, &.{.{
        .name = t_name,
        .span = synthetic_span,
        .type_id = type_param_id,
        .constraint = .{ .text = try std.testing.allocator.dupe(u8, "Equatable<T>"), .span = synthetic_span },
    }});
    module.store.setGenericFunctionTypeParams(generic_id, params);
    module.store.setFunctionReturnType(function_id, type_param_id);
    _ = try module.store.addParam(function_id, value_name, type_param_id, synthetic_span);

    const rendered = try module.store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings(
        \\HirModule
        \\  GenericFunction identity FunctionId(0) -> TypeId(3)
        \\    TypeParams
        \\      #0 T: TypeId(3) constraint Equatable<T>
        \\    Params
        \\      ParamId(0) value: TypeId(3)
        \\
    , rendered);
}

test "marker kind registry recognizes known one-parameter markers only" {
    try std.testing.expectEqual(MarkerKind.copy, MarkerKind.fromDeclaredName("Copy", 1));
    try std.testing.expectEqual(MarkerKind.move, MarkerKind.fromDeclaredName("Move", 1));
    try std.testing.expectEqual(MarkerKind.trivial, MarkerKind.fromDeclaredName("Trivial", 1));
    try std.testing.expectEqual(MarkerKind.relocatable, MarkerKind.fromDeclaredName("Relocatable", 1));
    try std.testing.expectEqual(MarkerKind.pod, MarkerKind.fromDeclaredName("Pod", 1));
    try std.testing.expectEqual(MarkerKind.user, MarkerKind.fromDeclaredName("Copy", 2));
    try std.testing.expectEqual(MarkerKind.user, MarkerKind.fromDeclaredName("ThreadSafe", 1));
}

test "concept impl storage and lookup helpers" {
    var module = HirModule.init(std.testing.allocator);
    defer module.deinit();
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var type_store = try types.TypeStore.init(std.testing.allocator);
    defer type_store.deinit();

    const copy_name = try interner.intern("Copy");
    const vec_name = try interner.intern("Vec2");
    const concept_id = try module.store.addConcept(copy_name, true, false, synthetic_span);
    const struct_id = try module.store.addStruct(vec_name);
    const target_type = try type_store.addStructType(struct_id);

    try std.testing.expect(!module.store.hasConceptImpl(concept_id, target_type));
    const impl_id = try module.store.addConceptImpl(concept_id, target_type, &.{}, false, synthetic_span);
    try std.testing.expect(module.store.hasConceptImpl(concept_id, target_type));
    try std.testing.expectEqual(impl_id, module.store.findConceptImpl(concept_id, target_type).?);
}
