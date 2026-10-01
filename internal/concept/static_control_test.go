package concept

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestR9aRuntimeStaticControlArtifactAndNative(t *testing.T) {
	source, err := os.ReadFile("../../language/evt1/generic-library-closure/valid/runtime_static_control.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "static_control.concept", string(source), nil)
	consumer := `module Closure.StaticUse; profile Core; import Closure.StaticControl;
int Use(int value) { return Select<int>(value) + Select<double>(value) + OnlyFour<int>(value) + Repeat<4>(value) + Fixed(value); }`
	for _, verify := range []bool{false, true} {
		module, err := ParseWithSemanticModules("static_use.concept", consumer, map[string][]byte{"Closure.StaticControl": artifact})
		if err != nil {
			t.Fatal(err)
		}
		policy := ConservativeCompilationPolicy()
		if verify {
			policy = VerifyCompilationPolicy()
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(consumer), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		body := moduleOutput(t, outputs, ".generated.c")
		if strings.Contains(body, "NotAnOperation") || strings.Contains(body, "for (") || strings.Contains(body, "if (sizeof") {
			t.Fatalf("static control survived in runtime C: %s", body)
		}
		assertR8cStrictC11(t, outputs, "static_use.generated.c")
		runFoundationNativeHarness(t, outputs, "static_harness.c", `#include "static_use.generated.h"
int main(void) { return concept_closure__static_use_use(3) == 58 ? 0 : 1; }`)
	}
	_, err = ParseWithSemanticModules("bad_static_use.concept", `profile Core; import Closure.StaticControl; int Bad() { return OnlyFour<double>(3); }`, map[string][]byte{"Closure.StaticControl": artifact})
	if err == nil || !strings.Contains(err.Error(), "NotAnOperation") {
		t.Fatalf("selected invalid branch was accepted: %v", err)
	}
}

func TestR9aRuntimeStaticControlDiagnostics(t *testing.T) {
	cases := []struct{ name, source, code string }{
		{"runtime condition", `int Bad(int value) { comptime if (value > 0) { return 1; } return 0; }`, "CV4200"},
		{"runtime inferred initializer", `int Bad(int value) { comptime auto constant = value; return constant; }`, "CV4200"},
		{"runtime loop", `int Bad(int value) { comptime for (i in 0..value) { value += i; } return value; }`, "CV4200"},
		{"mixed open condition", `template <typename T> int Bad(int value) { comptime if (SizeOf<T>() == 4 and value > 0) { return 1; } return 0; }`, "CV4200"},
		{"selected invalid", `int Bad() { comptime if (false) { return 1; } else { return Missing(); } }`, "CV"},
		{"non bool", `int Bad() { comptime if (1) { return 1; } return 0; }`, "CV4186"},
		{"iteration limit", `int Bad() { comptime for (i in 0..257) { int value = i; } return 0; }`, "CV4206"},
		{"reference item", `int Bad() { comptime for (ref int i in 0..3) { int value = i; } return 0; }`, "FOREACH_ITERATOR_INVALID"},
		{"wrong item type", `int Bad() { comptime for (double i in 0..3) { int value = 0; } return 0; }`, "FOREACH_ITEM_TYPE_MISMATCH"},
		{"total expansion fuel", `int Bad() { comptime for (i in 0..256) { comptime for (j in 0..256) { int value = i + j; } } return 0; }`, "CV4204"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("static_bad.concept", "profile Core; "+tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestR9aComptimeAutoAndVarCompatibility(t *testing.T) {
	for _, spelling := range []string{"auto", "var", "var int<array>[3]"} {
		t.Run(spelling, func(t *testing.T) {
			source := "profile Core; int Main() { comptime " + spelling + " steps = [1, 2, 3]; int total = 0; comptime for (i in steps) { total += i; } return total; }"
			module, err := Parse("auto_alias.concept", source)
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, []byte(source))
			if err != nil {
				t.Fatal(err)
			}
			assertR8cStrictC11(t, outputs, "auto_alias.generated.c")
			runFoundationNativeHarness(t, outputs, "auto_alias_harness.c", "#include \"auto_alias.generated.h\"\nint main(void) { return concept_auto_alias_main() == 6 ? 0 : 1; }\n")
		})
	}
}

func TestR9aStaticControlPreservesAuthoritativeAllocationProof(t *testing.T) {
	source := `module Static.Effect; profile Core; concept Hot<declaration F> { requires compiler.NoAllocation(F); }
int Fast() { comptime if (true) { return 1; } else { return Missing(); } }
void Check() { Assert.Concept<Hot>(Fast, "selected body only"); }`
	if _, err := Parse("static_effect.concept", source); err != nil {
		t.Fatal(err)
	}
}

func TestR9aStaticControlArtifactDeterminism100(t *testing.T) {
	source := `module Closure.Deferred; profile Core; template <typename T> int Pick(int value) { comptime if (SizeOf<T>() == 4) { return value + 1; } else { return Missing(value); } }`
	consumer := `profile Core; import Closure.Deferred; int Use(int value) { return Pick<int>(value); }`
	var firstArtifact, firstC, firstMIR []byte
	for run := 0; run < 100; run++ {
		artifact := buildSemanticArtifact(t, "deferred.concept", source, nil)
		module, err := ParseWithSemanticModules("deferred_use.concept", consumer, map[string][]byte{"Closure.Deferred": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(consumer))
		if err != nil {
			t.Fatal(err)
		}
		c, mir := []byte(moduleOutput(t, outputs, ".generated.c")), []byte(moduleOutput(t, outputs, ".mir.json"))
		if run == 0 {
			firstArtifact, firstC, firstMIR = artifact, c, mir
		}
		if !bytes.Equal(firstArtifact, artifact) || !bytes.Equal(firstC, c) || !bytes.Equal(firstMIR, mir) {
			t.Fatalf("static specialization differs at run %d", run)
		}
	}
}
