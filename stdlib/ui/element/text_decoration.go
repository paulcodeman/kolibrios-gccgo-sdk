package ui

import "kos"

func underlineYForLine(y int, metrics fontMetrics) int {
	height := metrics.height
	if height <= 1 {
		return y
	}
	ascent := metrics.ascent
	if ascent <= 0 || ascent > height {
		ascent = height * 3 / 4
	}
	offset := height / 12
	if offset < 1 {
		offset = 1
	}
	underlineY := y + ascent + offset
	maxY := y + height - 1
	if underlineY > maxY {
		underlineY = maxY
	}
	if underlineY < y {
		underlineY = y
	}
	return underlineY
}

func decorationMetrics(font *ttfFont, charWidth int) fontMetrics {
	if font != nil {
		return font.metrics
	}
	metrics := defaultFontMetrics()
	if charWidth > 0 {
		metrics.width = charWidth
	}
	return metrics
}

func drawTextDecorations(canvas *Canvas, x int, y int, line string, style Style, font *ttfFont, charWidth int, color kos.Color) {
	if canvas == nil {
		return
	}
	if textDecorationForStyle(style) != TextDecorationUnderline {
		return
	}
	metrics := decorationMetrics(font, charWidth)
	width := textWidthWithFont(line, font, metrics.width)
	if width <= 0 {
		return
	}
	canvas.FillRect(x, underlineYForLine(y, metrics), width, 1, color)
}
