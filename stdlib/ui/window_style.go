package ui

// UpdateStyle mutates the window style and applies layout-related changes.
func (window *Window) UpdateStyle(update func(style *Style)) bool {
	if window == nil || update == nil {
		return false
	}
	oldStyle := window.Style
	oldOverflow := window.overflowModeY()
	oldVisual := visualKeyFor(oldStyle)
	update(&window.Style)
	changed := window.applyStyleBounds()
	if window.styleLayoutChanged(oldStyle, window.Style) {
		window.layoutDirty = true
		window.renderListValid = false
		window.hoverDirty = true
		window.lastMouseValid = false
		changed = true
	}
	if !styleVisualKeyEqual(oldVisual, visualKeyFor(window.Style)) {
		if window.client.Width > 0 && window.client.Height > 0 {
			window.Invalidate(Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height})
		}
		changed = true
	}
	newOverflow := window.overflowModeY()
	if oldOverflow != newOverflow {
		window.updateScrollMetrics()
		window.scrollRedraw = true
		if window.client.Width > 0 && window.client.Height > 0 {
			window.Invalidate(Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height})
		}
		return true
	}
	return changed
}

func (window *Window) contentRect() Rect {
	if window == nil {
		return Rect{}
	}
	base := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	return contentRectFor(base, window.Style)
}

func (window *Window) applyStyleBounds() bool {
	if window == nil {
		return false
	}
	x := window.X
	y := window.Y
	width := window.Width
	height := window.Height
	if value, ok := resolveLength(window.Style.Left); ok {
		x = value
	}
	if value, ok := resolveLength(window.Style.Top); ok {
		y = value
	}
	if value, ok := resolveLength(window.Style.Width); ok {
		width = value
	}
	if value, ok := resolveLength(window.Style.Height); ok {
		height = value
	}
	changed := window.setPosition(x, y)
	if window.setSize(width, height) {
		changed = true
	}
	return changed
}

func (window *Window) styleLayoutChanged(oldStyle Style, newStyle Style) bool {
	if borderWidthFor(oldStyle) != borderWidthFor(newStyle) {
		return true
	}
	oldPadding, _ := resolveSpacingNormalized(oldStyle.Padding)
	newPadding, _ := resolveSpacingNormalized(newStyle.Padding)
	return oldPadding != newPadding
}

func (window *Window) overflowModeY() OverflowMode {
	if window == nil {
		return OverflowVisible
	}
	if window.Style.Overflow == nil && window.Style.OverflowY == nil {
		return OverflowAuto
	}
	return overflowModeFor(window.Style, "y")
}

func (window *Window) scrollEnabled() bool {
	if window == nil || !WindowScrollYEnabled {
		return false
	}
	mode := window.overflowModeY()
	return mode == OverflowScroll || mode == OverflowAuto
}
