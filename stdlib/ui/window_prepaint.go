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
	dirty              Rect
	clearMode          windowPrepaintClearMode
	clearColor         kos.Color
	backgroundCache    *Canvas
	applyScrollBlit    bool
	applyTranslateBlit bool
}

func (window *Window) buildPrepaintPlan() (windowPrepaintPlan, bool) {
	if window == nil || window.canvas == nil || !window.dirtySet {
		return windowPrepaintPlan{}, false
	}
	state := window.windowPropertyStateValue()
	full := Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height}
	plan := windowPrepaintPlan{
		mode:  windowPrepaintPartial,
		dirty: window.dirty,
	}
	if plan.dirty == full {
		plan.mode = windowPrepaintFull
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
	if state.scroll.enabled && window.canUseScrollBlit(state.scroll.viewport) {
		plan.applyScrollBlit = true
	}
	if len(window.translateBlits) != 0 {
		plan.applyTranslateBlit = true
	}
	return plan, true
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
		window.canvas.FillRect(plan.dirty.X, plan.dirty.Y, plan.dirty.Width, plan.dirty.Height, plan.clearColor)
	case windowPrepaintClearCache:
		if plan.backgroundCache != nil {
			window.canvas.BlitFrom(plan.backgroundCache, plan.dirty, plan.dirty.X, plan.dirty.Y)
		}
	}
}
