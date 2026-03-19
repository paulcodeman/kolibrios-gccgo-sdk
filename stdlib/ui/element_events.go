package ui

func dispatchElementHandler(handler interface{}, current *Element, event *Event) bool {
	if handler == nil || current == nil {
		return false
	}
	if event != nil {
		if event.Target == nil {
			event.Target = current
		}
		if event.CurrentTarget == nil {
			event.CurrentTarget = current
		}
		if event.Phase == EventPhaseNone {
			event.Phase = EventPhaseTarget
		}
	}
	switch typed := handler.(type) {
	case func():
		typed()
		return true
	case func(*Element):
		typed(current)
		return true
	case func(Event):
		if event == nil {
			typed(Event{})
		} else {
			typed(*event)
		}
		return true
	case func(*Event):
		typed(event)
		return true
	case func(*Element, Event):
		if event == nil {
			typed(current, Event{})
		} else {
			typed(current, *event)
		}
		return true
	case func(*Element, *Event):
		typed(current, event)
		return true
	default:
		return false
	}
}

func dispatchElementValueHandler(handler interface{}, current *Element, target *Element, event *Event) bool {
	if dispatchElementHandler(handler, current, event) {
		return true
	}
	if handler == nil {
		return false
	}
	valueTarget := target
	if valueTarget == nil {
		valueTarget = current
	}
	if valueTarget == nil {
		return false
	}
	switch typed := handler.(type) {
	case func(string):
		typed(valueTarget.text())
		return true
	case func(bool):
		typed(valueTarget.checked)
		return true
	case func(int):
		typed(valueTarget.value)
		return true
	case func(*Element, string):
		typed(current, valueTarget.text())
		return true
	case func(*Element, bool):
		typed(current, valueTarget.checked)
		return true
	case func(*Element, int):
		typed(current, valueTarget.value)
		return true
	default:
		return false
	}
}

func elementEventPath(element *Element) []*Element {
	if element == nil {
		return nil
	}
	path := make([]*Element, 0, 4)
	for current := element; current != nil; current = current.Parent {
		path = append(path, current)
	}
	return path
}

func eventTargetElement(event *Event) *Element {
	if event == nil {
		return nil
	}
	target, _ := event.Target.(*Element)
	return target
}

func dispatchElementEvent(event *Event, path []*Element, handler func(*Element) interface{}) bool {
	if event == nil || len(path) == 0 || handler == nil {
		return false
	}
	handled := false
	for index, current := range path {
		if current == nil {
			continue
		}
		if index > 0 && !event.Bubbles {
			break
		}
		event.CurrentTarget = current
		if index == 0 {
			event.Phase = EventPhaseTarget
		} else {
			event.Phase = EventPhaseBubble
		}
		if dispatchElementHandler(handler(current), current, event) {
			handled = true
		}
		if event.PropagationStopped() {
			break
		}
	}
	return handled
}

func dispatchElementValueEvent(event *Event, path []*Element, handler func(*Element) interface{}) bool {
	if event == nil || len(path) == 0 || handler == nil {
		return false
	}
	target := eventTargetElement(event)
	if target == nil {
		target = path[0]
	}
	handled := false
	for index, current := range path {
		if current == nil {
			continue
		}
		if index > 0 && !event.Bubbles {
			break
		}
		event.CurrentTarget = current
		if index == 0 {
			event.Phase = EventPhaseTarget
		} else {
			event.Phase = EventPhaseBubble
		}
		if dispatchElementValueHandler(handler(current), current, target, event) {
			handled = true
		}
		if event.PropagationStopped() {
			break
		}
	}
	return handled
}

func (element *Element) dispatchInputEvent() bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:       EventInput,
		Target:     element,
		Bubbles:    true,
		Cancelable: false,
	}
	return dispatchElementValueEvent(event, elementEventPath(element), func(current *Element) interface{} {
		return current.OnInput
	})
}

func (element *Element) dispatchChangeEvent() bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:       EventChange,
		Target:     element,
		Bubbles:    true,
		Cancelable: false,
	}
	return dispatchElementValueEvent(event, elementEventPath(element), func(current *Element) interface{} {
		return current.OnChange
	})
}

func (element *Element) dispatchMouseEnterEvent(x int, y int) bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:          EventMouseEnter,
		Target:        element,
		CurrentTarget: element,
		Phase:         EventPhaseTarget,
		X:             x,
		Y:             y,
		Cancelable:    false,
	}
	return dispatchElementHandler(element.OnMouseEnter, element, event)
}

func (element *Element) dispatchMouseLeaveEvent(x int, y int) bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:          EventMouseLeave,
		Target:        element,
		CurrentTarget: element,
		Phase:         EventPhaseTarget,
		X:             x,
		Y:             y,
		Cancelable:    false,
	}
	return dispatchElementHandler(element.OnMouseLeave, element, event)
}

func (element *Element) HandleMouseMove(x int, y int) bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:       EventMouseMove,
		X:          x,
		Y:          y,
		Target:     element,
		Bubbles:    true,
		Cancelable: true,
	}
	handled := dispatchElementEvent(event, elementEventPath(element), func(current *Element) interface{} {
		return current.OnMouseMove
	})
	if event.DefaultPrevented() {
		return handled
	}
	if element.isTextInput() {
		if element.handleTextMouseDrag(x, y) {
			handled = true
		}
	} else if element.isRange() {
		if element.handleRangeMouseDrag(x, y) {
			handled = true
		}
	}
	return handled
}

func (element *Element) HandleMouseDown(x int, y int, button MouseButton) bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:       EventMouseDown,
		X:          x,
		Y:          y,
		Button:     button,
		Target:     element,
		Bubbles:    true,
		Cancelable: true,
	}
	handled := dispatchElementEvent(event, elementEventPath(element), func(current *Element) interface{} {
		return current.OnMouseDown
	})
	if event.DefaultPrevented() {
		return handled
	}
	if element.isTextInput() {
		if element.handleTextMouseDown(x, y) {
			handled = true
		}
	} else if element.isRange() {
		if element.handleRangeMouseDown(x, y) {
			handled = true
		}
	}
	return handled
}

func (element *Element) HandleMouseUp(x int, y int, button MouseButton) bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:       EventMouseUp,
		X:          x,
		Y:          y,
		Button:     button,
		Target:     element,
		Bubbles:    true,
		Cancelable: true,
	}
	handled := dispatchElementEvent(event, elementEventPath(element), func(current *Element) interface{} {
		return current.OnMouseUp
	})
	if event.DefaultPrevented() {
		return handled
	}
	if element.isTextInput() {
		if element.handleTextMouseUp() {
			handled = true
		}
	} else if element.isRange() {
		if element.handleRangeMouseUp() {
			handled = true
		}
	}
	return handled
}
