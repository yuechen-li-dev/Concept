# EVT2 activation-stack layout and current lowering boundary

Concept pushdown automata require a bounded stack of independently live,
machine-specific activations. A machine tag is distinct from the state ID
inside its frame. `push Child goto Resume` saves the parent frame's `Resume`
state, constructs a fresh Child frame, and makes it top; child completion or
`pop` removes that frame and exposes the unchanged parent frame. Bare `yield`
re-enters the beginning of the same top machine state on the next Step.

`planActivationStack` now consumes the closed, validated MIR machine graph.
It uses stable MIR machine runtime ordinals as `u32` tags and per-machine state
ordinals as `u32` state IDs. The capacity is the existing eight live frames.
Depth means the number of live activations, so root Init publishes depth one.
The layout computes each reachable machine's frame with Concept's
`evt1StructFieldOffsets` and `evt1TypeGeometry`, then computes maximum frame
size and alignment. A slot reserves a tag and aligned storage for that
maximum. Shared state stays outside the activation slots.

| Fixture | Machine frame sizes | Max frame | Slot | Total caller-owned frame |
| --- | --- | --- | --- | --- |
| `machine_parent_resume` | Parent 8, Child 4 bytes | 8 bytes, align 4 | 12 bytes | 108 bytes |
| `machine_multiple_nested_frames` | Parent 4, Child 4, Grandchild 4 bytes | 4 bytes, align 4 | 8 bytes | 76 bytes |
| `machine_recursive_frames` | Run 8 bytes | 8 bytes, align 4 | 12 bytes | 108 bytes |

The generated root Init currently supports scalar `int`, `uint`, and `bool`
persistent fields. It stores shared constructor arguments, the root state and
machine fields, then the root tag, and **publishes depth last**. Its verified
LIR uses typed scalar subobject addresses and ordinary stores. The CMIRAMD1
bridge is unchanged, and a native Windows AMD64 harness calls the encoded
Init against the 108-byte parent/child frame. This establishes root storage
geometry and initialization, not native push/pop execution.

LIR `index_address` now also accepts a guarded pointer base, an arbitrary
positive constant byte stride, and a fixed byte offset. Its verifier requires
the matching index guard and bounds the largest displacement before MachineIR.
The AMD64 MachineIR lowerer emits a multiply and scale-one LEA for stride 12;
the Concept allocator and encoder still see only ordinary operations. A native
test calculates `top = depth - 1`, guards it, and loads tags from slots zero,
one, and two with the real 12-byte stride. A second native test writes those
slots while checking surrounding sentinels. These are direct LIR/backend
probes, not a source `Step` implementation.

Activation layout is semantic and independent of target addressing modes.
Non-power-of-two activation strides are legalized during MachineIR lowering;
AMD64 SIB scale restrictions do not affect frame layout.

The next lowering step is to connect this address path to generated Step,
dispatch machine tags and states, then lower guarded child construction and
typed destruction on pop. Current `GenerateLIR` still rejects push with
`EVT2_UNSUPPORTED_AUTOMATA_PUSH_POP`. In particular, scalar root stores do
not establish general `Storage<T>` lifetime semantics for owned child frames;
those remain an explicit native boundary. The existing C backend is the
pushdown behavior oracle and already uses fixed specialized frame arrays.

The required publication contract is: push constructs a complete typed child
activation before publishing increased depth; pop destroys the live child
activation before publishing reduced depth. `machineTag` identifies the live
machine-frame type in an activation slot. The slot is raw storage when
inactive, typed live storage while active, and is never used as union-style
type punning. These push/pop contracts remain to be established in generated
native Step.

The intended activation stack is fixed-capacity caller-owned storage. It has
no heap fallback at overflow. Before MachineIR, push/pop must become ordinary
bounded memory and control flow, leaving the Concept allocator and AMD64
encoder unaware of automata. That latter contract is a target, not a claim
about the current Step implementation.
