# EVT1 R7d2 conformance: shared-access authority

## Baseline and result

- Baseline and R7d1 atomic commit: `cea7969` (`Add Concept atomic synchronization substrate`).
- R7c scheduler commit: `1faeb29` (`Implement DragonGod scheduling`).
- Compiler: `concept-evt1-stage0-go`.
- Starting worktree: clean.

Result: **meaningful progression**. Phase 1's explicit-source blocker is
removed: compiler-owned synchronization propositions can be declared,
required by named concepts, asserted, explained, emitted in MIR, and
transported through `concept-module.v1`. R7d2's full success floor is not met;
automatic access-set derivation and worker-safe DragonGod algorithms remain
the next bounded work.

## Vocabulary and exact syntax

The justified initial vocabulary is:

- `sync.ExclusiveWriter<Writer, Subject>`
- `sync.SingleProducer<Producer, Queue>`
- `sync.SingleConsumer<Consumer, Queue>`
- `sync.SynchronizedAccess<Subject>`
- `sync.PublishedBefore<Write, Observation>`
- `sync.ExactlyOnce<Obligation>`

Existing `Disjoint<A, B>` remains the one storage-disjointness proposition.
There is no duplicate `sync.Disjoint`, synonymous `SingleWriter`, or
unnecessary `SingleReader`.

An explicit concrete authority contract uses ordinary module `requires`:

```concept
requires sync.ExclusiveWriter<AgentA, TemperatureSlot>;
```

A reusable obligation uses ordinary concept composition:

```concept
concept TemperatureAccess<TAgent, TSlot>
{
    requires sync.ExclusiveWriter<TAgent, TSlot>;
    requires sync.SynchronizedAccess<TSlot>;
}
```

The current assertion grammar names the proposition outside its subject list:

```concept
Assert.Concept<sync.ExclusiveWriter>(
    AgentA,
    TemperatureSlot,
    "AgentA must be the only writer"
);
```

This deliberate adaptation uses the existing relational assertion machinery.

## Proof and trust semantics

Subject arguments are resolved `Type` values and stored as ordered typed
semantic subjects. No runtime identity or name-based library heuristic is
introduced. Ordered matching prevents writer/subject and producer/queue
reversal.

A concrete top-level declaration is explicit module authority with `Declared`
origin. It may close a local proof unless the compiler already has conflicting
authority. `ExclusiveWriter`, `SingleProducer`, and `SingleConsumer`
declarations for the same protected subject reject distinct authorities with
`SYNC_AUTHORITY_CONTRADICTION`.

A `requires sync.X` inside a generic or named concept is only an obligation. It
does not add a global fact. Concrete substitution must find a matching
declared, derived, or imported fact. A missing fact is `Unknown`; a known
different exclusive authority is `Disproven`.

Imported facts use `ModuleFactSummary`, preserving the distinction from local
`Declared` authority. Future access analysis will use existing derived origin
classes rather than relabeling declarations as proof.

## Artifact and runtime boundary

`concept-module.v1` adds an inspectable, hash-covered `shared_access_facts`
array containing bounded typed `MIRSemanticFact` records. Consumers load that
summary without source reparse; imported assertions themselves are still not
replayed. Compile-time contracts erase before runtime and add no heap, proof
registry, writer registry, or synchronization state.

This slice does not authorize guard elision, atomic simplification, or a
worker-safety claim. No derived writer/producer/consumer set exists yet, so
Planner continues retaining all R7d1 C11 atomics. Existing DragonGod
MemoryState, EventBus, and ActuatorHost remain single-worker components.

## Evidence

- Direct, composed, Proven, Disproven, and Unknown proof tests.
- All six propositions parse and assert through ordinary Concept source.
- Contradictory exclusive declarations reject statically.
- Artifact-only import proves a named producer/exactly-once bundle with
  `ModuleFactSummary` evidence.
- Proof graphs expose independent nested requirements and their origins.

The next blocker is a bounded shared-access operation summary (`Read`,
`Write`, `Publish`, `Consume`, `Claim`, `Commit`) from which the compiler can
derive writer, producer, and consumer sets. Until that exists, integrating
guards into DragonGod would merely trust declarations rather than prove the
real implementation.

## Validation

- `go test ./...`: pass (`internal/concept` 101.209 s; Vulkan profile pass).
- `go vet ./...`: pass.
- Root and `legacy/poc3-zig` `zig build test`: pass.
- Real Oct `make Test --file Make.oct`: Standard 3 facts pass; DragonGod 18
  facts pass and its scheduler benchmark runs.
- Shared-access focused tests: pass, including 100-run proof JSON and artifact
  byte determinism.
- Active EVT1 corpus: 712 `.concept` fixtures.
- `git diff --check`: pass.

The temporary `.oct` registry/dependency cache used to resolve the already
declared `Make@0.1.0` bootstrap dependency was removed after validation.
