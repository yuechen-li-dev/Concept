package concept

import (
	"fmt"
	"strconv"
	"strings"
)

// LIR is target independent. Computations define one value; mutable source
// variables use explicit slots until a later promotion pass exists.
type LIRModule struct {
	PlanID        string
	SemanticFacts []MIRSemanticFact
	Functions     []LIRFunction
}
type LIRType string
type LIRValue struct {
	ID   int
	Type LIRType
}
type LIRSlot struct {
	ID         int
	Name       string
	Type       LIRType
	Element    LIRType
	Extent     int
	ParamValue int // -1 unless bound to an aggregate parameter
	Facts      []string
}
type LIRInstruction struct {
	Op          string
	Result      int // -1 for no result
	Type        LIRType
	Args        []int
	Slot        int // -1 unless this is a slot or array operation
	Literal     string
	Extent      int
	Stride      int
	FrameField  int // -1 unless this is a caller-owned machine-frame address
	FrameOffset int
	Source      Span
	Decision    string
	Facts       []string
}
type LIRTerminator struct {
	Op     string
	Value  int
	True   int
	False  int
	Source Span
	Reason string // explicit trap reason
}
type LIRBlock struct {
	ID           int
	Label        string // stable machine-state provenance for inspection
	Instructions []LIRInstruction
	Term         LIRTerminator
}
type LIRFunction struct {
	Identity  string
	Name      string
	Params    []LIRValue
	Result    LIRType
	Slots     []LIRSlot
	Blocks    []LIRBlock
	Source    Span
	Facts     []string
	Decisions []string
	Machine   *LIRMachineFunction
}

// LIRMachineFunction describes a closed caller-owned frame shared by its
// generated Init and Step functions. It is metadata for verified ordinary LIR
// address, load, store, branch, and return operations, not another IR layer.
type LIRMachineFunction struct {
	Identity  string
	Role      string // init or step
	Size      int
	Alignment int
	Fields    []LIRMachineField
	States    []LIRMachineState
}
type LIRMachineField struct {
	ID        int
	Name      string
	Type      LIRType
	Offset    int
	Size      int
	Alignment int
}
type LIRMachineState struct {
	ID     int
	Name   string
	Block  int
	Source Span
}

