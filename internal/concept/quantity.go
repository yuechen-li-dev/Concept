package concept

import (
	"fmt"
	"strings"
)

// QuantityDimension is the compile-time-only normalized unit identity carried
// by numeric types. It follows Oct's fixed exponent-vector authority and adds
// Information as the systems-storage dimension. Scale is an exact rational
// relative to the canonical bases (bit for Information).
type QuantityDimension struct {
	Exponents        [8]int `json:"exponents"`
	ScaleNumerator   int    `json:"scale_numerator"`
	ScaleDenominator int    `json:"scale_denominator"`
}

const (
	quantityLength = iota
	quantityMass
	quantityTime
	quantityCurrent
	quantityTemperature
	quantityAmount
	quantityLuminousIntensity
	quantityInformation
)

var quantityBaseNames = [...]string{"m", "kg", "s", "A", "K", "mol", "cd", "bit"}

func dimensionlessQuantity() QuantityDimension {
	return QuantityDimension{ScaleNumerator: 1, ScaleDenominator: 1}
}

func quantityFromUnit(name string) (QuantityDimension, bool) {
	d := dimensionlessQuantity()
	if name == "Hz" {
		d.Exponents[quantityTime] = -1
		return d, true
	}
	if name == "byte" {
		d.Exponents[quantityInformation] = 1
		d.ScaleNumerator = 8
		return d, true
	}
	for i, base := range quantityBaseNames {
		if name == base {
			d.Exponents[i] = 1
			return d, true
		}
	}
	return QuantityDimension{}, false
}

func (d QuantityDimension) normalized() QuantityDimension {
	if d.ScaleNumerator == 0 {
		d.ScaleNumerator = 1
	}
	if d.ScaleDenominator == 0 {
		d.ScaleDenominator = 1
	}
	g := quantityGCD(d.ScaleNumerator, d.ScaleDenominator)
	d.ScaleNumerator /= g
	d.ScaleDenominator /= g
	return d
}

