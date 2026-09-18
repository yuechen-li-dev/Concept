package concept

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const r7d5SynchronizationSource = `module Sync.Consumption;
profile Core;
import Standard.Synchronization.Atomic;

record struct WriterContext {}
record struct ReaderContext {}
record struct ClaimerContext {}
record struct CommitterContext {}
struct Payload { int value; }
struct Publication { AtomicInt state; }
struct Obligation { AtomicInt state; }

[[execution_context(WriterContext)]]
[[semantic_access("Publish", publication)]]
void PublishPayload(ref Payload payload, ref Publication publication)
{
    payload.value = 42;
    AtomicStoreIntrinsic(ref publication.state, 1, MemoryOrder::Release);
}

[[execution_context(ReaderContext)]]
[[semantic_access("Consume", publication)]]
int ConsumePayload(ref const Payload payload, ref const Publication publication)
{
    int ready = AtomicLoadIntrinsic(ref const publication.state, MemoryOrder::Acquire);
    if (ready == 1) { return payload.value; }
    return 0;
}

[[execution_context(ClaimerContext)]]
[[semantic_access("Claim", obligation)]]
bool ClaimOnce(ref Obligation obligation)
{
    int expected = 0;
    return AtomicCompareExchangeIntrinsic(
        ref obligation.state,
        ref expected,
        1,
        MemoryOrder::AcquireRelease,
        MemoryOrder::Acquire
    );
}

[[execution_context(CommitterContext)]]
[[semantic_access("Commit", obligation)]]
bool CommitOnce(ref Obligation obligation)
{
    int expected = 1;
    return AtomicCompareExchangeIntrinsic(
        ref obligation.state,
        ref expected,
        2,
        MemoryOrder::Release,
        MemoryOrder::Acquire
    );
}

`

func r7d5SynchronizationModule(t *testing.T) Module {
	t.Helper()
	atomic := buildSemanticArtifact(t, "Standard/Synchronization/Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"), nil)
	source := r7d5SynchronizationSource + `
void CheckSynchronizationProofs()
{
    Assert.Concept<sync.SynchronizedAccess>(Publication, "atomic accesses synchronize publication state");
    Assert.Concept<sync.PublishedBefore>(Payload, Payload, "release/acquire orders payload observation");
    Assert.Concept<sync.ExactlyOnce>(Obligation, "claim and commit CAS transitions are unique");
}
`
	module, err := ParseWithSemanticModules("sync_consumption.concept", source, map[string][]byte{"Standard.Synchronization.Atomic": atomic})
	if err != nil {
		t.Fatal(err)
	}
	return module
}

func TestR7d5DerivesSynchronizedAccessPublishedBeforeAndExactlyOnce(t *testing.T) {
	module := r7d5SynchronizationModule(t)
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		kind SemanticFactKind
		args []Type
		want string
	}{
		{FactSynchronizedAccess, []Type{{Name: "Publication", Kind: TypeStruct}}, "atomic"},
		{FactPublishedBefore, []Type{{Name: "Payload", Kind: TypeStruct}, {Name: "Payload", Kind: TypeStruct}}, "release publication"},
		{FactExactlyOnce, []Type{{Name: "Obligation", Kind: TypeStruct}}, "CAS"},
	}
	for _, tc := range cases {
		got := evt1DerivedSharedAccess(env, tc.kind, tc.args)
		if got.Outcome != FactProven || !strings.Contains(got.Evidence.Detail, tc.want) {
			t.Fatalf("%s was not derived from its mechanism: %#v", tc.kind, got)
		}
	}
}

