package concept

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var inferenceValidFixtures = []string{
	"infer_basic.concept", "infer_guarded.concept", "infer_disabled_candidate_zero.concept",
	"infer_probability_access.concept", "infer_hardmax.concept", "infer_decide_hardmax_equivalence.concept",
	"infer_tiny_linear_expert.concept", "infer_automata_state_scores.concept", "infer_machine_field_scores.concept",
	"infer_evaluation_counts.concept", "infer_large_logits.concept", "infer_infinity_policy.concept",
	"infer_pass_return.concept",
	"infer_tensor_score.concept",
	"transition_infer_hardmax.concept", "transition_infer_guarded.concept", "transition_infer_cleanup.concept",
}

var inferenceInvalidFixtures = map[string]string{
	"infer_bad_guard.concept":                 "INFER_GUARD_REQUIRES_BOOL",
	"infer_int_score_rejected.concept":        "INFER_SCORE_REQUIRES_FLOAT",
	"infer_unknown_candidate_access.concept":  "INFERENCE_UNKNOWN_CANDIDATE",
	"infer_empty.concept":                     "INFER_EMPTY",
	"infer_duplicate_candidate.concept":       "INFER_DUPLICATE_CANDIDATE",
	"infer_unknown_candidate.concept":         "INFER_UNKNOWN_CANDIDATE",
	"infer_truthiness_rejected.concept":       "CV4186",
	"transition_infer_without_policy.concept": "TRANSITION_INFER_REQUIRES_POLICY",
	"transition_infer_unknown_policy.concept": "TRANSITION_INFER_UNKNOWN_POLICY",
	"transition_infer_unknown_target.concept": "TRANSITION_INFER_UNKNOWN_TARGET",
}

func inferenceFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "inference", class, file)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	return outputs
}

func TestInferenceConformance(t *testing.T) {
	for _, file := range inferenceValidFixtures {
		t.Run("valid/"+file, func(t *testing.T) { inferenceFixture(t, "valid", file) })
	}
	for file, code := range inferenceInvalidFixtures {
		t.Run("invalid/"+file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "inference", "invalid", file)
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			var diagnostic Diagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != code {
				t.Fatalf("diagnostic = %v, want %s", err, code)
			}
		})
	}
}

