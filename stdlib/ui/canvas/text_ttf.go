package ui

import "kos"

func (canvas *Canvas) DrawTextFont(x int, y int, color kos.Color, text string, font *ttfFont) {
	if canvas == nil || font == nil || !font.available() || text == "" || FastNoTextDraw {
		return
	}
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.DrawTextFont(x, y, color, text, font.surface)
}
