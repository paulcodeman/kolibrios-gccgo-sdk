/* Copyright 2009 The Go Authors. All rights reserved.
 * Use of this source code is governed by a BSD-style license.
 * Adapt libgo runtime/mfinal.go registration and queueing to the SDK's
 * managed allocation headers, stop-the-world collector and Go scheduler. */
#ifndef KOLIBRI_RUNTIME_FINALIZER_H
#define KOLIBRI_RUNTIME_FINALIZER_H

typedef struct {
    go_type_descriptor common;
    const go_type_descriptor* elem;
} runtime_finalizer_pointer_type;

typedef struct {
    go_type_descriptor common;
    bool variadic;
    go_slice in, out;
} runtime_finalizer_function_type;

typedef struct runtime_finalizer_record {
    struct runtime_finalizer_record* next;
    go_empty_interface object, function;
    uint8_t candidate;
} runtime_finalizer_record;

static runtime_finalizer_record* runtime_finalizers;
static runtime_finalizer_record* runtime_finalizers_ready;
static runtime_finalizer_record* runtime_finalizer_active;
static uint32_t runtime_finalizer_worker_running;
static uint32_t runtime_finalizer_pending;

/* This is the existing, upstream-derived reflect FFI backend. OpenCode
 * already links reflect; an SDK application using finalizers must also
 * link reflect so the complete function ABI conversion is available. */
extern void runtime_finalizer_reflect_call(const void*, void*, bool, bool, void**, void**)
    __asm__("reflect_call") __attribute__((weak));

void runtime_set_finalizer(go_empty_interface object, go_empty_interface function)
    __asm__("runtime.SetFinalizer");
void runtime_set_finalizer(go_empty_interface object, go_empty_interface function) {
    const runtime_finalizer_pointer_type* pointer_type;
    const runtime_finalizer_function_type* function_type;
    const go_type_descriptor* argument_type;
    runtime_gc_header* header;
    runtime_finalizer_record** link;
    runtime_finalizer_record* record;

    if (object.type == NULL || (object.type->kind & GO_TYPE_KIND_MASK) != 22) {
        runtime_fail_simple("runtime.SetFinalizer: first argument is not a pointer");
    }
    pointer_type = (const runtime_finalizer_pointer_type*)object.type;
    if (pointer_type->elem == NULL) {
        runtime_fail_simple("runtime.SetFinalizer: nil element type");
    }
    runtime_lock_mutex(&runtime_gc_lock);
    header = runtime_gc_find_header_for_address(object.data);
    if (header == NULL || pointer_type->elem->size == 0) {
        /* Upstream gccgo also accepts linker-allocated and zero-sized
         * objects without installing a finalizer. */
        runtime_unlock_mutex(&runtime_gc_lock);
        return;
    }
    if (object.data != runtime_gc_payload(header)
        && (pointer_type->elem->ptrdata != 0 || pointer_type->elem->size >= RUNTIME_TINY_SIZE)) {
        runtime_fail_simple("runtime.SetFinalizer: pointer is not at allocation start");
    }
    link = &runtime_finalizers;
    while (*link != NULL && (*link)->object.data != object.data) {
        link = &(*link)->next;
    }
    if (function.type == NULL) {
        record = *link;
        if (record != NULL) {
            *link = record->next;
            free(record);
        }
        runtime_unlock_mutex(&runtime_gc_lock);
        return;
    }
    if ((function.type->kind & GO_TYPE_KIND_MASK) != 19) {
        runtime_fail_simple("runtime.SetFinalizer: second argument is not a function");
    }
    function_type = (const runtime_finalizer_function_type*)function.type;
    if (function_type->variadic || function_type->in.len != 1) {
        runtime_fail_simple("runtime.SetFinalizer: invalid function parameters");
    }
    argument_type = ((const go_type_descriptor* const*)function_type->in.values)[0];
    if (!runtime_type_descriptor_matches(argument_type, object.type)) {
        bool compatible = false;
        if ((argument_type->kind & GO_TYPE_KIND_MASK) == 22) {
            const runtime_finalizer_pointer_type* argument_pointer =
                (const runtime_finalizer_pointer_type*)argument_type;
            compatible = (argument_type->uncommon == NULL || object.type->uncommon == NULL)
                && runtime_type_descriptor_matches(argument_pointer->elem, pointer_type->elem);
        } else if ((argument_type->kind & GO_TYPE_KIND_MASK) == GO_TYPE_KIND_INTERFACE) {
            compatible = runtime_ifaceT2Ip(argument_type, object.type);
        }
        if (!compatible) {
            runtime_fail_simple("runtime.SetFinalizer: incompatible argument type");
        }
    }
    if (*link != NULL) {
        runtime_fail_simple("runtime.SetFinalizer: finalizer already set");
    }
    if (runtime_finalizer_reflect_call == NULL) {
        runtime_fail_simple("runtime.SetFinalizer: reflect FFI backend must be linked");
    }
    record = (runtime_finalizer_record*)malloc(sizeof(*record));
    if (record == NULL) {
        runtime_panicmem();
    }
    record->object = object;
    record->function = function;
    record->candidate = 0;
    record->next = runtime_finalizers;
    runtime_finalizers = record;
    runtime_unlock_mutex(&runtime_gc_lock);
}

