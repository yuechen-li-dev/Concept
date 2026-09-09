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

var automataTransitionValidFixtures = []string{
	"transition_match_enum.concept",
	"transition_match_payload_enum.concept",
	"transition_match_result.concept",
	"transition_match_updates_state.concept",
	"transition_decide_int_scores.concept",
	"transition_decide_float_scores.concept",
	"transition_decide_guarded.concept",
	"transition_decide_unguarded_fallback.concept",
	"transition_decide_tie_first_wins.concept",
	"transition_decide_same_target_multiple_candidates.concept",
	"transition_decide_current_state.concept",
	"transition_decide_shared_state_score.concept",
	"transition_decide_machine_field_score.concept",
	"transition_evaluation_counts.concept",
}

var automataTransitionInvalidFixtures = map[string]string{
	"transition_match_nonexhaustive.concept":      "TRANSITION_MATCH_NONEXHAUSTIVE",
	"transition_match_duplicate_arm.concept":      "TRANSITION_MATCH_DUPLICATE_ARM",
	"transition_match_unknown_target.concept":     "TRANSITION_MATCH_UNKNOWN_TARGET",
	"transition_match_requires_matchable.concept": "TRANSITION_MATCH_REQUIRES_MATCHABLE",
	"transition_match_invalid_pattern.concept":    "CV4110",
	"transition_decide_empty.concept":             "TRANSITION_DECIDE_EMPTY",
	"transition_decide_bad_guard.concept":         "TRANSITION_DECIDE_GUARD_REQUIRES_BOOL",
	"transition_decide_bad_score_type.concept":    "TRANSITION_DECIDE_SCORE_TYPE_INVALID",
	"transition_decide_mixed_score_type.concept":  "TRANSITION_DECIDE_SCORE_TYPE_MISMATCH",
	"transition_decide_unknown_target.concept":    "TRANSITION_DECIDE_UNKNOWN_TARGET",
}

func automataTransitionFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "automata", "transitions", class, file)
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

