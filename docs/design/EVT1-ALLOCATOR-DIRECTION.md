# EVT1 allocator direction

Status: R6e prerequisites structurally resolved; allocator remains unimplemented

## R6e prerequisite checkpoint

R6d supplied ordinary generic runtime types, `SizeOf<T>` / `AlignOf<T>`,
structural generic ownership/Drop/provenance, and bounded external C/effect
declarations. R6e supplies reusable semantic artifacts, exact import
resolution, cross-module constraints, authoritative effect summaries, and
interface-operation effect allowances. This is readiness evidence only: it
adds no allocator framework, heap policy, or allocator-specific compiler law.

## R6d blocker update

R6d has made meaningful local-compilation progress without retrying the
allocator:

| R6c blocker | R6d status |
|---|---|
| User-defined generic runtime types | Resolved locally for structs/classes, nesting, integer non-type parameters, functions, and methods; generic type constraints remain partial |
| Generic `SizeOf<T>` / `AlignOf<T>` | Resolved through the ordinary fixed-layout geometry authority, including after function-template substitution |
| Applied ownership/provenance/Drop | Resolved locally through concrete fields, structural reverse Drop, move checking, and explicit generic `ref struct` provenance |
| Reusable Core multi-module compilation | Unresolved; this is the next isolated blocker |
| Hosted external-memory / ABI seam | Resolved for bounded scalar/enum/builtin-pointer `extern "C"` declarations and strict-C11 linkage |
| Authoritative `Allocates` effect | Resolved for local declarations, MIR origin, call propagation, and `NoAllocation` proof projection; module/interface summaries remain unresolved |

The allocator itself is still not implemented. Cross-module generic and effect
authority must exist before R6e can honestly consume this substrate.

## Decision

The current frozen EVT1 language is insufficient to implement the requested
reference allocator framework as an ordinary reusable Concept library.
This is feasibility category C, not A or B.

A fixed-buffer bump-carving experiment could be written with the existing
`Span<byte>` and `Subspan` operations. That experiment would not establish the
requested library, because the remaining gaps cannot be closed by one minimal
library-facing primitive. Implementing through compiler-known allocator types,
name-based effect inference, or backend-only `malloc` helpers would create the
fake behavior R6c explicitly rejects.

## Baseline

- Baseline: `79d59cdc38782d4d14a0762f111540347ec23b7d`
- Compiler: `concept-evt1-stage0-go`
- R6a: `81b1fc41003130d28b6f2ae4d84e3de71bbdddfa`
- R6b: `79d59cdc38782d4d14a0762f111540347ec23b7d`
- The worktree was clean at audit start.
- `go test ./...`, `go vet ./...`, root `zig build test`, and
  `legacy/poc3-zig/zig build test` passed.

## Architectural inventory

| Capability | Current authority | Allocator consequence |
|---|---|---|
| Raw address / pointer | Unsafe `T*`, address-of, and dereference exist | There is no canonical `RawAddress`, checked pointer arithmetic, or raw-region construction API |
| Byte and fixed storage | Fixed `byte<array>[N]` storage exists | Caller-owned fixed storage is available |
| Layout | Named semantic layouts expose `LayoutSize`, `LayoutAlign`, and `LayoutOffset` | There is no generic `SizeOf<T>` or `AlignOf<T>` for typed allocation |
| Bind | Arrays and ndarrays can share established storage | It cannot bind an allocator-provided raw region to an arbitrary `T` owner |
| Ownership and Drop | `owned T`, explicit move, and concrete `Drop(owned T)` witnesses exist | Concrete local ownership works, but a reusable applied `Allocation<T>` owner cannot be declared |
| References and provenance | `ref`, `ref const`, `scoped`, ref structs, and bounded result provenance exist | Views can preserve an existing source lifetime; there is no allocation-lifetime source to bind |
| Span | `Span<T>`, `ReadOnlySpan<T>`, and `Subspan` are non-owning views | Fixed-buffer carving is possible, but Span supplies no ownership or release authority |
| Templates | One constrained type parameter on function templates | User-defined generic structs/classes and allocator policy types are not supported; templates cannot invoke templates |
| Concepts/interfaces | One type parameter with operation and compiler-analysis requirements | Contracts can describe operations, but do not currently attach an allocation effect to the selected operation |
| Operation summaries | Local call traversal and missing-summary handling exist for `NoAllocation` | There is no explicit authoritative `Allocates` declaration or verified contract summary |
| Core library composition | Core imports and multi-module compilation remain deferred | Separate `library/memory/*.concept` files cannot form a reusable compiled library |
| Hosted storage | The strict-C11 generator is bootstrap evidence; stable C ABI/FFI remains deferred | An ordinary Core `HostedMemorySource` cannot call an admitted `malloc/free` adapter |

