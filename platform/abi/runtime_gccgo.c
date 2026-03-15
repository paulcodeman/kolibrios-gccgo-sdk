#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include <unwind.h>

#include "runtime_panic.h"

#define RUNTIME_USED __attribute__((used))

#ifndef KOLIBRI_CUSTOM_UNWIND_FDE
#define KOLIBRI_CUSTOM_UNWIND_FDE 1
#ifndef KOLIBRI_UNWIND_DEBUG
#define KOLIBRI_UNWIND_DEBUG 0
#endif
#endif

#ifndef KOLIBRI_RT_DEBUG
#define KOLIBRI_RT_DEBUG 0
#endif

#ifndef KOLIBRI_RT_DEBUG_FILE
#define KOLIBRI_RT_DEBUG_FILE 0
#endif

extern void* malloc(size_t size);
extern void* realloc(void* ptr, size_t size);
extern void free(void* ptr);
void* memset(void* dest, int value, size_t size);
void* memmove(void* dest, const void* src, size_t size);
extern uint32_t runtime_kos_heap_init_raw(void);
extern uint32_t runtime_kos_heap_alloc_raw(uint32_t size);
extern uint32_t runtime_kos_heap_free_raw(uint32_t ptr);
extern uint32_t runtime_kos_heap_realloc_raw(uint32_t size, uint32_t ptr);
extern uint32_t runtime_kos_load_dll_cstring_raw(const char* path);
extern uint32_t runtime_kos_get_free_ram_raw(void) __asm__("go_0kos.GetFreeRAM");
extern int32_t runtime_kos_get_current_thread_slot_raw(void) __asm__("go_0kos.GetCurrentThreadSlotRaw");
extern int32_t runtime_kos_create_thread_raw(uint32_t entry, uint32_t stack) __asm__("go_0kos.CreateThreadRaw");
extern int32_t runtime_kos_get_thread_info_raw(uint8_t* buffer, int32_t slot) __asm__("go_0kos.GetThreadInfo");
extern void runtime_kos_exit_raw(void) __asm__("go_0kos.ExitRaw");
extern void runtime_kolibri_thread_entry(void);
void runtime_console_bridge_close(uint32_t close_window);

typedef struct {
    uint32_t state;
} runtime_mutex;

static void runtime_lock_mutex(runtime_mutex* m);
static void runtime_unlock_mutex(runtime_mutex* m);
static inline uint32_t runtime_atomic_load_u32(const volatile uint32_t* value);
static inline void runtime_atomic_store_u32(uint32_t* value, uint32_t next);
static inline bool runtime_atomic_cas_u32(uint32_t* value, uint32_t expected, uint32_t desired);
__attribute__((noreturn)) static void runtime_exit_process(void);

// Minimal pthread stubs for libgcc_eh on KolibriOS (single-threaded runtime).
// libgcc uses weak pthread_* symbols; providing no-ops avoids hangs in __register_frame.
// TODO: replace with real pthread/TLS hooks once the runtime supports multi-threading.
typedef int pthread_once_t;
typedef unsigned int pthread_key_t;

int pthread_once(pthread_once_t* once_control, void (*init_routine)(void)) {
    if (once_control != NULL && *once_control == 0) {
        *once_control = 1;
        if (init_routine != NULL) {
            init_routine();
        }
    }
    return 0;
}

int pthread_mutex_lock(void* mutex) {
    (void)mutex;
    return 0;
}

int pthread_mutex_unlock(void* mutex) {
    (void)mutex;
    return 0;
}

int pthread_cond_wait(void* cond, void* mutex) {
    (void)cond;
    (void)mutex;
    return 0;
}

int pthread_cond_broadcast(void* cond) {
    (void)cond;
    return 0;
}

int pthread_key_create(pthread_key_t* key, void (*destructor)(void*)) {
    static pthread_key_t next_key = 1;
    if (key != NULL) {
        *key = next_key++;
    }
    (void)destructor;
    return 0;
}

int __pthread_key_create(pthread_key_t* key, void (*destructor)(void*)) {
    return pthread_key_create(key, destructor);
}

void* pthread_getspecific(pthread_key_t key) {
    (void)key;
    return NULL;
}

int pthread_setspecific(pthread_key_t key, const void* value) {
    (void)key;
    (void)value;
    return 0;
}

// Size-class pool for future runtime subsystems (rendering, etc.).
#define RUNTIME_POOL_MIN_SHIFT 5u
#define RUNTIME_POOL_MAX_SHIFT 20u
#define RUNTIME_POOL_CLASS_COUNT (RUNTIME_POOL_MAX_SHIFT - RUNTIME_POOL_MIN_SHIFT + 1u)
#define RUNTIME_POOL_MAGIC 0x504F4F4Cu
#define RUNTIME_POOL_CLASS_INDEX_SYSTEM 0xFFFFu
#define RUNTIME_POOL_MAX_CACHED_BYTES (2u * 1024u * 1024u)
#define RUNTIME_POOL_CLASS_BYTES_LIMIT (256u * 1024u)
#define RUNTIME_POOL_CLASS_MIN_CACHED 2u
#define RUNTIME_POOL_CLASS_MAX_CACHED 256u

static void* kos_memcpy(void* dest, const void* src, size_t size);

typedef struct runtime_pool_node {
    struct runtime_pool_node* next;
} runtime_pool_node;

static runtime_pool_node* runtime_pool_free_lists[RUNTIME_POOL_CLASS_COUNT];
static uint16_t runtime_pool_free_counts[RUNTIME_POOL_CLASS_COUNT];
static size_t runtime_pool_cached_bytes = 0;
static runtime_mutex runtime_pool_lock;

typedef struct runtime_pool_header {
    uint32_t magic;
    uint16_t class_index;
    uint16_t flags;
    uint32_t reserved0;
    uint32_t reserved1;
} runtime_pool_header;

#define RUNTIME_POOL_HEADER_SIZE ((size_t)sizeof(runtime_pool_header))

static int runtime_pool_class_index(size_t size, size_t* class_size_out) {
    size_t class_size;
    uint32_t shift;

    if (size == 0) {
        size = 1;
    }

    shift = RUNTIME_POOL_MIN_SHIFT;
    class_size = ((size_t)1u) << shift;
    while (class_size < size && shift < RUNTIME_POOL_MAX_SHIFT) {
        class_size <<= 1;
        shift++;
    }
    if (class_size < size) {
        return -1;
    }
    if (class_size_out != NULL) {
        *class_size_out = class_size;
    }

    return (int)(shift - RUNTIME_POOL_MIN_SHIFT);
}

static uint16_t runtime_pool_class_cap(size_t class_size) {
    size_t cap;

    if (class_size == 0) {
        return 0;
    }

    cap = RUNTIME_POOL_CLASS_BYTES_LIMIT / class_size;
    if (cap < RUNTIME_POOL_CLASS_MIN_CACHED) {
        cap = RUNTIME_POOL_CLASS_MIN_CACHED;
    }
    if (cap > RUNTIME_POOL_CLASS_MAX_CACHED) {
        cap = RUNTIME_POOL_CLASS_MAX_CACHED;
    }
    return (uint16_t)cap;
}

static void* runtime_pool_alloc(size_t size) {
    size_t class_size;
    int class_index;
    runtime_pool_node* node;

    class_index = runtime_pool_class_index(size, &class_size);
    if (class_index < 0) {
        return malloc(size);
    }

    node = runtime_pool_free_lists[class_index];
    if (node != NULL) {
        runtime_pool_free_lists[class_index] = node->next;
        if (runtime_pool_free_counts[class_index] > 0) {
            runtime_pool_free_counts[class_index]--;
        }
        if (runtime_pool_cached_bytes >= class_size) {
            runtime_pool_cached_bytes -= class_size;
        } else {
            runtime_pool_cached_bytes = 0;
        }
        return (void*)node;
    }

    return malloc(class_size);
}

static void runtime_pool_release(void* ptr, size_t size) {
    size_t class_size;
    uint16_t class_cap;
    int class_index;
    runtime_pool_node* node;

    if (ptr == NULL) {
        return;
    }

    class_index = runtime_pool_class_index(size, &class_size);
    if (class_index < 0) {
        free(ptr);
        return;
    }

    class_cap = runtime_pool_class_cap(class_size);
    if (class_cap == 0 ||
        runtime_pool_free_counts[class_index] >= class_cap ||
        runtime_pool_cached_bytes + class_size > RUNTIME_POOL_MAX_CACHED_BYTES) {
        free(ptr);
        return;
    }

    node = (runtime_pool_node*)ptr;
    node->next = runtime_pool_free_lists[class_index];
    runtime_pool_free_lists[class_index] = node;
    runtime_pool_free_counts[class_index]++;
    runtime_pool_cached_bytes += class_size;
}

static runtime_pool_header* runtime_pool_header_from_payload(void* payload) {
    return (runtime_pool_header*)((uint8_t*)payload - RUNTIME_POOL_HEADER_SIZE);
}

static void runtime_pool_header_init(runtime_pool_header* header, uint16_t class_index) {
    header->magic = RUNTIME_POOL_MAGIC;
    header->class_index = class_index;
    header->flags = 0;
    header->reserved0 = 0;
    header->reserved1 = 0;
}

void* runtime_pool_malloc(size_t size) {
    runtime_pool_header* header = NULL;
    size_t total;
    int class_index;
    void* result = NULL;

    runtime_lock_mutex(&runtime_pool_lock);
    if (size > (size_t)-1 - RUNTIME_POOL_HEADER_SIZE) {
        runtime_unlock_mutex(&runtime_pool_lock);
        return NULL;
    }

    total = size + RUNTIME_POOL_HEADER_SIZE;
    class_index = runtime_pool_class_index(total, NULL);
    if (class_index < 0) {
        header = (runtime_pool_header*)malloc(total);
        if (header == NULL) {
            runtime_unlock_mutex(&runtime_pool_lock);
            return NULL;
        }
        runtime_pool_header_init(header, RUNTIME_POOL_CLASS_INDEX_SYSTEM);
        result = (void*)((uint8_t*)header + RUNTIME_POOL_HEADER_SIZE);
        runtime_unlock_mutex(&runtime_pool_lock);
        return result;
    }

    header = (runtime_pool_header*)runtime_pool_alloc(total);
    if (header == NULL) {
        runtime_unlock_mutex(&runtime_pool_lock);
        return NULL;
    }
    runtime_pool_header_init(header, (uint16_t)class_index);
    result = (void*)((uint8_t*)header + RUNTIME_POOL_HEADER_SIZE);
    runtime_unlock_mutex(&runtime_pool_lock);
    return result;
}

void runtime_pool_free(void* ptr) {
    runtime_pool_header* header;
    uint16_t class_index;
    size_t class_size;

    if (ptr == NULL) {
        return;
    }

    runtime_lock_mutex(&runtime_pool_lock);
    header = runtime_pool_header_from_payload(ptr);
    if (header->magic != RUNTIME_POOL_MAGIC) {
        runtime_unlock_mutex(&runtime_pool_lock);
        free(ptr);
        return;
    }

    class_index = header->class_index;
    if (class_index >= RUNTIME_POOL_CLASS_COUNT && class_index != RUNTIME_POOL_CLASS_INDEX_SYSTEM) {
        runtime_unlock_mutex(&runtime_pool_lock);
        free(header);
        return;
    }
    if (class_index == RUNTIME_POOL_CLASS_INDEX_SYSTEM) {
        runtime_unlock_mutex(&runtime_pool_lock);
        free(header);
        return;
    }

    class_size = ((size_t)1u) << ((size_t)class_index + RUNTIME_POOL_MIN_SHIFT);
    runtime_pool_release(header, class_size);
    runtime_unlock_mutex(&runtime_pool_lock);
}

void* runtime_pool_realloc(void* ptr, size_t size) {
    runtime_pool_header* header;
    uint16_t class_index;
    size_t class_size;
    size_t old_capacity;
    size_t total;
    void* out;

    if (ptr == NULL) {
        return runtime_pool_malloc(size);
    }
    if (size == 0) {
        runtime_pool_free(ptr);
        return NULL;
    }

    runtime_lock_mutex(&runtime_pool_lock);
    header = runtime_pool_header_from_payload(ptr);
    if (header->magic != RUNTIME_POOL_MAGIC) {
        runtime_unlock_mutex(&runtime_pool_lock);
        return realloc(ptr, size);
    }

    class_index = header->class_index;
    if (class_index >= RUNTIME_POOL_CLASS_COUNT && class_index != RUNTIME_POOL_CLASS_INDEX_SYSTEM) {
        class_index = RUNTIME_POOL_CLASS_INDEX_SYSTEM;
    }
    if (class_index == RUNTIME_POOL_CLASS_INDEX_SYSTEM) {
        if (size > (size_t)-1 - RUNTIME_POOL_HEADER_SIZE) {
            runtime_unlock_mutex(&runtime_pool_lock);
            return NULL;
        }
        total = size + RUNTIME_POOL_HEADER_SIZE;
        header = (runtime_pool_header*)realloc(header, total);
        if (header == NULL) {
            runtime_unlock_mutex(&runtime_pool_lock);
            return NULL;
        }
        runtime_pool_header_init(header, RUNTIME_POOL_CLASS_INDEX_SYSTEM);
        out = (void*)((uint8_t*)header + RUNTIME_POOL_HEADER_SIZE);
        runtime_unlock_mutex(&runtime_pool_lock);
        return out;
    }

    class_size = ((size_t)1u) << ((size_t)class_index + RUNTIME_POOL_MIN_SHIFT);
    if (class_size <= RUNTIME_POOL_HEADER_SIZE) {
        runtime_unlock_mutex(&runtime_pool_lock);
        return NULL;
    }
    old_capacity = class_size - RUNTIME_POOL_HEADER_SIZE;
    if (size <= old_capacity) {
        runtime_unlock_mutex(&runtime_pool_lock);
        return ptr;
    }
    runtime_unlock_mutex(&runtime_pool_lock);
    out = runtime_pool_malloc(size);
    if (out == NULL) {
        return NULL;
    }
    kos_memcpy(out, ptr, old_capacity);
    runtime_pool_free(ptr);
    return out;
}

typedef struct {
    const char* str;
    intptr_t len;
} go_string;

typedef struct {
    unsigned char* values;
    intptr_t len;
    intptr_t cap;
} go_slice;

typedef struct {
    int32_t r;
    intptr_t pos;
} runtime_decoderune_result;

runtime_decoderune_result runtime_decoderune(go_string s, intptr_t k);

typedef struct {
    uint32_t wait;
    uint32_t notify;
    uintptr_t lock;
    void* head;
    void* tail;
} runtime_notify_list;

typedef struct {
    uintptr_t fn;
} runtime_func_val;

typedef struct go_type_descriptor {
    uintptr_t size;
    uintptr_t ptrdata;
    uint32_t hash;
    uint8_t tflag;
    uint8_t align;
    uint8_t field_align;
    uint8_t kind;
    bool (**equal)(const void* left, const void* right);
    const void* gcdata;
    const go_string* name;
    const void* uncommon;
    const void* ptr_to_this;
} go_type_descriptor;

typedef struct {
    const go_type_descriptor* type;
} go_interface_method_table;

typedef struct {
    const go_interface_method_table* methods;
    const void* data;
} go_interface;

typedef struct {
    const go_string* name;
    const go_string* package_path;
    const void* methods;
    uint32_t method_count;
    uint32_t exported_method_count;
} go_uncommon_type;

typedef struct {
    const go_type_descriptor common;
    const void* methods;
    uint32_t method_count;
    uint32_t exported_method_count;
} go_interface_type_descriptor;

typedef struct {
    const go_string* name;
    const go_string* package_path;
    const go_type_descriptor* type;
} go_interface_method_descriptor;

typedef struct {
    const go_string* name;
    const go_string* package_path;
    const go_type_descriptor* interface_type;
    const go_type_descriptor* concrete_type;
    void* function;
} go_named_type_method_descriptor;

typedef struct {
    go_interface value;
    bool ok;
} go_interface_assert_result;

typedef bool (*go_equal_function)(const void* left, const void* right);
typedef uint32_t (*go_hash_function)(const void* value);

typedef struct {
    go_type_descriptor common;
    const go_type_descriptor* key_type;
    const go_type_descriptor* value_type;
    const go_type_descriptor* bucket_type;
    void* hasher;
    uint8_t key_size;
    uint8_t value_size;
    uint8_t bucket_size;
    uint8_t flags;
    uint32_t extra;
} go_map_type_descriptor;

typedef struct {
    const go_type_descriptor common;
    const go_type_descriptor* elem_type;
    uintptr_t dir;
} go_chan_type_descriptor;

enum {
    RUNTIME_G_IDLE = 0,
    RUNTIME_G_RUNNABLE = 1,
    RUNTIME_G_RUNNING = 2,
    RUNTIME_G_WAITING = 3,
    RUNTIME_G_DEAD = 4,
};

#define RUNTIME_G_STACK_SIZE (256u * 1024u)

typedef struct runtime_hchan runtime_hchan;
typedef struct runtime_sudog runtime_sudog;

#if KOLIBRI_RT_DEBUG
static void runtime_debug_event(const char* tag, runtime_g* g, runtime_hchan* c, uint32_t extra);
#endif
typedef struct runtime_waitq runtime_waitq;
typedef struct runtime_scase runtime_scase;
typedef struct runtime_selectgo_result runtime_selectgo_result;
typedef struct runtime_selectnbrecv_result runtime_selectnbrecv_result;

struct runtime_waitq {
    runtime_sudog* first;
    runtime_sudog* last;
};

struct runtime_hchan {
    uint32_t qcount;
    uint32_t dataqsiz;
    void* buf;
    uint16_t elemsize;
    uint16_t pad;
    uint32_t closed;
    const go_type_descriptor* elemtype;
    uint32_t sendx;
    uint32_t recvx;
    runtime_waitq recvq;
    runtime_waitq sendq;
    runtime_mutex lock;
};

struct runtime_sudog {
    runtime_g* g;
    runtime_sudog* next;
    void* elem;
    runtime_hchan* c;
    int32_t select_index;
    uint8_t is_select;
    uint8_t success;
    uint8_t pad0;
    uint8_t pad1;
};

struct runtime_scase {
    runtime_hchan* c;
    void* elem;
};

struct runtime_selectgo_result {
    int32_t selected;
    int32_t recvOK;
};

struct runtime_selectnbrecv_result {
    uint8_t selected;
    uint8_t received;
};

typedef struct {
    void* value;
    uint32_t ok;
} go_mapaccess2_result;

typedef struct {
    void* key_data;
    void* value_data;
    uint32_t hash;
    uint8_t state;
    uint8_t padding[3];
} runtime_map_entry;

typedef struct {
    intptr_t len;
    runtime_map_entry* entries;
    intptr_t used;
    intptr_t cap;
    const go_map_type_descriptor* type;
    void* zero_value;
} runtime_map;

typedef void (*runtime_defer_fn)(void* arg);

static runtime_m runtime_m0;
static runtime_g runtime_g0;
static runtime_m* runtime_allm = NULL;
static runtime_g* runtime_allg = NULL;
static runtime_g* runtime_runq_head = NULL;
static runtime_g* runtime_runq_tail = NULL;
static runtime_g* runtime_deadg = NULL;
static uint8_t runtime_g_initialized = 0;
static runtime_mutex runtime_sched_lock;
static runtime_mutex runtime_m_lock;
static runtime_mutex runtime_gc_lock;

#define RUNTIME_MAX_THREAD_SLOTS 256u
static runtime_m* runtime_m_by_slot[RUNTIME_MAX_THREAD_SLOTS + 1];
static uint32_t runtime_m_count = 0;
static uint32_t runtime_m_pending = 0;
static uint32_t runtime_max_threads = 1;
static uint32_t runtime_started = 0;
static volatile uint32_t runtime_world_stopping = 0;
static volatile uint32_t runtime_world_waiting = 0;
static volatile uint32_t runtime_world_stopper_tid = 0;
static void runtime_debug_mark(const char* tag);
static void runtime_debug_eh_frame_summary(void);
static void runtime_gc_mark_pointer(const void* value);
typedef struct runtime_gc_header runtime_gc_header;
typedef void (*runtime_gc_scan_fn)(runtime_gc_header* header);
static void* runtime_gc_alloc_managed(size_t size, const go_type_descriptor* descriptor, runtime_gc_scan_fn scan, void* aux, uintptr_t count);
static void* runtime_gc_alloc_array(const go_type_descriptor* descriptor, intptr_t count, size_t total_size);
static void* runtime_gc_payload(runtime_gc_header* header);
static void runtime_fail_simple(const char* reason);
static void runtime_lock_mutex(runtime_mutex* m);
static void runtime_unlock_mutex(runtime_mutex* m);
static inline uint32_t runtime_atomic_load_u32(const volatile uint32_t* value);
static inline void runtime_atomic_store_u32(uint32_t* value, uint32_t next);
static inline bool runtime_atomic_cas_u32(uint32_t* value, uint32_t expected, uint32_t desired);
static inline uint32_t runtime_atomic_xadd_u32(volatile uint32_t* value, uint32_t delta);
static void runtime_yield(void);
static void runtime_sleep_ticks(uint32_t ticks);
static runtime_g* runtime_newg(void (*entry)(void*), void* arg);
static void runtime_schedule(void);
void runtime_panicmem(void);
void runtime_typedmemmove(const go_type_descriptor* descriptor, void* dest, const void* src);
void throw(go_string message);
size_t strlen(const char* str);
void abort(void);
void runtime_gc_set_stack_top(const void* ptr);

// Minimal DWARF EH decoding for _Unwind_Find_FDE.
#define DW_EH_PE_omit     0xff
#define DW_EH_PE_absptr   0x00
#define DW_EH_PE_uleb128  0x01
#define DW_EH_PE_udata2   0x02
#define DW_EH_PE_udata4   0x03
#define DW_EH_PE_udata8   0x04
#define DW_EH_PE_sleb128  0x09
#define DW_EH_PE_sdata2   0x0A
#define DW_EH_PE_sdata4   0x0B
#define DW_EH_PE_sdata8   0x0C
#define DW_EH_PE_pcrel    0x10
#define DW_EH_PE_textrel  0x20
#define DW_EH_PE_datarel  0x30
#define DW_EH_PE_aligned  0x50
#define DW_EH_PE_indirect 0x80

static uint32_t runtime_kolibri_current_thread_slot(void) {
    int32_t slot = runtime_kos_get_current_thread_slot_raw();
    if (slot > 0) {
        return (uint32_t)slot;
    }
    return 0;
}

static uintptr_t runtime_current_sp(void) {
    uintptr_t sp;
    __asm__ volatile("movl %%esp, %0" : "=r"(sp));
    return sp;
}

static runtime_g* runtime_find_g_by_stack(uintptr_t sp) {
    runtime_g* g = runtime_allg;

    while (g != NULL) {
        if (g->stack_base != NULL && g->stack_top != 0) {
            uintptr_t base = (uintptr_t)g->stack_base;
            uintptr_t top = (uintptr_t)g->stack_top;
            if (sp >= base && sp < top) {
                return g;
            }
        }
        g = g->all_next;
    }
    return NULL;
}

static runtime_m* runtime_getm_by_stack(void) {
    uintptr_t sp = runtime_current_sp();
    runtime_m* m = runtime_allm;

    while (m != NULL) {
        runtime_g* g0 = m->g0;
        if (g0 != NULL && g0->stack_base != NULL && g0->stack_top != 0) {
            uintptr_t base = (uintptr_t)g0->stack_base;
            uintptr_t top = (uintptr_t)g0->stack_top;
            if (sp >= base && sp < top) {
                return m;
            }
        }
        m = m->next;
    }
    return NULL;
}

static uint32_t runtime_kolibri_find_thread_slot_by_tid(uint32_t tid) {
    uint8_t buffer[1024];
    int32_t max_slot;

    if (tid == 0) {
        return 0;
    }
    max_slot = runtime_kos_get_thread_info_raw(buffer, -1);
    if (max_slot < 1) {
        return 0;
    }
    for (int32_t slot = 1; slot <= max_slot; slot++) {
        int32_t res = runtime_kos_get_thread_info_raw(buffer, slot);
        if (res < 0) {
            continue;
        }
        uint32_t id = *(uint32_t*)(buffer + 30);
        if (id == tid) {
            return (uint32_t)slot;
        }
    }
    return 0;
}

static void runtime_allm_add(runtime_m* m) {
    if (m == NULL) {
        return;
    }
    runtime_lock_mutex(&runtime_m_lock);
    m->next = runtime_allm;
    runtime_allm = m;
    runtime_m_count++;
    if (runtime_m_pending > 0) {
        runtime_m_pending--;
    }
    runtime_unlock_mutex(&runtime_m_lock);
}

static runtime_m* runtime_m_by_slot_load(uint32_t slot) {
    if (slot == 0 || slot > RUNTIME_MAX_THREAD_SLOTS) {
        return NULL;
    }
    return __atomic_load_n(&runtime_m_by_slot[slot], __ATOMIC_ACQUIRE);
}

static void runtime_m_by_slot_store(uint32_t slot, runtime_m* m) {
    if (slot == 0 || slot > RUNTIME_MAX_THREAD_SLOTS) {
        return;
    }
    __atomic_store_n(&runtime_m_by_slot[slot], m, __ATOMIC_RELEASE);
}

static runtime_m* runtime_getm(void) {
    runtime_m* m = NULL;
    uintptr_t sp = runtime_current_sp();

    if (runtime_g_initialized) {
        runtime_g* g = runtime_find_g_by_stack(sp);
        if (g != NULL && g->m != NULL) {
            return g->m;
        }
        runtime_m* by_stack = runtime_getm_by_stack();
        if (by_stack != NULL) {
            return by_stack;
        }
    }

    uint32_t slot = runtime_kolibri_current_thread_slot();
    if (slot != 0) {
        m = runtime_m_by_slot_load(slot);
        if (m != NULL) {
            return m;
        }
    }

    return &runtime_m0;
}

static void runtime_bind_m(runtime_m* m) {
    if (m == NULL) {
        return;
    }
    runtime_m_by_slot_store(m->tid, m);
}

static void runtime_stop_world(void) {
    runtime_m* m = runtime_getm();
    runtime_world_stopper_tid = m != NULL ? m->tid : 0;
    runtime_world_stopping = 1;
    for (;;) {
        uint32_t waiting = runtime_atomic_load_u32(&runtime_world_waiting);
        uint32_t count = runtime_m_count;
        if (count == 0 || waiting + 1 >= count) {
            break;
        }
        runtime_yield();
    }
}

static void runtime_start_world(void) {
    runtime_world_stopping = 0;
}

static void runtime_poll_world_stop(void) {
    runtime_m* m;

    if (!runtime_world_stopping) {
        return;
    }
    m = runtime_getm();
    if (m != NULL && m->tid == runtime_world_stopper_tid) {
        return;
    }
    if (m != NULL && m->curg != NULL) {
        uintptr_t marker;
        m->curg->context.esp = (uint32_t)(uintptr_t)&marker;
    }
    runtime_atomic_xadd_u32(&runtime_world_waiting, 1);
    while (runtime_world_stopping) {
        runtime_yield();
    }
    runtime_atomic_xadd_u32(&runtime_world_waiting, (uint32_t)-1);
}

static void runtime_set_current_g(runtime_g* g) {
    runtime_m* m = runtime_getm();
    if (m != NULL) {
        m->curg = g;
    }
}

static void runtime_init_g0(void) {
    runtime_m0.curg = &runtime_g0;
    runtime_m0.gsignal = NULL;
    runtime_m0.tid = runtime_kolibri_current_thread_slot();
    runtime_m0.g0 = &runtime_g0;
    runtime_m0.next = NULL;
    runtime_g0.m = &runtime_m0;
    runtime_g0.lockedm = &runtime_m0;
    runtime_g0.entrysp = 0;
    runtime_g0.status = RUNTIME_G_RUNNING;
    runtime_g0.stack_base = NULL;
    runtime_g0.stack_top = 0;
    runtime_g0.stack_size = 0;
    runtime_g0.sched_next = NULL;
    runtime_g0.all_next = NULL;
    runtime_g0.entry = NULL;
    runtime_g0.entry_arg = NULL;
    runtime_g0.select_done = -1;
    runtime_g0.select_recvok = 0;
    runtime_allg = &runtime_g0;
    runtime_allm_add(&runtime_m0);
    runtime_bind_m(&runtime_m0);
    runtime_set_current_g(&runtime_g0);
    runtime_g_initialized = 1;
}

runtime_g* runtime_getg(void) {
    if (!runtime_g_initialized) {
        runtime_init_g0();
    }
    runtime_g* g = runtime_find_g_by_stack(runtime_current_sp());
    if (g != NULL) {
        return g;
    }
    return runtime_getm()->curg;
}

extern void runtime_swapcontext(runtime_context* from, runtime_context* to);

static void runtime_runq_enqueue(runtime_g* g) {
    if (g == NULL) {
        return;
    }
    runtime_lock_mutex(&runtime_sched_lock);
    g->sched_next = NULL;
    if (runtime_runq_tail == NULL) {
        runtime_runq_head = g;
        runtime_runq_tail = g;
        runtime_unlock_mutex(&runtime_sched_lock);
        if (runtime_m_count > 1 && runtime_getm() == &runtime_m0) {
            runtime_yield();
        }
        return;
    }
    runtime_runq_tail->sched_next = g;
    runtime_runq_tail = g;
    runtime_unlock_mutex(&runtime_sched_lock);
    if (runtime_m_count > 1 && runtime_getm() == &runtime_m0) {
        runtime_yield();
    }
}

static runtime_g* runtime_runq_dequeue_for_m(runtime_m* m) {
    runtime_g* prev = NULL;
    runtime_g* g;

    runtime_lock_mutex(&runtime_sched_lock);
    g = runtime_runq_head;
    while (g != NULL) {
        if (g->lockedm == NULL || g->lockedm == m) {
            if (prev != NULL) {
                prev->sched_next = g->sched_next;
            } else {
                runtime_runq_head = g->sched_next;
            }
            if (runtime_runq_tail == g) {
                runtime_runq_tail = prev;
            }
            g->sched_next = NULL;
            runtime_unlock_mutex(&runtime_sched_lock);
            return g;
        }
        prev = g;
        g = g->sched_next;
    }
    runtime_unlock_mutex(&runtime_sched_lock);
    return NULL;
}

static void runtime_allg_add(runtime_g* g) {
    if (g == NULL) {
        return;
    }
    runtime_lock_mutex(&runtime_sched_lock);
    g->all_next = runtime_allg;
    runtime_allg = g;
    runtime_unlock_mutex(&runtime_sched_lock);
}

static void runtime_allg_remove(runtime_g* g) {
    runtime_g* prev;
    runtime_g* cur;

    if (g == NULL) {
        return;
    }
    runtime_lock_mutex(&runtime_sched_lock);
    prev = NULL;
    cur = runtime_allg;
    while (cur != NULL) {
        if (cur == g) {
            if (prev != NULL) {
                prev->all_next = cur->all_next;
            } else {
                runtime_allg = cur->all_next;
            }
            cur->all_next = NULL;
            runtime_unlock_mutex(&runtime_sched_lock);
            return;
        }
        prev = cur;
        cur = cur->all_next;
    }
    runtime_unlock_mutex(&runtime_sched_lock);
}

static void runtime_enqueue_dead(runtime_g* g) {
    if (g == NULL) {
        return;
    }
    runtime_lock_mutex(&runtime_sched_lock);
    g->sched_next = runtime_deadg;
    runtime_deadg = g;
    runtime_unlock_mutex(&runtime_sched_lock);
}

static void runtime_free_dead(void) {
    for (;;) {
        runtime_g* g;

        runtime_lock_mutex(&runtime_sched_lock);
        g = runtime_deadg;
        if (g == NULL) {
            runtime_unlock_mutex(&runtime_sched_lock);
            return;
        }
        runtime_deadg = g->sched_next;
        g->sched_next = NULL;
        runtime_unlock_mutex(&runtime_sched_lock);

        runtime_allg_remove(g);
        if (g->stack_base != NULL) {
            free(g->stack_base);
        }
        free(g);
    }
}

static void runtime_makecontext(runtime_context* ctx, void (*fn)(void), void* stack, size_t size) {
    uintptr_t sp;

    if (ctx == NULL) {
        return;
    }
    ctx->ebx = 0;
    ctx->esi = 0;
    ctx->edi = 0;
    ctx->ebp = 0;
    ctx->esp = 0;
    ctx->eip = 0;

    if (fn == NULL || stack == NULL || size < 16u) {
        return;
    }

    sp = (uintptr_t)stack + size;
    sp &= ~(uintptr_t)0xFu;
    sp -= sizeof(uintptr_t);
    *(uintptr_t*)sp = 0;

    ctx->esp = (uint32_t)sp;
    ctx->eip = (uint32_t)(uintptr_t)fn;
}

static void runtime_switch(runtime_g* from, runtime_g* to) {
    runtime_m* m;
    if (from == NULL || to == NULL || from == to) {
        return;
    }
    m = NULL;
    if (from->m != NULL) {
        m = from->m;
    } else if (to->m != NULL) {
        m = to->m;
    } else {
        m = runtime_getm();
    }
    to->m = m;
    runtime_set_current_g(to);
    runtime_swapcontext(&from->context, &to->context);
    runtime_set_current_g(from);
}

