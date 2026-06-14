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
    DuplicateModule,
    ModuleDeclarationRequired,
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
    DuplicateInterfaceRequirement,
    InterfaceRequiresRequirement,
    InvalidInterfaceImplTarget,
    MissingInterfaceRequirementImpl,
    InvalidInterfaceRequirementImplSignature,
    ExtraInterfaceImplFunction,
    DuplicateInterfaceImpl,
    DuplicateInterfaceImplFunction,
    DynCoercionRequiresImpl,
    DynCoercionRequiresPlace,
    UnknownInterfaceMethod,
    InterfaceCallArityMismatch,
    InterfaceCallTypeMismatch,
    InterfaceCallRequiresMutableDyn,
    DynRequiresInterface,
    DynRequiresBorrowedReference,
    InterfaceRuntimeUnsupported,
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
    CompileTimeUnknownTargetField,
    CompileTimeTargetMetadataRequiresCompileTime,
    CompileTimeTargetMetadataUnavailable,
    StaticAssertFailed,
    StaticAssertRequiresBool,
    UseBeforeInitialization,
    UseAfterMove,
    MoveRequiresPlace,
    PartialMoveUnsupported,
    ImplicitCopyRequiresCopy,
    MaybeMovedUse,
    MaybeUninitializedUse,
    InvalidDropImpl,
    AssignmentRequiresReplacement,
    UseOfPartiallyInitializedValue,
    ManualInitRequiresTypeArgument,
    ManualInitAssumeInitRequiresUnsafe,
    ManualInitInvalidOperation,
    DropParamUnsupported,
    InvalidAttribute,
    FactRequiresZeroArgFunction,
    TheoryRequiresInlineData,
    InlineDataArityMismatch,
    InlineDataTypeMismatch,
    TestAttributeOutsideTestFile,
    TestFunctionReturnTypeInvalid,
    InlineDataRequiresTheory,
    ConflictingTestAttributes,
    DuplicateTestAttribute,
    TestExpectationRequiresReason,
    TestReasonMustBeNonEmpty,
    TestIntrinsicOutsideTestFile,
    TestIntrinsicTypeMismatch,
    ExpectEqualUnsupportedType,
    TestIntrinsicArityMismatch,
    TestRelationUnsupported,
    TestRelationOutsideExpectThat,
    AllocationInNoAllocFunction,
    AllocationEffectMismatch,
    ArenaAllocDropTypeUnsupported,
    DuplicateAllocationEffect,
    AllocationEffectInvalidTarget,
    OpaqueAllocationTypeByValueUnsupported,
    ArenaAllocArityMismatch,
    ArenaAllocRequiresArenaPointer,
    ArenaAllocRequiresConcreteType,
    ArenaAllocationInComptimeUnsupported,
    ArenaResetDestroyRequiresArenaPointer,
    ArenaResetDestroyArityMismatch,
    ArenaResetDestroyTypeArgsUnsupported,
    ArenaResetDestroyInComptimeUnsupported,
    MachineRequiresState,
    DuplicateMachineState,
    UnknownMachineState,
    TransitionOutsideMachineState,
    MachineSemanticsNotImplemented,
    ExternCNotImplemented,
    UnsupportedCAbiType,
    ExternCRequiresFunctionDeclaration,
    ExternCFunctionCannotHaveBody,
    ExportCRequiresFunctionDefinition,
    ExportCFunctionCannotBeGeneric,
    DuplicateCAbiSymbol,
    ReprCInvalidTarget,
    ReprCUnsupportedFieldType,
    ReprCEmptyStructUnsupported,
    ExternUnsupportedAbi,
    UnsupportedReprAbi,
    VarargsUnsupported,

    pub fn format(self: DiagnosticCode) []const u8 {
        return switch (self) {
            .InvalidCharacter => "CON0001",
            .UnterminatedString => "CON0002",
            .UnterminatedBlockComment => "CON0006",
            .UnexpectedToken => "CON0003",
            .ExpectedItem => "CON0004",
            .DuplicateModuleDeclaration => "CON0005",
            .DuplicateModule => "CON0270",
            .ModuleDeclarationRequired => "CON0276",
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
            .DuplicateInterfaceRequirement => "CON0240",
            .InterfaceRequiresRequirement => "CON0241",
            .InvalidInterfaceImplTarget => "CON0244",
            .MissingInterfaceRequirementImpl => "CON0245",
            .InvalidInterfaceRequirementImplSignature => "CON0246",
            .ExtraInterfaceImplFunction => "CON0247",
            .DuplicateInterfaceImplFunction => "CON0248",
            .DynCoercionRequiresImpl => "CON0249",
            .DynCoercionRequiresPlace => "CON0250",
            .UnknownInterfaceMethod => "CON0251",
            .InterfaceCallArityMismatch => "CON0252",
            .InterfaceCallTypeMismatch => "CON0253",
            .InterfaceCallRequiresMutableDyn => "CON0254",
            .DynRequiresInterface => "CON0257",
            .DynRequiresBorrowedReference => "CON0258",
            .DuplicateInterfaceImpl => "CON0256",
            .InterfaceRuntimeUnsupported => "CON0255",
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
            .CompileTimeUnknownTargetField => "CON0144",
            .CompileTimeTargetMetadataRequiresCompileTime => "CON0145",
            .CompileTimeTargetMetadataUnavailable => "CON0146",
            .UseBeforeInitialization => "CON0150",
            .UseAfterMove => "CON0151",
            .MoveRequiresPlace => "CON0152",
            .PartialMoveUnsupported => "CON0153",
            .ImplicitCopyRequiresCopy => "CON0154",
            .MaybeMovedUse => "CON0155",
            .MaybeUninitializedUse => "CON0156",
            .InvalidDropImpl => "CON0157",
            .AssignmentRequiresReplacement => "CON0160",
            .UseOfPartiallyInitializedValue => "CON0161",
            .ManualInitRequiresTypeArgument => "CON0162",
            .ManualInitAssumeInitRequiresUnsafe => "CON0163",
            .ManualInitInvalidOperation => "CON0164",
            .DropParamUnsupported => "CON0165",
            .InvalidAttribute => "CON0172",
            .FactRequiresZeroArgFunction => "CON0173",
            .TheoryRequiresInlineData => "CON0174",
            .InlineDataArityMismatch => "CON0175",
            .InlineDataTypeMismatch => "CON0176",
            .TestAttributeOutsideTestFile => "CON0177",
            .TestFunctionReturnTypeInvalid => "CON0178",
            .InlineDataRequiresTheory => "CON0179",
            .ConflictingTestAttributes => "CON0180",
            .DuplicateTestAttribute => "CON0181",
            .TestExpectationRequiresReason => "CON0170",
            .TestReasonMustBeNonEmpty => "CON0171",
            .TestIntrinsicOutsideTestFile => "CON0182",
            .TestIntrinsicTypeMismatch => "CON0183",
            .ExpectEqualUnsupportedType => "CON0184",
            .TestIntrinsicArityMismatch => "CON0185",
            .TestRelationUnsupported => "CON0186",
            .TestRelationOutsideExpectThat => "CON0187",
            .AllocationInNoAllocFunction => "CON0190",
            .AllocationEffectMismatch => "CON0191",
            .ArenaAllocDropTypeUnsupported => "CON0194",
            .DuplicateAllocationEffect => "CON0197",
            .AllocationEffectInvalidTarget => "CON0198",
            .OpaqueAllocationTypeByValueUnsupported => "CON0199",
            .ArenaAllocArityMismatch => "CON0204",
            .ArenaAllocRequiresArenaPointer => "CON0202",
            .ArenaAllocRequiresConcreteType => "CON0203",
            .ArenaAllocationInComptimeUnsupported => "CON0201",
            .ArenaResetDestroyRequiresArenaPointer => "CON0205",
            .ArenaResetDestroyArityMismatch => "CON0206",
            .ArenaResetDestroyTypeArgsUnsupported => "CON0207",
            .ArenaResetDestroyInComptimeUnsupported => "CON0210",
            .MachineRequiresState => "CON0220",
            .DuplicateMachineState => "CON0221",
            .UnknownMachineState => "CON0222",
            .TransitionOutsideMachineState => "CON0223",
            .MachineSemanticsNotImplemented => "CON0231",
            .ExternCNotImplemented => "CON0259",
            .UnsupportedCAbiType => "CON0260",
            .ExternCRequiresFunctionDeclaration => "CON0261",
            .ExternCFunctionCannotHaveBody => "CON0262",
            .ExportCRequiresFunctionDefinition => "CON0263",
            .ExportCFunctionCannotBeGeneric => "CON0264",
            .DuplicateCAbiSymbol => "CON0265",
            .ReprCInvalidTarget => "CON0266",
            .ReprCUnsupportedFieldType => "CON0267",
            .ReprCEmptyStructUnsupported => "CON0268",
            .ExternUnsupportedAbi => "CON026A",
            .UnsupportedReprAbi => "CON026B",
            .VarargsUnsupported => "CON0269",
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

pub fn duplicateModule(allocator: std.mem.Allocator, name: []const u8, span: SourceSpan, previous_path: []const u8) !Diagnostic {
    return .{
        .code = .DuplicateModule,
        .severity = .@"error",
        .message = try std.fmt.allocPrint(allocator, "duplicate module '{s}'; previous declaration is in {s}", .{ name, previous_path }),
        .primary_span = span,
        .help = "module names must be unique in one compilation unit; v0 does not allow one module to span multiple files",
        .owns_message = true,
    };
}

pub fn moduleDeclarationRequired(allocator: std.mem.Allocator, path: []const u8, span: SourceSpan) !Diagnostic {
    return .{
        .code = .ModuleDeclarationRequired,
        .severity = .@"error",
        .message = try std.fmt.allocPrint(allocator, "module declaration required in multi-source file '{s}'", .{path}),
        .primary_span = span,
        .help = "Phase 16 v0 requires exactly one module declaration per file in multi-source fixtures",
        .owns_message = true,
    };
}

pub fn machineSemanticsNotImplemented(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .MachineSemanticsNotImplemented,
        .@"error",
        "this executable machine form is not implemented yet",
        span,
    ).withHelp("literal-transition machines are executable; match/decide transitions and broader machine body forms remain parsed, validated, and represented in HIR but are not lowered to executable C yet");
}

pub fn externCNotImplemented(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .ExternCNotImplemented,
        .@"error",
        "extern C declarations are parsed but not lowered yet",
        span,
    ).withHelp("Phase 15 M1 preserves extern C declarations in the AST; HIR, MIR, backend emission, linking, and calls are deferred");
}

