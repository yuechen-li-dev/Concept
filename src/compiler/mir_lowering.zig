// ─────────────────────────────────────────────────────────────────────────────
// Imports and public API
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const mir = @import("mir.zig");
const mir_validator = @import("mir_validator.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");

pub const LoweringError = anyerror;

pub fn lowerModule(
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
) LoweringError!mir.MirModule {
    var lowerer = ModuleLowerer.init(allocator, semantic_module);
    return lowerer.lower();
}

// ─────────────────────────────────────────────────────────────────────────────
// Module lowering
// ─────────────────────────────────────────────────────────────────────────────

const ModuleLowerer = struct {
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
    mir_module: mir.MirModule,

    fn init(allocator: std.mem.Allocator, semantic_module: *semantics.SemanticModule) ModuleLowerer {
        return .{
            .allocator = allocator,
            .semantic_module = semantic_module,
            .mir_module = mir.MirModule.init(allocator),
        };
    }

    fn lower(self: *ModuleLowerer) LoweringError!mir.MirModule {
        errdefer self.mir_module.deinit();

        for (self.semantic_module.hir.functions.items, 0..) |function, index| {
            if (function.body == null) continue;
            const hir_function_id = hir.FunctionId{ .index = @intCast(index) };
            if (function.is_compile_time) continue;
            if (self.semantic_module.hir.isGenericFunction(hir_function_id)) continue;
            if (self.semantic_module.hir.isConceptWitnessFunction(hir_function_id) and !self.semantic_module.hir.isReferencedConceptWitnessFunction(hir_function_id)) continue;
            try self.lowerFunction(hir_function_id, function);
        }

        const lowered = self.mir_module;
        self.mir_module = undefined;
        return lowered;
    }

    fn lowerFunction(self: *ModuleLowerer, hir_function_id: hir.FunctionId, function: hir.HirFunction) LoweringError!void {
        const function_id = try self.mir_module.store.addFunction(hir_function_id, function.name, function.return_type, function.span);
        var function_lowerer = FunctionLowerer.init(self.allocator, self.semantic_module, &self.mir_module.store, function_id);
        defer function_lowerer.deinit();

        for (function.params) |param_id| {
            const param = self.semantic_module.hir.getParam(param_id);
            const local_id = try self.mir_module.store.addLocal(function_id, param.name, .param, param.type_id, param.span);
            try function_lowerer.param_map.put(param_id, local_id);
        }

        const entry = try self.mir_module.store.addBlock(function_id, function.span);
        const exit = if (function.body) |body| try function_lowerer.lowerStmt(body, entry) else entry;

        const block = self.mir_module.store.getBlock(exit);
        if (block.terminator == null) {
            try self.mir_module.store.setTerminator(exit, .{ .span = function.span, .kind = .@"unreachable" });
        }
    }
};

// ─────────────────────────────────────────────────────────────────────────────
// Function lowering state
// ─────────────────────────────────────────────────────────────────────────────

const LoweredExpr = struct {
    operand: mir.MirOperand,
    block: mir.MirBlockId,
};

