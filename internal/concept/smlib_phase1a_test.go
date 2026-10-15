package concept

import (
	"fmt"
	"strings"
	"testing"
)

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
