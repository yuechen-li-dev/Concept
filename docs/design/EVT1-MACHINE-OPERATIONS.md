# EVT1 machine operations

An architecture-qualified intrinsic has a typed Standard declaration, stable
Concept identity, and a `machine_intrinsic` MIR operation. Its privilege and
ordering classification are explicit MIR fields. The planner retains each
operation as an ordered target helper call. The current backend emits a C ABI
call and deterministic target-specific assembly; helper text is implementation,
not semantics. `concept-module.v1` carries the declaration and annotation to
artifact-only consumers.

AMD64 `Cpuid` is an ordinary typed library wrapper returning `CpuidResult`.
Its implementation exercises the multi-operand structured asm model with
fixed EAX/EBX/ECX/EDX registers. The generated helper passes a single frame
pointer across the C ABI. Each operand has an ordered eight-byte slot, a
Concept scalar type, direction, and optional fixed register. The helper
preserves RBX and its frame register, loads `in`/`inout` slots, executes one
instruction, and stores `out`/`inout` slots. The C wrapper initializes output
storage and performs typed copies. No runtime allocation or register allocator
is involved.

`NoAllocation` is derived from the compiler-known intrinsic and inline helper
paths. `HardwareRead`/`HardwareWrite` remain distinct MMIO and port access
facts. `Privileged` is derived from operations such as `Cli` and propagates
through ordinary wrappers. `MachineOrdering` specifies the exact barrier
category rather than treating all operations as a generic memory clobber.
Explicit inline-asm memory effects survive MIR and planning. A call boundary
prevents C compiler movement across target helpers; no claim is made that
`memory readwrite` by itself creates a CPU fence.

The model has no runtime intrinsic registry, target dispatcher, broad ISA
database, naked function, or GCC constraint language. EVT2 can lower these
same semantic operation IDs through LIR and MachineIR to native instructions.
