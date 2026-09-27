package concept

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR8cStandardUnitCatalog(t *testing.T) {
	const names = "nm um mm cm m km mg g kg ns us ms s A mA K mol cd bit byte Hz kHz MHz GHz N mN kN Pa kPa MPa GPa W mW kW MW V mV kV"
	if len(standardQuantityUnits) != len(strings.Fields(names)) {
		t.Fatalf("catalog size changed: got %d", len(standardQuantityUnits))
	}
	for i, name := range strings.Fields(names) {
		if standardQuantityUnits[i].Name != name {
			t.Fatalf("catalog spelling at %d: %s", i, standardQuantityUnits[i].Name)
		}
		if _, ok := quantityFromUnit(name); !ok {
			t.Fatalf("missing %s", name)
		}
	}
	for _, name := range []string{"dm", "dam", "hm", "daN"} {
		if _, ok := quantityFromUnit(name); ok {
			t.Fatalf("unsupported spelling accepted: %s", name)
		}
	}
}

func TestR8cDimensionAndScaleLaws(t *testing.T) {
	unit := func(name string) QuantityDimension {
		d, ok := quantityFromUnit(name)
		if !ok {
			t.Fatal(name)
		}
		return d
	}
	if !unit("kg").Multiply(unit("m")).Divide(unit("s").Pow(2)).Equal(unit("N")) {
		t.Fatal("newton dimension")
	}
	if !unit("N").Divide(unit("m").Pow(2)).Equal(unit("Pa")) {
		t.Fatal("pascal dimension")
	}
	if !unit("s").Pow(-1).Equal(unit("Hz")) {
		t.Fatal("hertz dimension")
	}
	if !unit("m").Divide(unit("m")).IsDimensionless() {
		t.Fatal("cancellation")
	}
	if !unit("mm").SameDimension(unit("m")) || unit("mm").Equal(unit("m")) {
		t.Fatal("dimension/scale conflated")
	}
	n, d := unit("mm").ScaleRatioTo(unit("m"))
	if n != 1 || d != 1000 {
		t.Fatalf("mm ratio %d/%d", n, d)
	}
	n, d = unit("MPa").ScaleRatioTo(unit("Pa"))
	if n != 1000000 || d != 1 {
		t.Fatalf("MPa ratio %d/%d", n, d)
	}
}

func TestR8cScientificTokenization(t *testing.T) {
	tokens, err := lexEVT1("1e-3m 1e3m 1e-3 * m")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1e-3", "m", "1e3", "m", "1e-3", "*", "m"}
	if len(tokens) != len(want) {
		t.Fatalf("tokens: %+v", tokens)
	}
	for i, token := range tokens {
		if token.Lexeme != want[i] {
			t.Fatalf("token %d: %q", i, token.Lexeme)
		}
	}
}

func TestR8cExplainScale(t *testing.T) {
	const source = `profile Core;
void Proof(float<mm> millimeters, float<m> meters) {
  Assert.Concept<UnitScale<1, 1000>>(millimeters, meters, "exact scale proof");
}`
	graph, err := ExplainSource("scale.concept", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	rendered := RenderProofVerbose(graph)
	if !strings.Contains(rendered, "same normalized dimension") || !strings.Contains(rendered, "exact scale ratio 1/1000") {
		t.Fatal(rendered)
	}
}

func TestR8cLiteralsCastsAndMagnitude(t *testing.T) {
	const source = `module R8c.Units; profile Core;
float<m> Meter() { return 1.0m; }
float<m> Tenth() { return 1e-1m; }
float<m> Thousandth() { return 1e-3m; }
float<m> Thousand() { return 1e3m; }
int<m> One() { return 1m; }
float<Pa> Pressure() { return 1MPa as float<Pa>; }
float<Hz> Frequency() { return 1GHz as float<Hz>; }
float<m> ConvertLength(float<mm> x) { return x as float<m>; }
double<m> ConvertAndWiden(float<mm> x) { return x as double<m>; }
float<Pa> ConvertPressure(float<MPa> x) { return x as float<Pa>; }
float<N> Force(float<kg> mass, float<m/s^2> acceleration) { return mass * acceleration; }
float<Pa> Stress(float<N> force, float<m^2> area) { return force / area; }
float Ratio(float<mm> small, float<m> large) { return small / large; }
float Raw(float<mm> x) { return Magnitude(x); }
table<2> EngineeringScalars { float<mm> distance; double<Pa> pressure; }
int ArrayAndSpan(float<mm> x) {
  float<mm>[2] values = [x, x];
  Span<float<mm>> view = Span(values);
  return Len(view);
}
`
	module, err := Parse("units.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := string(outputs["units.mir.json"])
	if !strings.Contains(mir, "unit_scaled_float") || !strings.Contains(mir, "exact unit scale ratio 1/1000") {
		t.Fatal("scaled cast absent from MIR")
	}
	c := string(outputs["units.generated.c"])
	if !strings.Contains(c, "1000000.0") || !strings.Contains(c, "1000.0") {
		t.Fatal("scale arithmetic absent from C")
	}
	assertR8cStrictC11(t, outputs, "units.generated.c")
	runFoundationNativeHarness(t, outputs, "r8c_units_harness.c", `#include "units.generated.h"
int main(void) {
 if (concept_r8c__units_meter() != 1.0f) return 1;
 if (concept_r8c__units_tenth() != 0.1f) return 2;
 if (concept_r8c__units_thousandth() != 0.001f) return 3;
 if (concept_r8c__units_thousand() != 1000.0f) return 4;
 if (concept_r8c__units_one() != 1) return 5;
 if (concept_r8c__units_pressure() != 1000000.0f) return 6;
 if (concept_r8c__units_frequency() != 1000000000.0f) return 7;
 if (concept_r8c__units_convert_length(2500.0f) != 2.5f) return 8;
 if (concept_r8c__units_convert_and_widen(2500.0f) != 2.5) return 9;
 if (concept_r8c__units_convert_pressure(2.0f) != 2000000.0f) return 10;
 if (concept_r8c__units_force(2.0f, 3.0f) != 6.0f) return 11;
 if (concept_r8c__units_stress(6.0f, 2.0f) != 3.0f) return 12;
 if (concept_r8c__units_ratio(1.0f, 1.0f) != 0.001f) return 13;
 if (concept_r8c__units_raw(5.0f) != 5.0f) return 14;
 if (concept_r8c__units_array_and_span(5.0f) != 2) return 15;
 return 0;
}`)
}

func assertR8cStrictC11(t *testing.T, outputs Outputs, source string) {
	t.Helper()
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"gcc", "clang"} {
		if _, err := exec.LookPath(compiler); err != nil {
			continue
		}
		cmd := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-fsyntax-only", filepath.Join(dir, source))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11: %v\n%s", compiler, err, output)
		}
	}
}

