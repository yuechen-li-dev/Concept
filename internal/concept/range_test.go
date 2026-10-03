package concept

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestRangeCountedLoopCodeGeneration(t *testing.T) {
	source := runtimeTestSource(t, "range_counted.concept_test")
	module, err := Parse("range_counted.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["range_counted.generated.c"])
	if strings.Contains(generated, "malloc(") || strings.Contains(generated, "MoveNext(") || !strings.Contains(generated, "inline counted iterator") {
		t.Fatal("static range did not lower to an inline counted loop")
	}
}

func TestRangeInvalid(t *testing.T) {
	for _, source := range []string{
		"profile Core; int Main() { for (i in 0..10 step 0) { } return 0; }",
		"profile Core; int Main() { for (i in 10..0) { } return 0; }",
		"profile Core; int Main() { for (i in 0..10 descend 1) { } return 0; }",
		"profile Core; int Main() { for (i in 0..10 step 2 descend 1) { } return 0; }",
	} {
		if _, err := Parse("invalid_range.concept", source); err == nil {
			t.Fatalf("accepted invalid range: %s", source)
		}
	}
}

func TestRangeMIRComparisonAndDeterminism(t *testing.T) {
	source := `profile Core;
int Main()
{
    int total = 0;
    for (i in 0..10) { total += i; }
    int cursor = 0;
    while (cursor < 10) bounded(10) { total += cursor; cursor++; }
    return total;
}`
	module, err := Parse("range_compare.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	first, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(first["range_compare.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range mir.Functions {
		for _, each := range fn.Foreaches {
			if each.SourceKind == "range" && each.IteratorStrategy == "BuiltinInlineIterator" && each.NoAllocation {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("range did not use the ordinary allocation-free foreach MIR")
	}
	c := string(first["range_compare.generated.c"])
	if strings.Count(c, "while (") < 2 || strings.Contains(c, "malloc(") || strings.Contains(c, "MoveNext(") {
		t.Fatal("range and bounded while did not lower to inline C loops")
	}
	runFoundationNativeHarness(t, first, "range_compare_harness.c", "#include \"range_compare.generated.h\"\nint main(void) { return concept_range_compare_main() == 90 ? 0 : 1; }\n")
	for i := 0; i < determinismRuns(); i++ {
		again, err := Generate(module, []byte(source))
		if err != nil || !reflect.DeepEqual(first, again) {
			t.Fatalf("range generation drift at run %d: %v", i+1, err)
		}
	}
}
