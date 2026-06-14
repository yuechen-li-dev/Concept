// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

const std = @import("std");

const diagnostics = @import("diagnostics.zig");
const hir = @import("hir.zig");
const semantics = @import("semantics.zig");
const types = @import("types.zig");
const compile_time = @import("compile_time.zig");

pub const DiagnosticBag = diagnostics.DiagnosticBag;
pub const CheckError = anyerror;

const synthetic_span = hir.synthetic_span;

pub fn checkExecutable(
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    diagnostic_bag: ?*DiagnosticBag,
) CheckError!void {
    var checker = Checker.init(allocator, module, diagnostic_bag);
    defer checker.deinit();
    try checker.checkModule();
}

pub fn checkTestModule(
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    diagnostic_bag: ?*DiagnosticBag,
) CheckError!void {
    var checker = Checker.init(allocator, module, diagnostic_bag);
    defer checker.deinit();
    try checker.checkTestModule();
}

// ─────────────────────────────────────────────────────────────────────────────
// Checker state
// ─────────────────────────────────────────────────────────────────────────────

const Checker = struct {
    allocator: std.mem.Allocator,
    module: *semantics.SemanticModule,
    diagnostics: ?*DiagnosticBag,
    unsafe_depth: usize = 0,
    compile_time_context_depth: usize = 0,
    instantiations: std.ArrayList(InstantiationEntry),
    active_instantiation_depth: usize = 0,

    const max_instantiation_depth = 32;

    const InstantiationEntry = struct {
        generic_id: hir.GenericFunctionId,
        type_args: []types.TypeId,
        function: hir.FunctionId,
    };

    fn init(allocator: std.mem.Allocator, module: *semantics.SemanticModule, diagnostic_bag: ?*DiagnosticBag) Checker {
        return .{
            .allocator = allocator,
            .module = module,
            .diagnostics = diagnostic_bag,
            .instantiations = std.ArrayList(InstantiationEntry).empty,
        };
    }

    fn deinit(self: *Checker) void {
        for (self.instantiations.items) |entry| self.allocator.free(entry.type_args);
        self.instantiations.deinit(self.allocator);
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Function/main validation
    // ─────────────────────────────────────────────────────────────────────────────

    fn checkModule(self: *Checker) CheckError!void {
        try self.checkStaticAsserts();
        try self.checkAggregateDynStorage();
        try self.checkInterfaceRequirements();
        try self.checkInterfaceImpls();

        const main_id = self.findMain() orelse {
            try self.report(.MissingMain, "expected top-level 'main' function");
            return error.InvalidSemanticModule;
        };
        const main = self.module.hir.getFunction(main_id);
        if (!sameType(main.return_type, self.module.types.intType())) {
            try self.reportAt(.InvalidMainSignature, "main must return int", main.span);
            return error.InvalidSemanticModule;
        }
        if (main.params.len != 0) {
            try self.reportAt(.InvalidMainSignature, "main must not have parameters", main.span);
            return error.InvalidSemanticModule;
        }
        if (main.body == null) {
            try self.reportAt(.InvalidMainSignature, "main must have a body", main.span);
            return error.InvalidSemanticModule;
        }

        try self.checkFunctionBodies();
    }

    fn checkTestModule(self: *Checker) CheckError!void {
        try self.checkStaticAsserts();
        try self.checkAggregateDynStorage();
        try self.checkInterfaceRequirements();
        try self.checkInterfaceImpls();
        try self.checkFunctionBodies();
    }

    fn checkFunctionBodies(self: *Checker) CheckError!void {
        for (self.module.hir.functions.items, 0..) |function, index| {
            const function_id = hir.FunctionId{ .index = @intCast(index) };
            if (function.is_extern) {
                try self.checkExternFunctionDeclaration(function_id, function);
                continue;
            }
            if (self.module.hir.isGenericFunction(function_id)) {
                if (function.is_exported) {
                    try self.reportAt(.ExportCFunctionCannotBeGeneric, "export C function cannot be generic", function.span);
                    return error.InvalidSemanticModule;
                }
                continue;
            }
            if (function.is_exported) try self.checkExportCFunction(function_id, function);
            try self.checkDropParams(function_id, function);
            if (function.body) |body| {
                if (function.is_compile_time) {
                    self.compile_time_context_depth += 1;
                }
                if (function.is_unsafe) self.unsafe_depth += 1;
                defer {
                    if (function.is_unsafe) self.unsafe_depth -= 1;
                    if (function.is_compile_time) self.compile_time_context_depth -= 1;
                }
                try self.checkStmt(function_id, body, function.return_type);
                if (function.is_compile_time) try self.checkCompileTimeFunctionEligibility(function);
            }
            try self.checkOpaqueAllocationTypeSignature(function);
            try self.checkInterfaceRuntimeSignature(function);
        }
    }

    fn checkExternFunctionDeclaration(self: *Checker, function_id: hir.FunctionId, function: hir.HirFunction) CheckError!void {
        if (function.body != null) {
            try self.reportAt(.InvalidCall, "extern C function declaration must not have a body", function.span);
            return error.InvalidSemanticModule;
        }
        if (function.is_compile_time or self.module.hir.isGenericFunction(function_id) or function.is_concept_witness) {
            try self.reportAt(.UnsupportedCAbiType, "extern C function declaration cannot be comptime, generic, or a concept witness", function.span);
            return error.InvalidSemanticModule;
        }
        if (function.extern_abi == null or function.extern_abi.? != .c or function.c_symbol_name == null) {
            try self.reportAt(.UnsupportedCAbiType, "extern C function declaration is missing C ABI metadata", function.span);
            return error.InvalidSemanticModule;
        }
        if (!self.isSupportedCAbiReturnType(function.return_type)) {
            try self.reportAt(.UnsupportedCAbiType, "unsupported extern C return type", function.span);
            return error.InvalidSemanticModule;
        }
        for (function.params) |param_id| {
            const param = self.module.hir.getParam(param_id);
            if (!self.isSupportedCAbiParamType(param.type_id)) {
                try self.reportAt(.UnsupportedCAbiType, "unsupported extern C parameter type", param.span);
                return error.InvalidSemanticModule;
            }
        }
    }

    fn checkExportCFunction(self: *Checker, function_id: hir.FunctionId, function: hir.HirFunction) CheckError!void {
        if (function.body == null) {
            try self.reportAt(.ExportCRequiresFunctionDefinition, "export C function requires a body", function.span);
            return error.InvalidSemanticModule;
        }
        if (self.module.hir.isGenericFunction(function_id) or function.is_concept_witness) {
            try self.reportAt(.ExportCFunctionCannotBeGeneric, "export C function cannot be generic or a concept witness", function.span);
            return error.InvalidSemanticModule;
        }
        if (function.extern_abi == null or function.extern_abi.? != .c or function.c_symbol_name == null) {
            try self.reportAt(.UnsupportedCAbiType, "export C function is missing C ABI metadata", function.span);
            return error.InvalidSemanticModule;
        }
        if (!self.isSupportedCAbiReturnType(function.return_type)) {
            try self.reportAt(.UnsupportedCAbiType, "unsupported export C return type", function.span);
            return error.InvalidSemanticModule;
        }
        for (function.params) |param_id| {
            const param = self.module.hir.getParam(param_id);
            if (!self.isSupportedCAbiParamType(param.type_id)) {
                try self.reportAt(.UnsupportedCAbiType, "unsupported export C parameter type", param.span);
                return error.InvalidSemanticModule;
            }
        }
    }

    fn isSupportedCAbiReturnType(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .void, .int, .bool, .alloc_error => true,
            .pointer => |pointer| self.isSupportedCAbiPointerPointee(pointer.pointee),
            else => false,
        };
    }

    fn isSupportedCAbiParamType(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .int, .bool, .alloc_error => true,
            .pointer => |pointer| self.isSupportedCAbiPointerPointee(pointer.pointee),
            else => false,
        };
    }

    fn isSupportedCAbiPointerPointee(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .void, .int, .bool, .arena, .allocator, .alloc_error => true,
            else => false,
        };
    }

    fn checkDropParams(self: *Checker, function_id: hir.FunctionId, function: hir.HirFunction) CheckError!void {
        for (function.params) |param_id| {
            const param = self.module.hir.getParam(param_id);
            const drop_info = self.module.hasDrop(param.type_id) orelse continue;
            if (drop_info.function.index == function_id.index) continue;
            try self.reportAt(.DropParamUnsupported, "by-value parameter with Drop<T> is not supported until parameter cleanup is implemented", param.span);
            return error.InvalidSemanticModule;
        }
    }

    fn checkOpaqueAllocationTypeSignature(self: *Checker, function: hir.HirFunction) CheckError!void {
        if (self.isByValueOpaqueAllocationHandle(function.return_type)) {
            try self.reportAt(.OpaqueAllocationTypeByValueUnsupported, "Arena and Allocator are opaque allocation handles and are not supported by value in v0; pass a pointer instead", function.span);
            return error.InvalidSemanticModule;
        }
        for (function.params) |param_id| {
            const param = self.module.hir.getParam(param_id);
            if (!self.isByValueOpaqueAllocationHandle(param.type_id)) continue;
            try self.reportAt(.OpaqueAllocationTypeByValueUnsupported, "Arena and Allocator are opaque allocation handles and are not supported by value in v0; pass a pointer instead", param.span);
            return error.InvalidSemanticModule;
        }
    }

    fn isByValueOpaqueAllocationHandle(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .arena, .allocator => true,
            else => false,
        };
    }

    fn checkInterfaceRuntimeSignature(self: *Checker, function: hir.HirFunction) CheckError!void {
        if (self.isInterfaceRuntimeType(function.return_type)) {
            try self.reportAt(.InterfaceRuntimeUnsupported, "interface types are not supported as ordinary return values in Phase 14 M2", function.span);
            return error.InvalidSemanticModule;
        }
        if (self.isDynInterfaceType(function.return_type)) {
            try self.reportAt(.InterfaceRuntimeUnsupported, "dyn interface references are not supported as return values in Phase 14 M4", function.span);
            return error.InvalidSemanticModule;
        }
        for (function.params) |param_id| {
            const param = self.module.hir.getParam(param_id);
            if (!self.isInterfaceRuntimeType(param.type_id)) continue;
            try self.reportAt(.InterfaceRuntimeUnsupported, "interface types are not supported as ordinary parameters in Phase 14 M2", param.span);
            return error.InvalidSemanticModule;
        }
    }

    fn checkInterfaceRequirements(self: *Checker) CheckError!void {
        for (self.module.hir.interfaces.items) |interface_decl| {
            for (interface_decl.requirements) |requirement_id| {
                const requirement = self.module.hir.getInterfaceRequirement(requirement_id);
                // Phase 14 v0 keeps interface declarations HIR-only. A requirement
                // may name ordinary value and pointer types, but interface types
                // themselves cannot be used as runtime values before dyn exists.
                if (self.isInterfaceRuntimeType(requirement.return_type)) {
                    try self.reportAt(.InterfaceRuntimeUnsupported, "interface types are not supported as interface requirement returns in Phase 14 M2", requirement.span);
                    return error.InvalidSemanticModule;
                }
                if (self.isDynInterfaceType(requirement.return_type)) {
                    try self.reportAt(.InterfaceRuntimeUnsupported, "dyn interface references are not supported in interface requirements in Phase 14 M4", requirement.span);
                    return error.InvalidSemanticModule;
                }
                for (requirement.params) |param_id| {
                    const param = self.module.hir.getInterfaceParam(param_id);
                    if (!self.isInterfaceRuntimeType(param.type_id) and !self.isDynInterfaceType(param.type_id)) continue;
                    try self.reportAt(.InterfaceRuntimeUnsupported, "interface runtime types are not supported as interface requirement parameters in Phase 14 M4", param.span);
                    return error.InvalidSemanticModule;
                }
            }
        }
    }

    fn checkAggregateDynStorage(self: *Checker) CheckError!void {
        for (self.module.hir.structs.items) |struct_decl| {
            for (struct_decl.fields) |field_id| {
                const field = self.module.hir.getField(field_id);
                if (!self.isDynInterfaceType(field.type_id)) continue;
                try self.reportAt(.InterfaceRuntimeUnsupported, "dyn interface references are not supported as struct fields in Phase 14 M4", field.span);
                return error.InvalidSemanticModule;
            }
        }
    }

    fn checkInterfaceImpls(self: *Checker) CheckError!void {
        var seen = std.AutoHashMap(InterfaceImplKey, hir.SourceSpan).init(self.allocator);
        defer seen.deinit();

        for (self.module.hir.interface_impls.items) |interface_impl| {
            const key = InterfaceImplKey{ .interface_index = interface_impl.interface_id.index, .target_index = interface_impl.target_type.index };
            if (seen.contains(key)) {
                try self.reportAt(.DuplicateInterfaceImpl, "duplicate interface impl", interface_impl.span);
                return error.InvalidSemanticModule;
            }
            try seen.put(key, interface_impl.span);

            if (self.isInvalidInterfaceImplTarget(interface_impl.target_type)) {
                try self.reportAt(.InvalidInterfaceImplTarget, "invalid interface impl target", interface_impl.span);
                return error.InvalidSemanticModule;
            }

            const interface_decl = self.module.hir.getInterface(interface_impl.interface_id);
            var function_names = std.AutoHashMap(hir.SymbolId, hir.SourceSpan).init(self.allocator);
            defer function_names.deinit();
            for (interface_impl.functions) |function_id| {
                const function = self.module.hir.getFunction(function_id);
                if (function_names.contains(function.name)) {
                    try self.reportAt(.DuplicateInterfaceImplFunction, "duplicate function in interface impl", function.span);
                    return error.InvalidSemanticModule;
                }
                try function_names.put(function.name, function.span);
            }

            for (interface_decl.requirements) |requirement_id| {
                const requirement = self.module.hir.getInterfaceRequirement(requirement_id);
                const function_id = self.findInterfaceImplFunction(interface_impl, requirement.name) orelse {
                    try self.reportAt(.MissingInterfaceRequirementImpl, "missing interface requirement implementation", interface_impl.span);
                    return error.InvalidSemanticModule;
                };
                const function = self.module.hir.getFunction(function_id);
                if (!sameType(function.return_type, requirement.return_type) or function.params.len != requirement.params.len + 1) {
                    try self.reportAt(.InvalidInterfaceRequirementImplSignature, "interface requirement implementation signature does not match", function.span);
                    return error.InvalidSemanticModule;
                }
                const receiver = self.module.hir.getParam(function.params[0]);
                if (!sameType(receiver.type_id, interface_impl.target_type)) {
                    try self.reportAt(.InvalidInterfaceRequirementImplSignature, "interface requirement implementation receiver does not match target type", receiver.span);
                    return error.InvalidSemanticModule;
                }
                for (requirement.params, 0..) |required_param_id, required_index| {
                    const required_param = self.module.hir.getInterfaceParam(required_param_id);
                    const actual_param = self.module.hir.getParam(function.params[required_index + 1]);
                    if (!sameType(actual_param.type_id, required_param.type_id)) {
                        try self.reportAt(.InvalidInterfaceRequirementImplSignature, "interface requirement implementation parameter does not match", actual_param.span);
                        return error.InvalidSemanticModule;
                    }
                }
            }

            for (interface_impl.functions) |function_id| {
                const function = self.module.hir.getFunction(function_id);
                if (self.findInterfaceRequirement(interface_decl.*, function.name) == null) {
                    try self.reportAt(.ExtraInterfaceImplFunction, "extra function in interface impl", function.span);
                    return error.InvalidSemanticModule;
                }
            }
        }
    }

    const InterfaceImplKey = struct {
        interface_index: u32,
        target_index: u32,
    };

    fn isInvalidInterfaceImplTarget(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .void, .interface_type, .dyn_interface, .type_param => true,
            else => false,
        };
    }

    fn findInterfaceImplFunction(self: *Checker, interface_impl: hir.HirInterfaceImpl, name: hir.SymbolId) ?hir.FunctionId {
        var found: ?hir.FunctionId = null;
        for (interface_impl.functions) |function_id| {
            const function = self.module.hir.getFunction(function_id);
            if (function.name.index != name.index) continue;
            if (found != null) return null;
            found = function_id;
        }
        return found;
    }

    fn findInterfaceRequirement(self: *Checker, interface_decl: hir.HirInterface, name: hir.SymbolId) ?hir.InterfaceRequirementId {
        for (interface_decl.requirements) |requirement_id| {
            const requirement = self.module.hir.getInterfaceRequirement(requirement_id);
            if (requirement.name.index == name.index) return requirement_id;
        }
        return null;
    }

    fn isInterfaceRuntimeType(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .interface_type => true,
            .pointer => |pointer| self.isInterfaceRuntimeType(pointer.pointee),
            else => false,
        };
    }

    fn isDynInterfaceType(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .dyn_interface => true,
            else => false,
        };
    }

    fn checkStaticAsserts(self: *Checker) CheckError!void {
        for (self.module.hir.static_asserts.items) |static_assert| {
            self.compile_time_context_depth += 1;
            defer self.compile_time_context_depth -= 1;
            _ = try self.checkExpr(null, self.module.types.voidType(), static_assert.expr);
            var evaluator = compile_time.CompileTimeEvaluator.init(self.module, self.allocator);
            const value = evaluator.evaluateExpr(static_assert.expr) catch |err| {
                try self.reportCompileTimeError(err, static_assert.span);
                return error.InvalidSemanticModule;
            };
            switch (value) {
                .bool => |ok| {
                    if (!ok) {
                        try self.reportAt(.StaticAssertFailed, "static assertion failed", static_assert.span);
                        return error.InvalidSemanticModule;
                    }
                },
                else => {
                    try self.reportAt(.StaticAssertRequiresBool, "static assertion requires bool expression", static_assert.span);
                    return error.InvalidSemanticModule;
                },
            }
        }
    }

    fn findMain(self: *Checker) ?hir.FunctionId {
        for (self.module.hir.functions.items, 0..) |function, index| {
            const function_id = hir.FunctionId{ .index = @intCast(index) };
            if (self.module.hir.isGenericFunction(function_id)) continue;
            if (self.module.hir.isConceptWitnessFunction(function_id)) continue;
            if (function.is_extern) continue;
            if (function.is_compile_time) continue;
            if (std.mem.eql(u8, self.module.interner.text(function.name), "main")) return function_id;
        }
        return null;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Statement checking
    // ─────────────────────────────────────────────────────────────────────────────

    fn checkStmt(self: *Checker, function_id: hir.FunctionId, stmt_id: hir.StmtId, return_type: types.TypeId) CheckError!void {
        const stmt = self.module.hir.getStmt(stmt_id).*;
        switch (stmt.kind) {
            .block => |stmts| for (stmts) |child| try self.checkStmt(function_id, child, return_type),
            .return_stmt => |maybe_value| {
                const value = maybe_value orelse {
                    try self.reportAt(.TypeMismatch, "return statements must have an expression", stmt.span);
                    return error.InvalidSemanticModule;
                };
                const value_type = try self.checkExpr(function_id, return_type, value);
                try self.requireSame(value_type, return_type, "return expression type does not match function return type", self.exprSpan(value));
            },
            .transition_stmt => {
                try self.reportAt(.MachineSemanticsNotImplemented, "machine transition statements are not executable in function HIR", stmt.span);
                return error.InvalidSemanticModule;
            },
            .local_decl => |decl| {
                const local = self.module.hir.getLocal(decl.local);
                if (self.isInterfaceRuntimeType(local.type_id) or self.isDynInterfaceType(local.type_id)) {
                    try self.reportAt(.InterfaceRuntimeUnsupported, "interface runtime types are not supported as ordinary locals in Phase 14 M4", local.span);
                    return error.InvalidSemanticModule;
                }
                const init_type = try self.checkExpr(function_id, return_type, decl.initializer);
                try self.requireSame(init_type, local.type_id, "local initializer type does not match declared type", self.exprSpan(decl.initializer));
            },
            .expr_stmt => |expr_id| {
                const value_type = try self.checkExpr(function_id, return_type, expr_id);
                if (self.isMustUseType(value_type)) {
                    try self.reportAt(.IgnoredMustUseValue, "value must be used; use discard to ignore intentionally", self.exprSpan(expr_id));
                    return error.InvalidSemanticModule;
                }
            },
            .discard_stmt => |expr_id| {
                const value_type = try self.checkExpr(function_id, return_type, expr_id);
                if (sameType(value_type, self.module.types.voidType())) {
                    try self.reportAt(.TypeMismatch, "discard requires a value expression", self.exprSpan(expr_id));
                    return error.InvalidSemanticModule;
                }
            },
            .arena_reset => |op| try self.checkArenaStorageOp(function_id, return_type, op, stmt.span),
            .arena_destroy => |op| try self.checkArenaStorageOp(function_id, return_type, op, stmt.span),
            .assignment => |assignment| {
                const target_type = try self.placeType(assignment.target, stmt.span);
                const value_type = try self.checkExpr(function_id, return_type, assignment.value);
                if (!sameType(value_type, target_type)) {
                    const code: diagnostics.DiagnosticCode = switch (assignment.target) {
                        .field => .FieldAssignmentTypeMismatch,
                        else => .TypeMismatch,
                    };
                    try self.reportAt(code, "assignment expression type does not match target type", self.exprSpan(assignment.value));
                    return error.InvalidSemanticModule;
                }
            },
            .if_stmt => |if_stmt| {
                const condition_type = try self.checkExpr(function_id, return_type, if_stmt.condition);
                try self.requireBool(condition_type, "if condition must be bool", self.exprSpan(if_stmt.condition));
                try self.checkStmt(function_id, if_stmt.then_block, return_type);
                if (if_stmt.else_block) |else_block| try self.checkStmt(function_id, else_block, return_type);
            },
            .while_stmt => |while_stmt| {
                const condition_type = try self.checkExpr(function_id, return_type, while_stmt.condition);
                try self.requireBool(condition_type, "while condition must be bool", self.exprSpan(while_stmt.condition));
                try self.checkStmt(function_id, while_stmt.body, return_type);
            },
            .unsafe_block => |body| {
                self.unsafe_depth += 1;
                defer self.unsafe_depth -= 1;
                try self.checkStmt(function_id, body, return_type);
            },
            .match_stmt => |match_stmt| {
                const scrutinee_type = try self.checkExpr(function_id, return_type, match_stmt.scrutinee);
                const scrutinee_kind = self.module.types.kind(scrutinee_type);
                if (!self.isInt(scrutinee_type) and !self.isBool(scrutinee_type) and scrutinee_kind != .enum_type) {
                    try self.reportAt(.TypeMismatch, "match scrutinee must be int, bool, or enum", self.exprSpan(match_stmt.scrutinee));
                    return error.InvalidSemanticModule;
                }
                var seen_ints = std.StringHashMap(void).init(self.allocator);
                defer seen_ints.deinit();
                var seen_variants = std.AutoHashMap(hir.VariantId, void).init(self.allocator);
                defer seen_variants.deinit();
                var seen_true = false;
                var seen_false = false;
                var seen_wildcard = false;
                for (match_stmt.arms) |arm| {
                    switch (arm.pattern) {
                        .int_literal => |text| {
                            try self.requireSame(self.module.types.intType(), scrutinee_type, "match pattern type does not match scrutinee type", arm.pattern_span);
                            if (seen_ints.contains(text)) {
                                try self.reportAt(.TypeMismatch, "duplicate literal match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            try seen_ints.put(text, {});
                        },
                        .bool_literal => |value| {
                            try self.requireSame(self.module.types.boolType(), scrutinee_type, "match pattern type does not match scrutinee type", arm.pattern_span);
                            if ((value and seen_true) or (!value and seen_false)) {
                                try self.reportAt(.TypeMismatch, "duplicate literal match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            if (value) seen_true = true else seen_false = true;
                        },
                        .enum_variant => |pattern| {
                            if (scrutinee_kind != .enum_type or scrutinee_kind.enum_type.index != pattern.enum_id.index) {
                                try self.reportAt(.EnumPatternTypeMismatch, "enum pattern type does not match match scrutinee", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            if (seen_variants.contains(pattern.variant_id)) {
                                try self.reportAt(.EnumPatternTypeMismatch, "duplicate enum variant match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            try seen_variants.put(pattern.variant_id, {});
                        },
                        .wildcard => {
                            if (seen_wildcard) {
                                try self.reportAt(.TypeMismatch, "duplicate wildcard match pattern", arm.pattern_span);
                                return error.InvalidSemanticModule;
                            }
                            seen_wildcard = true;
                        },
                    }
                    try self.checkStmt(function_id, arm.body, return_type);
                }
            },
        }
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Expression checking
    // ─────────────────────────────────────────────────────────────────────────────

    fn checkExpr(self: *Checker, current_function_id: ?hir.FunctionId, return_type: types.TypeId, expr_id: hir.ExprId) CheckError!types.TypeId {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .int_literal => self.module.types.intType(),
            .bool_literal => self.module.types.boolType(),
            .local_ref => |id| self.module.hir.getLocal(id).type_id,
            .param_ref => |id| self.module.hir.getParam(id).type_id,
            .machine_param_ref => |id| self.module.hir.getMachineParam(id).type_id,
            .group => |inner| try self.checkExpr(current_function_id, return_type, inner),
            .compile_time => |compile_time_expr| blk: {
                self.compile_time_context_depth += 1;
                defer self.compile_time_context_depth -= 1;
                _ = try self.checkExpr(current_function_id, return_type, compile_time_expr.operand);
                var evaluator = compile_time.CompileTimeEvaluator.init(self.module, self.allocator);
                const value = evaluator.evaluateExpr(compile_time_expr.operand) catch |err| {
                    try self.reportCompileTimeError(err, expr.span);
                    return error.InvalidSemanticModule;
                };
                try self.module.compile_time_values.put(expr_id, value);
                break :blk value.typeOf(self.module.types);
            },
            .concept_requirement_call => {
                try self.reportAt(.InvalidConceptRequirementCall, "concept requirement call was not rewritten before checking", expr.span);
                return error.InvalidSemanticModule;
            },
            .target_metadata => |metadata| blk: {
                if (self.compile_time_context_depth == 0) {
                    try self.reportAt(.CompileTimeTargetMetadataRequiresCompileTime, "target metadata requires a compile-time context", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk metadata.query.typeOf(self.module.types);
            },
            .test_intrinsic => |test_intrinsic| blk: {
                try self.checkTestIntrinsic(current_function_id, return_type, expr, test_intrinsic);
                break :blk self.module.types.voidType();
            },
            .dyn_coerce => |coerce| blk: {
                _ = try self.checkExpr(current_function_id, return_type, coerce.source);
                try self.requireDynCoerceValid(coerce, expr.span);
                break :blk coerce.result_type;
            },
            .interface_call => |call| blk: {
                const receiver_type = try self.checkExpr(current_function_id, return_type, call.receiver);
                const dyn = switch (self.module.types.kind(receiver_type)) {
                    .dyn_interface => |dyn| dyn,
                    else => {
                        try self.reportAt(.InvalidCall, "interface call receiver must be a dyn interface reference", self.exprSpan(call.receiver));
                        return error.InvalidSemanticModule;
                    },
                };
                if (dyn.interface_id.index != call.interface_id.index) {
                    try self.reportAt(.InvalidCall, "interface call receiver interface mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (!dyn.is_mut) {
                    try self.reportAt(.InterfaceCallRequiresMutableDyn, "interface method call requires a mutable dyn reference", self.exprSpan(call.receiver));
                    return error.InvalidSemanticModule;
                }
                if (call.interface_id.index >= self.module.hir.interfaces.items.len or call.requirement_id.index >= self.module.hir.interface_requirements.items.len) {
                    try self.reportAt(.UnknownInterfaceMethod, "unknown interface method", expr.span);
                    return error.InvalidSemanticModule;
                }
                const interface_decl = self.module.hir.getInterface(call.interface_id);
                if (call.requirement_index >= interface_decl.requirements.len or interface_decl.requirements[call.requirement_index].index != call.requirement_id.index) {
                    try self.reportAt(.UnknownInterfaceMethod, "unknown interface method", expr.span);
                    return error.InvalidSemanticModule;
                }
                const requirement = self.module.hir.getInterfaceRequirement(call.requirement_id);
                if (requirement.parent.index != call.interface_id.index) {
                    try self.reportAt(.UnknownInterfaceMethod, "unknown interface method", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (call.args.len != requirement.params.len) {
                    try self.reportAt(.InterfaceCallArityMismatch, "interface method call argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (call.args, requirement.params) |arg, param_id| {
                    const arg_type = try self.checkExpr(current_function_id, return_type, arg);
                    const param_type = self.module.hir.getInterfaceParam(param_id).type_id;
                    if (!sameType(arg_type, param_type)) {
                        try self.reportAt(.InterfaceCallTypeMismatch, "interface method call argument type mismatch", self.exprSpan(arg));
                        return error.InvalidSemanticModule;
                    }
                }
                if (!sameType(call.result_type, requirement.return_type)) {
                    try self.reportAt(.InterfaceCallTypeMismatch, "interface method call result type mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk call.result_type;
            },
            .call => |call| blk: {
                var arg_types = std.ArrayList(types.TypeId).empty;
                defer arg_types.deinit(self.allocator);
                for (call.args) |arg| try arg_types.append(self.allocator, try self.checkExpr(current_function_id, return_type, arg));

                const resolved_function = if (self.genericFunctionFor(call.function)) |generic_id|
                    try self.instantiateGenericCall(generic_id, call, arg_types.items, expr.span)
                else
                    call.function;

                if (resolved_function.index != call.function.index) {
                    self.module.hir.getExprMut(expr_id).kind.call.function = resolved_function;
                }

                const callee = self.module.hir.getFunction(resolved_function);
                if (callee.is_compile_time and self.compile_time_context_depth == 0) {
                    try self.reportAt(.CompileTimeFunctionRequired, "compile-time function call requires a compile-time context", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (callee.body == null and !callee.is_extern) {
                    try self.reportAt(.InvalidCall, "cannot call function without body", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (callee.is_unsafe and self.unsafe_depth == 0) {
                    try self.reportAt(.UnsafeCallRequiresUnsafe, "unsafe function call requires unsafe context", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (call.args.len != callee.params.len) {
                    try self.reportAt(.InvalidCall, "function call argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (arg_types.items, callee.params, call.args, 0..) |arg_type, param_id, arg_expr, arg_index| {
                    const param_type = self.module.hir.getParam(param_id).type_id;
                    const checked_arg = try self.coerceDynCallArg(arg_expr, arg_type, param_type);
                    if (checked_arg.index != arg_expr.index) {
                        call.args[arg_index] = checked_arg;
                        arg_types.items[arg_index] = param_type;
                    } else {
                        try self.requireCallSame(arg_type, param_type, "function call argument type mismatch", self.exprSpan(arg_expr));
                    }
                }
                try self.checkAllocationEffectCall(current_function_id, callee.*, expr.span);
                break :blk callee.return_type;
            },
            .machine_construct => |construct| blk: {
                const machine = self.module.hir.getMachine(construct.machine);
                if (construct.args.len != machine.params.len) {
                    try self.reportAt(.InvalidCall, "machine constructor argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (construct.args, machine.params) |arg, param_id| {
                    const arg_type = try self.checkExpr(current_function_id, return_type, arg);
                    const param_type = self.module.hir.getMachineParam(param_id).type_id;
                    try self.requireCallSame(arg_type, param_type, "machine constructor argument type mismatch", self.exprSpan(arg));
                }
                break :blk try self.module.types.addMachineType(construct.machine);
            },
            .machine_step => |machine_expr| blk: {
                const machine_type = try self.checkExpr(current_function_id, return_type, machine_expr);
                if (self.module.types.kind(machine_type) != .machine_type) {
                    try self.reportAt(.InvalidCall, "Step expects a machine argument", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk self.module.types.voidType();
            },
            .machine_complete => |machine_expr| blk: {
                const machine_type = try self.checkExpr(current_function_id, return_type, machine_expr);
                if (self.module.types.kind(machine_type) != .machine_type) {
                    try self.reportAt(.InvalidCall, "Complete expects a machine argument", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk self.module.types.boolType();
            },
            .machine_result => |machine_expr| blk: {
                const machine_type = try self.checkExpr(current_function_id, return_type, machine_expr);
                const machine_id = switch (self.module.types.kind(machine_type)) {
                    .machine_type => |id| id,
                    else => {
                        try self.reportAt(.InvalidCall, "Result expects a machine argument", expr.span);
                        return error.InvalidSemanticModule;
                    },
                };
                break :blk self.module.hir.getMachine(machine_id).return_type;
            },
            .arena_alloc => |arena_alloc| blk: {
                const arena_type = try self.checkExpr(current_function_id, return_type, arena_alloc.arena_expr);
                const expected_arena_pointer = try self.module.types.addPointerType(self.module.types.arenaType());
                if (!sameType(arena_type, expected_arena_pointer)) {
                    try self.reportAt(.ArenaAllocRequiresArenaPointer, "Arena.alloc requires an Arena* argument", self.exprSpan(arena_alloc.arena_expr));
                    return error.InvalidSemanticModule;
                }

                try self.checkArenaAllocationEffect(current_function_id, expr.span);
                try self.checkArenaAllocatedType(arena_alloc.allocated_type, expr.span);

                const expected_result = try self.module.types.addPointerType(arena_alloc.allocated_type);
                if (!sameType(arena_alloc.result_type, expected_result)) {
                    try self.reportAt(.TypeMismatch, "arena allocation result type must be a raw pointer to the allocated type", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk arena_alloc.result_type;
            },
            // ─────────────────────────────────────────────────────────────────────────────
            // Match/enum checking
            // ─────────────────────────────────────────────────────────────────────────────

            .struct_literal => |literal| blk: {
                if (literal.struct_id.index >= self.module.hir.structs.items.len) {
                    try self.reportAt(.UnknownStructLiteralType, "struct literal type must name a top-level struct", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (!sameType(literal.type_id, try self.structType(literal.struct_id))) {
                    try self.reportAt(.UnknownStructLiteralType, "struct literal type must name a top-level struct", expr.span);
                    return error.InvalidSemanticModule;
                }
                const struct_decl = self.module.hir.getStruct(literal.struct_id);
                var seen = std.AutoHashMap(hir.FieldId, diagnostics.SourceSpan).init(self.allocator);
                defer seen.deinit();
                for (literal.fields) |field_value| {
                    if (field_value.field_id.index >= self.module.hir.fields.items.len) {
                        try self.reportAt(.UnknownStructLiteralField, "unknown struct literal field", field_value.span);
                        return error.InvalidSemanticModule;
                    }
                    const field = self.module.hir.getField(field_value.field_id);
                    if (field.parent.index != literal.struct_id.index) {
                        try self.reportAt(.UnknownStructLiteralField, "unknown struct literal field", field_value.span);
                        return error.InvalidSemanticModule;
                    }
                    if (seen.contains(field_value.field_id)) {
                        try self.reportAt(.DuplicateStructLiteralField, "duplicate struct literal field", field_value.span);
                        return error.InvalidSemanticModule;
                    }
                    try seen.put(field_value.field_id, field_value.span);
                    const value_type = try self.checkExpr(current_function_id, return_type, field_value.value);
                    if (!sameType(value_type, field.type_id)) {
                        try self.reportAt(.StructFieldInitializerTypeMismatch, "struct field initializer type mismatch", self.exprSpan(field_value.value));
                        return error.InvalidSemanticModule;
                    }
                }
                for (struct_decl.fields) |field_id| {
                    if (!seen.contains(field_id)) {
                        try self.reportAt(.MissingStructLiteralField, "missing struct literal field", expr.span);
                        return error.InvalidSemanticModule;
                    }
                }
                break :blk literal.type_id;
            },
            .field_access => |field_access| blk: {
                const receiver_type = try self.checkExpr(current_function_id, return_type, field_access.receiver);
                const receiver_kind = self.module.types.kind(receiver_type);
                if (receiver_kind != .struct_type) {
                    try self.reportAt(.FieldAccessNonStruct, "field access receiver must be a struct value", expr.span);
                    return error.InvalidSemanticModule;
                }
                const field_id = self.findField(receiver_kind.struct_type, field_access.field_name) orelse {
                    try self.reportAt(.UnknownFieldAccess, "unknown field on struct value", field_access.field_span);
                    return error.InvalidSemanticModule;
                };
                break :blk self.module.hir.getField(field_id).type_id;
            },
            .enum_constructor => |constructor| blk: {
                const variant = self.module.hir.getVariant(constructor.variant_id);
                if (variant.parent.index != constructor.enum_id.index or constructor.args.len != variant.payload_fields.len) {
                    try self.reportAt(.EnumConstructorArityMismatch, "enum constructor argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                for (constructor.args, variant.payload_fields) |arg, payload_id| {
                    const arg_type = try self.checkExpr(current_function_id, return_type, arg);
                    const payload_type = self.module.hir.getEnumPayloadField(payload_id).type_id;
                    if (!sameType(arg_type, payload_type)) {
                        try self.reportAt(.EnumConstructorTypeMismatch, "enum constructor argument type mismatch", self.exprSpan(arg));
                        return error.InvalidSemanticModule;
                    }
                }
                break :blk try self.enumType(constructor.enum_id);
            },
            // ─────────────────────────────────────────────────────────────────────────────
            // Decide checking
            // ─────────────────────────────────────────────────────────────────────────────

            .decide => |decide| blk: {
                if (!sameType(decide.enum_type, try self.enumType(decide.enum_id))) {
                    try self.reportAt(.UnknownDecideEnum, "decide target type must be an enum", expr.span);
                    return error.InvalidSemanticModule;
                }
                var has_unconditional = false;
                for (decide.arms) |arm| {
                    const variant = self.module.hir.getVariant(arm.variant_id);
                    if (variant.parent.index != decide.enum_id.index) {
                        try self.reportAt(.UnknownDecideVariant, "unknown decide variant", arm.span);
                        return error.InvalidSemanticModule;
                    }
                    if (variant.payload_fields.len != 0) {
                        try self.reportAt(.DecideVariantHasPayload, "decide candidate variant must not have payload fields", arm.span);
                        return error.InvalidSemanticModule;
                    }
                    if (arm.condition) |condition| {
                        const condition_type = try self.checkExpr(current_function_id, return_type, condition);
                        if (!self.isBool(condition_type)) {
                            try self.reportAt(.DecideConditionNotBool, "decide arm condition must be bool", self.exprSpan(condition));
                            return error.InvalidSemanticModule;
                        }
                    } else {
                        has_unconditional = true;
                    }
                    const score_type = try self.checkExpr(current_function_id, return_type, arm.score);
                    if (!self.isInt(score_type)) {
                        try self.reportAt(.DecideScoreNotInt, "decide arm score must be int", self.exprSpan(arm.score));
                        return error.InvalidSemanticModule;
                    }
                }
                if (!has_unconditional) {
                    try self.reportAt(.DecideMissingUnconditionalArm, "decide expression requires at least one unconditional arm", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk decide.enum_type;
            },
            .unary => |unary| blk: {
                const operand_type = try self.checkExpr(current_function_id, return_type, unary.operand);
                switch (unary.op) {
                    .negate => {
                        try self.requireInt(operand_type, "arithmetic unary operator requires int operand", expr.span);
                        break :blk self.module.types.intType();
                    },
                    .logical_not => {
                        try self.requireBool(operand_type, "logical unary operator requires bool operand", expr.span);
                        break :blk self.module.types.boolType();
                    },
                }
            },
            .address_of => |operand| blk: {
                const place_type = try self.addressableExprType(current_function_id, return_type, operand, expr.span);
                break :blk try self.module.types.addPointerType(place_type);
            },
            .deref => |operand| blk: {
                const operand_type = try self.checkExpr(current_function_id, return_type, operand);
                const pointee = switch (self.module.types.kind(operand_type)) {
                    .pointer => |pointer| pointer.pointee,
                    else => {
                        try self.reportAt(.DerefRequiresPointer, "dereference requires raw pointer operand", expr.span);
                        return error.InvalidSemanticModule;
                    },
                };
                if (self.unsafe_depth == 0) {
                    try self.reportAt(.DerefRequiresUnsafe, "raw pointer dereference requires unsafe context", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk pointee;
            },
            .move_expr => |operand| try self.movePlaceExprType(current_function_id, return_type, operand, expr.span),
            .manual_init_assume => |slot| blk: {
                if (self.unsafe_depth == 0) {
                    try self.reportAt(.ManualInitAssumeInitRequiresUnsafe, "manualAssumeInit requires unsafe context", expr.span);
                    return error.InvalidSemanticModule;
                }
                const slot_type = try self.checkExpr(current_function_id, return_type, slot);
                const payload = self.module.types.manualInitPayload(slot_type) orelse {
                    try self.reportAt(.ManualInitInvalidOperation, "manualAssumeInit requires ManualInit<T> operand", expr.span);
                    return error.InvalidSemanticModule;
                };
                break :blk payload;
            },
            // ─────────────────────────────────────────────────────────────────────────────
            // Result/try checking
            // ─────────────────────────────────────────────────────────────────────────────

            .try_expr => |operand| blk: {
                const operand_type = try self.checkExpr(current_function_id, return_type, operand);
                const operand_shape = self.module.resultShapeForType(operand_type) orelse {
                    try self.reportAt(.TryOperandNotResult, "try operand must be a Result-shaped enum", expr.span);
                    return error.InvalidSemanticModule;
                };
                if (self.module.resultShapeForType(return_type) == null) {
                    try self.reportAt(.TryOutsideResultFunction, "try requires the enclosing function to return a Result-shaped enum", expr.span);
                    return error.InvalidSemanticModule;
                }
                if (!sameType(operand_type, return_type)) {
                    try self.reportAt(.TryResultTypeMismatch, "try operand Result type must match the enclosing function return type", expr.span);
                    return error.InvalidSemanticModule;
                }
                break :blk operand_shape.ok_type;
            },
            .binary => |binary| blk: {
                const left_type = try self.checkExpr(current_function_id, return_type, binary.left);
                const right_type = try self.checkExpr(current_function_id, return_type, binary.right);
                switch (binary.op) {
                    .add, .subtract, .multiply, .divide, .modulo => {
                        try self.requireIntPair(left_type, right_type, "arithmetic binary operator requires int operands", expr.span);
                        break :blk self.module.types.intType();
                    },
                    .less, .less_equal, .greater, .greater_equal => {
                        try self.requireIntPair(left_type, right_type, "comparison operator requires int operands", expr.span);
                        break :blk self.module.types.boolType();
                    },
                    .equal_equal, .bang_equal => {
                        if (!sameType(left_type, right_type) or (!self.isInt(left_type) and !self.isBool(left_type))) {
                            try self.reportAt(.TypeMismatch, "equality operator requires matching int or bool operands", expr.span);
                            return error.InvalidSemanticModule;
                        }
                        break :blk self.module.types.boolType();
                    },
                    .logical_and, .logical_or => {
                        try self.requireBoolPair(left_type, right_type, "logical binary operator requires bool operands", expr.span);
                        break :blk self.module.types.boolType();
                    },
                }
            },
        };
    }

    fn reportCompileTimeError(self: *Checker, err: compile_time.CompileTimeError, span: diagnostics.SourceSpan) CheckError!void {
        switch (err) {
            error.UnsupportedExpression => try self.reportAt(.CompileTimeUnsupportedExpression, "compile-time expression uses an unsupported expression form", span),
            error.UnsupportedStatement => try self.reportAt(.CompileTimeUnsupportedStatement, "compile-time function executed an unsupported statement form", span),
            error.TypeMismatch => try self.reportAt(.CompileTimeTypeMismatch, "compile-time expression has an unsupported value type mismatch", span),
            error.DivisionByZero => try self.reportAt(.CompileTimeDivisionByZero, "compile-time expression divides by zero", span),
            error.Overflow => try self.reportAt(.CompileTimeOverflow, "compile-time expression overflows", span),
            error.EvaluationFailed => try self.reportAt(.CompileTimeEvaluationFailed, "compile-time expression evaluation failed", span),
            error.FunctionRequired => try self.reportAt(.CompileTimeFunctionRequired, "compile-time call requires a function marked comptime", span),
            error.UnsupportedBody => try self.reportAt(.CompileTimeFunctionUnsupportedBody, "compile-time function body is unsupported", span),
            error.UnsupportedSignature => try self.reportAt(.CompileTimeFunctionUnsupportedSignature, "compile-time function signature is unsupported", span),
            error.RecursionLimit => try self.reportAt(.CompileTimeRecursionLimit, "compile-time function recursion limit exceeded", span),
            error.ArgumentTypeMismatch => try self.reportAt(.CompileTimeArgumentTypeMismatch, "compile-time function argument type mismatch", span),
            error.UnsupportedLocalType => try self.reportAt(.CompileTimeUnsupportedLocalType, "compile-time local type is unsupported", span),
            error.MissingReturn => try self.reportAt(.CompileTimeMissingReturn, "compile-time function did not return on the executed path", span),
            error.IfRequiresBool => try self.reportAt(.CompileTimeIfRequiresBool, "compile-time if condition must evaluate to bool", span),
            error.UnboundLocal => try self.reportAt(.CompileTimeUnboundLocal, "compile-time local reference has no active binding", span),
            error.AssignmentTypeMismatch => try self.reportAt(.CompileTimeAssignmentTypeMismatch, "compile-time local assignment type mismatch", span),
            error.FuelExhausted => try self.reportAt(.CompileTimeFuelExhausted, "compile-time evaluation exceeded its step limit; likely non-terminating or too-expensive compile-time execution", span),
            error.WhileRequiresBool => try self.reportAt(.CompileTimeWhileRequiresBool, "compile-time while condition must evaluate to bool", span),
            error.CapabilityNotGranted => try self.reportAt(.CompileTimeCapabilityNotGranted, "compile-time capability not granted; capability-bearing comptime function cannot be evaluated yet", span),
            error.TargetMetadataUnavailable => try self.reportAt(.CompileTimeTargetMetadataUnavailable, "compile-time target metadata is unavailable", span),
        }
    }

    fn coerceDynCallArg(self: *Checker, arg_expr: hir.ExprId, arg_type: types.TypeId, param_type: types.TypeId) CheckError!hir.ExprId {
        const target_dyn = switch (self.module.types.kind(param_type)) {
            .dyn_interface => |dyn| dyn,
            else => return arg_expr,
        };

        if (sameType(arg_type, param_type)) return arg_expr;

        if (self.module.types.kind(arg_type) == .dyn_interface) return arg_expr;

        if (!self.isAddressablePlace(arg_expr)) {
            try self.reportAt(.DynCoercionRequiresPlace, "dyn coercion requires an addressable concrete place", self.exprSpan(arg_expr));
            return error.InvalidSemanticModule;
        }

        const impl_id = self.module.hir.findInterfaceImpl(target_dyn.interface_id, arg_type) orelse {
            try self.reportAt(.DynCoercionRequiresImpl, "dyn coercion requires an interface impl for the concrete type", self.exprSpan(arg_expr));
            return error.InvalidSemanticModule;
        };

        return try self.module.hir.addExpr(.{ .dyn_coerce = .{
            .source = arg_expr,
            .interface_id = target_dyn.interface_id,
            .impl_id = impl_id,
            .result_type = param_type,
        } }, self.exprSpan(arg_expr));
    }

    fn requireDynCoerceValid(self: *Checker, coerce: hir.HirDynCoerce, span: diagnostics.SourceSpan) CheckError!void {
        const dyn = switch (self.module.types.kind(coerce.result_type)) {
            .dyn_interface => |dyn| dyn,
            else => {
                try self.reportAt(.TypeMismatch, "dyn coercion result type must be dyn interface", span);
                return error.InvalidSemanticModule;
            },
        };
        if (dyn.interface_id.index != coerce.interface_id.index) {
            try self.reportAt(.TypeMismatch, "dyn coercion interface does not match result type", span);
            return error.InvalidSemanticModule;
        }
        if (coerce.impl_id.index >= self.module.hir.interface_impls.items.len) {
            try self.reportAt(.DynCoercionRequiresImpl, "dyn coercion requires an interface impl for the concrete type", span);
            return error.InvalidSemanticModule;
        }
        const interface_impl = self.module.hir.getInterfaceImpl(coerce.impl_id);
        if (interface_impl.interface_id.index != coerce.interface_id.index) {
            try self.reportAt(.DynCoercionRequiresImpl, "dyn coercion requires an interface impl for the concrete type", span);
            return error.InvalidSemanticModule;
        }
        const source_type = try self.checkExpr(null, self.module.types.voidType(), coerce.source);
        if (!sameType(source_type, interface_impl.target_type)) {
            try self.reportAt(.DynCoercionRequiresImpl, "dyn coercion requires an interface impl for the concrete type", span);
            return error.InvalidSemanticModule;
        }
        if (!self.isAddressablePlace(coerce.source)) {
            try self.reportAt(.DynCoercionRequiresPlace, "dyn coercion requires an addressable concrete place", span);
            return error.InvalidSemanticModule;
        }
    }

    fn isAddressablePlace(self: *Checker, expr_id: hir.ExprId) bool {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref, .param_ref => true,
            .group => |inner| self.isAddressablePlace(inner),
            .field_access => |field_access| self.isAddressableBase(field_access.receiver),
            else => false,
        };
    }

    fn isAddressableBase(self: *Checker, expr_id: hir.ExprId) bool {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref, .param_ref => true,
            .group => |inner| self.isAddressableBase(inner),
            else => false,
        };
    }

    fn checkCompileTimeFunctionEligibility(self: *Checker, function: hir.HirFunction) CheckError!void {
        try self.checkCompileTimeCapabilities(function);
        if (!self.isCompileTimeScalar(function.return_type)) {
            try self.reportAt(.CompileTimeFunctionUnsupportedSignature, "compile-time function signature is unsupported", function.span);
            return error.InvalidSemanticModule;
        }
        for (function.params) |param_id| {
            const param = self.module.hir.getParam(param_id);
            if (!self.isCompileTimeScalar(param.type_id)) {
                try self.reportAt(.CompileTimeFunctionUnsupportedSignature, "compile-time function signature is unsupported", param.span);
                return error.InvalidSemanticModule;
            }
        }

        const body = function.body orelse {
            try self.reportAt(.CompileTimeFunctionUnsupportedBody, "compile-time function body is unsupported", function.span);
            return error.InvalidSemanticModule;
        };
        const body_stmt = self.module.hir.getStmt(body);
        if (body_stmt.kind != .block) {
            try self.reportAt(.CompileTimeFunctionUnsupportedBody, "compile-time function body is unsupported", body_stmt.span);
            return error.InvalidSemanticModule;
        }
    }

    fn checkCompileTimeCapabilities(self: *Checker, function: hir.HirFunction) CheckError!void {
        if (function.compile_time_capabilities.len == 0) return;
        if (!function.is_compile_time) {
            try self.reportAt(.CompileTimeCapabilityListRequiresFunction, "compile-time capability list requires a comptime function", function.span);
            return error.InvalidSemanticModule;
        }
        for (function.compile_time_capabilities, 0..) |capability, index| {
            const resolved = hir.CompileTimeCapability.fromName(capability.name) orelse {
                try self.reportAt(.CompileTimeUnknownCapability, "unknown compile-time capability", capability.span);
                return error.InvalidSemanticModule;
            };
            for (function.compile_time_capabilities[0..index]) |previous| {
                const previous_resolved = hir.CompileTimeCapability.fromName(previous.name) orelse continue;
                if (previous_resolved == resolved) {
                    try self.reportAt(.CompileTimeDuplicateCapability, "duplicate compile-time capability", capability.span);
                    return error.InvalidSemanticModule;
                }
            }
        }
    }

    fn isCompileTimeScalar(self: *Checker, type_id: types.TypeId) bool {
        return self.isInt(type_id) or self.isBool(type_id);
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Type helper functions
    // ─────────────────────────────────────────────────────────────────────────────

    fn containsTypeParam(self: *Checker, type_id: types.TypeId) bool {
        return switch (self.module.types.kind(type_id)) {
            .type_param => true,
            .pointer => |pointer| self.containsTypeParam(pointer.pointee),
            .manual_init => |manual_init| self.containsTypeParam(manual_init.payload),
            else => false,
        };
    }

    fn placeType(self: *Checker, target: hir.AssignTarget, span: diagnostics.SourceSpan) CheckError!types.TypeId {
        return switch (target) {
            .local => |id| self.module.hir.getLocal(id).type_id,
            .param => |id| self.module.hir.getParam(id).type_id,
            .field => |field_place| blk: {
                const base_type = switch (field_place.base) {
                    .local => |id| self.module.hir.getLocal(id).type_id,
                    .param => |id| self.module.hir.getParam(id).type_id,
                };
                const base_kind = self.module.types.kind(base_type);
                if (base_kind != .struct_type) {
                    try self.reportAt(.FieldAccessNonStruct, "field assignment receiver must be a struct place", span);
                    return error.InvalidSemanticModule;
                }
                if (field_place.field_id.index >= self.module.hir.fields.items.len) {
                    try self.reportAt(.UnknownFieldAccess, "unknown field on struct value", field_place.field_span);
                    return error.InvalidSemanticModule;
                }
                const field = self.module.hir.getField(field_place.field_id);
                if (field.parent.index != base_kind.struct_type.index) {
                    try self.reportAt(.UnknownFieldAccess, "unknown field on struct value", field_place.field_span);
                    return error.InvalidSemanticModule;
                }
                break :blk field.type_id;
            },
        };
    }

    fn findField(self: *Checker, struct_id: hir.StructId, field_name: hir.SymbolId) ?hir.FieldId {
        const struct_decl = self.module.hir.getStruct(struct_id);
        for (struct_decl.fields) |field_id| {
            const field = self.module.hir.getField(field_id);
            if (field.name.index == field_name.index) return field_id;
        }
        return null;
    }

    fn addressableExprType(self: *Checker, current_function_id: ?hir.FunctionId, return_type: types.TypeId, expr_id: hir.ExprId, span: diagnostics.SourceSpan) CheckError!types.TypeId {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| self.module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.module.hir.getParam(param_id).type_id,
            .machine_param_ref => |param_id| self.module.hir.getMachineParam(param_id).type_id,
            .group => |inner| try self.addressableExprType(current_function_id, return_type, inner, span),
            .field_access => |field_access| blk: {
                const receiver_type = try self.checkExpr(current_function_id, return_type, field_access.receiver);
                const receiver_kind = self.module.types.kind(receiver_type);
                if (receiver_kind != .struct_type) {
                    try self.reportAt(.FieldAccessNonStruct, "field access receiver must be a struct value", expr.span);
                    return error.InvalidSemanticModule;
                }
                const field_id = self.findField(receiver_kind.struct_type, field_access.field_name) orelse {
                    try self.reportAt(.UnknownFieldAccess, "unknown field on struct value", field_access.field_span);
                    return error.InvalidSemanticModule;
                };
                if (self.addressableBase(field_access.receiver) == null) {
                    try self.reportAt(.AddressOfRequiresPlace, "address-of requires a local, parameter, or one-level field place", span);
                    return error.InvalidSemanticModule;
                }
                break :blk self.module.hir.getField(field_id).type_id;
            },
            else => {
                try self.reportAt(.AddressOfRequiresPlace, "address-of requires a local, parameter, or one-level field place", span);
                return error.InvalidSemanticModule;
            },
        };
    }

    fn movePlaceExprType(self: *Checker, current_function_id: ?hir.FunctionId, return_type: types.TypeId, expr_id: hir.ExprId, span: diagnostics.SourceSpan) CheckError!types.TypeId {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| self.module.hir.getLocal(local_id).type_id,
            .param_ref => |param_id| self.module.hir.getParam(param_id).type_id,
            .machine_param_ref => |param_id| self.module.hir.getMachineParam(param_id).type_id,
            .group => |inner| try self.movePlaceExprType(current_function_id, return_type, inner, span),
            .field_access => |field_access| {
                _ = try self.checkExpr(current_function_id, return_type, field_access.receiver);
                try self.reportAt(.PartialMoveUnsupported, "field and partial moves are not supported yet", span);
                return error.InvalidSemanticModule;
            },
            else => {
                try self.reportAt(.MoveRequiresPlace, "move requires a whole local or parameter place", span);
                return error.InvalidSemanticModule;
            },
        };
    }

    fn addressableBase(self: *Checker, expr_id: hir.ExprId) ?hir.AssignBase {
        const expr = self.module.hir.getExpr(expr_id).*;
        return switch (expr.kind) {
            .local_ref => |local_id| .{ .local = local_id },
            .param_ref => |param_id| .{ .param = param_id },
            .machine_param_ref => return null,
            .group => |inner| self.addressableBase(inner),
            else => null,
        };
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Generic function instantiation
    // ─────────────────────────────────────────────────────────────────────────────

    const TypeSubstitution = struct {
        type_params: []const hir.HirTypeParam,
        concrete_types: []const types.TypeId,
    };

    fn genericFunctionFor(self: *Checker, function_id: hir.FunctionId) ?hir.GenericFunctionId {
        for (self.module.hir.generic_functions.items, 0..) |generic, index| {
            if (generic.function.index == function_id.index) return .{ .index = @intCast(index) };
        }
        return null;
    }

    fn instantiateGenericCall(self: *Checker, generic_id: hir.GenericFunctionId, call: anytype, arg_types: []const types.TypeId, span: diagnostics.SourceSpan) CheckError!hir.FunctionId {
        const generic = self.module.hir.getGenericFunction(generic_id).*;
        const generic_function = self.module.hir.getFunction(generic.function);
        if (call.args.len != generic_function.params.len) {
            try self.reportAt(.InvalidCall, "function call argument count mismatch", span);
            return error.InvalidSemanticModule;
        }
        if (self.active_instantiation_depth >= max_instantiation_depth) {
            try self.reportAt(.UnsupportedGenericInstantiation, "generic instantiation depth limit exceeded", span);
            return error.InvalidSemanticModule;
        }

        const inferred = try self.allocator.alloc(?types.TypeId, generic.type_params.len);
        defer self.allocator.free(inferred);
        @memset(inferred, null);

        for (generic_function.params, arg_types) |param_id, arg_type| {
            const param_type = self.module.hir.getParam(param_id).type_id;
            try self.inferType(param_type, arg_type, generic.type_params, inferred, span);
        }

        var concrete_types = try self.allocator.alloc(types.TypeId, generic.type_params.len);
        errdefer self.allocator.free(concrete_types);
        for (inferred, 0..) |maybe_type, index| {
            concrete_types[index] = maybe_type orelse {
                try self.reportAt(.GenericTypeParameterUninferred, "generic type parameter could not be inferred from call arguments", span);
                return error.InvalidSemanticModule;
            };
        }

        try self.checkConceptConstraints(generic, concrete_types, span);

        if (self.findInstantiation(generic_id, concrete_types)) |existing| {
            self.allocator.free(concrete_types);
            return existing;
        }

        const function_id = try self.cloneGenericFunction(generic_id, concrete_types, span);
        try self.instantiations.append(self.allocator, .{
            .generic_id = generic_id,
            .type_args = concrete_types,
            .function = function_id,
        });

        self.active_instantiation_depth += 1;
        defer self.active_instantiation_depth -= 1;
        const concrete_function = self.module.hir.getFunction(function_id);
        try self.checkDropParams(function_id, concrete_function.*);
        if (concrete_function.body) |body| try self.checkStmt(function_id, body, concrete_function.return_type);
        try self.checkOpaqueAllocationTypeSignature(concrete_function.*);
        return function_id;
    }

    fn checkConceptConstraints(self: *Checker, generic: hir.HirGenericFunction, concrete_types: []const types.TypeId, span: diagnostics.SourceSpan) CheckError!void {
        const subst = TypeSubstitution{ .type_params = generic.type_params, .concrete_types = concrete_types };
        for (generic.type_params) |param| {
            const constraint = param.constraint orelse continue;
            const concept_id = constraint.concept_id orelse {
                try self.reportAt(.UnsupportedConceptConstraint, "unsupported concept constraint; expected Concept<T>", constraint.span);
                return error.InvalidSemanticModule;
            };
            if (constraint.type_args.len != 1) {
                try self.reportAt(.UnsupportedConceptConstraint, "unsupported concept constraint; expected Concept<T>", constraint.span);
                return error.InvalidSemanticModule;
            }
            const target_type = try self.substituteType(constraint.type_args[0], subst, span);
            if (self.module.hir.findConceptImpl(concept_id, target_type) == null) {
                const concept = self.module.hir.getConcept(concept_id);
                var type_buffer: std.Io.Writer.Allocating = .init(self.allocator);
                defer type_buffer.deinit();
                try self.writeTypeSuffix(&type_buffer.writer, target_type);
                if (self.diagnostics) |bag| {
                    try bag.append(try diagnostics.unsatisfiedConceptConstraint(self.allocator, span, self.module.interner.text(concept.name), type_buffer.written()));
                }
                return error.InvalidSemanticModule;
            }
        }
    }

    fn resolveWitnessForRequirement(self: *Checker, concept_id: hir.ConceptId, requirement_index: u32, subst: TypeSubstitution, span: diagnostics.SourceSpan) CheckError!hir.FunctionId {
        const generic_constraint = try self.findConstraintForConcept(concept_id, subst, span);
        const target_type = try self.substituteType(generic_constraint.type_args[0], subst, span);
        const impl_id = self.module.hir.findConceptImpl(concept_id, target_type) orelse {
            try self.reportAt(.UnsatisfiedConceptConstraint, "missing concept impl for requirement call", span);
            return error.InvalidSemanticModule;
        };
        const concept = self.module.hir.getConcept(concept_id);
        if (requirement_index >= concept.requirements.len) {
            try self.reportAt(.InvalidConceptRequirementCall, "invalid concept requirement call", span);
            return error.InvalidSemanticModule;
        }
        const requirement = concept.requirements[requirement_index];
        const concept_impl = self.module.hir.getConceptImpl(impl_id);
        for (concept_impl.functions) |function_id| {
            const function = self.module.hir.getFunction(function_id);
            if (function.name.index == requirement.name.index) return function_id;
        }
        try self.reportAt(.InvalidConceptRequirementCall, "concept impl does not provide required witness", span);
        return error.InvalidSemanticModule;
    }

    fn findConstraintForConcept(self: *Checker, concept_id: hir.ConceptId, subst: TypeSubstitution, span: diagnostics.SourceSpan) CheckError!hir.HirTypeConstraint {
        for (subst.type_params) |param| {
            const constraint = param.constraint orelse continue;
            if (constraint.concept_id) |candidate| {
                if (candidate.index == concept_id.index) return constraint;
            }
        }
        try self.reportAt(.InvalidConceptRequirementCall, "concept requirement call is not backed by an active constraint", span);
        return error.InvalidSemanticModule;
    }

    fn findInstantiation(self: *Checker, generic_id: hir.GenericFunctionId, concrete_types: []const types.TypeId) ?hir.FunctionId {
        for (self.instantiations.items) |entry| {
            if (entry.generic_id.index != generic_id.index or entry.type_args.len != concrete_types.len) continue;
            var matches = true;
            for (entry.type_args, concrete_types) |left, right| {
                if (!sameType(left, right)) {
                    matches = false;
                    break;
                }
            }
            if (matches) return entry.function;
        }
        return null;
    }

    fn inferType(self: *Checker, pattern: types.TypeId, actual: types.TypeId, type_params: []const hir.HirTypeParam, inferred: []?types.TypeId, span: diagnostics.SourceSpan) CheckError!void {
        if (self.typeParamIndex(pattern, type_params)) |index| {
            try self.recordInference(index, actual, inferred, span);
            return;
        }
        switch (self.module.types.kind(pattern)) {
            .pointer => |pattern_pointer| {
                const actual_pointer = switch (self.module.types.kind(actual)) {
                    .pointer => |pointer| pointer,
                    else => {
                        try self.reportAt(.UnsupportedGenericInstantiation, "generic pointer parameter requires pointer argument", span);
                        return error.InvalidSemanticModule;
                    },
                };
                try self.inferType(pattern_pointer.pointee, actual_pointer.pointee, type_params, inferred, span);
            },
            .type_param => {
                try self.reportAt(.UnsupportedGenericInstantiation, "unsupported generic type parameter pattern", span);
                return error.InvalidSemanticModule;
            },
            else => if (!sameType(pattern, actual)) {
                try self.reportAt(.InvalidCall, "function call argument type mismatch", span);
                return error.InvalidSemanticModule;
            },
        }
    }

    fn recordInference(self: *Checker, index: usize, actual: types.TypeId, inferred: []?types.TypeId, span: diagnostics.SourceSpan) CheckError!void {
        if (inferred[index]) |existing| {
            if (!sameType(existing, actual)) {
                try self.reportAt(.GenericTypeInferenceConflict, "generic type inference produced conflicting concrete types", span);
                return error.InvalidSemanticModule;
            }
        } else {
            inferred[index] = actual;
        }
    }

    fn typeParamIndex(self: *Checker, type_id: types.TypeId, type_params: []const hir.HirTypeParam) ?usize {
        _ = self;
        for (type_params, 0..) |param, index| {
            if (sameType(type_id, param.type_id)) return index;
        }
        return null;
    }

    fn cloneGenericFunction(self: *Checker, generic_id: hir.GenericFunctionId, concrete_types: []const types.TypeId, span: diagnostics.SourceSpan) CheckError!hir.FunctionId {
        const generic = self.module.hir.getGenericFunction(generic_id).*;
        const source_function = self.module.hir.getFunction(generic.function).*;
        const subst = TypeSubstitution{ .type_params = generic.type_params, .concrete_types = concrete_types };
        const generated_name = try self.instantiatedFunctionName(source_function.name, concrete_types);
        const return_type = try self.substituteType(source_function.return_type, subst, span);
        const function_id = try self.module.hir.addFunctionWithSafety(generated_name, return_type, source_function.is_unsafe, source_function.span);
        self.module.hir.setFunctionAllocationEffect(function_id, source_function.allocation_effect);
        self.module.hir.markFunctionInstantiation(function_id);

        var param_map = std.AutoHashMap(hir.ParamId, hir.ParamId).init(self.allocator);
        defer param_map.deinit();
        var local_map = std.AutoHashMap(hir.LocalId, hir.LocalId).init(self.allocator);
        defer local_map.deinit();

        for (source_function.params) |old_param_id| {
            const old_param = self.module.hir.getParam(old_param_id);
            const new_type = try self.substituteType(old_param.type_id, subst, old_param.span);
            const new_param = try self.module.hir.addParam(function_id, old_param.name, new_type, old_param.span);
            try param_map.put(old_param_id, new_param);
        }
        for (source_function.locals) |old_local_id| {
            const old_local = self.module.hir.getLocal(old_local_id);
            const new_type = try self.substituteType(old_local.type_id, subst, old_local.span);
            const new_local = try self.module.hir.addLocal(function_id, old_local.name, new_type, old_local.span);
            try local_map.put(old_local_id, new_local);
        }
        if (source_function.body) |body| {
            const new_body = try self.cloneStmt(body, subst, &param_map, &local_map, function_id, span);
            self.module.hir.setFunctionBody(function_id, new_body);
        }
        return function_id;
    }

    fn substituteType(self: *Checker, type_id: types.TypeId, subst: TypeSubstitution, span: diagnostics.SourceSpan) CheckError!types.TypeId {
        if (self.typeParamIndex(type_id, subst.type_params)) |index| return subst.concrete_types[index];
        return switch (self.module.types.kind(type_id)) {
            .arena, .allocator, .alloc_error => type_id,
            .pointer => |pointer| try self.module.types.addPointerType(try self.substituteType(pointer.pointee, subst, span)),
            .type_param => {
                try self.reportAt(.UnsupportedGenericInstantiation, "unsupported type parameter in generic instantiation", span);
                return error.InvalidSemanticModule;
            },
            else => type_id,
        };
    }

    fn cloneStmt(self: *Checker, stmt_id: hir.StmtId, subst: TypeSubstitution, param_map: *std.AutoHashMap(hir.ParamId, hir.ParamId), local_map: *std.AutoHashMap(hir.LocalId, hir.LocalId), function_id: hir.FunctionId, span: diagnostics.SourceSpan) CheckError!hir.StmtId {
        const stmt = self.module.hir.getStmt(stmt_id).*;
        const kind: hir.HirStmtKind = switch (stmt.kind) {
            .block => |children| blk: {
                var new_children = try self.allocator.alloc(hir.StmtId, children.len);
                errdefer self.allocator.free(new_children);
                for (children, 0..) |child, index| new_children[index] = try self.cloneStmt(child, subst, param_map, local_map, function_id, span);
                break :blk .{ .block = new_children };
            },
            .return_stmt => |maybe_value| .{ .return_stmt = if (maybe_value) |value| try self.cloneExpr(value, subst, param_map, local_map, span) else null },
            .local_decl => |decl| .{ .local_decl = .{ .local = local_map.get(decl.local).?, .initializer = try self.cloneExpr(decl.initializer, subst, param_map, local_map, span) } },
            .assignment => |assignment| .{ .assignment = .{ .target = self.cloneAssignTarget(assignment.target, param_map, local_map), .value = try self.cloneExpr(assignment.value, subst, param_map, local_map, span) } },
            .expr_stmt => |expr_id| .{ .expr_stmt = try self.cloneExpr(expr_id, subst, param_map, local_map, span) },
            .discard_stmt => |expr_id| .{ .discard_stmt = try self.cloneExpr(expr_id, subst, param_map, local_map, span) },
            .arena_reset => |op| .{ .arena_reset = .{
                .arena_expr = try self.cloneExpr(op.arena_expr, subst, param_map, local_map, span),
                .arena_type = try self.substituteType(op.arena_type, subst, span),
            } },
            .arena_destroy => |op| .{ .arena_destroy = .{
                .arena_expr = try self.cloneExpr(op.arena_expr, subst, param_map, local_map, span),
                .arena_type = try self.substituteType(op.arena_type, subst, span),
            } },
            .if_stmt => |if_stmt| .{ .if_stmt = .{
                .condition = try self.cloneExpr(if_stmt.condition, subst, param_map, local_map, span),
                .then_block = try self.cloneStmt(if_stmt.then_block, subst, param_map, local_map, function_id, span),
                .else_block = if (if_stmt.else_block) |else_block| try self.cloneStmt(else_block, subst, param_map, local_map, function_id, span) else null,
            } },
            .while_stmt => |while_stmt| .{ .while_stmt = .{ .condition = try self.cloneExpr(while_stmt.condition, subst, param_map, local_map, span), .body = try self.cloneStmt(while_stmt.body, subst, param_map, local_map, function_id, span) } },
            .unsafe_block => |body| .{ .unsafe_block = try self.cloneStmt(body, subst, param_map, local_map, function_id, span) },
            .transition_stmt => return error.InvalidSemanticModule,
            .match_stmt => |match_stmt| blk: {
                var arms = try self.allocator.alloc(hir.HirMatchArm, match_stmt.arms.len);
                errdefer self.allocator.free(arms);
                for (match_stmt.arms, 0..) |arm, index| arms[index] = .{ .pattern = try self.clonePattern(arm.pattern, param_map, local_map), .pattern_span = arm.pattern_span, .body = try self.cloneStmt(arm.body, subst, param_map, local_map, function_id, span) };
                break :blk .{ .match_stmt = .{ .scrutinee = try self.cloneExpr(match_stmt.scrutinee, subst, param_map, local_map, span), .arms = arms } };
            },
        };
        return self.module.hir.addStmt(kind, stmt.span);
    }

    fn cloneExpr(self: *Checker, expr_id: hir.ExprId, subst: TypeSubstitution, param_map: *std.AutoHashMap(hir.ParamId, hir.ParamId), local_map: *std.AutoHashMap(hir.LocalId, hir.LocalId), span: diagnostics.SourceSpan) CheckError!hir.ExprId {
        const expr = self.module.hir.getExpr(expr_id).*;
        const kind: hir.HirExprKind = switch (expr.kind) {
            .int_literal => |text| .{ .int_literal = try self.allocator.dupe(u8, text) },
            .bool_literal => |value| .{ .bool_literal = value },
            .local_ref => |id| .{ .local_ref = local_map.get(id).? },
            .param_ref => |id| .{ .param_ref = param_map.get(id).? },
            .machine_param_ref => return error.InvalidSemanticModule,
            .machine_construct, .machine_step, .machine_complete, .machine_result => return error.InvalidSemanticModule,
            .call => |call| blk: {
                var args = try self.allocator.alloc(hir.ExprId, call.args.len);
                errdefer self.allocator.free(args);
                for (call.args, 0..) |arg, index| args[index] = try self.cloneExpr(arg, subst, param_map, local_map, span);
                break :blk .{ .call = .{ .function = call.function, .args = args } };
            },
            .arena_alloc => |arena_alloc| .{ .arena_alloc = .{
                .arena_expr = try self.cloneExpr(arena_alloc.arena_expr, subst, param_map, local_map, span),
                .allocated_type = try self.substituteType(arena_alloc.allocated_type, subst, span),
                .result_type = try self.substituteType(arena_alloc.result_type, subst, span),
            } },
            .concept_requirement_call => |call| blk: {
                var args = try self.allocator.alloc(hir.ExprId, call.args.len);
                errdefer self.allocator.free(args);
                for (call.args, 0..) |arg, index| args[index] = try self.cloneExpr(arg, subst, param_map, local_map, span);
                const witness = try self.resolveWitnessForRequirement(call.concept_id, call.requirement_index, subst, expr.span);
                self.module.hir.markConceptWitnessReferenced(witness);
                break :blk .{ .call = .{ .function = witness, .args = args } };
            },
            .interface_call => |call| blk: {
                var args = try self.allocator.alloc(hir.ExprId, call.args.len);
                errdefer self.allocator.free(args);
                for (call.args, 0..) |arg, index| args[index] = try self.cloneExpr(arg, subst, param_map, local_map, span);
                break :blk .{ .interface_call = .{
                    .receiver = try self.cloneExpr(call.receiver, subst, param_map, local_map, span),
                    .interface_id = call.interface_id,
                    .requirement_id = call.requirement_id,
                    .requirement_index = call.requirement_index,
                    .args = args,
                    .result_type = try self.substituteType(call.result_type, subst, span),
                } };
            },
            .enum_constructor => |constructor| blk: {
                var args = try self.allocator.alloc(hir.ExprId, constructor.args.len);
                errdefer self.allocator.free(args);
                for (constructor.args, 0..) |arg, index| args[index] = try self.cloneExpr(arg, subst, param_map, local_map, span);
                break :blk .{ .enum_constructor = .{ .enum_id = constructor.enum_id, .variant_id = constructor.variant_id, .args = args } };
            },
            .struct_literal => |literal| blk: {
                var fields = try self.allocator.alloc(hir.HirStructLiteralField, literal.fields.len);
                errdefer self.allocator.free(fields);
                for (literal.fields, 0..) |field, index| fields[index] = .{ .field_id = field.field_id, .value = try self.cloneExpr(field.value, subst, param_map, local_map, span), .span = field.span };
                break :blk .{ .struct_literal = .{ .struct_id = literal.struct_id, .type_id = try self.substituteType(literal.type_id, subst, span), .fields = fields } };
            },
            .field_access => |field_access| .{ .field_access = .{ .receiver = try self.cloneExpr(field_access.receiver, subst, param_map, local_map, span), .field_name = field_access.field_name, .field_span = field_access.field_span } },
            .target_metadata => |metadata| .{ .target_metadata = metadata },
            .decide => |decide| blk: {
                var arms = try self.allocator.alloc(hir.HirDecideArm, decide.arms.len);
                errdefer self.allocator.free(arms);
                for (decide.arms, 0..) |arm, index| arms[index] = .{ .variant_id = arm.variant_id, .condition = if (arm.condition) |condition| try self.cloneExpr(condition, subst, param_map, local_map, span) else null, .score = try self.cloneExpr(arm.score, subst, param_map, local_map, span), .span = arm.span };
                break :blk .{ .decide = .{ .enum_type = try self.substituteType(decide.enum_type, subst, span), .enum_id = decide.enum_id, .arms = arms } };
            },
            .group => |inner| .{ .group = try self.cloneExpr(inner, subst, param_map, local_map, span) },
            .unary => |unary| .{ .unary = .{ .op = unary.op, .operand = try self.cloneExpr(unary.operand, subst, param_map, local_map, span) } },
            .address_of => |operand| .{ .address_of = try self.cloneExpr(operand, subst, param_map, local_map, span) },
            .deref => |operand| .{ .deref = try self.cloneExpr(operand, subst, param_map, local_map, span) },
            .dyn_coerce => |coerce| .{ .dyn_coerce = .{
                .source = try self.cloneExpr(coerce.source, subst, param_map, local_map, span),
                .interface_id = coerce.interface_id,
                .impl_id = coerce.impl_id,
                .result_type = try self.substituteType(coerce.result_type, subst, span),
            } },
            .move_expr => |operand| .{ .move_expr = try self.cloneExpr(operand, subst, param_map, local_map, span) },
            .manual_init_assume => |slot| .{ .manual_init_assume = try self.cloneExpr(slot, subst, param_map, local_map, span) },
            .try_expr => |operand| .{ .try_expr = try self.cloneExpr(operand, subst, param_map, local_map, span) },
            .compile_time => |compile_time_expr| .{ .compile_time = .{ .operand = try self.cloneExpr(compile_time_expr.operand, subst, param_map, local_map, span), .span = compile_time_expr.span } },
            .binary => |binary| .{ .binary = .{ .op = binary.op, .left = try self.cloneExpr(binary.left, subst, param_map, local_map, span), .right = try self.cloneExpr(binary.right, subst, param_map, local_map, span) } },
            .test_intrinsic => |test_intrinsic| blk: {
                var operands = try self.allocator.alloc(hir.ExprId, test_intrinsic.operands.len);
                errdefer self.allocator.free(operands);
                for (test_intrinsic.operands, 0..) |operand, index| {
                    operands[index] = try self.cloneExpr(operand, subst, param_map, local_map, span);
                }
                break :blk .{ .test_intrinsic = .{
                    .kind = test_intrinsic.kind,
                    .operands = operands,
                    .reason = try self.allocator.dupe(u8, test_intrinsic.reason),
                    .reason_span = test_intrinsic.reason_span,
                } };
            },
        };
        return self.module.hir.addExpr(kind, expr.span);
    }

    fn cloneAssignTarget(self: *Checker, target: hir.AssignTarget, param_map: *std.AutoHashMap(hir.ParamId, hir.ParamId), local_map: *std.AutoHashMap(hir.LocalId, hir.LocalId)) hir.AssignTarget {
        return switch (target) {
            .local => |id| .{ .local = local_map.get(id).? },
            .param => |id| .{ .param = param_map.get(id).? },
            .field => |field| .{ .field = .{ .base = self.cloneAssignBase(field.base, param_map, local_map), .field_id = field.field_id, .field_span = field.field_span } },
        };
    }

    fn cloneAssignBase(self: *Checker, base: hir.AssignBase, param_map: *std.AutoHashMap(hir.ParamId, hir.ParamId), local_map: *std.AutoHashMap(hir.LocalId, hir.LocalId)) hir.AssignBase {
        _ = self;
        return switch (base) {
            .local => |id| .{ .local = local_map.get(id).? },
            .param => |id| .{ .param = param_map.get(id).? },
        };
    }

    fn clonePattern(self: *Checker, pattern: hir.HirMatchPattern, param_map: *std.AutoHashMap(hir.ParamId, hir.ParamId), local_map: *std.AutoHashMap(hir.LocalId, hir.LocalId)) CheckError!hir.HirMatchPattern {
        _ = param_map;
        return switch (pattern) {
            .int_literal => |text| .{ .int_literal = try self.allocator.dupe(u8, text) },
            .bool_literal => |value| .{ .bool_literal = value },
            .wildcard => .wildcard,
            .enum_variant => |variant| blk: {
                var bindings = try self.allocator.alloc(hir.HirPatternBinding, variant.bindings.len);
                errdefer self.allocator.free(bindings);
                for (variant.bindings, 0..) |binding, index| {
                    bindings[index] = binding;
                    bindings[index].local = local_map.get(binding.local).?;
                }
                break :blk .{ .enum_variant = .{ .enum_id = variant.enum_id, .variant_id = variant.variant_id, .bindings = bindings } };
            },
        };
    }

    fn instantiatedFunctionName(self: *Checker, base_name: hir.SymbolId, concrete_types: []const types.TypeId) CheckError!hir.SymbolId {
        var base_buffer: std.Io.Writer.Allocating = .init(self.allocator);
        defer base_buffer.deinit();
        try base_buffer.writer.writeAll(self.module.interner.text(base_name));
        try base_buffer.writer.writeAll("__");
        for (concrete_types, 0..) |type_id, index| {
            if (index != 0) try base_buffer.writer.writeAll("__");
            try self.writeTypeSuffix(&base_buffer.writer, type_id);
        }

        const candidate = try self.module.interner.intern(base_buffer.written());
        if (!self.functionNameExists(candidate)) return candidate;

        var suffix: usize = 0;
        while (true) : (suffix += 1) {
            const text = try std.fmt.allocPrint(self.allocator, "{s}__inst{d}", .{ base_buffer.written(), suffix });
            defer self.allocator.free(text);
            const suffixed = try self.module.interner.intern(text);
            if (!self.functionNameExists(suffixed)) return suffixed;
        }
    }

    fn functionNameExists(self: *Checker, name: hir.SymbolId) bool {
        for (self.module.hir.functions.items) |function| {
            if (function.name.index == name.index) return true;
        }
        for (self.module.hir.generic_functions.items) |generic| {
            if (generic.name.index == name.index) return true;
        }
        return false;
    }

    fn writeTypeSuffix(self: *Checker, writer: *std.Io.Writer, type_id: types.TypeId) CheckError!void {
        switch (self.module.types.kind(type_id)) {
            .void => try writer.writeAll("void"),
            .int => try writer.writeAll("int"),
            .bool => try writer.writeAll("bool"),
            .arena => try writer.writeAll("Arena"),
            .allocator => try writer.writeAll("Allocator"),
            .alloc_error => try writer.writeAll("AllocError"),
            .struct_type => |struct_id| try writer.print("struct_{s}", .{self.module.interner.text(self.module.hir.getStruct(struct_id).name)}),
            .enum_type => |enum_id| try writer.print("enum_{s}", .{self.module.interner.text(self.module.hir.getEnum(enum_id).name)}),
            .machine_type => |machine_id| try writer.print("machine_{s}", .{self.module.interner.text(self.module.hir.getMachine(machine_id).name)}),
            .interface_type => |interface_id| try writer.print("interface_{s}", .{self.module.interner.text(self.module.hir.getInterface(interface_id).name)}),
            .dyn_interface => |dyn| {
                if (dyn.is_mut) try writer.writeAll("mut_");
                try writer.print("dyn_interface_{s}", .{self.module.interner.text(self.module.hir.getInterface(dyn.interface_id).name)});
            },
            .pointer => |pointer| {
                try self.writeTypeSuffix(writer, pointer.pointee);
                try writer.writeAll("_ptr");
            },
            .manual_init => |manual_init| {
                try writer.writeAll("manual_init_");
                try self.writeTypeSuffix(writer, manual_init.payload);
            },
            .type_param => try writer.writeAll("type_param"),
        }
    }

    fn structType(self: *Checker, struct_id: hir.StructId) CheckError!types.TypeId {
        for (self.module.types.types.items, 0..) |kind, index| {
            if (kind == .struct_type and kind.struct_type.index == struct_id.index) return .{ .index = @intCast(index) };
        }
        try self.reportAt(.TypeMismatch, "unknown struct literal type", synthetic_span);
        return error.InvalidSemanticModule;
    }

    fn enumType(self: *Checker, enum_id: hir.EnumId) CheckError!types.TypeId {
        for (self.module.types.types.items, 0..) |kind, index| {
            if (kind == .enum_type and kind.enum_type.index == enum_id.index) return .{ .index = @intCast(index) };
        }
        try self.reportAt(.TypeMismatch, "unknown enum constructor type", synthetic_span);
        return error.InvalidSemanticModule;
    }

    fn checkAllocationEffectCall(self: *Checker, current_function_id: ?hir.FunctionId, callee: hir.HirFunction, span: diagnostics.SourceSpan) CheckError!void {
        const caller_id = current_function_id orelse return;
        const caller = self.module.hir.getFunction(caller_id);
        if (caller.allocation_effect != .noalloc or callee.allocation_effect == .noalloc) return;

        const caller_name = self.module.interner.text(caller.name);
        const callee_name = self.module.interner.text(callee.name);
        const callee_effect = allocationEffectDiagnosticName(callee.allocation_effect);
        const reason = if (callee.allocation_effect == .unspecified)
            "has no noalloc guarantee"
        else
            "has allocation effect";
        if (self.diagnostics) |bag| {
            const message = if (callee.allocation_effect == .unspecified)
                try std.fmt.allocPrint(
                    bag.allocator,
                    "noalloc function '{s}' cannot call '{s}' because it {s}",
                    .{ caller_name, callee_name, reason },
                )
            else
                try std.fmt.allocPrint(
                    bag.allocator,
                    "noalloc function '{s}' cannot call '{s}' because it {s} '{s}'",
                    .{ caller_name, callee_name, reason, callee_effect },
                );
            errdefer bag.allocator.free(message);
            try bag.append(.{
                .code = .AllocationEffectMismatch,
                .severity = .@"error",
                .message = message,
                .primary_span = span,
                .owns_message = true,
            });
        }
        return error.InvalidSemanticModule;
    }

    fn checkArenaAllocationEffect(self: *Checker, current_function_id: ?hir.FunctionId, span: diagnostics.SourceSpan) CheckError!void {
        const caller_id = current_function_id orelse return;
        const caller = self.module.hir.getFunction(caller_id);
        if (caller.is_compile_time or self.compile_time_context_depth != 0) {
            try self.reportAt(.ArenaAllocationInComptimeUnsupported, "arena allocation is not supported during compile-time execution in Phase 12 v0", span);
            return error.InvalidSemanticModule;
        }
        if (caller.allocation_effect == .noalloc) {
            try self.reportAt(.AllocationInNoAllocFunction, "noalloc function cannot perform arena allocation", span);
            return error.InvalidSemanticModule;
        }
    }

    fn checkArenaStorageOp(self: *Checker, current_function_id: hir.FunctionId, return_type: types.TypeId, op: hir.ArenaStorageOp, span: diagnostics.SourceSpan) CheckError!void {
        const caller = self.module.hir.getFunction(current_function_id);
        if (caller.is_compile_time or self.compile_time_context_depth != 0) {
            try self.reportAt(.ArenaResetDestroyInComptimeUnsupported, "Arena.reset and Arena.destroy are not supported during compile-time execution in Phase 12 v0", span);
            return error.InvalidSemanticModule;
        }

        const expected_arena_pointer = try self.module.types.addPointerType(self.module.types.arenaType());
        if (!sameType(op.arena_type, expected_arena_pointer)) {
            try self.reportAt(.ArenaResetDestroyRequiresArenaPointer, "Arena.reset and Arena.destroy require an Arena* argument", span);
            return error.InvalidSemanticModule;
        }

        const arena_type = try self.checkExpr(current_function_id, return_type, op.arena_expr);
        if (!sameType(arena_type, expected_arena_pointer)) {
            try self.reportAt(.ArenaResetDestroyRequiresArenaPointer, "Arena.reset and Arena.destroy require an Arena* argument", self.exprSpan(op.arena_expr));
            return error.InvalidSemanticModule;
        }
    }

    fn checkArenaAllocatedType(self: *Checker, allocated_type: types.TypeId, span: diagnostics.SourceSpan) CheckError!void {
        switch (self.module.arenaAllocationTypeEligibility(allocated_type)) {
            .ok => {},
            .contains_type_param => {
                try self.reportAt(.ArenaAllocRequiresConcreteType, "Arena.alloc requires a concrete allocated type", span);
                return error.InvalidSemanticModule;
            },
            .void_type => {
                try self.reportAt(.ArenaAllocRequiresConcreteType, "Arena.alloc cannot allocate void", span);
                return error.InvalidSemanticModule;
            },
            .opaque_handle => {
                try self.reportAt(.OpaqueAllocationTypeByValueUnsupported, "Arena and Allocator are opaque allocation handles and cannot be arena-allocated by value in v0", span);
                return error.InvalidSemanticModule;
            },
            .drop_required => {
                try self.reportAt(.ArenaAllocDropTypeUnsupported, "arena allocation of types that require Drop is not supported in Phase 12 v0", span);
                return error.InvalidSemanticModule;
            },
            .manual_init_drop_payload => {
                try self.reportAt(.ArenaAllocDropTypeUnsupported, "arena allocation of ManualInit<T> where T requires Drop is not supported in Phase 12 v0", span);
                return error.InvalidSemanticModule;
            },
        }
    }

    fn checkTestIntrinsic(self: *Checker, current_function_id: ?hir.FunctionId, return_type: types.TypeId, expr: hir.HirExpr, test_intrinsic: hir.HirTestIntrinsic) CheckError!void {
        switch (test_intrinsic.kind) {
            .assert_true, .assert_false, .expect_true, .expect_false, .expect_that_true, .expect_that_false => {
                if (test_intrinsic.operands.len != 1) {
                    try self.reportAt(.TestIntrinsicArityMismatch, "test intrinsic argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                const condition_type = try self.checkExpr(current_function_id, return_type, test_intrinsic.operands[0]);
                if (!self.isBool(condition_type)) {
                    try self.reportAt(.TestIntrinsicTypeMismatch, "test intrinsic argument type mismatch", self.exprSpan(test_intrinsic.operands[0]));
                    return error.InvalidSemanticModule;
                }
            },
            .expect_equal_int, .expect_equal_bool, .expect_that_equal_int, .expect_that_equal_bool => {
                if (test_intrinsic.operands.len != 2) {
                    try self.reportAt(.TestIntrinsicArityMismatch, "test intrinsic argument count mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
                const expected_type = try self.checkExpr(current_function_id, return_type, test_intrinsic.operands[0]);
                const actual_type = try self.checkExpr(current_function_id, return_type, test_intrinsic.operands[1]);
                const required_type = switch (test_intrinsic.kind) {
                    .expect_equal_int, .expect_that_equal_int => self.module.types.intType(),
                    .expect_equal_bool, .expect_that_equal_bool => self.module.types.boolType(),
                    else => unreachable,
                };
                if (!sameType(expected_type, required_type) or !sameType(actual_type, required_type)) {
                    try self.reportAt(.TestIntrinsicTypeMismatch, "test intrinsic argument type mismatch", expr.span);
                    return error.InvalidSemanticModule;
                }
            },
        }
        if (std.mem.trim(u8, test_intrinsic.reason, " \t\r\n").len == 0) {
            try self.reportAt(.TestReasonMustBeNonEmpty, "test intrinsic because reason must be non-empty", test_intrinsic.reason_span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireSame(self: *Checker, actual: types.TypeId, expected: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!sameType(actual, expected)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireCallSame(self: *Checker, actual: types.TypeId, expected: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!sameType(actual, expected)) {
            try self.reportAt(.InvalidCall, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireInt(self: *Checker, actual: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isInt(actual)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireBool(self: *Checker, actual: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isBool(actual)) {
            try self.reportAt(.InvalidConditionType, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireIntPair(self: *Checker, left: types.TypeId, right: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isInt(left) or !self.isInt(right)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn requireBoolPair(self: *Checker, left: types.TypeId, right: types.TypeId, message: []const u8, span: diagnostics.SourceSpan) CheckError!void {
        if (!self.isBool(left) or !self.isBool(right)) {
            try self.reportAt(.TypeMismatch, message, span);
            return error.InvalidSemanticModule;
        }
    }

    fn isInt(self: *Checker, actual: types.TypeId) bool {
        return sameType(actual, self.module.types.intType());
    }

    fn isBool(self: *Checker, actual: types.TypeId) bool {
        return sameType(actual, self.module.types.boolType());
    }

    fn isMustUseType(self: *Checker, actual: types.TypeId) bool {
        const kind = self.module.types.kind(actual);
        if (kind != .enum_type) return false;
        return self.module.hir.getEnum(kind.enum_type).is_must_use;
    }

    fn exprSpan(self: *Checker, expr_id: hir.ExprId) diagnostics.SourceSpan {
        return self.module.hir.getExpr(expr_id).span;
    }

    // ─────────────────────────────────────────────────────────────────────────────
    // Diagnostic helpers
    // ─────────────────────────────────────────────────────────────────────────────

    fn report(self: *Checker, code: diagnostics.DiagnosticCode, message: []const u8) !void {
        try self.reportAt(code, message, synthetic_span);
    }

    fn reportAt(self: *Checker, code: diagnostics.DiagnosticCode, message: []const u8, span: diagnostics.SourceSpan) !void {
        if (self.diagnostics) |bag| try bag.append(diagnostics.makeDiagnostic(code, .@"error", message, span));
    }
};

fn sameType(a: types.TypeId, b: types.TypeId) bool {
    return a.index == b.index;
}

fn allocationEffectDiagnosticName(effect: hir.AllocationEffect) []const u8 {
    return switch (effect) {
        .alloc => "alloc",
        .noalloc => "noalloc",
        .unspecified => "unspecified",
    };
}

// ─────────────────────────────────────────────────────────────────────────────
// Tests
// ─────────────────────────────────────────────────────────────────────────────

const TestModule = struct {
    module: semantics.SemanticModule,

    fn init() !TestModule {
        return .{ .module = try semantics.SemanticModule.init(std.testing.allocator) };
    }

    fn deinit(self: *TestModule) void {
        self.module.deinit();
    }

    fn name(self: *TestModule, text: []const u8) !hir.SymbolId {
        return self.module.interner.intern(text);
    }

    fn function(self: *TestModule, name_text: []const u8, return_type: types.TypeId) !hir.FunctionId {
        return self.module.hir.addFunction(try self.name(name_text), return_type, synthetic_span);
    }

    fn unsafeFunction(self: *TestModule, name_text: []const u8, return_type: types.TypeId) !hir.FunctionId {
        return self.module.hir.addFunctionWithSafety(try self.name(name_text), return_type, true, synthetic_span);
    }

    fn compileTimeFunction(self: *TestModule, name_text: []const u8, return_type: types.TypeId) !hir.FunctionId {
        const function_id = try self.module.hir.addFunction(try self.name(name_text), return_type, synthetic_span);
        self.module.hir.markFunctionCompileTime(function_id);
        return function_id;
    }

    fn setAllocationEffect(self: *TestModule, function_id: hir.FunctionId, effect: hir.AllocationEffect) void {
        self.module.hir.setFunctionAllocationEffect(function_id, effect);
    }

    fn interfaceDecl(self: *TestModule, name_text: []const u8) !hir.InterfaceId {
        const interface_id = try self.module.hir.addInterface(try self.name(name_text), synthetic_span);
        _ = try self.module.types.addInterfaceType(interface_id);
        return interface_id;
    }

    fn interfaceRequirement(self: *TestModule, interface_id: hir.InterfaceId, name_text: []const u8, return_type: types.TypeId) !hir.InterfaceRequirementId {
        return self.module.hir.addInterfaceRequirement(interface_id, try self.name(name_text), return_type, synthetic_span);
    }

    fn interfaceParam(self: *TestModule, requirement_id: hir.InterfaceRequirementId, name_text: []const u8, type_id: types.TypeId) !hir.InterfaceParamId {
        return self.module.hir.addInterfaceParam(requirement_id, try self.name(name_text), type_id, synthetic_span);
    }

    fn param(self: *TestModule, function_id: hir.FunctionId, name_text: []const u8, type_id: types.TypeId) !hir.ParamId {
        return self.module.hir.addParam(function_id, try self.name(name_text), type_id, synthetic_span);
    }

    fn local(self: *TestModule, function_id: hir.FunctionId, name_text: []const u8, type_id: types.TypeId) !hir.LocalId {
        return self.module.hir.addLocal(function_id, try self.name(name_text), type_id, synthetic_span);
    }

    fn block(self: *TestModule, stmts: []const hir.StmtId) !hir.StmtId {
        const owned = try std.testing.allocator.dupe(hir.StmtId, stmts);
        return self.module.hir.addStmt(.{ .block = owned }, synthetic_span);
    }

    fn setBody(self: *TestModule, function_id: hir.FunctionId, stmt_id: hir.StmtId) void {
        self.module.hir.setFunctionBody(function_id, stmt_id);
    }

    fn int(self: *TestModule, text: []const u8) !hir.ExprId {
        return self.module.hir.addExpr(.{ .int_literal = try std.testing.allocator.dupe(u8, text) }, synthetic_span);
    }

    fn boolLit(self: *TestModule, value: bool) !hir.ExprId {
        return self.module.hir.addExpr(.{ .bool_literal = value }, synthetic_span);
    }

    fn ret(self: *TestModule, expr: ?hir.ExprId) !hir.StmtId {
        return self.module.hir.addStmt(.{ .return_stmt = expr }, synthetic_span);
    }

    fn checkPass(self: *TestModule) !void {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try checkExecutable(std.testing.allocator, &self.module, &bag);
        try std.testing.expectEqual(@as(usize, 0), bag.count());
    }

    fn checkTestPass(self: *TestModule) !void {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try checkTestModule(std.testing.allocator, &self.module, &bag);
        try std.testing.expectEqual(@as(usize, 0), bag.count());
    }

    fn checkFail(self: *TestModule, expected: diagnostics.DiagnosticCode) !void {
        _ = try self.checkFailSpan(expected);
    }

    fn checkTestFail(self: *TestModule, expected: diagnostics.DiagnosticCode) !void {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try std.testing.expectError(error.InvalidSemanticModule, checkTestModule(std.testing.allocator, &self.module, &bag));
        try std.testing.expectEqual(@as(usize, 1), bag.count());
        try std.testing.expectEqual(expected, bag.diagnostics.items[0].code);
    }

    fn checkFailSpan(self: *TestModule, expected: diagnostics.DiagnosticCode) !diagnostics.SourceSpan {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try std.testing.expectError(error.InvalidSemanticModule, checkExecutable(std.testing.allocator, &self.module, &bag));
        try std.testing.expectEqual(@as(usize, 1), bag.count());
        try std.testing.expectEqual(expected, bag.diagnostics.items[0].code);
        return bag.diagnostics.items[0].primary_span;
    }

    fn checkFailMessageContains(self: *TestModule, expected: diagnostics.DiagnosticCode, needle: []const u8) !void {
        var bag = DiagnosticBag.init(std.testing.allocator);
        defer bag.deinit();
        try std.testing.expectError(error.InvalidSemanticModule, checkExecutable(std.testing.allocator, &self.module, &bag));
        try std.testing.expectEqual(@as(usize, 1), bag.count());
        try std.testing.expectEqual(expected, bag.diagnostics.items[0].code);
        try std.testing.expect(std.mem.indexOf(u8, bag.diagnostics.items[0].message, needle) != null);
    }
};

fn testSpan(start: usize) diagnostics.SourceSpan {
    return .{ .start = start, .length = 1 };
}

fn addTestStmtAt(store: *hir.HirStore, kind: hir.HirStmtKind, span: diagnostics.SourceSpan) !hir.StmtId {
    return store.addStmt(kind, span);
}

fn addTestExprAt(store: *hir.HirStore, kind: hir.HirExprKind, span: diagnostics.SourceSpan) !hir.ExprId {
    return store.addExpr(kind, span);
}

fn addTestStmt(store: *hir.HirStore, kind: hir.HirStmtKind) !hir.StmtId {
    return store.addStmt(kind, synthetic_span);
}

fn addTestExpr(store: *hir.HirStore, kind: hir.HirExprKind) !hir.ExprId {
    return store.addExpr(kind, synthetic_span);
}

fn addMainReturnInt(tm: *TestModule, value: []const u8) !hir.FunctionId {
    const main_id = try tm.function("main", tm.module.types.intType());
    const ret_stmt = try tm.ret(try tm.int(value));
    tm.setBody(main_id, try tm.block(&.{ret_stmt}));
    return main_id;
}

test "HIR checker accepts ordinary interface requirement signatures" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const interface_id = try tm.interfaceDecl("Writer");
    const requirement_id = try tm.interfaceRequirement(interface_id, "Write", tm.module.types.voidType());
    _ = try tm.interfaceParam(requirement_id, "value", tm.module.types.intType());
    _ = try tm.interfaceParam(requirement_id, "newline", tm.module.types.boolType());

    try tm.checkTestPass();
}

test "HIR checker rejects interface requirement interface return" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const interface_id = try tm.interfaceDecl("Writer");
    const writer_type = try tm.module.types.addInterfaceType(interface_id);
    _ = try tm.interfaceRequirement(interface_id, "Clone", writer_type);

    try tm.checkTestFail(.InterfaceRuntimeUnsupported);
}

test "HIR checker rejects interface requirement interface parameters" {
    var value_tm = try TestModule.init();
    defer value_tm.deinit();
    const value_interface_id = try value_tm.interfaceDecl("Writer");
    const value_writer_type = try value_tm.module.types.addInterfaceType(value_interface_id);
    const value_requirement_id = try value_tm.interfaceRequirement(value_interface_id, "Take", value_tm.module.types.voidType());
    _ = try value_tm.interfaceParam(value_requirement_id, "value", value_writer_type);
    try value_tm.checkTestFail(.InterfaceRuntimeUnsupported);

    var pointer_tm = try TestModule.init();
    defer pointer_tm.deinit();
    const pointer_interface_id = try pointer_tm.interfaceDecl("Writer");
    const pointer_writer_type = try pointer_tm.module.types.addInterfaceType(pointer_interface_id);
    const pointer_requirement_id = try pointer_tm.interfaceRequirement(pointer_interface_id, "Take", pointer_tm.module.types.voidType());
    _ = try pointer_tm.interfaceParam(pointer_requirement_id, "value", try pointer_tm.module.types.addPointerType(pointer_writer_type));
    try pointer_tm.checkTestFail(.InterfaceRuntimeUnsupported);
}

test "HIR checker accepts dyn interface function parameters" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const interface_id = try tm.interfaceDecl("Writer");
    _ = try tm.interfaceRequirement(interface_id, "Write", tm.module.types.voidType());
    const writer_dyn = try tm.module.types.addDynInterfaceType(interface_id, false);
    const writer_dyn_mut = try tm.module.types.addDynInterfaceType(interface_id, true);
    const function_id = try tm.function("Emit", tm.module.types.voidType());
    _ = try tm.param(function_id, "writer", writer_dyn);
    _ = try tm.param(function_id, "mut_writer", writer_dyn_mut);

    try tm.checkTestPass();
}

test "HIR checker rejects deferred dyn interface storage surfaces" {
    var return_tm = try TestModule.init();
    defer return_tm.deinit();
    const return_interface = try return_tm.interfaceDecl("Writer");
    _ = try return_tm.interfaceRequirement(return_interface, "Write", return_tm.module.types.voidType());
    const return_dyn = try return_tm.module.types.addDynInterfaceType(return_interface, false);
    _ = try return_tm.function("GetWriter", return_dyn);
    try return_tm.checkTestFail(.InterfaceRuntimeUnsupported);

    var requirement_tm = try TestModule.init();
    defer requirement_tm.deinit();
    const requirement_interface = try requirement_tm.interfaceDecl("Writer");
    const requirement_dyn = try requirement_tm.module.types.addDynInterfaceType(requirement_interface, false);
    const requirement_id = try requirement_tm.interfaceRequirement(requirement_interface, "Take", requirement_tm.module.types.voidType());
    _ = try requirement_tm.interfaceParam(requirement_id, "writer", requirement_dyn);
    try requirement_tm.checkTestFail(.InterfaceRuntimeUnsupported);

    var field_tm = try TestModule.init();
    defer field_tm.deinit();
    const field_interface = try field_tm.interfaceDecl("Writer");
    _ = try field_tm.interfaceRequirement(field_interface, "Write", field_tm.module.types.voidType());
    const field_dyn = try field_tm.module.types.addDynInterfaceType(field_interface, false);
    const holder = try field_tm.module.hir.addStruct(try field_tm.name("Holder"));
    _ = try field_tm.module.types.addStructType(holder);
    _ = try field_tm.module.hir.addField(holder, try field_tm.name("writer"), field_dyn, synthetic_span);
    try field_tm.checkTestFail(.InterfaceRuntimeUnsupported);

    var local_tm = try TestModule.init();
    defer local_tm.deinit();
    const local_interface = try local_tm.interfaceDecl("Writer");
    _ = try local_tm.interfaceRequirement(local_interface, "Write", local_tm.module.types.voidType());
    const local_dyn = try local_tm.module.types.addDynInterfaceType(local_interface, false);
    const function_id = try local_tm.function("Use", local_tm.module.types.voidType());
    const local_id = try local_tm.local(function_id, "writer", local_dyn);
    const init = try local_tm.int("0");
    const local_stmt = try local_tm.module.hir.addStmt(.{ .local_decl = .{ .local = local_id, .initializer = init } }, synthetic_span);
    local_tm.setBody(function_id, try local_tm.block(&.{local_stmt}));
    try local_tm.checkTestFail(.InterfaceRuntimeUnsupported);
}

test "HIR checker still rejects ordinary interface type signatures" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const interface_id = try tm.interfaceDecl("Writer");
    const writer_type = try tm.module.types.addInterfaceType(interface_id);
    _ = try tm.interfaceRequirement(interface_id, "Write", tm.module.types.voidType());
    const function_id = try tm.function("Use", tm.module.types.intType());
    _ = try tm.param(function_id, "writer", writer_type);

    try tm.checkTestFail(.InterfaceRuntimeUnsupported);
}

test "HIR checker accepts return int main" {
    var tm = try TestModule.init();
    defer tm.deinit();
    _ = try addMainReturnInt(&tm, "0");
    try tm.checkPass();
}

test "HIR checker accepts locals and assignments" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const b = try tm.local(main_id, "b", tm.module.types.boolType());
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("1") } });
    const init_b = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = b, .initializer = try tm.boolLit(true) } });
    const add = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .add, .left = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .right = try tm.int("2") } });
    const assign_x = try addTestStmt(&tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = add } });
    const ret_stmt = try tm.ret(try addTestExpr(&tm.module.hir, .{ .local_ref = x }));
    tm.setBody(main_id, try tm.block(&.{ init_x, init_b, assign_x, ret_stmt }));
    try tm.checkPass();
}

test "HIR checker accepts function calls" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const add_id = try tm.function("add", tm.module.types.intType());
    const a = try tm.param(add_id, "a", tm.module.types.intType());
    const b = try tm.param(add_id, "b", tm.module.types.intType());
    const sum = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .add, .left = try addTestExpr(&tm.module.hir, .{ .param_ref = a }), .right = try addTestExpr(&tm.module.hir, .{ .param_ref = b }) } });
    tm.setBody(add_id, try tm.block(&.{try tm.ret(sum)}));
    const main_id = try tm.function("main", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{ try tm.int("1"), try tm.int("2") });
    tm.setBody(main_id, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .call = .{ .function = add_id, .args = args } }))}));
    try tm.checkPass();
}

test "HIR checker inserts dyn coercion for concrete call argument" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const writer = try tm.interfaceDecl("Writer");
    const write_req = try tm.interfaceRequirement(writer, "Write", tm.module.types.voidType());
    _ = try tm.interfaceParam(write_req, "value", tm.module.types.intType());
    const write_impl = try tm.module.hir.addInterfaceImplMethodFunction(try tm.name("Write"), tm.module.types.voidType(), false, synthetic_span);
    _ = try tm.param(write_impl, "self", tm.module.types.intType());
    _ = try tm.param(write_impl, "value", tm.module.types.intType());
    tm.setBody(write_impl, try tm.block(&.{}));
    const impl_functions = try std.testing.allocator.dupe(hir.FunctionId, &.{write_impl});
    const impl_id = try tm.module.hir.addInterfaceImpl(writer, tm.module.types.intType(), impl_functions, synthetic_span);

    const dyn_writer = try tm.module.types.addDynInterfaceType(writer, true);
    const emit = try tm.function("Emit", tm.module.types.voidType());
    _ = try tm.param(emit, "writer", dyn_writer);
    tm.setBody(emit, try tm.block(&.{}));

    const main = try tm.function("main", tm.module.types.intType());
    const local = try tm.local(main, "writer", tm.module.types.intType());
    const local_decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local, .initializer = try tm.int("1") } });
    const local_expr = try addTestExpr(&tm.module.hir, .{ .local_ref = local });
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{local_expr});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = emit, .args = args } });
    tm.setBody(main, try tm.block(&.{ local_decl, try addTestStmt(&tm.module.hir, .{ .expr_stmt = call }), try tm.ret(try tm.int("0")) }));

    try tm.checkPass();

    const checked_call = tm.module.hir.getExpr(call).kind.call;
    const coerce = tm.module.hir.getExpr(checked_call.args[0]).kind.dyn_coerce;
    try std.testing.expectEqual(local_expr, coerce.source);
    try std.testing.expectEqual(writer, coerce.interface_id);
    try std.testing.expectEqual(impl_id, coerce.impl_id);
    try std.testing.expectEqual(dyn_writer, coerce.result_type);
}

test "HIR checker enforces direct allocation effects for noalloc callers" {
    var ok = try TestModule.init();
    defer ok.deinit();
    const pure = try ok.function("pure", ok.module.types.intType());
    ok.setAllocationEffect(pure, .noalloc);
    ok.setBody(pure, try ok.block(&.{try ok.ret(try ok.int("1"))}));
    const ok_caller = try ok.function("okCaller", ok.module.types.intType());
    ok.setAllocationEffect(ok_caller, .noalloc);
    const ok_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    ok.setBody(ok_caller, try ok.block(&.{try ok.ret(try addTestExpr(&ok.module.hir, .{ .call = .{ .function = pure, .args = ok_args } }))}));
    _ = try addMainReturnInt(&ok, "0");
    try ok.checkPass();

    var bad_alloc = try TestModule.init();
    defer bad_alloc.deinit();
    const may_alloc = try bad_alloc.function("mayAlloc", bad_alloc.module.types.intType());
    bad_alloc.setAllocationEffect(may_alloc, .alloc);
    bad_alloc.setBody(may_alloc, try bad_alloc.block(&.{try bad_alloc.ret(try bad_alloc.int("1"))}));
    const noalloc_caller = try bad_alloc.function("noallocCaller", bad_alloc.module.types.intType());
    bad_alloc.setAllocationEffect(noalloc_caller, .noalloc);
    const alloc_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const alloc_call = try addTestExprAt(&bad_alloc.module.hir, .{ .call = .{ .function = may_alloc, .args = alloc_args } }, testSpan(40));
    bad_alloc.setBody(noalloc_caller, try bad_alloc.block(&.{try bad_alloc.ret(alloc_call)}));
    _ = try addMainReturnInt(&bad_alloc, "0");
    try bad_alloc.checkFailMessageContains(.AllocationEffectMismatch, "allocation effect 'alloc'");
    try std.testing.expectEqual(testSpan(40), try bad_alloc.checkFailSpan(.AllocationEffectMismatch));

    var bad_unspecified = try TestModule.init();
    defer bad_unspecified.deinit();
    const unspecified = try bad_unspecified.function("unspecified", bad_unspecified.module.types.intType());
    bad_unspecified.setBody(unspecified, try bad_unspecified.block(&.{try bad_unspecified.ret(try bad_unspecified.int("1"))}));
    const bad_caller = try bad_unspecified.function("badCaller", bad_unspecified.module.types.intType());
    bad_unspecified.setAllocationEffect(bad_caller, .noalloc);
    const unspecified_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    bad_unspecified.setBody(bad_caller, try bad_unspecified.block(&.{try bad_unspecified.ret(try addTestExpr(&bad_unspecified.module.hir, .{ .call = .{ .function = unspecified, .args = unspecified_args } }))}));
    _ = try addMainReturnInt(&bad_unspecified, "0");
    try bad_unspecified.checkFailMessageContains(.AllocationEffectMismatch, "no noalloc guarantee");
}

test "HIR checker permits alloc and unspecified callers to call any allocation effect" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const pure = try tm.function("pure", tm.module.types.intType());
    tm.setAllocationEffect(pure, .noalloc);
    tm.setBody(pure, try tm.block(&.{try tm.ret(try tm.int("1"))}));
    const may_alloc = try tm.function("mayAlloc", tm.module.types.intType());
    tm.setAllocationEffect(may_alloc, .alloc);
    tm.setBody(may_alloc, try tm.block(&.{try tm.ret(try tm.int("2"))}));
    const unspecified = try tm.function("unspecified", tm.module.types.intType());
    tm.setBody(unspecified, try tm.block(&.{try tm.ret(try tm.int("3"))}));

    const alloc_caller = try tm.function("allocCaller", tm.module.types.intType());
    tm.setAllocationEffect(alloc_caller, .alloc);
    const alloc_expr = try addTestExpr(&tm.module.hir, .{ .binary = .{
        .op = .add,
        .left = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = pure, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } }),
        .right = try addTestExpr(&tm.module.hir, .{ .binary = .{
            .op = .add,
            .left = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = may_alloc, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } }),
            .right = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = unspecified, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } }),
        } }),
    } });
    tm.setBody(alloc_caller, try tm.block(&.{try tm.ret(alloc_expr)}));

    const unspecified_caller = try tm.function("unspecifiedCaller", tm.module.types.intType());
    tm.setBody(unspecified_caller, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .call = .{ .function = may_alloc, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } }))}));
    _ = try addMainReturnInt(&tm, "0");
    try tm.checkPass();
}

test "HIR checker accepts allocation surface pointer params and AllocError value params" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const arena_ptr = try tm.module.types.addPointerType(tm.module.types.arenaType());
    const allocator_ptr = try tm.module.types.addPointerType(tm.module.types.allocatorType());
    const f = try tm.function("usesAllocationSurface", tm.module.types.intType());
    _ = try tm.param(f, "arena", arena_ptr);
    _ = try tm.param(f, "allocator", allocator_ptr);
    _ = try tm.param(f, "err", tm.module.types.allocErrorType());
    tm.setBody(f, try tm.block(&.{try tm.ret(try tm.int("0"))}));

    var bag = DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try checkTestModule(std.testing.allocator, &tm.module, &bag);
    try std.testing.expectEqual(@as(usize, 0), bag.count());
}

