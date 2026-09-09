package concept

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const ProofSchema = "concept-proof.v1"

type ProofNodeKind string

const (
	ProofGoal          ProofNodeKind = "Goal"
	ProofRequirement   ProofNodeKind = "Requirement"
	ProofSubgoal       ProofNodeKind = "Subgoal"
	ProofKnownFact     ProofNodeKind = "KnownFact"
	ProofDerivedFact   ProofNodeKind = "DerivedFact"
	ProofMissingFact   ProofNodeKind = "MissingFact"
	ProofContradiction ProofNodeKind = "Contradiction"
	ProofDependency    ProofNodeKind = "Dependency"
	ProofSubject       ProofNodeKind = "Subject"
)

type ProofEdgeKind string

const (
	ProofRequires      ProofEdgeKind = "Requires"
	ProofDerivedFrom   ProofEdgeKind = "DerivedFrom"
	ProofDependsOn     ProofEdgeKind = "DependsOn"
	ProofConflictsWith ProofEdgeKind = "ConflictsWith"
	ProofBlockedBy     ProofEdgeKind = "BlockedBy"
)

type ProofSubjectDescription struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type ProofNode struct {
	ID         string                `json:"id"`
	Kind       ProofNodeKind         `json:"kind"`
	Label      string                `json:"label"`
	Outcome    SemanticFactCertainty `json:"outcome,omitempty"`
	Detail     string                `json:"detail,omitempty"`
	Origin     SemanticFactOrigin    `json:"origin,omitempty"`
	SourceSpan Span                  `json:"source_span,omitempty"`
}

type ProofEdge struct {
	From string        `json:"from"`
	To   string        `json:"to"`
	Kind ProofEdgeKind `json:"kind"`
}

type ProofRepair struct {
	Class      string   `json:"class"`
	Candidates []string `json:"candidates,omitempty"`
}

type ProofGraph struct {
	Schema        string                    `json:"schema"`
	Source        string                    `json:"source"`
	Goal          string                    `json:"goal"`
	Outcome       SemanticFactCertainty     `json:"outcome"`
	Subjects      []ProofSubjectDescription `json:"subjects"`
	Nodes         []ProofNode               `json:"nodes"`
	Edges         []ProofEdge               `json:"edges"`
	FailedNodes   []string                  `json:"failed_nodes,omitempty"`
	UnknownNodes  []string                  `json:"unknown_nodes,omitempty"`
	RepairClasses []ProofRepair             `json:"repair_classes,omitempty"`
	SourceSpans   []Span                    `json:"source_spans,omitempty"`
	FactOrigins   []SemanticFactOrigin      `json:"fact_origins,omitempty"`
	Reason        string                    `json:"reason"`
	SourceSpan    Span                      `json:"source_span"`
}

func proofNodeID(goal string, kind ProofNodeKind, order int, label string, span Span) string {
	identity := fmt.Sprintf("%s|%s|%d|%s|%d:%d", goal, kind, order, label, span.Line, span.Column)
	return "proof-node-" + digest([]byte(identity))[:16]
}

func (g *ProofGraph) addNode(kind ProofNodeKind, label, detail string, outcome SemanticFactCertainty, origin SemanticFactOrigin, span Span) string {
	id := proofNodeID(g.Goal, kind, len(g.Nodes), label, span)
	g.Nodes = append(g.Nodes, ProofNode{ID: id, Kind: kind, Label: label, Detail: detail, Outcome: outcome, Origin: origin, SourceSpan: span})
	if outcome == FactDisproven {
		g.FailedNodes = append(g.FailedNodes, id)
	}
	if outcome == FactUnknown {
		g.UnknownNodes = append(g.UnknownNodes, id)
	}
	return id
}

func (g *ProofGraph) addEdge(from, to string, kind ProofEdgeKind) {
	g.Edges = append(g.Edges, ProofEdge{From: from, To: to, Kind: kind})
}

func (g *ProofGraph) normalize() {
	g.SourceSpans = nil
	g.FactOrigins = nil
	seenSpans := map[string]bool{}
	seenOrigins := map[SemanticFactOrigin]bool{}
	for _, node := range g.Nodes {
		spanKey := fmt.Sprintf("%d:%d", node.SourceSpan.Line, node.SourceSpan.Column)
		if node.SourceSpan.Line > 0 && !seenSpans[spanKey] {
			seenSpans[spanKey] = true
			g.SourceSpans = append(g.SourceSpans, node.SourceSpan)
		}
		if node.Origin != "" && !seenOrigins[node.Origin] {
			seenOrigins[node.Origin] = true
			g.FactOrigins = append(g.FactOrigins, node.Origin)
		}
	}
	for i := range g.RepairClasses {
		sort.Strings(g.RepairClasses[i].Candidates)
		if len(g.RepairClasses[i].Candidates) > 8 {
			g.RepairClasses[i].Candidates = g.RepairClasses[i].Candidates[:8]
		}
	}
	sort.SliceStable(g.RepairClasses, func(i, j int) bool { return g.RepairClasses[i].Class < g.RepairClasses[j].Class })
}

