# Phase 19: Yielding machines and explicit suspension

## Core doctrine

```text
yield is explicit machine-step suspension.

yield is not return.

yield is not transition.

yield is not async.

yield is not a generator.

yield does not allocate.

yield does not create a task.

yield does not schedule anything.

yield preserves the current machine state.

yield leaves the machine incomplete.

The next Step re-enters the same state body from the beginning.

DragonGod may schedule yielded machines, but Concept core does not.
```

## 1. Motivation

Phase 18 made machines composable and deterministic enough for hierarchical
state-machine and utility-selection patterns. Machine frames can now own child
frames by value, explicitly step children, inspect child completion and results,
branch through runtime bool `transition match`, select through deterministic
`transition decide`, expose numeric `State(machine)`, and route machine runtime
failures through the shared `cpt_panic` path.

DragonGod needs long-lived tickable behaviors. A guard, planner, poller, or
controller often needs to do useful work on this tick without completing and
without moving to another state. The machine needs a way to explicitly say:

```text
I am not complete.
I am not transitioning.
I am yielding this Step.
Call Step again later.
```

That primitive should be machine-local, explicit, and compiler-lowered. It
should not be a library convention that pretends completion means waiting, and it
should not require callers to encode every wait as an artificial transition to
another state.

Phase 19 therefore introduces the design for bare `yield;` as explicit
machine-step suspension. Bare `yield;` is enough for DragonGod v0 because it
lets a state decline to finish the current `Step` while preserving the current
state and ordinary frame storage.

This must not become C# iterators, Rust async, Unity coroutines, futures, tasks,
or hidden scheduler semantics. Concept core remains explicit: no hidden task, no
ambient executor, no continuation magic, no heap allocation, and no implicit
DragonGod runtime.

## 2. Terminology and semantic choice

Phase 19 implements bare `yield;` only.

Rejected for v0:

```text
yield return
yield value
suspend
resume
await
async
generator protocols
continuation yield
```

Core v0 meaning:

```text
`yield;` exits the current Step immediately.
The machine remains incomplete.
The machine remains in the current state.
No result is written.
No transition is taken.
No scheduler is invoked.
No task is created.
No heap allocation occurs.
The next Step re-enters the same state body from the beginning.
```

## 3. State-level yield vs continuation yield

Supported v0 model:

```cpp
state Waiting {
    if (!Complete(child)) {
        yield;
    }

    return Result(child);
}
```

Meaning:

- `yield;` exits the current `Step`.
- The current state remains `Waiting`.
- The next `Step` starts `Waiting` from the top.

Deferred continuation model:

```cpp
state Work {
    int x = 1;
    yield;
    x = x + 1;
    yield;
    return x;
}
```

Continuation yield is deferred because it requires suspended instruction points,
lifted locals, sub-state program counters, deeper storage-state interaction, and
lowering that resembles coroutine machinery. It is not necessary for DragonGod
v0.

State doctrine:

```text
Phase 19 yield is state-level suspension, not intra-state continuation.
```

## 4. Syntax

V0 syntax:

```cpp
yield;
```

Rules:

- Statement only.
- No expression form.
- No value.
- No `yield return`.
- No `yield expr`.
- No `suspend`.
- Only allowed inside machine state bodies.
- Allowed inside blocks and branches within machine state bodies.
- Rejected in ordinary functions.
- Rejected at top level.
- Rejected in compile-time contexts.
- Rejected in concepts, interfaces, and structs unless specifically inside a
  machine state body.

Example:

```cpp
machine Waiter {
    state Waiting {
        yield;
    }
}
```

Invalid examples:

```cpp
int f() {
    yield;
}

machine Bad {
    state Start {
        yield 1;
    }
}

machine Bad2 {
    state Start {
        yield return 1;
    }
}

suspend;
```

## 5. Relationship to transition and return

The three machine control outcomes are:

```text
transition Target;  -> change current state
return value;       -> complete machine with result
yield;              -> stop this Step, stay in current state, remain incomplete
```

Rules:

- `yield;` does not complete the machine.
- `yield;` does not write result storage.
- `yield;` does not change state.
- `yield;` does not invoke panic.
- `yield;` is terminal for the current `Step` path.
- Statements after an unconditional `yield;` are unreachable in principle.
- P19 v0 may reject obvious statements-after-yield only if existing control-flow
  infrastructure makes that easy; otherwise that diagnostic is deferred.

## 6. Lowering model

Conceptual backend lowering:

```c
case STATE_Waiting:
    if (!condition) {
        return;
    }

    frame->complete = true;
    frame->result = value;
    return;
```

A `yield;` lowers to:

```c
return;
```

without mutating:

- state;
- completion flag;
- result slot.

Requirements:

- No new frame field for continuation point.
- No suspend index.
- No lifted locals.
- No scheduler call.
- No heap allocation.
- No helper call required.
- No `cpt_panic`.
- No `__builtin_trap`.

## 7. Interaction with Complete and Result

A yielded machine is incomplete. `Complete(machine)` after a yielding `Step`
returns `false`. `Result(machine)` after a yielding `Step` still panics with the
existing stable reason:

```text
machine result cannot be read before completion
```

`State(machine)` after yield returns the same state id as before yield, because
the state did not change.

Example:

```cpp
machine Waiter {
    state Waiting {
        yield;
    }
}

int main() {
    Waiter w = Waiter();
    Step(w);

    if (Complete(w)) {
        return 1;
    }

    return 0;
}
```

Expected behavior:

```text
return 0
```