pub fn unsupportedCAbiType(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnsupportedCAbiType,
        .@"error",
        "unsupported C ABI type",
        span,
    ).withHelp("Phase 15 M2 extern C declarations support void returns, int, bool, AllocError, and raw pointers to void, int, bool, Arena, Allocator, or AllocError only");
}

pub fn externCRequiresFunctionDeclaration(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .ExternCRequiresFunctionDeclaration,
        .@"error",
        "extern C block entries must be function declarations",
        span,
    ).withHelp("Phase 15 M1 supports only semicolon-terminated foreign function signatures inside extern \"C\" blocks");
}

pub fn externCFunctionCannotHaveBody(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .ExternCFunctionCannotHaveBody,
        .@"error",
        "extern C function declarations cannot have bodies",
        span,
    ).withHelp("declare the foreign function signature with ';' and provide the implementation in C or another linked object");
}

pub fn exportCRequiresFunctionDefinition(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .ExportCRequiresFunctionDefinition,
        .@"error",
        "export C requires a function definition",
        span,
    ).withHelp("Phase 15 M4 supports only export \"C\" on function definitions with bodies");
}

pub fn duplicateCAbiSymbol(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateCAbiSymbol,
        .@"error",
        "duplicate C ABI symbol",
        span,
    ).withHelp("Phase 15 v0 rejects duplicate C ABI declarations/exports by declared symbol name in one module");
}

