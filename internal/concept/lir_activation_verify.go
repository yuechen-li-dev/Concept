package concept

import (
	"fmt"
	"strconv"
)

type activationBlockView struct {
	defs      map[int]LIRInstruction
	positions map[int]int
}

func activationView(block LIRBlock) activationBlockView {
	v := activationBlockView{defs: map[int]LIRInstruction{}, positions: map[int]int{}}
	for i, in := range block.Instructions {
		if in.Result >= 0 {
			v.defs[in.Result] = in
			v.positions[in.Result] = i
		}
	}
	return v
}
func (v activationBlockView) constant(id int, literal string) bool {
	in := v.defs[id]
	return in.Op == "const" && in.Literal == literal
}
func (v activationBlockView) headerAddress(id, field int) bool {
	in := v.defs[id]
	return in.Op == "activation_address" && in.FrameField == field
}
func (v activationBlockView) depth(id int) bool {
	in := v.defs[id]
	return in.Op == "load" && len(in.Args) == 1 && v.headerAddress(in.Args[0], 0)
}
func (v activationBlockView) top(id int) bool {
	in := v.defs[id]
	return in.Op == "sub" && len(in.Args) == 2 && v.depth(in.Args[0]) && v.constant(in.Args[1], "1")
}
func (v activationBlockView) slotLoad(id, offset int, l ActivationStackLayout) bool {
	in := v.defs[id]
	if in.Op != "load" || len(in.Args) != 1 {
		return false
	}
	addr := v.defs[in.Args[0]]
	return addr.Op == "index_address" && len(addr.Args) == 2 && v.top(addr.Args[1]) && addr.FrameOffset == l.SlotsOffset+offset
}
func (v activationBlockView) comparison(block LIRBlock, literal string, offset int, l ActivationStackLayout) bool {
	cmp := v.defs[block.Term.Value]
	return block.Term.Op == "branch" && cmp.Op == "eq" && len(cmp.Args) == 2 && v.slotLoad(cmp.Args[0], offset, l) && v.constant(cmp.Args[1], literal)
}

// Dominance is computed from executable CFG edges. Labels and transfer records
// identify source regions but cannot confer a safety fact on instructions.
func activationDominators(f LIRFunction) ([]map[int]bool, [][]int, error) {
	n := len(f.Blocks)
	pred := make([][]int, n)
	for i, b := range f.Blocks {
		if b.ID != i {
			return nil, nil, fmt.Errorf("LIR_BAD_BLOCK_ID b%d", b.ID)
		}
		targets := []int{}
		switch b.Term.Op {
		case "jump":
			targets = append(targets, b.Term.True)
		case "branch":
			targets = append(targets, b.Term.True, b.Term.False)
		}
		for _, t := range targets {
			if t < 0 || t >= n {
				return nil, nil, fmt.Errorf("LIR_BAD_ACTIVATION_CFG")
			}
			pred[t] = append(pred[t], b.ID)
		}
	}
	dom := make([]map[int]bool, n)
	for i := range dom {
		dom[i] = map[int]bool{}
		if i == 0 {
			dom[i][0] = true
		} else {
			for j := 0; j < n; j++ {
				dom[i][j] = true
			}
		}
	}
	changed := true
	for changed {
		changed = false
		for i := 1; i < n; i++ {
			next := map[int]bool{i: true}
			if len(pred[i]) == 0 {
				return nil, nil, fmt.Errorf("LIR_UNREACHABLE_ACTIVATION_BLOCK b%d", i)
			}
			for k := range dom[pred[i][0]] {
				all := true
				for _, p := range pred[i][1:] {
					if !dom[p][k] {
						all = false
						break
					}
				}
				if all {
					next[k] = true
				}
			}
			if len(next) != len(dom[i]) {
				changed = true
			} else {
				for k := range next {
					if !dom[i][k] {
						changed = true
						break
					}
				}
			}
			dom[i] = next
		}
	}
	return dom, pred, nil
}

