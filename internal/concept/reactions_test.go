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

// Input reactions: `automata X with input T`, `on Pattern [when g |
// otherwise] => ...`, state-level `otherwise`, and Step(instance, Machine,
// input) returning a must-use StepOutcome.

var reactionValidFixtures = map[string]int{
	// Outcomes 1 Transitioned, 2 Unhandled, 3 Ambiguous, then state and pushes.
	"door_reactions.concept":            12111131,
	"internal_and_input_states.concept": 1201,
}

var reactionInvalidFixtures = map[string]string{
	"automata_input_not_enum.concept":          "AUTOMATA_INPUT_INVALID",
	"on_duplicate_otherwise.concept":           "ON_DUPLICATE",
	"on_guard_not_bool.concept":                "ON_GUARD_REQUIRES_BOOL",
	"on_mixed_state_body.concept":              "ON_MIXED_STATE_BODY",
	"on_otherwise_unreachable.concept":         "ON_OTHERWISE_UNREACHABLE",
	"on_otherwise_without_guards.concept":      "ON_OTHERWISE_WITHOUT_GUARDS",
	"on_overlap_guarded_and_unguarded.concept": "ON_OVERLAP",
	"on_overlap_same_guard.concept":            "ON_OVERLAP",
	"on_overlap_unguarded_twice.concept":       "ON_OVERLAP",
	"on_requires_input.concept":                "ON_REQUIRES_INPUT",
	"on_unknown_target.concept":                "MACHINE_UNKNOWN_STATE",
	"step_input_unexpected.concept":            "MACHINE_STEP_INPUT_UNEXPECTED",
	"step_outcome_ignored.concept":             "MUST_USE_RESULT_IGNORED",
	"step_requires_input.concept":              "MACHINE_STEP_REQUIRES_INPUT",
}

func reactionFixturePath(class, file string) string {
	return filepath.Join("..", "..", "language", "evt1", "automata", "reactions", class, file)
}

func reactionFixture(t *testing.T, file string) (Module, []byte, Outputs) {
	t.Helper()
	path := reactionFixturePath("valid", file)
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
	return module, source, outputs
}

func TestReactionsConformance(t *testing.T) {
	for file := range reactionValidFixtures {
		t.Run("valid/"+file, func(t *testing.T) { reactionFixture(t, file) })
	}
	for file, code := range reactionInvalidFixtures {
		t.Run("invalid/"+file, func(t *testing.T) {
			path := reactionFixturePath("invalid", file)
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

func TestReactionsNativeC11(t *testing.T) {
	t.Parallel()
	for file, want := range reactionValidFixtures {
		t.Run(file, func(t *testing.T) {
			_, _, outputs := reactionFixture(t, file)
			base := strings.TrimSuffix(file, ".concept")
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == %d ? 0 : 1; }\n", base, base, want)
			runFoundationNativeHarness(t, outputs, "reactions_harness.c", harness)
			body := string(outputs[base+".generated.c"])
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "scheduler"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
}

func TestReactionsMIRAndPlan(t *testing.T) {
	module, _, outputs := reactionFixture(t, "door_reactions.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["door_reactions.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	door := mir.Automata[0]
	if door.InputType != "DoorSignal" {
		t.Fatalf("input type = %q", door.InputType)
	}
	var got []string
	for _, reaction := range door.Machines[0].States[0].Reactions {
		got = append(got, fmt.Sprintf("%s|%s|%t|%s", reaction.Pattern, reaction.Guard, reaction.Otherwise, reaction.Target))
	}
	want := []string{
		"DoorSignal::Push|((not locked) and (force > 2))|false|Open",
		"DoorSignal::Push|(force > 9)|false|Broken",
		"DoorSignal::Push||true|",
		"DoorSignal::Lock||false|",
		"DoorSignal::Unlock||false|",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("reactions drifted:\n%s", strings.Join(got, "\n"))
	}
	catchAll := door.Machines[0].States[1].Reactions[1]
	if !catchAll.CatchAll || catchAll.Target != "Open" {
		t.Fatalf("catch-all drifted: %+v", catchAll)
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(planBytes), "InputReactionSwitch") {
		t.Fatal("plan omitted the input reaction strategy")
	}
}

func TestReactionsNormalAndVerify(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc unavailable")
	}
	source, err := os.ReadFile(reactionFixturePath("valid", "door_reactions.concept"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Door.concept"), []byte("module Door;\n"+string(source)), 0644); err != nil {
		t.Fatal(err)
	}
	facts := `module DoorTests;
profile Core;
import Door;

[[fact]]
void OutcomesFollowTheReactionSet()
{
    Assert.Equal(Main(), 12111131, "trail of outcomes and states");
}

[[fact]]
void AmbiguousLeavesTheStateUnchanged()
{
    instance Door door(false, 0);
    Assert.True(Step(door, Run, DoorSignal::Push(20)) == StepOutcome::Ambiguous, "two guards hold");
    Assert.Equal(State(door, Run), 0, "still Closed");
}
`
	if err := os.WriteFile(filepath.Join(dir, "door.concept_test"), []byte(facts), 0644); err != nil {
		t.Fatal(err)
	}
	manifest, err := DiscoverTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: filepath.Join(dir, ".test-results")})
		if err != nil {
			t.Fatal(err)
		}
		if run.Failed != 0 || run.Passed != 2 {
			t.Fatalf("verify=%v: %+v", verify, run.Results)
		}
	}
}
