package concept

import "fmt"

// Bootstrap producer consistency. The independent Concept validator owns
// artifact admission on the Cathedral path. Neither computes field packing.
func verifyObjectMachine(m ObjectModule) error {
	return verifyObjectContract(m, nil)
}
func verifyObjectContract(m ObjectModule, authority []BorrowFunction) error {
	return verifyObjectContractEnums(m, authority, nil)
}
func verifyObjectContractEnums(m ObjectModule, authority []BorrowFunction, enums []LIREnum) error {
	bad := func(s string) error { return fmt.Errorf("MIR_CMIRAMD5_%s", s) }
	if len(m.Types) == 0 || len(m.Types) > 256 || len(m.Functions) == 0 || len(m.Functions) > 128 || len(m.Dependencies) > 128 {
		return bad("CAPACITY")
	}
	for i, d := range m.Dependencies {
		if d.Identity == "" || !objectDigest(d.Digest) || (i > 0 && m.Dependencies[i-1].Identity >= d.Identity) {
			return bad("DEPENDENCY_IDENTITY")
		}
	}
	types := make([]LIRType, len(m.Types))
	layouts := []LIRAggregateLayout{}
	identities := map[string]bool{}
	for i, t := range m.Types {
		if t.ID != i || t.Identity == "" || t.LayoutIdentity == "" || identities[t.Identity] || t.Evidence != "checked_layout" {
			return bad("TYPE_IDENTITY")
		}
		identities[t.Identity] = true
		if _, e := machineBridgeTag(objectBridgeTagsObjectKind, t.Kind); e != nil {
			return bad("TYPE_KIND")
		}
		if t.Kind == "aggregate" {
			if t.Copy != "checked_value" || t.Element != -1 || t.Extent != 0 || t.Stride != 0 {
				return bad("COPY_ELIGIBILITY")
			}
			l := LIRAggregateLayout{Type: LIRType(t.LayoutIdentity), Identity: t.Identity, Name: t.Name, Kind: "struct", Representation: t.Representation, Size: t.Size, Alignment: t.Align, CopyEligibility: t.Copy}
			if t.Immutable {
				l.Kind = "record"
			}
			for j, f := range t.Fields {
				if f.ID != j || f.Type < 0 || f.Type >= i || f.Identity == "" || !objectDigest(f.SemanticType) {
					return bad("FIELD_IDENTITY")
				}
				for k := 0; k < j; k++ {
					if t.Fields[k].Identity == f.Identity {
						return bad("DUPLICATE_FIELD_IDENTITY")
					}
				}
				field := LIRAggregateField{Name: f.Name, Type: types[f.Type], TypeIdentity: f.SemanticType, DeclarationIdentity: f.Identity, Offset: f.Offset, Size: f.Size, Alignment: f.Align, Readonly: f.Readonly}
				child := m.Types[f.Type]
				if f.LayoutIdentity != child.LayoutIdentity {
					return bad("NESTED_LAYOUT_IDENTITY")
				}
				if child.Kind == "array" {
					field.ElementType = types[child.Element]
					field.Extent = child.Extent
					field.Stride = child.Stride
					field.ElementAlignment = m.Types[child.Element].Align
				}
				l.Fields = append(l.Fields, field)
			}
			types[i] = l.Type
			layouts = append(layouts, l)
		} else if t.Kind == "array" {
			if t.Element < 0 || t.Element >= i || t.Extent <= 0 || t.Extent > 256 || t.Stride <= 0 || t.Size <= 0 || t.Size > 1048576 || t.Copy != "checked_value" || t.Representation != "fixed_array" || len(t.Fields) != 0 || t.Immutable {
				return bad("ARRAY_GEOMETRY")
			}
			child := m.Types[t.Element]
			if child.Kind == "void" || t.Stride != child.Size || t.Align != child.Align || t.Size/t.Extent != t.Stride || t.Size%t.Extent != 0 {
				return bad("ARRAY_GEOMETRY")
			}
			types[i] = LIRType(fmt.Sprintf("[%d]%s", t.Extent, types[t.Element]))
			if t.Identity != string(types[i]) || t.LayoutIdentity != string(types[i]) {
				return bad("ARRAY_IDENTITY")
			}
		} else {
			var domain *LIREnum
			for j := range enums {
				if string(enums[j].Type) == t.LayoutIdentity {
					domain = &enums[j]
					break
				}
			}
			width := scalarWidth(t.Kind)
			if t.Kind == "void" {
				width = 0
			}
			copy := "checked_value"
			if t.Kind == "void" {
				copy = "none"
			}
			if t.Size != width || t.Align != max(1, width) || (domain == nil && (t.Identity != t.Kind || t.LayoutIdentity != t.Kind || t.Name != t.Kind)) || t.Representation != "scalar" || t.Element != -1 || t.Extent != 0 || t.Stride != 0 || t.Immutable || len(t.Fields) != 0 || t.Copy != copy {
				return bad("SCALAR_TYPE")
			}
			types[i] = LIRType(t.Kind)
			if domain != nil {
				types[i] = domain.Type
			}
		}
	}
	if len(layouts) == 0 && authority == nil && len(enums) == 0 {
		return bad("MISSING_AGGREGATE_LAYOUT")
	}
	lir := LIRModule{Aggregates: layouts, Enums: enums}
	facts := map[string]bool{}
	typeAt := func(id int) (LIRType, error) {
		if id < 0 || id >= len(types) {
			return "", bad("VALUE_TYPE")
		}
		return types[id], nil
	}
	for fi, f := range m.Functions {
		if f.Target != "amd64-windows" || f.ABI != "win64" || len(f.Values) > 512 || len(f.Slots) > 64 || len(f.Blocks) > 128 || len(f.Blocks) == 0 || len(f.Params) > 32 || f.Source.Line <= 0 || f.Source.Column <= 0 {
			return bad("FUNCTION")
		}
		result, e := typeAt(f.ResultType)
		if e != nil {
			return e
		}
		lf := LIRFunction{Identity: f.Identity, Name: f.Name, Result: result, Source: Span{Line: f.Source.Line, Column: f.Source.Column}, Facts: f.Facts, Decisions: f.Decisions}
		if authority != nil {
			var err error
			lf.Borrow, err = borrowLIRContract(authority[fi], f, types)
			if err != nil {
				return err
			}
		}
		defs := make([]int, len(f.Values))
		valueType := func(id int) (LIRType, error) {
			if id < 0 || id >= len(f.Values) {
				return "", bad("VALUE_ID")
			}
			v := f.Values[id]
			t, e := typeAt(v.Type)
			if e != nil {
				return "", e
			}
			if v.Class == "location" {
				if lf.Borrow != nil {
					for _, ref := range lf.Borrow.References {
						if ref.Value == id && ref.Evidence != "checked_place" {
							return lirReference(t, ref.Mutable), nil
						}
					}
				}
				return lirPointer(t), nil
			}
			return t, nil
		}
		for i, v := range f.Values {
			if v.ID != i || v.Type < 0 || v.Type >= len(types) {
				return bad("VALUE_ID")
			}
			kind := m.Types[v.Type].Kind
			if v.LayoutIdentity != m.Types[v.Type].LayoutIdentity {
				return bad("VALUE_LAYOUT_IDENTITY")
			}
			if kind == "void" || v.Class != "location" && v.Class != "scalar" && v.Class != "aggregate" || (v.Class == "aggregate") != (kind == "aggregate" || kind == "array") && v.Class != "location" {
				return bad("VALUE_CLASS")
			}
		}
		for i, p := range f.Params {
			if p.Index != i || p.Value < 0 || p.Value >= len(f.Values) || p.Type != f.Values[p.Value].Type || f.Values[p.Value].Class == "location" && authority == nil {
				return bad("PARAMETER")
			}
			t, e := valueType(p.Value)
			if e != nil {
				return e
			}
			defs[p.Value]++
			lf.Params = append(lf.Params, LIRValue{ID: p.Value, Type: t})
		}
		for i, s := range f.Slots {
			if s.ID != i || s.Space != "SystemMemory" || s.ParamValue != -1 {
				return bad("STORAGE")
			}
			t, e := typeAt(s.Type)
			if e != nil {
				return e
			}
			if t == "void" {
				return bad("STORAGE")
			}
			if s.LayoutIdentity != m.Types[s.Type].LayoutIdentity {
				return bad("STORAGE_LAYOUT_IDENTITY")
			}
			lf.Slots = append(lf.Slots, LIRSlot{ID: s.ID, Name: s.Name, Type: t, Readonly: s.Readonly, ParamValue: -1})
		}
		count := 0
		for bi, b := range f.Blocks {
			if b.ID != bi {
				return bad("BLOCK")
			}
			lb := LIRBlock{ID: b.ID, Term: LIRTerminator{Op: b.Term.Op, Value: b.Term.Value, True: b.Term.True, False: b.Term.False, Source: Span{Line: b.Term.Source.Line, Column: b.Term.Source.Column}}}
			for ii, in := range b.Instructions {
				count++
				if count > 1024 || in.LIRBlock != bi || in.LIRInstruction < 0 || in.Source.Line <= 0 || in.Source.Column <= 0 {
					return bad("OPERATION_PROVENANCE")
				}
				if _, e := machineBridgeTag(objectBridgeTagsObjectOpcode, in.Op); e != nil {
					return bad("OPCODE")
				}
				t, e := typeAt(in.Type)
				if e != nil {
					return e
				}
				wantCopy := "none"
				kind := m.Types[in.Type].Kind
				if (kind == "aggregate" || kind == "array") && !(authority != nil && in.Op == "copy_value" && in.Result >= 0 && in.Result < len(f.Values) && f.Values[in.Result].Class == "location") && (in.Op == "copy_value" || in.Op == "replace_member" || in.Op == "initialize_slot" || in.Op == "load_slot" || in.Op == "store_slot" || in.Op == "load" || in.Op == "store") {
					wantCopy = "checked_value"
				}
				if in.Copy != wantCopy {
					return bad("COPY_ELIGIBILITY")
				}
				if in.Result >= 0 {
					if in.Result >= len(f.Values) || f.Values[in.Result].Type != in.Type {
						return bad("RESULT_TYPE")
					}
					defs[in.Result]++
					t, e = valueType(in.Result)
					if e != nil {
						return e
					}
				}
				li := LIRInstruction{Op: in.Op, Result: in.Result, Type: t, Slot: in.Slot, Member: in.Member, FrameOffset: in.Offset, Extent: in.Extent, Stride: in.Stride, Literal: in.Literal, Source: Span{Line: in.Source.Line, Column: in.Source.Column}, Decision: in.Decision, Facts: in.Facts}
				if authority != nil {
					li.Scope = borrowSiteScope(authority[fi], bi, ii)
					if in.Op == "copy_value" && in.Result >= 0 && f.Values[in.Result].Class == "location" {
						li.Op = "borrow"
					}
				}
				if in.ParentType != -1 {
					li.Aggregate, e = typeAt(in.ParentType)
					if e != nil {
						return e
					}
				}
				for _, a := range in.Args {
					if a.Value < 0 || a.Value >= len(f.Values) {
						return bad("OPERAND")
					}
					li.Args = append(li.Args, a.Value)
				}
				if in.Op == "call" {
					if len(in.Calls) != 1 {
						return bad("CALL_SIGNATURE")
					}
					c := in.Calls[0]
					if c.ResultType != in.Type || len(c.Arguments) != len(in.Args) || len(c.Arguments) > 32 {
						return bad("CALL_SIGNATURE")
					}
					li.CallTarget, li.CallABI = c.Target, c.Convention
					for _, target := range m.Functions {
						if target.Identity == c.Target {
							li.Callee = target.Name
						}
					}
					for i, a := range c.Arguments {
						at, e := typeAt(a.Type)
						if e != nil {
							return e
						}
						if authority != nil && a.Value >= 0 && a.Value < len(f.Values) && f.Values[a.Value].Class == "location" {
							at, e = valueType(a.Value)
							if e != nil {
								return e
							}
						}
						if a.Value != in.Args[i].Value {
							return bad("CALL_SIGNATURE")
						}
						li.ArgTypes = append(li.ArgTypes, at)
					}
				} else if len(in.Calls) != 0 {
					return bad("CALL_SIGNATURE")
				}
				for _, id := range in.Facts {
					facts[id] = true
				}
				lb.Instructions = append(lb.Instructions, li)
			}
			lf.Blocks = append(lf.Blocks, lb)
		}
		for _, n := range defs {
			if n != 1 {
				return bad("VALUE_DEFINITION")
			}
		}
		for _, id := range f.Facts {
			facts[id] = true
		}
		lir.Functions = append(lir.Functions, lf)
	}
	for id := range facts {
		lir.SemanticFacts = append(lir.SemanticFacts, MIRSemanticFact{ID: id})
	}
	if err := VerifyLIR(lir); err != nil {
		return fmt.Errorf("MIR_CMIRAMD5_%w", err)
	}
	return nil
}

func objectDigest(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
