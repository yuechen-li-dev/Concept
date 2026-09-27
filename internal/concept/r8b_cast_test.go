package concept

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestR8bCastAliasSemantics(t *testing.T) {
	sources := []string{
		`module R8b.Alias; profile Core; float F(int x) { return x as float; }`,
		`module R8b.Alias; profile Core; float F(int x) { return x as float32; }`,
		`module R8b.Alias; profile Core; float F(int x) { return static_cast<float>(x); }`,
	}
	var firstC []byte
	var firstOps []MIROperation
	for i, source := range sources {
		module, err := Parse("alias.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		var mir MIR
		if err := json.Unmarshal(outputs["alias.mir.json"], &mir); err != nil {
			t.Fatal(err)
		}
		ops := append([]MIROperation(nil), mir.Functions[0].Operations...)
		for j := range ops {
			ops[j].SourceSpan = Span{}
		}
		if i == 0 {
			firstC, firstOps = outputs["alias.generated.c"], ops
			continue
		}
		if !bytes.Equal(firstC, outputs["alias.generated.c"]) {
			t.Fatalf("cast spelling %d changed generated C", i)
		}
		if len(ops) != len(firstOps) {
			t.Fatalf("cast spelling %d changed MIR operation count", i)
		}
		for j := range ops {
			if ops[j].Kind != firstOps[j].Kind || ops[j].Type != firstOps[j].Type || ops[j].ReturnType != firstOps[j].ReturnType || ops[j].Detail != firstOps[j].Detail {
				t.Fatalf("cast spelling %d changed MIR operation %d", i, j)
			}
		}
	}
}

func TestR8bImplicitConversionBoundary(t *testing.T) {
	if _, err := Parse("literal.concept", `module R8b.Literal; profile Core; uint8 Five() { uint8 x = 5; return x; }`); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		`module R8b.Bad; profile Core; float Narrow(double x) { return x; }`,
		`module R8b.Bad; profile Core; uint8 Narrow(uint x) { return x; }`,
	} {
		if _, err := Parse("bad.concept", source); err == nil {
			t.Fatalf("implicit runtime narrowing accepted: %s", source)
		}
	}
}

func TestR8bCastPreservesCallEffects(t *testing.T) {
	const source = `module R8b.Effects; profile Core;
extern "C" float Foreign(float x);
double Converted(float x) { return Foreign(x) as double; }
void Proof() { Assert.Concept<NoAllocation>(Converted, "foreign allocation status is unknown"); }
`
	_, err := Parse("effects.concept", source)
	if err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_UNKNOWN") {
		t.Fatalf("cast hid its operand call effect: %v", err)
	}
}

