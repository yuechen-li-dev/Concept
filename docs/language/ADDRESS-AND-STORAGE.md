# Address and storage

Status: EVT1 R6g foundation; provenance-proof integration remains bounded

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
an address lexically derived from `AddressOf(ref backing)`, byte units, fixed
geometry, and emits extent/alignment guards. Reconstructed addresses remain
unknown-provenance and reject. Bind allocates and copies nothing.
`Initialize(storage, value)` transitions Uninitialized to
Initialized and returns `ref T`; `Destroy(storage)` invokes ordinary `Drop`
when present and returns to Uninitialized. Double initialization and destroying
uninitialized storage are errors. `Storage<T>` itself is never readable as T.

`Span<T>` remains a view of live contiguous objects; layout remains declared
storage geometry; tensor shapes remain dimensionless element counts.

R6g adds no source `T*`, implicit cast, object-scaled pointer arithmetic,
reinterpret cast, MMIO/volatile rule, allocator, heap, or runtime metadata
registry. The remaining hard boundary is transport of interval/provenance facts
through arbitrary ordinary region values. Trusted-external-region introduction
and known subregion disjointness remain the next semantic-fact task; the
compiler rejects rather than inventing provenance for `AddressFromBits`.
