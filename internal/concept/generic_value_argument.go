package concept

import (
	"fmt"
	"strconv"
)

// Static-control items and local comptime declarations have lexical authority,
// just like module constants. Close their value leaves before the scope-free
// instance cache is consulted. A runtime shadow must not fall through to a
// same-named module constant. Ordinary parameter range checks remain centralized
// in evt1ResolveGenericValueArgument.
func evt1CloseScopedTemplateValueArguments(env *semanticEnv, scope *evt1Scope, call *TemplateCallExpr) error {
	decl, ok := env.templates[call.Callee]
	if !ok || len(decl.Parameters) != len(evt1TemplateCallArgs(call)) {
		return nil // ordinary instantiation owns unknown-template/arity errors
	}
	args := append([]Type{}, evt1TemplateCallArgs(call)...)
	for i, parameter := range decl.Parameters {
		arg := args[i]
		if parameter.Kind != "value" || !isIdentifier(arg.Name) || arg.PointerTo != nil || arg.ArrayElem != nil || len(arg.TypeArgs) != 0 {
			continue
		}
		binding, found := scope.lookup(arg.Name)
		if !found {
			continue
		}
		if binding.templateValue {
			continue // the owning generic closure supplies this symbolic leaf
		}
		if !binding.comptime {
			return evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", arg.Name+" is not a bounded compile-time integer", arg.Span)
		}
		constant, evalErr := evt1EvalExpr(newEVT1ComptimeState(env), evt1EvalScopeFromValidation(scope, env), &NameExpr{Name: arg.Name, Span: arg.Span})
		if evalErr != nil || constant.Kind != ValueInt || !evt1IntegralRepresentation(constant.Type) || constant.WideUint {
			return evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", arg.Name+" is not a bounded compile-time integer", arg.Span)
		}
		value := constant.IntValue
		if value < 0 {
			return evt1Diagnostic("GENERIC_NON_TYPE_ARGUMENT_INVALID", "non-type template argument must be non-negative", arg.Span)
		}
		resolved, err := evt1ResolveGenericValueArgument(env, Type{Name: strconv.Itoa(value), Kind: TypeTemplateValue, Span: arg.Span}, parameter.ValueType)
		if err != nil {
			return err
		}
		args[i] = resolved
	}
	call.TypeArgs, call.TypeArg = args, args[0]
	return nil
}

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
		if _, err := strconv.Atoi(argument.Name); err != nil && isIdentifier(argument.Name) {
			// The parser marks lexically bound value parameters explicitly.
			// Preserve this leaf until the owning template supplies a value.
			return argument, nil
		}
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
