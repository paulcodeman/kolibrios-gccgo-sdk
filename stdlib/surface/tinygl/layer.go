package tinygl

import (
	"kos"
	"surface"
)

type Layer struct {
	lib       kos.TinyGL
	libReady  bool
	libFailed bool
	ctx       kos.TinyGLContext
	rect      surface.Rect
}

func (layer *Layer) Rect() surface.Rect {
	if layer == nil {
		return surface.Rect{}
	}
	return layer.rect
}

func (layer *Layer) Render(rect surface.Rect, draw func(gl *kos.TinyGL, ctx *kos.TinyGLContext)) bool {
	if layer == nil || draw == nil || rect.Empty() || rect.Width <= 0 || rect.Height <= 0 {
		return false
	}
	if layer.libFailed {
		return false
	}
	if !layer.libReady {
		lib, ok := kos.LoadTinyGL()
		if !ok {
			layer.libFailed = true
			return false
		}
		layer.lib = lib
		layer.libReady = true
	}

	if !layer.ctx.Initialized() {
		if !layer.lib.MakeCurrent(rect.X, rect.Y, rect.Width, rect.Height, &layer.ctx) {
			return false
		}
	} else {
		if rect.Width != layer.rect.Width || rect.Height != layer.rect.Height {
			layer.lib.Viewport(0, 0, rect.Width, rect.Height)
		}
		if rect.X != layer.rect.X || rect.Y != layer.rect.Y {
			layer.ctx.SetPosition(rect.X, rect.Y)
		}
	}

	layer.rect = rect
	draw(&layer.lib, &layer.ctx)
	layer.lib.SwapBuffers()
	return true
}
