package concept

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func standardMemorySource(t *testing.T, relative string) string {
	t.Helper()
	path := filepath.Join("..", "..", "libraries", filepath.FromSlash(relative))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}

func standardMemoryArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	artifacts := map[string][]byte{}
	artifacts["Standard.MemoryGeometry"] = buildSemanticArtifact(t, "Standard/MemoryGeometry.concept", standardMemorySource(t, "Standard/MemoryGeometry.concept"), nil)
	artifacts["Standard.Memory.Core"] = buildSemanticArtifact(t, "Standard/Memory/Core.concept", standardMemorySource(t, "Standard/Memory/Core.concept"), artifacts)
	artifacts["Standard.Memory.Bump"] = buildSemanticArtifact(t, "Standard/Memory/Bump.concept", standardMemorySource(t, "Standard/Memory/Bump.concept"), artifacts)
	artifacts["Standard.Memory.Pool"] = buildSemanticArtifact(t, "Standard/Memory/Pool.concept", standardMemorySource(t, "Standard/Memory/Pool.concept"), artifacts)
	artifacts["Standard.Memory.Ownership"] = buildSemanticArtifact(t, "Standard/Memory/Ownership.concept", standardMemorySource(t, "Standard/Memory/Ownership.concept"), artifacts)
	artifacts["Standard.Memory.MonotonicOwnership"] = buildSemanticArtifact(t, "Standard/Memory/MonotonicOwnership.concept", standardMemorySource(t, "Standard/Memory/MonotonicOwnership.concept"), artifacts)
	artifacts["Platform.Host.Memory"] = buildSemanticArtifact(t, "Standard/Platform/Host/Memory.concept", standardMemorySource(t, "Standard/Platform/Host/Memory.concept"), artifacts)
	artifacts["Standard.Memory.Host"] = buildSemanticArtifact(t, "Standard/Memory/Host.concept", standardMemorySource(t, "Standard/Memory/Host.concept"), artifacts)
	return artifacts
}

func TestStandardMemoryModulesCompileAsOrdinarySemanticArtifacts(t *testing.T) {
	standardMemoryArtifacts(t)
}

