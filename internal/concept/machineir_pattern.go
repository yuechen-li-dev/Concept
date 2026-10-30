package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Checked semantic association/transport only. All physical decisions remain
// in Concept Cathedral, including typed tag tests and required-tag traps.
func projectPatternMachine(lir LIRModule) (MachineModule, error) {
	base, err := projectTaggedContract(lir, true)
	if err != nil {
		return MachineModule{}, err
	}
	out := PatternModule{Base: *base.Tagged}
	ids := map[LIRType]int{}
	for _, t := range out.Base.Base.Objects.Types {
		ids[LIRType(t.LayoutIdentity)] = t.ID
	}
	for _, e := range lir.Enums {
		out.Enums = append(out.Enums, EnumDomain{Type: ids[e.Type], Identity: e.Identity, LayoutIdentity: string(e.Type), Size: e.Size, Align: e.Alignment, Variants: e.Variants})
	}
	for fi, f := range lir.Functions {
		for _, flow := range f.Patterns {
			wire := PatternFlow{Function: fi, SubjectSlot: flow.SubjectSlot, ResultSlot: flow.ResultSlot, Join: flow.Join, Scope: flow.Scope}
			for _, a := range flow.Arms {
				arm := PatternArm{Dispatch: a.Dispatch, Body: a.Body, Scope: a.Scope}
				for _, s := range a.Bindings {
					arm.Bindings = append(arm.Bindings, PatternBinding{Slot: s})
				}
				wire.Arms = append(wire.Arms, arm)
			}
			out.Flows = append(out.Flows, wire)
		}
		for bi, b := range out.Base.Base.Objects.Functions[fi].Blocks {
			for ii, in := range b.Instructions {
				original := f.Blocks[bi].Instructions[in.LIRInstruction]
				if original.Op == "test_variant" || original.Op == "require_variant" {
					out.Operations = append(out.Operations, PatternOperation{Function: fi, Block: bi, Instruction: ii, Kind: original.Op})
				}
			}
		}
	}
	m := MachineModule{Pattern: &out}
	return m, VerifyMachineIR(m)
}
func verifyPatternMachine(m PatternModule) error {
	bad := func(s string) error { return fmt.Errorf("MIR_CMIRAMD9_%s", s) }
	objects := m.Base.Base.Objects
	if len(objects.Functions) == 0 || len(objects.Functions) > 128 || len(objects.Types) == 0 || len(objects.Types) > 256 || len(m.Enums) > 128 || len(m.Base.Domains) > 128 || len(m.Base.Base.Authority) != len(objects.Functions) || len(m.Flows) == 0 || len(m.Flows) > 8192 || len(m.Operations) > 131072 {
		return bad("CAPACITY")
	}
	tagged, err := checkedTaggedDomains(objects, m.Base.Domains)
	if err != nil {
		return err
	}
	var enums []LIREnum
	seen := map[int]bool{}
	for _, d := range m.Enums {
		if d.Type < 0 || d.Type >= len(objects.Types) || seen[d.Type] {
			return bad("DOMAIN")
		}
		seen[d.Type] = true
		t := objects.Types[d.Type]
		if t.Kind != "u32" || t.Identity != d.Identity || t.LayoutIdentity != d.LayoutIdentity || t.Size != d.Size || t.Align != d.Align {
			return bad("DOMAIN")
		}
		enums = append(enums, LIREnum{Type: LIRType(d.LayoutIdentity), Identity: d.Identity, Name: t.Name, Size: d.Size, Alignment: d.Align, Variants: d.Variants})
	}
	ops := map[taggedSite]string{}
	add := func(fi, bi, ii int, kind string) error {
		if fi < 0 || fi >= len(objects.Functions) || bi < 0 || bi >= len(objects.Functions[fi].Blocks) || ii < 0 || ii >= len(objects.Functions[fi].Blocks[bi].Instructions) {
			return bad("SITE")
		}
		s := taggedSite{fi, bi, ii}
		if ops[s] != "" {
			return bad("SITE")
		}
		ops[s] = kind
		return nil
	}
	prior := taggedSite{-1, -1, -1}
	ordered := func(s taggedSite) bool {
		return s.function > prior.function || s.function == prior.function && (s.block > prior.block || s.block == prior.block && s.instruction > prior.instruction)
	}
	for _, op := range m.Base.Operations {
		s := taggedSite{op.Function, op.Block, op.Instruction}
		if !ordered(s) || op.Kind != "make_tagged" && op.Kind != "checked_payload" {
			return bad("OPERATION")
		}
		if err := add(op.Function, op.Block, op.Instruction, op.Kind); err != nil {
			return err
		}
		prior = s
	}
	prior = taggedSite{-1, -1, -1}
	for _, op := range m.Operations {
		s := taggedSite{op.Function, op.Block, op.Instruction}
		if !ordered(s) || op.Kind != "test_variant" && op.Kind != "require_variant" {
			return bad("OPERATION")
		}
		if err := add(op.Function, op.Block, op.Instruction, op.Kind); err != nil {
			return err
		}
		prior = s
	}
	flows := map[int][]LIRPatternFlow{}
	priorFunction := -1
	for _, p := range m.Flows {
		if p.Function < priorFunction || p.Function < 0 || p.Function >= len(objects.Functions) {
			return bad("FLOW")
		}
		priorFunction = p.Function
		flow := LIRPatternFlow{SubjectSlot: p.SubjectSlot, ResultSlot: p.ResultSlot, Join: p.Join, Scope: p.Scope}
		for _, a := range p.Arms {
			arm := LIRPatternArm{Dispatch: a.Dispatch, Body: a.Body, Scope: a.Scope}
			for _, s := range a.Bindings {
				arm.Bindings = append(arm.Bindings, s.Slot)
			}
			flow.Arms = append(flow.Arms, arm)
		}
		flows[p.Function] = append(flows[p.Function], flow)
	}
	return verifyObjectContractTagged(objects, m.Base.Base.Authority, enums, tagged, ops, flows)
}
func encodePatternMachineBridge(m PatternModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	h := patternBridgeWireHeader{version: PatternMachineBridgeVersion}
	copy(h.magic[:], PatternMachineBridgeSchema)
	hash, _ := hex.DecodeString(PatternMachineBridgeSchemaHash)
	copy(h.schemaHash[:], hash)
	if err := w.writeWirePatternHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWirePatternModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodePatternMachineBridge(data []byte) (MachineModule, error) {
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWirePatternHeader()
	if err != nil {
		return MachineModule{}, fmt.Errorf("MIR_CMIRAMD9_HEADER: %w", err)
	}
	if string(h.magic[:]) != PatternMachineBridgeSchema || h.version != PatternMachineBridgeVersion || hex.EncodeToString(h.schemaHash[:]) != PatternMachineBridgeSchemaHash {
		return MachineModule{}, fmt.Errorf("MIR_CMIRAMD9_IDENTITY")
	}
	doc, err := r.readWirePatternModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return MachineModule{}, fmt.Errorf("MIR_CMIRAMD9_TRAILING")
	}
	m := MachineModule{Pattern: &doc}
	return m, VerifyMachineIR(m)
}
