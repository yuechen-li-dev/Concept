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
templates may be unconstrained or use the existing single named Concept
constraint:

```concept
template <typename T>
requires Movable<T>
T MoveThrough(T value) { return move value; }
```

Applied fields feed the existing structural copy, move, reverse Drop, and
provenance authorities. A generic lifetime-carrying aggregate is therefore a
`ref struct`, just like its non-generic equivalent; no lifetime parameter is
introduced.

Current bounded edge: reusable cross-module template artifacts and constraints
on generic type declarations are not implemented yet. Partial specialization,
variadics, template-template parameters, and template metaprogramming remain
out of scope.
