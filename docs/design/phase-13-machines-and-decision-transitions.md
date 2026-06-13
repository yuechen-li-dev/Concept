# Phase 13: Explicit machines and decision-driven transitions

P13-M0 was a documentation-only milestone. It defines Concept's core machine
model, state declarations, transition statements, state-target expressions, and
HIR/MIR lowering direction.

P13-M0 does not implement compiler behavior. It does not add parser syntax,
machine lowering, runtime support, DragonGod kernel features, stack HFSM,
blackboards, mailbox buses, actuators, persistence, hysteresis, `min_commit`,
policy memory, a `board` keyword, or new `decide` semantics.

P13-M1 adds the parser and AST scaffold for `machine` declarations and nested
`state` declarations. The Stage 0 parser now preserves the machine name,
parameter list, result type, allocation effect metadata, attributes accepted by
the existing attribute parser, ordered state list, and each state's ordinary
statement body. `machine` and `state` are currently reserved lexer keywords
rather than contextual keywords. `state` is only accepted as a declaration form
inside a machine body.

P13-M1 deliberately does not implement machine HIR/MIR lowering, semantic state
validation, transition parsing or validation, runtime frames, step/resume
support, DragonGod features, `board`, stack HFSM, hidden heap behavior,
implicit scheduling, or new `decide` semantics. A machine that reaches semantic
declaration collection is rejected with `CON0231
MachineSemanticsNotImplemented` instead of silently disappearing.

P13-M2 adds the first semantic state validation layer. Each machine now has a
machine-local closed state universe: machines require at least one state,
duplicate state names in the same machine are rejected, state names remain
scoped to their containing machine, identical state names may appear in
different machines, source order is preserved, and the first declared state is
recorded as the v0 initial state. The semantic shell preserves machine name,
parameters, result type, allocation-effect metadata, attributes, ordered
states, and initial-state metadata for future transition validation. P13-M2
still does not lower machines to executable HIR/MIR/runtime behavior, does not
parse or validate transitions, and does not add DragonGod features, stack HFSM,
`board`, mailbox, actuator, policy memory, hidden heap, scheduler, or async
behavior. Otherwise valid machines still report `CON0231
MachineSemanticsNotImplemented`.

P13-M3 adds the first transition statement form:

```cpp
transition TargetState;
```

The parser accepts literal state transitions inside machine state bodies,
including ordinary nested blocks within a state body. Transition statements are
represented distinctly in the AST and preserve the bare target state name and
span. Semantic collection validates each literal target against the containing
machine's local state table: self-transitions, transitions to the initial
state, and transitions to later states are valid, while unknown or cross-machine
state targets report `CON0222 UnknownMachineState`. `transition` outside a
machine state body reports `CON0223 TransitionOutsideMachineState` where the
parser can encounter it. Otherwise valid machines with valid transitions still
report `CON0231 MachineSemanticsNotImplemented`.

P13-M3 deliberately does not implement `transition match`, `transition decide`,
transition target expressions, reachability/completeness analysis, HIR/MIR
machine lowering, runtime frames, step/resume behavior, DragonGod features,
stack HFSM, `board`, mailbox, actuator, policy memory, hidden heap, scheduler,
or async behavior.

P13-M4 adds deterministic match-driven transition targets in the current Stage
0 match syntax:

```cpp
transition match (kind) {
    TokenKind::Identifier => Identifier;
    TokenKind::Number => Number;
    _ => Error;
};
```

The parser accepts `transition match (...) { ... };` inside machine state
bodies, preserves the scrutinee expression, reuses existing match-pattern AST
forms, preserves arm source order, and records each arm's bare state target
name and span. Semantic collection validates every arm target against the
containing machine's local state table. Self targets, the initial state, and
later states are valid; unknown and cross-machine targets report `CON0222
UnknownMachineState` before the general machine placeholder. Otherwise valid
machines with literal or match transitions still report `CON0231
MachineSemanticsNotImplemented`.

