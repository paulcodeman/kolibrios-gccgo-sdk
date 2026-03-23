#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "runtime.h"

#undef runtime_exit

extern char __end;
extern uint32_t runtime_kos_load_dll_cstring_raw(const char* path);
extern uint32_t runtime_kos_heap_init_raw(void);
extern uint32_t runtime_kos_heap_alloc_raw(uint32_t size);
extern uint32_t runtime_kos_heap_free_raw(uint32_t ptr);
extern uint32_t runtime_kos_heap_realloc_raw(uint32_t size, uint32_t ptr);
extern int32_t runtime_kos_create_thread_raw(uint32_t entry, uint32_t stack) __asm__("go_0kos.CreateThreadRaw");
extern int32_t runtime_kos_get_current_thread_slot_raw(void) __asm__("go_0kos.GetCurrentThreadSlotRaw");
extern void runtime_ginit(void);
extern void runtime_check(void);
extern void runtime_args(int32 argc, byte** argv);
extern void runtime_osinit(void);
extern void runtime_schedinit(void);
extern void runtime_cpuinit(void);
extern void* runtime_mstart(void* arg);
extern void runtime_main(void*);
extern G* allocg(void) __asm__(GOSYM_PREFIX "runtime.allocg");
extern void kickoff(void) __asm__(GOSYM_PREFIX "runtime.kickoff");
extern void minit(void) __asm__(GOSYM_PREFIX "runtime.minit");
extern void mstart1(void) __asm__(GOSYM_PREFIX "runtime.mstart1");
extern void mexit(bool) __asm__(GOSYM_PREFIX "runtime.mexit");
extern void gothrow(String) __attribute__((noreturn));
extern void gothrow(String) __asm__(GOSYM_PREFIX "runtime.throw");
extern void kos_thread_bootstrap(void* arg) __asm__("kos.ThreadBootstrap");
extern void runtime_swapcontext(void* from, void* to);
extern void runtime_loadcontext(void* ctx);
extern void runtime_libgo_thread_entry(void);

typedef struct {
  const char* name;
  void* data;
} kos_dll_export;

typedef struct {
  uint32_t imports;
  const char* library_name;
} kos_dll_import_library;

struct caller_ret {
  uintptr_t pc;
  String file;
  intgo line;
  _Bool ok;
};

typedef struct {
  uintptr_t ebx;
  uintptr_t esi;
  uintptr_t edi;
  uintptr_t ebp;
  uintptr_t esp;
  uintptr_t eip;
} runtime_libgo_context;

#if defined(__i386__)
#define KOS_STDCALL __attribute__((stdcall))
#else
#define KOS_STDCALL
#endif

typedef uint32_t (KOS_STDCALL *kos_stdcall0_fn)(void);
typedef uint32_t (KOS_STDCALL *kos_stdcall1_fn)(uint32_t);
typedef uint32_t (KOS_STDCALL *kos_stdcall2_fn)(uint32_t, uint32_t);
typedef uint32_t (KOS_STDCALL *kos_stdcall3_fn)(uint32_t, uint32_t, uint32_t);
typedef uint32_t (KOS_STDCALL *kos_stdcall4_fn)(uint32_t, uint32_t, uint32_t, uint32_t);
typedef uint32_t (KOS_STDCALL *kos_stdcall5_fn)(uint32_t, uint32_t, uint32_t, uint32_t, uint32_t);
typedef uint32_t (KOS_STDCALL *kos_stdcall6_fn)(uint32_t, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t);
typedef uint32_t (KOS_STDCALL *kos_stdcall7_fn)(uint32_t, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t);
typedef void (KOS_STDCALL *kos_stdcall1_void_fn)(uint32_t);
typedef void (KOS_STDCALL *kos_stdcall2_void_fn)(uint32_t, uint32_t);
typedef void (KOS_STDCALL *kos_stdcall5_void_fn)(uint32_t, uint32_t, uint32_t, uint32_t, uint32_t);
typedef uint32_t (*kos_cdecl2_fn)(uint32_t, uint32_t);
typedef uint32_t (*kos_cdecl5_fn)(uint32_t, uint32_t, uint32_t, uint32_t, uint32_t);

bool runtime_isarchive;
bool runtime_isstarted;
_Bool runtime_iscgo;
uintptr __go_end;

static uint32_t runtime_libgo_threads = 1;
static uint32_t runtime_console_bridge_table;
static uint32_t runtime_console_bridge_write_proc;
static uint32_t runtime_console_bridge_exit_proc;
static uint32_t runtime_console_bridge_gets_proc;
static uint32_t runtime_kos_heap_initialized;
static uint64_t runtime_kos_heap_alloc_count;
static uint64_t runtime_kos_heap_alloc_bytes;
static uint64_t runtime_kos_heap_free_count;
static uint64_t runtime_kos_heap_realloc_count;
static uint64_t runtime_kos_heap_realloc_bytes;
static char* runtime_window_title_buffer;
static size_t runtime_window_title_capacity;
uint32 __go_runtime_in_callers;
bool runtime_usestackmaps;
uintptr runtime_stacks_sys;

