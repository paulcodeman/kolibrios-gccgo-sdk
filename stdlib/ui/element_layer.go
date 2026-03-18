package ui

// ElementRetainedLayers enables retained subtree layers for large static box
// containers so redraw falls back to a single blit instead of replaying every
// descendant item.
var ElementRetainedLayers = true

const (
	elementRetainedLayerMinDescendants = 4
	elementRetainedLayerMinArea        = 16384
)

func (element *Element) invalidateRetainedLayerChain() {
	for current := element; current != nil; current = current.Parent {
		current.subtreeLayerValid = false
		current.subtreeLayerDirty = Rect{}
		current.subtreeLayerDirtySet = false
	}
}

func (element *Element) noteRetainedSubtreeDirty(rect Rect) {
	if element == nil || rect.Empty() {
		return
	}
	if element.subtreeLayerDirtySet {
		element.subtreeLayerDirty = UnionRect(element.subtreeLayerDirty, rect)
	} else {
		element.subtreeLayerDirty = rect
		element.subtreeLayerDirtySet = true
	}
}

func (element *Element) clearRetainedSubtreeDirty() {
	if element == nil {
		return
	}
	element.subtreeLayerDirty = Rect{}
	element.subtreeLayerDirtySet = false
}

func (element *Element) useRetainedSubtreeLayer(style Style) bool {
	if element == nil || !ElementRetainedLayers || FastNoCache {
		return false
	}
	if element.kind != ElementKindBox || len(element.Children) == 0 {
		return false
	}
	if attachment, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok && attachment == BackgroundAttachmentFixed {
		return false
	}
	visual := element.subtreeBounds()
	if visual.Empty() || visual.Width <= 0 || visual.Height <= 0 {
		return false
	}
	if visual.Width*visual.Height < elementRetainedLayerMinArea {
		return false
	}
	descendants, ok := element.retainedSubtreeDescendants()
	if !ok || descendants < elementRetainedLayerMinDescendants {
		return false
	}
	return true
}

func (element *Element) retainedSubtreeDescendants() (int, bool) {
	if element == nil {
		return 0, false
	}
	count := 0
	for _, node := range element.Children {
		child, ok := node.(*Element)
		if !ok || child == nil {
			return 0, false
		}
		switch child.kind {
		case ElementKindButton, ElementKindInput, ElementKindTextarea, ElementKindTinyGL:
			return 0, false
		}
		style := child.effectiveStyle()
		if attachment, ok := resolveBackgroundAttachment(style.backgroundAttachment); ok && attachment == BackgroundAttachmentFixed {
			return 0, false
		}
		count++
		subtreeCount, ok := child.retainedSubtreeDescendants()
		if !ok {
			return 0, false
		}
		count += subtreeCount
	}
	return count, true
}

func mergeElementLayerClip(bounds Rect, parent clipState, rect Rect, clipX bool, clipY bool) clipState {
	if !clipX && !clipY {
		return parent
	}
	base := bounds
	if parent.set {
		base = parent.rect
	}
	if clipX {
		base.X = rect.X
		base.Width = rect.Width
	}
	if clipY {
		base.Y = rect.Y
		base.Height = rect.Height
	}
	base = IntersectRect(base, bounds)
	if parent.set {
		base = IntersectRect(base, parent.rect)
	}
	return clipState{rect: base, set: true}
}

func (element *Element) drawRetainedSubtreeNode(canvas *Canvas, originX int, originY int, clip clipState) {
	if element == nil || canvas == nil || nodeHidden(element) {
		return
	}
	style := element.effectiveStyle()
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	if rect.Empty() {
		return
	}
	localRect := rect
	localRect.X -= originX
	localRect.Y -= originY
	pushed := false
	if clip.set {
		if clip.rect.Empty() {
			return
		}
		canvas.PushClip(clip.rect)
		pushed = true
	}
	if !element.tryDrawFromCache(canvas, localRect, style) {
		element.drawToRect(canvas, localRect, style)
	}
	childClip := clip
	if len(element.Children) > 0 {
		clipX, clipY := overflowClipAxes(style)
		if clipX || clipY {
			canvasBounds := Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()}
			childClip = mergeElementLayerClip(canvasBounds, clip, contentRectFor(localRect, style), clipX, clipY)
		}
		for _, node := range element.Children {
			child, ok := node.(*Element)
			if !ok || child == nil {
				continue
			}
			child.drawRetainedSubtreeNode(canvas, originX, originY, childClip)
		}
	}
	if pushed {
		canvas.PopClip()
	}
}

func (element *Element) ensureRetainedSubtreeLayer(style Style) (Rect, bool) {
	if element == nil || !element.useRetainedSubtreeLayer(style) {
		return Rect{}, false
	}
	visual := element.subtreeBounds()
	if visual.Empty() {
		return Rect{}, false
	}
	if element.subtreeLayer == nil || element.subtreeLayerWidth != visual.Width || element.subtreeLayerHeight != visual.Height {
		element.subtreeLayer = NewCanvasAlpha(visual.Width, visual.Height)
		element.subtreeLayerWidth = visual.Width
		element.subtreeLayerHeight = visual.Height
		element.subtreeLayerValid = false
		element.clearRetainedSubtreeDirty()
	}
	if element.subtreeLayer == nil {
		return Rect{}, false
	}
	if !element.subtreeLayerValid {
		element.redrawRetainedSubtreeLayer(style, visual)
	}
	return visual, true
}

func (element *Element) redrawRetainedSubtreeLayer(style Style, visual Rect) {
	if element == nil || element.subtreeLayer == nil || visual.Empty() {
		return
	}
	element.subtreeLayer.ClearTransparent()
	element.drawRetainedSubtreeNode(element.subtreeLayer, visual.X, visual.Y, clipState{})
	element.subtreeLayerValid = true
	element.clearRetainedSubtreeDirty()
}

func (element *Element) updateRetainedSubtreeLayer(style Style, visual Rect) bool {
	if element == nil || element.subtreeLayer == nil || !element.subtreeLayerValid || !element.subtreeLayerDirtySet {
		return false
	}
	dirty := IntersectRect(element.subtreeLayerDirty, visual)
	element.clearRetainedSubtreeDirty()
	if dirty.Empty() {
		return true
	}
	localDirty := Rect{
		X:      dirty.X - visual.X,
		Y:      dirty.Y - visual.Y,
		Width:  dirty.Width,
		Height: dirty.Height,
	}
	element.subtreeLayer.ClearRectTransparent(localDirty.X, localDirty.Y, localDirty.Width, localDirty.Height)
	element.drawRetainedSubtreeNode(element.subtreeLayer, visual.X, visual.Y, clipState{rect: localDirty, set: true})
	return true
}

func (element *Element) tryDrawFromRetainedSubtreeLayer(canvas *Canvas, style Style, offsetY int) bool {
	if element == nil || canvas == nil {
		return false
	}
	visual, ok := element.ensureRetainedSubtreeLayer(style)
	if !ok || element.subtreeLayer == nil {
		return false
	}
	if element.subtreeLayerDirtySet && element.subtreeLayerValid {
		if !element.updateRetainedSubtreeLayer(style, visual) {
			element.redrawRetainedSubtreeLayer(style, visual)
		}
	}
	if offsetY != 0 {
		visual.Y += offsetY
	}
	canvas.BlitFrom(element.subtreeLayer, Rect{X: 0, Y: 0, Width: element.subtreeLayerWidth, Height: element.subtreeLayerHeight}, visual.X, visual.Y)
	return true
}
