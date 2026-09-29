# EVT1 R5j Conformance — Exact Concrete Callable Type Storage

Status: implemented and qualified against the Go bootstrap, Generic C11
backend, and both retained Zig suites.

## Baseline and scope

- Baseline HEAD: `e6558666d150b07cdedbad09d47de12427f8dd15`
- Compiler: `concept-evt1-stage0-go`
- Prior R5i corpus: 20 valid, 12 invalid
- R5j corpus: 18 valid, 8 invalid

R5j makes R5i's existing generated callable type source-usable without exposing
compiler-owned environment names. Canonical syntax is:

```concept
auto MakeScaler(int multiplier) { /* returns one callable literal */ }
using ScaleCallback = typeof(MakeScaler(1));
```

`type Name = typeof(expr)` is accepted only as a compatibility alias.
`concept` remains the constraint/interface declaration. The alias is
transparent rather than nominal and `typeof` is module-scoped,
compile-time-only, and unevaluated. A unique factory name queries its exact
callable return when reference arguments make a call expression inapplicable.

## Semantic and storage evidence

An inferred `auto` function return must converge to one exact callable identity.
Explicit alias returns call the same factory identity. Struct and class fields,
by-value parameters, `ref const` readonly invocation, `ref` mutable invocation,
Option storage, async uses, dyn capture, and Span capture all pass through their
ordinary type/ownership paths. Separate callable literals retain separate types
even with identical signatures and layouts.

The strong automata specimen creates an accumulator environment once from a
persistent `with state` reference. Its machine field is `MachinePersistent`,
has an 8-byte/8-byte-aligned inline environment, uses `DirectCallable`, has
`Allocation: None`, and records `ConstructOncePersistAcrossSteps`. Two `Step`
calls update the same external value. Copy/owned fields use structural
copy/move/Drop; machine completion uses existing reverse frame cleanup.

Lifetime-bound callable fields require a `ref struct`. Lifetime-bound machine
fields require an initializer derived from persistent automata state. Short
machine refs, unrestricted aggregate escape, mutation through `ref const`,
reuse after consuming invocation, erased-to-concrete recovery, distinct exact
identity assignment, existential `auto` fields, and invalid type queries have
stable rejection specimens.

## MIR, Planner, and C11

`MIRTypeAlias` exposes the unevaluated query, exact callable identity,
environment identity, size/alignment, inline storage, direct dispatch, and
no-allocation fact. `ConcreteCallableTypePlan` preserves those facts. Machine
storage repeats its concrete geometry and persistence policy. Validators reject
missing or changed alias/plan evidence.

Generic C11 dependency-orders generated environment structs with ordinary
aggregate declarations. Exact fields are emitted inline and exact parameters
use ordinary by-value/pointer ABI according to Concept ownership. Calls target
the statically named invoke routine. The generated header also emits a
deterministic C typedef for each source alias; this is not a public ABI promise.

The representative machine specimen produces byte-identical MIR, Plan, and C
across 100 runs. Generated C is scanned for `malloc`, `calloc`, `realloc`,
`std::function`, RTTI, type registry, closure boxes, and vtables; none is
present. All eighteen valid programs compile and execute as strict C11,
covering fields, all parameter modes, machine persistence, Option, owned
cleanup, async, dyn, and Span.

## Deferred and recommended R5k

Owning erased callbacks, heap boxes, SBO, generalized reflection, stable
callable ABI, arrays requiring new grammar, scheduler/cancellation/channels,
and LIR remain deferred. Recommended R5k is final R5 semantic
reconciliation/freeze—diagnostic, documentation, and cross-feature consistency
only, with no major new feature.
