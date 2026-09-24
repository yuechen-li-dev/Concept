package concept

import (
	"strings"
	"testing"
)

func TestClosedGenericRefConstArrayEmitsStorageView(t *testing.T) {
	const source = `module GenericArrayView;
profile Core;
template <typename T, usize N>
int Count(ref const T<array>[N] values) { return Len(values); }
int Main() { int<array>[3] values = [1, 2, 3]; return Count<int, 3>(ref const values); }
`
	module, err := Parse("GenericArrayView.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for name, artifact := range outputs {
		if strings.HasSuffix(name, ".generated.h") && !strings.Contains(string(artifact), "concept_ref_const_array_1_int") {
			t.Fatalf("closed generic array view missing from C header: %s", name)
		}
	}
	runFoundationNativeHarness(t, outputs, "generic_array_view_harness.c", "#include \"GenericArrayView.generated.h\"\nint main(void) { return concept_generic_array_view_main() == 3 ? 0 : 1; }\n")
}