const FunctionLowerer = struct {
    allocator: std.mem.Allocator,
    semantic_module: *semantics.SemanticModule,
    store: *mir.MirStore,
    function_id: mir.MirFunctionId,
    local_map: std.AutoHashMap(hir.LocalId, mir.MirLocalId),
    param_map: std.AutoHashMap(hir.ParamId, mir.MirLocalId),

    fn init(
        allocator: std.mem.Allocator,
        semantic_module: *semantics.SemanticModule,
        store: *mir.MirStore,
        function_id: mir.MirFunctionId,
    ) FunctionLowerer {
        return .{
            .allocator = allocator,
            .semantic_module = semantic_module,
            .store = store,
            .function_id = function_id,
            .local_map = std.AutoHashMap(hir.LocalId, mir.MirLocalId).init(allocator),
            .param_map = std.AutoHashMap(hir.ParamId, mir.MirLocalId).init(allocator),
        };
    }

    fn deinit(self: *FunctionLowerer) void {
        self.param_map.deinit();
        self.local_map.deinit();
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Statement lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerStmt(self: *FunctionLowerer, stmt_id: hir.StmtId, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        if (self.isTerminated(block_id)) return block_id;

        const stmt = self.semantic_module.hir.getStmt(stmt_id).*;
        switch (stmt.kind) {
            .block => |children| {
                var current = block_id;
                for (children) |child| {
                    if (self.isTerminated(current)) return current;
                    current = try self.lowerStmt(child, current);
                }
                return current;
            },
            .local_decl => |decl| {
                const local_id = try self.ensureLocal(decl.local);
                const lowered = try self.lowerExpr(decl.initializer, block_id);
                try self.store.appendStatement(lowered.block, .{
                    .span = stmt.span,
                    .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(local_id), mir.MirRvalue.use_(lowered.operand)),
                });
                return lowered.block;
            },
            .expr_stmt => |expr_id| {
                const lowered = try self.lowerExpr(expr_id, block_id);
                return lowered.block;
            },
            .discard_stmt => |expr_id| {
                const lowered = try self.lowerExpr(expr_id, block_id);
                return lowered.block;
            },
            .arena_reset => |op| return try self.lowerArenaStorageOp(op.arena_expr, block_id, stmt.span, .reset),
            .arena_destroy => |op| return try self.lowerArenaStorageOp(op.arena_expr, block_id, stmt.span, .destroy),
            .assignment => |assignment| {
                const place = try self.resolveAssignTarget(assignment.target);
                const lowered = try self.lowerExpr(assignment.value, block_id);
                try self.store.appendStatement(lowered.block, .{
                    .span = stmt.span,
                    .kind = mir.MirStatementKind.assignTo(place, mir.MirRvalue.use_(lowered.operand)),
                });
                return lowered.block;
            },
            .return_stmt => |maybe_value| {
                if (maybe_value) |expr_id| {
                    const lowered = try self.lowerExpr(expr_id, block_id);
                    try self.terminateBlock(lowered.block, stmt.span, mir.MirTerminatorKind.returnValue(lowered.operand));
                    return lowered.block;
                }
                try self.terminateBlock(block_id, stmt.span, mir.MirTerminatorKind.returnValue(null));
                return block_id;
            },
            .if_stmt => |if_stmt| return self.lowerIf(stmt, if_stmt, block_id),
            .while_stmt => |while_stmt| return self.lowerWhile(stmt, while_stmt, block_id),
            .unsafe_block => |body| return self.lowerStmt(body, block_id),
            .match_stmt => |match_stmt| return self.lowerMatch(stmt, match_stmt, block_id),
        }
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Structured control flow lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerIf(self: *FunctionLowerer, stmt: hir.HirStmt, if_stmt: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        const condition_lowered = try self.lowerExpr(if_stmt.condition, block_id);
        const condition = condition_lowered.operand;
        const switch_block = condition_lowered.block;
        const then_block = try self.newBlock(stmt.span);
        const join_block = try self.newBlock(stmt.span);
        const else_block = if (if_stmt.else_block != null) try self.newBlock(stmt.span) else join_block;

        try self.terminateBlock(switch_block, stmt.span, mir.MirTerminatorKind.switchBool(condition, then_block, else_block));

        const then_end = try self.lowerStmt(if_stmt.then_block, then_block);
        const then_falls_through = !self.isTerminated(then_end);
        try self.ensureGotoTo(then_end, join_block, stmt.span);

        var has_fallthrough = then_falls_through or if_stmt.else_block == null;
        if (if_stmt.else_block) |else_stmt_id| {
            const else_end = try self.lowerStmt(else_stmt_id, else_block);
            const else_falls_through = !self.isTerminated(else_end);
            try self.ensureGotoTo(else_end, join_block, stmt.span);
            has_fallthrough = has_fallthrough or else_falls_through;
        }

        if (!has_fallthrough) {
            try self.terminateBlock(join_block, stmt.span, .@"unreachable");
        }
        return join_block;
    }

    fn lowerWhile(self: *FunctionLowerer, stmt: hir.HirStmt, while_stmt: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        const condition_block = try self.newBlock(stmt.span);
        const body_block = try self.newBlock(stmt.span);
        const exit_block = try self.newBlock(stmt.span);

        try self.terminateBlock(block_id, stmt.span, mir.MirTerminatorKind.gotoBlock(condition_block));

        const condition_lowered = try self.lowerExpr(while_stmt.condition, condition_block);
        const condition = condition_lowered.operand;
        try self.terminateBlock(condition_lowered.block, stmt.span, mir.MirTerminatorKind.switchBool(condition, body_block, exit_block));

        const body_end = try self.lowerStmt(while_stmt.body, body_block);
        try self.ensureGotoTo(body_end, condition_block, stmt.span);

        return exit_block;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Match lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerMatch(self: *FunctionLowerer, stmt: hir.HirStmt, match_stmt: anytype, block_id: mir.MirBlockId) LoweringError!mir.MirBlockId {
        const scrutinee_lowered = try self.lowerExpr(match_stmt.scrutinee, block_id);
        const scrutinee = scrutinee_lowered.operand;
        const switch_block = scrutinee_lowered.block;
        const scrutinee_type = try self.inferExprType(match_stmt.scrutinee);
        const join_block = try self.newBlock(stmt.span);
        const arm_blocks = try self.allocator.alloc(mir.MirBlockId, match_stmt.arms.len);
        defer self.allocator.free(arm_blocks);

        for (match_stmt.arms, 0..) |_, index| {
            arm_blocks[index] = try self.newBlock(stmt.span);
        }

        var payload_scrutinee: ?mir.MirOperand = null;
        if (sameType(scrutinee_type, self.semantic_module.types.boolType())) {
            try self.terminateBlock(switch_block, stmt.span, try self.lowerBoolMatchTerminator(scrutinee, match_stmt.arms, arm_blocks, join_block));
        } else if (sameType(scrutinee_type, self.semantic_module.types.intType())) {
            try self.terminateBlock(switch_block, stmt.span, try self.lowerIntMatchTerminator(scrutinee, match_stmt.arms, arm_blocks, join_block));
        } else if (self.semantic_module.types.kind(scrutinee_type) == .enum_type) {
            payload_scrutinee = try scrutinee.clone(self.allocator);
            const tag_temp = try self.addTemp(self.semantic_module.types.intType());
            try self.store.appendStatement(switch_block, .{
                .span = stmt.span,
                .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(tag_temp), mir.MirRvalue.enumTag(scrutinee)),
            });
            const tag_operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(tag_temp));
            try self.terminateBlock(switch_block, stmt.span, try self.lowerEnumMatchTerminator(tag_operand, match_stmt.arms, arm_blocks, join_block));
        } else {
            return error.UnsupportedControlFlow;
        }

        var has_fallthrough = false;
        for (match_stmt.arms, 0..) |arm, index| {
            try self.lowerPatternBindings(arm.pattern, payload_scrutinee, arm_blocks[index], stmt.span);
            const arm_end = try self.lowerStmt(arm.body, arm_blocks[index]);
            if (!self.isTerminated(arm_end)) {
                has_fallthrough = true;
                try self.ensureGotoTo(arm_end, join_block, stmt.span);
            }
        }

        if (!has_fallthrough and !self.joinIsDefaultTarget(switch_block, join_block)) {
            try self.terminateBlock(join_block, stmt.span, .@"unreachable");
        }
        return join_block;
    }

    fn lowerPatternBindings(self: *FunctionLowerer, pattern: hir.HirMatchPattern, maybe_scrutinee: ?mir.MirOperand, block_id: mir.MirBlockId, span: ?hir.SourceSpan) LoweringError!void {
        switch (pattern) {
            .enum_variant => |enum_variant| {
                if (enum_variant.bindings.len == 0) return;
                const scrutinee = maybe_scrutinee orelse return error.UnsupportedControlFlow;
                for (enum_variant.bindings) |binding| {
                    const mapped = try self.ensureLocal(binding.local);
                    try self.store.appendStatement(block_id, .{
                        .span = span,
                        .kind = mir.MirStatementKind.assignTo(
                            mir.MirPlace.localPlace(mapped),
                            mir.MirRvalue.enumPayloadField(try scrutinee.clone(self.allocator), binding.payload_field),
                        ),
                    });
                }
            },
            else => {},
        }
    }

    fn lowerBoolMatchTerminator(
        self: *FunctionLowerer,
        scrutinee: mir.MirOperand,
        arms: []const hir.HirMatchArm,
        arm_blocks: []const mir.MirBlockId,
        default_target: mir.MirBlockId,
    ) LoweringError!mir.MirTerminatorKind {
        _ = self;
        var true_target: ?mir.MirBlockId = null;
        var false_target: ?mir.MirBlockId = null;
        var wildcard_target: ?mir.MirBlockId = null;

        for (arms, 0..) |arm, index| {
            switch (arm.pattern) {
                .bool_literal => |value| if (value) {
                    true_target = arm_blocks[index];
                } else {
                    false_target = arm_blocks[index];
                },
                .wildcard => wildcard_target = arm_blocks[index],
                .int_literal, .enum_variant => return error.UnsupportedControlFlow,
            }
        }

        const fallback = wildcard_target orelse default_target;
        return mir.MirTerminatorKind.switchBool(scrutinee, true_target orelse fallback, false_target orelse fallback);
    }

    fn lowerIntMatchTerminator(
        self: *FunctionLowerer,
        scrutinee: mir.MirOperand,
        arms: []const hir.HirMatchArm,
        arm_blocks: []const mir.MirBlockId,
        default_target: mir.MirBlockId,
    ) LoweringError!mir.MirTerminatorKind {
        var cases = std.ArrayList(mir.MirSwitchIntCase).empty;
        defer cases.deinit(self.allocator);
        var fallback = default_target;

        for (arms, 0..) |arm, index| {
            switch (arm.pattern) {
                .int_literal => |text| try cases.append(self.allocator, .{ .value = text, .target = arm_blocks[index] }),
                .wildcard => fallback = arm_blocks[index],
                .bool_literal, .enum_variant => return error.UnsupportedControlFlow,
            }
        }

        return try mir.MirTerminatorKind.switchInt(self.allocator, scrutinee, cases.items, fallback);
    }

    fn lowerEnumMatchTerminator(
        self: *FunctionLowerer,
        tag_operand: mir.MirOperand,
        arms: []const hir.HirMatchArm,
        arm_blocks: []const mir.MirBlockId,
        default_target: mir.MirBlockId,
    ) LoweringError!mir.MirTerminatorKind {
        var cases = std.ArrayList(mir.MirSwitchIntCase).empty;
        defer {
            for (cases.items) |case| self.allocator.free(case.value);
            cases.deinit(self.allocator);
        }
        var fallback = default_target;

        for (arms, 0..) |arm, index| {
            switch (arm.pattern) {
                .enum_variant => |pattern| {
                    const tag_value = try std.fmt.allocPrint(self.allocator, "{d}", .{try self.variantTag(pattern.enum_id, pattern.variant_id)});
                    try cases.append(self.allocator, .{ .value = tag_value, .target = arm_blocks[index] });
                },
                .wildcard => fallback = arm_blocks[index],
                .int_literal, .bool_literal => return error.UnsupportedControlFlow,
            }
        }

        return try mir.MirTerminatorKind.switchInt(self.allocator, tag_operand, cases.items, fallback);
    }

    fn variantTag(self: *FunctionLowerer, enum_id: hir.EnumId, variant_id: hir.VariantId) LoweringError!usize {
        const enum_decl = self.semantic_module.hir.getEnum(enum_id);
        for (enum_decl.variants, 0..) |candidate, index| {
            if (candidate.index == variant_id.index) return index;
        }
        return error.UnsupportedControlFlow;
    }

    fn joinIsDefaultTarget(self: *FunctionLowerer, switch_block: mir.MirBlockId, join_block: mir.MirBlockId) bool {
        const terminator = self.store.getBlock(switch_block).terminator orelse return false;
        return switch (terminator.kind) {
            .switch_bool => |switch_bool| switch_bool.true_target.index == join_block.index or switch_bool.false_target.index == join_block.index,
            .switch_int => |switch_int| switch_int.default_target.index == join_block.index,
            else => false,
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Expression lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerExpr(self: *FunctionLowerer, expr_id: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => |text| .{ .operand = try mir.MirOperand.intLiteral(self.allocator, text), .block = block_id },
            .bool_literal => |value| .{ .operand = mir.MirOperand.boolLiteral(value), .block = block_id },
            .local_ref => |local_id| .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(try self.resolveLocal(local_id))), .block = block_id },
            .param_ref => |param_id| .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(try self.resolveParam(param_id))), .block = block_id },
            .group => |inner| try self.lowerExpr(inner, block_id),
            .compile_time => try self.lowerCompileTime(expr_id, block_id),
            .unary => |unary| try self.lowerUnary(expr, unary, block_id),
            .address_of => |operand| try self.lowerAddressOf(expr, operand, block_id),
            .deref => |operand| try self.lowerDeref(expr, operand, block_id),
            .move_expr => |operand| try self.lowerMove(expr, operand, block_id),
            .manual_init_assume => |slot| try self.lowerManualInitAssume(expr, slot, block_id),
            .binary => |binary| try self.lowerBinary(expr, binary, block_id),
            .call => |call| try self.lowerCall(expr, call, block_id),
            .arena_alloc => |arena_alloc| try self.lowerArenaAlloc(expr, arena_alloc, block_id),
            .concept_requirement_call, .target_metadata, .test_intrinsic => error.InvalidMirLowering,
            .enum_constructor => |constructor| try self.lowerEnumConstructor(expr, constructor, block_id),
            .struct_literal => |literal| try self.lowerStructLiteral(expr, literal, block_id),
            .field_access => |field_access| try self.lowerFieldAccess(expr, field_access, block_id),
            .try_expr => |operand| try self.lowerTry(expr, operand, block_id),
            .decide => |decide| try self.lowerDecide(expr, decide, block_id),
        };
    }

    fn lowerCompileTime(self: *FunctionLowerer, expr_id: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const value = self.semantic_module.compile_time_values.get(expr_id) orelse return error.InvalidMirLowering;
        return switch (value) {
            .int => blk: {
                const text = try value.toIntLiteral(self.allocator);
                defer self.allocator.free(text);
                break :blk .{ .operand = try mir.MirOperand.intLiteral(self.allocator, text), .block = block_id };
            },
            .bool => |bool_value| .{ .operand = mir.MirOperand.boolLiteral(bool_value), .block = block_id },
        };
    }

    fn lowerUnary(self: *FunctionLowerer, expr: hir.HirExpr, unary: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const lowered = try self.lowerExpr(unary.operand, block_id);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.unaryOp(lowerUnaryOp(unary.op), lowered.operand),
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = lowered.block };
    }

    fn lowerAddressOf(self: *FunctionLowerer, expr: hir.HirExpr, operand: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const place = try self.lowerAddressablePlace(operand);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(temp), mir.MirRvalue.addressOf(place)),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = block_id };
    }

    fn lowerDeref(self: *FunctionLowerer, expr: hir.HirExpr, operand: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const lowered = try self.lowerExpr(operand, block_id);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(temp), mir.MirRvalue.dereference(lowered.operand)),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = lowered.block };
    }

    fn lowerMove(self: *FunctionLowerer, expr: hir.HirExpr, operand: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const place = try self.lowerMovablePlace(operand);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(temp), mir.MirRvalue.movePlace(place)),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = block_id };
    }

    fn lowerManualInitAssume(self: *FunctionLowerer, expr: hir.HirExpr, slot: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const lowered = try self.lowerExpr(slot, block_id);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.manualInitAssume(lowered.operand),
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = lowered.block };
    }

    fn lowerMovablePlace(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!mir.MirPlace {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| mir.MirPlace.localPlace(try self.resolveLocal(local_id)),
            .param_ref => |param_id| mir.MirPlace.localPlace(try self.resolveParam(param_id)),
            .group => |inner| try self.lowerMovablePlace(inner),
            else => error.InvalidMirLowering,
        };
    }

    fn lowerAddressablePlace(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!mir.MirPlace {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| mir.MirPlace.localPlace(try self.resolveLocal(local_id)),
            .param_ref => |param_id| mir.MirPlace.localPlace(try self.resolveParam(param_id)),
            .group => |inner| try self.lowerAddressablePlace(inner),
            .field_access => |field_access| mir.MirPlace.fieldPlace(try self.lowerAddressableBase(field_access.receiver), try self.resolveFieldAccess(field_access.receiver, field_access.field_name)),
            else => error.InvalidMirLowering,
        };
    }

    fn lowerAddressableBase(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!mir.MirLocalId {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| try self.resolveLocal(local_id),
            .param_ref => |param_id| try self.resolveParam(param_id),
            .group => |inner| try self.lowerAddressableBase(inner),
            else => error.InvalidMirLowering,
        };
    }

    fn lowerBinary(self: *FunctionLowerer, expr: hir.HirExpr, binary: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const left = try self.lowerExpr(binary.left, block_id);
        const right = try self.lowerExpr(binary.right, left.block);
        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        try self.store.appendStatement(right.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.binaryOp(lowerBinaryOp(binary.op), left.operand, right.operand),
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = right.block };
    }

    fn lowerCall(self: *FunctionLowerer, expr: hir.HirExpr, call: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const args = try self.allocator.alloc(mir.MirOperand, call.args.len);
        var args_owned = true;
        var initialized: usize = 0;
        errdefer if (args_owned) deinitInitializedOperands(self.allocator, args, initialized);

        var current = block_id;
        for (call.args) |arg_expr| {
            const lowered = try self.lowerExpr(arg_expr, current);
            args[initialized] = lowered.operand;
            initialized += 1;
            current = lowered.block;
        }

        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        args_owned = false;
        try self.store.appendStatement(current, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                .{ .call = .{ .function = call.function, .args = args } },
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = current };
    }

    fn lowerArenaAlloc(self: *FunctionLowerer, expr: hir.HirExpr, arena_alloc: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const arena_lowered = try self.lowerExpr(arena_alloc.arena_expr, block_id);
        const temp = try self.addTemp(arena_alloc.result_type);
        try self.store.appendStatement(arena_lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.arenaAlloc(arena_lowered.operand, arena_alloc.allocated_type, arena_alloc.result_type),
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = arena_lowered.block };
    }

    const ArenaStorageOpKind = enum { reset, destroy };

    fn lowerArenaStorageOp(self: *FunctionLowerer, arena_expr: hir.ExprId, block_id: mir.MirBlockId, span: ?hir.SourceSpan, kind: ArenaStorageOpKind) LoweringError!mir.MirBlockId {
        const arena_lowered = try self.lowerExpr(arena_expr, block_id);
        try self.store.appendStatement(arena_lowered.block, .{
            .span = span,
            .kind = switch (kind) {
                .reset => mir.MirStatementKind.arenaReset(arena_lowered.operand),
                .destroy => mir.MirStatementKind.arenaDestroy(arena_lowered.operand),
            },
        });
        return arena_lowered.block;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Enum / Result / Try lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerEnumConstructor(self: *FunctionLowerer, expr: hir.HirExpr, constructor: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const args = try self.allocator.alloc(mir.MirOperand, constructor.args.len);
        var args_owned = true;
        var initialized: usize = 0;
        errdefer if (args_owned) deinitInitializedOperands(self.allocator, args, initialized);

        var current = block_id;
        for (constructor.args) |arg_expr| {
            const lowered = try self.lowerExpr(arg_expr, current);
            args[initialized] = lowered.operand;
            initialized += 1;
            current = lowered.block;
        }

        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        args_owned = false;
        try self.store.appendStatement(current, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                .{ .enum_constructor = .{ .enum_id = constructor.enum_id, .variant_id = constructor.variant_id, .args = args } },
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = current };
    }

    fn lowerFieldAccess(self: *FunctionLowerer, expr: hir.HirExpr, field_access: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const lowered = try self.lowerExpr(field_access.receiver, block_id);
        const field_id = try self.resolveFieldAccess(field_access.receiver, field_access.field_name);
        const temp = try self.addTemp(self.semantic_module.hir.getField(field_id).type_id);
        try self.store.appendStatement(lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                mir.MirRvalue.fieldAccess(lowered.operand, field_id),
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = lowered.block };
    }

    fn lowerStructLiteral(self: *FunctionLowerer, expr: hir.HirExpr, literal: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const fields = try self.allocator.alloc(mir.MirStructFieldValue, literal.fields.len);
        var fields_owned = true;
        var initialized: usize = 0;
        errdefer if (fields_owned) self.allocator.free(fields);

        var current = block_id;
        for (literal.fields) |field| {
            const lowered = try self.lowerExpr(field.value, current);
            fields[initialized] = .{ .field_id = field.field_id, .value = lowered.operand };
            initialized += 1;
            current = lowered.block;
        }

        const temp = try self.addTemp(try self.inferExprTypeFrom(expr));
        fields_owned = false;
        try self.store.appendStatement(current, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(temp),
                .{ .struct_constructor = .{ .struct_id = literal.struct_id, .fields = fields } },
            ),
        });
        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(temp)), .block = current };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Decide lowering
    // ─────────────────────────────────────────────────────────────────────────────

    fn lowerDecide(self: *FunctionLowerer, expr: hir.HirExpr, decide: anytype, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const bool_type = self.semantic_module.types.boolType();
        const int_type = self.semantic_module.types.intType();
        const has_winner = try self.addNamedTemp("hasWinner", bool_type, expr.span);
        const best_score = try self.addNamedTemp("bestScore", int_type, expr.span);
        const best_value = try self.addNamedTemp("bestValue", decide.enum_type, expr.span);

        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(has_winner), mir.MirRvalue.use_(mir.MirOperand.boolLiteral(false))),
        });
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(best_score), mir.MirRvalue.use_(try mir.MirOperand.intLiteral(self.allocator, "0"))),
        });

        const initial_variant = self.firstUnconditionalVariant(decide.arms) orelse return error.UnsupportedExpression;
        try self.store.appendStatement(block_id, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(best_value),
                .{ .enum_constructor = .{ .enum_id = decide.enum_id, .variant_id = initial_variant, .args = &.{} } },
            ),
        });

        var current = block_id;
        for (decide.arms) |arm| {
            const score_block = if (arm.condition) |condition_expr| blk: {
                const condition_lowered = try self.lowerExpr(condition_expr, current);
                const eligible_block = try self.newBlock(arm.span);
                const next_arm_block = try self.newBlock(arm.span);
                try self.terminateBlock(condition_lowered.block, arm.span, mir.MirTerminatorKind.switchBool(condition_lowered.operand, eligible_block, next_arm_block));
                current = next_arm_block;
                break :blk eligible_block;
            } else current;

            const after_arm_block = if (arm.condition == null) try self.newBlock(arm.span) else current;
            const score_lowered = try self.lowerExpr(arm.score, score_block);
            const score_temp = try self.addNamedTemp("scoreTemp", int_type, arm.span);
            try self.store.appendStatement(score_lowered.block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(score_temp), mir.MirRvalue.use_(score_lowered.operand)),
            });

            const missing_winner = try self.addTemp(bool_type);
            try self.store.appendStatement(score_lowered.block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(
                    mir.MirPlace.localPlace(missing_winner),
                    mir.MirRvalue.unaryOp(.logical_not, mir.MirOperand.copyPlace(mir.MirPlace.localPlace(has_winner))),
                ),
            });
            const score_beats_best = try self.addTemp(bool_type);
            try self.store.appendStatement(score_lowered.block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(
                    mir.MirPlace.localPlace(score_beats_best),
                    mir.MirRvalue.binaryOp(
                        .greater,
                        mir.MirOperand.copyPlace(mir.MirPlace.localPlace(score_temp)),
                        mir.MirOperand.copyPlace(mir.MirPlace.localPlace(best_score)),
                    ),
                ),
            });
            const should_update = try self.addNamedTemp("shouldUpdate", bool_type, arm.span);
            try self.store.appendStatement(score_lowered.block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(
                    mir.MirPlace.localPlace(should_update),
                    mir.MirRvalue.binaryOp(
                        .logical_or,
                        mir.MirOperand.copyPlace(mir.MirPlace.localPlace(missing_winner)),
                        mir.MirOperand.copyPlace(mir.MirPlace.localPlace(score_beats_best)),
                    ),
                ),
            });

            const update_block = try self.newBlock(arm.span);
            try self.terminateBlock(score_lowered.block, arm.span, mir.MirTerminatorKind.switchBool(
                mir.MirOperand.copyPlace(mir.MirPlace.localPlace(should_update)),
                update_block,
                after_arm_block,
            ));
            try self.store.appendStatement(update_block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(has_winner), mir.MirRvalue.use_(mir.MirOperand.boolLiteral(true))),
            });
            try self.store.appendStatement(update_block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(
                    mir.MirPlace.localPlace(best_score),
                    mir.MirRvalue.use_(mir.MirOperand.copyPlace(mir.MirPlace.localPlace(score_temp))),
                ),
            });
            try self.store.appendStatement(update_block, .{
                .span = arm.span,
                .kind = mir.MirStatementKind.assignTo(
                    mir.MirPlace.localPlace(best_value),
                    .{ .enum_constructor = .{ .enum_id = decide.enum_id, .variant_id = arm.variant_id, .args = &.{} } },
                ),
            });
            try self.terminateBlock(update_block, arm.span, mir.MirTerminatorKind.gotoBlock(after_arm_block));
            current = after_arm_block;
        }

        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(best_value)), .block = current };
    }

    fn firstUnconditionalVariant(self: *FunctionLowerer, arms: []const hir.HirDecideArm) ?hir.VariantId {
        _ = self;
        for (arms) |arm| {
            if (arm.condition == null) return arm.variant_id;
        }
        return null;
    }

    fn lowerTry(self: *FunctionLowerer, expr: hir.HirExpr, operand_expr: hir.ExprId, block_id: mir.MirBlockId) LoweringError!LoweredExpr {
        const operand_lowered = try self.lowerExpr(operand_expr, block_id);
        const result_type = try self.inferExprType(operand_expr);
        const shape = self.semantic_module.resultShapeForType(result_type) orelse return error.MissingResultShape;
        const result_temp = try self.addTemp(result_type);
        try self.store.appendStatement(operand_lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(result_temp), mir.MirRvalue.use_(operand_lowered.operand)),
        });
        const tag_temp = try self.addTemp(self.semantic_module.types.intType());
        const result_copy = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(result_temp));
        try self.store.appendStatement(operand_lowered.block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(mir.MirPlace.localPlace(tag_temp), mir.MirRvalue.enumTag(result_copy)),
        });

        const ok_block = try self.newBlock(expr.span);
        const err_block = try self.newBlock(expr.span);
        const cont_block = try self.newBlock(expr.span);
        const ok_tag = try std.fmt.allocPrint(self.allocator, "{d}", .{try self.variantTag(self.semantic_module.types.kind(result_type).enum_type, shape.ok_variant)});
        defer self.allocator.free(ok_tag);
        const cases = [_]mir.MirSwitchIntCase{.{ .value = ok_tag, .target = ok_block }};
        try self.terminateBlock(operand_lowered.block, expr.span, try mir.MirTerminatorKind.switchInt(
            self.allocator,
            mir.MirOperand.copyPlace(mir.MirPlace.localPlace(tag_temp)),
            &cases,
            err_block,
        ));

        const ok_temp = try self.addTemp(shape.ok_type);
        try self.store.appendStatement(ok_block, .{
            .span = expr.span,
            .kind = mir.MirStatementKind.assignTo(
                mir.MirPlace.localPlace(ok_temp),
                mir.MirRvalue.enumPayloadField(mir.MirOperand.copyPlace(mir.MirPlace.localPlace(result_temp)), shape.ok_payload),
            ),
        });
        try self.terminateBlock(ok_block, expr.span, mir.MirTerminatorKind.gotoBlock(cont_block));
        try self.terminateBlock(err_block, expr.span, mir.MirTerminatorKind.returnValue(mir.MirOperand.copyPlace(mir.MirPlace.localPlace(result_temp))));

        return .{ .operand = mir.MirOperand.copyPlace(mir.MirPlace.localPlace(ok_temp)), .block = cont_block };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Type inference helpers
    // ─────────────────────────────────────────────────────────────────────────────

    fn inferExprType(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!types.TypeId {
        return self.inferExprTypeFrom(self.semantic_module.hir.getExpr(expr_id).*);
    }

    fn inferExprTypeFrom(self: *FunctionLowerer, expr: hir.HirExpr) LoweringError!types.TypeId {
        return switch (expr.kind) {
            .int_literal => self.semantic_module.types.intType(),
            .bool_literal => self.semantic_module.types.boolType(),
            .local_ref => |local_id| self.semantic_module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.semantic_module.hir.getParam(param_id).type_id,
            .group => |inner| try self.inferExprType(inner),
            .compile_time => |compile_time_expr| try self.inferExprType(compile_time_expr.operand),
            .unary => |unary| switch (unary.op) {
                .negate => self.semantic_module.types.intType(),
                .logical_not => self.semantic_module.types.boolType(),
            },
            .address_of => |operand| try self.semantic_module.types.addPointerType(try self.inferAddressableExprType(operand)),
            .move_expr => |operand| try self.inferMovableExprType(operand),
            .manual_init_assume => |slot| blk: {
                const slot_type = try self.inferExprType(slot);
                break :blk self.semantic_module.types.manualInitPayload(slot_type) orelse return error.InvalidMirLowering;
            },
            .deref => |operand| blk: {
                const operand_type = try self.inferExprType(operand);
                break :blk switch (self.semantic_module.types.kind(operand_type)) {
                    .pointer => |pointer| pointer.pointee,
                    else => error.InvalidMirLowering,
                };
            },
            .binary => |binary| switch (binary.op) {
                .multiply, .divide, .modulo, .add, .subtract => self.semantic_module.types.intType(),
                .less, .less_equal, .greater, .greater_equal, .equal_equal, .bang_equal, .logical_and, .logical_or => self.semantic_module.types.boolType(),
            },
            .call => |call| self.semantic_module.hir.getFunction(call.function).return_type,
            .arena_alloc => |arena_alloc| arena_alloc.result_type,
            .concept_requirement_call, .test_intrinsic => error.InvalidMirLowering,
            .target_metadata => |metadata| metadata.query.typeOf(self.semantic_module.types),
            .enum_constructor => |constructor| try self.enumType(constructor.enum_id),
            .struct_literal => |literal| literal.type_id,
            .field_access => |field_access| self.semantic_module.hir.getField(try self.resolveFieldAccess(field_access.receiver, field_access.field_name)).type_id,
            .decide => |decide| decide.enum_type,
            .try_expr => |operand| blk: {
                const operand_type = try self.inferExprType(operand);
                const shape = self.semantic_module.resultShapeForType(operand_type) orelse return error.MissingResultShape;
                break :blk shape.ok_type;
            },
        };
    }

    fn resolveFieldAccess(self: *FunctionLowerer, receiver: hir.ExprId, field_name: hir.SymbolId) LoweringError!hir.FieldId {
        const receiver_type = try self.inferExprType(receiver);
        const kind = self.semantic_module.types.kind(receiver_type);
        if (kind != .struct_type) return error.InvalidMirLowering;
        const struct_decl = self.semantic_module.hir.getStruct(kind.struct_type);
        for (struct_decl.fields) |field_id| {
            const field = self.semantic_module.hir.getField(field_id);
            if (field.name.index == field_name.index) return field_id;
        }
        return error.InvalidMirLowering;
    }

    fn inferAddressableExprType(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!types.TypeId {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| self.semantic_module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.semantic_module.hir.getParam(param_id).type_id,
            .group => |inner| try self.inferAddressableExprType(inner),
            .field_access => |field_access| self.semantic_module.hir.getField(try self.resolveFieldAccess(field_access.receiver, field_access.field_name)).type_id,
            else => error.InvalidMirLowering,
        };
    }

    fn inferMovableExprType(self: *FunctionLowerer, expr_id: hir.ExprId) LoweringError!types.TypeId {
        const expr = self.semantic_module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| self.semantic_module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.semantic_module.hir.getParam(param_id).type_id,
            .group => |inner| try self.inferMovableExprType(inner),
            else => error.InvalidMirLowering,
        };
    }

    fn structType(self: *FunctionLowerer, struct_id: hir.StructId) LoweringError!types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .struct_type and kind.struct_type.index == struct_id.index) return .{ .index = @intCast(index) };
        }
        return error.MissingStructType;
    }

    fn enumType(self: *FunctionLowerer, enum_id: hir.EnumId) LoweringError!types.TypeId {
        for (self.semantic_module.types.types.items, 0..) |kind, index| {
            if (kind == .enum_type and kind.enum_type.index == enum_id.index) return .{ .index = @intCast(index) };
        }
        return error.MissingEnumType;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Local/parameter resolution helpers
    // ─────────────────────────────────────────────────────────────────────────────

    fn ensureLocal(self: *FunctionLowerer, local_id: hir.LocalId) LoweringError!mir.MirLocalId {
        if (self.local_map.get(local_id)) |mapped| return mapped;
        const local = self.semantic_module.hir.getLocal(local_id);
        const mapped = try self.store.addLocal(self.function_id, local.name, .user, local.type_id, local.span);
        try self.local_map.put(local_id, mapped);
        return mapped;
    }

    fn resolveLocal(self: *FunctionLowerer, local_id: hir.LocalId) LoweringError!mir.MirLocalId {
        return self.local_map.get(local_id) orelse error.MissingLocalMapping;
    }

    fn resolveParam(self: *FunctionLowerer, param_id: hir.ParamId) LoweringError!mir.MirLocalId {
        return self.param_map.get(param_id) orelse error.MissingParamMapping;
    }

    fn resolveAssignTarget(self: *FunctionLowerer, target: hir.AssignTarget) LoweringError!mir.MirPlace {
        return switch (target) {
            .local => |local_id| mir.MirPlace.localPlace(try self.resolveLocal(local_id)),
            .param => |param_id| mir.MirPlace.localPlace(try self.resolveParam(param_id)),
            .field => |field| mir.MirPlace.fieldPlace(try self.resolveAssignBase(field.base), field.field_id),
        };
    }

    fn resolveAssignBase(self: *FunctionLowerer, base: hir.AssignBase) LoweringError!mir.MirLocalId {
        return switch (base) {
            .local => |local_id| try self.resolveLocal(local_id),
            .param => |param_id| try self.resolveParam(param_id),
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // MIR construction helpers
    // ─────────────────────────────────────────────────────────────────────────────

    fn addTemp(self: *FunctionLowerer, type_id: types.TypeId) LoweringError!mir.MirLocalId {
        return self.store.addLocal(self.function_id, null, .temp, type_id, null);
    }

    fn addNamedTemp(self: *FunctionLowerer, name: []const u8, type_id: types.TypeId, span: ?hir.SourceSpan) LoweringError!mir.MirLocalId {
        return self.store.addLocal(self.function_id, try self.semantic_module.interner.intern(name), .temp, type_id, span);
    }

    fn newBlock(self: *FunctionLowerer, span: ?hir.SourceSpan) LoweringError!mir.MirBlockId {
        return self.store.addBlock(self.function_id, span);
    }

    fn terminateBlock(self: *FunctionLowerer, block_id: mir.MirBlockId, span: ?hir.SourceSpan, kind: mir.MirTerminatorKind) LoweringError!void {
        try self.store.setTerminator(block_id, .{ .span = span, .kind = kind });
    }

    fn ensureGotoTo(self: *FunctionLowerer, block_id: mir.MirBlockId, target: mir.MirBlockId, span: ?hir.SourceSpan) LoweringError!void {
        if (!self.isTerminated(block_id)) {
            try self.terminateBlock(block_id, span, mir.MirTerminatorKind.gotoBlock(target));
        }
    }

    fn isTerminated(self: *FunctionLowerer, block_id: mir.MirBlockId) bool {
        return self.store.getBlock(block_id).terminator != null;
    }
};

fn lowerUnaryOp(op: hir.UnaryOp) mir.MirUnaryOp {
    return switch (op) {
        .negate => .negate,
        .logical_not => .logical_not,
    };
}

fn lowerBinaryOp(op: hir.BinaryOp) mir.MirBinaryOp {
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

fn sameType(left: types.TypeId, right: types.TypeId) bool {
    return left.index == right.index;
}

fn deinitOperand(allocator: std.mem.Allocator, operand: mir.MirOperand) void {
    switch (operand) {
        .int_literal => |text| allocator.free(text),
        else => {},
    }
}

fn deinitInitializedOperands(allocator: std.mem.Allocator, operands: []mir.MirOperand, initialized: usize) void {
    for (operands[0..initialized]) |operand| deinitOperand(allocator, operand);
    if (operands.len > 0) allocator.free(operands);
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

test "MIR lowering debug snapshot includes enum tag switch" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Status"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const ok = try module.hir.addVariant(enum_id, try intern(&module, "Ok"), hir.synthetic_span);
    const err = try module.hir.addVariant(enum_id, try intern(&module, "Err"), hir.synthetic_span);
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const status = try addLocal(&module, main, "status", enum_type);
    const status_init = try module.hir.addExpr(.{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok, .args = &.{} } }, hir.synthetic_span);
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = status, .initializer = status_init } }, hir.synthetic_span);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .enum_variant = .{ .enum_id = enum_id, .variant_id = ok, .bindings = &.{} } }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "7") }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .{ .enum_variant = .{ .enum_id = enum_id, .variant_id = err, .bindings = &.{} } }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try module.hir.addExpr(.{ .local_ref = status }, hir.synthetic_span), .arms = arms } }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumTag(Copy(MirLocalId(0)))") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "SwitchInt Copy(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "[0: MirBlockId") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "1: MirBlockId") != null);
}

