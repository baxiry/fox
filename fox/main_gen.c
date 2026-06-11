#include <stdio.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
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
    int32_t res = ({
        _Result_int __tmp_err_env = divide(10, 20);
        if (__tmp_err_env.header.error_flag == 1) {
            printf("Runtime Panic: unhandled error in function main! Message: %s\n", __tmp_err_env.value.error.msg);
            exit(1);
        }
        __tmp_err_env.value.success;
    });
    printf("Operation succeeded! Result is: %d\n", res);
    res = ({
        _Result_int __tmp_err_env = divide(10, 20);
        if (__tmp_err_env.header.error_flag == 1) {
            printf("Runtime Panic: unhandled error in function main! Message: %s\n", __tmp_err_env.value.error.msg);
            exit(1);
        }
        __tmp_err_env.value.success;
    }) + 30;
    printf("Operation succeeded! Result is: %d\n", res);
    res = ({
        _Result_int __tmp_err_env = divide(10, 0);
        if (__tmp_err_env.header.error_flag == 1) {
            printf("Runtime Panic: unhandled error in function main! Message: %s\n", __tmp_err_env.value.error.msg);
            exit(1);
        }
        __tmp_err_env.value.success;
    }) + 30;
    printf("error: %d\n", res);
    return 0;
}

