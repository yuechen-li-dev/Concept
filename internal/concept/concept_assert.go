package concept

import (
	"fmt"
	"sort"
	"strings"
)

type conceptAssertionSubject struct {
	description ProofSubjectDescription
	typeValue   Type
	binding     *evt1ValueBinding
	function    *FunctionDecl
	span        Span
}

func evt1ValidateConceptAssertion(env *semanticEnv, scope *evt1Scope, call *CallExpr) (Type, error) {
	void, _ := evt1BuiltinType("void", call.Span)
	if len(call.Args) < 2 {
		return Type{}, evt1Diagnostic("CONCEPT_ASSERT_REASON_REQUIRED", "Assert.Concept requires semantic subject argument(s) followed by a mandatory reason string", call.Span)
	}
	reason, ok := call.Args[len(call.Args)-1].(*StringLiteral)
	if !ok {
		return Type{}, evt1Diagnostic("CONCEPT_ASSERT_REASON_REQUIRED", "Assert.Concept requires a string literal reason as its final argument", call.Args[len(call.Args)-1].exprSpan())
	}
	if strings.TrimSpace(reason.Value) == "" {
		return Type{}, evt1Diagnostic("CONCEPT_ASSERT_REASON_NONEMPTY", "Assert.Concept reason must not be empty or whitespace", reason.Span)
	}
	subjects := make([]conceptAssertionSubject, 0, len(call.Args)-1)
	for _, argument := range call.Args[:len(call.Args)-1] {
		name, ok := argument.(*NameExpr)
		if !ok {
			return Type{}, evt1Diagnostic("CONCEPT_ASSERT_SUBJECT_NOT_QUERYABLE", "Assert.Concept subjects must name an existing value, type, or operation; runtime expressions are not evaluated", argument.exprSpan())
		}
		subject, err := evt1ResolveConceptAssertionSubject(env, scope, name)
		if err != nil {
			return Type{}, err
		}
		subjects = append(subjects, subject)
	}
	graph, err := evt1BuildConceptAssertionGraph(env, call.ConceptGoal, call.ConceptParameters, subjects, reason.Value, call.Span)
	if err != nil {
		return Type{}, err
	}
	call.Intrinsic = "concept_assert"
	proof := MIRSemanticProof{ID: "assert-proof-" + digest([]byte(graph.Goal + "|" + fmt.Sprintf("%d:%d", call.Span.Line, call.Span.Column)))[:16], Concept: "Assert.Concept", Analysis: call.ConceptGoal, FactKind: evt1FactKind(call.ConceptGoal), Satisfied: graph.Outcome == FactProven, Outcome: string(graph.Outcome), Parameters: append([]int{}, call.ConceptParameters...), Origin: FactOriginCompilerAnalysis, SourceSpan: call.Span}
	for _, subject := range subjects {
		proof.Subjects = append(proof.Subjects, MIRSemanticSubject{Kind: subject.description.Kind, Name: subject.description.Name, Type: subject.description.Type})
	}
	env.semanticProofs = append(env.semanticProofs, proof)
	env.proofGraphs = append(env.proofGraphs, graph)
	if graph.Outcome != FactProven {
		code := "CONCEPT_ASSERT_DISPROVEN"
		if graph.Outcome == FactUnknown {
			code = "CONCEPT_ASSERT_UNKNOWN"
		}
		return Type{}, Diagnostic{Code: code, Message: fmt.Sprintf("%s %s: %s", strings.ToUpper(string(graph.Outcome)), graph.Goal, reason.Value), Span: call.Span, Proof: &graph}
	}
	return void, nil
}

