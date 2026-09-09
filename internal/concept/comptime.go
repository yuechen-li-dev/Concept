package concept

import (
	"fmt"
	"strings"
)

const (
	evt1ComptimeMaxFuel            = 4096
	evt1ComptimeMaxLoopBound       = 256
	evt1ComptimeMaxCallDepth       = 32
	evt1ComptimeMaxArrayLength     = 64
	evt1StorageMaxFixedExtent      = 1048576
	evt1ComptimeMaxArrayNesting    = 8
	evt1ComptimeMaxArrayCells      = 512
	evt1ComptimeMaxLiteralElements = 512
)

type evt1EvalBinding struct {
	value    Value
	mutable  bool
	comptime bool
}

type evt1EvalScope struct {
	parent *evt1EvalScope
	values map[string]evt1EvalBinding
}

func newEVT1EvalScope(parent *evt1EvalScope) *evt1EvalScope {
	return &evt1EvalScope{parent: parent, values: map[string]evt1EvalBinding{}}
}

func (s *evt1EvalScope) declare(name string, binding evt1EvalBinding) {
	s.values[name] = binding
}

func (s *evt1EvalScope) lookup(name string) (evt1EvalBinding, bool) {
	for scope := s; scope != nil; scope = scope.parent {
		if binding, ok := scope.values[name]; ok {
			return binding, true
		}
	}
	return evt1EvalBinding{}, false
}

func (s *evt1EvalScope) assign(name string, value Value) bool {
	for scope := s; scope != nil; scope = scope.parent {
		if binding, ok := scope.values[name]; ok {
			binding.value = value
			scope.values[name] = binding
			return true
		}
	}
	return false
}

type evt1ComptimeState struct {
	env            *semanticEnv
	fuel           int
	callDepth      int
	stack          []string
	globalState    map[string]string
	staticMessages map[*StaticAssert]string
}

func newEVT1ComptimeState(env *semanticEnv) *evt1ComptimeState {
	return &evt1ComptimeState{
		env:            env,
		fuel:           evt1ComptimeMaxFuel,
		globalState:    map[string]string{},
		staticMessages: map[*StaticAssert]string{},
	}
}

func (s *evt1ComptimeState) push(frame string) error {
	s.stack = append(s.stack, frame)
	if len(s.stack) > evt1ComptimeMaxCallDepth {
		return evt1Diagnostic("CV4211", "comptime call depth exceeded at "+strings.Join(s.stack, " -> "), Span{})
	}
	return nil
}

func (s *evt1ComptimeState) pop() {
	if len(s.stack) > 0 {
		s.stack = s.stack[:len(s.stack)-1]
	}
}

func (s *evt1ComptimeState) spend(span Span, cost int) error {
	s.fuel -= cost
	if s.fuel < 0 {
		path := strings.Join(s.stack, " -> ")
		if path == "" {
			path = "<root>"
		}
		return evt1Diagnostic("CV4204", "comptime fuel exhausted along "+path, span)
	}
	return nil
}

func evt1IsComptimeType(env *semanticEnv, t Type) bool {
	if t.PointerTo != nil || t.isBorrowLike() || t.isOwned() || len(t.TypeArgs) > 0 {
		return false
	}
	if t.ArrayElem != nil {
		return evt1IsComptimeType(env, *t.ArrayElem)
	}
	if _, ok := evt1BuiltinType(t.Name, t.Span); ok {
		return t.Name == "int" || t.Name == "bool" || t.Name == "string"
	}
	if structDecl, ok := env.structs[t.Name]; ok {
		for _, field := range structDecl.Fields {
			if !evt1IsComptimeType(env, field.Type) {
				return false
			}
		}
		return true
	}
	if enumDecl, ok := env.enums[t.Name]; ok {
		for _, variant := range enumDecl.Variants {
			for _, field := range variant.Payload {
				if !evt1IsComptimeType(env, field.Type) {
					return false
				}
			}
		}
		return true
	}
	return false
}

