# DragonGod library

`libraries/DragonGod` is the only active DragonGod implementation. Its ordinary
immutable manifest credits `CODEX` and declares the sole package dependency
`Standard`.

The R7b surface is Core, Agent, Memory, Machine, Events, Actuation, Trace,
Replay, Interrupts, Time, AMD64, and AArch64. It is the native agentic compute
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
There is no private allocator, hidden heap, scheduler, collector, VM/page-table
system, runtime registry, Vulkan policy, reflection, or serialization layer.

Build with `go run ./cmd/concept package build DragonGod`; test with `go run
./cmd/concept package test DragonGod`. The repository bootstrap equivalent is
the `BuildDragonGod` or `TestDragonGod` target in `Make.oct`.
