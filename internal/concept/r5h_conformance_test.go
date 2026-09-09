package concept

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var evt1R5hInvalidCases = map[string]string{
	"async_interface_sync_impl_mismatch.concept":              "CV4156",
	"sync_interface_async_impl_mismatch.concept":              "CV4156",
	"async_interface_wrong_return.concept":                    "CV4156",
	"async_interface_wrong_receiver_mutability.concept":       "CV4155",
	"async_interface_private_method_not_satisfy.concept":      "INTERFACE_PRIVATE_MEMBER_CANNOT_SATISFY",
	"async_interface_open_generic_not_dyn_compatible.concept": "INTERFACE_ASYNC_METHOD_NOT_DYN_COMPATIBLE",
	"dyn_async_receiver_escape.concept":                       "DYN_ASYNC_RECEIVER_LIFETIME_INVALID",
	"dyn_async_mutable_from_const.concept":                    "DYN_MUTABLE_FROM_CONST",
	"dyn_async_active_copy.concept":                           "ASYNC_COPY_INVALID",
	"dyn_async_witness_invalid.concept":                       "DYN_CONCRETE_TYPE_DOES_NOT_SATISFY",
}

func r5hFixture(t *testing.T, kind, name string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1-r5h", "core", kind, name)
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

func generateR5hFixture(t *testing.T, name string) (Module, Outputs) {
	t.Helper()
	module, source := r5hFixture(t, "valid", name)
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	return module, outputs
}

func TestEVT1R5hCorpusConformance(t *testing.T) {
	valid, err := filepath.Glob(filepath.Join("..", "..", "language", "evt1-r5h", "core", "valid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	invalid, err := filepath.Glob(filepath.Join("..", "..", "language", "evt1-r5h", "core", "invalid", "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	if len(valid) != 19 || len(invalid) != 10 {
		t.Fatalf("R5h corpus count drift: valid=%d invalid=%d", len(valid), len(invalid))
	}
	for _, path := range valid {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, path := range invalid {
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err == nil {
				_, err = Generate(module, source)
			}
			var diagnostic Diagnostic
			want := evt1R5hInvalidCases[filepath.Base(path)]
			if !errors.As(err, &diagnostic) || diagnostic.Code != want || diagnostic.SemanticCategory() == "" {
				t.Fatalf("diagnostic = %v, want %s", err, want)
			}
		})
	}
}

func TestEVT1R5hWitnessMIRPlanAndC(t *testing.T) {
	module, outputs := generateR5hFixture(t, "dyn_async_method_await.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["dyn_async_method_await.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	if len(mir.Witnesses) != 1 || len(mir.Witnesses[0].AsyncMethods) != 1 {
		t.Fatalf("missing async witness entry: %+v", mir.Witnesses)
	}
	entry := mir.Witnesses[0].AsyncMethods[0]
	if entry.Signature != "Async<int> Read(ref Device)" || entry.MachineIdentity != "Read#async#machine" || entry.EventualType.Name != "int" {
		t.Fatalf("async witness normalization drift: %+v", entry)
	}
	foundDynConstructor := false
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "dyn_call" && op.AsyncConstructor {
				foundDynConstructor = op.ReturnType == "Async<int>" && op.Evaluation == "ExactlyOnce" && op.OutcomeTransfer == "MoveOnce"
			}
		}
	}
	if !foundDynConstructor {
		t.Fatal("dyn call MIR did not retain ordinary Async<T> constructor semantics")
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	foundPlan := false
	for _, fn := range plan.Functions {
		for _, dispatch := range fn.Dispatch {
			if dispatch.AsyncConstructor {
				foundPlan = dispatch.Strategy == "WitnessIndirect" && dispatch.ReturnType == "Async<int>"
			}
		}
		if fn.Function == "Load" && (fn.Async == nil || fn.Async.ChildInvocation != "MachinePush" || fn.Async.Scheduler != "None" || fn.Async.SavedPC != "None") {
			t.Fatalf("dyn-await caller lost ordinary async plan: %+v", fn.Async)
		}
	}
	if !foundPlan {
		t.Fatal("plan omitted witness-indirect Async<T> constructor")
	}
	c := string(outputs["dyn_async_method_await.generated.c"])
	h := string(outputs["dyn_async_method_await.generated.h"])
	for _, required := range []string{"concept_async_operation (*Read)(void*", ".witness->Read(", "concept_async_operation cv_async_child_", "concept_async_adopt(async_operation"} {
		if !strings.Contains(c+h, required) {
			t.Fatalf("generated strict-C11 evidence missing %q", required)
		}
	}
	if strings.Count(c, ".witness->Read(") != 1 {
		t.Fatalf("dyn async constructor dispatch count = %d, want one", strings.Count(c, ".witness->Read("))
	}
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "pthread", "executor", "event_loop", "__builtin_coro", "std::coroutine", "promise", "future", "rtti", "per_object_vtable"} {
		if strings.Contains(strings.ToLower(c+h), forbidden) {
			t.Fatalf("generated artifacts contain forbidden runtime marker %q", forbidden)
		}
	}
}

func TestEVT1R5hAliasesNormalizeIdentically(t *testing.T) {
	shortModule, _ := r5hFixture(t, "valid", "dyn_async_method_await.concept")
	longModule, _ := r5hFixture(t, "valid", "long_alias_async_interface.concept")
	shortReq := shortModule.Concepts[0].Requirements[0].(*OperationRequirement)
	longReq := longModule.Concepts[0].Requirements[0].(*OperationRequirement)
	if !shortReq.ReturnType.Equal(longReq.ReturnType) || evt1Signature(shortReq.ReturnType, shortReq.Name, shortReq.Params) != evt1Signature(longReq.ReturnType, longReq.Name, longReq.Params) {
		t.Fatalf("async/asynchronous interface requirements differ: short=%+v long=%+v", shortReq, longReq)
	}
	if !shortModule.Functions[1].Async || !longModule.Functions[1].Async {
		t.Fatal("await aliases did not normalize to ordinary async functions")
	}
}

func TestEVT1R5hDeterministicWitnessAndPlan(t *testing.T) {
	module, source := r5hFixture(t, "valid", "mixed_sync_async_interface.concept")
	var wantMIR, wantPlan []byte
	for i := 0; i < 100; i++ {
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		plan, err := GeneratePlan(module, GenericC11Target())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			wantMIR = append([]byte{}, outputs["mixed_sync_async_interface.mir.json"]...)
			wantPlan = append([]byte{}, plan...)
			continue
		}
		if !bytes.Equal(wantMIR, outputs["mixed_sync_async_interface.mir.json"]) || !bytes.Equal(wantPlan, plan) {
			t.Fatalf("witness or plan changed at iteration %d", i)
		}
	}
}

