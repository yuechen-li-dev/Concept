// Code generated from checked Standard.Backend.BridgeSchema9; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const PatternMachineBridgeSchema = "CMIRAMD9"
const PatternMachineBridgeVersion = 9
const PatternMachineBridgeSchemaHash = "77ee5fc973f76bd7ae51e4d4f47ca0aaaeefa6354ebec88ac91bb6e0bff98a3c"
const PatternMachineBridgeMaxItems = 1048576
const PatternMachineBridgeHeaderSize = 44

var patternBridgeTagsPatternOperationKind = []string{"test_variant", "require_variant"}

type patternBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWirePatternHeader(v patternBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWirePatternHeader() (patternBridgeWireHeader, error) {
	var v patternBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type PatternOperation struct {
	Function    int
	Block       int
	Instruction int
	Kind        string
}

func (w *machineBridgeWriter) writeWirePatternOperation(v PatternOperation) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Function)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Function offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternOperation", "Function", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Block)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Block offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternOperation", "Block", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Instruction)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Instruction offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternOperation", "Instruction", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(patternBridgeTagsPatternOperationKind, v.Kind); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Kind offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternOperation", "Kind", offset)
	}
	return nil
}
func (r machineBridgeReader) readWirePatternOperation() (PatternOperation, error) {
	var v PatternOperation
	var err error
	v.Function, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Function offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Block, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Block offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Instruction, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Instruction offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Kind, err = r.tag(patternBridgeTagsPatternOperationKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternOperation.Kind offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type PatternBinding struct {
	Slot int
}

func (w *machineBridgeWriter) writeWirePatternBinding(v PatternBinding) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Slot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternBinding.Slot offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternBinding", "Slot", offset)
	}
	return nil
}
func (r machineBridgeReader) readWirePatternBinding() (PatternBinding, error) {
	var v PatternBinding
	var err error
	v.Slot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternBinding.Slot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type PatternArm struct {
	Dispatch int
	Body     int
	Scope    int
	Bindings []PatternBinding
}

func (w *machineBridgeWriter) writeWirePatternArm(v PatternArm) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Dispatch)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Dispatch offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternArm", "Dispatch", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Body)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Body offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternArm", "Body", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Scope)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Scope offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternArm", "Scope", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Bindings) > PatternMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Bindings))
			}
			if err := w.u32(len(v.Bindings)); err != nil {
				return err
			}
			for _, element := range v.Bindings {
				if err := w.writeWirePatternBinding(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Bindings offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternArm", "Bindings", offset)
	}
	return nil
}
func (r machineBridgeReader) readWirePatternArm() (PatternArm, error) {
	var v PatternArm
	var err error
	v.Dispatch, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Dispatch offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Body, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Body offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scope, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Scope offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Bindings, err = func() ([]PatternBinding, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]PatternBinding, n)
		for i := range v {
			v[i], err = r.readWirePatternBinding()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternArm.Bindings offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type PatternFlow struct {
	Function    int
	SubjectSlot int
	ResultSlot  int
	Join        int
	Scope       int
	Arms        []PatternArm
}

func (w *machineBridgeWriter) writeWirePatternFlow(v PatternFlow) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Function)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Function offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternFlow", "Function", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.SubjectSlot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.SubjectSlot offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternFlow", "SubjectSlot", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.ResultSlot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.ResultSlot offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternFlow", "ResultSlot", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Join)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Join offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternFlow", "Join", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Scope)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Scope offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternFlow", "Scope", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Arms) > PatternMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Arms))
			}
			if err := w.u32(len(v.Arms)); err != nil {
				return err
			}
			for _, element := range v.Arms {
				if err := w.writeWirePatternArm(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Arms offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternFlow", "Arms", offset)
	}
	return nil
}
func (r machineBridgeReader) readWirePatternFlow() (PatternFlow, error) {
	var v PatternFlow
	var err error
	v.Function, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Function offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.SubjectSlot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.SubjectSlot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.ResultSlot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.ResultSlot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Join, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Join offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scope, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Scope offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Arms, err = func() ([]PatternArm, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]PatternArm, n)
		for i := range v {
			v[i], err = r.readWirePatternArm()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternFlow.Arms offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type PatternModule struct {
	Base       TaggedModule
	Enums      []EnumDomain
	Operations []PatternOperation
	Flows      []PatternFlow
}

func (w *machineBridgeWriter) writeWirePatternModule(v PatternModule) error {
	{
		offset := w.Len()
		if err := w.writeWireTaggedModule(v.Base); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Base offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternModule", "Base", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Enums) > PatternMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Enums))
			}
			if err := w.u32(len(v.Enums)); err != nil {
				return err
			}
			for _, element := range v.Enums {
				if err := w.writeWireEnumDomain(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Enums offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternModule", "Enums", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Operations) > PatternMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Operations))
			}
			if err := w.u32(len(v.Operations)); err != nil {
				return err
			}
			for _, element := range v.Operations {
				if err := w.writeWirePatternOperation(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Operations offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternModule", "Operations", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Flows) > PatternMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Flows))
			}
			if err := w.u32(len(v.Flows)); err != nil {
				return err
			}
			for _, element := range v.Flows {
				if err := w.writeWirePatternFlow(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Flows offset=%d: %w", offset, err)
		}
		w.traceField("WirePatternModule", "Flows", offset)
	}
	return nil
}
func (r machineBridgeReader) readWirePatternModule() (PatternModule, error) {
	var v PatternModule
	var err error
	v.Base, err = r.readWireTaggedModule()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Base offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Enums, err = func() ([]EnumDomain, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]EnumDomain, n)
		for i := range v {
			v[i], err = r.readWireEnumDomain()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Enums offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Operations, err = func() ([]PatternOperation, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]PatternOperation, n)
		for i := range v {
			v[i], err = r.readWirePatternOperation()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Operations offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Flows, err = func() ([]PatternFlow, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]PatternFlow, n)
		for i := range v {
			v[i], err = r.readWirePatternFlow()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WirePatternModule.Flows offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
