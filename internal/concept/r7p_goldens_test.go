package concept

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestR7pDomainGoldensNormalAndVerify(t *testing.T) {
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
		if run.Failed != 0 || run.Passed != len(manifest.Tests) {
			t.Fatalf("verify=%v: %d passed, %d failed: %+v", verify, run.Passed, run.Failed, run.Results)
		}
	}
}
