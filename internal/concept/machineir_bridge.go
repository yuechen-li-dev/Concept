package concept

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// legacyMachineBridgeSchema is a deliberately small, endian-stable semantic boundary.
// Its wire layout is specified in docs/design/EVT2-MACHINEIR-BRIDGE.md.
const legacyMachineBridgeSchema = "CMIRAMD1"

const machineBridgeMaxItems = 1 << 20

var machineBridgeKinds = []string{"", "vreg", "preg", "imm", "slot", "mem", "block"}
var machineBridgeOps = []string{"", "MOV", "LOAD", "STORE", "LEA", "ADD", "SUB", "IMUL", "UMUL", "CMP", "TEST", "SETCC", "TRAP", "JMP", "JCC", "RET"}
var machineBridgeConds = []string{"", "E", "NE", "L", "LE", "G", "GE", "B", "BE", "A", "AE", "O", "C"}

func machineBridgeTag(values []string, value string) (int, error) {
	for i, v := range values {
		if v == value {
			return i, nil
		}
	}
	return 0, fmt.Errorf("MIR_BRIDGE_UNKNOWN_TAG %q", value)
}
func machineBridgeUntag(values []string, tag int) (string, error) {
	if tag < 0 || tag >= len(values) {
		return "", fmt.Errorf("MIR_BRIDGE_UNKNOWN_TAG %d", tag)
	}
	return values[tag], nil
}
func (w *machineBridgeWriter) tag(values []string, value string) error {
	tag, err := machineBridgeTag(values, value)
	if err != nil {
		return err
	}
	return w.i32(tag)
}

type machineBridgeWriter struct {
	bytes.Buffer
	trace *[]machineBridgeWireField
}

func (w *machineBridgeWriter) u32(v int) error {
	if v < 0 || v > math.MaxUint32 {
		return fmt.Errorf("MIR_BRIDGE_RANGE %d", v)
	}
	return binary.Write(&w.Buffer, binary.LittleEndian, uint32(v))
}
func (w *machineBridgeWriter) i32(v int) error {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return fmt.Errorf("MIR_BRIDGE_RANGE %d", v)
	}
	return binary.Write(&w.Buffer, binary.LittleEndian, int32(v))
}
func (w *machineBridgeWriter) bool(v bool) error {
	if v {
		return w.i32(1)
	}
	return w.i32(0)
}
func (w *machineBridgeWriter) str(v string) error {
	if err := w.u32(len(v)); err != nil {
		return err
	}
	_, err := w.WriteString(v)
	return err
}
func (w *machineBridgeWriter) strings(v []string) error {
	if err := w.u32(len(v)); err != nil {
		return err
	}
	for _, s := range v {
		if err := w.str(s); err != nil {
			return err
		}
	}
	return nil
}
func (w *machineBridgeWriter) span(s Span) error {
	if err := w.i32(s.Line); err != nil {
		return err
	}
	return w.i32(s.Column)
}
func (w *machineBridgeWriter) operand(o MachineOperand) error {
	for _, v := range []int{o.ID, o.Width, o.Base, o.BaseSlot, o.Index, o.Scale, o.Disp} {
		if err := w.i32(v); err != nil {
			return err
		}
	}
	if err := w.tag(machineBridgeKinds, o.Kind); err != nil {
		return err
	}
	if err := w.str(o.Literal); err != nil {
		return err
	}
	if err := w.str(o.Region); err != nil {
		return err
	}
	return w.bool(o.Signed)
}

