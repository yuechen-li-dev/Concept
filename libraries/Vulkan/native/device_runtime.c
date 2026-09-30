/* The Vulkan-loader implementation of concept_vulkan.h. It does the
 * mechanical parts of Vulkan (create-info structs, memory-type selection,
 * command recording) so the Concept side states only the decisions. */
#include <vulkan/vulkan.h>

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "concept_vulkan.h"

/* Live-object counts, kept by the exported wrappers at the end of this file.
   They are the same observers the test device exports, so lifetime facts
   (Drop destroys exactly once, a dropped Submission is waited) run unchanged
   on a real GPU. A real device cannot see a double destroy; the validation
   layers report that instead, so ConceptVkTestDoubleDestroys stays 0. */
static int live_buffers = 0;
static int live_pipelines = 0;
static int live_contexts = 0;
static int pending_submissions = 0;

static ComputeContext impl_CreateComputeContext(void) {
    ComputeContext out;
    memset(&out, 0, sizeof out);
    VkApplicationInfo app = {.sType = VK_STRUCTURE_TYPE_APPLICATION_INFO};
    app.pApplicationName = "Concept";
    app.apiVersion = VK_API_VERSION_1_1;
    VkInstanceCreateInfo instanceInfo = {.sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO};
    instanceInfo.pApplicationInfo = &app;
    out.code = vkCreateInstance(&instanceInfo, NULL, &out.instance);
    if (out.code != VK_SUCCESS) return out;

    uint32_t count = 0;
    vkEnumeratePhysicalDevices(out.instance, &count, NULL);
    VkPhysicalDevice physical[16];
    if (count > 16) count = 16;
    vkEnumeratePhysicalDevices(out.instance, &count, physical);
    for (uint32_t d = 0; d < count && out.physical == VK_NULL_HANDLE; d++) {
        uint32_t families = 0;
        vkGetPhysicalDeviceQueueFamilyProperties(physical[d], &families, NULL);
        VkQueueFamilyProperties properties[32];
        if (families > 32) families = 32;
        vkGetPhysicalDeviceQueueFamilyProperties(physical[d], &families, properties);
        for (uint32_t f = 0; f < families; f++) {
            if (properties[f].queueFlags & VK_QUEUE_COMPUTE_BIT) {
                out.physical = physical[d];
                out.queueFamily = f;
                break;
            }
        }
    }
    if (out.physical == VK_NULL_HANDLE) {
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        vkDestroyInstance(out.instance, NULL);
        out.instance = VK_NULL_HANDLE;
        return out;
    }

    float priority = 1.0f;
    VkDeviceQueueCreateInfo queueInfo = {.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO};
    queueInfo.queueFamilyIndex = out.queueFamily;
    queueInfo.queueCount = 1;
    queueInfo.pQueuePriorities = &priority;
    VkDeviceCreateInfo deviceInfo = {.sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO};
    deviceInfo.queueCreateInfoCount = 1;
    deviceInfo.pQueueCreateInfos = &queueInfo;
    out.code = vkCreateDevice(out.physical, &deviceInfo, NULL, &out.device);
    if (out.code != VK_SUCCESS) {
        vkDestroyInstance(out.instance, NULL);
        out.instance = VK_NULL_HANDLE;
        return out;
    }
    vkGetDeviceQueue(out.device, out.queueFamily, 0, &out.queue);

    VkCommandPoolCreateInfo poolInfo = {.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO};
    poolInfo.queueFamilyIndex = out.queueFamily;
    out.code = vkCreateCommandPool(out.device, &poolInfo, NULL, &out.commands);
    if (out.code != VK_SUCCESS) {
        vkDestroyDevice(out.device, NULL);
        vkDestroyInstance(out.instance, NULL);
        out.device = VK_NULL_HANDLE;
        out.instance = VK_NULL_HANDLE;
    }
    return out;
}

static void impl_DestroyComputeContext(VkInstance instance, VkDevice device, VkCommandPool commands) {
    vkDeviceWaitIdle(device);
    vkDestroyCommandPool(device, commands, NULL);
    vkDestroyDevice(device, NULL);
    vkDestroyInstance(instance, NULL);
}

