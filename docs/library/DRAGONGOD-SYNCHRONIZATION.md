# DragonGod synchronization

DragonGod's intended coordination surface is EventBus, GlobalBlackboard, and
Actuation. These are explicit caller-owned authorities, not ambient globals.

The first R7d substrate is `Standard.Synchronization.Atomic`. `AtomicInt` is an
immovable value whose storage lowers to C11 `_Atomic int`. Its ordinary library
API is `LoadAtomic`, `StoreAtomic`, `ExchangeAtomic`,
`CompareExchangeAtomic`, and `FetchAddAtomic`.

Every operation has an explicit-order overload and a sequentially consistent
default overload. Explicit orders are Relaxed, Acquire, Release,
AcquireRelease, and Sequential. Load, store, read-modify-write, and CAS-failure
orders use operation-specific C11 mappings; an invalid combination is
strengthened to sequential consistency.

The current Planner conservatively retains every atomic. Future weakening
requires a proven ownership or exclusivity fact and must preserve publication
ordering. There is no `volatile` fallback.

The existing R7b `MemoryState`, `EventBus`, and `ActuatorHost` remain
single-worker components. They must not be shared between host workers until
the remaining R7d authority proofs and guarded algorithms are completed.

R7d2 Phase 1 supplies explicit source propositions for exclusive writer,
single producer/consumer, synchronized access, publication visibility, and
internal exactly-once authority. They compose and cross semantic module
artifacts, but DragonGod does not yet emit derived access summaries for its
real operations. These components therefore remain single-worker; a source
declaration is not being substituted for proof of their algorithms.
