# Phase 13 Machines

Phase 13 introduces Concept core machines as compiler-visible control frames.
The closed v0 runnable subset is intentionally small: single-active-state
machine frames with scalar `int` and `bool` parameters/results, explicit
`MachineName(...)` construction, `Step(machine)`, `Complete(machine)`, and
`Result(machine)`.

Literal transitions are runnable:

```cpp
state Closed {
    transition Open;
}
```

`Step(machine);` executes the active state until a transition or return and is
statement-like: it produces no usable value. It cannot be assigned, returned,
used as a condition, passed as an argument, or used in an expression. `transition
Target;` updates the current state and ends the step. `return value` stores the
result, marks the frame complete, and ends the step. Extra `Step(machine)` calls
after completion are no-ops. `Complete(machine)` returns `bool`.
`Result(machine)` returns the machine result type and is only valid after
completion; the generated C backend traps incomplete result access.

`transition match (...) { ... };` and `transition decide { ... };` are syntax,
validation, and HIR scaffolds in this milestone. They validate machine-local
state targets, but runtime lowering remains deferred. Match examples use the
current C++-shaped syntax, including `transition match (kind) { ... };`,
`TokenKind::Identifier`-style enum qualification, and `_ => Target` default
arms.

Machine frames are explicit value storage. Generated C uses a state enum, frame
struct, constructor helper, and step helper. There is no hidden heap, `malloc`,
scheduler, async runtime, stack HFSM, `board` keyword, blackboard, mailbox bus,
actuator layer, hysteresis, `min_commit`, or policy memory in Concept core v0.
Boards/blackboards are ordinary structs and library conventions in Concept.
Those behavior architecture pieces belong above the core language, for example
in DragonGod, built on top of Concept machines rather than inside core v0.

Files:

- `door-machine.concept`: runnable literal-transition machine returning `int`.
- `toggle-machine.concept`: runnable completion/no-op behavior returning
  `bool`.
- `machine-transition-model.concept`: design-only match/decide syntax model;
  not a runnable backend example yet.
