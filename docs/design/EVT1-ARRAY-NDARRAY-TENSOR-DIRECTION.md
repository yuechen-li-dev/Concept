# EVT1 array, ndarray, and tensor direction

Status: R4j semantic facts canonical; optimization deferred

## Layering

```text
storage
  array       rank-1 contiguous storage
  ndarray     rank-N contiguous storage

storage binding
  bind        entire-storage non-owning reshape

semantic geometry
  layout      fixed named regions over one bound storage object
  stream      zero-storage channels aliasing layout regions

borrowed views
  Span<T>           mutable bounded contiguous subregion
  ReadOnlySpan<T>   readonly bounded contiguous subregion

mathematical interpretations
  vector      exact rank-1 tensor shorthand
  matrix      exact rank-2 tensor shorthand
  tensor      canonical rank-N indexed algebra over existing storage

future optimization
  shape-aware lowering
  SIMD, GPU, and native backend selection
```

These layers must not collapse into one another. An ndarray is one contiguous
storage object, not `Array<Array<...>>`, and multidimensional shape does not
grant matrix/tensor algebra. Conversely, a tensor does not own storage merely
because it is a tensor. The R4h tensor interpretation refers to array/ndarray
storage or a borrowed view without redefining its allocation policy; future
vector and matrix aliases follow the same law.

## R4d storage law

The canonical source spellings are `T<array>[extent]` and
`T<ndarray>[extent0, extent1, ...]`. The former has rank exactly one. The latter
has rank at least one and is the canonical multidimensional family. The legacy
`T[N]` spelling remains a rank-1 compatibility form. Repeated legacy suffixes
remain nested arrays and are not equivalent to ndarray.

Rank, ordered shape dimensions, fixed/runtime extent status, contiguity, and
row-major layout survive semantic analysis into MIR. Array and ndarray are
contiguous by definition. Ndarray uses last-index-contiguous row-major
linearization; R4d deliberately has no layout policy parameter.

Fixed shapes are inline value storage. The backend uses an assignable wrapper
containing one C array, preventing C parameter decay from becoming Concept
semantics. An ndarray wrapper contains one flat scalar array. Copyability,
movability, immovability, const projection, references, Option/Result payloads,
and owned move/drop behavior compose through the existing structural rules.
Droppable fixed-storage elements are cleaned up in reverse linear order.

Runtime extent is a value-level shape fact, not an allocation request. The
semantic ownership categories are:

- fixed inline storage, executable in R4d;
- external/non-owning storage, supplied by explicit provenance-preserving bind
  descriptors and bounded Span views;
- owned dynamic storage, reserved for explicit allocator integration.

The array/ndarray type family does not select one universal ownership category.
Until an explicit storage source exists, a bare runtime-shaped declaration is
rejected. The compiler must not lower it to `malloc`, a hidden arena, a C VLA,
or an unproven pointer descriptor.

## R4e binding law

`bind source` connects an expected `ref` or `ref const` array/ndarray target to
the source's entire contiguous storage. The target supplies rank and shape;
the source supplies storage and lifetime provenance. Exact element identity and
exact total element count are required. Fixed counts are compile-time proofs;
runtime counts use overflow-safe deterministic guards.

Binding may reshape array to ndarray, flatten ndarray to array, or reshape one
ndarray into another without reordering elements. It creates only a non-owning
access path: no allocation, element copy, move, resize, ownership transfer, or
lifetime extension occurs. A bound view is storage interpretation, not tensor
semantics. Mutable and const access compose with ordinary place rules, while
scoped and call-result provenance remain attached to the original storage.

## Index and shape surface

Rank-1 storage uses `values[index]`. Ndarray uses one comma-separated index per
dimension, such as `values[row, column]`. Each index is integer, each fixed
constant is diagnosed when provably out of bounds, and runtime indexes take the
existing terminal panic path. `Len` remains rank-1 length; `Rank` and
`Shape(value, dimension)` expose bounded shape information without allocating a
shape object.

Nested ndarray literals are syntax sugar only. The compiler verifies rank,
rectangularity, dimensions, and element types, then flattens elements in
row-major source order into one ndarray value. Jagged storage is not part of
this family.

## R4f layout composition

A fixed array or ndarray may be one region in an R4f semantic layout. Layout
binding compares total byte extent and alignment against one whole contiguous
backing object; projection reuses the ordinary typed storage descriptor, shape,
row-major indexing, and provenance. This is fixed semantic geometry, not a
general subview: offsets and extents are compile-time declarations, and stream
channels only rename declared regions. Layout/stream compose above R4d/R4e
storage without changing array, ndarray, or the R4h tensor meaning layered above it.

## R4g bounded subregions

`Span<T>` and `ReadOnlySpan<T>` are rank-one borrowed views over a contiguous
source interval. They may consume whole array/ndarray storage, an R4e bound
view, or an R4f layout region/stream channel. Ndarray construction linearizes
the already row-major scalar storage and does not preserve rank-aware indexing.
`Subspan` narrows the same parent region using half-open bounds; it neither
reshapes whole storage nor introduces mathematical interpretation.

The resulting layering is:

```text
array / ndarray
    physical storage

layout
    memory geometry

bind
    whole-storage association

stream
    semantic channels

Span / ReadOnlySpan
    borrowed bounded subregions

vector / matrix / tensor
    mathematical interpretation
```

## R4h tensor interpretation

`tensor<T, Rank>` is the bounded mathematical layer above storage and
borrowed views. `Tensor(source)` preserves shape, region identity, base offset,
alignment, mutability, contiguity, and provenance while adding no allocation,
copy, ownership, or lifetime. Ordinary indices remain zero-based.

Exact-shape elementwise `+`, `-`, and `*`, exact-type scalar multiplication,
statement-local Einstein indexed assignment, and generalized last-axis/
first-axis `@` contraction lower through explicit Tensor MIR and a dedicated
loop-synthesis stage. Broadcasting is forbidden. Contraction destination/input
overlap is rejected conservatively; declared-disjoint R4f regions retain the
identities needed to prove separation.

R4i adds `tensor<T> name[shape...]` as declaration sugar only. It derives rank,
synthesizes the mechanically determined fixed `T<ndarray>[shape...]` backing,
initializes that storage, and forms the ordinary tensor view. Thus `ndarray`
remains the storage type, `tensor` remains the mathematical interpretation,
and the fixed shaped declaration connects them only where storage is fully
determined. Runtime and external sources remain explicit.

`vector<T>` and `matrix<T>` are exact spellings of `tensor<T, 1>` and
`tensor<T, 2>`, with no separate storage, type identity, algebra, witness, or
MIR. Scalar fill is fixed-storage initialization, not broadcasting.

## Deferred work

R4i does not port PoC3 Slice or FixedBuffer and does not implement an allocator
framework, stack-allocation surface, `dyn`, generalized alias/noalias analysis,
strided/sparse/tiled layouts, SIMD, GPU lowering, or a native backend. Those
features must consume the storage facts established here without changing
their meaning retroactively.

## Subsequent milestone boundary

R4j consumes the explicit backing witnesses for bounded fact qualification
without changing storage ownership. A later tensor group/columnar record
remains design direction only and
must reuse layout semantics. Allocator-backed owned dynamic storage remains
separate work.

R4j realizes the bounded qualification step: arrays/ndarrays contribute
contiguity, bounded extent, rank, and fixed/runtime shape; tensor backing
preserves those storage facts and adds mathematical interpretation. Facts are
typed MIR evidence for concept and future optimizer consumers, not runtime
metadata. Exact variadic source-level shape predicates remain deferred because
rank and full shape are already inspectable without adding awkward syntax.
