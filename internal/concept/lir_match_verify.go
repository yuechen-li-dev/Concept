package concept

import "fmt"

func patternSuccessors(b LIRBlock) []int {
	switch b.Term.Op {
	case "jump":
		return []int{b.Term.True}
	case "branch":
		return []int{b.Term.True, b.Term.False}
	}
	return nil
}
func verifyLIRPatternFlow(f LIRFunction, m LIRModule) error {
	bad := func(s string) error { return fmt.Errorf("LIR_PATTERN_%s", s) }
	if len(f.Patterns) == 0 {
		for _, b := range f.Blocks {
			for _, in := range b.Instructions {
				if in.Op == "test_variant" || in.Op == "require_variant" {
					return bad("MISSING_FLOW")
				}
			}
		}
		return nil
	}
	if len(f.Patterns) > 64 || f.Borrow == nil || len(f.Blocks) > 128 || len(f.Slots) > 64 {
		return bad("CAPACITY")
	}
	scopes := f.Borrow
	dom := lirDominators(f)
	slot := func(id int) (LIRSlot, bool) {
		if id < 0 || id >= len(f.Slots) || f.Slots[id].ID != id {
			return LIRSlot{}, false
		}
		return f.Slots[id], true
	}
	validBlock := func(id int) bool { return id >= 0 && id < len(f.Blocks) }
	type domain struct {
		count  int
		fields [][]LIRAggregateField
	}
	domains := map[LIRType]domain{}
	for _, d := range m.Enums {
		domains[d.Type] = domain{count: len(d.Variants)}
	}
	for _, d := range m.TaggedEnums {
		v := domain{count: len(d.Variants)}
		for _, a := range d.Variants {
			v.fields = append(v.fields, a.Fields)
		}
		domains[d.Type] = v
	}
	allowed := map[[2]int]bool{}
	roles := map[int]bool{}
	for _, flow := range f.Patterns {
		if flow.Join < -1 || flow.ResultSlot < -1 {
			return bad("JOIN")
		}
		s, ok := slot(flow.SubjectSlot)
		if !ok || !s.Readonly || roles[s.ID] || flow.Scope < 0 || flow.Scope >= len(scopes.Scopes) || scopes.SlotScopes[s.ID] != flow.Scope || len(flow.Arms) == 0 || len(flow.Arms) > 256 {
			return bad("SUBJECT")
		}
		roles[s.ID] = true
		d, isEnum := domains[s.Type]
		if !isEnum && !lirInteger(s.Type) {
			return bad("SUBJECT")
		}
		if isEnum && len(flow.Arms) != d.count {
			return bad("EXHAUSTIVENESS")
		}
		if flow.Join != -1 && !validBlock(flow.Join) {
			return bad("JOIN")
		}
		if flow.ResultSlot >= 0 {
			r, ok := slot(flow.ResultSlot)
			if !ok || roles[r.ID] || flow.Join < 0 || scopes.SlotScopes[r.ID] != flow.Scope {
				return bad("RESULT")
			}
			roles[r.ID] = true
			join := f.Blocks[flow.Join]
			if len(join.Instructions) == 0 || join.Instructions[0].Op != "load_slot" || join.Instructions[0].Slot != r.ID {
				return bad("RESULT")
			}
			for _, block := range f.Blocks {
				for _, in := range block.Instructions {
					if in.Slot == r.ID && (in.Op == "store_slot" || in.Op == "initialize_slot") {
						inside := false
						for _, arm := range flow.Arms {
							inside = inside || borrowScopeContains(scopes, arm.Scope, in.Scope)
						}
						if !inside {
							return bad("RESULT")
						}
					}
				}
			}
		}
		initBlock, initializations := -1, 0
		for bi, b := range f.Blocks {
			for _, in := range b.Instructions {
				if in.Slot == s.ID && (in.Op == "store_slot" || in.Op == "initialize_slot") {
					initializations++
					initBlock = bi
					if in.Op != "initialize_slot" || in.Scope != flow.Scope {
						return bad("SNAPSHOT")
					}
				}
			}
		}
		if initializations != 1 || !validBlock(flow.Arms[0].Dispatch) || !dom[flow.Arms[0].Dispatch][initBlock] {
			return bad("SNAPSHOT")
		}
		seen := map[int]bool{}
		literals := map[string]bool{}
		blocks := map[int]bool{}
		armScopes := map[int]bool{}
		for ai, arm := range flow.Arms {
			if !validBlock(arm.Dispatch) || !validBlock(arm.Body) || blocks[arm.Dispatch] || blocks[arm.Body] || arm.Scope <= 0 || arm.Scope >= len(scopes.Scopes) || armScopes[arm.Scope] || scopes.Scopes[arm.Scope].Parent != flow.Scope {
				return bad("ARM")
			}
			blocks[arm.Dispatch] = true
			blocks[arm.Body] = true
			armScopes[arm.Scope] = true
			dispatch := f.Blocks[arm.Dispatch]
			body := f.Blocks[arm.Body]
			last := ai == len(flow.Arms)-1
			variant := -1
			expectedPred := initBlock
			if ai > 0 {
				expectedPred = flow.Arms[ai-1].Dispatch
			}
			for bi, b := range f.Blocks {
				for _, to := range patternSuccessors(b) {
					if to == arm.Body && bi != arm.Dispatch || to == arm.Dispatch && bi != expectedPred {
						return bad("ARM_EDGE")
					}
				}
			}
			if dispatch.Term.True != arm.Body {
				return bad("ARM_EDGE")
			}
			if last {
				if dispatch.Term.Op != "jump" {
					return bad("ARM_EDGE")
				}
			} else if dispatch.Term.Op != "branch" || dispatch.Term.False != flow.Arms[ai+1].Dispatch {
				return bad("ARM_EDGE")
			}
			if isEnum {
				if len(dispatch.Instructions) != 2 {
					return bad("DISPATCH")
				}
				load, test := dispatch.Instructions[0], dispatch.Instructions[1]
				want := "test_variant"
				if last {
					want = "require_variant"
				}
				if load.Op != "load_slot" || load.Slot != s.ID || load.Type != s.Type || test.Op != want || len(test.Args) != 1 || test.Args[0] != load.Result || test.Aggregate != s.Type || load.Scope != flow.Scope || test.Scope != flow.Scope || load.Source != test.Source || test.Member < 0 || test.Member >= d.count || seen[test.Member] {
					return bad("DISPATCH")
				}
				if !last && dispatch.Term.Value != test.Result {
					return bad("DISPATCH")
				}
				variant = test.Member
				seen[variant] = true
				allowed[[2]int{arm.Dispatch, 1}] = true
			} else if last {
				if len(dispatch.Instructions) != 0 {
					return bad("WILDCARD")
				}
			} else {
				if len(dispatch.Instructions) != 3 {
					return bad("LITERAL")
				}
				load, literal, test := dispatch.Instructions[0], dispatch.Instructions[1], dispatch.Instructions[2]
				if load.Op != "load_slot" || load.Slot != s.ID || literal.Op != "const" || literal.Type != s.Type || test.Op != "eq" || len(test.Args) != 2 || test.Args[0] != load.Result || test.Args[1] != literal.Result || dispatch.Term.Value != test.Result || literals[literal.Literal] {
					return bad("LITERAL")
				}
				literals[literal.Literal] = true
			}
			var fields []LIRAggregateField
			if isEnum && len(d.fields) > 0 {
				fields = d.fields[variant]
			}
			if len(arm.Bindings) != len(fields) {
				return bad("BINDING")
			}
			if len(fields) > 0 {
				if len(body.Instructions) < 1+2*len(fields) {
					return bad("BINDING")
				}
				load := body.Instructions[0]
				if load.Op != "load_slot" || load.Slot != s.ID || load.Type != s.Type || load.Scope != arm.Scope {
					return bad("BINDING")
				}
				for fi, field := range fields {
					id := arm.Bindings[fi]
					binding, ok := slot(id)
					if !ok || roles[id] || binding.Type != field.Type || scopes.SlotScopes[id] != arm.Scope {
						return bad("BINDING")
					}
					roles[id] = true
					read, init := body.Instructions[1+fi*2], body.Instructions[2+fi*2]
					if read.Op != "checked_payload" || read.Aggregate != s.Type || read.Member != variant || read.Extent != fi || read.Type != field.Type || len(read.Args) != 1 || read.Args[0] != load.Result || read.Scope != arm.Scope || init.Op != "initialize_slot" || init.Slot != id || len(init.Args) != 1 || init.Args[0] != read.Result || init.Scope != arm.Scope {
						return bad("BINDING")
					}
					allowed[[2]int{arm.Body, 1 + fi*2}] = true
				}
			}
			region := map[int]bool{}
			queue := []int{arm.Body}
			for len(queue) > 0 {
				bi := queue[0]
				queue = queue[1:]
				if bi == flow.Join || region[bi] {
					continue
				}
				if !validBlock(bi) {
					return bad("ARM_EDGE")
				}
				for _, other := range flow.Arms {
					if bi == other.Dispatch || bi == other.Body && bi != arm.Body {
						return bad("ARM_EDGE")
					}
				}
				region[bi] = true
				queue = append(queue, patternSuccessors(f.Blocks[bi])...)
			}
			for bi, block := range f.Blocks {
				for _, in := range block.Instructions {
					inArm := borrowScopeContains(scopes, arm.Scope, in.Scope)
					bindingUse := false
					for _, id := range arm.Bindings {
						bindingUse = bindingUse || (in.Slot == id)
					}
					if (inArm || bindingUse) && (!region[bi] || !inArm) {
						return bad("BINDING_SCOPE")
					}
				}
			}
		}
	}
	for bi, b := range f.Blocks {
		for ii, in := range b.Instructions {
			if in.Op == "checked_payload" || in.Op == "test_variant" || in.Op == "require_variant" {
				if !allowed[[2]int{bi, ii}] {
					return bad("ACCESS_AUTHORITY")
				}
			}
			if in.Op == "load_slot" || in.Op == "store_slot" || in.Op == "initialize_slot" || in.Op == "slot_address" {
				if in.Slot < 0 || in.Slot >= len(f.Slots) || !borrowScopeContains(scopes, scopes.SlotScopes[in.Slot], in.Scope) {
					return bad("SLOT_SCOPE")
				}
			}
		}
	}
	return verifyPatternSlotInitialization(f)
}

