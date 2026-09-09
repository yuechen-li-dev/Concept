# EVT1 class and interface direction

Status: R5h async member composition implemented over the R4k narrow model

## Taxonomy

```text
struct           ordinary data/value aggregate
record struct    immutable data/value aggregate
immovable struct storage-bound data/value aggregate
class            fields, methods, and public/private boundaries
concept          generalized compile-time contract and proof system
interface        dyn-reifiable specialized concept
dyn Interface    borrowed erased reference plus runtime witness
template         static specialization mechanism
```

A class is an encapsulation binder, not a storage policy. It has the same
ordinary local, field-derived copy/move/Drop, and explicit ownership laws as
other Concept aggregates. It adds method grouping and two visibility sections.
Class members default private; struct members default public.

Methods normalize to ordinary functions whose first parameter is an explicit
receiver. The default receiver is `ref Class self`; authors may spell
`ref const Class self` for readonly access. `value.Method(args)` inserts that
receiver, and an unambiguous bare member in a method body means `self.member`.
Concrete calls are direct and static.

An async struct or class method uses the same receiver normalization, access
control, and aggregate storage law. Its callable result is `Async<T>`, so it
may satisfy an async interface operation without an `implements` declaration
or class-runtime change. Sync and async callable results must match exactly.

An interface reuses concept requirements and satisfaction. Its public method
requirements begin with an erased `ref T self` or `ref const T self`; concrete
field requirements synthesize runtime accessors. `requires Other<T>` composes
contracts without imposing base-object layout. Templates consume the same
contract statically, while explicit `dyn I = ref place` construction reifies a
static `(I, concrete type)` witness for runtime dispatch.

## Deliberate exclusions

R4k has no inheritance, protected or friend access, abstract class, virtual or
override keyword, constructor/destructor subsystem, implicit reference
semantics, hidden heap, object header, GC, RTTI, reflection registry, downcast,
or owning dyn. Drop remains the only lifecycle mechanism. A future owning dyn
must first choose explicit erased storage and allocator policies.

## Planner boundary

R4l classifies ordinary class/struct method calls as direct calls. Runtime dyn
method or field access is planned separately from interface satisfaction as
witness-indirect dispatch or a witness field accessor. The Planner consumes
the deterministic witness identity already present in MIR and cannot add
inheritance, RTTI, allocation, or a per-object vtable.

## R5i receiver capture

The normalized method receiver `self` is an ordinary lexical binding for
capture purposes. It is never implicitly captured. A callable that needs object
state captures `ref self` or `ref const self` and then uses `self.field` or a
method call. An unbound `field` capture never expands to `self.field`. Existing
receiver constness, provenance, and interface dispatch rules remain
authoritative inside the callable environment.
