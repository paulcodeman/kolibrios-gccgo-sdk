/* Copyright 2009 The Go Authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.  */

#include <stdint.h>

#include "runtime.h"

int64
runtime_nanotime1(void)
{
  uint32_t eax = 26;
  uint32_t ebx = 10;
  uint32_t edx;

  __asm__ volatile("int $0x40"
                   : "+a"(eax), "+b"(ebx), "=d"(edx)
                   :
                   : "ecx", "esi", "edi", "memory", "cc");
  return ((int64) edx << 32) | (int64) eax;
}
