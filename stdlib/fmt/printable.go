package fmt

func quotedBsearch16(values []uint16, target uint16) int {
	low, high := 0, len(values)
	for low < high {
		middle := low + (high-low)>>1
		if values[middle] < target {
			low = middle + 1
		} else {
			high = middle
		}
	}
	return low
}

func quotedBsearch32(values []uint32, target uint32) int {
	low, high := 0, len(values)
	for low < high {
		middle := low + (high-low)>>1
		if values[middle] < target {
			low = middle + 1
		} else {
			high = middle
		}
	}
	return low
}

func quotedIsPrint(value rune) bool {
	if value <= 0xFF {
		if 0x20 <= value && value <= 0x7E {
			return true
		}
		if 0xA1 <= value && value <= 0xFF {
			return value != 0xAD
		}
		return false
	}

	if 0 <= value && value < 1<<16 {
		r := uint16(value)
		index := quotedBsearch16(isPrint16, r)
		if index >= len(isPrint16) || r < isPrint16[index&^1] || isPrint16[index|1] < r {
			return false
		}
		notPrintIndex := quotedBsearch16(isNotPrint16, r)
		return notPrintIndex >= len(isNotPrint16) || isNotPrint16[notPrintIndex] != r
	}

	r := uint32(value)
	index := quotedBsearch32(isPrint32, r)
	if index >= len(isPrint32) || r < isPrint32[index&^1] || isPrint32[index|1] < r {
		return false
	}
	if value >= 0x20000 {
		return true
	}

	value -= 0x10000
	notPrintIndex := quotedBsearch16(isNotPrint32, uint16(value))
	return notPrintIndex >= len(isNotPrint32) || isNotPrint32[notPrintIndex] != uint16(value)
}
