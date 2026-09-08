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

var evt1R5dValid = []string{
	"yield_basic.concept", "yield_transient_local_recreated.concept", "yield_transient_owned_cleanup.concept", "yield_persistent_owned_survives.concept",
	"yield_same_state_reentry.concept", "yield_shared_state_persists.concept", "yield_machine_field_persists.concept",
	"foreach_array.concept", "foreach_ndarray_linear.concept", "foreach_span.concept", "foreach_readonly_span.concept", "foreach_ref_mutation.concept", "foreach_custom_iterator.concept", "foreach_source_evaluated_once.concept", "foreach_cleanup.concept", "foreach_inside_automata_state.concept", "persistent_iterator_manual_step.concept",
	"foreach_iterator_evaluation_counts.concept",
}

var evt1R5dInvalid = map[string]string{
	"yield_outside_state.concept":              "YIELD_OUTSIDE_STATE",
	"yield_top_level.concept":                  "YIELD_OUTSIDE_STATE",
	"yield_in_comptime.concept":                "YIELD_OUTSIDE_STATE",
	"foreach_noniterable.concept":              "FOREACH_SOURCE_NOT_ITERABLE",
	"foreach_item_type_mismatch.concept":       "FOREACH_ITEM_TYPE_MISMATCH",
	"foreach_mutable_ref_readonly.concept":     "FOREACH_MUTABLE_REF_FROM_READONLY",
	"foreach_noncopyable_value_item.concept":   "FOREACH_NONCOPYABLE_VALUE_ITEM",
	"foreach_iterator_lifetime_escape.concept": "CV4521",
}

