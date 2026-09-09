package concept

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var plannerValidFixtures = []string{
	"plan_scalar_function.concept",
	"plan_struct_copy.concept",
	"plan_owned_cleanup.concept",
	"plan_result_question.concept",
	"plan_try_except.concept",
	"plan_span_index.concept",
	"plan_subspan.concept",
	"plan_layout_projection.concept",
	"plan_stream_projection.concept",
	"plan_dyn_dispatch.concept",
	"plan_tensor_elementwise.concept",
	"plan_tensor_contract.concept",
	"plan_tensor_runtime_shape.concept",
}

func plannerModule(t *testing.T, file string) (Module, []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "language", "evt1", "planner", "valid", file)
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

func semanticPlan(t *testing.T, file string, target TargetCapabilities) (MIR, SemanticFactSet, *LoweringPlan) {
	t.Helper()
	module, _ := plannerModule(t, file)
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	if err := evt1NormalizeModuleStorageTypes(&module, env); err != nil {
		t.Fatal(err)
	}
	mir := buildMIR(module, env)
	evt1QualifyMIRFacts(&mir)
	facts := NewSemanticFactSet(mir.SemanticFacts)
	plan, err := PlanModule(&mir, &facts, target, *env.profile, ConservativeCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return mir, facts, plan
}

func TestPlannerConformanceAndDeterministicArtifact(t *testing.T) {
	for _, file := range plannerValidFixtures {
		t.Run(file, func(t *testing.T) {
			module, source := plannerModule(t, file)
			first, err := GeneratePlan(module, GenericC11Target())
			if err != nil {
				t.Fatal(err)
			}
			second, err := GeneratePlan(module, GenericC11Target())
			if err != nil {
				t.Fatal(err)
			}
			if string(first) != string(second) {
				t.Fatal("plan artifact is nondeterministic")
			}
			var plan LoweringPlan
			if err := json.Unmarshal(first, &plan); err != nil {
				t.Fatal(err)
			}
			if plan.Schema != PlanSchema || plan.PlanID == "" || plan.Compiler != CompilerID || len(plan.Functions) == 0 {
				t.Fatalf("incomplete plan header: %+v", plan)
			}
			if _, err := Generate(module, source); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPlannerTensorFactsEligibilityAndFallback(t *testing.T) {
	_, _, fixed := semanticPlan(t, "plan_tensor_contract.concept", GenericC11Target())
	tp := fixed.Functions[0].Tensors[0]
	if tp.Strategy != "DirectLoopNest" || tp.AliasPolicy != "ProvenDisjoint" || tp.Vectorization.Outcome != DecisionEligible || tp.Vectorization.Selected || tp.Vectorization.Contiguous != FactProven || tp.Vectorization.DisjointInputs != FactProven || tp.Vectorization.FixedShape != FactProven {
		t.Fatalf("fixed tensor plan = %+v", tp)
	}
	if len(tp.Alignment) != 3 {
		t.Fatalf("tensor alignments = %+v", tp.Alignment)
	}
	for region, alignment := range tp.Alignment {
		if alignment != 64 {
			t.Fatalf("alignment[%s] = %d", region, alignment)
		}
	}

	_, _, runtime := semanticPlan(t, "plan_tensor_runtime_shape.concept", GenericC11Target())
	rp := runtime.Functions[0].Tensors[0]
	if rp.ShapeCompatibility != "RuntimeGuardRequired" || rp.BoundsPolicy != "HoistedRuntime" || len(rp.RuntimeGuards) == 0 || rp.Vectorization.Outcome != DecisionConservativeFallback || rp.Vectorization.Selected {
		t.Fatalf("runtime tensor plan = %+v", rp)
	}
}

func TestPlannerBoundsCleanupDispatchLayoutAndFailurePlans(t *testing.T) {
	_, _, span := semanticPlan(t, "plan_span_index.concept", GenericC11Target())
	if len(span.Functions[0].Bounds) != 1 || span.Functions[0].Bounds[0].Strategy != "PerAccessRuntime" {
		t.Fatalf("span bounds = %+v", span.Functions[0].Bounds)
	}
	_, _, subspan := semanticPlan(t, "plan_subspan.concept", GenericC11Target())
	if len(subspan.Functions[0].Bounds) == 0 || subspan.Functions[0].Bounds[0].Strategy != "HoistedRuntime" {
		t.Fatalf("subspan bounds = %+v", subspan.Functions[0].Bounds)
	}
	_, _, cleanup := semanticPlan(t, "plan_owned_cleanup.concept", GenericC11Target())
	cp := cleanup.Function("Cleanup").Cleanup
	if cp.Strategy != "ReverseDeclarationOrder" || len(cp.Drops) != 2 || cp.Drops[0].Owner != "second" || cp.Drops[1].Owner != "first" {
		t.Fatalf("cleanup plan = %+v", cp)
	}
	if !strings.Contains(string(mustJSON(t, cleanup)), `"strategy":"DropOldBeforeReplaceThenInitializeNew"`) {
		t.Fatal("owned replacement did not receive an explicit drop-old plan")
	}
	moved := cleanup.Function("MoveOut").Cleanup
	if len(moved.Drops) != 0 || len(moved.Suppressed) != 1 || moved.Suppressed[0] != "value" {
		t.Fatalf("moved-owner cleanup plan = %+v", moved)
	}
	_, _, dyn := semanticPlan(t, "plan_dyn_dispatch.concept", GenericC11Target())
	joined, _ := json.Marshal(dyn)
	if !strings.Contains(string(joined), `"strategy":"ObjectWitnessPair"`) || !strings.Contains(string(joined), `"strategy":"WitnessIndirect"`) || !strings.Contains(string(joined), `"NoAllocation"`) {
		t.Fatalf("dyn plan omitted witness dispatch: %s", joined)
	}
	_, _, layout := semanticPlan(t, "plan_layout_projection.concept", GenericC11Target())
	if !strings.Contains(string(mustJSON(t, layout)), `"representation":"StaticOffsets"`) {
		t.Fatal("fixed layout did not select static offsets")
	}
	_, _, stream := semanticPlan(t, "plan_stream_projection.concept", GenericC11Target())
	if !strings.Contains(string(mustJSON(t, stream)), `"representation":"ErasedAliasLayoutDescriptor"`) {
		t.Fatal("stream did not retain zero-storage representation")
	}
	_, _, failure := semanticPlan(t, "plan_result_question.concept", GenericC11Target())
	if !strings.Contains(string(mustJSON(t, failure)), `"strategy":"BranchAndEarlyReturn"`) {
		t.Fatal("Result ? did not receive a failure plan")
	}
}

func TestPlannerSameMIRPlansForThreeTargets(t *testing.T) {
	var identities []string
	var mirIdentities []string
	for _, target := range []TargetCapabilities{GenericC11Target(), X86_64GenericTarget(), AArch64GenericTarget()} {
		_, _, plan := semanticPlan(t, "plan_tensor_contract.concept", target)
		identities = append(identities, plan.PlanID)
		mirIdentities = append(mirIdentities, plan.MIRIdentity)
	}
	if mirIdentities[0] != mirIdentities[1] || mirIdentities[1] != mirIdentities[2] {
		t.Fatalf("planning mutated MIR: %v", mirIdentities)
	}
	if identities[0] == identities[1] || identities[1] == identities[2] || identities[0] == identities[2] {
		t.Fatalf("target-specific plans share identity: %v", identities)
	}
}

func TestPlannerMalformedPlansAreRejected(t *testing.T) {
	type mutation struct {
		name  string
		file  string
		code  string
		apply func(*LoweringPlan)
	}
	cases := []mutation{
		{"missing runtime guard", "plan_span_index.concept", "PLAN_RUNTIME_GUARD_MISSING", func(p *LoweringPlan) {
			for i := range p.Functions[0].Decisions {
				if p.Functions[0].Decisions[i].Operation == "span_index" {
					p.Functions[0].Decisions[i].RuntimeGuards = nil
				}
			}
		}},
		{"unsupported strategy", "plan_scalar_function.concept", "PLAN_TARGET_UNSUPPORTED", func(p *LoweringPlan) { p.Functions[0].Decisions[0].Strategy = "VectorSIMD" }},
		{"invalid tensor", "plan_tensor_contract.concept", "PLAN_TENSOR_INVALID", func(p *LoweringPlan) { p.Functions[0].Tensors[0].Strategy = "Unknown" }},
		{"invalid witness", "plan_dyn_dispatch.concept", "PLAN_WITNESS_INVALID", func(p *LoweringPlan) {
			for f := range p.Functions {
				for i := range p.Functions[f].Dispatch {
					if p.Functions[f].Dispatch[i].Operation == "dyn_make" {
						p.Functions[f].Dispatch[i].Witness = "missing"
					}
				}
			}
		}},
		{"incomplete cleanup", "plan_owned_cleanup.concept", "PLAN_CLEANUP_INCOMPLETE", func(p *LoweringPlan) { fp := p.Function("Cleanup"); fp.Cleanup.Drops = fp.Cleanup.Drops[:1] }},
		{"absent consumed fact", "plan_tensor_contract.concept", "PLAN_REQUIRED_FACT_MISSING", func(p *LoweringPlan) { p.Functions[0].Decisions[0].Evidence.FactIDs = []string{"fact-absent"} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mir, facts, plan := semanticPlan(t, tc.file, GenericC11Target())
			tc.apply(plan)
			plan.PlanID = loweringPlanIdentity(plan)
			var diagnostic Diagnostic
			if err := ValidateLoweringPlan(&mir, &facts, plan); !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
				t.Fatalf("diagnostic = %v, want %s", err, tc.code)
			}
		})
	}
	t.Run("stale MIR", func(t *testing.T) {
		mir, facts, plan := semanticPlan(t, "plan_scalar_function.concept", GenericC11Target())
		plan.MIRIdentity = "stale"
		var diagnostic Diagnostic
		if err := ValidateLoweringPlan(&mir, &facts, plan); !errors.As(err, &diagnostic) || diagnostic.Code != "PLAN_ARTIFACT_INVALID" {
			t.Fatalf("diagnostic = %v", err)
		}
	})
	t.Run("nondeterministic identity", func(t *testing.T) {
		mir, facts, plan := semanticPlan(t, "plan_scalar_function.concept", GenericC11Target())
		plan.PlanID = "random"
		var diagnostic Diagnostic
		if err := ValidateLoweringPlan(&mir, &facts, plan); !errors.As(err, &diagnostic) || diagnostic.Code != "PLAN_ARTIFACT_INVALID" {
			t.Fatalf("diagnostic = %v", err)
		}
	})
	t.Run("consumed fact not proven", func(t *testing.T) {
		mir, facts, plan := semanticPlan(t, "plan_span_index.concept", GenericC11Target())
		var factID string
		for _, decision := range plan.Functions[0].Decisions {
			if decision.Operation == "span_index" && len(decision.Evidence.FactIDs) > 0 {
				factID = decision.Evidence.FactIDs[0]
			}
		}
		for i := range facts.facts {
			if facts.facts[i].ID == factID {
				facts.facts[i].Outcome = FactDisproven
			}
		}
		var diagnostic Diagnostic
		if err := ValidateLoweringPlan(&mir, &facts, plan); !errors.As(err, &diagnostic) || diagnostic.Code != "PLAN_REQUIRED_FACT_DISPROVEN" {
			t.Fatalf("diagnostic = %v", err)
		}
	})
}

func TestPlannerPlannerBackedC11Execution(t *testing.T) {
	module, source := plannerModule(t, "plan_tensor_elementwise.concept")
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	harness := "#include \"plan_tensor_elementwise.generated.h\"\nint main(void) { return concept_plan_tensor_elementwise_tensor_elementwise() == 12 ? 0 : 1; }\n"
	runFoundationNativeHarness(t, outputs, "planner_harness.c", harness)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}
