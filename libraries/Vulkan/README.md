# Vulkan

Vulkan 1.4 compute in ordinary Concept. `concept/Vulkan.concept` owns resource
lifetime, typed failures, descriptors, and derived synchronization. Its flat
native boundary is measured through `manifest.concept`.

- `Context` owns its instance, device, queue resources, and cache.
- `BorrowDevice(DeviceBorrow)` attaches to an existing device. `BorrowedContext`
  owns a separate command pool and cache; dropping it leaves the owner's device,
  instance, and queue alive. The caller supplies the queue family and records
  actually enabled synchronization2 (bit 1) and pushDescriptor (bit 2) features.
  Missing enabled capabilities are rejected before resource creation.
- `DeviceOf(ref const Context)` and a borrowed context's `view` provide a
  nonowning `DeviceView`. `CreateBufferOn`, `UploadOn`, `DownloadOn`,
  `CreatePipelineOn`, and `BeginRecordingOn` share that view. Their resources and
  submissions must retire before the device owner.
- Device entry points are registered per device and reference-counted across
  attachments. Independent owners do not overwrite one another's dispatch.
- Native integrations can install `ConceptVkSetQueueSubmit` while quiescent to
  join their owner's queue synchronization. Its borrowed callback owner must
  outlive every attachment using that registration. This is a native adapter
  seam, not a general Concept callback ABI.
- `ConceptVkCreatePipelineFromModule` consumes a package-created shader module
  on success and failure, retaining the package loader's content authority.
- `BufferRange` borrows a byte extent without owning its allocation. `CopyBytes`
  checks source/destination bounds and same-buffer overlap before recording,
  derives transfer dependencies, and admits byte offsets/counts. An empty range
  records nothing. Invalid ranges return the library admission code `-7601`
  through `VulkanError::Unknown`; they do not masquerade as allocation failures.
  Distinct handles must refer to disjoint memory, and the allocation owner must
  outlive the recording and its submission.

- `CreateBufferSharedOn<T>(view, count, placement, first, second)` creates a
  buffer for two queue families. Equal families use exclusive sharing; distinct
  families use concurrent sharing. The native boundary rejects unknown families.
  Sharing removes ownership transfers, not the need for memory dependencies.
- `SignalHandoff(move recording)` submits a producer and returns an owned
  `QueueHandoff`. `ReceiveHandoff(ref handoff, move consumer)` submits exactly
  one consumer on the same device, waiting on the binary semaphore at all commands.
  `WaitTransfer` observes only the producer; `WaitHandoff` observes both sides.
  Dropping the handoff joins consumer and producer before releasing its semaphore,
  including a canceled signal with no consumer. Buffer owners must outlive these
  submissions, and a borrowed device owner must outlive the handoff.
- Integrations using the native queue callback must also install
  `ConceptVkSetQueueSubmitSync` for semaphore submissions. A legacy callback is
  never bypassed to issue an unsynchronized raw queue call. Both registrations
  must use the same owner and remain quiescent while being configured.

`native/test_device.c` supplies deterministic GPU-free package tests.
`native/device_runtime.c` supplies the loader-backed implementation.

```console
concept check libraries/Vulkan
concept test libraries/Vulkan
concept test libraries/Vulkan --verify
```

For Vulkan-profile programs, select `CONCEPT_VULKAN_RUNTIME=device` and supply
`VULKAN_SDK` to execute against the real loader. GPU-free package tests and real
GPU execution are separate evidence. `Wait` only marks completion after success
or terminal device loss. Other wait errors cannot authorize resource release;
recovery must retain pending work until completion is established.

`Poll(ref submission)` returns `Result<bool,VulkanError>`: `false` is still
pending, `true` confirms completion, and a driver error remains an error without
authorizing resource release. On a signaled fence it uses the immediate `Wait`
path to retire native submission accounting. Repeated `Poll`/`Wait` calls are
idempotent and cannot retire another submission's bookkeeping.
