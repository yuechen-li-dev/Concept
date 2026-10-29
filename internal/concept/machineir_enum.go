package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

// Only checked semantic production and bootstrap transport live here.
// Cathedral owns carrier admission, instructions, storage, ABI and encoding.
func projectEnumMachine(lir LIRModule) (MachineModule, error) {
	base, err := projectObjectContract(lir, true)
	if err != nil {
		return MachineModule{}, err
	}
	out := EnumModule{Objects: *base.Object}
	for _, e := range lir.Enums {
		for _, t := range out.Objects.Types {
			if t.LayoutIdentity == string(e.Type) {
				out.Domains = append(out.Domains, EnumDomain{Type: t.ID, Identity: e.Identity, LayoutIdentity: string(e.Type), Size: e.Size, Align: e.Alignment, Variants: e.Variants})
			}
		}
	}
	m := MachineModule{Enum: &out}
	return m, VerifyMachineIR(m)
}
func verifyEnumMachine(m EnumModule) error {
	bad := func(s string) error { return fmt.Errorf("MIR_CMIRAMD7_%s", s) }
	if len(m.Domains) == 0 || len(m.Domains) > 128 {
		return bad("DOMAINS")
	}
	seen := map[int]bool{}
	var enums []LIREnum
	for _, d := range m.Domains {
		if d.Type < 0 || d.Type >= len(m.Objects.Types) || seen[d.Type] {
			return bad("DOMAIN_TYPE")
		}
		seen[d.Type] = true
		t := m.Objects.Types[d.Type]
		if t.Kind != "u32" || t.Identity != d.Identity || t.LayoutIdentity != d.LayoutIdentity || t.Size != d.Size || t.Align != d.Align {
			return bad("REPRESENTATION")
		}
		enums = append(enums, LIREnum{Type: LIRType(d.LayoutIdentity), Identity: d.Identity, Name: t.Name, Size: d.Size, Alignment: d.Align, Variants: d.Variants})
	}
	for _, t := range m.Objects.Types {
		if t.Kind == "aggregate" || t.Kind == "array" {
			return bad("UNSUPPORTED_TYPE")
		}
	}
	for _, f := range m.Objects.Functions {
		for _, v := range f.Values {
			if v.Class != "scalar" {
				return bad("UNSUPPORTED_VALUE")
			}
		}
	}
	return verifyObjectContractEnums(m.Objects, nil, enums)
}
func encodeEnumMachineBridge(m EnumModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	h := enumBridgeWireHeader{version: EnumMachineBridgeVersion}
	copy(h.magic[:], EnumMachineBridgeSchema)
	sum, _ := hex.DecodeString(EnumMachineBridgeSchemaHash)
	copy(h.schemaHash[:], sum)
	if err := w.writeWireEnumHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWireEnumModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeEnumMachineBridge(data []byte) (MachineModule, error) {
	bad := func(s string) (MachineModule, error) { return MachineModule{}, fmt.Errorf("MIR_CMIRAMD7_%s", s) }
	if len(data) < EnumMachineBridgeHeaderSize {
		return bad("TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWireEnumHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(h.magic[:]) != EnumMachineBridgeSchema || h.version != EnumMachineBridgeVersion {
		return bad("VERSION")
	}
	sum, _ := hex.DecodeString(EnumMachineBridgeSchemaHash)
	if !bytes.Equal(h.schemaHash[:], sum) {
		return bad("SCHEMA_HASH")
	}
	document, err := r.readWireEnumModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return bad("TRAILING_BYTES")
	}
	out := MachineModule{Enum: &document}
	return out, VerifyMachineIR(out)
}
