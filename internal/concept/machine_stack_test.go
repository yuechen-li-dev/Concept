package concept

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var machineStackValidFixtures = []string{
	"machine_push_pop_neutral.concept", "machine_child_success.concept", "machine_child_failure.concept",
	"machine_parent_resume.concept", "machine_child_yield_then_complete.concept", "machine_shared_state_nested.concept",
	"machine_private_state_nested.concept", "machine_owned_child_cleanup.concept", "machine_result_after_completion.concept",
	"machine_multiple_nested_frames.concept", "machine_recursive_frames.concept",
	"machine_owned_completion_transfer.concept",
}

var machineStackInvalidFixtures = map[string]string{
	"machine_push_unknown.concept":              "MACHINE_PUSH_UNKNOWN",
	"machine_stack_underflow.concept":           "MACHINE_STACK_UNDERFLOW",
	"machine_complete_wrong_value_type.concept": "MACHINE_COMPLETE_VALUE_TYPE_MISMATCH",
	"machine_fail_wrong_error_type.concept":     "MACHINE_FAIL_ERROR_TYPE_MISMATCH",
	"machine_double_completion.concept":         "MACHINE_DOUBLE_COMPLETION",
}

func machineStackFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", class, file)
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

func TestMachineStackConformance(t *testing.T) {
	for _, file := range machineStackValidFixtures {
		t.Run("valid/"+file, func(t *testing.T) { machineStackFixture(t, "valid", file) })
	}
	for file, code := range machineStackInvalidFixtures {
		t.Run("invalid/"+file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "invalid", file)
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
	// Overflow and pre-completion Result are deliberately runtime-negative:
	// both are well typed and have stable generated panic paths.
	machineStackFixture(t, "invalid", "machine_stack_overflow.concept")
	machineStackFixture(t, "invalid", "machine_result_before_completion.concept")
}

func TestMachineStackNativeC11(t *testing.T) {
	cases := []struct {
		file, call string
		want       int
	}{
		{"machine_push_pop_neutral.concept", "concept_machine_push_pop_neutral_main", 11},
		{"machine_child_success.concept", "concept_machine_child_success_main", 42},
		{"machine_child_failure.concept", "concept_machine_child_failure_main", 23},
		{"machine_parent_resume.concept", "concept_machine_parent_resume_main", 7},
		{"machine_child_yield_then_complete.concept", "concept_machine_child_yield_then_complete_main", 12},
		{"machine_shared_state_nested.concept", "concept_machine_shared_state_nested_main", 17},
		{"machine_private_state_nested.concept", "concept_machine_private_state_nested_main", 4},
		{"machine_owned_child_cleanup.concept", "concept_machine_owned_child_cleanup_main", 0},
		{"machine_result_after_completion.concept", "concept_machine_result_after_completion_main", 17},
		{"machine_multiple_nested_frames.concept", "concept_machine_multiple_nested_frames_main", 111},
		{"machine_recursive_frames.concept", "concept_machine_recursive_frames_main", 3},
		{"machine_owned_completion_transfer.concept", "concept_machine_owned_completion_transfer_main", 0},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := machineStackFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return %s() == %d ? 0 : 1; }\n", base, tc.call, tc.want)
			runFoundationNativeHarness(t, outputs, "machine_stack_harness.c", harness)
			body := string(outputs[base+".generated.c"])
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "scheduler", "program_counter", "setjmp"} {
				if strings.Contains(strings.ToLower(body), forbidden) {
					t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
}

func TestMachineStackRuntimeNegative(t *testing.T) {
	for _, file := range []string{"machine_stack_overflow.concept", "machine_result_before_completion.concept"} {
		t.Run(file, func(t *testing.T) {
			outputs := machineStackFixture(t, "invalid", file)
			compiler, err := exec.LookPath("gcc")
			if err != nil {
				compiler, err = exec.LookPath("clang")
			}
			if err != nil {
				t.Skip("C compiler unavailable")
			}
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			base := strings.TrimSuffix(file, ".concept")
			harness := filepath.Join(dir, "negative.c")
			if err := os.WriteFile(harness, []byte(fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return %s(); }\n", base, "concept_"+base+"_main")), 0644); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "negative.exe")
			if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, filepath.Join(dir, base+".generated.c"), harness, "-o", exe).CombinedOutput(); err != nil {
				t.Fatalf("compile: %v\n%s", err, out)
			}
			out, err := exec.Command(exe).CombinedOutput()
			if err == nil {
				t.Fatal("runtime-negative specimen unexpectedly succeeded")
			}
			message := string(out)
			if file == "machine_stack_overflow.concept" && !strings.Contains(message, "automata machine stack capacity exceeded") {
				t.Fatalf("overflow reason drift: %s", message)
			}
			if file == "machine_result_before_completion.concept" && !strings.Contains(message, "machine result cannot be read before completion") {
				t.Fatalf("result reason drift: %s", message)
			}
		})
	}
}

func TestMachineStackMIRAndPlanner(t *testing.T) {
	outputs := machineStackFixture(t, "valid", "machine_child_success.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["machine_child_success.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	stack := mir.Automata[0].MachineStack
	if stack == nil || stack.Capacity != 8 || stack.Storage != "InlineBoundedSpecializedFrames" || stack.Scheduler != "None" || stack.Continuation != "ExplicitState" {
		t.Fatalf("stack MIR drift: %+v", stack)
	}
	if got := mir.Automata[0].Machines[0].States[0].MachineControl[0]; got.Kind != "push_machine" || got.Machine != "Child" || got.ResumeState != "Resumed" {
		t.Fatalf("push MIR drift: %+v", got)
	}
	if got := mir.Automata[0].Machines[1].States[0].MachineControl[0]; got.Kind != "pop_machine" || got.Outcome != "Success" || got.PayloadType != "int" {
		t.Fatalf("completion MIR drift: %+v", got)
	}
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "valid", "machine_child_success.concept")
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
	p := plan.Automata[0].MachineStack
	if p == nil || p.Storage != "InlineBoundedStack" || p.Scheduler != "None" || p.Continuation != "ExplicitState" || len(p.Pushes) != 1 || len(p.Completions) != 1 {
		t.Fatalf("stack plan drift: %+v", p)
	}
	malformed := mir
	malformed.Automata[0].MachineStack.Scheduler = "HiddenRuntime"
	if err := evt1ValidateMIR(malformed); diagnosticCode(err) != "MACHINE_STACK_MIR_INVALID" {
		t.Fatalf("malformed stack MIR = %v", err)
	}
}

func TestMachineStackOwnedOutcomeTransferShape(t *testing.T) {
	neutral := string(machineStackFixture(t, "valid", "machine_owned_child_cleanup.concept")["machine_owned_child_cleanup.generated.c"])
	if strings.Count(neutral, "concept_machine_owned_child_cleanup_drop(instance->child_frames[slot].resource);") != 1 {
		t.Fatal("owned child frame does not have exactly one runtime drop site")
	}
	body := string(machineStackFixture(t, "valid", "machine_owned_completion_transfer.concept")["machine_owned_completion_transfer.generated.c"])
	for _, required := range []string{"resource_moved = true", "if (!instance->child_frames[slot].resource_moved)", "last_child_outcome.success", "last_child_outcome.tag == 2u"} {
		if !strings.Contains(body, required) {
			t.Fatalf("owned outcome transfer evidence missing %q", required)
		}
	}
}
