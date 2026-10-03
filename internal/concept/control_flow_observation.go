package concept

import "fmt"

// IfLadderObservation is a semantic source observation, not a parser node or
// optimizer fact. Identity comes from the validator's lexical binding and
// checked field path. Unsupported/effectful expressions produce no observation.
type IfLadderObservation struct {
	FunctionOwner string
	FunctionSite  Span
	Site          Span
	SubjectID     string
	Subject       string
	BranchCount   int
	FiniteDomain  bool
}

func elseIfContinuation(s *IfStmt) *IfStmt {
	if s.Else == nil || len(s.Else.Statements) != 1 {
		return nil
	}
	next, ok := s.Else.Statements[0].(*IfStmt)
	if !ok || next.Span != s.Else.Span {
		return nil
	}
	return next
}

func ladderSiteKey(fn FunctionDecl, site Span) string {
	return fmt.Sprintf("%s|%d:%d|%d:%d", fn.Module, fn.Span.Line, fn.Span.Column, site.Line, site.Column)
}

func observationFunctionKey(env *semanticEnv, fn FunctionDecl) string {
	owner := fn.Module
	if owner == "" {
		owner = env.moduleName
		if owner == "" {
			owner = env.sourcePath
		}
	}
	return fmt.Sprintf("%s|%d:%d", owner, fn.Span.Line, fn.Span.Column)
}

func completeControlFlowObservation(env *semanticEnv, fn FunctionDecl) {
	if !env.options.observeControlFlow {
		return
	}
	if env.controlFlowReady == nil {
		env.controlFlowReady = map[string]bool{}
	}
	env.controlFlowReady[observationFunctionKey(env, fn)] = true
}

func observationUnparen(expr Expr) Expr {
	for {
		p, ok := expr.(*ParenExpr)
		if !ok {
			return expr
		}
		expr = p.Value
	}
}

// This deliberately admits only checked storage reads. Calls, indexing,
// dereferences, computed expressions and reference aliases are unknown here.
func observedDiscriminant(env *semanticEnv, scope *evt1Scope, expr Expr) (id, name string, t Type, ok bool) {
	switch e := observationUnparen(expr).(type) {
	case *NameExpr:
		binding, found := scope.lookup(e.Name)
		if !found || binding.t.isReference() || binding.isInstance() {
			return "", "", Type{}, false
		}
		id = fmt.Sprintf("%s|%s|%d:%d", ladderSiteKey(*env.observationFunction, Span{}), e.Name, binding.declarationSpan.Line, binding.declarationSpan.Column)
		return id, e.Name, evt1CanonicalType(env, binding.t), true
	case *FieldExpr:
		id, name, receiver, found := observedDiscriminant(env, scope, e.Receiver)
		if !found {
			return "", "", Type{}, false
		}
		field, found := env.fieldSets[receiver.Name][e.Field]
		if !found || field.isReference() {
			return "", "", Type{}, false
		}
		return id + "." + receiver.Name + "." + e.Field, name + "." + e.Field, evt1CanonicalType(env, field), true
	}
	return "", "", Type{}, false
}

func observedCase(env *semanticEnv, expr Expr, subject Type) (string, bool) {
	switch e := observationUnparen(expr).(type) {
	case *IntLiteral:
		if subject.Kind == TypeEnum {
			return "", false
		}
		negative := e.Negative && e.Magnitude != 0
		return fmt.Sprintf("integer:%t:%d", negative, e.Magnitude), true
	case *ConstructExpr:
		// Equality and exhaustiveness currently admit tag-only enums. Payload
		// constructors are not stable cases, even when their arguments are pure.
		enum, found := env.enums[subject.Name]
		if !found || e.EnumName != enum.Name || len(e.Args) != 0 {
			return "", false
		}
		for _, variant := range enum.Variants {
			if len(variant.Payload) != 0 {
				return "", false
			}
		}
		for _, variant := range enum.Variants {
			if variant.Name == e.VariantName {
				return enum.Module + "|" + enum.Name + "::" + variant.Name, true
			}
		}
	}
	return "", false
}

