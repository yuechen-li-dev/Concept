package concept

import (
	"bytes"
	"strings"
	"testing"
)

func TestQuantityCannotImplicitlyChangeMeaning(t *testing.T) {
	for _, tc := range []struct {
		name, source, diagnostic string
	}{
		{"assignment erasure", `float F(float<m> distance) { float plain = distance; return plain; }`, "CV4106"},
		{"argument erasure", `float Plain(float x) { return x; } float F(float<m> distance) { return Plain(distance); }`, "CV4107"},
		{"return erasure", `float F(float<m> distance) { return distance; }`, "CV4116"},
		{"cast erasure", `float F(float<m> distance) { return distance as float; }`, "CAST_QUANTITY_SEMANTICS"},
		{"scalar attachment", `float<m> F(float value) { return value; }`, "CV4116"},
		{"dimension mixing", `float<s> F(float<m> distance) { return distance; }`, "CV4116"},
		{"generic erasure", `template <typename T> T Identity(T x) { return x; } float F(float<m> distance) { return Identity<float>(distance); }`, "CV4107"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("r8g_quantity.concept", "profile Core; "+tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
				t.Fatalf("expected %s, got %v", tc.diagnostic, err)
			}
		})
	}
	const valid = `module Contextual.Quantity; profile Core;
float Plain(float x) { return x; }
float Explicit(float<m> distance) { return Plain(Magnitude(distance)); }
float<m> Attach(float scalar) { return interpret scalar as float<m>; }
float<mm> Convert(float<m> distance) { return distance as float<mm>; }
`
	module, err := Parse("r8g_quantity.concept", valid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(valid)); err != nil {
		t.Fatal(err)
	}
	mathSource := r7pGoldenSource(t, "Standard", "Math.concept")
	mathArtifact, err := CompileSemanticModule("Standard/Math.concept", mathSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, call := range []string{"Abs(distance)", "Min(distance, distance)", "Max(distance, distance)"} {
		consumer := "module Contextual.MathSafety; profile Core; import Standard.Math; float F(float<m> distance) { return " + call + "; }"
		_, err := ParseWithSemanticModules("r8g_math_safety.concept", consumer, map[string][]byte{"Standard.Math": mathArtifact})
		if err == nil || !strings.Contains(err.Error(), "CV4116") {
			t.Fatalf("%s silently erased quantity through Standard.Math: %v", call, err)
		}
	}
}

