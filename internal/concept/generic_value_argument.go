package concept

import (
	"fmt"
	"strconv"
)

// A value argument has one identity after closure, regardless of whether its
// source spells a literal or a named comptime declaration. The parameter type
// supplies the context; a declaration's own type is still checked by comptime
// evaluation before its value can be used here.
func evt1ResolveGenericValueArgument(env *semanticEnv, argument Type, parameterType Type) (Type, error) {
	if argument.PointerTo != nil || argument.ArrayElem != nil || len(argument.TypeArgs) != 0 {
		return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", "non-type template parameter requires a compile-time integer", argument.Span)
	}
	var value int
	if argument.Kind == TypeTemplateValue {
		parsed, err := strconv.ParseUint(argument.Name, 0, 64)
		if err != nil || parsed > uint64(^uint(0)>>1) {
			return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", fmt.Sprintf("invalid compile-time integer %s", argument.Name), argument.Span)
		}
		value = int(parsed)
	} else {
		decl, ok := env.comptimeDecls[argument.Name]
		if !ok {
			return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", fmt.Sprintf("%s is not a compile-time integer", argument.Name), argument.Span)
		}
		resolved, err := evt1EvaluateGlobalComptimeDecl(newEVT1ComptimeState(env), decl)
		if err != nil {
			return Type{}, err
		}
		if resolved.Kind != ValueInt || !evt1IntegralRepresentation(resolved.Type) {
			return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", fmt.Sprintf("%s is not a compile-time integer", argument.Name), argument.Span)
		}
		if resolved.WideUint {
			return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", fmt.Sprintf("%s exceeds bounded generic value range", argument.Name), argument.Span)
		}
		value = resolved.IntValue
	}
	if value < 0 {
		return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", "non-type template argument must be non-negative", argument.Span)
	}
	_, max, _, _, ok := evt1IntegerTypeRange(parameterType)
	if !ok || uint64(value) > max {
		return Type{}, evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", fmt.Sprintf("template value %d is out of range for %s", value, parameterType.String()), argument.Span)
	}
	return Type{Name: strconv.Itoa(value), Kind: TypeTemplateValue, Span: argument.Span}, nil
}
