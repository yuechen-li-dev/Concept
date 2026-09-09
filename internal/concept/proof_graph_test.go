package concept

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func proofFixture(t *testing.T, name string) (string, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "tooling", "proofs", name)
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.ToSlash(path), body
}

func TestAssertConceptProvenErasesBeforeC(t *testing.T) {
	path, source := proofFixture(t, "concept_alignment_proven.concept")
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["concept_alignment_proven.generated.c"])
	if strings.Contains(generated, "Concept<") || strings.Contains(generated, "concept_assert") {
		t.Fatal("compile-time concept assertion leaked into generated C")
	}
	graph, err := ExplainSource(path, string(source), 0)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Schema != ProofSchema || graph.Outcome != FactProven || graph.Goal != "Aligned<16>(weights)" {
		t.Fatalf("unexpected proof graph: %+v", graph)
	}
}

func TestAssertConceptDistinguishesDisprovenAndUnknown(t *testing.T) {
	cases := []struct {
		name    string
		outcome SemanticFactCertainty
		needle  string
	}{
		{"concept_alignment_disproven.concept", FactDisproven, "known effective alignment = 4"},
		{"concept_rank_disproven.concept", FactDisproven, "known rank = 2"},
		{"concept_callable_capture_disproven.concept", FactDisproven, "capture `scratch`"},
		{"concept_span_lifetime_disproven.concept", FactDisproven, "backing-storage lifetime dependency"},
		{"concept_lifetime_disproven.concept", FactDisproven, "local(2) -> local(1)"},
		{"concept_interface_requirement_disproven.concept", FactDisproven, "missing required operation"},
		{"concept_copyability_disproven.concept", FactDisproven, "Socket Clone"},
		{"concept_disjoint_unknown.concept", FactUnknown, "no statically non-overlapping intervals"},
		{"concept_lifetime_unknown.concept", FactUnknown, "unknown(1) -> unknown(1)"},
		{"concept_noallocation_unknown.concept", FactUnknown, "external operation has no compiler-known allocation summary"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, source := proofFixture(t, tc.name)
			_, err := Parse(path, string(source))
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Proof == nil {
				t.Fatalf("expected structured proof diagnostic, got %v", err)
			}
			if diagnostic.Proof.Outcome != tc.outcome || !strings.Contains(RenderProofSummary(*diagnostic.Proof), tc.needle) {
				t.Fatalf("unexpected proof diagnostic:\n%s", RenderProofSummary(*diagnostic.Proof))
			}
		})
	}
}

func TestAssertConceptExpandsNamedConceptAndOperationProof(t *testing.T) {
	for _, name := range []string{"concept_user_concept_proven.concept", "concept_relational_user_concept_proven.concept", "concept_noallocation_proven.concept", "concept_lifetime_proven.concept", "concept_type_subject_proven.concept"} {
		path, source := proofFixture(t, name)
		graph, err := ExplainSource(path, string(source), 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if graph.Outcome != FactProven || len(graph.Nodes) < 2 {
			t.Fatalf("%s proof is incomplete: %+v", name, graph)
		}
	}
}

func TestAssertConceptPreservationProofsAndLineSelection(t *testing.T) {
	path, source := proofFixture(t, "concept_preservation_proven.concept")
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["concept_preservation_proven.mir.json"]), `"proof_graphs"`) {
		t.Fatal("MIR omitted compile-time proof graphs")
	}
	graph, err := ExplainSource(path, string(source), 11)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Goal != "NoOwnershipTransfer(values)" || graph.Outcome != FactProven {
		t.Fatalf("line-selected graph=%+v", graph)
	}
}

func TestProofJSONIsByteDeterministicAcrossOneHundredRuns(t *testing.T) {
	for _, name := range []string{"concept_alignment_proven.concept", "concept_callable_capture_disproven.concept", "concept_disjoint_unknown.concept", "concept_noallocation_proven.concept"} {
		path, source := proofFixture(t, name)
		var want []byte
		for i := 0; i < 100; i++ {
			graph, err := ExplainSource(path, string(source), 0)
			if err != nil {
				t.Fatalf("%s run %d: %v", name, i, err)
			}
			got, err := SerializeProof(graph)
			if err != nil {
				t.Fatal(err)
			}
			if i == 0 {
				want = got
			} else if string(got) != string(want) {
				t.Fatalf("%s proof changed at run %d", name, i)
			}
		}
	}
}

func TestProofHumanGoldens(t *testing.T) {
	cases := []struct{ fixture, golden string }{
		{"concept_alignment_proven.concept", "alignment-proven.txt"},
		{"concept_callable_capture_disproven.concept", "callable-lifetime-disproven.txt"},
		{"concept_disjoint_unknown.concept", "disjoint-unknown.txt"},
	}
	for _, tc := range cases {
		path, source := proofFixture(t, tc.fixture)
		graph, err := ExplainSource(path, string(source), 0)
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join("..", "..", "language", "evt1", "tooling", "proofs", "golden", tc.golden))
		if err != nil {
			t.Fatal(err)
		}
		if got := RenderProofSummary(graph) + "\n"; got != string(want) {
			t.Fatalf("%s golden drift:\n%s", tc.fixture, got)
		}
	}
}

func TestAssertConceptReasonAndUnevaluatedSubjectRules(t *testing.T) {
	cases := []struct{ source, code string }{
		{"profile Core; int Main() { int x = 1; Assert.Concept<NoAllocation>(x); return x; }", "CONCEPT_ASSERT_REASON_REQUIRED"},
		{"profile Core; int Main() { int x = 1; Assert.Concept<NoAllocation>(x, \" \" ); return x; }", "CONCEPT_ASSERT_REASON_NONEMPTY"},
		{"profile Core; int Main() { int x = 1; Assert.Concept<NoAllocation>(x + 1, \"reason\"); return x; }", "CONCEPT_ASSERT_SUBJECT_NOT_QUERYABLE"},
	}
	for _, tc := range cases {
		_, err := Parse("assert-rule.concept", tc.source)
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
			t.Fatalf("diagnostic=%v, want %s", err, tc.code)
		}
	}
}

func TestAssertConceptFiltersIllegalCaptureRepairs(t *testing.T) {
	source := `profile Core;
immovable struct Socket { int handle; }
int Main() {
    Socket socket = Socket{7};
    auto callback = callback() with (ref socket) { return socket.handle; };
    Assert.Concept<LifetimeSafe>(callback, "socket borrow must not escape");
    return 0;
}`
	_, err := Parse("repair-filter.concept", source)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Proof == nil {
		t.Fatalf("diagnostic=%v", err)
	}
	rendered := RenderProofSummary(*diagnostic.Proof)
	if strings.Contains(rendered, "capture `socket` by value") || strings.Contains(rendered, "move ownership of `socket`") {
		t.Fatalf("illegal repair candidate was not filtered:\n%s", rendered)
	}
}

func TestAsyncLifetimeFailureUsesProofGraph(t *testing.T) {
	path, source := proofFixture(t, "concept_async_lifetime_disproven.concept")
	_, err := Parse(path, string(source))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "ASYNC_PERSISTENT_REF_ESCAPE" || diagnostic.Proof == nil {
		t.Fatalf("expected async proof diagnostic, got %v", err)
	}
	rendered := RenderProofSummary(*diagnostic.Proof)
	for _, text := range []string{"Async persistent field", "live across await", "operation lifetime"} {
		if !strings.Contains(rendered, text) {
			t.Fatalf("async proof omitted %q:\n%s", text, rendered)
		}
	}
}
