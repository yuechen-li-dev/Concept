package concept

import (
	"strings"
	"testing"
)

func TestIntegerMatchContextAndExecution(t *testing.T) {
	source := runtimeTestSource(t, "integer_match.concept_test")
	module, err := Parse("R8g/IntegerMatch.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	mir := string(outputs["integermatch.mir.json"])
	for _, needle := range []string{`"detail": "0"`, `"detail": "_"`, `"detail": "-1"`} {
		if !strings.Contains(mir, needle) {
			t.Fatalf("integer match MIR omitted %s", needle)
		}
	}
}

func TestIntegerMatchDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, source, code string }{
		{"nonexhaustive expression", `profile Core; int F(int x) { return match (x) { 0 => 1, 1 => 2 }; }`, "CV4115"},
		{"nonexhaustive statement", `profile Core; int F(int x) { match (x) { 0 => { return 1; } } return 2; }`, "CV4115"},
		{"duplicate numeric spelling", `profile Core; int F(int x) { return match (x) { 0 => 1, 0x0 => 2, _ => 3 }; }`, "CV4113"},
		{"unreachable after wildcard", `profile Core; int F(int x) { return match (x) { _ => 1, 0 => 2 }; }`, "CV4113"},
		{"literal out of range", `profile Core; int F(uint8 x) { return match (x) { 256 => 1, _ => 2 }; }`, "CV4644"},
		{"enum wildcard still explicit", `profile Core; enum State { Ready, Stopped } int F(State x) { return match (x) { State::Ready => 1, _ => 2 }; }`, "CV4109"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse("r8g_integer_invalid.concept", tc.source)
			if err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("expected %s, got %v", tc.code, err)
			}
		})
	}
}
