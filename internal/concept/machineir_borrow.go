package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
)

// Checked semantic projection and transport only. Physical references, their
// allocation and Win64 placement remain Cathedral decisions.
func projectBorrowMachine(lir LIRModule) (MachineModule, error) {
	return projectBorrowContract(lir, false)
}
func projectBorrowContract(lir LIRModule, extension bool) (MachineModule, error) {
	base, err := projectObjectContract(lir, true)
	if err != nil {
		return MachineModule{}, err
	}
	out := BorrowModule{Objects: *base.Object}
	for fi, f := range lir.Functions {
		meta := BorrowFunction{Identity: f.Identity}
		if c := f.Borrow; c != nil {
			for _, s := range c.Scopes {
				meta.Scopes = append(meta.Scopes, BorrowScope{ID: s.ID, Parent: s.Parent, Source: ObjectSpan{Line: s.Source.Line, Column: s.Source.Column}})
			}
			meta.SlotScopes = append([]int(nil), c.SlotScopes...)
			for _, r := range c.References {
				v := out.Objects.Functions[fi].Values[r.Value]
				meta.References = append(meta.References, BorrowReference{Value: r.Value, Target: v.Type, SemanticTarget: r.SemanticTarget, LayoutIdentity: v.LayoutIdentity, RootKind: r.RootKind, Root: r.Root, Parent: r.Parent, Mutable: r.Mutable, Lifetime: r.Lifetime, Evidence: r.Evidence, Provenance: r.Provenance, Escape: "noescape", Space: "SystemMemory", Source: ObjectSpan{Line: r.Source.Line, Column: r.Source.Column}})
			}
		} else {
			meta.Scopes = []BorrowScope{{ID: 0, Parent: -1, Source: ObjectSpan{Line: f.Source.Line, Column: f.Source.Column}}}
			meta.SlotScopes = make([]int, len(f.Slots))
			// Existing local object places also have authority. They are not source
			// references and cannot become call arguments in the v6 verifier.
			for _, b := range f.Blocks {
				for _, in := range b.Instructions {
					if lirAddressTarget(in.Type) == "" || in.Result < 0 {
						continue
					}
					r := BorrowReference{Value: in.Result, SemanticTarget: digest([]byte(string(lirAddressTarget(in.Type)))), Target: out.Objects.Functions[fi].Values[in.Result].Type, LayoutIdentity: out.Objects.Functions[fi].Values[in.Result].LayoutIdentity, Parent: -1, Escape: "noescape", Space: "SystemMemory", Evidence: "checked_place", Source: ObjectSpan{Line: in.Source.Line, Column: in.Source.Column}}
					if in.Op == "slot_address" {
						r.RootKind = "local"
						r.Root = in.Slot
						r.Mutable = !f.Slots[in.Slot].Readonly
						r.Provenance = fmt.Sprintf("%s|slot:%d", f.Identity, in.Slot)
					} else {
						for _, p := range meta.References {
							if p.Value == in.Args[0] {
								r.RootKind = p.RootKind
								r.Root = p.Root
								r.Parent = p.Value
								r.Mutable = p.Mutable
								r.Provenance = p.Provenance
							}
						}
						if in.Op == "field_address" {
							for _, l := range lir.Aggregates {
								if l.Type == in.Aggregate {
									r.Mutable = r.Mutable && l.Kind != "record" && !l.Fields[in.Member].Readonly
								}
							}
						}
					}
					meta.References = append(meta.References, r)
				}
			}
		}
		for bi, b := range out.Objects.Functions[fi].Blocks {
			for ii, in := range b.Instructions {
				scope := 0
				if f.Borrow != nil {
					scope = f.Blocks[bi].Instructions[in.LIRInstruction].Scope
				}
				meta.Sites = append(meta.Sites, BorrowSite{Block: bi, Instruction: ii, Scope: scope})
			}
		}
		out.Authority = append(out.Authority, meta)
	}
	m := MachineModule{Borrow: &out}
	if extension {
		return m, nil
	}
	return m, VerifyMachineIR(m)
}
func encodeBorrowMachineBridge(m BorrowModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	h := borrowBridgeWireHeader{version: BorrowMachineBridgeVersion}
	copy(h.magic[:], BorrowMachineBridgeSchema)
	sum, _ := hex.DecodeString(BorrowMachineBridgeSchemaHash)
	copy(h.schemaHash[:], sum)
	if err := w.writeWireBorrowHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWireBorrowModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeBorrowMachineBridge(data []byte) (MachineModule, error) {
	bad := func(s string) (MachineModule, error) { return MachineModule{}, fmt.Errorf("MIR_CMIRAMD6_%s", s) }
	if len(data) < BorrowMachineBridgeHeaderSize {
		return bad("TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWireBorrowHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(h.magic[:]) != BorrowMachineBridgeSchema || h.version != BorrowMachineBridgeVersion {
		return bad("VERSION")
	}
	sum, _ := hex.DecodeString(BorrowMachineBridgeSchemaHash)
	if !bytes.Equal(h.schemaHash[:], sum) {
		return bad("SCHEMA_HASH")
	}
	m, err := r.readWireBorrowModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return bad("TRAILING_BYTES")
	}
	result := MachineModule{Borrow: &m}
	return result, VerifyMachineIR(result)
}
func borrowSiteScope(m BorrowFunction, block, instruction int) int {
	for _, s := range m.Sites {
		if s.Block == block && s.Instruction == instruction {
			return s.Scope
		}
	}
	return -1
}
func borrowLIRContract(meta BorrowFunction, f ObjectFunction, types []LIRType) (*LIRBorrowContract, error) {
	bad := func(s string) (*LIRBorrowContract, error) { return nil, fmt.Errorf("MIR_CMIRAMD6_%s", s) }
	if meta.Identity != f.Identity || len(meta.References) > 512 || len(meta.Scopes) == 0 || len(meta.Scopes) > 128 || len(meta.SlotScopes) != len(f.Slots) {
		return bad("AUTHORITY_CAPACITY")
	}
	c := &LIRBorrowContract{SlotScopes: meta.SlotScopes}
	for _, s := range meta.Scopes {
		c.Scopes = append(c.Scopes, LIRBorrowScope{ID: s.ID, Parent: s.Parent, Source: Span{Line: s.Source.Line, Column: s.Source.Column}})
	}
	for _, r := range meta.References {
		if r.Target < 0 || r.Target >= len(types) || r.Value < 0 || r.Value >= len(f.Values) || f.Values[r.Value].Class != "location" || r.Target != f.Values[r.Value].Type || r.LayoutIdentity != f.Values[r.Value].LayoutIdentity || r.Escape != "noescape" || r.Space != "SystemMemory" {
			return bad("REFERENCE_IDENTITY")
		}
		c.References = append(c.References, LIRBorrowReference{Value: r.Value, Target: types[r.Target], SemanticTarget: r.SemanticTarget, RootKind: r.RootKind, Root: r.Root, Parent: r.Parent, Mutable: r.Mutable, Lifetime: r.Lifetime, Evidence: r.Evidence, Provenance: r.Provenance, Source: Span{Line: r.Source.Line, Column: r.Source.Column}})
	}
	count := 0
	for bi, b := range f.Blocks {
		for ii := range b.Instructions {
			if count >= len(meta.Sites) || meta.Sites[count].Block != bi || meta.Sites[count].Instruction != ii {
				return bad("SCOPE_SITE")
			}
			count++
		}
	}
	if count != len(meta.Sites) {
		return bad("SCOPE_SITE")
	}
	return c, nil
}
func verifyBorrowMachine(m BorrowModule) error {
	if len(m.Authority) != len(m.Objects.Functions) {
		return fmt.Errorf("MIR_CMIRAMD6_FUNCTION_AUTHORITY")
	}
	err := verifyObjectContract(m.Objects, m.Authority)
	if err == nil || strings.HasPrefix(err.Error(), "MIR_CMIRAMD6_") {
		return err
	}
	return fmt.Errorf("MIR_CMIRAMD6_%s", strings.TrimPrefix(err.Error(), "MIR_CMIRAMD5_"))
}
func borrowMachineString(m BorrowModule) string {
	var b strings.Builder
	b.WriteString("CMIRAMD6 checked borrow authority; SystemMemory; noescape\n")
	b.WriteString(m.Objects.String())
	for _, f := range m.Authority {
		fmt.Fprintf(&b, "authority %s\n", f.Identity)
		for _, r := range f.References {
			fmt.Fprintf(&b, "  ref v%d target=t%d root=%s:%d parent=v%d mutable=%t lifetime=scope%d evidence=%s provenance=%q noescape\n", r.Value, r.Target, r.RootKind, r.Root, r.Parent, r.Mutable, r.Lifetime, r.Evidence, r.Provenance)
		}
	}
	return b.String()
}
