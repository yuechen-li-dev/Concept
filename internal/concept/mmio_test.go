package concept

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const mmioSource = `module HardwareProbe;
profile Core;
struct DeviceMemory {}

bits Status : uint8
{
    ready: 0;
    mode: 1..3;
    empty: 5;
}

bits HighFlags : uint64
{
    high: 63;
}

uint8 Probe(Address<DeviceMemory> address)
{
    MmioLoad<uint8>(address);
    uint8 first = MmioLoad<uint8>(address);
    uint8 second = MmioLoad<uint8>(address);
    Status value = Status{second};
    Status changed = value with { mode = 3; ready = true; };
    MmioStore<uint8>(address, changed.raw);
    MmioStore<uint8>(address, changed.raw);
    return first;
}

void Wide(Address<DeviceMemory> address)
{
    uint16 half = MmioLoad<uint16>(address);
    uint32 word = MmioLoad<uint32>(address);
    uint64 whole = MmioLoad<uint64>(address);
    MmioStore<uint16>(address, half);
    MmioStore<uint32>(address, word);
    MmioStore<uint64>(address, whole);
}

void Ordered(Address<DeviceMemory> first, Address<DeviceMemory> second, Address<DeviceMemory> third)
{
    MmioStore<uint8>(first, 1);
    MmioLoad<uint8>(second);
    MmioStore<uint8>(third, 3);
}

uint8 DecodeMode(uint8 raw)
{
    Status value = Status{raw};
    return value.mode;
}

bool IsEmpty(uint8 raw)
{
    Status value = Status{raw};
    return value.empty;
}

bool IsHigh(uint64 raw)
{
    HighFlags value = HighFlags{raw};
    return value.high;
}

bool SameStatus(uint8 left, uint8 right)
{
    return Status{left} == Status{right};
}

uint8 ReplaceMode(uint8 raw, uint8 mode)
{
    Status value = Status{raw};
    Status changed = value with { mode = mode; };
    return changed.raw;
}

void Prove()
{
    Assert.Concept<HardwareRead>(Probe, "probe retains each read");
    Assert.Concept<HardwareWrite>(Probe, "probe retains each write");
    Assert.Concept<NoAllocation>(Probe, "scalar MMIO allocates no storage");
}
`

func TestMmioAndBitsReachMIRPlannerAndC11(t *testing.T) {
	module, err := Parse("hardware_probe.concept", mmioSource)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Structs) != 3 || module.Structs[1].BitsRepresentation != "uint8" || len(module.Structs[1].BitFields) != 3 {
		t.Fatalf("bits layout missing: %+v", module.Structs)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	if size, align, err := evt1TypeGeometry(env, Type{Name: "Status", Kind: TypeStruct}); err != nil || size != 1 || align != 1 {
		t.Fatalf("bits scalar geometry: size=%d align=%d err=%v", size, align, err)
	}
	if size, align, err := evt1TypeGeometry(env, Type{Name: "HighFlags", Kind: TypeStruct}); err != nil || size != 8 || align != 8 {
		t.Fatalf("64-bit bits scalar geometry: size=%d align=%d err=%v", size, align, err)
	}
	outputs, err := Generate(module, []byte(mmioSource))
	if err != nil {
		t.Fatal(err)
	}
	body := string(outputs["hardware_probe.generated.c"])
	if strings.Contains(string(outputs["hardware_probe.generated.h"]), "volatile") {
		t.Fatal("volatile leaked into ordinary generated source types")
	}
	if !strings.Contains(body, "volatile uint8_t") || !strings.Contains(body, "volatile uint16_t") || !strings.Contains(body, "volatile uint32_t") || !strings.Contains(body, "volatile uint64_t") {
		t.Fatalf("MMIO widths missing from C:\n%s", body)
	}
	var mir MIR
	if err := json.Unmarshal(outputs["hardware_probe.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	reads, writes := 0, 0
	var probeOrder []string
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "mmio_read" {
				reads++
				if fn.Name == "Probe" {
					probeOrder = append(probeOrder, op.Kind)
				}
			}
			if op.Kind == "mmio_write" {
				writes++
				if fn.Name == "Probe" {
					probeOrder = append(probeOrder, op.Kind)
				}
			}
		}
	}
	if reads != 7 || writes != 7 {
		t.Fatalf("MMIO operations lost: %d reads, %d writes", reads, writes)
	}
	if !reflect.DeepEqual(probeOrder, []string{"mmio_read", "mmio_read", "mmio_read", "mmio_write", "mmio_write"}) {
		t.Fatalf("MMIO MIR order changed: %v", probeOrder)
	}
	if mir.Structs[1].BitsRepresentation != "uint8" || len(mir.Structs[1].BitFields) != 3 {
		t.Fatal("MIR lost deterministic bits layout")
	}
	plan, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plan), "RetainOrderedVolatileAccess") {
		t.Fatal("planner did not retain MMIO")
	}
	runHostedMmioHarness(t, outputs)
}

