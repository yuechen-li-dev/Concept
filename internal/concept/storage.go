package concept

import (
	"fmt"
	"sort"
	"strings"
)

func evt1StorageRank(t Type) int {
	if t.ArrayElem == nil {
		return 0
	}
	if len(t.Shape) > 0 {
		return len(t.Shape)
	}
	return 1
}

func evt1StorageHasRuntimeShape(t Type) bool {
	for _, dimension := range t.Shape {
		if dimension.Runtime {
			return true
		}
	}
	return false
}

func evt1StorageElementCount(t Type) int {
	if t.ArrayElem == nil || evt1StorageHasRuntimeShape(t) {
		return 0
	}
	if len(t.Shape) == 0 {
		return t.ArrayLength
	}
	total := 1
	for _, dimension := range t.Shape {
		total *= dimension.Extent
	}
	return total
}

func evt1StorageIndices(expr *IndexExpr) []Expr {
	if len(expr.Indices) > 0 {
		return expr.Indices
	}
	if expr.Index != nil {
		return []Expr{expr.Index}
	}
	return nil
}

func evt1StorageCName(t Type) string {
	shape := make([]string, 0, len(t.Shape))
	for _, dimension := range t.Shape {
		if dimension.Runtime {
			shape = append(shape, "runtime")
		} else {
			shape = append(shape, fmt.Sprintf("%d", dimension.Extent))
		}
	}
	kind := t.StorageKind
	if kind == "" {
		kind = StorageArray
	}
	return "concept_" + string(kind) + "_" + strings.Join(shape, "_x_") + "_" + evt1TypeIdentity(*t.ArrayElem)
}

func evt1StorageViewCName(t Type) string {
	mutability := "mut"
	if t.Const {
		mutability = "const"
	}
	return fmt.Sprintf("concept_ref_%s_%s_%d_%s", mutability, t.StorageKind, evt1StorageRank(t), evt1TypeIdentity(*t.ArrayElem))
}

func evt1StorageOwnership(t Type) string {
	if evt1StorageHasRuntimeShape(t) {
		return "external"
	}
	return "fixed_inline"
}

func evt1StoragePanicReason(t Type) string {
	if t.StorageKind == StorageNDArray {
		return "Concept ndarray index out of bounds"
	}
	return "Concept array index out of bounds"
}

func evt1FlattenArrayLiteral(expr Expr) []Expr {
	literal, ok := expr.(*ArrayLiteralExpr)
	if !ok {
		return []Expr{expr}
	}
	var flattened []Expr
	for _, element := range literal.Elements {
		flattened = append(flattened, evt1FlattenArrayLiteral(element)...)
	}
	return flattened
}

func evt1NormalizeModuleStorageTypes(module *Module, env *semanticEnv) error {
	resolve := func(t Type) (Type, error) { return evt1ResolveType(env, nil, t) }
	for i := range module.Structs {
		for j := range module.Structs[i].Fields {
			resolved, err := resolve(module.Structs[i].Fields[j].Type)
			if err != nil {
				return err
			}
			module.Structs[i].Fields[j].Type = resolved
		}
	}
	for i := range module.Enums {
		for j := range module.Enums[i].Variants {
			for k := range module.Enums[i].Variants[j].Payload {
				resolved, err := resolve(module.Enums[i].Variants[j].Payload[k].Type)
				if err != nil {
					return err
				}
				module.Enums[i].Variants[j].Payload[k].Type = resolved
			}
		}
	}
	normalizeFunction := func(fn *FunctionDecl) error {
		resolved, err := resolve(fn.ReturnType)
		if err != nil {
			return err
		}
		fn.ReturnType = resolved
		for i := range fn.Params {
			resolved, err := resolve(fn.Params[i].Type)
			if err != nil {
				return err
			}
			fn.Params[i].Type = resolved
		}
		return nil
	}
	for i := range module.Functions {
		if err := normalizeFunction(&module.Functions[i]); err != nil {
			return err
		}
	}
	for i := range module.ComptimeFns {
		if err := normalizeFunction(&module.ComptimeFns[i]); err != nil {
			return err
		}
	}
	env.functions = map[string][]FunctionDecl{}
	for _, fn := range module.Functions {
		env.functions[fn.Name] = append(env.functions[fn.Name], fn)
	}
	return nil
}

