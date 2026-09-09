package concept

import (
	"fmt"
	"strings"
)

func (f *evt1FunctionLowerer) lowerTestToolingCall(call *CallExpr, indent int) (string, string, Type) {
	voidType, _ := evt1BuiltinType("void", call.Span)
	if call.Intrinsic == "test_foretell_checkpoint" {
		_, message, _ := f.lowerExpr(call.Args[0], indent)
		return "", fmt.Sprintf("fprintf(stderr, \"CONCEPT_TEST_CHECKPOINT|%d|%%s\\n\", %s)", call.Span.Line, message), voidType
	}

	values := call.Args[:len(call.Args)-1]
	_, reason, _ := f.lowerExpr(call.Args[len(call.Args)-1], indent)
	var prelude strings.Builder
	exprs := make([]string, len(values))
	types := make([]Type, len(values))
	for i, value := range values {
		valuePrelude, expression, valueType := f.lowerExpr(value, indent)
		prelude.WriteString(valuePrelude)
		temp := f.nextTemp("assert_value")
		prelude.WriteString(ind(indent) + fmt.Sprintf("%s %s = %s;\n", evt1CType(valueType), temp, expression))
		exprs[i], types[i] = temp, valueType
	}
	failure := func(condition, detail string, customArgs ...string) (string, string, Type) {
		printArgs := evt1TestPrintArgs(types, exprs)
		if len(customArgs) > 0 {
			printArgs = ", " + strings.Join(customArgs, ", ")
		}
		prelude.WriteString(ind(indent) + "if (!(" + condition + ")) {\n")
		prelude.WriteString(ind(indent+1) + fmt.Sprintf("fprintf(stderr, \"CONCEPT_TEST_ASSERT|%s|%d|%d|%%s%s\\n\", %s%s);\n", call.Callee, call.Span.Line, call.Span.Column, detail, reason, printArgs))
		prelude.WriteString(ind(indent+1) + "exit(101);\n")
		prelude.WriteString(ind(indent) + "}\n")
		return prelude.String(), "(void)0", voidType
	}
	switch call.Intrinsic {
	case "test_assert_true":
		return failure(exprs[0], "|actual=%d")
	case "test_assert_false":
		return failure("!("+exprs[0]+")", "|actual=%d")
	case "test_assert_equals":
		condition := exprs[0] + " == " + exprs[1]
		if types[0].Name == "string" {
			condition = "strcmp(" + exprs[0] + ", " + exprs[1] + ") == 0"
		} else if types[0].Kind == TypeEnum {
			condition = exprs[0] + ".tag == " + exprs[1] + ".tag"
		}
		return failure(condition, "|actual="+evt1TestFormat(types[0])+"|expected="+evt1TestFormat(types[1]))
	case "test_assert_near":
		condition := fmt.Sprintf("(%s == %s) || ((%s >= 0) && isfinite((double)%s) && isfinite((double)%s) && isfinite((double)%s) && fabs((double)%s - (double)%s) <= (double)%s)", exprs[0], exprs[1], exprs[2], exprs[0], exprs[1], exprs[2], exprs[0], exprs[1], exprs[2])
		return failure(condition, "|actual="+evt1TestFormat(types[0])+"|expected="+evt1TestFormat(types[1])+"|tolerance="+evt1TestFormat(types[2]))
	case "test_assert_error":
		detail := "|result_tag=%d"
		args := []string{exprs[0] + ".tag"}
		payload := evt1FailureSuccessType(types[0])
		if payload.Name != "void" && evt1TestPrintableType(payload) {
			detail += "|success=" + evt1TestFormat(payload)
			value := exprs[0] + ".payload.ok.value"
			if payload.Name == "float" {
				value = "(double)" + value
			} else if payload.Kind == TypeEnum {
				value += ".tag"
			}
			args = append(args, value)
		}
		return failure(exprs[0]+".tag == 1", detail, args...)
	case "test_assert_lgtm":
		detail := "|result_tag=%d"
		args := []string{exprs[0] + ".tag"}
		payload := evt1FailureErrorType(types[0])
		if evt1TestPrintableType(payload) {
			detail += "|error=" + evt1TestFormat(payload)
			value := exprs[0] + ".payload.error.error"
			if payload.Name == "float" {
				value = "(double)" + value
			} else if payload.Kind == TypeEnum {
				value += ".tag"
			}
			args = append(args, value)
		}
		return failure(exprs[0]+".tag == 0", detail, args...)
	default:
		return "", "(void)0", voidType
	}
}

func evt1TestFormat(t Type) string {
	switch t.Name {
	case "float":
		return "%g"
	case "uint", "byte":
		return "%u"
	case "string":
		return "%s"
	default:
		return "%d"
	}
}

func evt1TestPrintArgs(types []Type, exprs []string) string {
	var args []string
	for i, expression := range exprs {
		if evt1IsResultType(types[i]) {
			args = append(args, expression+".tag")
		} else if types[i].Kind == TypeEnum {
			args = append(args, expression+".tag")
		} else if types[i].Name == "float" {
			args = append(args, "(double)"+expression)
		} else {
			args = append(args, expression)
		}
	}
	if len(args) == 0 {
		return ""
	}
	return ", " + strings.Join(args, ", ")
}

func evt1TestPrintableType(t Type) bool {
	return evt1TestNumericType(t) || t.Name == "bool" || t.Name == "string" || t.Kind == TypeEnum
}

func evt1ModuleUsesTestNear(module Module) bool {
	return evt1ModuleUsesTestIntrinsic(module, "test_assert_near")
}
func evt1ModuleUsesTestStringEquality(module Module) bool {
	return evt1ModuleUsesTestIntrinsic(module, "test_assert_equals")
}

func evt1ModuleUsesTestIntrinsic(module Module, intrinsic string) bool {
	var exprUses func(Expr) bool
	exprUses = func(expr Expr) bool {
		switch e := expr.(type) {
		case *CallExpr:
			if e.Intrinsic == intrinsic {
				return true
			}
			for _, arg := range e.Args {
				if exprUses(arg) {
					return true
				}
			}
		case *BinaryExpr:
			return exprUses(e.Left) || exprUses(e.Right)
		case *ParenExpr:
			return exprUses(e.Value)
		}
		return false
	}
	var blockUses func(Block) bool
	blockUses = func(block Block) bool {
		for _, raw := range block.Statements {
			switch s := raw.(type) {
			case *ExprStmt:
				if exprUses(s.Value) {
					return true
				}
			case *VarDecl:
				if exprUses(s.Value) {
					return true
				}
			case *IfStmt:
				if exprUses(s.Condition) || blockUses(s.Then) || (s.Else != nil && blockUses(*s.Else)) {
					return true
				}
			}
		}
		return false
	}
	for _, fn := range module.Functions {
		if fn.Body != nil && blockUses(*fn.Body) {
			return true
		}
	}
	return false
}
