package kos

func WaitEvent() EventType {
	PollRuntimeGCRaw()
	return EventType(Event())
}

func PollEvent() EventType {
	PollRuntimeGCRaw()
	return EventType(CheckEvent())
}

func WaitEventFor(timeout uint32) EventType {
	PollRuntimeGCRaw()
	return EventType(WaitEventTimeout(timeout))
}

func SwapEventMask(mask EventMask) EventMask {
	return EventMask(SetEventMask(uint32(mask)))
}

func CurrentButtonID() ButtonID {
	return ButtonID(GetButtonID())
}
