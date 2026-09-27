package concept

import (
	"fmt"
	"strings"
	"testing"
)

func TestVerifyPoolReleasePoisonsDeadBytesWithoutChangingLayout(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `module PoolPoison;
profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;

struct Tracked { int value; }
extern "C" void ObserveDestroy(int value);
void Drop(owned Tracked item) { ObserveDestroy(item.value); }

static_assert(SizeOf<PoolAllocator>() == 1056, "pool geometry remains semantic");
static_assert(AlignOf<PoolAllocator>() == 8, "pool alignment remains semantic");

PoolAllocator MakeDropped()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<Tracked>(), AlignOf<Tracked>());
    owned Allocation<Tracked, PoolAllocator> owner = Allocate<Tracked, PoolAllocator>(ref pool, Tracked{17})!;
    Drop<Tracked, PoolAllocator>(move owner);
    return pool;
}

int Reuse()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    owned Allocation<int, PoolAllocator> first = Allocate<int, PoolAllocator>(ref pool, 17)!;
    Drop<int, PoolAllocator>(move first);
    owned Allocation<int, PoolAllocator> second = Allocate<int, PoolAllocator>(ref pool, 42)!;
    ref int value = AllocationValue<int, PoolAllocator>(ref second);
    return value;
}`
	module, err := ParseWithSemanticModules("PoolPoison.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	var normalHeader []byte
	for _, tc := range []struct {
		name   string
		policy CompilationPolicy
		poison bool
	}{
		{"normal", ConservativeCompilationPolicy(), false},
		{"verify", VerifyCompilationPolicy(), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			outputs, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), tc.policy)
			if err != nil {
				t.Fatal(err)
			}
			if !tc.poison {
				normalHeader = outputs["poolpoison.generated.h"]
			} else if string(normalHeader) != string(outputs["poolpoison.generated.h"]) {
				t.Fatal("Verify changed PoolAllocator's generated C representation")
			}
			implementation := moduleOutput(t, outputs, ".generated.c")
			if strings.Contains(implementation, "concept_verify_poison_released_region(") != tc.poison {
				t.Fatalf("pool poison lowering does not match %s mode", tc.name)
			}
			if tc.poison {
				for run := 2; run <= 100; run++ {
					repeated, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), tc.policy)
					if err != nil || !equalOutputs(outputs, repeated) {
						t.Fatalf("Verify pool output changed on run %d: %v", run, err)
					}
				}
			}
			harness := fmt.Sprintf(`#include "poolpoison.generated.h"
#include <stddef.h>
static int destroy_seen = 0;
void ObserveDestroy(int value) { if (value == 17) destroy_seen = 1; }
_Static_assert(sizeof(concept_pool_allocator) == 1056, "pool C and semantic size differ");
_Static_assert(_Alignof(concept_pool_allocator) == 8, "pool C and semantic alignment differ");
int main(void) {
    concept_pool_allocator pool = concept_pool_poison_make_dropped();
    if (destroy_seen != 1) return 3;
    const unsigned char *bytes = (const unsigned char *)&pool.backing;
    for (size_t index = 0; index < 4; ++index) {
        if ((bytes[index] == 0xDD) != %d) return 1;
    }
    if (concept_pool_poison_reuse() != 42) return 2;
    return 0;
}
`, map[bool]int{false: 0, true: 1}[tc.poison])
			runFoundationNativeHarness(t, outputs, "poolpoison_harness.c", harness)
		})
	}
}

func TestVerifyPoisonRejectsDeviceMemory(t *testing.T) {
	source := `profile Core;
struct DeviceMemory {}
void Bad(Address<DeviceMemory> address, usize<byte> length)
{
    VerifyPoisonReleasedRegion(address, length);
}`
	_, err := Parse("device_poison.concept", source)
	if err == nil || !strings.Contains(err.Error(), "VERIFY_POISON_ARGUMENTS") {
		t.Fatalf("device memory was accepted for release poisoning: %v", err)
	}
}

func TestPoolOwnerAccessAfterDropIsRejected(t *testing.T) {
	artifacts := standardMemoryArtifacts(t)
	source := `profile Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;
void Bad()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    owned Allocation<int, PoolAllocator> owner = Allocate<int, PoolAllocator>(ref pool, 17)!;
    Drop<int, PoolAllocator>(move owner);
    ref int stale = AllocationValue<int, PoolAllocator>(ref owner);
}`
	_, err := ParseWithSemanticModules("stale_pool_owner.concept", source, artifacts)
	if err == nil || !strings.Contains(err.Error(), "CV4502") {
		t.Fatalf("access after Pool owner Drop was not rejected: %v", err)
	}
}
