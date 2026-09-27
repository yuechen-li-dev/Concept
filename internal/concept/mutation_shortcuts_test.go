package concept

import (
	"strings"
	"testing"
)

func TestR7x2MutationShortcutsNativeC11(t *testing.T) {
	source := `profile Core;

int Next(ref int calls)
{
    calls = calls + 1;
    return 0;
}

int Main()
{
    int n = 10;
    n++;
    ++n;
    n--;
    --n;
    n += 5;
    n -= 3;
    n *= 2;
    n /= 3;
    n %= 3;
    n &= 3;
    n |= 4;
    n ^= 7;
    n <<= 2;
    n >>= 1;
    int euclidean = -5;
    euclidean %= 3;
    int calls = 0;
    int<array>[1] values = [4];
    values[Next(ref calls)] += n;
    return n * 10 + values[0] + calls + euclidean * 100;
}`
	module, err := Parse("mutation_shortcuts.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "mutation_shortcuts_harness.c", "#include \"mutation_shortcuts.generated.h\"\nint main(void) { return concept_mutation_shortcuts_main() == 127 ? 0 : 1; }\n")
	generated := string(outputs["mutation_shortcuts.generated.c"])
	if strings.Count(generated, "concept_mutation_shortcuts_next(&(calls))") != 1 {
		t.Fatal("compound indexed place did not evaluate its index exactly once")
	}
}

func TestR7x2MutationShortcutsRejectInvalid(t *testing.T) {
	for _, source := range []string{
		"profile Core; int Main() { const int n = 1; n++; return n; }",
		"profile Core; int Main() { int n = 1; int value = n++; return value; }",
		"profile Core; int Main() { int n = 1; int value = ++n; return value; }",
		"profile Core; int Main() { bool flag = true; flag += 1; return 0; }",
	} {
		if _, err := Parse("invalid_mutation.concept", source); err == nil {
			t.Fatalf("accepted invalid mutation: %s", source)
		}
	}
}
