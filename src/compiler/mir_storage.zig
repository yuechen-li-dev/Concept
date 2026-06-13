const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const interner = @import("interner.zig");
const mir = @import("mir.zig");
const semantics = @import("semantics.zig");
const source = @import("source.zig");
const types = @import("types.zig");

pub const StorageState = enum {
    uninitialized,
    initialized,
    moved,
    /// Some incoming paths have an initialized value and some do not.
    maybe_initialized,
    /// Some incoming paths have an initialized value and some have moved it.
    maybe_moved,
    /// Struct storage has direct-field initializedness tracked separately.
    partially_initialized,
};

pub const StorageDiagnostic = struct {
    kind: Kind,
    local: mir.MirLocalId,
    span: source.SourceSpan,

    pub const Kind = enum {
        use_before_initialization,
        use_after_move,
        maybe_uninitialized_use,
        maybe_moved_use,
        implicit_copy_requires_copy,
        assignment_requires_replacement,
        use_of_partially_initialized_value,
    };
};

pub const AssignmentDecision = enum {
    allow_initialization,
    allow_reinitialization,
    allow_copy_replacement,
    reject_replacement_required,
    reject_maybe_uninitialized,
    reject_maybe_moved,
};

pub const CopyDropContext = struct {
    is_copy: bool,
    has_drop: bool,
};

pub fn canAssignToState(target_state: StorageState, context: CopyDropContext) AssignmentDecision {
    return switch (target_state) {
        .uninitialized => .allow_initialization,
        .moved => .allow_reinitialization,
        .initialized => if (!context.has_drop and context.is_copy) .allow_copy_replacement else .reject_replacement_required,
        .maybe_initialized => .reject_maybe_uninitialized,
        .maybe_moved => .reject_maybe_moved,
        .partially_initialized => .reject_maybe_uninitialized,
    };
}

const FieldStorageState = struct {
    struct_type: types.TypeId,
    fields: []StorageState,

    fn clone(self: FieldStorageState, allocator: std.mem.Allocator) !FieldStorageState {
        const fields = try allocator.alloc(StorageState, self.fields.len);
        @memcpy(fields, self.fields);
        return .{ .struct_type = self.struct_type, .fields = fields };
    }

    fn deinit(self: *FieldStorageState, allocator: std.mem.Allocator) void {
        allocator.free(self.fields);
        self.* = undefined;
    }
};

const StateFrame = struct {
    allocator: std.mem.Allocator,
    locals: []StorageState,
    field_states: []?FieldStorageState,

    fn init(allocator: std.mem.Allocator, local_count: usize) !StateFrame {
        const locals = try allocator.alloc(StorageState, local_count);
        errdefer allocator.free(locals);
        const field_states = try allocator.alloc(?FieldStorageState, local_count);
        errdefer allocator.free(field_states);
        @memset(field_states, null);
        return .{ .allocator = allocator, .locals = locals, .field_states = field_states };
    }

    fn empty(allocator: std.mem.Allocator) StateFrame {
        return .{ .allocator = allocator, .locals = &.{}, .field_states = &.{} };
    }

    fn clone(self: StateFrame) !StateFrame {
        var cloned = try StateFrame.init(self.allocator, self.locals.len);
        errdefer cloned.deinit();
        @memcpy(cloned.locals, self.locals);
        for (self.field_states, 0..) |maybe_fields, index| {
            if (maybe_fields) |field_state| {
                cloned.field_states[index] = try field_state.clone(self.allocator);
            }
        }
        return cloned;
    }

    fn deinit(self: *StateFrame) void {
        for (self.field_states) |*maybe_fields| {
            if (maybe_fields.*) |*field_state| {
                field_state.deinit(self.allocator);
            }
        }
        if (self.field_states.len > 0) self.allocator.free(self.field_states);
        if (self.locals.len > 0) self.allocator.free(self.locals);
        self.* = undefined;
    }

    fn clearFieldState(self: *StateFrame, local_id: mir.MirLocalId) void {
        if (self.field_states[local_id.index]) |*field_state| {
            field_state.deinit(self.allocator);
            self.field_states[local_id.index] = null;
        }
    }
};

pub const StorageAnalysis = struct {
    allocator: std.mem.Allocator,
    final_states: []StorageState,
    diagnostics: []StorageDiagnostic,

    pub fn deinit(self: *StorageAnalysis) void {
        self.allocator.free(self.final_states);
        self.allocator.free(self.diagnostics);
        self.* = undefined;
    }

    pub fn stateOf(self: StorageAnalysis, local_id: mir.MirLocalId) StorageState {
        return self.final_states[local_id.index];
    }
};

pub const AnalysisError = error{InvalidStorageState} || std.mem.Allocator.Error;

pub fn analyzeModule(
    allocator: std.mem.Allocator,
    semantic_module: *const semantics.SemanticModule,
    mir_module: *mir.MirModule,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
) AnalysisError!void {
    for (mir_module.store.functions.items, 0..) |function, index| {
        var analysis = try analyzeFunction(allocator, semantic_module, mir_module, .{ .index = @intCast(index) }, function, diagnostic_bag);
        defer analysis.deinit();
    }
}

pub fn analyzeFunction(
    allocator: std.mem.Allocator,
    semantic_module: *const semantics.SemanticModule,
    mir_module: *mir.MirModule,
    function_id: mir.MirFunctionId,
    function: mir.MirFunction,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
) AnalysisError!StorageAnalysis {
    var analyzer = Analyzer.init(allocator, semantic_module, mir_module, function_id, function, diagnostic_bag);
    return analyzer.analyze();
}