func verifyLIRActivationStep(f LIRFunction) error {
	m := f.Activation
	l := m.Layout
	if err := verifyActivationLayout(l); err != nil {
		return err
	}
	if f.Machine != nil || f.Result != "machine_step_result" || len(f.Params) != 1 || f.Params[0].ID != 0 || f.Params[0].Type != LIRType("ptr<activation:"+l.Identity+">") || len(f.Blocks) < 3 || len(m.InitFields) != 2+len(l.SharedFields) {
		return fmt.Errorf("LIR_BAD_ACTIVATION_STEP")
	}
	fields := append([]LIRMachineField{{ID: 0, Name: "depth", Type: "u32", Offset: 0, Size: 4, Alignment: 4}, {ID: 1, Name: "completed", Type: "bool", Offset: 4, Size: 1, Alignment: 1}}, l.SharedFields...)
	for i, field := range fields {
		actual := m.InitFields[i]
		if actual.ID != i || actual.Name != field.Name || actual.Type != field.Type || actual.Offset != field.Offset || actual.Size != field.Size || actual.Alignment != field.Alignment {
			return fmt.Errorf("LIR_BAD_ACTIVATION_STEP_HEADER")
		}
	}
	dom, pred, err := activationDominators(f)
	if err != nil {
		return err
	}
	views := make([]activationBlockView, len(f.Blocks))
	for i, b := range f.Blocks {
		views[i] = activationView(b)
	}
	entry := f.Blocks[0]
	done := f.Blocks[1]
	entryDef := views[0].defs[entry.Term.Value]
	if entry.Term.Op != "branch" || entry.Term.True != 1 || entryDef.Op != "load" || len(entryDef.Args) != 1 || !views[0].headerAddress(entryDef.Args[0], 1) || len(done.Instructions) != 1 || done.Instructions[0].Op != "machine_result" || done.Instructions[0].Literal != "Completed" || done.Term.Op != "return" || done.Term.Value != done.Instructions[0].Result {
		return fmt.Errorf("LIR_BAD_ACTIVATION_COMPLETED_PATH")
	}
	tagBlocks := make([]int, len(l.Machines))
	owner := make([]int, len(f.Blocks))
	for i := range owner {
		owner[i] = -1
	}
	guardID := entry.Term.False
	if guardID < 0 || guardID >= len(f.Blocks) {
		return fmt.Errorf("LIR_MISSING_ACTIVATION_LIVE_DEPTH_GUARD")
	}
	guard := f.Blocks[guardID]
	cmp := views[guardID].defs[guard.Term.Value]
	if guard.Term.Op != "branch" || cmp.Op != "gt" || len(cmp.Args) != 2 || !views[guardID].depth(cmp.Args[0]) || !views[guardID].constant(cmp.Args[1], "0") || guard.Term.False < 0 || guard.Term.False >= len(f.Blocks) || f.Blocks[guard.Term.False].Term.Op != "trap" || f.Blocks[guard.Term.False].Term.Reason != "invalid_machine_depth" {
		return fmt.Errorf("LIR_MISSING_ACTIVATION_LIVE_DEPTH_GUARD")
	}
	tagBlock := guard.Term.True
	for i, machine := range l.Machines {
		if tagBlock < 0 || tagBlock >= len(f.Blocks) {
			return fmt.Errorf("LIR_MISSING_ACTIVATION_TAG_ARM")
		}
		tagBlocks[i] = tagBlock
		block := f.Blocks[tagBlock]
		if block.Label != "tag."+machine.Name || !views[tagBlock].comparison(block, strconv.Itoa(int(machine.Tag)), l.SlotTagOffset, l) {
			return fmt.Errorf("LIR_BAD_ACTIVATION_TAG_DISPATCH")
		}
		stateBlock := block.Term.True
		for j, state := range machine.States {
			if stateBlock < 0 || stateBlock >= len(f.Blocks) || state.Block < 0 || state.Block >= len(f.Blocks) {
				return fmt.Errorf("LIR_BAD_ACTIVATION_STATE_DISPATCH")
			}
			sb := f.Blocks[stateBlock]
			if sb.Label != "dispatch."+machine.Name+"."+state.Name || !views[stateBlock].comparison(sb, strconv.Itoa(j), l.SlotDataOffset, l) || sb.Term.True != state.Block || f.Blocks[state.Block].Label != "state."+machine.Name+"."+state.Name {
				return fmt.Errorf("LIR_BAD_ACTIVATION_STATE_DISPATCH")
			}
			stateBlock = sb.Term.False
		}
		if stateBlock < 0 || stateBlock >= len(f.Blocks) || f.Blocks[stateBlock].Term.Op != "trap" || f.Blocks[stateBlock].Term.Reason != "invalid_machine_state" {
			return fmt.Errorf("LIR_MISSING_ACTIVATION_INVALID_STATE")
		}
		// State entry, not merely the tag comparison, must dominate body access.
		for j := range f.Blocks {
			if dom[j][block.Term.True] {
				owner[j] = i
			}
		}
		tagBlock = block.Term.False
	}
	if tagBlock < 0 || tagBlock >= len(f.Blocks) || f.Blocks[tagBlock].Term.Op != "trap" || f.Blocks[tagBlock].Term.Reason != "invalid_machine_tag" {
		return fmt.Errorf("LIR_MISSING_ACTIVATION_INVALID_TAG")
	}
	pushes := map[int]LIRActivationTransfer{}
	publishes := map[int]bool{}
	destroys := map[int]bool{}
	for _, t := range m.Transfers {
		if t.Block < 0 || t.Block >= len(f.Blocks) || owner[t.Block] < 0 || l.Machines[owner[t.Block]].Tag != t.ParentTag {
			return fmt.Errorf("LIR_BAD_ACTIVATION_TRANSFER")
		}
		if t.Kind == "push" {
			if _, exists := pushes[t.Block]; exists {
				return fmt.Errorf("LIR_DUPLICATE_ACTIVATION_PUSH")
			}
			pushes[t.Block] = t
			continue
		}
		if t.Kind != "pop" && t.Kind != "complete" {
			return fmt.Errorf("LIR_BAD_ACTIVATION_TRANSFER_KIND %s", t.Kind)
		}
		if len(t.DestroyBlocks) != len(l.Machines) || len(t.PublishBlocks) != len(l.Machines) {
			return fmt.Errorf("LIR_MISSING_ACTIVATION_DESTROY_ARM")
		}
		start := f.Blocks[t.Block].Term
		if start.Op != "jump" {
			return fmt.Errorf("LIR_BAD_ACTIVATION_POP_CFG")
		}
		for _, source := range t.SourceBlocks {
			if source < 0 || source >= len(f.Blocks) || owner[source] < 0 || f.Blocks[source].Term.Op != "jump" || f.Blocks[source].Term.True != start.True {
				return fmt.Errorf("LIR_BAD_ACTIVATION_POP_SOURCE")
			}
		}
		arm := start.True
		for i, machine := range l.Machines {
			d, p := t.DestroyBlocks[i], t.PublishBlocks[i]
			if arm < 0 || arm >= len(f.Blocks) || d < 0 || d >= len(f.Blocks) || p < 0 || p >= len(f.Blocks) || destroys[d] || publishes[p] {
				return fmt.Errorf("LIR_BAD_ACTIVATION_DESTROY_REGION")
			}
			destroys[d], publishes[p] = true, true
			dispatch := f.Blocks[arm]
			destroy := f.Blocks[d]
			publish := f.Blocks[p]
			if !views[arm].comparison(dispatch, strconv.Itoa(int(machine.Tag)), l.SlotTagOffset, l) || dispatch.Term.True != d || destroy.Label != "destroy."+machine.Identity || len(destroy.Instructions) != 0 || destroy.Term.Op != "jump" || destroy.Term.True != p || len(pred[d]) != 1 || pred[d][0] != arm || len(pred[p]) != 1 || pred[p][0] != d || !dom[p][d] {
				return fmt.Errorf("LIR_BAD_ACTIVATION_TYPED_DESTROY")
			}
			stores := 0
			for _, in := range publish.Instructions {
				if in.Op == "index_address" {
					return fmt.Errorf("LIR_ACTIVATION_USE_AFTER_DESTROY")
				}
				if in.Op == "store" {
					if len(in.Args) != 2 || !views[p].headerAddress(in.Args[0], 0) || !views[p].top(in.Args[1]) {
						return fmt.Errorf("LIR_BAD_ACTIVATION_POP_PUBLICATION")
					}
					stores++
				}
			}
			if stores != 1 {
				return fmt.Errorf("LIR_MISSING_ACTIVATION_POP_PUBLICATION")
			}
			cmp := views[p].defs[publish.Term.Value]
			if publish.Term.Op != "branch" || cmp.Op != "eq" || len(cmp.Args) != 2 || !views[p].top(cmp.Args[0]) || !views[p].constant(cmp.Args[1], "0") {
				return fmt.Errorf("LIR_BAD_ACTIVATION_ROOT_COMPLETION")
			}
			for edge, target := range []int{publish.Term.True, publish.Term.False} {
				if target < 0 || target >= len(f.Blocks) || len(pred[target]) != 1 || pred[target][0] != p {
					return fmt.Errorf("LIR_BAD_ACTIVATION_COMPLETION_CFG")
				}
				end := f.Blocks[target]
				ev := views[target]
				result := ev.defs[end.Term.Value]
				expected := "Active"
				if edge == 0 {
					expected = "Completed"
				}
				if end.Term.Op != "return" || result.Op != "machine_result" || result.Literal != expected {
					return fmt.Errorf("LIR_BAD_ACTIVATION_COMPLETION_BOUNDARY")
				}
				completionStores := 0
				for _, in := range end.Instructions {
					if in.Op == "index_address" {
						return fmt.Errorf("LIR_ACTIVATION_USE_AFTER_DESTROY")
					}
					if in.Op == "store" {
						if edge != 0 || len(in.Args) != 2 || !ev.headerAddress(in.Args[0], 1) || !ev.constant(in.Args[1], "true") {
							return fmt.Errorf("LIR_BAD_ACTIVATION_COMPLETION_STORE")
						}
						completionStores++
					}
				}
				if edge == 0 && completionStores != 1 {
					return fmt.Errorf("LIR_MISSING_ACTIVATION_COMPLETION_STORE")
				}
			}
			arm = dispatch.Term.False
		}
		if arm < 0 || arm >= len(f.Blocks) || f.Blocks[arm].Term.Op != "trap" || f.Blocks[arm].Term.Reason != "invalid_machine_tag" {
			return fmt.Errorf("LIR_MISSING_ACTIVATION_DESTROY_ARM")
		}
	}
	for i, block := range f.Blocks {
		v := views[i]
		for _, in := range block.Instructions {
			if in.Op == "index_address" {
				if !dom[i][guard.Term.True] {
					return fmt.Errorf("LIR_MISSING_ACTIVATION_LIVE_DEPTH_GUARD")
				}
				if len(in.Args) != 2 || in.Args[0] != 0 || in.Extent != l.Capacity || in.Stride != l.SlotSize {
					return fmt.Errorf("LIR_BAD_ACTIVATION_SLOT_ADDRESS")
				}
				offset := in.FrameOffset - l.SlotsOffset
				if v.depth(in.Args[1]) {
					t, ok := pushes[i]
					if !ok {
						return fmt.Errorf("LIR_UNREGISTERED_ACTIVATION_INIT")
					}
					child := activationMachineByTag(l, t.ChildTag)
					if child == nil || !activationFieldAt(*child, offset-l.SlotDataOffset, in.Type) && !(offset == l.SlotTagOffset && in.Type == "ptr<u32>") {
						return fmt.Errorf("LIR_BAD_ACTIVATION_CHILD_FIELD")
					}
				} else if v.top(in.Args[1]) {
					if offset == l.SlotTagOffset && in.Type == "ptr<u32>" {
						continue
					}
					if owner[i] < 0 || !activationFieldAt(l.Machines[owner[i]], offset-l.SlotDataOffset, in.Type) {
						return fmt.Errorf("LIR_BAD_ACTIVATION_CURRENT_FIELD")
					}
				} else {
					return fmt.Errorf("LIR_BAD_ACTIVATION_SLOT_INDEX")
				}
			}
			if in.Op == "store" && len(in.Args) == 2 && v.headerAddress(in.Args[0], 0) {
				if _, ok := pushes[i]; !ok && !publishes[i] {
					return fmt.Errorf("LIR_ACTIVATION_DEPTH_OUTSIDE_TRANSFER")
				}
			}
		}
	}
	for block, t := range pushes {
		if err := verifyActivationPushBlock(f.Blocks[block], views[block], t, l); err != nil {
			return err
		}
	}
	return nil
}
func activationMachineByTag(l ActivationStackLayout, tag uint32) *ActivationMachineLayout {
	for i := range l.Machines {
		if l.Machines[i].Tag == tag {
			return &l.Machines[i]
		}
	}
	return nil
}
func activationFieldAt(m ActivationMachineLayout, offset int, typ LIRType) bool {
	for _, f := range m.Fields {
		if f.Offset == offset && typ == LIRType("ptr<"+string(f.Type)+">") {
			return true
		}
	}
	return false
}
func verifyActivationPushBlock(block LIRBlock, v activationBlockView, t LIRActivationTransfer, l ActivationStackLayout) error {
	child := activationMachineByTag(l, t.ChildTag)
	parent := activationMachineByTag(l, t.ParentTag)
	if child == nil || parent == nil || t.ResumeState < 0 || t.ResumeState >= len(parent.States) {
		return fmt.Errorf("LIR_BAD_ACTIVATION_PUSH_TYPES")
	}
	initialized := map[int]bool{}
	capacityGuard := -1
	tagAt, depthAt, resumeAt := -1, -1, -1
	lastStore := -1
	for pos, in := range block.Instructions {
		if in.Op == "check_index" && len(in.Args) == 1 && v.depth(in.Args[0]) && in.Extent == l.Capacity {
			capacityGuard = pos
		}
		if in.Op != "store" || len(in.Args) != 2 {
			continue
		}
		lastStore = pos
		addr := v.defs[in.Args[0]]
		if addr.Op == "index_address" && len(addr.Args) == 2 {
			offset := addr.FrameOffset - l.SlotsOffset
			if v.depth(addr.Args[1]) {
				if capacityGuard < 0 || capacityGuard >= pos {
					return fmt.Errorf("LIR_MISSING_ACTIVATION_CAPACITY_GUARD")
				}
				if offset == l.SlotTagOffset {
					if !v.constant(in.Args[1], strconv.Itoa(int(child.Tag))) || len(initialized) != len(child.Fields) || resumeAt < 0 {
						return fmt.Errorf("LIR_ACTIVATION_TAG_BEFORE_INIT")
					}
					tagAt = pos
				} else {
					for _, field := range child.Fields {
						if offset == l.SlotDataOffset+field.Offset {
							if tagAt >= 0 || depthAt >= 0 {
								return fmt.Errorf("LIR_ACTIVATION_INIT_AFTER_PUBLICATION")
							}
							if field.ID == 0 && !v.constant(in.Args[1], "0") {
								return fmt.Errorf("LIR_BAD_ACTIVATION_CHILD_STATE")
							}
							initialized[field.ID] = true
						}
					}
				}
			} else if v.top(addr.Args[1]) && offset == l.SlotDataOffset {
				if capacityGuard < 0 || capacityGuard >= pos {
					return fmt.Errorf("LIR_ACTIVATION_CONTINUATION_BEFORE_CAPACITY")
				}
				if !v.constant(in.Args[1], strconv.Itoa(t.ResumeState)) {
					return fmt.Errorf("LIR_BAD_ACTIVATION_CONTINUATION")
				}
				resumeAt = pos
			}
		}
		if v.headerAddress(in.Args[0], 0) {
			next := v.defs[in.Args[1]]
			if tagAt < 0 || resumeAt < 0 || next.Op != "add" || len(next.Args) != 2 || !v.depth(next.Args[0]) || !v.constant(next.Args[1], "1") {
				return fmt.Errorf("LIR_ACTIVATION_DEPTH_BEFORE_INIT")
			}
			depthAt = pos
		}
	}
	if len(initialized) != len(child.Fields) || resumeAt < 0 || tagAt < 0 || depthAt != lastStore || depthAt < 0 {
		return fmt.Errorf("LIR_MISSING_ACTIVATION_PUSH_PUBLICATION")
	}
	result := v.defs[block.Term.Value]
	if block.Term.Op != "return" || result.Op != "machine_result" || result.Literal != "Active" {
		return fmt.Errorf("LIR_BAD_ACTIVATION_PUSH_BOUNDARY")
	}
	return nil
}
