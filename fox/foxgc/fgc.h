
#ifndef FGC_H
#define FGC_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

// Optimized 8-Byte Object Header layout for absolute memory alignment
typedef struct {
    uint16_t cycle;    // 2 Bytes: The marking epoch cycle tracker
    uint16_t type_tag; // 2 Bytes: Reserved for Tagged Unions and metadata
    uint32_t flags;    // 4 Bytes: Reserved for future features and GC states
} FoxHeader; // Total size = 2 + 2 + 4 = 8 Bytes (Perfect 64-bit balance)

// Size-class descriptor aligned with the new 1MB Flat Block Map design
typedef struct {
    uintptr_t start;
    uintptr_t cursor;
    uintptr_t end;
    size_t obj_size; // Total stride (Data payload + 8-Byte FoxHeader)
} FgcClass;

// Compile-time mathematical configuration for the 1MB Flat Block Map
#define BLOCK_SIZE_BITS 20 // 2^20 Bytes equals exactly 1 Megabyte

// 1MB block size bitmask baseline
#define BLOCK_SIZE (1ULL << BLOCK_SIZE_BITS)

// 4 Gigabytes of continuous virtual memory
#define HEAP_SIZE (4ULL * 1024 * 1024 * 1024)

// Total indices in the block map (4096 blocks)
#define NUM_BLOCKS (HEAP_SIZE / BLOCK_SIZE)

// Categorization of pools stored in the fast L1 cache map array
typedef enum {
    POOL_FREE = 0,
    POOL_32_BYTE = 1,
    POOL_64_BYTE = 2,
    POOL_128_BYTE = 3,
    POOL_256_BYTE = 4,
    POOL_512_BYTE = 5,
    POOL_1024_BYTE = 6,
    POOL_2048_BYTE = 7,
    POOL_4096_BYTE = 8
} PoolType;

// Mathematical properties dictionary optimized to bypass standard integer
// division
typedef struct {
    uint32_t object_size; // Object width stride
    bool is_power_of_two; // Power of 2 optimization flag
    uint32_t size_mask;   // Fast modulo bitmask: Size - 1
    uint32_t
        magic_multiplier; // Precomputed Reciprocal Divider magic multiplier
    uint8_t shift_amount; // Precomputed Reciprocal Divider shift offset
} PoolProperties;

#define NUM_CLASSES 8

// Global Execution Anchors and Context Matrices Shared Across System Layers
extern FgcClass fgc_classes[NUM_CLASSES];
extern uint64_t global_current_cycle;
extern void *fgc_heap_base;
extern uintptr_t global_stack_top;

// Fast Metadata Lookups and Flat Block Mapping Global Registries
extern uint8_t block_map[NUM_BLOCKS];
extern PoolProperties pool_props[NUM_CLASSES + 1];

// Global Core System Function API Signatures
void fgc_init(void *main_stack_top);
void *fgc_alloc(uint8_t class_idx);
void fgc_collect(void *current_stack_bottom);
void fgc_trigger_collection(void *current_stack_bottom);

#endif // FGC_H