static void runtime_ready(runtime_g* g) {
    if (g == NULL) {
        return;
    }
    if (g->status == RUNTIME_G_DEAD) {
        return;
    }
    uint32_t parking = runtime_atomic_load_u32(&g->parking);
    if (parking == 1u) {
        runtime_atomic_store_u32(&g->parking, 0);
        g->status = RUNTIME_G_RUNNING;
        return;
    }
    runtime_atomic_store_u32(&g->parking, 0);
    if (g->status == RUNTIME_G_WAITING) {
        g->status = RUNTIME_G_RUNNABLE;
        runtime_runq_enqueue(g);
    }
}

static void runtime_gopark_after_unlock(runtime_g* g, runtime_m* m);

static void runtime_gopark_internal(void) {
    runtime_g* g = runtime_getg();
    runtime_m* m = runtime_getm();
    if (g == NULL || m == NULL || g == m->g0) {
        return;
    }
    g->status = RUNTIME_G_WAITING;
    runtime_atomic_store_u32(&g->parking, 1);
    runtime_gopark_after_unlock(g, m);
}

static void runtime_gopark_after_unlock(runtime_g* g, runtime_m* m) {
    if (g == NULL || m == NULL || g == m->g0) {
        return;
    }
    if (!runtime_atomic_cas_u32(&g->parking, 1, 2)) {
#if KOLIBRI_RT_DEBUG
        runtime_debug_event("park skip", g, NULL, g->parking);
#endif
        g->status = RUNTIME_G_RUNNING;
        return;
    }
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("park sleep", g, NULL, g->parking);
#endif
    runtime_switch(g, m->g0);
}

static void runtime_gosched_internal(void) {
    runtime_g* g = runtime_getg();
    runtime_m* m = runtime_getm();
    if (g == NULL || m == NULL || g == m->g0) {
        return;
    }
    g->status = RUNTIME_G_RUNNABLE;
    runtime_runq_enqueue(g);
    runtime_switch(g, m->g0);
}

void runtime_Gosched(void) __asm__("runtime.Gosched");
void runtime_Gosched(void) {
    runtime_gosched_internal();
}

void runtime_LockOSThread(void) __asm__("runtime.LockOSThread");
void runtime_LockOSThread(void) {
    runtime_g* g = runtime_getg();
    runtime_m* m = runtime_getm();
    if (g == NULL || m == NULL) {
        return;
    }
    g->lockedm = m;
}

void runtime_UnlockOSThread(void) __asm__("runtime.UnlockOSThread");
void runtime_UnlockOSThread(void) {
    runtime_g* g = runtime_getg();
    if (g == NULL) {
        return;
    }
    g->lockedm = NULL;
}

__attribute__((noreturn)) static void runtime_goexit_internal(void) {
    runtime_g* g = runtime_getg();
    runtime_m* m = runtime_getm();
    if (g == NULL || m == NULL || g == m->g0) {
        runtime_fail_simple("goexit on g0");
    }
    g->status = RUNTIME_G_DEAD;
    runtime_enqueue_dead(g);
    runtime_switch(g, m->g0);
    for (;;) {
    }
}

static void runtime_go_start(void) {
    runtime_g* g = runtime_getg();
    if (g != NULL && g->entry != NULL) {
        g->entry(g->entry_arg);
    }
    runtime_goexit_internal();
}

static void (*runtime_app_init_fn)(void) = NULL;
static void (*runtime_app_main_fn)(void) = NULL;

static void runtime_app_entry(void* arg) {
    (void)arg;
    if (runtime_app_init_fn != NULL) {
        runtime_app_init_fn();
    }
    if (runtime_app_main_fn != NULL) {
        runtime_app_main_fn();
    }
}

typedef struct {
    runtime_m* m;
    runtime_g* g0;
    void* stack_base;
    uintptr_t stack_top;
    uint32_t stack_size;
} runtime_m_start_record;

static uint32_t runtime_thread_stack_pointer(void* stack_base, size_t size, void* arg) {
    uintptr_t top;

    if (stack_base == NULL || size < 16u) {
        return 0;
    }

    top = (uintptr_t)stack_base + size;
    top &= ~(uintptr_t)0xFu;
    if (top < (uintptr_t)stack_base + 4u) {
        return 0;
    }
    top -= 4u;
    *(uint32_t*)top = (uint32_t)(uintptr_t)arg;
    return (uint32_t)top;
}

void runtime_m_start(runtime_m_start_record* start) {
    runtime_m* m;
    runtime_g* g0;

    if (start == NULL) {
        return;
    }
    m = start->m;
    g0 = start->g0;
    if (m == NULL || g0 == NULL) {
        return;
    }

    m->curg = g0;
    m->g0 = g0;
    m->gsignal = NULL;
    {
        uint32_t slot = runtime_kolibri_current_thread_slot();
        if (slot == 0) {
            slot = runtime_kolibri_find_thread_slot_by_tid(m->tid);
        }
        if (slot != 0) {
            m->tid = slot;
        }
    }

    g0->m = m;
    g0->lockedm = m;
    g0->entrysp = 0;
    g0->status = RUNTIME_G_RUNNING;
    g0->stack_base = start->stack_base;
    g0->stack_top = start->stack_top;
    g0->stack_size = start->stack_size;
    g0->sched_next = NULL;
    g0->all_next = NULL;
    g0->entry = NULL;
    g0->entry_arg = NULL;
    g0->select_done = -1;
    g0->select_recvok = 0;

    runtime_allg_add(g0);
    runtime_allm_add(m);
    runtime_bind_m(m);
    runtime_set_current_g(g0);

    free(start);
    runtime_schedule();
    runtime_kos_exit_raw();
}

static int runtime_spawn_m(void) {
    runtime_m* m;
    runtime_g* g0;
    runtime_m_start_record* start;
    void* stack;
    uint32_t stack_ptr;
    uint32_t stack_size = 0x10000u;
    int32_t raw_id;

    runtime_lock_mutex(&runtime_m_lock);
    runtime_m_pending++;
    runtime_unlock_mutex(&runtime_m_lock);

    m = (runtime_m*)malloc(sizeof(runtime_m));
    if (m == NULL) {
        runtime_lock_mutex(&runtime_m_lock);
        if (runtime_m_pending > 0) {
            runtime_m_pending--;
        }
        runtime_unlock_mutex(&runtime_m_lock);
        return 0;
    }
    memset(m, 0, sizeof(*m));

    g0 = (runtime_g*)malloc(sizeof(runtime_g));
    if (g0 == NULL) {
        free(m);
        runtime_lock_mutex(&runtime_m_lock);
        if (runtime_m_pending > 0) {
            runtime_m_pending--;
        }
        runtime_unlock_mutex(&runtime_m_lock);
        return 0;
    }
    memset(g0, 0, sizeof(*g0));

    stack = malloc(stack_size);
    if (stack == NULL) {
        free(g0);
        free(m);
        runtime_lock_mutex(&runtime_m_lock);
        if (runtime_m_pending > 0) {
            runtime_m_pending--;
        }
        runtime_unlock_mutex(&runtime_m_lock);
        return 0;
    }

    start = (runtime_m_start_record*)malloc(sizeof(runtime_m_start_record));
    if (start == NULL) {
        free(stack);
        free(g0);
        free(m);
        runtime_lock_mutex(&runtime_m_lock);
        if (runtime_m_pending > 0) {
            runtime_m_pending--;
        }
        runtime_unlock_mutex(&runtime_m_lock);
        return 0;
    }

    start->m = m;
    start->g0 = g0;
    start->stack_base = stack;
    start->stack_top = (uintptr_t)stack + stack_size;
    start->stack_size = stack_size;

    stack_ptr = runtime_thread_stack_pointer(stack, stack_size, start);
    if (stack_ptr == 0) {
        free(start);
        free(stack);
        free(g0);
        free(m);
        runtime_lock_mutex(&runtime_m_lock);
        if (runtime_m_pending > 0) {
            runtime_m_pending--;
        }
        runtime_unlock_mutex(&runtime_m_lock);
        return 0;
    }

    raw_id = runtime_kos_create_thread_raw((uint32_t)(uintptr_t)&runtime_kolibri_thread_entry, stack_ptr);
    if (raw_id < 0) {
        free(start);
        free(stack);
        free(g0);
        free(m);
        runtime_lock_mutex(&runtime_m_lock);
        if (runtime_m_pending > 0) {
            runtime_m_pending--;
        }
        runtime_unlock_mutex(&runtime_m_lock);
        return 0;
    }

    m->tid = (uint32_t)raw_id;
    return 1;
}

extern char __end;
extern char __memory_top;

void runtime_kolibri_start(void (*init)(void), void (*main)(void)) {
    if (!runtime_g_initialized) {
        runtime_init_g0();
    }
    runtime_gc_set_stack_top(&__memory_top);
    runtime_started = 1;
    runtime_app_init_fn = init;
    runtime_app_main_fn = main;
    runtime_g* g = runtime_newg(runtime_app_entry, NULL);
    runtime_runq_enqueue(g);
    if (runtime_max_threads > 1) {
        uint32_t target = runtime_max_threads - 1;
        while (target > 0) {
            if (!runtime_spawn_m()) {
                break;
            }
            target--;
        }
    }
    runtime_schedule();
    runtime_console_bridge_close(1);
    runtime_exit_process();
}

static runtime_g* runtime_newg(void (*entry)(void*), void* arg) {
    runtime_g* g;
    void* stack;

    g = (runtime_g*)malloc(sizeof(runtime_g));
    if (g == NULL) {
        runtime_panicmem();
    }
    memset(g, 0, sizeof(*g));
    g->m = runtime_getm();
    g->entry = entry;
    g->entry_arg = arg;
    g->status = RUNTIME_G_RUNNABLE;
    g->select_done = -1;
    g->select_recvok = 0;
    g->stack_size = RUNTIME_G_STACK_SIZE;
    stack = malloc(g->stack_size);
    if (stack == NULL) {
        free(g);
        runtime_panicmem();
    }
    g->stack_base = stack;
    g->stack_top = (uintptr_t)stack + g->stack_size;
    runtime_makecontext(&g->context, runtime_go_start, stack, g->stack_size);
    runtime_allg_add(g);
    return g;
}

runtime_g* __go_go(uintptr_t fn, void* arg) {
    runtime_g* g = runtime_newg((void (*)(void*))(uintptr_t)fn, arg);
    runtime_runq_enqueue(g);
    return g;
}

static void runtime_schedule(void) {
    runtime_m* m = runtime_getm();
    runtime_g* g0 = (m != NULL && m->g0 != NULL) ? m->g0 : &runtime_g0;
    for (;;) {
        runtime_free_dead();
        runtime_poll_world_stop();
        runtime_g* next = runtime_runq_dequeue_for_m(m);
        if (next == NULL) {
            if (m != &runtime_m0) {
                runtime_yield();
                continue;
            }
        runtime_lock_mutex(&runtime_sched_lock);
        runtime_g* scan = runtime_allg;
        bool any_runnable = false;
        bool any_running = false;
        bool any_waiting = false;
            while (scan != NULL) {
                if (scan->m != NULL && scan->m->g0 == scan) {
                    scan = scan->all_next;
                    continue;
                }
                if (scan->status == RUNTIME_G_RUNNABLE) {
                    any_runnable = true;
                } else if (scan->status == RUNTIME_G_RUNNING) {
                    any_running = true;
                } else if (scan->status == RUNTIME_G_WAITING) {
                    any_waiting = true;
                }
                scan = scan->all_next;
            }
        runtime_unlock_mutex(&runtime_sched_lock);
        if (any_runnable || any_running) {
            if (any_runnable) {
                runtime_yield();
            } else {
                runtime_sleep_ticks(1);
            }
            continue;
        }
        if (any_waiting) {
#if KOLIBRI_RT_DEBUG
            runtime_debug_event("deadlock", runtime_getg(), NULL, runtime_getg() != NULL ? runtime_getg()->parking : 0);
#endif
                runtime_fail_simple("all goroutines asleep - deadlock");
            }
            return;
        }
        next->status = RUNTIME_G_RUNNING;
        runtime_switch(g0, next);
    }
}

void runtime_block(void) __asm__("runtime.block");
void runtime_block(void) {
    for (;;) {
        runtime_gopark_internal();
    }
}

void runtime_panicgonil(void) __asm__("runtime.panicgonil");
void runtime_panicgonil(void) {
    throw((go_string){ "go of nil func value", 20 });
}

static runtime_sudog* runtime_sudog_alloc(runtime_hchan* c, void* elem, int32_t index, uint8_t is_select) {
    runtime_sudog* sd = (runtime_sudog*)malloc(sizeof(runtime_sudog));
    if (sd == NULL) {
        runtime_panicmem();
    }
    memset(sd, 0, sizeof(*sd));
    sd->g = runtime_getg();
    sd->elem = elem;
    sd->c = c;
    sd->select_index = index;
    sd->is_select = is_select;
    sd->success = 0;
    return sd;
}

static void runtime_sudog_free(runtime_sudog* sd) {
    if (sd == NULL) {
        return;
    }
    free(sd);
}

static void runtime_waitq_enqueue(runtime_waitq* q, runtime_sudog* sd) {
    if (q == NULL || sd == NULL) {
        return;
    }
    sd->next = NULL;
    if (q->last == NULL) {
        q->first = sd;
        q->last = sd;
        return;
    }
    q->last->next = sd;
    q->last = sd;
}

static runtime_sudog* runtime_waitq_dequeue(runtime_waitq* q) {
    runtime_sudog* sd;
    if (q == NULL) {
        return NULL;
    }
    for (;;) {
        sd = q->first;
        if (sd == NULL) {
            return NULL;
        }
        if (sd->is_select && sd->g != NULL && sd->g->select_done != -1) {
            q->first = sd->next;
            if (q->first == NULL) {
                q->last = NULL;
            }
            sd->next = NULL;
            continue;
        }
        q->first = sd->next;
        if (q->first == NULL) {
            q->last = NULL;
        }
        sd->next = NULL;
        return sd;
    }
}

static void runtime_waitq_remove(runtime_waitq* q, runtime_sudog* sd) {
    runtime_sudog* prev;
    runtime_sudog* cur;

    if (q == NULL || sd == NULL) {
        return;
    }
    prev = NULL;
    cur = q->first;
    while (cur != NULL) {
        if (cur == sd) {
            if (prev != NULL) {
                prev->next = cur->next;
            } else {
                q->first = cur->next;
            }
            if (q->last == cur) {
                q->last = prev;
            }
            cur->next = NULL;
            return;
        }
        prev = cur;
        cur = cur->next;
    }
}

static void runtime_wake_sudog(runtime_sudog* sd, int recvok) {
    runtime_g* g;
    runtime_hchan* c;
    uint32_t success;
    if (sd == NULL) {
        return;
    }
    g = sd->g;
    c = sd->c;
    success = recvok ? 1u : 0u;
    sd->success = success;
    if (sd->is_select) {
        if (g != NULL && g->select_done != -1) {
            return;
        }
        if (g != NULL) {
            g->select_done = sd->select_index;
            g->select_recvok = recvok;
        }
    }
    runtime_ready(g);
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("wake", g, c, success);
#endif
}

static void runtime_chan_lock(runtime_hchan* c) {
    if (c == NULL) {
        return;
    }
    runtime_lock_mutex(&c->lock);
}

static void runtime_chan_unlock(runtime_hchan* c) {
    if (c == NULL) {
        return;
    }
    runtime_unlock_mutex(&c->lock);
}

static void runtime_chan_zero(runtime_hchan* c, void* elem) {
    if (c == NULL || elem == NULL || c->elemsize == 0) {
        return;
    }
    memset(elem, 0, (size_t)c->elemsize);
}

static void runtime_chan_copy(runtime_hchan* c, void* dst, void* src) {
    if (c == NULL || dst == NULL || src == NULL || c->elemsize == 0) {
        return;
    }
    if (c->elemtype != NULL) {
        runtime_typedmemmove(c->elemtype, dst, src);
    } else {
        memmove(dst, src, (size_t)c->elemsize);
    }
}

static bool runtime_chan_send(runtime_hchan* c, void* elem, bool block) {
    runtime_sudog* sd;
    runtime_g* g;
    runtime_m* m;

    if (c == NULL) {
        if (!block) {
            return false;
        }
        runtime_gopark_internal();
        return false;
    }

    runtime_chan_lock(c);

    if (c->closed) {
        runtime_chan_unlock(c);
        throw((go_string){ "send on closed channel", 22 });
    }

    sd = runtime_waitq_dequeue(&c->recvq);
    if (sd != NULL) {
        runtime_chan_copy(c, sd->elem, elem);
        runtime_wake_sudog(sd, 1);
        runtime_chan_unlock(c);
        return true;
    }

    if (c->dataqsiz != 0 && c->qcount < c->dataqsiz) {
        void* slot = (unsigned char*)c->buf + (uintptr_t)c->sendx * c->elemsize;
        runtime_chan_copy(c, slot, elem);
        c->sendx = (c->sendx + 1) % c->dataqsiz;
        c->qcount++;
        runtime_chan_unlock(c);
        return true;
    }

    if (!block) {
        runtime_chan_unlock(c);
        return false;
    }

    sd = runtime_sudog_alloc(c, elem, -1, 0);
    runtime_waitq_enqueue(&c->sendq, sd);
    g = runtime_getg();
    m = runtime_getm();
    if (g != NULL) {
        g->status = RUNTIME_G_WAITING;
        runtime_atomic_store_u32(&g->parking, 1);
    }
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("send park", g, c, g != NULL ? g->parking : 0);
#endif
    runtime_chan_unlock(c);
    runtime_gopark_after_unlock(g, m);

    if (!sd->success) {
        throw((go_string){ "send on closed channel", 22 });
    }
    runtime_sudog_free(sd);
    return true;
}

static bool runtime_chan_recv(runtime_hchan* c, void* elem, bool block, bool* recvok) {
    runtime_sudog* sd;
    runtime_g* g;
    runtime_m* m;

    if (recvok != NULL) {
        *recvok = false;
    }

    if (c == NULL) {
        if (!block) {
            return false;
        }
        runtime_gopark_internal();
        return false;
    }

    runtime_chan_lock(c);

    if (c->qcount > 0) {
        void* slot = (unsigned char*)c->buf + (uintptr_t)c->recvx * c->elemsize;
        if (elem != NULL) {
            runtime_chan_copy(c, elem, slot);
        }
        c->recvx = (c->recvx + 1) % c->dataqsiz;
        c->qcount--;

        sd = runtime_waitq_dequeue(&c->sendq);
        if (sd != NULL) {
            void* send_slot = (unsigned char*)c->buf + (uintptr_t)c->sendx * c->elemsize;
            runtime_chan_copy(c, send_slot, sd->elem);
            c->sendx = (c->sendx + 1) % c->dataqsiz;
            c->qcount++;
            runtime_wake_sudog(sd, 1);
        }

        runtime_chan_unlock(c);
        if (recvok != NULL) {
            *recvok = true;
        }
        return true;
    }

    sd = runtime_waitq_dequeue(&c->sendq);
    if (sd != NULL) {
        if (elem != NULL) {
            runtime_chan_copy(c, elem, sd->elem);
        }
        runtime_wake_sudog(sd, 1);
        runtime_chan_unlock(c);
        if (recvok != NULL) {
            *recvok = true;
        }
        return true;
    }

    if (c->closed) {
        runtime_chan_unlock(c);
        if (elem != NULL) {
            runtime_chan_zero(c, elem);
        }
        if (recvok != NULL) {
            *recvok = false;
        }
        return true;
    }

    if (!block) {
        runtime_chan_unlock(c);
        return false;
    }

    sd = runtime_sudog_alloc(c, elem, -1, 0);
    runtime_waitq_enqueue(&c->recvq, sd);
    g = runtime_getg();
    m = runtime_getm();
    if (g != NULL) {
        g->status = RUNTIME_G_WAITING;
        runtime_atomic_store_u32(&g->parking, 1);
    }
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("recv park", g, c, g != NULL ? g->parking : 0);
#endif
    runtime_chan_unlock(c);
    runtime_gopark_after_unlock(g, m);

    if (!sd->success) {
        if (elem != NULL) {
            runtime_chan_zero(c, elem);
        }
        if (recvok != NULL) {
            *recvok = false;
        }
    } else if (recvok != NULL) {
        *recvok = true;
    }
    runtime_sudog_free(sd);
    return true;
}

static bool runtime_select_try_send_locked(runtime_hchan* c, void* elem) {
    runtime_sudog* sd;

    if (c == NULL) {
        return false;
    }
    if (c->closed) {
        throw((go_string){ "send on closed channel", 22 });
    }
    sd = runtime_waitq_dequeue(&c->recvq);
    if (sd != NULL) {
        runtime_chan_copy(c, sd->elem, elem);
        runtime_wake_sudog(sd, 1);
        return true;
    }
    if (c->dataqsiz != 0 && c->qcount < c->dataqsiz) {
        void* slot = (unsigned char*)c->buf + (uintptr_t)c->sendx * c->elemsize;
        runtime_chan_copy(c, slot, elem);
        c->sendx = (c->sendx + 1) % c->dataqsiz;
        c->qcount++;
        return true;
    }
    return false;
}

static bool runtime_select_try_recv_locked(runtime_hchan* c, void* elem, bool* recvok) {
    runtime_sudog* sd;

    if (recvok != NULL) {
        *recvok = false;
    }
    if (c == NULL) {
        return false;
    }
    if (c->qcount > 0) {
        void* slot = (unsigned char*)c->buf + (uintptr_t)c->recvx * c->elemsize;
        if (elem != NULL) {
            runtime_chan_copy(c, elem, slot);
        }
        c->recvx = (c->recvx + 1) % c->dataqsiz;
        c->qcount--;

        sd = runtime_waitq_dequeue(&c->sendq);
        if (sd != NULL) {
            void* send_slot = (unsigned char*)c->buf + (uintptr_t)c->sendx * c->elemsize;
            runtime_chan_copy(c, send_slot, sd->elem);
            c->sendx = (c->sendx + 1) % c->dataqsiz;
            c->qcount++;
            runtime_wake_sudog(sd, 1);
        }

        if (recvok != NULL) {
            *recvok = true;
        }
        return true;
    }
    sd = runtime_waitq_dequeue(&c->sendq);
    if (sd != NULL) {
        if (elem != NULL) {
            runtime_chan_copy(c, elem, sd->elem);
        }
        runtime_wake_sudog(sd, 1);
        if (recvok != NULL) {
            *recvok = true;
        }
        return true;
    }
    if (c->closed) {
        if (elem != NULL) {
            runtime_chan_zero(c, elem);
        }
        if (recvok != NULL) {
            *recvok = false;
        }
        return true;
    }
    return false;
}

static int32_t runtime_select_build_lock_order(runtime_scase* cas0, int32_t ncases, runtime_hchan*** order_out) {
    runtime_hchan** order;
    int32_t count = 0;
    int32_t i;
    int32_t j;

    if (order_out == NULL || cas0 == NULL || ncases <= 0) {
        return 0;
    }

    order = (runtime_hchan**)malloc((size_t)ncases * sizeof(runtime_hchan*));
    if (order == NULL) {
        runtime_panicmem();
    }

    for (i = 0; i < ncases; i++) {
        runtime_hchan* c = cas0[i].c;
        if (c == NULL) {
            continue;
        }
        for (j = 0; j < count; j++) {
            if (order[j] == c) {
                break;
            }
        }
        if (j == count) {
            order[count++] = c;
        }
    }

    for (i = 0; i < count; i++) {
        for (j = i + 1; j < count; j++) {
            if ((uintptr_t)order[j] < (uintptr_t)order[i]) {
                runtime_hchan* tmp = order[i];
                order[i] = order[j];
                order[j] = tmp;
            }
        }
    }

    *order_out = order;
    return count;
}

static void runtime_select_lock_all(runtime_hchan** order, int32_t count) {
    int32_t i;
    if (order == NULL || count <= 0) {
        return;
    }
    for (i = 0; i < count; i++) {
        runtime_chan_lock(order[i]);
    }
}

static void runtime_select_unlock_all(runtime_hchan** order, int32_t count) {
    int32_t i;
    if (order == NULL || count <= 0) {
        return;
    }
    for (i = count - 1; i >= 0; i--) {
        runtime_chan_unlock(order[i]);
    }
}

static void runtime_gc_scan_hchan(runtime_gc_header* header) {
    runtime_hchan* c;

    if (header == NULL) {
        return;
    }
    c = (runtime_hchan*)runtime_gc_payload(header);
    if (c == NULL) {
        return;
    }
    runtime_gc_mark_pointer(c->buf);
}

runtime_hchan* runtime_makechan(go_chan_type_descriptor* t, int32_t size) __asm__("runtime.makechan");
runtime_hchan* runtime_makechan(go_chan_type_descriptor* t, int32_t size) {
    runtime_hchan* c;
    size_t mem;

    if (t == NULL) {
        runtime_panicmem();
    }
    if (size < 0) {
        runtime_panicmem();
    }
    if (t->elem_type == NULL || t->elem_type->size >= (uintptr_t)(1u << 16)) {
        runtime_panicmem();
    }

    mem = 0;
    if ((size_t)size > ((size_t)-1) / (size_t)t->elem_type->size) {
        runtime_panicmem();
    }
    mem = (size_t)size * (size_t)t->elem_type->size;

    c = (runtime_hchan*)runtime_gc_alloc_managed(sizeof(runtime_hchan), NULL, runtime_gc_scan_hchan, NULL, 0);
    if (c == NULL) {
        runtime_panicmem();
    }
    memset(c, 0, sizeof(*c));
    c->dataqsiz = (uint32_t)size;
    c->elemsize = (uint16_t)(t->elem_type != NULL ? t->elem_type->size : 0);
    c->elemtype = t->elem_type;

    if (size > 0 && mem > 0) {
        c->buf = runtime_gc_alloc_array(t->elem_type, size, mem);
    } else {
        c->buf = NULL;
    }

    return c;
}

runtime_hchan* runtime_makechan64(go_chan_type_descriptor* t, int64_t size) __asm__("runtime.makechan64");
runtime_hchan* runtime_makechan64(go_chan_type_descriptor* t, int64_t size) {
    if (size > INT32_MAX || size < 0) {
        runtime_panicmem();
    }
    return runtime_makechan(t, (int32_t)size);
}

void runtime_chansend1(runtime_hchan* c, void* elem) __asm__("runtime.chansend1");
void runtime_chansend1(runtime_hchan* c, void* elem) {
    runtime_chan_send(c, elem, true);
}

void runtime_chanrecv1(runtime_hchan* c, void* elem) __asm__("runtime.chanrecv1");
void runtime_chanrecv1(runtime_hchan* c, void* elem) {
    bool ok;
    runtime_chan_recv(c, elem, true, &ok);
}

uint8_t runtime_chanrecv2(runtime_hchan* c, void* elem) __asm__("runtime.chanrecv2");
uint8_t runtime_chanrecv2(runtime_hchan* c, void* elem) {
    bool ok;
    runtime_chan_recv(c, elem, true, &ok);
    return ok ? 1 : 0;
}

uint8_t runtime_selectnbsend(runtime_hchan* c, void* elem) __asm__("runtime.selectnbsend");
uint8_t runtime_selectnbsend(runtime_hchan* c, void* elem) {
    return runtime_chan_send(c, elem, false) ? 1 : 0;
}

void runtime_selectnbrecv(runtime_selectnbrecv_result* ret, void* elem, runtime_hchan* c) __asm__("runtime.selectnbrecv");
void runtime_selectnbrecv(runtime_selectnbrecv_result* ret, void* elem, runtime_hchan* c) {
    bool ok;
    bool done = runtime_chan_recv(c, elem, false, &ok);
    if (ret == NULL) {
        return;
    }
    ret->selected = done ? 1 : 0;
    ret->received = ok ? 1 : 0;
}

void runtime_closechan(runtime_hchan* c) __asm__("runtime.closechan");
void runtime_closechan(runtime_hchan* c) {
    runtime_sudog* sd;
    if (c == NULL) {
        return;
    }

    runtime_chan_lock(c);
    if (c->closed) {
        runtime_chan_unlock(c);
        throw((go_string){ "close of closed channel", 23 });
    }
    c->closed = 1;
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("close", runtime_getg(), c, 0);
#endif

    while ((sd = runtime_waitq_dequeue(&c->recvq)) != NULL) {
        runtime_wake_sudog(sd, 0);
    }
    while ((sd = runtime_waitq_dequeue(&c->sendq)) != NULL) {
        runtime_wake_sudog(sd, 0);
    }

    runtime_chan_unlock(c);
}

void runtime_selectgo(runtime_selectgo_result* ret, runtime_scase* cas0, uint16_t* order0, int32_t nsends, int32_t nrecvs, bool block) __asm__("runtime.selectgo");
void runtime_selectgo(runtime_selectgo_result* ret, runtime_scase* cas0, uint16_t* order0, int32_t nsends, int32_t nrecvs, bool block) {
    runtime_g* g;
    runtime_m* m;
    int32_t ncases;
    int32_t i;
    int32_t selected = -1;
    int32_t recvok = 0;
    runtime_sudog** sudogs;
    runtime_hchan** lock_order = NULL;
    int32_t lock_count = 0;

    (void)order0;

    if (ret == NULL) {
        return;
    }
    ret->selected = -1;
    ret->recvOK = 0;

    if (cas0 == NULL) {
        if (block) {
            runtime_block();
        }
        return;
    }

    ncases = nsends + nrecvs;
    if (ncases == 0) {
        if (block) {
            runtime_block();
        }
        return;
    }
    if (!block) {
        g = runtime_getg();
        lock_count = runtime_select_build_lock_order(cas0, ncases, &lock_order);
        runtime_select_lock_all(lock_order, lock_count);
        for (i = 0; i < ncases; i++) {
            runtime_scase* sc = &cas0[i];
            if (sc->c == NULL) {
                continue;
            }
            if (i < nsends) {
                if (runtime_select_try_send_locked(sc->c, sc->elem)) {
                    selected = i;
                    recvok = 0;
#if KOLIBRI_RT_DEBUG
                    runtime_debug_event("select nb", g, sc->c, (uint32_t)i);
#endif
                    goto done_unlock;
                }
            } else {
                bool ok;
                if (runtime_select_try_recv_locked(sc->c, sc->elem, &ok)) {
                    selected = i;
                    recvok = ok ? 1 : 0;
#if KOLIBRI_RT_DEBUG
                    runtime_debug_event("select nb", g, sc->c, (uint32_t)i);
#endif
                    goto done_unlock;
                }
            }
        }
done_unlock:
        runtime_select_unlock_all(lock_order, lock_count);
        free(lock_order);
        ret->selected = selected;
        ret->recvOK = recvok;
        return;
    }

retry_block:
    selected = -1;
    recvok = 0;
    g = runtime_getg();
    if (g == NULL) {
        return;
    }
    g->select_done = -1;
    g->select_recvok = 0;
    m = runtime_getm();

    sudogs = (runtime_sudog**)malloc((size_t)ncases * sizeof(runtime_sudog*));
    if (sudogs == NULL) {
        runtime_panicmem();
    }
    for (i = 0; i < ncases; i++) {
        sudogs[i] = NULL;
    }

    lock_count = runtime_select_build_lock_order(cas0, ncases, &lock_order);
    runtime_select_lock_all(lock_order, lock_count);

    for (i = 0; i < ncases; i++) {
        runtime_scase* sc = &cas0[i];
        if (sc->c == NULL) {
            continue;
        }
        if (i < nsends) {
            if (runtime_select_try_send_locked(sc->c, sc->elem)) {
                selected = i;
                recvok = 0;
#if KOLIBRI_RT_DEBUG
                runtime_debug_event("select imm", g, sc->c, (uint32_t)i);
#endif
                goto selected_unlock;
            }
        } else {
            bool ok;
            if (runtime_select_try_recv_locked(sc->c, sc->elem, &ok)) {
                selected = i;
                recvok = ok ? 1 : 0;
#if KOLIBRI_RT_DEBUG
                runtime_debug_event("select imm", g, sc->c, (uint32_t)i);
#endif
                goto selected_unlock;
            }
        }
    }

    for (i = 0; i < ncases; i++) {
        runtime_scase* sc = &cas0[i];
        if (sc->c == NULL) {
            continue;
        }
        runtime_sudog* sd = runtime_sudog_alloc(sc->c, sc->elem, i, 1);
        sudogs[i] = sd;
        if (i < nsends) {
            runtime_waitq_enqueue(&sc->c->sendq, sd);
        } else {
            runtime_waitq_enqueue(&sc->c->recvq, sd);
        }
    }

    g->status = RUNTIME_G_WAITING;
    runtime_atomic_store_u32(&g->parking, 1);
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("select park", g, NULL, 0);
#endif
    runtime_select_unlock_all(lock_order, lock_count);
    free(lock_order);
    lock_order = NULL;
    lock_count = 0;
    runtime_gopark_after_unlock(g, m);

    selected = g->select_done;
    recvok = g->select_recvok;
#if KOLIBRI_RT_DEBUG
    runtime_debug_event("select wake", g, NULL, (uint32_t)selected);
#endif

cleanup:

    for (i = 0; i < ncases; i++) {
        runtime_sudog* sd = sudogs[i];
        if (sd == NULL) {
            continue;
        }
        if (i == selected) {
            runtime_sudog_free(sd);
            continue;
        }
        runtime_chan_lock(sd->c);
        if (i < nsends) {
            runtime_waitq_remove(&sd->c->sendq, sd);
        } else {
            runtime_waitq_remove(&sd->c->recvq, sd);
        }
        runtime_chan_unlock(sd->c);
        runtime_sudog_free(sd);
    }
    free(sudogs);

    if (selected < 0) {
#if KOLIBRI_RT_DEBUG
        runtime_debug_event("select retry", g, NULL, 0);
#endif
        goto retry_block;
    }

done:
    ret->selected = selected;
    ret->recvOK = recvok;
    return;

selected_unlock:
    runtime_select_unlock_all(lock_order, lock_count);
    free(lock_order);
    lock_order = NULL;
    lock_count = 0;
    free(sudogs);
    goto done;
}

