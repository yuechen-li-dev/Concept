package concept

import (
	"encoding/json"
	"strings"
	"testing"
)

func atomicModule(t *testing.T) Module {
	t.Helper()
	module, err := Parse("Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"))
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestAtomicOperationsReachMIRPlannerAndStrictC11(t *testing.T) {
	module := atomicModule(t)
	source := []byte(standardMemorySource(t, "Standard/Synchronization/Atomic.concept"))
	outputs, err := Generate(module, source)
	if err != nil {
		t.Fatal(err)
	}
	header := string(outputs["atomic.generated.h"])
	body := string(outputs["atomic.generated.c"])
	for _, want := range []string{"#include <stdatomic.h>", "_Atomic int value", "memory_order_acquire", "memory_order_release", "memory_order_seq_cst"} {
		if !strings.Contains(header, want) {
			t.Fatalf("generated atomic header omitted %q:\n%s", want, header)
		}
	}
	for _, want := range []string{"atomic_load_explicit", "atomic_store_explicit", "atomic_exchange_explicit", "atomic_compare_exchange_strong_explicit", "atomic_fetch_add_explicit"} {
		if !strings.Contains(body, want) {
			t.Fatalf("generated atomic body omitted %q:\n%s", want, body)
		}
	}
	var mir MIR
	if err := json.Unmarshal(outputs["atomic.mir.json"], &mir); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, fn := range mir.Functions {
		for _, op := range fn.Operations {
			kinds[op.Kind] = true
		}
	}
	for _, want := range []string{"atomic_load", "atomic_store", "atomic_exchange", "atomic_compare_exchange", "atomic_fetch_add"} {
		if !kinds[want] {
			t.Fatalf("MIR omitted %s", want)
		}
	}
	planBytes, err := GeneratePlan(module, GenericC11Target())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(planBytes), `"category": "SynchronizationPlan"`) || !strings.Contains(string(planBytes), `"strategy": "RetainC11Atomic"`) {
		t.Fatalf("Planner omitted conservative atomic decision:\n%s", planBytes)
	}
	runFoundationNativeHarness(t, outputs, "atomic_harness.c", "#include \"atomic.generated.h\"\nint main(void) { concept_atomic_int value = {.value=1}; concept_atomic_store_atomic__atomic_int_int_memory_order(&value, 4, (concept_memory_order){.tag=2}); return concept_atomic_load_atomic__atomic_int_memory_order(&value, (concept_memory_order){.tag=1}) == 4 ? 0 : 1; }\n")
}

func TestAtomicSemanticArtifactIsConsumableWithoutSource(t *testing.T) {
	artifact := buildSemanticArtifact(t, "Standard/Synchronization/Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"), nil)
	consumer := `module AtomicConsumer;
profile Core;
import Standard.Synchronization.Atomic;
int Use()
{
    AtomicInt value = AtomicInt{2};
    return FetchAddAtomic(ref value, 3, MemoryOrder::Sequential);
}`
	module, err := ParseWithSemanticModules("AtomicConsumer.concept", consumer, map[string][]byte{"Standard.Synchronization.Atomic": artifact})
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(consumer))
	if err != nil {
		t.Fatal(err)
	}
	generated := string(outputs["atomicconsumer.generated.c"])
	if !strings.Contains(generated, "fetch_add_atomic") {
		t.Fatalf("artifact-only consumer did not preserve ordinary atomic library call:\n%s", generated)
	}
}
