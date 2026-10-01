package concept

import (
	"fmt"
	"strconv"
	"strings"
)

const evt1NumericCastErrorName = "NumericCastError"

func evt1BuiltinNumericCastErrorEnum() EnumDecl {
	return EnumDecl{Name: evt1NumericCastErrorName, Variants: []VariantDecl{
		{Name: "NotFinite", Tag: 0}, {Name: "OutOfRange", Tag: 1},
	}}
}

func evt1IsNumericRoundOperation(name string) bool {
	switch name {
	case "TruncTo", "FloorTo", "CeilTo", "RoundTo":
		return true
	default:
		return false
	}
}

func evt1NumericRoundResultType(target Type, span Span) Type {
	return Type{Name: "Result", Kind: TypeApplied, TypeArgs: []Type{target,
		{Name: evt1NumericCastErrorName, Kind: TypeEnum, Span: span}}, Span: span}
}

func evt1ValidateNumericRoundCall(env *semanticEnv, scope *evt1Scope, call *TemplateCallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if len(call.TypeArgs) != 1 || len(call.Args) != 1 {
		return Type{}, evt1Diagnostic("NUMERIC_ROUND_ARITY", call.Callee+" requires one integer target and one floating argument", call.Span)
	}
	typeParams := ""
	if templateInfo != nil {
		typeParams = evt1TemplateParameterSet(templateInfo.Decl.Parameters)
	}
	if err := validateKnownType(env, call.TypeArg, call.Span, typeParams, false); err != nil {
		return Type{}, err
	}
	call.TypeArg = evt1CanonicalType(env, call.TypeArg)
	targetOpen := templateInfo != nil && evt1TypeDependsOnAnyParameter(call.TypeArg, templateInfo.Decl.Parameters)
	if !targetOpen && (!evt1IntegralRepresentation(call.TypeArg) || call.TypeArg.Quantity != nil) {
		return Type{}, evt1Diagnostic("NUMERIC_ROUND_TARGET", call.Callee+" target must be an unqualified integer representation", call.Span)
	}
	source, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	sourceOpen := templateInfo != nil && evt1TypeDependsOnAnyParameter(source, templateInfo.Decl.Parameters)
	// The intrinsic fixes its rounding and Result shape in an open body. Every
	// closed instance revalidates representation legality after substitution,
	// just as an explicit numeric cast does; no numeric fact is granted here.
	if !sourceOpen && (!evt1IsFloating(source) || source.Quantity != nil) {
		return Type{}, evt1Diagnostic("NUMERIC_ROUND_SOURCE", call.Callee+" requires an unqualified floating source", call.Span)
	}
	call.ResolvedType = evt1NumericRoundResultType(call.TypeArg, call.Span)
	return call.ResolvedType, nil
}

func evt1MIRUsesNumericRound(mir MIR) bool {
	uses := func(ops []MIROperation) bool {
		for _, op := range ops {
			if op.Kind == "float_round_to_integer" {
				return true
			}
		}
		return false
	}
	for _, fn := range mir.Functions {
		if uses(fn.Operations) {
			return true
		}
	}
	for _, instance := range mir.Instances {
		if uses(instance.Operations) {
			return true
		}
	}
	for _, template := range mir.Templates {
		if uses(template.Operations) {
			return true
		}
	}
	return false
}

func evt1ExactConversionFact(source, target Type) semanticFactResult {
	result := semanticFactResult{Outcome: FactDisproven, Origin: FactOriginCompilerAnalysis,
		Evidence: SemanticFactEvidence{Detail: "conversion is not exact for every source value"}}
	if !evt1NumericRepresentation(source) || !evt1NumericRepresentation(target) {
		return result
	}
	if (source.Quantity == nil) != (target.Quantity == nil) {
		return result
	}
	if source.Quantity != nil && !source.Quantity.Equal(*target.Quantity) {
		return result
	}
	if source.Equal(target) {
		result.Outcome = FactProven
		result.Evidence.Detail = "identical numeric representation and unit"
		return result
	}
	if sourceFloat, yes := evt1FloatRepresentationInfo(source); yes {
		if targetFloat, yes := evt1FloatRepresentationInfo(target); yes && targetFloat.MantissaBits >= sourceFloat.MantissaBits && targetFloat.ExponentBits >= sourceFloat.ExponentBits {
			result.Outcome = FactProven
			result.Evidence.Detail = "target floating format includes source precision and exponent range"
		}
		return result
	}
	sourceSigned, sourceMax, sourceMin, sourceWidth, _ := evt1IntegerTypeRange(source)
	if targetFloat, yes := evt1FloatRepresentationInfo(target); yes {
		precision := targetFloat.MantissaBits + 1
		needed := sourceWidth
		if sourceSigned {
			needed--
		}
		// binary16's finite range ends below 2^16. Wider formats cover all
		// current integer representations when their precision suffices.
		if precision >= needed && (targetFloat.Bits != 16 || needed <= 15) {
			result.Outcome = FactProven
			result.Evidence.Detail = fmt.Sprintf("%d-bit integer values fit %d-bit floating precision and exponent range", sourceWidth, precision)
		}
		return result
	}
	targetSigned, targetMax, targetMin, _, ok := evt1IntegerTypeRange(target)
	if !ok {
		return result
	}
	if (!sourceSigned || targetSigned) && targetMax >= sourceMax && (!sourceSigned || targetMin >= sourceMin) {
		result.Outcome = FactProven
		result.Evidence.Detail = "target integer range contains every source value"
	}
	return result
}

