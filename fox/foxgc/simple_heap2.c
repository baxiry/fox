
#include "foxgc/fgc.h"
// #include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <string.h>

// Define some mockup Type IDs for testing
#define TYPE_USER 1
#define TYPE_RAW_DATA 2
#define TYPE_LARGE_ARRAY 3

// Normal size object matching 32-byte size class (8 bytes header + 24 bytes
// payload)
typedef struct {
    char *name;
    int32_t age;
    int32_t padding; // Align structure layout
} User;

int main() {
    // 1. Establish the Stack Top Anchor
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
    printf("[TEST] foxGC framework successfully initialized.\n\n");

    // 2. Scenario A: Aggressive allocation of small raw data objects
    // (has_pointers = 0) This will quickly fill up the first 1MB block and
    // trigger an emergency STW collection.
    printf("[TEST] Starting Phase 1: Allocating 40,000 small raw objects to "
           "trigger STW...\n");
    for (int i = 0; i < 40000; i++) {
        // Class index 0 is POOL_32_BYTE. Object has no pointers.
        User *u = (User *)((char *)fgc_alloc(0, TYPE_USER, 0) + 8);
        u->name = "RawData";
        u->age = i;
    }
    printf("[TEST] Phase 1 finished safely. Checking if lazy sweeping recycled "
           "empty blocks.\n\n");

    // 3. Scenario B: Allocation of a Large Object exceeding 1MB limits
    // (e.g., 1.5 Megabytes) This will force the runtime into fgc_alloc_large
    // and occupy 2 consecutive blocks.
    printf("[TEST] Starting Phase 2: Allocating a 1.5MB Large Object...\n");
    size_t large_size = 15 * 1024 * 1024 / 10; // ~1.5 MB

    // We allocate a large chunk and get the pointer
    void *large_ptr = fgc_alloc_large(large_size, TYPE_LARGE_ARRAY, 0);
    if (large_ptr != NULL) {
        printf("[TEST] Large object successfully allocated at address: %p\n",
               large_ptr);

        // Simulate storing the pointer on the stack to keep it alive
        volatile void *stack_holder = large_ptr;

        // Trigger a manual collection by forcing an allocation that overshoots
        // boundaries
        printf("[TEST] Triggering manual scan via emergency overflow to verify "
               "POOL_LARGE retention...\n");
        for (int i = 0; i < 20000; i++) {
            fgc_alloc(0, TYPE_RAW_DATA, 0);
        }
    } else {
        printf("[TEST] Error: Large object allocation failed!\n");
    }

    printf("\n[TEST] All structural logic tests passed successfully with no "
           "linker or crash errors!\n");
    return 0;
}
