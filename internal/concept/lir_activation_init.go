package concept

import (
	"fmt"
	"strconv"
)

func lowerActivationRootInit(a MIRAutomata, layout ActivationStackLayout) (LIRFunction, error) {
	var root *MIRMachine
	var rootLayout *ActivationMachineLayout
	for i := range a.Machines {
		if a.Machines[i].Name == a.RootMachine {
			root = &a.Machines[i]
			break
		}
	}
	for i := range layout.Machines {
		if layout.Machines[i].Name == a.RootMachine {
			rootLayout = &layout.Machines[i]
			break
		}
	}
	if root == nil || rootLayout == nil {
		return LIRFunction{}, fmt.Errorf("EVT2_ACTIVATION_ROOT_MISSING %s", a.Name)
	}
	meta := &LIRActivationFunction{Role: "init", Layout: layout}
	addField := func(name string, typ LIRType, offset int) int {
		id := len(meta.InitFields)
		size := lirWidth(typ)
		if typ == "bool" {
			size = 1
		}
		meta.InitFields = append(meta.InitFields, LIRMachineField{ID: id, Name: name, Type: typ, Offset: offset, Size: size, Alignment: size})
		return id
	}
	depthID := addField("depth", "u32", layout.DepthOffset)
	doneID := addField("completed", "bool", layout.DoneOffset)
	sharedIDs := make([]int, len(layout.SharedFields))
	for i, field := range layout.SharedFields {
		sharedIDs[i] = addField(field.Name, field.Type, field.Offset)
	}
	tagID := addField("root.tag", "u32", layout.SlotsOffset+layout.SlotTagOffset)
	frameIDs := make([]int, len(rootLayout.Fields))
	for i, field := range rootLayout.Fields {
		frameIDs[i] = addField("root."+field.Name, field.Type, layout.SlotsOffset+layout.SlotDataOffset+field.Offset)
	}
	b := &lirBuilder{fn: LIRFunction{Identity: layout.Identity + "#activation-init", Name: a.Name + "$activation_init", Result: "void", Source: a.SourceSpan, Activation: meta}, names: map[string]lirBinding{}, frameParam: 0}
	b.fn.Params = append(b.fn.Params, LIRValue{ID: 0, Type: LIRType("ptr<activation:" + layout.Identity + ">")})
	b.nextValue = 1
	b.newBlock()
	sharedParams := make([]int, len(a.StateEnvironment.Fields))
	for i, field := range a.StateEnvironment.Fields {
		id := b.nextValue
		b.nextValue++
		b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: layout.SharedFields[i].Type})
		b.names[field.Name] = lirBinding{value: id, typ: field.Type}
		sharedParams[i] = id
	}
	store := func(fieldID, value int, span Span) {
		field := meta.InitFields[fieldID]
		addr := b.emit(LIRInstruction{Op: "activation_address", Result: 0, Type: LIRType("ptr<" + string(field.Type) + ">"), Args: []int{b.frameParam}, Slot: -1, FrameField: fieldID, FrameOffset: field.Offset, Source: span})
		b.emit(LIRInstruction{Op: "store", Result: -1, Type: field.Type, Args: []int{addr, value}, Slot: -1, Source: span})
	}
	store(doneID, b.constant("bool", "false", a.SourceSpan), a.SourceSpan)
	for i, field := range a.StateEnvironment.Fields {
		store(sharedIDs[i], sharedParams[i], field.SourceSpan)
	}
	store(frameIDs[0], b.constant("u32", "0", root.SourceSpan), root.SourceSpan)
	for i, field := range root.Fields {
		var value int
		if field.Initializer != nil {
			v, typ, err := b.expr(field.Initializer)
			if err != nil {
				return LIRFunction{}, fmt.Errorf("EVT2_ACTIVATION_INITIALIZER %s: %w", field.Name, err)
			}
			if typ != rootLayout.Fields[i+1].Type {
				return LIRFunction{}, fmt.Errorf("EVT2_ACTIVATION_INITIALIZER_TYPE %s", field.Name)
			}
			value = v
		} else {
			zero := "0"
			if rootLayout.Fields[i+1].Type == "bool" {
				zero = "false"
			}
			value = b.constant(rootLayout.Fields[i+1].Type, zero, field.SourceSpan)
		}
		store(frameIDs[i+1], value, field.SourceSpan)
	}
	store(tagID, b.constant("u32", strconv.FormatUint(uint64(layout.RootTag), 10), root.SourceSpan), root.SourceSpan)
	// Publishing depth last makes the initialized root the first live activation.
	store(depthID, b.constant("u32", "1", root.SourceSpan), root.SourceSpan)
	b.terminate(LIRTerminator{Op: "return", Source: a.SourceSpan})
	return b.fn, nil
}

