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

func TestAMD64TypedIntrinsicMIRHelperAndTarget(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("amd64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["amd64.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "machine_intrinsic" {
				if !op.NoAllocation {
					t.Fatalf("machine operation may allocate: %+v", op)
				}
				got = append(got, op.Detail)
			}
		}
	}
	for _, want := range []string{"AMD64.Pause", "AMD64.ReadTimestamp", "AMD64.In8", "AMD64.In16", "AMD64.In32", "AMD64.Out8", "AMD64.Out16", "AMD64.Out32"} {
		if !strings.Contains(strings.Join(got, ","), want) || !strings.Contains(string(outputs["amd64.machine.S"]), machineIntrinsics[want].symbol+":") {
			t.Fatalf("missing %s from MIR or helper: %+v", want, got)
		}
	}
	if !strings.Contains(string(outputs["amd64.generated.c"]), "ConceptAMD64Pause(") {
		t.Fatal("typed Pause wrapper did not call target helper")
	}
	_, err = GenerateForTarget(module, source, AArch64GenericTarget())
	if err == nil || !strings.Contains(err.Error(), "MACHINE_ARCHITECTURE_MISMATCH") {
		t.Fatalf("wrong target was accepted: %v", err)
	}
}

func TestAMD64PauseAndTimestampExecuteNatively(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("AMD64 execution requires an AMD64 host")
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC is not available")
	}
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("amd64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := evt1SemanticSymbolBase(module)
	harness := fmt.Sprintf("#include \"amd64.generated.h\"\nint main(void) { %s(); uint64_t a = %s(); uint64_t b = %s(); return b < a; }\n", evt1FunctionSymbol(base, "Pause"), evt1FunctionSymbol(base, "ReadTimestamp"), evt1FunctionSymbol(base, "ReadTimestamp"))
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "machine-test")
	build := nativeCommand(t, compiler, "-std=c11", "-pedantic-errors", "-O2", "amd64.generated.c", "amd64.machine.S", "harness.c", "-o", binary)
	build.Dir = dir
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("strict C11 wrapper/helper build failed: %v\n%s", err, output)
	}
	if output, err := nativeCommand(t, binary).CombinedOutput(); err != nil {
		t.Fatalf("native Pause/timestamp execution failed: %v\n%s", err, output)
	}
}

func TestAMD64CpuidTypedWrapperExecutesNatively(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("AMD64 execution requires AMD64 host")
	}
	compiler, err := exec.LookPath("gcc")
	if err != nil {
		t.Skip("GCC unavailable")
	}
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("amd64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["amd64.machine.S"]), "cpuid") || !strings.Contains(string(outputs["amd64.machine.S"]), "push %rbx") {
		t.Fatal("CPUID helper lacks fixed-register preservation")
	}
	dir := t.TempDir()
	for name, body := range outputs {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := evt1SemanticSymbolBase(module)
	harness := fmt.Sprintf("#include \"amd64.generated.h\"\nint main(void) { %s r = %s(0u,0u); return r.eax == 0u || r.ebx == 0u || r.ecx == 0u || r.edx == 0u; }\n", evt1CName("CpuidResult"), evt1FunctionSymbol(base, "Cpuid"))
	if err := os.WriteFile(filepath.Join(dir, "harness.c"), []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}
	build := exec.Command(compiler, "-std=c11", "-pedantic-errors", "-O2", "amd64.generated.c", "amd64.machine.S", "harness.c", "-o", "cpuid-test")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("strict C11 CPUID build: %v\n%s", err, out)
	}
	if out, err := exec.Command(filepath.Join(dir, "cpuid-test")).CombinedOutput(); err != nil {
		t.Fatalf("native CPUID: %v\n%s", err, out)
	}
}

