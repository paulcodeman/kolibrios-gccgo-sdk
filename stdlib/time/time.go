package time

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"kos"
)

type Duration int64

const (
	Nanosecond  Duration = 1
	Microsecond          = 1000 * Nanosecond
	Millisecond          = 1000 * Microsecond
	Second               = 1000 * Millisecond
	Minute               = 60 * Second
	Hour                 = 60 * Minute
)

func durationUnit(value string) (Duration, bool) {
	switch value {
	case "ns":
		return Nanosecond, true
	case "us", "\u00b5s", "\u03bcs":
		return Microsecond, true
	case "ms":
		return Millisecond, true
	case "s":
		return Second, true
	case "m":
		return Minute, true
	case "h":
		return Hour, true
	default:
		return 0, false
	}
}

func (d Duration) Seconds() float64 {
	return float64(d) / float64(Second)
}

func (d Duration) String() string {
	sign := ""
	if d < 0 {
		sign = "-"
		d = -d
	}

	switch {
	case d == 0:
		return "0s"
	case d%Hour == 0:
		return sign + strconv.FormatInt(int64(d/Hour), 10) + "h"
	case d%Minute == 0:
		return sign + strconv.FormatInt(int64(d/Minute), 10) + "m"
	case d%Second == 0:
		return sign + strconv.FormatInt(int64(d/Second), 10) + "s"
	case d%Millisecond == 0:
		return sign + strconv.FormatInt(int64(d/Millisecond), 10) + "ms"
	case d%Microsecond == 0:
		return sign + strconv.FormatInt(int64(d/Microsecond), 10) + "us"
	default:
		return sign + strconv.FormatInt(int64(d), 10) + "ns"
	}
}

type Month int

const (
	January Month = 1 + iota
	February
	March
	April
	May
	June
	July
	August
	September
	October
	November
	December
)

type Weekday int

const (
	Sunday Weekday = iota
	Monday
	Tuesday
	Wednesday
	Thursday
	Friday
	Saturday
)

const RFC1123 = "Mon, 02 Jan 2006 15:04:05 MST"
const RFC3339 = "2006-01-02T15:04:05Z07:00"
const RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
const layoutASN1UTCTimeMinutes = "0601021504Z0700"
const layoutASN1UTCTimeSeconds = "060102150405Z0700"
const layoutASN1GeneralizedTime = "20060102150405Z0700"

type Location struct {
	name   string
	offset int
}

var UTC = &Location{name: "UTC", offset: 0}
var Local = UTC

type Time struct {
	unixSeconds  int64
	nanosecond   int32
	monotonicNS  int64
	hasMonotonic bool
	loc          *Location
}

const (
	nanosecondsPerSecond      = int64(1000000000)
	nanosecondsPerCentisecond = int64(10000000)
	secondsPerMinute          = int64(60)
	secondsPerHour            = int64(60 * 60)
	secondsPerDay             = int64(24 * 60 * 60)
	daysPer400Years           = int64(146097)
	unixToCivilEpochDays      = int64(719468)
	maxInt64                  = int64(1<<63 - 1)
	minInt64                  = int64(-1 << 63)
	maxDurationSeconds        = maxInt64 / nanosecondsPerSecond
	minDurationSeconds        = minInt64 / nanosecondsPerSecond
)

func FixedZone(name string, offset int) *Location {
	return &Location{name: name, offset: offset}
}

func (loc *Location) String() string {
	if loc == nil {
		return "UTC"
	}
	if loc.name == "" {
		return "UTC"
	}

	return loc.name
}

func Now() Time {
	loc := locationOrUTC(Local)
	startClock := kos.SystemTime()
	date := kos.SystemDate()
	endClock := kos.SystemTime()
	if clockSeconds(endClock) < clockSeconds(startClock) {
		date = kos.SystemDate()
	}

	seconds := unixFromCivil(
		int64(expandClockYear(date.Year)),
		int64(date.Month),
		int64(date.Day),
		int64(endClock.Hour),
		int64(endClock.Minute),
		int64(endClock.Second),
	) - int64(loc.offset)

	return Time{
		unixSeconds:  seconds,
		nanosecond:   0,
		monotonicNS:  int64(kos.UptimeNanoseconds()),
		hasMonotonic: true,
		loc:          loc,
	}
}

