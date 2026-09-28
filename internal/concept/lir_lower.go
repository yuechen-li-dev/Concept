package concept

import (
	"fmt"
	"sort"
	"strconv"
)

// GenerateLIR runs the existing semantic MIR -> facts -> General Planner
// contract, then lowers only the EVT2 core subset. It never invokes C emission.
func GenerateLIR(module Module) (LIRModule, error) {
	env, err := analyzeModule(module)
	if err != nil {
		return LIRModule{}, err
	}
	evt1MaterializeGenericInstances(&module, env)
	evt1MaterializeGenericProofSummaries(&module, env)
	if err = evt1NormalizeModuleStorageTypes(&module, env); err != nil {
		return LIRModule{}, err
	}
	mir := buildMIR(module, env)
	evt1ProjectPersistentFactSubjects(&mir)
	evt1QualifyMIRFacts(&mir)
	evt1QualifyHardwareFacts(&mir, env)
	if err = evt1ValidateMIR(mir); err != nil {
		return LIRModule{}, err
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	plan, err := PlanModule(&mir, &facts, GenericC11Target(), *env.profile, ConservativeCompilationPolicy())
	if err != nil {
		return LIRModule{}, err
	}
	return LowerMirToLir(&mir, plan, env)
}

// LowerMirToLir consumes validated MIR and the exact planner output. The
// semantic body is retained in-memory because the legacy MIR JSON is a summary.
func LowerMirToLir(mir *MIR, plan *LoweringPlan, env *semanticEnv) (LIRModule, error) {
	if mir == nil || plan == nil || len(plan.Functions) != len(mir.Functions) {
		return LIRModule{}, fmt.Errorf("EVT2_PLAN_MISMATCH")
	}
	facts := NewSemanticFactSet(mir.SemanticFacts)
	if err := ValidateLoweringPlan(mir, &facts, plan); err != nil {
		return LIRModule{}, err
	}
	out := LIRModule{PlanID: plan.PlanID, SemanticFacts: append([]MIRSemanticFact(nil), mir.SemanticFacts...)}
	sort.Slice(out.SemanticFacts, func(i, j int) bool { return out.SemanticFacts[i].ID < out.SemanticFacts[j].ID })
	for _, automata := range mir.Automata {
		generated, err := lowerAutomataToLIR(automata, env)
		if err != nil {
			return LIRModule{}, err
		}
		out.Functions = append(out.Functions, generated...)
	}
	var err error
	for i, fn := range mir.Functions {
		if fn.SemanticBody == nil {
			continue
		} // declarations have no code to lower
		if plan.Functions[i].Function != fn.Name {
			return LIRModule{}, fmt.Errorf("EVT2_PLAN_MISMATCH %s", fn.Name)
		}
		b := &lirBuilder{mir: fn, plan: plan.Functions[i], current: 0, names: map[string]lirBinding{}}
		b.fn = LIRFunction{Identity: fn.DeclarationIdentity, Name: fn.Name, Source: fn.SourceSpan}
		for _, fact := range mir.SemanticFacts {
			for _, subject := range fact.Subjects {
				if subject.Function == fn.Name || subject.Kind == "function" && subject.Name == fn.Name {
					b.fn.Facts = append(b.fn.Facts, fact.ID)
					break
				}
			}
		}
		sort.Strings(b.fn.Facts)
		if b.fn.Identity == "" {
			return LIRModule{}, fmt.Errorf("EVT2_MIR_IDENTITY_MISSING %s", fn.Name)
		}
		b.fn.Result, err = lirType(fn.ReturnType)
		if err != nil {
			return LIRModule{}, fmt.Errorf("%s: %w", fn.Name, err)
		}
		b.newBlock()
		for _, p := range fn.Params {
			t, e := lirType(p.Type)
			if e != nil {
				return LIRModule{}, fmt.Errorf("%s: %w", fn.Name, e)
			}
			if p.Type.Kind == TypeArray {
				s := b.slot(p.Name, p.Type)
				id := b.value()
				b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: t})
				b.fn.Slots[s].ParamValue = id
				b.names[p.Name] = lirBinding{slot: s, typ: p.Type, array: true}
			} else {
				id := b.value()
				b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: t})
				b.names[p.Name] = lirBinding{value: id, typ: p.Type}
			}
		}
		if err = b.block(*fn.SemanticBody); err != nil {
			return LIRModule{}, fmt.Errorf("%s: %w", fn.Name, err)
		}
		if b.fn.Blocks[b.current].Term.Op == "" {
			if b.fn.Result != "void" {
				return LIRModule{}, fmt.Errorf("EVT2_MISSING_RETURN %s", fn.Name)
			}
			b.terminate(LIRTerminator{Op: "return", Source: fn.SourceSpan})
		}
		for _, d := range b.plan.Decisions {
			b.fn.Decisions = append(b.fn.Decisions, d.ID+":"+d.Strategy)
		}
		out.Functions = append(out.Functions, b.fn)
	}
	if err := VerifyLIR(out); err != nil {
		return LIRModule{}, err
	}
	return out, nil
}

