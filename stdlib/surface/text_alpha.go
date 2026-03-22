package surface

import "kos"

func (buffer *Buffer) drawTextAlpha(x int, y int, color kos.Color, text string, alpha uint8) {
	buffer.drawTextAlphaClipped(x, y, color, text, alpha, Rect{})
}

func (buffer *Buffer) drawTextAlphaClipped(x int, y int, color kos.Color, text string, alpha uint8, clip Rect) {
	if buffer == nil || text == "" || alpha == 0 {
		return
	}
	columns := textColumnCount(text)
	if columns == 0 {
		return
	}
	width := columns * DefaultCharWidth
	height := DefaultFontHeight
	if width <= 0 || height <= 0 {
		return
	}
	if x < 0 || y < 0 || x >= buffer.width || y >= buffer.height {
		return
	}
	if x+width > buffer.width {
		maxCols := (buffer.width - x) / DefaultCharWidth
		if maxCols <= 0 {
			return
		}
		if columns > maxCols {
			text = textSliceColumns(text, 0, maxCols)
			columns = maxCols
		}
		width = columns * DefaultCharWidth
		if width <= 0 || text == "" {
			return
		}
	}
	if y+height > buffer.height {
		height = buffer.height - y
		if height <= 0 {
			return
		}
	}
	clipSet := !clip.Empty()
	visX0 := 0
	visY0 := 0
	visX1 := width
	visY1 := height
	if clipSet {
		textRect := Rect{X: x, Y: y, Width: width, Height: height}
		visible := IntersectRect(textRect, clip)
		if visible.Empty() {
			return
		}
		visX0 = visible.X - x
		visY0 = visible.Y - y
		visX1 = visX0 + visible.Width
		visY1 = visY0 + visible.Height
	}
	backup := make([]uint32, width*height)
	rowStart := 2 + y*buffer.width + x
	for row := 0; row < height; row++ {
		srcIndex := rowStart + row*buffer.width
		dstIndex := row * width
		copy(backup[dstIndex:dstIndex+width], buffer.data[srcIndex:srcIndex+width])
	}
	sentinel := uint32(0x00FF00FF)
	if (uint32(color) & 0xFFFFFF) == (sentinel & 0xFFFFFF) {
		sentinel = 0x0000FF00
	}
	buffer.fillRectValue(x, y, width, height, sentinel)
	kos.DrawTextBuffer(x, y, color, text, buffer.headerPtr())
	colorValue := uint32(color) & 0xFFFFFF
	opaque := 0xFF000000 | colorValue
	for row := 0; row < height; row++ {
		index := rowStart + row*buffer.width
		backupIndex := row * width
		if clipSet && (row < visY0 || row >= visY1) {
			copy(buffer.data[index:index+width], backup[backupIndex:backupIndex+width])
			continue
		}
		for col := 0; col < width; col++ {
			if clipSet && (col < visX0 || col >= visX1) {
				buffer.data[index+col] = backup[backupIndex+col]
				continue
			}
			value := buffer.data[index+col]
			if (value & 0xFFFFFF) == (sentinel & 0xFFFFFF) {
				buffer.data[index+col] = backup[backupIndex+col]
				continue
			}
			if alpha >= 255 {
				buffer.data[index+col] = opaque
				continue
			}
			buffer.data[index+col] = buffer.blendPixel(backup[backupIndex+col], colorValue, alpha)
		}
	}
}
