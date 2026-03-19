package ui

func (window *Window) invalidateHitGrid() {
	if window == nil {
		return
	}
	window.hitGridValid = false
}

func (window *Window) ensureHitGrid() {
	if window == nil {
		return
	}
	if window.hitGridValid {
		return
	}
	if !window.renderListValid || len(window.renderList) == 0 {
		window.hitGrid.reset()
		window.hitGridValid = false
		return
	}
	window.hitGrid.build(window.client, window.currentDisplayList())
	window.hitGridValid = true
}
