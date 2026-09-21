#include <gc.h>
#include <stdio.h>
#include <stdlib.h>
#include <time.h>

typedef struct GNode {
    int id;
    struct GNode *link;
} GNode;

int main() {
    // تهيئة Boehm GC
    GC_INIT();

    clock_t start = clock();

    for (int i = 0; i < 500000; i++) {
        // حجز عقدتين على الـ Heap عبر Boehm GC
        GNode *nodeA = (GNode *)GC_MALLOC(sizeof(GNode));
        GNode *nodeB = (GNode *)GC_MALLOC(sizeof(GNode));

        nodeA->id = 1;
        nodeB->id = 2;

        // إغلاق الدائرة
        nodeA->link = nodeB;
        nodeB->link = nodeA;

        // بمجرد انتهاء الدورة تفقد المؤشرات مرجعها من الـ Stack
    }

    clock_t end = clock();
    double cpu_time_used = ((double)(end - start)) / CLOCKS_PER_SEC;

    printf("Boehm GC Time: %.4f seconds\n", cpu_time_used);
    return 0;
}
