# EVT1 R4h conformance record

Status: complete bounded corpus for tensor views, Einstein indexing, and
generalized contraction

R4h is implemented by `concept-evt1-stage0-go` through the active
parse/validate/Tensor-MIR/tensor-lowering/strict-C11 path.

## Corpus and classification

The corpus contains 29 `PASS` cases under `language/evt1/tensor/semantics`: 17 valid
programs and 12 static rejection programs. There are zero
`EXPECTED-DIVERGENCE`, `EVT1-GAP`, `POC3-LEGACY`, or `SPEC-AMBIGUITY` cases.
PoC3 has no tensor parity claim. SDSL-V contributes semantic lineage for
indexed data and compile-time population; the generalized Core surface is
EVT1-new / SDSL-V-derived.

Valid evidence covers rank-one through rank-three construction, bound ndarray,
layout region, stream channel, rank-one Span, zero-based indexing, exact-shape
add/multiply, scalar multiply, rank-two and rank-three `@`, Einstein matrix
contraction, fixed symbolic comptime initialization, readonly input, and
runtime-shaped contraction guards.

Static rejection covers rank/shape mismatch, attempted broadcasting, readonly
destination, contraction mismatch, symbolic extent mismatch, invalid output
and reduction indices, contraction aliasing, rank-two construction from a
shape-less Span, local tensor lifetime escape, and symbolic-index escape from
its statement.

## MIR and native evidence

MIR exposes storage-neutral `tensor_view` operations and separate validated
`tensor_operations`. Matrix Einstein evidence records free `[i,j]`, reduction
`[k]`, indexed operands, output and input region identities, shape, alignment,
mutability, provenance, alias policy, and the dedicated loop-lowering stage.
Malformed Tensor MIR rejects with `TENSOR_MIR_INVALID`.

Fifteen strict-C11 harnesses compare integer and floating-point numeric results for construction,
elementwise, scalar, matrix/higher-rank contraction, Einstein contraction, and
readonly composition. Separate child processes prove the tensor index and
runtime contraction mismatch terminal reasons. Generated C is checked for the
absence of allocation/copy helpers, BLAS, and MLIR paths.

The comptime identity source contains symbolic indexed assignment and no
handwritten loop. Its `static_assert` succeeds, and MIR retains the
`tensor_index_assign` plan in the comptime function.

## Diagnostic families

| Codes | Stable family |
|---|---|
| `CV4610`-`CV4615` | tensor type, destination, source, rank, and element laws |
| `CV4616`-`CV4618` | destination mutability, exact shape, and index arity |
| `CV4619`-`CV4622` | symbolic index, extent, output, and reduction laws |
| `CV4624` | generalized contraction shape mismatch |
| `CV4625` | contraction alias hazard |
| `CV4626` | malformed Tensor MIR |

## Boundary

R4h includes no vector/matrix aliases, broadcasting, rank-zero tensor, dynamic
tensor allocator/runtime, arbitrary strides, sparse tensors, named axes,
autograd, BLAS, SIMD, GPU kernel generation, MLIR, or generalized semiring
abstraction. R4i should remain a bounded shorthand/scalar-result qualification
pass if consumer evidence warrants it.