test "HIR checker rejects by-value opaque allocation handles" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const f = try tm.function("bad", tm.module.types.intType());
    _ = try tm.param(f, "arena", tm.module.types.arenaType());
    tm.setBody(f, try tm.block(&.{try tm.ret(try tm.int("0"))}));

    var bag = DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, checkTestModule(std.testing.allocator, &tm.module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.OpaqueAllocationTypeByValueUnsupported, bag.diagnostics.items[0].code);
}

test "HIR checker rejects field access on opaque allocation handles" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const f = try tm.function("bad", tm.module.types.intType());
    const arena = try tm.param(f, "arena", tm.module.types.arenaType());
    const arena_ref = try addTestExpr(&tm.module.hir, .{ .param_ref = arena });
    const field = try addTestExpr(&tm.module.hir, .{ .field_access = .{ .receiver = arena_ref, .field_name = try tm.name("someField"), .field_span = synthetic_span } });
    tm.setBody(f, try tm.block(&.{try tm.ret(field)}));

    var bag = DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();
    try std.testing.expectError(error.InvalidSemanticModule, checkTestModule(std.testing.allocator, &tm.module, &bag));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expectEqual(diagnostics.DiagnosticCode.FieldAccessNonStruct, bag.diagnostics.items[0].code);
}

test "HIR checker checks nested noalloc call expressions" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const pure = try tm.function("pure", tm.module.types.intType());
    tm.setAllocationEffect(pure, .noalloc);
    tm.setBody(pure, try tm.block(&.{try tm.ret(try tm.int("1"))}));
    const may_alloc = try tm.function("mayAlloc", tm.module.types.intType());
    tm.setAllocationEffect(may_alloc, .alloc);
    tm.setBody(may_alloc, try tm.block(&.{try tm.ret(try tm.int("2"))}));
    const caller = try tm.function("caller", tm.module.types.intType());
    tm.setAllocationEffect(caller, .noalloc);
    const nested = try addTestExpr(&tm.module.hir, .{ .binary = .{
        .op = .add,
        .left = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = pure, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } }),
        .right = try addTestExprAt(&tm.module.hir, .{ .call = .{ .function = may_alloc, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } }, testSpan(70)),
    } });
    tm.setBody(caller, try tm.block(&.{try tm.ret(nested)}));
    _ = try addMainReturnInt(&tm, "0");
    try std.testing.expectEqual(testSpan(70), try tm.checkFailSpan(.AllocationEffectMismatch));
}

