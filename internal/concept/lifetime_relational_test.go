package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type relationalLifetimeConformanceCase struct {
	name             string
	source           string
	accepted         bool
	diagnosticFamily string
	classification   ConformanceExpectation
	provenance       string
}

var relationalLifetimeConformanceCases = []relationalLifetimeConformanceCase{
	{name: "call result from parameter", source: "valid/call_result_from_parameter.concept", accepted: true, classification: ConformancePass, provenance: "R4a foundation plus EVT1-new relational semantics"},
	{name: "call result ref struct", source: "valid/call_result_ref_struct.concept", accepted: true, classification: ConformancePass, provenance: "R4a foundation plus EVT1-new relational semantics"},
	{name: "outlives requirement pass", source: "valid/outlives_requirement_pass.concept", accepted: true, classification: ConformancePass, provenance: "EVT1-new relational semantics"},
	{name: "scoped result non escaping", source: "valid/scoped_result_non_escaping.concept", accepted: true, classification: ConformancePass, provenance: "R4a foundation plus EVT1-new relational semantics"},
	{name: "nested ref struct provenance", source: "valid/nested_ref_struct_provenance.concept", accepted: true, classification: ConformancePass, provenance: "EVT1-new relational semantics"},
	{name: "multi source shortest lifetime", source: "valid/multi_source_shortest_lifetime.concept", accepted: true, classification: ConformancePass, provenance: "EVT1-new relational semantics"},
	{name: "call result escape source", source: "invalid/call_result_escape_source.concept", diagnosticFamily: "CALL_RESULT_ESCAPES_SOURCE", classification: ConformancePass, provenance: "EVT1-new relational semantics"},
	{name: "outlives requirement fail", source: "invalid/outlives_requirement_fail.concept", diagnosticFamily: "OUTLIVES_REQUIREMENT_FAILED", classification: ConformancePass, provenance: "EVT1-new relational semantics"},
	{name: "call result unknown provenance", source: "invalid/call_result_unknown_provenance.concept", diagnosticFamily: "OUTLIVES_PROVENANCE_UNKNOWN", classification: ConformancePass, provenance: "EVT1-new relational semantics"},
	{name: "scoped result escape", source: "invalid/scoped_result_escape.concept", diagnosticFamily: "SCOPED_RESULT_ESCAPE", classification: ConformancePass, provenance: "R4a foundation plus EVT1-new relational semantics"},
	{name: "shorter result into longer place", source: "invalid/shorter_result_into_longer_place.concept", diagnosticFamily: "LIFETIME_OUTLIVES_VIOLATION", classification: ConformancePass, provenance: "R4a foundation plus EVT1-new relational semantics"},
}

func TestRelationalLifetimeConformance(t *testing.T) {
	accepted, rejected := 0, 0
	for _, tc := range relationalLifetimeConformanceCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.classification == "" || tc.provenance == "" {
				t.Fatal("classification and provenance are required")
			}
			path := filepath.Join("..", "..", "language", "evt1", "lifetime", "relational", filepath.FromSlash(tc.source))
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if tc.accepted {
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Generate(module, source); err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid R4b case was accepted")
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != tc.diagnosticFamily {
				t.Fatalf("diagnostic family = %s, want %s: %v", diagnostic.SemanticCategory(), tc.diagnosticFamily, err)
			}
		})
		if tc.accepted {
			accepted++
		} else {
			rejected++
		}
	}
	if accepted != 6 || rejected != 5 {
		t.Fatalf("R4b count drift: valid=%d invalid=%d", accepted, rejected)
	}
}

