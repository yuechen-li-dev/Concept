package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

const r9bStaticExtentSource = `module Research.Extent; profile Core;
using Array4 = int<array>[4];
int Read(ref const Array4 values) {
    Assert.Concept<StaticExtent<4>>(Array4, "fixed storage extent");
    return values[1];
}
`

func TestR9bStaticExtentAssertionsAndArtifact(t *testing.T) {
	module, err := Parse("extent.concept", r9bStaticExtentSource)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		n    int
		want SemanticFactCertainty
	}{{4, FactProven}, {3, FactDisproven}} {
		result := evt1TypeFact(env, FactStaticExtent, env.typeAliases["Array4"], []int{tc.n})
		if result.Outcome != tc.want || result.Origin != FactOriginLayout || result.Evidence.Extent != 4 {
			t.Fatalf("extent %d: %+v", tc.n, result)
		}
	}
	artifact := buildSemanticArtifact(t, "extent.concept", r9bStaticExtentSource, nil)
	consumer := `module Research.ExtentUse; profile Core; import Research.Extent;
int Main() { Assert.Concept<StaticExtent<4>>(Array4, "artifact fixed storage"); return 0; }
`
	if _, err := ParseWithSemanticModules("extent_use.concept", consumer, map[string][]byte{"Research.Extent": artifact}); err != nil {
		t.Fatal(err)
	}
	bad := strings.Replace(r9bStaticExtentSource, "StaticExtent<4>", "StaticExtent<3>", 1)
	_, err = Parse("bad_extent.concept", bad)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") || !strings.Contains(err.Error(), "StaticExtent") {
		t.Fatalf("extent mismatch was accepted: %v", err)
	}
}

func TestR9bStaticExtentPlannerShadow(t *testing.T) {
	source := `profile Core;
int Sum() { vector<int> A[4] = [1,2,3,4]; vector<int> B[4] = [5,6,7,8]; vector<int> C[4] = 0; C = A + B; return C[1]; }
`
	module, err := Parse("extent_tensor.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["extent_tensor.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	qualified := map[string]bool{}
	for _, fact := range mir.SemanticFacts {
		if fact.Kind != FactStaticExtent {
			continue
		}
		if fact.Origin != FactOriginLayout || len(fact.Parameters) != 1 || len(fact.Evidence.Shape) != 1 || fact.Parameters[0] != fact.Evidence.Shape[0].Extent || fact.Evidence.Shape[0].Runtime {
			t.Fatalf("extent diverged from fixed-shape authority: %+v", fact)
		}
		qualified[fact.ID] = true
	}
	if len(qualified) == 0 {
		t.Fatal("no inline extent facts")
	}
	seen := false
	for _, fn := range mir.Functions {
		for i, operation := range fn.TensorOperations {
			tensor := planTensor(i, operation, NewSemanticFactSet(mir.SemanticFacts))
			for _, id := range tensor.Vectorization.Evidence.FactIDs {
				seen = seen || qualified[id]
			}
			if tensor.Vectorization.Selected {
				t.Fatal("research fact enabled SIMD")
			}
		}
	}
	if !seen {
		t.Fatal("Planner lost the structural extent evidence")
	}
	runFoundationNativeHarness(t, outputs, "extent_tensor_harness.c", "#include \"extent_tensor.generated.h\"\nint main(void) { return "+evt1FunctionSymbol(evt1SemanticSymbolBase(module), "Sum")+"() == 8 ? 0 : 1; }\n")
}

func TestR9bStaticExtentUnknownAndDeterminism(t *testing.T) {
	source := r9bStaticExtentSource + `
using Partial = int<raw>[4];
using Grid = int<ndarray>[2,2];
int View(Span<int> input) { return 0; }
`
	module, err := Parse("extent.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	for _, subject := range []Type{env.typeAliases["Partial"], env.typeAliases["Grid"], env.functions["View"][0].Params[0].Type} {
		if result := evt1TypeFact(env, FactStaticExtent, subject, []int{4}); result.Outcome != FactUnknown {
			t.Fatalf("capacity or view became exact live extent: %+v", result)
		}
	}
	first, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		parsed, err := Parse("extent.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(parsed, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(outputs["extent.mir.json"], first["extent.mir.json"]) {
			t.Fatalf("extent truth/proof/facts changed on run %d", run)
		}
	}
}

func TestR9bDeclaredPredicateCannotGrantExtent(t *testing.T) {
	source := `profile Core;
comptime bool Yes(typename subject) { return true; }
concept ExtentClaim<T> { requires Yes(T); }
int View(Span<int> input) { Assert.Concept<ExtentClaim>(input, "descriptive claim only"); return 0; }
`
	module, err := Parse("extent_claim.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["extent_claim.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fact := range mir.SemanticFacts {
		if fact.Kind == FactStaticExtent {
			t.Fatalf("declared predicate forged structural extent: %+v", fact)
		}
	}
}
