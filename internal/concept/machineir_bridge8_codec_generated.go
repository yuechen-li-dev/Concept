// Code generated from checked Standard.Backend.BridgeSchema8; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const TaggedMachineBridgeSchema = "CMIRAMD8"
const TaggedMachineBridgeVersion = 8
const TaggedMachineBridgeSchemaHash = "8dc34f916ab8293e838db026d720825f10d2a7c12006c19924a0fffea94c0711"
const TaggedMachineBridgeMaxItems = 1048576
const TaggedMachineBridgeHeaderSize = 44

var taggedBridgeTagsTaggedOperationKind = []string{"make_tagged", "checked_payload"}

type taggedBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireTaggedHeader(v taggedBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireTaggedHeader() (taggedBridgeWireHeader, error) {
	var v taggedBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type TaggedVariant struct {
	Identity string
	Name     string
	Tag      int
	Size     int
	Align    int
	Fields   []ObjectField
}

func (w *machineBridgeWriter) writeWireTaggedVariant(v TaggedVariant) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedVariant", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedVariant", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Tag)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Tag offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedVariant", "Tag", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Size)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Size offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedVariant", "Size", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Align)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Align offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedVariant", "Align", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Fields) > TaggedMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Fields))
			}
			if err := w.u32(len(v.Fields)); err != nil {
				return err
			}
			for _, element := range v.Fields {
				if err := w.writeWireObjectField(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Fields offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedVariant", "Fields", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireTaggedVariant() (TaggedVariant, error) {
	var v TaggedVariant
	var err error
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Tag, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Tag offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Size, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Size offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Align, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Align offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Fields, err = func() ([]ObjectField, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectField, n)
		for i := range v {
			v[i], err = r.readWireObjectField()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedVariant.Fields offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type TaggedDomain struct {
	Type          int
	PayloadOffset int
	Variants      []TaggedVariant
}

func (w *machineBridgeWriter) writeWireTaggedDomain(v TaggedDomain) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedDomain.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedDomain", "Type", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.PayloadOffset)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedDomain.PayloadOffset offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedDomain", "PayloadOffset", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Variants) > TaggedMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Variants))
			}
			if err := w.u32(len(v.Variants)); err != nil {
				return err
			}
			for _, element := range v.Variants {
				if err := w.writeWireTaggedVariant(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedDomain.Variants offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedDomain", "Variants", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireTaggedDomain() (TaggedDomain, error) {
	var v TaggedDomain
	var err error
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedDomain.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.PayloadOffset, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedDomain.PayloadOffset offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Variants, err = func() ([]TaggedVariant, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]TaggedVariant, n)
		for i := range v {
			v[i], err = r.readWireTaggedVariant()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedDomain.Variants offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type TaggedOperation struct {
	Function    int
	Block       int
	Instruction int
	Kind        string
}

func (w *machineBridgeWriter) writeWireTaggedOperation(v TaggedOperation) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Function)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Function offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedOperation", "Function", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Block)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Block offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedOperation", "Block", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Instruction)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Instruction offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedOperation", "Instruction", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(taggedBridgeTagsTaggedOperationKind, v.Kind); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Kind offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedOperation", "Kind", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireTaggedOperation() (TaggedOperation, error) {
	var v TaggedOperation
	var err error
	v.Function, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Function offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Block, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Block offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Instruction, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Instruction offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Kind, err = r.tag(taggedBridgeTagsTaggedOperationKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedOperation.Kind offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type TaggedModule struct {
	Base       BorrowModule
	Domains    []TaggedDomain
	Operations []TaggedOperation
}

func (w *machineBridgeWriter) writeWireTaggedModule(v TaggedModule) error {
	{
		offset := w.Len()
		if err := w.writeWireBorrowModule(v.Base); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedModule.Base offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedModule", "Base", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Domains) > TaggedMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Domains))
			}
			if err := w.u32(len(v.Domains)); err != nil {
				return err
			}
			for _, element := range v.Domains {
				if err := w.writeWireTaggedDomain(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedModule.Domains offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedModule", "Domains", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Operations) > TaggedMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Operations))
			}
			if err := w.u32(len(v.Operations)); err != nil {
				return err
			}
			for _, element := range v.Operations {
				if err := w.writeWireTaggedOperation(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedModule.Operations offset=%d: %w", offset, err)
		}
		w.traceField("WireTaggedModule", "Operations", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireTaggedModule() (TaggedModule, error) {
	var v TaggedModule
	var err error
	v.Base, err = r.readWireBorrowModule()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedModule.Base offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Domains, err = func() ([]TaggedDomain, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]TaggedDomain, n)
		for i := range v {
			v[i], err = r.readWireTaggedDomain()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedModule.Domains offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Operations, err = func() ([]TaggedOperation, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]TaggedOperation, n)
		for i := range v {
			v[i], err = r.readWireTaggedOperation()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireTaggedModule.Operations offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
