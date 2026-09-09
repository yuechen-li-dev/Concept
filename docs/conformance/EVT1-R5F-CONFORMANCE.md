# EVT1 R5f conformance

Baseline: `fc050586725a1041c9423e6dd6a65552e2ea38e3`
(`concept-evt1-stage0-go`).

Inventory classification: **B, partial substrate existed**. R5e already owned
the bounded top-frame Step, child push/pop, explicit parent state, completion
outcome, cleanup, overflow, and result-gating laws. R5f adds source-level
automation and does not create another async runtime. No canonical async-like
surface was found in the active Concept/Vulkan compiler; Oct Remember/Resume
remains library-policy pressure rather than an async implementation.

## Corpus and diagnostics

The canonical corpus has 23 valid sources, six static-invalid sources, and two
runtime-negative sources. Static categories are:

| Case | Diagnostic |
|---|---|
| await outside async | `AWAIT_OUTSIDE_ASYNC` |
| await non-Async value | `AWAIT_REQUIRES_ASYNC_VALUE` |
| unprovable persistent reference | `ASYNC_PERSISTENT_REF_ESCAPE` |
| neutral completion of value async | `ASYNC_NEUTRAL_VALUE_MISMATCH` |
| copy active operation | `ASYNC_COPY_INVALID` |
| consume movable result twice | `ASYNC_DOUBLE_RESULT_CONSUME` |

The runtime-negative depth case terminates with `machine stack capacity
exceeded`; premature Result terminates with `result requested before
completion`. Malformed operation paths retain deterministic runtime checks in
the generated substrate.

## Inspectable proof

Async MIR preserves `F#async`, `F#async#machine`, `F#async#state`, eventual and
operation types, generated states, persistent fields, and each await's operand,
continuation, live set, source span, exactly-once evaluation, bounded push, and
exactly-once outcome consumption. MIR validation rejects a synthesized
scheduler.

`AsyncPlan` selects `GeneratedMachine`, `Inline`,
`ExplicitGeneratedState`, `MachinePush`, `Scheduler: None`, and
`SavedPC: None`. It exposes a 1024-byte aligned frame slot, capacity-eight
child depth, state count, await plans, persistent fields, and cleanup strategy.
Plan mutation is rejected before backend lowering.

GenericC11 emits fixed inline operation/frame/result storage, per-function
frame/init/step functions, explicit switch tags, static frame/result capacity
assertions, child push/adoption, pop/outcome transfer, and one-shot result
access. Native tests cover sequential/nested calls, a yielded child, control
flow, iteration, ref/owned/tensor/class state, `?`, manual operation driving,
and operand evaluation once. Generated outputs are scanned for heap,
scheduler/executor/event-loop/thread, saved-PC, setjmp/longjmp, and coroutine
framework markers.

## Manual equivalence specimen

`async_manual_equivalence.concept` places a hand-written R5e Parent/Child
machine stack beside an async Parent awaiting async Child. The manual path
requires three explicit Steps; the generated path is driven until Complete.
Both produce 42 in the same native process. This demonstrates that await is
generated continuation-state plumbing, not concurrent execution.

## Bounded omissions

R5f defers awaits in match arms, more than one await in a single simple
branch/loop body, async interface/dyn methods, general CFG normalization,
schedulers, OS I/O, cancellation, timeouts, select/race, channels, generator
yield values, heap futures, and LIR. Recommended R5g is bounded reducible
async-control-flow normalization over the same machine stack.
