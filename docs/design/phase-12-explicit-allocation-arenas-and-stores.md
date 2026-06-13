# Phase 12: Explicit allocation, arenas, and ID-based stores

P12-M0 is a documentation-only milestone. It defines Concept's explicit
allocation doctrine, arena model, allocation effects, and ID-based store
direction before implementation begins.

P12-M0 does not implement compiler behavior. It does not add parser syntax,
allocator runtime code, arena allocation intrinsics, generic containers,
region-lifetime checking, MIR operations, backend lowering, fixtures, or a CLI
command.

Phase 12 starts after the closed Phase 11 first-class testing work. Earlier
phases established the required substrate: Phase 6 introduced unsafe/raw pointer
v0, Phase 7 introduced runtime structs and places, Phase 8 introduced
concepts/templates and marker concepts, Phase 9 introduced deterministic
compile-time execution v0, Phase 10 introduced ownership/move/Drop/ManualInit
storage-state accounting, and Phase 11 introduced a native testing substrate.

## Core doctrine

```text
Allocation is explicit.

Allocation has an effect.

Arenas are deterministic storage regions, not lifetime magic.

Arena reset or destroy invalidates arena storage. Concept does not globally
prove arbitrary arena escape safety.

Raw arena pointers are low-level and auditable.

Compiler-style data should prefer stable IDs and stores over fragile pointer
graphs.

No hidden process-global heap is assumed by the core language.

Drop types are not arena-allocated in v0 unless destructor-list machinery
exists.

Concept does not become Rust 2 through region lifetimes.

Concept does not become C++ allocator template cathedral.
```

## 1. Motivation

Concept now has enough language substrate to need an allocation doctrine before
implementation grows new storage APIs by accident. The closed phases gave the
language unsafe pointers, runtime structs and places, concepts/templates,
comptime, ownership/move/Drop/ManualInit storage-state accounting, and a native
test system. The next foundational missing piece is explicit allocation and
storage-region discipline.

Compilers, runtimes, kernels, parsers, ASTs, HIR/MIR stores, diagnostics,
symbol tables, build tools, allocators, and high-performance native libraries
all need allocation. Concept cannot pretend those allocations are ambient
convenience. Allocation must be visible in APIs, visible in effects, and visible
to review.

Phase 12 should avoid both bad exits:

- hidden global heap convenience;
- Rust-style lifetime-region machinery.

The first exit makes core language operations secretly depend on hosted runtime
state. That is incompatible with freestanding, kernel, embedded, and no-runtime
profiles. The second exit turns arenas into lifetime parameters, region
generics, and borrow-checker-shaped API pressure. Concept is explicitly not Rust
2.

Arena allocation is useful. It is a natural tool for parsers, short-lived
compiler phases, diagnostics, scratch data, page-oriented runtime structures,
and subsystems that want deterministic bulk release. But raw arena pointers
should not become the default compiler data model. Long-lived compiler graphs
written as pointer forests are fragile under reallocation, serialization,
incremental rebuilding, diagnostics, and phase boundaries.

Bad/default pointer forest style:

```cpp
Expr* expr = arenaAlloc<Expr>(arena);
```

Preferred compiler-data style:

```cpp
ExprId expr = ExprStore.add(store, Expr.IntLiteral(42));
Expr node = ExprStore.get(store, expr);
```

Raw arena pointers may still exist for low-level work, FFI, runtime plumbing,
kernels, custom allocators, and tightly audited internals. For compiler-authoring
code, stable IDs and stores should be the preferred pattern.

## 2. Design thesis

```text
Arena allocation is not the goal by itself.

Explicit storage discipline is the goal.
```

A Concept program should make storage ownership visible. The code should show
whether storage comes from a caller-provided allocator, an arena, a fixed store,
stack storage, a kernel heap, a device page allocator, or some later hosted
default allocator.

Allocation should be represented in semantic effects. A function that allocates
is not semantically identical to a function that only computes. Profiles and
callers need to know the difference.

Bulk storage should be deterministic. Arena reset and destroy are valuable
because they give clear storage-phase boundaries and predictable cleanup of
non-Drop storage.

Pointer escape from storage regions should be auditable. Concept should not
pretend arbitrary raw pointers remain valid after arena reset or destroy.

