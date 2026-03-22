package core

import (
	"image"

	"golang.org/x/image/math/fixed"
	"kos"
)

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

func (buffer *Buffer) DrawTextFont(x int, y int, color kos.Color, text string, font *Font) {
	if buffer == nil || font == nil || text == "" {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	clip := Rect{Width: buffer.width, Height: buffer.height}
	if buffer.clip.set {
		if buffer.clip.rect.Empty() {
			return
		}
		clip = IntersectRect(clip, buffer.clip.rect)
		if clip.Empty() {
			return
		}
	}
	if x >= clip.X+clip.Width {
		return
	}
	lineHeight := font.metrics.Height
	if lineHeight <= 0 {
		lineHeight = DefaultFontHeight
	}
	if y+lineHeight <= clip.Y || y >= clip.Y+clip.Height {
		return
	}
	textWidth := font.MeasureString(text)
	if textWidth > 0 && x+textWidth <= clip.X {
		return
	}
	buffer.drawTextFontClipped(x, y, kos.Color(rgb), alpha, text, font, clip)
}

func (buffer *Buffer) drawTextFontClipped(x int, y int, color kos.Color, alpha uint8, text string, font *Font, clip Rect) {
	if buffer == nil || font == nil || text == "" || alpha == 0 {
		return
	}
	colorValue := uint32(color) & 0xFFFFFF
	baseline := y + font.metrics.Ascent
	dotY := fixed.I(baseline)
	dotX := fixed.I(x)
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			dotX += font.kern(prev, r)
		}
		dot := fixed.Point26_6{X: dotX, Y: dotY}
		dr, mask, maskp, advance, ok := font.glyph(dot, r)
		if ok && mask != nil && !dr.Empty() {
			buffer.drawGlyphMask(dr, mask, maskp, colorValue, alpha, clip)
		}
		dotX += advance
		prev = r
	}
}

func (buffer *Buffer) drawGlyphMask(dr image.Rectangle, mask image.Image, maskp image.Point, colorValue uint32, alpha uint8, clip Rect) {
	if buffer == nil || mask == nil || alpha == 0 || dr.Empty() {
		return
	}
	minX := dr.Min.X
	minY := dr.Min.Y
	maxX := dr.Max.X
	maxY := dr.Max.Y
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > buffer.width {
		maxX = buffer.width
	}
	if maxY > buffer.height {
		maxY = buffer.height
	}
	if clip.Width > 0 && clip.Height > 0 {
		if minX < clip.X {
			minX = clip.X
		}
		if minY < clip.Y {
			minY = clip.Y
		}
		if clipMaxX := clip.X + clip.Width; maxX > clipMaxX {
			maxX = clipMaxX
		}
		if clipMaxY := clip.Y + clip.Height; maxY > clipMaxY {
			maxY = clipMaxY
		}
	}
	if minX >= maxX || minY >= maxY {
		return
	}
	switch m := mask.(type) {
	case *image.Alpha:
		maskRect := m.Rect
		for y := minY; y < maxY; y++ {
			maskY := maskp.Y + (y - dr.Min.Y)
			if maskY < maskRect.Min.Y || maskY >= maskRect.Max.Y {
				continue
			}
			rowStart := (maskY - maskRect.Min.Y) * m.Stride
			maskX := maskp.X + (minX - dr.Min.X)
			dstIndex := 2 + y*buffer.width + minX
			for x := minX; x < maxX; x++ {
				if maskX >= maskRect.Min.X && maskX < maskRect.Max.X {
					a := m.Pix[rowStart+(maskX-maskRect.Min.X)]
					if a != 0 {
						effective := a
						if alpha < 255 {
							effective = uint8((int(a)*int(alpha) + 127) / 255)
						}
						if effective >= 255 {
							buffer.data[dstIndex] = 0xFF000000 | colorValue
						} else if effective != 0 {
							buffer.data[dstIndex] = buffer.blendPixel(buffer.data[dstIndex], colorValue, effective)
						}
					}
				}
				maskX++
				dstIndex++
			}
		}
	default:
		maskBounds := mask.Bounds()
		for y := minY; y < maxY; y++ {
			maskY := maskp.Y + (y - dr.Min.Y)
			if maskY < maskBounds.Min.Y || maskY >= maskBounds.Max.Y {
				continue
			}
			maskX := maskp.X + (minX - dr.Min.X)
			dstIndex := 2 + y*buffer.width + minX
			for x := minX; x < maxX; x++ {
				if maskX >= maskBounds.Min.X && maskX < maskBounds.Max.X {
					_, _, _, ma := mask.At(maskX, maskY).RGBA()
					if ma != 0 {
						effective := uint8(ma >> 8)
						if alpha < 255 {
							effective = uint8((int(effective)*int(alpha) + 127) / 255)
						}
						if effective >= 255 {
							buffer.data[dstIndex] = 0xFF000000 | colorValue
						} else if effective != 0 {
							buffer.data[dstIndex] = buffer.blendPixel(buffer.data[dstIndex], colorValue, effective)
						}
					}
				}
				maskX++
				dstIndex++
			}
		}
	}
}
