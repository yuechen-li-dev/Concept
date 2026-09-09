package concept

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var callableCaptureInvalidCases = map[string]string{
	"callback_implicit_local_capture.concept":  "CALLABLE_CAPTURE_REQUIRED",
	"callback_implicit_self_capture.concept":   "CALLABLE_CAPTURE_REQUIRED",
	"callback_duplicate_capture.concept":       "CALLABLE_CAPTURE_DUPLICATE",
	"callback_copy_noncopyable.concept":        "CALLABLE_CAPTURE_COPY_NONCOPYABLE",
	"callback_move_moved_value.concept":        "CV4502",
	"callback_mutate_ref_const.concept":        "CV4513",
	"callback_ref_escape.concept":              "CALLABLE_CAPTURE_LIFETIME_INVALID",
	"callback_scoped_escape.concept":           "CALLABLE_CAPTURE_LIFETIME_INVALID",
	"callback_erased_local_escape.concept":     "CALLABLE_CAPTURE_LIFETIME_INVALID",
	"callback_erased_owning_temporary.concept": "CALLBACK_OWNING_TEMPORARY_REJECTED",
	"callback_const_mutating_invoke.concept":   "CALLABLE_CONST_MUTATING_INVOKE",
	"callback_consumed_capture_reuse.concept":  "CV4502",
}

func callableCaptureFixture(t *testing.T, kind, name string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "callable", "capture", kind, name)
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

func generateCallableCaptureOutputs(t *testing.T, name string) (Module, Outputs) {
	t.Helper()
	module, source := callableCaptureFixture(t, "valid", name)
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	return module, outputs
}

