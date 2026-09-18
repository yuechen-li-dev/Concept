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

R7d1 supplies the strict-C11 atomic substrate. R7d2 supplies explicit
shared-access propositions. R7d4 supplies structural access evidence and
tri-state writer/producer/consumer derivation. No atomic weakening, fence
removal, lock elision, or race-freedom claim is made.

R7d2 Phase 1 supplies the explicit relation vocabulary and artifact transport.
The relations are real source-level obligations, not analyzer-only facts, but
current DragonGod sources deliberately remain unannotated and single-worker;
their first R7d4 pass is mostly Unknown. Worker-safe Blackboard, EventBus, and
Actuation algorithms plus proof-consuming Planner decisions are later work.

Collector implications remain future work: shared roots require explicit
ownership, root mutation requires synchronization, and a collector may later
coordinate at semantic machine-step boundaries. R7d adds no collector.
