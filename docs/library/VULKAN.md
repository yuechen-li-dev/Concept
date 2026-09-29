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

- `VkDevice`, `VkBuffer`, `VkDeviceMemory` handles;
- `VulkanError` (the failing `VkResult` codes, `Unknown(code)` otherwise) and
  `Check(code)`; non-negative codes are success or status;
- `BufferUsage` and `MemoryProperty` flag words;
- `Buffer`, an owned buffer plus its memory, and
  `CreateBuffer(device, size, usage, properties) -> Result<Buffer, VulkanError>`.

Package tests link `native/test_device.c`, a GPU-free implementation of the C
boundary, and run in Normal and Verify:

```text
go run ./cmd/concept test libraries/Vulkan --verify
```

## Style

Use `on` for reactions to external input and `transition` for a next state
the machine computes itself. Keep the C boundary flat: handles, integers,
and `[[repr(C)]]` records, so the Concept side needs no raw pointers.

## Not yet

The loader-backed implementation of `concept_vulkan.h`, command recording,
descriptors, pipelines, and the compute-dispatch vertical against an
equivalent C program are VK8. Barrier and access derivation follow it.