extern char __eh_frame_start;
extern char __eh_frame_end;

void runtime_register_eh_frames(void) {
    static uint8_t runtime_eh_frames_registered = 0;

    if (runtime_eh_frames_registered) {
        return;
    }
    runtime_eh_frames_registered = 1;
#if KOLIBRI_UNWIND_DEBUG
    runtime_debug_mark("EH:begin");
    runtime_debug_eh_frame_summary();
    runtime_debug_mark("EH:skip");
#endif
}

typedef struct {
    runtime_map* map;
    intptr_t index;
} runtime_map_iter_state;

typedef struct {
    void* key;
    void* value;
    runtime_map_iter_state* state;
} runtime_map_iterator;

typedef struct runtime_gc_page_entry runtime_gc_page_entry;

typedef struct {
    const void* addr;
    uintptr_t size;
    uintptr_t ptrdata;
    const uint8_t* gcdata;
} runtime_gc_root_descriptor;

typedef struct runtime_gc_root_block {
    struct runtime_gc_root_block* next;
    uintptr_t count;
    runtime_gc_root_descriptor roots[];
} runtime_gc_root_block;

struct runtime_gc_header {
    runtime_gc_header* next;
    runtime_gc_header* prev;
    uintptr_t size;
    const go_type_descriptor* descriptor;
    runtime_gc_scan_fn scan;
    uintptr_t aux;
    runtime_gc_page_entry* page_entries;
    uint8_t marked;
};

struct runtime_gc_page_entry {
    uintptr_t page_base;
    runtime_gc_header* header;
    runtime_gc_page_entry* next_in_bucket;
    runtime_gc_page_entry* prev_in_bucket;
    runtime_gc_page_entry* next_in_header;
};

#define GO_TYPE_KIND_DIRECT_IFACE 0x20u
#define GO_TYPE_KIND_MASK 0x1Fu
#define GO_TYPE_KIND_INTERFACE 0x14u
#define GO_TYPE_KIND_STRING 0x18u
#define GO_TYPE_KIND_FLOAT32 0x0Du
#define GO_TYPE_KIND_FLOAT64 0x0Eu
#define GO_TYPE_KIND_COMPLEX64 0x0Fu
#define GO_TYPE_KIND_COMPLEX128 0x10u

#define RUNTIME_GC_PAGE_SHIFT 12u
#define RUNTIME_GC_PAGE_SIZE (1u << RUNTIME_GC_PAGE_SHIFT)
#define RUNTIME_GC_PAGE_MASK (RUNTIME_GC_PAGE_SIZE - 1u)
#define RUNTIME_GC_PAGE_BUCKETS 4096u

typedef struct {
    uintptr_t size;
} go_type_size_only_descriptor;

void runtime_panicmem(void);
void runtime_typedmemmove(const go_type_descriptor* descriptor, void* dest, const void* src);
static void runtime_gc_mark_pointer(const void* value);
static void runtime_gc_collect_impl(void);
static void runtime_gc_collect_impl_locked(void);
static void runtime_gc_collect_impl_locked(void);
static void* runtime_gc_alloc_managed(size_t size, const go_type_descriptor* descriptor, runtime_gc_scan_fn scan, void* aux, uintptr_t count);
static void* runtime_gc_alloc_object(const go_type_descriptor* descriptor);
static void* runtime_gc_alloc_array(const go_type_descriptor* descriptor, intptr_t count, size_t total_size);
static runtime_map* runtime_gc_alloc_map_object(void);
static runtime_map_entry* runtime_gc_alloc_map_entries(runtime_map* map, intptr_t cap);
static runtime_map_iter_state* runtime_gc_alloc_map_iter_state(void);
static void runtime_gc_free_exact(void* ptr);
static uint32_t runtime_strhash_impl(const void* value);
static uint32_t runtime_hash_interface(const go_type_descriptor* descriptor, const void* data);
static go_equal_function runtime_resolve_equal_function(const go_type_descriptor* descriptor);
void* runtime_mapassign(const go_map_type_descriptor* map_type, runtime_map* map, const void* key);
void* runtime_mapaccess1(const go_map_type_descriptor* map_type, runtime_map* map, const void* key);
go_mapaccess2_result runtime_mapaccess2(const go_map_type_descriptor* map_type, runtime_map* map, const void* key);
void runtime_mapdelete(const go_map_type_descriptor* map_type, runtime_map* map, const void* key);
void runtime_mapclear(const go_map_type_descriptor* map_type, runtime_map* map);
void* runtime_mapassign__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key);
void* runtime_mapassign__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key);
void* runtime_mapassign__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key);
void* runtime_mapassign__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len);
void* runtime_mapaccess1__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key);
void* runtime_mapaccess1__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key);
void* runtime_mapaccess1__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key);
void* runtime_mapaccess1__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len);
go_mapaccess2_result runtime_mapaccess2__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key);
go_mapaccess2_result runtime_mapaccess2__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key);
go_mapaccess2_result runtime_mapaccess2__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key);
go_mapaccess2_result runtime_mapaccess2__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len);
void runtime_mapdelete__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key);
void runtime_mapdelete__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key);
void runtime_mapdelete__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key);
void runtime_mapdelete__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len);
void runtime_deferprocStack(runtime_defer* d, uint8_t* frame, runtime_defer_fn fn, void* arg);
void runtime_deferproc(uint8_t* frame, runtime_defer_fn fn, void* arg);
void runtime_deferreturn(uint8_t* frame);
void runtime_checkdefer(uint8_t* frame);
bool runtime_canrecover(void* frame);
bool runtime_setdeferretaddr(void* retaddr);
go_mapaccess2_result runtime_ifaceE2T2P(const go_type_descriptor* target_type, const go_type_descriptor* source_type, const void* source_data);
go_mapaccess2_result runtime_ifaceI2T2P(const go_type_descriptor* target_type, const go_interface_method_table* source_methods, const void* source_data);
bool runtime_ifaceT2Ip(const go_type_descriptor* target_type, const go_type_descriptor* source_type);
go_string runtime_intstring(void* tmp, int64_t value);
int runtime_cmpstring(const char* left, intptr_t left_len, const char* right, intptr_t right_len);
intptr_t runtime_typedslicecopy(const go_type_descriptor* descriptor, void* dst, intptr_t dstlen, const void* src, intptr_t srclen);
void* runtime_makeslice64(const go_type_descriptor* descriptor, int64_t len, int64_t cap);
go_interface runtime_getOverflowError(void);
go_interface runtime_getDivideError(void);
void runtime_printlock(void);
void runtime_printunlock(void);
void runtime_printstring(const char* value, intptr_t len);
void runtime_printint(int64_t value);
uint32_t runtime_fastrand(void);
uintptr_t runtime_memhash(const void* value, uintptr_t seed, uintptr_t size);
uintptr_t runtime_memhash8(const void* value, uintptr_t seed);
uintptr_t runtime_memhash16(const void* value, uintptr_t seed);
uintptr_t runtime_memhash32(const void* value, uintptr_t seed);
uintptr_t runtime_memhash64(const void* value, uintptr_t seed);

static const char runtime_hex_digits[] = "0123456789ABCDEF";
static uint32_t runtime_fastrand_state = 1;
static const go_type_descriptor RUNTIME_USED runtime_unsafe_pointer_descriptor = {
    sizeof(void*),
    sizeof(void*),
    0,
    0,
    0,
    0,
    GO_TYPE_KIND_DIRECT_IFACE,
    NULL,
    NULL,
    NULL,
    NULL,
    NULL,
};

static int kos_memcmp(const void* left, const void* right, size_t size);

static size_t kos_strlen(const char* str) {
    const char* cursor = str;
    while (*cursor != '\0') {
        cursor++;
    }
    return (size_t)(cursor - str);
}

static int kos_strcmp(const char* left, const char* right) {
    while (*left != '\0' && *left == *right) {
        left++;
        right++;
    }
    return (int)(*(const unsigned char*)left) - (int)(*(const unsigned char*)right);
}

static bool runtime_string_equals(const go_string* left, const go_string* right) {
    size_t size;

    if (left == right) {
        return true;
    }
    if (left == NULL || right == NULL) {
        return false;
    }
    if (left->len != right->len) {
        return false;
    }
    if (left->len == 0) {
        return true;
    }
    if (left->str == NULL || right->str == NULL) {
        return false;
    }

    size = (size_t)left->len;
    return kos_memcmp(left->str, right->str, size) == 0;
}

int runtime_cmpstring(const char* left, intptr_t left_len, const char* right, intptr_t right_len) {
    intptr_t index;
    intptr_t min_len;

    if (left == right && left_len == right_len) {
        return 0;
    }
    if (left == NULL) {
        return right_len == 0 ? 0 : -1;
    }
    if (right == NULL) {
        return left_len == 0 ? 0 : 1;
    }

    min_len = left_len < right_len ? left_len : right_len;
    for (index = 0; index < min_len; index++) {
        unsigned char a = (unsigned char)left[index];
        unsigned char b = (unsigned char)right[index];
        if (a != b) {
            return a < b ? -1 : 1;
        }
    }

    if (left_len == right_len) {
        return 0;
    }
    return left_len < right_len ? -1 : 1;
}

static void* kos_memcpy(void* dest, const void* src, size_t size) {
    unsigned char* out = (unsigned char*)dest;
    const unsigned char* in = (const unsigned char*)src;

    while (size-- > 0) {
        *out++ = *in++;
    }

    return dest;
}

static void* kos_memmove(void* dest, const void* src, size_t size) {
    unsigned char* out;
    const unsigned char* in;
    size_t index;

    if (dest == src || size == 0) {
        return dest;
    }

    out = (unsigned char*)dest;
    in = (const unsigned char*)src;
    if (out < in || out >= in + size) {
        return kos_memcpy(dest, src, size);
    }

    for (index = size; index > 0; index--) {
        out[index - 1] = in[index - 1];
    }

    return dest;
}

static int kos_memcmp(const void* left, const void* right, size_t size) {
    const unsigned char* left_bytes = (const unsigned char*)left;
    const unsigned char* right_bytes = (const unsigned char*)right;
    size_t index;

    for (index = 0; index < size; index++) {
        if (left_bytes[index] != right_bytes[index]) {
            return (int)left_bytes[index] - (int)right_bytes[index];
        }
    }

    return 0;
}

static void* kos_memset(void* dest, int value, size_t size) {
    unsigned char* out = (unsigned char*)dest;

    while (size-- > 0) {
        *out++ = (unsigned char)value;
    }

    return dest;
}

#if KOLIBRI_RT_DEBUG_FILE
#define RUNTIME_DEBUG_FILE_BUF_SIZE 4096u

typedef struct {
    uint32_t subfunc;
    uint32_t offset;
    uint32_t offset_hi;
    uint32_t size;
    uint32_t data;
    char path[64];
} runtime_fs70_req;

static const char runtime_debug_file_path[] = "/rd/1/goruntime-debug.txt";
static char runtime_debug_file_buf[RUNTIME_DEBUG_FILE_BUF_SIZE];
static uint32_t runtime_debug_file_len = 0;
static uint32_t runtime_debug_file_offset = 0;
static uint8_t runtime_debug_file_initialized = 0;
static volatile uint32_t runtime_debug_file_lock = 0;

static void runtime_debug_file_lock_acquire(void) {
    while (!runtime_atomic_cas_u32((uint32_t*)&runtime_debug_file_lock, 0u, 1u)) {
        runtime_yield();
    }
}

static void runtime_debug_file_lock_release(void) {
    runtime_atomic_store_u32((uint32_t*)&runtime_debug_file_lock, 0u);
}

static void runtime_debug_file_copy_path(runtime_fs70_req* req) {
    size_t i = 0;
    if (req == NULL) {
        return;
    }
    while (runtime_debug_file_path[i] != '\0' && i + 1 < sizeof(req->path)) {
        req->path[i] = runtime_debug_file_path[i];
        i++;
    }
    req->path[i] = '\0';
}

static uint32_t runtime_debug_file_sys70(runtime_fs70_req* req) {
    uint32_t eax = 70;
    uint32_t ebx = (uint32_t)(uintptr_t)req;
    __asm__ volatile("int $0x40"
                     : "+a"(eax), "+b"(ebx)
                     :
                     : "ecx", "edx", "esi", "edi", "memory", "cc");
    return eax;
}

static void runtime_debug_file_init(void) {
    runtime_fs70_req req;

    kos_memset(&req, 0, sizeof(req));
    req.subfunc = 2;
    req.size = 0;
    req.data = 0;
    runtime_debug_file_copy_path(&req);
    runtime_debug_file_sys70(&req);
    runtime_debug_file_offset = 0;
    runtime_debug_file_initialized = 1;
}

static void runtime_debug_file_flush_unlocked(void) {
    runtime_fs70_req req;
    uint32_t result;

    if (runtime_debug_file_len == 0) {
        return;
    }
    if (!runtime_debug_file_initialized) {
        runtime_debug_file_init();
    }

    kos_memset(&req, 0, sizeof(req));
    req.subfunc = 3;
    req.offset = runtime_debug_file_offset;
    req.offset_hi = 0;
    req.size = runtime_debug_file_len;
    req.data = (uint32_t)(uintptr_t)runtime_debug_file_buf;
    runtime_debug_file_copy_path(&req);

    result = runtime_debug_file_sys70(&req);
    if (result == 5u) {
        runtime_debug_file_init();
        result = runtime_debug_file_sys70(&req);
    }
    if (result == 0u) {
        runtime_debug_file_offset += runtime_debug_file_len;
    }
    runtime_debug_file_len = 0;
}

static void runtime_debug_file_write_byte(unsigned char value) {
    runtime_debug_file_lock_acquire();
    runtime_debug_file_buf[runtime_debug_file_len++] = (char)value;
    if (runtime_debug_file_len >= RUNTIME_DEBUG_FILE_BUF_SIZE) {
        runtime_debug_file_flush_unlocked();
    }
    runtime_debug_file_lock_release();
}

static void runtime_debug_file_flush(void) {
    runtime_debug_file_lock_acquire();
    runtime_debug_file_flush_unlocked();
    runtime_debug_file_lock_release();
}
#endif

static void runtime_debug_write_byte(unsigned char value) {
    uint32_t eax;
    uint32_t ebx;
    uint32_t ecx;

    eax = 63;
    ebx = 1;
    ecx = (uint32_t)value;
    __asm__ volatile("int $0x40"
                     : "+a"(eax), "+b"(ebx), "+c"(ecx)
                     :
                     : "edx", "esi", "edi", "memory", "cc");

#if KOLIBRI_RT_DEBUG_FILE
    runtime_debug_file_write_byte(value);
#endif
}

static void runtime_debug_write_bytes(const char* value, size_t size) {
    size_t index;

    if (value == NULL) {
        return;
    }

    for (index = 0; index < size; index++) {
        runtime_debug_write_byte((unsigned char)value[index]);
    }
}

static void runtime_debug_write_cstring(const char* value) {
    if (value == NULL) {
        return;
    }

    runtime_debug_write_bytes(value, kos_strlen(value));
}

static void runtime_debug_write_hex32(uint32_t value) {
    int shift;

    runtime_debug_write_cstring("0x");
    for (shift = 28; shift >= 0; shift -= 4) {
        runtime_debug_write_byte((unsigned char)runtime_hex_digits[(value >> shift) & 0x0F]);
    }
}

static void runtime_debug_write_newline(void) {
    runtime_debug_write_byte('\r');
    runtime_debug_write_byte('\n');
#if KOLIBRI_RT_DEBUG_FILE
    runtime_debug_file_flush();
#endif
}

#if KOLIBRI_RT_DEBUG
static uint32_t runtime_debug_budget = 1200u;

static bool runtime_debug_take(void) {
    if (runtime_debug_budget == 0u) {
        return false;
    }
    runtime_debug_budget--;
    return true;
}

static void runtime_debug_event(const char* tag, runtime_g* g, runtime_hchan* c, uint32_t extra) {
    if (!runtime_debug_take()) {
        return;
    }
    runtime_debug_write_cstring(tag);
    runtime_debug_write_cstring(" g=");
    runtime_debug_write_hex32((uint32_t)(uintptr_t)g);
    runtime_debug_write_cstring(" c=");
    runtime_debug_write_hex32((uint32_t)(uintptr_t)c);
    runtime_debug_write_cstring(" m=");
    runtime_debug_write_hex32((uint32_t)(uintptr_t)runtime_getm());
    runtime_debug_write_cstring(" tid=");
    runtime_debug_write_hex32(runtime_kolibri_current_thread_slot());
    runtime_debug_write_cstring(" ex=");
    runtime_debug_write_hex32(extra);
    runtime_debug_write_newline();
}
#endif

static void runtime_debug_mark(const char* tag) {
    if (tag == NULL) {
        return;
    }
    runtime_debug_write_cstring(tag);
    runtime_debug_write_newline();
}

static void runtime_debug_eh_frame_summary(void) {
    const uint8_t* start = (const uint8_t*)&__eh_frame_start;
    const uint8_t* end = (const uint8_t*)&__eh_frame_end;
    const uint8_t* p = start;
    size_t size = 0;
    size_t used = 0;
    uint32_t len;
    uint32_t id;
    uint32_t entries = 0;

    runtime_debug_write_cstring("EH:start=");
    runtime_debug_write_hex32((uint32_t)(uintptr_t)start);
    runtime_debug_write_cstring(" end=");
    runtime_debug_write_hex32((uint32_t)(uintptr_t)end);
    runtime_debug_write_newline();

    if (end > start) {
        size = (size_t)(end - start);
    }
    runtime_debug_write_cstring("EH:size=");
    runtime_debug_write_hex32((uint32_t)size);
    runtime_debug_write_newline();

    if (size < 8) {
        runtime_debug_mark("EH:scan too small");
        return;
    }

    len = *(const uint32_t*)p;
    id = *(const uint32_t*)(p + 4);
    runtime_debug_write_cstring("EH:first len=");
    runtime_debug_write_hex32(len);
    runtime_debug_write_cstring(" id=");
    runtime_debug_write_hex32(id);
    runtime_debug_write_newline();

    while (used + 4 <= size) {
        len = *(const uint32_t*)p;
        if (len == 0) {
            runtime_debug_write_cstring("EH:scan ok entries=");
            runtime_debug_write_hex32(entries);
            runtime_debug_write_newline();
            return;
        }
        if (len == 0xffffffffu) {
            runtime_debug_mark("EH:scan dwarf64");
            return;
        }
        if (len > size - used - 4) {
            runtime_debug_write_cstring("EH:scan bad len=");
            runtime_debug_write_hex32(len);
            runtime_debug_write_newline();
            return;
        }
        p += 4 + len;
        used += 4 + len;
        entries++;
        if (entries > 0x10000u) {
            runtime_debug_mark("EH:scan too many");
            return;
        }
    }

    runtime_debug_mark("EH:scan overflow");
}

typedef unsigned int uword;
typedef int sword;
typedef unsigned char ubyte;
struct dwarf_cie {
    uword length;
    sword CIE_id;
    ubyte version;
    unsigned char augmentation[];
} __attribute__((packed, aligned(__alignof__(void*))));

struct dwarf_fde {
    uword length;
    sword CIE_delta;
    unsigned char pc_begin[];
} __attribute__((packed, aligned(__alignof__(void*))));

static inline const struct dwarf_cie* runtime_get_cie(const struct dwarf_fde* f) {
    return (const struct dwarf_cie*)((const char*)&f->CIE_delta - f->CIE_delta);
}

static inline const struct dwarf_fde* runtime_next_fde(const struct dwarf_fde* f) {
    return (const struct dwarf_fde*)((const char*)f + f->length + sizeof(f->length));
}

static const unsigned char* runtime_read_uleb128(const unsigned char* p, _uleb128_t* val) {
    unsigned int shift = 0;
    _uleb128_t result = 0;
    unsigned char byte;

    do {
        byte = *p++;
        result |= ((_uleb128_t)byte & 0x7f) << shift;
        shift += 7;
    } while (byte & 0x80);

    *val = result;
    return p;
}

static const unsigned char* runtime_read_sleb128(const unsigned char* p, _sleb128_t* val) {
    unsigned int shift = 0;
    _uleb128_t result = 0;
    unsigned char byte;

    do {
        byte = *p++;
        result |= ((_uleb128_t)byte & 0x7f) << shift;
        shift += 7;
    } while (byte & 0x80);

    if (shift < 8 * sizeof(result) && (byte & 0x40) != 0) {
        result |= -(((_uleb128_t)1) << shift);
    }

    *val = (_sleb128_t)result;
    return p;
}

static const unsigned char* runtime_read_uleb128_limited(const unsigned char* p, const unsigned char* end,
                                                         _uleb128_t* val) {
    unsigned int shift = 0;
    _uleb128_t result = 0;
    unsigned char byte;
    int count = 0;

    if (p == NULL || end == NULL || p >= end) {
        return NULL;
    }

    do {
        if (p >= end || count++ > 10) {
            return NULL;
        }
        byte = *p++;
        result |= ((_uleb128_t)byte & 0x7f) << shift;
        shift += 7;
    } while (byte & 0x80);

    *val = result;
    return p;
}

static const unsigned char* runtime_read_sleb128_limited(const unsigned char* p, const unsigned char* end,
                                                         _sleb128_t* val) {
    unsigned int shift = 0;
    _uleb128_t result = 0;
    unsigned char byte = 0;
    int count = 0;

    if (p == NULL || end == NULL || p >= end) {
        return NULL;
    }

    do {
        if (p >= end || count++ > 10) {
            return NULL;
        }
        byte = *p++;
        result |= ((_uleb128_t)byte & 0x7f) << shift;
        shift += 7;
    } while (byte & 0x80);

    if (shift < 8 * sizeof(result) && (byte & 0x40) != 0) {
        result |= -(((_uleb128_t)1) << shift);
    }

    *val = (_sleb128_t)result;
    return p;
}

static const unsigned char* runtime_read_encoded_value_with_base(unsigned char encoding, _Unwind_Ptr base,
                                                                  const unsigned char* p, _Unwind_Ptr* val) {
    union unaligned {
        void* ptr;
        unsigned u2 __attribute__((mode(HI)));
        unsigned u4 __attribute__((mode(SI)));
        unsigned u8 __attribute__((mode(DI)));
        signed s2 __attribute__((mode(HI)));
        signed s4 __attribute__((mode(SI)));
        signed s8 __attribute__((mode(DI)));
    } __attribute__((packed));

    const union unaligned* u = (const union unaligned*)p;
    _Unwind_Internal_Ptr result;

    if (encoding == DW_EH_PE_aligned) {
        _Unwind_Internal_Ptr a = (_Unwind_Internal_Ptr)p;
        a = (a + sizeof(void*) - 1) & -((intptr_t)sizeof(void*));
        result = *(_Unwind_Internal_Ptr*)a;
        p = (const unsigned char*)(_Unwind_Internal_Ptr)(a + sizeof(void*));
    } else {
        switch (encoding & 0x0f) {
            case DW_EH_PE_absptr:
                result = (_Unwind_Internal_Ptr)u->ptr;
                p += sizeof(void*);
                break;
            case DW_EH_PE_uleb128: {
                _uleb128_t tmp;
                p = runtime_read_uleb128(p, &tmp);
                result = (_Unwind_Internal_Ptr)tmp;
                break;
            }
            case DW_EH_PE_sleb128: {
                _sleb128_t tmp;
                p = runtime_read_sleb128(p, &tmp);
                result = (_Unwind_Internal_Ptr)tmp;
                break;
            }
            case DW_EH_PE_udata2:
                result = u->u2;
                p += 2;
                break;
            case DW_EH_PE_udata4:
                result = u->u4;
                p += 4;
                break;
            case DW_EH_PE_udata8:
                result = u->u8;
                p += 8;
                break;
            case DW_EH_PE_sdata2:
                result = u->s2;
                p += 2;
                break;
            case DW_EH_PE_sdata4:
                result = u->s4;
                p += 4;
                break;
            case DW_EH_PE_sdata8:
                result = u->s8;
                p += 8;
                break;
            default:
                abort();
        }

        if (result != 0) {
            result += ((encoding & 0x70) == DW_EH_PE_pcrel ? (_Unwind_Internal_Ptr)u : base);
            if (encoding & DW_EH_PE_indirect) {
                result = *(_Unwind_Internal_Ptr*)result;
            }
        }
    }

    *val = result;
    return p;
}

static int runtime_get_cie_encoding(const struct dwarf_cie* cie) {
    const unsigned char* aug;
    const unsigned char* p;
    const unsigned char* cie_end;
    const unsigned char* scan;
    _Unwind_Ptr dummy;
    _uleb128_t utmp = 0;
    _sleb128_t stmp = 0;
    size_t aug_len = 0;

    if (cie == NULL) {
        return DW_EH_PE_omit;
    }
    cie_end = (const unsigned char*)cie + sizeof(uword) + cie->length;
    if (cie_end <= (const unsigned char*)cie) {
        return DW_EH_PE_omit;
    }
    aug = cie->augmentation;
    scan = aug;
    while (scan < cie_end && *scan != 0) {
        scan++;
    }
    if (scan >= cie_end) {
        return DW_EH_PE_omit;
    }
    aug_len = (size_t)(scan - aug);
    p = aug + aug_len + 1;
    if (p >= cie_end) {
        return DW_EH_PE_omit;
    }
    if (__builtin_expect(cie->version >= 4, 0)) {
        if (p + 2 > cie_end || p[0] != sizeof(void*) || p[1] != 0) {
            return DW_EH_PE_omit;
        }
        p += 2;
        if (p >= cie_end) {
            return DW_EH_PE_omit;
        }
    }

    if (aug[0] != 'z') {
        return DW_EH_PE_absptr;
    }

    p = runtime_read_uleb128_limited(p, cie_end, &utmp);
    if (p == NULL) {
        return DW_EH_PE_omit;
    }
    p = runtime_read_sleb128_limited(p, cie_end, &stmp);
    if (p == NULL) {
        return DW_EH_PE_omit;
    }
    if (cie->version == 1) {
        if (p >= cie_end) {
            return DW_EH_PE_omit;
        }
        p++;
    } else {
        p = runtime_read_uleb128_limited(p, cie_end, &utmp);
        if (p == NULL) {
            return DW_EH_PE_omit;
        }
    }

    aug++;
    p = runtime_read_uleb128_limited(p, cie_end, &utmp);
    if (p == NULL) {
        return DW_EH_PE_omit;
    }
    while (1) {
        if (*aug == 'R') {
            return *p;
        } else if (*aug == 'P') {
            if (p + 1 >= cie_end) {
                return DW_EH_PE_omit;
            }
            p = runtime_read_encoded_value_with_base(*p, 0, p + 1, &dummy);
        } else if (*aug == 'L') {
            if (p >= cie_end) {
                return DW_EH_PE_omit;
            }
            p++;
        } else if (*aug == '\0') {
            break;
        }
        aug++;
    }

    return DW_EH_PE_absptr;
}

static void runtime_debug_cie_personality(const struct dwarf_cie* cie) {
    const unsigned char* p;
    const unsigned char* aug;
    const unsigned char* cie_end;
    _uleb128_t utmp = 0;
    _sleb128_t stmp = 0;
    _Unwind_Ptr personality = 0;
    unsigned char encoding = 0;

    if (cie == NULL) {
        return;
    }

    cie_end = (const unsigned char*)cie + sizeof(uword) + cie->length;
    if (cie_end <= (const unsigned char*)cie) {
        return;
    }

    aug = cie->augmentation;
    p = aug;
    while (p < cie_end && *p != 0) {
        p++;
    }
    if (p >= cie_end) {
        return;
    }
    p++; // skip NUL

    if (p >= cie_end) {
        return;
    }

    if (cie->version >= 4) {
        if (p + 2 > cie_end) {
            return;
        }
        p += 2;
    }

    if (aug[0] != 'z') {
        return;
    }

    p = runtime_read_uleb128_limited(p, cie_end, &utmp);
    if (p == NULL) {
        return;
    }
    p = runtime_read_sleb128_limited(p, cie_end, &stmp);
    if (p == NULL) {
        return;
    }
    if (cie->version == 1) {
        if (p >= cie_end) {
            return;
        }
        p++;
    } else {
        p = runtime_read_uleb128_limited(p, cie_end, &utmp);
        if (p == NULL) {
            return;
        }
    }

    p = runtime_read_uleb128_limited(p, cie_end, &utmp);
    if (p == NULL) {
        return;
    }

    aug++; // skip 'z'
    while (*aug != '\0') {
        if (p >= cie_end) {
            return;
        }
        if (*aug == 'P') {
            encoding = *p++;
            p = runtime_read_encoded_value_with_base(encoding, 0, p, &personality);
#if KOLIBRI_UNWIND_DEBUG
            runtime_debug_write_cstring("CIE:P enc=");
            runtime_debug_write_hex32((uint32_t)encoding);
            runtime_debug_write_cstring(" val=");
            runtime_debug_write_hex32((uint32_t)personality);
            runtime_debug_write_newline();
#endif
        } else if (*aug == 'L') {
#if KOLIBRI_UNWIND_DEBUG
            runtime_debug_write_cstring("CIE:L enc=");
            runtime_debug_write_hex32((uint32_t)*p);
            runtime_debug_write_newline();
#endif
            p++;
        } else if (*aug == 'R') {
#if KOLIBRI_UNWIND_DEBUG
            runtime_debug_write_cstring("CIE:R enc=");
            runtime_debug_write_hex32((uint32_t)*p);
            runtime_debug_write_newline();
#endif
            p++;
        }
        aug++;
    }
}

#if defined(KOLIBRI_CUSTOM_UNWIND_FDE)
struct dwarf_eh_bases {
    void* tbase;
    void* dbase;
    void* func;
};