func runHostedMmioHarness(t *testing.T, outputs Outputs) {
	t.Helper()
	compilers := r7kCCompilers()
	if len(compilers) == 0 {
		t.Skip("Clang and GCC unavailable")
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	adapter := `#ifndef MMIO_ADAPTER_H
#define MMIO_ADAPTER_H
#include <stdint.h>
typedef struct { uintptr_t address; int width; uint64_t value; int write; } Trace;
extern Trace trace[16];
extern int trace_count;
uint64_t fake_read(uintptr_t address, int width);
void fake_write(uintptr_t address, int width, uint64_t value);
#define CONCEPT_MMIO_READ_UINT8(a) ((uint8_t)fake_read((a), 8))
#define CONCEPT_MMIO_READ_UINT16(a) ((uint16_t)fake_read((a), 16))
#define CONCEPT_MMIO_READ_UINT32(a) ((uint32_t)fake_read((a), 32))
#define CONCEPT_MMIO_READ_UINT64(a) ((uint64_t)fake_read((a), 64))
#define CONCEPT_MMIO_WRITE_UINT8(a,v) fake_write((a), 8, (v))
#define CONCEPT_MMIO_WRITE_UINT16(a,v) fake_write((a), 16, (v))
#define CONCEPT_MMIO_WRITE_UINT32(a,v) fake_write((a), 32, (v))
#define CONCEPT_MMIO_WRITE_UINT64(a,v) fake_write((a), 64, (v))
#endif
`
	simulator := `#include "adapter.h"
Trace trace[16];
int trace_count;
uint64_t fake_read(uintptr_t address, int width) {
    uint64_t value = width == 8 ? (uint64_t)(trace_count == 0 ? 1 : trace_count == 1 ? 2 : 0x20) : width == 16 ? 0x1234 : width == 32 ? 0x87654321 : 0x123456789abcdef0;
    trace[trace_count++] = (Trace){address, width, value, 0};
    return value;
}
void fake_write(uintptr_t address, int width, uint64_t value) {
    trace[trace_count++] = (Trace){address, width, value, 1};
}
`
	harness := `#include "hardware_probe.generated.h"
#include "adapter.h"
int main(int argc, char **argv) {
    if (argc > 1 && argv[1][0] == 'b') {
        concept_hardware_probe_replace_mode(0x80, 8);
        return 15;
    }
    uintptr_t address = (uintptr_t)0x1000;
    if (concept_hardware_probe_probe(address) != 2) return 1;
    if (trace_count != 5) return 2;
    for (int i = 0; i < 3; ++i) if (trace[i].write || trace[i].width != 8 || trace[i].address != address) return 3;
    for (int i = 3; i < 5; ++i) if (!trace[i].write || trace[i].width != 8 || trace[i].value != 0x27) return 4;
    concept_hardware_probe_wide(address);
    if (trace_count != 11) return 5;
    for (int i = 5; i < 8; ++i) if (trace[i].write || trace[i].width != (16 << (i-5))) return 6;
    for (int i = 8; i < 11; ++i) if (!trace[i].write || trace[i].width != (16 << (i-8)) || trace[i].value != trace[i-3].value) return 7;
    if (concept_hardware_probe_decode_mode(0x26) != 3) return 8;
    if (!concept_hardware_probe_is_empty(0x20)) return 9;
    if (!concept_hardware_probe_is_high(UINT64_C(0x8000000000000000))) return 10;
    if (!concept_hardware_probe_same_status(0x27, 0x27) || concept_hardware_probe_same_status(0x27, 0x26)) return 16;
    if (concept_hardware_probe_replace_mode(0x80, 5) != 0x8a) return 15;
    concept_hardware_probe_ordered(address, address + 1u, address + 2u);
    if (trace_count != 14) return 11;
    if (!trace[11].write || trace[11].address != address || trace[11].value != 1) return 12;
    if (trace[12].write || trace[12].address != address + 1u || trace[12].width != 8) return 13;
    if (!trace[13].write || trace[13].address != address + 2u || trace[13].value != 3) return 14;
    return 0;
}
`
	for name, source := range map[string]string{"adapter.h": adapter, "simulator.c": simulator, "harness.c": harness} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, compiler := range compilers {
		exe := filepath.Join(dir, filepath.Base(compiler)+"-mmio.exe")
		command := exec.Command(compiler, "-std=c11", "-pedantic", "-pedantic-errors", "-O2", "-Wall", "-Wextra", "-include", filepath.Join(dir, "adapter.h"), filepath.Join(dir, "hardware_probe.generated.c"), filepath.Join(dir, "simulator.c"), filepath.Join(dir, "harness.c"), "-o", exe)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11 MMIO build: %v\n%s", compiler, err, output)
		}
		for i := 0; i < 100; i++ {
			if output, err := exec.Command(exe).CombinedOutput(); err != nil {
				t.Fatalf("%s hosted device trace run %d: %v\n%s", compiler, i, err, output)
			}
		}
		if output, err := exec.Command(exe, "bad").CombinedOutput(); err == nil || !strings.Contains(string(output), "bits field value out of range") {
			t.Fatalf("%s dynamic bits range check: err=%v\n%s", compiler, err, output)
		}
	}
}

