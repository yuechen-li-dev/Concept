package concept

import (
	"fmt"
	"math"
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
	lir, err := LowerMirToLir(&mir, plan, env)
	if err == nil && len(lir.Aggregates) != 0 {
		lir.Dependencies = append([]SemanticModuleDependency(nil), module.ImportedModuleIdentities...)
		sort.Slice(lir.Dependencies, func(i, j int) bool { return lir.Dependencies[i].ModuleIdentity < lir.Dependencies[j].ModuleIdentity })
	}
	return lir, err
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
	aggregates := &lirAggregateTypes{env: env, known: map[string]LIRType{}, visiting: map[string]bool{}}
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
		b := &lirBuilder{mir: fn, plan: plan.Functions[i], env: env, aggregates: aggregates, current: 0, names: map[string]lirBinding{}}
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
		b.fn.Result, err = b.lowerType(fn.ReturnType)
		if err != nil {
			return LIRModule{}, fmt.Errorf("%s: %w", fn.Name, err)
		}
		b.newBlock()
		for ordinal, p := range fn.Params {
			t, e := b.lowerType(p.Type)
			if e != nil {
				return LIRModule{}, fmt.Errorf("%s: %w", fn.Name, e)
			}
			if p.Type.isReference() {
				id := b.value()
				b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: t})
				b.names[p.Name] = lirBinding{value: id, typ: p.Type}
				b.referenceParameter(id, ordinal, p.Type, lirAddressTarget(t))
			} else if p.Type.Kind == TypeArray {
				s := b.slot(p.Name, p.Type)
				id := b.value()
				b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: t})
				b.fn.Slots[s].ParamValue = id
				b.names[p.Name] = lirBinding{slot: s, typ: p.Type, array: true}
			} else if lirAggregateType(t) {
				id := b.value()
				b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: t})
				s := b.slot(p.Name, p.Type)
				b.fn.Slots[s].Readonly = p.Type.Const
				b.emit(LIRInstruction{Op: "initialize_slot", Result: -1, Type: t, Args: []int{id}, Slot: s, Source: fn.SourceSpan})
				b.names[p.Name] = lirBinding{slot: s, typ: p.Type, local: true}
			} else {
				id := b.value()
				b.fn.Params = append(b.fn.Params, LIRValue{ID: id, Type: t})
				b.names[p.Name] = lirBinding{value: id, typ: p.Type}
			}
		}
		if err = b.block(*fn.SemanticBody); err != nil {
			return LIRModule{}, fmt.Errorf("%s: %w", fn.Name, err)
		}
		if b.borrowErr != nil {
			return LIRModule{}, b.borrowErr
		}
		if b.hasBorrow {
			b.fn.Borrow = &LIRBorrowContract{Scopes: b.scopes, SlotScopes: b.slotScopes, References: b.references}
		} else {
			for bi := range b.fn.Blocks {
				for ii := range b.fn.Blocks[bi].Instructions {
					b.fn.Blocks[bi].Instructions[ii].Scope = 0
				}
				b.fn.Blocks[bi].Term.Scope = 0
			}
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
	out.Aggregates = aggregates.layouts
	out.Enums = aggregates.enums
	if err := VerifyLIR(out); err != nil {
		return LIRModule{}, err
	}
	return out, nil
}