func evt1CollectStorageTypes(module Module, env *semanticEnv) []Type {
	types := map[string]Type{}
	var add func(Type)
	add = func(t Type) {
		resolved, err := evt1ResolveType(env, nil, t)
		if err == nil {
			t = resolved
		}
		if t.PointerTo != nil {
			add(*t.PointerTo)
		}
		for _, arg := range t.TypeArgs {
			add(arg)
		}
		if t.ArrayElem == nil || evt1StorageHasRuntimeShape(t) {
			return
		}
		add(*t.ArrayElem)
		types[evt1TypeIdentity(t)] = t.valueType()
	}
	for _, decl := range module.Structs {
		for _, field := range decl.Fields {
			add(field.Type)
		}
	}
	for _, decl := range module.Enums {
		for _, variant := range decl.Variants {
			for _, field := range variant.Payload {
				add(field.Type)
			}
		}
	}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, statement := range block.Statements {
			switch s := statement.(type) {
			case *VarDecl:
				add(s.Type)
				if s.InlineTensor != nil {
					add(s.InlineTensor.BackingType)
				}
			case *Block:
				visitBlock(*s)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitBlock(s.Body)
			case *ForeachStmt:
				add(s.SourceType)
				add(s.ItemType)
				visitBlock(s.Body)
			case *MatchStmt:
				for _, arm := range s.Arms {
					visitBlock(arm.Block)
				}
			case *TryStmt:
				visitBlock(s.Body)
				for _, arm := range s.Except {
					visitBlock(arm.Body)
				}
			}
		}
	}
	for _, fn := range module.Functions {
		add(fn.ReturnType)
		for _, param := range fn.Params {
			add(param.Type)
		}
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	for _, automata := range module.Automata {
		for _, field := range automata.StateFields {
			add(field.Type)
		}
		for _, machine := range automata.Machines {
			for _, field := range machine.Fields {
				add(field.Type)
			}
			for _, state := range machine.States {
				if state.Body != nil {
					visitBlock(*state.Body)
				}
			}
		}
	}
	keys := make([]string, 0, len(types))
	for key := range types {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		left, right := types[keys[i]], types[keys[j]]
		leftDepth, rightDepth := evt1ArrayDepth(left), evt1ArrayDepth(right)
		if leftDepth != rightDepth {
			return leftDepth < rightDepth
		}
		return keys[i] < keys[j]
	})
	out := make([]Type, 0, len(keys))
	for _, key := range keys {
		out = append(out, types[key])
	}
	return out
}