func r7kCCompilers() []string {
	var available []string
	for _, name := range []string{"clang", "gcc"} {
		if path, err := exec.LookPath(name); err == nil {
			available = append(available, path)
		}
	}
	return available
}

func TestMmioArtifactsAndPlanAreByteIdenticalAcross100Runs(t *testing.T) {
	var reference Outputs
	var referencePlan, referenceArtifact []byte
	for i := 0; i < 100; i++ {
		module, err := Parse("hardware_probe.concept", mmioSource)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(mmioSource))
		if err != nil {
			t.Fatal(err)
		}
		plan, err := GeneratePlan(module, GenericC11Target())
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := CompileSemanticModule("hardware_probe.concept", mmioSource, nil)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			reference, referencePlan, referenceArtifact = outputs, plan, artifact
			continue
		}
		if !reflect.DeepEqual(reference, outputs) || !reflect.DeepEqual(referencePlan, plan) || !reflect.DeepEqual(referenceArtifact, artifact) {
			t.Fatalf("MMIO MIR/C/proof/plan/artifact changed on run %d", i)
		}
	}
}

func TestOrdinaryMemoryDoesNotAcquireMmioSemantics(t *testing.T) {
	source := `profile Core;
uint8 ReadOrdinary(ref const uint8 value)
{
    return value;
}`
	module, err := Parse("ordinary_memory.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(outputs["ordinary_memory.generated.c"]), "volatile") || strings.Contains(string(outputs["ordinary_memory.generated.h"]), "volatile") {
		t.Fatal("ordinary memory was lowered as volatile")
	}
	var mir MIR
	if err := json.Unmarshal(outputs["ordinary_memory.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			if op.Kind == "mmio_read" || op.Kind == "mmio_write" {
				t.Fatal("ordinary memory became a hardware transaction")
			}
		}
	}
}

