package ui

import "kos"

func effectiveBoxSizing(style Style) BoxSizing {
	if value, ok := resolveBoxSizing(style.boxSizing); ok {
		return value
	}
	return BoxSizingBorderBox
}

func styleVisibility(style Style) VisibilityMode {
	if value, ok := resolveVisibility(style.visibility); ok {
		return value
	}
	return VisibilityVisible
}

func styleVisible(style Style) bool {
	return styleVisibility(style) != VisibilityHidden
}

func styleHiddenByVisibility(style Style, inherited bool) bool {
	if value, ok := resolveVisibility(style.visibility); ok {
		return value == VisibilityHidden
	}
	return inherited
}

func lineHeightForStyle(style Style, fallback int) int {
	if value, ok := resolveLineHeight(style.lineHeight); ok {
		return value
	}
	if fallback > 0 {
		return fallback
	}
	return defaultFontHeight
}

func textDecorationForStyle(style Style) TextDecoration {
	if value, ok := resolveTextDecoration(style.textDecoration); ok {
		return value
	}
	return TextDecorationNone
}

func whiteSpaceForStyle(style Style) WhiteSpaceMode {
	if value, ok := resolveWhiteSpace(style.whiteSpace); ok {
		return value
	}
	return WhiteSpaceNormal
}

func overflowWrapForStyle(style Style) OverflowWrapMode {
	if value, ok := resolveOverflowWrap(style.overflowWrap); ok {
		return value
	}
	return OverflowWrapNormal
}

func wordBreakForStyle(style Style) WordBreakMode {
	if value, ok := resolveWordBreak(style.wordBreak); ok {
		return value
	}
	return WordBreakNormal
}

func outlineWidthFor(style Style) int {
	if value, ok := resolveLength(style.outlineWidth); ok && value > 0 {
		return value
	}
	return 0
}

func outlineOffsetFor(style Style) int {
	if value, ok := resolveLength(style.outlineOffset); ok {
		return value
	}
	return 0
}

func outlineRadiusFor(style Style) int {
	if value, ok := resolveLength(style.outlineRadius); ok && value > 0 {
		return value
	}
	return 0
}

func outlineColorFor(style Style) (kos.Color, bool) {
	return resolveColor(style.outlineColor)
}

func borderWidthsFor(style Style) Spacing {
	base := borderWidthFor(style)
	widths := Spacing{
		Left:   base,
		Top:    base,
		Right:  base,
		Bottom: base,
	}
	if value, ok := resolveLength(style.borderLeftWidth); ok && value >= 0 {
		widths.Left = value
	}
	if value, ok := resolveLength(style.borderTopWidth); ok && value >= 0 {
		widths.Top = value
	}
	if value, ok := resolveLength(style.borderRightWidth); ok && value >= 0 {
		widths.Right = value
	}
	if value, ok := resolveLength(style.borderBottomWidth); ok && value >= 0 {
		widths.Bottom = value
	}
	if widths.Left < 0 {
		widths.Left = 0
	}
	if widths.Top < 0 {
		widths.Top = 0
	}
	if widths.Right < 0 {
		widths.Right = 0
	}
	if widths.Bottom < 0 {
		widths.Bottom = 0
	}
	return widths
}

func borderWidthsAny(widths Spacing) bool {
	return widths.Left > 0 || widths.Top > 0 || widths.Right > 0 || widths.Bottom > 0
}

func borderColorsFor(style Style) (kos.Color, kos.Color, kos.Color, kos.Color, bool) {
	base, baseSet := resolveColor(style.borderColor)
	top := base
	right := base
	bottom := base
	left := base
	set := baseSet
	if value, ok := resolveColor(style.borderTopColor); ok {
		top = value
		set = true
	}
	if value, ok := resolveColor(style.borderRightColor); ok {
		right = value
		set = true
	}
	if value, ok := resolveColor(style.borderBottomColor); ok {
		bottom = value
		set = true
	}
	if value, ok := resolveColor(style.borderLeftColor); ok {
		left = value
		set = true
	}
	return top, right, bottom, left, set
}

