package ui

import "kos"

type windowScrollPropertyState struct {
	enabled    bool
	viewport   Rect
	offsetY    int
	drawnY     int
	deltaY     int
	scrollMaxY int
}

type windowClipPropertyState struct {
	clipX bool
	clipY bool
	root  clipState
}

type windowEffectPropertyState struct {
	simpleBackground bool
	backgroundColor  kos.Color
	backgroundCache  *Canvas
	needsFullRedraw  bool
}

type windowPropertyState struct {
	content      Rect
	contentValid bool
	scroll       windowScrollPropertyState
	scrollValid  bool
	clip         windowClipPropertyState
	clipValid    bool
	effect       windowEffectPropertyState
	effectValid  bool
}

func (window *Window) invalidateWindowPropertyState() {
	if window == nil {
		return
	}
	window.invalidateWindowContentPropertyState()
	window.invalidateWindowClipPropertyState()
	window.invalidateWindowScrollPropertyState()
	window.invalidateWindowEffectPropertyState()
	window.invalidateWindowDisplayState()
}

func (window *Window) invalidateWindowContentPropertyState() {
	if window == nil {
		return
	}
	window.propertyState.content = Rect{}
	window.propertyState.contentValid = false
	window.propertyState.scroll = windowScrollPropertyState{}
	window.propertyState.scrollValid = false
	window.propertyState.clip = windowClipPropertyState{}
	window.propertyState.clipValid = false
	if window.frameStateActive {
		window.frameState.properties.content = Rect{}
		window.frameState.properties.contentValid = false
		window.frameState.properties.scroll = windowScrollPropertyState{}
		window.frameState.properties.scrollValid = false
		window.frameState.properties.clip = windowClipPropertyState{}
		window.frameState.properties.clipValid = false
		window.frameState.prepaint = windowPrepaintPlan{}
		window.frameState.prepaintValid = false
	}
	window.invalidateWindowDisplayState()
}

func (window *Window) invalidateWindowScrollPropertyState() {
	if window == nil {
		return
	}
	window.propertyState.scroll = windowScrollPropertyState{}
	window.propertyState.scrollValid = false
	if window.frameStateActive {
		window.frameState.properties.scroll = windowScrollPropertyState{}
		window.frameState.properties.scrollValid = false
		window.frameState.prepaint = windowPrepaintPlan{}
		window.frameState.prepaintValid = false
	}
	window.invalidateWindowDisplayState()
}

func (window *Window) invalidateWindowClipPropertyState() {
	if window == nil {
		return
	}
	window.propertyState.clip = windowClipPropertyState{}
	window.propertyState.clipValid = false
	if window.frameStateActive {
		window.frameState.properties.clip = windowClipPropertyState{}
		window.frameState.properties.clipValid = false
	}
	window.invalidateWindowDisplayState()
}

func (window *Window) invalidateWindowEffectPropertyState() {
	if window == nil {
		return
	}
	window.propertyState.effect = windowEffectPropertyState{}
	window.propertyState.effectValid = false
	if window.frameStateActive {
		window.frameState.properties.effect = windowEffectPropertyState{}
		window.frameState.properties.effectValid = false
		window.frameState.prepaint = windowPrepaintPlan{}
		window.frameState.prepaintValid = false
	}
}

func (window *Window) windowContentPropertyStateValue() Rect {
	if window == nil {
		return Rect{}
	}
	if window.propertyState.contentValid {
		return window.propertyState.content
	}
	content := window.contentRect()
	window.propertyState.content = content
	window.propertyState.contentValid = true
	return content
}

func (window *Window) windowScrollPropertyStateValue() windowScrollPropertyState {
	if window == nil {
		return windowScrollPropertyState{}
	}
	if window.propertyState.scrollValid {
		return window.propertyState.scroll
	}
	content := window.windowContentPropertyStateValue()
	state := window.computeScrollPropertyState(content)
	window.propertyState.scroll = state
	window.propertyState.scrollValid = true
	return state
}