func TestStandardMemoryConcreteBumpResultRetainsBindableRegionFacts(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	_, err := ParseWithSemanticModules("ConcreteBump.concept", `module ConcreteBump;
profile Core;
import Standard.Memory.Bump;
Result<int, AllocationError> Use()
{
    FixedBufferMemorySource source = MakeFixedBufferMemorySource();
    MemoryRegion<SystemMemory> parent = RegionOf(ref source);
    BumpAllocator bump = MakeBumpAllocator(parent);
    MemoryRegion<SystemMemory> region = Allocate(ref bump, SizeOf<int>(), AlignOf<int>())?;
    Storage<int> storage = bind<int>(region);
    Initialize(storage, 9);
    int value = Value(storage);
    Destroy(storage);
    return Result::Ok(9);
}
`, artifacts)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStandardMemoryFixedBumpMonotonicRunsAsStrictC11(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `module FixedSpecimen;
profile Core;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;

struct Widget { int value; }

int Main()
{
    FixedBufferMemorySource source = MakeFixedBufferMemorySource();
    MemoryRegion<SystemMemory> region = RegionOf(ref source);
    BumpAllocator bump = MakeBumpAllocator(region);
    owned MonotonicAllocation<Widget, BumpAllocator> owner =
        AllocateMonotonic<Widget, BumpAllocator>(ref bump, Widget{41})!;
    ref Widget widget = MonotonicAllocationValue<Widget, BumpAllocator>(ref owner);
    widget.value = widget.value + 1;
    return widget.value;
}
`
	module, err := ParseWithSemanticModules("FixedSpecimen.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	implementation := moduleOutput(t, outputs, ".generated.c")
	if strings.Contains(implementation, "malloc") || strings.Contains(implementation, "free(") || strings.Contains(implementation, "memcpy") {
		t.Fatalf("fixed-buffer allocator path contains forbidden runtime machinery:\n%s", implementation)
	}
	runFoundationNativeHarness(t, outputs, "fixedspecimen_harness.c", "#include \"fixedspecimen.generated.h\"\nint main(void) { return concept_fixedspecimen_main() == 42 ? 0 : 1; }\n")
}

func TestStandardMemoryPoolTypedOwnerReleasesAndReuses(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `module PoolSpecimen;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;

int Main()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    owned Allocation<int, PoolAllocator> first = Allocate<int, PoolAllocator>(ref pool, 17)!;
    ref int firstValue = AllocationValue<int, PoolAllocator>(ref first);
    int observed = firstValue;
    Drop<int, PoolAllocator>(move first);
    owned Allocation<int, PoolAllocator> second = Allocate<int, PoolAllocator>(ref pool, 25)!;
    ref int secondValue = AllocationValue<int, PoolAllocator>(ref second);
    ref const int constSecondValue = ConstAllocationValue<int, PoolAllocator>(ref const second);
    return observed + constSecondValue;
}
`
	module, err := ParseWithSemanticModules("PoolSpecimen.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "poolspecimen_harness.c", "#include \"poolspecimen.generated.h\"\nint main(void) { return concept_poolspecimen_main() == 42 ? 0 : 1; }\n")
}

func TestStandardMemoryEffectsAndBumpCapabilitySplit(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	_, err := ParseWithSemanticModules("EffectSpecimen.concept", `module EffectSpecimen;
profile Core;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;
struct Widget { int value; }
int Build(ref BumpAllocator bump)
{
    owned MonotonicAllocation<Widget, BumpAllocator> owner = AllocateMonotonic<Widget, BumpAllocator>(ref bump, Widget{1})!;
    return MonotonicAllocationValue<Widget, BumpAllocator>(ref owner).value;
}
void Verify()
{
    Assert.Concept<NoAllocation>(Build, "building a widget allocates storage");
}
`, artifacts)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_DISPROVEN" {
		t.Fatalf("expected typed allocation to disprove NoAllocation, got %v", err)
	}

	_, err = ParseWithSemanticModules("BumpReleaseInvalid.concept", `module BumpReleaseInvalid;
profile Core;
import Standard.Memory.Bump;
requires ReleasableAllocator<BumpAllocator>;
`, artifacts)
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CV4153" {
		t.Fatalf("expected bump allocator to reject releasable capability, got %v", err)
	}

	_, err = ParseWithSemanticModules("EffectPositive.concept", `module EffectPositive;
profile Core;
import Standard.Memory.Bump;
void Verify()
{
    Assert.Concept<NoAllocation>(Remaining, "capacity inspection does not allocate");
}
`, artifacts)
	if err != nil {
		t.Fatalf("expected imported Remaining helper to prove NoAllocation: %v", err)
	}

	_, err = ParseWithSemanticModules("EffectUnknown.concept", `module EffectUnknown;
profile Core;
extern "C" void Opaque();
void Verify()
{
    Assert.Concept<NoAllocation>(Opaque, "opaque foreign effects remain unknown");
}
`, artifacts)
	if !errors.As(err, &diagnostic) || diagnostic.Code != "CONCEPT_ASSERT_UNKNOWN" {
		t.Fatalf("expected opaque foreign operation to leave NoAllocation unknown, got %v", err)
	}
}

func TestStandardMemoryArtifactsAreDeterministicAcross100Runs(t *testing.T) {
	first := standardMemoryArtifacts(t)
	consumerSource := `module DeterministicMemory;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;
int Main()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    owned Allocation<int, PoolAllocator> owner = Allocate<int, PoolAllocator>(ref pool, 42)!;
    return AllocationValue<int, PoolAllocator>(ref owner);
}
`
	consumer, err := ParseWithSemanticModules("DeterministicMemory.concept", consumerSource, first)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutputs, err := Generate(consumer, []byte(consumerSource))
	if err != nil {
		t.Fatal(err)
	}
	moduleRoot := filepath.Join("..", "..", "libraries")
	freshConsumer, err := ParseWithBuiltSemanticModuleRoots("DeterministicMemory.concept", consumerSource, []string{moduleRoot})
	if err != nil {
		t.Fatal(err)
	}
	freshOutputs, err := Generate(freshConsumer, []byte(consumerSource))
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range expectedOutputs {
		if !bytes.Equal(freshOutputs[name], expected) {
			t.Fatalf("fresh and artifact-only consumer output differ for %s", name)
		}
	}
	for run := 1; run < 100; run++ {
		again := standardMemoryArtifacts(t)
		for name, expected := range first {
			if !bytes.Equal(again[name], expected) {
				t.Fatalf("artifact %s changed on run %d", name, run)
			}
		}
		againConsumer, err := ParseWithSemanticModules("DeterministicMemory.concept", consumerSource, again)
		if err != nil {
			t.Fatal(err)
		}
		againOutputs, err := Generate(againConsumer, []byte(consumerSource))
		if err != nil {
			t.Fatal(err)
		}
		for name, expected := range expectedOutputs {
			if !bytes.Equal(againOutputs[name], expected) {
				t.Fatalf("consumer output %s changed on run %d", name, run)
			}
		}
	}
}

func TestStandardMemoryTypedOwnerMovesWithoutRelocationAndDropsBeforeRelease(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `module OwnershipSpecimen;
profile Core;
import Standard.Memory.Ownership;

extern "C" void ObserveDestroy(int value);
extern "C" void ObserveRelease();

struct Resource { int value; }
void Drop(owned Resource resource) { ObserveDestroy(resource.value); }

class TrackingAllocator
{
public:
    int<array>[1] backing;
    bool released;

    Result<MemoryRegion<SystemMemory>, AllocationError> Allocate(
        ref TrackingAllocator self,
        usize<byte> size,
        usize<byte> alignment
    )
    {
        TrackingAllocationBoundary();
        return Result::Ok(MemoryRegion<SystemMemory>{
            AddressOf<SystemMemory>(ref self.backing),
            size,
            alignment
        });
    }

    Result<void, AllocationError> Release(
        ref TrackingAllocator self,
        borrow const MemoryRegion<SystemMemory> region
    )
    {
        ObserveRelease();
        self.released = true;
        return Result::Ok();
    }
};

void TrackingAllocationBoundary() {}
requires compiler.Allocates(TrackingAllocationBoundary);
requires ReleasableAllocator<TrackingAllocator>;

TrackingAllocator MakeTrackingAllocator()
{
    int<array>[1] backing = [0];
    return TrackingAllocator{move backing, false};
}

Result<int, AllocationError> Run()
{
    TrackingAllocator allocator = MakeTrackingAllocator();
    owned Allocation<Resource, TrackingAllocator> first =
        Allocate<Resource, TrackingAllocator>(ref allocator, Resource{7})?;
    owned Allocation<Resource, TrackingAllocator> second = move first;
    ref Resource resource = AllocationValue<Resource, TrackingAllocator>(ref second);
    int observed = resource.value;
    return Result::Ok(7);
}

int Main() { return Run()!; }
`
	module, err := ParseWithSemanticModules("OwnershipSpecimen.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	implementation := moduleOutput(t, outputs, ".generated.c")
	if strings.Contains(implementation, "memcpy") || strings.Contains(implementation, "malloc") {
		t.Fatalf("typed-owner path relocates or secretly allocates:\n%s", implementation)
	}
	harness := `#include "ownershipspecimen.generated.h"
#include <stdio.h>
static int state = 0;
static int destroys = 0;
static int releases = 0;
void ObserveDestroy(int value) { if (value == 7 && state == 0) state = 1; destroys += 1; }
void ObserveRelease(void) { if (state == 1) state = 2; releases += 1; }
int main(void) {
  int value = concept_ownershipspecimen_main();
  if (!(value == 7 && state == 2 && destroys == 1 && releases == 1)) printf("value=%d state=%d destroys=%d releases=%d\n", value, state, destroys, releases);
  return value == 7 && state == 2 && destroys == 1 && releases == 1 ? 0 : 1;
}
`
	runFoundationNativeHarness(t, outputs, "ownershipspecimen_harness.c", harness)
}

func TestStandardMemoryImmovableObjectOwnerCanMove(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `module ImmovableSpecimen;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;

immovable struct Stable { int value; }

int Main()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<Stable>(), AlignOf<Stable>());
    MemoryRegion<SystemMemory> region = Allocate(ref pool, SizeOf<Stable>(), AlignOf<Stable>())!;
    Storage<Stable> storage = bind<Stable>(region);
    Initialize(storage, Stable{42});
    owned Allocation<Stable, PoolAllocator> first = Allocation<Stable, PoolAllocator>{ref pool, region, move storage};
    owned Allocation<Stable, PoolAllocator> second = move first;
    ref Stable stable = AllocationValue<Stable, PoolAllocator>(ref second);
    return stable.value;
}
`
	module, err := ParseWithSemanticModules("ImmovableSpecimen.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(moduleOutput(t, outputs, ".generated.c"), "memcpy") {
		t.Fatal("moving the allocation owner relocated the immovable object")
	}
	runFoundationNativeHarness(t, outputs, "immovablespecimen_harness.c", "#include \"immovablespecimen.generated.h\"\nint main(void) { return concept_immovablespecimen_main() == 42 ? 0 : 1; }\n")
}

func TestStandardMemoryHostedSourceFreesExactlyOnce(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `module HostSpecimen;
profile Core;
import Standard.Memory.Host;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;

struct Widget { int value; }

int Main()
{
    usize<byte> size = 64;
    usize<byte> alignment = 4;
    HostedMemorySource source = AcquireHostedMemorySource(size, alignment)!;
    MemoryRegion<SystemMemory> region = RegionOf(ref const source);
    BumpAllocator bump = MakeBumpAllocator(region);
    owned MonotonicAllocation<Widget, BumpAllocator> owner =
        AllocateMonotonic<Widget, BumpAllocator>(ref bump, Widget{42})!;
    ref Widget widget = MonotonicAllocationValue<Widget, BumpAllocator>(ref owner);
    return widget.value;
}
`
	module, err := ParseWithSemanticModules("HostSpecimen.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "hostspecimen.generated.h"
static _Alignas(16) uint8_t bytes[64];
static int frees = 0;
uint8_t* ConceptHostAllocate(size_t size, size_t alignment) { return size <= 64 && alignment <= 16 ? bytes : 0; }
void ConceptHostFree(uint8_t* address) { if (address == bytes) frees += 1; }
bool ConceptHostAddressIsNull(uint8_t* address) { return address == 0; }
int main(void) { int value = concept_hostspecimen_main(); return value == 42 && frees == 1 ? 0 : 1; }
`
	runFoundationNativeHarness(t, outputs, "hostspecimen_harness.c", harness)
}

func TestStandardMemoryDragonGodAndPrometheusSpecimensRunAsStrictC11(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	specimens := []struct {
		path      string
		source    string
		header    string
		expected  int
		forbidden []string
	}{
		{
			path: "DragonGodMemory.concept", header: "dragongodmemory.generated.h", expected: 42,
			forbidden: []string{"malloc", "calloc", "realloc", "free(", "memcpy", "GC", "RTTI"},
			source: `module DragonGodMemory;
profile Core;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;
struct BootInfo { int hartCount; int bootStage; }
int Main()
{
    FixedBufferMemorySource firmwareBacking = MakeFixedBufferMemorySource();
    BumpAllocator boot = MakeBumpAllocator(RegionOf(ref firmwareBacking));
    owned MonotonicAllocation<BootInfo, BumpAllocator> owner =
        AllocateMonotonic<BootInfo, BumpAllocator>(ref boot, BootInfo{4, 38})!;
    ref BootInfo info = MonotonicAllocationValue<BootInfo, BumpAllocator>(ref owner);
    return info.hartCount + info.bootStage;
}
`,
		},
		{
			path: "PrometheusMemory.concept", header: "prometheusmemory.generated.h", expected: 42,
			forbidden: []string{"malloc", "calloc", "realloc", "free(", "memcpy"},
			source: `module PrometheusMemory;
profile Core;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;
struct TensorBacking { int x0; int x1; int x2; int x3; }
int Main()
{
    FixedBufferMemorySource alignedBacking = MakeFixedBufferMemorySource();
    BumpAllocator tensors = MakeBumpAllocator(RegionOf(ref alignedBacking));
    owned MonotonicAllocation<TensorBacking, BumpAllocator> owner =
        AllocateMonotonic<TensorBacking, BumpAllocator>(ref tensors, TensorBacking{10, 11, 12, 9})!;
    ref TensorBacking tensor = MonotonicAllocationValue<TensorBacking, BumpAllocator>(ref owner);
    return tensor.x0 + tensor.x1 + tensor.x2 + tensor.x3;
}
`,
		},
	}
	for _, specimen := range specimens {
		t.Run(specimen.path, func(t *testing.T) {
			module, err := ParseWithSemanticModules(specimen.path, specimen.source, artifacts)
			if err != nil {
				t.Fatal(err)
			}
			outputs, err := Generate(module, []byte(specimen.source))
			if err != nil {
				t.Fatal(err)
			}
			implementation := moduleOutput(t, outputs, ".generated.c")
			for _, forbidden := range specimen.forbidden {
				if strings.Contains(implementation, forbidden) {
					t.Fatalf("generated firmware/tensor path contains forbidden %q", forbidden)
				}
			}
			harness := "#include \"" + specimen.header + "\"\nint main(void) { return concept_" + strings.TrimSuffix(strings.ToLower(specimen.header), ".generated.h") + "_main() == " + fmt.Sprint(specimen.expected) + " ? 0 : 1; }\n"
			runFoundationNativeHarness(t, outputs, strings.TrimSuffix(specimen.header, ".generated.h")+"_harness.c", harness)
		})
	}
}

func TestStandardMemoryAllocationOwnerCopyRejects(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	_, err := ParseWithSemanticModules("OwnerCopyInvalid.concept", `module OwnerCopyInvalid;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;
void Bad()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    Allocation<int, PoolAllocator> first = Allocate<int, PoolAllocator>(ref pool, 1)!;
    Allocation<int, PoolAllocator> copied = first;
}
`, artifacts)
	var diagnostic Diagnostic
	if !errors.As(err, &diagnostic) || (diagnostic.Code != "CV4133" && diagnostic.Code != "CV4501") {
		t.Fatalf("expected allocation owner copy rejection, got %v", err)
	}
}

func TestStandardMemoryOwnerAndBackingReferenceEscapesReject(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	invalid := map[string]string{
		"OwnerReferenceEscape.concept": `module OwnerReferenceEscape;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;
ref int Bad(ref PoolAllocator pool)
{
    owned Allocation<int, PoolAllocator> owner = Allocate<int, PoolAllocator>(ref pool, 1)!;
    return AllocationValue<int, PoolAllocator>(ref owner);
}
`,
		"BackingReferenceEscape.concept": `module BackingReferenceEscape;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;
Result<Allocation<int, PoolAllocator>, AllocationError> Bad()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    return Allocate<int, PoolAllocator>(ref pool, 1);
}
`,
	}
	for path, source := range invalid {
		t.Run(path, func(t *testing.T) {
			if _, err := ParseWithSemanticModules(path, source, artifacts); err == nil {
				t.Fatal("expected lifetime escape rejection")
			}
		})
	}
}
