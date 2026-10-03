package concept

import (
	"os"
	"path/filepath"
	"testing"
)

func runtimeTestSource(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", "tests", "runtime", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// Runtime expectations live in Concept. Go retains the compiler, formatter,
// MIR and independent C/native differential oracles that cannot test themselves.
func TestRuntimeConceptSpecifications(t *testing.T) {
	manifest, err := DiscoverTests("../../tests/runtime")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tests) != 6 {
		t.Fatalf("runtime specification count changed: %d", len(manifest.Tests))
	}
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: t.TempDir()})
			if err != nil || run.Passed != 6 || run.Failed != 0 {
				t.Fatalf("runtime specifications: %v %+v", err, run)
			}
		})
	}
}
