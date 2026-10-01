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
	functionKey := func(fn FunctionDecl) string {
		parts := []string{fn.MethodOf, fn.Name}
		for _, parameter := range fn.Params {
			parts = append(parts, parameter.Type.String())
		}
		return strings.Join(parts, "\x00")
	}
	for _, fn := range module.Functions {
		existingFunctions[functionKey(fn)] = true
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
			key := functionKey(method)
			if !existingFunctions[key] {
				module.Functions = append(module.Functions, method)
				existingFunctions[key] = true
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
	// An application inside an open generic declaration is still symbolic.
	// Materializing it now would collapse its child argument structure into a
	// nominal string such as Owner<T>, preventing the eventual T -> Concrete
	// binding from reaching the nested application. Closed applications are
	// monomorphized below through the ordinary cache.
	if evt1TypeContainsConceptParameter(application) {
		return application, nil
	}
	if decl.Constraint.ConceptName != "" {
		bindings := evt1TemplateBindings(decl.Parameters, application.TypeArgs)
		constraintArguments := evt1SubstituteArguments(evt1ConstraintArguments(decl.Constraint), bindings)
		open := false
		for _, argument := range constraintArguments {
			open = open || evt1TypeContainsConceptParameter(argument)
		}
		if !open {
			if err := checkConceptApplicationSatisfaction(env, decl.Constraint.ConceptName, constraintArguments, nil, application.Span); err != nil {
				return Type{}, err
			}
		}
	}
	parts := make([]string, len(application.TypeArgs))
	for i, arg := range application.TypeArgs {
		if decl.Parameters[i].Kind == "value" {
			resolved, err := evt1ResolveGenericValueArgument(env, arg, decl.Parameters[i].ValueType)
			if err != nil {
				return Type{}, err
			}
			arg = resolved
			application.TypeArgs[i] = resolved
		}
		if decl.Parameters[i].Kind == "type" {
			resolved, err := evt1ResolveType(env, nil, arg)
			if err != nil {
				return Type{}, err
			}
			arg = resolved
			application.TypeArgs[i] = resolved
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
	metadata := evt1NewGenericApplication(decl, application.TypeArgs)
	key := evt1GenericApplicationKey(metadata)
	if name, ok := env.genericTypeKeys[key]; ok {
		return evt1AppliedConcreteType(application, name, metadata), nil
	}
	if previous, ok := env.structs[identity]; ok && !evt1GenericApplicationsEqual(previous.Application, metadata) {
		return Type{}, evt1Diagnostic("GENERIC_IDENTITY_COLLISION", "distinct generic applications have the same display spelling: "+identity, application.Span)
	}
	if env.genericInstantiating[key] {
		return Type{}, evt1Diagnostic("GENERIC_RECURSIVE_INSTANTIATION", "recursive generic instantiation: "+identity, application.Span)
	}
	env.genericInstantiating[key] = true
	defer delete(env.genericInstantiating, key)

	instance := evt1CloneStructDecl(decl.Struct)
	instance.Name = identity
	instance.Application = metadata
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
	if instance.Table && instance.TableSized && len(instance.Fields) != 0 {
		instance.TableCardinality = instance.Fields[0].Type.ArrayLength
		instance.TableCardinalityExpression = fmt.Sprintf("%d", instance.TableCardinality)
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
		// Non-type parameters can occur anywhere a method type can occur, not
		// only in the aggregate's fields. Close array extents and nested generic
		// arguments before registering the concrete method overload.
		method.ReturnType = evt1SubstituteGenericValueExtents(method.ReturnType, decl.Parameters, application.TypeArgs)
		for k := range method.Params {
			method.Params[k].Type = evt1SubstituteGenericValueExtents(method.Params[k].Type, decl.Parameters, application.TypeArgs)
		}
		method.MethodOf = identity
		for k := range method.Params {
			if method.Params[k].Name == "self" {
				method.Params[k].Type.Name = identity
				method.Params[k].Type.Kind = TypeStruct
				method.Params[k].Type.Application = metadata
			}
		}
		instance.Methods[i] = method
	}
	for _, field := range instance.Fields {
		if err := evt1RequireClosedType(field.Type, identity+"."+field.Name, application.Span); err != nil {
			return Type{}, err
		}
	}
	for _, method := range instance.Methods {
		if err := evt1RequireClosedType(method.ReturnType, identity+"."+method.Name+" result", application.Span); err != nil {
			return Type{}, err
		}
		for _, parameter := range method.Params {
			if err := evt1RequireClosedType(parameter.Type, identity+"."+method.Name+" parameter "+parameter.Name, application.Span); err != nil {
				return Type{}, err
			}
		}
	}
	env.structs[identity] = instance
	env.genericTypeInstances[identity] = instance
	env.genericTypeKeys[key] = identity
	env.genericTypeApplications[identity] = application
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
	return evt1AppliedConcreteType(application, identity, metadata), nil
}

func evt1SubstituteGenericValueExtents(t Type, params []GenericParameter, args []Type) Type {
	values := map[string]int{}
	for i, param := range params {
		if param.Kind == "value" {
			value, err := strconv.Atoi(args[i].Name)
			if err != nil {
				// Invalid value arguments are diagnosed before substitution; do
				// not turn an unexpected parse failure into a semantic zero.
				continue
			}
			values[param.Name] = value
		}
	}
	if value, found := values[t.Name]; found && t.PointerTo == nil && t.ArrayElem == nil && len(t.TypeArgs) == 0 && len(t.CallableParams) == 0 && t.CallableResult == nil {
		t.Name = strconv.Itoa(value)
		t.Kind = TypeTemplateValue
		return t
	}
	var replace func(Expr) Expr
	replace = func(expr Expr) Expr {
		switch e := expr.(type) {
		case *NameExpr:
			if value, found := values[e.Name]; found {
				if value < 0 {
					return &IntLiteral{Magnitude: uint64(-int64(value)), Negative: true, Lexeme: fmt.Sprint(-int64(value)), Span: e.Span}
				}
				return &IntLiteral{Magnitude: uint64(value), Lexeme: fmt.Sprint(value), Span: e.Span}
			}
		case *BinaryExpr:
			copy := *e
			copy.Left, copy.Right = replace(e.Left), replace(e.Right)
			return &copy
		case *ParenExpr:
			copy := *e
			copy.Value = replace(e.Value)
			return &copy
		}
		return expr
	}
	if t.Kind == TypeTemplateValue {
		if value, found := values[t.Name]; found {
			t.Name = strconv.Itoa(value)
		}
		return t
	}
	if t.ArrayLengthExpr != nil {
		t.ArrayLengthExpr = replace(t.ArrayLengthExpr)
	}
	t.Shape = append([]StorageDimension(nil), t.Shape...)
	for i := range t.Shape {
		if t.Shape[i].Expr != nil {
			t.Shape[i].Expr = replace(t.Shape[i].Expr)
		}
	}
	if t.ArrayElem != nil {
		elem := evt1SubstituteGenericValueExtents(*t.ArrayElem, params, args)
		t.ArrayElem = &elem
	}
	t.TypeArgs = append([]Type(nil), t.TypeArgs...)
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1SubstituteGenericValueExtents(t.TypeArgs[i], params, args)
	}
	t.CallableParams = append([]Type(nil), t.CallableParams...)
	for i := range t.CallableParams {
		t.CallableParams[i] = evt1SubstituteGenericValueExtents(t.CallableParams[i], params, args)
	}
	if t.CallableResult != nil {
		result := evt1SubstituteGenericValueExtents(*t.CallableResult, params, args)
		t.CallableResult = &result
	}
	return t
}

func evt1AppliedConcreteType(source Type, identity string, application *GenericApplication) Type {
	return Type{Name: identity, Kind: TypeStruct, Application: application, Ownership: source.Ownership, Const: source.Const, Scoped: source.Scoped, Imported: source.Imported, Unsafe: source.Unsafe, Span: source.Span}
}

// evt1StructView returns the field structure of either a concrete nominal
// struct or an applied generic struct which is still open. It does not cache or
// materialize the open application.
func evt1StructView(env *semanticEnv, t Type) (StructDecl, bool) {
	base := t.valueType()
	if decl, ok := env.structs[base.Name]; ok {
		return decl, true
	}
	if len(base.TypeArgs) == 0 {
		return StructDecl{}, false
	}
	generic, ok := env.genericTypes[base.Name]
	if !ok || len(generic.Parameters) != len(base.TypeArgs) {
		return StructDecl{}, false
	}
	decl := evt1CloneStructDecl(generic.Struct)
	decl.Name = base.String()
	for i := range decl.Fields {
		for j, parameter := range generic.Parameters {
			if parameter.Kind == "type" {
				decl.Fields[i].Type = evt1SubstituteType(decl.Fields[i].Type, parameter.Name, base.TypeArgs[j])
			}
		}
		decl.Fields[i].Type = evt1SubstituteGenericValueExtents(decl.Fields[i].Type, generic.Parameters, base.TypeArgs)
	}
	return decl, true
}

func evt1CloneStructDecl(decl StructDecl) StructDecl {
	decl.Fields = append([]Field(nil), decl.Fields...)
	decl.Methods = append([]FunctionDecl(nil), decl.Methods...)
	for i := range decl.Methods {
		decl.Methods[i].Params = append([]Param(nil), decl.Methods[i].Params...)
	}
	return decl
}
