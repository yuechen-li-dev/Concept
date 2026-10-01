package concept

type evt1EvaluationMode uint8

const (
	evt1RuntimeEvaluation evt1EvaluationMode = iota
	evt1ComptimeEvaluation
)

// The context is inherited by scopes, including helpers that revisit an
// expression to check a call argument or an assignable access path. The public
// validator's older flag signatures remain at the boundary during migration.
type evt1ValidationContext struct {
	Template   *evt1TemplateInfo
	Evaluation evt1EvaluationMode
}

func evt1ExpressionContext(template *evt1TemplateInfo, comptime bool) evt1ValidationContext {
	context := evt1ValidationContext{Template: template}
	if comptime {
		context.Evaluation = evt1ComptimeEvaluation
	}
	return context
}

func evt1EnterValidationContext(scope *evt1Scope, template *evt1TemplateInfo, comptime bool) func() {
	if scope == nil {
		return func() {}
	}
	previous := scope.context
	scope.context = evt1ExpressionContext(template, comptime)
	return func() { scope.context = previous }
}

func evt1ValidationIsComptime(scope *evt1Scope) bool {
	return scope != nil && scope.context.Evaluation == evt1ComptimeEvaluation
}
