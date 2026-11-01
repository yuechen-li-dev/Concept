package concept

// Typed source predicates shared by C bootstrap and checked LIR production.
// Recognize their enclosing expressions before evaluating earlier operands.
// Any nested expression that inserts CFG must preserve already-evaluated values.
func expressionMayBranch(expr Expr) bool { return expressionControl(expr, true) }
func expressionControl(expr Expr, pureCFG bool) bool {
	var children []Expr
	switch e := expr.(type) {
	case *FailureExpr:
		if evt1IsFailureType(e.ResolvedType) && (e.Op == "?" || e.Op == "!") {
			return true
		}
		children = []Expr{e.Value, e.Else}
	case *BinaryExpr:
		if pureCFG && (e.Op == "and" || e.Op == "or") {
			return true
		}
		children = []Expr{e.Left, e.Right}
	case *CallExpr:
		children = e.Args
	case *ConstructExpr:
		children = e.Args
	case *StructConstructExpr:
		children = e.Args
	case *IfExpr:
		if pureCFG {
			return true
		}
		children = []Expr{e.Condition, e.Then, e.Else}
	case *MatchExpr:
		if pureCFG {
			return true
		}
		children = []Expr{e.Subject}
		for _, arm := range e.Arms {
			children = append(children, arm.Value)
		}
	case *ParenExpr:
		children = []Expr{e.Value}
	case *UnaryExpr:
		children = []Expr{e.Value}
	case *CastExpr:
		children = []Expr{e.Value}
	}
	for _, child := range children {
		if expressionControl(child, pureCFG) {
			return true
		}
	}
	return false
}
func expressionsMayBranch(exprs []Expr) bool {
	for _, e := range exprs {
		if expressionMayBranch(e) {
			return true
		}
	}
	return false
}

func expressionsMayFail(exprs []Expr) bool {
	for _, expr := range exprs {
		if expressionControl(expr, false) {
			return true
		}
	}
	return false
}