func evt1ResolveConceptAssertionSubject(env *semanticEnv, scope *evt1Scope, name *NameExpr) (conceptAssertionSubject, error) {
	if binding, ok := scope.lookup(name.Name); ok {
		copy := binding
		span := binding.declarationSpan
		if span.Line == 0 {
			span = name.Span
		}
		return conceptAssertionSubject{description: ProofSubjectDescription{Kind: "value", Name: name.Name, Type: binding.t.String()}, typeValue: binding.t, binding: &copy, span: span}, nil
	}
	if candidates := env.functions[name.Name]; len(candidates) == 1 {
		fn := candidates[0]
		return conceptAssertionSubject{description: ProofSubjectDescription{Kind: "operation", Name: name.Name, Type: evt1FunctionSignature(fn)}, function: &fn, span: name.Span}, nil
	} else if len(candidates) > 1 {
		return conceptAssertionSubject{}, evt1Diagnostic("CONCEPT_ASSERT_SUBJECT_AMBIGUOUS", fmt.Sprintf("semantic operation subject %s is overloaded", name.Name), name.Span)
	}
	if decl, ok := env.structs[name.Name]; ok {
		t := Type{Name: decl.Name, Kind: TypeStruct, Span: name.Span}
		return conceptAssertionSubject{description: ProofSubjectDescription{Kind: "type", Name: name.Name, Type: name.Name}, typeValue: t, span: name.Span}, nil
	}
	if alias, ok := env.typeAliases[name.Name]; ok {
		return conceptAssertionSubject{description: ProofSubjectDescription{Kind: "type", Name: name.Name, Type: alias.String()}, typeValue: alias, span: name.Span}, nil
	}
	return conceptAssertionSubject{}, evt1Diagnostic("CONCEPT_ASSERT_SUBJECT_UNKNOWN", fmt.Sprintf("unknown semantic subject %s", name.Name), name.Span)
}

func evt1BuildConceptAssertionGraph(env *semanticEnv, goal string, parameters []int, subjects []conceptAssertionSubject, reason string, span Span) (ProofGraph, error) {
	goalLabel := evt1ProofGoalLabel(goal, parameters, subjects)
	graph := ProofGraph{Schema: ProofSchema, Source: env.sourcePath, Goal: goalLabel, Outcome: FactUnknown, Reason: reason, SourceSpan: span}
	for _, subject := range subjects {
		graph.Subjects = append(graph.Subjects, subject.description)
	}
	root := graph.addNode(ProofGoal, goalLabel, "compile-time semantic assertion", "", FactOriginCompilerAnalysis, span)
	if analysis, ok := evt1SemanticAnalysisRegistry[goal]; ok {
		if len(parameters) != analysis.ParameterArity {
			return ProofGraph{}, evt1Diagnostic("CONCEPT_ASSERT_PARAMETER_INVALID", fmt.Sprintf("%s requires %d analysis parameter(s)", goal, analysis.ParameterArity), span)
		}
		if analysis.ValidateParameters != nil {
			if err := analysis.ValidateParameters(parameters, span); err != nil {
				return ProofGraph{}, err
			}
		}
		wantSubjects := analysis.TypeArity
		if analysis.SubjectArity != 0 {
			wantSubjects = analysis.SubjectArity
		}
		if len(subjects) != wantSubjects {
			return ProofGraph{}, evt1Diagnostic("CONCEPT_ASSERT_ARITY_MISMATCH", fmt.Sprintf("%s requires %d semantic subject(s), got %d", goal, wantSubjects, len(subjects)), span)
		}
		graph.Outcome = evt1ProjectDirectAnalysis(env, &graph, root, goal, parameters, subjects)
	} else if _, ok := env.concepts[goal]; ok {
		if len(parameters) != 0 || len(subjects) != 1 {
			return ProofGraph{}, evt1Diagnostic("CONCEPT_ASSERT_ARITY_MISMATCH", fmt.Sprintf("concept %s requires one semantic subject and no analysis parameters", goal), span)
		}
		graph.Outcome = evt1ProjectNamedConcept(env, &graph, root, goal, subjects[0].typeValue, subjects[0].binding, nil, span)
	} else {
		return ProofGraph{}, evt1Diagnostic("CONCEPT_ASSERT_GOAL_UNKNOWN", fmt.Sprintf("unknown concept or compiler analysis %s", goal), span)
	}
	graph.Nodes[0].Outcome = graph.Outcome
	evt1DeriveProofRepairs(env, &graph, goal, subjects)
	graph.normalize()
	return graph, nil
}

func evt1ProofGoalLabel(goal string, parameters []int, subjects []conceptAssertionSubject) string {
	name := goal
	if len(parameters) > 0 {
		parts := make([]string, len(parameters))
		for i, parameter := range parameters {
			parts[i] = fmt.Sprint(parameter)
		}
		name += "<" + strings.Join(parts, ", ") + ">"
	}
	names := make([]string, len(subjects))
	for i, subject := range subjects {
		names[i] = subject.description.Name
	}
	return name + "(" + strings.Join(names, ", ") + ")"
}

