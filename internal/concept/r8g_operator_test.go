package concept

import (
	"bytes"
	"strings"
	"testing"
)

func TestR8gRequiredOperatorsCloseThroughArtifact(t *testing.T) {
	const producer = `module R8g.Operators; profile Core;
concept Productable<T> { requires T operator*(T left, T right); }
concept Addable<T> { requires T operator+(T left, T right); }
concept Equatable<T> { requires bool operator==(T left, T right); }
template <typename T> requires Productable<T>
T Product(T left, T right) { return left * right; }
template <typename T> requires Addable<T>
T Sum(T left, T right) { return left + right; }
template <typename T> requires Equatable<T>
bool Equal(T left, T right) { return left == right; }
`
	artifact, err := CompileSemanticModule("R8g/Operators.concept", producer, nil)
	if err != nil {
		t.Fatal(err)
	}
	const consumer = `module R8g.Consumer; profile Core; import R8g.Operators;
enum Signal { Ready, Stopped }
float FloatProduct() { return Product<float>(2.0, 3.0); }
double DoubleProduct() { return Product<double>(2.0, 3.0); }
float<m> DistanceSum() { return Sum<float<m>>(interpret 2.0 as float<m>, interpret 3.0 as float<m>); }
bool SameSignal() { return Equal<Signal>(Signal::Ready, Signal::Ready); }
`
	module, err := ParseWithSemanticModules("R8g/Consumer.concept", consumer, map[string][]byte{"R8g.Operators": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "consumer.generated.c")
	runFoundationNativeHarness(t, outputs, "consumer_harness.c", `#include "consumer.generated.h"
int main(void) {
  if (concept_r8g__consumer_float_product() != 6.0f) return 1;
  if (concept_r8g__consumer_double_product() != 6.0) return 2;
  if (concept_r8g__consumer_distance_sum() != 5.0f) return 3;
  if (!concept_r8g__consumer_same_signal()) return 4;
  return 0;
}`)
	for _, needle := range []string{"operator*", "operator+", "operator=="} {
		if !strings.Contains(string(outputs["consumer.mir.json"]), needle) {
			t.Fatalf("closed operator witness %s absent from MIR", needle)
		}
	}
}

func TestR8gOperatorArtifactAndOutputDeterministic(t *testing.T) {
	const producer = `module R8g.StableOperators; profile Core;
concept Equatable<T> { requires bool operator==(T left, T right); }
template <typename T> requires Equatable<T>
bool Equal(T left, T right) { return left == right; }
`
	const consumer = `module R8g.StableConsumer; profile Core; import R8g.StableOperators;
enum State { Ready, Stopped }
bool Main() { return Equal<State>(State::Ready, State::Ready); }
`
	var firstArtifact []byte
	var firstOutputs map[string][]byte
	for run := 0; run < 100; run++ {
		artifact, err := CompileSemanticModule("R8g/StableOperators.concept", producer, nil)
		if err != nil {
			t.Fatal(err)
		}
		module, err := ParseWithSemanticModules("R8g/StableConsumer.concept", consumer, map[string][]byte{"R8g.StableOperators": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(consumer))
		if err != nil {
			t.Fatal(err)
		}
		if run == 0 {
			firstArtifact, firstOutputs = artifact, outputs
			continue
		}
		if !bytes.Equal(artifact, firstArtifact) || len(outputs) != len(firstOutputs) {
			t.Fatalf("artifact or output set changed on run %d", run+1)
		}
		for name, body := range firstOutputs {
			if !bytes.Equal(body, outputs[name]) {
				t.Fatalf("%s changed on run %d", name, run+1)
			}
		}
	}
}

func TestR8gRequiredOperatorDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"unconstrained", `profile Core; template <typename T> T Product(T a, T b) { return a * b; }`, "CV4175"},
		{"wrong result", `profile Core; concept Wrong<T> { requires bool operator*(T a, T b); } template <typename T> requires Wrong<T> bool Product(T a, T b) { return a * b; } bool Use() { return Product<float>(2.0, 3.0); }`, "CV4156"},
		{"missing closed witness", `profile Core; concept Productable<T> { requires T operator*(T a, T b); } struct Pair { int x; } template <typename T> requires Productable<T> T Product(T a, T b) { return a * b; } Pair Use() { return Product<Pair>(Pair{2}, Pair{3}); }`, "CV4153"},
		{"quantity result changes dimension", `profile Core; concept Productable<T> { requires T operator*(T a, T b); } template <typename T> requires Productable<T> T Product(T a, T b) { return a * b; } float<m> Use() { return Product<float<m>>(interpret 2.0 as float<m>, interpret 3.0 as float<m>); }`, "CV4156"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("r8g_operator_invalid.concept", tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}

func TestR8gTypedStandardMathThroughArtifact(t *testing.T) {
	mathSource := r7pGoldenSource(t, "Standard", "Math.concept")
	artifact, err := CompileSemanticModule("Standard/Math.concept", mathSource, nil)
	if err != nil {
		t.Fatal(err)
	}
	const source = `module R8g.TypedMath; profile Core; import Standard.Math;
float<m> DistanceAbs(float<m> value) { return Abs(value); }
float<m> DistanceMin(float<m> left, float<m> right) { return Min(left, right); }
double<Pa> PressureMax(double<Pa> left, double<Pa> right) { return Max(left, right); }
double DoubleAbs(double value) { return Abs(value); }
double DoubleMin(double left, double right) { return Min(left, right); }
`
	module, err := ParseWithSemanticModules("R8g/TypedMath.concept", source, map[string][]byte{"Standard.Math": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "typedmath.generated.c")
	runFoundationNativeHarness(t, outputs, "typedmath_harness.c", `#include "typedmath.generated.h"
int main(void) {
  if (concept_r8g__typed_math_distance_abs(-3.0f) != 3.0f) return 1;
  if (concept_r8g__typed_math_distance_min(2.0f, 5.0f) != 2.0f) return 2;
  if (concept_r8g__typed_math_pressure_max(2.0, 5.0) != 5.0) return 3;
  if (concept_r8g__typed_math_double_abs(-3.0) != 3.0) return 4;
  if (concept_r8g__typed_math_double_min(2.0, 5.0) != 2.0) return 5;
  if ((1.0 / concept_r8g__typed_math_double_abs(-0.0)) < 0.0) return 6;
  return 0;
}`)
	if !strings.Contains(string(outputs["typedmath.mir.json"]), "operator-") {
		t.Fatal("typed Abs did not close unary operator witness")
	}
	const halfSource = `module R8g.HalfMath; profile Core; import Standard.Math;
half HalfAbs(half value) { return Abs(value); }
half HalfMin(half left, half right) { return Min(left, right); }
half HalfMax(half left, half right) { return Max(left, right); }
`
	halfModule, err := ParseWithSemanticModules("R8g/HalfMath.concept", halfSource, map[string][]byte{"Standard.Math": artifact})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(halfModule, []byte(halfSource)); err != nil {
		t.Fatal(err)
	}
}
