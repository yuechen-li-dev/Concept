const std = @import("std");
const source_model = @import("source.zig");

pub const SourceFile = source_model.SourceFile;
pub const SourceLocation = source_model.SourceLocation;
pub const SourceSpan = source_model.SourceSpan;

pub const Severity = enum {
    @"error",
    warning,
    note,
    help,

    pub fn format(self: Severity) []const u8 {
        return switch (self) {
            .@"error" => "error",
            .warning => "warning",
            .note => "note",
            .help => "help",
        };
    }
};

pub const DiagnosticCode = enum {
    InvalidCharacter,
    UnterminatedString,
    UnterminatedBlockComment,
    UnexpectedToken,
    ExpectedItem,
    DuplicateModuleDeclaration,
    UnterminatedChar,
    EmptyCharLiteral,
    InvalidEscapeSequence,
    InvalidExecutableSubset,
    DuplicateTopLevelName,
    UnknownTypeName,
    DuplicateStructField,
    DuplicateEnumVariant,
    UnsupportedTypeSyntax,
    DuplicateParameterName,
    UnknownIdentifier,
    UnknownFunction,
    DuplicateLocalName,
    TypeMismatch,
    InvalidMainSignature,
    MissingMain,
    InvalidConditionType,
    InvalidCall,
    UnknownEnumConstructor,
    UnknownEnumVariant,
    EnumConstructorArityMismatch,
    EnumConstructorTypeMismatch,
    UnknownEnumPattern,
    EnumPatternTypeMismatch,
    InvalidMirBlock,
    InvalidMirLocal,
    MissingTerminator,
    InvalidMirType,
    InvalidMirOperand,
    UnsupportedCBackendType,
    EnumPayloadBindingArityMismatch,
    DuplicatePatternBinding,
    InvalidPatternBinding,
    IgnoredMustUseValue,
    TryOperandNotResult,
    TryOutsideResultFunction,
    TryResultTypeMismatch,
    UnknownDecideEnum,
    UnknownDecideVariant,
    DecideVariantHasPayload,
    DecideConditionNotBool,
    DecideScoreNotInt,
    DecideMissingUnconditionalArm,
    UnsafeCallRequiresUnsafe,
    AddressOfRequiresPlace,
    DerefRequiresPointer,
    DerefRequiresUnsafe,
    UnknownStructLiteralType,
    UnknownStructLiteralField,
    DuplicateStructLiteralField,
    MissingStructLiteralField,
    StructFieldInitializerTypeMismatch,
    FieldAccessNonStruct,
    UnknownFieldAccess,
    FieldAssignmentNonPlace,
    FieldAssignmentTypeMismatch,
    ConstrainedGenericInstantiationUnsupported,
    GenericTypeInferenceConflict,
    GenericTypeParameterUninferred,
    UnsupportedGenericInstantiation,
    DuplicateConceptTypeParameter,
    DuplicateConceptRequirement,
    InvalidConceptRequirement,
    MarkerConceptCannotHaveRequirements,
    NonMarkerConceptRequiresBody,
    DuplicateImpl,
    UnknownConcept,
    ConceptArityMismatch,
    InvalidImplTarget,
    MissingConceptRequirementImpl,
    InvalidConceptRequirementImplSignature,
    ExtraConceptImplFunction,
    DuplicateConceptImplFunction,
    MarkerConceptImplCannotHaveFunctions,
    UnsafeImplRequired,
    UnsafeImplNotAllowed,
    UnsatisfiedConceptConstraint,
    UnknownConceptConstraint,
    ConceptConstraintArityMismatch,
    UnsupportedConceptConstraint,
    AmbiguousConceptRequirementCall,
    UnknownConceptRequirementCall,
    InvalidConceptRequirementCall,
    CompileTimeUnsupportedExpression,
    CompileTimeUnsupportedStatement,
    CompileTimeTypeMismatch,
    CompileTimeEvaluationFailed,
    CompileTimeDivisionByZero,
    CompileTimeOverflow,
    CompileTimeFunctionRequired,
    CompileTimeFunctionUnsupportedBody,
    CompileTimeFunctionUnsupportedSignature,
    CompileTimeRecursionLimit,
    CompileTimeArgumentTypeMismatch,
    CompileTimeUnsupportedLocalType,
    CompileTimeMissingReturn,
    CompileTimeIfRequiresBool,
    CompileTimeUnboundLocal,
    CompileTimeAssignmentTypeMismatch,
    CompileTimeFuelExhausted,
    CompileTimeWhileRequiresBool,
    CompileTimeUnknownCapability,
    CompileTimeDuplicateCapability,
    CompileTimeCapabilityNotGranted,
    CompileTimeCapabilityListRequiresFunction,
    StaticAssertFailed,
    StaticAssertRequiresBool,

    pub fn format(self: DiagnosticCode) []const u8 {
        return switch (self) {
            .InvalidCharacter => "CON0001",
            .UnterminatedString => "CON0002",
            .UnterminatedBlockComment => "CON0006",
            .UnexpectedToken => "CON0003",
            .ExpectedItem => "CON0004",
            .DuplicateModuleDeclaration => "CON0005",
            .UnterminatedChar => "CON0007",
            .EmptyCharLiteral => "CON0008",
            .InvalidEscapeSequence => "CON0009",
            .InvalidExecutableSubset => "CON0010",
            .DuplicateTopLevelName => "CON0020",
            .UnknownTypeName => "CON0021",
            .DuplicateStructField => "CON0022",
            .DuplicateEnumVariant => "CON0023",
            .UnsupportedTypeSyntax => "CON0024",
            .DuplicateParameterName => "CON0025",
            .UnknownIdentifier => "CON0026",
            .UnknownFunction => "CON0027",
            .DuplicateLocalName => "CON0028",
            .TypeMismatch => "CON0029",
            .InvalidMainSignature => "CON0030",
            .MissingMain => "CON0031",
            .InvalidConditionType => "CON0032",
            .InvalidCall => "CON0033",
            .UnknownEnumConstructor => "CON0034",
            .UnknownEnumVariant => "CON0035",
            .EnumConstructorArityMismatch => "CON0036",
            .EnumConstructorTypeMismatch => "CON0037",
            .UnknownEnumPattern => "CON0038",
            .EnumPatternTypeMismatch => "CON0039",
            .InvalidMirBlock => "CON0040",
            .InvalidMirLocal => "CON0041",
            .MissingTerminator => "CON0042",
            .InvalidMirType => "CON0043",
            .InvalidMirOperand => "CON0044",
            .UnsupportedCBackendType => "CON0045",
            .EnumPayloadBindingArityMismatch => "CON0046",
            .DuplicatePatternBinding => "CON0047",
            .InvalidPatternBinding => "CON0048",
            .IgnoredMustUseValue => "CON0049",
            .TryOperandNotResult => "CON0050",
            .TryOutsideResultFunction => "CON0051",
            .TryResultTypeMismatch => "CON0052",
            .UnknownDecideEnum => "CON0060",
            .UnknownDecideVariant => "CON0061",
            .DecideVariantHasPayload => "CON0062",
            .DecideConditionNotBool => "CON0063",
            .DecideScoreNotInt => "CON0064",
            .DecideMissingUnconditionalArm => "CON0065",
            .UnsafeCallRequiresUnsafe => "CON0070",
            .AddressOfRequiresPlace => "CON0071",
            .DerefRequiresPointer => "CON0072",
            .DerefRequiresUnsafe => "CON0073",
            .UnknownStructLiteralType => "CON0074",
            .UnknownStructLiteralField => "CON0075",
            .DuplicateStructLiteralField => "CON0076",
            .MissingStructLiteralField => "CON0077",
            .StructFieldInitializerTypeMismatch => "CON0078",
            .FieldAccessNonStruct => "CON0079",
            .UnknownFieldAccess => "CON0080",
            .FieldAssignmentNonPlace => "CON0081",
            .FieldAssignmentTypeMismatch => "CON0082",
            .ConstrainedGenericInstantiationUnsupported => "CON0087",
            .GenericTypeInferenceConflict => "CON0088",
            .GenericTypeParameterUninferred => "CON0089",
            .UnsupportedGenericInstantiation => "CON0090",
            .DuplicateConceptTypeParameter => "CON0091",
            .DuplicateConceptRequirement => "CON0092",
            .InvalidConceptRequirement => "CON0093",
            .MarkerConceptCannotHaveRequirements => "CON0094",
            .NonMarkerConceptRequiresBody => "CON0095",
            .DuplicateImpl => "CON0098",
            .UnknownConcept => "CON0099",
            .ConceptArityMismatch => "CON0100",
            .InvalidImplTarget => "CON0101",
            .MissingConceptRequirementImpl => "CON0102",
            .InvalidConceptRequirementImplSignature => "CON0103",
            .ExtraConceptImplFunction => "CON0104",
            .DuplicateConceptImplFunction => "CON0105",
            .MarkerConceptImplCannotHaveFunctions => "CON0106",
            .UnsafeImplRequired => "CON0107",
            .UnsafeImplNotAllowed => "CON0108",
            .UnsatisfiedConceptConstraint => "CON0109",
            .UnknownConceptConstraint => "CON0110",
            .ConceptConstraintArityMismatch => "CON0111",
            .UnsupportedConceptConstraint => "CON0112",
            .AmbiguousConceptRequirementCall => "CON0113",
            .UnknownConceptRequirementCall => "CON0114",
            .InvalidConceptRequirementCall => "CON0115",
            .CompileTimeUnsupportedExpression => "CON0120",
            .CompileTimeTypeMismatch => "CON0121",
            .CompileTimeEvaluationFailed => "CON0122",
            .CompileTimeDivisionByZero => "CON0123",
            .CompileTimeOverflow => "CON0124",
            .StaticAssertFailed => "CON0125",
            .StaticAssertRequiresBool => "CON0126",
            .CompileTimeFunctionRequired => "CON0127",
            .CompileTimeFunctionUnsupportedBody => "CON0128",
            .CompileTimeFunctionUnsupportedSignature => "CON0129",
            .CompileTimeRecursionLimit => "CON0130",
            .CompileTimeArgumentTypeMismatch => "CON0131",
            .CompileTimeUnsupportedStatement => "CON0132",
            .CompileTimeUnsupportedLocalType => "CON0133",
            .CompileTimeMissingReturn => "CON0134",
            .CompileTimeIfRequiresBool => "CON0135",
            .CompileTimeUnboundLocal => "CON0136",
            .CompileTimeAssignmentTypeMismatch => "CON0137",
            .CompileTimeFuelExhausted => "CON0138",
            .CompileTimeWhileRequiresBool => "CON0139",
            .CompileTimeUnknownCapability => "CON0140",
            .CompileTimeDuplicateCapability => "CON0141",
            .CompileTimeCapabilityNotGranted => "CON0142",
            .CompileTimeCapabilityListRequiresFunction => "CON0143",
        };
    }
};

