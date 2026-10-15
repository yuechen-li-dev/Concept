package concept

import (
	"strings"
	"testing"
)

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
