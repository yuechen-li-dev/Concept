package concept

import "fmt"

// Logical snapshots keep already-evaluated operands valid across the new CFG.
// This does not allocate registers or physical stack storage.
// Ordinary conditional CFG does not invalidate an already checked local
// reference. Keep its existing dominance/lifetime witness. Actual failure exits
// retain the explicitly unqualified local-address boundary from h5/h7.
func (b *lirBuilder) saveBeforeExpression(value int, typ LIRType, span Span, later []Expr) (int, error) {
	if lirAddressTarget(typ) != "" && !expressionsMayFail(later) {
		if _, known := b.findReference(value); known {
			return -1, nil
		}
	}
	return b.saveBeforePropagation(value, typ, span)
}
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
	if e.Op != "?" && e.Op != "!" {
		return 0, "", fmt.Errorf("EVT2H5_UNSUPPORTED_RESULT_OPERATOR %s", e.Op)
	}
	source := evt1CanonicalType(b.env, e.ResolvedType)
	target := evt1CanonicalType(b.env, b.mir.ReturnType)
	if !evt1TypeCopyable(b.env, source) || evt1TypeHasDrop(b.env, source) {
		return 0, "", fmt.Errorf("EVT2H5_UNSUPPORTED_PROPAGATION_CLEANUP")
	}
	var handler *lirHandlerContext
	if e.Op == "?" && evt1IsResultType(source) && e.Else == nil {
		handler = b.findLocalHandler(evt1FailureErrorType(source))
	}
	if e.Op == "?" && handler == nil {
		if evt1IsOptionType(source) && !evt1IsOptionType(target) || evt1IsResultType(source) && !evt1IsResultType(target) {
			return 0, "", fmt.Errorf("EVT2H5_PROPAGATION_RETURN_CHANNEL")
		}
		if evt1IsResultType(source) && e.Else == nil && !evt1FailureErrorType(source).Equal(evt1FailureErrorType(target)) {
			return 0, "", fmt.Errorf("EVT2H5_PROPAGATION_ERROR_TYPE")
		}
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
	if evt1IsOptionType(source) {
		patterns = []Pattern{{EnumName: evt1OptionName, VariantName: "Some", Span: e.Span}, {EnumName: evt1OptionName, VariantName: "None", Span: e.Span}}
		b.hasFallibleControl = true
	}
	kind := "propagate"
	if e.Op == "!" {
		patterns = patterns[:1]
		kind = "require_success"
		b.hasFallibleControl = true
	}
	if e.Else != nil {
		kind = "remap"
	}
	if success.Name != "void" {
		patterns[0].Bindings = []string{name + "value"}
	}
	flowIndex := -1
	siteIndex := -1
	err = b.lowerMatch(e.Value, patterns, slot, e.Span, func(arm int) error {
		if flowIndex < 0 {
			for i := len(b.fn.Patterns) - 1; i >= 0; i-- {
				for _, a := range b.fn.Patterns[i].Arms {
					if a.Scope == b.scope {
						flowIndex = i
					}
				}
				if flowIndex >= 0 {
					break
				}
			}
			siteIndex = len(b.failureSites)
			b.failureSites = append(b.failureSites, LIRFailureSite{Flow: flowIndex, Kind: kind, Handler: -1})
		}
		if arm == 0 {
			if slot < 0 {
				return nil
			}
			return b.storeResult(slot, typ, &NameExpr{Name: name + "value", Span: e.Span}, e.Span)
		}
		if handler != nil {
			h, err := b.realizeHandler(handler)
			if err != nil {
				return err
			}
			b.failureSites[siteIndex].Handler = handler.record
			id, t, err := b.expr(&NameExpr{Name: name + "error", Span: e.Span})
			if err != nil {
				return err
			}
			if t != h.Type {
				return fmt.Errorf("EVT2H7_HANDLER_ERROR_TYPE")
			}
			b.emit(LIRInstruction{Op: "initialize_slot", Result: -1, Type: t, Args: []int{id}, Slot: h.InputSlot, Source: e.Span})
			b.terminate(LIRTerminator{Op: "jump", True: h.Entry, Source: e.Span})
			return nil
		}
		var value Expr = &NameExpr{Name: name + "error", Span: e.Span}
		if e.Else != nil {
			value = e.Else
		}
		constructor := &ConstructExpr{EnumName: evt1ResultName, VariantName: "Error", Args: []Expr{value}, ResolvedType: target, Span: e.Span}
		if evt1IsOptionType(source) {
			constructor.EnumName = evt1OptionName
			constructor.VariantName = "None"
			constructor.Args = nil
		}
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
