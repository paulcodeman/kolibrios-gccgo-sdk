package ui

import (
	"kos"
	surfacetinygl "surface/tinygl"
)

// TinyGLRenderer is invoked after the UI canvas is blitted to the window.
// rect is the element content box in client coordinates.
type TinyGLRenderer func(gl *kos.TinyGL, ctx *kos.TinyGLContext, rect Rect)

type tinyGLState struct {
	renderer TinyGLRenderer
	layer    surfacetinygl.Layer
	dirty    bool
}

// SetTinyGLRenderer wires a TinyGL renderer to a tinygl element.
func (element *Element) SetTinyGLRenderer(renderer TinyGLRenderer) bool {
	if element == nil || !element.isTinyGL() {
		return false
	}
	state := element.tinyGLState()
	state.renderer = renderer
	state.dirty = true
	element.markDirty()
	return true
}

// MarkTinyGLDirty requests a TinyGL redraw for this element.
func (element *Element) MarkTinyGLDirty() bool {
	if element == nil || !element.isTinyGL() {
		return false
	}
	state := element.tinyGLState()
	if state == nil {
		return false
	}
	state.dirty = true
	element.markDirty()
	return true
}

func (element *Element) tinyGLState() *tinyGLState {
	if element == nil {
		return nil
	}
	if element.tinygl == nil {
		element.tinygl = &tinyGLState{}
	}
	return element.tinygl
}

func (element *Element) drawTinyGL(window *Window, full bool, dirty Rect) {
	if element == nil || window == nil || !element.isTinyGL() {
		return
	}
	state := element.tinygl
	if state == nil || state.renderer == nil {
		return
	}
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	style := element.effectiveStyle()
	content := contentRectFor(rect, style)
	if content.Empty() || content.Width <= 0 || content.Height <= 0 {
		return
	}
	if window.scrollEnabled() && window.scrollY != 0 {
		content.Y -= window.scrollY
	}
	windowRect := Rect{
		X:      window.client.X + content.X,
		Y:      window.client.Y + content.Y,
		Width:  content.Width,
		Height: content.Height,
	}
	rectChanged := windowRect != state.layer.Rect()
	if !full {
		if WindowTinyGLRedrawOnDirty {
			if !state.dirty && !rectChanged && IntersectRect(content, dirty).Empty() {
				return
			}
		} else {
			if !state.dirty && !rectChanged {
				return
			}
		}
	}
	if !state.layer.Render(windowRect, func(gl *kos.TinyGL, ctx *kos.TinyGLContext) {
		state.renderer(gl, ctx, content)
	}) {
		return
	}
	state.dirty = false
}
