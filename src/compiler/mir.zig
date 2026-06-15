const std = @import("std");

const hir = @import("hir.zig");
const interner_module = @import("interner.zig");
const source = @import("source.zig");
const types = @import("types.zig");

pub const Interner = interner_module.Interner;
pub const SourceSpan = source.SourceSpan;

pub const MirFunctionId = MirId("MirFunctionId");
pub const MirLocalId = MirId("MirLocalId");
pub const MirBlockId = MirId("MirBlockId");
pub const MirMachineStateId = MirId("MirMachineStateId");

// P13-M6 names the MIR machine scaffold shape without emitting executable
// machine MIR yet. Actual frame layout, resume dispatch, transitions, and
// backend codegen remain future milestones.
pub const MirMachineState = struct {
    hir_state_index: u32,
    name: hir.SymbolId,
    span: SourceSpan,
};

pub const MirMachineTransitionTarget = union(enum) {
    state: MirMachineStateId,
    unsupported_match,
    unsupported_decide,
};

fn MirId(comptime label: []const u8) type {
    return struct {
        index: u32,

        pub fn format(self: @This(), writer: *std.Io.Writer) std.Io.Writer.Error!void {
            try writer.print(label ++ "({d})", .{self.index});
        }
    };
}

pub const MirLocalKind = enum {
    param,
    user,
    temp,
};

pub const MirPlace = union(enum) {
    local: MirLocalId,
    field: struct { base: MirLocalId, field_id: hir.FieldId },

    pub fn localPlace(id: MirLocalId) MirPlace {
        return .{ .local = id };
    }

    pub fn fieldPlace(base: MirLocalId, field_id: hir.FieldId) MirPlace {
        return .{ .field = .{ .base = base, .field_id = field_id } };
    }
};

pub const MirOperand = union(enum) {
    int_literal: []const u8,
    bool_literal: bool,
    copy: MirPlace,

    pub fn intLiteral(allocator: std.mem.Allocator, text: []const u8) !MirOperand {
        return .{ .int_literal = try allocator.dupe(u8, text) };
    }

    pub fn boolLiteral(value: bool) MirOperand {
        return .{ .bool_literal = value };
    }

    pub fn copyPlace(place: MirPlace) MirOperand {
        return .{ .copy = place };
    }

    pub fn clone(self: MirOperand, allocator: std.mem.Allocator) !MirOperand {
        return switch (self) {
            .int_literal => |text| try MirOperand.intLiteral(allocator, text),
            .bool_literal => |value| MirOperand.boolLiteral(value),
            .copy => |place| MirOperand.copyPlace(place),
        };
    }

    fn deinit(self: MirOperand, allocator: std.mem.Allocator) void {
        switch (self) {
            .int_literal => |text| allocator.free(text),
            else => {},
        }
    }
};

pub const MirUnaryOp = enum {
    negate,
    logical_not,

    pub fn lexeme(self: MirUnaryOp) []const u8 {
        return switch (self) {
            .negate => "-",
            .logical_not => "!",
        };
    }
};

