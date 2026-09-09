package concept

import (
	"fmt"
	"strings"
)

var evt1TestKinds = map[string]bool{"fact": true, "theory": true, "benchmark": true, "prophecy": true}

func evt1ValidateTestMetadata(module Module) error {
	seenIDs := map[string]bool{}
	for _, fn := range module.Functions {
		if len(fn.Attributes) == 0 {
			continue
		}
		primary := ""
		foretold := false
		artifacts := 0
		for _, attribute := range fn.Attributes {
			switch {
			case evt1TestKinds[attribute.Name]:
				if len(attribute.Args) != 0 {
					return evt1Diagnostic("TEST_ATTRIBUTE_ARGUMENT_INVALID", fmt.Sprintf("[[%s]] does not accept arguments in R6a", attribute.Name), attribute.Span)
				}
				if primary != "" {
					return evt1Diagnostic("TEST_KIND_CONFLICT", fmt.Sprintf("test function %s cannot combine [[%s]] and [[%s]]", fn.Name, primary, attribute.Name), attribute.Span)
				}
				primary = attribute.Name
			case attribute.Name == "artifact":
				if len(attribute.Args) != 1 {
					return evt1Diagnostic("TEST_ARTIFACT_INVALID", "[[artifact]] requires exactly one string path", attribute.Span)
				}
				path, ok := attribute.Args[0].(*StringLiteral)
				if !ok || strings.TrimSpace(path.Value) == "" {
					return evt1Diagnostic("TEST_ARTIFACT_INVALID", "[[artifact]] requires a non-empty string literal path", attribute.Span)
				}
				artifacts++
			case attribute.Name == "foretold":
				if len(attribute.Args) != 0 {
					return evt1Diagnostic("TEST_ATTRIBUTE_ARGUMENT_INVALID", "[[foretold]] does not accept arguments", attribute.Span)
				}
				foretold = true
			default:
				return evt1Diagnostic("TEST_ATTRIBUTE_UNKNOWN", fmt.Sprintf("unknown attribute [[%s]]", attribute.Name), attribute.Span)
			}
		}
		if primary == "" {
			return evt1Diagnostic("TEST_KIND_REQUIRED", fmt.Sprintf("function %s has test metadata but no test kind", fn.Name), fn.Span)
		}
		if foretold && primary != "prophecy" {
			return evt1Diagnostic("TEST_FORETOLD_REQUIRES_PROPHECY", "[[foretold]] requires [[prophecy]]", fn.Span)
		}
		if fn.ReturnType.Name != "void" {
			return evt1Diagnostic("TEST_"+strings.ToUpper(primary)+"_REQUIRES_VOID_RETURN", fmt.Sprintf("[[%s]] function %s must return void", primary, fn.Name), fn.Span)
		}
		if primary != "theory" && len(fn.Params) != 0 {
			return evt1Diagnostic("TEST_"+strings.ToUpper(primary)+"_REQUIRES_ZERO_PARAMETERS", fmt.Sprintf("[[%s]] function %s must have zero parameters", primary, fn.Name), fn.Span)
		}
		if primary == "theory" {
			if len(fn.Params) == 0 {
				return evt1Diagnostic("TEST_THEORY_REQUIRES_PARAMETERS", "[[theory]] requires at least one parameter", fn.Span)
			}
			if artifacts == 0 {
				return evt1Diagnostic("TEST_THEORY_REQUIRES_ARTIFACT", "[[theory]] requires a JSON [[artifact]]", fn.Span)
			}
			for _, param := range fn.Params {
				if param.Type.Kind != TypeBuiltin || !(evt1TestNumericType(param.Type) || param.Type.Name == "bool" || param.Type.Name == "string") {
					return evt1Diagnostic("TEST_THEORY_PARAMETER_UNSUPPORTED", fmt.Sprintf("theory parameter %s has unsupported type %s", param.Name, param.Type.String()), param.Span)
				}
			}
		}
		if seenIDs[fn.Name] {
			return evt1Diagnostic("TEST_DUPLICATE_IDENTITY", fmt.Sprintf("duplicate test identity %s", fn.Name), fn.Span)
		}
		seenIDs[fn.Name] = true
	}
	return nil
}
