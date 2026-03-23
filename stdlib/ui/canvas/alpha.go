package ui

import "kos"

func (canvas *Canvas) FillRectAlpha(x int, y int, width int, height int, color kos.Color, alpha uint8) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRectAlpha(x, y, width, height, color, alpha)
}

func (canvas *Canvas) FillRoundedRectAlpha(x int, y int, width int, height int, radii CornerRadii, color kos.Color, alpha uint8) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRoundedRectAlpha(x, y, width, height, radii, color, alpha)
}
