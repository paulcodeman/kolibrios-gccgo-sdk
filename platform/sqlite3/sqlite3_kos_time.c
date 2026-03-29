#include <stddef.h>
#include <stdint.h>
#include <time.h>

#include "upstream/sqlite3.h"

extern uint32_t sqlite3_kos_get_date(void) __asm__("go_0kos.GetDate");
extern uint32_t sqlite3_kos_get_time(void) __asm__("go_0kos.GetTime");

static int
sqlite3_kos_bcd_to_int(unsigned char value)
{
  return ((value >> 4) * 10) + (value & 0x0f);
}

static sqlite3_int64
sqlite3_kos_days_from_civil(int year, int month, int day)
{
  int era;
  unsigned yoe;
  unsigned doy;
  unsigned doe;

  year -= month <= 2;
  era = (year >= 0 ? year : year - 399) / 400;
  yoe = (unsigned) (year - era * 400);
  doy = (unsigned) ((153 * (month + (month > 2 ? -3 : 9)) + 2) / 5 + day - 1);
  doe = yoe * 365u + yoe / 4u - yoe / 100u + doy;
  return (sqlite3_int64) era * 146097 + (sqlite3_int64) doe - 719468;
}

static void
sqlite3_kos_civil_from_days(sqlite3_int64 days, int* year, int* month, int* day)
{
  sqlite3_int64 era;
  unsigned doe;
  unsigned yoe;
  unsigned doy;
  unsigned mp;
  int y;
  int m;

  days += 719468;
  era = (days >= 0 ? days : days - 146096) / 146097;
  doe = (unsigned) (days - era * 146097);
  yoe = (doe - doe / 1460u + doe / 36524u - doe / 146096u) / 365u;
  y = (int) yoe + (int) era * 400;
  doy = doe - (365u * yoe + yoe / 4u - yoe / 100u);
  mp = (5u * doy + 2u) / 153u;
  *day = (int) (doy - (153u * mp + 2u) / 5u + 1u);
  m = (int) (mp + (mp < 10u ? 3u : (unsigned) -9));
  y += m <= 2;
  *year = y;
  *month = m;
}

static int
sqlite3_kos_day_of_year(int year, int month, int day)
{
  return (int) (sqlite3_kos_days_from_civil(year, month, day) -
                sqlite3_kos_days_from_civil(year, 1, 1));
}

static void
sqlite3_kos_fill_tm(sqlite3_int64 seconds, struct tm* out)
{
  sqlite3_int64 days;
  sqlite3_int64 rem;
  int year;
  int month;
  int day;
  int wday;

  days = seconds / 86400;
  rem = seconds % 86400;
  if (rem < 0) {
    rem += 86400;
    days--;
  }

  sqlite3_kos_civil_from_days(days, &year, &month, &day);
  out->tm_hour = (int) (rem / 3600);
  rem %= 3600;
  out->tm_min = (int) (rem / 60);
  out->tm_sec = (int) (rem % 60);
  out->tm_mday = day;
  out->tm_mon = month - 1;
  out->tm_year = year - 1900;
  out->tm_yday = sqlite3_kos_day_of_year(year, month, day);
  wday = (int) ((days + 4) % 7);
  if (wday < 0) {
    wday += 7;
  }
  out->tm_wday = wday;
  out->tm_isdst = 0;
}

sqlite3_int64
sqlite3_kos_current_unix_millis(void)
{
  uint32_t raw_date = sqlite3_kos_get_date();
  uint32_t raw_time = sqlite3_kos_get_time();
  int year;
  int month;
  int day;
  int hour;
  int minute;
  int second;
  sqlite3_int64 days;
  sqlite3_int64 seconds;

  year = 2000 + sqlite3_kos_bcd_to_int((unsigned char) raw_date);
  month = sqlite3_kos_bcd_to_int((unsigned char) (raw_date >> 8));
  day = sqlite3_kos_bcd_to_int((unsigned char) (raw_date >> 16));

  hour = sqlite3_kos_bcd_to_int((unsigned char) raw_time);
  minute = sqlite3_kos_bcd_to_int((unsigned char) (raw_time >> 8));
  second = sqlite3_kos_bcd_to_int((unsigned char) (raw_time >> 16));

  days = sqlite3_kos_days_from_civil(year, month, day);
  seconds = days * 86400 + hour * 3600 + minute * 60 + second;
  return seconds * 1000;
}

static struct tm sqlite3_kos_local_tm;
static struct tm sqlite3_kos_gm_tm;

time_t
time(time_t* out)
{
  time_t value = (time_t) (sqlite3_kos_current_unix_millis() / 1000);

  if (out != NULL) {
    *out = value;
  }
  return value;
}

struct tm*
gmtime_r(const time_t* clock, struct tm* out)
{
  if (clock == NULL || out == NULL) {
    return NULL;
  }
  sqlite3_kos_fill_tm((sqlite3_int64) *clock, out);
  return out;
}

struct tm*
localtime_r(const time_t* clock, struct tm* out)
{
  return gmtime_r(clock, out);
}

struct tm*
gmtime(const time_t* clock)
{
  return gmtime_r(clock, &sqlite3_kos_gm_tm);
}

struct tm*
localtime(const time_t* clock)
{
  return localtime_r(clock, &sqlite3_kos_local_tm);
}
