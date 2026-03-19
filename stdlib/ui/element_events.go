package ui

func dispatchElementHandler(handler interface{}, element *Element, event Event) bool {
	if handler == nil || element == nil {
		return false
	}
	switch current := handler.(type) {
	case func():
		current()
		return true
	case func(*Element):
		current(element)
		return true
	case func(Event):
		current(event)
		return true
	case func(*Element, Event):
		current(element, event)
		return true
	default:
		return false
	}
}

func dispatchElementValueHandler(handler interface{}, element *Element, event Event) bool {
	if dispatchElementHandler(handler, element, event) {
		return true
	}
	if handler == nil || element == nil {
		return false
	}
	switch current := handler.(type) {
	case func(string):
		current(element.text())
		return true
	case func(bool):
		current(element.checked)
		return true
	case func(int):
		current(element.value)
		return true
	default:
		return false
	}
}

func (element *Element) dispatchInputEvent() bool {
	if element == nil {
		return false
	}
	return dispatchElementValueHandler(element.OnInput, element, Event{
		Type:   EventInput,
		Target: element,
	})
}

func (element *Element) dispatchChangeEvent() bool {
	if element == nil {
		return false
	}
	return dispatchElementValueHandler(element.OnChange, element, Event{
		Type:   EventChange,
		Target: element,
	})
}

func (element *Element) HandleMouseMove(x int, y int) bool {
	if element == nil {
		return false
	}
	handled := false
	if element.isTextInput() {
		if element.handleTextMouseDrag(x, y) {
			handled = true
		}
	} else if element.isRange() {
		if element.handleRangeMouseDrag(x, y) {
			handled = true
		}
	}
	if dispatchElementHandler(element.OnMouseMove, element, Event{
		Type:   EventMouseMove,
		X:      x,
		Y:      y,
		Target: element,
	}) {
		handled = true
	}
	return handled
}

func (element *Element) HandleMouseDown(x int, y int, button MouseButton) bool {
	if element == nil {
		return false
	}
	handled := false
	if element.isTextInput() {
		if element.handleTextMouseDown(x, y) {
			handled = true
		}
	} else if element.isRange() {
		if element.handleRangeMouseDown(x, y) {
			handled = true
		}
	}
	if dispatchElementHandler(element.OnMouseDown, element, Event{
		Type:   EventMouseDown,
		X:      x,
		Y:      y,
		Button: button,
		Target: element,
	}) {
		handled = true
	}
	return handled
}

func (element *Element) HandleMouseUp(x int, y int, button MouseButton) bool {
	if element == nil {
		return false
	}
	handled := false
	if element.isTextInput() {
		if element.handleTextMouseUp() {
			handled = true
		}
	} else if element.isRange() {
		if element.handleRangeMouseUp() {
			handled = true
		}
	}
	if dispatchElementHandler(element.OnMouseUp, element, Event{
		Type:   EventMouseUp,
		X:      x,
		Y:      y,
		Button: button,
		Target: element,
	}) {
		handled = true
	}
	return handled
}
