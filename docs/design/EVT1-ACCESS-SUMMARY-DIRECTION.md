# EVT1 access-summary direction

Explicit sync propositions describe what must be true. Access summaries
describe what code actually does. Derived proofs compare the two. Planner
optimization is a later consumer of those proofs.

An access subject is structural: semantic module/function/value or parameter
identity, parameter ordinal, field/index path, resolved value type, and any
known region interval. Text is retained for diagnostics, but a concatenated
name is never the internal authority. Existing region and `Disjoint` facts
remain authoritative; R7d4 adds no second storage geometry model.

`Exact` closes a semantic place. `Interval` closes a backing identity and
fixed slot. `Opaque` means the compiler cannot close the relevant shared-access
identity. Opaque is not failure, but it prevents positive exclusivity or
cardinality proof. Dynamic indices and mutable foreign references without an
explicit contract are Opaque.

Declared sync facts are not inputs to access-set derivation. Otherwise the
proof system would merely rediscover its assumptions. Matching derived
evidence strengthens a declaration; multiple known contexts disprove it;
incomplete evidence remains Unknown. `SynchronizedAccess`, `PublishedBefore`,
and `ExactlyOnce` remain explicit/Unknown unless their stronger path and
control-flow conditions are independently established.

Access entries are deterministic, hash-covered `concept-module.v1` data.
Imported entries become `ModuleAccessSummary`; foreign explicit contracts keep
`DeclaredForeign`. A missing older summary is Unknown. No source reparse,
runtime registry, arbitrary pointer alias analysis, borrow checker, SMT solver,
or whole-world analysis is introduced.
