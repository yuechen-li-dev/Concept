package concept

import (
	"fmt"
	"sort"
	"strings"
)

// Compile-time subjects: `declaration` and `typename` are opaque compile-time
// values that stand for a checked declaration and a checked type. They are
// inspected only through `compiler.*` observations, which project state the
// compiler already holds and never judge it. They cannot reach runtime code,
// MIR values, or C. Innate concepts (EVT2-INNATE-CONCEPTS.md) are written as
// compile-time predicates over these values.

const (
	ValueDeclaration ValueKind = "declaration"
	ValueTypename    ValueKind = "typename"

	evt1DeclarationTypeName = "declaration"
	evt1TypenameTypeName    = "typename"
)

// evt1DeclarationRef identifies one checked declaration. Parent is the
// enclosing type for a field; Index is the field's position.
type evt1DeclarationRef struct {
	Kind       DeclarationKind
	Name       string
	Owner      string
	Parent     string
	Index      int
	Provenance DeclarationProvenance
	Site       Span
}

func (r evt1DeclarationRef) qualifiedName() string {
	if r.Parent != "" {
		return r.Parent + "." + r.Name
	}
	return r.Name
}

func (r evt1DeclarationRef) equal(other evt1DeclarationRef) bool {
	return r.Kind == other.Kind && r.Name == other.Name && r.Owner == other.Owner && r.Parent == other.Parent && r.Index == other.Index && r.Site == other.Site
}

func evt1IsSubjectTypeName(name string) bool {
	return name == evt1DeclarationTypeName || name == evt1TypenameTypeName
}

// evt1TypeIsComptimeOnly reports whether t is, or contains, a compile-time
// subject. Such types exist only during compile-time evaluation.
func evt1TypeIsComptimeOnly(env *semanticEnv, t Type) bool {
	return evt1TypeIsComptimeOnlyVisiting(env, t, map[string]bool{})
}

func evt1TypeIsComptimeOnlyVisiting(env *semanticEnv, t Type, visiting map[string]bool) bool {
	if evt1IsSubjectTypeName(t.Name) && t.Kind == TypeBuiltin {
		return true
	}
	if t.ArrayElem != nil && evt1TypeIsComptimeOnlyVisiting(env, *t.ArrayElem, visiting) {
		return true
	}
	if t.PointerTo != nil && evt1TypeIsComptimeOnlyVisiting(env, *t.PointerTo, visiting) {
		return true
	}
	for _, argument := range t.TypeArgs {
		if evt1TypeIsComptimeOnlyVisiting(env, argument, visiting) {
			return true
		}
	}
	if env == nil || visiting[t.Name] {
		return false
	}
	visiting[t.Name] = true
	defer delete(visiting, t.Name)
	if decl, ok := env.structs[t.Name]; ok {
		for _, field := range decl.Fields {
			if evt1TypeIsComptimeOnlyVisiting(env, field.Type, visiting) {
				return true
			}
		}
	}
	if decl, ok := env.enums[t.Name]; ok {
		for _, variant := range decl.Variants {
			for _, field := range variant.Payload {
				if evt1TypeIsComptimeOnlyVisiting(env, field.Type, visiting) {
					return true
				}
			}
		}
	}
	return false
}

// evt1RejectComptimeOnlyType keeps compile-time subjects out of runtime code.
func evt1RejectComptimeOnlyType(env *semanticEnv, t Type, span Span, where string) error {
	if evt1TypeIsComptimeOnly(env, t) {
		return evt1Diagnostic("COMPTIME_ONLY_TYPE", fmt.Sprintf("%s %s holds a compile-time subject (declaration or typename) and exists only in comptime code", where, t.String()), span)
	}
	return nil
}

// evt1EraseComptimeOnlyTypes removes aggregates that hold compile-time
// subjects before lowering. Validation has already kept them out of runtime
// signatures and locals, so nothing lowered refers to them.
func evt1EraseComptimeOnlyTypes(env *semanticEnv, module Module) Module {
	var structs []StructDecl
	for _, decl := range module.Structs {
		if !evt1TypeIsComptimeOnly(env, Type{Name: decl.Name, Kind: TypeStruct}) {
			structs = append(structs, decl)
		}
	}
	var enums []EnumDecl
	for _, decl := range module.Enums {
		if !evt1TypeIsComptimeOnly(env, Type{Name: decl.Name, Kind: TypeEnum}) {
			enums = append(enums, decl)
		}
	}
	module.Structs, module.Enums = structs, enums
	return module
}