func TestR8cRejections(t *testing.T) {
	for _, sample := range []struct{ source, code string }{
		{`float<m> F() { return 1dm; }`, "QUANTITY_UNIT_UNKNOWN"},
		{`float<dm> F() { return 1.0m; }`, "QUANTITY_UNIT_UNKNOWN"},
		{`float<s> F(float<m> x) { return x as float<s>; }`, "CAST_QUANTITY_SEMANTICS"},
		{`float<m> F(float x) { return x as float<m>; }`, "CAST_QUANTITY_SEMANTICS"},
		{`float F(float<m> x) { return x as float; }`, "CAST_QUANTITY_SEMANTICS"},
		{`int<m> F(int<mm> x) { return x as int<m>; }`, "CAST_QUANTITY_SEMANTICS"},
		{`int F(int<mm> x, int<m> y) { return x / y; }`, "QUANTITY_SCALE_INTEGRAL"},
		{`float<nm^3> F(float x) { return AssumeQuantity<float<nm^3>>(x); }`, "QUANTITY_SCALE_OVERFLOW"},
		{`float<mm/m> F(float x) { return x; }`, "QUANTITY_DIMENSIONLESS_SCALE"},
	} {
		_, err := Parse("bad.concept", "profile Core; "+sample.source)
		if err == nil || !strings.Contains(err.Error(), sample.code) {
			t.Fatalf("%s: %v", sample.source, err)
		}
	}
}

func TestR8cComptimeAndArtifactTransport(t *testing.T) {
	const proof = `module R8c.Proof; profile Core;
static_assert(Magnitude(1m) == 1, "quantity literal evaluates at comptime");
static_assert(Magnitude(1e-1m) == 0.1, "scientific unit literal evaluates at comptime");
float<m> FromMillimeters(float<mm> x) { return x as float<m>; }
float<Pa> FromMegapascals(float<MPa> x) { return x as float<Pa>; }
float<N> Newton(float<kg> mass, float<m/s^2> acceleration) { return mass * acceleration; }
float<Pa> TensorPressure() {
  tensor<float> basis[1, 1] = [[2.0]];
  tensor<float<Pa>> pressure[1] = [1.0Pa];
  tensor<float<Pa>> output[1] = 0.0Pa;
  output[i] = basis[i, j] * pressure[j];
  return output[0];
}
void Facts(float<mm> millimeters, float<m> meters, float<MPa> mega, float<Pa> pascals, float<kg*m/s^2> derivedForce, float<N> force, float<Hz> hertz, float<s^-1> inverseSeconds, float scalar) {
  Assert.Concept<SameDimension>(millimeters, meters, "mm and m share length dimension");
  Assert.Concept<UnitScale<1, 1000>>(millimeters, meters, "mm to m is exact one thousandth");
  Assert.Concept<UnitScale<1000000, 1>>(mega, pascals, "MPa to Pa exact scale");
  Assert.Concept<SameDimension>(derivedForce, force, "newton matches base dimensions");
  Assert.Concept<SameDimension>(hertz, inverseSeconds, "hertz is inverse seconds");
  Assert.Concept<Dimensionless>(scalar, "plain scalar has no dimension");
  Assert.Concept<NoAllocation>(FromMillimeters, "unit cast has no allocation");
}`
	artifact, err := CompileSemanticModule("proof.concept", proof, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module R8c.Consumer; profile Core; import R8c.Proof;
float<m> Length(float<mm> x) { return FromMillimeters(x); }
float<Pa> Pressure(float<MPa> x) { return FromMegapascals(x); }
float<N> Force(float<kg> x, float<m/s^2> y) { return Newton(x, y); }
float<Pa> ProjectPressure() { return TensorPressure(); }
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"R8c.Proof": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTargetWithPolicy(module, []byte(consumer), GenericC11Target(), VerifyCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["consumer.mir.json"]), "m/s^2") {
		t.Fatal("artifact consumer lost derived dimension")
	}
}

