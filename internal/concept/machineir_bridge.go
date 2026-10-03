package concept

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Audited generic little-endian byte primitives. Record order and tags are generated.
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
	if len(v) > MachineBridgeMaxItems {
		return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v))
	}
	if err := w.u32(len(v)); err != nil {
		return err
	}
	_, err := w.WriteString(v)
	return err
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
func (r machineBridgeReader) tag(values []string) (string, error) {
	tag, err := r.i32()
	if err != nil {
		return "", err
	}
	return machineBridgeUntag(values, tag)
}
