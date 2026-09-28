package concept

import (
	"fmt"
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
	Op       string
	Result   int // -1 for no result
	Type     LIRType
	Args     []int
	Slot     int // -1 unless this is a slot or array operation
	Literal  string
	Extent   int
	Stride   int
	Source   Span
	Decision string
	Facts    []string
}
type LIRTerminator struct {
	Op     string
	Value  int
	True   int
	False  int
	Source Span
}
type LIRBlock struct {
	ID           int
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
			fmt.Fprintf(&out, "b%d:\n", b.ID)
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
			default:
				return fmt.Errorf("LIR_MISSING_TERMINATOR b%d", b.ID)
			}
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
	case "i32", "u32":
		return 4
	case "i64", "u64":
		return 8
	}
	return 0
}
