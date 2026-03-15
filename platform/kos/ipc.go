package kos

const ipcBufferHeaderSize = 8

type IPCBufferSummary struct {
	Used          uint32
	MessageCount  uint32
	LastSenderPID uint32
	LastLength    uint32
	LastFirstByte byte
	LastHasData   bool
}

func RegisterIPCBuffer(buffer []byte) IPCStatus {
	return IPCStatus(SetIPCArea(bufferPointer(buffer), uint32(len(buffer))))
}

func SendIPCRaw(pid uint32, data *byte, size uint32) IPCStatus {
	return IPCStatus(SendIPCMessage(pid, data, size))
}

func SendIPC(pid uint32, data []byte) IPCStatus {
	return IPCStatus(SendIPCMessage(pid, bufferPointer(data), uint32(len(data))))
}

func IPCBufferUsed(buffer []byte) uint32 {
	if len(buffer) < ipcBufferHeaderSize {
		return 0
	}

	used := littleEndianUint32(buffer, 4)
	maxUsed := uint32(len(buffer) - ipcBufferHeaderSize)
	if used > maxUsed {
		return maxUsed
	}

	return used
}

func IPCBufferIsLocked(buffer []byte) bool {
	if len(buffer) < 4 {
		return false
	}

	return littleEndianUint32(buffer, 0) != 0
}

func ResetIPCBuffer(buffer []byte) {
	if len(buffer) < ipcBufferHeaderSize {
		return
	}

	writeUint32(buffer, 4, 0)
}

func InspectIPCBuffer(buffer []byte) IPCBufferSummary {
	summary := IPCBufferSummary{
		Used: IPCBufferUsed(buffer),
	}

	if len(buffer) < ipcBufferHeaderSize {
		return summary
	}

	limit := int(summary.Used) + ipcBufferHeaderSize
	if limit > len(buffer) {
		limit = len(buffer)
	}

	index := ipcBufferHeaderSize
	for index+ipcBufferHeaderSize <= limit {
		length := littleEndianUint32(buffer, index+4)
		next := index + ipcBufferHeaderSize + int(length)
		if next > limit {
			break
		}

		summary.MessageCount++
		summary.LastSenderPID = littleEndianUint32(buffer, index)
		summary.LastLength = length
		summary.LastHasData = length > 0
		if length > 0 {
			summary.LastFirstByte = buffer[index+ipcBufferHeaderSize]
		} else {
			summary.LastFirstByte = 0
		}

		index = next
	}

	return summary
}

func bufferPointer(data []byte) *byte {
	if len(data) == 0 {
		return nil
	}

	return &data[0]
}

func writeUint32(buffer []byte, offset int, value uint32) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
	buffer[offset+2] = byte(value >> 16)
	buffer[offset+3] = byte(value >> 24)
}
