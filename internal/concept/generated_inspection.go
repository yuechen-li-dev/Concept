package concept

import (
	"encoding/json"
	"fmt"
	"strings"
)

// InspectGeneratedDeclarations returns the checked ordinary declaration and
// its origin. The JSON form is a stable structured view of the same AST used
// by binding, semantic analysis, and lowering.
func InspectGeneratedDeclarations(module Module, symbol string) ([]byte, error) {
	type entry struct {
		Declaration FunctionDecl    `json:"declaration"`
		Origin      GeneratedOrigin `json:"origin"`
	}
	var entries []entry
	for _, fn := range module.Functions {
		if fn.Generated != nil && (symbol == "" || symbol == fn.Name || symbol == fn.Generated.Identity) {
			entries = append(entries, entry{Declaration: fn, Origin: *fn.Generated})
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no generated declaration matches %q", symbol)
	}
	return json.MarshalIndent(entries, "", "  ")
}

// ExplainGeneratedDeclaration adds provenance to the existing proof schema.
// A declaration reaches this point only after the ordinary analyzer accepted
// it; the proof is about that checked declaration, not a generator promise.
func ExplainGeneratedDeclaration(module Module, symbol string) (ProofGraph, error) {
	var matched *FunctionDecl
	for i := range module.Functions {
		fn := &module.Functions[i]
		if fn.Generated != nil && (symbol == fn.Name || symbol == fn.Generated.Identity) {
			if matched != nil {
				return ProofGraph{}, fmt.Errorf("generated symbol %q is ambiguous; use a generated identity", symbol)
			}
			matched = fn
		}
	}
	if matched == nil {
		return ProofGraph{}, fmt.Errorf("no generated declaration matches %q", symbol)
	}
	origin := matched.Generated
	goal := matched.Name + "(" + evt1FunctionParamSignature(*matched) + ")"
	graph := ProofGraph{Schema: ProofSchema, Source: module.Path, Goal: goal, Outcome: FactProven, Reason: "generated declaration passed ordinary semantic checking", SourceSpan: origin.GenerationSite, Generated: origin}
	root := graph.addNode(ProofGoal, goal, "checked generated declaration", FactProven, FactOriginGeneratedByReflection, origin.GenerationSite)
	generator := origin.Generator
	if origin.GeneratorModule != "" {
		generator = origin.GeneratorModule + "." + generator
	}
	generatorNode := graph.addNode(ProofDependency, generator, "generator declaration", FactProven, FactOriginGeneratedByReflection, origin.GeneratorSpan)
	graph.addEdge(root, generatorNode, ProofDerivedFrom)
	reflectedDetail := "reflected type"
	if origin.ReflectedModule != "" {
		reflectedDetail += " from module " + origin.ReflectedModule
	}
	reflected := graph.addNode(ProofSubject, origin.ReflectedType.String(), reflectedDetail, FactProven, FactOriginGeneratedByReflection, origin.GenerationSite)
	graph.addEdge(root, reflected, ProofDependsOn)
	for _, input := range origin.Inputs {
		label := strings.TrimSpace(input.Name + " : " + input.Type.String())
		field := graph.addNode(ProofDependency, label, "reflected field", FactProven, FactOriginGeneratedByReflection, input.Span)
		graph.addEdge(reflected, field, ProofDependsOn)
	}
	graph.normalize()
	return graph, nil
}

// ExplainGeneratedConcept validates the requested concept through the normal
// witness lookup, then attributes a generated required operation.
func ExplainGeneratedConcept(module Module, goal string) (ProofGraph, error) {
	open := strings.IndexByte(goal, '<')
	if open < 1 || !strings.HasSuffix(goal, ">") {
		return ProofGraph{}, fmt.Errorf("generated concept goal must be Name<ClosedType>")
	}
	conceptName := strings.TrimSpace(goal[:open])
	typeName := strings.TrimSpace(goal[open+1 : len(goal)-1])
	env, err := analyzeModule(module)
	if err != nil {
		return ProofGraph{}, err
	}
	decl, ok := env.concepts[conceptName]
	if !ok {
		return ProofGraph{}, fmt.Errorf("unknown concept %s", conceptName)
	}
	target := Type{Name: typeName, Kind: TypeStruct}
	if _, ok := env.enums[typeName]; ok {
		target.Kind = TypeEnum
	}
	if err := validateKnownType(env, target, Span{}, "", false); err != nil {
		return ProofGraph{}, err
	}
	if err := checkConceptApplicationSatisfaction(env, conceptName, []Type{target}, nil, Span{}); err != nil {
		return ProofGraph{}, err
	}
	for _, requirement := range decl.Requirements {
		operation, ok := requirement.(*OperationRequirement)
		if !ok {
			continue
		}
		for _, fn := range module.Functions {
			if fn.Generated == nil || fn.Name != operation.Name || len(fn.Params) == 0 || fn.Params[0].Type.valueType().Name != typeName {
				continue
			}
			graph, err := ExplainGeneratedDeclaration(module, fn.Generated.Identity)
			if err != nil {
				return ProofGraph{}, err
			}
			graph.Goal = goal
			graph.Reason = "concept satisfied by an ordinary checked generated required operation"
			if len(graph.Nodes) > 0 {
				graph.Nodes[0].Label = goal
			}
			return graph, nil
		}
	}
	return ProofGraph{}, fmt.Errorf("concept %s is satisfied without a generated required operation", goal)
}
