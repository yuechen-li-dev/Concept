# EVT1 reference and span direction

Status: R4g Span and ReadOnlySpan are implemented bounded EVT1 semantics; the language specification is normative.

R3 establishes `ref T` and `ref const T` as explicit non-owning aliases of
existing places. R4a adds ref-struct lifetime bounds and `scoped`; R4b adds
source-derived call-result summaries and demand-driven relational `Outlives`
proof. R4g composes those authorities rather than adding a second borrowing
system.

The active bounded vocabulary is:

```text
ref T
ref const T
ref struct
scoped ref T
Span<T>
ReadOnlySpan<T>
```

`Span<T>` is a mutable bounded borrowed contiguous interval.
`ReadOnlySpan<T>` is its readonly counterpart. The canonical source forms are:

```concept
Span<int> values = Span(storage);
ReadOnlySpan<int> readonly = ReadOnlySpan(values);
Span<int> middle = Subspan(values, offset, length);
```

The explicit `ReadOnlySpan(mutableSpan)` constructor is the one-way
mutable-to-readonly conversion. No reverse conversion or const stripping is
allowed. `Subspan` is a free function because the current EVT1 expression
surface has ordinary free-function calls but no general method-call facility.

R4e `bind` remains narrower: it associates an expected array/ndarray shape
with an entire contiguous source. It has no partial offset or length. Span
adds bounded half-open subregions without changing bind, storage, or lifetime
law. Arrays and ndarrays supply physical storage; R4f layout regions and stream
channels supply semantic region facts; Span consumes those facts.

The compiler-visible region payload is:

```text
element type
provenance
region identity
backing byte offset
relative element offset
length
byte extent
alignment
mutability
contiguity
```

Construction preserves the source provenance and region identity. Subspan
retains the same parent identity and narrows only the relative offset and
extent. Its interval is `[baseOffset, baseOffset + length)`. An empty interval
at the parent's end is valid. Alignment is derived conservatively from parent
alignment and element byte offset; arbitrary offsets never retain an unsafe
stronger claim.

Local, parameter, scoped, layout-region, stream-channel, and helper-result
paths all flow through the R4a/R4b provenance representation and existing
escape checks. A helper cannot launder a local or scoped backing lifetime.
No runtime lifetime object is emitted.

Span copies or moves only its non-owning descriptor. It never copies, moves,
owns, allocates, or drops backing storage. Viewing an immovable backing object
is valid because construction does not relocate it. Option/Result composition
uses the existing ref-struct carrier rule and preserves provenance through
success extraction.

The retired PoC3 `Slice<T>` fixtures remain pressure for pointer-plus-length
lowering, bounds checks, and readonly access. They are not an EVT1 surface or
lifetime contract. R4g supersedes their direction with Span/ReadOnlySpan; it is
not a mechanical Slice port.

R4g adds no named lifetimes, non-lexical lifetime inference, reborrow lattice,
mutable-alias solver, generalized noalias checker, raw-pointer constructor,
strided or multidimensional span, stack allocation, or owned dynamic vector.
