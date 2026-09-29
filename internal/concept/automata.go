package concept

import (
	"fmt"
	"strings"
)

const (
	evt1AutomataMaxDecls = 8
)

type StateDecl struct {
	Name     string `json:"name"`
	Initial  bool   `json:"initial,omitempty"`
	Terminal bool   `json:"terminal,omitempty"`
	Body     *Block `json:"body,omitempty"`
	Span     Span   `json:"span"`
}

type MachineDecl struct {
	Name       string      `json:"name"`
	Initial    bool        `json:"initial,omitempty"`
	ResultType Type        `json:"result_type"`
	ErrorType  Type        `json:"error_type"`
	Fields     []Field     `json:"fields,omitempty"`
	States     []StateDecl `json:"states,omitempty"`
	Span       Span        `json:"span"`
}

type AutomataDecl struct {
	Name        string        `json:"name"`
	Module      string        `json:"-"`
	InputType   Type          `json:"input_type,omitempty"`
	StateFields []Field       `json:"state_fields,omitempty"`
	Machines    []MachineDecl `json:"machines,omitempty"`
	Span        Span          `json:"span"`
}

type evt1AutomataInfo struct {
	Decl                 AutomataDecl
	RootMachine          string
	MachineOrdinal       map[string]int
	StateOrdinal         map[string]map[string]int
	MachineReachable     map[string]bool
	StateReachable       map[string]map[string]bool
	MaxActiveDepth       int
	ContinuationCapacity int
	CompletionStepBound  int
	GraphIdentity        string
	TopologyIdentity     string
	RuntimeIdentity      string
}

type MIRAutomata struct {
	Name                 string                       `json:"name"`
	InputType            string                       `json:"input_type,omitempty"`
	RootMachine          string                       `json:"root_machine"`
	MaxActiveDepth       int                          `json:"max_active_depth"`
	ContinuationCapacity int                          `json:"continuation_capacity"`
	CompletionStepBound  int                          `json:"completion_step_bound"`
	GraphIdentity        string                       `json:"graph_identity"`
	TopologyIdentity     string                       `json:"topology_identity,omitempty"`
	RuntimeIdentity      string                       `json:"runtime_identity,omitempty"`
	SourceSpan           Span                         `json:"source_span"`
	StateEnvironment     *MIRAutomataStateEnvironment `json:"state_environment,omitempty"`
	Machines             []MIRMachine                 `json:"machines,omitempty"`
	MachineStack         *MIRMachineStack             `json:"machine_stack,omitempty"`
}

type MIRMachine struct {
	Name           string                 `json:"name"`
	Initial        bool                   `json:"initial,omitempty"`
	RuntimeOrdinal int                    `json:"runtime_ordinal"`
	Reachable      bool                   `json:"reachable,omitempty"`
	SourceSpan     Span                   `json:"source_span"`
	Fields         []MIRPersistentStorage `json:"fields,omitempty"`
	States         []MIRState             `json:"states,omitempty"`
	ResultType     *Type                  `json:"result_type,omitempty"`
	ErrorType      *Type                  `json:"error_type,omitempty"`
}

type MIRMachineStack struct {
	Capacity     int    `json:"capacity"`
	Storage      string `json:"storage"`
	Scheduler    string `json:"scheduler"`
	Continuation string `json:"continuation"`
	SharedState  string `json:"shared_state"`
}

type MIRMachineControl struct {
	Kind        string `json:"kind"`
	Machine     string `json:"machine,omitempty"`
	ResumeState string `json:"resume_state,omitempty"`
	Outcome     string `json:"outcome,omitempty"`
	PayloadType string `json:"payload_type,omitempty"`
	CleanupEdge string `json:"cleanup_edge"`
	SourceSpan  Span   `json:"source_span"`
}

