# EVT1 allocator direction

Status: R6l progression; direct typed-storage ownership works, nested generic carriers remain blocked

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

## R6k feasibility result

R6k re-ran the prerequisite gate against the real ordinary-library path. Eight
historical blockers remain resolved, but the combined
`generic -> bind<T> -> Initialize -> owner field -> Drop/Destroy` path is not.
The failure is below allocator policy and is reproducible without declaring an
allocator:

| prerequisite | R6k result |
|---|---|
| generic runtime types | Resolved |
| `SizeOf<T>` / `AlignOf<T>` | Resolved |
| generic ownership/provenance/Drop | **Not resolved for initialized `Storage<T>` owners** |
| semantic modules | Resolved |
| `extern "C"` | Resolved |
| `Allocates` | Resolved |
| quantities/address/storage | **Not resolved for open-template `bind<T>`** |
| semantic fact transport | Resolved for value facts; operational initialized state does not survive owner fields |
| trusted external storage authority | Resolved |

Concrete evidence is pinned by `r6k_allocator_feasibility_test.go`:

- an ordinary template body calling `bind<T>` rejects its open type parameter
  with `CV4148` before consumer instantiation;
- `Destroy(owner.storage)` is not a storage lifetime operation and rejects as
  unknown function `CV4027`, because `Initialize`/`Destroy` accept only a named
  `Storage<T>` local;
- recovering the field is not an escape hatch: `move owner.storage` rejects
  with `CV4507`, because move accepts only a whole local or parameter place;
- retaining the `ref T` returned by `Initialize` in a returned ref owner rejects
  with `CV4521`; its provenance is not related through the typed-storage binding
  to the backing parameter.

These are general typed-storage ownership and provenance gaps. An
`Allocation<T, TAllocator>` cannot truthfully preserve the required sequence,
provide `Value`, or implement `Destroy(T) -> Release(region)` exactly once.
Bump, pool, and arena policy would only create raw regions around that missing
owner transition, so R6k stops before publishing partial allocator APIs.

## Required next substrate decision

Before the allocator library can resume, a post-freeze proposal must complete
the general typed-storage ownership boundary. At minimum it must prove:

- compiler-known storage operations accept an open template type and validate
  after ordinary concrete substitution;
- initialized/uninitialized object state transports through aggregate fields
  and whole-owner moves without runtime proof metadata;
- `Destroy` can consume an initialized storage place reached through an owned
  aggregate, or an equally general structural Drop rule can express the same
  transition;
- the reference returned by `Initialize` retains the backing region/source
  provenance when stored in and returned with its owner;
- moving the owner remains exactly-once while copying it remains structurally
  rejected.

This proposal is not permission to recognize allocator or owner names, add
`new`/`delete`, a global heap, reinterpret casts, unchecked pointer arithmetic,
variadics, a GC, or a runtime registry. The transition must be useful for any
ordinary initialized typed-storage owner and must remain compile-time-only.

## Preserved allocation-effect law

`compiler.Allocates` continues to mean an allocation operation boundary, not
necessarily a heap or syscall. Once an ordinary allocator exists, its
allocation operation must carry that effect through `concept-module.v1` so
`NoAllocation` is Disproven for callers. Pure metadata queries remain eligible
for Proven, while unrelated opaque extern calls remain Unknown.

## Deferred allocator surface

After the initialized-storage owner substrate exists, a later milestone may
implement under ordinary semantic library roots:

- `MemoryRegion` and `AllocationError`;
- `MemorySource` and minimal honest capability splits;
- fixed-buffer and hosted memory sources;
- bump, pool, and arena allocators;
- movable-only typed allocation owners with object Drop before storage release;
- Span/tensor projections backed by explicit allocated storage.

No allocator source or placeholder API is added by this feasibility stop.
Doing so would publish contracts that have no honest executable
implementation.

## Allocator history

```text
R6c
    generic/library/effect substrate missing
R6f
    quantity/address/storage substrate missing
R6i
    trusted external storage authority missing
R6j
    foreign contracts solved external authority
R6k
    initialized Storage<T> cannot cross the typed-owner boundary
R6l
    direct initialized owner fields work; nested generic Result/Option carrier
    substitution remains the allocator-readiness blocker
```
