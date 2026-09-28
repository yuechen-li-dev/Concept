package concept

import (
	"fmt"
	"strconv"
	"strings"
)

// MachineIR is AMD64-specific, but retains virtual registers and abstract frames.
// No opcode below is an encoded instruction or an allocated register assignment.
type MachineModule struct{ Functions []MachineFunction }
type MachineReg int

const (
	RAX MachineReg = iota
	RBX
	RCX
	RDX
	RSI
	RDI
	RBP
	RSP
	R8
	R9
	R10
	R11
	R12
	R13
	R14
	R15
)

var machineRegNames = [...]string{"rax", "rbx", "rcx", "rdx", "rsi", "rdi", "rbp", "rsp", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15"}

func (r MachineReg) String() string {
	if r < 0 || int(r) >= len(machineRegNames) {
		return "invalid"
	}
	return machineRegNames[r]
}
func (r MachineReg) Valid() bool { return r >= 0 && int(r) < len(machineRegNames) }

type MachineRegisterWrite struct {
	Parent            MachineReg
	Width             int
	ZeroExtendsParent bool
}

func (r MachineReg) WriteEffect(width int) (MachineRegisterWrite, error) {
	if !r.Valid() || !machineWidthValid(width) {
		return MachineRegisterWrite{}, fmt.Errorf("MIR_BAD_REGISTER_VIEW")
	}
	return MachineRegisterWrite{Parent: r, Width: width, ZeroExtendsParent: width == 4}, nil
}
func (r MachineReg) Alias(width int) string {
	if !r.Valid() {
		return "invalid"
	}
	aliases := [16][4]string{
		{"al", "ax", "eax", "rax"}, {"bl", "bx", "ebx", "rbx"}, {"cl", "cx", "ecx", "rcx"}, {"dl", "dx", "edx", "rdx"},
		{"sil", "si", "esi", "rsi"}, {"dil", "di", "edi", "rdi"}, {"bpl", "bp", "ebp", "rbp"}, {"spl", "sp", "esp", "rsp"},
		{"r8b", "r8w", "r8d", "r8"}, {"r9b", "r9w", "r9d", "r9"}, {"r10b", "r10w", "r10d", "r10"}, {"r11b", "r11w", "r11d", "r11"},
		{"r12b", "r12w", "r12d", "r12"}, {"r13b", "r13w", "r13d", "r13"}, {"r14b", "r14w", "r14d", "r14"}, {"r15b", "r15w", "r15d", "r15"},
	}
	switch width {
	case 1:
		return aliases[r][0]
	case 2:
		return aliases[r][1]
	case 4:
		return aliases[r][2]
	case 8:
		return aliases[r][3]
	}
	return "invalid"
}

var Win64CallerSaved = []MachineReg{RAX, RCX, RDX, R8, R9, R10, R11}
var Win64CalleeSaved = []MachineReg{RBX, RBP, RSI, RDI, R12, R13, R14, R15}

type MachineVReg struct {
	ID      int
	Width   int
	Address bool
}
type MachineStackSlot struct {
	ID               int
	Size             int
	Align            int
	Source           string
	IncomingIndirect bool
	BaseVReg         int
}
type MachineFrame struct {
	LocalSize   int
	Alignment   int
	ShadowSpace int
	HasCalls    bool
}

// Win64CallFrameBytes reserves locals plus the required outgoing home area.
// At function entry RSP is 8 mod 16, so a nonleaf subtraction is 8 mod 16.
func Win64CallFrameBytes(localSize int, hasCalls bool) (int, error) {
	if localSize < 0 {
		return 0, fmt.Errorf("MIR_BAD_FRAME_SIZE")
	}
	if !hasCalls {
		return localSize, nil
	}
	n := localSize + 32
	return (n+7)/16*16 + 8, nil
}

type MachineArg struct {
	Index       int
	Type        LIRType
	Width       int
	Indirect    bool
	Register    MachineReg
	StackOffset int
	OnStack     bool
	VReg        int
}

// Win64 arguments use four positional register slots. A fifth argument starts
// at [entry RSP+40]; large aggregates are passed by pointer to caller storage.
func Win64Argument(index int, typ LIRType) (MachineArg, error) {
	if index < 0 {
		return MachineArg{}, fmt.Errorf("MIR_BAD_ARG_INDEX")
	}
	w, indirect, err := machineABIWidth(typ)
	if err != nil {
		return MachineArg{}, err
	}
	a := MachineArg{Index: index, Type: typ, Width: w, Indirect: indirect, VReg: -1}
	if index < 4 {
		a.Register = [...]MachineReg{RCX, RDX, R8, R9}[index]
	} else {
		a.OnStack = true
		a.StackOffset = 40 + 8*(index-4)
	}
	return a, nil
}
func Win64Return(typ LIRType) (MachineReg, int, error) {
	if typ == "void" {
		return RAX, 0, nil
	}
	w := machineScalarWidth(typ)
	if w == 0 {
		return 0, 0, fmt.Errorf("MIR_UNSUPPORTED_RETURN %s", typ)
	}
	return RAX, w, nil
}
func machineABIWidth(t LIRType) (int, bool, error) {
	if w := machineScalarWidth(t); w != 0 {
		return w, false, nil
	}
	var count int
	var elem string
	if _, e := fmt.Sscanf(string(t), "[%d]%s", &count, &elem); e == nil && count > 0 && machineScalarWidth(LIRType(elem)) > 0 {
		size := count * machineScalarWidth(LIRType(elem))
		if size == 1 || size == 2 || size == 4 || size == 8 {
			return size, false, nil
		}
		return 8, true, nil
	}
	return 0, false, fmt.Errorf("MIR_UNSUPPORTED_ABI_TYPE %s", t)
}
func machineScalarWidth(t LIRType) int {
	if t == "bool" {
		return 1
	}
	return lirWidth(t)
}
func machineSigned(t LIRType) bool { return strings.HasPrefix(string(t), "i") }

type MachineOperand struct {
	Kind     string // vreg, preg, imm, slot, mem, block
	ID       int
	Width    int
	Signed   bool
	Literal  string
	Base     int
	BaseSlot int
	Index    int
	Scale    int
	Disp     int
	Region   string
}

func mv(id, w int) MachineOperand { return MachineOperand{Kind: "vreg", ID: id, Width: w} }
func mp(r MachineReg, w int) MachineOperand {
	return MachineOperand{Kind: "preg", ID: int(r), Width: w}
}
func ms(id, w int) MachineOperand { return MachineOperand{Kind: "slot", ID: id, Width: w} }
func mb(id int) MachineOperand    { return MachineOperand{Kind: "block", ID: id} }
func (o MachineOperand) String() string {
	switch o.Kind {
	case "vreg":
		return fmt.Sprintf("v%d:%d", o.ID, o.Width*8)
	case "preg":
		return MachineReg(o.ID).Alias(o.Width)
	case "imm":
		return fmt.Sprintf("%s:%d", o.Literal, o.Width*8)
	case "slot":
		return fmt.Sprintf("s%d:%d", o.ID, o.Width*8)
	case "block":
		return fmt.Sprintf("b%d", o.ID)
	case "mem":
		base := fmt.Sprintf("v%d", o.Base)
		if o.BaseSlot >= 0 {
			base = fmt.Sprintf("s%d", o.BaseSlot)
		}
		if o.Index >= 0 {
			base += fmt.Sprintf("+v%d*%d", o.Index, o.Scale)
		}
		if o.Disp != 0 {
			base += fmt.Sprintf("%+d", o.Disp)
		}
		return fmt.Sprintf("[%s]:%d", base, o.Width*8)
	}
	return "<invalid>"
}

type MachineInstruction struct {
	Op             string
	Dst            MachineOperand
	Src            []MachineOperand
	Width          int
	FlagsDef       int
	FlagsUse       int
	Cond           string
	LIRBlock       int
	LIRInstruction int
	Source         Span
	Decision       string
	Facts          []string
}
type MachineTerminator struct {
	Op       string
	True     int
	False    int
	Cond     string
	FlagsUse int
	Source   Span
}
type MachineBlock struct {
	ID           int
	LIRBlock     int
	Instructions []MachineInstruction
	Term         MachineTerminator
}
type MachineFunction struct {
	Identity  string
	Name      string
	Target    string
	ABI       string
	Result    LIRType
	Args      []MachineArg
	VRegs     []MachineVReg
	Slots     []MachineStackSlot
	Frame     MachineFrame
	Blocks    []MachineBlock
	Facts     []string
	Decisions []string
	Source    Span
}

func (m MachineModule) String() string {
	var b strings.Builder
	for fi, f := range m.Functions {
		if fi > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "fn %s [%s] amd64-windows win64 -> %s\n", f.Name, f.Identity, f.Result)
		for _, a := range f.Args {
			loc := MachineOperand{Kind: "preg", ID: int(a.Register), Width: a.Width}.String()
			if a.OnStack {
				loc = fmt.Sprintf("[entry rsp+%d]", a.StackOffset)
			}
			mode := ""
			if a.Indirect {
				mode = " indirect"
			}
			fmt.Fprintf(&b, "  arg %d %s%s %s -> v%d\n", a.Index, a.Type, mode, loc, a.VReg)
		}
		for _, s := range f.Slots {
			kind := "local"
			if s.IncomingIndirect {
				kind = fmt.Sprintf("incoming-v%d", s.BaseVReg)
			}
			fmt.Fprintf(&b, "  slot s%d %s size=%d align=%d %s\n", s.ID, s.Source, s.Size, s.Align, kind)
		}
		fmt.Fprintf(&b, "  frame locals=%d align=%d shadow=%d\n", f.Frame.LocalSize, f.Frame.Alignment, f.Frame.ShadowSpace)
		for _, block := range f.Blocks {
			fmt.Fprintf(&b, "b%d (lir b%d):\n", block.ID, block.LIRBlock)
			for _, in := range block.Instructions {
				fmt.Fprintf(&b, "  %s", in.Op)
				if in.Dst.Kind != "" {
					fmt.Fprintf(&b, " %s", in.Dst)
				}
				for _, s := range in.Src {
					fmt.Fprintf(&b, " %s", s)
				}
				if in.Cond != "" {
					fmt.Fprintf(&b, " %s", in.Cond)
				}
				if in.FlagsDef >= 0 {
					fmt.Fprintf(&b, " flags=f%d", in.FlagsDef)
				}
				if in.FlagsUse >= 0 {
					fmt.Fprintf(&b, " uses=f%d", in.FlagsUse)
				}
				if in.Decision != "" {
					fmt.Fprintf(&b, " [%s]", in.Decision)
				}
				b.WriteByte('\n')
			}
			t := block.Term
			fmt.Fprintf(&b, "  %s", t.Op)
			if t.Op == "JMP" {
				fmt.Fprintf(&b, " b%d", t.True)
			}
			if t.Op == "JCC" {
				fmt.Fprintf(&b, " %s f%d b%d b%d", t.Cond, t.FlagsUse, t.True, t.False)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}
func machineImmediate(lit string, w int, signed bool) (MachineOperand, error) {
	if lit == "true" {
		lit = "1"
	}
	if lit == "false" {
		lit = "0"
	}
	if !machineWidthValid(w) {
		return MachineOperand{}, fmt.Errorf("MIR_BAD_IMMEDIATE_WIDTH")
	}
	var err error
	if signed {
		_, err = strconv.ParseInt(lit, 10, w*8)
	} else {
		_, err = strconv.ParseUint(lit, 10, w*8)
	}
	if err != nil {
		return MachineOperand{}, fmt.Errorf("MIR_BAD_IMMEDIATE %s:%d", lit, w*8)
	}
	return MachineOperand{Kind: "imm", Literal: lit, Width: w, Signed: signed}, nil
}
