package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestR8gBoundedExhaustionBoundary(t *testing.T) {
	const source = `module R8g.Bounded; profile Core;
int Zero(int target) {
    int i = 0;
    while (i < target) bounded(0) { i += 1; } else { return 99; }
    return i;
}
int One(int target) {
    int i = 0;
    while (i < target) bounded(1) { i += 1; } else { return 99; }
    return i;
}
int Two(int target) {
    int i = 0;
    while (i < target) bounded(2) { i += 1; } else { return 99; }
    return i;
}
int Unhandled(int target) {
    int i = 0;
    while (i < target) bounded(1) { i += 1; }
    return i;
}
`
	module, err := Parse("R8g/Bounded.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	for _, policy := range []CompilationPolicy{ConservativeCompilationPolicy(), VerifyCompilationPolicy()} {
		outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), policy)
		if err != nil {
			t.Fatal(err)
		}
		assertR8cStrictC11(t, outputs, "bounded.generated.c")
		runFoundationNativeHarness(t, outputs, "bounded_harness.c", `#include "bounded.generated.h"
int main(void) {
  if (concept_r8g__bounded_zero(0) != 0) return 1;
  if (concept_r8g__bounded_zero(1) != 99) return 2;
  if (concept_r8g__bounded_one(0) != 0) return 3;
  if (concept_r8g__bounded_one(1) != 1) return 4;
  if (concept_r8g__bounded_one(2) != 99) return 5;
  if (concept_r8g__bounded_two(2) != 2) return 6;
  if (concept_r8g__bounded_two(3) != 99) return 7;
  if (concept_r8g__bounded_unhandled(0) != 0) return 8;
  if (concept_r8g__bounded_unhandled(1) != 1) return 9;
  return 0;
}`)
		body := string(outputs["bounded.generated.c"])
		if strings.Contains(body, "bounded while exhausted") != policy.Verify {
			t.Fatalf("Verify=%v exhaustion diagnostic mismatch", policy.Verify)
		}
	}
}

func TestR8gVerifyReportsUnhandledBoundedExhaustion(t *testing.T) {
	dir := t.TempDir()
	const source = `module R8g.Exhausted; profile Core;
[[fact]] void Exceeds() {
    int i = 0;
    while (i < 2) bounded(1) { i += 1; }
    Assert.Equals(i, 1, "normal truncation remains documented");
}`
	if err := os.WriteFile(filepath.Join(dir, "exhausted.concept_test"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	manifest, err := DiscoverTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, verify := range []bool{false, true} {
		run, err := RunTests(manifest, TestRunOptions{Verify: verify, ResultsDir: filepath.Join(dir, "results")})
		if err != nil {
			t.Fatal(err)
		}
		if verify {
			if run.Failed != 1 || len(run.Results) != 1 || !strings.Contains(run.Results[0].Stderr, "bounded while exhausted") {
				t.Fatalf("Verify did not report exhaustion: %+v", run)
			}
		} else if run.Passed != 1 || run.Failed != 0 {
			t.Fatalf("Normal bounded behavior changed: %+v", run)
		}
	}
}

func TestR8gAsyncBoundedExhaustionHandler(t *testing.T) {
	const source = `module R8g.AsyncBounded; profile Core;
async int Tick(int n) { return n; }
async int Count(int target) {
    int i = 0;
    while (i < target) bounded(1) {
        i += await Tick(1);
    } else { return 99; }
    return i;
}
int Run(int target) {
    Async<int> task = Count(target);
    while (not Complete(task)) bounded(4) { Step(task); }
    return Result(task);
}
`
	module, err := Parse("R8g/AsyncBounded.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "asyncbounded.generated.c")
	runFoundationNativeHarness(t, outputs, "asyncbounded_harness.c", `#include "asyncbounded.generated.h"
int main(void) {
  if (concept_r8g__async_bounded_run(0) != 0) return 1;
  if (concept_r8g__async_bounded_run(1) != 1) return 2;
  if (concept_r8g__async_bounded_run(2) != 99) return 3;
  return 0;
}`)
}