func Unix(sec int64, nsec int64) Time {
	sec, nsec = normalizeUnix(sec, nsec)
	return Time{
		unixSeconds: sec,
		nanosecond:  int32(nsec),
		loc:         locationOrUTC(Local),
	}
}

func Date(year int, month Month, day int, hour int, minute int, second int, nanosecond int, loc *Location) Time {
	loc = locationOrUTC(loc)
	seconds := unixFromCivil(
		int64(year),
		int64(month),
		int64(day),
		int64(hour),
		int64(minute),
		int64(second),
	)
	seconds -= int64(loc.offset)
	seconds, nsec := normalizeUnix(seconds, int64(nanosecond))
	return Time{
		unixSeconds: seconds,
		nanosecond:  int32(nsec),
		loc:         loc,
	}
}

func Parse(layout string, value string) (Time, error) {
	if layout == RFC1123 {
		return parseRFC1123(value)
	}
	switch layout {
	case layoutASN1UTCTimeMinutes, layoutASN1UTCTimeSeconds, layoutASN1GeneralizedTime:
		return parseNumericLayout(layout, value)
	}

	return parseNumericLayout(layout, value)
}

// ParseDuration parses a duration string.
// A duration string is a possibly signed sequence of decimal numbers, each with
// optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m".
// Valid time units are "ns", "us", "ms", "s", "m", "h".
func ParseDuration(value string) (Duration, error) {
	orig := value
	neg := false
	if value != "" {
		switch value[0] {
		case '-':
			neg = true
			value = value[1:]
		case '+':
			value = value[1:]
		}
	}
	if value == "0" {
		return 0, nil
	}
	if value == "" {
		return 0, errors.New("time: invalid duration " + orig)
	}

	var total int64
	for value != "" {
		hasInt := false
		intPart := int64(0)
		i := 0
		for i < len(value) && value[i] >= '0' && value[i] <= '9' {
			i++
		}
		if i > 0 {
			hasInt = true
			parsed, err := strconv.ParseInt(value[:i], 10, 64)
			if err != nil {
				return 0, errors.New("time: invalid duration " + orig)
			}
			intPart = parsed
		}
		value = value[i:]

		fracPart := int64(0)
		scale := int64(1)
		if len(value) > 0 && value[0] == '.' {
			value = value[1:]
			j := 0
			for j < len(value) && value[j] >= '0' && value[j] <= '9' {
				if fracPart <= maxInt64/10 {
					fracPart = fracPart*10 + int64(value[j]-'0')
					scale *= 10
				}
				j++
			}
			if j == 0 && !hasInt {
				return 0, errors.New("time: invalid duration " + orig)
			}
			value = value[j:]
		} else if !hasInt {
			return 0, errors.New("time: invalid duration " + orig)
		}

		k := 0
		for k < len(value) && (value[k] < '0' || value[k] > '9') && value[k] != '.' {
			k++
		}
		if k == 0 {
			return 0, errors.New("time: missing unit in duration " + orig)
		}
		unitStr := value[:k]
		value = value[k:]

		unit, ok := durationUnit(unitStr)
		if !ok {
			return 0, errors.New("time: unknown unit " + unitStr + " in duration " + orig)
		}

		if intPart != 0 {
			if intPart > maxInt64/int64(unit) {
				return 0, errors.New("time: invalid duration " + orig)
			}
			total += intPart * int64(unit)
		}
		if fracPart != 0 {
			total += int64(float64(fracPart) * (float64(unit) / float64(scale)))
		}
		if total > maxInt64 {
			return 0, errors.New("time: invalid duration " + orig)
		}
	}

	if neg {
		total = -total
	}
	return Duration(total), nil
}