For P13-M4, match arm results are intentionally narrow: each result must be a
bare machine state name. Arbitrary state-valued expressions, nested `decide`
results, qualified external state targets, user-visible machine state enums,
machine HIR/MIR lowering, runtime frames, scheduling, and DragonGod features
remain deferred. Transition-match currently performs target validation only;
full machine-body expression typing and match exhaustiveness for the transition
scrutinee remain part of later machine lowering/checking work.

P13-M5 adds contextual utility-scored decide-driven transition targets:

```cpp
transition decide {
    Attack when canAttack score attackScore;
    Heal when canHeal score healScore;
    Idle score 0;
};
```

The parser accepts `transition decide { ... };` inside machine state bodies,
preserves decide cases in source order, records each case's bare state target
name/span, and preserves optional `when` condition expressions plus required
`score` expressions. Semantic collection validates every decide case target
against the containing machine's local state table. Self targets, the initial
state, and later states are valid; unknown and cross-machine targets report
`CON0222 UnknownMachineState` before the general machine placeholder. Otherwise
valid machines with literal, match, or decide transitions still report
`CON0231 MachineSemanticsNotImplemented`.

For P13-M5, decide case results are intentionally narrow: each result must be a
bare machine state name. Arbitrary state-valued expressions, nested `match` or
`decide` results, qualified external state targets, user-visible machine state
enums, machine HIR/MIR lowering, runtime frames, scheduling, and DragonGod
features remain deferred. Transition-decide currently performs target
validation only; full condition/score typechecking for machine transition
decide cases remains deferred until machine bodies lower/typecheck through the
normal executable path. `decide` remains stateless in Concept core; policy
memory, hysteresis, and `min_commit` remain DragonGod/library features.

P13-M6 adds the explicit HIR machine lowering scaffold. Machine declarations
now have a real HIR item that preserves name, parameters, result type,
allocation-effect metadata, attributes, ordered states, and the v0 initial
state index. Each HIR state can reference its lowered state-body block, and
transition statements are represented in HIR with literal, match, or decide
transition targets. Literal targets, match arm targets, and decide case targets
resolve to machine-local state indexes while preserving target spans for later
diagnostics/debugging. HIR debug output exposes the machine, state, and
transition structure.

P13-M6 deliberately remains non-executable. Otherwise valid machines still
report `CON0231 MachineSemanticsNotImplemented`, with wording updated to state
that declarations are parsed, validated, and represented in HIR but executable
machine lowering/runtime support is not implemented yet. MIR lowering refuses
machine-containing modules instead of silently dropping them or treating them as
functions, and the C backend must not emit machine code. P13-M6 does not add
runtime frames, resume dispatch, Step/Active/Complete/Result builtins,
yield/suspend, lifted locals, Drop/lifetime handling, machine effect
enforcement, DragonGod features, stack HFSM, `board`, mailbox/event buses,
actuators, dirty-key tracking, persistence, hysteresis, `min_commit`, policy
memory, hidden heap behavior, scheduler behavior, or async behavior.

P13-M7 adds the first executable machine runtime model. A machine declaration
now defines a nominal frame value type, and `MachineName(...)` constructs that
frame with the first declared state active, `complete = false`, zero-initialized
result storage, and machine parameters captured into explicit frame fields.
`Step(machine)` executes the active state once; `transition Target;` updates the
frame state and ends the step; `return value;` stores the result, marks the
frame complete, and ends the step. `Complete(machine)` reads the completion
flag and `Result(machine)` reads the stored result after completion. Calling
`Step` on a completed machine is a no-op.

P13-M7 deliberately supports a narrow executable subset: single-active-state
machines, by-value frame storage, `int`/`bool` result and parameter storage,
direct `MachineName(...)` construction into an explicit local frame, literal
state transitions, simple returns, simple machine-parameter references, and
simple conditional state-body control that already lowers through HIR. The C
backend emits deterministic state enums, frame structs, constructor helpers,
and step functions. No generated machine code uses `malloc`, hidden heap
allocation, an implicit scheduler, async machinery, stack HFSM, child machines,
boards, blackboards, mailbox buses, actuators, persistence, hysteresis,
`min_commit`, policy memory, state history, or debug/timeline visualization.