func TestR7d5MissingOrderingAndPartialSynchronizationDoNotOverprove(t *testing.T) {
	source := strings.ReplaceAll(r7d5SynchronizationSource, "MemoryOrder::Release", "MemoryOrder::Relaxed")
	source = strings.ReplaceAll(source, "MemoryOrder::Acquire);\n    if (ready", "MemoryOrder::Relaxed);\n    if (ready")
	atomic := buildSemanticArtifact(t, "Standard/Synchronization/Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"), nil)
	module, err := ParseWithSemanticModules("sync_missing_order.concept", source, map[string][]byte{"Standard.Synchronization.Atomic": atomic})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	ordered := evt1DerivedSharedAccess(env, FactPublishedBefore, []Type{{Name: "Payload", Kind: TypeStruct}, {Name: "Payload", Kind: TypeStruct}})
	if ordered.Outcome != FactUnknown {
		t.Fatalf("relaxed source order overproved publication: %#v", ordered)
	}

	unsafeSource := `module Sync.Partial; profile Core;
struct A {} struct B {} struct Slot { int value; }
[[execution_context(A)]] void WriteA(ref Slot slot) { slot.value = 1; }
[[execution_context(B)]] int ReadB(ref const Slot slot) { return slot.value; }`
	unsafeModule, err := Parse("partial_sync.concept", unsafeSource)
	if err != nil {
		t.Fatal(err)
	}
	unsafeEnv, err := analyzeModule(unsafeModule)
	if err != nil {
		t.Fatal(err)
	}
	result := evt1DerivedSharedAccess(unsafeEnv, FactSynchronizedAccess, []Type{{Name: "Slot", Kind: TypeStruct}})
	if result.Outcome != FactDisproven {
		t.Fatalf("unguarded cross-context conflict was not disproven: %#v", result)
	}
}

func TestR7d5SynchronizationOrderingSurvivesArtifactOnlyConsumption(t *testing.T) {
	atomic := buildSemanticArtifact(t, "Standard/Synchronization/Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"), nil)
	artifact := buildSemanticArtifact(t, "sync_consumption.concept", r7d5SynchronizationSource, map[string][]byte{"Standard.Synchronization.Atomic": atomic})
	var decoded SemanticModuleArtifact
	if err := json.Unmarshal(artifact, &decoded); err != nil {
		t.Fatal(err)
	}
	ordered := false
	for _, entry := range decoded.AccessSummaries {
		ordered = ordered || (entry.Operation == AccessAtomicWrite && strings.Contains(entry.MemoryOrder, "Release") && entry.Mechanism != "")
	}
	if !ordered {
		t.Fatal("concept-module.v1 omitted release publication mechanism evidence")
	}
	consumer := `module Sync.Consumer;
profile Core;
import Sync.Consumption;
void CheckImportedOrdering()
{
    Assert.Concept<sync.PublishedBefore>(Payload, Payload, "artifact-only publication proof");
    Assert.Concept<sync.ExactlyOnce>(Obligation, "artifact-only claim proof");
}`
	module, err := ParseWithSemanticModules("sync_consumer.concept", consumer, map[string][]byte{"Sync.Consumption": artifact, "Standard.Synchronization.Atomic": atomic})
	if err != nil {
		t.Fatal(err)
	}
	env, err := analyzeModule(module)
	if err != nil {
		t.Fatal(err)
	}
	proof := evt1DerivedSharedAccess(env, FactPublishedBefore, []Type{{Name: "Payload", Kind: TypeStruct}, {Name: "Payload", Kind: TypeStruct}})
	if proof.Outcome != FactProven {
		t.Fatalf("artifact-only publication proof did not close: %#v", proof)
	}
}

func TestR7d5SameSourceVerificationRetainsAndOptimizedSimplifiesPrivateAtomic(t *testing.T) {
	atomic := buildSemanticArtifact(t, "Standard/Synchronization/Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"), nil)
	source := `module Sync.PrivateAtomic;
profile Core;
import Standard.Synchronization.Atomic;
record struct SoloContext {}
[[execution_context(SoloContext)]]
int Increment(ref AtomicInt value)
{
    int prior = AtomicLoadIntrinsic(ref const value, MemoryOrder::Relaxed);
    AtomicStoreIntrinsic(ref value, prior + 1, MemoryOrder::Relaxed);
    return prior;
}`
	module, err := ParseWithSemanticModules("private_atomic.concept", source, map[string][]byte{"Standard.Synchronization.Atomic": atomic})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), OptimizedCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	verifiedHeader := string(moduleOutput(t, verified, ".generated.h"))
	verifiedBody := string(moduleOutput(t, verified, ".generated.c"))
	optimizedHeader := string(moduleOutput(t, optimized, ".generated.h"))
	optimizedBody := string(moduleOutput(t, optimized, ".generated.c"))
	if !strings.Contains(verifiedHeader, "_Atomic int value") || !strings.Contains(verifiedBody, "atomic_load_explicit") {
		t.Fatal("verification lowering did not retain C11 atomic synchronization")
	}
	if strings.Contains(optimizedHeader, "_Atomic int value") || strings.Contains(optimizedBody, "atomic_load_explicit") || !strings.Contains(optimizedHeader, "int value") {
		t.Fatalf("optimized lowering did not simplify the proven private atomic:\n%s\n%s", optimizedHeader, optimizedBody)
	}
	plan, err := GeneratePlanWithPolicy(module, GenericC11Target(), OptimizedCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(plan, []byte(`"strategy": "SimplifyAtomic"`)) || !bytes.Contains(plan, []byte(`"NoPublicationOrdering"`)) {
		t.Fatalf("optimized plan omitted its proof conjunction:\n%s", plan)
	}
	for run := 0; run < 100; run++ {
		again, generateErr := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), OptimizedCompilationPolicy())
		if generateErr != nil || !equalOutputs(optimized, again) {
			t.Fatalf("optimized MIR/artifact/C output changed on run %d: %v", run+1, generateErr)
		}
		againPlan, planErr := GeneratePlanWithPolicy(module, GenericC11Target(), OptimizedCompilationPolicy())
		if planErr != nil || !bytes.Equal(plan, againPlan) {
			t.Fatalf("optimized synchronization plan changed on run %d: %v", run+1, planErr)
		}
	}
}