func Sleep(duration Duration) {
	if duration <= 0 {
		return
	}

	centiseconds, remainder := divModUint64(unsignedAbsInt64(int64(duration)), uint32(nanosecondsPerCentisecond))
	if remainder != 0 {
		centiseconds++
	}

	for centiseconds > 0 {
		chunk := centiseconds
		if chunk > uint64(^uint32(0)) {
			chunk = uint64(^uint32(0))
		}

		kos.SleepCentiseconds(uint32(chunk))
		centiseconds -= chunk
	}
}

type Timer struct {
	C chan Time

	mu     sync.Mutex
	active bool
	seq    uint64
	fn     func()
}

func After(duration Duration) <-chan Time {
	return NewTimer(duration).C
}

func NewTimer(duration Duration) *Timer {
	timer := &Timer{
		C: make(chan Time, 1),
	}
	timer.Reset(duration)
	return timer
}

func AfterFunc(duration Duration, f func()) *Timer {
	timer := &Timer{fn: f}
	timer.Reset(duration)
	return timer
}

func (timer *Timer) Stop() bool {
	if timer == nil {
		return false
	}

	timer.mu.Lock()
	wasActive := timer.active
	if wasActive {
		timer.seq++
		timer.active = false
	}
	timer.mu.Unlock()
	return wasActive
}

func (timer *Timer) Reset(duration Duration) bool {
	if timer == nil {
		return false
	}

	timer.mu.Lock()
	wasActive := timer.active
	timer.seq++
	seq := timer.seq
	timer.active = true
	timer.mu.Unlock()

	go timer.waitAndFire(seq, duration)
	return wasActive
}

func (timer *Timer) waitAndFire(seq uint64, duration Duration) {
	Sleep(duration)

	timer.mu.Lock()
	if !timer.active || timer.seq != seq {
		timer.mu.Unlock()
		return
	}
	timer.active = false
	fn := timer.fn
	ch := timer.C
	timer.mu.Unlock()

	if fn != nil {
		fn()
		return
	}
	if ch == nil {
		return
	}
	select {
	case ch <- Now():
	default:
	}
}

func Until(value Time) Duration {
	return value.Sub(Now())
}

func Since(value Time) Duration {
	return Now().Sub(value)
}

func (value Time) Add(duration Duration) Time {
	result := Unix(value.unixSeconds, int64(value.nanosecond)+int64(duration))
	result.loc = value.Location()
	if value.hasMonotonic {
		result.monotonicNS = value.monotonicNS + int64(duration)
		result.hasMonotonic = true
	}

	return result
}

func (value Time) Sub(other Time) Duration {
	if value.hasMonotonic && other.hasMonotonic {
		return clampDurationParts(0, value.monotonicNS-other.monotonicNS)
	}

	return clampDurationParts(value.unixSeconds-other.unixSeconds, int64(value.nanosecond)-int64(other.nanosecond))
}

func (value Time) Before(other Time) bool {
	return value.compare(other) < 0
}

func (value Time) After(other Time) bool {
	return value.compare(other) > 0
}

func (value Time) Equal(other Time) bool {
	return value.compare(other) == 0
}

func (value Time) IsZero() bool {
	return value.unixSeconds == 0 && value.nanosecond == 0
}

func (value Time) UTC() Time {
	return value.In(UTC)
}

func (value Time) Local() Time {
	return value.In(Local)
}

func (value Time) In(loc *Location) Time {
	value.loc = locationOrUTC(loc)
	return value
}

func (value Time) Location() *Location {
	return locationOrUTC(value.loc)
}

func (value Time) Zone() (name string, offset int) {
	loc := value.Location()
	return loc.String(), loc.offset
}

func (value Time) String() string {
	return formatISO(value) + " " + zoneName(value.Location())
}

func (value Time) Unix() int64 {
	return value.unixSeconds
}

func (value Time) UnixNano() int64 {
	return value.unixSeconds*nanosecondsPerSecond + int64(value.nanosecond)
}