test "HIR checker allows recursive noalloc calls" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const countdown = try tm.function("countDown", tm.module.types.intType());
    tm.setAllocationEffect(countdown, .noalloc);
    _ = try tm.param(countdown, "x", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{try tm.int("0")});
    tm.setBody(countdown, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .call = .{ .function = countdown, .args = args } }))}));
    _ = try addMainReturnInt(&tm, "0");
    try tm.checkPass();
}

test "HIR checker applies allocation effects to comptime and unsafe functions" {
    var comptime_tm = try TestModule.init();
    defer comptime_tm.deinit();
    const may_alloc_const = try comptime_tm.compileTimeFunction("mayAllocConst", comptime_tm.module.types.intType());
    comptime_tm.setAllocationEffect(may_alloc_const, .alloc);
    comptime_tm.setBody(may_alloc_const, try comptime_tm.block(&.{try comptime_tm.ret(try comptime_tm.int("1"))}));
    const bad_const = try comptime_tm.compileTimeFunction("badConst", comptime_tm.module.types.intType());
    comptime_tm.setAllocationEffect(bad_const, .noalloc);
    const const_call = try addTestExpr(&comptime_tm.module.hir, .{ .call = .{ .function = may_alloc_const, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } });
    comptime_tm.setBody(bad_const, try comptime_tm.block(&.{try comptime_tm.ret(const_call)}));
    _ = try addMainReturnInt(&comptime_tm, "0");
    try comptime_tm.checkFail(.AllocationEffectMismatch);

    var unsafe_tm = try TestModule.init();
    defer unsafe_tm.deinit();
    const raw = try unsafe_tm.unsafeFunction("raw", unsafe_tm.module.types.intType());
    unsafe_tm.setAllocationEffect(raw, .noalloc);
    unsafe_tm.setBody(raw, try unsafe_tm.block(&.{try unsafe_tm.ret(try unsafe_tm.int("1"))}));
    const caller = try unsafe_tm.function("caller", unsafe_tm.module.types.intType());
    unsafe_tm.setAllocationEffect(caller, .noalloc);
    const unsafe_call = try addTestExpr(&unsafe_tm.module.hir, .{ .call = .{ .function = raw, .args = try std.testing.allocator.dupe(hir.ExprId, &.{}) } });
    const unsafe_body = try unsafe_tm.block(&.{try unsafe_tm.ret(unsafe_call)});
    unsafe_tm.setBody(caller, try unsafe_tm.block(&.{try addTestStmt(&unsafe_tm.module.hir, .{ .unsafe_block = unsafe_body })}));
    _ = try addMainReturnInt(&unsafe_tm, "0");
    try unsafe_tm.checkPass();
}