Runtime lowering for `transition match (...) { ... };` and `transition decide {
... };` remains deferred after P13-M7. Those forms are still parsed, validated,
and preserved in HIR with resolved machine-local state indexes, but executable
C emission reports the existing machine-semantics unsupported diagnostic for
that transition form.

P13-M8 stabilizes the P13-M7 runtime subset with examples, fixtures, generated
C shape assertions, and incomplete-result hardening. Runnable examples now live
under `examples/phase13/` for literal-transition machines. The fixture corpus
covers `Step`, `Complete`, `Result`, scalar `int` and `bool` results, captured
scalar parameters, multiple literal transitions, completion checks before and
after return, extra completed steps as no-ops, runtime trapping for
`Result(machine)` before completion, and explicit backend refusal for
match/decide transition runtime lowering. The C backend now emits a small
trap helper for incomplete result access, plus the same explicit state enum,
frame struct, constructor, and step helper shape. P13-M8 does not add
DragonGod kernel features, a `board` keyword, hidden heap allocation, `malloc`,
a scheduler, async runtime machinery, stack HFSM, blackboards, mailbox buses,
actuators, persistence, hysteresis, `min_commit`, or policy memory.

## Core doctrine

```text
A Concept machine is a compiler-visible control frame.

A state is a named control region.

A transition is a typed decision that selects the next state.

A transition target may be a literal state, a match expression producing a
state, or a decide expression producing a state.

match is exact transition selection.

decide is utility-scored transition selection.

Concept provides machine semantics.

DragonGod provides behavior architecture.

Core Concept v0 does not include stack HFSM, blackboards, mailbox buses,
actuators, dirty-key tracking, persistence, hysteresis, min_commit, or policy
memory.

Machine lowering must be explicit and inspectable in HIR/MIR.

Machines must not imply hidden heap allocation, async runtime machinery, or an
implicit scheduler.
```

## 1. Motivation

Phase 13 follows Phase 12 because Concept now has enough substrate to make
explicit machine control meaningful instead of decorative. Earlier phases added
enums and `match`, judgment expressions through `decide`, runtime structs and
places, MIR, ownership/move/Drop/storage-state accounting, first-class testing,
and an explicit allocation doctrine.

Those pieces matter together. A machine primitive touches control flow, typed
state targets, storage frames, resource cleanup, testing, and allocation
visibility. Without Phase 12, a machine frame could accidentally become hidden
heap machinery. Without Phase 10, suspended or lifted storage would be
under-specified. Without Phase 5a, utility-scored transition selection would be
only a library convention.

Many of Concept's target domains are naturally state-machine-shaped:

- lexers;
- parsers;
- compilers;
- protocol handlers;
- drivers;
- runtimes;
- incremental pipelines;
- cooperative control systems;
- game and AI behavior systems.

C++ usually encodes these as switch/if forests over enum state:

```cpp
switch (state) {
    case State::Start:
        if (isDigit(c)) {
            state = State::Number;
        } else if (isAlpha(c)) {
            state = State::Identifier;
        } else {
            state = State::Error;
        }
        break;
}
```

That representation is mechanically honest. There is a state tag, a dispatch,
some conditions, and an assignment to the next state. But it is cognitively
backwards: the programmer reads incidental dispatch mechanics before seeing the
control regions and transition decisions.

Concept should make the control frame and transition decision explicit. The goal
is not less typing. The goal is visible, typed, inspectable control progression:

