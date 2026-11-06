package concept

// Constructors with omitted type arguments need a destination, including when
// wrapped in parentheses or conditional/match expressions. Ordinary expressions
// retain their existing bottom-up inference and overload scores.
func evt1NeedsCarrierContext(expr Expr) bool {
	switch e := expr.(type) {
	case *ConstructExpr:
		return e.EnumName == "Result" || e.EnumName == "Option"
	case *ParenExpr:
		return evt1NeedsCarrierContext(e.Value)
	case *IfExpr:
		return evt1NeedsCarrierContext(e.Then) || evt1NeedsCarrierContext(e.Else)
	case *MatchExpr:
		for _, arm := range e.Arms {
			if evt1NeedsCarrierContext(arm.Value) {
				return true
			}
		}
	}
	return false
}

// Other arguments may narrow the call first. All remaining candidates must
// agree on each contextual parameter: constructor payloads never guess which
// overload the author intended. Validate each carrier expression once, after
// context has been selected, without speculatively consuming carrier payloads.
func evt1ContextualizeCarrierArguments(env *semanticEnv, scope *evt1Scope, name string, args []Expr, types []Type, info *evt1TemplateInfo, comptime bool) error {
	needsContext := false
	for _, t := range types {
		needsContext = needsContext || t.Name == ""
	}
	if !needsContext {
		return nil
	}
	candidates := append([]FunctionDecl{}, env.functions[name]...)
	if fn, ok := env.comptimeFunctions[name]; ok {
		candidates = append(candidates, fn)
	}
	if info != nil {
		for _, requirement := range info.Requirements {
			if requirement.Operation.Name == name {
				op := requirement.Operation
				candidates = append(candidates, FunctionDecl{Name: op.Name, Params: op.Params, ReturnType: op.ReturnType})
			}
		}
	}
	var viable []FunctionDecl
	bestScore := -1
	for _, fn := range candidates {
		if len(fn.Params) != len(args) {
			continue
		}
		ok := true
		score := 0
		for i, arg := range args {
			if types[i].Name != "" && validateCallArgument(env, scope, fn.Params[i].Type, arg, types[i], info) != nil {
				ok = false
				break
			}
			if types[i].Name != "" {
				if _, literal := arg.(*IntLiteral); !literal && !evt1CanonicalType(env, fn.Params[i].Type).Equal(evt1CanonicalType(env, types[i])) {
					score++
				}
			}
		}
		if ok {
			if bestScore < 0 || score < bestScore {
				viable, bestScore = nil, score
			}
			if score == bestScore {
				viable = append(viable, fn)
			}
		}
	}
	for i, arg := range args {
		if types[i].Name != "" {
			continue
		}
		if len(viable) == 0 {
			// Preserve the ordinary missing-context diagnostic for unknown calls
			// and templates whose type arguments cannot be inferred yet.
			actual, err := validateExpr(env, scope, arg, info, comptime)
			if err != nil {
				return err
			}
			types[i] = actual
			continue
		}
		expected := evt1CanonicalType(env, viable[0].Params[i].Type)
		for _, fn := range viable[1:] {
			if !expected.Equal(evt1CanonicalType(env, fn.Params[i].Type)) {
				return evt1Diagnostic("CV4182", "call "+name+" is ambiguous: carrier argument requires a typed local or explicit template arguments", arg.exprSpan())
			}
		}
		actual, err := validateExprAgainstExpected(env, scope, arg, expected, info, comptime)
		if err != nil {
			return err
		}
		types[i] = actual
	}
	return nil
}
