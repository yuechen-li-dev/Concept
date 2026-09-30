/* A GPU-free implementation of concept_vulkan.h for tests.
 *
 * It hands out distinct fake handles, backs buffers with host memory, fails
 * like a driver would, and counts live objects so tests can observe
 * ownership. It records command buffers and executes them at submit.
 *
 * It also validates synchronization: for every buffer it tracks the last
 * device write, which stages that write has been made visible to by
 * barriers, and the reads not yet ordered before a later write. An access
 * that a barrier does not cover is a hazard (ConceptVkTestHazards), and so
 * is a host read of device-written data without a barrier to the host.
 *
 * It cannot run SPIR-V. A dispatch runs the host equivalent of the kernels it
 * knows by file name:
 *   double.spv  binding 1[i] = 2 * binding 0[i]      (int32)
 *   scale.spv   binding 1[i] = binding 0[i] * scale, for i < count
 *               push constants { uint32 count; float scale; }  (float) */
#include <stdlib.h>
#include <string.h>

#include "concept_vulkan.h"

#define BUFFERS 64
#define PIPELINES 16
#define RECORDINGS 8
#define COMMANDS 256
#define STAGE_COMPUTE 1u
#define STAGE_TRANSFER 2u
#define STAGE_HOST 4u

static unsigned char tokens[1024];
static void* token(int index) { return &tokens[index]; }

static int live_buffers = 0;
static int live_pipelines = 0;
static int live_contexts = 0;
static int pending_submissions = 0;
static int destroyed_twice = 0;
static int barriers = 0;
static int hazards = 0;

/* ---- Buffers ---- */

typedef struct FakeBuffer {
    int live;
    uint32_t placement;
    uint64_t size;
    uint8_t* data;
    /* synchronization state */
    uint32_t writeStage;    /* stage of the last device write, 0 if none */
    uint32_t visibleTo;     /* stages the last write is visible to */
    uint32_t pendingReads;  /* read stages not yet ordered before a write */
    uint32_t readsOrdered;  /* stages later writes are ordered after those reads */
} FakeBuffer;

static FakeBuffer buffers[BUFFERS];

static int buffer_slot(VkBuffer buffer) {
    for (int i = 0; i < BUFFERS; i++) if (buffer == (VkBuffer)token(i)) return i;
    return -1;
}
static int memory_slot(VkDeviceMemory memory) {
    for (int i = 0; i < BUFFERS; i++) if (memory == (VkDeviceMemory)token(64 + i)) return i;
    return -1;
}

/* ---- Pipelines ---- */

typedef struct FakePipeline {
    int live;
    int kernel; /* 0 unknown, 1 double, 2 scale */
    uint32_t pushBytes;
    uint32_t bindings;
} FakePipeline;

static FakePipeline pipelines[PIPELINES];

static int pipeline_slot(VkPipeline pipeline) {
    for (int i = 0; i < PIPELINES; i++) if (pipeline == (VkPipeline)token(128 + i)) return i;
    return -1;
}

/* ---- Recordings ---- */

enum { CMD_BARRIER, CMD_DISPATCH, CMD_COPY, CMD_FILL, CMD_TIMESTAMP };

typedef struct Command {
    int kind;
    uint32_t a, b, c;          /* barrier: src, writes, dst; dispatch: x, y, z; fill: word */
    int pipeline;
    int buffers[16];
    uint32_t bindings[16];
    uint32_t count;
    uint64_t bytes;
    uint8_t push[128];
    uint32_t pushBytes;
} Command;

typedef struct FakeRecording {
    int live;
    int submitted;
    uint32_t timestamps;
    Command commands[COMMANDS];
    int count;
} FakeRecording;

static FakeRecording recordings[RECORDINGS];

static int recording_slot(VkCommandBuffer commands) {
    for (int i = 0; i < RECORDINGS; i++) if (commands == (VkCommandBuffer)token(256 + i)) return i;
    return -1;
}

static Command* append(VkCommandBuffer commands) {
    int r = recording_slot(commands);
    if (r < 0 || !recordings[r].live || recordings[r].count >= COMMANDS) return NULL;
    Command* command = &recordings[r].commands[recordings[r].count++];
    memset(command, 0, sizeof *command);
    return command;
}

