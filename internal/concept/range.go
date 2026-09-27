package concept

import (
	"fmt"
	"strings"
)

func evt1RangeType(element Type, span Span) Type {
	return Type{Name: "Range", Kind: TypeRange, TypeArgs: []Type{element.valueType()}, Span: span}
}

func evt1RangeElement(t Type) (Type, bool) {
	if t.Kind != TypeRange || len(t.TypeArgs) != 1 || !evt1IntegralRepresentation(t.TypeArgs[0]) {
		return Type{}, false
	}
	return t.TypeArgs[0], true
}

func evt1ValidateRangeExpr(env *semanticEnv, scope *evt1Scope, expr *BinaryExpr, templateInfo *evt1TemplateInfo, inComptimeFn bool) (Type, error) {
	if expr.Op == ".." {
		return evt1ValidateRangeBounds(env, scope, expr, templateInfo, inComptimeFn, false)
	}
	base, ok := expr.Left.(*BinaryExpr)
	if !ok || base.Op != ".." {
		return Type{}, evt1Diagnostic("RANGE_MODIFIER_INVALID", "step and descend require a finite start..end range", expr.Span)
	}
	baseType, err := evt1ValidateRangeBounds(env, scope, base, templateInfo, inComptimeFn, expr.Op == "descend")
	if err != nil {
		return Type{}, err
	}
	element := baseType.TypeArgs[0]
	if literal, ok := expr.Right.(*IntLiteral); ok {
		if err := evt1ResolveIntegerLiteral(literal, element); err != nil {
			return Type{}, err
		}
	} else {
		stepType, err := validateExpr(env, scope, expr.Right, templateInfo, inComptimeFn)
		if err != nil {
			return Type{}, err
		}
		if !stepType.valueType().Equal(element.valueType()) {
			return Type{}, evt1Diagnostic("RANGE_STEP_TYPE_MISMATCH", fmt.Sprintf("range %s magnitude must be %s", expr.Op, element.String()), expr.Right.exprSpan())
		}
	}
	if literal, ok := expr.Right.(*IntLiteral); ok && (literal.Magnitude == 0 || literal.Negative) {
		return Type{}, evt1Diagnostic("RANGE_STEP_INVALID", "range step or descend magnitude must be positive", literal.Span)
	}
	expr.ResolvedType = baseType
	return baseType, nil
}

func evt1ValidateRangeBounds(env *semanticEnv, scope *evt1Scope, expr *BinaryExpr, templateInfo *evt1TemplateInfo, inComptimeFn, descending bool) (Type, error) {
	var startType, endType Type
	var err error
	startLiteral, startIsLiteral := expr.Left.(*IntLiteral)
	endLiteral, endIsLiteral := expr.Right.(*IntLiteral)
	if startIsLiteral && !endIsLiteral {
		endType, err = validateExpr(env, scope, expr.Right, templateInfo, inComptimeFn)
		if err == nil && evt1IntegralRepresentation(endType) {
			err = evt1ResolveIntegerLiteral(startLiteral, endType)
			startType = endType
		}
	} else {
		startType, err = validateExpr(env, scope, expr.Left, templateInfo, inComptimeFn)
		if err == nil && endIsLiteral && evt1IntegralRepresentation(startType) {
			err = evt1ResolveIntegerLiteral(endLiteral, startType)
			endType = startType
		} else if err == nil {
			endType, err = validateExpr(env, scope, expr.Right, templateInfo, inComptimeFn)
		}
	}
	if err != nil {
		return Type{}, err
	}
	if !evt1IntegralRepresentation(startType) || !startType.valueType().Equal(endType.valueType()) || startType.Quantity != nil || endType.Quantity != nil {
		return Type{}, evt1Diagnostic("RANGE_ENDPOINT_TYPE_MISMATCH", "range endpoints require the same dimensionless integer type", expr.Span)
	}
	if startIsLiteral && endIsLiteral {
		invalid := false
		if startType.Name == "int" || startType.Name == "isize" {
			start, startOK := startLiteral.signed64()
			end, endOK := endLiteral.signed64()
			invalid = startOK && endOK && ((descending && start < end) || (!descending && start > end))
		} else {
			invalid = (descending && startLiteral.Magnitude < endLiteral.Magnitude) || (!descending && startLiteral.Magnitude > endLiteral.Magnitude)
		}
		if invalid {
			return Type{}, evt1Diagnostic("RANGE_DIRECTION_INVALID", "range direction contradicts its endpoints", expr.Span)
		}
	}
	result := evt1RangeType(startType, expr.Span)
	expr.ResolvedType = result
	return result, nil
}