// EncodeMachineBridge accepts only verified MachineIR and never reads source or
// the inspection printer. The output is deterministic for identical input.
func encodeMachineBridgeManual(m MachineModule) ([]byte, error) {
	if err := VerifyMachineIR(m); err != nil {
		return nil, err
	}
	w := &machineBridgeWriter{}
	w.WriteString(legacyMachineBridgeSchema)
	if err := w.u32(len(m.Functions)); err != nil {
		return nil, err
	}
	for _, f := range m.Functions {
		for _, s := range []string{f.Identity, f.Name, f.Target, f.ABI, string(f.Result)} {
			if err := w.str(s); err != nil {
				return nil, err
			}
		}
		if err := w.span(f.Source); err != nil {
			return nil, err
		}
		if err := w.strings(f.Facts); err != nil {
			return nil, err
		}
		if err := w.strings(f.Decisions); err != nil {
			return nil, err
		}
		for _, n := range []int{f.Frame.LocalSize, f.Frame.Alignment, f.Frame.ShadowSpace} {
			if err := w.i32(n); err != nil {
				return nil, err
			}
		}
		if err := w.bool(f.Frame.HasCalls); err != nil {
			return nil, err
		}
		if err := w.u32(len(f.Args)); err != nil {
			return nil, err
		}
		for _, a := range f.Args {
			if err := w.str(string(a.Type)); err != nil {
				return nil, err
			}
			for _, n := range []int{a.Index, a.Width, machineBridgeBoolInt(a.Indirect), int(a.Register), a.StackOffset, machineBridgeBoolInt(a.OnStack), a.VReg} {
				if err := w.i32(n); err != nil {
					return nil, err
				}
			}
		}
		if err := w.u32(len(f.VRegs)); err != nil {
			return nil, err
		}
		for _, v := range f.VRegs {
			for _, n := range []int{v.ID, v.Width, machineBridgeBoolInt(v.Address)} {
				if err := w.i32(n); err != nil {
					return nil, err
				}
			}
		}
		if err := w.u32(len(f.Slots)); err != nil {
			return nil, err
		}
		for _, s := range f.Slots {
			for _, n := range []int{s.ID, s.Size, s.Align, machineBridgeBoolInt(s.IncomingIndirect), s.BaseVReg} {
				if err := w.i32(n); err != nil {
					return nil, err
				}
			}
			if err := w.str(s.Source); err != nil {
				return nil, err
			}
		}
		if err := w.u32(len(f.Blocks)); err != nil {
			return nil, err
		}
		for _, b := range f.Blocks {
			for _, n := range []int{b.ID, b.LIRBlock} {
				if err := w.i32(n); err != nil {
					return nil, err
				}
			}
			if err := w.u32(len(b.Instructions)); err != nil {
				return nil, err
			}
			for _, in := range b.Instructions {
				if err := w.tag(machineBridgeOps, in.Op); err != nil {
					return nil, err
				}
				if err := w.operand(in.Dst); err != nil {
					return nil, err
				}
				if err := w.u32(len(in.Src)); err != nil {
					return nil, err
				}
				for _, src := range in.Src {
					if err := w.operand(src); err != nil {
						return nil, err
					}
				}
				for _, n := range []int{in.Width, in.FlagsDef, in.FlagsUse, in.LIRBlock, in.LIRInstruction} {
					if err := w.i32(n); err != nil {
						return nil, err
					}
				}
				if err := w.tag(machineBridgeConds, in.Cond); err != nil {
					return nil, err
				}
				if err := w.span(in.Source); err != nil {
					return nil, err
				}
				if err := w.str(in.Decision); err != nil {
					return nil, err
				}
				if err := w.strings(in.Facts); err != nil {
					return nil, err
				}
			}
			t := b.Term
			if err := w.tag(machineBridgeOps, t.Op); err != nil {
				return nil, err
			}
			for _, n := range []int{t.True, t.False, t.FlagsUse} {
				if err := w.i32(n); err != nil {
					return nil, err
				}
			}
			if err := w.tag(machineBridgeConds, t.Cond); err != nil {
				return nil, err
			}
			if err := w.span(t.Source); err != nil {
				return nil, err
			}
		}
	}
	return w.Bytes(), nil
}

func machineBridgeBoolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

type machineBridgeReader struct{ *bytes.Reader }

