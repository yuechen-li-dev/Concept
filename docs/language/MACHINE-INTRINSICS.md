# Machine intrinsics

If Concept knows what the machine operation means, use a typed intrinsic.
Inline assembly exists only for operations or sequences the language does not
otherwise model.

`Standard.Machine.AMD64` exports `Pause`, `ReadTimestamp`, 8/16/32-bit
`In`/`Out`, `Cpuid`, `Cli`, `Sti`, `Hlt`, `Lfence`, `Sfence`, and `Mfence`.
`Cpuid(uint32 leaf, uint32 subleaf)` returns a complete `CpuidResult` record
with `eax`, `ebx`, `ecx`, and `edx` fields. Its ordinary typed wrapper uses
fixed-register structured asm internally. `ReadTimestamp` observes a CPU
counter and has no wall-clock promise. `Pause` is a processor hint.

`Standard.Machine.AArch64` exports `Yield`, `Wfi`, `Dmb`, `Dsb`, and `Isb`.
The barrier identities retain distinct ordering properties in MIR:
AMD64 load, store, and full fences; AArch64 inner-shareable memory order,
completion, and instruction-stream synchronization. These operations are
preserved by the planner. They are not aliases for a compiler-only fence.
The current backend emits deterministic target assembly helpers, and Clang
cross-assembles the AArch64 helper on the AMD64 development host. This is a
code-generation check, not native AArch64 execution.

`Cli`, `Sti`, and `Hlt` carry `Privileged` as a semantic fact; AArch64 `Wfi`
is conservatively classified the same way for the current target profile.
Privilege is distinct from memory unsafety. Hosted tests inspect MIR, proofs,
and object generation without executing privileged instructions. The current
profile has no privilege authority gate.

MMIO and port I/O are different machine mechanisms. Concept represents them
differently. MMIO uses `Address<DeviceMemory>` with `MmioLoad/Store`; AMD64
port I/O uses a `uint16` port number with typed `In/Out`. DragonGod retains
separate MMIO and port UART libraries and uses `Pause` in the port poll loop.

Compiler-owned `[[machine("Architecture.Operation")]]` annotations are
restricted to exact signatures in the matching Standard module. MIR and
`concept-module.v1` retain their typed identity; the C11 backend emits only
used, inspectable `.machine.S` helpers. Generated C remains strict C11. There
is no runtime operation registry or dispatcher.

R7l machine operations are Concept semantics. Current C/compiler-helper
lowering is an implementation strategy, not the semantic definition. EVT2
native backends will lower the same operations directly through MachineIR.
