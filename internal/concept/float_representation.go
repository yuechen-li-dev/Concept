package concept

import (
	"fmt"
	"math"
	"strings"
)

// FloatRepresentation records the encoding independently of a quantity unit.
// These values are serialized with Type in concept-module artifacts.
type FloatRepresentation string

const (
	FloatBinary16 FloatRepresentation = "binary16"
	FloatBinary32 FloatRepresentation = "binary32"
	FloatBinary64 FloatRepresentation = "binary64"
	FloatBFloat16 FloatRepresentation = "bfloat16"
	Float8E4M3    FloatRepresentation = "float8e4m3"
	Float8E5M2    FloatRepresentation = "float8e5m2"
)

type FloatRepresentationInfo struct {
	Representation FloatRepresentation
	Bits           int
	ExponentBits   int
	MantissaBits   int
}

func evt1FloatRepresentationInfo(t Type) (FloatRepresentationInfo, bool) {
	if t.Kind != TypeBuiltin {
		return FloatRepresentationInfo{}, false
	}
	representation := t.FloatRepresentation
	// Older artifacts and compiler-created float values predate this field.
	if representation == "" {
		switch t.Name {
		case "half", "float16":
			representation = FloatBinary16
		case "float", "float32":
			representation = FloatBinary32
		case "double", "float64":
			representation = FloatBinary64
		}
	}
	switch representation {
	case FloatBinary16:
		return FloatRepresentationInfo{representation, 16, 5, 10}, true
	case FloatBinary32:
		return FloatRepresentationInfo{representation, 32, 8, 23}, true
	case FloatBinary64:
		return FloatRepresentationInfo{representation, 64, 11, 52}, true
	case FloatBFloat16:
		return FloatRepresentationInfo{representation, 16, 8, 7}, true
	case Float8E4M3:
		return FloatRepresentationInfo{representation, 8, 4, 3}, true
	case Float8E5M2:
		return FloatRepresentationInfo{representation, 8, 5, 2}, true
	default:
		return FloatRepresentationInfo{}, false
	}
}

func evt1IsFloating(t Type) bool {
	_, ok := evt1FloatRepresentationInfo(t)
	return ok
}

func evt1ContextualFloatLiteral(expr Expr) *FloatLiteral {
	switch e := expr.(type) {
	case *FloatLiteral:
		return e
	case *ParenExpr:
		return evt1ContextualFloatLiteral(e.Value)
	case *UnaryExpr:
		if e.Op == "-" {
			return evt1ContextualFloatLiteral(e.Value)
		}
	}
	return nil
}

func evt1ResolveFloatLiteral(literal *FloatLiteral, target Type) error {
	info, ok := evt1FloatRepresentationInfo(target)
	if !ok {
		return evt1Diagnostic("FLOAT_LITERAL_TARGET_INVALID", "float literal requires a floating scalar target", literal.Span)
	}
	max, minSubnormal := math.MaxFloat64, math.SmallestNonzeroFloat64
	switch info.Representation {
	case FloatBinary16:
		max, minSubnormal = 65504, math.Ldexp(1, -24)
	case FloatBinary32:
		max, minSubnormal = math.MaxFloat32, math.SmallestNonzeroFloat32
	}
	magnitude := math.Abs(literal.Value)
	if math.IsInf(magnitude, 0) || math.IsNaN(magnitude) || magnitude > max || (magnitude != 0 && magnitude < minSubnormal/2) {
		return evt1Diagnostic("FLOAT_LITERAL_OUT_OF_RANGE", fmt.Sprintf("float literal is outside the representable range of %s", target.String()), literal.Span)
	}
	literal.ResolvedType = target.valueType()
	return nil
}

func evt1RenderFloatLiteral(literal *FloatLiteral, target Type) string {
	value := fmt.Sprintf("%g", literal.Value)
	if !strings.ContainsAny(value, ".eE") {
		value += ".0"
	}
	if info, ok := evt1FloatRepresentationInfo(target); !ok || info.Representation != FloatBinary64 {
		value += "f"
	}
	return value
}
