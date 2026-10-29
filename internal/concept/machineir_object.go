package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// This is checked semantic production/transport, not object placement, copying
// or ABI lowering. All aggregate geometry is supplied by the checked LIR.
func projectObjectMachine(lir LIRModule) (MachineModule, error) {
	return projectObjectContract(lir, false)
}
func projectObjectContract(lir LIRModule, borrow bool) (MachineModule, error) {
	out := ObjectModule{}
	for _, d := range lir.Dependencies {
		out.Dependencies = append(out.Dependencies, ObjectDependency{Identity: d.ModuleIdentity, Digest: d.ContentSHA256})
	}
	types := map[LIRType]int{}
	locations := map[LIRType]int{}
	add := func(key LIRType, t ObjectType) int {
		t.ID = len(out.Types)
		types[key] = t.ID
		locations[lirPointer(key)] = t.ID
		out.Types = append(out.Types, t)
		return t.ID
	}
	var valueType func(LIRType) (int, error)
	valueType = func(t LIRType) (int, error) {
		if target := lirAddressTarget(t); target != "" {
			return valueType(target)
		}
		if id, ok := types[t]; ok {
			return id, nil
		}
		if id, ok := locations[t]; ok {
			return id, nil
		}
		for _, e := range lir.Enums {
			if e.Type == t {
				return add(t, ObjectType{Identity: e.Identity, LayoutIdentity: string(e.Type), Name: e.Name, Kind: "u32", Representation: "scalar", Size: e.Size, Align: e.Alignment, Element: -1, Copy: "checked_value", Evidence: "checked_layout"}), nil
			}
		}
		if t == "void" || lirCallScalar(t) {
			width := scalarWidth(string(t))
			copy := "checked_value"
			if t == "void" {
				copy = "none"
			}
			return add(t, ObjectType{Identity: string(t), LayoutIdentity: string(t), Name: string(t), Kind: string(t), Representation: "scalar", Size: width, Align: max(1, width), Element: -1, Copy: copy, Evidence: "checked_layout"}), nil
		}
		return -1, fmt.Errorf("MIR_CMIRAMD5_MISSING_CHECKED_TYPE %s", t)
	}
	for _, l := range lir.Aggregates {
		fields := []ObjectField{}
		for i, f := range l.Fields {
			var id int
			var err error
			if f.Extent > 0 {
				element, e := valueType(f.ElementType)
				if e != nil {
					return MachineModule{}, e
				}
				if known, ok := types[f.Type]; ok {
					id = known
				} else {
					id = add(f.Type, ObjectType{Identity: string(f.Type), LayoutIdentity: string(f.Type), Name: string(f.Type), Kind: "array", Representation: "fixed_array", Size: f.Size, Align: f.Alignment, Element: element, Extent: f.Extent, Stride: f.Stride, Copy: "checked_value", Evidence: "checked_layout"})
				}
			} else {
				id, err = valueType(f.Type)
				if err != nil {
					return MachineModule{}, err
				}
			}
			fields = append(fields, ObjectField{ID: i, Identity: f.DeclarationIdentity, Name: f.Name, SemanticType: f.TypeIdentity, Type: id, LayoutIdentity: out.Types[id].LayoutIdentity, Offset: f.Offset, Size: f.Size, Align: f.Alignment, Readonly: f.Readonly})
		}
		add(l.Type, ObjectType{Identity: l.Identity, LayoutIdentity: string(l.Type), Name: l.Name, Kind: "aggregate", Representation: l.Representation, Size: l.Size, Align: l.Alignment, Immutable: l.Kind == "record", Element: -1, Copy: l.CopyEligibility, Evidence: "checked_layout", Fields: fields})
	}
	_, _ = valueType("void")
	for _, f := range lir.Functions {
		if f.Machine != nil || f.Activation != nil {
			return MachineModule{}, fmt.Errorf("MIR_CMIRAMD5_UNSUPPORTED_AUTOMATA_FRAME %s", f.Name)
		}
		sf := ObjectFunction{Identity: f.Identity, Name: f.Name, Target: "amd64-windows", ABI: "win64", Source: ObjectSpan{Line: f.Source.Line, Column: f.Source.Column}, Facts: append([]string(nil), f.Facts...), Decisions: append([]string(nil), f.Decisions...)}
		var err error
		sf.ResultType, err = valueType(f.Result)
		if err != nil {
			return MachineModule{}, err
		}
		values := map[int]ObjectValue{}
		register := func(id int, t LIRType) error {
			tid, e := valueType(t)
			if e != nil {
				return e
			}
			class := "scalar"
			if lirAddressTarget(t) != "" {
				class = "location"
			} else if k := out.Types[tid].Kind; k == "aggregate" || k == "array" {
				class = "aggregate"
			}
			values[id] = ObjectValue{ID: id, Type: tid, LayoutIdentity: out.Types[tid].LayoutIdentity, Class: class}
			return nil
		}
		next := 0
		for _, p := range f.Params {
			if err := register(p.ID, p.Type); err != nil {
				return MachineModule{}, err
			}
			sf.Params = append(sf.Params, ObjectParameter{Index: len(sf.Params), Value: p.ID, Type: values[p.ID].Type})
			next = max(next, p.ID+1)
		}
		for _, b := range f.Blocks {
			for _, in := range b.Instructions {
				next = max(next, in.Result+1)
			}
		}
		for _, s := range f.Slots {
			tid, e := valueType(s.Type)
			if e != nil {
				return MachineModule{}, e
			}
			sf.Slots = append(sf.Slots, ObjectSlot{ID: s.ID, Name: s.Name, Type: tid, LayoutIdentity: out.Types[tid].LayoutIdentity, Readonly: s.Readonly, ParamValue: s.ParamValue, Space: "SystemMemory"})
		}
		for _, b := range f.Blocks {
			sb := ObjectBlock{ID: b.ID, Term: ObjectTerminator{Op: b.Term.Op, Value: b.Term.Value, True: b.Term.True, False: b.Term.False, Source: ObjectSpan{Line: b.Term.Source.Line, Column: b.Term.Source.Column}}}
			for ii, in := range b.Instructions {
				tid, e := valueType(in.Type)
				if e != nil {
					return MachineModule{}, e
				}
				si := ObjectInstruction{Op: in.Op, Result: in.Result, Type: tid, Slot: in.Slot, ParentType: -1, Member: in.Member, Offset: in.FrameOffset, Extent: in.Extent, Stride: in.Stride, Copy: "none", Literal: in.Literal, LIRBlock: b.ID, LIRInstruction: ii, Source: ObjectSpan{Line: in.Source.Line, Column: in.Source.Column}, Decision: in.Decision, Facts: append([]string(nil), in.Facts...)}
				for _, a := range in.Args {
					si.Args = append(si.Args, ObjectValueUse{Value: a})
				}
				if in.Op == "borrow" {
					si.Op = "copy_value"
				}
				if in.Aggregate != "" {
					si.ParentType, e = valueType(in.Aggregate)
					if e != nil {
						return MachineModule{}, e
					}
				}
				if in.Op == "record_with" {
					// Ordinary logical value construction, independent of AMD64:
					// one source snapshot, followed by ordered fresh member values.
					base := in.Args[0]
					copyResult := in.Result
					if len(in.Members) > 0 {
						copyResult = next
						next++
					}
					si.Op, si.Result, si.Args, si.Copy = "copy_value", copyResult, []ObjectValueUse{{Value: base}}, "checked_value"
					if err := register(copyResult, in.Type); err != nil {
						return MachineModule{}, err
					}
					sb.Instructions = append(sb.Instructions, si)
					base = copyResult
					for i, member := range in.Members {
						result := in.Result
						if i != len(in.Members)-1 {
							result = next
							next++
						}
						update := si
						update.Op, update.Result, update.Member, update.Args = "replace_member", result, member, []ObjectValueUse{{Value: base}, {Value: in.Args[i+1]}}
						if err := register(result, in.Type); err != nil {
							return MachineModule{}, err
						}
						sb.Instructions = append(sb.Instructions, update)
						base = result
					}
					continue
				}
				if in.Result >= 0 {
					if err := register(in.Result, in.Type); err != nil {
						return MachineModule{}, err
					}
				}
				// Checked readonly scalar bindings have initialization stores only.
				// Preserve that distinction explicitly in v5, including loop-local
				// reinitialization; the frozen scalar contracts remain unchanged.
				if in.Op == "store_slot" && in.Slot >= 0 && f.Slots[slotOrdinal(f.Slots, in.Slot)].Readonly && lirCallScalar(in.Type) {
					si.Op = "initialize_slot"
				}
				if k := out.Types[tid].Kind; (k == "aggregate" || k == "array") && (in.Op == "initialize_slot" || in.Op == "load_slot" || in.Op == "store_slot" || in.Op == "load" || in.Op == "store") {
					si.Copy = "checked_value"
				}
				if in.Op == "call" {
					c := ObjectCall{Target: in.CallTarget, Convention: in.CallABI, ResultType: tid}
					for i, a := range in.Args {
						at, e := valueType(in.ArgTypes[i])
						if e != nil {
							return MachineModule{}, e
						}
						c.Arguments = append(c.Arguments, ObjectCallArgument{Value: a, Type: at})
					}
					si.Calls = []ObjectCall{c}
				}
				sb.Instructions = append(sb.Instructions, si)
			}
			sf.Blocks = append(sf.Blocks, sb)
		}
		for _, v := range values {
			sf.Values = append(sf.Values, v)
		}
		sort.Slice(sf.Values, func(i, j int) bool { return sf.Values[i].ID < sf.Values[j].ID })
		out.Functions = append(out.Functions, sf)
	}
	m := MachineModule{Object: &out}
	if borrow {
		return m, nil
	}
	return m, VerifyMachineIR(m)
}