const Analyzer = struct {
    allocator: std.mem.Allocator,
    semantic_module: *const semantics.SemanticModule,
    mir_module: *mir.MirModule,
    function_id: mir.MirFunctionId,
    function: mir.MirFunction,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
    diagnostics: std.ArrayList(StorageDiagnostic),
    had_error: bool,
    emit_diagnostics: bool,

    fn init(
        allocator: std.mem.Allocator,
        semantic_module: *const semantics.SemanticModule,
        mir_module: *mir.MirModule,
        function_id: mir.MirFunctionId,
        function: mir.MirFunction,
        diagnostic_bag: ?*diagnostics.DiagnosticBag,
    ) Analyzer {
        return .{
            .allocator = allocator,
            .semantic_module = semantic_module,
            .mir_module = mir_module,
            .function_id = function_id,
            .function = function,
            .diagnostic_bag = diagnostic_bag,
            .diagnostics = std.ArrayList(StorageDiagnostic).empty,
            .had_error = false,
            .emit_diagnostics = true,
        };
    }

    fn analyze(self: *Analyzer) AnalysisError!StorageAnalysis {
        errdefer self.diagnostics.deinit(self.allocator);

        const local_count = self.mir_module.store.locals.items.len;
        var initial_states = try StateFrame.init(self.allocator, local_count);
        errdefer initial_states.deinit();
        @memset(initial_states.locals, .initialized);

        for (self.function.locals) |local_id| {
            initial_states.locals[local_id.index] = .uninitialized;
        }
        for (self.function.params) |param_id| {
            initial_states.locals[param_id.index] = .initialized;
        }

        const block_count = self.function.blocks.len;
        var in_states = try self.allocator.alloc(StateFrame, block_count);
        defer {
            for (in_states) |*states| {
                if (states.locals.len > 0) states.deinit();
            }
            self.allocator.free(in_states);
        }
        for (in_states) |*states| states.* = StateFrame.empty(self.allocator);

        var reachable = try self.allocator.alloc(bool, block_count);
        defer self.allocator.free(reachable);
        @memset(reachable, false);

        var queued = try self.allocator.alloc(bool, block_count);
        defer self.allocator.free(queued);
        @memset(queued, false);

        var worklist = std.ArrayList(usize).empty;
        defer worklist.deinit(self.allocator);

        if (block_count > 0) {
            in_states[0] = try initial_states.clone();
            reachable[0] = true;
            queued[0] = true;
            try worklist.append(self.allocator, 0);
        }

        var final_states = initial_states;
        initial_states = StateFrame.empty(self.allocator);
        var have_final = false;
        errdefer final_states.deinit();

        self.emit_diagnostics = false;
        while (worklist.items.len > 0) {
            const block_index = worklist.pop().?;
            queued[block_index] = false;
            const block_id = self.function.blocks[block_index];
            const block = self.mir_module.store.getBlock(block_id);

            var states = try in_states[block_index].clone();
            defer states.deinit();

            for (block.statements) |statement| {
                try self.analyzeStatement(&states, statement);
            }

            if (block.terminator) |terminator| {
                try self.analyzeTerminator(&states, terminator);
                switch (terminator.kind) {
                    .goto => |target| try self.propagate(states, target, in_states, reachable, queued, &worklist),
                    .switch_bool => |switch_bool| {
                        try self.propagate(states, switch_bool.true_target, in_states, reachable, queued, &worklist);
                        try self.propagate(states, switch_bool.false_target, in_states, reachable, queued, &worklist);
                    },
                    .switch_int => |switch_int| {
                        for (switch_int.cases) |case| {
                            try self.propagate(states, case.target, in_states, reachable, queued, &worklist);
                        }
                        try self.propagate(states, switch_int.default_target, in_states, reachable, queued, &worklist);
                    },
                    .return_, .@"unreachable" => {
                        if (have_final) {
                            _ = try joinFrames(&final_states, states);
                        } else {
                            try copyFrameInto(&final_states, states);
                            have_final = true;
                        }
                    },
                }
            } else {
                if (have_final) {
                    _ = try joinFrames(&final_states, states);
                } else {
                    try copyFrameInto(&final_states, states);
                    have_final = true;
                }
            }
        }
        self.emit_diagnostics = true;

        for (self.function.blocks, 0..) |block_id, block_index| {
            if (!reachable[block_index]) continue;
            const block = self.mir_module.store.getBlock(block_id);
            var states = try in_states[block_index].clone();
            defer states.deinit();

            for (block.statements) |statement| {
                try self.analyzeStatement(&states, statement);
            }
            if (block.terminator) |terminator| {
                try self.analyzeTerminator(&states, terminator);
                try self.insertCleanupBeforeTerminator(&states, block_id, terminator);
            }
        }

        if (!have_final and block_count > 0) {
            try copyFrameInto(&final_states, in_states[0]);
        }

        if (self.had_error) {
            return error.InvalidStorageState;
        }

        const owned_diagnostics = try self.diagnostics.toOwnedSlice(self.allocator);
        const owned_final_states = final_states.locals;
        for (final_states.field_states) |*maybe_fields| {
            if (maybe_fields.*) |*field_state| {
                field_state.deinit(self.allocator);
            }
        }
        self.allocator.free(final_states.field_states);
        final_states.field_states = &.{};
        final_states.locals = &.{};
        return .{
            .allocator = self.allocator,
            .final_states = owned_final_states,
            .diagnostics = owned_diagnostics,
        };
    }

    fn analyzeStatement(self: *Analyzer, states: *StateFrame, statement: mir.MirStatement) AnalysisError!void {
        switch (statement.kind) {
            .assign => |assignment| {
                const had_error_before = self.had_error;
                try self.readRvalue(states, assignment.rvalue, statement.span);
                if (self.had_error != had_error_before) return;
                try self.writePlace(states, assignment.place, statement.span);
            },
            .interface_call => |call| {
                try self.readOperand(states, call.receiver, statement.span);
                for (call.args) |arg| try self.readOperand(states, arg, statement.span);
            },
            .drop => |drop| try self.dropPlace(states, drop.place, statement.span),
            .arena_reset => |arena_operand| try self.readOperand(states, arena_operand, statement.span),
            .arena_destroy => |arena_operand| try self.readOperand(states, arena_operand, statement.span),
            .machine_step => |machine_operand| try self.readOperand(states, machine_operand, statement.span),
        }
    }

    fn analyzeTerminator(self: *Analyzer, states: *StateFrame, terminator: mir.MirTerminator) AnalysisError!void {
        switch (terminator.kind) {
            .goto, .@"unreachable" => {},
            .return_ => |maybe_operand| if (maybe_operand) |operand| try self.readOperand(states, operand, terminator.span),
            .switch_bool => |switch_bool| try self.readOperand(states, switch_bool.discriminant, terminator.span),
            .switch_int => |switch_int| try self.readOperand(states, switch_int.discriminant, terminator.span),
        }
    }

    fn readRvalue(self: *Analyzer, states: *StateFrame, rvalue: mir.MirRvalue, span: ?source.SourceSpan) AnalysisError!void {
        switch (rvalue) {
            .use => |operand| try self.readOperand(states, operand, span),
            .move => |place| try self.movePlace(states, place, span),
            .manual_init_assume => |operand| try self.readOperand(states, operand, span),
            .unary => |unary| try self.readOperand(states, unary.operand, span),
            .address_of => |place| try self.readPlace(states, place, span),
            .deref => |operand| try self.readOperand(states, operand, span),
            .dyn_coerce => |coerce| try self.readPlace(states, coerce.source, span),
            .binary => |binary| {
                try self.readOperand(states, binary.left, span);
                try self.readOperand(states, binary.right, span);
            },
            .call => |call| for (call.args) |arg| try self.readOperand(states, arg, span),
            .interface_call => |call| {
                try self.readOperand(states, call.receiver, span);
                for (call.args) |arg| try self.readOperand(states, arg, span);
            },
            .arena_alloc => |arena_alloc| try self.readOperand(states, arena_alloc.arena_operand, span),
            .machine_construct => |construct| for (construct.args) |arg| try self.readOperand(states, arg, span),
            .machine_complete => |operand| try self.readOperand(states, operand, span),
            .machine_result => |operand| try self.readOperand(states, operand, span),
            .enum_constructor => |constructor| for (constructor.args) |arg| try self.readOperand(states, arg, span),
            .struct_constructor => |constructor| for (constructor.fields) |field| try self.readOperand(states, field.value, span),
            .enum_tag => |operand| try self.readOperand(states, operand, span),
            .enum_payload_field => |payload| try self.readOperand(states, payload.enum_operand, span),
            .field_access => |field_access| {
                switch (field_access.receiver) {
                    .copy => |receiver_place| switch (receiver_place) {
                        .local => |base| {
                            const field_place = mir.MirPlace.fieldPlace(base, field_access.field_id);
                            try self.readPlace(states, field_place, span);
                            try self.checkImplicitCopy(states, field_place, span);
                        },
                        .field => {
                            try self.readOperand(states, field_access.receiver, span);
                            try self.checkImplicitCopyOfType(localOfPlace(receiver_place), self.semantic_module.hir.getField(field_access.field_id).type_id, true, span);
                        },
                    },
                    else => try self.readOperand(states, field_access.receiver, span),
                }
            },
        }
    }

    fn readOperand(self: *Analyzer, states: *StateFrame, operand: mir.MirOperand, span: ?source.SourceSpan) AnalysisError!void {
        switch (operand) {
            .int_literal, .bool_literal => {},
            .copy => |place| {
                try self.readPlace(states, place, span);
                try self.checkImplicitCopy(states, place, span);
            },
        }
    }

    fn readPlace(self: *Analyzer, states: *StateFrame, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = localOfPlace(place);
        const state = states.locals[local_id.index];
        switch (state) {
            .initialized => {},
            .uninitialized => try self.report(.use_before_initialization, local_id, span),
            .moved => try self.report(.use_after_move, local_id, span),
            .maybe_initialized => try self.report(.maybe_uninitialized_use, local_id, span),
            .maybe_moved => try self.report(.maybe_moved_use, local_id, span),
            .partially_initialized => switch (place) {
                .local => try self.report(.use_of_partially_initialized_value, local_id, span),
                .field => |field| {
                    const field_state = states.field_states[local_id.index] orelse {
                        try self.report(.use_of_partially_initialized_value, local_id, span);
                        return;
                    };
                    const index = self.fieldIndex(field_state.struct_type, field.field_id) orelse {
                        try self.report(.use_of_partially_initialized_value, local_id, span);
                        return;
                    };
                    switch (field_state.fields[index]) {
                        .initialized => {},
                        .uninitialized => try self.report(.use_before_initialization, local_id, span),
                        .moved => try self.report(.use_after_move, local_id, span),
                        .maybe_initialized => try self.report(.maybe_uninitialized_use, local_id, span),
                        .maybe_moved => try self.report(.maybe_moved_use, local_id, span),
                        .partially_initialized => try self.report(.use_of_partially_initialized_value, local_id, span),
                    }
                },
            },
        }
    }

    fn writePlace(self: *Analyzer, states: *StateFrame, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = localOfPlace(place);
        const local = self.mir_module.store.getLocal(local_id);
        if (local.kind == .temp) {
            states.locals[local_id.index] = .initialized;
            states.clearFieldState(local_id);
            return;
        }

        switch (place) {
            .field => |field| return self.writeFieldPlace(states, field, span),
            .local => {},
        }

        const target_type = self.placeType(place);
        const decision = canAssignToState(states.locals[local_id.index], .{
            .is_copy = self.isCopyType(target_type),
            .has_drop = self.semantic_module.hasDrop(target_type) != null,
        });
        switch (decision) {
            .allow_initialization, .allow_reinitialization, .allow_copy_replacement => {
                states.locals[local_id.index] = .initialized;
                states.clearFieldState(local_id);
            },
            .reject_replacement_required => try self.report(.assignment_requires_replacement, local_id, span),
            .reject_maybe_uninitialized => try self.report(.maybe_uninitialized_use, local_id, span),
            .reject_maybe_moved => try self.report(.maybe_moved_use, local_id, span),
        }
    }

    fn movePlace(self: *Analyzer, states: *StateFrame, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        try self.readPlace(states, place, span);
        const local_id = localOfPlace(place);
        if (!self.isCopyLocal(local_id)) {
            states.locals[local_id.index] = .moved;
            states.clearFieldState(local_id);
        }
    }

    fn dropPlace(self: *Analyzer, states: *StateFrame, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = localOfPlace(place);
        try self.readPlace(states, place, span);
        switch (place) {
            .local => if (states.locals[local_id.index] == .initialized) {
                states.locals[local_id.index] = .moved;
                states.clearFieldState(local_id);
            },
            .field => |field| {
                if (states.locals[local_id.index] == .partially_initialized) {
                    if (states.field_states[local_id.index]) |*field_state| {
                        if (self.fieldIndex(field_state.struct_type, field.field_id)) |index| {
                            field_state.fields[index] = .moved;
                        }
                    }
                }
            },
        }
    }

    fn insertCleanupBeforeTerminator(self: *Analyzer, states: *StateFrame, block_id: mir.MirBlockId, terminator: mir.MirTerminator) AnalysisError!void {
        if (!self.emit_diagnostics) return;
        switch (terminator.kind) {
            .return_ => {},
            else => return,
        }
        const block = self.mir_module.store.getBlock(block_id);
        for (block.statements) |statement| {
            if (std.meta.activeTag(statement.kind) == .drop) return;
        }

        var index = self.function.locals.len;
        while (index > 0) {
            index -= 1;
            const local_id = self.function.locals[index];
            const local = self.mir_module.store.getLocal(local_id);
            if (local.kind != .user) continue;
            const whole_drop_info = self.semantic_module.hasDrop(local.type_id);
            switch (states.locals[local_id.index]) {
                .initialized => {
                    const drop_info = whole_drop_info orelse continue;
                    try self.mir_module.store.appendStatement(block_id, .{
                        .span = terminator.span,
                        .kind = mir.MirStatementKind.dropPlace(mir.MirPlace.localPlace(local_id), drop_info.function),
                    });
                    states.locals[local_id.index] = .moved;
                    states.clearFieldState(local_id);
                },
                .moved, .uninitialized => {},
                .maybe_moved => if (whole_drop_info != null) try self.report(.maybe_moved_use, local_id, terminator.span),
                .maybe_initialized => if (whole_drop_info != null) try self.report(.maybe_uninitialized_use, local_id, terminator.span),
                .partially_initialized => try self.insertPartialCleanup(states, block_id, local_id, terminator.span),
            }
        }
    }

    fn writeFieldPlace(self: *Analyzer, states: *StateFrame, field_place: anytype, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = field_place.base;
        const target_type = self.semantic_module.hir.getField(field_place.field_id).type_id;
        const context = CopyDropContext{
            .is_copy = self.isCopyType(target_type),
            .has_drop = self.semantic_module.hasDrop(target_type) != null,
        };
        switch (states.locals[local_id.index]) {
            .initialized => {
                const decision = canAssignToState(.initialized, context);
                switch (decision) {
                    .allow_copy_replacement => {},
                    .reject_replacement_required => try self.report(.assignment_requires_replacement, local_id, span),
                    else => unreachable,
                }
            },
            .uninitialized => {
                const field_state = try self.ensurePartialFieldState(states, local_id);
                try self.initializePartialField(states, local_id, field_state, field_place.field_id);
            },
            .partially_initialized => {
                const field_state = try self.ensurePartialFieldState(states, local_id);
                const field_index = self.fieldIndex(field_state.struct_type, field_place.field_id) orelse {
                    try self.report(.use_of_partially_initialized_value, local_id, span);
                    return;
                };
                const decision = canAssignToState(field_state.fields[field_index], context);
                switch (decision) {
                    .allow_initialization, .allow_reinitialization, .allow_copy_replacement => try self.initializePartialField(states, local_id, field_state, field_place.field_id),
                    .reject_replacement_required => try self.report(.assignment_requires_replacement, local_id, span),
                    .reject_maybe_uninitialized => try self.report(.maybe_uninitialized_use, local_id, span),
                    .reject_maybe_moved => try self.report(.maybe_moved_use, local_id, span),
                }
            },
            .moved => try self.report(.use_after_move, local_id, span),
            .maybe_moved => try self.report(.maybe_moved_use, local_id, span),
            .maybe_initialized => try self.report(.maybe_uninitialized_use, local_id, span),
        }
    }

    fn ensurePartialFieldState(self: *Analyzer, states: *StateFrame, local_id: mir.MirLocalId) AnalysisError!*FieldStorageState {
        if (states.field_states[local_id.index] == null) {
            const local = self.mir_module.store.getLocal(local_id);
            const struct_id = switch (self.semantic_module.types.kind(local.type_id)) {
                .struct_type => |struct_id| struct_id,
                else => return error.InvalidStorageState,
            };
            const struct_decl = self.semantic_module.hir.getStruct(struct_id);
            const fields = try self.allocator.alloc(StorageState, struct_decl.fields.len);
            @memset(fields, .uninitialized);
            states.field_states[local_id.index] = .{ .struct_type = local.type_id, .fields = fields };
        }
        states.locals[local_id.index] = .partially_initialized;
        if (states.field_states[local_id.index]) |*field_state| return field_state;
        unreachable;
    }

    fn initializePartialField(self: *Analyzer, states: *StateFrame, local_id: mir.MirLocalId, field_state: *FieldStorageState, field_id: hir.FieldId) AnalysisError!void {
        const field_index = self.fieldIndex(field_state.struct_type, field_id) orelse return error.InvalidStorageState;
        field_state.fields[field_index] = .initialized;
        if (allFieldsInitialized(field_state.fields)) {
            states.locals[local_id.index] = .initialized;
            states.clearFieldState(local_id);
        }
    }

    fn insertPartialCleanup(self: *Analyzer, states: *StateFrame, block_id: mir.MirBlockId, local_id: mir.MirLocalId, span: ?source.SourceSpan) AnalysisError!void {
        const field_state = states.field_states[local_id.index] orelse {
            try self.report(.use_of_partially_initialized_value, local_id, span);
            return;
        };
        const struct_id = switch (self.semantic_module.types.kind(field_state.struct_type)) {
            .struct_type => |struct_id| struct_id,
            else => return error.InvalidStorageState,
        };
        const struct_decl = self.semantic_module.hir.getStruct(struct_id);
        var index = struct_decl.fields.len;
        while (index > 0) {
            index -= 1;
            if (field_state.fields[index] != .initialized) continue;
            const field_id = struct_decl.fields[index];
            const field = self.semantic_module.hir.getField(field_id);
            const drop_info = self.semantic_module.hasDrop(field.type_id) orelse continue;
            try self.mir_module.store.appendStatement(block_id, .{
                .span = span,
                .kind = mir.MirStatementKind.dropPlace(mir.MirPlace.fieldPlace(local_id, field_id), drop_info.function),
            });
        }
        states.locals[local_id.index] = .moved;
        states.clearFieldState(local_id);
    }

    fn isCopyLocal(self: *Analyzer, local_id: mir.MirLocalId) bool {
        const type_id = self.mir_module.store.getLocal(local_id).type_id;
        return self.isCopyType(type_id);
    }

    fn isCopyType(self: *Analyzer, type_id: types.TypeId) bool {
        if (self.semantic_module.hasDrop(type_id) != null) return false;
        return self.semantic_module.types.isCopyType(&self.semantic_module.hir, type_id);
    }

    fn placeType(self: *Analyzer, place: mir.MirPlace) types.TypeId {
        return switch (place) {
            .local => |local_id| self.mir_module.store.getLocal(local_id).type_id,
            .field => |field| self.semantic_module.hir.getField(field.field_id).type_id,
        };
    }

    fn checkImplicitCopy(self: *Analyzer, states: *const StateFrame, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = localOfPlace(place);
        if (!self.isPlaceDefinitelyInitialized(states, place)) return;
        const allow_temp = place == .field;
        try self.checkImplicitCopyOfType(local_id, self.placeType(place), allow_temp, span);
    }

    fn checkImplicitCopyOfType(self: *Analyzer, local_id: mir.MirLocalId, type_id: types.TypeId, allow_temp: bool, span: ?source.SourceSpan) AnalysisError!void {
        const local = self.mir_module.store.getLocal(local_id);
        switch (local.kind) {
            .param, .user => {},
            .temp => if (!allow_temp) return,
        }
        if (!self.isCopyType(type_id)) {
            try self.report(.implicit_copy_requires_copy, local_id, span);
        }
    }

    fn isPlaceDefinitelyInitialized(self: *Analyzer, states: *const StateFrame, place: mir.MirPlace) bool {
        const local_id = localOfPlace(place);
        return switch (states.locals[local_id.index]) {
            .initialized => true,
            .partially_initialized => switch (place) {
                .local => false,
                .field => |field| blk: {
                    const field_state = states.field_states[local_id.index] orelse break :blk false;
                    const index = self.fieldIndex(field_state.struct_type, field.field_id) orelse break :blk false;
                    break :blk field_state.fields[index] == .initialized;
                },
            },
            else => false,
        };
    }

    fn propagate(
        self: *Analyzer,
        states: StateFrame,
        target: mir.MirBlockId,
        in_states: []StateFrame,
        reachable: []bool,
        queued: []bool,
        worklist: *std.ArrayList(usize),
    ) AnalysisError!void {
        const target_index = self.blockIndex(target) orelse return;
        if (!reachable[target_index]) {
            in_states[target_index] = try states.clone();
            reachable[target_index] = true;
            if (!queued[target_index]) {
                queued[target_index] = true;
                try worklist.append(self.allocator, target_index);
            }
            return;
        }

        const changed = try joinFrames(&in_states[target_index], states);
        if (changed and !queued[target_index]) {
            queued[target_index] = true;
            try worklist.append(self.allocator, target_index);
        }
    }

    fn report(self: *Analyzer, kind: StorageDiagnostic.Kind, local_id: mir.MirLocalId, span: ?source.SourceSpan) AnalysisError!void {
        if (!self.emit_diagnostics) return;
        const diagnostic_span = span orelse self.mir_module.store.getLocal(local_id).source_span orelse self.function.source_span orelse source.SourceSpan{ .start = 0, .length = 0 };
        try self.diagnostics.append(self.allocator, .{
            .kind = kind,
            .local = local_id,
            .span = diagnostic_span,
        });
        if (self.diagnostic_bag) |bag| {
            try bag.append(switch (kind) {
                .use_before_initialization => diagnostics.useBeforeInitialization(diagnostic_span),
                .use_after_move => diagnostics.useAfterMove(diagnostic_span),
                .maybe_uninitialized_use => diagnostics.maybeUninitializedUse(diagnostic_span),
                .maybe_moved_use => diagnostics.maybeMovedUse(diagnostic_span),
                .implicit_copy_requires_copy => diagnostics.implicitCopyRequiresCopy(diagnostic_span),
                .assignment_requires_replacement => diagnostics.assignmentRequiresReplacement(diagnostic_span),
                .use_of_partially_initialized_value => diagnostics.useOfPartiallyInitializedValue(diagnostic_span),
            });
        }
        self.had_error = true;
    }

    fn fieldIndex(self: *Analyzer, struct_type: types.TypeId, field_id: hir.FieldId) ?usize {
        const struct_id = switch (self.semantic_module.types.kind(struct_type)) {
            .struct_type => |struct_id| struct_id,
            else => return null,
        };
        const struct_decl = self.semantic_module.hir.getStruct(struct_id);
        for (struct_decl.fields, 0..) |candidate, index| {
            if (candidate.index == field_id.index) return index;
        }
        return null;
    }

    fn blockIndex(self: *Analyzer, block_id: mir.MirBlockId) ?usize {
        for (self.function.blocks, 0..) |candidate, index| {
            if (candidate.index == block_id.index) return index;
        }
        return null;
    }
};

