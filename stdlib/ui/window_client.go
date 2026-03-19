package ui

import "kos"

func (window *Window) syncWindowInfo() {
	if window == nil {
		return
	}

	info, _, ok := kos.ReadCurrentThreadInfo()
	if ok && info.WindowSize.X > 0 && info.WindowSize.Y > 0 {
		if window.Width != info.WindowSize.X || window.Height != info.WindowSize.Y {
			window.Width = info.WindowSize.X
			window.Height = info.WindowSize.Y
			window.Style.SetWidth(window.Width)
			window.Style.SetHeight(window.Height)
		}
	}
	if window.Style.width == nil {
		window.Style.SetWidth(window.Width)
	}
	if window.Style.height == nil {
		window.Style.SetHeight(window.Height)
	}

	updated := windowClientRect(window.Width, window.Height)
	if updated != window.client {
		window.client = updated
		window.invalidateWindowContentPropertyState()
		window.invalidateWindowEffectPropertyState()
		window.layoutDirty = true
		window.renderListValid = false
		window.invalidateHoverTracking()
		if window.OnResize != nil {
			window.OnResize(window.client)
		}
	}
}

func (window *Window) ensureCanvas() {
	if window == nil {
		return
	}
	if window.client.Empty() {
		window.canvas = nil
		return
	}
	if window.canvas == nil {
		window.canvas = NewCanvas(window.client.Width, window.client.Height)
		window.invalidateWindowClipPropertyState()
		window.invalidateWindowEffectPropertyState()
		window.renderListValid = false
		window.Invalidate(Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height})
		return
	}
	if window.canvas.Width() != window.client.Width || window.canvas.Height() != window.client.Height {
		window.canvas.Resize(window.client.Width, window.client.Height)
		window.invalidateWindowClipPropertyState()
		window.invalidateWindowEffectPropertyState()
		window.renderListValid = false
		window.Invalidate(Rect{X: 0, Y: 0, Width: window.client.Width, Height: window.client.Height})
	}
}

func windowClientRect(width int, height int) Rect {
	skin := kos.SkinHeight()
	if skin < 0 {
		skin = 0
	}
	x := windowClientLeft
	y := skin
	w := width - windowClientLeft - windowClientRight
	h := height - skin - windowClientBottom
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x, Y: y, Width: w, Height: h}
}
