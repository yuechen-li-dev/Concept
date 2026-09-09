# EVT1 async/await direction

Status: R5g structured normalization implemented

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

## R5g structured normalization

The compiler now exposes this pipeline:

```text
Async source MIR
    -> Structured Async CFG
    -> reducible control-flow normalization
    -> generated explicit machine states
    -> ordinary machine-stack MIR
    -> Planner
```

The normalized graph uses readable categories including `BasicBlock`,
`AwaitSuspend`, `AfterAwait`, `Branch`, `BranchJoin`, `MatchDispatch`,
`MatchJoin`, `LoopInit`, `LoopHeader`, `LoopBackedge`, `ForeachInit`,
`ForeachHeader`, `ForeachContinue`, `TryJoin`, and `Complete`. Source order,
not pointer identity, determines stable state names. Correctness and
inspectability take precedence over state minimization.

Branches and match arms can contain multiple awaits and nest in loops. Match
payload bindings use the ordinary frame liveness pass. Bounded while loops
return through explicit headers and backedges, and foreach retains its source
and iterator progress across child execution. A bounded `try`/`except` region
routes an awaited `Result` failure to a typed local handler state; `?` remains
ordinary Result propagation after the child outcome materializes. There is no
exception stack or coroutine interaction.

## Whiteboard before and after

The hand-authored equivalent needs names such as `Start`, `AfterRead`,
`LoopHeader`, `AfterProcess`, `LoopBackedge`, and `Complete`, plus explicit
push/outcome plumbing. The source can state the work directly:

```concept
async Result<int, Error> ProcessAll(Span<Item> items)
{
    int total = 0;
    foreach (Item item in items)
    {
        Data data = await Read(item)?;
        total = total + await Process(data)?;
    }
    return Result::Ok(total);
}
```

The emitted MIR and Plan contain those states and edges automatically. The
long spellings remain an intentionally rather spacious Easter egg; they still
buy no additional semantics.

## Bounded omissions and R5h

Await in control expressions, arbitrary or irreducible CFGs, recursive-depth
inference, async interface/dyn methods, OS-I/O integration, schedulers,
cancellation, timeout, select/race, channels, heap futures, and LIR remain
deferred. Recommended R5h is a bounded async callable/interface boundary:
prove statically selected interface methods and fixed witness dispatch over
the existing inline operation and machine stack, without adding dynamic task
allocation, scheduler ownership, or an async runtime ecosystem.
