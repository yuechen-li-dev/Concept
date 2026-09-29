# Vulkan

Vulkan in ordinary Concept (`profile Core`), with no compiler support beyond
`extern "C" handle`.

- `concept/Vulkan.concept` (`module Vulkan`): handles, `VulkanError` and
  `Check(VkResult)`, `BufferUsage`/`MemoryProperty` flag words as `bits`,
  and an owned `Buffer` whose `Drop` destroys it exactly once.
- `native/concept_vulkan.h`: the flat C boundary. Everything crossing it is
  a handle, an integer, or a `[[repr(C)]]` record whose layout is measured
  by `concept check` (see `manifest.concept`).
- `native/test_device.c`: a GPU-free implementation for package tests.

```text
go run ./cmd/concept check libraries/Vulkan
go run ./cmd/concept test libraries/Vulkan --verify
```

A loader-backed implementation of `concept_vulkan.h` and the compute
dispatch vertical are VK8 of
`docs/design/CONCEPT-VULKAN-RECONCILIATION-LADDER.md`.