#define RUNTIME_LIBGO_MAX_THREAD_SLOTS 1024u
#define RUNTIME_LIBGO_STACK_MIN ((sizeof(void*) < 8) ? (2u * 1024u * 1024u) : (4u * 1024u * 1024u))
#define RUNTIME_LIBGO_SIGNAL_STACK (32u * 1024u)
#define RUNTIME_LIBGO_THREAD_STACK (64u * 1024u)

static G* runtime_bridge_g_fallback;
static G* runtime_bridge_g_slots[RUNTIME_LIBGO_MAX_THREAD_SLOTS];

static size_t
kos_strlen(const char* value)
{
  return value == NULL ? 0u : strlen(value);
}

static int
kos_strcmp(const char* left, const char* right)
{
  if (left == right)
    return 0;
  if (left == NULL)
    return -1;
  if (right == NULL)
    return 1;
  return strcmp(left, right);
}

static void*
runtime_alloc_zeroed(size_t size)
{
  void* ptr;

  if (size == 0)
    size = 1;
  ptr = malloc(size);
  if (ptr != NULL)
    memset(ptr, 0, size);
  return ptr;
}

static runtime_libgo_context*
runtime_libgo_gctx(G* gp)
{
  if (gp == nil)
    return NULL;
  return (runtime_libgo_context*) (void*) &gp->stackcontext[0];
}

static uintptr_t
runtime_libgo_prepare_stack(uintptr_t top)
{
  top &= ~(uintptr_t) 0xFu;
  if (top < sizeof(uintptr_t))
    return 0;
  top -= sizeof(uintptr_t);
  *(uintptr_t*) top = 0;
  return top;
}

static void
runtime_libgo_init_context(runtime_libgo_context* ctx, uintptr_t sp, void (*entry)(void))
{
  if (ctx == NULL)
    return;
  memset(ctx, 0, sizeof(*ctx));
  ctx->esp = sp;
  ctx->eip = (uintptr_t) entry;
}

__attribute__((noreturn)) static void
runtime_libgo_bad_return(void)
{
  runtime_throw("runtime: unexpected context return");
}

static uintptr_t
runtime_libgo_make_stack_top(void* base, uintptr_t size)
{
  uintptr_t sp;

  if (base == NULL || size < 16u)
    return 0;
  sp = runtime_libgo_prepare_stack((uintptr_t) base + size);
  if (sp == 0)
    return 0;
  *(uintptr_t*) sp = (uintptr_t) &runtime_libgo_bad_return;
  return sp;
}

static uintptr_t
runtime_libgo_current_stack_top(void* anchor)
{
  uintptr_t sp;

  if (anchor == NULL)
    return 0;
  sp = runtime_libgo_prepare_stack((uintptr_t) anchor);
  if (sp == 0)
    return 0;
  *(uintptr_t*) sp = (uintptr_t) &runtime_libgo_bad_return;
  return sp;
}

static uint32_t
runtime_libgo_thread_stack_pointer(void* stack_base, size_t size, void* arg)
{
  uintptr_t top;

  if (stack_base == NULL || size < 16u)
    return 0;
  top = (uintptr_t) stack_base + size;
  top &= ~(uintptr_t) 0xFu;
  if (top < (uintptr_t) stack_base + sizeof(uint32_t))
    return 0;
  top -= sizeof(uint32_t);
  *(uint32_t*) top = (uint32_t) (uintptr_t) arg;
  return (uint32_t) top;
}

__attribute__((noreturn)) static void
runtime_libgo_invoke_entry(void)
{
  G* gp;
  FuncVal* fv;
  void (*pfn)(G*);
  G* arg;

  gp = runtime_g();
  if (gp == nil)
    runtime_throw("runtime: missing g in g0 entry");
  fv = gp->entry;
  arg = (G*) gp->param;
  gp->entry = nil;
  gp->param = nil;
  if (fv == nil)
    runtime_throw("runtime: missing g0 callback");

  pfn = (void (*)(G*)) fv->fn;
  __builtin_call_with_static_chain(pfn(arg), fv);
  runtime_throw("runtime: mcall function returned");
}

__attribute__((noreturn)) static void
runtime_libgo_g0_entry(void)
{
  M* mp;

  mp = runtime_m();
  if (mp == nil)
    runtime_throw("runtime: missing m in g0 entry");
  if (mp->exiting) {
    mexit(true);
    runtime_throw("runtime: mexit returned");
  }
  runtime_libgo_invoke_entry();
}

static void
runtime_libgo_init_g0_context(G* gp, void* anchor)
{
  uintptr_t sp;

  sp = runtime_libgo_current_stack_top(anchor);
  if (sp == 0)
    runtime_throw("runtime: failed to initialize g0 context");
  runtime_libgo_init_context(runtime_libgo_gctx(gp), sp, runtime_libgo_g0_entry);
}

