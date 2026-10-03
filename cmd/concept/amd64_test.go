package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuechen-li-dev/Concept/internal/concept"
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

func TestEVT2e2CLIContractAndBoundary(t *testing.T) {
	fixture := "../../internal/concept/testdata/evt2e_calls.concept"
	for _, command := range []string{"lir", "machineir", "amd64"} {
		cmd := exec.Command("go", "run", ".", command, fixture)
		output, err := cmd.CombinedOutput()
		if command == "amd64" {
			if err == nil || !strings.Contains(string(output), "AMD64_UNSUPPORTED_CALL_LOWERING") {
				t.Fatalf("imprecise later backend boundary: %v\n%s", err, output)
			}
		} else if err != nil || !strings.Contains(string(output), "call @") && !strings.Contains(string(output), "call i32 @") {
			t.Fatalf("%s call CLI: %v\n%s", command, err, output)
		}
	}
	artifacts, err := concept.BuildSemanticModuleArtifactsFromSources([]string{"../../libraries"}, []string{"Standard.Backend.BridgeDerive"})
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for name, data := range artifacts {
		path := filepath.Join(root, filepath.FromSlash(strings.ReplaceAll(name, ".", "/")+".concept-module.json"))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", ".", "generated", "../../libraries/Standard/Backend/BridgeCodec.concept")
	cmd.Env = append(os.Environ(), "CONCEPT_MODULE_ROOTS="+root)
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "WireMachineCall") {
		t.Fatalf("generated call declarations CLI: %v\n%s", err, output)
	}
}
