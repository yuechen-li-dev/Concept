/* A GPU-free implementation of concept_vulkan.h for tests. It hands out
 * distinct fake handles, backs buffers with host memory, fails like a driver
 * would, and counts live objects so tests can observe ownership.
 *
 * It cannot run SPIR-V. A dispatch runs the host equivalent of the kernels
 * it knows by file name, so the same program can be tested with and without
 * a GPU:
 *   double.spv   binding 1[i] = 2 * binding 0[i] for every int of binding 1 */
#include <stdlib.h>
#include <string.h>

#include "concept_vulkan.h"

#define SLOTS 8

static unsigned char tokens[64];
static int live_buffers = 0;
static int live_pipelines = 0;
static int live_contexts = 0;
static int pending_submissions = 0;
static int destroyed_twice = 0;

typedef struct FakeBuffer {
    int live;
    size_t size;
    int32_t* data;
} FakeBuffer;

static FakeBuffer buffers[SLOTS];

typedef struct FakePipeline {
    int live;
    int kernel; /* 0 unknown, 1 double */
    uint32_t storageBuffers;
    int bound[SLOTS];
} FakePipeline;

static FakePipeline pipelines[SLOTS];

static void* token(int index) { return &tokens[index]; }
static int buffer_slot(VkBuffer buffer) {
    for (int i = 0; i < SLOTS; i++) if (buffer == (VkBuffer)token(i)) return i;
    return -1;
}
static int memory_slot(VkDeviceMemory memory) {
    for (int i = 0; i < SLOTS; i++) if (memory == (VkDeviceMemory)token(8 + i)) return i;
    return -1;
}
static int pipeline_slot(VkPipeline pipeline) {
    for (int i = 0; i < SLOTS; i++) if (pipeline == (VkPipeline)token(16 + i)) return i;
    return -1;
}
static int set_slot(VkDescriptorSet set) {
    for (int i = 0; i < SLOTS; i++) if (set == (VkDescriptorSet)token(24 + i)) return i;
    return -1;
}

ComputeContext ConceptVkCreateComputeContext(void) {
    ComputeContext out;
    memset(&out, 0, sizeof out);
    out.instance = (VkInstance)token(40);
    out.physical = (VkPhysicalDevice)token(41);
    out.device = (VkDevice)token(42);
    out.queue = (VkQueue)token(43);
    out.commands = (VkCommandPool)token(44);
    live_contexts++;
    return out;
}

void ConceptVkDestroyComputeContext(VkInstance instance, VkDevice device, VkCommandPool commands) {
    (void)instance;
    (void)commands;
    if (device == (VkDevice)token(42)) live_contexts--;
}

BufferCreation ConceptVkCreateBuffer(VkPhysicalDevice physical, VkDevice device, size_t size, uint32_t usage, uint32_t properties) {
    BufferCreation out = {0, 0, 0};
    (void)physical;
    (void)device;
    (void)properties;
    if (size == 0 || usage == 0) {
        out.code = -3; /* VK_ERROR_INITIALIZATION_FAILED */
        return out;
    }
    if (size > (size_t)1 << 30) {
        out.code = -2; /* VK_ERROR_OUT_OF_DEVICE_MEMORY */
        return out;
    }
    for (int i = 0; i < SLOTS; i++) {
        if (!buffers[i].live) {
            buffers[i].live = 1;
            buffers[i].size = size;
            buffers[i].data = (int32_t*)calloc(size / sizeof(int32_t) + 1, sizeof(int32_t));
            live_buffers++;
            out.buffer = (VkBuffer)token(i);
            out.memory = (VkDeviceMemory)token(8 + i);
            return out;
        }
    }
    out.code = -10; /* VK_ERROR_TOO_MANY_OBJECTS */
    return out;
}

void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory) {
    (void)device;
    (void)memory;
    int i = buffer_slot(buffer);
    if (i < 0) return;
    if (!buffers[i].live) {
        destroyed_twice++;
        return;
    }
    free(buffers[i].data);
    buffers[i].live = 0;
    buffers[i].data = NULL;
    live_buffers--;
}

int32_t ConceptVkWriteInt(VkDevice device, VkDeviceMemory memory, size_t index, int32_t value) {
    (void)device;
    int i = memory_slot(memory);
    if (i < 0 || !buffers[i].live || index >= buffers[i].size / sizeof(int32_t)) return -5; /* VK_ERROR_MEMORY_MAP_FAILED */
    buffers[i].data[index] = value;
    return 0;
}

