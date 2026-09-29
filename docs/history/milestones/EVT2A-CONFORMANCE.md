# EVT2a LIR constitution

The target-independent `LIRModule`/`LIRFunction`/`LIRBlock` model, explicit typed virtual values, local slots, explicit terminators, deterministic Latin printer, and mandatory verifier live in `internal/concept/lir.go`.

`TestEVT2VerifierRejectsMalformedLIR` rejects undefined values, duplicate definitions, wrong operand types, missing terminators, wrong returns, and invalid branches. `TestEVT2VerifierRequiresIndexedGuardAndCoherentLayout` rejects an unguarded address and wrong element stride. There are no block arguments in this local-slot design, so block-argument arity is inapplicable.

The verifier is called by `LowerMirToLir` before the result is returned. No MachineIR, ABI, register allocation, or native emission exists here.
