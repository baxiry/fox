
#ifndef FGC_H
#define FGC_H

// #include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifndef STDBOOL_H_REPLACEMENT
#define STDBOOL_H_REPLACEMENT
#define bool _Bool
#define true 1
#define false 0
#endif

// Aligned 8-Byte Object Header layout optimized for rapid branchless tracing
typedef struct {
    uint16_t cycle;    // 2 Bytes: The marking epoch cycle tracker
    uint16_t type_tag; // 2 Bytes: Supports over 65k enterprise types & unions
    uint8_t has_pointers; // 1 Byte: Explicit flag (0 = primitive, 1 = trace)
    uint8_t
        padding[3]; // 3 Bytes: Pure trailing padding reserved for thread locks
} FoxHeader; // Total size = 2 + 2 + 1 + 3 = 8 Bytes (Perfect 64-bit alignment)

// Size-class descriptor matching the dynamic adaptive block map framework
typedef struct {
    uintptr_t start;
    uintptr_t cursor;
    uintptr_t end;
    size_t obj_size;
} FgcClass;

#define BLOCK_SIZE_BITS 20
#define BLOCK_SIZE (1ULL << BLOCK_SIZE_BITS)  // Exactly 1 Megabyte
#define HEAP_SIZE (4ULL * 1024 * 1024 * 1024) // 4 Gigabytes
#define NUM_BLOCKS (HEAP_SIZE / BLOCK_SIZE)   // 4096 blocks

typedef enum {
    POOL_FREE = 0,
    POOL_32_BYTE = 1,
    POOL_64_BYTE = 2,
    POOL_128_BYTE = 3,
    POOL_256_BYTE = 4,
    POOL_512_BYTE = 5,
    POOL_1024_BYTE = 6,
    POOL_2048_BYTE = 7,
    POOL_4096_BYTE = 8,
    POOL_LARGE = 9 // Dedicated marker for objects extending beyond 1MB limits
} PoolType;

#define NUM_CLASSES 8

extern FgcClass fgc_classes[NUM_CLASSES];
extern uint64_t global_current_cycle;
extern void *fgc_heap_base;
extern uintptr_t global_stack_top;
extern uint8_t block_map[NUM_BLOCKS];

void fgc_init(void *main_stack_top);
void *fgc_alloc(uint8_t class_idx, uint16_t type_tag, uint8_t has_pointers);
void *fgc_alloc_large(size_t size, uint16_t type_tag, uint8_t has_pointers);
void fgc_collect(void *current_stack_bottom);
void fgc_trigger_collection(void *current_stack_bottom);

#endif // FGC_H
