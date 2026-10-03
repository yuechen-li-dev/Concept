package main

import (
	"os/exec"
	"strings"
	"testing"
)

// The CLI must compile the imported backend library from source, not just
// transport MachineIR successfully in the internal native harness.
func TestAMD64BootstrapLoadsBackendImports(t *testing.T) {
	if _, err := exec.LookPath("gcc"); err != nil {
		if _, err := exec.LookPath("clang"); err != nil {
			t.Skip("C11 compiler unavailable")
		}
	}
	cmd := exec.Command("go", "run", ".", "amd64", "../../language/evt2/machine/valid/finite.concept")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Stage-0 -> C -> native AMD64 CLI: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "fn Counter.Run$step") ||
		!strings.Contains(string(output), "  0000: ") {
		t.Fatalf("missing emitted native bytes:\n%s", output)
	}
}