pub const DiagnosticNote = struct {
    message: []const u8,
};

pub const Diagnostic = struct {
    code: DiagnosticCode,
    severity: Severity,
    message: []const u8,
    primary_span: SourceSpan,
    help: ?[]const u8 = null,
    notes: []const DiagnosticNote = &.{},
    owns_message: bool = false,

    pub fn init(
        code: DiagnosticCode,
        severity: Severity,
        message: []const u8,
        primary_span: SourceSpan,
    ) Diagnostic {
        return .{
            .code = code,
            .severity = severity,
            .message = message,
            .primary_span = primary_span,
        };
    }

    pub fn withHelp(self: Diagnostic, help: []const u8) Diagnostic {
        var diagnostic = self;
        diagnostic.help = help;
        return diagnostic;
    }

    pub fn deinit(self: Diagnostic, allocator: std.mem.Allocator) void {
        if (self.owns_message) {
            allocator.free(self.message);
        }
    }
};

pub const DiagnosticBag = struct {
    allocator: std.mem.Allocator,
    diagnostics: std.ArrayList(Diagnostic),

    pub fn init(allocator: std.mem.Allocator) DiagnosticBag {
        return .{ .allocator = allocator, .diagnostics = std.ArrayList(Diagnostic).empty };
    }

    pub fn deinit(self: *DiagnosticBag) void {
        for (self.diagnostics.items) |diagnostic| {
            diagnostic.deinit(self.allocator);
        }
        self.diagnostics.deinit(self.allocator);
    }

    pub fn append(self: *DiagnosticBag, diagnostic: Diagnostic) !void {
        try self.diagnostics.append(self.allocator, diagnostic);
    }

    pub fn count(self: DiagnosticBag) usize {
        return self.diagnostics.items.len;
    }

    pub fn hasErrors(self: DiagnosticBag) bool {
        for (self.diagnostics.items) |diagnostic| {
            if (diagnostic.severity == .@"error") return true;
        }
        return false;
    }

    pub fn clear(self: *DiagnosticBag) void {
        for (self.diagnostics.items) |diagnostic| {
            diagnostic.deinit(self.allocator);
        }
        self.diagnostics.clearRetainingCapacity();
    }
};

