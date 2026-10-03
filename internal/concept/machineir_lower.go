package concept

import (
	"fmt"
	"strings"
)

// LowerLirToAmd64Machine selects Win64 AMD64 operations from verified LIR.
// Extra blocks make every retained failure edge explicit in the machine CFG.
func LowerLirToAmd64Machine(lir LIRModule) (MachineModule, error) {
	if err := VerifyLIR(lir); err != nil {
		return MachineModule{}, err
	}
	out := MachineModule{}
	for _, lf := range lir.Functions {
		b := machineBuilder{lir: lf, values: map[int]MachineOperand{}, types: map[int]LIRType{}, regions: map[int]string{}, frameOffsets: map[int]int{}, flags: map[int]machineCondition{}}
		b.fn = MachineFunction{Identity: lf.Identity, Name: lf.Name, Target: "amd64-windows", ABI: "win64", Result: lf.Result, Source: lf.Source, Facts: append([]string(nil), lf.Facts...), Decisions: append([]string(nil), lf.Decisions...), Frame: MachineFrame{Alignment: 16}}
		for _, lb := range lf.Blocks {
			b.fn.Blocks = append(b.fn.Blocks, MachineBlock{ID: lb.ID, LIRBlock: lb.ID})
		}
		for _, p := range lf.Params {
			a, e := Win64Argument(len(b.fn.Args), p.Type)
			if e != nil {
				return MachineModule{}, fmt.Errorf("%s: %w", lf.Name, e)
			}
			v := b.vreg(a.Width, a.Indirect)
			if strings.HasPrefix(string(p.Type), "ptr<") {
				b.fn.VRegs[v.ID].Address = true
			}
			a.VReg = v.ID
			b.fn.Args = append(b.fn.Args, a)
			b.values[p.ID] = mv(v.ID, a.Width)
			b.types[p.ID] = p.Type
			var src MachineOperand
			if a.OnStack {
				src = MachineOperand{Kind: "mem", Base: -1, BaseSlot: -1, Index: -1, Scale: 1, Disp: a.StackOffset, Width: a.Width, Region: "incoming-arg"}
			} else {
				src = mp(a.Register, a.Width)
			}
			b.emit(0, MachineInstruction{Op: "MOV", Dst: mv(v.ID, a.Width), Src: []MachineOperand{src}, Width: a.Width, LIRBlock: 0, LIRInstruction: -1, FlagsDef: -1, FlagsUse: -1})
		}
		for _, s := range lf.Slots {
			size := machineScalarWidth(s.Type)
			indirect := false
			base := -1
			if s.Extent > 0 {
				size = s.Extent * machineScalarWidth(s.Element)
				if s.ParamValue >= 0 {
					indirect = true
					base = b.values[s.ParamValue].ID
				}
			}
			if size <= 0 {
				return MachineModule{}, fmt.Errorf("MIR_UNSUPPORTED_SLOT %s", s.Type)
			}
			align := size
			if align > 8 {
				align = 8
			}
			if align < 1 {
				align = 1
			}
			b.fn.Slots = append(b.fn.Slots, MachineStackSlot{ID: s.ID, Size: size, Align: align, Source: s.Name, IncomingIndirect: indirect, BaseVReg: base})
			if !indirect {
				b.fn.Frame.LocalSize = (b.fn.Frame.LocalSize+align-1)/align*align + size
			}
		}
		for _, lb := range lf.Blocks {
			current := lb.ID
			for ii, in := range lb.Instructions {
				ctx := MachineInstruction{LIRBlock: lb.ID, LIRInstruction: ii, Source: in.Source, Decision: in.Decision, Facts: append([]string(nil), in.Facts...), FlagsDef: -1, FlagsUse: -1}
				get := func(i int) (MachineOperand, error) {
					if i >= len(in.Args) {
						return MachineOperand{}, fmt.Errorf("MIR_MISSING_OPERAND %s", in.Op)
					}
					x, ok := b.values[in.Args[i]]
					if !ok {
						return MachineOperand{}, fmt.Errorf("MIR_UNDEFINED_VALUE v%d", in.Args[i])
					}
					return x, nil
				}
				w := machineScalarWidth(in.Type)
				switch in.Op {
				case "call":
					ctx.Op = "CALL"
					call := MachineCall{Kind: "direct", Target: in.CallTarget, Convention: in.CallABI, Result: string(in.Type)}
					for i, id := range in.Args {
						value, err := get(i)
						if err != nil {
							return MachineModule{}, err
						}
						if value.Kind != "vreg" {
							return MachineModule{}, fmt.Errorf("MIR_CALL_ARGUMENT_NOT_VIRTUAL v%d", id)
						}
						call.Arguments = append(call.Arguments, MachineCallArgument{Value: value.ID, Type: string(in.ArgTypes[i])})
					}
					ctx.Calls = []MachineCall{call}
					if in.Type != "void" {
						v := b.vreg(w, false)
						ctx.Dst, ctx.Width = mv(v.ID, w), w
						b.bind(in.Result, in.Type, ctx.Dst)
					}
					b.emit(current, ctx)
				case "const":
					imm, e := machineImmediate(in.Literal, w, machineSigned(in.Type))
					if e != nil {
						return MachineModule{}, e
					}
					v := b.vreg(w, false)
					ctx.Op = "MOV"
					ctx.Dst = mv(v.ID, w)
					ctx.Src = []MachineOperand{imm}
					ctx.Width = w
					b.emit(current, ctx)
					b.bind(in.Result, in.Type, ctx.Dst)
				case "machine_result":
					tag := map[string]string{"Active": "0", "Yielded": "1", "Completed": "2"}[in.Literal]
					imm, e := machineImmediate(tag, 4, false)
					if e != nil {
						return MachineModule{}, e
					}
					v := b.vreg(4, false)
					ctx.Op, ctx.Dst, ctx.Src, ctx.Width = "MOV", mv(v.ID, 4), []MachineOperand{imm}, 4
					b.emit(current, ctx)
					b.bind(in.Result, in.Type, ctx.Dst)
				case "frame_field_address", "activation_address":
					base, e := get(0)
					if e != nil {
						return MachineModule{}, e
					}
					// A frame field is a fixed displacement from the caller's
					// pointer. Fold it into the consuming memory operand.
					b.bind(in.Result, in.Type, base)
					b.frameOffsets[in.Result] = in.FrameOffset
					b.regions[in.Result] = "machine-frame"
				case "load_slot":
					v := b.vreg(w, false)
					ctx.Op = "LOAD"
					ctx.Dst = mv(v.ID, w)
					ctx.Src = []MachineOperand{ms(in.Slot, w)}
					ctx.Width = w
					b.emit(current, ctx)
					b.bind(in.Result, in.Type, ctx.Dst)
				case "store_slot":
					x, e := get(0)
					if e != nil {
						return MachineModule{}, e
					}
					ctx.Op = "STORE"
					ctx.Dst = ms(in.Slot, w)
					ctx.Src = []MachineOperand{x}
					ctx.Width = w
					b.emit(current, ctx)
				case "checked_add", "checked_sub", "checked_mul", "add", "sub", "mul":
					left, e := get(0)
					if e != nil {
						return MachineModule{}, e
					}
					right, e := get(1)
					if e != nil {
						return MachineModule{}, e
					}
					v := b.vreg(w, false)
					dst := mv(v.ID, w)
					b.emit(current, MachineInstruction{Op: "MOV", Dst: dst, Src: []MachineOperand{left}, Width: w, LIRBlock: lb.ID, LIRInstruction: ii, Source: in.Source, FlagsDef: -1, FlagsUse: -1})
					ctx.Op = map[string]string{"checked_add": "ADD", "checked_sub": "SUB", "checked_mul": "IMUL", "add": "ADD", "sub": "SUB", "mul": "IMUL"}[in.Op]
					if !machineSigned(in.Type) && (in.Op == "checked_mul" || in.Op == "mul") {
						ctx.Op = "UMUL"
					}
					ctx.Dst = dst
					ctx.Src = []MachineOperand{right}
					ctx.Width = w
					ctx.FlagsDef = b.flag()
					b.emit(current, ctx)
					b.bind(in.Result, in.Type, dst)
					if len(in.Op) > 8 && in.Op[:8] == "checked_" {
						cond := "O"
						if !machineSigned(in.Type) {
							cond = "C"
						}
						current = b.failureEdge(current, lb.ID, ctx.FlagsDef, cond, "overflow", in.Source)
					}
				case "eq", "ne", "lt", "le", "gt", "ge":
					left, e := get(0)
					if e != nil {
						return MachineModule{}, e
					}
					right, e := get(1)
					if e != nil {
						return MachineModule{}, e
					}
					cond := machineCompareCondition(in.Op, b.types[in.Args[0]])
					ctx.Op = "CMP"
					ctx.Src = []MachineOperand{left, right}
					ctx.Width = left.Width
					ctx.FlagsDef = b.flag()
					b.emit(current, ctx)
					if ii == len(lb.Instructions)-1 && lb.Term.Op == "branch" && lb.Term.Value == in.Result {
						b.flags[in.Result] = machineCondition{flag: ctx.FlagsDef, cond: cond}
					} else {
						v := b.vreg(1, false)
						set := ctx
						set.Op = "SETCC"
						set.Dst = mv(v.ID, 1)
						set.Src = nil
						set.Width = 1
						set.FlagsDef = -1
						set.FlagsUse = ctx.FlagsDef
						set.Cond = cond
						b.emit(current, set)
						b.bind(in.Result, "bool", set.Dst)
					}
				case "check_index":
					idx, e := get(0)
					if e != nil {
						return MachineModule{}, e
					}
					imm, e := machineImmediate(fmt.Sprint(in.Extent), idx.Width, false)
					if e != nil {
						return MachineModule{}, e
					}
					ctx.Op = "CMP"
					ctx.Src = []MachineOperand{idx, imm}
					ctx.Width = idx.Width
					ctx.FlagsDef = b.flag()
					b.emit(current, ctx)
					current = b.failureEdge(current, lb.ID, ctx.FlagsDef, "AE", "bounds", in.Source)
				case "index_address":
					indexArg := 0
					baseID := -1
					region := ""
					disp := in.FrameOffset
					if in.Slot == -1 {
						base, e := get(0)
						if e != nil {
							return MachineModule{}, e
						}
						baseID = base.ID
						indexArg = 1
						region = b.regions[in.Args[0]]
						disp += b.frameOffsets[in.Args[0]]
					} else {
						slot := b.fn.Slots[in.Slot]
						if !slot.IncomingIndirect {
							return MachineModule{}, fmt.Errorf("MIR_UNSUPPORTED_ARRAY_BASE s%d", in.Slot)
						}
						baseID = slot.BaseVReg
						region = fmt.Sprintf("s%d", in.Slot)
					}
					idx, e := get(indexArg)
					if e != nil {
						return MachineModule{}, e
					}
					scale := in.Stride
					if scale != 1 && scale != 2 && scale != 4 && scale != 8 {
						factor := b.vreg(idx.Width, false)
						imm, e := machineImmediate(fmt.Sprint(scale), idx.Width, false)
						if e != nil {
							return MachineModule{}, e
						}
						b.emit(current, MachineInstruction{Op: "MOV", Dst: mv(factor.ID, idx.Width), Src: []MachineOperand{imm}, Width: idx.Width, LIRBlock: lb.ID, LIRInstruction: ii, FlagsDef: -1, FlagsUse: -1})
						product := b.vreg(idx.Width, false)
						b.emit(current, MachineInstruction{Op: "MOV", Dst: mv(product.ID, idx.Width), Src: []MachineOperand{idx}, Width: idx.Width, LIRBlock: lb.ID, LIRInstruction: ii, FlagsDef: -1, FlagsUse: -1})
						b.emit(current, MachineInstruction{Op: "IMUL", Dst: mv(product.ID, idx.Width), Src: []MachineOperand{mv(factor.ID, idx.Width)}, Width: idx.Width, LIRBlock: lb.ID, LIRInstruction: ii, FlagsDef: b.flag(), FlagsUse: -1})
						idx = mv(product.ID, idx.Width)
						scale = 1
					}
					mem := MachineOperand{Kind: "mem", Base: baseID, BaseSlot: -1, Index: idx.ID, Scale: scale, Disp: disp, Width: 8, Region: region}
					v := b.vreg(8, true)
					ctx.Op = "LEA"
					ctx.Dst = mv(v.ID, 8)
					ctx.Src = []MachineOperand{mem}
					ctx.Width = 8
					b.emit(current, ctx)
					b.bind(in.Result, in.Type, ctx.Dst)
					b.regions[in.Result] = mem.Region
				case "load", "store":
					addr, e := get(0)
					if e != nil {
						return MachineModule{}, e
					}
					mem := MachineOperand{Kind: "mem", Base: addr.ID, BaseSlot: -1, Index: -1, Scale: 1, Disp: b.frameOffsets[in.Args[0]], Width: w, Region: b.regions[in.Args[0]]}
					if in.Op == "load" {
						v := b.vreg(w, false)
						ctx.Op = "LOAD"
						ctx.Dst = mv(v.ID, w)
						ctx.Src = []MachineOperand{mem}
						ctx.Width = w
						b.emit(current, ctx)
						b.bind(in.Result, in.Type, ctx.Dst)
					} else {
						value, e := get(1)
						if e != nil {
							return MachineModule{}, e
						}
						ctx.Op = "STORE"
						ctx.Dst = mem
						ctx.Src = []MachineOperand{value}
						ctx.Width = w
						b.emit(current, ctx)
					}
				default:
					return MachineModule{}, fmt.Errorf("MIR_UNSUPPORTED_LIR_OP %s", in.Op)
				}
			}
			switch lb.Term.Op {
			case "jump":
				b.fn.Blocks[current].Term = MachineTerminator{Op: "JMP", True: lb.Term.True, FlagsUse: -1, Source: lb.Term.Source}
			case "branch":
				c, ok := b.flags[lb.Term.Value]
				if !ok {
					x, exists := b.values[lb.Term.Value]
					if !exists {
						return MachineModule{}, fmt.Errorf("MIR_UNDEFINED_BRANCH v%d", lb.Term.Value)
					}
					flag := b.flag()
					b.emit(current, MachineInstruction{Op: "TEST", Src: []MachineOperand{x, x}, Width: x.Width, FlagsDef: flag, FlagsUse: -1, LIRBlock: lb.ID, LIRInstruction: -1, Source: lb.Term.Source})
					c = machineCondition{flag: flag, cond: "NE"}
				}
				b.fn.Blocks[current].Term = MachineTerminator{Op: "JCC", True: lb.Term.True, False: lb.Term.False, Cond: c.cond, FlagsUse: c.flag, Source: lb.Term.Source}
			case "return":
				if lf.Result != "void" {
					x, ok := b.values[lb.Term.Value]
					if !ok {
						return MachineModule{}, fmt.Errorf("MIR_UNDEFINED_RETURN v%d", lb.Term.Value)
					}
					r, w, e := Win64Return(lf.Result)
					if e != nil {
						return MachineModule{}, e
					}
					b.emit(current, MachineInstruction{Op: "MOV", Dst: mp(r, w), Src: []MachineOperand{x}, Width: w, FlagsDef: -1, FlagsUse: -1, LIRBlock: lb.ID, LIRInstruction: -1, Source: lb.Term.Source})
				}
				b.fn.Blocks[current].Term = MachineTerminator{Op: "RET", FlagsUse: -1, Source: lb.Term.Source}
			case "trap":
				b.emit(current, MachineInstruction{Op: "TRAP", Width: 0, FlagsDef: -1, FlagsUse: -1, LIRBlock: lb.ID, LIRInstruction: -1, Source: lb.Term.Source, Decision: lb.Term.Reason})
				b.fn.Blocks[current].Term = MachineTerminator{Op: "TRAP", FlagsUse: -1, Source: lb.Term.Source}
			}
		}
		if err := VerifyMachineFunction(b.fn); err != nil {
			return MachineModule{}, fmt.Errorf("%s: %w", lf.Name, err)
		}
		out.Functions = append(out.Functions, b.fn)
	}
	return out, nil
}
func GenerateMachineIR(module Module) (MachineModule, error) {
	lir, e := GenerateLIR(module)
	if e != nil {
		return MachineModule{}, e
	}
	return LowerLirToAmd64Machine(lir)
}

