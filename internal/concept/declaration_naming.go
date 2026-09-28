package concept

import (
	"strings"
	"unicode"
)

func canonicalDeclarationStyle(kind DeclarationKind) string {
	switch kind {
	case TypeDeclaration, FunctionDeclaration, MethodDeclaration, ConceptDeclaration, InterfaceDeclaration, MachineDeclaration:
		return "PascalCase"
	case FieldDeclaration, LocalDeclaration, ParameterDeclaration:
		return "camelCase"
	default:
		return ""
	}
}

func declarationNameHasStyle(name, style string) bool {
	runes := []rune(name)
	if len(runes) == 0 {
		return false
	}
	switch style {
	case "PascalCase":
		if !unicode.IsUpper(runes[0]) {
			return false
		}
	case "camelCase":
		if !unicode.IsLower(runes[0]) {
			return false
		}
	case "snake_case":
		if !unicode.IsLower(runes[0]) {
			return false
		}
	default:
		return false
	}
	previousUnderscore := false
	for _, r := range runes {
		if r == '_' && style == "snake_case" {
			if previousUnderscore {
				return false
			}
			previousUnderscore = true
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
		if style == "snake_case" && unicode.IsUpper(r) {
			return false
		}
		previousUnderscore = false
	}
	return !previousUnderscore
}

func declarationNameSuggestion(name, style string) string {
	if style != "PascalCase" && style != "camelCase" {
		return ""
	}
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '_' })
	if len(parts) == 0 {
		return ""
	}
	var out strings.Builder
	for i, part := range parts {
		runes := []rune(part)
		for _, r := range runes {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				return ""
			}
		}
		if len(runes) == 0 {
			continue
		}
		// Segment acronyms as ordinary words, while retaining the interior
		// capitals of an existing mixed-case identifier such as SomeLocal.
		if len(parts) > 1 || strings.ToUpper(part) == part {
			for index := range runes {
				runes[index] = unicode.ToLower(runes[index])
			}
		}
		if style == "PascalCase" || i > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		} else {
			runes[0] = unicode.ToLower(runes[0])
		}
		out.WriteString(string(runes))
	}
	result := out.String()
	if result == name || !declarationNameHasStyle(result, style) {
		return ""
	}
	return result
}