func evt1RangeCName(element Type) string {
	return "concept_builtin_range_" + element.Name
}

func evt1RangeCDeclarations() string {
	var b strings.Builder
	for _, name := range []string{"int", "uint", "uint8", "byte", "uint64", "usize", "isize"} {
		element, _ := evt1BuiltinType(name, Span{})
		cType := evt1CType(element)
		fmt.Fprintf(&b, "typedef struct { %s start, end, step; bool descending; } %s;\n", cType, evt1RangeCName(element))
	}
	return b.String() + "\n"
}

func evt1ModuleUsesRange(l *lowering) bool {
	if evt1TypeUsed(l.module, func(t Type) bool { return t.Kind == TypeRange }) {
		return true
	}
	uses := func(fn MIRFunction) bool {
		for _, each := range fn.Foreaches {
			if each.SourceKind == "range" {
				return true
			}
		}
		for _, operation := range fn.Operations {
			if operation.Kind == "binary" && operation.Detail == ".." {
				return true
			}
		}
		return false
	}
	for _, fn := range l.mir.Functions {
		if uses(fn) {
			return true
		}
	}
	for _, instance := range l.env.templateInstances {
		if instance.Function.Body != nil {
			var fn MIRFunction
			collectMIROps(l.env, instance.Function.Body, &fn, nil)
			if uses(fn) {
				return true
			}
		}
	}
	// Machine state bodies are lowered directly and do not appear in the
	// ordinary function MIR list. Reuse the same operation collector.
	for _, automata := range l.module.Automata {
		for _, machine := range automata.Machines {
			for _, state := range machine.States {
				if state.Body != nil {
					var fn MIRFunction
					collectMIROps(l.env, state.Body, &fn, nil)
					if uses(fn) {
						return true
					}
				}
			}
		}
	}
	return false
}

func (f *evt1FunctionLowerer) lowerRangeExpr(expr *BinaryExpr, indent int) (string, string, Type) {
	bounds := expr
	stepExpr := Expr(&IntLiteral{Magnitude: 1, Lexeme: "1", ResolvedType: expr.ResolvedType.TypeArgs[0], Span: expr.Span})
	descending := false
	if expr.Op != ".." {
		bounds = expr.Left.(*BinaryExpr)
		stepExpr = expr.Right
		descending = expr.Op == "descend"
	}
	startPrelude, start, _ := f.lowerExpr(bounds.Left, indent)
	element := expr.ResolvedType.TypeArgs[0]
	endPrelude, end, _ := f.lowerExpr(bounds.Right, indent)
	stepPrelude, step, _ := f.lowerExpr(stepExpr, indent)
	startName, endName, stepName := f.nextTemp("range_start"), f.nextTemp("range_end"), f.nextTemp("range_step")
	cType := evt1CType(element)
	var b strings.Builder
	b.WriteString(startPrelude)
	fmt.Fprintf(&b, "%s%s %s = %s;\n", ind(indent), cType, startName, start)
	b.WriteString(endPrelude)
	fmt.Fprintf(&b, "%s%s %s = %s;\n", ind(indent), cType, endName, end)
	b.WriteString(stepPrelude)
	fmt.Fprintf(&b, "%s%s %s = %s;\n", ind(indent), cType, stepName, step)
	stepInvalid := stepName + " == 0"
	if element.Name == "int" || element.Name == "isize" {
		stepInvalid = stepName + " <= 0"
	}
	fmt.Fprintf(&b, "%sif (%s || %s %s %s) { concept_panic(\"invalid range step or direction\", %d, %d); }\n",
		ind(indent), stepInvalid, startName, map[bool]string{true: "<", false: ">"}[descending], endName, expr.Span.Line, expr.Span.Column)
	return b.String(), fmt.Sprintf("(%s){ .start = %s, .end = %s, .step = %s, .descending = %t }", evt1RangeCName(element), startName, endName, stepName, descending), expr.ResolvedType
}
