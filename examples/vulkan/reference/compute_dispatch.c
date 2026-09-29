/* examples/vulkan/ComputeDispatch.concept, written directly against Vulkan
 * in C for comparison: same buffers, pipeline, dispatch, wait and readback,
 * with the error handling and cleanup the Concept version gets from Result
 * and Drop. Usage: compute_dispatch <path/to/double.spv>; prints the sum. */
#include <vulkan/vulkan.h>

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#define COUNT 64

static int find_memory_type(VkPhysicalDevice physical, uint32_t bits, VkMemoryPropertyFlags wanted, uint32_t* type) {
    VkPhysicalDeviceMemoryProperties memory;
    vkGetPhysicalDeviceMemoryProperties(physical, &memory);
    for (uint32_t i = 0; i < memory.memoryTypeCount; i++) {
        if ((bits & (1u << i)) && (memory.memoryTypes[i].propertyFlags & wanted) == wanted) {
            *type = i;
            return 1;
        }
    }
    return 0;
}

static VkResult create_buffer(VkPhysicalDevice physical, VkDevice device, VkDeviceSize size, VkBuffer* buffer, VkDeviceMemory* memory) {
    VkBufferCreateInfo info = {.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO};
    info.size = size;
    info.usage = VK_BUFFER_USAGE_STORAGE_BUFFER_BIT;
    info.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    VkResult result = vkCreateBuffer(device, &info, NULL, buffer);
    if (result != VK_SUCCESS) return result;
    VkMemoryRequirements requirements;
    vkGetBufferMemoryRequirements(device, *buffer, &requirements);
    VkMemoryAllocateInfo allocation = {.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO};
    allocation.allocationSize = requirements.size;
    if (!find_memory_type(physical, requirements.memoryTypeBits, VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, &allocation.memoryTypeIndex)) {
        vkDestroyBuffer(device, *buffer, NULL);
        return VK_ERROR_FEATURE_NOT_PRESENT;
    }
    result = vkAllocateMemory(device, &allocation, NULL, memory);
    if (result != VK_SUCCESS) {
        vkDestroyBuffer(device, *buffer, NULL);
        return result;
    }
    result = vkBindBufferMemory(device, *buffer, *memory, 0);
    if (result != VK_SUCCESS) {
        vkFreeMemory(device, *memory, NULL);
        vkDestroyBuffer(device, *buffer, NULL);
    }
    return result;
}

static uint32_t* read_file(const char* path, size_t* size) {
    FILE* file = fopen(path, "rb");
    if (!file) return NULL;
    fseek(file, 0, SEEK_END);
    long length = ftell(file);
    fseek(file, 0, SEEK_SET);
    uint32_t* words = length > 0 && length % 4 == 0 ? malloc((size_t)length) : NULL;
    if (words && fread(words, 1, (size_t)length, file) != (size_t)length) {
        free(words);
        words = NULL;
    }
    fclose(file);
    *size = (size_t)length;
    return words;
}

