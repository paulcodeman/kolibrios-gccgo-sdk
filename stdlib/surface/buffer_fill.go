package surface

import "kos"

func (buffer *Buffer) Clear(color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.Clear(uint32(color))
}

func (buffer *Buffer) FillRect(x int, y int, width int, height int, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRect(x, y, width, height, uint32(color))
}

func (buffer *Buffer) FillRectAlpha(x int, y int, width int, height int, color kos.Color, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRectAlpha(x, y, width, height, uint32(color), alpha)
}

func (buffer *Buffer) ClearTransparent() {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.ClearTransparent()
}

func (buffer *Buffer) ClearRectTransparent(x int, y int, width int, height int) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.ClearRectTransparent(x, y, width, height)
}

func (buffer *Buffer) FillRoundedRect(x int, y int, width int, height int, radii CornerRadii, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRoundedRect(x, y, width, height, radii, uint32(color))
}

func (buffer *Buffer) FillRoundedRectAlpha(x int, y int, width int, height int, radii CornerRadii, color kos.Color, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.FillRoundedRectAlpha(x, y, width, height, radii, uint32(color), alpha)
}

func (buffer *Buffer) StrokeRect(x int, y int, width int, height int, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.StrokeRect(x, y, width, height, uint32(color))
}

func (buffer *Buffer) StrokeRectWidth(x int, y int, width int, height int, stroke int, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.StrokeRectWidth(x, y, width, height, stroke, uint32(color))
}

func (buffer *Buffer) StrokeRoundedRectWidth(x int, y int, width int, height int, radii CornerRadii, stroke int, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.StrokeRoundedRectWidth(x, y, width, height, radii, stroke, uint32(color))
}

func (buffer *Buffer) DrawShadow(rect Rect, shadow Shadow) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.DrawShadow(rect, rawShadow(shadow))
}

func (buffer *Buffer) DrawShadowRounded(rect Rect, shadow Shadow, radii CornerRadii) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.DrawShadowRounded(rect, rawShadow(shadow), radii)
}
