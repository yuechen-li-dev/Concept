package concept

import (
	"fmt"
	"strconv"
)

// lowerAutomataToLIR handles the single-frame canonical subset. The current
// EVT1 pushdown and signal-driven forms retain explicit unsupported boundaries.
func lowerAutomataToLIR(a MIRAutomata, env *semanticEnv) ([]LIRFunction, error) {
	if a.MachineStack == nil || a.StateEnvironment == nil {
		return nil, fmt.Errorf("EVT2_UNSUPPORTED_SIGNAL_AUTOMATA %s", a.Name)
	}
	for _, machine := range a.Machines {
		for _, state := range machine.States {
			for _, control := range state.MachineControl {
				if control.Kind == "push_machine" {
					return nil, fmt.Errorf("EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP %s.%s.%s", a.Name, machine.Name, state.Name)
				}
			}
		}
	}
	if len(a.Machines) != 1 {
		return nil, fmt.Errorf("EVT2_UNSUPPORTED_AUTOMATA_MULTI_MACHINE %s", a.Name)
	}
	machine := a.Machines[0]
	identity := a.GraphIdentity + ":" + machine.Name
	frame, err := machineFrameLayout(a, machine, identity, env)
	if err != nil {
		return nil, err
	}
	init, err := lowerMachineInit(a, machine, frame)
	if err != nil {
		return nil, err
	}
	step, err := lowerMachineStep(a, machine, frame)
	if err != nil {
		return nil, err
	}
	return []LIRFunction{init, step}, nil
}

func machineFrameLayout(a MIRAutomata, machine MIRMachine, identity string, env *semanticEnv) (*LIRMachineFunction, error) {
	fields := []Field{
		{Name: "current_state", Type: Type{Name: "uint32", Kind: TypeBuiltin}},
		{Name: "completed", Type: Type{Name: "bool", Kind: TypeBuiltin}},
	}
	storage := []MIRPersistentStorage{}
	storage = append(storage, a.StateEnvironment.Fields...)
	storage = append(storage, machine.Fields...)
	for _, field := range storage {
		if field.Type.Name != "int" && field.Type.Name != "uint" && field.Type.Name != "bool" {
			return nil, fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_FIELD_TYPE %s.%s", a.Name, field.Name)
		}
		fields = append(fields, Field{Name: field.Name, Type: field.Type})
	}
	offsets, size, align, err := evt1StructFieldOffsets(env, StructDecl{Name: identity, Fields: fields})
	if err != nil {
		return nil, err
	}
	frame := &LIRMachineFunction{Identity: identity, Size: size, Alignment: align}
	for i, field := range fields {
		lt, err := lirType(field.Type)
		if err != nil {
			return nil, err
		}
		fieldSize, fieldAlign, err := evt1TypeGeometry(env, field.Type)
		if err != nil {
			return nil, err
		}
		name := field.Name
		if i >= 2 && i < 2+len(a.StateEnvironment.Fields) {
			name = "shared." + name
		}
		if i >= 2+len(a.StateEnvironment.Fields) {
			name = "machine." + name
		}
		frame.Fields = append(frame.Fields, LIRMachineField{ID: i, Name: name, Type: lt, Offset: offsets[i], Size: fieldSize, Alignment: fieldAlign})
	}
	for i, state := range machine.States {
		if state.RuntimeOrdinal != i {
			return nil, fmt.Errorf("EVT2_MACHINE_STATE_ORDINAL %s.%s", machine.Name, state.Name)
		}
		frame.States = append(frame.States, LIRMachineState{ID: i, Name: state.Name, Block: -1, Source: state.SourceSpan})
	}
	return frame, nil
}

