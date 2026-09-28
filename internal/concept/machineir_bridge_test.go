package concept

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEVT2dMachineBridgeRoundTrip(t *testing.T) {
	machine := machineFixture(t)
	first, err := EncodeMachineBridge(machine)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(first, []byte(MachineBridgeSchema)) {
		t.Fatal("schema missing")
	}
	decoded, err := DecodeMachineBridge(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeMachineBridge(decoded)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("round trip changed artifact: %v", err)
	}
	if decoded.String() != machine.String() {
		t.Fatal("round trip changed MachineIR inspection")
	}
	for i := 0; i < 100; i++ {
		again, err := EncodeMachineBridge(machine)
		if err != nil || !bytes.Equal(first, again) {
			t.Fatalf("nondeterministic bridge on run %d: %v", i, err)
		}
	}
}

func TestEVT2dMachineBridgeRejectsMismatchAndCorruption(t *testing.T) {
	data, err := EncodeMachineBridge(machineFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	wrong := bytes.Clone(data)
	wrong[7] = '2'
	if _, err := DecodeMachineBridge(wrong); err == nil || !strings.Contains(err.Error(), "MIR_BRIDGE_SCHEMA_MISMATCH") {
		t.Fatalf("schema mismatch accepted: %v", err)
	}
	if _, err := DecodeMachineBridge(data[:len(data)-1]); err == nil {
		t.Fatal("truncated artifact accepted")
	}
	wrong = append(bytes.Clone(data), 1)
	if _, err := DecodeMachineBridge(wrong); err == nil || !strings.Contains(err.Error(), "MIR_BRIDGE_TRAILING_BYTES") {
		t.Fatalf("trailing bytes accepted: %v", err)
	}
}

func TestEVT2dConceptBackendCompilesThroughC11(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "libraries", "Standard", "Backend", "AMD64.concept")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(sourcePath, string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	var cfile string
	for name, body := range outputs {
		if !strings.HasSuffix(name, ".generated.c") && !strings.HasSuffix(name, ".generated.h") {
			continue
		}
		path := filepath.Join(dir, filepath.Base(name))
		if err := os.WriteFile(path, body, 0644); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".generated.c") {
			cfile = path
		}
	}
	if cfile == "" {
		t.Fatal("missing generated C")
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		compiler, err = exec.LookPath("clang")
	}
	if err != nil {
		t.Skip("no C11 compiler")
	}
	object := filepath.Join(dir, "amd64.o")
	cmd := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-c", cfile, "-o", object)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Concept backend C11 compilation: %v\n%s", err, out)
	}
}

