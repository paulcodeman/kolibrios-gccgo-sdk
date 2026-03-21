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
typedef struct runtime_pool_node runtime_pool_node;
typedef struct runtime_sudog runtime_sudog;

#ifndef RUNTIME_POOL_LOCAL_CLASS_COUNT
#define RUNTIME_POOL_LOCAL_CLASS_COUNT 8u
#endif

#ifndef RUNTIME_GC_SMALL_LOCAL_CLASS_COUNT
#define RUNTIME_GC_SMALL_LOCAL_CLASS_COUNT 8u
#endif

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
    runtime_g* nextg;
    runtime_g* deadg;
    runtime_g* park_g;
    runtime_g* enqg;
    uint32_t exit_check_counter;
    uintptr_t tiny;
    uint32_t tinyoffset;
    runtime_pool_node* pool_local_lists[RUNTIME_POOL_LOCAL_CLASS_COUNT];
    uint8_t pool_local_counts[RUNTIME_POOL_LOCAL_CLASS_COUNT];
    uintptr_t pool_local_chunk_cursor[RUNTIME_POOL_LOCAL_CLASS_COUNT];
    uint32_t pool_local_chunk_remaining[RUNTIME_POOL_LOCAL_CLASS_COUNT];
    uint32_t pool_local_bytes;
    runtime_pool_node* gc_small_local_lists[RUNTIME_GC_SMALL_LOCAL_CLASS_COUNT];
    uint8_t gc_small_local_counts[RUNTIME_GC_SMALL_LOCAL_CLASS_COUNT];
    uintptr_t gc_small_local_chunk_cursor[RUNTIME_GC_SMALL_LOCAL_CLASS_COUNT];
    uint32_t gc_small_local_chunk_remaining[RUNTIME_GC_SMALL_LOCAL_CLASS_COUNT];
    runtime_sudog* sudog_local_list;
    uint8_t sudog_local_count;
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