func machineBuilderFor(a MIRAutomata, machine MIRMachine, frame *LIRMachineFunction, role string) *lirBuilder {
	meta := *frame
	meta.Role = role
	meta.Fields = append([]LIRMachineField(nil), frame.Fields...)
	meta.States = append([]LIRMachineState(nil), frame.States...)
	b := &lirBuilder{machine: &meta, frameParam: 0, names: map[string]lirBinding{}, machineFields: map[string]int{}, machineStates: map[string]int{}}
	for _, state := range frame.States {
		b.machineStates[state.Name] = state.ID
	}
	for i, field := range frame.Fields {
		if i < 2 {
			continue
		}
		if len(field.Name) > 7 && field.Name[:7] == "shared." {
			b.machineFields[field.Name[7:]] = i
		}
		if role == "step" && len(field.Name) > 8 && field.Name[:8] == "machine." {
			b.machineFields[field.Name[8:]] = i
		}
	}
	b.fn = LIRFunction{Identity: frame.Identity + "#" + role, Name: a.Name + "." + machine.Name + "$" + role, Result: "void", Source: machine.SourceSpan, Machine: &meta}
	b.fn.Params = append(b.fn.Params, LIRValue{ID: 0, Type: LIRType("ptr<frame:" + frame.Identity + ">")})
	b.nextValue = 1
	b.newBlock()
	return b
}

func lowerMachineInit(a MIRAutomata, machine MIRMachine, frame *LIRMachineFunction) (LIRFunction, error) {
	b := machineBuilderFor(a, machine, frame, "init")
	sharedParams := make([]int, len(a.StateEnvironment.Fields))
	for i := range sharedParams {
		fieldID := i + 2
		sharedParams[i] = b.nextValue
		b.fn.Params = append(b.fn.Params, LIRValue{ID: b.nextValue, Type: frame.Fields[fieldID].Type})
		b.nextValue++
	}
	b.storeMachineState(0, machine.SourceSpan)
	b.storeMachineCompleted(false, machine.SourceSpan)
	for i, field := range a.StateEnvironment.Fields {
		fieldID := i + 2
		addr := b.frameFieldAddress(fieldID, field.SourceSpan)
		b.emit(LIRInstruction{Op: "store", Result: -1, Type: frame.Fields[fieldID].Type, Args: []int{addr, sharedParams[i]}, Slot: -1, Source: field.SourceSpan})
	}
	for i, field := range machine.Fields {
		fieldID := 2 + len(a.StateEnvironment.Fields) + i
		var value int
		if field.Initializer != nil {
			v, typ, err := b.expr(field.Initializer)
			if err != nil {
				return LIRFunction{}, fmt.Errorf("EVT2_MACHINE_INITIALIZER %s: %w", field.Name, err)
			}
			if typ != frame.Fields[fieldID].Type {
				return LIRFunction{}, fmt.Errorf("EVT2_MACHINE_INITIALIZER_TYPE %s", field.Name)
			}
			value = v
		} else {
			zero := "0"
			if frame.Fields[fieldID].Type == "bool" {
				zero = "false"
			}
			value = b.constant(frame.Fields[fieldID].Type, zero, field.SourceSpan)
		}
		addr := b.frameFieldAddress(fieldID, field.SourceSpan)
		b.emit(LIRInstruction{Op: "store", Result: -1, Type: frame.Fields[fieldID].Type, Args: []int{addr, value}, Slot: -1, Source: field.SourceSpan})
	}
	b.terminate(LIRTerminator{Op: "return", Source: machine.SourceSpan})
	return b.fn, nil
}

