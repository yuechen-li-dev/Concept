package concept

// This file is the in-package implementation seam for profile-specific
// effect and actuator semantics. It remains in the core package only because
// those validators share the extracted semantic environment's private types.
// The public Vulkan consumer boundary is internal/concept/profile/vulkan.

import "fmt"

const evt1ActuatorMaxDecls = 8

type evt1ActuatorInfo struct {
	Decl           ActuatorDecl
	Automata       *evt1AutomataInfo
	MechanismType  Type
	ErrorType      Type
	Identity       string
	ResultTypeName string
	FailureSlot    string
	FailureOrdinal int
}

func evt1ActuationResultTypeName(actuatorName string) string {
	return "ActuationResult_" + actuatorName
}

func evt1ActuatorFailureSlotName(actuatorName string) string {
	return evt1PayloadFieldName(actuatorName)
}

func evt1ActuatorLocalCName(actuatorName string) string {
	return evt1CName(actuatorName) + "_executor"
}

func evt1ActuatorLocalInitName(actuatorName string) string {
	return evt1CName(actuatorName) + "_executor_init"
}

func evt1ActuatorActuateName(actuatorName string) string {
	return evt1CName(actuatorName) + "_actuate"
}

func evt1BatchDiscardName(automataName string) string {
	return evt1CName(automataName) + "_effects_discard"
}

func evt1ResultTypeName(okType, errType Type) string {
	return "Result<" + okType.String() + "," + errType.String() + ">"
}

func evt1IsResultVoidErrorType(env *semanticEnv, t Type) (Type, bool) {
	if t.Name != "Result" || len(t.TypeArgs) != 2 {
		return Type{}, false
	}
	okType := evt1CanonicalType(env, t.TypeArgs[0])
	if okType.Name != "void" || okType.Kind != TypeBuiltin {
		return Type{}, false
	}
	return evt1CanonicalType(env, t.TypeArgs[1]), true
}

func evt1ValidateActuatorDecls(env *semanticEnv, module Module) error {
	if len(module.Actuators) > evt1ActuatorMaxDecls {
		return evt1Diagnostic("CV4313", fmt.Sprintf("actuator declaration count %d exceeds limit %d", len(module.Actuators), evt1ActuatorMaxDecls), module.Actuators[evt1ActuatorMaxDecls].Span)
	}
	seen := map[string]Span{}
	byAutomataCount := map[string]int{}
	for _, decl := range module.Actuators {
		if other, ok := seen[decl.Name]; ok {
			return evt1Diagnostic("CV4313", fmt.Sprintf("duplicate actuator declaration %s", decl.Name), other)
		}
		seen[decl.Name] = decl.Span
		byAutomataCount[decl.AutomataName]++
	}
	for _, decl := range module.Actuators {
		info, err := evt1ValidateActuatorDecl(env, decl)
		if err != nil {
			return err
		}
		info.FailureOrdinal = byAutomataCount[decl.AutomataName]
		env.actuators[decl.Name] = decl
		env.actuatorInfo[decl.Name] = info
		resultType := Type{Name: info.ResultTypeName, Kind: TypeStruct, Span: decl.Span}
		env.fieldSets[info.ResultTypeName] = map[string]Type{
			"outcome":        {Name: evt1ActuationOutcomeTypeName, Kind: TypeEnum, Span: decl.Span},
			"completedCount": {Name: "int", Kind: TypeBuiltin, Span: decl.Span},
			"failedIndex":    {Name: "int", Kind: TypeBuiltin, Span: decl.Span},
			"error":          info.ErrorType,
		}
		env.structs[info.ResultTypeName] = StructDecl{
			Name: info.ResultTypeName,
			Fields: []Field{
				{Name: "outcome", Type: Type{Name: evt1ActuationOutcomeTypeName, Kind: TypeEnum, Span: decl.Span}, Span: decl.Span},
				{Name: "completedCount", Type: Type{Name: "int", Kind: TypeBuiltin, Span: decl.Span}, Span: decl.Span},
				{Name: "failedIndex", Type: Type{Name: "int", Kind: TypeBuiltin, Span: decl.Span}, Span: decl.Span},
				{Name: "error", Type: info.ErrorType, Span: decl.Span},
			},
			Span: decl.Span,
		}
		_ = resultType
	}
	return nil
}

