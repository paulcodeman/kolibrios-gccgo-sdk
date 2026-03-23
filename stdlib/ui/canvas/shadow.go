package ui

func (canvas *Canvas) DrawShadow(rect Rect, shadow Shadow) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.DrawShadow(rect, shadow)
}

func (canvas *Canvas) DrawShadowRounded(rect Rect, shadow Shadow, radii CornerRadii) {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return
	}
	raw.DrawShadowRounded(rect, shadow, radii)
}