static BufferCreation impl_CreateBuffer(VkPhysicalDevice physical, VkDevice device, size_t size, uint32_t usage, uint32_t properties) {
    BufferCreation out;
    memset(&out, 0, sizeof out);
    VkBufferCreateInfo bufferInfo = {.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO};
    bufferInfo.size = size;
    bufferInfo.usage = usage;
    bufferInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    out.code = vkCreateBuffer(device, &bufferInfo, NULL, &out.buffer);
    if (out.code != VK_SUCCESS) return out;

    VkMemoryRequirements requirements;
    vkGetBufferMemoryRequirements(device, out.buffer, &requirements);
    VkPhysicalDeviceMemoryProperties memory;
    vkGetPhysicalDeviceMemoryProperties(physical, &memory);
    uint32_t type = UINT32_MAX;
    for (uint32_t i = 0; i < memory.memoryTypeCount; i++) {
        if ((requirements.memoryTypeBits & (1u << i)) && (memory.memoryTypes[i].propertyFlags & properties) == properties) {
            type = i;
            break;
        }
    }
    if (type == UINT32_MAX) {
        out.code = VK_ERROR_FEATURE_NOT_PRESENT;
        vkDestroyBuffer(device, out.buffer, NULL);
        out.buffer = VK_NULL_HANDLE;
        return out;
    }
    VkMemoryAllocateInfo allocation = {.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO};
    allocation.allocationSize = requirements.size;
    allocation.memoryTypeIndex = type;
    out.code = vkAllocateMemory(device, &allocation, NULL, &out.memory);
    if (out.code == VK_SUCCESS) out.code = vkBindBufferMemory(device, out.buffer, out.memory, 0);
    if (out.code != VK_SUCCESS) {
        vkDestroyBuffer(device, out.buffer, NULL);
        if (out.memory != VK_NULL_HANDLE) vkFreeMemory(device, out.memory, NULL);
        out.buffer = VK_NULL_HANDLE;
        out.memory = VK_NULL_HANDLE;
    }
    return out;
}

static void impl_DestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory) {
    vkDestroyBuffer(device, buffer, NULL);
    vkFreeMemory(device, memory, NULL);
}

int32_t ConceptVkWriteInt(VkDevice device, VkDeviceMemory memory, size_t index, int32_t value) {
    void* mapped = NULL;
    VkResult code = vkMapMemory(device, memory, index * sizeof(int32_t), sizeof(int32_t), 0, &mapped);
    if (code != VK_SUCCESS) return code;
    memcpy(mapped, &value, sizeof value);
    vkUnmapMemory(device, memory);
    return VK_SUCCESS;
}

IntRead ConceptVkReadInt(VkDevice device, VkDeviceMemory memory, size_t index) {
    IntRead out = {0, 0};
    void* mapped = NULL;
    out.code = vkMapMemory(device, memory, index * sizeof(int32_t), sizeof(int32_t), 0, &mapped);
    if (out.code != VK_SUCCESS) return out;
    memcpy(&out.value, mapped, sizeof out.value);
    vkUnmapMemory(device, memory);
    return out;
}

/* Relative kernel paths resolve against CONCEPT_VULKAN_KERNELS when set
 * (`concept test` sets it to the test's source directory). */
static FILE* open_kernel(const char* path) {
    FILE* file = fopen(path, "rb");
    const char* base = getenv("CONCEPT_VULKAN_KERNELS");
    if (file || !base || path[0] == '/' || path[0] == '\\' || (path[0] && path[1] == ':')) return file;
    char joined[4096];
    if ((size_t)snprintf(joined, sizeof joined, "%s/%s", base, path) >= sizeof joined) return NULL;
    return fopen(joined, "rb");
}

static uint32_t* read_spirv(const char* path, size_t* bytes) {
    FILE* file = open_kernel(path);
    if (!file) return NULL;
    fseek(file, 0, SEEK_END);
    long size = ftell(file);
    fseek(file, 0, SEEK_SET);
    if (size <= 0 || size % 4 != 0) {
        fclose(file);
        return NULL;
    }
    uint32_t* words = (uint32_t*)malloc((size_t)size);
    if (words && fread(words, 1, (size_t)size, file) != (size_t)size) {
        free(words);
        words = NULL;
    }
    fclose(file);
    *bytes = (size_t)size;
    return words;
}

