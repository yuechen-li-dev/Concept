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
func TestEVT2x3FiniteNativeStepAndEVT2x5DynamicAddressing(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native executable-memory test requires Windows AMD64")
	}
	outputs := backendTestOutputs(t, false)
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
  if (scenario == 4) {
    typedef void (*indexed_store_fn)(void*, uint32_t, uint32_t);
    indexed_store_fn store = NULL; memcpy(&store, &initCode, sizeof store);
    uint32_t protected_slots[11];
    for (int i = 0; i < 11; ++i) protected_slots[i] = 0xA5A5A5A5u;
    for (uint32_t i = 0; i < 3; ++i) store(&protected_slots[1], i, 101u + i);
    for (int i = 0; i < 11; ++i) {
      uint32_t expected = i == 2 ? 101u : i == 5 ? 102u : i == 8 ? 103u : 0xA5A5A5A5u;
      if (protected_slots[i] != expected) return 112;
    }
    VirtualFree(initCode, 0, MEM_RELEASE); free(bytes); return 0;
  }
  if (scenario == 5) {
    typedef uint32_t (*top_tag_fn)(void*);
    top_tag_fn topTag = NULL; memcpy(&topTag, &initCode, sizeof topTag);
    uint32_t protected_slots[29];
    for (int i = 0; i < 29; ++i) protected_slots[i] = 0xA5A5A5A5u;
    protected_slots[4] = 17u; protected_slots[7] = 19u; protected_slots[10] = 23u;
    protected_slots[1] = 1u; if (topTag(&protected_slots[1]) != 17u) return 113;
    protected_slots[1] = 2u; if (topTag(&protected_slots[1]) != 19u) return 113;
    protected_slots[1] = 3u; if (topTag(&protected_slots[1]) != 23u) return 113;
    if (protected_slots[0] != 0xA5A5A5A5u || protected_slots[28] != 0xA5A5A5A5u) return 114;
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
	objects := nativeFixtureObjects(t, outputs, compiler)
	args := append([]string{"-std=c11", "-I", dir}, objects...)
	args = append(args, harnessPath, "-o", executable)
	cmd := nativeCommand(t, compiler, args...)
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
	stride := LIRFunction{
		Identity: "StoreStride12", Name: "StoreStride12",
		Params: []LIRValue{{ID: 0, Type: "ptr<u32>"}, {ID: 1, Type: "u32"}, {ID: 2, Type: "u32"}}, Result: "void",
		Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{
			{Op: "check_index", Result: -1, Type: "void", Args: []int{1}, Slot: -1, Extent: 3},
			{Op: "index_address", Result: 3, Type: "ptr<u32>", Args: []int{0, 1}, Slot: -1, Extent: 3, Stride: layout.SlotSize, FrameOffset: layout.SlotDataOffset},
			{Op: "store", Result: -1, Type: "u32", Args: []int{3, 2}, Slot: -1},
		}, Term: LIRTerminator{Op: "return"}}},
	}
	machine, err = LowerLirToAmd64Machine(LIRModule{Functions: []LIRFunction{stride}})
	if err != nil {
		t.Fatal(err)
	}
	bridge, err = EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	artifact = filepath.Join(dir, "stride12.cmir")
	if err := os.WriteFile(artifact, bridge, 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := nativeCommand(t, executable, artifact, "4").CombinedOutput(); err != nil {
		t.Fatalf("12-byte indexed native store: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	top := LIRFunction{
		Identity: "TopTagStride12", Name: "TopTagStride12",
		Params: []LIRValue{{ID: 0, Type: "ptr<u32>"}}, Result: "u32",
		Blocks: []LIRBlock{{ID: 0, Instructions: []LIRInstruction{
			{Op: "load", Result: 1, Type: "u32", Args: []int{0}, Slot: -1},
			{Op: "const", Result: 2, Type: "u32", Literal: "1", Slot: -1},
			{Op: "sub", Result: 3, Type: "u32", Args: []int{1, 2}, Slot: -1},
			{Op: "check_index", Result: -1, Type: "void", Args: []int{3}, Slot: -1, Extent: layout.Capacity},
			{Op: "index_address", Result: 4, Type: "ptr<u32>", Args: []int{0, 3}, Slot: -1, Extent: layout.Capacity, Stride: layout.SlotSize, FrameOffset: layout.SlotsOffset + layout.SlotTagOffset},
			{Op: "load", Result: 5, Type: "u32", Args: []int{4}, Slot: -1},
		}, Term: LIRTerminator{Op: "return", Value: 5}}},
	}
	machine, err = LowerLirToAmd64Machine(LIRModule{Functions: []LIRFunction{top}})
	if err != nil {
		t.Fatal(err)
	}
	multiply, legalAddress := false, false
	for _, block := range machine.Functions[0].Blocks {
		for _, in := range block.Instructions {
			if in.Op == "IMUL" {
				multiply = true
			}
			if in.Op == "LEA" && len(in.Src) == 1 && in.Src[0].Kind == "mem" && in.Src[0].Scale == 1 && in.Src[0].Disp == layout.SlotsOffset+layout.SlotTagOffset {
				legalAddress = true
			}
		}
	}
	if !multiply || !legalAddress {
		t.Fatal("stride 12 was not legalized to multiply and scale-one address")
	}
	bridge, err = EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	artifact = filepath.Join(dir, "top-tag-stride12.cmir")
	if err := os.WriteFile(artifact, bridge, 0644); err != nil {
		t.Fatal(err)
	}
	if out, err := nativeCommand(t, executable, artifact, "5").CombinedOutput(); err != nil {
		t.Fatalf("dynamic top tag native load: %v\n%s", err, strings.TrimSpace(string(out)))
	}
}
