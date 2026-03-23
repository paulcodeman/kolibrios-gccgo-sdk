/* Copyright 2011 The Go Authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.  */

#include <stddef.h>
#include <stdint.h>

#include "runtime.h"

struct walltime_ret
{
  int64_t sec;
  int32_t nsec;
};

static inline uint32_t
kolibrios_get_time(void)
{
  uint32_t eax = 3;

  __asm__ volatile("int $0x40"
                   : "+a"(eax)
                   :
                   : "ebx", "ecx", "edx", "esi", "edi", "memory", "cc");
  return eax;
}

static inline uint32_t
kolibrios_get_date(void)
{
  uint32_t eax = 29;

  __asm__ volatile("int $0x40"
                   : "+a"(eax)
                   :
                   : "ebx", "ecx", "edx", "esi", "edi", "memory", "cc");
  return eax;
}

static int
bcd_byte(uint32_t value)
{
  int v = (int) (value & 0xffu);
  return ((v >> 4) * 10) + (v & 0x0f);
}

static int
is_leap(int year)
{
  return (year % 4 == 0) && ((year % 100 != 0) || (year % 400 == 0));
}

static int
days_in_month(int year, int month)
{
  static const int month_days[12] =
    { 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31 };

  int days = month_days[month - 1];
  if (month == 2 && is_leap(year))
    days++;
  return days;
}

static int64_t
unix_seconds_from_ymdhms(int year, int month, int day, int hour, int minute, int second)
{
  int y;
  int m;
  int64_t days = 0;

  for (y = 1970; y < year; ++y)
    days += is_leap(y) ? 366 : 365;

  for (m = 1; m < month; ++m)
    days += days_in_month(year, m);

  days += (int64_t) (day - 1);
  return days * 86400 + (int64_t) hour * 3600 + (int64_t) minute * 60 + (int64_t) second;
}

struct walltime_ret now(void) __asm__ (GOSYM_PREFIX "runtime.walltime")
  __attribute__ ((no_split_stack));

struct walltime_ret
now(void)
{
  uint32_t date = kolibrios_get_date();
  uint32_t time = kolibrios_get_time();
  struct walltime_ret ret;
  int year;
  int month;
  int day;
  int hour;
  int minute;
  int second;

  /* KolibriOS exposes only the low two digits of the year.  Use the
     current bootstrap convention of treating it as 2000..2099 until the
     native port carries full RTC/calendar handling.  */
  year = 2000 + bcd_byte(date);
  month = bcd_byte(date >> 8);
  day = bcd_byte(date >> 16);
  hour = bcd_byte(time);
  minute = bcd_byte(time >> 8);
  second = bcd_byte(time >> 16);

  ret.sec = unix_seconds_from_ymdhms(year, month, day, hour, minute, second);
  ret.nsec = 0;
  return ret;
}
