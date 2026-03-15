package ui

func (element *Element) SetText(window *Window, text string) {
	if element == nil || element.Text == text {
		return
	}
	targetWindow := window
	if targetWindow == nil {
		targetWindow = element.window
	}
	oldStyle := element.effectiveStyle()
	oldRect := element.resolveRect(nil, oldStyle)
	if targetWindow != nil {
		oldRect = element.resolveRect(targetWindow.canvas, oldStyle)
	}
	oldVisual := element.visualBoundsFor(oldRect, oldStyle)
	element.Text = text
	element.clearTextCache()
	if element.isTextInput() {
		if element.caret > len(text) {
			element.caret = len(text)
		}
		element.desiredCol = -1
		element.selectAnchor = element.caret
	}
	element.markDirtyIn(targetWindow)
	if targetWindow == nil {
		return
	}
	newStyle := element.effectiveStyle()
	newRect := element.resolveRect(targetWindow.canvas, newStyle)
	newVisual := element.visualBoundsFor(newRect, newStyle)
	dirty := oldVisual
	if !newVisual.Empty() {
		dirty = UnionRect(oldVisual, newVisual)
	}
	targetWindow.InvalidateContent(dirty)
}

func (element *Element) SetLabel(window *Window, label string) {
	if element == nil || element.Label == label {
		return
	}
	targetWindow := window
	if targetWindow == nil {
		targetWindow = element.window
	}
	oldStyle := element.effectiveStyle()
	oldRect := element.resolveRect(nil, oldStyle)
	if targetWindow != nil {
		oldRect = element.resolveRect(targetWindow.canvas, oldStyle)
	}
	oldVisual := element.visualBoundsFor(oldRect, oldStyle)
	element.Label = label
	element.clearTextCache()
	element.markDirtyIn(targetWindow)
	if targetWindow == nil {
		return
	}
	newStyle := element.effectiveStyle()
	newRect := element.resolveRect(targetWindow.canvas, newStyle)
	newVisual := element.visualBoundsFor(newRect, newStyle)
	dirty := oldVisual
	if !newVisual.Empty() {
		dirty = UnionRect(oldVisual, newVisual)
	}
	targetWindow.InvalidateContent(dirty)
}

func (element *Element) SetStyle(window *Window, style Style) {
	if element == nil {
		return
	}
	targetWindow := window
	if targetWindow == nil {
		targetWindow = element.window
	}
	oldStyle := element.effectiveStyle()
	oldRect := element.resolveRect(nil, oldStyle)
	if targetWindow != nil {
		oldRect = element.resolveRect(targetWindow.canvas, oldStyle)
	}
	oldVisual := element.visualBoundsFor(oldRect, oldStyle)
	element.Style = style
	element.markDirtyIn(targetWindow)
	if targetWindow == nil {
		return
	}
	newStyle := element.effectiveStyle()
	newRect := element.resolveRect(targetWindow.canvas, newStyle)
	newVisual := element.visualBoundsFor(newRect, newStyle)
	dirty := oldVisual
	if !newVisual.Empty() {
		dirty = UnionRect(oldVisual, newVisual)
	}
	targetWindow.InvalidateContent(dirty)
}

func (element *Element) Append(child Node) {
	if element == nil || child == nil {
		return
	}
	if node, ok := child.(*Element); ok {
		node.Parent = element
		node.setWindow(element.window)
	}
	element.Children = append(element.Children, child)
	if element.window != nil {
		element.window.layoutDirty = true
		element.window.renderListValid = false
		element.window.hoverDirty = true
		element.window.lastMouseValid = false
	}
}

func (element *Element) Remove(child Node) bool {
	if element == nil || child == nil {
		return false
	}
	for i, node := range element.Children {
		if node == child {
			if el, ok := node.(*Element); ok && el.Parent == element {
				el.Parent = nil
				el.setWindow(nil)
			}
			element.Children = append(element.Children[:i], element.Children[i+1:]...)
			if element.window != nil {
				element.window.layoutDirty = true
				element.window.renderListValid = false
				element.window.hoverDirty = true
				element.window.lastMouseValid = false
			}
			return true
		}
	}
	return false
}

func (element *Element) ClearChildren() {
	if element == nil {
		return
	}
	for _, node := range element.Children {
		if el, ok := node.(*Element); ok && el.Parent == element {
			el.Parent = nil
			el.setWindow(nil)
		}
	}
	element.Children = nil
	if element.window != nil {
		element.window.layoutDirty = true
		element.window.renderListValid = false
		element.window.hoverDirty = true
		element.window.lastMouseValid = false
	}
}
