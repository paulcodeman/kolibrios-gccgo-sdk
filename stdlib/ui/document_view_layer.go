package ui

// DocumentViewRetainedLayer enables drawing DocumentView into its own retained
// offscreen surface before compositing it into the window canvas.
var DocumentViewRetainedLayer = true

func (view *DocumentView) useRetainedLayer(style Style) bool {
	if view == nil || !DocumentViewRetainedLayer || FastNoCache {
		return false
	}
	if view.Document == nil {
		return false
	}
	if display, ok := resolveDisplay(style.display); ok && display == DisplayNone {
		return false
	}
	return true
}

func (view *DocumentView) retainedLayerVisual(style Style, rect Rect) (Rect, Rect) {
	visual := visualBoundsForStyle(rect, style, false)
	if visual.Empty() {
		return Rect{}, Rect{}
	}
	localRect := Rect{
		X:      rect.X - visual.X,
		Y:      rect.Y - visual.Y,
		Width:  rect.Width,
		Height: rect.Height,
	}
	return visual, localRect
}

func (view *DocumentView) retainedLayerKey(style Style, localRect Rect, visual Rect) (styleVisualKey, int, int, int, int) {
	return visualKeyFor(style), visual.Width, visual.Height, localRect.X, localRect.Y
}

func (view *DocumentView) ensureRetainedLayer(style Style) (Rect, Rect, bool) {
	if view == nil || view.layoutRect.Empty() {
		return Rect{}, Rect{}, false
	}
	visual, localRect := view.retainedLayerVisual(style, view.layoutRect)
	if visual.Empty() || localRect.Empty() {
		return visual, localRect, false
	}
	key, width, height, offsetX, offsetY := view.retainedLayerKey(style, localRect, visual)
	if width <= 0 || height <= 0 {
		return visual, localRect, false
	}
	if view.layerCanvas == nil || view.layerCanvas.Width() != width || view.layerCanvas.Height() != height {
		view.layerCanvas = NewCanvasAlpha(width, height)
		view.layerValid = false
	}
	if view.layerWidth != width || view.layerHeight != height ||
		view.layerOffsetX != offsetX || view.layerOffsetY != offsetY ||
		!styleVisualKeyEqual(view.layerVisualKey, key) {
		view.layerWidth = width
		view.layerHeight = height
		view.layerOffsetX = offsetX
		view.layerOffsetY = offsetY
		view.layerVisualKey = key
		view.layerValid = false
	}
	if view.layerCanvas == nil {
		return visual, localRect, false
	}
	if !view.layerValid {
		view.redrawRetainedLayer(style, visual, localRect)
	}
	return visual, localRect, view.layerValid
}

func (view *DocumentView) redrawRetainedLayer(style Style, visual Rect, localRect Rect) {
	if view == nil || view.layerCanvas == nil || visual.Empty() || localRect.Empty() {
		return
	}
	view.layerCanvas.ClearTransparent()
	drawStyledBox(view.layerCanvas, localRect, style, localRect, nil)
	if view.Document != nil {
		viewport := view.documentViewportRectIn(localRect, style)
		if !viewport.Empty() {
			view.layerCanvas.PushClip(viewport)
			view.Document.PaintOffset(view.layerCanvas, -visual.X, -visual.Y-view.scrollY)
			view.layerCanvas.PopClip()
		}
		view.drawDocumentScrollbar(view.layerCanvas, localRect, style)
	}
	view.drawnScrollY = view.scrollY
	view.layerValid = true
}

func (view *DocumentView) updateRetainedLayerForScroll(style Style, visual Rect, localRect Rect) bool {
	if view == nil || view.layerCanvas == nil || !view.layerValid || view.Document == nil {
		return false
	}
	viewport := view.documentViewportRectIn(localRect, style)
	if viewport.Empty() {
		return false
	}
	if !view.canUseScrollBlit(style, viewport) {
		return false
	}
	delta := view.pendingScrollDelta()
	if delta == 0 {
		return false
	}
	view.layerCanvas.ScrollRectY(viewport, -delta)
	exposed := scrollExposeRect(viewport, view.pendingScrollDelta())
	if !exposed.Empty() {
		view.layerCanvas.PushClip(exposed)
		view.Document.PaintOffset(view.layerCanvas, -visual.X, -visual.Y-view.scrollY)
		view.layerCanvas.PopClip()
	}
	view.drawDocumentScrollbar(view.layerCanvas, localRect, style)
	view.drawnScrollY = view.scrollY
	return true
}

func (view *DocumentView) drawRetainedLayer(canvas *Canvas, style Style, offsetY int) bool {
	if view == nil || canvas == nil || !view.useRetainedLayer(style) {
		return false
	}
	visual, localRect, ok := view.ensureRetainedLayer(style)
	if !ok || view.layerCanvas == nil {
		return false
	}
	if view.pendingScrollDelta() != 0 && view.layerValid {
		if !view.updateRetainedLayerForScroll(style, visual, localRect) {
			view.redrawRetainedLayer(style, visual, localRect)
		}
	}
	targetVisual := visual
	if offsetY != 0 {
		targetVisual.Y += offsetY
	}
	canvas.BlitFrom(view.layerCanvas, Rect{X: 0, Y: 0, Width: view.layerWidth, Height: view.layerHeight}, targetVisual.X, targetVisual.Y)
	return true
}
