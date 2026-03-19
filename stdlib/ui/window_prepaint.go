package ui

import "kos"

type windowPrepaintMode uint8

const (
	windowPrepaintNone windowPrepaintMode = iota
	windowPrepaintFull
	windowPrepaintPartial
)

type windowPrepaintClearMode uint8

const (
	windowPrepaintClearNone windowPrepaintClearMode = iota
	windowPrepaintClearSolid
	windowPrepaintClearCache
)

type windowPrepaintPlan struct {
	mode               windowPrepaintMode
	drawContent        bool
	drawScrollbar      bool
	visualOnly         bool
	dirty              Rect
	contentDirty       Rect
	scrollbarDirty     Rect
	scrollbarDirtySet  bool
	clearMode          windowPrepaintClearMode
	clearColor         kos.Color
	backgroundCache    *Canvas
	applyScrollBlit    bool
	applyTranslateBlit bool
}

func (window *Window) splitScrollbarDirty(dirty Rect) (Rect, Rect) {
	if window == nil || dirty.Empty() {
		return dirty, Rect{}
	}
	track, _, _, ok := window.windowScrollbarLayout()
	if !ok {
		return dirty, Rect{}
	}
	scrollbarDirty := IntersectRect(dirty, track)
	if scrollbarDirty.Empty() {
		return dirty, Rect{}
	}
	contentDirty := dirty
	if rectContainsRect(track, dirty) {
		contentDirty = Rect{}
	}
	return contentDirty, scrollbarDirty
}

func (window *Window) scrollDirtyRectWithState(scrollState windowScrollPropertyState) Rect {
	if window == nil {
		return Rect{}
	}
	viewport := scrollState.viewport
	if viewport.Empty() {
		return Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	}
	dirty := viewport
	if window.canUseScrollBlit(viewport) {
		exposed := scrollExposeRect(viewport, scrollState.deltaY)
		if !exposed.Empty() {
			dirty = exposed
		}
	}
	if scrollState.visible {
		dirty = UnionRect(dirty, scrollState.track)
	}
	return dirty
}

func (window *Window) buildPrepaintPlanWithState(scrollState windowScrollPropertyState, effectState windowEffectPropertyState, dirtyPlan windowDirtyPlan) (windowPrepaintPlan, bool) {
	if window == nil || window.canvas == nil || !window.dirtySet || !dirtyPlan.dirtySet {
		return windowPrepaintPlan{}, false
	}
	full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	plan := windowPrepaintPlan{
		mode:          windowPrepaintPartial,
		drawContent:   true,
		drawScrollbar: true,
		visualOnly:    dirtyPlan.hasDamage(windowDirtyDamageVisual),
		dirty:         dirtyPlan.dirty,
		contentDirty:  dirtyPlan.dirty,
	}
	if dirtyPlan.hasDamage(windowDirtyDamageFull) || plan.dirty == full {
		plan.mode = windowPrepaintFull
		plan.drawContent = true
		plan.drawScrollbar = true
		plan.dirty = full
		plan.contentDirty = full
		plan.scrollbarDirty = full
		return plan, true
	}
	if effectState.simpleBackground {
		plan.clearMode = windowPrepaintClearSolid
		plan.clearColor = effectState.backgroundColor
	} else if effectState.backgroundCache != nil {
		plan.clearMode = windowPrepaintClearCache
		plan.backgroundCache = effectState.backgroundCache
	} else if effectState.needsFullRedraw {
		plan.mode = windowPrepaintFull
		plan.drawContent = true
		plan.drawScrollbar = true
		plan.dirty = full
		return plan, true
	}
	if dirtyPlan.mode == windowDirtyPlanNone && !dirtyPlan.hasDamage(windowDirtyDamageScroll) {
		plan.contentDirty, plan.scrollbarDirty = window.splitScrollbarDirty(plan.dirty)
		plan.scrollbarDirtySet = !plan.scrollbarDirty.Empty()
		plan.drawContent = !plan.contentDirty.Empty()
		plan.drawScrollbar = plan.scrollbarDirtySet
	}
	if dirtyPlan.mode == windowDirtyPlanNone &&
		dirtyPlan.hasDamage(windowDirtyDamageScroll) &&
		scrollState.enabled &&
		window.canUseScrollBlit(scrollState.viewport) &&
		dirtyPlan.dirty == window.scrollDirtyRectWithState(scrollState) {
		plan.applyScrollBlit = true
		plan.contentDirty = scrollExposeRect(scrollState.viewport, scrollState.deltaY)
		if plan.contentDirty.Empty() {
			plan.contentDirty = scrollState.viewport
		}
		if scrollState.visible {
			plan.scrollbarDirty = scrollState.track
			plan.scrollbarDirtySet = true
		}
		plan.drawContent = !plan.contentDirty.Empty()
		plan.drawScrollbar = plan.scrollbarDirtySet
	}
	if dirtyPlan.hasDamage(windowDirtyDamageTranslate) && !dirtyPlan.hasDamage(windowDirtyDamageScroll) {
		plan.applyTranslateBlit = true
	}
	return plan, true
}

func (window *Window) buildPrepaintPlan() (windowPrepaintPlan, bool) {
	if window == nil {
		return windowPrepaintPlan{}, false
	}
	dirtyPlan, ok := window.currentFrameDirtyPlan()
	if !ok {
		return windowPrepaintPlan{}, false
	}
	return window.buildPrepaintPlanWithState(window.currentFrameScrollPropertyState(), window.currentFrameEffectPropertyState(), dirtyPlan)
}

func (window *Window) applyPrepaintPlan(plan windowPrepaintPlan) {
	if window == nil || window.canvas == nil {
		return
	}
	if plan.applyScrollBlit {
		window.applyPendingScrollBlit()
	}
	if plan.applyTranslateBlit {
		window.applyPendingTranslateBlits()
	}
	switch plan.clearMode {
	case windowPrepaintClearSolid:
		window.canvas.FillRect(plan.contentDirty.X, plan.contentDirty.Y, plan.contentDirty.Width, plan.contentDirty.Height, plan.clearColor)
	case windowPrepaintClearCache:
		if plan.backgroundCache != nil {
			window.canvas.BlitFrom(plan.backgroundCache, plan.contentDirty, plan.contentDirty.X, plan.contentDirty.Y)
		}
	}
}