func lirType(t Type) (LIRType, error) {
	if t.Kind == TypeArray && t.ArrayElem != nil && t.ArrayLength > 0 {
		el, e := lirType(*t.ArrayElem)
		if e != nil {
			return "", e
		}
		return LIRType(fmt.Sprintf("[%d]%s", t.ArrayLength, el)), nil
	}
	switch t.Name {
	case "void":
		return "void", nil
	case "bool":
		return "bool", nil
	case "int", "int32":
		return "i32", nil
	case "uint", "uint32":
		return "u32", nil
	case "int8":
		return "i8", nil
	case "uint8":
		return "u8", nil
	case "int16":
		return "i16", nil
	case "uint16":
		return "u16", nil
	case "int64":
		return "i64", nil
	case "uint64":
		return "u64", nil
	}
	return "", fmt.Errorf("EVT2_UNSUPPORTED_TYPE %s", t.String())
}

type lirBinding struct {
	value int
	slot  int
	typ   Type
	array bool
	local bool
}
type lirBuilder struct {
	mir           MIRFunction
	plan          FunctionPlan
	fn            LIRFunction
	current       int
	nextValue     int
	names         map[string]lirBinding
	machine       *LIRMachineFunction
	frameParam    int
	machineFields map[string]int
	machineStates map[string]int
	activeState   int
}

