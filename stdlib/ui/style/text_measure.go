package ui

import "golang.org/x/image/math/fixed"

func textWidthWithFont(text string, font *ttfFont, charWidth int) int {
	if text == "" {
		return 0
	}
	if charWidth <= 0 {
		charWidth = defaultCharWidth
	}
	if !font.available() {
		return textColumnCount(text) * charWidth
	}
	width := font.measureStringFixed(text).Ceil()
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
	if !font.available() {
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
			width += font.kern(prev, r)
		}
		width += font.glyphAdvance(r)
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
	if !font.available() {
		return x / charWidth
	}
	target := fixed.I(x)
	var width fixed.Int26_6
	prev := rune(-1)
	col := 0
	for _, r := range text {
		kern := fixed.Int26_6(0)
		if prev >= 0 {
			kern = font.kern(prev, r)
		}
		start := width + kern
		if start >= target {
			break
		}
		advance := font.glyphAdvance(r)
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
