package kos

func DebugHex(value uint32) {
	DebugOutHex(value)
}

func DebugChar(value byte) {
	DebugOutChar(value)
}

func DebugString(value string) {
	DebugOutStr(value)
}

func DebugReadByte() (byte, bool) {
	value := DebugReadRaw()
	return byte(value), value>>8 != 0
}
