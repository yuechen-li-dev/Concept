package concept

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestGeneratedFunctionEntersOrdinaryPipeline(t *testing.T) {
	source := `module GeneratedExample;
profile Core;
struct Item { int value; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) { return item.value; }
derive DeriveValue reflect<Item>;
int Main() { Item item = Item{7}; return ReadValue(ref const item); }
`
	module, err := Parse("generated.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Functions) != 2 || module.Functions[1].Generated == nil {
		t.Fatalf("generated function absent: %+v", module.Functions)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	implementation := moduleOutput(t, outputs, ".generated.c")
	for _, marker := range []string{"GeneratedByReflection", "FieldInfo", "Fields<"} {
		if strings.Contains(implementation, marker) {
			t.Fatalf("reflection metadata leaked into C: %s", marker)
		}
	}
}

func TestGeneratedFieldIterationIsStructural(t *testing.T) {
	source := `module GeneratedFields;
profile Core;
struct Item { [[chosen]] int left; int ignored; [[chosen]] int right; }
generator <typename T> DeriveSum
int Sum(ref const T item)
{
    int result = 0;
    foreach (FieldInfo field in Fields<T>(chosen)) {
        result = result + item.field;
    }
    return result;
}

derive DeriveSum reflect<Item>;
int Main() { Item item = Item{1, 99, 2}; return Sum(ref const item); }
`
	module, err := Parse("generated_fields.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Functions) != 2 || module.Functions[1].Generated == nil || len(module.Functions[1].Generated.Inputs) != 2 {
		t.Fatalf("field generation: %+v", module.Functions)
	}
	if len(module.Functions[1].Body.Statements) != 4 {
		t.Fatalf("expected ordinary expanded statements: %+v", module.Functions[1].Body)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedFunctionSurvivesArtifactOnlyImports(t *testing.T) {
	sourceA := `module A;
profile Core;
[[reflect]] struct Item { [[chosen]] int left; int ignored; [[chosen]] int right; }
generator <typename T> DeriveSum
int Sum(ref const T item) {
    int result = 0;
    foreach (FieldInfo field in Fields<T>(chosen)) { result = result + item.field; }
    return result;
}
`
	a, err := CompileSemanticModule("A.concept", sourceA, nil)
	if err != nil {
		t.Fatal(err)
	}
	sourceB := `module B; profile Core; import A;
derive DeriveSum reflect<Item>;
`
	b, err := CompileSemanticModule("B.concept", sourceB, map[string][]byte{"A": a})
	if err != nil {
		t.Fatal(err)
	}
	metadata, moduleB, err := LoadSemanticModuleArtifact(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Exports.Functions) != 1 || metadata.Exports.Functions[0] != "Sum" || len(moduleB.Functions) != 1 || moduleB.Functions[0].Generated == nil {
		t.Fatalf("generated export missing: %+v %+v", metadata.Exports, moduleB.Functions)
	}
	sourceC := `module C; profile Core; import B;
int Main() { Item item = Item{1, 99, 2}; return Sum(ref const item); }
`
	moduleC, err := ParseWithSemanticModules("C.concept", sourceC, map[string][]byte{"A": a, "B": b})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(moduleC, []byte(sourceC)); err != nil {
		t.Fatal(err)
	}
}

func TestGeneratedInspectionAndProofRetainProvenance(t *testing.T) {
	source := `module GeneratedProof; profile Core;
struct Item { [[chosen]] int value; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) {
    int result = 0;
    foreach (FieldInfo field in Fields<T>(chosen)) { result = result + item.field; }
    return result;
}
derive DeriveValue reflect<Item>;
`
	module, err := Parse("proof.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	view, err := InspectGeneratedDeclarations(module, "ReadValue")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"name": "ReadValue"`, `"generator": "DeriveValue"`, `"reflected_type"`, `"value"`} {
		if !strings.Contains(string(view), fragment) {
			t.Fatalf("missing %q in %s", fragment, view)
		}
	}
	graph, err := ExplainGeneratedDeclaration(module, "ReadValue")
	if err != nil {
		t.Fatal(err)
	}
	if graph.Outcome != FactProven || graph.Generated == nil || graph.Generated.Generator != "DeriveValue" || len(graph.Generated.Inputs) != 1 {
		t.Fatalf("generated proof: %+v", graph)
	}
}

func TestGeneratedConceptProofUsesRequiredOperation(t *testing.T) {
	source := `module GeneratedConcept; profile Core;
struct Item { int value; }
concept Readable<T> { requires int ReadValue(ref const T item); }
generator <typename T> DeriveValue
int ReadValue(ref const T item) { return item.value; }
derive DeriveValue reflect<Item>;
`
	module, err := Parse("concept.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ExplainGeneratedConcept(module, "Readable<Item>")
	if err != nil {
		t.Fatal(err)
	}
	if graph.Outcome != FactProven || graph.Goal != "Readable<Item>" || graph.Generated == nil || graph.Generated.Generator != "DeriveValue" {
		t.Fatalf("generated concept proof: %+v", graph)
	}
}

func TestGeneratedDeclarationConflictsDiagnoseBothSites(t *testing.T) {
	base := `module GeneratedConflict; profile Core;
struct Item { int value; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) { return item.value; }
`
	for _, suffix := range []string{
		"derive DeriveValue reflect<Item>;\nderive DeriveValue reflect<Item>;\n",
		"int ReadValue(ref const Item item) { return item.value; }\nderive DeriveValue reflect<Item>;\n",
	} {
		_, err := Parse("conflict.concept", base+suffix)
		if err == nil || !strings.Contains(err.Error(), "GENERATOR_DECLARATION_CONFLICT") || !strings.Contains(err.Error(), "generated by DeriveValue") {
			t.Fatalf("conflict diagnostic: %v", err)
		}
	}
}

func TestGeneratedInvalidBodyUsesOrdinaryChecker(t *testing.T) {
	source := `module GeneratedInvalid; profile Core;
struct Item { int value; }
generator <typename T> DeriveInvalid
int ReadValue(ref const T item) { return MissingOperation(item.value); }
derive DeriveInvalid reflect<Item>;
`
	_, err := Parse("invalid.concept", source)
	if err == nil || !strings.Contains(err.Error(), "MissingOperation") || !strings.Contains(err.Error(), "generated ReadValue by DeriveInvalid") {
		t.Fatalf("expected ordinary body diagnostic, got %v", err)
	}
}

func TestGeneratedUninitializedLocalUsesOrdinaryChecker(t *testing.T) {
	source := `module GeneratedUninitialized; profile Core;
record struct Person { int id; bool active; }
generator <typename T> DeriveRead
T ReadOctagon() { T result; result.id = 1; result.active = true; return result; }
derive DeriveRead reflect<Person>;
`
	_, err := Parse("uninitialized_generated.concept", source)
	if err == nil || !strings.Contains(err.Error(), "CV4560") || !strings.Contains(err.Error(), "generated ReadOctagon by DeriveRead") {
		t.Fatalf("expected ordinary initializer diagnostic for generated local, got %v", err)
	}
}

func TestGeneratedStructuralInvalidInputs(t *testing.T) {
	cases := []struct{ source, code string }{
		{`module InvalidQuery; profile Core;
struct Item { int value; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) {
    foreach (FieldInfo field in Fields<T>(0)) { return item.field; }
    return 0;
}
derive DeriveValue reflect<Item>;
`, "GENERATOR_FIELD_QUERY_INVALID"},
		{`module RecursiveGenerator; profile Core;
struct Item { int value; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) { derive DeriveValue reflect<Item>; return item.value; }
derive DeriveValue reflect<Item>;
`, "GENERATOR_RECURSION_UNSUPPORTED"},
	}
	for _, tc := range cases {
		_, err := Parse("invalid_generated.concept", tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("expected %s, got %v", tc.code, err)
		}
	}
}

func TestGeneratedDeclarationHundredRunDeterminism(t *testing.T) {
	source := `module GeneratedDeterminism; profile Core;
struct Item { [[chosen]] int value; }
generator <typename T> DeriveValue
int ReadValue(ref const T item) {
    int result = 0;
    foreach (FieldInfo field in Fields<T>(chosen)) { result = result + item.field; }
    return result;
}

derive DeriveValue reflect<Item>;
reflect<Item>;
`
	var wantArtifact, wantView, wantProof, wantReflection []byte
	var wantOutputs Outputs
	for run := 0; run < 100; run++ {
		artifact, err := CompileSemanticModule("determinism.concept", source, nil)
		if err != nil {
			t.Fatal(err)
		}
		module, err := Parse("determinism.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		view, err := InspectGeneratedDeclarations(module, "ReadValue")
		if err != nil {
			t.Fatal(err)
		}
		graph, err := ExplainGeneratedDeclaration(module, "ReadValue")
		if err != nil {
			t.Fatal(err)
		}
		proof, err := SerializeProof(graph)
		if err != nil {
			t.Fatal(err)
		}
		reflection, err := json.Marshal(module.ReflectionResults)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			wantArtifact, wantView, wantProof, wantReflection, wantOutputs = artifact, view, proof, reflection, outputs
			continue
		}
		if !bytes.Equal(artifact, wantArtifact) || !bytes.Equal(view, wantView) || !bytes.Equal(proof, wantProof) || !bytes.Equal(reflection, wantReflection) {
			t.Fatalf("generated identity, artifact, view, or proof changed on run %d", run)
		}
		for name, expected := range wantOutputs {
			if !bytes.Equal(outputs[name], expected) {
				t.Fatalf("%s changed on run %d", name, run)
			}
		}
	}
}
