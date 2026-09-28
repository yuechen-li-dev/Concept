package concept

import (
	"strings"
	"testing"
)

func TestRawInlineInitializedPrefixNative(t *testing.T) {
	source := `module RawPrefix;
profile Core;
struct SystemMemory {}
struct Item { int value; }
int Main()
{
    Item<raw>[4] backing = Uninitialized();
    int first = RawAppend(ref backing, Item{7});
    usize addressBefore = AddressBits(AddressOf<SystemMemory>(RawGet(ref backing, first)));
    int second = RawAppend(ref backing, Item{9});
    usize addressAfter = AddressBits(AddressOf<SystemMemory>(RawGet(ref backing, first)));
    Span<Item> values = RawValues(ref backing);
    if (first != 0 or second != 1 or addressBefore != addressAfter or
        RawCount(ref const backing) != 2 or Len(values) != 2)
    {
        return 1;
    }
    return values[0].value + values[1].value;
}`
	module, err := Parse("rawprefix.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["rawprefix.generated.c"])
	if strings.Contains(body, "malloc(") || strings.Contains(body, "realloc(") {
		t.Fatal("raw fixed prefix allocated")
	}
	runFoundationNativeHarness(t, outputs, "rawprefix_harness.c", "#include \"rawprefix.generated.h\"\nint main(void) { return concept_raw_prefix_main() == 16 ? 0 : 1; }\n")
}

func TestRawInlineImmovableFailureAndDrop(t *testing.T) {
	source := `module RawPinned;
profile Core;
extern "C" void ObserveDrop(int value);
extern "C" void Checkpoint(int stage);
enum BuildError { Failed }
struct Part { int value; }
void Drop(owned Part part) { ObserveDrop(part.value); }
immovable struct Pinned { owned Part part; int key; }
Result<int, BuildError> Key(bool fail)
{
    if (fail) { return Result::Error(BuildError::Failed); }
    return Result::Ok(9);
}

Result<int, BuildError> Build(bool fail)
{
    Pinned<raw>[4] backing = Uninitialized();
    int index = RawAppend(ref backing, Pinned{Part{7}, Key(fail)?});
    ReadOnlySpan<Pinned> values = RawValues(ref const backing);
    if (index != 0 or Len(values) != 1) { return Result::Ok(1); }
    return Result::Ok(values[0].key);
}
int Main()
{
    discard Build(true);
    Checkpoint(1);
    discard Build(false);
    Checkpoint(2);
    return 0;
}`
	module, err := Parse("rawpinned.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "rawpinned.generated.h"
#include <stdio.h>
static int drops = 0;
static int bad = 0;
void ObserveDrop(int value) { drops += 1; if (value != 7) bad = 1; }
void Checkpoint(int stage) { if (drops != stage) bad = 1; }
int main(void) { int result = concept_raw_pinned_main(); if (result != 0 || drops != 2 || bad) printf("result=%d drops=%d bad=%d\n", result, drops, bad); return result == 0 && drops == 2 && !bad ? 0 : 1; }
`
	runFoundationNativeHarness(t, outputs, "rawpinned_harness.c", harness)
}

func TestPartialInlineStorageRejectsZeroCapacity(t *testing.T) {
	for _, marker := range []string{"raw", "sparse"} {
		source := "profile Core; struct Cell { int value; } void Probe() { Cell<" + marker + ">[0] slots = Uninitialized(); }"
		if _, err := Parse("zero_storage.concept", source); err == nil || !strings.Contains(err.Error(), "RAW_STORAGE_SHAPE") {
			t.Fatalf("%s zero capacity = %v, want RAW_STORAGE_SHAPE", marker, err)
		}
	}
}
