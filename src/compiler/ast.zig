const std = @import("std");
const source_model = @import("source.zig");

pub const SourceSpan = source_model.SourceSpan;

pub const NameSegment = struct {
    text: []const u8,
    span: SourceSpan,
};

pub const QualifiedName = struct {
    parts: []NameSegment,
    span: SourceSpan,

    pub fn deinit(self: QualifiedName, allocator: std.mem.Allocator) void {
        allocator.free(self.parts);
    }

    pub fn write(self: QualifiedName, writer: anytype) !void {
        for (self.parts, 0..) |part, index| {
            if (index != 0) try writer.writeByte('.');
            try writer.writeAll(part.text);
        }
    }
};

pub const ModuleDecl = struct {
    name: QualifiedName,
    span: SourceSpan,

    pub fn deinit(self: ModuleDecl, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
    }
};

pub const ImportDecl = struct {
    name: QualifiedName,
    span: SourceSpan,

    pub fn deinit(self: ImportDecl, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
    }
};

pub const AttributeArg = union(enum) {
    identifier: NameSegment,
    int_literal: struct { text: []const u8, span: SourceSpan },
    bool_literal: struct { value: bool, span: SourceSpan },
    string_literal: struct { text: []const u8, span: SourceSpan },

    pub fn span(self: AttributeArg) SourceSpan {
        return switch (self) {
            .identifier => |arg| arg.span,
            .int_literal => |arg| arg.span,
            .bool_literal => |arg| arg.span,
            .string_literal => |arg| arg.span,
        };
    }

    pub fn writeDebug(self: AttributeArg, writer: anytype) !void {
        switch (self) {
            .identifier => |arg| try writer.writeAll(arg.text),
            .int_literal => |arg| try writer.writeAll(arg.text),
            .bool_literal => |arg| try writer.writeAll(if (arg.value) "true" else "false"),
            .string_literal => |arg| try writer.writeAll(arg.text),
        }
    }
};

pub const AttributeArguments = struct {
    args: []AttributeArg,
    span: SourceSpan,

    pub fn deinit(self: AttributeArguments, allocator: std.mem.Allocator) void {
        allocator.free(self.args);
    }
};

pub const Attribute = struct {
    name: QualifiedName,
    arguments: ?AttributeArguments = null,
    span: SourceSpan,

    pub fn deinit(self: Attribute, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
        if (self.arguments) |arguments| arguments.deinit(allocator);
    }

    pub fn writeDebug(self: Attribute, writer: anytype) !void {
        try writer.writeAll("  Attribute ");
        try self.name.write(writer);
        if (self.arguments) |arguments| {
            try writer.writeByte('(');
            for (arguments.args, 0..) |argument, index| {
                if (index != 0) try writer.writeAll(", ");
                try argument.writeDebug(writer);
            }
            try writer.writeByte(')');
        }
        try writer.writeByte('\n');
    }
};

pub const ArraySuffix = struct {
    length_text: []const u8,
    length_span: SourceSpan,
    owns_length_text: bool = false,
    span: SourceSpan,
};

pub const TypeName = struct {
    name: QualifiedName,
    generic_args: []TypeName = &.{},
    is_mut: bool = false,
    is_dyn: bool = false,
    dyn_span: SourceSpan = .{ .start = 0, .length = 0 },
    is_reference: bool = false,
    is_pointer: bool = false,
    array_suffixes: []ArraySuffix = &.{},
    span: SourceSpan,

    pub fn deinit(self: TypeName, allocator: std.mem.Allocator) void {
        self.name.deinit(allocator);
        for (self.generic_args) |generic_arg| {
            generic_arg.deinit(allocator);
        }
        allocator.free(self.generic_args);
        for (self.array_suffixes) |suffix| {
            if (suffix.owns_length_text) allocator.free(suffix.length_text);
        }
        allocator.free(self.array_suffixes);
    }

    pub fn write(self: TypeName, writer: anytype) !void {
        if (self.is_mut) try writer.writeAll("mut ");
        if (self.is_dyn) try writer.writeAll("dyn ");
        try self.name.write(writer);
        if (self.generic_args.len != 0) {
            try writer.writeByte('<');
            for (self.generic_args, 0..) |generic_arg, index| {
                if (index != 0) try writer.writeAll(", ");
                try generic_arg.write(writer);
            }
            try writer.writeByte('>');
        }
        if (self.is_reference) try writer.writeByte('&');
        if (self.is_pointer) try writer.writeByte('*');
        for (self.array_suffixes) |suffix| {
            try writer.writeByte('[');
            try writer.writeAll(suffix.length_text);
            try writer.writeByte(']');
        }
    }
};

pub const GenericParam = NameSegment;

pub const SignatureName = struct {
    base: NameSegment,
    operator_suffix: ?NameSegment = null,
    span: SourceSpan,

    pub fn write(self: SignatureName, writer: anytype) !void {
        try writer.writeAll(self.base.text);
        if (self.operator_suffix) |suffix| try writer.writeAll(suffix.text);
    }
};

pub const ParamDecl = struct {
    type_name: TypeName,
    name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: ParamDecl, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
    }
};

pub const SignatureDecl = struct {
    return_type: TypeName,
    name: SignatureName,
    params: []ParamDecl,
    span: SourceSpan,

    pub fn deinit(self: SignatureDecl, allocator: std.mem.Allocator) void {
        self.return_type.deinit(allocator);
        for (self.params) |param| {
            param.deinit(allocator);
        }
        allocator.free(self.params);
    }

    pub fn writeDebug(self: SignatureDecl, writer: anytype) !void {
        try writer.writeAll("    Signature ");
        try self.return_type.write(writer);
        try writer.writeByte(' ');
        try self.name.write(writer);
        try writer.writeByte('(');
        for (self.params, 0..) |param, index| {
            if (index != 0) try writer.writeAll(", ");
            try param.type_name.write(writer);
            try writer.writeByte(' ');
            try writer.writeAll(param.name.text);
        }
        try writer.writeAll(")\n");
    }
};

pub const FieldDecl = struct {
    type_name: TypeName,
    name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: FieldDecl, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
    }
};

pub const StructDecl = struct {
    attributes: []Attribute = &.{},
    is_export: bool,
    name: NameSegment,
    fields: []FieldDecl,
    span: SourceSpan,

    pub fn deinit(self: StructDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        for (self.fields) |field| {
            field.deinit(allocator);
        }
        allocator.free(self.fields);
    }
};