func (value Time) Nanosecond() int {
	return int(value.nanosecond)
}

func (value Time) Second() int {
	_, _, _, _, _, second := value.dateTime()
	return second
}

func (value Time) Minute() int {
	_, _, _, _, minute, _ := value.dateTime()
	return minute
}

func (value Time) Hour() int {
	_, _, _, hour, _, _ := value.dateTime()
	return hour
}

func (value Time) Day() int {
	_, _, day, _, _, _ := value.dateTime()
	return day
}

func (value Time) Date() (year int, month Month, day int) {
	year, month, day, _, _, _ = value.dateTime()
	return
}

func (value Time) Month() Month {
	_, month, _, _, _, _ := value.dateTime()
	return month
}

func (value Time) Year() int {
	year, _, _, _, _, _ := value.dateTime()
	return year
}

func (value Time) Clock() (hour int, min int, sec int) {
	_, _, _, hour, min, sec = value.dateTime()
	return
}

// AddDate returns the time corresponding to adding the given number of years,
// months, and days to t.
func (value Time) AddDate(years int, months int, days int) Time {
	year, month, day, hour, minute, second := value.dateTime()
	year += years

	monthIndex := int(month) - 1
	monthIndex += months
	year += monthIndex / 12
	monthIndex = monthIndex % 12
	if monthIndex < 0 {
		monthIndex += 12
		year--
	}

	month = Month(monthIndex + 1)
	return Date(year, month, day+days, hour, minute, second, value.Nanosecond(), value.Location())
}

func (value Time) Weekday() Weekday {
	loc := value.Location()
	days, _ := divModFloorInt64(value.unixSeconds+int64(loc.offset), uint32(secondsPerDay))
	weekday := (int(days) + 4) % 7
	if weekday < 0 {
		weekday += 7
	}
	return Weekday(weekday)
}

func (value Time) Format(layout string) string {
	switch layout {
	case RFC1123:
		return formatRFC1123(value)
	case "Mon, 02 Jan 2006":
		return formatDateWithWeekday(value)
	case "15:04:05 MST":
		return formatTimeWithZone(value)
	case "Mon, 02 Jan 2006 15:04:05 GMT":
		return formatRFC1123WithZone(value.UTC(), "GMT")
	case RFC3339, RFC3339Nano:
		return formatISOZ(value.UTC())
	case "2006-01-02T15:04:05.000Z":
		return formatISOZ(value.UTC())
	case layoutASN1UTCTimeMinutes:
		return formatCompactZoneTime(value, false, false)
	case layoutASN1UTCTimeSeconds:
		return formatCompactZoneTime(value, false, true)
	case layoutASN1GeneralizedTime:
		return formatCompactZoneTime(value, true, true)
	case "2006-01-02 15:04:05":
		return formatISO(value)
	case "2006-01-02":
		return formatDateOnly(value)
	case "15:04:05":
		return formatTimeOnly(value)
	default:
		return formatISO(value)
	}
}

func (value Time) compare(other Time) int {
	if value.hasMonotonic && other.hasMonotonic {
		switch {
		case value.monotonicNS < other.monotonicNS:
			return -1
		case value.monotonicNS > other.monotonicNS:
			return 1
		default:
			return 0
		}
	}

	switch {
	case value.unixSeconds < other.unixSeconds:
		return -1
	case value.unixSeconds > other.unixSeconds:
		return 1
	case value.nanosecond < other.nanosecond:
		return -1
	case value.nanosecond > other.nanosecond:
		return 1
	default:
		return 0
	}
}

func (value Time) dateTime() (year int, month Month, day int, hour int, minute int, second int) {
	loc := value.Location()
	days, daySeconds := divModFloorInt64(value.unixSeconds+int64(loc.offset), uint32(secondsPerDay))

	year, month, day = civilFromDays(days)
	hourQuotient, hourRemainder := divModUint64(uint64(daySeconds), uint32(secondsPerHour))
	minuteQuotient, secondRemainder := divModUint64(uint64(hourRemainder), uint32(secondsPerMinute))
	hour = int(hourQuotient)
	minute = int(minuteQuotient)
	second = int(secondRemainder)
	return
}

