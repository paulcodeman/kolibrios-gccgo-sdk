package ui

import (
	xfont "golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func measureStringFixed(face xfont.Face, text string) fixed.Int26_6 {
	if face == nil || text == "" {
		return 0
	}
	var width fixed.Int26_6
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			width += face.Kern(prev, r)
		}
		advance, _ := face.GlyphAdvance(r)
		width += advance
		prev = r
	}
	if width < 0 {
		return -width
	}
	return width
}

func textWidthWithFont(text string, font *ttfFont, charWidth int) int {
	if text == "" {
		return 0
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if font == nil || font.face == nil {
		return textColumnCount(text) * charWidth
	}
	width := measureStringFixed(font.face, text).Ceil()
	if width < 0 {
		width = -width
	}
	return width
}

func textWidthForColumns(text string, cols int, font *ttfFont, charWidth int) int {
	if text == "" || cols <= 0 {
		return 0
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if font == nil || font.face == nil {
		return cols * charWidth
	}
	var width fixed.Int26_6
	prev := rune(-1)
	count := 0
	for _, r := range text {
		if count >= cols {
			break
		}
		if prev >= 0 {
			width += font.face.Kern(prev, r)
		}
		advance, _ := font.face.GlyphAdvance(r)
		width += advance
		prev = r
		count++
	}
	pixels := width.Ceil()
	if pixels < 0 {
		pixels = -pixels
	}
	return pixels
}

func textColumnForX(text string, x int, font *ttfFont, charWidth int) int {
	if x <= 0 || text == "" {
		return 0
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if font == nil || font.face == nil {
		return x / charWidth
	}
	target := fixed.I(x)
	var width fixed.Int26_6
	prev := rune(-1)
	col := 0
	for _, r := range text {
		kern := fixed.Int26_6(0)
		if prev >= 0 {
			kern = font.face.Kern(prev, r)
		}
		start := width + kern
		if start >= target {
			break
		}
		advance, _ := font.face.GlyphAdvance(r)
		width = start + advance
		prev = r
		col++
	}
	return col
}

func maxLineWidth(lines []textLine, font *ttfFont, charWidth int) int {
	ensureTextLineMetrics(lines, font, charWidth)
	maxWidth := 0
	for _, line := range lines {
		if line.width > maxWidth {
			maxWidth = line.width
		}
	}
	return maxWidth
}
