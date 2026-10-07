/* Copyright 2009 The Go Authors. All rights reserved.
 * Use of this source code is governed by a BSD-style license.
 * Adapted from libgo runtime/sema.go's acquire/release and notifyList
 * protocols. Platform wait records live on parked goroutine stacks; hashed
 * linear queues replace libgo's profiling treap and use the SDK scheduler. */

typedef struct runtime_sync_wait {
    runtime_g* g;
    uint32_t* address;
    struct runtime_sync_wait* next;
    uint32_t ticket;
} runtime_sync_wait;

typedef struct {
    runtime_mutex lock;
    runtime_sync_wait* head;
    runtime_sync_wait* tail;
} runtime_sema_root;

#define RUNTIME_SEMA_BUCKETS 251u
static runtime_sema_root runtime_sema_roots[RUNTIME_SEMA_BUCKETS];

static runtime_sema_root* runtime_semroot(uint32_t* address) {
    return &runtime_sema_roots[((uintptr_t)address >> 3) % RUNTIME_SEMA_BUCKETS];
}

static bool runtime_cansemacquire(uint32_t* address) {
    for (;;) {
        uint32_t value = runtime_atomic_load_u32(address);
        if (value == 0) return false;
        if (runtime_atomic_cas_u32(address, value, value - 1)) return true;
    }
}

static void runtime_semacquire_impl(uint32_t* address, bool lifo) {
    if (address == NULL) return;
    if (runtime_cansemacquire(address)) return;
    runtime_g* g = runtime_getg();
    runtime_m* m = runtime_getm();
    if (g == NULL || m == NULL || g == m->g0) {
        while (!runtime_cansemacquire(address)) runtime_yield();
        return;
    }
    runtime_sema_root* root = runtime_semroot(address);
    runtime_sync_wait waiter = {g, address, NULL, 0};
    for (;;) {
        runtime_lock_mutex(&root->lock);
        /* Recheck after locking so a release cannot be lost between the
         * initial fast path and queue insertion. */
        if (runtime_cansemacquire(address)) {
            runtime_unlock_mutex(&root->lock);
            return;
        }
        waiter.next = NULL;
        waiter.ticket = 0;
        if (lifo) {
            waiter.next = root->head;
            root->head = &waiter;
            if (root->tail == NULL) root->tail = &waiter;
        } else {
            if (root->tail != NULL) root->tail->next = &waiter;
            else root->head = &waiter;
            root->tail = &waiter;
        }
        g->status = RUNTIME_G_WAITING;
        runtime_atomic_store_u32(&g->parking, 1);
        runtime_unlock_mutex(&root->lock);
        runtime_gopark_after_unlock(g, m);
        if (runtime_atomic_load_u32(&waiter.ticket) != 0 || runtime_cansemacquire(address)) return;
        /* A resumed goroutine may have migrated to another OS thread. */
        m = runtime_getm();
    }
}

void runtime_Semacquire(uint32_t* address) {
    runtime_semacquire_impl(address, false);
}

void runtime_SemacquireMutex(uint32_t* address, bool lifo, int32_t skipframes) {
    (void)skipframes;
    runtime_semacquire_impl(address, lifo);
}

void runtime_Semrelease(uint32_t* address, bool handoff, int32_t skipframes) {
    (void)skipframes;
    if (address == NULL) return;
    runtime_atomic_xadd_u32(address, 1);
    runtime_sema_root* root = runtime_semroot(address);
    runtime_lock_mutex(&root->lock);
    runtime_sync_wait* previous = NULL;
    runtime_sync_wait* waiter = root->head;
    while (waiter != NULL && waiter->address != address) {
        previous = waiter;
        waiter = waiter->next;
    }
    runtime_g* g = NULL;
    bool transfer = false;
    if (waiter != NULL) {
        if (previous != NULL) previous->next = waiter->next;
        else root->head = waiter->next;
        if (root->tail == waiter) root->tail = previous;
        g = waiter->g;
        transfer = handoff && runtime_cansemacquire(address);
        if (transfer) runtime_atomic_store_u32(&waiter->ticket, 1);
    }
    runtime_unlock_mutex(&root->lock);
    if (g != NULL) {
        /* Never access the stack wait record after ready. */
        runtime_ready(g);
        if (transfer) runtime_gosched_internal();
    }
}

static inline bool runtime_notify_less(uint32_t left, uint32_t right) {
    return ((int32_t)(left - right)) < 0;
}

uint32_t runtime_notifyListAdd(runtime_notify_list* list) {
    return runtime_atomic_xadd_u32(&list->wait, 1);
}

void runtime_notifyListWait(runtime_notify_list* list, uint32_t ticket) {
    runtime_g* g = runtime_getg();
    runtime_m* m = runtime_getm();
    runtime_mutex* lock = (runtime_mutex*)&list->lock;
    runtime_lock_mutex(lock);
    if (runtime_notify_less(ticket, runtime_atomic_load_u32(&list->notify))) {
        runtime_unlock_mutex(lock);
        return;
    }
    if (g == NULL || m == NULL || g == m->g0) {
        runtime_unlock_mutex(lock);
        runtime_fail_simple("condition wait outside goroutine");
    }
    runtime_sync_wait waiter = {g, NULL, NULL, ticket};
    runtime_sync_wait* tail = (runtime_sync_wait*)list->tail;
    if (tail != NULL) tail->next = &waiter;
    else list->head = &waiter;
    list->tail = &waiter;
    g->status = RUNTIME_G_WAITING;
    runtime_atomic_store_u32(&g->parking, 1);
    runtime_unlock_mutex(lock);
    runtime_gopark_after_unlock(g, m);
}

void runtime_notifyListNotifyAll(runtime_notify_list* list) {
    if (runtime_atomic_load_u32(&list->wait) == runtime_atomic_load_u32(&list->notify)) return;
    runtime_mutex* lock = (runtime_mutex*)&list->lock;
    runtime_lock_mutex(lock);
    runtime_sync_wait* waiter = (runtime_sync_wait*)list->head;
    list->head = NULL;
    list->tail = NULL;
    runtime_atomic_store_u32(&list->notify, runtime_atomic_load_u32(&list->wait));
    runtime_unlock_mutex(lock);
    while (waiter != NULL) {
        runtime_sync_wait* next = waiter->next;
        runtime_g* g = waiter->g;
        runtime_ready(g);
        waiter = next;
    }
}

void runtime_notifyListNotifyOne(runtime_notify_list* list) {
    if (runtime_atomic_load_u32(&list->wait) == runtime_atomic_load_u32(&list->notify)) return;
    runtime_mutex* lock = (runtime_mutex*)&list->lock;
    runtime_lock_mutex(lock);
    uint32_t ticket = runtime_atomic_load_u32(&list->notify);
    if (ticket == runtime_atomic_load_u32(&list->wait)) {
        runtime_unlock_mutex(lock);
        return;
    }
    runtime_atomic_store_u32(&list->notify, ticket + 1);
    runtime_sync_wait* previous = NULL;
    runtime_sync_wait* waiter = (runtime_sync_wait*)list->head;
    while (waiter != NULL && waiter->ticket != ticket) {
        previous = waiter;
        waiter = waiter->next;
    }
    runtime_g* g = NULL;
    if (waiter != NULL) {
        if (previous != NULL) previous->next = waiter->next;
        else list->head = waiter->next;
        if (list->tail == waiter) list->tail = previous;
        g = waiter->g;
    }
    runtime_unlock_mutex(lock);
    if (g != NULL) runtime_ready(g);
}
