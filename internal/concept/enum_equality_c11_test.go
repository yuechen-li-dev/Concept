package concept

import (
	"strings"
	"testing"
)

func TestPayloadEnumEqualityRequiresExplicitOperation(t *testing.T) {
	const source = `profile Core;
enum Choice { Value(int value), }
bool Equal(Choice left, Choice right) { return left == right; }
`
	_, err := Parse("payload_enum_equality.concept", source)
	if err == nil || !strings.Contains(err.Error(), "CV4830") {
		t.Fatalf("payload enum equality silently reached invalid C11: %v", err)
	}
}
