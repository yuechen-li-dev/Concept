package concept

import "fmt"

func verifyLIRFallible(f LIRFunction, m LIRModule) (map[int]bool, map[int]int, error) {
	guards, cuts := map[int]bool{}, map[int]int{}
	bad := func() (map[int]bool, map[int]int, error) { return nil, nil, fmt.Errorf("LIR_FALLIBLE_CONTROL") }
	c := f.Fallible
	if c == nil {
		return guards, cuts, nil
	}
	if f.Borrow == nil || len(f.Borrow.SlotScopes) != len(f.Slots) || len(f.Borrow.Scopes) == 0 || len(c.Tries) > 64 || len(c.Handlers) > 64 || len(c.Sites) > 64 || len(c.Sites) == 0 {
		return bad()
	}
	validSlot := func(s int) bool { return s >= 0 && s < len(f.Slots) }
	validBlock := func(s int) bool { return s >= 0 && s < len(f.Blocks) }
	validScope := func(s int) bool { return s > 0 && s < len(f.Borrow.Scopes) }
	seenScopes := map[int]bool{}
	for _, t := range c.Tries {
		if !validScope(t.Scope) || !validBlock(t.Join) || seenScopes[t.Scope] {
			return bad()
		}
		used := false
		for _, h := range c.Handlers {
			if h.Try >= 0 && h.Try < len(c.Tries) && c.Tries[h.Try] == t {
				used = true
			}
		}
		if !used {
			return bad()
		}
		seenScopes[t.Scope] = true
		cuts[t.Join] = f.Borrow.Scopes[t.Scope].Parent
	}
	seenSlots, entries := map[int]bool{}, map[int]bool{}
	for _, h := range c.Handlers {
		if h.Try < 0 || h.Try >= len(c.Tries) || !validSlot(h.InputSlot) || !validSlot(h.BindingSlot) || h.InputSlot == h.BindingSlot || !validBlock(h.Entry) || !validScope(h.Scope) || seenSlots[h.InputSlot] || seenSlots[h.BindingSlot] || entries[h.Entry] {
			return bad()
		}
		seenSlots[h.InputSlot], seenSlots[h.BindingSlot], entries[h.Entry] = true, true, true
		t := c.Tries[h.Try]
		parent := f.Borrow.Scopes[t.Scope].Parent
		if f.Borrow.Scopes[h.Scope].Parent != parent || f.Borrow.SlotScopes[h.InputSlot] != parent || f.Borrow.SlotScopes[h.BindingSlot] != h.Scope || f.Slots[h.InputSlot].Type != h.Type || f.Slots[h.BindingSlot].Type != h.Type || !f.Slots[h.BindingSlot].Readonly {
			return bad()
		}
		in := f.Blocks[h.Entry].Instructions
		if len(in) < 2 || in[0].Op != "load_slot" || in[0].Slot != h.InputSlot || in[0].Scope != h.Scope || in[1].Op != "initialize_slot" || in[1].Slot != h.BindingSlot || in[1].Scope != h.Scope || len(in[1].Args) != 1 || in[1].Args[0] != in[0].Result {
			return bad()
		}
		for _, p := range c.Handlers {
			if p.Entry != h.Entry && p.Try == h.Try && p.Type == h.Type {
				return bad()
			}
		}
		cuts[h.Entry] = h.Scope
	}
	domains := map[LIRType]LIRTaggedEnum{}
	for _, d := range m.TaggedEnums {
		domains[d.Type] = d
	}
	seenFlows := map[int]bool{}
	routes := map[int]int{}
	for _, site := range c.Sites {
		if site.Flow < 0 || site.Flow >= len(f.Patterns) || seenFlows[site.Flow] {
			return bad()
		}
		seenFlows[site.Flow] = true
		flow := f.Patterns[site.Flow]
		if !validSlot(flow.SubjectSlot) {
			return bad()
		}
		d, ok := domains[f.Slots[flow.SubjectSlot].Type]
		if !ok || len(d.Variants) != 2 {
			return bad()
		}
		if site.Kind == "require_success" {
			if site.Handler != -1 || len(flow.Arms) != 1 {
				return bad()
			}
			guards[site.Flow] = true
			continue
		}
		if site.Kind != "propagate" && site.Kind != "remap" || len(flow.Arms) != 2 || !validBlock(flow.Arms[1].Body) {
			return bad()
		}
		arm := flow.Arms[1]
		block := f.Blocks[arm.Body]
		if site.Handler >= 0 {
			if site.Kind != "propagate" || site.Handler >= len(c.Handlers) || len(d.Variants[1].Fields) != 1 || len(arm.Bindings) != 1 {
				return bad()
			}
			h := c.Handlers[site.Handler]
			et := d.Variants[1].Fields[0].Type
			if h.Type != et || !borrowScopeContains(f.Borrow, c.Tries[h.Try].Scope, flow.Scope) {
				return bad()
			}
			for _, other := range c.Handlers {
				if other.Type == et && borrowScopeContains(f.Borrow, c.Tries[other.Try].Scope, flow.Scope) && other.Try != h.Try && borrowScopeContains(f.Borrow, c.Tries[h.Try].Scope, c.Tries[other.Try].Scope) {
					return bad()
				}
			}
			in := block.Instructions
			if len(in) != 5 || in[3].Op != "load_slot" || in[3].Slot != arm.Bindings[0] || in[4].Op != "initialize_slot" || in[4].Slot != h.InputSlot || len(in[4].Args) != 1 || in[4].Args[0] != in[3].Result || in[4].Type != et || block.Term.Op != "jump" || block.Term.True != h.Entry {
				return bad()
			}
			routes[arm.Body] = site.Handler
		} else {
			if site.Handler != -1 || block.Term.Op != "return" {
				return bad()
			}
			ret, ok := domains[f.Result]
			if !ok || len(ret.Variants) != 2 || len(ret.Variants[1].Fields) != len(d.Variants[1].Fields) {
				return bad()
			}
			var returned *LIRInstruction
			for ii := range block.Instructions {
				if block.Instructions[ii].Result == block.Term.Value {
					returned = &block.Instructions[ii]
				}
			}
			if returned == nil || returned.Op != "make_tagged" || returned.Type != f.Result || returned.Member != 1 || len(returned.Args) != len(ret.Variants[1].Fields) {
				return bad()
			}
			if site.Kind == "propagate" && len(returned.Args) == 1 {
				if len(arm.Bindings) != 1 {
					return bad()
				}
				found := false
				for _, in := range block.Instructions {
					if in.Result == returned.Args[0] && in.Op == "load_slot" && in.Slot == arm.Bindings[0] {
						found = true
					}
				}
				if !found {
					return bad()
				}
			}
			if site.Kind == "propagate" {
				for i, field := range d.Variants[1].Fields {
					if field.Type != ret.Variants[1].Fields[i].Type {
						return bad()
					}
					for _, h := range c.Handlers {
						if h.Type == field.Type && borrowScopeContains(f.Borrow, c.Tries[h.Try].Scope, flow.Scope) {
							return bad()
						}
					}
				}
			}
		}
	}
	for hi, h := range c.Handlers {
		used := false
		for bi, b := range f.Blocks {
			for _, to := range patternSuccessors(b) {
				if to == h.Entry {
					route, ok := routes[bi]
					if !ok || route != hi {
						return bad()
					}
					used = true
				}
			}
			for ii, in := range b.Instructions {
				if in.Slot == h.InputSlot && (in.Op == "load_slot" || in.Op == "initialize_slot" || in.Op == "store_slot" || in.Op == "slot_address") {
					if bi == h.Entry && ii == 0 {
						continue
					}
					r, ok := routes[bi]
					if !ok || r != hi || ii != 4 || in.Op != "initialize_slot" {
						return bad()
					}
				}
			}
		}
		if !used {
			return bad()
		}
	}
	return guards, cuts, nil
}
