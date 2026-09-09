package concept

import (
	"fmt"
	"strings"
)

func evt1ValidateTestAssertion(env *semanticEnv, scope *evt1Scope, call *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	want := map[string]int{"True": 2, "False": 2, "Equals": 3, "Near": 4, "Error": 2, "LGTM": 2}
	argc, ok := want[call.Callee]
	if !ok {
		return Type{}, evt1Diagnostic("TEST_ASSERT_UNKNOWN", fmt.Sprintf("unknown test assertion Assert.%s", call.Callee), call.Span)
	}
	if len(call.Args) != argc {
		return Type{}, evt1Diagnostic("TEST_ASSERT_REASON_REQUIRED", fmt.Sprintf("Assert.%s requires %d value argument(s) followed by a mandatory reason string", call.Callee, argc-1), call.Span)
	}
	reason, ok := call.Args[len(call.Args)-1].(*StringLiteral)
	if !ok {
		return Type{}, evt1Diagnostic("TEST_ASSERT_REASON_REQUIRED", fmt.Sprintf("Assert.%s requires a string literal reason as its final argument", call.Callee), call.Args[len(call.Args)-1].exprSpan())
	}
	if strings.TrimSpace(reason.Value) == "" {
		return Type{}, evt1Diagnostic("TEST_ASSERT_REASON_NONEMPTY", fmt.Sprintf("Assert.%s reason must not be empty or whitespace", call.Callee), reason.Span)
	}
	valueTypes := make([]Type, len(call.Args)-1)
	for i := range valueTypes {
		t, err := validateExpr(env, scope, call.Args[i], templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		valueTypes[i] = evt1CanonicalType(env, t)
	}
	switch call.Callee {
	case "True", "False":
		if valueTypes[0].Name != "bool" {
			return Type{}, evt1Diagnostic("TEST_ASSERT_BOOL_REQUIRED", fmt.Sprintf("Assert.%s requires bool, got %s", call.Callee, valueTypes[0].String()), call.Args[0].exprSpan())
		}
	case "Equals":
		if !valueTypes[0].SameValueType(valueTypes[1]) {
			return Type{}, evt1Diagnostic("TEST_ASSERT_EQUALS_TYPE_MISMATCH", fmt.Sprintf("Assert.Equals requires identical value types, got %s and %s", valueTypes[0].String(), valueTypes[1].String()), call.Span)
		}
		if !evt1TestEqualityType(env, valueTypes[0]) {
			return Type{}, evt1Diagnostic("TEST_ASSERT_EQUALS_UNSUPPORTED", fmt.Sprintf("Assert.Equals does not support %s", valueTypes[0].String()), call.Span)
		}
	case "Near":
		if !valueTypes[0].SameValueType(valueTypes[1]) || !valueTypes[0].SameValueType(valueTypes[2]) || !evt1TestNumericType(valueTypes[0]) {
			return Type{}, evt1Diagnostic("TEST_ASSERT_NEAR_NUMERIC_REQUIRED", "Assert.Near requires three values of one numeric type", call.Span)
		}
		if literal, ok := call.Args[2].(*IntLiteral); ok && literal.Value < 0 {
			return Type{}, evt1Diagnostic("TEST_ASSERT_NEAR_NEGATIVE_TOLERANCE", "Assert.Near tolerance must not be negative", literal.Span)
		}
		if literal, ok := call.Args[2].(*FloatLiteral); ok && literal.Value < 0 {
			return Type{}, evt1Diagnostic("TEST_ASSERT_NEAR_NEGATIVE_TOLERANCE", "Assert.Near tolerance must not be negative", literal.Span)
		}
	case "Error", "LGTM":
		if !evt1IsResultType(valueTypes[0]) {
			return Type{}, evt1Diagnostic("TEST_ASSERT_RESULT_REQUIRED", fmt.Sprintf("Assert.%s requires Result<T,E>, got %s", call.Callee, valueTypes[0].String()), call.Args[0].exprSpan())
		}
	}
	call.Intrinsic = "test_assert_" + strings.ToLower(call.Callee)
	result, _ := evt1BuiltinType("void", call.Span)
	return result, nil
}

func evt1ValidateForetellCall(env *semanticEnv, scope *evt1Scope, call *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if call.Callee != "Checkpoint" {
		return Type{}, evt1Diagnostic("TEST_FORETELL_UNKNOWN", fmt.Sprintf("unknown Foretell operation %s", call.Callee), call.Span)
	}
	if len(call.Args) != 1 {
		return Type{}, evt1Diagnostic("TEST_FORETELL_CHECKPOINT_INVALID", "Foretell.Checkpoint requires one string literal", call.Span)
	}
	message, ok := call.Args[0].(*StringLiteral)
	if !ok || strings.TrimSpace(message.Value) == "" {
		return Type{}, evt1Diagnostic("TEST_FORETELL_CHECKPOINT_INVALID", "Foretell.Checkpoint requires a non-empty string literal", call.Args[0].exprSpan())
	}
	call.Intrinsic = "test_foretell_checkpoint"
	result, _ := evt1BuiltinType("void", call.Span)
	return result, nil
}

func evt1TestNumericType(t Type) bool {
	return t.Name == "int" || t.Name == "uint" || t.Name == "byte" || t.Name == "float"
}

func evt1TestEqualityType(env *semanticEnv, t Type) bool {
	if evt1TestNumericType(t) || t.Name == "bool" || t.Name == "string" {
		return true
	}
	if t.Kind != TypeEnum {
		return false
	}
	decl, ok := env.enums[t.Name]
	if !ok {
		return false
	}
	for _, variant := range decl.Variants {
		if len(variant.Payload) != 0 {
			return false
		}
	}
	return true
}
