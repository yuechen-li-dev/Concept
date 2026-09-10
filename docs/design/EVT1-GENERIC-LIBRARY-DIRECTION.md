# EVT1 generic library direction

Status: R6e reusable semantic module substrate complete

R6e resolves the R6d module blocker with deterministic `concept-module.v1`
artifacts. Generic types, functions, methods, non-type parameters, declaration
constraints, ownership/Drop, provenance, and operation summaries now cross an
acyclic import graph without source inclusion. Concrete instantiation still
occurs in the consumer and feeds the same layout, MIR, Planner, and C11 owners.

R6d is an approved post-freeze substrate extension required to enable ordinary
reusable libraries. It is not an opportunistic syntax expansion.

## Authority shape

```text
template declaration + compile-time arguments
    -> deterministic concrete applied type
    -> ordinary layout / ownership / provenance analysis
    -> typed MIR and strict-C11 evidence

operation Allocates contract
    -> local call-graph proof projection
    -> Proven / Disproven / Unknown NoAllocation result
```

Syntax is intentionally familiar to modern C++: `template <typename T>`,
`requires`, `Box<int>`, `SizeOf<T>()`, `AlignOf<T>()`, and `extern "C"`.
Semantics are Concept's: eager explicit constraint diagnostics, structural
ownership and provenance, no SFINAE, no header model, and no trait or lifetime
ceremony.

The implemented local substrate includes concrete generic structs/classes,
integer non-type parameters, nesting and caching, generic functions and
methods, shared-authority size/alignment queries, structural reverse Drop,
`ref struct` provenance, bounded C ABI validation, and local allocation-effect
proof propagation.

## R6f consumer result

R6e closed the reusable-module boundary. R6f then became the first attempted
ordinary library consumer to require explicit raw storage. It demonstrated
that generic/module transport is working but is not sufficient to implement an
allocator: Core still lacks checked `usize` arithmetic, raw-region slicing,
and provenance-preserving binding/initialization of a raw byte region as an
arbitrary `T`. Function templates also remain intentionally limited to one
type parameter, which prevents the canonical `Allocate<T, TAllocator>` helper.

Allocator work remains stopped at that general substrate boundary. No
allocator name, heap, GC, generic dictionary, reflection registry, package
solver, partial specialization, variadic template machinery, or generalized
effect algebra is introduced here.

## R6m structural closure

Open nested applications now retain their child type algebra in the semantic
payload. Consumers recursively substitute bound type and non-type parameters
before concrete canonicalization, and closed fields and callable signatures
are checked for unresolved parameters before MIR. Imported generic owners and
factories therefore use the same instantiation path as local declarations; no
carrier or library type name receives privileged handling.
