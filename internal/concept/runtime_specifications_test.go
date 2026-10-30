package concept

import (
	"os"
	"path/filepath"
	"strings"
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
	if len(manifest.Tests) == 0 {
		t.Fatal("no runtime specifications discovered")
	}
	for _, test := range manifest.Tests {
		for _, dependency := range test.module.ImportedModuleIdentities {
			if strings.HasPrefix(dependency.ModuleIdentity, "Standard.Backend.") {
				t.Fatalf("quick runtime fact %s imports Cathedral (%s); put backend qualification in tests/backend", test.TestID, dependency.ModuleIdentity)
			}
		}
	}
	runConceptSpecifications(t, manifest)
}

func TestBackendConceptSpecifications(t *testing.T) {
	manifest, err := DiscoverTestsWithModuleRoots("../../tests/backend", []string{"../../libraries"})
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Tests) == 0 {
		t.Fatal("no backend component facts discovered")
	}
	for _, test := range manifest.Tests {
		for _, dependency := range test.module.ImportedModuleIdentities {
			if strings.HasPrefix(dependency.ModuleIdentity, "Standard.Backend.AMD64") {
				t.Fatalf("%s loads the full Cathedral backend; keep component facts in leaf modules and use the Qualification lane for whole-backend checks", test.TestID)
			}
		}
	}
	runConceptSpecifications(t, manifest)
}

func runConceptSpecifications(t *testing.T, manifest TestManifest) {
	t.Helper()
	for _, verify := range []bool{false, true} {
		mode := "Normal"
		if verify {
			mode = "Verify"
		}
		t.Run(mode, func(t *testing.T) {
			run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: t.TempDir()})
			if err != nil || run.Failed != 0 || len(run.Results) == 0 {
				t.Fatalf("runtime specifications: %v %+v", err, run)
			}
			reported := make(map[string]bool)
			for _, result := range run.Results {
				reported[result.TestID] = true
			}
			for _, test := range manifest.Tests {
				if !reported[test.TestID] {
					t.Errorf("no result for discovered specification %s", test.TestID)
				}
			}
		})
	}
}
