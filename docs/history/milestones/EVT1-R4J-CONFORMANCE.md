# EVT1 R4j conformance record

Status: complete bounded semantic-fact qualification corpus

R4j is implemented by `concept-evt1-stage0-go` through the active semantic
analysis, existing region/view/tensor witnesses, deterministic MIR, and
strict-C11 erasure path.

## Corpus

The canonical corpus contains 24 `PASS` cases: 17 valid programs and 7 static
rejection programs. It covers array, Span, tensor, layout, bind, stream,
readonly, alignment, fixed/runtime shape, rank, same-region, declared-region
disjointness, independent inline tensors, fixed Subspan intervals, concept
consumption, invalid parameters/subjects, and conservative unknown overlap.

## Evidence

Focused tests prove that an aligned layout region retains region, provenance,
contiguity, boundedness, mutability, alignment, and storage-neutrality through
stream channel, Span, and tensor construction. An offset-one `int` Subspan of
a 64-byte-aligned region degrades safely to four-byte alignment. Two fixed
non-overlapping Subspans are disjoint; runtime overlap is not falsely proven.

Inline matrix multiplication exposes fixed shapes, exact ranks/shapes, distinct
regions, alignment, no-allocation views, and pairwise disjoint backing. A
runtime-bound matrix exposes `RuntimeShape`, not `FixedShape`.

Semantic proof and fact IDs are deterministic. MIR validation rejects malformed
facts as `SEMANTIC_FACT_MIR_INVALID`. Generated C contains no fact tables,
allocator calls, copying helpers, noalias syntax, SIMD, or vectorizer runtime.

## Boundary

R4j does not expose variadic source-level exact-shape requirements, generalized
local-value concept subjects, a general alias solver, optimization passes,
SIMD, tiling, GPU lowering, MLIR, LIR, allocator policy, or source `noalias`.

## Recommended next milestone

R4k should add a bounded, non-transforming optimization-eligibility consumer
that reads `SemanticFactSet` for existing tensor loop plans and reports exactly
which contiguity, alignment, shape, mutability, and disjointness proofs admit or
deny a candidate. It should first prove that no front-end fact is reconstructed;
SIMD generation, loop rewriting, and native lowering should remain later work.
