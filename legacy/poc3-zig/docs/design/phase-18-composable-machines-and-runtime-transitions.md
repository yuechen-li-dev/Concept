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
- **Can a machine frame be a field?** Planned for P18-M2+. P18-M1 deliberately
  does not implement nested machine fields or child frame initialization.
- **Can a machine frame be copied/moved?** It follows existing storage/move
  rules. Phase 18 does not invent special machine cloning. If a machine frame's
  contained fields make it non-copy, existing non-copy diagnostics should apply.
- **Can a machine frame contain child machine frames?** Planned for P18-M2+.
  P18-M1 only audits the existing single-frame value model.
- **How are result slots represented?** As part of the explicit frame, using the
  existing Phase 13 result storage model: a completion flag gates reads, and
  `Result(machine)` reads the result slot only after completion.
- **How are child machine fields initialized?** Zero-parameter child machine
  fields may be default-initialized as part of parent construction. Parameterized
  child fields are deferred unless existing explicit field initialization syntax
  makes the construction trivial and visible.


## P18-M1 implementation status: machine frame/value audit and hardening

P18-M1 is an audit and hardening milestone, not a nested-machine implementation milestone. It pins the current machine frame/value model before P18-M2 adds child machine fields.

### Current machine frame/value audit

- **Machine frame representation:** a machine declaration creates a nominal `machine_type` in the type store keyed by the HIR machine id. In executable C, each machine lowers to a `typedef struct` frame with a state enum field, `int complete`, a scalar result slot, and one field per captured machine parameter.
- **Construction behavior:** `MachineName(...)` lowers to a backend constructor function returning the frame by value. The constructor initializes the initial state, clears `complete`, zero-initializes the result slot, and copies scalar parameters into frame fields.
- **Local machine behavior:** machine frames are supported as ordinary locals. Multiple local frame instances are independent when constructed separately.
- **Step behavior:** `Step(machine)` is statement-like/void and now requires an assignable machine place (`CON0291`). The current supported place surface is a local or parameter machine binding. `Step(Simple())` is rejected instead of stepping a compiler-created temporary.
- **Complete behavior:** `Complete(machine)` accepts readable machine values/places and returns the frame completion flag. Non-machine operands are rejected with `CON0292`. Temporaries remain readable under the current value model, but relying on completion of a freshly constructed temporary is provisional and not a nested-machine feature.
- **Result behavior:** `Result(machine)` accepts readable machine values/places, returns the machine result type, and is valid only after completion at runtime. Non-machine operands are rejected with `CON0293`. Temporary reads remain part of the current readable-value behavior, but are provisional.
- **Result-before-completion panic behavior:** incomplete `Result(machine)` reads lower through the shared `cpt_panic("machine result cannot be read before completion")` path and preserve deterministic runtime failure exit code 101.
- **Copy/move/assignment behavior:** machine frames are currently copyable/assignable by value through existing storage rules. A copied frame copies state, completion, result, and captured parameter fields. This behavior is fixture-pinned as current/provisional; P18-M2 must revisit it before nested child frames make frame copies deeper.
- **Parameter/return behavior:** machine values can be represented as function parameter or return types in the current type/backend model when otherwise expressible, but P18-M1 does not broaden or promote this as stable API doctrine.
- **Field behavior:** machine fields are not implemented as a supported machine-composition feature in P18-M1. Nested machine fields, initialization, and operations on child fields are deferred to P18-M2+.
- **Backend frame layout summary:** current generated C uses a state enum, a by-value frame struct, a by-value constructor, and a pointer-taking step function. `Step` calls the step function with the address of the local frame; `Complete` reads `.complete`; `Result` reads `.result` after a `.complete` guard.

### Assumptions for P18-M2 nested fields

P18-M2 may build on nominal machine frame types, by-value frame constructors, explicit local storage, explicit pointer-taking step emission, completion/result fields, and shared panic routing. It must not assume stable final field names beyond the audited essentials, and it must decide whether by-value copying of parent frames with child frames is acceptable or should be restricted before child fields are stabilized.

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
P18-M5 makes the v0 utility-selection subset executable.

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
- Empty decide transitions are rejected with `CON0299`.
- Repeated target states are allowed; each candidate remains an independent source-order candidate.

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

## M2 status: nested machine fields / child frames

Phase 18 M2 adds the first storage-only form of explicit machine composition. A machine body may declare an ordinary field whose type is another machine type. For v0, accepted child machine fields must be default-constructible: the child machine must have zero parameters. Parameterized child machine fields are rejected with `CON0290 NestedMachineFieldRequiresDefaultConstruction` because explicit child-field initializer syntax is deferred.

