/* Flat C boundary between the Concept Vulkan library and a Vulkan runtime.
 *
 * Everything that crosses it is a handle, a scalar, a span, or one of the
 * records below, whose layouts `concept check` measures against
 * concept/Vulkan.concept. Status is a VkResult code: 0 is success, negative is
 * failure. Two runtimes implement it: test_device.c (no GPU; it also checks
 * synchronization) and device_runtime.c (the Vulkan 1.4 loader).
 *
 * The runtimes do only mechanics: create-info structs, memory-type choice,
 * command encoding. Decisions (placement, access, barriers, lifetimes) are
 * made on the Concept side. */
#ifndef CONCEPT_VULKAN_H
#define CONCEPT_VULKAN_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

#ifndef VK_VERSION_1_0
typedef struct VkInstance_T* VkInstance;
typedef struct VkPhysicalDevice_T* VkPhysicalDevice;
typedef struct VkDevice_T* VkDevice;
typedef struct VkQueue_T* VkQueue;
typedef struct VkCommandPool_T* VkCommandPool;
typedef struct VkCommandBuffer_T* VkCommandBuffer;
typedef struct VkFence_T* VkFence;
typedef struct VkBuffer_T* VkBuffer;
typedef struct VkDeviceMemory_T* VkDeviceMemory;
typedef struct VkShaderModule_T* VkShaderModule;
typedef struct VkDescriptorSetLayout_T* VkDescriptorSetLayout;
typedef struct VkPipelineLayout_T* VkPipelineLayout;
typedef struct VkPipeline_T* VkPipeline;
typedef struct VkPipelineCache_T* VkPipelineCache;
typedef struct VkQueryPool_T* VkQueryPool;
#endif

/* Spans cross by value as { data, length }, length in elements (spec section 21).
 * These are layout-identical to the generated concept_[readonly_]span_* types. */
typedef struct ConceptVkBytes { const uint8_t* data; size_t length; } ConceptVkBytes;
typedef struct ConceptVkMutableBytes { uint8_t* data; size_t length; } ConceptVkMutableBytes;

/* ---- Context ---------------------------------------------------------- */

/* Vulkan 1.4 instance and device on the chosen physical device (discrete
 * first; CONCEPT_VULKAN_DEVICE=<index or name substring> overrides), its
 * compute queue, a command pool, and a pipeline cache. */
typedef struct ContextCreation {
    int32_t code;
    uint32_t queueFamily;
    VkInstance instance;
    VkPhysicalDevice physical;
    VkDevice device;
    VkQueue queue;
    VkCommandPool commands;
    VkPipelineCache cache;
} ContextCreation;

/* deviceType follows VkPhysicalDeviceType. name is NUL-terminated UTF-8. */
typedef struct DeviceFacts {
    uint32_t apiVersion;
    uint32_t deviceType;
    uint32_t subgroupSize;
    uint32_t maxWorkgroupInvocations;
    uint32_t timestampValidBits;
    float timestampPeriod;
    uint64_t maxStorageBufferRange;
    uint8_t name[256];
} DeviceFacts;

ContextCreation ConceptVkCreateContext(void);
void ConceptVkDestroyContext(VkInstance instance, VkDevice device, VkCommandPool commands, VkPipelineCache cache);
DeviceFacts ConceptVkDeviceFacts(VkPhysicalDevice physical);

/* ---- Buffers ---------------------------------------------------------- */

/* placement: 0 device-local, 1 upload (host-visible, coherent),
 * 2 readback (host-visible, cached when available), 3 shared (device-local
 * and host-visible when the device has it, else host-visible). Buffers are
 * always storage + transfer source + transfer destination. */
typedef struct BufferCreation {
    int32_t code;
    uint32_t hostVisible;
    VkBuffer buffer;
    VkDeviceMemory memory;
} BufferCreation;

BufferCreation ConceptVkCreateBuffer(VkPhysicalDevice physical, VkDevice device, uint64_t bytes, uint32_t placement);
void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory);
int32_t ConceptVkWriteBytes(VkDevice device, VkDeviceMemory memory, uint64_t offset, ConceptVkBytes bytes);
int32_t ConceptVkReadBytes(VkDevice device, VkDeviceMemory memory, uint64_t offset, ConceptVkMutableBytes out);

/* ---- Pipelines -------------------------------------------------------- */

