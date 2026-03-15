package kos

type ClipboardType uint32
type ClipboardEncoding uint32
type ClipboardStatus int32

const (
	ClipboardTypeText       ClipboardType = 0
	ClipboardTypeTextBlock  ClipboardType = 1
	ClipboardTypeImage      ClipboardType = 2
	ClipboardTypeRaw        ClipboardType = 3
)

const (
	ClipboardEncodingUTF    ClipboardEncoding = 0
	ClipboardEncodingCP866  ClipboardEncoding = 1
	ClipboardEncodingCP1251 ClipboardEncoding = 2
)

const (
	ClipboardOK          ClipboardStatus = 0
	ClipboardError       ClipboardStatus = 1
	ClipboardUnavailable ClipboardStatus = -1
)

var clipboardHeapReady bool

func ClipboardSlotCount() (int, ClipboardStatus) {
	count := ClipboardSlotCountRaw()
	if count < 0 {
		return 0, ClipboardUnavailable
	}
	return count, ClipboardOK
}

func ClipboardSlotData(slot int) (uint32, ClipboardStatus) {
	ensureClipboardHeap()
	ptr := ClipboardSlotDataRaw(slot)
	if ptr == 0 {
		return 0, ClipboardError
	}
	if ptr == 1 {
		return 0, ClipboardError
	}
	if ptr == 0xFFFFFFFF {
		return 0, ClipboardUnavailable
	}
	return ptr, ClipboardOK
}

func ClipboardWrite(data []byte) ClipboardStatus {
	if len(data) == 0 {
		return ClipboardError
	}
	status := ClipboardWriteRaw(uint32(len(data)), &data[0])
	return clipboardStatusFromRaw(status)
}

func ClipboardDeleteLast() ClipboardStatus {
	return clipboardStatusFromRaw(ClipboardDeleteLastRaw())
}

func ClipboardUnlockBuffer() ClipboardStatus {
	return clipboardStatusFromRaw(ClipboardUnlockBufferRaw())
}

func ClipboardCopyText(text string) ClipboardStatus {
	return ClipboardCopyTextWithEncoding(text, ClipboardEncodingUTF)
}

func ClipboardCopyTextWithEncoding(text string, encoding ClipboardEncoding) ClipboardStatus {
	data := []byte(text)
	size := 12 + len(data) + 1
	buffer := make([]byte, size)
	putUint32LE(buffer, 0, uint32(size))
	putUint32LE(buffer, 4, uint32(ClipboardTypeText))
	putUint32LE(buffer, 8, uint32(encoding))
	copy(buffer[12:], data)
	buffer[len(buffer)-1] = 0
	return ClipboardWrite(buffer)
}

func clipboardStatusFromRaw(value int) ClipboardStatus {
	switch value {
	case 0:
		return ClipboardOK
	case 1:
		return ClipboardError
	default:
		return ClipboardUnavailable
	}
}

func ensureClipboardHeap() {
	if clipboardHeapReady {
		return
	}
	if InitHeapRaw() != 0 {
		clipboardHeapReady = true
	}
}