func locationOrUTC(loc *Location) *Location {
	if loc != nil {
		return loc
	}
	if UTC != nil {
		return UTC
	}

	return &Location{name: "UTC", offset: 0}
}

func normalizeUnix(seconds int64, nanoseconds int64) (int64, int64) {
	if nanoseconds >= 0 {
		quotient, remainder := divModUint64(uint64(nanoseconds), uint32(nanosecondsPerSecond))
		seconds += int64(quotient)
		nanoseconds = int64(remainder)
	} else {
		quotient, remainder := divModUint64(unsignedAbsInt64(nanoseconds), uint32(nanosecondsPerSecond))
		seconds -= int64(quotient)
		nanoseconds = -int64(remainder)
	}
	if nanoseconds < 0 {
		nanoseconds += nanosecondsPerSecond
		seconds--
	}

	return seconds, nanoseconds
}

func clampDurationParts(seconds int64, nanoseconds int64) Duration {
	if seconds > maxDurationSeconds {
		return Duration(maxInt64)
	}
	if seconds < minDurationSeconds {
		return Duration(minInt64)
	}

	total := seconds * nanosecondsPerSecond
	if nanoseconds > 0 && total > maxInt64-nanoseconds {
		return Duration(maxInt64)
	}
	if nanoseconds < 0 && total < minInt64-nanoseconds {
		return Duration(minInt64)
	}

	return Duration(total + nanoseconds)
}

func clockSeconds(value kos.ClockTime) int64 {
	return int64(value.Hour)*secondsPerHour +
		int64(value.Minute)*secondsPerMinute +
		int64(value.Second)
}

func expandClockYear(year byte) int {
	return 2000 + int(year)
}

func unixFromCivil(year int64, month int64, day int64, hour int64, minute int64, second int64) int64 {
	days := daysFromCivil(year, month, day)
	return days*secondsPerDay + hour*secondsPerHour + minute*secondsPerMinute + second
}

func daysFromCivil(year int64, month int64, day int64) int64 {
	if month <= 2 {
		year--
	}

	era, _ := divModFloorInt64(year, 400)
	yearOfEra := uint32(year - era*400)
	monthPrime := uint32(month)
	if monthPrime > 2 {
		monthPrime -= 3
	} else {
		monthPrime += 9
	}

	dayOfYear := ((153 * monthPrime) + 2) / 5
	dayOfYear += uint32(day) - 1
	dayOfEra := uint64(yearOfEra)*365 + uint64(yearOfEra/4) - uint64(yearOfEra/100) + uint64(dayOfYear)
	return era*daysPer400Years + int64(dayOfEra) - unixToCivilEpochDays
}

func civilFromDays(days int64) (year int, month Month, day int) {
	days += unixToCivilEpochDays
	era, _ := divModFloorInt64(days, uint32(daysPer400Years))
	dayOfEra := uint32(days - era*daysPer400Years)
	yearOfEra := (dayOfEra - dayOfEra/1460 + dayOfEra/36524 - dayOfEra/146096) / 365
	yearValue := int64(yearOfEra) + era*400
	dayOfYear := dayOfEra - (365*yearOfEra + yearOfEra/4 - yearOfEra/100)
	monthPrime := (5*dayOfYear + 2) / 153

	day = int(dayOfYear - ((153*monthPrime + 2) / 5) + 1)
	if monthPrime < 10 {
		month = Month(monthPrime + 3)
	} else {
		month = Month(monthPrime - 9)
		yearValue++
	}
	year = int(yearValue)
	return
}

func unsignedAbsInt64(value int64) uint64 {
	if value >= 0 {
		return uint64(value)
	}

	return uint64(^value) + 1
}

