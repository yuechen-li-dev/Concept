package concept

// A named integral comptime value can use an integral expected type when its
// value fits. This is the same contextual rule as an integer literal, while
// the declaration retains its own type for uses without context.
func evt1ContextualComptimeInteger(env *semanticEnv, expr Expr, expected Type) (bool, error) {
	name, ok := expr.(*NameExpr)
	if !ok || !evt1IntegralRepresentation(expected) || expected.Quantity != nil {
		return false, nil
	}
	decl, ok := env.comptimeDecls[name.Name]
	if !ok {
		return false, nil
	}
	value, err := evt1EvaluateGlobalComptimeDecl(newEVT1ComptimeState(env), decl)
	if err != nil {
		return false, err
	}
	if value.Kind != ValueInt || !evt1IntegralRepresentation(value.Type) || value.Type.Quantity != nil {
		return false, nil
	}
	literal := &IntLiteral{Span: name.Span}
	if value.WideUint {
		literal.Magnitude = value.UintValue
	} else if value.IntValue < 0 {
		literal.Negative = true
		literal.Magnitude = uint64(-int64(value.IntValue))
	} else {
		literal.Magnitude = uint64(value.IntValue)
	}
	if err := evt1ResolveIntegerLiteral(literal, expected); err != nil {
		return false, err
	}
	return true, nil
}