// ---- Values ---------------------------------------------------------------

func evt1DeclarationValue(ref evt1DeclarationRef) Value {
	return Value{Kind: ValueDeclaration, Type: Type{Name: evt1DeclarationTypeName, Kind: TypeBuiltin}, Declaration: &ref}
}

func evt1TypenameValue(t Type) Value {
	captured := t
	return Value{Kind: ValueTypename, Type: Type{Name: evt1TypenameTypeName, Kind: TypeBuiltin}, Typename: &captured}
}

// evt1TypeDeclarationRef finds the declaration of a struct or enum by its
// semantic name, including closed generic instances.
func evt1TypeDeclarationRef(env *semanticEnv, name string) (evt1DeclarationRef, bool) {
	if decl, ok := env.structs[name]; ok {
		provenance := DeclarationAuthored
		if strings.Contains(name, "<") {
			provenance = DeclarationGenerated
		}
		return evt1DeclarationRef{Kind: TypeDeclaration, Name: decl.Name, Owner: decl.Module, Index: -1, Provenance: provenance, Site: decl.Span}, true
	}
	if decl, ok := env.enums[name]; ok {
		return evt1DeclarationRef{Kind: TypeDeclaration, Name: decl.Name, Owner: decl.Module, Index: -1, Provenance: DeclarationAuthored, Site: decl.Span}, true
	}
	return evt1DeclarationRef{}, false
}

func evt1FieldDeclarationRef(env *semanticEnv, structName string, index int) (evt1DeclarationRef, bool) {
	decl, ok := env.structs[structName]
	if !ok || index < 0 || index >= len(decl.Fields) {
		return evt1DeclarationRef{}, false
	}
	parent, _ := evt1TypeDeclarationRef(env, structName)
	field := decl.Fields[index]
	return evt1DeclarationRef{Kind: FieldDeclaration, Name: field.Name, Owner: decl.Module, Parent: decl.Name, Index: index, Provenance: parent.Provenance, Site: field.Span}, true
}

// evt1InvokeComptimeFunction calls a comptime function with values the
// compiler supplies, such as declaration subjects. It is the entry point for
// compile-time predicates.
func evt1InvokeComptimeFunction(env *semanticEnv, name string, args []Value, span Span) (Value, error) {
	fn, ok := env.comptimeFunctions[name]
	if !ok {
		return Value{}, evt1Diagnostic("CV4210", fmt.Sprintf("%s is not a comptime function", name), span)
	}
	if len(fn.Params) != len(args) {
		return Value{}, evt1Diagnostic("CV4106", fmt.Sprintf("%s expects %d argument(s), got %d", name, len(fn.Params), len(args)), span)
	}
	state := newEVT1ComptimeState(env)
	if err := state.push("comptime fn " + name); err != nil {
		return Value{}, err
	}
	defer state.pop()
	scope := evt1SeedComptimeScope(env)
	for i, arg := range args {
		scope.declare(fn.Params[i].Name, evt1EvalBinding{value: arg, mutable: true, comptime: true})
	}
	returnType, err := evt1ResolveType(env, nil, fn.ReturnType)
	if err != nil {
		return Value{}, err
	}
	result, err := evt1ExecComptimeBlock(state, scope, *fn.Body, returnType)
	if err != nil {
		return Value{}, err
	}
	if result == nil {
		return Value{}, evt1Diagnostic("CV4212", fmt.Sprintf("comptime function %s did not return a value", name), fn.Span)
	}
	return *result, nil
}

// ---- Observations ---------------------------------------------------------

type evt1Observation struct {
	params []string
	result string
}

