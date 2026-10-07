/* Runtime adapters for unchanged Go 1.23 synchronization sources.
 * KolibriOS runtime semaphores and condition tickets yield cooperatively.
 * On 386, int and int32 have the same ABI. */
#include <stdint.h>
#include <stdbool.h>

typedef struct {
    const unsigned char* str;
    intptr_t len;
} sync_string;
extern void throw(sync_string) __attribute__((noreturn));
extern void runtime_Semacquire(uint32_t*);
extern void runtime_SemacquireMutex(uint32_t*, bool, int32_t);
extern void runtime_Semrelease(uint32_t*, bool, int32_t);
extern uint32_t runtime_notifyListAdd(void*);
extern void runtime_notifyListWait(void*, uint32_t);
extern void runtime_notifyListNotifyAll(void*);
extern void runtime_notifyListNotifyOne(void*);
extern void runtime_notifyListCheck(uintptr_t);
extern bool runtime_canSpin(int32_t);
extern void runtime_doSpin(void);
extern int64_t runtime_nanotime(void);

void sync_throw(sync_string s) __asm__("sync.throw") __attribute__((noreturn));
void sync_throw(sync_string s) { throw(s); }
void sync_fatal(sync_string s) __asm__("sync.fatal") __attribute__((noreturn));
void sync_fatal(sync_string s) { throw(s); }

void sync_Semacquire(uint32_t* p) __asm__("sync.runtime__Semacquire");
void sync_Semacquire(uint32_t* p) { runtime_Semacquire(p); }
#define SYNC_SEMACQUIRE(Name) \
void sync_##Name(uint32_t*, bool, int32_t) __asm__("sync.runtime__" #Name); \
void sync_##Name(uint32_t* p, bool lifo, int32_t skip) { runtime_SemacquireMutex(p, lifo, skip); }
SYNC_SEMACQUIRE(SemacquireMutex)
SYNC_SEMACQUIRE(SemacquireRWMutexR)
SYNC_SEMACQUIRE(SemacquireRWMutex)
void sync_Semrelease(uint32_t*, bool, int32_t) __asm__("sync.runtime__Semrelease");
void sync_Semrelease(uint32_t* p, bool handoff, int32_t skip) { runtime_Semrelease(p, handoff, skip); }
uint32_t sync_notifyListAdd(void*) __asm__("sync.runtime__notifyListAdd");
uint32_t sync_notifyListAdd(void* l) { return runtime_notifyListAdd(l); }
void sync_notifyListWait(void*, uint32_t) __asm__("sync.runtime__notifyListWait");
void sync_notifyListWait(void* l, uint32_t t) { runtime_notifyListWait(l, t); }
#define SYNC_NOTIFY(Name) \
void sync_##Name(void*) __asm__("sync.runtime__" #Name); \
void sync_##Name(void* l) { runtime_##Name(l); }
SYNC_NOTIFY(notifyListNotifyAll)
SYNC_NOTIFY(notifyListNotifyOne)
void sync_notifyListCheck(uintptr_t) __asm__("sync.runtime__notifyListCheck");
void sync_notifyListCheck(uintptr_t size) { runtime_notifyListCheck(size); }
bool sync_canSpin(int32_t) __asm__("sync.runtime__canSpin");
bool sync_canSpin(int32_t n) { return runtime_canSpin(n); }
void sync_doSpin(void) __asm__("sync.runtime__doSpin");
void sync_doSpin(void) { runtime_doSpin(); }
int64_t sync_nanotime(void) __asm__("sync.runtime__nanotime");
int64_t sync_nanotime(void) { return runtime_nanotime(); }
