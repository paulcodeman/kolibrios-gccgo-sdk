package ui

import "kos"

func (canvas *Canvas) Clear(color kos.Color) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.Clear(color)
}

func (canvas *Canvas) ClearTransparent() {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.ClearTransparent()
}

func (canvas *Canvas) ClearRectTransparent(x int, y int, width int, height int) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.ClearRectTransparent(x, y, width, height)
}

func (canvas *Canvas) FillRect(x int, y int, width int, height int, color kos.Color) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRect(x, y, width, height, color)
}

func (canvas *Canvas) FillRoundedRect(x int, y int, width int, height int, radii CornerRadii, color kos.Color) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.FillRoundedRect(x, y, width, height, radii, color)
}

func (canvas *Canvas) StrokeRect(x int, y int, width int, height int, color kos.Color) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.StrokeRect(x, y, width, height, color)
}

func (canvas *Canvas) StrokeRectWidth(x int, y int, width int, height int, stroke int, color kos.Color) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.StrokeRectWidth(x, y, width, height, stroke, color)
}

func (canvas *Canvas) StrokeRoundedRectWidth(x int, y int, width int, height int, radii CornerRadii, stroke int, color kos.Color) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.StrokeRoundedRectWidth(x, y, width, height, radii, stroke, color)
}
