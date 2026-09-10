# Address and storage

Status: EVT1 R6j includes bounded trusted external storage authority

The affine algebra is `Address<Space> + usize<byte> -> Address<Space>` and
`Address<Space> - Address<Space> -> isize<byte>`. Address plus address is
invalid. Address spaces are ordinary nominal struct tags; different spaces do
not combine implicitly. `AddressFromBits<Space>`, `AddressBits`, and
`AddressOf<Space>(ref backing)` are explicit boundaries. C11 represents an
address as `uintptr_t`; its space tag is compile-time-only.

`MemoryRegion<Space>` is ordinary generic library code containing start,
length, and alignment. `Standard.MemoryGeometry` supplies `SystemMemory`,
`AlignUp`, exact common-alignment degradation, and bounded `Subregion`. A
region denotes `[start, start + length)` and implies neither ownership nor a
live object.

`Storage<T>` is the static typed-uninitialized view at the lifetime boundary.
`bind<T>(address, extent)` requires the `SystemMemory` host-accessible space,
an address derived from `AddressOf(ref backing)` or an explicitly declared
foreign storage authority, byte units, fixed
geometry, and emits extent/alignment guards. Reconstructed addresses remain
unknown-provenance and reject. Bind allocates and copies nothing.
`Initialize(storage, value)` transitions Uninitialized to
Initialized and returns `ref T`; `Destroy(storage)` invokes ordinary `Drop`
when present and returns to Uninitialized. Double initialization and destroying
uninitialized storage are errors. `Storage<T>` itself is never readable as T.

R6k established an important current limit: those operational states are
tracked only for named `Storage<T>` locals. Open-template `bind<T>` does not yet
validate, initialized state does not travel through aggregate fields, a storage
field cannot be moved out or passed to `Destroy`, and the reference returned by
`Initialize` cannot yet be returned as part of a storage owner with the
backing's provenance. This is a general typed-storage ownership gap, not an
allocator rule; allocator library work remains blocked on resolving it without
runtime metadata.

`Span<T>` remains a view of live contiguous objects; layout remains declared
storage geometry; tensor shapes remain dimensionless element counts.

R6j's foreign path binds the region to a live owned/leased wrapper and preserves
`DeclaredForeign` through function and module summaries. The guarded
establishment operation is restricted to the declaring module and erases to an
ordinary region value in C. Owner escape, longer-lived assignment, and use after
move/drop reject. `AddressFromBits` remains untrusted.

R6g/R6j add no source `T*`, implicit cast, object-scaled pointer arithmetic,
reinterpret cast, MMIO/volatile rule, allocator, heap, or runtime metadata
registry. Freshness/disjointness and MMIO/volatile remain deferred; the compiler
rejects rather than inventing provenance for reconstructed addresses.