fn localOfPlace(place: mir.MirPlace) mir.MirLocalId {
    return switch (place) {
        .local => |local_id| local_id,
        .field => |field| field.base,
    };
}

fn allFieldsInitialized(fields: []const StorageState) bool {
    for (fields) |field| {
        if (field != .initialized) return false;
    }
    return true;
}

fn copyFrameInto(destination: *StateFrame, source_frame: StateFrame) !void {
    @memcpy(destination.locals, source_frame.locals);
    for (destination.field_states, 0..) |*maybe_fields, index| {
        if (maybe_fields.*) |*field_state| {
            field_state.deinit(destination.allocator);
            maybe_fields.* = null;
        }
        if (source_frame.field_states[index]) |field_state| {
            maybe_fields.* = try field_state.clone(destination.allocator);
        }
    }
}

fn joinFrames(existing: *StateFrame, incoming: StateFrame) !bool {
    var changed = false;
    for (existing.locals, incoming.locals, 0..) |*current, next, index| {
        const joined = joinStorageStateWithFields(current.*, existing.field_states[index], next, incoming.field_states[index]);
        if (joined != current.*) {
            current.* = joined;
            changed = true;
        }
        if (joined == .partially_initialized) {
            if (existing.field_states[index] == null and incoming.field_states[index] != null) {
                existing.field_states[index] = try incoming.field_states[index].?.clone(existing.allocator);
                changed = true;
            }
        } else if (existing.field_states[index] != null) {
            existing.clearFieldState(.{ .index = @intCast(index) });
            changed = true;
        }
    }
    return changed;
}

