# EVT2x native automata lowering

EVT2x is a sequence break after EVT2d and before EVT2e. This document records
the existing semantic authority and native-lowering boundary. Finite machine
Init and Step execute natively; pushdown Step does not yet execute natively.

> Concept pushdown automata execute as a bounded stack of independently live,
> machine-specific activations. Each activation has a stable machine tag and
> its own typed persistent frame storage.

This remains the overall EVT2x target. EVT2x2 established the finite-state
MachineFrame and verified Step LIR. EVT2x3 executed that subset in AMD64.
EVT2x4 has a closed activation inventory, exact bounded storage layout, and
native root Init, while dynamic Step push/pop remains unsupported.

## Existing semantics to preserve

The canonical EVT1 automata model has one shared `with state` environment,
independently named machines, and named states inside each machine. State
ordinals are assigned within each machine in declaration order. The closed
identity is the automata graph identity plus machine runtime ordinal plus state
runtime ordinal; source spelling alone is insufficient. A `Step` executes only
the top active machine frame. Each frame has its own current state and machine
persistent fields; state-body locals are transient. A transition changes the
active frame's state. `push Child goto Resume` stores the caller frame with its
explicit `Resume` state, then activates an initialized child. Completion or
pop removes the child and exposes its typed outcome. Bare `yield;` returns from
the step; the next step re-enters the same state from its beginning. It does
not save an instruction pointer. Completion at the root marks the instance
completed, and later steps are no-ops.

The EVT1 C11 oracle uses caller-owned instance storage: shared fields, a
`uint8_t` depth, eight machine tags, completion status, and specialized
fixed arrays of eight frames per machine. Each frame has a `uint8_t`
`current_state` and its declared persistent fields. It guards pushes against
capacity eight and aborts on overflow. Validation rejects a root pop and a
second pop in one path. Invalid state dispatch aborts. The native design must
preserve the same semantic results; the exact C struct geometry is not an ABI
contract for native code.

MIR already records machine and state ordinals, graph identity, stack capacity,
storage classifications, control summaries, yields, and semantic facts. EVT2x
now also retains the validated state body and persistent field initializer in
memory, using the same `json:"-"` pattern as ordinary function bodies.
Serialized checked MIR remains a summary. Native lowering can consume typed
MIR without parsing either debug text or generated C.

## EVT2x2 lowering boundary

`concept lir` now lowers supported single-machine state bodies into generated
Init and Step functions with a caller-owned frame address parameter, typed
field addresses, loads/stores, compare/branch dispatch, explicit returns, and
an invalid-state trap. The generated function and state blocks carry stable
closed-machine provenance. See `EVT2-MACHINE-FRAME.md` for exact geometry and
Step results. Push/pop and multiple-machine execution diagnose explicitly.
`concept machineir` lowers this finite subset through CMIRAMD1 and the
Concept-written allocator and encoder. The pushdown case still diagnoses
`EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP` at LIR entry. See
`EVT2-ACTIVATION-STACK.md` for the exact new layout and native root Init.

The existing fixed capacity is a storage bound, not a per-step execution
bound. A proven static push depth may permit a Planner-authorized omitted
guard; unknown safety retains a guard. Native NoAllocation and Bounded claims
must be established for the actual step path, not inferred from C emission or
from an allocating test harness. A bounded machine can execute with
caller-owned fixed frame storage and no heap allocation once that path is
qualified.

> Future async lowering should reuse this explicit persistent state/frame model
> where semantically appropriate rather than inventing an unrelated runtime
> continuation system.

Explicit state structure may later support profiled transition traces. EVT2x
does not implement a scheduler, EventBus, async lowering, JIT, state optimizer,
or trace specialization.
