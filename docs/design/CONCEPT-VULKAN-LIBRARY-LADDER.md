# Concept Vulkan library ladder (VL)

Goal: prepare `libraries/Vulkan` for a Prometheus rewrite in Concept by
letting the compiler write the parts of Vulkan that are mechanical. Follows
the VK0–VK9 reconciliation ladder (`CONCEPT-VULKAN-RECONCILIATION-LADDER.md`)
and the Prometheus audit (`PROMETHEUS-AUDIT-2026-09.md`).

| Step | What | Status |
| --- | --- | --- |
| VL1 | Spans cross `extern "C"` as `{pointer, length}`; `AsBytes` byte views | done (`9102ad4`) |
| VL2 | `Buffer<T>` with placements; `Upload`/`Download` with automatic staging | done |
| VL3 | Vulkan 1.4 floor: push descriptors, synchronization2, pipeline cache, device choice, validation switch | done |
| VL4 | `Recording`: per-buffer access history, derived barriers, host barrier at submit, timestamps; test device checks synchronization | done |
| VL5 | `concept vulkan-bind`: SPIR-V reflection to a generated module; fingerprint; stale-binding check in `concept test` | done |
| VL6 | Examples (`ScaleChain`, generated `DoubleKernel`/`ScaleKernel`), HLSL twin, `tools/vulkan/run_gpu.ps1`, docs | done; GPU run pending |

## Compiler work the library needed

- Function templates may call function templates with deduced arguments
  (spec §31). Deduction skips parameters that mention no template parameter,
  so `Bind(0, ref const buffer, …)` deduces `T` from the buffer and converts
  the literal.
- Imported generic instances (closed nominal structs such as `Buffer<int>`)
  are re-read as applications for deduction and `Drop` lookup.
- `assert` and `static_assert` substitute inside template bodies.
- Record construction inside templates sees the template's parameters.
- Declared types whose names match a builtin case-insensitively (`Double`)
  no longer collide with it in C.

## `stream` and `layout`

Asked: can `stream` describe the resource bindings?

Not as it stands. A `stream` aliases regions of one fixed layout; Vulkan
bindings are separate runtime-sized buffers bound independently, and their
access has to be known per command for barrier derivation. The binding list
is instead generated from SPIR-V reflection, with access taken from
`NonWritable`.

`layout` does fit the push-constant block: a push block is a fixed byte
image with offsets the shader chooses. The generator emits a
`layout <Name>Constants` at the reflected offsets with a `static_assert` on
its size, so the compiler checks the geometry.

Where `stream`'s machinery will pay off: typed views over mapped memory
(binding a layout over a mapped staging range, L1 in the deferred list), so
uploads of structured data stop going through `ReadOnlySpan<T>` copies.

## Deferred

- L1 typed views over mapped/foreign memory (layout bound over a mapped range).
- Timeline semaphores; a dedicated transfer queue; several contexts per
  process in the device runtime.
- Images and samplers; ray queries; FFT (in scope per the audit, later).
- Batch planning through the selector (Prometheus A3) once the rewrite starts.
- `oct make` kernel build target; a Concept-native `profile spirv` much later.