fn joinStorageStateWithFields(left: StorageState, left_fields: ?FieldStorageState, right: StorageState, right_fields: ?FieldStorageState) StorageState {
    if (left == .partially_initialized and right == .partially_initialized) {
        if (sameFieldStates(left_fields, right_fields)) return .partially_initialized;
        return .maybe_initialized;
    }
    if (left == .partially_initialized or right == .partially_initialized) return .maybe_initialized;
    return joinStorageState(left, right);
}

fn sameFieldStates(left: ?FieldStorageState, right: ?FieldStorageState) bool {
    const left_state = left orelse return false;
    const right_state = right orelse return false;
    if (left_state.struct_type.index != right_state.struct_type.index) return false;
    if (left_state.fields.len != right_state.fields.len) return false;
    for (left_state.fields, right_state.fields) |left_field, right_field| {
        if (left_field != right_field) return false;
    }
    return true;
}

fn joinStorageState(left: StorageState, right: StorageState) StorageState {
    if (left == right) return left;
    if (left == .partially_initialized or right == .partially_initialized) return .maybe_initialized;

    const left_uninit = left == .uninitialized or left == .maybe_initialized;
    const right_uninit = right == .uninitialized or right == .maybe_initialized;
    const left_moved = left == .moved or left == .maybe_moved;
    const right_moved = right == .moved or right == .maybe_moved;
    const any_uninit = left_uninit or right_uninit;
    const any_moved = left_moved or right_moved;
    const any_initialized = left == .initialized or right == .initialized or left == .maybe_initialized or right == .maybe_initialized or left == .maybe_moved or right == .maybe_moved;

    if (any_uninit) {
        // Mixed moved/uninitialized paths are represented as maybe-initialized:
        // at least one incoming path lacks a usable live value, and P10-M4 does
        // not add a combined public maybe-unusable state.
        return .maybe_initialized;
    }
    if (any_moved and any_initialized) return .maybe_moved;
    if (any_moved) return .moved;
    return .initialized;
}