test "MIR lowering debug snapshot routes enum wildcard as default" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Status"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const ok = try module.hir.addVariant(enum_id, try intern(&module, "Ok"), hir.synthetic_span);
    _ = try module.hir.addVariant(enum_id, try intern(&module, "Err"), hir.synthetic_span);
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const status = try addLocal(&module, main, "status", enum_type);
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = status, .initializer = try module.hir.addExpr(.{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok, .args = &.{} } }, hir.synthetic_span) } }, hir.synthetic_span);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .enum_variant = .{ .enum_id = enum_id, .variant_id = ok, .bindings = &.{} } }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "7") }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try module.hir.addExpr(.{ .local_ref = status }, hir.synthetic_span), .arms = arms } }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "SwitchInt Copy(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "[0: MirBlockId") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "] default: MirBlockId") != null);
}

test "MIR lowering debug snapshot includes enum payload extraction" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "ParseResult"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const ok = try module.hir.addVariant(enum_id, try intern(&module, "Ok"), hir.synthetic_span);
    const value_field = try module.hir.addEnumPayloadField(ok, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    const err = try module.hir.addVariant(enum_id, try intern(&module, "Err"), hir.synthetic_span);
    _ = try module.hir.addEnumPayloadField(err, try intern(&module, "code"), module.types.intType(), hir.synthetic_span);
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const result = try addLocal(&module, main, "result", enum_type);
    const ctor_arg = try intExpr(&module, "7");
    const ctor_args = try std.testing.allocator.alloc(hir.ExprId, 1);
    ctor_args[0] = ctor_arg;
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = result, .initializer = try module.hir.addExpr(.{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok, .args = ctor_args } }, hir.synthetic_span) } }, hir.synthetic_span);
    const value = try addLocal(&module, main, "value", module.types.intType());
    const bindings = try std.testing.allocator.alloc(hir.HirPatternBinding, 1);
    bindings[0] = .{ .name = try intern(&module, "value"), .local = value, .payload_field = value_field, .type_id = module.types.intType(), .span = hir.synthetic_span };
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .enum_variant = .{ .enum_id = enum_id, .variant_id = ok, .bindings = bindings } }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = value }, hir.synthetic_span) }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try module.hir.addExpr(.{ .local_ref = result }, hir.synthetic_span), .arms = arms } }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumPayloadField(Copy(MirLocalId(0)), EnumPayloadFieldId(0))") != null);
}

