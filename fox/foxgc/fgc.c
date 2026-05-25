
#include "fgc.h"
#include <setjmp.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>

uint64_t global_current_cycle = 1;
void *fgc_heap_base = NULL;
uintptr_t global_stack_top = 0;
FgcClass fgc_classes[NUM_CLASSES];
uint8_t block_map[NUM_BLOCKS];

uint32_t g_max_allowed_blocks = 16;
uint32_t g_active_blocks_allocated = 8;
bool g_block_contains_live_data[NUM_BLOCKS];

// init heap
void fgc_init(void *main_stack_top) {
    global_stack_top = (uintptr_t)main_stack_top;

    fgc_heap_base = mmap(NULL, HEAP_SIZE, PROT_READ | PROT_WRITE,
                         MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);

    if (fgc_heap_base == MAP_FAILED) {
        perror("[foxGC] Fatal: Failed to map memory layout");
        exit(1);
    }

    for (size_t i = 0; i < NUM_BLOCKS; i++) {
        block_map[i] = POOL_FREE;
        g_block_contains_live_data[i] = false;
    }

    size_t configurations[NUM_CLASSES] = {32,  64,   128,  256,
                                          512, 1024, 2048, 4096};

    uintptr_t current_chunk = (uintptr_t)fgc_heap_base;
    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].start = current_chunk;
        fgc_classes[i].cursor = current_chunk;
        fgc_classes[i].end = current_chunk + BLOCK_SIZE;
        fgc_classes[i].obj_size = configurations[i];

        size_t block_idx =
            (current_chunk - (uintptr_t)fgc_heap_base) >> BLOCK_SIZE_BITS;
        block_map[block_idx] = (uint8_t)(i + 1);

        current_chunk += BLOCK_SIZE;
    }
}

// trace objects

void fgc_trace_object(uintptr_t start_obj_address, uint8_t start_p_type) {
    size_t configurations[NUM_CLASSES] = {32,  64,   128,  256,
                                          512, 1024, 2048, 4096};

    uintptr_t heap_start = (uintptr_t)fgc_heap_base;
    uintptr_t heap_end = heap_start + HEAP_SIZE;

    uintptr_t current_obj_addr = start_obj_address;
    uint8_t current_p_type = start_p_type;
    uint64_t chain_depth = 0; // counter to track how deep we go inline

    printf("[foxGC-TRACE] >>> Starting Flat Deep Tracing for graph root at: "
           "0x%lx\n",
           (unsigned long)start_obj_address);

    while (1) {
        size_t pool_stride = configurations[current_p_type - 1];
        uintptr_t current_word = current_obj_addr + 8;
        uintptr_t end_address = current_obj_addr + pool_stride;

        uintptr_t next_obj_to_trace = 0;
        uint8_t next_p_type = 0;

        while (current_word < end_address) {
            uintptr_t potential_ptr = *(uintptr_t *)current_word;
            current_word += 8;

            if (potential_ptr < heap_start || potential_ptr >= heap_end) {
                continue;
            }

            uintptr_t child_relative = potential_ptr - heap_start;
            size_t child_block_idx = child_relative >> BLOCK_SIZE_BITS;

            uint8_t child_p_type = block_map[child_block_idx];
            if (child_p_type == POOL_FREE || child_p_type == POOL_LARGE) {
                continue;
            }

            size_t child_step = configurations[child_p_type - 1];
            uintptr_t child_pool_start =
                heap_start + (child_block_idx << BLOCK_SIZE_BITS);
            uintptr_t child_offset_within_block =
                potential_ptr - child_pool_start;

            uintptr_t child_base_address =
                child_pool_start +
                (child_offset_within_block / child_step) * child_step;
            FoxHeader *child_header = (FoxHeader *)child_base_address;

            if (child_header->cycle != (uint16_t)global_current_cycle) {
                child_header->cycle = (uint16_t)global_current_cycle;
                g_block_contains_live_data[child_block_idx] = true;

                if (child_header->has_pointers == 1) {
                    next_obj_to_trace = child_base_address;
                    next_p_type = child_p_type;
                }
            }
        }

        if (next_obj_to_trace != 0) {
            current_obj_addr = next_obj_to_trace;
            current_p_type = next_p_type;
            chain_depth++;

            // Print a status update every 50,000 nodes to avoid flooding the
            // terminal screen too fast
            if (chain_depth % 50000 == 0) {
                printf("[foxGC-TRACE] Successfully protected %llu linked nodes "
                       "inline. Current address: 0x%lx\n",
                       (unsigned long long)chain_depth,
                       (unsigned long)current_obj_addr);
            }
        } else {
            printf("[foxGC-TRACE] <<< Chain end reached safely. Total inlined "
                   "nodes traced and protected: %llu\n",
                   (unsigned long long)chain_depth);
            break;
        }
    }
}

