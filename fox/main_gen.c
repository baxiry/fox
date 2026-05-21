#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>

#include "foxgc/fgc.h"

int main() {
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
    for (int32_t i = 0; (i < 5); i = (i + 1)) {
                printf("Iteration: %d\n", i);
    }
    int32_t x;
    for (x = 0; (x < 5); x = (x + 1)) {
                printf("Iteration: %d\n", x);
    }
    return 0;
}

