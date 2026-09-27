package concept

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestR7x2RangeValueAndCountedLoopNativeC11(t *testing.T) {
	source := `profile Core;

int Sum(Range<int> values)
{
    Assert.Concept<Bounded>(values, "finite range has a bound");
    int result = 0;
    for (value in values) { result += value; }
    return result;
}

int Main()
{
    Range<int> plain = 0..10;
    int result = Sum(plain);
    for (i in 0..10 step 2) { result += i; }
    for (i in 10..0 descend 2) { result += i; }
    for (i in 3..3) { result += 1000; }
    int top = 2147483647;
    for (i in 2147483646..top) { result += i - 2147483646; }
    return result;
}`
	module, err := Parse("range_counted.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "range_counted_harness.c", "#include \"range_counted.generated.h\"\nint main(void) { return concept_range_counted_main() == 95 ? 0 : 1; }\n")
	generated := string(outputs["range_counted.generated.c"])
	if strings.Contains(generated, "malloc(") || strings.Contains(generated, "MoveNext(") || !strings.Contains(generated, "inline counted iterator") {
		t.Fatal("static range did not lower to an inline counted loop")
	}
}

func TestR7x2RangeInvalid(t *testing.T) {
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

func TestR7x2RangeAsyncNativeC11(t *testing.T) {
	source := `profile Core;
async int Child(int value) { return value; }
async int Sum()
{
    int total = 0;
    for (i in 6..0 descend 2) {
        int observed = await Child(i);
        total += observed;
    }
    return total;
}
int Main()
{
    Async<int> work = Sum();
    while (not Complete(work)) bounded(16) { Step(work); }
    return Result(work);
}`
	module, err := Parse("range_async.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "range_async_harness.c", "#include \"range_async.generated.h\"\nint main(void) { return concept_range_async_main() == 12 ? 0 : 1; }\n")
}

func TestR7x2RangeMachineNativeC11(t *testing.T) {
	source := `profile Core;
automata Counter with state { int total; }
{
    machine MainMachine
    {
        state Start
        {
            for (i in 1..5) { state.total += i; }
            complete;
        }
    }
}
int Main()
{
    instance Counter counter(0);
    Step(counter, MainMachine);
    return counter.state.total;
}`
	module, err := Parse("range_machine.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "range_machine_harness.c", "#include \"range_machine.generated.h\"\nint main(void) { return concept_range_machine_main() == 10 ? 0 : 1; }\n")
}

func TestR7x2RangeMIRComparisonAndDeterminism(t *testing.T) {
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
