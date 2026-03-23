package core

func (buffer *Buffer) DrawLine(x0 int, y0 int, x1 int, y1 int, color uint32) {
	if buffer == nil {
		return
	}
	colorValue, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	if x0 == x1 {
		if y1 < y0 {
			y0, y1 = y1, y0
		}
		if alpha >= 255 {
			buffer.FillRect(x0, y0, 1, y1-y0+1, colorValue)
			return
		}
		buffer.FillRectAlpha(x0, y0, 1, y1-y0+1, colorValue, alpha)
		return
	}
	if y0 == y1 {
		if x1 < x0 {
			x0, x1 = x1, x0
		}
		if alpha >= 255 {
			buffer.FillRect(x0, y0, x1-x0+1, 1, colorValue)
			return
		}
		buffer.FillRectAlpha(x0, y0, x1-x0+1, 1, colorValue, alpha)
		return
	}
	opaque := alpha >= 255
	clipSet := buffer.clip.set
	clip := buffer.clip.rect
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx - dy
	for {
		if x0 >= 0 && y0 >= 0 && x0 < buffer.width && y0 < buffer.height &&
			(!clipSet || clip.Contains(x0, y0)) {
			index := 2 + y0*buffer.width + x0
			if opaque {
				buffer.data[index] = 0xFF000000 | colorValue
			} else {
				buffer.data[index] = buffer.blendPixel(buffer.data[index], colorValue, alpha)
			}
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
