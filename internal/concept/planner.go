package concept

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const PlanSchema = "concept-evt1-plan.v1"

type TargetArchitecture string

const (
	ArchitectureGenericC11 TargetArchitecture = "GenericC11"
	ArchitectureX86_64     TargetArchitecture = "X86_64_Generic"
	ArchitectureAArch64    TargetArchitecture = "AArch64_Generic"
)

type TargetCapabilities struct {
	Architecture            TargetArchitecture `json:"architecture"`
	PointerBits             int                `json:"pointer_bits"`
	Endian                  string             `json:"endian"`
	NativeIntegerBits       int                `json:"native_integer_bits"`
	PreferredAlignment      int                `json:"preferred_alignment"`
	SupportsUnalignedAccess bool               `json:"supports_unaligned_access"`
	VectorRegisterBits      int                `json:"vector_register_bits,omitempty"`
	HasSIMD                 bool               `json:"has_simd"`
	HasFMA                  bool               `json:"has_fma"`
}

func GenericC11Target() TargetCapabilities {
	return TargetCapabilities{Architecture: ArchitectureGenericC11, PointerBits: 64, Endian: "little", NativeIntegerBits: 32, PreferredAlignment: 8, SupportsUnalignedAccess: false}
}

func X86_64GenericTarget() TargetCapabilities {
	return TargetCapabilities{Architecture: ArchitectureX86_64, PointerBits: 64, Endian: "little", NativeIntegerBits: 64, PreferredAlignment: 16, SupportsUnalignedAccess: true, VectorRegisterBits: 128, HasSIMD: true, HasFMA: false}
}

func AArch64GenericTarget() TargetCapabilities {
	return TargetCapabilities{Architecture: ArchitectureAArch64, PointerBits: 64, Endian: "little", NativeIntegerBits: 64, PreferredAlignment: 16, SupportsUnalignedAccess: true, VectorRegisterBits: 128, HasSIMD: true, HasFMA: true}
}

func TargetByName(name string) (TargetCapabilities, error) {
	switch strings.ToLower(strings.ReplaceAll(name, "-", "_")) {
	case "", "genericc11", "generic_c11", "c11":
		return GenericC11Target(), nil
	case "x86_64", "x86_64_generic", "amd64":
		return X86_64GenericTarget(), nil
	case "aarch64", "aarch64_generic", "arm64":
		return AArch64GenericTarget(), nil
	default:
		return TargetCapabilities{}, evt1Diagnostic("PLAN_TARGET_UNSUPPORTED", fmt.Sprintf("planning target %q is not supported", name), Span{})
	}
}

type CompilationPolicy struct {
	PreserveRuntimeGuards bool `json:"preserve_runtime_guards"`
	ForbidAllocation      bool `json:"forbid_allocation"`
	ForbidImplicitCopy    bool `json:"forbid_implicit_copy"`
}

func ConservativeCompilationPolicy() CompilationPolicy {
	return CompilationPolicy{PreserveRuntimeGuards: true, ForbidAllocation: true, ForbidImplicitCopy: true}
}

type DecisionCertainty string

const (
	DecisionRequired             DecisionCertainty = "Required"
	DecisionEligible             DecisionCertainty = "Eligible"
	DecisionSelected             DecisionCertainty = "Selected"
	DecisionRejected             DecisionCertainty = "Rejected"
	DecisionConservativeFallback DecisionCertainty = "ConservativeFallback"
)

type PlanningEvidence struct {
	FactIDs []string `json:"fact_ids,omitempty"`
	Claims  []string `json:"claims,omitempty"`
	Detail  string   `json:"detail,omitempty"`
}

type PlanningDecision struct {
	ID            string            `json:"id"`
	MIRID         string            `json:"mir_id,omitempty"`
	Category      string            `json:"category"`
	Operation     string            `json:"operation"`
	Strategy      string            `json:"strategy"`
	Certainty     DecisionCertainty `json:"certainty"`
	Evidence      PlanningEvidence  `json:"evidence"`
	RuntimeGuards []string          `json:"runtime_guards,omitempty"`
	SourceSpan    Span              `json:"source_span"`
}

type OptimizationEligibility struct {
	Contiguous          SemanticFactCertainty `json:"contiguous"`
	Bounded             SemanticFactCertainty `json:"bounded"`
	Alignment           SemanticFactCertainty `json:"alignment"`
	DisjointInputs      SemanticFactCertainty `json:"disjoint_inputs"`
	FixedShape          SemanticFactCertainty `json:"fixed_shape"`
	NoAllocation        SemanticFactCertainty `json:"no_allocation"`
	NoCopy              SemanticFactCertainty `json:"no_copy"`
	NoOwnershipTransfer SemanticFactCertainty `json:"no_ownership_transfer"`
	Outcome             DecisionCertainty     `json:"outcome"`
	Selected            bool                  `json:"selected"`
	Evidence            PlanningEvidence      `json:"evidence"`
}

