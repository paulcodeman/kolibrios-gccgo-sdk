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
	content Rect
	scroll  windowScrollPropertyState
	clip    windowClipPropertyState
	effect  windowEffectPropertyState
}

func (window *Window) invalidateWindowPropertyState() {
	if window == nil {
		return
	}
	window.propertyState = windowPropertyState{}
	window.propertyStateValid = false
}

func (window *Window) windowPropertyStateValue() windowPropertyState {
	if window == nil {
		return windowPropertyState{}
	}
	if window.propertyStateValid {
		return window.propertyState
	}
	state := window.computeWindowPropertyState()
	window.propertyState = state
	window.propertyStateValid = true
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
