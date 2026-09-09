package concept

import "fmt"

func validateForeachStmt(env *semanticEnv, scope *evt1Scope, stmt *ForeachStmt, returnType Type, templateInfo *evt1TemplateInfo, inComptimeFn bool) error {
	if inComptimeFn {
		return evt1Diagnostic("FOREACH_ITERATOR_INVALID", "foreach is a runtime iterator operation", stmt.Span)
	}
	sourceType, err := validateExpr(env, scope, stmt.Source, templateInfo, false)
	if err != nil {
		return err
	}
	sourceType = evt1CanonicalType(env, sourceType)
	stmt.SourceType = sourceType
	var element Type
	if sourceType.ArrayElem != nil {
		element = evt1CanonicalType(env, *sourceType.ArrayElem)
		stmt.SourceKind = "array"
		if sourceType.StorageKind == StorageNDArray {
			stmt.SourceKind = "ndarray"
		}
		stmt.IteratorType = Type{Name: "builtin_linear_iterator", Kind: TypeStruct, Span: stmt.Span}
	} else if evt1IsSpanType(sourceType) {
		element = evt1CanonicalType(env, evt1SpanElement(sourceType))
		stmt.SourceKind = "span"
		if !evt1SpanMutable(sourceType) {
			stmt.SourceKind = "readonly_span"
		}
		stmt.IteratorType = Type{Name: "builtin_span_iterator", Kind: TypeStruct, Span: stmt.Span}
	} else {
		iterator, customElement, findErr := evt1ResolveForeachProtocol(env, sourceType, stmt.Span)
		if findErr != nil {
			return findErr
		}
		stmt.SourceKind, stmt.IteratorType, element = "custom", iterator, customElement
	}
	stmt.ElementType = element
	itemValue := evt1CanonicalType(env, stmt.ItemType.valueType())
	if !itemValue.Equal(element.valueType()) {
		return evt1Diagnostic("FOREACH_ITEM_TYPE_MISMATCH", fmt.Sprintf("foreach item type %s does not match iterator element %s", stmt.ItemType.String(), element.String()), stmt.Span)
	}
	if stmt.ItemType.Ownership == "ref" {
		if (stmt.SourceKind == "readonly_span" || sourceType.Const) && !stmt.ItemType.Const {
			return evt1Diagnostic("FOREACH_MUTABLE_REF_FROM_READONLY", "mutable foreach reference cannot be produced from ReadOnlySpan", stmt.Span)
		}
		if stmt.SourceKind == "custom" {
			current := evt1ForeachProtocolFunction(env, "Current", stmt.IteratorType)
			if current.Name == "" || current.ReturnType.Ownership != "ref" || (!stmt.ItemType.Const && current.ReturnType.Const) {
				return evt1Diagnostic("FOREACH_ITERATOR_INVALID", "custom ref foreach requires Current(ref const Iterator) to return a compatible ref", stmt.Span)
			}
		}
	} else if stmt.ItemType.isOwned() || !evt1TypeCopyable(env, element) {
		return evt1Diagnostic("FOREACH_NONCOPYABLE_VALUE_ITEM", fmt.Sprintf("foreach value iteration would copy non-copyable %s; use ref iteration", element.String()), stmt.Span)
	}
	child := evt1CloneScope(scope)
	child.declare(stmt.ItemName, evt1ValueBinding{t: stmt.ItemType, mutable: !stmt.ItemType.Const, state: evt1StorageInitialized, provenance: evt1LifetimeProvenance{Kind: evt1ProvenanceLocal, Depth: child.depth, Scoped: stmt.ItemType.Scoped}})
	return validateBlock(env, child, returnType, stmt.Body, templateInfo, false)
}

func evt1ResolveForeachProtocol(env *semanticEnv, source Type, span Span) (Type, Type, error) {
	for _, get := range env.functions["GetIterator"] {
		if len(get.Params) != 1 || !get.Params[0].Type.valueType().Equal(source.valueType()) {
			continue
		}
		iterator := evt1CanonicalType(env, get.ReturnType)
		move := evt1ForeachProtocolFunction(env, "MoveNext", iterator)
		current := evt1ForeachProtocolFunction(env, "Current", iterator)
		if move.Name == "" || current.Name == "" || move.ReturnType.Name != "bool" || move.Params[0].Type.Ownership != "ref" || move.Params[0].Type.Const || current.Params[0].Type.Ownership != "ref" || !current.Params[0].Type.Const {
			continue
		}
		return iterator, evt1CanonicalType(env, current.ReturnType.valueType()), nil
	}
	return Type{}, Type{}, evt1Diagnostic("FOREACH_SOURCE_NOT_ITERABLE", fmt.Sprintf("type %s provides neither contiguous builtin iteration nor GetIterator/MoveNext/Current", source.String()), span)
}

