package concept

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

func projectFallibleMachine(lir LIRModule, base PatternModule, ids map[LIRType]int) (MachineModule, error) {
	out := FallibleModule{Base: base}
	flowBase := 0
	for fi, f := range lir.Functions {
		if c := f.Fallible; c != nil {
			tb, hb := len(out.Tries), len(out.Handlers)
			for _, t := range c.Tries {
				out.Tries = append(out.Tries, FallibleTry{Function: fi, Scope: t.Scope, Join: t.Join})
			}
			for _, h := range c.Handlers {
				out.Handlers = append(out.Handlers, FallibleHandler{Try: tb + h.Try, Type: ids[h.Type], InputSlot: h.InputSlot, BindingSlot: h.BindingSlot, Entry: h.Entry, Scope: h.Scope})
			}
			for _, s := range c.Sites {
				h := -1
				if s.Handler >= 0 {
					h = hb + s.Handler
				}
				out.Sites = append(out.Sites, FallibleSite{Flow: flowBase + s.Flow, Kind: s.Kind, Handler: h})
			}
		}
		flowBase += len(f.Patterns)
	}
	m := MachineModule{Fallible: &out}
	return m, VerifyMachineIR(m)
}

func fallibleControls(m FallibleModule) (map[int]*LIRFallibleContract, error) {
	bad := func() (map[int]*LIRFallibleContract, error) { return nil, fmt.Errorf("MIR_CMIRAMD10_CONTROL") }
	if len(m.Sites) < 1 || len(m.Sites) > 8192 || len(m.Tries) > 8192 || len(m.Handlers) > 8192 {
		return bad()
	}
	objects := m.Base.Base.Base.Objects
	out := map[int]*LIRFallibleContract{}
	get := func(fi int) *LIRFallibleContract {
		if out[fi] == nil {
			out[fi] = &LIRFallibleContract{}
		}
		return out[fi]
	}
	tlocal, hlocal, flocal := map[int]int{}, map[int]int{}, map[int]int{}
	prior := -1
	for i, t := range m.Tries {
		if t.Function < 0 || t.Function >= len(objects.Functions) || t.Function < prior {
			return bad()
		}
		prior = t.Function
		c := get(t.Function)
		tlocal[i] = len(c.Tries)
		c.Tries = append(c.Tries, LIRTryRegion{t.Scope, t.Join})
	}
	for i, h := range m.Handlers {
		if h.Try < 0 || h.Try >= len(m.Tries) || h.Type < 0 || h.Type >= len(objects.Types) {
			return bad()
		}
		c := get(m.Tries[h.Try].Function)
		hlocal[i] = len(c.Handlers)
		c.Handlers = append(c.Handlers, LIRFailureHandler{Try: tlocal[h.Try], Type: LIRType(objects.Types[h.Type].LayoutIdentity), InputSlot: h.InputSlot, BindingSlot: h.BindingSlot, Entry: h.Entry, Scope: h.Scope})
	}
	counts := map[int]int{}
	for i, f := range m.Base.Flows {
		flocal[i] = counts[f.Function]
		counts[f.Function]++
	}
	seen := map[int]bool{}
	for _, s := range m.Sites {
		if s.Flow < 0 || s.Flow >= len(m.Base.Flows) || seen[s.Flow] {
			return bad()
		}
		seen[s.Flow] = true
		fi := m.Base.Flows[s.Flow].Function
		if fi < 0 || fi >= len(objects.Functions) {
			return bad()
		}
		h := -1
		if s.Handler >= 0 {
			if s.Handler >= len(m.Handlers) || m.Tries[m.Handlers[s.Handler].Try].Function != fi {
				return bad()
			}
			h = hlocal[s.Handler]
		} else if s.Handler != -1 {
			return bad()
		}
		c := get(fi)
		c.Sites = append(c.Sites, LIRFailureSite{Flow: flocal[s.Flow], Kind: s.Kind, Handler: h})
	}
	return out, nil
}
func verifyFallibleMachine(m FallibleModule) error {
	controls, err := fallibleControls(m)
	if err != nil {
		return err
	}
	return verifyPatternMachineControl(m.Base, controls)
}

func encodeFallibleMachineBridge(m FallibleModule) ([]byte, error) {
	w := &machineBridgeWriter{}
	h := fallibleBridgeWireHeader{version: FallibleMachineBridgeVersion}
	copy(h.magic[:], FallibleMachineBridgeSchema)
	hash, _ := hex.DecodeString(FallibleMachineBridgeSchemaHash)
	copy(h.schemaHash[:], hash)
	if err := w.writeWireFallibleHeader(h); err != nil {
		return nil, err
	}
	if err := w.writeWireFallibleModule(m); err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}
func DecodeFallibleMachineBridge(data []byte) (MachineModule, error) {
	r := machineBridgeReader{bytes.NewReader(data)}
	h, err := r.readWireFallibleHeader()
	if err != nil {
		return MachineModule{}, err
	}
	if string(h.magic[:]) != FallibleMachineBridgeSchema || h.version != FallibleMachineBridgeVersion || hex.EncodeToString(h.schemaHash[:]) != FallibleMachineBridgeSchemaHash {
		return MachineModule{}, fmt.Errorf("MIR_CMIRAMD10_IDENTITY")
	}
	d, err := r.readWireFallibleModule()
	if err != nil {
		return MachineModule{}, err
	}
	if r.Len() != 0 {
		return MachineModule{}, fmt.Errorf("MIR_CMIRAMD10_TRAILING")
	}
	m := MachineModule{Fallible: &d}
	return m, VerifyMachineIR(m)
}
