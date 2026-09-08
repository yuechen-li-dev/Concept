# EVT1 R4e conformance record

Status: complete bounded corpus for explicit storage binding

Baseline commit: `cb98af21b02968ff96d0b6e0cabe7d51306fdc81`

Compiler: `concept-evt1-stage0-go`

Baseline repository test inventory: 89 top-level Go test functions. The R4d
corpus was 30/30 `PASS` (17 valid, 13 invalid), and both Zig suites passed.

## Corpus and classification

R4e contains 23 readable Concept sources under `language/evt1-r4e/core`:

- 12 valid cases;
- 9 statically invalid cases with exact diagnostics;
- 2 compile-valid/runtime-invalid terminal-panic cases.

All 23 classify as `PASS`. They cover fixed and runtime array-to-ndarray
binding, ndarray reshape and flattening, mutable and readonly aliases,
runtime rank/shape queries, scoped and call-result provenance, ref-struct
containment, rebinding an existing runtime view, binding after `?`, contextual
target/source/element/const failures, fixed size mismatch, lifetime escape,
runtime size mismatch, and shape-product overflow.

R4e is EVT1-new. PoC3 Phase 21 supplies supporting fixed-wrapper, contiguous
indexing, and value-semantics pressure only. R4a/R4b supply provenance law and
R4d supplies storage identity, shape, indexing, and row-major layout. There is
no claimed PoC3 `bind` parity.

## MIR and C evidence

MIR retains `bind_storage` with source/target storage kind, target rank and
shape, fixed proof or runtime check mode, mutability, source provenance, and
explicit no-copy/no-allocation/no-ownership-transfer facts. MIR validation
rejects incomplete binding records.

The strict-C11 backend emits a non-owning local descriptor containing an
element pointer and inline shape dimensions. Runtime multiplication checks
`SIZE_MAX` before exact equality with the source count. The descriptor points
at the fixed wrapper or prior bound descriptor's existing `data`; generated
R4e output contains no `malloc`, allocator, VLA, `memcpy`, Span, Slice, or
tensor mechanism.

Twelve successful native fixtures prove fixed reshape, runtime shape, ndarray
reshape, flattening, bidirectional source/view mutation visibility, readonly
access, runtime `Rank`/`Shape`, scoped use, helper-result provenance,
runtime-view rebinding, ref-struct containment, and bind after Result `?`.
Two terminal native fixtures prove deterministic runtime mismatch and
shape-product-overflow panic reasons.

## Diagnostics

| Semantic family | Code | Rule |
| --- | --- | --- |
| bind requires contextual target | `CV4562` | destination supplies shape |
| bind requires array target | `CV4563` | target is ref/ref const storage |
| bind requires storage | `CV4564` | source is contiguous array/ndarray |
| bind element type mismatch | `CV4565` | exact identity only |
| bind shape size mismatch | `CV4566` | fixed totals must match |
| bind mutable from const | `CV4567` | const cannot gain mutability |
| bind MIR invalid | `CV4568` | required MIR facts are present |

Existing `CV4513`, `CV4511`, and `CV4522` continue to diagnose mutation through
`ref const`, local escape, and scoped escape. Runtime mismatch and overflow use
stable terminal reasons, not Result or exception control flow.

## Scope boundary

`bind` consumes exactly the entire contiguous source. R4e does not implement
Span/ReadOnlySpan, Slice, FixedBuffer, raw-pointer binding, offsets, subregions,
strides, alternate layouts, owned dynamic arrays, allocators, stack allocation,
vector/matrix/tensor semantics, Einstein notation, alias analysis, SIMD, GPU,
or native-backend optimization.

The recommended next milestone is R4f: specify provenance-preserving bounded
subregions with Span/ReadOnlySpan while retaining `bind` as exact whole-storage
reshape and keeping allocation and mathematical interpretation separate.
