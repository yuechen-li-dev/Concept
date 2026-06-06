const std = @import("std");

const interner_module = @import("interner.zig");
const types = @import("types.zig");
pub const Interner = interner_module.Interner;
pub const SymbolId = interner_module.SymbolId;

pub const ItemId = SemanticId("ItemId");
pub const FunctionId = SemanticId("FunctionId");
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

pub const HirFunction = struct {
    item: ItemId,
    name: SymbolId,
    return_type: types.TypeId,
    params: []ParamId,
    locals: []LocalId,
    body: ?StmtId = null,
};

pub const HirLocal = struct {
    parent: FunctionId,
    name: SymbolId,
    type_id: types.TypeId,
};

pub const AssignTarget = union(enum) {
    local: LocalId,
    param: ParamId,
};

pub const HirStmt = union(enum) {
    block: []StmtId,
    return_stmt: ?ExprId,
    local_decl: struct { local: LocalId, initializer: ExprId },
    assignment: struct { target: AssignTarget, value: ExprId },
    if_stmt: struct { condition: ExprId, then_block: StmtId, else_block: ?StmtId },
    while_stmt: struct { condition: ExprId, body: StmtId },
    match_stmt: struct { scrutinee: ExprId, arms: []HirMatchArm },
};

pub const HirMatchPattern = union(enum) {
    int_literal: []const u8,
    bool_literal: bool,
    wildcard,
};

pub const HirMatchArm = struct {
    pattern: HirMatchPattern,
    body: StmtId,
};

pub const HirExpr = union(enum) {
    int_literal: []const u8,
    bool_literal: bool,
    local_ref: LocalId,
    param_ref: ParamId,
    call: struct { function: FunctionId, args: []ExprId },
    group: ExprId,
    unary: struct { op: UnaryOp, operand: ExprId },
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
    type_id: types.TypeId,
};

pub const HirStruct = struct {
    item: ItemId,
    name: SymbolId,
    fields: []FieldId,
};

pub const HirEnum = struct {
    item: ItemId,
    name: SymbolId,
    variants: []VariantId,
};

pub const HirField = struct {
    parent: StructId,
    name: SymbolId,
    type_id: types.TypeId,
};

pub const HirVariant = struct {
    parent: EnumId,
    name: SymbolId,
    payload_fields: []EnumPayloadFieldId,
};

