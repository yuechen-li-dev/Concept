package concept

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestR8gAssertionContextDoubleQuantityAndExactError(t *testing.T) {
	dir := t.TempDir()
	const source = `module R8g.TestSurface; profile Core;
enum Problem { Missing, Full }
Result<int, Problem> Fail() { return Result::Error(Problem::Missing); }
[[fact]] void ContextualEquals() {
    uint32 value = 0;
    Assert.Equals(value, 0, "right literal takes uint32 context");
    Assert.Equals(0, value, "left literal takes uint32 context");
}
[[fact]] void TypedNear() {
    double actual = 1.000000000001;
    Assert.Near(actual, 1.0, 0.000000000002, "double precision is preserved");
    double<m> distance = interpret (1.001 as double) as double<m>;
    double<m> expected = interpret (1.0 as double) as double<m>;
    double<mm> tolerance = interpret (2.0 as double) as double<mm>;
    Assert.Near(distance, expected, tolerance, "tolerance scales within one dimension");
}
[[fact]] void SpecificError() {
    Assert.FailsWith(Fail(), Problem::Missing, "exact error variant");
}`
	path := filepath.Join(dir, "surface.concept_test")
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
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
		if run.Passed != 3 || run.Failed != 0 {
			t.Fatalf("Verify=%v: %+v", verify, run)
		}
	}
}

func TestR8gAssertionRejectsWrongErrorAndDimension(t *testing.T) {
	for _, tc := range []struct{ source, code string }{
		{`profile Core; enum A { X } enum B { X } Result<int,A> F() { return Result::Error(A::X); } [[fact]] void Test() { Assert.FailsWith(F(), B::X, "wrong error type"); }`, "TEST_ASSERT_EQUALS_TYPE_MISMATCH"},
		{`profile Core; [[fact]] void Test() { double<m> x = interpret (1.0 as double) as double<m>; double<s> tolerance = interpret (1.0 as double) as double<s>; Assert.Near(x, x, tolerance, "wrong dimension"); }`, "TEST_ASSERT_NEAR_NUMERIC_REQUIRED"},
	} {
		_, err := Parse("r8g_invalid_assert.concept_test", tc.source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("expected %s, got %v", tc.code, err)
		}
	}
}
