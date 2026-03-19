package ui

type windowDisplayState struct {
	list  DisplayList
	valid bool
}

func (window *Window) invalidateWindowDisplayState() {
	if window == nil {
		return
	}
	window.displayState = windowDisplayState{}
	if window.frameStateActive {
		window.frameState.display = DisplayList{}
		window.frameState.displayValid = false
	}
}

func (window *Window) computeCurrentDisplayList() DisplayList {
	if window == nil {
		return DisplayList{}
	}
	return DisplayList{
		items:        window.renderList,
		rootClip:     window.rootClipState(),
		scrollOffset: window.scrollPaintOffset(),
	}
}

func (window *Window) currentDisplayList() DisplayList {
	if window == nil {
		return DisplayList{}
	}
	if window.frameStateActive {
		if window.frameState.displayValid {
			return window.frameState.display
		}
		list := window.computeCurrentDisplayList()
		window.frameState.display = list
		window.frameState.displayValid = true
		return list
	}
	if window.displayState.valid {
		return window.displayState.list
	}
	list := window.computeCurrentDisplayList()
	window.displayState.list = list
	window.displayState.valid = true
	return list
}
