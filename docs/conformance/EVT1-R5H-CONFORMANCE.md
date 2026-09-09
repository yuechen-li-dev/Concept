# EVT1 R5h conformance

Baseline: `46b3f4c7a26e90496a40587be8548c007dbfd34b`
(`concept-evt1-stage0-go`).

R5h composes the implemented R4k interface/witness/dyn system with the R5f/R5g
inline async machine system. It creates no second satisfaction engine, async
vtable, task object, scheduler, executor, heap fallback, or coroutine ABI.

## Semantic surface

`requires async T Method(ref U self);` and its `asynchronous` alias normalize
to the ordinary callable result `Async<T>`. Struct and class methods satisfy
the requirement by exact normalized receiver, parameter, and return matching.
Sync/async mismatches reject without implicit wrapping or unwrapping. Static
templates call the concrete constructor directly; sync functions may construct
and return async work without awaiting it.

An explicit borrowed `dyn I` call uses the existing witness table to select the
concrete constructor once. The returned movable-only operation stores concrete
generated frames and is then driven or awaited through ordinary
Step/Complete/Result and child-push semantics. The witness is not consulted on
later Steps. Receiver constness and provenance are preserved; an operation
cannot escape local or scoped dyn backing.

## Corpus and diagnostics

The canonical corpus contains 19 accepted and 10 statically rejected readable
sources under `language/evt1/async/interface`. It covers struct/class satisfaction,
static async and synchronous-start templates, construction, immediate await,
`await ... ?`, manual driving, async void, mixed and composed interfaces,
compiler facts, inference, scoped/const receivers, Span/tensor state, aliases, and static
versus dyn equivalence.

Rejected cases cover sync/async mismatch, wrong eventual return, receiver
constness, private capability, open generic runtime shape, receiver escape,
mutable use through const dyn, active-operation copy, and invalid witness
construction. Existing `CV4155`/`CV4156`, interface, dyn-const, and async-copy
families remain authoritative. R5h adds focused families for open generic async
methods, dyn async receiver escape, malformed async witness MIR, and invalid
result-transfer MIR.

## MIR, Planner, and strict-C11 evidence

Witness MIR records normalized `Async<T>` signatures, eventual types, and
concrete `Method#async#machine` identities. Dyn-call MIR records
`AsyncConstructor`, `ExactlyOnce`, `MoveOnce`, and the existing no-copy,
no-allocation, and no-ownership-transfer dispatch facts. Malformed witness and
dispatch plan mutations reject.

`DispatchPlan` remains `WitnessIndirect` and records the `Async<T>` return. The
concrete method and async caller retain ordinary `AsyncPlan` records with
`MachinePush`, `Scheduler: None`, and `SavedPC: None`. Repeating full MIR and
plan generation 100 times is byte-identical.

Strict-C11 native execution covers all 19 accepted specimens.
Generated witnesses use `concept_async_operation (*Method)(void*, ...)` and
static const tables. The immediate-await specimen contains exactly one
`witness->Read` constructor call, one local returned operation, and one adoption
into the existing stack. Static and dyn paths produce the same observable value
and name the same concrete async machine; only constructor dispatch differs.
The generated C scan rejects allocation, threads, executors, event loops,
coroutine builtins, promise/future frameworks, RTTI, and per-object vtables.

## Boundary and recommended R5i

R5h does not add owning dyn, public ABI stability, open generic runtime methods,
general overload resolution, cancellation, timeout, channels, race/select,
OS-I/O policy, heap futures, generalized function objects, closures, or LIR.
Large operations currently use correct internal by-value C lowering; optional
caller-provided placement is future Planner work.

Recommended R5i is a bounded non-owning callable/callback value with an explicit
capture list, fixed witness shape, and ordinary provenance. It should prove a
second consumer for the same callable mechanics while forbidding inferred
closures, owning erased captures, hidden allocation, and scheduler policy.
