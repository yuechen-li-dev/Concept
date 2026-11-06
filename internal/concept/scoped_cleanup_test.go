package concept

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestScopedOwnershipCleanupIdentityAndNativeC11(t *testing.T) {
	path := "../../tests/runtime/scoped_cleanup.concept_test"
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var first Outputs
	for sample := 0; sample < 3; sample++ {
		module, err := Parse(path, string(source))
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, source)
		if err != nil {
			t.Fatal(err)
		}
		if sample == 0 {
			first = outputs
		} else {
			for name, expected := range first {
				if !bytes.Equal(expected, outputs[name]) {
					t.Fatalf("fresh scoped cleanup generation differs at sample %d: %s", sample, name)
				}
			}
		}
	}
	var mir MIR
	if err := json.Unmarshal(first["scoped_cleanup.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fn := range mir.Functions {
		seen := map[string]bool{}
		for _, cleanup := range fn.Cleanups {
			if seen[cleanup.Owner] {
				t.Fatalf("duplicate cleanup identity in %s: %s", fn.Name, cleanup.Owner)
			}
			seen[cleanup.Owner] = true
			if cleanup.SourceName == "value" && !cleanupOwner(fn.Cleanups, "value") {
				t.Fatal("planner lost the source-name connection for replacement")
			}
		}
	}
	assertR8cStrictC11(t, first, "scoped_cleanup.generated.c")
	const harness = `#include "scoped_cleanup.generated.h"
int main(void) {
    concept_runtime__scoped_cleanup_cleanup_uses_lexical_binding_identity();
    return 0;
}
`
	runFoundationNativeHarness(t, first, "scoped_cleanup_harness.c", harness)
}
