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
	source := runtimeTestSource(t, "contextual_arrays.concept_test")
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
}

func TestContextualFloatLiterals(t *testing.T) {
	source := runtimeTestSource(t, "contextual_floats.concept_test")
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
	source := runtimeTestSource(t, "enum_geometry.concept_test")
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
	_, err = Parse("r8g_payload_geometry.concept", `profile Core; enum Payload { Value(int x), Empty } static_assert(SizeOf<Payload>() == 4, "payload is not scalar");`)
	if err == nil || !strings.Contains(err.Error(), "CV4573") {
		t.Fatalf("payload enum unexpectedly acquired scalar geometry: %v", err)
	}
}

func TestBorrowedResultErrorRemap(t *testing.T) {
	source := runtimeTestSource(t, "error_remap.concept_test")
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
