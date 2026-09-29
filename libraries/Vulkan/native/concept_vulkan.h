/* Flat C boundary between the Concept Vulkan library and a Vulkan runtime.
 * Everything that crosses it is a handle, an integer, a string, or one of
 * the records below, whose layouts `concept check` measures. Status is a
 * VkResult code: 0 is success, negative is failure. Two runtimes implement
 * it: test_device.c (no GPU) and device_runtime.c (the Vulkan loader). */
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
typedef struct VkDescriptorPool_T* VkDescriptorPool;
typedef struct VkDescriptorSet_T* VkDescriptorSet;
#endif

/* One instance, the first physical device with a compute queue, a logical
 * device, that queue, and a command pool for it. */
typedef struct ComputeContext {
    int32_t code;
    uint32_t queueFamily;
    VkInstance instance;
    VkPhysicalDevice physical;
    VkDevice device;
    VkQueue queue;
    VkCommandPool commands;
} ComputeContext;

typedef struct BufferCreation {
    int32_t code;
    VkBuffer buffer;
    VkDeviceMemory memory;
} BufferCreation;

typedef struct IntRead {
    int32_t code;
    int32_t value;
} IntRead;

/* A compute pipeline over `storageBuffers` storage-buffer bindings
 * (set 0, bindings 0..n-1), with one descriptor set allocated for it. */
typedef struct PipelineCreation {
    int32_t code;
    uint32_t storageBuffers;
    VkShaderModule shader;
    VkDescriptorSetLayout setLayout;
    VkPipelineLayout layout;
    VkPipeline pipeline;
    VkDescriptorPool descriptors;
    VkDescriptorSet set;
} PipelineCreation;

typedef struct SubmissionRecord {
    int32_t code;
    uint32_t groups;
    VkCommandBuffer commands;
    VkFence fence;
} SubmissionRecord;

ComputeContext ConceptVkCreateComputeContext(void);
void ConceptVkDestroyComputeContext(VkInstance instance, VkDevice device, VkCommandPool commands);

BufferCreation ConceptVkCreateBuffer(VkPhysicalDevice physical, VkDevice device, size_t size, uint32_t usage, uint32_t properties);
void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory);
int32_t ConceptVkWriteInt(VkDevice device, VkDeviceMemory memory, size_t index, int32_t value);
IntRead ConceptVkReadInt(VkDevice device, VkDeviceMemory memory, size_t index);

PipelineCreation ConceptVkCreateComputePipeline(VkDevice device, const char* spirvPath, uint32_t storageBuffers);
void ConceptVkDestroyComputePipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline, VkDescriptorPool descriptors);
void ConceptVkBindStorageBuffer(VkDevice device, VkDescriptorSet set, uint32_t binding, VkBuffer buffer, size_t size);

SubmissionRecord ConceptVkSubmitDispatch(VkDevice device, VkQueue queue, VkCommandPool commands, VkPipeline pipeline, VkPipelineLayout layout, VkDescriptorSet set, uint32_t groups);
int32_t ConceptVkWait(VkDevice device, VkCommandPool commands, VkCommandBuffer submitted, VkFence fence);

#ifdef __cplusplus
}
#endif

#endif
