package main

import "kos"

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var hexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
var decimalPowers = [...]uint64{
	10000000000000000000,
	1000000000000000000,
	100000000000000000,
	10000000000000000,
	1000000000000000,
	100000000000000,
	10000000000000,
	1000000000000,
	100000000000,
	10000000000,
	1000000000,
	100000000,
	10000000,
	1000000,
	100000,
	10000,
	1000,
	100,
	10,
	1,
}

func formatInt(value int) string {
	if value < 0 {
		return "-" + formatUint64Decimal(uint64(^value)+1)
	}

	return formatUint64Decimal(uint64(value))
}

func formatUint64Decimal(value uint64) string {
	if value == 0 {
		return "0"
	}

	text := ""
	started := false

	for index := 0; index < len(decimalPowers); index++ {
		digit := uint32(0)
		for value >= decimalPowers[index] {
			value -= decimalPowers[index]
			digit++
		}

		if digit != 0 || started {
			text += decimalDigits[digit]
			started = true
		}
	}

	return text
}

func formatHexBytes(data []byte) string {
	if len(data) == 0 {
		return ""
	}

	text := ""
	for index := 0; index < len(data); index++ {
		text += hexDigits[uint32(data[index]>>4)]
		text += hexDigits[uint32(data[index]&0x0F)]
	}

	return text
}

func trimTrailingNewline(value string) string {
	if len(value) > 0 && value[len(value)-1] == '\n' {
		return value[:len(value)-1]
	}

	return value
}

func hasTrailingNewline(value string) bool {
	return len(value) > 0 && value[len(value)-1] == '\n'
}

func formatFileSystemStatus(status kos.FileSystemStatus) string {
	switch status {
	case kos.FileSystemOK:
		return "ok"
	case kos.FileSystemUnsupported:
		return "unsupported (2)"
	case kos.FileSystemNotFound:
		return "not found (5)"
	case kos.FileSystemEOF:
		return "eof (6)"
	case kos.FileSystemBadPointer:
		return "bad pointer (7)"
	case kos.FileSystemDiskFull:
		return "disk full (8)"
	case kos.FileSystemInternalError:
		return "internal (9)"
	case kos.FileSystemAccessDenied:
		return "denied (10)"
	case kos.FileSystemDeviceError:
		return "device (11)"
	case kos.FileSystemNeedsMoreMemory:
		return "memory (12)"
	}

	return "status " + formatInt(int(status))
}