```cpp
machine Lexer(mut LexerInput& input) -> Token {
    state Start {
        transition match (input.peekClass()) {
            CharClass::Digit  => Number;
            CharClass::Letter => Identifier;
            CharClass::End    => Done;
            _                 => Error;
        };
    }

    state Number {
        return input.readNumber();
    }

    state Identifier {
        return input.readIdentifier();
    }

    state Done {
        return Token.End;
    }

    state Error {
        return Token.Error;
    }
}
```

## 2. Concept vs Octomata vs DragonGod

Octomata is Oct's explicit behavioral/control runtime model. It has flows,
states, guard `when`, utility `when`, controller-bound `when policy`, board
memory, Step/Active/Complete/Result inspection, and resume/suspend mechanics.
It separates behavior progression from durable data lanes.

Oct needs `board` because Oct records are immutable and flows still need mutable
flow-local control memory. `board` is a language/runtime answer to a specific
constraint in Oct's data model.

Concept has different substrate. It already has mutable structs, places, field
assignment, ownership state, arenas, and explicit allocation. Concept therefore
does not need a core `board` keyword merely to provide mutable memory.
Machine-local control memory should use ordinary structs, fields, parameters,
and later explicit machine frame fields.

Concept's core responsibility is lowerable machine semantics: named states,
typed transition targets, explicit transition terminators, completion, and
inspectable HIR/MIR representation.

DragonGod is the full behavior kernel built in Concept. DragonGod owns the
larger architecture: stack HFSM, blackboards, typed mailbox event bus,
actuators, dirty-key tracking, persistence, hysteresis and `min_commit`, policy
memory, consideration scoring frameworks, utility curves, debugging, and
visualization. DragonGod should build on Concept machines and `decide`; those
features should not be placed inside the language core v0.

```text
Concept gives the engine block.
DragonGod builds the dragon cockpit.
```

## 3. Core semantic model

```text
machine:
  a compiler-visible control frame with named states and a result type.

state:
  a named control region inside a machine.

transition:
  statement that selects the next state and ends the current machine step.

return:
  completes the machine with the declared result type.

suspend/yield:
  optional future statement that ends the current step without changing state.
```

Core v0 is intentionally narrow:

```text
Core v0 is single-active-state.

No state stack.

No child machines.

No mailbox.

No built-in blackboard.

No policy memory.

No hidden scheduler.

No hidden heap.
```

## 4. Machine declaration syntax

Initial source shape:

```cpp
machine Name(params) -> ReturnType {
    state Start {
        ...
    }

    state Other {
        ...
    }
}
```

Example:

```cpp
machine DoorMachine(mut DoorInput& input) -> DoorResult {
    state Closed {
        transition match (input.command) {
            DoorCommand::Open => Opening;
            _                 => Closed;
        };
    }

    state Opening {
        return DoorResult.Opening;
    }
}
```

Rules:

- `machine` is a new declaration form.
- A machine has a name, parameters, result type, and a set of states.
- A machine must declare at least one state.
- State names are scoped to the machine.
- Machine parameters follow normal function parameter rules.
- Machine result types follow normal function return type rules.
- Machine effect annotations should be possible eventually.

Recommended v0 initial-state rule: the first declared state is the initial
state. This keeps the first implementation small and matches the visual reading
order of a state machine. An explicit `initial StateName;` declaration remains a
future option if later examples show that separating source order from initial
state materially improves readability or generated diagnostics.

Future effect shape:

```cpp
noalloc machine Lexer(mut LexerInput& input) -> Token {
    ...
}
```

P13-M0 reserves that direction only. It does not require implementing effect
syntax on machines.

## 5. State declaration syntax

```cpp
state Name {
    statements
}
```

Rules:

- State names must be unique within the machine.
- State bodies may contain ordinary statements plus machine-control statements.
- `transition` targets must refer to states in the same machine.
- `return` completes the machine.
- Local variables that do not cross step boundaries are ordinary locals.
- Later phases decide lifted locals for `suspend`/`yield`.
- In v0, there are no nested state declarations.
- In v0, there is no state stack.

## 6. Transition statement

Core literal syntax:

