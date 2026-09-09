package concept

import "sort"

// BuiltinTypeDefinition is profile-owned data consumed by the shared parser,
// semantic environment, and C lowerer. Core code does not special-case domain
// type names.
type BuiltinTypeDefinition struct {
	Name         string
	CType        string
	CDeclaration string
	Fields       map[string]Type
	NeedsHeaders []string
}

// ProfileDefinition is the deliberately small EVT1 profile registry contract.
// It describes admissions; it is not a general compiler plugin framework.
type ProfileDefinition struct {
	Name               string
	BuiltinTypes       map[string]BuiltinTypeDefinition
	BuiltinEnums       []EnumDecl
	AdmittedImports    map[string]struct{}
	AllowDomainImports bool
	AllowEffects       bool
	AllowActuators     bool
}

func evt1CoreBuiltinDefinitions() map[string]BuiltinTypeDefinition {
	return map[string]BuiltinTypeDefinition{
		"byte":   {Name: "byte", CType: "uint8_t"},
		"uint8":  {Name: "uint8", CType: "uint8_t"},
		"uint":   {Name: "uint", CType: "uint32_t"},
		"float":  {Name: "float", CType: "float"},
		"int":    {Name: "int", CType: "int"},
		"void":   {Name: "void", CType: "void"},
		"bool":   {Name: "bool", CType: "bool"},
		"string": {Name: "string", CType: "const char*"},
		"uint64": {Name: "uint64", CType: "uint64_t"},
		"usize":  {Name: "usize", CType: "size_t"},
		"isize":  {Name: "isize", CType: "ptrdiff_t"},
	}
}

var coreProfileDefinition = ProfileDefinition{
	Name:         "Core",
	BuiltinTypes: evt1CoreBuiltinDefinitions(),
	BuiltinEnums: []EnumDecl{evt1BuiltinAutomataDispatchOutcomeEnum()},
}

var profileDefinitions = map[string]*ProfileDefinition{
	coreProfileDefinition.Name:   &coreProfileDefinition,
	vulkanProfileDefinition.Name: &vulkanProfileDefinition,
}

func evt1ProfileDefinition(name string) (*ProfileDefinition, bool) {
	definition, ok := profileDefinitions[name]
	return definition, ok
}

func (p *ProfileDefinition) builtinType(name string, span Span) (Type, bool) {
	if p == nil {
		return Type{}, false
	}
	_, ok := p.BuiltinTypes[name]
	if !ok {
		return Type{}, false
	}
	return Type{Name: name, Kind: TypeBuiltin, Span: span}, true
}

func (p *ProfileDefinition) compilerOwnedType(name string) bool {
	for _, enumDecl := range p.BuiltinEnums {
		if enumDecl.Name == name {
			return true
		}
	}
	return false
}

func (p *ProfileDefinition) headersFor(module Module) []string {
	seen := map[string]bool{}
	for name, builtin := range p.BuiltinTypes {
		if !evt1TypeUsed(module, func(t Type) bool { return t.Name == name }) {
			continue
		}
		for _, header := range builtin.NeedsHeaders {
			seen[header] = true
		}
	}
	var headers []string
	for header := range seen {
		headers = append(headers, header)
	}
	sort.Strings(headers)
	return headers
}

func evt1BuiltinDefinition(name string) (BuiltinTypeDefinition, bool) {
	for _, profile := range profileDefinitions {
		if builtin, ok := profile.BuiltinTypes[name]; ok {
			return builtin, true
		}
	}
	return BuiltinTypeDefinition{}, false
}

func evt1BuiltinType(name string, span Span) (Type, bool) {
	return coreProfileDefinition.builtinType(name, span)
}
