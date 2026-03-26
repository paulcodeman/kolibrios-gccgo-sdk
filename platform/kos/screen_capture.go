package kos

func GraphicsBitsPerPixel() int {
	return int(GetGraphicsBitsPerPixelRaw())
}

func GraphicsBytesPerLine() int {
	return int(GetGraphicsBytesPerLineRaw())
}

func CopyGraphicsBuffer(buffer []byte, screenOffset uint32) bool {
	if len(buffer) == 0 {
		return false
	}

	CopyGraphicsBufferRaw(bufferPointer(buffer), screenOffset, uint32(len(buffer)))
	return true
}

// ReadScreenArea copies a screen rectangle into buffer using packed BGR bytes.
// The caller owns buffer and must provide at least width*height*3 bytes.
func ReadScreenArea(buffer []byte, width int, height int, x int, y int) bool {
	if width <= 0 || height <= 0 || x < 0 || y < 0 {
		return false
	}

	needed := width * height * 3
	if needed <= 0 || len(buffer) < needed {
		return false
	}

	screenW, screenH := ScreenSize()
	if x+width > screenW || y+height > screenH {
		return false
	}

	ReadScreenAreaRaw(bufferPointer(buffer), width, height, x, y)
	return true
}