pub const MirBinaryOp = enum {
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

    pub fn lexeme(self: MirBinaryOp) []const u8 {
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

pub const MirStructFieldValue = struct {
    field_id: hir.FieldId,
    value: MirOperand,
};

pub const MirInterfaceCall = struct {
    receiver: MirOperand,
    interface_id: hir.InterfaceId,
    requirement_id: hir.InterfaceRequirementId,
    requirement_index: u32,
    args: []MirOperand,
    result_type: types.TypeId,
};

pub const MirCallee = union(enum) {
    internal: hir.FunctionId,
    extern_c: struct {
        function: hir.FunctionId,
        symbol: hir.SymbolId,
        result_type: types.TypeId,
    },

    pub fn functionId(self: MirCallee) hir.FunctionId {
        return switch (self) {
            .internal => |function| function,
            .extern_c => |extern_c| extern_c.function,
        };
    }
};

pub const MirCall = struct {
    callee: MirCallee,
    args: []MirOperand,

    pub fn initInternal(allocator: std.mem.Allocator, function: hir.FunctionId, args: []const MirOperand) !MirCall {
        return .{ .callee = .{ .internal = function }, .args = try cloneOperands(allocator, args) };
    }

    pub fn initExternC(allocator: std.mem.Allocator, function: hir.FunctionId, symbol: hir.SymbolId, result_type: types.TypeId, args: []const MirOperand) !MirCall {
        return .{ .callee = .{ .extern_c = .{ .function = function, .symbol = symbol, .result_type = result_type } }, .args = try cloneOperands(allocator, args) };
    }

    fn clone(self: MirCall, allocator: std.mem.Allocator) !MirCall {
        return switch (self.callee) {
            .internal => |function| MirCall.initInternal(allocator, function, self.args),
            .extern_c => |extern_c| MirCall.initExternC(allocator, extern_c.function, extern_c.symbol, extern_c.result_type, self.args),
        };
    }

    fn deinit(self: MirCall, allocator: std.mem.Allocator) void {
        deinitOperands(allocator, self.args);
        if (self.args.len > 0) allocator.free(self.args);
    }
};

pub const MirRvalue = union(enum) {
    use: MirOperand,
    move: MirPlace,
    manual_init_assume: MirOperand,
    unary: struct {
        op: MirUnaryOp,
        operand: MirOperand,
    },
    address_of: MirPlace,
    deref: MirOperand,
    dyn_coerce: struct {
        source: MirPlace,
        interface_id: hir.InterfaceId,
        impl_id: hir.InterfaceImplId,
        result_type: types.TypeId,
    },
    binary: struct {
        op: MirBinaryOp,
        left: MirOperand,
        right: MirOperand,
    },
    call: MirCall,
    interface_call: MirInterfaceCall,
    arena_alloc: struct {
        arena_operand: MirOperand,
        allocated_type: types.TypeId,
        result_type: types.TypeId,
    },
    enum_constructor: struct {
        enum_id: hir.EnumId,
        variant_id: hir.VariantId,
        args: []MirOperand,
    },
    struct_constructor: struct {
        struct_id: hir.StructId,
        fields: []MirStructFieldValue,
    },
    array_constructor: []MirOperand,
    enum_tag: MirOperand,
    enum_payload_field: struct {
        enum_operand: MirOperand,
        payload_field: hir.EnumPayloadFieldId,
    },
    field_access: struct {
        receiver: MirOperand,
        field_id: hir.FieldId,
    },
    array_index: struct {
        base: MirOperand,
        index: MirOperand,
        length: u64,
        result_type: types.TypeId,
    },
    machine_construct: struct {
        machine: hir.MachineId,
        args: []MirOperand,
    },
    machine_complete: MirOperand,
    machine_result: MirOperand,
    machine_state: MirOperand,

    pub fn use_(operand: MirOperand) MirRvalue {
        return .{ .use = operand };
    }

    pub fn movePlace(place: MirPlace) MirRvalue {
        return .{ .move = place };
    }

    pub fn manualInitAssume(operand: MirOperand) MirRvalue {
        return .{ .manual_init_assume = operand };
    }

    pub fn unaryOp(op: MirUnaryOp, operand: MirOperand) MirRvalue {
        return .{ .unary = .{ .op = op, .operand = operand } };
    }

    pub fn addressOf(place: MirPlace) MirRvalue {
        return .{ .address_of = place };
    }

    pub fn dereference(operand: MirOperand) MirRvalue {
        return .{ .deref = operand };
    }

    pub fn dynCoerce(source_place: MirPlace, interface_id: hir.InterfaceId, impl_id: hir.InterfaceImplId, result_type: types.TypeId) MirRvalue {
        return .{ .dyn_coerce = .{ .source = source_place, .interface_id = interface_id, .impl_id = impl_id, .result_type = result_type } };
    }

    pub fn binaryOp(op: MirBinaryOp, left: MirOperand, right: MirOperand) MirRvalue {
        return .{ .binary = .{ .op = op, .left = left, .right = right } };
    }

    pub fn callFunction(allocator: std.mem.Allocator, function: hir.FunctionId, args: []const MirOperand) !MirRvalue {
        return .{ .call = try MirCall.initInternal(allocator, function, args) };
    }

    pub fn callExternC(allocator: std.mem.Allocator, function: hir.FunctionId, symbol: hir.SymbolId, result_type: types.TypeId, args: []const MirOperand) !MirRvalue {
        return .{ .call = try MirCall.initExternC(allocator, function, symbol, result_type, args) };
    }

    pub fn interfaceCall(allocator: std.mem.Allocator, call: MirInterfaceCall) !MirRvalue {
        const owned_args = try cloneOperands(allocator, call.args);
        errdefer {
            deinitOperands(allocator, owned_args);
            if (owned_args.len > 0) allocator.free(owned_args);
        }
        const receiver = try call.receiver.clone(allocator);
        return .{ .interface_call = .{
            .receiver = receiver,
            .interface_id = call.interface_id,
            .requirement_id = call.requirement_id,
            .requirement_index = call.requirement_index,
            .args = owned_args,
            .result_type = call.result_type,
        } };
    }

    pub fn arenaAlloc(arena_operand: MirOperand, allocated_type: types.TypeId, result_type: types.TypeId) MirRvalue {
        return .{ .arena_alloc = .{ .arena_operand = arena_operand, .allocated_type = allocated_type, .result_type = result_type } };
    }

    pub fn enumConstructor(allocator: std.mem.Allocator, enum_id: hir.EnumId, variant_id: hir.VariantId, args: []const MirOperand) !MirRvalue {
        const owned_args = try cloneOperands(allocator, args);
        return .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = variant_id, .args = owned_args } };
    }

    pub fn structConstructor(allocator: std.mem.Allocator, struct_id: hir.StructId, fields: []const MirStructFieldValue) !MirRvalue {
        const owned_fields = try allocator.alloc(MirStructFieldValue, fields.len);
        errdefer allocator.free(owned_fields);
        for (fields, 0..) |field, index| {
            owned_fields[index] = .{ .field_id = field.field_id, .value = try field.value.clone(allocator) };
        }
        return .{ .struct_constructor = .{ .struct_id = struct_id, .fields = owned_fields } };
    }

    pub fn arrayConstructor(allocator: std.mem.Allocator, elements: []const MirOperand) !MirRvalue {
        return .{ .array_constructor = try cloneOperands(allocator, elements) };
    }

    pub fn enumTag(operand: MirOperand) MirRvalue {
        return .{ .enum_tag = operand };
    }

    pub fn enumPayloadField(operand: MirOperand, payload_field: hir.EnumPayloadFieldId) MirRvalue {
        return .{ .enum_payload_field = .{ .enum_operand = operand, .payload_field = payload_field } };
    }

    pub fn fieldAccess(receiver: MirOperand, field_id: hir.FieldId) MirRvalue {
        return .{ .field_access = .{ .receiver = receiver, .field_id = field_id } };
    }

    pub fn arrayIndex(base: MirOperand, index: MirOperand, length: u64, result_type: types.TypeId) MirRvalue {
        return .{ .array_index = .{ .base = base, .index = index, .length = length, .result_type = result_type } };
    }

    fn clone(self: MirRvalue, allocator: std.mem.Allocator) !MirRvalue {
        return switch (self) {
            .use => |operand| MirRvalue.use_(try operand.clone(allocator)),
            .move => |place| MirRvalue.movePlace(place),
            .manual_init_assume => |operand| MirRvalue.manualInitAssume(try operand.clone(allocator)),
            .unary => |unary_rvalue| MirRvalue.unaryOp(unary_rvalue.op, try unary_rvalue.operand.clone(allocator)),
            .address_of => |place| MirRvalue.addressOf(place),
            .deref => |operand| MirRvalue.dereference(try operand.clone(allocator)),
            .dyn_coerce => |coerce| MirRvalue.dynCoerce(coerce.source, coerce.interface_id, coerce.impl_id, coerce.result_type),
            .binary => |binary_rvalue| MirRvalue.binaryOp(
                binary_rvalue.op,
                try binary_rvalue.left.clone(allocator),
                try binary_rvalue.right.clone(allocator),
            ),
            .call => |call_rvalue| .{ .call = try call_rvalue.clone(allocator) },
            .interface_call => |call| try MirRvalue.interfaceCall(allocator, call),
            .arena_alloc => |arena_alloc| MirRvalue.arenaAlloc(try arena_alloc.arena_operand.clone(allocator), arena_alloc.allocated_type, arena_alloc.result_type),
            .enum_constructor => |constructor| try MirRvalue.enumConstructor(allocator, constructor.enum_id, constructor.variant_id, constructor.args),
            .struct_constructor => |constructor| try MirRvalue.structConstructor(allocator, constructor.struct_id, constructor.fields),
            .array_constructor => |elements| try MirRvalue.arrayConstructor(allocator, elements),
            .enum_tag => |operand| MirRvalue.enumTag(try operand.clone(allocator)),
            .enum_payload_field => |payload| MirRvalue.enumPayloadField(try payload.enum_operand.clone(allocator), payload.payload_field),
            .field_access => |field_access| MirRvalue.fieldAccess(try field_access.receiver.clone(allocator), field_access.field_id),
            .array_index => |array_index| MirRvalue.arrayIndex(try array_index.base.clone(allocator), try array_index.index.clone(allocator), array_index.length, array_index.result_type),
            .machine_construct => |construct| .{ .machine_construct = .{ .machine = construct.machine, .args = try cloneOperands(allocator, construct.args) } },
            .machine_complete => |operand| .{ .machine_complete = try operand.clone(allocator) },
            .machine_result => |operand| .{ .machine_result = try operand.clone(allocator) },
            .machine_state => |operand| .{ .machine_state = try operand.clone(allocator) },
        };
    }

    fn deinit(self: MirRvalue, allocator: std.mem.Allocator) void {
        switch (self) {
            .use => |operand| operand.deinit(allocator),
            .move => {},
            .manual_init_assume => |operand| operand.deinit(allocator),
            .unary => |unary_rvalue| unary_rvalue.operand.deinit(allocator),
            .address_of => {},
            .deref => |operand| operand.deinit(allocator),
            .dyn_coerce => {},
            .binary => |binary_rvalue| {
                binary_rvalue.left.deinit(allocator);
                binary_rvalue.right.deinit(allocator);
            },
            .call => |call_rvalue| call_rvalue.deinit(allocator),
            .interface_call => |call| {
                call.receiver.deinit(allocator);
                deinitOperands(allocator, call.args);
                if (call.args.len > 0) allocator.free(call.args);
            },
            .arena_alloc => |arena_alloc| arena_alloc.arena_operand.deinit(allocator),
            .enum_constructor => |constructor| {
                deinitOperands(allocator, constructor.args);
                if (constructor.args.len > 0) allocator.free(constructor.args);
            },
            .struct_constructor => |constructor| {
                for (constructor.fields) |field| field.value.deinit(allocator);
                if (constructor.fields.len > 0) allocator.free(constructor.fields);
            },
            .array_constructor => |elements| {
                deinitOperands(allocator, elements);
                if (elements.len > 0) allocator.free(elements);
            },
            .enum_tag => |operand| operand.deinit(allocator),
            .enum_payload_field => |payload| payload.enum_operand.deinit(allocator),
            .field_access => |field_access| field_access.receiver.deinit(allocator),
            .array_index => |array_index| {
                array_index.base.deinit(allocator);
                array_index.index.deinit(allocator);
            },
            .machine_construct => |construct| {
                deinitOperands(allocator, construct.args);
                if (construct.args.len > 0) allocator.free(construct.args);
            },
            .machine_complete => |operand| operand.deinit(allocator),
            .machine_result => |operand| operand.deinit(allocator),
            .machine_state => |operand| operand.deinit(allocator),
        }
    }
};

