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
typedef struct runtime_context runtime_context;

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
    uint32_t tid;
    runtime_g* g0;
    struct runtime_m* next;
};

struct runtime_context {
    uint32_t ebx;
    uint32_t esi;
    uint32_t edi;
    uint32_t ebp;
    uint32_t esp;
    uint32_t eip;
};

struct runtime_g {
    runtime_m* m;
    runtime_m* lockedm;
    runtime_defer* _defer;
    runtime_panic* _panic;
    void* exception;
    uint32_t sig;
    uintptr_t entrysp;
    uint8_t deferring;
    uint8_t goexiting;
    uint8_t isforeign;
    uint8_t ranCgocallBackDone;
    runtime_context context;
    void (*entry)(void*);
    void* entry_arg;
    runtime_g* sched_next;
    runtime_g* all_next;
    void* stack_base;
    uintptr_t stack_top;
    uint32_t stack_size;
    uint32_t status;
    uint32_t parking;
    int32_t select_done;
    int32_t select_recvok;
};

runtime_g* runtime_getg(void);

#ifdef __cplusplus
}
#endif

#endif