func TestR8cTensorComposition(t *testing.T) {
	const source = `module R8c.Tensor; profile Core;
float<Pa> Transform() {
  tensor<float> basis[1, 1] = [[2.0]];
  tensor<float<Pa>> stress[1] = [1.0Pa];
  tensor<float<Pa>> output[1] = 0.0Pa;
  output[i] = basis[i, j] * stress[j];
  return output[0];
}`
	module, err := Parse("tensor.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["tensor.mir.json"]), "tensor") {
		t.Fatal("tensor MIR absent")
	}
	runFoundationNativeHarness(t, outputs, "r8c_tensor_harness.c", `#include "tensor.generated.h"
int main(void) { return concept_r8c__tensor_transform() == 2.0f ? 0 : 1; }`)
}

func TestR8cDeterminism100(t *testing.T) {
	const source = `module R8c.Deterministic; profile Core;
static_assert(Magnitude(1e-1m) == 0.1, "scientific literal");
float<m> Scaled(float<mm> x) { return x as float<m>; }
float<Pa> Pressure(float<MPa> x) { return x as float<Pa>; }
float<N> Force(float<kg> mass, float<m/s^2> acceleration) { return mass * acceleration; }
float<Pa> Tensor() {
  tensor<float> basis[1, 1] = [[2.0]];
  tensor<float<Pa>> stress[1] = [1.0Pa];
  tensor<float<Pa>> out[1] = 0.0Pa;
  out[i] = basis[i, j] * stress[j];
  return out[0];
}
void Proof(float<mm> mm, float<m> m) {
  Assert.Concept<UnitScale<1, 1000>>(mm, m, "exact rational scale");
  Assert.Concept<NoAllocation>(Scaled, "scaled cast has no allocation");
}`
	var first Outputs
	var firstArtifact []byte
	for run := 0; run < 100; run++ {
		module, err := Parse("deterministic.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := CompileSemanticModule("deterministic.concept", source, nil)
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			first, firstArtifact = outputs, artifact
			continue
		}
		if !bytes.Equal(artifact, firstArtifact) {
			t.Fatalf("artifact differs at run %d", run)
		}
		if len(outputs) != len(first) {
			t.Fatalf("output count differs at run %d", run)
		}
		for name, body := range first {
			if !bytes.Equal(outputs[name], body) {
				t.Fatalf("%s differs at run %d", name, run)
			}
		}
	}
}

func TestR8cNormalVerifyParity(t *testing.T) {
	const source = `module R8c.Parity; profile Core;
float<m> ConvertLength(float<mm> x) { return x as float<m>; }
float<Pa> ConvertPressure(float<MPa> x) { return x as float<Pa>; }
float Ratio(float<mm> x, float<m> y) { return x / y; }
float Raw(float<mm> x) { return Magnitude(x); }`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("parity.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "parity.generated.c")
		runFoundationNativeHarness(t, outputs, "r8c_parity_harness.c", `#include "parity.generated.h"
int main(void) {
 if (concept_r8c__parity_convert_length(2500.0f) != 2.5f) return 1;
 if (concept_r8c__parity_convert_pressure(2.0f) != 2000000.0f) return 2;
 if (concept_r8c__parity_ratio(1.0f, 1.0f) != 0.001f) return 3;
 if (concept_r8c__parity_raw(5.0f) != 5.0f) return 4;
 return 0;
}`)
	}
}

func TestR8cHalfUnitExtension(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("GCC half extension unavailable")
	}
	const source = `module R8c.Half; profile Core;
float<m> Widen(half<mm> x) { return x as float<m>; }`
	module, err := Parse("half.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "r8c_half_harness.c", `#include "half.generated.h"
int main(void) { return concept_r8c__half_widen((_Float16)2000.0f) == 2.0f ? 0 : 1; }`)
}
