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
};

pub const StorageDiagnostic = struct {
    kind: Kind,
    local: mir.MirLocalId,
    span: source.SourceSpan,

    pub const Kind = enum {
        use_before_initialization,
        use_after_move,
        implicit_copy_requires_copy,
    };
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
    mir_module: *const mir.MirModule,
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
    mir_module: *const mir.MirModule,
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
    mir_module: *const mir.MirModule,
    function_id: mir.MirFunctionId,
    function: mir.MirFunction,
    diagnostic_bag: ?*diagnostics.DiagnosticBag,
    diagnostics: std.ArrayList(StorageDiagnostic),
    had_error: bool,

    fn init(
        allocator: std.mem.Allocator,
        semantic_module: *const semantics.SemanticModule,
        mir_module: *const mir.MirModule,
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
        };
    }

    fn analyze(self: *Analyzer) AnalysisError!StorageAnalysis {
        errdefer self.diagnostics.deinit(self.allocator);

        const local_count = self.mir_module.store.locals.items.len;
        var initial_states = try self.allocator.alloc(StorageState, local_count);
        errdefer self.allocator.free(initial_states);
        @memset(initial_states, .initialized);

        for (self.function.locals) |local_id| {
            initial_states[local_id.index] = .uninitialized;
        }
        for (self.function.params) |param_id| {
            initial_states[param_id.index] = .initialized;
        }

        const block_count = self.function.blocks.len;
        var in_states = try self.allocator.alloc([]StorageState, block_count);
        defer {
            for (in_states) |states| {
                if (states.len > 0) self.allocator.free(states);
            }
            self.allocator.free(in_states);
        }
        for (in_states) |*states| states.* = &.{};

        var reachable = try self.allocator.alloc(bool, block_count);
        defer self.allocator.free(reachable);
        @memset(reachable, false);

        var queued = try self.allocator.alloc(bool, block_count);
        defer self.allocator.free(queued);
        @memset(queued, false);

        var worklist = std.ArrayList(usize).empty;
        defer worklist.deinit(self.allocator);

        if (block_count > 0) {
            in_states[0] = try cloneStates(self.allocator, initial_states);
            reachable[0] = true;
            queued[0] = true;
            try worklist.append(self.allocator, 0);
        }

        const final_states = initial_states;
        initial_states = &.{};
        var have_final = false;
        errdefer self.allocator.free(final_states);

        while (worklist.items.len > 0) {
            const block_index = worklist.pop().?;
            queued[block_index] = false;
            const block_id = self.function.blocks[block_index];
            const block = self.mir_module.store.getBlock(block_id);

            const states = try cloneStates(self.allocator, in_states[block_index]);
            defer self.allocator.free(states);

            for (block.statements) |statement| {
                try self.analyzeStatement(states, statement);
            }

            if (block.terminator) |terminator| {
                try self.analyzeTerminator(states, terminator);
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
                        @memcpy(final_states, states);
                        have_final = true;
                    },
                }
            } else {
                @memcpy(final_states, states);
                have_final = true;
            }
        }

        if (!have_final and block_count > 0) {
            @memcpy(final_states, in_states[0]);
        }

        if (self.had_error) {
            return error.InvalidStorageState;
        }

        const owned_diagnostics = try self.diagnostics.toOwnedSlice(self.allocator);

        return .{
            .allocator = self.allocator,
            .final_states = final_states,
            .diagnostics = owned_diagnostics,
        };
    }

    fn analyzeStatement(self: *Analyzer, states: []StorageState, statement: mir.MirStatement) AnalysisError!void {
        switch (statement.kind) {
            .assign => |assignment| {
                try self.readRvalue(states, assignment.rvalue, statement.span);
                try self.writePlace(states, assignment.place);
            },
        }
    }

    fn analyzeTerminator(self: *Analyzer, states: []StorageState, terminator: mir.MirTerminator) AnalysisError!void {
        switch (terminator.kind) {
            .goto, .@"unreachable" => {},
            .return_ => |maybe_operand| if (maybe_operand) |operand| try self.readOperand(states, operand, terminator.span),
            .switch_bool => |switch_bool| try self.readOperand(states, switch_bool.discriminant, terminator.span),
            .switch_int => |switch_int| try self.readOperand(states, switch_int.discriminant, terminator.span),
        }
    }

    fn readRvalue(self: *Analyzer, states: []StorageState, rvalue: mir.MirRvalue, span: ?source.SourceSpan) AnalysisError!void {
        switch (rvalue) {
            .use => |operand| try self.readOperand(states, operand, span),
            .move => |place| try self.movePlace(states, place, span),
            .unary => |unary| try self.readOperand(states, unary.operand, span),
            .address_of => |place| try self.readPlace(states, place, span),
            .deref => |operand| try self.readOperand(states, operand, span),
            .binary => |binary| {
                try self.readOperand(states, binary.left, span);
                try self.readOperand(states, binary.right, span);
            },
            .call => |call| for (call.args) |arg| try self.readOperand(states, arg, span),
            .enum_constructor => |constructor| for (constructor.args) |arg| try self.readOperand(states, arg, span),
            .struct_constructor => |constructor| for (constructor.fields) |field| try self.readOperand(states, field.value, span),
            .enum_tag => |operand| try self.readOperand(states, operand, span),
            .enum_payload_field => |payload| try self.readOperand(states, payload.enum_operand, span),
            .field_access => |field_access| try self.readOperand(states, field_access.receiver, span),
        }
    }

    fn readOperand(self: *Analyzer, states: []StorageState, operand: mir.MirOperand, span: ?source.SourceSpan) AnalysisError!void {
        switch (operand) {
            .int_literal, .bool_literal => {},
            .copy => |place| {
                try self.readPlace(states, place, span);
                try self.checkImplicitCopy(states, place, span);
            },
        }
    }

    fn readPlace(self: *Analyzer, states: []StorageState, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = switch (place) {
            .local => |local_id| local_id,
            .field => |field| field.base,
        };
        const state = states[local_id.index];
        switch (state) {
            .initialized => {},
            .uninitialized => try self.report(.use_before_initialization, local_id, span),
            .moved => try self.report(.use_after_move, local_id, span),
        }
    }

    fn writePlace(self: *Analyzer, states: []StorageState, place: mir.MirPlace) AnalysisError!void {
        _ = self;
        switch (place) {
            .local => |local_id| states[local_id.index] = .initialized,
            // P10-M1 tracks whole-local storage state only. Field writes require
            // the aggregate storage to be initialized before this pass sees them.
            .field => |field| states[field.base.index] = .initialized,
        }
    }

    fn movePlace(self: *Analyzer, states: []StorageState, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        try self.readPlace(states, place, span);
        const local_id = switch (place) {
            .local => |local_id| local_id,
            .field => |field| field.base,
        };
        if (!self.isCopyLocal(local_id)) {
            states[local_id.index] = .moved;
        }
    }

    fn isCopyLocal(self: *Analyzer, local_id: mir.MirLocalId) bool {
        const type_id = self.mir_module.store.getLocal(local_id).type_id;
        return self.semantic_module.types.isCopyType(&self.semantic_module.hir, type_id);
    }

    fn checkImplicitCopy(self: *Analyzer, states: []const StorageState, place: mir.MirPlace, span: ?source.SourceSpan) AnalysisError!void {
        const local_id = switch (place) {
            .local => |local_id| local_id,
            .field => return,
        };
        if (states[local_id.index] != .initialized) return;
        const local = self.mir_module.store.getLocal(local_id);
        switch (local.kind) {
            .param, .user => {},
            .temp => return,
        }
        if (!self.semantic_module.types.isCopyType(&self.semantic_module.hir, local.type_id)) {
            try self.report(.implicit_copy_requires_copy, local_id, span);
        }
    }

    fn propagate(
        self: *Analyzer,
        states: []const StorageState,
        target: mir.MirBlockId,
        in_states: [][]StorageState,
        reachable: []bool,
        queued: []bool,
        worklist: *std.ArrayList(usize),
    ) AnalysisError!void {
        const target_index = self.blockIndex(target) orelse return;
        if (!reachable[target_index]) {
            in_states[target_index] = try cloneStates(self.allocator, states);
            reachable[target_index] = true;
            if (!queued[target_index]) {
                queued[target_index] = true;
                try worklist.append(self.allocator, target_index);
            }
            return;
        }

        const changed = try joinStates(in_states[target_index], states);
        if (changed and !queued[target_index]) {
            queued[target_index] = true;
            try worklist.append(self.allocator, target_index);
        }
    }

    fn report(self: *Analyzer, kind: StorageDiagnostic.Kind, local_id: mir.MirLocalId, span: ?source.SourceSpan) AnalysisError!void {
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
                .implicit_copy_requires_copy => diagnostics.implicitCopyRequiresCopy(diagnostic_span),
            });
        }
        self.had_error = true;
    }

    fn blockIndex(self: *Analyzer, block_id: mir.MirBlockId) ?usize {
        for (self.function.blocks, 0..) |candidate, index| {
            if (candidate.index == block_id.index) return index;
        }
        return null;
    }
};