pub const EnumPayloadField = struct {
    type_name: TypeName,
    name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: EnumPayloadField, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
    }
};

pub const EnumVariant = struct {
    name: NameSegment,
    payload_fields: []EnumPayloadField,
    span: SourceSpan,

    pub fn deinit(self: EnumVariant, allocator: std.mem.Allocator) void {
        for (self.payload_fields) |field| {
            field.deinit(allocator);
        }
        allocator.free(self.payload_fields);
    }
};

pub const EnumDecl = struct {
    attributes: []Attribute = &.{},
    is_export: bool,
    is_must_use: bool = false,
    name: NameSegment,
    variants: []EnumVariant,
    span: SourceSpan,

    pub fn deinit(self: EnumDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        for (self.variants) |variant| {
            variant.deinit(allocator);
        }
        allocator.free(self.variants);
    }
};

pub const ConceptDecl = struct {
    attributes: []Attribute = &.{},
    name: NameSegment,
    generic_params: []GenericParam,
    signatures: []SignatureDecl,
    is_marker: bool = false,
    is_unsafe: bool = false,
    span: SourceSpan,

    pub fn deinit(self: ConceptDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        allocator.free(self.generic_params);
        for (self.signatures) |signature| {
            signature.deinit(allocator);
        }
        allocator.free(self.signatures);
    }
};

pub const InterfaceDecl = struct {
    attributes: []Attribute = &.{},
    name: NameSegment,
    signatures: []SignatureDecl,
    span: SourceSpan,

    pub fn deinit(self: InterfaceDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        for (self.signatures) |signature| {
            signature.deinit(allocator);
        }
        allocator.free(self.signatures);
    }
};

pub const ExternAbi = enum {
    c,

    pub fn debugName(self: ExternAbi) []const u8 {
        return switch (self) {
            .c => "C",
        };
    }
};

pub const ExternFunctionDecl = struct {
    signature: SignatureDecl,
    span: SourceSpan,

    pub fn deinit(self: ExternFunctionDecl, allocator: std.mem.Allocator) void {
        self.signature.deinit(allocator);
    }
};

pub const ExternBlock = struct {
    abi: ExternAbi,
    abi_span: SourceSpan,
    declarations: []ExternFunctionDecl,
    span: SourceSpan,

    pub fn deinit(self: ExternBlock, allocator: std.mem.Allocator) void {
        for (self.declarations) |declaration| {
            declaration.deinit(allocator);
        }
        allocator.free(self.declarations);
    }
};

