# EVT1 machine operations

An architecture-qualified machine intrinsic has a typed declaration in an
ordinary library, a Concept operation identity, and a nonallocating MIR
operation. The planner retains the operation and orders it in the source
operation stream. The current backend emits a C ABI call and an inspectable
architecture helper. The helper is an implementation of the operation, not
its definition. Semantic module artifacts carry the typed declaration and
machine annotation for source-independent consumers.

`HardwareRead` is derived for timestamp and port input, and `HardwareWrite`
for port output, through ordinary transitive call analysis. The `NoAllocation`
proof accepts compiler-known machine declarations and their wrappers. No
ordinary-memory fence or CPU barrier is claimed by these operations.

The first structured `unsafe asm AMD64` statement carries one typed register
operand, explicit clobbers, and an explicit memory effect through parsing,
semantic validation, MIR, artifacts, planning, and helper generation. The
backend conservatively uses a separate call boundary to retain ordering;
there is no runtime dispatcher or GCC constraint syntax in Concept source.
Multiple operands and deeper register-clobber validation remain separate
work. Privileged operations also require an explicit privilege fact or
authority rule before they can be classified honestly.
