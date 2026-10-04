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

func TestEVT2f2CLIContractAndBoundary(t *testing.T) {
	fixture := "../../language/evt2f/valid/scalars.concept"
	for _, command := range []string{"lir", "machineir", "amd64"} {
		output, err := exec.Command("go", "run", ".", command, fixture).CombinedOutput()
		text := string(output)
		if command == "amd64" {
			if err == nil || !strings.Contains(text, "AMD64_UNSUPPORTED_SSE_ENCODING") || strings.Contains(text, "native image bytes=") {
				t.Fatalf("native boundary: %v\n%s", err, text)
			}
		} else {
			if err != nil || !strings.Contains(text, "fadd float32") || !strings.Contains(text, "0x3ff0000000000001") {
				t.Fatalf("%s: %v\n%s", command, err, text)
			}
			if command == "machineir" && (!strings.Contains(text, "CMIRAMD4 v4") || !strings.Contains(text, "float32 / XMM")) {
				t.Fatal(text)
			}
		}
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

func TestEVT2f3CLIPlanningBoundary(t *testing.T) {
	output, err := exec.Command("go", "run", ".", "amd64", "../../language/evt2f/valid/planning.concept").CombinedOutput()
	text := strings.ReplaceAll(string(output), "\r\n", "\n")
	if err == nil || !strings.Contains(text, "AMD64_UNSUPPORTED_SSE_ENCODING") || strings.Contains(text, "native image bytes=") {
		t.Fatalf("planning boundary: %v\n%s", err, text)
	}
	for _, want := range []string{"fn MixedA scalar planning", "incoming arg0 -> RCX\n  incoming arg1 -> XMM1\n  incoming arg2 -> R8\n  incoming arg3 -> XMM3", "incoming arg0 -> XMM0\n  incoming arg1 -> RDX\n  incoming arg2 -> XMM2\n  incoming arg3 -> R9", "arg4 v4:float64 -> stack[0]", "arg5 v5:float32 -> stack[1]", "arg6 v6:i32 -> stack[2]", "arg7 v7:bool -> stack[3]", "return XMM0 ->", "return void", "xmm-clobbers=XMM0..XMM5", "frame size=104 unaligned=104", "float32 -> [rsp+44] size=4 align=4", "float64 -> [rsp+48] size=8 align=8", "temp[0]", "abstract-call (no SSE bytes)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "[rsp-1]") {
		t.Fatal("physical locations printed as frame offsets")
	}
}
