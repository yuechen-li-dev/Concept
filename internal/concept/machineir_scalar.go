package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
)

// CMIRAMD4 is a schema-generated virtual MachineIR variant. This checked
// projection selects no physical ABI location, SSE instruction or frame policy.
func scalarRepresentation(t LIRType) string {
	if strings.HasPrefix(string(t), "ptr<") || strings.HasPrefix(string(t), "[") {
		return "address"
	}
	return string(t)
}
func scalarWidth(rep string) int {
	if rep == "bool" {
		return 1
	}
	if rep == "address" {
		return 8
	}
	return lirWidth(LIRType(rep))
}
func scalarClass(rep string) string {
	if lirFloating(LIRType(rep)) {
		return "XMM"
	}
	return "GPR"
}
func scalarSpan(s Span) ScalarSpan { return ScalarSpan{Line: s.Line, Column: s.Column} }

func projectScalarMachine(lir LIRModule) (MachineModule, error) {
	out := ScalarMachineModule{}
	for _, f := range lir.Functions {
		if f.Machine != nil || f.Activation != nil {
			return MachineModule{}, fmt.Errorf("MIR_CMIRAMD4_UNSUPPORTED_MACHINE_FRAME %s", f.Name)
		}
		sf := ScalarMachineFunction{Identity: f.Identity, Name: f.Name, Target: "amd64-windows", ABI: "win64", Result: scalarRepresentation(f.Result), Source: scalarSpan(f.Source), Facts: append([]string(nil), f.Facts...), Decisions: append([]string(nil), f.Decisions...)}
		types := map[int]LIRType{}
		add := func(id int, t LIRType) {
			rep := scalarRepresentation(t)
			for len(sf.VRegs) <= id {
				sf.VRegs = append(sf.VRegs, ScalarMachineVReg{ID: len(sf.VRegs)})
			}
			sf.VRegs[id] = ScalarMachineVReg{ID: id, Class: scalarClass(rep), Representation: rep, Width: scalarWidth(rep), Address: rep == "address"}
			types[id] = t
		}
		for i, p := range f.Params {
			add(p.ID, p.Type)
			sf.Params = append(sf.Params, ScalarMachineParameter{Index: i, Value: p.ID, Type: string(p.Type)})
		}
		for _, s := range f.Slots {
			rep := scalarRepresentation(s.Type)
			if s.Extent > 0 {
				rep = scalarRepresentation(s.Element)
			}
			size := scalarWidth(rep)
			if s.Extent > 0 {
				size *= s.Extent
			}
			sf.Slots = append(sf.Slots, ScalarMachineSlot{ID: s.ID, Name: s.Name, Representation: rep, Size: size, Align: scalarWidth(rep), Extent: s.Extent, ParamValue: s.ParamValue})
		}
		for _, b := range f.Blocks {
			sb := ScalarMachineBlock{ID: b.ID, Term: ScalarMachineTerminator{Op: b.Term.Op, Value: b.Term.Value, True: b.Term.True, False: b.Term.False, Source: scalarSpan(b.Term.Source)}}
			for ii, in := range b.Instructions {
				si := ScalarMachineInstruction{Op: in.Op, Result: in.Result, Representation: scalarRepresentation(in.Type), Slot: in.Slot, Literal: in.Literal, Extent: in.Extent, Stride: in.Stride, LIRBlock: b.ID, LIRInstruction: ii, Source: scalarSpan(in.Source), Decision: in.Decision, Facts: append([]string(nil), in.Facts...)}
				if in.Op == "const" && lirFloating(in.Type) {
					si.Op = "fconst"
				}
				if len(in.Args) > 0 && lirFloating(types[in.Args[0]]) {
					switch in.Op {
					case "eq", "ne", "lt", "le", "gt", "ge":
						si.Op = "f" + in.Op
					}
				}
				for _, id := range in.Args {
					si.Args = append(si.Args, ScalarValueUse{Value: id})
				}
				if in.Op == "call" {
					c := ScalarMachineCall{Target: in.CallTarget, Convention: in.CallABI, Result: si.Representation}
					for i, id := range in.Args {
						c.Arguments = append(c.Arguments, ScalarMachineCallArgument{Value: id, Type: scalarRepresentation(in.ArgTypes[i])})
					}
					si.Calls = []ScalarMachineCall{c}
				}
				if in.Result >= 0 {
					add(in.Result, in.Type)
				}
				sb.Instructions = append(sb.Instructions, si)
			}
			sf.Blocks = append(sf.Blocks, sb)
		}
		out.Functions = append(out.Functions, sf)
	}
	m := MachineModule{Scalar: &out}
	return m, VerifyMachineIR(m)
}