func evt1CollectStorageViewTypes(module Module, env *semanticEnv) []Type {
	types := map[string]Type{}
	var add func(Type)
	add = func(t Type) {
		resolved, err := evt1ResolveType(env, nil, t)
		if err == nil {
			t = resolved
		}
		if t.PointerTo != nil {
			add(*t.PointerTo)
		}
		for _, arg := range t.TypeArgs {
			add(arg)
		}
		if t.ArrayElem != nil {
			if t.isReference() {
				types[evt1StorageViewCName(t)] = t
			}
			add(*t.ArrayElem)
		}
	}
	var visitBlock func(Block)
	visitBlock = func(block Block) {
		for _, statement := range block.Statements {
			switch s := statement.(type) {
			case *VarDecl:
				add(s.Type)
			case *Block:
				visitBlock(*s)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
				visitBlock(s.Body)
			case *ForeachStmt:
				add(s.SourceType)
				add(s.ItemType)
				visitBlock(s.Body)
			case *MatchStmt:
				for _, arm := range s.Arms {
					visitBlock(arm.Block)
				}
			case *TryStmt:
				visitBlock(s.Body)
				for _, arm := range s.Except {
					visitBlock(arm.Body)
				}
			}
		}
	}
	for _, decl := range module.Structs {
		for _, field := range decl.Fields {
			add(field.Type)
		}
	}
	for _, decl := range module.Enums {
		for _, variant := range decl.Variants {
			for _, field := range variant.Payload {
				add(field.Type)
			}
		}
	}
	for _, fn := range module.Functions {
		add(fn.ReturnType)
		for _, param := range fn.Params {
			add(param.Type)
		}
		if fn.Body != nil {
			visitBlock(*fn.Body)
		}
	}
	for _, automata := range module.Automata {
		for _, field := range automata.StateFields {
			add(field.Type)
		}
		for _, machine := range automata.Machines {
			for _, field := range machine.Fields {
				add(field.Type)
			}
			for _, state := range machine.States {
				if state.Body != nil {
					visitBlock(*state.Body)
				}
			}
		}
	}
	for _, layout := range env.layouts {
		for _, region := range layout.Regions {
			if region.Type.ArrayElem == nil {
				continue
			}
			view := region.Type
			view.Ownership = "ref"
			types[evt1StorageViewCName(view)] = view
			view.Const = true
			types[evt1StorageViewCName(view)] = view
		}
	}
	keys := make([]string, 0, len(types))
	for key := range types {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Type, 0, len(keys))
	for _, key := range keys {
		out = append(out, types[key])
	}
	return out
}

func evt1ModuleUsesStorageBounds(module Module) bool {
	var usesExpr func(Expr) bool
	usesExpr = func(expr Expr) bool {
		switch e := expr.(type) {
		case *IndexExpr:
			return true
		case *CallExpr:
			if e.Callee == "Subspan" || e.Callee == evt1SpanMutableName || e.Callee == evt1SpanReadonlyName {
				return true
			}
			if e.Callee == "Shape" {
				return true
			}
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *TemplateCallExpr:
			if e.Callee == "bind" || e.Callee == "Convert" {
				return true
			}
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *ArrayLiteralExpr:
			for _, element := range e.Elements {
				if usesExpr(element) {
					return true
				}
			}
		case *FieldExpr:
			return usesExpr(e.Receiver)
		case *ParenExpr:
			return usesExpr(e.Value)
		case *UnaryExpr:
			return usesExpr(e.Value)
		case *BinaryExpr:
			return usesExpr(e.Left) || usesExpr(e.Right)
		case *RefExpr:
			return usesExpr(e.Value)
		case *BindExpr:
			return true
		case *MoveExpr:
			return usesExpr(e.Value)
		case *IfExpr:
			return usesExpr(e.Condition) || usesExpr(e.Then) || usesExpr(e.Else)
		case *MatchExpr:
			if usesExpr(e.Subject) {
				return true
			}
			for _, arm := range e.Arms {
				if usesExpr(arm.Value) {
					return true
				}
			}
		case *ConstructExpr:
			for _, arg := range e.Args {
				if usesExpr(arg) {
					return true
				}
			}
		case *FailureExpr:
			return usesExpr(e.Value)
		}
		return false
	}
	var visitBlock func(Block) bool
	visitBlock = func(block Block) bool {
		for _, statement := range block.Statements {
			switch s := statement.(type) {
			case *VarDecl:
				if s.Value != nil && usesExpr(s.Value) {
					return true
				}
			case *AssignStmt:
				if usesExpr(s.Target) || usesExpr(s.Value) {
					return true
				}
			case *ReturnStmt:
				if s.Value != nil && usesExpr(s.Value) {
					return true
				}
			case *ExprStmt:
				if usesExpr(s.Value) {
					return true
				}
			case *IfStmt:
				if usesExpr(s.Condition) || visitBlock(s.Then) || (s.Else != nil && visitBlock(*s.Else)) {
					return true
				}
			case *WhileStmt:
				if usesExpr(s.Condition) || visitBlock(s.Body) {
					return true
				}
			case *ForeachStmt:
				if usesExpr(s.Source) || visitBlock(s.Body) {
					return true
				}
			case *Block:
				if visitBlock(*s) {
					return true
				}
			}
		}
		return false
	}
	for _, fn := range module.Functions {
		if fn.Body != nil && visitBlock(*fn.Body) {
			return true
		}
	}
	return false
}

