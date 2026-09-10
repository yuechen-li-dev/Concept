# EVT1 allocator direction

Status: R6j resolves trusted external-region authority; reference allocator library remains deferred to R6k

## Doctrine

```text
MemorySource
    says where raw storage comes from.

Allocator
    says how storage is carved or reused.

Allocation<T, TAllocator>
    owns one initialized T placed in allocator-provided storage.
```

These remain intended ordinary imported Concept library abstractions. The
compiler must not recognize allocator names, invent a hidden heap, add an
allocator MIR family, or lower allocator calls through a private backend path.

## R6f feasibility result

R6d and R6e resolved the six blockers recorded by R6c: generic runtime types,
`SizeOf<T>()` / `AlignOf<T>()`, structural generic ownership and provenance,
reusable semantic modules, the bounded external C ABI, and authoritative
operation effects. R6f nevertheless exposed a more primitive general gap that
the R6c inventory had grouped under raw storage rather than its blocker
checklist:

```text
ordinary raw storage manipulation
    raw-address offset and comparison
    overflow-safe usize arithmetic
    raw byte region -> typed storage binding
    initialization through that typed binding
```

The current `byte*` surface is an external-ABI carrier. Core expressions have
no pointer arithmetic, address-of, or dereference operation. The existing
`bind` authority accepts established array/ndarray storage and produces
non-owning array/ndarray views; it cannot bind `byte*` or a runtime byte
subregion as arbitrary `T` storage. Even ordinary `usize + usize` is outside
the current scalar arithmetic set.

That makes all three policies impossible as ordinary library code:

- a bump allocator cannot compute or return an aligned subregion;
- a pool cannot identify, validate, or reuse raw slots;
- an arena cannot carve a parent region into child regions.

It also blocks typed `Allocation<T, TAllocator>` from performing the required
visible sequence `SizeOf<T>()`, `AlignOf<T>()`, allocate, bind, initialize.
Pretending that a `Span<byte>` is an arbitrary typed object would bypass the
established element-type and provenance rules rather than use them.

A second, independent ergonomic gap is that function templates still accept
exactly one type parameter. The requested ordinary
`Allocate<T, TAllocator>(ref allocator)` helper cannot be declared. This gap
does not by itself block raw allocators, but it blocks the canonical typed
helper and must be resolved generally rather than by an allocator-specific
overload.

## R6g substrate result

R6g supplies unit-aware byte geometry, affine `Address<Space>`, an ordinary
`MemoryRegion<SystemMemory>` library with `AlignUp`/`Subregion`, `Storage<T>`
bind/Initialize/Destroy, and ordered multi-parameter function templates. It
introduces no allocator.

R6h resolves the remaining prerequisite: region origin, interval, alignment,
and lifetime facts survive ordinary values, slicing, generic wrappers, module
artifacts, and calls. An arbitrary `AddressFromBits` remains Unknown and cannot
be bound as trusted storage.

| prerequisite | status |
|---|---|
| generic runtime types | Resolved |
| SizeOf/AlignOf | Resolved |
| generic ownership/provenance | Resolved |
| semantic modules | Resolved |
| host ABI seam | Resolved |
| operation effects | Resolved |
| quantity/address/storage model | Resolved |
| semantic fact transport | Resolved |
| trusted foreign storage authority | Resolved |

The allocator framework is still not implemented. R6j supplies only the general
owned/leased foreign storage boundary and a hosted specimen; it adds no retry,
policy, heap, pool, arena, or allocator-specific compiler rule.

## Required next substrate decision

Before the allocator library can resume, a post-freeze proposal must define a
small, general explicit-storage boundary. At minimum it must prove:

- checked arithmetic for `usize`, including overflow detection;
- a canonical non-owning raw-region value with byte extent and alignment;
- bounded raw-region slicing/offset operations with preserved provenance;
- explicit binding and initialization of suitably sized/aligned raw storage as
  `T`, without ownership transfer or a hidden copy;
- enough generic function parameters or inference to express a typed helper.

This proposal is not permission to add allocator types, `new`/`delete`, a
global heap, reinterpret casts, unchecked pointer arithmetic, variadics, a GC,
or a runtime registry. The storage operation must be useful independently of
allocators and carry size, alignment, mutability, and provenance through the
ordinary semantic fact system.

## Preserved allocation-effect law

`compiler.Allocates` continues to mean an allocation operation boundary, not
necessarily a heap or syscall. Once an ordinary allocator exists, its
allocation operation must carry that effect through `concept-module.v1` so
`NoAllocation` is Disproven for callers. Pure metadata queries remain eligible
for Proven, while unrelated opaque extern calls remain Unknown.

## Deferred allocator surface

After the explicit-storage substrate exists, R6f may implement, under ordinary
semantic library roots:

- `MemoryRegion` and `AllocationError`;
- `MemorySource` and minimal honest capability splits;
- fixed-buffer and hosted memory sources;
- bump, pool, and arena allocators;
- movable-only typed allocation owners with object Drop before storage release;
- Span/tensor projections backed by explicit allocated storage.

No allocator source or placeholder API is added by this feasibility stop.
Doing so would publish contracts that have no honest executable
implementation.
