package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Checked semantic projection and generated transport only. Cathedral owns
// variant admission, active-field copying, native storage, ABI and encoding.
func projectTaggedMachine(lir LIRModule) (MachineModule, error) {
	return projectTaggedContract(lir, false)
}
func projectTaggedContract(lir LIRModule, pattern bool) (MachineModule, error) {
	if len(lir.Enums) != 0 && !pattern {
		return MachineModule{}, fmt.Errorf("EVT2H2_UNSUPPORTED_ORDINARY_ENUM_CONTEXT")
	}
	base, err := projectBorrowContract(lir, true)
	if err != nil {
		return MachineModule{}, err
	}
	out := TaggedModule{Base: *base.Borrow}
	ids := map[LIRType]int{}
	for _, t := range out.Base.Objects.Types {
		ids[LIRType(t.LayoutIdentity)] = t.ID
	}
	for _, e := range lir.TaggedEnums {
		id, ok := ids[e.Type]
		if !ok {
			return MachineModule{}, fmt.Errorf("MIR_CMIRAMD8_MISSING_DOMAIN")
		}
		d := TaggedDomain{Type: id, PayloadOffset: e.PayloadOffset}
		for _, v := range e.Variants {
			variant := TaggedVariant{Identity: v.Identity, Name: v.Name, Tag: v.Tag, Size: v.Size, Align: v.Alignment}
			for i, f := range v.Fields {
				tid, ok := ids[f.Type]
				if !ok {
					return MachineModule{}, fmt.Errorf("MIR_CMIRAMD8_MISSING_PAYLOAD_TYPE")
				}
				variant.Fields = append(variant.Fields, ObjectField{ID: i, Identity: f.DeclarationIdentity, Name: f.Name, SemanticType: f.TypeIdentity, Type: tid, LayoutIdentity: string(f.Type), Offset: f.Offset, Size: f.Size, Align: f.Alignment, Readonly: f.Readonly})
			}
			d.Variants = append(d.Variants, variant)
		}
		out.Domains = append(out.Domains, d)
	}
	for fi, f := range out.Base.Objects.Functions {
		for bi, b := range f.Blocks {
			for ii, in := range b.Instructions {
				original := lir.Functions[fi].Blocks[bi].Instructions[in.LIRInstruction]
				if original.Op == "make_tagged" || original.Op == "checked_payload" {
					out.Operations = append(out.Operations, TaggedOperation{Function: fi, Block: bi, Instruction: ii, Kind: original.Op})
				}
			}
		}
	}
	m := MachineModule{Tagged: &out}
	if pattern {
		return m, nil
	}
	return m, VerifyMachineIR(m)
}

func verifyTaggedMachine(m TaggedModule) error {
	bad := func(s string) error { return fmt.Errorf("MIR_CMIRAMD8_%s", s) }
	objects := m.Base.Objects
	if len(m.Domains) < 1 || len(m.Domains) > 128 || len(objects.Types) < 1 || len(objects.Types) > 256 || len(objects.Functions) < 1 || len(objects.Functions) > 128 || len(m.Base.Authority) != len(objects.Functions) || len(m.Operations) > 131072 {
		return bad("CAPACITY")
	}
	domains, err := checkedTaggedDomains(objects, m.Domains)
	if err != nil {
		return err
	}
	ops := map[taggedSite]string{}
	prior := taggedSite{-1, -1, -1}
	for _, op := range m.Operations {
		s := taggedSite{op.Function, op.Block, op.Instruction}
		if s.function < 0 || s.function >= len(objects.Functions) {
			return bad("OPERATION_SITE")
		}
		f := objects.Functions[s.function]
		if s.block < 0 || s.block >= len(f.Blocks) || s.instruction < 0 || s.instruction >= len(f.Blocks[s.block].Instructions) {
			return bad("OPERATION_SITE")
		}
		if s.function < prior.function || s.function == prior.function && (s.block < prior.block || s.block == prior.block && s.instruction <= prior.instruction) {
			return bad("OPERATION_ORDER")
		}
		if op.Kind != "make_tagged" && op.Kind != "checked_payload" {
			return bad("OPERATION_KIND")
		}
		ops[s] = op.Kind
		prior = s
	}
	return verifyObjectContractTagged(objects, m.Base.Authority, nil, domains, ops)
}

func encodeTaggedMachineBridge(m TaggedModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	h := taggedBridgeWireHeader{version: TaggedMachineBridgeVersion}
	copy(h.magic[:], TaggedMachineBridgeSchema)
	sum, _ := hex.DecodeString(TaggedMachineBridgeSchemaHash)
	copy(h.schemaHash[:], sum)
	if err := w.writeWireTaggedHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWireTaggedModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeTaggedMachineBridge(data []byte) (MachineModule, error) {
	bad := func(s string) (MachineModule, error) { return MachineModule{}, fmt.Errorf("MIR_CMIRAMD8_%s", s) }
	if len(data) < TaggedMachineBridgeHeaderSize {
		return bad("TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWireTaggedHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(h.magic[:]) != TaggedMachineBridgeSchema || h.version != TaggedMachineBridgeVersion {
		return bad("VERSION")
	}
	sum, _ := hex.DecodeString(TaggedMachineBridgeSchemaHash)
	if !bytes.Equal(h.schemaHash[:], sum) {
		return bad("SCHEMA_HASH")
	}
	value, err := r.readWireTaggedModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return bad("TRAILING_BYTES")
	}
	m := MachineModule{Tagged: &value}
	return m, VerifyMachineIR(m)
}

func checkedTaggedDomains(objects ObjectModule, records []TaggedDomain) ([]LIRTaggedEnum, error) {
	bad := func(s string) error { return fmt.Errorf("MIR_CMIRAMD8_%s", s) }
	domains := []LIRTaggedEnum{}
	seen := map[int]bool{}
	for _, d := range records {
		if d.Type < 0 || d.Type >= len(objects.Types) || seen[d.Type] {
			return nil, bad("DOMAIN")
		}
		seen[d.Type] = true
		t := objects.Types[d.Type]
		e := LIRTaggedEnum{Type: LIRType(t.LayoutIdentity), Identity: t.Identity, Name: t.Name, Representation: "checked_tagged_union", CopyEligibility: t.Copy, Size: t.Size, Alignment: t.Align, PayloadOffset: d.PayloadOffset}
		for _, v := range d.Variants {
			variant := LIRTaggedVariant{Identity: v.Identity, Name: v.Name, Tag: v.Tag, Size: v.Size, Alignment: v.Align}
			for i, f := range v.Fields {
				if f.ID != i || f.Type < 0 || f.Type >= d.Type {
					return nil, bad("FIELD_TYPE")
				}
				child := objects.Types[f.Type]
				if f.LayoutIdentity != child.LayoutIdentity {
					return nil, bad("FIELD_LAYOUT")
				}
				field := LIRAggregateField{Name: f.Name, Type: LIRType(child.LayoutIdentity), TypeIdentity: f.SemanticType, Offset: f.Offset, Size: f.Size, Alignment: f.Align, Readonly: f.Readonly, DeclarationIdentity: f.Identity}
				if child.Kind == "array" {
					if child.Element < 0 || child.Element >= f.Type {
						return nil, bad("ARRAY")
					}
					field.ElementType = LIRType(objects.Types[child.Element].LayoutIdentity)
					field.Extent = child.Extent
					field.Stride = child.Stride
					field.ElementAlignment = objects.Types[child.Element].Align
				}
				variant.Fields = append(variant.Fields, field)
			}
			e.Variants = append(e.Variants, variant)
		}
		domains = append(domains, e)
	}

	return domains, nil
}