test "MIR lowering lowers address-of and deref" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const int_ptr = try module.types.addPointerType(module.types.intType());
    const x = try addLocal(&module, main, "x", module.types.intType());
    const p = try addLocal(&module, main, "p", int_ptr);
    const decl_x = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "7") } }, hir.synthetic_span);
    const addr = try module.hir.addExpr(.{ .address_of = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    const decl_p = try module.hir.addStmt(.{ .local_decl = .{ .local = p, .initializer = addr } }, hir.synthetic_span);
    const deref = try module.hir.addExpr(.{ .deref = try module.hir.addExpr(.{ .local_ref = p }, hir.synthetic_span) }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = deref }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl_x, decl_p, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "AddressOf(MirLocalId(0))") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Deref(Copy(MirLocalId(1)))") != null);
}

test "MIR lowering lowers address-of param and deref" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "readParamAddress", module.types.intType(), false);
    const int_ptr = try module.types.addPointerType(module.types.intType());
    const x = try addParam(&module, main, "x", module.types.intType());
    const p = try addLocal(&module, main, "p", int_ptr);
    const addr = try module.hir.addExpr(.{ .address_of = try module.hir.addExpr(.{ .param_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    const decl_p = try module.hir.addStmt(.{ .local_decl = .{ .local = p, .initializer = addr } }, hir.synthetic_span);
    const deref = try module.hir.addExpr(.{ .deref = try module.hir.addExpr(.{ .local_ref = p }, hir.synthetic_span) }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = deref }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl_p, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "AddressOf(MirLocalId(0))") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Deref(Copy(MirLocalId(1)))") != null);
}

test "MIR lowering lowers address-of field place" {
    var module = try newModule();
    defer module.deinit();
    const vec_id = try module.hir.addStruct(try intern(&module, "Vec2"));
    const vec_type = try module.types.addStructType(vec_id);
    const x_field = try module.hir.addField(vec_id, try intern(&module, "x"), module.types.intType(), hir.synthetic_span);
    const y_field = try module.hir.addField(vec_id, try intern(&module, "y"), module.types.intType(), hir.synthetic_span);
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const int_ptr = try module.types.addPointerType(module.types.intType());
    const v = try addLocal(&module, main, "v", vec_type);
    const p = try addLocal(&module, main, "p", int_ptr);
    const fields = try std.testing.allocator.dupe(hir.HirStructLiteralField, &.{
        .{ .field_id = x_field, .value = try intExpr(&module, "7"), .span = hir.synthetic_span },
        .{ .field_id = y_field, .value = try intExpr(&module, "4"), .span = hir.synthetic_span },
    });
    const lit = try module.hir.addExpr(.{ .struct_literal = .{ .struct_id = vec_id, .type_id = vec_type, .fields = fields } }, hir.synthetic_span);
    const decl_v = try module.hir.addStmt(.{ .local_decl = .{ .local = v, .initializer = lit } }, hir.synthetic_span);
    const access = try module.hir.addExpr(.{ .field_access = .{ .receiver = try module.hir.addExpr(.{ .local_ref = v }, hir.synthetic_span), .field_name = try intern(&module, "x"), .field_span = hir.synthetic_span } }, hir.synthetic_span);
    const addr = try module.hir.addExpr(.{ .address_of = access }, hir.synthetic_span);
    const decl_p = try module.hir.addStmt(.{ .local_decl = .{ .local = p, .initializer = addr } }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl_v, decl_p, try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span) });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "AddressOf(Field(MirLocalId(0), FieldId(0)))") != null);
}