pub const MirStatement = struct {
    span: ?SourceSpan,
    kind: MirStatementKind,

    fn clone(self: MirStatement, allocator: std.mem.Allocator) !MirStatement {
        return .{ .span = self.span, .kind = try self.kind.clone(allocator) };
    }

    fn deinit(self: MirStatement, allocator: std.mem.Allocator) void {
        self.kind.deinit(allocator);
    }
};

pub const MirStatementKind = union(enum) {
    assign: struct {
        place: MirPlace,
        rvalue: MirRvalue,
    },
    call: MirCall,
    interface_call: MirInterfaceCall,
    drop: struct {
        place: MirPlace,
        function: hir.FunctionId,
    },
    arena_reset: MirOperand,
    arena_destroy: MirOperand,
    machine_step: MirOperand,
    panic: struct {
        reason: []const u8,
        reason_span: SourceSpan,
    },
    assert_stmt: struct {
        condition: MirOperand,
        reason: []const u8,
        condition_span: SourceSpan,
        reason_span: SourceSpan,
    },

    pub fn assignTo(place: MirPlace, rvalue: MirRvalue) MirStatementKind {
        return .{ .assign = .{ .place = place, .rvalue = rvalue } };
    }

    pub fn callFunction(allocator: std.mem.Allocator, function: hir.FunctionId, args: []const MirOperand) !MirStatementKind {
        return .{ .call = try MirCall.initInternal(allocator, function, args) };
    }

    pub fn callExternC(allocator: std.mem.Allocator, function: hir.FunctionId, symbol: hir.SymbolId, result_type: types.TypeId, args: []const MirOperand) !MirStatementKind {
        return .{ .call = try MirCall.initExternC(allocator, function, symbol, result_type, args) };
    }

    pub fn interfaceCall(allocator: std.mem.Allocator, call: MirInterfaceCall) !MirStatementKind {
        const owned_args = try cloneOperands(allocator, call.args);
        errdefer {
            deinitOperands(allocator, owned_args);
            if (owned_args.len > 0) allocator.free(owned_args);
        }
        const receiver = try call.receiver.clone(allocator);
        return .{ .interface_call = .{
            .receiver = receiver,
            .interface_id = call.interface_id,
            .requirement_id = call.requirement_id,
            .requirement_index = call.requirement_index,
            .args = owned_args,
            .result_type = call.result_type,
        } };
    }

    pub fn dropPlace(place: MirPlace, function: hir.FunctionId) MirStatementKind {
        return .{ .drop = .{ .place = place, .function = function } };
    }

    pub fn arenaReset(arena_operand: MirOperand) MirStatementKind {
        return .{ .arena_reset = arena_operand };
    }

    pub fn arenaDestroy(arena_operand: MirOperand) MirStatementKind {
        return .{ .arena_destroy = arena_operand };
    }

    pub fn panicStmt(allocator: std.mem.Allocator, reason: []const u8, reason_span: SourceSpan) !MirStatementKind {
        return .{ .panic = .{ .reason = try allocator.dupe(u8, reason), .reason_span = reason_span } };
    }

    pub fn assertStmt(allocator: std.mem.Allocator, condition: MirOperand, reason: []const u8, condition_span: SourceSpan, reason_span: SourceSpan) !MirStatementKind {
        return .{ .assert_stmt = .{
            .condition = try condition.clone(allocator),
            .reason = try allocator.dupe(u8, reason),
            .condition_span = condition_span,
            .reason_span = reason_span,
        } };
    }

    fn clone(self: MirStatementKind, allocator: std.mem.Allocator) !MirStatementKind {
        return switch (self) {
            .assign => |assignment| MirStatementKind.assignTo(assignment.place, try assignment.rvalue.clone(allocator)),
            .call => |call| .{ .call = try call.clone(allocator) },
            .interface_call => |call| try MirStatementKind.interfaceCall(allocator, call),
            .drop => |drop| MirStatementKind.dropPlace(drop.place, drop.function),
            .arena_reset => |arena_operand| MirStatementKind.arenaReset(try arena_operand.clone(allocator)),
            .arena_destroy => |arena_operand| MirStatementKind.arenaDestroy(try arena_operand.clone(allocator)),
            .machine_step => |machine_operand| .{ .machine_step = try machine_operand.clone(allocator) },
            .panic => |panic_stmt| try MirStatementKind.panicStmt(allocator, panic_stmt.reason, panic_stmt.reason_span),
            .assert_stmt => |assert_stmt| try MirStatementKind.assertStmt(allocator, assert_stmt.condition, assert_stmt.reason, assert_stmt.condition_span, assert_stmt.reason_span),
        };
    }

    fn deinit(self: MirStatementKind, allocator: std.mem.Allocator) void {
        switch (self) {
            .assign => |assignment| assignment.rvalue.deinit(allocator),
            .call => |call| call.deinit(allocator),
            .interface_call => |call| {
                call.receiver.deinit(allocator);
                deinitOperands(allocator, call.args);
                if (call.args.len > 0) allocator.free(call.args);
            },
            .drop => {},
            .arena_reset => |arena_operand| arena_operand.deinit(allocator),
            .arena_destroy => |arena_operand| arena_operand.deinit(allocator),
            .machine_step => |machine_operand| machine_operand.deinit(allocator),
            .panic => |panic_stmt| allocator.free(panic_stmt.reason),
            .assert_stmt => |assert_stmt| {
                assert_stmt.condition.deinit(allocator);
                allocator.free(assert_stmt.reason);
            },
        }
    }
};

