# EVT1 R6f conformance: reference allocator feasibility

Status: honest stop (task end state C)

Baseline HEAD: `1cda266f5e2bafad936764651a37d7671e330be0`

Compiler identity: `concept-evt1-stage0-go`

R6c commit: `8cebcd97c5e4419e0bcb16573bae7b69b60f7cf4`

R6d commit: `1f16339b6ab228fce12866b3fade0d451b921f66`

R6e commit: `1cda266f5e2bafad936764651a37d7671e330be0`

## Baseline

The worktree was clean at audit start. The baseline passed:

- `go test ./...`;
- `go vet ./...`;
- root `zig build test`;
- `legacy/poc3-zig/zig build test`.

The active module fixtures use exact roots beneath
`language/evt1/tooling/modules`, generic type constraints use
`template <typename T> requires Concept<T>`, layout queries are
`SizeOf<T>()` / `AlignOf<T>()`, external functions use `extern "C"`, and
allocation effects use `requires compiler.Allocates(Operation);`. R6e artifacts
preserve imported generic ownership, reverse structural Drop, provenance, and
operation summaries.

## R6c blocker reconciliation

| R6c checklist item | Current status |
|---|---|
| User-defined generic runtime types | Resolved by R6d |
| `SizeOf<T>()` / `AlignOf<T>()` | Resolved by R6d |
| Generic ownership/provenance/Drop | Resolved by R6d and transported by R6e |
| Reusable modules | Resolved by R6e `concept-module.v1` |
| Hosted ABI seam | Resolved by R6d `extern "C"` |
| Operation effects | Resolved by R6d/R6e summaries |

The original checklist is closed. It was not sufficient: executable allocator
algorithms also require a general raw-storage manipulation boundary.

## Compiler evidence

Four minimal audit probes were checked with the real `concept check` path.

`usize` arithmetic:

```concept
usize AddSizes(usize left, usize right)
{
    return left + right;
}
```

Result: `CV4028`, because `usize` is not admitted by the ordinary scalar
arithmetic rules. Alignment rounding, bounds checks, and overflow-safe end
offset calculation therefore cannot be written with the required API type.

Raw-address arithmetic:

```concept
byte* Advance(byte* address, usize offset)
{
    return address + offset;
}
```

Result: `CV4028`. Pointer types are bounded ABI carriers, not a raw-storage
expression model.

Raw-address typed binding:

```concept
template <typename T>
ref T BindObject(byte* address)
{
    ref T value = bind address;
    return value;
}
```

Result: `CV4563`: `bind` requires a `ref` or `ref const` array/ndarray storage
target. It cannot establish a typed object lifetime over allocator-provided raw
storage.

Two-parameter function template:

```concept
template <typename T, typename TAllocator>
T Allocate(ref TAllocator allocator)
{
    return T{};
}
```

Result: `CV4020` at the comma because function templates require exactly one
type parameter.

Implementation inspection agrees with the executable diagnostics:

- binary validation admits selected identical scalar families but not `usize`
  or pointers;
- unary parsing/validation has no raw address-of or dereference operation;
- storage binding is restricted to established array/ndarray view targets;
- `parseTemplateDecl` requires exactly `template <typename T>`.

## Why the allocator was not approximated

A `MemoryRegion` containing an inert `byte*` would not support aligned carving,
pool identity, arena subregions, or typed initialization. A `Span<byte>`-only
experiment would preserve byte element semantics and cannot become arbitrary
`T` without a compiler shortcut. Backend-injected casts or allocator-name
lowering would violate the milestone's no-cheating rule.

Accordingly, no `Standard.Memory` placeholder module, allocator-specific
compiler branch, hidden malloc fallback, allocator MIR node, heap, GC, or
runtime registry was added.

## Outcome

R6f ends in feasibility category C. The exact next blocker is a general,
provenance-preserving explicit raw-storage region and typed initialization
boundary, plus general multi-parameter function-template support for the
canonical typed helper. Allocator implementation must remain stopped until
those are approved and proven independently of allocator names.
