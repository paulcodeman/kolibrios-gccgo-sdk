package kos

func WaitEvent() EventType {
	threaded := runtimeThreaded()
	for {
		PollRuntimeGCRaw()
		PollRuntimeWorldStopRaw()
		event := EventType(WaitEventTimeout(1))
		if event != EventNone {
			return event
		}
		if !threaded {
			Gosched()
		}
	}
}

func PollEvent() EventType {
	PollRuntimeGCRaw()
	PollRuntimeWorldStopRaw()
	return EventType(CheckEvent())
}

func WaitEventFor(timeout uint32) EventType {
	if timeout == 0 {
		return WaitEvent()
	}
	threaded := runtimeThreaded()
	remaining := timeout
	for {
		PollRuntimeGCRaw()
		PollRuntimeWorldStopRaw()
		step := remaining
		if step > 1 {
			step = 1
		}
		event := EventType(WaitEventTimeout(step))
		if event != EventNone {
			return event
		}
		if remaining <= step {
			return event
		}
		remaining -= step
		if !threaded {
			Gosched()
		}
	}
}

func SwapEventMask(mask EventMask) EventMask {
	return EventMask(SetEventMask(uint32(mask)))
}

func CurrentButtonID() ButtonID {
	return ButtonID(GetButtonID())
}

func runtimeThreaded() bool {
	return GetRuntimeMCountRaw() > 1
}