func evt1ProjectDirectAnalysis(env *semanticEnv, graph *ProofGraph, root string, goal string, parameters []int, subjects []conceptAssertionSubject) SemanticFactCertainty {
	for _, subject := range subjects {
		id := graph.addNode(ProofSubject, fmt.Sprintf("%s : %s", subject.description.Name, subject.description.Type), "semantic identity resolved without runtime evaluation", FactProven, FactOriginCompilerAnalysis, subject.span)
		graph.addEdge(root, id, ProofDependsOn)
	}
	if goal == string(FactDisjoint) || goal == string(FactSameRegion) {
		return evt1ProjectRegionRelation(graph, root, goal, subjects)
	}
	if goal == string(FactOutlives) {
		return evt1ProjectOutlives(graph, root, subjects)
	}
	if goal == string(FactNoAllocation) && subjects[0].function != nil {
		return evt1ProjectNoAllocation(env, graph, root, *subjects[0].function, map[string]bool{})
	}
	if (goal == string(FactLifetimeSafe) || goal == string(FactNonEscaping)) && subjects[0].binding != nil && subjects[0].typeValue.Kind == TypeCallable {
		return evt1ProjectCallableLifetime(env, graph, root, goal, subjects[0])
	}
	result := evt1TypeFact(env, evt1FactKind(goal), subjects[0].typeValue, parameters)
	if subjects[0].binding != nil {
		result = evt1RefineValueFact(goal, parameters, *subjects[0].binding, result)
	}
	detail := result.Evidence.Detail
	if detail == "" && (goal == string(FactNoCopy) || goal == string(FactNoOwnershipTransfer)) && (evt1IsSpanType(subjects[0].typeValue) || evt1IsTensorType(subjects[0].typeValue)) {
		detail = "view construction preserves backing storage and ownership"
	}
	if goal == string(FactAligned) {
		detail = fmt.Sprintf("known effective alignment = %d; required alignment = %d", result.Evidence.Alignment, parameters[0])
	}
	if goal == string(FactRank) {
		detail = fmt.Sprintf("known rank = %d; required rank = %d", result.Evidence.Rank, parameters[0])
	}
	kind := ProofKnownFact
	edge := ProofDerivedFrom
	if result.Outcome == FactUnknown {
		kind, edge = ProofMissingFact, ProofBlockedBy
		detail = "no compiler-known fact closes this requirement"
	} else if result.Outcome == FactDisproven {
		kind, edge = ProofContradiction, ProofConflictsWith
	}
	id := graph.addNode(kind, goal+" semantic fact", detail, result.Outcome, result.Origin, subjects[0].span)
	graph.addEdge(root, id, edge)
	return result.Outcome
}

func evt1RefineValueFact(goal string, parameters []int, binding evt1ValueBinding, fallback semanticFactResult) semanticFactResult {
	if binding.tensorFacts != nil {
		facts := binding.tensorFacts
		switch SemanticFactKind(goal) {
		case FactAligned:
			fallback.Evidence.Alignment, fallback.Origin = facts.Alignment, FactOriginTensorBacking
			fallback.Outcome = FactDisproven
			if len(parameters) == 1 && facts.Alignment > 0 && facts.Alignment%parameters[0] == 0 {
				fallback.Outcome = FactProven
			}
		case FactRank:
			fallback.Evidence.Rank, fallback.Origin = facts.Rank, FactOriginTensorBacking
			fallback.Outcome = FactDisproven
			if len(parameters) == 1 && facts.Rank == parameters[0] {
				fallback.Outcome = FactProven
			}
		case FactNoAllocation:
			if facts.NoAllocation {
				fallback.Outcome = FactProven
			}
		case FactNoCopy:
			if facts.NoCopy {
				fallback.Outcome = FactProven
			}
		case FactNoOwnershipTransfer:
			if facts.NoOwnershipTransfer {
				fallback.Outcome = FactProven
			}
		}
	}
	if binding.spanFacts != nil && SemanticFactKind(goal) == FactAligned {
		fallback.Evidence.Alignment, fallback.Origin = binding.spanFacts.Alignment, FactOriginSpan
		fallback.Outcome = FactDisproven
		if len(parameters) == 1 && binding.spanFacts.Alignment > 0 && binding.spanFacts.Alignment%parameters[0] == 0 {
			fallback.Outcome = FactProven
		}
	}
	return fallback
}

