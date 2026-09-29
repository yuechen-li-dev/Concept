# EVT1 R7c conformance: Schedule the Dragon

## Baseline

- Baseline and R7b commit: `dbb00a0855a66cdfb6a92bdb1fa10ee5f29837a7`
  (`Restore DragonGod Dominatus parity`).
- R7a commit: `f291c74bd23b1ae410b539a6725f77194299617f`.
- Compiler: `concept-evt1-stage0-go`.
- Package graph: `DragonGod -> Standard`; Standard has no dependencies.
- Canonical modules added: `DragonGod.Scheduling.Core`, `.Queue`, and `.Fifo`.
- Native path: Concept -> MIR -> Planner -> strict C11 -> host compiler -> fact
  executable.

The starting worktree was clean. Go, Go vet, root Zig, legacy PoC3 Zig,
Standard package, and DragonGod package baselines passed.

## Result

R7c is a deterministic, fixed-capacity, single-worker scheduler expressed
entirely as DragonGod library code.

| Success-floor requirement | Evidence | Result |
|---|---|---|
| context and stable ID | generic context/table; monotonic non-reused `ContextId` | Pass |
| no duplicated authority | scheduler table owns machines; FIFO stores IDs only | Pass |
| bounded FIFO/round-robin | ring wrap, duplicate, empty, and capacity facts | Pass |
| bounded configurable quantum | native facts for 1, 2, and 4 steps | Pass |
| lifecycle | Ready/Running/Waiting/Sleeping/Completed/Failed native facts | Pass |
| event, interrupt, and time wake | explicit event-kind notification and deterministic clock facts | Pass |
| multiple contexts and fairness | A=5, B=3, C=4 trace is `1,2,3,1,2,3,1,2,3,1,3,1` | Pass |
| failure isolation | failed reason retained while healthy peer completes | Pass |
| Dominatus-shaped execution | R7b `Mind.ExecuteBatch` and scheduled `Mind` adapter reach identical state/status/step count | Pass |
| structured trace | R7b recorder extended with typed scheduler events | Pass |
| Standard.Memory | dynamically provisioned policy value uses `MonotonicAllocation`; fixed core remains inline | Pass |
| bounded/no hidden heap | artifact-only generated C rejects heap/runtime scheduler symbols | Pass |
| strict C11 | every package fact and artifact-only consumer compiles and runs natively | Pass |
| 100-run determinism | one native fact hashes 100 complete structured traces; existing package test builds graph 100 times | Pass |
| compiler independence | no scheduler MIR/opcode/Planner/type-name branch added | Pass |

The package suite contains 18 passing facts and one informational scheduler
benchmark. The scheduler implementation is 434 Concept source lines across its
three modules.

The artifact-only representative emits 63,340 bytes of C. The informational
benchmark executes twelve decisions; five native iterations measured a 6.5822
ms mean (6.0864 ms minimum, 6.7299 ms maximum) including fact harness overhead.
No performance threshold is asserted.

## Boundaries

R7c introduces no compiler changes. The ordinary generic `Schedulable`
requirement allows application adapters to resume their existing machine state;
the scheduler does not duplicate `Step` or yield semantics. `Mind` batch parity
proves the R7b loop is reused rather than forked into another execution engine.

The fixed scheduler has bounded inline context, ready, and trace storage. The
artifact-only consumer imports compiled Standard/DragonGod modules without
source roots, runs strict C11, and scans out `malloc`, `calloc`, `realloc`,
`free`, scheduler runtime singleton names, and scheduler-specific MIR/Planner
names. A compiler semantic fact proves the concrete fixed-capacity
`ScheduleOne` wrapper is `NoAllocation`; allocation is explicit only when a
caller chooses a `Standard.Memory` owner during setup.

Host multi-thread execution is not implemented. The repository has no canonical
atomic/synchronization surface for safe shared scheduler mutation, and native
thread timing would not preserve the single-worker replay claim. This is an
intentional R7c boundary, not simulated concurrency.

Collector interaction is documented only: future root ownership and pause
coordination may align with execution contexts and machine step boundaries.
No R7d collector work begins here.

## Final validation

- `go test ./...`, `go vet ./...`, root `zig build test`, and legacy PoC3
  `zig build test`: pass.
- Oct `make Test --file Make.oct`: pass after syncing the declared local
  `Make@0.1.0` dependency; Standard has 2 native facts and DragonGod has 18
  native facts plus 1 benchmark.
- `Make.octest` under Oct auto execution: 7 pass; the 2 ConceptBuild facts run
  interpreted because the current Oct compiled-test worker does not resolve the
  already-synced Make package. The real compiled Make execution and every
  Concept/strict-C11 scheduler lane pass.
- Full active EVT1 corpus: 712 `.concept` fixtures, 749 total files.
- `git diff --check`: pass.
