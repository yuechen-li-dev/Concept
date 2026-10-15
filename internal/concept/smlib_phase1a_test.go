package concept

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestSMLibPhase1aTensorValueShapeProofs(t *testing.T) {
	source := `profile Core;
int Check(int count) {
    int<array>[4] storage = [1 ...];
    tensor<int,1> fixed = Tensor(storage);
    Assert.Concept<FixedShape>(fixed, "fixed backing extent is retained");
    ref int<array>[count] view = bind storage;
    tensor<int,1> dynamic = Tensor(view);
    Assert.Concept<RuntimeShape>(dynamic, "runtime backing extent is retained");
    return 0;
}
int Main() { return Check(4); }`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("tensor_proof.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "tensor_proof.generated.c")
		runFoundationNativeHarness(t, outputs, "tensor_proof_harness.c", "#include \"tensor_proof.generated.h\"\nint main(void){return concept_tensor_proof_main();}")
	}
	for _, bad := range []string{
		strings.Replace(source, "<FixedShape>(fixed", "<RuntimeShape>(fixed", 1),
		strings.Replace(source, "<RuntimeShape>(dynamic", "<FixedShape>(dynamic", 1),
	} {
		if _, err := Parse("invalid.concept", bad); err == nil || !strings.Contains(err.Error(), "CONCEPT_ASSERT_DISPROVEN") {
			t.Fatalf("wrong shape proof diagnostic: %v", err)
		}
	}
}

func TestSMLibPhase1aDependentStaticAssert(t *testing.T) {
	provider := `module Library.Square; profile Core;
template <usize N, usize M> int Extent() {
    static_assert(N > 0u and M == N*N, "square capacity");
    return N as int;
}`
	artifact := buildSemanticArtifact(t, "Library/Square.concept", provider, nil)
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		source := `module Consumer; profile Core; import Library.Square;
int Main() { return Extent<3,9>() + Extent<8,64>(); }`
		module, err := ParseWithSemanticModules("consumer.concept", source, map[string][]byte{"Library.Square": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarness(t, outputs, "static_generic_harness.c", "#include \"consumer.generated.h\"\nint main(void) { return concept_consumer_main() == 11 ? 0 : 1; }")
		for _, args := range []string{"0,0", "3,8"} {
			bad := fmt.Sprintf("profile Core; import Library.Square; int Main(){return Extent<%s>();}", args)
			if _, err := ParseWithSemanticModules("invalid.concept", bad, map[string][]byte{"Library.Square": artifact}); err == nil || !strings.Contains(err.Error(), "CV4207") {
				t.Fatalf("invalid square accepted or wrong diagnostic: %v", err)
			}
		}
	}
	for _, body := range []string{
		`static_assert(N == 3u and runtime > 0, "runtime operand");`,
		`static_assert(N);`,
		`static_assert(N > 0u, N);`,
	} {
		source := "profile Core; template <usize N> int Check(int runtime) {" + body + "return 0;}"
		if _, err := Parse("invalid.concept", source); err == nil {
			t.Fatal("invalid dependent assertion accepted")
		}
	}
}

func TestSMLibPhase1aImportedTensorTemplate(t *testing.T) {
	provider := `module Library.TensorKernel; profile Core;
template <usize N> void Scale(ref const double<array>[N] input, ref double<array>[N] output) {
    tensor<double, 1> X = Tensor(input);
    tensor<double, 1> Y = Tensor(output);
    Y[i] = X[i] * 2.0;
}`
	artifact := buildSemanticArtifact(t, "Library/TensorKernel.concept", provider, nil)
	source := `module Consumer; profile Core; import Library.TensorKernel;
double Main() {
    double<array>[3] x = [1.0, 2.0, 3.0];
    double<array>[3] y = [0.0 ...];
    Scale<3>(ref const x, ref y);
    return y[2];
}`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("consumer.concept", source, map[string][]byte{"Library.TensorKernel": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "consumer.generated.c")
		runFoundationNativeHarness(t, outputs, "tensor_generic_harness.c", "#include \"consumer.generated.h\"\nint main(void) { return concept_consumer_main() == 6.0 ? 0 : 1; }")
		invalid := strings.Replace(source, "double<array>[3] y", "double<array>[2] y", 1)
		if _, err := ParseWithSemanticModules("invalid.concept", invalid, map[string][]byte{"Library.TensorKernel": artifact}); err == nil {
			t.Fatal("dimension-mismatched tensor generic accepted")
		}
	}
}

func TestSMLibPhase1aClosedCallbackEnvironments(t *testing.T) {
	provider := `module Library.Adapter; profile Core;
int Apply(callback<int -> int> fn) { return fn(7); }
template <typename T> int Forward(ref const T item) {
    const auto callbackValue = callback(int x) with (ref const item) { return x; };
    Assert.Concept<LifetimeSafe>(callbackValue, "adapter borrows only the parameter");
    int<array>[3] storage = [0 ...];
    Assert.Concept<Rank<1>>(storage, "proof parameters survive concrete instantiation");
    return Apply(ref callbackValue);
}`
	artifact := buildSemanticArtifact(t, "Library/Adapter.concept", provider, nil)
	source := `module Consumer; profile Core; import Library.Adapter;
int Main() { int a=1; double b=2.0; return Forward<int>(ref const a)+Forward<double>(ref const b)+Forward<int>(ref const a); }`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := ParseWithSemanticModules("consumer.concept", source, map[string][]byte{"Library.Adapter": artifact})
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "consumer.generated.c")
		runFoundationNativeHarness(t, outputs, "generic_callback_harness.c", "#include \"consumer.generated.h\"\nint main(void){return concept_consumer_main()==21?0:1;}")
		for name, data := range outputs {
			if strings.HasSuffix(name, ".mir.json") {
				var mir MIR
				if err := json.Unmarshal(data, &mir); err != nil {
					t.Fatal(err)
				}
				if len(mir.CallbackWitnesses) != 2 || mir.CallbackWitnesses[0].ConcreteCallable == mir.CallbackWitnesses[1].ConcreteCallable {
					t.Fatalf("closed callback identities: %+v", mir.CallbackWitnesses)
				}
			}
		}
	}
}

func TestSMLibPhase1aGenericOwnedCaptureCleanup(t *testing.T) {
	source := `profile Core;
extern "C" void Observed(int id);
struct First { int id; } struct Second { int id; }
void Drop(owned First x) { Observed(x.id); }
void Drop(owned Second x) { Observed(x.id); }
template <typename T> int Scope(owned T item) {
    const auto fn = callback() with (move item) { return 7; };
    return fn();
}
int Main() { owned First a=First{1}; owned Second b=Second{2}; return Scope<First>(move a)+Scope<Second>(move b); }`
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		module, err := Parse("generic_owner.concept", source)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		runFoundationNativeHarness(t, outputs, "generic_owner_harness.c", `#include "generic_owner.generated.h"
static int drops=0;void Observed(int id){drops+=id;}
int main(void){return concept_generic_owner_main()==14 && drops==3 ? 0:1;}`)
	}
	invalid := `profile Core; template <typename T> callback<int -> int> Escape(T seed) { const auto fn=callback(int x) with(seed){return x;};return ref fn; }`
	if _, err := Parse("generic_escape.concept", invalid); err == nil {
		t.Fatal("generic local callback environment escaped")
	}
}
