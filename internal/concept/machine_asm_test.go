package concept

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

const asmSource = `module AsmProbe; profile Core;
uint32 Reverse(uint32 value) {
    unsafe asm AMD64 { "bswap {value}"; inout register value; clobber flags; memory none; }
    return value;
}
uint32 Constant() {
    uint32 result = 0;
    unsafe asm AMD64 { "mov $42, {result}"; out register result; memory none; }
    return result;
}
uint32 Observe(uint32 value) {
    unsafe asm AMD64 { "cmp $7, {value}"; in register value; clobber flags; memory read; }
    return value;
}
uint32 NamedClobber(uint32 value) {
    unsafe asm AMD64 { "mov {value}, %r10d"; in register value; clobber r10; memory none; }
    return value;
}
uint32 ConstInput() {
    const uint32 value = 7;
    unsafe asm AMD64 { "cmp $7, {value}"; in register value; clobber flags; memory none; }
    return value;
}
void Prove() { Assert.Concept<NoAllocation>(Reverse, "inline instruction does not allocate"); }`

const multiAsmSource = `module MultiAsm; profile Core;
uint32 Add(uint32 a, uint32 b) {
    unsafe asm AMD64 { "add {a}, {b}"; in register a; inout register b; clobber flags; memory none; }
    return b;
}
uint32 SwapFirst(uint32 a, uint32 b) {
    unsafe asm AMD64 { "xchg {a}, {b}"; inout register a; inout register b; memory none; }
    return a;
}
uint32 Fixed(uint32 value) {
    unsafe asm AMD64 { "bswap {value}"; inout eax value; memory none; }
    return value;
}`

func TestMultiOperandAsmNativeAndConflicts(t *testing.T) {
	module, err := Parse("multi_asm.concept", multiAsmSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(multiAsmSource), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	assembly := string(outputs["multi_asm.machine.S"])
	if !strings.Contains(assembly, "add %r8d, %r9d") || !strings.Contains(assembly, "bswap %eax") {
		t.Fatal(assembly)
	}
	for _, tc := range []struct{ body, code string }{
		{`"add {a}, {b}"; in eax a; in eax b; memory none;`, "ASM_REGISTER_CONFLICT"},
		{`"add {a}, {b} # {c}"; inout register a; inout register b; in register c; clobber r10; clobber r11; memory none;`, "ASM_OPERAND_LIMIT"},
		{`"bswap {a}"; inout register a; in register a; memory none;`, "ASM_OPERAND_DUPLICATE"},
		{`"bswap {a}"; inout eax a; clobber eax; memory none;`, "ASM_REGISTER_CONFLICT"},
	} {
		_, err := Parse("bad_multi.concept", "profile Core; void Bad(uint32 a, uint32 b, uint32 c) { unsafe asm AMD64 { "+tc.body+" } }")
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("expected %s, got %v", tc.code, err)
		}
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("AMD64 execution requires AMD64 host")
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC unavailable")
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := evt1SemanticSymbolBase(module)
	harness := fmt.Sprintf("#include \"multi_asm.generated.h\"\nint main(void) { return %s(7,9)!=16 || %s(7,9)!=9 || %s(0x11223344u)!=0x44332211u; }\n", evt1FunctionSymbol(base, "Add"), evt1FunctionSymbol(base, "SwapFirst"), evt1FunctionSymbol(base, "Fixed"))
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-O2", "multi_asm.generated.c", "multi_asm.machine.S", "harness.c", "-o", "multi-test")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	if out, err := exec.Command(filepath.Join(dir, "multi-test")).CombinedOutput(); err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
}

func TestStructuredAMD64AsmOperandsMIRAndExecution(t *testing.T) {
	module, err := Parse("asm_probe.concept", asmSource)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(asmSource), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["asm_probe.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	roles := map[string]bool{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "machine_asm" {
				if op.MachineAssembly == nil || !op.NoAllocation || op.MachineAssembly.Architecture != "AMD64" || op.MachineAssembly.ControlFlow != "local" {
					t.Fatalf("asm MIR lost typed effects: %+v", op)
				}
				for _, operand := range op.MachineAssembly.Operands {
					roles[operand.Mode] = true
				}
			}
		}
	}
	if !roles["in"] || !roles["out"] || !roles["inout"] || !strings.Contains(string(outputs["asm_probe.machine.S"]), "bswap %r8d") {
		t.Fatalf("assembly operand roles or helper missing: %+v", roles)
	}
	if _, err := GenerateForTarget(module, []byte(asmSource), AArch64GenericTarget()); err == nil || !strings.Contains(err.Error(), "MACHINE_ARCHITECTURE_MISMATCH") {
		t.Fatalf("wrong architecture accepted: %v", err)
	}
	if runtime.GOARCH != "amd64" {
		t.Skip("AMD64 execution requires AMD64 host")
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC is unavailable")
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := evt1SemanticSymbolBase(module)
	harness := fmt.Sprintf("#include \"asm_probe.generated.h\"\nint main(void) { return %s(0x11223344u) != 0x44332211u || %s() != 42u || %s(7u) != 7u || %s(9u) != 9u || %s() != 7u; }\n", evt1FunctionSymbol(base, "Reverse"), evt1FunctionSymbol(base, "Constant"), evt1FunctionSymbol(base, "Observe"), evt1FunctionSymbol(base, "NamedClobber"), evt1FunctionSymbol(base, "ConstInput"))
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "asm-test")
	build := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-O2", "asm_probe.generated.c", "asm_probe.machine.S", "harness.c", "-o", binary)
	build.Dir = dir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("strict C11/asm build: %v\n%s", err, output)
	}
	if output, err := nativeCommand(t, binary).CombinedOutput(); err != nil {
		t.Fatalf("asm operand execution: %v\n%s", err, output)
	}
}