pub fn externUnsupportedAbi(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .ExternUnsupportedAbi,
        .@"error",
        "unsupported extern ABI",
        span,
    ).withHelp("Phase 15 M1 supports only extern \"C\" block declarations; C++ interop remains deferred");
}

pub fn varargsUnsupported(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .VarargsUnsupported,
        .@"error",
        "varargs are not supported in extern C declarations",
        span,
    ).withHelp("declare fixed-arity C functions only; varargs are deferred beyond Phase 15 v0");
}

pub fn dynRequiresInterface(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DynRequiresInterface,
        .@"error",
        "dyn target must be an interface",
        span,
    ).withHelp("Phase 14 M4 supports only borrowed dyn interface references such as dyn Writer&");
}

pub fn dynRequiresBorrowedReference(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DynRequiresBorrowedReference,
        .@"error",
        "dyn requires a borrowed interface reference",
        span,
    ).withHelp("use dyn Interface& or mut dyn Interface&; by-value dyn and raw dyn pointers are not supported in Phase 14 M4");
}

pub fn machineRequiresState(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .MachineRequiresState,
        .@"error",
        "machine must declare at least one state",
        span,
    ).withHelp("declare the v0 initial state as the first state in the machine body");
}

pub fn duplicateMachineState(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DuplicateMachineState,
        .@"error",
        "duplicate machine state name",
        span,
    ).withHelp("state names are scoped to one machine and must be unique within that machine");
}

