package concept

import (
	"strconv"
)

// evt1InferClosedTemplateArguments closes a template from its parameter types.
// Both concept proof and an ordinary call use this result; value arguments
// (notably fixed storage extents) are part of the same closed identity.
func evt1InferClosedTemplateArguments(env *semanticEnv, decl TemplateDecl, actual []Type) ([]Type, bool) {
	parameters := decl.Parameters
	if len(parameters) == 0 {
		parameters = []GenericParameter{{Name: decl.TypeParam, Kind: "type"}}
	}
	if len(decl.Params) != len(actual) {
		return nil, false
	}
	bindings := make(map[string]Type, len(parameters))
	for i, parameter := range decl.Params {
		if !evt1InferTemplatePattern(env, parameter.Type, actual[i], parameters, bindings) {
			return nil, false
		}
	}
	args := make([]Type, len(parameters))
	for i, parameter := range parameters {
		arg, ok := bindings[parameter.Name]
		if !ok {
			return nil, false
		}
		args[i] = arg
	}
	instance, err := evt1InstantiateTemplateFunctionArgs(decl, parameters, args)
	if err != nil {
		return nil, false
	}
	for i, parameter := range instance.Params {
		closed, err := evt1ResolveType(env, nil, parameter.Type)
		if err != nil || !evt1RequiredOperationTypeEqual(env, closed, actual[i]) {
			return nil, false
		}
	}
	return args, true
}

func evt1InferTemplatePattern(env *semanticEnv, pattern, actual Type, parameters []GenericParameter, bindings map[string]Type) bool {
	// Closed generic structs are represented by their nominal instance name
	// during ordinary expression checking. Reopen the recorded application for
	// inference, while retaining the same concrete type for final checking.
	if application, ok := env.genericTypeApplications[actual.valueType().Name]; ok {
		application.Ownership, application.Const = actual.Ownership, actual.Const
		actual = application
	}
	if len(pattern.TypeArgs) > 0 && len(actual.TypeArgs) == 0 {
		// An imported artifact may carry only the closed nominal instance.
		// Previously inferred bindings can still close this pattern exactly.
		closed := evt1SubstituteBindings(pattern, bindings)
		if !evt1TypeContainsConceptParameter(closed) {
			if resolved, err := evt1ResolveType(env, nil, closed); err == nil {
				return evt1RequiredOperationTypeEqual(env, resolved, actual)
			}
		}
	}
	for _, parameter := range parameters {
		if parameter.Kind == "type" && pattern.Name == parameter.Name && pattern.PointerTo == nil && pattern.ArrayElem == nil && len(pattern.TypeArgs) == 0 {
			candidate := actual.valueType()
			if prior, ok := bindings[parameter.Name]; ok {
				return evt1SemanticTypeEqual(env, prior, candidate)
			}
			bindings[parameter.Name] = candidate
			return true
		}
	}
	if (pattern.PointerTo == nil) != (actual.PointerTo == nil) || (pattern.ArrayElem == nil) != (actual.ArrayElem == nil) || len(pattern.TypeArgs) != len(actual.TypeArgs) {
		return false
	}
	if pattern.PointerTo != nil && !evt1InferTemplatePattern(env, *pattern.PointerTo, *actual.PointerTo, parameters, bindings) {
		return false
	}
	if pattern.ArrayElem != nil {
		if pattern.StorageKind != "" && actual.StorageKind != "" && pattern.StorageKind != actual.StorageKind {
			return false
		}
		if !evt1InferTemplateExtent(pattern, actual, parameters, bindings) || !evt1InferTemplatePattern(env, *pattern.ArrayElem, *actual.ArrayElem, parameters, bindings) {
			return false
		}
	} else if pattern.Name != actual.Name && pattern.PointerTo == nil {
		return false
	}
	for i := range pattern.TypeArgs {
		if !evt1InferTemplatePattern(env, pattern.TypeArgs[i], actual.TypeArgs[i], parameters, bindings) {
			return false
		}
	}
	return true
}

func evt1InferTemplateExtent(pattern, actual Type, parameters []GenericParameter, bindings map[string]Type) bool {
	var expression Expr = pattern.ArrayLengthExpr
	if len(pattern.Shape) == 1 && pattern.Shape[0].Expr != nil {
		expression = pattern.Shape[0].Expr
	}
	if name, ok := expression.(*NameExpr); ok {
		for _, parameter := range parameters {
			if parameter.Kind != "value" || parameter.Name != name.Name {
				continue
			}
			extent := actual.ArrayLength
			if len(actual.Shape) == 1 {
				if actual.Shape[0].Runtime {
					return false
				}
				if actual.Shape[0].Extent != 0 {
					extent = actual.Shape[0].Extent
				} else if parsed, err := strconv.Atoi(actual.Shape[0].Expression); err == nil {
					extent = parsed
				} else if literal, ok := actual.Shape[0].Expr.(*IntLiteral); ok && !literal.Negative {
					extent = int(literal.Magnitude)
				}
			}
			candidate := Type{Name: strconv.Itoa(extent), Kind: TypeTemplateValue}
			if prior, found := bindings[name.Name]; found {
				return prior.Name == candidate.Name
			}
			bindings[name.Name] = candidate
			return true
		}
	}
	if pattern.ArrayLength > 0 && pattern.ArrayLength != actual.ArrayLength {
		return false
	}
	return true
}
