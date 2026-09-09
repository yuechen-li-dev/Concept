package concept

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var automataStateValidFixtures = []string{
	"automata_basic.concept", "automata_with_state_value.concept", "automata_with_state_ref.concept", "automata_with_state_owned.concept",
	"automata_multiple_machines.concept", "automata_shared_state_between_machines.concept", "machine_persistent_field.concept",
	"machine_basic_transition.concept", "machine_transient_local.concept", "machine_owned_field_cleanup.concept",
	"automata_span_capture_valid.concept", "automata_dyn_capture_valid.concept",
}

var automataStateInvalidFixtures = map[string]string{
	"automata_implicit_capture_rejected.concept":   "AUTOMATA_CAPTURE_IMPLICIT_FORBIDDEN",
	"automata_ref_capture_escape.concept":          "AUTOMATA_CAPTURE_LIFETIME_INVALID",
	"automata_owned_capture_without_move.concept":  "CV4512",
	"automata_duplicate_machine.concept":           "AUTOMATA_DUPLICATE_MACHINE",
	"machine_duplicate_state.concept":              "MACHINE_DUPLICATE_STATE",
	"machine_unknown_transition.concept":           "MACHINE_UNKNOWN_STATE",
	"machine_sibling_private_state_access.concept": "AUTOMATA_CAPTURE_IMPLICIT_FORBIDDEN",
	"machine_transient_local_escape.concept":       "AUTOMATA_CAPTURE_IMPLICIT_FORBIDDEN",
}

func automataStateFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "automata", "state", class, file)
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

