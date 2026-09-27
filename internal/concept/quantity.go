package concept

import (
	"fmt"
	"math"
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

// This is an explicit vocabulary, not an SI-prefix generator. The rational
// scale is relative to m, kg, s, A, K, mol, cd, and bit respectively.
var standardQuantityUnits = []struct {
	Name                   string
	Exponents              [8]int
	Numerator, Denominator int
}{
	{"nm", [8]int{1}, 1, 1000000000},
	{"um", [8]int{1}, 1, 1000000},
	{"mm", [8]int{1}, 1, 1000},
	{"cm", [8]int{1}, 1, 100},
	{"m", [8]int{1}, 1, 1},
	{"km", [8]int{1}, 1000, 1},
	{"mg", [8]int{0, 1}, 1, 1000000},
	{"g", [8]int{0, 1}, 1, 1000},
	{"kg", [8]int{0, 1}, 1, 1},
	{"ns", [8]int{0, 0, 1}, 1, 1000000000},
	{"us", [8]int{0, 0, 1}, 1, 1000000},
	{"ms", [8]int{0, 0, 1}, 1, 1000},
	{"s", [8]int{0, 0, 1}, 1, 1},
	{"A", [8]int{0, 0, 0, 1}, 1, 1},
	{"mA", [8]int{0, 0, 0, 1}, 1, 1000},
	{"K", [8]int{0, 0, 0, 0, 1}, 1, 1},
	{"mol", [8]int{0, 0, 0, 0, 0, 1}, 1, 1},
	{"cd", [8]int{0, 0, 0, 0, 0, 0, 1}, 1, 1},
	{"bit", [8]int{0, 0, 0, 0, 0, 0, 0, 1}, 1, 1},
	{"byte", [8]int{0, 0, 0, 0, 0, 0, 0, 1}, 8, 1},
	{"Hz", [8]int{0, 0, -1}, 1, 1},
	{"kHz", [8]int{0, 0, -1}, 1000, 1},
	{"MHz", [8]int{0, 0, -1}, 1000000, 1},
	{"GHz", [8]int{0, 0, -1}, 1000000000, 1},
	{"N", [8]int{1, 1, -2}, 1, 1},
	{"mN", [8]int{1, 1, -2}, 1, 1000},
	{"kN", [8]int{1, 1, -2}, 1000, 1},
	{"Pa", [8]int{-1, 1, -2}, 1, 1},
	{"kPa", [8]int{-1, 1, -2}, 1000, 1},
	{"MPa", [8]int{-1, 1, -2}, 1000000, 1},
	{"GPa", [8]int{-1, 1, -2}, 1000000000, 1},
	{"W", [8]int{2, 1, -3}, 1, 1},
	{"mW", [8]int{2, 1, -3}, 1, 1000},
	{"kW", [8]int{2, 1, -3}, 1000, 1},
	{"MW", [8]int{2, 1, -3}, 1000000, 1},
	{"V", [8]int{2, 1, -3, -1}, 1, 1},
	{"mV", [8]int{2, 1, -3, -1}, 1, 1000},
	{"kV", [8]int{2, 1, -3, -1}, 1000, 1},
}

func dimensionlessQuantity() QuantityDimension {
	return QuantityDimension{ScaleNumerator: 1, ScaleDenominator: 1}
}

func quantityFromUnit(name string) (QuantityDimension, bool) {
	for _, unit := range standardQuantityUnits {
		if name == unit.Name {
			return QuantityDimension{Exponents: unit.Exponents, ScaleNumerator: unit.Numerator, ScaleDenominator: unit.Denominator}, true
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

func (d QuantityDimension) ScaleRatioTo(other QuantityDimension) (int, int) {
	n, den, ok := d.ScaleRatioToChecked(other)
	if !ok {
		panic("quantity scale ratio exceeds bounded rational range")
	}
	return n, den
}

func (d QuantityDimension) ScaleRatioToChecked(other QuantityDimension) (int, int, bool) {
	a, b := d.normalized(), other.normalized()
	// Cancel before multiplying to keep ordinary curated scales in int64.
	n, den := a.ScaleNumerator, a.ScaleDenominator
	x, y := b.ScaleDenominator, b.ScaleNumerator
	g := quantityGCD(n, y)
	n, y = n/g, y/g
	g = quantityGCD(x, den)
	x, den = x/g, den/g
	if n > math.MaxInt/x || den > math.MaxInt/y {
		return 0, 0, false
	}
	return n * x, den * y, true
}

func quantityScaleMultiply(a, b QuantityDimension) (int, int, bool) {
	a, b = a.normalized(), b.normalized()
	// Cross cancellation precedes multiplication, so equivalent large scales
	// never overflow merely because they were written as products.
	g := quantityGCD(a.ScaleNumerator, b.ScaleDenominator)
	a.ScaleNumerator, b.ScaleDenominator = a.ScaleNumerator/g, b.ScaleDenominator/g
	g = quantityGCD(b.ScaleNumerator, a.ScaleDenominator)
	b.ScaleNumerator, a.ScaleDenominator = b.ScaleNumerator/g, a.ScaleDenominator/g
	if a.ScaleNumerator > math.MaxInt/b.ScaleNumerator || a.ScaleDenominator > math.MaxInt/b.ScaleDenominator {
		return 0, 0, false
	}
	return a.ScaleNumerator * b.ScaleNumerator, a.ScaleDenominator * b.ScaleDenominator, true
}

func (d QuantityDimension) MultiplyChecked(other QuantityDimension) (QuantityDimension, bool) {
	out := dimensionlessQuantity()
	for i := range out.Exponents {
		out.Exponents[i] = d.Exponents[i] + other.Exponents[i]
	}
	n, den, ok := quantityScaleMultiply(d, other)
	if !ok {
		return QuantityDimension{}, false
	}
	out.ScaleNumerator, out.ScaleDenominator = n, den
	return out.normalized(), true
}

func (d QuantityDimension) DivideChecked(other QuantityDimension) (QuantityDimension, bool) {
	inverse := other.normalized()
	inverse.ScaleNumerator, inverse.ScaleDenominator = inverse.ScaleDenominator, inverse.ScaleNumerator
	for i := range inverse.Exponents {
		inverse.Exponents[i] = -inverse.Exponents[i]
	}
	return d.MultiplyChecked(inverse)
}

func (d QuantityDimension) PowChecked(exponent int) (QuantityDimension, bool) {
	out, base := dimensionlessQuantity(), d.normalized()
	if exponent < 0 {
		base.ScaleNumerator, base.ScaleDenominator = base.ScaleDenominator, base.ScaleNumerator
		for i := range base.Exponents {
			base.Exponents[i] = -base.Exponents[i]
		}
		exponent = -exponent
	}
	for exponent > 0 {
		if exponent&1 != 0 {
			var ok bool
			out, ok = out.MultiplyChecked(base)
			if !ok {
				return QuantityDimension{}, false
			}
		}
		exponent >>= 1
		if exponent != 0 {
			var ok bool
			base, ok = base.MultiplyChecked(base)
			if !ok {
				return QuantityDimension{}, false
			}
		}
	}
	return out, true
}

func (d QuantityDimension) Multiply(other QuantityDimension) QuantityDimension {
	out, ok := d.MultiplyChecked(other)
	if !ok {
		panic("quantity scale exceeds bounded rational range")
	}
	return out
}

func (d QuantityDimension) Divide(other QuantityDimension) QuantityDimension {
	out, ok := d.DivideChecked(other)
	if !ok {
		panic("quantity scale exceeds bounded rational range")
	}
	return out
}

func (d QuantityDimension) Pow(exponent int) QuantityDimension {
	out, ok := d.PowChecked(exponent)
	if !ok {
		panic("quantity scale exceeds bounded rational range")
	}
	return out
}

func (d QuantityDimension) String() string {
	d = d.normalized()
	if d.IsDimensionless() {
		return ""
	}
	for _, unit := range standardQuantityUnits {
		if d.Exponents == unit.Exponents && d.ScaleNumerator == unit.Numerator && d.ScaleDenominator == unit.Denominator {
			return unit.Name
		}
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
		left += "/" + strings.Join(denominator, "*")
	}
	if d.ScaleNumerator != 1 || d.ScaleDenominator != 1 {
		return fmt.Sprintf("(%d/%d)*%s", d.ScaleNumerator, d.ScaleDenominator, left)
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
	case "int", "uint", "uint8", "uint16", "uint32", "byte", "uint64", "usize", "isize", "half", "float", "double":
		return t.Kind == TypeBuiltin
	default:
		return false
	}
}

func evt1IntegralRepresentation(t Type) bool {
	_, floating := evt1FloatRepresentationInfo(t)
	return evt1NumericRepresentation(t) && !floating
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
		result, ok := leftDimension.MultiplyChecked(rightDimension)
		if !ok {
			return Type{}, true, evt1Diagnostic("QUANTITY_SCALE_OVERFLOW", "exact unit scale exceeds the bounded compile-time rational range", span)
		}
		if result.IsDimensionless() && !result.Equal(dimensionlessQuantity()) && evt1IntegralRepresentation(left) {
			return Type{}, true, evt1Diagnostic("QUANTITY_SCALE_INTEGRAL", "scaled dimensionless integer result would require fractional conversion; use floating operands", span)
		}
		return evt1QuantityResult(left, result, span), true, nil
	}
	if op == "/" {
		result, ok := leftDimension.DivideChecked(rightDimension)
		if !ok {
			return Type{}, true, evt1Diagnostic("QUANTITY_SCALE_OVERFLOW", "exact unit scale exceeds the bounded compile-time rational range", span)
		}
		if result.IsDimensionless() && !result.Equal(dimensionlessQuantity()) && evt1IntegralRepresentation(left) {
			return Type{}, true, evt1Diagnostic("QUANTITY_SCALE_INTEGRAL", "scaled dimensionless integer result would require fractional conversion; use floating operands", span)
		}
		return evt1QuantityResult(left, result, span), true, nil
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
