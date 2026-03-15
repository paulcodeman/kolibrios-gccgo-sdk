package kos

import "unsafe"

const CurrentThreadSlot = -1

type ThreadInfo struct {
	CPUUsage            uint32
	WindowStackPosition uint16
	WindowStackSlot     uint16
	Name                string
	ProcessAddress      uint32
	UsedMemoryMinus1    uint32
	ID                  uint32
	WindowPosition      Point
	WindowSize          Point
	Status              ThreadStatus
	ClientPosition      Point
	ClientSize          Point
	WindowState         WindowState
	EventMask           EventMask
	KeyboardMode        KeyboardMode
}

type ThreadStart func()

type ThreadDebugEvent struct {
	Stage     string
	Entry     uint32
	Stack     uint32
	Record    uint32
	RawID     int
	ThreadID  uint32
	StackSize int
}

// ThreadDebug is an optional hook for diagnosing thread creation and startup.
// When set, it is called at key points during CreateThread and ThreadBootstrap.
var ThreadDebug func(ThreadDebugEvent)

const (
	DefaultThreadStackSize = 0x10000
	MinThreadStackSize     = 0x1000
)

type threadStartRecord struct {
	fn    ThreadStart
	stack []byte
	tid   uint32
}

var threadStartRecords []*threadStartRecord

func ReadThreadInfo(slot int) (info ThreadInfo, maxSlot int, ok bool) {
	var buffer [1024]byte

	maxSlot = GetThreadInfo(&buffer[0], slot)
	if maxSlot < 0 {
		return ThreadInfo{}, maxSlot, false
	}

	info = ThreadInfo{
		CPUUsage:            littleEndianUint32(buffer[:], 0),
		WindowStackPosition: littleEndianUint16(buffer[:], 4),
		WindowStackSlot:     littleEndianUint16(buffer[:], 6),
		Name:                trimASCIIField(buffer[10:21]),
		ProcessAddress:      littleEndianUint32(buffer[:], 22),
		UsedMemoryMinus1:    littleEndianUint32(buffer[:], 26),
		ID:                  littleEndianUint32(buffer[:], 30),
		WindowPosition: Point{
			X: int(littleEndianUint32(buffer[:], 34)),
			Y: int(littleEndianUint32(buffer[:], 38)),
		},
		WindowSize: Point{
			X: int(littleEndianUint32(buffer[:], 42)),
			Y: int(littleEndianUint32(buffer[:], 46)),
		},
		Status: ThreadStatus(littleEndianUint16(buffer[:], 50)),
		ClientPosition: Point{
			X: int(littleEndianUint32(buffer[:], 54)),
			Y: int(littleEndianUint32(buffer[:], 58)),
		},
		ClientSize: Point{
			X: int(littleEndianUint32(buffer[:], 62)),
			Y: int(littleEndianUint32(buffer[:], 66)),
		},
		WindowState:  WindowState(buffer[70]),
		EventMask:    EventMask(littleEndianUint32(buffer[:], 71)),
		KeyboardMode: KeyboardMode(buffer[75]),
	}

	return info, maxSlot, true
}

func ReadCurrentThreadInfo() (info ThreadInfo, maxSlot int, ok bool) {
	return ReadThreadInfo(CurrentThreadSlot)
}

func CurrentThreadID() (id uint32, ok bool) {
	var buffer [1024]byte

	maxSlot := GetThreadInfo(&buffer[0], CurrentThreadSlot)
	if maxSlot < 0 {
		return 0, false
	}

	return littleEndianUint32(buffer[:], 30), true
}

func CurrentThreadSlotIndex() (slot int, ok bool) {
	if slot = GetCurrentThreadSlotRaw(); slot > 0 {
		return slot, true
	}
	id, ok := CurrentThreadID()
	if !ok {
		return 0, false
	}

	_, maxSlot, ok := ReadCurrentThreadInfo()
	if !ok {
		return 0, false
	}

	for slot = 1; slot <= maxSlot; slot++ {
		info, _, ok := ReadThreadInfo(slot)
		if !ok {
			continue
		}

		if info.ID == id {
			return slot, true
		}
	}

	return 0, false
}

