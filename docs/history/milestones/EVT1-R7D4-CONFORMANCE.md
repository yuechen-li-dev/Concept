# EVT1 R7d4 conformance — semantic access summaries

R7d4 derives bounded compiler-only access summaries from ordinary typed
Concept syntax. The closed vocabulary is `Read`, `Write`, `AtomicRead`,
`AtomicWrite`, `Publish`, `Consume`, `Claim`, and `Commit`. Subjects retain a
semantic root identity plus structural field/index path; display names are not
the sole identity. Resolution is `Exact`, `Interval`, or `Opaque`.

`[[execution_context(ContextType)]]` explicitly marks an entry operation.
`[[semantic_access("Publish", parameter)]]` (and the other closed access
kinds) explicitly gives a function or foreign declaration channel semantics.
The compiler never infers those meanings from function names.

Local assignments produce writes and value operands produce reads. Fixed
indices retain interval identity; dynamic indices are Opaque. The five R7d1
atomic intrinsics produce distinct atomic reads/writes, with read-modify-write
operations producing both. Direct calls substitute parameter subjects and a
finite monotone fixpoint closes recursive call graphs. A function reachable
from multiple declared contexts has `SharedContext`, which blocks positive
cardinality proof.

The focused `TestR7d4*` suite covers exact writes, propagation, contexts,
producer/consumer contracts, two-writer falsification, declared contradiction,
dynamic-index opacity, atomic distinction, recursion termination, foreign
opacity/authority, artifact transport, deterministic serialization, and the
method-name anti-heuristic.

## Current DragonGod falsification pass

The existing DragonGod sources were analyzed before any worker-safe rewrite.
They do not yet declare R7d4 execution contexts or channel access annotations,
so the honest first pass is conservative:

| Component | Subject | Derived authority | Resolution | Result | Why |
| --- | --- | --- | --- | --- | --- |
| Blackboard | mutable memory slots | ordinary reads/writes only | context open | Unknown | no declared concurrent entry context |
| EventBus | bounded event queues | no semantic Publish/Consume contract | channel open | Unknown | names are not semantics |
| Actuation | claim/commit obligations | no semantic Claim/Commit contract | obligation open | Unknown | exactly-once control-flow proof is deferred |

This is the expected falsification result, not a worker-safety claim. Synthetic
two-context fixtures produce real `Disproven` results and
`SYNC_AUTHORITY_CONTRADICTION`; DragonGod is not annotated merely to make the
matrix green.

R7d4 does not elide guards, locks, or atomics and does not make Blackboard,
EventBus, or Actuation multi-worker safe.
