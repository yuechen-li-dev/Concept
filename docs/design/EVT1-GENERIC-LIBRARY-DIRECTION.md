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

## Convergence boundary

The next independent blocker is reusable semantic modules. The compiler has no
resolver that turns `import Standard.Generic` into a typed dependency, no
`concept-module.v1` artifact, and therefore no cross-module generic body or
effect-summary authority. Adding a fixture-copy convention would disguise that
missing owner and would not converge on R6d success.

Until that boundary exists, allocator work remains stopped. No allocator name,
heap, GC, generic dictionary, reflection registry, package solver, partial
specialization, variadic template machinery, or generalized effect algebra is
introduced here.
