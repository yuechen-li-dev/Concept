package concept

import (
	"strings"
	"unicode"
)

func evt1ConceptParameters(decl ConceptDecl) []GenericParameter {
	if len(decl.Parameters) != 0 {
		return decl.Parameters
	}
	if decl.TypeParam == "" {
		return nil
	}
	return []GenericParameter{{Name: decl.TypeParam, Kind: "type", Span: decl.Span}}
}

func evt1RequirementArguments(requirement *PrerequisiteRequirement) []Type {
	if len(requirement.TypeArgs) != 0 {
		return requirement.TypeArgs
	}
	if requirement.TypeArg.Name == "" && requirement.TypeArg.Kind == "" {
		return nil
	}
	return []Type{requirement.TypeArg}
}

func evt1ConstraintArguments(constraint TemplateConstraint) []Type {
	if len(constraint.TypeArgs) != 0 {
		return constraint.TypeArgs
	}
	if constraint.TypeArg.Name == "" && constraint.TypeArg.Kind == "" {
		return nil
	}
	return []Type{constraint.TypeArg}
}

func evt1ConceptBindings(decl ConceptDecl, arguments []Type) (map[string]Type, bool) {
	parameters := evt1ConceptParameters(decl)
	if len(parameters) != len(arguments) {
		return nil, false
	}
	bindings := make(map[string]Type, len(parameters))
	for i, parameter := range parameters {
		bindings[parameter.Name] = arguments[i]
	}
	return bindings, true
}

func evt1SubstituteRequirementBindings(requirement OperationRequirement, bindings map[string]Type) OperationRequirement {
	out := requirement
	out.ReturnType = evt1SubstituteBindings(requirement.ReturnType, bindings)
	out.Params = make([]Param, len(requirement.Params))
	for i, parameter := range requirement.Params {
		out.Params[i] = parameter
		out.Params[i].Type = evt1SubstituteBindings(parameter.Type, bindings)
	}
	return out
}

func evt1SubstituteArguments(arguments []Type, bindings map[string]Type) []Type {
	out := make([]Type, len(arguments))
	for i, argument := range arguments {
		out[i] = evt1SubstituteBindings(argument, bindings)
	}
	return out
}

func evt1ConceptApplicationLabel(name string, arguments []Type) string {
	parts := make([]string, len(arguments))
	for i, argument := range arguments {
		parts[i] = argument.String()
	}
	return name + "<" + strings.Join(parts, ", ") + ">"
}

func evt1TemplateBindings(parameters []GenericParameter, arguments []Type) map[string]Type {
	bindings := make(map[string]Type, len(parameters))
	for i, parameter := range parameters {
		if i < len(arguments) {
			bindings[parameter.Name] = arguments[i]
		}
	}
	return bindings
}

func evt1TypeDependsOnAnyParameter(t Type, parameters []GenericParameter) bool {
	for _, parameter := range parameters {
		if evt1TypeDependsOnParam(t, parameter.Name) {
			return true
		}
	}
	return false
}

func evt1TemplateParameterSet(parameters []GenericParameter) string {
	var names []string
	for _, parameter := range parameters {
		if parameter.Kind == "type" {
			names = append(names, parameter.Name)
		}
	}
	return strings.Join(names, "|")
}

func evt1SubstituteRequirementPath(path []string, bindings map[string]Type) []string {
	out := make([]string, len(path))
	for i, entry := range path {
		var result strings.Builder
		for offset := 0; offset < len(entry); {
			r := rune(entry[offset])
			if unicode.IsLetter(r) || r == '_' {
				end := offset + 1
				for end < len(entry) {
					next := rune(entry[end])
					if !unicode.IsLetter(next) && !unicode.IsDigit(next) && next != '_' {
						break
					}
					end++
				}
				name := entry[offset:end]
				if replacement, ok := bindings[name]; ok {
					result.WriteString(replacement.String())
				} else {
					result.WriteString(name)
				}
				offset = end
				continue
			}
			result.WriteByte(entry[offset])
			offset++
		}
		out[i] = result.String()
	}
	return out
}