func TestInferenceNativeC11(t *testing.T) {
	cases := []struct{ file, call, condition, include string }{
		{"infer_basic.concept", "concept_infer_basic_main()", "fabsf(value - 0.5f) < 0.0001f", "#include <math.h>\n"},
		{"infer_disabled_candidate_zero.concept", "concept_infer_disabled_candidate_zero_main()", "value == 0.0f", ""},
		{"infer_probability_access.concept", "concept_infer_probability_access_main()", "fabsf(value - 0.75f) < 0.0001f", "#include <math.h>\n"},
		{"infer_large_logits.concept", "concept_infer_large_logits_main()", "fabsf(value - 1.0f) < 0.0001f", "#include <math.h>\n"},
		{"infer_tiny_linear_expert.concept", "concept_infer_tiny_linear_expert_main(1.0f, 0.5f)", "fabsf(value - 0.7407749f) < 0.0001f", "#include <math.h>\n"},
		{"infer_pass_return.concept", "concept_infer_pass_return_main()", "value > 0.73f && value < 0.74f", ""},
		{"infer_tensor_score.concept", "concept_infer_tensor_score_main()", "value > 0.73f && value < 0.74f", ""},
		{"infer_infinity_policy.concept", "concept_infer_infinity_policy_main(INFINITY, INFINITY, 1.0f)", "fabsf(value - 0.5f) < 0.0001f", "#include <math.h>\n"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := inferenceFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := tc.include + fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { float value = %s; return %s ? 0 : 1; }\n", base, tc.call, tc.condition)
			runFoundationNativeHarness(t, outputs, "inference_harness.c", harness)
			body := string(outputs[base+".generated.c"])
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "tensorflow", "onnx", "random", "scheduler"} {
				if strings.Contains(strings.ToLower(body), forbidden) {
					t.Fatalf("forbidden inference runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
	integerCases := []struct {
		file string
		want int
	}{{"infer_hardmax.concept", 2}, {"infer_decide_hardmax_equivalence.concept", 111}, {"infer_evaluation_counts.concept", 22}, {"transition_infer_hardmax.concept", 2}, {"transition_infer_guarded.concept", 2}, {"transition_infer_cleanup.concept", 1}}
	for _, tc := range integerCases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := inferenceFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == %d ? 0 : 1; }\n", base, base, tc.want)
			runFoundationNativeHarness(t, outputs, "inference_harness.c", harness)
			body := strings.ToLower(string(outputs[base+".generated.c"]))
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "random", "scheduler"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("forbidden inference runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
}

func TestInferenceMIRPlannerAndLowering(t *testing.T) {
	outputs := inferenceFixture(t, "valid", "infer_basic.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["infer_basic.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	inference := mir.Functions[0].Inferences[0]
	if inference.ScoreType != "float" || inference.Normalization != "StableSoftMax" || inference.Temperature != 1.0 || inference.NoEnabledPolicy != "Panic" || inference.NaNPolicy != "Panic" || len(inference.Candidates) != 2 {
		t.Fatalf("infer MIR drift: %+v", inference)
	}
	path := filepath.Join("..", "..", "language", "evt1", "inference", "valid", "infer_basic.concept")
	source, _ := os.ReadFile(path)
	module, _ := Parse(filepath.ToSlash(path), string(source))
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	inferencePlan := plan.Functions[0].Inferences[0]
	if inferencePlan.Normalization != "StableSoftMax" || !inferencePlan.MaxSubtraction || inferencePlan.Storage != "InlineFixed" || inferencePlan.SelectedSIMD {
		t.Fatalf("inference plan drift: %+v", inferencePlan)
	}
	for _, target := range []TargetCapabilities{X86_64GenericTarget(), AArch64GenericTarget()} {
		planned, err := GeneratePlan(module, target)
		if err != nil {
			t.Fatal(err)
		}
		var nativePlan LoweringPlan
		if err := json.Unmarshal(planned, &nativePlan); err != nil {
			t.Fatal(err)
		}
		if nativePlan.Functions[0].Inferences[0].SelectedSIMD || nativePlan.Functions[0].Inferences[0].Normalization != "StableSoftMax" {
			t.Fatalf("target %s changed R5c inference plan", target.Architecture)
		}
	}
	malformed := mir
	malformed.Functions[0].Inferences[0].Normalization = "NaiveSoftMax"
	if err := evt1ValidateMIR(malformed); diagnosticCode(err) != "INFER_MIR_INVALID" {
		t.Fatalf("malformed infer MIR diagnostic = %v", err)
	}
	body := string(outputs["infer_basic.generated.c"])
	for _, required := range []string{"expf(", "-INFINITY", "isinf(", "candidate_order", "inference score is NaN"} {
		if !strings.Contains(body, required) {
			t.Fatalf("lowering omits %q", required)
		}
	}
	transition := inferenceFixture(t, "valid", "transition_infer_hardmax.concept")
	var transitionMIR MIR
	if err := json.Unmarshal(transition["transition_infer_hardmax.mir.json"], &transitionMIR); err != nil {
		t.Fatal(err)
	}
	tm := transitionMIR.Automata[0].Machines[0].States[0].TransitionInferences[0]
	if tm.ScoreType != "float" || tm.Policy != "HardMax" || tm.Normalization != "StableSoftMax" || tm.CleanupEdge != "TransientBeforeStateUpdate" {
		t.Fatalf("transition infer MIR drift: %+v", tm)
	}
	transitionBody := string(transition["transition_infer_hardmax.generated.c"])
	if strings.Index(transitionBody, "expf(") > strings.Index(transitionBody, "current_state =") {
		t.Fatal("transition state updated before inference")
	}
	cleanupBody := string(inferenceFixture(t, "valid", "transition_infer_cleanup.concept")["transition_infer_cleanup.generated.c"])
	dropAt := strings.Index(cleanupBody, "concept_transition_infer_cleanup_drop(token);")
	stateAt := strings.Index(cleanupBody, "current_state = concept_chooser_state_run_running")
	if dropAt < 0 || stateAt < 0 || dropAt > stateAt {
		t.Fatal("transition inference cleanup must precede state update")
	}
}

func TestInferenceTerminalPanicPaths(t *testing.T) {
	cases := []struct{ file, call, reason string }{
		{"infer_no_enabled_runtime.concept", "concept_infer_no_enabled_runtime_main()", "inference has no enabled candidates"},
		{"infer_nan_runtime.concept", "concept_infer_nan_runtime_main(NAN)", "inference score is NaN"},
		{"infer_all_negative_infinity_runtime.concept", "concept_infer_all_negative_infinity_runtime_main(-INFINITY)", "inference normalization has no finite support"},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			compiler, err := exec.LookPath("gcc")
			if err != nil {
				compiler, err = exec.LookPath("clang")
			}
			if err != nil {
				t.Skip("C compiler unavailable")
			}
			outputs := inferenceFixture(t, "invalid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			harness := "#include <math.h>\n#include \"" + base + ".generated.h\"\nint main(void) { (void)" + tc.call + "; return 0; }\n"
			harnessPath := filepath.Join(dir, "panic_harness.c")
			if err := os.WriteFile(harnessPath, []byte(harness), 0644); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "panic.exe")
			generated := filepath.Join(dir, base+".generated.c")
			if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, generated, harnessPath, "-o", exe).CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, out)
			}
			out, err := exec.Command(exe).CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.reason) {
				t.Fatalf("panic evidence missing: err=%v output=%s", err, out)
			}
		})
	}
}

func TestInferenceSoftmaxReference(t *testing.T) {
	p0 := math.Exp(0) / (math.Exp(0) + math.Exp(math.Log(3)))
	if math.Abs(p0-0.25) > 1e-12 {
		t.Fatal("test reference drift")
	}
}