func (r machineBridgeReader) i32() (int, error) {
	var x int32
	err := binary.Read(r.Reader, binary.LittleEndian, &x)
	return int(x), err
}
func (r machineBridgeReader) count() (int, error) {
	var x uint32
	if err := binary.Read(r.Reader, binary.LittleEndian, &x); err != nil {
		return 0, err
	}
	if x > MachineBridgeMaxItems {
		return 0, fmt.Errorf("MIR_BRIDGE_COUNT %d", x)
	}
	return int(x), nil
}
func (r machineBridgeReader) str() (string, error) {
	n, err := r.count()
	if err != nil {
		return "", err
	}
	if n > r.Len() {
		return "", io.ErrUnexpectedEOF
	}
	b := make([]byte, n)
	_, err = io.ReadFull(r.Reader, b)
	return string(b), err
}
func (r machineBridgeReader) strings() ([]string, error) {
	n, err := r.count()
	if err != nil {
		return nil, err
	}
	v := make([]string, n)
	for i := range v {
		v[i], err = r.str()
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}
func (r machineBridgeReader) span() (Span, error) {
	line, err := r.i32()
	if err != nil {
		return Span{}, err
	}
	column, err := r.i32()
	return Span{Line: line, Column: column}, err
}
func (r machineBridgeReader) tag(values []string) (string, error) {
	tag, err := r.i32()
	if err != nil {
		return "", err
	}
	return machineBridgeUntag(values, tag)
}
func (r machineBridgeReader) operand() (MachineOperand, error) {
	o := MachineOperand{}
	fields := []*int{&o.ID, &o.Width, &o.Base, &o.BaseSlot, &o.Index, &o.Scale, &o.Disp}
	for _, p := range fields {
		v, err := r.i32()
		if err != nil {
			return o, err
		}
		*p = v
	}
	var err error
	if o.Kind, err = r.tag(machineBridgeKinds); err != nil {
		return o, err
	}
	if o.Literal, err = r.str(); err != nil {
		return o, err
	}
	if o.Region, err = r.str(); err != nil {
		return o, err
	}
	signed, err := r.i32()
	if signed != 0 && signed != 1 {
		return o, fmt.Errorf("MIR_BRIDGE_BOOLEAN %d", signed)
	}
	o.Signed = signed == 1
	return o, err
}

// DecodeMachineBridge provides an independent artifact-only Go test oracle for
// the Concept reader. A malformed or different schema is always rejected.
func decodeMachineBridgeManual(data []byte) (MachineModule, error) {
	if len(data) < len(legacyMachineBridgeSchema) || string(data[:len(legacyMachineBridgeSchema)]) != legacyMachineBridgeSchema {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_SCHEMA_MISMATCH: want %s", legacyMachineBridgeSchema)
	}
	r := machineBridgeReader{bytes.NewReader(data[len(legacyMachineBridgeSchema):])}
	nf, err := r.count()
	if err != nil {
		return MachineModule{}, err
	}
	m := MachineModule{Functions: make([]MachineFunction, nf)}
	for fi := range m.Functions {
		f := &m.Functions[fi]
		for _, p := range []*string{&f.Identity, &f.Name, &f.Target, &f.ABI} {
			if *p, err = r.str(); err != nil {
				return MachineModule{}, err
			}
		}
		result, e := r.str()
		if e != nil {
			return MachineModule{}, e
		}
		f.Result = LIRType(result)
		if f.Source, err = r.span(); err != nil {
			return MachineModule{}, err
		}
		if f.Facts, err = r.strings(); err != nil {
			return MachineModule{}, err
		}
		if f.Decisions, err = r.strings(); err != nil {
			return MachineModule{}, err
		}
		for _, p := range []*int{&f.Frame.LocalSize, &f.Frame.Alignment, &f.Frame.ShadowSpace} {
			if *p, err = r.i32(); err != nil {
				return MachineModule{}, err
			}
		}
		calls, e := r.i32()
		if e != nil || calls != 0 && calls != 1 {
			return MachineModule{}, fmt.Errorf("MIR_BRIDGE_FRAME_CALLS %d: %v", calls, e)
		}
		f.Frame.HasCalls = calls == 1
		na, e := r.count()
		if e != nil {
			return MachineModule{}, e
		}
		f.Args = make([]MachineArg, na)
		for i := range f.Args {
			a := &f.Args[i]
			s, e := r.str()
			if e != nil {
				return MachineModule{}, e
			}
			a.Type = LIRType(s)
			vals := make([]int, 7)
			for j := range vals {
				if vals[j], err = r.i32(); err != nil {
					return MachineModule{}, err
				}
			}
			if vals[2] < 0 || vals[2] > 1 || vals[5] < 0 || vals[5] > 1 {
				return MachineModule{}, fmt.Errorf("MIR_BRIDGE_ARG_BOOLEAN")
			}
			a.Index, a.Width, a.Indirect, a.Register, a.StackOffset, a.OnStack, a.VReg = vals[0], vals[1], vals[2] == 1, MachineReg(vals[3]), vals[4], vals[5] == 1, vals[6]
		}
		nv, e := r.count()
		if e != nil {
			return MachineModule{}, e
		}
		f.VRegs = make([]MachineVReg, nv)
		for i := range f.VRegs {
			v := &f.VRegs[i]
			if v.ID, err = r.i32(); err != nil {
				return MachineModule{}, err
			}
			if v.Width, err = r.i32(); err != nil {
				return MachineModule{}, err
			}
			a, e := r.i32()
			if e != nil || a < 0 || a > 1 {
				return MachineModule{}, fmt.Errorf("MIR_BRIDGE_VREG_ADDRESS")
			}
			v.Address = a == 1
		}
		ns, e := r.count()
		if e != nil {
			return MachineModule{}, e
		}
		f.Slots = make([]MachineStackSlot, ns)
		for i := range f.Slots {
			s := &f.Slots[i]
			for _, p := range []*int{&s.ID, &s.Size, &s.Align} {
				if *p, err = r.i32(); err != nil {
					return MachineModule{}, err
				}
			}
			ind, e := r.i32()
			if e != nil || ind < 0 || ind > 1 {
				return MachineModule{}, fmt.Errorf("MIR_BRIDGE_SLOT_INDIRECT")
			}
			s.IncomingIndirect = ind == 1
			if s.BaseVReg, err = r.i32(); err != nil {
				return MachineModule{}, err
			}
			if s.Source, err = r.str(); err != nil {
				return MachineModule{}, err
			}
		}
		nb, e := r.count()
		if e != nil {
			return MachineModule{}, e
		}
		f.Blocks = make([]MachineBlock, nb)
		for i := range f.Blocks {
			b := &f.Blocks[i]
			if b.ID, err = r.i32(); err != nil {
				return MachineModule{}, err
			}
			if b.LIRBlock, err = r.i32(); err != nil {
				return MachineModule{}, err
			}
			ni, e := r.count()
			if e != nil {
				return MachineModule{}, e
			}
			b.Instructions = make([]MachineInstruction, ni)
			for j := range b.Instructions {
				in := &b.Instructions[j]
				if in.Op, err = r.tag(machineBridgeOps); err != nil {
					return MachineModule{}, err
				}
				if in.Dst, err = r.operand(); err != nil {
					return MachineModule{}, err
				}
				nsrc, e := r.count()
				if e != nil {
					return MachineModule{}, e
				}
				in.Src = make([]MachineOperand, nsrc)
				for k := range in.Src {
					if in.Src[k], err = r.operand(); err != nil {
						return MachineModule{}, err
					}
				}
				for _, p := range []*int{&in.Width, &in.FlagsDef, &in.FlagsUse, &in.LIRBlock, &in.LIRInstruction} {
					if *p, err = r.i32(); err != nil {
						return MachineModule{}, err
					}
				}
				if in.Cond, err = r.tag(machineBridgeConds); err != nil {
					return MachineModule{}, err
				}
				if in.Source, err = r.span(); err != nil {
					return MachineModule{}, err
				}
				if in.Decision, err = r.str(); err != nil {
					return MachineModule{}, err
				}
				if in.Facts, err = r.strings(); err != nil {
					return MachineModule{}, err
				}
			}
			if b.Term.Op, err = r.tag(machineBridgeOps); err != nil {
				return MachineModule{}, err
			}
			for _, p := range []*int{&b.Term.True, &b.Term.False, &b.Term.FlagsUse} {
				if *p, err = r.i32(); err != nil {
					return MachineModule{}, err
				}
			}
			if b.Term.Cond, err = r.tag(machineBridgeConds); err != nil {
				return MachineModule{}, err
			}
			if b.Term.Source, err = r.span(); err != nil {
				return MachineModule{}, err
			}
		}
	}
	if r.Len() != 0 {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_TRAILING_BYTES %d", r.Len())
	}
	if err := VerifyMachineIR(m); err != nil {
		return MachineModule{}, err
	}
	return m, nil
}