test "HIR checker accepts if match while" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("0") } });
    const while_body = try tm.block(&.{try addTestStmt(&tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = try tm.int("1") } })});
    const while_stmt = try addTestStmt(&tm.module.hir, .{ .while_stmt = .{ .condition = try tm.boolLit(false), .body = while_body } });
    const match_body = try addTestStmt(&tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = try tm.int("2") } });
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{.{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = match_body }});
    const match_stmt = try addTestStmt(&tm.module.hir, .{ .match_stmt = .{ .scrutinee = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .arms = arms } });
    const then_block = try tm.block(&.{match_stmt});
    const if_stmt = try addTestStmt(&tm.module.hir, .{ .if_stmt = .{ .condition = try tm.boolLit(true), .then_block = then_block, .else_block = null } });
    const ret_stmt = try tm.ret(try addTestExpr(&tm.module.hir, .{ .local_ref = x }));
    tm.setBody(main_id, try tm.block(&.{ init_x, while_stmt, if_stmt, ret_stmt }));
    try tm.checkPass();
}

test "HIR checker accepts pointer return local copy and call argument" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const int_ptr = try tm.module.types.addPointerType(tm.module.types.intType());

    const identity_id = try tm.function("identity", int_ptr);
    const p = try tm.param(identity_id, "p", int_ptr);
    const q = try tm.local(identity_id, "q", int_ptr);
    const init_q = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = q, .initializer = try addTestExpr(&tm.module.hir, .{ .param_ref = p }) } });
    const ret_q = try tm.ret(try addTestExpr(&tm.module.hir, .{ .local_ref = q }));
    tm.setBody(identity_id, try tm.block(&.{ init_q, ret_q }));

    const use_id = try tm.function("use", tm.module.types.intType());
    const input = try tm.param(use_id, "input", int_ptr);
    const output = try tm.local(use_id, "output", int_ptr);
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{try addTestExpr(&tm.module.hir, .{ .param_ref = input })});
    const init_output = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = output, .initializer = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = identity_id, .args = args } }) } });
    tm.setBody(use_id, try tm.block(&.{ init_output, try tm.ret(try tm.int("0")) }));
    _ = try addMainReturnInt(&tm, "0");

    try tm.checkPass();
}

