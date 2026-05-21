
#include "fgc.h"
#include <setjmp.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>

uint64_t global_current_cycle = 1;
void *fgc_heap_base = NULL;
uintptr_t global_stack_top = 0;
FgcClass fgc_classes[NUM_CLASSES];

// Global block map matrix and size properties map with a power of 2
uint8_t block_map[NUM_BLOCKS];
PoolProperties pool_props[NUM_CLASSES + 1];

void fgc_init(void *main_stack_top) {
    global_stack_top = (uintptr_t)main_stack_top;

    // Reserve the virtual 4GB RAM
    fgc_heap_base = mmap(NULL, HEAP_SIZE, PROT_READ | PROT_WRITE,
                         MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);

    if (fgc_heap_base == MAP_FAILED) {
        perror("[foxGC] Fatal: Failed to map huge memory block layout");
        exit(1);
    }

    global_current_cycle = 1;
    uintptr_t current_chunk = (uintptr_t)fgc_heap_base;

    // Initially, all 4096 blocks are designated as free blocks (POOL_FREE).
    for (size_t i = 0; i < NUM_BLOCKS; i++) {
        block_map[i] = POOL_FREE;
    }

    size_t configurations[NUM_CLASSES] = {32,  64,   128,  256,
                                          512, 1024, 2048, 4096};

    // Fill in the constant properties matrix for powers of 2 and calculate the
    // masks in advance
    for (int i = 0; i < NUM_CLASSES; i++) {
        size_t size = configurations[i];
        PoolType p_type = (PoolType)(i + 1);

        pool_props[p_type].object_size = size;
        pool_props[p_type].is_power_of_two = (size & (size - 1)) == 0;
        pool_props[p_type].size_mask = (uint32_t)(size - 1);
        pool_props[p_type].magic_multiplier = 0;
        pool_props[p_type].shift_amount = 0;
    }

    // Limited initial grants: Only one 1MB block extension per pool at launch
    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].start = current_chunk;
        fgc_classes[i].cursor = current_chunk;
        fgc_classes[i].end =
            current_chunk + BLOCK_SIZE; // نهاية النطاق 1MB فقط!
        fgc_classes[i].obj_size = configurations[i];

        // Register the excerpted block on the global block map
        size_t block_idx =
            (current_chunk - (uintptr_t)fgc_heap_base) >> BLOCK_SIZE_BITS;
        block_map[block_idx] = (uint8_t)(i + 1);

        current_chunk += BLOCK_SIZE;
    }

    printf("[foxGC] Global 1MB Flat Block Arena established. 4GB pre-allocated "
           "at: %p\n",
           fgc_heap_base);
}

/* Core allocation sequencing interface with deterministic step execution
 * control */
void *fgc_alloc(uint8_t class_idx) {
    FgcClass *c = &fgc_classes[class_idx];
    size_t step = c->obj_size;

    /* Global static tracking counter for active memory blocks across the
     * runtime */
    static uint32_t active_blocks_allocated =
        8; /* Starts with 8 blocks initialized at boot */

    while (1) {
        if (c->cursor + step > c->end) {
            uintptr_t heap_start = (uintptr_t)fgc_heap_base;
            int allocated_new_block = 0;

            /* Explicit rigid limit: Stop and trigger STW as soon as we hit
             * 8 total megabytes */
            if (active_blocks_allocated >= 8) {
                active_blocks_allocated =
                    8; /* Reset counter to baseline startup blocks */

                volatile uintptr_t current_stack_anchor;
                fgc_trigger_collection((void *)&current_stack_anchor);

                c->cursor = fgc_classes[class_idx].start;
                return fgc_alloc(class_idx);
            }

            /* Scan the flat block map to carve a clean 1MB chunk from the free
             * pool */
            for (size_t block = 0; block < NUM_BLOCKS; block++) {
                if (block_map[block] == POOL_FREE) {
                    uintptr_t new_block_addr =
                        heap_start + (block << BLOCK_SIZE_BITS);
                    block_map[block] = (uint8_t)(class_idx + 1);

                    c->start = new_block_addr;
                    c->cursor = new_block_addr;
                    c->end = new_block_addr + BLOCK_SIZE;

                    active_blocks_allocated++; /* Advance the active block
                                                  tracking metric */
                    allocated_new_block = 1;
                    break;
                }
            }

            /* Ultimate fallback safety guard if the mapped architecture
             * completely suffocates */
            if (!allocated_new_block) {
                active_blocks_allocated = 8;
                volatile uintptr_t current_stack_anchor;
                fgc_trigger_collection((void *)&current_stack_anchor);
                c->cursor = fgc_classes[class_idx].start;
                return fgc_alloc(class_idx);
            }
        }

        FoxHeader *header = (FoxHeader *)c->cursor;

        /* Lazy sweep tracking phase to safely recycle dead objects in-place */
        if (header->cycle != (uint16_t)global_current_cycle) {
            void *raw_allocated_ptr = (void *)c->cursor;

            header->cycle = (uint16_t)global_current_cycle;
            c->cursor += step;

            return raw_allocated_ptr;
        }

        c->cursor += step;
    }
}

/* Synchronize the dynamic memory execution boundaries before scanning */
void fgc_trigger_collection(void *current_stack_bottom) {
    printf("[foxGC] Arena exhausted. Initiating absolute STW marking round.\n");
    fgc_collect(current_stack_bottom);
}

/* Single-pass linear stack scanning architecture running at maximum throughput
 */
void fgc_collect(void *current_stack_bottom) {
    global_current_cycle++;

    jmp_buf regs;
    if (setjmp(regs) != 0) {
        return;
    }

    uintptr_t stack_top = global_stack_top;
    uintptr_t stack_bottom =
        (uintptr_t)current_stack_bottom; // استخدام المرساة الممررة بدقة صخرية

    /* Enforce rigid architectural bounds alignment independently of direction
     */
    if (stack_bottom > stack_top) {
        uintptr_t temp = stack_top;
        stack_top = stack_bottom;
        stack_bottom = temp;
    }

    uintptr_t heap_start = (uintptr_t)fgc_heap_base;
    uintptr_t heap_end = heap_start + HEAP_SIZE;

    for (uintptr_t *ptr = (uintptr_t *)stack_bottom;
         ptr < (uintptr_t *)stack_top; ptr++) {
        uintptr_t val = *ptr;

        if (val < heap_start || val >= heap_end) {
            continue;
        }

        uintptr_t relative_address = val - heap_start;
        size_t block_index = relative_address >> BLOCK_SIZE_BITS;

        uint8_t p_type = block_map[block_index];
        if (p_type == POOL_FREE) {
            continue;
        }

        PoolProperties props = pool_props[p_type];
        uint32_t offset_within_block =
            (uint32_t)(relative_address & (BLOCK_SIZE - 1));
        uint32_t remainder = 1;

        if (props.is_power_of_two) {
            remainder = offset_within_block & props.size_mask;
        } else {
            uint32_t quotient = (uint32_t)(((uint64_t)offset_within_block *
                                            props.magic_multiplier) >>
                                           32) >>
                                props.shift_amount;
            remainder = offset_within_block - (quotient * props.object_size);
        }

        if (remainder == 0) {
            FoxHeader *live_header = (FoxHeader *)val;
            live_header->cycle = (uint16_t)global_current_cycle;
        }
    }

    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].cursor = fgc_classes[i].start;
    }
}