func evt1ProjectCallableLifetime(env *semanticEnv, graph *ProofGraph, root, goal string, subject conceptAssertionSubject) SemanticFactCertainty {
	outcome := FactProven
	callable, _ := subject.binding.source.(*CallableExpr)
	if callable == nil {
		outcome = SemanticFactCertainty(subject.typeValue.CallableProvenance)
		if outcome != FactProven && outcome != FactDisproven && outcome != FactUnknown {
			if subject.typeValue.CallableProvenance == string(evt1ProvenanceLocal) {
				outcome = FactDisproven
			} else if subject.typeValue.CallableProvenance == string(evt1ProvenanceUnknown) {
				outcome = FactUnknown
			} else {
				outcome = FactProven
			}
		}
		id := graph.addNode(ProofKnownFact, "callable environment provenance", subject.typeValue.CallableProvenance, outcome, FactOriginControlFlow, subject.span)
		graph.addEdge(root, id, ProofRequires)
		return outcome
	}
	for _, capture := range callable.Captures {
		captureOutcome := FactProven
		if strings.HasPrefix(capture.Provenance, string(evt1ProvenanceLocal)) && evt1CaptureCarriesProvenance(capture) {
			captureOutcome = FactDisproven
			outcome = FactDisproven
		} else if strings.HasPrefix(capture.Provenance, string(evt1ProvenanceUnknown)) {
			captureOutcome = FactUnknown
			if outcome != FactDisproven {
				outcome = FactUnknown
			}
		}
		kind := ProofKnownFact
		edge := ProofRequires
		if captureOutcome == FactDisproven {
			kind, edge = ProofContradiction, ProofConflictsWith
		}
		if captureOutcome == FactUnknown {
			kind, edge = ProofMissingFact, ProofBlockedBy
		}
		detail := fmt.Sprintf("capture kind=%s, provenance=%s", capture.Kind, capture.Provenance)
		if evt1IsSpanType(capture.Type) || evt1IsTensorType(capture.Type) {
			detail += "; view preserves its backing-storage lifetime dependency"
		}
		id := graph.addNode(kind, "capture `"+capture.Name+"`", detail, captureOutcome, FactOriginControlFlow, capture.Span)
		graph.addEdge(root, id, edge)
	}
	if len(callable.Captures) == 0 {
		id := graph.addNode(ProofKnownFact, "empty callable environment", "static provenance", FactProven, FactOriginControlFlow, callable.Span)
		graph.addEdge(root, id, ProofDerivedFrom)
	}
	return outcome
}

func evt1ProjectRegionRelation(graph *ProofGraph, root, goal string, subjects []conceptAssertionSubject) SemanticFactCertainty {
	region := func(subject conceptAssertionSubject) (*evt1SpanFacts, string) {
		if subject.binding == nil {
			return nil, ""
		}
		if subject.binding.spanFacts != nil {
			return subject.binding.spanFacts, subject.binding.spanFacts.RegionID
		}
		if subject.binding.regionFacts != nil {
			return subject.binding.regionFacts, subject.binding.regionFacts.RegionID
		}
		if subject.binding.tensorFacts != nil {
			f := subject.binding.tensorFacts
			return &evt1SpanFacts{RegionID: f.RegionID, BaseOffsetExpression: f.BaseOffset, Alignment: f.Alignment}, f.RegionID
		}
		return nil, ""
	}
	aFacts, a := region(subjects[0])
	bFacts, b := region(subjects[1])
	known := graph.addNode(ProofKnownFact, "region identities", fmt.Sprintf("%s=%s; %s=%s", subjects[0].description.Name, emptyAsUnknown(a), subjects[1].description.Name, emptyAsUnknown(b)), FactProven, FactOriginCompilerAnalysis, graph.SourceSpan)
	graph.addEdge(root, known, ProofDependsOn)
	outcome, detail := FactUnknown, "no distinct declared region identity or statically known non-overlapping intervals"
	if goal == string(FactSameRegion) {
		if a != "" && a == b {
			outcome, detail = FactProven, "both subjects preserve one stable region identity"
		} else if a != "" && b != "" {
			outcome, detail = FactDisproven, "subjects have distinct stable region identities"
		}
	} else if a != "" && b != "" && a != b && (strings.HasPrefix(a, "inline:") || strings.HasPrefix(a, "storage:") || strings.Contains(a, ".")) && (strings.HasPrefix(b, "inline:") || strings.HasPrefix(b, "storage:") || strings.Contains(b, ".")) {
		outcome, detail = FactProven, "distinct compiler-known storage regions"
	} else if a == b && a != "" && aFacts != nil && bFacts != nil {
		// Runtime or missing interval evidence cannot prove overlap or separation.
		detail = "same backing region is known, but no statically non-overlapping intervals are available"
	}
	kind, edge := ProofMissingFact, ProofBlockedBy
	if outcome == FactProven {
		kind, edge = ProofDerivedFact, ProofDerivedFrom
	}
	if outcome == FactDisproven {
		kind, edge = ProofContradiction, ProofConflictsWith
	}
	id := graph.addNode(kind, goal+" relation", detail, outcome, FactOriginCompilerAnalysis, graph.SourceSpan)
	graph.addEdge(root, id, edge)
	return outcome
}

