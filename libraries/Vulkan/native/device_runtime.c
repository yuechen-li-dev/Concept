/* The Vulkan 1.4 implementation of concept_vulkan.h.
 *
 * It does the mechanical parts of Vulkan (create-info structs, feature
 * chains, memory-type choice, command encoding) so the Concept side states
 * only decisions. Buffers are bound with push descriptors and synchronized
 * with synchronization2 barriers, both core in Vulkan 1.4; their entry
 * points are loaded from the device so an older loader library still works.
 *
 * Environment:
 *   CONCEPT_VULKAN_DEVICE      device index, or a substring of its name
 *   CONCEPT_VULKAN_VALIDATION  1 enables VK_LAYER_KHRONOS_validation with
 *                              synchronization validation; its warnings and
 *                              errors go to stderr, and validation findings
 *                              are what ConceptVkTestHazards counts
 *   CONCEPT_VULKAN_KERNELS     directory relative kernel paths resolve in
 *
 * One context per process: the loaded device entry points are global. */
#include <vulkan/vulkan.h>

#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "concept_vulkan.h"

#define STAGE_COMPUTE 1u
#define STAGE_TRANSFER 2u
#define STAGE_HOST 4u

static int live_buffers = 0;
static int live_pipelines = 0;
static int live_contexts = 0;
static int pending_submissions = 0;
static int barriers = 0;

static PFN_vkCmdPipelineBarrier2 cmdPipelineBarrier2 = NULL;
static PFN_vkCmdPushDescriptorSet cmdPushDescriptorSet = NULL;
static PFN_vkCmdWriteTimestamp2 cmdWriteTimestamp2 = NULL;

static VkDebugUtilsMessengerEXT messenger = VK_NULL_HANDLE;
static int validation_findings = 0;

static VKAPI_ATTR VkBool32 VKAPI_CALL on_validation_message(
    VkDebugUtilsMessageSeverityFlagBitsEXT severity, VkDebugUtilsMessageTypeFlagsEXT types,
    const VkDebugUtilsMessengerCallbackDataEXT* data, void* user) {
    (void)user;
    /* Loader chatter (layer naming, API versions of unrelated layers) is
     * GENERAL; only validation messages are findings. */
    const char* message = data && data->pMessage ? data->pMessage : "";
    if (!(types & VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT)) {
        fprintf(stderr, "vulkan loader: %s\n", message);
        return VK_FALSE;
    }
    validation_findings++;
    fprintf(stderr, "vulkan %s: %s\n",
            (severity & VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT) ? "error" : "warning", message);
    return VK_FALSE;
}

static void destroy_messenger(VkInstance instance) {
    if (messenger == VK_NULL_HANDLE) return;
    PFN_vkDestroyDebugUtilsMessengerEXT destroy =
        (PFN_vkDestroyDebugUtilsMessengerEXT)vkGetInstanceProcAddr(instance, "vkDestroyDebugUtilsMessengerEXT");
    if (destroy) destroy(instance, messenger, NULL);
    messenger = VK_NULL_HANDLE;
}

/* ---- Context ---------------------------------------------------------- */

static int compute_family(VkPhysicalDevice physical, uint32_t* family, uint32_t* timestampBits) {
    uint32_t families = 0;
    vkGetPhysicalDeviceQueueFamilyProperties(physical, &families, NULL);
    VkQueueFamilyProperties properties[32];
    if (families > 32) families = 32;
    vkGetPhysicalDeviceQueueFamilyProperties(physical, &families, properties);
    for (uint32_t f = 0; f < families; f++) {
        if (properties[f].queueFlags & VK_QUEUE_COMPUTE_BIT) {
            if (family) *family = f;
            if (timestampBits) *timestampBits = properties[f].timestampValidBits;
            return 1;
        }
    }
    return 0;
}

static int contains_ignoring_case(const char* text, const char* part) {
    size_t n = strlen(text), m = strlen(part);
    for (size_t i = 0; m <= n && i <= n - m; i++) {
        size_t j = 0;
        while (j < m && tolower((unsigned char)text[i + j]) == tolower((unsigned char)part[j])) j++;
        if (j == m) return 1;
    }
    return 0;
}

static int rank(uint32_t type) {
    switch (type) {
    case VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU: return 4;
    case VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU: return 3;
    case VK_PHYSICAL_DEVICE_TYPE_VIRTUAL_GPU: return 2;
    case VK_PHYSICAL_DEVICE_TYPE_CPU: return 1;
    default: return 0;
    }
}

