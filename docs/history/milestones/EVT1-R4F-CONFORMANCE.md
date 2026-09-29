# EVT1 R4f conformance record

Status: complete bounded corpus for semantic layout declarations and zero-storage streams

R4f is implemented by `concept-evt1-stage0-go`. It uses the active
parse/validate/MIR/strict-C11 path; Oct/SDSL-V was inspected only for semantic
lineage and is not a build or runtime dependency.

## Corpus and classification

R4f contains 29 `PASS` cases under `language/evt1/storage/layout-stream`: 16 valid programs
and 13 invalid programs. The valid corpus covers fixed scalar/array/ndarray
regions, canonical and explicit geometry, compile-time queries, exact backing
binding, mutable and const projections, basic and multi-channel streams,
layout-to-stream binding, channel alias visibility, lexical/call-result
provenance, and scoped provenance. The invalid corpus covers duplicate or
unknown declarations, invalid/unsupported geometry, overlap, size/alignment
mismatch, layout mismatch, const escalation, const-channel mutation, and local
lifetime escape.

The layout surface is EVT1-new but informed by SDSL-V's semantic layout graph.
The zero-storage channel mapping is SDSL-derived. Binding, constness, and
provenance are compositions of R3, R4b, R4d, and R4e laws; no parity is claimed
for a source-language SDSL runtime.

## Semantic and MIR evidence

The typed model retains each layout's final byte size/alignment and each
region's stable identity, ordinary Core type, byte offset, byte extent,
alignment, and disjoint region set. Each stream retains its source layout and
channel-to-region mapping and is explicitly marked `zero_storage`.

MIR retains `layout_bind`, `stream_bind`, and `region_projection` operations
with backing identity, mutability, provenance, byte geometry, and the facts
`no_copy`, `no_allocation`, `no_ownership_transfer`, and `same_backing`.
Validation rejects incomplete or inconsistent graphs and operations.

Strict-C11 harnesses execute ndarray layout access, elevated alignment, mutable
region aliases, stream-channel aliases, const reads, and helper-return
provenance. Generated output is checked for the absence of `malloc`, `calloc`,
`realloc`, `memcpy`, vtables, and Span machinery. An alignment harness verifies
the executable fixed backing guarantee.

## Diagnostic families

| Family | Codes | Law |
|---|---|---|
| layout declaration/geometry | `CV4570`-`CV4576` | unique fixed regions; valid aligned, non-overlapping geometry |
| layout binding | `CV4577`-`CV4579`, `CV4590`-`CV4591` | reference target; contiguous source; const, exact-size, and alignment preservation |
| stream declaration/binding | `CV4580`-`CV4586` | known layout/regions, unique channels, matching referenced layout, no const escalation |
| bounded queries and MIR | `CV4592`-`CV4594` | closed layout queries and complete semantic graphs |

Existing `CV4511`, `CV4513`, and `CV4522` remain authoritative for outward
escape, mutation through `ref const`, and scoped escape. R4f does not create a
parallel lifetime or const diagnostic system.

## Scope boundary

Only fixed compile-time layouts and zero-storage streams are admitted. Runtime
layout parameters, Span/ReadOnlySpan, Slice, FixedBuffer, allocation,
`stackalloc`, raw-pointer binding, stream execution/composition, mathematical
tensor interpretation, general reflection, GPU lowering, and stable ABI law
remain deferred.

The next milestone should be chosen from evidence after R4f; this record does
not pre-authorize Span or another view system.
