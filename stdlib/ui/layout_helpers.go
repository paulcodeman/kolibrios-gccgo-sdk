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
	if value, ok := resolveLength(style.BorderWidth); ok && value > 0 {
		return value
	}
	return 0
}

func contentRectFor(rect Rect, style Style) Rect {
	if rect.Empty() {
		return rect
	}
	padding, _ := resolveSpacingNormalized(style.Padding)
	border := borderWidthFor(style)
	insetLeft := padding.Left + border
	insetTop := padding.Top + border
	insetRight := padding.Right + border
	insetBottom := padding.Bottom + border
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
		childElement, ok := child.(*Element)
		if !ok || childElement == nil {
			continue
		}
		if childElement.layoutHidden {
			continue
		}
		if childElement.layoutPosition == PositionAbsolute {
			continue
		}
		bounds := childElement.layoutRect
		bottom := bounds.Y + bounds.Height
		if childElement.layoutMarginSet {
			bottom += childElement.layoutMargin.Bottom
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
		childElement, ok := child.(*Element)
		if !ok || childElement == nil {
			continue
		}
		if childElement.layoutHidden {
			continue
		}
		if childElement.layoutPosition == PositionAbsolute {
			continue
		}
		bounds := childElement.layoutRect
		right := bounds.X + bounds.Width
		if childElement.layoutMarginSet {
			right += childElement.layoutMargin.Right
		}
		if right > maxRight {
			maxRight = right
		}
	}
	return maxRight
}

func overflowClipAxes(style Style) (bool, bool) {
	clipX := overflowModeFor(style, "x")
	clipY := overflowModeFor(style, "y")
	return clipX == OverflowHidden || clipX == OverflowScroll || clipX == OverflowAuto,
		clipY == OverflowHidden || clipY == OverflowScroll || clipY == OverflowAuto
}
