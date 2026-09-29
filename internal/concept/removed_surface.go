package concept

// The M-era signal automata dialect and its effect system were retired in
// favor of Core step machines (VK5). Each removed spelling is rejected with
// a diagnostic that names its replacement.

var evt1RemovedSurfaces = map[string]struct{ code, message string }{
	"signal automata": {"REMOVED_SIGNAL_AUTOMATA", "`automata X(Signal, borrow context: T)` was removed; write `automata X with input Signal` and move the context into `with state { ... }`"},
	"initial":         {"REMOVED_INITIAL", "`initial` was removed; the first machine and the first state of each machine are initial"},
	"finish":          {"REMOVED_FINISH", "`finish;` was removed; enter a `terminal state`, or write `complete;`"},
	"on goto":         {"REMOVED_ON_GOTO", "`on Signal::A goto S;` was removed; write `on Signal::A => S;` or `on Signal::A => { push M goto S; }`"},
	"dispatch":        {"REMOVED_DISPATCH", "`dispatch(instance, signal)` was removed; write `Step(instance, Machine, input)`, which returns a must-use StepOutcome"},
	"effect":          {"REMOVED_EFFECT", "`effect` declarations were removed; declare the effects as an enum and push them to a Standard.Collection.Outbox"},
	"effects":         {"REMOVED_EFFECT", "`effects` batches were removed; pass a Standard.Collection.Outbox to the automata as `scoped ref` state"},
	"emit":            {"REMOVED_EFFECT", "`emit` was removed; push the effect to a Standard.Collection.Outbox"},
	"actuator":        {"REMOVED_ACTUATOR", "`actuator`, `actuation`, and `actuate` were removed; an actuator is an ordinary function that matches exhaustively over the effect enum"},
}

func evt1RemovedSurface(name string, span Span) error {
	removed := evt1RemovedSurfaces[name]
	return evt1Diagnostic(removed.code, removed.message, span)
}