/* Chooses a Vulkan 1.4 device with a compute queue: the one named by
 * CONCEPT_VULKAN_DEVICE, else the highest-ranked type (discrete first). */
static VkPhysicalDevice choose_device(VkInstance instance, uint32_t* family) {
    uint32_t count = 0;
    vkEnumeratePhysicalDevices(instance, &count, NULL);
    VkPhysicalDevice physical[16];
    if (count > 16) count = 16;
    vkEnumeratePhysicalDevices(instance, &count, physical);
    const char* wanted = getenv("CONCEPT_VULKAN_DEVICE");
    int byIndex = wanted && *wanted && isdigit((unsigned char)wanted[0]);
    VkPhysicalDevice best = VK_NULL_HANDLE;
    int bestRank = -1;
    for (uint32_t d = 0; d < count; d++) {
        VkPhysicalDeviceProperties properties;
        vkGetPhysicalDeviceProperties(physical[d], &properties);
        uint32_t f = 0;
        if (properties.apiVersion < VK_API_VERSION_1_4 || !compute_family(physical[d], &f, NULL)) continue;
        if (wanted && *wanted) {
            if ((byIndex && (uint32_t)atoi(wanted) == d) || (!byIndex && contains_ignoring_case(properties.deviceName, wanted))) {
                *family = f;
                return physical[d];
            }
            continue;
        }
        if (rank(properties.deviceType) > bestRank) {
            bestRank = rank(properties.deviceType);
            best = physical[d];
            *family = f;
        }
    }
    return best;
}

static void destroy_partial(ContextCreation* out) {
    if (out->commands) vkDestroyCommandPool(out->device, out->commands, NULL);
    if (out->device) vkDestroyDevice(out->device, NULL);
    if (out->instance) {
        destroy_messenger(out->instance);
        vkDestroyInstance(out->instance, NULL);
    }
    VkResult code = (VkResult)out->code;
    memset(out, 0, sizeof *out);
    out->code = code;
}

