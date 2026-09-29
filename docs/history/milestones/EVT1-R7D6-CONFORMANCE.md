# EVT1 R7d6 conformance

Result: **Meaningful progression**. Baseline `c4c0811`;
`R7d4` `b409e6a`; `R7d5` `c9a75c6`;
compiler `concept-evt1-stage0-go`; artifact `concept-module.v1`.

The failing synthetic artifact-only imported generic proof first returned
`Unknown` with `no closed access set for Channel<QueueX>`. Its artifact now
contains an inspectable symbolic generic access entry, and the consumer
derives `SingleProducer<AgentA, Channel<QueueX>> = Proven` from a closed entry.
One second instance with a competing AgentB producer is Disproven without
changing the first instance. A generic value argument `16` closes in the
subject identity, a generic execution context closes to AgentA, an ordinary
opaque context remains Unknown, and a contradictory concrete declaration
raises `SYNC_AUTHORITY_CONTRADICTION`. The artifact-only closed access set is
byte identical across 100 runs.

The first full suite run exposed a package artifact instability: distinct
call positions shared an access key even though their sequence positions
differed. Canonically retaining the earliest position for that bounded key
restored both 100-run package-graph determinism and recursive deduplication.

The DragonGod package artifact exports symbolic EventBus Publish, Consume,
Read, and Write entries. Its methods currently have ordinary
unresolved execution contexts. There is no real EventBus SPSC generated-C
elision evidence. Verification and optimized lowering retain the producer
guard. No R7e worker work was started.

The narrow Planner/backend scope test has three concrete owners: one exact
single-context instance selects `ElideSynchronization`, an overlapping
two-context instance selects `RetainSynchronization`, and an instance with no
evidence also retains the guard. Both Planner and C lowering call the same
concrete-owner predicate. Ordinary R7d5 planning is unchanged.

`go test ./...`, `go vet ./...`, both Zig test builds, `oct make BurnIn`,
Standard package build/test, and DragonGod package build/test passed.
Focused R7d4/R7d5/R7d6 tests passed separately.

The next blocker is concrete imported method call-site context propagation
and a justified publication-aware SPSC guard rule. The current EventBus
summary has `OrdinaryContext`, and the R7d5 guard policy rejects all
Publish/Consume sets, including concrete SPSC.
