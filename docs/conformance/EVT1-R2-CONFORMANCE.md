# Concept EVT1 R2 value/place conformance

Status: complete bounded R2 evidence

- Baseline: `43fc5164f281a57432255aa9f27e52a8befb8904`
- Active compiler: `concept-evt1-stage0-go`
- Corpus: `language/evt1/values`
- Harness: `internal/concept/values_records_test.go`
- Total: 32
- `PASS`: 30
- `EXPECTED-DIVERGENCE`: 2
- `EVT1-GAP`: 0
- `POC3-LEGACY`: 0
- `SPEC-AMBIGUITY`: 0

## Classification decisions

The two expected divergences are whole-value ordinary struct copy and
assignment. Canonical EVT1 makes an ordinary struct structurally copyable when
all fields are copyable. PoC3 Phase 10 instead rejects implicit struct copy
without explicit `Copy` conformance (`CON0154`). This is a deliberate semantic
decision, not an implementation gap.

`record struct`, `const` binding syntax, and `with` have no exact PoC3 syntax
counterpart. Their positive and negative cases are classified as EVT1-new
`PASS` evidence. The harness does not manufacture divergences from absent
legacy syntax.

## Evidence map

| Area | R2 evidence | PoC3/reference provenance | Classification |
|---|---|---|---|
| mutable struct field read/write | `struct_mutable_field.concept`, field mismatch rejection | Phase 7 field access/assignment fixtures | `PASS` |
| ordinary struct copy/assignment | `struct_value_copy.concept`, `struct_value_assignment.concept` | Phase 10 implicit-copy rejections | `EXPECTED-DIVERGENCE` |
| const places | local reassignment, projected field, copy-to-mutable, const parameter read/write | R2 decision; older qualifier/place pressure | `PASS` |
| records | construction, read, copy, binding reassignment, field-write rejection | EVT1-new | `PASS` |
| record update | valid Core/Vulkan update, source preservation, exactly-once base lowering, unknown/duplicate/type/noncopyable rejection | EVT1-new with Phase 7/10 diagnostic analogues | `PASS` |
| record equality | bounded compile-time structural equality | existing Go structural equality | `PASS` |
| immovable final storage | direct construction and field mutation | Concept Vulkan M1B-A | `PASS` |
| immovable restrictions | copy, whole assignment, parameter, return, ordinary/record embed, enum payload | Concept Vulkan M1B-A plus PoC3 non-copyable pressure | `PASS` |
| typed `let` | exact const alias and missing-type rejection | R2 optional decision | `PASS` |
| profile boundary | Vulkan record update uses the same core AST/validator/lowerer | R1 profile ownership law | `PASS` |

The native C11 harness executes mutable field assignment, record `with`
update while observing the original value, and immovable final-storage field
mutation. MIR asserts record metadata plus `record_with`; a generated-C oracle
proves a call used as the `with` base appears exactly once at the use site.

## Explicit boundary

R2 does not claim full ownership reconciliation. Move invalidation, drop,
borrow checking, aliases/noalias, allocation, runtime arrays and collections,
Option/Result, C ABI, interfaces/dyn, panic/assert, testing, machine/automata,
decide/yield, and multi-module compilation remain outside this slice.

## R3 handoff (completed)

R3 was implemented as a bounded value-transfer reconciliation: explicit `move` for
non-copyable movable locals, deterministic moved-state diagnostics across
straight-line and simple branch flow, call/return transfer, and the minimum
drop scheduling needed to make those paths executable. It should reuse the R2
place and copyability classifications, keep immovable values in final storage,
and excluded allocators, generalized lifetime inference, runtime collections,
interfaces, and C ABI work. See `EVT1-R3-CONFORMANCE.md` for executable counts
and the recommended R4 boundary.
