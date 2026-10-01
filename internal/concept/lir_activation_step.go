package concept

import (
	"fmt"
	"strconv"
	"strings"
)

type activationStepContext struct {
	automata           MIRAutomata
	machine            int
	topIndices         map[int]int
	completionTransfer int
}

func lowerActivationStep(a MIRAutomata, layout ActivationStackLayout) (LIRFunction, error) {
	// C terminal settling can complete several suspended frames in one Step.
	// Do not mistake an empty terminal body for an ordinary active state.
	for _, machine := range a.Machines {
		for _, state := range machine.States {
			if state.Terminal {
				return LIRFunction{}, fmt.Errorf("EVT2_UNSUPPORTED_PUSHDOWN_TERMINAL %s.%s", machine.Name, state.Name)
			}
		}
	}
	meta := &LIRActivationFunction{Role: "step", Layout: layout}
	meta.Layout.Machines = append([]ActivationMachineLayout(nil), layout.Machines...)
	meta.InitFields = []LIRMachineField{{ID: 0, Name: "depth", Type: "u32", Offset: layout.DepthOffset, Size: 4, Alignment: 4}, {ID: 1, Name: "completed", Type: "bool", Offset: layout.DoneOffset, Size: 1, Alignment: 1}}
	for _, field := range layout.SharedFields {
		field.ID = len(meta.InitFields)
		meta.InitFields = append(meta.InitFields, field)
	}
	b := &lirBuilder{fn: LIRFunction{Identity: layout.Identity + "#activation-step", Name: a.Name + "$activation_step", Result: "machine_step_result", Source: a.SourceSpan, Activation: meta}, names: map[string]lirBinding{}, activation: &activationStepContext{automata: a, topIndices: map[int]int{}, completionTransfer: -1}, frameParam: 0, nextValue: 1}
	b.fn.Params = []LIRValue{{ID: 0, Type: LIRType("ptr<activation:" + layout.Identity + ">")}}
	b.newBlock()
	done := b.newBlock()
	depthGuard := b.newBlock()
	tags := make([]int, len(layout.Machines))
	for i := range tags {
		tags[i] = b.newBlock()
	}
	invalidTag := b.newBlock()
	invalidState := b.newBlock()
	invalidDepth := b.newBlock()
	b.current = 0
	completed := b.loadActivationHeader(1, a.SourceSpan)
	b.terminate(LIRTerminator{Op: "branch", Value: completed, True: done, False: depthGuard, Source: a.SourceSpan})
	b.current = depthGuard
	b.fn.Blocks[depthGuard].Label = "validate-live-depth"
	depth := b.loadActivationHeader(0, a.SourceSpan)
	zero := b.constant("u32", "0", a.SourceSpan)
	positive := b.emit(LIRInstruction{Op: "gt", Result: 0, Type: "bool", Args: []int{depth, zero}, Slot: -1, Source: a.SourceSpan})
	b.terminate(LIRTerminator{Op: "branch", Value: positive, True: tags[0], False: invalidDepth, Source: a.SourceSpan})
	b.current = done
	b.fn.Blocks[done].Label = "already-completed"
	b.returnMachineResult("Completed", a.SourceSpan)
	for i, frame := range layout.Machines {
		var machine MIRMachine
		for _, m := range a.Machines {
			if m.Name == frame.Name {
				machine = m
				break
			}
		}
		b.selectActivationMachine(i)
		stateDispatch := make([]int, len(frame.States))
		stateBlocks := make([]int, len(frame.States))
		meta.Layout.Machines[i].States = append([]LIRMachineState(nil), frame.States...)
		for j := range stateDispatch {
			stateDispatch[j] = b.newBlock()
			stateBlocks[j] = b.newBlock()
			meta.Layout.Machines[i].States[j].Block = stateBlocks[j]
		}
		b.current = tags[i]
		b.fn.Blocks[b.current].Label = "tag." + frame.Name
		tag := b.loadTopTag(a.SourceSpan)
		expected := b.constant("u32", strconv.Itoa(int(frame.Tag)), a.SourceSpan)
		cmp := b.emit(LIRInstruction{Op: "eq", Result: 0, Type: "bool", Args: []int{tag, expected}, Slot: -1, Source: a.SourceSpan})
		next := invalidTag
		if i+1 < len(tags) {
			next = tags[i+1]
		}
		b.terminate(LIRTerminator{Op: "branch", Value: cmp, True: stateDispatch[0], False: next, Source: a.SourceSpan})
		for j, state := range machine.States {
			b.current = stateDispatch[j]
			b.fn.Blocks[b.current].Label = "dispatch." + frame.Name + "." + state.Name
			value, _ := b.loadMachineField(0, state.SourceSpan)
			expected := b.constant("u32", strconv.Itoa(j), state.SourceSpan)
			cmp := b.emit(LIRInstruction{Op: "eq", Result: 0, Type: "bool", Args: []int{value, expected}, Slot: -1, Source: state.SourceSpan})
			next := invalidState
			if j+1 < len(stateDispatch) {
				next = stateDispatch[j+1]
			}
			b.terminate(LIRTerminator{Op: "branch", Value: cmp, True: stateBlocks[j], False: next, Source: state.SourceSpan})
		}
		for j, state := range machine.States {
			if state.SemanticBody == nil {
				return LIRFunction{}, fmt.Errorf("EVT2_MACHINE_BODY_MISSING %s.%s", machine.Name, state.Name)
			}
			b.current = stateBlocks[j]
			b.fn.Blocks[b.current].Label = "state." + frame.Name + "." + state.Name
			b.activeState = j
			if err := b.block(*state.SemanticBody); err != nil {
				return LIRFunction{}, fmt.Errorf("%s.%s: %w", machine.Name, state.Name, err)
			}
			if b.fn.Blocks[b.current].Term.Op == "" {
				b.storeMachineState(j, state.SourceSpan)
				b.returnMachineResult("Active", state.SourceSpan)
			}
		}
	}
	b.current = invalidTag
	b.fn.Blocks[b.current].Label = "invalid-tag"
	b.terminate(LIRTerminator{Op: "trap", Reason: "invalid_machine_tag", Source: a.SourceSpan})
	b.current = invalidState
	b.fn.Blocks[b.current].Label = "invalid-state"
	b.terminate(LIRTerminator{Op: "trap", Reason: "invalid_machine_state", Source: a.SourceSpan})
	b.current = invalidDepth
	b.fn.Blocks[b.current].Label = "invalid-depth"
	b.terminate(LIRTerminator{Op: "trap", Reason: "invalid_machine_depth", Source: a.SourceSpan})
	return b.fn, nil
}

