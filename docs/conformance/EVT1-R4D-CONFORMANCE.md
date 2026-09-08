# EVT1 R4d conformance record

Baseline: `df1432dab052bddd78c3f5d877d9d1d7cc007a1c`

Compiler: `concept-evt1-stage0-go`

R4c baseline: 24/24 PASS (14 valid, 10 invalid).

Baseline repository test inventory: 84 top-level Go test functions; the R4d
implementation adds five top-level Go tests (89 current).

## Classification

The R4d corpus contains 30 readable Concept sources:

- 30 PASS;
- 0 EXPECTED-DIVERGENCE;
- 0 EVT1-GAP;
- 0 POC3-LEGACY;
- 0 SPEC-AMBIGUITY.

There are 17 valid and 13 invalid cases. Fixed rank-1 literals, value copy,
index mutation, `Len`, bounds panic, and wrapper lowering cite PoC3 Phase 21
evidence. Ndarray, rank-aware comma indexing, rectangular flattening,
`Rank`/`Shape`, and row-major storage are EVT1-new/SDSL-V-derived semantics.
This classification does not pretend PoC3 implemented ndarray.

## Executable evidence

`internal/concept/r4d_conformance_test.go` checks every source and exact invalid
diagnostic category. MIR inspection requires a rank-3 ndarray storage record
with ordered `[2,3,4]` shape, `contiguous=true`, `layout=row-major`, and
`ownership=fixed_inline`; it also requires `ndarray_index`, `rank_query`, and
`shape_query` operations. The same source statically evaluates a fixed rank-2
ndarray literal, both shape dimensions, rank, and a comma-separated index.

Sixteen successful native strict-C11 specimens cover fixed reads and writes,
copy independence, rank-2 indexing and mutation, nested ndarray literals,
rank/shape queries, mutable array reference projection, const ndarray reference
projection, Option/Result array payload lowering, movable-only elements,
owned-array move/drop, and reverse element-drop traversal. A
separate native process calls a fixed array with a runtime out-of-range index
and must abort with `Concept array index out of bounds` and source coordinates.

The generated fixed representation is an assignable wrapper with `data[N]`.
Ndarray uses one flat `data[N0 * ... * Nk]` member. No generated R4d artifact
contains `malloc`, an allocator hook, a C variable-length array, Slice,
FixedBuffer, Span, or tensor lowering.

## Diagnostics

The active stable categories exercised by R4d are:

| Semantic family | Code | Evidence |
| --- | --- | --- |
| const indexed mutation | `CV4128` | array const projection |
| array element noncopyable | `CV4133` | structural copy rejection |
| array element immovable | `CV4134` | structural final-storage rejection |
| array index type invalid | `CV4232` | non-integer index |
| array index out of bounds | `CV4233` | fixed constant index |
| ndarray literal rank mismatch | `CV4554` | rank validator |
| ndarray literal shape mismatch | `CV4555` | rectangular wrong shape |
| ndarray literal ragged | `CV4556` | unequal nested row shapes |
| ndarray index arity mismatch | `CV4557` | too few/too many indices |
| runtime array requires explicit storage | `CV4558` | runtime extent local |
| runtime ndarray requires explicit storage | `CV4559` | runtime shape local |
| ndarray index out of bounds | `CV4561` | fixed constant dimension OOB |

Existing `CV4231` continues to cover invalid storage receivers. Diagnostic
semantic families are authoritative; numeric
renumbering remains deferred.

## Scope boundary

R4d does not implement runtime storage construction. It types runtime extent
expressions and then rejects a bare owning local because there is no explicit
storage source. It also does not implement Slice, FixedBuffer, Span,
ReadOnlySpan, vector, matrix, tensor, allocator APIs, stack allocation,
alternative layouts, alias analysis, SIMD, GPU lowering, or TensorIR.