func evt1ProjectOutlives(graph *ProofGraph, root string, subjects []conceptAssertionSubject) SemanticFactCertainty {
	if subjects[0].binding == nil || subjects[1].binding == nil {
		id := graph.addNode(ProofMissingFact, "lifetime provenance", "Outlives requires two value subjects with compiler-known provenance", FactUnknown, FactOriginCompilerAnalysis, graph.SourceSpan)
		graph.addEdge(root, id, ProofBlockedBy)
		return FactUnknown
	}
	a, b := subjects[0].binding.provenance, subjects[1].binding.provenance
	outcome := FactProven
	if a.Kind == evt1ProvenanceUnknown || b.Kind == evt1ProvenanceUnknown {
		outcome = FactUnknown
	} else if evt1LifetimeShorterThan(a, b) {
		outcome = FactDisproven
	}
	kind, edge := ProofDerivedFact, ProofDerivedFrom
	if outcome == FactUnknown {
		kind, edge = ProofMissingFact, ProofBlockedBy
	}
	if outcome == FactDisproven {
		kind, edge = ProofContradiction, ProofConflictsWith
	}
	id := graph.addNode(kind, "provenance ordering", fmt.Sprintf("%s(%d) -> %s(%d)", a.Kind, a.Depth, b.Kind, b.Depth), outcome, FactOriginControlFlow, graph.SourceSpan)
	graph.addEdge(root, id, edge)
	return outcome
}

func evt1ProjectNoAllocation(env *semanticEnv, graph *ProofGraph, root string, fn FunctionDecl, visiting map[string]bool) SemanticFactCertainty {
	if effect, ok := env.operationEffects[fn.Name]; ok {
		origin := FactOriginDeclaredEffect
		if fn.ExternABI != "" {
			origin = FactOriginExternalContractEffect
		}
		id := graph.addNode(ProofContradiction, fn.Name+" Allocates", "authoritative may-allocate operation contract", FactDisproven, origin, effect.Span)
		graph.addEdge(root, id, ProofConflictsWith)
		return FactDisproven
	}
	if visiting[fn.Name] {
		id := graph.addNode(ProofMissingFact, fn.Name, "recursive call summary is not closed", FactUnknown, FactOriginCompilerAnalysis, fn.Span)
		graph.addEdge(root, id, ProofBlockedBy)
		return FactUnknown
	}
	if fn.Body == nil {
		id := graph.addNode(ProofMissingFact, fn.Name, "external operation has no compiler-known allocation summary", FactUnknown, FactOriginCompilerAnalysis, fn.Span)
		graph.addEdge(root, id, ProofBlockedBy)
		return FactUnknown
	}
	visiting[fn.Name] = true
	defer delete(visiting, fn.Name)
	calls := evt1DirectCalls(*fn.Body)
	outcome := FactProven
	fnNode := graph.addNode(ProofSubgoal, fn.Name, "local operation body", "", FactOriginCompilerAnalysis, fn.Span)
	graph.addEdge(root, fnNode, ProofRequires)
	for _, name := range calls {
		candidates := env.functions[name]
		if len(candidates) != 1 {
			id := graph.addNode(ProofMissingFact, name, "call target or allocation summary is not uniquely available", FactUnknown, FactOriginCompilerAnalysis, fn.Span)
			graph.addEdge(fnNode, id, ProofBlockedBy)
			outcome = FactUnknown
			continue
		}
		child := evt1ProjectNoAllocation(env, graph, fnNode, candidates[0], visiting)
		if child != FactProven {
			outcome = child
		}
	}
	if len(calls) == 0 {
		id := graph.addNode(ProofKnownFact, "no allocating operation", "EVT1 body contains no allocation-capable construct", FactProven, FactOriginCompilerAnalysis, fn.Span)
		graph.addEdge(fnNode, id, ProofDerivedFrom)
	}
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == fnNode {
			graph.Nodes[i].Outcome = outcome
		}
	}
	return outcome
}

