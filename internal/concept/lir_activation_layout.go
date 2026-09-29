package concept

import "fmt"

// ActivationStackLayout is the closed, caller-owned storage plan for one
// automata graph. A slot's tag selects the only live frame type in its storage.
// The plan describes layout; construction/destruction belong to LIR lowering.
type ActivationStackLayout struct {
	Identity       string
	Capacity       int
	RootTag        uint32
	SharedFields   []LIRMachineField
	Machines       []ActivationMachineLayout
	MaxFrameSize   int
	MaxFrameAlign  int
	SlotTagOffset  int
	SlotDataOffset int
	SlotSize       int
	SlotAlign      int
	DepthOffset    int
	DoneOffset     int
	SlotsOffset    int
	Size           int
	Alignment      int
}

type ActivationMachineLayout struct {
	Tag       uint32
	Identity  string
	Name      string
	Size      int
	Alignment int
	Fields    []LIRMachineField
	States    []LIRMachineState
}

func planActivationStack(a MIRAutomata, env *semanticEnv) (ActivationStackLayout, error) {
	if a.MachineStack == nil || a.StateEnvironment == nil || a.GraphIdentity == "" || a.RootMachine == "" {
		return ActivationStackLayout{}, fmt.Errorf("EVT2_ACTIVATION_GRAPH_INCOMPLETE %s", a.Name)
	}
	if a.MachineStack.Capacity != evt1MachineStackCapacity || a.MachineStack.Storage != "InlineBoundedSpecializedFrames" {
		return ActivationStackLayout{}, fmt.Errorf("EVT2_ACTIVATION_BOUND_INVALID %s", a.Name)
	}
	plan := ActivationStackLayout{Identity: a.GraphIdentity, Capacity: a.MachineStack.Capacity, MaxFrameAlign: 1}
	tags := map[uint32]bool{}
	names := map[string]bool{}
	rootFound := false
	for _, machine := range a.Machines {
		if !machine.Reachable {
			continue
		}
		tag := uint32(machine.RuntimeOrdinal)
		if machine.RuntimeOrdinal < 0 || tags[tag] || names[machine.Name] {
			return ActivationStackLayout{}, fmt.Errorf("EVT2_ACTIVATION_TAG_INVALID %s.%s", a.Name, machine.Name)
		}
		tags[tag], names[machine.Name] = true, true
		if machine.Name == a.RootMachine {
			plan.RootTag, rootFound = tag, true
		}
		frame, err := activationMachineFrame(a, machine, env)
		if err != nil {
			return ActivationStackLayout{}, err
		}
		plan.Machines = append(plan.Machines, frame)
		if frame.Size > plan.MaxFrameSize {
			plan.MaxFrameSize = frame.Size
		}
		if frame.Alignment > plan.MaxFrameAlign {
			plan.MaxFrameAlign = frame.Alignment
		}
	}
	if !rootFound || len(plan.Machines) == 0 {
		return ActivationStackLayout{}, fmt.Errorf("EVT2_ACTIVATION_ROOT_MISSING %s.%s", a.Name, a.RootMachine)
	}
	for _, machine := range a.Machines {
		if !machine.Reachable {
			continue
		}
		for _, state := range machine.States {
			for _, control := range state.MachineControl {
				if control.Kind == "push_machine" && !names[control.Machine] {
					return ActivationStackLayout{}, fmt.Errorf("EVT2_ACTIVATION_TARGET_MISSING %s.%s.%s", a.Name, machine.Name, state.Name)
				}
			}
		}
	}
	// Tag and storage are separate subobjects. Storage starts at an offset
	// aligned for every reachable frame, so no frame is reinterpreted as another.
	plan.SlotTagOffset = 0
	plan.SlotAlign = max(4, plan.MaxFrameAlign)
	plan.SlotDataOffset = evt1AlignUp(4, plan.MaxFrameAlign)
	plan.SlotSize = evt1AlignUp(plan.SlotDataOffset+plan.MaxFrameSize, plan.SlotAlign)
	plan.DepthOffset = 0
	plan.DoneOffset = 4
	sharedEnd := 5
	for i, field := range a.StateEnvironment.Fields {
		lt, err := activationScalarType(a.Name, field)
		if err != nil {
			return ActivationStackLayout{}, err
		}
		size, align, err := evt1TypeGeometry(env, field.Type)
		if err != nil {
			return ActivationStackLayout{}, err
		}
		sharedEnd = evt1AlignUp(sharedEnd, align)
		plan.SharedFields = append(plan.SharedFields, LIRMachineField{ID: i, Name: "shared." + field.Name, Type: lt, Offset: sharedEnd, Size: size, Alignment: align})
		sharedEnd += size
	}
	plan.Alignment = plan.SlotAlign
	plan.SlotsOffset = evt1AlignUp(sharedEnd, plan.SlotAlign)
	plan.Size = evt1AlignUp(plan.SlotsOffset+plan.Capacity*plan.SlotSize, plan.Alignment)
	return plan, nil
}

func activationMachineFrame(a MIRAutomata, machine MIRMachine, env *semanticEnv) (ActivationMachineLayout, error) {
	identity := a.GraphIdentity + ":" + machine.Name
	fields := []Field{{Name: "current_state", Type: Type{Name: "uint32", Kind: TypeBuiltin}}}
	for _, field := range machine.Fields {
		if _, err := activationScalarType(a.Name, field); err != nil {
			return ActivationMachineLayout{}, err
		}
		fields = append(fields, Field{Name: field.Name, Type: field.Type})
	}
	offsets, size, align, err := evt1StructFieldOffsets(env, StructDecl{Name: identity, Fields: fields})
	if err != nil {
		return ActivationMachineLayout{}, err
	}
	frame := ActivationMachineLayout{Tag: uint32(machine.RuntimeOrdinal), Identity: identity, Name: machine.Name, Size: size, Alignment: align}
	for i, field := range fields {
		lt, err := lirType(field.Type)
		if err != nil {
			return ActivationMachineLayout{}, err
		}
		fieldSize, fieldAlign, err := evt1TypeGeometry(env, field.Type)
		if err != nil {
			return ActivationMachineLayout{}, err
		}
		name := field.Name
		if i > 0 {
			name = "machine." + name
		}
		frame.Fields = append(frame.Fields, LIRMachineField{ID: i, Name: name, Type: lt, Offset: offsets[i], Size: fieldSize, Alignment: fieldAlign})
	}
	for i, state := range machine.States {
		if state.RuntimeOrdinal != i {
			return ActivationMachineLayout{}, fmt.Errorf("EVT2_ACTIVATION_STATE_ORDINAL %s.%s.%s", a.Name, machine.Name, state.Name)
		}
		frame.States = append(frame.States, LIRMachineState{ID: i, Name: state.Name, Block: -1, Source: state.SourceSpan})
	}
	return frame, nil
}

func activationScalarType(automata string, field MIRPersistentStorage) (LIRType, error) {
	if field.Type.isOwned() || field.Type.Name != "int" && field.Type.Name != "uint" && field.Type.Name != "bool" {
		return "", fmt.Errorf("EVT2_UNSUPPORTED_ACTIVATION_FIELD_TYPE %s.%s", automata, field.Name)
	}
	return lirType(field.Type)
}