pub const ImplDecl = struct {
    attributes: []Attribute = &.{},
    concept_name: TypeName,
    target_types: []TypeName,
    functions: []FunctionDecl,
    is_unsafe: bool = false,
    is_marker_semicolon: bool = false,
    span: SourceSpan,

    pub fn deinit(self: ImplDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        self.concept_name.deinit(allocator);
        for (self.target_types) |target_type| {
            target_type.deinit(allocator);
        }
        allocator.free(self.target_types);
        for (self.functions) |function| {
            function.deinit(allocator);
        }
        allocator.free(self.functions);
    }
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

pub const Expr = union(enum) {
    int_literal: IntLiteralExpr,
    bool_literal: BoolLiteralExpr,
    string_literal: StringLiteralExpr,
    identifier: IdentifierExpr,
    group: GroupExpr,
    unary: UnaryExpr,
    address_of: PrefixExpr,
    deref: PrefixExpr,
    move_expr: PrefixExpr,
    try_expr: TryExpr,
    compile_time: CompileTimeExpr,
    binary: BinaryExpr,
    call: CallExpr,
    method_call: MethodCallExpr,
    enum_constructor: EnumConstructorExpr,
    struct_literal: StructLiteralExpr,
    field_access: FieldAccessExpr,
    decide: DecideExpr,

    pub const IntLiteralExpr = struct { text: []const u8, span: SourceSpan };
    pub const BoolLiteralExpr = struct { value: bool, span: SourceSpan };
    pub const StringLiteralExpr = struct { text: []const u8, span: SourceSpan };
    pub const IdentifierExpr = struct { name: NameSegment, span: SourceSpan };
    pub const GroupExpr = struct { inner: *Expr, span: SourceSpan };
    pub const UnaryExpr = struct { op: UnaryOp, operand: *Expr, span: SourceSpan };
    pub const PrefixExpr = struct { operand: *Expr, span: SourceSpan };
    pub const TryExpr = struct { operand: *Expr, span: SourceSpan };
    pub const CompileTimeExpr = struct { operand: *Expr, span: SourceSpan };
    pub const BinaryExpr = struct { op: BinaryOp, left: *Expr, right: *Expr, span: SourceSpan };
    pub const CallExpr = struct {
        qualifier: ?NameSegment = null,
        callee: NameSegment,
        type_args: []TypeName = &.{},
        args: []*Expr,
        span: SourceSpan,
    };
    pub const MethodCallExpr = struct {
        receiver: *Expr,
        method_name: NameSegment,
        args: []*Expr,
        span: SourceSpan,
    };
    pub const EnumConstructorExpr = struct { enum_name: NameSegment, variant_name: NameSegment, args: []*Expr, span: SourceSpan };
    pub const StructLiteralExpr = struct { type_name: NameSegment, fields: []StructLiteralField, span: SourceSpan };
    pub const StructLiteralField = struct { name: NameSegment, value: *Expr, span: SourceSpan };
    pub const FieldAccessExpr = struct { receiver: *Expr, field_name: NameSegment, span: SourceSpan };
    pub const DecideExpr = struct { type_name: TypeName, arms: []DecideArm, span: SourceSpan };
    pub const DecideArm = struct {
        variant_name: NameSegment,
        condition: ?*Expr,
        score: *Expr,
        span: SourceSpan,

        pub fn deinit(self: DecideArm, allocator: std.mem.Allocator) void {
            if (self.condition) |condition| {
                condition.deinit(allocator);
                allocator.destroy(condition);
            }
            self.score.deinit(allocator);
            allocator.destroy(self.score);
        }
    };

    pub fn span(self: Expr) SourceSpan {
        return switch (self) {
            .int_literal => |expr| expr.span,
            .bool_literal => |expr| expr.span,
            .string_literal => |expr| expr.span,
            .identifier => |expr| expr.span,
            .group => |expr| expr.span,
            .unary => |expr| expr.span,
            .address_of => |expr| expr.span,
            .deref => |expr| expr.span,
            .move_expr => |expr| expr.span,
            .try_expr => |expr| expr.span,
            .compile_time => |expr| expr.span,
            .binary => |expr| expr.span,
            .call => |expr| expr.span,
            .method_call => |expr| expr.span,
            .enum_constructor => |expr| expr.span,
            .struct_literal => |expr| expr.span,
            .field_access => |expr| expr.span,
            .decide => |expr| expr.span,
        };
    }

    pub fn deinit(self: *Expr, allocator: std.mem.Allocator) void {
        switch (self.*) {
            .group => |expr| {
                expr.inner.deinit(allocator);
                allocator.destroy(expr.inner);
            },
            .unary, .address_of, .deref, .move_expr => |expr| {
                expr.operand.deinit(allocator);
                allocator.destroy(expr.operand);
            },
            .try_expr, .compile_time => |expr| {
                expr.operand.deinit(allocator);
                allocator.destroy(expr.operand);
            },
            .binary => |expr| {
                expr.left.deinit(allocator);
                allocator.destroy(expr.left);
                expr.right.deinit(allocator);
                allocator.destroy(expr.right);
            },
            .call => |expr| {
                for (expr.type_args) |type_arg| {
                    type_arg.deinit(allocator);
                }
                if (expr.type_args.len > 0) allocator.free(expr.type_args);
                for (expr.args) |arg| {
                    arg.deinit(allocator);
                    allocator.destroy(arg);
                }
                allocator.free(expr.args);
            },
            .method_call => |expr| {
                expr.receiver.deinit(allocator);
                allocator.destroy(expr.receiver);
                for (expr.args) |arg| {
                    arg.deinit(allocator);
                    allocator.destroy(arg);
                }
                allocator.free(expr.args);
            },
            .enum_constructor => |expr| {
                for (expr.args) |arg| {
                    arg.deinit(allocator);
                    allocator.destroy(arg);
                }
                allocator.free(expr.args);
            },
            .struct_literal => |expr| {
                for (expr.fields) |field| {
                    field.value.deinit(allocator);
                    allocator.destroy(field.value);
                }
                allocator.free(expr.fields);
            },
            .field_access => |expr| {
                expr.receiver.deinit(allocator);
                allocator.destroy(expr.receiver);
            },
            .decide => |expr| {
                expr.type_name.deinit(allocator);
                for (expr.arms) |arm| arm.deinit(allocator);
                allocator.free(expr.arms);
            },
            .int_literal, .bool_literal, .string_literal, .identifier => {},
        }
    }

    pub fn writeDebug(self: Expr, writer: anytype, depth: usize) !void {
        try writeIndent(writer, depth);
        switch (self) {
            .int_literal => |expr| {
                try writer.writeAll("Int ");
                try writer.writeAll(expr.text);
                try writer.writeByte('\n');
            },
            .bool_literal => |expr| {
                try writer.writeAll("Bool ");
                try writer.writeAll(if (expr.value) "true" else "false");
                try writer.writeByte('\n');
            },
            .string_literal => |expr| {
                try writer.writeAll("String ");
                try writer.writeAll(expr.text);
                try writer.writeByte('\n');
            },
            .identifier => |expr| {
                try writer.writeAll("Identifier ");
                try writer.writeAll(expr.name.text);
                try writer.writeByte('\n');
            },
            .group => |expr| {
                try writer.writeAll("Group\n");
                try expr.inner.writeDebug(writer, depth + 1);
            },
            .unary => |expr| {
                try writer.writeAll("Unary ");
                try writer.writeAll(expr.op.lexeme());
                try writer.writeByte('\n');
                try expr.operand.writeDebug(writer, depth + 1);
            },
            .address_of => |expr| {
                try writer.writeAll("AddressOf\n");
                try expr.operand.writeDebug(writer, depth + 1);
            },
            .deref => |expr| {
                try writer.writeAll("Deref\n");
                try expr.operand.writeDebug(writer, depth + 1);
            },
            .move_expr => |expr| {
                try writer.writeAll("Move\n");
                try expr.operand.writeDebug(writer, depth + 1);
            },
            .try_expr => |expr| {
                try writer.writeAll("Try\n");
                try expr.operand.writeDebug(writer, depth + 1);
            },
            .compile_time => |expr| {
                try writer.writeAll("CompileTime\n");
                try expr.operand.writeDebug(writer, depth + 1);
            },
            .binary => |expr| {
                try writer.writeAll("Binary ");
                try writer.writeAll(expr.op.lexeme());
                try writer.writeByte('\n');
                try expr.left.writeDebug(writer, depth + 1);
                try expr.right.writeDebug(writer, depth + 1);
            },
            .call => |expr| {
                try writer.writeAll("Call ");
                if (expr.qualifier) |qualifier| {
                    try writer.writeAll(qualifier.text);
                    try writer.writeByte('.');
                }
                try writer.writeAll(expr.callee.text);
                if (expr.type_args.len != 0) {
                    try writer.writeByte('<');
                    for (expr.type_args, 0..) |type_arg, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try type_arg.write(writer);
                    }
                    try writer.writeByte('>');
                }
                try writer.writeByte('\n');
                for (expr.args) |arg| try arg.writeDebug(writer, depth + 1);
            },
            .method_call => |expr| {
                try writer.writeAll("MethodCall .");
                try writer.writeAll(expr.method_name.text);
                try writer.writeByte('\n');
                try expr.receiver.writeDebug(writer, depth + 1);
                for (expr.args) |arg| try arg.writeDebug(writer, depth + 1);
            },
            .enum_constructor => |expr| {
                try writer.writeAll("EnumConstructor ");
                try writer.writeAll(expr.enum_name.text);
                try writer.writeAll("::");
                try writer.writeAll(expr.variant_name.text);
                try writer.writeByte('\n');
                for (expr.args) |arg| try arg.writeDebug(writer, depth + 1);
            },
            .struct_literal => |expr| {
                try writer.writeAll("StructLiteral ");
                try writer.writeAll(expr.type_name.text);
                try writer.writeByte('\n');
                for (expr.fields) |field| {
                    try writeIndent(writer, depth + 1);
                    try writer.writeAll("Field ");
                    try writer.writeAll(field.name.text);
                    try writer.writeByte('\n');
                    try field.value.writeDebug(writer, depth + 2);
                }
            },
            .field_access => |expr| {
                try writer.writeAll("FieldAccess .");
                try writer.writeAll(expr.field_name.text);
                try writer.writeByte('\n');
                try expr.receiver.writeDebug(writer, depth + 1);
            },
            .decide => |expr| {
                try writer.writeAll("Decide ");
                try expr.type_name.write(writer);
                try writer.writeByte('\n');
                for (expr.arms) |arm| {
                    try writeIndent(writer, depth + 1);
                    try writer.writeAll("Arm ");
                    try writer.writeAll(arm.variant_name.text);
                    try writer.writeByte('\n');
                    if (arm.condition) |condition| {
                        try writeIndent(writer, depth + 2);
                        try writer.writeAll("When\n");
                        try condition.writeDebug(writer, depth + 3);
                    }
                    try writeIndent(writer, depth + 2);
                    try writer.writeAll("Score\n");
                    try arm.score.writeDebug(writer, depth + 3);
                }
            },
        }
    }
};