Compiler data should use stable handles where possible. IDs make ownership,
storage, comparison, serialization, and diagnostics simpler than long-lived raw
pointers.

The language should support freestanding profiles where heap allocation is
unavailable, forbidden, or supplied only through explicit target-specific
allocators.

## 3. Allocation effects

Phase 12 reserves the initial allocation-effect vocabulary:

```text
noalloc
alloc
```

Possible future source shape:

```cpp
noalloc int Add(int a, int b) {
    return a + b;
}

alloc ExprId ParseExpr(Parser& parser) {
    // may allocate
}
```

Initial rules:

- `noalloc` means the function must not directly allocate.
- `alloc` means the function may allocate.
- Effect checking may start shallow in v0.
- Full transitive effect inference/checking is future work.
- Allocation effects are metadata and semantic constraints, not optimizer hints
  only.

Direct allocation inside a `noalloc` function should be rejected once allocation
intrinsics exist. Until allocation intrinsics exist, P12-M1/P12-M2 can scaffold
metadata and shallow checking without pretending all allocation paths are known.

Allocation effects should eventually interact with profiles:

- freestanding;
- kernel;
- compiler/tooling;
- hosted/application.

Freestanding and kernel profiles may default many declarations to `noalloc`.
Hosted/application profiles may permit more allocation, but the core language
still should not assume a process-global heap.

Suggested future diagnostics:

```text
CON0190 AllocationInNoAllocFunction
CON0191 AllocationEffectMismatch
CON0197 DuplicateAllocationEffect
CON0198 AllocationEffectInvalidTarget
```

### P12-M1 implementation status

P12-M1 implements function-level allocation-effect syntax:

```cpp
noalloc int Add(int a, int b) {
    return a + b;
}

alloc int BuildSomething() {
    return 42;
}
```

The Stage 0 lexer reserves `alloc` and `noalloc` as keywords. Function
declarations store an `AllocationEffect` with the states `unspecified`,
`noalloc`, and `alloc`; existing functions without an explicit effect default
to `unspecified`. The effect is preserved from AST function declarations into
HIR function declarations, including generic/template function bodies and
generic instantiations.

`unsafe`, `comptime`, and allocation effects are independent metadata axes in
P12-M1. The parser accepts them as leading function specifiers before the
return type. Duplicate allocation effects are rejected with
`CON0197 DuplicateAllocationEffect`, and conflicting `alloc`/`noalloc`
specifier pairs are rejected with `CON0191 AllocationEffectMismatch`.
Allocation effects on non-function targets that the parser can reach are
rejected with `CON0198 AllocationEffectInvalidTarget`.

P12-M1 intentionally does not add allocation operations, arenas, allocator
runtime code, direct allocation checking, transitive effect checking, call graph
effect checking, region lifetime checking, generic stores, MIR allocation
operations, backend allocation lowering, or hidden heap behavior.

### P12-M2 implementation status

P12-M2 makes `noalloc` meaningful for direct function calls. During HIR
checking, a function marked `noalloc` may directly call only callees whose
allocation effect is known `noalloc`. Direct calls from a `noalloc` caller to an
`alloc` callee are rejected with `CON0191 AllocationEffectMismatch`.

Functions with omitted allocation effects remain `unspecified`. An
`unspecified` callee has not promised `noalloc`, so direct calls from a
`noalloc` caller to an `unspecified` callee are also rejected with `CON0191`.
For now, `alloc` callers and `unspecified` callers may call `noalloc`, `alloc`,
or `unspecified` callees.

This is shallow direct-call checking only. P12-M2 does not add allocation
operations, arenas, allocator runtime code, transitive effect inference or
checking, profile-specific defaults, region lifetime checking, generic stores,
MIR allocation operations, backend allocation lowering, or hidden heap
behavior.

### P12-M3 implementation status

P12-M3 adds compiler-known allocation surface type names:

```text
Arena
Allocator
AllocError
```

These names are resolved by the Stage 0 compiler as a v0 bridge because Concept
does not yet have the core/prelude module surface needed to host them as normal
library declarations.

`Arena` is an opaque storage-region handle placeholder. `Allocator` is an
opaque allocator placeholder. Neither type has runtime behavior, allocation
behavior, fields, constructors, reset/destroy behavior, or ordinary struct
literal construction in P12-M3. By-value `Arena` and `Allocator` signatures are
rejected with `CON0199 OpaqueAllocationTypeByValueUnsupported`; the intended
early shape is pointer use such as:

