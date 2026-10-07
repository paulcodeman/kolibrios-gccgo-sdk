/* Copyright 2011 The Go Authors. All rights reserved.
 * Use of this source code is governed by a BSD-style license.
 * Adapt libgo atomic.c's sequentially consistent primitives to Go 1.23
 * And/Or and pointer operations. The SDK collector stops the world and
 * does not require a concurrent write barrier on pointer publication. */
#include <stdint.h>
#include <stdbool.h>

extern void panicUnaligned(void) __asm__("sync_1atomic.panicUnaligned") __attribute__((noreturn));

int32_t AndInt32(int32_t*, int32_t) __asm__("sync_1atomic.AndInt32");
int32_t AndInt32(int32_t* addr, int32_t mask) {  return __atomic_fetch_and(addr, mask, __ATOMIC_SEQ_CST); }

int32_t OrInt32(int32_t*, int32_t) __asm__("sync_1atomic.OrInt32");
int32_t OrInt32(int32_t* addr, int32_t mask) {  return __atomic_fetch_or(addr, mask, __ATOMIC_SEQ_CST); }

uint32_t AndUint32(uint32_t*, uint32_t) __asm__("sync_1atomic.AndUint32");
uint32_t AndUint32(uint32_t* addr, uint32_t mask) {  return __atomic_fetch_and(addr, mask, __ATOMIC_SEQ_CST); }

uint32_t OrUint32(uint32_t*, uint32_t) __asm__("sync_1atomic.OrUint32");
uint32_t OrUint32(uint32_t* addr, uint32_t mask) {  return __atomic_fetch_or(addr, mask, __ATOMIC_SEQ_CST); }

int64_t AndInt64(int64_t*, int64_t) __asm__("sync_1atomic.AndInt64");
int64_t AndInt64(int64_t* addr, int64_t mask) { if (((uintptr_t)addr & 7) != 0) panicUnaligned(); return __atomic_fetch_and(addr, mask, __ATOMIC_SEQ_CST); }

int64_t OrInt64(int64_t*, int64_t) __asm__("sync_1atomic.OrInt64");
int64_t OrInt64(int64_t* addr, int64_t mask) { if (((uintptr_t)addr & 7) != 0) panicUnaligned(); return __atomic_fetch_or(addr, mask, __ATOMIC_SEQ_CST); }

uint64_t AndUint64(uint64_t*, uint64_t) __asm__("sync_1atomic.AndUint64");
uint64_t AndUint64(uint64_t* addr, uint64_t mask) { if (((uintptr_t)addr & 7) != 0) panicUnaligned(); return __atomic_fetch_and(addr, mask, __ATOMIC_SEQ_CST); }

uint64_t OrUint64(uint64_t*, uint64_t) __asm__("sync_1atomic.OrUint64");
uint64_t OrUint64(uint64_t* addr, uint64_t mask) { if (((uintptr_t)addr & 7) != 0) panicUnaligned(); return __atomic_fetch_or(addr, mask, __ATOMIC_SEQ_CST); }

uintptr_t AndUintptr(uintptr_t*, uintptr_t) __asm__("sync_1atomic.AndUintptr");
uintptr_t AndUintptr(uintptr_t* addr, uintptr_t mask) {  return __atomic_fetch_and(addr, mask, __ATOMIC_SEQ_CST); }

uintptr_t OrUintptr(uintptr_t*, uintptr_t) __asm__("sync_1atomic.OrUintptr");
uintptr_t OrUintptr(uintptr_t* addr, uintptr_t mask) {  return __atomic_fetch_or(addr, mask, __ATOMIC_SEQ_CST); }

void* SwapPointer(void**, void*) __asm__("sync_1atomic.SwapPointer");
void* SwapPointer(void** addr, void* value) { return __atomic_exchange_n(addr, value, __ATOMIC_SEQ_CST); }

_Bool CompareAndSwapPointer(void**, void*, void*) __asm__("sync_1atomic.CompareAndSwapPointer");
_Bool CompareAndSwapPointer(void** addr, void* old, void* value) { return __atomic_compare_exchange_n(addr, &old, value, false, __ATOMIC_SEQ_CST, __ATOMIC_RELAXED); }

void StorePointer(void**, void*) __asm__("sync_1atomic.StorePointer");
void StorePointer(void** addr, void* value) { __atomic_store_n(addr, value, __ATOMIC_SEQ_CST); }