pub fn makeDiagnostic(
    code: DiagnosticCode,
    severity: Severity,
    message: []const u8,
    span: SourceSpan,
) Diagnostic {
    return Diagnostic.init(code, severity, message, span);
}

pub fn invalidCharacter(allocator: std.mem.Allocator, span: SourceSpan, byte: u8) !Diagnostic {
    return .{
        .code = .InvalidCharacter,
        .severity = .@"error",
        .message = try std.fmt.allocPrint(allocator, "unexpected character '{c}'", .{byte}),
        .primary_span = span,
        .help = "remove this character or use a valid token",
        .owns_message = true,
    };
}

pub fn unterminatedString(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnterminatedString,
        .@"error",
        "unterminated string literal",
        span,
    ).withHelp("add a closing quote before the end of the line or file");
}

pub fn unterminatedBlockComment(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnterminatedBlockComment,
        .@"error",
        "unterminated block comment",
        span,
    ).withHelp("add a closing */ before the end of the file");
}

pub fn unterminatedChar(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnterminatedChar,
        .@"error",
        "unterminated char literal",
        span,
    ).withHelp("add a closing quote before the end of the line or file");
}

pub fn emptyCharLiteral(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .EmptyCharLiteral,
        .@"error",
        "empty char literal",
        span,
    ).withHelp("put one character or escape sequence between the quotes");
}

