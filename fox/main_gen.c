#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>

#include "foxgc/fgc.h"

int main() {
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
        Status.Active* s = (Status.Active*)((char*)fgc_alloc(0, 1, 0) + 8);
    s->id = 777;
    return 0;
}

