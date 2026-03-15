package kos

func ReservePorts(start uint16, end uint16) bool {
	return SetPortsRaw(0, uint32(start), uint32(end)) == 0
}

func ReleasePorts(start uint16, end uint16) bool {
	return SetPortsRaw(1, uint32(start), uint32(end)) == 0
}

func WritePortByte(port uint16, value byte) {
	PortWriteByteRaw(uint32(port), value)
}

func WritePortString(port uint16, value string) {
	for index := 0; index < len(value); index++ {
		WritePortByte(port, value[index])
	}
}
