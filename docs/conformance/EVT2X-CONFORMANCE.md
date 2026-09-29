# EVT2x conformance

Status: **meaningful progression**, not native automata conformance. Compiler
ID: `concept-evt1-stage0-go`. Baseline:
`35b539b88ebe081e3814c2e02a574821f569384c`.

`TestEVT2xMachineSemanticInputSurvivesMIR` proves that a real validated
push/resume fixture retains the parent's persistent-field initializer and
typed push and resume bodies through MIR construction. The checked MIR JSON
omits this in-memory-only lowering input. `GenerateLIR` and the `concept lir`
CLI now report `EVT2_UNSUPPORTED_AUTOMATA_LOWERING Worker` for that fixture,
instead of allowing an automata-only module to appear as empty native LIR.

The existing C11 machine oracle continues to qualify push/pop, nested frames,
yield/re-entry, persistence, completion, overflow, and independent instances.
No EVT2x machine fixture executes emitted AMD64 bytes yet. No state operation,
frame access, bridge change, or native step result is claimed as implemented.
The next isolated blocker is executable LIR lowering of
`MIRState.SemanticBody` into caller-owned frame storage and dispatch.

The existing EVT2d seven-function native path and `CMIRAMD1` schema remain
unchanged. These facts do not constitute native machine execution.
