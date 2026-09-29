# EVT1 R5g conformance

Baseline: `0a0104f4d13374a2fe270ed3f29486407d5f106c`
(`concept-evt1-stage0-go`).

R5g adds one explicit compiler boundary: structured async CFG normalization.
It turns reducible sequence, branches, matches, bounded loops, foreach, nested
combinations, and one bounded local try/except shape into deterministic
generated states over the unchanged R5e machine stack. It adds no scheduler,
saved PC, continuation object, heap frame, executor, coroutine runtime, or LIR.

## Graph and normalization evidence

The graph taxonomy is `BasicBlock`, `AwaitSuspend`, `AfterAwait`, `Branch`,
`BranchJoin`, `MatchDispatch`, `MatchJoin`, `LoopInit`, `LoopHeader`,
`LoopBackedge`, `ForeachInit`, `ForeachHeader`, `ForeachContinue`, `TryJoin`,
and `Complete`. Await splitting is mechanical; deterministic DFS numbering
produces stable `Function#KindN` identities. MIR exposes states, typed active
fields, edges and await/branch/join/loop counts. Validation rejects unknown or
unreachable targets, missing continuations, and inconsistent state order.

The frame field set combines ordinary live-across-await analysis with graph
liveness for locals used in another generated state, including backing
dependencies. Ordinary semantic analysis remains authoritative for match
exhaustiveness, definite assignment (`CV4560`), maybe-moved joins (`CV4503`),
cleanup, and ref provenance (`ASYNC_PERSISTENT_REF_ESCAPE`). Unsupported await
conditions/sources reject as `ASYNC_CONTROL_FLOW_NOT_REDUCIBLE`,
`ASYNC_MATCH_NORMALIZATION_INVALID`, or
`ASYNC_FOREACH_NORMALIZATION_INVALID`.

## Corpus

The canonical R5g corpus has 23 valid and five invalid human-readable sources.
The 17 required structured specimens cover both branch sides, nested if,
match and payload arms, while, loop-carried state, loop/match composition,
foreach and nested foreach, try/except, Result `?`, ownership, references,
moved-state joining, and complex hand-written-machine equivalence. Six
additional accepted cases cover evaluation counters, custom iterator counts,
class control flow, inference composition, Span-in-loop persistence, and the
long keyword aliases.

Strict-C11 native execution requires 42 from every accepted source. The
counter specimens prove branch and match evaluation once, loop-condition
evaluation once per admitted iteration, await operands once, skipped-child
construction zero times, foreach source/GetIterator once, MoveNext four times
for three items, and one Current-derived item per successful iteration.
Branch ownership proves one transient drop before suspension and one persistent
drop at completion. Nested outer references and class `self` remain valid;
short-lived branch references reject.

`async_manual_equivalence_complex.concept` places a hand-written match/loop/
two-child machine beside sequential async source and observes the same result.
The complex `async`/`await` and `asynchronous`/`awaitchronous` specimens produce
identical normalized state graphs. Plan generation is byte-identical over 100
runs. Generated C is scanned for allocation, threads, executor/event-loop,
coroutine, setjmp/longjmp, continuation-pointer, and saved-PC markers.

## Planner and backend

`AsyncPlan` records `ControlFlowStrategy: StructuredStateGraph`, generated
state/await/branch/join/loop counts, state mappings, edges, persistent fields,
frame size/alignment, `Scheduler: None`, and `SavedPC: None`. Exact regeneration
validation prevents the Planner from inventing semantics. GenericC11 consumes
the normalized graph as switch-tag transitions and ordinary machine-stack
push/pop/outcome operations.

## Deferred and recommended R5h

Await in control expressions, arbitrary/irreducible graphs, async interface or
dyn methods, recursive depth inference, OS-I/O policy, scheduler features,
cancellation, timeout, channels, select/race, heap futures, generator values,
and LIR remain deferred. Recommended R5h is bounded async callable/interface
composition using statically known methods and fixed witness dispatch over the
same inline `Async<T>` and machine stack. It must not grow into a promises,
task, scheduler, or heap-runtime ecosystem.
