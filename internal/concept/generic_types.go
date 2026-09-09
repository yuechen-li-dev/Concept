package concept

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// evt1MaterializeGenericInstances preserves monomorphized declarations across
// the public Parse -> Generate boundary. Parse and Generate intentionally run
// semantic analysis independently, so retaining only an analysis-local cache
// would leave concrete type names unresolvable during generation.
func evt1MaterializeGenericInstances(module *Module, env *semanticEnv) {
	existingStructs := map[string]bool{}
	for _, decl := range module.Structs {
		existingStructs[decl.Name] = true
	}
	existingFunctions := map[string]bool{}
	for _, fn := range module.Functions {
		existingFunctions[fn.Name] = true
	}
	var names []string
	for name := range env.genericTypeInstances {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		decl := env.genericTypeInstances[name]
		if !existingStructs[decl.Name] {
			module.Structs = append(module.Structs, decl)
			existingStructs[decl.Name] = true
		}
		for _, method := range decl.Methods {
			if !existingFunctions[method.Name] {
				module.Functions = append(module.Functions, method)
				existingFunctions[method.Name] = true
			}
		}
	}
}

func evt1InstantiateGenericType(env *semanticEnv, application Type) (Type, error) {
	decl, ok := env.genericTypes[application.Name]
	if !ok {
		return Type{}, evt1Diagnostic("GENERIC_UNKNOWN", "unknown generic type "+application.Name, application.Span)
	}
	if len(application.TypeArgs) != len(decl.Parameters) {
		return Type{}, evt1Diagnostic("GENERIC_ARGUMENT_COUNT", fmt.Sprintf("%s requires %d template argument(s), got %d", decl.Name, len(decl.Parameters), len(application.TypeArgs)), application.Span)
	}
	if decl.Constraint.ConceptName != "" {
		parameterIndex := -1
		for i, parameter := range decl.Parameters {
			if parameter.Name == decl.Constraint.TypeArg.Name {
				parameterIndex = i
				break
			}
		}
		if parameterIndex < 0 || decl.Parameters[parameterIndex].Kind != "type" {
			return Type{}, evt1Diagnostic("GENERIC_CONSTRAINT_INVALID", "generic type constraint must target a type parameter", decl.Constraint.Span)
		}
		if application.TypeArgs[parameterIndex].Kind != TypeConceptParam {
			if err := checkConceptSatisfaction(env, decl.Constraint.ConceptName, application.TypeArgs[parameterIndex], nil, application.Span); err != nil {
				return Type{}, err
			}
		}
	}
	parts := make([]string, len(application.TypeArgs))
	for i, arg := range application.TypeArgs {
		if decl.Parameters[i].Kind == "type" {
			if arg.Kind == TypeTemplateValue {
				return Type{}, evt1Diagnostic("GENERIC_ARGUMENT_KIND", "type template parameter requires a type", arg.Span)
			}
			if arg.Kind != TypeConceptParam {
				if err := validateKnownType(env, arg, arg.Span, "", false); err != nil {
					return Type{}, err
				}
			}
		} else {
			if arg.Kind != TypeTemplateValue {
				return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", "non-type template parameter requires a compile-time integer", arg.Span)
			}
		}
		parts[i] = arg.String()
	}
	identity := decl.Name + "<" + strings.Join(parts, ", ") + ">"
	if instance, ok := env.genericTypeInstances[identity]; ok {
		env.structs[identity] = instance
		return evt1AppliedConcreteType(application, identity), nil
	}
	if env.genericInstantiating[identity] {
		return Type{}, evt1Diagnostic("GENERIC_RECURSIVE_INSTANTIATION", "recursive generic instantiation: "+identity, application.Span)
	}
	env.genericInstantiating[identity] = true
	defer delete(env.genericInstantiating, identity)

	instance := decl.Struct
	instance.Name = identity
	for i := range instance.Fields {
		fieldType := instance.Fields[i].Type
		for j, param := range decl.Parameters {
			if param.Kind == "type" {
				fieldType = evt1SubstituteType(fieldType, param.Name, application.TypeArgs[j])
			}
		}
		fieldType = evt1SubstituteGenericValueExtents(fieldType, decl.Parameters, application.TypeArgs)
		resolved, err := evt1ResolveType(env, nil, fieldType)
		if err != nil {
			return Type{}, err
		}
		instance.Fields[i].Type = resolved
	}
	for i := range instance.Methods {
		method := instance.Methods[i]
		for j, param := range decl.Parameters {
			method.ReturnType = evt1SubstituteType(method.ReturnType, param.Name, application.TypeArgs[j])
			for k := range method.Params {
				method.Params[k].Type = evt1SubstituteType(method.Params[k].Type, param.Name, application.TypeArgs[j])
			}
			if method.Body != nil {
				body, err := evt1SubstituteBlock(*method.Body, param.Name, application.TypeArgs[j])
				if err != nil {
					return Type{}, err
				}
				method.Body = &body
			}
		}
		method.MethodOf = identity
		for k := range method.Params {
			if method.Params[k].Name == "self" {
				method.Params[k].Type.Name = identity
				method.Params[k].Type.Kind = TypeStruct
			}
		}
		instance.Methods[i] = method
	}
	env.structs[identity] = instance
	env.genericTypeInstances[identity] = instance
	for _, method := range instance.Methods {
		duplicate := false
		for _, existing := range env.functions[method.Name] {
			if existing.MethodOf == method.MethodOf {
				duplicate = true
				break
			}
		}
		if !duplicate {
			env.functions[method.Name] = append(env.functions[method.Name], method)
		}
	}
	fields := map[string]Type{}
	for _, field := range instance.Fields {
		fields[field.Name] = field.Type
	}
	env.fieldSets[identity] = fields
	return evt1AppliedConcreteType(application, identity), nil
}

func evt1SubstituteGenericValueExtents(t Type, params []GenericParameter, args []Type) Type {
	values := map[string]int{}
	for i, param := range params {
		if param.Kind == "value" {
			value, _ := strconv.Atoi(args[i].Name)
			values[param.Name] = value
		}
	}
	replace := func(expr Expr) Expr {
		if name, ok := expr.(*NameExpr); ok {
			if value, found := values[name.Name]; found {
				return &IntLiteral{Value: value, Span: name.Span}
			}
		}
		return expr
	}
	if t.ArrayLengthExpr != nil {
		t.ArrayLengthExpr = replace(t.ArrayLengthExpr)
	}
	for i := range t.Shape {
		if t.Shape[i].Expr != nil {
			t.Shape[i].Expr = replace(t.Shape[i].Expr)
		}
	}
	if t.ArrayElem != nil {
		elem := evt1SubstituteGenericValueExtents(*t.ArrayElem, params, args)
		t.ArrayElem = &elem
	}
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1SubstituteGenericValueExtents(t.TypeArgs[i], params, args)
	}
	return t
}

func evt1AppliedConcreteType(source Type, identity string) Type {
	return Type{Name: identity, Kind: TypeStruct, Ownership: source.Ownership, Const: source.Const, Scoped: source.Scoped, Imported: source.Imported, Unsafe: source.Unsafe, Span: source.Span}
}
