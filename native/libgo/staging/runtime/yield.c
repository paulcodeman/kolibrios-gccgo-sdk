/* Copyright 2011 The Go Authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.  */

#include <stddef.h>
#include <stdint.h>

#if defined (__i386__) || defined (__x86_64__)
#include <xmmintrin.h>
#endif

#include "runtime.h"

static inline uint64_t
kolibrios_nanotime(void)
{
  uint32_t eax = 26;
  uint32_t ebx = 10;
  uint32_t edx;

  __asm__ volatile("int $0x40"
                   : "+a"(eax), "+b"(ebx), "=d"(edx)
                   :
                   : "ecx", "esi", "edi", "memory", "cc");
  return ((uint64_t) edx << 32) | (uint64_t) eax;
}

void
runtime_procyield(uint32 cnt)
{
  volatile uint32 i;

  for (i = 0; i < cnt; ++i)
    {
#if defined (__i386__) || defined (__x86_64__)
      _mm_pause();
#endif
    }
}

void runtime_osyield(void)
  __attribute__ ((no_split_stack));

void
runtime_osyield(void)
{
  uint32_t eax = 68;
  uint32_t ebx = 1;

  __asm__ volatile("int $0x40"
                   : "+a"(eax), "+b"(ebx)
                   :
                   : "ecx", "edx", "esi", "edi", "memory", "cc");
}

void
runtime_usleep(uint32 us)
{
  uint64_t start;
  uint64_t target;

  if (us == 0)
    {
      runtime_osyield();
      return;
    }

  start = kolibrios_nanotime();
  target = start + (uint64_t) us * 1000ULL;

  for (;;)
    {
      uint64_t now = kolibrios_nanotime();
      if (now >= target)
        return;

      if (target - now >= 10000000ULL)
        {
          uint32_t eax = 5;
          uint32_t ebx = 1;

          __asm__ volatile("int $0x40"
                           : "+a"(eax), "+b"(ebx)
                           :
                           : "ecx", "edx", "esi", "edi", "memory", "cc");
          continue;
        }

      runtime_osyield();
    }
}

uint32 runtime_kolibriosThreadSlot(void)
  __asm__ (GOSYM_PREFIX "runtime.kolibriosThreadSlot")
  __attribute__ ((no_split_stack));

uint32
runtime_kolibriosThreadSlot(void)
{
  uint32_t eax = 51;
  uint32_t ebx = 2;

  __asm__ volatile("int $0x40"
                   : "+a"(eax), "+b"(ebx)
                   :
                   : "ecx", "edx", "esi", "edi", "memory", "cc");
  return eax;
}
