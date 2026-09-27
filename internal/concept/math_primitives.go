package concept

// These exact C11 float signatures have compiler-known, allocation-free effects.
// Other extern declarations remain unknown to the effect system.
func evt1KnownC11MathPrimitive(fn FunctionDecl) bool {
	if fn.ExternABI != "C" || fn.ReturnType.Name != "float" || fn.Body != nil {
		return false
	}
	arities := map[string]int{
		"fabsf": 1, "sqrtf": 1, "expf": 1, "logf": 1, "log2f": 1,
		"log10f": 1, "floorf": 1, "ceilf": 1, "roundf": 1, "truncf": 1,
		"sinf": 1, "cosf": 1, "tanf": 1, "asinf": 1, "acosf": 1,
		"atanf": 1, "powf": 2, "atan2f": 2,
	}
	count, ok := arities[fn.Name]
	if !ok || len(fn.Params) != count {
		return false
	}
	for _, param := range fn.Params {
		if param.Type.Name != "float" || param.Type.Ownership != "" || param.Type.Quantity != nil {
			return false
		}
	}
	return true
}

func evt1ModuleUsesC11Math(module Module) bool {
	for _, fn := range module.Functions {
		if evt1KnownC11MathPrimitive(fn) {
			return true
		}
	}
	return false
}
