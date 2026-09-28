# EVT2d convergence ledger

Baseline: `f5137ee5bab5db7e89da1bc5740eeca7c1e1d594`, compiler ID `concept-evt1-stage0-go`. Work is isolated on `codex/evt2d-concept-amd64` in a managed worktree. The main checkout was not edited.

The real Stage-0 MachineIR for all seven `language/evt2/valid/core.concept` functions is serialized as `CMIRAMD1` and consumed by `Standard.Backend.AMD64`. The allocator, frame finalizer, and byte encoder are authored in Concept. The existing C11 backend compiles them; `concept amd64 <file>` runs that bootstrap executable and displays raw bytes. There is no Go allocator or encoder and no assembler.

## Qualified behavior

| Gate | Result |
| --- | --- |
| Seven-function bridge round trip and 100-run byte identity | Pass |
| Concept backend strict C11 compilation, `check`, `lint`, `format --check` | Pass |
| `NoAllocation(EmitFunction)` proof through allocator, frame, writer, encoder | Proven by `concept explain` |
| Direct Concept encoder REX/ModRM/SIB exact-byte matrix | Pass |
| Direct allocator exhaustion and 100-run assignment/frame equality | Pass |
| Seven native functions through Windows RW→RX executable memory, 100-run emitted-byte and call equality with C oracle | Pass |
| Add and Max exact byte snapshots; checked and bounds failure branch shape | Pass |
| Value-level `decide` valid/invalid corpus, native C11, tie order, no-enabled panic | Pass |
| Focused EVT2 tests and semantic corpus manifest | Pass |
| `go vet ./...`; root and legacy `zig build test` | Pass; pass / pass |
| Standard Normal / Verify | 35 / 35 pass, 3 benchmarks each |
| DragonGod Normal / Verify | 23 / 23 pass, 1 benchmark each |
| Golden Normal / Verify | 130 / 130 pass, 2 benchmarks each |

The requested exact baseline and final `go test ./... -count=1` runs both failed in the same pre-existing Windows/checkout categories: five `m.lib` link tests, proof text golden drift, 18 stale EVT1 checked outputs, and missing TinyXML2 upstream content in the worktree. This matches the EVT2c baseline ledger and the later qualification fix in `a6aa92d`; none of the EVT2d or value-decision tests failed. The final run had no native-thread flake. These failures are outside the MachineIR/Concept backend path and are not hidden by changing golden counts or weakening tests.

## Value decision follow-up

The initial operand decoder's integer-tag ladders became exhaustive `match` expressions. A value-level `decide` was absent from the parser: only machine `transition decide` existed. EVT2d adds an ordinary expression for a typed payload-free enum destination, with guarded `int` or `float` scores, source-order evaluation, first-maximum tie choice, and explicit NaN/no-enabled failure. Its native C11 and corpus tests establish a usable one-shot utility choice outside automata. `ValidateOperand` remains an exhaustive `match`: its checks are categorical validity conditions, and scoring them would require an extra result enum and arbitrary utility values.

## Bounded native subset

Allocation uses global CFG liveness and deterministic linear intervals over at most 16 blocks and 32 virtual registers. The pool is Win64 caller-saved R10, R11, RAX, R8, R9. Exhaustion is explicit; spills are not implemented. Leaf frames omit stack changes; source locals are packed and the Win64 stack alignment requirement is maintained. Calls, outgoing shadow space, callee saves, floating point/SIMD, unwind metadata, and object files are deferred. Native failure edges use UD2; tests inspect those branches without executing traps in the main test process. EVT2e was not started.