func TestContextualArrayLiteralLowering(t *testing.T) {
	const source = `module Contextual.Arrays; profile Core;
struct Inner { uint8[4] bytes; int count; }
struct Outer { Inner inner; }
template <typename T> struct Box { T[4] values; }
enum Packet { Bytes(uint8[4] bytes), Empty }
Inner Make() { return Inner{[0, 1, 2, 3], 4}; }
Outer Nested() { return Outer{Inner{[7 ... 4], 9}}; }
Box<uint8> Generic() { return Box<uint8>{[1, 2, 3, 4]}; }
int GenericFirst() { Box<uint8> box = Generic(); return box.values[0] as int; }
int First(uint8[4] bytes) { return bytes[0] as int; }
int DirectArrayArgument() { return First([9, 8, 7, 6]); }
Packet MakePacket() { return Packet::Bytes([5, 4, 3, 2]); }
int PacketFirst() { Packet packet = MakePacket(); return match (packet) { Packet::Bytes(bytes) => bytes[0] as int, Packet::Empty => 0 }; }
int Sum(Inner value) { return (value.bytes[0] as int) + (value.bytes[1] as int) + (value.bytes[2] as int) + (value.bytes[3] as int) + value.count; }
int Pass() { return Sum(Inner{[4, 5, 6, 7], 8}); }
`
	module, err := Parse("r8g_arrays.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["r8g_arrays.generated.c"])
	if strings.Contains(c, "concept_array_4_int") {
		t.Fatalf("array literal lost uint8 element context:\n%s", c)
	}
	assertR8cStrictC11(t, outputs, "r8g_arrays.generated.c")
	runFoundationNativeHarness(t, outputs, "arrays_harness.c", `#include "r8g_arrays.generated.h"
int main(void) {
  if (concept_contextual__arrays_sum(concept_contextual__arrays_make()) != 10) return 1;
  if (concept_contextual__arrays_pass() != 30) return 2;
  if (concept_contextual__arrays_generic_first() != 1) return 3;
  if (concept_contextual__arrays_direct_array_argument() != 9) return 4;
  if (concept_contextual__arrays_packet_first() != 5) return 5;
  return 0;
}`)
}

func TestContextualFloatLiterals(t *testing.T) {
	const source = `module Contextual.Floats; profile Core;
struct Sample { double value; double[2] pair; }
enum Choice { Left, Right }
enum ChoiceError { Rejected }
double Identity(double x) { return x; }
double UnitCircle(double c, double s) { return c*c + s*s - 1.0; }
double Negative() { double value = -1.0; return value; }
bool IsTwenty(double x) { return x == 20.0; }
Sample MakeSample() { return Sample{1.25, [2.5, -3.75]}; }
double Call() { return Identity(1.0); }
double TensorLiteral() { tensor<double> values[2] = [1.0, 2.0]; return values[1]; }
double MatchFloat(Choice choice) { return match (choice) { Choice::Left => 1.0, Choice::Right => -2.0 }; }
double MatchFloatValue() { return MatchFloat(Choice::Left); }
Result<double, ChoiceError> MatchResult(Choice choice) { return match (choice) { Choice::Left => Result::Ok(1.0), Choice::Right => Result::Error(ChoiceError::Rejected) }; }
double MatchResultValue() { return MatchResult(Choice::Left)!; }
`
	module, err := Parse("r8g_floats.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["r8g_floats.generated.c"])
	if strings.Contains(c, "1.25f") || strings.Contains(c, "2.5f") || strings.Contains(c, "3.75f") {
		t.Fatal("double literal was lowered through binary32")
	}
	assertR8cStrictC11(t, outputs, "r8g_floats.generated.c")
	runFoundationNativeHarness(t, outputs, "floats_harness.c", `#include "r8g_floats.generated.h"
int main(void) {
  if (concept_contextual__floats_call() != 1.0) return 1;
  if (concept_contextual__floats_negative() != -1.0) return 2;
  if (!concept_contextual__floats_is_twenty(20.0)) return 3;
  if (concept_contextual__floats_unit_circle(1.0, 0.0) != 0.0) return 4;
  if (concept_contextual__floats_tensor_literal() != 2.0) return 6;
  if (concept_contextual__floats_match_float_value() != 1.0) return 7;
  if (concept_contextual__floats_match_result_value() != 1.0) return 8;
  concept_sample sample = concept_contextual__floats_make_sample();
  if (sample.value != 1.25 || sample.pair.data[0] != 2.5 || sample.pair.data[1] != -3.75) return 5;
  return 0;
}`)
	for _, tc := range []struct{ source, code string }{
		{`float F() { return 1e100; }`, "FLOAT_LITERAL_OUT_OF_RANGE"},
		{`half F() { return 70000.0; }`, "FLOAT_LITERAL_OUT_OF_RANGE"},
	} {
		_, err := Parse("r8g_float_range.concept", "profile Core; "+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("%s: expected %s, got %v", tc.source, tc.code, err)
		}
	}
}

func TestPayloadFreeEnumGeometry(t *testing.T) {
	const source = `module Contextual.Enums; profile Core;
enum TokenKind { Number, Operator, End }
struct Token { TokenKind kind; int value; }
table<2> Rows { TokenKind kind; }
static_assert(SizeOf<TokenKind>() == 4, "enum backing is uint32");
static_assert(AlignOf<TokenKind>() == 4, "enum alignment is fixed");
static_assert(SizeOf<Token>() == 8, "enum-containing struct is fixed");
int Read(ReadOnlySpan<Token> tokens) { return tokens[0].value; }
int Main() {
  TokenKind[2] kinds = [TokenKind::Number, TokenKind::End];
  Token[2] tokens = [Token{kinds[0], 11}, Token{kinds[1], 12}];
  ReadOnlySpan<Token> span = ReadOnlySpan(tokens);
  return Read(span);
}
`
	module, err := Parse("r8g_enums.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	h := string(outputs["r8g_enums.generated.h"])
	if !strings.Contains(h, "uint32_t tag;") {
		t.Fatal("payload-free enum does not have fixed uint32 backing")
	}
	assertR8cStrictC11(t, outputs, "r8g_enums.generated.c")
	runFoundationNativeHarness(t, outputs, "enums_harness.c", `#include "r8g_enums.generated.h"
int main(void) { return concept_contextual__enums_main() == 11 ? 0 : 1; }`)
	_, err = Parse("r8g_payload_geometry.concept", `profile Core; enum Payload { Value(int x), Empty } static_assert(SizeOf<Payload>() == 4, "payload is not scalar");`)
	if err == nil || !strings.Contains(err.Error(), "CV4573") {
		t.Fatalf("payload enum unexpectedly acquired scalar geometry: %v", err)
	}
}

func TestBorrowedResultErrorRemap(t *testing.T) {
	const source = `module Contextual.Remap; profile Core;
enum SourceError { Missing }
enum TargetError { Unavailable }
struct Page { int value; }
TargetError Map(ref int calls) { calls++; return TargetError::Unavailable; }
Result<ref Page, SourceError> Find(ref Page page, bool fail) {
  if (fail) { return Result::Error(SourceError::Missing); }
  return Result::Ok(ref page);
}
Result<ref Page, TargetError> Access(ref Page page, bool fail, ref int calls) {
  ref Page found = Find(ref page, fail) ? else Map(ref calls);
  return Result::Ok(ref found);
}
int Check(bool fail) {
  Page page = Page{17};
  int calls = 0;
  Result<ref Page, TargetError> result = Access(ref page, fail, ref calls);
  int answer = match (result) {
    Result::Ok(found) => found.value,
    Result::Error(error) => -1,
  };
  return answer - calls * 100;
}
`
	module, err := Parse("r8g_remap.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["r8g_remap.generated.c"])
	if !strings.Contains(c, "cv_remapped_error") {
		t.Fatal("remap was not lowered on the error branch")
	}
	assertR8cStrictC11(t, outputs, "r8g_remap.generated.c")
	runFoundationNativeHarness(t, outputs, "remap_harness.c", `#include "r8g_remap.generated.h"
int main(void) {
  if (concept_contextual__remap_check(false) != 17) return 1;
  if (concept_contextual__remap_check(true) != -101) return 2;
  return 0;
}`)
	artifact, err := CompileSemanticModule("r8g_remap.concept", source, nil)
	if err != nil {
		t.Fatal(err)
	}
	consumer := `module Contextual.RemapConsumer; profile Core; import Contextual.Remap; int Main() { return Check(true); }`
	imported, err := ParseWithSemanticModules("r8g_remap_consumer.concept", consumer, map[string][]byte{"Contextual.Remap": artifact})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(imported, []byte(consumer)); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ source, diagnostic string }{
		{`Option<int> F(Option<int> value) { return value ? else 1; }`, "CV4543"},
		{`enum Source { Bad } enum Target { Bad } Result<int, Target> F(Result<int, Source> value) { int number = value ? else Source::Bad; return Result::Ok(number); }`, "CV4543"},
	} {
		_, err := Parse("r8g_remap_invalid.concept", "profile Core; "+tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.diagnostic) {
			t.Fatalf("%s: expected %s, got %v", tc.source, tc.diagnostic, err)
		}
	}
}

func TestContextualTypingArtifactsDeterministic(t *testing.T) {
	const source = `module Contextual.Determinism; profile Core;
enum Kind { One, Two }
struct Value { Kind kind; uint8[4] bytes; double number; }
Value Make() { return Value{Kind::One, [1, 2, 3, 4], -1.25}; }
`
	var firstC, firstMIR, firstArtifact []byte
	for run := 0; run < 100; run++ {
		module, err := Parse("r8g_determinism.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := CompileSemanticModule("r8g_determinism.concept", source, nil)
		if err != nil {
			t.Fatal(err)
		}
		generatedC, mir := outputs["r8g_determinism.generated.c"], outputs["r8g_determinism.mir.json"]
		if run == 0 {
			firstC, firstMIR, firstArtifact = generatedC, mir, artifact
			continue
		}
		if !bytes.Equal(firstC, generatedC) || !bytes.Equal(firstMIR, mir) || !bytes.Equal(firstArtifact, artifact) {
			t.Fatalf("R8g artifact bytes changed on run %d", run+1)
		}
	}
}