func (m LIRModule) String() string {
	var out strings.Builder
	if m.PlanID != "" {
		fmt.Fprintf(&out, "plan %s\n", m.PlanID)
	}
	for _, fact := range m.SemanticFacts {
		fmt.Fprintf(&out, "fact %s %s %s\n", fact.ID, fact.Kind, fact.Outcome)
	}
	for i, f := range m.Functions {
		if i != 0 || m.PlanID != "" || len(m.SemanticFacts) > 0 {
			out.WriteByte('\n')
		}
		fmt.Fprintf(&out, "fn %s [%s] (", f.Name, f.Identity)
		for j, p := range f.Params {
			if j != 0 {
				out.WriteString(", ")
			}
			fmt.Fprintf(&out, "v%d: %s", p.ID, p.Type)
		}
		fmt.Fprintf(&out, ") -> %s\n", f.Result)
		if f.Machine != nil {
			fmt.Fprintf(&out, "  machine %s %s frame size=%d align=%d\n", f.Machine.Role, f.Machine.Identity, f.Machine.Size, f.Machine.Alignment)
			for _, field := range f.Machine.Fields {
				fmt.Fprintf(&out, "  field %d %s: %s offset=%d\n", field.ID, field.Name, field.Type, field.Offset)
			}
			for _, state := range f.Machine.States {
				fmt.Fprintf(&out, "  state %d %s -> b%d\n", state.ID, state.Name, state.Block)
			}
		}
		for _, fact := range f.Facts {
			fmt.Fprintf(&out, "  fact %s\n", fact)
		}
		for _, decision := range f.Decisions {
			fmt.Fprintf(&out, "  decision %s\n", decision)
		}
		for _, s := range f.Slots {
			if s.Extent > 0 {
				fmt.Fprintf(&out, "  s%d %s: [%d]%s", s.ID, s.Name, s.Extent, s.Element)
				if s.ParamValue >= 0 {
					fmt.Fprintf(&out, " = param v%d", s.ParamValue)
				}
				out.WriteByte('\n')
			} else {
				fmt.Fprintf(&out, "  s%d %s: %s\n", s.ID, s.Name, s.Type)
			}
		}
		for _, b := range f.Blocks {
			fmt.Fprintf(&out, "b%d:", b.ID)
			if b.Label != "" {
				fmt.Fprintf(&out, " ; %s", b.Label)
			}
			out.WriteByte('\n')
			for _, in := range b.Instructions {
				out.WriteString("  ")
				if in.Result >= 0 {
					fmt.Fprintf(&out, "v%d = ", in.Result)
				}
				fmt.Fprintf(&out, "%s %s", in.Op, in.Type)
				if in.Slot >= 0 {
					fmt.Fprintf(&out, " s%d", in.Slot)
				}
				for _, arg := range in.Args {
					fmt.Fprintf(&out, " v%d", arg)
				}
				if in.Literal != "" {
					fmt.Fprintf(&out, " %s", in.Literal)
				}
				if in.Extent > 0 {
					fmt.Fprintf(&out, " extent=%d", in.Extent)
				}
				if in.Stride > 0 {
					fmt.Fprintf(&out, " stride=%d", in.Stride)
				}
				if in.Op == "frame_field_address" {
					fmt.Fprintf(&out, " field=%d offset=%d", in.FrameField, in.FrameOffset)
				}
				if in.Decision != "" {
					fmt.Fprintf(&out, " [%s]", in.Decision)
				}
				out.WriteByte('\n')
			}
			switch b.Term.Op {
			case "return":
				if f.Result == "void" {
					out.WriteString("  ret\n")
				} else {
					fmt.Fprintf(&out, "  ret v%d\n", b.Term.Value)
				}
			case "jump":
				fmt.Fprintf(&out, "  jump b%d\n", b.Term.True)
			case "branch":
				fmt.Fprintf(&out, "  branch v%d b%d b%d\n", b.Term.Value, b.Term.True, b.Term.False)
			case "trap":
				fmt.Fprintf(&out, "  trap %s\n", b.Term.Reason)
			default:
				fmt.Fprintf(&out, "  <invalid terminator %q>\n", b.Term.Op)
			}
		}
	}
	return out.String()
}