pub fn unknownMachineState(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .UnknownMachineState,
        .@"error",
        "unknown machine state",
        span,
    ).withHelp("transition targets must name a state declared in the same machine");
}

pub fn transitionOutsideMachineState(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .TransitionOutsideMachineState,
        .@"error",
        "transition statement outside machine state",
        span,
    ).withHelp("place transition statements inside a machine state body");
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

pub fn duplicateInterfaceRequirement(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateInterfaceRequirement, .@"error", "duplicate interface requirement", span);
}

pub fn interfaceRequiresRequirement(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .InterfaceRequiresRequirement,
        .@"error",
        "interface requires at least one requirement",
        span,
    ).withHelp("Phase 14 v0 rejects empty runtime interfaces; use a marker concept for static marker behavior");
}

pub fn invalidInterfaceImplTarget(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidInterfaceImplTarget, .@"error", "invalid interface impl target", span);
}

pub fn missingInterfaceRequirementImpl(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MissingInterfaceRequirementImpl, .@"error", "missing interface requirement implementation", span);
}

pub fn invalidInterfaceRequirementImplSignature(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidInterfaceRequirementImplSignature, .@"error", "interface requirement implementation signature does not match", span);
}

pub fn extraInterfaceImplFunction(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ExtraInterfaceImplFunction, .@"error", "extra function in interface impl", span);
}

pub fn duplicateInterfaceImpl(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateInterfaceImpl, .@"error", "duplicate interface impl", span);
}

pub fn duplicateInterfaceImplFunction(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateInterfaceImplFunction, .@"error", "duplicate function in interface impl", span);
}

pub fn dynCoercionRequiresImpl(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DynCoercionRequiresImpl,
        .@"error",
        "dyn coercion requires an interface impl for the concrete type",
        span,
    ).withHelp("add an impl Interface<ConcreteType> or pass an existing dyn interface reference");
}