func TestCallableCaptureCorpusConformance(t *testing.T) {
	valid, _ := filepath.Glob(filepath.Join("..", "..", "language", "evt1", "callable", "capture", "valid", "*.concept"))
	invalid, _ := filepath.Glob(filepath.Join("..", "..", "language", "evt1", "callable", "capture", "invalid", "*.concept"))
	if len(valid) != 20 || len(invalid) != 12 {
		t.Fatalf("R5i corpus count drift: valid=%d invalid=%d", len(valid), len(invalid))
	}
	for _, path := range valid {
		t.Run("valid/"+filepath.Base(path), func(t *testing.T) {
			source, _ := os.ReadFile(path)
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Generate(module, source); err != nil {
				t.Fatal(err)
			}
			if _, err := GeneratePlan(module, GenericC11Target()); err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, path := range invalid {
		t.Run("invalid/"+filepath.Base(path), func(t *testing.T) {
			source, _ := os.ReadFile(path)
			_, err := Parse(filepath.ToSlash(path), string(source))
			if got, want := diagnosticCode(err), callableCaptureInvalidCases[filepath.Base(path)]; got != want {
				t.Fatalf("diagnostic=%s want=%s: %v", got, want, err)
			}
		})
	}
}

func TestCallableCaptureMIRPlannerAndRepresentation(t *testing.T) {
	module, outputs := generateCallableCaptureOutputs(t, "callback_evaluation_order.concept")
	var mir MIR
	for name, body := range outputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(body, &mir); err != nil {
				t.Fatal(err)
			}
		}
	}
	var mainMIR MIRFunction
	for _, fn := range mir.Functions {
		if fn.Name == "Main" {
			mainMIR = fn
		}
	}
	if len(mainMIR.Callables) != 1 {
		t.Fatalf("callable MIR missing: %+v", mir.Functions)
	}
	callable := mainMIR.Callables[0]
	if callable.Identity != "Main#callback0" || callable.Environment.Identity != "Main#callback0#environment" || len(callable.Environment.Fields) != 3 || !strings.HasPrefix(callable.Environment.Fields[0].Source, "Capture(") || !strings.HasPrefix(callable.Environment.Fields[1].Source, "Capture(") || callable.Environment.Fields[1].EvaluationOrder != 1 || !callable.NoAllocation {
		t.Fatalf("callable MIR=%+v", callable)
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	var mainPlan FunctionPlan
	for _, fn := range plan.Functions {
		if fn.Function == "Main" {
			mainPlan = fn
		}
	}
	if len(mainPlan.Callables) != 1 {
		t.Fatal("callable plan missing")
	}
	cp := mainPlan.Callables[0]
	if cp.Representation != "InlineEnvironment" || cp.Dispatch != "DirectCallable" || cp.Allocation != "None" || cp.EnvironmentAlignment < 1 {
		t.Fatalf("callable plan=%+v", cp)
	}
	_, erasedOutputs := generateCallableCaptureOutputs(t, "callback_erased_borrow.concept")
	var erasedMIR MIR
	for name, body := range erasedOutputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(body, &erasedMIR); err != nil {
				t.Fatal(err)
			}
			if len(erasedMIR.CallbackWitnesses) != 1 || erasedMIR.CallbackWitnesses[0].ConcreteCallable != "Main#callback0" || erasedMIR.CallbackWitnesses[0].Signature.Kind != TypeCallback || !erasedMIR.CallbackWitnesses[0].NoAllocation {
				t.Fatalf("callback witness=%+v", erasedMIR.CallbackWitnesses)
			}
		}
	}
	erasedMIR.CallbackWitnesses[0].Adapter = ""
	if code := diagnosticCode(evt1ValidateMIR(erasedMIR)); code != "CALLBACK_WITNESS_INVALID" {
		t.Fatalf("malformed callback witness diagnostic=%s", code)
	}
	brokenMIRBytes, _ := json.Marshal(mir)
	var brokenMIR MIR
	if err := json.Unmarshal(brokenMIRBytes, &brokenMIR); err != nil {
		t.Fatal(err)
	}
	for i := range brokenMIR.Functions {
		if brokenMIR.Functions[i].Name == "Main" {
			brokenMIR.Functions[i].Callables[0].Environment.Fields[1].EvaluationOrder = 0
		}
	}
	if code := diagnosticCode(evt1ValidateMIR(brokenMIR)); code != "CALLABLE_MIR_INVALID" {
		t.Fatalf("malformed MIR diagnostic=%s", code)
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	for i := range plan.Functions {
		if plan.Functions[i].Function == "Main" {
			plan.Functions[i].Callables[0].Allocation = "Heap"
		}
	}
	plan.PlanID = loweringPlanIdentity(&plan)
	if code := diagnosticCode(ValidateLoweringPlan(&mir, &facts, &plan)); code != "PLAN_CALLABLE_INVALID" {
		t.Fatalf("malformed plan diagnostic=%s", code)
	}
}

func TestCallableCaptureExactEnvironmentGeometry(t *testing.T) {
	cases := map[string]struct{ size, alignment int }{
		"callback_empty_capture.concept":    {1, 1},
		"callback_evaluation_order.concept": {16, 8},
		"callback_span_capture.concept":     {16, 8},
		"callback_tensor_capture.concept":   {16, 8},
		"callback_dyn_capture.concept":      {16, 8},
	}
	for name, want := range cases {
		_, outputs := generateCallableCaptureOutputs(t, name)
		for outputName, body := range outputs {
			if !strings.HasSuffix(outputName, ".mir.json") {
				continue
			}
			var mir MIR
			if err := json.Unmarshal(body, &mir); err != nil {
				t.Fatal(err)
			}
			var got MIRCaptureEnvironment
			for _, fn := range mir.Functions {
				if len(fn.Callables) != 0 {
					got = fn.Callables[0].Environment
					break
				}
			}
			if got.Identity == "" {
				t.Fatalf("%s has no callable environment", name)
			}
			if got.Size != want.size || got.Alignment != want.alignment {
				t.Fatalf("%s geometry=%d/%d want=%d/%d", name, got.Size, got.Alignment, want.size, want.alignment)
			}
		}
	}
}

func TestCallableCaptureMoveOnlyCallableTransferAndDropState(t *testing.T) {
	prefix := `profile Core;
struct Resource { int value; }
void Drop(owned Resource resource) { }
int Use(owned Resource resource) { return resource.value; }
int Main() {
  owned Resource resource = Resource{42};
  auto consume = callback() with (move resource) { return Use(move resource); };
`
	copySource := prefix + "  const auto duplicate = consume;\n  return 0;\n}\n"
	if _, err := Parse("callable-copy.concept", copySource); diagnosticCode(err) != "CV4501" {
		t.Fatalf("move-only callable copy diagnostic=%s: %v", diagnosticCode(err), err)
	}
	moveSource := prefix + "  auto transferred = move consume;\n  return transferred();\n}\n"
	module, err := Parse("callable-transfer.concept", moveSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(moveSource))
	if err != nil {
		t.Fatal(err)
	}
	var generated string
	for name, body := range outputs {
		if strings.HasSuffix(name, ".generated.c") || strings.HasSuffix(name, ".generated.h") {
			generated += string(body)
		}
	}
	for _, expected := range []string{"bool __live_resource;", ".__live_resource = true", "environment->__live_resource = false;", "if ((transferred).__live_resource)"} {
		if !strings.Contains(generated, expected) {
			t.Fatalf("generated C omits %q\n%s", expected, generated)
		}
	}
}

func TestCallableCaptureLetVarCompatibilityAliases(t *testing.T) {
	source := `profile Core;
int Main() {
  let readonly = callback(int value) { return value + 1; };
  var mutable = callback() with (count = 0) { count = count + 1; return count; };
  return readonly(4) + mutable();
}`
	module, err := Parse("callable-aliases.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if !module.Functions[0].Body.Statements[0].(*VarDecl).Const || module.Functions[0].Body.Statements[1].(*VarDecl).Const {
		t.Fatal("let/var compatibility aliases did not preserve const/mutable meaning")
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}

func TestCallableCaptureDeterminismAndForbiddenRuntime(t *testing.T) {
	module, source := callableCaptureFixture(t, "valid", "callback_async_interface_composition.concept")
	var firstMIR, firstPlan []byte
	for i := 0; i < 100; i++ {
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		var mir []byte
		for name, body := range outputs {
			if strings.HasSuffix(name, ".mir.json") {
				mir = body
			}
			if strings.HasSuffix(name, ".generated.c") {
				lower := strings.ToLower(string(body))
				for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "closure heap", "gc_register", "rtti", "task runtime", "scheduler", "executor", "hidden copy box"} {
					if strings.Contains(lower, forbidden) {
						t.Fatalf("generated C contains forbidden runtime marker %q", forbidden)
					}
				}
			}
		}
		plan, err := GeneratePlan(module, GenericC11Target())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstMIR, firstPlan = append([]byte{}, mir...), append([]byte{}, plan...)
			continue
		}
		if string(mir) != string(firstMIR) || string(plan) != string(firstPlan) {
			t.Fatalf("R5i artifacts changed at iteration %d", i)
		}
	}
}