func (m ScalarMachineModule) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s v%d virtual scalar contract\n", ScalarMachineBridgeSchema, ScalarMachineBridgeVersion)
	for _, f := range m.Functions {
		fmt.Fprintf(&b, "fn %s [%s] %s %s -> %s source=%d:%d\n", f.Name, f.Identity, f.Target, f.ABI, f.Result, f.Source.Line, f.Source.Column)
		for _, p := range f.Params {
			fmt.Fprintf(&b, "  arg %d %s -> v%d (virtual)\n", p.Index, p.Type, p.Value)
		}
		for _, v := range f.VRegs {
			fmt.Fprintf(&b, "  v%d : %s / %s width=%d address=%t\n", v.ID, v.Representation, v.Class, v.Width, v.Address)
		}
		for _, s := range f.Slots {
			fmt.Fprintf(&b, "  slot s%d %s : %s size=%d align=%d extent=%d param=v%d\n", s.ID, s.Name, s.Representation, s.Size, s.Align, s.Extent, s.ParamValue)
		}
		fmt.Fprintf(&b, "  facts=%q decisions=%q\n", f.Facts, f.Decisions)
		for _, block := range f.Blocks {
			fmt.Fprintf(&b, "b%d:\n", block.ID)
			for _, in := range block.Instructions {
				fmt.Fprintf(&b, "  v%d = %s %s", in.Result, in.Op, in.Representation)
				for _, a := range in.Args {
					fmt.Fprintf(&b, " v%d", a.Value)
				}
				fmt.Fprintf(&b, " slot=%d literal=%q extent=%d stride=%d ; lir b%d/i%d source=%d:%d decision=%q facts=%q\n", in.Slot, in.Literal, in.Extent, in.Stride, in.LIRBlock, in.LIRInstruction, in.Source.Line, in.Source.Column, in.Decision, in.Facts)
				for _, c := range in.Calls {
					fmt.Fprintf(&b, "    call @%s cc %s args=%v -> %s\n", c.Target, c.Convention, c.Arguments, c.Result)
				}
			}
			fmt.Fprintf(&b, "  %s v%d b%d b%d source=%d:%d\n", block.Term.Op, block.Term.Value, block.Term.True, block.Term.False, block.Term.Source.Line, block.Term.Source.Column)
		}
	}
	return b.String()
}

func encodeScalarMachineBridge(m ScalarMachineModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	hash, _ := hex.DecodeString(ScalarMachineBridgeSchemaHash)
	h := scalarBridgeWireHeader{version: ScalarMachineBridgeVersion}
	copy(h.magic[:], ScalarMachineBridgeSchema)
	copy(h.schemaHash[:], hash)
	if err := w.writeWireScalarHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWireScalarMachineModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

// Strict v4 reader; the public dispatcher separately retains frozen v3 support.
func DecodeScalarMachineBridge(data []byte) (MachineModule, error) {
	fail := func(reason string) (MachineModule, error) {
		return MachineModule{}, fmt.Errorf("MIR_CMIRAMD4_%s", reason)
	}
	if len(data) < ScalarMachineBridgeHeaderSize {
		return fail("TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWireScalarHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(h.magic[:]) != ScalarMachineBridgeSchema || h.version != ScalarMachineBridgeVersion {
		return fail("VERSION_MISMATCH")
	}
	hash, _ := hex.DecodeString(ScalarMachineBridgeSchemaHash)
	if !bytes.Equal(h.schemaHash[:], hash) {
		return fail("SCHEMA_HASH_MISMATCH")
	}
	scalar, err := r.readWireScalarMachineModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return fail("TRAILING_BYTES")
	}
	m := MachineModule{Scalar: &scalar}
	return m, VerifyMachineIR(m)
}
