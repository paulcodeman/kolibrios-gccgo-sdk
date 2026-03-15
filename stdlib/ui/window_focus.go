package ui

import "kos"

func (window *Window) setFocus(target Node) bool {
	if window == nil {
		return false
	}
	if element, ok := target.(*Element); ok && element != nil && !element.isFocusable() {
		target = nil
	}
	if window.focused == target {
		return false
	}
	prev := window.focused
	needsRedraw := false
	if window.focused != nil {
		if aware, ok := window.focused.(FocusAware); ok {
			if aware.SetFocus(false) {
				needsRedraw = true
				window.noteDirty(window.focused)
			}
		}
	}
	window.focused = nil
	window.caretBlinkResetAt = 0
	if target != nil {
		if aware, ok := target.(FocusAware); ok {
			if aware.SetFocus(true) {
				needsRedraw = true
				window.noteDirty(target)
			}
			window.focused = target
		}
	}
	if element, ok := window.focused.(*Element); ok && element.isTextInput() {
		window.caretBlinkResetAt = kos.UptimeCentiseconds()
	}
	if window.invalidateFocusNode(prev) {
		needsRedraw = true
	}
	if window.invalidateFocusNode(window.focused) {
		needsRedraw = true
	}
	return needsRedraw
}

func (window *Window) invalidateFocusNode(node Node) bool {
	element, ok := node.(*Element)
	if !ok || element == nil || !element.isTextInput() {
		return false
	}
	rect := element.VisualBounds()
	if rect.Empty() {
		rect = element.Bounds()
	}
	if rect.Empty() {
		return false
	}
	window.InvalidateContent(rect)
	return true
}

func (window *Window) focusNext() bool {
	if window == nil {
		return false
	}
	var focusables []Node
	window.collectFocusables(window.nodes, &focusables)
	if len(focusables) == 0 {
		return false
	}
	if window.focused == nil {
		return window.setFocus(focusables[0])
	}
	index := -1
	for i, node := range focusables {
		if node == window.focused {
			index = i
			break
		}
	}
	if index < 0 {
		return window.setFocus(focusables[0])
	}
	next := focusables[(index+1)%len(focusables)]
	return window.setFocus(next)
}

func (window *Window) focusPrev() bool {
	if window == nil {
		return false
	}
	var focusables []Node
	window.collectFocusables(window.nodes, &focusables)
	if len(focusables) == 0 {
		return false
	}
	if window.focused == nil {
		return window.setFocus(focusables[len(focusables)-1])
	}
	index := -1
	for i, node := range focusables {
		if node == window.focused {
			index = i
			break
		}
	}
	if index < 0 {
		return window.setFocus(focusables[len(focusables)-1])
	}
	prevIndex := index - 1
	if prevIndex < 0 {
		prevIndex = len(focusables) - 1
	}
	return window.setFocus(focusables[prevIndex])
}

func (window *Window) collectFocusables(nodes []Node, out *[]Node) {
	if window == nil {
		return
	}
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if nodeHidden(node) {
			continue
		}
		if element, ok := node.(*Element); ok {
			if element.isFocusable() {
				*out = append(*out, node)
			}
			if len(element.Children) > 0 {
				window.collectFocusables(element.Children, out)
			}
			continue
		}
		if _, ok := node.(FocusAware); ok {
			*out = append(*out, node)
		}
	}
}
