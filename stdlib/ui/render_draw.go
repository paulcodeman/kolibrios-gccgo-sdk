package ui

import "kos"

func (window *Window) rootClipState() clipState {
	if window == nil || window.canvas == nil {
		return clipState{}
	}
	clipX, clipY := overflowClipAxes(window.Style)
	if !clipY && window.scrollEnabled() {
		clipY = true
	}
	if !clipX && !clipY {
		return clipState{}
	}
	content := window.contentRect()
	if content.Empty() {
		return clipState{rect: Rect{}, set: true}
	}
	canvasBounds := Rect{X: 0, Y: 0, Width: window.canvas.Width(), Height: window.canvas.Height()}
	base := canvasBounds
	if clipX {
		base.X = content.X
		base.Width = content.Width
	}
	if clipY {
		base.Y = content.Y
		base.Height = content.Height
	}
	base = IntersectRect(base, canvasBounds)
	return clipState{rect: base, set: true}
}

func (window *Window) drawRenderList(full bool, dirty Rect, stats *FrameStats) {
	if window == nil || window.canvas == nil {
		return
	}
	scrollOffset := 0
	if window.scrollEnabled() && window.scrollY != 0 {
		scrollOffset = -window.scrollY
	}
	if clip := window.rootClipState(); clip.set {
		window.canvas.PushClip(clip.rect)
		defer window.canvas.PopClip()
	}
	if !full && !dirty.Empty() {
		window.canvas.PushClip(dirty)
		defer window.canvas.PopClip()
	}
	renderList := window.renderList
	for _, item := range renderList {
		if item.node == nil {
			continue
		}
		paint := item.paint
		if scrollOffset != 0 {
			paint.Y += scrollOffset
		}
		if paint.Empty() {
			continue
		}
		if !full && IntersectRect(paint, dirty).Empty() {
			continue
		}
		var element *Element
		if el, ok := item.node.(*Element); ok && el != nil {
			element = el
		}
		if item.clip.set {
			clipRect := item.clip.rect
			if scrollOffset != 0 {
				clipRect.Y += scrollOffset
			}
			window.canvas.PushClip(clipRect)
		}
		if stats != nil && !window.DisableNodeTiming {
			start := kos.UptimeNanoseconds()
			if scrollOffset != 0 && element != nil {
				window.drawElementWithOffset(element, scrollOffset)
			} else {
				item.node.DrawTo(window.canvas)
			}
			elapsed := kos.UptimeNanoseconds() - start
			if stats != nil && !window.DisableNodeTiming {
				stats.NodesNs += elapsed
			}
		} else {
			if scrollOffset != 0 && element != nil {
				window.drawElementWithOffset(element, scrollOffset)
			} else {
				item.node.DrawTo(window.canvas)
			}
		}
		if item.clip.set {
			window.canvas.PopClip()
		}
		if aware, ok := item.node.(DirtyAware); ok {
			aware.ClearDirty()
		}
	}
}

func (window *Window) drawElementWithOffset(element *Element, offsetY int) {
	if window == nil || window.canvas == nil || element == nil {
		return
	}
	style := element.effectiveStyle()
	if display, ok := resolveDisplay(style.Display); ok && display == DisplayNone {
		return
	}
	element.updateRenderKey(style)
	rect := element.layoutRect
	if rect.Empty() {
		element.applyLayout(window.canvas, style)
		rect = element.layoutRect
	}
	if rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	if offsetY != 0 {
		rect.Y += offsetY
	}
	if element.tryDrawFromCache(window.canvas, rect, style) {
		return
	}
	element.drawToRect(window.canvas, rect, style)
}