pub fn invalidEscapeSequence(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .InvalidEscapeSequence,
        .@"error",
        "invalid escape sequence",
        span,
    ).withHelp("use a recognized escape sequence");
}

pub fn duplicateConceptTypeParameter(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateConceptTypeParameter, .@"error", "duplicate concept type parameter", span);
}

pub fn duplicateConceptRequirement(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateConceptRequirement, .@"error", "duplicate concept requirement", span);
}

pub fn invalidConceptRequirement(span: SourceSpan, message: []const u8) Diagnostic {
    return Diagnostic.init(.InvalidConceptRequirement, .@"error", message, span);
}

pub fn markerConceptCannotHaveRequirements(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MarkerConceptCannotHaveRequirements, .@"error", "marker concept cannot have requirements", span);
}

pub fn nonMarkerConceptRequiresBody(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.NonMarkerConceptRequiresBody, .@"error", "non-marker concept requires a requirement body", span);
}

pub fn duplicateImpl(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateImpl, .@"error", "duplicate concept impl", span);
}

pub fn unknownConcept(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnknownConcept, .@"error", "unknown concept", span);
}

pub fn conceptArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ConceptArityMismatch, .@"error", "concept impl type argument count does not match concept arity", span);
}

pub fn invalidImplTarget(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidImplTarget, .@"error", "invalid concept impl target", span);
}

pub fn missingConceptRequirementImpl(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MissingConceptRequirementImpl, .@"error", "missing concept requirement implementation", span);
}

pub fn invalidConceptRequirementImplSignature(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidConceptRequirementImplSignature, .@"error", "concept requirement implementation signature does not match", span);
}

pub fn extraConceptImplFunction(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ExtraConceptImplFunction, .@"error", "extra function in concept impl", span);
}

pub fn duplicateConceptImplFunction(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateConceptImplFunction, .@"error", "duplicate function in concept impl", span);
}

pub fn markerConceptImplCannotHaveFunctions(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MarkerConceptImplCannotHaveFunctions, .@"error", "marker concept impl cannot have functions", span);
}

pub fn unsafeImplRequired(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnsafeImplRequired, .@"error", "unsafe marker concept requires unsafe impl", span);
}

pub fn unsafeImplNotAllowed(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnsafeImplNotAllowed, .@"error", "unsafe impl is not allowed for safe concepts", span);
}

pub fn unsatisfiedConceptConstraint(allocator: std.mem.Allocator, span: SourceSpan, concept_name: []const u8, type_name: []const u8) !Diagnostic {
    return .{
        .code = .UnsatisfiedConceptConstraint,
        .severity = .@"error",
        .message = try std.fmt.allocPrint(allocator, "type '{s}' does not satisfy concept constraint '{s}'", .{ type_name, concept_name }),
        .primary_span = span,
        .owns_message = true,
    };
}

pub fn unknownConceptConstraint(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnknownConceptConstraint, .@"error", "unknown concept in template constraint", span);
}

pub fn conceptConstraintArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ConceptConstraintArityMismatch, .@"error", "concept constraint type argument count does not match concept arity", span);
}

pub fn unsupportedConceptConstraint(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnsupportedConceptConstraint, .@"error", "unsupported concept constraint; expected Concept<T>", span);
}

pub fn ambiguousConceptRequirementCall(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.AmbiguousConceptRequirementCall, .@"error", "ambiguous concept requirement call", span);
}

pub fn unknownConceptRequirementCall(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnknownConceptRequirementCall, .@"error", "unknown concept requirement call", span);
}

pub fn invalidConceptRequirementCall(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidConceptRequirementCall, .@"error", "invalid concept requirement call", span);
}

