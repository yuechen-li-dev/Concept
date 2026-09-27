package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const r8aScalarSource = `module R8a.Scalars;
profile Core;
concept Floating<T> { requires compiler.Floating<T>(); }
concept BinaryFloat<T> { requires compiler.BinaryFloat<T>(); }
concept ThirtyTwo<T> { requires compiler.ScalarBits<T>(32); }
concept ExponentEight<T> { requires compiler.FloatExponentBits<T>(8); }
concept MantissaTwentyThree<T> { requires compiler.FloatMantissaBits<T>(23); }
concept NativeC<T> { requires compiler.CAbiLayout<T>(); }
requires Floating<half>;
requires Floating<float32>;
requires Floating<double>;
requires BinaryFloat<float64>;
requires ThirtyTwo<float>;
requires ExponentEight<float32>;
requires MantissaTwentyThree<float>;
requires NativeC<double>;
static_assert(SizeOf<half>() == 2, "binary16 storage");
static_assert(SizeOf<float16>() == 2, "binary16 alias storage");
static_assert(SizeOf<float>() == 4, "binary32 storage");
static_assert(SizeOf<float32>() == 4, "binary32 alias storage");
static_assert(SizeOf<double>() == 8, "binary64 storage");
static_assert(SizeOf<float64>() == 8, "binary64 alias storage");
static_assert(AlignOf<half>() == 2, "current target binary16 alignment");
static_assert(AlignOf<float>() == 4, "current target binary32 alignment");
static_assert(AlignOf<double>() == 8, "current target binary64 alignment");
struct Values { half a; float32 b; double c; float<K> kelvin; double<m> distance; half<K> temperature; }
reflect<Values>;
half AddHalf(half a, float16 b) { return a + b; }
float AddFloat(float a, float32 b) { return a + b; }
double AddDouble(double a, float64 b) { return a + b; }
float<K> AddKelvin(float<K> a, float32<K> b) { return a + b; }
half<K> AddTemperature(half<K> a, float16<K> b) { return a + b; }
void Proof() {
    Assert.Concept<Floating>(float32, "alias denotes a floating scalar");
    Assert.Concept<ScalarBits<64>>(float64, "binary64 width is semantic");
}
`

func TestR8aScalarCanonicalizationAndFacts(t *testing.T) {
	module, err := Parse("Scalars.concept", r8aScalarSource)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"half", "float16"}, {"float", "float32"}, {"double", "float64"}} {
		a, _ := evt1BuiltinType(pair[0], Span{})
		b, _ := evt1BuiltinType(pair[1], Span{})
		if !a.Equal(b) || evt1TypeIdentity(a) != evt1TypeIdentity(b) || evt1CType(a) != evt1CType(b) {
			t.Fatalf("%s and %s have different identities", pair[0], pair[1])
		}
	}
	if len(module.ReflectionResults) != 1 || len(module.ReflectionResults[0].Fields) != 6 {
		t.Fatalf("missing scalar reflection: %+v", module.ReflectionResults)
	}
	fields := module.ReflectionResults[0].Fields
	for i, want := range []struct {
		name string
		rep  FloatRepresentation
	}{{"half", FloatBinary16}, {"float", FloatBinary32}, {"double", FloatBinary64}, {"float", FloatBinary32}, {"double", FloatBinary64}, {"half", FloatBinary16}} {
		if fields[i].Type.Name != want.name || fields[i].Type.FloatRepresentation != want.rep {
			t.Fatalf("field %d: got %s %s, want %s %s", i, fields[i].Type.Name, fields[i].Type.FloatRepresentation, want.name, want.rep)
		}
	}
	if fields[3].Type.Quantity == nil || fields[4].Type.Quantity == nil || fields[5].Type.Quantity == nil {
		t.Fatal("unit qualification was lost")
	}
	outputs, err := Generate(module, []byte(r8aScalarSource))
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["scalars.generated.c"])
	for _, want := range []string{"_Float16", "float", "double"} {
		if !strings.Contains(c, want) {
			t.Fatalf("generated C lacks %s", want)
		}
	}
	verifyOutputs, err := GenerateForTargetWithPolicy(module, []byte(r8aScalarSource), GenericC11Target(), VerifyCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	verifyC := string(verifyOutputs["scalars.generated.c"])
	for _, want := range []string{"_Float16", "float", "double"} {
		if !strings.Contains(verifyC, want) {
			t.Fatalf("Verify C lacks %s", want)
		}
	}
}