static PipelineCreation impl_CreateComputePipeline(VkDevice device, const char* spirvPath, uint32_t storageBuffers) {
    PipelineCreation out;
    memset(&out, 0, sizeof out);
    out.storageBuffers = storageBuffers;
    if (storageBuffers == 0 || storageBuffers > 8) {
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        return out;
    }
    size_t bytes = 0;
    uint32_t* words = read_spirv(spirvPath, &bytes);
    if (!words) {
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        return out;
    }
    VkShaderModuleCreateInfo shaderInfo = {.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO};
    shaderInfo.codeSize = bytes;
    shaderInfo.pCode = words;
    out.code = vkCreateShaderModule(device, &shaderInfo, NULL, &out.shader);
    free(words);
    if (out.code != VK_SUCCESS) return out;

    VkDescriptorSetLayoutBinding bindings[8];
    memset(bindings, 0, sizeof bindings);
    for (uint32_t i = 0; i < storageBuffers; i++) {
        bindings[i].binding = i;
        bindings[i].descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_BUFFER;
        bindings[i].descriptorCount = 1;
        bindings[i].stageFlags = VK_SHADER_STAGE_COMPUTE_BIT;
    }
    VkDescriptorSetLayoutCreateInfo setInfo = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO};
    setInfo.bindingCount = storageBuffers;
    setInfo.pBindings = bindings;
    out.code = vkCreateDescriptorSetLayout(device, &setInfo, NULL, &out.setLayout);
    if (out.code != VK_SUCCESS) return out;

    VkPipelineLayoutCreateInfo layoutInfo = {.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO};
    layoutInfo.setLayoutCount = 1;
    layoutInfo.pSetLayouts = &out.setLayout;
    out.code = vkCreatePipelineLayout(device, &layoutInfo, NULL, &out.layout);
    if (out.code != VK_SUCCESS) return out;

    VkComputePipelineCreateInfo pipelineInfo = {.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO};
    pipelineInfo.stage.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    pipelineInfo.stage.stage = VK_SHADER_STAGE_COMPUTE_BIT;
    pipelineInfo.stage.module = out.shader;
    pipelineInfo.stage.pName = "main";
    pipelineInfo.layout = out.layout;
    out.code = vkCreateComputePipelines(device, VK_NULL_HANDLE, 1, &pipelineInfo, NULL, &out.pipeline);
    if (out.code != VK_SUCCESS) return out;

    VkDescriptorPoolSize poolSize = {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, storageBuffers};
    VkDescriptorPoolCreateInfo poolInfo = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO};
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 1;
    poolInfo.pPoolSizes = &poolSize;
    out.code = vkCreateDescriptorPool(device, &poolInfo, NULL, &out.descriptors);
    if (out.code != VK_SUCCESS) return out;

    VkDescriptorSetAllocateInfo allocation = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO};
    allocation.descriptorPool = out.descriptors;
    allocation.descriptorSetCount = 1;
    allocation.pSetLayouts = &out.setLayout;
    out.code = vkAllocateDescriptorSets(device, &allocation, &out.set);
    return out;
}

static void impl_DestroyComputePipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline, VkDescriptorPool descriptors) {
    vkDestroyDescriptorPool(device, descriptors, NULL);
    vkDestroyPipeline(device, pipeline, NULL);
    vkDestroyPipelineLayout(device, layout, NULL);
    vkDestroyDescriptorSetLayout(device, setLayout, NULL);
    vkDestroyShaderModule(device, shader, NULL);
}

void ConceptVkBindStorageBuffer(VkDevice device, VkDescriptorSet set, uint32_t binding, VkBuffer buffer, size_t size) {
    VkDescriptorBufferInfo bufferInfo = {buffer, 0, size};
    VkWriteDescriptorSet write = {.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET};
    write.dstSet = set;
    write.dstBinding = binding;
    write.descriptorCount = 1;
    write.descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_BUFFER;
    write.pBufferInfo = &bufferInfo;
    vkUpdateDescriptorSets(device, 1, &write, 0, NULL);
}

