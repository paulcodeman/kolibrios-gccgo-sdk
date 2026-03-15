#ifndef RUNTIME_PANIC_H
#define RUNTIME_PANIC_H

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

struct go_type_descriptor;

typedef struct {
    const struct go_type_descriptor* type;
    const void* data;
} go_empty_interface;

typedef struct runtime_panic runtime_panic;
typedef struct runtime_defer runtime_defer;
typedef struct runtime_g runtime_g;
typedef struct runtime_m runtime_m;

struct runtime_panic {
    runtime_panic* link;
    go_empty_interface arg;
    uint8_t recovered;
    uint8_t isforeign;
    uint8_t aborted;
    uint8_t goexit;
};

struct runtime_defer {
    runtime_defer* link;
    uint8_t* frame;
    runtime_panic* panic_stack;
    runtime_panic* panic;
    uintptr_t pfn;
    void* arg;
    uintptr_t retaddr;
    uint8_t makefunccanrecover;
    uint8_t heap;
};

struct runtime_m {
    runtime_g* curg;
    runtime_g* gsignal;
    int32_t mallocing;
    const char* preemptoff;
    int32_t locks;
};

struct runtime_g {
    runtime_m* m;
    runtime_defer* _defer;
    runtime_panic* _panic;
    void* exception;
    uint32_t sig;
    uintptr_t entrysp;
    uint8_t deferring;
    uint8_t goexiting;
    uint8_t isforeign;
    uint8_t ranCgocallBackDone;
};

runtime_g* runtime_getg(void);

#ifdef __cplusplus
}
#endif

#endif
