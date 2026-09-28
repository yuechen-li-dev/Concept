package concept

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExplainMustUse projects the existing core MustUse obligation into the proof
// graph. It does not add attributes or change the obligation's truth.
func ExplainMustUse(path, symbol string, roots []string) (ProofGraph, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return ProofGraph{}, err
	}
	module, err := ParseWithSemanticModuleRoots(filepath.ToSlash(path), string(body), roots)
	if err != nil && strings.Contains(err.Error(), "MODULE_IMPORT_MISSING") {
		module, err = ParseWithBuiltSemanticModuleRoots(filepath.ToSlash(path), string(body), roots)
	}
	if err != nil {
		return ProofGraph{}, err
	}
	env, err := analyzeModule(module)
	if err != nil {
		return ProofGraph{}, err
	}
	graph := ProofGraph{Schema: ProofSchema, Source: module.Path, Goal: "MustUse(" + symbol + ")", Reason: "core semantic result-use obligation"}
	var direct, viaType bool
	var site Span
	var origin SemanticFactOrigin = FactOriginDeclared
	var detail string
	if candidates := env.functions[symbol]; len(candidates) == 1 {
		fn := candidates[0]
		site = fn.Span
		direct = evt1HasNamedAttribute(fn.Attributes, "must_use")
		viaType = evt1MustUseType(env, fn.ReturnType)
		if direct {
			detail = "[[must_use]] on function " + symbol
		}
		if viaType && !direct {
			detail = "[[must_use]] on return type " + fn.ReturnType.String()
		}
		if fn.ExternABI != "" {
			origin = FactOriginDeclaredForeign
			detail += "; foreign declaration"
		}
		if fn.Module != module.Name {
			detail += "; imported from semantic artifact " + fn.Module
			if origin != FactOriginDeclaredForeign {
				origin = FactOriginModuleFactSummary
			}
		}
	} else if len(candidates) > 1 {
		return ProofGraph{}, fmt.Errorf("MUST_USE_SUBJECT_AMBIGUOUS: %s has %d overloads", symbol, len(candidates))
	} else if decl, ok := env.enums[symbol]; ok {
		site, direct = decl.Span, evt1HasNamedAttribute(decl.Attributes, "must_use")
		if direct {
			detail = "[[must_use]] on type " + symbol
		}
		if decl.Module != module.Name {
			detail += "; imported from semantic artifact " + decl.Module
			origin = FactOriginModuleFactSummary
		}
	} else if decl, ok := env.structs[symbol]; ok {
		site, direct = decl.Span, evt1HasNamedAttribute(decl.Attributes, "must_use")
		if direct {
			detail = "[[must_use]] on type " + symbol
		}
		if decl.Module != module.Name {
			detail += "; imported from semantic artifact " + decl.Module
			origin = FactOriginModuleFactSummary
		}
	} else {
		return ProofGraph{}, fmt.Errorf("MUST_USE_SUBJECT_UNKNOWN: %s", symbol)
	}
	graph.SourceSpan = site
	if direct || viaType {
		graph.Outcome = FactProven
	} else {
		graph.Outcome = FactDisproven
		detail = "no MustUse declaration applies"
	}
	root := graph.addNode(ProofGoal, graph.Goal, "core semantic obligation", graph.Outcome, FactOriginCompilerAnalysis, site)
	node := graph.addNode(ProofKnownFact, symbol, detail, graph.Outcome, origin, site)
	graph.addEdge(root, node, ProofDependsOn)
	graph.normalize()
	return graph, nil
}
