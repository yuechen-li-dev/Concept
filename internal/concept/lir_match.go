package concept

import "fmt"

// Source-flow associations only; operands, types, scopes and CFG edges stay in
// their existing canonical records. Cathedral independently checks these links.
type LIRPatternFlow struct {
	SubjectSlot, ResultSlot, Join, Scope int
	Arms                                 []LIRPatternArm
}
type LIRPatternArm struct {
	Dispatch, Body, Scope int
	Bindings              []int
}

func (b *lirBuilder) checkedExpressionType(expr Expr) (Type, error) {
	scope := newEVT1Scope(nil)
	scope.returnType = b.mir.ReturnType
	for name, binding := range b.names {
		scope.declare(name, evt1ValueBinding{t: binding.typ, mutable: true, state: evt1StorageInitialized})
	}
	return validateExpr(b.env, scope, expr, nil, false)
}
func (b *lirBuilder) resultSlot(typ Type, span Span) (int, LIRType, error) {
	t, err := b.lowerType(typ)
	if err != nil {
		return 0, "", err
	}
	if t == "void" || lirReferenceType(t) {
		return 0, "", fmt.Errorf("EVT2H3_UNSUPPORTED_RESULT %s", typ.String())
	}
	return b.slot(fmt.Sprintf("$result%d", len(b.fn.Slots)), typ), t, nil
}
func (b *lirBuilder) storeResult(slot int, typ LIRType, expr Expr, span Span) error {
	value, actual, err := b.expr(expr)
	if err != nil {
		return err
	}
	if actual != typ {
		return fmt.Errorf("EVT2H3_RESULT_TYPE")
	}
	op := "initialize_slot"
	if len(b.fn.Patterns) == 0 && !lirAggregateType(typ) {
		op = "store_slot"
	}
	b.emit(LIRInstruction{Op: op, Result: -1, Type: typ, Args: []int{value}, Slot: slot, Source: span})
	return nil
}
func (b *lirBuilder) ifExpression(e *IfExpr) (int, LIRType, error) {
	typ, err := b.checkedExpressionType(e)
	if err != nil {
		return 0, "", err
	}
	slot, t, err := b.resultSlot(typ, e.Span)
	if err != nil {
		return 0, "", err
	}
	cond, ct, err := b.expr(e.Condition)
	if err != nil {
		return 0, "", err
	}
	if ct != "bool" {
		return 0, "", fmt.Errorf("EVT2_BRANCH_REQUIRES_BOOL")
	}
	then, other, join := b.newBlock(), b.newBlock(), b.newBlock()
	b.terminate(LIRTerminator{Op: "branch", Value: cond, True: then, False: other, Source: e.Span})
	for _, arm := range []struct {
		block int
		value Expr
	}{{then, e.Then}, {other, e.Else}} {
		b.current = arm.block
		if err = b.storeResult(slot, t, arm.value, e.Span); err != nil {
			return 0, "", err
		}
		b.terminate(LIRTerminator{Op: "jump", True: join, Source: e.Span})
	}
	b.current = join
	return b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: slot, Source: e.Span}), t, nil
}
func (b *lirBuilder) matchExpression(e *MatchExpr) (int, LIRType, error) {
	typ, err := b.checkedExpressionType(e)
	if err != nil {
		return 0, "", err
	}
	slot, t, err := b.resultSlot(typ, e.Span)
	if err != nil {
		return 0, "", err
	}
	patterns := make([]Pattern, len(e.Arms))
	for i, a := range e.Arms {
		patterns[i] = a.Pattern
	}
	err = b.lowerMatch(e.Subject, patterns, slot, e.Span, func(i int) error { return b.storeResult(slot, t, e.Arms[i].Value, e.Arms[i].Span) })
	if err != nil {
		return 0, "", err
	}
	return b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: slot, Source: e.Span}), t, nil
}
func (b *lirBuilder) matchStatement(s *MatchStmt) error {
	patterns := make([]Pattern, len(s.Arms))
	for i, a := range s.Arms {
		patterns[i] = a.Pattern
	}
	return b.lowerMatch(s.Subject, patterns, -1, s.Span, func(i int) error { return b.block(s.Arms[i].Block) })
}
func (b *lirBuilder) lowerMatch(subject Expr, patterns []Pattern, resultSlot int, span Span, body func(int) error) error {
	if b.machine != nil {
		return fmt.Errorf("EVT2H3_UNSUPPORTED_MACHINE_MATCH")
	}
	st, err := b.checkedExpressionType(subject)
	if err != nil {
		return err
	}
	if st.isReference() || !evt1TypeCopyable(b.env, st) || evt1TypeHasDrop(b.env, st) {
		return fmt.Errorf("EVT2H3_UNSUPPORTED_MATCH_AUTHORITY %s", st.String())
	}
	value, t, err := b.expr(subject)
	if err != nil {
		return err
	}
	if !lirTaggedType(t) && !lirEnumType(t) && !lirInteger(t) {
		return fmt.Errorf("EVT2H3_UNSUPPORTED_SUBJECT %s", st.String())
	}
	b.hasBorrow = true // Preserve the existing lexical scope records for admission.
	subjectSlot := b.slot(fmt.Sprintf("$subject%d", len(b.fn.Slots)), st)
	b.fn.Slots[subjectSlot].Readonly = true
	b.emit(LIRInstruction{Op: "initialize_slot", Result: -1, Type: t, Args: []int{value}, Slot: subjectSlot, Source: span})
	flow := LIRPatternFlow{SubjectSlot: subjectSlot, ResultSlot: resultSlot, Join: -1, Scope: b.scope}
	for range patterns {
		flow.Arms = append(flow.Arms, LIRPatternArm{Dispatch: b.newBlock(), Body: b.newBlock()})
	}
	if resultSlot >= 0 {
		flow.Join = b.newBlock()
	}
	index := len(b.fn.Patterns)
	b.fn.Patterns = append(b.fn.Patterns, flow)
	b.terminate(LIRTerminator{Op: "jump", True: flow.Arms[0].Dispatch, Source: span})
	oldNames, oldScope := b.names, b.scope
	var open []int
	for i, p := range patterns {
		arm := &b.fn.Patterns[index].Arms[i]
		b.current = arm.Dispatch
		b.scope = oldScope
		b.names = oldNames
		variant := VariantDecl{}
		if lirEnumType(t) || lirTaggedType(t) {
			decl := b.env.enums[st.Name]
			if evt1IsFailureType(st) {
				decl, _ = evt1FailureEnumDecl(st)
			}
			variant, _ = evt1LookupVariant(decl, p.VariantName)
			snapshot := b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: subjectSlot, Source: span})
			op, result, rt := "test_variant", 0, LIRType("bool")
			if i == len(patterns)-1 {
				op, result, rt = "require_variant", -1, "void"
			}
			cond := b.emit(LIRInstruction{Op: op, Result: result, Type: rt, Args: []int{snapshot}, Slot: -1, Aggregate: t, Member: variant.Tag, Source: span})
			if result >= 0 {
				b.terminate(LIRTerminator{Op: "branch", Value: cond, True: arm.Body, False: b.fn.Patterns[index].Arms[i+1].Dispatch, Source: span})
			} else {
				b.terminate(LIRTerminator{Op: "jump", True: arm.Body, Source: span})
			}
		} else if p.Wildcard {
			b.terminate(LIRTerminator{Op: "jump", True: arm.Body, Source: span})
		} else {
			snapshot := b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: subjectSlot, Source: span})
			literal, lt, e := b.expr(p.Literal)
			if e != nil {
				return e
			}
			if lt != t {
				return fmt.Errorf("EVT2H3_LITERAL_TYPE")
			}
			cond := b.emit(LIRInstruction{Op: "eq", Result: 0, Type: "bool", Args: []int{snapshot, literal}, Slot: -1, Source: span})
			b.terminate(LIRTerminator{Op: "branch", Value: cond, True: arm.Body, False: b.fn.Patterns[index].Arms[i+1].Dispatch, Source: span})
		}
		armScope := len(b.scopes)
		b.scopes = append(b.scopes, LIRBorrowScope{ID: armScope, Parent: oldScope, Source: p.Span})
		arm.Scope = armScope
		b.current = arm.Body
		b.scope = armScope
		b.names = map[string]lirBinding{}
		for n, v := range oldNames {
			b.names[n] = v
		}
		if len(p.Bindings) > 0 {
			snapshot := b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: subjectSlot, Source: span})
			for fi, name := range p.Bindings {
				field := variant.Payload[fi]
				ft, e := b.lowerType(field.Type)
				if e != nil {
					return e
				}
				bound := b.emit(LIRInstruction{Op: "checked_payload", Result: 0, Type: ft, Args: []int{snapshot}, Slot: -1, Aggregate: t, Member: variant.Tag, Extent: fi, Source: p.Span})
				s := b.slot(name, field.Type)
				b.fn.Slots[s].Readonly = field.Type.Const
				b.emit(LIRInstruction{Op: "initialize_slot", Result: -1, Type: ft, Args: []int{bound}, Slot: s, Source: p.Span})
				b.names[name] = lirBinding{slot: s, typ: field.Type, local: true}
				b.fn.Patterns[index].Arms[i].Bindings = append(b.fn.Patterns[index].Arms[i].Bindings, s)
			}
		}
		if err = body(i); err != nil {
			return err
		}
		if b.fn.Blocks[b.current].Term.Op == "" {
			open = append(open, b.current)
		}
	}
	b.names, b.scope = oldNames, oldScope
	if len(open) > 0 {
		join := b.fn.Patterns[index].Join
		if join < 0 {
			join = b.newBlock()
			b.fn.Patterns[index].Join = join
		}
		for _, block := range open {
			b.current = block
			b.terminate(LIRTerminator{Op: "jump", True: join, Source: span})
		}
		b.current = join
	}
	return nil
}

func verifyLIRVariantOperation(in LIRInstruction, arg LIRType, tagged map[LIRType]LIRTaggedEnum, enums []LIREnum) error {
	count := 0
	if d, ok := tagged[arg]; ok {
		count = len(d.Variants)
	}
	for _, e := range enums {
		if e.Type == arg {
			count = len(e.Variants)
		}
	}
	if count == 0 || in.Aggregate != arg || len(in.Args) != 1 || in.Member < 0 || in.Member >= count || in.Slot != -1 || in.Extent != 0 || in.Stride != 0 || in.FrameOffset != 0 || in.Literal != "" {
		return fmt.Errorf("LIR_PATTERN_VARIANT")
	}
	if in.Op == "test_variant" && (in.Type != "bool" || in.Result < 0) || in.Op == "require_variant" && (in.Type != "void" || in.Result != -1) {
		return fmt.Errorf("LIR_PATTERN_VARIANT")
	}
	return nil
}
