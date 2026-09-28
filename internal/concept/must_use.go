package concept

// MustUse is declaration metadata. The result type is resolved by the normal
// semantic validator before this check, so aliases and closed generic results
// use the same rule as direct declarations.
func evt1MustUseType(env *semanticEnv, t Type) bool {
	if env == nil {
		return false
	}
	seen := map[string]bool{}
	for t.PointerTo == nil && t.ArrayElem == nil && len(t.TypeArgs) == 0 {
		alias, ok := env.typeAliases[t.Name]
		if !ok {
			break
		}
		if seen[t.Name] {
			return false
		}
		seen[t.Name] = true
		t = alias
	}
	if t.PointerTo != nil || t.ArrayElem != nil {
		return false
	}
	if decl, ok := evt1FailureEnumDecl(t); ok {
		return evt1HasNamedAttribute(decl.Attributes, "must_use")
	}
	if decl, ok := env.enums[t.Name]; ok {
		return evt1HasNamedAttribute(decl.Attributes, "must_use")
	}
	if decl, ok := env.structs[t.Name]; ok {
		return evt1HasNamedAttribute(decl.Attributes, "must_use")
	}
	if decl, ok := env.genericTypes[t.Name]; ok {
		return evt1HasNamedAttribute(decl.Struct.Attributes, "must_use")
	}
	return false
}

func evt1MustUseFunctionResult(expr Expr) bool {
	switch e := expr.(type) {
	case *CallExpr:
		return e.MustUseResult
	case *TemplateCallExpr:
		return e.MustUseResult
	case *ParenExpr:
		return evt1MustUseFunctionResult(e.Value)
	default:
		return false
	}
}
