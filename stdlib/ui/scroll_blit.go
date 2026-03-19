package ui

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func scrollExposeRect(viewport Rect, scrollDelta int) Rect {
	if viewport.Empty() || scrollDelta == 0 {
		return Rect{}
	}
	delta := absInt(scrollDelta)
	if delta >= viewport.Height {
		return viewport
	}
	if scrollDelta > 0 {
		return Rect{
			X:      viewport.X,
			Y:      viewport.Y + viewport.Height - delta,
			Width:  viewport.Width,
			Height: delta,
		}
	}
	return Rect{
		X:      viewport.X,
		Y:      viewport.Y,
		Width:  viewport.Width,
		Height: delta,
	}
}

func styleHasOpaqueSolidBackground(style Style) bool {
	if style.background == nil {
		return false
	}
	if style.gradient != nil || style.shadow != nil {
		return false
	}
	if borderWidthFor(style) > 0 {
		return false
	}
	if resolveBorderRadius(style).Active() {
		return false
	}
	if value, ok := resolveOpacity(style.opacity); ok && value < 255 {
		return false
	}
	background, ok := resolveColor(style.background)
	if !ok {
		return false
	}
	_, alpha := colorValueAndAlpha(background)
	return alpha == 255
}

func (window *Window) markPresentRect(rect Rect) {
	if window == nil || rect.Empty() {
		return
	}
	client := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	rect = IntersectRect(rect, client)
	if rect.Empty() {
		return
	}
	if window.presentRectSet {
		window.presentRect = UnionRect(window.presentRect, rect)
		return
	}
	window.presentRect = rect
	window.presentRectSet = true
}

func (window *Window) clearPresentRect() {
	if window == nil {
		return
	}
	window.presentRect = Rect{}
	window.presentRectSet = false
}

func (window *Window) syncScrollDrawState() {
	if window == nil {
		return
	}
	defer func() {
		window.skipScrollBlitOnce = false
	}()
	if window.drawnScrollY == window.scrollY {
		return
	}
	window.drawnScrollY = window.scrollY
	window.invalidateWindowScrollPropertyState()
}

func (window *Window) scrollViewportRect() Rect {
	if window == nil {
		return Rect{}
	}
	return window.currentFrameScrollPropertyState().viewport
}

func (window *Window) pendingScrollDelta() int {
	if window == nil {
		return 0
	}
	return window.currentFrameScrollPropertyState().deltaY
}

func (window *Window) canUseScrollBlit(viewport Rect) bool {
	if window == nil || window.canvas == nil || viewport.Empty() {
		return false
	}
	if window.skipScrollBlitOnce {
		return false
	}
	if !window.currentFrameEffectPropertyState().simpleBackground {
		return false
	}
	delta := window.pendingScrollDelta()
	if delta == 0 {
		return false
	}
	return absInt(delta) < viewport.Height
}

func (window *Window) applyPendingScrollBlit() bool {
	if window == nil || window.canvas == nil {
		return false
	}
	viewport := window.scrollViewportRect()
	if !window.canUseScrollBlit(viewport) {
		return false
	}
	delta := window.pendingScrollDelta()
	if delta == 0 {
		return false
	}
	window.canvas.ScrollRectY(viewport, -delta)
	window.markPresentRect(viewport)
	return true
}

func (view *DocumentView) pendingScrollDelta() int {
	if view == nil {
		return 0
	}
	return view.scrollY - view.drawnScrollY
}

func (view *DocumentView) canUseScrollBlit(style Style, viewport Rect) bool {
	if view == nil || viewport.Empty() {
		return false
	}
	if view.skipScrollBlitOnce {
		return false
	}
	if !styleHasOpaqueSolidBackground(style) {
		return false
	}
	delta := view.pendingScrollDelta()
	if delta == 0 {
		return false
	}
	return absInt(delta) < viewport.Height
}

func (view *DocumentView) applyPendingScrollBlit(canvas *Canvas, style Style, viewport Rect) bool {
	if view == nil || canvas == nil || viewport.Empty() {
		return false
	}
	if !view.canUseScrollBlit(style, viewport) {
		return false
	}
	delta := view.pendingScrollDelta()
	if delta == 0 {
		return false
	}
	canvas.ScrollRectY(viewport, -delta)
	if view.window != nil {
		view.window.markPresentRect(viewport)
	}
	return true
}
