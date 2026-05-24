#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>

#include "foxgc/fgc.h"

typedef struct {
    int32_t id;
    int32_t padding[256];
} User;

int main() {
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
    int32_t i;
    for (i = 0; (i < 1000000); i = (i + 1)) {
                    User* u = (User*)((char*)fgc_alloc(6, 1, 0) + 8);
        u->id = 123;
    }
    return 0;
}