// SetRuntimeThreads configures how many OS threads the runtime should use.
// Returns the effective thread count.
func SetRuntimeThreads(count int) int {
	if count < 1 {
		count = 1
	}
	return int(SetRuntimeThreadsRaw(uint32(count)))
}

// CreateThread spawns a KolibriOS thread that runs fn using the bootstrap runtime.
// Note: the bootstrap runtime GC is single-threaded; avoid concurrent Go
// execution across threads unless you accept the current limitations.
func CreateThread(fn ThreadStart, stackSize int) (tid uint32, ok bool) {
	if fn == nil {
		return 0, false
	}
	if stackSize <= 0 {
		stackSize = DefaultThreadStackSize
	}
	if stackSize < MinThreadStackSize {
		stackSize = MinThreadStackSize
	}

	stack := make([]byte, stackSize)
	record := &threadStartRecord{
		fn:    fn,
		stack: stack,
	}
	threadStartRecords = append(threadStartRecords, record)
	recordPtr := uint32(uintptr(unsafe.Pointer(record)))
	if ThreadDebug != nil {
		ThreadDebug(ThreadDebugEvent{
			Stage:     "create_prepare",
			Record:    recordPtr,
			StackSize: len(stack),
		})
	}

	stackPtr, ok := threadStackPointer(stack)
	if !ok {
		threadStartRecords = threadStartRecords[:len(threadStartRecords)-1]
		return 0, false
	}
	*(*uint32)(unsafe.Pointer(uintptr(stackPtr))) = uint32(uintptr(unsafe.Pointer(record)))

	entry := ThreadEntryAddrRaw()
	if ThreadDebug != nil {
		ThreadDebug(ThreadDebugEvent{
			Stage:  "create_call",
			Entry:  entry,
			Stack:  stackPtr,
			Record: recordPtr,
		})
	}
	rawID := CreateThreadRaw(entry, stackPtr)
	if rawID < 0 && hasFreeThreadSlot() {
		// Retry once in case the thread slot table hasn't settled yet.
		PollRuntimeGCRaw()
		SleepCentiseconds(1)
		rawID = CreateThreadRaw(entry, stackPtr)
	}
	if ThreadDebug != nil {
		ThreadDebug(ThreadDebugEvent{
			Stage: "create_return",
			RawID: rawID,
		})
	}
	if rawID < 0 {
		threadStartRecords = threadStartRecords[:len(threadStartRecords)-1]
		return 0, false
	}

	record.tid = uint32(rawID)
	return record.tid, true
}

func hasFreeThreadSlot() bool {
	_, maxSlot, ok := ReadCurrentThreadInfo()
	if !ok || maxSlot <= 0 {
		return false
	}
	for slot := 1; slot <= maxSlot; slot++ {
		info, _, ok := ReadThreadInfo(slot)
		if !ok {
			continue
		}
		if info.Status == ThreadSlotFree {
			return true
		}
	}
	return false
}

// ThreadBootstrap is used by the thread entry trampoline.
// It is not intended for direct use by applications.
//
//go:noinline
func ThreadBootstrap(record *threadStartRecord) {
	if ThreadDebug != nil {
		var recordPtr uint32
		if record != nil {
			recordPtr = uint32(uintptr(unsafe.Pointer(record)))
		}
		ThreadDebug(ThreadDebugEvent{
			Stage:    "bootstrap",
			Record:   recordPtr,
			ThreadID: record.tid,
		})
	}
	if record == nil || record.fn == nil {
		return
	}
	record.fn()
}

func threadStackPointer(stack []byte) (uint32, bool) {
	if len(stack) < 16 {
		return 0, false
	}
	base := uintptr(unsafe.Pointer(&stack[0]))
	top := base + uintptr(len(stack))
	top &= ^uintptr(0xF)
	if top < base+4 {
		top = base + uintptr(len(stack))
		if top < base+4 {
			return 0, false
		}
	}
	return uint32(top - 4), true
}
