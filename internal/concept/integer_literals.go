package concept

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	evt1IntMin = int64(math.MinInt32)
	evt1IntMax = int64(math.MaxInt32)
)

func evt1ParseIntegerLiteral(lexeme string, negative bool, span Span) (*IntLiteral, error) {
	magnitude, err := strconv.ParseUint(lexeme, 0, 64)
	if err != nil {
		return nil, evt1Diagnostic("CV4644", fmt.Sprintf("integer literal %s is outside the supported uint64 magnitude range", lexeme), span)
	}
	return &IntLiteral{Magnitude: magnitude, Negative: negative, Lexeme: lexeme, Span: span}, nil
}

func evt1IntegerTypeRange(t Type) (negativeAllowed bool, maxMagnitude uint64, minMagnitude uint64, width int, ok bool) {
	t = t.valueType()
	switch t.Name {
	case "int":
		return true, uint64(math.MaxInt32), uint64(math.MaxInt32) + 1, 32, true
	case "uint":
		return false, uint64(math.MaxUint32), 0, 32, true
	case "uint8", "byte":
		return false, uint64(math.MaxUint8), 0, 8, true
	case "uint64", "usize":
		return false, math.MaxUint64, 0, 64, true
	case "isize":
		// EVT1 pins scalar arithmetic independently of the C host. isize remains
		// the explicitly pointer-sized interop type and admits the portable i64
		// source range; the target C compiler performs the final ABI constraint.
		return true, uint64(math.MaxInt64), uint64(math.MaxInt64) + 1, 64, true
	default:
		return false, 0, 0, 0, false
	}
}

func evt1ResolveIntegerLiteral(literal *IntLiteral, target Type) error {
	negativeAllowed, maxMagnitude, minMagnitude, width, ok := evt1IntegerTypeRange(target)
	if !ok {
		return evt1Diagnostic("CV4644", fmt.Sprintf("integer literal %s cannot target non-integral type %s", literal.Source(), target.String()), literal.Span)
	}
	limit := maxMagnitude
	if literal.Negative {
		if !negativeAllowed {
			return evt1Diagnostic("CV4644", fmt.Sprintf("integer literal %s is out of range for %s (%d-bit unsigned)", literal.Source(), target.String(), width), literal.Span)
		}
		limit = minMagnitude
	}
	if literal.Magnitude > limit {
		return evt1Diagnostic("CV4644", fmt.Sprintf("integer literal %s is out of range for %s (%d-bit)", literal.Source(), target.String(), width), literal.Span)
	}
	literal.ResolvedType = target.valueType()
	return nil
}

func (literal *IntLiteral) Source() string {
	lexeme := literal.Lexeme
	if lexeme == "" {
		lexeme = strconv.FormatUint(literal.Magnitude, 10)
	}
	if literal.Negative {
		return "-" + lexeme
	}
	return lexeme
}

func (literal *IntLiteral) signed64() (int64, bool) {
	if literal.Negative {
		if literal.Magnitude > uint64(math.MaxInt64)+1 {
			return 0, false
		}
		if literal.Magnitude == uint64(math.MaxInt64)+1 {
			return math.MinInt64, true
		}
		return -int64(literal.Magnitude), true
	}
	if literal.Magnitude > uint64(math.MaxInt64) {
		return 0, false
	}
	return int64(literal.Magnitude), true
}

func (literal *IntLiteral) boundedInt() (int, bool) {
	value, ok := literal.signed64()
	if !ok || int64(int(value)) != value {
		return 0, false
	}
	return int(value), true
}

func evt1RenderIntegerLiteral(literal *IntLiteral) string {
	t := literal.ResolvedType
	if t.Name == "" {
		t = Type{Name: "int", Kind: TypeBuiltin}
	}
	source := strconv.FormatUint(literal.Magnitude, 10)
	if literal.Negative {
		if t.Name == "int" && literal.Magnitude == uint64(math.MaxInt32)+1 {
			return "(-INT32_C(2147483647) - INT32_C(1))"
		}
		return "(-" + source + ")"
	}
	switch t.Name {
	case "uint", "uint8", "byte":
		return "UINT32_C(" + source + ")"
	case "uint64", "usize":
		return "UINT64_C(" + source + ")"
	case "int":
		return "INT32_C(" + source + ")"
	default:
		return source
	}
}

func evt1MIRUsesDefinedIntegerArithmetic(mir MIR) bool {
	for _, fn := range mir.Functions {
		for _, operation := range fn.Operations {
			if operation.Kind != "binary" {
				continue
			}
			switch operation.Detail {
			case "+", "-", "*", "/", "<<", ">>":
				return true
			}
		}
	}
	return false
}

