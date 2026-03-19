package ui

import "kos"

func (window *Window) Redraw() {
	if window == nil {
		return
	}
	window.syncWindowInfo()
	window.ensureCanvas()

	window.drawFrame()
	if presenter := window.presenter(); presenter != nil {
		presenter.PresentFull(window.canvas)
	}
	if WindowEnableTinyGL {
		window.drawTinyGL(true, Rect{})
	}
	window.dirtySet = false
	window.clearPresentRect()
	window.syncScrollDrawState()
	window.noteCaretBlinkDrawn()
}

func (window *Window) RedrawContent() {
	if window == nil {
		return
	}
	if window.client.Empty() || window.canvas == nil {
		window.syncWindowInfo()
	}
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
	if window.client.Empty() || window.canvas == nil {
		window.syncWindowInfo()
	}
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
	window.clearPresentRect()
	window.syncScrollDrawState()
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
	window.clearPresentRect()
	window.syncScrollDrawState()
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
	window.clearPresentRect()
	window.syncScrollDrawState()
}

func (window *Window) drawFrame() {
	if window == nil || window.canvas == nil {
		return
	}
	window.resetTranslateBlits()
	if window.layoutDirty {
		window.layoutFlow()
	}
	window.ensureRenderList()
	window.drawBackgroundFull()
	window.drawRenderList(true, Rect{}, nil)
	window.drawWindowScrollbar(true, Rect{})
	window.dirtySet = false
	window.layoutDirty = false
	window.clearPresentRect()
	window.syncScrollDrawState()
}

func (window *Window) drawFrameStats(stats *FrameStats) {
	if window == nil || window.canvas == nil || stats == nil {
		return
	}
	window.resetTranslateBlits()
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
	window.clearPresentRect()
	window.syncScrollDrawState()
}

func (window *Window) drawDirty() {
	plan, ok := window.buildPrepaintPlan()
	if !ok {
		return
	}
	if plan.mode == windowPrepaintFull {
		window.resetTranslateBlits()
		window.drawFrame()
		window.dirty = plan.dirty
		window.dirtySet = true
		return
	}
	window.applyPrepaintPlan(plan)
	window.ensureRenderList()
	window.drawRenderList(false, plan.dirty, nil)
	window.drawWindowScrollbar(false, plan.dirty)
	window.syncScrollDrawState()
}

func (window *Window) drawDirtyStats(stats *FrameStats) {
	if stats == nil {
		return
	}
	plan, ok := window.buildPrepaintPlan()
	if !ok {
		return
	}
	start := kos.UptimeNanoseconds()
	if plan.mode == windowPrepaintFull {
		window.resetTranslateBlits()
		window.drawFrameStats(stats)
		window.dirty = plan.dirty
		window.dirtySet = true
		return
	}
	window.applyPrepaintPlan(plan)
	afterClear := kos.UptimeNanoseconds()
	stats.ClearNs = afterClear - start
	startList := kos.UptimeNanoseconds()
	window.ensureRenderList()
	stats.RenderListNs = kos.UptimeNanoseconds() - startList
	window.drawRenderList(false, plan.dirty, stats)
	window.drawWindowScrollbar(false, plan.dirty)
	stats.DrawNs = kos.UptimeNanoseconds() - start
	window.syncScrollDrawState()
}

func (window *Window) blitDirty() {
	if window == nil || window.canvas == nil || !window.dirtySet {
		return
	}
	rect := window.dirty
	if window.presentRectSet {
		rect = UnionRect(rect, window.presentRect)
	}
	if presenter := window.presenter(); presenter != nil {
		presenter.PresentRect(window.canvas, rect)
	}
	window.dirtySet = false
	window.clearPresentRect()
}