func (b *lirBuilder) value() int { n := b.nextValue; b.nextValue++; return n }
func (b *lirBuilder) newBlock() int {
	n := len(b.fn.Blocks)
	b.fn.Blocks = append(b.fn.Blocks, LIRBlock{ID: n})
	return n
}
func (b *lirBuilder) terminate(t LIRTerminator) { b.fn.Blocks[b.current].Term = t }
func (b *lirBuilder) emit(in LIRInstruction) int {
	if in.Result >= 0 {
		in.Result = b.value()
	}
	b.fn.Blocks[b.current].Instructions = append(b.fn.Blocks[b.current].Instructions, in)
	return in.Result
}
func (b *lirBuilder) slot(name string, t Type) int {
	id := len(b.fn.Slots)
	lt, _ := lirType(t)
	s := LIRSlot{ID: id, Name: name, Type: lt, ParamValue: -1}
	if t.Kind == TypeArray && t.ArrayElem != nil {
		s.Element, _ = lirType(*t.ArrayElem)
		s.Extent = t.ArrayLength
	}
	b.fn.Slots = append(b.fn.Slots, s)
	return id
}
func (b *lirBuilder) constant(t LIRType, literal string, span Span) int {
	return b.emit(LIRInstruction{Op: "const", Result: 0, Type: t, Slot: -1, Literal: literal, Source: span})
}
func (b *lirBuilder) block(block Block) error {
	old := b.names
	b.names = make(map[string]lirBinding, len(old))
	for k, v := range old {
		b.names[k] = v
	}
	defer func() { b.names = old }()
	for _, stmt := range block.Statements {
		if b.fn.Blocks[b.current].Term.Op != "" {
			break
		}
		if err := b.statement(stmt); err != nil {
			return err
		}
	}
	return nil
}
func (b *lirBuilder) statement(stmt Statement) error {
	switch s := stmt.(type) {
	case *VarDecl:
		if s.Comptime {
			return fmt.Errorf("EVT2_UNSUPPORTED_COMPTIME_LOCAL at %d:%d", s.Span.Line, s.Span.Column)
		}
		if s.Type.Kind == TypeArray {
			return fmt.Errorf("EVT2_UNSUPPORTED_LOCAL_ARRAY at %d:%d", s.Span.Line, s.Span.Column)
		}
		slot := b.slot(s.Name, s.Type)
		b.names[s.Name] = lirBinding{slot: slot, typ: s.Type, local: true}
		if s.Value != nil {
			v, t, e := b.expr(s.Value)
			if e != nil {
				return e
			}
			lt, _ := lirType(s.Type)
			if t != lt {
				return fmt.Errorf("EVT2_TYPE_MISMATCH local %s", s.Name)
			}
			b.emit(LIRInstruction{Op: "store_slot", Result: -1, Type: lt, Args: []int{v}, Slot: slot, Source: s.Span})
		}
	case *AssignStmt:
		if field, ok := s.Target.(*FieldExpr); ok && b.machine != nil {
			if s.CompoundOp != "" {
				return fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_COMPOUND_ASSIGN at %d:%d", s.Span.Line, s.Span.Column)
			}
			id, err := b.machineFieldID(field)
			if err != nil {
				return err
			}
			v, typ, err := b.expr(s.Value)
			if err != nil {
				return err
			}
			if typ != b.machine.Fields[id].Type {
				return fmt.Errorf("EVT2_TYPE_MISMATCH machine field %s", field.Field)
			}
			addr := b.frameFieldAddress(id, field.Span)
			b.emit(LIRInstruction{Op: "store", Result: -1, Type: typ, Args: []int{addr, v}, Slot: -1, Source: s.Span})
			return nil
		}
		if s.CompoundOp != "" {
			if _, ok := s.Target.(*NameExpr); !ok {
				return fmt.Errorf("EVT2_UNSUPPORTED_COMPOUND_TARGET %T", s.Target)
			}
		}
		switch target := s.Target.(type) {
		case *NameExpr:
			binding, ok := b.names[target.Name]
			if !ok || !binding.local {
				return fmt.Errorf("EVT2_UNSUPPORTED_ASSIGN %s", target.Name)
			}
			v, t, e := b.expr(s.Value)
			if e != nil {
				return e
			}
			lt, _ := lirType(binding.typ)
			if lt != t {
				return fmt.Errorf("EVT2_TYPE_MISMATCH assign")
			}
			b.emit(LIRInstruction{Op: "store_slot", Result: -1, Type: t, Args: []int{v}, Slot: binding.slot, Source: s.Span})
		case *IndexExpr:
			addr, elem, e := b.indexAddress(target)
			if e != nil {
				return e
			}
			v, t, e := b.expr(s.Value)
			if e != nil {
				return e
			}
			if elem != t {
				return fmt.Errorf("EVT2_TYPE_MISMATCH index store")
			}
			b.emit(LIRInstruction{Op: "store", Result: -1, Type: t, Args: []int{addr, v}, Slot: -1, Source: s.Span})
		default:
			return fmt.Errorf("EVT2_UNSUPPORTED_ASSIGN_TARGET %T", s.Target)
		}
	case *ReturnStmt:
		if s.Value == nil {
			b.terminate(LIRTerminator{Op: "return", Source: s.Span})
			return nil
		}
		v, t, e := b.expr(s.Value)
		if e != nil {
			return e
		}
		if t != b.fn.Result {
			return fmt.Errorf("EVT2_RETURN_TYPE %s", t)
		}
		b.terminate(LIRTerminator{Op: "return", Value: v, Source: s.Span})
	case *IfStmt:
		cond, t, e := b.expr(s.Condition)
		if e != nil {
			return e
		}
		if t != "bool" {
			return fmt.Errorf("EVT2_BRANCH_REQUIRES_BOOL")
		}
		then := b.newBlock()
		other := b.newBlock()
		b.terminate(LIRTerminator{Op: "branch", Value: cond, True: then, False: other, Source: s.Span})
		b.current = then
		if e = b.block(s.Then); e != nil {
			return e
		}
		thenEnd := b.current
		b.current = other
		if s.Else != nil {
			if e = b.block(*s.Else); e != nil {
				return e
			}
		}
		elseEnd := b.current
		thenOpen := b.fn.Blocks[thenEnd].Term.Op == ""
		elseOpen := b.fn.Blocks[elseEnd].Term.Op == ""
		if thenOpen || elseOpen {
			join := b.newBlock()
			if thenOpen {
				b.current = thenEnd
				b.terminate(LIRTerminator{Op: "jump", True: join, Source: s.Span})
			}
			if elseOpen {
				b.current = elseEnd
				b.terminate(LIRTerminator{Op: "jump", True: join, Source: s.Span})
			}
			b.current = join
		} else {
			b.current = elseEnd
		}
	case *ForeachStmt:
		if b.machine != nil {
			return fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_FOREACH at %d:%d", s.Span.Line, s.Span.Column)
		}
		return b.foreach(s)
	case *Block:
		return b.block(*s)
	case *TransitionStmt:
		if b.machine == nil {
			return fmt.Errorf("EVT2_UNSUPPORTED_STATEMENT %T", stmt)
		}
		id, ok := b.machineStates[s.Target]
		if !ok {
			return fmt.Errorf("EVT2_MACHINE_UNKNOWN_STATE %s", s.Target)
		}
		b.storeMachineState(id, s.Span)
		b.returnMachineResult("Active", s.Span)
	case *YieldStmt:
		if b.machine == nil {
			return fmt.Errorf("EVT2_UNSUPPORTED_STATEMENT %T", stmt)
		}
		// EVT1 yield re-enters the same source state on the next Step.
		b.storeMachineState(b.activeState, s.Span)
		b.returnMachineResult("Yielded", s.Span)
	case *MachineCompleteStmt:
		if b.machine == nil {
			return fmt.Errorf("EVT2_UNSUPPORTED_STATEMENT %T", stmt)
		}
		if s.Operation == "pop" {
			return fmt.Errorf("EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP")
		}
		if s.Kind != "neutral" || s.Value != nil {
			return fmt.Errorf("EVT2_UNSUPPORTED_MACHINE_OUTCOME %s", s.Kind)
		}
		b.storeMachineCompleted(true, s.Span)
		b.returnMachineResult("Completed", s.Span)
	case *PushMachineStmt:
		return fmt.Errorf("EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP")
	default:
		return fmt.Errorf("EVT2_UNSUPPORTED_STATEMENT %T at %d:%d", stmt, stmt.statementSpan().Line, stmt.statementSpan().Column)
	}
	return nil
}

