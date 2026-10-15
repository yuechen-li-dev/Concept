package concept

// Unqualified value-query spellings are fallback builtins, not reserved names.
// Namespace projection has already resolved qualified declarations. Structured
// storage/machine operations and compiler.* observations retain their own laws.
func evt1ImplicitValueIntrinsic(env *semanticEnv, call *CallExpr) bool {
	if call.Member || call.CallableInvoke || len(env.functions[call.Callee]) != 0 {
		return false
	}
	_, declared := env.comptimeFunctions[call.Callee]
	return !declared
}
func evt1ValueIntrinsicInScope(env *semanticEnv, scope *evt1Scope, call *CallExpr) bool {
	if _, bound := scope.lookup(call.Callee); bound {
		return false
	}
	return evt1ImplicitValueIntrinsic(env, call)
}
