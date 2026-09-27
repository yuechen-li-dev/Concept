package concept

import (
	"fmt"
	"strings"
)

// Raw fixed storage is an owning, partially initialized contiguous array.
// Its count is the sole live-prefix boundary; ordinary indexing and Span
// construction are deliberately unavailable on the raw representation.
func evt1ValidateRawStorageCall(env *semanticEnv, scope *evt1Scope, call *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, bool, error) {
	if strings.HasPrefix(call.Callee, "Sparse") {
		return evt1ValidateSparseStorageCall(env, scope, call, templateInfo, inComptimeFn)
	}
	switch call.Callee {
	case "RawCount", "RawAppend", "RawGet", "RawValues", "Emplace":
	default:
		return Type{}, false, nil
	}
	if len(call.Args) == 0 {
		return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", call.Callee+" requires a raw storage reference", call.Span)
	}
	backing, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, true, err
	}
	if call.Callee == "Emplace" {
		if len(call.Args) != 2 || !backing.isReference() || backing.Const {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", "Emplace requires a mutable store reference and one initializer", call.Span)
		}
		if err := evt1RawMutationBorrowCheck(env, scope, call.Args[0], templateInfo); err != nil {
			return Type{}, true, err
		}
		fields := env.fieldSets[backing.valueType().Name]
		raw, ok := fields["backing"]
		intrinsic := "dense_emplace"
		var element Type
		if ok && raw.StorageKind == StorageRaw && raw.ArrayElem != nil && len(fields) == 1 {
			element = evt1CanonicalType(env, *raw.ArrayElem)
		} else if slots, ok := fields["slots"]; ok && slots.StorageKind == StorageSparse && slots.ArrayElem != nil && fields["generation"].ArrayElem != nil && fields["free"].ArrayElem != nil {
			element = evt1CanonicalType(env, *slots.ArrayElem)
			intrinsic = "generational_emplace"
		} else {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", "Emplace requires a dense raw store or generational slot store", call.Args[0].exprSpan())
		}
		value, err := validateExprAgainstExpected(env, scope, call.Args[1], element, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, true, err
		}
		if !evt1TypesCompatible(env, element, value, "") || evt1IsImmovableValueType(env, element) && !evt1CanDirectInitialize(env, element, call.Args[1]) {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_INITIALIZER", "Emplace requires a compatible final-slot initializer", call.Args[1].exprSpan())
		}
		if !evt1TypeCopyable(env, element) && !evt1CanTransferInitialize(env, element, call.Args[1]) {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_MOVE_REQUIRED", "noncopyable Emplace initializer requires a move or direct construction", call.Args[1].exprSpan())
		}
		idName := "Id"
		if intrinsic == "generational_emplace" {
			idName = "GenerationalId"
		}
		id := Type{Name: idName, Kind: TypeApplied, TypeArgs: []Type{element}, Span: call.Span}
		id, err = evt1ResolveType(env, scope, id)
		if err != nil {
			return Type{}, true, err
		}
		call.Intrinsic = intrinsic
		return id, true, nil
	}
	if backing.StorageKind != StorageRaw || !backing.isReference() || backing.ArrayElem == nil {
		return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", call.Callee+" requires ref T<raw>[Capacity]", call.Args[0].exprSpan())
	}
	if evt1StorageHasRuntimeShape(backing) {
		return Type{}, true, evt1Diagnostic("RAW_STORAGE_SHAPE", "raw storage requires fixed capacity", call.Span)
	}
	element := evt1CanonicalType(env, *backing.ArrayElem)
	integer, _ := evt1BuiltinType("int", call.Span)
	switch call.Callee {
	case "RawCount":
		if len(call.Args) != 1 {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", "RawCount expects one argument", call.Span)
		}
		call.Intrinsic = "raw_count"
		return integer, true, nil
	case "RawAppend":
		if len(call.Args) != 2 || backing.Const {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", "RawAppend requires mutable raw storage and one initializer", call.Span)
		}
		if err := evt1RawMutationBorrowCheck(env, scope, call.Args[0], templateInfo); err != nil {
			return Type{}, true, err
		}
		value, err := validateExprAgainstExpected(env, scope, call.Args[1], element, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, true, err
		}
		if !evt1TypesCompatible(env, element, value, "") {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_INITIALIZER", fmt.Sprintf("RawAppend expected %s but got %s", element, value), call.Args[1].exprSpan())
		}
		if evt1IsImmovableValueType(env, element) && !evt1CanDirectInitialize(env, element, call.Args[1]) {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_IMMOVABLE", "immovable elements require direct aggregate construction", call.Args[1].exprSpan())
		}
		if !evt1TypeCopyable(env, element) && !evt1CanTransferInitialize(env, element, call.Args[1]) {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_MOVE_REQUIRED", "noncopyable initializer requires a move or direct construction", call.Args[1].exprSpan())
		}
		call.Intrinsic = "raw_append"
		return integer, true, nil
	case "RawGet":
		if len(call.Args) != 2 {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", "RawGet expects storage and an index", call.Span)
		}
		index, err := validateExpr(env, scope, call.Args[1], templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, true, err
		}
		if index.Name != "int" {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_INDEX", "RawGet index must be int", call.Args[1].exprSpan())
		}
		call.Intrinsic = "raw_get"
		out := element
		out.Ownership, out.Const = "ref", backing.Const
		return out, true, nil
	case "RawValues":
		if len(call.Args) != 1 {
			return Type{}, true, evt1Diagnostic("RAW_STORAGE_ARGUMENTS", "RawValues expects one argument", call.Span)
		}
		name := evt1SpanMutableName
		mutable := "mutable"
		if backing.Const {
			name, mutable = evt1SpanReadonlyName, "readonly"
		}
		_, alignment, err := evt1SpanElementGeometry(env, element)
		if err != nil && !evt1TypeContainsConceptParameter(element) {
			return Type{}, true, err
		}
		if alignment < 1 {
			alignment = 1
		}
		facts := evt1SpanFacts{
			ElementType: element, RegionID: "raw:" + evt1ExprIdentity(call.Args[0]),
			BaseOffsetExpression: "0", LengthExpression: "runtime",
			ByteExtentExpression: "runtime", Alignment: alignment,
			Mutable: !backing.Const, Provenance: evt1ExprProvenance(env, scope, call.Args[0]),
		}
		evt1ApplySpanFacts(call, facts, "raw_values", mutable)
		return evt1SpanType(name, element, call.Span), true, nil
	}
	return Type{}, false, nil
}