type MIRState struct {
	Name string `json:"name"`
	// SemanticBody is the validated source of state control/dataflow for EVT2.
	// The checked MIR JSON remains a summary, as for MIRFunction.SemanticBody.
	SemanticBody         *Block                 `json:"-"`
	Initial              bool                   `json:"initial,omitempty"`
	Terminal             bool                   `json:"terminal,omitempty"`
	RuntimeOrdinal       int                    `json:"runtime_ordinal"`
	Reachable            bool                   `json:"reachable,omitempty"`
	SourceSpan           Span                   `json:"source_span"`
	TransitionMatches    []MIRTransitionMatch   `json:"transition_matches,omitempty"`
	TransitionDecisions  []MIRTransitionDecide  `json:"transition_decisions,omitempty"`
	TransitionInferences []MIRTransitionInfer   `json:"transition_inferences,omitempty"`
	Yields               []MIRYield             `json:"yields,omitempty"`
	Foreaches            []MIRForeach           `json:"foreach,omitempty"`
	Operations           []MIROperation         `json:"operations,omitempty"`
	Storage              []MIRPersistentStorage `json:"storage,omitempty"`
	MachineControl       []MIRMachineControl    `json:"machine_control,omitempty"`
	Reactions            []MIRReaction          `json:"reactions,omitempty"`
}

// MIRReaction summarizes one `on` reaction of a state; its body is lowered
// from the state's SemanticBody.
type MIRReaction struct {
	Pattern          string   `json:"pattern,omitempty"`
	PayloadBindings  []string `json:"payload_bindings,omitempty"`
	Guard            string   `json:"guard,omitempty"`
	Otherwise        bool     `json:"otherwise,omitempty"`
	CatchAll         bool     `json:"catch_all,omitempty"`
	Target           string   `json:"target,omitempty"`
	DeclarationOrder int      `json:"declaration_order"`
	SourceSpan       Span     `json:"source_span"`
}

type MIRTransitionInfer struct {
	Candidates      []MIRDecisionCandidate `json:"candidates"`
	ScoreType       string                 `json:"score_type"`
	Normalization   string                 `json:"normalization"`
	Policy          string                 `json:"policy"`
	NoEnabledPolicy string                 `json:"no_enabled_policy"`
	NaNPolicy       string                 `json:"nan_policy"`
	InfinityPolicy  string                 `json:"infinity_policy"`
	CleanupEdge     string                 `json:"cleanup_edge"`
	SourceSpan      Span                   `json:"source_span"`
}

type MIRTransitionMatch struct {
	Scrutinee     string                  `json:"scrutinee"`
	Arms          []MIRTransitionMatchArm `json:"arms"`
	Exhaustive    bool                    `json:"exhaustive"`
	NoMatchPolicy string                  `json:"no_match_policy"`
	CleanupEdge   string                  `json:"cleanup_edge"`
	SourceSpan    Span                    `json:"source_span"`
}

type MIRTransitionMatchArm struct {
	Pattern          string   `json:"pattern"`
	PayloadBindings  []string `json:"payload_bindings,omitempty"`
	Guard            string   `json:"guard,omitempty"`
	TargetState      string   `json:"target_state"`
	DeclarationOrder int      `json:"declaration_order"`
	SourceSpan       Span     `json:"source_span"`
}

type MIRTransitionDecide struct {
	Candidates      []MIRDecisionCandidate `json:"candidates"`
	ScoreType       Type                   `json:"score_type"`
	TiePolicy       string                 `json:"tie_policy"`
	NoEnabledPolicy string                 `json:"no_enabled_policy"`
	CleanupEdge     string                 `json:"cleanup_edge"`
	SourceSpan      Span                   `json:"source_span"`
}

type MIRDecisionCandidate struct {
	TargetState      string `json:"target_state"`
	Guard            string `json:"guard,omitempty"`
	Score            string `json:"score"`
	DeclarationOrder int    `json:"declaration_order"`
	SourceSpan       Span   `json:"source_span"`
}

type MIRAutomataStateEnvironment struct {
	Identity   string                 `json:"identity"`
	Fields     []MIRPersistentStorage `json:"fields"`
	Shared     bool                   `json:"shared"`
	Explicit   bool                   `json:"explicit"`
	SourceSpan Span                   `json:"source_span"`
}

type MIRPersistentStorage struct {
	Identity string `json:"identity"`
	// Initializer is retained in memory for native frame construction.
	Initializer    Expr   `json:"-"`
	Name           string `json:"name"`
	Type           Type   `json:"type"`
	Classification string `json:"classification"`
	Ordinal        int    `json:"ordinal"`
	Mutable        bool   `json:"mutable"`
	HasDrop        bool   `json:"has_drop,omitempty"`
	Provenance     string `json:"provenance,omitempty"`
	SourceSpan     Span   `json:"source_span"`
}

func evt1MachineOutcomeTypeName(automataName, machineName string) string {
	return automataName + machineName + "MachineOutcome"
}

