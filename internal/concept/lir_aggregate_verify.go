package concept

import (
	"fmt"
	"strings"
)

// Verify only checked object evidence. This is not a native lifetime engine:
// source references, raw storage, move and Drop have not been admitted here.
// Aggregate slots become live through complete typed value initialization.
func verifyLIRAggregateFunction(f LIRFunction, layouts map[LIRType]LIRAggregateLayout) error {
	objects := map[int]bool{}
	for _, slot := range f.Slots {
		if lirAggregateType(slot.Type) {
			if _, ok := layouts[slot.Type]; !ok || slot.Extent != 0 || slot.ParamValue != -1 {
				return fmt.Errorf("LIR_BAD_OBJECT_SLOT s%d", slot.ID)
			}
			objects[slot.ID] = true
		}
	}
	checkValueType := func(t LIRType) error {
		if lirAggregateType(t) {
			if _, ok := layouts[t]; !ok {
				return fmt.Errorf("LIR_UNKNOWN_OBJECT_TYPE %s", t)
			}
		}
		if strings.HasPrefix(string(t), "ptr<") && (strings.Contains(string(t), "aggregate:") || lirTaggedType(lirAddressTarget(t))) && f.Borrow == nil {
			return fmt.Errorf("LIR_OBJECT_REFERENCE_NOT_QUALIFIED %s", t)
		}
		return nil
	}
	if err := checkValueType(f.Result); err != nil {
		return err
	}
	for _, param := range f.Params {
		if err := checkValueType(param.Type); err != nil {
			return err
		}
	}
	for _, block := range f.Blocks {
		for _, in := range block.Instructions {
			if lirAggregateType(in.Type) {
				if err := checkValueType(in.Type); err != nil {
					return err
				}
			}
		}
	}
	if len(layouts) == 0 {
		return nil
	}
	// Definite initialization is a forward must analysis over reachable CFG
	// predecessors. Non-entry blocks start at top, including loop backedges.
	preds := make([][]int, len(f.Blocks))
	successors := func(b LIRBlock) []int {
		switch b.Term.Op {
		case "jump":
			return []int{b.Term.True}
		case "branch":
			return []int{b.Term.True, b.Term.False}
		}
		return nil
	}
	reachable := make([]bool, len(f.Blocks))
	queue := []int{0}
	reachable[0] = true
	for len(queue) != 0 {
		block := queue[0]
		queue = queue[1:]
		for _, target := range successors(f.Blocks[block]) {
			preds[target] = append(preds[target], block)
			if !reachable[target] {
				reachable[target] = true
				queue = append(queue, target)
			}
		}
	}
	copySet := func(src map[int]bool) map[int]bool {
		dst := map[int]bool{}
		for slot := range src {
			dst[slot] = true
		}
		return dst
	}
	entry, exit := make([]map[int]bool, len(f.Blocks)), make([]map[int]bool, len(f.Blocks))
	for i := range f.Blocks {
		entry[i], exit[i] = copySet(objects), copySet(objects)
	}
	entry[0] = map[int]bool{}
	changed := true
	for changed {
		changed = false
		for i, block := range f.Blocks {
			if !reachable[i] {
				continue
			}
			incoming := map[int]bool{}
			if i != 0 {
				incoming = copySet(objects)
				for _, pred := range preds[i] {
					for slot := range incoming {
						if !exit[pred][slot] {
							delete(incoming, slot)
						}
					}
				}
			}
			entry[i] = incoming
			outgoing := copySet(incoming)
			for _, in := range block.Instructions {
				if in.Op == "initialize_slot" && objects[in.Slot] {
					outgoing[in.Slot] = true
				}
			}
			if len(outgoing) != len(exit[i]) {
				changed = true
			} else {
				for slot := range outgoing {
					if !exit[i][slot] {
						changed = true
						break
					}
				}
			}
			exit[i] = outgoing
		}
	}
	for bi, block := range f.Blocks {
		live := copySet(entry[bi])
		// Unreachable instructions still need well-formed typed derivations.
		if !reachable[bi] {
			live = map[int]bool{}
		}
		locations := map[int]LIRType{}
		writable := map[int]bool{}
		if f.Borrow != nil {
			for _, r := range f.Borrow.References {
				locations[r.Value] = r.Target
				writable[r.Value] = r.Mutable
			}
		}
		for _, in := range block.Instructions {
			// Object locations are compiler-internal address derivations, not
			// source references. This slice cannot return, store or pass them.
			for ai, arg := range in.Args {
				if _, location := locations[arg]; location && (in.Op == "call" && f.Borrow == nil || in.Op == "initialize_slot" || in.Op == "store_slot" || in.Op == "store" && ai == 1) {
					return fmt.Errorf("LIR_OBJECT_ADDRESS_ESCAPE b%d", bi)
				}
			}
			switch in.Op {
			case "initialize_slot", "store_slot":
				if objects[in.Slot] {
					if in.Op == "store_slot" && !live[in.Slot] {
						return fmt.Errorf("LIR_OBJECT_NOT_LIVE %s b%d s%d", f.Name, bi, in.Slot)
					}
					if in.Op == "store_slot" && f.Slots[slotOrdinal(f.Slots, in.Slot)].Readonly {
						return fmt.Errorf("LIR_OBJECT_READONLY b%d", bi)
					}
					live[in.Slot] = true
				}
			case "load_slot", "slot_address":
				if objects[in.Slot] && !live[in.Slot] {
					return fmt.Errorf("LIR_OBJECT_NOT_LIVE %s b%d s%d", f.Name, bi, in.Slot)
				}
				if in.Op == "slot_address" {
					slot := f.Slots[slotOrdinal(f.Slots, in.Slot)]
					locations[in.Result] = slot.Type
					writable[in.Result] = !slot.Readonly
				}
			case "field_address":
				if locations[in.Args[0]] != in.Aggregate {
					return fmt.Errorf("LIR_BAD_OBJECT_PROVENANCE b%d", bi)
				}
				locations[in.Result] = layouts[in.Aggregate].Fields[in.Member].Type
				writable[in.Result] = writable[in.Args[0]] && layouts[in.Aggregate].Kind == "struct" && !layouts[in.Aggregate].Fields[in.Member].Readonly
			case "index_address":
				if in.Slot == -1 && len(in.Args) == 2 {
					if base, object := locations[in.Args[0]]; object {
						element, extent, ok := lirArrayValue(base)
						size, _, err := lirAggregateGeometry(element, layouts, 0)
						if !ok || err != nil || extent != in.Extent || size != in.Stride || in.FrameOffset != 0 || in.Type != lirPointer(element) {
							return fmt.Errorf("LIR_BAD_OBJECT_INDEX b%d", bi)
						}
						locations[in.Result] = element
						writable[in.Result] = writable[in.Args[0]]
					}
				}
			}
			if in.Op == "store" {
				if _, object := locations[in.Args[0]]; object && !writable[in.Args[0]] {
					return fmt.Errorf("LIR_OBJECT_READONLY b%d", bi)
				}
			}
			if strings.HasPrefix(string(in.Type), "ptr<") && strings.Contains(string(in.Type), "aggregate:") && in.Result >= 0 {
				if _, ok := locations[in.Result]; !ok {
					return fmt.Errorf("LIR_BAD_OBJECT_PROVENANCE b%d", bi)
				}
			}
		}
		if block.Term.Op == "return" {
			if _, location := locations[block.Term.Value]; location && f.Result != "void" {
				return fmt.Errorf("LIR_OBJECT_ADDRESS_ESCAPE b%d", bi)
			}
		}
	}
	return nil
}

func slotOrdinal(slots []LIRSlot, id int) int {
	for i, slot := range slots {
		if slot.ID == id {
			return i
		}
	}
	return -1 // Ordinary slot verification has already rejected unknown IDs.
}