func (window *Window) windowClipPropertyStateValue() windowClipPropertyState {
	if window == nil {
		return windowClipPropertyState{}
	}
	if window.propertyState.clipValid {
		return window.propertyState.clip
	}
	content := window.windowContentPropertyStateValue()
	state := window.computeClipPropertyState(content)
	window.propertyState.clip = state
	window.propertyState.clipValid = true
	return state
}

func (window *Window) windowEffectPropertyStateValue() windowEffectPropertyState {
	if window == nil {
		return windowEffectPropertyState{needsFullRedraw: true}
	}
	if window.propertyState.effectValid {
		return window.propertyState.effect
	}
	state := window.computeEffectPropertyState()
	window.propertyState.effect = state
	window.propertyState.effectValid = true
	return state
}

func (window *Window) windowPropertyStateValue() windowPropertyState {
	if window == nil {
		return windowPropertyState{}
	}
	state := window.propertyState
	if !state.contentValid {
		state.content = window.contentRect()
		state.contentValid = true
	}
	if !state.scrollValid {
		state.scroll = window.computeScrollPropertyState(state.content)
		state.scrollValid = true
	}
	if !state.clipValid {
		state.clip = window.computeClipPropertyState(state.content)
		state.clipValid = true
	}
	if !state.effectValid {
		state.effect = window.computeEffectPropertyState()
		state.effectValid = true
	}
	window.propertyState = state
	return state
}

func (window *Window) computeScrollPropertyState(content Rect) windowScrollPropertyState {
	state := windowScrollPropertyState{
		enabled: window != nil && window.scrollEnabled(),
		offsetY: window.scrollY,
		drawnY:  window.drawnScrollY,
		deltaY:  window.scrollY - window.drawnScrollY,
	}
	if !state.enabled {
		return state
	}
	state.viewport = content
	state.scrollMaxY = window.scrollMaxY
	return state
}

func (window *Window) computeClipPropertyState(content Rect) windowClipPropertyState {
	if window == nil || window.canvas == nil {
		return windowClipPropertyState{}
	}
	clipX, clipY := overflowClipAxes(window.Style)
	if !clipY && window.scrollEnabled() {
		clipY = true
	}
	state := windowClipPropertyState{
		clipX: clipX,
		clipY: clipY,
	}
	if !clipX && !clipY {
		return state
	}
	if content.Empty() {
		state.root = clipState{rect: Rect{}, set: true}
		return state
	}
	canvasBounds := Rect{X: 0, Y: 0, Width: window.canvas.Width(), Height: window.canvas.Height()}
	base := canvasBounds
	if clipX {
		base.X = content.X
		base.Width = content.Width
	}
	if clipY {
		base.Y = content.Y
		base.Height = content.Height
	}
	base = IntersectRect(base, canvasBounds)
	state.root = clipState{rect: base, set: true}
	return state
}

func (window *Window) computeEffectPropertyState() windowEffectPropertyState {
	if window == nil {
		return windowEffectPropertyState{needsFullRedraw: true}
	}
	if color, ok := window.simpleBackgroundColor(); ok {
		return windowEffectPropertyState{
			simpleBackground: true,
			backgroundColor:  color,
		}
	}
	if cache := window.ensureBackgroundCache(); cache != nil {
		return windowEffectPropertyState{
			backgroundCache: cache,
		}
	}
	return windowEffectPropertyState{
		needsFullRedraw: true,
	}
}

func (window *Window) computeWindowPropertyState() windowPropertyState {
	if window == nil {
		return windowPropertyState{}
	}
	content := window.contentRect()
	return windowPropertyState{
		content: content,
		scroll:  window.computeScrollPropertyState(content),
		clip:    window.computeClipPropertyState(content),
		effect:  window.computeEffectPropertyState(),
	}
}