func divModUint64(value uint64, divisor uint32) (uint64, uint32) {
	quotient := uint64(0)
	remainder := uint64(0)
	divisor64 := uint64(divisor)

	for shift := uint(64); shift > 0; shift-- {
		remainder = (remainder << 1) | ((value >> (shift - 1)) & 1)
		if remainder >= divisor64 {
			remainder -= divisor64
			quotient |= uint64(1) << (shift - 1)
		}
	}

	return quotient, uint32(remainder)
}

func divModFloorInt64(value int64, divisor uint32) (int64, uint32) {
	if value >= 0 {
		quotient, remainder := divModUint64(uint64(value), divisor)
		return int64(quotient), remainder
	}

	quotient, remainder := divModUint64(unsignedAbsInt64(value), divisor)
	if remainder == 0 {
		return -int64(quotient), 0
	}

	return -int64(quotient) - 1, divisor - remainder
}

var shortWeekdayNames = [...]string{
	"Sun",
	"Mon",
	"Tue",
	"Wed",
	"Thu",
	"Fri",
	"Sat",
}

var shortMonthNames = [...]string{
	"Jan",
	"Feb",
	"Mar",
	"Apr",
	"May",
	"Jun",
	"Jul",
	"Aug",
	"Sep",
	"Oct",
	"Nov",
	"Dec",
}

func formatRFC1123(value Time) string {
	return formatRFC1123WithZone(value, zoneName(value.Location()))
}

func formatRFC1123WithZone(value Time, zone string) string {
	year, month, day, hour, minute, second := value.dateTime()
	weekday := value.Weekday()
	return shortWeekdayNames[weekday] + ", " +
		pad2(day) + " " +
		shortMonthNames[int(month)-1] + " " +
		pad4(year) + " " +
		pad2(hour) + ":" + pad2(minute) + ":" + pad2(second) + " " +
		zone
}

func formatDateWithWeekday(value Time) string {
	year, month, day, _, _, _ := value.dateTime()
	weekday := value.Weekday()
	return shortWeekdayNames[weekday] + ", " +
		pad2(day) + " " +
		shortMonthNames[int(month)-1] + " " +
		pad4(year)
}

func formatTimeWithZone(value Time) string {
	_, _, _, hour, minute, second := value.dateTime()
	return pad2(hour) + ":" + pad2(minute) + ":" + pad2(second) + " " + zoneName(value.Location())
}

func formatISO(value Time) string {
	year, month, day, hour, minute, second := value.dateTime()
	return pad4(year) + "-" + pad2(int(month)) + "-" + pad2(day) + " " +
		pad2(hour) + ":" + pad2(minute) + ":" + pad2(second)
}

func formatISOZ(value Time) string {
	year, month, day, hour, minute, second := value.dateTime()
	millisecond := int(value.nanosecond / 1000000)
	return pad4(year) + "-" + pad2(int(month)) + "-" + pad2(day) + "T" +
		pad2(hour) + ":" + pad2(minute) + ":" + pad2(second) + "." + pad3(millisecond) + "Z"
}

func formatDateOnly(value Time) string {
	year, month, day, _, _, _ := value.dateTime()
	return pad4(year) + "-" + pad2(int(month)) + "-" + pad2(day)
}

func formatCompactZoneTime(value Time, fullYear bool, withSeconds bool) string {
	year, month, day, hour, minute, second := value.dateTime()
	text := ""
	if fullYear {
		text = pad4(year)
	} else {
		text = pad2(year % 100)
	}
	text += pad2(int(month)) + pad2(day) + pad2(hour) + pad2(minute)
	if withSeconds {
		text += pad2(second)
	}

	loc := value.Location()
	if loc.offset == 0 {
		return text + "Z"
	}
	return text + formatZoneOffset(loc.offset)
}

func formatTimeOnly(value Time) string {
	_, _, _, hour, minute, second := value.dateTime()
	return pad2(hour) + ":" + pad2(minute) + ":" + pad2(second)
}