func evt1RawMutationBorrowCheck(env *semanticEnv, scope *evt1Scope, argument Expr, templateInfo *evt1TemplateInfo) error {
	if reference, ok := argument.(*RefExpr); ok {
		argument = reference.Value
	}
	place, err := validateAssignable(env, scope, argument, templateInfo)
	if err == nil && scope.hasObjectBorrow(place.path) {
		return evt1Diagnostic("RAW_STORAGE_BORROW_CONFLICT", "cannot append while a view of the backing storage is live", argument.exprSpan())
	}
	return nil
}

// Sparse storage owns independently live slots. Its live bits are metadata for
// destruction and checked access, never a claim that payloads form a prefix.
func evt1ValidateSparseStorageCall(env *semanticEnv, scope *evt1Scope, call *CallExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, bool, error) {
	switch call.Callee {
	case "SparseInitialize", "SparseGet", "SparseOccupied", "SparseRemove":
	default:
		return Type{}, false, nil
	}
	if len(call.Args) < 2 {
		return Type{}, true, evt1Diagnostic("SPARSE_STORAGE_ARGUMENTS", call.Callee+" requires storage and index", call.Span)
	}
	backing, err := validateExpr(env, scope, call.Args[0], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, true, err
	}
	if backing.StorageKind != StorageSparse || !backing.isReference() || backing.ArrayElem == nil {
		return Type{}, true, evt1Diagnostic("SPARSE_STORAGE_ARGUMENTS", "expected ref T<sparse>[Capacity]", call.Args[0].exprSpan())
	}
	index, err := validateExpr(env, scope, call.Args[1], templateInfo, inComptimeFn)
	if err != nil {
		return Type{}, true, err
	}
	if index.Name != "int" {
		return Type{}, true, evt1Diagnostic("SPARSE_STORAGE_INDEX", "sparse slot index must be int", call.Args[1].exprSpan())
	}
	element := evt1CanonicalType(env, *backing.ArrayElem)
	switch call.Callee {
	case "SparseGet":
		if len(call.Args) != 2 {
			break
		}
		call.Intrinsic = "sparse_get"
		result := element
		result.Ownership, result.Const = "ref", backing.Const
		return result, true, nil
	case "SparseOccupied":
		if len(call.Args) != 2 {
			break
		}
		call.Intrinsic = "sparse_occupied"
		result, _ := evt1BuiltinType("bool", call.Span)
		return result, true, nil
	case "SparseRemove":
		if len(call.Args) != 2 || backing.Const {
			break
		}
		if err := evt1RawMutationBorrowCheck(env, scope, call.Args[0], templateInfo); err != nil {
			return Type{}, true, err
		}
		call.Intrinsic = "sparse_remove"
		result, _ := evt1BuiltinType("void", call.Span)
		return result, true, nil
	case "SparseInitialize":
		if len(call.Args) != 3 || backing.Const {
			break
		}
		if err := evt1RawMutationBorrowCheck(env, scope, call.Args[0], templateInfo); err != nil {
			return Type{}, true, err
		}
		value, err := validateExprAgainstExpected(env, scope, call.Args[2], element, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, true, err
		}
		if !evt1TypesCompatible(env, element, value, "") || evt1IsImmovableValueType(env, element) && !evt1CanDirectInitialize(env, element, call.Args[2]) || !evt1TypeCopyable(env, element) && !evt1CanTransferInitialize(env, element, call.Args[2]) {
			return Type{}, true, evt1Diagnostic("SPARSE_STORAGE_INITIALIZER", "sparse slot requires a compatible movable or direct initializer", call.Args[2].exprSpan())
		}
		call.Intrinsic = "sparse_initialize"
		result, _ := evt1BuiltinType("void", call.Span)
		return result, true, nil
	}
	return Type{}, true, evt1Diagnostic("SPARSE_STORAGE_ARGUMENTS", "invalid sparse storage arguments", call.Span)
}

