# Concepts

R7f2 adds the operation contract `requires compiler.InvalidatesBorrows(Operation, resourceParameter);`. It describes possible lifetime invalidation through a reference parameter; ordinary mutation has no such implication. See [destructive resource effects](../design/EVT1-DESTRUCTIVE-RESOURCE-EFFECTS.md).

R7f1: An operation requirement may itself have `template <typename U>` parameters. These are scoped within that requirement and structurally matched to a concrete generic function. Multi-type-parameter concepts already support value and reference parameters; non-type concept parameters remain unsupported. See [generic library closure](../design/EVT1-GENERIC-LIBRARY-CLOSURE.md).

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

## Compiler-owned concepts

Compiler-owned concepts are namespaced semantic propositions understood by
the compiler but used through ordinary source machinery. The initial
synchronization family is directly stateable in concept bodies:

```concept
concept TemperatureAccess<TAgent, TSlot>
{
    requires sync.ExclusiveWriter<TAgent, TSlot>;
    requires sync.SynchronizedAccess<TSlot>;
}
```

Concrete module requirements may declare synchronization authority. Inside a
generic or named concept, the same spelling is an obligation that must be
established by the concrete caller or context; it is not an unsafe global
assumption. Missing evidence is `Unknown`, known conflicting authority is
`Disproven`, and matching authority is `Proven`. Existing `Disjoint` is reused
without a second namespaced disjointness system.

R7d4 can independently derive writer, producer, and consumer cardinality.
Execution entries use `[[execution_context(ContextType)]]`; operations whose
channel meaning is not intrinsic use
`[[semantic_access("Publish", parameter)]]` and the other closed access-kind
spellings. These annotations state semantics explicitly. A function named
`Publish`, `Read`, or `Commit` has no special meaning by spelling alone.
## Generated witnesses

A generated function can satisfy a concept's required operation through the same lookup and signature rules as a handwritten function. `concept explain <file> --generated <operation>` displays the generated origin and reflected inputs for that checked operation. `concept explain <file> --concept 'Trace<Node>'` checks a single-argument concept goal and attributes its generated required operation.