func TestR8aArtifactOnlyScalarRepresentation(t *testing.T) {
	artifact, err := CompileSemanticModule("Scalars.concept", r8aScalarSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, decoded, err := LoadSemanticModuleArtifact(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Structs) != 1 || len(decoded.Structs[0].Fields) < 3 {
		t.Fatal("artifact lost scalar declarations")
	}
	for i, rep := range []FloatRepresentation{FloatBinary16, FloatBinary32, FloatBinary64} {
		if decoded.Structs[0].Fields[i].Type.FloatRepresentation != rep {
			t.Fatalf("artifact lost %s representation", rep)
		}
	}
	consumer := `module Consumer; profile Core; import R8a.Scalars;
float32<K> Warm(float<K> a, float32<K> b) { return AddKelvin(a, b); }
double Sum(double a, float64 b) { return AddDouble(a, b); }
void ConsumerProof() { Assert.Concept<ScalarBits<64>>(double, "imported scalar fact"); }
`
	if _, err := ParseWithSemanticModules("Consumer.concept", consumer, map[string][]byte{"R8a.Scalars": artifact}); err != nil {
		t.Fatal(err)
	}
}

func TestR8aPrecisionArgumentDiagnostic(t *testing.T) {
	for _, spelling := range []string{"float<32>", "float<64>"} {
		_, err := Parse("width.concept", "profile Core; "+spelling+" Value();")
		if err == nil || !strings.Contains(err.Error(), "unit/dimension expression, not a precision parameter") || !strings.Contains(err.Error(), "double / float64") {
			t.Fatalf("%s: %v", spelling, err)
		}
	}
}

func TestR8aExplainRepresentation(t *testing.T) {
	const source = `profile Core;
using Kelvin = float<K>;
void Proof() { Assert.Concept<ScalarBits<32>>(Kelvin, "binary32 kelvin"); }`
	graph, err := ExplainSource("explain.concept", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	if rendered := RenderProofVerbose(graph); !strings.Contains(rendered, "floating representation binary32") || !strings.Contains(rendered, "Kelvin : float<K>") {
		t.Fatal(rendered)
	}
}

func TestR8aUnsupportedFormats(t *testing.T) {
	for _, name := range []string{"bfloat16", "float8e4m3", "float8e5m2", "float8", "float4"} {
		_, err := Parse("unsupported.concept", "profile Core; "+name+" Value();")
		if err == nil || !strings.Contains(err.Error(), "FLOAT_REPRESENTATION_UNSUPPORTED") {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestR8aInvalidFactsAndHalfABI(t *testing.T) {
	for _, source := range []string{
		`profile Core; concept Wrong<T> { requires compiler.ScalarBits<T>(64); } requires Wrong<float>;`,
		`profile Core; concept NativeC<T> { requires compiler.CAbiLayout<T>(); } requires NativeC<half>;`,
	} {
		_, err := Parse("invalid_fact.concept", source)
		if err == nil || !strings.Contains(err.Error(), "CV4640") {
			t.Fatalf("invalid representation/ABI fact: %v", err)
		}
	}
}

func TestR8aHalfCompilerExtension(t *testing.T) {
	const source = `module R8a.Half; profile Core; half Add(half a, float16 b) { return a + b; }`
	module, err := Parse("Half.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, compiler := range []string{"clang", "gcc"} {
		if _, err := exec.LookPath(compiler); err != nil {
			t.Skipf("%s unavailable", compiler)
		}
		dir := t.TempDir()
		for name, body := range outputs {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(compiler, "-std=c11", "-fsyntax-only", filepath.Join(dir, "half.generated.c"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s binary16 extension: %v\n%s", compiler, err, output)
		}
		// The installed Clang Windows/MSVC target lacks the compiler-rt half
		// conversion helpers at link time; its syntax acceptance is not a
		// runnable backend claim. GCC supplies those helpers on this host.
		if compiler != "gcc" {
			continue
		}
		harness := `#include "half.generated.h"
int main(void) {
    _Float16 result = concept_r8a__half_add((_Float16)1.5f, (_Float16)2.0f);
    return result == (_Float16)3.5f ? 0 : 1;
}
`
		if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
			t.Fatal(err)
		}
		executable := filepath.Join(dir, "half-test.exe")
		cmd = exec.Command(compiler, "-std=c11", filepath.Join(dir, "half.generated.c"), filepath.Join(dir, "harness.c"), "-o", executable)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s binary16 executable: %v\n%s", compiler, err, output)
		}
		cmd = exec.Command(executable)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s binary16 arithmetic: %v\n%s", compiler, err, output)
		}
	}
}

func TestR8aGenericAliasesAndContainers(t *testing.T) {
	const source = `module R8a.Generic; profile Core;
template <typename T> T Identity(T value) { return value; }
half A(half x) { return Identity<half>(x); }
half B(float16 x) { return Identity<float16>(x); }
float C(float x) { return Identity<float>(x); }
float D(float32 x) { return Identity<float32>(x); }
double E(double x) { return Identity<double>(x); }
double F(float64 x) { return Identity<float64>(x); }
table<2> Scalars { half h; double d; }
int ArrayAndSpan(half a, double b) {
    half[2] hs = [a, a]; double[2] ds = [b, b];
    Span<half> h = Span(hs); ReadOnlySpan<double> d = ReadOnlySpan(ds);
    return Len(h) + Len(d);
}
double TensorDouble(double a, double b) { tensor<double> t[2] = [a, b]; return t[1]; }
float<m> TensorQuantity(float<m> a) { tensor<float<m>> t[1] = [a]; return t[0]; }
`
	module, err := Parse("Generic.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["generic.generated.c"])
	for _, name := range []string{"half", "float", "double"} {
		if got := strings.Count(c, "static "+evt1CType(Type{Name: name, Kind: TypeBuiltin})+" concept_template_identity__"+name+"("); got != 2 {
			t.Fatalf("%s generated %d prototype/definitions, want one instance", name, got)
		}
	}
}

func TestR8aDeterminism100(t *testing.T) {
	if testing.Short() {
		t.Skip("100-run artifact and backend identity is a full gate")
	}
	var baselineArtifact []byte
	var baselineOutputs Outputs
	var baselineProof string
	for i := 0; i < 100; i++ {
		artifact, err := CompileSemanticModule("Scalars.concept", r8aScalarSource, nil)
		if err != nil {
			t.Fatal(err)
		}
		module, err := Parse("Scalars.concept", r8aScalarSource)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(r8aScalarSource))
		if err != nil {
			t.Fatal(err)
		}
		proof, err := ExplainSource("Scalars.concept", r8aScalarSource, 0)
		if err != nil {
			t.Fatal(err)
		}
		proofText := RenderProofVerbose(proof)
		if i == 0 {
			baselineArtifact, baselineOutputs, baselineProof = artifact, outputs, proofText
			continue
		}
		if !bytes.Equal(artifact, baselineArtifact) || len(outputs) != len(baselineOutputs) || proofText != baselineProof {
			t.Fatalf("artifact or output set changed on run %d", i+1)
		}
		for name, body := range baselineOutputs {
			if !bytes.Equal(body, outputs[name]) {
				t.Fatalf("%s changed on run %d", name, i+1)
			}
		}
	}
}

func TestR8aDoubleStrictC11(t *testing.T) {
	const source = `module R8a.Strict; profile Core;
double Add(double a, float64 b) { return a + b; }
double Scale(double a) { return a * 1.0000000000000002; }
double Main() { double a = 1.0000000000000002; double b = 2.0; return Add(a, b); }
`
	module, err := Parse("Strict.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["strict.generated.c"])
	if !strings.Contains(c, "1.0000000000000002") || strings.Contains(c, "1.0000000000000002f") {
		t.Fatal("binary64 literal was rounded through binary32")
	}
	for _, compiler := range []string{"clang", "gcc"} {
		if _, err := exec.LookPath(compiler); err != nil {
			t.Skipf("%s unavailable", compiler)
		}
		dir := t.TempDir()
		for name, body := range outputs {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-fsyntax-only", filepath.Join(dir, "strict.generated.c"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11: %v\n%s", compiler, err, output)
		}
		harness := `#include "strict.generated.h"
int main(void) { return concept_r8a__strict_scale(1.0) > 1.0 ? 0 : 1; }
`
		if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
			t.Fatal(err)
		}
		executable := filepath.Join(dir, "strict-test.exe")
		cmd = exec.Command(compiler, "-std=c11", "-pedantic-errors", filepath.Join(dir, "strict.generated.c"), filepath.Join(dir, "harness.c"), "-o", executable)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11 executable: %v\n%s", compiler, err, output)
		}
		cmd = exec.Command(executable)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s binary64 arithmetic: %v\n%s", compiler, err, output)
		}
	}
}
