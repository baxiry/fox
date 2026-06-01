#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>

#include "foxgc/fgc.h"

typedef struct Status {
    int32_t _tag;
    union {
        struct {
            int32_t id;
        } Active;
        struct {
            int32_t reason;
        } Inactive;
    } variants;
} Status;

int main() {
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
        Status* s = (Status*)((char*)fgc_alloc(0, 1, 0) + 8);
    s->_tag = 2;
    s->variants.Inactive.reason = 0;
    switch (s->_tag) {
    case 1:
        printf("Status is active with ID: %d\n", s->variants.Active.id); break;
    case 2:
        printf("s in inactive %d\n", s->variants.Inactive.reason); break;
    }
    return 0;
}