pub const ReturnStmt = struct {
    value: ?*Expr,
    span: SourceSpan,

    pub fn deinit(self: ReturnStmt, allocator: std.mem.Allocator) void {
        if (self.value) |value| {
            value.deinit(allocator);
            allocator.destroy(value);
        }
    }
};

pub const LocalDeclStmt = struct {
    type_name: TypeName,
    name: NameSegment,
    initializer: *Expr,
    span: SourceSpan,

    pub fn deinit(self: LocalDeclStmt, allocator: std.mem.Allocator) void {
        self.type_name.deinit(allocator);
        self.initializer.deinit(allocator);
        allocator.destroy(self.initializer);
    }
};

pub const AssignmentStmt = struct {
    target: *Expr,
    value: *Expr,
    span: SourceSpan,

    pub fn deinit(self: AssignmentStmt, allocator: std.mem.Allocator) void {
        self.target.deinit(allocator);
        allocator.destroy(self.target);
        self.value.deinit(allocator);
        allocator.destroy(self.value);
    }
};

pub const ExprStmt = struct {
    value: *Expr,
    span: SourceSpan,

    pub fn deinit(self: ExprStmt, allocator: std.mem.Allocator) void {
        self.value.deinit(allocator);
        allocator.destroy(self.value);
    }
};

pub const DiscardStmt = struct {
    value: *Expr,
    span: SourceSpan,

    pub fn deinit(self: DiscardStmt, allocator: std.mem.Allocator) void {
        self.value.deinit(allocator);
        allocator.destroy(self.value);
    }
};

pub const IfStmt = struct {
    condition: *Expr,
    then_block: BlockStmt,
    else_block: ?BlockStmt,
    span: SourceSpan,

    pub fn deinit(self: IfStmt, allocator: std.mem.Allocator) void {
        self.condition.deinit(allocator);
        allocator.destroy(self.condition);
        self.then_block.deinit(allocator);
        if (self.else_block) |else_block| else_block.deinit(allocator);
    }
};

pub const WhileStmt = struct {
    condition: *Expr,
    body: BlockStmt,
    span: SourceSpan,

    pub fn deinit(self: WhileStmt, allocator: std.mem.Allocator) void {
        self.condition.deinit(allocator);
        allocator.destroy(self.condition);
        self.body.deinit(allocator);
    }
};

pub const UnsafeBlockStmt = struct {
    body: BlockStmt,
    span: SourceSpan,

    pub fn deinit(self: UnsafeBlockStmt, allocator: std.mem.Allocator) void {
        self.body.deinit(allocator);
    }
};

pub const PatternBinding = struct {
    name: NameSegment,
};

pub const EnumVariantPattern = struct {
    enum_name: NameSegment,
    variant_name: NameSegment,
    bindings: []PatternBinding,
    span: SourceSpan,

    pub fn deinit(self: EnumVariantPattern, allocator: std.mem.Allocator) void {
        allocator.free(self.bindings);
    }
};

pub const MatchPattern = union(enum) {
    int_literal: Expr.IntLiteralExpr,
    bool_literal: Expr.BoolLiteralExpr,
    wildcard: SourceSpan,
    enum_variant: EnumVariantPattern,

    pub fn span(self: MatchPattern) SourceSpan {
        return switch (self) {
            .int_literal => |pattern| pattern.span,
            .bool_literal => |pattern| pattern.span,
            .wildcard => |pattern_span| pattern_span,
            .enum_variant => |pattern| pattern.span,
        };
    }

    pub fn writeDebug(self: MatchPattern, writer: anytype) !void {
        switch (self) {
            .int_literal => |pattern| try writer.writeAll(pattern.text),
            .bool_literal => |pattern| try writer.writeAll(if (pattern.value) "true" else "false"),
            .wildcard => try writer.writeByte('_'),
            .enum_variant => |pattern| {
                try writer.print("{s}::{s}", .{ pattern.enum_name.text, pattern.variant_name.text });
                if (pattern.bindings.len != 0) {
                    try writer.writeByte('(');
                    for (pattern.bindings, 0..) |binding, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try writer.writeAll(binding.name.text);
                    }
                    try writer.writeByte(')');
                }
            },
        }
    }
};

pub const MatchArm = struct {
    pattern: MatchPattern,
    body: Stmt,
    span: SourceSpan,

    pub fn deinit(self: MatchArm, allocator: std.mem.Allocator) void {
        switch (self.pattern) {
            .enum_variant => |pattern| pattern.deinit(allocator),
            else => {},
        }
        self.body.deinit(allocator);
    }
};

pub const MatchStmt = struct {
    scrutinee: *Expr,
    arms: []MatchArm,
    span: SourceSpan,

    pub fn deinit(self: MatchStmt, allocator: std.mem.Allocator) void {
        self.scrutinee.deinit(allocator);
        allocator.destroy(self.scrutinee);
        for (self.arms) |arm| arm.deinit(allocator);
        allocator.free(self.arms);
    }
};

pub const TransitionMatchArm = struct {
    pattern: MatchPattern,
    target_name: NameSegment,
    span: SourceSpan,

    pub fn deinit(self: TransitionMatchArm, allocator: std.mem.Allocator) void {
        switch (self.pattern) {
            .enum_variant => |pattern| pattern.deinit(allocator),
            else => {},
        }
    }
};

pub const TransitionMatchTarget = struct {
    scrutinee: *Expr,
    arms: []TransitionMatchArm,
    span: SourceSpan,

    pub fn deinit(self: TransitionMatchTarget, allocator: std.mem.Allocator) void {
        self.scrutinee.deinit(allocator);
        allocator.destroy(self.scrutinee);
        for (self.arms) |arm| arm.deinit(allocator);
        allocator.free(self.arms);
    }
};

