# EVT1 R5i Conformance — Explicit Callable Capture Environments

Status: implemented and qualified against the Go bootstrap, Generic C11 backend,
and both retained Zig suites.

## Baseline and scope

- Baseline: `d225f534f00ba50020ba6e28cbbe125648bf5327`
- Compiler: `concept-evt1-stage0-go`
- Prior R5h corpus: 19 valid, 10 invalid
- R5i corpus: 20 valid, 12 invalid

R5i adds one callable literal, four canonical explicit capture forms, concrete
inline callable values, and bounded non-owning erased callbacks. It adds no heap
closure, owning erasure, scheduler, executor, RTTI, registry, GC, or LIR.

## Surface and semantic result

`callback(parameters) [with (...)] { body }` is canonical. Generated concrete
types use `auto name = ...` or `const auto name = ...`; `var` and `let` are
compatibility aliases covered by a focused parser/generation test, not canonical
R5i corpus spelling. Empty literals omit `with` or use `with ()`. Captures are
`x` (copy), `move x`, `ref x`, and
`ref const x`. The bounded named copy initializer `field = expression` supplies
native exactly-once/left-to-right evidence. Move/ref initializers remain
identifier/place forms.

Outer locals, parameters, and `self` are inaccessible unless explicit. Free
functions and module/type/comptime symbols require no capture. Duplicate and
unknown sources reject. Captures are ordinary bindings, so structural copy,
moved state, const mutation, provenance, Outlives, Span/tensor backing, dyn
provenance, and Drop remain owned by their existing analyses.

The concrete identity is `<function>#callback<n>` and the environment is
`<function>#callback<n>#environment`. Fields retain capture order. Generic C11
geometry is recorded exactly for the supported representations, including the
one-byte empty C struct, pointer/descriptor pairs, and ranked tensor views.
Concrete invocation is a direct static call over the environment pointer.

Callable copyability and Drop derive structurally. A move-only capture makes the
callable move-only. Owned fields drop in reverse capture order; refs do not drop
referents. A moved-out owned field has explicit live-state in the inline
environment, suppressing only that field's later Drop. Mutation or consumption
of environment state requires mutable callable storage. Source validation uses
the ordinary moved-state diagnostic for subsequent invalid invocation.

## Erased callback

`callback<P... -> R>` is `{ environment pointer, invoke pointer }`. Construction
requires `ref` to named concrete callable storage. It neither copies nor owns the
environment. Signature and invocation mutability are checked exactly; returned
local erasure and owning temporary erasure reject. One deterministic
`MIRCallbackWitness` is emitted for each `(signature, concrete callable)` pair,
with no per-instance metadata.

## MIR and Planner evidence

`MIRCallable`, `MIRCaptureEnvironment`, and `MIRCaptureBinding` expose code and
environment identity, capture kind/type/provenance/mutability/ownership, field
and region identity, cleanup tracking, evaluation order, geometry, direct
dispatch, and `NoAllocation`. `MIRCallbackWitness` exposes exact signature,
adapter, and borrowed environment identity. Malformed capture order and witness
records reject validation.

`CallablePlan` selects `InlineEnvironment`, `DirectCallable`, and `None` while
preserving size, alignment, fields, mutability, and consumption. Capture and
callback operations select `CaptureEnvironmentPlan`/`InlineEnvironment` and
`CallbackDispatchPlan`/`StaticWitnessIndirect`. Callback aggregate evidence is
`EnvironmentFunctionPair`, `NonOwning`, and `NoAllocation`. Plan mutation rejects
with `PLAN_CALLABLE_INVALID` or the existing artifact validator.

## Composition evidence

The corpus executes copy/move/ref/ref-const and copied mutable environments;
explicit `self`; scoped refs; Span, rank-one tensor view, and dyn capture;
concrete callable liveness across await; erased local and argument calls; and an
R5h async dyn-interface call followed by callback invocation. The async frame
holds the concrete callable as an ordinary persistent typed field. R5i does not
add async callable literal syntax. Concrete callable machine-field spelling is
deferred because EVT1 cannot name a literal's generated type; no existential
storage was introduced.

The required `callback_static_ref_return.concept` filename exercises the
implemented static case available today: a callable reaches a free-function code
symbol without capture. EVT1 has no runtime global-storage declaration surface,
so a positive returned `ref globalState` capture would be fictional and remains
deferred with that surface.

## Diagnostics

The canonical invalid corpus pins:

- `CALLABLE_CAPTURE_REQUIRED`
- `CALLABLE_CAPTURE_DUPLICATE`
- `CALLABLE_CAPTURE_COPY_NONCOPYABLE`
- `CALLABLE_CAPTURE_LIFETIME_INVALID`
- `CALLBACK_OWNING_TEMPORARY_REJECTED`
- `CALLABLE_CONST_MUTATING_INVOKE`
- ordinary `CV4502` moved-state and `CV4513` readonly-mutation diagnostics

Additional structural tests pin `CV4501` for copying a move-only callable,
`CALLABLE_MIR_INVALID` for reordered capture MIR, and
`PLAN_CALLABLE_INVALID` for invented heap allocation.

## Native, determinism, and runtime evidence

All 20 valid sources compile under strict C11 with warnings as errors and run to
their expected result. The evaluation-order specimen observes `1` then `2`, each
once, observes that construction does not invoke the body, and observes exactly
one later invocation. The ownership shape test pins an explicit live bit, its
initialization, clearing on move-out, and conditional final cleanup.

The representative async/dyn/callback specimen generates byte-identical MIR and
Plan across 100 runs. Stable callable and callback-witness identities are part of
those artifacts. Generated C is scanned for `malloc`, `calloc`, `realloc`,
closure heap, GC registration, RTTI, task runtime, scheduler, executor, and
hidden copy box; none is present.

## Deferred and recommended R5j

Fallible capture construction/partial cleanup, move/ref aliases, owning erased
callbacks, heap or small-buffer storage, variadic ABI, async callback literals,
and generalized callable reflection remain deferred. Recommended R5j is a
bounded callable type surface for explicitly named concrete storage in fields
and parameters, with an automata machine-field proof and no existential or heap
erasure.
