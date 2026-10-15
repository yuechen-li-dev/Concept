package concept

// Explicit T{} is semantic zero construction of a complete plain aggregate,
// never a partially live object, a null borrow, or a user-defined constructor.
func evt1ExpandEmptyAggregate(env *semanticEnv, scope *evt1Scope, e *StructConstructExpr) error {
	t := Type{Name: e.StructName, Kind: TypeStruct, Span: e.Span}
	if e.StructType.Name != "" {
		var err error
		t, err = evt1ResolveType(env, scope, e.StructType)
		if err != nil {
			return err
		}
	}
	value, err := evt1ZeroInitializer(env, t, map[string]bool{}, e.Span)
	if err != nil {
		return err
	}
	if constructed, ok := value.(*StructConstructExpr); ok {
		e.Args = constructed.Args
	}
	return nil
}
func evt1ZeroInitializer(env *semanticEnv, t Type, visiting map[string]bool, span Span) (Expr, error) {
	reject := func() (Expr, error) {
		return nil, evt1Diagnostic("AGGREGATE_ZERO_UNAVAILABLE", "explicit empty construction is unavailable for "+t.String()+"; initialize every field explicitly", span)
	}
	t = evt1CanonicalType(env, t)
	if t.Ownership != "" || evt1DropFunction(env, t) != nil || evt1TypeHasDrop(env, t) || !evt1TypeCopyable(env, t) {
		return reject()
	}
	if t.ArrayElem != nil {
		if t.StorageKind != StorageArray || evt1StorageRank(t) != 1 || evt1StorageHasRuntimeShape(t) {
			return reject()
		}
		element, err := evt1ZeroInitializer(env, *t.ArrayElem, visiting, span)
		if err != nil {
			return nil, err
		}
		return &ArrayLiteralExpr{Elements: []Expr{&RepeatInitializer{Value: element, FillRemainder: true, Span: span}}, Span: span}, nil
	}
	if evt1IsFloating(t) {
		return &FloatLiteral{Value: 0, ResolvedType: t, Span: span}, nil
	}
	if evt1IntegralRepresentation(t) {
		return &IntLiteral{Magnitude: 0, ResolvedType: t, Span: span}, nil
	}
	if t.Kind == TypeBuiltin && t.Name == "bool" {
		return &BoolLiteral{Value: false, Span: span}, nil
	}
	decl, ok := evt1StructView(env, t)
	if !ok || decl.Class || decl.Ref || decl.Immovable || decl.Table || decl.BitsRepresentation != "" || visiting[t.String()] {
		return reject()
	}
	visiting[t.String()] = true
	defer delete(visiting, t.String())
	var fields []Expr
	for _, field := range decl.Fields {
		if field.Initializer != nil || len(field.Attributes) != 0 {
			return reject()
		}
		value, err := evt1ZeroInitializer(env, field.Type, visiting, span)
		if err != nil {
			return nil, err
		}
		fields = append(fields, value)
	}
	return &StructConstructExpr{StructName: t.Name, StructType: t, Args: fields, Span: span}, nil
}