const struct dwarf_fde* _Unwind_Find_FDE(void* pc, struct dwarf_eh_bases* bases) {
    const uint8_t* start = (const uint8_t*)&__eh_frame_start;
    const uint8_t* end = (const uint8_t*)&__eh_frame_end;
    const struct dwarf_fde* fde;
    _Unwind_Ptr pc_val;
    uint32_t iterations = 0;
#if KOLIBRI_UNWIND_DEBUG
    static uint32_t fde_debug_calls = 0;
    uint8_t debug_now = 0;
    static _Unwind_Ptr last_pc_val = 0;
    static uint32_t repeat_pc_count = 0;
#endif

    if (bases == NULL || start == NULL || end == NULL || end <= start) {
        return NULL;
    }

    if (pc == NULL) {
        return NULL;
    }
    pc_val = (_Unwind_Ptr)pc;
    if (pc_val > 0) {
        pc_val -= 1;
    }

#if KOLIBRI_UNWIND_DEBUG
    if (fde_debug_calls < 32) {
        debug_now = 1;
        fde_debug_calls++;
        runtime_debug_write_cstring("FDE:pc=");
        runtime_debug_write_hex32((uint32_t)pc_val);
        runtime_debug_write_newline();
    } else {
        fde_debug_calls++;
    }

    if (pc_val == last_pc_val) {
        repeat_pc_count++;
        if (repeat_pc_count == 32) {
            runtime_debug_mark("FDE:pc repeat");
        }
        if (repeat_pc_count > 512) {
            runtime_debug_mark("FDE:pc stuck");
            return NULL;
        }
    } else {
        last_pc_val = pc_val;
        repeat_pc_count = 0;
    }
#endif
    fde = (const struct dwarf_fde*)start;

    while ((const uint8_t*)fde + sizeof(uword) <= end) {
        uword length = fde->length;
        size_t remaining = (size_t)(end - (const uint8_t*)fde);
        iterations++;
        if (iterations > 0x4000u) {
            return NULL;
        }
        if (length == 0) {
            return NULL;
        }
        if (length > remaining - sizeof(uword)) {
            return NULL;
        }
#if KOLIBRI_UNWIND_DEBUG
        if (debug_now) {
            runtime_debug_write_cstring("FDE:len=");
            runtime_debug_write_hex32((uint32_t)length);
            runtime_debug_write_cstring(" delta=");
            runtime_debug_write_hex32((uint32_t)fde->CIE_delta);
            runtime_debug_write_newline();
        }
#endif
        if (fde->CIE_delta != 0) {
            const struct dwarf_cie* cie = runtime_get_cie(fde);
            const uint8_t* cie_ptr = (const uint8_t*)cie;
            const uint8_t* cie_end = NULL;
            int encoding;

            if (cie_ptr < start || cie_ptr + sizeof(uword) > end) {
#if KOLIBRI_UNWIND_DEBUG
                if (debug_now) {
                    runtime_debug_mark("FDE:bad cie ptr");
                }
#endif
                fde = runtime_next_fde(fde);
                continue;
            }
            cie_end = cie_ptr + sizeof(uword) + cie->length;
            if (cie_end <= cie_ptr || cie_end > end) {
#if KOLIBRI_UNWIND_DEBUG
                if (debug_now) {
                    runtime_debug_mark("FDE:bad cie len");
                }
#endif
                fde = runtime_next_fde(fde);
                continue;
            }

            encoding = runtime_get_cie_encoding(cie);
            if (encoding != DW_EH_PE_omit) {
                _Unwind_Ptr pc_begin = 0;
                _Unwind_Ptr pc_range = 0;
                const unsigned char* p = runtime_read_encoded_value_with_base(encoding, 0, fde->pc_begin, &pc_begin);
                runtime_read_encoded_value_with_base(encoding & 0x0f, 0, p, &pc_range);
#if KOLIBRI_UNWIND_DEBUG
                if (debug_now) {
                    runtime_debug_write_cstring("FDE:enc=");
                    runtime_debug_write_hex32((uint32_t)encoding);
                    runtime_debug_write_cstring(" begin=");
                    runtime_debug_write_hex32((uint32_t)pc_begin);
                    runtime_debug_write_cstring(" range=");
                    runtime_debug_write_hex32((uint32_t)pc_range);
                    runtime_debug_write_newline();
                }
#endif
                if (pc_val >= pc_begin && pc_val < pc_begin + pc_range) {
                    bases->tbase = 0;
                    bases->dbase = 0;
                    bases->func = (void*)pc_begin;
                    return fde;
                }
            }
        }
        fde = runtime_next_fde(fde);
    }

    return NULL;
}
#endif

static void runtime_debug_write_int64(int64_t value) {
    char buffer[32];
    uint64_t magnitude;
    size_t index;

    magnitude = (uint64_t)value;
    if (value < 0) {
        magnitude = (uint64_t)(-value);
    }

    index = 0;
    if (magnitude == 0) {
        buffer[index++] = '0';
    } else {
        while (magnitude != 0 && index < sizeof(buffer)) {
            buffer[index++] = (char)('0' + (magnitude % 10));
            magnitude /= 10;
        }
    }

    if (value < 0 && index < sizeof(buffer)) {
        buffer[index++] = '-';
    }

    while (index > 0) {
        index--;
        runtime_debug_write_byte((unsigned char)buffer[index]);
    }
}

static uint32_t runtime_hash_bytes(const unsigned char* data, size_t size) {
    uint32_t hash;
    size_t index;

    if (data == NULL || size == 0) {
        return 0;
    }

    hash = 2166136261u;
    for (index = 0; index < size; index++) {
        hash ^= (uint32_t)data[index];
        hash *= 16777619u;
    }

    return hash;
}

static uint32_t runtime_hash_bytes_seeded(const unsigned char* data, size_t size, uint32_t seed) {
    uint32_t hash;
    size_t index;

    if (data == NULL || size == 0) {
        return seed;
    }

    hash = 2166136261u ^ seed;
    for (index = 0; index < size; index++) {
        hash ^= (uint32_t)data[index];
        hash *= 16777619u;
    }

    return hash;
}

static uint32_t runtime_hash_float32_value(float value) {
    uint32_t bits;

    kos_memcpy(&bits, &value, sizeof(bits));
    if (value == 0.0f) {
        bits = 0;
    } else if (value != value) {
        bits = 0x7fc00000u;
    }

    return runtime_hash_bytes((const unsigned char*)&bits, sizeof(bits));
}

static uint32_t runtime_hash_float64_value(double value) {
    uint64_t bits;

    kos_memcpy(&bits, &value, sizeof(bits));
    if (value == 0.0) {
        bits = 0;
    } else if (value != value) {
        bits = 0x7ff8000000000000ull;
    }

    return runtime_hash_bytes((const unsigned char*)&bits, sizeof(bits));
}

static uint32_t runtime_hash_value(const go_type_descriptor* descriptor, const void* data) {
    uint8_t kind;
    uintptr_t direct_value;

    if (descriptor == NULL) {
        return 0;
    }

    if ((descriptor->kind & GO_TYPE_KIND_DIRECT_IFACE) != 0) {
        direct_value = (uintptr_t)data;
        return runtime_hash_bytes((const unsigned char*)&direct_value, sizeof(direct_value));
    }

    if (data == NULL) {
        return 0;
    }

    kind = descriptor->kind & GO_TYPE_KIND_MASK;
    if (kind == GO_TYPE_KIND_INTERFACE) {
        return runtime_hash_interface(descriptor, data);
    }
    if (kind == GO_TYPE_KIND_STRING) {
        return runtime_strhash_impl(data);
    }
    if (kind == GO_TYPE_KIND_FLOAT32) {
        return runtime_hash_float32_value(*(const float*)data);
    }
    if (kind == GO_TYPE_KIND_FLOAT64) {
        return runtime_hash_float64_value(*(const double*)data);
    }
    if (kind == GO_TYPE_KIND_COMPLEX64) {
        const float* parts = (const float*)data;
        uint32_t re = runtime_hash_float32_value(parts[0]);
        uint32_t im = runtime_hash_float32_value(parts[1]);
        return re ^ (im * 16777619u);
    }
    if (kind == GO_TYPE_KIND_COMPLEX128) {
        const double* parts = (const double*)data;
        uint32_t re = runtime_hash_float64_value(parts[0]);
        uint32_t im = runtime_hash_float64_value(parts[1]);
        return re ^ (im * 16777619u);
    }

    return runtime_hash_bytes((const unsigned char*)data, (size_t)descriptor->size);
}

static inline uint32_t runtime_atomic_load_u32(const volatile uint32_t* value) {
    return __atomic_load_n(value, __ATOMIC_ACQUIRE);
}

static inline void runtime_atomic_store_u32(uint32_t* value, uint32_t next) {
    __atomic_store_n(value, next, __ATOMIC_RELEASE);
}

static inline bool runtime_atomic_cas_u32(uint32_t* value, uint32_t expected, uint32_t desired) {
    return __sync_bool_compare_and_swap(value, expected, desired);
}

static inline uint32_t runtime_atomic_xadd_u32(volatile uint32_t* value, uint32_t delta) {
    return __sync_fetch_and_add(value, delta);
}

static inline uintptr_t runtime_atomic_load_uintptr(const uintptr_t* value) {
    return __atomic_load_n(value, __ATOMIC_ACQUIRE);
}

static inline void runtime_atomic_store_uintptr(uintptr_t* value, uintptr_t next) {
    __atomic_store_n(value, next, __ATOMIC_RELEASE);
}

static void runtime_yield(void);

static void runtime_lock_mutex(runtime_mutex* m) {
    if (m == NULL) {
        return;
    }
    for (;;) {
        if (runtime_world_stopping) {
            runtime_poll_world_stop();
        }
        if (runtime_atomic_cas_u32(&m->state, 0, 1)) {
            return;
        }
        runtime_yield();
    }
}

static void runtime_unlock_mutex(runtime_mutex* m) {
    if (m == NULL) {
        return;
    }
    runtime_atomic_store_u32(&m->state, 0);
}

void runtime_lock(runtime_mutex* m) __asm__("runtime.lock");
void runtime_unlock(runtime_mutex* m) __asm__("runtime.unlock");

void runtime_lock(runtime_mutex* m) {
    runtime_lock_mutex(m);
}

void runtime_unlock(runtime_mutex* m) {
    runtime_unlock_mutex(m);
}

static void runtime_yield(void) {
    uint32_t eax = 68;
    uint32_t ebx = 1;

    __asm__ volatile("int $0x40"
                     : "+a"(eax), "+b"(ebx)
                     :
                     : "ecx", "edx", "esi", "edi", "memory", "cc");
}

static void runtime_sleep_ticks(uint32_t ticks) {
    uint32_t eax = 5;
    uint32_t ebx = ticks;

    __asm__ volatile("int $0x40"
                     : "+a"(eax), "+b"(ebx)
                     :
                     : "ecx", "edx", "esi", "edi", "memory", "cc");
}

uint32_t runtime_fastrand(void) {
    runtime_fastrand_state = runtime_fastrand_state * 1664525u + 1013904223u;
    return runtime_fastrand_state;
}

__attribute__((noreturn)) static void runtime_exit_process(void) {
    int32_t eax;

    eax = -1;
    __asm__ volatile("int $0x40"
                     : "+a"(eax)
                     :
                     : "ebx", "ecx", "edx", "esi", "edi", "memory", "cc");
    for (;;) {
    }
}

__attribute__((noreturn)) static void runtime_fail_simple(const char* reason) {
    runtime_debug_write_cstring("runtime panic: ");
    runtime_debug_write_cstring(reason);
    runtime_debug_write_newline();
    runtime_exit_process();
}

__attribute__((noreturn)) static void runtime_fail_pair(const char* reason, const char* first_name, uint32_t first_value, const char* second_name, uint32_t second_value) {
    runtime_debug_write_cstring("runtime panic: ");
    runtime_debug_write_cstring(reason);
    runtime_debug_write_cstring(" (");
    runtime_debug_write_cstring(first_name);
    runtime_debug_write_cstring("=");
    runtime_debug_write_hex32(first_value);
    runtime_debug_write_cstring(", ");
    runtime_debug_write_cstring(second_name);
    runtime_debug_write_cstring("=");
    runtime_debug_write_hex32(second_value);
    runtime_debug_write_cstring(")");
    runtime_debug_write_newline();
    runtime_exit_process();
}

void runtime_printlock(void) {
}

void runtime_printunlock(void) {
}

void runtime_printstring(const char* value, intptr_t len) {
    if (value == NULL || len <= 0) {
        return;
    }

    runtime_debug_write_bytes(value, (size_t)len);
}

void runtime_printint(int64_t value) {
    runtime_debug_write_int64(value);
}

__attribute__((noreturn)) void throw(go_string message) {
    runtime_debug_write_cstring("runtime panic: ");
    if (message.str != NULL && message.len > 0) {
        runtime_debug_write_bytes((const unsigned char*)message.str, (size_t)message.len);
    }
    runtime_debug_write_newline();
    runtime_exit_process();
}

void runtime_Semacquire(uint32_t* semaphore) {
    if (semaphore == NULL) {
        return;
    }

    for (;;) {
        uint32_t value = runtime_atomic_load_u32(semaphore);
        if (value > 0 && runtime_atomic_cas_u32(semaphore, value, value - 1)) {
            return;
        }
        runtime_yield();
    }
}

void runtime_SemacquireMutex(uint32_t* semaphore, bool lifo, int32_t skipframes) {
    (void)lifo;
    (void)skipframes;
    runtime_Semacquire(semaphore);
}

void runtime_Semrelease(uint32_t* semaphore, bool handoff, int32_t skipframes) {
    (void)handoff;
    (void)skipframes;
    if (semaphore == NULL) {
        return;
    }
    runtime_atomic_xadd_u32(semaphore, 1);
}

static inline bool runtime_notify_less(uint32_t left, uint32_t right) {
    return ((int32_t)(left - right)) < 0;
}

uint32_t runtime_notifyListAdd(runtime_notify_list* list) {
    if (list == NULL) {
        return 0;
    }
    return runtime_atomic_xadd_u32(&list->wait, 1);
}

void runtime_notifyListWait(runtime_notify_list* list, uint32_t ticket) {
    if (list == NULL) {
        return;
    }
    while (!runtime_notify_less(ticket, runtime_atomic_load_u32(&list->notify))) {
        runtime_yield();
    }
}

void runtime_notifyListNotifyAll(runtime_notify_list* list) {
    if (list == NULL) {
        return;
    }
    runtime_atomic_store_u32(&list->notify, runtime_atomic_load_u32(&list->wait));
}

void runtime_notifyListNotifyOne(runtime_notify_list* list) {
    if (list == NULL) {
        return;
    }
    for (;;) {
        uint32_t notify = runtime_atomic_load_u32(&list->notify);
        uint32_t wait = runtime_atomic_load_u32(&list->wait);
        if (notify == wait) {
            return;
        }
        if (runtime_atomic_cas_u32(&list->notify, notify, notify + 1)) {
            return;
        }
    }
}

void runtime_notifyListCheck(uintptr_t size) {
    if (size != sizeof(runtime_notify_list)) {
        runtime_fail_simple("notifyList size mismatch");
    }
}

bool runtime_canSpin(int32_t iter) {
    (void)iter;
    return false;
}

void runtime_doSpin(void) {
    runtime_yield();
}

int64_t runtime_nanotime(void) {
    uint32_t eax = 26;
    uint32_t ebx = 10;
    uint32_t edx = 0;

    __asm__ volatile("int $0x40"
                     : "+a"(eax), "+b"(ebx), "=d"(edx)
                     :
                     : "ecx", "esi", "edi", "memory", "cc");

    return ((int64_t)edx << 32) | (int64_t)eax;
}

void runtime_registerPoolCleanup(runtime_func_val* cleanup) {
    (void)cleanup;
}

int runtime_procPin(void) {
    return 0;
}

void runtime_procUnpin(void) {
}

uintptr_t runtime_internal_atomic_load_acquintptr(const uintptr_t* value) {
    return runtime_atomic_load_uintptr(value);
}

uintptr_t runtime_internal_atomic_store_reluintptr(uintptr_t* value, uintptr_t next) {
    runtime_atomic_store_uintptr(value, next);
    return next;
}

static size_t kos_slice_allocation_size(const go_type_descriptor* descriptor, intptr_t count) {
    size_t element_size;

    if (count < 0) {
        runtime_panicmem();
    }

    if (count == 0) {
        return 0;
    }

    element_size = 0;
    if (descriptor != NULL) {
        element_size = (size_t)descriptor->size;
    }

    if (element_size == 0) {
        return 1;
    }

    if ((size_t)count > ((size_t)-1) / element_size) {
        runtime_panicmem();
    }

    return (size_t)count * element_size;
}

static int RUNTIME_USED runtime_write_barrier_enabled = 0;
static char* runtime_window_title_buffer = NULL;
static size_t runtime_window_title_capacity = 0;
static runtime_gc_header* runtime_gc_objects = NULL;
static runtime_gc_root_block* runtime_gc_roots = NULL;
static void* runtime_gc_stack_top = NULL;
static size_t runtime_gc_live_bytes = 0;
static size_t runtime_gc_live_objects = 0;
static size_t runtime_gc_threshold = 32768u;
static size_t runtime_gc_collection_count = 0;
static int runtime_gc_running = 0;
static int runtime_gc_enabled = 0;
static int runtime_gc_poll_retry = 0;
static uint32_t runtime_gc_low_mem_kb = 8192u;
static uint32_t runtime_gc_memcheck_counter = 0;
static uint8_t runtime_gc_mark_token = 1;
static runtime_gc_page_entry* runtime_gc_page_buckets[RUNTIME_GC_PAGE_BUCKETS];
static uintptr_t runtime_gc_heap_min = 0;
static uintptr_t runtime_gc_heap_max = 0;
static uint8_t runtime_gc_page_index_complete = 1;
static uint64_t runtime_kos_heap_alloc_count = 0;
static uint64_t runtime_kos_heap_alloc_bytes = 0;
static uint64_t runtime_kos_heap_free_count = 0;
static uint64_t runtime_kos_heap_realloc_count = 0;
static uint64_t runtime_kos_heap_realloc_bytes = 0;
static uint64_t runtime_gc_alloc_count = 0;
static uint64_t runtime_gc_alloc_bytes = 0;

static void runtime_gc_mark_pointer(const void* value);
static void runtime_gc_collect_impl(void);

static void runtime_gc_record_stack_top(const void* ptr) {
    uintptr_t value;

    if (ptr == NULL) {
        return;
    }

    value = (uintptr_t)ptr;
    if (runtime_gc_stack_top == NULL || value > (uintptr_t)runtime_gc_stack_top) {
        runtime_gc_stack_top = (void*)value;
    }
}

void runtime_gc_set_stack_top(const void* ptr) {
    runtime_gc_record_stack_top(ptr);
    if (!runtime_g_initialized) {
        runtime_init_g0();
    }
    runtime_g0.stack_top = (uintptr_t)runtime_gc_stack_top;
    runtime_gc_enabled = runtime_gc_stack_top != NULL;
}

static void* runtime_gc_payload(runtime_gc_header* header) {
    if (header == NULL) {
        return NULL;
    }

    return (void*)(header + 1);
}

static const void* runtime_gc_payload_const(const runtime_gc_header* header) {
    if (header == NULL) {
        return NULL;
    }

    return (const void*)(header + 1);
}

static uintptr_t runtime_gc_page_base(uintptr_t address) {
    return address & ~(uintptr_t)RUNTIME_GC_PAGE_MASK;
}

static size_t runtime_gc_page_bucket(uintptr_t page_base) {
    uintptr_t key;

    key = page_base >> RUNTIME_GC_PAGE_SHIFT;
    return (size_t)((key * (uintptr_t)2654435761u) & (RUNTIME_GC_PAGE_BUCKETS - 1u));
}

static void runtime_gc_update_heap_bounds_on_alloc(runtime_gc_header* header) {
    uintptr_t start;
    uintptr_t end;

    if (header == NULL) {
        return;
    }

    start = (uintptr_t)runtime_gc_payload(header);
    end = start + header->size;
    if (end < start) {
        end = (uintptr_t)-1;
    }

    if (runtime_gc_heap_min == 0 || start < runtime_gc_heap_min) {
        runtime_gc_heap_min = start;
    }
    if (end > runtime_gc_heap_max) {
        runtime_gc_heap_max = end;
    }
}

static void runtime_gc_recompute_heap_bounds(void) {
    runtime_gc_header* current;

    runtime_gc_heap_min = 0;
    runtime_gc_heap_max = 0;
    for (current = runtime_gc_objects; current != NULL; current = current->next) {
        uintptr_t start;
        uintptr_t end;

        start = (uintptr_t)runtime_gc_payload(current);
        end = start + current->size;
        if (end < start) {
            end = (uintptr_t)-1;
        }
        if (end < start) {
            end = (uintptr_t)-1;
        }

        if (runtime_gc_heap_min == 0 || start < runtime_gc_heap_min) {
            runtime_gc_heap_min = start;
        }
        if (end > runtime_gc_heap_max) {
            runtime_gc_heap_max = end;
        }
    }
}

static void runtime_gc_index_add(runtime_gc_header* header) {
    uintptr_t start;
    uintptr_t end;
    uintptr_t page;
    uintptr_t last_page;

    if (header == NULL) {
        return;
    }

    start = (uintptr_t)runtime_gc_payload(header);
    end = start + header->size;
    if (end <= start) {
        runtime_gc_page_index_complete = 0;
        return;
    }

    page = runtime_gc_page_base(start);
    last_page = runtime_gc_page_base(end - 1u);
    while (1) {
        runtime_gc_page_entry* entry;
        size_t bucket;

        entry = (runtime_gc_page_entry*)malloc(sizeof(runtime_gc_page_entry));
        if (entry == NULL) {
            runtime_gc_page_index_complete = 0;
            break;
        }

        entry->page_base = page;
        entry->header = header;
        entry->prev_in_bucket = NULL;
        bucket = runtime_gc_page_bucket(page);
        entry->next_in_bucket = runtime_gc_page_buckets[bucket];
        if (entry->next_in_bucket != NULL) {
            entry->next_in_bucket->prev_in_bucket = entry;
        }
        runtime_gc_page_buckets[bucket] = entry;

        entry->next_in_header = header->page_entries;
        header->page_entries = entry;

        if (page == last_page) {
            break;
        }
        page += (uintptr_t)RUNTIME_GC_PAGE_SIZE;
        if (page == 0) {
            break;
        }
    }
}

static void runtime_gc_index_remove(runtime_gc_header* header) {
    runtime_gc_page_entry* entry;

    if (header == NULL) {
        return;
    }

    entry = header->page_entries;
    while (entry != NULL) {
        runtime_gc_page_entry* next;
        size_t bucket;

        next = entry->next_in_header;
        bucket = runtime_gc_page_bucket(entry->page_base);
        if (entry->prev_in_bucket != NULL) {
            entry->prev_in_bucket->next_in_bucket = entry->next_in_bucket;
        } else {
            runtime_gc_page_buckets[bucket] = entry->next_in_bucket;
        }
        if (entry->next_in_bucket != NULL) {
            entry->next_in_bucket->prev_in_bucket = entry->prev_in_bucket;
        }
        free(entry);
        entry = next;
    }

    header->page_entries = NULL;
}

static runtime_gc_header* runtime_gc_index_lookup(const void* address) {
    uintptr_t target;
    uintptr_t page;
    runtime_gc_page_entry* entry;
    size_t bucket;

    if (address == NULL) {
        return NULL;
    }

    target = (uintptr_t)address;
    page = runtime_gc_page_base(target);
    bucket = runtime_gc_page_bucket(page);
    for (entry = runtime_gc_page_buckets[bucket]; entry != NULL; entry = entry->next_in_bucket) {
        uintptr_t start;
        uintptr_t end;

        if (entry->page_base != page) {
            continue;
        }

        start = (uintptr_t)runtime_gc_payload(entry->header);
        end = start + entry->header->size;
        if (end < start) {
            end = (uintptr_t)-1;
        }
        if (target >= start && target < end) {
            return entry->header;
        }
    }

    return NULL;
}

static uintptr_t runtime_gc_min_uintptr(uintptr_t left, uintptr_t right) {
    if (left < right) {
        return left;
    }

    return right;
}

static runtime_gc_header* runtime_gc_find_exact_header(const void* payload) {
    runtime_gc_header* current;

    if (payload == NULL) {
        return NULL;
    }

    for (current = runtime_gc_objects; current != NULL; current = current->next) {
        if (runtime_gc_payload_const(current) == payload) {
            return current;
        }
    }

    return NULL;
}

static void runtime_gc_update_allocation_aux(void* payload, uintptr_t aux) {
    runtime_gc_header* header;

    header = runtime_gc_find_exact_header(payload);
    if (header == NULL) {
        return;
    }

    header->aux = aux;
}

static runtime_gc_header* runtime_gc_find_header_for_address_linear(const void* address) {
    runtime_gc_header* current;
    uintptr_t target;

    if (address == NULL) {
        return NULL;
    }

    target = (uintptr_t)address;
    for (current = runtime_gc_objects; current != NULL; current = current->next) {
        uintptr_t start;
        uintptr_t end;

        start = (uintptr_t)runtime_gc_payload(current);
        end = start + current->size;
        if (target >= start && target < end) {
            return current;
        }
    }

    return NULL;
}

static runtime_gc_header* runtime_gc_find_header_for_address(const void* address) {
    runtime_gc_header* header;
    uintptr_t target;

    if (address == NULL) {
        return NULL;
    }

    target = (uintptr_t)address;
    if (runtime_gc_heap_min == 0 || target < runtime_gc_heap_min || target >= runtime_gc_heap_max) {
        return NULL;
    }

    header = runtime_gc_index_lookup(address);
    if (header != NULL || runtime_gc_page_index_complete) {
        return header;
    }

    return runtime_gc_find_header_for_address_linear(address);
}

static void runtime_gc_link_allocation(runtime_gc_header* header) {
    if (header == NULL) {
        return;
    }

    header->prev = NULL;
    header->next = runtime_gc_objects;
    if (runtime_gc_objects != NULL) {
        runtime_gc_objects->prev = header;
    }
    runtime_gc_objects = header;
    runtime_gc_live_bytes += header->size;
    runtime_gc_live_objects++;
    runtime_gc_update_heap_bounds_on_alloc(header);
    runtime_gc_index_add(header);
}

static void runtime_gc_unlink_allocation(runtime_gc_header* header) {
    if (header == NULL) {
        return;
    }

    runtime_gc_index_remove(header);
    if (header->prev != NULL) {
        header->prev->next = header->next;
    } else {
        runtime_gc_objects = header->next;
    }
    if (header->next != NULL) {
        header->next->prev = header->prev;
    }
    if (runtime_gc_live_bytes >= header->size) {
        runtime_gc_live_bytes -= header->size;
    } else {
        runtime_gc_live_bytes = 0;
    }
    if (runtime_gc_live_objects > 0) {
        runtime_gc_live_objects--;
    }

    if (runtime_gc_heap_min != 0) {
        uintptr_t start;
        uintptr_t end;

        start = (uintptr_t)runtime_gc_payload(header);
        end = start + header->size;
        if (end < start) {
            end = (uintptr_t)-1;
        }

        if (runtime_gc_heap_min == start || runtime_gc_heap_max == end) {
            runtime_gc_recompute_heap_bounds();
        }
    }
}

static void runtime_gc_scan_conservative_words(const void* base, uintptr_t bytes) {
    const uintptr_t* cursor;
    uintptr_t count;
    uintptr_t index;

    if (base == NULL || bytes < sizeof(uintptr_t)) {
        return;
    }

    cursor = (const uintptr_t*)base;
    count = bytes / sizeof(uintptr_t);
    for (index = 0; index < count; index++) {
        runtime_gc_mark_pointer((const void*)cursor[index]);
    }
}

static void runtime_gc_scan_precise_words(const void* base, uintptr_t size, uintptr_t ptrdata, const uint8_t* gcdata) {
    const void* const* words;
    uintptr_t limit;
    uintptr_t word_count;
    uintptr_t index;

    if (base == NULL || size == 0 || ptrdata == 0) {
        return;
    }

    limit = runtime_gc_min_uintptr(size, ptrdata);
    if (limit == 0) {
        return;
    }

    words = (const void* const*)base;
    word_count = limit / sizeof(void*);
    if ((limit % sizeof(void*)) != 0) {
        word_count++;
    }

    for (index = 0; index < word_count; index++) {
        if (gcdata != NULL) {
            uintptr_t byte_index;
            uint8_t mask;

            byte_index = index / 8u;
            mask = (uint8_t)(1u << (index % 8u));
            if ((gcdata[byte_index] & mask) == 0) {
                continue;
            }
        }

        runtime_gc_mark_pointer(words[index]);
    }
}

static void runtime_gc_mark_header(runtime_gc_header* header) {
    if (header == NULL || header->marked == runtime_gc_mark_token) {
        return;
    }

    header->marked = runtime_gc_mark_token;
    if (header->scan != NULL) {
        header->scan(header);
    }
}

static void runtime_gc_mark_pointer(const void* value) {
    runtime_gc_header* header;

    header = runtime_gc_find_header_for_address(value);
    if (header != NULL) {
        runtime_gc_mark_header(header);
    }
}

static void runtime_gc_scan_descriptor_object(runtime_gc_header* header) {
    const go_type_descriptor* descriptor;

    if (header == NULL) {
        return;
    }

    descriptor = header->descriptor;
    if (descriptor == NULL || descriptor->ptrdata == 0) {
        return;
    }

    runtime_gc_scan_precise_words(runtime_gc_payload(header), header->size, descriptor->ptrdata, (const uint8_t*)descriptor->gcdata);
}

static void runtime_gc_scan_descriptor_array(runtime_gc_header* header) {
    const go_type_descriptor* descriptor;
    unsigned char* base;
    uintptr_t index;
    uintptr_t element_size;

    if (header == NULL) {
        return;
    }

    descriptor = header->descriptor;
    if (descriptor == NULL || descriptor->ptrdata == 0 || descriptor->size == 0) {
        return;
    }

    base = (unsigned char*)runtime_gc_payload(header);
    element_size = descriptor->size;
    for (index = 0; index < header->aux; index++) {
        runtime_gc_scan_precise_words(base + index * element_size, element_size, descriptor->ptrdata, (const uint8_t*)descriptor->gcdata);
    }
}

static void runtime_gc_scan_runtime_map(runtime_gc_header* header) {
    runtime_map* map;

    map = (runtime_map*)runtime_gc_payload(header);
    if (map == NULL) {
        return;
    }

    runtime_gc_mark_pointer(map->entries);
    runtime_gc_mark_pointer(map->zero_value);
}

static void runtime_gc_scan_runtime_map_entries(runtime_gc_header* header) {
    runtime_map_entry* entries;
    uintptr_t index;

    entries = (runtime_map_entry*)runtime_gc_payload(header);
    if (entries == NULL) {
        return;
    }

    for (index = 0; index < header->aux; index++) {
        if (entries[index].state != 1) {
            continue;
        }
        runtime_gc_mark_pointer(entries[index].key_data);
        runtime_gc_mark_pointer(entries[index].value_data);
    }
}

static void runtime_gc_scan_runtime_map_iter(runtime_gc_header* header) {
    runtime_map_iter_state* state;

    state = (runtime_map_iter_state*)runtime_gc_payload(header);
    if (state == NULL) {
        return;
    }

    runtime_gc_mark_pointer(state->map);
}

#if defined(__i386__)
#define RUNTIME_GC_REGISTER_SLOTS 8u
__attribute__((noinline)) static void runtime_gc_capture_registers(uintptr_t* out) {
    __asm__ __volatile__(
        "movl %%eax, 0(%0)\n\t"
        "movl %%ebx, 4(%0)\n\t"
        "movl %%ecx, 8(%0)\n\t"
        "movl %%edx, 12(%0)\n\t"
        "movl %%esi, 16(%0)\n\t"
        "movl %%edi, 20(%0)\n\t"
        "movl %%ebp, 24(%0)\n\t"
        "movl %%esp, 28(%0)\n\t"
        :
        : "r"(out)
        : "memory");
}

__attribute__((noinline)) static void runtime_gc_scrub_registers(void) {
    __asm__ __volatile__(
        "xorl %%eax, %%eax\n\t"
        "xorl %%ebx, %%ebx\n\t"
        "xorl %%ecx, %%ecx\n\t"
        "xorl %%edx, %%edx\n\t"
        "xorl %%esi, %%esi\n\t"
        "xorl %%edi, %%edi\n\t"
        :
        :
        : "eax", "ebx", "ecx", "edx", "esi", "edi", "memory");
}
#elif defined(__x86_64__)
#define RUNTIME_GC_REGISTER_SLOTS 16u
__attribute__((noinline)) static void runtime_gc_capture_registers(uintptr_t* out) {
    __asm__ __volatile__(
        "movq %%rax, 0(%0)\n\t"
        "movq %%rbx, 8(%0)\n\t"
        "movq %%rcx, 16(%0)\n\t"
        "movq %%rdx, 24(%0)\n\t"
        "movq %%rsi, 32(%0)\n\t"
        "movq %%rdi, 40(%0)\n\t"
        "movq %%rbp, 48(%0)\n\t"
        "movq %%rsp, 56(%0)\n\t"
        "movq %%r8, 64(%0)\n\t"
        "movq %%r9, 72(%0)\n\t"
        "movq %%r10, 80(%0)\n\t"
        "movq %%r11, 88(%0)\n\t"
        "movq %%r12, 96(%0)\n\t"
        "movq %%r13, 104(%0)\n\t"
        "movq %%r14, 112(%0)\n\t"
        "movq %%r15, 120(%0)\n\t"
        :
        : "r"(out)
        : "memory");
}

__attribute__((noinline)) static void runtime_gc_scrub_registers(void) {
    __asm__ __volatile__(
        "xorq %%rax, %%rax\n\t"
        "xorq %%rbx, %%rbx\n\t"
        "xorq %%rcx, %%rcx\n\t"
        "xorq %%rdx, %%rdx\n\t"
        "xorq %%rsi, %%rsi\n\t"
        "xorq %%rdi, %%rdi\n\t"
        "xorq %%r8, %%r8\n\t"
        "xorq %%r9, %%r9\n\t"
        "xorq %%r10, %%r10\n\t"
        "xorq %%r11, %%r11\n\t"
        "xorq %%r12, %%r12\n\t"
        "xorq %%r13, %%r13\n\t"
        "xorq %%r14, %%r14\n\t"
        "xorq %%r15, %%r15\n\t"
        :
        :
        : "rax", "rbx", "rcx", "rdx", "rsi", "rdi", "r8", "r9", "r10", "r11", "r12", "r13", "r14", "r15", "memory");
}
#else
#define RUNTIME_GC_REGISTER_SLOTS 1u
__attribute__((noinline)) static void runtime_gc_capture_registers(uintptr_t* out) {
    out[0] = 0;
}

__attribute__((noinline)) static void runtime_gc_scrub_registers(void) {
}
#endif

