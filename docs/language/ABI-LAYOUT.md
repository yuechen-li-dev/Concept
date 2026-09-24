# C ABI values and layout (R7j)

Concept does not infer foreign layout from intent. Native ABI claims are explicit and, where possible, verified against the selected external compiler.

Use `[[repr(C)]]` on a nonempty `record struct` whose fields have supported fixed C value representations. Field order is declaration order. Natural field alignment determines offsets and tail padding for the selected target. `repr(C)` describes native memory and calling representation; it is not a packed format, a portable wire format, or a claim about arbitrary C++ classes. Octagon is the semantic data exchange format.

```concept
[[repr(C)]] record struct Pair { int x; int y; }
extern "C" Pair MakePair(int x, int y);
extern "C" Pair RoundTripPair(Pair value);
static_assert(SizeOf<Pair>() == 8, "pair size on the selected target");
static_assert(AlignOf<Pair>() == 4, "pair alignment");
static_assert(OffsetOf<Pair>(Pair.y) == 4, "typed field identity");
```

The ordinary proposition `concept CAbiValue<T> { requires compiler.CAbiLayout<T>(); }` explains representation eligibility. It checks `repr(C)`, each field, fixed extent, and absence of destruction authority. It does not claim native measurement by itself. The native project ABI probe supplies that separate evidence. `concept explain` reports the compiler analysis and its blockers.

Nested `repr(C)` records and fixed arrays are admitted. The C11 backend emits ordinary C structs and passes aggregates by value in prototypes and calls; the external C compiler selects register or hidden return conventions. A C pointer field is currently rejected from `repr(C)` value structs because its foreign handle and nullability contract have not been represented. Opaque pointer parameters already supported by the scalar FFI remain separate. C enum representation, packed structs, explicit alignment overrides, unions, bitfields, flexible arrays, function pointers, and long double are deferred. Bool fields are rejected until representation is verified.

`repr(C)` answers how a value is represented. Foreign concepts answer what the operation or value means. The two contracts are separate. In particular, memory layout does not infer ownership, nullability, or lifetime semantics.
