package ui

func (window *Window) noteScrollMetricsBoundsChanged() {
	if window == nil {
		return
	}
	window.scrollMetricsGen++
	if window.scrollMetricsGen == 0 {
		window.scrollMetricsGen = 1
	}
	window.scrollMetricsValid = false
}

func (window *Window) setScrollMaxY(maxScroll int) {
	if window == nil {
		return
	}
	if maxScroll < 0 {
		maxScroll = 0
	}
	oldMaxScroll := window.scrollMaxY
	if oldMaxScroll == maxScroll {
		return
	}
	window.scrollMaxY = maxScroll
	window.invalidateWindowScrollMetricsState()
	if window.backgroundScrollEffectChanged(oldMaxScroll, maxScroll) {
		window.invalidateWindowEffectPropertyState()
	}
}

func (window *Window) updateScrollMetrics() {
	if window == nil {
		return
	}
	enabled := window.scrollEnabled()
	content := window.contentRect()
	if window.scrollMetricsValid &&
		window.scrollMetricsCacheGen == window.scrollMetricsGen &&
		window.scrollMetricsCacheRect == content &&
		window.scrollMetricsCacheOn == enabled {
		return
	}
	defer func() {
		window.scrollMetricsCacheGen = window.scrollMetricsGen
		window.scrollMetricsCacheRect = content
		window.scrollMetricsCacheOn = enabled
		window.scrollMetricsValid = true
	}()
	if !enabled {
		window.setScrollMaxY(0)
		if window.scrollY != 0 {
			window.scrollY = 0
			window.noteScrollChanged()
		}
		return
	}
	if content.Empty() || content.Height <= 0 {
		window.setScrollMaxY(0)
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
	window.setScrollMaxY(maxScroll)
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
	state := window.windowScrollbarState()
	return scrollbarLayoutForState(state)
}

func scrollbarLayoutForState(state windowScrollPropertyState) (Rect, Rect, int, bool) {
	if !state.visible {
		return Rect{}, Rect{}, 0, false
	}
	return state.track, state.thumb, state.scrollMaxY, true
}

func (window *Window) windowScrollbarState() windowScrollPropertyState {
	if window == nil || !window.scrollEnabled() {
		return windowScrollPropertyState{}
	}
	window.updateScrollMetrics()
	return window.windowScrollPropertyStateValue()
}

func windowScrollbarHitWithState(state windowScrollPropertyState, x int, y int) bool {
	track, _, _, ok := scrollbarLayoutForState(state)
	if !ok {
		return false
	}
	return track.Contains(x, y)
}

func (window *Window) windowScrollbarHit(x int, y int) bool {
	return windowScrollbarHitWithState(window.windowScrollbarState(), x, y)
}

func (window *Window) handleWindowScrollbarMouseDownWithState(state windowScrollPropertyState, x int, y int) bool {
	if window == nil || !window.scrollEnabled() {
		return false
	}
	track, thumb, maxScroll, ok := scrollbarLayoutForState(state)
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

func (window *Window) handleWindowScrollbarMouseDown(x int, y int) bool {
	return window.handleWindowScrollbarMouseDownWithState(window.windowScrollbarState(), x, y)
}

func (window *Window) handleWindowScrollbarDragWithState(state windowScrollPropertyState, y int) bool {
	if window == nil || !window.scrollDragActive {
		return false
	}
	track, thumb, maxScroll, ok := scrollbarLayoutForState(state)
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

func (window *Window) handleWindowScrollbarDrag(y int) bool {
	return window.handleWindowScrollbarDragWithState(window.windowScrollbarState(), y)
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

func (window *Window) drawWindowScrollbarWithState(state windowScrollPropertyState, full bool, dirty Rect) {
	if window == nil || window.canvas == nil || !window.scrollEnabled() {
		return
	}
	track, thumb, _, ok := scrollbarLayoutForState(state)
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

func (window *Window) drawWindowScrollbar(full bool, dirty Rect) {
	if window == nil {
		return
	}
	state := window.windowScrollbarState()
	if window.frameStateActive {
		state = window.currentFrameScrollPropertyState()
	}
	window.drawWindowScrollbarWithState(state, full, dirty)
}

func (window *Window) noteScrollChanged() {
	if window == nil {
		return
	}
	window.invalidateWindowScrollPropertyState()
	if window.backgroundScrollDependent() {
		window.invalidateWindowEffectPropertyState()
	}
	window.invalidateHoverTracking()
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
	scrollState := window.windowScrollbarState()
	if scrollState.visible {
		dirty = UnionRect(dirty, scrollState.track)
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
