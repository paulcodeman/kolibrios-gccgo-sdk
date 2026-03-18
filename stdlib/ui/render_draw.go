package ui

func (window *Window) rootClipState() clipState {
	if window == nil || window.canvas == nil {
		return clipState{}
	}
	clipX, clipY := overflowClipAxes(window.Style)
	if !clipY && window.scrollEnabled() {
		clipY = true
	}
	if !clipX && !clipY {
		return clipState{}
	}
	content := window.contentRect()
	if content.Empty() {
		return clipState{rect: Rect{}, set: true}
	}
	canvasBounds := Rect{X: 0, Y: 0, Width: window.canvas.Width(), Height: window.canvas.Height()}
	base := canvasBounds
	if clipX {
		base.X = content.X
		base.Width = content.Width
	}
	if clipY {
		base.Y = content.Y
		base.Height = content.Height
	}
	base = IntersectRect(base, canvasBounds)
	return clipState{rect: base, set: true}
}

func (window *Window) drawRenderList(full bool, dirty Rect, stats *FrameStats) {
	if window == nil || window.canvas == nil {
		return
	}
	window.currentDisplayList().Paint(window, full, dirty, stats)
}

func (window *Window) drawElementWithOffset(element *Element, offsetY int) {
	if window == nil || window.canvas == nil || element == nil {
		return
	}
	style := element.effectiveStyle()
	if display, ok := resolveDisplay(style.display); ok && display == DisplayNone {
		return
	}
	element.updateRenderKey(style)
	rect := element.layoutRect
	if rect.Empty() {
		element.applyLayout(window.canvas, style)
		rect = element.layoutRect
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	if element.tryDrawFromRetainedSubtreeLayer(window.canvas, style, offsetY) {
		return
	}
	if offsetY != 0 {
		rect.Y += offsetY
	}
	if element.tryDrawFromCache(window.canvas, rect, style) {
		return
	}
	element.drawToRect(window.canvas, rect, style)
}
