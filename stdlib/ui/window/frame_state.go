package ui

type windowFrameState struct {
	properties    windowPropertyState
	dirty         windowDirtyPlan
	dirtyValid    bool
	display       DisplayList
	displayValid  bool
	prepaint      windowPrepaintPlan
	prepaintValid bool
}

func (window *Window) beginWindowFrameState() {
	if window == nil {
		return
	}
	window.frameState = windowFrameState{}
	window.frameStateActive = true
}

func (window *Window) endWindowFrameState() {
	if window == nil {
		return
	}
	window.frameState = windowFrameState{}
	window.frameStateActive = false
}

func (window *Window) currentFrameScrollPropertyState() windowScrollPropertyState {
	if window == nil {
		return windowScrollPropertyState{}
	}
	if !window.frameStateActive {
		return window.windowScrollPropertyStateValue()
	}
	if window.frameState.properties.scrollValid {
		return window.frameState.properties.scroll
	}
	state := window.windowScrollPropertyStateValue()
	window.frameState.properties.scroll = state
	window.frameState.properties.scrollValid = true
	return state
}

func (window *Window) currentFrameClipPropertyState() windowClipPropertyState {
	if window == nil {
		return windowClipPropertyState{}
	}
	if !window.frameStateActive {
		return window.windowClipPropertyStateValue()
	}
	if window.frameState.properties.clipValid {
		return window.frameState.properties.clip
	}
	state := window.windowClipPropertyStateValue()
	window.frameState.properties.clip = state
	window.frameState.properties.clipValid = true
	return state
}

func (window *Window) currentFrameEffectPropertyState() windowEffectPropertyState {
	if window == nil {
		return windowEffectPropertyState{}
	}
	if !window.frameStateActive {
		return window.windowEffectPropertyStateValue()
	}
	if window.frameState.properties.effectValid {
		return window.frameState.properties.effect
	}
	state := window.windowEffectPropertyStateValue()
	window.frameState.properties.effect = state
	window.frameState.properties.effectValid = true
	return state
}

func (window *Window) currentFrameScrollPaintOffset() int {
	if window == nil {
		return 0
	}
	state := window.currentFrameScrollPropertyState()
	if !state.enabled || state.offsetY == 0 {
		return 0
	}
	return -state.offsetY
}

func (window *Window) noteFrameDirtyPlan(plan windowDirtyPlan) {
	if window == nil || !window.frameStateActive {
		return
	}
	window.frameState.dirty = plan
	window.frameState.dirtyValid = true
	window.frameState.prepaint = windowPrepaintPlan{}
	window.frameState.prepaintValid = false
}

func (window *Window) currentFrameDirtyPlan() (windowDirtyPlan, bool) {
	if window == nil {
		return windowDirtyPlan{}, false
	}
	if window.frameStateActive && window.frameState.dirtyValid {
		return window.frameState.dirty, true
	}
	return windowDirtyPlan{}, false
}

func (window *Window) currentFramePrepaintPlan() (windowPrepaintPlan, bool) {
	if window == nil {
		return windowPrepaintPlan{}, false
	}
	if window.frameStateActive && window.frameState.prepaintValid {
		return window.frameState.prepaint, true
	}
	dirtyPlan, ok := window.currentFrameDirtyPlan()
	if !ok {
		return windowPrepaintPlan{}, false
	}
	plan, ok := window.buildPrepaintPlanWithState(window.currentFrameScrollPropertyState(), window.currentFrameEffectPropertyState(), dirtyPlan)
	if ok && window.frameStateActive {
		window.frameState.prepaint = plan
		window.frameState.prepaintValid = true
	}
	return plan, ok
}