func TestRelationalLifetimeMIRRelationalEvidence(t *testing.T) {
	outputs := generateRelationalLifetimeFixture(t, "outlives_requirement_pass.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["outlives_requirement_pass.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	var makeView *MIRFunction
	for i := range mir.Functions {
		if mir.Functions[i].Name == "MakeView" {
			makeView = &mir.Functions[i]
		}
	}
	if makeView == nil || makeView.ResultProvenance == nil || makeView.ResultProvenance.Kind != "parameter" || len(makeView.ResultProvenance.ParameterIndices) != 1 || makeView.ResultProvenance.ParameterIndices[0] != 0 {
		t.Fatalf("parameter-derived result summary missing: %+v", makeView)
	}
	if len(mir.SemanticProofs) != 1 {
		t.Fatalf("expected one demand-driven proof, got %+v", mir.SemanticProofs)
	}
	proof := mir.SemanticProofs[0]
	if proof.Analysis != "Outlives" || proof.Outcome != "proven" || !proof.Satisfied || len(proof.Subjects) != 2 || proof.Subjects[0].Kind != "parameter" || proof.Subjects[1].Kind != "result" || len(proof.ProvenanceFacts) != 1 {
		t.Fatalf("relational proof evidence missing: %+v", proof)
	}
	if proof.Subjects[0].ParameterIndex == nil || *proof.Subjects[0].ParameterIndex != 0 || proof.ID == "" {
		t.Fatalf("deterministic proof identity lacks its parameter subject: %+v", proof)
	}

	second := generateRelationalLifetimeFixture(t, "outlives_requirement_pass.concept")
	if !bytes.Equal(outputs["outlives_requirement_pass.mir.json"], second["outlives_requirement_pass.mir.json"]) {
		t.Fatal("R4b MIR proof serialization is not deterministic")
	}
}

func TestRelationalLifetimeNestedResultSummary(t *testing.T) {
	outputs := generateRelationalLifetimeFixture(t, "nested_ref_struct_provenance.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["nested_ref_struct_provenance.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fn := range mir.Functions {
		if fn.Name == "MakeOuter" && fn.ResultProvenance != nil && fn.ResultProvenance.Kind == "parameter" && len(fn.ResultProvenance.ParameterIndices) == 1 && fn.ResultProvenance.ParameterIndices[0] == 0 {
			return
		}
	}
	t.Fatal("nested ref-struct result did not preserve parameter provenance")
}

func TestRelationalLifetimePassThroughResultSummary(t *testing.T) {
	outputs := generateRelationalLifetimeFixture(t, "call_result_ref_struct.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["call_result_ref_struct.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fn := range mir.Functions {
		if fn.Name == "ForwardCursor" && fn.ResultProvenance != nil && fn.ResultProvenance.Kind == "parameter" && len(fn.ResultProvenance.ParameterIndices) == 1 && fn.ResultProvenance.ParameterIndices[0] == 0 {
			return
		}
	}
	t.Fatal("direct pass-through call did not preserve parameter provenance")
}

func TestRelationalLifetimeShortestOfParametersSummary(t *testing.T) {
	outputs := generateRelationalLifetimeFixture(t, "multi_source_shortest_lifetime.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["multi_source_shortest_lifetime.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fn := range mir.Functions {
		if fn.Name == "MakePair" && fn.ResultProvenance != nil && fn.ResultProvenance.Kind == "shortest_of_parameters" && len(fn.ResultProvenance.ParameterIndices) == 2 && fn.ResultProvenance.ParameterIndices[0] == 0 && fn.ResultProvenance.ParameterIndices[1] == 1 {
			return
		}
	}
	t.Fatal("multi-source result did not record the shortest parameter set")
}

func TestRelationalLifetimeRelationalSubjectValidation(t *testing.T) {
	source := []byte("profile Core; ref struct View { ref const int data; } View Make(ref const int source); concept Bad<T> { requires T Make(ref const int source); requires compiler.Outlives(missing, result); }")
	_, err := Parse("invalid_subject.concept", string(source))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "RELATIONAL_REQUIREMENT_INVALID_SUBJECT" {
		t.Fatalf("invalid relational subject was not diagnosed: %v", err)
	}
}

func TestRelationalLifetimeUnknownCallResultAssignment(t *testing.T) {
	source := []byte("profile Core; ref struct View { ref const int data; } View Unknown(ref const int source); int Bad() { int outerSource = 1; View outer = View{ref const outerSource}; { int innerSource = 2; outer = Unknown(ref const innerSource); } return outer.data; }")
	_, err := Parse("unknown_call_assignment.concept", string(source))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "CALL_RESULT_LIFETIME_UNKNOWN" {
		t.Fatalf("unknown call-result assignment was not diagnosed: %v", err)
	}
}

func TestRelationalLifetimeByValueParameterIsNotReturnableStorage(t *testing.T) {
	source := []byte("profile Core; ref const int Bad(int value) { return ref const value; }")
	_, err := Parse("by_value_parameter_escape.concept", string(source))
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "REF_ESCAPE_LOCAL" {
		t.Fatalf("reference to by-value parameter storage escaped: %v", err)
	}
}

func TestRelationalLifetimeNativeC11(t *testing.T) {
	cases := []struct {
		fixture string
		call    string
		want    int
	}{
		{fixture: "call_result_from_parameter.concept", call: "concept_call_result_from_parameter_main", want: 42},
		{fixture: "nested_ref_struct_provenance.concept", call: "concept_nested_ref_struct_provenance_main", want: 42},
		{fixture: "outlives_requirement_pass.concept", call: "concept_outlives_requirement_pass_main", want: 42},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			outputs := generateRelationalLifetimeFixture(t, tc.fixture)
			base := strings.TrimSuffix(tc.fixture, ".concept")
			harness := "#include \"" + base + ".generated.h\"\n\nint main(void) { return " + tc.call + "() == " + strconv.Itoa(tc.want) + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, base+"_harness.c", harness)
		})
	}
}

func generateRelationalLifetimeFixture(t *testing.T, fixture string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "lifetime", "relational", "valid", fixture)
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
