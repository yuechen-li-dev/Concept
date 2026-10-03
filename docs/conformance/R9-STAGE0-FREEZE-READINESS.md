# Stage-0 freeze readiness

The MachineIR bridge duplication item is **CLOSED** by R9a2. R9a closure is
complete and ready for the separate R9b research phase. Stage-0 has not been
frozen automatically. R9b has reached a partial progression checkpoint;
frontend self-hosting has not begun.

| Closure criterion | Status |
| --- | --- |
| Closed generic semantic identity survives artifacts structurally | CLOSED in R9a; explicit ownership retained |
| Validation contexts and modifier/attribute admission | CLOSED within the qualified R9a boundary |
| Numeric/indexing/generic repr(C) closure | CLOSED in R9a; field defaults remain explicitly unadmitted |
| Runtime static control | CLOSED in the bounded evaluator/C11 path; comptime auto primary, var compatibility retained |
| Ordinary declared predicates and root project policy | CLOSED in R9a with declared proof provenance |
| CV4138 innate migration and library scale evidence | CLOSED in R9a; limits/mutex retained |
| Single-source MachineIR bridge | CLOSED in R9a2: checked Concept schema, generated Go/Concept codecs, version/hash, live shadow parity, switch and manual retirement |
| Caller-supplied backend encoding workspace | CLOSED for the R9a MVP; other backend phases remain bounded |
| Regression qualification | PASS: full Go, vet, race, corpus, native EVT2/EVT2x, Standard/DragonGod/Golden/GPU-free Vulkan Normal and Verify |
| Frozen legacy Zig | Unchanged; baseline tests passed, further suites run only when its compiler/build/test paths change per updated user policy |

The live bridge is CMIRAMD2 / numeric version 2, with semantic SHA-256
`99d13195dd9968f6d9807075e6ac8890ec711d569ad2dcc35ff0c3ca56709593`.
Both codecs derive from Standard.Backend.BridgeSchema; no independent manual
field-order authority or legacy fallback remains. Complete canonical roundtrip,
artifact-only identity, malformed-input Result behavior and 100-run determinism
are qualified, including byte-identical native outputs.

This readiness statement preserves the existing research boundaries: no full
frontend self-hosting, generic Verdict or fact-granting trust claim; no broad
Vulkan/hardware release claim; no unbounded AMD64 backend. These deferred scopes
are documented rather than treated as silently accepted features.
See R9A2-CONVERGENCE and R9A2-CONFORMANCE for commits and evidence; R9A documents
retain the historical milestone result with the bridge closeout noted.

## R9b impact: partial progression

| Addition or research item | Surface / freeze impact |
| --- | --- |
| StaticExtent<N> | Changes the compiler semantic query/fact surface. Future frontend must preserve exact one-dimensional fixed extent, Unknown for runtime views/capacity, and layout provenance. Planner use remains a shadow; no new native optimization. |
| Closed generic comptime record values | Extends existing closed generic evaluation; no new syntax. Frontend/evaluator must preserve structured identity and typed contextual payloads. Resource-bearing/open types remain rejected. |
| Typed Verdict predicate protocol | Unresolved PREDICATE_REQUIREMENT_INVALID boundary; blocks a claim that R9b evidence semantics are frozen. Closed template function CV4201 also remains. |
| PlainData / FiniteDomain | Deferred admission, no new surface yet. Library-level rules may remain possible post-freeze only when their observations/trust are stable. |
| Relocatable / StableAddress / ClosedWorld | Scoped research, unadmitted. No intrinsic/contextual authority silently installed. |
| Contiguous / Disjoint | Existing canonical vocabulary retained, no competing names or new optimization permissions. |

R9B-CONVERGENCE and R9B-RESEARCH-LEDGER distinguish qualified changes from open
research. R9b is not complete and Stage-0 is not frozen.