test "MIR lowering stable snapshot for Phase 7 struct places" {
    var module = try newModule();
    defer module.deinit();

    const vec_id = try module.hir.addStruct(try intern(&module, "Vec2"));
    const vec_type = try module.types.addStructType(vec_id);
    const x_field = try module.hir.addField(vec_id, try intern(&module, "x"), module.types.intType(), hir.synthetic_span);
    const y_field = try module.hir.addField(vec_id, try intern(&module, "y"), module.types.intType(), hir.synthetic_span);
    const int_ptr = try module.types.addPointerType(module.types.intType());

    const main = try addFunction(&module, "main", module.types.intType(), false);
    const v = try addLocal(&module, main, "v", vec_type);
    const px = try addLocal(&module, main, "px", int_ptr);
    const fields = try std.testing.allocator.dupe(hir.HirStructLiteralField, &.{
        .{ .field_id = x_field, .value = try intExpr(&module, "3"), .span = hir.synthetic_span },
        .{ .field_id = y_field, .value = try intExpr(&module, "4"), .span = hir.synthetic_span },
    });
    const literal = try module.hir.addExpr(.{ .struct_literal = .{ .struct_id = vec_id, .type_id = vec_type, .fields = fields } }, hir.synthetic_span);
    const decl_v = try module.hir.addStmt(.{ .local_decl = .{ .local = v, .initializer = literal } }, hir.synthetic_span);
    const assign_x = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .field = .{ .base = .{ .local = v }, .field_id = x_field, .field_span = hir.synthetic_span } }, .value = try intExpr(&module, "11") } }, hir.synthetic_span);
    const field_access = try module.hir.addExpr(.{ .field_access = .{ .receiver = try module.hir.addExpr(.{ .local_ref = v }, hir.synthetic_span), .field_name = try intern(&module, "x"), .field_span = hir.synthetic_span } }, hir.synthetic_span);
    const addr = try module.hir.addExpr(.{ .address_of = field_access }, hir.synthetic_span);
    const decl_px = try module.hir.addStmt(.{ .local_decl = .{ .local = px, .initializer = addr } }, hir.synthetic_span);
    const deref = try module.hir.addExpr(.{ .deref = try module.hir.addExpr(.{ .local_ref = px }, hir.synthetic_span) }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = deref }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl_v, assign_x, decl_px, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try mir_validator.validateModule(std.testing.allocator, &module, &mir_module, &diagnostic_bag);
    try std.testing.expectEqual(@as(usize, 0), diagnostic_bag.count());

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user v: TypeId(6)
        \\      MirLocalId(1) temp <temp>: TypeId(6)
        \\      MirLocalId(2) user px: TypeId(7)
        \\      MirLocalId(3) temp <temp>: TypeId(7)
        \\      MirLocalId(4) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(1) = StructConstructor StructId(0)(FieldId(0)=Int 3, FieldId(1)=Int 4)
        \\        MirLocalId(0) = Use(Copy(MirLocalId(1)))
        \\        Field(MirLocalId(0), FieldId(0)) = Use(Int 11)
        \\        MirLocalId(3) = AddressOf(Field(MirLocalId(0), FieldId(0)))
        \\        MirLocalId(2) = Use(Copy(MirLocalId(3)))
        \\        MirLocalId(4) = Deref(Copy(MirLocalId(2)))
        \\        Return Copy(MirLocalId(4))
        \\
    , snapshot);
}

