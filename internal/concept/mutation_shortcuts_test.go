package concept

import (
	"strings"
	"testing"
)

func TestMutationShortcutsIndexCodeGeneration(t *testing.T) {
	source := runtimeTestSource(t, "mutation_shortcuts.concept_test")
	module, err := Parse("mutation_shortcuts.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["mutation_shortcuts.generated.c"])
	if strings.Count(generated, "concept_mutation_shortcuts_next(&(calls))") != 1 {
		t.Fatal("compound indexed place did not evaluate its index exactly once")
	}
}

func TestMutationShortcutsRejectInvalid(t *testing.T) {
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
