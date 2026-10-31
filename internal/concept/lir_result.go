package concept

import "fmt"

// Logical snapshots keep already-evaluated operands valid across the new CFG.
// This does not allocate registers or physical stack storage.
func (b *lirBuilder) saveBeforePropagation(value int, typ LIRType, span Span) (int, error) {
	for _, p := range b.fn.Params {
		if p.ID == value {
			return -1, nil
		}
	}
	if lirAddressTarget(typ) != "" {
		return -1, fmt.Errorf("EVT2H5_UNSUPPORTED_REFERENCE_ACROSS_PROPAGATION")
	}
	id := len(b.fn.Slots)
	slot := LIRSlot{ID: id, Name: fmt.Sprintf("$evaluated%d", id), Type: typ, ParamValue: -1}
	if element, extent, ok := lirArrayValue(typ); ok {
		slot.Element = element
		slot.Extent = extent
	}
	b.fn.Slots = append(b.fn.Slots, slot)
	b.slotScopes = append(b.slotScopes, b.scope)
	op := "store_slot"
	if lirAggregateType(typ) || slot.Extent > 0 {
		op = "initialize_slot"
	}
	b.emit(LIRInstruction{Op: op, Result: -1, Type: typ, Args: []int{value}, Slot: id, Source: span})
	return id, nil
}
func (b *lirBuilder) loadAfterPropagation(value, slot int, typ LIRType, span Span) int {
	if slot < 0 {
		return value
	}
	return b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: typ, Slot: slot, Source: span})
}

// Checked source control flow only. The ordinary pattern contract proves the
// active variant; tagged copy, return ABI and encoding remain in Cathedral.
func (b *lirBuilder) resultPropagation(e *FailureExpr) (int, LIRType, error) {
	if e.Op != "?" {
		return 0, "", fmt.Errorf("EVT2H5_UNSUPPORTED_RESULT_OPERATOR %s", e.Op)
	}
	source := evt1CanonicalType(b.env, e.ResolvedType)
	target := evt1CanonicalType(b.env, b.mir.ReturnType)
	if !evt1IsResultType(target) {
		return 0, "", fmt.Errorf("EVT2H5_PROPAGATION_RETURN_CHANNEL")
	}
	if !evt1TypeCopyable(b.env, source) || evt1TypeHasDrop(b.env, source) {
		return 0, "", fmt.Errorf("EVT2H5_UNSUPPORTED_PROPAGATION_CLEANUP")
	}
	if e.Else == nil && !evt1FailureErrorType(source).Equal(evt1FailureErrorType(target)) {
		return 0, "", fmt.Errorf("EVT2H5_PROPAGATION_ERROR_TYPE")
	}
	success := evt1FailureSuccessType(source)
	slot, typ := -1, LIRType("void")
	var err error
	if success.Name != "void" {
		slot, typ, err = b.resultSlot(success, e.Span)
		if err != nil {
			return 0, "", err
		}
	}
	name := fmt.Sprintf("$propagate%d", len(b.fn.Slots))
	patterns := []Pattern{{EnumName: evt1ResultName, VariantName: "Ok", Span: e.Span}, {EnumName: evt1ResultName, VariantName: "Error", Bindings: []string{name + "error"}, Span: e.Span}}
	if success.Name != "void" {
		patterns[0].Bindings = []string{name + "value"}
	}
	err = b.lowerMatch(e.Value, patterns, slot, e.Span, func(arm int) error {
		if arm == 0 {
			if slot < 0 {
				return nil
			}
			return b.storeResult(slot, typ, &NameExpr{Name: name + "value", Span: e.Span}, e.Span)
		}
		var value Expr = &NameExpr{Name: name + "error", Span: e.Span}
		if e.Else != nil {
			value = e.Else
		}
		constructor := &ConstructExpr{EnumName: evt1ResultName, VariantName: "Error", Args: []Expr{value}, ResolvedType: target, Span: e.Span}
		id, t, err := b.expr(constructor)
		if err != nil {
			return err
		}
		if t != b.fn.Result {
			return fmt.Errorf("EVT2H5_PROPAGATION_RESULT_TYPE")
		}
		b.terminate(LIRTerminator{Op: "return", Value: id, Source: e.Span})
		return nil
	})
	if err != nil {
		return 0, "", err
	}
	if slot < 0 {
		return -1, "void", nil
	}
	return b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: typ, Slot: slot, Source: e.Span}), typ, nil
}
