package concept

import (
	"encoding/json"
	"sort"
	"strconv"
)

// GenericApplication retains the defining declaration and ordered arguments of
// a closed nominal type. Names on Type and StructDecl are symbol/display names,
// never a serialization of semantic identity that consumers must parse.
type GenericApplication struct {
	Declaration GenericDeclarationIdentity `json:"declaration"`
	Arguments   []GenericArgument          `json:"arguments"`
}

type GenericDeclarationIdentity struct {
	Module string `json:"module"`
	Name   string `json:"name"`
}

type GenericArgument struct {
	Kind    string `json:"kind"`
	Type    Type   `json:"type"`
	Integer int64  `json:"integer,omitempty"`
}

type SemanticGenericApplication struct {
	NominalName string              `json:"nominal_name"`
	Application *GenericApplication `json:"application"`
}

func semanticGenericApplications(module Module) []SemanticGenericApplication {
	var applications []SemanticGenericApplication
	for _, decl := range module.Structs {
		if decl.Application != nil {
			applications = append(applications, SemanticGenericApplication{NominalName: decl.Name, Application: decl.Application})
		}
	}
	sort.Slice(applications, func(i, j int) bool { return applications[i].NominalName < applications[j].NominalName })
	return applications
}

func evt1StructCName(decl StructDecl) string {
	if decl.Application != nil {
		return evt1GenericCName(decl.Application)
	}
	return evt1CName(decl.Name)
}

func evt1GenericCName(application *GenericApplication) string {
	return evt1CName(application.Type().String()) + "__g_" + digest([]byte(evt1GenericApplicationKey(application)))[:16]
}

func evt1NewGenericApplication(decl GenericTypeDecl, args []Type) *GenericApplication {
	application := &GenericApplication{Declaration: GenericDeclarationIdentity{Module: decl.Module, Name: decl.Name}}
	for i, arg := range args {
		argument := GenericArgument{Kind: decl.Parameters[i].Kind, Type: evt1GenericMetadataType(arg, true)}
		if argument.Kind == "value" {
			// Resolution has already checked the integer and its parameter type.
			argument.Integer, _ = strconv.ParseInt(arg.Name, 10, 64)
			argument.Type = evt1GenericMetadataType(decl.Parameters[i].ValueType, true)
		}
		application.Arguments = append(application.Arguments, argument)
	}
	return application
}

// Type projects the stored application into the existing open-pattern and
// substitution machinery. This projection never reads a nominal display name.
func (a *GenericApplication) Type() Type {
	t := Type{Name: a.Declaration.Name, Kind: TypeApplied}
	for _, arg := range a.Arguments {
		if arg.Kind == "value" {
			t.TypeArgs = append(t.TypeArgs, Type{Name: strconv.FormatInt(arg.Integer, 10), Kind: TypeTemplateValue})
		} else {
			t.TypeArgs = append(t.TypeArgs, arg.Type)
		}
	}
	return t
}

func evt1GenericApplicationOf(env *semanticEnv, t Type) (Type, bool) {
	if t.Application != nil {
		return t.Application.Type(), true
	}
	if env != nil {
		if decl, ok := env.structs[t.Name]; ok && decl.Application != nil {
			return decl.Application.Type(), true
		}
	}
	return Type{}, false
}

func evt1GenericApplicationsEqual(a, b *GenericApplication) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Declaration != b.Declaration || len(a.Arguments) != len(b.Arguments) {
		return false
	}
	for i, left := range a.Arguments {
		right := b.Arguments[i]
		if left.Kind != right.Kind || left.Integer != right.Integer || !left.Type.Equal(right.Type) {
			return false
		}
	}
	return true
}

func evt1TypeHasGenericApplication(t Type) bool {
	if t.Application != nil {
		return true
	}
	if t.PointerTo != nil && evt1TypeHasGenericApplication(*t.PointerTo) || t.ArrayElem != nil && evt1TypeHasGenericApplication(*t.ArrayElem) {
		return true
	}
	for _, arg := range append(append([]Type{}, t.TypeArgs...), t.CallableParams...) {
		if evt1TypeHasGenericApplication(arg) {
			return true
		}
	}
	return t.CallableResult != nil && evt1TypeHasGenericApplication(*t.CallableResult)
}

// Cache identity excludes source locations and the display spelling of nested
// closed applications. JSON is a deterministic structural encoding, not syntax.
func evt1GenericApplicationKey(a *GenericApplication) string {
	copy := *a
	copy.Arguments = append([]GenericArgument(nil), a.Arguments...)
	for i := range copy.Arguments {
		copy.Arguments[i].Type = evt1GenericIdentityType(copy.Arguments[i].Type)
	}
	body, err := json.Marshal(copy)
	if err != nil {
		panic(err) // Type has no JSON fields that can fail to encode.
	}
	return string(body)
}

func evt1GenericIdentityType(t Type) Type {
	return evt1GenericMetadataType(t, false)
}

func evt1GenericMetadataType(t Type, retainSymbols bool) Type {
	t.Span = Span{}
	t.Imported = false
	// Column marks a table field's declaration provenance. The same array
	// passed as a type argument has ordinary array identity, not field identity.
	t.Column = false
	if t.ArrayElem != nil {
		t.ArrayLengthExpr = nil
		t.Shape = append([]StorageDimension(nil), t.Shape...)
		for i := range t.Shape {
			if !t.Shape[i].Runtime {
				t.Shape[i].Expr = nil
			}
		}
	}
	if t.Application != nil {
		copy := *t.Application
		copy.Arguments = append([]GenericArgument(nil), copy.Arguments...)
		for i := range copy.Arguments {
			copy.Arguments[i].Type = evt1GenericMetadataType(copy.Arguments[i].Type, retainSymbols)
		}
		t.Application = &copy
		if !retainSymbols {
			t.Name = ""
		}
	}
	if t.PointerTo != nil {
		base := evt1GenericMetadataType(*t.PointerTo, retainSymbols)
		t.PointerTo = &base
	}
	if t.ArrayElem != nil {
		base := evt1GenericMetadataType(*t.ArrayElem, retainSymbols)
		t.ArrayElem = &base
	}
	t.TypeArgs = append([]Type(nil), t.TypeArgs...)
	for i := range t.TypeArgs {
		t.TypeArgs[i] = evt1GenericMetadataType(t.TypeArgs[i], retainSymbols)
	}
	t.CallableParams = append([]Type(nil), t.CallableParams...)
	for i := range t.CallableParams {
		t.CallableParams[i] = evt1GenericMetadataType(t.CallableParams[i], retainSymbols)
	}
	if t.CallableResult != nil {
		base := evt1GenericMetadataType(*t.CallableResult, retainSymbols)
		t.CallableResult = &base
	}
	return t
}