```cpp
noalloc int UsesArena(Arena* arena) {
    return 0;
}

noalloc int UsesAllocator(Allocator* allocator) {
    return 0;
}
```

`AllocError` is a small compiler-known value placeholder reserved for future
allocation failure surfaces. It is copyable, has no Drop behavior, exposes no
public fields, and does not imply any allocation operation.

The MIR C backend can spell opaque allocation handles in pointer position as:

```c
struct cpt_Arena* arena;
struct cpt_Allocator* allocator;
```

It does not emit by-value incomplete opaque handle types. `AllocError` currently
lowers as an integer placeholder in backend-supported value positions.

Future operations remain reserved and unimplemented:

```cpp
Arena arena = Arena.create();       // future
T* value = Arena.alloc<T>(arena);   // P12-M4
Arena.reset(arena);                 // future P12-M5
Arena.destroy(arena);               // future P12-M5
```

P12-M3 does not add `Arena.alloc<T>`, `Arena.create`, reset/destroy semantics,
allocator runtime code, hidden heap behavior, ID stores, MIR allocation
operations, backend allocation lowering beyond opaque pointer spelling, region
lifetime checking, arena pointer escape analysis, or Drop-in-arena behavior.

### P12-M4 implementation status

P12-M4 adds a narrow compiler-recognized arena allocation intrinsic with the
source spelling:

```cpp
alloc Vec2* MakeVec(Arena* arena) {
    Vec2* value = Arena.alloc<Vec2>(arena);
    return value;
}
```

`Arena.alloc<T>(arena)` requires exactly one explicit type argument and exactly
one value argument. The value argument must have type `Arena*`. The result type
is the raw pointer type `T*`.

Arena allocation is an allocation operation. It is accepted inside functions
marked `alloc`, accepted inside functions with unspecified allocation effect for
now, and rejected inside `noalloc` functions with
`CON0190 AllocationInNoAllocFunction`. This diagnostic is distinct from the
P12-M2 direct call-edge diagnostic `CON0191 AllocationEffectMismatch`.

P12-M4 v0 supports only concrete non-Drop allocated types. Allocation of a type
with a `Drop<T>` impl is rejected with
`CON0194 ArenaAllocDropTypeUnsupported`. Allocation of `Arena` or `Allocator`
by value is rejected with `CON0199 OpaqueAllocationTypeByValueUnsupported`.
`AllocError` remains a copyable value placeholder and may be allocated.

The HIR contains a dedicated `arena_alloc` expression carrying the arena
operand, allocated type, result type, and source span. MIR lowers this to an
explicit `ArenaAlloc` rvalue. The MIR C backend emits an external helper
declaration and an auditable helper call:

```c
void* cpt_arena_alloc(struct cpt_Arena* arena, unsigned long size, unsigned long align);
(T*)cpt_arena_alloc(arena, sizeof(T), _Alignof(T))
```

The backend does not provide a runtime implementation and does not silently use
`malloc`. Hosted runtime support remains future work.

P12-M4 does not add arena create/reset/destroy, allocation failure handling,
`AllocError` return paths, destructor lists, arena allocation of Drop types,
`Arena.allocUninit<T>`, `Arena.allocInit<T>`, region lifetime checking, pointer
escape analysis, pointer field access sugar, `->`, auto-deref, ID stores, or a
hidden global heap. Arena allocation returns a raw pointer; existing raw pointer
rules still apply. The pointer value is initialized, but pointed-to storage
initializedness is not tracked by Phase 12 v0.

## 4. Profiles and hidden heap policy

Phase 12 policy:

```text
Concept core does not assume a process-global heap.

Hosted profiles may provide a default allocator later.

Freestanding/kernel profiles may reject default allocation entirely.

Allocation APIs should prefer explicit allocator or arena parameters.
```

Consequences:

- no implicit `new`;
- no hidden allocation in core operations;
- no default heap baked into the language core;
- allocator-aware APIs pass allocators or arenas explicitly;
- ordinary operations must not silently allocate hidden memory;
- core compiler semantics must remain compatible with no-heap environments.

If a hosted default allocator is added later, it is profile-specific. It is not
part of Concept's freestanding core semantics, and it must not make allocator
requirements disappear from systems-profile APIs.

