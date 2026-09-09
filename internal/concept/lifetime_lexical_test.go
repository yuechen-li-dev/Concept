package concept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type lexicalLifetimeConformanceCase struct {
	name             string
	source           string
	accepted         bool
	diagnosticFamily string
}

var lexicalLifetimeConformanceCases = []lexicalLifetimeConformanceCase{
	{name: "local ref struct", source: "valid/ref_struct_local.concept", accepted: true},
	{name: "ref struct passed downward", source: "valid/ref_struct_pass_down.concept", accepted: true},
	{name: "ref struct returned from parameter", source: "valid/ref_struct_return_parameter.concept", accepted: true},
	{name: "scoped ref parameter", source: "valid/scoped_ref_parameter.concept", accepted: true},
	{name: "concept requested lifetime proof", source: "valid/lifetime_concept_satisfied.concept", accepted: true},
	{name: "live owned replacement", source: "valid/owned_replacement_drop.concept", accepted: true},
	{name: "ref struct local return", source: "invalid/ref_struct_return_local.concept", diagnosticFamily: "REF_STRUCT_ESCAPE"},
	{name: "scoped reference return", source: "invalid/scoped_ref_escape.concept", diagnosticFamily: "SCOPED_ESCAPE"},
	{name: "shorter reference assigned outward", source: "invalid/ref_struct_longer_storage.concept", diagnosticFamily: "LIFETIME_OUTLIVES_VIOLATION"},
	{name: "compiler lifetime requirement unsatisfied", source: "invalid/lifetime_concept_unsatisfied.concept", diagnosticFamily: "LIFETIME_REQUIREMENT_UNSATISFIED"},
	{name: "reference in unrestricted aggregate", source: "invalid/ref_struct_invalid_field.concept", diagnosticFamily: "REF_STRUCT_INVALID_FIELD"},
}

func TestLexicalLifetimeConformance(t *testing.T) {
	accepted, rejected := 0, 0
	for _, tc := range lexicalLifetimeConformanceCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "lifetime", "lexical", filepath.FromSlash(tc.source))
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
				t.Fatal("invalid R4a case was accepted")
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) {
				t.Fatalf("rejection was not a stable diagnostic: %v", err)
			}
			if diagnostic.SemanticCategory() != tc.diagnosticFamily {
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
		t.Fatalf("R4a count drift: valid=%d invalid=%d", accepted, rejected)
	}
}

func TestLexicalLifetimeMIRLifetimeAndProofFacts(t *testing.T) {
	outputs := generateLexicalLifetimeFixture(t, "lifetime_concept_satisfied.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["lifetime_concept_satisfied.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Structs) != 1 || !mir.Structs[0].Ref {
		t.Fatalf("ref struct MIR fact missing: %+v", mir.Structs)
	}
	if len(mir.SemanticProofs) == 0 || mir.SemanticProofs[0].Analysis != "LifetimeSafe" || !mir.SemanticProofs[0].Satisfied {
		t.Fatalf("concept-driven lifetime proof missing: %+v", mir.SemanticProofs)
	}
	foundAnalysis := false
	for _, req := range mir.Concepts[0].Requirements {
		foundAnalysis = foundAnalysis || req.Kind == "compiler_analysis" && req.Name == "LifetimeSafe"
	}
	if !foundAnalysis {
		t.Fatalf("compiler analysis requirement missing: %+v", mir.Concepts[0].Requirements)
	}
}

func TestLexicalLifetimeOwnedReplacementDropsOldFirst(t *testing.T) {
	body := string(generateLexicalLifetimeFixture(t, "owned_replacement_drop.concept")["owned_replacement_drop.generated.c"])
	mainAt := strings.Index(body, "int concept_owned_replacement_drop_main")
	if mainAt < 0 {
		t.Fatal("generated Main missing")
	}
	mainBody := body[mainAt:]
	dropAt := strings.Index(mainBody, "concept_owned_replacement_drop_drop(resource);")
	replaceAt := strings.Index(mainBody, "resource = concept_owned_replacement_drop_acquire_b();")
	if dropAt < 0 || replaceAt < 0 || dropAt > replaceAt || strings.Count(mainBody, "concept_owned_replacement_drop_drop(resource);") != 2 {
		t.Fatalf("replacement must drop old then assign new and later clean up once:\n%s", mainBody)
	}
}

func TestLexicalLifetimeMIRCleanupValidationRejectsInconsistentObligations(t *testing.T) {
	bad := MIR{Functions: []MIRFunction{{
		Name: "Bad",
		Cleanups: []MIRCleanup{
			{Owner: "resource", DropFunction: "Drop", State: "live", Order: 1},
			{Owner: "resource", DropFunction: "Drop", State: "maybe_moved", Order: 2},
		},
	}}}
	err := evt1ValidateMIR(bad)
	if err == nil {
		t.Fatal("inconsistent MIR cleanup obligations were accepted")
	}
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.SemanticCategory() != "DROP_OF_MOVED_VALUE" {
		t.Fatalf("unexpected MIR cleanup diagnostic: %v", err)
	}
}

func TestLexicalLifetimeNativeC11(t *testing.T) {
	cases := []struct {
		fixture, call string
		want          int
	}{
		{fixture: "ref_struct_local.concept", call: "concept_ref_struct_local_main", want: 42},
		{fixture: "scoped_ref_parameter.concept", call: "concept_scoped_ref_parameter_main", want: 42},
		{fixture: "ref_struct_return_parameter.concept", call: "concept_ref_struct_return_parameter_main", want: 42},
		{fixture: "owned_replacement_drop.concept", call: "concept_owned_replacement_drop_main", want: 42},
	}
	for _, tc := range cases {
		t.Run(tc.fixture, func(t *testing.T) {
			outputs := generateLexicalLifetimeFixture(t, tc.fixture)
			base := strings.TrimSuffix(tc.fixture, ".concept")
			harness := "#include \"" + base + ".generated.h\"\n\nint main(void) { return " + tc.call + "() == " + strconv.Itoa(tc.want) + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, base+"_harness.c", harness)
		})
	}
}

func generateLexicalLifetimeFixture(t *testing.T, fixture string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "lifetime", "lexical", "valid", fixture)
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
