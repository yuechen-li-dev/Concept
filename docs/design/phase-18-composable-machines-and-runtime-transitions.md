# Phase 18: Composable machines and runtime transitions

## Core doctrine

```text
Machines compose explicitly.

A nested machine is a field, not a scheduler task.

A machine frame is an explicit runtime value.

Step is explicit.

Completion is explicit.

Result is explicit and panics before completion.

transition match is deterministic runtime branching.

transition decide is deterministic utility selection.

Runtime machine failures use the shared panic path.

No hidden heap.

No hidden scheduler.

No async runtime.

No event bus keyword.

No blackboard keyword.

No DragonGod-specific runtime in core Concept.

DragonGod is built on these primitives; it is not baked into the language.
```

## 1. Motivation

Phase 13 made machines real enough for literal-transition execution. It closed
explicit `machine` declarations, `state` blocks, runnable literal transitions,
nominal machine frame construction, `Step(machine)`, `Complete(machine)`, and
`Result(machine)`. That was enough to prove that Concept machines are explicit
runtime values lowered through the real compiler path, but not enough to express
hierarchical state machines: child machines could not yet be stored inside parent
frames, stepped explicitly, or used as ordinary composition units.

Phase 17 provided the shared runtime failure substrate. Runtime `panic`, runtime
`assert`, and machine `Result(machine)` before completion now converge on the
same backend-owned `cpt_panic` helper and deterministic exit behavior. Phase 18
should use that substrate for every new machine runtime failure path instead of
adding ad-hoc traps or helper exits.

DragonGod needs machines that can be composed, stepped, inspected, and driven by
deterministic runtime transitions. A guard behavior should be able to own a
patrol child frame. A combat behavior should be able to branch on child
completion. A utility behavior should choose among states using stable scores and
source-order tie-breaking. These are language-machine primitives, not a hidden
DragonGod runtime.

Phase 18 therefore bridges Concept's machine primitive into a DragonGod-ready
substrate without making DragonGod a language feature. It enables hierarchical
state-machine patterns and deterministic utility decisions while preserving the
Concept doctrine: explicit values, explicit storage, explicit steps, and no
ambient scheduler.

Central goal:

```text
Make machines composable and make non-literal transitions executable.
```

## 2. Machine frame/value model

A machine value is an explicit runtime frame. The frame contains the current
state, completion flag, result slot, captured parameters, and any compiler-lifted
or machine-declared storage required by the machine body.

A machine frame is not any of the following:

```text
heap task
actor
coroutine object
scheduler fiber
hidden async job
```

`Step(machine)` mutates an explicit machine place. `Complete(machine)` inspects a
machine frame. `Result(machine)` inspects a machine frame and panics if the frame
is incomplete. Machine frames are ordinary storage values with compiler-defined
layout.

Machine copy/move behavior follows existing storage and move semantics. Phase 18
does not introduce hidden cloning, reference-counting, heap identity, or
scheduler identity for machine values.

Design answers for v0:

- **Can a machine frame be a local?** Yes. Phase 13 already supports local
  machine values created with `MachineName(...)`.
- **Can a machine frame be a field?** Yes in Phase 18 v0, when the field type is
  a machine type and initialization can be satisfied by the conservative child
  initialization rules.
- **Can a machine frame be copied/moved?** It follows existing storage/move
  rules. Phase 18 does not invent special machine cloning. If a machine frame's
  contained fields make it non-copy, existing non-copy diagnostics should apply.
- **Can a machine frame contain child machine frames?** Yes. A parent frame may
  contain child frames by value as fields.
- **How are result slots represented?** As part of the explicit frame, using the
  existing Phase 13 result storage model: a completion flag gates reads, and
  `Result(machine)` reads the result slot only after completion.
- **How are child machine fields initialized?** Zero-parameter child machine
  fields may be default-initialized as part of parent construction. Parameterized
  child fields are deferred unless existing explicit field initialization syntax
  makes the construction trivial and visible.

## 3. Nested machine fields / child frames

Phase 18 v0 introduces nested machine composition by storing child machine frames
as fields of parent machine frames.

Target shape:

```cpp
machine Patrol {
    state Start {
        transition Done;
    }

    state Done {
        return 7;
    }
}

machine Guard {
    Patrol patrol;

    state Start {
        Step(patrol);
        transition Check;
    }

    state Check {
        transition Done;
    }

    state Done {
        return Result(patrol);
    }
}
```

Rules:

- A machine may contain fields whose types are other machines.
- A nested machine is stored by value inside the parent machine frame.
- A nested machine is not a scheduled task.
- A parent machine must explicitly call `Step(child)`.
- A parent machine may call `Complete(child)`.
- A parent machine may call `Result(child)` after completion.
- `Result(child)` before completion uses the shared panic path.
- Nested machine fields should initially support zero-parameter child machines.
- Parameterized nested machine fields are deferred unless explicit
  initialization already exists and is trivial.
- Dynamic or heap child machines are deferred.

## 4. Nested machine initialization

Conservative v0 rules:

- Zero-parameter child machine fields can be default-initialized as part of parent
  construction.
- Parameterized child machine fields are deferred unless explicit field
  initialization syntax already supports them.
- No hidden heap allocation.
- No implicit scheduler registration.
- No implicit parent-child stepping.
- No dynamic child machine lists in v0.

Default construction is enough for v0 if the child machine has no parameters and
its frame can be initialized with the same constructor state as `Child()`. Phase
18 should not block on a broad field-initializer feature.

A future explicit child initializer syntax may be required for parameterized
children or for non-default child construction:

```cpp
machine Guard {
    Patrol patrol = Patrol();
}
```

That syntax should remain explicit and should still initialize a by-value child
frame, not allocate or schedule a child task.

## 5. Step / Complete / Result composition

Nested machine operations use the existing machine operations on child places:

```cpp
Step(child);
Complete(child);
Result(child);
```

Rules:

- `Step(child)` requires an assignable machine place.
- `Complete(child)` accepts a readable machine place.
- `Result(child)` accepts a readable machine place but panics if incomplete.
- These operations work on local machines and nested machine fields.
- No hidden scheduler.
- No implicit loop.
- No automatic stepping of children.
- No result read before completion.

Example:

```cpp
state Waiting {
    Step(patrol);

    transition match (Complete(patrol)) {
        true => Done;
        false => Waiting;
    };
}
```

## 6. Runtime `transition match`

Phase 13 already has syntax and validation scaffolding for `transition match`.
Phase 18 should make it executable.

Example:

```cpp
transition match (Complete(patrol)) {
    true => Done;
    false => Waiting;
};
```

Rules:

- `transition match` evaluates a scrutinee expression at runtime.
- Matching is deterministic.
- Matching selects exactly one target state.
- State targets must exist.
- No fallthrough.
- No allocation.
- Supported v0 scrutinee types should be conservative:
  - `bool`;
  - enums if already stable enough;
  - `int` only if case-label validation is already clean.
- For `bool`, both `true` and `false` must be covered or a default arm must
  exist.
- For enums, all known variants must be covered or a default arm must exist.
- If runtime no-match is possible, route to shared panic with stable reason:

  ```text
  machine transition match found no matching case
  ```

- Prefer compile-time coverage validation where feasible.

## 7. Runtime `transition decide`

Phase 13 already has syntax and validation scaffolding for `transition decide`.
Phase 18 should make it executable.

Example:

```cpp
transition decide {
    Attack when canAttack score attackScore;
    Flee when lowHealth score 100;
    Idle score 0;
};
```

Rules:

- `transition decide` evaluates candidates in source order.
- Each candidate has:
  - target state;
  - optional `when` guard;
  - `score` expression.
- Missing `when` means enabled.
- `when` guard type must be `bool`.
- `score` type is `int` in v0.
- Disabled candidates are ignored.
- Highest score wins.
- Ties break by source order.
- If no candidates are enabled, route to shared panic with stable reason:

  ```text
  machine decision transition has no enabled candidates
  ```

- No randomness.
- No allocation.
- No hidden scheduler.
- No implicit blackboard.
- Target states must exist.

Deterministic tie-breaking is required for tests and DragonGod reproducibility.
The same inputs, scores, guards, and source order must always select the same
state. That keeps fixture results stable, enables replayable behavior debugging,
and prevents utility decisions from smuggling in scheduler timing, randomness, or
iteration-order dependence.

## 8. Machine introspection helpers

Phase 18 should design, and may implement, a minimal helper useful for tests and
DragonGod loops:

```cpp
State(machine) -> int
```

Rules:

- `State(machine)` returns a stable machine-state numeric id, or an enum-like
  value if that representation is already available.
- It is for tests, debug loops, and DragonGod control loops.
- It should not require strings.
- `StateName(machine)` is deferred because runtime strings and formatting are
  not ready.
- No reflection system.
- No runtime list of states.
- No dynamic metadata allocation.

