package main

import "kos"

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var hexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}

func formatUint32(value uint32) string {
	if value < 10 {
		return decimalDigits[value]
	}

	return formatUint32(value/10) + decimalDigits[value%10]
}

func formatHex32(value uint32) string {
	return "0x" +
		hexDigits[(value>>28)&0x0F] +
		hexDigits[(value>>24)&0x0F] +
		hexDigits[(value>>20)&0x0F] +
		hexDigits[(value>>16)&0x0F] +
		hexDigits[(value>>12)&0x0F] +
		hexDigits[(value>>8)&0x0F] +
		hexDigits[(value>>4)&0x0F] +
		hexDigits[value&0x0F]
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
	return hexDigits[uint32((value>>shift)&0x0F)]
}

func formatHexByte(value byte) string {
	return hexDigits[(value>>4)&0x0F] + hexDigits[value&0x0F]
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

	return "status " + formatUint32(uint32(status))
}

func formatBytePreview(data []byte) string {
	if len(data) == 0 {
		return "empty"
	}

	preview := formatHexByte(data[0])
	for index := 1; index < len(data); index++ {
		preview += " " + formatHexByte(data[index])
	}

	return preview
}
