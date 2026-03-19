package ui

// UpdateStyle mutates the window style and applies layout-related changes.
func (window *Window) UpdateStyle(update func(style *Style)) bool {
	if window == nil || update == nil {
		return false
	}
	oldStyle := window.Style
	oldVisual := visualKeyFor(oldStyle)
	oldInsets := boxInsets(oldStyle)
	oldClipX, oldClipY := window.styleClipAxes(oldStyle)
	oldOverflow := window.overflowModeYForStyle(oldStyle)
	update(&window.Style)
	newVisual := visualKeyFor(window.Style)
	newInsets := boxInsets(window.Style)
	newClipX, newClipY := window.styleClipAxes(window.Style)
	newOverflow := window.overflowModeY()
	if oldInsets != newInsets {
		window.invalidateWindowContentPropertyState()
	} else {
		if oldClipX != newClipX || oldClipY != newClipY {
			window.invalidateWindowClipPropertyState()
		}
		if oldOverflow != newOverflow {
			window.invalidateWindowScrollPropertyState()
		}
	}
	if !styleVisualKeyEqual(oldVisual, newVisual) {
		window.invalidateWindowEffectPropertyState()
	}
	changed := window.applyStyleBounds()
	if window.styleLayoutChanged(oldStyle, window.Style) {
		window.layoutDirty = true
		window.renderListValid = false
		window.hoverDirty = true
		window.lastMouseValid = false
		changed = true
	}
	if !styleVisualKeyEqual(oldVisual, newVisual) {
		if window.client.Width > 0 && window.client.Height > 0 {
			window.Invalidate(Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height})
		}
		changed = true
	}
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
	if value, ok := resolveLength(window.Style.left); ok {
		x = value
	}
	if value, ok := resolveLength(window.Style.top); ok {
		y = value
	}
	if value, ok := resolveLength(window.Style.width); ok {
		width = value
	}
	if value, ok := resolveLength(window.Style.height); ok {
		height = value
	}
	changed := window.setPosition(x, y)
	if window.setSize(width, height) {
		changed = true
	}
	return changed
}

func (window *Window) styleLayoutChanged(oldStyle Style, newStyle Style) bool {
	if boxInsets(oldStyle) != boxInsets(newStyle) {
		return true
	}
	return false
}

func (window *Window) overflowModeYForStyle(style Style) OverflowMode {
	if window == nil {
		return OverflowVisible
	}
	if style.overflow == nil && style.overflowY == nil {
		return OverflowAuto
	}
	return overflowModeFor(style, "y")
}

func (window *Window) scrollEnabledForStyle(style Style) bool {
	if window == nil || !WindowScrollYEnabled {
		return false
	}
	mode := window.overflowModeYForStyle(style)
	return mode == OverflowScroll || mode == OverflowAuto
}

func (window *Window) styleClipAxes(style Style) (bool, bool) {
	if window == nil {
		return false, false
	}
	clipX, clipY := overflowClipAxes(style)
	if !clipY && window.scrollEnabledForStyle(style) {
		clipY = true
	}
	return clipX, clipY
}

func (window *Window) styleDisplayStateChanged(oldStyle Style, newStyle Style) bool {
	if window == nil {
		return true
	}
	if boxInsets(oldStyle) != boxInsets(newStyle) {
		return true
	}
	oldClipX, oldClipY := window.styleClipAxes(oldStyle)
	newClipX, newClipY := window.styleClipAxes(newStyle)
	return oldClipX != newClipX || oldClipY != newClipY
}

func (window *Window) overflowModeY() OverflowMode {
	if window == nil {
		return OverflowVisible
	}
	return window.overflowModeYForStyle(window.Style)
}

func (window *Window) scrollEnabled() bool {
	return window.scrollEnabledForStyle(window.Style)
}
