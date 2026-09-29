/* Flat C boundary between the Concept Vulkan library and a Vulkan loader.
 * Each function returns plain values, so the Concept side needs no pointers:
 * handles cross as opaque pointers and status as a VkResult code. */
#ifndef CONCEPT_VULKAN_H
#define CONCEPT_VULKAN_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct VkDevice_T* VkDevice;
typedef struct VkBuffer_T* VkBuffer;
typedef struct VkDeviceMemory_T* VkDeviceMemory;

typedef struct BufferCreation {
    int32_t code;
    VkBuffer buffer;
    VkDeviceMemory memory;
} BufferCreation;

BufferCreation ConceptVkCreateBuffer(VkDevice device, size_t size, uint32_t usage, uint32_t properties);
void ConceptVkDestroyBuffer(VkDevice device, VkBuffer buffer, VkDeviceMemory memory);

#ifdef __cplusplus
}
#endif

#endif