test "MIR storage joins branch states conservatively" {
    try std.testing.expectEqual(StorageState.initialized, joinStorageState(.initialized, .initialized));
    try std.testing.expectEqual(StorageState.moved, joinStorageState(.moved, .moved));
    try std.testing.expectEqual(StorageState.uninitialized, joinStorageState(.uninitialized, .uninitialized));
    try std.testing.expectEqual(StorageState.maybe_moved, joinStorageState(.initialized, .moved));
    try std.testing.expectEqual(StorageState.maybe_moved, joinStorageState(.moved, .initialized));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageState(.initialized, .uninitialized));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageState(.uninitialized, .initialized));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageState(.moved, .uninitialized));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageState(.uninitialized, .moved));
    try std.testing.expectEqual(StorageState.maybe_moved, joinStorageState(.maybe_moved, .initialized));
    try std.testing.expectEqual(StorageState.maybe_moved, joinStorageState(.maybe_moved, .moved));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageState(.maybe_moved, .uninitialized));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageState(.maybe_initialized, .moved));
}

test "MIR storage rejects maybe-state reads" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("x"), .param, semantic_module.types.intType(), hir.synthetic_span);
    _ = try module.store.addBlock(function, hir.synthetic_span);

    var states = try StateFrame.init(std.testing.allocator, module.store.locals.items.len);
    defer states.deinit();
    @memset(states.locals, .initialized);

    var analyzer = Analyzer.init(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analyzer.diagnostics.deinit(std.testing.allocator);

    states.locals[local.index] = .maybe_moved;
    try analyzer.readPlace(&states, .{ .local = local }, hir.synthetic_span);
    try std.testing.expectEqual(StorageDiagnostic.Kind.maybe_moved_use, analyzer.diagnostics.items[0].kind);

    states.locals[local.index] = .maybe_initialized;
    try analyzer.readPlace(&states, .{ .local = local }, hir.synthetic_span);
    try std.testing.expectEqual(StorageDiagnostic.Kind.maybe_uninitialized_use, analyzer.diagnostics.items[1].kind);
}

test "MIR storage branch move join produces maybe moved" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const box_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const cond = try module.store.addLocal(function, try interner_value.intern("cond"), .param, type_store.boolType(), hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, box_type, hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const then_block = try module.store.addBlock(function, hir.synthetic_span);
    const join_block = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.appendStatement(entry, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.copyPlace(.{ .local = cond }), then_block, join_block) });
    try module.store.appendStatement(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.movePlace(.{ .local = source_local })) });
    try module.store.setTerminator(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(join_block) });
    try module.store.setTerminator(join_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = source_local })) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.MaybeMovedUse, bag.diagnostics.items[0].code);
}

test "MIR storage branch maybe initialized join is rejected on read" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const cond = try module.store.addLocal(function, try interner_value.intern("cond"), .param, type_store.boolType(), hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("x"), .user, type_store.intType(), hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const then_block = try module.store.addBlock(function, hir.synthetic_span);
    const join_block = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.copyPlace(.{ .local = cond }), then_block, join_block) });
    try module.store.appendStatement(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.setTerminator(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(join_block) });
    try module.store.setTerminator(join_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = local })) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.MaybeUninitializedUse, bag.diagnostics.items[0].code);
}

test "MIR storage returned move path does not poison later join" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const box_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const cond = try module.store.addLocal(function, try interner_value.intern("cond"), .param, type_store.boolType(), hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, box_type, hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const then_block = try module.store.addBlock(function, hir.synthetic_span);
    const join_block = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.appendStatement(entry, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.copyPlace(.{ .local = cond }), then_block, join_block) });
    try module.store.appendStatement(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.movePlace(.{ .local = source_local })) });
    try module.store.setTerminator(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "7")) });
    try module.store.setTerminator(join_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "42")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.maybe_moved, analysis.stateOf(source_local));
}

test "MIR storage worklist converges on scalar loop" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("x"), .user, type_store.intType(), hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const condition = try module.store.addBlock(function, hir.synthetic_span);
    const body = try module.store.addBlock(function, hir.synthetic_span);
    const exit = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.appendStatement(entry, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "0"))) });
    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(condition) });
    try module.store.setTerminator(condition, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.boolLiteral(false), body, exit) });
    try module.store.appendStatement(body, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.setTerminator(body, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(condition) });
    try module.store.setTerminator(exit, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = local })) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(local));
}

test "MIR storage initializes params and rejects uninitialized local return" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), .{ .index = 1 }, hir.synthetic_span);
    const param = try module.store.addLocal(function, try interner_value.intern("p"), .param, .{ .index = 1 }, hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("x"), .user, .{ .index = 1 }, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = local })) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UseBeforeInitialization, bag.diagnostics.items[0].code);

    var ok_module = mir.MirModule.init(std.testing.allocator);
    defer ok_module.deinit();
    const ok_function = try ok_module.store.addFunction(.{ .index = 0 }, try interner_value.intern("id"), .{ .index = 1 }, hir.synthetic_span);
    const ok_param = try ok_module.store.addLocal(ok_function, try interner_value.intern("p"), .param, .{ .index = 1 }, hir.synthetic_span);
    const ok_block = try ok_module.store.addBlock(ok_function, hir.synthetic_span);
    try ok_module.store.setTerminator(ok_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = ok_param })) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &ok_module, ok_function, ok_module.store.getFunction(ok_function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(ok_param));
    try std.testing.expectEqual(StorageState.initialized, analysis.final_states[param.index]);
}

test "MIR storage marks assigned locals initialized before read" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), .{ .index = 1 }, hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("x"), .user, .{ .index = 1 }, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = local })) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(local));
}

test "MIR storage move marks non-copy local moved" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const non_copy_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), non_copy_type, hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, non_copy_type, hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, non_copy_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.movePlace(.{ .local = source_local })) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = source_local })) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UseAfterMove, bag.diagnostics.items[0].code);
}

test "MIR storage move keeps int local initialized" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), .{ .index = 1 }, hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, .{ .index = 1 }, hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, .{ .index = 1 }, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.movePlace(.{ .local = source_local })) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = source_local })) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(source_local));
}

