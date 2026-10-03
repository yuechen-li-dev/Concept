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

func TestEVT2e5CLIContractAndEmission(t *testing.T) {
	fixture := "../../internal/concept/testdata/evt2e_calls.concept"
	for _, command := range []string{"lir", "machineir", "amd64"} {
		cmd := exec.Command("go", "run", ".", command, fixture)
		output, err := cmd.CombinedOutput()
		if command == "amd64" {
			if err != nil || !strings.Contains(string(output), "native image bytes=") {
				t.Fatalf("native module emission failed: %v\n%s", err, output)
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

func TestEVT2e5CLIDebugPlans(t *testing.T) {
	cases := []struct {
		name, source string
		wants        []string
	}{
		{"zero", "int Target(){return 7;} int Caller(){return Target();}", []string{"call @Target|Target()", "shadow=32 stack-bytes=0", "return RAX ->"}},
		{"four", "int Target(int a,int b,int c,int d){return a;} int Caller(int a,int b,int c,int d){return Target(a,b,c,d);}", []string{"arg0 v0:i32 width=4 -> RCX", "arg3 v3:i32 width=4 -> R9", "stack-bytes=0"}},
		{"five", "int Caller(int a,int b,int c,int d,int e){return Target(a,b,c,d,e);} int Target(int a,int b,int c,int d,int e){return e;}", []string{"arg4 v4:i32 width=4 -> stack[0]", "stack-bytes=8", "move R9 -> stack[0]"}},
		{"live", "int Target(int a){return a;} int Caller(int a,int b,int c){int local=a+b;return local+Target(c);}", []string{"live v", "MustPreserve", "range="}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.name+".concept")
			if err := os.WriteFile(path, []byte("profile Core;\n"+tc.source), 0644); err != nil {
				t.Fatal(err)
			}
			output, err := exec.Command("go", "run", ".", "amd64", path).CombinedOutput()
			if err != nil || !strings.Contains(string(output), "native image bytes=") {
				t.Fatalf("native module emission failed: %v\n%s", err, output)
			}
			for _, want := range tc.wants {
				if !strings.Contains(string(output), want) {
					t.Fatalf("missing %q:\n%s", want, output)
				}
			}
			for _, want := range []string{"frame size=", "base=post-prologue-RSP", "region shadow [rsp+0 .. +32)", "prologue", "epilogue (each return)", "pre-call", "abstract-call @", "post-call"} {
				if !strings.Contains(string(output), want) {
					t.Fatalf("missing concrete frame/action output %q:\n%s", want, output)
				}
			}
			t.Logf("%s", output)
		})
	}
}
