package concept

import "fmt"

const evt1InferenceName = "Inference"

func evt1IsInferenceType(t Type) bool {
	return t.Name == evt1InferenceName && len(t.TypeArgs) == 1
}

func evt1InferenceCandidateType(t Type) Type {
	if !evt1IsInferenceType(t) {
		return Type{}
	}
	return t.TypeArgs[0]
}

func evt1InferenceCName(t Type) string {
	return "concept_inference_" + evt1TypeIdentity(evt1InferenceCandidateType(t))
}

func validateInferExpr(env *semanticEnv, scope *evt1Scope, expr *InferExpr, expected Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if !evt1IsInferenceType(expected) {
		return Type{}, evt1Diagnostic("INFER_REQUIRES_INFERENCE_CONTEXT", fmt.Sprintf("infer requires an Inference<T> destination, got %s", expected.String()), expr.Span)
	}
	candidateType := evt1CanonicalType(env, evt1InferenceCandidateType(expected))
	decl, ok := env.enums[candidateType.Name]
	if !ok {
		return Type{}, evt1Diagnostic("INFERENCE_TYPE_INVALID", fmt.Sprintf("Inference candidate type must be an enum, got %s", candidateType.String()), expected.Span)
	}
	if len(expr.Candidates) == 0 {
		return Type{}, evt1Diagnostic("INFER_EMPTY", "infer requires at least one candidate", expr.Span)
	}
	variants := make(map[string]VariantDecl, len(decl.Variants))
	for _, variant := range decl.Variants {
		variants[variant.Name] = variant
	}
	seen := map[string]bool{}
	for i := range expr.Candidates {
		candidate := &expr.Candidates[i]
		variant, exists := variants[candidate.Identity]
		if !exists {
			return Type{}, evt1Diagnostic("INFER_UNKNOWN_CANDIDATE", fmt.Sprintf("candidate %s does not belong to %s", candidate.Identity, candidateType.Name), candidate.Span)
		}
		if len(variant.Payload) != 0 {
			return Type{}, evt1Diagnostic("INFER_UNKNOWN_CANDIDATE", fmt.Sprintf("candidate %s must be a payload-free enum variant", candidate.Identity), candidate.Span)
		}
		if seen[candidate.Identity] {
			return Type{}, evt1Diagnostic("INFER_DUPLICATE_CANDIDATE", fmt.Sprintf("candidate %s appears more than once", candidate.Identity), candidate.Span)
		}
		seen[candidate.Identity] = true
		if candidate.DeclarationOrder != i {
			return Type{}, evt1Diagnostic("INFER_MIR_INVALID", "inference candidate order is not declaration order", candidate.Span)
		}
		if candidate.Guard != nil {
			guardType, err := validateExpr(env, scope, candidate.Guard, templateInfo, inComptimeFn)
			if err != nil {
				return Type{}, err
			}
			if guardType.Name != "bool" {
				return Type{}, evt1Diagnostic("INFER_GUARD_REQUIRES_BOOL", fmt.Sprintf("infer guard must be bool, got %s", guardType.String()), candidate.Guard.exprSpan())
			}
		}
		scoreType, err := validateExpr(env, scope, candidate.Score, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if scoreType.Name != "float" {
			return Type{}, evt1Diagnostic("INFER_SCORE_REQUIRES_FLOAT", fmt.Sprintf("infer score must be float, got %s", scoreType.String()), candidate.Score.exprSpan())
		}
	}
	expr.CandidateType = candidateType
	return evt1CanonicalType(env, expected), nil
}

func evt1EnumVariant(env *semanticEnv, t Type, name string) (VariantDecl, bool) {
	decl, ok := env.enums[t.Name]
	if !ok {
		return VariantDecl{}, false
	}
	for _, variant := range decl.Variants {
		if variant.Name == name {
			return variant, true
		}
	}
	return VariantDecl{}, false
}

func evt1InferenceCandidateTypes(module Module) map[string]bool {
	out := map[string]bool{}
	visitType := func(t Type) {
		if evt1IsInferenceType(t) {
			out[evt1InferenceCandidateType(t).Name] = true
		}
	}
	var visitExpr func(Expr)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *InferExpr:
			if e.CandidateType.Name != "" {
				out[e.CandidateType.Name] = true
			}
			for _, c := range e.Candidates {
				if c.Guard != nil {
					visitExpr(c.Guard)
				}
				visitExpr(c.Score)
			}
		case *ParenExpr:
			visitExpr(e.Value)
		case *UnaryExpr:
			visitExpr(e.Value)
		case *FailureExpr:
			visitExpr(e.Value)
		case *MoveExpr:
			visitExpr(e.Value)
		case *RefExpr:
			visitExpr(e.Value)
		case *BindExpr:
			visitExpr(e.Source)
		case *FieldExpr:
			visitExpr(e.Receiver)
		case *IndexExpr:
			visitExpr(e.Base)
			for _, x := range evt1StorageIndices(e) {
				visitExpr(x)
			}
		case *BinaryExpr:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *CallExpr:
			if e.Receiver != nil {
				visitExpr(e.Receiver)
			}
			for _, x := range e.Args {
				visitExpr(x)
			}
		case *TemplateCallExpr:
			for _, x := range e.Args {
				visitExpr(x)
			}
		case *ConstructExpr:
			for _, x := range e.Args {
				visitExpr(x)
			}
		case *StructConstructExpr:
			for _, x := range e.Args {
				visitExpr(x)
			}
		case *ArrayLiteralExpr:
			for _, x := range e.Elements {
				visitExpr(x)
			}
		case *WithExpr:
			visitExpr(e.Base)
			for _, x := range e.Updates {
				visitExpr(x.Value)
			}
		case *MatchExpr:
			visitExpr(e.Subject)
			for _, x := range e.Arms {
				visitExpr(x.Value)
			}
		case *IfExpr:
			visitExpr(e.Condition)
			visitExpr(e.Then)
			visitExpr(e.Else)
		}
	}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, statement := range block.Statements {
			switch s := statement.(type) {
			case *VarDecl:
				visitType(s.Type)
				visitExpr(s.Value)
			case *AssignStmt:
				visitExpr(s.Target)
				visitExpr(s.Value)
			case *ReturnStmt:
				if s.Value != nil {
					visitExpr(s.Value)
				}
			case *ExprStmt:
				visitExpr(s.Value)
			case *AssertStmt:
				visitExpr(s.Condition)
			case *IfStmt:
				visitExpr(s.Condition)
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitExpr(s.Condition)
				visitBlock(s.Body)
			case *ForeachStmt:
				visitExpr(s.Source)
				visitBlock(s.Body)
			case *MatchStmt:
				visitExpr(s.Subject)
				for _, arm := range s.Arms {
					visitBlock(arm.Block)
				}
			case *TryStmt:
				visitBlock(s.Body)
				for _, arm := range s.Except {
					visitBlock(arm.Body)
				}
			case *TransitionInferStmt:
				out[""] = true
				for _, c := range s.Candidates {
					if c.Guard != nil {
						visitExpr(c.Guard)
					}
					visitExpr(c.Score)
				}
			case *Block:
				visitBlock(*s)
			}
		}
	}
	for _, fn := range module.Functions {
		visitType(fn.ReturnType)
		for _, p := range fn.Params {
			visitType(p.Type)
		}
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	for _, fn := range module.ComptimeFns {
		visitType(fn.ReturnType)
		for _, p := range fn.Params {
			visitType(p.Type)
		}
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	for _, automata := range module.Automata {
		for _, field := range automata.StateFields {
			visitType(field.Type)
		}
		for _, machine := range automata.Machines {
			for _, field := range machine.Fields {
				visitType(field.Type)
			}
			for _, state := range machine.States {
				if state.Body != nil {
					visitBlock(*state.Body)
				}
			}
		}
	}
	return out
}