test "MIR storage rejects manually marked moved local read" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), .{ .index = 1 }, hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("x"), .param, .{ .index = 1 }, hir.synthetic_span);
    _ = try module.store.addBlock(function, hir.synthetic_span);

    var states = try StateFrame.init(std.testing.allocator, module.store.locals.items.len);
    defer states.deinit();
    @memset(states.locals, .initialized);
    states.locals[local.index] = .moved;

    var analyzer = Analyzer.init(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analyzer.diagnostics.deinit(std.testing.allocator);
    try analyzer.readPlace(&states, .{ .local = local }, hir.synthetic_span);
    try std.testing.expect(analyzer.had_error);
    try std.testing.expectEqual(StorageDiagnostic.Kind.use_after_move, analyzer.diagnostics.items[0].kind);
}

test "MIR storage field access requires initialized base" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), .{ .index = 1 }, hir.synthetic_span);
    const base = try module.store.addLocal(function, try interner_value.intern("pair"), .user, .{ .index = 1 }, hir.synthetic_span);
    const out = try module.store.addLocal(function, try interner_value.intern("out"), .user, .{ .index = 1 }, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = out }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(mir.MirPlace.fieldPlace(base, .{ .index = 0 })))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UseBeforeInitialization, bag.diagnostics.items[0].code);
}

test "MIR storage allows ordinary copy of int local" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, type_store.intType(), hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, type_store.intType(), hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "41"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(.{ .local = source_local }))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = dest_local })) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(source_local));
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(dest_local));
}

test "MIR storage rejects ordinary copy of struct local" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const box_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, box_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(.{ .local = source_local }))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.ImplicitCopyRequiresCopy, bag.diagnostics.items[0].code);
}

test "MIR storage allows explicit move of struct local" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const box_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const source_local = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const dest_local = try module.store.addLocal(function, try interner_value.intern("b"), .user, box_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = source_local }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = dest_local }, mir.MirRvalue.movePlace(.{ .local = source_local })) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.moved, analysis.stateOf(source_local));
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(dest_local));
}

test "MIR storage rejects implicit copy of struct param in call and return" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const box_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("forward"), box_type, hir.synthetic_span);
    const param = try module.store.addLocal(function, try interner_value.intern("b"), .param, box_type, hir.synthetic_span);
    const temp = try module.store.addLocal(function, null, .temp, box_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    const args = [_]mir.MirOperand{mir.MirOperand.copyPlace(.{ .local = param })};
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, try mir.MirRvalue.callFunction(std.testing.allocator, .{ .index = 1 }, &args)) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = param })) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 2), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.ImplicitCopyRequiresCopy, bag.diagnostics.items[0].code);
    try std.testing.expectEqual(diagnostics.DiagnosticCode.ImplicitCopyRequiresCopy, bag.diagnostics.items[1].code);
}

test "MIR storage allows non-Copy temporaries and struct constructors as fresh values" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const type_store = &semantic_module.types;
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const box_type = try type_store.addStructType(.{ .index = 0 });
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), type_store.intType(), hir.synthetic_span);
    const temp = try module.store.addLocal(function, null, .temp, box_type, hir.synthetic_span);
    const local = try module.store.addLocal(function, try interner_value.intern("b"), .user, box_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = local }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(.{ .local = temp }))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(local));
}

fn addDropImplForTest(semantic_module: *semantics.SemanticModule, type_id: types.TypeId) !hir.FunctionId {
    const drop_name = try semantic_module.interner.intern("Drop");
    const concept_id = try semantic_module.hir.addConcept(drop_name, false, false, hir.synthetic_span);
    const t_name = try semantic_module.interner.intern("T");
    const t_type = try semantic_module.types.addTypeParam(.{ .kind = .concept, .index = concept_id.index }, 0, t_name);
    const type_params = try std.testing.allocator.alloc(hir.HirTypeParam, 1);
    type_params[0] = .{ .name = t_name, .span = hir.synthetic_span, .type_id = t_type };
    semantic_module.hir.setConceptTypeParams(concept_id, type_params);

    const value_name = try semantic_module.interner.intern("value");
    const requirement_params = try std.testing.allocator.alloc(hir.HirConceptParam, 1);
    requirement_params[0] = .{ .name = value_name, .span = hir.synthetic_span, .type_id = t_type };
    const requirements = try std.testing.allocator.alloc(hir.HirConceptRequirement, 1);
    requirements[0] = .{
        .name = try semantic_module.interner.intern("drop"),
        .return_type = semantic_module.types.voidType(),
        .params = requirement_params,
        .span = hir.synthetic_span,
    };
    semantic_module.hir.setConceptRequirements(concept_id, requirements);

    const drop_fn = try semantic_module.hir.addConceptWitnessFunction(try semantic_module.interner.intern("drop"), semantic_module.types.voidType(), false, hir.synthetic_span);
    _ = try semantic_module.hir.addParam(drop_fn, value_name, type_id, hir.synthetic_span);
    semantic_module.hir.markConceptWitnessReferenced(drop_fn);
    var functions = try std.testing.allocator.alloc(hir.FunctionId, 1);
    functions[0] = drop_fn;
    _ = try semantic_module.hir.addConceptImpl(concept_id, type_id, functions, false, hir.synthetic_span);
    return drop_fn;
}

test "Drop lookup recognizes explicit Drop impl for struct only" {
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();

    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    try std.testing.expect(semantic_module.hasDrop(semantic_module.types.intType()) == null);
    try std.testing.expect(semantic_module.hasDrop(semantic_module.types.boolType()) == null);
    try std.testing.expect(semantic_module.hasDrop(file_type) == null);

    const drop_fn = try addDropImplForTest(&semantic_module, file_type);
    const drop_info = semantic_module.hasDrop(file_type).?;
    try std.testing.expectEqual(file_type, drop_info.type_id);
    try std.testing.expectEqual(drop_fn, drop_info.function);
}

test "MIR storage tracks partial struct field initialization and allows initialized field read" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const pair_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("Pair"));
    const left = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("left"), semantic_module.types.intType(), hir.synthetic_span);
    const right = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("right"), semantic_module.types.intType(), hir.synthetic_span);
    _ = right;
    const pair_type = try semantic_module.types.addStructType(pair_struct);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const p = try module.store.addLocal(function, try interner_value.intern("p"), .user, pair_type, hir.synthetic_span);
    const out = try module.store.addLocal(function, try interner_value.intern("out"), .user, semantic_module.types.intType(), hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(p, left), mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "20"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = out }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(mir.MirPlace.fieldPlace(p, left)))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.partially_initialized, analysis.stateOf(p));
}

test "MIR storage rejects uninitialized field read from partial struct" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const pair_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("Pair"));
    const left = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("left"), semantic_module.types.intType(), hir.synthetic_span);
    const right = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("right"), semantic_module.types.intType(), hir.synthetic_span);
    const pair_type = try semantic_module.types.addStructType(pair_struct);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const p = try module.store.addLocal(function, try interner_value.intern("p"), .user, pair_type, hir.synthetic_span);
    const out = try module.store.addLocal(function, try interner_value.intern("out"), .user, semantic_module.types.intType(), hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(p, left), mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "20"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = out }, mir.MirRvalue.use_(mir.MirOperand.copyPlace(mir.MirPlace.fieldPlace(p, right)))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UseBeforeInitialization, bag.diagnostics.items[0].code);
}

test "MIR storage promotes partial struct to initialized after all fields are written" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const pair_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("Pair"));
    const left = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("left"), semantic_module.types.intType(), hir.synthetic_span);
    const right = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("right"), semantic_module.types.intType(), hir.synthetic_span);
    const pair_type = try semantic_module.types.addStructType(pair_struct);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const p = try module.store.addLocal(function, try interner_value.intern("p"), .user, pair_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(p, left), mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "20"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(p, right), mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "22"))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(mir.MirPlace.fieldPlace(p, left))) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(p));
}

test "MIR storage rejects whole move of partially initialized struct" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const pair_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("Pair"));
    const left = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("left"), semantic_module.types.intType(), hir.synthetic_span);
    _ = try semantic_module.hir.addField(pair_struct, try semantic_module.interner.intern("right"), semantic_module.types.intType(), hir.synthetic_span);
    const pair_type = try semantic_module.types.addStructType(pair_struct);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const p = try module.store.addLocal(function, try interner_value.intern("p"), .user, pair_type, hir.synthetic_span);
    const q = try module.store.addLocal(function, try interner_value.intern("q"), .user, pair_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(p, left), mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "20"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = q }, mir.MirRvalue.movePlace(.{ .local = p })) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.UseOfPartiallyInitializedValue, bag.diagnostics.items[0].code);
}

