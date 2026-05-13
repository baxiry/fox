
#include <stdio.h>
#include "fgc.h"

// Define the absolute global epoch counter
uint64_t global_current_cycle = 1;

void fgc_init(void) {
    // Basic confirmation tracking print for our runtime
    printf("[foxGC] Runtime initialized with core logic.\n");
}
