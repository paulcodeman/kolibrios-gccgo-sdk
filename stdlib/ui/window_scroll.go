package ui

func (window *Window) updateScrollMetrics() {
	if window == nil {
		return
	}
	if !window.scrollEnabled() {
		window.scrollMaxY = 0
		if window.scrollY != 0 {
			window.scrollY = 0
			window.noteScrollChanged()
		}
		return
	}
	content := window.contentRect()
	if content.Empty() || content.Height <= 0 {
		window.scrollMaxY = 0
		if window.scrollY != 0 {
			window.scrollY = 0
			window.noteScrollChanged()
		}
		return
	}
	maxBottom := content.Y
	for _, bounds := range window.nodeBounds {
		if bounds.Empty() {
			continue
		}
		bottom := bounds.Y + bounds.Height
		if bottom > maxBottom {
			maxBottom = bottom
		}
	}
	maxScroll := maxBottom - (content.Y + content.Height)
	if maxScroll < 0 {
		maxScroll = 0
	}
	window.scrollMaxY = maxScroll
	if window.scrollY < 0 {
		window.scrollY = 0
		window.noteScrollChanged()
		return
	}
	if window.scrollY > maxScroll {
		window.scrollY = maxScroll
		window.noteScrollChanged()
	}
}

func (window *Window) windowScrollbarLayout() (Rect, Rect, int, bool) {
	if window == nil || !window.scrollEnabled() {
		return Rect{}, Rect{}, 0, false
	}
	window.updateScrollMetrics()
	content := window.contentRect()
	if window.scrollMaxY <= 0 || content.Empty() {
		return Rect{}, Rect{}, 0, false
	}
	scrollbar := resolveScrollbarStyle(window.Style)
	width := scrollbar.width
	if width <= 0 {
		return Rect{}, Rect{}, 0, false
	}
	minWidth := width + scrollbar.padding.Left + scrollbar.padding.Right
	if content.Width <= minWidth {
		return Rect{}, Rect{}, 0, false
	}
	track := Rect{
		X:      content.X + content.Width - width - scrollbar.padding.Right,
		Y:      content.Y + scrollbar.padding.Top,
		Width:  width,
		Height: content.Height - scrollbar.padding.Top - scrollbar.padding.Bottom,
	}
	if track.Width <= 0 || track.Height <= 0 {
		return Rect{}, Rect{}, 0, false
	}
	contentHeight := content.Height + window.scrollMaxY
	thumbHeight := 0
	if contentHeight > 0 {
		thumbHeight = track.Height * content.Height / contentHeight
	}
	if thumbHeight < defaultScrollbarMinThumb {
		thumbHeight = defaultScrollbarMinThumb
	}
	if thumbHeight > track.Height {
		thumbHeight = track.Height
	}
	thumbY := track.Y
	offsetRange := track.Height - thumbHeight
	if offsetRange > 0 && window.scrollMaxY > 0 {
		thumbY = track.Y + window.scrollY*offsetRange/window.scrollMaxY
	}
	thumb := Rect{
		X:      track.X,
		Y:      thumbY,
		Width:  track.Width,
		Height: thumbHeight,
	}
	return track, thumb, window.scrollMaxY, true
}

func (window *Window) windowScrollbarHit(x int, y int) bool {
	track, _, _, ok := window.windowScrollbarLayout()
	if !ok {
		return false
	}
	return track.Contains(x, y)
}

func (window *Window) handleWindowScrollbarMouseDown(x int, y int) bool {
	if window == nil || !window.scrollEnabled() {
		return false
	}
	track, thumb, maxScroll, ok := window.windowScrollbarLayout()
	if !ok || maxScroll <= 0 {
		return false
	}
	if !track.Contains(x, y) {
		return false
	}
	if thumb.Contains(x, y) {
		window.scrollDragActive = true
		window.scrollDragOffset = y - thumb.Y
		return true
	}
	trackRange := track.Height - thumb.Height
	if trackRange <= 0 {
		return false
	}
	target := y - track.Y - thumb.Height/2
	if target < 0 {
		target = 0
	} else if target > trackRange {
		target = trackRange
	}
	next := target * maxScroll / trackRange
	if next != window.scrollY {
		window.scrollY = next
		window.noteScrollChanged()
	}
	return true
}

