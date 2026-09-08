# EVT1 array, ndarray, and tensor direction

Status: R4d storage layer canonical; views, mathematics, and optimization deferred

## Layering

```text
storage
  array       rank-1 contiguous storage
  ndarray     rank-N contiguous storage

borrowed views
  Span<T>           deferred
  ReadOnlySpan<T>   deferred

mathematical interpretations
  vector      deferred rank-1 mathematics
  matrix      deferred rank-2 mathematics
  tensor      deferred rank-N indexed algebra

future optimization
  shape-aware lowering
  Einstein contraction and loop synthesis
  SIMD, GPU, and native backend selection
```

These layers must not collapse into one another. An ndarray is one contiguous
storage object, not `Array<Array<...>>`, and multidimensional shape does not
grant matrix/tensor algebra. Conversely, a tensor does not own storage merely
because it is a tensor. A future vector, matrix, or tensor interpretation may
refer to array/ndarray storage or a borrowed view without redefining its
allocation policy.

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
- external/non-owning storage, reserved for an explicit provenance-preserving
  constructor or future Span-based view;
- owned dynamic storage, reserved for explicit allocator integration.

The array/ndarray type family does not select one universal ownership category.
Until an explicit storage source exists, a bare runtime-shaped declaration is
rejected. The compiler must not lower it to `malloc`, a hidden arena, a C VLA,
or an unproven pointer descriptor.

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

## Deferred work

R4d does not port PoC3 Slice or FixedBuffer and does not implement Span,
ReadOnlySpan, vector, matrix, tensor, Einstein notation, TensorIR, an allocator
framework, stack allocation, `dyn`, generalized alias/noalias analysis,
strided/sparse/tiled layouts, SIMD, GPU lowering, or a native backend. Those
features must consume the storage facts established here without changing
their meaning retroactively.

## Recommended next milestone

R4e should close one boundary only: an explicit provenance-preserving way to
bind runtime shape to existing storage. It should choose whether that surface
is a restricted external descriptor constructor or the first Span/
ReadOnlySpan view milestone. It must reuse R4a/R4b lifetime provenance, reject
escape/laundering, and still introduce no allocator or mathematical tensor
semantics. Allocator-backed owned dynamic storage should remain later work.