// evt1Observations is the closed observation vocabulary. Each entry is a
// projection of checked semantic state; none decides whether a program is
// valid. Adding one is a compiler change, versioned with the innate module.
var evt1Observations = map[string]evt1Observation{
	// declarations
	"Name":                 {[]string{"declaration"}, "string"},
	"QualifiedName":        {[]string{"declaration"}, "string"},
	"IsType":               {[]string{"declaration"}, "bool"},
	"IsField":              {[]string{"declaration"}, "bool"},
	"IsAuthored":           {[]string{"declaration"}, "bool"},
	"IsGenerated":          {[]string{"declaration"}, "bool"},
	"IsForeign":            {[]string{"declaration"}, "bool"},
	"Parent":               {[]string{"declaration"}, "declaration"},
	"TypeOf":               {[]string{"declaration"}, "typename"},
	"Owned":                {[]string{"declaration"}, "bool"},
	"FieldCount":           {[]string{"declaration"}, "int"},
	"Field":                {[]string{"declaration", "int"}, "declaration"},
	"HasAttribute":         {[]string{"declaration", "string"}, "bool"},
	"HasAttributeArgument": {[]string{"declaration", "string", "string"}, "bool"},
	"IsRecord":             {[]string{"declaration"}, "bool"},
	"IsClass":              {[]string{"declaration"}, "bool"},
	"IsRefStruct":          {[]string{"declaration"}, "bool"},
	"IsImmovable":          {[]string{"declaration"}, "bool"},
	"IsTable":              {[]string{"declaration"}, "bool"},
	// types
	"TypeName":         {[]string{"typename"}, "string"},
	"IsScalar":         {[]string{"typename"}, "bool"},
	"IsHandle":         {[]string{"typename"}, "bool"},
	"IsStruct":         {[]string{"typename"}, "bool"},
	"IsEnum":           {[]string{"typename"}, "bool"},
	"IsArray":          {[]string{"typename"}, "bool"},
	"IsPointer":        {[]string{"typename"}, "bool"},
	"IsBorrowLike":     {[]string{"typename"}, "bool"},
	"IsOwnedType":      {[]string{"typename"}, "bool"},
	"IsCallable":       {[]string{"typename"}, "bool"},
	"IsDyn":            {[]string{"typename"}, "bool"},
	"IsAsync":          {[]string{"typename"}, "bool"},
	"HasTypeArguments": {[]string{"typename"}, "bool"},
	"RuntimeShape":     {[]string{"typename"}, "bool"},
	"Element":          {[]string{"typename"}, "typename"},
	"Declaration":      {[]string{"typename"}, "declaration"},
	"HasDrop":          {[]string{"typename"}, "bool"},
	"NeedsDrop":        {[]string{"typename"}, "bool"},
}

func evt1IsObservationCall(e *CallExpr) bool {
	receiver, ok := e.Receiver.(*NameExpr)
	return ok && e.Member && receiver.Name == "compiler"
}