pub fn duplicateTopLevelName(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateTopLevelName,
        .@"error",
        "duplicate top-level declaration name",
        span,
    ).withHelp("top-level functions, structs, and enums share one namespace in Phase 3");
}

pub fn unknownTypeName(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownTypeName,
        .@"error",
        "unknown type name",
        span,
    ).withHelp("Phase 3 declaration types can name void, int, bool, top-level structs, or top-level enums");
}

pub fn duplicateStructField(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateStructField,
        .@"error",
        "duplicate struct field name",
        span,
    ).withHelp("field names must be unique within a struct");
}

pub fn duplicateEnumVariant(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateEnumVariant,
        .@"error",
        "duplicate enum variant name",
        span,
    ).withHelp("variant names must be unique within an enum");
}

pub fn unsupportedTypeSyntax(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnsupportedTypeSyntax,
        .@"error",
        "unsupported declaration type syntax",
        span,
    ).withHelp("Phase 3 supports only simple, non-generic, non-reference, non-pointer declaration type names");
}

pub fn duplicateParameterName(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateParameterName,
        .@"error",
        "duplicate function parameter name",
        span,
    ).withHelp("parameter names must be unique within a function declaration");
}

pub fn render(writer: anytype, source: SourceFile, diagnostic: Diagnostic) !void {
    const location = try source.spanStartLocation(diagnostic.primary_span);
    const line = lineSlice(source, location.line);
    const underline_width = underlineWidth(source, diagnostic.primary_span, location, line);

    try writer.print("{s} {s}: {s}\n", .{
        diagnostic.code.format(),
        diagnostic.severity.format(),
        diagnostic.message,
    });
    try writer.print("--> {s}:{d}:{d}\n", .{
        source.display_name,
        location.line,
        location.column,
    });
    try writer.writeAll("|\n");
    try writer.print("{d} | {s}\n", .{ location.line, line });
    try writer.writeAll("| ");
    try writeRepeatedByte(writer, ' ', location.column - 1);
    try writeRepeatedByte(writer, '^', underline_width);
    try writer.writeByte('\n');

    if (diagnostic.help) |help| {
        try writer.writeAll("|\n");
        try writer.print("help: {s}\n", .{help});
    }
}

fn lineSlice(source: SourceFile, line_number: usize) []const u8 {
    const line_index = line_number - 1;
    const line_start = source.line_starts[line_index];
    const next_line_start = if (line_index + 1 < source.line_starts.len)
        source.line_starts[line_index + 1]
    else
        source.text.len;

    var line_end = next_line_start;
    if (line_end > line_start and source.text[line_end - 1] == '\n') {
        line_end -= 1;
    }
    if (line_end > line_start and source.text[line_end - 1] == '\r') {
        line_end -= 1;
    }

    return source.text[line_start..line_end];
}

fn underlineWidth(source: SourceFile, span: SourceSpan, location: SourceLocation, line: []const u8) usize {
    _ = source;
    const line_remaining = if (location.column - 1 < line.len)
        line.len - (location.column - 1)
    else
        0;

    if (span.length == 0 or line_remaining == 0) return 1;
    return @max(@as(usize, 1), @min(span.length, line_remaining));
}

fn writeRepeatedByte(writer: anytype, byte: u8, count: usize) !void {
    var index: usize = 0;
    while (index < count) : (index += 1) {
        try writer.writeByte(byte);
    }
}

test "severity has stable string formatting" {
    try std.testing.expectEqualStrings("error", Severity.@"error".format());
    try std.testing.expectEqualStrings("warning", Severity.warning.format());
    try std.testing.expectEqualStrings("note", Severity.note.format());
    try std.testing.expectEqualStrings("help", Severity.help.format());
}