static void runtime_gc_mark_registered_roots(void) {
    runtime_gc_root_block* block;
    uintptr_t index;

    for (block = runtime_gc_roots; block != NULL; block = block->next) {
        for (index = 0; index < block->count; index++) {
            runtime_gc_root_descriptor* root;

            root = block->roots + index;
            runtime_gc_scan_precise_words(root->addr, root->size, root->ptrdata, root->gcdata);
        }
    }
}

static void runtime_gc_mark_defers(void) {
    runtime_g* g;
    for (g = runtime_allg; g != NULL; g = g->all_next) {
        runtime_defer* current;
        for (current = g->_defer; current != NULL; current = current->link) {
            runtime_gc_mark_pointer(current);
            runtime_gc_mark_pointer(current->arg);
        }
    }
}

static void runtime_gc_clamp_stack_range(runtime_g* g, uintptr_t* start, uintptr_t* end) {
    uintptr_t low = 0;
    uintptr_t high = 0;

    if (g == &runtime_g0) {
        low = (uintptr_t)&__end;
        high = (uintptr_t)&__memory_top;
    } else if (g->stack_base != NULL && g->stack_top != 0) {
        low = (uintptr_t)g->stack_base;
        high = (uintptr_t)g->stack_top;
    }

    if (low == 0 || high == 0) {
        return;
    }

    if (*start < low) {
        *start = low;
    }
    if (*end > high) {
        *end = high;
    }
}

__attribute__((noinline)) static void runtime_gc_mark_roots_and_stack(void) {
    uintptr_t registers[RUNTIME_GC_REGISTER_SLOTS];
    runtime_g* g;

    runtime_gc_capture_registers(registers);
    runtime_gc_scan_conservative_words(registers, sizeof(registers));
    runtime_gc_mark_registered_roots();
    runtime_gc_mark_defers();
    runtime_g* current = runtime_getg();
    for (g = runtime_allg; g != NULL; g = g->all_next) {
        uintptr_t marker;
        uintptr_t start;
        uintptr_t end;

        if (g->status == RUNTIME_G_DEAD) {
            continue;
        }

        if (g == current) {
            marker = 0;
            start = (uintptr_t)&marker;
            end = (uintptr_t)g->stack_top;
        } else {
            start = (uintptr_t)g->context.esp;
            end = (uintptr_t)g->stack_top;
        }

        if (end < start) {
            uintptr_t swap;

            swap = start;
            start = end;
            end = swap;
        }
        runtime_gc_clamp_stack_range(g, &start, &end);
        if (end == 0 || start == 0) {
            continue;
        }
        if (end >= sizeof(uintptr_t)) {
            end -= sizeof(uintptr_t);
        } else {
            continue;
        }
        if (end < start) {
            continue;
        }
        if (end >= start) {
            runtime_gc_scan_conservative_words((const void*)start, end - start + sizeof(uintptr_t));
        }

        if (g->entry_arg != NULL) {
            runtime_gc_mark_pointer(g->entry_arg);
        }
        if (g->_panic != NULL) {
            runtime_gc_mark_pointer(g->_panic);
        }
    }
}

static void runtime_gc_update_threshold(void) {
    size_t base;

    base = runtime_gc_live_bytes;
    if (base < 32768u) {
        base = 32768u;
    }
    if (base > ((size_t)-1) - (base / 2u) - 4096u) {
        runtime_gc_threshold = (size_t)-1;
        return;
    }

    runtime_gc_threshold = base + (base / 2u) + 4096u;
}

static void runtime_gc_collect_impl_locked(void) {
    runtime_gc_header* current;
    runtime_gc_header* next;

    if (!runtime_gc_enabled || runtime_gc_running) {
        return;
    }

    runtime_stop_world();
    runtime_gc_running = 1;
    runtime_gc_mark_token = (runtime_gc_mark_token == 1u) ? 2u : 1u;

    runtime_gc_mark_roots_and_stack();
    current = runtime_gc_objects;
    while (current != NULL) {
        next = current->next;
        if (current->marked != runtime_gc_mark_token) {
            runtime_gc_unlink_allocation(current);
            runtime_pool_free(current);
        }
        current = next;
    }

    runtime_gc_collection_count++;
    runtime_gc_running = 0;
    runtime_gc_update_threshold();
    runtime_start_world();
}

static void runtime_gc_collect_impl(void) {
    runtime_lock_mutex(&runtime_gc_lock);
    runtime_gc_collect_impl_locked();
    runtime_unlock_mutex(&runtime_gc_lock);
}

static void runtime_gc_maybe_collect_locked(size_t requested_size) {
    if (!runtime_gc_enabled || runtime_gc_running) {
        return;
    }
    if (requested_size == 0) {
        requested_size = 1;
    }
    if (runtime_gc_low_mem_kb != 0) {
        runtime_gc_memcheck_counter++;
        if (runtime_gc_memcheck_counter >= 256u) {
            uint32_t free_kb;

            runtime_gc_memcheck_counter = 0;
            free_kb = runtime_kos_get_free_ram_raw();
            if (free_kb != 0 && free_kb < runtime_gc_low_mem_kb) {
                runtime_gc_poll_retry = 1;
                runtime_gc_collect_impl_locked();
            }
        }
    }
    if (requested_size > (size_t)-1 - runtime_gc_live_bytes || runtime_gc_live_bytes + requested_size >= runtime_gc_threshold) {
        runtime_gc_poll_retry = 1;
        runtime_gc_collect_impl_locked();
    }
}

static void* runtime_gc_alloc_managed(size_t size, const go_type_descriptor* descriptor, runtime_gc_scan_fn scan, void* aux, uintptr_t count) {
    runtime_gc_header* header;
    size_t payload_size;
    void* payload;

    (void)aux;

    payload_size = size == 0 ? 1u : size;
    if (payload_size > (size_t)-1 - sizeof(runtime_gc_header)) {
        runtime_panicmem();
    }

    runtime_lock_mutex(&runtime_gc_lock);
    runtime_gc_maybe_collect_locked(payload_size);
    header = (runtime_gc_header*)runtime_pool_malloc(sizeof(runtime_gc_header) + payload_size);
    if (header == NULL) {
        runtime_unlock_mutex(&runtime_gc_lock);
        return NULL;
    }
    runtime_gc_alloc_count++;
    runtime_gc_alloc_bytes += payload_size;

    header->descriptor = descriptor;
    header->scan = scan;
    header->size = (uintptr_t)payload_size;
    header->aux = count;
    header->page_entries = NULL;
    header->marked = 0;
    runtime_gc_link_allocation(header);

    payload = runtime_gc_payload(header);
    kos_memset(payload, 0, payload_size);
    runtime_unlock_mutex(&runtime_gc_lock);
    return payload;
}

static void* runtime_gc_alloc_object(const go_type_descriptor* descriptor) {
    size_t size;
    runtime_gc_scan_fn scan;

    size = 0;
    if (descriptor != NULL) {
        size = (size_t)descriptor->size;
    }
    scan = NULL;
    if (descriptor != NULL && descriptor->ptrdata != 0 && descriptor->size != 0) {
        scan = runtime_gc_scan_descriptor_object;
    }

    return runtime_gc_alloc_managed(size, descriptor, scan, NULL, 0);
}

static void* runtime_gc_alloc_array(const go_type_descriptor* descriptor, intptr_t count, size_t total_size) {
    runtime_gc_scan_fn scan;

    scan = NULL;
    if (descriptor != NULL && descriptor->ptrdata != 0 && descriptor->size != 0 && count > 0) {
        scan = runtime_gc_scan_descriptor_array;
    }

    return runtime_gc_alloc_managed(total_size, descriptor, scan, NULL, count > 0 ? (uintptr_t)count : 0);
}

static runtime_map* runtime_gc_alloc_map_object(void) {
    return (runtime_map*)runtime_gc_alloc_managed(sizeof(runtime_map), NULL, runtime_gc_scan_runtime_map, NULL, 0);
}

static runtime_map_entry* runtime_gc_alloc_map_entries(runtime_map* map, intptr_t cap) {
    if (cap <= 0) {
        return NULL;
    }

    return (runtime_map_entry*)runtime_gc_alloc_managed(
        (size_t)cap * sizeof(runtime_map_entry),
        NULL,
        runtime_gc_scan_runtime_map_entries,
        NULL,
        (uintptr_t)cap);
}

static runtime_map_iter_state* runtime_gc_alloc_map_iter_state(void) {
    return (runtime_map_iter_state*)runtime_gc_alloc_managed(sizeof(runtime_map_iter_state), NULL, runtime_gc_scan_runtime_map_iter, NULL, 0);
}

static void runtime_gc_free_exact(void* ptr) {
    runtime_gc_header* header;

    if (ptr == NULL) {
        return;
    }

    runtime_lock_mutex(&runtime_gc_lock);
    header = runtime_gc_find_exact_header(ptr);
    if (header == NULL) {
        runtime_unlock_mutex(&runtime_gc_lock);
        free(ptr);
        return;
    }

    runtime_gc_unlink_allocation(header);
    runtime_pool_free(header);
    runtime_gc_update_threshold();
    runtime_unlock_mutex(&runtime_gc_lock);
}

void runtime_force_gc(void) {
    runtime_lock_mutex(&runtime_gc_lock);
    runtime_gc_collect_impl_locked();
    runtime_gc_scrub_registers();
    runtime_gc_collect_impl_locked();
    runtime_gc_poll_retry = 0;
    runtime_unlock_mutex(&runtime_gc_lock);
}

void runtime_gc_poll(void) {
    runtime_lock_mutex(&runtime_gc_lock);
    if (!runtime_gc_enabled || runtime_gc_running) {
        runtime_unlock_mutex(&runtime_gc_lock);
        return;
    }

    runtime_gc_collect_impl_locked();
    runtime_gc_scrub_registers();
    runtime_gc_collect_impl_locked();
    runtime_gc_poll_retry = runtime_gc_live_bytes >= runtime_gc_threshold;
    runtime_unlock_mutex(&runtime_gc_lock);
}

uint32_t runtime_kolibri_set_threads(uint32_t count) __asm__("runtime_kolibri_set_threads");
uint32_t runtime_kolibri_set_threads(uint32_t count) {
    if (count == 0) {
        count = 1;
    }
    if (count > RUNTIME_MAX_THREAD_SLOTS) {
        count = RUNTIME_MAX_THREAD_SLOTS;
    }
    runtime_lock_mutex(&runtime_m_lock);
    runtime_max_threads = count;
    runtime_unlock_mutex(&runtime_m_lock);

    if (runtime_started) {
        for (;;) {
            uint32_t current;

            runtime_lock_mutex(&runtime_m_lock);
            current = runtime_m_count + runtime_m_pending;
            runtime_unlock_mutex(&runtime_m_lock);

            if (current >= runtime_max_threads) {
                break;
            }
            if (!runtime_spawn_m()) {
                break;
            }
        }
        /* Give newly created threads a moment to enter the scheduler. */
        for (uint32_t spins = 10000; spins > 0; spins--) {
            uint32_t pending;
            uint32_t current;

            runtime_lock_mutex(&runtime_m_lock);
            pending = runtime_m_pending;
            current = runtime_m_count;
            runtime_unlock_mutex(&runtime_m_lock);

            if (pending == 0 || current >= runtime_max_threads) {
                break;
            }
            runtime_yield();
        }
    }
    return runtime_max_threads;
}

uint32_t runtime_kolibri_get_threads(void) __asm__("runtime_kolibri_get_threads");
uint32_t runtime_kolibri_get_threads(void) {
    return runtime_max_threads;
}

size_t runtime_gc_live_object_count(void) {
    return runtime_gc_live_objects;
}

size_t runtime_gc_live_bytes_count(void) {
    return runtime_gc_live_bytes;
}

static void runtime_unwind_stack(void);
void runtime_rethrowException(void) __asm__("runtime.rethrowException");
uintptr_t runtime_unwindExceptionSize(void) __asm__("runtime.unwindExceptionSize");
void runtime_throwException(void) __asm__("runtime.throwException");

static void runtime_freedefer(runtime_defer* d) {
    runtime_gc_header* header;

    if (d == NULL) {
        return;
    }
    if (!d->heap) {
        return;
    }
    header = runtime_gc_find_header_for_address(d);
    if (header != NULL) {
        runtime_gc_free_exact(d);
    }
}

void runtime_deferprocStack(runtime_defer* d, uint8_t* frame, runtime_defer_fn fn, void* arg) {
    runtime_g* g = runtime_getg();

    if (d == NULL || g == NULL) {
        runtime_panicmem();
    }

    d->pfn = (uintptr_t)fn;
    d->retaddr = 0;
    d->makefunccanrecover = 0;
    d->heap = 0;
    d->frame = frame;
    d->arg = arg;
    d->panic = NULL;
    d->panic_stack = g->_panic;
    d->link = g->_defer;
    g->_defer = d;
}

void runtime_deferproc(uint8_t* frame, runtime_defer_fn fn, void* arg) {
    runtime_defer* d;

    d = (runtime_defer*)runtime_gc_alloc_managed(sizeof(runtime_defer), NULL, NULL, NULL, 0);
    if (d == NULL) {
        runtime_panicmem();
    }
    d->heap = 1;

    runtime_deferprocStack(d, frame, fn, arg);
}

void runtime_deferreturn(uint8_t* frame) {
    runtime_g* g = runtime_getg();
    if (frame == NULL) {
        return;
    }
    if (g == NULL) {
        return;
    }

    while (g->_defer != NULL && g->_defer->frame == frame) {
        runtime_defer* current = g->_defer;
        uintptr_t pfn = current->pfn;

        current->pfn = 0;

        if (pfn != 0) {
            runtime_defer_fn fn = (runtime_defer_fn)(uintptr_t)pfn;
            g->deferring = 1;
            fn(current->arg);
            g->deferring = 0;
        }

        g->_defer = current->link;
        runtime_freedefer(current);
    }

    *frame = 1;
}

void runtime_checkdefer(uint8_t* frame) {
    runtime_g* g = runtime_getg();
    runtime_defer* d;

    if (g == NULL) {
        runtime_fail_simple("no g in checkdefer");
    }

    d = g->_defer;
    if (d != NULL && d->pfn == 0 && d->frame == frame) {
        g->_defer = d->link;
        runtime_freedefer(d);
        if (frame != NULL) {
            *frame = 1;
        }
        return;
    }

    runtime_rethrowException();
    runtime_fail_simple("rethrowException returned");
}

bool runtime_canrecover(void* retaddr) {
    runtime_g* g = runtime_getg();
    runtime_defer* d;
    uintptr_t ret;

    if (g == NULL) {
        return false;
    }
    d = g->_defer;
    if (d == NULL) {
        return false;
    }
    if (d->panic_stack == g->_panic) {
        return false;
    }
    if (d->retaddr == 0) {
        /* Allow recover during active defers even if retaddr wasn't set. */
        if (g->deferring && d->panic == g->_panic) {
            return true;
        }
        return false;
    }
    ret = (uintptr_t)__builtin_extract_return_addr(retaddr);
    if (ret <= d->retaddr && ret + 16 >= d->retaddr) {
        return true;
    }
    /* Fallback: allow recover when we cannot match return addresses. */
    return true;
}

bool runtime_setdeferretaddr(void* retaddr) {
    runtime_g* g = runtime_getg();

    if (g != NULL && g->_defer != NULL) {
        /* Match libgo: store the raw label/return address as provided. */
        g->_defer->retaddr = (uintptr_t)retaddr;
    }
    return false;
}

void runtime_gorecover(go_empty_interface* out) {
    runtime_g* g;
    runtime_panic* p;

    if (out == NULL) {
        return;
    }

    out->type = NULL;
    out->data = NULL;

    g = runtime_getg();
    if (g == NULL) {
        return;
    }
    p = g->_panic;
    if (p == NULL) {
        return;
    }
    if (p->goexit || p->recovered) {
        return;
    }
    p->recovered = 1;
    *out = p->arg;
}

uint32_t runtime_bootstrap_has_gc(void) {
    return 1u;
}

static bool RUNTIME_USED runtime_memequal0_impl(const void* left, const void* right) {
    (void)left;
    (void)right;
    return true;
}

static bool RUNTIME_USED runtime_memequal8_impl(const void* left, const void* right) {
    const unsigned char* left_bytes;
    const unsigned char* right_bytes;

    if (left == NULL || right == NULL) {
        return false;
    }

    left_bytes = (const unsigned char*)left;
    right_bytes = (const unsigned char*)right;
    return left_bytes[0] == right_bytes[0];
}

static bool RUNTIME_USED runtime_memequal16_impl(const void* left, const void* right) {
    const uint16_t* left_words;
    const uint16_t* right_words;

    if (left == NULL || right == NULL) {
        return false;
    }

    left_words = (const uint16_t*)left;
    right_words = (const uint16_t*)right;
    return left_words[0] == right_words[0];
}

static bool RUNTIME_USED runtime_memequal32_impl(const void* left, const void* right) {
    const uint32_t* left_words;
    const uint32_t* right_words;

    if (left == NULL || right == NULL) {
        return false;
    }

    left_words = (const uint32_t*)left;
    right_words = (const uint32_t*)right;
    return left_words[0] == right_words[0];
}

static bool RUNTIME_USED runtime_memequal64_impl(const void* left, const void* right) {
    const uint32_t* left_words;
    const uint32_t* right_words;

    if (left == NULL || right == NULL) {
        return false;
    }

    left_words = (const uint32_t*)left;
    right_words = (const uint32_t*)right;
    return left_words[0] == right_words[0] &&
           left_words[1] == right_words[1];
}

static bool RUNTIME_USED runtime_memequal128_impl(const void* left, const void* right) {
    const uint32_t* left_words;
    const uint32_t* right_words;

    if (left == NULL || right == NULL) {
        return false;
    }

    left_words = (const uint32_t*)left;
    right_words = (const uint32_t*)right;
    return left_words[0] == right_words[0] &&
           left_words[1] == right_words[1] &&
           left_words[2] == right_words[2] &&
           left_words[3] == right_words[3];
}

static const char* runtime_prepare_window_title_impl(uint32_t prefix, int use_prefix, const char* src, intptr_t len) {
    char* resized;
    size_t needed;
    size_t offset;

    if (src == NULL) {
        return NULL;
    }

    if (len < 0) {
        len = 0;
    }

    offset = use_prefix ? 1u : 0u;
    needed = offset + (size_t)len + 1;
    if (runtime_window_title_buffer == NULL || needed > runtime_window_title_capacity) {
        resized = (char*)realloc(runtime_window_title_buffer, needed);
        if (resized == NULL) {
            return runtime_window_title_buffer;
        }

        runtime_window_title_buffer = resized;
        runtime_window_title_capacity = needed;
    }

    if (use_prefix) {
        runtime_window_title_buffer[0] = (char)prefix;
    }

    if (len > 0) {
        kos_memcpy(runtime_window_title_buffer + offset, src, (size_t)len);
    }
    runtime_window_title_buffer[offset + (size_t)len] = '\0';

    return runtime_window_title_buffer;
}

const char* runtime_prepare_window_title(const char* src, intptr_t len) {
    return runtime_prepare_window_title_impl(0, 0, src, len);
}

const char* runtime_prepare_window_title_with_prefix(uint32_t prefix, const char* src, intptr_t len) {
    return runtime_prepare_window_title_impl(prefix, 1, src, len);
}

char* runtime_alloc_cstring(const char* src, intptr_t len) {
    char* out;

    if (src == NULL) {
        return NULL;
    }

    if (len < 0) {
        len = 0;
    }

    out = (char*)runtime_gc_alloc_managed((size_t)len + 1, NULL, NULL, NULL, 0);
    if (out == NULL) {
        return NULL;
    }

    if (len > 0) {
        kos_memcpy(out, src, (size_t)len);
    }
    out[len] = '\0';

    return out;
}

void runtime_free_cstring(void* ptr) {
    if (ptr != NULL) {
        runtime_gc_free_exact(ptr);
    }
}

uint32_t runtime_pointer_value(void* ptr) {
    return (uint32_t)(uintptr_t)ptr;
}

go_string runtime_cstring_to_gostring(uint32_t ptr_addr) {
    const char* src;
    intptr_t len;
    char* out;
    go_string result;

    src = (const char*)(uintptr_t)ptr_addr;
    if (src == NULL) {
        result.str = NULL;
        result.len = 0;
        return result;
    }

    len = (intptr_t)kos_strlen(src);
    out = (char*)runtime_gc_alloc_managed((size_t)len + 1, NULL, NULL, NULL, 0);
    if (out == NULL) {
        result.str = NULL;
        result.len = 0;
        return result;
    }

    if (len > 0) {
        kos_memcpy(out, src, (size_t)len);
    }
    out[len] = '\0';
    result.str = out;
    result.len = len;
    return result;
}

go_slice runtime_copy_bytes(uint32_t ptr_addr, uint32_t size) {
    go_slice result;
    unsigned char* out;

    result.values = NULL;
    result.len = 0;
    result.cap = 0;
    if (ptr_addr == 0 || size == 0) {
        return result;
    }

    out = (unsigned char*)runtime_gc_alloc_managed((size_t)size, NULL, NULL, NULL, 0);
    if (out == NULL) {
        return result;
    }

    kos_memcpy(out, (const void*)(uintptr_t)ptr_addr, (size_t)size);
    result.values = out;
    result.len = (intptr_t)size;
    result.cap = (intptr_t)size;
    return result;
}

uint32_t runtime_read_u32(uint32_t base, uint32_t offset) {
    if (base == 0) {
        return 0;
    }

    return *(const uint32_t*)(uintptr_t)(base + offset);
}

typedef struct {
    const char* name;
    void* data;
} kos_dll_export;

#if defined(__i386__)
#define KOS_STDCALL __attribute__((stdcall))
#else
#define KOS_STDCALL
#endif

typedef uint32_t (KOS_STDCALL *kos_stdcall0_fn)(void);
typedef uint32_t (KOS_STDCALL *kos_stdcall1_fn)(uint32_t arg0);
typedef uint32_t (KOS_STDCALL *kos_stdcall2_fn)(uint32_t arg0, uint32_t arg1);
typedef uint32_t (KOS_STDCALL *kos_stdcall3_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2);
typedef uint32_t (KOS_STDCALL *kos_stdcall4_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3);
typedef uint32_t (KOS_STDCALL *kos_stdcall5_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4);
typedef uint32_t (KOS_STDCALL *kos_stdcall6_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4, uint32_t arg5);
typedef uint32_t (KOS_STDCALL *kos_stdcall7_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4, uint32_t arg5, uint32_t arg6);
typedef void (KOS_STDCALL *kos_stdcall1_void_fn)(uint32_t arg0);
typedef void (KOS_STDCALL *kos_stdcall2_void_fn)(uint32_t arg0, uint32_t arg1);
typedef void (KOS_STDCALL *kos_stdcall5_void_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4);
typedef uint32_t (*kos_cdecl2_fn)(uint32_t arg0, uint32_t arg1);
typedef uint32_t (*kos_cdecl5_fn)(uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4);

typedef struct {
    uint32_t imports;
    const char* library_name;
} kos_dll_import_library;

static uint32_t runtime_console_bridge_table = 0;
static uint32_t runtime_console_bridge_write_proc = 0;
static uint32_t runtime_console_bridge_exit_proc = 0;
static uint32_t runtime_console_bridge_gets_proc = 0;
static uint32_t runtime_kos_heap_initialized = 0;

static void runtime_kos_dialog_noop(void) {
}

uint32_t KOS_STDCALL runtime_kos_dll_load_imports(uint32_t import_table_addr);

uint32_t runtime_kos_lookup_dll_export(uint32_t table_addr, const char* name) {
    const kos_dll_export* cursor;

    if (table_addr == 0 || name == NULL) {
        return 0;
    }

    cursor = (const kos_dll_export*)(uintptr_t)table_addr;
    while (cursor->name != NULL) {
        if (kos_strcmp(cursor->name, name) == 0) {
            return (uint32_t)(uintptr_t)cursor->data;
        }
        cursor++;
    }

    return 0;
}

static int runtime_kos_ensure_heap(void) {
    if (runtime_kos_heap_initialized != 0) {
        return 1;
    }

    if (runtime_kos_heap_init_raw() == 0) {
        return 0;
    }

    runtime_kos_heap_initialized = 1;
    return 1;
}

static uint32_t KOS_STDCALL runtime_kos_dll_mem_alloc(uint32_t size) {
    if (!runtime_kos_ensure_heap()) {
        return 0;
    }

    uint32_t result = runtime_kos_heap_alloc_raw(size);
    if (result != 0) {
        runtime_kos_heap_alloc_count++;
        runtime_kos_heap_alloc_bytes += size;
    }
    return result;
}

static uint32_t KOS_STDCALL runtime_kos_dll_mem_free(uint32_t ptr) {
    if (ptr == 0) {
        return 1;
    }
    if (!runtime_kos_ensure_heap()) {
        return 0;
    }

    uint32_t result = runtime_kos_heap_free_raw(ptr);
    if (result != 0) {
        runtime_kos_heap_free_count++;
    }
    return result;
}

static uint32_t KOS_STDCALL runtime_kos_dll_mem_realloc(uint32_t ptr, uint32_t size) {
    if (!runtime_kos_ensure_heap()) {
        return 0;
    }

    uint32_t result = runtime_kos_heap_realloc_raw(size, ptr);
    if (result != 0) {
        runtime_kos_heap_realloc_count++;
        runtime_kos_heap_realloc_bytes += size;
    }
    return result;
}

static uint32_t runtime_kos_load_named_dll(const char* name) {
    static const char prefix[] = "/sys/lib/";
    char path[256];
    size_t prefix_len;
    size_t name_len;

    if (name == NULL || name[0] == 0) {
        return 0;
    }

    if (name[0] == '/') {
        return runtime_kos_load_dll_cstring_raw(name);
    }

    prefix_len = sizeof(prefix) - 1;
    name_len = kos_strlen(name);
    if (prefix_len + name_len + 1 > sizeof(path)) {
        return 0;
    }

    kos_memcpy(path, prefix, prefix_len);
    kos_memcpy(path + prefix_len, name, name_len + 1);
    return runtime_kos_load_dll_cstring_raw(path);
}

static int runtime_kos_link_dll_imports(uint32_t table_addr, uint32_t imports_addr) {
    uint32_t* cursor;

    if (table_addr == 0 || imports_addr == 0) {
        return 0;
    }

    cursor = (uint32_t*)(uintptr_t)imports_addr;
    while (*cursor != 0) {
        uint32_t proc = runtime_kos_lookup_dll_export(table_addr, (const char*)(uintptr_t)(*cursor));
        if (proc == 0) {
            return 0;
        }
        *cursor = proc;
        cursor++;
    }

    return 1;
}

static uint32_t runtime_kos_dll_lib_init_proc(uint32_t table_addr) {
    const kos_dll_export* exports;

    if (table_addr == 0) {
        return 0;
    }

    exports = (const kos_dll_export*)(uintptr_t)table_addr;
    if (exports->name == NULL) {
        return 0;
    }
    if (exports->name[0] == 'l' &&
        exports->name[1] == 'i' &&
        exports->name[2] == 'b' &&
        exports->name[3] == '_') {
        return (uint32_t)(uintptr_t)exports->data;
    }

    return 0;
}

uint32_t runtime_kos_call_stdcall3(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall3_fn)(uintptr_t)proc)(arg0, arg1, arg2);
}

uint32_t runtime_kos_call_stdcall4(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall4_fn)(uintptr_t)proc)(arg0, arg1, arg2, arg3);
}

uint32_t runtime_kos_heap_alloc_count_read(void) {
    return (uint32_t)runtime_kos_heap_alloc_count;
}

uint32_t runtime_kos_heap_alloc_bytes_read(void) {
    return (uint32_t)runtime_kos_heap_alloc_bytes;
}

uint32_t runtime_kos_heap_free_count_read(void) {
    return (uint32_t)runtime_kos_heap_free_count;
}

uint32_t runtime_kos_heap_realloc_count_read(void) {
    return (uint32_t)runtime_kos_heap_realloc_count;
}

uint32_t runtime_kos_heap_realloc_bytes_read(void) {
    return (uint32_t)runtime_kos_heap_realloc_bytes;
}

uint32_t runtime_gc_alloc_count_read(void) {
    return (uint32_t)runtime_gc_alloc_count;
}

uint32_t runtime_gc_alloc_bytes_read(void) {
    return (uint32_t)runtime_gc_alloc_bytes;
}

uint32_t runtime_gc_live_bytes_read(void) {
    return (uint32_t)runtime_gc_live_bytes;
}

uint32_t runtime_gc_threshold_read(void) {
    return (uint32_t)runtime_gc_threshold;
}

uint32_t runtime_gc_collection_count_read(void) {
    return (uint32_t)runtime_gc_collection_count;
}

uint32_t runtime_gc_poll_retry_read(void) {
    return (uint32_t)runtime_gc_poll_retry;
}

uint32_t runtime_kos_call_stdcall5(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall5_fn)(uintptr_t)proc)(arg0, arg1, arg2, arg3, arg4);
}

uint32_t runtime_kos_call_stdcall6(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4, uint32_t arg5) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall6_fn)(uintptr_t)proc)(arg0, arg1, arg2, arg3, arg4, arg5);
}

uint32_t runtime_kos_call_stdcall7(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4, uint32_t arg5, uint32_t arg6) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall7_fn)(uintptr_t)proc)(arg0, arg1, arg2, arg3, arg4, arg5, arg6);
}

uint32_t runtime_kos_call_cdecl2(uint32_t proc, uint32_t arg0, uint32_t arg1) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_cdecl2_fn)(uintptr_t)proc)(arg0, arg1);
}

uint32_t runtime_kos_call_cdecl5(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_cdecl5_fn)(uintptr_t)proc)(arg0, arg1, arg2, arg3, arg4);
}

uint32_t runtime_kos_call_stdcall0(uint32_t proc) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall0_fn)(uintptr_t)proc)();
}

uint32_t runtime_kos_dialog_noop_addr(void) {
    return (uint32_t)(uintptr_t)&runtime_kos_dialog_noop;
}

uint32_t runtime_kos_call_stdcall1(uint32_t proc, uint32_t arg0) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall1_fn)(uintptr_t)proc)(arg0);
}

uint32_t runtime_kos_call_stdcall2(uint32_t proc, uint32_t arg0, uint32_t arg1) {
    if (proc == 0) {
        return 0;
    }

    return ((kos_stdcall2_fn)(uintptr_t)proc)(arg0, arg1);
}

void runtime_kos_call_stdcall1_void(uint32_t proc, uint32_t arg0) {
    if (proc == 0) {
        return;
    }

    ((kos_stdcall1_void_fn)(uintptr_t)proc)(arg0);
}

void runtime_kos_call_stdcall2_void(uint32_t proc, uint32_t arg0, uint32_t arg1) {
    if (proc == 0) {
        return;
    }

    ((kos_stdcall2_void_fn)(uintptr_t)proc)(arg0, arg1);
}

void runtime_kos_call_stdcall5_void(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4) {
    if (proc == 0) {
        return;
    }

    ((kos_stdcall5_void_fn)(uintptr_t)proc)(arg0, arg1, arg2, arg3, arg4);
}

uint32_t runtime_kos_init_dll_library(uint32_t proc) {
    if (proc == 0) {
        return 1;
    }

#if defined(__i386__)
    {
        uint32_t alloc_proc = (uint32_t)(uintptr_t)runtime_kos_dll_mem_alloc;
        uint32_t free_proc = (uint32_t)(uintptr_t)runtime_kos_dll_mem_free;
        uint32_t realloc_proc = (uint32_t)(uintptr_t)runtime_kos_dll_mem_realloc;
        uint32_t load_proc = (uint32_t)(uintptr_t)runtime_kos_dll_load_imports;

        __asm__ volatile (
            "call *%[init_proc]\n\t"
            : "+a" (alloc_proc),
              "+b" (free_proc),
              "+c" (realloc_proc),
              "+d" (load_proc)
            : [init_proc] "m" (proc)
            : "memory", "cc", "esi", "edi"
        );

        return alloc_proc;
    }
#else
    return runtime_kos_call_stdcall4(
        proc,
        (uint32_t)(uintptr_t)runtime_kos_dll_mem_alloc,
        (uint32_t)(uintptr_t)runtime_kos_dll_mem_free,
        (uint32_t)(uintptr_t)runtime_kos_dll_mem_realloc,
        (uint32_t)(uintptr_t)runtime_kos_dll_load_imports
    );
#endif
}

uint32_t KOS_STDCALL runtime_kos_dll_load_imports(uint32_t import_table_addr) {
    const kos_dll_import_library* cursor;
    uint32_t dll_table;
    uint32_t dll_load_proc;

    {
        static const char dll_loader_path[] = "/sys/lib/dll.obj";

        dll_table = runtime_kos_load_dll_cstring_raw(dll_loader_path);
        dll_load_proc = 0;
        if (dll_table != 0) {
            dll_load_proc = runtime_kos_lookup_dll_export(dll_table, "dll_load");
        }
    }
    if (dll_load_proc != 0) {
        return runtime_kos_call_stdcall1(dll_load_proc, import_table_addr);
    }

    cursor = (const kos_dll_import_library*)(uintptr_t)import_table_addr;
    if (cursor == NULL) {
        return 1;
    }

    while (cursor->imports != 0) {
        uint32_t table_addr;
        uint32_t init_proc;

        table_addr = runtime_kos_load_named_dll(cursor->library_name);
        if (table_addr == 0) {
            return 1;
        }
        if (!runtime_kos_link_dll_imports(table_addr, cursor->imports)) {
            return 1;
        }

        init_proc = runtime_kos_dll_lib_init_proc(table_addr);
        if (init_proc != 0 && runtime_kos_init_dll_library(init_proc) != 0) {
            return 1;
        }

        cursor++;
    }

    return 0;
}

