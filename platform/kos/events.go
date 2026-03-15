package kos

func WaitEvent() EventType {
	PollRuntimeGCRaw()
	if !runtimeThreaded() {
		PollRuntimeWorldStopRaw()
		return EventType(Event())
	}
	for {
		PollRuntimeWorldStopRaw()
		event := EventType(WaitEventTimeout(1))
		if event != EventNone {
			return event
		}
	}
}

func PollEvent() EventType {
	PollRuntimeGCRaw()
	PollRuntimeWorldStopRaw()
	return EventType(CheckEvent())
}

func WaitEventFor(timeout uint32) EventType {
	PollRuntimeGCRaw()
	if timeout == 0 {
		return WaitEvent()
	}
	if !runtimeThreaded() {
		PollRuntimeWorldStopRaw()
		return EventType(WaitEventTimeout(timeout))
	}
	remaining := timeout
	for {
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