__attribute__((noreturn)) static void
runtime_kolibri_exit_code(uint32_t code)
{
  uint32_t eax = UINT32_MAX;

  __asm__ __volatile__(
      "int $0x40"
      :
      : "a"(eax), "b"(code)
      : "ecx", "edx", "esi", "edi", "memory", "cc");
  __builtin_unreachable();
}

static int
runtime_kos_ensure_heap(void)
{
  if (runtime_kos_heap_initialized != 0)
    return 1;
  if (runtime_kos_heap_init_raw() == 0)
    return 0;
  runtime_kos_heap_initialized = 1;
  return 1;
}

G*
runtime_g(void)
{
  int32_t slot;

  slot = runtime_kos_get_current_thread_slot_raw();
  if (slot > 0 && (uint32_t) slot < RUNTIME_LIBGO_MAX_THREAD_SLOTS) {
    G* gp = runtime_bridge_g_slots[slot];
    if (gp != nil)
      return gp;
  }
  return runtime_bridge_g_fallback;
}

M*
runtime_m(void)
{
  G* gp = runtime_g();
  return gp == nil ? nil : gp->m;
}

void
runtime_setg(G* gp)
{
  int32_t slot;

  slot = runtime_kos_get_current_thread_slot_raw();
  if (slot > 0 && (uint32_t) slot < RUNTIME_LIBGO_MAX_THREAD_SLOTS) {
    runtime_bridge_g_slots[slot] = gp;
  } else {
    runtime_bridge_g_fallback = gp;
  }
}

void runtime_newosproc(M*) __asm__(GOSYM_PREFIX "runtime.newosproc");

void
runtime_newosproc(M* mp)
{
  void* stack;
  uint32_t sp;
  int32_t raw_id;

  if (mp == nil)
    runtime_throw("runtime: newosproc nil m");

  stack = runtime_alloc_zeroed(RUNTIME_LIBGO_THREAD_STACK);
  if (stack == NULL)
    runtime_throw("runtime: cannot allocate thread stack");

  sp = runtime_libgo_thread_stack_pointer(stack, RUNTIME_LIBGO_THREAD_STACK, mp);
  if (sp == 0)
    runtime_throw("runtime: invalid thread stack");

  raw_id = runtime_kos_create_thread_raw((uint32_t) (uintptr_t) &runtime_libgo_thread_entry, sp);
  if (raw_id < 0)
    runtime_throw("runtime: CreateThreadRaw failed");
}

void
runtime_gogo(G* newg)
{
  runtime_libgo_context* ctx;

  if (newg == nil)
    runtime_throw("runtime: gogo nil g");
  ctx = runtime_libgo_gctx(newg);
  if (ctx == NULL)
    runtime_throw("runtime: gogo nil context");

  newg->fromgogo = true;
  runtime_setg(newg);
  runtime_loadcontext(ctx);
  runtime_throw("runtime: gogo returned");
}

void
runtime_mcall(FuncVal* fv)
{
  G* gp;
  M* mp;
  void* afterregs;

  gp = runtime_g();
  if (gp == nil || gp->m == nil)
    runtime_throw("runtime: mcall without g");
  mp = gp->m;
  if (gp == mp->g0)
    runtime_throw("runtime: mcall called on m->g0 stack");

  gp->gcnextsp = (uintptr_t) (&afterregs);
  gp->gcnextsp2 = 0;
  gp->fromgogo = false;

  mp->g0->entry = fv;
  mp->g0->param = gp;
  runtime_setg(mp->g0);
  runtime_swapcontext(runtime_libgo_gctx(gp), runtime_libgo_gctx(mp->g0));
}

void resetNewG(G*, void**, uintptr*) __asm__(GOSYM_PREFIX "runtime.resetNewG");

void
resetNewG(G* newg, void** sp, uintptr* spsize)
{
  if (newg == nil || sp == NULL || spsize == NULL)
    runtime_throw("runtime: bad resetNewG");
  *sp = newg->gcinitialsp;
  *spsize = newg->gcstacksize;
  if (*spsize == 0)
    runtime_throw("runtime: bad spsize in resetNewG");
  newg->gcnextsp = (uintptr_t) (*sp);
  newg->gcnextsp2 = 0;
}

void makeGContext(G*, byte*, uintptr) __asm__(GOSYM_PREFIX "runtime.makeGContext");

void
makeGContext(G* gp, byte* sp, uintptr spsize)
{
  uintptr_t top;

  if (gp == nil)
    runtime_throw("runtime: makeGContext nil g");
  top = runtime_libgo_make_stack_top(sp, spsize);
  if (top == 0)
    runtime_throw("runtime: makeGContext bad stack");
  runtime_libgo_init_context(runtime_libgo_gctx(gp), top, kickoff);
}

void stackfree(G*) __asm__(GOSYM_PREFIX "runtime.stackfree");

void
stackfree(G* gp)
{
  if (gp == nil)
    return;
  if (gp->gcstacksize != 0 && gp->gcinitialsp != NULL) {
    free(gp->gcinitialsp);
    gp->gcinitialsp = NULL;
    gp->gcstacksize = 0;
  }
}

G* runtime_malg(bool, bool, byte**, uintptr*) __asm__(GOSYM_PREFIX "runtime.malg");

