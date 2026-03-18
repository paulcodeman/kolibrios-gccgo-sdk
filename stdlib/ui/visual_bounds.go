package ui

func visualBoundsForStyle(rect Rect, style Style, includeTextShadow bool) Rect {
	if rect.Empty() {
		return rect
	}
	visual := rect
	if shadow, ok := resolveShadow(style.shadow); ok {
		blur := shadow.Blur
		if blur < 0 {
			blur = 0
		}
		left := visual.X
		top := visual.Y
		right := visual.X + visual.Width
		bottom := visual.Y + visual.Height
		shadowLeft := rect.X + shadow.OffsetX - blur
		shadowTop := rect.Y + shadow.OffsetY - blur
		shadowRight := rect.X + shadow.OffsetX + rect.Width + blur
		shadowBottom := rect.Y + shadow.OffsetY + rect.Height + blur
		if shadowLeft < left {
			left = shadowLeft
		}
		if shadowTop < top {
			top = shadowTop
		}
		if shadowRight > right {
			right = shadowRight
		}
		if shadowBottom > bottom {
			bottom = shadowBottom
		}
		visual = Rect{X: left, Y: top, Width: right - left, Height: bottom - top}
	}
	if includeTextShadow {
		if shadow, ok := resolveTextShadow(style.textShadow); ok {
			shadowRect := Rect{
				X:      rect.X + shadow.OffsetX,
				Y:      rect.Y + shadow.OffsetY,
				Width:  rect.Width,
				Height: rect.Height,
			}
			visual = UnionRect(visual, shadowRect)
		}
	}
	if width := outlineWidthFor(style); width > 0 {
		offset := outlineOffsetFor(style)
		expand := width + offset
		outline := Rect{
			X:      rect.X - expand,
			Y:      rect.Y - expand,
			Width:  rect.Width + expand*2,
			Height: rect.Height + expand*2,
		}
		visual = UnionRect(visual, outline)
	}
	return visual
}
