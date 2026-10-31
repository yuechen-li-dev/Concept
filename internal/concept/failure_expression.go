package concept

// Typed source predicates shared by C bootstrap and checked LIR production.
// Recognize their enclosing expressions before evaluating earlier operands.
func resultQuestion(expr Expr) bool {
	var children []Expr
	switch e := expr.(type) {
	case *FailureExpr:
		if evt1IsResultType(e.ResolvedType) && e.Op == "?" {
			return true
		}
		children = []Expr{e.Value, e.Else}
	case *BinaryExpr:
		children = []Expr{e.Left, e.Right}
	case *CallExpr:
		children = e.Args
	case *ConstructExpr:
		children = e.Args
	case *StructConstructExpr:
		children = e.Args
	case *IfExpr:
		children = []Expr{e.Condition, e.Then, e.Else}
	case *MatchExpr:
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
		if resultQuestion(child) {
			return true
		}
	}
	return false
}
func resultQuestions(exprs []Expr) bool {
	for _, e := range exprs {
		if resultQuestion(e) {
			return true
		}
	}
	return false
}
