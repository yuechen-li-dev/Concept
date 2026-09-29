# EVT1 R7fR collector convergence

Result: **Success at the collector-library level**. Baseline and R7f3 were
`2fc0014e7dba90fe85edea5be4d182c9c7deae82`; R7f2 was
`3e830976f5d0d43a9e63bb332c1a19d2860a9b60`; R7f1 was
`58dae662d4559b2c0ea6843767e36a54dfbd706d`; R7e was
`b1747d7d3ffba7510aff6484c6863f70c214cf6e`. The compiler ID is
`concept-evt1-stage0-go`. The initial worktree was clean. The R7f3 report
remains the historical honest stop; this report records the persistent
collector and the repairs made around it.

## Persistent source and progression

`libraries/Standard/Collection/Core.concept` is the shipped module. It began
as fixed optional authority slots and progressed through construction,
allocation, borrow, roots, Trace, mark, sweep, native graph execution, and an
artifact-only consumer. It was never deleted in response to a compiler
failure. The slot stores `Option<owned Allocation<T,A>>`, rather than a bare
`Storage<T>`: the ordinary `Allocation` owner carries the releasable region
and implements `Destroy` before `Release` on Drop. A bare `Storage<T>` has no
such Drop and remains invalid for indexed replacement.

| Blocker and reproducer | Classification and root cause | Repair and focused regression | Progress afterward |
| --- | --- | --- | --- |
| `CV4133` constructing `[Option::None ...]` | Compiler bug: repeated literal was not accepted as direct independent construction | Direct-initialize repeated array values; indexed authority test | Collector metadata constructs |
| `CV4133` at `slots[index] = move candidate` | Missing general indexed replacement rule: projected noncopyable place lacked Drop-before-overwrite | Permit transferable replacement only for a Drop-bearing type; lower old Drop before incoming move; `TestR7fRIndexedAuthorityReplacement` proves two exact Drops at a dynamic index | Allocation slot owns one authority |
| Strict C11 omitted applied `Option` declaration before its array and emitted generic Drop name for the wrong instance | Compiler bugs in dependency ordering and generic symbol selection | Dependency-sort named, failure, view, and storage declarations; select closed generic Drop; `TestR7fRAppliedFailureArrayCOrder` | Allocation executes in C11 |
| Borrowing an optional allocation payload had no ordinary checked projection | Missing general library primitive | `OptionValue(ref Option<T>)` and readonly overload check `Some`, then return a payload reference | Borrow reaches initialized T |
| `CV4522` from an open generic `ConstAllocationValue` return | Compiler bug: lifetime derivation tried to close type parameters before instantiation | Derive parameter-relative provenance from the open template body; `TestR7fRGenericBorrowProvenanceAndConstraintClosure` | Borrow compiles |
| `CV4176` calling a nested template through `Collectible<T,A>` | Compiler bug: nested constraint check compared only direct concept names | Check the caller's prerequisite closure; same focused generic regression | Trace call compiles |
| `Collect` falsely conflicted with its own collector value borrowing a pool | Compiler bug: a value containing a reference was treated as an alias of its referent | Follow borrowed resource paths only for an actual reference alias; artifact consumer proves scoped legal and live-borrow invalid sequences | Sweep compiles and executes |
| `Result<ref T,E>` and `Result<ref const T,E>` shared a declaration key | Compiler bug: generic identity erased reference qualifiers | Use the emitted failure type identity for declaration keys; graph fixture instantiates both borrow forms through C11 | Mutable graph construction works |
| `Borrow(...)!` lost its collector borrow path | Compiler bug: failure unwrap erased object-borrow evidence | Propagate paths through `FailureExpr` and bind; artifact-only static-invalid consumer rejects overlapping `Collect` | Borrow/Collect exclusion holds |
| Concrete `NoAllocation(CollectOnce)` was unknown at `Len` and `Value` | Compiler proof gap for compiler-defined storage inspection | Mark only unshadowed `Len`, `Value`, and `OptionValue` intrinsics as nonallocating; artifact consumer proves collection wrapper | Concrete no-allocation proof passes |
| Nested repeated array in an aggregate rendered `/* comptime_array */` | C lowering bug | Emit typed C11 compound literals for evaluated arrays; `TestR7fRRepeatedArrayInsideAggregateC11` | Allocator failure path executes |
| 100-run instantiated MIR differed | Nondeterministic generic-instance map traversal chose different duplicate storage-type source spans | Sort template-instance keys before storage-type collection; explicit 100-run gate | Artifact, graph, MIR, C all byte-identical |

