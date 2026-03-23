package ui

import "kos"

func (canvas *Canvas) DrawText(x int, y int, color kos.Color, text string) {
	if canvas == nil || text == "" || FastNoTextDraw {
		return
	}
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.DrawText(x, y, color, text)
}
