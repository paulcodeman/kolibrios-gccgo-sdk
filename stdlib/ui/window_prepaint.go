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

func (window *Window) splitVisualScrollbarDirty(dirty Rect) (Rect, Rect, bool) {
	if window == nil || dirty.Empty() {
		return dirty, Rect{}, false
	}
	track, _, _, ok := window.windowScrollbarLayout()
	if !ok {
		return dirty, Rect{}, false
	}
	scrollbarDirty := IntersectRect(dirty, track)
	if scrollbarDirty.Empty() {
		return dirty, Rect{}, false
	}
	contentDirty := dirty
	if rectContainsRect(track, dirty) {
		contentDirty = Rect{}
	}
	return contentDirty, scrollbarDirty, true
}

func (window *Window) scrollDirtyRectWithState(state windowPropertyState) Rect {
	if window == nil {
		return Rect{}
	}
	viewport := state.scroll.viewport
	if viewport.Empty() {
		return Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	}
	dirty := viewport
	if window.canUseScrollBlit(viewport) {
		exposed := scrollExposeRect(viewport, state.scroll.deltaY)
		if !exposed.Empty() {
			dirty = exposed
		}
	}
	if track, _, _, ok := window.windowScrollbarLayout(); ok {
		dirty = UnionRect(dirty, track)
	}
	return dirty
}

func (window *Window) buildPrepaintPlanWithState(state windowPropertyState, dirtyPlan windowDirtyPlan) (windowPrepaintPlan, bool) {
	if window == nil || window.canvas == nil || !window.dirtySet || !dirtyPlan.dirtySet {
		return windowPrepaintPlan{}, false
	}
	full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	plan := windowPrepaintPlan{
		mode:         windowPrepaintPartial,
		visualOnly:   dirtyPlan.hasDamage(windowDirtyDamageVisual),
		dirty:        dirtyPlan.dirty,
		contentDirty: dirtyPlan.dirty,
	}
	if dirtyPlan.hasDamage(windowDirtyDamageFull) || plan.dirty == full {
		plan.mode = windowPrepaintFull
		plan.dirty = full
		plan.contentDirty = full
		plan.scrollbarDirty = full
		return plan, true
	}
	if state.effect.simpleBackground {
		plan.clearMode = windowPrepaintClearSolid
		plan.clearColor = state.effect.backgroundColor
	} else if state.effect.backgroundCache != nil {
		plan.clearMode = windowPrepaintClearCache
		plan.backgroundCache = state.effect.backgroundCache
	} else if state.effect.needsFullRedraw {
		plan.mode = windowPrepaintFull
		plan.dirty = full
		return plan, true
	}
	if plan.visualOnly {
		plan.contentDirty, plan.scrollbarDirty, plan.scrollbarDirtySet = window.splitVisualScrollbarDirty(plan.dirty)
	}
	if dirtyPlan.mode == windowDirtyPlanNone &&
		dirtyPlan.hasDamage(windowDirtyDamageScroll) &&
		state.scroll.enabled &&
		window.canUseScrollBlit(state.scroll.viewport) &&
		dirtyPlan.dirty == window.scrollDirtyRectWithState(state) {
		plan.applyScrollBlit = true
		plan.contentDirty = scrollExposeRect(state.scroll.viewport, state.scroll.deltaY)
		if plan.contentDirty.Empty() {
			plan.contentDirty = state.scroll.viewport
		}
		if track, _, _, ok := window.windowScrollbarLayout(); ok {
			plan.scrollbarDirty = track
			plan.scrollbarDirtySet = true
		}
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
	return window.buildPrepaintPlanWithState(window.currentFramePropertyState(), dirtyPlan)
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
