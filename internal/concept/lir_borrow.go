package concept

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Only validateExpr constructs this witness after the ordinary place, liveness
// and authority checks. User concepts cannot manufacture it.
type checkedBorrowOrigin struct {
	Target     Type
	Root       string
	Provenance evt1LifetimeProvenance
	Mutable    bool
}
type LIRBorrowScope struct {
	ID, Parent int
	Source     Span
}
type LIRBorrowReference struct {
	Value                  int
	Target                 LIRType
	SemanticTarget         string
	RootKind               string
	Root, Parent, Lifetime int
	Mutable                bool
	Evidence, Provenance   string
	Source                 Span
}
type LIRBorrowContract struct {
	Scopes     []LIRBorrowScope
	SlotScopes []int
	References []LIRBorrowReference
}

func referenceSemanticType(t Type) string {
	data, _ := json.Marshal(evt1GenericIdentityType(t.valueType()))
	return digest(data)
}
func (b *lirBuilder) referenceTarget(value int, t Type) int {
	for i := range b.references {
		if b.references[i].Value == value {
			b.references[i].SemanticTarget = referenceSemanticType(evt1CanonicalType(b.env, t))
		}
	}
	return value
}
func lirReference(t LIRType, mutable bool) LIRType {
	if mutable {
		return LIRType("ref<" + string(t) + ">")
	}
	return LIRType("ref_const<" + string(t) + ">")
}
func lirReferenceType(t LIRType) bool {
	return strings.HasPrefix(string(t), "ref<") || strings.HasPrefix(string(t), "ref_const<")
}
func lirAddressTarget(t LIRType) LIRType {
	for _, prefix := range []string{"ptr<", "ref<", "ref_const<"} {
		if strings.HasPrefix(string(t), prefix) && strings.HasSuffix(string(t), ">") {
			return LIRType(string(t)[len(prefix) : len(t)-1])
		}
	}
	return ""
}
func (b *lirBuilder) referenceParameter(value, ordinal int, typ Type, target LIRType) {
	b.hasBorrow = true
	b.references = append(b.references, LIRBorrowReference{Value: value, Target: target, SemanticTarget: referenceSemanticType(evt1CanonicalType(b.env, typ)), RootKind: "parameter", Root: ordinal, Parent: -1, Lifetime: 0, Mutable: !typ.Const, Evidence: "checked_parameter", Provenance: fmt.Sprintf("%s|parameter:%d", b.fn.Identity, ordinal), Source: b.fn.Source})
}
func (b *lirBuilder) findReference(value int) (LIRBorrowReference, bool) {
	for _, r := range b.references {
		if r.Value == value {
			return r, true
		}
	}
	return LIRBorrowReference{}, false
}
func (b *lirBuilder) noteLocation(in LIRInstruction) {
	// Legacy scalar-array slots and automata addresses belong to their frozen
	// input contracts, not to checked source-reference construction.
	if in.Op == "index_address" && in.Slot >= 0 {
		return
	}
	target := lirAddressTarget(in.Type)
	if target == "" || in.Result < 0 {
		return
	}
	r := LIRBorrowReference{Value: in.Result, Target: target, Parent: -1, Lifetime: b.scope, Source: in.Source, Evidence: "checked_place"}
	if in.Op == "slot_address" {
		r.RootKind = "local"
		r.Root = in.Slot
		r.Mutable = !b.fn.Slots[in.Slot].Readonly
		r.Provenance = fmt.Sprintf("%s|slot:%d", b.fn.Identity, in.Slot)
	} else if len(in.Args) > 0 {
		parent, ok := b.findReference(in.Args[0])
		if !ok {
			b.borrowErr = fmt.Errorf("LIR_REFERENCE_ORIGIN_MISSING v%d", in.Args[0])
			return
		}
		r.RootKind = parent.RootKind
		r.Root = parent.Root
		r.Parent = parent.Value
		r.Mutable = parent.Mutable
		r.Provenance = parent.Provenance
		if in.Op == "field_address" {
			for _, l := range b.aggregates.layouts {
				if l.Type == in.Aggregate {
					r.Mutable = r.Mutable && l.Kind != "record" && !l.Fields[in.Member].Readonly
				}
			}
		}
		if in.Op == "borrow" {
			r.Evidence = "checked_borrow"
			r.Mutable = strings.HasPrefix(string(in.Type), "ref<")
		}
	} else {
		return
	}
	b.references = append(b.references, r)
}
func (b *lirBuilder) borrowExpr(e *RefExpr) (int, LIRType, error) {
	if e.checkedBorrow == nil || (e.checkedBorrow.Provenance.Kind != evt1ProvenanceLocal && e.checkedBorrow.Provenance.Kind != evt1ProvenanceParameter) || e.checkedBorrow.Root == "" {
		return 0, "", fmt.Errorf("EVT2_REFERENCE_AUTHORITY_UNKNOWN")
	}
	base, target, err := b.objectAddress(e.Value)
	if err != nil {
		return 0, "", err
	}
	t, err := b.lowerType(target.valueType())
	if err != nil {
		return 0, "", err
	}
	checked, err := b.lowerType(e.checkedBorrow.Target)
	if err != nil || checked != t {
		return 0, "", fmt.Errorf("EVT2_REFERENCE_CHECKED_TARGET")
	}
	typ := lirReference(t, e.checkedBorrow.Mutable)
	b.hasBorrow = true
	id := b.emit(LIRInstruction{Op: "borrow", Result: 0, Type: typ, Args: []int{base}, Slot: -1, Source: e.Span})
	return b.referenceTarget(id, e.checkedBorrow.Target), typ, nil
}