int runtime_console_bridge_ready(void) {
    return runtime_console_bridge_write_proc != 0;
}

void runtime_console_bridge_set(uint32_t table, uint32_t write_proc, uint32_t exit_proc, uint32_t gets_proc) {
    runtime_console_bridge_table = table;
    runtime_console_bridge_write_proc = write_proc;
    runtime_console_bridge_exit_proc = exit_proc;
    runtime_console_bridge_gets_proc = gets_proc;
}

void runtime_console_bridge_clear(uint32_t table) {
    if (runtime_console_bridge_table == table) {
        runtime_console_bridge_table = 0;
        runtime_console_bridge_write_proc = 0;
        runtime_console_bridge_exit_proc = 0;
        runtime_console_bridge_gets_proc = 0;
    }
}

int runtime_console_bridge_write(uint32_t data, uint32_t size) {
    if (runtime_console_bridge_write_proc == 0 || data == 0 || size == 0) {
        return 0;
    }

    ((kos_stdcall2_void_fn)(uintptr_t)runtime_console_bridge_write_proc)(data, size);
    return 1;
}

int runtime_console_bridge_read_line(uint32_t data, uint32_t size) {
    if (runtime_console_bridge_gets_proc == 0 || data == 0 || size < 2) {
        return 0;
    }

    return ((kos_stdcall2_fn)(uintptr_t)runtime_console_bridge_gets_proc)(data, size) != 0;
}

void runtime_console_bridge_close(uint32_t close_window) {
    if (runtime_console_bridge_exit_proc == 0) {
        return;
    }

    ((kos_stdcall1_void_fn)(uintptr_t)runtime_console_bridge_exit_proc)(close_window);
    runtime_console_bridge_table = 0;
    runtime_console_bridge_write_proc = 0;
    runtime_console_bridge_exit_proc = 0;
    runtime_console_bridge_gets_proc = 0;
}

static size_t runtime_type_size(const go_type_descriptor* descriptor) {
    if (descriptor == NULL) {
        return 0;
    }

    return (size_t)descriptor->size;
}

static size_t runtime_map_key_size(const go_map_type_descriptor* map_type) {
    if (map_type == NULL) {
        return 0;
    }
    if (map_type->key_type != NULL && map_type->key_type->size != 0) {
        return (size_t)map_type->key_type->size;
    }
    if (map_type->key_size != 0) {
        return (size_t)map_type->key_size;
    }

    return 0;
}

static size_t runtime_map_value_size(const go_map_type_descriptor* map_type) {
    if (map_type == NULL) {
        return 0;
    }
    if (map_type->value_type != NULL && map_type->value_type->size != 0) {
        return (size_t)map_type->value_type->size;
    }
    if (map_type->value_size != 0) {
        return (size_t)map_type->value_size;
    }

    return 0;
}

static void* runtime_alloc_zeroed(size_t size) {
    return runtime_gc_alloc_managed(size, NULL, NULL, NULL, 0);
}

static runtime_map* runtime_alloc_map(void) {
    return runtime_gc_alloc_map_object();
}

static bool runtime_map_bind_type(runtime_map* map, const go_map_type_descriptor* map_type) {
    if (map == NULL || map_type == NULL) {
        return false;
    }
    if (map->type == NULL) {
        map->type = map_type;
        return true;
    }

    return map->type == map_type;
}

static void* runtime_map_zero_value_for_type(const go_map_type_descriptor* map_type) {
    size_t value_size;

    value_size = runtime_map_value_size(map_type);
    if (map_type != NULL && map_type->value_type != NULL) {
        return runtime_gc_alloc_object(map_type->value_type);
    }

    return runtime_alloc_zeroed(value_size);
}

static void* runtime_map_zero_value(runtime_map* map, const go_map_type_descriptor* map_type) {
    if (map == NULL) {
        return runtime_map_zero_value_for_type(map_type);
    }
    if (map->zero_value == NULL) {
        map->zero_value = runtime_map_zero_value_for_type(map_type);
    }

    return map->zero_value;
}

#define RUNTIME_MAP_MIN_CAP 8
#define RUNTIME_MAP_MAX_LOAD_NUM 3
#define RUNTIME_MAP_MAX_LOAD_DEN 4

static intptr_t runtime_map_next_power_of_two(intptr_t value) {
    intptr_t out;

    if (value <= 1) {
        return 1;
    }

    out = 1;
    while (out < value) {
        if (out > INTPTR_MAX / 2) {
            return value;
        }
        out <<= 1;
    }

    return out;
}

static bool runtime_map_rehash(runtime_map* map, intptr_t new_cap) {
    runtime_map_entry* resized;
    runtime_map_entry* previous_entries;
    intptr_t previous_cap;
    intptr_t index;
    intptr_t mask;

    if (map == NULL) {
        return false;
    }

    if (new_cap < RUNTIME_MAP_MIN_CAP) {
        new_cap = RUNTIME_MAP_MIN_CAP;
    }
    new_cap = runtime_map_next_power_of_two(new_cap);

    resized = runtime_gc_alloc_map_entries(map, new_cap);
    if (resized == NULL) {
        return false;
    }

    previous_entries = map->entries;
    previous_cap = map->cap;

    map->entries = resized;
    map->cap = new_cap;
    map->len = 0;
    map->used = 0;
    runtime_gc_update_allocation_aux(map->entries, (uintptr_t)map->cap);

    if (previous_entries != NULL && previous_cap > 0) {
        mask = map->cap - 1;
        for (index = 0; index < previous_cap; index++) {
            runtime_map_entry* entry = &previous_entries[index];
            if (entry->state != 1) {
                continue;
            }

            intptr_t slot = (intptr_t)(entry->hash & (uint32_t)mask);
            while (resized[slot].state == 1) {
                slot = (slot + 1) & mask;
            }
            resized[slot] = *entry;
            resized[slot].state = 1;
            map->len++;
            map->used++;
        }
        runtime_gc_free_exact(previous_entries);
    }

    return true;
}

static bool runtime_map_reserve(runtime_map* map, intptr_t needed) {
    intptr_t desired;

    if (map == NULL) {
        return false;
    }

    if (needed < 0) {
        return false;
    }

    desired = map->cap;
    if (desired < RUNTIME_MAP_MIN_CAP) {
        desired = RUNTIME_MAP_MIN_CAP;
    }

    while (desired > 0 && desired * RUNTIME_MAP_MAX_LOAD_NUM < needed * RUNTIME_MAP_MAX_LOAD_DEN) {
        if (desired > INTPTR_MAX / 2) {
            break;
        }
        desired <<= 1;
    }

    if (map->cap == 0) {
        return runtime_map_rehash(map, desired);
    }

    if (desired != map->cap) {
        return runtime_map_rehash(map, desired);
    }

    if (map->used > map->len * 2) {
        return runtime_map_rehash(map, map->cap);
    }

    return true;
}

static uint32_t RUNTIME_USED runtime_memhash32_impl(const void* value) {
    uint32_t hash;

    if (value == NULL) {
        return 0;
    }

    hash = *(const uint32_t*)value;
    hash ^= hash >> 16;
    hash *= 0x7feb352du;
    hash ^= hash >> 15;
    hash *= 0x846ca68bu;
    hash ^= hash >> 16;
    return hash;
}

static uint32_t RUNTIME_USED runtime_memhash8_impl(const void* value) {
    if (value == NULL) {
        return 0;
    }
    return runtime_hash_bytes((const unsigned char*)value, 1);
}

static uint32_t RUNTIME_USED runtime_memhash16_impl(const void* value) {
    if (value == NULL) {
        return 0;
    }
    return runtime_hash_bytes((const unsigned char*)value, 2);
}

static uint32_t RUNTIME_USED runtime_memhash64_impl(const void* value) {
    if (value == NULL) {
        return 0;
    }
    return runtime_hash_bytes((const unsigned char*)value, 8);
}

uintptr_t runtime_memhash(const void* value, uintptr_t seed, uintptr_t size) {
    if (value == NULL || size == 0) {
        return seed;
    }

    return (uintptr_t)runtime_hash_bytes_seeded((const unsigned char*)value, (size_t)size, (uint32_t)seed);
}

uintptr_t runtime_memhash8(const void* value, uintptr_t seed) {
    return runtime_memhash(value, seed, 1);
}

uintptr_t runtime_memhash16(const void* value, uintptr_t seed) {
    return runtime_memhash(value, seed, 2);
}

uintptr_t runtime_memhash32(const void* value, uintptr_t seed) {
    uint32_t hash;

    if (value == NULL) {
        return seed;
    }

    hash = runtime_memhash32_impl(value);
    hash ^= (uint32_t)seed + 0x9e3779b9u + (hash << 6) + (hash >> 2);
    return (uintptr_t)hash;
}

uintptr_t runtime_memhash64(const void* value, uintptr_t seed) {
    return runtime_memhash(value, seed, 8);
}

static uint32_t RUNTIME_USED runtime_strhash_impl(const void* value) {
    const go_string* text;
    uint32_t hash;
    intptr_t index;

    if (value == NULL) {
        return 0;
    }

    text = (const go_string*)value;
    if (text->str == NULL || text->len <= 0) {
        return 0;
    }

    hash = 2166136261u;
    for (index = 0; index < text->len; index++) {
        hash ^= (uint32_t)(unsigned char)text->str[index];
        hash *= 16777619u;
    }

    return hash;
}

static uint32_t RUNTIME_USED runtime_nilinterhash_impl(const void* value) {
    const go_empty_interface* iface;
    uint32_t type_hash;
    uint32_t value_hash;

    if (value == NULL) {
        return 0;
    }

    iface = (const go_empty_interface*)value;
    if (iface->type == NULL) {
        return 0;
    }

    type_hash = iface->type->hash;
    value_hash = runtime_hash_value(iface->type, iface->data);
    return type_hash ^ (value_hash + 0x9e3779b9u + (type_hash << 6) + (type_hash >> 2));
}

static uint32_t RUNTIME_USED runtime_interhash_impl(const void* value) {
    const go_interface* iface;
    const go_type_descriptor* concrete_type;
    uint32_t type_hash;
    uint32_t value_hash;

    if (value == NULL) {
        return 0;
    }

    iface = (const go_interface*)value;
    if (iface->methods == NULL || iface->methods->type == NULL) {
        return 0;
    }

    concrete_type = iface->methods->type;
    type_hash = concrete_type->hash;
    value_hash = runtime_hash_value(concrete_type, iface->data);
    return type_hash ^ (value_hash + 0x9e3779b9u + (type_hash << 6) + (type_hash >> 2));
}

static uint32_t runtime_hash_interface(const go_type_descriptor* descriptor, const void* data) {
    const go_interface_type_descriptor* iface_type;

    if (descriptor == NULL) {
        return 0;
    }

    iface_type = (const go_interface_type_descriptor*)descriptor;
    if (iface_type->method_count == 0 && iface_type->exported_method_count == 0) {
        return runtime_nilinterhash_impl(data);
    }

    return runtime_interhash_impl(data);
}

static uint32_t RUNTIME_USED runtime_f32hash_impl(const void* value) {
    float input;

    if (value == NULL) {
        return 0;
    }

    input = *(const float*)value;
    return runtime_hash_float32_value(input);
}

static uint32_t RUNTIME_USED runtime_f64hash_impl(const void* value) {
    double input;

    if (value == NULL) {
        return 0;
    }

    input = *(const double*)value;
    return runtime_hash_float64_value(input);
}

static bool RUNTIME_USED runtime_f32equal_impl(const void* left, const void* right) {
    if (left == right) {
        return true;
    }

    if (left == NULL || right == NULL) {
        return false;
    }

    return *(const float*)left == *(const float*)right;
}

static bool RUNTIME_USED runtime_f64equal_impl(const void* left, const void* right) {
    if (left == right) {
        return true;
    }

    if (left == NULL || right == NULL) {
        return false;
    }

    return *(const double*)left == *(const double*)right;
}

static bool runtime_map_key_equal(const go_type_descriptor* descriptor, const void* left, const void* right, size_t key_size) {
    go_equal_function equal;

    if (descriptor != NULL) {
        equal = runtime_resolve_equal_function(descriptor);
        if (equal == NULL) {
            runtime_fail_simple("map key not comparable");
        }
        return equal(left, right);
    }
    if (key_size == 0) {
        return true;
    }

    return kos_memcmp(left, right, key_size) == 0;
}

static intptr_t runtime_map_find_generic(const go_map_type_descriptor* map_type, runtime_map* map, const void* key) {
    size_t key_size;
    uint32_t hash;
    intptr_t mask;
    intptr_t index;
    intptr_t probe;

    if (map == NULL || map->cap == 0) {
        return -1;
    }

    key_size = runtime_map_key_size(map_type);
    if (map_type != NULL && map_type->key_type != NULL) {
        hash = runtime_hash_value(map_type->key_type, key);
    } else {
        hash = runtime_hash_bytes((const unsigned char*)key, key_size);
    }

    mask = map->cap - 1;
    index = (intptr_t)(hash & (uint32_t)mask);
    for (probe = 0; probe < map->cap; probe++) {
        runtime_map_entry* entry = &map->entries[index];
        if (entry->state == 0) {
            return -1;
        }
        if (entry->state == 1 && entry->hash == hash &&
            runtime_map_key_equal(map_type != NULL ? map_type->key_type : NULL,
                                  entry->key_data,
                                  key,
                                  key_size)) {
            return index;
        }
        index = (index + 1) & mask;
    }

    return -1;
}

static intptr_t runtime_map_find_fast32(runtime_map* map, uint32_t key) {
    uint32_t hash;
    intptr_t mask;
    intptr_t index;
    intptr_t probe;

    if (map == NULL || map->cap == 0) {
        return -1;
    }

    hash = runtime_memhash32_impl(&key);
    mask = map->cap - 1;
    index = (intptr_t)(hash & (uint32_t)mask);
    for (probe = 0; probe < map->cap; probe++) {
        runtime_map_entry* entry = &map->entries[index];
        if (entry->state == 0) {
            return -1;
        }
        if (entry->state == 1 && entry->hash == hash) {
            const uint32_t* stored = (const uint32_t*)entry->key_data;
            if (stored != NULL && stored[0] == key) {
                return index;
            }
        }
        index = (index + 1) & mask;
    }

    return -1;
}

static intptr_t runtime_map_find_faststr(runtime_map* map, const char* key_ptr, intptr_t key_len) {
    go_string key;
    uint32_t hash;
    intptr_t mask;
    intptr_t index;
    intptr_t probe;

    if (map == NULL || map->cap == 0) {
        return -1;
    }

    key.str = key_ptr;
    key.len = key_len;
    hash = runtime_strhash_impl(&key);
    mask = map->cap - 1;
    index = (intptr_t)(hash & (uint32_t)mask);
    for (probe = 0; probe < map->cap; probe++) {
        runtime_map_entry* entry = &map->entries[index];
        if (entry->state == 0) {
            return -1;
        }
        if (entry->state == 1 && entry->hash == hash) {
            const go_string* stored = (const go_string*)entry->key_data;
            if (runtime_string_equals(&key, stored)) {
                return index;
            }
        }
        index = (index + 1) & mask;
    }

    return -1;
}

static runtime_map_entry* runtime_map_insert_fast32(runtime_map* map, const go_map_type_descriptor* map_type, uint32_t key) {
    runtime_map_entry* entry;
    uint32_t hash;
    intptr_t mask;
    intptr_t index;
    intptr_t probe;
    intptr_t tombstone;

    if (map == NULL || !runtime_map_bind_type(map, map_type)) {
        return NULL;
    }

    if (!runtime_map_reserve(map, map->len + 1)) {
        return NULL;
    }

    if (map->cap == 0) {
        return NULL;
    }

    hash = runtime_memhash32_impl(&key);
    mask = map->cap - 1;
    index = (intptr_t)(hash & (uint32_t)mask);
    tombstone = -1;
    for (probe = 0; probe < map->cap; probe++) {
        entry = &map->entries[index];
        if (entry->state == 0) {
            if (tombstone >= 0) {
                entry = &map->entries[tombstone];
            }
            break;
        }
        if (entry->state == 2) {
            if (tombstone < 0) {
                tombstone = index;
            }
        } else if (entry->hash == hash) {
            const uint32_t* stored = (const uint32_t*)entry->key_data;
            if (stored != NULL && stored[0] == key) {
                return entry;
            }
        }
        index = (index + 1) & mask;
    }

    if (entry->state == 0) {
        map->used++;
    }
    map->len++;
    if (map_type != NULL && map_type->key_type != NULL) {
        entry->key_data = runtime_gc_alloc_object(map_type->key_type);
    } else {
        entry->key_data = runtime_alloc_zeroed(runtime_map_key_size(map_type));
    }
    if (map_type != NULL && map_type->value_type != NULL) {
        entry->value_data = runtime_gc_alloc_object(map_type->value_type);
    } else {
        entry->value_data = runtime_alloc_zeroed(runtime_map_value_size(map_type));
    }
    if (entry->key_data == NULL || entry->value_data == NULL) {
        if (entry->key_data != NULL) {
            runtime_gc_free_exact(entry->key_data);
        }
        if (entry->value_data != NULL) {
            runtime_gc_free_exact(entry->value_data);
        }
        return NULL;
    }

    *(uint32_t*)entry->key_data = key;
    entry->hash = hash;
    entry->state = 1;
    return entry;
}

static runtime_map_entry* runtime_map_insert_faststr(runtime_map* map, const go_map_type_descriptor* map_type, const char* key_ptr, intptr_t key_len) {
    runtime_map_entry* entry;
    go_string* stored;
    go_string key;
    uint32_t hash;
    intptr_t mask;
    intptr_t index;
    intptr_t probe;
    intptr_t tombstone;

    if (map == NULL || !runtime_map_bind_type(map, map_type)) {
        return NULL;
    }

    if (!runtime_map_reserve(map, map->len + 1)) {
        return NULL;
    }

    if (map->cap == 0) {
        return NULL;
    }

    key.str = key_ptr;
    key.len = key_len;
    hash = runtime_strhash_impl(&key);
    mask = map->cap - 1;
    index = (intptr_t)(hash & (uint32_t)mask);
    tombstone = -1;
    for (probe = 0; probe < map->cap; probe++) {
        entry = &map->entries[index];
        if (entry->state == 0) {
            if (tombstone >= 0) {
                entry = &map->entries[tombstone];
            }
            break;
        }
        if (entry->state == 2) {
            if (tombstone < 0) {
                tombstone = index;
            }
        } else if (entry->hash == hash) {
            const go_string* existing = (const go_string*)entry->key_data;
            if (runtime_string_equals(&key, existing)) {
                return entry;
            }
        }
        index = (index + 1) & mask;
    }

    if (entry->state == 0) {
        map->used++;
    }
    map->len++;
    if (map_type != NULL && map_type->key_type != NULL) {
        entry->key_data = runtime_gc_alloc_object(map_type->key_type);
    } else {
        entry->key_data = runtime_alloc_zeroed(runtime_map_key_size(map_type));
    }
    if (map_type != NULL && map_type->value_type != NULL) {
        entry->value_data = runtime_gc_alloc_object(map_type->value_type);
    } else {
        entry->value_data = runtime_alloc_zeroed(runtime_map_value_size(map_type));
    }
    if (entry->key_data == NULL || entry->value_data == NULL) {
        if (entry->key_data != NULL) {
            runtime_gc_free_exact(entry->key_data);
        }
        if (entry->value_data != NULL) {
            runtime_gc_free_exact(entry->value_data);
        }
        return NULL;
    }

    stored = (go_string*)entry->key_data;
    stored->str = key_ptr;
    stored->len = key_len;
    entry->hash = hash;
    entry->state = 1;
    return entry;
}

static runtime_map_entry* runtime_map_insert_generic(runtime_map* map, const go_map_type_descriptor* map_type, const void* key) {
    runtime_map_entry* entry;
    size_t key_size;
    size_t value_size;
    uint32_t hash;
    intptr_t mask;
    intptr_t index;
    intptr_t probe;
    intptr_t tombstone;

    if (map == NULL || !runtime_map_bind_type(map, map_type)) {
        return NULL;
    }

    if (!runtime_map_reserve(map, map->len + 1)) {
        return NULL;
    }

    if (map->cap == 0) {
        return NULL;
    }

    key_size = runtime_map_key_size(map_type);
    if (map_type != NULL && map_type->key_type != NULL) {
        hash = runtime_hash_value(map_type->key_type, key);
    } else {
        hash = runtime_hash_bytes((const unsigned char*)key, key_size);
    }
    mask = map->cap - 1;
    index = (intptr_t)(hash & (uint32_t)mask);
    tombstone = -1;
    for (probe = 0; probe < map->cap; probe++) {
        entry = &map->entries[index];
        if (entry->state == 0) {
            if (tombstone >= 0) {
                entry = &map->entries[tombstone];
            }
            break;
        }
        if (entry->state == 2) {
            if (tombstone < 0) {
                tombstone = index;
            }
        } else if (entry->hash == hash &&
                   runtime_map_key_equal(map_type != NULL ? map_type->key_type : NULL,
                                         entry->key_data,
                                         key,
                                         key_size)) {
            return entry;
        }
        index = (index + 1) & mask;
    }

    if (entry->state == 0) {
        map->used++;
    }
    map->len++;
    key_size = runtime_map_key_size(map_type);
    value_size = runtime_map_value_size(map_type);
    if (map_type != NULL && map_type->key_type != NULL) {
        entry->key_data = runtime_gc_alloc_object(map_type->key_type);
    } else {
        entry->key_data = runtime_alloc_zeroed(key_size);
    }
    if (map_type != NULL && map_type->value_type != NULL) {
        entry->value_data = runtime_gc_alloc_object(map_type->value_type);
    } else {
        entry->value_data = runtime_alloc_zeroed(value_size);
    }
    if (entry->key_data == NULL || entry->value_data == NULL) {
        if (entry->key_data != NULL) {
            runtime_gc_free_exact(entry->key_data);
        }
        if (entry->value_data != NULL) {
            runtime_gc_free_exact(entry->value_data);
        }
        return NULL;
    }

    if (key_size != 0) {
        if (map_type != NULL && map_type->key_type != NULL) {
            runtime_typedmemmove(map_type->key_type, entry->key_data, key);
        } else {
            kos_memcpy(entry->key_data, key, key_size);
        }
    }
    entry->hash = hash;
    entry->state = 1;
    return entry;
}

static void runtime_map_remove_at(runtime_map* map, intptr_t index) {
    if (map == NULL || map->cap == 0 || index < 0 || index >= map->cap) {
        return;
    }

    if (map->entries[index].key_data != NULL) {
        runtime_gc_free_exact(map->entries[index].key_data);
    }
    if (map->entries[index].value_data != NULL) {
        runtime_gc_free_exact(map->entries[index].value_data);
    }

    map->entries[index].key_data = NULL;
    map->entries[index].value_data = NULL;
    map->entries[index].hash = 0;
    map->entries[index].state = 2;
    map->len--;
}

void runtime_mapclear(const go_map_type_descriptor* map_type, runtime_map* map) {
    (void)map_type;

    if (map == NULL) {
        return;
    }

    if (map->cap == 0 || map->entries == NULL) {
        map->len = 0;
        map->used = 0;
        return;
    }

    for (intptr_t i = 0; i < map->cap; i++) {
        runtime_map_entry* entry = &map->entries[i];
        if (entry->key_data != NULL) {
            runtime_gc_free_exact(entry->key_data);
        }
        if (entry->value_data != NULL) {
            runtime_gc_free_exact(entry->value_data);
        }
        entry->key_data = NULL;
        entry->value_data = NULL;
        entry->hash = 0;
        entry->state = 0;
    }

    map->len = 0;
    map->used = 0;
}

void* runtime_makemap__small(void) {
    return runtime_alloc_map();
}

void* runtime_makemap(const go_map_type_descriptor* map_type, intptr_t hint, void* ignored) {
    runtime_map* map;

    (void)ignored;

    if (hint < 0) {
        runtime_panicmem();
    }

    map = runtime_alloc_map();
    if (map == NULL) {
        return NULL;
    }
    if (map_type != NULL) {
        map->type = map_type;
        if (hint > 0) {
            runtime_map_reserve(map, hint);
        }
    }

    return map;
}

void* __go_construct_map(const go_map_type_descriptor* map_type,
                         uintptr_t count,
                         uintptr_t entry_size,
                         uintptr_t key_size,
                         const void* data) {
    runtime_map* map;
    const uint8_t* cursor;
    uintptr_t index;
    size_t value_size;

    map = runtime_alloc_map();
    if (map == NULL) {
        return NULL;
    }
    if (map_type != NULL) {
        map->type = map_type;
        if (count > 0) {
            runtime_map_reserve(map, (intptr_t)count);
        }
    }

    if (count == 0 || data == NULL) {
        return map;
    }

    value_size = 0;
    if (map_type != NULL && map_type->value_type != NULL && map_type->value_type->size != 0) {
        value_size = (size_t)map_type->value_type->size;
    } else if (entry_size >= key_size) {
        value_size = (size_t)(entry_size - key_size);
    }

    cursor = (const uint8_t*)data;
    for (index = 0; index < count; index++) {
        const uint8_t* key_ptr = cursor;
        const uint8_t* value_ptr = cursor + key_size;
        void* slot = NULL;

        slot = runtime_mapassign(map_type, map, key_ptr);

        if (slot != NULL && value_size > 0) {
            if (map_type != NULL && map_type->value_type != NULL) {
                runtime_typedmemmove(map_type->value_type, slot, value_ptr);
            } else {
                kos_memcpy(slot, value_ptr, value_size);
            }
        }

        cursor += entry_size;
    }

    return map;
}

static bool runtime_map_key_is_fast32(const go_map_type_descriptor* map_type) {
    uint8_t kind;

    if (map_type == NULL || map_type->key_type == NULL) {
        return false;
    }

    kind = map_type->key_type->kind & GO_TYPE_KIND_MASK;
    if (kind == GO_TYPE_KIND_STRING || kind == GO_TYPE_KIND_FLOAT32 || kind == GO_TYPE_KIND_FLOAT64) {
        return false;
    }

    return map_type->key_type->size == 4 && kind <= 0x0Cu;
}

void* runtime_mapassign(const go_map_type_descriptor* map_type, runtime_map* map, const void* key) {
    intptr_t index;
    runtime_map_entry* entry;

    if (map == NULL) {
        runtime_fail_simple("assignment to nil map");
    }

    if (map_type != NULL && map_type->key_type != NULL) {
        uint8_t kind = map_type->key_type->kind & GO_TYPE_KIND_MASK;
        if (kind == GO_TYPE_KIND_STRING) {
            const go_string* key_string = (const go_string*)key;
            return runtime_mapassign__faststr(map_type, map, key_string->str, key_string->len);
        }
        if (runtime_map_key_is_fast32(map_type)) {
            uint32_t key32 = 0;
            kos_memcpy(&key32, key, sizeof(uint32_t));
            return runtime_mapassign__fast32(map_type, map, key32);
        }
    }

    index = runtime_map_find_generic(map_type, map, key);
    if (index >= 0) {
        return map->entries[index].value_data;
    }

    entry = runtime_map_insert_generic(map, map_type, key);
    if (entry == NULL) {
        runtime_panicmem();
    }

    return entry->value_data;
}

void* runtime_mapassign__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key) {
    intptr_t index;
    runtime_map_entry* entry;

    if (map == NULL) {
        runtime_fail_simple("assignment to nil map");
    }

    index = runtime_map_find_fast32(map, key);
    if (index >= 0) {
        return map->entries[index].value_data;
    }

    entry = runtime_map_insert_fast32(map, map_type, key);
    if (entry == NULL) {
        runtime_panicmem();
    }

    return entry->value_data;
}

void* runtime_mapassign__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key) {
    return runtime_mapassign__fast32(map_type, map, (uint32_t)key);
}

void* runtime_mapassign__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key) {
    return runtime_mapassign(map_type, map, &key);
}

void* runtime_mapassign__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len) {
    intptr_t index;
    runtime_map_entry* entry;

    if (map == NULL) {
        runtime_fail_simple("assignment to nil map");
    }

    index = runtime_map_find_faststr(map, key_ptr, key_len);
    if (index >= 0) {
        return map->entries[index].value_data;
    }

    entry = runtime_map_insert_faststr(map, map_type, key_ptr, key_len);
    if (entry == NULL) {
        runtime_panicmem();
    }

    return entry->value_data;
}

void* runtime_mapaccess1(const go_map_type_descriptor* map_type, runtime_map* map, const void* key) {
    intptr_t index;

    if (map == NULL) {
        return runtime_map_zero_value(map, map_type);
    }

    if (map_type != NULL && map_type->key_type != NULL) {
        uint8_t kind = map_type->key_type->kind & GO_TYPE_KIND_MASK;
        if (kind == GO_TYPE_KIND_STRING) {
            const go_string* key_string = (const go_string*)key;
            return runtime_mapaccess1__faststr(map_type, map, key_string->str, key_string->len);
        }
        if (runtime_map_key_is_fast32(map_type)) {
            uint32_t key32 = 0;
            kos_memcpy(&key32, key, sizeof(uint32_t));
            return runtime_mapaccess1__fast32(map_type, map, key32);
        }
    }

    index = runtime_map_find_generic(map_type, map, key);
    if (index >= 0) {
        return map->entries[index].value_data;
    }

    return runtime_map_zero_value(map, map_type);
}

void* runtime_mapaccess1__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key) {
    intptr_t index;

    index = runtime_map_find_fast32(map, key);
    if (index >= 0) {
        return map->entries[index].value_data;
    }

    return runtime_map_zero_value(map, map_type);
}

void* runtime_mapaccess1__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key) {
    return runtime_mapaccess1__fast32(map_type, map, (uint32_t)key);
}

void* runtime_mapaccess1__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key) {
    return runtime_mapaccess1(map_type, map, &key);
}

void* runtime_mapaccess1__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len) {
    intptr_t index;

    index = runtime_map_find_faststr(map, key_ptr, key_len);
    if (index >= 0) {
        return map->entries[index].value_data;
    }

    return runtime_map_zero_value(map, map_type);
}

go_mapaccess2_result runtime_mapaccess2(const go_map_type_descriptor* map_type, runtime_map* map, const void* key) {
    go_mapaccess2_result result;
    intptr_t index;

    result.ok = 0;

    if (map == NULL) {
        result.value = runtime_map_zero_value(map, map_type);
        return result;
    }

    if (map_type != NULL && map_type->key_type != NULL) {
        uint8_t kind = map_type->key_type->kind & GO_TYPE_KIND_MASK;
        if (kind == GO_TYPE_KIND_STRING) {
            const go_string* key_string = (const go_string*)key;
            return runtime_mapaccess2__faststr(map_type, map, key_string->str, key_string->len);
        }
        if (runtime_map_key_is_fast32(map_type)) {
            uint32_t key32 = 0;
            kos_memcpy(&key32, key, sizeof(uint32_t));
            return runtime_mapaccess2__fast32(map_type, map, key32);
        }
    }

    index = runtime_map_find_generic(map_type, map, key);
    if (index >= 0) {
        result.value = map->entries[index].value_data;
        result.ok = 1;
        return result;
    }

    result.value = runtime_map_zero_value(map, map_type);
    return result;
}

go_mapaccess2_result runtime_mapaccess2__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key) {
    go_mapaccess2_result result;
    intptr_t index;

    result.ok = 0;
    index = runtime_map_find_fast32(map, key);
    if (index >= 0) {
        result.value = map->entries[index].value_data;
        result.ok = 1;
        return result;
    }

    result.value = runtime_map_zero_value(map, map_type);
    return result;
}

go_mapaccess2_result runtime_mapaccess2__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key) {
    return runtime_mapaccess2__fast32(map_type, map, (uint32_t)key);
}

go_mapaccess2_result runtime_mapaccess2__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key) {
    return runtime_mapaccess2(map_type, map, &key);
}

go_mapaccess2_result runtime_mapaccess2__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len) {
    go_mapaccess2_result result;
    intptr_t index;

    result.ok = 0;
    index = runtime_map_find_faststr(map, key_ptr, key_len);
    if (index >= 0) {
        result.value = map->entries[index].value_data;
        result.ok = 1;
        return result;
    }

    result.value = runtime_map_zero_value(map, map_type);
    return result;
}

