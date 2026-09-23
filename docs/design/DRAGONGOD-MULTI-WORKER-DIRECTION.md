# DragonGod multi-worker direction

R7e maps caller-owned native workers to logical `ExecutionContext` values.
`WorkerId` is stable semantic identity; it is never a native handle. The
platform-neutral `ParallelScheduler` owns one context table, a bounded ready
queue of IDs, and one short guard for shared scheduling state. A worker
claims Ready -> Running under that guard, releases it, calls the existing
`StepScheduled` for one semantic quantum, then publishes the result under the
guard. No machine value moves into a queue or host-thread object.

The Win32 native validation adapter creates, runs, and joins host threads.
The scheduler contains no Win32 call and can later be driven by a bare-metal
core with the same `RunWorker` semantics. Host creation may allocate outside
the bounded scheduling path. The application must keep shared fixture storage
alive through every join. Registration and clock advancement use the guarded
API; direct mutation of the public `core` while workers run is outside the
contract.

Ready IDs are unique. Registration reserves a queue slot for every live
context, including Running contexts, so release can always requeue. A wake
arriving during a quantum is recorded per context and applied to its Waiting
or Sleeping outcome. One event kind can be pending for a Running context; a
second distinct kind returns `CapacityExhausted`, requiring the caller to
retry. Deadline release rechecks the explicit monotonic clock. Idle means
there is no ready or running work; callers may later wake and run again.

The worker trace is a guarded bounded ring with explicit overwrite count.
It records selection, claim, quantum begin/end, wake, requeue, completion,
and failure with `WorkerId`. It is diagnostic state, not replay authority.
Single-worker FIFO scheduling remains deterministic. Native multi-worker
interleavings vary; final semantic invariants are the oracle.

R7e prioritizes real multi-worker functionality over isolated proof-system
perfection. Safe over-synchronization is acceptable. Incorrect or unprovable
concurrency is not. EventBus still retains its safe producer guard because
an imported generic method's access summary has unresolved caller execution
context. R7e does not use library-name recognition to close that proof.

Collector observations for R7f are per-context roots in the context table,
shared Blackboard roots, worker rendezvous, and explicit machine Step
boundaries. R7e adds no collector, work stealing, priorities, affinity,
SMP startup, or hidden scheduler runtime.
