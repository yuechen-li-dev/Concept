# Machine intrinsics (R7l progression)

Use typed machine intrinsics when an operation has known semantics.

Use inline assembly only when the instruction sequence itself is the point.

The current architecture namespace is the ordinary module
`Standard.Machine.AMD64`. Import it and call `Pause()`, `ReadTimestamp()`,
`In8/16/32(port)`, or `Out8/16/32(port, value)`. Concept's present package
syntax imports a module and then uses its exported names without a dotted call
qualifier. `uint16` is the port number. `ReadTimestamp` reads a processor
counter; it is not a wall clock. `Pause` is a processor hint. Port I/O is
privileged on ordinary hosted operating systems and must not be executed by
an ordinary user-mode test.

MMIO and port I/O are different machine mechanisms. Concept represents them
differently. MMIO uses `Address<DeviceMemory>` and `MmioLoad/Store`; port I/O
uses a `uint16` port and typed `In/Out` operations. The DragonGod MMIO UART
and AMD64 port UART retain that difference in ordinary library code.

The compiler-owned `[[machine("AMD64.*")]]` annotation is restricted to
exact signatures in `Standard.Machine.AMD64`. It records a stable operation
identity in MIR and the semantic module artifact. It is not a user-facing
assembly constraint language. Each operation has a deterministic generated
`.machine.S` implementation; ordinary generated C remains strict C11 and
calls a typed C ABI symbol. The helper is emitted only for modules containing
machine operations. No runtime registry or dispatcher is involved.

R7l machine operations are Concept semantics. Current C/compiler-helper
lowering is an implementation strategy, not the semantic definition. EVT2
native backends will lower the same operations directly.

This implementation is a bounded progression: CPUID, privileged machine-state
instructions, and AArch64 operations are not yet available. A one-operand
structured AMD64 asm escape hatch is documented in `INLINE-ASSEMBLY.md`.
The generic C11 planning target uses the current host architecture
for machine-helper eligibility; an explicit AArch64 target rejects AMD64
operations. A future target contract should encode operating-system ABI and
object format before broader cross-compilation is claimed.