If `State(machine)` is too much for the executable subset before match/decide
hardening, it should be deferred to P18-M6 as a hardening/helper milestone rather
than blocking nested child frames or transition lowering.

## 9. Runtime failure integration

All machine runtime failures introduced or touched in Phase 18 route through the
Phase 17 shared panic path. Stable reason strings should be documented and
fixture-pinned.

Suggested reason strings:

```text
machine result cannot be read before completion
invalid machine state reached
machine transition match found no matching case
machine decision transition has no enabled candidates
```

Rules:

- Do not introduce `__builtin_trap` or ad-hoc exit helpers for new machine
  runtime failure sites.
- The `cpt_panic` helper should be shared and emitted once.
- Existing successful machine semantics must remain unchanged.
- Compile-time machine validation errors remain diagnostics, not runtime panic.

## 10. Interaction with DragonGod

DragonGod should use Concept machines as explicit runtime values. It can build
blackboards, mailboxes, stores, score functions, and actuators as libraries and
patterns on top of language primitives.

Phase 18 does not add any of these as keywords:

```text
board
blackboard
mailbox
event
scheduler
actor
```

Phase 18 provides:

- nested machines for hierarchy;
- explicit stepping for control;
- `transition match` for deterministic branching;
- `transition decide` for utility selection;
- shared panic failures for invalid runtime states.

DragonGod is built on these primitives; it is not baked into the language.

## 11. Interaction with Phase 17 runtime failure

Phase 17 is now the canonical runtime failure substrate. `Result(machine)` before
completion already uses shared panic. Phase 18 transition runtime failures should
also use shared panic.

Panic reasons should be non-empty and stable. Phase 18 does not add exceptions,
unwinding, or stack traces.

## 12. Interaction with Phase 16 modules

Machines can exist in modules. A parent machine may eventually contain a child
machine type from an imported module if qualified type support already handles
that type position.

Phase 18 does not add new import semantics. Examples should use same-module
machines first. Cross-module machine composition can be added if it falls out
naturally and is tested through the existing Phase 16 multi-source model.

## 13. Non-goals for Phase 18 v0

Phase 18 v0 explicitly defers:

- `yield` / suspend / resume;
- coroutine-style lifted locals;
- async runtime;
- scheduler;
- actor model;
- event queues;
- blackboard keyword;
- mailbox keyword;
- dynamic child machine allocation;
- heap-owned machines;
- parallel/orthogonal states;
- history states;
- entry/exit actions unless already natural;
- UML statechart semantics;
- state inheritance;
- transition inheritance;
- reflection;
- string state names;
- dynamic machine lists;
- parameterized child machine fields unless explicit initialization already
  exists;
- DragonGod-specific compiler/runtime hooks.

## 14. Diagnostics planning

Suggested diagnostics:

```text
CON0290 NestedMachineFieldRequiresDefaultConstruction
CON0291 StepRequiresMachinePlace
CON0292 CompleteRequiresMachineValue
CON0293 ResultRequiresMachineValue
CON0294 TransitionMatchUnsupportedScrutinee
CON0295 TransitionMatchNonExhaustive
CON0296 TransitionMatchCaseTypeMismatch
CON0297 TransitionDecideGuardMustBeBool
CON0298 TransitionDecideScoreMustBeInt
CON0299 TransitionDecideRequiresCandidate
CON029A MachineDecisionNoEnabledCandidate
CON029B MachineTransitionMatchNoCase
CON029C InvalidMachineStateReached
CON029D StateRequiresMachineValue
```

Use exact names later based on implementation. Runtime failure codes like
`CON029A` and `CON029B` may be compile-time diagnostics only if the failure is
statically knowable. Otherwise runtime failures use the stable panic reason
strings documented above.

## 15. Milestone plan

```text
P18-M0  Design doc: composable machines and runtime transitions
P18-M1  Machine frame/value model audit and hardening
P18-M2  Nested machine fields / child machine frames
P18-M3  Step/Complete/Result composition for nested machines
P18-M4  Runtime transition match lowering
P18-M5  Runtime transition decide lowering
P18-M6  Machine introspection helpers
P18-M7  Runtime failure hardening for machine transitions
P18-M8  Examples/fixtures: hierarchical machines and utility decisions
P18-M9  Closeout
```

## 16. Status update scope

This document is a design milestone only. P18-M0 does not implement compiler
behavior and should not change lexer, parser, HIR, MIR, or backend behavior.
Light status/index updates may mention that Phase 18 has started and that the
new design document defines the intended machine-composition and runtime
transition work.