func TestAutomataTransitionsConformance(t *testing.T) {
	for _, file := range automataTransitionValidFixtures {
		t.Run("valid/"+file, func(t *testing.T) { automataTransitionFixture(t, "valid", file) })
	}
	for file, code := range automataTransitionInvalidFixtures {
		t.Run("invalid/"+file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "automata", "transitions", "invalid", file)
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

func TestAutomataTransitionsNativeC11(t *testing.T) {
	cases := []struct {
		file string
		want int
	}{
		{"transition_match_enum.concept", 1},
		{"transition_match_payload_enum.concept", 2},
		{"transition_match_result.concept", 1},
		{"transition_match_updates_state.concept", 11},
		{"transition_decide_int_scores.concept", 2},
		{"transition_decide_float_scores.concept", 2},
		{"transition_decide_guarded.concept", 2},
		{"transition_decide_unguarded_fallback.concept", 2},
		{"transition_decide_tie_first_wins.concept", 1},
		{"transition_decide_same_target_multiple_candidates.concept", 1},
		{"transition_decide_current_state.concept", 0},
		{"transition_decide_shared_state_score.concept", 2},
		{"transition_decide_machine_field_score.concept", 2},
		{"transition_evaluation_counts.concept", 122},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := automataTransitionFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			call := "concept_" + base + "_main"
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return %s() == %d ? 0 : 1; }\n", base, call, tc.want)
			runFoundationNativeHarness(t, outputs, "automata_transitions_harness.c", harness)
			body := string(outputs[base+".generated.c"])
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "coroutine", "scheduler"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
}

func TestAutomataTransitionsMIRAndPlanner(t *testing.T) {
	outputs := automataTransitionFixture(t, "valid", "transition_evaluation_counts.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["transition_evaluation_counts.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Automata) != 1 || len(mir.Automata[0].Machines) != 2 {
		t.Fatalf("missing R5b automata MIR: %+v", mir.Automata)
	}
	match := mir.Automata[0].Machines[0].States[0].TransitionMatches[0]
	decision := mir.Automata[0].Machines[1].States[0].TransitionDecisions[0]
	if !match.Exhaustive || match.NoMatchPolicy != "Panic" || match.CleanupEdge != "TransientBeforeStateUpdate" || len(match.Arms) != 2 {
		t.Fatalf("transition match MIR drift: %+v", match)
	}
	if decision.ScoreType.Name != "int" || decision.TiePolicy != "DeclarationOrderFirstMax" || decision.NoEnabledPolicy != "Panic" || len(decision.Candidates) != 3 || decision.Candidates[2].DeclarationOrder != 2 {
		t.Fatalf("transition decide MIR drift: %+v", decision)
	}

	path := filepath.Join("..", "..", "language", "evt1", "automata", "transitions", "valid", "transition_evaluation_counts.concept")
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
	matchPlan := plan.Automata[0].Machines[0].TransitionMatches[0]
	decidePlan := plan.Automata[0].Machines[1].TransitionDecisions[0]
	if matchPlan.Strategy != "CategoricalSwitch" || matchPlan.EvaluationOrder != "ScrutineeOnceThenSelectedTarget" {
		t.Fatalf("transition match plan drift: %+v", matchPlan)
	}
	if decidePlan.Strategy != "GuardedStraightLineHardmax" || decidePlan.TiePolicy != "DeclarationOrderFirstMax" || decidePlan.ScoreEvaluationOrder != "EnabledDeclarationOrderOnce" || decidePlan.BranchlessEligible {
		t.Fatalf("transition decide plan drift: %+v", decidePlan)
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
		candidatePlan := nativePlan.Automata[0].Machines[1].TransitionDecisions[0]
		if candidatePlan.Strategy != "GuardedStraightLineHardmax" || candidatePlan.BranchlessEligible {
			t.Fatalf("target %s selected an R5b optimization: %+v", target.Architecture, candidatePlan)
		}
	}

	malformed := mir
	malformed.Automata[0].Machines[1].States[0].TransitionDecisions[0].TiePolicy = "LastWins"
	if err := evt1ValidateMIR(malformed); diagnosticCode(err) != "TRANSITION_DECIDE_MIR_INVALID" {
		t.Fatalf("malformed decide MIR diagnostic = %v", err)
	}
	var malformedMatch MIR
	if err := json.Unmarshal(outputs["transition_evaluation_counts.mir.json"], &malformedMatch); err != nil {
		t.Fatal(err)
	}
	malformedMatch.Automata[0].Machines[0].States[0].TransitionMatches[0].Exhaustive = false
	if err := evt1ValidateMIR(malformedMatch); diagnosticCode(err) != "TRANSITION_MATCH_MIR_INVALID" {
		t.Fatalf("malformed match MIR diagnostic = %v", err)
	}
}

func TestAutomataTransitionsEvaluationAndCleanupShape(t *testing.T) {
	evidence := string(automataTransitionFixture(t, "valid", "transition_evaluation_counts.concept")["transition_evaluation_counts.generated.c"])
	if strings.Count(evidence, "concept_transition_evaluation_counts_count_guard") != 3 {
		t.Fatalf("each authored guard call must appear once (including definition):\n%s", evidence)
	}
	if strings.Count(evidence, "concept_transition_evaluation_counts_count_score") != 4 {
		t.Fatalf("each authored score call must appear once (including definition):\n%s", evidence)
	}
	if strings.Count(evidence, "concept_transition_evaluation_counts_read_signal") != 2 {
		t.Fatalf("match scrutinee call must appear once (including definition):\n%s", evidence)
	}
	if strings.Contains(evidence, ">=") {
		t.Fatal("hardmax lowering must use strict greater-than so the first maximum wins")
	}

	cleanup := string(automataTransitionFixture(t, "valid", "transition_match_updates_state.concept")["transition_match_updates_state.generated.c"])
	stateAt := strings.Index(cleanup, "current_state = concept_toggle_state_run_running")
	dropAt := strings.Index(cleanup, "concept_transition_match_updates_state_drop(temporary);")
	if dropAt < 0 || stateAt < 0 || dropAt > stateAt {
		t.Fatalf("transient cleanup was not emitted before transition commit:\n%s", cleanup)
	}
	decideCleanup := string(automataTransitionFixture(t, "valid", "transition_decide_int_scores.concept")["transition_decide_int_scores.generated.c"])
	decideDropAt := strings.Index(decideCleanup, "concept_transition_decide_int_scores_drop(temporary);")
	decideStateAt := strings.Index(decideCleanup, "current_state = cv_decision_best_state")
	if decideDropAt < 0 || decideStateAt < 0 || decideDropAt > decideStateAt {
		t.Fatalf("decision transient cleanup was not emitted before state commit:\n%s", decideCleanup)
	}
	if !strings.Contains(cleanup, "machine transition match found no matching case") {
		t.Fatal("stable transition match no-case panic reason missing from defensive C path")
	}
}

func TestAutomataTransitionsTerminalPanicPaths(t *testing.T) {
	cases := []struct {
		file, call, include, reason string
	}{
		{"transition_decide_no_enabled_runtime.concept", "concept_transition_decide_no_enabled_runtime_main()", "", "machine decision transition has no enabled candidates"},
		{"transition_decide_nan_runtime.concept", "concept_transition_decide_nan_runtime_main(NAN)", "#include <math.h>\n", "machine decision transition score is NaN"},
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
			outputs := automataTransitionFixture(t, "invalid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			harness := tc.include + "#include \"" + base + ".generated.h\"\nint main(void) { (void)" + tc.call + "; return 0; }\n"
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
			if err == nil || !strings.Contains(string(out), tc.reason) || !strings.Contains(string(out), "Concept panic at") {
				t.Fatalf("deterministic panic evidence missing: err=%v output=%s", err, out)
			}
		})
	}
}