/* ---- Context ---- */

ContextCreation ConceptVkCreateContext(void) {
    ContextCreation out;
    memset(&out, 0, sizeof out);
    out.instance = (VkInstance)token(512);
    out.physical = (VkPhysicalDevice)token(513);
    out.device = (VkDevice)token(514);
    out.queue = (VkQueue)token(515);
    out.commands = (VkCommandPool)token(516);
    out.cache = (VkPipelineCache)token(517);
    live_contexts++;
    return out;
}

void ConceptVkDestroyContext(VkInstance instance, VkDevice device, VkCommandPool commands, VkPipelineCache cache) {
    (void)instance;
    (void)commands;
    (void)cache;
    if (device == (VkDevice)token(514)) live_contexts--;
}

DeviceFacts ConceptVkDeviceFacts(VkPhysicalDevice physical) {
    DeviceFacts facts;
    (void)physical;
    memset(&facts, 0, sizeof facts);
    facts.apiVersion = (1u << 22) | (4u << 12);
    facts.deviceType = 2; /* discrete */
    facts.subgroupSize = 32;
    facts.maxWorkgroupInvocations = 1024;
    facts.timestampValidBits = 64;
    facts.timestampPeriod = 1.0f;
    facts.maxStorageBufferRange = (uint64_t)1 << 30;
    memcpy(facts.name, "Concept test device", sizeof "Concept test device");
    return facts;
}

/* ---- Buffers ---- */