func SerializeProof(graph ProofGraph) ([]byte, error) {
	graph.normalize()
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(graph); err != nil {
		return nil, err
	}
	return body.Bytes(), nil
}

func RenderProofSummary(graph ProofGraph) string { return renderProof(graph, false) }

func RenderProofVerbose(graph ProofGraph) string { return renderProof(graph, true) }

func renderProof(graph ProofGraph, verbose bool) string {
	var b strings.Builder
	status := strings.ToUpper(string(graph.Outcome))
	if graph.Outcome == FactProven {
		fmt.Fprintf(&b, "%s %s\n", status, graph.Goal)
	} else {
		fmt.Fprintf(&b, "Cannot prove %s\nOutcome: %s\n", graph.Goal, status)
	}
	fmt.Fprintf(&b, "Reason:\n  %q\n", graph.Reason)
	if graph.Outcome == FactUnknown {
		b.WriteString("\nThe compiler does not have enough evidence to prove this constraint.\n")
	} else if graph.Outcome == FactDisproven {
		b.WriteString("\nKnown facts contradict the requested constraint.\n")
	}
	children := map[string][]ProofEdge{}
	for _, edge := range graph.Edges {
		children[edge.From] = append(children[edge.From], edge)
	}
	nodes := map[string]ProofNode{}
	for _, node := range graph.Nodes {
		nodes[node.ID] = node
	}
	if len(graph.Nodes) > 0 {
		b.WriteString("\nGoal\n")
		renderProofNode(&b, graph.Nodes[0], nodes, children, 1, 0, verbose, map[string]bool{})
	}
	if len(graph.RepairClasses) > 0 {
		b.WriteString("\nPossible ways to satisfy this proof:\n")
		for _, repair := range graph.RepairClasses {
			fmt.Fprintf(&b, "  - %s\n", repair.Class)
			for _, candidate := range repair.Candidates {
				fmt.Fprintf(&b, "      %s\n", candidate)
			}
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderProofNode(b *strings.Builder, node ProofNode, nodes map[string]ProofNode, children map[string][]ProofEdge, indent, depth int, verbose bool, seen map[string]bool) {
	if depth >= 8 || seen[node.ID] {
		return
	}
	seen[node.ID] = true
	pad := strings.Repeat("  ", indent)
	status := ""
	if node.Outcome != "" {
		status = "  " + strings.ToUpper(string(node.Outcome))
	}
	fmt.Fprintf(b, "%s%s%s\n", pad, node.Label, status)
	if node.Detail != "" {
		fmt.Fprintf(b, "%s  %s\n", pad, node.Detail)
	}
	if verbose && node.Origin != "" {
		fmt.Fprintf(b, "%s  origin=%s source=%d:%d\n", pad, node.Origin, node.SourceSpan.Line, node.SourceSpan.Column)
	}
	edges := children[node.ID]
	if !verbose && len(edges) > 8 {
		edges = edges[:8]
	}
	for _, edge := range edges {
		child, ok := nodes[edge.To]
		if !ok {
			continue
		}
		if verbose {
			fmt.Fprintf(b, "%s  %s\n", pad, edge.Kind)
		}
		renderProofNode(b, child, nodes, children, indent+1, depth+1, verbose, seen)
	}
	if !verbose && len(children[node.ID]) > len(edges) {
		fmt.Fprintf(b, "%s... %d additional proof branches omitted (use --verbose)\n", strings.Repeat("  ", indent+1), len(children[node.ID])-len(edges))
	}
}

// ExplainSource inspects source assertions without changing source semantics.
// R6b intentionally accepts source-position queries only; arbitrary query
// language remains deferred.
func ExplainSource(path, text string, line int) (ProofGraph, error) {
	tokens, err := lexEVT1(text)
	if err != nil {
		return ProofGraph{}, err
	}
	p := &parser{path: path, tokens: tokens}
	module, err := p.parseModule()
	if err != nil {
		return ProofGraph{}, err
	}
	env, err := analyzeModule(module)
	if err != nil {
		if diagnostic, ok := err.(Diagnostic); ok && diagnostic.Proof != nil && (line == 0 || diagnostic.Proof.SourceSpan.Line == line) {
			return *diagnostic.Proof, nil
		}
		return ProofGraph{}, err
	}
	for _, graph := range env.proofGraphs {
		if line == 0 || graph.SourceSpan.Line == line {
			return graph, nil
		}
	}
	return ProofGraph{}, fmt.Errorf("no Assert.Concept assertion found at requested source position")
}