func (f *evt1FunctionLowerer) lowerStorageLiteral(literal ArrayLiteralExpr, expected Type, indent int) (string, string, Type) {
	var prelude strings.Builder
	values := make([]string, 0, evt1StorageElementCount(expected))
	for i, element := range evt1FlattenArrayLiteral(&literal) {
		elementPrelude, value, elementType := f.lowerExpr(element, indent)
		prelude.WriteString(elementPrelude)
		name := f.nextTemp(fmt.Sprintf("element_%d", i+1))
		prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(elementType), name, value))
		values = append(values, name)
	}
	expression := fmt.Sprintf("(%s){ .data = { %s } }", evt1CType(expected), strings.Join(values, ", "))
	return prelude.String(), expression, expected
}

func (f *evt1FunctionLowerer) lowerStorageIndex(index *IndexExpr, indent int, place bool) (string, string, Type) {
	var prelude strings.Builder
	var base string
	var baseType Type
	if place {
		basePrelude, basePlace, resolvedType, _ := f.lowerLValue(index.Base, indent)
		prelude.WriteString(basePrelude)
		base, baseType = basePlace, resolvedType
	} else {
		basePrelude, baseValue, resolvedType := f.lowerExpr(index.Base, indent)
		prelude.WriteString(basePrelude)
		base, baseType = baseValue, resolvedType
	}
	indices := evt1StorageIndices(index)
	operation := "array_index"
	if len(indices) > 1 {
		operation = "ndarray_index"
	}
	boundsStrategy := f.plannedStrategyAt(operation, index.Span, "PerAccessRuntime")
	indexNames := make([]string, 0, len(indices))
	for i, indexExpr := range indices {
		indexPrelude, indexValue, _ := f.lowerExpr(indexExpr, indent)
		prelude.WriteString(indexPrelude)
		name := f.nextTemp(fmt.Sprintf("index_%d", i+1))
		prelude.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", name, indexValue))
		extent := f.lowerStorageExtent(base, baseType, i)
		if boundsStrategy == "PerAccessRuntime" {
			prelude.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || (size_t)%s >= %s) { concept_panic(%q, %d, %d); }\n", name, name, extent, evt1StoragePanicReason(baseType), index.Span.Line, index.Span.Column))
		}
		indexNames = append(indexNames, name)
	}
	offset := indexNames[0]
	for i := 1; i < len(indexNames); i++ {
		offset = fmt.Sprintf("((%s) * %s + %s)", offset, f.lowerStorageExtent(base, baseType, i), indexNames[i])
	}
	return prelude.String(), fmt.Sprintf("(%s).data[%s]", base, offset), *baseType.ArrayElem
}

func (f *evt1FunctionLowerer) lowerStorageExtent(base string, storageType Type, dimension int) string {
	if storageType.isReference() && storageType.Shape[dimension].Runtime {
		return fmt.Sprintf("(%s).shape[%d]", base, dimension)
	}
	return fmt.Sprintf("%d", storageType.Shape[dimension].Extent)
}

