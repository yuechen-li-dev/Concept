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
			case *Block:
				visitBlock(*s)
			case *IfStmt:
				visitBlock(s.Then)
				if s.Else != nil {
					visitBlock(*s.Else)
				}
			case *WhileStmt:
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

func evt1ModuleUsesStorageBounds(module Module) bool {
	var usesExpr func(Expr) bool
	usesExpr = func(expr Expr) bool {
		switch e := expr.(type) {
		case *IndexExpr:
			return true
		case *CallExpr:
			if e.Callee == "Shape" {
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
	indexNames := make([]string, 0, len(indices))
	for i, indexExpr := range indices {
		indexPrelude, indexValue, _ := f.lowerExpr(indexExpr, indent)
		prelude.WriteString(indexPrelude)
		name := f.nextTemp(fmt.Sprintf("index_%d", i+1))
		prelude.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", name, indexValue))
		extent := baseType.Shape[i].Extent
		prelude.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || %s >= %d) { concept_panic(%q, %d, %d); }\n", name, name, extent, evt1StoragePanicReason(baseType), index.Span.Line, index.Span.Column))
		indexNames = append(indexNames, name)
	}
	offset := indexNames[0]
	for i := 1; i < len(indexNames); i++ {
		offset = fmt.Sprintf("((%s) * %d + %s)", offset, baseType.Shape[i].Extent, indexNames[i])
	}
	return prelude.String(), fmt.Sprintf("(%s).data[%s]", base, offset), *baseType.ArrayElem
}

func (f *evt1FunctionLowerer) lowerStorageQuery(call *CallExpr, indent int) (string, string, Type) {
	intType, _ := evt1BuiltinType("int", call.Span)
	prelude, _, storageType := f.lowerExpr(call.Args[0], indent)
	switch call.Callee {
	case "Len":
		return prelude, fmt.Sprintf("%d", storageType.Shape[0].Extent), intType
	case "Rank":
		return prelude, fmt.Sprintf("%d", evt1StorageRank(storageType)), intType
	case "Shape":
		dimensionPrelude, dimensionValue, _ := f.lowerExpr(call.Args[1], indent)
		if dimension, err := evt1EvalExpr(newEVT1ComptimeState(f.l.env), f.evalScope(), call.Args[1]); err == nil {
			return prelude + dimensionPrelude, fmt.Sprintf("%d", storageType.Shape[dimension.IntValue].Extent), intType
		}
		var out strings.Builder
		out.WriteString(prelude)
		out.WriteString(dimensionPrelude)
		name := f.nextTemp("shape_dimension")
		out.WriteString(ind(indent) + fmt.Sprintf("int %s = %s;\n", name, dimensionValue))
		out.WriteString(ind(indent) + fmt.Sprintf("if (%s < 0 || %s >= %d) { concept_panic(%q, %d, %d); }\n", name, name, evt1StorageRank(storageType), evt1StoragePanicReason(storageType), call.Span.Line, call.Span.Column))
		expression := fmt.Sprintf("%d", storageType.Shape[len(storageType.Shape)-1].Extent)
		for i := len(storageType.Shape) - 2; i >= 0; i-- {
			expression = fmt.Sprintf("(%s == %d ? %d : %s)", name, i, storageType.Shape[i].Extent, expression)
		}
		return out.String(), expression, intType
	}
	return prelude, "0", intType
}
