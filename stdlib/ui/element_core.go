package ui

func newElement(kind ElementKind, text string) *Element {
	return &Element{
		kind:       kind,
		Text:       text,
		desiredCol: -1,
	}
}

// Kind returns the element kind.
func (element *Element) Kind() ElementKind {
	if element == nil {
		return ElementKindUnknown
	}
	return element.kind
}

func (element *Element) setWindow(window *Window) {
	if element == nil {
		return
	}
	element.window = window
	element.invalidateEffectiveStyleCache()
	element.invalidateBoundsCache()
	element.subtreeLayerValid = false
	element.renderVisitGen = 0
	element.layoutVisitGen = 0
	element.dirtyQueueGen = 0
	for _, child := range element.Children {
		if aware, ok := child.(windowAware); ok && aware != nil {
			aware.setWindow(window)
		}
	}
}

func (element *Element) markDirty() {
	element.markDirtyIn(nil)
}

func (element *Element) markDirtyIn(window *Window) {
	if element == nil {
		return
	}
	element.dirty = true
	target := window
	if target == nil {
		target = element.window
	}
	if target == nil {
		return
	}
	target.noteDirty(element)
	if element.layoutDirtyInCurrentContainer() {
		element.invalidateBoundsCache()
		target.layoutDirty = true
		target.renderListValid = false
	}
}

// MarkDirty requests a redraw for the element.
func (element *Element) MarkDirty() {
	if element == nil {
		return
	}
	element.markDirty()
}

func (element *Element) Invalidate(window *Window) {
	if element == nil || window == nil {
		return
	}
	style := element.effectiveStyle()
	oldRect := element.layoutRect
	oldVisual := element.visualBoundsFor(oldRect, style)
	rect := element.resolveRect(window.canvas, style)
	if oldVisual.Empty() {
		oldVisual = element.visualBoundsFor(rect, style)
	}
	newVisual := element.visualBoundsFor(rect, style)
	dirty := oldVisual
	if !newVisual.Empty() {
		dirty = UnionRect(oldVisual, newVisual)
	}
	window.InvalidateContent(dirty)
}

func (element *Element) Dirty() bool {
	if element == nil {
		return false
	}
	style := element.effectiveStyle()
	element.updateRenderKey(style)
	return element.dirty
}

func (element *Element) ClearDirty() {
	if element == nil {
		return
	}
	element.dirty = false
}

func (element *Element) invalidateEffectiveStyleCache() {
	if element == nil {
		return
	}
	element.effectiveStyleCache = Style{}
	element.effectiveStyleValid = false
}

func (element *Element) invalidateBoundsCache() {
	if element == nil {
		return
	}
	element.invalidateTextInputLayoutCache()
	element.visualRect = Rect{}
	element.visualRectValid = false
	for current := element; current != nil; current = current.Parent {
		current.subtreeRect = Rect{}
		current.subtreeRectValid = false
	}
}
