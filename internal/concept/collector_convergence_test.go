package concept

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func r7fRCollectionArtifacts(t *testing.T) map[string][]byte {
	t.Helper()
	artifacts := standardMemoryArtifacts(t)
	artifacts["Standard.Collection.Core"] = buildSemanticArtifact(t,
		"Standard/Collection/Core.concept", standardMemorySource(t, "Standard/Collection/Core.concept"), artifacts)
	return artifacts
}

func TestR7fRCollectorHundredRunDeterminism(t *testing.T) {
	if os.Getenv("CONCEPT_R7FR_DETERMINISM") != "1" {
		t.Skip("run explicitly for the R7fR 100-run gate")
	}
	root := filepath.Join("..", "..", "libraries")
	out := t.TempDir()
	dependencies := r7fRCollectionArtifacts(t)
	source := `module CollectorDeterminism; profile Core;
import Standard.Collection.Core; import Standard.Memory.Pool;
void TraceReferences(ref const int value, ref TraceVisitor<int> visitor) { }
int Main() { PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>()); MarkSweepCollector<int, 2, PoolAllocator> collector = MakeCollector<int, 2, PoolAllocator>(ref pool, 7); CollectorHandle<int> handle = AllocateObject<int, 2, PoolAllocator>(ref collector, 5)!; return Collect<int, 2, PoolAllocator>(ref collector)!; }`
	var firstGraph, firstArtifact []byte
	var firstMIR, firstC string
	for i := 0; i < 100; i++ {
		graph, err := BuildPackage(root, out, "Standard")
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := MarshalPackageGraph(graph)
		if err != nil {
			t.Fatal(err)
		}
		artifact, err := os.ReadFile(filepath.Join(out, "Standard", "modules", "Standard", "Collection", "Core.concept-module.json"))
		if err != nil {
			t.Fatal(err)
		}
		dependencies["Standard.Collection.Core"] = artifact
		module, err := ParseWithSemanticModules("CollectorDeterminism.concept", source, dependencies)
		if err != nil {
			t.Fatal(err)
		}
		outputs, err := Generate(module, []byte(source))
		if err != nil {
			t.Fatal(err)
		}
		mir, c := moduleOutput(t, outputs, ".mir.json"), moduleOutput(t, outputs, ".generated.c")
		if i == 0 {
			firstGraph, firstArtifact, firstMIR, firstC = encoded, artifact, mir, c
			continue
		}
		if !bytes.Equal(encoded, firstGraph) || !bytes.Equal(artifact, firstArtifact) || mir != firstMIR || c != firstC {
			t.Fatalf("collector output changed on run %d: graph=%v artifact=%v MIR=%v C=%v", i+1, !bytes.Equal(encoded, firstGraph), !bytes.Equal(artifact, firstArtifact), mir != firstMIR, c != firstC)
		}
	}
}

func TestR7fRCollectorPackageFactRunsNative(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	if _, err := BuildPackage(root, t.TempDir(), "Standard"); err != nil {
		t.Fatal(err)
	}
	manifest, err := DiscoverTests(filepath.Join(root, "Standard"))
	if err != nil {
		t.Fatal(err)
	}
	run, err := RunTests(manifest, TestRunOptions{Filter: "Collector"})
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Results) == 0 {
		t.Fatal("collector fact was not discovered")
	}
	for _, result := range run.Results {
		if result.Status != "PASS" {
			t.Fatalf("%s: %s: %+v: %s", result.TestID, result.Status, result.Failure, strings.TrimSpace(result.Stderr))
		}
	}
}