test "HIR checker accepts address-of local and param" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const int_ptr = try tm.module.types.addPointerType(tm.module.types.intType());
    const helper_id = try tm.function("helper", int_ptr);
    const param_id = try tm.param(helper_id, "p", tm.module.types.intType());
    tm.setBody(helper_id, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .address_of = try addTestExpr(&tm.module.hir, .{ .param_ref = param_id }) }))}));

    const main_id = try tm.function("main", tm.module.types.intType());
    const local_id = try tm.local(main_id, "x", tm.module.types.intType());
    const ptr_local = try tm.local(main_id, "p", int_ptr);
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local_id, .initializer = try tm.int("1") } });
    const addr = try addTestExpr(&tm.module.hir, .{ .address_of = try addTestExpr(&tm.module.hir, .{ .local_ref = local_id }) });
    const init_p = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = ptr_local, .initializer = addr } });
    tm.setBody(main_id, try tm.block(&.{ init_x, init_p, try tm.ret(try tm.int("0")) }));
    try tm.checkPass();
}

test "HIR checker accepts deref inside unsafe contexts" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const int_ptr = try tm.module.types.addPointerType(tm.module.types.intType());
    const read_id = try tm.unsafeFunction("read", tm.module.types.intType());
    const param_id = try tm.param(read_id, "p", int_ptr);
    const deref_param = try addTestExpr(&tm.module.hir, .{ .deref = try addTestExpr(&tm.module.hir, .{ .param_ref = param_id }) });
    tm.setBody(read_id, try tm.block(&.{try tm.ret(deref_param)}));

    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const p_local = try tm.local(main_id, "p", int_ptr);
    const init_x = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("1") } });
    const init_p = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = p_local, .initializer = try addTestExpr(&tm.module.hir, .{ .address_of = try addTestExpr(&tm.module.hir, .{ .local_ref = x }) }) } });
    const deref_local = try addTestExpr(&tm.module.hir, .{ .deref = try addTestExpr(&tm.module.hir, .{ .local_ref = p_local }) });
    const unsafe_body = try tm.block(&.{try tm.ret(deref_local)});
    const unsafe_stmt = try addTestStmt(&tm.module.hir, .{ .unsafe_block = unsafe_body });
    tm.setBody(main_id, try tm.block(&.{ init_x, init_p, unsafe_stmt }));
    try tm.checkPass();
}