func (f *evt1FunctionLowerer) lowerStorageQuery(call *CallExpr, indent int) (string, string, Type) {
	intType, _ := evt1BuiltinType("int", call.Span)
	prelude, base, storageType := f.lowerExpr(call.Args[0], indent)
	switch call.Callee {
	case "Len":
		return prelude, f.lowerStorageExtent(base, storageType, 0), intType
	case "Rank":
		return prelude, fmt.Sprintf("%d", evt1StorageRank(storageType)), intType
	case "Shape":
		dimensionPrelude, dimensionValue, _ := f.lowerExpr(call.Args[1], indent)
		if dimension, err := evt1EvalExpr(newEVT1ComptimeState(f.l.env), f.evalScope(), call.Args[1]); err == nil {
			return prelude + dimensionPrelude, f.lowerStorageExtent(base, storageType, dimension.IntValue), intType
		}
		var out strings.Builder
		out.WriteString(prelude)
		out.WriteString(dimensionPrelude)
		name := f.nextTemp("shape_dimension")
		out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", name, dimensionValue))
		out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || %s >= %d) { concept_panic(%q, %d, %d); }\n", name, name, evt1StorageRank(storageType), evt1StoragePanicReason(storageType), call.Span.Line, call.Span.Column))
		expression := f.lowerStorageExtent(base, storageType, len(storageType.Shape)-1)
		for i := len(storageType.Shape) - 2; i >= 0; i-- {
			expression = fmt.Sprintf("(%s == %d ? %s : %s)", name, i, f.lowerStorageExtent(base, storageType, i), expression)
		}
		return out.String(), expression, intType
	}
	return prelude, "0", intType
}

func (f *evt1FunctionLowerer) lowerStorageView(source Expr, target Type, checked bool, span Span, indent int) (string, string, Type) {
	sourcePrelude, sourceValue, sourceType := f.lowerExpr(source, indent)
	var b strings.Builder
	b.WriteString(sourcePrelude)
	dimensions := make([]string, 0, len(target.Shape))
	for i, dimension := range target.Shape {
		if !dimension.Runtime {
			dimensions = append(dimensions, fmt.Sprintf("%d", dimension.Extent))
			continue
		}
		prelude, value, _ := f.lowerExpr(dimension.Expr, indent)
		b.WriteString(prelude)
		name := f.nextTemp(fmt.Sprintf("bind_extent_%d", i+1))
		b.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", name, value))
		dimensions = append(dimensions, name)
	}
	if checked {
		targetCount := f.lowerCheckedShapeProduct(&b, dimensions, "bind_target_count", span, indent, true)
		sourceCount := fmt.Sprintf("%d", evt1StorageElementCount(sourceType))
		if sourceType.isReference() {
			sourceDimensions := make([]string, 0, len(sourceType.Shape))
			for i := range sourceType.Shape {
				sourceDimensions = append(sourceDimensions, fmt.Sprintf("(%s).shape[%d]", sourceValue, i))
			}
			sourceCount = f.lowerCheckedShapeProduct(&b, sourceDimensions, "bind_source_count", span, indent, false)
		}
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s != %s) { concept_panic(%q, %d, %d); }\n", targetCount, sourceCount, "Concept bind shape does not match storage size", span.Line, span.Column))
	}
	data := fmt.Sprintf("(%s).data", sourceValue)
	expression := fmt.Sprintf("(%s){ .data = %s, .shape = { %s } }", evt1StorageViewCName(target), data, strings.Join(dimensions, ", "))
	return b.String(), expression, target
}

func (f *evt1FunctionLowerer) lowerCheckedShapeProduct(b *strings.Builder, dimensions []string, hint string, span Span, indent int, signed bool) string {
	name := f.nextTemp(hint)
	b.WriteString(ind(indent) + fmt.Sprintf("size_t %s = 1u;\n", name))
	for _, dimension := range dimensions {
		negative := ""
		if signed {
			negative = fmt.Sprintf("%s < 0 || ", dimension)
		}
		b.WriteString(ind(indent) + fmt.Sprintf("if (%s((size_t)%s != 0u && %s > SIZE_MAX / (size_t)%s)) { concept_panic(%q, %d, %d); }\n", negative, dimension, name, dimension, "Concept bind shape product overflow", span.Line, span.Column))
		b.WriteString(ind(indent) + fmt.Sprintf("%s *= (size_t)%s;\n", name, dimension))
	}
	return name
}