static ContextCreation impl_CreateContext(void) {
    ContextCreation out;
    memset(&out, 0, sizeof out);

    VkApplicationInfo app = {.sType = VK_STRUCTURE_TYPE_APPLICATION_INFO};
    app.pApplicationName = "Concept";
    app.apiVersion = VK_API_VERSION_1_4;
    VkInstanceCreateInfo instanceInfo = {.sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO};
    instanceInfo.pApplicationInfo = &app;
    const char* validation = "VK_LAYER_KHRONOS_validation";
    const char* extensions[] = {VK_EXT_DEBUG_UTILS_EXTENSION_NAME, VK_EXT_LAYER_SETTINGS_EXTENSION_NAME};
    const VkBool32 on = VK_TRUE;
    VkLayerSettingEXT syncValidation = {validation, "validate_sync", VK_LAYER_SETTING_TYPE_BOOL32_EXT, 1, &on};
    VkLayerSettingsCreateInfoEXT validationSettings = {.sType = VK_STRUCTURE_TYPE_LAYER_SETTINGS_CREATE_INFO_EXT};
    validationSettings.settingCount = 1;
    validationSettings.pSettings = &syncValidation;
    VkDebugUtilsMessengerCreateInfoEXT messengerInfo = {.sType = VK_STRUCTURE_TYPE_DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT};
    messengerInfo.messageSeverity = VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT | VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT;
    messengerInfo.messageType = VK_DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT | VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT;
    messengerInfo.pfnUserCallback = on_validation_message;
    const char* wantValidation = getenv("CONCEPT_VULKAN_VALIDATION");
    int validate = wantValidation && strcmp(wantValidation, "1") == 0;
    if (validate) {
        instanceInfo.enabledLayerCount = 1;
        instanceInfo.ppEnabledLayerNames = &validation;
        instanceInfo.enabledExtensionCount = 2;
        instanceInfo.ppEnabledExtensionNames = extensions;
        validationSettings.pNext = &messengerInfo; /* covers instance creation and destruction */
        instanceInfo.pNext = &validationSettings;
    }
    out.code = vkCreateInstance(&instanceInfo, NULL, &out.instance);
    if (out.code != VK_SUCCESS) {
        out.instance = VK_NULL_HANDLE;
        return out;
    }
    if (validate) {
        PFN_vkCreateDebugUtilsMessengerEXT create =
            (PFN_vkCreateDebugUtilsMessengerEXT)vkGetInstanceProcAddr(out.instance, "vkCreateDebugUtilsMessengerEXT");
        out.code = create ? create(out.instance, &messengerInfo, NULL, &messenger) : VK_ERROR_EXTENSION_NOT_PRESENT;
        if (out.code != VK_SUCCESS) {
            messenger = VK_NULL_HANDLE;
            destroy_partial(&out);
            return out;
        }
        fprintf(stderr, "vulkan: VK_LAYER_KHRONOS_validation enabled with synchronization validation\n");
    }

    out.physical = choose_device(out.instance, &out.queueFamily);
    if (out.physical == VK_NULL_HANDLE) {
        out.code = VK_ERROR_INCOMPATIBLE_DRIVER;
        destroy_partial(&out);
        return out;
    }

    VkPhysicalDeviceVulkan14Features supported14 = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_4_FEATURES};
    VkPhysicalDeviceVulkan13Features supported13 = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_3_FEATURES, .pNext = &supported14};
    VkPhysicalDeviceFeatures2 supported = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2, .pNext = &supported13};
    vkGetPhysicalDeviceFeatures2(out.physical, &supported);
    if (!supported13.synchronization2 || !supported14.pushDescriptor) {
        out.code = VK_ERROR_FEATURE_NOT_PRESENT;
        destroy_partial(&out);
        return out;
    }
    VkPhysicalDeviceVulkan14Features enabled14 = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_4_FEATURES};
    enabled14.pushDescriptor = VK_TRUE;
    VkPhysicalDeviceVulkan13Features enabled13 = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_VULKAN_1_3_FEATURES, .pNext = &enabled14};
    enabled13.synchronization2 = VK_TRUE;
    VkPhysicalDeviceFeatures2 enabled = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_FEATURES_2, .pNext = &enabled13};

    float priority = 1.0f;
    VkDeviceQueueCreateInfo queueInfo = {.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO};
    queueInfo.queueFamilyIndex = out.queueFamily;
    queueInfo.queueCount = 1;
    queueInfo.pQueuePriorities = &priority;
    VkDeviceCreateInfo deviceInfo = {.sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO, .pNext = &enabled};
    deviceInfo.queueCreateInfoCount = 1;
    deviceInfo.pQueueCreateInfos = &queueInfo;
    out.code = vkCreateDevice(out.physical, &deviceInfo, NULL, &out.device);
    if (out.code != VK_SUCCESS) {
        out.device = VK_NULL_HANDLE;
        destroy_partial(&out);
        return out;
    }
    cmdPipelineBarrier2 = (PFN_vkCmdPipelineBarrier2)vkGetDeviceProcAddr(out.device, "vkCmdPipelineBarrier2");
    cmdPushDescriptorSet = (PFN_vkCmdPushDescriptorSet)vkGetDeviceProcAddr(out.device, "vkCmdPushDescriptorSet");
    cmdWriteTimestamp2 = (PFN_vkCmdWriteTimestamp2)vkGetDeviceProcAddr(out.device, "vkCmdWriteTimestamp2");
    if (!cmdPipelineBarrier2 || !cmdPushDescriptorSet || !cmdWriteTimestamp2) {
        out.code = VK_ERROR_INCOMPATIBLE_DRIVER;
        destroy_partial(&out);
        return out;
    }
    vkGetDeviceQueue(out.device, out.queueFamily, 0, &out.queue);

    VkCommandPoolCreateInfo poolInfo = {.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO};
    poolInfo.queueFamilyIndex = out.queueFamily;
    poolInfo.flags = VK_COMMAND_POOL_CREATE_TRANSIENT_BIT;
    out.code = vkCreateCommandPool(out.device, &poolInfo, NULL, &out.commands);
    if (out.code != VK_SUCCESS) {
        out.commands = VK_NULL_HANDLE;
        destroy_partial(&out);
        return out;
    }
    VkPipelineCacheCreateInfo cacheInfo = {.sType = VK_STRUCTURE_TYPE_PIPELINE_CACHE_CREATE_INFO};
    out.code = vkCreatePipelineCache(out.device, &cacheInfo, NULL, &out.cache);
    if (out.code != VK_SUCCESS) {
        out.cache = VK_NULL_HANDLE;
        destroy_partial(&out);
    }
    return out;
}

