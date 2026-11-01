// Code generated from checked Standard.Backend.BridgeSchema10; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const FallibleMachineBridgeSchema = "CMIRAMDA"
const FallibleMachineBridgeVersion = 10
const FallibleMachineBridgeSchemaHash = "ea57ae74f5ae54ece90ce429edbc4a41ce309d106582c84fdb5d70d15242941d"
const FallibleMachineBridgeMaxItems = 1048576
const FallibleMachineBridgeHeaderSize = 44

var fallibleBridgeTagsFallibleSiteKind = []string{"require_success", "propagate", "remap"}

type fallibleBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireFallibleHeader(v fallibleBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireFallibleHeader() (fallibleBridgeWireHeader, error) {
	var v fallibleBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type FallibleTry struct {
	Function int
	Scope    int
	Join     int
}

func (w *machineBridgeWriter) writeWireFallibleTry(v FallibleTry) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Function)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleTry.Function offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleTry", "Function", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Scope)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleTry.Scope offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleTry", "Scope", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Join)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleTry.Join offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleTry", "Join", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireFallibleTry() (FallibleTry, error) {
	var v FallibleTry
	var err error
	v.Function, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleTry.Function offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scope, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleTry.Scope offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Join, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleTry.Join offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type FallibleHandler struct {
	Try         int
	Type        int
	InputSlot   int
	BindingSlot int
	Entry       int
	Scope       int
}

func (w *machineBridgeWriter) writeWireFallibleHandler(v FallibleHandler) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Try)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Try offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHandler", "Try", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Type)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Type offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHandler", "Type", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.InputSlot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.InputSlot offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHandler", "InputSlot", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.BindingSlot)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.BindingSlot offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHandler", "BindingSlot", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Entry)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Entry offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHandler", "Entry", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Scope)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Scope offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleHandler", "Scope", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireFallibleHandler() (FallibleHandler, error) {
	var v FallibleHandler
	var err error
	v.Try, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Try offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Type, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Type offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.InputSlot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.InputSlot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.BindingSlot, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.BindingSlot offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Entry, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Entry offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scope, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleHandler.Scope offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type FallibleSite struct {
	Flow    int
	Kind    string
	Handler int
}

func (w *machineBridgeWriter) writeWireFallibleSite(v FallibleSite) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Flow)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleSite.Flow offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleSite", "Flow", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(fallibleBridgeTagsFallibleSiteKind, v.Kind); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleSite.Kind offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleSite", "Kind", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Handler)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleSite.Handler offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleSite", "Handler", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireFallibleSite() (FallibleSite, error) {
	var v FallibleSite
	var err error
	v.Flow, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleSite.Flow offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Kind, err = r.tag(fallibleBridgeTagsFallibleSiteKind)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleSite.Kind offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Handler, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleSite.Handler offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type FallibleModule struct {
	Base     PatternModule
	Tries    []FallibleTry
	Handlers []FallibleHandler
	Sites    []FallibleSite
}

func (w *machineBridgeWriter) writeWireFallibleModule(v FallibleModule) error {
	{
		offset := w.Len()
		if err := w.writeWirePatternModule(v.Base); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Base offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleModule", "Base", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Tries) > FallibleMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Tries))
			}
			if err := w.u32(len(v.Tries)); err != nil {
				return err
			}
			for _, element := range v.Tries {
				if err := w.writeWireFallibleTry(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Tries offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleModule", "Tries", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Handlers) > FallibleMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Handlers))
			}
			if err := w.u32(len(v.Handlers)); err != nil {
				return err
			}
			for _, element := range v.Handlers {
				if err := w.writeWireFallibleHandler(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Handlers offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleModule", "Handlers", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Sites) > FallibleMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Sites))
			}
			if err := w.u32(len(v.Sites)); err != nil {
				return err
			}
			for _, element := range v.Sites {
				if err := w.writeWireFallibleSite(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Sites offset=%d: %w", offset, err)
		}
		w.traceField("WireFallibleModule", "Sites", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireFallibleModule() (FallibleModule, error) {
	var v FallibleModule
	var err error
	v.Base, err = r.readWirePatternModule()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Base offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Tries, err = func() ([]FallibleTry, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]FallibleTry, n)
		for i := range v {
			v[i], err = r.readWireFallibleTry()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Tries offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Handlers, err = func() ([]FallibleHandler, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]FallibleHandler, n)
		for i := range v {
			v[i], err = r.readWireFallibleHandler()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Handlers offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Sites, err = func() ([]FallibleSite, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]FallibleSite, n)
		for i := range v {
			v[i], err = r.readWireFallibleSite()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireFallibleModule.Sites offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
