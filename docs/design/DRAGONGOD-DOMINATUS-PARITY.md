# DragonGod and Dominatus parity

Dominatus is a C# application kernel and the semantic acceptance oracle for R7b.
DragonGod is its native Concept expression: an agentic compute substrate, not
merely a traditional operating-system kernel. This is behavioral parity, not API
or source compatibility.

## Consumer inventory and parity matrix

| Capability | Historical provider | Historical API | Dominatus use | R7a equivalent | Status | Action |
|---|---|---|---|---|---|---|
| machine execution | PoC `Kernel.Mind`/`Kernel.Automata`; C# `Dominatus.Core.Hfsm` | signal/frame stack; `HfsmInstance.Tick` and `StateReturn` | advance HFSM state, nest/resume, wait, terminate | machine mode only | Needs new canonical primitive | add `Mind.Step`, signals, frames, bounded batch |
| memory/change detection | PoC `Kernel.Memory`; C# Blackboard | key read/write/fallback/revision; `BbKey<T>`, `DirtyKeys`, `Revision` | state authority and dependency invalidation | boot/owned memory only | Needs rewrite | add bounded `MemoryState` and dirty keys |
| events | PoC `Kernel.Events`; C# `AiEventBus` | push/read/tail cursor; publish/read cursor | ordered semantic fact delivery | none | Needs new canonical primitive | add caller-owned `EventBus` and cursor |
| actuation | PoC `Kernel.Actuation`; C# `ActuatorHost` | allow/deny/complete/fail; dispatch/tick/deferred completion | external-work obligation lifecycle | none | Needs new canonical primitive | add receipt/status host; leave meaning to app |
| trace | PoC `Kernel.Trace`; C# `IAiTraceSink` | record steps/signals/transitions; typed callbacks | deterministic observation and diagnostics | none | Needs rewrite | add bounded typed trace recorder |
| replay/checkpoint | PoC Replay/Persistence; C# Persistence | apply-all semantic inputs; `ReplayDriver`; `DominatusCheckpointBuilder` | deterministic restore and external-input replay | none | Needs rewrite | add explicit replay log/cursor and checkpoint copy |
| time/interrupts | PoC clock; C# `AiClock` | tick/deadline/wakeup | timeout gating and hosted advancement | R7a Time/Interrupts | Already covered partly | extend unit-typed clock; retain interrupt boundary |
| identity/context | PoC IDs/Mind context; C# `AiAgent`/`AiWorld` | stable IDs and explicit service aggregation | correlate agents, work, traces, replay | processor/kernel context | Needs rewrite | add value IDs and `AgentContext` |
| decision/graph | PoC Decision/Graph; C# policy/HFSM definitions | scoring/hysteresis/edges | application behavior selection | language machines/`decide`/`infer` | Can delete from kernel | keep policy and graph meaning application-owned |
| parallel runner/persistence format | C# `ParallelAiWorldRunner`/persistence envelopes | host parallel tick; serialized snapshots | deployment policy and durable saves | none | Can defer | R7c scheduler; later application/tooling serialization |

