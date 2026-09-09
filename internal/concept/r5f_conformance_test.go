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

func r5fFixture(t *testing.T, kind, name string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r5f", "core", kind, name)
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

func TestEVT1R5fRuntimeStackOverflow(t *testing.T) {
	module, source := r5fFixture(t, "invalid", "async_stack_overflow.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
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
	harness := filepath.Join(dir, "overflow.c")
	if err := os.WriteFile(harness, []byte("#include \"async_stack_overflow.generated.h\"\nint main(void) { return concept_async_stack_overflow_main(); }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "overflow.exe")
	if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, filepath.Join(dir, "async_stack_overflow.generated.c"), harness, "-o", exe).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	out, err := exec.Command(exe).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "machine stack capacity exceeded") {
		t.Fatalf("overflow behavior drift: err=%v out=%s", err, out)
	}
}

func TestEVT1R5fRuntimeResultBeforeCompletion(t *testing.T) {
	module, source := r5fFixture(t, "invalid", "async_result_before_completion.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
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
	harness := filepath.Join(dir, "premature_result.c")
	if err := os.WriteFile(harness, []byte("#include \"async_result_before_completion.generated.h\"\nint main(void) { return concept_async_result_before_completion_main(); }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "premature_result.exe")
	if out, err := exec.Command(compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, filepath.Join(dir, "async_result_before_completion.generated.c"), harness, "-o", exe).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	out, err := exec.Command(exe).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "result requested before completion") {
		t.Fatalf("premature result behavior drift: err=%v out=%s", err, out)
	}
}

func TestEVT1R5fStaticDiagnostics(t *testing.T) {
	cases := map[string]string{
		"await_outside_async.concept":            "AWAIT_OUTSIDE_ASYNC",
		"await_nonasync_value.concept":           "AWAIT_REQUIRES_ASYNC_VALUE",
		"async_ref_lifetime_escape.concept":      "ASYNC_PERSISTENT_REF_ESCAPE",
		"async_value_neutral_completion.concept": "ASYNC_NEUTRAL_VALUE_MISMATCH",
		"async_copy_active_operation.concept":    "ASYNC_COPY_INVALID",
		"async_double_result_consume.concept":    "ASYNC_DOUBLE_RESULT_CONSUME",
	}
	for file, code := range cases {
		path := filepath.Join("..", "..", "language", "evt1-r5f", "core", "invalid", file)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(filepath.ToSlash(path), string(source))
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != code || diagnostic.SemanticCategory() == "" {
			t.Fatalf("%s diagnostic = %v, want %s", file, err, code)
		}
	}
}

func TestEVT1R5fBasicMIRPlanAndC(t *testing.T) {
	module, source := r5fFixture(t, "valid", "async_basic.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs[filepath.Base("async_basic.mir.json")], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Functions) < 2 || mir.Functions[1].Async == nil || len(mir.Functions[1].Async.AwaitPoints) != 2 {
		t.Fatalf("missing async MIR: %+v", mir.Functions)
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	if plan.Functions[1].Async == nil || plan.Functions[1].Async.Scheduler != "None" || plan.Functions[1].Async.SavedPC != "None" {
		t.Fatalf("invalid async plan: %+v", plan.Functions[1].Async)
	}
	if plan.Functions[1].Async.AsyncFrameSize < 1 || plan.Functions[1].Async.AsyncFrameAlignment < 1 || len(plan.Functions[1].Async.GeneratedStates) < 4 {
		t.Fatalf("async plan omits inspectable frame geometry or states: %+v", plan.Functions[1].Async)
	}
	c := string(outputs[filepath.Base("async_basic.generated.c")])
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "setjmp(", "longjmp(", "pthread", "executor", "event_loop"} {
		if strings.Contains(c, forbidden) {
			t.Fatalf("generated async C contains %q", forbidden)
		}
	}
	harness := fmt.Sprintf("#include \"async_basic.generated.h\"\nint main(void) { return concept_async_basic_main(); }\n")
	runR1NativeHarness(t, outputs, "r5f_harness.c", harness)
}

func TestEVT1R5fAliasesAndNestedNativeC11(t *testing.T) {
	shortModule, shortSource := r5fFixture(t, "valid", "async_basic.concept")
	longModule, longSource := r5fFixture(t, "valid", "asynchronous_alias.concept")
	shortOutputs, err := Generate(shortModule, shortSource)
	if err != nil {
		t.Fatal(err)
	}
	longOutputs, err := Generate(longModule, longSource)
	if err != nil {
		t.Fatal(err)
	}
	var shortMIR, longMIR MIR
	if err := json.Unmarshal(shortOutputs["async_basic.mir.json"], &shortMIR); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(longOutputs["asynchronous_alias.mir.json"], &longMIR); err != nil {
		t.Fatal(err)
	}
	shortAsync, longAsync := shortMIR.Functions[1].Async, longMIR.Functions[1].Async
	if len(shortAsync.AwaitPoints) != 2 || len(longAsync.AwaitPoints) != 2 || shortAsync.Identity != longAsync.Identity ||
		fmt.Sprint(shortAsync.GeneratedStates) != fmt.Sprint(longAsync.GeneratedStates) || fmt.Sprint(shortAsync.PersistentFields) != fmt.Sprint(longAsync.PersistentFields) {
		t.Fatalf("short/long aliases did not normalize: short=%+v long=%+v", shortMIR.Functions[1].Async, longMIR.Functions[1].Async)
	}
	for i := range shortAsync.AwaitPoints {
		a, b := shortAsync.AwaitPoints[i], longAsync.AwaitPoints[i]
		if a.Continuation != b.Continuation || a.Evaluation != b.Evaluation || a.ChildPush != b.ChildPush || a.OutcomeConsume != b.OutcomeConsume || fmt.Sprint(a.LiveAcross) != fmt.Sprint(b.LiveAcross) {
			t.Fatalf("await alias semantics differ at %d: short=%+v long=%+v", i, a, b)
		}
	}
	if !shortModule.Functions[0].Async || !longModule.Functions[0].Async || len(evt1AnalyzeAsync(shortModule.Functions[1]).Awaits) != len(evt1AnalyzeAsync(longModule.Functions[1]).Awaits) {
		t.Fatal("short/long aliases did not normalize to the same AST semantic kinds")
	}
	shortPlanBytes, err := GeneratePlan(shortModule, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	longPlanBytes, err := GeneratePlan(longModule, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var shortPlan, longPlan LoweringPlan
	if err := json.Unmarshal(shortPlanBytes, &shortPlan); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(longPlanBytes, &longPlan); err != nil {
		t.Fatal(err)
	}
	shortPlanAsync, _ := json.Marshal(shortPlan.Functions[1].Async)
	longPlanAsync, _ := json.Marshal(longPlan.Functions[1].Async)
	if string(shortPlanAsync) != string(longPlanAsync) {
		t.Fatalf("short/long aliases produced different AsyncPlan records:\nshort=%s\nlong=%s", shortPlanAsync, longPlanAsync)
	}
	for _, tc := range []struct {
		name    string
		module  Module
		source  []byte
		outputs Outputs
	}{
		{"async_basic", shortModule, shortSource, shortOutputs},
		{"asynchronous_alias", longModule, longSource, longOutputs},
	} {
		harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main(); }\n", tc.name, tc.name)
		runR1NativeHarness(t, tc.outputs, tc.name+"_harness.c", harness)
	}
	nested, nestedSource := r5fFixture(t, "valid", "async_nested_child.concept")
	nestedOutputs, err := Generate(nested, nestedSource)
	if err != nil {
		t.Fatal(err)
	}
	runR1NativeHarness(t, nestedOutputs, "nested_harness.c", "#include \"async_nested_child.generated.h\"\nint main(void) { return concept_async_nested_child_main(); }\n")
	for _, name := range []string{"async_named_operation", "async_result_question"} {
		module, source := r5fFixture(t, "valid", name+".concept")
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main(); }\n", name, name)
		runR1NativeHarness(t, outputs, name+"_harness.c", harness)
	}
	for _, name := range []string{"async_two_awaits", "async_live_local_across_await", "async_dead_local_not_hoisted", "async_owned_local_across_await", "async_ref_lifetime_valid", "async_class_method", "async_tensor_live_across_await", "async_move_value", "async_await_inside_foreach", "async_if_await", "async_while_await", "async_void"} {
		module, source := r5fFixture(t, "valid", name+".concept")
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == 42 ? 0 : 1; }\n", name, name)
		runR1NativeHarness(t, outputs, name+"_harness.c", harness)
	}
	for _, name := range []string{"async_await_evaluated_once", "async_manual_equivalence"} {
		module, source := r5fFixture(t, "valid", name+".concept")
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main(); }\n", name, name)
		runR1NativeHarness(t, outputs, name+"_harness.c", harness)
	}
}

func TestEVT1R5fCorpusLivenessCleanupAndMutationGuards(t *testing.T) {
	valid, err := filepath.Glob(filepath.Join("..", "..", "language", "evt1-r5f", "core", "valid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := filepath.Glob(filepath.Join("..", "..", "language", "evt1-r5f", "core", "invalid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) != 23 || len(invalid) != 8 {
		t.Fatalf("R5f corpus count drift: valid=%d invalid=%d", len(valid), len(invalid))
	}
	for _, path := range valid {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		module, err := Parse(filepath.ToSlash(path), string(source))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		generated := string(outputs[base+".generated.c"])
		for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "thrd_create", "CreateThread", "pthread", "executor", "event_loop", "setjmp(", "longjmp(", "__builtin_coro", "std::coroutine", "promise", "future"} {
			if strings.Contains(generated, forbidden) {
				t.Fatalf("%s generated C contains forbidden runtime marker %q", path, forbidden)
			}
		}
	}

	for file, function := range map[string]string{
		"async_live_local_across_await.concept": "LiveLocal",
		"async_dead_local_not_hoisted.concept":  "DeadLocal",
	} {
		module, _ := r5fFixture(t, "valid", file)
		env, err := analyzeModule(module)
		if err != nil {
			t.Fatal(err)
		}
		mir := buildMIR(module, env)
		var fields []MIRName
		for _, fn := range mir.Functions {
			if fn.Name == function {
				fields = fn.Async.PersistentFields
			}
		}
		names := fmt.Sprint(fields)
		if function == "LiveLocal" && !strings.Contains(names, "retained") {
			t.Fatalf("live local was not hoisted: %s", names)
		}
		if function == "DeadLocal" && strings.Contains(names, "dead") {
			t.Fatalf("dead local was unnecessarily hoisted: %s", names)
		}
	}

	ownedModule, ownedSource := r5fFixture(t, "valid", "async_owned_local_across_await.concept")
	ownedOutputs, err := Generate(ownedModule, ownedSource)
	if err != nil {
		t.Fatal(err)
	}
	ownedC := string(ownedOutputs["async_owned_local_across_await.generated.c"])
	if strings.Count(ownedC, "drop(frame->retained)") != 1 {
		t.Fatalf("persistent owned value must have one generated cleanup edge")
	}

	module, source := r5fFixture(t, "valid", "async_basic.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["async_basic.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	mir.Functions[1].Async.Scheduler = "InventedExecutor"
	if code := diagnosticCode(evt1ValidateMIR(mir)); code != "AWAIT_MIR_INVALID" {
		t.Fatalf("async MIR mutation diagnostic = %s", code)
	}
	if err := json.Unmarshal(outputs["async_basic.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	plan, err := PlanModule(&mir, &facts, GenericC11Target(), coreProfileDefinition, ConservativeCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	plan.Functions[1].Async.Scheduler = "InventedExecutor"
	plan.PlanID = loweringPlanIdentity(plan)
	if code := diagnosticCode(ValidateLoweringPlan(&mir, &facts, plan)); code != "PLAN_ASYNC_INVALID" {
		t.Fatalf("async plan mutation diagnostic = %s", code)
	}
}