func quantityGCD(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func (d QuantityDimension) IsDimensionless() bool {
	for _, exponent := range d.Exponents {
		if exponent != 0 {
			return false
		}
	}
	return true
}

func (d QuantityDimension) SameDimension(other QuantityDimension) bool {
	return d.Exponents == other.Exponents
}

func (d QuantityDimension) Equal(other QuantityDimension) bool {
	a, b := d.normalized(), other.normalized()
	return a.Exponents == b.Exponents && a.ScaleNumerator == b.ScaleNumerator && a.ScaleDenominator == b.ScaleDenominator
}

func (d QuantityDimension) Multiply(other QuantityDimension) QuantityDimension {
	out := dimensionlessQuantity()
	for i := range out.Exponents {
		out.Exponents[i] = d.Exponents[i] + other.Exponents[i]
	}
	out.ScaleNumerator = d.normalized().ScaleNumerator * other.normalized().ScaleNumerator
	out.ScaleDenominator = d.normalized().ScaleDenominator * other.normalized().ScaleDenominator
	return out.normalized()
}

func (d QuantityDimension) Divide(other QuantityDimension) QuantityDimension {
	out := dimensionlessQuantity()
	for i := range out.Exponents {
		out.Exponents[i] = d.Exponents[i] - other.Exponents[i]
	}
	out.ScaleNumerator = d.normalized().ScaleNumerator * other.normalized().ScaleDenominator
	out.ScaleDenominator = d.normalized().ScaleDenominator * other.normalized().ScaleNumerator
	return out.normalized()
}

func (d QuantityDimension) Pow(exponent int) QuantityDimension {
	out := dimensionlessQuantity()
	for i := range out.Exponents {
		out.Exponents[i] = d.Exponents[i] * exponent
	}
	base := d.normalized()
	for i := 0; i < exponent; i++ {
		out.ScaleNumerator *= base.ScaleNumerator
		out.ScaleDenominator *= base.ScaleDenominator
	}
	if exponent < 0 {
		out.ScaleNumerator, out.ScaleDenominator = 1, 1
		for i := 0; i > exponent; i-- {
			out.ScaleNumerator *= base.ScaleDenominator
			out.ScaleDenominator *= base.ScaleNumerator
		}
	}
	return out.normalized()
}

func (d QuantityDimension) String() string {
	d = d.normalized()
	if d.IsDimensionless() {
		return ""
	}
	names := quantityBaseNames
	if d.Exponents[quantityInformation] == 1 && d.ScaleNumerator == 8 && d.ScaleDenominator == 1 {
		names[quantityInformation] = "byte"
	}
	var numerator, denominator []string
	for i, exponent := range d.Exponents {
		name := names[i]
		if exponent > 0 {
			numerator = append(numerator, quantityUnitTerm(name, exponent))
		}
		if exponent < 0 {
			denominator = append(denominator, quantityUnitTerm(name, -exponent))
		}
	}
	left := "1"
	if len(numerator) > 0 {
		left = strings.Join(numerator, "*")
	}
	if len(denominator) > 0 {
		return left + "/" + strings.Join(denominator, "*")
	}
	return left
}

func quantityUnitTerm(name string, exponent int) string {
	if exponent == 1 {
		return name
	}
	return fmt.Sprintf("%s^%d", name, exponent)
}

func evt1QuantityType(representation string, dimension QuantityDimension, span Span) Type {
	return Type{Name: representation, Kind: TypeBuiltin, Quantity: &dimension, Span: span}
}

func evt1ByteQuantityType(span Span) Type {
	d, _ := quantityFromUnit("byte")
	return evt1QuantityType("usize", d, span)
}

func evt1NumericRepresentation(t Type) bool {
	switch t.Name {
	case "int", "uint", "uint8", "byte", "uint64", "usize", "isize", "float":
		return t.Kind == TypeBuiltin
	default:
		return false
	}
}

func evt1IntegralRepresentation(t Type) bool {
	return evt1NumericRepresentation(t) && t.Name != "float"
}

func evt1DimensionlessNumeric(t Type) bool {
	return evt1NumericRepresentation(t) && (t.Quantity == nil || t.Quantity.IsDimensionless())
}

func evt1QuantityResult(t Type, dimension QuantityDimension, span Span) Type {
	out := t.valueType()
	out.Span = span
	if dimension.IsDimensionless() {
		out.Quantity = nil
	} else {
		dimension = dimension.normalized()
		out.Quantity = &dimension
	}
	return out
}

func evt1ValidateNumericBinary(left, right Type, op string, span Span) (Type, bool, error) {
	if !evt1NumericRepresentation(left) || !evt1NumericRepresentation(right) || left.Name != right.Name {
		return Type{}, false, nil
	}
	leftDimension, rightDimension := dimensionlessQuantity(), dimensionlessQuantity()
	if left.Quantity != nil {
		leftDimension = left.Quantity.normalized()
	}
	if right.Quantity != nil {
		rightDimension = right.Quantity.normalized()
	}
	comparison := op == "<" || op == ">" || op == "<=" || op == ">=" || op == "==" || op == "!="
	if comparison || op == "+" || op == "-" || op == "%" {
		if op == "%" {
			if !evt1IntegralRepresentation(left) {
				return Type{}, true, evt1Diagnostic("MODULO_REQUIRES_INTEGRAL", "% requires integral operands", span)
			}
			if left.Name == "int" || left.Name == "isize" {
				return Type{}, true, evt1Diagnostic("SIGNED_EUCLIDEAN_MODULO_DEFERRED", "signed % is reserved for Euclidean modulo lowering with explicit zero and minimum-value handling; C remainder semantics are not exposed", span)
			}
		}
		if !leftDimension.Equal(rightDimension) {
			return Type{}, true, evt1Diagnostic("QUANTITY_DIMENSION_MISMATCH", fmt.Sprintf("cannot apply %s to %s and %s; operands require identical units", op, left.String(), right.String()), span)
		}
		if comparison {
			out, _ := evt1BuiltinType("bool", span)
			return out, true, nil
		}
		return evt1QuantityResult(left, leftDimension, span), true, nil
	}
	if op == "*" {
		return evt1QuantityResult(left, leftDimension.Multiply(rightDimension), span), true, nil
	}
	if op == "/" {
		return evt1QuantityResult(left, leftDimension.Divide(rightDimension), span), true, nil
	}
	if op == "&" || op == "|" || op == "^" || op == "<<" || op == ">>" {
		if !evt1IntegralRepresentation(left) || !evt1DimensionlessNumeric(left) || !evt1DimensionlessNumeric(right) {
			return Type{}, true, evt1Diagnostic("QUANTITY_BITWISE_REQUIRES_DIMENSIONLESS", fmt.Sprintf("%s requires dimensionless integral operands", op), span)
		}
		return left, true, nil
	}
	return Type{}, false, nil
}

func evt1ByteDisplacement(t Type) bool {
	if t.Quantity == nil || (t.Name != "usize" && t.Name != "isize") {
		return false
	}
	unit, _ := quantityFromUnit("byte")
	return t.Quantity.Equal(unit)
}

func evt1AddressType(space Type, span Span) Type {
	return Type{Name: "Address", Kind: TypeAddress, TypeArgs: []Type{space.valueType()}, Span: span}
}

func evt1StorageType(element Type, span Span) Type {
	return Type{Name: "Storage", Kind: TypeTypedStorage, TypeArgs: []Type{element.valueType()}, Span: span}
}
