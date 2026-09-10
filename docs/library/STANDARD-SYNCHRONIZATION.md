# Standard synchronization

`Standard.Synchronization.Atomic` supplies the R7d1 strict-C11 machine
substrate: `AtomicInt`, explicit `MemoryOrder`, and load, store, exchange,
compare-exchange, and fetch-add operations. The ordinary library owns the API;
MIR and Planner retain general atomic semantics.

R7d2 adds compiler-owned source propositions rather than new runtime objects:

```concept
concept SpscChannel<TProducer, TConsumer, TQueue>
{
    requires sync.SingleProducer<TProducer, TQueue>;
    requires sync.SingleConsumer<TConsumer, TQueue>;
}
```

The six initial propositions are `sync.ExclusiveWriter`,
`sync.SingleProducer`, `sync.SingleConsumer`, `sync.SynchronizedAccess`,
`sync.PublishedBefore`, and `sync.ExactlyOnce`. Ordinary `Disjoint` remains
the sole storage non-overlap fact.

These propositions erase before runtime. A concrete module declaration records
explicit authority; a generic requirement must be established at concrete
substitution. Imported authority comes only from the dependency's hashed
semantic artifact.

The current Standard library does not yet expose a guard or spin lock. Adding
one before derived access proofs and a real second consumer would not authorize
safe elision. Direct atomic operations remain the low-level escape hatch.
