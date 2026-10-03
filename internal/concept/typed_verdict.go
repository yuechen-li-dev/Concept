package concept

import (
	"fmt"
	"strings"
)

// PredicateVerdict is metadata on the existing proof lattice, never an
// optimizer fact. Only the selected payload is retained.
type PredicateVerdict struct {
	Predicate     string                `json:"predicate"`
	Type          Type                  `json:"type"`
	Outcome       SemanticFactCertainty `json:"outcome"`
	Evidence      *Value                `json:"evidence,omitempty"`
	Refutation    *Value                `json:"refutation,omitempty"`
	Message       string                `json:"message,omitempty"`
	At            Span                  `json:"at,omitempty"`
	FactAuthority []SemanticFactKind    `json:"fact_authority,omitempty"`
}

func evt1IsTypedVerdict(t Type) bool {
	return t.Name == "Verdict" && len(t.TypeArgs) == 2 && t.ArrayElem == nil && t.PointerTo == nil && !t.isBorrowLike() && !t.isOwned()
}

func evt1VerdictEnum(t Type) EnumDecl {
	return EnumDecl{Name: "Verdict", Variants: []VariantDecl{
		{Name: "Proven", Tag: 0, Payload: []Field{{Name: "evidence", Type: t.TypeArgs[0]}}},
		{Name: "Disproven", Tag: 1, Payload: []Field{{Name: "refutation", Type: t.TypeArgs[1]}}},
		{Name: "Unknown", Tag: 2},
	}}
}

func evt1ValidateVerdictConstruct(env *semanticEnv, scope *evt1Scope, expr *ConstructExpr, expected Type, templateInfo *evt1TemplateInfo, comptime bool) (Type, error) {
	if !comptime {
		return Type{}, evt1Diagnostic("COMPTIME_ONLY_TYPE", "Verdict<E,R> exists only during comptime semantic evaluation", expr.Span)
	}
	resolved, err := evt1ResolveType(env, scope, expected)
	if err != nil {
		return Type{}, err
	}
	variant, ok := evt1LookupVariant(evt1VerdictEnum(resolved), expr.VariantName)
	if expr.EnumName != "Verdict" || !ok {
		return Type{}, evt1Diagnostic("VERDICT_CASE_INVALID", "Verdict cases are Proven(evidence), Disproven(refutation), Unknown", expr.Span)
	}
	if len(expr.Args) != len(variant.Payload) {
		return Type{}, evt1Diagnostic("CV4106", "wrong payload count for Verdict::"+expr.VariantName, expr.Span)
	}
	for i, arg := range expr.Args {
		actual, err := validateExprAgainstExpected(env, scope, arg, variant.Payload[i].Type, templateInfo, true)
		if err != nil {
			return Type{}, err
		}
		if !evt1SemanticTypeEqual(env, actual, variant.Payload[i].Type) {
			return Type{}, evt1Diagnostic("CV4107", fmt.Sprintf("Verdict::%s requires %s, got %s", expr.VariantName, variant.Payload[i].Type.String(), actual.String()), arg.exprSpan())
		}
	}
	expr.ResolvedType = resolved
	return resolved, nil
}