func lowerMachineStep(a MIRAutomata, machine MIRMachine, frame *LIRMachineFunction) (LIRFunction, error) {
	b := machineBuilderFor(a, machine, frame, "step")
	b.terminalStates = map[string]bool{}
	for _, state := range machine.States {
		if state.Terminal {
			b.terminalStates[state.Name] = true
		}
	}
	b.fn.Result = "machine_step_result"
	completedBlock := b.newBlock()
	dispatch := make([]int, len(machine.States))
	for i := range dispatch {
		dispatch[i] = b.newBlock()
	}
	stateBlocks := make([]int, len(machine.States))
	for i, state := range machine.States {
		stateBlocks[i] = b.newBlock()
		b.fn.Blocks[stateBlocks[i]].Label = "state." + state.Name
		b.machine.States[i].Block = stateBlocks[i]
	}
	invalidBlock := b.newBlock()
	b.fn.Blocks[invalidBlock].Label = "invalid-state"
	completedValue, _ := b.loadMachineField(1, machine.SourceSpan)
	b.terminate(LIRTerminator{Op: "branch", Value: completedValue, True: completedBlock, False: dispatch[0], Source: machine.SourceSpan})
	b.current = completedBlock
	b.fn.Blocks[completedBlock].Label = "already-completed"
	b.returnMachineResult("Completed", machine.SourceSpan)
	for i, state := range machine.States {
		b.current = dispatch[i]
		b.fn.Blocks[b.current].Label = "dispatch." + state.Name
		current, _ := b.loadMachineField(0, state.SourceSpan)
		tag := b.constant("u32", strconv.Itoa(i), state.SourceSpan)
		match := b.emit(LIRInstruction{Op: "eq", Result: 0, Type: "bool", Args: []int{current, tag}, Slot: -1, Source: state.SourceSpan})
		next := invalidBlock
		if i+1 < len(dispatch) {
			next = dispatch[i+1]
		}
		b.terminate(LIRTerminator{Op: "branch", Value: match, True: stateBlocks[i], False: next, Source: state.SourceSpan})
	}
	for i, state := range machine.States {
		if state.SemanticBody == nil {
			return LIRFunction{}, fmt.Errorf("EVT2_MACHINE_BODY_MISSING %s.%s", machine.Name, state.Name)
		}
		b.current = stateBlocks[i]
		b.activeState = i
		if err := b.block(*state.SemanticBody); err != nil {
			return LIRFunction{}, fmt.Errorf("%s.%s: %w", machine.Name, state.Name, err)
		}
		if b.fn.Blocks[b.current].Term.Op == "" {
			b.storeMachineState(i, state.SourceSpan)
			b.returnMachineResult("Active", state.SourceSpan)
		}
	}
	b.current = invalidBlock
	b.terminate(LIRTerminator{Op: "trap", Reason: "invalid_machine_state", Source: machine.SourceSpan})
	return b.fn, nil
}

func (b *lirBuilder) frameFieldAddress(id int, span Span) int {
	field := b.machine.Fields[id]
	return b.emit(LIRInstruction{Op: "frame_field_address", Result: 0, Type: LIRType("ptr<" + string(field.Type) + ">"), Args: []int{b.frameParam}, Slot: -1, FrameField: id, FrameOffset: field.Offset, Source: span})
}
func (b *lirBuilder) loadMachineField(id int, span Span) (int, LIRType) {
	field := b.machine.Fields[id]
	addr := b.frameFieldAddress(id, span)
	return b.emit(LIRInstruction{Op: "load", Result: 0, Type: field.Type, Args: []int{addr}, Slot: -1, Source: span}), field.Type
}
func (b *lirBuilder) machineFieldID(expr *FieldExpr) (int, error) {
	base, ok := expr.Receiver.(*NameExpr)
	if !ok {
		return 0, fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_FIELD %s", expr.Field)
	}
	prefix := ""
	if base.Name == "state" {
		prefix = "shared."
	} else if base.Name == "machine" {
		prefix = "machine."
	} else {
		return 0, fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_FIELD %s.%s", base.Name, expr.Field)
	}
	for i, field := range b.machine.Fields {
		if field.Name == prefix+expr.Field {
			return i, nil
		}
	}
	return 0, fmt.Errorf("EVT2_UNKNOWN_MACHINE_FIELD %s%s", prefix, expr.Field)
}
func (b *lirBuilder) storeMachineState(id int, span Span) {
	value := b.constant("u32", strconv.Itoa(id), span)
	addr := b.frameFieldAddress(0, span)
	b.emit(LIRInstruction{Op: "store", Result: -1, Type: "u32", Args: []int{addr, value}, Slot: -1, Source: span})
}
func (b *lirBuilder) storeMachineCompleted(value bool, span Span) {
	literal := "false"
	if value {
		literal = "true"
	}
	v := b.constant("bool", literal, span)
	addr := b.frameFieldAddress(1, span)
	b.emit(LIRInstruction{Op: "store", Result: -1, Type: "bool", Args: []int{addr, v}, Slot: -1, Source: span})
}
func (b *lirBuilder) returnMachineResult(result string, span Span) {
	v := b.emit(LIRInstruction{Op: "machine_result", Result: 0, Type: "machine_step_result", Slot: -1, Literal: result, Source: span})
	b.terminate(LIRTerminator{Op: "return", Value: v, Source: span})
}
