# R9c convergence

R9c starts from clean `3cb83bc4e4d9ac96fc86b151380a2c129cc60ca8` with
compiler `concept-evt1-stage0-go`. The result is an explicit Stage-0/Cathedral
authority plan and one production verifier vocabulary switch. The C bootstrap
path remains intact, Stage-0 is not frozen, and EVT2e is not started.

## Scope and result

The checked Concept bridge Condition enum now supplies the production Go
MachineIR verifier's accepted condition set through the generated bridge tag
table. A frozen legacy truth table proves shadow agreement. This retires the
hand-coded Go switch while retaining the generated Go seed projection.
The CLI's Stage-0 backend host now resolves the imported Concept backend
library before C generation. The real `concept amd64` command emitted native
bytes for the finite and parent/child machine specimens after previously
failing CV4401; a command-level regression test pins that bootstrap route.
Current MachineIR/LIR call gaps are documented before EVT2e implementation.
The migration matrix records present/desired owner, readiness, bootstrap
dependency, switch and Go-retention condition for every requested subsystem;
the authority ledger records live states. The frontend strategy is incremental
rule extraction over checked observations. The C backend remains a portable
bootstrap/fallback route.

## Qualification

| Gate | Final result |
| --- | --- |
| `go test ./... -count=1` after CLI repair | PASS; `internal/concept` 377.557s, `cmd/concept` 1.988s; semantic corpus, Vulkan and EVT2/EVT2x native tests included |
| `go vet ./...` | PASS |
| Focused helper/verifier/bridge tests | PASS |
| Focused `-race` helper, generated bridge, native frozen-byte and 100-run tests | PASS, 10.746s |
| CLI `-race` regression | PASS, 3.051s |
| Standard Normal / Verify | 42 / 42 passed |
| DragonGod Normal / Verify | 23 / 23 passed |
| Golden Normal / Verify | 130 / 130 passed |
| Direct `concept amd64` finite and parent/child | PASS, native bytes emitted through Stage-0 C host |
| AMD64 source `format --check`, lint with library roots | PASS |
| Unchanged BridgeSchema `format --check` | Fails at baseline; not reformatted by R9c |
| Frozen Zig suites | SKIPPED under unchanged-path policy |

The first full Go run also passed before the CLI repair (`internal/concept`
386.877s); the final rerun above is the qualifying result. Existing bridge
native frozen oracles emit each function 100 times; generated codecs and
MachineIR have dedicated 100-run determinism checks. No CMIRAMD2 format or
native instruction output changed. `R9C-CONFORMANCE.md` records the contract
and helper proof. The legacy Zig compiler/build/test paths are unchanged, so
both frozen Zig suites are skipped. No object writer, broad frontend migration,
Cathedral self-rebuild or EVT2e call implementation is claimed.
