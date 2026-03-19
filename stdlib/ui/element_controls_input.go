package ui

import "kos"

func (element *Element) rangeValueFromPoint(x int, rect Rect, style Style) int {
	if element == nil || !element.isRange() {
		return 0
	}
	track := element.rangeTrackRect(rect, style)
	if track.Width <= 0 {
		return element.minValue
	}
	if x <= track.X {
		return element.minValue
	}
	if x >= track.X+track.Width {
		return element.maxValue
	}
	offset := x - track.X
	span := element.controlValueSpan()
	if span <= 0 {
		return element.minValue
	}
	value := element.minValue + offset*span/track.Width
	step := element.StepValue()
	if step > 1 {
		delta := value - element.minValue
		value = element.minValue + ((delta + step/2) / step * step)
	}
	return element.clampControlValue(value)
}

func (element *Element) handleRangeMouseDown(x int, y int) bool {
	if element == nil || !element.isRange() {
		return false
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	thumb := element.rangeThumbRect(rect, style)
	content := contentRectFor(rect, style)
	if !content.Contains(x, y) {
		return false
	}
	element.rangeDragActive = true
	changed := element.SetValue(element.rangeValueFromPoint(x, rect, style))
	if changed {
		element.dispatchInputEvent()
		element.dispatchChange()
	}
	return changed || thumb.Contains(x, y)
}

func (element *Element) handleRangeMouseDrag(x int, y int) bool {
	if element == nil || !element.isRange() || !element.rangeDragActive {
		return false
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	changed := element.SetValue(element.rangeValueFromPoint(x, rect, style))
	if changed {
		element.dispatchInputEvent()
		element.dispatchChange()
	}
	return changed
}

func (element *Element) handleRangeMouseUp() bool {
	if element == nil || !element.isRange() {
		return false
	}
	if !element.rangeDragActive {
		return false
	}
	element.rangeDragActive = false
	return false
}

func (element *Element) dispatchClickEvent(event Event) bool {
	return dispatchElementHandler(element.OnClick, element, event)
}

func (element *Element) handleControlClick(event Event) bool {
	if element == nil {
		return false
	}
	switch {
	case element.isCheckable():
		if element.ToggleChecked() {
			element.dispatchInputEvent()
			element.dispatchChange()
			return true
		}
	case element.isRange():
		style := element.effectiveStyle()
		rect := element.layoutRect
		if rect.Empty() {
			rect = element.Bounds()
		}
		if element.SetValue(element.rangeValueFromPoint(event.X, rect, style)) {
			element.dispatchInputEvent()
			element.dispatchChange()
			return true
		}
	}
	return false
}

func (element *Element) handleControlKey(key kos.KeyEvent) bool {
	if element == nil || !element.focused {
		return false
	}
	if key.Empty || key.Hotkey {
		return false
	}
	control := kos.ControlKeysStatus()
	if control.Ctrl() || control.Alt() {
		return false
	}
	switch {
	case element.isCheckable():
		if key.Code == 13 || key.Code == 32 {
			changed := element.ToggleChecked()
			if changed {
				element.dispatchInputEvent()
				element.dispatchChange()
				element.dispatchClickEvent(Event{Type: EventClick, Target: element})
			}
			return changed
		}
	case element.isRange():
		step := element.StepValue()
		pageStep := element.controlValueSpan() / 10
		if pageStep < step {
			pageStep = step
		}
		switch {
		case key.ScanCode == 0x4B:
			if element.SetValue(element.value - step) {
				element.dispatchInputEvent()
				element.dispatchChange()
				return true
			}
			return false
		case key.ScanCode == 0x4D:
			if element.SetValue(element.value + step) {
				element.dispatchInputEvent()
				element.dispatchChange()
				return true
			}
			return false
		case key.ScanCode == 0x47:
			if element.SetValue(element.minValue) {
				element.dispatchInputEvent()
				element.dispatchChange()
				return true
			}
			return false
		case key.ScanCode == 0x4F:
			if element.SetValue(element.maxValue) {
				element.dispatchInputEvent()
				element.dispatchChange()
				return true
			}
			return false
		case key.ScanCode == 0x49:
			if element.SetValue(element.value + pageStep) {
				element.dispatchInputEvent()
				element.dispatchChange()
				return true
			}
			return false
		case key.ScanCode == 0x51:
			if element.SetValue(element.value - pageStep) {
				element.dispatchInputEvent()
				element.dispatchChange()
				return true
			}
			return false
		}
	}
	return false
}
