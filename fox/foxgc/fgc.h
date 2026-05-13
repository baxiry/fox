
#ifndef FGC_H
#define FGC_H

#include <stdint.h>
#include <stdbool.h>

// Forward declaration of global tracking cycle
extern uint64_t global_current_cycle;

// Forward declaration of runtime initialization function
void fgc_init(void);

#endif // FGC_H

