package concept

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestR7aManifestsAreOrdinaryImmutableConceptData(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	manifests, err := DiscoverPackageManifests(root)
	if err != nil {
		t.Fatal(err)
	}
	standard, ok := manifests["Standard"]
	if !ok || standard.Author != "CODEX" || standard.Kind != "Library" || standard.Version != (PackageVersion{0, 0, 1}) || len(standard.Dependencies) != 0 {
		t.Fatalf("unexpected Standard manifest: %#v", standard)
	}
	dragonGod, ok := manifests["DragonGod"]
	if !ok || dragonGod.Author != "CODEX" || dragonGod.Kind != "Library" || dragonGod.Version != (PackageVersion{0, 0, 1}) || len(dragonGod.Dependencies) != 1 || dragonGod.Dependencies[0] != "Standard" {
		t.Fatalf("unexpected DragonGod manifest: %#v", dragonGod)
	}
	order, err := PackageBuildOrder(manifests, "DragonGod")
	if err != nil || len(order) != 2 || order[0] != "Standard" || order[1] != "DragonGod" {
		t.Fatalf("unexpected package order %v: %v", order, err)
	}
}

func TestR7aPackageCycleDiagnosticIsBounded(t *testing.T) {
	manifests := map[string]PackageManifestValue{
		"A": {Name: "A", Dependencies: []string{"B"}},
		"B": {Name: "B", Dependencies: []string{"A"}},
	}
	_, err := PackageBuildOrder(manifests, "A")
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("PACKAGE_DEPENDENCY_CYCLE")) {
		t.Fatalf("expected package cycle diagnostic, got %v", err)
	}
}

func TestR7aStandardAndDragonGodPackagesBuildDeterministically(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	first, err := BuildPackage(root, output, "DragonGod")
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := MarshalPackageGraph(first)
	for run := 1; run < 100; run++ {
		again, err := BuildPackage(root, output, "DragonGod")
		if err != nil {
			t.Fatalf("run %d: %v", run, err)
		}
		body, _ := MarshalPackageGraph(again)
		if !bytes.Equal(body, expected) {
			t.Fatalf("package graph changed on run %d", run)
		}
	}
	if len(first.Packages) != 2 || first.Packages[0].Name != "Standard" || first.Packages[1].Name != "DragonGod" {
		t.Fatalf("unexpected graph: %#v", first.Packages)
	}
	if first.Packages[0].Author != "CODEX" || first.Packages[1].Author != "CODEX" ||
		first.Packages[1].Version != (PackageVersion{0, 0, 1}) || first.Packages[1].Kind != "Library" ||
		len(first.Packages[1].Dependencies) != 1 || first.Packages[1].Dependencies[0] != "Standard" {
		t.Fatalf("package graph did not preserve manifest metadata: %#v", first.Packages)
	}
}

func TestR7aPromotedStandardMemoryArtifactIsEquivalent(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "Standard"); err != nil {
		t.Fatal(err)
	}
	expected := standardMemoryArtifacts(t)["Standard.Memory.Bump"]
	actual, err := os.ReadFile(filepath.Join(output, "Standard", "modules", "Standard", "Memory", "Bump.concept-module.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)) {
		t.Fatal("promoted Standard.Memory artifact is not semantically equivalent to the R6p build path")
	}
}

