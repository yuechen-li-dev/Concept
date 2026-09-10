# EVT1 generic concept constraint direction

Status: R6o implemented compiler substrate

## Authority

A Concept `concept` is a named semantic structure describing what must be true.
A generic declaration's `requires Foo<...>` clause stores and demands a concept
application; it is not a trait attached to template parameter zero. Application
arguments are ordinary structural types and may name any generic parameter,
repeat a parameter, combine several parameters, or combine generic and concrete
types.

During open-generic validation the declared application is an assumption of the
generic contract. Its recursively composed required operations are available to
exact call and member-call resolution with origin `GenericRequirement`. This
does not mutate a parameter into a nominal fake type and does not create a
runtime witness. At concrete instantiation the application and every required
operation signature are structurally substituted and checked again against the
existing static concept witness machinery with origin `ConcreteWitness`.

## Composition and identity

A single requirement uses `requires Foo<T>`. Related requirements are grouped
in a named concept:

```concept
concept ReadWrite<T>
{
    requires Readable<T>;
    requires Writable<T>;
}
```

Nested applications retain their arguments and proof path. Cycles use the
existing bounded cycle diagnostic. Diamond paths deduplicate identical
operation signatures by structural signature identity; incompatible
requirements remain distinct and exact-overload rules diagnose an ambiguous
use.

Required operation effects remain semantic authority. An operation licensed
by `requires compiler.Allocates(Operation);` marks the open requirement call as
allocating, survives the module payload, and is checked against the concrete
implementation. `NoAllocation` follows the instantiated operation rather than
guessing from its name.

## Closure invariant

A closed generic instance may contain no bound template parameter in its
constraint arguments, nested concept closure, required-operation return or
parameter types, function signature, or materialized aggregate structure. R6o
extends R6m's structural substitution and closed-type validator to those
requirement surfaces.

## Deliberate boundary

R6o adds no boolean constraint algebra, subsumption ordering, SFINAE, partial
specialization, variadics, runtime witnesses, dynamic dispatch, reflection, or
allocator-specific compiler recognition.