func evt1EvalNumericCast(value Value, target Type, span Span) (Value, error) {
	if value.Type.Quantity != nil || target.Quantity != nil {
		if value.Type.Quantity == nil || target.Quantity == nil || !value.Type.Quantity.SameDimension(*target.Quantity) {
			return Value{}, evt1Diagnostic("CAST_QUANTITY_SEMANTICS", "as cannot attach, erase, or change a quantity dimension", span)
		}
		if !value.Type.Quantity.Equal(*target.Quantity) {
			if !evt1IsFloating(target) {
				return Value{}, evt1Diagnostic("CAST_QUANTITY_SEMANTICS", "scaled integer quantity cast requires an explicit exactness policy", span)
			}
			n, d, ok := value.Type.Quantity.ScaleRatioToChecked(*target.Quantity)
			if !ok {
				return Value{}, evt1Diagnostic("QUANTITY_SCALE_OVERFLOW", "exact unit scale ratio exceeds the bounded compile-time rational range", span)
			}
			var numeric float64
			if value.Kind == ValueInt {
				numeric = float64(value.IntValue)
			} else if value.Kind == ValueFloat {
				numeric = value.FloatValue
			} else {
				return Value{}, evt1Diagnostic("CAST_COMPTIME_UNSUPPORTED", "quantity cast requires a numeric value", span)
			}
			numeric = numeric * float64(n) / float64(d)
			if target.Name == "float" {
				numeric = float64(float32(numeric))
			}
			return Value{Kind: ValueFloat, Type: target, FloatValue: numeric}, nil
		}
		unqualifiedSource, unqualifiedTarget := value.Type, target
		unqualifiedSource.Quantity, unqualifiedTarget.Quantity = nil, nil
		value.Type = unqualifiedSource
		converted, err := evt1EvalNumericCast(value, unqualifiedTarget, span)
		converted.Type = target
		return converted, err
	}
	if value.Kind == ValueFloat && evt1IntegralRepresentation(target) {
		return Value{}, evt1Diagnostic("FLOAT_TO_INT_ROUNDING_REQUIRED", "floating-point to integer conversion requires a named rounding operation", span)
	}
	if evt1IntegralRepresentation(target) {
		if value.Kind != ValueInt {
			return Value{}, evt1Diagnostic("CAST_COMPTIME_UNSUPPORTED", "comptime integer cast requires an integer value", span)
		}
		signed, max, min, _, _ := evt1IntegerTypeRange(target)
		n := int64(value.IntValue)
		if n < 0 && (!signed || uint64(-(n+1))+1 > min) || n >= 0 && uint64(n) > max {
			return Value{}, evt1Diagnostic("INTEGER_CAST_OUT_OF_RANGE", "integer cast is out of range at comptime", span)
		}
		value.Type = target
		return value, nil
	}
	if !evt1IsFloating(target) || target.Name == "half" {
		return Value{}, evt1Diagnostic("CAST_COMPTIME_UNSUPPORTED", "bounded comptime conversion supports float and double targets", span)
	}
	var numeric float64
	switch value.Kind {
	case ValueInt:
		numeric = float64(value.IntValue)
	case ValueFloat:
		numeric = value.FloatValue
	default:
		return Value{}, evt1Diagnostic("CAST_COMPTIME_UNSUPPORTED", "comptime numeric cast requires a numeric value", span)
	}
	if target.Name == "float" {
		numeric = float64(float32(numeric))
	}
	return Value{Kind: ValueFloat, Type: target, FloatValue: numeric}, nil
}

