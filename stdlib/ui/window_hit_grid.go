package ui

// WindowHitGridMinItems disables grid construction for small render lists where
// a linear reverse scan is cheaper than rebuilding the grid.
var WindowHitGridMinItems = 24

func (window *Window) invalidateHitGrid() {
	if window == nil {
		return
	}
	window.hitGridValid = false
}

func (window *Window) shouldUseHitGrid(display DisplayList) bool {
	if window == nil || !window.renderListValid {
		return false
	}
	if WindowHitGridMinItems > 0 && len(display.Items()) < WindowHitGridMinItems {
		return false
	}
	return len(display.Items()) > 0
}

func (window *Window) ensureHitGrid() {
	if window == nil {
		return
	}
	window.ensureHitGridWithDisplay(window.currentDisplayList())
}

func (window *Window) ensureHitGridWithDisplay(display DisplayList) bool {
	if window == nil {
		return false
	}
	if !window.shouldUseHitGrid(display) {
		window.hitGrid.reset()
		window.hitGridValid = false
		return false
	}
	if window.hitGridValid {
		return true
	}
	if !window.renderListValid || len(window.renderList) == 0 {
		window.hitGrid.reset()
		window.hitGridValid = false
		return false
	}
	window.hitGrid.build(window.client, display)
	window.hitGridValid = true
	return true
}
