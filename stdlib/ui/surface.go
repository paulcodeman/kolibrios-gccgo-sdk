package ui

import "kos"

type Surface interface {
	Bounds() Rect
	Canvas() *Canvas
}

type Presenter interface {
	PresentFull(*Canvas)
	PresentRect(*Canvas, Rect)
}

type windowSurface struct {
	window *Window
}

func (surface windowSurface) Bounds() Rect {
	if surface.window == nil {
		return Rect{}
	}
	return surface.window.client
}

func (surface windowSurface) Canvas() *Canvas {
	if surface.window == nil {
		return nil
	}
	return surface.window.canvas
}

func (window *Window) surface() Surface {
	if window == nil {
		return nil
	}
	return windowSurface{window: window}
}

type windowPresenter struct {
	window *Window
}

func (window *Window) presenter() Presenter {
	if window == nil {
		return nil
	}
	return windowPresenter{window: window}
}

func (presenter windowPresenter) PresentFull(canvas *Canvas) {
	window := presenter.window
	if window == nil {
		return
	}
	kos.BeginRedraw()
	kos.OpenWindow(window.X, window.Y, window.Width, window.Height, window.Title)
	if canvas != nil {
		canvas.BlitToWindow(window.client.X, window.client.Y)
	}
	kos.EndRedraw()
}

func (presenter windowPresenter) PresentRect(canvas *Canvas, rect Rect) {
	window := presenter.window
	if window == nil || canvas == nil || rect.Empty() {
		return
	}
	canvas.BlitRectToWindow(rect, window.client.X+rect.X, window.client.Y+rect.Y)
}