pub fn dynCoercionRequiresPlace(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .DynCoercionRequiresPlace,
        .@"error",
        "dyn coercion requires an addressable concrete place",
        span,
    ).withHelp("borrowed dyn references cannot be formed from temporaries in Phase 14 M5");
}

pub fn unknownInterfaceMethod(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnknownInterfaceMethod, .@"error", "unknown interface method", span);
}

pub fn interfaceCallArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InterfaceCallArityMismatch, .@"error", "interface method call argument count mismatch", span);
}

pub fn interfaceCallTypeMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InterfaceCallTypeMismatch, .@"error", "interface method call argument type mismatch", span);
}

pub fn interfaceCallRequiresMutableDyn(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .InterfaceCallRequiresMutableDyn,
        .@"error",
        "interface method call requires a mutable dyn reference",
        span,
    ).withHelp("Phase 14 M6 requires mut dyn Interface& because interface receiver mutability is not modeled on requirements yet");
}

pub fn interfaceRuntimeUnsupported(span: SourceSpan) Diagnostic {
    return Diagnostic.init(
        .InterfaceRuntimeUnsupported,
        .@"error",
        "interface runtime use is not implemented yet",
        span,
    ).withHelp("Phase 14 M3 validates interface declarations and impl conformance, but dyn references, interface values, vtables, and calls remain deferred");
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
    try std.testing.expectEqualStrings("CON0143", DiagnosticCode.CompileTimeCapabilityListRequiresFunction.format());
    try std.testing.expectEqualStrings("CON0144", DiagnosticCode.CompileTimeUnknownTargetField.format());
    try std.testing.expectEqualStrings("CON0145", DiagnosticCode.CompileTimeTargetMetadataRequiresCompileTime.format());
    try std.testing.expectEqualStrings("CON0146", DiagnosticCode.CompileTimeTargetMetadataUnavailable.format());
    try std.testing.expectEqualStrings("CON0150", DiagnosticCode.UseBeforeInitialization.format());
    try std.testing.expectEqualStrings("CON0151", DiagnosticCode.UseAfterMove.format());
    try std.testing.expectEqualStrings("CON0152", DiagnosticCode.MoveRequiresPlace.format());
    try std.testing.expectEqualStrings("CON0153", DiagnosticCode.PartialMoveUnsupported.format());
    try std.testing.expectEqualStrings("CON0154", DiagnosticCode.ImplicitCopyRequiresCopy.format());
    try std.testing.expectEqualStrings("CON0155", DiagnosticCode.MaybeMovedUse.format());
    try std.testing.expectEqualStrings("CON0156", DiagnosticCode.MaybeUninitializedUse.format());
    try std.testing.expectEqualStrings("CON0157", DiagnosticCode.InvalidDropImpl.format());
    try std.testing.expectEqualStrings("CON0160", DiagnosticCode.AssignmentRequiresReplacement.format());
    try std.testing.expectEqualStrings("CON0161", DiagnosticCode.UseOfPartiallyInitializedValue.format());
    try std.testing.expectEqualStrings("CON0162", DiagnosticCode.ManualInitRequiresTypeArgument.format());
    try std.testing.expectEqualStrings("CON0163", DiagnosticCode.ManualInitAssumeInitRequiresUnsafe.format());
    try std.testing.expectEqualStrings("CON0164", DiagnosticCode.ManualInitInvalidOperation.format());
    try std.testing.expectEqualStrings("CON0165", DiagnosticCode.DropParamUnsupported.format());
    try std.testing.expectEqualStrings("CON0240", DiagnosticCode.DuplicateInterfaceRequirement.format());
    try std.testing.expectEqualStrings("CON0241", DiagnosticCode.InterfaceRequiresRequirement.format());
    try std.testing.expectEqualStrings("CON0244", DiagnosticCode.InvalidInterfaceImplTarget.format());
    try std.testing.expectEqualStrings("CON0245", DiagnosticCode.MissingInterfaceRequirementImpl.format());
    try std.testing.expectEqualStrings("CON0246", DiagnosticCode.InvalidInterfaceRequirementImplSignature.format());
    try std.testing.expectEqualStrings("CON0247", DiagnosticCode.ExtraInterfaceImplFunction.format());
    try std.testing.expectEqualStrings("CON0248", DiagnosticCode.DuplicateInterfaceImplFunction.format());
    try std.testing.expectEqualStrings("CON0249", DiagnosticCode.DynCoercionRequiresImpl.format());
    try std.testing.expectEqualStrings("CON0250", DiagnosticCode.DynCoercionRequiresPlace.format());
    try std.testing.expectEqualStrings("CON0251", DiagnosticCode.UnknownInterfaceMethod.format());
    try std.testing.expectEqualStrings("CON0252", DiagnosticCode.InterfaceCallArityMismatch.format());
    try std.testing.expectEqualStrings("CON0253", DiagnosticCode.InterfaceCallTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0254", DiagnosticCode.InterfaceCallRequiresMutableDyn.format());
    try std.testing.expectEqualStrings("CON0255", DiagnosticCode.InterfaceRuntimeUnsupported.format());
    try std.testing.expectEqualStrings("CON0256", DiagnosticCode.DuplicateInterfaceImpl.format());
    try std.testing.expectEqualStrings("CON0257", DiagnosticCode.DynRequiresInterface.format());
    try std.testing.expectEqualStrings("CON0258", DiagnosticCode.DynRequiresBorrowedReference.format());
    try std.testing.expectEqualStrings("CON0259", DiagnosticCode.ExternCNotImplemented.format());
    try std.testing.expectEqualStrings("CON0261", DiagnosticCode.ExternCRequiresFunctionDeclaration.format());
    try std.testing.expectEqualStrings("CON0262", DiagnosticCode.ExternCFunctionCannotHaveBody.format());
    try std.testing.expectEqualStrings("CON0269", DiagnosticCode.VarargsUnsupported.format());
    try std.testing.expectEqualStrings("CON026A", DiagnosticCode.ExternUnsupportedAbi.format());
    try std.testing.expectEqualStrings("CON0172", DiagnosticCode.InvalidAttribute.format());
    try std.testing.expectEqualStrings("CON0173", DiagnosticCode.FactRequiresZeroArgFunction.format());
    try std.testing.expectEqualStrings("CON0174", DiagnosticCode.TheoryRequiresInlineData.format());
    try std.testing.expectEqualStrings("CON0175", DiagnosticCode.InlineDataArityMismatch.format());
    try std.testing.expectEqualStrings("CON0176", DiagnosticCode.InlineDataTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0177", DiagnosticCode.TestAttributeOutsideTestFile.format());
    try std.testing.expectEqualStrings("CON0178", DiagnosticCode.TestFunctionReturnTypeInvalid.format());
    try std.testing.expectEqualStrings("CON0179", DiagnosticCode.InlineDataRequiresTheory.format());
    try std.testing.expectEqualStrings("CON0180", DiagnosticCode.ConflictingTestAttributes.format());
    try std.testing.expectEqualStrings("CON0181", DiagnosticCode.DuplicateTestAttribute.format());
    try std.testing.expectEqualStrings("CON0170", DiagnosticCode.TestExpectationRequiresReason.format());
    try std.testing.expectEqualStrings("CON0171", DiagnosticCode.TestReasonMustBeNonEmpty.format());
    try std.testing.expectEqualStrings("CON0182", DiagnosticCode.TestIntrinsicOutsideTestFile.format());
    try std.testing.expectEqualStrings("CON0183", DiagnosticCode.TestIntrinsicTypeMismatch.format());
    try std.testing.expectEqualStrings("CON0184", DiagnosticCode.ExpectEqualUnsupportedType.format());
    try std.testing.expectEqualStrings("CON0185", DiagnosticCode.TestIntrinsicArityMismatch.format());
    try std.testing.expectEqualStrings("CON0186", DiagnosticCode.TestRelationUnsupported.format());
    try std.testing.expectEqualStrings("CON0187", DiagnosticCode.TestRelationOutsideExpectThat.format());
    try std.testing.expectEqualStrings("CON0190", DiagnosticCode.AllocationInNoAllocFunction.format());
    try std.testing.expectEqualStrings("CON0191", DiagnosticCode.AllocationEffectMismatch.format());
    try std.testing.expectEqualStrings("CON0194", DiagnosticCode.ArenaAllocDropTypeUnsupported.format());
    try std.testing.expectEqualStrings("CON0197", DiagnosticCode.DuplicateAllocationEffect.format());
    try std.testing.expectEqualStrings("CON0198", DiagnosticCode.AllocationEffectInvalidTarget.format());
    try std.testing.expectEqualStrings("CON0199", DiagnosticCode.OpaqueAllocationTypeByValueUnsupported.format());
    try std.testing.expectEqualStrings("CON0201", DiagnosticCode.ArenaAllocationInComptimeUnsupported.format());
    try std.testing.expectEqualStrings("CON0202", DiagnosticCode.ArenaAllocRequiresArenaPointer.format());
    try std.testing.expectEqualStrings("CON0203", DiagnosticCode.ArenaAllocRequiresConcreteType.format());
    try std.testing.expectEqualStrings("CON0204", DiagnosticCode.ArenaAllocArityMismatch.format());
    try std.testing.expectEqualStrings("CON0205", DiagnosticCode.ArenaResetDestroyRequiresArenaPointer.format());
    try std.testing.expectEqualStrings("CON0206", DiagnosticCode.ArenaResetDestroyArityMismatch.format());
    try std.testing.expectEqualStrings("CON0207", DiagnosticCode.ArenaResetDestroyTypeArgsUnsupported.format());
    try std.testing.expectEqualStrings("CON0210", DiagnosticCode.ArenaResetDestroyInComptimeUnsupported.format());
    try std.testing.expectEqualStrings("CON0220", DiagnosticCode.MachineRequiresState.format());
    try std.testing.expectEqualStrings("CON0221", DiagnosticCode.DuplicateMachineState.format());
    try std.testing.expectEqualStrings("CON0222", DiagnosticCode.UnknownMachineState.format());
    try std.testing.expectEqualStrings("CON0223", DiagnosticCode.TransitionOutsideMachineState.format());
    try std.testing.expectEqualStrings("CON0231", DiagnosticCode.MachineSemanticsNotImplemented.format());
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

pub fn useBeforeInitialization(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UseBeforeInitialization, .@"error", "use of uninitialized storage", span);
}

pub fn useAfterMove(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UseAfterMove, .@"error", "use of moved storage", span);
}

pub fn moveRequiresPlace(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MoveRequiresPlace, .@"error", "move requires a whole local or parameter place", span);
}

