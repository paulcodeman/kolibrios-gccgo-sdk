package ui

type translateBlitOp struct {
	src Rect
	dst Rect
}

func (element *Element) canUseTranslateBlit(style Style) bool {
	if element == nil {
		return false
	}
	if !element.canUseDirtyClip(style) {
		return false
	}
	if element.isFocusable() || element.OnClick != nil {
		return false
	}
	if !element.StyleHover.IsZero() || !element.StyleActive.IsZero() || !element.StyleFocus.IsZero() {
		return false
	}
	if element.hovered || element.active || element.focused {
		return false
	}
	return true
}

func rectContainsRect(outer Rect, inner Rect) bool {
	if outer.Empty() || inner.Empty() {
		return false
	}
	return inner.X >= outer.X &&
		inner.Y >= outer.Y &&
		inner.X+inner.Width <= outer.X+outer.Width &&
		inner.Y+inner.Height <= outer.Y+outer.Height
}

func translateExposeRect(src Rect, dst Rect) Rect {
	if src.Empty() || dst.Empty() || src.Width != dst.Width || src.Height != dst.Height {
		return Rect{}
	}
	dx := dst.X - src.X
	dy := dst.Y - src.Y
	switch {
	case dx > 0 && dy == 0:
		return Rect{X: src.X, Y: src.Y, Width: dx, Height: src.Height}
	case dx < 0 && dy == 0:
		return Rect{X: src.X + src.Width + dx, Y: src.Y, Width: -dx, Height: src.Height}
	case dy > 0 && dx == 0:
		return Rect{X: src.X, Y: src.Y, Width: src.Width, Height: dy}
	case dy < 0 && dx == 0:
		return Rect{X: src.X, Y: src.Y + src.Height + dy, Width: src.Width, Height: -dy}
	default:
		return Rect{}
	}
}

func (window *Window) resetTranslateBlits() {
	if window == nil || window.translateBlits == nil {
		return
	}
	window.translateBlits = window.translateBlits[:0]
}

func (window *Window) translateBlitIntersectsExisting(src Rect, dst Rect) bool {
	if window == nil {
		return false
	}
	union := UnionRect(src, dst)
	for _, op := range window.translateBlits {
		if !IntersectRect(union, UnionRect(op.src, op.dst)).Empty() {
			return true
		}
	}
	return false
}

func (window *Window) translateBlitBlockedByLaterItems(index int, dst Rect, scrollOffset int) bool {
	if window == nil {
		return true
	}
	for i := index + 1; i < len(window.renderList); i++ {
		paint := window.renderList[i].paint
		if scrollOffset != 0 {
			paint.Y += scrollOffset
		}
		if !IntersectRect(paint, dst).Empty() {
			return true
		}
	}
	return false
}

func (window *Window) oldRenderKeyFor(node Node, oldKeys map[Node]elementRenderKey) (elementRenderKey, bool) {
	if oldKeys != nil {
		if key, ok := oldKeys[node]; ok {
			return key, true
		}
	}
	element, ok := node.(*Element)
	if !ok || element == nil {
		return elementRenderKey{}, false
	}
	return element.renderKey, true
}

func (window *Window) tryTranslateBlit(node Node, oldBounds Rect, newBounds Rect, oldKeys map[Node]elementRenderKey, scrollOffset int) (Rect, bool) {
	if window == nil || window.canvas == nil || window.pendingScrollDelta() != 0 {
		return Rect{}, false
	}
	element, ok := node.(*Element)
	if !ok || element == nil || len(element.Children) > 0 {
		return Rect{}, false
	}
	if oldBounds.Empty() || newBounds.Empty() {
		return Rect{}, false
	}
	if oldBounds.Width != newBounds.Width || oldBounds.Height != newBounds.Height {
		return Rect{}, false
	}
	dx := newBounds.X - oldBounds.X
	dy := newBounds.Y - oldBounds.Y
	if (dx == 0 && dy == 0) || (dx != 0 && dy != 0) {
		return Rect{}, false
	}
	if absInt(dx) >= oldBounds.Width || absInt(dy) >= oldBounds.Height {
		return Rect{}, false
	}
	style := element.effectiveStyle()
	if !element.canUseTranslateBlit(style) {
		return Rect{}, false
	}
	oldKey, ok := window.oldRenderKeyFor(node, oldKeys)
	if !ok || !elementRenderKeyEqual(oldKey, element.renderKeyFor(style)) {
		return Rect{}, false
	}
	index, ok := window.renderIndex[node]
	if !ok || index < 0 || index >= len(window.renderList) {
		return Rect{}, false
	}
	item := window.renderList[index]
	if item.clip.set {
		return Rect{}, false
	}
	src := oldBounds
	dst := newBounds
	if scrollOffset != 0 {
		src.Y += scrollOffset
		dst.Y += scrollOffset
	}
	canvasBounds := Rect{X: 0, Y: 0, Width: window.canvas.Width(), Height: window.canvas.Height()}
	if !rectContainsRect(canvasBounds, src) || !rectContainsRect(canvasBounds, dst) {
		return Rect{}, false
	}
	rootClip := window.rootClipState()
	if rootClip.set && (!rectContainsRect(rootClip.rect, src) || !rectContainsRect(rootClip.rect, dst)) {
		return Rect{}, false
	}
	if window.translateBlitIntersectsExisting(src, dst) {
		return Rect{}, false
	}
	if window.translateBlitBlockedByLaterItems(index, dst, scrollOffset) {
		return Rect{}, false
	}
	exposed := translateExposeRect(src, dst)
	if exposed.Empty() {
		return Rect{}, false
	}
	window.translateBlits = append(window.translateBlits, translateBlitOp{src: src, dst: dst})
	window.markPresentRect(dst)
	return exposed, true
}

func (window *Window) applyPendingTranslateBlits() bool {
	if window == nil || window.canvas == nil || len(window.translateBlits) == 0 {
		return false
	}
	for _, op := range window.translateBlits {
		if op.src.Empty() || op.dst.Empty() {
			continue
		}
		window.canvas.BlitSelf(op.src, op.dst.X, op.dst.Y)
	}
	window.translateBlits = window.translateBlits[:0]
	return true
}

func copyElementRenderKeys(bounds map[Node]Rect) map[Node]elementRenderKey {
	if len(bounds) == 0 {
		return nil
	}
	keys := make(map[Node]elementRenderKey, len(bounds))
	for node := range bounds {
		element, ok := node.(*Element)
		if !ok || element == nil {
			continue
		}
		keys[node] = element.renderKey
	}
	if len(keys) == 0 {
		return nil
	}
	return keys
}