IntRead ConceptVkReadInt(VkDevice device, VkDeviceMemory memory, size_t index) {
    IntRead out = {0, 0};
    (void)device;
    int i = memory_slot(memory);
    if (i < 0 || !buffers[i].live || index >= buffers[i].size / sizeof(int32_t)) {
        out.code = -5;
        return out;
    }
    out.value = buffers[i].data[index];
    return out;
}

static int ends_with(const char* text, const char* suffix) {
    size_t a = strlen(text), b = strlen(suffix);
    return a >= b && strcmp(text + a - b, suffix) == 0;
}

PipelineCreation ConceptVkCreateComputePipeline(VkDevice device, const char* spirvPath, uint32_t storageBuffers) {
    PipelineCreation out;
    memset(&out, 0, sizeof out);
    (void)device;
    out.storageBuffers = storageBuffers;
    if (storageBuffers == 0 || storageBuffers > SLOTS || spirvPath == NULL) {
        out.code = -3;
        return out;
    }
    for (int i = 0; i < SLOTS; i++) {
        if (!pipelines[i].live) {
            memset(&pipelines[i], 0, sizeof pipelines[i]);
            for (int b = 0; b < SLOTS; b++) pipelines[i].bound[b] = -1;
            pipelines[i].live = 1;
            pipelines[i].kernel = ends_with(spirvPath, "double.spv") ? 1 : 0;
            pipelines[i].storageBuffers = storageBuffers;
            live_pipelines++;
            out.shader = (VkShaderModule)token(32);
            out.setLayout = (VkDescriptorSetLayout)token(33);
            out.layout = (VkPipelineLayout)token(34);
            out.pipeline = (VkPipeline)token(16 + i);
            out.descriptors = (VkDescriptorPool)token(35);
            out.set = (VkDescriptorSet)token(24 + i);
            return out;
        }
    }
    out.code = -10;
    return out;
}

void ConceptVkDestroyComputePipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline, VkDescriptorPool descriptors) {
    (void)device;
    (void)shader;
    (void)setLayout;
    (void)layout;
    (void)descriptors;
    int i = pipeline_slot(pipeline);
    if (i < 0) return;
    if (!pipelines[i].live) {
        destroyed_twice++;
        return;
    }
    pipelines[i].live = 0;
    live_pipelines--;
}

void ConceptVkBindStorageBuffer(VkDevice device, VkDescriptorSet set, uint32_t binding, VkBuffer buffer, size_t size) {
    (void)device;
    (void)size;
    int p = set_slot(set);
    if (p >= 0 && binding < SLOTS) pipelines[p].bound[binding] = buffer_slot(buffer);
}

SubmissionRecord ConceptVkSubmitDispatch(VkDevice device, VkQueue queue, VkCommandPool commands, VkPipeline pipeline, VkPipelineLayout layout, VkDescriptorSet set, uint32_t groups) {
    SubmissionRecord out;
    memset(&out, 0, sizeof out);
    (void)device;
    (void)queue;
    (void)commands;
    (void)layout;
    (void)set;
    out.groups = groups;
    int p = pipeline_slot(pipeline);
    if (p < 0 || !pipelines[p].live || groups == 0) {
        out.code = -3;
        return out;
    }
    if (pipelines[p].kernel == 1) {
        int in = pipelines[p].bound[0], result = pipelines[p].bound[1];
        if (in < 0 || result < 0) {
            out.code = -3;
            return out;
        }
        size_t count = buffers[result].size / sizeof(int32_t);
        size_t available = buffers[in].size / sizeof(int32_t);
        for (size_t i = 0; i < count && i < available; i++) buffers[result].data[i] = 2 * buffers[in].data[i];
    }
    out.commands = (VkCommandBuffer)token(48);
    out.fence = (VkFence)token(49);
    pending_submissions++;
    return out;
}

int32_t ConceptVkWait(VkDevice device, VkCommandPool commands, VkCommandBuffer submitted, VkFence fence) {
    (void)device;
    (void)commands;
    (void)submitted;
    (void)fence;
    if (pending_submissions > 0) pending_submissions--;
    return 0;
}

/* Test observers. */
int ConceptVkTestLiveBuffers(void) { return live_buffers; }
int ConceptVkTestLivePipelines(void) { return live_pipelines; }
int ConceptVkTestLiveContexts(void) { return live_contexts; }
int ConceptVkTestPendingSubmissions(void) { return pending_submissions; }
int ConceptVkTestDoubleDestroys(void) { return destroyed_twice; }