func lirType(t Type) (LIRType, error) {
	if t.isBorrowLike() || t.isOwned() || t.PointerTo != nil {
		return "", fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_AUTHORITY %s", t.String())
	}
	if t.Kind == TypeArray && t.ArrayElem != nil && t.ArrayLength > 0 {
		el, e := lirType(*t.ArrayElem)
		if e != nil {
			return "", e
		}
		return LIRType(fmt.Sprintf("[%d]%s", t.ArrayLength, el)), nil
	}
	if info, ok := evt1FloatRepresentationInfo(t); ok {
		switch info.Representation {
		case FloatBinary32:
			return "float32", nil
		case FloatBinary64:
			return "float64", nil
		default:
			return "", fmt.Errorf("EVT2_UNSUPPORTED_FLOAT_REPRESENTATION %s", info.Representation)
		}
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
	case "uint8", "byte":
		return "u8", nil
	case "int16":
		return "i16", nil
	case "uint16":
		return "u16", nil
	case "int64", "isize":
		return "i64", nil
	case "uint64", "usize":
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
	scope         int
	scopes        []LIRBorrowScope
	slotScopes    []int
	references    []LIRBorrowReference
	hasBorrow     bool
	borrowErr     error
	mir           MIRFunction
	plan          FunctionPlan
	env           *semanticEnv
	aggregates    *lirAggregateTypes
	fn            LIRFunction
	current       int
	nextValue     int
	names         map[string]lirBinding
	machine       *LIRMachineFunction
	frameParam    int
	machineFields map[string]int
	machineStates map[string]int
	// terminalStates marks states whose entry completes the machine.
	terminalStates map[string]bool
	activeState    int
	activation     *activationStepContext
}

func (b *lirBuilder) value() int { n := b.nextValue; b.nextValue++; return n }
func (b *lirBuilder) newBlock() int {
	n := len(b.fn.Blocks)
	b.fn.Blocks = append(b.fn.Blocks, LIRBlock{ID: n})
	return n
}
func (b *lirBuilder) terminate(t LIRTerminator) {
	if t.Op == "return" && b.fn.Result == "void" {
		t.Value = -1
	}
	t.Scope = b.scope
	b.fn.Blocks[b.current].Term = t
}
func (b *lirBuilder) emit(in LIRInstruction) int {
	if in.Result >= 0 {
		in.Result = b.value()
	}
	in.Scope = b.scope
	if b.aggregates != nil && b.machine == nil {
		b.noteLocation(in)
	}
	b.fn.Blocks[b.current].Instructions = append(b.fn.Blocks[b.current].Instructions, in)
	return in.Result
}
func (b *lirBuilder) slot(name string, t Type) int {
	id := len(b.fn.Slots)
	lt, _ := b.lowerType(t)
	s := LIRSlot{ID: id, Name: name, Type: lt, ParamValue: -1}
	if t.Kind == TypeArray && t.ArrayElem != nil {
		s.Element, _ = b.lowerType(*t.ArrayElem)
		s.Extent = t.ArrayLength
	}
	b.fn.Slots = append(b.fn.Slots, s)
	b.slotScopes = append(b.slotScopes, b.scope)
	return id
}
func (b *lirBuilder) constant(t LIRType, literal string, span Span) int {
	return b.emit(LIRInstruction{Op: "const", Result: 0, Type: t, Slot: -1, Literal: literal, Source: span})
}
func (b *lirBuilder) block(block Block) error {
	if len(b.scopes) == 0 {
		b.scopes = append(b.scopes, LIRBorrowScope{ID: 0, Parent: -1, Source: b.fn.Source})
	}
	oldScope := b.scope
	b.scope = len(b.scopes)
	b.scopes = append(b.scopes, LIRBorrowScope{ID: b.scope, Parent: oldScope, Source: block.Span})
	defer func() { b.scope = oldScope }()
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
	case *ExprStmt:
		if _, _, err := b.expr(s.Value); err != nil {
			return err
		}
	case *VarDecl:
		if s.Comptime {
			return fmt.Errorf("EVT2_UNSUPPORTED_COMPTIME_LOCAL at %d:%d", s.Span.Line, s.Span.Column)
		}
		if s.Type.isReference() {
			v, t, err := b.expr(s.Value)
			if err != nil {
				return err
			}
			expected, err := b.lowerType(s.Type)
			if err != nil || t != expected {
				return fmt.Errorf("EVT2_REFERENCE_INITIALIZER %s", s.Name)
			}
			b.names[s.Name] = lirBinding{value: v, typ: s.Type}
			return nil
		}
		if s.Type.Kind == TypeArray {
			return fmt.Errorf("EVT2_UNSUPPORTED_LOCAL_ARRAY at %d:%d", s.Span.Line, s.Span.Column)
		}
		lt, err := b.lowerType(s.Type)
		if err != nil {
			return err
		}
		slot := b.slot(s.Name, s.Type)
		b.fn.Slots[slot].Readonly = s.Const || s.Type.Const
		b.names[s.Name] = lirBinding{slot: slot, typ: s.Type, local: true}
		if s.Value != nil {
			v, t, e := b.expr(s.Value)
			if e != nil {
				return e
			}
			if t != lt {
				return fmt.Errorf("EVT2_TYPE_MISMATCH local %s", s.Name)
			}
			op := "store_slot"
			if lirAggregateType(lt) {
				op = "initialize_slot"
			}
			b.emit(LIRInstruction{Op: op, Result: -1, Type: lt, Args: []int{v}, Slot: slot, Source: s.Span})
		}
	case *AssignStmt:
		if field, ok := s.Target.(*FieldExpr); ok && b.machine == nil {
			if s.CompoundOp != "" {
				return fmt.Errorf("EVT2_UNSUPPORTED_OBJECT_COMPOUND_ASSIGN")
			}
			addr, target, err := b.objectAddress(field)
			if err != nil {
				return err
			}
			v, actual, err := b.expr(s.Value)
			if err != nil {
				return err
			}
			expected, err := b.lowerType(target)
			if err != nil || actual != expected {
				return fmt.Errorf("EVT2_TYPE_MISMATCH object field %s", field.Field)
			}
			b.emit(LIRInstruction{Op: "store", Result: -1, Type: expected, Args: []int{addr, v}, Slot: -1, Source: s.Span})
			return nil
		}
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
			if ok && binding.typ.isReference() {
				v, t, err := b.expr(s.Value)
				if err != nil {
					return err
				}
				expected, err := b.lowerType(binding.typ.valueType())
				if err != nil || t != expected {
					return fmt.Errorf("EVT2_REFERENCE_REBIND_UNSUPPORTED")
				}
				b.emit(LIRInstruction{Op: "store", Result: -1, Type: t, Args: []int{binding.value, v}, Slot: -1, Source: s.Span})
				return nil
			}
			if !ok || !binding.local {
				return fmt.Errorf("EVT2_UNSUPPORTED_ASSIGN %s", target.Name)
			}
			v, t, e := b.expr(s.Value)
			if e != nil {
				return e
			}
			lt, _ := b.lowerType(binding.typ)
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
		if b.terminalStates[s.Target] {
			b.storeMachineCompleted(true, s.Span)
			b.returnMachineResult("Completed", s.Span)
			return nil
		}
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
		if b.activation != nil {
			return b.lowerActivationCompletion(s)
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
		if b.activation != nil {
			return b.lowerActivationPush(s)
		}
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
	case *ConstructExpr:
		decl, ok := b.env.enums[e.EnumName]
		if !ok {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_ENUM_CONSTRUCTOR %s", e.EnumName)
		}
		t, err := b.lowerType(Type{Name: decl.Name, Kind: TypeEnum, Span: e.Span})
		if err != nil {
			return 0, "", err
		}
		if !lirEnumType(t) || len(e.Args) != 0 {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_ENUM_CONSTRUCTOR")
		}
		for _, v := range decl.Variants {
			if v.Name == e.VariantName {
				return b.constant(t, strconv.Itoa(v.Tag), e.Span), t, nil
			}
		}
		return 0, "", fmt.Errorf("EVT2_UNKNOWN_ENUM_VARIANT")
	case *RefExpr:
		return b.borrowExpr(e)
	case *ParenExpr:
		return b.expr(e.Value)
	case *InterpretExpr:
		v, source, err := b.expr(e.Value)
		if err != nil {
			return 0, "", err
		}
		target, err := lirType(e.Target)
		if err != nil || source != target || !lirFloating(target) {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_INTERPRET %s", e.Target.String())
		}
		// Semantic validation has already proved the interpretation. Quantity
		// meaning changes, but its identical representation needs no LIR op.
		return v, target, nil
	case *FloatLiteral:
		t, err := lirType(e.ResolvedType)
		if err != nil {
			return 0, "", err
		}
		if !lirFloating(t) {
			return 0, "", fmt.Errorf("EVT2_FLOAT_LITERAL_TYPE %s", t)
		}
		return b.constant(t, lirFloatLiteral(t, e.Value), e.Span), t, nil
	case *UnaryExpr:
		v, t, err := b.expr(e.Value)
		if err != nil {
			return 0, "", err
		}
		if e.Op != "-" || !lirFloating(t) {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_UNARY %s", e.Op)
		}
		return b.emit(LIRInstruction{Op: "fneg", Result: 0, Type: t, Args: []int{v}, Slot: -1, Source: e.Span}), t, nil
	case *CallExpr:
		if e.Intrinsic != "" || e.Member || e.Receiver != nil || e.CallableInvoke || e.DynDispatch || e.resolvedSignature == "" {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_CALL %s at %d:%d", e.Callee, e.Span.Line, e.Span.Column)
		}
		var target *FunctionDecl
		for i := range b.env.functions[e.Callee] {
			candidate := &b.env.functions[e.Callee][i]
			if evt1FunctionParamSignature(*candidate) == e.resolvedSignature {
				target = candidate
				break
			}
		}
		if target == nil || target.Body == nil || target.ExternABI != "" || target.Async || len(target.Params) != len(e.Args) {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_CALL_TARGET %s at %d:%d", e.Callee, e.Span.Line, e.Span.Column)
		}
		resultType, err := b.lowerType(target.ReturnType)
		if err != nil {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_CALL_RESULT %s: %w", e.Callee, err)
		}
		args := make([]int, 0, len(e.Args))
		argTypes := make([]LIRType, 0, len(e.Args))
		for i, arg := range e.Args {
			v, actual, err := b.expr(arg)
			if err != nil {
				return 0, "", err
			}
			expected, err := b.lowerType(target.Params[i].Type)
			if err != nil || (!lirCallScalar(expected) && !lirAggregateType(expected) && !lirReferenceType(expected)) || actual != expected {
				return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_CALL_ARGUMENT %s arg=%d", e.Callee, i)
			}
			args = append(args, v)
			argTypes = append(argTypes, expected)
		}
		if resultType != "void" && !lirCallScalar(resultType) && !lirAggregateType(resultType) {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_CALL_RESULT %s: %s", e.Callee, resultType)
		}
		result := 0
		if resultType == "void" {
			result = -1
		}
		id := b.emit(LIRInstruction{Op: "call", Result: result, Type: resultType, Args: args, Slot: -1, Callee: target.Name, CallTarget: evt1FunctionProvenanceKey(*target), CallABI: "win64", ArgTypes: argTypes, Source: e.Span})
		return id, resultType, nil
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
		t, _ := b.lowerType(v.typ)
		if v.typ.isReference() {
			target := lirAddressTarget(t)
			return b.emit(LIRInstruction{Op: "load", Result: 0, Type: target, Args: []int{v.value}, Slot: -1, Source: e.Span}), target, nil
		}
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
		if e.UnitScaleDenominator != 0 && e.UnitScaleNumerator != e.UnitScaleDenominator {
			return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_QUANTITY_SCALE %d/%d at %d:%d", e.UnitScaleNumerator, e.UnitScaleDenominator, e.Span.Line, e.Span.Column)
		}
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
		if lirFloating(lt) {
			op = map[string]string{"+": "fadd", "-": "fsub", "*": "fmul", "/": "fdiv", "==": "eq", "!=": "ne", "<": "lt", "<=": "le", ">": "gt", ">=": "ge"}[e.Op]
		}
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
			addr, sourceType, err := b.objectAddress(e)
			if err != nil {
				return 0, "", err
			}
			t, err := b.lowerType(sourceType)
			if err != nil {
				return 0, "", err
			}
			return b.emit(LIRInstruction{Op: "load", Result: 0, Type: t, Args: []int{addr}, Slot: -1, Source: e.Span}), t, nil
		}
		id, err := b.machineFieldID(e)
		if err != nil {
			return 0, "", err
		}
		v, typ := b.loadMachineField(id, e.Span)
		return v, typ, nil
	case *StructConstructExpr:
		return b.recordConstruction(e)
	case *WithExpr:
		return b.recordWith(e)
	default:
		return 0, "", fmt.Errorf("EVT2_UNSUPPORTED_EXPR %T at %d:%d", expr, expr.exprSpan().Line, expr.exprSpan().Column)
	}
}

// Literal bits project the existing checked semantic value exactly once to its
// declared representation. No decimal printer or native instruction selection
// participates in this target-independent input contract.
func lirFloatLiteral(t LIRType, value float64) string {
	if t == "float32" {
		return fmt.Sprintf("0x%08x", math.Float32bits(float32(value)))
	}
	return fmt.Sprintf("0x%016x", math.Float64bits(value))
}

func (b *lirBuilder) indexAddress(e *IndexExpr) (int, LIRType, error) {
	base, ok := e.Base.(*NameExpr)
	if !ok {
		return b.objectIndexAddress(e)
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