| Historical capability | Dominatus usage | Canonical replacement | Parity result | Notes |
|---|---|---|---|---|
| `HfsmInstance`, state stack, `StateReturn` | Tick an agent, transition, push/pop, wait, complete/fail | `DragonGod.Machine.Execution.Mind`, `MindSignal`, `Step`, `ExecuteBatch` | Rewritten | R5 one-step semantics; no coroutine or scheduler |
| `AiEventBus`, `EventCursor` | Publish ordered facts and read from a consumer cursor | `DragonGod.Events.Core.EventBus`, `Dispatch`, `ReadNext`, `TailCursor` | Rewritten | Bounded caller-owned array; no global bus |
| `Blackboard`, `BbKey`, revision/dirty keys | Read/write agent state and scan only changed dependencies | `DragonGod.Memory.State.MemoryState`, `MemoryKey` | Rewritten | Equal writes do not increment revision or dirty a key |
| `ActuatorHost`, pending commands | Dispatch accepted work and later complete/fail it | `DragonGod.Actuation.Core.ActuatorHost` | Rewritten | Stable `ActuationId`; pending is an explicit obligation |
| `IAiTraceSink` | Observe steps, transitions, events, actuation, terminal results | `DragonGod.Trace.Core.TraceRecorder`, `TraceEntry`, `TraceKind` | Rewritten | Typed records, not string authority |
| `ReplayDriver` and replay inputs | Reapply deterministic semantic inputs | `DragonGod.Replay.Core.ReplayLog`, `ReplayDriver` | Rewritten | Explicit cursor, time, event, memory, actuation, seed inputs |
| `DominatusCheckpointBuilder` | Copy executable agent state for restore | `DragonGod.Replay.Checkpoint.KernelCheckpoint`, `CheckpointSlot` | Rewritten | Explicit in-memory copy; no reflection/serialization framework |
| `AiClock`/timeouts | Gate behavior on monotonic time | `DragonGod.Time.Core.Clock`, `Deadline`, `Advance`, `Reached` | Rewritten | Quantity `s`, not raw milliseconds |
| Agent/world identity | Stable semantic identity across execution and replay | `AgentId`, `MachineId`, `BatchId`, `StateId`, `ActuationId` | Rewritten | Value identity; never pointer identity |
| Seeded behavior | Reproduce choices from a known seed | `AgentContext.rngSeed` plus replay input | Mechanism restored | Random policy/algorithm remains application-owned |
| Decision policies and scoring | Choose application behavior | ordinary Concept `match`, `transition decide`, `infer` | Moved | Language/application semantics, not a kernel subsystem |
| HFSM graph definitions | Define application states and edges | Concept machines plus Dominatus/application data | Moved | No generic graph framework in DragonGod |
| Concurrent/parallel runners | Optional host execution policy | deterministic `DragonGod.Scheduling` single-worker policy | Partial | logical context scheduling restored; host parallelism awaits synchronization |
| Persistence envelopes | Save external application state | none in R7b | Deferred | Explicit checkpoint exists; serialization does not |
| Vulkan presentation/device work | Consumer rendering | existing Vulkan consumers/conformance | Unchanged | Never kernel policy |

## Legacy DragonGod inventory

The archived PoC supplied IDs/reasons/clock, fixed-slot Memory, Automata/Mind,
Decision, Graph, Actuation, Events, Trace, Replay, and checkpoint experiments.
Its useful mechanisms map to the table above. Fixed four-slot/depth ladders,
saved-PC workarounds, private lifetime/allocator machinery, scheduler-adjacent
policy, and Vulkan coupling are deleted rather than preserved.

## Responsibility moves

| Legacy responsibility | Current owner |
|---|---|
| allocator and owned allocation | `Standard.Memory` |
| stable execution/event/actuation mechanisms | `DragonGod` |
| local state transitions | Concept machine and transition semantics |
| decision scores and application meaning | Dominatus/application code using `decide` or `infer` |
| application graph definitions | Dominatus/application code |
| Vulkan lifecycle and presentation | Vulkan consumer |
| serialization/persistence format | future application/tooling work |
| ready queues, fairness, bounded quanta | R7c `DragonGod.Scheduling` |

## R7c scheduler parity

The direct R7b `Mind.ExecuteBatch` path and an R7c `Schedulable` adapter over the
same `Mind` execute identical signal work and finish with the same authored
state, terminal status, and step count. This preserves `Mind` as computation and
adds scheduling only as policy over when it receives a bounded step quantum.

R7c also restores multiple logical execution contexts, deterministic FIFO
fairness, explicit wait/event and sleep/time readiness, and failure isolation.
It does not restore the C# host parallel runner: Concept/DragonGod does not yet
have a general synchronization substrate sufficient for safe shared scheduler
mutation.

## Authority law

DragonGod owns bounded execution context, stable identities, deterministic event
transport, actuation obligations, monotonic time, trace capture, and replay/
checkpoint mechanisms. Applications own what events, memory keys, states,
decisions, and actuations mean. Capacities are configuration, not semantic
constants: the same generic toolkit is instantiated at the required bound.
