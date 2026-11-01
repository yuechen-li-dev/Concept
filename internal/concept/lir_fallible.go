package concept

import "fmt"

// Checked control associations only. All physical realization remains Concept.
type LIRFallibleContract struct {
	Tries    []LIRTryRegion
	Handlers []LIRFailureHandler
	Sites    []LIRFailureSite
}
type LIRTryRegion struct{ Scope, Join int }
type LIRFailureHandler struct {
	Try, InputSlot, BindingSlot, Entry, Scope int
	Type                                      LIRType
}
type LIRFailureSite struct {
	Flow    int
	Kind    string
	Handler int
}
type lirTryContext struct {
	parentScope, scope, join, record int
	names                            map[string]lirBinding
	handlers                         map[string]*lirHandlerContext
	outer                            []*lirTryContext
	span                             Span
}
type lirHandlerContext struct {
	arm    ExceptArm
	owner  *lirTryContext
	record int
}

// Handlers are materialized only when a checked failure can reach them. The
// input slot belongs to the surrounding scope; the immutable user binding
// belongs only to its handler. No physical storage policy is decided here.
func (b *lirBuilder) tryStatement(s *TryStmt) error {
	ctx := &lirTryContext{parentScope: b.scope, scope: len(b.scopes), join: -1, record: -1, names: b.names, handlers: map[string]*lirHandlerContext{}, outer: append([]*lirTryContext(nil), b.tryStack...), span: s.Span}
	for _, arm := range s.Except {
		arm.ErrorType = evt1CanonicalType(b.env, arm.ErrorType)
		ctx.handlers[evt1TypeIdentity(arm.ErrorType)] = &lirHandlerContext{arm: arm, owner: ctx, record: -1}
	}
	b.tryStack = append(b.tryStack, ctx)
	err := b.block(s.Body)
	b.tryStack = ctx.outer
	if err != nil {
		return err
	}
	if ctx.record < 0 {
		return nil
	}
	open := []int{}
	if b.fn.Blocks[b.current].Term.Op == "" {
		open = append(open, b.current)
	}
	for _, arm := range s.Except {
		h := ctx.handlers[evt1TypeIdentity(evt1CanonicalType(b.env, arm.ErrorType))]
		if h.record < 0 {
			continue
		}
		meta := b.failureControl.Handlers[h.record]
		b.current = meta.Entry
		b.scope = meta.Scope
		b.names = map[string]lirBinding{}
		for name, binding := range ctx.names {
			b.names[name] = binding
		}
		value := b.emit(LIRInstruction{Op: "load_slot", Result: 0, Type: meta.Type, Slot: meta.InputSlot, Source: arm.Span})
		b.emit(LIRInstruction{Op: "initialize_slot", Result: -1, Type: meta.Type, Args: []int{value}, Slot: meta.BindingSlot, Source: arm.Span})
		b.names[arm.Binding] = lirBinding{slot: meta.BindingSlot, typ: evt1CanonicalType(b.env, arm.ErrorType), local: true}
		if err := b.block(arm.Body); err != nil {
			return err
		}
		if b.fn.Blocks[b.current].Term.Op == "" {
			open = append(open, b.current)
		}
	}
	b.names, b.scope = ctx.names, ctx.parentScope
	for _, block := range open {
		b.current = block
		b.terminate(LIRTerminator{Op: "jump", True: ctx.join, Source: s.Span})
	}
	if len(open) > 0 {
		b.current = ctx.join
		b.fn.Blocks[ctx.join].Term = LIRTerminator{}
	}
	return nil
}

func (b *lirBuilder) findLocalHandler(t Type) *lirHandlerContext {
	key := evt1TypeIdentity(evt1CanonicalType(b.env, t))
	for i := len(b.tryStack) - 1; i >= 0; i-- {
		if h := b.tryStack[i].handlers[key]; h != nil {
			return h
		}
	}
	return nil
}
func (b *lirBuilder) realizeHandler(h *lirHandlerContext) (LIRFailureHandler, error) {
	if h.record >= 0 {
		return b.failureControl.Handlers[h.record], nil
	}
	ctx := h.owner
	if ctx.record < 0 {
		ctx.join = b.newBlock()
		b.fn.Blocks[ctx.join].Term = LIRTerminator{Op: "jump", True: ctx.join, Scope: ctx.parentScope, Source: ctx.span}
		ctx.record = len(b.failureControl.Tries)
		b.failureControl.Tries = append(b.failureControl.Tries, LIRTryRegion{ctx.scope, ctx.join})
	}
	t, err := b.lowerType(h.arm.ErrorType)
	if err != nil {
		return LIRFailureHandler{}, err
	}
	old := b.scope
	b.scope = ctx.parentScope
	input := b.slot(fmt.Sprintf("$error%d", len(b.fn.Slots)), h.arm.ErrorType)
	scope := len(b.scopes)
	b.scopes = append(b.scopes, LIRBorrowScope{ID: scope, Parent: ctx.parentScope, Source: h.arm.Span})
	b.scope = scope
	binding := b.slot(h.arm.Binding, h.arm.ErrorType)
	b.fn.Slots[binding].Readonly = true
	b.scope = old
	meta := LIRFailureHandler{Try: ctx.record, InputSlot: input, BindingSlot: binding, Entry: b.newBlock(), Scope: scope, Type: t}
	h.record = len(b.failureControl.Handlers)
	b.failureControl.Handlers = append(b.failureControl.Handlers, meta)
	b.hasFallibleControl, b.hasBorrow = true, true
	return meta, nil
}
