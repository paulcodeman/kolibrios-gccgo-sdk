package core

func (buffer *Buffer) DrawShadow(rect Rect, shadow Shadow) {
	if buffer == nil || shadow.Alpha == 0 || rect.Empty() {
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
		buffer.FillRectAlpha(x, y, w, h, shadow.Color, uint8(alpha))
	}
}

func (buffer *Buffer) DrawShadowRounded(rect Rect, shadow Shadow, radii CornerRadii) {
	if buffer == nil || shadow.Alpha == 0 || rect.Empty() {
		return
	}
	if !radii.Active() {
		buffer.DrawShadow(rect, shadow)
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
		buffer.FillRoundedRectAlpha(x, y, w, h, layerRadii, shadow.Color, uint8(alpha))
	}
}
