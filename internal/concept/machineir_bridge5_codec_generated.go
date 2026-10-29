// Code generated from checked Standard.Backend.BridgeSchema5; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const ObjectMachineBridgeSchema = "CMIRAMD5"
const ObjectMachineBridgeVersion = 5
const ObjectMachineBridgeSchemaHash = "7925f9d6ccb54c2af7ca32322bcd59748916f3310e4d95fcf15a5f5ad32c9b60"
const ObjectMachineBridgeMaxItems = 1048576
const ObjectMachineBridgeHeaderSize = 44

var objectBridgeTagsObjectKind = []string{"void", "bool", "i8", "u8", "i16", "u16", "i32", "u32", "i64", "u64", "float32", "float64", "aggregate", "array"}
var objectBridgeTagsObjectRepresentation = []string{"scalar", "native", "C", "fixed_array"}
var objectBridgeTagsObjectValueClass = []string{"scalar", "aggregate", "location"}
var objectBridgeTagsObjectCopyKind = []string{"none", "checked_value"}
var objectBridgeTagsObjectEvidence = []string{"checked_layout"}
var objectBridgeTagsObjectAddressSpace = []string{"SystemMemory"}
var objectBridgeTagsObjectOpcode = []string{"const", "make_aggregate", "make_array", "copy_value", "replace_member", "initialize_slot", "load_slot", "store_slot", "slot_address", "field_address", "load", "store", "index_address", "check_index", "checked_add", "checked_sub", "checked_mul", "add", "sub", "mul", "eq", "ne", "lt", "le", "gt", "ge", "fadd", "fsub", "fmul", "fdiv", "fneg", "call"}
var objectBridgeTagsObjectTerminator = []string{"return", "jump", "branch"}

type objectBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireObjectHeader(v objectBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectHeader() (objectBridgeWireHeader, error) {
	var v objectBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectSpan struct {
	Line   int
	Column int
}

func (w *machineBridgeWriter) writeWireObjectSpan(v ObjectSpan) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Line)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSpan.Line offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSpan", "Line", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Column)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSpan.Column offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSpan", "Column", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectSpan() (ObjectSpan, error) {
	var v ObjectSpan
	var err error
	v.Line, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSpan.Line offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Column, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSpan.Column offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectDependency struct {
	Identity string
	Digest   string
}

func (w *machineBridgeWriter) writeWireObjectDependency(v ObjectDependency) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectDependency.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectDependency", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Digest)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectDependency.Digest offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectDependency", "Digest", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectDependency() (ObjectDependency, error) {
	var v ObjectDependency
	var err error
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectDependency.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Digest, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectDependency.Digest offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectField struct {
	ID             int
	Identity       string
	Name           string
	SemanticType   string
	Type           int
	LayoutIdentity string
	Offset         int
	Size           int
	Align          int
	Readonly       bool
}

func (w *machineBridgeWriter) writeWireObjectField(v ObjectField) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.SemanticType)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.SemanticType offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "SemanticType", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Type", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.LayoutIdentity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.LayoutIdentity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "LayoutIdentity", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Offset)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Offset offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Offset", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Size)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Size offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Size", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Align)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Align offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Align", offset)
	}
	{
		offset := w.Len()
		if err := w.bool(v.Readonly); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Readonly offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectField", "Readonly", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectField() (ObjectField, error) {
	var v ObjectField
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.SemanticType, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.SemanticType offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LayoutIdentity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.LayoutIdentity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Offset, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Offset offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Size, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Size offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Align, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Align offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Readonly, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectField.Readonly offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectType struct {
	ID             int
	Identity       string
	LayoutIdentity string
	Name           string
	Kind           string
	Representation string
	Size           int
	Align          int
	Immutable      bool
	Element        int
	Extent         int
	Stride         int
	Copy           string
	Evidence       string
	Fields         []ObjectField
}

func (w *machineBridgeWriter) writeWireObjectType(v ObjectType) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.LayoutIdentity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.LayoutIdentity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "LayoutIdentity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectKind, v.Kind); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Kind offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Kind", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectRepresentation, v.Representation); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Representation offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Representation", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Size)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Size offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Size", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Align)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Align offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Align", offset)
	}
	{
		offset := w.Len()
		if err := w.bool(v.Immutable); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Immutable offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Immutable", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Element)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Element offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Element", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Extent)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Extent offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Extent", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Stride)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Stride offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Stride", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectCopyKind, v.Copy); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Copy offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Copy", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectEvidence, v.Evidence); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Evidence offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Evidence", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Fields) > ObjectMachineBridgeMaxItems {
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
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Fields offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectType", "Fields", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectType() (ObjectType, error) {
	var v ObjectType
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LayoutIdentity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.LayoutIdentity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Kind, err = r.tag(objectBridgeTagsObjectKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Kind offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Representation, err = r.tag(objectBridgeTagsObjectRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Representation offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Size, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Size offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Align, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Align offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Immutable, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Immutable offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Element, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Element offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Extent, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Extent offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Stride, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Stride offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Copy, err = r.tag(objectBridgeTagsObjectCopyKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Copy offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Evidence, err = r.tag(objectBridgeTagsObjectEvidence)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Evidence offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectType.Fields offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectValue struct {
	ID             int
	Type           int
	LayoutIdentity string
	Class          string
}

func (w *machineBridgeWriter) writeWireObjectValue(v ObjectValue) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectValue", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectValue", "Type", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.LayoutIdentity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.LayoutIdentity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectValue", "LayoutIdentity", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectValueClass, v.Class); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.Class offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectValue", "Class", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectValue() (ObjectValue, error) {
	var v ObjectValue
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LayoutIdentity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.LayoutIdentity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Class, err = r.tag(objectBridgeTagsObjectValueClass)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValue.Class offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectValueUse struct {
	Value int
}

func (w *machineBridgeWriter) writeWireObjectValueUse(v ObjectValueUse) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValueUse.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectValueUse", "Value", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectValueUse() (ObjectValueUse, error) {
	var v ObjectValueUse
	var err error
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectValueUse.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectParameter struct {
	Index int
	Value int
	Type  int
}

func (w *machineBridgeWriter) writeWireObjectParameter(v ObjectParameter) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Index)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectParameter.Index offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectParameter", "Index", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectParameter.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectParameter", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectParameter.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectParameter", "Type", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectParameter() (ObjectParameter, error) {
	var v ObjectParameter
	var err error
	v.Index, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectParameter.Index offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectParameter.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectParameter.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectSlot struct {
	ID             int
	Name           string
	Type           int
	LayoutIdentity string
	Readonly       bool
	ParamValue     int
	Space          string
}

func (w *machineBridgeWriter) writeWireObjectSlot(v ObjectSlot) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "Type", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.LayoutIdentity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.LayoutIdentity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "LayoutIdentity", offset)
	}
	{
		offset := w.Len()
		if err := w.bool(v.Readonly); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Readonly offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "Readonly", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.ParamValue)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.ParamValue offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "ParamValue", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectAddressSpace, v.Space); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Space offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectSlot", "Space", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectSlot() (ObjectSlot, error) {
	var v ObjectSlot
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LayoutIdentity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.LayoutIdentity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Readonly, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Readonly offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ParamValue, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.ParamValue offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Space, err = r.tag(objectBridgeTagsObjectAddressSpace)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectSlot.Space offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectCallArgument struct {
	Value int
	Type  int
}

func (w *machineBridgeWriter) writeWireObjectCallArgument(v ObjectCallArgument) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCallArgument.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectCallArgument", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCallArgument.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectCallArgument", "Type", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectCallArgument() (ObjectCallArgument, error) {
	var v ObjectCallArgument
	var err error
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCallArgument.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCallArgument.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectCall struct {
	Target     string
	Convention string
	Arguments  []ObjectCallArgument
	ResultType int
}

func (w *machineBridgeWriter) writeWireObjectCall(v ObjectCall) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Target)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.Target offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectCall", "Target", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Convention)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.Convention offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectCall", "Convention", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Arguments) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Arguments))
			}
			if err := w.u32(len(v.Arguments)); err != nil {
				return err
			}
			for _, element := range v.Arguments {
				if err := w.writeWireObjectCallArgument(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.Arguments offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectCall", "Arguments", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.ResultType)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.ResultType offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectCall", "ResultType", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectCall() (ObjectCall, error) {
	var v ObjectCall
	var err error
	v.Target, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.Target offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Convention, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.Convention offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Arguments, err = func() ([]ObjectCallArgument, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectCallArgument, n)
		for i := range v {
			v[i], err = r.readWireObjectCallArgument()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.Arguments offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ResultType, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectCall.ResultType offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectInstruction struct {
	Op             string
	Result         int
	Type           int
	Args           []ObjectValueUse
	Slot           int
	ParentType     int
	Member         int
	Offset         int
	Extent         int
	Stride         int
	Copy           string
	Literal        string
	LIRBlock       int
	LIRInstruction int
	Source         ObjectSpan
	Decision       string
	Facts          []string
	Calls          []ObjectCall
}

func (w *machineBridgeWriter) writeWireObjectInstruction(v ObjectInstruction) error {
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectOpcode, v.Op); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Op offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Op", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Result)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Result offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Result", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Type", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Args) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Args))
			}
			if err := w.u32(len(v.Args)); err != nil {
				return err
			}
			for _, element := range v.Args {
				if err := w.writeWireObjectValueUse(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Args offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Args", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Slot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Slot offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Slot", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.ParentType)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.ParentType offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "ParentType", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Member)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Member offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Member", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Offset)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Offset offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Offset", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Extent)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Extent offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Extent", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Stride)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Stride offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Stride", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectCopyKind, v.Copy); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Copy offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Copy", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Literal)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Literal offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Literal", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.LIRBlock)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.LIRBlock offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "LIRBlock", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.LIRInstruction)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.LIRInstruction offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "LIRInstruction", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireObjectSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Source", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Decision)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Decision offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Decision", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Facts) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Facts))
			}
			if err := w.u32(len(v.Facts)); err != nil {
				return err
			}
			for _, element := range v.Facts {
				if err := w.str(string(element)); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Facts offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Facts", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Calls) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Calls))
			}
			if err := w.u32(len(v.Calls)); err != nil {
				return err
			}
			for _, element := range v.Calls {
				if err := w.writeWireObjectCall(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Calls offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectInstruction", "Calls", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectInstruction() (ObjectInstruction, error) {
	var v ObjectInstruction
	var err error
	v.Op, err = r.tag(objectBridgeTagsObjectOpcode)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Op offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Result, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Result offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Args, err = func() ([]ObjectValueUse, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectValueUse, n)
		for i := range v {
			v[i], err = r.readWireObjectValueUse()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Args offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Slot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Slot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ParentType, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.ParentType offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Member, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Member offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Offset, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Offset offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Extent, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Extent offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Stride, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Stride offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Copy, err = r.tag(objectBridgeTagsObjectCopyKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Copy offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Literal, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Literal offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRBlock, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.LIRBlock offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRInstruction, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.LIRInstruction offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireObjectSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Decision, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Decision offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Facts, err = func() ([]string, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]string, n)
		for i := range v {
			v[i], err = r.str()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Facts offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Calls, err = func() ([]ObjectCall, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectCall, n)
		for i := range v {
			v[i], err = r.readWireObjectCall()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectInstruction.Calls offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectTerminator struct {
	Op     string
	Value  int
	True   int
	False  int
	Source ObjectSpan
}

func (w *machineBridgeWriter) writeWireObjectTerminator(v ObjectTerminator) error {
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectTerminator, v.Op); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.Op offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectTerminator", "Op", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectTerminator", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.True)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.True offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectTerminator", "True", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.False)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.False offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectTerminator", "False", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireObjectSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectTerminator", "Source", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectTerminator() (ObjectTerminator, error) {
	var v ObjectTerminator
	var err error
	v.Op, err = r.tag(objectBridgeTagsObjectTerminator)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.Op offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.True, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.True offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.False, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.False offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireObjectSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectTerminator.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectBlock struct {
	ID           int
	Instructions []ObjectInstruction
	Term         ObjectTerminator
}

func (w *machineBridgeWriter) writeWireObjectBlock(v ObjectBlock) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectBlock.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectBlock", "ID", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Instructions) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Instructions))
			}
			if err := w.u32(len(v.Instructions)); err != nil {
				return err
			}
			for _, element := range v.Instructions {
				if err := w.writeWireObjectInstruction(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectBlock.Instructions offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectBlock", "Instructions", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireObjectTerminator(v.Term); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectBlock.Term offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectBlock", "Term", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectBlock() (ObjectBlock, error) {
	var v ObjectBlock
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectBlock.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Instructions, err = func() ([]ObjectInstruction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectInstruction, n)
		for i := range v {
			v[i], err = r.readWireObjectInstruction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectBlock.Instructions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Term, err = r.readWireObjectTerminator()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectBlock.Term offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectFunction struct {
	Identity   string
	Name       string
	Target     string
	ABI        string
	ResultType int
	Source     ObjectSpan
	Facts      []string
	Decisions  []string
	Params     []ObjectParameter
	Values     []ObjectValue
	Slots      []ObjectSlot
	Blocks     []ObjectBlock
}

func (w *machineBridgeWriter) writeWireObjectFunction(v ObjectFunction) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Target)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Target offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Target", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.ABI)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.ABI offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "ABI", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.ResultType)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.ResultType offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "ResultType", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireObjectSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Source", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Facts) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Facts))
			}
			if err := w.u32(len(v.Facts)); err != nil {
				return err
			}
			for _, element := range v.Facts {
				if err := w.str(string(element)); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Facts offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Facts", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Decisions) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Decisions))
			}
			if err := w.u32(len(v.Decisions)); err != nil {
				return err
			}
			for _, element := range v.Decisions {
				if err := w.str(string(element)); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Decisions offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Decisions", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Params) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Params))
			}
			if err := w.u32(len(v.Params)); err != nil {
				return err
			}
			for _, element := range v.Params {
				if err := w.writeWireObjectParameter(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Params offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Params", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Values) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Values))
			}
			if err := w.u32(len(v.Values)); err != nil {
				return err
			}
			for _, element := range v.Values {
				if err := w.writeWireObjectValue(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Values offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Values", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Slots) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Slots))
			}
			if err := w.u32(len(v.Slots)); err != nil {
				return err
			}
			for _, element := range v.Slots {
				if err := w.writeWireObjectSlot(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Slots offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Slots", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Blocks) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Blocks))
			}
			if err := w.u32(len(v.Blocks)); err != nil {
				return err
			}
			for _, element := range v.Blocks {
				if err := w.writeWireObjectBlock(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Blocks offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectFunction", "Blocks", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectFunction() (ObjectFunction, error) {
	var v ObjectFunction
	var err error
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Target, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Target offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ABI, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.ABI offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ResultType, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.ResultType offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireObjectSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Facts, err = func() ([]string, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]string, n)
		for i := range v {
			v[i], err = r.str()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Facts offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Decisions, err = func() ([]string, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]string, n)
		for i := range v {
			v[i], err = r.str()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Decisions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Params, err = func() ([]ObjectParameter, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectParameter, n)
		for i := range v {
			v[i], err = r.readWireObjectParameter()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Params offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Values, err = func() ([]ObjectValue, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectValue, n)
		for i := range v {
			v[i], err = r.readWireObjectValue()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Values offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Slots, err = func() ([]ObjectSlot, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectSlot, n)
		for i := range v {
			v[i], err = r.readWireObjectSlot()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Slots offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Blocks, err = func() ([]ObjectBlock, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectBlock, n)
		for i := range v {
			v[i], err = r.readWireObjectBlock()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectFunction.Blocks offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ObjectModule struct {
	Dependencies []ObjectDependency
	Types        []ObjectType
	Functions    []ObjectFunction
}

func (w *machineBridgeWriter) writeWireObjectModule(v ObjectModule) error {
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Dependencies) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Dependencies))
			}
			if err := w.u32(len(v.Dependencies)); err != nil {
				return err
			}
			for _, element := range v.Dependencies {
				if err := w.writeWireObjectDependency(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectModule.Dependencies offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectModule", "Dependencies", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Types) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Types))
			}
			if err := w.u32(len(v.Types)); err != nil {
				return err
			}
			for _, element := range v.Types {
				if err := w.writeWireObjectType(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectModule.Types offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectModule", "Types", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Functions) > ObjectMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Functions))
			}
			if err := w.u32(len(v.Functions)); err != nil {
				return err
			}
			for _, element := range v.Functions {
				if err := w.writeWireObjectFunction(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireObjectModule.Functions offset=%d: %w", offset, err)
		}
		w.traceField("WireObjectModule", "Functions", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireObjectModule() (ObjectModule, error) {
	var v ObjectModule
	var err error
	v.Dependencies, err = func() ([]ObjectDependency, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectDependency, n)
		for i := range v {
			v[i], err = r.readWireObjectDependency()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectModule.Dependencies offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Types, err = func() ([]ObjectType, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectType, n)
		for i := range v {
			v[i], err = r.readWireObjectType()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectModule.Types offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Functions, err = func() ([]ObjectFunction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ObjectFunction, n)
		for i := range v {
			v[i], err = r.readWireObjectFunction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireObjectModule.Functions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
