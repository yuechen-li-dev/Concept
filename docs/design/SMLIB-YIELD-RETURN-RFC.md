# RFC: value-producing loops and `yield return`

Status: investigation only; recommendation **defer**. No syntax, AST, MIR,
automata behavior, or runtime implementation is added by SMLib hardening.

## Motivation and necessity

The SMLib loops use early convergence, boundary termination, evaluator errors,
and nested searches. Existing functions returning `Result` records, finite
ranges, explicit while conditions, and explicit automata state transitions can
express all those cases. The port already executes without imperative loop
escapes. A loop value could reduce mutable output state in a search, but is not
necessary to implement these algorithms. Adding it to frozen EVT1 needs stronger
usability evidence than the absence of conventional break/continue.

Reviewed authorities: CONCEPT_EVT1_LANGUAGE_SPEC sections on lexical lifetime,
reverse-order cleanup, iteration/yield, machine stacks, and failure propagation;
docs/language/LOOPS.md; iteration_yield.go; machine_stack.go; validateBlock and
ordinary return/ownership checks; generated lexical cleanup; and the port's
Solve1d, oscillation probing, and SolveNd inner candidate search.

## Candidate grammar (not accepted language)

A restricted, explicitly typed loop expression would avoid changing existing
statement loops or making every loop a potential escape target:

```text
loop-expression := loop < Type > while ( Expression ) Block else Block
                 | loop < Type > for ( identifier in Expression ) Block else Block
loop-completion := yield return Expression ;
```

`loop` would be contextual in expression position, not a globally reserved
identifier. This additional marking is a design cost: introducing `yield return`
alone leaves the grammar and target category underdetermined. The candidate
requires explicit result type T; inference is deferred. Normal exhaustion must
execute an else block that definitely produces T with yield return, or terminates
the enclosing function through ordinary return/failure. No implicit default,
implicit Option, last-value rule, or uninitialized result is permitted.

## Exact candidate semantics

Evaluate the iterable once, as existing for does; evaluate while conditions in
their ordinary order. `yield return e;` evaluates e exactly once, transfers a T
into final result storage, cleans all exited lexical scopes in reverse order,
and finishes the innermost enclosing **marked value loop**. It never suspends.
Nested ordinary statement loops do not create value targets; nested marked loops
do. Labels and nonlocal loop selection are excluded. An else block targets its
own marked loop. A completion cannot cross a function or callback boundary.
Returning from a function remains ordinary return; Result `?` remains ordinary
failure propagation. No `yield continue`, `yield next`, conventional break,
continue, or C-style for is proposed.

Existing `yield;` still ends the current automata Step and preserves explicit
persistent state. `complete;`, `complete value;`, and `fail error;` still pop or
complete machines with their existing outcomes. Initially forbid marked loops
and yield-return inside automata state bodies (including nested state blocks)
and async bodies, until their interaction with existing Step-ending paths is
separately designed. Callbacks are separate lexical control regions and cannot
complete a loop outside themselves.

## Ownership, references, and cleanup

Use existing construction/transfer rules: noncopyable named values require
explicit move, no hidden copies or allocation. Evaluate and store the result
before cleanup. A moved result owner is removed from the exited scopes' live
sets, and all other live owners drop exactly once. References to loop locals,
iteration items backed by temporary iterables, or shorter-lived nested storage
cannot escape. Borrowed results need ordinary destination provenance checks.
Every normal/completion path must initialize the result once and satisfy the
same ownership join. MaybeMoved Drop owners remain rejected under existing
invariants; the construct must not invent dynamic cleanup stacks to admit them.

## Examples in the candidate grammar

Valid candidate, with both early completion and normal exhaustion explicit:

```concept
Option<int> first = loop<Option<int>> for (value in values) {
    if (value > 0) { yield return Option::Some(value); }
} else {
    yield return Option::None;
};
```

Valid nested candidate: completion selects the nearest marked value loop,
independently of ordinary nested statement loops. A caller can use the inner
result in the outer loop without nonlocal control transfer.

Invalid candidates:

```concept
yield return 1;                  // no marked value loop
loop<int> while (ready) { }      // no definite exhaustion result
loop<int> while (ready) {
    yield return true;           // wrong result type
} else { yield return 0; };
loop<ref int> while (ready) {
    int local = 1;
    yield return ref local;      // escaping local storage
} else { return fallback; };
```

These are RFC examples, not compile-positive fixtures or implemented syntax.

## AST and MIR strategy if later accepted

Introduce a distinct ValueLoopExpr with result type, existing loop kind/source,
body, and mandatory exhaustion body. LoopCompletionStmt holds its resolved target
identity, expression, transfer, and cleanup boundary. Semantic validation maintains
a lexical value-loop target stack independently of automata transitions, and
closes that stack at callback/function boundaries. Infer no target in C emission.

Lower to ordinary explicit CFG: preheader, condition/iterator, body, exhaustion,
and merge. Each completion initializes the result place, emits existing lexical
cleanup to the target boundary, and branches to the merge; exhaustion does the
same. Result storage has an initialization proof across predecessors (SSA phi for
copyable values or existing final-place storage for ownership-sensitive values).
No hidden continuation frame, heap, scheduler, persistent state, or runtime
reflection is involved. Normal/Verify must share this semantic CFG.

Required future tests: zero iterations, nested marked/ordinary loops, callback
boundaries, moves and reverse Drop order, error propagation, reference escape,
MaybeMoved joins, single evaluation, and all forbidden machine/async placements.

## Recommendation

**Defer.** First use the compiler fixes and current structured loops in real
numerical code. Function extraction already permits early return with exact
cleanup, while explicit automata are appropriate when persistent state and
stepping are actually intended. A value loop adds grammar, target, result-place,
and ownership-join obligations and should have a separate accepted semantic
milestone. Do not prototype or merge it during this hardening task.