test "HIR checker rejects invalid pointer operations" {
    var address_tm = try TestModule.init();
    defer address_tm.deinit();
    const int_ptr = try address_tm.module.types.addPointerType(address_tm.module.types.intType());
    const main_a = try address_tm.function("main", address_tm.module.types.intType());
    const ptr_local = try address_tm.local(main_a, "p", int_ptr);
    const bad_addr = try addTestExpr(&address_tm.module.hir, .{ .address_of = try address_tm.int("1") });
    const init_p = try addTestStmt(&address_tm.module.hir, .{ .local_decl = .{ .local = ptr_local, .initializer = bad_addr } });
    address_tm.setBody(main_a, try address_tm.block(&.{ init_p, try address_tm.ret(try address_tm.int("0")) }));
    try address_tm.checkFail(.AddressOfRequiresPlace);

    var nonptr_tm = try TestModule.init();
    defer nonptr_tm.deinit();
    const main_b = try nonptr_tm.function("main", nonptr_tm.module.types.intType());
    const bad_deref = try addTestExpr(&nonptr_tm.module.hir, .{ .deref = try nonptr_tm.int("1") });
    nonptr_tm.setBody(main_b, try nonptr_tm.block(&.{try nonptr_tm.ret(bad_deref)}));
    try nonptr_tm.checkFail(.DerefRequiresPointer);

    var unsafe_tm = try TestModule.init();
    defer unsafe_tm.deinit();
    const unsafe_ptr = try unsafe_tm.module.types.addPointerType(unsafe_tm.module.types.intType());
    const main_c = try unsafe_tm.function("main", unsafe_tm.module.types.intType());
    const p = try unsafe_tm.local(main_c, "p", unsafe_ptr);
    const bad_unsafe = try addTestExpr(&unsafe_tm.module.hir, .{ .deref = try addTestExpr(&unsafe_tm.module.hir, .{ .local_ref = p }) });
    const init_ptr = try addTestStmt(&unsafe_tm.module.hir, .{ .local_decl = .{ .local = p, .initializer = try addTestExpr(&unsafe_tm.module.hir, .{ .address_of = try addTestExpr(&unsafe_tm.module.hir, .{ .local_ref = try unsafe_tm.local(main_c, "x", unsafe_tm.module.types.intType()) }) }) } });
    unsafe_tm.setBody(main_c, try unsafe_tm.block(&.{ init_ptr, try unsafe_tm.ret(bad_unsafe) }));
    try unsafe_tm.checkFail(.DerefRequiresUnsafe);
}

