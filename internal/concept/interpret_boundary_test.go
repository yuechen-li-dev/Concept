package concept

import (
	"bytes"
	"strings"
	"testing"
)

func TestInterpretBoundary(t *testing.T) {
	const source = `module Interpret.Boundary; profile Core;
float<mm> Attach(float raw) { return interpret raw as float<mm>; }
float Restore(float raw) { return Magnitude(interpret raw as float<mm>); }
float<m> Scale(float raw) { return (interpret raw as float<mm>) as float<m>; }
double<mm> ScaleWiden(float raw) { return (interpret raw as float<m>) as double<mm>; }
double<m> Widen(float raw) { return interpret (raw as double) as double<m>; }
double<Pa> Pressure(double raw) { return interpret raw as double<Pa>; }
int<byte> Bytes(int raw) { return interpret raw as int<byte>; }
float<MPa> Mega(float raw) { return interpret raw as float<MPa>; }
float<Pa> Pascals(float raw) { return (interpret raw as float<MPa>) as float<Pa>; }
`
	module, err := Parse("boundary.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := string(outputs["boundary.mir.json"])
	if !strings.Contains(mir, "interpret_scalar_semantics") || !strings.Contains(mir, "ExplicitInterpretation") || !strings.Contains(mir, "unit_scaled_float") {
		t.Fatal(mir)
	}
	c := string(outputs["boundary.generated.c"])
	if !strings.Contains(c, "return raw;") {
		t.Fatal("same-representation interpretation did not erase to the original scalar")
	}
	if strings.Contains(c, "malloc(") {
		t.Fatal("interpretation allocated")
	}
	assertR8cStrictC11(t, outputs, "boundary.generated.c")
	runFoundationNativeHarness(t, outputs, "boundary_harness.c", `#include "boundary.generated.h"
int main(void) {
 if (concept_interpret__boundary_attach(1200.0f) != 1200.0f) return 1;
 if (concept_interpret__boundary_restore(1200.0f) != 1200.0f) return 2;
 if (concept_interpret__boundary_scale(1200.0f) != 1.2f) return 3;
 double widened = concept_interpret__boundary_scale_widen(1.2f);
 if (widened < 1199.999 || widened > 1200.001) return 9;
 if (concept_interpret__boundary_widen(12.5f) != 12.5) return 4;
 if (concept_interpret__boundary_pressure(2.0) != 2.0) return 5;
 if (concept_interpret__boundary_bytes(12) != 12) return 6;
 if (concept_interpret__boundary_mega(2.0f) != 2.0f) return 7;
 if (concept_interpret__boundary_pascals(2.0f) != 2000000.0f) return 8;
 return 0;
}`)
}

func TestInterpretRejectsMisuse(t *testing.T) {
	for _, tc := range []struct{ source, code, hint string }{
		{`float<m> F(float x) { return x as float<m>; }`, "CAST_QUANTITY_SEMANTICS", "interpret"},
		{`float<m> F(float<mm> x) { return interpret x as float<m>; }`, "INTERPRET_USE_AS", "use value as T"},
		{`float<m> F(float<m> x) { return interpret x as float<m>; }`, "INTERPRET_INVALID", "use the value directly"},
		{`float F(float<m> x) { return interpret x as float; }`, "INTERPRET_USE_MAGNITUDE", "Magnitude"},
		{`float<m> F(double x) { return interpret x as float<m>; }`, "INTERPRET_REPRESENTATION", "convert first"},
		{`float F(int x) { return interpret x as float; }`, "INTERPRET_INVALID", "cannot reinterpret bits"},
		{`float<s> F(float<m> x) { return interpret x as float<s>; }`, "INTERPRET_INVALID", "dimension"},
		{`float<m> F(float x) { return interpret x + 1.0 as float<m>; }`, "INTERPRET_SYNTAX", "parenthesize"},
		{`ref int F(ref const int x) { return interpret x as ref int; }`, "INTERPRET_INVALID", "authority"},
		{`int F(Storage<int> x) { return interpret x as int; }`, "INTERPRET_INVALID", "storage"},
		{`struct SystemMemory {} struct DeviceMemory {} Address<DeviceMemory> F(Address<SystemMemory> x) { return interpret x as Address<DeviceMemory>; }`, "INTERPRET_INVALID", "address spaces"},
	} {
		_, err := Parse("invalid.concept", "profile Core; "+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) || !strings.Contains(err.Error(), tc.hint) {
			t.Errorf("%s: %v", tc.source, err)
		}
	}
}

func TestInterpretForeignAndProtocolBoundary(t *testing.T) {
	const source = `module Interpret.External; profile Core;
extern "C" float ReadDistanceMeters();
record struct Packet { float altitudeMeters; float pressurePascals; }
float<m> FromNative() { float raw = ReadDistanceMeters(); return interpret raw as float<m>; }
float<Pa> FromPacket(Packet packet) { return interpret packet.pressurePascals as float<Pa>; }
float<m> Attach(float raw) { return interpret raw as float<m>; }
void Proof() { Assert.Concept<NoAllocation>(Attach, "interpret itself does not allocate"); }
`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("external.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(outputs["external.mir.json"]), "ExplicitInterpretation") {
			t.Fatal("foreign/protocol interpretation absent from MIR")
		}
		assertR8cStrictC11(t, outputs, "external.generated.c")
		runFoundationNativeHarness(t, outputs, "external_harness.c", `#include "external.generated.h"
static int reads = 0;
float ReadDistanceMeters(void) { reads++; return 12.5f; }
int main(void) {
 if (concept_interpret__external_from_native() != 12.5f) return 1;
 if (reads != 1) return 3;
 concept_packet packet = {3.0f, 101325.0f};
 if (concept_interpret__external_from_packet(packet) != 101325.0f) return 2;
 return 0;
}`)
	}
	graph, err := ExplainSource("external.concept", source, 4)
	if err != nil {
		t.Fatal(err)
	}
	rendered := RenderProofVerbose(graph)
	for _, want := range []string{"ExplicitInterpretation", "float<m>", "dimension m", "scale 1/1", "source scalar representation", "not independently proved"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("explain lacks %q: %s", want, rendered)
		}
	}
	const story = `module Interpret.Explain; profile Core;
float<m> Distance(float raw) {
  float<mm> millimeters = interpret raw as float<mm>;
  return millimeters as float<m>;
}`
	interpretation, err := ExplainSource("story.concept", story, 3)
	if err != nil {
		t.Fatal(err)
	}
	conversion, err := ExplainSource("story.concept", story, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(RenderProofVerbose(interpretation), "ExplicitInterpretation") || !strings.Contains(RenderProofVerbose(conversion), "exact unit scale ratio 1/1000") {
		t.Fatal("explain conflated interpretation and conversion")
	}
}