pub const MirSwitchIntCase = struct {
    value: []const u8,
    target: MirBlockId,

    fn clone(self: MirSwitchIntCase, allocator: std.mem.Allocator) !MirSwitchIntCase {
        return .{ .value = try allocator.dupe(u8, self.value), .target = self.target };
    }

    fn deinit(self: MirSwitchIntCase, allocator: std.mem.Allocator) void {
        allocator.free(self.value);
    }
};

pub const MirTerminator = struct {
    span: ?SourceSpan,
    kind: MirTerminatorKind,

    fn clone(self: MirTerminator, allocator: std.mem.Allocator) !MirTerminator {
        return .{ .span = self.span, .kind = try self.kind.clone(allocator) };
    }

    fn deinit(self: MirTerminator, allocator: std.mem.Allocator) void {
        self.kind.deinit(allocator);
    }
};

pub const MirTerminatorKind = union(enum) {
    goto: MirBlockId,
    return_: ?MirOperand,
    switch_bool: struct {
        discriminant: MirOperand,
        true_target: MirBlockId,
        false_target: MirBlockId,
    },
    switch_int: struct {
        discriminant: MirOperand,
        cases: []MirSwitchIntCase,
        default_target: MirBlockId,
    },
    @"unreachable",

    pub fn gotoBlock(target: MirBlockId) MirTerminatorKind {
        return .{ .goto = target };
    }

    pub fn returnValue(operand: ?MirOperand) MirTerminatorKind {
        return .{ .return_ = operand };
    }

    pub fn switchBool(discriminant: MirOperand, true_target: MirBlockId, false_target: MirBlockId) MirTerminatorKind {
        return .{ .switch_bool = .{ .discriminant = discriminant, .true_target = true_target, .false_target = false_target } };
    }

    pub fn switchInt(
        allocator: std.mem.Allocator,
        discriminant: MirOperand,
        cases: []const MirSwitchIntCase,
        default_target: MirBlockId,
    ) !MirTerminatorKind {
        errdefer discriminant.deinit(allocator);
        const owned_cases = try cloneSwitchIntCases(allocator, cases);
        return .{ .switch_int = .{ .discriminant = discriminant, .cases = owned_cases, .default_target = default_target } };
    }

    fn clone(self: MirTerminatorKind, allocator: std.mem.Allocator) !MirTerminatorKind {
        return switch (self) {
            .goto => |target| MirTerminatorKind.gotoBlock(target),
            .return_ => |operand| MirTerminatorKind.returnValue(if (operand) |value| try value.clone(allocator) else null),
            .switch_bool => |switch_bool| MirTerminatorKind.switchBool(
                try switch_bool.discriminant.clone(allocator),
                switch_bool.true_target,
                switch_bool.false_target,
            ),
            .switch_int => |switch_int| try MirTerminatorKind.switchInt(
                allocator,
                try switch_int.discriminant.clone(allocator),
                switch_int.cases,
                switch_int.default_target,
            ),
            .@"unreachable" => .@"unreachable",
        };
    }

    fn deinit(self: MirTerminatorKind, allocator: std.mem.Allocator) void {
        switch (self) {
            .return_ => |operand| if (operand) |value| value.deinit(allocator),
            .switch_bool => |switch_bool| switch_bool.discriminant.deinit(allocator),
            .switch_int => |switch_int| {
                switch_int.discriminant.deinit(allocator);
                for (switch_int.cases) |case| case.deinit(allocator);
                if (switch_int.cases.len > 0) allocator.free(switch_int.cases);
            },
            else => {},
        }
    }
};

pub const MirModule = struct {
    store: MirStore,

    pub fn init(allocator: std.mem.Allocator) MirModule {
        return .{ .store = MirStore.init(allocator) };
    }

    pub fn deinit(self: *MirModule) void {
        self.store.deinit();
        self.* = undefined;
    }
};

pub const MirFunctionLinkage = union(enum) {
    internal,
    export_c: struct { symbol: hir.SymbolId },
};

pub const MirFunction = struct {
    hir_function: hir.FunctionId,
    name: hir.SymbolId,
    return_type: types.TypeId,
    linkage: MirFunctionLinkage = .internal,
    params: []MirLocalId,
    locals: []MirLocalId,
    blocks: []MirBlockId,
    source_span: ?SourceSpan,
};

pub const MirLocal = struct {
    parent: MirFunctionId,
    name: ?hir.SymbolId,
    kind: MirLocalKind,
    type_id: types.TypeId,
    source_span: ?SourceSpan,
};

pub const MirBlock = struct {
    parent: MirFunctionId,
    statements: []MirStatement,
    terminator: ?MirTerminator,
    source_span: ?SourceSpan,
};

