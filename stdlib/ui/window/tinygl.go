package ui

func (window *Window) drawTinyGL(full bool, dirty Rect) {
	if window == nil {
		return
	}
	if !WindowEnableTinyGL {
		return
	}
	window.ensureRenderList()
	if len(window.tinyglNodes) == 0 {
		return
	}
	for _, element := range window.tinyglNodes {
		if element == nil || !element.isTinyGL() {
			continue
		}
		element.drawTinyGL(window, full, dirty)
	}
}
