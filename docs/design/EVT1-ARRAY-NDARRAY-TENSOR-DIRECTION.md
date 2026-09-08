# EVT1 array, ndarray, and tensor direction

Status: R4f semantic layout regions canonical; spans, mathematics, and optimization deferred

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
storage without changing array, ndarray, or future tensor meaning.

## Deferred work

R4f does not port PoC3 Slice or FixedBuffer and does not implement Span,
ReadOnlySpan, vector, matrix, tensor, Einstein notation, TensorIR, an allocator
framework, stack allocation, `dyn`, generalized alias/noalias analysis,
strided/sparse/tiled layouts, SIMD, GPU lowering, or a native backend. Those
features must consume the storage facts established here without changing
their meaning retroactively.

## Subsequent milestone boundary

Future Span/ReadOnlySpan work may specify arbitrary bounded subregions on the
same provenance machinery, but must not reinterpret R4f declared geometry as a
runtime slicing facility. Allocator-backed owned dynamic storage and
mathematical vector/matrix/tensor interpretations remain separate work.
