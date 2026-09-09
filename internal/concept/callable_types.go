package concept

import "fmt"

// evt1PrepareExactCallableTypes is the bounded exact-callable type-surface prepass. It
// infers exact callable returns first, resolves unevaluated typeof queries, and
// then substitutes transparent aliases into the ordinary declaration graph.
// No runtime value is evaluated or materialized here.
func evt1PrepareExactCallableTypes(env *semanticEnv, module *Module, typeNames map[string]Span) error {
	for i := range module.Functions {
		if module.Functions[i].ReturnType.Kind != TypeInferred {
			continue
		}
		inferred, err := evt1InferExactCallableReturn(env, module.Functions[i])
		if err != nil {
			return err
		}
		module.Functions[i].ReturnType = inferred
	}
	evt1RebuildFunctionTable(env, module.Functions)

	for i := range module.TypeAliases {
		decl := &module.TypeAliases[i]
		if other, exists := typeNames[decl.Name]; exists {
			return evt1Diagnostic("CALLABLE_TYPE_ALIAS_DUPLICATE", fmt.Sprintf("duplicate type declaration %s", decl.Name), other)
		}
		if _, exists := env.typeAliases[decl.Name]; exists {
			return evt1Diagnostic("CALLABLE_TYPE_ALIAS_DUPLICATE", fmt.Sprintf("duplicate exact type alias %s", decl.Name), decl.Span)
		}
		var queryType Type
		var err error
		if name, ok := decl.Query.(*NameExpr); ok && len(env.functions[name.Name]) == 1 && env.functions[name.Name][0].ReturnType.Kind == TypeCallable {
			// A unique function name queries its exact return type. This is the
			// argument-free form for factories whose parameters are references.
			queryType = env.functions[name.Name][0].ReturnType
		} else {
			queryType, err = validateExpr(env, evt1ModuleScope(env), decl.Query, nil, false)
			if err != nil {
				return evt1Diagnostic("CALLABLE_TYPE_QUERY_INVALID", "invalid unevaluated typeof query: "+err.Error(), decl.Span)
			}
		}
		queryType = evt1CanonicalType(env, queryType.valueType())
		if queryType.Kind != TypeCallable {
			return evt1Diagnostic("CALLABLE_TYPE_QUERY_INVALID", fmt.Sprintf("exact callable type aliases require a concrete callable expression, got %s", queryType.String()), decl.Span)
		}
		decl.ResolvedType = queryType
		env.typeAliases[decl.Name] = queryType
		typeNames[decl.Name] = decl.Span
	}

	evt1ApplyExactCallableTypes(module, env)
	for _, decl := range module.Structs {
		env.structs[decl.Name] = decl
	}
	for _, decl := range module.Automata {
		env.automata[decl.Name] = decl
	}
	evt1RebuildFunctionTable(env, module.Functions)
	return nil
}

func evt1InferExactCallableReturn(env *semanticEnv, fn FunctionDecl) (Type, error) {
	if fn.Body == nil {
		return Type{}, evt1Diagnostic("CALLABLE_RETURN_INFERENCE_INVALID", fmt.Sprintf("inferred return function %s requires a body", fn.Name), fn.Span)
	}
	scope := evt1ModuleScope(env)
	scope.functionName = fn.Name
	for i, param := range fn.Params {
		resolved, err := evt1ResolveType(env, nil, param.Type)
		if err != nil {
			return Type{}, err
		}
		provenance := evt1InitialParameterProvenance(env, resolved, i, scope.depth)
		scope.declare(param.Name, evt1ValueBinding{t: resolved, mutable: !param.Type.Const, state: evt1StorageInitialized, provenance: provenance, spanFacts: evt1ParameterSpanFacts(env, param.Name, resolved, provenance)})
	}
	returns := evt1ReturnExpressions(*fn.Body)
	if len(returns) == 0 {
		return Type{}, evt1Diagnostic("CALLABLE_RETURN_INFERENCE_INVALID", fmt.Sprintf("inferred return function %s has no value return", fn.Name), fn.Span)
	}
	var exact Type
	for _, expr := range returns {
		t, err := validateExpr(env, scope, expr, nil, false)
		if err != nil {
			return Type{}, err
		}
		t = evt1CanonicalType(env, t.valueType())
		if t.Kind != TypeCallable {
			return Type{}, evt1Diagnostic("CALLABLE_RETURN_INFERENCE_INVALID", fmt.Sprintf("inferred return function %s must return one exact concrete callable type", fn.Name), expr.exprSpan())
		}
		if exact.Name == "" {
			exact = t
		} else if !exact.Equal(t) {
			return Type{}, evt1Diagnostic("CALLABLE_RETURN_TYPE_MISMATCH", "all inferred returns must have the same concrete callable identity", expr.exprSpan())
		}
	}
	exact.Span = fn.ReturnType.Span
	return exact, nil
}