func TestStructuredAsmInvalidForms(t *testing.T) {
	for _, tc := range []struct{ body, code string }{
		{`unsafe asm AArch64 { "mov {value}"; inout register value; memory none; }`, "ASM_ARCHITECTURE_INVALID"},
		{`unsafe asm AMD64 { "not {value}"; inout register value; }`, "ASM_MEMORY_EFFECT_REQUIRED"},
		{`unsafe asm AMD64 { "mov %ebx, {value}"; out register value; memory none; }`, "ASM_REGISTER_UNDECLARED"},
		{`unsafe asm AMD64 { "mov {value}, %r10d"; in register value; memory none; }`, "ASM_REGISTER_UNDECLARED"},
		{`unsafe asm AMD64 { "jmp {value}"; in register value; memory none; }`, "ASM_CONTROL_FLOW_UNSUPPORTED"},
		{`unsafe asm AMD64 { "not {value}"; inout register value; clobber unknown; memory none; }`, "ASM_CLOBBER_INVALID"},
		{`unsafe asm AMD64 { "not {value}"; inout register value; in register value; memory none; }`, "ASM_OPERAND_DUPLICATE"},
	} {
		source := "profile Core; void Bad(uint32 value) { " + tc.body + " }"
		_, err := Parse("bad_asm.concept", source)
		if err == nil || !strings.Contains(err.Error(), tc.code) {
			t.Fatalf("expected %s for %s, got %v", tc.code, tc.body, err)
		}
	}
	_, err := Parse("bad_asm_aggregate.concept", `profile Core; record struct Pair { uint32 a; uint32 b; }
void Bad(Pair value) { unsafe asm AMD64 { "not {value}"; inout register value; memory none; } }`)
	if err == nil || !strings.Contains(err.Error(), "ASM_OPERAND_TYPE") {
		t.Fatalf("aggregate asm operand accepted: %v", err)
	}
	_, err = Parse("bad_asm_const.concept", `profile Core;
void Bad() { const uint32 value = 1; unsafe asm AMD64 { "not {value}"; out register value; memory none; } }`)
	if err == nil || !strings.Contains(err.Error(), "ASM_OUTPUT_INVALID") {
		t.Fatalf("const asm output accepted: %v", err)
	}
}

func TestStructuredAsmArtifactAndMMIOOrdering(t *testing.T) {
	artifact := buildSemanticArtifact(t, "AsmProbe.concept", asmSource, nil)
	consumer := `module AsmConsumer; profile Core; import AsmProbe;
struct DeviceMemory {}
uint8 Ordered(Address<DeviceMemory> first, Address<DeviceMemory> second, uint32 value) {
    MmioStore<uint8>(first, 1);
    unsafe asm AMD64 { "not {value}"; inout register value; clobber flags; memory readwrite; }
    return MmioLoad<uint8>(second);
}

uint32 Imported(uint32 value) { return Reverse(value); }
void ConsumerProof() { Assert.Concept<NoAllocation>(Imported, "imported asm body does not allocate"); }`
	module, err := ParseWithSemanticModules("asm_consumer.concept", consumer, map[string][]byte{"AsmProbe": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(consumer), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["asm_consumer.machine.S"]), "bswap %r8d") {
		t.Fatal("artifact-only consumer lost imported asm helper")
	}
	plan, err := GeneratePlan(module, X86_64GenericTarget())
	if err != nil || !strings.Contains(string(plan), "MemoryEffect:readwrite") {
		t.Fatalf("planner lost explicit asm memory effect: %v\n%s", err, plan)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["asm_consumer.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fn := range mir.Functions {
		if fn.Name != "Ordered" {
			continue
		}
		var ordered []string
		for _, op := range fn.Operations {
			if op.Kind == "mmio_write" || op.Kind == "machine_asm" || op.Kind == "mmio_read" {
				ordered = append(ordered, op.Kind)
				if op.Kind == "machine_asm" && (op.MachineAssembly == nil || op.MachineAssembly.MemoryEffect != "readwrite") {
					t.Fatalf("asm memory effect lost in MIR: %+v", op)
				}
			}
		}
		if strings.Join(ordered, ",") != "mmio_write,machine_asm,mmio_read" {
			t.Fatalf("MMIO/asm source order changed: %v", ordered)
		}
		return
	}
	t.Fatal("Ordered function missing")
}

func TestStructuredAsmArtifactsAreByteIdenticalAcross100Runs(t *testing.T) {
	var expected Outputs
	var expectedArtifact []byte
	for i := 0; i < determinismRuns(); i++ {
		module, err := Parse("asm_probe.concept", asmSource)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTarget(module, []byte(asmSource), X86_64GenericTarget())
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := CompileSemanticModule("asm_probe.concept", asmSource, nil)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && (!reflect.DeepEqual(outputs, expected) || !reflect.DeepEqual(artifact, expectedArtifact)) {
			t.Fatalf("structured asm output changed on run %d", i+1)
		}
		expected, expectedArtifact = outputs, artifact
	}
}
