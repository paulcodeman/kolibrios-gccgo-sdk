/* go-unsafe-pointer.c -- unsafe.Pointer type descriptor for Go.

   Copyright 2009 The Go Authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.  */

#include <stddef.h>

/* KolibriOS bootstrap uses the existing gccgo descriptor/function layouts. */
typedef struct {
  go_type_descriptor typ;
  const go_type_descriptor *elem;
} runtime_unsafe_ptr_type;

enum {
  tflagRegularMemory = 1 << 3,
  kindUnsafePointer = 26,
  kindPtr = 22,
  kindDirectIface = GO_TYPE_KIND_DIRECT_IFACE
};

static bool runtime_memequal32_impl(const void *, const void *);
static go_equal_function runtime_unsafe_pointer_equal = runtime_memequal32_impl;
__asm__(".global runtime.pointerequal..f");
__asm__(".set runtime.pointerequal..f, runtime_unsafe_pointer_equal");

/* This file provides the type descriptor for the unsafe.Pointer type.
   The unsafe package is defined by the compiler itself, which means
   that there is no package to compile to define the type
   descriptor.  */

extern const go_type_descriptor unsafe_Pointer
  __asm__ ("unsafe.Pointer..d");

extern const uint8_t unsafe_Pointer_gc[]
  __asm__ ("unsafe.Pointer..g");

/* Used to determine the field alignment.  */
struct field_align
{
  char c;
  void *p;
};

/* The reflection string.  */
#define REFLECTION "unsafe.Pointer"
static const go_string reflection_string =
{
  (const char *) REFLECTION,
  sizeof REFLECTION - 1
};

const uint8_t unsafe_Pointer_gc[] = { 1 };


const go_type_descriptor unsafe_Pointer =
{
  /* size */
  sizeof (void *),
  /* ptrdata */
  sizeof (void *),
  /* hash */
  78501163U,
  /* tflag */
  tflagRegularMemory,
  /* align */
  __alignof (void *),
  /* fieldAlign */
  offsetof (struct field_align, p) - 1,
  /* kind */
  kindUnsafePointer | kindDirectIface,
  /* equal */
  &runtime_unsafe_pointer_equal,
  /* gcdata */
  unsafe_Pointer_gc,
  /* _string */
  &reflection_string,
  /* uncommontype */
  NULL,
  /* ptrToThis */
  NULL
};

/* We also need the type descriptor for the pointer to unsafe.Pointer,
   since any package which refers to that type descriptor will expect
   it to be defined elsewhere.  */

extern const runtime_unsafe_ptr_type pointer_unsafe_Pointer
  __asm__ ("unsafe.Pointer..p");

/* The reflection string.  */
#define PREFLECTION "*unsafe.Pointer"
static const go_string preflection_string =
{
  (const char *) PREFLECTION,
  sizeof PREFLECTION - 1,
};

extern const uint8_t pointer_unsafe_Pointer_gc[]
  __asm__ ("unsafe.Pointer..p..g");

const uint8_t pointer_unsafe_Pointer_gc[] = { 1 };

const runtime_unsafe_ptr_type pointer_unsafe_Pointer =
{
  /* type */
  {
    /* size */
    sizeof (void *),
    /* ptrdata */
    sizeof (void *),
    /* hash */
    1256018616U,
    /* tflag */
    tflagRegularMemory,
    /* align */
    __alignof (void *),
    /* fieldAlign */
    offsetof (struct field_align, p) - 1,
    /* kind */
    kindPtr | kindDirectIface,
    /* equalfn */
    &runtime_unsafe_pointer_equal,
    /* gcdata */
    pointer_unsafe_Pointer_gc,
    /* _string */
    &preflection_string,
    /* uncommontype */
    NULL,
    /* ptrToThis */
    NULL
  },
  /* elem */
  &unsafe_Pointer
};
