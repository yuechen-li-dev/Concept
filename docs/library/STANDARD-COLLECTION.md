# Standard.Collection

R7g1 adds `Standard.Collection.DeriveTrace`. `derive DeriveTrace reflect<Node>;` emits an ordinary checked `TraceReferences` witness. Direct handle fields are selected through the library's `TraceEdge<E,T>` concept; `[[trace_array]]` marks fixed handle arrays and table columns for ordinary iteration. A `TraceNested<F,T>` witness delegates from a nested composite field. `derive DeriveEnumTrace reflect<Choice>;` creates an exhaustive payload-enum match using the same edge concept. The collector continues to accept handwritten witnesses.

`Standard.Collection.Core` ships a bounded homogeneous, non-moving mark/sweep
collector as ordinary Concept library code. Concept has no mandatory garbage
collector. The caller selects a `ReleasableAllocator<A>`, creates
`MarkSweepCollector<T, Capacity, A>`, registers roots explicitly, implements
`TraceReferences(ref const T, ref TraceVisitor<T>)`, and calls `Collect`.
There is no runtime reflection, stack scan, hidden heap, collector thread, or
collector-specific MIR instruction.

`AllocateObject` obtains a typed `Allocation<T,A>` through Standard.Memory and
moves its sole authority into a fixed `Option<owned Allocation<T,A>>` slot.
Allocation failure leaves the slot free and returns `OutOfMemory`. Each handle
contains a slot index, generation, and caller-selected domain. Callers must
give independent collectors distinct domains. `Borrow` and `BorrowMutable`
reject stale, foreign-domain, and out-of-range handles. Slot reuse increments
the generation. A slot whose generation reaches the maximum `int` is
permanently retired on reclamation, so an ancient handle never aliases a
wrapped generation.

`RegisterRoot` returns a bounded root-table index; duplicate roots are
independent registrations. `UnregisterRoot` removes that entry. Root indices
are manual tokens, so the caller must unregister each live token exactly once
and must not reuse a stale token after its entry has been reassigned. A scoped
`RootGuard` is deferred. Root and object capacity are both `Capacity`.

`Trace<T>` is synchronous and explicit. It enumerates outgoing handles through
`Visit`; invalid edges abort collection before sweep. `TraceReferences` gets
read-only object access. A fixed local index array and spans over collector
metadata form the worklist. Marking on enqueue places each slot on the list at
most once, including cycles and diamonds. Sweep walks slots in increasing
index order, replaces unreachable occupied Options with `None`, and thereby
runs `Allocation<T,A>` Drop: `Destroy<T>` before allocator `Release`. It then
clears occupancy and increments the generation. Collection complexity is
`O(Capacity + reachable edges)` with no collection-time worklist allocation.
The concrete PoolAllocator and no-edge trace witness pass an authoritative
`NoAllocation` assertion; an arbitrary user trace or allocator release must
be checked as part of its concrete collection policy.

Collector borrows are scoped. `Collect` declares
`compiler.InvalidatesBorrows(Collect, collector)`. A live collector-derived
borrow blocks it; the same call succeeds after the borrow's scope ends.
R7f2 also rejects resource-sensitive leases across await and yield. The
collector exposes no async borrow exception.

See [EVT1-R7FR-CONVERGENCE.md](../conformance/EVT1-R7FR-CONVERGENCE.md) for
native, artifact-only, graph, ownership, and determinism evidence. DragonGod
rendezvous, heterogeneous objects, moving or concurrent collection, weak
references, and finalizers remain outside this first library.
