package ui

func resolveSpacingNormalized(value *Spacing) (Spacing, bool) {
	spacing, ok := resolveSpacing(value)
	if !ok || spacing == nil {
		return Spacing{}, false
	}
	valueSpacing := *spacing
	if valueSpacing.Left < 0 {
		valueSpacing.Left = 0
	}
	if valueSpacing.Right < 0 {
		valueSpacing.Right = 0
	}
	if valueSpacing.Top < 0 {
		valueSpacing.Top = 0
	}
	if valueSpacing.Bottom < 0 {
		valueSpacing.Bottom = 0
	}
	return valueSpacing, true
}

func spacingAny(value Spacing) bool {
	return value.Left != 0 || value.Right != 0 || value.Top != 0 || value.Bottom != 0
}

func borderWidthFor(style Style) int {
	if value, ok := resolveLength(style.borderWidth); ok && value > 0 {
		return value
	}
	return 0
}

func contentRectFor(rect Rect, style Style) Rect {
	if rect.Empty() {
		return rect
	}
	insets := boxInsets(style)
	insetLeft := insets.Left
	insetTop := insets.Top
	insetRight := insets.Right
	insetBottom := insets.Bottom
	width := rect.Width - insetLeft - insetRight
	height := rect.Height - insetTop - insetBottom
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	return Rect{
		X:      rect.X + insetLeft,
		Y:      rect.Y + insetTop,
		Width:  width,
		Height: height,
	}
}

func maxChildBottom(element *Element) int {
	if element == nil {
		return 0
	}
	maxBottom := element.layoutRect.Y
	for _, child := range element.Children {
		if child == nil {
			continue
		}
		if nodeHidden(child) {
			continue
		}
		style, bounds, margin, marginSet, ok := childLayoutMetrics(child)
		if !ok {
			continue
		}
		if effectivePosition(style) == PositionAbsolute {
			continue
		}
		bottom := bounds.Y + bounds.Height
		if marginSet {
			bottom += margin.Bottom
		}
		if bottom > maxBottom {
			maxBottom = bottom
		}
	}
	return maxBottom
}

func maxChildRight(element *Element) int {
	if element == nil {
		return 0
	}
	maxRight := element.layoutRect.X
	for _, child := range element.Children {
		if child == nil {
			continue
		}
		if nodeHidden(child) {
			continue
		}
		style, bounds, margin, marginSet, ok := childLayoutMetrics(child)
		if !ok {
			continue
		}
		if effectivePosition(style) == PositionAbsolute {
			continue
		}
		right := bounds.X + bounds.Width
		if marginSet {
			right += margin.Right
		}
		if right > maxRight {
			maxRight = right
		}
	}
	return maxRight
}

func childLayoutMetrics(node Node) (Style, Rect, Spacing, bool, bool) {
	switch child := node.(type) {
	case *Element:
		if child == nil {
			return Style{}, Rect{}, Spacing{}, false, false
		}
		return child.effectiveStyle(), child.layoutRect, child.layoutMargin, child.layoutMarginSet, true
	case *DocumentView:
		if child == nil {
			return Style{}, Rect{}, Spacing{}, false, false
		}
		margin, marginSet := resolvedMargin(child.effectiveStyle())
		return child.effectiveStyle(), child.layoutRect, margin, marginSet, true
	default:
		return Style{}, Rect{}, Spacing{}, false, false
	}
}

func overflowClipAxes(style Style) (bool, bool) {
	clipX := overflowModeFor(style, "x")
	clipY := overflowModeFor(style, "y")
	return clipX == OverflowHidden || clipX == OverflowScroll || clipX == OverflowAuto,
		clipY == OverflowHidden || clipY == OverflowScroll || clipY == OverflowAuto
}

func paintClipAxes(style Style) (bool, bool) {
	clipX, clipY := overflowClipAxes(style)
	if styleContainsPaint(style) {
		clipX = true
		clipY = true
	}
	return clipX, clipY
}
