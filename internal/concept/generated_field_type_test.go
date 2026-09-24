package concept

import (
	"bytes"
	"testing"
)

func TestGeneratedFieldTypeSelectsOrdinaryOverloads(t *testing.T) {
	const source = `module R7hFieldType; profile Core;
record struct Pair { int X; bool Y; }
template <typename T> struct TypeToken { int marker; };
int Read(TypeToken<int> token) { return 1; }
bool Read(TypeToken<bool> token) { return true; }
generator <typename T> DeriveMake
T Make() {
    return T{foreach (FieldInfo item in Fields<T>()) { item = Read(TypeToken<FieldType<item>>{0}); }};
}
derive DeriveMake reflect<Pair>;
int Main() { Pair pair = Make(); if (pair.Y) { return pair.X; } return 0; }
`
	module, err := Parse("R7hFieldType.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		again, err := Parse("R7hFieldType.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		current, err := Generate(again, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if len(current) != len(outputs) {
			t.Fatalf("generated output count changed on run %d", run+1)
		}
		for path, initial := range outputs {
			value, present := current[path]
			if !present || !bytes.Equal(initial, value) {
				t.Fatalf("generated output %s changed on run %d", path, run+1)
			}
		}
	}
	const harness = `#include "r7hfieldtype.generated.h"
int main(void) { return concept_r7h_field_type_main() == 1 ? 0 : 1; }
`
	runFoundationNativeHarness(t, outputs, "r7h_field_type_harness.c", harness)
}

func TestGeneratedFieldTypeSurvivesArtifactOnlyDerivation(t *testing.T) {
	const sourceA = `module FieldTypeA; profile Core;
[[reflect]] record struct Pair { int X; bool Y; }
`
	a, err := CompileSemanticModule("FieldTypeA.concept", sourceA, nil)
	if err != nil {
		t.Fatal(err)
	}
	const sourceB = `module FieldTypeB; profile Core; import FieldTypeA;
template <typename T> struct TypeToken { int marker; };
int Read(TypeToken<int> token) { return 3; }
bool Read(TypeToken<bool> token) { return true; }
generator <typename T> DeriveMake
T Make() { return T{foreach (FieldInfo item in Fields<T>()) { item = Read(TypeToken<FieldType<item>>{0}); }}; }
derive DeriveMake reflect<Pair>;
`
	b, err := CompileSemanticModule("FieldTypeB.concept", sourceB, map[string][]byte{"FieldTypeA": a})
	if err != nil {
		t.Fatal(err)
	}
	const sourceC = `module FieldTypeC; profile Core; import FieldTypeA; import FieldTypeB;
int Main() { Pair pair = Make(); if (pair.Y) { return pair.X; } return 0; }
`
	module, err := ParseWithSemanticModules("FieldTypeC.concept", sourceC, map[string][]byte{"FieldTypeA": a, "FieldTypeB": b})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(sourceC)); err != nil {
		t.Fatal(err)
	}
}
