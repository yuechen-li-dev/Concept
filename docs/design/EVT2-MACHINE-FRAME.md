# EVT2x2 MachineFrame and Step LIR

> A native machine instance is represented by caller-owned fixed frame storage.
> The frame contains explicit current state and persistent machine fields.
> Yield does not allocate a continuation object.

EVT2x2 supports one canonical machine in an automata declaration, with scalar
`int`, `uint`, and `bool` shared/machine fields. The frame layout is computed by
the same `evt1StructFieldOffsets` / `evt1TypeGeometry` authority used by
`SizeOf` and `AlignOf`. Field order is `current_state: uint32`,
`completed: bool`, shared `with state` fields in declaration order, then
machine-persistent fields in declaration order. Each field has a checked
offset, size, and alignment. For `finite.concept`, the offsets are 0, 4, 8,
and 12; size 16, alignment 4. Source state IDs are the existing per-machine
MIR runtime ordinals, scoped by the automata graph identity and machine name.

Generated `Counter.Run$init` accepts the frame address and shared constructor
values. It stores state ID zero, `completed=false`, the shared values, and
the machine field's preserved initializer (or its scalar zero value).
Generated `Counter.Run$step` accepts the same frame address. Each instance
therefore has independent storage. Calling Step before Init violates the
caller contract. Neither function uses compiler-global machine state.

Step first checks `completed`; a completed frame returns `Completed` without
mutation. Otherwise it loads `current_state` and follows a deterministic
compare/branch chain to exactly one source state. An unrecognized state reaches
an explicit `invalid_machine_state` trap block. A `transition Target` stores
the target ID and ends this Step; the target body runs on the next Step.
`complete;` stores `completed=true` and returns `Completed`. A state body
falling through leaves its state unchanged and returns `Active`.

**EVT1 yield semantics:** bare `yield;` stores the current source state ID and
returns `Yielded`. A later Step dispatches to the *beginning of the same source
state*. Statements after the yield do not run in that invocation. They may run
on a later invocation only if the re-entered state's control flow reaches
them. There is no saved instruction pointer and no generated after-yield
substate. The guarded `yield_resume.concept` and `multi_yield.concept`
fixtures and C11 oracle pin this rule. This is the actual language behavior;
splitting into after-yield continuation states would change it.

`machine_step_result` is a payload-free logical LIR type with fixed four-byte
enum geometry and symbolic `Active`, `Yielded`, and `Completed` members.
`Active` means a transition or ordinary state step ended while the machine
remains live. `Failed` is not materialized: the existing invalid-state
behavior is a trap. EVT2x3 maps the symbolic LIR results to 0/1/2 in EAX
and executes finite Init/Step through the Concept-written AMD64 backend.

The verifier checks frame pointer identity, field geometry and offsets,
closed state IDs and dispatch targets, frame initialization, saved state on
live returns, completion status before completed returns, and an invalid-state
trap. The Init/Step body uses ordinary LIR address, load, store, arithmetic,
branch, and return operations. The two new LIR operations are
`frame_field_address` and symbolic `machine_result`. MachineIR now folds
fixed frame-field offsets into ordinary memory operands and lowers the result
to an integer return; CMIRAMD1 remains unchanged.

The supported LIR has no heap operation. A generated Step is compiler-internal,
so `Assert.Concept` cannot currently address it; a formal NoAllocation proof
is not claimed. MIR and Planner retain their existing machine bounds, which
are not conflated with the frame byte size or a per-step execution bound.
The `concept-module.v1` semantic payload transports the typed automata AST,
including state bodies and persistent initializers. A test decodes a real
artifact and lowers its closed machine to the same Init/Step LIR. Import
composition into a separate consumer is not yet part of this fixture lane.

> Pushdown needs tagged, independently live machine activations. EVT2x4 now
> computes their closed inventory and exact bounded storage layout and lowers
> root Init; dynamic Step, push, and pop remain pending. See
> `EVT2-ACTIVATION-STACK.md`.

> Future async lowering should reuse the persistent-frame/state mechanism
> where semantically compatible.