test "diagnostic code has stable string formatting" {
    try std.testing.expectEqualStrings("CON0001", DiagnosticCode.InvalidCharacter.format());
    try std.testing.expectEqualStrings("CON0002", DiagnosticCode.UnterminatedString.format());
    try std.testing.expectEqualStrings("CON0003", DiagnosticCode.UnexpectedToken.format());
    try std.testing.expectEqualStrings("CON0004", DiagnosticCode.ExpectedItem.format());
    try std.testing.expectEqualStrings("CON0005", DiagnosticCode.DuplicateModuleDeclaration.format());
    try std.testing.expectEqualStrings("CON0006", DiagnosticCode.UnterminatedBlockComment.format());
    try std.testing.expectEqualStrings("CON0007", DiagnosticCode.UnterminatedChar.format());
    try std.testing.expectEqualStrings("CON0008", DiagnosticCode.EmptyCharLiteral.format());
    try std.testing.expectEqualStrings("CON0009", DiagnosticCode.InvalidEscapeSequence.format());
    try std.testing.expectEqualStrings("CON0040", DiagnosticCode.InvalidMirBlock.format());
    try std.testing.expectEqualStrings("CON0041", DiagnosticCode.InvalidMirLocal.format());
    try std.testing.expectEqualStrings("CON0042", DiagnosticCode.MissingTerminator.format());
    try std.testing.expectEqualStrings("CON0043", DiagnosticCode.InvalidMirType.format());
    try std.testing.expectEqualStrings("CON0044", DiagnosticCode.InvalidMirOperand.format());
    try std.testing.expectEqualStrings("CON0049", DiagnosticCode.IgnoredMustUseValue.format());
    try std.testing.expectEqualStrings("CON0050", DiagnosticCode.TryOperandNotResult.format());
    try std.testing.expectEqualStrings("CON0051", DiagnosticCode.TryOutsideResultFunction.format());
    try std.testing.expectEqualStrings("CON0052", DiagnosticCode.TryResultTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0060", DiagnosticCode.UnknownDecideEnum.format());
    try std.testing.expectEqualStrings("CON0061", DiagnosticCode.UnknownDecideVariant.format());
    try std.testing.expectEqualStrings("CON0062", DiagnosticCode.DecideVariantHasPayload.format());
    try std.testing.expectEqualStrings("CON0063", DiagnosticCode.DecideConditionNotBool.format());
    try std.testing.expectEqualStrings("CON0064", DiagnosticCode.DecideScoreNotInt.format());
    try std.testing.expectEqualStrings("CON0065", DiagnosticCode.DecideMissingUnconditionalArm.format());
    try std.testing.expectEqualStrings("CON0070", DiagnosticCode.UnsafeCallRequiresUnsafe.format());
    try std.testing.expectEqualStrings("CON0071", DiagnosticCode.AddressOfRequiresPlace.format());
    try std.testing.expectEqualStrings("CON0072", DiagnosticCode.DerefRequiresPointer.format());
    try std.testing.expectEqualStrings("CON0073", DiagnosticCode.DerefRequiresUnsafe.format());
    try std.testing.expectEqualStrings("CON0074", DiagnosticCode.UnknownStructLiteralType.format());
    try std.testing.expectEqualStrings("CON0075", DiagnosticCode.UnknownStructLiteralField.format());
    try std.testing.expectEqualStrings("CON0076", DiagnosticCode.DuplicateStructLiteralField.format());
    try std.testing.expectEqualStrings("CON0077", DiagnosticCode.MissingStructLiteralField.format());
    try std.testing.expectEqualStrings("CON0078", DiagnosticCode.StructFieldInitializerTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0081", DiagnosticCode.FieldAssignmentNonPlace.format());
    try std.testing.expectEqualStrings("CON0082", DiagnosticCode.FieldAssignmentTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0120", DiagnosticCode.CompileTimeUnsupportedExpression.format());
    try std.testing.expectEqualStrings("CON0121", DiagnosticCode.CompileTimeTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0122", DiagnosticCode.CompileTimeEvaluationFailed.format());
    try std.testing.expectEqualStrings("CON0123", DiagnosticCode.CompileTimeDivisionByZero.format());
    try std.testing.expectEqualStrings("CON0124", DiagnosticCode.CompileTimeOverflow.format());
    try std.testing.expectEqualStrings("CON0125", DiagnosticCode.StaticAssertFailed.format());
    try std.testing.expectEqualStrings("CON0126", DiagnosticCode.StaticAssertRequiresBool.format());
    try std.testing.expectEqualStrings("CON0127", DiagnosticCode.CompileTimeFunctionRequired.format());
    try std.testing.expectEqualStrings("CON0128", DiagnosticCode.CompileTimeFunctionUnsupportedBody.format());
    try std.testing.expectEqualStrings("CON0129", DiagnosticCode.CompileTimeFunctionUnsupportedSignature.format());
    try std.testing.expectEqualStrings("CON0130", DiagnosticCode.CompileTimeRecursionLimit.format());
    try std.testing.expectEqualStrings("CON0131", DiagnosticCode.CompileTimeArgumentTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0132", DiagnosticCode.CompileTimeUnsupportedStatement.format());
    try std.testing.expectEqualStrings("CON0133", DiagnosticCode.CompileTimeUnsupportedLocalType.format());
    try std.testing.expectEqualStrings("CON0134", DiagnosticCode.CompileTimeMissingReturn.format());
    try std.testing.expectEqualStrings("CON0135", DiagnosticCode.CompileTimeIfRequiresBool.format());
    try std.testing.expectEqualStrings("CON0136", DiagnosticCode.CompileTimeUnboundLocal.format());
    try std.testing.expectEqualStrings("CON0137", DiagnosticCode.CompileTimeAssignmentTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0138", DiagnosticCode.CompileTimeFuelExhausted.format());
    try std.testing.expectEqualStrings("CON0139", DiagnosticCode.CompileTimeWhileRequiresBool.format());
    try std.testing.expectEqualStrings("CON0140", DiagnosticCode.CompileTimeUnknownCapability.format());
    try std.testing.expectEqualStrings("CON0141", DiagnosticCode.CompileTimeDuplicateCapability.format());
    try std.testing.expectEqualStrings("CON0142", DiagnosticCode.CompileTimeCapabilityNotGranted.format());
}

test "diagnostic bag counts diagnostics and detects errors" {
    var bag = DiagnosticBag.init(std.testing.allocator);
    defer bag.deinit();

    try std.testing.expectEqual(@as(usize, 0), bag.count());
    try std.testing.expect(!bag.hasErrors());

    try bag.append(makeDiagnostic(.ExpectedItem, .warning, "expected item", .{ .start = 0, .length = 1 }));
    try std.testing.expectEqual(@as(usize, 1), bag.count());
    try std.testing.expect(!bag.hasErrors());

    try bag.append(makeDiagnostic(.UnexpectedToken, .@"error", "unexpected token", .{ .start = 1, .length = 1 }));
    try std.testing.expectEqual(@as(usize, 2), bag.count());
    try std.testing.expect(bag.hasErrors());

    bag.clear();
    try std.testing.expectEqual(@as(usize, 0), bag.count());
    try std.testing.expect(!bag.hasErrors());
}

test "construct invalid character diagnostic" {
    const span = SourceSpan{ .start = 8, .length = 1 };
    const diagnostic = try invalidCharacter(std.testing.allocator, span, '@');
    defer diagnostic.deinit(std.testing.allocator);

    try std.testing.expectEqual(DiagnosticCode.InvalidCharacter, diagnostic.code);
    try std.testing.expectEqual(Severity.@"error", diagnostic.severity);
    try std.testing.expectEqualStrings("unexpected character '@'", diagnostic.message);
    try std.testing.expectEqual(span, diagnostic.primary_span);
    try std.testing.expectEqualStrings("remove this character or use a valid token", diagnostic.help.?);
}

test "render single-line diagnostic with caret" {
    const source = try SourceFile.init(
        std.testing.allocator,
        "tests/diagnostics/invalid_token.concept",
        "let x = @;\n",
    );
    defer source.deinit(std.testing.allocator);

    const diagnostic = try invalidCharacter(std.testing.allocator, .{ .start = 8, .length = 1 }, '@');
    defer diagnostic.deinit(std.testing.allocator);

    var output: std.Io.Writer.Allocating = .init(std.testing.allocator);
    defer output.deinit();
    try render(&output.writer, source, diagnostic);

    try std.testing.expectEqualStrings(
        \\CON0001 error: unexpected character '@'
        \\--> tests/diagnostics/invalid_token.concept:1:9
        \\|
        \\1 | let x = @;
        \\|         ^
        \\|
        \\help: remove this character or use a valid token
        \\
    , output.written());
}

test "render diagnostic with help text" {
    const source = try SourceFile.init(std.testing.allocator, "string.concept", "let s = \"unterminated");
    defer source.deinit(std.testing.allocator);

    const diagnostic = unterminatedString(.{ .start = 8, .length = 13 });

    var output: std.Io.Writer.Allocating = .init(std.testing.allocator);
    defer output.deinit();
    try render(&output.writer, source, diagnostic);

    try std.testing.expect(std.mem.indexOf(u8, output.written(), "help: add a closing quote before the end of the line or file") != null);
}

test "render EOF-adjacent span" {
    const source = try SourceFile.init(std.testing.allocator, "eof.concept", "let x = ");
    defer source.deinit(std.testing.allocator);

    const diagnostic = makeDiagnostic(
        .UnexpectedToken,
        .@"error",
        "unexpected end of file",
        .{ .start = source.len(), .length = 0 },
    );

    var output: std.Io.Writer.Allocating = .init(std.testing.allocator);
    defer output.deinit();
    try render(&output.writer, source, diagnostic);

    try std.testing.expectEqualStrings(
        \\CON0003 error: unexpected end of file
        \\--> eof.concept:1:9
        \\|
        \\1 | let x = 
        \\|         ^
        \\
    , output.written());
}

pub fn unknownDecideEnum(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownDecideEnum,
        .@"error",
        "decide target type must be an enum",
        span,
    ).withHelp("decide expressions must name a top-level enum type");
}

pub fn unknownDecideVariant(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownDecideVariant,
        .@"error",
        "unknown decide variant",
        span,
    ).withHelp("decide arms must name a variant declared by the target enum");
}

pub fn decideVariantHasPayload(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DecideVariantHasPayload,
        .@"error",
        "decide candidate variant must not have payload fields",
        span,
    );
}

pub fn decideConditionNotBool(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DecideConditionNotBool,
        .@"error",
        "decide arm condition must be bool",
        span,
    );
}

pub fn decideScoreNotInt(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DecideScoreNotInt,
        .@"error",
        "decide arm score must be int",
        span,
    );
}

pub fn decideMissingUnconditionalArm(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DecideMissingUnconditionalArm,
        .@"error",
        "decide expression requires at least one unconditional arm",
        span,
    );
}