test "MIR lowering validates Phase 7 struct value params returns and calls" {
    var module = try newModule();
    defer module.deinit();

    const vec_id = try module.hir.addStruct(try intern(&module, "Vec2"));
    const vec_type = try module.types.addStructType(vec_id);
    const x_field = try module.hir.addField(vec_id, try intern(&module, "x"), module.types.intType(), hir.synthetic_span);
    const y_field = try module.hir.addField(vec_id, try intern(&module, "y"), module.types.intType(), hir.synthetic_span);

    const make = try addFunction(&module, "makeVec", vec_type, false);
    const make_x = try addParam(&module, make, "x", module.types.intType());
    const make_y = try addParam(&module, make, "y", module.types.intType());
    const make_fields = try std.testing.allocator.dupe(hir.HirStructLiteralField, &.{
        .{ .field_id = x_field, .value = try module.hir.addExpr(.{ .param_ref = make_x }, hir.synthetic_span), .span = hir.synthetic_span },
        .{ .field_id = y_field, .value = try module.hir.addExpr(.{ .param_ref = make_y }, hir.synthetic_span), .span = hir.synthetic_span },
    });
    const make_literal = try module.hir.addExpr(.{ .struct_literal = .{ .struct_id = vec_id, .type_id = vec_type, .fields = make_fields } }, hir.synthetic_span);
    try setBody(&module, make, &.{try module.hir.addStmt(.{ .return_stmt = make_literal }, hir.synthetic_span)});

    const sum = try addFunction(&module, "sum", module.types.intType(), false);
    const sum_v = try addParam(&module, sum, "v", vec_type);
    const sum_v_ref_x = try module.hir.addExpr(.{ .param_ref = sum_v }, hir.synthetic_span);
    const sum_v_ref_y = try module.hir.addExpr(.{ .param_ref = sum_v }, hir.synthetic_span);
    const sum_x = try module.hir.addExpr(.{ .field_access = .{ .receiver = sum_v_ref_x, .field_name = try intern(&module, "x"), .field_span = hir.synthetic_span } }, hir.synthetic_span);
    const sum_y = try module.hir.addExpr(.{ .field_access = .{ .receiver = sum_v_ref_y, .field_name = try intern(&module, "y"), .field_span = hir.synthetic_span } }, hir.synthetic_span);
    const sum_expr = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = sum_x, .right = sum_y } }, hir.synthetic_span);
    try setBody(&module, sum, &.{try module.hir.addStmt(.{ .return_stmt = sum_expr }, hir.synthetic_span)});

    const main = try addFunction(&module, "main", module.types.intType(), false);
    const v = try addLocal(&module, main, "v", vec_type);
    const make_args = try std.testing.allocator.dupe(hir.ExprId, &.{ try intExpr(&module, "3"), try intExpr(&module, "4") });
    const make_call = try module.hir.addExpr(.{ .call = .{ .function = make, .args = make_args } }, hir.synthetic_span);
    const decl_v = try module.hir.addStmt(.{ .local_decl = .{ .local = v, .initializer = make_call } }, hir.synthetic_span);
    const sum_args = try std.testing.allocator.dupe(hir.ExprId, &.{try module.hir.addExpr(.{ .local_ref = v }, hir.synthetic_span)});
    const sum_call = try module.hir.addExpr(.{ .call = .{ .function = sum, .args = sum_args } }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl_v, try module.hir.addStmt(.{ .return_stmt = sum_call }, hir.synthetic_span) });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    var diagnostic_bag = diagnostics.DiagnosticBag.init(std.testing.allocator);
    defer diagnostic_bag.deinit();
    try mir_validator.validateModule(std.testing.allocator, &module, &mir_module, &diagnostic_bag);
    try std.testing.expectEqual(@as(usize, 0), diagnostic_bag.count());

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function makeVec -> TypeId(6)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "StructConstructor StructId(0)(FieldId(0)=Copy(MirLocalId(0)), FieldId(1)=Copy(MirLocalId(1)))") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function sum -> TypeId(1)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "FieldAccess(Copy(MirLocalId(3)), FieldId(0))") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "FieldAccess(Copy(MirLocalId(3)), FieldId(1))") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call FunctionId(0)(Int 3, Int 4)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call FunctionId(1)(Copy(MirLocalId(") != null);
}

fn newModule() !semantics.SemanticModule {
    return semantics.SemanticModule.init(std.testing.allocator);
}

fn intern(module: *semantics.SemanticModule, text: []const u8) !hir.SymbolId {
    return module.interner.intern(text);
}

fn addFunction(module: *semantics.SemanticModule, name: []const u8, return_type: types.TypeId, has_body: bool) !hir.FunctionId {
    const function_id = try module.hir.addFunction(try intern(module, name), return_type, hir.synthetic_span);
    if (has_body) {
        const body = try module.hir.addStmt(.{ .block = &.{} }, hir.synthetic_span);
        module.hir.setFunctionBody(function_id, body);
    }
    return function_id;
}

fn setBody(module: *semantics.SemanticModule, function_id: hir.FunctionId, statements: []const hir.StmtId) !void {
    const owned = try std.testing.allocator.alloc(hir.StmtId, statements.len);
    @memcpy(owned, statements);
    const body = try module.hir.addStmt(.{ .block = owned }, hir.synthetic_span);
    module.hir.setFunctionBody(function_id, body);
}

fn addParam(module: *semantics.SemanticModule, function_id: hir.FunctionId, name: []const u8, type_id: types.TypeId) !hir.ParamId {
    return module.hir.addParam(function_id, try intern(module, name), type_id, hir.synthetic_span);
}

fn addLocal(module: *semantics.SemanticModule, function_id: hir.FunctionId, name: []const u8, type_id: types.TypeId) !hir.LocalId {
    return module.hir.addLocal(function_id, try intern(module, name), type_id, hir.synthetic_span);
}

fn intExpr(module: *semantics.SemanticModule, text: []const u8) !hir.ExprId {
    return module.hir.addExpr(.{ .int_literal = try std.testing.allocator.dupe(u8, text) }, hir.synthetic_span);
}

fn boolExpr(module: *semantics.SemanticModule, value: bool) !hir.ExprId {
    return module.hir.addExpr(.{ .bool_literal = value }, hir.synthetic_span);
}

fn blockStmt(module: *semantics.SemanticModule, statements: []const hir.StmtId) !hir.StmtId {
    const owned = try std.testing.allocator.alloc(hir.StmtId, statements.len);
    @memcpy(owned, statements);
    return module.hir.addStmt(.{ .block = owned }, hir.synthetic_span);
}

fn unaryExpr(module: *semantics.SemanticModule, op: hir.UnaryOp, operand: hir.ExprId) !hir.ExprId {
    return module.hir.addExpr(.{ .unary = .{ .op = op, .operand = operand } }, hir.synthetic_span);
}

fn binaryExpr(module: *semantics.SemanticModule, op: hir.BinaryOp, left: hir.ExprId, right: hir.ExprId) !hir.ExprId {
    return module.hir.addExpr(.{ .binary = .{ .op = op, .left = left, .right = right } }, hir.synthetic_span);
}

fn decideExpr(module: *semantics.SemanticModule, enum_type: types.TypeId, enum_id: hir.EnumId, arms: []const hir.HirDecideArm) !hir.ExprId {
    const owned = try std.testing.allocator.alloc(hir.HirDecideArm, arms.len);
    @memcpy(owned, arms);
    return module.hir.addExpr(.{ .decide = .{ .enum_type = enum_type, .enum_id = enum_id, .arms = owned } }, hir.synthetic_span);
}

test "MIR lowering debug snapshot lowers decide with one unconditional arm" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Mode"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const nominal = try module.hir.addVariant(enum_id, try intern(&module, "Nominal"), hir.synthetic_span);
    const main = try addFunction(&module, "main", enum_type, false);
    const selected = try addLocal(&module, main, "selected", enum_type);
    const decide = try decideExpr(&module, enum_type, enum_id, &.{.{ .variant_id = nominal, .condition = null, .score = try intExpr(&module, "0"), .span = hir.synthetic_span }});
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = selected, .initializer = decide } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = selected }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "temp hasWinner") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "temp bestScore") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "temp bestValue") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor EnumId(0)::VariantId(0)()") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary > Copy") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Return Copy(MirLocalId(0))") != null);
}

test "MIR lowering debug snapshot lowers decide condition and fallback" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Mode"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const hot = try module.hir.addVariant(enum_id, try intern(&module, "Hot"), hir.synthetic_span);
    const normal = try module.hir.addVariant(enum_id, try intern(&module, "Normal"), hir.synthetic_span);
    const main = try addFunction(&module, "main", enum_type, false);
    const decide = try decideExpr(&module, enum_type, enum_id, &.{
        .{ .variant_id = hot, .condition = try boolExpr(&module, true), .score = try intExpr(&module, "10"), .span = hir.synthetic_span },
        .{ .variant_id = normal, .condition = null, .score = try intExpr(&module, "0"), .span = hir.synthetic_span },
    });
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = decide }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "SwitchBool Bool true true: MirBlockId") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Use(Int 10)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Use(Int 0)") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor EnumId(0)::VariantId(0)()") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor EnumId(0)::VariantId(1)()") != null);
}

test "MIR lowering debug snapshot preserves decide duplicate variants and tie comparison" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Choice"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const a = try module.hir.addVariant(enum_id, try intern(&module, "A"), hir.synthetic_span);
    const b = try module.hir.addVariant(enum_id, try intern(&module, "B"), hir.synthetic_span);
    const main = try addFunction(&module, "main", enum_type, false);
    const decide = try decideExpr(&module, enum_type, enum_id, &.{
        .{ .variant_id = a, .condition = null, .score = try intExpr(&module, "5"), .span = hir.synthetic_span },
        .{ .variant_id = a, .condition = null, .score = try intExpr(&module, "7"), .span = hir.synthetic_span },
        .{ .variant_id = b, .condition = null, .score = try intExpr(&module, "7"), .span = hir.synthetic_span },
    });
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = decide }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor EnumId(0)::VariantId(0)()") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumConstructor EnumId(0)::VariantId(1)()") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary > Copy") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Binary >=") == null);
}

