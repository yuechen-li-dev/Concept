# Vulkan in Concept

Vulkan code is ordinary Concept. There is no second language: the M-era
Concept Vulkan surface (signal automata, `dispatch`, `effect`, `actuator`,
Vulkan builtin types) was reconciled into Core and removed. See
`docs/design/CONCEPT-VULKAN-RECONCILIATION-LADDER.md` for the record.

## What Core provides

| Need | Core feature |
| --- | --- |
| Vulkan object handles | `extern "C" handle VkBuffer;` (spec 21): copyable, `==`/`!=` only, `Option<VkBuffer>` for absence |
| `VkResult` | `Result<T, VulkanError>` with `?` and `!`; results are must-use |
| Flag words | `bits` declarations matching the Vulkan bit positions |
| Object lifetime | `owned` values with `Drop`; `move` transfers destruction |
| Lifecycles driven by events | step machines `with input`, `on` reactions, `terminal state`, must-use `StepOutcome` (spec 22) |
| Effects a machine decides on | an enum pushed to `Standard.Collection.Outbox`, carried out by an exhaustive `match` |
| C structs | `[[repr(C)]]` records with measured layout (`concept check` on a native project) |

## profile Vulkan

`profile Vulkan;` is Core semantics plus setup you would otherwise write by
hand:

- `import Vulkan;` is implied;
- `concept check` and `concept test` find `libraries/Vulkan` (or
  `CONCEPT_VULKAN_LIBRARY`) and its measured C layouts;
- `concept test` compiles and links the Vulkan runtime: the GPU-free test
  device by default, the loader-backed runtime with
  `CONCEPT_VULKAN_RUNTIME=device` (include and library paths from
  `VULKAN_SDK`).

You never write the C header, a native manifest, or link flags.
`examples/vulkan` shows the result.

## libraries/Vulkan

`module Vulkan` over a flat C boundary (`native/concept_vulkan.h`):

- handles for the objects below;
- `VulkanError` (the failing `VkResult` codes, `Unknown(code)` otherwise) and
  `Check(code)`; non-negative codes are success or status;
- `BufferUsage` and `MemoryProperty` flag words;
- owned `Context` (instance, compute-capable device, queue, command pool),
  `Buffer` (with `WriteInt`/`ReadInt` for host-visible memory),
  `ComputePipeline` (storage-buffer bindings, one descriptor set,
  `BindStorage`), and `Submission` (`Dispatch` returns it; `Wait` consumes
  it; dropping it unwaited waits).

Two runtimes implement the boundary: `native/test_device.c` (no GPU; it runs
the host equivalent of the kernels it knows, such as `double.spv`) and
`native/device_runtime.c` (the Vulkan loader). Package tests run in Normal
and Verify:

```text
go run ./cmd/concept test libraries/Vulkan --verify
```

## Style

Use `on` for reactions to external input and `transition` for a next state
the machine computes itself. Keep the C boundary flat: handles, integers,
and `[[repr(C)]]` records, so the Concept side needs no raw pointers.

## Compute dispatch

`examples/vulkan/ComputeDispatch.concept` is the vertical: buffers, a
pipeline, a dispatch, a wait, and a readback in 33 lines, with every object
owned and every failure a `VulkanError`. `reference/compute_dispatch.c` is
the same program against Vulkan directly (about 200 lines); the mechanics it
spells out live once in `device_runtime.c`. `tools/vk8/run_vk8.ps1` runs
both on a machine with a GPU and the Vulkan SDK.

## Not yet

Access and barrier derivation (the constitution's typed buffers and derived
synchronization), more than one context per process in the device runtime,
and non-int buffer access.
