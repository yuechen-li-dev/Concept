# EVT1 async/await direction

Status: R5f implemented

## One execution model

An `async` function is compiler-authored machine syntax, not a second runtime.
Its operation owns a fixed inline stack of generated frames. An `await` lowers
to operand evaluation, an explicit parent continuation state, child-frame
push, child completion/pop, and exactly-once outcome consumption. Only an
explicit caller `Step` advances the top frame. `asynchronous` and
`awaitchronous` normalize to those same AST nodes and never select alternate
code paths. The long spellings are intentionally indulgent; the compiler is
not.

The generated identities are `F#async`, `F#async#state`, and
`F#async#machine`. MIR retains the await operand, eventual type, live-across
set, continuation identity, exactly-once laws, and source span. `AsyncPlan`
selects inline storage, generated-state continuation, bounded machine push,
`Scheduler: None`, and `SavedPC: None` before C emission.

## Suspension, iteration, and storage

Await resumes after its source suspension point because the compiler emits a
continuation state. Bare automata `yield;` remains different: it keeps the
current explicit state and re-enters that state's body from the beginning.
An async foreach therefore persists its compiler-generated index/current-item
state; a foreach inside a bare yielded state remains transient.

Parameters and liveness-selected locals occupy the generated frame. Ref and
view dependencies retain their backing provenance; a reference that cannot be
proved to outlive the operation is rejected. Owned persistent fields survive
child execution and drop once at completion. Lexically dead owned locals drop
before a push. Active operations are movable-only, and await of a named
operation is spelled `await move operation`.

## Separation from scheduling

`Async<T>` supplies only `Step`, `Complete`, and completion-gated `Result`.
Copyable results may be reread; movable-only results are one-shot. It owns no
worker, clock, queue, thread, executor, event loop, OS request, cancellation,
timeout, channel, or allocation policy. A future DragonGod or OS-I/O library
may hold operations and decide when to call Step, but that policy remains an
ordinary library above the same protocol.

## Bounded R5f surface and next direction

R5f supports sequential awaits plus simple `if`, bounded `while`, and fixed
contiguous `foreach` lowering. Match-arm awaits, general reducible control-flow
lowering, async interface methods/dyn, and OS-I/O integration are deferred.

Recommended R5g scope is a bounded general async control-flow normalizer:
lower reducible nested branches, match arms, loops, early exits, and cleanup
edges into the same explicit generated-state graph. It should not add a
scheduler, cancellation, timeout, select/race, channels, heap futures, or LIR.