## Blocking evidence

### Generic policy and owner types

The parser accepts only constrained function templates:

```concept
template <typename T>
requires SomeConcept<T>
T Operation(...)
{
    // ...
}
```

`parseTemplateDecl` immediately expects `requires` and then parses a function.
`validateKnownType` accepts applied runtime types only for compiler-owned
families such as `Result`, `Option`, `Async`, `Inference`, and Span. An ordinary
`Allocation<Item>` or `PoolAllocator<Item>` is rejected as an unknown type
application. Adding special cases for allocator names would violate the
ordinary-template requirement.

This blocks all of the following requested laws together:

- `Allocation<T>` as movable-only release authority;
- `PoolAllocator<T>` slot geometry and typed reuse;
- `BumpAllocator<Source>` and `ArenaAllocator<ParentAllocator>` composition;
- generic Drop ordering of `T` before region release.

### Generic type geometry

EVT1 layout queries currently describe named `layout` declarations. They do
not provide generic runtime-type `SizeOf<T>` and `AlignOf<T>`. Typed allocation
therefore cannot mechanically request the correct region geometry, and a pool
cannot derive slot size/alignment without duplicating compiler layout rules.

### Reusable library boundary

The requested semantic file layout presumes reusable Core library composition.
General Core imports and multi-module compilation are explicitly deferred.
Copying definitions into every specimen would test a pasted program, not a
canonical allocator library.

### Hosted source boundary

The active Core line has no settled external C function/ABI surface. Emitting a
backend-only `malloc/free` path for a source-named operation would make the
backend invent allocation behavior absent from typed MIR. That is not an
ordinary `HostedMemorySource` adapter and would normalize hidden allocation
into the compiler.

### Allocation effect authority

R6b's `NoAllocation` proof follows uniquely resolved local calls, proves a
known local leaf because there is no allocation-capable construct, and leaves
opaque calls unknown. No source or concept contract currently identifies an
operation as `Allocates`. This gap is independently tractable, but adding an
unused allocation attribute without an implementable allocator library would
not complete R6c and would prematurely create a new effect surface.

## Why this is not category B

A category-B correction must be one smallest missing library-facing primitive.
R6c instead requires coordinated new law in at least these independent areas:

1. user-defined generic runtime types and their instantiation/lowering;
2. generic type layout queries;
3. ownership, provenance, and Drop for applied generic owners;
4. reusable Core module/library composition;
5. an explicit hosted external-memory adapter boundary; and
6. semantic operation-effect contracts and propagation.

Those changes span grammar, type identity, validation, MIR, C lowering,
ownership cleanup, provenance, modules, and ABI policy. Calling that set a
minimal allocator correction would reopen the R5 freeze under a library label.

## Required prerequisite decision

Before allocator implementation resumes, a post-freeze proposal must choose
and prove the smallest general, non-allocator-specific foundation for:

- user-defined generic structs/classes with ordinary specialization;
- generic `SizeOf<T>` and `AlignOf<T>`;
- applied-type ownership/Drop/provenance;
- reusable Core library compilation;
- explicit external operation admission for hosted adapters; and
- operation semantic summaries, initially `Allocates`.

The proposal must keep allocator names out of these mechanisms. Once those
general seams exist, R6c can implement `MemoryRegion`, sources, bump/pool/arena
policies, typed allocation, and proof fixtures as ordinary library code.

## Preserved direction

The eventual allocator law remains:

```text
Allocation is library policy over an explicit memory source.
Allocator does not mean heap.
```

This audit adds no `new`, `delete`, global allocator, hidden heap, GC, allocator
grammar, name-based effect inference, or compiler-special allocator type. The
existing R6b `NoAllocation` Proven/Unknown behavior remains authoritative until
an explicit allocation operation summary can be attached to real operations.