type TensorPlan struct {
	MIRIndex           int                     `json:"mir_index"`
	Operation          string                  `json:"operation"`
	Strategy           string                  `json:"strategy"`
	TraversalOrder     []string                `json:"traversal_order,omitempty"`
	FreeIndices        []string                `json:"free_indices,omitempty"`
	ReductionIndices   []string                `json:"reduction_indices,omitempty"`
	DestinationRegion  string                  `json:"destination_region,omitempty"`
	InputRegions       []string                `json:"input_regions,omitempty"`
	AliasPolicy        string                  `json:"alias_policy"`
	Alignment          map[string]int          `json:"alignment,omitempty"`
	BoundsPolicy       string                  `json:"bounds_policy"`
	ShapeCompatibility string                  `json:"shape_compatibility"`
	RuntimeGuards      []string                `json:"runtime_guards,omitempty"`
	Vectorization      OptimizationEligibility `json:"vectorization"`
}

type BoundsPlan struct {
	MIRID     string            `json:"mir_id"`
	Operation string            `json:"operation"`
	Strategy  string            `json:"strategy"`
	Certainty DecisionCertainty `json:"certainty"`
	Evidence  PlanningEvidence  `json:"evidence"`
	Guard     string            `json:"guard,omitempty"`
}

type CleanupAction struct {
	Owner        string `json:"owner"`
	Type         string `json:"type"`
	DropFunction string `json:"drop_function"`
	Order        int    `json:"order"`
	Action       string `json:"action"`
}

type CleanupPlan struct {
	Strategy   string          `json:"strategy"`
	Drops      []CleanupAction `json:"drops,omitempty"`
	Suppressed []string        `json:"suppressed,omitempty"`
}

type DispatchPlan struct {
	MIRID     string            `json:"mir_id"`
	Operation string            `json:"operation"`
	Strategy  string            `json:"strategy"`
	Witness   string            `json:"witness,omitempty"`
	Entry     string            `json:"entry,omitempty"`
	Certainty DecisionCertainty `json:"certainty"`
	Evidence  PlanningEvidence  `json:"evidence"`
}

type AggregatePlan struct {
	Identity       string           `json:"identity"`
	Kind           string           `json:"kind"`
	Representation string           `json:"representation"`
	Evidence       PlanningEvidence `json:"evidence"`
}

