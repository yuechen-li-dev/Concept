package concept

import "fmt"

// Source short-circuit semantics, expressed using the existing bool slots and
// CFG. No logical opcode, physical location or backend policy is introduced.
func (b *lirBuilder) logicalExpression(e *BinaryExpr) (int, LIRType, error) {
	typ, _ := evt1BuiltinType("bool", e.Span)
	slot := b.slot(fmt.Sprintf("$logical%d", len(b.fn.Slots)), typ)
	left, lt, err := b.expr(e.Left)
	if err != nil {
		return 0, "", err
	}
	if lt != "bool" {
		return 0, "", fmt.Errorf("EVT2H8_LOGICAL_REQUIRES_BOOL")
	}
	store := func(value int) {
		op := "store_slot"
		if len(b.fn.Patterns) > 0 {
			op = "initialize_slot"
		}
		b.emit(LIRInstruction{Op: op, Result: -1, Type: "bool", Args: []int{value}, Slot: slot, Source: e.Span})
	}
	store(left)
	rhs, join := b.newBlock(), b.newBlock()
	yes, no := rhs, join
	if e.Op == "or" {
		yes, no = join, rhs
	}
	b.terminate(LIRTerminator{Op: "branch", Value: left, True: yes, False: no, Source: e.Span})
	b.current = rhs
	right, rt, err := b.expr(e.Right)
	if err != nil {
		return 0, "", err
	}
	if rt != "bool" {
		return 0, "", fmt.Errorf("EVT2H8_LOGICAL_REQUIRES_BOOL")
	}
	store(right)
	b.terminate(LIRTerminator{Op: "jump", True: join, Source: e.Span})
	b.current = join
	return b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: "bool", Slot: slot, Source: e.Span}), "bool", nil
}

func (b *lirBuilder) whileStatement(s *WhileStmt) error {
	condition, body, exit := b.newBlock(), b.newBlock(), b.newBlock()
	b.terminate(LIRTerminator{Op: "jump", True: condition, Source: s.Span})
	b.current = condition
	value, typ, err := b.expr(s.Condition)
	if err != nil {
		return err
	}
	if typ != "bool" {
		return fmt.Errorf("EVT2H8_WHILE_REQUIRES_BOOL")
	}
	b.terminate(LIRTerminator{Op: "branch", Value: value, True: body, False: exit, Source: s.Span})
	b.current = body
	if err := b.block(s.Body); err != nil {
		return err
	}
	if b.fn.Blocks[b.current].Term.Op == "" {
		b.terminate(LIRTerminator{Op: "jump", True: condition, Source: s.Span})
	}
	b.current = exit
	return nil
}
