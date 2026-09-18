# EVT1 tables and ellipsis conformance

Status: implemented by `concept-evt1-stage0-go`.

The conformance surface is intentionally bounded to fixed columnar tables and
initializer repetition. `internal/concept/table_repetition_test.go` proves:

- `table<N>`, `record table<N>`, and `ref table<N>` identity;
- type and non-type generic substitution;
- column indexing and array-to-Span reuse;
- mechanical `SizeOf`/`AlignOf` and structure-of-arrays C11 layout;
- module artifact transport and artifact-only consumers;
- explicit-count, fill-remainder, multiple-segment, zero-count, generic-count,
  and per-element side-effect semantics;
- overflow, negative, non-comptime, unknown-remainder, and ownership errors;
- compact MIR and byte-identical module artifacts across 100 runs; and
- native strict-C11 execution.

Permanent readable fixtures live in
`language/evt1/tables-initializers`. The Prometheus-shaped specimen uses a
fixed coordinate table for numeric computation without adding Vulkan, SIMD,
or planner optimization. DragonGod dogfoods ellipsis in its agent/replay
fixtures.

Deferred by design: dynamic owning table storage, row proxies, row-wise
transposition syntax, dataframe/database operations, reflection,
serialization, automatic AoS/SoA conversion, and GPU-specific lowering.

