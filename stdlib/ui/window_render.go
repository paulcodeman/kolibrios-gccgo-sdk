package ui

import "kos"

func (window *Window) Redraw() {
	if window == nil {
		return
	}
	window.syncWindowInfo()
	window.ensureCanvas()

	window.drawFrame()
	kos.BeginRedraw()
	kos.OpenWindow(window.X, window.Y, window.Width, window.Height, window.Title)
	if window.canvas != nil {
		window.canvas.BlitToWindow(window.client.X, window.client.Y)
	}
	if WindowEnableTinyGL {
		window.drawTinyGL(true, Rect{})
	}
	kos.EndRedraw()
	window.dirtySet = false
	window.noteCaretBlinkDrawn()
}

func (window *Window) RedrawContent() {
	if window == nil {
		return
	}
	window.syncWindowInfo()
	window.ensureCanvas()
	if !window.collectDirty() {
		return
	}
	window.drawDirty()
	window.blitDirty()
	if WindowEnableTinyGL {
		window.drawTinyGL(false, window.dirty)
	}
	window.noteCaretBlinkDrawn()
}

func (window *Window) RedrawContentStats(stats *FrameStats) {
	if window == nil {
		return
	}
	if stats == nil {
		window.RedrawContent()
		return
	}
	*stats = FrameStats{}
	window.syncWindowInfo()
	window.ensureCanvas()
	start := kos.UptimeNanoseconds()
	if !window.collectDirty() {
		stats.TotalNs = 0
		stats.DrawNs = 0
		stats.BlitNs = 0
		return
	}
	window.drawDirtyStats(stats)
	afterDraw := kos.UptimeNanoseconds()
	window.blitDirty()
	if WindowEnableTinyGL {
		window.drawTinyGL(false, window.dirty)
	}
	end := kos.UptimeNanoseconds()
	stats.BlitNs = end - afterDraw
	stats.TotalNs = end - start
	window.noteCaretBlinkDrawn()
}

// RenderStats performs a headless render pass into the offscreen canvas without
// issuing window blits. Useful for headless profiling runs.
func (window *Window) RenderStats(stats *FrameStats) {
	if window == nil {
		return
	}
	if stats == nil {
		return
	}
	*stats = FrameStats{}
	window.ensureCanvas()
	start := kos.UptimeNanoseconds()
	if !window.collectDirty() {
		stats.TotalNs = 0
		stats.DrawNs = 0
		stats.BlitNs = 0
		return
	}
	window.drawDirtyStats(stats)
	end := kos.UptimeNanoseconds()
	stats.TotalNs = end - start
	// No blit in headless mode; mark dirty as processed.
	window.dirtySet = false
}

// RenderListStats draws the current render list without layout or render-list rebuilds.
// Useful for isolating drawRenderList performance.
func (window *Window) RenderListStats(stats *FrameStats) {
	if window == nil || stats == nil {
		return
	}
	*stats = FrameStats{}
	window.ensureCanvas()
	// Ensure we have a render list at least once.
	window.ensureRenderList()
	start := kos.UptimeNanoseconds()
	window.drawBackgroundFull()
	afterClear := kos.UptimeNanoseconds()
	stats.ClearNs = afterClear - start
	window.drawRenderList(true, Rect{}, stats)
	stats.DrawNs = kos.UptimeNanoseconds() - start
	stats.TotalNs = stats.DrawNs
	window.dirtySet = false
}

// RenderStatsFull performs a full render pass (layout + full redraw) for headless runs.
// It skips dirty-diff tracking to avoid expensive merges during stress tests.
func (window *Window) RenderStatsFull(stats *FrameStats) {
	if window == nil {
		return
	}
	if stats == nil {
		return
	}
	*stats = FrameStats{}
	window.ensureCanvas()
	start := kos.UptimeNanoseconds()
	window.drawFrameStats(stats)
	end := kos.UptimeNanoseconds()
	stats.TotalNs = end - start
}

func (window *Window) drawFrame() {
	if window == nil || window.canvas == nil {
		return
	}
	if window.layoutDirty {
		window.layoutFlow()
	}
	window.ensureRenderList()
	window.drawBackgroundFull()
	window.drawRenderList(true, Rect{}, nil)
	window.drawWindowScrollbar(true, Rect{})
	window.dirtySet = false
	window.layoutDirty = false
}

func (window *Window) drawFrameStats(stats *FrameStats) {
	if window == nil || window.canvas == nil || stats == nil {
		return
	}
	if window.layoutDirty {
		startLayout := kos.UptimeNanoseconds()
		window.layoutFlow()
		stats.LayoutNs = kos.UptimeNanoseconds() - startLayout
	}
	startList := kos.UptimeNanoseconds()
	window.ensureRenderList()
	stats.RenderListNs = kos.UptimeNanoseconds() - startList
	start := kos.UptimeNanoseconds()
	window.drawBackgroundFull()
	afterClear := kos.UptimeNanoseconds()
	stats.ClearNs = afterClear - start
	window.drawRenderList(true, Rect{}, stats)
	window.drawWindowScrollbar(true, Rect{})
	stats.DrawNs = kos.UptimeNanoseconds() - start
	window.dirtySet = false
	window.layoutDirty = false
}

func (window *Window) drawDirty() {
	if window == nil || window.canvas == nil || !window.dirtySet {
		return
	}
	full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	if window.dirty == full {
		window.drawFrame()
		window.dirty = full
		window.dirtySet = true
		return
	}
	if color, ok := window.simpleBackgroundColor(); ok {
		window.canvas.FillRect(window.dirty.X, window.dirty.Y, window.dirty.Width, window.dirty.Height, color)
	} else if cache := window.ensureBackgroundCache(); cache != nil {
		window.canvas.BlitFrom(cache, window.dirty, window.dirty.X, window.dirty.Y)
	} else {
		window.drawFrame()
		window.dirty = full
		window.dirtySet = true
		return
	}
	window.ensureRenderList()
	window.drawRenderList(false, window.dirty, nil)
	window.drawWindowScrollbar(false, window.dirty)
}

func (window *Window) drawDirtyStats(stats *FrameStats) {
	if window == nil || window.canvas == nil || stats == nil || !window.dirtySet {
		return
	}
	start := kos.UptimeNanoseconds()
	full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	if window.dirty == full {
		window.drawFrameStats(stats)
		window.dirty = full
		window.dirtySet = true
		return
	}
	if color, ok := window.simpleBackgroundColor(); ok {
		window.canvas.FillRect(window.dirty.X, window.dirty.Y, window.dirty.Width, window.dirty.Height, color)
	} else if cache := window.ensureBackgroundCache(); cache != nil {
		window.canvas.BlitFrom(cache, window.dirty, window.dirty.X, window.dirty.Y)
	} else {
		window.drawFrameStats(stats)
		window.dirty = full
		window.dirtySet = true
		return
	}
	afterClear := kos.UptimeNanoseconds()
	stats.ClearNs = afterClear - start
	startList := kos.UptimeNanoseconds()
	window.ensureRenderList()
	stats.RenderListNs = kos.UptimeNanoseconds() - startList
	window.drawRenderList(false, window.dirty, stats)
	window.drawWindowScrollbar(false, window.dirty)
	stats.DrawNs = kos.UptimeNanoseconds() - start
}

func (window *Window) blitDirty() {
	if window == nil || window.canvas == nil || !window.dirtySet {
		return
	}
	rect := window.dirty
	window.canvas.BlitRectToWindow(rect, window.client.X+rect.X, window.client.Y+rect.Y)
	window.dirtySet = false
}
