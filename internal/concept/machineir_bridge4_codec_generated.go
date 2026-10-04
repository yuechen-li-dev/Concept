// Code generated from checked Standard.Backend.BridgeSchema4; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const ScalarMachineBridgeSchema = "CMIRAMD4"
const ScalarMachineBridgeVersion = 4
const ScalarMachineBridgeSchemaHash = "aa0cedb528f0d0f780a7c1809b60f87a483f3de6fec77684eeef4c94f278836c"
const ScalarMachineBridgeMaxItems = 1048576
const ScalarMachineBridgeHeaderSize = 44

var scalarBridgeTagsRegisterClass = []string{"GPR", "XMM"}
var scalarBridgeTagsScalarRepresentation = []string{"void", "bool", "i8", "u8", "i16", "u16", "i32", "u32", "i64", "u64", "float32", "float64", "address"}
var scalarBridgeTagsScalarOpcode = []string{"const", "fconst", "load_slot", "store_slot", "load", "store", "index_address", "check_index", "checked_add", "checked_sub", "checked_mul", "add", "sub", "mul", "eq", "ne", "lt", "le", "gt", "ge", "fadd", "fsub", "fmul", "fdiv", "fneg", "feq", "fne", "flt", "fle", "fgt", "fge", "call"}
var scalarBridgeTagsScalarTerminator = []string{"return", "jump", "branch"}

type scalarBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireScalarHeader(v scalarBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarHeader() (scalarBridgeWireHeader, error) {
	var v scalarBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarSpan struct {
	Line   int
	Column int
}

func (w *machineBridgeWriter) writeWireScalarSpan(v ScalarSpan) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Line)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarSpan.Line offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarSpan", "Line", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Column)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarSpan.Column offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarSpan", "Column", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarSpan() (ScalarSpan, error) {
	var v ScalarSpan
	var err error
	v.Line, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarSpan.Line offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Column, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarSpan.Column offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarValueUse struct {
	Value int
}

func (w *machineBridgeWriter) writeWireScalarValueUse(v ScalarValueUse) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarValueUse.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarValueUse", "Value", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarValueUse() (ScalarValueUse, error) {
	var v ScalarValueUse
	var err error
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarValueUse.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineVReg struct {
	ID             int
	Class          string
	Representation string
	Width          int
	Address        bool
}

func (w *machineBridgeWriter) writeWireScalarMachineVReg(v ScalarMachineVReg) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineVReg", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsRegisterClass, v.Class); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Class offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineVReg", "Class", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarRepresentation, v.Representation); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Representation offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineVReg", "Representation", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Width)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Width offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineVReg", "Width", offset)
	}
	{
		offset := w.Len()
		if err := w.bool(v.Address); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Address offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineVReg", "Address", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineVReg() (ScalarMachineVReg, error) {
	var v ScalarMachineVReg
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Class, err = r.tag(scalarBridgeTagsRegisterClass)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Class offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Representation, err = r.tag(scalarBridgeTagsScalarRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Representation offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Width, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Width offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Address, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineVReg.Address offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineParameter struct {
	Index int
	Value int
	Type  string
}

func (w *machineBridgeWriter) writeWireScalarMachineParameter(v ScalarMachineParameter) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Index)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineParameter.Index offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineParameter", "Index", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineParameter.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineParameter", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineParameter.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineParameter", "Type", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineParameter() (ScalarMachineParameter, error) {
	var v ScalarMachineParameter
	var err error
	v.Index, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineParameter.Index offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineParameter.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineParameter.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineSlot struct {
	ID             int
	Name           string
	Representation string
	Size           int
	Align          int
	Extent         int
	ParamValue     int
}

func (w *machineBridgeWriter) writeWireScalarMachineSlot(v ScalarMachineSlot) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarRepresentation, v.Representation); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Representation offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "Representation", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Size)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Size offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "Size", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Align)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Align offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "Align", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Extent)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Extent offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "Extent", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.ParamValue)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.ParamValue offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineSlot", "ParamValue", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineSlot() (ScalarMachineSlot, error) {
	var v ScalarMachineSlot
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Representation, err = r.tag(scalarBridgeTagsScalarRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Representation offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Size, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Size offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Align, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Align offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Extent, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.Extent offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ParamValue, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineSlot.ParamValue offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineCallArgument struct {
	Value int
	Type  string
}

func (w *machineBridgeWriter) writeWireScalarMachineCallArgument(v ScalarMachineCallArgument) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCallArgument.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineCallArgument", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarRepresentation, v.Type); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCallArgument.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineCallArgument", "Type", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineCallArgument() (ScalarMachineCallArgument, error) {
	var v ScalarMachineCallArgument
	var err error
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCallArgument.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.tag(scalarBridgeTagsScalarRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCallArgument.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineCall struct {
	Target     string
	Convention string
	Arguments  []ScalarMachineCallArgument
	Result     string
}

func (w *machineBridgeWriter) writeWireScalarMachineCall(v ScalarMachineCall) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Target)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Target offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineCall", "Target", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Convention)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Convention offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineCall", "Convention", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Arguments) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Arguments))
			}
			if err := w.u32(len(v.Arguments)); err != nil {
				return err
			}
			for _, element := range v.Arguments {
				if err := w.writeWireScalarMachineCallArgument(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Arguments offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineCall", "Arguments", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarRepresentation, v.Result); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Result offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineCall", "Result", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineCall() (ScalarMachineCall, error) {
	var v ScalarMachineCall
	var err error
	v.Target, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Target offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Convention, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Convention offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Arguments, err = func() ([]ScalarMachineCallArgument, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineCallArgument, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineCallArgument()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Arguments offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Result, err = r.tag(scalarBridgeTagsScalarRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineCall.Result offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineInstruction struct {
	Op             string
	Result         int
	Representation string
	Args           []ScalarValueUse
	Slot           int
	Literal        string
	Extent         int
	Stride         int
	LIRBlock       int
	LIRInstruction int
	Source         ScalarSpan
	Decision       string
	Facts          []string
	Calls          []ScalarMachineCall
}

func (w *machineBridgeWriter) writeWireScalarMachineInstruction(v ScalarMachineInstruction) error {
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarOpcode, v.Op); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Op offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Op", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Result)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Result offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Result", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarRepresentation, v.Representation); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Representation offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Representation", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Args) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Args))
			}
			if err := w.u32(len(v.Args)); err != nil {
				return err
			}
			for _, element := range v.Args {
				if err := w.writeWireScalarValueUse(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Args offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Args", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Slot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Slot offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Slot", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Literal)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Literal offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Literal", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Extent)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Extent offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Extent", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Stride)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Stride offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Stride", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.LIRBlock)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.LIRBlock offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "LIRBlock", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.LIRInstruction)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.LIRInstruction offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "LIRInstruction", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireScalarSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Source", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Decision)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Decision offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Decision", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Facts) > ScalarMachineBridgeMaxItems {
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
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Facts offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Facts", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Calls) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Calls))
			}
			if err := w.u32(len(v.Calls)); err != nil {
				return err
			}
			for _, element := range v.Calls {
				if err := w.writeWireScalarMachineCall(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Calls offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineInstruction", "Calls", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineInstruction() (ScalarMachineInstruction, error) {
	var v ScalarMachineInstruction
	var err error
	v.Op, err = r.tag(scalarBridgeTagsScalarOpcode)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Op offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Result, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Result offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Representation, err = r.tag(scalarBridgeTagsScalarRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Representation offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Args, err = func() ([]ScalarValueUse, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarValueUse, n)
		for i := range v {
			v[i], err = r.readWireScalarValueUse()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Args offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Slot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Slot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Literal, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Literal offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Extent, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Extent offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Stride, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Stride offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRBlock, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.LIRBlock offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRInstruction, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.LIRInstruction offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireScalarSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Decision, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Decision offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Facts offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Calls, err = func() ([]ScalarMachineCall, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineCall, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineCall()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineInstruction.Calls offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineTerminator struct {
	Op     string
	Value  int
	True   int
	False  int
	Source ScalarSpan
}

func (w *machineBridgeWriter) writeWireScalarMachineTerminator(v ScalarMachineTerminator) error {
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarTerminator, v.Op); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.Op offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineTerminator", "Op", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineTerminator", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.True)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.True offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineTerminator", "True", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.False)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.False offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineTerminator", "False", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireScalarSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineTerminator", "Source", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineTerminator() (ScalarMachineTerminator, error) {
	var v ScalarMachineTerminator
	var err error
	v.Op, err = r.tag(scalarBridgeTagsScalarTerminator)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.Op offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.True, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.True offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.False, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.False offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireScalarSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineTerminator.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineBlock struct {
	ID           int
	Instructions []ScalarMachineInstruction
	Term         ScalarMachineTerminator
}

func (w *machineBridgeWriter) writeWireScalarMachineBlock(v ScalarMachineBlock) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineBlock.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineBlock", "ID", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Instructions) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Instructions))
			}
			if err := w.u32(len(v.Instructions)); err != nil {
				return err
			}
			for _, element := range v.Instructions {
				if err := w.writeWireScalarMachineInstruction(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineBlock.Instructions offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineBlock", "Instructions", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireScalarMachineTerminator(v.Term); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineBlock.Term offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineBlock", "Term", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineBlock() (ScalarMachineBlock, error) {
	var v ScalarMachineBlock
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineBlock.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Instructions, err = func() ([]ScalarMachineInstruction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineInstruction, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineInstruction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineBlock.Instructions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Term, err = r.readWireScalarMachineTerminator()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineBlock.Term offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineFunction struct {
	Identity  string
	Name      string
	Target    string
	ABI       string
	Result    string
	Source    ScalarSpan
	Facts     []string
	Decisions []string
	Params    []ScalarMachineParameter
	VRegs     []ScalarMachineVReg
	Slots     []ScalarMachineSlot
	Blocks    []ScalarMachineBlock
}

func (w *machineBridgeWriter) writeWireScalarMachineFunction(v ScalarMachineFunction) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Target)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Target offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Target", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.ABI)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.ABI offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "ABI", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(scalarBridgeTagsScalarRepresentation, v.Result); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Result offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Result", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireScalarSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Source", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Facts) > ScalarMachineBridgeMaxItems {
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
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Facts offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Facts", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Decisions) > ScalarMachineBridgeMaxItems {
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
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Decisions offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Decisions", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Params) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Params))
			}
			if err := w.u32(len(v.Params)); err != nil {
				return err
			}
			for _, element := range v.Params {
				if err := w.writeWireScalarMachineParameter(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Params offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Params", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.VRegs) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.VRegs))
			}
			if err := w.u32(len(v.VRegs)); err != nil {
				return err
			}
			for _, element := range v.VRegs {
				if err := w.writeWireScalarMachineVReg(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.VRegs offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "VRegs", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Slots) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Slots))
			}
			if err := w.u32(len(v.Slots)); err != nil {
				return err
			}
			for _, element := range v.Slots {
				if err := w.writeWireScalarMachineSlot(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Slots offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Slots", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Blocks) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Blocks))
			}
			if err := w.u32(len(v.Blocks)); err != nil {
				return err
			}
			for _, element := range v.Blocks {
				if err := w.writeWireScalarMachineBlock(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Blocks offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineFunction", "Blocks", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineFunction() (ScalarMachineFunction, error) {
	var v ScalarMachineFunction
	var err error
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Target, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Target offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ABI, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.ABI offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Result, err = r.tag(scalarBridgeTagsScalarRepresentation)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Result offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireScalarSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Facts offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Decisions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Params, err = func() ([]ScalarMachineParameter, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineParameter, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineParameter()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Params offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.VRegs, err = func() ([]ScalarMachineVReg, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineVReg, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineVReg()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.VRegs offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Slots, err = func() ([]ScalarMachineSlot, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineSlot, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineSlot()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Slots offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Blocks, err = func() ([]ScalarMachineBlock, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineBlock, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineBlock()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineFunction.Blocks offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type ScalarMachineModule struct {
	Functions []ScalarMachineFunction
}

func (w *machineBridgeWriter) writeWireScalarMachineModule(v ScalarMachineModule) error {
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Functions) > ScalarMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Functions))
			}
			if err := w.u32(len(v.Functions)); err != nil {
				return err
			}
			for _, element := range v.Functions {
				if err := w.writeWireScalarMachineFunction(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineModule.Functions offset=%d: %w", offset, err)
		}
		w.traceField("WireScalarMachineModule", "Functions", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireScalarMachineModule() (ScalarMachineModule, error) {
	var v ScalarMachineModule
	var err error
	v.Functions, err = func() ([]ScalarMachineFunction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]ScalarMachineFunction, n)
		for i := range v {
			v[i], err = r.readWireScalarMachineFunction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireScalarMachineModule.Functions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