G*
runtime_malg(bool allocatestack, bool signalstack, byte** ret_stack, uintptr* ret_stacksize)
{
  G* newg;
  uintptr_t stacksize;
  byte* stack;

  static byte* unused_stack;
  static uintptr_t unused_stacksize;

  if (ret_stack == NULL)
    ret_stack = &unused_stack;
  if (ret_stacksize == NULL)
    ret_stacksize = &unused_stacksize;
  *ret_stack = NULL;
  *ret_stacksize = 0;

  newg = allocg();
  if (newg == nil)
    runtime_throw("runtime: allocg failed");

  if (!allocatestack)
    return newg;

  stacksize = signalstack ? RUNTIME_LIBGO_SIGNAL_STACK : RUNTIME_LIBGO_STACK_MIN;
  stack = (byte*) runtime_alloc_zeroed((size_t) stacksize);
  if (stack == NULL)
    runtime_throw("runtime: cannot allocate goroutine stack");

  *ret_stack = stack;
  *ret_stacksize = stacksize;
  newg->gcinitialsp = stack;
  newg->gcstacksize = stacksize;
  newg->gcnextsp = (uintptr_t) stack;
  newg->gcinitialsp2 = NULL;
  newg->gcnextsp2 = 0;
  return newg;
}

void setGContext(void) __asm__(GOSYM_PREFIX "runtime.setGContext");

void
setGContext(void)
{
  G* gp;
  int anchor;

  gp = runtime_g();
  if (gp == nil)
    return;
  gp->gcinitialsp = &anchor;
  gp->gcstack = 0;
  gp->gcstacksize = 0;
  gp->gcnextsp = (uintptr_t) (&anchor);
  gp->gcinitialsp2 = NULL;
  gp->gcnextsp2 = 0;
  runtime_libgo_init_g0_context(gp, &anchor);
}

void*
runtime_mstart(void* arg)
{
  M* mp;
  G* gp;
  int anchor;

  mp = (M*) arg;
  if (mp == nil || mp->g0 == nil)
    return NULL;

  gp = mp->g0;
  gp->m = mp;
  mp->curg = gp;
  runtime_setg(gp);

  gp->entry = nil;
  gp->param = nil;

  minit();

  gp->gcinitialsp = &anchor;
  gp->gcstack = 0;
  gp->gcstacksize = 0;
  gp->gcnextsp = (uintptr_t) (&anchor);
  gp->gcinitialsp2 = NULL;
  gp->gcnextsp2 = 0;
  runtime_libgo_init_g0_context(gp, &anchor);

  if (mp->exiting) {
    mexit(true);
    return NULL;
  }

  mstart1();
  runtime_throw("runtime: mstart1 returned");
}

void
runtime_m_start(void* arg)
{
  runtime_mstart(arg);
}

void
runtime_throw(const char* message)
{
  gothrow(runtime_gostringnocopy((const byte*) message));
}

void
runtime_panicstring(const char* message)
{
  runtime_throw(message);
}

void runtime_abort(void) __asm__(GOSYM_PREFIX "runtime.abort");

void
runtime_abort(void)
{
  runtime_kolibri_exit_code(1);
}

void runtime_exit(int32 code) __asm__(GOSYM_PREFIX "runtime.exit");

void
runtime_exit(int32 code)
{
  runtime_kolibri_exit_code((uint32_t) code);
}

struct caller_ret Caller(intgo skip) __asm__(GOSYM_PREFIX "runtime.Caller");

struct caller_ret
Caller(intgo skip)
{
  struct caller_ret ret;
  (void) skip;
  memset(&ret, 0, sizeof(ret));
  return ret;
}

intgo Callers(intgo skip, struct __go_open_array pc) __asm__(GOSYM_PREFIX "runtime.Callers");

intgo
Callers(intgo skip, struct __go_open_array pc)
{
  (void) skip;
  (void) pc;
  return 0;
}

int32
runtime_callers(int32 skip, Location* locbuf, int32 max, bool keep_callers)
{
  (void) skip;
  (void) locbuf;
  (void) max;
  (void) keep_callers;
  return 0;
}

int32
runtime_callersRaw(uintptr* pcbuf, int32 max)
{
  (void) pcbuf;
  (void) max;
  return 0;
}

uintptr runtime_funcentry(uintptr pc) __asm__(GOSYM_PREFIX "runtime.funcentry");

uintptr
runtime_funcentry(uintptr pc)
{
  (void) pc;
  return 0;
}

struct funcfileline_return
runtime_funcfileline(uintptr targetpc, int32 index, bool more)
{
  struct funcfileline_return ret;
  (void) targetpc;
  (void) index;
  (void) more;
  memset(&ret, 0, sizeof(ret));
  return ret;
}

int32 runtime_pcInlineCallers(uintptr pc, Location* locbuf, int32 max) __asm__(GOSYM_PREFIX "runtime.pcInlineCallers");

int32
runtime_pcInlineCallers(uintptr pc, Location* locbuf, int32 max)
{
  (void) pc;
  (void) locbuf;
  (void) max;
  return 0;
}