func TestAutomataStateConformance(t *testing.T) {
	for _, file := range automataStateValidFixtures {
		t.Run("valid/"+file, func(t *testing.T) { automataStateFixture(t, "valid", file) })
	}
	for file, code := range automataStateInvalidFixtures {
		t.Run("invalid/"+file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "automata", "state", "invalid", file)
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

func TestAutomataStateMIRAndPlanner(t *testing.T) {
	outputs := automataStateFixture(t, "valid", "automata_basic.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["automata_basic.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Automata) != 1 || mir.Automata[0].StateEnvironment == nil {
		t.Fatalf("missing automata state environment: %+v", mir.Automata)
	}
	a := mir.Automata[0]
	if a.StateEnvironment.Identity != "Counter#state" || a.StateEnvironment.Fields[0].Classification != "AutomataState" || a.Machines[0].Fields[0].Classification != "MachinePersistent" || a.Machines[0].States[0].Storage[0].Classification != "TransientLocal" {
		t.Fatalf("storage classification drift: %+v", a)
	}
	if !hasMIROperation(mir.Functions[0].Operations, "step_machine") {
		t.Fatalf("explicit machine step missing from MIR: %+v", mir.Functions[0].Operations)
	}
	modulePath := filepath.Join("..", "..", "language", "evt1", "automata", "state", "valid", "automata_basic.concept")
	source, _ := os.ReadFile(modulePath)
	module, _ := Parse(filepath.ToSlash(modulePath), string(source))
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Automata) != 1 || plan.Automata[0].EnvironmentStrategy != "InlineExplicitStruct" || plan.Automata[0].Scheduler != "None" || plan.Automata[0].Machines[0].DispatchStrategy != "Switch" {
		t.Fatalf("automata plan drift: %+v", plan.Automata)
	}
	if !hasPlanningDecision(plan.Functions[0].Decisions, "AutomataPlan", "ExplicitMachineSwitch") {
		t.Fatalf("explicit machine step planning missing: %+v", plan.Functions[0].Decisions)
	}

	var malformedMIR MIR
	if err := json.Unmarshal(outputs["automata_basic.mir.json"], &malformedMIR); err != nil {
		t.Fatal(err)
	}
	malformedMIR.Automata[0].StateEnvironment.Fields[0].Classification = "TransientLocal"
	if err := evt1ValidateMIR(malformedMIR); diagnosticCode(err) != "AUTOMATA_MIR_INVALID" {
		t.Fatalf("malformed automata MIR diagnostic = %v, want AUTOMATA_MIR_INVALID", err)
	}

	plan.Automata[0].Scheduler = "ImplicitRuntime"
	plan.PlanID = loweringPlanIdentity(&plan)
	facts := NewSemanticFactSet(mir.SemanticFacts)
	if err := ValidateLoweringPlan(&mir, &facts, &plan); diagnosticCode(err) != "PLAN_AUTOMATA_INVALID" {
		t.Fatalf("malformed automata plan diagnostic = %v, want PLAN_AUTOMATA_INVALID", err)
	}
}

func hasMIROperation(operations []MIROperation, kind string) bool {
	for _, operation := range operations {
		if operation.Kind == kind {
			return true
		}
	}
	return false
}

func hasPlanningDecision(decisions []PlanningDecision, category, strategy string) bool {
	for _, decision := range decisions {
		if decision.Category == category && decision.Strategy == strategy {
			return true
		}
	}
	return false
}

func diagnosticCode(err error) string {
	var diagnostic Diagnostic
	if errors.As(err, &diagnostic) {
		return diagnostic.Code
	}
	return ""
}

func TestAutomataStateNativeC11(t *testing.T) {
	cases := []struct {
		file, call string
		want       int
	}{
		{"automata_basic.concept", "concept_automata_basic_main", 11},
		{"automata_with_state_value.concept", "concept_automata_with_state_value_main", 3},
		{"automata_with_state_ref.concept", "concept_automata_with_state_ref_main", 42},
		{"automata_with_state_owned.concept", "concept_automata_with_state_owned_main", 0},
		{"automata_multiple_machines.concept", "concept_automata_multiple_machines_main", 0},
		{"automata_shared_state_between_machines.concept", "concept_automata_shared_state_between_machines_main", 15},
		{"machine_persistent_field.concept", "concept_machine_persistent_field_main", 2},
		{"machine_basic_transition.concept", "concept_machine_basic_transition_main", 1},
		{"machine_transient_local.concept", "concept_machine_transient_local_main", 4},
		{"machine_owned_field_cleanup.concept", "concept_machine_owned_field_cleanup_main", 0},
		{"automata_span_capture_valid.concept", "concept_automata_span_capture_valid_main", 3},
		{"automata_dyn_capture_valid.concept", "concept_automata_dyn_capture_valid_main", 2},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := automataStateFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return %s() == %d ? 0 : 1; }\n", base, tc.call, tc.want)
			runFoundationNativeHarness(t, outputs, "automata_state_harness.c", harness)
			body := string(outputs[base+".generated.c"])
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "coroutine", "scheduler"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
}

func TestAutomataStateCleanupShape(t *testing.T) {
	owned := string(automataStateFixture(t, "valid", "automata_with_state_owned.concept")["automata_with_state_owned.generated.c"])
	if strings.Count(owned, "concept_automata_with_state_owned_drop(instance->shared.resource);") != 1 {
		t.Fatalf("owned automata capture does not have exactly one drop site:\n%s", owned)
	}
	transient := string(automataStateFixture(t, "valid", "automata_with_state_owned.concept")["automata_with_state_owned.generated.c"])
	if !strings.Contains(transient, "concept_owner_drop(&owner);") {
		t.Fatal("automata lifetime cleanup missing")
	}
	shape := string(automataStateFixture(t, "valid", "automata_basic.concept")["automata_basic.generated.c"])
	structStart, stepStart := strings.Index(shape, "typedef struct concept_counter_instance"), strings.Index(shape, "static void concept_counter_step_run")
	if structStart < 0 || stepStart < 0 {
		t.Fatal("automata storage or step function missing")
	}
	if strings.Contains(shape[structStart:stepStart], "transient") || !strings.Contains(shape[stepStart:], "int transient") {
		t.Fatalf("transient local was lifted into persistent storage:\n%s", shape)
	}
	transientOwned := string(automataStateFixture(t, "valid", "machine_transient_local.concept")["machine_transient_local.generated.c"])
	if strings.Count(transientOwned, "concept_machine_transient_local_drop(temporary);") != 2 {
		t.Fatalf("transient owned local does not drop at step exit:\n%s", transientOwned)
	}
	transitionAt := strings.Index(transientOwned, "current_state = concept_worker_state_run_working")
	dropAt := strings.Index(transientOwned, "concept_machine_transient_local_drop(temporary);")
	if dropAt < 0 || transitionAt < 0 || dropAt > transitionAt {
		t.Fatalf("transition leaves state before transient cleanup:\n%s", transientOwned)
	}
}
