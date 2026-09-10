# EVT1 shared-access proof direction

## Doctrine

Explicit first. Derived second. Optimized last.

Every synchronization fact used for correctness or optimization must first
have an explicit source-level representation. Automatic derivation improves
authoring and strengthens evidence; it is not a hidden semantic system.

Compiler-owned concepts are semantic propositions understood by the compiler.
They remain normal Concept concepts from the programmer's perspective: they
can be required, composed, asserted, explained, and transported through
modules. They require no runtime metadata.

## Phase 1 authority

The initial concurrent domain is one configured DragonGod runtime domain:
different execution contexts that may eventually be eligible to run on
different workers. It is neither process-global timeless uniqueness nor a
distributed-systems claim.

`ExclusiveWriter<W, S>` means mutation of `S` in that domain is authorized
through `W`. `SingleProducer<P, Q>` and `SingleConsumer<C, Q>` bound queue
publication and consumption authority. `SynchronizedAccess<S>` states that
conflicting concurrent accesses are mediated by synchronization adequate for
the declared memory semantics. `PublishedBefore<A, B>` is a bounded
publication/visibility edge, not general temporal logic. `ExactlyOnce<O>` is
exactly one internal claim/commit authority; it says nothing about external
physical side effects.

Concrete module declarations are trusted contracts only where the language
permits them. They retain `Declared` origin and cannot override a proven
contradiction. Generic requirements add obligations, not facts. Foreign trust
must eventually use the existing explicit foreign boundary and retain
`DeclaredForeign` origin.

## Derivation boundary

Phase 2 should add a bounded, demand-driven access summary with general access
kinds such as `Read`, `Write`, `AtomicRead`, `AtomicWrite`, `Publish`,
`Consume`, `Claim`, and `Commit`. Each entry must carry structural subject,
operation, context when statically known, module origin, and imported-summary
provenance.

For a subject, one closed known writer may derive `ExclusiveWriter`; multiple
known writers disprove a single-writer claim; an opaque path yields `Unknown`.
Producer and consumer derivation follows the same cardinality rule. Known
fixed slot non-overlap reuses existing `Disjoint` interval authority. Dynamic
keys remain `Unknown`. This does not require arbitrary pointer alias analysis.

Artifacts should transport bounded access summaries and substitute generic
subjects through the existing generic closure path. A closed artifact must
contain no unresolved template parameters. Imported source is never reopened.

## Optimization authority

Only Proven synchronization facts may authorize removal or weakening of
correctness-affecting synchronization. Unknown retains the safe mechanism.
Disproven rejects an incompatible contract.

Exclusive writer authority alone never proves that a plain store is safe:
publication to concurrent readers may still require release/acquire ordering.
A future general Planner decision may retain or elide synchronization only
from proof evidence, never from `GlobalBlackboard`, `EventBus`, `Actuation`, or
other library type names.

No runtime proof/race registry, volatile synchronization, whole-world scan, or
second shared-storage system belongs in this design.