static SubmissionRecord impl_SubmitDispatch(VkDevice device, VkQueue queue, VkCommandPool commands, VkPipeline pipeline, VkPipelineLayout layout, VkDescriptorSet set, uint32_t groups) {
    SubmissionRecord out;
    memset(&out, 0, sizeof out);
    out.groups = groups;
    VkCommandBufferAllocateInfo allocation = {.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO};
    allocation.commandPool = commands;
    allocation.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    allocation.commandBufferCount = 1;
    out.code = vkAllocateCommandBuffers(device, &allocation, &out.commands);
    if (out.code != VK_SUCCESS) return out;

    VkCommandBufferBeginInfo begin = {.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO};
    begin.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(out.commands, &begin);
    vkCmdBindPipeline(out.commands, VK_PIPELINE_BIND_POINT_COMPUTE, pipeline);
    vkCmdBindDescriptorSets(out.commands, VK_PIPELINE_BIND_POINT_COMPUTE, layout, 0, 1, &set, 0, NULL);
    vkCmdDispatch(out.commands, groups, 1, 1);
    /* Make shader writes visible to host reads after the fence. */
    VkMemoryBarrier barrier = {.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER};
    barrier.srcAccessMask = VK_ACCESS_SHADER_WRITE_BIT;
    barrier.dstAccessMask = VK_ACCESS_HOST_READ_BIT;
    vkCmdPipelineBarrier(out.commands, VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT, VK_PIPELINE_STAGE_HOST_BIT, 0, 1, &barrier, 0, NULL, 0, NULL);
    out.code = vkEndCommandBuffer(out.commands);
    if (out.code != VK_SUCCESS) return out;

    VkFenceCreateInfo fenceInfo = {.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO};
    out.code = vkCreateFence(device, &fenceInfo, NULL, &out.fence);
    if (out.code != VK_SUCCESS) return out;
    VkSubmitInfo submit = {.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO};
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &out.commands;
    out.code = vkQueueSubmit(queue, 1, &submit, out.fence);
    return out;
}

static int32_t impl_Wait(VkDevice device, VkCommandPool commands, VkCommandBuffer submitted, VkFence fence) {
    VkResult code = vkWaitForFences(device, 1, &fence, VK_TRUE, UINT64_MAX);
    vkDestroyFence(device, fence, NULL);
    vkFreeCommandBuffers(device, commands, 1, &submitted);
    return code;
}

/* Exported entry points: forward to the implementation and keep the counts. */
ComputeContext ConceptVkCreateComputeContext(void) {
    ComputeContext out = impl_CreateComputeContext();
    if (out.code == VK_SUCCESS) live_contexts++;
    return out;
}

void ConceptVkDestroyComputeContext(VkInstance instance, VkDevice device, VkCommandPool commands) {
    impl_DestroyComputeContext(instance, device, commands);
    live_contexts--;
}

BufferCreation ConceptVkCreateBuffer(VkPhysicalDevice physical, VkDevice device, size_t size, uint32_t usage, uint32_t properties) {
    BufferCreation out = impl_CreateBuffer(physical, device, size, usage, properties);
    if (out.code == VK_SUCCESS) live_buffers++;
    return out;
}

void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory) {
    impl_DestroyBuffer(device, buffer, memory);
    live_buffers--;
}

PipelineCreation ConceptVkCreateComputePipeline(VkDevice device, const char* spirvPath, uint32_t storageBuffers) {
    PipelineCreation out = impl_CreateComputePipeline(device, spirvPath, storageBuffers);
    if (out.code == VK_SUCCESS) live_pipelines++;
    return out;
}

void ConceptVkDestroyComputePipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline, VkDescriptorPool descriptors) {
    impl_DestroyComputePipeline(device, shader, setLayout, layout, pipeline, descriptors);
    live_pipelines--;
}

SubmissionRecord ConceptVkSubmitDispatch(VkDevice device, VkQueue queue, VkCommandPool commands, VkPipeline pipeline, VkPipelineLayout layout, VkDescriptorSet set, uint32_t groups) {
    SubmissionRecord out = impl_SubmitDispatch(device, queue, commands, pipeline, layout, set, groups);
    if (out.code == VK_SUCCESS) pending_submissions++;
    return out;
}

int32_t ConceptVkWait(VkDevice device, VkCommandPool commands, VkCommandBuffer submitted, VkFence fence) {
    int32_t code = impl_Wait(device, commands, submitted, fence);
    if (pending_submissions > 0) pending_submissions--;
    return code;
}

int ConceptVkTestLiveBuffers(void) { return live_buffers; }
int ConceptVkTestLivePipelines(void) { return live_pipelines; }
int ConceptVkTestLiveContexts(void) { return live_contexts; }
int ConceptVkTestPendingSubmissions(void) { return pending_submissions; }
int ConceptVkTestDoubleDestroys(void) { return 0; }