## Collector semantics and executable evidence

The homogeneous collector uses allocator-backed, non-moving objects. Each
handle is `(index, generation, domain)`. `ValidHandle` checks all three,
occupancy, and bounds before access. The caller supplies distinct domain IDs
for independent collectors. Root entries are explicit and bounded by
`Capacity`; duplicate registrations have separate tokens. `Trace<T>` requires
the synchronous `TraceReferences(ref const T, ref TraceVisitor<T>)` operation.
`Visit` rejects invalid or foreign edges. A generic fixed local array holds
the index worklist; marking on enqueue bounds it to one entry per slot.
Marking starts from registered roots. Sweep visits slots in index order,
replaces unreachable optional owners with `None`, runs the ordinary
allocation Drop, clears occupancy, and increments generation.

The Standard facts pass strict C11 for rooted and unrooted objects, a
three-object cycle, a diamond with a shared tail, a self-cycle, an unrooted
cyclic island beside a rooted graph, multiple collections, stale handle
rejection after slot reuse, allocation failure leaving metadata free, and two
independent collectors. `TestR7fRCollectorExactDropAndRelease` observes Drop
IDs `2,1` in deterministic sweep order, then ID `3` when a collector with a
live root leaves scope. Every ID appears exactly once, and the test verifies
that swept Pool slots are released. The Standard.Memory
`Allocation<T,A>` Drop itself performs `Destroy<T>` before `Release`.

`Collect` declares `compiler.InvalidatesBorrows(Collect, collector)`.
`TestR7fRCollectorArtifactOnlyConsumer` imports semantic artifacts alone,
instantiates the collector, executes strict C11, proves
`NoAllocation(CollectOnce)` for a concrete Pool/no-edge Trace policy, and
checks that a live unwrapped borrow causes
`DESTRUCTIVE_ACCESS_WITH_LIVE_BORROW`. The legal scoped borrow then Collect
executes. R7f2 suspension rejection remains unchanged. The same consumer
holds an ordinary value, a separately owned Standard.Memory allocation, and
a collector allocation at once.

`TestR7fRCollectorHundredRunDeterminism` is run explicitly with
`CONCEPT_R7FR_DETERMINISM=1`: 100 consecutive Standard package builds and
artifact-only consumer generations yielded byte-identical package graph,
collector semantic artifact, instantiated MIR, and generated C. Normal
`go test ./...` skips that long gate.

The EVT1 manifest now contains 395 valid, 256 static-invalid, and 13
runtime-negative fixtures. New ordinary-language cases cover indexed owned
Option replacement, repeated arrays in aggregates, and an unwrapped borrow
that blocks a destructive call.

## Scope and boundaries

Collection has no worklist allocation; `AllocateObject` is explicitly
allocating. Concrete `NoAllocation` proof depends on the user's Trace and
allocator Release witnesses. Domain uniqueness and manual root-token use are
caller contracts. Exhausted generations retire their slots instead of
wrapping. A RootGuard, automatic root discovery, heterogeneous
witnesses, finalizers, weak references, moving/generational/concurrent GC,
and reflection-derived tracing are deferred. No DragonGod scheduler boundary
or multi-worker rendezvous was added. The existing R7e scheduler tests remain
green; integration can follow at quantum boundaries after this collector core.

No collector name appears in production compiler semantics, MIR opcodes,
Planner policies, or runtime hooks. The added compiler work applies to
ordinary indexed ownership, applied type ordering, generic provenance,
prerequisite closure, qualified failure types, nested C arrays, resource
paths, and deterministic storage facts.

## Validation and performance sanity

Go 1.27.0 and Zig 0.16.0 were used. `go test ./...`, `go vet ./...`, both Zig `build test`
lanes, focused collector/native tests, the full EVT1 corpus, and the explicit
100-run gate passed. Oct was unavailable on PATH; the existing adjacent-Oct
validation executable at `%TEMP%/concept-r7f3-oct.exe` ran `make BurnIn`,
`make TestStandard` (8 passed), and `make TestDragonGod` (21 passed). The Oct
targets invoke the Standard and DragonGod builds before their tests.

No throughput threshold was set. The implementation uses one bounded local
index worklist, marks each slot at most once, and scans the fixed slot table
once per collection. The native fixtures execute repeated collection,
unreachable cycles, and roots without a runtime allocation path; wall-clock
throughput was not used to qualify this correctness milestone.