void runtime_Fieldtrack(void* addr) __asm__(GOSYM_PREFIX "runtime.Fieldtrack");

void
runtime_Fieldtrack(void* addr)
{
  (void) addr;
}

void _cgo_notify_runtime_init_done(void) __asm__(GOSYM_PREFIX "runtime.__cgo__notify__runtime__init__done");

void
_cgo_notify_runtime_init_done(void)
{
}

void
runtime_entersyscall(void)
{
}

void
runtime_entersyscallblock(void)
{
}

bool doscanstack(G* gp, void* scanword) __asm__("runtime.doscanstack");

bool
doscanstack(G* gp, void* scanword)
{
  (void) gp;
  (void) scanword;
  return false;
}

void getTraceback(G* me, G* gp) __asm__(GOSYM_PREFIX "runtime.getTraceback");

void
getTraceback(G* me, G* gp)
{
  (void) me;
  (void) gp;
}

void gtraceback(G* gp) __asm__(GOSYM_PREFIX "runtime.gtraceback");

void
gtraceback(G* gp)
{
  (void) gp;
}

void doscanstackswitch(G* me, G* gp) __asm__(GOSYM_PREFIX "runtime.doscanstackswitch");

void
doscanstackswitch(G* me, G* gp)
{
  (void) me;
  (void) gp;
}

void reflect_call(const struct functype* type, FuncVal* fn, _Bool is_interface, _Bool is_closure, void** stack, void** ret)
  __asm__(GOSYM_PREFIX "runtime.reflectcall");

void
reflect_call(const struct functype* type, FuncVal* fn, _Bool is_interface, _Bool is_closure, void** stack, void** ret)
{
  (void) type;
  (void) fn;
  (void) is_interface;
  (void) is_closure;
  (void) stack;
  (void) ret;
  runtime_throw("runtime.reflectcall is not implemented on KolibriOS libgo bridge");
}

void go_0kos_thread_bootstrap(void* arg) __asm__("go_0kos.ThreadBootstrap");

void
go_0kos_thread_bootstrap(void* arg)
{
  kos_thread_bootstrap(arg);
}

void
runtime_kolibri_start(void (*init)(void), void (*main_fn)(void))
{
  (void) init;
  (void) main_fn;

  runtime_isarchive = false;
  if (runtime_isstarted)
    return;
  runtime_isstarted = true;
  __go_end = (uintptr) &__end;

  runtime_ginit();
  runtime_cpuinit();
  runtime_check();
  runtime_args(0, NULL);
  runtime_osinit();
  runtime_schedinit();
  __go_go((uintptr) runtime_main, NULL);
  runtime_mstart(runtime_m());
  runtime_kolibri_exit_code(0);
}

void runtime_gc_set_stack_top(const void* ptr)
{
  (void) ptr;
}

__attribute__((noreturn)) void runtime_kolibri_exit_process(void) __asm__("runtime_kolibri_exit_process");

__attribute__((noreturn)) void
runtime_kolibri_exit_process(void)
{
  runtime_kolibri_exit_code(0);
}

__attribute__((noreturn)) void runtime_kolibri_exit_thread(void) __asm__("runtime_kolibri_exit_thread");

__attribute__((noreturn)) void
runtime_kolibri_exit_thread(void)
{
  runtime_kolibri_exit_code(0);
}

void
runtime_kolibri_poll_world_stop(void)
{
}

uint32_t
runtime_kolibri_get_m_count(void)
{
  return 1;
}

uint32_t
runtime_kolibri_start_locked(uintptr record_ptr, uint32_t stack_size)
{
  (void) record_ptr;
  (void) stack_size;
  return 0;
}

uint32_t
runtime_kolibri_set_threads(uint32_t count)
{
  if (count == 0)
    count = 1;
  runtime_libgo_threads = count;
  return runtime_libgo_threads;
}

uint32_t
runtime_kolibri_get_threads(void)
{
  return runtime_libgo_threads;
}

uint32_t
runtime_bootstrap_has_gc(void)
{
  return 0;
}

void
runtime_gc_poll(void)
{
}

static const char*
runtime_prepare_window_title_impl(uint32_t prefix, int use_prefix, const char* src, intptr_t len)
{
  char* resized;
  size_t needed;
  size_t offset;

  if (src == NULL)
    return NULL;
  if (len < 0)
    len = 0;

  offset = use_prefix ? 1u : 0u;
  needed = offset + (size_t) len + 1u;
  if (runtime_window_title_buffer == NULL || needed > runtime_window_title_capacity) {
    resized = (char*) realloc(runtime_window_title_buffer, needed);
    if (resized == NULL)
      return runtime_window_title_buffer;
    runtime_window_title_buffer = resized;
    runtime_window_title_capacity = needed;
  }

  if (use_prefix)
    runtime_window_title_buffer[0] = (char) prefix;
  if (len > 0)
    memcpy(runtime_window_title_buffer + offset, src, (size_t) len);
  runtime_window_title_buffer[offset + (size_t) len] = '\0';
  return runtime_window_title_buffer;
}

