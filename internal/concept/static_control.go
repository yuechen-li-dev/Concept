package concept

import (
	"fmt"
	"reflect"
)

// Static control uses the ordinary evaluator and rewrites into ordinary blocks
// before runtime validation and MIR. Open generic controls retain their AST.
func evt1ExpandStaticControl(env *semanticEnv, scope *evt1Scope, stmt Statement, template *evt1TemplateInfo) (*Block, bool, error) {
	var operand Expr
	switch s := stmt.(type) {
	case *IfStmt:
		if !s.Comptime {
			return nil, false, nil
		}
		operand = s.Condition
	case *ForeachStmt:
		if !s.Comptime {
			return nil, false, nil
		}
		operand = s.Source
	default:
		return nil, false, nil
	}
	dependent, err := evt1StaticControlDependencies(scope, operand, template)
	if err != nil {
		return nil, true, err
	}
	operandType, err := validateExpr(env, scope, operand, template, true)
	if err != nil {
		return nil, true, err
	}
	if s, ok := stmt.(*IfStmt); ok && operandType.Name != "bool" {
		return nil, true, evt1Diagnostic("CV4186", "comptime if requires bool", s.Condition.exprSpan())
	}
	if dependent {
		return nil, true, nil
	}
	if scope.staticExpansion == nil {
		scope.staticExpansion = newEVT1ComptimeState(env)
		if err := scope.staticExpansion.push("static expansion " + scope.functionName); err != nil {
			return nil, true, err
		}
	}
	state, evalScope := scope.staticExpansion, evt1EvalScopeFromValidation(scope, env)
	switch s := stmt.(type) {
	case *IfStmt:
		condition, err := evt1EvalExpr(state, evalScope, operand)
		if err != nil {
			return nil, true, err
		}
		if condition.Kind != ValueBool {
			return nil, true, evt1Diagnostic("CV4186", "comptime if requires bool", operand.exprSpan())
		}
		if condition.BoolValue {
			return &s.Then, true, nil
		}
		if s.Else != nil {
			return s.Else, true, nil
		}
		return &Block{Span: s.Span}, true, nil
	case *ForeachStmt:
		itemType := s.ItemType
		var element Type
		if operandType.ArrayElem != nil && operandType.StorageKind != StorageNDArray {
			element = *operandType.ArrayElem
		} else if t, ok := evt1RangeElement(operandType); ok {
			element = t
		} else {
			return nil, true, evt1Diagnostic("FOREACH_ITERATOR_INVALID", "comptime for iterates a range or a fixed rank-1 array", s.Span)
		}
		if itemType.Name == "" {
			itemType = element
		} else {
			itemType, err = evt1ResolveType(env, scope, itemType)
			if err != nil {
				return nil, true, err
			}
		}
		if itemType.isReference() || itemType.isOwned() {
			return nil, true, evt1Diagnostic("FOREACH_ITERATOR_INVALID", "comptime for produces values, not references or ownership transfers", s.Span)
		}
		if !evt1CanonicalType(env, itemType.valueType()).Equal(evt1CanonicalType(env, element.valueType())) {
			return nil, true, evt1Diagnostic("FOREACH_ITEM_TYPE_MISMATCH", fmt.Sprintf("comptime item type %s does not match %s", itemType.String(), element.String()), s.Span)
		}
		items, err := evt1ComptimeIteratorValues(state, evalScope, s.Source, itemType, s.Span)
		if err != nil {
			return nil, true, err
		}
		out := &Block{Span: s.Span}
		for _, item := range items {
			if err := state.spend(s.Span, 1); err != nil {
				return nil, true, err
			}
			body, err := evt1SubstituteBlock(s.Body, "#static-control-clone", Type{})
			if err != nil {
				return nil, true, err
			}
			binding := &VarDecl{Comptime: true, Type: item.Type, Name: s.ItemName, Value: &ComptimeValueExpr{Value: item, Span: s.Span}, Span: s.Span}
			body.Statements = append([]Statement{binding}, body.Statements...)
			out.Statements = append(out.Statements, &body)
		}
		return out, true, nil
	}
	panic("unreachable static control")
}

func evt1BlockHasStaticControl(block Block) bool {
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *IfStmt:
			if s.Comptime || evt1BlockHasStaticControl(s.Then) || (s.Else != nil && evt1BlockHasStaticControl(*s.Else)) {
				return true
			}
		case *ForeachStmt:
			if s.Comptime || evt1BlockHasStaticControl(s.Body) {
				return true
			}
		case *WhileStmt:
			if evt1BlockHasStaticControl(s.Body) || (s.Else != nil && evt1BlockHasStaticControl(*s.Else)) {
				return true
			}
		case *Block:
			if evt1BlockHasStaticControl(*s) {
				return true
			}
		case *MatchStmt:
			for _, arm := range s.Arms {
				if evt1BlockHasStaticControl(arm.Block) {
					return true
				}
			}
		case *TryStmt:
			if evt1BlockHasStaticControl(s.Body) {
				return true
			}
			for _, arm := range s.Except {
				if evt1BlockHasStaticControl(arm.Body) {
					return true
				}
			}
		}
	}
	return false
}

// Inspect structural operands before deferring an open layout query. Otherwise
// a runtime name following an unresolved T could escape the evaluator's error.
func evt1StaticControlDependencies(scope *evt1Scope, expr Expr, template *evt1TemplateInfo) (bool, error) {
	parameters := map[string]bool{}
	if template != nil {
		for _, p := range template.Decl.Parameters {
			parameters[p.Name] = true
		}
	}
	dependent := false
	visited := map[string]bool{}
	var walk func(reflect.Value) error
	walk = func(v reflect.Value) error {
		if !v.IsValid() {
			return nil
		}
		if v.CanInterface() {
			switch e := v.Interface().(type) {
			case *NameExpr:
				if parameters[e.Name] {
					dependent = true
					return nil
				}
				if binding, ok := scope.lookup(e.Name); ok {
					if !binding.comptime {
						return evt1Diagnostic("CV4200", fmt.Sprintf("runtime name %s is unavailable in comptime evaluation", e.Name), e.Span)
					}
					if !binding.hasValue && binding.source != nil && !visited[e.Name] {
						visited[e.Name] = true
						return walk(reflect.ValueOf(binding.source))
					}
				}
				return nil
			case Type:
				if parameters[e.Name] {
					dependent = true
				}
			}
		}
		switch v.Kind() {
		case reflect.Interface, reflect.Pointer:
			if !v.IsNil() {
				return walk(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if err := walk(v.Field(i)); err != nil {
					return err
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				if err := walk(v.Index(i)); err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := walk(reflect.ValueOf(expr))
	return dependent, err
}
