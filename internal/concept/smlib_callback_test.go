package concept

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSMLibCallbackBindingAndAdapters(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
	}{
		{"smlib_callback_local_record", 0}, {"smlib_callback_repeated_erasure", 0},
		{"smlib_callback_erasure_unwrap", 2}, {"smlib_callback_identity", 121},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join("..", "..", "language", "evt1", "callable", "capture", "valid", tc.name+".concept")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
				module, err := Parse(tc.name+".concept", string(body))
				if err != nil {
					t.Fatal(err)
				}
				outputs, err := GenerateForTargetWithPolicy(module, body, GenericC11Target(), policy)
				if err != nil {
					t.Fatal(err)
				}
				again, err := GenerateForTargetWithPolicy(module, body, GenericC11Target(), policy)
				if err != nil {
					t.Fatal(err)
				}
				for name, data := range outputs {
					if !bytes.Equal(data, again[name]) {
						t.Fatalf("nondeterministic %s", name)
					}
				}
				assertR8cStrictC11(t, outputs, tc.name+".generated.c")
				harness := fmt.Sprintf("#include \"%s.generated.h\"\nint main(void) { return concept_%s_main() == %d ? 0 : 1; }\n", tc.name, tc.name, tc.want)
				runFoundationNativeHarness(t, outputs, "smlib_callback_harness.c", harness)
			}
		})
	}
}
func TestSMLibCallbackInferenceRetainsSafety(t *testing.T) {
	for _, source := range []string{
		`int Main() { const auto c=callback(bool choose) { if(choose) { int x=1; return x; } else { return true; } }; return 0; }`,
		`int Main() { const auto c=callback() { return missing; }; return 0; }`,
		`int Main() { const auto c=callback() { int x=1; return ref x; }; return 0; }`,
		`int Main() { int x=1; const auto c=callback() { int y=x; return y; }; return 0; }`,
	} {
		if _, err := Parse("smlib_callback_invalid.concept", "profile Core; "+source); err == nil {
			t.Fatal("unsafe callback accepted: " + source)
		}
	}
	// Distinct source identities, even with identical signatures, need distinct adapters.
	path := filepath.Join("..", "..", "language", "evt1", "callable", "capture", "valid", "smlib_callback_identity.concept")
	body, _ := os.ReadFile(path)
	module, err := Parse("smlib_callback_identity.concept", string(body))
	if err != nil {
		t.Fatal(err)
	}
	witnesses := evt1ModuleCallbackWitnesses(module)
	if len(witnesses) != 3 {
		t.Fatalf("want 3 distinct callback identities, got %+v", witnesses)
	}
	outputs, err := Generate(module, body)
	if err != nil {
		t.Fatal(err)
	}
	c := string(outputs["smlib_callback_identity.generated.c"])
	for _, w := range witnesses {
		if strings.Count(c, "static int32_t "+w.Adapter+"(") != 1 {
			t.Fatalf("adapter must be emitted once: %s", w.Adapter)
		}
	}
}