func (window *Window) handleWindowScrollbarDrag(y int) bool {
	if window == nil || !window.scrollDragActive {
		return false
	}
	track, thumb, maxScroll, ok := window.windowScrollbarLayout()
	if !ok || maxScroll <= 0 {
		return false
	}
	trackRange := track.Height - thumb.Height
	if trackRange <= 0 {
		return false
	}
	target := y - window.scrollDragOffset - track.Y
	if target < 0 {
		target = 0
	} else if target > trackRange {
		target = trackRange
	}
	next := target * maxScroll / trackRange
	if next != window.scrollY {
		window.scrollY = next
		window.noteScrollChanged()
		return true
	}
	return false
}

func scrollBarRadii(radius int) CornerRadii {
	if radius <= 0 {
		return CornerRadii{}
	}
	return CornerRadii{
		TopLeft:     radius,
		TopRight:    radius,
		BottomRight: radius,
		BottomLeft:  radius,
	}
}

func (window *Window) drawWindowScrollbar(full bool, dirty Rect) {
	if window == nil || window.canvas == nil || !window.scrollEnabled() {
		return
	}
	track, thumb, _, ok := window.windowScrollbarLayout()
	if !ok {
		return
	}
	if !full {
		union := UnionRect(track, thumb)
		if IntersectRect(union, dirty).Empty() {
			return
		}
		window.canvas.PushClip(dirty)
		defer window.canvas.PopClip()
	}
	scrollbar := resolveScrollbarStyle(window.Style)
	radii := scrollBarRadii(scrollbar.radius)
	window.canvas.FillRoundedRect(track.X, track.Y, track.Width, track.Height, radii, scrollbar.track)
	window.canvas.FillRoundedRect(thumb.X, thumb.Y, thumb.Width, thumb.Height, radii, scrollbar.thumb)
}

func (window *Window) noteScrollChanged() {
	if window == nil {
		return
	}
	window.invalidateWindowPropertyState()
	window.hoverDirty = true
	window.lastMouseValid = false
	window.scrollRedraw = true
	if window.renderListValid {
		window.invalidateHitGrid()
	}
	viewport := window.scrollViewportRect()
	if viewport.Empty() {
		full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
		window.Invalidate(full)
		return
	}
	dirty := viewport
	delta := window.pendingScrollDelta()
	if window.canUseScrollBlit(viewport) {
		exposed := scrollExposeRect(viewport, delta)
		if !exposed.Empty() {
			dirty = exposed
		}
		window.markPresentRect(viewport)
	}
	if track, _, _, ok := window.windowScrollbarLayout(); ok {
		dirty = UnionRect(dirty, track)
	}
	window.Invalidate(dirty)
}

func (window *Window) scrollWindowBy(deltaY int) bool {
	if window == nil || !window.scrollEnabled() || deltaY == 0 {
		return false
	}
	if !window.renderListValid {
		window.ensureRenderList()
	}
	window.updateScrollMetrics()
	if window.scrollMaxY <= 0 {
		return false
	}
	metrics := metricsForStyle(window.Style)
	lineHeight := metrics.height
	if lineHeight <= 0 {
		lineHeight = defaultFontHeight
	}
	step := lineHeight * 3
	if step < lineHeight {
		step = lineHeight
	}
	prev := window.scrollY
	window.scrollY += deltaY * step
	if window.scrollY < 0 {
		window.scrollY = 0
	} else if window.scrollY > window.scrollMaxY {
		window.scrollY = window.scrollMaxY
	}
	if window.scrollY != prev {
		window.noteScrollChanged()
		return true
	}
	return false
}
