# Concept EVT1 R1 differential conformance

Status: executable R1 evidence

## Baseline snapshot

- Concept HEAD: `7a0a9e0f9eb131708c10e314a719185779417615`
- Active compiler: `concept-evt1-stage0-go`
- Go: `go1.27.0 windows/amd64`
- Go top-level test functions: 43 before R1; 48 after R1
- PoC3 corpus: 1,478 tracked `.concept`, `.conception`, or `.con_test`
  inputs before R1, including 1,296 primary `language/` fixtures
- Reconciliation rows: 19 Merge, 13 Keep PoC3, 8 Keep Go, 5 Deferred,
  3 Redesign, and 2 Profile-only
- Baseline result: clean worktree; Go tests/vet, root Zig reference tests, and
  `legacy/poc3-zig` tests passed

The R1 harness is `internal/concept/conformance_test.go`. It uses paired
semantic evidence rather than source-text equality: each table entry names a
PoC3 fixture or authority row, its canonical EVT1 translation under
`language/evt1/foundation`, the reconciliation status, the expected
classification, and an accepted/rejected or MIR/C semantic oracle. Diagnostic
comparisons use the active `CV` semantic category; they do not require PoC3
`CON` number equality.

```text
PoC3 source fixture or recorded matrix evidence
    -> explicit semantic translation
canonical EVT1 Core source
    -> active Go parser, validator, MIR, and C generator
semantic oracle and divergence classification
```

## Summary

| Area | Cases | PASS | Expected divergence | EVT1 gap | PoC3 legacy | Spec ambiguity |
|---|---:|---:|---:|---:|---:|---:|
| structs | 2 | 2 | 0 | 0 | 0 | 0 |
| payload enums and base match | 6 | 6 | 0 | 0 | 0 | 0 |
| runtime control flow | 1 | 1 | 0 | 0 | 0 | 0 |
| concepts and templates | 3 | 3 | 0 | 0 | 0 | 0 |
| bounded comptime and compile-time arrays | 4 | 3 | 1 | 0 | 0 | 0 |
| profile isolation | 5 | 5 | 0 | 0 | 0 | 0 |
| **Total** | **21** | **20** | **1** | **0** | **0** | **0** |

## Semantic oracles

- Struct evidence checks declaration shape, positional construction, field
  access, value copy followed by independent mutation, and initializer type
  rejection.
- Payload-enum evidence checks nullary, one-payload, and two-payload variants,
  qualified construction, wrong payload rejection, exhaustive coverage,
  duplicate arms, payload binding type, and expression result unification.
- Control-flow evidence compiles an `if` expression, boolean/comparison
  expressions, assignment/value flow, and an ordinary runtime `while`; the C
  oracle confirms the loop remains runtime code.
- Concept/template evidence checks a prerequisite closure, explicit
  `requires Concept<Type>;` proofs, missing-operation rejection, explicit
  concrete calls over struct and enum types, two concrete instances, and reuse
  of one generated identity by duplicate identical calls.
- Comptime/array evidence checks values, a function, `static_assert`, bounded
  `while`, nested fixed arrays, `Len`, indexing, structural equality, runtime
  erasure through MIR facts, runtime-call rejection, and constant bounds
  rejection.
- Profile evidence checks Core denial of a Vulkan runtime type, a Vulkan
  import, `effect`, `actuator`, and an undeclared mechanism call. No
  Prometheus-specific compiler builtin or type exists; the sole Prometheus
  compiler datum is the Vulkan profile's admitted import marker.

## Non-PASS cases

### `unbounded comptime loop` — `EXPECTED-DIVERGENCE`

- Source evidence:
  `language/phase9-compile-time/valid/compile_time_function_while_sum_run.valid.conception`
- EVT1 translation:
  `language/evt1/foundation/invalid/comptime_unbounded_loop.concept`
- Matrix row: `while`, status `Merge`.
- Observed difference: PoC3 accepts its fuel-bounded historical spelling;
  EVT1 rejects the corresponding compile-time loop with `CV4205` unless it
  carries `bounded(limit)`.
- Reason: R0 explicitly retained the Go bounded compile-time law.
- Compiler change required: no.
- Specification decision required: no; section 15 already states this law.

There are no `EVT1-GAP`, `POC3-LEGACY`, or `SPEC-AMBIGUITY` cases in the
bounded executable set. Deferred families remain inventory pressure and were
not converted into artificial R1 failures.

## PoC3 sources used

Seventeen distinct PoC3 fixtures were inspected and retained as provenance:

- `language/phase2-execution/valid/while_with_if.valid.conception`
- `language/phase3-semantics/invalid/hir_check_return_type_mismatch.invalid.conception`
- `language/phase5-sum-types/valid/enum_match_payload_bind_int.valid.conception`
- `language/phase5-sum-types/valid/enum_match_payload_bind_two.valid.conception`
- `language/phase5-sum-types/invalid/enum_constructor_payload_type_mismatch.invalid.conception`
- `language/phase5-sum-types/invalid/enum_match_duplicate_variant.invalid.conception`
- `language/phase7-runtime-structs/valid/field_access_run.valid.conception`
- `language/phase7-runtime-structs/valid/struct_literal_basic.valid.conception`
- `language/phase7-runtime-structs/invalid/struct_literal_type_mismatch.invalid.conception`
- `language/phase8-concepts-templates/valid/template_constrained_equal.valid.conception`
- `language/phase8-concepts-templates/invalid/constrained_missing_impl.invalid.conception`
- `language/phase8-concepts-templates/invalid/impl_missing_requirement.invalid.conception`
- `language/phase9-compile-time/valid/compile_time_function_while_sum_run.valid.conception`
- `language/phase9-compile-time/valid/static_assert_compile_time_while.valid.conception`
- `language/phase9-compile-time/invalid/compile_time_function_runtime_call.invalid.conception`
- `language/phase21-arrays-slices-fixed-buffers/valid/array_literal_nested.valid.conception`
- `language/phase21-arrays-slices-fixed-buffers/invalid/array_index_constant_high.invalid.conception`

## Explicit exclusions

No parity claim is made for runtime arrays, Slice, FixedBuffer, Option, Result,
allocation, full ownership, C ABI, dyn/interfaces, runtime panic/assert,
machines/automata reconciliation, decide, yield, Concept-native testing, or
multi-module compilation. R1 added none of those features.

## Recommended R2

Keep R2 bounded to the next matrix-selected semantic question. A useful next
step is diagnostic-family reconciliation plus a small second conformance slice
for value/place and immovable rules. Do not begin runtime arrays, ownership, or
machine reconciliation without selecting that row explicitly in the matrix.
