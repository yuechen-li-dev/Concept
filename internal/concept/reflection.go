package concept

import "fmt"

// ReflectionRequest is source-level, explicit, compile-time-only intent.
// The operand is a Type AST, never a name looked up by a runtime string API.
type ReflectionRequest struct {
	Type Type `json:"type"`
	Span Span `json:"span"`
}

type FieldInfo struct {
	DeclaringType Type        `json:"declaring_type"`
	Type          Type        `json:"type"`
	Name          string      `json:"name"`
	Index         int         `json:"index"`
	Attributes    []Attribute `json:"attributes,omitempty"`
	Mutable       bool        `json:"mutable"`
	Visibility    string      `json:"visibility"`
	Span          Span        `json:"span"`
}

type EnumCaseInfo struct {
	Name       string      `json:"name"`
	Tag        int         `json:"tag"`
	Payload    []FieldInfo `json:"payload,omitempty"`
	Attributes []Attribute `json:"attributes,omitempty"`
	Span       Span        `json:"span"`
}

// TypeInfo is a one-level structural snapshot. Field types retain their
// semantic identity without recursively expanding their declarations.
type TypeInfo struct {
	Type             Type           `json:"type"`
	Kind             string         `json:"kind"`
	Name             string         `json:"name"`
	Module           string         `json:"module,omitempty"`
	GenericArguments []Type         `json:"generic_arguments,omitempty"`
	Fields           []FieldInfo    `json:"fields,omitempty"`
	EnumCases        []EnumCaseInfo `json:"enum_cases,omitempty"`
	Attributes       []Attribute    `json:"attributes,omitempty"`
	TableCardinality *int           `json:"table_cardinality,omitempty"`
	ElementType      *Type          `json:"element_type,omitempty"`
	Extent           *int           `json:"extent,omitempty"`
	ReflectionSite   Span           `json:"reflection_site"`
}

func evt1ReflectableAttributes(attributes []Attribute, span Span) error {
	for _, attribute := range attributes {
		if attribute.Name != "reflect" || len(attribute.Args) != 0 {
			return evt1Diagnostic("REFLECT_ATTRIBUTE_INVALID", "type reflection permission requires [[reflect]] without arguments", attribute.Span)
		}
	}
	return nil
}

func evt1HasReflectPermission(attributes []Attribute) bool {
	for _, attribute := range attributes {
		if attribute.Name == "reflect" {
			return true
		}
	}
	return false
}

func evt1BuildReflectionResults(module *Module, env *semanticEnv) error {
	module.ReflectionResults = nil
	for _, decl := range module.Structs {
		if err := evt1ReflectableAttributes(decl.Attributes, decl.Span); err != nil {
			return err
		}
	}
	for _, decl := range module.Enums {
		if err := evt1ReflectableAttributes(decl.Attributes, decl.Span); err != nil {
			return err
		}
	}
	for _, decl := range module.GenericTypes {
		if err := evt1ReflectableAttributes(decl.Struct.Attributes, decl.Span); err != nil {
			return err
		}
	}
	for _, request := range module.ReflectionRequests {
		info, err := evt1ReflectType(env, request)
		if err != nil {
			return err
		}
		module.ReflectionResults = append(module.ReflectionResults, info)
	}
	return nil
}

func evt1ReflectType(env *semanticEnv, request ReflectionRequest) (TypeInfo, error) {
	resolved, err := evt1ResolveType(env, nil, request.Type)
	if err != nil {
		return TypeInfo{}, err
	}
	if err := validateKnownType(env, resolved, request.Span, "", false); err != nil {
		return TypeInfo{}, err
	}
	if evt1TypeContainsConceptParameter(resolved) {
		return TypeInfo{}, evt1Diagnostic("REFLECT_OPEN_GENERIC", "reflect requires a closed type", request.Span)
	}
	info := TypeInfo{Type: resolved, Kind: string(resolved.Kind), Name: resolved.String(), GenericArguments: resolved.TypeArgs, ReflectionSite: request.Span}
	if application, ok := env.genericTypeApplications[resolved.Name]; ok {
		info.GenericArguments = append([]Type{}, application.TypeArgs...)
	} else if len(info.GenericArguments) == 0 && len(request.Type.TypeArgs) != 0 {
		info.GenericArguments = append([]Type{}, request.Type.TypeArgs...)
	}
	if resolved.ArrayElem != nil {
		info.ElementType = resolved.ArrayElem
		if len(resolved.Shape) == 0 {
			extent := resolved.ArrayLength
			info.Extent = &extent
		} else if len(resolved.Shape) == 1 && !resolved.Shape[0].Runtime {
			extent := resolved.Shape[0].Extent
			info.Extent = &extent
		}
	}
	if decl, ok := env.structs[resolved.Name]; ok {
		owner, attributes := decl.Module, decl.Attributes
		if application, ok := env.genericTypeApplications[resolved.Name]; ok {
			if generic, ok := env.genericTypes[application.Name]; ok {
				owner, attributes = generic.Module, generic.Struct.Attributes
			}
		}
		if owner != "" && owner != env.moduleName && !evt1HasReflectPermission(attributes) {
			return TypeInfo{}, evt1Diagnostic("REFLECT_PERMISSION_REQUIRED", fmt.Sprintf("%s in module %s requires [[reflect]] for cross-module structural reflection", resolved.Name, owner), request.Span)
		}
		info.Module = owner
		info.Attributes = attributes
		switch {
		case decl.Table:
			info.Kind = "table"
		case decl.Record:
			info.Kind = "record"
		case decl.Ref:
			info.Kind = "ref_struct"
		case decl.Class:
			info.Kind = "class"
		default:
			info.Kind = "struct"
		}
		if decl.Table && decl.TableSized {
			cardinality := decl.TableCardinality
			info.TableCardinality = &cardinality
		}
		for index, field := range decl.Fields {
			info.Fields = append(info.Fields, FieldInfo{DeclaringType: resolved, Type: field.Type, Name: field.Name, Index: index, Attributes: field.Attributes, Mutable: !decl.Record && !field.Type.Const, Visibility: field.Visibility, Span: field.Span})
		}
		return info, nil
	}
	if decl, ok := env.enums[resolved.Name]; ok {
		if decl.Module != "" && decl.Module != env.moduleName && !evt1HasReflectPermission(decl.Attributes) {
			return TypeInfo{}, evt1Diagnostic("REFLECT_PERMISSION_REQUIRED", fmt.Sprintf("%s in module %s requires [[reflect]] for cross-module structural reflection", resolved.Name, decl.Module), request.Span)
		}
		info.Kind, info.Module, info.Attributes = "enum", decl.Module, decl.Attributes
		for _, variant := range decl.Variants {
			caseInfo := EnumCaseInfo{Name: variant.Name, Tag: variant.Tag, Attributes: variant.Attributes, Span: variant.Span}
			for index, field := range variant.Payload {
				caseInfo.Payload = append(caseInfo.Payload, FieldInfo{DeclaringType: resolved, Type: field.Type, Name: field.Name, Index: index, Attributes: field.Attributes, Visibility: "public", Span: field.Span})
			}
			info.EnumCases = append(info.EnumCases, caseInfo)
		}
		return info, nil
	}
	return info, nil
}