func (b *lirBuilder) foreach(s *ForeachStmt) error {
	rangeExpr, ok := s.Source.(*BinaryExpr)
	if !ok || rangeExpr.Op != ".." {
		return fmt.Errorf("EVT2_UNSUPPORTED_FOREACH_SOURCE")
	}
	start, t, e := b.expr(rangeExpr.Left)
	if e != nil {
		return e
	}
	end, et, e := b.expr(rangeExpr.Right)
	if e != nil {
		return e
	}
	if t != et || !lirInteger(t) {
		return fmt.Errorf("EVT2_RANGE_TYPE")
	}
	itemType := s.ItemType
	if itemType.Name == "" {
		itemType = s.ElementType
	}
	slot := b.slot(s.ItemName, itemType)
	endSlot := b.slot(s.ItemName+"$end", itemType)
	b.emit(LIRInstruction{Op: "store_slot", Result: -1, Type: t, Args: []int{start}, Slot: slot, Source: s.Span})
	b.emit(LIRInstruction{Op: "store_slot", Result: -1, Type: t, Args: []int{end}, Slot: endSlot, Source: s.Span})
	condBlock := b.newBlock()
	bodyBlock := b.newBlock()
	exitBlock := b.newBlock()
	b.terminate(LIRTerminator{Op: "jump", True: condBlock, Source: s.Span})
	b.current = condBlock
	cur := b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: slot, Source: s.Span})
	limit := b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: endSlot, Source: s.Span})
	cmp := b.emit(LIRInstruction{Op: "lt", Result: 0, Type: "bool", Args: []int{cur, limit}, Slot: -1, Source: s.Span})
	b.terminate(LIRTerminator{Op: "branch", Value: cmp, True: bodyBlock, False: exitBlock, Source: s.Span})
	b.current = bodyBlock
	old := b.names
	b.names = make(map[string]lirBinding, len(old)+1)
	for k, v := range old {
		b.names[k] = v
	}
	b.names[s.ItemName] = lirBinding{slot: slot, typ: itemType, local: true}
	e = b.block(s.Body)
	b.names = old
	if e != nil {
		return e
	}
	if b.fn.Blocks[b.current].Term.Op == "" {
		cur = b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: slot, Source: s.Span})
		one := b.constant(t, "1", s.Span)
		next := b.emit(LIRInstruction{Op: "checked_add", Result: 0, Type: t, Args: []int{cur, one}, Slot: -1, Source: s.Span})
		b.emit(LIRInstruction{Op: "store_slot", Result: -1, Type: t, Args: []int{next}, Slot: slot, Source: s.Span})
		b.terminate(LIRTerminator{Op: "jump", True: condBlock, Source: s.Span})
	}
	b.current = exitBlock
	return nil
}