static void impl_DestroyContext(VkInstance instance, VkDevice device, VkCommandPool commands, VkPipelineCache cache) {
    vkDeviceWaitIdle(device);
    vkDestroyPipelineCache(device, cache, NULL);
    vkDestroyCommandPool(device, commands, NULL);
    vkDestroyDevice(device, NULL);
    destroy_messenger(instance);
    vkDestroyInstance(instance, NULL);
}

DeviceFacts ConceptVkDeviceFacts(VkPhysicalDevice physical) {
    DeviceFacts facts;
    memset(&facts, 0, sizeof facts);
    VkPhysicalDeviceSubgroupProperties subgroup = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_SUBGROUP_PROPERTIES};
    VkPhysicalDeviceProperties2 properties = {.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2, .pNext = &subgroup};
    vkGetPhysicalDeviceProperties2(physical, &properties);
    facts.apiVersion = properties.properties.apiVersion;
    facts.deviceType = (uint32_t)properties.properties.deviceType;
    facts.subgroupSize = subgroup.subgroupSize;
    facts.maxWorkgroupInvocations = properties.properties.limits.maxComputeWorkGroupInvocations;
    facts.timestampPeriod = properties.properties.limits.timestampPeriod;
    facts.maxStorageBufferRange = properties.properties.limits.maxStorageBufferRange;
    compute_family(physical, NULL, &facts.timestampValidBits);
    size_t length = strlen(properties.properties.deviceName);
    if (length >= sizeof facts.name) length = sizeof facts.name - 1;
    memcpy(facts.name, properties.properties.deviceName, length);
    return facts;
}

/* ---- Buffers ---------------------------------------------------------- */

/* Memory-type preferences per placement: the first requirement set that a
 * type satisfies wins; within it, the lowest type index. */
static uint32_t choose_memory(VkPhysicalDevice physical, uint32_t allowed, uint32_t placement) {
    const VkMemoryPropertyFlags visible = VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT;
    VkMemoryPropertyFlags want[3] = {0, 0, 0};
    VkMemoryPropertyFlags avoid[3] = {0, 0, 0};
    int options = 0;
    switch (placement) {
    case 0:
        want[0] = VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT;
        options = 1;
        break;
    case 1:
        want[0] = visible; avoid[0] = VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT;
        want[1] = visible;
        options = 2;
        break;
    case 2:
        want[0] = visible | VK_MEMORY_PROPERTY_HOST_CACHED_BIT;
        want[1] = visible;
        options = 2;
        break;
    default:
        want[0] = visible | VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT;
        want[1] = visible;
        options = 2;
        break;
    }
    VkPhysicalDeviceMemoryProperties memory;
    vkGetPhysicalDeviceMemoryProperties(physical, &memory);
    for (int o = 0; o < options; o++) {
        for (uint32_t i = 0; i < memory.memoryTypeCount; i++) {
            VkMemoryPropertyFlags flags = memory.memoryTypes[i].propertyFlags;
            if ((allowed & (1u << i)) && (flags & want[o]) == want[o] && (flags & avoid[o]) == 0) return i;
        }
    }
    return UINT32_MAX;
}