// Normalize all compatibility forms here. This is a projection, not a second
// predicate evaluator: callers feed its Outcome into the ordinary proof graph.
func evt1ProjectPredicateVerdict(predicate string, value Value, at Span) (PredicateVerdict, error) {
	result := PredicateVerdict{Predicate: predicate, Type: value.Type, At: at, Outcome: FactUnknown}
	if value.Kind == ValueBool {
		result.Outcome = FactDisproven
		result.Message = predicate + " returned false"
		if value.BoolValue {
			result.Outcome, result.Message = FactProven, predicate+" returned true"
		}
		return result, nil
	}
	if value.Kind != ValueEnum || value.EnumName != "Verdict" {
		return result, fmt.Errorf("%s returned an invalid predicate result", predicate)
	}
	if evt1IsTypedVerdict(value.Type) {
		switch value.Variant {
		case "Proven", "Disproven":
			if len(value.Payload) != 1 {
				break
			}
			payload := value.Payload[0]
			if err := evt1BoundVerdictPayload(payload); err != nil {
				return result, err
			}
			result.Message = predicate + " returned Verdict::" + value.Variant
			if value.Variant == "Proven" {
				result.Outcome, result.Evidence = FactProven, &payload
			} else {
				result.Outcome, result.Refutation = FactDisproven, &payload
				result.Message = "refutation: " + evt1CompactVerdictValue(payload)
			}
			return result, nil
		case "Unknown":
			if len(value.Payload) == 0 {
				result.Message = predicate + " returned Verdict::Unknown (evidence unavailable)"
				return result, nil
			}
		}
	} else {
		if value.Variant == "Holds" && len(value.Payload) == 0 {
			result.Outcome, result.Message = FactProven, predicate+" returned Verdict::Holds"
			return result, nil
		}
		if value.Variant == "Refuted" && len(value.Payload) == 2 && value.Payload[0].Declaration != nil && value.Payload[1].Kind == ValueString && value.Payload[1].StringValue != "" {
			result.Outcome, result.Message, result.At = FactDisproven, value.Payload[1].StringValue, value.Payload[0].Declaration.Site
			return result, nil
		}
	}
	return result, fmt.Errorf("%s returned an invalid or empty proof result", predicate)
}

func evt1CompactVerdictValue(value Value) string {
	text := value.Render()
	if len(text) > 240 {
		return text[:237] + "..."
	}
	return text
}

func evt1VerdictDiagnosticDetail(verdict PredicateVerdict) string {
	if verdict.Refutation != nil && !strings.HasPrefix(verdict.Message, "refutation:") {
		return verdict.Message + "; refutation: " + evt1CompactVerdictValue(*verdict.Refutation)
	}
	return verdict.Message
}

func evt1BoundVerdictPayload(value Value) error {
	nodes, textBytes := 0, 0
	var visit func(Value, int) error
	visit = func(v Value, depth int) error {
		nodes++
		textBytes += len(v.StringValue)
		if nodes > evt1ComptimeMaxArrayCells || depth > evt1ComptimeMaxArrayNesting || textBytes > evt1ComptimeMaxStringBytes {
			return evt1Diagnostic("VERDICT_PAYLOAD_LIMIT", "semantic payload exceeds bounded value size/depth/string limits", v.Type.Span)
		}
		for _, field := range sortedValueKeys(v.Fields) {
			if err := visit(v.Fields[field], depth+1); err != nil {
				return err
			}
		}
		for _, list := range [][]Value{v.Payload, v.Elements} {
			for _, child := range list {
				if err := visit(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return visit(value, 0)
}

func evt1InvokePredicateOnMeasured(env, subjects *semanticEnv, name string, args []Value, span Span, usage *evt1ComptimeUsage) (PredicateVerdict, error) {
	state := newEVT1ComptimeState(env)
	state.subjects, state.usage = subjects, usage
	value, err := evt1InvokeComptimeValues(state, name, args, span)
	if err != nil {
		return PredicateVerdict{}, err
	}
	result, err := evt1ProjectPredicateVerdict(name, value, span)
	if err != nil {
		return result, err
	}
	if result.Refutation != nil {
		if _, ok := env.comptimeFunctions[name+"Describe"]; ok {
			message, err := evt1InvokeComptimeValues(state, name+"Describe", []Value{*result.Refutation}, span)
			if err != nil {
				return result, err
			}
			if message.Kind != ValueString || strings.TrimSpace(message.StringValue) == "" {
				return result, fmt.Errorf("%sDescribe must render a nonempty refutation message", name)
			}
			result.Message = message.StringValue
		}
	}
	return result, nil
}
