# DragonGod scheduling direction

The scheduler is not a language runtime primitive.

Concept supplies resumable machine semantics. DragonGod supplies execution
mechanisms. Scheduling policy is ordinary library code.

## Authority

An `ExecutionContext` is logical resumable computation. A `Worker` is an
execution resource. A scheduler maps ready contexts onto workers. These are
deliberately distinct: contexts do not imply host threads, and host threads do
not own application machine meaning.

The R7c scheduler owns scheduling authority, its context table, ready IDs,
wait/sleep bookkeeping, clock, and trace. Each machine value appears exactly
once in the table. The FIFO queue contains `ContextId` values only. Applications
own machine transitions, event meaning, memory, decisions, and actuation policy.
The compiler owns only ordinary machine lowering and `Step` semantics.

The initial policy is deterministic FIFO/round-robin with caller-selected
positive semantic step quantum. Continuously ready contexts rotate without
starvation. Waiting requires an explicit matching event; sleeping requires an
explicit monotonic deadline; interrupts report readiness and never schedule
directly. Failures are isolated unless a caller elects a separate fail-fast
policy.

## Substrate decision

| Primitive | Current provider | Scheduler use | Sufficient? | R7c action |
|---|---|---|---|---|
| explicit machine step/yield | Concept machine semantics and `Schedulable` adapter | bounded quantum | yes | reuse |
| stable identity | `DragonGod.Core.Types` | value `ContextId` | yes | add one value ID |
| bounded storage | fixed arrays and `Standard.Memory` | context, queue, trace policy state | yes | reuse |
| deterministic batch | `Mind.ExecuteBatch` | direct parity oracle | yes | reuse through adapter |
| events and dirty memory | R7b Events/Memory | explicit wake and application state | yes | consume event kind only |
| monotonic time | `DragonGod.Time.Core` | sleep deadlines | yes | reuse |
| interrupts | `DragonGod.Interrupts.Core` | readiness fact | yes | adapt to notification |
| structured trace/replay | R7b `TraceRecorder` and explicit inputs | schedule diagnostics and deterministic derivation | yes | extend trace kinds |
| actuation | R7b `ActuatorHost` | machine-owned obligations | yes | no scheduler policy |
| atomics/synchronization | none canonical | shared multi-worker mutation | no | defer host threads |

R7d has since supplied the first canonical strict-C11 integer atomic substrate.
The row remains insufficient for multi-worker scheduling because shared-access
authority proofs and worker-safe scheduler notification have not yet landed.

## Collector interaction

R7d may need per-context root ownership, allocation-state visibility, and
pause/coordination at explicit machine step boundaries. R7c adds no GC roots,
barriers, heap scanning, conventional safepoints, or collector hooks. Step
boundaries are only an observation for the next milestone.

Deferred work remains generic-automata context authority, priority/deadline
policies, work stealing, NUMA, affinity, realtime proofs, process/userspace ABI,
bare-metal SMP startup, VM/pages, reflection, serialization, native LIR, and the
collector itself.