void runtime_mapdelete(const go_map_type_descriptor* map_type, runtime_map* map, const void* key) {
    intptr_t index;

    if (map == NULL) {
        return;
    }

    if (map_type != NULL && map_type->key_type != NULL) {
        uint8_t kind = map_type->key_type->kind & GO_TYPE_KIND_MASK;
        if (kind == GO_TYPE_KIND_STRING) {
            const go_string* key_string = (const go_string*)key;
            runtime_mapdelete__faststr(map_type, map, key_string->str, key_string->len);
            return;
        }
        if (runtime_map_key_is_fast32(map_type)) {
            uint32_t key32 = 0;
            kos_memcpy(&key32, key, sizeof(uint32_t));
            runtime_mapdelete__fast32(map_type, map, key32);
            return;
        }
    }

    index = runtime_map_find_generic(map_type, map, key);
    if (index >= 0) {
        runtime_map_remove_at(map, index);
    }
}

void runtime_mapdelete__fast32(const go_map_type_descriptor* map_type, runtime_map* map, uint32_t key) {
    intptr_t index;

    (void)map_type;

    index = runtime_map_find_fast32(map, key);
    if (index >= 0) {
        runtime_map_remove_at(map, index);
    }
}

void runtime_mapdelete__fast32ptr(const go_map_type_descriptor* map_type, runtime_map* map, uintptr_t key) {
    runtime_mapdelete__fast32(map_type, map, (uint32_t)key);
}

void runtime_mapdelete__fast64(const go_map_type_descriptor* map_type, runtime_map* map, uint64_t key) {
    runtime_mapdelete(map_type, map, &key);
}

void runtime_mapdelete__faststr(const go_map_type_descriptor* map_type, runtime_map* map, const char* key_ptr, intptr_t key_len) {
    intptr_t index;

    (void)map_type;

    index = runtime_map_find_faststr(map, key_ptr, key_len);
    if (index >= 0) {
        runtime_map_remove_at(map, index);
    }
}

void runtime_mapiterinit(const go_map_type_descriptor* map_type, runtime_map* map, runtime_map_iterator* iterator) {
    runtime_map_iter_state* state;
    intptr_t index;

    (void)map_type;

    if (iterator == NULL) {
        return;
    }

    iterator->key = NULL;
    iterator->value = NULL;
    iterator->state = NULL;

    if (map == NULL || map->len == 0 || map->cap == 0) {
        return;
    }

    state = runtime_gc_alloc_map_iter_state();
    if (state == NULL) {
        return;
    }

    state->map = map;
    state->index = 0;
    iterator->state = state;

    for (index = 0; index < map->cap; index++) {
        if (map->entries[index].state == 1) {
            state->index = index;
            iterator->key = map->entries[index].key_data;
            iterator->value = map->entries[index].value_data;
            return;
        }
    }

    runtime_gc_free_exact(state);
    iterator->state = NULL;
    iterator->key = NULL;
    iterator->value = NULL;
}

void runtime_mapiternext(runtime_map_iterator* iterator) {
    runtime_map_iter_state* state;
    intptr_t next_index;

    if (iterator == NULL || iterator->state == NULL) {
        if (iterator != NULL) {
            iterator->key = NULL;
            iterator->value = NULL;
        }
        return;
    }

    state = iterator->state;
    if (state->map == NULL || state->map->cap == 0) {
        runtime_gc_free_exact(state);
        iterator->key = NULL;
        iterator->value = NULL;
        iterator->state = NULL;
        return;
    }

    next_index = state->index + 1;
    for (; next_index < state->map->cap; next_index++) {
        if (state->map->entries[next_index].state == 1) {
            state->index = next_index;
            iterator->key = state->map->entries[next_index].key_data;
            iterator->value = state->map->entries[next_index].value_data;
            return;
        }
    }

    runtime_gc_free_exact(state);
    iterator->key = NULL;
    iterator->value = NULL;
    iterator->state = NULL;
}

static bool RUNTIME_USED runtime_memequal_impl(const void* left, const void* right, size_t size) {
    if (left == NULL || right == NULL) {
        return false;
    }

    return kos_memcmp(left, right, size) == 0;
}

go_string runtime_intstring(void* tmp, int64_t value) {
    go_string out;
    uint32_t rune_value;
    char buffer[4];
    size_t length;
    char* result;

    (void)tmp;

    out.str = NULL;
    out.len = 0;

    if (value < 0 || value > 0x10FFFF) {
        rune_value = 0xFFFDu;
    } else {
        rune_value = (uint32_t)value;
        if (rune_value >= 0xD800u && rune_value <= 0xDFFFu) {
            rune_value = 0xFFFDu;
        }
    }

    if (rune_value <= 0x7Fu) {
        buffer[0] = (char)rune_value;
        length = 1;
    } else if (rune_value <= 0x7FFu) {
        buffer[0] = (char)(0xC0u | (rune_value >> 6));
        buffer[1] = (char)(0x80u | (rune_value & 0x3Fu));
        length = 2;
    } else if (rune_value <= 0xFFFFu) {
        buffer[0] = (char)(0xE0u | (rune_value >> 12));
        buffer[1] = (char)(0x80u | ((rune_value >> 6) & 0x3Fu));
        buffer[2] = (char)(0x80u | (rune_value & 0x3Fu));
        length = 3;
    } else {
        buffer[0] = (char)(0xF0u | (rune_value >> 18));
        buffer[1] = (char)(0x80u | ((rune_value >> 12) & 0x3Fu));
        buffer[2] = (char)(0x80u | ((rune_value >> 6) & 0x3Fu));
        buffer[3] = (char)(0x80u | (rune_value & 0x3Fu));
        length = 4;
    }

    result = (char*)runtime_gc_alloc_managed(length + 1, NULL, NULL, NULL, 0);
    if (result == NULL) {
        return out;
    }

    kos_memcpy(result, buffer, length);
    result[length] = '\0';
    out.str = result;
    out.len = (intptr_t)length;
    return out;
}

go_string runtime_concatstrings(uintptr_t ignored, const go_string* strings, size_t count) {
    size_t total_length;
    size_t offset;
    size_t index;
    char* result;
    go_string out;

    (void)ignored;

    if (strings == NULL || count == 0) {
        out.str = NULL;
        out.len = 0;
        return out;
    }

    total_length = 0;
    for (index = 0; index < count; index++) {
        if (strings[index].str != NULL && strings[index].len > 0) {
            total_length += (size_t)strings[index].len;
        }
    }

    result = (char*)runtime_gc_alloc_managed(total_length + 1, NULL, NULL, NULL, 0);
    if (result == NULL) {
        out.str = NULL;
        out.len = 0;
        return out;
    }

    offset = 0;
    for (index = 0; index < count; index++) {
        go_string current;
        size_t length;

        current = strings[index];
        if (current.str == NULL || current.len <= 0) {
            continue;
        }

        length = (size_t)current.len;
        kos_memcpy(result + offset, current.str, length);
        offset += length;
    }

    result[offset] = '\0';
    out.str = result;
    out.len = (intptr_t)offset;
    return out;
}

void runtime_set_byte_string(unsigned char* dest, const unsigned char* src, size_t size) {
    if (dest == NULL || src == NULL) {
        return;
    }

    kos_memcpy(dest, src, size);
}

void* runtime_makeslice(const go_type_descriptor* descriptor, intptr_t len, intptr_t cap) {
    size_t total_size;
    void* memory;

    if (len < 0 || cap < len) {
        runtime_panicmem();
    }

    if (cap == 0) {
        return NULL;
    }

    total_size = kos_slice_allocation_size(descriptor, cap);

    memory = runtime_gc_alloc_array(descriptor, cap, total_size);
    if (memory == NULL) {
        return NULL;
    }
    return memory;
}

void* runtime_makeslice64(const go_type_descriptor* descriptor, int64_t len, int64_t cap) {
    if (len < 0 || cap < len) {
        runtime_panicmem();
    }
    if (len > (int64_t)INTPTR_MAX || cap > (int64_t)INTPTR_MAX) {
        runtime_panicmem();
    }

    return runtime_makeslice(descriptor, (intptr_t)len, (intptr_t)cap);
}

go_slice runtime_growslice(const go_type_descriptor* descriptor, void* old_values, intptr_t old_len, intptr_t old_cap, intptr_t new_len) {
    go_slice result;
    size_t old_size;
    size_t new_size;
    intptr_t new_cap;
    void* memory;

    result.values = NULL;
    result.len = 0;
    result.cap = 0;

    if (old_len < 0 || old_cap < old_len || new_len < old_len) {
        runtime_panicmem();
    }

    new_cap = old_cap;
    if (new_cap < 1) {
        new_cap = 1;
    }

    while (new_cap < new_len) {
        if (new_cap > INTPTR_MAX / 2) {
            new_cap = new_len;
            break;
        }
        new_cap *= 2;
    }
    if (new_cap < new_len) {
        new_cap = new_len;
    }

    new_size = kos_slice_allocation_size(descriptor, new_cap);
    memory = runtime_gc_alloc_array(descriptor, new_cap, new_size);
    if (memory == NULL) {
        return result;
    }
    old_size = kos_slice_allocation_size(descriptor, old_len);
    if (old_values != NULL && old_size > 0) {
        kos_memmove(memory, old_values, old_size);
    }

    result.values = (unsigned char*)memory;
    result.len = new_len;
    result.cap = new_cap;
    return result;
}

void runtime_typedmemmove(const go_type_descriptor* descriptor, void* dest, const void* src) {
    size_t size;

    if (dest == NULL || src == NULL || dest == src) {
        return;
    }

    size = 0;
    if (descriptor != NULL) {
        size = (size_t)descriptor->size;
    }

    if (size == 0) {
        return;
    }

    kos_memmove(dest, src, size);
}

intptr_t runtime_typedslicecopy(const go_type_descriptor* descriptor, void* dst, intptr_t dstlen, const void* src, intptr_t srclen) {
    intptr_t n;
    size_t size;
    size_t total;

    if (dstlen < 0 || srclen < 0) {
        runtime_panicmem();
    }

    n = dstlen < srclen ? dstlen : srclen;
    if (n == 0) {
        return 0;
    }

    size = 0;
    if (descriptor != NULL) {
        size = (size_t)descriptor->size;
    }
    if (size == 0) {
        return n;
    }
    if ((size_t)n > (size_t)-1 / size) {
        runtime_panicmem();
    }

    total = size * (size_t)n;
    if (dst != NULL && src != NULL) {
        kos_memmove(dst, src, total);
    }
    return n;
}

go_string runtime_slicebytetostring(void* ignored, const unsigned char* src, intptr_t len) {
    char* out;
    go_string result;

    (void)ignored;

    if (src == NULL || len <= 0) {
        result.str = NULL;
        result.len = 0;
        return result;
    }

    out = (char*)runtime_gc_alloc_managed((size_t)len + 1, NULL, NULL, NULL, 0);
    if (out == NULL) {
        result.str = NULL;
        result.len = 0;
        return result;
    }

    kos_memcpy(out, src, (size_t)len);
    out[len] = '\0';

    result.str = out;
    result.len = len;
    return result;
}

go_slice runtime_stringtoslicebyte(void* ignored, const char* src, intptr_t len) {
    go_slice result;

    (void)ignored;

    result.values = NULL;
    result.len = 0;
    result.cap = 0;
    if (src == NULL || len <= 0) {
        return result;
    }

    result.values = (unsigned char*)runtime_gc_alloc_managed((size_t)len, NULL, NULL, NULL, 0);
    if (result.values == NULL) {
        return result;
    }

    kos_memcpy(result.values, src, (size_t)len);
    result.len = len;
    result.cap = len;
    return result;
}

go_slice runtime_stringtoslicerune(void* ignored, const char* src, intptr_t len) {
    go_slice result;
    go_string s;
    intptr_t count;
    intptr_t index;
    intptr_t pos;
    int32_t* out;

    (void)ignored;

    result.values = NULL;
    result.len = 0;
    result.cap = 0;
    if (src == NULL || len <= 0) {
        return result;
    }

    s.str = src;
    s.len = len;
    count = 0;
    for (pos = 0; pos < len;) {
        runtime_decoderune_result r = runtime_decoderune(s, pos);
        pos = r.pos;
        count++;
    }
    if (count <= 0) {
        return result;
    }

    out = (int32_t*)runtime_gc_alloc_managed((size_t)count * sizeof(int32_t), NULL, NULL, NULL, 0);
    if (out == NULL) {
        return result;
    }

    index = 0;
    for (pos = 0; pos < len && index < count;) {
        runtime_decoderune_result r = runtime_decoderune(s, pos);
        out[index++] = r.r;
        pos = r.pos;
    }

    result.values = (unsigned char*)out;
    result.len = count;
    result.cap = count;
    return result;
}

go_string runtime_slicerunetostring(void* ignored, const int32_t* src, intptr_t len) {
    go_string out;
    size_t total;
    size_t offset;
    intptr_t index;
    char* result;

    (void)ignored;

    out.str = NULL;
    out.len = 0;
    if (src == NULL || len <= 0) {
        return out;
    }

    total = 0;
    for (index = 0; index < len; index++) {
        uint32_t rune_value = (uint32_t)src[index];
        if (src[index] < 0 || rune_value > 0x10FFFFu || (rune_value >= 0xD800u && rune_value <= 0xDFFFu)) {
            rune_value = 0xFFFDu;
        }
        if (rune_value <= 0x7Fu) {
            total += 1;
        } else if (rune_value <= 0x7FFu) {
            total += 2;
        } else if (rune_value <= 0xFFFFu) {
            total += 3;
        } else {
            total += 4;
        }
    }

    result = (char*)runtime_gc_alloc_managed(total + 1, NULL, NULL, NULL, 0);
    if (result == NULL) {
        return out;
    }

    offset = 0;
    for (index = 0; index < len; index++) {
        uint32_t rune_value = (uint32_t)src[index];
        if (src[index] < 0 || rune_value > 0x10FFFFu || (rune_value >= 0xD800u && rune_value <= 0xDFFFu)) {
            rune_value = 0xFFFDu;
        }
        if (rune_value <= 0x7Fu) {
            result[offset++] = (char)rune_value;
        } else if (rune_value <= 0x7FFu) {
            result[offset++] = (char)(0xC0u | (rune_value >> 6));
            result[offset++] = (char)(0x80u | (rune_value & 0x3Fu));
        } else if (rune_value <= 0xFFFFu) {
            result[offset++] = (char)(0xE0u | (rune_value >> 12));
            result[offset++] = (char)(0x80u | ((rune_value >> 6) & 0x3Fu));
            result[offset++] = (char)(0x80u | (rune_value & 0x3Fu));
        } else {
            result[offset++] = (char)(0xF0u | (rune_value >> 18));
            result[offset++] = (char)(0x80u | ((rune_value >> 12) & 0x3Fu));
            result[offset++] = (char)(0x80u | ((rune_value >> 6) & 0x3Fu));
            result[offset++] = (char)(0x80u | (rune_value & 0x3Fu));
        }
    }

    result[offset] = '\0';
    out.str = result;
    out.len = (intptr_t)offset;
    return out;
}

void runtime_write_barrier(void** slot, void* ptr) {
    if (slot != NULL) {
        *slot = ptr;
    }
}

void runtime_gc_write_barrier(void** slot, void* ptr) {
    runtime_write_barrier(slot, ptr);
}

static bool RUNTIME_USED runtime_strequal_impl(const void* left_value, const void* right_value) {
    const go_string* left;
    const go_string* right;
    size_t length;

    if (left_value == NULL || right_value == NULL) {
        return false;
    }

    left = (const go_string*)left_value;
    right = (const go_string*)right_value;

    if (left->len != right->len) {
        return false;
    }

    if (left->str == right->str) {
        return true;
    }

    if (left->str == NULL || right->str == NULL) {
        return false;
    }

    length = (size_t)left->len;
    return kos_memcmp(left->str, right->str, length) == 0;
}

static go_equal_function runtime_resolve_equal_function(const go_type_descriptor* descriptor) {
    if (descriptor == NULL || descriptor->equal == NULL) {
        return NULL;
    }

    return *descriptor->equal;
}

static bool runtime_type_descriptor_matches(const go_type_descriptor* left, const go_type_descriptor* right) {
    if (left == right) {
        return true;
    }
    if (left == NULL || right == NULL) {
        return false;
    }

    return left->size == right->size &&
        left->ptrdata == right->ptrdata &&
        left->hash == right->hash &&
        left->align == right->align &&
        left->field_align == right->field_align &&
        left->kind == right->kind &&
        runtime_string_equals(left->name, right->name);
}

static const go_named_type_method_descriptor* runtime_find_named_method(const go_uncommon_type* uncommon, const go_interface_method_descriptor* target_method) {
    const go_named_type_method_descriptor* methods;
    const go_named_type_method_descriptor* current;
    uint32_t index;

    if (uncommon == NULL || target_method == NULL || uncommon->methods == NULL || uncommon->method_count == 0) {
        return NULL;
    }

    methods = (const go_named_type_method_descriptor*)uncommon->methods;
    for (index = 0; index < uncommon->method_count; index++) {
        current = methods + index;
        if (!runtime_string_equals(current->name, target_method->name)) {
            continue;
        }
        if (!runtime_string_equals(current->package_path, target_method->package_path)) {
            continue;
        }
        if (!runtime_type_descriptor_matches(current->interface_type, target_method->type)) {
            continue;
        }

        return current;
    }

    return NULL;
}

static go_interface_method_table* runtime_build_interface_method_table(const go_interface_type_descriptor* target_interface, const go_type_descriptor* source_type) {
    const go_interface_method_descriptor* target_methods;
    const go_named_type_method_descriptor* source_method;
    const go_uncommon_type* uncommon;
    uintptr_t size;
    uintptr_t index;
    void** table_entries;

    if (target_interface == NULL || source_type == NULL) {
        return NULL;
    }
    if ((target_interface->common.kind & GO_TYPE_KIND_MASK) != GO_TYPE_KIND_INTERFACE) {
        return NULL;
    }

    size = sizeof(void*) + (uintptr_t)target_interface->method_count * sizeof(void*);
    table_entries = (void**)runtime_gc_alloc_managed((size_t)size, NULL, NULL, NULL, 0);
    if (table_entries == NULL) {
        return NULL;
    }

    table_entries[0] = (void*)source_type;
    if (target_interface->method_count == 0 || target_interface->methods == NULL) {
        return (go_interface_method_table*)table_entries;
    }

    uncommon = (const go_uncommon_type*)source_type->uncommon;
    target_methods = (const go_interface_method_descriptor*)target_interface->methods;
    for (index = 0; index < (uintptr_t)target_interface->method_count; index++) {
        source_method = runtime_find_named_method(uncommon, target_methods + index);
        if (source_method == NULL || source_method->function == NULL) {
            runtime_gc_free_exact(table_entries);
            return NULL;
        }

        table_entries[index + 1] = source_method->function;
    }

    return (go_interface_method_table*)table_entries;
}

static void runtime_zero_typed_value(const go_type_descriptor* descriptor, void* dest) {
    size_t size;

    if (descriptor == NULL || dest == NULL) {
        return;
    }

    size = (size_t)descriptor->size;
    if (size == 0) {
        return;
    }

    kos_memset(dest, 0, size);
}

static void runtime_copy_typed_value(const go_type_descriptor* descriptor, void* dest, const void* src) {
    uintptr_t direct_value;
    size_t size;

    if (descriptor == NULL || dest == NULL) {
        return;
    }

    size = (size_t)descriptor->size;
    if (size == 0) {
        return;
    }

    if ((descriptor->kind & GO_TYPE_KIND_DIRECT_IFACE) != 0) {
        direct_value = (uintptr_t)src;
        kos_memcpy(dest, &direct_value, size);
        return;
    }

    if (src == NULL) {
        runtime_zero_typed_value(descriptor, dest);
        return;
    }

    runtime_typedmemmove(descriptor, dest, src);
}

static bool runtime_value_equal(const go_type_descriptor* descriptor, const void* left_data, const void* right_data) {
    go_equal_function equal;

    if (descriptor == NULL) {
        return true;
    }

    if ((descriptor->kind & GO_TYPE_KIND_DIRECT_IFACE) != 0) {
        return left_data == right_data;
    }

    equal = runtime_resolve_equal_function(descriptor);
    if (equal == NULL) {
        runtime_fail_simple("equality on non-comparable type");
    }

    return equal(left_data, right_data);
}

bool runtime_efaceeq(const go_type_descriptor* left_type, const void* left_data, const go_type_descriptor* right_type, const void* right_data) {
    if (left_type != right_type) {
        return false;
    }

    return runtime_value_equal(left_type, left_data, right_data);
}

bool RUNTIME_USED runtime_nilinterequal(const void* left_value, const void* right_value) {
    const go_empty_interface* left;
    const go_empty_interface* right;

    left = (const go_empty_interface*)left_value;
    right = (const go_empty_interface*)right_value;
    if (left == NULL || right == NULL) {
        return left == right;
    }

    return runtime_efaceeq(left->type, left->data, right->type, right->data);
}

bool runtime_ifaceE2T2(const go_type_descriptor* target_type, const go_type_descriptor* source_type, const void* source_data, void* target_value) {
    if (target_type == NULL) {
        return false;
    }

    if (target_type != source_type) {
        runtime_zero_typed_value(target_type, target_value);
        return false;
    }

    runtime_copy_typed_value(target_type, target_value, source_data);
    return true;
}

bool runtime_ifaceI2T2(const go_type_descriptor* target_type, const go_interface_method_table* source_methods, const void* source_data, void* target_value) {
    const go_type_descriptor* source_type;

    source_type = NULL;
    if (source_methods != NULL) {
        source_type = source_methods->type;
    }

    return runtime_ifaceE2T2(target_type, source_type, source_data, target_value);
}

go_mapaccess2_result runtime_ifaceE2T2P(const go_type_descriptor* target_type, const go_type_descriptor* source_type, const void* source_data) {
    go_mapaccess2_result result;

    result.value = NULL;
    result.ok = 0;

    if (target_type == NULL || source_type == NULL) {
        return result;
    }

    if (target_type != source_type) {
        return result;
    }

    result.value = (void*)source_data;
    result.ok = 1;
    return result;
}

go_mapaccess2_result runtime_ifaceI2T2P(const go_type_descriptor* target_type, const go_interface_method_table* source_methods, const void* source_data) {
    const go_type_descriptor* source_type = NULL;

    if (source_methods != NULL) {
        source_type = source_methods->type;
    }

    return runtime_ifaceE2T2P(target_type, source_type, source_data);
}

bool runtime_ifaceT2Ip(const go_type_descriptor* target_type, const go_type_descriptor* source_type) {
    go_interface_method_table* methods;

    if (target_type == NULL || source_type == NULL) {
        return false;
    }
    if ((target_type->kind & GO_TYPE_KIND_MASK) != GO_TYPE_KIND_INTERFACE) {
        return false;
    }

    methods = runtime_build_interface_method_table((const go_interface_type_descriptor*)target_type, source_type);
    if (methods == NULL) {
        return false;
    }

    runtime_gc_free_exact(methods);
    return true;
}

go_interface_method_table* runtime_assertitab(const go_type_descriptor* target_type, const go_type_descriptor* source_type) {
    go_interface_method_table* methods;

    if (target_type == NULL) {
        runtime_fail_simple("interface assertion has no target type");
    }
    if ((target_type->kind & GO_TYPE_KIND_MASK) != GO_TYPE_KIND_INTERFACE) {
        runtime_fail_simple("assertitab target is not an interface");
    }
    if (source_type == NULL) {
        runtime_fail_simple("interface assertion on nil value");
    }

    methods = runtime_build_interface_method_table((const go_interface_type_descriptor*)target_type, source_type);
    if (methods == NULL) {
        runtime_fail_pair("interface assertion failed", "want", runtime_pointer_value((void*)target_type), "have", runtime_pointer_value((void*)source_type));
    }

    return methods;
}

go_interface_method_table* runtime_requireitab(const go_type_descriptor* target_type, const go_type_descriptor* source_type) {
    if (source_type == NULL) {
        return NULL;
    }

    return runtime_assertitab(target_type, source_type);
}

go_interface_assert_result runtime_ifaceE2I2(const go_type_descriptor* target_type, const go_type_descriptor* source_type, const void* source_data) {
    go_interface_assert_result result;

    result.value.methods = NULL;
    result.value.data = NULL;
    result.ok = false;

    if (source_type == NULL) {
        return result;
    }

    result.value.methods = runtime_build_interface_method_table((const go_interface_type_descriptor*)target_type, source_type);
    if (result.value.methods == NULL) {
        return result;
    }

    result.value.data = source_data;
    result.ok = true;
    return result;
}

go_interface_assert_result runtime_ifaceI2I2(const go_type_descriptor* target_type, const go_interface_method_table* source_methods, const void* source_data) {
    const go_type_descriptor* source_type;

    source_type = NULL;
    if (source_methods != NULL) {
        source_type = source_methods->type;
    }

    return runtime_ifaceE2I2(target_type, source_type, source_data);
}

bool runtime_ifaceeq(const go_interface_method_table* left_methods, const void* left_data, const go_interface_method_table* right_methods, const void* right_data) {
    const go_type_descriptor* left_type;
    const go_type_descriptor* right_type;

    if (left_methods == NULL) {
        return right_methods == NULL;
    }
    if (right_methods == NULL) {
        return false;
    }

    left_type = left_methods->type;
    right_type = right_methods->type;
    if (left_type != right_type) {
        return false;
    }

    return runtime_value_equal(left_type, left_data, right_data);
}

bool runtime_ifacevaleq(const go_interface_method_table* left_methods, const void* left_data, const go_type_descriptor* right_type, const void* right_data) {
    const go_type_descriptor* left_type;

    if (left_methods == NULL) {
        return false;
    }

    left_type = left_methods->type;
    if (left_type != right_type) {
        return false;
    }

    return runtime_value_equal(left_type, left_data, right_data);
}

bool runtime_interequal(const void* left_value, const void* right_value) {
    const go_interface* left;
    const go_interface* right;

    if (left_value == NULL || right_value == NULL) {
        return false;
    }

    left = (const go_interface*)left_value;
    right = (const go_interface*)right_value;
    return runtime_ifaceeq(left->methods, left->data, right->methods, right->data);
}

int memcmp(const void* left, const void* right, size_t size) {
    if (left == NULL || right == NULL) {
        return left == right ? 0 : (left == NULL ? -1 : 1);
    }

    return kos_memcmp(left, right, size);
}

void* runtime_newobject(const go_type_descriptor* descriptor) {
    void* memory;

    memory = runtime_gc_alloc_object(descriptor);
    if (memory == NULL) {
        return NULL;
    }
    return memory;
}

void runtime_panicmem(void) {
    runtime_fail_simple("memory or bounds failure");
}

runtime_decoderune_result runtime_decoderune(go_string s, intptr_t k) {
    const uint32_t rune_error = 0xFFFD;
    const unsigned char* data;
    intptr_t remaining;
    runtime_decoderune_result out;

    out.r = (int32_t)rune_error;
    out.pos = k + 1;

    if (k < 0 || k >= s.len) {
        return out;
    }

    data = (const unsigned char*)s.str + k;
    remaining = s.len - k;
    if (remaining <= 0) {
        return out;
    }

    if (data[0] < 0x80) {
        out.r = data[0];
        out.pos = k + 1;
        return out;
    }

    if (data[0] >= 0xC0 && data[0] < 0xE0) {
        if (remaining > 1 && data[1] >= 0x80 && data[1] <= 0xBF) {
            uint32_t r = ((uint32_t)(data[0] & 0x1F) << 6) | (uint32_t)(data[1] & 0x3F);
            if (r > 0x7F) {
                out.r = (int32_t)r;
                out.pos = k + 2;
                return out;
            }
        }
    } else if (data[0] >= 0xE0 && data[0] < 0xF0) {
        if (remaining > 2 &&
            data[1] >= 0x80 && data[1] <= 0xBF &&
            data[2] >= 0x80 && data[2] <= 0xBF) {
            uint32_t r = ((uint32_t)(data[0] & 0x0F) << 12) |
                         ((uint32_t)(data[1] & 0x3F) << 6) |
                         (uint32_t)(data[2] & 0x3F);
            if (r > 0x7FF && !(r >= 0xD800 && r <= 0xDFFF)) {
                out.r = (int32_t)r;
                out.pos = k + 3;
                return out;
            }
        }
    } else if (data[0] >= 0xF0 && data[0] < 0xF8) {
        if (remaining > 3 &&
            data[1] >= 0x80 && data[1] <= 0xBF &&
            data[2] >= 0x80 && data[2] <= 0xBF &&
            data[3] >= 0x80 && data[3] <= 0xBF) {
            uint32_t r = ((uint32_t)(data[0] & 0x07) << 18) |
                         ((uint32_t)(data[1] & 0x3F) << 12) |
                         ((uint32_t)(data[2] & 0x3F) << 6) |
                         (uint32_t)(data[3] & 0x3F);
            if (r > 0xFFFF && r <= 0x10FFFF) {
                out.r = (int32_t)r;
                out.pos = k + 4;
                return out;
            }
        }
    }

    return out;
}

__attribute__((noreturn)) void runtime_panicdottype(const go_type_descriptor* target_type, const go_type_descriptor* source_type, const go_type_descriptor* interface_type) {
    (void)interface_type;

    runtime_fail_pair("type assertion failed", "want", runtime_pointer_value((void*)target_type), "have", runtime_pointer_value((void*)source_type));
}

void runtime_goPanicIndex(int32_t index, int32_t bound) {
    runtime_fail_pair("index out of range", "index", (uint32_t)index, "bound", (uint32_t)bound);
}

void runtime_goPanicIndexU(uint32_t index, uint32_t bound) {
    runtime_fail_pair("index out of range", "index", index, "bound", bound);
}

void runtime_goPanicSliceAlen(int32_t index, int32_t bound) {
    runtime_fail_pair("slice upper bound exceeds length", "index", (uint32_t)index, "len", (uint32_t)bound);
}

void runtime_goPanicSliceAlenU(uint32_t index, uint32_t bound) {
    runtime_fail_pair("slice upper bound exceeds length", "index", index, "len", bound);
}

void runtime_goPanicSliceAcap(int32_t index, int32_t bound) {
    runtime_fail_pair("slice upper bound exceeds capacity", "index", (uint32_t)index, "cap", (uint32_t)bound);
}

void runtime_goPanicSliceAcapU(uint32_t index, uint32_t bound) {
    runtime_fail_pair("slice upper bound exceeds capacity", "index", index, "cap", bound);
}

void runtime_goPanicSliceB(int32_t low, int32_t high) {
    runtime_fail_pair("invalid slice bounds", "low", (uint32_t)low, "high", (uint32_t)high);
}

void runtime_goPanicSliceBU(uint32_t low, uint32_t high) {
    runtime_fail_pair("invalid slice bounds", "low", low, "high", high);
}

void runtime_goPanicSlice3Alen(int32_t index, int32_t bound) {
    runtime_fail_pair("3-index slice exceeds length", "index", (uint32_t)index, "len", (uint32_t)bound);
}

void runtime_goPanicSlice3AlenU(uint32_t index, uint32_t bound) {
    runtime_fail_pair("3-index slice exceeds length", "index", index, "len", bound);
}

void runtime_goPanicSlice3Acap(int32_t index, int32_t bound) {
    runtime_fail_pair("3-index slice exceeds capacity", "index", (uint32_t)index, "cap", (uint32_t)bound);
}

void runtime_goPanicSlice3AcapU(uint32_t index, uint32_t bound) {
    runtime_fail_pair("3-index slice exceeds capacity", "index", index, "cap", bound);
}

void runtime_goPanicSlice3B(int32_t index, int32_t bound) {
    runtime_fail_pair("invalid 3-index slice bounds", "index", (uint32_t)index, "bound", (uint32_t)bound);
}

void runtime_goPanicSlice3BU(uint32_t index, uint32_t bound) {
    runtime_fail_pair("invalid 3-index slice bounds", "index", index, "bound", bound);
}

void runtime_goPanicSlice3C(int32_t low, int32_t high) {
    runtime_fail_pair("invalid 3-index slice range", "low", (uint32_t)low, "high", (uint32_t)high);
}

void runtime_goPanicSlice3CU(uint32_t low, uint32_t high) {
    runtime_fail_pair("invalid 3-index slice range", "low", low, "high", high);
}

void runtime_goPanicSliceConvert(int32_t index, int32_t bound) {
    runtime_fail_pair("slice conversion out of range", "index", (uint32_t)index, "bound", (uint32_t)bound);
}

void runtime_goPanicExtendIndex(int32_t index, int32_t bound) {
    runtime_goPanicIndex(index, bound);
}

void runtime_goPanicExtendSliceAlen(int32_t index, int32_t bound) {
    runtime_goPanicSliceAlen(index, bound);
}

void runtime_goPanicExtendSliceAcap(int32_t index, int32_t bound) {
    runtime_goPanicSliceAcap(index, bound);
}

void runtime_goPanicExtendSliceB(int32_t low, int32_t high) {
    runtime_goPanicSliceB(low, high);
}

void runtime_goPanicExtendIndexU(uint32_t index, uint32_t bound) {
    runtime_goPanicIndexU(index, bound);
}

void runtime_goPanicExtendSliceAlenU(uint32_t index, uint32_t bound) {
    runtime_goPanicSliceAlenU(index, bound);
}