func evt1ObservationNames() string {
	names := make([]string, 0, len(evt1Observations))
	for name := range evt1Observations {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func validateObservationCall(env *semanticEnv, scope *evt1Scope, e *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if !inComptimeFn {
		return Type{}, evt1Diagnostic("OBSERVATION_RUNTIME", fmt.Sprintf("compiler.%s is a compile-time observation; call it from a comptime function", e.Callee), e.Span)
	}
	observation, ok := evt1Observations[e.Callee]
	if !ok {
		return Type{}, evt1Diagnostic("OBSERVATION_UNKNOWN", fmt.Sprintf("compiler.%s is not an observation; the observations are %s", e.Callee, evt1ObservationNames()), e.Span)
	}
	if len(e.Args) != len(observation.params) {
		return Type{}, evt1Diagnostic("OBSERVATION_ARGUMENTS", fmt.Sprintf("compiler.%s expects %d argument(s), got %d", e.Callee, len(observation.params), len(e.Args)), e.Span)
	}
	for i, arg := range e.Args {
		want, _ := evt1BuiltinType(observation.params[i], arg.exprSpan())
		if literal, ok := arg.(*IntLiteral); ok && want.Name == "int" {
			if err := evt1ResolveIntegerLiteral(literal, want); err != nil {
				return Type{}, err
			}
			continue
		}
		got, err := validateExpr(env, scope, arg, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if got.valueType().Name != want.Name || got.ArrayElem != nil {
			return Type{}, evt1Diagnostic("OBSERVATION_ARGUMENTS", fmt.Sprintf("compiler.%s argument %d must be %s, got %s", e.Callee, i+1, want.Name, got.String()), arg.exprSpan())
		}
	}
	e.Intrinsic = "observation"
	result, _ := evt1BuiltinType(observation.result, e.Span)
	return result, nil
}

func evt1ObservationError(name, message string, span Span) error {
	return evt1Diagnostic("OBSERVATION_INVALID", fmt.Sprintf("compiler.%s: %s", name, message), span)
}

func evt1EvalObservation(state *evt1ComptimeState, scope *evt1EvalScope, e *CallExpr) (Value, error) {
	env := state.env
	args := make([]Value, len(e.Args))
	for i, arg := range e.Args {
		value, err := evt1EvalExpr(state, scope, arg)
		if err != nil {
			return Value{}, err
		}
		args[i] = value
	}
	boolean := func(b bool) (Value, error) {
		t, _ := evt1BuiltinType("bool", e.Span)
		return Value{Kind: ValueBool, Type: t, BoolValue: b}, nil
	}
	text := func(s string) (Value, error) {
		t, _ := evt1BuiltinType("string", e.Span)
		return Value{Kind: ValueString, Type: t, StringValue: s}, nil
	}
	integer := func(n int) (Value, error) {
		t, _ := evt1BuiltinType("int", e.Span)
		return Value{Kind: ValueInt, Type: t, IntValue: n}, nil
	}
	fail := func(message string) (Value, error) {
		return Value{}, evt1ObservationError(e.Callee, message, e.Span)
	}

	observation := evt1Observations[e.Callee]
	if len(observation.params) > 0 && observation.params[0] == "declaration" {
		if args[0].Kind != ValueDeclaration || args[0].Declaration == nil {
			return fail("expects a declaration")
		}
		ref := *args[0].Declaration
		structDecl, isStruct := env.structs[ref.Name]
		isStruct = isStruct && ref.Kind == TypeDeclaration
		field := func() (Field, bool) {
			parent, ok := env.structs[ref.Parent]
			if ref.Kind != FieldDeclaration || !ok || ref.Index < 0 || ref.Index >= len(parent.Fields) {
				return Field{}, false
			}
			return parent.Fields[ref.Index], true
		}
		switch e.Callee {
		case "Name":
			return text(ref.Name)
		case "QualifiedName":
			return text(ref.qualifiedName())
		case "IsType":
			return boolean(ref.Kind == TypeDeclaration)
		case "IsField":
			return boolean(ref.Kind == FieldDeclaration)
		case "IsAuthored":
			return boolean(ref.Provenance == DeclarationAuthored)
		case "IsGenerated":
			return boolean(ref.Provenance == DeclarationGenerated)
		case "IsForeign":
			return boolean(ref.Provenance == DeclarationForeign)
		case "Parent":
			parent, ok := evt1TypeDeclarationRef(env, ref.Parent)
			if ref.Kind != FieldDeclaration || !ok {
				return fail(ref.qualifiedName() + " has no enclosing type")
			}
			return evt1DeclarationValue(parent), nil
		case "TypeOf":
			if f, ok := field(); ok {
				return evt1TypenameValue(f.Type), nil
			}
			if ref.Kind == TypeDeclaration {
				if _, ok := env.enums[ref.Name]; ok {
					return evt1TypenameValue(Type{Name: ref.Name, Kind: TypeEnum}), nil
				}
				if _, ok := env.structs[ref.Name]; ok {
					return evt1TypenameValue(Type{Name: ref.Name, Kind: TypeStruct}), nil
				}
			}
			return fail(ref.qualifiedName() + " has no type")
		case "Owned":
			f, ok := field()
			if !ok {
				return fail(ref.qualifiedName() + " is not a field")
			}
			return boolean(f.Type.isOwned())
		case "FieldCount":
			if !isStruct {
				return fail(ref.qualifiedName() + " is not a struct, class, or record declaration")
			}
			return integer(len(structDecl.Fields))
		case "Field":
			if !isStruct {
				return fail(ref.qualifiedName() + " is not a struct, class, or record declaration")
			}
			fieldRef, ok := evt1FieldDeclarationRef(env, ref.Name, args[1].IntValue)
			if !ok {
				return fail(fmt.Sprintf("%s has no field %d", ref.Name, args[1].IntValue))
			}
			return evt1DeclarationValue(fieldRef), nil
		case "HasAttribute", "HasAttributeArgument":
			var attributes []Attribute
			if isStruct {
				attributes = structDecl.Attributes
			} else if enumDecl, ok := env.enums[ref.Name]; ok && ref.Kind == TypeDeclaration {
				attributes = enumDecl.Attributes
			} else if f, ok := field(); ok {
				attributes = f.Attributes
			}
			for _, attribute := range attributes {
				if attribute.Name != args[1].StringValue {
					continue
				}
				if e.Callee == "HasAttribute" {
					return boolean(true)
				}
				for _, argument := range attribute.Args {
					if evt1AttributeArgumentSpelling(argument) == args[2].StringValue {
						return boolean(true)
					}
				}
			}
			return boolean(false)
		case "IsRecord", "IsClass", "IsRefStruct", "IsImmovable", "IsTable":
			if !isStruct {
				return boolean(false)
			}
			return boolean(map[string]bool{
				"IsRecord": structDecl.Record, "IsClass": structDecl.Class, "IsRefStruct": structDecl.Ref,
				"IsImmovable": structDecl.Immovable, "IsTable": structDecl.Table,
			}[e.Callee])
		}
		return fail("unhandled declaration observation")
	}

	if args[0].Kind != ValueTypename || args[0].Typename == nil {
		return fail("expects a typename")
	}
	t := *args[0].Typename
	value := t.valueType()
	switch e.Callee {
	case "TypeName":
		return text(value.String())
	case "IsScalar":
		_, builtin := evt1BuiltinType(value.Name, e.Span)
		return boolean(builtin && value.Kind == TypeBuiltin && value.ArrayElem == nil && value.PointerTo == nil && value.Name != "void" && !evt1IsSubjectTypeName(value.Name))
	case "IsHandle":
		return boolean(evt1IsHandle(env, value))
	case "IsStruct":
		_, ok := env.structs[value.Name]
		return boolean(ok && value.ArrayElem == nil && value.PointerTo == nil)
	case "IsEnum":
		_, ok := env.enums[value.Name]
		return boolean(ok && value.ArrayElem == nil && value.PointerTo == nil)
	case "IsArray":
		return boolean(t.ArrayElem != nil)
	case "IsPointer":
		return boolean(t.PointerTo != nil)
	case "IsBorrowLike":
		return boolean(t.isBorrowLike())
	case "IsOwnedType":
		return boolean(t.isOwned())
	case "IsCallable":
		return boolean(t.Kind == TypeCallable || t.Kind == TypeCallback)
	case "IsDyn":
		return boolean(t.Kind == TypeDyn)
	case "IsAsync":
		return boolean(t.Kind == TypeAsync)
	case "HasTypeArguments":
		return boolean(len(t.TypeArgs) != 0)
	case "RuntimeShape":
		return boolean(t.ArrayElem != nil && evt1StorageHasRuntimeShape(t))
	case "Element":
		if t.ArrayElem == nil {
			return fail(t.String() + " is not an array")
		}
		return evt1TypenameValue(*t.ArrayElem), nil
	case "Declaration":
		ref, ok := evt1TypeDeclarationRef(env, value.Name)
		if !ok || value.ArrayElem != nil || value.PointerTo != nil {
			return fail(t.String() + " has no struct or enum declaration")
		}
		return evt1DeclarationValue(ref), nil
	case "HasDrop":
		return boolean(evt1DropFunction(env, value) != nil)
	case "NeedsDrop":
		return boolean(evt1StorageElementHasDrop(env, value))
	}
	return fail("unhandled type observation")
}

func evt1AttributeArgumentSpelling(argument Expr) string {
	switch a := argument.(type) {
	case *NameExpr:
		return a.Name
	case *StringLiteral:
		return a.Value
	case *IntLiteral:
		return a.Lexeme
	}
	return ""
}
