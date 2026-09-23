# Collector Collections design boundary

Concept does not have a mandatory garbage collector. The intended
`Standard.Collection` library would combine explicit allocation, explicit
roots, synchronous explicit `Trace<T>`, and explicit reclamation. It would be
bounded, homogeneous, non-moving, and manually triggered. No runtime
reflection, conservative stack scan, hidden GC runtime, or async borrow is
part of this design.

The first storage model should use `ReleasableAllocator<A>` to obtain each
object region, `Storage<T>` to own the initialized object authority, and a
fixed slot table to hold generation, mark, region, and storage state. A fixed
local worklist would carry slot indices. Each outgoing handle would be
enumerated by the user's synchronous `Trace<T>` operation. Sweep would call
`Destroy` exactly once before `Release`, in slot order.

The table cannot yet hold an allocated storage authority after construction:
`Option<Storage<T>><array>[Capacity]` starts empty, but indexed assignment of
`move Option::Some(storage)` is rejected as a copy of a non-copyable type.
That diagnostic is the current design gate. Any language repair must apply to
ordinary non-copyable arrays and preserve their lifetime and Drop behavior.

Collector borrows must be scoped. A future `Collect` must declare
`compiler.InvalidatesBorrows(Collect, collector)`, so a collector-derived
borrow ends before Collect. R7f2 already rejects resource-sensitive leases
across await and yield; no suspension exception is proposed. Multiple
collectors must have distinct handle domains without a global registry.

Heterogeneous type witnesses, derived Trace through future reflection,
reference counting, weak references, generational or moving collection,
concurrent collection, and DragonGod rendezvous are deferred. Manual
`Trace<T>` would remain valid if a future reflection facility derives it.