void runtime_goPanicExtendSliceAcapU(uint32_t index, uint32_t bound) {
    runtime_goPanicSliceAcapU(index, bound);
}

void runtime_goPanicExtendSliceBU(uint32_t low, uint32_t high) {
    runtime_goPanicSliceBU(low, high);
}

static void runtime_unwind_stack(void) {
    runtime_g* g = runtime_getg();
    uintptr_t size;
    void* buffer;

    if (g == NULL) {
        runtime_fail_simple("panic");
    }

#if KOLIBRI_UNWIND_DEBUG
    runtime_debug_mark("U:begin");
#endif
    runtime_register_eh_frames();
#if KOLIBRI_UNWIND_DEBUG
    runtime_debug_mark("U:afterreg");
#endif
    size = runtime_unwindExceptionSize();
    if (size == 0) {
        runtime_fail_simple("unwindExceptionSize");
    }
#if KOLIBRI_UNWIND_DEBUG
    runtime_debug_mark("U:alloc");
#endif
    buffer = runtime_pool_alloc((size_t)size);
    if (buffer == NULL) {
        runtime_panicmem();
    }

    g->exception = buffer;
#if KOLIBRI_UNWIND_DEBUG
    runtime_debug_mark("U:throw");
#endif
    runtime_throwException();
}

__attribute__((noreturn)) void runtime_gopanic(go_empty_interface value) {
    runtime_g* g = runtime_getg();
    runtime_panic p;

    if (g == NULL) {
        runtime_fail_simple("panic");
    }

    p.link = g->_panic;
    p.arg = value;
    p.recovered = 0;
    p.isforeign = 0;
    p.aborted = 0;
    p.goexit = 0;
    g->_panic = &p;

    for (;;) {
        runtime_defer* d = g->_defer;
        uintptr_t pfn;
        runtime_defer_fn fn;

        if (d == NULL) {
            break;
        }

        pfn = d->pfn;
        if (pfn == 0) {
            if (d->panic != NULL) {
                d->panic->aborted = 1;
            }
            d->panic = NULL;
            g->_defer = d->link;
            runtime_freedefer(d);
            continue;
        }

        d->pfn = 0;
        d->panic = &p;

        fn = (runtime_defer_fn)(uintptr_t)pfn;
        g->deferring = 1;
        fn(d->arg);
        g->deferring = 0;

        if (g->_defer != d) {
            runtime_fail_simple("bad defer entry in panic");
        }
        d->panic = NULL;

        if (p.recovered) {
            g->_panic = p.link;
            while (g->_panic != NULL && g->_panic->aborted) {
                g->_panic = g->_panic->link;
            }
            if (g->_panic == NULL) {
                g->sig = 0;
            }
            runtime_unwind_stack();
            runtime_fail_simple("unwindStack returned");
        }

        if (d->frame != NULL) {
            *d->frame = 0;
        }
        g->_defer = d->link;
        runtime_freedefer(d);
    }

    runtime_fail_simple("panic");
}

void runtime_panicdivide(void) {
    runtime_fail_simple("divide by zero");
}

static uint64_t runtime_udivmod64(uint64_t n, uint64_t d, uint64_t* rem) {
    uint64_t q = 0;
    uint64_t r = 0;
    int i;

    if (d == 0) {
        runtime_panicdivide();
        if (rem != NULL) {
            *rem = 0;
        }
        return 0;
    }

    for (i = 63; i >= 0; i--) {
        r = (r << 1) | ((n >> (uint32_t)i) & 1u);
        if (r >= d) {
            r -= d;
            q |= (uint64_t)1u << (uint32_t)i;
        }
    }

    if (rem != NULL) {
        *rem = r;
    }
    return q;
}

uint64_t __udivdi3(uint64_t n, uint64_t d) {
    return runtime_udivmod64(n, d, NULL);
}

uint64_t __udivmoddi4(uint64_t n, uint64_t d, uint64_t* rp) {
    uint64_t r;
    uint64_t q = runtime_udivmod64(n, d, &r);
    if (rp != NULL) {
        *rp = r;
    }
    return q;
}

uint64_t __umoddi3(uint64_t n, uint64_t d) {
    uint64_t r;
    runtime_udivmod64(n, d, &r);
    return r;
}

int64_t __divdi3(int64_t a, int64_t b) {
    uint64_t ua;
    uint64_t ub;
    uint64_t q;
    int neg = 0;

    if (a < 0) {
        ua = (uint64_t)(-a);
        neg = 1;
    } else {
        ua = (uint64_t)a;
    }
    if (b < 0) {
        ub = (uint64_t)(-b);
        neg ^= 1;
    } else {
        ub = (uint64_t)b;
    }
    q = runtime_udivmod64(ua, ub, NULL);
    if (neg) {
        return -(int64_t)q;
    }
    return (int64_t)q;
}

int64_t __moddi3(int64_t a, int64_t b) {
    uint64_t ua;
    uint64_t ub;
    uint64_t r;
    int neg = 0;

    if (a < 0) {
        ua = (uint64_t)(-a);
        neg = 1;
    } else {
        ua = (uint64_t)a;
    }
    if (b < 0) {
        ub = (uint64_t)(-b);
    } else {
        ub = (uint64_t)b;
    }
    runtime_udivmod64(ua, ub, &r);
    if (neg) {
        return -(int64_t)r;
    }
    return (int64_t)r;
}

bool runtime_efacevaleq(go_empty_interface left, const go_type_descriptor* right_type, const void* right_data) {
    if (left.type != right_type) {
        return false;
    }
    return runtime_value_equal(right_type, left.data, right_data);
}

typedef struct {
    float real;
    float imag;
} runtime_complex64;

typedef struct {
    double real;
    double imag;
} runtime_complex128;

static bool runtime_c64equal_impl(const void* left, const void* right) {
    const runtime_complex64* l = (const runtime_complex64*)left;
    const runtime_complex64* r = (const runtime_complex64*)right;
    if (l == NULL || r == NULL) {
        return false;
    }
    return l->real == r->real && l->imag == r->imag;
}

static bool runtime_c128equal_impl(const void* left, const void* right) {
    const runtime_complex128* l = (const runtime_complex128*)left;
    const runtime_complex128* r = (const runtime_complex128*)right;
    if (l == NULL || r == NULL) {
        return false;
    }
    return l->real == r->real && l->imag == r->imag;
}

static go_equal_function RUNTIME_USED runtime_c64equal_descriptor = runtime_c64equal_impl;
static go_equal_function RUNTIME_USED runtime_c128equal_descriptor = runtime_c128equal_impl;

go_interface runtime_getOverflowError(void) {
    go_interface err;

    err.methods = NULL;
    err.data = NULL;
    return err;
}

go_interface runtime_getDivideError(void) {
    go_interface err;

    err.methods = NULL;
    err.data = NULL;
    return err;
}

void runtime_register_gcroots(void* roots) {
    runtime_gc_root_block* block;
    runtime_gc_root_block* current;

    block = (runtime_gc_root_block*)roots;
    if (block == NULL) {
        return;
    }

    for (current = runtime_gc_roots; current != NULL; current = current->next) {
        if (current == block) {
            return;
        }
    }

    block->next = runtime_gc_roots;
    runtime_gc_roots = block;
}

void runtime_register_type_descriptors(const void* typelists, int count) {
    (void)typelists;
    (void)count;
}

static void RUNTIME_USED runtime_noop_import(void) {
}

static const unsigned char RUNTIME_USED runtime_empty_types[1] = {0};

void* memmove(void* dest, const void* src, size_t size) {
    if (dest == NULL || src == NULL) {
        return dest;
    }

    return kos_memmove(dest, src, size);
}

void* memcpy(void* dest, const void* src, size_t size) {
    if (dest == NULL || src == NULL) {
        return dest;
    }

    return kos_memcpy(dest, src, size);
}

void* memset(void* dest, int value, size_t size) {
    if (dest == NULL) {
        return NULL;
    }

    return kos_memset(dest, value, size);
}

size_t strlen(const char* str) {
    size_t len = 0;

    if (str == NULL) {
        return 0;
    }

    while (str[len] != '\0') {
        if (len == 4096) {
#if KOLIBRI_UNWIND_DEBUG
            runtime_debug_mark("strlen:long");
#endif
            return len;
        }
        len++;
    }
    return len;
}

__attribute__((noreturn)) void abort(void) {
    runtime_fail_simple("abort");
}

struct link_map;

#define DLFO_STRUCT_HAS_EH_DBASE 1
#define DLFO_STRUCT_HAS_EH_COUNT 0

typedef struct dl_find_object {
    unsigned long long dlfo_flags;
    void* dlfo_map_start;
    void* dlfo_map_end;
    struct link_map* dlfo_link_map;
    void* dlfo_eh_frame;
#if DLFO_STRUCT_HAS_EH_DBASE
    void* dlfo_eh_dbase;
#if __SIZEOF_POINTER__ == 4
    unsigned int __dlfo_eh_dbase_pad;
#endif
#endif
#if DLFO_STRUCT_HAS_EH_COUNT
    int dlfo_eh_count;
    unsigned int __dlfo_eh_count_pad;
#endif
    unsigned long long __dflo_reserved[7];
} dl_find_object;

extern char __eh_frame_hdr_start;

int _dl_find_object(void* address, dl_find_object* result) {
    if (result == NULL) {
        return -1;
    }

    (void)address;
    kos_memset(result, 0, sizeof(*result));
    result->dlfo_eh_frame = &__eh_frame_hdr_start;
#if DLFO_STRUCT_HAS_EH_DBASE
    result->dlfo_eh_dbase = &__eh_frame_hdr_start;
#endif
    return 0;
}

void* __unsafe_get_addr(void* base, size_t offset) {
    if (base == NULL) {
        return NULL;
    }

    return (void*)((unsigned char*)base + offset);
}

__asm__(".global runtime.memequal0..f");
static go_equal_function RUNTIME_USED runtime_memequal0_descriptor = runtime_memequal0_impl;
__asm__(".set runtime.memequal0..f, runtime_memequal0_descriptor");

__asm__(".global runtime.memequal32..f");
static go_equal_function RUNTIME_USED runtime_memequal32_descriptor = runtime_memequal32_impl;
__asm__(".set runtime.memequal32..f, runtime_memequal32_descriptor");

__asm__(".global runtime.memequal16..f");
static go_equal_function RUNTIME_USED runtime_memequal16_descriptor = runtime_memequal16_impl;
__asm__(".set runtime.memequal16..f, runtime_memequal16_descriptor");

__asm__(".global runtime.memequal8..f");
static go_equal_function RUNTIME_USED runtime_memequal8_descriptor = runtime_memequal8_impl;
__asm__(".set runtime.memequal8..f, runtime_memequal8_descriptor");

__asm__(".global runtime.memequal64..f");
static go_equal_function RUNTIME_USED runtime_memequal64_descriptor = runtime_memequal64_impl;
__asm__(".set runtime.memequal64..f, runtime_memequal64_descriptor");

__asm__(".global runtime.memequal128..f");
static go_equal_function RUNTIME_USED runtime_memequal128_descriptor = runtime_memequal128_impl;
__asm__(".set runtime.memequal128..f, runtime_memequal128_descriptor");

__asm__(".global runtime.c64equal..f");
__asm__(".set runtime.c64equal..f, runtime_c64equal_descriptor");

__asm__(".global runtime.c128equal..f");
__asm__(".set runtime.c128equal..f, runtime_c128equal_descriptor");

__asm__(".global runtime.memequal");
__asm__(".set runtime.memequal, runtime_memequal_impl");

__asm__(".global runtime.memequal0");
__asm__(".set runtime.memequal0, runtime_memequal0_impl");

__asm__(".global runtime.memequal64");
__asm__(".set runtime.memequal64, runtime_memequal64_impl");

__asm__(".global runtime.memequal32");
__asm__(".set runtime.memequal32, runtime_memequal32_impl");

__asm__(".global runtime.memequal16");
__asm__(".set runtime.memequal16, runtime_memequal16_impl");

__asm__(".global runtime.memequal8");
__asm__(".set runtime.memequal8, runtime_memequal8_impl");

__asm__(".global runtime.concatstrings");
__asm__(".set runtime.concatstrings, runtime_concatstrings");

__asm__(".global runtime.SetByteString");
__asm__(".set runtime.SetByteString, runtime_set_byte_string");

__asm__(".global runtime.writeBarrier");
__asm__(".set runtime.writeBarrier, runtime_write_barrier_enabled");

__asm__(".global runtime.gcWriteBarrier");
__asm__(".set runtime.gcWriteBarrier, runtime_gc_write_barrier");

__asm__(".global runtime.strequal..f");
static go_equal_function RUNTIME_USED runtime_strequal_descriptor = runtime_strequal_impl;
__asm__(".set runtime.strequal..f, runtime_strequal_descriptor");

__asm__(".global runtime.strequal");
__asm__(".set runtime.strequal, runtime_strequal_impl");

__asm__(".global runtime.memhash32..f");
static go_hash_function RUNTIME_USED runtime_memhash32_descriptor = runtime_memhash32_impl;
__asm__(".set runtime.memhash32..f, runtime_memhash32_descriptor");

__asm__(".global runtime.memhash8..f");
static go_hash_function RUNTIME_USED runtime_memhash8_descriptor = runtime_memhash8_impl;
__asm__(".set runtime.memhash8..f, runtime_memhash8_descriptor");

__asm__(".global runtime.memhash16..f");
static go_hash_function RUNTIME_USED runtime_memhash16_descriptor = runtime_memhash16_impl;
__asm__(".set runtime.memhash16..f, runtime_memhash16_descriptor");

__asm__(".global runtime.memhash64..f");
static go_hash_function RUNTIME_USED runtime_memhash64_descriptor = runtime_memhash64_impl;
__asm__(".set runtime.memhash64..f, runtime_memhash64_descriptor");

__asm__(".global runtime.strhash..f");
static go_hash_function RUNTIME_USED runtime_strhash_descriptor = runtime_strhash_impl;
__asm__(".set runtime.strhash..f, runtime_strhash_descriptor");

__asm__(".global runtime.memhash32");
__asm__(".set runtime.memhash32, runtime_memhash32");

__asm__(".global runtime.memhash8");
__asm__(".set runtime.memhash8, runtime_memhash8");

__asm__(".global runtime.memhash16");
__asm__(".set runtime.memhash16, runtime_memhash16");

__asm__(".global runtime.memhash64");
__asm__(".set runtime.memhash64, runtime_memhash64");

__asm__(".global runtime.memhash");
__asm__(".set runtime.memhash, runtime_memhash");

__asm__(".global runtime.strhash");
__asm__(".set runtime.strhash, runtime_strhash_impl");

__asm__(".global runtime.f32hash..f");
static go_hash_function RUNTIME_USED runtime_f32hash_descriptor = runtime_f32hash_impl;
__asm__(".set runtime.f32hash..f, runtime_f32hash_descriptor");

__asm__(".global runtime.f64hash..f");
static go_hash_function RUNTIME_USED runtime_f64hash_descriptor = runtime_f64hash_impl;
__asm__(".set runtime.f64hash..f, runtime_f64hash_descriptor");

__asm__(".global runtime.f32hash");
__asm__(".set runtime.f32hash, runtime_f32hash_impl");

__asm__(".global runtime.f64hash");
__asm__(".set runtime.f64hash, runtime_f64hash_impl");

__asm__(".global runtime.f32equal..f");
static go_equal_function RUNTIME_USED runtime_f32equal_descriptor = runtime_f32equal_impl;
__asm__(".set runtime.f32equal..f, runtime_f32equal_descriptor");

__asm__(".global runtime.f64equal..f");
static go_equal_function RUNTIME_USED runtime_f64equal_descriptor = runtime_f64equal_impl;
__asm__(".set runtime.f64equal..f, runtime_f64equal_descriptor");

__asm__(".global runtime.f32equal");
__asm__(".set runtime.f32equal, runtime_f32equal_impl");

__asm__(".global runtime.f64equal");
__asm__(".set runtime.f64equal, runtime_f64equal_impl");

__asm__(".global runtime.interhash..f");
static go_hash_function RUNTIME_USED runtime_interhash_descriptor = runtime_interhash_impl;
__asm__(".set runtime.interhash..f, runtime_interhash_descriptor");

__asm__(".global runtime.interhash");
__asm__(".set runtime.interhash, runtime_interhash_impl");

__asm__(".global runtime.nilinterhash..f");
static go_hash_function RUNTIME_USED runtime_nilinterhash_descriptor = runtime_nilinterhash_impl;
__asm__(".set runtime.nilinterhash..f, runtime_nilinterhash_descriptor");

__asm__(".global runtime.nilinterhash");
__asm__(".set runtime.nilinterhash, runtime_nilinterhash_impl");

__asm__(".global runtime.getg");
__asm__(".set runtime.getg, runtime_getg");

__asm__(".global runtime.deferprocStack");
__asm__(".set runtime.deferprocStack, runtime_deferprocStack");

__asm__(".global runtime.deferproc");
__asm__(".set runtime.deferproc, runtime_deferproc");

__asm__(".global runtime.deferreturn");
__asm__(".set runtime.deferreturn, runtime_deferreturn");

__asm__(".global runtime.checkdefer");
__asm__(".set runtime.checkdefer, runtime_checkdefer");

__asm__(".global runtime.canrecover");
__asm__(".set runtime.canrecover, runtime_canrecover");

__asm__(".global runtime.setdeferretaddr");
__asm__(".set runtime.setdeferretaddr, runtime_setdeferretaddr");

__asm__(".global runtime.gorecover");
__asm__(".set runtime.gorecover, runtime_gorecover");

__asm__(".global runtime.makemap__small");
__asm__(".set runtime.makemap__small, runtime_makemap__small");

__asm__(".global runtime.makemap");
__asm__(".set runtime.makemap, runtime_makemap");

__asm__(".global runtime.mapassign__fast32");
__asm__(".set runtime.mapassign__fast32, runtime_mapassign__fast32");

__asm__(".global runtime.mapassign__fast32ptr");
__asm__(".set runtime.mapassign__fast32ptr, runtime_mapassign__fast32ptr");

__asm__(".global runtime.mapassign__fast64");
__asm__(".set runtime.mapassign__fast64, runtime_mapassign__fast64");

__asm__(".global runtime.mapassign");
__asm__(".set runtime.mapassign, runtime_mapassign");

__asm__(".global runtime.mapassign__faststr");
__asm__(".set runtime.mapassign__faststr, runtime_mapassign__faststr");

__asm__(".global runtime.mapaccess1__fast32");
__asm__(".set runtime.mapaccess1__fast32, runtime_mapaccess1__fast32");

__asm__(".global runtime.mapaccess1__fast32ptr");
__asm__(".set runtime.mapaccess1__fast32ptr, runtime_mapaccess1__fast32ptr");

__asm__(".global runtime.mapaccess1__fast64");
__asm__(".set runtime.mapaccess1__fast64, runtime_mapaccess1__fast64");

__asm__(".global runtime.mapaccess1");
__asm__(".set runtime.mapaccess1, runtime_mapaccess1");

__asm__(".global runtime.mapaccess1__faststr");
__asm__(".set runtime.mapaccess1__faststr, runtime_mapaccess1__faststr");

__asm__(".global runtime.mapaccess2__fast32");
__asm__(".set runtime.mapaccess2__fast32, runtime_mapaccess2__fast32");

__asm__(".global runtime.mapaccess2__fast32ptr");
__asm__(".set runtime.mapaccess2__fast32ptr, runtime_mapaccess2__fast32ptr");

__asm__(".global runtime.mapaccess2__fast64");
__asm__(".set runtime.mapaccess2__fast64, runtime_mapaccess2__fast64");

__asm__(".global runtime.mapaccess2");
__asm__(".set runtime.mapaccess2, runtime_mapaccess2");

__asm__(".global runtime.mapaccess2__faststr");
__asm__(".set runtime.mapaccess2__faststr, runtime_mapaccess2__faststr");

__asm__(".global runtime.mapdelete__fast32");
__asm__(".set runtime.mapdelete__fast32, runtime_mapdelete__fast32");

__asm__(".global runtime.mapdelete__fast32ptr");
__asm__(".set runtime.mapdelete__fast32ptr, runtime_mapdelete__fast32ptr");

__asm__(".global runtime.mapdelete__fast64");
__asm__(".set runtime.mapdelete__fast64, runtime_mapdelete__fast64");

__asm__(".global runtime.mapdelete");
__asm__(".set runtime.mapdelete, runtime_mapdelete");

__asm__(".global runtime.mapdelete__faststr");
__asm__(".set runtime.mapdelete__faststr, runtime_mapdelete__faststr");

__asm__(".global runtime.mapclear");
__asm__(".set runtime.mapclear, runtime_mapclear");

__asm__(".global runtime.mapiterinit");
__asm__(".set runtime.mapiterinit, runtime_mapiterinit");

__asm__(".global runtime.mapiternext");
__asm__(".set runtime.mapiternext, runtime_mapiternext");

__asm__(".global runtime.ifaceeq");
__asm__(".set runtime.ifaceeq, runtime_ifaceeq");

__asm__(".global runtime.ifacevaleq");
__asm__(".set runtime.ifacevaleq, runtime_ifacevaleq");

__asm__(".global runtime.efaceeq");
__asm__(".set runtime.efaceeq, runtime_efaceeq");
__asm__(".global runtime.efacevaleq");
__asm__(".set runtime.efacevaleq, runtime_efacevaleq");

__asm__(".global runtime.ifaceE2T2");
__asm__(".set runtime.ifaceE2T2, runtime_ifaceE2T2");

__asm__(".global runtime.ifaceI2T2");
__asm__(".set runtime.ifaceI2T2, runtime_ifaceI2T2");

__asm__(".global runtime.ifaceE2T2P");
__asm__(".set runtime.ifaceE2T2P, runtime_ifaceE2T2P");

__asm__(".global runtime.ifaceI2T2P");
__asm__(".set runtime.ifaceI2T2P, runtime_ifaceI2T2P");

__asm__(".global runtime.ifaceT2Ip");
__asm__(".set runtime.ifaceT2Ip, runtime_ifaceT2Ip");

__asm__(".global runtime.assertitab");
__asm__(".set runtime.assertitab, runtime_assertitab");

__asm__(".global runtime.requireitab");
__asm__(".set runtime.requireitab, runtime_requireitab");

__asm__(".global runtime.ifaceE2I2");
__asm__(".set runtime.ifaceE2I2, runtime_ifaceE2I2");

__asm__(".global runtime.ifaceI2I2");
__asm__(".set runtime.ifaceI2I2, runtime_ifaceI2I2");

__asm__(".global runtime.interequal");
__asm__(".set runtime.interequal, runtime_interequal");

__asm__(".global runtime.interequal..f");
static go_equal_function RUNTIME_USED runtime_interequal_descriptor = runtime_interequal;
__asm__(".set runtime.interequal..f, runtime_interequal_descriptor");

__asm__(".global runtime.nilinterequal");
__asm__(".set runtime.nilinterequal, runtime_nilinterequal");

__asm__(".global runtime.nilinterequal..f");
static go_equal_function RUNTIME_USED runtime_nilinterequal_descriptor = runtime_nilinterequal;
__asm__(".set runtime.nilinterequal..f, runtime_nilinterequal_descriptor");

__asm__(".global runtime.newobject");
__asm__(".set runtime.newobject, runtime_newobject");

__asm__(".global runtime.makeslice");
__asm__(".set runtime.makeslice, runtime_makeslice");

__asm__(".global runtime.makeslice64");
__asm__(".set runtime.makeslice64, runtime_makeslice64");

__asm__(".global runtime.growslice");
__asm__(".set runtime.growslice, runtime_growslice");

__asm__(".global runtime.typedmemmove");
__asm__(".set runtime.typedmemmove, runtime_typedmemmove");

__asm__(".global runtime.typedslicecopy");
__asm__(".set runtime.typedslicecopy, runtime_typedslicecopy");

__asm__(".global runtime.slicebytetostring");
__asm__(".set runtime.slicebytetostring, runtime_slicebytetostring");

__asm__(".global runtime.stringtoslicebyte");
__asm__(".set runtime.stringtoslicebyte, runtime_stringtoslicebyte");
__asm__(".global runtime.stringtoslicerune");
__asm__(".set runtime.stringtoslicerune, runtime_stringtoslicerune");
__asm__(".global runtime.slicerunetostring");
__asm__(".set runtime.slicerunetostring, runtime_slicerunetostring");

__asm__(".global runtime.memmove");
__asm__(".set runtime.memmove, memmove");

__asm__(".global runtime.intstring");
__asm__(".set runtime.intstring, runtime_intstring");

__asm__(".global runtime.cmpstring");
__asm__(".set runtime.cmpstring, runtime_cmpstring");

__asm__(".global runtime.printlock");
__asm__(".set runtime.printlock, runtime_printlock");

__asm__(".global runtime.printunlock");
__asm__(".set runtime.printunlock, runtime_printunlock");

__asm__(".global runtime.printstring");
__asm__(".set runtime.printstring, runtime_printstring");

__asm__(".global runtime.printint");
__asm__(".set runtime.printint, runtime_printint");

__asm__(".global runtime.fastrand");
__asm__(".set runtime.fastrand, runtime_fastrand");

__asm__(".global runtime.getOverflowError");
__asm__(".set runtime.getOverflowError, runtime_getOverflowError");

__asm__(".global runtime.getDivideError");
__asm__(".set runtime.getDivideError, runtime_getDivideError");

__asm__(".global runtime.panicdottype");
__asm__(".set runtime.panicdottype, runtime_panicdottype");

__asm__(".global runtime.goPanicIndex");
__asm__(".set runtime.goPanicIndex, runtime_goPanicIndex");

__asm__(".global runtime.goPanicIndexU");
__asm__(".set runtime.goPanicIndexU, runtime_goPanicIndexU");

__asm__(".global runtime.goPanicSliceAlen");
__asm__(".set runtime.goPanicSliceAlen, runtime_goPanicSliceAlen");

__asm__(".global runtime.goPanicSliceAlenU");
__asm__(".set runtime.goPanicSliceAlenU, runtime_goPanicSliceAlenU");

__asm__(".global runtime.goPanicSliceAcap");
__asm__(".set runtime.goPanicSliceAcap, runtime_goPanicSliceAcap");

__asm__(".global runtime.goPanicSliceAcapU");
__asm__(".set runtime.goPanicSliceAcapU, runtime_goPanicSliceAcapU");

__asm__(".global runtime.goPanicSliceB");
__asm__(".set runtime.goPanicSliceB, runtime_goPanicSliceB");

__asm__(".global runtime.goPanicSliceBU");
__asm__(".set runtime.goPanicSliceBU, runtime_goPanicSliceBU");

__asm__(".global runtime.goPanicSlice3Alen");
__asm__(".set runtime.goPanicSlice3Alen, runtime_goPanicSlice3Alen");

__asm__(".global runtime.goPanicSlice3AlenU");
__asm__(".set runtime.goPanicSlice3AlenU, runtime_goPanicSlice3AlenU");

__asm__(".global runtime.goPanicSlice3Acap");
__asm__(".set runtime.goPanicSlice3Acap, runtime_goPanicSlice3Acap");

__asm__(".global runtime.goPanicSlice3AcapU");
__asm__(".set runtime.goPanicSlice3AcapU, runtime_goPanicSlice3AcapU");

__asm__(".global runtime.goPanicSlice3B");
__asm__(".set runtime.goPanicSlice3B, runtime_goPanicSlice3B");

__asm__(".global runtime.goPanicSlice3BU");
__asm__(".set runtime.goPanicSlice3BU, runtime_goPanicSlice3BU");

__asm__(".global runtime.goPanicSlice3C");
__asm__(".set runtime.goPanicSlice3C, runtime_goPanicSlice3C");

__asm__(".global runtime.goPanicSlice3CU");
__asm__(".set runtime.goPanicSlice3CU, runtime_goPanicSlice3CU");

__asm__(".global runtime.goPanicSliceConvert");
__asm__(".set runtime.goPanicSliceConvert, runtime_goPanicSliceConvert");
__asm__(".global runtime.goPanicExtendIndex");
__asm__(".set runtime.goPanicExtendIndex, runtime_goPanicExtendIndex");
__asm__(".global runtime.goPanicExtendIndexU");
__asm__(".set runtime.goPanicExtendIndexU, runtime_goPanicExtendIndexU");
__asm__(".global runtime.goPanicExtendSliceAcap");
__asm__(".set runtime.goPanicExtendSliceAcap, runtime_goPanicExtendSliceAcap");
__asm__(".global runtime.goPanicExtendSliceAlen");
__asm__(".set runtime.goPanicExtendSliceAlen, runtime_goPanicExtendSliceAlen");
__asm__(".global runtime.goPanicExtendSliceAcapU");
__asm__(".set runtime.goPanicExtendSliceAcapU, runtime_goPanicExtendSliceAcapU");
__asm__(".global runtime.goPanicExtendSliceAlenU");
__asm__(".set runtime.goPanicExtendSliceAlenU, runtime_goPanicExtendSliceAlenU");
__asm__(".global runtime.goPanicExtendSliceB");
__asm__(".set runtime.goPanicExtendSliceB, runtime_goPanicExtendSliceB");
__asm__(".global runtime.goPanicExtendSliceBU");
__asm__(".set runtime.goPanicExtendSliceBU, runtime_goPanicExtendSliceBU");

__asm__(".global runtime.panicmem");
__asm__(".set runtime.panicmem, runtime_panicmem");
__asm__(".global runtime.gopanic");
__asm__(".set runtime.gopanic, runtime_gopanic");
__asm__(".global runtime.panicdivide");
__asm__(".set runtime.panicdivide, runtime_panicdivide");
__asm__(".global runtime.decoderune");
__asm__(".set runtime.decoderune, runtime_decoderune");

__asm__(".global unsafe.Pointer..d");
__asm__(".set unsafe.Pointer..d, runtime_unsafe_pointer_descriptor");

__asm__(".global runtime.registerGCRoots");
__asm__(".set runtime.registerGCRoots, runtime_register_gcroots");

__asm__(".global runtime.registerTypeDescriptors");
__asm__(".set runtime.registerTypeDescriptors, runtime_register_type_descriptors");

__asm__(".global runtime..import");
__asm__(".set runtime..import, runtime_noop_import");

__asm__(".global internal_1cpu..import");
__asm__(".set internal_1cpu..import, runtime_noop_import");

__asm__(".global internal_1reflectlite..import");
__asm__(".set internal_1reflectlite..import, runtime_noop_import");

__asm__(".global internal_1oserror..import");
__asm__(".set internal_1oserror..import, runtime_noop_import");

__asm__(".global sync..import");
__asm__(".set sync..import, runtime_noop_import");

__asm__(".global internal_1unsafeheader..import");
__asm__(".set internal_1unsafeheader..import, runtime_noop_import");

__asm__(".global runtime..types");
__asm__(".set runtime..types, runtime_empty_types");

__asm__(".global internal_1cpu..types");
__asm__(".set internal_1cpu..types, runtime_empty_types");

__asm__(".global internal_1reflectlite..types");
__asm__(".set internal_1reflectlite..types, runtime_empty_types");

__asm__(".global internal_1oserror..types");
__asm__(".set internal_1oserror..types, runtime_empty_types");

__asm__(".global internal_1itoa..types");
__asm__(".set internal_1itoa..types, runtime_empty_types");

__asm__(".global internal_1race..types");
__asm__(".set internal_1race..types, runtime_empty_types");

__asm__(".global sync..types");
__asm__(".set sync..types, runtime_empty_types");

__asm__(".global sync_1atomic..types");
__asm__(".set sync_1atomic..types, runtime_empty_types");

__asm__(".global internal_1unsafeheader..types");
__asm__(".set internal_1unsafeheader..types, runtime_empty_types");

__asm__(".global internal_1abi..types");
__asm__(".set internal_1abi..types, runtime_empty_types");

__asm__(".global internal_1bytealg..types");
__asm__(".set internal_1bytealg..types, runtime_empty_types");

__asm__(".global internal_1goarch..types");
__asm__(".set internal_1goarch..types, runtime_empty_types");

__asm__(".global internal_1goexperiment..types");
__asm__(".set internal_1goexperiment..types, runtime_empty_types");

__asm__(".global internal_1goos..types");
__asm__(".set internal_1goos..types, runtime_empty_types");

__asm__(".global runtime_1internal_1atomic..types");
__asm__(".set runtime_1internal_1atomic..types, runtime_empty_types");

__asm__(".global runtime_1internal_1math..types");
__asm__(".set runtime_1internal_1math..types, runtime_empty_types");

__asm__(".global runtime_1internal_1sys..types");
__asm__(".set runtime_1internal_1sys..types, runtime_empty_types");

__asm__(".global runtime_1internal_1atomic.LoadAcquintptr");
__asm__(".set runtime_1internal_1atomic.LoadAcquintptr, runtime_internal_atomic_load_acquintptr");

__asm__(".global runtime_1internal_1atomic.StoreReluintptr");
__asm__(".set runtime_1internal_1atomic.StoreReluintptr, runtime_internal_atomic_store_reluintptr");