static BufferCreation impl_CreateBuffer(VkPhysicalDevice physical, VkDevice device, uint64_t bytes, uint32_t placement) {
    BufferCreation out;
    memset(&out, 0, sizeof out);
    if (bytes == 0 || placement > 3) {
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        return out;
    }
    VkBufferCreateInfo bufferInfo = {.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO};
    bufferInfo.size = bytes;
    bufferInfo.usage = VK_BUFFER_USAGE_STORAGE_BUFFER_BIT | VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT | VK_BUFFER_USAGE_TRANSFER_SRC_BIT | VK_BUFFER_USAGE_TRANSFER_DST_BIT;
    bufferInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    out.code = vkCreateBuffer(device, &bufferInfo, NULL, &out.buffer);
    if (out.code != VK_SUCCESS) {
        out.buffer = VK_NULL_HANDLE;
        return out;
    }
    VkMemoryRequirements requirements;
    vkGetBufferMemoryRequirements(device, out.buffer, &requirements);
    uint32_t type = choose_memory(physical, requirements.memoryTypeBits, placement);
    if (type == UINT32_MAX) {
        out.code = VK_ERROR_FEATURE_NOT_PRESENT;
        vkDestroyBuffer(device, out.buffer, NULL);
        out.buffer = VK_NULL_HANDLE;
        return out;
    }
    VkPhysicalDeviceMemoryProperties memory;
    vkGetPhysicalDeviceMemoryProperties(physical, &memory);
    out.hostVisible = (memory.memoryTypes[type].propertyFlags & VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) ? 1u : 0u;
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

int32_t ConceptVkWriteBytes(VkDevice device, VkDeviceMemory memory, uint64_t offset, ConceptVkBytes bytes) {
    if (bytes.length == 0) return VK_SUCCESS;
    void* mapped = NULL;
    VkResult code = vkMapMemory(device, memory, offset, bytes.length, 0, &mapped);
    if (code != VK_SUCCESS) return code;
    memcpy(mapped, bytes.data, bytes.length);
    vkUnmapMemory(device, memory);
    return VK_SUCCESS;
}

int32_t ConceptVkReadBytes(VkDevice device, VkDeviceMemory memory, uint64_t offset, ConceptVkMutableBytes out) {
    if (out.length == 0) return VK_SUCCESS;
    void* mapped = NULL;
    VkResult code = vkMapMemory(device, memory, offset, out.length, 0, &mapped);
    if (code != VK_SUCCESS) return code;
    memcpy(out.data, mapped, out.length);
    vkUnmapMemory(device, memory);
    return VK_SUCCESS;
}

/* ---- Pipelines -------------------------------------------------------- */

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

static PipelineCreation impl_CreatePipeline(VkDevice device, VkPipelineCache cache, const char* kernelPath, const char* entry, ConceptVkBindingSlots bindings, uint32_t pushBytes) {
    PipelineCreation out;
    memset(&out, 0, sizeof out);
    out.pushBytes = pushBytes;
    if (bindings.length == 0 || bindings.length > 16 || pushBytes > 256 || pushBytes % 4 != 0) {
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        return out;
    }
    size_t bytes = 0;
    uint32_t* words = read_spirv(kernelPath, &bytes);
    if (!words) {
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        return out;
    }
    VkShaderModuleCreateInfo shaderInfo = {.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO};
    shaderInfo.codeSize = bytes;
    shaderInfo.pCode = words;
    out.code = vkCreateShaderModule(device, &shaderInfo, NULL, &out.shader);
    free(words);
    if (out.code != VK_SUCCESS) {
        memset(&out, 0, sizeof out);
        out.code = VK_ERROR_INITIALIZATION_FAILED;
        return out;
    }

    VkDescriptorSetLayoutBinding layoutBindings[16];
    for (size_t i = 0; i < bindings.length; i++) {
        memset(&layoutBindings[i], 0, sizeof layoutBindings[i]);
        layoutBindings[i].binding = bindings.data[i].binding;
        layoutBindings[i].descriptorType = bindings.data[i].kind == 1 ? VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER : VK_DESCRIPTOR_TYPE_STORAGE_BUFFER;
        layoutBindings[i].descriptorCount = 1;
        layoutBindings[i].stageFlags = VK_SHADER_STAGE_COMPUTE_BIT;
    }
    VkDescriptorSetLayoutCreateInfo setInfo = {.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO};
    setInfo.flags = VK_DESCRIPTOR_SET_LAYOUT_CREATE_PUSH_DESCRIPTOR_BIT;
    setInfo.bindingCount = (uint32_t)bindings.length;
    setInfo.pBindings = layoutBindings;
    out.code = vkCreateDescriptorSetLayout(device, &setInfo, NULL, &out.setLayout);
    if (out.code == VK_SUCCESS) {
        VkPushConstantRange range = {VK_SHADER_STAGE_COMPUTE_BIT, 0, pushBytes};
        VkPipelineLayoutCreateInfo layoutInfo = {.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO};
        layoutInfo.setLayoutCount = 1;
        layoutInfo.pSetLayouts = &out.setLayout;
        layoutInfo.pushConstantRangeCount = pushBytes > 0 ? 1u : 0u;
        layoutInfo.pPushConstantRanges = &range;
        out.code = vkCreatePipelineLayout(device, &layoutInfo, NULL, &out.layout);
    }
    if (out.code == VK_SUCCESS) {
        VkComputePipelineCreateInfo pipelineInfo = {.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO};
        pipelineInfo.stage.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
        pipelineInfo.stage.stage = VK_SHADER_STAGE_COMPUTE_BIT;
        pipelineInfo.stage.module = out.shader;
        pipelineInfo.stage.pName = entry;
        pipelineInfo.layout = out.layout;
        out.code = vkCreateComputePipelines(device, cache, 1, &pipelineInfo, NULL, &out.pipeline);
    }
    if (out.code != VK_SUCCESS) {
        /* A failed creation owns nothing. */
        if (out.pipeline) vkDestroyPipeline(device, out.pipeline, NULL);
        if (out.layout) vkDestroyPipelineLayout(device, out.layout, NULL);
        if (out.setLayout) vkDestroyDescriptorSetLayout(device, out.setLayout, NULL);
        vkDestroyShaderModule(device, out.shader, NULL);
        int32_t code = out.code;
        memset(&out, 0, sizeof out);
        out.code = code;
    }
    return out;
}

static void impl_DestroyPipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline) {
    vkDestroyPipeline(device, pipeline, NULL);
    vkDestroyPipelineLayout(device, layout, NULL);
    vkDestroyDescriptorSetLayout(device, setLayout, NULL);
    vkDestroyShaderModule(device, shader, NULL);
}