```cpp
transition TargetState;
```

Generalized syntax:

```cpp
transition stateExpression;
```

The `stateExpression` must evaluate to a valid state target of the current
machine.

Examples:

```cpp
transition Idle;
```

```cpp
transition match (event) {
    Event::Tick => Running;
    Event::Stop => Stopped;
    _          => Error;
};
```

```cpp
transition decide {
    Attack when canAttack score attackScore;
    Heal   when canHeal   score healScore;
    Idle                      score 0;
};
```

Rules:

- `transition` is only valid inside a machine state body.
- `transition` ends the current step.
- Code after an unconditional transition in the same block is unreachable.
- The transition target must be a state of the current machine.
- `transition` does not return a value to the machine caller.
- `transition` does not complete the machine; `return` completes the machine.
- Transition expressions are side-effect checked normally.
- Transition via `decide` uses existing `decide` semantics where possible.

## 7. Machine state target type

Each machine has an internal closed state target type. State names are
constructors or values of that type inside the machine. `transition` expects
this state target type.

Conceptual lowering:

```cpp
enum LexerMachineState {
    Start,
    Number,
    Identifier,
    Done,
    Error
};
```

Source code does not need to expose this enum directly in M0.

Rules:

- State target values are closed over declared states.
- Unknown state targets are compile errors.
- Duplicate state names are compile errors.
- State target values should not be confused with ordinary enums unless
  deliberately exposed later.
- `match` and `decide` transition expressions must produce the machine state
  target type.

## 8. Literal-state transitions

Literal-state transitions are the recommended first implementation target:

```cpp
state Start {
    transition Done;
}
```

Validation:

- `Done` must be a declared state.
- An unknown state is a diagnostic.
- `transition` must appear inside a state body.
- `transition` outside a machine is a diagnostic.
- Duplicate state names are a diagnostic.
- A machine with no states is a diagnostic.

Suggested diagnostics:

```text
CON0220 MachineRequiresState
CON0221 DuplicateMachineState
CON0222 UnknownMachineState
CON0223 TransitionOutsideMachineState
CON0224 InvalidTransitionTarget
```

## 9. Match-driven transitions

```cpp
state Start {
    transition match (input.peekClass()) {
        CharClass::Digit  => Number;
        CharClass::Letter => Identifier;
        CharClass::End    => Done;
        _                 => Error;
    };
}
```

Rules:

- The `match` expression must produce the machine state target type.
- Each arm target must be a valid state target.
- Existing `match` exhaustiveness and type rules apply.
- `match` is exact deterministic transition selection.
- `match` is the degenerate deterministic form of transition decision.

P13-M4 implementation status:

- Stage 0 parses `transition match (...) { ... };` inside machine state bodies.
- Match-driven transition targets are represented in the AST/scaffold.
- Arm patterns reuse existing match-pattern syntax.
- Each arm result is currently a bare machine state name, not an arbitrary
  state-valued expression.
- Arm targets validate against the containing machine's local state table.
- Unknown or cross-machine arm targets report `CON0222 UnknownMachineState`.
- Valid match transitions are represented in HIR after P13-M6. Runtime lowering
  remains deferred after P13-M8, and backend execution attempts report `CON0231
  MachineSemanticsNotImplemented` for the unsupported transition form.
- `transition decide` remains deferred to P13-M5.
- Full transition-match exhaustiveness/type checking remains deferred until the
  machine-local state target type is modeled in semantic lowering.

## 10. Decide-driven transitions

Possible explicit form:

```cpp
state Decide {
    transition decide BrainState {
        Attack when canAttack score attackScore;
        Heal   when canHeal   score healScore;
        Flee   when lowHealth score fleeScore;
        Idle                      score 0;
    };
}
```

Preferred contextual form:

```cpp
state Decide {
    transition decide {
        Attack when canAttack score attackScore;
        Heal   when canHeal   score healScore;
        Idle                      score 0;
    };
}
```