func r5dFixture(t *testing.T, class, file string) Outputs {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r5d", "core", class, file)
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

func TestEVT1R5dConformance(t *testing.T) {
	for _, file := range evt1R5dValid {
		t.Run("valid/"+file, func(t *testing.T) { r5dFixture(t, "valid", file) })
	}
	for file, code := range evt1R5dInvalid {
		t.Run("invalid/"+file, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1-r5d", "core", "invalid", file)
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

func TestEVT1R5dNativeC11(t *testing.T) {
	cases := []struct {
		file, call string
		want       int
	}{
		{"yield_basic.concept", "concept_yield_basic_main", 2},
		{"yield_transient_local_recreated.concept", "concept_yield_transient_local_recreated_main", 6},
		{"yield_transient_owned_cleanup.concept", "concept_yield_transient_owned_cleanup_main", 0},
		{"yield_persistent_owned_survives.concept", "concept_yield_persistent_owned_survives_main", 8},
		{"yield_same_state_reentry.concept", "concept_yield_same_state_reentry_main", 20},
		{"yield_shared_state_persists.concept", "concept_yield_shared_state_persists_main", 11},
		{"yield_machine_field_persists.concept", "concept_yield_machine_field_persists_main", 3},
		{"foreach_array.concept", "concept_foreach_array_main", 10},
		{"foreach_ndarray_linear.concept", "concept_foreach_ndarray_linear_main", 123456},
		{"foreach_span.concept", "concept_foreach_span_main", 10},
		{"foreach_readonly_span.concept", "concept_foreach_readonly_span_main", 15},
		{"foreach_ref_mutation.concept", "concept_foreach_ref_mutation_main", 18},
		{"foreach_custom_iterator.concept", "concept_foreach_custom_iterator_main", 9},
		{"foreach_iterator_evaluation_counts.concept", "concept_foreach_iterator_evaluation_counts_main", 143},
		{"foreach_source_evaluated_once.concept", "concept_foreach_source_evaluated_once_main", 19},
		{"foreach_cleanup.concept", "concept_foreach_cleanup_main", 7},
		{"foreach_inside_automata_state.concept", "concept_foreach_inside_automata_state_main", 2},
		{"persistent_iterator_manual_step.concept", "concept_persistent_iterator_manual_step_main", 3},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			outputs := r5dFixture(t, "valid", tc.file)
			base := strings.TrimSuffix(tc.file, ".concept")
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return %s() == %d ? 0 : 1; }\n", base, tc.call, tc.want)
			runR1NativeHarness(t, outputs, "r5d_harness.c", harness)
			body := string(outputs[base+".generated.c"])
			for _, forbidden := range []string{"malloc", "calloc", "realloc", "coroutine", "scheduler", "program_counter"} {
				if strings.Contains(strings.ToLower(body), forbidden) {
					t.Fatalf("forbidden runtime mechanism %q appeared", forbidden)
				}
			}
		})
	}
}

func TestEVT1R5dMIRAndPlanner(t *testing.T) {
	yieldOutputs := r5dFixture(t, "valid", "yield_basic.concept")
	var yieldMIR MIR
	if err := json.Unmarshal(yieldOutputs["yield_basic.mir.json"], &yieldMIR); err != nil {
		t.Fatal(err)
	}
	y := yieldMIR.Automata[0].Machines[0].States[0].Yields[0]
	if !y.PreserveCurrentState || !y.PreservePersistentStorage || y.WriteResult || y.MarkComplete || y.CleanupEdge != "TransientBeforeStepReturn" {
		t.Fatalf("yield MIR drift: %+v", y)
	}

	foreachOutputs := r5dFixture(t, "valid", "foreach_array.concept")
	var eachMIR MIR
	if err := json.Unmarshal(foreachOutputs["foreach_array.mir.json"], &eachMIR); err != nil {
		t.Fatal(err)
	}
	each := eachMIR.Functions[0].Foreaches[0]
	if each.SourceEvaluation != "ExactlyOnce" || each.IteratorStrategy != "BuiltinInlineIterator" || !each.NoAllocation {
		t.Fatalf("foreach MIR drift: %+v", each)
	}
	malformedEach := eachMIR
	malformedEach.Functions[0].Foreaches[0].NoAllocation = false
	if err := evt1ValidateMIR(malformedEach); diagnosticCode(err) != "FOREACH_MIR_INVALID" {
		t.Fatalf("malformed foreach MIR = %v", err)
	}

	path := filepath.Join("..", "..", "language", "evt1-r5d", "core", "valid", "yield_basic.concept")
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
	yp := plan.Automata[0].Machines[0].Yields[0]
	if yp.Strategy != "ReenterStateFromStart" || yp.CoroutineFrame != "None" || yp.Scheduler != "None" || yp.PreserveTransientLocals {
		t.Fatalf("yield plan drift: %+v", yp)
	}

	malformed := yieldMIR
	malformed.Automata[0].Machines[0].States[0].Yields[0].WriteResult = true
	if err := evt1ValidateMIR(malformed); diagnosticCode(err) != "YIELD_MIR_INVALID" {
		t.Fatalf("malformed yield MIR = %v", err)
	}

	transientBody := string(r5dFixture(t, "valid", "yield_transient_owned_cleanup.concept")["yield_transient_owned_cleanup.generated.c"])
	dropAt := strings.Index(transientBody, "concept_yield_transient_owned_cleanup_drop(temporary);")
	yieldAt := strings.Index(transientBody, "CONCEPT_STEP_YIELDED")
	if dropAt < 0 || yieldAt < 0 || dropAt > yieldAt {
		t.Fatalf("transient Drop must precede yield return")
	}
	persistentBody := string(r5dFixture(t, "valid", "yield_persistent_owned_survives.concept")["yield_persistent_owned_survives.generated.c"])
	stepStart := strings.Index(persistentBody, "static void concept_owner_step_run")
	stepEnd := strings.Index(persistentBody[stepStart:], "static void concept_owner_init") + stepStart
	if strings.Contains(persistentBody[stepStart:stepEnd], "concept_yield_persistent_owned_survives_drop") {
		t.Fatalf("persistent owner dropped during yield")
	}
	arrayBody := string(foreachOutputs["foreach_array.generated.c"])
	if !strings.Contains(arrayBody, "backing storage is not copied") || strings.Contains(arrayBody, "malloc") {
		t.Fatalf("array foreach allocation/copy law drift")
	}
}
