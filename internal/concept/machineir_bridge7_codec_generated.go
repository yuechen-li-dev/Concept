// Code generated from checked Standard.Backend.BridgeSchema7; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const EnumMachineBridgeSchema = "CMIRAMD7"
const EnumMachineBridgeVersion = 7
const EnumMachineBridgeSchemaHash = "74992e3d74be0581484be7f3c6341b7e75654d59b76255f075771367858d33c2"
const EnumMachineBridgeMaxItems = 1048576
const EnumMachineBridgeHeaderSize = 44

type enumBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireEnumHeader(v enumBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireEnumHeader() (enumBridgeWireHeader, error) {
	var v enumBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type EnumVariant struct {
	Name string
	Tag  int
}

func (w *machineBridgeWriter) writeWireEnumVariant(v EnumVariant) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Name)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumVariant.Name offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumVariant", "Name", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Tag)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumVariant.Tag offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumVariant", "Tag", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireEnumVariant() (EnumVariant, error) {
	var v EnumVariant
	var err error
	v.Name, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumVariant.Name offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Tag, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumVariant.Tag offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type EnumDomain struct {
	Type           int
	Identity       string
	LayoutIdentity string
	Size           int
	Align          int
	Variants       []EnumVariant
}

func (w *machineBridgeWriter) writeWireEnumDomain(v EnumDomain) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumDomain", "Type", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumDomain", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.LayoutIdentity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.LayoutIdentity offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumDomain", "LayoutIdentity", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Size)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Size offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumDomain", "Size", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Align)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Align offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumDomain", "Align", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Variants) > EnumMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Variants))
			}
			if err := w.u32(len(v.Variants)); err != nil {
				return err
			}
			for _, element := range v.Variants {
				if err := w.writeWireEnumVariant(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Variants offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumDomain", "Variants", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireEnumDomain() (EnumDomain, error) {
	var v EnumDomain
	var err error
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LayoutIdentity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.LayoutIdentity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Size, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Size offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Align, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Align offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Variants, err = func() ([]EnumVariant, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]EnumVariant, n)
		for i := range v {
			v[i], err = r.readWireEnumVariant()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumDomain.Variants offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type EnumModule struct {
	Objects ObjectModule
	Domains []EnumDomain
}

func (w *machineBridgeWriter) writeWireEnumModule(v EnumModule) error {
	{
		offset := w.Len()
		if err := w.writeWireObjectModule(v.Objects); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumModule.Objects offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumModule", "Objects", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Domains) > EnumMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Domains))
			}
			if err := w.u32(len(v.Domains)); err != nil {
				return err
			}
			for _, element := range v.Domains {
				if err := w.writeWireEnumDomain(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireEnumModule.Domains offset=%d: %w", offset, err)
		}
		w.traceField("WireEnumModule", "Domains", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireEnumModule() (EnumModule, error) {
	var v EnumModule
	var err error
	v.Objects, err = r.readWireObjectModule()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumModule.Objects offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Domains, err = func() ([]EnumDomain, error) {
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
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireEnumModule.Domains offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