test "HIR checker rejects pointer type mismatches" {
    var return_mismatch = try TestModule.init();
    defer return_mismatch.deinit();
    const int_ptr = try return_mismatch.module.types.addPointerType(return_mismatch.module.types.intType());
    const bool_ptr = try return_mismatch.module.types.addPointerType(return_mismatch.module.types.boolType());
    const bad_return = try return_mismatch.function("bad", int_ptr);
    const bp = try return_mismatch.param(bad_return, "bp", bool_ptr);
    return_mismatch.setBody(bad_return, try return_mismatch.block(&.{try return_mismatch.ret(try addTestExpr(&return_mismatch.module.hir, .{ .param_ref = bp }))}));
    _ = try addMainReturnInt(&return_mismatch, "0");
    try return_mismatch.checkFail(.TypeMismatch);

    var local_mismatch = try TestModule.init();
    defer local_mismatch.deinit();
    const local_int_ptr = try local_mismatch.module.types.addPointerType(local_mismatch.module.types.intType());
    const local_bool_ptr = try local_mismatch.module.types.addPointerType(local_mismatch.module.types.boolType());
    const local_helper = try local_mismatch.function("localMismatch", local_mismatch.module.types.intType());
    const source = try local_mismatch.param(local_helper, "source", local_bool_ptr);
    const target = try local_mismatch.local(local_helper, "target", local_int_ptr);
    const bad_init = try addTestStmt(&local_mismatch.module.hir, .{ .local_decl = .{ .local = target, .initializer = try addTestExpr(&local_mismatch.module.hir, .{ .param_ref = source }) } });
    local_mismatch.setBody(local_helper, try local_mismatch.block(&.{ bad_init, try local_mismatch.ret(try local_mismatch.int("0")) }));
    _ = try addMainReturnInt(&local_mismatch, "0");
    try local_mismatch.checkFail(.TypeMismatch);

    var call_mismatch = try TestModule.init();
    defer call_mismatch.deinit();
    const call_int_ptr = try call_mismatch.module.types.addPointerType(call_mismatch.module.types.intType());
    const call_bool_ptr = try call_mismatch.module.types.addPointerType(call_mismatch.module.types.boolType());
    const callee = try call_mismatch.function("takesIntPtr", call_mismatch.module.types.intType());
    _ = try call_mismatch.param(callee, "p", call_int_ptr);
    call_mismatch.setBody(callee, try call_mismatch.block(&.{try call_mismatch.ret(try call_mismatch.int("1"))}));
    const caller = try call_mismatch.function("callMismatch", call_mismatch.module.types.intType());
    const bool_param = try call_mismatch.param(caller, "bp", call_bool_ptr);
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{try addTestExpr(&call_mismatch.module.hir, .{ .param_ref = bool_param })});
    call_mismatch.setBody(caller, try call_mismatch.block(&.{try call_mismatch.ret(try addTestExpr(&call_mismatch.module.hir, .{ .call = .{ .function = callee, .args = call_args } }))}));
    _ = try addMainReturnInt(&call_mismatch, "0");
    try call_mismatch.checkFail(.InvalidCall);
}

test "HIR checker rejects missing main" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const helper = try tm.function("helper", tm.module.types.intType());
    tm.setBody(helper, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    try tm.checkFail(.MissingMain);
}

test "HIR checker rejects bad main return type and params" {
    var bad_return = try TestModule.init();
    defer bad_return.deinit();
    const main_bool = try bad_return.function("main", bad_return.module.types.boolType());
    bad_return.setBody(main_bool, try bad_return.block(&.{try bad_return.ret(try bad_return.boolLit(true))}));
    try bad_return.checkFail(.InvalidMainSignature);

    var bad_params = try TestModule.init();
    defer bad_params.deinit();
    const main_param = try bad_params.function("main", bad_params.module.types.intType());
    _ = try bad_params.param(main_param, "argc", bad_params.module.types.intType());
    bad_params.setBody(main_param, try bad_params.block(&.{try bad_params.ret(try bad_params.int("0"))}));
    try bad_params.checkFail(.InvalidMainSignature);
}

test "HIR checker rejects return local and assignment mismatches" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    tm.setBody(main_id, try tm.block(&.{try tm.ret(try tm.boolLit(true))}));
    try tm.checkFail(.TypeMismatch);

    var local_tm = try TestModule.init();
    defer local_tm.deinit();
    const local_main = try local_tm.function("main", local_tm.module.types.intType());
    const x = try local_tm.local(local_main, "x", local_tm.module.types.intType());
    local_tm.setBody(local_main, try local_tm.block(&.{try addTestStmt(&local_tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try local_tm.boolLit(false) } })}));
    try local_tm.checkFail(.TypeMismatch);

    var assign_tm = try TestModule.init();
    defer assign_tm.deinit();
    const assign_main = try assign_tm.function("main", assign_tm.module.types.intType());
    const y = try assign_tm.local(assign_main, "y", assign_tm.module.types.intType());
    const decl_y = try addTestStmt(&assign_tm.module.hir, .{ .local_decl = .{ .local = y, .initializer = try assign_tm.int("0") } });
    const assign_y = try addTestStmt(&assign_tm.module.hir, .{ .assignment = .{ .target = .{ .local = y }, .value = try assign_tm.boolLit(true) } });
    assign_tm.setBody(assign_main, try assign_tm.block(&.{ decl_y, assign_y }));
    try assign_tm.checkFail(.TypeMismatch);
}

test "HIR checker rejects bad conditions calls and match patterns" {
    var if_tm = try TestModule.init();
    defer if_tm.deinit();
    const if_main = try if_tm.function("main", if_tm.module.types.intType());
    if_tm.setBody(if_main, try if_tm.block(&.{try addTestStmt(&if_tm.module.hir, .{ .if_stmt = .{ .condition = try if_tm.int("1"), .then_block = try if_tm.block(&.{}), .else_block = null } })}));
    try if_tm.checkFail(.InvalidConditionType);

    var while_tm = try TestModule.init();
    defer while_tm.deinit();
    const while_main = try while_tm.function("main", while_tm.module.types.intType());
    while_tm.setBody(while_main, try while_tm.block(&.{try addTestStmt(&while_tm.module.hir, .{ .while_stmt = .{ .condition = try while_tm.int("1"), .body = try while_tm.block(&.{}) } })}));
    try while_tm.checkFail(.InvalidConditionType);

    var call_count = try TestModule.init();
    defer call_count.deinit();
    const callee = try call_count.function("callee", call_count.module.types.intType());
    _ = try call_count.param(callee, "x", call_count.module.types.intType());
    call_count.setBody(callee, try call_count.block(&.{try call_count.ret(try call_count.int("0"))}));
    const call_main = try call_count.function("main", call_count.module.types.intType());
    const no_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    call_count.setBody(call_main, try call_count.block(&.{try call_count.ret(try addTestExpr(&call_count.module.hir, .{ .call = .{ .function = callee, .args = no_args } }))}));
    try call_count.checkFail(.InvalidCall);

    var call_type = try TestModule.init();
    defer call_type.deinit();
    const id = try call_type.function("id", call_type.module.types.intType());
    _ = try call_type.param(id, "x", call_type.module.types.intType());
    call_type.setBody(id, try call_type.block(&.{try call_type.ret(try call_type.int("0"))}));
    const main = try call_type.function("main", call_type.module.types.intType());
    const bad_args = try std.testing.allocator.dupe(hir.ExprId, &.{try call_type.boolLit(false)});
    call_type.setBody(main, try call_type.block(&.{try call_type.ret(try addTestExpr(&call_type.module.hir, .{ .call = .{ .function = id, .args = bad_args } }))}));
    try call_type.checkFail(.InvalidCall);

    var match_tm = try TestModule.init();
    defer match_tm.deinit();
    const match_main = try match_tm.function("main", match_tm.module.types.intType());
    const body = try match_tm.block(&.{});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{.{ .pattern = .{ .bool_literal = true }, .pattern_span = synthetic_span, .body = body }});
    match_tm.setBody(match_main, try match_tm.block(&.{try addTestStmt(&match_tm.module.hir, .{ .match_stmt = .{ .scrutinee = try match_tm.int("0"), .arms = arms } })}));
    try match_tm.checkFail(.TypeMismatch);
}

test "HIR checker rejects duplicate match pattern and wildcard" {
    var dup_lit = try TestModule.init();
    defer dup_lit.deinit();
    const lit_main = try dup_lit.function("main", dup_lit.module.types.intType());
    const body = try dup_lit.block(&.{});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{ .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = body }, .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, .pattern_span = synthetic_span, .body = body } });
    dup_lit.setBody(lit_main, try dup_lit.block(&.{try addTestStmt(&dup_lit.module.hir, .{ .match_stmt = .{ .scrutinee = try dup_lit.int("0"), .arms = arms } })}));
    try dup_lit.checkFail(.TypeMismatch);

    var dup_wild = try TestModule.init();
    defer dup_wild.deinit();
    const wild_main = try dup_wild.function("main", dup_wild.module.types.intType());
    const wild_body = try dup_wild.block(&.{});
    const wild_arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{ .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = wild_body }, .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = wild_body } });
    dup_wild.setBody(wild_main, try dup_wild.block(&.{try addTestStmt(&dup_wild.module.hir, .{ .match_stmt = .{ .scrutinee = try dup_wild.boolLit(false), .arms = wild_arms } })}));
    try dup_wild.checkFail(.TypeMismatch);
}

