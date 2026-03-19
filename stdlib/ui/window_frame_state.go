package ui

type windowFrameState struct {
	properties      windowPropertyState
	propertiesValid bool
	display         DisplayList
	displayValid    bool
	prepaint        windowPrepaintPlan
	prepaintValid   bool
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

func (window *Window) currentFramePropertyState() windowPropertyState {
	if window == nil {
		return windowPropertyState{}
	}
	if window.frameStateActive && window.frameState.propertiesValid {
		return window.frameState.properties
	}
	state := window.windowPropertyStateValue()
	if window.frameStateActive {
		window.frameState.properties = state
		window.frameState.propertiesValid = true
	}
	return state
}

func (window *Window) currentFrameScrollPropertyState() windowScrollPropertyState {
	if window == nil {
		return windowScrollPropertyState{}
	}
	if window.frameStateActive && window.frameState.propertiesValid {
		return window.frameState.properties.scroll
	}
	return window.computeScrollPropertyState(window.contentRect())
}

func (window *Window) currentFrameClipPropertyState() windowClipPropertyState {
	if window == nil {
		return windowClipPropertyState{}
	}
	if window.frameStateActive && window.frameState.propertiesValid {
		return window.frameState.properties.clip
	}
	return window.computeClipPropertyState(window.contentRect())
}

func (window *Window) currentFrameEffectPropertyState() windowEffectPropertyState {
	if window == nil {
		return windowEffectPropertyState{}
	}
	if window.frameStateActive && window.frameState.propertiesValid {
		return window.frameState.properties.effect
	}
	return window.computeEffectPropertyState()
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

func (window *Window) currentFramePrepaintPlan() (windowPrepaintPlan, bool) {
	if window == nil {
		return windowPrepaintPlan{}, false
	}
	if window.frameStateActive && window.frameState.prepaintValid {
		return window.frameState.prepaint, true
	}
	plan, ok := window.buildPrepaintPlanWithState(window.currentFramePropertyState())
	if ok && window.frameStateActive {
		window.frameState.prepaint = plan
		window.frameState.prepaintValid = true
	}
	return plan, ok
}