// VerifyLIR rejects malformed lowering before any later MachineIR consumer.
// Non-parameter values are block-local; cross-block state goes through slots.
func VerifyLIR(m LIRModule) error {
	facts := map[string]bool{}
	for _, fact := range m.SemanticFacts {
		if fact.ID == "" || facts[fact.ID] {
			return fmt.Errorf("LIR_BAD_FACT_ID %s", fact.ID)
		}
		facts[fact.ID] = true
	}
	for _, f := range m.Functions {
		if err := verifyLIRMachineFunction(f); err != nil {
			return err
		}
		for _, id := range f.Facts {
			if !facts[id] {
				return fmt.Errorf("LIR_UNKNOWN_FACT %s", id)
			}
		}
		if f.Identity == "" || len(f.Blocks) == 0 {
			return fmt.Errorf("LIR_INVALID_FUNCTION %s", f.Name)
		}
		all := map[int]bool{}
		params := map[int]LIRType{}
		for _, p := range f.Params {
			if all[p.ID] {
				return fmt.Errorf("LIR_DUPLICATE_VALUE v%d", p.ID)
			}
			all[p.ID] = true
			params[p.ID] = p.Type
		}
		slots := map[int]LIRSlot{}
		for _, s := range f.Slots {
			for _, id := range s.Facts {
				if !facts[id] {
					return fmt.Errorf("LIR_UNKNOWN_FACT %s", id)
				}
			}
			if _, ok := slots[s.ID]; ok {
				return fmt.Errorf("LIR_DUPLICATE_SLOT s%d", s.ID)
			}
			slots[s.ID] = s
			if s.ParamValue >= 0 && params[s.ParamValue] != s.Type {
				return fmt.Errorf("LIR_BAD_PARAMETER_SLOT s%d", s.ID)
			}
			if s.Extent < 0 || s.Extent > 0 && s.Element == "" {
				return fmt.Errorf("LIR_BAD_ARRAY s%d", s.ID)
			}
		}
		for bi, b := range f.Blocks {
			if b.ID != bi {
				return fmt.Errorf("LIR_BLOCK_ORDER b%d", b.ID)
			}
			known := map[int]LIRType{}
			guards := map[[2]int]bool{}
			for id, t := range params {
				known[id] = t
			}
			for _, in := range b.Instructions {
				for _, id := range in.Facts {
					if !facts[id] {
						return fmt.Errorf("LIR_UNKNOWN_FACT %s", id)
					}
				}
				for _, a := range in.Args {
					if _, ok := known[a]; !ok {
						return fmt.Errorf("LIR_UNDEFINED_VALUE v%d in b%d", a, b.ID)
					}
				}
				if in.Result >= 0 {
					if all[in.Result] {
						return fmt.Errorf("LIR_DUPLICATE_VALUE v%d", in.Result)
					}
					if in.Type == "" {
						return fmt.Errorf("LIR_MISSING_TYPE v%d", in.Result)
					}
					all[in.Result] = true
					known[in.Result] = in.Type
				}
				argType := func(i int) LIRType {
					if i >= len(in.Args) {
						return ""
					}
					return known[in.Args[i]]
				}
				s, hasSlot := slots[in.Slot]
				switch in.Op {
				case "const":
					if in.Result < 0 || len(in.Args) != 0 || in.Literal == "" {
						return fmt.Errorf("LIR_BAD_CONST b%d", b.ID)
					}
				case "machine_result":
					if f.Machine == nil || f.Machine.Role != "step" || in.Result < 0 || in.Type != "machine_step_result" || len(in.Args) != 0 || (in.Literal != "Active" && in.Literal != "Yielded" && in.Literal != "Completed") {
						return fmt.Errorf("LIR_BAD_MACHINE_RESULT b%d", b.ID)
					}
				case "frame_field_address":
					if f.Machine == nil || in.FrameField < 0 || in.FrameField >= len(f.Machine.Fields) || in.Result < 0 || len(in.Args) != 1 || in.Args[0] != f.Params[0].ID {
						return fmt.Errorf("LIR_BAD_FRAME_ADDRESS b%d", b.ID)
					}
					field := f.Machine.Fields[in.FrameField]
					if in.FrameOffset != field.Offset || in.Type != LIRType("ptr<"+string(field.Type)+">") {
						return fmt.Errorf("LIR_BAD_FRAME_OFFSET b%d", b.ID)
					}
				case "load_slot":
					if !hasSlot || s.Extent > 0 || in.Type != s.Type || len(in.Args) != 0 || in.Result < 0 {
						return fmt.Errorf("LIR_BAD_SLOT_LOAD b%d", b.ID)
					}
				case "store_slot":
					if !hasSlot || s.Extent > 0 || in.Result >= 0 || len(in.Args) != 1 || argType(0) != s.Type {
						return fmt.Errorf("LIR_BAD_SLOT_STORE b%d", b.ID)
					}
				case "checked_add", "checked_sub", "checked_mul", "add", "sub", "mul":
					if in.Result < 0 || len(in.Args) != 2 || argType(0) != in.Type || argType(1) != in.Type || !lirInteger(in.Type) {
						return fmt.Errorf("LIR_BAD_ARITHMETIC b%d", b.ID)
					}
				case "eq", "ne", "lt", "le", "gt", "ge":
					if in.Result < 0 || len(in.Args) != 2 || argType(0) != argType(1) || (!lirInteger(argType(0)) && argType(0) != "bool") || in.Type != "bool" {
						return fmt.Errorf("LIR_BAD_COMPARE b%d", b.ID)
					}
				case "check_index":
					if in.Result >= 0 || len(in.Args) != 1 || !lirInteger(argType(0)) || in.Extent <= 0 || in.Type != "void" {
						return fmt.Errorf("LIR_BAD_INDEX_CHECK b%d", b.ID)
					}
					guards[[2]int{in.Args[0], in.Extent}] = true
				case "index_address":
					if !hasSlot || s.Extent <= 0 || s.Extent != in.Extent || in.Stride != lirWidth(s.Element) || len(in.Args) != 1 || !lirInteger(argType(0)) || in.Type != LIRType("ptr<"+string(s.Element)+">") || in.Result < 0 {
						return fmt.Errorf("LIR_BAD_INDEX_ADDRESS b%d", b.ID)
					}
					if !guards[[2]int{in.Args[0], in.Extent}] {
						return fmt.Errorf("LIR_MISSING_INDEX_GUARD b%d", b.ID)
					}
				case "load":
					if in.Result < 0 || len(in.Args) != 1 || argType(0) != LIRType("ptr<"+string(in.Type)+">") {
						return fmt.Errorf("LIR_BAD_LOAD b%d", b.ID)
					}
				case "store":
					if in.Result >= 0 || len(in.Args) != 2 || argType(0) != LIRType("ptr<"+string(in.Type)+">") || argType(1) != in.Type {
						return fmt.Errorf("LIR_BAD_STORE b%d", b.ID)
					}
				default:
					return fmt.Errorf("LIR_UNSUPPORTED_OP %s", in.Op)
				}
			}
			switch b.Term.Op {
			case "return":
				if f.Result != "void" && known[b.Term.Value] != f.Result {
					return fmt.Errorf("LIR_BAD_RETURN b%d", b.ID)
				}
			case "jump":
				if b.Term.True < 0 || b.Term.True >= len(f.Blocks) {
					return fmt.Errorf("LIR_BAD_TARGET b%d", b.ID)
				}
			case "branch":
				if known[b.Term.Value] != "bool" || b.Term.True < 0 || b.Term.True >= len(f.Blocks) || b.Term.False < 0 || b.Term.False >= len(f.Blocks) {
					return fmt.Errorf("LIR_BAD_BRANCH b%d", b.ID)
				}
			case "trap":
				if f.Machine == nil || b.Term.Reason != "invalid_machine_state" {
					return fmt.Errorf("LIR_BAD_TRAP b%d", b.ID)
				}
			default:
				return fmt.Errorf("LIR_MISSING_TERMINATOR b%d", b.ID)
			}
		}
		if err := verifyLIRMachineEffects(f); err != nil {
			return err
		}
	}
	return nil
}

