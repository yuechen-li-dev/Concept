package concept

import (
	"reflect"
	"sort"
	"strings"
)

// Private span preconditions are inferred only from the complete set of direct
// closed calls. They are not exported ABI promises or no-alias information.
type localSpanCall struct {
	key     string
	lengths []int64
}

type localBoundsCalls struct {
	expected map[Expr]string
	seen     map[Expr]localSpanCall
	disabled bool
}

func localTemplateCallKey(e Expr) string {
	var name string
	var args []Type
	switch e := e.(type) {
	case *TemplateCallExpr:
		name, args = e.Callee, e.TypeArgs
		if len(args) == 0 {
			args = []Type{e.TypeArg}
		}
	case *CallExpr:
		if len(e.InferredTemplateArgs) == 0 {
			return ""
		}
		name, args = e.Callee, e.InferredTemplateArgs
	default:
		return ""
	}
	identities := make([]string, len(args))
	for i, arg := range args {
		identities[i] = evt1TypeIdentity(arg)
	}
	return name + "|" + strings.Join(identities, "__")
}

func newLocalBoundsCalls(module Module, env *semanticEnv) *localBoundsCalls {
	c := &localBoundsCalls{expected: map[Expr]string{}, seen: map[Expr]localSpanCall{}}
	// Their implicit entry/cleanup/protocol edges are intentionally outside
	// this bounded direct-call analysis. Do not infer partial call sets.
	if len(module.Automata) != 0 {
		c.disabled = true
	}
	for _, decl := range module.Structs {
		if len(decl.Methods) != 0 {
			c.disabled = true
		}
	}
	for _, instance := range env.templateInstances {
		if len(instance.RequirementBindings) != 0 || instance.ConstraintConcept != "" {
			c.disabled = true
		}
	}
	for _, fn := range evt1ModuleWithConcreteFunctions(module, env).Functions {
		if fn.Body == nil || fn.Comptime {
			continue
		}
		evt1WalkTypedBlock(*fn.Body, func(e Expr) {
			if key := localTemplateCallKey(e); key != "" {
				c.expected[e] = key
			}
			// No inference through erased calls, captures or exported function
			// values. These need richer escape/callgraph summaries first.
			switch e := e.(type) {
			case *CallableExpr:
				c.disabled = true
			case *CallExpr:
				if e.CallableInvoke || e.DynDispatch {
					c.disabled = true
				}
			case *RefExpr:
				if e.CallbackErase {
					c.disabled = true
				}
			}
		})
	}
	return c
}

func (c *localBoundsCalls) record(e Expr, values []localValue) {
	if c == nil || c.disabled {
		return
	}
	key, ok := c.expected[e]
	if !ok {
		return
	}
	lengths := make([]int64, len(values))
	for i, value := range values {
		if value.lengthHeld {
			lengths[i] = value.lengthMin
		}
	}
	c.seen[e] = localSpanCall{key, lengths}
}

func (c *localBoundsCalls) bounds(env *semanticEnv) map[string][]int64 {
	out := map[string][]int64{}
	if c.disabled {
		return out
	}
	for expr, key := range c.expected {
		instance := env.templateInstances[key]
		if instance == nil || instance.Function.Async || instance.ConstraintConcept != "" || len(instance.RequirementBindings) != 0 {
			continue
		}
		call, seen := c.seen[expr]
		lengths := make([]int64, len(instance.Function.Params))
		if seen && len(call.lengths) == len(lengths) {
			copy(lengths, call.lengths)
		}
		for i, p := range instance.Function.Params {
			if !evt1IsSpanType(p.Type) || p.Type.isReference() {
				lengths[i] = 0
			}
		}
		if previous, ok := out[key]; ok {
			for i := range lengths {
				lengths[i] = min(lengths[i], previous[i])
			}
		}
		out[key] = lengths
	}
	return out
}

func inferLocalBounds(module Module, env *semanticEnv) {
	calls := newLocalBoundsCalls(module, env)
	inputs := map[string][]int64{}
	// Bounded propagation is pessimistic from zero. Failure to reach a fixed
	// point loses opportunities; it never authorizes a stronger assumption.
	for round := 0; round < 4; round++ {
		calls.seen = map[Expr]localSpanCall{}
		env.localSpanInputs = inputs
		for _, fn := range module.Functions {
			proveFunctionLocalBounds(fn, env, nil, calls)
		}
		keys := make([]string, 0, len(env.templateInstances))
		for key := range env.templateInstances {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			proveFunctionLocalBounds(env.templateInstances[key].Function, env, inputs[key], calls)
		}
		next := calls.bounds(env)
		if reflect.DeepEqual(next, inputs) {
			return
		}
		inputs = next
	}
}

// A tiny constant-result summary for straight-line checked integer helpers.
// Calls are retained at runtime. This is abstract interpretation, not execution
// of arbitrary functions, and it cannot consult globals, loops or foreign code.
func (a *localBoundsAnalysis) constantCall(call *CallExpr, args []localValue) localValue {
	if call.Member || call.CallableInvoke || call.DynDispatch || call.resolvedSignature == "" {
		return localValue{}
	}
	var found *FunctionDecl
	for _, candidate := range a.env.functions[call.Callee] {
		if evt1FunctionParamSignature(candidate) == call.resolvedSignature {
			copy := candidate
			found = &copy
			break
		}
	}
	if found == nil || found.Body == nil || found.ReturnType.Name != "int" || found.ExternABI != "" || found.Async || len(args) != len(found.Params) {
		return localValue{}
	}
	if a.summaries == nil {
		a.summaries = map[string]bool{}
	}
	id := evt1FunctionProvenanceKey(*found)
	if a.summaries[id] || len(a.summaries) >= 4 {
		return localValue{}
	}
	a.summaries[id] = true
	defer delete(a.summaries, id)
	state := localBoundsState{}
	for i, p := range found.Params {
		if p.Type.Name != "int" || p.Type.isReference() || !args[i].intKnown || args[i].integer.lo != args[i].integer.hi {
			return localValue{}
		}
		state[p.Name] = args[i]
	}
	for _, stmt := range found.Body.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			if s.Type.Name != "int" || s.Type.isReference() {
				return localValue{}
			}
			v := a.value(s.Value, state)
			if !v.intKnown {
				return localValue{}
			}
			state[s.Name] = v
		case *ReturnStmt:
			return a.value(s.Value, state)
		default:
			return localValue{}
		}
	}
	return localValue{}
}

func (a *localBoundsAnalysis) constantBool(expr Expr, s localBoundsState) (bool, bool) {
	if value, ok := expr.(*BoolLiteral); ok {
		return value.Value, true
	}
	if p, ok := expr.(*ParenExpr); ok {
		return a.constantBool(p.Value, s)
	}
	b, ok := expr.(*BinaryExpr)
	if !ok {
		return false, false
	}
	x, y := a.value(b.Left, s), a.value(b.Right, s)
	if !x.intKnown || !y.intKnown || x.integer.lo != x.integer.hi || y.integer.lo != y.integer.hi {
		return false, false
	}
	l, r := x.integer.lo, y.integer.lo
	switch b.Op {
	case "<":
		return l < r, true
	case "<=":
		return l <= r, true
	case ">":
		return l > r, true
	case ">=":
		return l >= r, true
	case "==":
		return l == r, true
	case "!=":
		return l != r, true
	}
	return false, false
}
