# DragonGod direction

R7a replaces archaeology with one active library: `libraries/DragonGod`.
DragonGod is optional kernel/platform substrate built in Concept. It is not a
compiler runtime, and the compiler never assumes the package exists.

## Historical inventory

Implementation A was the pre-EVT1 `dragon-god/` PoC kernel. It contributed IDs,
reason/status vocabulary, a tick clock, fixed-slot Memory, Automata stack and
signals, Mind, Decision, Actuation, Events, Graph, Trace, Replay, and checkpoint
experiments. Its source used obsolete syntax and encoded scheduler-adjacent agent
policy. It now lives only at `legacy/dragon-god-poc` and no active build reads it.

Implementation B was the EVT1 Concept/Vulkan DragonGod lineage in
`examples/evt1/evt1_dragongod_m*_language.concept` and matching Vulkan examples.
It contributed current automata/effect semantics, runtime/platform boundary
evidence, and strict generated artifacts. Those files remain conformance examples,
not production authority. Vulkan lifecycle and device policy remain consumers;
no Vulkan name appears in canonical kernel APIs.

| Historical capability | Decision | R7a disposition |
|---|---|---|
| IDs, status, execution context | Rewrite | `DragonGod.Core.Types` |
| integer tick clock | Rewrite | unit-typed `DragonGod.Time.Core` |
| backing memory authority | Replace | firmware source over `Standard.Memory` |
| private/fixed-slot allocator ideas | Delete | no second allocator framework |
| basic machine mode/halt vocabulary | Rewrite | `DragonGod.Machine.Core` |
| interrupt wakeup vocabulary | Rewrite | `DragonGod.Interrupts.Core` |
| AMD64 capability facts | Keep/rewrite | `DragonGod.Platform.AMD64.Core` |
| AArch64 capability facts | Keep/rewrite | `DragonGod.Platform.AArch64.Core` |
| Automata saved-PC/workaround machinery | Delete | canonical R5 machine semantics remain language-owned |
| Mind, Actuation, Events, Trace, Replay, checkpoint | Rewrite | canonical configurable R7b mechanisms under `DragonGod.*` |
| Decision and Graph policy | Replace boundary | Concept semantics and application-owned meaning |
| scheduler policy and ready queues | Rewrite | R7c `DragonGod.Scheduling` |
| synchronization | Build incrementally | R7d strict-C11 atomics exist; shared-authority proofs remain |
| collector/root graph | Defer | after synchronization |
| Vulkan device/lifecycle policy | Replace boundary | stays in Vulkan consumers/conformance |

Historical tests that prove current language semantics remain in the legacy
oracle or EVT1 corpus. PoC tests tied only to fixed slots, old mutation spelling,
or saved-PC workarounds are archival and are not migrated into the active package.

## Canonical architecture

- `Core` owns processor identity, kernel status, and execution context.
- `Memory` owns firmware backing adaptation, `BootInfo`, and a representative
  kernel object. Allocation policy is entirely `Standard.Memory`.
- `Machine` owns architecture-neutral execution modes and explicit halt state.
- `Interrupts` owns a deterministic value-shaped wakeup fact; no registry exists.
- `Time` owns monotonic time and deadline vocabulary using the supported `s`
  quantity. Subsecond units await general scale-prefix support.
- `Platform.AMD64` and `Platform.AArch64` expose only honest current capability
  records. They do not fake complete ports.
- `Scheduling` owns explicit context authority, a bounded FIFO of IDs, semantic
  step quanta, wait/sleep readiness, and structured decisions.
- `Synchronization` begins with `Standard.Synchronization.Atomic`; the
  authoritative scheduler remains single-worker because shared multi-worker
  authority and race freedom cannot yet be proven.

The BootInfo path is firmware backing -> `MemoryRegion<SystemMemory>` ->
`BumpAllocator` -> `MonotonicAllocation<BootInfo, BumpAllocator>` -> `Value`.
The kernel-object path uses `PoolAllocator` -> `Allocation<KernelObject,
PoolAllocator>`, moves the owner without relocating the object, reads it, and
drops/releases it at scope exit. Both execute through strict C11 tests.

R7b adds configurable Mind execution, deterministic batches, dirty memory,
events, actuation obligations, structured trace, replay, checkpoint, explicit
agent context, and seed state. R7c builds deterministic single-worker scheduling
over those mechanisms without compiler semantics or a hidden runtime. The first
R7d slice adds general integer atomic machine semantics but not worker-safe
DragonGod authorities. Collector semantics remain deferred.

The C# Dominatus repository is an application-kernel parity oracle. Its useful
mechanisms are native DragonGod APIs; its policy remains application-owned. No
separate Dominatus module/namespace or compatibility bridge is part of the
architecture. See `DRAGONGOD-DOMINATUS-PARITY.md` for the audited mapping.

Intended layering is Concept compiler -> Standard -> DragonGod -> optional
collector/platform-worker adapters -> Go/.NET/Vulkan consumers. Standard has no
DragonGod dependency, and ordinary Concept programs need neither DragonGod nor
runtime namespace/package metadata.