func TestBitsInvalidLayouts(t *testing.T) {
	for _, tc := range []struct{ body, code string }{
		{"bits X : uint8 { a: 7..8; }", "BITS_RANGE_INVALID"},
		{"bits X : uint8 { a: 3..2; }", "BITS_RANGE_INVALID"},
		{"bits X : uint8 { a: 1..3; b: 3; }", "BITS_FIELD_OVERLAP"},
		{"bits X : uint8 { a: 1; a: 2; }", "BITS_FIELD_DUPLICATE"},
		{"bits X : uint8 { mode: 1..3; } X Bad() { X value = X{0}; return value with { mode = 8; }; }", "BITS_FIELD_VALUE_OUT_OF_RANGE"},
	} {
		_, err := Parse("bad_bits.concept", "profile Core; "+tc.body)
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
			t.Fatalf("%s: expected %s, got %v", tc.body, tc.code, err)
		}
	}
}

func TestMmioRejectsWrongWidthAndAddressSpace(t *testing.T) {
	for _, tc := range []struct{ body, code string }{
		{"void Bad(Address<SystemMemory> address) { MmioLoad<uint32>(address); }", "MMIO_DEVICE_ADDRESS_REQUIRED"},
		{"void Bad(Address<DeviceMemory> address) { MmioLoad<int>(address); }", "MMIO_WIDTH_INVALID"},
		{"void Bad(Address<DeviceMemory> address) { MmioLoad<UartLike>(address); }", "MMIO_WIDTH_INVALID"},
		{"void Bad(ref uint32 value) { MmioLoad<uint32>(value); }", "MMIO_DEVICE_ADDRESS_REQUIRED"},
		{"void Bad(Address<DeviceMemory> address) { MmioStore<uint8>(address, 256); }", "CV4644"},
		{"void Bad() { const usize bits = 3; MmioLoad<uint32>(AddressFromBits<DeviceMemory>(bits)); }", "MMIO_ALIGNMENT_INVALID"},
		{"void Bad(Address<DeviceMemory> address) { Address<SystemMemory> ordinary = address; }", "CV4106"},
		{"void Bad(Address<DeviceMemory> address) { Address<DeviceMemory> invalid = address + address; }", "ADDRESS_AFFINE_INVALID"},
	} {
		source := "profile Core; struct SystemMemory {} struct DeviceMemory {} record struct UartLike { uint32 raw; } " + tc.body
		_, err := Parse("bad_mmio.concept", source)
		var diagnostic Diagnostic
		if !errors.As(err, &diagnostic) || diagnostic.Code != tc.code {
			t.Fatalf("%s: expected %s, got %v", tc.body, tc.code, err)
		}
	}
}

func TestDeviceRegionUsesExistingForeignAuthority(t *testing.T) {
	geometrySource, err := os.ReadFile("../../libraries/Standard/MemoryGeometry.concept")
	if err != nil {
		t.Fatal(err)
	}
	mmioSource, err := os.ReadFile("../../libraries/Standard/Hardware/Mmio.concept")
	if err != nil {
		t.Fatal(err)
	}
	deps := map[string][]byte{
		"Standard.MemoryGeometry": buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", string(geometrySource), nil),
		"Standard.Hardware.Mmio":  buildSemanticArtifact(t, "Standard/Hardware/Mmio.concept", string(mmioSource), nil),
	}
	source := `module DeviceRegionFixture;
profile Core;
import Standard.MemoryGeometry;
import Standard.Hardware.Mmio;
extern "C" byte* DeviceRegionAddress(usize<byte> length, usize<byte> alignment);
foreign concept DeviceRegionContract on DeviceRegionAddress
{
    requires compiler.ExternalStorage<DeviceMemory>(result, length, alignment);
}
struct DeviceRegionLease
{
    byte* address;
    usize<byte> length;
    usize<byte> alignment;
}
MemoryRegion<DeviceMemory> RegionOfDevice(ref const DeviceRegionLease lease)
{
    return EstablishExternalRegion<DeviceMemory>(lease, lease.address, lease.length, lease.alignment, "DeviceRegionContract");
}`
	module, err := ParseWithSemanticModules("DeviceRegionFixture.concept", source, deps)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(module, []byte(source)); err != nil {
		t.Fatal(err)
	}
}

