package ui

func (element *Element) controlValueSpan() int {
	if element == nil {
		return 0
	}
	span := element.maxValue - element.minValue
	if span < 0 {
		span = 0
	}
	return span
}

func (element *Element) clampControlValue(value int) int {
	if element == nil {
		return value
	}
	if element.maxValue < element.minValue {
		element.maxValue = element.minValue
	}
	if value < element.minValue {
		value = element.minValue
	}
	if value > element.maxValue {
		value = element.maxValue
	}
	return value
}

func (element *Element) dispatchChange() {
	if element == nil {
		return
	}
	element.dispatchChangeEvent()
}

func (element *Element) sameRadioGroup(other *Element) bool {
	if element == nil || other == nil || element == other || !element.isRadio() || !other.isRadio() {
		return false
	}
	if element.Parent != other.Parent {
		return false
	}
	return element.controlGroup == other.controlGroup
}

func (element *Element) setCheckedInternal(checked bool, notify bool) bool {
	if element == nil || !element.isCheckable() {
		return false
	}
	if element.checked == checked {
		return false
	}
	element.checked = checked
	element.markDirty()
	if notify {
		element.dispatchChange()
	}
	return true
}

func (element *Element) SetChecked(checked bool) bool {
	if element == nil || !element.isCheckable() {
		return false
	}
	changed := false
	if element.isRadio() && checked {
		if parent := element.Parent; parent != nil {
			for _, node := range parent.Children {
				peer, ok := node.(*Element)
				if !ok || !element.sameRadioGroup(peer) {
					continue
				}
				if peer.setCheckedInternal(false, false) {
					changed = true
				}
			}
		}
	}
	if element.setCheckedInternal(checked, false) {
		changed = true
	}
	return changed
}

func (element *Element) Checked() bool {
	return element != nil && element.checked
}

func (element *Element) ToggleChecked() bool {
	if element == nil || !element.isCheckable() {
		return false
	}
	if element.isRadio() {
		return element.SetChecked(true)
	}
	return element.SetChecked(!element.checked)
}

func (element *Element) SetGroup(group string) bool {
	if element == nil || !element.isRadio() {
		return false
	}
	if element.controlGroup == group {
		return false
	}
	element.controlGroup = group
	element.markDirty()
	return true
}

func (element *Element) Group() string {
	if element == nil {
		return ""
	}
	return element.controlGroup
}

func (element *Element) SetRangeBounds(min int, max int) bool {
	if element == nil || (!element.isProgress() && !element.isRange()) {
		return false
	}
	if max < min {
		max = min
	}
	changed := false
	if element.minValue != min {
		element.minValue = min
		changed = true
	}
	if element.maxValue != max {
		element.maxValue = max
		changed = true
	}
	clamped := element.clampControlValue(element.value)
	if clamped != element.value {
		element.value = clamped
		changed = true
	}
	if changed {
		element.markDirty()
	}
	return changed
}

func (element *Element) SetStepValue(step int) bool {
	if element == nil || !element.isRange() {
		return false
	}
	if step <= 0 {
		step = 1
	}
	if element.stepValue == step {
		return false
	}
	element.stepValue = step
	element.markDirty()
	return true
}

func (element *Element) SetValue(value int) bool {
	if element == nil || (!element.isProgress() && !element.isRange()) {
		return false
	}
	value = element.clampControlValue(value)
	if element.value == value {
		return false
	}
	element.value = value
	element.markDirty()
	return true
}

func (element *Element) Value() int {
	if element == nil {
		return 0
	}
	return element.value
}

func (element *Element) MinValue() int {
	if element == nil {
		return 0
	}
	return element.minValue
}

func (element *Element) MaxValue() int {
	if element == nil {
		return 0
	}
	return element.maxValue
}

func (element *Element) StepValue() int {
	if element == nil {
		return 0
	}
	if element.stepValue <= 0 {
		return 1
	}
	return element.stepValue
}

func (element *Element) ValueFraction() float64 {
	if element == nil {
		return 0
	}
	span := element.controlValueSpan()
	if span <= 0 {
		return 0
	}
	return float64(element.value-element.minValue) / float64(span)
}
