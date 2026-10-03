package concept

import "fmt"

// Closed templates retain the ordinary instantiation identity and execute with
// the caller's evaluator state; they do not reset fuel or recursion depth.
func evt1InvokeClosedTemplateValues(state *evt1ComptimeState, instance *evt1TemplateInstance, args []Value, span Span) (Value, error) {
	fn := instance.Function
	frame := "comptime template " + instance.Key
	if fn.Body == nil || len(fn.Params) != len(args) {
		return Value{}, evt1Diagnostic("COMPTIME_TEMPLATE_BODY_MISSING", "closed comptime template requires a transported body and matching arguments", span)
	}
	active := 0
	for _, entry := range state.stack {
		if entry == frame {
			active++
		}
	}
	if active > 0 && fn.RecursionBound == 0 {
		return Value{}, evt1Diagnostic("CV4217", "recursive comptime template requires bounded(N)", span)
	}
	if fn.RecursionBound > 0 && active >= fn.RecursionBound {
		return Value{}, evt1Diagnostic("CV4206", fmt.Sprintf("comptime template %s exceeds recursion bound %d", instance.TemplateName, fn.RecursionBound), span)
	}
	if err := state.push(frame); err != nil {
		return Value{}, err
	}
	defer state.pop()
	scope := evt1SeedComptimeScope(state.env)
	for i, value := range args {
		if !evt1IsComptimeType(state.env, fn.Params[i].Type) {
			return Value{}, evt1Diagnostic("COMPTIME_TEMPLATE_UNSUPPORTED", "template argument is not a comptime-supported value", span)
		}
		scope.declare(fn.Params[i].Name, evt1EvalBinding{value: value, mutable: true, comptime: true})
	}
	result, err := evt1ExecComptimeBlock(state, scope, *fn.Body, fn.ReturnType)
	if err != nil {
		return Value{}, err
	}
	if result == nil {
		return Value{}, evt1Diagnostic("CV4212", "closed template did not return a value", span)
	}
	return *result, nil
}
