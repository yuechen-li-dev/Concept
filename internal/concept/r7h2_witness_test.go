package concept

import (
	"bytes"
	"testing"
)

const r7h2WitnessSource = `profile Core;
concept Codec<T> { requires int Encode(ref const T value); }
template <typename E, usize N>
int Encode(ref const E<array>[N] value) { return Len(value); }
requires Codec<int<array>[4]>;
int Use()
{
    int<array>[4] values = [1, 2, 3, 4];
    return Encode(ref const values);
}
`

func TestR7h2ClosedArrayWitnessCall(t *testing.T) {
	module, err := Parse("r7h2_closed_array_witness.concept", r7h2WitnessSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(r7h2WitnessSource))
	if err != nil {
		t.Fatal(err)
	}
	for run := 1; run < 100; run++ {
		again, err := Parse("r7h2_closed_array_witness.concept", r7h2WitnessSource)
		if err != nil {
			t.Fatal(err)
		}
		current, err := Generate(again, []byte(r7h2WitnessSource))
		if err != nil {
			t.Fatal(err)
		}
		if len(current) != len(outputs) {
			t.Fatalf("output count changed on run %d", run+1)
		}
		for path, initial := range outputs {
			if !bytes.Equal(initial, current[path]) {
				t.Fatalf("output %s changed on run %d", path, run+1)
			}
		}
	}
	runFoundationNativeHarness(t, outputs, "r7h2_closed_array_witness_harness.c", "#include \"r7h2_closed_array_witness.generated.h\"\nint main(void) { return concept_r7h2_closed_array_witness_use() == 4 ? 0 : 1; }\n")
}

func TestR7h2ArtifactOnlyGeneratedWitnessCall(t *testing.T) {
	const sourceA = `module WitnessA; profile Core;
concept Codec<T> { requires int Encode(ref const T value); }
template <typename E, usize N>
int Encode(ref const E<array>[N] value) { return Len(value); }
`
	a, err := CompileSemanticModule("WitnessA.concept", sourceA, nil)
	if err != nil {
		t.Fatal(err)
	}
	const sourceB = `module WitnessB; profile Core; import WitnessA;
[[reflect]] record struct Packet { int<array>[4] Values; }
generator <typename T> DeriveCount
int Count(ref const T value) {
    int total = 0;
    foreach (FieldInfo field in Fields<T>()) { total = total + Encode(ref const value.field); }
    return total;
}
derive DeriveCount reflect<Packet>;
requires Codec<int<array>[4]>;
`
	artifacts := map[string][]byte{"WitnessA": a}
	b, err := CompileSemanticModule("WitnessB.concept", sourceB, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	artifacts["WitnessB"] = b
	const sourceC = `module WitnessC; profile Core; import WitnessA; import WitnessB;
int Main() { Packet packet = Packet{Values = [1, 2, 3, 4]}; return Count(ref const packet); }
`
	module, err := ParseWithSemanticModules("WitnessC.concept", sourceC, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(sourceC))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "r7h2_artifact_harness.c", "#include \"witnessc.generated.h\"\nint main(void) { return concept_witness_c_main() == 4 ? 0 : 1; }\n")
}
