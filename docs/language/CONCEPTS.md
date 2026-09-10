# Concepts

Concept concepts are named semantic requirement structures. They can describe
types, operations, fields, lifetimes, regions, effects, foreign boundaries, and
relationships among generic arguments.

```concept
concept Convertible<TSource, TDestination>
{
    requires TDestination Convert(TSource value);
}
```

An application such as `Convertible<Source, Destination>` is a proposition. At
a concrete `requires` declaration it must be proven. On a generic declaration
it is assumed while validating that declaration and revalidated after concrete
substitution.

Concept bodies may contain multiple `requires` entries and may compose other
concepts. The canonical source style is one named requirement on a generic
declaration; define another named concept to group related requirements. The
language does not use `&&`, `||`, or `!` constraint expressions.

C++20 concepts primarily constrain templates. Concept concepts have the
broader semantic role above; R6o completes the recognizable generic-template
use case within that existing model. Satisfaction remains compile-time static
proof. An `interface` is the deliberately narrower dyn-compatible concept form
and does not turn ordinary concept constraints into runtime dispatch.

`Standard.Memory` is R6p's concrete proof of this model.
`ReleasableAllocator<A>` composes `Allocator<A>`; typed allocation and generic
Drop use required operations for the second template parameter. Imported
effects remain structural artifact data, with no runtime witness or
allocator-specific dispatch.
