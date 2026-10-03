package concept

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const r9bEvidenceSource = `module Research.Evidence;
profile Core;
enum Truth { Proven, Disproven, Unknown }
enum Failure { None, ContainsReference, UnknownLayout }
template <typename E, typename R> record struct EvidenceResult {
    Truth truth;
    E evidence;
    R refutation;
}
template <typename T> record struct Box { T value; }
comptime EvidenceResult<Box<uint8>, Failure> Inspect() {
    return EvidenceResult<Box<uint8>, Failure>{Truth::Disproven, Box<uint8>{7}, Failure::ContainsReference};
}
comptime EvidenceResult<Box<uint8>, Failure> Result = Inspect();
static_assert(Result.evidence.value == 7, "closed typed evidence");
static_assert(Result.truth == Truth::Disproven, "truth retained");
static_assert(Result.refutation == Failure::ContainsReference, "typed refutation retained");
int Use() { return 7; }
`

func TestR9bClosedComptimeEvidence(t *testing.T) {
	module, err := Parse("evidence.concept", r9bEvidenceSource)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(r9bEvidenceSource)); err != nil {
		t.Fatal(err)
	}
}

func TestR9bClosedComptimeEvidenceArtifact(t *testing.T) {
	artifact := buildSemanticArtifact(t, "evidence.concept", r9bEvidenceSource, nil)
	consumer := `module Research.Use; profile Core; import Research.Evidence;
comptime EvidenceResult<Box<uint8>, Failure> Imported = Inspect();
static_assert(Imported.evidence.value == 7, "artifact only closed evidence");
static_assert(Imported.refutation == Failure::ContainsReference, "artifact typed refutation");
int Main() { return Use(); }
`
	module, err := ParseWithSemanticModules("evidence_use.concept", consumer, map[string][]byte{"Research.Evidence": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "evidence_harness.c", "#include \"evidence_use.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Main")+"() == 7 ? 0 : 1; }\n")
	for run := 1; run < 100; run++ {
		if got := buildSemanticArtifact(t, "evidence.concept", r9bEvidenceSource, nil); !bytes.Equal(got, artifact) {
			t.Fatalf("closed evidence identity changed at run %d", run)
		}
	}
}

func TestR9bClosedComptimeEvidenceRejectsResources(t *testing.T) {
	for _, field := range []string{"ref int value;", "int* value;", "owned int value;"} {
		source := `module Research.Bad; profile Core;
template <typename T> struct Box { ` + field + ` }
comptime Box<int> Inspect() { return Box<int>{0}; }
`
		_, err := Parse("bad_evidence.concept", source)
		if err == nil || !strings.Contains(err.Error(), "CV4216") {
			t.Fatalf("%s: expected comptime type rejection, got %v", field, err)
		}
	}
}

// Keep the next boundary executable. A typed record can carry evidence, but
// the predicate protocol must not infer proof authority from its field names.
func TestR9bTypedEvidencePredicateBoundary(t *testing.T) {
	source := r9bEvidenceSource + `
comptime EvidenceResult<Box<uint8>, Failure> Judge(typename subject) { return Inspect(); }
concept TypedEvidence<T> { requires Judge(T); }
`
	_, err := Parse("typed_predicate.concept", source)
	if err == nil || !strings.Contains(err.Error(), "PREDICATE_REQUIREMENT_INVALID") {
		t.Fatalf("unqualified evidence record became a proof: %v", err)
	}
}

func TestR9bComptimeEvidenceCost(t *testing.T) {
	// Same values, assertions and nesting, with ordinary records as the
	// comparison. This measures the closed-type cost, not innate contention.
	ordinary := strings.Replace(r9bEvidenceSource, `template <typename E, typename R> record struct EvidenceResult {
    Truth truth;
    E evidence;
    R refutation;
}
template <typename T> record struct Box { T value; }`, `record struct ByteBox { uint8 value; }
record struct ConcreteResult { Truth truth; ByteBox evidence; Failure refutation; }`, 1)
	ordinary = strings.ReplaceAll(ordinary, "EvidenceResult<Box<uint8>, Failure>", "ConcreteResult")
	ordinary = strings.ReplaceAll(ordinary, "Box<uint8>", "ByteBox")
	for _, sample := range []struct{ name, source string }{{"ordinary", ordinary}, {"closed-generic", r9bEvidenceSource}} {
		start := time.Now()
		for run := 0; run < 100; run++ {
			if _, err := Parse("cost.concept", sample.source); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("%s 100 complete parses: %s", sample.name, time.Since(start))
	}
	module, err := Parse("cost.concept", r9bEvidenceSource)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	var usage evt1ComptimeUsage
	value, err := evt1InvokeComptimeFunctionOnMeasured(env, env, "Inspect", nil, Span{}, &usage)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("closed evidence: %s; fuel=%d depth=%d loop=%d array=%d", value.Render(), usage.Fuel, usage.Depth, usage.Loop, usage.Array)
}

func TestR9bNegativeCorpusDiagnostics(t *testing.T) {
	for _, specimen := range []struct{ file, code string }{
		{"extent_mismatch", "CONCEPT_ASSERT_DISPROVEN"},
		{"extent_partial_capacity", "CONCEPT_ASSERT_UNKNOWN"},
		{"closed_evidence_reference", "CV4216"},
		{"closed_evidence_owned", "CV4216"},
		{"typed_predicate_protocol", "PREDICATE_REQUIREMENT_INVALID"},
	} {
		path := filepath.Join("..", "..", "language", "evt1", "semantic-vocabulary", "invalid", specimen.file+".concept")
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(path, string(source))
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != specimen.code {
			t.Fatalf("%s: want %s, got %v", specimen.file, specimen.code, err)
		}
	}
}
