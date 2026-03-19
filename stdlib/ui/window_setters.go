package ui

import "kos"

func (window *Window) SetLeft(x int) bool {
	if window == nil {
		return false
	}
	return window.setPosition(x, window.Y)
}

func (window *Window) SetTop(y int) bool {
	if window == nil {
		return false
	}
	return window.setPosition(window.X, y)
}

func (window *Window) SetPosition(x int, y int) bool {
	if window == nil {
		return false
	}
	return window.setPosition(x, y)
}

func (window *Window) SetWidth(width int) bool {
	if window == nil {
		return false
	}
	window.Style.SetWidth(width)
	return window.setSize(width, window.Height)
}

func (window *Window) SetHeight(height int) bool {
	if window == nil {
		return false
	}
	window.Style.SetHeight(height)
	return window.setSize(window.Width, height)
}

func (window *Window) SetSize(width int, height int) bool {
	if window == nil {
		return false
	}
	window.Style.SetWidth(width)
	window.Style.SetHeight(height)
	return window.setSize(width, height)
}

func (window *Window) SetBounds(x int, y int, width int, height int) bool {
	if window == nil {
		return false
	}
	window.Style.SetLeft(x)
	window.Style.SetTop(y)
	window.Style.SetWidth(width)
	window.Style.SetHeight(height)
	changed := window.setPosition(x, y)
	if window.setSize(width, height) {
		changed = true
	}
	return changed
}

func (window *Window) SetTitle(title string) bool {
	if window == nil {
		return false
	}
	if window.Title == title {
		return false
	}
	window.Title = title
	return true
}

// SetRight positions the window so its right edge is "right" pixels from the screen edge.
func (window *Window) SetRight(right int) bool {
	if window == nil {
		return false
	}
	screenW, _ := kos.ScreenSize()
	if screenW <= 0 {
		return false
	}
	width := window.Width
	if width < 1 {
		width = 1
	}
	x := screenW - width - right
	if x < 0 {
		x = 0
	}
	return window.SetLeft(x)
}

// SetBottom positions the window so its bottom edge is "bottom" pixels from the screen edge.
func (window *Window) SetBottom(bottom int) bool {
	if window == nil {
		return false
	}
	_, screenH := kos.ScreenSize()
	if screenH <= 0 {
		return false
	}
	height := window.Height
	if height < 1 {
		height = 1
	}
	y := screenH - height - bottom
	if y < 0 {
		y = 0
	}
	return window.SetTop(y)
}

func (window *Window) setSize(width int, height int) bool {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	changed := window.Width != width || window.Height != height
	window.Style.SetWidth(width)
	window.Style.SetHeight(height)
	if !changed {
		return false
	}
	window.Width = width
	window.Height = height
	updated := windowClientRect(width, height)
	if updated != window.client {
		window.client = updated
		window.invalidateWindowPropertyState()
		window.layoutDirty = true
		window.renderListValid = false
		window.hoverDirty = true
		window.lastMouseValid = false
		if window.OnResize != nil {
			window.OnResize(window.client)
		}
	}
	return true
}

func (window *Window) setPosition(x int, y int) bool {
	if window == nil {
		return false
	}
	changed := window.X != x || window.Y != y
	if changed {
		window.X = x
		window.Y = y
	}
	window.Style.SetLeft(x)
	window.Style.SetTop(y)
	return changed
}