pub const TransitionDecideCase = struct {
    target_name: NameSegment,
    condition: ?*Expr,
    score: *Expr,
    span: SourceSpan,

    pub fn deinit(self: TransitionDecideCase, allocator: std.mem.Allocator) void {
        if (self.condition) |condition| {
            condition.deinit(allocator);
            allocator.destroy(condition);
        }
        self.score.deinit(allocator);
        allocator.destroy(self.score);
    }
};

pub const TransitionDecideTarget = struct {
    cases: []TransitionDecideCase,
    span: SourceSpan,

    pub fn deinit(self: TransitionDecideTarget, allocator: std.mem.Allocator) void {
        for (self.cases) |case| case.deinit(allocator);
        allocator.free(self.cases);
    }
};

pub const TransitionTarget = union(enum) {
    literal_state: NameSegment,
    match_state: TransitionMatchTarget,
    decide_state: TransitionDecideTarget,

    pub fn deinit(self: TransitionTarget, allocator: std.mem.Allocator) void {
        switch (self) {
            .literal_state => {},
            .match_state => |match_target| match_target.deinit(allocator),
            .decide_state => |decide_target| decide_target.deinit(allocator),
        }
    }
};

pub const TransitionStmt = struct {
    target: TransitionTarget,
    span: SourceSpan,

    pub fn deinit(self: TransitionStmt, allocator: std.mem.Allocator) void {
        self.target.deinit(allocator);
    }
};

pub const PanicStmt = struct {
    reason: Expr.StringLiteralExpr,
    reason_span: SourceSpan,
    span: SourceSpan,
};

pub const YieldStmt = struct {
    span: SourceSpan,
};

pub const AssertStmt = struct {
    condition: *Expr,
    reason: Expr.StringLiteralExpr,
    condition_span: SourceSpan,
    reason_span: SourceSpan,
    span: SourceSpan,

    pub fn deinit(self: AssertStmt, allocator: std.mem.Allocator) void {
        self.condition.deinit(allocator);
        allocator.destroy(self.condition);
    }
};

pub const Stmt = union(enum) {
    local_decl: LocalDeclStmt,
    assignment: AssignmentStmt,
    expr_stmt: ExprStmt,
    discard_stmt: DiscardStmt,
    panic_stmt: PanicStmt,
    yield_stmt: YieldStmt,
    assert_stmt: AssertStmt,
    return_stmt: ReturnStmt,
    transition_stmt: TransitionStmt,
    if_stmt: IfStmt,
    while_stmt: WhileStmt,
    unsafe_block: UnsafeBlockStmt,
    match_stmt: MatchStmt,
    block_stmt: BlockStmt,

    pub fn deinit(self: Stmt, allocator: std.mem.Allocator) void {
        switch (self) {
            .local_decl => |stmt| stmt.deinit(allocator),
            .assignment => |stmt| stmt.deinit(allocator),
            .expr_stmt => |stmt| stmt.deinit(allocator),
            .discard_stmt => |stmt| stmt.deinit(allocator),
            .panic_stmt => {},
            .yield_stmt => {},
            .assert_stmt => |stmt| stmt.deinit(allocator),
            .return_stmt => |stmt| stmt.deinit(allocator),
            .transition_stmt => |stmt| stmt.deinit(allocator),
            .if_stmt => |stmt| stmt.deinit(allocator),
            .while_stmt => |stmt| stmt.deinit(allocator),
            .unsafe_block => |stmt| stmt.deinit(allocator),
            .match_stmt => |stmt| stmt.deinit(allocator),
            .block_stmt => |stmt| stmt.deinit(allocator),
        }
    }

    pub fn writeDebug(self: Stmt, writer: anytype, depth: usize) !void {
        switch (self) {
            .local_decl => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("LocalDecl ");
                try stmt.type_name.write(writer);
                try writer.writeByte(' ');
                try writer.writeAll(stmt.name.text);
                try writer.writeByte('\n');
                try stmt.initializer.writeDebug(writer, depth + 1);
            },
            .assignment => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("Assignment\n");
                try stmt.target.writeDebug(writer, depth + 1);
                try stmt.value.writeDebug(writer, depth + 1);
            },
            .expr_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("ExprStmt\n");
                try stmt.value.writeDebug(writer, depth + 1);
            },
            .discard_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("Discard\n");
                try stmt.value.writeDebug(writer, depth + 1);
            },
            .panic_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("Panic ");
                try writer.writeAll(stmt.reason.text);
                try writer.writeByte('\n');
            },
            .yield_stmt => {
                try writeIndent(writer, depth);
                try writer.writeAll("Yield\n");
            },
            .assert_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("Assert ");
                try writer.writeAll(stmt.reason.text);
                try writer.writeByte('\n');
                try stmt.condition.writeDebug(writer, depth + 1);
            },
            .return_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("Return\n");
                if (stmt.value) |value| try value.writeDebug(writer, depth + 1);
            },
            .transition_stmt => |stmt| {
                try writeIndent(writer, depth);
                switch (stmt.target) {
                    .literal_state => |target_name| {
                        try writer.writeAll("Transition ");
                        try writer.writeAll(target_name.text);
                        try writer.writeByte('\n');
                    },
                    .match_state => |match_target| {
                        try writer.writeAll("TransitionMatch\n");
                        try writeIndent(writer, depth + 1);
                        try writer.writeAll("Scrutinee\n");
                        try match_target.scrutinee.writeDebug(writer, depth + 2);
                        for (match_target.arms) |arm| {
                            try writeIndent(writer, depth + 1);
                            try writer.writeAll("Arm ");
                            try arm.pattern.writeDebug(writer);
                            try writer.writeAll(" => ");
                            try writer.writeAll(arm.target_name.text);
                            try writer.writeByte('\n');
                        }
                    },
                    .decide_state => |decide_target| {
                        try writer.writeAll("TransitionDecide\n");
                        for (decide_target.cases) |case| {
                            try writeIndent(writer, depth + 1);
                            try writer.writeAll("Case ");
                            try writer.writeAll(case.target_name.text);
                            if (case.condition) |condition| {
                                try writer.writeAll(" when\n");
                                try condition.writeDebug(writer, depth + 2);
                                try writeIndent(writer, depth + 1);
                                try writer.writeAll("Score\n");
                            } else {
                                try writer.writeAll(" score\n");
                            }
                            try case.score.writeDebug(writer, depth + 2);
                        }
                    },
                }
            },
            .if_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("If\n");
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Condition\n");
                try stmt.condition.writeDebug(writer, depth + 2);
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Then\n");
                for (stmt.then_block.statements) |child| try child.writeDebug(writer, depth + 2);
                if (stmt.else_block) |else_block| {
                    try writeIndent(writer, depth + 1);
                    try writer.writeAll("Else\n");
                    for (else_block.statements) |child| try child.writeDebug(writer, depth + 2);
                }
            },
            .while_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("While\n");
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Condition\n");
                try stmt.condition.writeDebug(writer, depth + 2);
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Body\n");
                for (stmt.body.statements) |child| try child.writeDebug(writer, depth + 2);
            },
            .unsafe_block => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("UnsafeBlock\n");
                for (stmt.body.statements) |child| try child.writeDebug(writer, depth + 1);
            },
            .match_stmt => |stmt| {
                try writeIndent(writer, depth);
                try writer.writeAll("Match\n");
                try writeIndent(writer, depth + 1);
                try writer.writeAll("Scrutinee\n");
                try stmt.scrutinee.writeDebug(writer, depth + 2);
                for (stmt.arms) |arm| {
                    try writeIndent(writer, depth + 1);
                    try writer.writeAll("Arm ");
                    try arm.pattern.writeDebug(writer);
                    try writer.writeByte('\n');
                    try arm.body.writeDebug(writer, depth + 2);
                }
            },
            .block_stmt => |block| {
                try writeIndent(writer, depth);
                try writer.writeAll("Block\n");
                for (block.statements) |child| try child.writeDebug(writer, depth + 1);
            },
        }
    }
};

