
#include "fgc.h"
#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>

uint64_t global_current_cycle = 1;
FgcClass fgc_classes[NUM_CLASSES];
void *fgc_heap_base = NULL;

#define HEAP_SIZE (32ULL * 1024ULL) // 32KB test layout

void fgc_init(void) {
    fgc_heap_base = mmap(NULL, HEAP_SIZE, PROT_READ | PROT_WRITE,
                         MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (fgc_heap_base == MAP_FAILED) {
        perror("[foxGC] Fatal: Failed to map memory block");
        exit(1);
    }

    global_current_cycle = 1;
    uintptr_t current_chunk = (uintptr_t)fgc_heap_base;
    uintptr_t chunk_size = HEAP_SIZE / NUM_CLASSES;

    // Initial base configurations including the 8-Byte Header padding
    size_t configurations[NUM_CLASSES] = {32,  64,   128,  256,
                                          512, 1024, 2048, 4096};

    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].start = current_chunk;
        fgc_classes[i].cursor = current_chunk;
        fgc_classes[i].end = current_chunk + chunk_size;
        fgc_classes[i].obj_size = configurations[i];

        current_chunk += chunk_size;
    }

    printf("[foxGC] Heap baseline established. 32KB allocated continuously at: "
           "%p\n",
           fgc_heap_base);
}

void *fgc_alloc(uint8_t class_idx) {
    FgcClass *c = &fgc_classes[class_idx];
    size_t step = c->obj_size;
    uintptr_t scan_start = c->cursor;

    while (1) {
        if (c->cursor + step > c->end) {
            c->cursor = c->start;
        }

        FoxHeader *header = (FoxHeader *)c->cursor;

        // Core extraction check (cycle != current_cycle)
        if (header->cycle != (uint16_t)global_current_cycle) {
            void *raw_allocated_ptr = (void *)c->cursor;

            header->cycle = (uint16_t)global_current_cycle;
            c->cursor += step;

            // Return the absolute start pointer (including the 8-byte header
            // for routing)
            return raw_allocated_ptr;
        }

        c->cursor += step;

        if (c->cursor == scan_start) {
            fgc_trigger_collection();
            return fgc_alloc(class_idx);
        }
    }
}

void fgc_trigger_collection(void) {
    printf("[foxGC] Emergency: Out of memory block space! Triggering STW "
           "Collection.\n");
    global_current_cycle++;
}
