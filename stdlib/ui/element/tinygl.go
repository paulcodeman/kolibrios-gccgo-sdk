package ui

import "kos"

// TinyGLRenderer is invoked after the UI canvas is blitted to the window.
// rect is the element content box in client coordinates.
type TinyGLRenderer func(gl *kos.TinyGL, ctx *kos.TinyGLContext, rect Rect)

type tinyGLState struct {
	renderer  TinyGLRenderer
	lib       kos.TinyGL
	libReady  bool
	libFailed bool
	ctx       kos.TinyGLContext
	lastRect  Rect
	dirty     bool
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
	rectChanged := windowRect != state.lastRect
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
	if state.libFailed {
		return
	}
	if !state.libReady {
		lib, ok := kos.LoadTinyGL()
		if !ok {
			state.libFailed = true
			return
		}
		state.lib = lib
		state.libReady = true
	}

	if !state.ctx.Initialized() {
		if !state.lib.MakeCurrent(windowRect.X, windowRect.Y, windowRect.Width, windowRect.Height, &state.ctx) {
			return
		}
	} else {
		if windowRect.Width != state.lastRect.Width || windowRect.Height != state.lastRect.Height {
			state.lib.Viewport(0, 0, windowRect.Width, windowRect.Height)
		}
		if windowRect.X != state.lastRect.X || windowRect.Y != state.lastRect.Y {
			state.ctx.SetPosition(windowRect.X, windowRect.Y)
		}
	}

	state.lastRect = windowRect
	state.renderer(&state.lib, &state.ctx, content)
	state.lib.SwapBuffers()
	state.dirty = false
}