/* ---- Recording -------------------------------------------------------- */

RecordingBegin ConceptVkBeginRecording(VkDevice device, VkCommandPool commands, uint32_t timestamps) {
    RecordingBegin out;
    memset(&out, 0, sizeof out);
    out.timestamps = timestamps;
    VkCommandBufferAllocateInfo allocation = {.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO};
    allocation.commandPool = commands;
    allocation.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    allocation.commandBufferCount = 1;
    out.code = vkAllocateCommandBuffers(device, &allocation, &out.commands);
    if (out.code != VK_SUCCESS) {
        out.commands = VK_NULL_HANDLE;
        return out;
    }
    if (timestamps > 0) {
        VkQueryPoolCreateInfo queryInfo = {.sType = VK_STRUCTURE_TYPE_QUERY_POOL_CREATE_INFO};
        queryInfo.queryType = VK_QUERY_TYPE_TIMESTAMP;
        queryInfo.queryCount = timestamps;
        out.code = vkCreateQueryPool(device, &queryInfo, NULL, &out.queries);
        if (out.code != VK_SUCCESS) {
            vkFreeCommandBuffers(device, commands, 1, &out.commands);
            out.commands = VK_NULL_HANDLE;
            out.queries = VK_NULL_HANDLE;
            return out;
        }
    }
    VkCommandBufferBeginInfo begin = {.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO};
    begin.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    out.code = vkBeginCommandBuffer(out.commands, &begin);
    if (out.code == VK_SUCCESS && out.queries) vkCmdResetQueryPool(out.commands, out.queries, 0, timestamps);
    if (out.code != VK_SUCCESS) {
        if (out.queries) vkDestroyQueryPool(device, out.queries, NULL);
        vkFreeCommandBuffers(device, commands, 1, &out.commands);
        out.commands = VK_NULL_HANDLE;
        out.queries = VK_NULL_HANDLE;
    }
    return out;
}

void ConceptVkEndAbandoned(VkDevice device, VkCommandPool commands, VkCommandBuffer recording, VkQueryPool queries) {
    if (queries) vkDestroyQueryPool(device, queries, NULL);
    vkFreeCommandBuffers(device, commands, 1, &recording);
}

static VkPipelineStageFlags2 stage_flags(uint32_t stages) {
    VkPipelineStageFlags2 flags = 0;
    if (stages & STAGE_COMPUTE) flags |= VK_PIPELINE_STAGE_2_COMPUTE_SHADER_BIT;
    if (stages & STAGE_TRANSFER) flags |= VK_PIPELINE_STAGE_2_TRANSFER_BIT;
    if (stages & STAGE_HOST) flags |= VK_PIPELINE_STAGE_2_HOST_BIT;
    return flags;
}

static VkAccessFlags2 write_access(uint32_t stages) {
    VkAccessFlags2 flags = 0;
    if (stages & STAGE_COMPUTE) flags |= VK_ACCESS_2_SHADER_STORAGE_WRITE_BIT;
    if (stages & STAGE_TRANSFER) flags |= VK_ACCESS_2_TRANSFER_WRITE_BIT;
    if (stages & STAGE_HOST) flags |= VK_ACCESS_2_HOST_WRITE_BIT;
    return flags;
}

