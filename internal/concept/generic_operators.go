package concept

import "fmt"

// Operator requirements use the same closure and witness records as named
// operations. A compiler-known witness is checked by the ordinary expression
// validator, so its legality cannot drift from a concrete operator expression.
func evt1RequiredOperatorToken(name string) (string, bool) {
	switch name {
	case "operator+", "operator-", "operator*", "operator/", "operator==", "operator!=", "operator<", "operator>", "operator<=", "operator>=":
		return name[len("operator"):], true
	}
	return "", false
}

func evt1ValidateOpenRequiredOperator(env *semanticEnv, info *evt1TemplateInfo, name string, operands []Type, span Span) (Type, error) {
	var matched *evt1TemplateRequirement
	for i := range info.Requirements {
		requirement := &info.Requirements[i]
		if requirement.Operation.Name != name || len(requirement.Operation.Params) != len(operands) {
			continue
		}
		exact := true
		for j, operand := range operands {
			if !evt1TypesCompatible(env, requirement.Operation.Params[j].Type, operand, info.Decl.TypeParam) {
				exact = false
				break
			}
		}
		if !exact {
			continue
		}
		if matched != nil {
			return Type{}, evt1Diagnostic("CV4177", fmt.Sprintf("open operator %s is ambiguously guaranteed by %s", name, info.Decl.Constraint.ConceptName), span)
		}
		matched = requirement
	}
	if matched == nil {
		return Type{}, evt1Diagnostic("CV4175", fmt.Sprintf("open operator %s requires an exact operation requirement", name), span)
	}
	info.CallBindings[evt1SpanKey(span)] = evt1TemplateCallBinding{CallSpan: span, Requirement: *matched}
	return evt1CanonicalType(env, matched.Operation.ReturnType), nil
}

func evt1LookupBuiltinOperatorWitness(env *semanticEnv, required OperationRequirement, span Span, prefix string) (FunctionDecl, error) {
	op, ok := evt1RequiredOperatorToken(required.Name)
	if !ok || len(required.Params) < 1 || len(required.Params) > 2 || len(required.Params) == 1 && op != "-" {
		return FunctionDecl{}, evt1Diagnostic("CV4153", fmt.Sprintf("%s is missing required operator %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params)), span)
	}
	scope := newEVT1Scope(nil)
	scope.declare("__operator_left", evt1ValueBinding{t: required.Params[0].Type, state: evt1StorageInitialized})
	left := &NameExpr{Name: "__operator_left", Span: span}
	var expr Expr = &UnaryExpr{Op: op, Value: left, Span: span}
	if len(required.Params) == 2 {
		scope.declare("__operator_right", evt1ValueBinding{t: required.Params[1].Type, state: evt1StorageInitialized})
		expr = &BinaryExpr{Op: op, Left: left, Right: &NameExpr{Name: "__operator_right", Span: span}, Span: span}
	}
	actual, err := validateExpr(env, scope, expr, nil, false)
	if err != nil {
		return FunctionDecl{}, evt1Diagnostic("CV4153", fmt.Sprintf("%s is missing required operator %s: %v", prefix, evt1Signature(required.ReturnType, required.Name, required.Params), err), span)
	}
	if !evt1RequiredOperationTypeEqual(env, actual, required.ReturnType) {
		return FunctionDecl{}, evt1Diagnostic("CV4156", fmt.Sprintf("%s requires %s but builtin operator returns %s", prefix, evt1Signature(required.ReturnType, required.Name, required.Params), actual.String()), span)
	}
	return FunctionDecl{Name: required.Name, ReturnType: actual, Params: required.Params, Span: span}, nil
}
