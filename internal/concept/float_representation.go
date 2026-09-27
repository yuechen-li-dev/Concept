package concept

import (
	"fmt"
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
