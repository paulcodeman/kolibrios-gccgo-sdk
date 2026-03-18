package ui

import "kos"

const (
	defaultFocusRingWidth = 2
	defaultFocusRingGap   = 1
)

const defaultFocusRingColor kos.Color = kos.Color((160 << 24) | 0x1A73E8)

func elementShowsDefaultFocusRing(element *Element) bool {
	if element == nil || !element.focused || !element.isFocusable() {
		return false
	}
	return element.StyleFocus.IsZero()
}

func documentNodeShowsDefaultFocusRing(node *DocumentNode) bool {
	if node == nil || !node.focused || !documentNodeCanFocus(node) {
		return false
	}
	return node.StyleFocus.IsZero()
}

func focusRingBounds(rect Rect) Rect {
	if rect.Empty() {
		return rect
	}
	expand := defaultFocusRingGap + defaultFocusRingWidth
	return Rect{
		X:      rect.X - expand,
		Y:      rect.Y - expand,
		Width:  rect.Width + expand*2,
		Height: rect.Height + expand*2,
	}
}

func focusRingRadii(style Style) CornerRadii {
	expand := defaultFocusRingGap + defaultFocusRingWidth
	radii := resolveBorderRadius(style)
	if !radii.Active() {
		return CornerRadii{}
	}
	return CornerRadii{
		TopLeft:     radii.TopLeft + expand,
		TopRight:    radii.TopRight + expand,
		BottomRight: radii.BottomRight + expand,
		BottomLeft:  radii.BottomLeft + expand,
	}
}

func drawDefaultFocusRing(canvas *Canvas, rect Rect, style Style) {
	if canvas == nil || rect.Empty() {
		return
	}
	ring := focusRingBounds(rect)
	if ring.Empty() {
		return
	}
	canvas.StrokeRoundedRectWidth(ring.X, ring.Y, ring.Width, ring.Height, focusRingRadii(style), defaultFocusRingWidth, defaultFocusRingColor)
}