pub fn unknownIdentifier(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownIdentifier,
        .@"error",
        "unknown identifier",
        span,
    ).withHelp("function bodies can reference visible parameters and locals");
}

pub fn unknownFunction(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownFunction,
        .@"error",
        "unknown function",
        span,
    ).withHelp("function calls must name a top-level function");
}

pub fn unknownEnumConstructor(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownEnumConstructor,
        .@"error",
        "unknown enum constructor",
        span,
    ).withHelp("enum constructors must use a top-level enum name");
}

pub fn unknownEnumVariant(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownEnumVariant,
        .@"error",
        "unknown enum variant",
        span,
    ).withHelp("qualified enum constructors must name a variant declared by that enum");
}

pub fn unknownEnumPattern(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownEnumPattern,
        .@"error",
        "unknown enum pattern",
        span,
    ).withHelp("enum variant patterns must use a top-level enum and one of its declared variants");
}

pub fn enumPatternTypeMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .EnumPatternTypeMismatch,
        .@"error",
        "enum pattern type does not match match scrutinee",
        span,
    );
}

pub fn enumConstructorArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .EnumConstructorArityMismatch,
        .@"error",
        "enum constructor argument count mismatch",
        span,
    );
}

pub fn enumConstructorTypeMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .EnumConstructorTypeMismatch,
        .@"error",
        "enum constructor argument type mismatch",
        span,
    );
}

