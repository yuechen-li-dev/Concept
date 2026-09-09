# EVT1 R6g conformance

Baseline: `0f3f7d20df2618c983591c1460894f38d9badde6`

Compiler: `concept-evt1-stage0-go`

R6f commit: `0f3f7d2 Document EVT1 allocator storage boundary`

## Implemented and demonstrated

- normalized compile-time dimensions and exact unit scale;
- `bit`, `byte`, `m`, `s`, `K`, and `Hz = s^-1`;
- coherent quantity arithmetic and exact integral conversion;
- byte-qualified layout queries and complete systems-oriented `usize` parsing;
- Euclidean signed-modulo semantics pinned without exposing C remainder;
- affine `Address<Space>` identity and algebra;
- explicit address-bit operations and host-space guarded typed bind;
- ordinary imported `MemoryRegion<Space>`, `AlignUp`, and SystemMemory
  `Subregion` library code;
- `Storage<T>` Uninitialized/Initialized transitions, Initialize, Destroy, and
  ordinary Drop integration;
- ordered multi-parameter function templates with type and dimensionless
  `usize` arguments, deterministic identity, and cross-module instantiation;
- strict C11 erasure to numeric scalars, `uintptr_t`, structs, and bounded bind
  checks with no unit registry or allocator.

## Oct classification

A: exponent vectors, normalization, dimensional compatibility, coherent
multiply/divide/power, signed exponents, and derived Hz.

B: C++-recognizable `representation<unit>` spelling and Concept Information
scale for exact bit/byte conversion.

C: pixel/UI units, Celsius sugar, runtime unit registries, and broad domain
conversion catalogs.

## Legacy byte audit

The active compiler/language/docs search found 352 textual `byte` occurrences
across 53 compiler or Concept-source files at the R6g checkpoint. Active
Concept source uses fall into: legacy `byte<array>` raw cells (C), C ABI
`byte*` carrier declarations (D), and new qualified byte extents (B). Numeric
eight-bit intent migrates to `uint8` (A). R6g keeps the old scalar spelling as
an explicit compatibility alias and performs no blind rewrite; the migration
record prevents adding new ambiguity.

## Isolated boundary

R6g is Meaningful progression rather than full milestone Success. Existing
semantic facts attach region/provenance primarily to compiler-known layout,
Span, tensor, and lexical reference paths. They do not yet carry a general
origin plus interval through every ordinary `MemoryRegion` value, constructor,
call, and artifact boundary. Therefore trusted external region establishment,
known/disproven/unknown subregion disjointness, and proof-backed alignment facts
are not claimed. `bind<T>` accepts lexical storage-backed origins and rejects
arbitrary reconstructed addresses rather than faking provenance. Allocator R6h
remains blocked on general region-fact transport.

No allocator framework, source raw pointer, reinterpret cast, MMIO, volatile,
mapping, heap, GC, scheduler, or runtime registry was added.

## Qualification

- `go test ./...` passed, including focused R6g quantity, address, storage,
  module-artifact, 100-run generated-C determinism, and strict C11 native
  specimens;
- `go vet ./...` passed;
- root `zig build test` passed;
- `legacy/poc3-zig` `zig build test` passed;
- `git diff --check` passed;
- the changed compiler and ordinary Concept library code introduce no malloc,
  GC, RTTI, runtime registry, hidden pointer table, or scheduler.