pub const HirEnumPayloadField = struct {
    parent: VariantId,
    name: SymbolId,
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
    params: std.ArrayList(HirParam),
    locals: std.ArrayList(HirLocal),
    stmts: std.ArrayList(HirStmt),
    exprs: std.ArrayList(HirExpr),
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
            .params = std.ArrayList(HirParam).empty,
            .locals = std.ArrayList(HirLocal).empty,
            .stmts = std.ArrayList(HirStmt).empty,
            .exprs = std.ArrayList(HirExpr).empty,
            .structs = std.ArrayList(HirStruct).empty,
            .enums = std.ArrayList(HirEnum).empty,
            .fields = std.ArrayList(HirField).empty,
            .variants = std.ArrayList(HirVariant).empty,
            .enum_payload_fields = std.ArrayList(HirEnumPayloadField).empty,
        };
    }

    pub fn deinit(self: *HirStore) void {
        for (self.functions.items) |function| {
            if (function.params.len > 0) self.allocator.free(function.params);
            if (function.locals.len > 0) self.allocator.free(function.locals);
        }
        for (self.stmts.items) |stmt| {
            switch (stmt) {
                .block => |children| if (children.len > 0) self.allocator.free(children),
                .match_stmt => |match_stmt| if (match_stmt.arms.len > 0) {
                    for (match_stmt.arms) |arm| {
                        switch (arm.pattern) {
                            .int_literal => |text| self.allocator.free(text),
                            else => {},
                        }
                    }
                    self.allocator.free(match_stmt.arms);
                },
                else => {},
            }
        }
        for (self.exprs.items) |expr| {
            switch (expr) {
                .int_literal => |text| self.allocator.free(text),
                .call => |call| if (call.args.len > 0) self.allocator.free(call.args),
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
        self.exprs.deinit(self.allocator);
        self.stmts.deinit(self.allocator);
        self.locals.deinit(self.allocator);
        self.params.deinit(self.allocator);
        self.functions.deinit(self.allocator);
        self.items.deinit(self.allocator);
        self.* = undefined;
    }

    pub fn addFunction(self: *HirStore, name: SymbolId, return_type: types.TypeId) !FunctionId {
        const id = FunctionId{ .index = try nextIndex(self.functions.items.len, error.TooManyFunctions) };
        const item = try self.addItem(.{ .function = id });
        errdefer _ = self.items.pop();
        try self.functions.append(self.allocator, .{
            .item = item,
            .name = name,
            .return_type = return_type,
            .params = &.{},
            .locals = &.{},
            .body = null,
        });
        return id;
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

    pub fn addEnum(self: *HirStore, name: SymbolId) !EnumId {
        const id = EnumId{ .index = try nextIndex(self.enums.items.len, error.TooManyEnums) };
        const item = try self.addItem(.{ .enum_ = id });
        errdefer _ = self.items.pop();
        try self.enums.append(self.allocator, .{
            .item = item,
            .name = name,
            .variants = &.{},
        });
        return id;
    }

    pub fn addParam(self: *HirStore, parent: FunctionId, name: SymbolId, type_id: types.TypeId) !ParamId {
        _ = self.getFunction(parent);
        const id = ParamId{ .index = try nextIndex(self.params.items.len, error.TooManyParams) };
        try self.params.append(self.allocator, .{ .parent = parent, .name = name, .type_id = type_id });
        errdefer _ = self.params.pop();

        const function_decl = self.getFunctionMut(parent);
        function_decl.params = try appendId(self.allocator, ParamId, function_decl.params, id);
        return id;
    }

    pub fn addLocal(self: *HirStore, parent: FunctionId, name: SymbolId, type_id: types.TypeId) !LocalId {
        _ = self.getFunction(parent);
        const id = LocalId{ .index = try nextIndex(self.locals.items.len, error.TooManyLocals) };
        try self.locals.append(self.allocator, .{ .parent = parent, .name = name, .type_id = type_id });
        errdefer _ = self.locals.pop();

        const function_decl = self.getFunctionMut(parent);
        function_decl.locals = try appendId(self.allocator, LocalId, function_decl.locals, id);
        return id;
    }

    pub fn addStmt(self: *HirStore, stmt: HirStmt) !StmtId {
        const id = StmtId{ .index = try nextIndex(self.stmts.items.len, error.TooManyStmts) };
        try self.stmts.append(self.allocator, stmt);
        return id;
    }

    pub fn addExpr(self: *HirStore, expr: HirExpr) !ExprId {
        const id = ExprId{ .index = try nextIndex(self.exprs.items.len, error.TooManyExprs) };
        try self.exprs.append(self.allocator, expr);
        return id;
    }

    pub fn setFunctionBody(self: *HirStore, id: FunctionId, body: StmtId) void {
        self.getFunctionMut(id).body = body;
    }

    pub fn setFunctionReturnType(self: *HirStore, id: FunctionId, type_id: types.TypeId) void {
        self.getFunctionMut(id).return_type = type_id;
    }

    pub fn addField(self: *HirStore, parent: StructId, name: SymbolId, type_id: types.TypeId) !FieldId {
        _ = self.getStruct(parent);
        const id = FieldId{ .index = try nextIndex(self.fields.items.len, error.TooManyFields) };
        try self.fields.append(self.allocator, .{ .parent = parent, .name = name, .type_id = type_id });
        errdefer _ = self.fields.pop();

        const struct_decl = self.getStructMut(parent);
        struct_decl.fields = try appendId(self.allocator, FieldId, struct_decl.fields, id);
        return id;
    }

    pub fn addVariant(self: *HirStore, parent: EnumId, name: SymbolId) !VariantId {
        _ = self.getEnum(parent);
        const id = VariantId{ .index = try nextIndex(self.variants.items.len, error.TooManyVariants) };
        try self.variants.append(self.allocator, .{ .parent = parent, .name = name, .payload_fields = &.{} });
        errdefer _ = self.variants.pop();

        const enum_decl = self.getEnumMut(parent);
        enum_decl.variants = try appendId(self.allocator, VariantId, enum_decl.variants, id);
        return id;
    }

    pub fn addEnumPayloadField(self: *HirStore, parent: VariantId, name: SymbolId, type_id: types.TypeId) !EnumPayloadFieldId {
        _ = self.getVariant(parent);
        const id = EnumPayloadFieldId{ .index = try nextIndex(self.enum_payload_fields.items.len, error.TooManyEnumPayloadFields) };
        try self.enum_payload_fields.append(self.allocator, .{ .parent = parent, .name = name, .type_id = type_id });
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
        for (self.items.items) |item| {
            switch (item) {
                .function => |id| {
                    const function = self.getFunction(id);
                    try writer.print("  Function {s} -> {f}\n", .{ interner.text(function.name), function.return_type });
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
                    try writer.print("  Enum {s}\n", .{interner.text(enum_decl.name)});
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
        switch (self.getStmt(id).*) {
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
        switch (self.getExpr(id).*) {
            .int_literal => |text| try writer.print("Int {s}\n", .{text}),
            .bool_literal => |value| try writer.print("Bool {s}\n", .{if (value) "true" else "false"}),
            .local_ref => |local| try writer.print("LocalRef {f}\n", .{local}),
            .param_ref => |param| try writer.print("ParamRef {f}\n", .{param}),
            .call => |call| {
                try writer.print("Call {f}\n", .{call.function});
                for (call.args) |arg| try self.writeExprDebug(writer, arg, depth + 1);
            },
            .group => |inner| {
                try writer.writeAll("Group\n");
                try self.writeExprDebug(writer, inner, depth + 1);
            },
            .unary => |unary| {
                try writer.print("Unary {s}\n", .{unary.op.lexeme()});
                try self.writeExprDebug(writer, unary.operand, depth + 1);
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
    try std.testing.expectEqual(@as(usize, 0), store.structs.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.enums.items.len);
}

test "add function with interned name and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = HirStore.init(std.testing.allocator);
    defer store.deinit();

    const main_name = try interner.intern("main");
    const function_id = try store.addFunction(main_name, .{ .index = 1 });
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
    const x_id = try store.addField(vec3_id, try interner.intern("x"), .{ .index = 1 });
    const y_id = try store.addField(vec3_id, try interner.intern("y"), .{ .index = 1 });
    const z_id = try store.addField(vec3_id, try interner.intern("z"), .{ .index = 1 });
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

    const token_id = try store.addEnum(try interner.intern("Token"));
    const identifier_id = try store.addVariant(token_id, try interner.intern("Identifier"));
    const end_id = try store.addVariant(token_id, try interner.intern("End"));
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

    _ = try module.store.addFunction(try interner.intern("main"), .{ .index = 0 });
    const vec3_id = try module.store.addStruct(try interner.intern("Vec3"));
    _ = try module.store.addField(vec3_id, try interner.intern("x"), .{ .index = 1 });
    _ = try module.store.addField(vec3_id, try interner.intern("y"), .{ .index = 1 });
    _ = try module.store.addField(vec3_id, try interner.intern("z"), .{ .index = 1 });
    const token_id = try module.store.addEnum(try interner.intern("Token"));
    _ = try module.store.addVariant(token_id, try interner.intern("Identifier"));
    _ = try module.store.addVariant(token_id, try interner.intern("End"));

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
    }
}

fn writePattern(writer: *std.Io.Writer, pattern: HirMatchPattern) !void {
    switch (pattern) {
        .int_literal => |text| try writer.writeAll(text),
        .bool_literal => |value| try writer.writeAll(if (value) "true" else "false"),
        .wildcard => try writer.writeByte('_'),
    }
}
