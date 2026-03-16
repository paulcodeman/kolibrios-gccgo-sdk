package main

import "kos"

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var hexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}

func formatInt(value int) string {
	if value < 0 {
		return "-" + formatUint32(uint32(-value))
	}

	return formatUint32(uint32(value))
}

func formatUint32(value uint32) string {
	if value < 10 {
		return decimalDigits[value]
	}

	return formatUint32(value/10) + decimalDigits[value%10]
}

func formatHex8(value byte) string {
	return "0x" +
		hexDigits[(value>>4)&0x0F] +
		hexDigits[value&0x0F]
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

func formatKernelVersion(info kos.KernelVersionInfo) string {
	return formatUint32(uint32(info.Major)) + "." +
		formatUint32(uint32(info.Minor)) + "." +
		formatUint32(uint32(info.Patch)) + "." +
		formatUint32(uint32(info.Build))
}

func formatKernelABI(info kos.KernelVersionInfo) string {
	return formatUint32(uint32(info.ABIMajor)) + "." +
		formatUint32(uint32(info.ABIMinor))
}

func formatRect(rect kos.Rect) string {
	return "(" + formatInt(rect.Left) + "," + formatInt(rect.Top) + ")-(" +
		formatInt(rect.Right) + "," + formatInt(rect.Bottom) + ")"
}

func formatSkinMargins(margins kos.SkinMargins) string {
	return "L" + formatInt(margins.Left) +
		" T" + formatInt(margins.Top) +
		" R" + formatInt(margins.Right) +
		" B" + formatInt(margins.Bottom)
}

func formatKeyboardLanguage(language kos.KeyboardLanguage) string {
	switch language {
	case kos.KeyboardLanguageEnglish:
		return "en (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageFinnish:
		return "fi (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageGerman:
		return "ge (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageRussian:
		return "ru (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageFrench:
		return "fr (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageEstonian:
		return "et (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageUkrainian:
		return "ua (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageItalian:
		return "it (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageBelarusian:
		return "be (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageSpanish:
		return "sp (" + formatInt(int(language)) + ")"
	case kos.KeyboardLanguageCatalan:
		return "ca (" + formatInt(int(language)) + ")"
	}

	return "unknown (" + formatInt(int(language)) + ")"
}

func formatLayoutChecksums(normal kos.KeyboardLayoutTable, shift kos.KeyboardLayoutTable, alt kos.KeyboardLayoutTable) string {
	return "N=" + formatHex32(layoutChecksum(normal)) +
		" S=" + formatHex32(layoutChecksum(shift)) +
		" A=" + formatHex32(layoutChecksum(alt))
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

func layoutChecksum(layout kos.KeyboardLayoutTable) uint32 {
	checksum := uint32(2166136261)

	for index := 0; index < len(layout); index++ {
		checksum ^= uint32(layout[index])
		checksum *= 16777619
	}

	return checksum
}