func (f *evt1FunctionLowerer) lowerNumericRound(call *TemplateCallExpr, indent int) (string, string, Type) {
	prelude, value, _ := f.lowerExpr(call.Args[0], indent)
	resultType := call.ResolvedType
	_, _, _, bits, _ := evt1IntegerTypeRange(call.TypeArg)
	signed, _, _, _, _ := evt1IntegerTypeRange(call.TypeArg)
	input := f.nextTemp("round_source")
	rounded := f.nextTemp("rounded")
	result := f.nextTemp("round_result")
	var out strings.Builder
	out.WriteString(prelude)
	out.WriteString(ind(indent) + fmt.Sprintf("double %s = (double)(%s);\n", input, value))
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", evt1CType(resultType), result))
	errorCtor := evt1FailureConstructorName(resultType, "Error")
	okCtor := evt1FailureConstructorName(resultType, "Ok")
	notFinite := evt1ConstructorName(evt1NumericCastErrorName, "NotFinite") + "()"
	outOfRange := evt1ConstructorName(evt1NumericCastErrorName, "OutOfRange") + "()"
	out.WriteString(ind(indent) + fmt.Sprintf("if (!isfinite(%s)) {\n", input))
	out.WriteString(ind(indent+1) + fmt.Sprintf("%s = %s(%s);\n", result, errorCtor, notFinite))
	out.WriteString(ind(indent) + "} else {\n")
	mode := map[string]string{"TruncTo": "trunc", "FloorTo": "floor", "CeilTo": "ceil"}[call.Callee]
	if mode != "" {
		out.WriteString(ind(indent+1) + fmt.Sprintf("double %s = %s(%s);\n", rounded, mode, input))
	} else {
		base := f.nextTemp("round_base")
		fraction := f.nextTemp("round_fraction")
		out.WriteString(ind(indent+1) + fmt.Sprintf("double %s = floor(%s);\n", base, input))
		out.WriteString(ind(indent+1) + fmt.Sprintf("double %s = %s - %s;\n", fraction, input, base))
		out.WriteString(ind(indent+1) + fmt.Sprintf("double %s = %s < 0.5 ? %s : (%s > 0.5 ? %s + 1.0 : (fmod(%s, 2.0) == 0.0 ? %s : %s + 1.0));\n", rounded, fraction, base, fraction, base, base, base, base))
	}
	lower := "0.0"
	upper := fmt.Sprintf("0x1p%d", bits)
	if signed {
		lower = fmt.Sprintf("-0x1p%d", bits-1)
		upper = fmt.Sprintf("0x1p%d", bits-1)
	}
	out.WriteString(ind(indent+1) + fmt.Sprintf("if (%s < %s || %s >= %s) {\n", rounded, lower, rounded, upper))
	out.WriteString(ind(indent+2) + fmt.Sprintf("%s = %s(%s);\n", result, errorCtor, outOfRange))
	out.WriteString(ind(indent+1) + "} else {\n")
	out.WriteString(ind(indent+2) + fmt.Sprintf("%s = %s((%s)%s);\n", result, okCtor, evt1CType(call.TypeArg), rounded))
	out.WriteString(ind(indent+1) + "}\n")
	out.WriteString(ind(indent) + "}\n")
	return out.String(), result, resultType
}

// lowerNumericCast evaluates the source once. Integer checks precede the C
// conversion, so an unrepresentable value never reaches an implementation-
// defined signed conversion or an accidental modulo conversion.
func (f *evt1FunctionLowerer) lowerNumericCast(cast *CastExpr, indent int) (string, string, Type) {
	prelude, value, source := f.lowerExpr(cast.Value, indent)
	target := cast.Target
	if literal, ok := cast.Value.(*FloatLiteral); ok && cast.SourceType.Name == "double" && target.Name == "double" {
		prelude, value, source = "", evt1RenderFloatLiteral(literal, target), target
	}
	temp := f.nextTemp("cast_source")
	var out strings.Builder
	out.WriteString(prelude)
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(source), temp, value))
	if cast.Kind == "unit_scaled_float" {
		numerator, denominator := source.Quantity.ScaleRatioTo(*target.Quantity)
		return out.String(), fmt.Sprintf("((%s)(((double)%s * %d.0) / %d.0))", evt1CType(target), temp, numerator, denominator), target
	}
	if cast.Kind == "integer_checked_range" {
		sourceSigned, _, _, _, _ := evt1IntegerTypeRange(source)
		targetSigned, targetMax, targetMin, _, _ := evt1IntegerTypeRange(target)
		var invalid []string
		if targetSigned {
			if sourceSigned {
				lower := "-" + strconv.FormatUint(targetMin, 10)
				upper := strconv.FormatUint(targetMax, 10)
				if targetMin == uint64(1)<<63 {
					lower = "INT64_MIN"
				}
				if targetMax == uint64(1)<<63-1 {
					upper = "INT64_MAX"
				}
				invalid = append(invalid, fmt.Sprintf("(int64_t)%s < %s", temp, lower), fmt.Sprintf("(int64_t)%s > %s", temp, upper))
			} else {
				invalid = append(invalid, fmt.Sprintf("(uint64_t)%s > UINT64_C(%d)", temp, targetMax))
			}
		} else {
			if sourceSigned {
				invalid = append(invalid, temp+" < 0")
			}
			if targetMax != ^uint64(0) {
				invalid = append(invalid, fmt.Sprintf("(uint64_t)%s > UINT64_C(%d)", temp, targetMax))
			}
		}
		if len(invalid) > 0 {
			out.WriteString(ind(indent) + fmt.Sprintf("if (%s) { concept_panic(\"integer cast out of range\", %d, %d); }\n", strings.Join(invalid, " || "), cast.Span.Line, cast.Span.Column))
		}
	}
	return out.String(), fmt.Sprintf("((%s)%s)", evt1CType(target), temp), target
}
