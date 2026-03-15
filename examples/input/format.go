package main

var messageDecimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
var messageHexDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "A", "B", "C", "D", "E", "F"}

func messageFormatInt(value int) string {
	if value < 0 {
		return "-" + messageFormatUint32(uint32(-value))
	}

	return messageFormatUint32(uint32(value))
}

func messageFormatUint32(value uint32) string {
	if value < 10 {
		return messageDecimalDigits[value]
	}

	return messageFormatUint32(value/10) + messageDecimalDigits[value%10]
}

func messageFormatHex32(value uint32) string {
	return "0x" +
		messageHexDigits[(value>>28)&0x0F] +
		messageHexDigits[(value>>24)&0x0F] +
		messageHexDigits[(value>>20)&0x0F] +
		messageHexDigits[(value>>16)&0x0F] +
		messageHexDigits[(value>>12)&0x0F] +
		messageHexDigits[(value>>8)&0x0F] +
		messageHexDigits[(value>>4)&0x0F] +
		messageHexDigits[value&0x0F]
}
