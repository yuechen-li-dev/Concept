# EVT1 semantic facts and optimization direction

Status: R4j bounded implementation; optimization remains deferred

## Principle

Known semantic facts are compiler assets. Storage, layout, binding, stream,
Span, and tensor validation already establish intent that later consumers must
not reconstruct from lowered pointers.

> Optimization should be downstream of semantic proof.
>
> The optimizer should consume facts already established by the language and
> program, not spend its budget recreating known intent.

R4j therefore adds proof infrastructure, not an optimizer.

## Model

The closed fact vocabulary covers `Contiguous`, `Bounded`, `Mutable`,
`Readonly`, `Aligned(N)`, `RegionIdentity`, `SameRegion`, `Disjoint`,
`FixedShape`, `RuntimeShape`, `Rank(N)`, internal `Shape`, `NoAllocation`,
`NoCopy`, `NoOwnershipTransfer`, and the existing `LifetimeSafe`,
`NonEscaping`, and `Outlives` analyses. Facts are typed records, not strings.

Every fact has typed subjects, optional integer parameters, a
`Proven`/`Disproven`/`Unknown` certainty, an origin, deterministic identity,
and bounded evidence such as region IDs, interval, alignment, rank, or shape.
Unknown never satisfies a required proof.

Facts may describe a type guarantee or one concrete value/region. For example,
all `Span<int>` values are contiguous and rank one, but `Aligned(span, 32)` is
value-level because a Subspan offset can reduce the safe alignment. R4j does
not promote instance evidence into a type fact.

## Qualification

- arrays and ndarrays contribute contiguous row-major storage, bounded extent,
  rank, shape class, and inline/external storage evidence;
- layouts contribute stable region identity, byte interval, alignment, and
  declared sibling disjointness;
- `bind` preserves source identity and records no allocation, copy, or
  ownership transfer;
- streams preserve the mapped layout region while adding only channel identity;
- Span preserves its parent region, narrows the half-open interval, and reduces
  alignment conservatively with the offset;
- tensor centrally qualifies its closed `Inline`, `NDArray`, `BoundNDArray`,
  `Span`, `LayoutRegion`, or `StreamChannel` backing witness, adding rank and
  mathematical shape without replacing storage facts.

Only bounded implications are admitted: stronger power-of-two alignment
implies a divisible weaker alignment, fixed shape implies bounded extent,
readonly disproves mutable, distinct declared layout regions are disjoint, and
constant non-overlapping intervals in one parent region are disjoint. Runtime
overlap without existing constraints remains unknown; there is no alias solver.

## Consumers

Concept requirements can request unary type guarantees and parameterized
`Aligned<T>(N)` or `Rank<T>(N)` facts. The existing operation-subject mechanism
supports bounded relational requirements; `SameRegion(source, result)` consumes
preserved call-result provenance, while a `Disjoint(left, right)` request over
unconstrained parameters fails as unknown.

MIR contains a deterministic `semantic_facts` registry plus enriched
`semantic_proofs`. `SemanticFactSet` exposes `FactsFor`, `Prove`,
`KnownAlignment`, `RegionOf`, and `AreDisjoint`; future passes consume this
registry without rerunning front-end inference.

A future vectorization policy can require contiguous, sufficiently aligned
inputs and output, plus proven output/input disjointness. A future Fortran-like
mutable output plus readonly inputs can yield noalias-like lowering only when
region identity and disjointness prove it. Concept does not add `restrict`, a
`noalias` keyword, SIMD, tiling, GPU lowering, MLIR, or speculative heuristics
in R4j.

Exact variadic source-level `Shape<T>(...)` and broader local-value subjects are
deliberately deferred. Exact shape evidence is already retained in MIR.

