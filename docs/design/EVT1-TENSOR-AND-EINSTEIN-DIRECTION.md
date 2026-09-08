# EVT1 tensor and Einstein direction

Status: R4j fact-qualified bounded implementation

## Purpose and lineage

R4h generalizes the SDSL-V pressure for indexed shader data and avoidance of
manual nested comptime loops into Core Concept semantics. The repository
retains SDSL-V's separation between semantic layout/channel identity and
physical resource allocation, but no closed prior tensor implementation was
available to port. The active Go compiler therefore supplies the executable
authority and classifies the surface as EVT1-new / SDSL-V-derived.

The governing distinction is:

```text
array / ndarray             physical contiguous storage
layout                      semantic memory geometry
bind                        whole-storage association / reshape
stream                      semantic channel mapping
Span / ReadOnlySpan         bounded borrowed intervals
tensor                      mathematical indexed interpretation
```

Tensor is not a storage container. Construction neither allocates nor copies,
does not transfer ownership, and preserves the backing region, base offset,
alignment, mutability, shape, contiguity, and lifetime provenance.

## Source model

The canonical type and constructor are:

```concept
tensor<int, 2> A = Tensor(aStorage);
```

The destination fixes element type and positive rank. The source supplies the
shape and storage facts. Fixed/bound ndarrays and shaped layout/stream regions
are direct sources. Span supplies exactly one dimension and therefore forms
only rank one; higher-rank Span conversion requires a future explicit shape
operation and is never inferred from length.

R4i adds the concrete fixed form:

```concept
tensor<int> A[2, 2] = [[1, 2], [3, 4]];
```

This follows the explicit-decision/automatic-consequence doctrine: source
states element type and shape, and the compiler derives rank and one fixed
inline ndarray backing because no allocator or storage-source decision remains.
The backing is a normal local/global value region, and the tensor is the
ordinary non-owning view over it. Runtime/external storage continues to require
`Tensor(source)`.

Nested literals reuse ndarray validation. An exact scalar initializer fills
the fixed backing once per element as initialization sugar. Omission follows
the existing rule that fixed locals require an initializer. Each synthesized
region has a deterministic source-derived identity such as
`inline:A#storage@line:column`; distinct declarations are disjoint. `const`
propagates readonly access to both backing and view. Structural element
copy/move/drop law remains the ndarray law; the view never independently drops
the backing.

`vector<T>` and `matrix<T>` normalize exactly to `tensor<T, 1>` and
`tensor<T, 2>` before semantic use. They add neither types nor witnesses nor
MIR systems. Rank mismatch is diagnosed at the shaped declaration.

Ordinary `A[0, 1]` indexing is zero-based and rank-exact. Whole-tensor
assignment computes into existing mutable storage. An expression never creates
an owned result tensor.

## Symbolic state and validation

An indexed assignment introduces one bounded statement-local index scope:

```concept
C[i, j] = A[i, k] * B[k, j];
```

`i`, `j`, and `k` are compiler-owned index symbols, not runtime locals. Their
conceptual state is `extent(name)` plus the current loop value. LHS symbols are
unique free indices. RHS-only symbols are reductions and occur exactly twice
in the bounded subset. All occurrences must agree on extent; every LHS symbol
must appear on the RHS. Symbols disappear after the statement.

This same form supports fixed initialization:

```concept
T[i, j] = if (i == j) 1 else 0;
```

The bounded comptime evaluator executes the validated index plan over the
aliased fixed storage. No persistent hidden program state is introduced. A
future explicit `indices(...)` block may expose configuration if a concrete
consumer needs it; R4h does not canonize speculative syntax.

## Elementwise and contraction laws

Outside indexed notation, `+`, `-`, and `*` are elementwise and require exact
operand/destination shapes. Exact-element-type scalar multiplication is
allowed. Scalar application is not broadcasting, and no singleton, row, or
column expansion is inferred.

`@` contracts the last axis of its left operand with the first axis of its
right operand. The output concatenates all remaining axes. Matrix multiply is
the rank-two special case; higher ranks use the same law. Einstein notation is
the explicit general form. Noncanonical/multi-axis convenience syntax is
deferred because indexed notation already expresses the operation.

