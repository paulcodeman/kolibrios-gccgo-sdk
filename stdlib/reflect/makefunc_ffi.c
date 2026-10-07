/* Copyright 2014 The Go Authors. All rights reserved.
   BSD-style license in LICENSE. Adapted from libgo reflect/makefunc_ffi_c.c.
   The bootstrap runtime already checks active defers in runtime_canrecover;
   it does not have libgo's separate makefuncfficanrecover frame bookkeeping. */
#include <ffi.h>
#include <stdbool.h>
#include <stdint.h>
extern void ffiCallbackGo(void *, void **, void *, int32_t, bool)
    __asm__("reflect.ffiCallbackGo");
static void ffi_callback(ffi_cif *cif, void *results, void **args, void *closure) {
    (void)cif;
    ffiCallbackGo(results, args, closure, sizeof(ffi_arg),
                  __BYTE_ORDER__ == __ORDER_BIG_ENDIAN__);
}
void makeFuncFFI(void *cif, void *impl) __asm__("reflect.makeFuncFFI");
void makeFuncFFI(void *cif, void *impl) {
    ffi_prep_go_closure(impl, (ffi_cif *)cif, ffi_callback);
}
