# EVT1 instance-sensitive proof transport

Status: R7d6 meaningful progression. Imported generic access evidence closes,
and generic guard decisions use only the concrete owner's access set.

**Invariant:** A closed generic instantiation must have a closed proof
environment. Type closure without proof closure is incomplete semantic
instantiation. No bound parameter may survive in a concrete access subject,
execution context, or proof subject consumed by a decision.

`concept-module.v1` now carries open generic method access entries with an
`open_generic_owner` and structural `Type` subjects. A consumer preserves that
symbolic entry and projects a separate entry for each canonical concrete
generic instance. `instance` identifies the full concrete application. The
projection uses the existing structural type binding operation; the exporter
never mutates its open generic declaration. The closed entry keeps
`ModuleAccessSummary` origin after import and gets a deterministic identity.
Converging call paths with the same structural access retain one bounded
entry and canonically use the earliest sequence position. This keeps recursive
summaries finite and independent of fixpoint visitation order.
Parse and Generate preserve the demanded concrete entries across independent
semantic analysis passes.

Substitution closes symbolic subjects. Derivation establishes concrete proof
status. Substitution alone does not turn an open requirement into Proven
evidence. The existing bounded writer, producer, and consumer derivation runs
over closed access sets. A competing producer yields Disproven; an unresolved
context yields Unknown. A declared contradiction is rejected.

Missing or incomplete concrete proof evidence becomes Unknown. Unknown retains
synchronization. Older artifacts without open access entries cannot acquire
proof merely because their semantic payload contains a generic body.

## Remaining compiler boundary

`EventBus<Configuration, Capacity>` exports open Publish/Consume access entries,
but the methods carry `OrdinaryContext`: neither generic parameter denotes an
execution context, and the imported call sites are not yet propagated into
those method entries. Generic Planner and generated-C guard decisions now use
the same concrete-owner filter. An open declaration, another concrete instance,
and missing evidence cannot authorize one instance's guard removal. The
existing R7d5 no-Publish/Consume policy still retains the real EventBus guard
in both SPSC and MPSC uses.

R7e native workers exercise EventBus publication and consumption through
compiled Concept call paths. A consumer observes the preceding Blackboard
write. This establishes correctness with the retained producer guard; it does
not add generic call-site context attribution or claim SPSC guard elision.

The next bounded step is a general call-site-to-imported-method access mapping
with concrete receiver and caller execution context, followed by a separately
justified publication-aware guard rule. It must preserve the verification
policy and never infer a producer from a generic type name.
