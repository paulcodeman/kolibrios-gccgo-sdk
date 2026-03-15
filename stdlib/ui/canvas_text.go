package ui

import "kos"

func (canvas *Canvas) DrawText(x int, y int, color kos.Color, text string) {
	if canvas == nil || text == "" {
		return
	}
	if FastNoTextDraw {
		return
	}
	columns := textColumnCount(text)
	if columns == 0 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	if canvas.clip.set {
		clip := canvas.clip.rect
		if clip.Empty() {
			return
		}
		if y+defaultFontHeight <= clip.Y || y >= clip.Y+clip.Height {
			return
		}
		if x >= clip.X+clip.Width {
			return
		}
		if x+columns*defaultCharWidth <= clip.X {
			return
		}
		if x < clip.X {
			skip := (clip.X - x + defaultCharWidth - 1) / defaultCharWidth
			if skip >= columns {
				return
			}
			text = textSliceColumns(text, skip, columns)
			columns -= skip
			x += skip * defaultCharWidth
		}
		maxWidth := clip.X + clip.Width - x
		if maxWidth <= 0 {
			return
		}
		maxChars := maxWidth / defaultCharWidth
		if maxChars <= 0 {
			return
		}
		if columns > maxChars {
			text = textSliceColumns(text, 0, maxChars)
			columns = maxChars
		}
		if text == "" {
			return
		}
		partialY := clip.Y > y || clip.Y+clip.Height < y+defaultFontHeight
		if partialY {
			if canvas.alpha || alpha < 255 {
				canvas.drawTextAlphaClipped(x, y, kos.Color(rgb), text, alpha, clip)
			} else {
				canvas.drawTextAlphaClipped(x, y, kos.Color(rgb), text, 255, clip)
			}
			return
		}
	}
	if x < 0 || y < 0 || x >= canvas.width || y >= canvas.height {
		return
	}
	if y+defaultFontHeight > canvas.height {
		return
	}
	if canvas.alpha || alpha < 255 {
		canvas.drawTextAlpha(x, y, kos.Color(rgb), text, alpha)
		return
	}
	maxChars := (canvas.width - x) / defaultCharWidth
	if maxChars <= 0 {
		return
	}
	if columns > maxChars {
		text = textSliceColumns(text, 0, maxChars)
		if text == "" {
			return
		}
	}
	kos.DrawTextBuffer(x, y, kos.Color(rgb), text, canvas.headerPtr())
}

func (canvas *Canvas) drawTextAlpha(x int, y int, color kos.Color, text string, alpha uint8) {
	canvas.drawTextAlphaClipped(x, y, color, text, alpha, Rect{})
}

func (canvas *Canvas) drawTextAlphaClipped(x int, y int, color kos.Color, text string, alpha uint8, clip Rect) {
	if canvas == nil || text == "" || alpha == 0 {
		return
	}
	columns := textColumnCount(text)
	if columns == 0 {
		return
	}
	width := columns * defaultCharWidth
	height := defaultFontHeight
	if width <= 0 || height <= 0 {
		return
	}
	if x < 0 || y < 0 || x >= canvas.width || y >= canvas.height {
		return
	}
	if x+width > canvas.width {
		maxCols := (canvas.width - x) / defaultCharWidth
		if maxCols <= 0 {
			return
		}
		if columns > maxCols {
			text = textSliceColumns(text, 0, maxCols)
			columns = maxCols
		}
		width = columns * defaultCharWidth
		if width <= 0 || text == "" {
			return
		}
	}
	if y+height > canvas.height {
		height = canvas.height - y
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
	backup := getTextPixelSlice(width * height)
	if len(backup) < width*height {
		backup = make([]uint32, width*height)
	}
	rowStart := 2 + y*canvas.width + x
	for row := 0; row < height; row++ {
		srcIndex := rowStart + row*canvas.width
		dstIndex := row * width
		copy(backup[dstIndex:dstIndex+width], canvas.data[srcIndex:srcIndex+width])
	}
	sentinel := uint32(0x00FF00FF)
	if (uint32(color) & 0xFFFFFF) == (sentinel & 0xFFFFFF) {
		sentinel = 0x0000FF00
	}
	canvas.fillRectValue(x, y, width, height, sentinel)
	kos.DrawTextBuffer(x, y, color, text, canvas.headerPtr())
	colorValue := uint32(color) & 0xFFFFFF
	opaque := 0xFF000000 | colorValue
	for row := 0; row < height; row++ {
		index := rowStart + row*canvas.width
		backupIndex := row * width
		if clipSet && (row < visY0 || row >= visY1) {
			copy(canvas.data[index:index+width], backup[backupIndex:backupIndex+width])
			continue
		}
		for col := 0; col < width; col++ {
			if clipSet && (col < visX0 || col >= visX1) {
				canvas.data[index+col] = backup[backupIndex+col]
				continue
			}
			value := canvas.data[index+col]
			if (value & 0xFFFFFF) == (sentinel & 0xFFFFFF) {
				canvas.data[index+col] = backup[backupIndex+col]
				continue
			}
			if alpha >= 255 {
				canvas.data[index+col] = opaque
				continue
			}
			dst := backup[backupIndex+col]
			canvas.data[index+col] = canvas.blendPixel(dst, colorValue, alpha)
		}
	}
	releaseTextPixels(backup)
}