func evt1ForeachProtocolFunction(env *semanticEnv, name string, iterator Type) FunctionDecl {
	for _, fn := range env.functions[name] {
		if len(fn.Params) == 1 && fn.Params[0].Type.valueType().Equal(iterator.valueType()) {
			return fn
		}
	}
	return FunctionDecl{}
}

func (f *evt1FunctionLowerer) lowerForeachStmt(stmt ForeachStmt, indent int) string {
	prelude, sourceExpr, sourceType := f.lowerExpr(stmt.Source, indent)
	sourceName := f.nextTemp("foreach_source")
	indexName := f.nextTemp("foreach_index")
	var b string
	b += ind(indent) + "{\n" + prelude
	f.pushScope()
	sourceAccess := sourceName
	_, namedSource := stmt.Source.(*NameExpr)
	storageByRef := sourceType.ArrayElem != nil && namedSource
	if storageByRef {
		qualifier := ""
		if sourceType.Const {
			qualifier = "const "
		}
		b += ind(indent+1) + fmt.Sprintf("%s%s* %s = &(%s); /* source evaluated once; backing storage is not copied */\n", qualifier, evt1CType(sourceType), sourceName, sourceExpr)
		sourceAccess = "(*" + sourceName + ")"
	} else {
		b += ind(indent+1) + fmt.Sprintf("%s %s = %s; /* source evaluated once */\n", evt1CType(sourceType), sourceName, sourceExpr)
		f.registerOwner(sourceName, sourceType)
	}
	if stmt.SourceKind != "custom" {
		length := fmt.Sprintf("%d", evt1StorageElementCount(sourceType))
		if evt1IsSpanType(sourceType) {
			length = sourceAccess + ".length"
		}
		b += ind(indent+1) + fmt.Sprintf("size_t %s = 0u; /* inline iterator state */\n", indexName)
		b += ind(indent+1) + fmt.Sprintf("while (%s < (size_t)(%s)) {\n", indexName, length)
		access := fmt.Sprintf("%s.data[%s]", sourceAccess, indexName)
		f.pushScope()
		itemName := f.bindName(stmt.ItemName, stmt.ItemType)
		itemExpr := access
		if stmt.ItemType.Ownership == "ref" {
			itemExpr = "&(" + access + ")"
		}
		b += ind(indent+2) + fmt.Sprintf("%s %s = %s;\n", evt1CType(stmt.ItemType), itemName, itemExpr)
		b += ind(indent+2) + fmt.Sprintf("%s = %s + 1u;\n", indexName, indexName)
		b += f.lowerBlock(stmt.Body, indent+2)
		b += f.lowerCurrentScopeDrops(indent + 2)
		f.popScope()
		b += ind(indent+1) + "}\n"
		b += f.lowerCurrentScopeDrops(indent + 1)
		f.popScope()
		b += ind(indent) + "}\n"
		return b
	}
	get := evt1ForeachProtocolFunction(f.l.env, "GetIterator", sourceType)
	move := evt1ForeachProtocolFunction(f.l.env, "MoveNext", stmt.IteratorType)
	current := evt1ForeachProtocolFunction(f.l.env, "Current", stmt.IteratorType)
	getArg := sourceName
	if get.Params[0].Type.isBorrowLike() {
		getArg = "&" + sourceName
	}
	iteratorName := f.nextTemp("foreach_iterator")
	b += ind(indent+1) + fmt.Sprintf("%s %s = %s(%s);\n", evt1CType(stmt.IteratorType), iteratorName, evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, get), getArg)
	f.registerOwner(iteratorName, stmt.IteratorType)
	moveArg := iteratorName
	if move.Params[0].Type.isBorrowLike() {
		moveArg = "&" + iteratorName
	}
	currentArg := iteratorName
	if current.Params[0].Type.isBorrowLike() {
		currentArg = "&" + iteratorName
	}
	b += ind(indent+1) + fmt.Sprintf("while (%s(%s)) {\n", evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, move), moveArg)
	f.pushScope()
	itemName := f.bindName(stmt.ItemName, stmt.ItemType)
	b += ind(indent+2) + fmt.Sprintf("%s %s = %s(%s);\n", evt1CType(stmt.ItemType), itemName, evt1FunctionSymbolForDecl(f.l.outputBase, f.l.env, current), currentArg)
	b += f.lowerBlock(stmt.Body, indent+2)
	b += f.lowerCurrentScopeDrops(indent + 2)
	f.popScope()
	b += ind(indent+1) + "}\n"
	b += f.lowerCurrentScopeDrops(indent + 1)
	f.popScope()
	b += ind(indent) + "}\n"
	return b
}
