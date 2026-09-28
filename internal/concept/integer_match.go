package concept

import (
	"fmt"
	"strings"
)

func evt1PatternLabel(pattern Pattern) string {
	if pattern.Wildcard {
		return "_"
	}
	if pattern.Literal != nil {
		return pattern.Literal.Source()
	}
	return pattern.EnumName + "::" + pattern.VariantName
}

func evt1IntegerMatchSubject(subject string, t Type) string {
	if t.isReference() {
		return "*" + subject
	}
	return subject
}

func (f *evt1FunctionLowerer) lowerIntegerMatchStmt(stmt MatchStmt, prelude, subject string, subjectType Type, indent int) string {
	temp := f.nextTemp("subject")
	var b strings.Builder
	b.WriteString(prelude)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(subjectType), temp, subject))
	b.WriteString(ind(indent) + fmt.Sprintf("switch (%s) {\n", evt1IntegerMatchSubject(temp, subjectType)))
	for _, arm := range stmt.Arms {
		label := "default"
		if arm.Pattern.Literal != nil {
			label = "case " + evt1RenderIntegerLiteral(arm.Pattern.Literal)
		}
		b.WriteString(ind(indent) + label + ":\n")
		b.WriteString(ind(indent+1) + "{\n")
		f.pushScope()
		b.WriteString(f.lowerBlock(arm.Block, indent+2))
		b.WriteString(f.lowerCurrentScopeDrops(indent + 2))
		f.popScope()
		b.WriteString(ind(indent+2) + "break;\n")
		b.WriteString(ind(indent+1) + "}\n")
	}
	b.WriteString(ind(indent) + "}\n")
	return b.String()
}

func (f *evt1FunctionLowerer) lowerIntegerMatchExpr(expr *MatchExpr, prelude, subject string, subjectType Type, indent int) (string, string, Type) {
	resultType, _ := validateMatchExpr(f.l.env, f.typeScope(), *expr, nil, false)
	subjectTemp := f.nextTemp("match_subject")
	resultTemp := f.nextTemp("match_result")
	var b strings.Builder
	b.WriteString(prelude)
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(subjectType), subjectTemp, subject))
	b.WriteString(ind(indent) + fmt.Sprintf("%s %s;\n", evt1CType(resultType), resultTemp))
	b.WriteString(ind(indent) + fmt.Sprintf("switch (%s) {\n", evt1IntegerMatchSubject(subjectTemp, subjectType)))
	for _, arm := range expr.Arms {
		label := "default"
		if arm.Pattern.Literal != nil {
			label = "case " + evt1RenderIntegerLiteral(arm.Pattern.Literal)
		}
		b.WriteString(ind(indent) + label + ":\n")
		b.WriteString(ind(indent+1) + "{\n")
		f.pushScope()
		armPrelude, armExpr, _ := f.lowerExprExpected(arm.Value, resultType, indent+2)
		b.WriteString(armPrelude)
		b.WriteString(ind(indent+2) + fmt.Sprintf("%s = %s;\n", resultTemp, armExpr))
		b.WriteString(f.lowerCurrentScopeDrops(indent + 2))
		f.popScope()
		b.WriteString(ind(indent+2) + "break;\n")
		b.WriteString(ind(indent+1) + "}\n")
	}
	b.WriteString(ind(indent) + "}\n")
	return b.String(), resultTemp, resultType
}
