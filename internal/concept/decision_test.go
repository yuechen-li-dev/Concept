package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestValueDecideNativeC11(t *testing.T) {
	path := filepath.Join("..", "..", "language", "evt1", "decision", "valid", "decide_value_int.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, outputs, "decide_value_int.generated.c")
	runFoundationNativeHarness(t, outputs, "decision_harness.c", `#include "decide_value_int.generated.h"
int main(void) { return concept_decide_value_int_main() == 2 ? 0 : 1; }
`)
	generated := string(outputs["decide_value_int.generated.c"])
	if !strings.Contains(generated, "decision_best_score") || !strings.Contains(generated, "decision has no enabled candidates") {
		t.Fatal("value decision did not lower to direct guarded hardmax")
	}
	floatPath := filepath.Join("..", "..", "language", "evt1", "decision", "valid", "decide_value_float.concept")
	floatSource, err := os.ReadFile(floatPath)
	if err != nil {
		t.Fatal(err)
	}
	floatModule, err := Parse(floatPath, string(floatSource))
	if err != nil {
		t.Fatal(err)
	}
	floatOutputs, err := Generate(floatModule, floatSource)
	if err != nil {
		t.Fatal(err)
	}
	assertR8cStrictC11(t, floatOutputs, "decide_value_float.generated.c")
	runFoundationNativeHarness(t, floatOutputs, "decision_float_harness.c", `#include "decide_value_float.generated.h"
int main(void) { return concept_decide_value_float_main() == 1 ? 0 : 1; }
`)
}

func TestValueDecideDiagnosticsAndNoEnabled(t *testing.T) {
	for file, want := range map[string]string{
		"decide_empty.concept":             "DECIDE_EMPTY",
		"decide_unknown_candidate.concept": "DECIDE_UNKNOWN_CANDIDATE",
		"decide_mixed_scores.concept":      "DECIDE_SCORE_TYPE_MISMATCH",
		"decide_bad_guard.concept":         "DECIDE_GUARD_REQUIRES_BOOL",
	} {
		path := filepath.Join("..", "..", "language", "evt1", "decision", "invalid", file)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Parse(path, string(source))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: expected %s, got %v", file, want, err)
		}
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		t.Skip("C11 compiler unavailable")
	}
	path := filepath.Join("..", "..", "language", "evt1", "decision", "invalid", "decide_no_enabled_runtime.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(path, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	harness := `#include "decide_no_enabled_runtime.generated.h"
int main(void) { (void)concept_decide_no_enabled_runtime_bad(); return 0; }
`
	harnessPath := filepath.Join(dir, "decision_panic_harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness), 0644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(dir, "decision-panic.exe")
	generated := filepath.Join(dir, "decide_no_enabled_runtime.generated.c")
	if out, err := nativeCommand(t, compiler, withHostLinkArgs("-std=c11", "-Wall", "-Wextra", "-I", dir, generated, harnessPath, "-o", exe)...).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	out, err := nativeCommand(t, exe).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "decision has no enabled candidates") {
		t.Fatalf("missing no-enabled panic: err=%v output=%s", err, out)
	}
}