func TestEVT2dConceptBackendNativeAddMax(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native executable-memory test requires Windows AMD64")
	}
	sourcePath := filepath.Join("..", "..", "libraries", "Standard", "Backend", "AMD64.concept")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(sourcePath, string(source))
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
	oraclePath := filepath.Join("..", "..", "language", "evt2", "valid", "core.concept")
	oracleSource, err := os.ReadFile(oraclePath)
	if err != nil {
		t.Fatal(err)
	}
	oracleModule, err := Parse(oraclePath, string(oracleSource))
	if err != nil {
		t.Fatal(err)
	}
	oracleOutputs, err := Generate(oracleModule, oracleSource)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, oracleOutputs); err != nil {
		t.Fatal(err)
	}
	var cfile string
	for name := range outputs {
		if strings.HasSuffix(name, ".generated.c") {
			cfile = filepath.Join(dir, name)
		}
	}
	if cfile == "" {
		t.Fatal("missing backend C")
	}
	oracleC := filepath.Join(dir, "core.generated.c")
	bridge, err := EncodeMachineBridge(machineFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(dir, "fixture.cmir")
	if err := os.WriteFile(artifact, bridge, 0644); err != nil {
		t.Fatal(err)
	}
	const harness = `#include "amd64.generated.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
extern int32_t concept_core_add(int32_t a, int32_t b);
extern int32_t concept_core_max(int32_t a, int32_t b);
typedef struct { int32_t data[4]; } concept_array_4_int;
extern int32_t concept_core_checked_index(concept_array_4_int values, int32_t index);
extern int32_t concept_core_store_index(concept_array_4_int values, int32_t index, int32_t value);
extern int32_t concept_core_early(int32_t value);
extern int32_t concept_core_sum4(concept_array_4_int values);
extern int32_t concept_core_choose(bool condition, int32_t a, int32_t b);
#ifdef _WIN32
#include <windows.h>
#endif
int main(int argc, char** argv) {
  if (argc != 5) return 90;
  FILE* file = fopen(argv[1], "rb");
  if (!file) return 91;
  if (fseek(file, 0, SEEK_END) != 0) return 92;
  long size = ftell(file);
  if (size <= 0 || size > 1000000 || fseek(file, 0, SEEK_SET) != 0) return 93;
  unsigned char* bridge = malloc((size_t)size + 1u);
  if (!bridge || fread(bridge, 1, (size_t)size, file) != (size_t)size) return 94;
  fclose(file);
  unsigned char code[1024] = {0};
  concept_readonly_span_byte input = {bridge, (size_t)size};
  concept_span_byte output = {code, sizeof code};
  int ordinal = atoi(argv[2]);
  int a = atoi(argv[3]);
  int b = atoi(argv[4]);
  unsigned char originalVersion = bridge[7];
  bridge[7] = '2';
  concept_result_int_backend_error wrongVersion = concept_standard__backend__amd64_emit_function(input, ordinal, output);
  if (wrongVersion.tag == 0 || wrongVersion.payload.error.error.tag != 1) return 103;
  bridge[7] = originalVersion;
  bridge[size] = 0;
  concept_readonly_span_byte withTrailingByte = {bridge, (size_t)size + 1u};
  if (concept_standard__backend__amd64_emit_function(withTrailingByte, ordinal, output).tag == 0) return 113;
  concept_span_byte shortOutput = {code, 1};
  concept_result_int_backend_error tooSmall = concept_standard__backend__amd64_emit_function(input, ordinal, shortOutput);
  if (tooSmall.tag == 0 || tooSmall.payload.error.error.tag != 5) return 104;
  concept_result_int_backend_error result = concept_standard__backend__amd64_emit_function(input, ordinal, output);
  if (result.tag != 0) { fprintf(stderr, "backend error tag=%u\n", result.payload.error.error.tag); return 95; }
  int length = result.payload.ok.value;
  concept_backend_function function = concept_standard__backend__amd64_empty_function();
  if (concept_standard__backend__amd64_decode_function(input, ordinal, &function).tag != 0) return 105;
  concept_allocation allocation = concept_standard__backend__amd64_empty_allocation();
  if (concept_standard__backend__amd64_allocate_registers(&function, &allocation).tag != 0) return 106;
  concept_result_finalized_frame_backend_error frame = concept_standard__backend__amd64_finalize_frame(&function);
  if (frame.tag != 0) return 107;
  if ((ordinal == 0 || ordinal == 1) && frame.payload.ok.value.byteCount != 0) return 108;
  for (int repeat = 0; repeat < 100; ++repeat) {
    unsigned char next[1024] = {0};
    concept_span_byte nextOutput = {next, sizeof next};
    concept_result_int_backend_error again = concept_standard__backend__amd64_emit_function(input, ordinal, nextOutput);
    if (again.tag != 0 || again.payload.ok.value != length || memcmp(code, next, (size_t)length) != 0) return 100;
    concept_allocation nextAllocation = concept_standard__backend__amd64_empty_allocation();
    if (concept_standard__backend__amd64_allocate_registers(&function, &nextAllocation).tag != 0) return 109;
    for (int id = 0; id < function.virtualRegCount; ++id) {
      if (allocation.start.data[id] != nextAllocation.start.data[id] ||
          allocation.end.data[id] != nextAllocation.end.data[id] ||
          allocation.assigned.data[id] != nextAllocation.assigned.data[id] ||
          allocation.physical.data[id].tag != nextAllocation.physical.data[id].tag) return 110;
    }
    concept_result_finalized_frame_backend_error nextFrame = concept_standard__backend__amd64_finalize_frame(&function);
    if (nextFrame.tag != 0 || nextFrame.payload.ok.value.byteCount != frame.payload.ok.value.byteCount) return 111;
    for (int slot = 0; slot < function.slotCount; ++slot) {
      if (nextFrame.payload.ok.value.slotOffsets.data[slot] != frame.payload.ok.value.slotOffsets.data[slot]) return 112;
    }
  }
  for (int i = 0; i < length; ++i) printf("%02x", code[i]);
  printf("\n");
#ifdef _WIN32
  void* executable = VirtualAlloc(NULL, (size_t)length, MEM_RESERVE | MEM_COMMIT, PAGE_READWRITE);
  if (!executable) return 96;
  memcpy(executable, code, (size_t)length);
  DWORD oldProtection = 0;
  if (!VirtualProtect(executable, (size_t)length, PAGE_EXECUTE_READ, &oldProtection)) return 97;
  if (!FlushInstructionCache(GetCurrentProcess(), executable, (size_t)length)) return 98;
  int actual = 0;
  int expected = 0;
  if (ordinal == 5) {
    typedef int (*native_choose_fn)(bool, int, int);
    native_choose_fn native = NULL;
    memcpy(&native, &executable, sizeof native);
    for (int repeat = 0; repeat < 100; ++repeat) actual = native(a != 0, b, 7);
    expected = concept_core_choose(a != 0, b, 7);
  } else if (ordinal == 2) {
    typedef int (*native_sum_fn)(const int*);
    native_sum_fn native = NULL;
    concept_array_4_int values = {{a, b, 3, 4}};
    memcpy(&native, &executable, sizeof native);
    for (int repeat = 0; repeat < 100; ++repeat) actual = native(values.data);
    expected = concept_core_sum4(values);
  } else if (ordinal == 6) {
    typedef int (*native_early_fn)(int);
    native_early_fn native = NULL;
    memcpy(&native, &executable, sizeof native);
    for (int repeat = 0; repeat < 100; ++repeat) actual = native(a);
    expected = concept_core_early(a);
  } else if (ordinal == 3) {
    typedef int (*native_index_fn)(const int*, int);
    native_index_fn native = NULL;
    concept_array_4_int values = {{10, 20, 30, 40}};
    memcpy(&native, &executable, sizeof native);
    for (int repeat = 0; repeat < 100; ++repeat) actual = native(values.data, b);
    expected = concept_core_checked_index(values, b);
  } else if (ordinal == 4) {
    typedef int (*native_store_fn)(int*, int, int);
    native_store_fn native = NULL;
    concept_array_4_int values = {{10, 20, 30, 40}};
    concept_array_4_int original = values;
    memcpy(&native, &executable, sizeof native);
    for (int repeat = 0; repeat < 100; ++repeat) actual = native(values.data, b, a);
    expected = concept_core_store_index(original, b, a);
    if (values.data[b] != a) return 102;
  } else {
    typedef int (*native_fn)(int, int);
    native_fn native = NULL;
    memcpy(&native, &executable, sizeof native);
    for (int repeat = 0; repeat < 100; ++repeat) actual = native(a, b);
    expected = ordinal == 0 ? concept_core_add(a, b) : concept_core_max(a, b);
  }
  if (actual != expected) return 101;
  printf("%d\n", actual);
  if (!VirtualFree(executable, 0, MEM_RELEASE)) return 99;
#endif
  free(bridge);
  return 0;
}

`
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
	executable := filepath.Join(dir, "backend-native.exe")
	cmd := nativeCommand(t, compiler, "-std=c11", "-Wall", "-Wextra", "-I", dir, cfile, oracleC, harnessPath, "-o", executable)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("native backend build: %v\n%s", err, out)
	}
	for _, tc := range []struct{ ordinal, a, b, want string }{
		{"0", "2", "3", "5"}, {"0", "-7", "5", "-2"}, {"0", "1000000", "-100", "999900"},
		{"1", "2", "3", "3"}, {"1", "7", "-2", "7"}, {"1", "-5", "-1", "-1"},
		{"2", "1", "2", "10"}, {"2", "-1", "5", "11"},
		{"3", "0", "0", "10"}, {"3", "0", "3", "40"},
		{"4", "77", "2", "77"}, {"4", "-9", "0", "-9"},
		{"5", "1", "-8", "-8"}, {"5", "0", "-8", "7"},
		{"6", "-2", "0", "-2"}, {"6", "0", "0", "0"}, {"6", "5", "0", "4"},
	} {
		out, err := nativeCommand(t, executable, artifact, tc.ordinal, tc.a, tc.b).CombinedOutput()
		if err != nil {
			t.Fatalf("native backend %s: %v\n%s", tc.ordinal, err, out)
		}
		lines := strings.Fields(string(out))
		if len(lines) != 2 || lines[1] != tc.want {
			t.Fatalf("native backend %s: %q", tc.ordinal, out)
		}
		if want, ok := map[string]string{
			"0": "4189ca4189d34589d24501da0f8005000000e9020000000f0b4489d0c3",
			"1": "4189ca4189d34539da0f8f05000000e9040000004489d0c3e9000000004489d8c3",
		}[tc.ordinal]; ok && lines[0] != want {
			t.Fatalf("function %s bytes changed: got %s want %s", tc.ordinal, lines[0], want)
		}
		if tc.ordinal == "0" && (!strings.Contains(lines[0], "0f80") || !strings.Contains(lines[0], "0f0b")) {
			t.Fatal("Add lost JO or UD2 overflow path")
		}
		if (tc.ordinal == "3" || tc.ordinal == "4") && (!strings.Contains(lines[0], "0f83") || !strings.Contains(lines[0], "0f0b")) {
			t.Fatalf("function %s lost unsigned bounds branch or UD2", tc.ordinal)
		}
		t.Logf("function %s bytes=%s result=%s", tc.ordinal, lines[0], lines[1])
	}
}