func (b *lirBuilder) expr(expr Expr) (int, LIRType, error) {
	switch e := expr.(type) {
	case *NameExpr:
		v, ok := b.names[e.Name]
		if !ok {
			if b.machine != nil {
				if id, found := b.machineFields[e.Name]; found {
					v, typ := b.loadMachineField(id, e.Span)
					return v, typ, nil
				}
			}
			return 0, "", fmt.Errorf("EVT2_UNKNOWN_NAME %s", e.Name)
		}
		if v.array {
			return 0, "", fmt.Errorf("EVT2_ARRAY_VALUE %s", e.Name)
		}
		t, _ := lirType(v.typ)
		if v.local {
			return b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: t, Slot: v.slot, Source: e.Span}), t, nil
		}
		return v.value, t, nil
	case *IntLiteral:
		t, err := lirType(e.ResolvedType)
		if err != nil {
			return 0, "", err
		}
		lit := strconv.FormatUint(e.Magnitude, 10)
		if e.Negative {
			lit = "-" + lit
		}
		return b.constant(t, lit, e.Span), t, nil
	case *BoolLiteral:
		lit := "false"
		if e.Value {
			lit = "true"
		}
		return b.constant("bool", lit, e.Span), "bool", nil
	case *BinaryExpr:
		left, lt, err := b.expr(e.Left)
		if err != nil {
			return 0, "", err
		}
		right, rt, err := b.expr(e.Right)
		if err != nil {
			return 0, "", err
		}
		if lt != rt {
			return 0, "", fmt.Errorf("EVT2_BINARY_TYPE_MISMATCH")
		}
		op := map[string]string{"+": "checked_add", "-": "checked_sub", "*": "checked_mul", "==": "eq", "!=": "ne", "<": "lt", "<=": "le", ">": "gt", ">=": "ge"}[e.Op]
		if op == "" {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_BINARY %s", e.Op)
		}
		result := lt
		if op == "eq" || op == "ne" || op == "lt" || op == "le" || op == "gt" || op == "ge" {
			result = "bool"
		}
		return b.emit(LIRInstruction{Op: op, Result: 0, Type: result, Args: []int{left, right}, Slot: -1, Source: e.Span}), result, nil
	case *IndexExpr:
		addr, t, err := b.indexAddress(e)
		if err != nil {
			return 0, "", err
		}
		return b.emit(LIRInstruction{Op: "load", Result: 0, Type: t, Args: []int{addr}, Slot: -1, Source: e.Span}), t, nil
	case *FieldExpr:
		if b.machine == nil {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_EXPR %T", expr)
		}
		id, err := b.machineFieldID(e)
		if err != nil {
			return 0, "", err
		}
		v, typ := b.loadMachineField(id, e.Span)
		return v, typ, nil
	default:
		return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_EXPR %T at %d:%d", expr, expr.exprSpan().Line, expr.exprSpan().Column)
	}
}

func (b *lirBuilder) indexAddress(e *IndexExpr) (int, LIRType, error) {
	base, ok := e.Base.(*NameExpr)
	if !ok {
		return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_INDEX_BASE")
	}
	binding, ok := b.names[base.Name]
	if !ok || !binding.array {
		return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_INDEX_BASE %s", base.Name)
	}
	indices := evt1StorageIndices(e)
	if len(indices) != 1 {
		return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_INDEX_RANK")
	}
	idx, t, err := b.expr(indices[0])
	if err != nil {
		return 0, "", err
	}
	if !lirInteger(t) {
		return 0, "", fmt.Errorf("EVT2_INDEX_TYPE")
	}
	slot := b.fn.Slots[binding.slot]
	decision := ""
	var decisionFacts []string
	for _, d := range b.plan.Decisions {
		if d.Category == "BoundsPlan" && d.SourceSpan == e.Span {
			if d.Strategy != "PerAccessRuntime" {
				return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_BOUNDS_PLAN %s", d.Strategy)
			}
			decision = d.ID + ":" + d.Strategy
			decisionFacts = append(decisionFacts, d.Evidence.FactIDs...)
			sort.Strings(decisionFacts)
			break
		}
	}
	if decision == "" {
		return 0, "", fmt.Errorf("EVT2_BOUNDS_PLAN_MISSING at %d:%d", e.Span.Line, e.Span.Column)
	}
	b.emit(LIRInstruction{Op: "check_index", Result: -1, Type: "void", Args: []int{idx}, Slot: -1, Extent: slot.Extent, Source: e.Span, Decision: decision, Facts: decisionFacts})
	addrType := LIRType("ptr<" + string(slot.Element) + ">")
	addr := b.emit(LIRInstruction{Op: "index_address", Result: 0, Type: addrType, Args: []int{idx}, Slot: binding.slot, Extent: slot.Extent, Stride: lirWidth(slot.Element), Source: e.Span})
	return addr, slot.Element, nil
}