func TestR7fRCollectorArtifactOnlyConsumer(t *testing.T) {
	artifacts := r7fRCollectionArtifacts(t)
	source := `module ArtifactCollectorConsumer;
profile Core;
import Standard.Collection.Core;
import Standard.Memory.Pool;
import Standard.Memory.Ownership;
void TraceReferences(ref const int value, ref TraceVisitor<int> visitor) { }
int CollectOnce(ref MarkSweepCollector<int, 2, PoolAllocator> collector)
{
    return Collect<int, 2, PoolAllocator>(ref collector)!;
}
requires compiler.InvalidatesBorrows(CollectOnce, collector);
int Main()
{
    Assert.Concept<NoAllocation>(CollectOnce, "collection uses only bounded local metadata and release");
    PoolAllocator pool = MakePoolAllocator(SizeOf<int>(), AlignOf<int>());
    int ordinary = 3;
    owned Allocation<int, PoolAllocator> manual = Allocate<int, PoolAllocator>(ref pool, 4)!;
    MarkSweepCollector<int, 2, PoolAllocator> collector = MakeCollector<int, 2, PoolAllocator>(ref pool, 72);
    CollectorHandle<int> handle = AllocateObject<int, 2, PoolAllocator>(ref collector, 19)!;
    int root = RegisterRoot<int, 2, PoolAllocator>(ref collector, handle)!;
    int first = CollectOnce(ref collector);
    if (first != 0) { return 1; }
    {
        ref const int value = Borrow<int, 2, PoolAllocator>(ref const collector, handle)!;
        if (value != 19) { return 2; }
    }
    UnregisterRoot<int, 2, PoolAllocator>(ref collector, root);
    int second = CollectOnce(ref collector);
    if (second != 1) { return 3; }
    if (ordinary != 3) { return 4; }
    {
        ref int manualValue = AllocationValue<int, PoolAllocator>(ref manual);
        if (manualValue != 4) { return 5; }
    }
    Drop<int, PoolAllocator>(move manual);
    return 0;
}`
	module, err := ParseWithSemanticModules("ArtifactCollectorConsumer.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"malloc(", "calloc(", "realloc(", "gc_register", "stack_scan", "runtime_reflection", "MIRGC", "MIRMark", "MIRSweep", "MIRRoot"} {
		if strings.Contains(moduleOutput(t, outputs, ".generated.c"), forbidden) || strings.Contains(moduleOutput(t, outputs, ".mir.json"), forbidden) {
			t.Fatalf("collector output contains hidden runtime or opcode %q", forbidden)
		}
	}
	runFoundationNativeHarness(t, outputs, "artifact_collector_harness.c", "#include \"artifactcollectorconsumer.generated.h\"\nint main(void) { return concept_artifact_collector_consumer_main(); }\n")

	invalid := strings.Replace(source, "    {\n        ref const int value", "    ref const int held = Borrow<int, 2, PoolAllocator>(ref const collector, handle)!;\n    {\n        ref const int value", 1)
	invalid = strings.Replace(invalid, "    return 0;", "    return held;", 1)
	_, err = ParseWithSemanticModules("InvalidCollectorBorrow.concept", invalid, artifacts)
	if err == nil || !strings.Contains(err.Error(), "DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW") {
		t.Fatalf("live borrow overlapping Collect = %v", err)
	}
	awaitInvalid := `module CollectorAwaitInvalid; profile Core;
import Standard.Collection.Core; import Standard.Memory.Pool;
async int Child() { return 1; }
async int Work(ref const MarkSweepCollector<int, 2, PoolAllocator> collector, CollectorHandle<int> handle)
{ ref const int value = Borrow<int, 2, PoolAllocator>(ref const collector, handle)!; int step = await Child(); return value + step; }`
	_, err = ParseWithSemanticModules("CollectorAwaitInvalid.concept", awaitInvalid, artifacts)
	if err == nil || !strings.Contains(err.Error(), "ASYNC_PERSISTENT_REF_ESCAPE") {
		t.Fatalf("collector borrow across await = %v", err)
	}
}

func TestR7fRCollectorExactDropAndRelease(t *testing.T) {
	artifacts := r7fRCollectionArtifacts(t)
	source := `module CollectorDropProof;
profile Core;
import Standard.Collection.Core;
import Standard.Memory.Pool;
extern "C" void ObserveDrop(int id);
struct Resource { int id; }
void Drop(owned Resource resource) { ObserveDrop(resource.id); }
void TraceReferences(ref const Resource value, ref TraceVisitor<Resource> visitor) { }
void LeaveLive()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<Resource>(), AlignOf<Resource>());
    MarkSweepCollector<Resource, 1, PoolAllocator> collector = MakeCollector<Resource, 1, PoolAllocator>(ref pool, 54);
    CollectorHandle<Resource> handle = AllocateObject<Resource, 1, PoolAllocator>(ref collector, Resource{3})!;
    int root = RegisterRoot<Resource, 1, PoolAllocator>(ref collector, handle)!;
}
int Main()
{
    PoolAllocator pool = MakePoolAllocator(SizeOf<Resource>(), AlignOf<Resource>());
    MarkSweepCollector<Resource, 2, PoolAllocator> collector = MakeCollector<Resource, 2, PoolAllocator>(ref pool, 53);
    CollectorHandle<Resource> first = AllocateObject<Resource, 2, PoolAllocator>(ref collector, Resource{1})!;
    CollectorHandle<Resource> second = AllocateObject<Resource, 2, PoolAllocator>(ref collector, Resource{2})!;
    int root = RegisterRoot<Resource, 2, PoolAllocator>(ref collector, first)!;
    if (Collect<Resource, 2, PoolAllocator>(ref collector)! != 1) { return 1; }
    if (not pool.allocated[0] or pool.allocated[1]) { return 2; }
    UnregisterRoot<Resource, 2, PoolAllocator>(ref collector, root);
    if (Collect<Resource, 2, PoolAllocator>(ref collector)! != 1) { return 3; }
    if (pool.allocated[0] or pool.allocated[1]) { return 4; }
    if (Collect<Resource, 2, PoolAllocator>(ref collector)! != 0) { return 5; }
    LeaveLive();
    return 0;
}`
	module, err := ParseWithSemanticModules("CollectorDropProof.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "collectordropproof.generated.h"
static int drops = 0;
static int first = 0;
static int second = 0;
static int third = 0;
void ObserveDrop(int id) { if (drops == 0) first = id; else if (drops == 1) second = id; else third = id; drops++; }
int main(void) { int result = concept_collector_drop_proof_main(); return result == 0 && drops == 3 && first == 2 && second == 1 && third == 3 ? 0 : 1; }
`
	runFoundationNativeHarness(t, outputs, "collector_drop_proof_harness.c", harness)
}

func TestR7fRIndexedAuthorityReplacement(t *testing.T) {
	source := `module IndexedAuthority;
profile Core;
extern "C" void ObserveDrop(int value);
struct Resource { int value; }
void Drop(owned Resource resource) { ObserveDrop(resource.value); }
int Main()
{
    Option<owned Resource><array>[2] slots = [Option::None ...];
    int index = 1;
    Resource first = Resource{1};
    slots[index] = Option::Some(move first);
    Resource second = Resource{2};
    slots[index] = Option::Some(move second);
    slots[index] = Option::None;
    return 0;
}`
	module, err := Parse("IndexedAuthority.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "indexedauthority.generated.h"
static int count = 0;
static int first = 0;
static int second = 0;
void ObserveDrop(int value) { if (count == 0) first = value; else second = value; count++; }
int main(void) { return concept_indexed_authority_main() == 0 && count == 2 && first == 1 && second == 2 ? 0 : 1; }
`
	runFoundationNativeHarness(t, outputs, "indexed_authority_harness.c", harness)
}

func TestR7fRAppliedFailureArrayCOrder(t *testing.T) {
	source := `module AppliedArrayOrder;
profile Core;
struct Holder { Option<int><array>[3] values; }
int Main()
{
    Option<int><array>[3] values = [Option::None ...];
    values[1] = Option::Some(7);
    Holder holder = Holder{move values};
    return match (holder.values[1]) { Option::Some(value) => value, Option::None => 0, };
}`
	module, err := Parse("AppliedArrayOrder.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "applied_array_order_harness.c", "#include \"appliedarrayorder.generated.h\"\nint main(void) { return concept_applied_array_order_main() == 7 ? 0 : 1; }\n")
}

func TestR7fRRepeatedArrayInsideAggregateC11(t *testing.T) {
	source := `module NestedRepeatedArray;
profile Core;
struct Payload { int<array>[3] values; }
int Main() { Payload payload = Payload{[4 ...]}; return payload.values[2]; }`
	module, err := Parse("NestedRepeatedArray.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "nested_repeated_array_harness.c", "#include \"nestedrepeatedarray.generated.h\"\nint main(void) { return concept_nested_repeated_array_main() == 4 ? 0 : 1; }\n")
}

func TestR7fRGenericBorrowProvenanceAndConstraintClosure(t *testing.T) {
	source := `module GenericBorrowClosure;
profile Core;
concept Readable<T> { requires int Read(ref const T value); }
concept IndirectReadable<T> { requires Readable<T>; }
struct Item { int value; }
int Read(ref const Item value) { return value.value; }
template <typename T> requires Readable<T>
int ReadInner(ref const T value) { return Read(ref const value); }
template <typename T> requires IndirectReadable<T>
int ReadOuter(ref const T value) { return ReadInner<T>(ref const value); }
template <typename T>
ref const T Forward(ref const T value) { return ref const value; }
template <typename T>
ref const T ForwardAgain(ref const T value) { return Forward<T>(ref const value); }
int Main() { Item item = Item{9}; ref const Item borrow = ForwardAgain<Item>(ref const item); return ReadOuter<Item>(ref const borrow); }`
	module, err := Parse("GenericBorrowClosure.concept", source)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "generic_borrow_closure_harness.c", "#include \"genericborrowclosure.generated.h\"\nint main(void) { return concept_generic_borrow_closure_main() == 9 ? 0 : 1; }\n")
}