pub const MirStore = struct {
    allocator: std.mem.Allocator,
    functions: std.ArrayList(MirFunction),
    locals: std.ArrayList(MirLocal),
    blocks: std.ArrayList(MirBlock),

    pub fn init(allocator: std.mem.Allocator) MirStore {
        return .{
            .allocator = allocator,
            .functions = std.ArrayList(MirFunction).empty,
            .locals = std.ArrayList(MirLocal).empty,
            .blocks = std.ArrayList(MirBlock).empty,
        };
    }

    pub fn deinit(self: *MirStore) void {
        for (self.functions.items) |function| {
            if (function.params.len > 0) self.allocator.free(function.params);
            if (function.locals.len > 0) self.allocator.free(function.locals);
            if (function.blocks.len > 0) self.allocator.free(function.blocks);
        }
        for (self.blocks.items) |block| {
            for (block.statements) |statement| statement.deinit(self.allocator);
            if (block.statements.len > 0) self.allocator.free(block.statements);
            if (block.terminator) |terminator| terminator.deinit(self.allocator);
        }

        self.blocks.deinit(self.allocator);
        self.locals.deinit(self.allocator);
        self.functions.deinit(self.allocator);
        self.* = undefined;
    }

    pub fn addFunction(
        self: *MirStore,
        hir_function: hir.FunctionId,
        name: hir.SymbolId,
        return_type: types.TypeId,
        source_span: ?SourceSpan,
    ) !MirFunctionId {
        return self.addFunctionWithLinkage(hir_function, name, return_type, source_span, .internal);
    }

    pub fn addFunctionWithLinkage(
        self: *MirStore,
        hir_function: hir.FunctionId,
        name: hir.SymbolId,
        return_type: types.TypeId,
        source_span: ?SourceSpan,
        linkage: MirFunctionLinkage,
    ) !MirFunctionId {
        const id = MirFunctionId{ .index = try nextIndex(self.functions.items.len, error.TooManyMirFunctions) };
        try self.functions.append(self.allocator, .{
            .hir_function = hir_function,
            .name = name,
            .return_type = return_type,
            .linkage = linkage,
            .params = &.{},
            .locals = &.{},
            .blocks = &.{},
            .source_span = source_span,
        });
        return id;
    }

    pub fn addLocal(
        self: *MirStore,
        parent: MirFunctionId,
        name: ?hir.SymbolId,
        kind: MirLocalKind,
        type_id: types.TypeId,
        source_span: ?SourceSpan,
    ) !MirLocalId {
        _ = self.getFunction(parent);
        const id = MirLocalId{ .index = try nextIndex(self.locals.items.len, error.TooManyMirLocals) };
        try self.locals.append(self.allocator, .{
            .parent = parent,
            .name = name,
            .kind = kind,
            .type_id = type_id,
            .source_span = source_span,
        });
        errdefer _ = self.locals.pop();

        const function = self.getFunctionMut(parent);
        switch (kind) {
            .param => function.params = try appendId(self.allocator, MirLocalId, function.params, id),
            .user, .temp => function.locals = try appendId(self.allocator, MirLocalId, function.locals, id),
        }
        return id;
    }

    pub fn addBlock(
        self: *MirStore,
        parent: MirFunctionId,
        source_span: ?SourceSpan,
    ) !MirBlockId {
        _ = self.getFunction(parent);
        const id = MirBlockId{ .index = try nextIndex(self.blocks.items.len, error.TooManyMirBlocks) };
        try self.blocks.append(self.allocator, .{
            .parent = parent,
            .statements = &.{},
            .terminator = null,
            .source_span = source_span,
        });
        errdefer _ = self.blocks.pop();

        const function = self.getFunctionMut(parent);
        function.blocks = try appendId(self.allocator, MirBlockId, function.blocks, id);
        return id;
    }

    pub fn appendStatement(self: *MirStore, block_id: MirBlockId, statement: MirStatement) !void {
        const block = self.getBlockMut(block_id);
        errdefer statement.deinit(self.allocator);
        block.statements = try appendStatementValue(self.allocator, block.statements, statement);
    }

    pub fn setTerminator(self: *MirStore, block_id: MirBlockId, terminator: MirTerminator) !void {
        const block = self.getBlockMut(block_id);
        if (block.terminator != null) return error.MirBlockAlreadyTerminated;
        block.terminator = terminator;
    }

    pub fn getFunction(self: *const MirStore, id: MirFunctionId) *const MirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    pub fn getFunctionMut(self: *MirStore, id: MirFunctionId) *MirFunction {
        const index: usize = id.index;
        std.debug.assert(index < self.functions.items.len);
        return &self.functions.items[index];
    }

    pub fn getLocal(self: *const MirStore, id: MirLocalId) *const MirLocal {
        const index: usize = id.index;
        std.debug.assert(index < self.locals.items.len);
        return &self.locals.items[index];
    }

    pub fn getBlock(self: *const MirStore, id: MirBlockId) *const MirBlock {
        const index: usize = id.index;
        std.debug.assert(index < self.blocks.items.len);
        return &self.blocks.items[index];
    }

    pub fn getBlockMut(self: *MirStore, id: MirBlockId) *MirBlock {
        const index: usize = id.index;
        std.debug.assert(index < self.blocks.items.len);
        return &self.blocks.items[index];
    }

    pub fn debugString(self: *const MirStore, allocator: std.mem.Allocator, interner: Interner) ![]u8 {
        var buffer: std.Io.Writer.Allocating = .init(allocator);
        errdefer buffer.deinit();
        const writer = &buffer.writer;

        try writer.writeAll("MirModule\n");
        for (self.functions.items) |function| {
            try writer.print("  Function {s} -> {f}\n", .{ interner.text(function.name), function.return_type });
            if (function.params.len != 0) {
                try writer.writeAll("    Params\n");
                for (function.params) |local_id| {
                    try self.writeLocalDebug(writer, local_id, interner);
                }
            }
            if (function.locals.len != 0) {
                try writer.writeAll("    Locals\n");
                for (function.locals) |local_id| {
                    try self.writeLocalDebug(writer, local_id, interner);
                }
            }
            if (function.blocks.len != 0) {
                try writer.writeAll("    Blocks\n");
                for (function.blocks) |block_id| {
                    try writer.print("      {f}\n", .{block_id});
                    const block = self.getBlock(block_id);
                    for (block.statements) |statement| {
                        try writer.writeAll("        ");
                        try self.writeStatementDebug(writer, statement);
                        try writer.writeByte('\n');
                    }
                    if (block.terminator) |terminator| {
                        try writer.writeAll("        ");
                        try self.writeTerminatorDebug(writer, terminator);
                        try writer.writeByte('\n');
                    }
                }
            }
        }

        return try buffer.toOwnedSlice();
    }

    fn writeLocalDebug(self: *const MirStore, writer: *std.Io.Writer, id: MirLocalId, interner: Interner) !void {
        const local = self.getLocal(id);
        const name = if (local.name) |symbol| interner.text(symbol) else "<temp>";
        try writer.print("      {f} {s} {s}: {f}\n", .{ id, @tagName(local.kind), name, local.type_id });
    }

    fn writeStatementDebug(self: *const MirStore, writer: *std.Io.Writer, statement: MirStatement) !void {
        _ = self;
        switch (statement.kind) {
            .assign => |assignment| {
                try writePlaceDebug(writer, assignment.place);
                try writer.writeAll(" = ");
                try writeRvalueDebug(writer, assignment.rvalue);
            },
            .call => |call| {
                try writer.writeAll("Statement");
                try writeCallDebug(writer, call);
            },
            .interface_call => |call| {
                try writer.print("InterfaceCall {f} {f} #{d} -> {f}(", .{ call.interface_id, call.requirement_id, call.requirement_index, call.result_type });
                try writeOperandDebug(writer, call.receiver);
                for (call.args) |arg| {
                    try writer.writeAll(", ");
                    try writeOperandDebug(writer, arg);
                }
                try writer.writeByte(')');
            },
            .drop => |drop| {
                try writer.writeAll("Drop ");
                try writePlaceDebug(writer, drop.place);
                try writer.print(" via {f}", .{drop.function});
            },
            .arena_reset => |arena_operand| {
                try writer.writeAll("ArenaReset ");
                try writeOperandDebug(writer, arena_operand);
            },
            .arena_destroy => |arena_operand| {
                try writer.writeAll("ArenaDestroy ");
                try writeOperandDebug(writer, arena_operand);
            },
            .machine_step => |machine_operand| {
                try writer.writeAll("MachineStep ");
                try writeOperandDebug(writer, machine_operand);
            },
            .panic => |panic_stmt| try writer.print("Panic \"{s}\"", .{panic_stmt.reason}),
            .assert_stmt => |assert_stmt| {
                try writer.print("Assert \"{s}\" ", .{assert_stmt.reason});
                try writeOperandDebug(writer, assert_stmt.condition);
            },
        }
    }

    fn writeTerminatorDebug(self: *const MirStore, writer: *std.Io.Writer, terminator: MirTerminator) !void {
        _ = self;
        switch (terminator.kind) {
            .goto => |target| try writer.print("Goto {f}", .{target}),
            .return_ => |operand| {
                try writer.writeAll("Return");
                if (operand) |value| {
                    try writer.writeByte(' ');
                    try writeOperandDebug(writer, value);
                }
            },
            .switch_bool => |switch_bool| {
                try writer.writeAll("SwitchBool ");
                try writeOperandDebug(writer, switch_bool.discriminant);
                try writer.print(" true: {f}, false: {f}", .{ switch_bool.true_target, switch_bool.false_target });
            },
            .switch_int => |switch_int| {
                try writer.writeAll("SwitchInt ");
                try writeOperandDebug(writer, switch_int.discriminant);
                try writer.writeAll(" [");
                for (switch_int.cases, 0..) |case, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try writer.print("{s}: {f}", .{ case.value, case.target });
                }
                try writer.print("] default: {f}", .{switch_int.default_target});
            },
            .@"unreachable" => try writer.writeAll("Unreachable"),
        }
    }
};