func evt1SeedComptimeScope(env *semanticEnv) *evt1EvalScope {
	scope := newEVT1EvalScope(nil)
	for name, value := range env.comptimeValues {
		scope.declare(name, evt1EvalBinding{value: value, mutable: false, comptime: true})
	}
	return scope
}

func evt1EvaluateModuleComptime(env *semanticEnv, module Module) error {
	state := newEVT1ComptimeState(env)
	for _, decl := range module.ComptimeDecls {
		if _, err := evt1EvaluateGlobalComptimeDecl(state, decl); err != nil {
			return err
		}
	}
	for i := range module.StaticAsserts {
		if err := evt1EvaluateStaticAssert(state, evt1SeedComptimeScope(env), &module.StaticAsserts[i]); err != nil {
			return err
		}
	}
	return nil
}

func evt1EvaluateGlobalComptimeDecl(state *evt1ComptimeState, decl ComptimeDecl) (Value, error) {
	if value, ok := state.env.comptimeValues[decl.Name]; ok {
		return value, nil
	}
	switch state.globalState[decl.Name] {
	case "evaluating":
		return Value{}, evt1Diagnostic("CV4209", "comptime declaration cycle involving "+decl.Name, decl.Span)
	case "done":
		return state.env.comptimeValues[decl.Name], nil
	}
	state.globalState[decl.Name] = "evaluating"
	defer func() {
		if state.globalState[decl.Name] == "evaluating" {
			state.globalState[decl.Name] = "done"
		}
	}()
	if err := state.push("comptime " + decl.Name); err != nil {
		return Value{}, err
	}
	defer state.pop()
	scope := evt1SeedComptimeScope(state.env)
	resolvedType, err := evt1ResolveType(state.env, nil, decl.Type)
	if err != nil {
		return Value{}, err
	}
	value, err := evt1EvalExprTyped(state, scope, decl.Value, &resolvedType)
	if err != nil {
		return Value{}, err
	}
	if !evt1CanonicalType(state.env, resolvedType).Equal(evt1CanonicalType(state.env, value.Type)) {
		return Value{}, evt1Diagnostic("CV4202", fmt.Sprintf("comptime declaration %s expected %s but got %s", decl.Name, resolvedType.String(), value.Type.String()), decl.Span)
	}
	state.env.comptimeValues[decl.Name] = value
	return value, nil
}

func evt1EvaluateStaticAssert(state *evt1ComptimeState, scope *evt1EvalScope, assertion *StaticAssert) error {
	if err := state.push("static_assert"); err != nil {
		return err
	}
	defer state.pop()
	condition, err := evt1EvalExpr(state, scope, assertion.Condition)
	if err != nil {
		return err
	}
	if condition.Kind != ValueBool {
		return evt1Diagnostic("CV4207", "static_assert condition must evaluate to bool", assertion.Condition.exprSpan())
	}
	message := ""
	if assertion.Message != nil {
		value, err := evt1EvalExpr(state, scope, assertion.Message)
		if err != nil {
			return err
		}
		if value.Kind != ValueString {
			return evt1Diagnostic("CV4208", "static_assert message must evaluate to string", assertion.Message.exprSpan())
		}
		message = value.StringValue
	}
	if !condition.BoolValue {
		if message != "" {
			return evt1Diagnostic("CV4207", "static_assert failed: "+message, assertion.Span)
		}
		return evt1Diagnostic("CV4207", "static_assert failed", assertion.Span)
	}
	return nil
}

func evt1EvalExpr(state *evt1ComptimeState, scope *evt1EvalScope, expr Expr) (Value, error) {
	return evt1EvalExprTyped(state, scope, expr, nil)
}