test "MIR lowering debug snapshot lowers decide negative score" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Choice"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const low = try module.hir.addVariant(enum_id, try intern(&module, "Low"), hir.synthetic_span);
    const main = try addFunction(&module, "main", enum_type, false);
    const negative = try unaryExpr(&module, .negate, try intExpr(&module, "1"));
    const decide = try decideExpr(&module, enum_type, enum_id, &.{.{ .variant_id = low, .condition = null, .score = negative, .span = hir.synthetic_span }});
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = decide }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Unary - Int 1") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "temp scoreTemp") != null);
}

test "MIR lowering debug snapshot lowers decide in call argument" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "Choice"), false);
    const enum_type = try module.types.addEnumType(enum_id);
    const a = try module.hir.addVariant(enum_id, try intern(&module, "A"), hir.synthetic_span);
    const take = try addFunction(&module, "take", module.types.intType(), false);
    _ = try addParam(&module, take, "choice", enum_type);
    try setBody(&module, take, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)});
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const decide = try decideExpr(&module, enum_type, enum_id, &.{.{ .variant_id = a, .condition = null, .score = try intExpr(&module, "0"), .span = hir.synthetic_span }});
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{decide});
    const call = try module.hir.addExpr(.{ .call = .{ .function = take, .args = args } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = call }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Function main") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "temp bestValue") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Call FunctionId(0)(Copy(") != null);
}

test "MIR lowering skips function declarations without bodies" {
    var module = try newModule();
    defer module.deinit();
    _ = try addFunction(&module, "declared", module.types.intType(), false);
    const main = try addFunction(&module, "main", module.types.intType(), true);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    try std.testing.expectEqual(@as(usize, 1), mir_module.store.functions.items.len);
    try std.testing.expectEqualStrings("main", module.interner.text(mir_module.store.functions.items[0].name));
}

test "MIR lowering skips compile-time-only functions" {
    var module = try newModule();
    defer module.deinit();
    const answer = try addFunction(&module, "answer", module.types.intType(), true);
    module.hir.markFunctionCompileTime(answer);
    const local = try addLocal(&module, answer, "value", module.types.intType());
    const local_decl = try module.hir.addStmt(.{ .local_decl = .{ .local = local, .initializer = try intExpr(&module, "42") } }, hir.synthetic_span);
    const local_ref = try module.hir.addExpr(.{ .local_ref = local }, hir.synthetic_span);
    const then_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = local_ref }, hir.synthetic_span)});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = then_block, .else_block = null } }, hir.synthetic_span);
    try setBody(&module, answer, &.{ local_decl, if_stmt, try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span) });
    const main = try addFunction(&module, "main", module.types.intType(), true);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    try std.testing.expectEqual(@as(usize, 1), mir_module.store.functions.items.len);
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "answer") == null);
}

test "MIR lowering lowers compile-time values as ordinary literals" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const add = try binaryExpr(&module, .add, try intExpr(&module, "40"), try intExpr(&module, "2"));
    const compile_int = try module.hir.addExpr(.{ .compile_time = .{ .operand = add, .span = hir.synthetic_span } }, hir.synthetic_span);
    try module.compile_time_values.put(compile_int, .{ .int = 42 });
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = compile_int }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqualStrings("42", block.terminator.?.kind.return_.?.int_literal);
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "CompileTime") == null);
}

test "MIR lowering ignores static assertions" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    try module.hir.addStaticAssert(try module.hir.addExpr(.{ .bool_literal = true }, hir.synthetic_span), hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "0") }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "StaticAssert") == null);
}

test "MIR lowering lowers return int" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "42") }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(@as(usize, 0), block.statements.len);
    try std.testing.expectEqualStrings("42", block.terminator.?.kind.return_.?.int_literal);
}

test "MIR lowering lowers empty body to unreachable" {
    var module = try newModule();
    defer module.deinit();
    _ = try addFunction(&module, "main", module.types.intType(), true);

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(@as(usize, 0), block.statements.len);
    try std.testing.expectEqual(mir.MirTerminatorKind.@"unreachable", block.terminator.?.kind);
}

test "MIR lowering lowers local declaration and local reference return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "1") } }, hir.synthetic_span);
    const ret_expr = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = ret_expr }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 1), function.locals.len);
    const local = mir_module.store.getLocal(function.locals[0]);
    try std.testing.expectEqual(mir.MirLocalKind.user, local.kind);
    try std.testing.expectEqualStrings("x", module.interner.text(local.name.?));

    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    try std.testing.expectEqualStrings("1", block.statements[0].kind.assign.rvalue.use.int_literal);
    try std.testing.expectEqual(function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers assignment" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "1") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span);
    const ret_expr = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = ret_expr }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, assign, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 2), block.statements.len);
    try std.testing.expectEqual(function.locals[0], block.statements[1].kind.assign.place.local);
    try std.testing.expectEqualStrings("2", block.statements[1].kind.assign.rvalue.use.int_literal);
}

test "MIR lowering lowers parameter assignment and grouped bool return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.boolType(), false);
    const flag = try addParam(&module, main, "flag", module.types.boolType());
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .param = flag }, .value = try boolExpr(&module, true) } }, hir.synthetic_span);
    const ref = try module.hir.addExpr(.{ .param_ref = flag }, hir.synthetic_span);
    const group = try module.hir.addExpr(.{ .group = ref }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = group }, hir.synthetic_span);
    try setBody(&module, main, &.{ assign, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 1), function.params.len);
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(function.params[0], block.statements[0].kind.assign.place.local);
    try std.testing.expectEqual(true, block.statements[0].kind.assign.rvalue.use.bool_literal);
    try std.testing.expectEqual(function.params[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers unary expression temp" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const one = try intExpr(&module, "1");
    const unary = try module.hir.addExpr(.{ .unary = .{ .op = .negate, .operand = one } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = unary }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    try std.testing.expectEqual(mir.MirLocalKind.temp, mir_module.store.getLocal(function.locals[0]).kind);
    try std.testing.expectEqual(mir.MirUnaryOp.negate, block.statements[0].kind.assign.rvalue.unary.op);
    try std.testing.expectEqual(function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers binary expression temp" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const binary = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try intExpr(&module, "1"), .right = try intExpr(&module, "2") } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = binary }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    const block = mir_module.store.getBlock(function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    try std.testing.expectEqual(mir.MirBinaryOp.add, block.statements[0].kind.assign.rvalue.binary.op);
    try std.testing.expectEqualStrings("1", block.statements[0].kind.assign.rvalue.binary.left.int_literal);
    try std.testing.expectEqualStrings("2", block.statements[0].kind.assign.rvalue.binary.right.int_literal);
    try std.testing.expectEqual(function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers function call temp" {
    var module = try newModule();
    defer module.deinit();
    const add = try addFunction(&module, "add", module.types.intType(), false);
    const a = try addParam(&module, add, "a", module.types.intType());
    const b = try addParam(&module, add, "b", module.types.intType());
    const add_expr = try module.hir.addExpr(.{ .binary = .{
        .op = .add,
        .left = try module.hir.addExpr(.{ .param_ref = a }, hir.synthetic_span),
        .right = try module.hir.addExpr(.{ .param_ref = b }, hir.synthetic_span),
    } }, hir.synthetic_span);
    try setBody(&module, add, &.{try module.hir.addStmt(.{ .return_stmt = add_expr }, hir.synthetic_span)});

    const main = try addFunction(&module, "main", module.types.intType(), false);
    const args = try std.testing.allocator.alloc(hir.ExprId, 2);
    args[0] = try intExpr(&module, "1");
    args[1] = try intExpr(&module, "2");
    const call = try module.hir.addExpr(.{ .call = .{ .function = add, .args = args } }, hir.synthetic_span);
    try setBody(&module, main, &.{try module.hir.addStmt(.{ .return_stmt = call }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const add_function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 2), add_function.params.len);

    const main_function = mir_module.store.functions.items[1];
    const block = mir_module.store.getBlock(main_function.blocks[0]);
    try std.testing.expectEqual(@as(usize, 1), block.statements.len);
    const lowered_call = block.statements[0].kind.assign.rvalue.call;
    try std.testing.expectEqual(hir.FunctionId{ .index = 0 }, lowered_call.function);
    try std.testing.expectEqual(@as(usize, 2), lowered_call.args.len);
    try std.testing.expectEqualStrings("1", lowered_call.args[0].int_literal);
    try std.testing.expectEqualStrings("2", lowered_call.args[1].int_literal);
    try std.testing.expectEqual(main_function.locals[0], block.terminator.?.kind.return_.?.copy.local);
}

test "MIR lowering lowers if without else" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span);
    const then_block = try blockStmt(&module, &.{assign});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = then_block, .else_block = null } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 3), function.blocks.len);
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_bool, std.meta.activeTag(mir_module.store.getBlock(function.blocks[0]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.goto, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.return_, std.meta.activeTag(mir_module.store.getBlock(function.blocks[2]).terminator.?.kind));
}

test "MIR lowering lowers if else with branch return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const then_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)});
    const assign_else = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span);
    const else_block = try blockStmt(&module, &.{assign_else});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, false), .then_block = then_block, .else_block = else_block } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 4), function.blocks.len);
    try std.testing.expectEqual(mir.MirTerminatorKind.return_, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.goto, std.meta.activeTag(mir_module.store.getBlock(function.blocks[3]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.return_, std.meta.activeTag(mir_module.store.getBlock(function.blocks[2]).terminator.?.kind));
}

test "MIR lowering lowers if else when both branches return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const then_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span)});
    const else_block = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "2") }, hir.synthetic_span)});
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = then_block, .else_block = else_block } }, hir.synthetic_span);
    const dead = try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "3") }, hir.synthetic_span);
    try setBody(&module, main, &.{ if_stmt, dead });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(mir.MirTerminatorKind.@"unreachable", mir_module.store.getBlock(function.blocks[2]).terminator.?.kind);
}

test "MIR lowering lowers while with assignment back edge" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const condition = try module.hir.addExpr(.{ .binary = .{ .op = .less, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "3") } }, hir.synthetic_span);
    const increment = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "1") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = increment } }, hir.synthetic_span);
    const while_stmt = try module.hir.addStmt(.{ .while_stmt = .{ .condition = condition, .body = try blockStmt(&module, &.{assign}) } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), while_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(@as(usize, 4), function.blocks.len);
    try std.testing.expectEqual(function.blocks[1], mir_module.store.getBlock(function.blocks[0]).terminator.?.kind.goto);
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_bool, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
    try std.testing.expectEqual(function.blocks[1], mir_module.store.getBlock(function.blocks[2]).terminator.?.kind.goto);
}

