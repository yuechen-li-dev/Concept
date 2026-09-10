package concept

import "fmt"

// evt1SubstituteType is the canonical structural type substitution operation.
// Type is a record-shaped algebra: nominal leaves have no children, while the
// fields below represent every current composite constructor. Keeping this
// traversal centralized prevents carrier-, storage-, and owner-specific
// substitution paths.
func evt1SubstituteType(t Type, typeParam string, concreteType Type) Type {
	// Expression-level type operands are parsed before their surrounding
	// template declaration is attached, so a leaf may still carry its nominal
	// parser kind. Inside this binding scope the matching identifier is the
	// template parameter regardless of that provisional kind.
	if t.Name == typeParam && t.PointerTo == nil && t.ArrayElem == nil && len(t.TypeArgs) == 0 && len(t.CallableParams) == 0 && t.CallableResult == nil {
		out := concreteType
		out.Ownership = t.Ownership
		out.Const = t.Const
		out.Scoped = t.Scoped
		out.Imported = t.Imported
		out.Unsafe = t.Unsafe
		out.Span = t.Span
		return out
	}
	if t.PointerTo != nil {
		base := evt1SubstituteType(*t.PointerTo, typeParam, concreteType)
		t.PointerTo = &base
	}
	if t.ArrayElem != nil {
		elem := evt1SubstituteType(*t.ArrayElem, typeParam, concreteType)
		t.ArrayElem = &elem
	}
	t.TypeArgs = append([]Type(nil), t.TypeArgs...)
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1SubstituteType(t.TypeArgs[i], typeParam, concreteType)
	}
	t.CallableParams = append([]Type(nil), t.CallableParams...)
	for i := range t.CallableParams {
		t.CallableParams[i] = evt1SubstituteType(t.CallableParams[i], typeParam, concreteType)
	}
	if t.CallableResult != nil {
		result := evt1SubstituteType(*t.CallableResult, typeParam, concreteType)
		t.CallableResult = &result
	}
	return t
}

func evt1TypeContainsConceptParameter(t Type) bool {
	if t.Kind == TypeConceptParam {
		return true
	}
	if t.PointerTo != nil && evt1TypeContainsConceptParameter(*t.PointerTo) {
		return true
	}
	if t.ArrayElem != nil && evt1TypeContainsConceptParameter(*t.ArrayElem) {
		return true
	}
	for _, arg := range t.TypeArgs {
		if evt1TypeContainsConceptParameter(arg) {
			return true
		}
	}
	for _, param := range t.CallableParams {
		if evt1TypeContainsConceptParameter(param) {
			return true
		}
	}
	return t.CallableResult != nil && evt1TypeContainsConceptParameter(*t.CallableResult)
}

func evt1RequireClosedType(t Type, context string, span Span) error {
	if evt1TypeContainsConceptParameter(t) {
		return evt1Diagnostic("GENERIC_INSTANTIATION_NOT_CLOSED", fmt.Sprintf("closed instantiation %s retains unresolved template type %s", context, t.String()), span)
	}
	return nil
}
