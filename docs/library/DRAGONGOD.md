# DragonGod library

`libraries/DragonGod` is the only active DragonGod implementation. Its ordinary
immutable manifest credits `CODEX` and declares the sole package dependency
`Standard`.

The R7c surface is Core, Agent, Memory, Machine, Events, Actuation, Trace,
Replay, Scheduling, Interrupts, Time, AMD64, and AArch64. It is the native agentic compute
substrate corresponding to the useful architecture of the C# Dominatus
application kernel. There is deliberately no separate Dominatus package or
namespace.

`AgentContext<Configuration, Capacity>` composes a Mind, dirty-tracked memory,
event bus, actuator host, monotonic clock, and explicit random seed. Component
capacities are compile-time configuration and tests prove multiple sizes; no
four-slot/four-depth PoC constant is architectural. `Mind.Step` advances one
explicit signal, while `ExecuteBatch` runs caller-owned work in deterministic
order under a budget. Replay and checkpoint state are explicit and bounded.

Memory continues to dogfood Standard.Memory for owned allocation paths. The
agentic components use inline arrays when no dynamic allocation is required.
Scheduling is ordinary fixed-capacity library policy: the scheduler owns a
context table, its ready queue stores only stable IDs, and a caller-selected
semantic step quantum drives deterministic FIFO rotation. Waiting, monotonic
sleep, failure isolation, and typed trace are explicit. See
`DRAGONGOD-SCHEDULING.md`. R7d's initial strict-C11 atomic substrate lives in
Standard; the current DragonGod EventBus, MemoryState, and ActuatorHost remain
single-worker until their shared-authority proofs and guarded algorithms exist.
See `DRAGONGOD-SYNCHRONIZATION.md`. There is no private allocator, hidden runtime
scheduler, collector, VM/page-table system, runtime registry, Vulkan policy,
reflection, or serialization layer.

Build with `go run ./cmd/concept package build DragonGod`; test with `go run
./cmd/concept package test DragonGod`. The repository bootstrap equivalent is
the `BuildDragonGod` or `TestDragonGod` target in `Make.oct`.
