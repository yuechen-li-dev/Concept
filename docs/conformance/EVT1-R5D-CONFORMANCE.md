# EVT1 R5d conformance

Status: implemented and executable

Baseline: `3097b37fa76ef1aa2bcd652b87aa39c0f562ff62`

Compiler: `concept-evt1-stage0-go`

R5d has 26 canonical cases: 18 valid and eight static rejection cases. All are
`PASS`; there are zero `EXPECTED-DIVERGENCE`, `EVT1-GAP`, `POC3-LEGACY`, or
`SPEC-AMBIGUITY` cases. PoC3 Phase 19 is cited as semantic provenance for bare
yield; iterator and foreach are EVT1-new and no parity is fabricated.

Native strict-C11 evidence covers same-state re-entry, transient recreation,
automata and machine persistence, transient owned cleanup, persistent owned
survival, array and row-major ndarray iteration, Span/ReadOnlySpan iteration,
custom protocol dispatch, exactly-once source evaluation, and transient
foreach restart after yield. MIR and Planner tests pin explicit Yield/Foreach
records and reject malformed yield MIR.

Generated C contains no malloc family, scheduler, coroutine frame, saved
program counter, generator runtime, or iterator registry. Complete/Result
source syntax remains partially reconciled; yield itself neither completes a
machine nor writes result storage. Generator yield, ranges/pipelines, and LIR
remain deferred. Recommended R5e is bounded completion/result reconciliation,
without expanding bare yield into continuation semantics.