pub const BlockStmt = struct {
    statements: []Stmt,
    span: SourceSpan,

    pub fn deinit(self: BlockStmt, allocator: std.mem.Allocator) void {
        for (self.statements) |stmt| stmt.deinit(allocator);
        allocator.free(self.statements);
    }
};

pub const FunctionBody = struct {
    span: SourceSpan,
    block: ?BlockStmt = null,

    pub fn deinit(self: FunctionBody, allocator: std.mem.Allocator) void {
        if (self.block) |block| block.deinit(allocator);
    }

    pub fn writeDebug(self: FunctionBody, writer: anytype) !void {
        try writer.writeAll("    Body\n");
        if (self.block) |block| {
            for (block.statements) |stmt| try stmt.writeDebug(writer, 3);
        }
    }
};

pub const TypeParamDecl = struct {
    name: NameSegment,
    constraint: ?TypeName = null,
    span: SourceSpan,

    pub fn deinit(self: TypeParamDecl, allocator: std.mem.Allocator) void {
        if (self.constraint) |constraint| constraint.deinit(allocator);
    }
};

pub const TemplateDecl = struct {
    attributes: []Attribute = &.{},
    params: []TypeParamDecl,
    body: FunctionDecl,
    span: SourceSpan,

    pub fn deinit(self: TemplateDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        for (self.params) |param| param.deinit(allocator);
        allocator.free(self.params);
        self.body.deinit(allocator);
    }
};

pub const StaticAssertDecl = struct {
    expr: *Expr,
    span: SourceSpan,

    pub fn deinit(self: StaticAssertDecl, allocator: std.mem.Allocator) void {
        self.expr.deinit(allocator);
        allocator.destroy(self.expr);
    }
};

pub const CompileTimeCapabilitySyntax = struct {
    name: NameSegment,
};

pub const AllocationEffect = enum {
    unspecified,
    noalloc,
    alloc,

    pub fn debugName(self: AllocationEffect) []const u8 {
        return switch (self) {
            .unspecified => "Unspecified",
            .noalloc => "NoAlloc",
            .alloc => "Alloc",
        };
    }
};

pub const FunctionDecl = struct {
    attributes: []Attribute = &.{},
    is_export: bool,
    export_abi: ?ExternAbi = null,
    export_abi_span: SourceSpan = .{ .start = 0, .length = 0 },
    is_unsafe: bool = false,
    is_compile_time: bool = false,
    allocation_effect: AllocationEffect = .unspecified,
    compile_time_capabilities: []CompileTimeCapabilitySyntax = &.{},
    signature: SignatureDecl,
    body: ?FunctionBody,
    span: SourceSpan,

    pub fn deinit(self: FunctionDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        allocator.free(self.compile_time_capabilities);
        self.signature.deinit(allocator);
        if (self.body) |body| body.deinit(allocator);
    }
};

pub const MachineStateDecl = struct {
    name: NameSegment,
    body: BlockStmt,
    span: SourceSpan,

    pub fn deinit(self: MachineStateDecl, allocator: std.mem.Allocator) void {
        self.body.deinit(allocator);
    }
};

pub const MachineDecl = struct {
    attributes: []Attribute = &.{},
    name: NameSegment,
    params: []ParamDecl,
    return_type: TypeName,
    fields: []FieldDecl = &.{},
    states: []MachineStateDecl,
    allocation_effect: AllocationEffect = .unspecified,
    span: SourceSpan,

    pub fn deinit(self: MachineDecl, allocator: std.mem.Allocator) void {
        for (self.attributes) |attribute| attribute.deinit(allocator);
        allocator.free(self.attributes);
        for (self.params) |param| param.deinit(allocator);
        allocator.free(self.params);
        self.return_type.deinit(allocator);
        for (self.fields) |field| field.deinit(allocator);
        allocator.free(self.fields);
        for (self.states) |state| state.deinit(allocator);
        allocator.free(self.states);
    }
};