test "MIR lowering lowers match int default and bool" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const int_arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    int_arms[0] = .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span)}) };
    int_arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "3") } }, hir.synthetic_span)}) };
    const int_match = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try intExpr(&module, "1"), .arms = int_arms } }, hir.synthetic_span);
    const bool_arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    bool_arms[0] = .{ .pattern = .{ .bool_literal = true }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "4") } }, hir.synthetic_span)}) };
    bool_arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "5") }, hir.synthetic_span)}) };
    const bool_match = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try boolExpr(&module, false), .arms = bool_arms } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span), int_match, bool_match, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const function = mir_module.store.functions.items[0];
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_int, std.meta.activeTag(mir_module.store.getBlock(function.blocks[0]).terminator.?.kind));
    try std.testing.expectEqual(mir.MirTerminatorKind.switch_bool, std.meta.activeTag(mir_module.store.getBlock(function.blocks[1]).terminator.?.kind));
}

test "MIR lowering adds unreachable for unterminated straight-line body" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "1") } }, hir.synthetic_span);
    try setBody(&module, main, &.{decl});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(mir.MirTerminatorKind.@"unreachable", block.terminator.?.kind);
}

test "MIR lowering ignores statements after return" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const ret = try module.hir.addStmt(.{ .return_stmt = try intExpr(&module, "1") }, hir.synthetic_span);
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "2") } }, hir.synthetic_span);
    try setBody(&module, main, &.{ ret, decl });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const block = mir_module.store.getBlock(mir_module.store.functions.items[0].blocks[0]);
    try std.testing.expectEqual(@as(usize, 0), block.statements.len);
    try std.testing.expectEqualStrings("1", block.terminator.?.kind.return_.?.int_literal);
}

test "MIR lowering debug snapshot is stable for straight-line function" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const binary = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try intExpr(&module, "1"), .right = try intExpr(&module, "2") } }, hir.synthetic_span);
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = binary } }, hir.synthetic_span);
    const ret_expr = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = ret_expr }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\      MirLocalId(1) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(1) = Binary + Int 1, Int 2
        \\        MirLocalId(0) = Use(Copy(MirLocalId(1)))
        \\        Return Copy(MirLocalId(0))
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for if without else" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span);
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, true), .then_block = try blockStmt(&module, &.{assign}), .else_block = null } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchBool Bool true true: MirBlockId(1), false: MirBlockId(2)
        \\      MirBlockId(1)
        \\        MirLocalId(0) = Use(Int 1)
        \\        Goto MirBlockId(2)
        \\      MirBlockId(2)
        \\        Return Copy(MirLocalId(0))
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for while count loop" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const condition = try module.hir.addExpr(.{ .binary = .{ .op = .less, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "3") } }, hir.synthetic_span);
    const increment = try module.hir.addExpr(.{ .binary = .{ .op = .add, .left = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span), .right = try intExpr(&module, "1") } }, hir.synthetic_span);
    const assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = increment } }, hir.synthetic_span);
    const while_stmt = try module.hir.addStmt(.{ .while_stmt = .{ .condition = condition, .body = try blockStmt(&module, &.{assign}) } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, while_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\      MirLocalId(1) temp <temp>: TypeId(2)
        \\      MirLocalId(2) temp <temp>: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        Goto MirBlockId(1)
        \\      MirBlockId(1)
        \\        MirLocalId(1) = Binary < Copy(MirLocalId(0)), Int 3
        \\        SwitchBool Copy(MirLocalId(1)) true: MirBlockId(2), false: MirBlockId(3)
        \\      MirBlockId(2)
        \\        MirLocalId(2) = Binary + Copy(MirLocalId(0)), Int 1
        \\        MirLocalId(0) = Use(Copy(MirLocalId(2)))
        \\        Goto MirBlockId(1)
        \\      MirBlockId(3)
        \\        Return Copy(MirLocalId(0))
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for match int default" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "3") } }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try intExpr(&module, "1"), .arms = arms } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchInt Int 1 [1: MirBlockId(2)] default: MirBlockId(3)
        \\      MirBlockId(1)
        \\        Return Copy(MirLocalId(0))
        \\      MirBlockId(2)
        \\        MirLocalId(0) = Use(Int 2)
        \\        Goto MirBlockId(1)
        \\      MirBlockId(3)
        \\        MirLocalId(0) = Use(Int 3)
        \\        Goto MirBlockId(1)
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for if else join" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const then_assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span);
    const else_assign = try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span);
    const if_stmt = try module.hir.addStmt(.{ .if_stmt = .{ .condition = try boolExpr(&module, false), .then_block = try blockStmt(&module, &.{then_assign}), .else_block = try blockStmt(&module, &.{else_assign}) } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, if_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchBool Bool false true: MirBlockId(1), false: MirBlockId(3)
        \\      MirBlockId(1)
        \\        MirLocalId(0) = Use(Int 1)
        \\        Goto MirBlockId(2)
        \\      MirBlockId(2)
        \\        Return Copy(MirLocalId(0))
        \\      MirBlockId(3)
        \\        MirLocalId(0) = Use(Int 2)
        \\        Goto MirBlockId(2)
        \\
    , snapshot);
}

test "MIR lowering debug snapshot is stable for match bool" {
    var module = try newModule();
    defer module.deinit();
    const main = try addFunction(&module, "main", module.types.intType(), false);
    const x = try addLocal(&module, main, "x", module.types.intType());
    const decl = try module.hir.addStmt(.{ .local_decl = .{ .local = x, .initializer = try intExpr(&module, "0") } }, hir.synthetic_span);
    const arms = try std.testing.allocator.alloc(hir.HirMatchArm, 2);
    arms[0] = .{ .pattern = .{ .bool_literal = true }, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "1") } }, hir.synthetic_span)}) };
    arms[1] = .{ .pattern = .wildcard, .pattern_span = hir.synthetic_span, .body = try blockStmt(&module, &.{try module.hir.addStmt(.{ .assignment = .{ .target = .{ .local = x }, .value = try intExpr(&module, "2") } }, hir.synthetic_span)}) };
    const match_stmt = try module.hir.addStmt(.{ .match_stmt = .{ .scrutinee = try boolExpr(&module, true), .arms = arms } }, hir.synthetic_span);
    const ret = try module.hir.addStmt(.{ .return_stmt = try module.hir.addExpr(.{ .local_ref = x }, hir.synthetic_span) }, hir.synthetic_span);
    try setBody(&module, main, &.{ decl, match_stmt, ret });

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();

    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);

    try std.testing.expectEqualStrings(
        \\MirModule
        \\  Function main -> TypeId(1)
        \\    Locals
        \\      MirLocalId(0) user x: TypeId(1)
        \\    Blocks
        \\      MirBlockId(0)
        \\        MirLocalId(0) = Use(Int 0)
        \\        SwitchBool Bool true true: MirBlockId(2), false: MirBlockId(3)
        \\      MirBlockId(1)
        \\        Return Copy(MirLocalId(0))
        \\      MirBlockId(2)
        \\        MirLocalId(0) = Use(Int 1)
        \\        Goto MirBlockId(1)
        \\      MirBlockId(3)
        \\        MirLocalId(0) = Use(Int 2)
        \\        Goto MirBlockId(1)
        \\
    , snapshot);
}

test "MIR lowering debug snapshot includes try propagation control flow" {
    var module = try newModule();
    defer module.deinit();
    const enum_id = try module.hir.addEnum(try intern(&module, "ParseResult"), true);
    const enum_type = try module.types.addEnumType(enum_id);
    const ok = try module.hir.addVariant(enum_id, try intern(&module, "Ok"), hir.synthetic_span);
    const ok_field = try module.hir.addEnumPayloadField(ok, try intern(&module, "value"), module.types.intType(), hir.synthetic_span);
    const err = try module.hir.addVariant(enum_id, try intern(&module, "Err"), hir.synthetic_span);
    const err_field = try module.hir.addEnumPayloadField(err, try intern(&module, "code"), module.types.intType(), hir.synthetic_span);
    module.hir.setEnumResultShape(enum_id, .{ .ok_variant = ok, .err_variant = err, .ok_payload = ok_field, .err_payload = err_field, .ok_type = module.types.intType(), .err_type = module.types.intType() });

    const parse = try addFunction(&module, "parse", enum_type, false);
    _ = try addParam(&module, parse, "value", module.types.intType());
    const parse_arg = try module.hir.addExpr(.{ .param_ref = .{ .index = 0 } }, hir.synthetic_span);
    const parse_args = try std.testing.allocator.dupe(hir.ExprId, &.{parse_arg});
    const parse_ok = try module.hir.addExpr(.{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok, .args = parse_args } }, hir.synthetic_span);
    try setBody(&module, parse, &.{try module.hir.addStmt(.{ .return_stmt = parse_ok }, hir.synthetic_span)});

    const add = try addFunction(&module, "add", enum_type, false);
    const call_arg = try intExpr(&module, "7");
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{call_arg});
    const call = try module.hir.addExpr(.{ .call = .{ .function = parse, .args = call_args } }, hir.synthetic_span);
    const tried = try module.hir.addExpr(.{ .try_expr = call }, hir.synthetic_span);
    const ret_args = try std.testing.allocator.dupe(hir.ExprId, &.{tried});
    const ret_ok = try module.hir.addExpr(.{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok, .args = ret_args } }, hir.synthetic_span);
    try setBody(&module, add, &.{try module.hir.addStmt(.{ .return_stmt = ret_ok }, hir.synthetic_span)});

    var mir_module = try lowerModule(std.testing.allocator, &module);
    defer mir_module.deinit();
    const snapshot = try mir_module.store.debugString(std.testing.allocator, module.interner);
    defer std.testing.allocator.free(snapshot);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumTag(Copy(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "SwitchInt Copy(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "EnumPayloadField(Copy(") != null);
    try std.testing.expect(std.mem.indexOf(u8, snapshot, "Return Copy(") != null);
}