fn writePlaceDebug(writer: *std.Io.Writer, place: MirPlace) !void {
    switch (place) {
        .local => |id| try writer.print("{f}", .{id}),
        .field => |field| try writer.print("Field({f}, {f})", .{ field.base, field.field_id }),
    }
}

fn writeOperandDebug(writer: *std.Io.Writer, operand: MirOperand) !void {
    switch (operand) {
        .int_literal => |text| try writer.print("Int {s}", .{text}),
        .bool_literal => |value| try writer.print("Bool {}", .{value}),
        .copy => |place| {
            try writer.writeAll("Copy(");
            try writePlaceDebug(writer, place);
            try writer.writeByte(')');
        },
    }
}

fn writeRvalueDebug(writer: *std.Io.Writer, rvalue: MirRvalue) !void {
    switch (rvalue) {
        .use => |operand| {
            try writer.writeAll("Use(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
        .move => |place| {
            try writer.writeAll("Move(");
            try writePlaceDebug(writer, place);
            try writer.writeByte(')');
        },
        .array_index => |array_index| {
            try writer.writeAll("ArrayIndex(");
            try writeOperandDebug(writer, array_index.base);
            try writer.writeAll(", ");
            try writeOperandDebug(writer, array_index.index);
            try writer.writeByte(')');
        },
        .manual_init_assume => |operand| {
            try writer.writeAll("ManualInitAssume(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
        .unary => |unary_rvalue| {
            try writer.print("Unary {s} ", .{unary_rvalue.op.lexeme()});
            try writeOperandDebug(writer, unary_rvalue.operand);
        },
        .address_of => |place| {
            try writer.writeAll("AddressOf(");
            try writePlaceDebug(writer, place);
            try writer.writeByte(')');
        },
        .deref => |operand| {
            try writer.writeAll("Deref(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
        .dyn_coerce => |coerce| {
            try writer.print("DynCoerce {f} via {f} -> {f} ", .{ coerce.interface_id, coerce.impl_id, coerce.result_type });
            try writePlaceDebug(writer, coerce.source);
        },
        .binary => |binary_rvalue| {
            try writer.print("Binary {s} ", .{binary_rvalue.op.lexeme()});
            try writeOperandDebug(writer, binary_rvalue.left);
            try writer.writeAll(", ");
            try writeOperandDebug(writer, binary_rvalue.right);
        },
        .call => |call_rvalue| try writeCallDebug(writer, call_rvalue),
        .interface_call => |call| {
            try writer.print("InterfaceCall {f} {f} #{d} -> {f}(", .{ call.interface_id, call.requirement_id, call.requirement_index, call.result_type });
            try writeOperandDebug(writer, call.receiver);
            for (call.args) |arg| {
                try writer.writeAll(", ");
                try writeOperandDebug(writer, arg);
            }
            try writer.writeByte(')');
        },
        .arena_alloc => |arena_alloc| {
            try writer.print("ArenaAlloc {f} -> {f}(", .{ arena_alloc.allocated_type, arena_alloc.result_type });
            try writeOperandDebug(writer, arena_alloc.arena_operand);
            try writer.writeByte(')');
        },
        .enum_constructor => |constructor| {
            try writer.print("EnumConstructor {f}::{f}(", .{ constructor.enum_id, constructor.variant_id });
            for (constructor.args, 0..) |arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try writeOperandDebug(writer, arg);
            }
            try writer.writeByte(')');
        },
        .struct_constructor => |constructor| {
            try writer.print("StructConstructor {f}(", .{constructor.struct_id});
            for (constructor.fields, 0..) |field, index| {
                if (index != 0) try writer.writeAll(", ");
                try writer.print("{f}=", .{field.field_id});
                try writeOperandDebug(writer, field.value);
            }
            try writer.writeByte(')');
        },
        .array_constructor => |elements| {
            try writer.writeAll("ArrayConstructor(");
            for (elements, 0..) |element, index| {
                if (index != 0) try writer.writeAll(", ");
                try writeOperandDebug(writer, element);
            }
            try writer.writeByte(')');
        },
        .enum_tag => |operand| {
            try writer.writeAll("EnumTag(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
        .enum_payload_field => |payload| {
            try writer.writeAll("EnumPayloadField(");
            try writeOperandDebug(writer, payload.enum_operand);
            try writer.print(", {f})", .{payload.payload_field});
        },
        .field_access => |field_access| {
            try writer.writeAll("FieldAccess(");
            try writeOperandDebug(writer, field_access.receiver);
            try writer.print(", {f})", .{field_access.field_id});
        },
        .machine_construct => |construct| {
            try writer.print("MachineConstruct {f}(", .{construct.machine});
            for (construct.args, 0..) |arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try writeOperandDebug(writer, arg);
            }
            try writer.writeByte(')');
        },
        .machine_complete => |operand| {
            try writer.writeAll("MachineComplete(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
        .machine_result => |operand| {
            try writer.writeAll("MachineResult(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
        .machine_state => |operand| {
            try writer.writeAll("MachineState(");
            try writeOperandDebug(writer, operand);
            try writer.writeByte(')');
        },
    }
}

fn writeCallDebug(writer: *std.Io.Writer, call: MirCall) !void {
    switch (call.callee) {
        .internal => |function| try writer.print("Call internal {f}(", .{function}),
        .extern_c => |extern_c| try writer.print("Call extern C {f} symbol {f} -> {f}(", .{ extern_c.function, extern_c.symbol, extern_c.result_type }),
    }
    for (call.args, 0..) |arg, index| {
        if (index != 0) try writer.writeAll(", ");
        try writeOperandDebug(writer, arg);
    }
    try writer.writeByte(')');
}

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

fn appendStatementValue(allocator: std.mem.Allocator, existing: []MirStatement, statement: MirStatement) ![]MirStatement {
    const expanded = try allocator.alloc(MirStatement, existing.len + 1);
    @memcpy(expanded[0..existing.len], existing);
    expanded[existing.len] = statement;
    if (existing.len > 0) allocator.free(existing);
    return expanded;
}

fn cloneOperands(allocator: std.mem.Allocator, operands: []const MirOperand) ![]MirOperand {
    const cloned = try allocator.alloc(MirOperand, operands.len);
    errdefer allocator.free(cloned);
    var initialized: usize = 0;
    errdefer for (cloned[0..initialized]) |operand| operand.deinit(allocator);

    for (operands) |operand| {
        cloned[initialized] = try operand.clone(allocator);
        initialized += 1;
    }
    return cloned;
}

fn deinitOperands(allocator: std.mem.Allocator, operands: []MirOperand) void {
    for (operands) |operand| operand.deinit(allocator);
}

fn cloneSwitchIntCases(allocator: std.mem.Allocator, cases: []const MirSwitchIntCase) ![]MirSwitchIntCase {
    const cloned = try allocator.alloc(MirSwitchIntCase, cases.len);
    errdefer allocator.free(cloned);
    var initialized: usize = 0;
    errdefer for (cloned[0..initialized]) |case| case.deinit(allocator);

    for (cases) |case| {
        cloned[initialized] = try case.clone(allocator);
        initialized += 1;
    }
    return cloned;
}

const testing_span = SourceSpan{ .start = 3, .length = 5 };

fn testingFunctionAndBlock(store: *MirStore, interner: *Interner) !struct { function: MirFunctionId, block: MirBlockId } {
    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const block_id = try store.addBlock(function_id, testing_span);
    return .{ .function = function_id, .block = block_id };
}

test "MIR ID formatting and equality" {
    const rendered = try std.fmt.allocPrint(std.testing.allocator, "{f} {f} {f}", .{
        MirFunctionId{ .index = 1 },
        MirLocalId{ .index = 2 },
        MirBlockId{ .index = 3 },
    });
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings("MirFunctionId(1) MirLocalId(2) MirBlockId(3)", rendered);
    try std.testing.expectEqual(MirFunctionId{ .index = 7 }, MirFunctionId{ .index = 7 });
    try std.testing.expectEqual(MirLocalId{ .index = 8 }, MirLocalId{ .index = 8 });
    try std.testing.expectEqual(MirBlockId{ .index = 9 }, MirBlockId{ .index = 9 });
}

test "empty MIR store" {
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    try std.testing.expectEqual(@as(usize, 0), store.functions.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.locals.items.len);
    try std.testing.expectEqual(@as(usize, 0), store.blocks.items.len);
}

test "add MIR function and lookup by ID" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    const main_name = try interner.intern("main");
    const function_id = try store.addFunction(.{ .index = 4 }, main_name, .{ .index = 1 }, testing_span);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(u32, 0), function_id.index);
    try std.testing.expectEqual(hir.FunctionId{ .index = 4 }, function.hir_function);
    try std.testing.expectEqual(main_name, function.name);
    try std.testing.expectEqual(types.TypeId{ .index = 1 }, function.return_type);
    try std.testing.expectEqual(testing_span, function.source_span.?);
    try std.testing.expectEqualStrings("main", interner.text(function.name));
    try std.testing.expectEqual(@as(usize, 0), function.params.len);
    try std.testing.expectEqual(@as(usize, 0), function.locals.len);
    try std.testing.expectEqual(@as(usize, 0), function.blocks.len);
}

test "add MIR locals and associate with function" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const argc_id = try store.addLocal(function_id, try interner.intern("argc"), .param, .{ .index = 1 }, testing_span);
    const x_id = try store.addLocal(function_id, try interner.intern("x"), .user, .{ .index = 1 }, testing_span);
    const temp_id = try store.addLocal(function_id, null, .temp, .{ .index = 1 }, null);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(usize, 1), function.params.len);
    try std.testing.expectEqual(argc_id, function.params[0]);
    try std.testing.expectEqual(@as(usize, 2), function.locals.len);
    try std.testing.expectEqual(x_id, function.locals[0]);
    try std.testing.expectEqual(temp_id, function.locals[1]);

    try std.testing.expectEqual(function_id, store.getLocal(argc_id).parent);
    try std.testing.expectEqual(MirLocalKind.param, store.getLocal(argc_id).kind);
    try std.testing.expectEqualStrings("argc", interner.text(store.getLocal(argc_id).name.?));
    try std.testing.expectEqual(MirLocalKind.user, store.getLocal(x_id).kind);
    try std.testing.expectEqualStrings("x", interner.text(store.getLocal(x_id).name.?));
    try std.testing.expectEqual(MirLocalKind.temp, store.getLocal(temp_id).kind);
    try std.testing.expectEqual(@as(?hir.SymbolId, null), store.getLocal(temp_id).name);
}

test "add MIR block and associate with function" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();

    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const block_id = try store.addBlock(function_id, testing_span);
    const function = store.getFunction(function_id);

    try std.testing.expectEqual(@as(u32, 0), block_id.index);
    try std.testing.expectEqual(@as(usize, 1), function.blocks.len);
    try std.testing.expectEqual(block_id, function.blocks[0]);
    try std.testing.expectEqual(function_id, store.getBlock(block_id).parent);
    try std.testing.expectEqual(testing_span, store.getBlock(block_id).source_span.?);
    try std.testing.expectEqual(@as(usize, 0), store.getBlock(block_id).statements.len);
    try std.testing.expectEqual(@as(?MirTerminator, null), store.getBlock(block_id).terminator);
}

test "MIR debug snapshot formatting uses interned names" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var module = MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function_id = try module.store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, testing_span);
    _ = try module.store.addLocal(function_id, try interner.intern("argc"), .param, .{ .index = 1 }, testing_span);
    _ = try module.store.addLocal(function_id, try interner.intern("x"), .user, .{ .index = 1 }, testing_span);
    _ = try module.store.addLocal(function_id, null, .temp, .{ .index = 1 }, null);
    _ = try module.store.addBlock(function_id, testing_span);

    const rendered = try module.store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Params
        \\      MirLocalId(0) param argc: TypeId(1)
        \\    Locals
        \\      MirLocalId(1) user x: TypeId(1)
        \\      MirLocalId(2) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\
    , rendered);
}

test "MIR places and operands render in assignments" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();
    const ids = try testingFunctionAndBlock(&store, &interner);
    const x_id = try store.addLocal(ids.function, try interner.intern("x"), .user, .{ .index = 1 }, null);
    const flag_id = try store.addLocal(ids.function, try interner.intern("flag"), .user, .{ .index = 2 }, null);

    try store.appendStatement(ids.block, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(x_id), MirRvalue.use_(try MirOperand.intLiteral(std.testing.allocator, "123"))) });
    try store.appendStatement(ids.block, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(flag_id), MirRvalue.use_(MirOperand.boolLiteral(true))) });
    try store.appendStatement(ids.block, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(x_id), MirRvalue.use_(MirOperand.copyPlace(MirPlace.localPlace(x_id)))) });

    const rendered = try store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "MirLocalId(0) = Use(Int 123)") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "MirLocalId(1) = Use(Bool true)") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "MirLocalId(0) = Use(Copy(MirLocalId(0)))") != null);
}