func zoneName(loc *Location) string {
	loc = locationOrUTC(loc)
	if loc.name != "" {
		return loc.name
	}
	if loc.offset == 0 {
		return "UTC"
	}
	return formatZoneOffset(loc.offset)
}

func formatZoneOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return sign + pad2(hours) + pad2(minutes)
}

func pad2(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func pad3(value int) string {
	if value < 10 {
		return "00" + strconv.Itoa(value)
	}
	if value < 100 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func pad4(value int) string {
	if value >= 1000 {
		return strconv.Itoa(value)
	}
	if value >= 100 {
		return "0" + strconv.Itoa(value)
	}
	if value >= 10 {
		return "00" + strconv.Itoa(value)
	}
	return "000" + strconv.Itoa(value)
}

func parseRFC1123(value string) (Time, error) {
	fields := strings.Fields(value)
	if len(fields) < 5 {
		return Time{}, errors.New("time: invalid RFC1123")
	}

	dayValue, ok := parseFixedInt(fields[1], 0, len(fields[1]))
	if !ok {
		return Time{}, errors.New("time: invalid day")
	}

	monthValue, ok := parseMonthShort(fields[2])
	if !ok {
		return Time{}, errors.New("time: invalid month")
	}

	yearValue, ok := parseFixedInt(fields[3], 0, len(fields[3]))
	if !ok {
		return Time{}, errors.New("time: invalid year")
	}

	timeParts := strings.Split(fields[4], ":")
	if len(timeParts) != 3 {
		return Time{}, errors.New("time: invalid time")
	}
	hourValue, ok := parseFixedInt(timeParts[0], 0, len(timeParts[0]))
	if !ok {
		return Time{}, errors.New("time: invalid hour")
	}
	minuteValue, ok := parseFixedInt(timeParts[1], 0, len(timeParts[1]))
	if !ok {
		return Time{}, errors.New("time: invalid minute")
	}
	secondValue, ok := parseFixedInt(timeParts[2], 0, len(timeParts[2]))
	if !ok {
		return Time{}, errors.New("time: invalid second")
	}

	zone := "UTC"
	if len(fields) >= 6 {
		zone = fields[5]
	}
	loc := locationFromZone(zone)

	return Date(yearValue, monthValue, dayValue, hourValue, minuteValue, secondValue, 0, loc), nil
}

func parseMonthShort(value string) (Month, bool) {
	switch value {
	case "Jan":
		return January, true
	case "Feb":
		return February, true
	case "Mar":
		return March, true
	case "Apr":
		return April, true
	case "May":
		return May, true
	case "Jun":
		return June, true
	case "Jul":
		return July, true
	case "Aug":
		return August, true
	case "Sep":
		return September, true
	case "Oct":
		return October, true
	case "Nov":
		return November, true
	case "Dec":
		return December, true
	default:
		return January, false
	}
}

func locationFromZone(zone string) *Location {
	if zone == "GMT" || zone == "UTC" {
		return UTC
	}
	return UTC
}

func parseNumericLayout(layout string, value string) (Time, error) {
	year := -1
	month := 1
	day := 1
	hour := 0
	minute := 0
	second := 0
	millisecond := 0
	offsetSeconds := 0
	offsetSet := false

	i := 0
	j := 0
	for i < len(layout) {
		switch {
		case strings.HasPrefix(layout[i:], "2006"):
			parsed, ok := parseFixedInt(value, j, j+4)
			if !ok {
				return Time{}, errors.New("time: invalid year")
			}
			year = parsed
			i += 4
			j += 4
		case strings.HasPrefix(layout[i:], "06"):
			parsed, ok := parseFixedInt(value, j, j+2)
			if !ok {
				return Time{}, errors.New("time: invalid year")
			}
			if parsed >= 69 {
				year = 1900 + parsed
			} else {
				year = 2000 + parsed
			}
			i += 2
			j += 2
		case strings.HasPrefix(layout[i:], "000"):
			parsed, ok := parseFixedInt(value, j, j+3)
			if !ok {
				return Time{}, errors.New("time: invalid millisecond")
			}
			millisecond = parsed
			i += 3
			j += 3
		case strings.HasPrefix(layout[i:], "Z0700"):
			if j >= len(value) {
				return Time{}, errors.New("time: invalid zone")
			}
			if value[j] == 'Z' {
				offsetSeconds = 0
				offsetSet = true
				i += 5
				j++
				break
			}
			if j+5 > len(value) {
				return Time{}, errors.New("time: invalid zone")
			}
			sign := value[j]
			if sign != '+' && sign != '-' {
				return Time{}, errors.New("time: invalid zone")
			}
			hours, ok := parseFixedInt(value, j+1, j+3)
			if !ok {
				return Time{}, errors.New("time: invalid zone hour")
			}
			minutes, ok := parseFixedInt(value, j+3, j+5)
			if !ok {
				return Time{}, errors.New("time: invalid zone minute")
			}
			offsetSeconds = (hours*60 + minutes) * 60
			if sign == '-' {
				offsetSeconds = -offsetSeconds
			}
			offsetSet = true
			i += 5
			j += 5
		case strings.HasPrefix(layout[i:], "-0700"):
			if j+5 > len(value) {
				return Time{}, errors.New("time: invalid zone")
			}
			sign := value[j]
			if sign != '+' && sign != '-' {
				return Time{}, errors.New("time: invalid zone")
			}
			hours, ok := parseFixedInt(value, j+1, j+3)
			if !ok {
				return Time{}, errors.New("time: invalid zone hour")
			}
			minutes, ok := parseFixedInt(value, j+3, j+5)
			if !ok {
				return Time{}, errors.New("time: invalid zone minute")
			}
			offsetSeconds = (hours*60 + minutes) * 60
			if sign == '-' {
				offsetSeconds = -offsetSeconds
			}
			offsetSet = true
			i += 5
			j += 5
		case strings.HasPrefix(layout[i:], "15"):
			parsed, ok := parseFixedInt(value, j, j+2)
			if !ok {
				return Time{}, errors.New("time: invalid hour")
			}
			hour = parsed
			i += 2
			j += 2
		case strings.HasPrefix(layout[i:], "04"):
			parsed, ok := parseFixedInt(value, j, j+2)
			if !ok {
				return Time{}, errors.New("time: invalid minute")
			}
			minute = parsed
			i += 2
			j += 2
		case strings.HasPrefix(layout[i:], "05"):
			parsed, ok := parseFixedInt(value, j, j+2)
			if !ok {
				return Time{}, errors.New("time: invalid second")
			}
			second = parsed
			i += 2
			j += 2
		case strings.HasPrefix(layout[i:], "01"):
			parsed, ok := parseFixedInt(value, j, j+2)
			if !ok {
				return Time{}, errors.New("time: invalid month")
			}
			month = parsed
			i += 2
			j += 2
		case strings.HasPrefix(layout[i:], "02"):
			parsed, ok := parseFixedInt(value, j, j+2)
			if !ok {
				return Time{}, errors.New("time: invalid day")
			}
			day = parsed
			i += 2
			j += 2
		default:
			if j >= len(value) || layout[i] != value[j] {
				return Time{}, errors.New("time: invalid layout")
			}
			i++
			j++
		}
	}

	if j != len(value) {
		return Time{}, errors.New("time: invalid value")
	}
	if year < 0 {
		return Time{}, errors.New("time: invalid year")
	}

	loc := UTC
	if offsetSet {
		loc = FixedZone("", offsetSeconds)
	}
	return Date(year, Month(month), day, hour, minute, second, millisecond*1000000, loc), nil
}

func parseFixedInt(value string, start int, end int) (int, bool) {
	if start < 0 || end > len(value) || start >= end {
		return 0, false
	}
	result := 0
	for i := start; i < end; i++ {
		ch := value[i]
		if ch < '0' || ch > '9' {
			return 0, false
		}
		result = result*10 + int(ch-'0')
	}
	return result, true
}
