# EVT1 R4i conformance record

Status: complete bounded corpus for inline tensor declaration ergonomics,
vector/matrix aliases, and scalar contraction results

R4i is implemented by `concept-evt1-stage0-go` through the active
parse/validate/storage-witness/Tensor-MIR/tensor-lowering/strict-C11 path.

## Corpus and classification

The corpus contains 27 `PASS` cases under `language/evt1-r4i/core`: 16 valid
programs and 11 static rejection programs. There are zero
`EXPECTED-DIVERGENCE`, `EVT1-GAP`, `POC3-LEGACY`, or `SPEC-AMBIGUITY` cases.
R4i is EVT1-new ergonomic sugar over R4h semantics. Missing PoC3 source syntax
is not classified as semantic divergence.

Valid evidence covers fixed inline rank one through three, nested literals,
exact scalar fill, readonly `const`, Einstein initialization, matrix and
rank-three `@`, exact `vector<T>`/`matrix<T>` aliases, API parameter identity,
rank-one scalar dot product, and preserved ndarray/layout/stream
`Tensor(source)` construction. The R4h suite continues to cover bound ndarray,
Span, and runtime-view behavior.

Static rejection covers runtime inline shape, nested literal mismatch,
unsupported tensor element type, invalid/zero shape, scalar-fill mismatch,
vector/matrix rank mismatch, scalar-result type mismatch, readonly destination,
broadcasting, and inline contraction destination overlap.

## Before and after

Before R4i:

```concept
int<ndarray>[2, 2] aStorage = [[1, 2], [3, 4]];
int<ndarray>[2, 2] bStorage = [[5, 6], [7, 8]];
int<ndarray>[2, 2] cStorage = [[0, 0], [0, 0]];
tensor<int, 2> A = Tensor(aStorage);
tensor<int, 2> B = Tensor(bStorage);
tensor<int, 2> C = Tensor(cStorage);
C = A @ B;
```

After R4i:

```concept
matrix<int> A[2, 2] = [[1, 2], [3, 4]];
matrix<int> B[2, 2] = [[5, 6], [7, 8]];
matrix<int> C[2, 2] = 0;
C = A @ B;
```

Generalized tensors retain explicit shape: `tensor<int> A[2, 2, 2] = ...;`.

## MIR and native evidence

Ordinary MIR exposes `tensor_inline_storage` plus `tensor_view`. Each backing
has a deterministic source-derived region identity, fixed ndarray type,
ordered shape, alignment, mutability, local provenance, and no-allocation
fact. Views retain same-region/no-copy/no-allocation/no-transfer facts.
Independent declarations have distinct region identities.

Tensor operations remain in the R4h `tensor_operations` collection. Matrix and
rank-three programs use `tensor_contract`; dot product uses
`tensor_scalar_contract` with a rank-zero output and two rank-one region
operands. There is no Vector or Matrix MIR.

Twelve strict-C11 harnesses execute inline indexing, literals, scalar fill,
Einstein fill, matrix/rank-three contraction, aliases, and dot product. Dot
product returns 32, matrix multiplication returns 50, and rank-three
contraction returns 46. Generated tensor C is checked for absence of `malloc`,
`calloc`, `realloc`, `memcpy`, BLAS, and MLIR mechanisms.

## Diagnostics and boundary

R4i adds `TENSOR_INLINE_REQUIRES_FIXED_SHAPE`, `VECTOR_RANK_MISMATCH`, and
`MATRIX_RANK_MISMATCH`, while reusing ndarray literal and R4h tensor element,
shape, broadcast, alias, and MIR families. The reserved inline-shape,
inline-element, and scalar-result families keep the bounded diagnostic ledger
closed.

R4i adds no runtime-owned allocation, tensor group/record syntax, broadcasting,
strides, transpose, slicing, named axes, autograd, BLAS, SIMD, GPU lowering,
native LIR, `dyn`, or allocator policy. A future tensor group/columnar record
must lower through existing layout semantics rather than create storage.
