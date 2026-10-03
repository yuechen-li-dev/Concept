package concept

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Applying innate concepts: after the validator accepts a module, every
// innate concept judges every declaration of its kind. The first refutation
// is the program's diagnostic; an evaluation that cannot decide is a compiler
// defect (INNATE_UNDECIDED), never a pass.

// evt1InnateEvaluation records one innate concept applied to one declaration,
// for `concept explain` and for tests.
type evt1InnateEvaluation struct {
	Concept string
	Code    string
	Subject evt1DeclarationRef
	Outcome SemanticFactCertainty
	Detail  string
	Verdict *PredicateVerdict
	At      Span
}

// The innate module's semantic environment is shared by every compilation in
// the process; evaluations through it are serialized.
var evt1InnateEvalMu sync.Mutex

const evt1InnatePrerequisiteDepth = 8

func evt1ApplyInnateConcepts(env *semanticEnv, module Module, innate evt1InnateSet) error {
	var concepts []ConceptDecl
	for _, decl := range innate.module.Concepts {
		code, _ := evt1InnateDiagnosticCode(decl)
		if decl.Innate && !env.options.innateOff[code] {
			concepts = append(concepts, decl)
		}
	}
	env.innateEvaluations = nil
	if len(concepts) == 0 {
		return nil
	}
	for _, subject := range evt1InnateSubjects(env, module) {
		for _, decl := range concepts {
			if DeclarationKind(evt1ConceptParameters(decl)[0].DeclarationKind) != subject.Kind {
				continue
			}
			code, _ := evt1InnateDiagnosticCode(decl)
			outcome, detail, at, verdict, err := evt1EvaluateInnateConcept(env, innate, decl, subject, 0)
			evaluation := evt1InnateEvaluation{Concept: decl.Name, Code: code, Subject: subject, Outcome: outcome, Detail: detail, At: at, Verdict: verdict}
			env.innateEvaluations = append(env.innateEvaluations, evaluation)
			if err != nil {
				undecided := evt1Diagnostic("INNATE_UNDECIDED", fmt.Sprintf("innate concept %s could not decide %s: %v. This is a defect in the compiler's innate module, not in your program", decl.Name, subject.qualifiedName(), err), subject.Site)
				return evt1WithInnateProof(undecided, module, evaluation)
			}
			if outcome == FactDisproven {
				return evt1WithInnateProof(evt1Diagnostic(code, detail, at), module, evaluation)
			}
		}
	}
	return nil
}

func evt1WithInnateProof(err error, module Module, evaluation evt1InnateEvaluation) error {
	diagnostic, ok := err.(Diagnostic)
	if !ok {
		return err
	}
	graph := evt1InnateProofGraph(module.Path, evaluation.Subject, []evt1InnateEvaluation{evaluation})
	diagnostic.Proof = &graph
	return diagnostic
}

// evt1InnateSubjects lists the declarations this compilation owns, in source
// order: each struct or enum declared by the module, each closed generic
// instance it materialized, and each struct's fields after their type.
// Imported declarations were judged when their own module was compiled; the
// artifact's innate identity records under which rules.
func evt1InnateSubjects(env *semanticEnv, module Module) []evt1DeclarationRef {
	var types []evt1DeclarationRef
	seen := map[string]bool{}
	addType := func(name string) {
		if seen[name] {
			return
		}
		if ref, ok := evt1TypeDeclarationRef(env, name); ok {
			seen[name] = true
			types = append(types, ref)
		}
	}
	for _, decl := range module.Structs {
		if decl.Module == module.Name && decl.Application == nil {
			addType(decl.Name)
		}
	}
	for _, decl := range module.Enums {
		if decl.Module == module.Name {
			addType(decl.Name)
		}
	}
	instances := make([]string, 0, len(env.genericTypeInstances))
	for name := range env.genericTypeInstances {
		instances = append(instances, name)
	}
	sort.Strings(instances)
	for _, name := range instances {
		addType(name)
	}
	sort.SliceStable(types, func(i, j int) bool {
		a, b := types[i].Site, types[j].Site
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return types[i].Name < types[j].Name
	})
	var subjects []evt1DeclarationRef
	for _, ref := range types {
		subjects = append(subjects, ref)
		if decl, ok := env.structs[ref.Name]; ok {
			for index := range decl.Fields {
				if field, ok := evt1FieldDeclarationRef(env, ref.Name, index); ok {
					subjects = append(subjects, field)
				}
			}
		}
	}
	return subjects
}

