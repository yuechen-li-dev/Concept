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

`Standard.Synchronization.Guard` adds the smallest general guarded substrate:
`SpinLock`, Acquire, Release, `Lock`, and an owned non-copyable `Guard` tied to
the borrowed lock. It uses R7d1 compare-exchange Acquire and store Release
operations; it never uses `volatile` or a registry. An explicit
`[[synchronization]]` semantic attribute, rather than operation or type names,
marks general Acquire/Release effects for access evidence and planning.

Guard removal requires the R7d5 proof conjunction: exact subjects, one explicit
execution context, and no publication/consume dependency. Unknown evidence
retains the guard. Direct atomic operations remain the low-level escape hatch.