func verifyLIRActivationFunction(f LIRFunction) error {
	m := f.Activation
	if m == nil {
		return nil
	}
	l := m.Layout
	if m.Role == "step" {
		return verifyLIRActivationStep(f)
	}
	if f.Machine != nil || m.Role != "init" || f.Result != "void" || len(f.Params) == 0 || f.Params[0].Type != LIRType("ptr<activation:"+l.Identity+">") || len(f.Blocks) != 1 || f.Blocks[0].Term.Op != "return" || l.Capacity != evt1MachineStackCapacity || l.MaxFrameSize <= 0 || l.MaxFrameAlign <= 0 || l.SlotDataOffset%l.MaxFrameAlign != 0 || l.SlotSize < l.SlotDataOffset+l.MaxFrameSize || l.Size < l.SlotsOffset+l.Capacity*l.SlotSize {
		return fmt.Errorf("LIR_BAD_ACTIVATION_FRAME %s", f.Name)
	}
	if err := verifyActivationLayout(l); err != nil {
		return err
	}
	if len(m.InitFields) < 4 || m.InitFields[0].Name != "depth" || m.InitFields[0].Offset != l.DepthOffset || m.InitFields[0].Type != "u32" || m.InitFields[1].Name != "completed" || m.InitFields[1].Offset != l.DoneOffset || m.InitFields[1].Type != "bool" {
		return fmt.Errorf("LIR_BAD_ACTIVATION_INIT_FIELDS %s", f.Name)
	}
	var root *ActivationMachineLayout
	for i := range l.Machines {
		if l.Machines[i].Tag == l.RootTag {
			root = &l.Machines[i]
			break
		}
	}
	if root == nil || len(m.InitFields) != 2+len(l.SharedFields)+1+len(root.Fields) {
		return fmt.Errorf("LIR_BAD_ACTIVATION_INIT_FIELDS %s", f.Name)
	}
	for i, field := range l.SharedFields {
		actual := m.InitFields[2+i]
		if actual.Name != field.Name || actual.Type != field.Type || actual.Offset != field.Offset {
			return fmt.Errorf("LIR_BAD_ACTIVATION_INIT_FIELDS %s", f.Name)
		}
	}
	tagID := 2 + len(l.SharedFields)
	if m.InitFields[tagID].Name != "root.tag" || m.InitFields[tagID].Type != "u32" || m.InitFields[tagID].Offset != l.SlotsOffset+l.SlotTagOffset {
		return fmt.Errorf("LIR_BAD_ACTIVATION_INIT_FIELDS %s", f.Name)
	}
	for i, field := range root.Fields {
		actual := m.InitFields[tagID+1+i]
		if actual.Name != "root."+field.Name || actual.Type != field.Type || actual.Offset != l.SlotsOffset+l.SlotDataOffset+field.Offset {
			return fmt.Errorf("LIR_BAD_ACTIVATION_INIT_FIELDS %s", f.Name)
		}
	}
	for i, field := range m.InitFields {
		if field.ID != i || field.Size <= 0 || field.Alignment <= 0 || field.Offset < 0 || field.Offset%field.Alignment != 0 || field.Offset+field.Size > l.Size || (field.Type != "bool" && lirWidth(field.Type) != field.Size) {
			return fmt.Errorf("LIR_BAD_ACTIVATION_FIELD %s field=%d", f.Name, i)
		}
	}
	address := map[int]int{}
	constants := map[int]string{}
	stores := make([]bool, len(m.InitFields))
	lastStore := -1
	for _, in := range f.Blocks[0].Instructions {
		switch in.Op {
		case "activation_address":
			address[in.Result] = in.FrameField
		case "const":
			constants[in.Result] = in.Literal
		case "store":
			if len(in.Args) == 2 {
				if id, ok := address[in.Args[0]]; ok && id >= 0 && id < len(stores) {
					stores[id], lastStore = true, id
					if id == 0 && constants[in.Args[1]] != "1" || id == 1 && constants[in.Args[1]] != "false" {
						return fmt.Errorf("LIR_BAD_ACTIVATION_INITIAL_VALUE %s", f.Name)
					}
					if id == tagID && constants[in.Args[1]] != strconv.FormatUint(uint64(l.RootTag), 10) || id == tagID+1 && constants[in.Args[1]] != "0" {
						return fmt.Errorf("LIR_BAD_ACTIVATION_INITIAL_VALUE %s", f.Name)
					}
				}
			}
		}
	}
	for i, stored := range stores {
		if !stored {
			return fmt.Errorf("LIR_MISSING_ACTIVATION_INIT field=%d", i)
		}
	}
	if lastStore != 0 {
		return fmt.Errorf("LIR_ACTIVATION_DEPTH_NOT_LAST %s", f.Name)
	}
	return nil
}