func encodeObjectMachineBridge(m ObjectModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	h := objectBridgeWireHeader{version: ObjectMachineBridgeVersion}
	copy(h.magic[:], ObjectMachineBridgeSchema)
	hash, _ := hex.DecodeString(ObjectMachineBridgeSchemaHash)
	copy(h.schemaHash[:], hash)
	if err := w.writeWireObjectHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWireObjectModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeObjectMachineBridge(data []byte) (MachineModule, error) {
	fail := func(s string) (MachineModule, error) { return MachineModule{}, fmt.Errorf("MIR_CMIRAMD5_%s", s) }
	if len(data) < ObjectMachineBridgeHeaderSize {
		return fail("TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWireObjectHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(h.magic[:]) != ObjectMachineBridgeSchema || h.version != ObjectMachineBridgeVersion {
		return fail("VERSION_MISMATCH")
	}
	hash, _ := hex.DecodeString(ObjectMachineBridgeSchemaHash)
	if !bytes.Equal(h.schemaHash[:], hash) {
		return fail("SCHEMA_HASH_MISMATCH")
	}
	value, err := r.readWireObjectModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return fail("TRAILING_BYTES")
	}
	m := MachineModule{Object: &value}
	return m, VerifyMachineIR(m)
}

func (m ObjectModule) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s v%d checked object contract sha256:%s\n", ObjectMachineBridgeSchema, ObjectMachineBridgeVersion, ObjectMachineBridgeSchemaHash)
	for _, d := range m.Dependencies {
		fmt.Fprintf(&b, "dependency %q sha256:%s\n", d.Identity, d.Digest)
	}
	for _, t := range m.Types {
		fmt.Fprintf(&b, "type t%d %s identity=%q layout=%q kind=%s repr=%s size=%d align=%d copy=%s immutable=%t\n", t.ID, t.Name, t.Identity, t.LayoutIdentity, t.Kind, t.Representation, t.Size, t.Align, t.Copy, t.Immutable)
		if t.Kind == "array" {
			fmt.Fprintf(&b, "  element=t%d extent=%d stride=%d\n", t.Element, t.Extent, t.Stride)
		}
		for _, f := range t.Fields {
			fmt.Fprintf(&b, "  field %d %s [%s] : t%d semantic=%s offset=%d size=%d align=%d readonly=%t\n", f.ID, f.Name, f.Identity, f.Type, f.SemanticType, f.Offset, f.Size, f.Align, f.Readonly)
		}
	}
	for _, f := range m.Functions {
		fmt.Fprintf(&b, "fn %s [%s] %s %s -> t%d source=%d:%d\n", f.Name, f.Identity, f.Target, f.ABI, f.ResultType, f.Source.Line, f.Source.Column)
		for _, p := range f.Params {
			fmt.Fprintf(&b, "  param %d v%d : t%d (semantic value)\n", p.Index, p.Value, p.Type)
		}
		for _, v := range f.Values {
			fmt.Fprintf(&b, "  v%d : t%d class=%s\n", v.ID, v.Type, v.Class)
		}
		for _, s := range f.Slots {
			fmt.Fprintf(&b, "  storage s%d %s : t%d readonly=%t space=%s (capacity; initialization required)\n", s.ID, s.Name, s.Type, s.Readonly, s.Space)
		}
		for _, block := range f.Blocks {
			fmt.Fprintf(&b, "b%d:\n", block.ID)
			for _, in := range block.Instructions {
				fmt.Fprintf(&b, "  v%d = %s t%d args=%v slot=%d parent=t%d member=%d offset=%d extent=%d stride=%d copy=%s literal=%q ; lir b%d/i%d source=%d:%d decision=%q facts=%q\n", in.Result, in.Op, in.Type, in.Args, in.Slot, in.ParentType, in.Member, in.Offset, in.Extent, in.Stride, in.Copy, in.Literal, in.LIRBlock, in.LIRInstruction, in.Source.Line, in.Source.Column, in.Decision, in.Facts)
				for _, c := range in.Calls {
					fmt.Fprintf(&b, "    call [%s] cc=%s args=%v -> t%d (unclassified)\n", c.Target, c.Convention, c.Arguments, c.ResultType)
				}
			}
			fmt.Fprintf(&b, "  %s v%d b%d b%d\n", block.Term.Op, block.Term.Value, block.Term.True, block.Term.False)
		}
	}
	return b.String()
}