// Adapt only the frame binding and machine control. Expressions and statements
// continue through the ordinary lirBuilder, including nested source branches.
func (b *lirBuilder) selectActivationMachine(i int) {
	frame := b.fn.Activation.Layout.Machines[i]
	b.activation.machine = i
	b.machine = &LIRMachineFunction{Fields: []LIRMachineField{frame.Fields[0], {Name: "completed", Type: "bool"}}}
	b.machineFields = map[string]int{}
	b.machineStates = map[string]int{}
	for _, field := range b.fn.Activation.Layout.SharedFields {
		field.ID = len(b.machine.Fields)
		b.machine.Fields = append(b.machine.Fields, field)
		b.machineFields[strings.TrimPrefix(field.Name, "shared.")] = field.ID
	}
	for _, field := range frame.Fields[1:] {
		field.ID = len(b.machine.Fields)
		b.machine.Fields = append(b.machine.Fields, field)
		b.machineFields[strings.TrimPrefix(field.Name, "machine.")] = field.ID
	}
	for _, state := range frame.States {
		b.machineStates[state.Name] = state.ID
	}
}

func (b *lirBuilder) activationHeaderAddress(id int, span Span) int {
	field := b.fn.Activation.InitFields[id]
	return b.emit(LIRInstruction{Op: "activation_address", Result: 0, Type: LIRType("ptr<" + string(field.Type) + ">"), Args: []int{0}, Slot: -1, FrameField: id, FrameOffset: field.Offset, Source: span})
}
func (b *lirBuilder) loadActivationHeader(id int, span Span) int {
	addr := b.activationHeaderAddress(id, span)
	return b.emit(LIRInstruction{Op: "load", Result: 0, Type: b.fn.Activation.InitFields[id].Type, Args: []int{addr}, Slot: -1, Source: span})
}
func (b *lirBuilder) storeActivationHeader(id, value int, span Span) {
	addr := b.activationHeaderAddress(id, span)
	b.emit(LIRInstruction{Op: "store", Result: -1, Type: b.fn.Activation.InitFields[id].Type, Args: []int{addr, value}, Slot: -1, Source: span})
}
func (b *lirBuilder) topIndex(span Span) int {
	if index, ok := b.activation.topIndices[b.current]; ok {
		return index
	}
	depth := b.loadActivationHeader(0, span)
	one := b.constant("u32", "1", span)
	index := b.emit(LIRInstruction{Op: "sub", Result: 0, Type: "u32", Args: []int{depth, one}, Slot: -1, Source: span})
	b.emit(LIRInstruction{Op: "check_index", Result: -1, Type: "void", Args: []int{index}, Slot: -1, Extent: b.fn.Activation.Layout.Capacity, Source: span})
	b.activation.topIndices[b.current] = index
	return index
}
func (b *lirBuilder) activationSlotAddress(index, offset int, typ LIRType, span Span) int {
	l := b.fn.Activation.Layout
	return b.emit(LIRInstruction{Op: "index_address", Result: 0, Type: LIRType("ptr<" + string(typ) + ">"), Args: []int{0, index}, Slot: -1, Extent: l.Capacity, Stride: l.SlotSize, FrameOffset: l.SlotsOffset + offset, Source: span})
}
func (b *lirBuilder) loadTopTag(span Span) int {
	addr := b.activationSlotAddress(b.topIndex(span), b.fn.Activation.Layout.SlotTagOffset, "u32", span)
	return b.emit(LIRInstruction{Op: "load", Result: 0, Type: "u32", Args: []int{addr}, Slot: -1, Source: span})
}
func (b *lirBuilder) activationFieldAddress(id int, span Span) int {
	field := b.machine.Fields[id]
	if strings.HasPrefix(field.Name, "shared.") {
		return b.activationHeaderAddress(id, span)
	}
	return b.activationSlotAddress(b.topIndex(span), b.fn.Activation.Layout.SlotDataOffset+field.Offset, field.Type, span)
}
func (b *lirBuilder) lowerActivationPush(s *PushMachineStmt) error {
	l := b.fn.Activation.Layout
	childID := -1
	for i, m := range l.Machines {
		if m.Name == s.Machine {
			childID = i
			break
		}
	}
	resume, ok := b.machineStates[s.ResumeState]
	if childID < 0 || !ok {
		return fmt.Errorf("EVT2_ACTIVATION_PUSH_TARGET %s", s.Machine)
	}
	child := l.Machines[childID]
	b.fn.Activation.Transfers = append(b.fn.Activation.Transfers, LIRActivationTransfer{Kind: "push", Block: b.current, ParentTag: l.Machines[b.activation.machine].Tag, ChildTag: child.Tag, ResumeState: resume})
	depth := b.loadActivationHeader(0, s.Span)
	b.emit(LIRInstruction{Op: "check_index", Result: -1, Type: "void", Args: []int{depth}, Slot: -1, Extent: l.Capacity, Source: s.Span})
	var source MIRMachine
	for _, m := range b.activation.automata.Machines {
		if m.Name == s.Machine {
			source = m
			break
		}
	}
	// Scalar initializers can trap. Keep the parent continuation untouched until
	// all initializers succeed; the child remains raw storage in the meantime.
	for i, field := range source.Fields {
		typ := child.Fields[i+1].Type
		var value int
		if field.Initializer != nil {
			v, t, err := b.activationInitializer(field.Initializer)
			if err != nil {
				return err
			}
			if t != typ {
				return fmt.Errorf("EVT2_ACTIVATION_INITIALIZER_TYPE %s", field.Name)
			}
			value = v
		} else {
			literal := "0"
			if typ == "bool" {
				literal = "false"
			}
			value = b.constant(typ, literal, field.SourceSpan)
		}
		addr := b.activationSlotAddress(depth, l.SlotDataOffset+child.Fields[i+1].Offset, typ, s.Span)
		b.emit(LIRInstruction{Op: "store", Result: -1, Type: typ, Args: []int{addr, value}, Slot: -1, Source: s.Span})
	}
	state := b.constant("u32", "0", s.Span)
	addr := b.activationSlotAddress(depth, l.SlotDataOffset, "u32", s.Span)
	b.emit(LIRInstruction{Op: "store", Result: -1, Type: "u32", Args: []int{addr, state}, Slot: -1, Source: s.Span})
	b.storeMachineState(resume, s.Span)
	tag := b.constant("u32", strconv.Itoa(int(child.Tag)), s.Span)
	addr = b.activationSlotAddress(depth, l.SlotTagOffset, "u32", s.Span)
	b.emit(LIRInstruction{Op: "store", Result: -1, Type: "u32", Args: []int{addr, tag}, Slot: -1, Source: s.Span})
	one := b.constant("u32", "1", s.Span)
	next := b.emit(LIRInstruction{Op: "add", Result: 0, Type: "u32", Args: []int{depth, one}, Slot: -1, Source: s.Span})
	b.storeActivationHeader(0, next, s.Span)
	b.returnMachineResult("Active", s.Span)
	return nil
}

