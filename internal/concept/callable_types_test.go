package concept

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var callableTypesInvalidCases = map[string]string{
	"callable_assign_different_literal_type.concept": "CV4106",
	"callable_auto_field_existential.concept":        "CALLABLE_AUTO_FIELD_EXISTENTIAL",
	"callable_erased_to_concrete.concept":            "CV4106",
	"callable_field_ref_escape.concept":              "CALLABLE_FIELD_REF_ESCAPE",
	"callable_machine_field_short_ref.concept":       "CALLABLE_MACHINE_FIELD_SHORT_REF",
	"callable_const_mutating_parameter.concept":      "CALLABLE_CONST_MUTATING_INVOKE",
	"callable_consumed_parameter_reuse.concept":      "CV4502",
	"callable_type_query_invalid.concept":            "CALLABLE_TYPE_QUERY_INVALID",
}

func callableTypesFixture(t *testing.T, kind, name string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "callable", "types", kind, name)
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

func TestCallableTypesCorpusConformance(t *testing.T) {
	valid, _ := filepath.Glob(filepath.Join("..", "..", "language", "evt1", "callable", "types", "valid", "*.concept"))
	invalid, _ := filepath.Glob(filepath.Join("..", "..", "language", "evt1", "callable", "types", "invalid", "*.concept"))
	if len(valid) != 18 || len(invalid) != 8 {
		t.Fatalf("R5j corpus count drift: valid=%d invalid=%d", len(valid), len(invalid))
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
			if got, want := diagnosticCode(err), callableTypesInvalidCases[filepath.Base(path)]; got != want {
				t.Fatalf("diagnostic=%s want=%s: %v", got, want, err)
			}
		})
	}
}

func TestCallableTypesCanonicalUsingAndTypeCompatibility(t *testing.T) {
	source := `profile Core; auto Make() { return callback() { return 1; }; } using Canonical = typeof(Make()); type Compatible = typeof(Make()); int Main() { Canonical a = Make(); Compatible b = Make(); return a() + b(); }`
	module, err := Parse("aliases.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if module.TypeAliases[0].Spelling != "using" || module.TypeAliases[1].Spelling != "type" {
		t.Fatalf("alias spellings=%+v", module.TypeAliases)
	}
	if !module.TypeAliases[0].ResolvedType.Equal(module.TypeAliases[1].ResolvedType) {
		t.Fatal("transparent aliases lost exact identity")
	}
}

func TestCallableTypesMIRPlanAndMachinePersistence(t *testing.T) {
	module, source := callableTypesFixture(t, "valid", "callable_machine_field_ref_capture.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	for name, body := range outputs {
		if strings.HasSuffix(name, ".mir.json") {
			if err := json.Unmarshal(body, &mir); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(mir.TypeAliases) != 1 {
		t.Fatalf("aliases=%+v", mir.TypeAliases)
	}
	a := mir.TypeAliases[0]
	if a.Name != "Accumulator" || !a.Unevaluated || a.EnvironmentSize != 8 || a.EnvironmentAlignment != 8 || a.Storage != "Inline" || a.Dispatch != "DirectCallable" || !a.NoAllocation {
		t.Fatalf("alias MIR=%+v", a)
	}
	brokenMIR := mir
	brokenMIR.TypeAliases = append([]MIRTypeAlias{}, mir.TypeAliases...)
	brokenMIR.TypeAliases[0].Unevaluated = false
	if code := diagnosticCode(evt1ValidateMIR(brokenMIR)); code != "CALLABLE_TYPE_MIR_INVALID" {
		t.Fatalf("malformed alias MIR diagnostic=%s", code)
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	var plan LoweringPlan
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	field := plan.Automata[0].Machines[0].Fields[0]
	if field.Classification != "MachinePersistent" || field.CallableIdentity != a.CallableIdentity || field.EnvironmentSize != 8 || field.Dispatch != "DirectCallable" || field.Allocation != "None" || field.Reconstruction != "ConstructOncePersistAcrossSteps" {
		t.Fatalf("machine callable plan=%+v", field)
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	plan.ConcreteCallableTypes[0].Allocation = "Heap"
	plan.PlanID = loweringPlanIdentity(&plan)
	if code := diagnosticCode(ValidateLoweringPlan(&mir, &facts, &plan)); code != "PLAN_CALLABLE_TYPE_INVALID" {
		t.Fatalf("malformed callable type plan diagnostic=%s", code)
	}
}

func TestCallableTypesOwnedMachineCallableHasOneGuardedCleanupPath(t *testing.T) {
	module, source := callableTypesFixture(t, "valid", "callable_machine_field_owned_capture.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	var generated string
	for name, body := range outputs {
		if strings.HasSuffix(name, ".generated.c") {
			generated = string(body)
		}
	}
	for _, evidence := range []string{"if ((instance->main_frames[slot].read).__live_resource)", "concept_owner_drop_main_frame(instance, frame_index)", "instance->depth = (uint8_t)(instance->depth - 1u)"} {
		if !strings.Contains(generated, evidence) {
			t.Fatalf("generated cleanup omits %q", evidence)
		}
	}
}

func TestCallableTypesDeterminismAndForbiddenRuntime(t *testing.T) {
	module, source := callableTypesFixture(t, "valid", "callable_machine_field_ref_capture.concept")
	var firstMIR, firstPlan, firstC []byte
	for i := 0; i < 100; i++ {
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		var mir, c []byte
		for name, body := range outputs {
			if strings.HasSuffix(name, ".mir.json") {
				mir = body
			}
			if strings.HasSuffix(name, ".generated.c") {
				c = body
			}
		}
		lower := strings.ToLower(string(c))
		for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "std::function", "rtti", "type registry", "closure box", "vtable"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("generated C contains %q", forbidden)
			}
		}
		plan, err := GeneratePlan(module, GenericC11Target())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			firstMIR, firstPlan, firstC = append([]byte{}, mir...), append([]byte{}, plan...), append([]byte{}, c...)
			continue
		}
		if string(mir) != string(firstMIR) || string(plan) != string(firstPlan) || string(c) != string(firstC) {
			t.Fatalf("R5j artifacts changed at iteration %d", i)
		}
	}
}

func TestCallableTypesNativeC11(t *testing.T) {
	cases := map[string]int{
		"callable_typeof_local": 12, "callable_type_alias": 18, "callable_function_return": 9, "callable_explicit_return_alias": 10,
		"callable_struct_field": 12, "callable_class_field": 10, "callable_parameter_value": 12, "callable_parameter_ref_const": 10, "callable_parameter_ref_mut": 23,
		"callable_machine_field": 12, "callable_machine_field_ref_capture": 2, "callable_machine_field_owned_capture": 0, "callable_machine_field_across_yield": 42,
		"callable_option_result_storage": 0, "callable_async_return": 9, "callable_async_live_field": 12, "callable_dyn_capture_field": 11, "callable_span_capture_field": 6,
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			module, source := callableTypesFixture(t, "valid", name+".concept")
			outputs, err := Generate(module, source)
			if err != nil {
				t.Fatal(err)
			}
			harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == %d ? 0 : 1; }\n", name, name, want)
			runFoundationNativeHarness(t, outputs, "callable_types_"+name+"_harness.c", harness)
		})
	}
}