func uniformBorderStyle(style Style) (int, kos.Color, bool) {
	widths := borderWidthsFor(style)
	if widths.Left <= 0 && widths.Top <= 0 && widths.Right <= 0 && widths.Bottom <= 0 {
		return 0, 0, false
	}
	topColor, rightColor, bottomColor, leftColor, ok := borderColorsFor(style)
	if !ok {
		return 0, 0, false
	}
	if widths.Left == widths.Top && widths.Left == widths.Right && widths.Left == widths.Bottom &&
		topColor == rightColor && topColor == bottomColor && topColor == leftColor {
		return widths.Left, topColor, true
	}
	return 0, 0, false
}

func boxInsets(style Style) Spacing {
	padding, _ := resolveSpacingNormalized(style.padding)
	border := borderWidthsFor(style)
	return Spacing{
		Left:   padding.Left + border.Left,
		Top:    padding.Top + border.Top,
		Right:  padding.Right + border.Right,
		Bottom: padding.Bottom + border.Bottom,
	}
}

func explicitOuterWidth(style Style) (int, bool) {
	value, ok := resolveLength(style.width)
	if !ok {
		return 0, false
	}
	if effectiveBoxSizing(style) == BoxSizingContentBox {
		insets := boxInsets(style)
		value += insets.Left + insets.Right
	}
	if value < 0 {
		value = 0
	}
	return value, true
}

func explicitOuterHeight(style Style) (int, bool) {
	value, ok := resolveLength(style.height)
	if !ok {
		return 0, false
	}
	if effectiveBoxSizing(style) == BoxSizingContentBox {
		insets := boxInsets(style)
		value += insets.Top + insets.Bottom
	}
	if value < 0 {
		value = 0
	}
	return value, true
}

func outerMinWidth(style Style) (int, bool) {
	value, ok := resolveLength(style.minWidth)
	if !ok {
		return 0, false
	}
	if effectiveBoxSizing(style) == BoxSizingContentBox {
		insets := boxInsets(style)
		value += insets.Left + insets.Right
	}
	if value < 0 {
		value = 0
	}
	return value, true
}

func outerMaxWidth(style Style) (int, bool) {
	value, ok := resolveLength(style.maxWidth)
	if !ok {
		return 0, false
	}
	if effectiveBoxSizing(style) == BoxSizingContentBox {
		insets := boxInsets(style)
		value += insets.Left + insets.Right
	}
	if value < 0 {
		return 0, false
	}
	return value, true
}

func outerMinHeight(style Style) (int, bool) {
	value, ok := resolveLength(style.minHeight)
	if !ok {
		return 0, false
	}
	if effectiveBoxSizing(style) == BoxSizingContentBox {
		insets := boxInsets(style)
		value += insets.Top + insets.Bottom
	}
	if value < 0 {
		value = 0
	}
	return value, true
}

func outerMaxHeight(style Style) (int, bool) {
	value, ok := resolveLength(style.maxHeight)
	if !ok {
		return 0, false
	}
	if effectiveBoxSizing(style) == BoxSizingContentBox {
		insets := boxInsets(style)
		value += insets.Top + insets.Bottom
	}
	if value < 0 {
		return 0, false
	}
	return value, true
}

func clampWidthForStyle(style Style, width int) int {
	if width < 0 {
		width = 0
	}
	if minWidth, ok := outerMinWidth(style); ok && width < minWidth {
		width = minWidth
	}
	if maxWidth, ok := outerMaxWidth(style); ok && width > maxWidth {
		width = maxWidth
	}
	return width
}

func clampHeightForStyle(style Style, height int) int {
	if height < 0 {
		height = 0
	}
	if minHeight, ok := outerMinHeight(style); ok && height < minHeight {
		height = minHeight
	}
	if maxHeight, ok := outerMaxHeight(style); ok && height > maxHeight {
		height = maxHeight
	}
	return height
}