// Initializers execute in the child construction scope, not the caller's state
// scope. Parent locals and persistent fields cannot shadow shared bindings.
func (b *lirBuilder) activationInitializer(expr Expr) (int, LIRType, error) {
	names, fields := b.names, b.machineFields
	b.names = map[string]lirBinding{}
	b.machineFields = map[string]int{}
	for i, field := range b.machine.Fields {
		if strings.HasPrefix(field.Name, "shared.") {
			b.machineFields[strings.TrimPrefix(field.Name, "shared.")] = i
		}
	}
	defer func() { b.names, b.machineFields = names, fields }()
	return b.expr(expr)
}

func (b *lirBuilder) lowerActivationCompletion(s *MachineCompleteStmt) error {
	if s.Kind != "neutral" || s.Value != nil {
		return fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_OUTCOME %s", s.Kind)
	}
	l := b.fn.Activation.Layout
	if id := b.activation.completionTransfer; id >= 0 {
		t := &b.fn.Activation.Transfers[id]
		t.SourceBlocks = append(t.SourceBlocks, b.current)
		b.terminate(LIRTerminator{Op: "jump", True: b.fn.Blocks[t.Block].Term.True, Source: s.Span})
		return nil
	}
	transfer := LIRActivationTransfer{Kind: s.Operation, Block: b.current, ParentTag: l.Machines[b.activation.machine].Tag}
	tags := make([]int, len(l.Machines))
	for i := range tags {
		tags[i] = b.newBlock()
	}
	invalid := b.newBlock()
	b.terminate(LIRTerminator{Op: "jump", True: tags[0], Source: s.Span})
	for i, m := range l.Machines {
		b.current = tags[i]
		b.fn.Blocks[b.current].Label = "destroy-tag." + m.Name
		tag := b.loadTopTag(s.Span)
		expected := b.constant("u32", strconv.Itoa(int(m.Tag)), s.Span)
		cmp := b.emit(LIRInstruction{Op: "eq", Result: 0, Type: "bool", Args: []int{tag, expected}, Slot: -1, Source: s.Span})
		destroy := b.newBlock()
		publish := b.newBlock()
		transfer.DestroyBlocks = append(transfer.DestroyBlocks, destroy)
		transfer.PublishBlocks = append(transfer.PublishBlocks, publish)
		next := invalid
		if i+1 < len(tags) {
			next = tags[i+1]
		}
		b.terminate(LIRTerminator{Op: "branch", Value: cmp, True: destroy, False: next, Source: s.Span})
		b.current = destroy
		b.fn.Blocks[destroy].Label = "destroy." + m.Identity
		// Scalar-only frames have empty typed destruction regions. Retain and
		// verify the region before publishing depth, without a backend pseudo-op.
		b.terminate(LIRTerminator{Op: "jump", True: publish, Source: s.Span})
		b.current = publish
		b.fn.Blocks[publish].Label = "publish-pop." + m.Name
		index := b.topIndex(s.Span)
		b.storeActivationHeader(0, index, s.Span)
		zero := b.constant("u32", "0", s.Span)
		root := b.emit(LIRInstruction{Op: "eq", Result: 0, Type: "bool", Args: []int{index, zero}, Slot: -1, Source: s.Span})
		completed := b.newBlock()
		active := b.newBlock()
		b.terminate(LIRTerminator{Op: "branch", Value: root, True: completed, False: active, Source: s.Span})
		b.current = completed
		b.storeMachineCompleted(true, s.Span)
		b.returnMachineResult("Completed", s.Span)
		b.current = active
		b.returnMachineResult("Active", s.Span)
	}
	b.current = invalid
	b.terminate(LIRTerminator{Op: "trap", Reason: "invalid_machine_tag", Source: s.Span})
	b.activation.completionTransfer = len(b.fn.Activation.Transfers)
	b.fn.Activation.Transfers = append(b.fn.Activation.Transfers, transfer)
	return nil
}