test "MIR storage rejects replacement of initialized Drop field" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const file_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("File"));
    const file_type = try semantic_module.types.addStructType(file_struct);
    _ = try addDropImplForTest(&semantic_module, file_type);
    const holder_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("Holder"));
    const file_field = try semantic_module.hir.addField(holder_struct, try semantic_module.interner.intern("file"), file_type, hir.synthetic_span);
    const holder_type = try semantic_module.types.addStructType(holder_struct);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const h = try module.store.addLocal(function, try interner_value.intern("h"), .user, holder_type, hir.synthetic_span);
    const f1 = try module.store.addLocal(function, null, .temp, file_type, hir.synthetic_span);
    const f2 = try module.store.addLocal(function, null, .temp, file_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = f1 }, try mir.MirRvalue.structConstructor(std.testing.allocator, file_struct, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = h }, try mir.MirRvalue.structConstructor(std.testing.allocator, holder_struct, &.{.{ .field_id = file_field, .value = mir.MirOperand.copyPlace(.{ .local = f1 }) }})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = f2 }, try mir.MirRvalue.structConstructor(std.testing.allocator, file_struct, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(h, file_field), mir.MirRvalue.use_(mir.MirOperand.copyPlace(.{ .local = f2 }))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.AssignmentRequiresReplacement, bag.diagnostics.items[0].code);
}

test "MIR storage drops initialized Drop fields of partial struct in reverse declaration order" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const file_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("File"));
    const file_type = try semantic_module.types.addStructType(file_struct);
    const file_drop = try addDropImplForTest(&semantic_module, file_type);
    const holder_struct = try semantic_module.hir.addStruct(try semantic_module.interner.intern("Holder"));
    const first = try semantic_module.hir.addField(holder_struct, try semantic_module.interner.intern("first"), file_type, hir.synthetic_span);
    const second = try semantic_module.hir.addField(holder_struct, try semantic_module.interner.intern("second"), file_type, hir.synthetic_span);
    _ = try semantic_module.hir.addField(holder_struct, try semantic_module.interner.intern("rest"), semantic_module.types.intType(), hir.synthetic_span);
    const holder_type = try semantic_module.types.addStructType(holder_struct);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const h = try module.store.addLocal(function, try interner_value.intern("h"), .user, holder_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(h, first), try mir.MirRvalue.structConstructor(std.testing.allocator, file_struct, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(mir.MirPlace.fieldPlace(h, second), try mir.MirRvalue.structConstructor(std.testing.allocator, file_struct, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    const statements = module.store.getBlock(block).statements;
    try std.testing.expectEqual(@as(usize, 4), statements.len);
    try std.testing.expectEqual(second, statements[2].kind.drop.place.field.field_id);
    try std.testing.expectEqual(file_drop, statements[2].kind.drop.function);
    try std.testing.expectEqual(first, statements[3].kind.drop.place.field.field_id);
    try std.testing.expectEqual(file_drop, statements[3].kind.drop.function);
}

test "MIR storage keeps matching partial branch joins and rejects different field sets conservatively" {
    try std.testing.expectEqual(StorageState.partially_initialized, joinStorageStateWithFields(
        .partially_initialized,
        FieldStorageState{ .struct_type = .{ .index = 7 }, .fields = @constCast(&[_]StorageState{ .initialized, .uninitialized }) },
        .partially_initialized,
        FieldStorageState{ .struct_type = .{ .index = 7 }, .fields = @constCast(&[_]StorageState{ .initialized, .uninitialized }) },
    ));
    try std.testing.expectEqual(StorageState.maybe_initialized, joinStorageStateWithFields(
        .partially_initialized,
        FieldStorageState{ .struct_type = .{ .index = 7 }, .fields = @constCast(&[_]StorageState{ .initialized, .uninitialized }) },
        .partially_initialized,
        FieldStorageState{ .struct_type = .{ .index = 7 }, .fields = @constCast(&[_]StorageState{ .uninitialized, .initialized }) },
    ));
}

test "MIR storage inserts Drop cleanup in reverse local initialization order" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    const drop_fn = try addDropImplForTest(&semantic_module, file_type);
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const a = try module.store.addLocal(function, try interner_value.intern("a"), .user, file_type, hir.synthetic_span);
    const b = try module.store.addLocal(function, try interner_value.intern("b"), .user, file_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = b }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    const statements = module.store.getBlock(block).statements;
    try std.testing.expectEqual(@as(usize, 4), statements.len);
    try std.testing.expectEqual(b, statements[2].kind.drop.place.local);
    try std.testing.expectEqual(drop_fn, statements[2].kind.drop.function);
    try std.testing.expectEqual(a, statements[3].kind.drop.place.local);
}

test "MIR storage skips moved Drop local at cleanup" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    _ = try addDropImplForTest(&semantic_module, file_type);
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const f = try module.store.addLocal(function, try interner_value.intern("f"), .user, file_type, hir.synthetic_span);
    const g = try module.store.addLocal(function, try interner_value.intern("g"), .user, file_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = f }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = g }, mir.MirRvalue.movePlace(.{ .local = f })) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    const statements = module.store.getBlock(block).statements;
    try std.testing.expectEqual(@as(usize, 3), statements.len);
    try std.testing.expectEqual(g, statements[2].kind.drop.place.local);
}

test "MIR storage diagnoses maybe-moved Drop cleanup" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    _ = try addDropImplForTest(&semantic_module, file_type);
    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const cond = try module.store.addLocal(function, try interner_value.intern("cond"), .param, semantic_module.types.boolType(), hir.synthetic_span);
    const f = try module.store.addLocal(function, try interner_value.intern("f"), .user, file_type, hir.synthetic_span);
    const temp = try module.store.addLocal(function, null, .temp, file_type, hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const then_block = try module.store.addBlock(function, hir.synthetic_span);
    const join = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.appendStatement(entry, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = f }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.copyPlace(.{ .local = cond }), then_block, join) });
    try module.store.appendStatement(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, mir.MirRvalue.movePlace(.{ .local = f })) });
    try module.store.setTerminator(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(join) });
    try module.store.setTerminator(join, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.MaybeMovedUse, bag.diagnostics.items[0].code);
}

fn addCopyImplForTest(semantic_module: *semantics.SemanticModule, type_id: types.TypeId) !void {
    const copy_name = try semantic_module.interner.intern("Copy");
    const concept_id = try semantic_module.hir.addConcept(copy_name, true, false, hir.synthetic_span);
    semantic_module.hir.setConceptKnownMarkerKind(concept_id, .copy);
    const t_name = try semantic_module.interner.intern("T");
    const t_type = try semantic_module.types.addTypeParam(.{ .kind = .concept, .index = concept_id.index }, 0, t_name);
    const type_params = try std.testing.allocator.alloc(hir.HirTypeParam, 1);
    type_params[0] = .{ .name = t_name, .span = hir.synthetic_span, .type_id = t_type };
    semantic_module.hir.setConceptTypeParams(concept_id, type_params);
    _ = try semantic_module.hir.addConceptImpl(concept_id, type_id, &.{}, false, hir.synthetic_span);
}