pub const Item = union(enum) {
    function_decl: FunctionDecl,
    machine_decl: MachineDecl,
    template_decl: TemplateDecl,
    struct_decl: StructDecl,
    enum_decl: EnumDecl,
    concept_decl: ConceptDecl,
    interface_decl: InterfaceDecl,
    extern_block: ExternBlock,
    impl_decl: ImplDecl,
    static_assert_decl: StaticAssertDecl,

    pub fn deinit(self: Item, allocator: std.mem.Allocator) void {
        switch (self) {
            .function_decl => |function_decl| function_decl.deinit(allocator),
            .machine_decl => |machine_decl| machine_decl.deinit(allocator),
            .template_decl => |template_decl| template_decl.deinit(allocator),
            .struct_decl => |struct_decl| struct_decl.deinit(allocator),
            .enum_decl => |enum_decl| enum_decl.deinit(allocator),
            .concept_decl => |concept_decl| concept_decl.deinit(allocator),
            .interface_decl => |interface_decl| interface_decl.deinit(allocator),
            .extern_block => |extern_block| extern_block.deinit(allocator),
            .impl_decl => |impl_decl| impl_decl.deinit(allocator),
            .static_assert_decl => |static_assert_decl| static_assert_decl.deinit(allocator),
        }
    }

    pub fn writeDebug(self: Item, writer: anytype) !void {
        switch (self) {
            .function_decl => |function_decl| {
                try writeAttributesDebug(function_decl.attributes, writer);
                if (function_decl.is_compile_time and function_decl.is_export and function_decl.is_unsafe) {
                    try writer.writeAll("  CompileTime Export Unsafe Function ");
                } else if (function_decl.is_compile_time and function_decl.is_export) {
                    try writer.writeAll("  CompileTime Export Function ");
                } else if (function_decl.is_compile_time and function_decl.is_unsafe) {
                    try writer.writeAll("  CompileTime Unsafe Function ");
                } else if (function_decl.is_compile_time) {
                    try writer.writeAll("  CompileTime Function ");
                } else if (function_decl.is_export and function_decl.is_unsafe) {
                    try writer.writeAll("  Export Unsafe Function ");
                } else if (function_decl.export_abi) |abi| {
                    try writer.print("  ExportFunction \"{s}\" ", .{abi.debugName()});
                } else if (function_decl.is_export) {
                    try writer.writeAll("  Export Function ");
                } else if (function_decl.is_unsafe) {
                    try writer.writeAll("  Unsafe Function ");
                } else {
                    try writer.writeAll("  Function ");
                }
                if (function_decl.compile_time_capabilities.len != 0) {
                    try writer.writeAll("capabilities=[");
                    for (function_decl.compile_time_capabilities, 0..) |capability, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try writer.writeAll(capability.name.text);
                    }
                    try writer.writeAll("] ");
                }
                if (function_decl.allocation_effect != .unspecified) {
                    try writer.writeAll("effect=");
                    try writer.writeAll(function_decl.allocation_effect.debugName());
                    try writer.writeByte(' ');
                }
                try function_decl.signature.return_type.write(writer);
                try writer.writeByte(' ');
                try function_decl.signature.name.write(writer);
                try writer.writeByte('(');
                for (function_decl.signature.params, 0..) |param, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try param.type_name.write(writer);
                    try writer.writeByte(' ');
                    try writer.writeAll(param.name.text);
                }
                try writer.writeAll(")\n");
                if (function_decl.body) |body| {
                    try body.writeDebug(writer);
                }
            },
            .machine_decl => |machine_decl| {
                try writeAttributesDebug(machine_decl.attributes, writer);
                try writer.writeAll("  Machine ");
                if (machine_decl.allocation_effect != .unspecified) {
                    try writer.writeAll("effect=");
                    try writer.writeAll(machine_decl.allocation_effect.debugName());
                    try writer.writeByte(' ');
                }
                try writer.writeAll(machine_decl.name.text);
                try writer.writeByte('(');
                for (machine_decl.params, 0..) |param, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try param.type_name.write(writer);
                    try writer.writeByte(' ');
                    try writer.writeAll(param.name.text);
                }
                try writer.writeAll(") -> ");
                try machine_decl.return_type.write(writer);
                try writer.writeByte('\n');
                for (machine_decl.states) |state| {
                    try writer.writeAll("    State ");
                    try writer.writeAll(state.name.text);
                    try writer.writeByte('\n');
                    try writer.writeAll("      Body\n");
                    for (state.body.statements) |stmt| try stmt.writeDebug(writer, 4);
                }
            },
            .template_decl => |template_decl| {
                try writeAttributesDebug(template_decl.attributes, writer);
                try writer.writeAll("  Template<");
                for (template_decl.params, 0..) |param, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try writer.writeAll(param.name.text);
                    if (param.constraint) |constraint| {
                        try writer.writeAll(": ");
                        try constraint.write(writer);
                    }
                }
                try writer.writeAll(">\n");
                try writer.writeAll("    Function ");
                if (template_decl.body.allocation_effect != .unspecified) {
                    try writer.writeAll("effect=");
                    try writer.writeAll(template_decl.body.allocation_effect.debugName());
                    try writer.writeByte(' ');
                }
                try template_decl.body.signature.return_type.write(writer);
                try writer.writeByte(' ');
                try template_decl.body.signature.name.write(writer);
                try writer.writeByte('(');
                for (template_decl.body.signature.params, 0..) |param, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try param.type_name.write(writer);
                    try writer.writeByte(' ');
                    try writer.writeAll(param.name.text);
                }
                try writer.writeAll(")\n");
                if (template_decl.body.body) |body| {
                    try body.writeDebug(writer);
                }
            },
            .struct_decl => |struct_decl| {
                try writeAttributesDebug(struct_decl.attributes, writer);
                if (struct_decl.is_export) {
                    try writer.writeAll("  Export Struct ");
                } else {
                    try writer.writeAll("  Struct ");
                }
                try writer.writeAll(struct_decl.name.text);
                try writer.writeByte('\n');

                for (struct_decl.fields) |field| {
                    try writer.writeAll("    Field ");
                    try field.type_name.write(writer);
                    try writer.writeByte(' ');
                    try writer.writeAll(field.name.text);
                    try writer.writeByte('\n');
                }
            },
            .enum_decl => |enum_decl| {
                try writeAttributesDebug(enum_decl.attributes, writer);
                if (enum_decl.is_export and enum_decl.is_must_use) {
                    try writer.writeAll("  Export MustUse Enum ");
                } else if (enum_decl.is_export) {
                    try writer.writeAll("  Export Enum ");
                } else if (enum_decl.is_must_use) {
                    try writer.writeAll("  MustUse Enum ");
                } else {
                    try writer.writeAll("  Enum ");
                }
                try writer.writeAll(enum_decl.name.text);
                try writer.writeByte('\n');

                for (enum_decl.variants) |variant| {
                    try writer.writeAll("    Variant ");
                    try writer.writeAll(variant.name.text);
                    try writer.writeByte('\n');

                    for (variant.payload_fields) |field| {
                        try writer.writeAll("      Payload ");
                        try field.type_name.write(writer);
                        try writer.writeByte(' ');
                        try writer.writeAll(field.name.text);
                        try writer.writeByte('\n');
                    }
                }
            },
            .concept_decl => |concept_decl| {
                try writeAttributesDebug(concept_decl.attributes, writer);
                if (concept_decl.is_unsafe and concept_decl.is_marker) {
                    try writer.writeAll("  Unsafe Marker Concept ");
                } else if (concept_decl.is_marker) {
                    try writer.writeAll("  Marker Concept ");
                } else if (concept_decl.is_unsafe) {
                    try writer.writeAll("  Unsafe Concept ");
                } else {
                    try writer.writeAll("  Concept ");
                }
                try writer.writeAll(concept_decl.name.text);
                if (concept_decl.generic_params.len != 0) {
                    try writer.writeByte('<');
                    for (concept_decl.generic_params, 0..) |generic_param, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try writer.writeAll(generic_param.text);
                    }
                    try writer.writeByte('>');
                }
                try writer.writeByte('\n');
                for (concept_decl.signatures) |signature| {
                    try signature.writeDebug(writer);
                }
            },
            .interface_decl => |interface_decl| {
                try writeAttributesDebug(interface_decl.attributes, writer);
                try writer.writeAll("  Interface ");
                try writer.writeAll(interface_decl.name.text);
                try writer.writeByte('\n');
                for (interface_decl.signatures) |signature| {
                    try signature.writeDebug(writer);
                }
            },
            .extern_block => |extern_block| {
                try writer.writeAll("  ExternBlock \"");
                try writer.writeAll(extern_block.abi.debugName());
                try writer.writeAll("\"\n");
                for (extern_block.declarations) |declaration| {
                    try writer.writeAll("    Function ");
                    try declaration.signature.return_type.write(writer);
                    try writer.writeByte(' ');
                    try declaration.signature.name.write(writer);
                    try writer.writeByte('(');
                    for (declaration.signature.params, 0..) |param, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try param.type_name.write(writer);
                        try writer.writeByte(' ');
                        try writer.writeAll(param.name.text);
                    }
                    try writer.writeAll(")\n");
                }
            },
            .static_assert_decl => |static_assert_decl| {
                try writer.writeAll("  StaticAssert\n");
                try static_assert_decl.expr.writeDebug(writer, 2);
            },
            .impl_decl => |impl_decl| {
                try writeAttributesDebug(impl_decl.attributes, writer);
                if (impl_decl.is_unsafe) {
                    try writer.writeAll("  Unsafe Impl ");
                } else {
                    try writer.writeAll("  Impl ");
                }
                try impl_decl.concept_name.write(writer);
                try writer.writeByte('<');
                for (impl_decl.target_types, 0..) |target_type, index| {
                    if (index != 0) try writer.writeAll(", ");
                    try target_type.write(writer);
                }
                try writer.writeByte('>');
                if (impl_decl.is_marker_semicolon) try writer.writeAll(" semicolon");
                try writer.writeByte('\n');
                for (impl_decl.functions) |function| {
                    try writer.writeAll("    Function ");
                    if (function.allocation_effect != .unspecified) {
                        try writer.writeAll("effect=");
                        try writer.writeAll(function.allocation_effect.debugName());
                        try writer.writeByte(' ');
                    }
                    try function.signature.return_type.write(writer);
                    try writer.writeByte(' ');
                    try function.signature.name.write(writer);
                    try writer.writeByte('(');
                    for (function.signature.params, 0..) |param, index| {
                        if (index != 0) try writer.writeAll(", ");
                        try param.type_name.write(writer);
                        try writer.writeByte(' ');
                        try writer.writeAll(param.name.text);
                    }
                    try writer.writeAll(")\n");
                    if (function.body) |body| try body.writeDebug(writer);
                }
            },
        }
    }
};

