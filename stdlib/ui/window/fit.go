package ui

import "kos"

// CenterOnScreen centers the window on the screen.
func (window *Window) CenterOnScreen() bool {
	if window == nil {
		return false
	}
	screenW, screenH := kos.ScreenSize()
	if screenW <= 0 || screenH <= 0 {
		return false
	}
	width := window.Width
	height := window.Height
	if rect, ok := window.SystemRect(); ok && rect.Width > 0 && rect.Height > 0 {
		width = rect.Width
		height = rect.Height
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	newX := (screenW - width) / 2
	newY := (screenH - height) / 2
	if newX < 0 {
		newX = 0
	}
	if newY < 0 {
		newY = 0
	}
	if newX == window.X && newY == window.Y {
		return false
	}
	return window.setPosition(newX, newY)
}