func TestAArch64MachineBarrierCrossTarget(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AArch64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("aarch64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, AArch64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	assembly := string(outputs["aarch64.machine.S"])
	for _, want := range []string{"yield", "wfi", "dmb ish", "dsb ish", "isb"} {
		if !strings.Contains(assembly, want) {
			t.Fatalf("missing %s in helper", want)
		}
	}
	var mir MIR
	if err := json.Unmarshal(outputs["aarch64.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Detail == "AArch64.Dmb" && op.MachineOrdering == "inner-shareable-memory" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("AArch64 DMB ordering fact missing from MIR")
	}
	if _, err := GenerateForTarget(module, source, X86_64GenericTarget()); err == nil || !strings.Contains(err.Error(), "MACHINE_ARCHITECTURE_MISMATCH") {
		t.Fatalf("AArch64 on AMD64 target: %v", err)
	}
	if compiler, err := exec.LookPath("clang"); err == nil {
		object := filepath.Join(t.TempDir(), "aarch64.o")
		build := exec.Command(compiler, "-target", "aarch64-none-elf", "-c", "-x", "assembler", "-", "-o", object)
		build.Stdin = strings.NewReader(assembly)
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("AArch64 helper cross-build: %v\n%s", err, out)
		}
	}
}

func TestMachineIntrinsicGenericAndAsyncArtifactConsumer(t *testing.T) {
	library, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/Machine/AMD64.concept", string(library), nil)
	source := `module MachineAsyncConsumer; profile Core; import Standard.Machine.AMD64;
template <typename T> T Wrap(T value) { Pause(); return value; }
async uint32 Child() {
    uint32 value = 7;
    Pause();
    unsafe asm AMD64 { "bswap {value}"; inout register value; memory none; }
    return Wrap<uint32>(value);
}

uint32 Run() {
    Async<uint32> work = Child();
    while (not Complete(work)) bounded(8) { Step(work); }
    return Result(work);
}`
	module, err := ParseWithSemanticModules("machine_async_consumer.concept", source, map[string][]byte{"Standard.Machine.AMD64": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(source), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["machine_async_consumer.machine.S"]), "pause") || !strings.Contains(string(outputs["machine_async_consumer.machine.S"]), "bswap") {
		t.Fatal("async generic consumer lost machine helper")
	}
}

func TestMachinePrivilegeAndBarrierFacts(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	module, err := Parse("amd64.concept", string(source))
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["amd64.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	privileged := map[string]bool{}
	ordering := map[string]string{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind != "machine_intrinsic" {
				continue
			}
			privileged[op.Detail] = op.Privileged
			ordering[op.Detail] = op.MachineOrdering
		}
	}
	for _, id := range []string{"AMD64.Cli", "AMD64.Sti", "AMD64.Hlt"} {
		if !privileged[id] {
			t.Fatalf("%s lacks privileged classification", id)
		}
	}
	for id, want := range map[string]string{"AMD64.Lfence": "load", "AMD64.Sfence": "store", "AMD64.Mfence": "full"} {
		if ordering[id] != want {
			t.Fatalf("%s ordering = %q", id, ordering[id])
		}
	}
	if privileged["AMD64.Pause"] {
		t.Fatal("Pause classified as privileged")
	}
	found := false
	for _, fact := range mir.SemanticFacts {
		if fact.Kind == FactPrivileged {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Privileged semantic fact missing")
	}
}

func TestMachineFenceOrdersMMIOInArtifactConsumer(t *testing.T) {
	library, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/Machine/AMD64.concept", string(library), nil)
	source := `module FenceConsumer; profile Core; import Standard.Machine.AMD64;
struct DeviceMemory {}
uint8 Ordered(Address<DeviceMemory> first, Address<DeviceMemory> second) {
    MmioStore<uint8>(first, 1);
    Mfence();
    return MmioLoad<uint8>(second);
}`
	module, err := ParseWithSemanticModules("fence_consumer.concept", source, map[string][]byte{"Standard.Machine.AMD64": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(source), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["fence_consumer.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	var ordered []string
	for _, fn := range mir.Functions {
		if fn.Name == "Ordered" {
			for _, op := range fn.Operations {
				if op.Kind == "mmio_write" || op.Kind == "mmio_read" || op.Kind == "call" && op.Detail == "Mfence" {
					ordered = append(ordered, op.Kind)
				}
			}
		}
	}
	if strings.Join(ordered, ",") != "mmio_write,call,mmio_read" {
		t.Fatalf("MMIO/fence order: %v", ordered)
	}
}

func TestMachinePrivilegeExplainByteIdenticalAcross100Runs(t *testing.T) {
	source := `module Standard.Machine.AMD64; profile Core;
[[machine("AMD64.Cli")]] extern "C" void ConceptAMD64Cli();
void Cli() { ConceptAMD64Cli(); }
void Prove() { Assert.Concept<Privileged>(Cli, "interrupt masking requires privilege"); }`
	var expected string
	for i := 0; i < 100; i++ {
		graph, err := ExplainSource("privileged.concept", source, 0)
		if err != nil {
			t.Fatal(err)
		}
		got := RenderProofSummary(graph)
		if !strings.Contains(got, "PROVEN Privileged") {
			t.Fatalf("privilege proof missing: %s", got)
		}
		if i > 0 && got != expected {
			t.Fatalf("privilege explain drift at run %d", i+1)
		}
		expected = got
	}
}

func TestAMD64MachineIntrinsicSignatureRejectsForgery(t *testing.T) {
	source := `module Standard.Machine.AMD64; profile Core;
[[machine("AMD64.In8")]] extern "C" uint8 ConceptAMD64In8(uint32 port);`
	_, err := Parse("bad_machine.concept", source)
	if err == nil || !strings.Contains(err.Error(), "MACHINE_INTRINSIC_SIGNATURE") {
		t.Fatalf("invalid port width was accepted: %v", err)
	}
}

func TestAMD64PortCallRejectsOutOfRangeLiteral(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/Machine/AMD64.concept", string(source), nil)
	consumer := `module BadPort; profile Core; import Standard.Machine.AMD64;
uint8 Bad() { return In8(65536); }`
	_, err = ParseWithSemanticModules("bad_port.concept", consumer, map[string][]byte{"Standard.Machine.AMD64": artifact})
	if err == nil || !strings.Contains(err.Error(), "CV4107") {
		t.Fatalf("out-of-range port was accepted: %v", err)
	}
}

func TestAMD64MachineIntrinsicArtifactOnlyConsumer(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	artifact := buildSemanticArtifact(t, "Standard/Machine/AMD64.concept", string(source), nil)
	consumer := `module MachineConsumer; profile Core; import Standard.Machine.AMD64;
uint64 Probe() { Pause(); return ReadTimestamp(); }
uint32 Vendor() { CpuidResult result = Cpuid(0, 0); return result.ebx; }
void Prove() {
    Assert.Concept<NoAllocation>(Probe, "machine call has no allocation");
    Assert.Concept<HardwareRead>(Probe, "timestamp read survives import");
    Assert.Concept<Privileged>(Cli, "privilege classification survives import");
}`
	module, err := ParseWithSemanticModules("machine_consumer.concept", consumer, map[string][]byte{"Standard.Machine.AMD64": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := GenerateForTarget(module, []byte(consumer), X86_64GenericTarget())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["machine_consumer.machine.S"]), "ConceptAMD64ReadTimestamp:") || !strings.Contains(string(outputs["machine_consumer.machine.S"]), "cpuid") {
		t.Fatal("artifact-only consumer lost machine helper")
	}
}

func TestAMD64MachineArtifactsAreByteIdenticalAcross100Runs(t *testing.T) {
	source, err := os.ReadFile("../../libraries/Standard/Machine/AMD64.concept")
	if err != nil {
		t.Fatal(err)
	}
	var expected Outputs
	var expectedArtifact []byte
	for i := 0; i < determinismRuns(); i++ {
		module, err := Parse("amd64.concept", string(source))
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := GenerateForTarget(module, source, X86_64GenericTarget())
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := CompileSemanticModule("amd64.concept", string(source), nil)
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 && (!reflect.DeepEqual(outputs, expected) || !reflect.DeepEqual(artifact, expectedArtifact)) {
			t.Fatalf("machine output changed on run %d", i+1)
		}
		expected, expectedArtifact = outputs, artifact
	}
}