// allocate objects
void *fgc_alloc(uint8_t class_idx, uint16_t type_tag, uint8_t has_pointers) {
    FgcClass *c = &fgc_classes[class_idx];
    size_t step = c->obj_size;

    if (step > BLOCK_SIZE) {
        return fgc_alloc_large(step, type_tag, has_pointers);
    }

    while (1) {
        if (c->cursor + step > c->end) {
            uintptr_t heap_start = (uintptr_t)fgc_heap_base;
            int allocated_new_block = 0;

            if (g_active_blocks_allocated >= g_max_allowed_blocks) {
                volatile uintptr_t current_stack_anchor;
                fgc_trigger_collection((void *)&current_stack_anchor);
                c->cursor = fgc_classes[class_idx].start;
                return fgc_alloc(class_idx, type_tag, has_pointers);
            }

            for (size_t block = 0; block < NUM_BLOCKS; block++) {
                if (block_map[block] == POOL_FREE) {
                    uintptr_t new_block_addr =
                        heap_start + (block << BLOCK_SIZE_BITS);
                    block_map[block] = (uint8_t)(class_idx + 1);

                    c->start = new_block_addr;
                    c->cursor = new_block_addr;
                    c->end = new_block_addr + BLOCK_SIZE;

                    g_active_blocks_allocated++;
                    allocated_new_block = 1;
                    break;
                }
            }

            if (!allocated_new_block) {
                volatile uintptr_t current_stack_anchor;
                fgc_trigger_collection((void *)&current_stack_anchor);
                c->cursor = fgc_classes[class_idx].start;
                return fgc_alloc(class_idx, type_tag, has_pointers);
            }
        }

        FoxHeader *header = (FoxHeader *)c->cursor;

        if (header->cycle != (uint16_t)global_current_cycle) {
            void *raw_allocated_ptr = (void *)c->cursor;

            header->cycle = (uint16_t)global_current_cycle;
            header->type_tag = type_tag;
            header->has_pointers = has_pointers;

            c->cursor += step;
            return raw_allocated_ptr;
        }

        c->cursor += step;
    }
}
// allocate big objects
void *fgc_alloc_large(size_t size, uint16_t type_tag, uint8_t has_pointers) {
    size_t total_needed_blocks = (size + BLOCK_SIZE - 1) >> BLOCK_SIZE_BITS;
    uintptr_t heap_start = (uintptr_t)fgc_heap_base;
    size_t found_consecutive = 0;
    size_t start_block = 0;

    for (size_t i = 8; i < NUM_BLOCKS; i++) {
        if (block_map[i] == POOL_FREE) {
            if (found_consecutive == 0)
                start_block = i;
            found_consecutive++;
            if (found_consecutive == total_needed_blocks) {
                break;
            }
        } else {
            found_consecutive = 0;
        }
    }

    if (found_consecutive < total_needed_blocks) {
        volatile uintptr_t current_stack_anchor;
        fgc_trigger_collection((void *)&current_stack_anchor);
        return fgc_alloc_large(size, type_tag, has_pointers);
    }

    for (size_t i = start_block; i < start_block + total_needed_blocks; i++) {
        block_map[i] = POOL_LARGE;
    }

    g_active_blocks_allocated += total_needed_blocks;

    uintptr_t allocated_address = heap_start + (start_block << BLOCK_SIZE_BITS);
    FoxHeader *header = (FoxHeader *)allocated_address;
    header->cycle = (uint16_t)global_current_cycle;
    header->type_tag = type_tag;
    header->has_pointers = has_pointers;

    return (void *)allocated_address;
}

// track and clean memory

void fgc_collect(void *current_stack_bottom) {
    global_current_cycle++;

    jmp_buf regs;
    if (setjmp(regs) != 0) {
    }

    uintptr_t stack_top = global_stack_top;
    uintptr_t stack_bottom = (uintptr_t)current_stack_bottom;

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

        if (p_type == POOL_LARGE) {
            size_t start_block = block_index;
            while (start_block > 0 &&
                   block_map[start_block - 1] == POOL_LARGE) {
                start_block--;
            }
            uintptr_t large_obj_addr =
                heap_start + (start_block << BLOCK_SIZE_BITS);
            FoxHeader *live_header = (FoxHeader *)large_obj_addr;

            if (live_header->cycle != (uint16_t)global_current_cycle) {
                live_header->cycle = (uint16_t)global_current_cycle;

                size_t scan_idx = start_block;
                while (scan_idx < NUM_BLOCKS &&
                       block_map[scan_idx] == POOL_LARGE) {
                    g_block_contains_live_data[scan_idx] = true;
                    scan_idx++;
                }
            }
            continue;
        }

        size_t configurations[NUM_CLASSES] = {32,  64,   128,  256,
                                              512, 1024, 2048, 4096};
        size_t step = configurations[p_type - 1];
        uintptr_t pool_start = heap_start + (block_index << BLOCK_SIZE_BITS);
        uintptr_t offset_within_block = val - pool_start;

        uintptr_t base_obj_address =
            pool_start + (offset_within_block / step) * step;
        FoxHeader *live_header = (FoxHeader *)base_obj_address;

        if (live_header->cycle != (uint16_t)global_current_cycle) {
            live_header->cycle = (uint16_t)global_current_cycle;
            g_block_contains_live_data[block_index] = true;

            if (live_header->has_pointers == 1) {
                fgc_trace_object(base_obj_address, p_type);
            }
        }
    }

    uint32_t total_retained_blocks = 8;

    for (size_t block = 8; block < NUM_BLOCKS; block++) {
        uint8_t p_type = block_map[block];
        if (p_type != POOL_FREE) {
            if (g_block_contains_live_data[block] == false) {
                block_map[block] = POOL_FREE;
            } else {
                total_retained_blocks++;
            }
        }
    }

    if (total_retained_blocks >= g_max_allowed_blocks) {
        g_max_allowed_blocks = g_max_allowed_blocks * 2;
    }

    g_active_blocks_allocated = total_retained_blocks;
    for (size_t i = 0; i < NUM_BLOCKS; i++) {
        g_block_contains_live_data[i] = false;
    }

    for (int i = 0; i < NUM_CLASSES; i++) {
        fgc_classes[i].cursor = fgc_classes[i].start;
    }
}

// allarm

void fgc_trigger_collection(void *current_stack_bottom) {
    fgc_collect(current_stack_bottom);
}
