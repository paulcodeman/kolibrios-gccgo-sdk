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
				window.noteFocusStateChange(window.focused)
			}
		}
	}
	window.focused = nil
	window.caretBlinkResetAt = 0
	if target != nil {
		if aware, ok := target.(FocusAware); ok {
			if aware.SetFocus(true) {
				needsRedraw = true
				window.noteFocusStateChange(target)
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
	if window.scrollNodeIntoView(window.focused) {
		needsRedraw = true
	}
	return needsRedraw
}

func (window *Window) focusStateNeedsDirtyNode(node Node) bool {
	if window == nil || node == nil {
		return false
	}
	if _, ok := node.(*DocumentView); ok {
		return false
	}
	element, ok := node.(*Element)
	if !ok || element == nil {
		return true
	}
	if element.isTextInput() {
		return true
	}
	if !elementUsesDefaultFocusRing(element) {
		return true
	}
	return false
}

func (window *Window) noteFocusStateChange(node Node) {
	if window == nil || node == nil {
		return
	}
	if window.focusStateNeedsDirtyNode(node) {
		window.noteDirty(node)
	}
}

func (window *Window) scrollNodeIntoView(node Node) bool {
	if window == nil || node == nil || !window.scrollEnabled() {
		return false
	}
	content := window.contentRect()
	if content.Empty() || content.Height <= 0 {
		return false
	}
	bounds := window.focusScrollBounds(node)
	if bounds.Empty() {
		return false
	}
	next := scrollRevealNearest(window.scrollY, content.Height, bounds.Y-content.Y, bounds.Height)
	maxScroll := window.scrollMaxY
	required := bounds.Y + bounds.Height - (content.Y + content.Height)
	if required > maxScroll {
		maxScroll = required
	}
	if next < 0 {
		next = 0
	}
	if maxScroll > 0 && next > maxScroll {
		next = maxScroll
	}
	if next == window.scrollY {
		return false
	}
	if maxScroll > window.scrollMaxY {
		window.scrollMaxY = maxScroll
	}
	window.scrollY = next
	window.noteScrollChanged()
	return true
}

func (window *Window) focusScrollBounds(node Node) Rect {
	if window == nil || node == nil {
		return Rect{}
	}
	switch current := node.(type) {
	case *Element:
		if current == nil {
			return Rect{}
		}
		return window.nodeVisualBoundsFor(current, true)
	case *DocumentView:
		if current == nil {
			return Rect{}
		}
		rect := current.layoutRect
		if rect.Empty() {
			rect = current.Bounds()
		}
		if rect.Empty() {
			return Rect{}
		}
		return visualBoundsForStyle(rect, current.effectiveStyle(), false)
	default:
		if visual, ok := node.(VisualBoundsAware); ok {
			return visual.VisualBounds()
		}
		if window.nodeBounds != nil {
			if bounds, ok := window.nodeBounds[node]; ok {
				return bounds
			}
		}
		return node.Bounds()
	}
}

func (window *Window) invalidateFocusNode(node Node) bool {
	if window == nil || node == nil {
		return false
	}
	switch current := node.(type) {
	case *Element:
		if current == nil {
			return false
		}
		rect := Rect{}
		switch {
		case current.isTextInput():
			rect = current.VisualBounds()
			if rect.Empty() {
				rect = current.Bounds()
			}
			if elementUsesDefaultFocusRing(current) {
				base := current.layoutRect
				if base.Empty() {
					base = current.Bounds()
				}
				rect = UnionRect(rect, focusRingBounds(base))
			}
		case elementUsesDefaultFocusRing(current):
			rect = current.layoutRect
			if rect.Empty() {
				rect = current.Bounds()
			}
			rect = focusRingBounds(rect)
		default:
			return false
		}
		if rect.Empty() {
			return false
		}
		window.InvalidateVisualContent(rect)
		return true
	default:
		return false
	}
}

func (window *Window) focusNext() bool {
	if window == nil {
		return false
	}
	focusables := window.focusableNodes()
	if len(focusables) == 0 {
		return false
	}
	if window.focused == nil {
		return window.setFocus(focusables[0])
	}
	index := window.focusableIndex(window.focused)
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
	focusables := window.focusableNodes()
	if len(focusables) == 0 {
		return false
	}
	if window.focused == nil {
		return window.setFocus(focusables[len(focusables)-1])
	}
	index := window.focusableIndex(window.focused)
	if index < 0 {
		return window.setFocus(focusables[len(focusables)-1])
	}
	prevIndex := index - 1
	if prevIndex < 0 {
		prevIndex = len(focusables) - 1
	}
	return window.setFocus(focusables[prevIndex])
}

func (window *Window) focusableNodes() []Node {
	if window == nil {
		return nil
	}
	window.ensureRenderList()
	if len(window.renderList) > 0 {
		return window.focusables
	}
	var focusables []Node
	window.collectFocusables(window.nodes, &focusables, false)
	return focusables
}

func (window *Window) focusableIndex(node Node) int {
	if window == nil || node == nil {
		return -1
	}
	window.ensureRenderList()
	if len(window.renderList) > 0 && window.focusIndex != nil {
		if index, ok := window.focusIndex[node]; ok {
			return index
		}
	}
	return -1
}

func (window *Window) appendFocusableFromRenderNode(node Node) {
	if window == nil || node == nil {
		return
	}
	switch current := node.(type) {
	case *Element:
		if current == nil || !current.isFocusable() {
			return
		}
	case FocusAware:
		// kept
	default:
		return
	}
	if _, ok := window.focusIndex[node]; ok {
		return
	}
	window.focusIndex[node] = len(window.focusables)
	window.focusables = append(window.focusables, node)
}

func (window *Window) collectFocusables(nodes []Node, out *[]Node, inheritedHidden bool) {
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
		currentHidden := inheritedHidden
		switch current := node.(type) {
		case *Element:
			if current != nil {
				currentHidden = styleHiddenByVisibility(current.effectiveStyle(), inheritedHidden)
			}
		case *DocumentView:
			if current != nil {
				currentHidden = styleHiddenByVisibility(current.effectiveStyle(), inheritedHidden)
			}
		}
		if element, ok := node.(*Element); ok {
			if !currentHidden && element.isFocusable() {
				*out = append(*out, node)
			}
			if len(element.Children) > 0 {
				window.collectFocusables(element.Children, out, currentHidden)
			}
			continue
		}
		if !currentHidden {
			if _, ok := node.(FocusAware); ok {
				*out = append(*out, node)
			}
		}
	}
}
