# DragonGod synchronization direction

An agentic thread is a schedulable execution context that coordinates with
other agents through explicit shared authorities such as EventBus,
GlobalBlackboard, and Actuation. It is not merely an OS thread wrapper.

Concept source expresses synchronization intent. Diagnostics and proofs
establish correctness. Runtime guards remain where proof is incomplete.
Proven-redundant guards may be erased for production execution.

## Authority

EventBus owns bounded communication, observation, and wake notification.
GlobalBlackboard owns explicit shared application or kernel knowledge.
Actuation owns stable side-effect obligations and internal claim/commit
authority. None is a process-global singleton, and none gives the scheduler
authority over application meaning.

`GlobalBlackboard` means explicit state shared within one configured runtime or
domain. The word global never means ambient mutable process state.

Exactly-once claims apply to internal scheduling and commit authority. A
foreign device or service must state its own retry and idempotency contract;
internal CAS cannot make an external physical effect transactional.

## Lowering doctrine

The source states the safe operation once. Proven exclusivity or disjointness
may later remove a mechanical guard. Publication ordering still remains when
readers can run concurrently. Unknown proof state retains the safe mechanism;
Disproven unsynchronized mutation is rejected.

R7d currently supplies the initial strict-C11 integer atomic substrate. It does
not yet supply the shared-access proof relations or guarded DragonGod
algorithms, so no atomic weakening, fence removal, lock elision, or race-freedom
claim is made yet.

Collector implications remain future work: shared roots require explicit
ownership, root mutation requires synchronization, and a collector may later
coordinate at semantic machine-step boundaries. R7d adds no collector.
