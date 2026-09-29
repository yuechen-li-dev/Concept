# EVT2x2 convergence ledger

Status: **success for the bounded EVT2x2 LIR milestone**. MachineIR and
native execution are the deliberate EVT2x3 boundary.

Started from clean `a62bee94cb960cb27fd5538dc16428913e67f7f0`, compiler
`concept-evt1-stage0-go`, in the existing `codex/evt2x-native-automata`
managed worktree. The main checkout remains untouched.

The prior `EVT2_UNSUPPORTED_AUTOMATA_LOWERING` boundary is removed for one
canonical machine with scalar persistent fields and finite-state/yield
control. Machine Init and Step now use verified LIR and explicit frame
metadata. The existing push fixture reaches the narrower
`EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP Worker.Parent.Start` boundary. Signal
automata, multiple-machine execution, non-scalar persistent state, machine
foreach, and non-neutral completion outcomes also diagnose explicitly.
MachineIR lowering stops at `EVT2_MACHINEIR_MACHINE_STEP_DEFERRED`.

The source language's yield re-enters the same state from its beginning.
The brief's illustrative after-yield continuation would contradict the
qualified EVT1 C backend, so EVT2x2 stores the source state ID and returns
`Yielded`. No generated continuation substate is needed or emitted.

Focused machine/EVT2 and semantic-corpus tests, existing EVT2d native tests,
`go vet ./...`, and root and legacy Zig suites passed. Standard Normal and
Verify each passed 35 tests; DragonGod each passed 23; Golden each passed
130. The full `go test ./... -count=1` failed only in the same documented
baseline categories: five Windows Clang links cannot open `m.lib`, proof text
golden drift, missing TinyXML2 upstream source in this worktree, and 18 stale
EVT1 checked outputs. No EVT2x2 or EVT2d native fixture failed.

EVT2x3 has not started. No CMIRAMD1, allocator, encoder, native executable
memory, pushdown stack, scheduler, or async implementation changed.