Accepted child machine fields are represented in HIR as machine fields and emitted in the C backend as by-value child frame storage in the parent frame. The parent constructor initializes each child field by invoking the zero-argument child machine constructor. Construction does not step the child and does not allocate heap storage, register scheduler tasks, start async work, or create hidden child lists.

M2 is intentionally not a nested execution milestone. `Step`, `Complete`, and `Result` composition for child fields is deferred to P18-M3 unless existing place/value machinery naturally supports a case that is explicitly tested. Current machine copy/assignment remains the provisional by-value behavior documented in M1; nested child frames are copied as part of the parent value under that provisional policy.

## P18-M3 implementation status: nested machine operations

P18-M3 enables explicit `Step`, `Complete`, and `Result` composition over zero-parameter child machine fields declared in a parent machine body. Inside a parent state body, a bare child field name resolves to that child frame when no local or parameter shadows it, so `Step(child);`, `Complete(child)`, and `Result(child)` operate on the child frame stored inside the current parent frame.

The implementation preserves the Phase 18 doctrine: a child machine is a field, not a task. Child frames are initialized by the parent constructor, but they are not automatically advanced. The backend emits a child step only for an explicit `Step(child)` call, passing the address of the child field in the parent frame; `Complete(child)` reads the child field's completion flag; `Result(child)` reads the child result through the existing shared completion guard and routes result-before-completion through `cpt_panic` with deterministic exit code 101.

Current copy/assignment behavior remains provisional: nested child fields copy as part of parent frame copy/assignment under the existing by-value storage semantics. This must be revisited before DragonGod or other libraries rely on copyable parent machines. Runtime `transition match` and runtime `transition decide` are implemented by later P18-M4/P18-M5 subsets; `yield`, schedulers, async, event buses, blackboards/mailboxes, dynamic child lists, heap-owned machines, parameterized child initialization, and DragonGod runtime hooks remain deferred/non-goals.

## P18-M4 status: runtime `transition match` lowering

P18-M4 makes the v0 bool subset of `transition match` executable. A machine state may now branch on a `bool` scrutinee, including machine parameters, boolean expressions, comparisons, and nested-machine `Complete(child)` expressions. Case labels are limited to `true`, `false`, and the existing wildcard default syntax. Bool matches must cover both boolean values unless a wildcard arm is present. Duplicate labels, non-bool scrutinees, mismatched labels, and empty matches are diagnostics.

Lowering is deterministic runtime branching: the scrutinee is evaluated at `Step` time and the selected arm assigns the parent frame state to the target state. Exhaustive bool matches lower to C `if`/`else` state assignment without heap allocation, scheduler hooks, async machinery, event buses, blackboards, or DragonGod runtime hooks.

Example:

```cpp
transition match (Complete(child)) {
    true => Done;
    false => Check;
};
```

## P18-M5 status: runtime `transition decide` lowering

P18-M5 makes `transition decide` executable for ordered candidates with optional `bool` guards and required `int` scores. Guards and scores are type-checked during HIR lowering: non-bool guards report `CON0297`, non-int scores report `CON0298`, empty candidate lists report `CON0299`, and unknown target states continue to use the existing machine-state diagnostic. Candidates are evaluated in source order, missing `when` means enabled, score expressions are evaluated only for enabled candidates, the highest score wins, and ties keep the earlier candidate because the backend uses a strict `>` comparison rather than `>=`.

The backend lowers decide transitions to ordinary imperative C inside the machine step path: local `cpt_has_candidate`, `cpt_best_score`, and `cpt_best_state` temporaries track the current winner, each candidate updates those temporaries only when enabled and strictly better, and the final selected state is assigned to the machine frame. If no candidate is enabled, generated code calls shared `cpt_panic` with the stable reason `machine decision transition has no enabled candidates`, producing the Phase 17 runtime failure exit code 101. No heap allocation, scheduler, async machinery, event bus, blackboard/mailbox, randomness, behavior-tree runtime, planner runtime, or DragonGod hook is introduced.

The fixture corpus covers highest-score selection, guard filtering, source-order tie-breaking, unconditional candidates, no-enabled shared panic, backend C shape, and composition with nested machines through `Complete(child)` and `Result(child)`. Yield, schedulers, async/event-bus facilities, dynamic child lists, heap-owned machines, `State(machine)`, DragonGod runtime hooks, floating-point scores, randomness, weighted selection, behavior-tree runtimes, and GOAP/planner runtimes remain outside the v0 subset.

## P18-M6 status: machine state introspection helper

P18-M6 adds the minimal `State(machine) -> int` machine introspection helper. `State` accepts one readable machine value/place, including local machine frames and nested child machine fields in machine state bodies, and returns the backend/compiler numeric state id for the frame's current state. The read is non-mutating, does not require completion, does not check `Result`, does not panic for ordinary states, and lowers as a direct frame state-field read.

