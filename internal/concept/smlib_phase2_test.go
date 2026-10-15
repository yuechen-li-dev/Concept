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

func TestSMLibPhase2GenericSubspanParameterFacts(t *testing.T) {
	const library = `module Geometry.Slices; profile Core;
template <usize N>
double Slice(Span<double> input, int offset) {
    Span<double> part = Subspan(input, offset, N as int);
    Assert.Concept<Contiguous>(part, "subspan of generic parameter retains contiguity");
    part[0] = 7.0;
    return part[0];
}
template <usize N>
double Read(ReadOnlySpan<double> input) {
    ReadOnlySpan<double> part = Subspan(input,0,N as int);
    Assert.Concept<Readonly>(part,"readonly capability survives generic closure");
    return part[0];
}
`
	artifact := buildSemanticArtifact(t, "Geometry/Slices.concept", library, nil)
	const source = `module Geometry.SliceConsumer; profile Core;
import Geometry.Slices;
double Main() {
    double<array>[4] values = [1.0,2.0,3.0,4.0];
    double value = Slice<2>(Span(values),1);
    return value + values[1] + Read<1>(ReadOnlySpan(values));
}
`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("slices.concept", source, map[string][]byte{"Geometry.Slices": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "slices.generated.c")
		runFoundationNativeHarness(t, outputs, "slices_main.c", `#include "slices.generated.h"
int main(void) {return concept_geometry__slice_consumer_main()==15.0?0:1;}`)
	}
	const escape = `module Bad; profile Core;
template <usize N> Span<double> Escape() { double<array>[4] values = [0.0 ...]; return Subspan(Span(values),1,N as int); }
Span<double> Main() { return Escape<2>(); }`
	_, err := ParseWithSemanticModules("escape.concept", escape, map[string][]byte{"Geometry.Slices": artifact})
	if err == nil || !strings.Contains(err.Error(), "CV4521") {
		t.Fatalf("generic subspan must not extend local backing lifetime: %v", err)
	}
}
