package concept

import (
	"strings"
	"testing"
)

func TestTagOnlyEnumEqualityLowersThroughC11(t *testing.T) {
	const source = `module EnumEquality; profile Core;
enum Status { Ready, Done, }
int Main() {
    Status first = Status::Ready;
    Status second = Status::Done;
    if (first == second) { return 1; }
    if (first != Status::Ready) { return 2; }
    return 0;
}
`
	module, err := Parse("EnumEquality.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "enum_equality_harness.c", "#include \"enumequality.generated.h\"\nint main(void) { return concept_enum_equality_main(); }\n")
}

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