The v0 state-id contract is intentionally narrow: ids are stable within one compiled machine definition, the initial state id is deterministic, ids are intended for tests, debug loops, and DragonGod control loops, and they are not source-level enum variant names. There is no cross-version ABI stability promise yet. P18-M6 does not add `StateName(machine)`, string state names, runtime metadata tables, reflection, state lists, a source-level state enum surface, scheduler/yield/async/event-bus behavior, blackboards/mailboxes, dynamic child lists, heap-owned machines, or DragonGod runtime hooks.

Invalid non-machine operands are rejected with `CON029D StateRequiresMachineValue`; invalid arity uses the existing invalid-call diagnostic. The Phase 18 fixture corpus now includes runtime, backend, and invalid coverage for initial state reads, after-step state reads, nested child state reads, no-completion-required reads, direct C state-field lowering, and non-machine/arity failures. Remaining P18 work is runtime failure hardening, examples/fixtures, and closeout.

## P18-M7 status: runtime failure hardening for machine transitions

P18-M7 audits and hardens machine runtime failure paths without adding new transition forms. All machine runtime failures touched by Phase 18 now route through the shared Phase 17 backend-owned `cpt_panic` helper and deterministic exit code 101 path. The stable machine runtime failure reasons are:

- `machine result cannot be read before completion`
- `machine decision transition has no enabled candidates`
- `machine transition match found no matching case`
- `invalid machine state reached`

`Result(machine)` and `Result(child)` before completion are fixture-pinned to emit calls to `cpt_panic("machine result cannot be read before completion")`; stale machine-result trap/helper names and `__builtin_trap` remain absent. `transition decide` no-enabled-candidate failure is fixture-pinned to call `cpt_panic("machine decision transition has no enabled candidates")`.

Bool v0 `transition match` remains exhaustively validated at compile time: both `true` and `false` must be covered unless a wildcard arm exists. Therefore ordinary bool matches have no reachable runtime no-case fixture; the stable no-case reason is reserved for a future runtime no-match path if a wider match subset requires one.

Machine step dispatch now emits a defensive default arm that calls `cpt_panic("invalid machine state reached")` if a corrupted frame reaches an impossible state id. This is a defensive runtime failure path only; statically invalid machine programs remain diagnostics.

The backend emits the `cpt_panic` helper once per generated C unit even when manual `panic`, `assert`, machine result guards, decide no-enabled guards, and invalid-state dispatch guards coexist. P18-M7 adds backend fixtures for local result, nested child result, decide no-enabled, multiple panic sites, and invalid-state defensive emission. The Phase 18 machine fixture corpus now contains 60 fixtures. Remaining Phase 18 work is examples/fixtures and closeout.

## P18-M8 implementation status: examples and integration fixtures

P18-M8 is a showcase and hardening milestone. It does not add new core machine
semantics. Instead it documents and pins the composable-machine substrate already
implemented by P18-M1 through P18-M7.

### Examples added

`examples/phase18/` now contains polished examples for:

- hierarchical parent/child machines where the parent owns a child frame by value
  and explicitly steps it;
- match-driven child completion with `transition match (Complete(child))`;
- deterministic utility selection with `transition decide` and source-order tie
  behavior for equal scores;
- a nested utility controller combining child stepping, `Complete(child)`,
  `Result(child)`, `State(child)`, and `transition decide`;
- numeric machine introspection through `State(machine)`;
- runtime failure notes for incomplete `Result(machine)`, no-enabled decisions,
  and backend invalid-state defensive paths.

The Phase 18 README states the key doctrine: these primitives are the substrate
DragonGod can build on, not DragonGod itself. There is still no scheduler, async
runtime, event bus keyword, blackboard keyword, mailbox keyword, hidden heap,
dynamic child list, heap-owned machine, reflection surface, or DragonGod runtime
hook in core Concept.

### Integration fixture coverage

The Phase 18 fixture corpus now includes integration run fixtures for explicit
child stepping, match branch selection on child completion, utility-decision
branch selection, nested decide/controller composition, and state observation.
It also includes a backend-shape fixture that pins the combined lowering shape:
parent frames contain child frame fields, constructors initialize child frames,
parent steps pass child frame addresses to child step functions, bool match lowers
to deterministic branching, decide lowering uses candidate variables and strict
`>` comparison, `State(child)` reads the numeric `.state` field, panic sites share
`cpt_panic`, and generated C remains free of hidden `malloc`, scheduler, async,
blackboard, mailbox, event-bus, and DragonGod runtime markers.

### Fixture count

After P18-M8, `language/phase18-machines/` contains 66 `.conception` fixtures:
43 valid fixtures and 23 invalid fixtures.

### Remaining Phase 18 work

