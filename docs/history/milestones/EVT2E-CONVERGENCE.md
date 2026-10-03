# EVT2e call handoff: meaningful progression

Baseline: `a85747a2ff061848e06ecbc10e68d88710cf278c` (`concept-evt1-stage0-go`). The existing bridge remains `CMIRAMD2`, version 2, schema hash `99d13195dd9968f6d9807075e6ac8890ec711d569ad2dcc35ff0c3ca56709593`. Its wire contract has not been changed.

The first missing link, a first-class LIR call, is present for checked direct functions with bodies, `bool` and fixed-width integer arguments, scalar or void results, and `win64` convention identity. The call preserves the validator-selected declaration signature, ordered argument values and types, and source span. The LIR verifier checks the callee and argument contract before the machine backend sees it. Stage-0 owns only this checked semantic transport; no new Go ABI placement, allocation, frame, or encoder algorithm was added.

`internal/concept/testdata/evt2e_calls.concept` exercises zero-argument, bool, void, scalar-result, and five-argument calls, including a local value live across a call. `concept lir` prints these calls. `concept machineir` currently fails explicitly with `MIR_UNSUPPORTED_LIR_OP call`. That is the next unresolved boundary: extend the single-source bridge schema with a versioned per-call MachineIR contract, then implement Win64 argument placement, clobber-aware allocation, saves, spills, coherent frames, and call encoding in Cathedral's Concept backend. The fixture has no native call execution claim yet.

The frozen Zig compiler path is unchanged.

## Qualification of this handoff

`go test ./... -count=1` passed after the LIR code change (`internal/concept` 381.348s); `go vet ./...`, focused `-race` call tests, the semantic corpus manifest test, and the existing EVT2/EVT2x focused native tests passed. Standard passed 42/42, DragonGod 23/23, and Golden 130/130 in both Normal and Verify modes. The fixture passed `concept check`, `concept format --check`, and `concept lint`. The new call LIR repeated identically in 100 runs. No new native call bytes, Win64 ABI probe, call frame size, spill count, or call performance measurement exists yet.
