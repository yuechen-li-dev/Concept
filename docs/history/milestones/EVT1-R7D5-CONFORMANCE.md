# EVT1 R7d5 conformance

R7d5 is the synchronization-proof-consumption milestone. Its executable
authority is `internal/concept/sync_consumption_test.go`, the existing R7d1-R7d4
tests, the EVT1 corpus, and package build/test commands.

## Milestone status

The delivered result is **Meaningful progression**, not full R7d5 success.
Worker safety and general proof-backed guard/atomic simplification are real,
but the current lowering cannot specialize an imported generic `EventBus`
method from a concrete caller's `SingleProducer`/`SingleConsumer` proof. The
SPSC use therefore retains the safe producer guard. Closing that gap requires
instance-sensitive proof transport into imported generic method lowering; a
queue-name branch or second handwritten SPSC API would be a brittle substitute.
The MPSC implementation is safe and the generic same-source elision mechanism
is demonstrated independently. Scheduler wake policy and the requested
per-operation performance matrix remain deferred with that integration gap.

## Required outcomes

| Concern | Conforming evidence |
|---|---|
| Evidence matrix | Proven may optimize; Unknown retains; Disproven contradicts a declared contract |
| Synchronized access | Atomic, exact single-context, or disjoint structural access proves; partial ordinary cross-context access does not |
| Publication | Write -> Release Publish -> Acquire Consume -> Read is required |
| Guard | Owned non-copyable guard, Acquire CAS, Release store, exactly one release |
| Atomic planning | Verification retains `_Atomic`; optimized exact private access uses ordinary storage |
| Guard planning | Exact single-context guard operations elide; cross-context/unknown operations remain |
| Blackboard | Per-slot writer guards, atomic publication and dirty state, separate insertion guard |
| EventBus | Bounded payload-before-count publication, Acquire consumption, guarded MPSC publication, explicit Full |
| Actuation | Stable ID, Pending -> Claimed -> Committed CAS, owned claim authority |
| Native seam | Bounded Win32 create/join tests; no scheduler worker loop |
| Artifacts | Access order/mechanism/sequence and semantic summaries use the existing module artifact |
| Allocation | Bounded steady-state operations contain no allocation construct and retain `NoAllocation` plans |

The negative cases are authoritative: relaxed-only publication remains
Unknown, partial ordinary synchronization is Disproven, an unknown execution
context retains synchronization, duplicate actuation transitions fail, and
the existing R7d4 contradiction tests continue to reject incompatible writer
or producer declarations.

## Generated-code checks

The focused tests compile one source twice. Verification C contains `_Atomic`
storage and C11 atomic operations. Optimized C for an exact private context
contains ordinary storage and loads/stores. A cross-context publication source
retains Release/Acquire atomics. Guard calls are similarly retained by default
and omitted only under the exact single-context conjunction.

The generated native concurrency fixtures compile as strict C11, explicitly
create and join every thread, and establish these invariants:

- a consumer observes the published Blackboard value;
- two disjoint slots progress independently;
- two producers append without overwrite;
- exactly one of two workers claims and commits one actuation.

Stress is supplemental to semantic proof. The native interleaving may vary;
the final invariants may not.

## Scope boundary

R7d5 does not add a multi-worker scheduler, detached threads, a second queue or
Blackboard API, memory-order weakening, a runtime proof/race registry, general
alias analysis, lock-free queue assumptions, or external exactly-once claims.
Those boundaries preserve R7c scheduler semantics and leave R7e responsible
for worker scheduling and wake policy.