The remaining Phase 18 work is closeout: reconcile documentation, confirm the
fixture matrix, and close the phase without expanding the semantics surface.

Still deferred:

- `yield`;
- scheduler;
- async;
- blackboard, mailbox, or event keywords;
- dynamic child lists;
- heap-owned machines;
- parameterized child initialization;
- enum/int runtime match;
- `StateName(machine)`;
- reflection;
- DragonGod runtime hooks.

## P18-M9 closeout status: Phase 18 closed

P18-M9 closes Phase 18. P18-M0 through P18-M9 are complete:

- P18-M0 design document;
- P18-M1 machine frame/value audit and hardening;
- P18-M2 nested machine fields / child frames;
- P18-M3 `Step` / `Complete` / `Result` composition for nested machines;
- P18-M4 runtime bool `transition match` lowering;
- P18-M5 runtime deterministic `transition decide` lowering;
- P18-M6 `State(machine) -> int`;
- P18-M7 runtime failure hardening for machine transitions;
- P18-M8 examples and integration fixtures;
- P18-M9 closeout documentation, coverage, and fixture registration audit.

Final supported Phase 18 surface:

- machine frame/value audit and hardening;
- machine fields in machine bodies;
- zero-parameter nested child machine fields;
- child frame by-value storage in parent frame;
- parent constructor child-frame initialization;
- `Step(child)`;
- `Complete(child)`;
- `Result(child)`;
- bool `transition match` at runtime;
- `transition match` labels: `true`, `false`, `_`;
- `transition match` bool exhaustiveness/default validation;
- deterministic `transition decide` at runtime;
- decide candidate optional bool guard;
- decide candidate required int score;
- missing `when` means enabled;
- disabled candidates ignored;
- highest score wins;
- strict `>` tie behavior preserving source order;
- no-enabled decide shared panic;
- `State(machine) -> int`;
- local and nested child machine introspection;
- shared `cpt_panic` for machine runtime failures;
- invalid machine state defensive panic;
- `examples/phase18`;
- Phase 18 integration fixtures.

Final Phase 18 fixture count is 66 fixtures under `language/phase18-machines/`:
43 valid fixtures and 23 invalid fixtures. The full language fixture corpus has
1052 `.conception` files in the current corpus. The Phase 18 fixture categories cover
frame/value hardening, nested machine fields, nested operations, runtime match,
runtime decide, `State(machine)`, runtime failure hardening, and integration
examples/backend shape. P18-M9 also registers the six P18-M8 integration fixtures
in the Zig test harness so every Phase 18 fixture file is exercised.

Final examples added under `examples/phase18/`:

- `hierarchical-child-machine.concept`;
- `match-child-completion.concept`;
- `utility-decision.concept`;
- `nested-utility-controller.concept`;
- `machine-introspection.concept`;
- `runtime-failure-notes.concept`;
- `README.md`.

Final backend hardening guarantees:

- machine frames are explicit runtime values;
- nested child frames are parent-frame fields stored by value;
- parent constructors initialize zero-parameter child frames explicitly;
- `Step(child)` lowers to an explicit child-frame address step;
- `Complete(child)`, `Result(child)`, and `State(child)` lower to explicit frame
  reads or guarded reads;
- bool `transition match` lowers to deterministic runtime branching;
- `transition decide` lowers to deterministic source-order candidate evaluation;
- machine runtime failures route through shared backend-owned `cpt_panic` with
  deterministic runtime failure behavior;
- generated Phase 18 backend fixtures do not introduce hidden heap allocation,
  scheduler machinery, async runtime machinery, blackboard/mailbox/event-bus
  runtime machinery, reflection metadata, `StateName` machinery, or DragonGod
  runtime hooks.

Final stable runtime machine failure reasons:

- `machine result cannot be read before completion`;
- `machine decision transition has no enabled candidates`;
- `machine transition match found no matching case`;
- `invalid machine state reached`.

Bool v0 `transition match` is statically exhaustive/default-covered, so an
ordinary bool no-case runtime fixture is not emitted. The match no-case reason is
documented for future wider match subsets.

Final deferred non-goals:

- new syntax beyond the implemented Phase 18 surface;
- enum/int runtime `transition match`;
- additional `transition decide` score types;
- `yield`, suspend, or resume;
- scheduler;
- async runtime;
- event bus keyword;
- blackboard or mailbox keywords;
- dynamic child lists;
- heap-owned machines;
- parameterized child initialization;
- `StateName(machine)`;
- reflection or runtime state metadata;
- generic HSM/UML statechart support;
- DragonGod-specific runtime hooks in core Concept.

Phase 18 therefore closes as a composable, explicit, pre-DragonGod machine
substrate. DragonGod can be built on top of these primitives, but core Concept
contains no DragonGod runtime.
