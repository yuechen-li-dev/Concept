# EVT1 tensor and Einstein direction

Status: R4h bounded implementation

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

## Alias and runtime policy

Elementwise corresponding-index in-place writes are allowed. Contraction
rejects destination/input region overlap; no heap or hidden scratch tensor is
created. R4f distinct region identities and R4g parent-region facts are reused
as evidence. This is a conservative bounded policy, not a general alias solver.

Fixed shape incompatibility rejects statically. Runtime shape equality,
contraction axes, output axes, index bounds, and shape products are checked
before access with deterministic terminal reasons.

## Deferred direction

`vector<T>` and `matrix<T>` are future spelling-only shorthands for
`tensor<T,1>` and `tensor<T,2>`. Rank-zero tensor results, explicit
noncanonical contraction syntax, broadcasting, strides, sparse/tiled storage,
owned dynamic tensor allocation, named axes, autograd, BLAS, SIMD, GPU kernels,
MLIR, and generalized semirings remain outside R4h.