pub fn enumPayloadBindingArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .EnumPayloadBindingArityMismatch,
        .@"error",
        "enum payload binding count does not match variant payload",
        span,
    ).withHelp("bind either no payload fields or exactly one name per payload field");
}

pub fn duplicatePatternBinding(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicatePatternBinding,
        .@"error",
        "duplicate pattern binding name",
        span,
    );
}

pub fn invalidPatternBinding(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .InvalidPatternBinding,
        .@"error",
        "invalid pattern binding",
        span,
    );
}

pub fn duplicateLocalName(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateLocalName,
        .@"error",
        "duplicate local variable name",
        span,
    ).withHelp("Phase 3 rejects local names that duplicate any visible parameter or local");
}

pub fn invalidMirBlock(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidMirBlock, .@"error", "invalid MIR block", span);
}

pub fn invalidMirLocal(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidMirLocal, .@"error", "invalid MIR local", span);
}

pub fn missingMirTerminator(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MissingTerminator, .@"error", "MIR block is missing terminator", span);
}

pub fn invalidMirType(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidMirType, .@"error", "invalid MIR type", span);
}

pub fn invalidMirOperand(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidMirOperand, .@"error", "invalid MIR operand", span);
}

pub fn unsupportedCBackendType(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnsupportedCBackendType,
        .@"error",
        "unsupported type in MIR C backend",
        span,
    ).withHelp("Phase 5 C backend v0 renders void, int, bool, and supported non-empty enums with int/bool payload fields");
}

pub fn unknownStructLiteralType(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnknownStructLiteralType, .@"error", "struct literal type must name a top-level struct", span);
}

pub fn unknownStructLiteralField(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnknownStructLiteralField, .@"error", "unknown struct literal field", span);
}

pub fn duplicateStructLiteralField(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateStructLiteralField, .@"error", "duplicate struct literal field", span);
}

pub fn missingStructLiteralField(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MissingStructLiteralField, .@"error", "missing struct literal field", span);
}

pub fn structFieldInitializerTypeMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.StructFieldInitializerTypeMismatch, .@"error", "struct field initializer type mismatch", span);
}