type machineCondition struct {
	flag int
	cond string
}
type machineBuilder struct {
	lir          LIRFunction
	fn           MachineFunction
	values       map[int]MachineOperand
	types        map[int]LIRType
	regions      map[int]string
	frameOffsets map[int]int
	flags        map[int]machineCondition
	nextFlag     int
}

func (b *machineBuilder) vreg(w int, address bool) MachineVReg {
	v := MachineVReg{ID: len(b.fn.VRegs), Width: w, Address: address}
	b.fn.VRegs = append(b.fn.VRegs, v)
	return v
}
func (b *machineBuilder) flag() int                                { id := b.nextFlag; b.nextFlag++; return id }
func (b *machineBuilder) bind(id int, t LIRType, o MachineOperand) { b.values[id] = o; b.types[id] = t }
func (b *machineBuilder) emit(block int, in MachineInstruction) {
	b.fn.Blocks[block].Instructions = append(b.fn.Blocks[block].Instructions, in)
}
func (b *machineBuilder) newBlock(lirBlock int) int {
	id := len(b.fn.Blocks)
	b.fn.Blocks = append(b.fn.Blocks, MachineBlock{ID: id, LIRBlock: lirBlock})
	return id
}
func (b *machineBuilder) failureEdge(current, lirBlock, flag int, cond, reason string, source Span) int {
	trap := b.newBlock(lirBlock)
	next := b.newBlock(lirBlock)
	b.fn.Blocks[current].Term = MachineTerminator{Op: "JCC", True: trap, False: next, Cond: cond, FlagsUse: flag, Source: source}
	b.fn.Blocks[trap].Instructions = []MachineInstruction{{Op: "TRAP", Width: 0, FlagsDef: -1, FlagsUse: -1, LIRBlock: lirBlock, LIRInstruction: -1, Source: source, Decision: reason}}
	b.fn.Blocks[trap].Term = MachineTerminator{Op: "TRAP", FlagsUse: -1, Source: source}
	return next
}
func machineCompareCondition(op string, t LIRType) string {
	signed := machineSigned(t)
	switch op {
	case "eq":
		return "E"
	case "ne":
		return "NE"
	case "lt":
		if signed {
			return "L"
		}
		return "B"
	case "le":
		if signed {
			return "LE"
		}
		return "BE"
	case "gt":
		if signed {
			return "G"
		}
		return "A"
	case "ge":
		if signed {
			return "GE"
		}
		return "AE"
	}
	return "invalid"
}
