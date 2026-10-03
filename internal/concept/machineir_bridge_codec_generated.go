// Code generated from checked Standard.Backend.BridgeSchema; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const DerivedMachineBridgeSchema = "CMIRAMD2"
const DerivedMachineBridgeVersion = 2
const DerivedMachineBridgeSchemaHash = "811a24e57119ec1e458deaf0b1e2c3dd36ca26b522e2bd31548e8bc8d7d2d4a6"

type bridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireHeader(v bridgeWireHeader) error {
	if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireHeader.magic offset=%d: %w", w.Len(), err)
	}
	if err := w.u32(int(v.version)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireHeader.version offset=%d: %w", w.Len(), err)
	}
	if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireHeader.schemaHash offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireHeader() (bridgeWireHeader, error) {
	var v bridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

var bridgeTagsRegister = []string{"RAX", "RBX", "RCX", "RDX", "RSI", "RDI", "RBP", "RSP", "R8", "R9", "R10", "R11", "R12", "R13", "R14", "R15"}
var bridgeTagsOperandKind = []string{"", "vreg", "preg", "imm", "slot", "mem", "block"}
var bridgeTagsOpcode = []string{"", "MOV", "LOAD", "STORE", "LEA", "ADD", "SUB", "IMUL", "UMUL", "CMP", "TEST", "SETCC", "TRAP", "JMP", "JCC", "RET"}
var bridgeTagsCondition = []string{"", "E", "NE", "L", "LE", "G", "GE", "B", "BE", "A", "AE", "O", "C"}

func (w *machineBridgeWriter) writeWireSpan(v Span) error {
	if err := w.i32(int(v.Line)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireSpan.Line offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Column)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireSpan.Column offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireSpan() (Span, error) {
	var v Span
	var err error
	v.Line, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireSpan.Line offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Column, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireSpan.Column offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineFrame(v MachineFrame) error {
	if err := w.i32(int(v.LocalSize)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.LocalSize offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Alignment)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.Alignment offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.ShadowSpace)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.ShadowSpace offset=%d: %w", w.Len(), err)
	}
	if err := w.bool(v.HasCalls); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.HasCalls offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineFrame() (MachineFrame, error) {
	var v MachineFrame
	var err error
	v.LocalSize, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.LocalSize offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Alignment, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.Alignment offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ShadowSpace, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.ShadowSpace offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.HasCalls, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFrame.HasCalls offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineOperand(v MachineOperand) error {
	if err := w.i32(int(v.ID)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.ID offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Width)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Width offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Base)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Base offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.BaseSlot)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.BaseSlot offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Index)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Index offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Scale)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Scale offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Disp)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Disp offset=%d: %w", w.Len(), err)
	}
	if err := w.tag(bridgeTagsOperandKind, v.Kind); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Kind offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Literal)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Literal offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Region)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Region offset=%d: %w", w.Len(), err)
	}
	if err := w.bool(v.Signed); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Signed offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineOperand() (MachineOperand, error) {
	var v MachineOperand
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Width, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Width offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Base, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Base offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.BaseSlot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.BaseSlot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Index, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Index offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scale, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Scale offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Disp, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Disp offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Kind, err = r.tag(bridgeTagsOperandKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Kind offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Literal, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Literal offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Region, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Region offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Signed, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineOperand.Signed offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineArg(v MachineArg) error {
	if err := w.str(string(v.Type)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Type offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Index)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Index offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Width)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Width offset=%d: %w", w.Len(), err)
	}
	if err := w.bool(v.Indirect); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Indirect offset=%d: %w", w.Len(), err)
	}
	if err := w.enum(int(v.Register), len(bridgeTagsRegister)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Register offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.StackOffset)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.StackOffset offset=%d: %w", w.Len(), err)
	}
	if err := w.bool(v.OnStack); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.OnStack offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.VReg)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.VReg offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineArg() (MachineArg, error) {
	var v MachineArg
	var err error
	{
		x, e := r.str()
		err = e
		v.Type = LIRType(x)
	}
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Index, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Index offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Width, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Width offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Indirect, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Indirect offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	{
		x, e := r.enum(len(bridgeTagsRegister))
		err = e
		v.Register = MachineReg(x)
	}
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.Register offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.StackOffset, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.StackOffset offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.OnStack, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.OnStack offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.VReg, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineArg.VReg offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineVReg(v MachineVReg) error {
	if err := w.i32(int(v.ID)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineVReg.ID offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Width)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineVReg.Width offset=%d: %w", w.Len(), err)
	}
	if err := w.bool(v.Address); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineVReg.Address offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineVReg() (MachineVReg, error) {
	var v MachineVReg
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineVReg.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Width, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineVReg.Width offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Address, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineVReg.Address offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineStackSlot(v MachineStackSlot) error {
	if err := w.i32(int(v.ID)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.ID offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Size)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.Size offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Align)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.Align offset=%d: %w", w.Len(), err)
	}
	if err := w.bool(v.IncomingIndirect); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.IncomingIndirect offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.BaseVReg)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.BaseVReg offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Source)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.Source offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineStackSlot() (MachineStackSlot, error) {
	var v MachineStackSlot
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Size, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.Size offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Align, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.Align offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.IncomingIndirect, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.IncomingIndirect offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.BaseVReg, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.BaseVReg offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineStackSlot.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineInstruction(v MachineInstruction) error {
	if err := w.tag(bridgeTagsOpcode, v.Op); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Op offset=%d: %w", w.Len(), err)
	}
	if err := w.writeWireMachineOperand(v.Dst); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Dst offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
		if err := w.u32(len(v.Src)); err != nil {
			return err
		}
		for _, element := range v.Src {
			if err := w.writeWireMachineOperand(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Src offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.Width)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Width offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.FlagsDef)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.FlagsDef offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.FlagsUse)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.FlagsUse offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.LIRBlock)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.LIRBlock offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.LIRInstruction)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.LIRInstruction offset=%d: %w", w.Len(), err)
	}
	if err := w.tag(bridgeTagsCondition, v.Cond); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Cond offset=%d: %w", w.Len(), err)
	}
	if err := w.writeWireSpan(v.Source); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Source offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Decision)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Decision offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
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
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Facts offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineInstruction() (MachineInstruction, error) {
	var v MachineInstruction
	var err error
	v.Op, err = r.tag(bridgeTagsOpcode)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Op offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Dst, err = r.readWireMachineOperand()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Dst offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Src, err = func() ([]MachineOperand, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineOperand, n)
		for i := range v {
			v[i], err = r.readWireMachineOperand()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Src offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Width, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Width offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.FlagsDef, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.FlagsDef offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.FlagsUse, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.FlagsUse offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRBlock, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.LIRBlock offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRInstruction, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.LIRInstruction offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Cond, err = r.tag(bridgeTagsCondition)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Cond offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Decision, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Decision offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineInstruction.Facts offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineTerminator(v MachineTerminator) error {
	if err := w.tag(bridgeTagsOpcode, v.Op); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.Op offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.True)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.True offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.False)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.False offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.FlagsUse)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.FlagsUse offset=%d: %w", w.Len(), err)
	}
	if err := w.tag(bridgeTagsCondition, v.Cond); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.Cond offset=%d: %w", w.Len(), err)
	}
	if err := w.writeWireSpan(v.Source); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.Source offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineTerminator() (MachineTerminator, error) {
	var v MachineTerminator
	var err error
	v.Op, err = r.tag(bridgeTagsOpcode)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.Op offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.True, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.True offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.False, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.False offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.FlagsUse, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.FlagsUse offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Cond, err = r.tag(bridgeTagsCondition)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.Cond offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineTerminator.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineBlock(v MachineBlock) error {
	if err := w.i32(int(v.ID)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.ID offset=%d: %w", w.Len(), err)
	}
	if err := w.i32(int(v.LIRBlock)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.LIRBlock offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
		if err := w.u32(len(v.Instructions)); err != nil {
			return err
		}
		for _, element := range v.Instructions {
			if err := w.writeWireMachineInstruction(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.Instructions offset=%d: %w", w.Len(), err)
	}
	if err := w.writeWireMachineTerminator(v.Term); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.Term offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineBlock() (MachineBlock, error) {
	var v MachineBlock
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LIRBlock, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.LIRBlock offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Instructions, err = func() ([]MachineInstruction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineInstruction, n)
		for i := range v {
			v[i], err = r.readWireMachineInstruction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.Instructions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Term, err = r.readWireMachineTerminator()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineBlock.Term offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineFunction(v MachineFunction) error {
	if err := w.str(string(v.Identity)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Identity offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Name)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Name offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Target)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Target offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.ABI)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.ABI offset=%d: %w", w.Len(), err)
	}
	if err := w.str(string(v.Result)); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Result offset=%d: %w", w.Len(), err)
	}
	if err := w.writeWireSpan(v.Source); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Source offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
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
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Facts offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
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
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Decisions offset=%d: %w", w.Len(), err)
	}
	if err := w.writeWireMachineFrame(v.Frame); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Frame offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
		if err := w.u32(len(v.Args)); err != nil {
			return err
		}
		for _, element := range v.Args {
			if err := w.writeWireMachineArg(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Args offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
		if err := w.u32(len(v.VRegs)); err != nil {
			return err
		}
		for _, element := range v.VRegs {
			if err := w.writeWireMachineVReg(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.VRegs offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
		if err := w.u32(len(v.Slots)); err != nil {
			return err
		}
		for _, element := range v.Slots {
			if err := w.writeWireMachineStackSlot(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Slots offset=%d: %w", w.Len(), err)
	}
	if err := func() error {
		if err := w.u32(len(v.Blocks)); err != nil {
			return err
		}
		for _, element := range v.Blocks {
			if err := w.writeWireMachineBlock(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Blocks offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineFunction() (MachineFunction, error) {
	var v MachineFunction
	var err error
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Target, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Target offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ABI, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.ABI offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	{
		x, e := r.str()
		err = e
		v.Result = LIRType(x)
	}
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Result offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Facts offset=%d: %w", r.Size()-int64(r.Len()), err)
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Decisions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Frame, err = r.readWireMachineFrame()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Frame offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Args, err = func() ([]MachineArg, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineArg, n)
		for i := range v {
			v[i], err = r.readWireMachineArg()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Args offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.VRegs, err = func() ([]MachineVReg, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineVReg, n)
		for i := range v {
			v[i], err = r.readWireMachineVReg()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.VRegs offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Slots, err = func() ([]MachineStackSlot, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineStackSlot, n)
		for i := range v {
			v[i], err = r.readWireMachineStackSlot()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Slots offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Blocks, err = func() ([]MachineBlock, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineBlock, n)
		for i := range v {
			v[i], err = r.readWireMachineBlock()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineFunction.Blocks offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
func (w *machineBridgeWriter) writeWireMachineModule(v MachineModule) error {
	if err := func() error {
		if err := w.u32(len(v.Functions)); err != nil {
			return err
		}
		for _, element := range v.Functions {
			if err := w.writeWireMachineFunction(element); err != nil {
				return err
			}
		}
		return nil
	}(); err != nil {
		return fmt.Errorf("MIR_BRIDGE_FIELD WireMachineModule.Functions offset=%d: %w", w.Len(), err)
	}
	return nil
}
func (r machineBridgeReader) readWireMachineModule() (MachineModule, error) {
	var v MachineModule
	var err error
	v.Functions, err = func() ([]MachineFunction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]MachineFunction, n)
		for i := range v {
			v[i], err = r.readWireMachineFunction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireMachineModule.Functions offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