func evt1ValidateActuatorDecl(env *semanticEnv, decl ActuatorDecl) (*evt1ActuatorInfo, error) {
	automataInfo, ok := env.automataInfo[decl.AutomataName]
	if !ok {
		return nil, evt1Diagnostic("CV4314", fmt.Sprintf("unknown automata %s in actuator %s", decl.AutomataName, decl.Name), decl.Span)
	}
	if len(automataInfo.EffectSet) == 0 {
		return nil, evt1Diagnostic("CV4314", fmt.Sprintf("actuator %s requires an effectful automata family, but %s emits no effects", decl.Name, decl.AutomataName), decl.Span)
	}
	if err := validateKnownType(env, decl.MechanismType, decl.Span, "", false); err != nil {
		return nil, err
	}
	mechanismType, err := evt1ResolveType(env, nil, decl.MechanismType)
	if err != nil {
		return nil, err
	}
	if !mechanismType.isBorrow() || mechanismType.Const || mechanismType.Imported || mechanismType.Unsafe || mechanismType.PointerTo != nil || mechanismType.ArrayElem != nil || len(mechanismType.TypeArgs) > 0 {
		return nil, evt1Diagnostic("CV4315", fmt.Sprintf("actuator %s mechanism must use one exact mutable borrow, got %s", decl.Name, mechanismType.String()), decl.Span)
	}
	if !evt1RuntimeTypeSafe(env, mechanismType.borrowBase()) {
		return nil, evt1Diagnostic("CV4315", fmt.Sprintf("actuator %s mechanism must use a legal runtime type, got %s", decl.Name, mechanismType.String()), decl.Span)
	}
	if err := validateKnownType(env, decl.ErrorType, decl.Span, "", false); err != nil {
		return nil, err
	}
	errorType, err := evt1ResolveType(env, nil, decl.ErrorType)
	if err != nil {
		return nil, err
	}
	if !evt1ActuatorErrorTypeAllowed(env, errorType) {
		return nil, evt1Diagnostic("CV4316", fmt.Sprintf("actuator %s error type must be a fixed copyable runtime value, got %s", decl.Name, errorType.String()), decl.Span)
	}
	if len(decl.Mappings) != len(automataInfo.EffectSet) {
		return nil, evt1Diagnostic("CV4317", fmt.Sprintf("actuator %s must map every effect in %s exactly once", decl.Name, decl.AutomataName), decl.Span)
	}
	seen := map[string]bool{}
	mappingIdentity := ""
	for _, effectName := range automataInfo.EffectSet {
		var mapping *ActuatorMapping
		for i := range decl.Mappings {
			if decl.Mappings[i].EffectName == effectName {
				mapping = &decl.Mappings[i]
				break
			}
		}
		if mapping == nil {
			return nil, evt1Diagnostic("CV4317", fmt.Sprintf("actuator %s is missing a mapping for effect %s", decl.Name, effectName), decl.Span)
		}
		if seen[mapping.EffectName] {
			return nil, evt1Diagnostic("CV4317", fmt.Sprintf("actuator %s duplicates effect mapping %s", decl.Name, mapping.EffectName), mapping.Span)
		}
		seen[mapping.EffectName] = true
		effectDecl := env.effects[effectName]
		if len(mapping.Params) != len(effectDecl.Params) {
			return nil, evt1Diagnostic("CV4317", fmt.Sprintf("actuator mapping for %s expected %d payload parameters but got %d", effectName, len(effectDecl.Params), len(mapping.Params)), mapping.Span)
		}
		paramNames := map[string]bool{}
		argTypes := []Type{mechanismType}
		for i := range effectDecl.Params {
			if err := validateKnownType(env, mapping.Params[i].Type, mapping.Params[i].Span, "", false); err != nil {
				return nil, err
			}
			resolvedParam, err := evt1ResolveType(env, nil, mapping.Params[i].Type)
			if err != nil {
				return nil, err
			}
			if !resolvedParam.Equal(effectDecl.Params[i].Type) {
				return nil, evt1Diagnostic("CV4317", fmt.Sprintf("actuator mapping for %s parameter %d expected %s but got %s", effectName, i+1, effectDecl.Params[i].Type.String(), resolvedParam.String()), mapping.Params[i].Span)
			}
			if paramNames[mapping.Params[i].Name] {
				return nil, evt1Diagnostic("CV4317", fmt.Sprintf("duplicate actuator mapping parameter %s", mapping.Params[i].Name), mapping.Params[i].Span)
			}
			paramNames[mapping.Params[i].Name] = true
			argTypes = append(argTypes, resolvedParam)
		}
		if len(mapping.ImplementationArgs) != 1+len(effectDecl.Params) {
			return nil, evt1Diagnostic("CV4318", fmt.Sprintf("actuator mapping for %s must call %s with mechanism plus exact payloads", effectName, mapping.ImplementationName), mapping.Span)
		}
		firstName, ok := mapping.ImplementationArgs[0].(*NameExpr)
		if !ok || firstName.Name != decl.MechanismName {
			return nil, evt1Diagnostic("CV4318", fmt.Sprintf("actuator mapping for %s must pass mechanism %s as the first implementation argument", effectName, decl.MechanismName), mapping.Span)
		}
		for i, param := range mapping.Params {
			nameExpr, ok := mapping.ImplementationArgs[i+1].(*NameExpr)
			if !ok || nameExpr.Name != param.Name {
				return nil, evt1Diagnostic("CV4318", fmt.Sprintf("actuator mapping for %s must pass payload %s directly and in order", effectName, param.Name), mapping.ImplementationArgs[i+1].exprSpan())
			}
		}
		fn, err := evt1ResolveActuatorImplementation(env, mapping.ImplementationName, argTypes, errorType, mapping.Span)
		if err != nil {
			return nil, err
		}
		mappingIdentity += "|" + effectName + "|" + evt1FunctionSignature(fn)
	}
	identity := digest([]byte(decl.Name + "|" + decl.AutomataName + "|" + mechanismType.String() + "|" + errorType.String() + mappingIdentity))
	return &evt1ActuatorInfo{
		Decl:           decl,
		Automata:       automataInfo,
		MechanismType:  mechanismType,
		ErrorType:      errorType,
		Identity:       identity,
		ResultTypeName: evt1ActuationResultTypeName(decl.Name),
		FailureSlot:    evt1ActuatorFailureSlotName(decl.Name),
	}, nil
}