func TestR7aArtifactOnlyDragonGodConsumerRunsStrictC11(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	source := `module R7a.ArtifactConsumer;
profile Core;
import DragonGod.Memory.Boot;
import Standard.Memory.Bump;
import Standard.Memory.MonotonicOwnership;
int Main()
{
    FirmwareMemorySource firmware = MakeFirmwareMemorySource();
    BumpAllocator bump = MakeBumpAllocator(FirmwareRegion(ref firmware));
    return InspectBootInfo(ref bump, BootInfo{4, 38})!;
}

`
	moduleRoots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("R7a/ArtifactConsumer.concept", source, moduleRoots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	runFoundationNativeHarness(t, outputs, "r7a_artifact_consumer_harness.c", "#include \"artifactconsumer.generated.h\"\nint main(void) { return concept_artifactconsumer_main() == 42 ? 0 : 1; }\n")
}

func TestR7bArtifactOnlyAgenticConsumerIsBoundedStrictC11(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	source := `module R7b.AgenticArtifactConsumer;
profile Core;
import DragonGod.Memory.State;
import DragonGod.Events.Core;
record struct ConsumerConfiguration { int identity; };
int Main()
{
    MemorySlot emptyMemory = EmptyMemorySlot();
    MemorySlot<array>[2] memorySlots = [emptyMemory, emptyMemory];
    MemoryState<ConsumerConfiguration, 2> memory = MemoryState<ConsumerConfiguration, 2>{0, memorySlots};
    Event emptyEvent = EmptyEvent();
    Event<array>[2] eventSlots = [emptyEvent, emptyEvent];
    EventBus<ConsumerConfiguration, 2> events = EventBus<ConsumerConfiguration, 2>{1, 0, eventSlots};
    EventId id = events.Dispatch(EventKind{4}, EventPayload{7})!;
    memory.Write(MemoryKey{1}, 7)!;
    return memory.Read(MemoryKey{1}, 0) + events.Count() + id.value - 1;
}
`
	moduleRoots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("R7b/AgenticArtifactConsumer.concept", source, moduleRoots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := string(moduleOutput(t, outputs, ".generated.c"))
	for _, forbidden := range []string{"malloc", "legacy/dragon-god-poc", "runtime_reflection", "saved_pc"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("agentic artifact consumer emitted forbidden runtime artifact %q", forbidden)
		}
	}
	runFoundationNativeHarness(t, outputs, "r7b_agentic_artifact_consumer_harness.c", "#include \"agenticartifactconsumer.generated.h\"\nint main(void) { return concept_agenticartifactconsumer_main() == 8 ? 0 : 1; }\n")
}

func TestR7cArtifactOnlySchedulerConsumerIsBoundedStrictC11(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	source := `module R7c.SchedulerArtifactConsumer;
profile Core;
import DragonGod.Scheduling.Core;
import DragonGod.Scheduling.Queue;
import DragonGod.Scheduling.Fifo;
import DragonGod.Trace.Core;
record struct ConsumerConfiguration { int identity; };
struct Work { int remaining; int steps; };
ExecutionStep StepScheduled(ref Work self)
{
    self.remaining = self.remaining - 1;
    self.steps = self.steps + 1;
    if (self.remaining == 0) { return ExecutionStep::Completed; }
    return ExecutionStep::Ready;
}
requires Schedulable<Work>;
int Main()
{
    ContextId emptyId = EmptyContextId();
    ContextId<array>[1] ids = [emptyId];
    Work emptyWork = Work{0, 0};
    Work<array>[1] machines = [emptyWork];
    ExecutionState<array>[1] states = [ExecutionState::Ready];
    EventKind<array>[1] kinds = [EventKind{0}];
    Deadline<array>[1] deadlines = [Deadline{MonotonicTime{0}}];
    Reason<array>[1] failures = [Reason{0}];
    bool<array>[1] active = [false];
    bool<array>[1] queued = [false];
    ContextTable<Work, 1> contexts = ContextTable<Work, 1>{move ids, move machines, move states, move kinds, move deadlines, move failures, move active, move queued};
    ContextId<array>[1] readyEntries = [emptyId];
    FifoReadyQueue<1> ready = FifoReadyQueue<1>{move readyEntries, 0, 0};
    TraceEntry emptyTrace = EmptyTraceEntry();
    TraceEntry<array>[16] traceEntries = [emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace, emptyTrace];
    TraceRecorder<ConsumerConfiguration, 16> trace = TraceRecorder<ConsumerConfiguration, 16>{0, move traceEntries};
    FifoScheduler<ConsumerConfiguration, Work, 1, 1, 16> scheduler = FifoScheduler<ConsumerConfiguration, Work, 1, 1, 16>{SchedulerConfiguration{1}, move contexts, move ready, Clock{MonotonicTime{0}}, move trace, 1, 0, 0};
    ContextId id = scheduler.Register(Work{3, 0})!;
    scheduler.RunUntilIdle(3)!;
    return scheduler.contexts.machines[0].steps + ExecutionStateCode(scheduler.State(id)!);
}
`
	moduleRoots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("R7c/SchedulerArtifactConsumer.concept", source, moduleRoots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ToLower(string(moduleOutput(t, outputs, ".generated.c")))
	t.Logf("artifact-only scheduler generated C: %d bytes", len(body))
	for _, forbidden := range []string{"malloc", "calloc", "realloc", "free", "schedulerplan", "mirscheduler", "mirthread", "current_scheduler"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("scheduler artifact consumer emitted forbidden runtime artifact %q", forbidden)
		}
	}
	runFoundationNativeHarness(t, outputs, "r7c_scheduler_artifact_consumer_harness.c", "#include \"schedulerartifactconsumer.generated.h\"\nint main(void) { return concept_schedulerartifactconsumer_main() == 8 ? 0 : 1; }\n")
}

func TestR7aPackageErrorsRemainTyped(t *testing.T) {
	_, err := PackageBuildOrder(map[string]PackageManifestValue{}, "Missing")
	if err == nil {
		t.Fatal("expected missing package error")
	}
	var diagnostic Diagnostic
	if errors.As(err, &diagnostic) {
		t.Fatalf("package graph errors must not masquerade as source diagnostics: %v", err)
	}
}
