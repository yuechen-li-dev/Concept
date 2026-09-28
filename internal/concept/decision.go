package concept

import "fmt"

// A value-level decision is deliberately bounded to payload-free enum values.
// The destination supplies the candidate domain, as with infer's typed context.
func validateDecideExpr(env *semanticEnv, scope *evt1Scope, expr *DecideExpr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	candidateType := evt1CanonicalType(env, expected)
	decl, ok := env.enums[candidateType.Name]
	if !ok {
		return Type{}, evt1Diagnostic("DECIDE_REQUIRES_ENUM_CONTEXT", fmt.Sprintf("decide requires an enum destination, got %s", expected.String()), expr.Span)
	}
	if len(expr.Candidates) == 0 {
		return Type{}, evt1Diagnostic("DECIDE_EMPTY", "decide requires at least one candidate", expr.Span)
	}
	variants := make(map[string]VariantDecl, len(decl.Variants))
	for _, variant := range decl.Variants {
		variants[variant.Name] = variant
	}
	seen := map[string]bool{}
	var scoreType Type
	for i := range expr.Candidates {
		candidate := &expr.Candidates[i]
		variant, exists := variants[candidate.Identity]
		if !exists || len(variant.Payload) != 0 {
			return Type{}, evt1Diagnostic("DECIDE_UNKNOWN_CANDIDATE", fmt.Sprintf("candidate %s must be a payload-free variant of %s", candidate.Identity, candidateType.Name), candidate.Span)
		}
		if seen[candidate.Identity] {
			return Type{}, evt1Diagnostic("DECIDE_DUPLICATE_CANDIDATE", fmt.Sprintf("candidate %s appears more than once", candidate.Identity), candidate.Span)
		}
		seen[candidate.Identity] = true
		if candidate.DeclarationOrder != i {
			return Type{}, evt1Diagnostic("DECIDE_MIR_INVALID", "decision candidate order is not declaration order", candidate.Span)
		}
		if candidate.Guard != nil {
			guardType, err := validateExpr(env, scope, candidate.Guard, templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if guardType.Name != "bool" {
				return Type{}, evt1Diagnostic("DECIDE_GUARD_REQUIRES_BOOL", fmt.Sprintf("decide guard must be bool, got %s", guardType.String()), candidate.Guard.exprSpan())
			}
		}
		current, err := validateExpr(env, scope, candidate.Score, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if current.Name != "int" && current.Name != "float" {
			return Type{}, evt1Diagnostic("DECIDE_SCORE_TYPE_INVALID", fmt.Sprintf("decide score must be int or float, got %s", current.String()), candidate.Score.exprSpan())
		}
		if i == 0 {
			scoreType = current
		} else if evt1TypeIdentity(current) != evt1TypeIdentity(scoreType) {
			return Type{}, evt1Diagnostic("DECIDE_SCORE_TYPE_MISMATCH", fmt.Sprintf("decide scores must have one exact type; first is %s but candidate %d is %s", scoreType.String(), i+1, current.String()), candidate.Score.exprSpan())
		}
	}
	expr.CandidateType = candidateType
	expr.ScoreType = scoreType
	return candidateType, nil
}