test "MIR unary binary and call rvalues render" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();
    const ids = try testingFunctionAndBlock(&store, &interner);
    const temp_id = try store.addLocal(ids.function, null, .temp, .{ .index = 1 }, null);

    try store.appendStatement(ids.block, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(temp_id), MirRvalue.unaryOp(.negate, try MirOperand.intLiteral(std.testing.allocator, "7"))) });
    try store.appendStatement(ids.block, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(temp_id), MirRvalue.binaryOp(.add, MirOperand.copyPlace(MirPlace.localPlace(temp_id)), try MirOperand.intLiteral(std.testing.allocator, "2"))) });
    const call_args = [_]MirOperand{ MirOperand.copyPlace(MirPlace.localPlace(temp_id)), MirOperand.boolLiteral(false) };
    try store.appendStatement(ids.block, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(temp_id), try MirRvalue.callFunction(std.testing.allocator, .{ .index = 9 }, &call_args)) });

    const rendered = try store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "MirLocalId(0) = Unary - Int 7") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "MirLocalId(0) = Binary + Copy(MirLocalId(0)), Int 2") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "MirLocalId(0) = Call internal FunctionId(9)(Copy(MirLocalId(0)), Bool false)") != null);
}

test "MIR terminators render and reject replacement" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();
    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const first_block = try store.addBlock(function_id, null);
    const second_block = try store.addBlock(function_id, null);

    try store.setTerminator(first_block, .{ .span = null, .kind = MirTerminatorKind.gotoBlock(second_block) });
    try std.testing.expectError(error.MirBlockAlreadyTerminated, store.setTerminator(first_block, .{ .span = null, .kind = .@"unreachable" }));

    const rendered = try store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Goto MirBlockId(1)") != null);
}