Existing `decide` likely names an enum or target type. For machine transitions,
`decide` may need to name the machine state target type or infer it from the
`transition` context.

Recommended v0 design: `transition decide { ... };` infers the machine state
target type from the transition context if feasible. `transition decide
MachineState { ... };` is reserved as an explicit form if the machine state
target type becomes nameable.

Rules:

- `decide` cases produce state targets.
- Conditions must be `bool`.
- Scores follow existing `decide` rules.
- Highest score wins according to existing `decide` semantics.
- `decide` is utility-scored transition selection.
- `decide` remains stateless in core.
- Hysteresis, `min_commit`, and policy memory are not core machine features.

P13-M0 does not implement decide-driven transitions and does not add new
`decide` semantics.

P13-M5 implementation status:

- Stage 0 parses `transition decide { ... };` inside machine state bodies.
- Decide-driven transition targets are represented in the AST/scaffold.
- Decide cases preserve source order, bare target state names/spans, optional
  `when` condition expressions, and required `score` expressions.
- Each case result is currently a bare machine state name, not an arbitrary
  state-valued expression.
- Candidate targets validate against the containing machine's local state
  table.
- Unknown or cross-machine candidate targets report `CON0222
  UnknownMachineState`.
- Valid decide transitions are represented in HIR after P13-M6. Runtime
  lowering remains deferred after P13-M8, and backend execution attempts report
  `CON0231 MachineSemanticsNotImplemented` for the unsupported transition form.
- Condition and score expression typing inside transition-decide is deferred
  until machine bodies lower/typecheck through the normal executable path.
- `decide` remains utility-scored transition selection and remains stateless in
  Concept core.
- Policy memory, hysteresis, `min_commit`, stack HFSM, blackboards, mailbox
  buses, actuators, persistence, a core `board` keyword, hidden heap behavior,
  scheduler behavior, and DragonGod kernel features remain outside core Phase
  13 M5.

## 11. Machine-local memory

Concept does not add a `board` keyword.

```text
Concept has mutable structs and places.
Machine-local/control memory should use ordinary structs, fields, parameters,
and later machine frame fields.
DragonGod may define blackboard conventions in its kernel.
```

Example:

```cpp
struct BrainBoard {
    int lastChoice;
    int commitTicks;
    bool targetVisible;
};

machine Brain(mut Context& ctx, mut BrainBoard& board) -> Action {
    state Decide {
        transition decide {
            Attack when board.targetVisible score ctx.attackScore;
            Idle                            score 0;
        };
    }

    state Attack {
        board.lastChoice = 1;
        return Action.Attack;
    }

    state Idle {
        board.lastChoice = 0;
        return Action.Idle;
    }
}
```

Clarifications:

- There is no special `board` keyword in Concept core.
- There is no dynamic blackboard in Concept core.
- There is no mailbox/event bus in Concept core.
- There is no dirty-key tracking in Concept core.

## 12. Step/resume/completion model

Conceptual runtime model:

```text
A machine instance stores current state and captured parameters/fields.

Step(machine) executes the current state until transition, return, or suspend.

transition changes current state and ends the step.

return completes the machine and stores the result.

suspend/yield ends the step without changing state.
```

P13-M7 builtins:

```text
Step(machine)
Complete(machine)
Result(machine)
```

`Active(machine)` remains deferred because user-visible state-name/string
support is not part of the current runtime subset. `Step`, `Complete`, and
`Result` resemble Octomata inspection concepts, but Concept keeps them tied to
explicit frame storage and compiler-lowered machine semantics rather than a
hidden scheduler or behavior kernel.

P13-M8 behavior:

- `Step(machine)` executes one active state and returns no value.
- `Complete(machine)` reads the explicit completion flag and is false before a
  machine returns.
- `Result(machine)` reads the stored result only after completion.
- `Result(machine)` before completion lowers to an explicit generated C trap
  instead of silently reading raw result storage.
