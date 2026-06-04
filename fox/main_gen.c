#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>

#include "foxgc/fgc.h"

typedef struct Error {
    char* msg;
    int32_t code;
} Error;

typedef struct _Result_int {
    FoxHeader header;
    union {
        int success;
        Error error;
    } value;
} _Result_int;

_Result_int divide(int32_t a, int32_t b) {
    if (b == 0) {
        _Result_int __ret_env;
        __ret_env.header.error_flag = 1;
        __ret_env.value.error.msg = "cannot divide by zero";
        __ret_env.value.error.code = 400;
        return __ret_env;
    }

_Result_int __ret_env;
    __ret_env.header.error_flag = 0;
    __ret_env.value.success = a / b;
    return __ret_env;
}

int main() {
    int32_t stack_top_anchor;
    fgc_init(&stack_top_anchor);
_Result_int res = divide(10, 0);
    switch (res.header.error_flag) {
    case 1:
        printf("Operation failed! Error: %s (Code: %d)\n", res.value.error.msg, res.value.error.code); break;
    default:
        printf("Operation succeeded! Result is: %d\n", res.value.success); break;
    }
    res = divide(10, 10);
    switch (res.header.error_flag) {
    case 1:
        printf("Operation failed! Error: %s (Code: %d)\n", res.value.error.msg, res.value.error.code); break;
    default:
        printf("Operation succeeded! Result is: %d\n", res.value.success); break;
    }
    return 0;
}