const char*
runtime_prepare_window_title(const char* src, intptr_t len)
{
  return runtime_prepare_window_title_impl(0, 0, src, len);
}

const char*
runtime_prepare_window_title_with_prefix(uint32_t prefix, const char* src, intptr_t len)
{
  return runtime_prepare_window_title_impl(prefix, 1, src, len);
}

char*
runtime_alloc_cstring(const char* src, intptr_t len)
{
  char* out;

  if (src == NULL)
    return NULL;
  if (len < 0)
    len = 0;

  out = (char*) runtime_alloc_zeroed((size_t) len + 1u);
  if (out == NULL)
    return NULL;
  if (len > 0)
    memcpy(out, src, (size_t) len);
  out[len] = '\0';
  return out;
}

void
runtime_free_cstring(void* ptr)
{
  free(ptr);
}

uint32_t
runtime_pointer_value(void* ptr)
{
  return (uint32_t) (uintptr_t) ptr;
}

String
runtime_cstring_to_gostring(uint32_t ptr_addr)
{
  String result;
  const char* src;
  intptr_t len;
  char* out;

  result.str = NULL;
  result.len = 0;
  src = (const char*) (uintptr_t) ptr_addr;
  if (src == NULL)
    return result;

  len = (intptr_t) kos_strlen(src);
  out = (char*) runtime_alloc_zeroed((size_t) len + 1u);
  if (out == NULL)
    return result;
  if (len > 0)
    memcpy(out, src, (size_t) len);
  out[len] = '\0';
  result.str = (const byte*) out;
  result.len = len;
  return result;
}

Slice
runtime_copy_bytes(uint32_t ptr_addr, uint32_t size)
{
  Slice result;
  byte* out;

  result.__values = NULL;
  result.__count = 0;
  result.__capacity = 0;
  if (ptr_addr == 0 || size == 0)
    return result;

  out = (byte*) runtime_alloc_zeroed((size_t) size);
  if (out == NULL)
    return result;
  memcpy(out, (const void*) (uintptr_t) ptr_addr, (size_t) size);
  result.__values = out;
  result.__count = (intgo) size;
  result.__capacity = (intgo) size;
  return result;
}

uint32_t
runtime_read_u32(uint32_t base, uint32_t offset)
{
  if (base == 0)
    return 0;
  return *(const uint32_t*) (uintptr_t) (base + offset);
}

static uint32_t KOS_STDCALL
runtime_kos_dll_mem_alloc(uint32_t size)
{
  uint32_t result;

  if (!runtime_kos_ensure_heap())
    return 0;
  result = runtime_kos_heap_alloc_raw(size);
  if (result != 0) {
    runtime_kos_heap_alloc_count++;
    runtime_kos_heap_alloc_bytes += size;
  }
  return result;
}

static uint32_t KOS_STDCALL
runtime_kos_dll_mem_free(uint32_t ptr)
{
  uint32_t result;

  if (ptr == 0)
    return 1;
  if (!runtime_kos_ensure_heap())
    return 0;
  result = runtime_kos_heap_free_raw(ptr);
  if (result != 0)
    runtime_kos_heap_free_count++;
  return result;
}

static uint32_t KOS_STDCALL
runtime_kos_dll_mem_realloc(uint32_t ptr, uint32_t size)
{
  uint32_t result;

  if (!runtime_kos_ensure_heap())
    return 0;
  result = runtime_kos_heap_realloc_raw(size, ptr);
  if (result != 0) {
    runtime_kos_heap_realloc_count++;
    runtime_kos_heap_realloc_bytes += size;
  }
  return result;
}

uint32_t
runtime_kos_lookup_dll_export(uint32_t table_addr, const char* name)
{
  const kos_dll_export* cursor;

  if (table_addr == 0 || name == NULL)
    return 0;
  cursor = (const kos_dll_export*) (uintptr_t) table_addr;
  while (cursor->name != NULL) {
    if (kos_strcmp(cursor->name, name) == 0)
      return (uint32_t) (uintptr_t) cursor->data;
    cursor++;
  }
  return 0;
}

static uint32_t
runtime_kos_load_named_dll(const char* name)
{
  static const char prefix[] = "/sys/lib/";
  char path[256];
  size_t prefix_len;
  size_t name_len;

  if (name == NULL || name[0] == '\0')
    return 0;
  if (name[0] == '/')
    return runtime_kos_load_dll_cstring_raw(name);

  prefix_len = sizeof(prefix) - 1u;
  name_len = kos_strlen(name);
  if (prefix_len + name_len + 1u > sizeof(path))
    return 0;

  memcpy(path, prefix, prefix_len);
  memcpy(path + prefix_len, name, name_len + 1u);
  return runtime_kos_load_dll_cstring_raw(path);
}

