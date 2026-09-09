package concept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var semanticFactConformanceCases = []ConformanceCase{
	{Name: "contiguous array", Source: "valid/fact_contiguous_array.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4 storage-derived"},
	{Name: "contiguous span", Source: "valid/fact_contiguous_span.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4g-derived"},
	{Name: "contiguous tensor", Source: "valid/fact_contiguous_tensor.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "fixed inline tensor", Source: "valid/fact_fixed_shape_inline_tensor.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4i-derived"},
	{Name: "runtime bound tensor", Source: "valid/fact_runtime_shape_bound_tensor.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4h-derived"},
	{Name: "layout alignment", Source: "valid/fact_alignment_layout_region.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4f-derived"},
	{Name: "subspan alignment", Source: "valid/fact_alignment_subspan.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4g-derived"},
	{Name: "readonly span", Source: "valid/fact_readonly_span.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4g-derived"},
	{Name: "bind no allocation", Source: "valid/fact_noallocation_bind.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4e-derived"},
	{Name: "stream no copy", Source: "valid/fact_nocopy_stream.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4f-derived"},
	{Name: "same region", Source: "valid/fact_region_same.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4b-derived"},
	{Name: "layout disjoint", Source: "valid/fact_disjoint_layout_regions.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4f-derived"},
	{Name: "inline tensor disjoint", Source: "valid/fact_disjoint_inline_tensors.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4i-derived"},
	{Name: "subspan disjoint", Source: "valid/fact_disjoint_subspans.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4g-derived"},
	{Name: "contiguous concept", Source: "valid/fact_concept_contiguous_buffer.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new"},
	{Name: "aligned concept", Source: "valid/fact_concept_aligned.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new"},
	{Name: "preservation chain", Source: "valid/fact_preservation_chain.concept", Expected: ConformancePass, Accepted: true, MatrixStatus: "EVT1-new / R4f-R4i composition"},
	{Name: "alignment too strong", Source: "invalid/fact_alignment_too_strong.concept", Expected: ConformancePass, DiagnosticCategory: "CV4641", MatrixStatus: "EVT1-new"},
	{Name: "readonly is not mutable", Source: "invalid/fact_mutable_readonly_span.concept", Expected: ConformancePass, DiagnosticCategory: "CV4648", MatrixStatus: "EVT1-new"},
	{Name: "same interval not disjoint", Source: "invalid/fact_disjoint_same_interval.concept", Expected: ConformancePass, DiagnosticCategory: "CV4645", MatrixStatus: "EVT1-new"},
	{Name: "runtime overlap unknown", Source: "invalid/fact_disjoint_unknown_runtime_overlap.concept", Expected: ConformancePass, DiagnosticCategory: "CV4641", MatrixStatus: "EVT1-new"},
	{Name: "runtime tensor not fixed", Source: "invalid/fact_fixed_shape_runtime_tensor.concept", Expected: ConformancePass, DiagnosticCategory: "CV4641", MatrixStatus: "EVT1-new"},
	{Name: "invalid subject", Source: "invalid/fact_invalid_subject.concept", Expected: ConformancePass, DiagnosticCategory: "CV4642", MatrixStatus: "EVT1-new"},
	{Name: "invalid alignment", Source: "invalid/fact_invalid_alignment_parameter.concept", Expected: ConformancePass, DiagnosticCategory: "CV4643", MatrixStatus: "EVT1-new"},
}

func generateR4jFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "semantic-facts", class, file)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	return outputs
}

