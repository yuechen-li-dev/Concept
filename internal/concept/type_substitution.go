package concept

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
)

// evt1CloneType detaches a type tree before a speculative analysis rewrites
// closed generic applications in place.
func evt1CloneType(t Type) Type {
	if t.PointerTo != nil {
		base := evt1CloneType(*t.PointerTo)
		t.PointerTo = &base
	}
	if t.ArrayElem != nil {
		element := evt1CloneType(*t.ArrayElem)
		t.ArrayElem = &element
	}
	t.TypeArgs = append([]Type(nil), t.TypeArgs...)
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1CloneType(t.TypeArgs[i])
	}
	t.CallableParams = append([]Type(nil), t.CallableParams...)
	for i := range t.CallableParams {
		t.CallableParams[i] = evt1CloneType(t.CallableParams[i])
	}
	if t.CallableResult != nil {
		result := evt1CloneType(*t.CallableResult)
		t.CallableResult = &result
	}
	t.Shape = append([]StorageDimension(nil), t.Shape...)
	return t
}

// evt1SubstituteType is the canonical structural type substitution operation.
// Type is a record-shaped algebra: nominal leaves have no children, while the
// fields below represent every current composite constructor. Keeping this
// traversal centralized prevents carrier-, storage-, and owner-specific
// substitution paths.
func evt1SubstituteType(t Type, typeParam string, concreteType Type) Type {
	return evt1SubstituteBindings(t, map[string]Type{typeParam: concreteType})
}

// Bindings are simultaneous: replacing A with B never captures a second B binding.
func evt1SubstituteBindings(t Type, bindings map[string]Type) Type {
	// Expression-level type operands are parsed before their surrounding
	// template declaration is attached, so a leaf may still carry its nominal
	// parser kind. Inside this binding scope the matching identifier is the
	// template parameter regardless of that provisional kind.
	if concreteType, bound := bindings[t.Name]; bound && t.PointerTo == nil && t.ArrayElem == nil && len(t.TypeArgs) == 0 && len(t.CallableParams) == 0 && t.CallableResult == nil {
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
		base := evt1SubstituteBindings(*t.PointerTo, bindings)
		t.PointerTo = &base
	}
	if t.ArrayElem != nil {
		elem := evt1SubstituteBindings(*t.ArrayElem, bindings)
		t.ArrayElem = &elem
	}
	// Shape expressions may contain type layout queries such as SizeOf<T>().
	// Close their type operands before evaluating the fixed extent.
	if t.ArrayLengthExpr != nil {
		if expr, err := evt1SubstituteExprBindings(t.ArrayLengthExpr, bindings); err == nil {
			t.ArrayLengthExpr = expr
		}
	}
	t.Shape = append([]StorageDimension(nil), t.Shape...)
	for i := range t.Shape {
		var dependencies []Type
		for _, dependency := range t.Shape[i].Dependencies {
			closed := evt1SubstituteBindings(dependency, bindings)
			if evt1TypeContainsConceptParameter(closed) {
				dependencies = append(dependencies, closed)
			}
		}
		t.Shape[i].Dependencies = evt1CanonicalTypeDependencies(dependencies)
		if t.Shape[i].Expr != nil {
			if expr, err := evt1SubstituteExprBindings(t.Shape[i].Expr, bindings); err == nil {
				t.Shape[i].Expr = expr
			}
		}
	}
	t.TypeArgs = append([]Type(nil), t.TypeArgs...)
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1SubstituteBindings(t.TypeArgs[i], bindings)
	}
	t.CallableParams = append([]Type(nil), t.CallableParams...)
	for i := range t.CallableParams {
		t.CallableParams[i] = evt1SubstituteBindings(t.CallableParams[i], bindings)
	}
	if t.CallableResult != nil {
		result := evt1SubstituteBindings(*t.CallableResult, bindings)
		t.CallableResult = &result
	}
	return t
}

func evt1TypeContainsConceptParameter(t Type) bool {
	for _, dimension := range t.Shape {
		if len(dimension.Dependencies) != 0 {
			return true
		}
	}
	if t.Kind == TypeConceptParam {
		return true
	}
	if t.Kind == TypeTemplateValue {
		_, err := strconv.Atoi(t.Name)
		return err != nil
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

func evt1CanonicalTypeDependencies(dependencies []Type) []Type {
	sort.Slice(dependencies, func(a, b int) bool {
		if dependencies[a].Kind != dependencies[b].Kind {
			return dependencies[a].Kind < dependencies[b].Kind
		}
		return dependencies[a].Name < dependencies[b].Name
	})
	return slices.CompactFunc(dependencies, func(a, b Type) bool { return a.Equal(b) })
}
