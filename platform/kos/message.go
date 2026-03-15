package kos

func SendActiveWindowMessage(event EventType, param uint32) MessageStatus {
	return MessageStatus(SendMessage(int(event), param))
}

func SendActiveWindowKey(key uint32) MessageStatus {
	return SendActiveWindowMessage(EventKey, key)
}

func SendActiveWindowButton(id ButtonID) MessageStatus {
	return SendActiveWindowMessage(EventButton, uint32(id))
}