Reduction is multiplication plus addition from arithmetic zero for the
supported scalar types. General semirings are future concept pressure, not an
R4h typeclass project.

The rank-one/rank-one case has no remaining free axis and therefore yields the
element scalar directly. Tensor MIR retains the two rank-one region operands,
the reduction axis, element type, and multiply-add accumulator while giving the
result rank zero. No source-level rank-zero tensor object is introduced.

## Tensor MIR and lowering

Tensor views remain ordinary MIR operations with complete storage facts.
Validated computations also appear in `MIRFunction.tensor_operations` with:

- output and operand tensors;
- element types, rank, ordered shape, region, alignment, mutability, and
  provenance;
- symbolic operand maps and free/reduction sets;
- shape guards and alias policy;
- the dedicated tensor-lowering contract.

Only after MIR validation does tensor lowering synthesize loops. Elementwise
operations use a checked linear traversal. Contractions use ordered free loops,
an arithmetic-zero accumulator, and ordered reduction loops. The result then
flows through the existing strict-C11 backend. This boundary preserves future
tiling, vectorization, GEMM recognition, GPU, or native-LIR opportunities
without implementing them now.

Inline declarations add `tensor_inline_storage` and the same `tensor_view`
operation to ordinary MIR. Vector/matrix spellings are already normalized.
Tensor computation still has exactly one Tensor MIR system. C lowering emits a
fixed ndarray wrapper and a descriptor pointing at its data; scalar fill and
literal initialization write that fixed value storage with no allocation.

Backing origin is a closed semantic classification: `Inline`, `NDArray`,
`BoundNDArray`, `Span`, `LayoutRegion`, or `StreamChannel`. The corresponding
`TensorBacking` evidence supplies shape, contiguous storage, provenance,
region identity, alignment, and mutability. Tensor algebra consumes that
evidence without runtime witness objects or source-level backing matching.

## Alias and runtime policy

Elementwise corresponding-index in-place writes are allowed. Contraction
rejects destination/input region overlap; no heap or hidden scratch tensor is
created. R4f distinct region identities and R4g parent-region facts are reused
as evidence. This is a conservative bounded policy, not a general alias solver.

Fixed shape incompatibility rejects statically. Runtime shape equality,
contraction axes, output axes, index bounds, and shape products are checked
before access with deterministic terminal reasons.

## Ergonomics evidence and deferred direction

Before R4i, fixed matrix multiplication required three storage declarations
and three `Tensor(...)` views. The same program is now:

```concept
matrix<int> A[2, 2] = [[1, 2], [3, 4]];
matrix<int> B[2, 2] = [[5, 6], [7, 8]];
matrix<int> C[2, 2] = 0;
C = A @ B;
```

The generalized spelling remains visible when it matters:

```concept
tensor<int> A[2, 2, 2] = [[[1, 2], [3, 4]], [[5, 6], [7, 8]]];
```

A future tensor group/columnar record may group related shaped fields and must
lower through existing layout semantics. R4i does not implement that syntax or
another storage system. Explicit noncanonical contraction syntax,
broadcasting, strides, transposes, slicing, sparse/tiled storage, owned dynamic
tensor allocation, named axes, autograd, BLAS, SIMD, GPU kernels, MLIR, and
generalized semirings remain deferred.

## R4j backing qualification

R4j centralizes fact qualification around the closed `TensorBackingKind`
classification. `Inline`, `NDArray`, `BoundNDArray`, `Span`, `LayoutRegion`,
and `StreamChannel` each supply one backing witness; tensor adds rank and
mathematical shape while preserving region identity, interval, alignment,
contiguity, mutability, provenance, and no-allocation/no-copy/no-transfer
evidence. Inline backing supplies a distinct stable region and fixed shape;
runtime-bound backing supplies explicit runtime shape.

Future tensor optimization consumes these MIR facts and proven operand/output
disjointness. R4j does not alter contraction lowering and adds no vectorizer,
SIMD, BLAS, GPU, MLIR, or noalias syntax.
