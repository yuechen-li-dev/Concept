# EVT2x2 conformance

Compiler: `concept-evt1-stage0-go`. Baseline:
`a62bee94cb960cb27fd5538dc16428913e67f7f0`.

`finite.concept`, `yield_resume.concept`, and `multi_yield.concept` lower from
validated machine MIR to two generated functions each: caller-owned frame
Init and Step. `concept lir` prints deterministic field layout, state IDs,
state-labeled CFG, transitions, yield, completion, and invalid-state trap.
The LIR verifier runs before output. A 100-run test pins byte-identical LIR
text for each fixture. The real C11 backend executes equivalent fixtures and
pins transition step boundaries, yield re-entry, completion stability, and
two independent instances.
The existing semantic artifact payload round-trips a machine body and field
initializer and the decoded module lowers to generated Init/Step LIR.

Malformed LIR tests reject a wrong frame parameter, field geometry/offset,
duplicate or missing state identity, lost dispatch edge, malformed invalid
trap, missing saved state on yield, invalid saved state, missing completion
status, and missing field initialization. The existing push/resume fixture
stops at `EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP`; an unsupported scalar field
stops at `EVT2_UNSUPPORTED_MACHINE_FIELD_TYPE`. No machine can silently
disappear from LIR.

MachineIR and AMD64 machine execution are intentionally deferred to EVT2x3.
The CMIRAMD1 bridge, Concept allocator, frame finalizer, and encoder are
unchanged. No native machine NoAllocation proof is claimed. `concept explain`
on `finite.concept` reports no `Assert.Concept` assertion at the requested
source position; generated Step is an internal LIR function and is not a
source-level proof subject yet.