func semanticFactsMIR(t *testing.T, file string) (MIR, Outputs) {
	t.Helper()
	outputs := generateR4jFixture(t, "valid", file)
	base := strings.TrimSuffix(file, ".concept")
	var mir MIR
	if err := json.Unmarshal(outputs[base+".mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	return mir, outputs
}

func TestSemanticFactsConformance(t *testing.T) {
	counts := map[ConformanceExpectation]int{}
	for _, tc := range semanticFactConformanceCases {
		t.Run(tc.Name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "semantic-facts", filepath.FromSlash(tc.Source))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			if tc.Accepted {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != tc.DiagnosticCategory {
				t.Fatalf("diagnostic = %v, want %s", err, tc.DiagnosticCategory)
			}
			if diagnostic.SemanticCategory() == diagnostic.Code {
				t.Fatalf("diagnostic %s has no semantic family", diagnostic.Code)
			}
		})
		counts[tc.Expected]++
	}
	if counts[ConformancePass] != 24 || len(counts) != 1 {
		t.Fatalf("classification drift: %+v", counts)
	}
}

func TestSemanticFactsProofsAreTypedAndDeterministic(t *testing.T) {
	first := generateR4jFixture(t, "valid", "fact_concept_aligned.concept")
	second := generateR4jFixture(t, "valid", "fact_concept_aligned.concept")
	if string(first["fact_concept_aligned.mir.json"]) != string(second["fact_concept_aligned.mir.json"]) {
		t.Fatal("semantic proof MIR is nondeterministic")
	}
	var mir MIR
	if err := json.Unmarshal(first["fact_concept_aligned.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.SemanticProofs) != 1 {
		t.Fatalf("proof count = %d", len(mir.SemanticProofs))
	}
	proof := mir.SemanticProofs[0]
	if proof.FactKind != FactAligned || proof.Outcome != string(FactProven) || proof.Origin != FactOriginType || len(proof.Parameters) != 1 || proof.Parameters[0] != 16 || proof.Evidence == nil || proof.Evidence.Alignment != 64 {
		t.Fatalf("incomplete aligned proof: %+v", proof)
	}
}

func TestSemanticFactsLayoutStreamSpanTensorPreservation(t *testing.T) {
	mir, outputs := semanticFactsMIR(t, "fact_preservation_chain.concept")
	facts := NewSemanticFactSet(mir.SemanticFacts)
	if facts.AreDisjoint("Buffers.input", "Buffers.output") != FactProven {
		t.Fatal("declared layout regions were not proven disjoint")
	}
	if region, ok := facts.RegionOf("BufferStream.Input"); !ok || region != "Buffers.input" {
		t.Fatalf("stream region = %q, %v", region, ok)
	}
	if alignment, ok := facts.KnownAlignment("BufferStream.Input"); !ok || alignment != 64 {
		t.Fatalf("stream alignment = %d, %v", alignment, ok)
	}
	spanFacts, tensorFacts := false, false
	for _, fact := range mir.SemanticFacts {
		if fact.Kind == FactAligned && fact.Evidence.Alignment == 64 && fact.Origin == FactOriginSpan {
			spanFacts = true
		}
		if fact.Kind == FactAligned && fact.Evidence.Alignment == 64 && fact.Origin == FactOriginTensorBacking {
			tensorFacts = true
		}
	}
	if !spanFacts || !tensorFacts {
		t.Fatalf("fact chain lost alignment: span=%v tensor=%v", spanFacts, tensorFacts)
	}
	generated := string(outputs["fact_preservation_chain.generated.c"])
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "semantic_fact", "noalias", "restrict"} {
		if strings.Contains(generated, forbidden) {
			t.Fatalf("runtime fact mechanism %q appeared", forbidden)
		}
	}
}

func TestSemanticFactsAlignmentDegradesAndIntervalsProveDisjoint(t *testing.T) {
	mir, _ := semanticFactsMIR(t, "fact_alignment_subspan.concept")
	degraded := false
	for _, fact := range mir.SemanticFacts {
		if fact.Kind == FactAligned && fact.Origin == FactOriginSpan && fact.Evidence.Detail == "span_subregion" && fact.Evidence.Alignment == 4 {
			degraded = true
		}
	}
	if !degraded {
		t.Fatal("offset-one subspan retained stale stronger alignment")
	}
	disjointMIR, _ := semanticFactsMIR(t, "fact_disjoint_subspans.concept")
	found := false
	for _, fact := range disjointMIR.SemanticFacts {
		if fact.Kind == FactDisjoint && fact.Evidence.Detail == "non-overlapping proven half-open intervals" {
			found = true
		}
	}
	if !found {
		t.Fatal("fixed non-overlapping subspans were not proven disjoint")
	}
}

func TestSemanticFactsTensorBackingQualification(t *testing.T) {
	fixed, _ := semanticFactsMIR(t, "fact_disjoint_inline_tensors.concept")
	fixedCount, disjoint := 0, false
	for _, fact := range fixed.SemanticFacts {
		if fact.Kind == FactFixedShape && fact.Origin == FactOriginTensorBacking {
			fixedCount++
		}
		if fact.Kind == FactDisjoint && fact.Origin == FactOriginTensorBacking {
			disjoint = true
		}
	}
	if fixedCount < 3 || !disjoint {
		t.Fatalf("inline tensor facts incomplete: fixed=%d disjoint=%v", fixedCount, disjoint)
	}
	runtime, _ := semanticFactsMIR(t, "fact_runtime_shape_bound_tensor.concept")
	runtimeShape := false
	for _, fact := range runtime.SemanticFacts {
		if fact.Kind == FactRuntimeShape && fact.Origin == FactOriginTensorBacking {
			runtimeShape = true
		}
	}
	if !runtimeShape {
		t.Fatal("bound runtime tensor lacks RuntimeShape")
	}
}

func TestSemanticFactsMIRRejectsIncompleteFact(t *testing.T) {
	err := evt1ValidateMIR(MIR{SemanticFacts: []MIRSemanticFact{{Kind: FactContiguous, Outcome: FactProven}}})
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4649" || diagnostic.SemanticCategory() != "SEMANTIC_FACT_MIR_INVALID" {
		t.Fatalf("unexpected semantic fact MIR diagnostic: %v", err)
	}
}