## 5. Arena model

Arenas are explicit deterministic storage regions.

Conceptual API shape, not final syntax:

```cpp
Arena arena = Arena.create();

T* value = Arena.alloc<T>(arena);

Arena.reset(arena);
Arena.destroy(arena);
```

Rules:

- arena allocation is visible;
- arena reset invalidates all storage from that arena;
- arena destroy invalidates all storage from that arena and releases backing
  storage;
- Concept v0 does not globally prove that arena pointers do not escape;
- obvious escape checks may be added later, but no full lifetime-region system;
- arena-allocated pointers are raw/auditable;
- arena values should not pretend to be ordinary owning values.

Explicit doctrine:

```text
An arena is a storage region.
It is not a proof of lifetime safety.
```

Arenas are useful because they give a deterministic storage phase. They are not
safe because the word "arena" appears in a type. A raw pointer returned from an
arena remains subject to ordinary raw pointer rules.

## 6. Arena pointer safety and escape honesty

Arena pointers may dangle after reset or destroy. Concept should not pretend
this is safe by default.

Phase 6 already established unsafe raw pointer dereference. Arena allocation may
eventually be implemented as a safe or unsafe operation depending on the chosen
surface, but dereferencing raw arena pointers remains subject to raw pointer
safety rules. If dereference of `T*` requires `unsafe`, dereference of an
arena-returned `T*` also requires `unsafe`.

The language may later require unsafe blocks for raw arena pointer operations,
for arena allocation itself, or for some subset such as raw pointer dereference
and pointer arithmetic. That choice belongs to an implementation milestone after
the allocation surface is concrete.

Escape analysis is future work. Concept may later reject obvious cases, such as
returning a raw pointer allocated from a local arena that is destroyed before the
return. Such checks are useful diagnostics. They are not a commitment to:

- Rust-style lifetime parameters;
- region generics;
- a borrow checker;
- a fake safety story.

Possible future diagnostics:

```text
CON0192 ArenaPointerEscapeUnsupported
CON0193 ArenaUseAfterDestroyObvious
```

## 7. Drop interaction

Phase 12 v0 rule:

```text
Arena allocation v0 supports only types that do not require Drop.
```

Arena reset and destroy free storage in bulk. If an arena stores values with
`Drop<T>`, reset/destroy must either:

- run destructors; or
- leak semantic resources.

Running destructors requires machinery Concept does not have for arenas yet:

- destructor lists;
- destructor registration;
- destructor ordering;
- partial initialization handling;
- failure behavior;
- interaction with `Drop<T>` witness calls;
- interaction with moved values and storage-state analysis.

That is future work, not v0. Therefore, v0 should reject arena allocation of
Drop types.

Invalid:

```cpp
struct File {
    int handle;
};

impl Drop<File> {
    void drop(File f) {
        // close handle
    }
}

Arena arena = Arena.create();
File* f = Arena.alloc<File>(arena); // invalid in v0
```

Suggested diagnostic:

```text
CON0194 ArenaAllocDropTypeUnsupported
```

Trivial and non-Drop structs may be arena-allocated. Drop-in-arena may be added
later with explicit destructor registration. Arena reset/destroy must not
silently bypass Drop for Drop types.

## 8. ManualInit interaction

`ManualInit<T>` is explicit manual storage-state control from Phase 10. Arena
storage and `ManualInit<T>` both deal with raw, uninitialized, or explicit
storage, but they are not the same thing.

```text
ManualInit<T> is per-place initialization control.

Arena is a bulk storage region.
```

Combining arenas and `ManualInit<T>` is future work. Phase 12 v0 should avoid
creating hidden initializedness loopholes through arena allocation.

Possible v0 rule:

```text
Arena.alloc<T> produces initialized T storage only if construction/initialization
is explicit and semantically checked.
```

Possible future surface:

```text
Arena.allocUninit<T> is future work and must interact with ManualInit.
```

Phase 12 does not design a full uninitialized arena allocation model. If a later
milestone adds uninitialized arena storage, it must integrate with
`ManualInit<T>`, unsafe boundaries, storage-state diagnostics, Drop restrictions,
and partial initialization rules.

## 9. ID-based stores

Core doctrine:

```text
Compiler-shaped data should prefer stable IDs over long-lived raw pointers.
```

Stable IDs are the better default for AST, HIR, MIR, symbol tables, type stores,
diagnostics, package graphs, import graphs, and compiler phase data.

Why:

- IDs survive array/vector reallocation.
- IDs are serializable and comparable.
- IDs make ownership and storage graphs simpler.
- IDs avoid borrowed pointer forests.
- IDs work well for AST/HIR/MIR/symbol tables/type stores.
- IDs are easier for tooling and diagnostics.
- IDs avoid subtle invalidation bugs.

Conceptual API:

```cpp
struct ExprId {
    int index;
};

struct ExprStore {
    // implementation hidden for now
};

ExprId id = ExprStore.add(store, Expr.IntLiteral(42));
Expr expr = ExprStore.get(store, id);
```

Possible generic future shape:

```cpp
Store<Expr, ExprId> exprs;

ExprId id = exprs.add(Expr.IntLiteral(42));
Expr expr = exprs.get(id);
```

P12 may not implement generic stores immediately. The design reserves the model:
ID stores may be backed by arenas, vectors, pages, segmented arrays, or
compiler-managed storage. The ID is the stable user-facing handle. Raw pointers
may be temporary implementation details for fast local access, but should not be
the public long-lived graph model.

## 10. Store ID typing

IDs should avoid accidental cross-store and cross-kind mixing.

Possible future explicit types:

```cpp
struct ExprId {
    int index;
};

struct TypeId {
    int index;
};
```

Possible future syntax, not implemented in Phase 12:

```cpp
id ExprId for Expr;
id TypeId for Type;
```

Rules:

- `ExprId` should not be interchangeable with `TypeId`.
- Store APIs should use explicit ID types.
- Untyped integer indexes are not the public model.
- IDs may eventually include generation counters for stale-ID detection, but
  that is not required for v0.

Generation counters are useful for debug builds, incremental stores, and stale
handle detection. They should remain an implementation choice until the language
surface has a reason to standardize them.

Possible future diagnostics:

```text
CON0195 StoreIdTypeMismatch
CON0196 InvalidStoreIdUse
```

## 11. Allocation API layering

Phase 12 defines three conceptual layers.

Layer A: raw allocation primitives.

- Low-level.
- Unsafe or auditable.
- For runtimes, kernels, custom allocators, embedded systems, and allocator
  implementations.

Layer B: arenas.

- Deterministic bulk storage.
- Good for parser/compiler phases, scratch data, diagnostics, and phase-local
  work.
- Reset/destroy invalidates storage.

Layer C: ID stores.

- Preferred compiler-facing abstraction.
- Stable handles.
- Backed by arena/vector/page storage internally.

```text
Most compiler code should live at Layer C.
Low-level systems code may use Layer A/B deliberately.
```

This layering keeps Concept from becoming either a hidden-heap language or a
C++ allocator-template maze. Low-level allocation exists. Arena allocation
exists. Compiler-shaped APIs should still present stable IDs where possible.

## 12. MIR direction

M0 does not implement MIR changes. Future MIR representation may include
allocation-related operations such as:

```text
AllocRegionCreate
AllocRegionDestroy
ArenaAlloc
ArenaReset
StoreAdd
StoreGet
```

Alternatively, early implementation may keep arena/store operations as
compiler-recognized calls until the lowering story is stable.

MIR direction:

- allocation effects should be represented or at least visible to MIR
  validation;
- `noalloc` checking may happen before or during MIR validation;
- storage-state analysis should understand allocation-related places enough not
  to create ownership loopholes;
- Drop restrictions should be checked before lowering or during HIR checking;
- backend code should not infer hidden allocation behavior absent from MIR or
  checked calls.

Phase 10 already made Drop and storage-state explicit in MIR. Phase 12 should
preserve that direction: allocation should not be a backend surprise.

## 13. C backend direction

M0 does not implement backend changes.

Future C backend direction:

- Arena may lower to a runtime support struct or C helper calls.
- ID stores may lower to arrays, vectors, pages, or backing storage chosen by
  the implementation.
- Freestanding backend output should not assume `malloc` unless the active
  profile permits it.
- Hosted backend output may later use malloc-backed arenas for convenience.
- Backend code must not silently allocate hidden memory for ordinary operations.