func (f *evt1FunctionLowerer) lowerSparseStorageCall(call *CallExpr, indent int) (string, string, Type) {
	storagePrelude, storage, storageType := f.lowerExpr(call.Args[0], indent)
	indexPrelude, indexExpr, _ := f.lowerExpr(call.Args[1], indent)
	storageName := f.nextTemp("sparse")
	indexName := f.nextTemp("sparse_index")
	var out strings.Builder
	out.WriteString(storagePrelude)
	out.WriteString(indexPrelude)
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(storageType), storageName, storage))
	out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", indexName, indexExpr))
	out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= sizeof(%s->data) / sizeof(%s->data[0])) { concept_panic(%q, %d, %d); }\n", indexName, indexName, storageName, storageName, "sparse slot index out of bounds", call.Span.Line, call.Span.Column))
	element := *storageType.ArrayElem
	if call.Intrinsic == "sparse_occupied" {
		result, _ := evt1BuiltinType("bool", call.Span)
		return out.String(), fmt.Sprintf("%s->live[%s]", storageName, indexName), result
	}
	if call.Intrinsic == "sparse_get" {
		out.WriteString(ind(indent) + fmt.Sprintf("if (!%s->live[%s]) { concept_panic(%q, %d, %d); }\n", storageName, indexName, "sparse slot is not live", call.Span.Line, call.Span.Column))
		result := element
		result.Ownership, result.Const = "ref", storageType.Const
		return out.String(), fmt.Sprintf("&%s->data[%s]", storageName, indexName), result
	}
	if call.Intrinsic == "sparse_remove" {
		out.WriteString(ind(indent) + fmt.Sprintf("if (!%s->live[%s]) { concept_panic(%q, %d, %d); }\n", storageName, indexName, "sparse slot is not live", call.Span.Line, call.Span.Column))
		out.WriteString(f.lowerDropValue(element, fmt.Sprintf("%s->data[%s]", storageName, indexName), indent))
		out.WriteString(ind(indent) + fmt.Sprintf("%s->live[%s] = false;\n", storageName, indexName))
		result, _ := evt1BuiltinType("void", call.Span)
		return out.String(), "((void)0)", result
	}
	out.WriteString(ind(indent) + fmt.Sprintf("if (%s->live[%s]) { concept_panic(%q, %d, %d); }\n", storageName, indexName, "sparse slot already live", call.Span.Line, call.Span.Column))
	if construct, ok := call.Args[2].(*StructConstructExpr); ok && construct.StructName == element.Name {
		out.WriteString(f.lowerInPlaceStructConstruct("&"+storageName+"->data["+indexName+"]", element, *construct, indent))
	} else {
		valuePrelude, value, _ := f.lowerExprExpected(call.Args[2], element, indent)
		out.WriteString(valuePrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("%s->data[%s] = %s;\n", storageName, indexName, value))
	}
	out.WriteString(ind(indent) + fmt.Sprintf("%s->live[%s] = true;\n", storageName, indexName))
	result, _ := evt1BuiltinType("void", call.Span)
	return out.String(), "((void)0)", result
}