fn writeAttributesDebug(attributes: []const Attribute, writer: anytype) !void {
    for (attributes) |attribute| {
        try attribute.writeDebug(writer);
    }
}

fn writeIndent(writer: anytype, depth: usize) !void {
    var index: usize = 0;
    while (index < depth) : (index += 1) {
        try writer.writeAll("  ");
    }
}

pub const CompilationUnit = struct {
    span: SourceSpan,
    module: ?ModuleDecl,
    imports: []ImportDecl,
    items: []Item,

    pub fn deinit(self: CompilationUnit, allocator: std.mem.Allocator) void {
        if (self.module) |module| module.deinit(allocator);
        for (self.imports) |import_decl| {
            import_decl.deinit(allocator);
        }
        allocator.free(self.imports);
        for (self.items) |item| {
            item.deinit(allocator);
        }
        allocator.free(self.items);
    }

    pub fn writeDebug(self: CompilationUnit, writer: anytype) !void {
        try writer.writeAll("CompilationUnit\n");
        if (self.module) |module| {
            try writer.writeAll("  Module ");
            try module.name.write(writer);
            try writer.writeByte('\n');
        }
        for (self.imports) |import_decl| {
            try writer.writeAll("  Import ");
            try import_decl.name.write(writer);
            try writer.writeByte('\n');
        }
        for (self.items) |item| {
            try item.writeDebug(writer);
        }
    }

    pub fn debugString(self: CompilationUnit, allocator: std.mem.Allocator) ![]u8 {
        var output: std.Io.Writer.Allocating = .init(allocator);
        errdefer output.deinit();
        try self.writeDebug(&output.writer);
        return try output.toOwnedSlice();
    }
};

pub fn spanFromBounds(start: usize, end: usize) SourceSpan {
    return .{ .start = start, .length = end - start };
}

pub fn mergeSpans(first: SourceSpan, second: SourceSpan) SourceSpan {
    const first_end = first.end() orelse first.start;
    const second_end = second.end() orelse second.start;
    return spanFromBounds(@min(first.start, second.start), @max(first_end, second_end));
}

test "qualified name debug writing joins segments with dots" {
    const parts = try std.testing.allocator.dupe(NameSegment, &.{
        .{ .text = "Example", .span = .{ .start = 0, .length = 7 } },
        .{ .text = "Compiler", .span = .{ .start = 8, .length = 8 } },
    });
    defer std.testing.allocator.free(parts);

    const name = QualifiedName{ .parts = parts, .span = .{ .start = 0, .length = 16 } };
    var output: std.Io.Writer.Allocating = .init(std.testing.allocator);
    defer output.deinit();

    try name.write(&output.writer);
    try std.testing.expectEqualStrings("Example.Compiler", output.written());
}