func verifyLIRMachineFunction(f LIRFunction) error {
	m := f.Machine
	if m == nil {
		return nil
	}
	if m.Identity == "" || (m.Role != "init" && m.Role != "step") || len(f.Params) == 0 || f.Params[0].Type != LIRType("ptr<frame:"+m.Identity+">") || m.Size <= 0 || m.Alignment <= 0 || len(m.Fields) < 2 {
		return fmt.Errorf("LIR_BAD_MACHINE_FRAME %s", f.Name)
	}
	if m.Role == "init" && f.Result != "void" || m.Role == "step" && f.Result != "machine_step_result" {
		return fmt.Errorf("LIR_BAD_MACHINE_RESULT_TYPE %s", f.Name)
	}
	if m.Role == "init" && (len(f.Blocks) != 1 || f.Blocks[0].Term.Op != "return") {
		return fmt.Errorf("LIR_BAD_MACHINE_INIT_CFG %s", f.Name)
	}
	end := 0
	for i, field := range m.Fields {
		if field.ID != i || field.Name == "" || field.Size <= 0 || field.Alignment <= 0 || field.Offset < end || field.Offset%field.Alignment != 0 || field.Offset+field.Size > m.Size || (field.Type != "bool" && lirWidth(field.Type) != field.Size) {
			return fmt.Errorf("LIR_BAD_MACHINE_FIELD %s field=%d", f.Name, i)
		}
		end = field.Offset + field.Size
	}
	if m.Fields[0].Name != "current_state" || m.Fields[0].Type != "u32" || m.Fields[0].Offset != 0 || m.Fields[1].Name != "completed" || m.Fields[1].Type != "bool" {
		return fmt.Errorf("LIR_BAD_MACHINE_STATUS %s", f.Name)
	}
	if m.Size%m.Alignment != 0 {
		return fmt.Errorf("LIR_BAD_MACHINE_FRAME %s", f.Name)
	}
	if len(m.States) == 0 {
		return fmt.Errorf("LIR_MISSING_MACHINE_STATES %s", f.Name)
	}
	for i, state := range m.States {
		if state.ID != i || state.Name == "" || (m.Role == "step" && (state.Block < 0 || state.Block >= len(f.Blocks) || f.Blocks[state.Block].Label != "state."+state.Name)) || (m.Role == "init" && state.Block != -1) {
			return fmt.Errorf("LIR_BAD_MACHINE_STATE %s state=%d", f.Name, i)
		}
	}
	return nil
}

