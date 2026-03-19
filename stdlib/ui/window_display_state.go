package ui

type windowDisplayState struct {
	items             []renderItem
	itemsValid        bool
	rootClip          clipState
	rootClipValid     bool
	scrollOffset      int
	scrollOffsetValid bool
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

func (window *Window) invalidateWindowDisplayItemsState() {
	if window == nil {
		return
	}
	window.displayState.items = nil
	window.displayState.itemsValid = false
	if window.frameStateActive {
		window.frameState.display = DisplayList{}
		window.frameState.displayValid = false
	}
}

func (window *Window) invalidateWindowDisplayClipState() {
	if window == nil {
		return
	}
	window.displayState.rootClip = clipState{}
	window.displayState.rootClipValid = false
	if window.frameStateActive {
		window.frameState.display = DisplayList{}
		window.frameState.displayValid = false
	}
}

func (window *Window) invalidateWindowDisplayScrollState() {
	if window == nil {
		return
	}
	window.displayState.scrollOffset = 0
	window.displayState.scrollOffsetValid = false
	if window.frameStateActive {
		window.frameState.display = DisplayList{}
		window.frameState.displayValid = false
	}
}

func (window *Window) windowDisplayItemsValue() []renderItem {
	if window == nil {
		return nil
	}
	if window.displayState.itemsValid {
		return window.displayState.items
	}
	window.displayState.items = window.renderList
	window.displayState.itemsValid = true
	return window.displayState.items
}

func (window *Window) windowDisplayRootClipValue() clipState {
	if window == nil {
		return clipState{}
	}
	if window.displayState.rootClipValid {
		return window.displayState.rootClip
	}
	window.displayState.rootClip = window.rootClipState()
	window.displayState.rootClipValid = true
	return window.displayState.rootClip
}

func (window *Window) windowDisplayScrollOffsetValue() int {
	if window == nil {
		return 0
	}
	if window.displayState.scrollOffsetValid {
		return window.displayState.scrollOffset
	}
	window.displayState.scrollOffset = window.scrollPaintOffset()
	window.displayState.scrollOffsetValid = true
	return window.displayState.scrollOffset
}

func (window *Window) computeCurrentDisplayList() DisplayList {
	if window == nil {
		return DisplayList{}
	}
	return DisplayList{
		items:        window.windowDisplayItemsValue(),
		rootClip:     window.windowDisplayRootClipValue(),
		scrollOffset: window.windowDisplayScrollOffsetValue(),
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
	return window.computeCurrentDisplayList()
}
