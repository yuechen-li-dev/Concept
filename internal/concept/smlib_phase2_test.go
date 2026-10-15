package concept

import (
	"strings"
	"testing"
)

func TestSMLibPhase2ComptimeArrayRuntimeIndex(t *testing.T) {
	const library = `module Geometry.Coefficients; profile Core;
comptime int<array>[5] PascalRow = [1,4,6,4,1];
comptime double<array>[3] Scale = [0.5,1.0,2.0];
`
	artifact := buildSemanticArtifact(t, "Geometry/Coefficients.concept", library, nil)
	const source = `module Geometry.Consumer; profile Core;
import Geometry.Coefficients;
int Read(int i) { return PascalRow[i]; }
double Scaled(int i) { return Scale[i]; }
`
	module, err := ParseWithSemanticModules("consumer.concept", source, map[string][]byte{"Geometry.Coefficients": artifact})
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "consumer.generated.c")
		runFoundationNativeHarness(t, outputs, "coefficients.c", `#include "consumer.generated.h"
int main(void) {
  return concept_geometry__consumer_read(2) == 6 && concept_geometry__consumer_scaled(0) == 0.5 ? 0 : 1;
}`)
	}
}

func TestSMLibPhase2ComptimeArrayBoundsStillReject(t *testing.T) {
	_, err := Parse("bad.concept", `profile Core;
comptime int<array>[3] Values = [1,2,3];
int Read() { return Values[3]; }
`)
	if err == nil || !strings.Contains(err.Error(), "CV4233") {
		t.Fatalf("constant array bounds must still reject: %v", err)
	}
}