func equalOutputs(left Outputs, right Outputs) bool {
	if len(left) != len(right) {
		return false
	}
	for name, content := range left {
		if !bytes.Equal(content, right[name]) {
			return false
		}
	}
	return true
}

func TestR7d5GuardElisionRequiresExactSingleContextConjunction(t *testing.T) {
	atomic := buildSemanticArtifact(t, "Standard/Synchronization/Atomic.concept", standardMemorySource(t, "Standard/Synchronization/Atomic.concept"), nil)
	guard := buildSemanticArtifact(t, "Standard/Synchronization/Guard.concept", standardMemorySource(t, "Standard/Synchronization/Guard.concept"), map[string][]byte{"Standard.Synchronization.Atomic": atomic})
	artifacts := map[string][]byte{"Standard.Synchronization.Atomic": atomic, "Standard.Synchronization.Guard": guard}
	source := `module Sync.PrivateGuard;
profile Core;
import Standard.Synchronization.Guard;
record struct SoloContext {}
struct Shared { int value; }
[[execution_context(SoloContext)]]
void Mutate(ref SpinLock lock, ref Shared shared)
{
    Acquire(ref lock);
    shared.value = 7;
    Release(ref lock);
}`
	module, err := ParseWithSemanticModules("private_guard.concept", source, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	optimized, err := GenerateForTargetWithPolicy(module, []byte(source), GenericC11Target(), OptimizedCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	verifiedBody := string(moduleOutput(t, verified, ".generated.c"))
	optimizedBody := string(moduleOutput(t, optimized, ".generated.c"))
	if !strings.Contains(verifiedBody, "_private_guard_acquire(&((*lock)))") || !strings.Contains(verifiedBody, "_private_guard_release(&((*lock)))") {
		t.Fatalf("verification lowering did not retain guard calls:\n%s", verifiedBody)
	}
	mutateAt := strings.Index(optimizedBody, "_private_guard_mutate(")
	optimizedMutate := optimizedBody
	if mutateAt >= 0 {
		optimizedMutate = optimizedBody[mutateAt:]
	}
	if strings.Contains(optimizedMutate, "_private_guard_acquire(") || strings.Contains(optimizedMutate, "_private_guard_release(") {
		t.Fatalf("optimized lowering retained a proven-redundant guard call:\n%s", optimizedBody)
	}
	plan, err := GeneratePlanWithPolicy(module, GenericC11Target(), OptimizedCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(plan, []byte(`"strategy": "ElideSynchronization"`)) || !bytes.Contains(plan, []byte(`"NoConflictingCrossContextAccess"`)) {
		t.Fatalf("guard plan omitted the required proof conjunction:\n%s", plan)
	}

	shared := strings.Replace(source, "struct Shared { int value; }", "record struct ReaderContext {}\nstruct Shared { int value; }", 1) + `
[[execution_context(ReaderContext)]]
int Observe(ref const Shared shared) { return shared.value; }
`
	sharedModule, err := ParseWithSemanticModules("shared_guard.concept", shared, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := GenerateForTargetWithPolicy(sharedModule, []byte(shared), GenericC11Target(), OptimizedCompilationPolicy())
	if err != nil {
		t.Fatal(err)
	}
	retainedBody := string(moduleOutput(t, retained, ".generated.c"))
	mutateAt = strings.Index(retainedBody, "_private_guard_mutate(")
	if mutateAt < 0 || !strings.Contains(retainedBody[mutateAt:], "_private_guard_acquire(") {
		t.Fatal("cross-context reader did not retain guard synchronization")
	}
}

func TestR7d5ActuationClaimRaceIsExactlyOnceInNativeThreads(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	source := `module Sync.WorkerSafe;
profile Core;
import DragonGod.Actuation.Core;
record struct Configuration { int value; }
struct RaceFixture
{
    ActuatorHost<Configuration, 1> host;
    ActuationId id;
}

RaceFixture MakeRaceFixture()
{
    ActuationSlot<array>[1] slots = [EmptyActuationSlot()];
    ActuatorHost<Configuration, 1> host = ActuatorHost<Configuration, 1>{1, MakeSpinLock(), move slots};
    ActuationReceipt receipt = host.Dispatch(Allow(Reason{7}))!;
    RaceFixture fixture = RaceFixture{move host, receipt.id};
    return move fixture;
}

int ClaimAndCommit(ref RaceFixture fixture)
{
    int success = 0;
    try
    {
        owned ActuationClaim claim = fixture.host.Claim(fixture.id)?;
        fixture.host.Commit(move claim, Reason{9})?;
        success = 1;
    }
    except (ActuationError error)
    {
        success = 0;
    }
    return success;
}

int IsCommitted(ref const RaceFixture fixture)
{
    if (ActuationStatusCode(fixture.host.Status(fixture.id)) == 4) { return 1; }
    return 0;
}`
	moduleRoots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("Sync/WorkerSafe.concept", source, moduleRoots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	harness := `#include "workersafe.generated.h"
#include <windows.h>

typedef struct {
  concept_race_fixture* fixture;
  int result;
} claim_args;

static DWORD WINAPI claim_thread(LPVOID raw) {
  claim_args* args = (claim_args*)raw;
  args->result = concept_sync__worker_safe_claim_and_commit(args->fixture);
  return 0;
}

int main(void) {
  concept_race_fixture fixture = concept_sync__worker_safe_make_race_fixture();
  claim_args left = {&fixture, 0};
  claim_args right = {&fixture, 0};
  HANDLE threads[2];
  threads[0] = CreateThread(NULL, 0, claim_thread, &left, 0, NULL);
  threads[1] = CreateThread(NULL, 0, claim_thread, &right, 0, NULL);
  if (threads[0] == NULL || threads[1] == NULL) return 2;
  WaitForMultipleObjects(2, threads, TRUE, INFINITE);
  CloseHandle(threads[0]);
  CloseHandle(threads[1]);
  if (left.result + right.result != 1) return 3;
  return concept_sync__worker_safe_is_committed(&fixture) == 1 ? 0 : 4;
}
`
	runFoundationNativeHarness(t, outputs, "r7d5_actuation_race.c", harness)
}

func TestR7d5BlackboardPublicationDisjointWritersAndMPSCNativeThreads(t *testing.T) {
	root := filepath.Join("..", "..", "libraries")
	output := t.TempDir()
	if _, err := BuildPackage(root, output, "DragonGod"); err != nil {
		t.Fatal(err)
	}
	source := `module Sync.SharedState;
profile Core;
import DragonGod.Memory.State;
import DragonGod.Events.Core;
record struct Configuration { int value; }
struct SharedFixture
{
    MemoryState<Configuration, 2> memory;
    EventBus<Configuration, 2> events;
}

SharedFixture MakeSharedFixture()
{
    MemorySlot<array>[2] memorySlots = [EmptyMemorySlot(), EmptyMemorySlot()];
    MemoryState<Configuration, 2> memory = MemoryState<Configuration, 2>{AtomicInt{0}, MakeSpinLock(), move memorySlots};
    memory.Write(MemoryKey{1}, 0)!;
    memory.Write(MemoryKey{2}, 0)!;
    Event<array>[2] eventSlots = [EmptyEvent(), EmptyEvent()];
    EventBus<Configuration, 2> events = EventBus<Configuration, 2>{1, AtomicInt{0}, MakeSpinLock(), eventSlots};
    SharedFixture fixture = SharedFixture{move memory, move events};
    return move fixture;
}

void WriteAndPublish(ref SharedFixture fixture)
{
    fixture.memory.Write(MemoryKey{1}, 42)!;
    fixture.events.Dispatch(EventKind{1}, EventPayload{42})!;
}

void PublishSecond(ref SharedFixture fixture)
{
    fixture.events.Dispatch(EventKind{2}, EventPayload{7})!;
}

void WriteDisjoint(ref SharedFixture fixture)
{
    int value = 1;
    while (value <= 100) bounded(100)
    {
        fixture.memory.Write(MemoryKey{2}, value)!;
        value = value + 1;
    }
}

int ObservePublished(ref const SharedFixture fixture)
{
    if (fixture.events.Count() >= 2) { return fixture.memory.Read(MemoryKey{1}, -1); }
    return -1;
}

int ReadDisjoint(ref const SharedFixture fixture)
{
    return fixture.memory.Read(MemoryKey{2}, -1);
}

int EventCount(ref const SharedFixture fixture)
{
    return fixture.events.Count();
}`
	moduleRoots := []string{filepath.Join(output, "Standard", "modules"), filepath.Join(output, "DragonGod", "modules")}
	module, err := ParseWithSemanticModuleRoots("Sync/SharedState.concept", source, moduleRoots)
	if err != nil {
		t.Fatal(err)
	}
	outputs, err := Generate(module, []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	generatedBody := string(moduleOutput(t, outputs, ".generated.c"))
	writeAt := strings.Index(generatedBody, "_write(concept_memory_state")
	writeEnd := -1
	if writeAt >= 0 {
		writeEnd = strings.Index(generatedBody[writeAt:], "\n}\n")
	}
	if writeAt < 0 || writeEnd < 0 || !strings.Contains(generatedBody[writeAt:writeAt+writeEnd], "_drop(") {
		t.Fatalf("shared-state Write lowering omitted guard cleanup:\n%s", generatedBody)
	}
	harness := `#include "sharedstate.generated.h"
#include <windows.h>

typedef struct { concept_shared_fixture* fixture; int observed; } shared_args;

static DWORD WINAPI writer(LPVOID raw) {
  shared_args* args = (shared_args*)raw;
  concept_sync__shared_state_write_and_publish(args->fixture);
  return 0;
}

static DWORD WINAPI second_producer(LPVOID raw) {
  shared_args* args = (shared_args*)raw;
  concept_sync__shared_state_publish_second(args->fixture);
  return 0;
}

static DWORD WINAPI disjoint_writer(LPVOID raw) {
  shared_args* args = (shared_args*)raw;
  concept_sync__shared_state_write_disjoint(args->fixture);
  return 0;
}

static DWORD WINAPI reader(LPVOID raw) {
  shared_args* args = (shared_args*)raw;
  for (int i = 0; i < 1000000; ++i) {
    int value = concept_sync__shared_state_observe_published(args->fixture);
    if (value >= 0) { args->observed = value; return 0; }
  }
  args->observed = -2;
  return 0;
}

int main(void) {
  concept_shared_fixture fixture = concept_sync__shared_state_make_shared_fixture();
  shared_args args = {&fixture, -1};
  HANDLE threads[4];
  threads[0] = CreateThread(NULL, 0, writer, &args, 0, NULL);
  threads[1] = CreateThread(NULL, 0, second_producer, &args, 0, NULL);
  threads[2] = CreateThread(NULL, 0, disjoint_writer, &args, 0, NULL);
  threads[3] = CreateThread(NULL, 0, reader, &args, 0, NULL);
  for (int i = 0; i < 4; ++i) if (threads[i] == NULL) return 2;
  WaitForMultipleObjects(4, threads, TRUE, INFINITE);
  for (int i = 0; i < 4; ++i) CloseHandle(threads[i]);
  if (args.observed != 42) return 3;
  if (concept_sync__shared_state_read_disjoint(&fixture) != 100) return 4;
  return concept_sync__shared_state_event_count(&fixture) == 2 ? 0 : 5;
}
`
	runFoundationNativeHarness(t, outputs, "r7d5_shared_state_threads.c", harness)
}
