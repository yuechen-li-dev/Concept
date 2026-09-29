# EVT1 R6h conformance

R6h preserves compiler-known semantic value facts across ordinary values and semantic module artifacts without changing runtime layouts.

Evidence is owned by:

- `internal/concept/fact_transport.go` for typed subjects, transport, joins, summaries, instantiation, artifact validation, and MIR projection;
- `internal/concept/fact_transport_test.go` for local/copy/move, Result/Option, aggregate/generic/capture, async/machine persistence, region relation, storage/bind, module cache, determinism, zero-cost, and golden proof checks;
- `language/evt1/tooling/semantic-facts/valid` for 22 accepted programs;
- `language/evt1/tooling/semantic-facts/unknown` for six rejected or explicitly Unknown boundary programs;
- `language/evt1/tooling/semantic-facts/*.concept_test` for four native dogfood facts;
- `language/evt1/tooling/proofs/golden/r6h-*.txt` for four checked human proof contracts.

The flagship module path is `parent region -> imported Subregion -> Result extraction -> local/copy -> imported generic Box field -> proof`. Known `[0,8)` and `[8,16)` intervals prove Disjoint; `[0,8)` and `[4,12)` disprove it; runtime offsets remain Unknown. `AddressFromBits` and artifacts missing value summaries remain Unknown through wrappers.

`fact_prometheus_transport.concept`, `fact_dragongod_transport.concept`, and `fact_allocator_readiness.concept` pin the requested consumer compositions. The allocator specimen stops at geometry, typed bind, initialization, and destruction; it does not introduce an allocator.

The full Go tests, vet, root Zig suite, and retired `legacy/poc3-zig` suite are the release gate. Strict-C11 execution is exercised by the four dogfood tests and focused native storage/quantity baselines. Generated-C scans reject runtime semantic metadata and heap introduction.

On the recorded Windows amd64 Ryzen 7 7700X run, the representative imported
Subregion parse, semantic analysis, MIR/Planner, and generation benchmark was
2.705 ms/op, 2,679,325 B/op, and 10,554 allocations/op over 100 iterations.
This is a whole-pipeline sanity measurement, not a claim of isolated speedup;
runtime execution receives no additional work.
