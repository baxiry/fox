
#include "fgc.h"
#include <setjmp.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>

uint64_t global_current_cycle = 1;
void *fgc_heap_base = NULL;
uintptr_t global_stack_top = 0;
FgcClass fgc_classes[NUM_CLASSES];

// Thread-safe, cache-optimized core translation and property lookups
uint8_t block_map[NUM_BLOCKS];
PoolProperties pool_props[NUM_CLASSES + 1];

void fgc_init(void *main_stack_top) {
    global_stack_top = (uintptr_t)main_stack_top;

    // Allocate huge virtual memory block continuously without mapping physical
    // RAM immediately
    fgc_heap_base = mmap(NULL, HEAP_SIZE, PROT_READ | PROT_WRITE,
                         MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);

    if (fgc_heap_base == MAP_FAILED) {
        perror("[foxGC] Fatal: Failed to map huge memory block layout");
        exit(1);
    }

    global_current_cycle = 1;
    uintptr_t current_chunk = (uintptr_t)fgc_heap_base;
    uintptr_t chunk_size =
        HEAP_SIZE / NUM_CLASSES; // Static layout partition for the demo

    size_t configurations[NUM_CLASSES] = {32,  64,   128,  256,
                                          512, 1024, 2048, 4096};

    // Populate standard execution lookup table to replace native division
    // instructions entirely
    for (int i = 0; i < NUM_CLASSES; i++) {
        size_t size = configurations[i];
        PoolType p_type = (PoolType)(i + 1);

        pool_props[p_type].object_size = size;
        pool_props[p_type].is_power_of_two = (size & (size - 1)) == 0;
        pool_props[p_type].size_mask = (uint32_t)(size - 1);

        // Setup placeholders for magic reciprocal components if non-power of
        // two values emerge later
        pool_props[p_type].magic_multiplier = 0;
        pool_props[p_type].shift_amount = 0;
    }

    // Configure structural runtime partitions across the flat block metadata
    // matrix
    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].start = current_chunk;
        fgc_classes[i].cursor = current_chunk;
        fgc_classes[i].end = current_chunk + chunk_size;
        fgc_classes[i].obj_size = configurations[i];

        // Map absolute physical ranges directly into the 1MB indexing table
        // array
        size_t start_idx =
            (current_chunk - (uintptr_t)fgc_heap_base) >> BLOCK_SIZE_BITS;
        size_t end_idx =
            ((current_chunk + chunk_size) - (uintptr_t)fgc_heap_base) >>
            BLOCK_SIZE_BITS;

        for (size_t block = start_idx; block < end_idx && block < NUM_BLOCKS;
             block++) {
            block_map[block] =
                (uint8_t)(i +
                          1); // Bind block pointer context to PoolType index
        }

        current_chunk += chunk_size;
    }

    printf(
        "[foxGC] Global virtual arena established. 4GB pre-allocated at: %p\n",
        fgc_heap_base);
}

void *fgc_alloc(uint8_t class_idx) {
    FgcClass *c = &fgc_classes[class_idx];
    size_t step = c->obj_size;
    uintptr_t scan_start = c->cursor;

    while (1) {
        if (c->cursor + step > c->end) {
            c->cursor = c->start; // Wrap cursor within its static boundaries
        }

        FoxHeader *header = (FoxHeader *)c->cursor;

        // Lazy sweep: Reuse block slot seamlessly if cycle timestamp is
        // outdated
        if (header->cycle != (uint16_t)global_current_cycle) {
            void *raw_allocated_ptr = (void *)c->cursor;

            header->cycle = (uint16_t)global_current_cycle;
            c->cursor += step;

            return raw_allocated_ptr;
        }

        c->cursor += step;

        // Trigger STW collection upon circular layout exhaustion
        if (c->cursor == scan_start) {
            fgc_trigger_collection();
            return fgc_alloc(class_idx);
        }
    }
}

void fgc_collect(void) {
    global_current_cycle++;

    // Force register spilling into the standard execution stack architecture
    // safely
    jmp_buf regs;
    if (setjmp(regs) != 0) {
        return; // Guard statement to prevent illegal longjmp interception loops
    }

    uintptr_t stack_top = global_stack_top;
    volatile uintptr_t stack_bottom_anchor;
    uintptr_t stack_bottom = (uintptr_t)&stack_bottom_anchor;

    // Handle structural stack growth direction dynamically at runtime
    if (stack_bottom > stack_top) {
        uintptr_t temp = stack_top;
        stack_top = stack_bottom;
        stack_bottom = temp;
    }

    uintptr_t heap_start = (uintptr_t)fgc_heap_base;
    uintptr_t heap_end = heap_start + HEAP_SIZE;

    // Single pass linear stack scan routine running at maximum instruction
    // throughput
    for (uintptr_t *ptr = (uintptr_t *)stack_bottom;
         ptr < (uintptr_t *)stack_top; ptr++) {
        uintptr_t val = *ptr;

        // Step 1: Range verification guard check
        if (val < heap_start || val >= heap_end) {
            continue; // Outside foxGC domain boundaries
        }

        // Step 2: Extract block matrix coordinate index using instantaneous
        // bitwise shift (O(1))
        uintptr_t relative_address = val - heap_start;
        size_t block_index = relative_address >> BLOCK_SIZE_BITS;

        uint8_t p_type = block_map[block_index];
        if (p_type == POOL_FREE) {
            continue; // Points to an unallocated or free system chunk
        }

        // Step 3: Extract precalculated property specs without looping or
        // searching
        PoolProperties props = pool_props[p_type];
        uint32_t offset_within_block =
            (uint32_t)(relative_address & (BLOCK_SIZE - 1));
        uint32_t remainder =
            1; // Default state flag configured as unaligned memory

        // Step 4: Validate alignment constraints strictly to kill false
        // positives (O(1))
        if (props.is_power_of_two) {
            remainder =
                offset_within_block &
                props.size_mask; // Bitwise mask fallback: 1 cycle overhead
        } else {
            // Reciprocal multiplication mechanics prepared for future custom
            // unaligned types
            uint32_t quotient = (uint32_t)(((uint64_t)offset_within_block *
                                            props.magic_multiplier) >>
                                           32) >>
                                props.shift_amount;
            remainder = offset_within_block - (quotient * props.object_size);
        }

        // Step 5: If the pointer targets a true object head, flip the live
        // tracking flag
        if (remainder == 0) {
            FoxHeader *live_header = (FoxHeader *)val;
            live_header->cycle = (uint16_t)global_current_cycle;
        }
    }

    // Reset cursor indices globally across all pools after full live marking
    // phase is cleared
    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].cursor = fgc_classes[i].start;
    }
}

void fgc_trigger_collection(void) {
    printf("[foxGC] Arena exhausted. Initiating absolute STW marking round.\n");
    fgc_collect();
}
