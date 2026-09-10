package concept

import "fmt"

func evt1ValidateExternalRegionEstablishment(env *semanticEnv, scope *evt1Scope, call *TemplateCallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if len(call.TypeArgs) != 1 || len(call.Args) != 5 {
		return Type{}, evt1Diagnostic("EXTERNAL_REGION_ARGUMENTS", "EstablishExternalRegion<Space> expects authority, address, extent, alignment, and contract identity", call.Span)
	}
	space := call.TypeArg
	if err := validateKnownType(env, space, space.Span, "", false); err != nil {
		return Type{}, err
	}
	if space.Kind != TypeStruct {
		return Type{}, evt1Diagnostic("ADDRESS_SPACE_TAG_INVALID", "external region requires a nominal address-space tag", space.Span)
	}
	authority, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	authorityIsRef := authority.isReference()
	if name, ok := call.Args[0].(*NameExpr); ok {
		if binding, found := scope.lookup(name.Name); found {
			authorityIsRef = authorityIsRef || binding.t.isReference()
		}
	}
	if !authorityIsRef {
		return Type{}, evt1Diagnostic("EXTERNAL_REGION_AUTHORITY_REQUIRED", "external region establishment requires an explicit ref to its live authority owner", call.Args[0].exprSpan())
	}
	addressType, err := validateExpr(env, scope, call.Args[1], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, err
	}
	if addressType.PointerTo == nil && addressType.Kind != TypeAddress {
		return Type{}, evt1Diagnostic("EXTERNAL_REGION_ADDRESS_INVALID", "external region address must be a foreign pointer or Address<Space>", call.Args[1].exprSpan())
	}
	for _, argument := range call.Args[2:4] {
		t, e := validateExpr(env, scope, argument, templateInfo, inComptimeFn)
		if e != nil {
			return Type{}, e
		}
		if !evt1ByteDisplacement(t) {
			return Type{}, evt1Diagnostic("EXTERNAL_REGION_GEOMETRY_UNIT", fmt.Sprintf("external region extent and alignment require usize<byte>, got %s", t.String()), argument.exprSpan())
		}
		if !evt1ExternalFieldOfAuthority(call.Args[0], argument) {
			return Type{}, evt1Diagnostic("FOREIGN_REGION_GEOMETRY_AUTHORITY_MISMATCH", "external region extent and alignment must come from the live authority wrapper", argument.exprSpan())
		}
	}
	identity, ok := call.Args[4].(*StringLiteral)
	if !ok || identity.Value == "" {
		return Type{}, evt1Diagnostic("FOREIGN_CONTRACT_IDENTITY_REQUIRED", "external region establishment requires a foreign contract string literal", call.Args[4].exprSpan())
	}
	contract, ok := env.foreignContracts[identity.Value]
	if !ok {
		return Type{}, evt1Diagnostic("FOREIGN_CONTRACT_UNKNOWN", fmt.Sprintf("unknown foreign contract %s", identity.Value), identity.Span)
	}
	if contract.AddressSpace.Name == "" {
		return Type{}, evt1Diagnostic("FOREIGN_STORAGE_CONTRACT_REQUIRED", fmt.Sprintf("foreign contract %s does not establish external storage", contract.Name), identity.Span)
	}
	declaringContext := contract.Module == env.moduleName || env.validatingModule == contract.Module
	if !declaringContext {
		return Type{}, evt1Diagnostic("FOREIGN_CONTRACT_CONTEXT_REQUIRED", fmt.Sprintf("EstablishExternalRegion is restricted to declaring contract module %s", contract.Module), identity.Span)
	}
	if contract.AddressSpace.Name != space.Name {
		return Type{}, evt1Diagnostic("FOREIGN_CONTRACT_ADDRESS_SPACE_MISMATCH", fmt.Sprintf("contract %s establishes %s, not %s", contract.Name, contract.AddressSpace.Name, space.Name), space.Span)
	}
	if space.Name == "SystemMemory" && !contract.HostAccessible {
		return Type{}, evt1Diagnostic("FOREIGN_HOST_ACCESS_UNKNOWN", fmt.Sprintf("contract %s does not declare HostAccessible(result)", contract.Name), identity.Span)
	}
	addressFacts := evt1SemanticFactsForExpr(env, scope, call.Args[1], addressType)
	fieldOfAuthority := evt1ExternalFieldOfAuthority(call.Args[0], call.Args[1])
	if (addressFacts == nil || addressFacts.Authority != contract.Name || addressFacts.RegionOrigin == "") && !fieldOfAuthority {
		return Type{}, evt1Diagnostic("FOREIGN_REGION_AUTHORITY_MISMATCH", fmt.Sprintf("address is not the declared result of %s", contract.Operation), call.Args[1].exprSpan())
	}
	return evt1InstantiateGenericType(env, Type{Name: "MemoryRegion", Kind: TypeApplied, TypeArgs: []Type{space.valueType()}, Span: call.Span})
}

func evt1ExternalFieldOfAuthority(authority, value Expr) bool {
	if ref, ok := authority.(*RefExpr); ok {
		authority = ref.Value
	}
	owner, ok := authority.(*NameExpr)
	if !ok {
		return false
	}
	field, ok := value.(*FieldExpr)
	if !ok {
		return false
	}
	receiver, ok := field.Receiver.(*NameExpr)
	return ok && receiver.Name == owner.Name
}