// evt1EvaluateInnateConcept decides decl for subject. All requirements must
// hold; the first refutation decides.
func evt1EvaluateInnateConcept(env *semanticEnv, innate evt1InnateSet, decl ConceptDecl, subject evt1DeclarationRef, depth int) (SemanticFactCertainty, string, Span, *PredicateVerdict, error) {
	if depth > evt1InnatePrerequisiteDepth {
		return FactUnknown, "", subject.Site, nil, fmt.Errorf("prerequisite concepts nest deeper than %d", evt1InnatePrerequisiteDepth)
	}
	var metadata *PredicateVerdict
	parameter := evt1ConceptParameters(decl)[0].Name
	for _, requirement := range decl.Requirements {
		switch r := requirement.(type) {
		case *PredicateRequirement:
			args := make([]Value, len(r.Subjects))
			for i := range r.Subjects {
				args[i] = evt1DeclarationValue(subject)
			}
			result, err := evt1InvokeInnatePredicate(env, innate.env, r.Predicate, args, subject, subject.Site)
			if err != nil {
				return FactUnknown, "", subject.Site, nil, fmt.Errorf("%s: %w", r.Predicate, err)
			}
			if evt1IsTypedVerdict(result.Type) {
				result.FactAuthority = evt1TrustedPredicateFactKinds(innate.env, decl)
				metadata = &result
			}
			if result.Outcome == FactUnknown {
				return FactUnknown, result.Message, result.At, metadata, fmt.Errorf("%s returned Unknown", r.Predicate)
			}
			if result.Outcome == FactDisproven {
				return result.Outcome, result.Message, result.At, metadata, nil
			}
		case *CompilerAnalysisRequirement:
			if len(r.SubjectArgs) != 1 || r.SubjectArgs[0].Name != parameter {
				return FactUnknown, "", subject.Site, nil, fmt.Errorf("compiler.%s is not applied to %s", r.Analysis, parameter)
			}
			declaration := DeclarationSubject{Kind: subject.Kind, Name: subject.Name, Owner: subject.Owner, Provenance: subject.Provenance, Site: subject.Site}
			graph := &ProofGraph{}
			outcome, detail := evt1ProjectDeclarationAnalysis(env, graph, "", r.Analysis, declaration)
			if outcome == FactUnknown {
				return FactUnknown, "", subject.Site, nil, fmt.Errorf("compiler.%s: %s", r.Analysis, detail)
			}
			if outcome == FactDisproven {
				return FactDisproven, fmt.Sprintf("%s: %s", subject.qualifiedName(), detail), subject.Site, metadata, nil
			}
		case *PrerequisiteRequirement:
			prerequisite, ok := innate.env.concepts[r.ConceptName]
			if !ok || len(evt1ConceptParameters(prerequisite)) != 1 || evt1ConceptParameters(prerequisite)[0].Kind != "declaration" {
				return FactUnknown, "", subject.Site, nil, fmt.Errorf("prerequisite %s is not a declaration concept of the innate module", r.ConceptName)
			}
			outcome, detail, at, verdict, err := evt1EvaluateInnateConcept(env, innate, prerequisite, subject, depth+1)
			if err != nil || outcome != FactProven {
				return outcome, detail, at, verdict, err
			}
		default:
			return FactUnknown, "", subject.Site, nil, fmt.Errorf("unsupported innate requirement")
		}
	}
	return FactProven, "", subject.Site, metadata, nil
}

func evt1InvokeInnatePredicate(env, innateEnv *semanticEnv, name string, args []Value, subject evt1DeclarationRef, span Span) (PredicateVerdict, error) {
	metrics := env.options.innateMetrics
	if metrics == nil {
		evt1InnateEvalMu.Lock()
		defer evt1InnateEvalMu.Unlock()
		return evt1InvokePredicateOnMeasured(innateEnv, env, name, args, span, nil)
	}
	start := time.Now()
	evt1InnateEvalMu.Lock()
	locked := time.Now()
	var usage evt1ComptimeUsage
	value, err := evt1InvokePredicateOnMeasured(innateEnv, env, name, args, span, &usage)
	finished := time.Now()
	evt1InnateEvalMu.Unlock()
	metrics.record(evt1InnateSample{Module: env.moduleName, Predicate: name, Subject: subject.qualifiedName(), Usage: usage, Wait: locked.Sub(start), Execution: finished.Sub(locked)})
	return value, err
}

// evt1InnateProofGraph presents the innate concepts that judged one
// declaration as a concept-proof.v1 graph.
func evt1InnateProofGraph(source string, subject evt1DeclarationRef, evaluations []evt1InnateEvaluation) ProofGraph {
	goal := "innate concepts of " + subject.qualifiedName()
	graph := ProofGraph{Schema: ProofSchema, Source: source, Goal: goal, Outcome: FactProven, SourceSpan: subject.Site,
		Subjects: []ProofSubjectDescription{{Kind: string(subject.Kind), Name: subject.qualifiedName()}}}
	root := graph.addNode(ProofGoal, goal, "every innate concept of this declaration's kind applies", FactProven, FactOriginCompilerAnalysis, subject.Site)
	reasons := []string{}
	for _, evaluation := range evaluations {
		kind, edge := ProofRequirement, ProofRequires
		detail := "holds"
		switch evaluation.Outcome {
		case FactDisproven:
			kind, edge, detail = ProofContradiction, ProofConflictsWith, evaluation.Detail
			graph.Outcome = FactDisproven
		case FactUnknown:
			kind, edge, detail = ProofMissingFact, ProofBlockedBy, "undecided"
			if graph.Outcome == FactProven {
				graph.Outcome = FactUnknown
			}
		}
		label := fmt.Sprintf("%s (%s)", evaluation.Concept, evaluation.Code)
		node := graph.addNode(kind, label, detail, evaluation.Outcome, FactOriginCompilerAnalysis, evaluation.At)
		graph.Nodes[len(graph.Nodes)-1].Verdict = evaluation.Verdict
		graph.addEdge(root, node, edge)
		reasons = append(reasons, label+": "+detail)
	}
	graph.Nodes[0].Outcome = graph.Outcome
	graph.Reason = strings.Join(reasons, "; ")
	graph.normalize()
	return graph
}

// evt1ExplainInnate returns the innate judgment of the declaration at line.
func evt1ExplainInnate(env *semanticEnv, source string, line int) (ProofGraph, bool) {
	var subject *evt1DeclarationRef
	var evaluations []evt1InnateEvaluation
	for i, evaluation := range env.innateEvaluations {
		if evaluation.Subject.Site.Line != line {
			continue
		}
		if subject == nil {
			subject = &env.innateEvaluations[i].Subject
		}
		if evaluation.Subject.equal(*subject) {
			evaluations = append(evaluations, evaluation)
		}
	}
	if subject == nil {
		return ProofGraph{}, false
	}
	return evt1InnateProofGraph(source, *subject, evaluations), true
}
