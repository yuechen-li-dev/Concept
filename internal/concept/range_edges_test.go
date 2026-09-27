package concept

import "testing"

func TestR7x2RangeSignedUnsignedOverflowEdgesNativeC11(t *testing.T) {
	source := `profile Core;
int Main()
{
    int count = 0;
    int high = 2147483647;
    for (i in 2147483646..high step 2147483647) { count++; }
    int low = -2147483647;
    for (i in 2147483647..low descend 2147483647) { count++; }
    uint64 end = 18446744073709551615;
    uint64 stride = 18446744073709551614;
    for (i in 0..end step stride) { count++; }
    for (i in end..0 descend stride) { count++; }
    return count;
}`
	module, err := Parse("range_edges.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "range_edges_harness.c", "#include \"range_edges.generated.h\"\nint main(void) { return concept_range_edges_main() == 7 ? 0 : 1; }\n")
}
