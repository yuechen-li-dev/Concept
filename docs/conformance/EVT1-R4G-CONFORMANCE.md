# EVT1 R4g conformance record

Status: complete bounded corpus for provenance-preserving Span and ReadOnlySpan

R4g is implemented by `concept-evt1-stage0-go` through the active
parse/validate/MIR/strict-C11 path. It adds no dependency on Oct or the retired
PoC3 compiler.

## Corpus and classification

R4g contains 29 `PASS` cases under `language/evt1-r4g/core`: 15 valid programs
and 14 invalid-path programs. Two invalid-path programs compile intentionally
and are executed as isolated terminal panic evidence. There are zero
`EXPECTED-DIVERGENCE`, `EVT1-GAP`, `POC3-LEGACY`, or `SPEC-AMBIGUITY` cases.

The corpus covers Span from fixed array, ndarray, runtime-shaped bound ndarray,
layout region, and stream channel; bidirectional mutation visibility; readonly
construction and conversion; Subspan and empty end intervals; Len; scoped and
helper provenance; immovable backing; and Result `?` composition. Rejections
cover const escalation, readonly mutation, constant bounds/index failure,
local/helper/layout/stream lifetime escape, Result lifetime laundering,
noncontiguous sources, element mismatch, and readonly-to-mutable conversion.

PoC3 Slice fixtures are design pressure for pointer-plus-length representation,
bounds checks, and readonly access only. The canonical surface and its
provenance/region laws are EVT1-new and C#-inspired, composed with R4a/R4b
lifetime evidence, R4d storage, R4e bind, and R4f layout/stream facts. R4g is
not classified as a direct Slice port.

## Semantic and MIR evidence

`Span<T>` and `ReadOnlySpan<T>` are compiler-known applied types treated as
ref-struct-like lifetime-bound descriptors. Each validated construction or
narrowing retains element type, lexical/call-result provenance, stable parent
region identity, backing byte offset, relative element offset, length, byte
extent, safe alignment, mutability, and contiguity.

MIR exposes `span_from_region`, `span_to_readonly`, `span_subregion`, and
`span_index`. Validation requires region identity, provenance, alignment,
mutability, contiguity, same-backing identity, bounds mode, and the facts
`no_copy`, `no_allocation`, and `no_ownership_transfer`. Subspan keeps the
parent region ID and composes a half-open relative offset rather than creating
an independent region.

## Native evidence

Fifteen strict-C11 success harnesses cover all construction sources, aliasing
in both directions, correct subspan writes, readonly reads, explicit
Span-to-ReadOnlySpan conversion, empty intervals, Len, scoped/helper
provenance, immovable backing, and Result extraction. Separate child processes
prove deterministic `Concept span bounds out of range` and `Concept span index
out of bounds` terminal paths.

Generated C uses one typed pointer plus `size_t length`; readonly pointers are
const-qualified. Tests reject `malloc`, `calloc`, `realloc`, and `memcpy` in
the generated evidence. Span descriptors have no Drop obligation and backing
ownership remains unchanged.

## Diagnostics

| Family | Codes | Law |
|---|---|---|
| source and element | `CV4600`-`CV4601` | known contiguous source and exact element identity |
| mutability | `CV4602`, `CV4605` | no const escalation or readonly mutation |
| interval/index | `CV4603`-`CV4604` | static rejection and guarded runtime half-open bounds |
| arithmetic | `CV4606`-`CV4607` | no element-to-byte wraparound |
| MIR | `CV4608` | complete bounded borrowed-region facts |

Existing `CV4521`, `CV4522`, and R4b call-result provenance checks remain the
authority for lifetime escape. Span does not create parallel lifetime
diagnostics for those cases.

## Deliberate boundary and next milestone

R4g does not implement NDSpan, strides, rank-aware span indexing, range or
iterator frameworks, vector/matrix/tensor mathematics, Einstein notation,
allocator-backed ownership, stack allocation, raw-pointer construction,
general alias/noalias solving, dyn, named lifetimes, NLL, or another borrow
checker.

The recommended next milestone is a separate R4h design-and-evidence pass for
the first mathematical interpretation consumer over existing storage/spans.
It should select one bounded vector or matrix case only after auditing the
consumer requirement, and it must leave R4g storage ownership, provenance,
region identity, and bounds laws unchanged.
