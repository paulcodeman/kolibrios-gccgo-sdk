package ui

import "kos"

func underlineYForLine(y int, lineHeight int) int {
	if lineHeight <= 1 {
		return y
	}
	offset := lineHeight / 8
	if offset < 1 {
		offset = 1
	}
	return y + lineHeight - offset
}

func drawTextDecorations(canvas *Canvas, x int, y int, line string, style Style, font *ttfFont, charWidth int, lineHeight int, color kos.Color) {
	if canvas == nil {
		return
	}
	if textDecorationForStyle(style) != TextDecorationUnderline {
		return
	}
	width := textWidthWithFont(line, font, charWidth)
	if width <= 0 {
		return
	}
	canvas.FillRect(x, underlineYForLine(y, lineHeight), width, 1, color)
}

func drawTextDecorationsRaw(x int, y int, line string, style Style, font *ttfFont, charWidth int, lineHeight int, color kos.Color) {
	if textDecorationForStyle(style) != TextDecorationUnderline {
		return
	}
	width := textWidthWithFont(line, font, charWidth)
	if width <= 0 {
		return
	}
	kos.DrawBar(x, underlineYForLine(y, lineHeight), width, 1, uint32(color))
}
