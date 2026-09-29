/* A GPU-free implementation of concept_vulkan.h for package tests. It hands
 * out distinct fake handles, fails like a driver would, and counts live
 * objects so tests can observe ownership. */
#include "concept_vulkan.h"

#define TEST_DEVICE_CAPACITY 8

static unsigned char buffers[TEST_DEVICE_CAPACITY];
static unsigned char memories[TEST_DEVICE_CAPACITY];
static int live[TEST_DEVICE_CAPACITY];
static int destroyed_twice = 0;

BufferCreation ConceptVkCreateBuffer(VkDevice device, size_t size, uint32_t usage, uint32_t properties) {
    BufferCreation out = {0, 0, 0};
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
    for (int i = 0; i < TEST_DEVICE_CAPACITY; i++) {
        if (!live[i]) {
            live[i] = 1;
            out.buffer = (VkBuffer)(void*)&buffers[i];
            out.memory = (VkDeviceMemory)(void*)&memories[i];
            return out;
        }
    }
    out.code = -10; /* VK_ERROR_TOO_MANY_OBJECTS */
    return out;
}

void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory) {
    (void)device;
    (void)memory;
    for (int i = 0; i < TEST_DEVICE_CAPACITY; i++) {
        if (buffer == (VkBuffer)(void*)&buffers[i]) {
            if (!live[i]) destroyed_twice++;
            live[i] = 0;
        }
    }
}

int ConceptVkTestLiveBuffers(void) {
    int count = 0;
    for (int i = 0; i < TEST_DEVICE_CAPACITY; i++) count += live[i];
    return count;
}

int ConceptVkTestDoubleDestroys(void) {
    return destroyed_twice;
}

static unsigned char device_storage;

VkDevice ConceptVkTestDevice(void) {
    return (VkDevice)(void*)&device_storage;
}
