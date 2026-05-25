#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>

#include "foxgc/fgc.h"

typedef struct GNode {
    int32_t id;
    struct GNode* link;
} GNode;

int main() {
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
    int32_t i;
    for (i = 0; (i < 500000); i = (i + 1)) {
                    GNode* nodeA = (GNode*)((char*)fgc_alloc(0, 1, 1) + 8);
        nodeA->id = 1;
                    GNode* nodeB = (GNode*)((char*)fgc_alloc(0, 1, 1) + 8);
        nodeB->id = 2;
                nodeA->link = nodeB;
                nodeB->link = nodeA;
    }
    return 0;
}