func TestCallableCaptureNativeC11(t *testing.T) {
	cases := map[string]int{
		"callback_empty_capture": 5, "callback_copy_capture": 12, "callback_move_capture": 7,
		"callback_ref_capture": 6, "callback_ref_const_capture": 12, "callback_copy_independent_value": 92,
		"callback_mutable_environment": 2, "callback_consumed_owned_capture": 42, "callback_self_explicit_capture": 4,
		"callback_scoped_ref": 6, "callback_static_ref_return": 12, "callback_span_capture": 6,
		"callback_tensor_capture": 9, "callback_dyn_capture": 11, "callback_erased_borrow": 15,
		"callback_erased_argument": 7, "callback_erased_const": 10, "callback_evaluation_order": 12112,
		"callback_async_live_across_await": 12, "callback_async_interface_composition": 34,
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			_, outputs := generateCallableCaptureOutputs(t, name+".concept")
			for outputName, body := range outputs {
				if !strings.HasSuffix(outputName, ".generated.c") {
					continue
				}
				lower := strings.ToLower(string(body))
				for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "closure heap", "gc_register", "rtti", "task runtime", "scheduler", "executor", "hidden copy box"} {
					if strings.Contains(lower, forbidden) {
						t.Fatalf("generated C contains forbidden runtime marker %q", forbidden)
					}
				}
			}
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == %d ? 0 : 1; }\n", name, name, want)
			runFoundationNativeHarness(t, outputs, "callable_capture_"+name+"_harness.c", harness)
		})
	}
}