func (f *evt1FunctionLowerer) lowerRawStorageCall(call *CallExpr, indent int) (string, string, Type) {
	prelude, backing, backingType := f.lowerExpr(call.Args[0], indent)
	backingName := f.nextTemp("raw")
	var out strings.Builder
	out.WriteString(prelude)
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(backingType), backingName, backing))
	element := *backingType.ArrayElem
	integer := Type{Name: "int", Kind: TypeBuiltin, Span: call.Span}
	switch call.Intrinsic {
	case "raw_count":
		return out.String(), backingName + "->count", integer
	case "raw_values":
		spanType := evt1SpanType(evt1SpanMutableName, element, call.Span)
		data := backingName + "->data"
		if backingType.Const {
			spanType.Name = evt1SpanReadonlyName
			data = "(const " + evt1CType(element) + "*)" + data
		}
		return out.String(), fmt.Sprintf("(%s){ .data = %s, .length = (size_t)%s->count }", evt1CType(spanType), data, backingName), spanType
	case "raw_get":
		indexPrelude, index, _ := f.lowerExpr(call.Args[1], indent)
		indexName := f.nextTemp("raw_index")
		out.WriteString(indexPrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", indexName, index))
		out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || %s >= %s->count) { concept_panic(%q, %d, %d); }\n", indexName, indexName, backingName, "raw storage index outside initialized prefix", call.Span.Line, call.Span.Column))
		result := element
		result.Ownership, result.Const = "ref", backingType.Const
		return out.String(), fmt.Sprintf("&%s->data[%s]", backingName, indexName), result
	case "raw_append":
		indexName := f.nextTemp("raw_index")
		out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s->count;\n", indexName, backingName))
		out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= sizeof(%s->data) / sizeof(%s->data[0])) { concept_panic(%q, %d, %d); }\n", indexName, indexName, backingName, backingName, "raw storage capacity exceeded", call.Span.Line, call.Span.Column))
		if construct, ok := call.Args[1].(*StructConstructExpr); ok && construct.StructName == element.Name {
			out.WriteString(f.lowerInPlaceStructConstruct("&"+backingName+"->data["+indexName+"]", element, *construct, indent))
		} else {
			valuePrelude, value, _ := f.lowerExprExpected(call.Args[1], element, indent)
			out.WriteString(valuePrelude)
			out.WriteString(ind(indent) + fmt.Sprintf("%s->data[%s] = %s;\n", backingName, indexName, value))
		}
		out.WriteString(ind(indent) + fmt.Sprintf("%s->count = %s + 1;\n", backingName, indexName))
		return out.String(), indexName, integer
	}
	return out.String(), "0", integer
}