func evt1EvalExprTyped(state *evt1ComptimeState, scope *evt1EvalScope, expr Expr, expected *Type) (Value, error) {
	if err := state.spend(expr.exprSpan(), 1); err != nil {
		return Value{}, err
	}
	switch e := expr.(type) {
	case *ParenExpr:
		return evt1EvalExprTyped(state, scope, e.Value, expected)
	case *IntLiteral:
		t, _ := evt1BuiltinType("int", e.Span)
		return Value{Kind: ValueInt, Type: t, IntValue: e.Value}, nil
	case *BoolLiteral:
		t, _ := evt1BuiltinType("bool", e.Span)
		return Value{Kind: ValueBool, Type: t, BoolValue: e.Value}, nil
	case *StringLiteral:
		t, _ := evt1BuiltinType("string", e.Span)
		return Value{Kind: ValueString, Type: t, StringValue: e.Value}, nil
	case *NameExpr:
		if binding, ok := scope.lookup(e.Name); ok && binding.comptime {
			return binding.value, nil
		}
		if decl, ok := state.env.comptimeDecls[e.Name]; ok {
			return evt1EvaluateGlobalComptimeDecl(state, decl)
		}
		return Value{}, evt1Diagnostic("CV4200", fmt.Sprintf("name %s is not available in comptime evaluation", e.Name), e.Span)
	case *DispatchExpr:
		return Value{}, evt1Diagnostic("CV4275", "dispatch is not available during comptime evaluation", e.Span)
	case *UnaryExpr:
		value, err := evt1EvalExpr(state, scope, e.Value)
		if err != nil {
			return Value{}, err
		}
		switch e.Op {
		case "-":
			if value.Kind != ValueInt {
				return Value{}, evt1Diagnostic("CV4201", "unary - requires int", e.Span)
			}
			return Value{Kind: ValueInt, Type: value.Type, IntValue: -value.IntValue}, nil
		case "not":
			if value.Kind != ValueBool {
				return Value{}, evt1Diagnostic("CV4201", "not requires bool", e.Span)
			}
			return Value{Kind: ValueBool, Type: value.Type, BoolValue: !value.BoolValue}, nil
		default:
			return Value{}, evt1Diagnostic("CV4201", "unsupported comptime unary operator "+e.Op, e.Span)
		}
	case *BinaryExpr:
		return evt1EvalBinaryExpr(state, scope, *e)
	case *FieldExpr:
		receiver, err := evt1EvalExpr(state, scope, e.Receiver)
		if err != nil {
			return Value{}, err
		}
		value, ok := receiver.Fields[e.Field]
		if !ok {
			return Value{}, evt1Diagnostic("CV4201", fmt.Sprintf("field %s is not available in comptime value", e.Field), e.Span)
		}
		return value, nil
	case *ArrayLiteralExpr:
		return evt1EvalArrayLiteral(state, scope, *e, expected)
	case *IndexExpr:
		base, err := evt1EvalExpr(state, scope, e.Base)
		if err != nil {
			return Value{}, err
		}
		if base.Kind != ValueArray {
			return Value{}, evt1Diagnostic("CV4231", "indexing requires a fixed compile-time array", e.Base.exprSpan())
		}
		indices := evt1StorageIndices(e)
		offset := 0
		for i, indexExpr := range indices {
			index, err := evt1EvalExpr(state, scope, indexExpr)
			if err != nil {
				return Value{}, err
			}
			if index.Kind != ValueInt {
				return Value{}, evt1Diagnostic("CV4232", "storage index must evaluate to int", indexExpr.exprSpan())
			}
			extent := base.Type.Shape[i].Extent
			if index.IntValue < 0 || index.IntValue >= extent {
				return Value{}, evt1Diagnostic("CV4233", fmt.Sprintf("storage index %d is out of range for extent %d", index.IntValue, extent), indexExpr.exprSpan())
			}
			offset = offset*extent + index.IntValue
		}
		return base.Elements[offset], nil
	case *StructConstructExpr:
		structDecl := state.env.structs[e.StructName]
		fields := map[string]Value{}
		for i, arg := range e.Args {
			value, err := evt1EvalExpr(state, scope, arg)
			if err != nil {
				return Value{}, err
			}
			fields[structDecl.Fields[i].Name] = value
		}
		return Value{
			Kind:       ValueStruct,
			Type:       Type{Name: e.StructName, Kind: TypeStruct, Span: e.Span},
			StructName: e.StructName,
			Fields:     fields,
		}, nil
	case *WithExpr:
		base, err := evt1EvalExpr(state, scope, e.Base)
		if err != nil {
			return Value{}, err
		}
		fields := make(map[string]Value, len(base.Fields))
		for name, value := range base.Fields {
			fields[name] = value
		}
		for _, update := range e.Updates {
			value, err := evt1EvalExpr(state, scope, update.Value)
			if err != nil {
				return Value{}, err
			}
			fields[update.Name] = value
		}
		base.Fields = fields
		return base, nil
	case *ConstructExpr:
		var payload []Value
		for _, arg := range e.Args {
			value, err := evt1EvalExpr(state, scope, arg)
			if err != nil {
				return Value{}, err
			}
			payload = append(payload, value)
		}
		return Value{
			Kind:     ValueEnum,
			Type:     Type{Name: e.EnumName, Kind: TypeEnum, Span: e.Span},
			EnumName: e.EnumName,
			Variant:  e.VariantName,
			Payload:  payload,
		}, nil
	case *IfExpr:
		condition, err := evt1EvalExpr(state, scope, e.Condition)
		if err != nil {
			return Value{}, err
		}
		if condition.Kind != ValueBool {
			return Value{}, evt1Diagnostic("CV4201", "comptime if condition must evaluate to bool", e.Condition.exprSpan())
		}
		if condition.BoolValue {
			return evt1EvalExpr(state, scope, e.Then)
		}
		return evt1EvalExpr(state, scope, e.Else)
	case *MatchExpr:
		return evt1EvalMatchExpr(state, scope, *e)
	case *CallExpr:
		if e.Callee == "Tensor" && len(e.Args) == 1 {
			return evt1EvalExpr(state, scope, e.Args[0])
		}
		if e.Callee == "Len" {
			if len(e.Args) != 1 {
				return Value{}, evt1Diagnostic("CV4234", fmt.Sprintf("Len expects exactly one argument, got %d", len(e.Args)), e.Span)
			}
			value, err := evt1EvalExpr(state, scope, e.Args[0])
			if err != nil {
				return Value{}, err
			}
			if value.Kind != ValueArray {
				return Value{}, evt1Diagnostic("CV4235", "Len requires a fixed compile-time array", e.Args[0].exprSpan())
			}
			t, _ := evt1BuiltinType("int", e.Span)
			return Value{Kind: ValueInt, Type: t, IntValue: len(value.Elements)}, nil
		}
		if e.Callee == "Rank" || e.Callee == "Shape" {
			value, err := evt1EvalExpr(state, scope, e.Args[0])
			if err != nil {
				return Value{}, err
			}
			t, _ := evt1BuiltinType("int", e.Span)
			if e.Callee == "Rank" {
				return Value{Kind: ValueInt, Type: t, IntValue: evt1StorageRank(value.Type)}, nil
			}
			dimension, err := evt1EvalExpr(state, scope, e.Args[1])
			if err != nil {
				return Value{}, err
			}
			if dimension.IntValue < 0 || dimension.IntValue >= len(value.Type.Shape) {
				return Value{}, evt1Diagnostic("CV4553", "shape dimension is out of range", e.Args[1].exprSpan())
			}
			return Value{Kind: ValueInt, Type: t, IntValue: value.Type.Shape[dimension.IntValue].Extent}, nil
		}
		return evt1EvalComptimeCall(state, scope, e.Callee, e.Args, e.Span)
	case *TemplateCallExpr:
		if evt1IsTypeLayoutQuery(e.Callee) {
			value, err := evt1LayoutQuery(state.env, e.Callee, e.TypeArg, e.Args)
			if err != nil {
				return Value{}, err
			}
			t, _ := evt1BuiltinType("int", e.Span)
			return Value{Kind: ValueInt, Type: t, IntValue: value}, nil
		}
		return Value{}, evt1Diagnostic("CV4201", "templates are not available during comptime evaluation", e.Span)
	default:
		return Value{}, evt1Diagnostic("CV4201", "unsupported comptime expression", expr.exprSpan())
	}
}