func evt1ValidateAutomataDecls(env *semanticEnv, module Module) error {
	if len(module.Automata) > evt1AutomataMaxDecls {
		return evt1Diagnostic("CV4265", fmt.Sprintf("automata declaration count %d exceeds limit %d", len(module.Automata), evt1AutomataMaxDecls), module.Automata[evt1AutomataMaxDecls].Span)
	}
	for _, decl := range module.Automata {
		info, err := evt1ValidateCanonicalAutomata(env, decl)
		if err != nil {
			return err
		}
		env.automataInfo[decl.Name] = info
	}
	return nil
}

func evt1ValidateCanonicalAutomata(env *semanticEnv, decl AutomataDecl) (*evt1AutomataInfo, error) {
	if len(decl.Machines) == 0 {
		return nil, evt1Diagnostic("AUTOMATA_MIR_INVALID", fmt.Sprintf("automata %s requires at least one machine", decl.Name), decl.Span)
	}
	info := &evt1AutomataInfo{Decl: decl, RootMachine: decl.Machines[0].Name, MachineOrdinal: map[string]int{}, StateOrdinal: map[string]map[string]int{}, MachineReachable: map[string]bool{}, StateReachable: map[string]map[string]bool{}}
	stateEnvName := decl.Name + "#state"
	stateTypes := map[string]Type{}
	for i := range decl.StateFields {
		field := &decl.StateFields[i]
		if _, exists := stateTypes[field.Name]; exists {
			return nil, evt1Diagnostic("AUTOMATA_STATE_DUPLICATE_FIELD", fmt.Sprintf("duplicate automata state field %s.%s", decl.Name, field.Name), field.Span)
		}
		if err := validateKnownType(env, field.Type, field.Span, "", false); err != nil {
			return nil, err
		}
		resolved, err := evt1ResolveType(env, nil, field.Type)
		if err != nil {
			return nil, err
		}
		field.Type = resolved
		if evt1CarriesInvalidatableReference(env, resolved) {
			return nil, evt1Diagnostic("SCOPED_AUTHORITY_PERSISTENT_FIELD", fmt.Sprintf("reference-struct field %s.%s cannot persist across machine suspension", decl.Name, field.Name), field.Span)
		}
		stateTypes[field.Name] = resolved
	}
	env.fieldSets[stateEnvName] = stateTypes
	var inputEnum EnumDecl
	if decl.InputType.Name != "" {
		resolved, err := evt1ResolveType(env, nil, decl.InputType)
		if err != nil {
			return nil, err
		}
		enumDecl, ok := env.enums[resolved.Name]
		if !ok || resolved.PointerTo != nil || resolved.ArrayElem != nil || len(resolved.TypeArgs) != 0 {
			return nil, evt1Diagnostic("AUTOMATA_INPUT_INVALID", fmt.Sprintf("automata %s input must be an enum type, got %s", decl.Name, decl.InputType.String()), decl.InputType.Span)
		}
		decl.InputType = resolved
		inputEnum = enumDecl
	}
	machineNames := map[string]bool{}
	allMachineNames := map[string]bool{}
	for _, declaredMachine := range decl.Machines {
		allMachineNames[declaredMachine.Name] = true
		outcomeFields := map[string]Type{"tag": {Name: "int", Kind: TypeBuiltin}}
		if declaredMachine.ResultType.Name != "void" {
			outcomeFields["success"] = declaredMachine.ResultType
		}
		if declaredMachine.ErrorType.Name != "void" {
			outcomeFields["failure"] = declaredMachine.ErrorType
		}
		env.fieldSets[evt1MachineOutcomeTypeName(decl.Name, declaredMachine.Name)] = outcomeFields
	}
	for mi := range decl.Machines {
		machine := &decl.Machines[mi]
		if machineNames[machine.Name] {
			return nil, evt1Diagnostic("AUTOMATA_DUPLICATE_MACHINE", fmt.Sprintf("duplicate machine %s in automata %s", machine.Name, decl.Name), machine.Span)
		}
		machineNames[machine.Name] = true
		if err := validateKnownType(env, machine.ResultType, machine.ResultType.Span, "", false); err != nil {
			return nil, err
		}
		if err := validateKnownType(env, machine.ErrorType, machine.ErrorType.Span, "", false); err != nil {
			return nil, err
		}
		machine.ResultType, _ = evt1ResolveType(env, nil, machine.ResultType)
		machine.ErrorType, _ = evt1ResolveType(env, nil, machine.ErrorType)
		outcomeFields := map[string]Type{"tag": {Name: "int", Kind: TypeBuiltin}}
		if machine.ResultType.Name != "void" {
			outcomeFields["success"] = machine.ResultType
		}
		if machine.ErrorType.Name != "void" {
			outcomeFields["failure"] = machine.ErrorType
		}
		env.fieldSets[evt1MachineOutcomeTypeName(decl.Name, machine.Name)] = outcomeFields
		info.MachineOrdinal[machine.Name] = mi
		info.MachineReachable[machine.Name] = true
		info.StateOrdinal[machine.Name] = map[string]int{}
		info.StateReachable[machine.Name] = map[string]bool{}
		machineTypeName := decl.Name + "#" + machine.Name + "#machine"
		machineTypes := map[string]Type{}
		initializerScope := evt1ModuleScope(env)
		for stateIndex, stateField := range decl.StateFields {
			provenance := evt1InitialParameterProvenance(env, stateField.Type, stateIndex, initializerScope.depth)
			initializerScope.declare(stateField.Name, evt1ValueBinding{t: stateField.Type, mutable: !stateField.Type.Const, state: evt1StorageInitialized, provenance: provenance})
		}
		for fi := range machine.Fields {
			field := &machine.Fields[fi]
			if _, exists := machineTypes[field.Name]; exists {
				return nil, evt1Diagnostic("MACHINE_STATE_ACCESS_INVALID", fmt.Sprintf("duplicate machine field %s.%s", machine.Name, field.Name), field.Span)
			}
			if err := validateKnownType(env, field.Type, field.Span, "", false); err != nil {
				return nil, err
			}
			resolved, err := evt1ResolveType(env, nil, field.Type)
			if err != nil {
				return nil, err
			}
			field.Type = resolved
			if evt1CarriesInvalidatableReference(env, resolved) {
				return nil, evt1Diagnostic("SCOPED_AUTHORITY_PERSISTENT_FIELD", fmt.Sprintf("reference-struct field %s.%s cannot persist across machine suspension", machine.Name, field.Name), field.Span)
			}
			if resolved.Kind == TypeCallable && resolved.CallableProvenance != string(evt1ProvenanceStatic) && field.Initializer == nil {
				return nil, evt1Diagnostic("CALLABLE_MACHINE_FIELD_SHORT_REF", fmt.Sprintf("lifetime-bound machine callable field %s.%s requires construction from persistent automata state", machine.Name, field.Name), field.Span)
			}
			if field.Initializer == nil && !evt1TypeCopyable(env, resolved) {
				return nil, evt1Diagnostic("MACHINE_STATE_ACCESS_INVALID", fmt.Sprintf("non-copyable machine field %s.%s requires an explicit initializer", machine.Name, field.Name), field.Span)
			}
			if field.Initializer != nil {
				initType, err := validateExprAgainstExpected(env, initializerScope, field.Initializer, resolved, nil, false)
				if err != nil {
					return nil, err
				}
				if !evt1TypesCompatible(env, resolved, initType, "") {
					return nil, evt1Diagnostic("MACHINE_STATE_ACCESS_INVALID", fmt.Sprintf("initializer for machine field %s.%s has type %s, expected %s", machine.Name, field.Name, initType.String(), resolved.String()), field.Initializer.exprSpan())
				}
				if !evt1CanTransferInitialize(env, resolved, field.Initializer) && !evt1TypeCopyable(env, resolved) {
					return nil, evt1Diagnostic("AUTOMATA_CAPTURE_MOVE_REQUIRED", fmt.Sprintf("machine field %s.%s initializer would copy non-copyable state", machine.Name, field.Name), field.Initializer.exprSpan())
				}
			}
			machineTypes[field.Name] = resolved
		}
		env.fieldSets[machineTypeName] = machineTypes
		if len(machine.States) == 0 {
			return nil, evt1Diagnostic("MACHINE_MIR_INVALID", fmt.Sprintf("machine %s requires at least one state", machine.Name), machine.Span)
		}
		stateNames := map[string]bool{}
		for si := range machine.States {
			state := &machine.States[si]
			if stateNames[state.Name] {
				return nil, evt1Diagnostic("MACHINE_DUPLICATE_STATE", fmt.Sprintf("duplicate state %s in machine %s", state.Name, machine.Name), state.Span)
			}
			stateNames[state.Name] = true
			info.StateOrdinal[machine.Name][state.Name] = si
			info.StateReachable[machine.Name][state.Name] = true
		}
		for si := range machine.States {
			state := &machine.States[si]
			if state.Body == nil {
				return nil, evt1Diagnostic("MACHINE_MIR_INVALID", fmt.Sprintf("state %s.%s requires a body", machine.Name, state.Name), state.Span)
			}
			if state.Terminal {
				// Entering a terminal state completes the machine neutrally in
				// the same Step, so it can never run a body.
				if si == 0 {
					return nil, evt1Diagnostic("TERMINAL_STATE_INITIAL", fmt.Sprintf("terminal state %s.%s cannot be the machine's first (initial) state", machine.Name, state.Name), state.Span)
				}
				if len(state.Body.Statements) != 0 {
					return nil, evt1Diagnostic("TERMINAL_STATE_BODY", fmt.Sprintf("terminal state %s.%s must have an empty body: entering it completes machine %s", machine.Name, state.Name, machine.Name), state.Body.Statements[0].statementSpan())
				}
			}
			if err := evt1ValidateMachineControlFlow(state.Body); err != nil {
				return nil, err
			}
			automataScope := evt1ModuleScope(env)
			automataScope.inAutomataState = true
			automataScope.automataName = decl.Name
			automataScope.machineName = machine.Name
			automataScope.machineNames = allMachineNames
			automataScope.machineResultType = machine.ResultType
			automataScope.machineErrorType = machine.ErrorType
			automataScope.transitionTargets = stateNames
			automataScope.declare("state", evt1ValueBinding{t: Type{Name: stateEnvName, Kind: TypeStruct}, mutable: true, state: evt1StorageInitialized})
			for _, field := range decl.StateFields {
				automataScope.declare(field.Name, evt1ValueBinding{t: field.Type, mutable: !field.Type.Const, state: evt1StorageInitialized, provenance: evt1LifetimeProvenance{Kind: evt1ProvenanceParameter}})
			}
			machineScope := newEVT1Scope(automataScope)
			machineScope.declare("machine", evt1ValueBinding{t: Type{Name: machineTypeName, Kind: TypeStruct}, mutable: true, state: evt1StorageInitialized})
			for _, field := range machine.Fields {
				machineScope.declare(field.Name, evt1ValueBinding{t: field.Type, mutable: !field.Type.Const, state: evt1StorageInitialized, provenance: evt1LifetimeProvenance{Kind: evt1ProvenanceParameter}})
			}
			if evt1StateHasReactions(*state.Body) {
				if decl.InputType.Name == "" {
					return nil, evt1Diagnostic("ON_REQUIRES_INPUT", fmt.Sprintf("state %s.%s reacts with `on`, but automata %s does not declare `with input`", machine.Name, state.Name, decl.Name), state.Span)
				}
				if err := evt1ValidateStateReactions(env, machineScope, decl.InputType, inputEnum, state); err != nil {
					return nil, err
				}
				continue
			}
			if err := validateBlock(env, machineScope, Type{Name: "void", Kind: TypeBuiltin}, *state.Body, nil, false); err != nil {
				return nil, err
			}
		}
	}
	info.Decl = decl
	info.MaxActiveDepth = 1
	info.CompletionStepBound = 1
	info.GraphIdentity = "automata-" + digest([]byte(evt1CanonicalAutomataIdentity(decl)))[:16]
	info.TopologyIdentity = info.GraphIdentity
	info.RuntimeIdentity = "runtime-" + digest([]byte(info.GraphIdentity + "|explicit-state"))[:16]
	return info, nil
}

func evt1CanonicalAutomataIdentity(decl AutomataDecl) string {
	var b strings.Builder
	b.WriteString(decl.Name + "#state")
	if decl.InputType.Name != "" {
		b.WriteString("|input:" + decl.InputType.String())
	}
	for _, field := range decl.StateFields {
		b.WriteString("|" + field.Name + ":" + field.Type.String())
	}
	for _, machine := range decl.Machines {
		b.WriteString("|machine:" + machine.Name)
		for _, field := range machine.Fields {
			b.WriteString("|field:" + field.Name + ":" + field.Type.String())
		}
		for _, state := range machine.States {
			b.WriteString("|state:" + state.Name)
		}
	}
	return b.String()
}