func evt1ReturnExpressions(block Block) []Expr {
	var out []Expr
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *ReturnStmt:
			if s.Value != nil {
				out = append(out, s.Value)
			}
		case *IfStmt:
			out = append(out, evt1ReturnExpressions(s.Then)...)
			if s.Else != nil {
				out = append(out, evt1ReturnExpressions(*s.Else)...)
			}
		case *Block:
			out = append(out, evt1ReturnExpressions(*s)...)
		}
	}
	return out
}

func evt1RebuildFunctionTable(env *semanticEnv, functions []FunctionDecl) {
	env.functions = map[string][]FunctionDecl{}
	for _, fn := range functions {
		env.functions[fn.Name] = append(env.functions[fn.Name], fn)
	}
}

// evt1ApplyExactCallableTypes makes aliases transparent to all existing
// ownership, layout, automata, MIR, and C-lowering code.
func evt1ApplyExactCallableTypes(module *Module, env *semanticEnv) {
	resolve := func(t Type) Type {
		if next, err := evt1ResolveType(env, nil, t); err == nil {
			return next
		}
		return t
	}
	for i := range module.TypeAliases {
		if t, ok := env.typeAliases[module.TypeAliases[i].Name]; ok {
			module.TypeAliases[i].ResolvedType = t
		}
	}
	for i := range module.Structs {
		for j := range module.Structs[i].Fields {
			module.Structs[i].Fields[j].Type = resolve(module.Structs[i].Fields[j].Type)
		}
		for j := range module.Structs[i].Methods {
			module.Structs[i].Methods[j].ReturnType = resolve(module.Structs[i].Methods[j].ReturnType)
			for k := range module.Structs[i].Methods[j].Params {
				module.Structs[i].Methods[j].Params[k].Type = resolve(module.Structs[i].Methods[j].Params[k].Type)
			}
		}
	}
	for i := range module.Functions {
		module.Functions[i].ReturnType = resolve(module.Functions[i].ReturnType)
		for j := range module.Functions[i].Params {
			module.Functions[i].Params[j].Type = resolve(module.Functions[i].Params[j].Type)
		}
	}
	for i := range module.Enums {
		for j := range module.Enums[i].Variants {
			for k := range module.Enums[i].Variants[j].Payload {
				module.Enums[i].Variants[j].Payload[k].Type = resolve(module.Enums[i].Variants[j].Payload[k].Type)
			}
		}
	}
	for i := range module.Automata {
		for j := range module.Automata[i].StateFields {
			module.Automata[i].StateFields[j].Type = resolve(module.Automata[i].StateFields[j].Type)
		}
		for j := range module.Automata[i].Machines {
			machine := &module.Automata[i].Machines[j]
			machine.ResultType, machine.ErrorType = resolve(machine.ResultType), resolve(machine.ErrorType)
			for k := range machine.Fields {
				machine.Fields[k].Type = resolve(machine.Fields[k].Type)
			}
		}
	}
}
