# DragonGod scheduling

DragonGod scheduling is ordinary Concept library policy. The compiler defines
resumable computation and `Step`; DragonGod supplies explicit execution and
resource mechanisms; `FifoScheduler` decides which ready execution context is
stepped next. There is no scheduler opcode, hidden runtime, process-global
registry, or scheduler thread.

## Public model

- `ExecutionContext<TMachine>` names the logical model: stable `ContextId`,
  owned machine state, lifecycle state, wait/sleep metadata, and failure reason.
- `ContextTable<TMachine, Capacity>` is the concrete bounded owner. It uses a
  struct-of-arrays layout because queues carry only IDs and never duplicate a
  machine value.
- `Schedulable<TMachine>` requires one ordinary
  `ExecutionStep StepScheduled(ref TMachine)` operation. Adapters decide how an
  application machine maps its existing step result to scheduler disposition.
- `FifoReadyQueue<Capacity>` is a caller-sized ring. It rejects duplicate IDs,
  exhaustion, and empty dequeue with `SchedulerError`.
- `FifoScheduler` owns the context table, ready queue, monotonic clock,
  structured trace, configuration, and scheduling authority.
- `Worker` identifies an execution resource. The authoritative R7c
  implementation has one worker; a context is never an OS thread.

`ScheduleOne` performs one FIFO selection and at most `stepQuantum` semantic
steps. A still-runnable context returns to the tail. `RunUntilIdle` takes an
explicit maximum decision count and distinguishes idle from budget exhaustion.
Quanta are semantic step counts, not milliseconds.

## Lifecycle and wakeups

The supported states are Ready, Running, Waiting, Sleeping, Completed, and
Failed. Valid scheduler transitions are Ready to Running; Running to Ready,
Waiting, Sleeping, Completed, or Failed; and Waiting/Sleeping to Ready. A
completed or failed context is never requeued. Failure remains local and its
typed reason is retained.

`NotifyEvent` wakes contexts waiting for an `EventKind`. An interrupt adapter may
turn `InterruptWakeup` into the corresponding event notification, but the
interrupt source does not choose execution order. `WakeExpired` uses the
scheduler's `Clock` and `Deadline`; deterministic tests advance that explicit
clock and never sleep the host.

## Allocation, ownership, and lifetime

The fixed-capacity scheduler and its steady-state tick use only inline arrays.
There is no `malloc`, `calloc`, `realloc`, or `free` in the bounded generated-C
path. If policy state is provisioned dynamically, it is an ordinary value owned
through `Standard.Memory`; DragonGod has no allocator. Context IDs increase
monotonically and are not reused during a scheduler lifetime. R7c does not expose
context removal, so stale-ID aliasing cannot occur. Ordinary aggregate Drop
destroys the scheduler-owned table and its machine values.

## Determinism and threading boundary

With the same initial machines, inputs, explicit clock, and configuration, the
single-worker FIFO trace and final state are deterministic and replayable. The
trace records ready, selected, quantum begin/end, wait, wake, sleep, completion,
and failure facts through the existing bounded `TraceRecorder`.

DragonGod currently has no general atomic or synchronization substrate adequate
for shared scheduler mutation. R7c therefore does not add a host-thread backend
or claim multi-worker determinism. A future worker backend must isolate platform
threads behind DragonGod platform capabilities and preserve this scheduler as
the single-worker semantic authority. R7c does not target POSIX scheduling
compatibility.