// Check frame mutations that establish the Init/Step contract. Ordinary LIR
// type checks alone cannot distinguish a suspended frame from a returned tag.
func verifyLIRMachineEffects(f LIRFunction) error {
	m := f.Machine
	if m == nil {
		return nil
	}
	initialized := make([]bool, len(m.Fields))
	initState, initCompleted := "", ""
	stateTargets := map[int]bool{}
	invalidTrap := false
	for _, b := range f.Blocks {
		addresses := map[int]int{}
		constants := map[int]string{}
		stored := map[int]string{}
		result := map[int]string{}
		for _, in := range b.Instructions {
			switch in.Op {
			case "frame_field_address":
				addresses[in.Result] = in.FrameField
			case "const":
				constants[in.Result] = in.Literal
			case "machine_result":
				result[in.Result] = in.Literal
			case "store":
				if len(in.Args) == 2 {
					if field, ok := addresses[in.Args[0]]; ok {
						initialized[field] = true
						stored[field] = constants[in.Args[1]]
						if field == 0 && stored[field] != "" {
							id, err := strconv.Atoi(stored[field])
							if err != nil || id < 0 || id >= len(m.States) {
								return fmt.Errorf("LIR_BAD_MACHINE_STATE_STORE b%d", b.ID)
							}
						}
					}
				}
			}
		}
		if m.Role == "step" {
			if b.Label == "already-completed" && (b.ID != 1 || len(b.Instructions) != 1 || b.Instructions[0].Op != "machine_result" || b.Instructions[0].Literal != "Completed" || b.Term.Op != "return") {
				return fmt.Errorf("LIR_BAD_MACHINE_COMPLETED_PATH b%d", b.ID)
			}
			if b.Term.Op == "branch" && strings.HasPrefix(b.Label, "dispatch.") {
				stateTargets[b.Term.True] = true
			}
			if b.Term.Op == "trap" && b.Term.Reason == "invalid_machine_state" {
				invalidTrap = true
			}
			if b.Term.Op == "return" {
				tag := result[b.Term.Value]
				if tag == "" {
					return fmt.Errorf("LIR_BAD_MACHINE_RETURN b%d", b.ID)
				}
				if (tag == "Yielded" || tag == "Active") && stored[0] == "" {
					return fmt.Errorf("LIR_MISSING_MACHINE_RESUME_STATE b%d", b.ID)
				}
				if tag == "Completed" && b.Label != "already-completed" && stored[1] != "true" {
					return fmt.Errorf("LIR_MISSING_MACHINE_COMPLETION b%d", b.ID)
				}
			}
		}
		if m.Role == "init" {
			initState, initCompleted = stored[0], stored[1]
		}
	}
	if m.Role == "init" {
		for i, present := range initialized {
			if !present {
				return fmt.Errorf("LIR_MISSING_MACHINE_INIT field=%d", i)
			}
		}
		if initState != "0" || initCompleted != "false" {
			return fmt.Errorf("LIR_BAD_MACHINE_INITIAL_STATE %s", f.Name)
		}
	} else {
		for _, state := range m.States {
			if !stateTargets[state.Block] {
				return fmt.Errorf("LIR_MISSING_MACHINE_DISPATCH state=%d", state.ID)
			}
		}
		if !invalidTrap {
			return fmt.Errorf("LIR_MISSING_MACHINE_INVALID_STATE")
		}
	}
	return nil
}

func lirInteger(t LIRType) bool {
	switch t {
	case "i8", "u8", "i16", "u16", "i32", "u32", "i64", "u64":
		return true
	}
	return false
}
func lirWidth(t LIRType) int {
	switch t {
	case "i8", "u8":
		return 1
	case "i16", "u16":
		return 2
	case "i32", "u32", "machine_step_result":
		return 4
	case "i64", "u64":
		return 8
	}
	return 0
}
