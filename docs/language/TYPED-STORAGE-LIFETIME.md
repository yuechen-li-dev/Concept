# Typed storage lifetime

Status: EVT1 R6l general field-state substrate implemented; nested generic
failure-carrier transport remains blocked

`Storage<T>` denotes suitably aligned and sized storage. It does not denote a
live `T`. `Initialize(storage, value)` begins one object lifetime in that
storage, and `Destroy(storage)` ends it while preserving the raw storage and
its region geometry.

The compiler tracks object state for a complete storage access path, including
nested aggregate fields. The state is compile-time-only and adds no initialized
bit, registry, or other metadata to `Storage<T>`. A branch joins differing
states conservatively. Double initialization, destruction of uninitialized
storage, and `Value(storage)` without a proven live object reject.

`Value(storage)` is the general storage operation used by an ordinary owner
method to derive `ref T` or `ref const T`. The reference inherits the storage
and owner provenance; destroying the object invalidates retained references.
A move with an outstanding object reference rejects conservatively.

Moving `Storage<T>` or an aggregate containing it transfers the descriptor,
object state, provenance, and destruction responsibility. It does not move or
copy the bytes of the live `T`. `Storage<T>` remains structurally non-copyable.
A generic user `Drop(owned Owner<T>)` may explicitly call
`Destroy(owner.storage)`; concrete instantiations are selected through ordinary
generic Drop resolution.

Open `bind<T>` is legal. The one-argument form accepts a trusted
`MemoryRegion<SystemMemory>` and defers concrete size/alignment checks until
instantiation/lowering. The existing address-plus-extent form remains valid.

R6l does not add an allocator, heap, pointer syntax, shared ownership, or hidden
object-state runtime. The current blocker is `Result<Owner<T>, E>`: nested open
generic applications are canonicalized before failure-carrier extraction and
the caller observes `Owner<T>` rather than `Owner<Concrete>`. Extending that
general carrier substitution path is required before R6l can be complete.
Imported generic owners also still expose their open `Storage<T>` field to the
consumer as unresolved `T` (`CV4148`); the deterministic artifact is preserved,
but consumer instantiation is not yet valid.
