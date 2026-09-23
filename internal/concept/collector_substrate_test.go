package concept

import (
	"strings"
	"testing"
)

func TestR7f3AppliedTypeFixedArraySyntaxInStrictC11(t *testing.T) {
	source := `profile Core;
template <typename T> struct Box { T value; };
template <usize N>
int CountSlots()
{
    Box<int><array>[N] slots = [Box<int>{1} ...];
    return Len(slots) + slots[0].value;
}
int Main() { return CountSlots<4>(); }
`
	module, err := Parse("collector_applied_array.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "collector_applied_array_harness.c", "#include \"collector_applied_array.generated.h\"\nint main(void) { return concept_collector_applied_array_main() == 5 ? 0 : 1; }\n")
}

func TestR7f3IndexedStorageAuthorityMoveRemainsRejected(t *testing.T) {
	source := `profile Core;
struct SystemMemory {}
void Probe()
{
    int<array>[1] backing = [0];
    Storage<int> storage = bind<int>(AddressOf<SystemMemory>(ref backing), SizeOf<int>());
    Option<Storage<int>><array>[2] slots = [Option::None ...];
    Option<Storage<int>> candidate = Option::Some(move storage);
    slots[0] = move candidate;
}
`
	_, err := Parse("collector_slot_move.concept", source)
	if err == nil || !strings.Contains(err.Error(), "CV4133") {
		t.Fatalf("indexed storage authority replacement = %v, want CV4133", err)
	}
}