static void runtime_finalizer_mark_roots(void) {
    runtime_finalizer_record* record;
    for (record = runtime_finalizers; record != NULL; record = record->next) {
        /* Keep closures alive without retaining the finalizable object. */
        runtime_gc_mark_pointer(record->function.data);
    }
    for (record = runtime_finalizers_ready; record != NULL; record = record->next) {
        runtime_gc_mark_pointer(record->object.data);
        runtime_gc_mark_pointer(record->function.data);
    }
    if (runtime_finalizer_active != NULL) {
        runtime_gc_mark_pointer(runtime_finalizer_active->object.data);
        runtime_gc_mark_pointer(runtime_finalizer_active->function.data);
    }
}

static void runtime_finalizer_prepare_queue(void) {
    runtime_finalizer_record* record;
    runtime_finalizer_record** link;
    /* Trace dependencies before selecting callbacks. If A references B,
     * B must survive A's finalization and cannot be queued in this cycle. */
    for (record = runtime_finalizers; record != NULL; record = record->next) {
        runtime_gc_header* header = runtime_gc_find_header_for_address(record->object.data);
        if (header != NULL && header->marked != runtime_gc_mark_token && header->scan != NULL) {
            header->scan(header);
        }
    }
    for (record = runtime_finalizers; record != NULL; record = record->next) {
        runtime_gc_header* header = runtime_gc_find_header_for_address(record->object.data);
        record->candidate = header != NULL && header->marked != runtime_gc_mark_token;
    }
    link = &runtime_finalizers;
    while (*link != NULL) {
        record = *link;
        if (!record->candidate) {
            link = &record->next;
            continue;
        }
        *link = record->next;
        record->next = runtime_finalizers_ready;
        runtime_finalizers_ready = record;
        runtime_atomic_store_u32(&runtime_finalizer_pending, 1);
        runtime_gc_mark_pointer(record->object.data);
    }
}

static void runtime_finalizer_worker(void* unused) {
    (void)unused;
    for (;;) {
        runtime_finalizer_record* record;
        const runtime_finalizer_function_type* function_type;
        const go_type_descriptor* argument_type;
        go_empty_interface empty_argument;
        go_interface interface_argument;
        void* pointer_argument;
        void* parameters[1];
        runtime_lock_mutex(&runtime_gc_lock);
        record = runtime_finalizers_ready;
        if (record == NULL) {
            runtime_atomic_store_u32(&runtime_finalizer_worker_running, 0);
            runtime_atomic_store_u32(&runtime_finalizer_pending, 0);
            runtime_unlock_mutex(&runtime_gc_lock);
            return;
        }
        runtime_finalizers_ready = record->next;
        runtime_finalizer_active = record;
        runtime_unlock_mutex(&runtime_gc_lock);
        function_type = (const runtime_finalizer_function_type*)record->function.type;
        argument_type = ((const go_type_descriptor* const*)function_type->in.values)[0];
        if ((argument_type->kind & GO_TYPE_KIND_MASK) == GO_TYPE_KIND_INTERFACE) {
            const go_interface_type_descriptor* interface_type =
                (const go_interface_type_descriptor*)argument_type;
            if (interface_type->method_count == 0) {
                empty_argument = record->object;
                parameters[0] = &empty_argument;
            } else {
                interface_argument.methods = runtime_assertitab(argument_type, record->object.type);
                interface_argument.data = record->object.data;
                parameters[0] = &interface_argument;
            }
        } else {
            pointer_argument = (void*)record->object.data;
            parameters[0] = &pointer_argument;
        }
        runtime_finalizer_reflect_call(function_type, (void*)record->function.data,
                                      false, false, parameters, NULL);
        runtime_lock_mutex(&runtime_gc_lock);
        runtime_finalizer_active = NULL;
        free(record);
        runtime_unlock_mutex(&runtime_gc_lock);
    }
}

static void runtime_finalizer_maybe_start(void) {
    bool start = false;
    if (!runtime_atomic_load_u32(&runtime_finalizer_pending)
        || runtime_atomic_load_u32(&runtime_finalizer_worker_running)) {
        return;
    }
    runtime_lock_mutex(&runtime_gc_lock);
    if (runtime_finalizers_ready != NULL && !runtime_finalizer_worker_running) {
        runtime_finalizer_worker_running = 1;
        start = true;
    }
    runtime_unlock_mutex(&runtime_gc_lock);
    if (start && __go_go((uintptr_t)&runtime_finalizer_worker, NULL) == NULL) {
        runtime_fail_simple("runtime.SetFinalizer: cannot start worker");
    }
}

#endif
