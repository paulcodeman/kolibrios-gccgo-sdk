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

func elementHandlerForType(current *Element, eventType EventType) interface{} {
	if current == nil {
		return nil
	}
	switch eventType {
	case EventClick:
		return current.OnClick
	case EventMouseDown:
		return current.OnMouseDown
	case EventMouseUp:
		return current.OnMouseUp
	case EventMouseMove:
		return current.OnMouseMove
	case EventMouseEnter:
		return current.OnMouseEnter
	case EventMouseLeave:
		return current.OnMouseLeave
	case EventScroll:
		return current.OnScroll
	case EventFocus:
		return current.OnFocus
	case EventBlur:
		return current.OnBlur
	case EventFocusIn:
		return current.OnFocusIn
	case EventFocusOut:
		return current.OnFocusOut
	case EventKeyDown:
		return current.OnKeyDown
	case EventInput:
		return current.OnInput
	case EventChange:
		return current.OnChange
	default:
		return nil
	}
}

func dispatchElementCaptureEvent(event *Event, path []*Element) bool {
	if event == nil || len(path) < 2 {
		return false
	}
	handled := false
	for index := len(path) - 1; index >= 1; index-- {
		current := path[index]
		if current == nil {
			continue
		}
		event.CurrentTarget = current
		event.Phase = EventPhaseCapture
		if dispatchElementHandler(current.OnEventCapture, current, event) {
			handled = true
		}
		if event.PropagationStopped() {
			break
		}
	}
	return handled
}

func dispatchElementEventOnCurrent(current *Element, event *Event) bool {
	if current == nil || event == nil {
		return false
	}
	handled := false
	if dispatchElementHandler(elementHandlerForType(current, event.Type), current, event) {
		handled = true
	}
	if dispatchElementHandler(current.OnEvent, current, event) {
		handled = true
	}
	return handled
}

func dispatchElementValueEventOnCurrent(current *Element, target *Element, event *Event) bool {
	if current == nil || event == nil {
		return false
	}
	handled := false
	if dispatchElementValueHandler(elementHandlerForType(current, event.Type), current, target, event) {
		handled = true
	}
	if dispatchElementHandler(current.OnEvent, current, event) {
		handled = true
	}
	return handled
}

func dispatchElementEvent(event *Event, path []*Element, handler func(*Element) interface{}) bool {
	if event == nil || len(path) == 0 || handler == nil {
		return false
	}
	handled := dispatchElementCaptureEvent(event, path)
	if event.PropagationStopped() {
		return handled
	}
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
		if dispatchElementEventOnCurrent(current, event) {
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
	handled := dispatchElementCaptureEvent(event, path)
	if event.PropagationStopped() {
		return handled
	}
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
		if dispatchElementValueEventOnCurrent(current, target, event) {
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

func (element *Element) dispatchFocusTransitionEvent(eventType EventType, bubbles bool) bool {
	if element == nil {
		return false
	}
	event := &Event{
		Type:       eventType,
		Target:     element,
		Bubbles:    bubbles,
		Cancelable: false,
	}
	return dispatchElementEvent(event, elementEventPath(element), func(current *Element) interface{} {
		return elementHandlerForType(current, eventType)
	})
}

func (element *Element) dispatchTargetOnlyEvent(eventType EventType) bool {
	return element.dispatchFocusTransitionEvent(eventType, false)
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
	return dispatchElementEventOnCurrent(element, event)
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
	return dispatchElementEventOnCurrent(element, event)
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