pub fn partialMoveUnsupported(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.PartialMoveUnsupported, .@"error", "field and partial moves are not supported yet", span);
}

pub fn implicitCopyRequiresCopy(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ImplicitCopyRequiresCopy, .@"error", "implicit copy of non-Copy value requires Copy<T> or explicit move", span);
}

pub fn maybeMovedUse(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MaybeMovedUse, .@"error", "value may have been moved on some control-flow paths", span);
}

pub fn maybeUninitializedUse(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.MaybeUninitializedUse, .@"error", "value may be uninitialized on some control-flow paths", span);
}

pub fn invalidDropImpl(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InvalidDropImpl, .@"error", "invalid Drop<T> impl", span);
}

pub fn assignmentRequiresReplacement(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.AssignmentRequiresReplacement, .@"error", "assignment would replace a non-Copy or Drop value; explicit replacement semantics are not implemented yet", span);
}

pub fn manualInitRequiresTypeArgument(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ManualInitRequiresTypeArgument, .@"error", "ManualInit requires exactly one type argument", span);
}

pub fn manualInitAssumeInitRequiresUnsafe(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ManualInitAssumeInitRequiresUnsafe, .@"error", "manualAssumeInit requires unsafe context", span);
}

pub fn manualInitInvalidOperation(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ManualInitInvalidOperation, .@"error", "invalid ManualInit operation", span);
}

