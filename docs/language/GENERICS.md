# Generic runtime types

Status: EVT1 R6d implemented local-compilation substrate

Concept uses modern-C++-recognizable spelling for ordinary monomorphized
runtime types and functions:

```concept
template <typename T>
struct Box
{
    T value;
};

template <typename T, usize Capacity>
struct FixedBuffer
{
    T<array>[Capacity] values;
};

template <typename T>
T Identity(T value)
{
    return value;
}
```

Applications such as `Box<int>` and `FixedBuffer<Widget, 128>` have stable
source identities and compile to one concrete representation per semantic
compilation unit. There is no runtime dictionary, type registry, reflection
table, specialization search, or SFINAE. Nested applications instantiate from
the inside out; recursive infinite instantiation is rejected with
`GENERIC_RECURSIVE_INSTANTIATION`.

Integer and `usize` non-type arguments are compile-time values. Generic class
methods reuse ordinary class/self rules after type substitution. Function
templates accept ordered type and dimensionless `usize` non-type parameters.
They may be unconstrained or use one named Concept application over any of the
ordered parameters:

```concept
template <typename T>
requires Movable<T>
T MoveThrough(T value) { return move value; }
```

Relational applications retain all arguments:

```concept
template <typename T, typename U>
requires Convertible<T, U>
U ConvertThrough(T value) { return Convert(value); }
```

During open checking the requirement supplies its recursively composed
operation closure. Concrete instantiation structurally substitutes every
application and operation argument and revalidates satisfaction. Group multiple
requirements in a named concept rather than a boolean expression.

Applied fields feed the existing structural copy, move, reverse Drop, and
provenance authorities. A generic lifetime-carrying aggregate is therefore a
`ref struct`, just like its non-generic equivalent; no lifetime parameter is
introduced.

R6e artifacts preserve these lists and instantiate them in the consumer without
source reparse. Partial specialization, variadics, template-template
parameters, and template metaprogramming remain out of scope.

## Structural substitution

R6m makes substitution recursive over the semantic type structure. Nested
applications, storage types, reference and ownership qualifiers, array and
ndarray elements/extents, address spaces, and callable signatures are rebuilt
from substituted children before canonicalization. There is no separate
`Result`, `Option`, `Storage`, or owner substitution rule.

Open applications remain symbolic in generic declarations and module
artifacts. Closed applications are cached under their concrete identity and
must contain no reachable template type parameter in their fields or callable
signatures. Thus `Result<Owner<T>, E>` closes to its concrete owner/error types,
and imported `Owner<Widget>` contains `Storage<Widget>` rather than stale
`Storage<T>`.
