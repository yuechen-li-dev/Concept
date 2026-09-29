package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The tour under examples/tour is the current-syntax introduction to
// Concept. These tests keep it compiling: the PoC3 examples it replaced
// stopped compiling unnoticed because nothing checked them.

const tourRoot = "../../examples/tour"

func tourSources(t *testing.T) []string {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(tourRoot, "*.concept"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	if len(paths) < 9 {
		t.Fatalf("expected the numbered tour sources, found %d", len(paths))
	}
	return paths
}

func TestTourExamplesGenerateStrictC11(t *testing.T) {
	t.Parallel()
	compiler, lookErr := exec.LookPath("gcc")
	if lookErr != nil {
		compiler, lookErr = exec.LookPath("clang")
	}
	for _, path := range tourSources(t) {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			module, err := Parse(filepath.ToSlash(path), string(source))
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, source)
			if err != nil {
				t.Fatal(err)
			}
			if lookErr != nil {
				return // no C compiler: the Concept-level check above still ran
			}
			dir := t.TempDir()
			if err := Write(dir, outputs); err != nil {
				t.Fatal(err)
			}
			for name := range outputs {
				if !strings.HasSuffix(name, ".generated.c") {
					continue
				}
				args := []string{"-std=c11", "-pedantic-errors", "-fsyntax-only", "-I", dir, filepath.Join(dir, name)}
				if out, err := nativeCommand(t, compiler, args...).CombinedOutput(); err != nil {
					t.Fatalf("%s is not strict C11: %v\n%s", name, err, out)
				}
			}
		})
	}
}

func TestTourTestsPassNormalAndVerify(t *testing.T) {
	t.Parallel()
	for _, dir := range []string{"testing", "modules"} {
		dir := dir
		t.Run(dir, func(t *testing.T) {
			manifest, err := DiscoverTests(filepath.Join(tourRoot, dir))
			if err != nil {
				t.Fatal(err)
			}
			if len(manifest.Tests) == 0 {
				t.Fatalf("no tests discovered in examples/tour/%s", dir)
			}
			for _, verify := range []bool{false, true} {
				run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: filepath.Join(t.TempDir(), "results")})
				if err != nil {
					t.Fatal(err)
				}
				if run.Failed != 0 || run.Passed == 0 {
					t.Fatalf("verify=%v: %d passed, %d failed: %+v", verify, run.Passed, run.Failed, run.Results)
				}
			}
		})
	}
}