static int
runtime_kos_link_dll_imports(uint32_t table_addr, uint32_t imports_addr)
{
  uint32_t* cursor;

  if (table_addr == 0 || imports_addr == 0)
    return 0;
  cursor = (uint32_t*) (uintptr_t) imports_addr;
  while (*cursor != 0) {
    uint32_t proc = runtime_kos_lookup_dll_export(table_addr, (const char*) (uintptr_t) *cursor);
    if (proc == 0)
      return 0;
    *cursor = proc;
    cursor++;
  }
  return 1;
}

static uint32_t
runtime_kos_dll_lib_init_proc(uint32_t table_addr)
{
  const kos_dll_export* exports;

  if (table_addr == 0)
    return 0;
  exports = (const kos_dll_export*) (uintptr_t) table_addr;
  if (exports->name == NULL)
    return 0;
  if (exports->name[0] == 'l' &&
      exports->name[1] == 'i' &&
      exports->name[2] == 'b' &&
      exports->name[3] == '_')
    return (uint32_t) (uintptr_t) exports->data;
  return 0;
}

uint32_t
runtime_kos_call_stdcall0(uint32_t proc)
{
  return proc == 0 ? 0 : ((kos_stdcall0_fn) (uintptr_t) proc)();
}

uint32_t
runtime_kos_call_stdcall1(uint32_t proc, uint32_t arg0)
{
  return proc == 0 ? 0 : ((kos_stdcall1_fn) (uintptr_t) proc)(arg0);
}

uint32_t
runtime_kos_call_stdcall2(uint32_t proc, uint32_t arg0, uint32_t arg1)
{
  return proc == 0 ? 0 : ((kos_stdcall2_fn) (uintptr_t) proc)(arg0, arg1);
}

uint32_t
runtime_kos_call_stdcall3(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2)
{
  return proc == 0 ? 0 : ((kos_stdcall3_fn) (uintptr_t) proc)(arg0, arg1, arg2);
}

uint32_t
runtime_kos_call_stdcall4(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3)
{
  return proc == 0 ? 0 : ((kos_stdcall4_fn) (uintptr_t) proc)(arg0, arg1, arg2, arg3);
}

uint32_t
runtime_kos_call_stdcall5(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4)
{
  return proc == 0 ? 0 : ((kos_stdcall5_fn) (uintptr_t) proc)(arg0, arg1, arg2, arg3, arg4);
}

uint32_t
runtime_kos_call_stdcall6(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4, uint32_t arg5)
{
  return proc == 0 ? 0 : ((kos_stdcall6_fn) (uintptr_t) proc)(arg0, arg1, arg2, arg3, arg4, arg5);
}

uint32_t
runtime_kos_call_stdcall7(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4, uint32_t arg5, uint32_t arg6)
{
  return proc == 0 ? 0 : ((kos_stdcall7_fn) (uintptr_t) proc)(arg0, arg1, arg2, arg3, arg4, arg5, arg6);
}

uint32_t
runtime_kos_call_cdecl2(uint32_t proc, uint32_t arg0, uint32_t arg1)
{
  return proc == 0 ? 0 : ((kos_cdecl2_fn) (uintptr_t) proc)(arg0, arg1);
}

uint32_t
runtime_kos_call_cdecl5(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4)
{
  return proc == 0 ? 0 : ((kos_cdecl5_fn) (uintptr_t) proc)(arg0, arg1, arg2, arg3, arg4);
}

void
runtime_kos_call_stdcall1_void(uint32_t proc, uint32_t arg0)
{
  if (proc != 0)
    ((kos_stdcall1_void_fn) (uintptr_t) proc)(arg0);
}

void
runtime_kos_call_stdcall2_void(uint32_t proc, uint32_t arg0, uint32_t arg1)
{
  if (proc != 0)
    ((kos_stdcall2_void_fn) (uintptr_t) proc)(arg0, arg1);
}

void
runtime_kos_call_stdcall5_void(uint32_t proc, uint32_t arg0, uint32_t arg1, uint32_t arg2, uint32_t arg3, uint32_t arg4)
{
  if (proc != 0)
    ((kos_stdcall5_void_fn) (uintptr_t) proc)(arg0, arg1, arg2, arg3, arg4);
}

static void
runtime_kos_dialog_noop(void)
{
}

uint32_t
runtime_kos_dialog_noop_addr(void)
{
  return (uint32_t) (uintptr_t) &runtime_kos_dialog_noop;
}

uint32_t KOS_STDCALL runtime_kos_dll_load_imports(uint32_t import_table_addr);

uint32_t
runtime_kos_init_dll_library(uint32_t proc)
{
  if (proc == 0)
    return 1;
#if defined(__i386__)
  {
    uint32_t alloc_proc = (uint32_t) (uintptr_t) runtime_kos_dll_mem_alloc;
    uint32_t free_proc = (uint32_t) (uintptr_t) runtime_kos_dll_mem_free;
    uint32_t realloc_proc = (uint32_t) (uintptr_t) runtime_kos_dll_mem_realloc;
    uint32_t load_proc = (uint32_t) (uintptr_t) runtime_kos_dll_load_imports;

    __asm__ volatile(
        "call *%[init_proc]\n\t"
        : "+a"(alloc_proc), "+b"(free_proc), "+c"(realloc_proc), "+d"(load_proc)
        : [init_proc] "m"(proc)
        : "memory", "cc", "esi", "edi");
    return alloc_proc;
  }
#else
  return runtime_kos_call_stdcall4(
      proc,
      (uint32_t) (uintptr_t) runtime_kos_dll_mem_alloc,
      (uint32_t) (uintptr_t) runtime_kos_dll_mem_free,
      (uint32_t) (uintptr_t) runtime_kos_dll_mem_realloc,
      (uint32_t) (uintptr_t) runtime_kos_dll_load_imports);
#endif
}

