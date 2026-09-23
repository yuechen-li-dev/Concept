# DragonGod workers

`DragonGod.Scheduling.Parallel` wraps the R7c `FifoScheduler` without changing
its single-worker API. Construct it with bounded context, queue, and trace
storage, then register machines. A host supplies one `Worker` with a distinct
`WorkerId` per native execution resource and calls `RunWorker` from each host
thread. Join every thread before inspecting unguarded final state.

`RunWorker` repeatedly claims one ready ID, steps its machine outside the
scheduler gate, and publishes Ready, Waiting, Sleeping, Completed, or Failed.
It returns when no Ready or Running contexts remain. Waiting contexts can be
woken with `Wake` or `NotifyEvent`; `AdvanceTime` updates the monotonic clock
and wakes expired sleepers. A wake racing a Running quantum is retained until
release. Completed and Failed contexts cannot be woken or requeued.

The gate covers queue, lifecycle, wake bookkeeping, and worker trace. It is
never held across `StepScheduled`. Application shared state uses its own
worker-safe authorities: per-slot guarded `MemoryState`, release/acquire
`EventBus`, and CAS-claimed `ActuatorHost`. Avoid direct writes to `core`,
including its clock and trace, during worker execution.

The queue reserves one slot per registered context. Registration after all
slots are used returns `CapacityExhausted` even if some contexts have
completed; IDs and slots are not recycled. Event notification can return
`CapacityExhausted` if two different event kinds would need to be remembered
for one running context. Callers can retry after that quantum ends.

The native Win32 test adapter lives in
`internal/concept/parallel_scheduler_test.go`: it creates and joins 1, 2,
and 4 host threads and invokes generated strict-C11 Concept functions.
This is the initial host backend. POSIX and bare-metal adapters are deferred.
