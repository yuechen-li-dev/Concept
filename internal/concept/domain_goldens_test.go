package concept

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDomainGoldensNormalAndVerify(t *testing.T) {
	root := filepath.Join("..", "..", "libraries", "Golden")
	manifest, err := DiscoverTests(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"Aerospace", "Cad", "Compiler", "Embedded", "Game", "Hft", "Hpc", "Storage"} {
		found := false
		for _, test := range manifest.Tests {
			if strings.HasPrefix(test.TestID, domain+"/") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing permanent %s golden facts", domain)
		}
	}
	for _, verify := range []bool{false, true} {
		run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: filepath.Join(t.TempDir(), "results")})
		if err != nil {
			t.Fatal(err)
		}
		// A theory contributes one result per data row, so compare test
		// identities rather than raw counts: every discovered test must have
		// produced results, and none may have failed.
		reported := map[string]bool{}
		for _, result := range run.Results {
			id, _, _ := strings.Cut(result.TestID, "[") // theory rows report as ID[row]
			reported[id] = true
		}
		missing := []string{}
		for _, test := range manifest.Tests {
			if !reported[test.TestID] {
				missing = append(missing, test.TestID)
			}
		}
		if run.Failed != 0 || len(missing) != 0 {
			t.Fatalf("verify=%v: %d passed, %d benchmarks, %d failed, missing %v: %+v", verify, run.Passed, run.Benchmarks, run.Failed, missing, run.Results)
		}
	}
}
