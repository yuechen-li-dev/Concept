# Vulkan examples

`profile Vulkan;` is Core plus Vulkan setup that would otherwise be written by
hand: the `Vulkan` library is imported implicitly, and `concept test` links the
Vulkan runtime (the GPU-free test device by default,
`CONCEPT_VULKAN_RUNTIME=device` for a real loader).

```text
go run ./cmd/concept check examples/vulkan/BufferLifetime.concept
go run ./cmd/concept test examples/vulkan --verify
```

On a real GPU, with the Vulkan SDK installed, compile the kernel and select the
device runtime (PowerShell):

```text
glslc examples\vulkan\kernels\double.comp -o examples\vulkan\kernels\double.spv
$env:CONCEPT_VULKAN_RUNTIME = "device"
go run ./cmd/concept test examples/vulkan --verify
```

`tools/vk8/run_vk8.ps1` runs the whole device check, including the C reference
in `reference/compute_dispatch.c`, and logs to `tools/vk8/out/vk8.log`.