func evt1EvalBinaryExpr(state *evt1ComptimeState, scope *evt1EvalScope, expr BinaryExpr) (Value, error) {
	if expr.Op == "and" || expr.Op == "or" {
		left, err := evt1EvalExpr(state, scope, expr.Left)
		if err != nil {
			return Value{}, err
		}
		if left.Kind != ValueBool {
			return Value{}, evt1Diagnostic("CV4201", "logical operators require bool operands", expr.Left.exprSpan())
		}
		if expr.Op == "and" && !left.BoolValue {
			t, _ := evt1BuiltinType("bool", expr.Span)
			return Value{Kind: ValueBool, Type: t, BoolValue: false}, nil
		}
		if expr.Op == "or" && left.BoolValue {
			t, _ := evt1BuiltinType("bool", expr.Span)
			return Value{Kind: ValueBool, Type: t, BoolValue: true}, nil
		}
		right, err := evt1EvalExpr(state, scope, expr.Right)
		if err != nil {
			return Value{}, err
		}
		if right.Kind != ValueBool {
			return Value{}, evt1Diagnostic("CV4201", "logical operators require bool operands", expr.Right.exprSpan())
		}
		t, _ := evt1BuiltinType("bool", expr.Span)
		if expr.Op == "and" {
			return Value{Kind: ValueBool, Type: t, BoolValue: left.BoolValue && right.BoolValue}, nil
		}
		return Value{Kind: ValueBool, Type: t, BoolValue: left.BoolValue || right.BoolValue}, nil
	}
	left, err := evt1EvalExpr(state, scope, expr.Left)
	if err != nil {
		return Value{}, err
	}
	right, err := evt1EvalExpr(state, scope, expr.Right)
	if err != nil {
		return Value{}, err
	}
	if (left.Kind == ValueArray || right.Kind == ValueArray) && (expr.Op == "<" || expr.Op == ">" || expr.Op == "<=" || expr.Op == ">=") {
		return Value{}, evt1Diagnostic("CV4236", "array ordering comparisons are not supported", expr.Span)
	}
	switch expr.Op {
	case "+", "-", "*", "<", ">", "<=", ">=":
		if left.Kind != ValueInt || right.Kind != ValueInt {
			return Value{}, evt1Diagnostic("CV4201", "integer operator requires int operands", expr.Span)
		}
		switch expr.Op {
		case "+":
			return Value{Kind: ValueInt, Type: left.Type, IntValue: left.IntValue + right.IntValue}, nil
		case "-":
			return Value{Kind: ValueInt, Type: left.Type, IntValue: left.IntValue - right.IntValue}, nil
		case "*":
			return Value{Kind: ValueInt, Type: left.Type, IntValue: left.IntValue * right.IntValue}, nil
		default:
			t, _ := evt1BuiltinType("bool", expr.Span)
			return Value{Kind: ValueBool, Type: t, BoolValue: evt1CompareInts(left.IntValue, right.IntValue, expr.Op)}, nil
		}
	case "==", "!=":
		if left.Kind == ValueArray || right.Kind == ValueArray {
			if !evt1ValueEqual(left, right) && expr.Op == "==" {
				t, _ := evt1BuiltinType("bool", expr.Span)
				return Value{Kind: ValueBool, Type: t, BoolValue: false}, nil
			}
		}
		equal := evt1ValueEqual(left, right)
		t, _ := evt1BuiltinType("bool", expr.Span)
		if expr.Op == "==" {
			return Value{Kind: ValueBool, Type: t, BoolValue: equal}, nil
		}
		return Value{Kind: ValueBool, Type: t, BoolValue: !equal}, nil
	default:
		return Value{}, evt1Diagnostic("CV4201", "unsupported comptime operator "+expr.Op, expr.Span)
	}
}

