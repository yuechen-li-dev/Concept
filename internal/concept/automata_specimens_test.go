package concept

import (
	"path/filepath"
	"strings"
	"testing"
)

// The automata specimens in examples/evt1 are Core step machines with input.
// Their M-era originals (signal dispatch, effects, actuators) are frozen in
// legacy/evt1-specimens; the effect/actuator pattern now lives in
// libraries/Standard/outbox.concept_test.

func automataSpecimenOutputs(t *testing.T, name string) Outputs {
	t.Helper()
	src := readEVT1Fixture(t, name)
	module, err := Parse(filepath.ToSlash(filepath.Join("examples", "evt1", name)), src)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	return outputs
}

func TestUnusedAutomataLeaveNoGeneratedCode(t *testing.T) {
	outputs := automataSpecimenOutputs(t, "lifecycle_automata.concept")
	header := string(outputs["lifecycle_automata.generated.h"])
	body := string(outputs["lifecycle_automata.generated.c"])
	for _, forbidden := range []string{"ResourceLifecycle", "resource_lifecycle", "SweepMachine", "AwaitingResume", "Cleanup", "LifecycleSignal", "step_outcome"} {
		if strings.Contains(header, forbidden) || strings.Contains(body, forbidden) {
			t.Fatalf("automata-only symbol leaked into generated output: %s", forbidden)
		}
	}
}

func TestAutomataSpecimensNativeC11(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file, checks string
	}{
		{"lifecycle_automata.concept", `
  if (concept_lifecycle_automata_root_retry_budget() != 2) return 1;
  if (concept_lifecycle_automata_derived_stack_depth() != 3) return 2;
  if (!concept_lifecycle_automata_finish_is_explicit()) return 3;`},
		// Outcome codes: 1 Transitioned, 2 Unhandled, 3 Finished,
		// 4 AlreadyFinished, 5 Ambiguous.
		{"automata_dispatch.concept", `
  if (concept_automata_dispatch_single_step_finish_code() != 34) return 1;
  if (concept_automata_dispatch_unhandled_preserves_state_code() != 21) return 2;
  if (concept_automata_dispatch_nested_push_resumes_caller_code() != 111113) return 3;
  if (concept_automata_dispatch_root_terminal_continuation_finishes_immediately_code() != 1111134) return 4;
  if (concept_automata_dispatch_child_failure_returns_to_parent_code() != 1111137) return 5;
  if (concept_automata_dispatch_independent_instances_stay_independent_code() != 1121) return 6;`},
		{"guarded_transitions.concept", `
  concept_lifecycle_context unique = {true, false};
  concept_lifecycle_context fallback = {false, false};
  concept_lifecycle_context ambiguous = {true, true};
  if (concept_guarded_transitions_unique_guard_selection_code(unique) != 13) return 1;
  if (concept_guarded_transitions_fallback_selection_code(fallback) != 113) return 2;
  if (concept_guarded_transitions_guarded_unhandled_preserves_state_code(fallback) != 1213) return 3;
  if (concept_guarded_transitions_ambiguous_preserves_state_code(ambiguous) != 51513) return 4;
  if (concept_guarded_transitions_already_finished_skips_guard_selection_code(unique) != 4) return 5;`},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := automataSpecimenOutputs(t, tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := "#include \"" + base + ".generated.h\"\n#include <stdbool.h>\nint main(void) {" + tc.checks + "\n  return 0;\n}\n"
			runFoundationNativeHarness(t, outputs, base+"_harness.c", harness)
		})
	}
}