int main(int argc, char** argv) {
    if (argc != 2) {
        fprintf(stderr, "usage: %s double.spv\n", argv[0]);
        return 2;
    }
    VkResult result = VK_SUCCESS;
    VkInstance instance = VK_NULL_HANDLE;
    VkDevice device = VK_NULL_HANDLE;
    VkCommandPool pool = VK_NULL_HANDLE;
    VkBuffer input = VK_NULL_HANDLE, output = VK_NULL_HANDLE;
    VkDeviceMemory inputMemory = VK_NULL_HANDLE, outputMemory = VK_NULL_HANDLE;
    VkShaderModule shader = VK_NULL_HANDLE;
    VkDescriptorSetLayout setLayout = VK_NULL_HANDLE;
    VkPipelineLayout layout = VK_NULL_HANDLE;
    VkPipeline pipeline = VK_NULL_HANDLE;
    VkDescriptorPool descriptors = VK_NULL_HANDLE;
    VkFence fence = VK_NULL_HANDLE;
    int sum = 0;

    VkApplicationInfo app = {.sType = VK_STRUCTURE_TYPE_APPLICATION_INFO, .apiVersion = VK_API_VERSION_1_1};
    VkInstanceCreateInfo instanceInfo = {.sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO, .pApplicationInfo = &app};
    if ((result = vkCreateInstance(&instanceInfo, NULL, &instance)) != VK_SUCCESS) goto done;

    uint32_t count = 16;
    VkPhysicalDevice physicals[16];
    vkEnumeratePhysicalDevices(instance, &count, physicals);
    VkPhysicalDevice physical = VK_NULL_HANDLE;
    uint32_t family = 0;
    for (uint32_t d = 0; d < count && physical == VK_NULL_HANDLE; d++) {
        uint32_t families = 32;
        VkQueueFamilyProperties properties[32];
        vkGetPhysicalDeviceQueueFamilyProperties(physicals[d], &families, properties);
        for (uint32_t f = 0; f < families; f++) {
            if (properties[f].queueFlags & VK_QUEUE_COMPUTE_BIT) {
                physical = physicals[d];
                family = f;
                break;
            }
        }
    }
    if (physical == VK_NULL_HANDLE) {
        result = VK_ERROR_INITIALIZATION_FAILED;
        goto done;
    }

    float priority = 1.0f;
    VkDeviceQueueCreateInfo queueInfo = {.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO, .queueFamilyIndex = family, .queueCount = 1, .pQueuePriorities = &priority};
    VkDeviceCreateInfo deviceInfo = {.sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO, .queueCreateInfoCount = 1, .pQueueCreateInfos = &queueInfo};
    if ((result = vkCreateDevice(physical, &deviceInfo, NULL, &device)) != VK_SUCCESS) goto done;
    VkQueue queue;
    vkGetDeviceQueue(device, family, 0, &queue);
    VkCommandPoolCreateInfo poolInfo = {.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, .queueFamilyIndex = family};
    if ((result = vkCreateCommandPool(device, &poolInfo, NULL, &pool)) != VK_SUCCESS) goto done;

    if ((result = create_buffer(physical, device, COUNT * sizeof(int32_t), &input, &inputMemory)) != VK_SUCCESS) goto done;
    if ((result = create_buffer(physical, device, COUNT * sizeof(int32_t), &output, &outputMemory)) != VK_SUCCESS) goto done;
    int32_t* mapped = NULL;
    if ((result = vkMapMemory(device, inputMemory, 0, VK_WHOLE_SIZE, 0, (void**)&mapped)) != VK_SUCCESS) goto done;
    for (int32_t i = 0; i < COUNT; i++) mapped[i] = i;
    vkUnmapMemory(device, inputMemory);

    size_t codeSize = 0;
    uint32_t* code = read_file(argv[1], &codeSize);
    if (!code) {
        result = VK_ERROR_INITIALIZATION_FAILED;
        goto done;
    }
    VkShaderModuleCreateInfo shaderInfo = {.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, .codeSize = codeSize, .pCode = code};
    result = vkCreateShaderModule(device, &shaderInfo, NULL, &shader);
    free(code);
    if (result != VK_SUCCESS) goto done;
    VkDescriptorSetLayoutBinding bindings[2] = {
        {.binding = 0, .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, .descriptorCount = 1, .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT},
        {.binding = 1, .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, .descriptorCount = 1, .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT},
    };
    VkDescriptorSetLayoutCreateInfo setInfo = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO, .bindingCount = 2, .pBindings = bindings};
    if ((result = vkCreateDescriptorSetLayout(device, &setInfo, NULL, &setLayout)) != VK_SUCCESS) goto done;
    VkPipelineLayoutCreateInfo layoutInfo = {.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO, .setLayoutCount = 1, .pSetLayouts = &setLayout};
    if ((result = vkCreatePipelineLayout(device, &layoutInfo, NULL, &layout)) != VK_SUCCESS) goto done;
    VkComputePipelineCreateInfo pipelineInfo = {.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO, .layout = layout};
    pipelineInfo.stage = (VkPipelineShaderStageCreateInfo){.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, .stage = VK_SHADER_STAGE_COMPUTE_BIT, .module = shader, .pName = "main"};
    if ((result = vkCreateComputePipelines(device, VK_NULL_HANDLE, 1, &pipelineInfo, NULL, &pipeline)) != VK_SUCCESS) goto done;
    VkDescriptorPoolSize poolSize = {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 2};
    VkDescriptorPoolCreateInfo descriptorInfo = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO, .maxSets = 1, .poolSizeCount = 1, .pPoolSizes = &poolSize};
    if ((result = vkCreateDescriptorPool(device, &descriptorInfo, NULL, &descriptors)) != VK_SUCCESS) goto done;
    VkDescriptorSet set;
    VkDescriptorSetAllocateInfo setAllocation = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO, .descriptorPool = descriptors, .descriptorSetCount = 1, .pSetLayouts = &setLayout};
    if ((result = vkAllocateDescriptorSets(device, &setAllocation, &set)) != VK_SUCCESS) goto done;
    VkDescriptorBufferInfo inputInfo = {input, 0, VK_WHOLE_SIZE}, outputInfo = {output, 0, VK_WHOLE_SIZE};
    VkWriteDescriptorSet writes[2] = {
        {.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, .dstSet = set, .dstBinding = 0, .descriptorCount = 1, .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, .pBufferInfo = &inputInfo},
        {.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, .dstSet = set, .dstBinding = 1, .descriptorCount = 1, .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, .pBufferInfo = &outputInfo},
    };
    vkUpdateDescriptorSets(device, 2, writes, 0, NULL);

    VkCommandBuffer commands;
    VkCommandBufferAllocateInfo commandInfo = {.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, .commandPool = pool, .level = VK_COMMAND_BUFFER_LEVEL_PRIMARY, .commandBufferCount = 1};
    if ((result = vkAllocateCommandBuffers(device, &commandInfo, &commands)) != VK_SUCCESS) goto done;
    VkCommandBufferBeginInfo begin = {.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, .flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT};
    vkBeginCommandBuffer(commands, &begin);
    vkCmdBindPipeline(commands, VK_PIPELINE_BIND_POINT_COMPUTE, pipeline);
    vkCmdBindDescriptorSets(commands, VK_PIPELINE_BIND_POINT_COMPUTE, layout, 0, 1, &set, 0, NULL);
    vkCmdDispatch(commands, 1, 1, 1);
    VkMemoryBarrier barrier = {.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER, .srcAccessMask = VK_ACCESS_SHADER_WRITE_BIT, .dstAccessMask = VK_ACCESS_HOST_READ_BIT};
    vkCmdPipelineBarrier(commands, VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT, VK_PIPELINE_STAGE_HOST_BIT, 0, 1, &barrier, 0, NULL, 0, NULL);
    if ((result = vkEndCommandBuffer(commands)) != VK_SUCCESS) goto done;
    VkFenceCreateInfo fenceInfo = {.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO};
    if ((result = vkCreateFence(device, &fenceInfo, NULL, &fence)) != VK_SUCCESS) goto done;
    VkSubmitInfo submit = {.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO, .commandBufferCount = 1, .pCommandBuffers = &commands};
    if ((result = vkQueueSubmit(queue, 1, &submit, fence)) != VK_SUCCESS) goto done;
    if ((result = vkWaitForFences(device, 1, &fence, VK_TRUE, UINT64_MAX)) != VK_SUCCESS) goto done;

    if ((result = vkMapMemory(device, outputMemory, 0, VK_WHOLE_SIZE, 0, (void**)&mapped)) != VK_SUCCESS) goto done;
    for (int i = 0; i < COUNT; i++) sum += mapped[i];
    vkUnmapMemory(device, outputMemory);

done:
    if (device != VK_NULL_HANDLE) {
        vkDeviceWaitIdle(device);
        vkDestroyFence(device, fence, NULL);
        vkDestroyDescriptorPool(device, descriptors, NULL);
        vkDestroyPipeline(device, pipeline, NULL);
        vkDestroyPipelineLayout(device, layout, NULL);
        vkDestroyDescriptorSetLayout(device, setLayout, NULL);
        vkDestroyShaderModule(device, shader, NULL);
        vkDestroyBuffer(device, output, NULL);
        vkFreeMemory(device, outputMemory, NULL);
        vkDestroyBuffer(device, input, NULL);
        vkFreeMemory(device, inputMemory, NULL);
        vkDestroyCommandPool(device, pool, NULL);
        vkDestroyDevice(device, NULL);
    }
    if (instance != VK_NULL_HANDLE) vkDestroyInstance(instance, NULL);
    if (result != VK_SUCCESS) {
        fprintf(stderr, "Vulkan error %d\n", (int)result);
        return 1;
    }
    printf("%d\n", sum);
    return sum == 4032 ? 0 : 1;
}
