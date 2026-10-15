package concept

// Context reaches numeric syntax only: names, casts, calls and typed values are
// boundaries. Representation context never invents a quantity in a product.
func evt1ContextualFloatTree(expr Expr, expected Type) error {
	if !evt1IsFloating(expected) {
		return nil
	}
	switch e := expr.(type) {
	case *FloatLiteral:
		return evt1ResolveFloatLiteral(e, expected)
	case *ParenExpr:
		return evt1ContextualFloatTree(e.Value, expected)
	case *UnaryExpr:
		if e.Op == "-" {
			return evt1ContextualFloatTree(e.Value, expected)
		}
	case *IfExpr:
		if err := evt1ContextualFloatTree(e.Then, expected); err != nil {
			return err
		}
		return evt1ContextualFloatTree(e.Else, expected)
	case *BinaryExpr:
		if e.Op == "+" || e.Op == "-" || e.Op == "*" || e.Op == "/" {
			scalar := expected.valueType()
			scalar.Quantity = nil
			if err := evt1ContextualFloatTree(e.Left, scalar); err != nil {
				return err
			}
			return evt1ContextualFloatTree(e.Right, scalar)
		}
	}
	return nil
}
func evt1FloatLiteralArithmetic(expr Expr) bool {
	switch e := expr.(type) {
	case *FloatLiteral:
		return true
	case *ParenExpr:
		return evt1FloatLiteralArithmetic(e.Value)
	case *UnaryExpr:
		return e.Op == "-" && evt1FloatLiteralArithmetic(e.Value)
	case *BinaryExpr:
		return (e.Op == "+" || e.Op == "-" || e.Op == "*" || e.Op == "/") && evt1FloatLiteralArithmetic(e.Left) && evt1FloatLiteralArithmetic(e.Right)
	}
	return false
}
