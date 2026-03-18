package ui

func scrollRevealNearest(current int, viewportSize int, itemStart int, itemSize int) int {
	if viewportSize <= 0 || itemSize <= 0 {
		return current
	}
	visibleStart := current
	visibleEnd := current + viewportSize
	itemEnd := itemStart + itemSize
	if itemStart >= visibleStart && itemEnd <= visibleEnd {
		return current
	}
	if itemStart < visibleStart {
		return itemStart
	}
	if itemEnd > visibleEnd {
		return itemEnd - viewportSize
	}
	return current
}
