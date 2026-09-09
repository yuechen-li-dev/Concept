# EVT1 Callable Capture Direction

Status: R5j exact type surface implemented.

## Authority law

```text
callable = code identity + explicitly declared environment
```

A closure is simply a callable with a capture environment; “closure” is not a
second language primitive. Capture is not implicit lexical magic. A callable
executes `with (...)` explicitly bound state, so an outer local, parameter, or
`self` that is absent from that list is inaccessible. There are no `[=]`, `[&]`,
default captures, implicit `this`, or ambient JavaScript/C#-style captures.

## Surface and binding

The sole R5i literal is `callback(parameters) [with (captures)] { body }`.
Because its generated concrete type is intentionally unspellable, canonical
storage uses C/C++-shaped `auto` or `const auto`. Historical `var` and inferred
`let` remain exact compatibility aliases and are not used in canonical examples.
Captures are `x`, `move x`, `ref x`, or `ref const x`. A bounded named copy
initializer, `field = expression`, exists to make source evaluation order
observable; move/ref aliases and fallible generalized construction are deferred.
Sources are evaluated exactly once from left to right and fields retain their
declared names and order.

Name resolution inside the body is locals, parameters, captures, then module
symbols. The implementation isolates the body from the lexical outer scope and
retains that scope only to diagnose `CALLABLE_CAPTURE_REQUIRED`. `self` is the
ordinary normalized receiver binding and must be captured as `ref self` or
another legal explicit form. Free functions and compile-time/type symbols are
code or compile-time identities, not runtime captures.

The three uses of `with` remain separate:

- record `with` is a copy-update expression;
- automata `with state` declares a named persistent execution environment;
- callable `with (...)` binds a callable environment.

## Ordinary semantic laws

Copy capture uses structural copyability and owns an independent field. Move
capture performs the ordinary initialized-to-moved transition and transfers the
Drop obligation. `ref` and `ref const` use the existing borrow, mutability,
region, and Outlives checks. Span, tensor views, dyn values, and scoped refs do
not receive callable-specific lifetime exceptions.

Callable copy/move behavior derives from the environment. A move-only field
makes the concrete callable move-only; an immovable field remains immovable.
Copied or moved field mutation requires mutable callable storage. A body move
marks the environment field moved and makes subsequent source-level invocation
invalid through the normal moved-state rule. Generated cleanup state prevents a
moved-out owned field from being dropped again; all remaining owned fields drop
once in reverse capture order. Ref fields never drop referents.

These facts do not define `Fn`, `FnMut`, or `FnOnce` families. Readonly access,
mutable access, and consumption are consequences of ordinary body operations
over ordinary environment fields.

## Concrete and erased representation

Each literal has stable source-order identities such as `Main#callback0` and
`Main#callback0#environment`. The concrete C representation is an inline struct
in capture order plus a statically named invoke function receiving the struct by
pointer. Concrete dispatch is direct and needs no witness.

`callback<P... -> R>` is a bounded erased callback. It contains a borrowed
environment pointer and a static invoke function pointer. MIR gives each
`(signature, concrete callable)` adapter a deterministic callback-witness
identity. It is the specialized analogue of dyn:

```text
dyn       = erased object ref      + interface witness
callback  = erased environment ref + invoke witness
```

Neither has per-object metadata or a registry. R5i erased callbacks are
non-owning. A concrete callable must first be stored, then explicitly borrowed;
an owning temporary is rejected rather than heap-boxed. The erased value keeps
the concrete environment's provenance and mutability requirement.

## MIR, facts, and planning

Callable MIR records the code/environment identities, signature, receiver
mutation/consumption, capture kind, type, provenance, mutability, ownership
effect, field and region identity, evaluation order, cleanup tracking, and
NoAllocation. Erasure adds an explicit callback witness. Validation rejects
missing or reordered fields and malformed witnesses.

`CallablePlan` preserves exact Generic C11 size/alignment and capture records,
selecting `InlineEnvironment`, `DirectCallable`, and `None` allocation.
Erased invocation selects `StaticWitnessIndirect`; the aggregate plan records a
non-owning `EnvironmentFunctionPair`. Planning consumes MIR and cannot repair or
invent capture semantics. R5i adds no LIR.

## Composition and boundaries

A callable live across `await` is an ordinary typed persistent frame field. Its
owned and borrowed contents keep their existing ownership/provenance. Dyn and
async-interface calls inside the callable retain their ordinary static witness
and generated-machine paths. No scheduler or async closure object is added.

Automata and callables share the explicit-state philosophy, but R5i does not add
syntax for spelling an unnamed concrete callable as a persistent machine field.
Such a field requires a later type-surface decision; no existential storage is
invented here.

Fallible capture construction is deferred because current named initializers
accept only ordinary non-fallible expressions. Generalized fallible initializers
would need to preserve left-to-right partial cleanup.

## Deferred direction

Owning erased callbacks require an explicit storage and lifetime policy. Heap
closures, small-buffer optimization, generalized callable reflection, variadic
callable ABI, async callback literal syntax, move/ref aliases, and generalized
fallible capture expressions remain outside R5j. Recommended R5k is final R5
semantic reconciliation and freeze, with no major new feature.

## R5j spellable exact types

`using Name = typeof(expression);` is the canonical C++-recognizable spelling;
`type` is compatibility-only. The transparent alias binds the existing
generated callable identity; it does not describe an environment schema and
does not allocate. `typeof` is module-scoped, unevaluated, and intentionally
not general reflection. `concept` keeps its constraint/interface meaning.

Exact aliases compose with inferred and explicit function returns, ordinary
fields, by-value and ref parameters, machine-persistent fields, Option/Result,
async frames, dyn captures, and Span captures. Same signature is insufficient
for assignment: code identity remains part of the type. Heterogeneous values
must explicitly borrow into the existing erased callback signature.

There is no `auto`/existential field, erased-to-concrete recovery, owning erased
field, heap fallback, SBO, RTTI, registry, or public callable ABI promise.
