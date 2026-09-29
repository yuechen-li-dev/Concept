# Vulkan examples

`profile Vulkan;` is Core plus Vulkan setup that would otherwise be written by
hand: the `Vulkan` library is imported implicitly, and `concept test` links the
Vulkan runtime (the GPU-free test device by default,
`CONCEPT_VULKAN_RUNTIME=device` for a real loader).

```text
go run ./cmd/concept check examples/vulkan/BufferLifetime.concept
go run ./cmd/concept test examples/vulkan --verify
```
