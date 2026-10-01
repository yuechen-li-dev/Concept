package concept

import "fmt"

func evt1IndexInteger(t Type) bool {
	return evt1IntegralRepresentation(t) && t.Quantity == nil
}

func evt1IndexSigned(t Type) bool {
	signed, _, _, _, _ := evt1IntegerTypeRange(t)
	return signed
}

func evt1IndexOutOfBounds(value Value, extent int) bool {
	if value.WideUint {
		return value.UintValue >= uint64(extent)
	}
	return value.IntValue < 0 || value.IntValue >= extent
}

// Compare at full width before an index is used as a pointer offset. The
// extent is already a checked non-negative host extent; unsigned indices do
// not pass through a signed representation or a narrowing size_t conversion.
func evt1IndexGuard(name, extent string, t Type) string {
	comparison := fmt.Sprintf("(uint64_t)%s >= (uint64_t)(%s)", name, extent)
	if evt1IndexSigned(t) {
		return name + " < 0 || " + comparison
	}
	return comparison
}

func evt1IndexCType(t Type) string {
	if t.Name == "int" {
		return "int"
	}
	return evt1CType(t)
}

func evt1IndexVerifyCall(t Type) (string, string) {
	if evt1IndexSigned(t) {
		return "concept_verify_bounds", "int64_t"
	}
	return "concept_verify_unsigned_bounds", "uint64_t"
}