- Calling `Step(machine)` after completion is a no-op.
- The supported executable parameter/result storage remains scalar `int` and
  `bool`.

Rules and constraints:

- There is no implicit scheduler.
- There is no hidden heap.
- Machine instances are explicit values/places.
- State frame storage is explicit or lowerable.
- P13-M7 step/resume runtime v0 supports literal transitions and simple
  return/completion paths.

## 13. Effects and allocation

Machines participate in effects.

Future syntax example:

```cpp
noalloc machine Lexer(mut LexerInput& input) -> Token {
    ...
}
```

Rules:

- A `noalloc machine` must not allocate.
- Machine state bodies obey ordinary effect rules.
- `transition` itself does not allocate.
- `decide` and `match` inside transition obey ordinary effect and type rules.
- There is no hidden machine heap allocation.
- Machine frame allocation/storage is explicit at the construction site or
  lowering site.

P13-M0 does not implement effect checking for machines. It documents the
planned interaction so machine work does not drift away from Phase 12's
allocation doctrine.

## 14. Ownership, Drop, and lifted locals

Machine lowering must preserve ownership and Drop semantics.

Design direction:

- Locals that do not cross step boundaries are ordinary locals.
- Locals that survive across `suspend`/`yield` must be lifted into machine frame
  storage.
- Lifted locals need storage-state tracking.
- Dropped/transitioned states must not leak resources.
- References crossing `yield`/`suspend` are restricted unless proven safe.
- Immovable machine frames may be needed for self-referential or
  lifted-reference cases.

P13-M0 does not implement this hardening. It records the future constraints
early because machines will eventually touch the same resource invariants as
ownership, Drop, and ManualInit.

## 15. HIR/MIR lowering direction

Conceptual lowered frame:

```cpp
struct LexerMachineFrame {
    LexerMachineState current;
    LexerInput* input;
    bool complete;
    Token result;
};
```

Resume/step lowering:

```text
Step(frame):
  switch frame.current:
    Start -> bb_start
    Number -> bb_number
    Identifier -> bb_identifier
    Done -> bb_done
    Error -> bb_error
```

Transition lowering:

```text
transition Target:
  frame.current = Target
  return/suspend step
```

Return lowering:

```text
frame.complete = true
frame.result = value
return
```

MIR responsibilities:

- machine state enum/tag;
- machine frame;
- state dispatch switch;
- transition terminators;
- yield/suspend terminators later;
- Drop/lifted-local cleanup;
- effect boundaries.

The important requirement is inspectability. Machine syntax should lower to
visible HIR/MIR constructs rather than becoming opaque runtime magic.

P13-M8 generated C shape for the executable literal-transition subset includes
one machine-local state enum, one frame struct with `state`, `complete`,
`result`, and scalar captured parameter fields, a constructor helper that
initializes the initial state and `complete = false`, and a step helper that
dispatches on the current state. Literal transitions assign the next state and
return from the step. Machine returns store the result, set `complete = true`,
and return from the step. Completed machine steps return immediately. Generated
machine C remains free of `malloc`, hidden heap helpers, scheduler helpers, and
async runtime helpers.

## 16. Diagnostics

P13-M0 only reserves a diagnostic inventory:

```text
CON0220 MachineRequiresState
CON0221 DuplicateMachineState
CON0222 UnknownMachineState
CON0223 TransitionOutsideMachineState
CON0224 InvalidTransitionTarget
CON0225 MachineStateUnreachable
CON0226 MachineStateMissingReturnOrTransition
CON0227 MachineReturnTypeMismatch
CON0228 MachineTransitionExpressionTypeMismatch
CON0229 MachineYieldReferenceEscape
CON0230 MachineEffectViolation
```

Later implementation milestones may adjust numbering if the existing diagnostic
space changes.

## 17. Non-goals for Phase 13 v0

Phase 13 v0 explicitly defers:

- stack HFSM;
- nested/child machines;
- DragonGod behavior kernel;
- blackboard keyword;
- typed mailbox event bus;
- actuators;
- dirty-key tracking;
- persistence;
- hysteresis;
- `min_commit`;
- policy memory;
- built-in utility consideration framework beyond existing `decide`;
- runtime scheduler;
- async/await;
- coroutine heap frames;
- hidden heap allocation;
- machine reflection/debug visualization;
- state history builtins;
- resume target slots;
- timeline inspection;
- distributed/evented actor model;
- parallel state regions;
- hierarchical state syntax;
- entry/exit actions unless explicitly designed later;
- state inheritance;
- UML-style statechart complexity.

## 18. Examples

Runnable P13-M8 examples live in `examples/phase13/`. The runnable files use
literal transitions only. The match/decide transition model example documents
syntax and validation, but is not a backend-runtime example yet.

### A. Runnable literal transition

```cpp
machine Door() -> int {
    state Closed {
        transition Open;
    }

    state Open {
        return 1;
    }
}
```

### B. Lexer deterministic match transition

```cpp
machine Lexer(mut LexerInput& input) -> Token {
    state Start {
        transition match (input.peekClass()) {
            CharClass::Digit  => Number;
            CharClass::Letter => Identifier;
            CharClass::End    => Done;
            _                 => Error;
        };
    }

    state Number { return input.readNumber(); }
    state Identifier { return input.readIdentifier(); }
    state Done { return Token.End; }
    state Error { return Token.Error; }
}
```

### C. Utility AI decide transition

```cpp
machine Brain(mut Context& ctx) -> Action {
    state Decide {
        transition decide {
            Attack when ctx.canAttack score ctx.attackScore;
            Heal   when ctx.canHeal   score ctx.healScore;
            Flee   when ctx.lowHealth score ctx.fleeScore;
            Idle                         score 0;
        };
    }

    state Attack { return Action.Attack; }
    state Heal { return Action.Heal; }
    state Flee { return Action.Flee; }
    state Idle { return Action.Idle; }
}
```

### D. DragonGod boundary

```cpp
struct BrainBoard {
    int lastChoice;
    int commitTicks;
};

machine Brain(mut Context& ctx, mut BrainBoard& board) -> Action {
    state Decide {
        transition decide {
            Attack when ctx.canAttack score ctx.attackScore;
            Idle                         score 0;
        };
    }

    state Attack {
        board.lastChoice = 1;
        return Action.Attack;
    }

    state Idle {
        board.lastChoice = 0;
        return Action.Idle;
    }
}
```

`board` is an ordinary struct parameter, not core syntax. The DragonGod kernel
can build richer blackboard and policy behavior on top of Concept machines,
ordinary storage, and `decide`.

## 19. Milestone plan

Proposed Phase 13 milestones:

```text
P13-M0  Design doc: machines and decision-driven transitions
P13-M1  machine/state parser + AST scaffold
P13-M2  state symbol table and validation
P13-M3  transition statement with literal state target
P13-M4  transition target expressions via match
P13-M5  transition target expressions via decide
P13-M6  HIR/MIR lowering scaffold
P13-M7  step/resume runtime model v0
P13-M8  examples, fixtures, runtime hardening, generated C shape assertions
P13-M9  Closeout
```

The ordering starts with syntax and validation, then expands transition
expressions, then lowers to HIR/MIR and runtime stepping. That keeps the first
implementation path narrow while preserving the doctrine: explicit machines,
typed transitions, no hidden heap, no implicit scheduler, and DragonGod outside
the core language.

## 20. Docs index / PoC status

Phase 13 should be recorded in the PoC status as a planned/in-progress
machine-design phase. P13-M0 is docs-only and should not claim parser,
lowering, runtime, or DragonGod implementation status.

## 21. Final checks

Even though P13-M0 is docs-only, completion should run:

```bash
zig version
zig build test
git diff --check
git status --short
```

Broad formatting should not be run if it would cause unrelated line-ending
churn.