uint32_t KOS_STDCALL
runtime_kos_dll_load_imports(uint32_t import_table_addr)
{
  const kos_dll_import_library* cursor;
  uint32_t dll_table;
  uint32_t dll_load_proc;

  static const char dll_loader_path[] = "/sys/lib/dll.obj";

  dll_table = runtime_kos_load_dll_cstring_raw(dll_loader_path);
  dll_load_proc = 0;
  if (dll_table != 0)
    dll_load_proc = runtime_kos_lookup_dll_export(dll_table, "dll_load");
  if (dll_load_proc != 0)
    return runtime_kos_call_stdcall1(dll_load_proc, import_table_addr);

  cursor = (const kos_dll_import_library*) (uintptr_t) import_table_addr;
  if (cursor == NULL)
    return 1;

  while (cursor->imports != 0) {
    uint32_t table_addr;
    uint32_t init_proc;

    table_addr = runtime_kos_load_named_dll(cursor->library_name);
    if (table_addr == 0)
      return 1;
    if (!runtime_kos_link_dll_imports(table_addr, cursor->imports))
      return 1;

    init_proc = runtime_kos_dll_lib_init_proc(table_addr);
    if (init_proc != 0 && runtime_kos_init_dll_library(init_proc) != 0)
      return 1;
    cursor++;
  }

  return 0;
}

int
runtime_console_bridge_ready(void)
{
  return runtime_console_bridge_write_proc != 0;
}

void
runtime_console_bridge_set(uint32_t table, uint32_t write_proc, uint32_t exit_proc, uint32_t gets_proc)
{
  runtime_console_bridge_table = table;
  runtime_console_bridge_write_proc = write_proc;
  runtime_console_bridge_exit_proc = exit_proc;
  runtime_console_bridge_gets_proc = gets_proc;
}

void
runtime_console_bridge_clear(uint32_t table)
{
  if (runtime_console_bridge_table == table) {
    runtime_console_bridge_table = 0;
    runtime_console_bridge_write_proc = 0;
    runtime_console_bridge_exit_proc = 0;
    runtime_console_bridge_gets_proc = 0;
  }
}

int
runtime_console_bridge_write(uint32_t data, uint32_t size)
{
  if (runtime_console_bridge_write_proc == 0 || data == 0 || size == 0)
    return 0;
  ((kos_stdcall2_void_fn) (uintptr_t) runtime_console_bridge_write_proc)(data, size);
  return 1;
}

int
runtime_console_bridge_read_line(uint32_t data, uint32_t size)
{
  if (runtime_console_bridge_gets_proc == 0 || data == 0 || size < 2)
    return 0;
  return ((kos_stdcall2_fn) (uintptr_t) runtime_console_bridge_gets_proc)(data, size) != 0;
}

void
runtime_console_bridge_close(uint32_t close_window)
{
  if (runtime_console_bridge_exit_proc == 0)
    return;
  ((kos_stdcall1_void_fn) (uintptr_t) runtime_console_bridge_exit_proc)(close_window);
  runtime_console_bridge_table = 0;
  runtime_console_bridge_write_proc = 0;
  runtime_console_bridge_exit_proc = 0;
  runtime_console_bridge_gets_proc = 0;
}

uint32_t
runtime_kos_heap_alloc_count_read(void)
{
  return (uint32_t) runtime_kos_heap_alloc_count;
}

uint32_t
runtime_kos_heap_alloc_bytes_read(void)
{
  return (uint32_t) runtime_kos_heap_alloc_bytes;
}

uint32_t
runtime_kos_heap_free_count_read(void)
{
  return (uint32_t) runtime_kos_heap_free_count;
}

uint32_t
runtime_kos_heap_realloc_count_read(void)
{
  return (uint32_t) runtime_kos_heap_realloc_count;
}

uint32_t
runtime_kos_heap_realloc_bytes_read(void)
{
  return (uint32_t) runtime_kos_heap_realloc_bytes;
}

uint32_t
runtime_gc_alloc_count_read(void)
{
  return 0;
}

uint32_t
runtime_gc_alloc_bytes_read(void)
{
  return 0;
}

uint32_t
runtime_gc_live_bytes_read(void)
{
  return 0;
}

uint32_t
runtime_gc_threshold_read(void)
{
  return 0;
}

uint32_t
runtime_gc_collection_count_read(void)
{
  return 0;
}

uint32_t
runtime_gc_poll_retry_read(void)
{
  return 0;
}
