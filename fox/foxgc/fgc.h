
#ifndef FGC_H
#define FGC_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

// Optimized 8-Byte Object Header layout for absolute memory alignment
typedef struct {
    uint16_t
        cycle; // 2 Bytes: The marking epoch cycle tracker (your exact idea)
    uint16_t type_tag; // 2 Bytes: Reserved for Tagged Unions and metadata
    uint32_t flags;    // 4 Bytes: Reserved for future features and GC states
} FoxHeader; // Total size = 2 + 2 + 4 = 8 Bytes (Perfect 64-bit balance)

// Size-class descriptor
typedef struct {
    uintptr_t start;
    uintptr_t cursor;
    uintptr_t end;
    size_t obj_size; // Total stride (Data payload + 8-Byte FoxHeader)
} FgcClass;

#define NUM_CLASSES 8
extern FgcClass fgc_classes[NUM_CLASSES];
extern uint64_t global_current_cycle;

void fgc_init(void);
void *fgc_alloc(uint8_t class_idx);
void fgc_trigger_collection(void);

#endif // FGC_H
