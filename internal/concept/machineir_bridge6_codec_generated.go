// Code generated from checked Standard.Backend.BridgeSchema6; DO NOT EDIT.
package concept

import (
	"fmt"
	"io"
)

const BorrowMachineBridgeSchema = "CMIRAMD6"
const BorrowMachineBridgeVersion = 6
const BorrowMachineBridgeSchemaHash = "67d71de72ece8b4b6ce438840543e6220fd3ddcc035e2142147c679b4c77e73d"
const BorrowMachineBridgeMaxItems = 1048576
const BorrowMachineBridgeHeaderSize = 44

var borrowBridgeTagsBorrowRoot = []string{"local", "parameter"}
var borrowBridgeTagsBorrowEvidence = []string{"checked_place", "checked_borrow", "checked_parameter"}
var borrowBridgeTagsBorrowEscape = []string{"noescape"}

type borrowBridgeWireHeader struct {
	magic      [8]byte
	version    uint32
	schemaHash [32]byte
}

func (w *machineBridgeWriter) writeWireBorrowHeader(v borrowBridgeWireHeader) error {
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.magic[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowHeader.magic offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowHeader", "magic", offset)
	}
	{
		offset := w.Len()
		if err := w.u32(int(v.version)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowHeader.version offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowHeader", "version", offset)
	}
	{
		offset := w.Len()
		if err := func() error { _, err := w.Write(v.schemaHash[:]); return err }(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowHeader.schemaHash offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowHeader", "schemaHash", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireBorrowHeader() (borrowBridgeWireHeader, error) {
	var v borrowBridgeWireHeader
	var err error
	v.magic, err = func() ([8]byte, error) { var v [8]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowHeader.magic offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.version, err = r.u32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowHeader.version offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.schemaHash, err = func() ([32]byte, error) { var v [32]byte; _, err := io.ReadFull(r.Reader, v[:]); return v, err }()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowHeader.schemaHash offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type BorrowScope struct {
	ID     int
	Parent int
	Source ObjectSpan
}

func (w *machineBridgeWriter) writeWireBorrowScope(v BorrowScope) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.ID)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowScope.ID offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowScope", "ID", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Parent)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowScope.Parent offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowScope", "Parent", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireObjectSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowScope.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowScope", "Source", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireBorrowScope() (BorrowScope, error) {
	var v BorrowScope
	var err error
	v.ID, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowScope.ID offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Parent, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowScope.Parent offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireObjectSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowScope.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type BorrowReference struct {
	Value          int
	Target         int
	SemanticTarget string
	LayoutIdentity string
	RootKind       string
	Root           int
	Parent         int
	Mutable        bool
	Lifetime       int
	Evidence       string
	Provenance     string
	Escape         string
	Space          string
	Source         ObjectSpan
}

func (w *machineBridgeWriter) writeWireBorrowReference(v BorrowReference) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Value)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Value offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Value", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Target)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Target offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Target", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.SemanticTarget)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.SemanticTarget offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "SemanticTarget", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.LayoutIdentity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.LayoutIdentity offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "LayoutIdentity", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(borrowBridgeTagsBorrowRoot, v.RootKind); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.RootKind offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "RootKind", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Root)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Root offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Root", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Parent)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Parent offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Parent", offset)
	}
	{
		offset := w.Len()
		if err := w.bool(v.Mutable); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Mutable offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Mutable", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Lifetime)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Lifetime offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Lifetime", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(borrowBridgeTagsBorrowEvidence, v.Evidence); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Evidence offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Evidence", offset)
	}
	{
		offset := w.Len()
		if err := w.str(string(v.Provenance)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Provenance offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Provenance", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(borrowBridgeTagsBorrowEscape, v.Escape); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Escape offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Escape", offset)
	}
	{
		offset := w.Len()
		if err := w.tag(objectBridgeTagsObjectAddressSpace, v.Space); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Space offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Space", offset)
	}
	{
		offset := w.Len()
		if err := w.writeWireObjectSpan(v.Source); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Source offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowReference", "Source", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireBorrowReference() (BorrowReference, error) {
	var v BorrowReference
	var err error
	v.Value, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Value offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Target, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Target offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.SemanticTarget, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.SemanticTarget offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.LayoutIdentity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.LayoutIdentity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.RootKind, err = r.tag(borrowBridgeTagsBorrowRoot)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.RootKind offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Root, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Root offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Parent, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Parent offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Mutable, err = r.boolean()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Mutable offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Lifetime, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Lifetime offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Evidence, err = r.tag(borrowBridgeTagsBorrowEvidence)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Evidence offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Provenance, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Provenance offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Escape, err = r.tag(borrowBridgeTagsBorrowEscape)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Escape offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Space, err = r.tag(objectBridgeTagsObjectAddressSpace)
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Space offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Source, err = r.readWireObjectSpan()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowReference.Source offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type BorrowSite struct {
	Block       int
	Instruction int
	Scope       int
}

func (w *machineBridgeWriter) writeWireBorrowSite(v BorrowSite) error {
	{
		offset := w.Len()
		if err := w.i32(int(v.Block)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowSite.Block offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowSite", "Block", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Instruction)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowSite.Instruction offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowSite", "Instruction", offset)
	}
	{
		offset := w.Len()
		if err := w.i32(int(v.Scope)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowSite.Scope offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowSite", "Scope", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireBorrowSite() (BorrowSite, error) {
	var v BorrowSite
	var err error
	v.Block, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowSite.Block offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Instruction, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowSite.Instruction offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scope, err = r.i32()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowSite.Scope offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type BorrowFunction struct {
	Identity   string
	Scopes     []BorrowScope
	SlotScopes []int
	References []BorrowReference
	Sites      []BorrowSite
}

func (w *machineBridgeWriter) writeWireBorrowFunction(v BorrowFunction) error {
	{
		offset := w.Len()
		if err := w.str(string(v.Identity)); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.Identity offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowFunction", "Identity", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Scopes) > BorrowMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Scopes))
			}
			if err := w.u32(len(v.Scopes)); err != nil {
				return err
			}
			for _, element := range v.Scopes {
				if err := w.writeWireBorrowScope(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.Scopes offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowFunction", "Scopes", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.SlotScopes) > BorrowMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.SlotScopes))
			}
			if err := w.u32(len(v.SlotScopes)); err != nil {
				return err
			}
			for _, element := range v.SlotScopes {
				if err := w.i32(int(element)); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.SlotScopes offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowFunction", "SlotScopes", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.References) > BorrowMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.References))
			}
			if err := w.u32(len(v.References)); err != nil {
				return err
			}
			for _, element := range v.References {
				if err := w.writeWireBorrowReference(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.References offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowFunction", "References", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Sites) > BorrowMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Sites))
			}
			if err := w.u32(len(v.Sites)); err != nil {
				return err
			}
			for _, element := range v.Sites {
				if err := w.writeWireBorrowSite(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.Sites offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowFunction", "Sites", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireBorrowFunction() (BorrowFunction, error) {
	var v BorrowFunction
	var err error
	v.Identity, err = r.str()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.Identity offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Scopes, err = func() ([]BorrowScope, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]BorrowScope, n)
		for i := range v {
			v[i], err = r.readWireBorrowScope()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.Scopes offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.SlotScopes, err = func() ([]int, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]int, n)
		for i := range v {
			v[i], err = r.i32()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.SlotScopes offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.References, err = func() ([]BorrowReference, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]BorrowReference, n)
		for i := range v {
			v[i], err = r.readWireBorrowReference()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.References offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Sites, err = func() ([]BorrowSite, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]BorrowSite, n)
		for i := range v {
			v[i], err = r.readWireBorrowSite()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowFunction.Sites offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}

type BorrowModule struct {
	Objects   ObjectModule
	Authority []BorrowFunction
}

func (w *machineBridgeWriter) writeWireBorrowModule(v BorrowModule) error {
	{
		offset := w.Len()
		if err := w.writeWireObjectModule(v.Objects); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowModule.Objects offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowModule", "Objects", offset)
	}
	{
		offset := w.Len()
		if err := func() error {
			if len(v.Authority) > BorrowMachineBridgeMaxItems {
				return fmt.Errorf("MIR_BRIDGE_COUNT %d", len(v.Authority))
			}
			if err := w.u32(len(v.Authority)); err != nil {
				return err
			}
			for _, element := range v.Authority {
				if err := w.writeWireBorrowFunction(element); err != nil {
					return err
				}
			}
			return nil
		}(); err != nil {
			return fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowModule.Authority offset=%d: %w", offset, err)
		}
		w.traceField("WireBorrowModule", "Authority", offset)
	}
	return nil
}
func (r machineBridgeReader) readWireBorrowModule() (BorrowModule, error) {
	var v BorrowModule
	var err error
	v.Objects, err = r.readWireObjectModule()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowModule.Objects offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	v.Authority, err = func() ([]BorrowFunction, error) {
		n, err := r.count()
		if err != nil {
			return nil, err
		}
		if n > r.Len() {
			return nil, fmt.Errorf("MIR_BRIDGE_COUNT exceeds remaining input")
		}
		v := make([]BorrowFunction, n)
		for i := range v {
			v[i], err = r.readWireBorrowFunction()
			if err != nil {
				return nil, err
			}
		}
		return v, nil
	}()
	if err != nil {
		return v, fmt.Errorf("MIR_BRIDGE_FIELD WireBorrowModule.Authority offset=%d: %w", r.Size()-int64(r.Len()), err)
	}
	return v, nil
}
