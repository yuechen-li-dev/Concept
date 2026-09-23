# DragonGod synchronization

DragonGod's intended coordination surface is EventBus, GlobalBlackboard, and
Actuation. These are explicit caller-owned authorities, not ambient globals.

The first R7d substrate is `Standard.Synchronization.Atomic`. `AtomicInt` is an
value whose verification storage lowers to C11 `_Atomic int`. Quiescent
bounded checkpoints retain value-copy behavior; concurrent access remains
restricted to the atomic API. Its ordinary library
API is `LoadAtomic`, `StoreAtomic`, `ExchangeAtomic`,
`CompareExchangeAtomic`, and `FetchAddAtomic`.

Every operation has an explicit-order overload and a sequentially consistent
default overload. Explicit orders are Relaxed, Acquire, Release,
AcquireRelease, and Sequential. Load, store, read-modify-write, and CAS-failure
orders use operation-specific C11 mappings; an invalid combination is
strengthened to sequential consistency.

The default verification Planner conservatively retains every atomic. R7d5's
optimized policy may simplify a whole private exact single-context atomic set
when no publication or ordering edge depends on it. Cross-context readers keep
release/acquire operations. There is no `volatile` fallback and no R7d5 memory
order weakening.

R7d5 makes the existing components worker-safe without introducing competing
APIs. `MemoryState` uses per-slot writer guards plus atomic publication and a
separate insertion guard, so disjoint occupied slots do not serialize on a
global lock. `EventBus` writes bounded payload under a producer guard before a
Release count publication; readers use Acquire count observation. `ActuatorHost`
uses atomic Pending -> Claimed -> Committed transitions and an owned claim
authority. R7e exercises these components under native multi-worker scheduling;
the EventBus producer guard remains because SPSC caller-context evidence is
still Unknown.
