# Standard.Memory

`Standard.Memory` is ordinary Concept library code. The compiler does not know
allocator, policy, or owner names. Its modules publish as `concept-module.v1`,
and consumers instantiate concepts and generic bodies without source reparse.

## Ownership layers

The layers remain separate:

1. A memory source owns or borrows the authority that makes bytes usable.
2. An allocator owns policy state for carving or reusing raw regions.
3. `MemoryRegion<SystemMemory>` describes origin, byte interval, and alignment.
4. `Storage<T>` is typed but uninitialized storage authority.
5. `Initialize` begins a live `T`; `Destroy` ends that object lifetime.
6. `Allocation<T,A>` owns the live object and allocator region. A `ref T` from
   `AllocationValue` is only a borrow bounded by those authorities.

This is not pointer ownership. An allocator is a policy for obtaining or
carving storage; a heap is only one possible storage source or organization.
There is no global/default allocator or hidden heap.

## Modules and policies

- `Standard.Memory.Core` defines `AllocationError`, `Allocator<A>`, composed
  `ReleasableAllocator<A>`, alignment validation, and overflow-checked
  `CheckedAlignUp`.
- `Standard.Memory.Bump` owns a fixed 256-byte, four-byte-aligned inline source
  and a monotonic bump policy. Padding is charged exactly. It has no individual
  `Release` and therefore does not satisfy `ReleasableAllocator`.
- `Standard.Memory.Pool` is a bounded 16-slot policy over 64 inline bytes. Its
  inline bitmap tracks exact origin, slot start, length, alignment, live state,
  exhaustion, double release, and deterministic reuse.
- `Standard.Memory.Ownership` defines releasable `Allocation<T,A>`, typed
  `Allocate<T,A>`, mutable/const value borrows, and constrained generic Drop.
- `Standard.Memory.MonotonicOwnership` provides the owner for allocators without
  individual release.
- `Standard.Memory.Host` wraps `Platform.Host.Memory.HostAllocation`; its region
  remains borrowed from that owned host authority and host free runs once.

The fixed source is intentionally corpus-sized rather than pretending to be a
production firmware capacity. Applications may construct a bump allocator over
any trusted region whose alignment satisfies their requests.

## Lifetime and failure laws

Typed allocation visibly performs `SizeOf<T>`, `AlignOf<T>`, allocator
`Allocate`, `bind<T>`, and `Initialize`. Failure uses
`Result<Allocation<T,A>, AllocationError>`; null is not a failure channel.

Releasable Drop performs `Destroy(storage)` and then `Release(region)`.
Monotonic Drop performs only `Destroy(storage)`; its raw bytes remain reserved.
Moving either owner transfers small authority descriptors and never relocates
live `T` bytes. Copy rejects, so even an immovable `T` can remain stationary
while its owner moves.

`compiler.Allocates` marks an ordinary allocation boundary. The effect crosses
concepts, generic calls, and imported artifacts. Inspection helpers remain
`NoAllocation`; opaque foreign operations without an effect contract remain
Unknown.

Arena policy, callable move capture, async persistence, and machine-field
integration are bounded follow-ons. They are not prerequisites for the fixed,
pool, hosted, typed-owner, module, or strict-C11 paths delivered here.