func verifyPatternSlotInitialization(f LIRFunction) error {
	n := len(f.Blocks)
	if n == 0 {
		return fmt.Errorf("LIR_PATTERN_CFG")
	}
	reachable := make([]bool, n)
	queue := []int{0}
	for len(queue) > 0 {
		b := queue[0]
		queue = queue[1:]
		if b < 0 || b >= n {
			return fmt.Errorf("LIR_PATTERN_CFG")
		}
		if reachable[b] {
			continue
		}
		reachable[b] = true
		queue = append(queue, patternSuccessors(f.Blocks[b])...)
	}
	exits := make([][]bool, n)
	for i := range exits {
		exits[i] = make([]bool, len(f.Slots))
		for s := range exits[i] {
			exits[i][s] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for bi, b := range f.Blocks {
			if !reachable[bi] {
				continue
			}
			live := make([]bool, len(f.Slots))
			for s := range live {
				live[s] = bi != 0
			}
			if bi != 0 {
				for p, prior := range f.Blocks {
					if reachable[p] {
						for _, to := range patternSuccessors(prior) {
							if to == bi {
								for s := range live {
									live[s] = live[s] && exits[p][s]
								}
							}
						}
					}
				}
			}
			for _, in := range b.Instructions {
				if (in.Op == "store_slot" || in.Op == "initialize_slot") && in.Slot >= 0 && in.Slot < len(live) {
					live[in.Slot] = true
				}
			}
			for s := range live {
				if live[s] != exits[bi][s] {
					exits[bi][s] = live[s]
					changed = true
				}
			}
		}
	}
	for bi, b := range f.Blocks {
		if !reachable[bi] {
			continue
		}
		live := make([]bool, len(f.Slots))
		for s := range live {
			live[s] = bi != 0
		}
		if bi != 0 {
			for p, prior := range f.Blocks {
				if reachable[p] {
					for _, to := range patternSuccessors(prior) {
						if to == bi {
							for s := range live {
								live[s] = live[s] && exits[p][s]
							}
						}
					}
				}
			}
		}
		for _, in := range b.Instructions {
			if in.Op == "load_slot" && !live[in.Slot] {
				return fmt.Errorf("LIR_PATTERN_UNINITIALIZED_JOIN")
			}
			if in.Op == "store_slot" || in.Op == "initialize_slot" {
				live[in.Slot] = true
			}
		}
	}
	return nil
}