test "assignment policy allows initialized Copy scalar replacement" {
    try std.testing.expectEqual(AssignmentDecision.allow_copy_replacement, canAssignToState(.initialized, .{ .is_copy = true, .has_drop = false }));
    try std.testing.expectEqual(AssignmentDecision.reject_replacement_required, canAssignToState(.initialized, .{ .is_copy = false, .has_drop = false }));
    try std.testing.expectEqual(AssignmentDecision.reject_replacement_required, canAssignToState(.initialized, .{ .is_copy = true, .has_drop = true }));
    try std.testing.expectEqual(AssignmentDecision.allow_reinitialization, canAssignToState(.moved, .{ .is_copy = false, .has_drop = true }));
    try std.testing.expectEqual(AssignmentDecision.allow_initialization, canAssignToState(.uninitialized, .{ .is_copy = false, .has_drop = true }));
    try std.testing.expectEqual(AssignmentDecision.reject_maybe_moved, canAssignToState(.maybe_moved, .{ .is_copy = true, .has_drop = false }));
    try std.testing.expectEqual(AssignmentDecision.reject_maybe_uninitialized, canAssignToState(.maybe_initialized, .{ .is_copy = true, .has_drop = false }));
}

test "MIR storage allows assignment replacement of initialized int and bool locals" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const x = try module.store.addLocal(function, try interner_value.intern("x"), .user, semantic_module.types.intType(), hir.synthetic_span);
    const flag = try module.store.addLocal(function, try interner_value.intern("flag"), .user, semantic_module.types.boolType(), hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = x }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "1"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = x }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "2"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = flag }, mir.MirRvalue.use_(mir.MirOperand.boolLiteral(false))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = flag }, mir.MirRvalue.use_(mir.MirOperand.boolLiteral(true))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = x })) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(x));
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(flag));
}

test "MIR storage rejects initialized non-Copy struct replacement" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const box_type = try semantic_module.types.addStructType(.{ .index = 0 });
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const a = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.AssignmentRequiresReplacement, bag.diagnostics.items[0].code);
}

test "MIR storage allows initialized Copy marker struct assignment" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const vec_type = try semantic_module.types.addStructType(.{ .index = 0 });
    try addCopyImplForTest(&semantic_module, vec_type);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const v = try module.store.addLocal(function, try interner_value.intern("v"), .user, vec_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = v }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = v }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(v));
}

test "MIR storage allows reinitialization after moving non-Copy struct" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const box_type = try semantic_module.types.addStructType(.{ .index = 0 });
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const a = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const b = try module.store.addLocal(function, try interner_value.intern("b"), .user, box_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = b }, mir.MirRvalue.movePlace(.{ .local = a })) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(a));
    try std.testing.expectEqual(StorageState.initialized, analysis.stateOf(b));
}

test "MIR storage rejects initialized Drop replacement and treats Drop as non-Copy" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    _ = try addDropImplForTest(&semantic_module, file_type);
    try addCopyImplForTest(&semantic_module, file_type);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const f = try module.store.addLocal(function, try interner_value.intern("f"), .user, file_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = f }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = f }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.AssignmentRequiresReplacement, bag.diagnostics.items[0].code);
}

test "ManualInit wrapper does not inherit contained Drop" {
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();

    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    _ = try addDropImplForTest(&semantic_module, file_type);
    const manual_file = try semantic_module.types.addManualInitType(file_type);

    try std.testing.expect(semantic_module.hasDrop(file_type) != null);
    try std.testing.expect(semantic_module.hasDrop(manual_file) == null);
    try std.testing.expect(!semantic_module.types.isCopyType(&semantic_module.hir, manual_file));
}

test "MIR storage skips ManualInit contained Drop but drops assumed value" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    const drop_fn = try addDropImplForTest(&semantic_module, file_type);
    const manual_file = try semantic_module.types.addManualInitType(file_type);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const slot = try module.store.addLocal(function, try interner_value.intern("slot"), .user, manual_file, hir.synthetic_span);
    const moved_slot = try module.store.addLocal(function, null, .temp, manual_file, hir.synthetic_span);
    const ready = try module.store.addLocal(function, try interner_value.intern("ready"), .user, file_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = slot }, mir.MirRvalue.use_(try mir.MirOperand.intLiteral(std.testing.allocator, "0"))) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = moved_slot }, mir.MirRvalue.movePlace(.{ .local = slot })) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = ready }, mir.MirRvalue.manualInitAssume(mir.MirOperand.copyPlace(.{ .local = moved_slot }))) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();

    try std.testing.expectEqual(StorageState.moved, analysis.stateOf(slot));
    const statements = module.store.getBlock(block).statements;
    try std.testing.expectEqual(@as(usize, 4), statements.len);
    try std.testing.expectEqual(ready, statements[3].kind.drop.place.local);
    try std.testing.expectEqual(drop_fn, statements[3].kind.drop.function);
}

test "MIR storage drops moved Drop local after reinitialization" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const file_type = try semantic_module.types.addStructType(try semantic_module.hir.addStruct(try semantic_module.interner.intern("File")));
    const drop_fn = try addDropImplForTest(&semantic_module, file_type);
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const a = try module.store.addLocal(function, try interner_value.intern("a"), .user, file_type, hir.synthetic_span);
    const b = try module.store.addLocal(function, try interner_value.intern("b"), .user, file_type, hir.synthetic_span);
    const block = try module.store.addBlock(function, hir.synthetic_span);
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = b }, mir.MirRvalue.movePlace(.{ .local = a })) });
    try module.store.appendStatement(block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var analysis = try analyzeFunction(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analysis.deinit();
    const statements = module.store.getBlock(block).statements;
    try std.testing.expectEqual(@as(usize, 5), statements.len);
    try std.testing.expectEqual(b, statements[3].kind.drop.place.local);
    try std.testing.expectEqual(drop_fn, statements[3].kind.drop.function);
    try std.testing.expectEqual(a, statements[4].kind.drop.place.local);
}

test "MIR storage rejects assignment to maybe-moved place" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const box_type = try semantic_module.types.addStructType(.{ .index = 0 });
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const cond = try module.store.addLocal(function, try interner_value.intern("cond"), .param, semantic_module.types.boolType(), hir.synthetic_span);
    const a = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const temp = try module.store.addLocal(function, null, .temp, box_type, hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const then_block = try module.store.addBlock(function, hir.synthetic_span);
    const join = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.appendStatement(entry, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.copyPlace(.{ .local = cond }), then_block, join) });
    try module.store.appendStatement(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = temp }, mir.MirRvalue.movePlace(.{ .local = a })) });
    try module.store.setTerminator(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(join) });
    try module.store.appendStatement(join, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(join, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.MaybeMovedUse, bag.diagnostics.items[0].code);
}

test "MIR storage rejects assignment to maybe-initialized place" {
    var interner_value = interner.Interner.init(std.testing.allocator);
    defer interner_value.deinit();
    var semantic_module = try semantics.SemanticModule.init(std.testing.allocator);
    defer semantic_module.deinit();
    const box_type = try semantic_module.types.addStructType(.{ .index = 0 });
    var module = mir.MirModule.init(std.testing.allocator);
    defer module.deinit();

    const function = try module.store.addFunction(.{ .index = 0 }, try interner_value.intern("main"), semantic_module.types.intType(), hir.synthetic_span);
    const cond = try module.store.addLocal(function, try interner_value.intern("cond"), .param, semantic_module.types.boolType(), hir.synthetic_span);
    const a = try module.store.addLocal(function, try interner_value.intern("a"), .user, box_type, hir.synthetic_span);
    const entry = try module.store.addBlock(function, hir.synthetic_span);
    const then_block = try module.store.addBlock(function, hir.synthetic_span);
    const join = try module.store.addBlock(function, hir.synthetic_span);

    try module.store.setTerminator(entry, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.switchBool(mir.MirOperand.copyPlace(.{ .local = cond }), then_block, join) });
    try module.store.appendStatement(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(then_block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.gotoBlock(join) });
    try module.store.appendStatement(join, .{ .span = hir.synthetic_span, .kind = mir.MirStatementKind.assignTo(.{ .local = a }, try mir.MirRvalue.structConstructor(std.testing.allocator, .{ .index = 0 }, &.{})) });
    try module.store.setTerminator(join, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(try mir.MirOperand.intLiteral(std.testing.allocator, "0")) });

    var bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidStorageState, analyzeModule(std.testing.allocator, &semantic_module, &module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.MaybeUninitializedUse, bag.diagnostics.items[0].code);
}
