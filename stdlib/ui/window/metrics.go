package ui

import "kos"

func (window *Window) threadInfo() (kos.ThreadInfo, bool) {
	if window == nil {
		return kos.ThreadInfo{}, false
	}
	window.syncThreadSlot(false)
	if window.threadSlotSet {
		info, _, ok := kos.ReadThreadInfo(window.threadSlot)
		return info, ok
	}
	info, _, ok := kos.ReadCurrentThreadInfo()
	return info, ok
}

// SystemRect returns the window position and size reported by KolibriOS.
func (window *Window) SystemRect() (Rect, bool) {
	info, ok := window.threadInfo()
	if !ok {
		return Rect{}, false
	}
	if info.WindowSize.X <= 0 || info.WindowSize.Y <= 0 {
		return Rect{}, false
	}
	return Rect{
		X:      info.WindowPosition.X,
		Y:      info.WindowPosition.Y,
		Width:  info.WindowSize.X,
		Height: info.WindowSize.Y,
	}, true
}

func (window *Window) SystemLeft() (int, bool) {
	rect, ok := window.SystemRect()
	if !ok {
		return 0, false
	}
	return rect.X, true
}

func (window *Window) SystemTop() (int, bool) {
	rect, ok := window.SystemRect()
	if !ok {
		return 0, false
	}
	return rect.Y, true
}

func (window *Window) SystemWidth() (int, bool) {
	rect, ok := window.SystemRect()
	if !ok {
		return 0, false
	}
	return rect.Width, true
}

func (window *Window) SystemHeight() (int, bool) {
	rect, ok := window.SystemRect()
	if !ok {
		return 0, false
	}
	return rect.Height, true
}

// ScreenMargins returns distances from window edges to the screen edges.
// Left/Top are the window position; Right/Bottom are distances to the edges.
func (window *Window) ScreenMargins() (left int, top int, right int, bottom int, ok bool) {
	rect, ok := window.SystemRect()
	if !ok {
		return 0, 0, 0, 0, false
	}
	screenW, screenH := kos.ScreenSize()
	if screenW <= 0 || screenH <= 0 {
		return 0, 0, 0, 0, false
	}
	left = rect.X
	top = rect.Y
	right = screenW - rect.X - rect.Width
	bottom = screenH - rect.Y - rect.Height
	return left, top, right, bottom, true
}

func (window *Window) ScreenRight() (int, bool) {
	_, _, right, _, ok := window.ScreenMargins()
	return right, ok
}

func (window *Window) ScreenBottom() (int, bool) {
	_, _, _, bottom, ok := window.ScreenMargins()
	return bottom, ok
}