func TestR8bCastAcrossAwait(t *testing.T) {
	const source = `module R8b.AsyncCast; profile Core;
async int Child() { return 42; }
async double Outer() { return (await Child()) as double; }
double Main() {
  Async<double> work = Outer();
  while (not Complete(work)) bounded(8) { Step(work); }
  return Result(work);
}
`
	module, err := Parse("async_cast.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outputs["async_cast.mir.json"], []byte("integer_to_float")) {
		t.Fatal("await cast missing from MIR")
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("GCC unavailable")
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	harness := `#include "async_cast.generated.h"
int main(void) { return concept_r8b__async_cast_main() == 42.0 ? 0 : 1; }
`
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("gcc", "-std=c11", "-pedantic-errors", filepath.Join(dir, "async_cast.generated.c"), filepath.Join(dir, "harness.c"), "-o", filepath.Join(dir, "async-cast.exe"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("async cast C11: %v\n%s", err, output)
	}
	cmd = exec.Command(filepath.Join(dir, "async-cast.exe"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("async cast runtime: %v\n%s", err, output)
	}
}

const r8bCastSource = `module R8b.Casts; profile Core;
float WidenInt(int x) { return x as float; }
double WidenFloat(float x) { return static_cast<double>(x); }
float NarrowFloat(double x) { return x as float32; }
double PreciseLiteral() { return 1.0000000000000002 as double; }
uint8 NarrowInt(int x) { return static_cast<uint8>(x); }
int SameInt(int x) { return x as int; }
uint SignedUnsigned(int x) { return x as uint; }
int UnsignedSigned(uint x) { return x as int; }
uint8 NarrowUnsigned(uint64 x) { return x as uint8; }
double<m> Distance(float<m> x) { return x as double<m>; }
void CastProof() {
  Assert.Concept<NoAllocation>(WidenInt, "explicit cast has no allocation");
  Assert.Concept<NoAllocation>(WidenFloat, "static_cast has no allocation");
  Assert.Concept<NoAllocation>(NarrowInt, "checked integer cast has no allocation");
}
`

const r8bRoundSource = `module R8b.Round; profile Core;
Result<int, NumericCastError> Truncate(float x) { return TruncTo<int>(x); }
Result<int, NumericCastError> Floor(float x) { return FloorTo<int>(x); }
Result<int, NumericCastError> Ceiling(float x) { return CeilTo<int>(x); }
Result<int, NumericCastError> Nearest(double x) { return RoundTo<int>(x); }
Result<uint8, NumericCastError> Unsigned(double x) { return RoundTo<uint8>(x); }
Result<uint8, NumericCastError> FloorUnsigned(double x) { return FloorTo<uint8>(x); }
Result<uint8, NumericCastError> CeilUnsigned(double x) { return CeilTo<uint8>(x); }
Result<uint8, NumericCastError> TruncUnsigned(double x) { return TruncTo<uint8>(x); }
Result<isize, NumericCastError> WideSigned(double x) { return TruncTo<isize>(x); }
Result<uint64, NumericCastError> WideUnsigned(double x) { return TruncTo<uint64>(x); }
Result<int, NumericCastError> Propagate(double x) {
  int y = FloorTo<int>(x)?;
  return Result::Ok(y);
}
void RoundProof() {
  Assert.Concept<NoAllocation>(Nearest, "rounding conversion has no allocation");
  Assert.Concept<NoAllocation>(FloorUnsigned, "rounding Result has no allocation");
}
`

const r8bProofSource = `module R8b.Proofs; profile Core;
concept ExactConversion<Source, Target> { requires compiler.ExactConversion<Source, Target>(); }
requires ExactConversion<int, double>;
requires ExactConversion<float, double>;
requires ExactConversion<float32, float64>;
void Proof() {
  Assert.Concept<ExactConversion>(int, double, "int32 fits binary64 exactly");
  Assert.Concept<ExactConversion>(float, double, "binary32 widens exactly");
  Assert.Concept<ExactConversion>(float32, float64, "alias identities canonicalize");
}
`

func TestR8bExactConversionFacts(t *testing.T) {
	if _, err := Parse("proofs.concept", r8bProofSource); err != nil {
		t.Fatal(err)
	}
	_, err := Parse("bad.concept", `module R8b.Bad; profile Core;
concept ExactConversion<Source, Target> { requires compiler.ExactConversion<Source, Target>(); }
requires ExactConversion<isize, float>;
`)
	if err == nil {
		t.Fatal("isize to float was incorrectly proven exact")
	}
}

func TestR8bArtifactOnlyConversions(t *testing.T) {
	casts, err := CompileSemanticModule("casts.concept", r8bCastSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	rounds, err := CompileSemanticModule("round.concept", r8bRoundSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer := `module R8b.Consumer; profile Core; import R8b.Casts; import R8b.Round;
double Cast(float x) { return WidenFloat(x); }
float FromInt(int x) { return WidenInt(x); }
double WrappedImported(int x) { return WidenInt(x) as double; }
Result<int, NumericCastError> Convert(double x) { return Nearest(x); }
void ConsumerProof() { Assert.Concept<ExactConversion>(int, double, "type-wide conversion fact survives artifact use"); }
`
	module, err := ParseWithSemanticModules("consumer.concept", consumer, map[string][]byte{"R8b.Casts": casts, "R8b.Round": rounds})
	if err != nil {
		t.Fatal(err)
	}
	consumerOutputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("GCC unavailable")
	}
	dir := t.TempDir()
	for _, item := range []struct{ path, source string }{{"casts.concept", r8bCastSource}, {"round.concept", r8bRoundSource}} {
		dependency, err := Parse(item.path, item.source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(dependency, []byte(item.source))
		if err != nil {
			t.Fatal(err)
		}
		for name, body := range outputs {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	for name, body := range consumerOutputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	harness := `#include "consumer.generated.h"
int main(void) {
  if (concept_r8b__consumer_cast(2.5f) != 2.5) return 1;
  if (concept_r8b__consumer_from_int(4) != 4.0f) return 2;
  if (concept_r8b__consumer_wrapped_imported(5) != 5.0) return 3;
  if (concept_r8b__consumer_convert(3.5).payload.ok.value != 4) return 4;
  return 0;
}
`
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("gcc", "-std=c11", "-pedantic-errors", filepath.Join(dir, "casts.generated.c"), filepath.Join(dir, "round.generated.c"), filepath.Join(dir, "consumer.generated.c"), filepath.Join(dir, "harness.c"), "-o", filepath.Join(dir, "artifact-test.exe"), "-lm")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("artifact-only C link: %v\n%s", err, output)
	}
	cmd = exec.Command(filepath.Join(dir, "artifact-test.exe"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("artifact-only runtime: %v\n%s", err, output)
	}
}

func TestR8bDeterminism100(t *testing.T) {
	const source = `module R8b.Deterministic; profile Core;
concept ExactConversion<Source, Target> { requires compiler.ExactConversion<Source, Target>(); }
requires ExactConversion<int, double>;
double Convert(int x) { return x as double; }
double Alias(float x) { return static_cast<double>(x); }
uint8 Narrow(int x) { return x as uint8; }
double<m> Distance(float<m> x) { return x as double<m>; }
Result<int, NumericCastError> Trunc(float x) { return TruncTo<int>(x); }
Result<int, NumericCastError> Floor(float x) { return FloorTo<int>(x); }
Result<int, NumericCastError> Ceil(float x) { return CeilTo<int>(x); }
Result<int, NumericCastError> Round(double x) { return RoundTo<int>(x); }
void Proof() { Assert.Concept<ExactConversion>(int, double, "int32 fits binary64"); }
`
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

func TestR8bComptimeCast(t *testing.T) {
	const source = `module R8b.Comptime; profile Core;
comptime double Five = 5 as double;
comptime double Six = static_cast<double>(6);
comptime double Precise = 1.0000000000000002 as double;
double Main() { return Five + Six + Precise; }
`
	module, err := Parse("comptime.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outputs["comptime.generated.c"], []byte("5.0")) || !bytes.Contains(outputs["comptime.generated.c"], []byte("6.0")) || !bytes.Contains(outputs["comptime.generated.c"], []byte("1.0000000000000002")) {
		t.Fatal("comptime double casts were not materialized")
	}
	_, err = Parse("bad.concept", `module R8b.Bad; profile Core; comptime int Bad = (-1 as uint8) as int;`)
	if err == nil || !strings.Contains(err.Error(), "INTEGER_CAST_OUT_OF_RANGE") {
		t.Fatalf("wrong comptime range diagnostic: %v", err)
	}
}

func TestR8bHalfExtensionLane(t *testing.T) {
	const source = `module R8b.Half; profile Core;
float Widen(half x) { return x as float; }
half Narrow(float x) { return static_cast<half>(x); }
`
	module, err := Parse("half.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("GCC extension lane unavailable")
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	harness := `#include "half.generated.h"
int main(void) {
  if (concept_r8b__half_widen((_Float16)1.5f) != 1.5f) return 1;
  if (concept_r8b__half_narrow(2.5f) != (_Float16)2.5f) return 2;
  return 0;
}
`
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("gcc", "-std=c11", filepath.Join(dir, "half.generated.c"), filepath.Join(dir, "harness.c"), "-o", filepath.Join(dir, "half-test.exe"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("GCC half extension: %v\n%s", err, output)
	}
	cmd = exec.Command(filepath.Join(dir, "half-test.exe"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("GCC half extension runtime: %v\n%s", err, output)
	}
}

func TestR8bVerifyConversionParity(t *testing.T) {
	module, err := Parse("round.concept", r8bRoundSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTargetWithPolicy(module, []byte(r8bRoundSource), GenericC11Target(), VerifyCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outputs["round.generated.c"], []byte("isfinite")) || !bytes.Contains(outputs["round.mir.json"], []byte("float_round_to_integer")) {
		t.Fatal("Verify discarded conversion semantics")
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("GCC unavailable")
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	harness := `#include "round.generated.h"
#include <math.h>
int main(void) {
  if (concept_r8b__round_nearest(3.5).payload.ok.value != 4) return 1;
  if (concept_r8b__round_nearest(NAN).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_NOT_FINITE) return 2;
  if (concept_r8b__round_unsigned(255.6).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_OUT_OF_RANGE) return 3;
  return 0;
}
`
	if err := os.WriteFile(filepath.Join(dir, "verify.c"), []byte(harness), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("gcc", "-std=c11", "-pedantic-errors", filepath.Join(dir, "round.generated.c"), filepath.Join(dir, "verify.c"), "-o", filepath.Join(dir, "verify.exe"), "-lm")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Verify strict C11: %v\n%s", err, output)
	}
	cmd = exec.Command(filepath.Join(dir, "verify.exe"))
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Verify runtime: %v\n%s", err, output)
	}
}

func TestR8bExplicitCastCore(t *testing.T) {
	module, err := Parse("casts.concept", r8bCastSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(r8bCastSource))
	if err != nil {
		t.Fatal(err)
	}
	mir := outputs["casts.mir.json"]
	for _, want := range []string{"integer_to_float", "float_representation", "integer_checked_range"} {
		if !bytes.Contains(mir, []byte(want)) {
			t.Fatalf("MIR missing %s", want)
		}
	}
	c := string(outputs["casts.generated.c"])
	if !strings.Contains(c, "integer cast out of range") || !strings.Contains(c, "cast_source") {
		t.Fatal("checked cast missing from generated C")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		if _, err := exec.LookPath(compiler); err != nil {
			continue
		}
		dir := t.TempDir()
		for name, body := range outputs {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
				t.Fatal(err)
			}
		}
		harness := `#include "casts.generated.h"
#include <math.h>
int main(void) {
  if (concept_r8b__casts_widen_int(3) != 3.0f) return 1;
  if (concept_r8b__casts_widen_float(2.5f) != 2.5) return 2;
  if (concept_r8b__casts_narrow_float(3.5) != 3.5f) return 3;
  if (concept_r8b__casts_narrow_int(255) != 255u) return 4;
  if (concept_r8b__casts_distance(4.5f) != 4.5) return 5;
  if (concept_r8b__casts_same_int(-4) != -4) return 6;
  if (concept_r8b__casts_signed_unsigned(255) != 255u) return 7;
  if (concept_r8b__casts_unsigned_signed(2147483647u) != 2147483647) return 8;
  if (concept_r8b__casts_narrow_unsigned(UINT64_C(255)) != 255u) return 9;
  if (concept_r8b__casts_narrow_float(1.000000059604644775390625) != 1.0f) return 10;
  if (!isinf(concept_r8b__casts_narrow_float(INFINITY))) return 11;
  if (!isnan(concept_r8b__casts_narrow_float(NAN))) return 12;
  if (concept_r8b__casts_precise_literal() <= 1.0) return 13;
  if (!isinf(concept_r8b__casts_narrow_float(1e300))) return 14;
  return 0;
}
`
		if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(compiler, "-std=c11", "-pedantic-errors", filepath.Join(dir, "casts.generated.c"), filepath.Join(dir, "harness.c"), "-o", filepath.Join(dir, "casts-test.exe"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11: %v\n%s", compiler, err, output)
		}
		cmd = exec.Command(filepath.Join(dir, "casts-test.exe"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s runtime: %v\n%s", compiler, err, output)
		}
		for i, expression := range []string{"concept_r8b__casts_narrow_int(256)", "concept_r8b__casts_signed_unsigned(-1)", "concept_r8b__casts_unsigned_signed(UINT32_C(2147483648))", "concept_r8b__casts_narrow_unsigned(UINT64_C(256))"} {
			trap := "#include \"casts.generated.h\"\nint main(void) { return " + expression + "; }\n"
			if err := os.WriteFile(filepath.Join(dir, "trap.c"), []byte(trap), 0600); err != nil {
				t.Fatal(err)
			}
			cmd = exec.Command(compiler, "-std=c11", "-pedantic-errors", filepath.Join(dir, "casts.generated.c"), filepath.Join(dir, "trap.c"), "-o", filepath.Join(dir, "trap.exe"))
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s trap %d link: %v\n%s", compiler, i, err, output)
			}
			cmd = exec.Command(filepath.Join(dir, "trap.exe"))
			if output, err := cmd.CombinedOutput(); err == nil || !bytes.Contains(output, []byte("integer cast out of range")) {
				t.Fatalf("%s trap %d did not reject: %v\n%s", compiler, i, err, output)
			}
		}
	}
	for _, snippet := range []string{
		`int Bad(float x) { return x as int; }`,
		`int Bad(float x) { return static_cast<int>(x); }`,
	} {
		_, err := Parse("bad.concept", "module R8b.Bad; profile Core; "+snippet)
		if err == nil || !strings.Contains(err.Error(), "FLOAT_TO_INT_ROUNDING_REQUIRED") {
			t.Fatalf("wrong float-to-int diagnostic: %v", err)
		}
	}
	for _, snippet := range []string{
		`float<m> Bad(float x) { return x as float<m>; }`,
		`float Bad(float<m> x) { return x as float; }`,
		`float<s> Bad(float<m> x) { return x as float<s>; }`,
		`uint<byte> Bad(uint<bit> x) { return x as uint<byte>; }`,
	} {
		_, err := Parse("bad.concept", "module R8b.Bad; profile Core; "+snippet)
		if err == nil || !strings.Contains(err.Error(), "CAST_QUANTITY_SEMANTICS") {
			t.Fatalf("wrong quantity diagnostic: %v", err)
		}
	}
	for _, sample := range []struct{ source, code string }{
		{`int Bad(float x) { return (int)x; }`, "C_STYLE_CAST_UNSUPPORTED"},
		{`int Bad(int x) { return reinterpret_cast<int>(x); }`, "REINTERPRET_CAST_UNSUPPORTED"},
		{`int Bad(int x) { return const_cast<int>(x); }`, "CONST_CAST_UNSUPPORTED"},
	} {
		_, err := Parse("bad.concept", "module R8b.Bad; profile Core; "+sample.source)
		if err == nil || !strings.Contains(err.Error(), sample.code) {
			t.Fatalf("wrong cast diagnostic: %v", err)
		}
	}
}

func TestR8bNamedRoundingBuilds(t *testing.T) {
	module, err := Parse("round.concept", r8bRoundSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(r8bRoundSource))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(outputs["round.mir.json"], []byte("float_round_to_integer")) {
		t.Fatal("rounding MIR operation missing")
	}
	c := string(outputs["round.generated.c"])
	for _, want := range []string{"isfinite", "fmod", "0x1p31", "0x1p8"} {
		if !strings.Contains(c, want) {
			t.Fatalf("generated C missing %s", want)
		}
	}
	if !strings.Contains(string(outputs["round.generated.h"]), "concept_numeric_cast_error") {
		t.Fatal("result error type missing from header")
	}
	for _, compiler := range []string{"gcc", "clang"} {
		if _, err := exec.LookPath(compiler); err != nil {
			continue
		}
		dir := t.TempDir()
		for name, body := range outputs {
			if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
				t.Fatal(err)
			}
		}
		cmd := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-fsyntax-only", filepath.Join(dir, "round.generated.c"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11: %v\n%s", compiler, err, output)
		}
		harness := `#include "round.generated.h"
#include <math.h>
int main(void) {
  if (concept_r8b__round_truncate(3.9f).payload.ok.value != 3) return 1;
  if (concept_r8b__round_truncate(-3.9f).payload.ok.value != -3) return 2;
  if (concept_r8b__round_floor(-3.9f).payload.ok.value != -4) return 3;
  if (concept_r8b__round_ceiling(-3.1f).payload.ok.value != -3) return 4;
  if (concept_r8b__round_nearest(2.5).payload.ok.value != 2) return 5;
  if (concept_r8b__round_nearest(3.5).payload.ok.value != 4) return 6;
  if (concept_r8b__round_nearest(-2.5).payload.ok.value != -2) return 7;
  if (concept_r8b__round_nearest(-3.5).payload.ok.value != -4) return 8;
  if (concept_r8b__round_nearest(-0.0).payload.ok.value != 0) return 9;
  if (concept_r8b__round_nearest(2147483647.0).payload.ok.value != 2147483647) return 10;
  if (concept_r8b__round_nearest(2147483648.0).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_OUT_OF_RANGE) return 11;
  if (concept_r8b__round_nearest(-2147483648.0).payload.ok.value != (-2147483647-1)) return 12;
  if (concept_r8b__round_nearest(-2147483649.0).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_OUT_OF_RANGE) return 13;
  if (concept_r8b__round_floor_unsigned(1.9).payload.ok.value != 1) return 14;
  if (concept_r8b__round_ceil_unsigned(255.0).payload.ok.value != 255) return 15;
  if (concept_r8b__round_trunc_unsigned(-0.5).payload.ok.value != 0) return 16;
  if (concept_r8b__round_unsigned(255.6).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_OUT_OF_RANGE) return 17;
  if (concept_r8b__round_nearest(NAN).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_NOT_FINITE) return 18;
  if (concept_r8b__round_nearest(INFINITY).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_NOT_FINITE) return 19;
  if (concept_r8b__round_nearest(-INFINITY).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_NOT_FINITE) return 20;
  if (concept_r8b__round_propagate(2.9).payload.ok.value != 2) return 21;
  if (concept_r8b__round_propagate(NAN).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_NOT_FINITE) return 22;
  if (concept_r8b__round_wide_signed(0x1p63).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_OUT_OF_RANGE) return 23;
  if (concept_r8b__round_wide_signed(-0x1p63).payload.ok.value != PTRDIFF_MIN) return 24;
  if (concept_r8b__round_wide_unsigned(0x1p64).payload.error.error.tag != CONCEPT_NUMERIC_CAST_ERROR_OUT_OF_RANGE) return 25;
  if (concept_r8b__round_wide_unsigned(0x1p64 - 0x1p11).payload.ok.value != UINT64_MAX - UINT64_C(2047)) return 26;
  return 0;
}
`
		if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0600); err != nil {
			t.Fatal(err)
		}
		args := []string{"-std=c11", "-pedantic-errors", filepath.Join(dir, "round.generated.c"), filepath.Join(dir, "harness.c"), "-o", filepath.Join(dir, "round-test.exe")}
		if compiler == "gcc" {
			args = append(args, "-lm")
		}
		cmd = exec.Command(compiler, args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s link: %v\n%s", compiler, err, output)
		}
		cmd = exec.Command(filepath.Join(dir, "round-test.exe"))
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s runtime: %v\n%s", compiler, err, output)
		}
	}
	_, err = Parse("bad.concept", "module R8b.Bad; profile Core; Result<float, NumericCastError> Bad(float x) { return FloorTo<float>(x); }")
	if err == nil || !strings.Contains(err.Error(), "NUMERIC_ROUND_TARGET") {
		t.Fatalf("wrong rounding target diagnostic: %v", err)
	}
}