func TestEVT2dConceptEncoderModRMSIB(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "libraries", "Standard", "Backend", "AMD64.concept")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse(sourcePath, string(source))
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
	const harness = `#include "amd64.generated.h"
#include <stdio.h>
#include <string.h>
static int check_memory(int base, int index, int scale, int displacement, int reg, const unsigned char* expected, size_t expectedLength) {
  unsigned char bytes[32] = {0};
  concept_byte_writer writer = {{bytes, sizeof bytes}, 0};
  concept_allocation allocation = {0};
  allocation.assigned.data[0] = true;
  allocation.physical.data[0].tag = (unsigned)base;
  allocation.assigned.data[1] = true;
  allocation.physical.data[1].tag = (unsigned)index;
  concept_machine_operand memory = {0};
  memory.kind.tag = CONCEPT_OPERAND_KIND_MEMORY;
  memory.base = 0;
  memory.baseSlot = -1;
  memory.index = index < 0 ? -1 : 1;
  memory.scale = scale;
  memory.displacement = displacement;
  memory.width = 4;
  concept_register destination = {(unsigned)reg};
  concept_result_void_backend_error result = concept_standard__backend__amd64_write_memory_reg(&writer, &allocation, 139, destination, memory, 4);
  return result.tag == 0 && writer.offset == (int)expectedLength && memcmp(bytes, expected, expectedLength) == 0;
}
static int check_reg(int opcode, int rm, int reg, const unsigned char* expected, size_t expectedLength) {
  unsigned char bytes[8] = {0};
  concept_byte_writer writer = {{bytes, sizeof bytes}, 0};
  concept_register a = {(unsigned)rm};
  concept_register b = {(unsigned)reg};
  concept_result_void_backend_error result = concept_standard__backend__amd64_write_reg_reg(&writer, opcode, a, b, 4);
  return result.tag == 0 && writer.offset == (int)expectedLength && memcmp(bytes, expected, expectedLength) == 0;
}
static int check_allocator_exhaustion(void) {
  concept_backend_function function = concept_standard__backend__amd64_empty_function();
  function.blockCount = 1;
  function.virtualRegCount = 6;
  function.instructionCount = 12;
  function.blocks.data[0].id = 0;
  function.blocks.data[0].firstInstruction = 0;
  function.blocks.data[0].instructionCount = 12;
  function.blocks.data[0].terminator.tag = CONCEPT_OPCODE_RET;
  for (int id = 0; id < 6; ++id) {
    concept_machine_instruction* definition = &function.instructions.data[id];
    definition->op.tag = CONCEPT_OPCODE_MOV;
    definition->destination.kind.tag = CONCEPT_OPERAND_KIND_VIRTUAL;
    definition->destination.id = id;
    definition->sourceCount = 1;
    definition->source0.kind.tag = CONCEPT_OPERAND_KIND_IMMEDIATE;
    concept_machine_instruction* use = &function.instructions.data[id + 6];
    use->op.tag = CONCEPT_OPCODE_MOV;
    use->destination.kind.tag = CONCEPT_OPERAND_KIND_PHYSICAL;
    use->sourceCount = 1;
    use->source0.kind.tag = CONCEPT_OPERAND_KIND_VIRTUAL;
    use->source0.id = id;
  }
  concept_allocation allocation = concept_standard__backend__amd64_empty_allocation();
  concept_result_void_backend_error result = concept_standard__backend__amd64_allocate_registers(&function, &allocation);
  return result.tag != 0 && result.payload.error.error.tag == CONCEPT_BACKEND_ERROR_REGISTER_EXHAUSTED;
}
int main(void) {
  if (!check_reg(137, CONCEPT_REGISTER_RAX, CONCEPT_REGISTER_RCX, (unsigned char[]){0x89,0xc8}, 2)) return 1;
  if (!check_reg(1, CONCEPT_REGISTER_RAX, CONCEPT_REGISTER_RDX, (unsigned char[]){0x01,0xd0}, 2)) return 2;
  if (!check_reg(57, CONCEPT_REGISTER_RCX, CONCEPT_REGISTER_RDX, (unsigned char[]){0x39,0xd1}, 2)) return 3;
  if (!check_memory(CONCEPT_REGISTER_RAX, -1, 1, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x8b,0x00}, 2)) return 4;
  if (!check_memory(CONCEPT_REGISTER_R8, -1, 1, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x41,0x8b,0x00}, 3)) return 5;
  if (!check_memory(CONCEPT_REGISTER_R12, -1, 1, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x41,0x8b,0x04,0x24}, 4)) return 6;
  if (!check_memory(CONCEPT_REGISTER_R13, -1, 1, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x41,0x8b,0x45,0x00}, 4)) return 7;
  if (!check_memory(CONCEPT_REGISTER_R12, CONCEPT_REGISTER_R9, 4, 8, CONCEPT_REGISTER_RAX, (unsigned char[]){0x43,0x8b,0x44,0x8c,0x08}, 5)) return 8;
  if (!check_memory(CONCEPT_REGISTER_RAX, CONCEPT_REGISTER_RCX, 1, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x8b,0x04,0x08}, 3)) return 9;
  if (!check_memory(CONCEPT_REGISTER_RAX, CONCEPT_REGISTER_RCX, 2, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x8b,0x04,0x48}, 3)) return 10;
  if (!check_memory(CONCEPT_REGISTER_RAX, CONCEPT_REGISTER_RCX, 4, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x8b,0x04,0x88}, 3)) return 11;
  if (!check_memory(CONCEPT_REGISTER_RAX, CONCEPT_REGISTER_RCX, 8, 0, CONCEPT_REGISTER_RAX, (unsigned char[]){0x8b,0x04,0xc8}, 3)) return 12;
  if (!check_memory(CONCEPT_REGISTER_RAX, -1, 1, 0x12345678, CONCEPT_REGISTER_RAX, (unsigned char[]){0x8b,0x80,0x78,0x56,0x34,0x12}, 6)) return 13;
  if (!check_allocator_exhaustion()) return 14;
  return 0;
}
`
	harnessPath := filepath.Join(dir, "encoding_harness.c")
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
	executable := filepath.Join(dir, "encoding-test.exe")
	cmd := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-I", dir, filepath.Join(dir, "amd64.generated.c"), harnessPath, "-o", executable)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("encoder harness build: %v\n%s", err, out)
	}
	if out, err := nativeCommand(t, executable).CombinedOutput(); err != nil {
		t.Fatalf("encoder field matrix: %v\n%s", err, out)
	}
}