pub fn dropParamUnsupported(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DropParamUnsupported, .@"error", "by-value parameter with Drop<T> is not supported until parameter cleanup is implemented", span);
}

pub fn invalidAttribute(span: SourceSpan, message: []const u8) Diagnostic {
    return Diagnostic.init(.InvalidAttribute, .@"error", message, span);
}

pub fn reprCInvalidTarget(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ReprCInvalidTarget, .@"error", "[Repr(C)] may only be applied to struct declarations", span);
}

pub fn unsupportedReprAbi(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UnsupportedReprAbi, .@"error", "unsupported repr ABI", span).withHelp("Phase 15 M5 supports only [Repr(C)] on structs; packed, custom alignment, repr enums, and other ABI strings are deferred");
}

pub fn factRequiresZeroArgFunction(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.FactRequiresZeroArgFunction, .@"error", "[Fact] test functions must not have parameters", span);
}

pub fn theoryRequiresInlineData(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TheoryRequiresInlineData, .@"error", "[Theory] test functions require at least one [InlineData] row", span);
}

pub fn inlineDataArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InlineDataArityMismatch, .@"error", "[InlineData] argument count must match test function parameter count", span);
}

pub fn inlineDataTypeMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InlineDataTypeMismatch, .@"error", "[InlineData] argument type does not match test function parameter type", span);
}