func evt1IntegerHelperName(base, operation string) string {
	return "concept_rt_" + base + "_" + operation
}

func evt1DefinedIntegerArithmeticHelpers(base, generatedBody string) string {
	var b strings.Builder
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "i32_add")+"(") {
		b.WriteString("int32_t " + evt1IntegerHelperName(base, "i32_add") + "(int32_t left, int32_t right, int line, int column) {\n")
		b.WriteString("  int64_t result = (int64_t)left + (int64_t)right;\n")
		b.WriteString("  if (result < INT32_MIN || result > INT32_MAX) { concept_panic(\"int32 addition overflow\", line, column); }\n")
		b.WriteString("  return (int32_t)result;\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "i32_sub")+"(") {
		b.WriteString("int32_t " + evt1IntegerHelperName(base, "i32_sub") + "(int32_t left, int32_t right, int line, int column) {\n")
		b.WriteString("  int64_t result = (int64_t)left - (int64_t)right;\n")
		b.WriteString("  if (result < INT32_MIN || result > INT32_MAX) { concept_panic(\"int32 subtraction overflow\", line, column); }\n")
		b.WriteString("  return (int32_t)result;\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "i32_mul")+"(") {
		b.WriteString("int32_t " + evt1IntegerHelperName(base, "i32_mul") + "(int32_t left, int32_t right, int line, int column) {\n")
		b.WriteString("  int64_t result = (int64_t)left * (int64_t)right;\n")
		b.WriteString("  if (result < INT32_MIN || result > INT32_MAX) { concept_panic(\"int32 multiplication overflow\", line, column); }\n")
		b.WriteString("  return (int32_t)result;\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "i32_div")+"(") {
		b.WriteString("int32_t " + evt1IntegerHelperName(base, "i32_div") + "(int32_t left, int32_t right, int line, int column) {\n")
		b.WriteString("  if (right == 0) { concept_panic(\"integer division by zero\", line, column); }\n")
		b.WriteString("  if (left == INT32_MIN && right == -1) { concept_panic(\"int32 division overflow\", line, column); }\n")
		b.WriteString("  return left / right;\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "i32_shl")+"(") {
		b.WriteString("int32_t " + evt1IntegerHelperName(base, "i32_shl") + "(int32_t value, int32_t count, int line, int column) {\n")
		b.WriteString("  if (count < 0 || count >= 32) { concept_panic(\"invalid int32 shift count\", line, column); }\n")
		b.WriteString("  uint32_t bits = ((uint32_t)value) << (uint32_t)count;\n")
		b.WriteString("  if (bits <= INT32_MAX) { return (int32_t)bits; }\n")
		b.WriteString("  return INT32_MIN + (int32_t)(bits - UINT32_C(2147483648));\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "i32_shr")+"(") {
		b.WriteString("int32_t " + evt1IntegerHelperName(base, "i32_shr") + "(int32_t value, int32_t count, int line, int column) {\n")
		b.WriteString("  if (count < 0 || count >= 32) { concept_panic(\"invalid int32 shift count\", line, column); }\n")
		b.WriteString("  int64_t divisor = INT64_C(1) << (uint32_t)count;\n")
		b.WriteString("  int64_t wide = value;\n")
		b.WriteString("  return (int32_t)(wide >= 0 ? wide / divisor : -(((-wide) + divisor - 1) / divisor));\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "u64_div")+"(") {
		b.WriteString("uint64_t " + evt1IntegerHelperName(base, "u64_div") + "(uint64_t left, uint64_t right, int bits, int line, int column) {\n")
		b.WriteString("  (void)bits;\n")
		b.WriteString("  if (right == 0) { concept_panic(\"integer division by zero\", line, column); }\n")
		b.WriteString("  return left / right;\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "u64_shl")+"(") {
		b.WriteString("uint64_t " + evt1IntegerHelperName(base, "u64_shl") + "(uint64_t value, uint64_t count, int bits, int line, int column) {\n")
		b.WriteString("  if (count >= (uint64_t)bits) { concept_panic(\"invalid unsigned shift count\", line, column); }\n")
		b.WriteString("  return value << count;\n}\n")
	}
	if strings.Contains(generatedBody, evt1IntegerHelperName(base, "u64_shr")+"(") {
		b.WriteString("uint64_t " + evt1IntegerHelperName(base, "u64_shr") + "(uint64_t value, uint64_t count, int bits, int line, int column) {\n")
		b.WriteString("  if (count >= (uint64_t)bits) { concept_panic(\"invalid unsigned shift count\", line, column); }\n")
		b.WriteString("  return value >> count;\n}\n\n")
	}
	return b.String()
}
