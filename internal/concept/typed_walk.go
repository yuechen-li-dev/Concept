package concept

// Walk the typed syntax, including nested closures and failure expressions.
// Semantic collectors share this traversal rather than recognizing source text.
func evt1WalkTypedExpr(expr Expr, visit func(Expr)) {
	if expr == nil {
		return
	}
	visit(expr)
	walk := func(e Expr) { evt1WalkTypedExpr(e, visit) }
	args := func(es []Expr) {
		for _, e := range es {
			walk(e)
		}
	}
	switch e := expr.(type) {
	case *CallExpr:
		walk(e.Receiver)
		args(e.Args)
	case *TemplateCallExpr:
		args(e.Args)
	case *BinaryExpr:
		walk(e.Left)
		walk(e.Right)
	case *UnaryExpr:
		walk(e.Value)
	case *RefExpr:
		walk(e.Value)
	case *MoveExpr:
		walk(e.Value)
	case *ParenExpr:
		walk(e.Value)
	case *CastExpr:
		walk(e.Value)
	case *InterpretExpr:
		walk(e.Value)
	case *AwaitExpr:
		walk(e.Value)
	case *FailureExpr:
		walk(e.Value)
		walk(e.Else)
	case *FieldExpr:
		walk(e.Receiver)
	case *IndexExpr:
		walk(e.Base)
		args(evt1StorageIndices(e))
	case *IfExpr:
		walk(e.Condition)
		walk(e.Then)
		walk(e.Else)
	case *MatchExpr:
		walk(e.Subject)
		for _, arm := range e.Arms {
			walk(arm.Value)
		}
	case *ConstructExpr:
		args(e.Args)
	case *StructConstructExpr:
		args(e.Args)
	case *ArrayLiteralExpr:
		args(e.Elements)
	case *RepeatInitializer:
		walk(e.Value)
		walk(e.Count)
	case *WithExpr:
		walk(e.Base)
		for _, field := range e.Updates {
			walk(field.Value)
		}
	case *BindExpr:
		walk(e.Source)
	case *CallableExpr:
		for _, capture := range e.Captures {
			walk(capture.Source)
		}
		evt1WalkTypedBlock(e.Body, visit)
	case *InferExpr:
		for _, c := range e.Candidates {
			walk(c.Guard)
			walk(c.Score)
		}
	case *DecideExpr:
		for _, c := range e.Candidates {
			walk(c.Guard)
			walk(c.Score)
		}
	}
}
func evt1WalkTypedBlock(block Block, visit func(Expr)) {
	walk := func(e Expr) { evt1WalkTypedExpr(e, visit) }
	child := func(b Block) { evt1WalkTypedBlock(b, visit) }
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			walk(s.Value)
		case *AssignStmt:
			walk(s.Target)
			walk(s.Value)
		case *ReturnStmt:
			walk(s.Value)
		case *ExprStmt:
			walk(s.Value)
		case *AssertStmt:
			walk(s.Condition)
			walk(s.Reason)
		case *StaticAssertStmt:
			walk(s.Condition)
			walk(s.Message)
		case *IfStmt:
			walk(s.Condition)
			child(s.Then)
			if s.Else != nil {
				child(*s.Else)
			}
		case *WhileStmt:
			walk(s.Condition)
			walk(s.Bound)
			child(s.Body)
			if s.Else != nil {
				child(*s.Else)
			}
		case *ForeachStmt:
			walk(s.Source)
			child(s.Body)
		case *MatchStmt:
			walk(s.Subject)
			for _, arm := range s.Arms {
				child(arm.Block)
			}
		case *TryStmt:
			child(s.Body)
			for _, arm := range s.Except {
				child(arm.Body)
			}
		case *Block:
			child(*s)
		case *OnStmt:
			walk(s.Guard)
			child(s.Body)
		case *MachineCompleteStmt:
			walk(s.Value)
		case *InstanceDecl:
			for _, arg := range s.StateArgs {
				walk(arg)
			}
		case *TransitionMatchStmt:
			walk(s.Subject)
			for _, arm := range s.Arms {
				walk(arm.Guard)
			}
		case *TransitionDecideStmt:
			for _, c := range s.Candidates {
				walk(c.Guard)
				walk(c.Score)
			}
		case *TransitionInferStmt:
			for _, c := range s.Candidates {
				walk(c.Guard)
				walk(c.Score)
			}
		}
	}
}