pub fn testAttributeOutsideTestFile(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestAttributeOutsideTestFile, .@"error", "test attributes are only valid in .con_test files", span);
}

pub fn testFunctionReturnTypeInvalid(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestFunctionReturnTypeInvalid, .@"error", "test functions must return void", span);
}

pub fn inlineDataRequiresTheory(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.InlineDataRequiresTheory, .@"error", "[InlineData] is only valid on [Theory] functions", span);
}

pub fn conflictingTestAttributes(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ConflictingTestAttributes, .@"error", "test function cannot be both [Fact] and [Theory]", span);
}

pub fn duplicateTestAttribute(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.DuplicateTestAttribute, .@"error", "duplicate test attribute", span);
}

pub fn testExpectationRequiresReason(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestExpectationRequiresReason, .@"error", "Assert/Expect test intrinsics require a because reason", span);
}

pub fn testReasonMustBeNonEmpty(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestReasonMustBeNonEmpty, .@"error", "test intrinsic because reason must be non-empty", span);
}

pub fn testIntrinsicOutsideTestFile(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestIntrinsicOutsideTestFile, .@"error", "Assert/Expect test intrinsics are only valid in .con_test files", span);
}

pub fn testIntrinsicTypeMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestIntrinsicTypeMismatch, .@"error", "test intrinsic argument type mismatch", span);
}

pub fn expectEqualUnsupportedType(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.ExpectEqualUnsupportedType, .@"error", "Expect.Equal supports only matching int or bool operands in v0", span);
}

pub fn testIntrinsicArityMismatch(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestIntrinsicArityMismatch, .@"error", "test intrinsic argument count mismatch", span);
}

pub fn testRelationUnsupported(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestRelationUnsupported, .@"error", "unsupported test relation for Phase 11 v0", span);
}

pub fn testRelationOutsideExpectThat(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.TestRelationOutsideExpectThat, .@"error", "test relation constructors are only supported inside Expect.That in Phase 11 v0", span);
}

pub fn useOfPartiallyInitializedValue(span: SourceSpan) Diagnostic {
    return Diagnostic.init(.UseOfPartiallyInitializedValue, .@"error", "value is only partially initialized; initialize all fields before using it as a whole", span);
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