func evt1DirectCalls(block Block) []string {
	seen := map[string]bool{}
	var visitExpr func(Expr)
	visitExpr = func(expr Expr) {
		switch e := expr.(type) {
		case *CallExpr:
			if !e.Member && e.Intrinsic == "" {
				seen[e.Callee] = true
			}
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		case *TemplateCallExpr:
			for _, arg := range e.Args {
				visitExpr(arg)
			}
		case *BinaryExpr:
			visitExpr(e.Left)
			visitExpr(e.Right)
		case *UnaryExpr:
			visitExpr(e.Value)
		case *ParenExpr:
			visitExpr(e.Value)
		case *IndexExpr:
			visitExpr(e.Base)
			for _, index := range e.Indices {
				visitExpr(index)
			}
		}
	}
	var visitBlock func(Block)
	visitBlock = func(current Block) {
		for _, statement := range current.Statements {
			switch s := statement.(type) {
			case *ExprStmt:
				visitExpr(s.Value)
			case *VarDecl:
				visitExpr(s.Value)
			case *AssignStmt:
				visitExpr(s.Value)
			case *ReturnStmt:
				visitExpr(s.Value)
			case *IfStmt:
				visitExpr(s.Condition)
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitExpr(s.Condition)
				visitBlock(s.Body)
			case *Block:
				visitBlock(*s)
			}
		}
	}
	visitBlock(block)
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func evt1ProjectNamedConcept(env *semanticEnv, graph *ProofGraph, parent, name string, concrete Type, binding *evt1ValueBinding, path []string, span Span) SemanticFactCertainty {
	if containsString(path, name) {
		return FactDisproven
	}
	path = append(path, name)
	decl := env.concepts[name]
	outcome := FactProven
	for _, raw := range decl.Requirements {
		label := "requirement"
		requirementOutcome := FactProven
		detail := ""
		switch requirement := raw.(type) {
		case *PrerequisiteRequirement:
			label = requirement.ConceptName + "<" + concrete.String() + ">"
			node := graph.addNode(ProofRequirement, label, "prerequisite concept", "", FactOriginDeclared, requirement.Span)
			graph.addEdge(parent, node, ProofRequires)
			requirementOutcome = evt1ProjectNamedConcept(env, graph, node, requirement.ConceptName, concrete, binding, path, span)
			setProofNodeOutcome(graph, node, requirementOutcome)
			if requirementOutcome == FactDisproven {
				outcome = FactDisproven
			} else if requirementOutcome == FactUnknown && outcome == FactProven {
				outcome = FactUnknown
			}
			continue
		case *OperationRequirement:
			required := evt1SubstituteRequirement(*requirement, decl.TypeParam, concrete)
			label = evt1Signature(required.ReturnType, required.Name, required.Params)
			if _, err := evt1LookupRequiredOperation(env, required, span, name+"<"+concrete.String()+">"); err != nil {
				requirementOutcome, detail = FactDisproven, err.Error()
			}
		case *FieldRequirement:
			label = "field " + requirement.Name
			structDecl, ok := env.structs[concrete.valueType().Name]
			if !ok {
				requirementOutcome, detail = FactDisproven, "subject has no structural fields"
			} else if actual, exists := env.fieldSets[structDecl.Name][requirement.Name]; !exists {
				requirementOutcome, detail = FactDisproven, "required field is missing"
			} else {
				if evt1FieldVisibility(structDecl, requirement.Name) == "private" {
					requirementOutcome, detail = FactDisproven, "required field is private"
				}
				expected := evt1SubstituteType(requirement.Type.valueType(), decl.TypeParam, concrete.valueType())
				if !evt1CanonicalType(env, actual.valueType()).Equal(evt1CanonicalType(env, expected)) {
					requirementOutcome, detail = FactDisproven, fmt.Sprintf("required %s; found %s", expected.String(), actual.String())
				}
				if !requirement.Readonly && structDecl.Record {
					requirementOutcome, detail = FactDisproven, "required field is mutable but record fields are readonly"
				}
			}
		case *CompilerAnalysisRequirement:
			label = requirement.Analysis
			analysis := evt1SemanticAnalysisRegistry[requirement.Analysis]
			if len(requirement.SubjectArgs) > 0 {
				bound, err := evt1BindRelationalRequirementSubjects(env, decl, concrete, requirement.SubjectArgs, span)
				if err != nil {
					requirementOutcome, detail = FactDisproven, err.Error()
				} else {
					analysisOutcome, facts := analysis.CheckSubjects(env, bound)
					requirementOutcome, detail = SemanticFactCertainty(analysisOutcome), strings.Join(facts, "; ")
				}
			} else {
				args := make([]Type, len(requirement.TypeArgs))
				for i, arg := range requirement.TypeArgs {
					args[i] = evt1SubstituteType(arg, decl.TypeParam, concrete)
				}
				result := analysis.CheckTypes(env, args, requirement.Parameters)
				if binding != nil {
					result = evt1RefineValueFact(requirement.Analysis, requirement.Parameters, *binding, result)
				}
				requirementOutcome, detail = result.Outcome, result.Evidence.Detail
			}
		}
		kind, edge := ProofRequirement, ProofRequires
		if requirementOutcome == FactDisproven {
			kind, edge = ProofContradiction, ProofConflictsWith
		}
		if requirementOutcome == FactUnknown {
			kind, edge = ProofMissingFact, ProofBlockedBy
		}
		node := graph.addNode(kind, label, detail, requirementOutcome, FactOriginDeclared, raw.requirementSpan())
		graph.addEdge(parent, node, edge)
		if requirementOutcome == FactDisproven {
			outcome = FactDisproven
		} else if requirementOutcome == FactUnknown && outcome == FactProven {
			outcome = FactUnknown
		}
	}
	return outcome
}

func evt1DeriveProofRepairs(env *semanticEnv, graph *ProofGraph, goal string, subjects []conceptAssertionSubject) {
	if graph.Outcome == FactProven {
		return
	}
	switch SemanticFactKind(goal) {
	case FactLifetimeSafe, FactNonEscaping, FactOutlives:
		candidates := []string{"shorten the destination lifetime", "extend the source lifetime"}
		if len(subjects) > 0 && subjects[0].binding != nil {
			if callable, ok := subjects[0].binding.source.(*CallableExpr); ok {
				for _, capture := range callable.Captures {
					if !strings.HasPrefix(capture.Provenance, string(evt1ProvenanceLocal)) {
						continue
					}
					repairType := capture.Type
					if capture.Kind == CaptureRef || capture.Kind == CaptureRefConst {
						repairType = capture.Type.borrowBase()
					} else if evt1IsSpanType(repairType) || evt1IsTensorType(repairType) || repairType.Kind == TypeDyn || evt1IsRefStructType(env, repairType) {
						continue
					}
					if evt1TypeCopyable(env, repairType) {
						candidates = append(candidates, "capture `"+capture.Name+"` by value")
					}
					if !evt1IsImmovableValueType(env, repairType) {
						candidates = append(candidates, "move ownership of `"+capture.Name+"` into the callable")
					}
				}
			}
		}
		graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "change the borrowed lifetime dependency", Candidates: candidates})
	case FactAligned:
		graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "provide storage whose effective alignment satisfies the requested boundary", Candidates: []string{"change the view offset or backing alignment"}})
	case FactDisjoint:
		graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "provide statically provable non-overlap", Candidates: []string{"use distinct declared regions", "use fixed non-overlapping intervals", "provide an explicit caller proof"}})
	case FactRank, FactShape:
		graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "make the subject shape agree with the required shape"})
	case FactNoAllocation:
		graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "remove or replace the unproven allocation dependency"})
	default:
		if _, ok := env.concepts[goal]; ok {
			for _, node := range graph.Nodes {
				if node.Outcome != FactDisproven {
					continue
				}
				if strings.Contains(node.Label, "(") {
					graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "provide the required operation", Candidates: []string{node.Label}})
					return
				}
				if strings.HasPrefix(node.Label, "field ") {
					graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "provide the required field", Candidates: []string{strings.TrimPrefix(node.Label, "field ")}})
					return
				}
			}
			graph.RepairClasses = append(graph.RepairClasses, ProofRepair{Class: "satisfy the failed declared requirement"})
		}
	}
}

func setProofNodeOutcome(graph *ProofGraph, id string, outcome SemanticFactCertainty) {
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == id {
			graph.Nodes[i].Outcome = outcome
			return
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}