func evt1CompareInts(left, right int, op string) bool {
	switch op {
	case "<":
		return left < right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case ">=":
		return left >= right
	default:
		return false
	}
}

func evt1ValueEqual(left, right Value) bool {
	if !evt1CanonicalType(nil, left.Type).Equal(evt1CanonicalType(nil, right.Type)) && left.Kind != right.Kind {
		return false
	}
	switch left.Kind {
	case ValueInt:
		return right.Kind == ValueInt && left.IntValue == right.IntValue
	case ValueBool:
		return right.Kind == ValueBool && left.BoolValue == right.BoolValue
	case ValueString:
		return right.Kind == ValueString && left.StringValue == right.StringValue
	case ValueStruct:
		if right.Kind != ValueStruct || left.StructName != right.StructName || len(left.Fields) != len(right.Fields) {
			return false
		}
		for name, value := range left.Fields {
			other, ok := right.Fields[name]
			if !ok || !evt1ValueEqual(value, other) {
				return false
			}
		}
		return true
	case ValueEnum:
		if right.Kind != ValueEnum || left.EnumName != right.EnumName || left.Variant != right.Variant || len(left.Payload) != len(right.Payload) {
			return false
		}
		for i := range left.Payload {
			if !evt1ValueEqual(left.Payload[i], right.Payload[i]) {
				return false
			}
		}
		return true
	case ValueArray:
		if right.Kind != ValueArray || len(left.Elements) != len(right.Elements) {
			return false
		}
		for i := range left.Elements {
			if !evt1ValueEqual(left.Elements[i], right.Elements[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func evt1EvalMatchExpr(state *evt1ComptimeState, scope *evt1EvalScope, expr MatchExpr) (Value, error) {
	subject, err := evt1EvalExpr(state, scope, expr.Subject)
	if err != nil {
		return Value{}, err
	}
	if subject.Kind != ValueEnum {
		return Value{}, evt1Diagnostic("CV4201", "comptime match subject must be an enum value", expr.Subject.exprSpan())
	}
	for _, arm := range expr.Arms {
		if arm.Pattern.EnumName != subject.EnumName || arm.Pattern.VariantName != subject.Variant {
			continue
		}
		armScope := newEVT1EvalScope(scope)
		for i, binding := range arm.Pattern.Bindings {
			armScope.declare(binding, evt1EvalBinding{value: subject.Payload[i], mutable: false, comptime: true})
		}
		return evt1EvalExpr(state, armScope, arm.Value)
	}
	return Value{}, evt1Diagnostic("CV4201", "comptime match found no selected arm", expr.Span)
}

func evt1EvalComptimeCall(state *evt1ComptimeState, scope *evt1EvalScope, name string, args []Expr, span Span) (Value, error) {
	fn, ok := state.env.comptimeFunctions[name]
	if !ok {
		return Value{}, evt1Diagnostic("CV4210", fmt.Sprintf("runtime function %s cannot be called during comptime evaluation", name), span)
	}
	if len(fn.Params) != len(args) {
		return Value{}, evt1Diagnostic("CV4106", fmt.Sprintf("wrong constructor or call payload count for %s: expected %d but got %d", name, len(fn.Params), len(args)), span)
	}
	if err := state.push("comptime fn " + name); err != nil {
		return Value{}, err
	}
	defer state.pop()
	callScope := evt1SeedComptimeScope(state.env)
	for i, arg := range args {
		paramType, err := evt1ResolveType(state.env, nil, fn.Params[i].Type)
		if err != nil {
			return Value{}, err
		}
		value, err := evt1EvalExprTyped(state, scope, arg, &paramType)
		if err != nil {
			return Value{}, err
		}
		callScope.declare(fn.Params[i].Name, evt1EvalBinding{value: value, mutable: true, comptime: true})
	}
	resolvedReturn, err := evt1ResolveType(state.env, nil, fn.ReturnType)
	if err != nil {
		return Value{}, err
	}
	result, err := evt1ExecComptimeBlock(state, callScope, *fn.Body, resolvedReturn)
	if err != nil {
		return Value{}, err
	}
	if result == nil {
		if fn.ReturnType.Name == "void" {
			return Value{Type: fn.ReturnType}, nil
		}
		return Value{}, evt1Diagnostic("CV4212", fmt.Sprintf("comptime function %s did not return a value", name), fn.Span)
	}
	return *result, nil
}

func evt1ExecComptimeBlock(state *evt1ComptimeState, scope *evt1EvalScope, block Block, returnType Type) (*Value, error) {
	local := newEVT1EvalScope(scope)
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *VarDecl:
			value, err := evt1EvalExprTyped(state, local, s.Value, &s.Type)
			if err != nil {
				return nil, err
			}
			local.declare(s.Name, evt1EvalBinding{value: value, mutable: !s.Comptime, comptime: true})
		case *InstanceDecl:
			return nil, evt1Diagnostic("CV4271", fmt.Sprintf("instance %s cannot be declared in comptime code", s.Name), s.Span)
		case *AssignStmt:
			if s.Tensor != nil {
				if err := evt1ExecComptimeTensorAssignment(state, local, s); err != nil {
					return nil, err
				}
				continue
			}
			nameExpr, ok := s.Target.(*NameExpr)
			if !ok {
				return nil, evt1Diagnostic("CV4213", "comptime assignment requires a named local", s.Target.exprSpan())
			}
			binding, ok := local.lookup(nameExpr.Name)
			if !ok || !binding.mutable {
				return nil, evt1Diagnostic("CV4213", fmt.Sprintf("comptime local %s is not mutable", nameExpr.Name), s.Span)
			}
			value, err := evt1EvalExpr(state, local, s.Value)
			if err != nil {
				return nil, err
			}
			local.assign(nameExpr.Name, value)
		case *ReturnStmt:
			if s.Value == nil {
				return nil, nil
			}
			value, err := evt1EvalExprTyped(state, local, s.Value, &returnType)
			if err != nil {
				return nil, err
			}
			return &value, nil
		case *ExprStmt:
			if _, err := evt1EvalExpr(state, local, s.Value); err != nil {
				return nil, err
			}
		case *StaticAssertStmt:
			if err := evt1EvaluateStaticAssert(state, local, &StaticAssert{Condition: s.Condition, Message: s.Message, Span: s.Span}); err != nil {
				return nil, err
			}
		case *WhileStmt:
			if s.Bound == nil {
				return nil, evt1Diagnostic("CV4205", "comptime while requires an explicit bounded(limit) clause", s.Span)
			}
			bound, err := evt1EvalExpr(state, local, s.Bound)
			if err != nil {
				return nil, err
			}
			if bound.Kind != ValueInt || bound.IntValue < 0 {
				return nil, evt1Diagnostic("CV4205", "bounded while requires a non-negative compile-time int bound", s.Bound.exprSpan())
			}
			if bound.IntValue > evt1ComptimeMaxLoopBound {
				return nil, evt1Diagnostic("CV4206", fmt.Sprintf("comptime loop bound %d exceeds limit %d", bound.IntValue, evt1ComptimeMaxLoopBound), s.Bound.exprSpan())
			}
			if bound.IntValue == 0 {
				continue
			}
			if err := state.push(fmt.Sprintf("while[%d]", bound.IntValue)); err != nil {
				return nil, err
			}
			for i := 0; i < bound.IntValue; i++ {
				condition, err := evt1EvalExpr(state, local, s.Condition)
				if err != nil {
					state.pop()
					return nil, err
				}
				if condition.Kind != ValueBool {
					state.pop()
					return nil, evt1Diagnostic("CV4201", "while condition must evaluate to bool", s.Condition.exprSpan())
				}
				if !condition.BoolValue {
					break
				}
				result, err := evt1ExecComptimeBlock(state, local, s.Body, returnType)
				if err != nil {
					state.pop()
					return nil, err
				}
				if result != nil {
					state.pop()
					return result, nil
				}
			}
			state.pop()
		case *MatchStmt:
			subject, err := evt1EvalExpr(state, local, s.Subject)
			if err != nil {
				return nil, err
			}
			if subject.Kind != ValueEnum {
				return nil, evt1Diagnostic("CV4201", "comptime match subject must be an enum value", s.Subject.exprSpan())
			}
			matched := false
			for _, arm := range s.Arms {
				if arm.Pattern.EnumName != subject.EnumName || arm.Pattern.VariantName != subject.Variant {
					continue
				}
				matched = true
				armScope := newEVT1EvalScope(local)
				for i, binding := range arm.Pattern.Bindings {
					armScope.declare(binding, evt1EvalBinding{value: subject.Payload[i], mutable: false, comptime: true})
				}
				result, err := evt1ExecComptimeBlock(state, armScope, arm.Block, returnType)
				if err != nil {
					return nil, err
				}
				if result != nil {
					return result, nil
				}
				break
			}
			if !matched {
				return nil, evt1Diagnostic("CV4201", "comptime match found no selected arm", s.Span)
			}
		case *Block:
			result, err := evt1ExecComptimeBlock(state, local, *s, returnType)
			if err != nil {
				return nil, err
			}
			if result != nil {
				return result, nil
			}
		default:
			return nil, evt1Diagnostic("CV4201", "unsupported comptime statement", stmt.statementSpan())
		}
	}
	return nil, nil
}

func evt1EvalArrayLiteral(state *evt1ComptimeState, scope *evt1EvalScope, expr ArrayLiteralExpr, expected *Type) (Value, error) {
	var arrayType Type
	if expected != nil && expected.ArrayElem != nil {
		arrayType = expected.valueType()
		if arrayType.StorageKind != StorageNDArray && len(expr.Elements) != arrayType.ArrayLength {
			return Value{}, evt1Diagnostic("CV4226", fmt.Sprintf("array literal expected %d elements but got %d", arrayType.ArrayLength, len(expr.Elements)), expr.Span)
		}
	} else if len(expr.Elements) == 0 {
		return Value{}, evt1Diagnostic("CV4225", "empty array literal requires an explicit fixed-array type", expr.Span)
	}
	if len(expr.Elements) > evt1ComptimeMaxLiteralElements {
		return Value{}, evt1Diagnostic("CV4224", fmt.Sprintf("array literal element count %d exceeds limit %d", len(expr.Elements), evt1ComptimeMaxLiteralElements), expr.Span)
	}
	literalElements := expr.Elements
	if arrayType.StorageKind == StorageNDArray {
		literalElements = evt1FlattenArrayLiteral(&expr)
	}
	elements := make([]Value, 0, len(literalElements))
	for i, element := range literalElements {
		var elemExpected *Type
		if arrayType.ArrayElem != nil {
			elemExpected = arrayType.ArrayElem
		}
		value, err := evt1EvalExprTyped(state, scope, element, elemExpected)
		if err != nil {
			return Value{}, err
		}
		if i == 0 && arrayType.ArrayElem == nil {
			elemType := value.Type.valueType()
			arrayType = Type{
				Name:        elemType.String() + "[]",
				Kind:        TypeArray,
				ArrayElem:   &elemType,
				ArrayLength: len(expr.Elements),
				StorageKind: StorageArray,
				Shape:       []StorageDimension{{Extent: len(expr.Elements), Expression: fmt.Sprintf("%d", len(expr.Elements))}},
				Contiguous:  true,
				Layout:      "row-major",
				Span:        expr.Span,
			}
		}
		if arrayType.ArrayElem != nil && !arrayType.ArrayElem.valueType().Equal(value.Type.valueType()) {
			return Value{}, evt1Diagnostic("CV4227", fmt.Sprintf("array literal element %d expected %s but got %s", i+1, arrayType.ArrayElem.String(), value.Type.String()), element.exprSpan())
		}
		elements = append(elements, value)
	}
	return Value{Kind: ValueArray, Type: arrayType, Elements: elements}, nil
}