## 8. Interaction with nested machines

DragonGod-relevant pattern:

```cpp
machine Child {
    state Start {
        return 7;
    }
}

machine Parent {
    Child child;

    state Waiting {
        Step(child);

        if (!Complete(child)) {
            yield;
        }

        return Result(child);
    }
}
```

Rules:

- Parent explicitly steps the child.
- If child is incomplete, parent yields.
- Parent remains in `Waiting`.
- The next `Step(parent)` re-enters `Waiting` and may step/check child again.
- No implicit child stepping.
- No scheduler.
- No event loop.

## 9. Interaction with transition match and transition decide

`yield;` may appear before or after ordinary statements in a state body according
to existing statement rules. `transition match` and `transition decide` remain
terminal transition forms. A state body path should not both yield and transition
in the same executed path.

`yield` does not evaluate transition candidates or match arms. `transition
decide` remains deterministic and source-order tied. A yielded machine can later
re-enter the same state and then choose a transition on a later `Step`.

Examples:

```cpp
state Waiting {
    if (!ready) {
        yield;
    }

    transition Done;
}
```

```cpp
state Waiting {
    if (!Complete(child)) {
        yield;
    }

    transition decide {
        Done score 10;
        Retry score 0;
    };
}
```

## 10. Runtime failure model

`yield;` itself is not a runtime failure. It does not panic and does not add a
new panic reason.

Existing runtime failures remain:

```text
machine result cannot be read before completion
machine decision transition has no enabled candidates
machine transition match found no matching case
invalid machine state reached
```

`Result(machine)` after yield uses the existing result-before-completion panic
path. `yield` does not introduce exceptions, unwinding, stack traces, or
scheduler cancellation.

## 11. Diagnostics planning

Suggested diagnostics:

```text
CON0300 YieldOnlyAllowedInMachineState
CON0301 YieldExpressionUnsupported
CON0302 YieldValueUnsupported
CON0303 YieldReturnUnsupported
CON0304 SuspendUnsupported
CON0305 YieldAfterTerminalStatement
```

Use exact names later based on implementation.

Rules:

- `yield;` outside machine states should be rejected.
- `yield` in expression position should be rejected.
- `yield value;` should be rejected.
- `yield return value;` should be rejected.
- `suspend;` should be rejected/reserved only if the parser sees it as an
  identifier today; do not add a `suspend` keyword unless required.
- Unreachable/after-terminal diagnostics are optional for v0 unless existing
  infrastructure supports them.

## 12. Interaction with Phase 18 and DragonGod

Phase 18 produced the pre-DragonGod machine substrate:

- Nested machines.
- Explicit child stepping.
- Runtime match.
- Runtime decide.
- `State(machine)`.
- Shared machine runtime failures.

Phase 19 adds tick/suspension behavior. DragonGod v0 can then be built as
library code, examples, and patterns over:

- Explicit machines.
- `Step`/`Complete`/`Result`.
- Nested child frames.
- `transition match`.
- `transition decide`.
- `State(machine)`.
- `yield`.
- Explicit structs/stores as context.

Concept core still does not include:

- Scheduler.
- Blackboard keyword.
- Mailbox keyword.
- Event bus.
- DragonGod runtime.

## 13. Non-goals for Phase 19

Explicitly deferred:

- `yield return`.
- `yield value`.
- `suspend`.
- `resume`.
- Continuation-preserving coroutine yield.
- Resume-after-yield statement continuation.
- Lifted locals for suspension.
- Sub-state program counters.
- `async`/`await`.
- Futures/promises/tasks.
- Scheduler.
- Event loop.
- Threads.
- Mailbox runtime.
- Blackboard runtime.
- Dynamic child machines.
- Heap-owned machines.
- Yield from ordinary functions.
- Generator/iterator protocols.
- Stream-of-values semantics.
- Cancellation.
- Timeout primitives.
- DragonGod runtime hooks.

## 14. Milestone plan

```text
P19-M0  Design doc: yielding machines and explicit suspension
P19-M1  yield statement syntax / AST / HIR scaffold (implemented: lexer keyword, parser statement, AST/HIR statement nodes, source spans, debug/deinit paths, machine-state semantic guard, and six fixtures)
P19-M2  yield validation: machine-state-only, statement-only, no value
P19-M3  backend lowering: yield exits Step without state/completion mutation
P19-M4  yield + Complete / Result / State behavior fixtures
P19-M5  yield + nested child machine ticking fixtures
P19-M6  yield + transition match / decide interaction fixtures
P19-M7  diagnostics and runtime-failure hardening
P19-M8  examples/fixtures: wait-until, polling, long-running behavior
P19-M9  closeout
```


## P19-M1 status: syntax / AST / HIR scaffold

P19-M1 is implemented as a syntax and compiler-representation scaffold only. Bare `yield;` now tokenizes as a keyword, parses in statement position inside machine state bodies, preserves its source span in AST and HIR, participates in AST/HIR deinit and debug dumps, and lowers from AST to HIR as a no-payload statement. `yield;` in ordinary functions is rejected with `CON0300`. Value-bearing forms such as `yield 1;`, `yield return 1;`, and expression-position `yield` remain rejected by the parser. Runtime lowering remains intentionally unsupported and deferred to P19-M3; P19-M2 will harden full validation. No `yield return`, `yield value`, `suspend`, continuation yield, async/generator protocol, scheduler, blackboard/mailbox runtime, or DragonGod runtime hook is introduced. The M1 fixture set adds six fixtures under `language/phase19-yielding-machines/`.