test "MIR return terminators render operand and void" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();
    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const value_block = try store.addBlock(function_id, null);
    const void_block = try store.addBlock(function_id, null);
    const x_id = try store.addLocal(function_id, try interner.intern("x"), .user, .{ .index = 1 }, null);

    try store.setTerminator(value_block, .{ .span = null, .kind = MirTerminatorKind.returnValue(MirOperand.copyPlace(MirPlace.localPlace(x_id))) });
    try store.setTerminator(void_block, .{ .span = null, .kind = MirTerminatorKind.returnValue(null) });

    const rendered = try store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Return Copy(MirLocalId(0))") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "\n        Return\n") != null);
}

test "MIR switch and unreachable terminators render" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var store = MirStore.init(std.testing.allocator);
    defer store.deinit();
    const function_id = try store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, null);
    const bool_block = try store.addBlock(function_id, null);
    const int_block = try store.addBlock(function_id, null);
    const true_block = try store.addBlock(function_id, null);
    const false_block = try store.addBlock(function_id, null);
    const default_block = try store.addBlock(function_id, null);
    const unreachable_block = try store.addBlock(function_id, null);
    const flag_id = try store.addLocal(function_id, try interner.intern("flag"), .user, .{ .index = 2 }, null);

    try store.setTerminator(bool_block, .{ .span = null, .kind = MirTerminatorKind.switchBool(MirOperand.copyPlace(MirPlace.localPlace(flag_id)), true_block, false_block) });
    const cases = [_]MirSwitchIntCase{
        .{ .value = "1", .target = true_block },
        .{ .value = "2", .target = false_block },
    };
    try store.setTerminator(int_block, .{ .span = null, .kind = try MirTerminatorKind.switchInt(std.testing.allocator, try MirOperand.intLiteral(std.testing.allocator, "3"), &cases, default_block) });
    try store.setTerminator(unreachable_block, .{ .span = null, .kind = .@"unreachable" });

    const rendered = try store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "SwitchBool Copy(MirLocalId(0)) true: MirBlockId(2), false: MirBlockId(3)") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "SwitchInt Int 3 [1: MirBlockId(2), 2: MirBlockId(3)] default: MirBlockId(4)") != null);
    try std.testing.expect(std.mem.indexOf(u8, rendered, "Unreachable") != null);
}

test "MIR debug snapshot includes statements and terminator" {
    var interner = Interner.init(std.testing.allocator);
    defer interner.deinit();
    var module = MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function_id = try module.store.addFunction(.{ .index = 0 }, try interner.intern("main"), .{ .index = 1 }, testing_span);
    const x_id = try module.store.addLocal(function_id, try interner.intern("x"), .user, .{ .index = 1 }, testing_span);
    const temp_id = try module.store.addLocal(function_id, null, .temp, .{ .index = 1 }, null);
    const block_id = try module.store.addBlock(function_id, testing_span);

    try module.store.appendStatement(block_id, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(x_id), MirRvalue.use_(try MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.appendStatement(block_id, .{ .span = null, .kind = MirStatementKind.assignTo(MirPlace.localPlace(temp_id), MirRvalue.binaryOp(.add, MirOperand.copyPlace(MirPlace.localPlace(x_id)), try MirOperand.intLiteral(std.testing.allocator, "2"))) });
    try module.store.setTerminator(block_id, .{ .span = null, .kind = MirTerminatorKind.returnValue(MirOperand.copyPlace(MirPlace.localPlace(temp_id))) });

    const rendered = try module.store.debugString(std.testing.allocator, interner);
    defer std.testing.allocator.free(rendered);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\      MirLocalId(1) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 1)
        \\        MirLocalId(1) = Binary + Copy(MirLocalId(0)), Int 2
        \\        Return Copy(MirLocalId(1))
        \\
    , rendered);
}
