# EVT2 activation stack

EVT2x6 qualifies source push/pop for bounded, scalar-frame step machines. The
existing finite-machine Init/Step ABI is preserved. Input reactions, owned
activation fields, typed outcome payloads, and pushdown terminal settling retain
explicit native boundaries.

`planActivationStack` consumes the closed validated MIR inventory and existing
General Planner contract. It uses MIR runtime ordinals as u32 machine tags and
per-machine state IDs. Shared state is outside the eight activation slots.
Each slot contains a tag and aligned storage for the maximum reachable frame.
Geometry comes from the existing semantic type/layout machinery.

| Fixture | Machine frames | Max size / alignment | Slot stride | Automata size / alignment |
| --- | --- | --- | --- | --- |
| EVT1 `machine_parent_resume` | Parent 8, Child 4 | 8 / 4 | 12 | 108 / 4 |
| EVT2 `parent_child` | Parent 8, Child 8 | 8 / 4 | 12 | 108 / 4 |
| EVT2 `pushdown` | Parent 8, Child 8, GrandChild 12 | 12 / 4 | 16 | 140 / 4 |

Native layout: depth u32 at 0, completed bool at 4, shared value i32 at 8,
slots at 12. Each machine frame starts with current_state u32 at 0.
Parent.preserved and Child.count are i32 at frame offset 4. GrandChild.touched
is bool at 4 and GrandChild.amount is i32 at 8. Frame sizes include alignment
padding. Native and C storage geometry differ; the differential compares
logical live fields, not C struct bytes or inactive scratch storage.

> Source push/pop semantics are eliminated during MIR/LIR lowering. By verified
> LIR, nested machine activation is represented entirely as explicit typed frame
> initialization/destruction, depth bookkeeping, memory access, and control flow.

Root Init initializes shared fields, root state/fields and tag, then publishes
depth one. Step first checks completion, then positive live depth before any
subtraction. Every indexed address retains the existing check_index guard.
Unknown safety never removes guards.

Push uses depth as child index and the existing arbitrary-stride index_address.
It checks capacity before touching child storage or continuation. Child field
initializers execute in their construction scope, with shared bindings and no
caller locals. Scalar fields initialize first, then child state zero, parent
continuation state, child tag, and increased depth. Deferring the continuation
write until all initializers succeed also preserves it on initializer failure.
Push returns Active; the child first executes on the next Step.

Completion/pop validates the actual top tag through an exhaustive closed chain.
Each arm enters a typed `destroy.<machine identity>` CFG region. Scalar-frame
destruction emits no instructions; the region remains structurally verified.
Only then does the arm publish depth minus one. The parent becomes top without
copying or reconstruction, and executes on the next Step. Shared neutral
completion CFG is reused for all source sites.

> Push publishes increased depth only after child activation initialization is
> complete. Pop publishes reduced depth only after child destruction is complete.

Explicit root pop follows the C oracle: it completes at depth zero. This decision
was authorized during x6 after identifying the conflict with the initial request's
root-pop rejection rule. Incomplete depth zero still traps. Completed instances
return Completed without reading inactive slots or changing any byte.

The verifier checks executable dominance, dynamic tag/state dispatch, exact
machine field geometry, capacity guards, initialization coverage, continuation,
initial state/tag values, depth publication order, exhaustive typed destruction,
and no frame access after destruction. Provenance records identify regions;
they cannot substitute for checks on ordinary instructions.

The native backend sees only ordinary operations. Stride 12 still legalizes to
multiply plus scale-one LEA. CMIRAMD1 is unchanged. See
`../conformance/EVT2X6-CONFORMANCE.md` for execution and negative evidence.

No heap fallback, hidden return-address stack, reflection registry, scheduler,
async integration, dispatch optimization, or owned-field Drop qualification is
introduced. Structural NoAllocation evidence for generated Step is its complete
verified opcode inventory and fixed caller-owned frame; a formal generated-Step
NoAllocation fact is not currently exposed.
