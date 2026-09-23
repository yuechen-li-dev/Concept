# EVT1 R7e conformance

Result: **Success** for the Win32 native-worker validation scope.

Baseline: `8dd0f0f7f1ce85bbf4693c91dad4f4662a81ce99`.
Compiler: `concept-evt1-stage0-go`. R7c scheduler: `1faeb29`.
R7d6 proof closure: `70bbf80`, `8dd0f0f`.

## Worker and authority model

`DragonGod.Scheduling.Parallel` adds `ParallelScheduler` over the existing
`FifoScheduler`. Its gate protects the ready queue, context lifecycle, wake
bookkeeping, registration, explicit clock, and bounded structured worker trace.
Workers release the gate around `StepScheduled`; the context table remains
the only machine owner. Registration reserves one queue slot per context.
`WorkerId` is semantic; a host thread handle is not a context ID.

The initial host adapter is the Win32 create/run/join harness in
`internal/concept/parallel_scheduler_test.go`. Generated Concept code is
compiled with `-std=c11 -Wall -Wextra`. The same scheduler source contains
no OS call, private heap, singleton, TLS semantic authority, task registry,
or runtime proof registry. POSIX and bare-metal adapters are deferred.

Ready -> Running is guarded. Only that worker touches its machine until the
quantum ends. Release publishes Ready, Waiting, Sleeping, Completed, or Failed
under the gate. Explicit `Wake` records a wake during Running; event
notification records one pending kind, with explicit capacity error on a
different concurrent kind. Deadline release checks the guarded monotonic
clock again. Completed and Failed cannot reenter the queue. Workers stop on
no Ready and no Running work; a later wake can start another run.

## Native evidence

The native artifact-only consumer imports compiled Standard and DragonGod
modules, without dependency source reparse. It runs 20 trials each at 1, 2,
and 4 workers and semantic quanta 1, 2, and 4 with four 1,000-step contexts:
4,000 final steps, `4,000 / quantum` decisions, at least two participating
workers when worker count exceeds one, and no per-context overlapping Step.
Twenty four-worker failure trials complete
three healthy contexts and retain one typed failure. Twenty trials each race
an explicit wake, event notification, or clock advance against a Running
context about to wait or sleep; each resumes exactly once and completes.

An eight-agent, four-worker fixture carries eight separate `Mind` values and
explicit references to one shared fixture. Its final oracle checks all eight
Mind successes, Blackboard write/read publication, EventBus payload delivery,
preoccupied disjoint and shared-slot Blackboard writes with dirty/revision state, and one
Actuation claim/commit. The same fixture passes at 1, 2, and 4 workers across
20 trials each. The native multi-worker trace is an interleaving and need not
be byte identical. The original one-worker FIFO trace remains deterministic.

`Assert.Concept<NoAllocation>` passes for both bounded worker paths and the
explicit, event, and deadline wake helpers. The
native generated-C audit rejects `malloc`, `calloc`, `realloc`, a hidden
current scheduler, and a scheduler plan. Host thread creation is outside the
Concept steady-state effect boundary. EventBus SPSC producer-guard elision
remains Unknown; the retained guard is safe. No call-site context attribution
or library-specific compiler rule was added.

The native informational probe measured these throughputs on this host,
including thread creation and join across 20 runs per cell. There is no
pass/fail threshold; small Steps are dominated by guard and queue contention.

| Workers | Quantum 1 | Quantum 2 | Quantum 4 |
|---:|---:|---:|---:|
| 1 | 3.88M steps/s | 6.63M steps/s | 10.22M steps/s |
| 2 | 2.51M steps/s | 4.92M steps/s | 9.12M steps/s |
| 4 | 1.69M steps/s | 3.25M steps/s | 6.31M steps/s |

## Validation and limits

The DragonGod package reports 21 passing facts and one benchmark. Standard
reports three passing facts. Native R7e strict-C11 execution, `go vet ./...`,
both Zig `zig build test` lanes, and `oct make BurnIn` pass. The Make Test lane
builds/tests Standard and DragonGod. The full Go suite includes the EVT1
corpus, R7d6 proof regressions, and 100-run package artifact determinism.

The scheduler is driven by an explicit host adapter. Host waiting currently
uses bounded spin while another worker finishes; parking, POSIX threads, and
bare-metal SMP startup are deferred. Its trace ring reports overwrites rather
than retaining unlimited history. A future collector will need per-context
root ownership, shared Blackboard roots, worker rendezvous, and Step-boundary
coordination. R7e adds no collector.

A preexisting ignored `.oct` package cache was present at baseline. Its
recursive removal was rejected by the command policy, so the requested
cache-free starting state could not be established. The Oct validation lanes
passed with that cache present.