fn cloneStates(allocator: std.mem.Allocator, states: []const StorageState) ![]StorageState {
    const cloned = try allocator.alloc(StorageState, states.len);
    @memcpy(cloned, states);
    return cloned;
}

fn joinStates(existing: []StorageState, incoming: []const StorageState) !bool {
    var changed = false;
    for (existing, incoming) |*current, next| {
        const joined = joinState(current.*, next);
        if (joined != current.*) {
            current.* = joined;
            changed = true;
        }
    }
    return changed;
}

fn joinState(left: StorageState, right: StorageState) StorageState {
    if (left == right) return left;
    // P10-M1 deliberately has no maybe-state lattice yet. Existing source MIR
    // initializes locals at declaration, so mixed joins are kept permissive until
    // P10-M4 introduces MaybeInitialized/MaybeMoved diagnostics.
    if (left == .initialized or right == .initialized) return .initialized;
    return .uninitialized;
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

    var states = try std.testing.allocator.alloc(StorageState, module.store.locals.items.len);
    defer std.testing.allocator.free(states);
    @memset(states, .initialized);
    states[local.index] = .moved;

    var analyzer = Analyzer.init(std.testing.allocator, &semantic_module, &module, function, module.store.getFunction(function).*, null);
    defer analyzer.diagnostics.deinit(std.testing.allocator);
    try analyzer.readPlace(states, .{ .local = local }, hir.synthetic_span);
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
    try module.store.setTerminator(block, .{ .span = hir.synthetic_span, .kind = mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(.{ .local = out })) });

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
