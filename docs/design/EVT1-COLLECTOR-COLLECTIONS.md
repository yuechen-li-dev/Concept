# Collector Collections design

Concept does not have a mandatory garbage collector. Collector Collections
are ordinary libraries over explicit allocation, roots, tracing, and
reclamation. The R7fR implementation is `Standard.Collection.Core`: bounded,
homogeneous, non-moving, manually triggered mark/sweep. It uses no runtime
reflection, conservative scan, global root registry, hidden GC runtime, or
collector-specific compiler, MIR, or Planner branch.

Each metadata slot owns an optional Standard.Memory `Allocation<T,A>` plus
occupancy, mark, and generation fields. The allocator provides storage;
`Trace<T>` enumerates outgoing collector handles; roots determine initial
reachability. A fixed local array supplies the mark worklist, and mark on
enqueue prevents duplicate queue entries. Sweep in slot order drops each
unreachable allocation, whose Drop destroys the object before releasing its
region. Object addresses never move.

Borrowed references derive from the collector's storage. `Collect` invalidates
collector-derived borrows, and a borrow must end before collection. The
R7f2 conservative suspension rule still bars resource-sensitive leases
across await/yield. Internal tracing runs under the exclusive collection call
using ordinary readonly allocation views and a bounded visitor.

`Trace<T>` is synchronous. `Standard.Collection.DeriveTrace` now derives an
ordinary `TraceReferences` witness for explicitly `[[trace]]` annotated
`CollectorHandle<T>` fields; the handwritten witness remains valid and the
collector has no dependency on reflection. General nested composite and
payload-enum derivation remains open. Heterogeneous
collections would need explicit static Trace/Destroy witnesses, rather than
ambient RTTI. Moving, generational, concurrent, and weak-reference policies
remain deferred. DragonGod can later place `Collect` at a scheduler quantum
boundary and coordinate workers using its existing rendezvous mechanisms;
R7fR introduces no scheduler integration or collector thread.

The current domain ID is chosen by the caller and must differ between
independent collectors. The current root API uses manual integer tokens.
These explicit contracts are recorded in the library documentation rather
than hidden in runtime state.
