package ui

func (canvas *Canvas) DrawShadow(rect Rect, shadow Shadow) {
	if canvas == nil || shadow.Alpha == 0 {
		return
	}
	if rect.Empty() {
		return
	}
	blur := shadow.Blur
	if blur < 0 {
		blur = 0
	}
	baseAlpha := int(shadow.Alpha)
	layers := blur + 1
	for i := blur; i >= 0; i-- {
		alpha := baseAlpha
		if blur > 0 {
			alpha = baseAlpha * (blur - i + 1) / layers
		}
		if alpha <= 0 {
			continue
		}
		x := rect.X + shadow.OffsetX - i
		y := rect.Y + shadow.OffsetY - i
		w := rect.Width + i*2
		h := rect.Height + i*2
		if w <= 0 || h <= 0 {
			continue
		}
		canvas.FillRectAlpha(x, y, w, h, shadow.Color, uint8(alpha))
	}
}

func (canvas *Canvas) DrawShadowRounded(rect Rect, shadow Shadow, radii CornerRadii) {
	if canvas == nil || shadow.Alpha == 0 {
		return
	}
	if rect.Empty() {
		return
	}
	if !radii.Active() {
		canvas.DrawShadow(rect, shadow)
		return
	}
	blur := shadow.Blur
	if blur < 0 {
		blur = 0
	}
	baseAlpha := int(shadow.Alpha)
	layers := blur + 1
	for i := blur; i >= 0; i-- {
		alpha := baseAlpha
		if blur > 0 {
			alpha = baseAlpha * (blur - i + 1) / layers
		}
		if alpha <= 0 {
			continue
		}
		x := rect.X + shadow.OffsetX - i
		y := rect.Y + shadow.OffsetY - i
		w := rect.Width + i*2
		h := rect.Height + i*2
		if w <= 0 || h <= 0 {
			continue
		}
		layerRadii := CornerRadii{
			TopLeft:     radii.TopLeft + i,
			TopRight:    radii.TopRight + i,
			BottomRight: radii.BottomRight + i,
			BottomLeft:  radii.BottomLeft + i,
		}
		canvas.FillRoundedRectAlpha(x, y, w, h, layerRadii, shadow.Color, uint8(alpha))
	}
}