func TestEVT1R5hRejectsMalformedAsyncWitnessAndPlan(t *testing.T) {
	broken := MIRInterfaceWitness{ID: "Reader__Device", Interface: "Reader", ConcreteType: "Device", Methods: []string{"Read"}, AsyncMethods: []MIRAsyncWitnessMethod{{Name: "Read", Signature: "Async<int> Read(ref Device)", ReturnType: evt1AsyncType(Type{Name: "int", Kind: TypeBuiltin}, "Read", Span{}), EventualType: Type{Name: "bool", Kind: TypeBuiltin}, MachineIdentity: "Read#async#machine"}}, NoAllocation: true}
	if code := diagnosticCode(evt1ValidateMIR(MIR{Witnesses: []MIRInterfaceWitness{broken}})); code != "DYN_ASYNC_WITNESS_INVALID" {
		t.Fatalf("malformed witness diagnostic = %s", code)
	}
	module, outputs := generateR5hFixture(t, "dyn_async_method_await.concept")
	var mir MIR
	if err := json.Unmarshal(outputs["dyn_async_method_await.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	plan, err := PlanModule(&mir, &facts, GenericC11Target(), coreProfileDefinition, ConservativeCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	_ = module
	for i := range plan.Functions {
		for j := range plan.Functions[i].Dispatch {
			if plan.Functions[i].Dispatch[j].AsyncConstructor {
				plan.Functions[i].Dispatch[j].ReturnType = "int"
			}
		}
	}
	plan.PlanID = loweringPlanIdentity(plan)
	if code := diagnosticCode(ValidateLoweringPlan(&mir, &facts, plan)); code != "PLAN_ASYNC_INVALID" {
		t.Fatalf("malformed async dispatch plan diagnostic = %s", code)
	}
}

func TestEVT1R5hNativeC11(t *testing.T) {
	for _, name := range []string{
		"async_interface_struct_satisfies",
		"async_interface_class_satisfies",
		"async_interface_static_template",
		"sync_start_async_interface",
		"dyn_async_method_construct",
		"dyn_async_method_await",
		"dyn_async_method_question",
		"dyn_async_manual_step",
		"dyn_async_void",
		"mixed_sync_async_interface",
		"composed_async_interface",
		"async_interface_semantic_fact",
		"dyn_async_scoped_receiver",
		"dyn_async_const_receiver",
		"dyn_async_inference",
		"dyn_async_span_state",
		"dyn_async_tensor_state",
		"long_alias_async_interface",
		"static_dyn_async_equivalence",
	} {
		t.Run(name, func(t *testing.T) {
			_, outputs := generateR5hFixture(t, name+".concept")
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == 42 ? 0 : 1; }\n", name, name)
			runR1NativeHarness(t, outputs, "r5h_"+name+"_harness.c", harness)
		})
	}
}
