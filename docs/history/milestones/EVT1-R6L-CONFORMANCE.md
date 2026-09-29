# EVT1 R6l conformance: initialized typed-storage ownership progression

Status: meaningful progression

Baseline HEAD: `f5eff4b3258ebef374a34468ddf064c9b50fe973`

Compiler identity: `concept-evt1-stage0-go`

R6k stop commit: `f5eff4b Document EVT1 allocator owner boundary`

## Implemented and verified

- open generic `bind<T>` and trusted `MemoryRegion<SystemMemory>` binding;
- field-sensitive Initialized/Uninitialized/Moved storage-object state;
- `Initialize(owner.storage)`, `Destroy(owner.storage)`, and reinitialization;
- descriptor-authority transfer through storage fields and whole owners;
- structurally movable-only owners through non-copyable `Storage<T>`;
- mutable and const `Value(storage)` references bounded by owner provenance;
- use-after-Destroy invalidation and conservative move rejection with live refs;
- explicit generic owner Drop instantiation;
- unchanged typed-storage runtime layout and ordinary C field lowering;
- Initialized fact serialization in bounded value summaries.

The focused suite includes successful generic bind, owner move/value/destroy,
reinitialize, generic Drop materialization, invalid state transitions, and the
four R6k boundary regressions. Generated-C assertions reject `malloc` and
`memcpy` in the representative owner path.

## Pinned next blocker

The `Result<Owner<T>, E>` factory/extraction specimen fails deterministically
with `CV4106: ... expected Owner<int> but got Owner<T>`. Nested generic owner
identity is lost across the current failure-carrier template path. This blocks
honest Result/Option/wrapper and imported factory state transport, and therefore
blocks completion of R6l and the allocator retry.

A 100-run byte-identical `Standard.TypedStorage` artifact is also proven, but
its consumer instantiation currently rejects the imported `Storage<T>` field
with `CV4148: unknown concept parameter T`. Both diagnostics are pinned rather
than bypassed with consumer source inclusion.

No allocator-specific compiler branch, allocator MIR node, runtime provenance
or object-state registry, hidden heap, `new`/`delete`, source `T*`, or allocator
library was introduced.