func TestUartBitsAndHardwareEffectsSurviveArtifactOnlyImport(t *testing.T) {
	standardSource, err := os.ReadFile("../../libraries/Standard/Hardware/Mmio.concept")
	if err != nil {
		t.Fatal(err)
	}
	standard := buildSemanticArtifact(t, "Standard/Hardware/Mmio.concept", string(standardSource), nil)
	uartSource, err := os.ReadFile("../../libraries/DragonGod/Hardware/Uart.concept")
	if err != nil {
		t.Fatal(err)
	}
	deps := map[string][]byte{"Standard.Hardware.Mmio": standard}
	uart := buildSemanticArtifact(t, "DragonGod/Hardware/Uart.concept", string(uartSource), deps)
	var artifact SemanticModuleArtifact
	if err := json.Unmarshal(uart, &artifact); err != nil {
		t.Fatal(err)
	}
	foundRead, foundWrite := false, false
	for _, summary := range artifact.HardwareEffects {
		if summary.Operation == "ReadStatus" {
			foundRead = summary.Read
		}
		if summary.Operation == "WriteByte" {
			foundWrite = summary.Write
		}
	}
	if !foundRead || !foundWrite {
		t.Fatalf("UART hardware effects missing from artifact: %+v", artifact.HardwareEffects)
	}
	consumer := `module UartConsumer;
profile Core;
import DragonGod.Hardware.Uart;
bool Observe(ref const Uart uart)
{
    UartLineStatus status = ReadStatus(ref const uart);
    return status.transmitterEmpty;
}
void Prove()
{
    Assert.Concept<HardwareRead>(ReadStatus, "imported UART read remains observable");
    Assert.Concept<HardwareWrite>(WriteByte, "imported UART write remains observable");
}`
	deps["DragonGod.Hardware.Uart"] = uart
	module, err := ParseWithSemanticModules("UartConsumer.concept", consumer, deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(module.Structs) < 2 {
		t.Fatal("imported UART bits type missing")
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputs["uartconsumer.generated.c"]), "CONCEPT_MMIO_READ_UINT8") {
		t.Fatal("artifact-only consumer lost MMIO lowering")
	}
	runHostedUartHarness(t, module, outputs)
}

func runHostedUartHarness(t *testing.T, module Module, outputs Outputs) {
	t.Helper()
	compilers := r7kCCompilers()
	if len(compilers) == 0 {
		t.Skip("Clang and GCC unavailable")
	}
	dir := t.TempDir()
	if err := Write(dir, outputs); err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	base := evt1SemanticSymbolBase(module)
	read := evt1FunctionSymbolForDecl(base, env, env.functions["ReadStatus"][0])
	write := evt1FunctionSymbolForDecl(base, env, env.functions["WriteByte"][0])
	poll := evt1FunctionSymbolForDecl(base, env, env.functions["PollTransmitter"][0])
	harness := fmt.Sprintf(`#include "uartconsumer.generated.h"
int main(void) {
    uint8_t ports[8] = {0};
    concept_uart uart = {.base = (uintptr_t)&ports[0]};
    ports[5] = 0x20;
    if (%s(&uart).raw != 0x20) return 1;
    if (!%s(&uart)) return 2;
    %s(&uart, 0x5a);
    if (ports[0] != 0x5a) return 3;
    ports[5] = 0;
    if (%s(&uart)) return 4;
    return 0;
}
`, read, poll, write, poll)
	harnessPath := filepath.Join(dir, "uart_harness.c")
	if err := os.WriteFile(harnessPath, []byte(harness), 0644); err != nil {
		t.Fatal(err)
	}
	for _, compiler := range compilers {
		exe := filepath.Join(dir, filepath.Base(compiler)+"-uart.exe")
		command := exec.Command(compiler, "-std=c11", "-pedantic", "-pedantic-errors", "-O2", "-I", dir, filepath.Join(dir, "uartconsumer.generated.c"), harnessPath, "-o", exe)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s strict C11 UART build: %v\n%s", compiler, err, output)
		}
		if output, err := exec.Command(exe).CombinedOutput(); err != nil {
			t.Fatalf("%s hosted UART specimen: %v\n%s", compiler, err, output)
		}
	}
}
