# EVT1 R7d conformance: agentic synchronization

## Baseline

- Baseline and R7c commit: `1faeb298328c171e07b3f699dbcef17d18b0c4bc`
  (`Implement DragonGod scheduling`).
- Compiler: `concept-evt1-stage0-go`.
- Package graph: `DragonGod -> Standard`; Standard has no dependencies.
- The starting worktree was clean.

## Result: meaningful progression

R7d does not yet meet its full success floor. This change removes the first
concrete blocker identified by R7c: strict-C11 atomic machine semantics now
exist on the normal Concept -> MIR -> Planner -> C11 path and survive semantic
module transport. It deliberately does not label the existing EventBus,
MemoryState, or ActuatorHost worker-safe before their algorithms and authority
proofs are implemented.

The added `Standard.Synchronization.Atomic` module provides an immovable
`AtomicInt` and ordinary library operations for load, store, exchange, strong
compare-exchange, and fetch-add. Each operation has an explicit `MemoryOrder`
form and a sequentially consistent overload. The supported orders are Relaxed,
Acquire, Release, AcquireRelease, and Sequential. Valid C11 orders map exactly;
orders invalid for a particular operation are conservatively strengthened to
sequential consistency instead of invoking undefined behavior.

MIR represents the five general operations as `atomic_load`, `atomic_store`,
`atomic_exchange`, `atomic_compare_exchange`, and `atomic_fetch_add`. Planner
emits a `SynchronizationPlan` with `RetainC11Atomic`, `AtomicAccess`,
`ExplicitMemoryOrder`, and `NoAllocation` evidence. It has no exclusivity fact
yet, so it performs no weakening or guard elision.

## Evidence and remaining blocker

- The Standard native fact executes all five operations, including both CAS
  outcomes and sequential-default behavior.
- Compiler tests inspect `_Atomic int`, `<stdatomic.h>`, all C11 operations,
  MIR operation kinds, the conservative Planner decision, native strict-C11
  execution, and an artifact-only consumer parse/generation path.
- The implementation adds no volatile synchronization, runtime registry,
  heap, DragonGod type recognition, or handwritten safe/fast library path.

The next isolated blocker is the semantic shared-access model: Concept cannot
yet express and prove worker relationships such as exclusive writer, disjoint
slot, single producer/consumer, or synchronized multi-writer access. Without
those facts, refactoring DragonGod authorities or erasing synchronization would
be ungrounded. The next R7d slice must add that demand-driven authority model
and use it to choose guarded versus proven-disjoint paths from the same source.

Full multi-worker scheduling remains R7e work.

## Validation

- `go test ./...`: pass (`internal/concept` 99.453 s; Vulkan profile pass).
- `go vet ./...`: pass.
- Root and `legacy/poc3-zig` `zig build test`: pass.
- Standard package: 3 facts pass, including the new atomic fact.
- DragonGod package: 18 facts pass and 1 informational benchmark runs.
- Real Oct `make Test --file Make.oct`: pass after registering the local Oct
  registry and syncing the already-declared `Make@0.1.0` dependency.
- Atomic C, MIR, Planner JSON, and `concept-module.v1`: one identical combined
  hash across 100 runs.
- Active EVT1 corpus: 712 `.concept` fixtures; the full Go suite is the corpus
  conformance gate.
- `git diff --check`: pass.

The temporary `.oct` dependency cache used for bootstrap validation was removed
afterward; it is reproducible and is not part of this change.