BufferCreation ConceptVkCreateBuffer(VkPhysicalDevice physical, VkDevice device, uint64_t bytes, uint32_t placement) {
    BufferCreation out;
    (void)physical;
    (void)device;
    memset(&out, 0, sizeof out);
    if (bytes == 0 || placement > 3) {
        out.code = -3; /* VK_ERROR_INITIALIZATION_FAILED */
        return out;
    }
    if (bytes > (uint64_t)1 << 30) {
        out.code = -2; /* VK_ERROR_OUT_OF_DEVICE_MEMORY */
        return out;
    }
    for (int i = 0; i < BUFFERS; i++) {
        if (!buffers[i].live) {
            memset(&buffers[i], 0, sizeof buffers[i]);
            buffers[i].live = 1;
            buffers[i].placement = placement;
            buffers[i].size = bytes;
            buffers[i].data = (uint8_t*)calloc((size_t)bytes, 1);
            live_buffers++;
            out.hostVisible = placement == 0 ? 0u : 1u;
            out.buffer = (VkBuffer)token(i);
            out.memory = (VkDeviceMemory)token(64 + i);
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

static int mappable(int i, uint64_t offset, size_t length) {
    return i >= 0 && buffers[i].live && buffers[i].placement != 0 && offset <= buffers[i].size && length <= buffers[i].size - offset;
}

int32_t ConceptVkWriteBytes(VkDevice device, VkDeviceMemory memory, uint64_t offset, ConceptVkBytes bytes) {
    (void)device;
    int i = memory_slot(memory);
    if (!mappable(i, offset, bytes.length)) return -5; /* VK_ERROR_MEMORY_MAP_FAILED */
    memcpy(buffers[i].data + offset, bytes.data, bytes.length);
    /* Host writes before a submit are visible to it (vkQueueSubmit). */
    buffers[i].writeStage = 0;
    buffers[i].pendingReads = 0;
    return 0;
}

int32_t ConceptVkReadBytes(VkDevice device, VkDeviceMemory memory, uint64_t offset, ConceptVkMutableBytes out) {
    (void)device;
    int i = memory_slot(memory);
    if (!mappable(i, offset, out.length)) return -5;
    if (buffers[i].writeStage != 0 && !(buffers[i].visibleTo & STAGE_HOST)) hazards++;
    memcpy(out.data, buffers[i].data + offset, out.length);
    return 0;
}

/* ---- Pipelines ---- */

static int ends_with(const char* text, const char* suffix) {
    size_t a = strlen(text), b = strlen(suffix);
    return a >= b && strcmp(text + a - b, suffix) == 0;
}

PipelineCreation ConceptVkCreatePipeline(VkDevice device, VkPipelineCache cache, const char* kernelPath, const char* entry, ConceptVkBindingSlots bindings, uint32_t pushBytes) {
    PipelineCreation out;
    (void)device;
    (void)cache;
    memset(&out, 0, sizeof out);
    out.pushBytes = pushBytes;
    if (kernelPath == NULL || entry == NULL || bindings.length == 0 || bindings.length > 16 || pushBytes > 128) {
        out.code = -3;
        return out;
    }
    for (int i = 0; i < PIPELINES; i++) {
        if (!pipelines[i].live) {
            memset(&pipelines[i], 0, sizeof pipelines[i]);
            pipelines[i].live = 1;
            pipelines[i].kernel = ends_with(kernelPath, "double.spv") ? 1 : ends_with(kernelPath, "scale.spv") ? 2 : 0;
            pipelines[i].pushBytes = pushBytes;
            pipelines[i].bindings = (uint32_t)bindings.length;
            live_pipelines++;
            out.shader = (VkShaderModule)token(160 + i);
            out.setLayout = (VkDescriptorSetLayout)token(176 + i);
            out.layout = (VkPipelineLayout)token(192 + i);
            out.pipeline = (VkPipeline)token(128 + i);
            return out;
        }
    }
    out.code = -10;
    return out;
}

void ConceptVkDestroyPipeline(VkDevice device, VkShaderModule shader, VkDescriptorSetLayout setLayout, VkPipelineLayout layout, VkPipeline pipeline) {
    (void)device;
    (void)shader;
    (void)setLayout;
    (void)layout;
    int i = pipeline_slot(pipeline);
    if (i < 0) return;
    if (!pipelines[i].live) {
        destroyed_twice++;
        return;
    }
    pipelines[i].live = 0;
    live_pipelines--;
}

/* ---- Recording ---- */

RecordingBegin ConceptVkBeginRecording(VkDevice device, VkCommandPool commands, uint32_t timestamps) {
    RecordingBegin out;
    (void)device;
    (void)commands;
    memset(&out, 0, sizeof out);
    for (int i = 0; i < RECORDINGS; i++) {
        if (!recordings[i].live) {
            recordings[i].live = 1;
            recordings[i].submitted = 0;
            recordings[i].count = 0;
            recordings[i].timestamps = timestamps;
            out.timestamps = timestamps;
            out.commands = (VkCommandBuffer)token(256 + i);
            out.queries = (VkQueryPool)token(272 + i);
            return out;
        }
    }
    out.code = -10;
    return out;
}

void ConceptVkEndAbandoned(VkDevice device, VkCommandPool commands, VkCommandBuffer recording, VkQueryPool queries) {
    (void)device;
    (void)commands;
    (void)queries;
    int r = recording_slot(recording);
    if (r >= 0) recordings[r].live = 0;
}

void ConceptVkCmdBarrier(VkCommandBuffer recording, uint32_t srcStages, uint32_t srcWrites, uint32_t dstStages) {
    Command* command = append(recording);
    if (!command) return;
    command->kind = CMD_BARRIER;
    command->a = srcStages;
    command->b = srcWrites;
    command->c = dstStages;
    barriers++;
}

void ConceptVkCmdDispatch(VkCommandBuffer recording, VkPipeline pipeline, VkPipelineLayout layout, ConceptVkDescriptorWrites writes, ConceptVkBytes pushConstants, uint32_t x, uint32_t y, uint32_t z) {
    (void)layout;
    Command* command = append(recording);
    if (!command) return;
    command->kind = CMD_DISPATCH;
    command->pipeline = pipeline_slot(pipeline);
    command->a = x;
    command->b = y;
    command->c = z;
    command->count = (uint32_t)(writes.length > 16 ? 16 : writes.length);
    for (uint32_t i = 0; i < command->count; i++) {
        command->buffers[i] = buffer_slot(writes.data[i].buffer);
        command->bindings[i] = writes.data[i].binding;
    }
    command->pushBytes = (uint32_t)(pushConstants.length > 128 ? 128 : pushConstants.length);
    if (command->pushBytes) memcpy(command->push, pushConstants.data, command->pushBytes);
}

void ConceptVkCmdCopy(VkCommandBuffer recording, VkBuffer from, VkBuffer to, uint64_t bytes) {
    Command* command = append(recording);
    if (!command) return;
    command->kind = CMD_COPY;
    command->buffers[0] = buffer_slot(from);
    command->buffers[1] = buffer_slot(to);
    command->bytes = bytes;
}

void ConceptVkCmdFill(VkCommandBuffer recording, VkBuffer buffer, uint64_t bytes, uint32_t word) {
    Command* command = append(recording);
    if (!command) return;
    command->kind = CMD_FILL;
    command->buffers[0] = buffer_slot(buffer);
    command->bytes = bytes;
    command->a = word;
}

void ConceptVkCmdTimestamp(VkCommandBuffer recording, VkQueryPool queries, uint32_t index) {
    (void)queries;
    Command* command = append(recording);
    if (!command) return;
    command->kind = CMD_TIMESTAMP;
    command->a = index;
}

/* ---- Execution with synchronization checking ---- */

static void access(int b, uint32_t stage, int writes) {
    if (b < 0 || !buffers[b].live) return;
    FakeBuffer* buffer = &buffers[b];
    if (buffer->writeStage != 0 && !(buffer->visibleTo & stage)) hazards++;          /* RAW, WAW */
    if (writes && buffer->pendingReads != 0 && !(buffer->readsOrdered & stage)) hazards++; /* WAR */
    if (writes) {
        buffer->writeStage = stage;
        buffer->visibleTo = 0;
        buffer->pendingReads = 0;
        buffer->readsOrdered = 0;
    } else {
        buffer->pendingReads |= stage;
        buffer->readsOrdered = 0;
    }
}

static void barrier(uint32_t src, uint32_t srcWrites, uint32_t dst) {
    for (int b = 0; b < BUFFERS; b++) {
        FakeBuffer* buffer = &buffers[b];
        if (!buffer->live) continue;
        if (buffer->writeStage != 0 && (buffer->writeStage & src) && srcWrites) buffer->visibleTo |= dst;
        if (buffer->pendingReads != 0 && (buffer->pendingReads & ~src) == 0) buffer->readsOrdered |= dst;
    }
}

static int32_t* ints(int b) { return (int32_t*)buffers[b].data; }
static float* floats(int b) { return (float*)buffers[b].data; }

static int bound(const Command* command, uint32_t binding) {
    for (uint32_t i = 0; i < command->count; i++) if (command->bindings[i] == binding) return command->buffers[i];
    return -1;
}

static void run_dispatch(const Command* command) {
    if (command->pipeline < 0) return;
    int kernel = pipelines[command->pipeline].kernel;
    int in = bound(command, 0), out = bound(command, 1);
    /* The kernels' real access: known kernels read binding 0 and write
       binding 1; an unknown kernel is assumed to read and write everything. */
    for (uint32_t i = 0; i < command->count; i++) {
        int writes = kernel == 0 ? 1 : command->bindings[i] == 1;
        access(command->buffers[i], STAGE_COMPUTE, writes);
    }
    if (in < 0 || out < 0) return;
    if (kernel == 1) {
        uint64_t n = buffers[out].size / 4, m = buffers[in].size / 4;
        for (uint64_t i = 0; i < n && i < m; i++) ints(out)[i] = 2 * ints(in)[i];
    } else if (kernel == 2 && command->pushBytes >= 8) {
        uint32_t count;
        float scale;
        memcpy(&count, command->push, 4);
        memcpy(&scale, command->push + 4, 4);
        uint64_t n = buffers[out].size / 4, m = buffers[in].size / 4;
        for (uint64_t i = 0; i < count && i < n && i < m; i++) floats(out)[i] = floats(in)[i] * scale;
    }
}

static void execute(FakeRecording* recording) {
    for (int c = 0; c < recording->count; c++) {
        const Command* command = &recording->commands[c];
        switch (command->kind) {
        case CMD_BARRIER:
            barrier(command->a, command->b, command->c);
            break;
        case CMD_DISPATCH:
            run_dispatch(command);
            break;
        case CMD_COPY: {
            int from = command->buffers[0], to = command->buffers[1];
            access(from, STAGE_TRANSFER, 0);
            access(to, STAGE_TRANSFER, 1);
            if (from >= 0 && to >= 0 && buffers[from].live && buffers[to].live) {
                uint64_t n = command->bytes;
                if (n > buffers[from].size) n = buffers[from].size;
                if (n > buffers[to].size) n = buffers[to].size;
                memmove(buffers[to].data, buffers[from].data, (size_t)n);
            }
            break;
        }
        case CMD_FILL: {
            int b = command->buffers[0];
            access(b, STAGE_TRANSFER, 1);
            if (b >= 0 && buffers[b].live) {
                for (uint64_t i = 0; i + 4 <= command->bytes && i + 4 <= buffers[b].size; i += 4) memcpy(buffers[b].data + i, &command->a, 4);
            }
            break;
        }
        default:
            break;
        }
    }
}

SubmissionRecord ConceptVkSubmit(VkDevice device, VkQueue queue, VkCommandBuffer recording) {
    SubmissionRecord out;
    (void)device;
    (void)queue;
    memset(&out, 0, sizeof out);
    int r = recording_slot(recording);
    if (r < 0 || !recordings[r].live || recordings[r].submitted) {
        out.code = -3;
        return out;
    }
    recordings[r].submitted = 1;
    execute(&recordings[r]);
    out.fence = (VkFence)token(288 + r);
    pending_submissions++;
    return out;
}

int32_t ConceptVkWait(VkDevice device, VkFence fence) {
    (void)device;
    (void)fence;
    if (pending_submissions > 0) pending_submissions--;
    /* A host-waited fence means the submission completed: later submissions
       on this queue see its device writes and its reads are finished. Host
       reads still need the barrier to the host stage recorded at submit. */
    for (int b = 0; b < BUFFERS; b++) {
        if (!buffers[b].live) continue;
        if (buffers[b].writeStage) buffers[b].visibleTo |= STAGE_COMPUTE | STAGE_TRANSFER;
        if (buffers[b].pendingReads) buffers[b].readsOrdered |= STAGE_COMPUTE | STAGE_TRANSFER;
    }
    return 0;
}

void ConceptVkRelease(VkDevice device, VkCommandPool commands, VkCommandBuffer recording, VkFence fence, VkQueryPool queries) {
    (void)device;
    (void)commands;
    (void)fence;
    (void)queries;
    int r = recording_slot(recording);
    if (r >= 0) recordings[r].live = 0;
}

/* Timestamps advance 1000 ns per recorded command. */
Elapsed ConceptVkElapsed(VkDevice device, VkPhysicalDevice physical, VkQueryPool queries, uint32_t from, uint32_t to) {
    Elapsed out;
    (void)device;
    (void)physical;
    memset(&out, 0, sizeof out);
    int r = -1;
    for (int i = 0; i < RECORDINGS; i++) if (queries == (VkQueryPool)token(272 + i)) r = i;
    if (r < 0) {
        out.code = -3;
        return out;
    }
    int fromAt = -1, toAt = -1;
    for (int c = 0; c < recordings[r].count; c++) {
        if (recordings[r].commands[c].kind != CMD_TIMESTAMP) continue;
        if (recordings[r].commands[c].a == from) fromAt = c;
        if (recordings[r].commands[c].a == to) toAt = c;
    }
    if (fromAt < 0 || toAt < 0) {
        out.code = -3;
        return out;
    }
    out.nanoseconds = (double)(toAt - fromAt) * 1000.0;
    return out;
}

/* ---- Observers ---- */
int ConceptVkTestLiveBuffers(void) { return live_buffers; }
int ConceptVkTestLivePipelines(void) { return live_pipelines; }
int ConceptVkTestLiveContexts(void) { return live_contexts; }
int ConceptVkTestPendingSubmissions(void) { return pending_submissions; }
int ConceptVkTestDoubleDestroys(void) { return destroyed_twice; }
int ConceptVkTestBarriers(void) { return barriers; }
int ConceptVkTestHazards(void) { return hazards; }
