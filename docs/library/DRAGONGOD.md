# DragonGod library

`libraries/DragonGod` is the only active DragonGod implementation. Its ordinary
immutable manifest credits `CODEX` and declares the sole package dependency
`Standard`.

The bounded R7a surface is Core, Memory, Machine, Interrupts, Time, AMD64, and
AArch64. Memory actively dogfoods Standard.Memory; there is no private allocator,
scheduler, collector, VM/page-table system, runtime registry, Vulkan policy,
reflection, or serialization layer.

Build with `go run ./cmd/concept package build DragonGod`; test with `go run
./cmd/concept package test DragonGod`. The repository bootstrap equivalent is
the `BuildDragonGod` or `TestDragonGod` target in `Make.oct`.