func TestInterpretArtifactOnlyAndDeterminism(t *testing.T) {
	const producer = `module Interpret.Producer; profile Core;
float<mm> ReadMillimeters(float raw) { return interpret raw as float<mm>; }
float<m> ReadMeters(float raw) { return ReadMillimeters(raw) as float<m>; }
`
	const consumer = `module Interpret.Consumer; profile Core; import Interpret.Producer;
float<m> Consume(float raw) { return ReadMeters(raw); }
`
	var firstArtifact []byte
	var firstOutputs Outputs
	var firstExplain string
	for run := 0; run < 100; run++ {
		graph, err := ExplainSource("producer.concept", producer, 2)
		if err != nil {
			t.Fatal(err)
		}
		explain := RenderProofVerbose(graph)
		artifact, err := CompileSemanticModule("producer.concept", producer, nil)
		if err != nil {
			t.Fatal(err)
		}
		loaded, _, err := LoadSemanticModuleArtifact(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if len(loaded.Interpretations) != 1 || loaded.Interpretations[0].Origin != FactOriginExplicitInterpretation || loaded.Interpretations[0].Target != "float<mm>" {
			t.Fatalf("artifact lost interpretation: %+v", loaded.Interpretations)
		}
		module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"Interpret.Producer": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(consumer))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(outputs["consumer.generated.h"]), "float concept_interpret__consumer_consume(float raw)") || module.Functions[len(module.Functions)-1].ReturnType.String() != "float<m>" {
			t.Fatal("artifact consumer lost quantity type")
		}
		if run == 0 {
			assertR8cStrictC11(t, outputs, "consumer.generated.c")
			runFoundationNativeHarness(t, outputs, "consumer_harness.c", `#include "consumer.generated.h"
int main(void) { return concept_interpret__consumer_consume(1200.0f) == 1.2f ? 0 : 1; }`)
			firstArtifact, firstOutputs, firstExplain = artifact, outputs, explain
			continue
		}
		if explain != firstExplain {
			t.Fatalf("explain drift at run %d", run)
		}
		if !bytes.Equal(firstArtifact, artifact) {
			t.Fatalf("artifact drift at run %d", run)
		}
		for name, first := range firstOutputs {
			if !bytes.Equal(first, outputs[name]) {
				t.Fatalf("%s drift at run %d", name, run)
			}
		}
	}
}

func TestInterpretComptimeAndGeneric(t *testing.T) {
	const source = `module Interpret.Generic; profile Core;
comptime float<mm> Stored = interpret 1200.0 as float<mm>;
static_assert(Magnitude(Stored) == 1200.0, "comptime interpretation keeps current-unit value");
template <typename T> float<m> Attach(T raw) { return interpret raw as float<m>; }
float<m> Use(float raw) { return Attach<float>(raw); }
`
	module, err := Parse("generic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["generic.mir.json"]), "interpret_scalar_semantics") {
		t.Fatal("generic interpretation missing from MIR")
	}
	_, err = Parse("generic_invalid.concept", strings.Replace(source, "Attach<float>(raw)", "Attach<int>(1)", 1))
	if err == nil || !strings.Contains(err.Error(), "INTERPRET_REPRESENTATION") {
		t.Fatalf("generic must check concrete representation: %v", err)
	}
}

func TestInterpretSurvivesAsyncLowering(t *testing.T) {
	const source = `module Interpret.AsyncBoundary; profile Core;
async float ReadRaw() { return 12.5; }
async float<m> ReadMeters() { return interpret (await ReadRaw()) as float<m>; }
`
	module, err := Parse("async_boundary.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["async_boundary.mir.json"]), "interpret_scalar_semantics") {
		t.Fatal("async lowering lost interpretation")
	}
}
