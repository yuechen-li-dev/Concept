# EVT1 R6p conformance — Standard.Memory

Status: Success

Baseline HEAD: `0920b961cc5f532c65505dc6bedb6caa03a9f7e0`

Compiler: `concept-evt1-stage0-go`; Go 1.27.0; Zig 0.16.0.

R6o commit: `0920b96 Implement concepts in Concept`.

The starting worktree was clean. The starting semantic corpus was 382 valid,
246 static-invalid, 13 runtime-negative, 5 compatibility, and 4 retained
expected-divergence fixtures. The active library roots are
`language/evt1/tooling/modules/Standard` and `Platform`.

## Prerequisite and result

The R6n neutral `Provider<TProvider>`, second-parameter owner, and constrained
generic Drop feasibility cases pass through R6o. R6p publishes ordinary
`Standard.Memory.Core`, `.Bump`, `.Pool`, `.Ownership`,
`.MonotonicOwnership`, and `.Host` modules.

`AllocationError` has `OutOfMemory`, `InvalidAlignment`, `InvalidRegion`, and
`DoubleRelease`. `Allocator<A>` requires Result allocation and its
`compiler.Allocates` effect. `ReleasableAllocator<A>` composes it and adds
Result release. Bump deliberately supplies no Release. Pool uses 16 bounded
inline slots, validates exact regions, and reuses a released slot.

The fixed source owns 256 inline bytes with four-byte alignment. Hosted source
ownership is the explicit R6j `HostAllocation`; the strict harness observes one
host free. `CheckedAlignUp` rejects zero alignment and detects unsigned wrap.
Policies reject alignment their backing cannot guarantee.

The typed path transports `Result<Allocation<T,A>, AllocationError>` through
`?`, binds and initializes ordinary `Storage<T>`, and exposes mutable and const
owner-relative borrows. Generic Drop destroys before releasing. The monotonic
variant destroys only. Copy rejects, owner movement emits no `memcpy`, and an
immovable payload remains stationary.

## Evidence

- `.concept_test` facts cover bump capacity, padding, multiple allocation,
  remaining capacity, exhaustion and invalid alignment; pool capacity, 16-slot
  exhaustion, distinct slots, release, reuse, double release, foreign origin,
  and malformed interior release.
- Focused tests build every module as `concept-module.v1`, compile an
  artifact-only consumer, compare a fresh source-DAG consumer, and compare all
  generated outputs plus artifacts over 100 runs.
- Strict C11 harnesses execute fixed bump, pool reuse, typed Result/Drop order,
  immovable owner movement, hosted exactly-once free, DragonGod firmware-style
  backing, and Prometheus aligned tensor-backing specimens.
- `NoAllocation` is Disproven for typed allocation, Proven for imported
  `Remaining`, and Unknown for an opaque foreign operation.
- Static-invalid cases reject owner copy, owner-borrow escape, allocator-backing
  escape, and Bump as a releasable allocator. Existing storage regressions keep
  use-after-Destroy, double-Destroy, invalid-bind, and lifetime checks pinned.

An uncached focused Go invocation that builds and executes the representative
pool typed-owner consumer completed in 1.095 seconds on the qualification host;
this includes Go test process and package overhead and is informational only.

The DragonGod generated C is scanned for malloc/calloc/realloc/free, memcpy,
GC, and RTTI. Prometheus allocates one aligned aggregate backing; direct Span
projection from that allocated aggregate exposed an existing generated-header
ordering limitation and is recorded as a bounded integration follow-on rather
than allocator policy. No Vulkan or new tensor semantics were added.

## General incidental fixes

Ordinary allocator code exposed narrow general defects. R6p now compares
imported/materialized structural types semantically; keys transported template
effects by signature; follows calls under failure, move, and ref expressions;
substitutes bounded while bodies using their actual return context; preserves
unknown loop summaries; recursively summarizes returns and address provenance;
uses transported provenance for region binding; instantiates multi-parameter
generic Drop; and never cleanup-owns borrowed parameters. None recognizes an
allocator, owner, policy, or `Standard.Memory` spelling.

## Audits and deferrals

MIR and Planner output contain ordinary calls, regions, storage, and effects.
Production compiler/runtime code has no allocator policy dispatch, runtime
allocator/ownership/region/proof registry, hidden heap, runtime witness, GC,
`new`/`delete`, source-level `T*`, or allocator artifact schema. Hosted code
reaches only the explicit declared host seam.

Arena is deferred as orthogonal bounded parent-chunk policy. Callable capture,
async, and machine owners are also bounded integration follow-ons.

```text
R6c  generic/library/effect substrate missing
R6f  quantity/address/storage model missing
R6i  trusted external storage authority missing
R6k  initialized typed-storage owner missing
R6l  initialized generic storage ownership added
R6m  structural generic substitution completed
R6n  blocked by first-parameter-only generic concepts
R6o  proper arbitrary-position and relational generic concepts implemented
R6p  Standard.Memory implemented as ordinary Concept library code
```
