package concept

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

const machineBridgeHeaderSize = MachineBridgeHeaderSize

type machineBridgeWireField struct {
	Record, Field string
	Offset, End   int
}

func (w *machineBridgeWriter) traceField(record, field string, offset int) {
	if w.trace != nil {
		*w.trace = append(*w.trace, machineBridgeWireField{record, field, offset, w.Len()})
	}
}
func machineBridgePayloadDifference(expected, actual []byte, fields []machineBridgeWireField) error {
	limit := len(expected)
	if len(actual) < limit {
		limit = len(actual)
	}
	offset := 0
	for offset < limit && expected[offset] == actual[offset] {
		offset++
	}
	if offset == len(expected) && offset == len(actual) {
		return nil
	}
	record, field := "bridge", "length"
	extent := int(^uint(0) >> 1)
	for _, f := range fields {
		if f.Offset <= offset && offset < f.End && f.End-f.Offset < extent {
			record, field, extent = f.Record, f.Field, f.End-f.Offset
		}
	}
	want, got := "EOF", "EOF"
	if offset < len(expected) {
		want = fmt.Sprintf("0x%02x", expected[offset])
	}
	if offset < len(actual) {
		got = fmt.Sprintf("0x%02x", actual[offset])
	}
	return fmt.Errorf("MIR_BRIDGE_BYTE_MISMATCH %s.%s expected_offset=%d actual_offset=%d expected=%s actual=%s", record, field, offset, offset, want, got)
}

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

// Production bridge invocation over the generated codec. No legacy fallback.
func EncodeMachineBridge(m MachineModule) ([]byte, error) {
	if err := VerifyMachineIR(m); err != nil {
		return nil, err
	}
	w := &machineBridgeWriter{}
	hash, err := hex.DecodeString(MachineBridgeSchemaHash)
	if err != nil {
		return nil, err
	}
	header := bridgeWireHeader{version: MachineBridgeVersion}
	copy(header.magic[:], MachineBridgeSchema)
	copy(header.schemaHash[:], hash)
	if err := w.writeWireHeader(header); err != nil {
		return nil, err
	}
	if err := w.writeWireMachineModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeMachineBridge(data []byte) (MachineModule, error) {
	if len(data) < machineBridgeHeaderSize {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_TRUNCATED_HEADER")
	}
	r := machineBridgeReader{bytes.NewReader(data)}
	header, err := r.readWireHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(header.magic[:]) != MachineBridgeSchema {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_SCHEMA_MISMATCH: want %s", MachineBridgeSchema)
	}
	if header.version != MachineBridgeVersion {
		return MachineModule{}, fmt.Errorf("MIR_BRIDGE_VERSION_MISMATCH: got %d want %d", header.version, MachineBridgeVersion)
	}
	hash, _ := hex.DecodeString(MachineBridgeSchemaHash)
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
