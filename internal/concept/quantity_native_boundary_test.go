package concept

import (
	"strings"
	"testing"
)

func TestAssumeQuantityAtNativeScalarBoundary(t *testing.T) {
	valid := `profile Core;
float<m> FromNative(float raw) { return AssumeQuantity<float<m>>(raw); }
float<m/s> SpeedFromNative(float raw) { return AssumeQuantity<float<m/s>>(raw); }
float<m> Main() { return FromNative(12.0); }
`
	module, err := Parse("quantity_boundary.concept", valid)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["quantity_boundary.generated.c"])
	if strings.Contains(generated, "unresolved_call") || !strings.Contains(generated, "return raw;") {
		t.Fatalf("quantity attachment must erase to the original scalar: %s", generated)
	}
	for _, source := range []string{
		`profile Core; float<m> Bad(int raw) { return AssumeQuantity<float<m>>(raw); }`,
		`profile Core; float<m> Bad(float<m> raw) { return AssumeQuantity<float<m>>(raw); }`,
		`profile Core; float Bad(float raw) { return AssumeQuantity<float>(raw); }`,
	} {
		_, err := Parse("quantity_boundary_invalid.concept", source)
		if err == nil || !strings.Contains(err.Error(), "QUANTITY_ATTACHMENT_INVALID") {
			t.Fatalf("expected quantity attachment diagnostic, got %v", err)
		}
	}
}

func TestQuantityLiteralIsDimensionlessForScaleOperations(t *testing.T) {
	const source = `profile Core;
float<m> Half(float<m> length) { return length / 2.0; }
float<m> Double(float<m> length) { return 2.0 * length; }
float<m> Offset(float<m> length) { return length + 2.0; }
float<m^2> Square(float<m> length) { return length * length; }
`
	module, err := Parse("quantity_scale.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}
