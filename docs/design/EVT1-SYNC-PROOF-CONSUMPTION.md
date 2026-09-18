# EVT1 synchronization proof consumption

R7d5 consumes bounded shared-access evidence without changing the source
program or recognizing library type names. The governing rule is:

> Declared facts express intent. Derived facts describe observed program
> behavior. Correctness-affecting optimization consumes derived proof. Unknown
> retains the safe mechanism. Disproven rejects incompatible assumptions.

## Evidence and mechanisms

`SynchronizedAccess<S>` is Proven when every potentially conflicting access to
`S` is atomic, is structurally disjoint, or has one exact execution context.
An unresolved subject is Unknown. Overlapping ordinary accesses from distinct
contexts are Disproven. Atomic evidence records the operation, memory order,
mechanism, structural subject, execution context, and sequence position.

`PublishedBefore<A, B>` is a bounded structural publication edge. It requires
a write of `A`, a later Release or AcquireRelease publication, an overlapping
Acquire or AcquireRelease consume, and a later read of `B`. Source order without
the release/acquire pair is not evidence.

`ExactlyOnce<O>` requires an exact obligation identity, one atomic Claim CAS,
one atomic Commit CAS, and no reset transition. It proves internal commit
authority only. It does not make a foreign service, device, or physical effect
transactional.

`Standard.Synchronization.Guard` supplies the general fallback: `SpinLock`
acquires with compare-exchange Acquire semantics, releases with a Release
store, and returns an owned, non-copyable `Guard`. Guard lifetime is tied to a
borrowed lock. The guarded DragonGod paths release explicitly before early
returns because EVT1 cleanup insertion does not yet cover every early-return
shape.

## Planner boundary

Verification lowering is the default and retains C11 atomics and explicit
Acquire/Release calls. Optimized lowering is selected through
`OptimizedCompilationPolicy`; it uses the same source and semantic result.

A guard is elided only when all accesses have exact structural subjects, share
one explicit execution context, and no Publish/Consume ordering edge depends
on that guard. Any unknown context, overlapping cross-context access, opaque
subject, or publication edge produces `RetainSynchronization`.

An atomic is simplified to an ordinary operation only when every atomic access
in the compiled unit has exact identity, all accesses share one explicit
context, and no Publish/Consume or non-Relaxed ordering requirement exists.
One writer with a concurrent reader therefore retains atomics. R7d5 does not
weaken memory orders; `RetainMemoryOrder` is the only implemented decision.

Declarations are checked as contracts but are not optimization authority by
themselves. Independently derived Proven facts may support planning without a
declaration. Missing artifact evidence is Unknown and retains synchronization.

## DragonGod application

`MemoryState` uses fixed bounded slots. A short insertion guard protects key
allocation, each occupied slot has an independent writer guard, and atomic
value/occupied/dirty fields publish and observe state. Disjoint slots never
share a writer guard. Existing slots support concurrent readers through
Acquire loads, and dirty tracking is the same atomic slot state.

`EventBus` remains one bounded append-only implementation. A producer guard
makes the MPSC path safe; payload initialization precedes a Release publication
of count and consumers use Acquire observation. Full queues return `Full` and
never overwrite. Proof-backed guard simplification is general rather than an
`EventBus`-specific SPSC implementation; multiple consumers remain outside the
current API because the cursor is caller-owned.

`ActuatorHost` allocates stable IDs under a guard. Claim performs CAS from
Pending to Claimed and returns an owned `ActuationClaim`; Commit consumes that
authority and performs CAS from Claimed to Committed. Duplicate claim or
commit receives an explicit deterministic error.

No scheduler worker loop is introduced. Native tests provide only a bounded
Win32 create/join adapter; the Concept synchronization libraries contain no
platform thread names.