func verifyActivationLayout(l ActivationStackLayout) error {
	if l.Identity == "" || l.Capacity != evt1MachineStackCapacity || len(l.Machines) == 0 || l.DepthOffset != 0 || l.DoneOffset != 4 || l.SlotTagOffset != 0 {
		return fmt.Errorf("LIR_BAD_ACTIVATION_FRAME %s", l.Identity)
	}
	maxSize, maxAlign := 0, 1
	tags := map[uint32]bool{}
	root := false
	for _, machine := range l.Machines {
		if machine.Identity == "" || machine.Name == "" || tags[machine.Tag] || machine.Size <= 0 || machine.Alignment <= 0 || machine.Size%machine.Alignment != 0 || len(machine.Fields) == 0 || len(machine.States) == 0 {
			return fmt.Errorf("LIR_BAD_ACTIVATION_MACHINE %s", machine.Name)
		}
		tags[machine.Tag] = true
		if machine.Tag == l.RootTag {
			root = true
		}
		end := 0
		for i, field := range machine.Fields {
			if field.ID != i || field.Alignment <= 0 || field.Offset != evt1AlignUp(end, field.Alignment) || field.Size != lirWidth(field.Type) && !(field.Type == "bool" && field.Size == 1) || field.Offset+field.Size > machine.Size {
				return fmt.Errorf("LIR_BAD_ACTIVATION_MACHINE_FIELD %s.%d", machine.Name, i)
			}
			end = field.Offset + field.Size
		}
		if machine.Fields[0].Name != "current_state" || machine.Fields[0].Type != "u32" || machine.Fields[0].Offset != 0 {
			return fmt.Errorf("LIR_BAD_ACTIVATION_MACHINE_STATE %s", machine.Name)
		}
		for i, state := range machine.States {
			if state.ID != i || state.Name == "" {
				return fmt.Errorf("LIR_BAD_ACTIVATION_STATE %s.%d", machine.Name, i)
			}
		}
		maxSize = max(maxSize, machine.Size)
		maxAlign = max(maxAlign, machine.Alignment)
	}
	if !root || l.MaxFrameSize != maxSize || l.MaxFrameAlign != maxAlign || l.SlotAlign != max(4, maxAlign) || l.SlotDataOffset != evt1AlignUp(4, maxAlign) || l.SlotSize != evt1AlignUp(l.SlotDataOffset+maxSize, l.SlotAlign) || l.Alignment != l.SlotAlign {
		return fmt.Errorf("LIR_BAD_ACTIVATION_SLOT %s", l.Identity)
	}
	sharedEnd := 5
	for i, field := range l.SharedFields {
		if field.ID != i || field.Alignment <= 0 || field.Offset != evt1AlignUp(sharedEnd, field.Alignment) || field.Size != lirWidth(field.Type) && !(field.Type == "bool" && field.Size == 1) {
			return fmt.Errorf("LIR_BAD_ACTIVATION_SHARED_FIELD %s.%d", l.Identity, i)
		}
		sharedEnd = field.Offset + field.Size
	}
	if l.SlotsOffset != evt1AlignUp(sharedEnd, l.SlotAlign) || l.Size != evt1AlignUp(l.SlotsOffset+l.Capacity*l.SlotSize, l.Alignment) {
		return fmt.Errorf("LIR_BAD_ACTIVATION_FRAME_SIZE %s", l.Identity)
	}
	return nil
}
