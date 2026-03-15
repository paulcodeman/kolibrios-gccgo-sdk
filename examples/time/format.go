package main

import "time"

var timeprobeDecimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var timeprobeHexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}

func formatInt64(value int64) string {
	if value < 0 {
		return "-" + formatUint32(uint32(-value))
	}

	return formatUint32(uint32(value))
}

func formatUint32(value uint32) string {
	if value < 10 {
		return timeprobeDecimalDigits[value]
	}

	return formatUint32(value/10) + timeprobeDecimalDigits[value%10]
}

func formatTwoDigits(value int) string {
	return timeprobeDecimalDigits[value/10] + timeprobeDecimalDigits[value%10]
}

func formatFourDigits(value int) string {
	return formatTwoDigits(value/100) + formatTwoDigits(value%100)
}

func formatTimeStamp(value time.Time) string {
	return formatFourDigits(value.Year()) + "-" +
		formatTwoDigits(int(value.Month())) + "-" +
		formatTwoDigits(value.Day()) + " " +
		formatTwoDigits(value.Hour()) + ":" +
		formatTwoDigits(value.Minute()) + ":" +
		formatTwoDigits(value.Second())
}

func formatDurationMilliseconds(value time.Duration) string {
	if value < 0 {
		return "-" + formatDurationMilliseconds(-value)
	}

	milliseconds := int(value) / int(time.Millisecond)
	return formatUint32(uint32(milliseconds)) + " ms"
}

func formatBoolWord(value bool) string {
	if value {
		return "ok"
	}

	return "fail"
}

func formatCentisecondsAsSeconds(value uint32) string {
	return formatUint32(value/100) + "." + formatTwoDigits(int(value%100)) + " s"
}

func formatHex64(value uint64) string {
	return "0x" +
		hexDigit64(value, 60) +
		hexDigit64(value, 56) +
		hexDigit64(value, 52) +
		hexDigit64(value, 48) +
		hexDigit64(value, 44) +
		hexDigit64(value, 40) +
		hexDigit64(value, 36) +
		hexDigit64(value, 32) +
		hexDigit64(value, 28) +
		hexDigit64(value, 24) +
		hexDigit64(value, 20) +
		hexDigit64(value, 16) +
		hexDigit64(value, 12) +
		hexDigit64(value, 8) +
		hexDigit64(value, 4) +
		hexDigit64(value, 0)
}

func hexDigit64(value uint64, shift uint) string {
	return timeprobeHexDigits[uint32((value>>shift)&0x0F)]
}