If a backend helper allocates, that allocation must correspond to an explicit
source allocation operation or a profile-permitted runtime support operation.
Generated C should remain auditable.

## 14. Testing plan

Phase 11 gives Phase 12 a native test substrate. Future Phase 12 tests should
use `.con_test` where possible once the relevant compiler surfaces exist.

Future tests:

- `noalloc` rejects direct allocation;
- arena allocation works for non-Drop structs;
- arena allocation rejects Drop types;
- arena reset/destroy behavior is represented;
- store add/get returns correct values;
- ID type mismatches are rejected;
- raw pointer operations still require unsafe where appropriate.

Testing allocation semantics requires clear separation between compile
diagnostics and runtime test failures. A rejected `noalloc` allocation or
Drop-in-arena attempt is a compile diagnostic. A store add/get behavior test is
a runtime/test-runner result once stores have executable support.

## 15. Non-goals for Phase 12 v0

Phase 12 v0 explicitly does not include:

- garbage collection;
- tracing GC;
- reference counting;
- hidden global heap;
- default allocator in core language;
- Rust-style borrow checking;
- Rust-style region lifetimes;
- lifetime parameters;
- full escape analysis;
- general allocator trait/vtable system;
- C++ allocator template model;
- arbitrary realloc;
- per-object arena Drop/destructor lists;
- arena allocation of Drop types;
- uninitialized arena allocation unless tied to `ManualInit<T>` later;
- generic `Vector<T, Allocator>`;
- complete generic `Store<T, Id>`;
- thread-local allocators;
- async/task allocators;
- memory sanitizer/debug allocator;
- compacting arenas;
- moving GC;
- pointer provenance formalization.

These are not rejected forever. They are rejected from Phase 12 v0 so the first
allocation work establishes the doctrine cleanly.

## 16. Milestone plan

Proposed Phase 12 milestone ladder:

```text
P12-M0  Design doc: explicit allocation, arenas, and ID-based stores
P12-M1  alloc/noalloc effect syntax and metadata scaffold
P12-M2  direct allocation-effect checking scaffold
P12-M3  Core allocation surface: AllocError, Arena, allocator placeholders
P12-M4  Arena allocation intrinsic for non-Drop concrete types
P12-M5  Arena reset/destroy semantics and invalidation rules
P12-M6  Drop interaction hardening follow-ups
P12-M7  MIR allocation/effect representation and C backend hardening
P12-M8  ID-based store design/prototype and examples
P12-M9  Closeout
```

This ordering keeps effects and metadata ahead of allocation operations, then
adds arena behavior, Drop hardening, MIR/backend representation, and finally
compiler-facing stores. If implementation discovers that ID-store scaffolding is
needed earlier to test arena-backed storage sanely, P12-M8 may move earlier, but
the doctrine should remain unchanged.

## 17. Docs index / PoC status

PoC3 already establishes the constitutional allocation direction:

- no hidden heap in core/freestanding layers;
- explicit allocator-aware allocation;
- hosted profiles may provide default allocators later;
- systems and freestanding profiles should not;
- stable IDs and index-based compiler arenas are the canonical
  compiler-authoring pattern;
- no core region lifetime system.

This Phase 12 document specializes those commitments into the implementation
roadmap for explicit allocation, arenas, and ID-based stores. It should be
listed from any future design-document index if one is added.

## 18. Close criteria

P12-M0 is successful if:

- explicit allocation doctrine is documented;
- `alloc`/`noalloc` effect direction is documented;
- hidden process-global heap policy is documented;
- arena reset/destroy invalidation is documented;
- arena pointers are described as raw/auditable rather than safe-by-region;
- Drop types are rejected from arena allocation v0 unless destructor-list
  machinery exists;
- `ManualInit<T>` interaction is separated from arena storage;
- ID-based stores are documented as the preferred compiler-data model;
- typed store IDs are reserved as the public handle shape;
- MIR and C backend directions are sketched without implementation;
- Phase 12 non-goals prevent region-lifetime, hidden-heap, and allocator-template
  drift.

## P12-M0 status

P12-M0 adds only this design document. It intentionally does not implement
lexer, parser, AST, HIR, MIR, backend, fixture, allocation intrinsic, allocator
runtime, arena, store, effect checker, region lifetime checker, CLI, or generic
container changes.
