package main

var decimalDigits = [...]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

func formatInt(value int) string {
	if value < 0 {
		return "-" + formatUint32(uint32(-value))
	}

	return formatUint32(uint32(value))
}

func formatSignedInt(value int) string {
	if value > 0 {
		return "+" + formatInt(value)
	}
	if value < 0 {
		return "-" + formatUint32(uint32(-value))
	}

	return "0"
}

func formatUint32(value uint32) string {
	if value < 10 {
		return decimalDigits[value]
	}

	return formatUint32(value/10) + decimalDigits[value%10]
}
