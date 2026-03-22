package surface

import (
	"image"

	"golang.org/x/image/math/fixed"
	"kos"
)

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