func (f *evt1FunctionLowerer) lowerDenseEmplace(call *CallExpr, indent int) (string, string, Type) {
	storePrelude, store, storeType := f.lowerExpr(call.Args[0], indent)
	raw := f.l.env.fieldSets[storeType.valueType().Name]["backing"]
	element := *raw.ArrayElem
	storeName := f.nextTemp("store")
	index := f.nextTemp("emplace_index")
	var out strings.Builder
	out.WriteString(storePrelude)
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(storeType), storeName, store))
	backing := storeName + "->backing"
	out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s.count;\n", index, backing))
	out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= sizeof(%s.data) / sizeof(%s.data[0])) { concept_panic(%q, %d, %d); }\n", index, index, backing, backing, "DenseStore capacity exceeded", call.Span.Line, call.Span.Column))
	if construct, ok := call.Args[1].(*StructConstructExpr); ok && construct.StructName == element.Name {
		out.WriteString(f.lowerInPlaceStructConstruct("&"+backing+".data["+index+"]", element, *construct, indent))
	} else {
		valuePrelude, value, _ := f.lowerExprExpected(call.Args[1], element, indent)
		out.WriteString(valuePrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("%s.data[%s] = %s;\n", backing, index, value))
	}
	out.WriteString(ind(indent) + fmt.Sprintf("%s.count = %s + 1;\n", backing, index))
	id := Type{Name: "Id", Kind: TypeApplied, TypeArgs: []Type{element}, Span: call.Span}
	id, _ = evt1ResolveType(f.l.env, f.typeScope(), id)
	return out.String(), fmt.Sprintf("(%s){ .index = %s }", evt1CType(id), index), id
}

func (f *evt1FunctionLowerer) lowerGenerationalEmplace(call *CallExpr, indent int) (string, string, Type) {
	storePrelude, store, storeType := f.lowerExpr(call.Args[0], indent)
	slots := f.l.env.fieldSets[storeType.valueType().Name]["slots"]
	element := *slots.ArrayElem
	storeName := f.nextTemp("store")
	index := f.nextTemp("emplace_index")
	reused := f.nextTemp("emplace_reused")
	var out strings.Builder
	out.WriteString(storePrelude)
	out.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(storeType), storeName, store))
	out.WriteString(ind(indent) + fmt.Sprintf("bool %s = %s->freeCount > 0;\n", reused, storeName))
	out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s ? %s->free.data[%s->freeCount - 1] : %s->next;\n", index, reused, storeName, storeName, storeName))
	out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= sizeof(%s->slots.data) / sizeof(%s->slots.data[0])) { concept_panic(%q, %d, %d); }\n", index, index, storeName, storeName, "GenerationalStore capacity exceeded", call.Span.Line, call.Span.Column))
	target := fmt.Sprintf("&%s->slots.data[%s]", storeName, index)
	if construct, ok := call.Args[1].(*StructConstructExpr); ok && construct.StructName == element.Name {
		out.WriteString(f.lowerInPlaceStructConstruct(target, element, *construct, indent))
	} else {
		valuePrelude, value, _ := f.lowerExprExpected(call.Args[1], element, indent)
		out.WriteString(valuePrelude)
		out.WriteString(ind(indent) + fmt.Sprintf("*%s = %s;\n", target, value))
	}
	out.WriteString(ind(indent) + fmt.Sprintf("%s->slots.live[%s] = true;\n", storeName, index))
	out.WriteString(ind(indent) + fmt.Sprintf("if (%s) { %s->freeCount -= 1; } else { %s->next += 1; }\n", reused, storeName, storeName))
	id := Type{Name: "GenerationalId", Kind: TypeApplied, TypeArgs: []Type{element}, Span: call.Span}
	id, _ = evt1ResolveType(f.l.env, f.typeScope(), id)
	return out.String(), fmt.Sprintf("(%s){ .index = %s, .generation = %s->generation.data[%s] }", evt1CType(id), index, storeName, index), id
}