/* kind: 0 storage buffer, 1 uniform buffer. */
typedef struct BindingSlot {
    uint32_t binding;
    uint32_t kind;
} BindingSlot;
typedef struct ConceptVkBindingSlots { const BindingSlot* data; size_t length; } ConceptVkBindingSlots;

/* One compute pipeline over set 0, laid out as push descriptors, with
 * pushBytes of push constants visible to the compute stage. */
typedef struct PipelineCreation {
    int32_t code;
    uint32_t pushBytes;
    VkShaderModule shader;
    VkDescriptorSetLayout setLayout;
    VkPipelineLayout layout;
    VkPipeline pipeline;
} PipelineCreation;

PipelineCreation ConceptVkCreatePipeline(VkDevice device, VkPipelineCache cache, const char* kernelPath, const char* entry, ConceptVkBindingSlots bindings, uint32_t pushBytes);
void ConceptVkDestroyPipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline);

/* ---- Recording -------------------------------------------------------- */

typedef struct RecordingBegin {
    int32_t code;
    uint32_t timestamps;
    VkCommandBuffer commands;
    VkQueryPool queries;
} RecordingBegin;

typedef struct DescriptorWrite {
    VkBuffer buffer;
    uint64_t offset;
    uint64_t range;
    uint32_t binding;
    uint32_t kind;
} DescriptorWrite;
typedef struct ConceptVkDescriptorWrites { const DescriptorWrite* data; size_t length; } ConceptVkDescriptorWrites;

/* Stage bits used by barriers: 1 compute, 2 transfer, 4 host. */
RecordingBegin ConceptVkBeginRecording(VkDevice device, VkCommandPool commands, uint32_t timestamps);
void ConceptVkEndAbandoned(VkDevice device, VkCommandPool commands, VkCommandBuffer recording, VkQueryPool queries);
/* A global synchronization2 memory barrier. srcWrites says whether the
 * source operations wrote (memory dependency) or only read (execution
 * dependency). Destination access is every access of dstStages. */
void ConceptVkCmdBarrier(VkCommandBuffer recording, uint32_t srcStages, uint32_t srcWrites, uint32_t dstStages);
void ConceptVkCmdDispatch(VkCommandBuffer recording, VkPipeline pipeline, VkPipelineLayout layout, ConceptVkDescriptorWrites writes, ConceptVkBytes pushConstants, uint32_t x, uint32_t y, uint32_t z);
void ConceptVkCmdCopy(VkCommandBuffer recording, VkBuffer from, VkBuffer to, uint64_t bytes);
void ConceptVkCmdFill(VkCommandBuffer recording, VkBuffer buffer, uint64_t bytes, uint32_t word);
void ConceptVkCmdTimestamp(VkCommandBuffer recording, VkQueryPool queries, uint32_t index);

typedef struct SubmissionRecord {
    int32_t code;
    uint32_t reserved;
    VkFence fence;
} SubmissionRecord;

SubmissionRecord ConceptVkSubmit(VkDevice device, VkQueue queue, VkCommandBuffer recording);
/* Waits for a submission. Release frees what it owned; timestamps are read
 * between the two, while the query pool is alive.
 */
int32_t ConceptVkWait(VkDevice device, VkFence fence);
void ConceptVkRelease(VkDevice device, VkCommandPool commands, VkCommandBuffer recording, VkFence fence, VkQueryPool queries);
/* Nanoseconds between two written timestamps of a finished submission. */
typedef struct Elapsed {
    int32_t code;
    uint32_t reserved;
    double nanoseconds;
} Elapsed;
Elapsed ConceptVkElapsed(VkDevice device, VkPhysicalDevice physical, VkQueryPool queries, uint32_t from, uint32_t to);

/* Lifetime observers. Both runtimes export them, so lifetime facts run on the
   test device and on a real GPU alike. */
int ConceptVkTestLiveBuffers(void);
int ConceptVkTestLivePipelines(void);
int ConceptVkTestLiveContexts(void);
int ConceptVkTestPendingSubmissions(void);
int ConceptVkTestDoubleDestroys(void);
/* Synchronization observers: barriers recorded, and hazards a submitted
   recording contained without a covering barrier (always 0 on a GPU; the
   test device checks). */
int ConceptVkTestBarriers(void);
int ConceptVkTestHazards(void);

#ifdef __cplusplus
}
#endif

#endif