static VkAccessFlags2 any_access(uint32_t stages) {
    VkAccessFlags2 flags = write_access(stages);
    if (stages & STAGE_COMPUTE) flags |= VK_ACCESS_2_SHADER_STORAGE_READ_BIT | VK_ACCESS_2_UNIFORM_READ_BIT;
    if (stages & STAGE_TRANSFER) flags |= VK_ACCESS_2_TRANSFER_READ_BIT;
    if (stages & STAGE_HOST) flags |= VK_ACCESS_2_HOST_READ_BIT;
    return flags;
}

void ConceptVkCmdBarrier(VkCommandBuffer recording, uint32_t srcStages, uint32_t srcWrites, uint32_t dstStages) {
    VkMemoryBarrier2 barrier = {.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER_2};
    barrier.srcStageMask = stage_flags(srcStages);
    barrier.srcAccessMask = srcWrites ? write_access(srcStages) : 0;
    barrier.dstStageMask = stage_flags(dstStages);
    barrier.dstAccessMask = srcWrites ? any_access(dstStages) : 0;
    VkDependencyInfo dependency = {.sType = VK_STRUCTURE_TYPE_DEPENDENCY_INFO};
    dependency.memoryBarrierCount = 1;
    dependency.pMemoryBarriers = &barrier;
    cmdPipelineBarrier2(recording, &dependency);
    barriers++;
}

void ConceptVkCmdDispatch(VkCommandBuffer recording, VkPipeline pipeline, VkPipelineLayout layout, ConceptVkDescriptorWrites writes, ConceptVkBytes pushConstants, uint32_t x, uint32_t y, uint32_t z) {
    VkDescriptorBufferInfo infos[16];
    VkWriteDescriptorSet sets[16];
    size_t count = writes.length > 16 ? 16 : writes.length;
    for (size_t i = 0; i < count; i++) {
        infos[i].buffer = writes.data[i].buffer;
        infos[i].offset = writes.data[i].offset;
        infos[i].range = writes.data[i].range;
        memset(&sets[i], 0, sizeof sets[i]);
        sets[i].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
        sets[i].dstBinding = writes.data[i].binding;
        sets[i].descriptorCount = 1;
        sets[i].descriptorType = writes.data[i].kind == 1 ? VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER : VK_DESCRIPTOR_TYPE_STORAGE_BUFFER;
        sets[i].pBufferInfo = &infos[i];
    }
    vkCmdBindPipeline(recording, VK_PIPELINE_BIND_POINT_COMPUTE, pipeline);
    cmdPushDescriptorSet(recording, VK_PIPELINE_BIND_POINT_COMPUTE, layout, 0, (uint32_t)count, sets);
    if (pushConstants.length > 0) vkCmdPushConstants(recording, layout, VK_SHADER_STAGE_COMPUTE_BIT, 0, (uint32_t)pushConstants.length, pushConstants.data);
    vkCmdDispatch(recording, x, y, z);
}

void ConceptVkCmdCopy(VkCommandBuffer recording, VkBuffer from, VkBuffer to, uint64_t bytes) {
    VkBufferCopy region = {0, 0, bytes};
    vkCmdCopyBuffer(recording, from, to, 1, &region);
}

void ConceptVkCmdFill(VkCommandBuffer recording, VkBuffer buffer, uint64_t bytes, uint32_t word) {
    vkCmdFillBuffer(recording, buffer, 0, bytes & ~(uint64_t)3, word);
}

void ConceptVkCmdTimestamp(VkCommandBuffer recording, VkQueryPool queries, uint32_t index) {
    cmdWriteTimestamp2(recording, VK_PIPELINE_STAGE_2_ALL_COMMANDS_BIT, queries, index);
}

static SubmissionRecord impl_Submit(VkDevice device, VkQueue queue, VkCommandBuffer recording) {
    SubmissionRecord out;
    memset(&out, 0, sizeof out);
    out.code = vkEndCommandBuffer(recording);
    if (out.code != VK_SUCCESS) return out;
    VkFenceCreateInfo fenceInfo = {.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO};
    out.code = vkCreateFence(device, &fenceInfo, NULL, &out.fence);
    if (out.code != VK_SUCCESS) {
        out.fence = VK_NULL_HANDLE;
        return out;
    }
    VkSubmitInfo submit = {.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO};
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &recording;
    out.code = vkQueueSubmit(queue, 1, &submit, out.fence);
    if (out.code != VK_SUCCESS) {
        vkDestroyFence(device, out.fence, NULL);
        out.fence = VK_NULL_HANDLE;
    }
    return out;
}