func evt1ObserveIfLadder(env *semanticEnv, scope *evt1Scope, root *IfStmt) {
	fn := *env.observationFunction
	if fn.Module == "" {
		fn.Module = env.moduleName
		if fn.Module == "" {
			fn.Module = env.sourcePath
		}
	}
	if env.ifLadderContinuations == nil {
		env.ifLadderContinuations = map[string]bool{}
	}
	if env.ifLadderContinuations[ladderSiteKey(fn, root.Span)] {
		return
	}
	var chain []*IfStmt
	for s := root; s != nil; s = elseIfContinuation(s) {
		chain = append(chain, s)
		if s != root {
			env.ifLadderContinuations[ladderSiteKey(fn, s.Span)] = true
		}
	}
	if len(chain) < 3 {
		return
	}
	observation := IfLadderObservation{FunctionOwner: fn.Module, FunctionSite: fn.Span, Site: root.Span, BranchCount: len(chain)}
	cases := map[string]bool{}
	for _, branch := range chain {
		comparison, ok := observationUnparen(branch.Condition).(*BinaryExpr)
		if !ok || comparison.Op != "==" || branch.Comptime {
			return
		}
		id, name, subject, ok := observedDiscriminant(env, scope, comparison.Left)
		value, stable := observedCase(env, comparison.Right, subject)
		if !ok || !stable {
			id, name, subject, ok = observedDiscriminant(env, scope, comparison.Right)
			value, stable = observedCase(env, comparison.Left, subject)
		}
		if !ok || !stable || cases[value] || observation.SubjectID != "" && observation.SubjectID != id {
			return
		}
		cases[value] = true
		observation.SubjectID, observation.Subject = id, name
		_, observation.FiniteDomain = env.enums[subject.Name]
	}
	env.ifLadders = append(env.ifLadders, observation)
}

func observedFunctionLadders(env *semanticEnv, declaration DeclarationSubject) []IfLadderObservation {
	var result []IfLadderObservation
	for _, ladder := range env.ifLadders {
		fn := FunctionDecl{Module: ladder.FunctionOwner, Span: ladder.FunctionSite}
		if ladder.FunctionOwner == declaration.Owner && ladder.FunctionSite == declaration.Site && !env.ifLadderContinuations[ladderSiteKey(fn, ladder.Site)] {
			result = append(result, ladder)
		}
	}
	return result
}

func ladderPreferenceMessage(ladder IfLadderObservation) string {
	reason := "clearer case structure"
	if ladder.FiniteDomain {
		reason = "readability and exhaustiveness"
	}
	return fmt.Sprintf("this %d-branch chain repeatedly discriminates `%s`; prefer `match (%s)` for %s", ladder.BranchCount, ladder.Subject, ladder.Subject, reason)
}

func projectControlFlowAnalysis(env *semanticEnv, graph *ProofGraph, parent string, declaration DeclarationSubject) (SemanticFactCertainty, string) {
	if declaration.Kind != FunctionDeclaration && declaration.Kind != MethodDeclaration || declaration.Provenance != DeclarationAuthored {
		return FactProven, "body observation applies to authored functions and methods"
	}
	if !env.controlFlowReady[observationFunctionKey(env, FunctionDecl{Module: declaration.Owner, Span: declaration.Site})] {
		return FactUnknown, "complete checked function body observation is unavailable"
	}
	ladders := observedFunctionLadders(env, declaration)
	for _, ladder := range ladders {
		node := graph.addNode(ProofContradiction, "categorical if ladder", ladderPreferenceMessage(ladder), FactDisproven, FactOriginCompilerAnalysis, ladder.Site)
		graph.addEdge(parent, node, ProofConflictsWith)
	}
	if len(ladders) > 0 {
		return FactDisproven, "checked function body contains a known match-shaped else-if ladder"
	}
	return FactProven, "no safely established match-shaped else-if ladder in the checked body"
}
