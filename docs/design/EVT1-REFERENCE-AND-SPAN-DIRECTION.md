# EVT1 reference and span direction

Status: R3 directional note; only the active `ref` rules in the language
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

`ref struct` is reserved for lifetime-bound aggregate values in a later
milestone. `scoped` is intended to make non-escape constraints explicit where
local inference is insufficient. Neither spelling is active in R3.

`Span<T>` and `ReadOnlySpan<T>` are intended to be lifetime-bound borrowed
contiguous views, not owning collections and not hidden heap abstractions.
Their construction, variance, slicing, storage, call, return, and escape rules
must be designed against the active reference model before implementation.

The retired PoC3 `Slice<T>` fixtures remain useful pressure for pointer-plus-
length representation, bounds checks, and read-only access. They are not a
surface or lifetime contract for EVT1. R3 therefore marks `Slice<T>` for
redesign and does not port Slice, Span, reference-containing aggregates,
stack allocation, allocators, arenas, or runtime collections.
