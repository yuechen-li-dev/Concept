package concept

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Exercise the emitted bytes through the same Win64 executable-memory path as
// EVT2d. The frame is caller-owned and its layout is pinned by the LIR test.
func TestEVT2x3FiniteNativeStep(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native executable-memory test requires Windows AMD64")
	}
	backendPath := filepath.Join("..", "..", "libraries", "Standard", "Backend", "AMD64.concept")
	backendSource, err := os.ReadFile(backendPath)
	if err != nil {
		t.Fatal(err)
	}
	backend, err := Parse(backendPath, string(backendSource))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(backend, backendSource)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	const harness = `#include "amd64.generated.h"
#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
typedef struct { uint32_t state; uint8_t completed; uint8_t pad[3]; int32_t count; int32_t increment; } frame;
static void* make_code(concept_readonly_span_byte bridge, int ordinal) {
  unsigned char code[4096] = {0};
  concept_span_byte output = {code, sizeof code};
  concept_result_int_backend_error result = concept_standard__backend__amd64_emit_function(bridge, ordinal, output);
  if (result.tag != 0) { fprintf(stderr, "backend %d error %u\n", ordinal, result.payload.error.error.tag); return NULL; }
  for (int repeat = 0; repeat < 100; ++repeat) {
    unsigned char next[4096] = {0};
    concept_span_byte again = {next, sizeof next};
    concept_result_int_backend_error emitted = concept_standard__backend__amd64_emit_function(bridge, ordinal, again);
    if (emitted.tag != 0 || emitted.payload.ok.value != result.payload.ok.value ||
        memcmp(code, next, (size_t)result.payload.ok.value) != 0) return NULL;
  }
  void* executable = VirtualAlloc(NULL, (size_t)result.payload.ok.value, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
  if (!executable) return NULL;
  memcpy(executable, code, (size_t)result.payload.ok.value);
  DWORD old = 0;
  if (!VirtualProtect(executable, (size_t)result.payload.ok.value, PAGE_EXECUTE_READ, &old)) return NULL;
  if (!FlushInstructionCache(GetCurrentProcess(), executable, (size_t)result.payload.ok.value)) return NULL;
  return executable;
}
int main(int argc, char** argv) {
  if (argc != 3) return 90;
  FILE* file = fopen(argv[1], "rb"); if (!file) return 91;
  if (fseek(file, 0, SEEK_END)) return 92;
  long n = ftell(file); if (n <= 0 || n > 1000000 || fseek(file, 0, SEEK_SET)) return 93;
  unsigned char* bytes = malloc((size_t)n); if (!bytes || fread(bytes, 1, (size_t)n, file) != (size_t)n) return 94;
  fclose(file);
  concept_readonly_span_byte bridge = {bytes, (size_t)n};
  void* initCode = make_code(bridge, 0); if (!initCode) return 95;
  int scenario = atoi(argv[2]);
  if (scenario == 3) {
    typedef void (*activation_init_fn)(void*, int32_t);
    activation_init_fn rootInit = NULL; memcpy(&rootInit, &initCode, sizeof rootInit);
    unsigned char instance[108] = {0};
    rootInit(instance, 13);
    uint32_t depth = 0, tag = 99, state = 99; int32_t shared = 0, preserved = 0;
    memcpy(&depth, instance + 0, 4); memcpy(&shared, instance + 8, 4);
    memcpy(&tag, instance + 12, 4); memcpy(&state, instance + 16, 4);
    memcpy(&preserved, instance + 20, 4);
    if (depth != 1 || instance[4] != 0 || shared != 13 || tag != 0 || state != 0 || preserved != 7) return 111;
    VirtualFree(initCode, 0, MEM_RELEASE); free(bytes); return 0;
  }
  void* stepCode = make_code(bridge, 1); if (!stepCode) return 96;
  typedef void (*init_fn)(frame*, int32_t);
  typedef uint32_t (*step_fn)(frame*);
  init_fn init = NULL; step_fn step = NULL;
  memcpy(&init, &initCode, sizeof init); memcpy(&step, &stepCode, sizeof step);
  frame a = {0}, b = {0};
  init(&a, 0); init(&b, 10);
  if (a.state != 0 || a.completed || a.count != 0) return 97;
  if (scenario == 0) {
    if (a.increment != 2) return 97;
    if (step(&a) != 0 || a.state != 1 || a.count != 2) return 98;
    if (step(&b) != 0 || b.state != 1 || b.count != 12) return 99;
    if (step(&a) != 0 || a.state != 2 || a.count != 4) return 100;
    if (step(&a) != 2 || !a.completed || a.count != 4) return 101;
    if (step(&a) != 2 || a.count != 4 || b.count != 12) return 102;
  } else if (scenario == 1) {
    if (step(&a) != 1 || a.state != 0 || a.count != 1) return 103;
    if (step(&b) != 2 || !b.completed || b.count != 11) return 104;
    if (step(&a) != 2 || !a.completed || a.count != 2) return 105;
    if (step(&a) != 2 || a.count != 2) return 106;
  } else if (scenario == 2) {
    if (step(&a) != 1 || a.state != 0 || a.count != 1) return 107;
    if (step(&a) != 1 || a.state != 0 || a.count != 2) return 108;
    if (step(&a) != 2 || !a.completed || a.count != 3) return 109;
  } else return 110;
  VirtualFree(initCode, 0, MEM_RELEASE); VirtualFree(stepCode, 0, MEM_RELEASE); free(bytes);
  return 0;
}`
	harnessPath := filepath.Join(dir, "harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness), 0644); err != nil {
		t.Fatal(err)
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		t.Skip("no C11 compiler")
	}
	executable := filepath.Join(dir, "machine-native.exe")
	cmd := nativeCommand(t, compiler, "-std=c11", "-I", dir, filepath.Join(dir, "amd64.generated.c"), harnessPath, "-o", executable)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native build: %v\n%s", err, out)
	}
	for scenario, file := range []string{"finite.concept", "yield_resume.concept", "multi_yield.concept"} {
		module, _ := evt2MachineFixture(t, file)
		machine, err := GenerateMachineIR(module)
		if err != nil {
			t.Fatal(err)
		}
		bridge, err := EncodeMachineBridge(machine)
		if err != nil {
			t.Fatal(err)
		}
		artifact := filepath.Join(dir, file+".cmir")
		if err := os.WriteFile(artifact, bridge, 0644); err != nil {
			t.Fatal(err)
		}
		if out, err := nativeCommand(t, executable, artifact, string(rune('0'+scenario))).CombinedOutput(); err != nil {
			t.Fatalf("%s native execution: %v\n%s", file, err, strings.TrimSpace(string(out)))
		}
	}
	path := filepath.Join("..", "..", "language", "evt1", "machine-stack", "valid", "machine_parent_resume.concept")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(filepath.ToSlash(path), string(source))
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	a := buildMIR(module, env).Automata[0]
	layout, err := planActivationStack(a, env)
	if err != nil {
		t.Fatal(err)
	}
	init, err := lowerActivationRootInit(a, layout)
	if err != nil {
		t.Fatal(err)
	}
	machine, err := LowerLirToAmd64Machine(LIRModule{Functions: []LIRFunction{init}})
	if err != nil {
		t.Fatal(err)
	}
	bridge, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "activation-init.cmir")
	if err := os.WriteFile(artifact, bridge, 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := nativeCommand(t, executable, artifact, "3").CombinedOutput(); err != nil {
		t.Fatalf("activation root Init native execution: %v\n%s", err, strings.TrimSpace(string(out)))
	}
}
