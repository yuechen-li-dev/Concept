# EVT1 structural generic substitution direction

Status: R6m implemented

Generic substitution is one structural operation over the complete `Type`
record. A bound template leaf is replaced while ownership, const, scoped,
imported, and unsafe qualifiers are preserved. The traversal then visits
pointer/reference bases, array and ndarray elements, ordinary applied-type
arguments, callable parameters, and callable results. Non-type bindings use
the same recursive shape for application arguments, array extents, ndarray
shape expressions, and callable children.

No carrier or library spelling participates. `Result`, `Option`, `Storage`,
`Address`, `MemoryRegion`, user owners, and deeper wrappers all close because
they use the same applied-type child list.

## Open and closed applications

An application containing a template parameter remains an open application.
It is not entered into the concrete-instance cache and is not flattened into a
nominal string. Generic struct construction retains both its applied `Type`
tree and its eventual nominal lowering name. Field validation and semantic
fact derivation can obtain a temporary structural view of an open generic
declaration without materializing it.

Once every argument is concrete, the existing canonical identity and instance
cache materialize the declaration. A closure check rejects any reachable
template type left in concrete fields, method signatures, function parameters,
or function results before executable validation and MIR lowering.

Nominal struct declarations are the cycle boundary. Substitution walks finite
type-expression children and never recursively expands a declaration through
its fields. Existing generic-instantiation recursion detection therefore
continues to diagnose true infinite application expansion.

## Semantic artifacts and summaries

`concept-module.v1` already serializes the typed generic declaration and body.
R6m preserves applied construction structure in that payload. The consumer
performs the same structural substitution and canonical instantiation without
source reparse. Field-sensitive value summaries are derived against structural
generic views, so `Initialized` storage facts survive an owner returned through
`Result` and an imported factory.

## Generator decision

OctGen was inspected before implementation. Its current external model is a
small, host-owned generator for specific declared records, not a Go type-algebra
schema or general visitor generator. The Concept `Type` algebra is one compact
record rather than a large sum type. A checked-in generator would duplicate the
schema and obscure the closure rules, so R6m uses a small handwritten central
mapper. Tests exercise every current composite child and fail when a reachable
parameter survives a closed instantiation.

Partial specialization, variadics, template-template parameters, generic
reflection, allocators, source pointers, and new module semantics remain
deferred.