type AutomataStoragePlan struct {
	Identity       string `json:"identity"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Classification string `json:"classification"`
	Strategy       string `json:"strategy"`
	HasDrop        bool   `json:"has_drop,omitempty"`
}

type MachinePlan struct {
	Identity             string                 `json:"identity"`
	CurrentStateSlot     string                 `json:"current_state_slot"`
	InitialState         string                 `json:"initial_state"`
	DispatchStrategy     string                 `json:"dispatch_strategy"`
	Fields               []AutomataStoragePlan  `json:"fields,omitempty"`
	StateIdentities      []string               `json:"state_identities"`
	Transitions          []string               `json:"transitions,omitempty"`
	TransitionMatches    []TransitionMatchPlan  `json:"transition_match_plans,omitempty"`
	TransitionDecisions  []TransitionDecidePlan `json:"transition_decide_plans,omitempty"`
	TransitionInferences []TransitionInferPlan  `json:"transition_infer_plans,omitempty"`
}

type InferencePlan struct {
	CandidateType   string `json:"candidate_type"`
	Normalization   string `json:"normalization"`
	MaxSubtraction  bool   `json:"max_subtraction"`
	CandidateCount  int    `json:"candidate_count"`
	Storage         string `json:"storage"`
	SIMDEligibility string `json:"simd_eligibility"`
	SelectedSIMD    bool   `json:"selected_simd"`
}

type TransitionInferPlan struct {
	State                string   `json:"state"`
	Strategy             string   `json:"strategy"`
	Policy               string   `json:"policy"`
	Normalization        string   `json:"normalization"`
	ScoreType            string   `json:"score_type"`
	GuardEvaluationOrder string   `json:"guard_evaluation_order"`
	ScoreEvaluationOrder string   `json:"score_evaluation_order"`
	Targets              []string `json:"targets"`
	CleanupEdge          string   `json:"cleanup_edge"`
}

type TransitionMatchPlan struct {
	State           string   `json:"state"`
	Strategy        string   `json:"strategy"`
	EvaluationOrder string   `json:"evaluation_order"`
	Targets         []string `json:"targets"`
	NoMatchPolicy   string   `json:"no_match_policy"`
	CleanupEdge     string   `json:"cleanup_edge"`
}

type TransitionDecidePlan struct {
	State                string   `json:"state"`
	Strategy             string   `json:"strategy"`
	GuardEvaluationOrder string   `json:"guard_evaluation_order"`
	ScoreEvaluationOrder string   `json:"score_evaluation_order"`
	Targets              []string `json:"targets"`
	ScoreType            string   `json:"score_type"`
	TiePolicy            string   `json:"tie_policy"`
	NoEnabledPolicy      string   `json:"no_enabled_policy"`
	CleanupEdge          string   `json:"cleanup_edge"`
	BranchlessEligible   bool     `json:"branchless_eligible"`
}

type AutomataPlan struct {
	Identity            string                `json:"identity"`
	StateEnvironment    string                `json:"state_environment"`
	EnvironmentStrategy string                `json:"environment_strategy"`
	StateFields         []AutomataStoragePlan `json:"state_fields,omitempty"`
	Machines            []MachinePlan         `json:"machines"`
	Scheduler           string                `json:"scheduler"`
	YieldStrategy       string                `json:"yield_strategy"`
}

type FunctionPlan struct {
	ID          string             `json:"id"`
	Function    string             `json:"function"`
	MIRIdentity string             `json:"mir_identity"`
	Target      TargetArchitecture `json:"target"`
	Decisions   []PlanningDecision `json:"decisions"`
	Tensors     []TensorPlan       `json:"tensor_plans,omitempty"`
	Inferences  []InferencePlan    `json:"inference_plans,omitempty"`
	Bounds      []BoundsPlan       `json:"bounds_plans,omitempty"`
	Cleanup     CleanupPlan        `json:"cleanup_plan"`
	Dispatch    []DispatchPlan     `json:"dispatch_plans,omitempty"`
	Failure     []PlanningDecision `json:"failure_plans,omitempty"`
	ControlFlow []PlanningDecision `json:"control_flow_plans,omitempty"`
}

type LoweringPlan struct {
	Schema      string             `json:"schema"`
	Compiler    string             `json:"compiler"`
	Module      string             `json:"module"`
	MIRIdentity string             `json:"mir_identity"`
	PlanID      string             `json:"plan_id"`
	Target      TargetCapabilities `json:"target"`
	Profile     string             `json:"profile"`
	Policy      CompilationPolicy  `json:"compilation_policy"`
	Functions   []FunctionPlan     `json:"functions"`
	Aggregates  []AggregatePlan    `json:"aggregate_plans,omitempty"`
	Automata    []AutomataPlan     `json:"automata_plans,omitempty"`
}

func PlanModule(module *MIR, facts *SemanticFactSet, target TargetCapabilities, profile ProfileDefinition, policy CompilationPolicy) (*LoweringPlan, error) {
	if module == nil || facts == nil {
		return nil, evt1Diagnostic("PLAN_ARTIFACT_INVALID", "planner requires explicit MIR and semantic facts", Span{})
	}
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	mirBytes, _ := json.Marshal(module)
	plan := &LoweringPlan{Schema: PlanSchema, Compiler: CompilerID, Module: module.Module, MIRIdentity: digest(mirBytes), Target: target, Profile: profile.Name, Policy: policy}
	for _, fn := range module.Functions {
		fp := planFunction(fn, *facts, target)
		plan.Functions = append(plan.Functions, fp)
	}
	plan.Aggregates = planAggregates(module)
	plan.Automata = planAutomata(module)
	plan.PlanID = loweringPlanIdentity(plan)
	if err := ValidateLoweringPlan(module, facts, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func loweringPlanIdentity(plan *LoweringPlan) string {
	identityInput, _ := json.Marshal(struct {
		MIR        string
		Target     TargetCapabilities
		Profile    string
		Policy     CompilationPolicy
		Functions  []FunctionPlan
		Aggregates []AggregatePlan
		Automata   []AutomataPlan
	}{plan.MIRIdentity, plan.Target, plan.Profile, plan.Policy, plan.Functions, plan.Aggregates, plan.Automata})
	return "plan-" + digest(identityInput)[:16]
}

// GeneratePlan exposes the deterministic planning artifact without invoking a
// backend. It shares the same validated MIR and semantic-fact construction as
// normal generation.
func GeneratePlan(module Module, target TargetCapabilities) ([]byte, error) {
	env, err := analyzeModule(module)
	if err != nil {
		return nil, err
	}
	if err := evt1NormalizeModuleStorageTypes(&module, env); err != nil {
		return nil, err
	}
	mir := buildMIR(module, env)
	evt1QualifyMIRFacts(&mir)
	if err := evt1ValidateMIR(mir); err != nil {
		return nil, err
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	plan, err := PlanModule(&mir, &facts, target, *env.profile, ConservativeCompilationPolicy())
	if err != nil {
		return nil, err
	}
	body, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

func planFunction(fn MIRFunction, facts SemanticFactSet, target TargetCapabilities) FunctionPlan {
	encoded, _ := json.Marshal(fn)
	fp := FunctionPlan{Function: fn.Name, MIRIdentity: digest(encoded), Target: target.Architecture, Cleanup: CleanupPlan{Strategy: "ReverseDeclarationOrder"}}
	for _, inference := range fn.Inferences {
		fp.Inferences = append(fp.Inferences, InferencePlan{CandidateType: inference.CandidateType.String(), Normalization: inference.Normalization, MaxSubtraction: true, CandidateCount: len(inference.Candidates), Storage: "InlineFixed", SIMDEligibility: "Deferred", SelectedSIMD: false})
	}
	for _, cleanup := range fn.Cleanups {
		if cleanup.State == "transferred" {
			fp.Cleanup.Suppressed = append(fp.Cleanup.Suppressed, cleanup.Owner)
			continue
		}
		fp.Cleanup.Drops = append(fp.Cleanup.Drops, CleanupAction{Owner: cleanup.Owner, Type: cleanup.Type, DropFunction: cleanup.DropFunction, Order: cleanup.Order, Action: "DropAtScopeExit"})
	}
	for _, op := range fn.Operations {
		d := planOperation(op, facts)
		if op.Kind == "assign" && cleanupOwner(fn.Cleanups, op.Detail) {
			d.Category, d.Strategy, d.Certainty = "CleanupPlan", "DropOldBeforeReplaceThenInitializeNew", DecisionRequired
			d.Evidence = PlanningEvidence{Claims: []string{"MIR cleanup obligation remains live after replacement"}}
			d.ID = "decision-" + digest([]byte(op.ID + "|" + d.Category + "|" + d.Strategy))[:16]
		}
		fp.Decisions = append(fp.Decisions, d)
		switch op.Kind {
		case "span_index", "array_index", "ndarray_index", "tensor_index", "span_subregion":
			fp.Bounds = append(fp.Bounds, BoundsPlan{MIRID: op.ID, Operation: op.Kind, Strategy: d.Strategy, Certainty: d.Certainty, Evidence: d.Evidence, Guard: firstGuard(d.RuntimeGuards)})
		case "dyn_make", "dyn_call", "dyn_field_get", "dyn_field_set", "call", "class_method_call":
			strategy := d.Strategy
			witness, entry := "", op.Detail
			if op.Kind == "dyn_make" {
				witness = strings.TrimSpace(strings.Split(op.Detail, "<-")[0])
			}
			fp.Dispatch = append(fp.Dispatch, DispatchPlan{MIRID: op.ID, Operation: op.Kind, Strategy: strategy, Witness: witness, Entry: entry, Certainty: d.Certainty, Evidence: d.Evidence})
		case "result_propagate", "option_propagate", "result_unroll", "option_unroll", "try_handler", "assert":
			fp.Failure = append(fp.Failure, d)
		case "while", "bounded_while", "if_stmt", "match_stmt":
			fp.ControlFlow = append(fp.ControlFlow, d)
		}
	}
	for i, tensor := range fn.TensorOperations {
		fp.Tensors = append(fp.Tensors, planTensor(i, tensor, facts))
	}
	fp.ID = "function-plan-" + digest([]byte(fn.Name + "|" + fp.MIRIdentity + "|" + string(target.Architecture)))[:16]
	return fp
}

func planOperation(op MIROperation, facts SemanticFactSet) PlanningDecision {
	d := PlanningDecision{MIRID: op.ID, Category: "CallPlan", Operation: op.Kind, Strategy: "Direct", Certainty: DecisionSelected, Evidence: PlanningEvidence{Detail: "ordinary MIR operation preserves semantic order"}, SourceSpan: op.SourceSpan}
	switch op.Kind {
	case "infer":
		d.Category, d.Strategy, d.Certainty = "InferencePlan", "ScalarStableSoftMax", DecisionSelected
		d.Evidence = PlanningEvidence{Claims: []string{"MaxSubtraction", "InlineFixed", "NoSIMD", "NoAllocation"}}
	case "transition_infer":
		d.Category, d.Strategy, d.Certainty = "InferenceTransitionPlan", "ScalarStableSoftMaxThenHardMax", DecisionSelected
		d.Evidence = PlanningEvidence{Claims: []string{"ExplicitPolicy", "NoRandomness", "TransientBeforeStateUpdate"}}
	case "span_index", "array_index", "ndarray_index", "tensor_index":
		d.Category, d.Strategy, d.Certainty = "BoundsPlan", "PerAccessRuntime", DecisionRequired
		d.RuntimeGuards = []string{op.BoundsCheck}
		if op.Kind == "array_index" || op.Kind == "ndarray_index" || op.Kind == "tensor_index" {
			d.RuntimeGuards = []string{"zero_le_index_lt_extent"}
		}
		if op.BoundsCheck == "statically_valid" {
			d.Strategy, d.Certainty, d.RuntimeGuards = "StaticEliminated", DecisionSelected, nil
		}
		d.Evidence = evidenceForSubject(facts, op.RegionID, FactBounded)
	case "span_subregion":
		d.Category, d.Strategy, d.Certainty = "BoundsPlan", "HoistedRuntime", DecisionRequired
		d.RuntimeGuards = []string{op.BoundsCheck}
		d.Evidence = evidenceForSubject(facts, op.RegionID, FactBounded)
	case "dyn_make":
		d.Category, d.Strategy = "DispatchPlan", "ObjectWitnessPair"
		d.Evidence = PlanningEvidence{Claims: []string{"NoAllocation", "NoCopy", "NoOwnershipTransfer"}, Detail: "MIR carries a deterministic witness identity"}
	case "dyn_call":
		d.Category, d.Strategy = "DispatchPlan", "WitnessIndirect"
		d.Evidence = PlanningEvidence{Claims: []string{"NoAllocation", "NoCopy", "NoOwnershipTransfer"}, Detail: "interface satisfaction was established before planning"}
	case "dyn_field_get", "dyn_field_set":
		d.Category, d.Strategy = "DispatchPlan", "WitnessFieldAccessor"
		d.Evidence = PlanningEvidence{Claims: []string{"NoAllocation", "NoCopy", "NoOwnershipTransfer"}}
	case "step_machine":
		d.Category, d.Strategy = "AutomataPlan", "ExplicitMachineSwitch"
		d.Evidence = PlanningEvidence{Claims: []string{"NoScheduler", "OneMachineOnly"}, Detail: "machine identity and current-state storage were fixed before planning"}
	case "state_machine":
		d.Category, d.Strategy = "AutomataPlan", "ReadCurrentStateTag"
		d.Evidence = PlanningEvidence{Claims: []string{"StableDeclarationOrder"}}
	case "result_propagate", "option_propagate":
		d.Category, d.Strategy, d.Certainty = "FailurePlan", "BranchAndEarlyReturn", DecisionRequired
	case "result_unroll", "option_unroll":
		d.Category, d.Strategy, d.Certainty = "FailurePlan", "BranchAndTerminalPanic", DecisionRequired
	case "try_handler":
		d.Category, d.Strategy, d.Certainty = "FailurePlan", "LocalHandlerBranch", DecisionRequired
	case "assert":
		d.Category, d.Strategy, d.Certainty = "FailurePlan", "ConditionalPanic", DecisionRequired
	case "while", "bounded_while":
		d.Category, d.Strategy = "ControlFlowPlan", "Loop"
	case "if_stmt":
		d.Category, d.Strategy = "ControlFlowPlan", "DirectBranch"
	case "match_stmt":
		d.Category, d.Strategy = "ControlFlowPlan", "MatchSwitch"
	case "layout_bind", "region_projection":
		d.Category, d.Strategy = "LayoutPlan", "StaticOffsets"
		d.Evidence = evidenceForSubject(facts, op.RegionID, FactAligned)
	case "stream_bind":
		d.Category, d.Strategy = "StreamPlan", "ErasedAliasLayoutDescriptor"
		d.Evidence = PlanningEvidence{Claims: []string{"NoAllocation", "NoCopy", "NoOwnershipTransfer"}, Detail: "stream is zero-storage MIR"}
	case "span_from_region", "span_to_readonly":
		d.Category, d.Strategy = "SpanPlan", "PointerLengthDescriptor"
		d.Evidence = evidenceForSubject(facts, op.RegionID, FactContiguous, FactBounded)
	case "tensor_inline_storage":
		d.Category, d.Strategy = "AggregatePlan", "InlineAggregate"
	case "tensor_view":
		d.Category, d.Strategy = "AggregatePlan", "PointerShapeDescriptor"
	}
	d.ID = "decision-" + digest([]byte(op.ID + "|" + d.Category + "|" + d.Strategy))[:16]
	return d
}

func planTensor(index int, tensor MIRTensorOperation, facts SemanticFactSet) TensorPlan {
	tp := TensorPlan{MIRIndex: index, Operation: tensor.Kind, Strategy: "DirectLoopNest", FreeIndices: append([]string{}, tensor.FreeIndices...), ReductionIndices: append([]string{}, tensor.ReduceIndices...), DestinationRegion: tensor.Output.RegionID, AliasPolicy: "ConservativeMayAlias", Alignment: map[string]int{}, BoundsPolicy: "AlreadyProven", ShapeCompatibility: "FixedShapeProven"}
	tp.TraversalOrder = append(append([]string{}, tp.FreeIndices...), tp.ReductionIndices...)
	if len(tp.TraversalOrder) == 0 && tensor.Output.Rank > 0 {
		for i := 0; i < tensor.Output.Rank; i++ {
			tp.TraversalOrder = append(tp.TraversalOrder, fmt.Sprintf("free%d", i))
		}
		tp.TraversalOrder = append(tp.TraversalOrder, "reduction")
	}
	regions := []string{tensor.Output.RegionID}
	for _, operand := range tensor.Operands {
		tp.InputRegions = append(tp.InputRegions, operand.RegionID)
		regions = append(regions, operand.RegionID)
	}
	allContiguous, allBounded, allFixed, allAllocation, allCopy, allTransfer, allAligned := FactProven, FactProven, FactProven, FactProven, FactProven, FactProven, FactProven
	var ids []string
	for _, region := range regions {
		for _, pair := range []struct {
			kind   SemanticFactKind
			result *SemanticFactCertainty
		}{{FactContiguous, &allContiguous}, {FactBounded, &allBounded}, {FactFixedShape, &allFixed}, {FactNoAllocation, &allAllocation}, {FactNoCopy, &allCopy}, {FactNoOwnershipTransfer, &allTransfer}, {FactAligned, &allAligned}} {
			outcome, factID := proveSubject(facts, pair.kind, region)
			if outcome != FactProven {
				*pair.result = outcome
			}
			if factID != "" {
				ids = append(ids, factID)
			}
		}
		if alignment, ok := facts.KnownAlignment(region); ok {
			tp.Alignment[region] = alignment
		} else {
			tp.Alignment[region] = 0
		}
	}
	disjoint := FactProven
	for _, input := range tp.InputRegions {
		if facts.AreDisjoint(tp.DestinationRegion, input) != FactProven {
			disjoint = FactUnknown
			continue
		}
		for _, fact := range facts.FactsFor(tp.DestinationRegion) {
			if fact.Kind == FactDisjoint && fact.Outcome == FactProven && factMentionsRegion(fact, input) {
				ids = append(ids, fact.ID)
				break
			}
		}
	}
	for _, guard := range tensor.ShapeGuards {
		if guard != "" {
			tp.RuntimeGuards = append(tp.RuntimeGuards, guard)
		}
	}
	if hasRuntimeShape(tensor.Output) {
		allFixed = FactDisproven
	}
	for _, operand := range tensor.Operands {
		if hasRuntimeShape(operand) {
			allFixed = FactDisproven
		}
	}
	if allFixed != FactProven {
		tp.ShapeCompatibility, tp.BoundsPolicy = "RuntimeGuardRequired", "HoistedRuntime"
		if len(tp.RuntimeGuards) == 0 {
			tp.RuntimeGuards = []string{"exact_shape_compatibility"}
		}
	}
	if disjoint == FactProven {
		tp.AliasPolicy = "ProvenDisjoint"
	}
	eligibility := DecisionEligible
	if allContiguous != FactProven || allBounded != FactProven || allFixed != FactProven || allAllocation != FactProven || allCopy != FactProven || allTransfer != FactProven || allAligned != FactProven || disjoint != FactProven {
		eligibility = DecisionConservativeFallback
	}
	tp.Vectorization = OptimizationEligibility{Contiguous: allContiguous, Bounded: allBounded, Alignment: allAligned, DisjointInputs: disjoint, FixedShape: allFixed, NoAllocation: allAllocation, NoCopy: allCopy, NoOwnershipTransfer: allTransfer, Outcome: eligibility, Selected: false, Evidence: PlanningEvidence{FactIDs: uniqueSorted(ids), Claims: []string{"Planner consumes qualified facts; SIMD selection is disabled in R4l"}}}
	return tp
}

func planAutomata(mir *MIR) []AutomataPlan {
	var out []AutomataPlan
	for _, automata := range mir.Automata {
		if automata.StateEnvironment == nil {
			continue
		}
		plan := AutomataPlan{Identity: automata.Name, StateEnvironment: automata.StateEnvironment.Identity, EnvironmentStrategy: "InlineExplicitStruct", Scheduler: "None", YieldStrategy: "Deferred"}
		for _, field := range automata.StateEnvironment.Fields {
			plan.StateFields = append(plan.StateFields, AutomataStoragePlan{Identity: field.Identity, Name: field.Name, Type: field.Type.String(), Classification: field.Classification, Strategy: "InlineField", HasDrop: field.HasDrop})
		}
		for _, machine := range automata.Machines {
			mp := MachinePlan{Identity: automata.Name + "." + machine.Name, CurrentStateSlot: automata.Name + "." + machine.Name + "#current-state", DispatchStrategy: "Switch", InitialState: automata.Name + "." + machine.Name + "." + machine.States[0].Name}
			for _, field := range machine.Fields {
				mp.Fields = append(mp.Fields, AutomataStoragePlan{Identity: field.Identity, Name: field.Name, Type: field.Type.String(), Classification: field.Classification, Strategy: "InlineField", HasDrop: field.HasDrop})
			}
			for _, state := range machine.States {
				mp.StateIdentities = append(mp.StateIdentities, automata.Name+"."+machine.Name+"."+state.Name)
				for _, op := range state.Operations {
					if op.Kind == "state_transition" {
						mp.Transitions = append(mp.Transitions, state.Name+"->"+op.Detail)
					}
				}
				for _, match := range state.TransitionMatches {
					planned := TransitionMatchPlan{State: state.Name, Strategy: "CategoricalSwitch", EvaluationOrder: "ScrutineeOnceThenSelectedTarget", NoMatchPolicy: match.NoMatchPolicy, CleanupEdge: match.CleanupEdge}
					for _, arm := range match.Arms {
						planned.Targets = append(planned.Targets, arm.TargetState)
					}
					mp.TransitionMatches = append(mp.TransitionMatches, planned)
				}
				for _, decision := range state.TransitionDecisions {
					planned := TransitionDecidePlan{State: state.Name, Strategy: "GuardedStraightLineHardmax", GuardEvaluationOrder: "DeclarationOrderOnce", ScoreEvaluationOrder: "EnabledDeclarationOrderOnce", ScoreType: decision.ScoreType.String(), TiePolicy: decision.TiePolicy, NoEnabledPolicy: decision.NoEnabledPolicy, CleanupEdge: decision.CleanupEdge, BranchlessEligible: false}
					for _, candidate := range decision.Candidates {
						planned.Targets = append(planned.Targets, candidate.TargetState)
					}
					mp.TransitionDecisions = append(mp.TransitionDecisions, planned)
				}
				for _, inference := range state.TransitionInferences {
					planned := TransitionInferPlan{State: state.Name, Strategy: "ScalarStableSoftMaxThenPolicy", Policy: inference.Policy, Normalization: inference.Normalization, ScoreType: inference.ScoreType, GuardEvaluationOrder: "DeclarationOrderOnce", ScoreEvaluationOrder: "EnabledDeclarationOrderOnce", CleanupEdge: inference.CleanupEdge}
					for _, candidate := range inference.Candidates {
						planned.Targets = append(planned.Targets, candidate.TargetState)
					}
					mp.TransitionInferences = append(mp.TransitionInferences, planned)
				}
			}
			plan.Machines = append(plan.Machines, mp)
		}
		out = append(out, plan)
	}
	return out
}

func planAggregates(mir *MIR) []AggregatePlan {
	var out []AggregatePlan
	for _, s := range mir.Structs {
		representation := "InlineAggregate"
		if s.Class {
			representation = "WrapperStruct"
		}
		out = append(out, AggregatePlan{Identity: s.Name, Kind: map[bool]string{true: "class", false: "struct"}[s.Class], Representation: representation})
	}
	for _, l := range mir.Layouts {
		out = append(out, AggregatePlan{Identity: l.Name, Kind: "layout", Representation: "StaticOffsets", Evidence: PlanningEvidence{Claims: []string{"NoRuntimeMetadata"}}})
	}
	for _, s := range mir.Streams {
		out = append(out, AggregatePlan{Identity: s.Name, Kind: "stream", Representation: "ErasedAliasLayoutDescriptor", Evidence: PlanningEvidence{Claims: []string{"ZeroStorage"}}})
	}
	for _, w := range mir.Witnesses {
		out = append(out, AggregatePlan{Identity: w.ID, Kind: "dyn witness", Representation: "ObjectWitnessPair", Evidence: PlanningEvidence{Claims: []string{"NoAllocation"}}})
	}
	for _, storage := range mir.StorageTypes {
		representation := "InlineAggregate"
		if storage.Type.isReference() {
			representation = "PointerShapeDescriptor"
		}
		out = append(out, AggregatePlan{Identity: storage.Type.String(), Kind: string(storage.StorageKind), Representation: representation, Evidence: PlanningEvidence{Claims: []string{"Contiguous", "RowMajor"}}})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind == out[j].Kind {
			return out[i].Identity < out[j].Identity
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

func cleanupOwner(cleanups []MIRCleanup, owner string) bool {
	for _, cleanup := range cleanups {
		if cleanup.Owner == owner {
			return true
		}
	}
	return false
}

func ValidateLoweringPlan(mir *MIR, facts *SemanticFactSet, plan *LoweringPlan) error {
	fail := func(code, message string) error { return evt1Diagnostic(code, message, Span{}) }
	if mir == nil || facts == nil || plan == nil || plan.Schema != PlanSchema || plan.PlanID == "" {
		return fail("PLAN_ARTIFACT_INVALID", "lowering plan omits schema, identity, MIR, or facts")
	}
	mirBytes, _ := json.Marshal(mir)
	if plan.MIRIdentity != digest(mirBytes) {
		return fail("PLAN_ARTIFACT_INVALID", "lowering plan references a stale MIR identity")
	}
	if plan.PlanID != loweringPlanIdentity(plan) {
		return fail("PLAN_ARTIFACT_INVALID", "lowering plan identity is not deterministic for its contents")
	}
	if err := validateTarget(plan.Target); err != nil {
		return err
	}
	if len(plan.Functions) != len(mir.Functions) {
		return fail("PLAN_ARTIFACT_INVALID", "lowering plan does not cover every MIR function")
	}
	expectedAutomataPlans := planAutomata(mir)
	if len(plan.Automata) != len(expectedAutomataPlans) {
		return fail("PLAN_AUTOMATA_INVALID", "lowering plan does not cover every explicit-state automata")
	}
	for _, automata := range plan.Automata {
		if automata.Identity == "" || automata.StateEnvironment == "" || automata.EnvironmentStrategy != "InlineExplicitStruct" || automata.Scheduler != "None" || automata.YieldStrategy != "Deferred" || len(automata.Machines) == 0 {
			return fail("PLAN_AUTOMATA_INVALID", "automata plan invents runtime policy or omits explicit storage")
		}
		for _, machine := range automata.Machines {
			if machine.CurrentStateSlot == "" || machine.DispatchStrategy != "Switch" || len(machine.StateIdentities) == 0 {
				return fail("PLAN_AUTOMATA_INVALID", "machine plan omits state identity or explicit dispatch")
			}
		}
	}
	expectedAutomataJSON, _ := json.Marshal(expectedAutomataPlans)
	actualAutomataJSON, _ := json.Marshal(plan.Automata)
	if string(actualAutomataJSON) != string(expectedAutomataJSON) {
		return fail("PLAN_AUTOMATA_INVALID", "automata plan does not preserve MIR storage, identity, transition, or dispatch evidence")
	}
	witnesses := map[string]bool{}
	for _, w := range mir.Witnesses {
		witnesses[w.ID] = true
	}
	for i, fp := range plan.Functions {
		fn := mir.Functions[i]
		encoded, _ := json.Marshal(fn)
		if fp.Function != fn.Name || fp.MIRIdentity != digest(encoded) {
			return fail("PLAN_ARTIFACT_INVALID", "function plan references stale or reordered MIR")
		}
		ops := map[string]MIROperation{}
		for _, op := range fn.Operations {
			ops[op.ID] = op
		}
		for _, d := range fp.Decisions {
			op, ok := ops[d.MIRID]
			if !ok {
				return fail("PLAN_UNSUPPORTED_MIR", "planning decision references unknown MIR operation "+d.MIRID)
			}
			if (d.Strategy == "PerAccessRuntime" || d.Strategy == "HoistedRuntime") && len(d.RuntimeGuards) == 0 {
				return fail("PLAN_RUNTIME_GUARD_MISSING", "bounds plan removed a required runtime guard")
			}
			if strings.HasPrefix(d.Strategy, "Vector") {
				return fail("PLAN_TARGET_UNSUPPORTED", "R4l cannot select vector lowering")
			}
			for _, id := range d.Evidence.FactIDs {
				fact, ok := factByID(*facts, id)
				if !ok {
					return fail("PLAN_REQUIRED_FACT_MISSING", "planning evidence references absent fact "+id)
				}
				if fact.Outcome != FactProven {
					return fail("PLAN_REQUIRED_FACT_DISPROVEN", "selected planning evidence is not proven: "+id)
				}
			}
			if op.Kind == "dyn_make" {
				w := strings.TrimSpace(strings.Split(op.Detail, "<-")[0])
				if !witnesses[w] {
					return fail("PLAN_WITNESS_INVALID", "dyn construction references invalid witness "+w)
				}
			}
		}
		for j, tp := range fp.Tensors {
			if tp.MIRIndex != j || j >= len(fn.TensorOperations) || tp.Strategy != "DirectLoopNest" {
				return fail("PLAN_TENSOR_INVALID", "tensor plan does not agree with Tensor MIR")
			}
			if tp.Vectorization.Selected {
				return fail("PLAN_INVALID_STRATEGY", "R4l tensor plan selected SIMD")
			}
			if tp.AliasPolicy == "ProvenDisjoint" {
				for _, input := range tp.InputRegions {
					if facts.AreDisjoint(tp.DestinationRegion, input) != FactProven {
						return fail("PLAN_REQUIRED_FACT_MISSING", "tensor plan requires an unproven disjoint-region fact")
					}
				}
			}
		}
		for _, dispatch := range fp.Dispatch {
			if dispatch.Operation == "dyn_make" && !witnesses[dispatch.Witness] {
				return fail("PLAN_WITNESS_INVALID", "dispatch plan references invalid witness "+dispatch.Witness)
			}
		}
		live := map[string]bool{}
		expectedCleanup := map[string]MIRCleanup{}
		transferred := map[string]bool{}
		for _, c := range fn.Cleanups {
			if c.State == "live" {
				live[c.Owner] = true
				expectedCleanup[c.Owner] = c
			} else {
				transferred[c.Owner] = true
			}
		}
		seenDrops := map[string]bool{}
		for _, d := range fp.Cleanup.Drops {
			expected, ok := expectedCleanup[d.Owner]
			if !ok || seenDrops[d.Owner] || d.Order != expected.Order || d.Type != expected.Type || d.DropFunction != expected.DropFunction || d.Action != "DropAtScopeExit" {
				return fail("PLAN_CLEANUP_INCOMPLETE", "cleanup plan contains a duplicate or inconsistent drop")
			}
			seenDrops[d.Owner] = true
			delete(live, d.Owner)
		}
		if len(live) > 0 {
			return fail("PLAN_CLEANUP_INCOMPLETE", "cleanup plan omits live MIR owner")
		}
		seenSuppressed := map[string]bool{}
		for _, owner := range fp.Cleanup.Suppressed {
			if !transferred[owner] || seenSuppressed[owner] {
				return fail("PLAN_CLEANUP_INCOMPLETE", "cleanup plan suppresses a non-transferred owner")
			}
			seenSuppressed[owner] = true
		}
		if len(seenSuppressed) != len(transferred) {
			return fail("PLAN_CLEANUP_INCOMPLETE", "cleanup plan omits a transferred-owner suppression")
		}
	}
	return nil
}

func validateTarget(target TargetCapabilities) error {
	if target.Architecture != ArchitectureGenericC11 && target.Architecture != ArchitectureX86_64 && target.Architecture != ArchitectureAArch64 {
		return evt1Diagnostic("PLAN_TARGET_UNSUPPORTED", "unknown target architecture", Span{})
	}
	if target.PointerBits != 32 && target.PointerBits != 64 {
		return evt1Diagnostic("PLAN_TARGET_UNSUPPORTED", "target pointer width must be 32 or 64", Span{})
	}
	if target.Endian != "little" && target.Endian != "big" {
		return evt1Diagnostic("PLAN_TARGET_UNSUPPORTED", "target endianness is invalid", Span{})
	}
	return nil
}

func evidenceForSubject(facts SemanticFactSet, subject string, kinds ...SemanticFactKind) PlanningEvidence {
	e := PlanningEvidence{}
	for _, kind := range kinds {
		outcome, id := proveSubject(facts, kind, subject)
		e.Claims = append(e.Claims, fmt.Sprintf("%s(%s)=%s", kind, subject, outcome))
		if id != "" {
			e.FactIDs = append(e.FactIDs, id)
		}
	}
	e.FactIDs = uniqueSorted(e.FactIDs)
	return e
}

func proveSubject(facts SemanticFactSet, kind SemanticFactKind, subject string) (SemanticFactCertainty, string) {
	for _, f := range facts.FactsFor(subject) {
		if f.Kind == kind {
			return f.Outcome, f.ID
		}
	}
	return FactUnknown, ""
}
func factByID(facts SemanticFactSet, id string) (MIRSemanticFact, bool) {
	for _, f := range facts.facts {
		if f.ID == id {
			return f, true
		}
	}
	return MIRSemanticFact{}, false
}
func factMentionsRegion(fact MIRSemanticFact, region string) bool {
	for _, subject := range fact.Subjects {
		if subject.RegionID == region || subject.Name == region {
			return true
		}
	}
	return false
}
func hasRuntimeShape(o MIRTensorOperand) bool {
	for _, d := range o.Shape {
		if d.Runtime {
			return true
		}
	}
	return false
}
func firstGuard(g []string) string {
	if len(g) > 0 {
		return g[0]
	}
	return ""
}
func uniqueSorted(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

func (p *LoweringPlan) Function(name string) *FunctionPlan {
	for i := range p.Functions {
		if p.Functions[i].Function == name {
			return &p.Functions[i]
		}
	}
	return nil
}
func (p *FunctionPlan) Strategy(operation string) (string, bool) {
	for _, d := range p.Decisions {
		if d.Operation == operation {
			return d.Strategy, true
		}
	}
	return "", false
}
func (p *FunctionPlan) StrategyAt(operation string, span Span) (string, bool) {
	for _, d := range p.Decisions {
		if d.Operation == operation && d.SourceSpan == span {
			return d.Strategy, true
		}
	}
	return p.Strategy(operation)
}