func evt1ResolveActuatorImplementation(env *semanticEnv, name string, argTypes []Type, errorType Type, span Span) (FunctionDecl, error) {
	candidates, ok := env.functions[name]
	if !ok || len(candidates) == 0 {
		return FunctionDecl{}, evt1Diagnostic("CV4318", fmt.Sprintf("unknown actuator implementation %s", name), span)
	}
	var matches []FunctionDecl
	for _, fn := range candidates {
		if len(fn.Params) != len(argTypes) {
			continue
		}
		ok := true
		for i := range fn.Params {
			if !evt1CanonicalType(env, fn.Params[i].Type).Equal(evt1CanonicalType(env, argTypes[i])) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		resultError, isResult := evt1IsResultVoidErrorType(env, fn.ReturnType)
		if !isResult || !evt1CanonicalType(env, resultError).Equal(evt1CanonicalType(env, errorType)) {
			continue
		}
		matches = append(matches, fn)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return FunctionDecl{}, evt1Diagnostic("CV4318", fmt.Sprintf("actuator implementation %s is ambiguous under exact-signature matching", name), span)
	}
	return FunctionDecl{}, evt1Diagnostic("CV4318", fmt.Sprintf("no exact actuator implementation matched %s", name), span)
}

func evt1ActuatorErrorTypeAllowed(env *semanticEnv, t Type) bool {
	if t.isBorrowLike() || t.ArrayElem != nil || len(t.TypeArgs) > 0 || t.Const || t.Imported || t.Unsafe {
		return false
	}
	if !evt1RuntimeTypeSafe(env, t) || !evt1TypeCopyable(env, t) {
		return false
	}
	if t.Name == "string" {
		return false
	}
	return true
}

func evt1ActuatorsForAutomata(module Module, automataName string) []string {
	var out []string
	for _, decl := range module.Actuators {
		if decl.AutomataName == automataName {
			out = append(out, decl.Name)
		}
	}
	return out
}

func evt1MustResolveActuatorFunction(env *semanticEnv, name string, effectDecl EffectDecl, info *evt1ActuatorInfo) FunctionDecl {
	argTypes := []Type{info.MechanismType}
	for _, param := range effectDecl.Params {
		argTypes = append(argTypes, param.Type)
	}
	fn, err := evt1ResolveActuatorImplementation(env, name, argTypes, info.ErrorType, info.Decl.Span)
	if err != nil {
		panic(err)
	}
	return fn
}
