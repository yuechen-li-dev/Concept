package concept

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyBoundsUsesSameSourceAndReportsObservation(t *testing.T) {
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("gcc unavailable")
	}
	path := filepath.Join("..", "..", "language", "evt1", "storage", "arrays", "valid", "array_runtime_bounds.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	normal, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), VerifyCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	verifiedAgain, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), VerifyCompilationPolicy())
	if err != nil || !equalOutputs(verified, verifiedAgain) {
		t.Fatalf("verify output is not deterministic: %v", err)
	}
	for run := 2; run <= 100; run++ {
		candidate, err := GenerateForTargetWithPolicy(module, source, GenericC11Target(), VerifyCompilationPolicy())
		if err != nil || !equalOutputs(verified, candidate) {
			t.Fatalf("verify output changed on run %d: %v", run, err)
		}
	}
	name := "array_runtime_bounds.generated.c"
	if strings.Contains(string(normal[name]), "concept_verify_bounds(") || !strings.Contains(string(verified[name]), "concept_verify_bounds(") {
		t.Fatal("verify bounds instrumentation must be present only in Verify C")
	}
	for _, tc := range []struct {
		label   string
		outputs Outputs
		index   int
		want    string
	}{
		{"normal-valid", normal, 2, ""},
		{"verify-valid", verified, 2, ""},
		{"verify-invalid", verified, 4, "index=4 extent=4"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			dir := t.TempDir()
			if err := Write(dir, tc.outputs); err != nil {
				t.Fatal(err)
			}
			harness := filepath.Join(dir, "main.c")
			if err := os.WriteFile(harness, []byte(fmt.Sprintf("#include \"array_runtime_bounds.generated.h\"\nint main(void) { return concept_array_runtime_bounds_read_checked(%d) == 3 ? 0 : 1; }\n", tc.index)), 0644); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(dir, "verify.exe")
			if out, err := nativeCommand(t, compiler, withHostLinkArgs("-std=c11", "-Wall", "-Wextra", "-Werror", "-I", dir, filepath.Join(dir, name), harness, "-o", exe)...).CombinedOutput(); err != nil {
				t.Fatalf("C11 compile: %v\n%s", err, out)
			}
			out, err := nativeCommand(t, exe).CombinedOutput()
			if tc.want == "" && err != nil {
				t.Fatalf("valid program failed: %v\n%s", err, out)
			}
			if tc.want != "" && (err == nil || !strings.Contains(string(out), tc.want) || !strings.Contains(string(out), "compiler-derived bounds violated")) {
				t.Fatalf("missing verification observation: %v\n%s", err, out)
			}
		})
	}
}

func TestVerifyTestRunnerClassifiesBoundsViolation(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("gcc unavailable")
	}
	dir := t.TempDir()
	source := `profile Core;
int Read(int index) {
    int<array>[4] values = [1, 2, 3, 4];
    return values[index];
}
[[fact]]
void CheckBounds() {
    Assert.True(Read(4) == 0, "unreachable");
}`
	if err := os.WriteFile(filepath.Join(dir, "verify_bounds.concept_test"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	manifest, err := DiscoverTests(dir)
	if err != nil {
		t.Fatal(err)
	}
	run, err := RunTests(manifest, TestRunOptions{Verify: true, ResultsDir: filepath.Join(dir, ".test-results")})
	if err != nil {
		t.Fatal(err)
	}
	if !run.Verify || run.Failed != 1 || len(run.Results) != 1 || run.Results[0].Failure == nil || run.Results[0].Failure.Kind != "verification" || !strings.Contains(run.Results[0].PanicReason, "index=4 extent=4") {
		t.Fatalf("test runner did not preserve verification evidence: %+v", run)
	}
}

func TestVerifyDoesNotChangeHardwareOperations(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source []byte
		target TargetCapabilities
	}{
		{"hardware_probe.concept", []byte(mmioSource), GenericC11Target()},
		{"amd64.concept", readVerifyMachineSource(t), X86_64GenericTarget()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module, err := Parse(tc.name, string(tc.source))
			if err != nil {
				t.Fatal(err)
			}
			normal, err := GenerateForTarget(module, tc.source, tc.target)
			if err != nil {
				t.Fatal(err)
			}
			verified, err := GenerateForTargetWithPolicy(module, tc.source, tc.target, VerifyCompilationPolicy())
			if err != nil {
				t.Fatal(err)
			}
			for _, suffix := range []string{".generated.c", ".machine.S"} {
				for path, ordinary := range normal {
					if strings.HasSuffix(path, suffix) && string(ordinary) != string(verified[path]) {
						t.Fatalf("Verify changed hardware artifact %s", path)
					}
				}
			}
		})
	}
}

func readVerifyMachineSource(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join("..", "..", "libraries", "Standard", "Machine", "AMD64.concept"))
	if err != nil {
		t.Fatal(err)
	}
	return contents
}
