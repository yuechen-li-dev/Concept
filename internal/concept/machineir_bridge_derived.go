package concept

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const machineBridgeHeaderSize = 8 + 4 + 32

func (r machineBridgeReader) u32() (uint32, error) {
	var value uint32
	err := binary.Read(r.Reader, binary.LittleEndian, &value)
	return value, err
}

func (w *machineBridgeWriter) enum(v, count int) error {
	if v < 0 || v >= count {
		return fmt.Errorf("MIR_BRIDGE_UNKNOWN_TAG %d", v)
	}
	return w.i32(v)
}
func (r machineBridgeReader) enum(count int) (int, error) {
	v, err := r.i32()
	if err != nil {
		return 0, err
	}
	if v < 0 || v >= count {
		return 0, fmt.Errorf("MIR_BRIDGE_UNKNOWN_TAG %d", v)
	}
	return v, nil
}
func (r machineBridgeReader) boolean() (bool, error) {
	v, err := r.i32()
	if err != nil {
		return false, err
	}
	if v != 0 && v != 1 {
		return false, fmt.Errorf("MIR_BRIDGE_BOOLEAN %d", v)
	}
	return v == 1, nil
}

// Shadow entry point until whole-corpus payload agreement is qualified.
func EncodeMachineBridgeDerived(m MachineModule) ([]byte, error) {
	if err := VerifyMachineIR(m); err != nil {
		return nil, err
	}
	w := &machineBridgeWriter{}
	hash, err := hex.DecodeString(DerivedMachineBridgeSchemaHash)
	if err != nil {
		return nil, err
	}
	header := bridgeWireHeader{version: DerivedMachineBridgeVersion}
	copy(header.magic[:], DerivedMachineBridgeSchema)
	copy(header.schemaHash[:], hash)
	if err := w.writeWireHeader(header); err != nil {
		return nil, err
	}
	if err := w.writeWireMachineModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeMachineBridgeDerived(data []byte) (MachineModule, error) {
	if len(data) < machineBridgeHeaderSize {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	header, err := r.readWireHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(header.magic[:]) != DerivedMachineBridgeSchema {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_SCHEMA_MISMATCH: want %s", DerivedMachineBridgeSchema)
	}
	if header.version != DerivedMachineBridgeVersion {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_VERSION_MISMATCH: got %d want %d", header.version, DerivedMachineBridgeVersion)
	}
	hash, _ := hex.DecodeString(DerivedMachineBridgeSchemaHash)
	if !bytes.Equal(header.schemaHash[:], hash) {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_SCHEMA_HASH_MISMATCH")
	}
	m, err := r.readWireMachineModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_TRAILING_BYTES %d", r.Len())
	}
	if err := VerifyMachineIR(m); err != nil {
		return MachineModule{}, err
	}
	return m, nil
}
