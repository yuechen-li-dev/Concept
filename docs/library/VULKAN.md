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

`module Vulkan` over a flat C boundary (`native/concept_vulkan.h`). The floor
is Vulkan 1.4: push descriptors and synchronization2 are core, so there are no
descriptor pools, no descriptor sets to allocate, and one barrier shape.

- `Context` (`CreateContext`, `Facts`): instance, a 1.4 device with a compute
  queue, command pool, and pipeline cache. Discrete GPUs rank first;
  `CONCEPT_VULKAN_DEVICE` picks by index or name substring (`AMD`, `3070`).
  `CONCEPT_VULKAN_VALIDATION=1` loads the Khronos validation layer.
- `Buffer<T>` (`CreateBuffer<T>(ctx, count, Placement)`): `Device`, `Upload`,
  `Readback`, or `Shared` memory. `Upload`/`Download` take spans and stage
  through a temporary buffer when the memory is not host-visible.
- `Pipeline` (`CreatePipeline`): push-descriptor set layout from a
  `BindingSlot` span, a push-constant range, and the context's cache.
- `Recording` (`BeginRecording(ctx, timestamps)`): `DispatchWith`, `Dispatch`,
  `Copy<T>`, `Fill<T>`, `Timestamp`. Each command names its buffers with
  `Bind<T>`/`BindUniform<T>` and an `Access` (`Read` or `Write`).
- `Submission` (`Submit` consumes the recording; `Wait`; `ElapsedNanoseconds`).
  Dropping an unwaited submission waits, then releases the command buffer.

### Derived synchronization

`Recording` keeps, per buffer, the stages that have read it and written it
since the last barrier. Before each command it compares the command's uses
with that history:

| Earlier | Now | Barrier |
| --- | --- | --- |
| write | read or write | memory dependency (source writes made available and visible) |
| read | write | execution dependency |
| read | read | none |

Everything a command needs becomes one global `VkMemoryBarrier2`. `Submit`
adds the barrier that makes device writes visible to the host. Independent
dispatches get no barrier. `recording.barriers` counts what was inserted, so
tests can pin it.

The test device checks the other side: it records each buffer's last write
stage, what that write has been made visible to, and unordered reads, and
counts a hazard whenever a command or host access touches memory without the
dependency Vulkan requires. `ConceptVkTestHazards()` is 0 in every library and
example test; one fact removes a barrier on purpose and expects a hazard.

### Kernel bindings: `concept vulkan-bind`

Binding code is generated from the SPIR-V, not written:

```text
go run ./cmd/concept vulkan-bind examples/vulkan/kernels/scale.spv -o examples/vulkan/ScaleKernel.concept
go run ./cmd/concept vulkan-bind --describe kernels/scale.spv
go run ./cmd/concept vulkan-bind --check -o ScaleKernel.concept kernels/scale.spv
```

It reflects the entry point, workgroup size, each binding (storage or
uniform, element type, `NonWritable`), and the push-constant block with its
offsets, and writes a module with:

- `comptime` workgroup constants and `<Name>GroupsFor(count)`;
- a `layout <Name>Constants` at the shader's offsets, pinned by
  `static_assert(LayoutSize<…>() == size)`, and a `record struct <Name>Push`;
- `class <Name>` and `Load<Name>(ref const Context)`;
- `Record<Name>(ref Recording, ref const <Name>, buffers…, push, groups)`,
  where read-only bindings take `ref const Buffer<T>` and are bound `Read`,
  and written bindings take `ref Buffer<T>` and are bound `Write`.

So the access that drives barrier derivation comes from the shader, and the
borrow checker sees the same thing. The header carries an interface
fingerprint (entry, workgroup, bindings, push layout; not names). `concept
test` on a `profile Vulkan` package reports a `vulkan-kernel-binding` failure
when a generated module's fingerprint no longer matches its `.spv`, and
`--check` does the same in a build. GLSL (`glslc`) and HLSL (`dxc -spirv`)
kernels are equivalent inputs; `run_gpu.ps1` checks that the two scale
kernels share a fingerprint.

### Tests

Two runtimes implement the boundary: `native/test_device.c` (no GPU; it runs
the host equivalent of the kernels it knows, `double.spv` and `scale.spv`,
checks synchronization, and gives each command 1000 ns of timestamp) and
`native/device_runtime.c` (the Vulkan loader). Package tests run in Normal
and Verify:

```text
go run ./cmd/concept test libraries/Vulkan --verify
```

## Style

Use `on` for reactions to external input and `transition` for a next state
the machine computes itself. Keep the C boundary flat: handles, integers,
spans, and `[[repr(C)]]` records, so the Concept side needs no raw pointers.
Spans cross `extern "C"` as `{pointer, length}` structs; `AsBytes` views a
span of plain data as bytes.

## Examples

`examples/vulkan`:

- `BufferLifetime`: ownership and placement;
- `ComputeDispatch`: upload, one generated-binding dispatch, readback;
  `reference/compute_dispatch.c` is the same program against Vulkan directly;
- `ScaleChain`: two dependent dispatches with push constants, one derived
  barrier, device timestamps. Its source names no stage, access mask,
  descriptor set, or push-constant offset.

`tools/vulkan/run_gpu.ps1` compiles the kernels (GLSL and HLSL), checks the
generated bindings, runs the C reference, the library and examples on the
test device and on the GPU (Normal and Verify, discrete and `AMD`), and a
synchronization-validation pass. It logs to `tools/vulkan/out/gpu.log`.

## Not yet

- Typed views over mapped memory (a `layout` bound over a mapped range, the
  same machinery `stream` uses). `stream` itself does not fit descriptor
  bindings: a stream aliases regions of one fixed layout, while bindings are
  separate runtime-sized buffers.
- Timeline semaphores, a transfer queue, more than one context per process
  in the device runtime.
- Images, ray queries, FFT.
- A Concept-native SPIR-V profile (kernels come from glslc/dxc until then).
