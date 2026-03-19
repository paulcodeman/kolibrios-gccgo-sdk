package ui

import (
	"image"

	"golang.org/x/image/math/fixed"
	"kos"
)

func (canvas *Canvas) DrawTextFont(x int, y int, color kos.Color, text string, font *ttfFont) {
	if canvas == nil || font == nil || text == "" {
		return
	}
	if FastNoTextDraw {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	clip := Rect{X: 0, Y: 0, Width: canvas.width, Height: canvas.height}
	if canvas.clip.set {
		if canvas.clip.rect.Empty() {
			return
		}
		clip = IntersectRect(clip, canvas.clip.rect)
		if clip.Empty() {
			return
		}
	}
	if x >= clip.X+clip.Width {
		return
	}
	lineHeight := font.metrics.height
	if lineHeight <= 0 {
		lineHeight = defaultFontHeight
	}
	if y+lineHeight <= clip.Y || y >= clip.Y+clip.Height {
		return
	}
	textWidth := textWidthWithFont(text, font, font.metrics.width)
	if textWidth > 0 && x+textWidth <= clip.X {
		return
	}
	canvas.drawTextFontClipped(x, y, kos.Color(rgb), alpha, text, font, clip)
}

func (canvas *Canvas) drawTextFontClipped(x int, y int, color kos.Color, alpha uint8, text string, font *ttfFont, clip Rect) {
	if canvas == nil || font == nil || text == "" || alpha == 0 {
		return
	}
	colorValue := uint32(color) & 0xFFFFFF
	baseline := y + font.metrics.ascent
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
			canvas.drawGlyphMask(dr, mask, maskp, colorValue, alpha, clip)
		}
		dotX += advance
		prev = r
	}
}

func (canvas *Canvas) drawGlyphMask(dr image.Rectangle, mask image.Image, maskp image.Point, colorValue uint32, alpha uint8, clip Rect) {
	if canvas == nil || mask == nil || alpha == 0 || dr.Empty() {
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
	if maxX > canvas.width {
		maxX = canvas.width
	}
	if maxY > canvas.height {
		maxY = canvas.height
	}
	if clip.Width > 0 && clip.Height > 0 {
		if minX < clip.X {
			minX = clip.X
		}
		if minY < clip.Y {
			minY = clip.Y
		}
		clipMaxX := clip.X + clip.Width
		clipMaxY := clip.Y + clip.Height
		if maxX > clipMaxX {
			maxX = clipMaxX
		}
		if maxY > clipMaxY {
			maxY = clipMaxY
		}
	}
	if minX >= maxX || minY >= maxY {
		return
	}
	if alpha >= 255 {
		alpha = 255
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
			dstIndex := 2 + y*canvas.width + minX
			for x := minX; x < maxX; x++ {
				if maskX >= maskRect.Min.X && maskX < maskRect.Max.X {
					a := m.Pix[rowStart+(maskX-maskRect.Min.X)]
					if a != 0 {
						effective := a
						if alpha < 255 {
							effective = uint8((int(a)*int(alpha) + 127) / 255)
						}
						if effective >= 255 {
							canvas.data[dstIndex] = 0xFF000000 | colorValue
						} else if effective != 0 {
							canvas.data[dstIndex] = canvas.blendPixel(canvas.data[dstIndex], colorValue, effective)
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
			dstIndex := 2 + y*canvas.width + minX
			for x := minX; x < maxX; x++ {
				if maskX >= maskBounds.Min.X && maskX < maskBounds.Max.X {
					_, _, _, ma := mask.At(maskX, maskY).RGBA()
					if ma != 0 {
						a := uint8(ma >> 8)
						effective := a
						if alpha < 255 {
							effective = uint8((int(a)*int(alpha) + 127) / 255)
						}
						if effective >= 255 {
							canvas.data[dstIndex] = 0xFF000000 | colorValue
						} else if effective != 0 {
							canvas.data[dstIndex] = canvas.blendPixel(canvas.data[dstIndex], colorValue, effective)
						}
					}
				}
				maskX++
				dstIndex++
			}
		}
	}
}
