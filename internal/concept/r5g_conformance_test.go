package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func r5gFixture(t *testing.T, kind, name string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r5g", "core", kind, name)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	return module, source
}

func TestEVT1R5gCorpusAndComplexAliases(t *testing.T) {
	valid, err := filepath.Glob(filepath.Join("..", "..", "language", "evt1-r5g", "core", "valid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := filepath.Glob(filepath.Join("..", "..", "language", "evt1-r5g", "core", "invalid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) != 23 || len(invalid) != 5 {
		t.Fatalf("R5g corpus count drift: valid=%d invalid=%d", len(valid), len(invalid))
	}

	readAsync := func(name string) *MIRAsyncFunction {
		module, source := r5gFixture(t, "valid", name)
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		var mir MIR
		base := strings.TrimSuffix(name, ".concept")
		if err := json.Unmarshal(outputs[base+".mir.json"], &mir); err != nil {
			t.Fatal(err)
		}
		for i := range mir.Functions {
			if mir.Functions[i].Name == "Work" {
				return mir.Functions[i].Async
			}
		}
		t.Fatalf("Work async MIR absent in %s", name)
		return nil
	}
	short := readAsync("async_match_multiple_awaits.concept")
	long := readAsync("asynchronous_structured_alias.concept")
	if short.ControlFlowStrategy != long.ControlFlowStrategy ||
		!reflect.DeepEqual(short.GeneratedStates, long.GeneratedStates) ||
		!reflect.DeepEqual(short.States, long.States) ||
		!reflect.DeepEqual(short.Edges, long.Edges) ||
		short.BranchCount != long.BranchCount || short.JoinCount != long.JoinCount || short.LoopCount != long.LoopCount {
		t.Fatal("complex async/asynchronous and await/awaitchronous spellings did not normalize to the same state graph")
	}

	unusedModule, err := Parse("unused_match_payload.concept", `profile Core;
enum Command { Read(int id), Idle }
async int Child() { return 1; }
async int Work(Command command) {
    match (command) {
        Command::Read(unused) => { int ignored = await Child(); }
        Command::Idle => { }
    }
    return 42;
}`)
	if err != nil {
		t.Fatal(err)
	}
	unusedOutputs, err := Generate(unusedModule, nil)
	if err != nil {
		t.Fatal(err)
	}
	var unusedMIR MIR
	if err := json.Unmarshal(unusedOutputs["unused_match_payload.mir.json"], &unusedMIR); err != nil {
		t.Fatal(err)
	}
	for _, fn := range unusedMIR.Functions {
		if fn.Name == "Work" {
			for _, field := range fn.Async.PersistentFields {
				if field.Name == "unused" {
					t.Fatal("unused match-arm payload leaked into the async frame")
				}
			}
		}
	}
}

func TestEVT1R5gStructuredNativeC11(t *testing.T) {
	for _, name := range []string{
		"async_two_awaits_one_branch", "async_two_awaits_else_branch", "async_nested_if_await",
		"async_match_await", "async_match_multiple_awaits", "async_match_payload_binding_await",
		"async_while_multiple_awaits", "async_loop_carried_value", "async_foreach_multiple_awaits",
		"async_nested_loop_match", "async_nested_foreach_await", "async_try_except_await",
		"async_question_inside_match", "async_branch_owned_lifetime", "async_branch_ref_lifetime",
		"async_branch_move_join", "async_manual_equivalence_complex",
		"async_evaluation_counts", "async_foreach_iterator_counts", "async_class_control_flow",
		"async_infer_branch", "async_tensor_loop", "asynchronous_structured_alias",
	} {
		t.Run(name, func(t *testing.T) {
			module, source := r5gFixture(t, "valid", name+".concept")
			outputs, err := Generate(module, source)
			if err != nil {
				t.Fatalf("generation: %v", err)
			}
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == 42 ? 0 : 1; }\n", name, name)
			runR1NativeHarness(t, outputs, name+"_harness.c", harness)
			if name == "async_foreach_iterator_counts" {
				generated := string(outputs[name+".generated.c"])
				if strings.Count(generated, "concept_async_foreach_iterator_counts_current(&(frame->foreach_iterator_0))") != 1 {
					t.Fatal("async foreach did not emit exactly one Current call at the successful-item header")
				}
			}
		})
	}
}

func TestEVT1R5gGraphPlanAndDeterminism(t *testing.T) {
	module, source := r5gFixture(t, "valid", "async_match_multiple_awaits.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["async_match_multiple_awaits.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	var async *MIRAsyncFunction
	for i := range mir.Functions {
		if mir.Functions[i].Name == "Work" {
			async = mir.Functions[i].Async
		}
	}
	if async == nil || async.ControlFlowStrategy != "StructuredStateGraph" || async.BranchCount != 1 || async.JoinCount < 1 || len(async.Edges) == 0 {
		t.Fatalf("structured async graph missing: %+v", async)
	}
	if err := evt1ValidateAsyncGraph(async, Span{}); err != nil {
		t.Fatal(err)
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	var asyncPlan *AsyncPlan
	for i := range plan.Functions {
		if plan.Functions[i].Function == "Work" {
			asyncPlan = plan.Functions[i].Async
		}
	}
	if asyncPlan == nil || asyncPlan.ControlFlowStrategy != "StructuredStateGraph" || asyncPlan.GeneratedStateCount != len(async.States) || asyncPlan.Scheduler != "None" || asyncPlan.SavedPC != "None" {
		t.Fatalf("structured async plan missing: %+v", asyncPlan)
	}
	for i := 0; i < 100; i++ {
		again, err := GeneratePlan(module, GenericC11Target())
		if err != nil || !bytes.Equal(planBytes, again) {
			t.Fatalf("plan drift at iteration %d: %v", i, err)
		}
	}
	malformed := *async
	malformed.Edges = append([]MIRAsyncEdge{}, async.Edges...)
	malformed.Edges[0].To = "absent"
	if code := diagnosticCode(evt1ValidateAsyncGraph(&malformed, Span{})); code != "ASYNC_GRAPH_INVALID" {
		t.Fatalf("malformed graph diagnostic = %s", code)
	}
	generated := string(outputs["async_match_multiple_awaits.generated.c"])
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "pthread", "executor", "event_loop", "setjmp(", "longjmp(", "__builtin_coro", "continuation_ptr", "saved_pc"} {
		if strings.Contains(generated, forbidden) {
			t.Fatalf("generated C contains forbidden marker %q", forbidden)
		}
	}
}

func TestEVT1R5gStaticDiagnostics(t *testing.T) {
	cases := map[string]string{
		"async_branch_uninitialized_join.concept": "CV4560",
		"async_branch_maybe_moved_join.concept":   "CV4503",
		"async_irreducible_control_flow.concept":  "ASYNC_CONTROL_FLOW_NOT_REDUCIBLE",
		"async_ref_escape_nested_branch.concept":  "ASYNC_PERSISTENT_REF_ESCAPE",
		"async_invalid_generated_join.concept":    "ASYNC_MATCH_NORMALIZATION_INVALID",
	}
	for name, want := range cases {
		path := filepath.Join("..", "..", "language", "evt1-r5g", "core", "invalid", name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(filepath.ToSlash(path), string(source))
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != want {
			t.Fatalf("%s diagnostic = %v, want %s", name, err, want)
		}
	}
	_, err := Parse("two_awaits_one_expression.concept", `profile Core;
async int Child() { return 21; }
async int Invalid() { return await Child() + await Child(); }`)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "ASYNC_CONTROL_FLOW_NOT_REDUCIBLE" {
		t.Fatalf("multi-await expression diagnostic = %v", err)
	}
}