test "HIR checker diagnostics use HIR source spans" {
    var if_tm = try TestModule.init();
    defer if_tm.deinit();
    const if_main = try if_tm.function("main", if_tm.module.types.intType());
    const if_cond = try addTestExprAt(&if_tm.module.hir, .{ .int_literal = try std.testing.allocator.dupe(u8, "1") }, testSpan(11));
    if_tm.setBody(if_main, try if_tm.block(&.{try addTestStmtAt(&if_tm.module.hir, .{ .if_stmt = .{ .condition = if_cond, .then_block = try if_tm.block(&.{}), .else_block = null } }, testSpan(10))}));
    try std.testing.expectEqual(testSpan(11), try if_tm.checkFailSpan(.InvalidConditionType));

    var assign_tm = try TestModule.init();
    defer assign_tm.deinit();
    const assign_main = try assign_tm.function("main", assign_tm.module.types.intType());
    const x = try assign_tm.local(assign_main, "x", assign_tm.module.types.intType());
    const decl_x = try addTestStmt(&assign_tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try assign_tm.int("0") } });
    const bad_value = try addTestExprAt(&assign_tm.module.hir, .{ .bool_literal = true }, testSpan(22));
    const assign_x = try addTestStmtAt(&assign_tm.module.hir, .{ .assignment = .{ .target = .{ .local = x }, .value = bad_value } }, testSpan(20));
    assign_tm.setBody(assign_main, try assign_tm.block(&.{ decl_x, assign_x }));
    try std.testing.expectEqual(testSpan(22), try assign_tm.checkFailSpan(.TypeMismatch));

    var ret_tm = try TestModule.init();
    defer ret_tm.deinit();
    const ret_main = try ret_tm.function("main", ret_tm.module.types.intType());
    const bad_ret = try addTestExprAt(&ret_tm.module.hir, .{ .bool_literal = false }, testSpan(33));
    ret_tm.setBody(ret_main, try ret_tm.block(&.{try addTestStmtAt(&ret_tm.module.hir, .{ .return_stmt = bad_ret }, testSpan(30))}));
    try std.testing.expectEqual(testSpan(33), try ret_tm.checkFailSpan(.TypeMismatch));

    var call_tm = try TestModule.init();
    defer call_tm.deinit();
    const callee = try call_tm.function("id", call_tm.module.types.intType());
    _ = try call_tm.param(callee, "x", call_tm.module.types.intType());
    call_tm.setBody(callee, try call_tm.block(&.{try call_tm.ret(try call_tm.int("0"))}));
    const call_main = try call_tm.function("main", call_tm.module.types.intType());
    const bad_arg = try addTestExprAt(&call_tm.module.hir, .{ .bool_literal = false }, testSpan(44));
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{bad_arg});
    const bad_call = try addTestExprAt(&call_tm.module.hir, .{ .call = .{ .function = callee, .args = args } }, testSpan(40));
    call_tm.setBody(call_main, try call_tm.block(&.{try call_tm.ret(bad_call)}));
    try std.testing.expectEqual(testSpan(44), try call_tm.checkFailSpan(.InvalidCall));

    var match_tm = try TestModule.init();
    defer match_tm.deinit();
    const match_main = try match_tm.function("main", match_tm.module.types.intType());
    const arm_body = try match_tm.block(&.{});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{.{ .pattern = .{ .bool_literal = true }, .pattern_span = testSpan(55), .body = arm_body }});
    match_tm.setBody(match_main, try match_tm.block(&.{try addTestStmtAt(&match_tm.module.hir, .{ .match_stmt = .{ .scrutinee = try match_tm.int("0"), .arms = arms } }, testSpan(50))}));
    try std.testing.expectEqual(testSpan(55), try match_tm.checkFailSpan(.TypeMismatch));
}

test "HIR checker accepts Phase 2-style lowered program" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const x = try tm.local(main_id, "x", tm.module.types.intType());
    const decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = x, .initializer = try tm.int("4") } });
    const then_body = try tm.block(&.{try tm.ret(try tm.int("9"))});
    const arms = try std.testing.allocator.dupe(hir.HirMatchArm, &.{ .{ .pattern = .{ .int_literal = try std.testing.allocator.dupe(u8, "4") }, .pattern_span = synthetic_span, .body = then_body }, .{ .pattern = .wildcard, .pattern_span = synthetic_span, .body = try tm.block(&.{try tm.ret(try tm.int("1"))}) } });
    const match_stmt = try addTestStmt(&tm.module.hir, .{ .match_stmt = .{ .scrutinee = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .arms = arms } });
    const condition = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .greater_equal, .left = try addTestExpr(&tm.module.hir, .{ .local_ref = x }), .right = try tm.int("4") } });
    const if_stmt = try addTestStmt(&tm.module.hir, .{ .if_stmt = .{ .condition = condition, .then_block = try tm.block(&.{match_stmt}), .else_block = try tm.block(&.{try tm.ret(try tm.int("0"))}) } });
    tm.setBody(main_id, try tm.block(&.{ decl, if_stmt }));
    try tm.checkPass();
}

test "HIR checker accepts discard and explicit uses of must_use enum values" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const enum_id = try tm.module.hir.addEnum(try tm.name("ParseResult"), true);
    const enum_type = try tm.module.types.addEnumType(enum_id);
    const ok_variant = try tm.module.hir.addVariant(enum_id, try tm.name("Ok"), synthetic_span);
    _ = try tm.module.hir.addEnumPayloadField(ok_variant, try tm.name("value"), tm.module.types.intType(), synthetic_span);

    const make = try tm.function("make", enum_type);
    _ = try tm.module.hir.addParam(make, try tm.name("value"), tm.module.types.intType(), synthetic_span);
    const make_return_arg = try tm.int("1");
    const make_args = try std.testing.allocator.dupe(hir.ExprId, &.{make_return_arg});
    const make_value = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok_variant, .args = make_args } });
    const make_return = try tm.ret(make_value);
    tm.setBody(make, try tm.block(&.{make_return}));

    const main = try tm.function("main", tm.module.types.intType());
    const constructor_arg = try tm.int("2");
    const constructor_args = try std.testing.allocator.dupe(hir.ExprId, &.{constructor_arg});
    const constructor = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok_variant, .args = constructor_args } });
    const local = try tm.local(main, "result", enum_type);
    const local_decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local, .initializer = constructor } });
    const call_arg = try tm.int("3");
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{call_arg});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = make, .args = call_args } });
    const discard_call = try addTestStmt(&tm.module.hir, .{ .discard_stmt = call });
    const non_must_use = try tm.int("4");
    const discard_int = try addTestStmt(&tm.module.hir, .{ .discard_stmt = non_must_use });
    const ret = try tm.ret(try tm.int("0"));
    tm.setBody(main, try tm.block(&.{ local_decl, discard_call, discard_int, ret }));

    try tm.checkPass();
}

test "HIR checker rejects ignored must_use expression statement" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const enum_id = try tm.module.hir.addEnum(try tm.name("ParseResult"), true);
    _ = try tm.module.types.addEnumType(enum_id);
    const ok_variant = try tm.module.hir.addVariant(enum_id, try tm.name("Ok"), synthetic_span);

    const main = try tm.function("main", tm.module.types.intType());
    const empty_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const constructor = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = enum_id, .variant_id = ok_variant, .args = empty_args } });
    const ignored = try addTestStmt(&tm.module.hir, .{ .expr_stmt = constructor });
    const ret = try tm.ret(try tm.int("0"));
    tm.setBody(main, try tm.block(&.{ ignored, ret }));

    try tm.checkFail(.IgnoredMustUseValue);
}

fn addResultEnum(tm: *TestModule, name: []const u8, must_use: bool) !struct { enum_id: hir.EnumId, type_id: types.TypeId, ok: hir.VariantId, err: hir.VariantId, ok_field: hir.EnumPayloadFieldId, err_field: hir.EnumPayloadFieldId } {
    const enum_id = try tm.module.hir.addEnum(try tm.name(name), must_use);
    const type_id = try tm.module.types.addEnumType(enum_id);
    const ok = try tm.module.hir.addVariant(enum_id, try tm.name("Ok"), synthetic_span);
    const ok_field = try tm.module.hir.addEnumPayloadField(ok, try tm.name("value"), tm.module.types.intType(), synthetic_span);
    const err = try tm.module.hir.addVariant(enum_id, try tm.name("Err"), synthetic_span);
    const err_field = try tm.module.hir.addEnumPayloadField(err, try tm.name("code"), tm.module.types.intType(), synthetic_span);
    tm.module.hir.setEnumResultShape(enum_id, .{ .ok_variant = ok, .err_variant = err, .ok_payload = ok_field, .err_payload = err_field, .ok_type = tm.module.types.intType(), .err_type = tm.module.types.intType() });
    return .{ .enum_id = enum_id, .type_id = type_id, .ok = ok, .err = err, .ok_field = ok_field, .err_field = err_field };
}

test "HIR checker accepts try over same Result-shaped enum" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    const parse = try tm.function("parse", result.type_id);
    const arg_expr = try tm.int("1");
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{arg_expr});
    const ok_expr = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = args } });
    tm.setBody(parse, try tm.block(&.{try tm.ret(ok_expr)}));
    const add = try tm.function("add", result.type_id);
    const call_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = parse, .args = call_args } });
    const tried = try addTestExpr(&tm.module.hir, .{ .try_expr = call });
    const local = try tm.local(add, "value", tm.module.types.intType());
    const decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local, .initializer = tried } });
    const local_ref = try addTestExpr(&tm.module.hir, .{ .local_ref = local });
    const ret_args = try std.testing.allocator.dupe(hir.ExprId, &.{local_ref});
    const ret_ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = ret_args } });
    tm.setBody(add, try tm.block(&.{ decl, try tm.ret(ret_ok) }));
    try tm.checkPass();
}

test "HIR checker rejects try operand that is not Result-shaped" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    const f = try tm.function("bad", result.type_id);
    tm.setBody(f, try tm.block(&.{try tm.ret(try addTestExpr(&tm.module.hir, .{ .try_expr = try tm.int("1") }))}));
    try tm.checkFail(.TryOperandNotResult);
}

test "HIR checker rejects try in non-Result-returning function" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    const ok_arg = try tm.int("1");
    const ok_args = try std.testing.allocator.dupe(hir.ExprId, &.{ok_arg});
    const ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = ok_args } });
    const tried = try addTestExpr(&tm.module.hir, .{ .try_expr = ok });
    tm.setBody(main, try tm.block(&.{ try addTestStmt(&tm.module.hir, .{ .discard_stmt = tried }), try tm.ret(try tm.int("0")) }));
    try tm.checkFail(.TryOutsideResultFunction);
}

test "HIR checker rejects mismatched try Result type" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const result = try addResultEnum(&tm, "ParseResult", true);
    const other = try addResultEnum(&tm, "OtherResult", true);
    const main = try tm.function("main", tm.module.types.intType());
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));
    const f = try tm.function("bad", other.type_id);
    const ok_arg = try tm.int("1");
    const ok_args = try std.testing.allocator.dupe(hir.ExprId, &.{ok_arg});
    const ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = result.enum_id, .variant_id = result.ok, .args = ok_args } });
    const tried = try addTestExpr(&tm.module.hir, .{ .try_expr = ok });
    const ret_arg = try tm.int("0");
    const ret_args = try std.testing.allocator.dupe(hir.ExprId, &.{ret_arg});
    const ret_ok = try addTestExpr(&tm.module.hir, .{ .enum_constructor = .{ .enum_id = other.enum_id, .variant_id = other.ok, .args = ret_args } });
    tm.setBody(f, try tm.block(&.{ try addTestStmt(&tm.module.hir, .{ .discard_stmt = tried }), try tm.ret(ret_ok) }));
    try tm.checkFail(.TryResultTypeMismatch);
}

test "HIR checker evaluates compile-time int and bool expressions" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const main = try tm.function("main", tm.module.types.intType());
    const add = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .add, .left = try tm.int("40"), .right = try tm.int("2") } });
    const compile_int = try addTestExpr(&tm.module.hir, .{ .compile_time = .{ .operand = add, .span = synthetic_span } });
    const comparison = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .less, .left = try tm.int("1"), .right = try tm.int("2") } });
    const compile_bool = try addTestExpr(&tm.module.hir, .{ .compile_time = .{ .operand = comparison, .span = synthetic_span } });
    const ok = try tm.local(main, "ok", tm.module.types.boolType());
    const ok_decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = ok, .initializer = compile_bool } });
    tm.setBody(main, try tm.block(&.{ ok_decl, try tm.ret(compile_int) }));

    try tm.checkPass();
    try std.testing.expect(tm.module.compile_time_values.get(compile_int).?.eql(.{ .int = 42 }));
    try std.testing.expect(tm.module.compile_time_values.get(compile_bool).?.eql(.{ .bool = true }));
}

test "HIR checker reports unsupported compile-time expression" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const main = try tm.function("main", tm.module.types.intType());
    const local = try tm.local(main, "x", tm.module.types.intType());
    const local_decl = try addTestStmt(&tm.module.hir, .{ .local_decl = .{ .local = local, .initializer = try tm.int("1") } });
    const local_ref = try addTestExpr(&tm.module.hir, .{ .local_ref = local });
    const compile_local = try addTestExpr(&tm.module.hir, .{ .compile_time = .{ .operand = local_ref, .span = synthetic_span } });
    tm.setBody(main, try tm.block(&.{ local_decl, try tm.ret(compile_local) }));

    try tm.checkFail(.CompileTimeUnboundLocal);
}

test "HIR checker evaluates static assertions" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const main = try tm.function("main", tm.module.types.intType());
    const comparison = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .equal_equal, .left = try tm.int("2"), .right = try tm.int("2") } });
    try tm.module.hir.addStaticAssert(comparison, synthetic_span);
    tm.setBody(main, try tm.block(&.{try tm.ret(try tm.int("0"))}));

    try tm.checkPass();
}

test "HIR checker rejects failing and non-bool static assertions" {
    var false_tm = try TestModule.init();
    defer false_tm.deinit();
    const false_main = try false_tm.function("main", false_tm.module.types.intType());
    try false_tm.module.hir.addStaticAssert(try false_tm.boolLit(false), synthetic_span);
    false_tm.setBody(false_main, try false_tm.block(&.{try false_tm.ret(try false_tm.int("0"))}));
    try false_tm.checkFail(.StaticAssertFailed);

    var int_tm = try TestModule.init();
    defer int_tm.deinit();
    const int_main = try int_tm.function("main", int_tm.module.types.intType());
    try int_tm.module.hir.addStaticAssert(try int_tm.int("42"), synthetic_span);
    int_tm.setBody(int_main, try int_tm.block(&.{try int_tm.ret(try int_tm.int("0"))}));
    try int_tm.checkFail(.StaticAssertRequiresBool);
}

test "HIR checker reports unsupported static assertion expression before main validation" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const helper = try tm.function("helper", tm.module.types.intType());
    tm.setBody(helper, try tm.block(&.{try tm.ret(try tm.int("1"))}));
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = helper, .args = args } });
    const comparison = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .equal_equal, .left = call, .right = try tm.int("1") } });
    try tm.module.hir.addStaticAssert(comparison, synthetic_span);

    try tm.checkFail(.CompileTimeFunctionRequired);
}

test "HIR checker evaluates compile-time function calls with arguments" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const add = try tm.compileTimeFunction("add", tm.module.types.intType());
    const a = try tm.param(add, "a", tm.module.types.intType());
    const b = try tm.param(add, "b", tm.module.types.intType());
    const a_ref = try addTestExpr(&tm.module.hir, .{ .param_ref = a });
    const b_ref = try addTestExpr(&tm.module.hir, .{ .param_ref = b });
    const sum = try addTestExpr(&tm.module.hir, .{ .binary = .{ .op = .add, .left = a_ref, .right = b_ref } });
    tm.setBody(add, try tm.block(&.{try tm.ret(sum)}));

    const main = try tm.function("main", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{ try tm.int("20"), try tm.int("22") });
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = add, .args = args } });
    const compile_call = try addTestExpr(&tm.module.hir, .{ .compile_time = .{ .operand = call, .span = synthetic_span } });
    tm.setBody(main, try tm.block(&.{try tm.ret(compile_call)}));

    try tm.checkPass();
    try std.testing.expect(tm.module.compile_time_values.get(compile_call).?.eql(.{ .int = 42 }));
}

test "HIR checker rejects runtime calls to compile-time functions" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const answer = try tm.compileTimeFunction("answer", tm.module.types.intType());
    tm.setBody(answer, try tm.block(&.{try tm.ret(try tm.int("42"))}));

    const main = try tm.function("main", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = answer, .args = args } });
    tm.setBody(main, try tm.block(&.{try tm.ret(call)}));

    try tm.checkFail(.CompileTimeFunctionRequired);
}

test "HIR checker rejects non-compile-time function calls from compile-time contexts" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const helper = try tm.function("helper", tm.module.types.intType());
    tm.setBody(helper, try tm.block(&.{try tm.ret(try tm.int("1"))}));

    const main = try tm.function("main", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = helper, .args = args } });
    const compile_call = try addTestExpr(&tm.module.hir, .{ .compile_time = .{ .operand = call, .span = synthetic_span } });
    tm.setBody(main, try tm.block(&.{try tm.ret(compile_call)}));

    try tm.checkFail(.CompileTimeFunctionRequired);
}

test "HIR checker rejects unsupported compile-time function body statement" {
    var tm = try TestModule.init();
    defer tm.deinit();

    const bad = try tm.compileTimeFunction("bad", tm.module.types.intType());
    const expr_stmt = try addTestStmt(&tm.module.hir, .{ .expr_stmt = try tm.int("0") });
    tm.setBody(bad, try tm.block(&.{ expr_stmt, try tm.ret(try tm.int("1")) }));

    const main = try tm.function("main", tm.module.types.intType());
    const args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&tm.module.hir, .{ .call = .{ .function = bad, .args = args } });
    const compile_call = try addTestExpr(&tm.module.hir, .{ .compile_time = .{ .operand = call, .span = synthetic_span } });
    tm.setBody(main, try tm.block(&.{try tm.ret(compile_call)}));

    try tm.checkFail(.CompileTimeUnsupportedStatement);
}

test "HIR checker reports compile-time division by zero and overflow" {
    var div_tm = try TestModule.init();
    defer div_tm.deinit();
    const div_main = try div_tm.function("main", div_tm.module.types.intType());
    const div = try addTestExpr(&div_tm.module.hir, .{ .binary = .{ .op = .divide, .left = try div_tm.int("1"), .right = try div_tm.int("0") } });
    div_tm.setBody(div_main, try div_tm.block(&.{try div_tm.ret(try addTestExpr(&div_tm.module.hir, .{ .compile_time = .{ .operand = div, .span = synthetic_span } }))}));
    try div_tm.checkFail(.CompileTimeDivisionByZero);

    var overflow_tm = try TestModule.init();
    defer overflow_tm.deinit();
    const overflow_main = try overflow_tm.function("main", overflow_tm.module.types.intType());
    const overflow = try addTestExpr(&overflow_tm.module.hir, .{ .binary = .{ .op = .add, .left = try overflow_tm.int("9223372036854775807"), .right = try overflow_tm.int("1") } });
    overflow_tm.setBody(overflow_main, try overflow_tm.block(&.{try overflow_tm.ret(try addTestExpr(&overflow_tm.module.hir, .{ .compile_time = .{ .operand = overflow, .span = synthetic_span } }))}));
    try overflow_tm.checkFail(.CompileTimeOverflow);
}

test "HIR checker accepts unsafe blocks" {
    var tm = try TestModule.init();
    defer tm.deinit();
    const main_id = try tm.function("main", tm.module.types.intType());
    const body = try tm.block(&.{try tm.ret(try tm.int("0"))});
    const unsafe_stmt = try addTestStmt(&tm.module.hir, .{ .unsafe_block = body });
    tm.setBody(main_id, try tm.block(&.{unsafe_stmt}));
    try tm.checkPass();
}

test "HIR checker enforces unsafe function calls" {
    var outside = try TestModule.init();
    defer outside.deinit();
    const helper = try outside.unsafeFunction("helper", outside.module.types.intType());
    outside.setBody(helper, try outside.block(&.{try outside.ret(try outside.int("1"))}));
    const outside_main = try outside.function("main", outside.module.types.intType());
    const outside_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    outside.setBody(outside_main, try outside.block(&.{try outside.ret(try addTestExpr(&outside.module.hir, .{ .call = .{ .function = helper, .args = outside_args } }))}));
    try outside.checkFail(.UnsafeCallRequiresUnsafe);

    var inside = try TestModule.init();
    defer inside.deinit();
    const inside_helper = try inside.unsafeFunction("helper", inside.module.types.intType());
    inside.setBody(inside_helper, try inside.block(&.{try inside.ret(try inside.int("1"))}));
    const inside_main = try inside.function("main", inside.module.types.intType());
    const inside_args = try std.testing.allocator.dupe(hir.ExprId, &.{});
    const call = try addTestExpr(&inside.module.hir, .{ .call = .{ .function = inside_helper, .args = inside_args } });
    const unsafe_body = try inside.block(&.{try inside.ret(call)});
    inside.setBody(inside_main, try inside.block(&.{try addTestStmt(&inside.module.hir, .{ .unsafe_block = unsafe_body })}));
    try inside.checkPass();
}
