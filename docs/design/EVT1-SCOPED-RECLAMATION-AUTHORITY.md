# EVT1 scoped reclamation authority

R7f2 follow-up: [destructive resource effects](EVT1-DESTRUCTIVE-RESOURCE-EFFECTS.md) makes `ref`-based invalidation explicit; [suspension scoped authority](EVT1-SUSPENSION-SCOPED-AUTHORITY.md) records conservative await/yield exclusion.

The supported exclusion pattern is an `owned` resource plus a lexical borrow. A reclaim operation consumes `owned Store`; `Reclaim(move store)` is rejected while a reference or reference-struct lease tied to `store` remains live. After the lease scope ends, the same move is legal. Another owner's reclaim is independent.

The validator records borrowed access paths on local reference and reference-struct values. Direct references, reference-returning calls and template calls, reference-struct construction, assignment, and branch joins propagate these paths. Move validation uses the existing `STORAGE_MOVE_WITH_LIVE_REFERENCE` diagnostic. There is no runtime borrow table or new ownership type.

The caller must express destructive authority by consuming `owned`. A `ref Store` parameter carries no destructive-effect contract, so a function that releases through a reference can evade this exclusion. This is an explicit limit. Aggregate provenance and suspension of a lease across async/yield also need further proof before a collector can rely on them. The tests establish local and artifact-only lexical cases, not a general lifetime theorem.
