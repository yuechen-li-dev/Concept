# EVT1 reference and span direction

Status: R4b relational lifetime foundation; active rules in the language
specification are normative.

R3 establishes `ref T` and `ref const T` as explicit, non-owning aliases of
existing places. A mutable reference permits the mutations already permitted
by the referent type and place. A const reference is a read-only view. Neither
form copies or transfers the referent, extends a temporary lifetime, or proves
a general lifetime relationship.

The intended later vocabulary is:

```text
ref T
ref const T
ref struct
scoped ref T
Span<T>
ReadOnlySpan<T>
```

R4a activates `ref struct` for lifetime-bound aggregate values and `scoped`
for an explicit non-escape constraint. The compiler tracks bounded lexical
provenance (`Local`, `Parameter`, static/global where available, or `Unknown`),
derives a ref struct bound from its shortest-lived referenced field, and
conservatively rejects uncertain outward flow. This is deliberately not a
global borrow checker and introduces no named lifetimes or alias-exclusivity
solver.

R4b adds compact result provenance without adding lifetime syntax. Selected
function bodies summarize a lifetime-bound result as deriving from one
parameter, the shortest of a finite parameter set, a static source, or
unknown. At each call, the summary is instantiated from actual argument
provenance. Function boundaries therefore preserve rather than reset bounds,
and scoped inputs remain scoped through derived results.

`Outlives(source, result)` is a demand-driven semantic relation over the
parameter and result of a concept-required operation. It succeeds only from a
proven summary relationship. A different source disproves it and unknown
provenance rejects it conservatively. The proof and its subjects are retained
in MIR, but no runtime lifetime object is emitted.

`Span<T>` and `ReadOnlySpan<T>` are intended to be lifetime-bound borrowed
contiguous views, not owning collections and not hidden heap abstractions.
Their construction, variance, slicing, storage, call, return, and escape rules
must be designed against the active reference model before implementation.
In particular, future span safety depends on source-derived call results,
`Outlives`, scoped/non-escaping constraints, and ref-struct lifetime bounds.

The retired PoC3 `Slice<T>` fixtures remain useful pressure for pointer-plus-
length representation, bounds checks, and read-only access. They are not a
surface or lifetime contract for EVT1. R3 therefore marks `Slice<T>` for
redesign and does not port Slice or Span. R4b establishes the relational
machinery those future views require; it does not implement
`Span<T>`, `ReadOnlySpan<T>`, `stackalloc`, or runtime `Slice<T>`.