static int32_t impl_Wait(VkDevice device, VkFence fence) {
    return vkWaitForFences(device, 1, &fence, VK_TRUE, UINT64_MAX);
}

void ConceptVkRelease(VkDevice device, VkCommandPool commands, VkCommandBuffer recording, VkFence fence, VkQueryPool queries) {
    if (fence) vkDestroyFence(device, fence, NULL);
    if (queries) vkDestroyQueryPool(device, queries, NULL);
    vkFreeCommandBuffers(device, commands, 1, &recording);
}

Elapsed ConceptVkElapsed(VkDevice device, VkPhysicalDevice physical, VkQueryPool queries, uint32_t from, uint32_t to) {
    Elapsed out;
    memset(&out, 0, sizeof out);
    uint64_t first = 0, second = 0;
    out.code = vkGetQueryPoolResults(device, queries, from, 1, sizeof first, &first, sizeof first, VK_QUERY_RESULT_64_BIT | VK_QUERY_RESULT_WAIT_BIT);
    if (out.code == VK_SUCCESS) out.code = vkGetQueryPoolResults(device, queries, to, 1, sizeof second, &second, sizeof second, VK_QUERY_RESULT_64_BIT | VK_QUERY_RESULT_WAIT_BIT);
    if (out.code != VK_SUCCESS) return out;
    DeviceFacts facts = ConceptVkDeviceFacts(physical);
    uint64_t mask = facts.timestampValidBits >= 64 ? ~(uint64_t)0 : (((uint64_t)1 << facts.timestampValidBits) - 1);
    out.nanoseconds = (double)((second - first) & mask) * (double)facts.timestampPeriod;
    return out;
}

/* ---- Exported entry points that keep the lifetime counts -------------- */

ContextCreation ConceptVkCreateContext(void) {
    ContextCreation out = impl_CreateContext();
    if (out.code == VK_SUCCESS) live_contexts++;
    return out;
}

void ConceptVkDestroyContext(VkInstance instance, VkDevice device, VkCommandPool commands, VkPipelineCache cache) {
    impl_DestroyContext(instance, device, commands, cache);
    live_contexts--;
}

BufferCreation ConceptVkCreateBuffer(VkPhysicalDevice physical, VkDevice device, uint64_t bytes, uint32_t placement) {
    BufferCreation out = impl_CreateBuffer(physical, device, bytes, placement);
    if (out.code == VK_SUCCESS) live_buffers++;
    return out;
}

void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory) {
    impl_DestroyBuffer(device, buffer, memory);
    live_buffers--;
}

PipelineCreation ConceptVkCreatePipeline(VkDevice device, VkPipelineCache cache, const char* kernelPath, const char* entry, ConceptVkBindingSlots bindings, uint32_t pushBytes) {
    PipelineCreation out = impl_CreatePipeline(device, cache, kernelPath, entry, bindings, pushBytes);
    if (out.code == VK_SUCCESS) live_pipelines++;
    return out;
}

void ConceptVkDestroyPipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline) {
    impl_DestroyPipeline(device, shader, setLayout, layout, pipeline);
    live_pipelines--;
}

SubmissionRecord ConceptVkSubmit(VkDevice device, VkQueue queue, VkCommandBuffer recording) {
    SubmissionRecord out = impl_Submit(device, queue, recording);
    if (out.code == VK_SUCCESS) pending_submissions++;
    return out;
}

int32_t ConceptVkWait(VkDevice device, VkFence fence) {
    int32_t code = impl_Wait(device, fence);
    if (pending_submissions > 0) pending_submissions--;
    return code;
}

int ConceptVkTestLiveBuffers(void) { return live_buffers; }
int ConceptVkTestLivePipelines(void) { return live_pipelines; }
int ConceptVkTestLiveContexts(void) { return live_contexts; }
int ConceptVkTestPendingSubmissions(void) { return pending_submissions; }
int ConceptVkTestDoubleDestroys(void) { return 0; }
int ConceptVkTestBarriers(void) { return barriers; }
/* A real device sees hazards only through the validation layer
 * (CONCEPT_VULKAN_VALIDATION=1); without it this stays 0. */
int ConceptVkTestHazards(void) { return validation_findings; }
