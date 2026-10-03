package concept

import (
	"fmt"
	"strings"
)

type MachineEffects struct{ ReadsMemory, WritesMemory, SetsFlags, ReadsFlags, Terminates, MayTrap bool }

func (in MachineInstruction) Effects() MachineEffects {
	e, _ := MachineOpcodeEffects(in.Op)
	if in.Op != "LEA" {
		for _, src := range in.Src {
			if src.Kind == "mem" || src.Kind == "slot" {
				e.ReadsMemory = true
			}
		}
		if in.Dst.Kind == "mem" || in.Dst.Kind == "slot" {
			e.WritesMemory = true
		}
	}
	return e
}

func MachineOpcodeEffects(op string) (MachineEffects, bool) {
	switch op {
	case "MOV", "LEA":
		return MachineEffects{}, true
	case "LOAD":
		return MachineEffects{ReadsMemory: true}, true
	case "STORE":
		return MachineEffects{WritesMemory: true}, true
	case "ADD", "SUB", "IMUL", "UMUL", "CMP", "TEST":
		return MachineEffects{SetsFlags: true}, true
	case "SETCC":
		return MachineEffects{ReadsFlags: true}, true
	case "TRAP":
		return MachineEffects{MayTrap: true}, true
	case "JMP", "JCC", "RET":
		return MachineEffects{Terminates: true, ReadsFlags: op == "JCC"}, true
	}
	return MachineEffects{}, false
}
func machineConditionValid(c string) bool {
	// The checked Concept BridgeSchema enum owns the accepted condition names.
	// Its generated Go table is also used by the bootstrap wire codec.
	tag, err := machineBridgeTag(bridgeTagsCondition, c)
	return err == nil && tag != 0 // Condition::None is not a FLAGS consumer.
}
func machineWidthValid(w int) bool { return w == 1 || w == 2 || w == 4 || w == 8 }
func VerifyMachineIR(m MachineModule) error {
	for _, f := range m.Functions {
		if e := VerifyMachineFunction(f); e != nil {
			return fmt.Errorf("%s: %w", f.Name, e)
		}
	}
	return nil
}
func VerifyMachineFunction(f MachineFunction) error {
	if f.Identity == "" || f.Target != "amd64-windows" || f.ABI != "win64" || len(f.Blocks) == 0 {
		return fmt.Errorf("MIR_BAD_FUNCTION")
	}
	for i, v := range f.VRegs {
		if v.ID != i || !machineWidthValid(v.Width) || v.Address && v.Width != 8 {
			return fmt.Errorf("MIR_BAD_VREG v%d", i)
		}
	}
	for i, s := range f.Slots {
		if s.ID != i || s.Size <= 0 || s.Align <= 0 || s.Align&(s.Align-1) != 0 || s.IncomingIndirect && (s.BaseVReg < 0 || s.BaseVReg >= len(f.VRegs) || !f.VRegs[s.BaseVReg].Address) {
			return fmt.Errorf("MIR_BAD_SLOT s%d", i)
		}
	}
	if f.Frame.Alignment != 16 || f.Frame.LocalSize < 0 || f.Frame.ShadowSpace != 0 && !f.Frame.HasCalls || f.Frame.HasCalls && f.Frame.ShadowSpace < 32 {
		return fmt.Errorf("MIR_BAD_FRAME")
	}
	for i, a := range f.Args {
		expect, e := Win64Argument(i, a.Type)
		if e != nil {
			return e
		}
		address := a.Indirect || strings.HasPrefix(string(a.Type), "ptr<")
		if a.Index != i || a.Width != expect.Width || a.Indirect != expect.Indirect || a.OnStack != expect.OnStack || a.Register != expect.Register || a.StackOffset != expect.StackOffset || a.VReg < 0 || a.VReg >= len(f.VRegs) || f.VRegs[a.VReg].Width != a.Width || f.VRegs[a.VReg].Address != address {
			return fmt.Errorf("MIR_BAD_ABI_ARG %d", i)
		}
	}
	check := func(o MachineOperand) error {
		switch o.Kind {
		case "vreg":
			if o.ID < 0 || o.ID >= len(f.VRegs) || o.Width != f.VRegs[o.ID].Width {
				return fmt.Errorf("MIR_BAD_VREG_OPERAND v%d", o.ID)
			}
		case "preg":
			if !MachineReg(o.ID).Valid() || !machineWidthValid(o.Width) {
				return fmt.Errorf("MIR_BAD_PREG")
			}
		case "imm":
			if !machineWidthValid(o.Width) || o.Literal == "" {
				return fmt.Errorf("MIR_BAD_IMMEDIATE")
			}
			if _, e := machineImmediate(o.Literal, o.Width, o.Signed); e != nil {
				return e
			}
		case "slot":
			if o.ID < 0 || o.ID >= len(f.Slots) || !machineWidthValid(o.Width) || o.Width > f.Slots[o.ID].Size || f.Slots[o.ID].IncomingIndirect {
				return fmt.Errorf("MIR_BAD_SLOT_OPERAND s%d", o.ID)
			}
		case "mem":
			if !machineWidthValid(o.Width) || o.Scale != 1 && o.Scale != 2 && o.Scale != 4 && o.Scale != 8 {
				return fmt.Errorf("MIR_BAD_MEMORY_SCALE")
			}
			if o.Region == "incoming-arg" {
				if o.Base != -1 || o.BaseSlot != -1 || o.Index != -1 || o.Disp < 40 {
					return fmt.Errorf("MIR_BAD_INCOMING_MEMORY")
				}
				break
			}
			if o.BaseSlot >= 0 {
				if o.BaseSlot >= len(f.Slots) {
					return fmt.Errorf("MIR_BAD_MEMORY_SLOT")
				}
			} else if o.Base < 0 || o.Base >= len(f.VRegs) || !f.VRegs[o.Base].Address {
				return fmt.Errorf("MIR_BAD_MEMORY_BASE")
			}
			if o.Index >= 0 && (o.Index >= len(f.VRegs) || f.VRegs[o.Index].Address) {
				return fmt.Errorf("MIR_BAD_MEMORY_INDEX")
			}
		case "block":
			if o.ID < 0 || o.ID >= len(f.Blocks) {
				return fmt.Errorf("MIR_BAD_TARGET b%d", o.ID)
			}
		default:
			return fmt.Errorf("MIR_BAD_OPERAND_KIND %s", o.Kind)
		}
		return nil
	}
	initialDefs := make([]int, len(f.VRegs))
	flagDefs := map[int]bool{}
	for bi, block := range f.Blocks {
		if block.ID != bi || block.LIRBlock < 0 {
			return fmt.Errorf("MIR_BAD_BLOCK b%d", bi)
		}
		flag := -1
		for _, in := range block.Instructions {
			effects, ok := MachineOpcodeEffects(in.Op)
			if !ok || effects.Terminates {
				return fmt.Errorf("MIR_BAD_OPCODE %s", in.Op)
			}
			if in.Dst.Kind != "" {
				if e := check(in.Dst); e != nil {
					return e
				}
			}
			for _, s := range in.Src {
				if e := check(s); e != nil {
					return e
				}
			}
			if in.Width != 0 && !machineWidthValid(in.Width) {
				return fmt.Errorf("MIR_BAD_WIDTH %s", in.Op)
			}
			if effects.SetsFlags {
				if in.FlagsDef < 0 || flagDefs[in.FlagsDef] {
					return fmt.Errorf("MIR_BAD_FLAGS_DEF")
				}
				flagDefs[in.FlagsDef] = true
				flag = in.FlagsDef
			} else if in.FlagsDef >= 0 {
				return fmt.Errorf("MIR_UNEXPECTED_FLAGS_DEF")
			}
			if effects.ReadsFlags {
				if flag < 0 || in.FlagsUse != flag || !machineConditionValid(in.Cond) {
					return fmt.Errorf("MIR_BAD_FLAGS_USE")
				}
			} else if in.FlagsUse >= 0 {
				return fmt.Errorf("MIR_UNEXPECTED_FLAGS_USE")
			}
			switch in.Op {
			case "MOV":
				if len(in.Src) != 1 || in.Dst.Kind != "vreg" && in.Dst.Kind != "preg" || in.Width != in.Dst.Width || in.Src[0].Width != in.Width {
					return fmt.Errorf("MIR_BAD_MOV")
				}
			case "LOAD":
				if len(in.Src) != 1 || in.Dst.Kind != "vreg" || in.Src[0].Kind != "slot" && in.Src[0].Kind != "mem" || in.Width != in.Dst.Width || in.Src[0].Width != in.Width {
					return fmt.Errorf("MIR_BAD_LOAD")
				}
			case "STORE":
				if len(in.Src) != 1 || in.Dst.Kind != "slot" && in.Dst.Kind != "mem" || in.Width != in.Dst.Width || in.Src[0].Width != in.Width {
					return fmt.Errorf("MIR_BAD_STORE")
				}
			case "LEA":
				if len(in.Src) != 1 || in.Src[0].Kind != "mem" || in.Dst.Kind != "vreg" || in.Width != 8 || !f.VRegs[in.Dst.ID].Address {
					return fmt.Errorf("MIR_BAD_LEA")
				}
			case "ADD", "SUB", "IMUL", "UMUL":
				if len(in.Src) != 1 || in.Dst.Kind != "vreg" || in.Width != in.Dst.Width || in.Src[0].Width != in.Width {
					return fmt.Errorf("MIR_BAD_ARITHMETIC")
				}
			case "CMP", "TEST":
				if len(in.Src) != 2 || in.Dst.Kind != "" || in.Src[0].Width != in.Width || in.Src[1].Width != in.Width {
					return fmt.Errorf("MIR_BAD_COMPARE")
				}
			case "SETCC":
				if len(in.Src) != 0 || in.Dst.Kind != "vreg" || in.Width != 1 || in.Dst.Width != 1 {
					return fmt.Errorf("MIR_BAD_SETCC")
				}
			case "TRAP":
				if in.Dst.Kind != "" || len(in.Src) != 0 || in.Width != 0 {
					return fmt.Errorf("MIR_BAD_TRAP")
				}
			}
			if in.Dst.Kind == "vreg" {
				switch in.Op {
				case "MOV", "LOAD", "LEA", "SETCC":
					initialDefs[in.Dst.ID]++
				case "ADD", "SUB", "IMUL", "UMUL":
					// The two-address operation updates its earlier MOV result.
				}
			}
		}
		t := block.Term
		switch t.Op {
		case "JMP":
			if t.True < 0 || t.True >= len(f.Blocks) || t.FlagsUse >= 0 {
				return fmt.Errorf("MIR_BAD_JUMP")
			}
		case "JCC":
			if t.True < 0 || t.True >= len(f.Blocks) || t.False < 0 || t.False >= len(f.Blocks) || !machineConditionValid(t.Cond) || flag < 0 || t.FlagsUse != flag {
				return fmt.Errorf("MIR_BAD_JCC")
			}
		case "RET":
			if t.FlagsUse >= 0 {
				return fmt.Errorf("MIR_BAD_RET")
			}
			if f.Result != "void" {
				_, w, e := Win64Return(f.Result)
				if e != nil {
					return e
				}
				if len(block.Instructions) == 0 {
					return fmt.Errorf("MIR_RETURN_REGISTER")
				}
				last := block.Instructions[len(block.Instructions)-1]
				if last.Op != "MOV" || last.Dst.Kind != "preg" || last.Dst.ID != int(RAX) || last.Dst.Width != w {
					return fmt.Errorf("MIR_RETURN_REGISTER")
				}
			}
		case "TRAP":
			if len(block.Instructions) == 0 || block.Instructions[len(block.Instructions)-1].Op != "TRAP" {
				return fmt.Errorf("MIR_BAD_TRAP_TERM")
			}
		default:
			return fmt.Errorf("MIR_MISSING_TERMINATOR b%d", bi)
		}
	}
	for id, count := range initialDefs {
		if count != 1 {
			return fmt.Errorf("MIR_VREG_DEFINITION v%d count=%d", id, count)
		}
	}
	return nil
}