// CFG dominance is consistency verification, not a source borrow checker.
func lirDominators(f LIRFunction) [][]bool {
	n := len(f.Blocks)
	d := make([][]bool, n)
	reachable := make([]bool, n)
	if n == 0 {
		return d
	}
	reachable[0] = true
	for changed := true; changed; {
		changed = false
		for i, b := range f.Blocks {
			if !reachable[i] {
				continue
			}
			for _, s := range lirSuccessors(b) {
				if s >= 0 && s < n && !reachable[s] {
					reachable[s] = true
					changed = true
				}
			}
		}
	}
	for i := range d {
		d[i] = make([]bool, n)
		for j := range d[i] {
			d[i][j] = reachable[i] && reachable[j]
		}
		if i == 0 {
			for j := range d[i] {
				d[i][j] = j == 0
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for i := 1; i < n; i++ {
			if !reachable[i] {
				continue
			}
			v := make([]bool, n)
			for j := range v {
				v[j] = true
			}
			for p, b := range f.Blocks {
				for _, s := range lirSuccessors(b) {
					if s == i && reachable[p] {
						for j := range v {
							v[j] = v[j] && d[p][j]
						}
					}
				}
			}
			v[i] = true
			for j := range v {
				if v[j] != d[i][j] {
					changed = true
					d[i][j] = v[j]
				}
			}
		}
	}
	return d
}
func lirSuccessors(b LIRBlock) []int {
	if b.Term.Op == "jump" {
		return []int{b.Term.True}
	}
	if b.Term.Op == "branch" {
		return []int{b.Term.True, b.Term.False}
	}
	return nil
}
func lirReferenceDefinitions(f LIRFunction) (map[int]LIRType, map[int]int) {
	types := map[int]LIRType{}
	blocks := map[int]int{}
	for _, p := range f.Params {
		types[p.ID] = p.Type
		blocks[p.ID] = -1
	}
	for _, b := range f.Blocks {
		for _, in := range b.Instructions {
			if in.Result >= 0 {
				types[in.Result] = in.Type
				blocks[in.Result] = b.ID
			}
		}
	}
	return types, blocks
}
func borrowScopeContains(c *LIRBorrowContract, parent, child int) bool {
	for child >= 0 && child < len(c.Scopes) {
		if parent == child {
			return true
		}
		child = c.Scopes[child].Parent
	}
	return false
}
func verifyLIRBorrowFunction(f LIRFunction, layouts map[LIRType]LIRAggregateLayout) error {
	c := f.Borrow
	if c == nil {
		return nil
	}
	bad := func(s string) error { return fmt.Errorf("LIR_REFERENCE_%s %s", s, f.Name) }
	if len(f.Blocks) == 0 || len(f.Blocks) > 128 {
		return bad("CAPACITY")
	}
	for i, b := range f.Blocks {
		if b.ID != i {
			return bad("CONTROL_FLOW")
		}
		for _, target := range lirSuccessors(b) {
			if target < 0 || target >= len(f.Blocks) {
				return bad("CONTROL_FLOW")
			}
		}
	}
	if len(c.Scopes) == 0 || len(c.Scopes) > 128 || len(c.References) > 512 || len(c.SlotScopes) != len(f.Slots) {
		return bad("CAPACITY")
	}
	for i, s := range c.Scopes {
		if s.ID != i || s.Parent >= i || s.Parent < -1 || s.Source.Line <= 0 || s.Source.Column <= 0 || (i == 0) != (s.Parent == -1) {
			return bad("LIFETIME")
		}
	}
	for _, s := range c.SlotScopes {
		if s < 0 || s >= len(c.Scopes) {
			return bad("LIFETIME")
		}
	}
	types, defs := lirReferenceDefinitions(f)
	refs := map[int]LIRBorrowReference{}
	ops := map[int]LIRInstruction{}
	for _, b := range f.Blocks {
		for _, in := range b.Instructions {
			if in.Result >= 0 {
				ops[in.Result] = in
			}
		}
	}
	previous := -1
	for _, r := range c.References {
		if r.Value <= previous {
			return bad("IDENTITY")
		}
		previous = r.Value
		if _, ok := refs[r.Value]; ok || lirAddressTarget(types[r.Value]) != r.Target || r.Provenance == "" || !objectDigest(r.SemanticTarget) || r.Source.Line <= 0 || r.Lifetime < 0 || r.Lifetime >= len(c.Scopes) {
			return bad("IDENTITY")
		}
		if r.RootKind == "parameter" {
			if r.Root < 0 || r.Root >= len(f.Params) || !lirReferenceType(f.Params[r.Root].Type) {
				return bad("PARAMETER")
			}
		} else if r.RootKind == "local" {
			if r.Root < 0 || r.Root >= len(f.Slots) || !borrowScopeContains(c, c.SlotScopes[r.Root], r.Lifetime) {
				return bad("LIFETIME")
			}
		} else {
			return bad("PROVENANCE")
		}
		if r.Parent < 0 {
			if r.Evidence == "checked_parameter" {
				if r.RootKind != "parameter" || f.Params[r.Root].ID != r.Value || r.Lifetime != 0 || r.Mutable != strings.HasPrefix(string(types[r.Value]), "ref<") {
					return bad("PARAMETER")
				}
			} else {
				in := ops[r.Value]
				if in.Op != "slot_address" || r.RootKind != "local" || in.Slot != r.Root || r.Mutable == f.Slots[r.Root].Readonly || r.Evidence != "checked_place" {
					return bad("PROVENANCE")
				}
			}
		} else {
			p, ok := refs[r.Parent]
			in := ops[r.Value]
			if !ok || len(in.Args) == 0 || in.Args[0] != r.Parent || p.RootKind != r.RootKind || p.Root != r.Root || p.Provenance != r.Provenance || !borrowScopeContains(c, p.Lifetime, r.Lifetime) || r.Mutable && !p.Mutable {
				return bad("AUTHORITY")
			}
			if in.Op == "borrow" {
				if r.Evidence != "checked_borrow" || r.Target != p.Target || r.SemanticTarget != p.SemanticTarget || r.Mutable != strings.HasPrefix(string(in.Type), "ref<") {
					return bad("AUTHORITY")
				}
			} else if r.Evidence != "checked_place" || (in.Op != "field_address" && in.Op != "index_address") {
				return bad("PROVENANCE")
			}
			if in.Op == "field_address" {
				l, ok := layouts[in.Aggregate]
				if !ok || in.Member < 0 || in.Member >= len(l.Fields) || r.Mutable != (p.Mutable && l.Kind != "record" && !l.Fields[in.Member].Readonly) {
					return bad("AUTHORITY")
				}
			}
			if in.Op == "index_address" && r.Mutable != p.Mutable {
				return bad("AUTHORITY")
			}

		}
		if in, ok := ops[r.Value]; ok {
			if in.Source != r.Source || in.Scope != r.Lifetime {
				return bad("PROVENANCE")
			}
		}
		refs[r.Value] = r
	}
	dom := lirDominators(f)
	for _, b := range f.Blocks {
		for _, in := range b.Instructions {
			if in.Scope < 0 || in.Scope >= len(c.Scopes) {
				return bad("LIFETIME")
			}
			if lirAddressTarget(in.Type) != "" && in.Result >= 0 {
				if _, ok := refs[in.Result]; !ok {
					return bad("MISSING_AUTHORITY")
				}
			}
			for ai, id := range in.Args {
				r, ok := refs[id]
				if !ok {
					continue
				}
				if !borrowScopeContains(c, r.Lifetime, in.Scope) {
					return bad("OUTSIDE_LIFETIME")
				}
				db := defs[id]
				if db >= 0 && db != b.ID && !dom[b.ID][db] {
					return bad("DOMINANCE")
				}
				if in.Op == "store" && ai == 0 && !r.Mutable {
					return bad("CONST_STORE")
				}
				if in.Op == "call" && !lirReferenceType(types[id]) || in.Op == "store_slot" || in.Op == "initialize_slot" || in.Op == "store" && ai == 1 {
					return bad("ESCAPE")
				}
			}
		}
		if b.Term.Op == "return" {
			if _, ok := refs[b.Term.Value]; ok && f.Result != "void" {
				return bad("RETURN_UNSUPPORTED")
			}
		}
	}
	return nil
}

func lirBorrowReference(f LIRFunction, value int) (LIRBorrowReference, bool) {
	if f.Borrow != nil {
		for _, r := range f.Borrow.References {
			if r.Value == value {
				return r, true
			}
		}
	}
	return LIRBorrowReference{}, false
}
