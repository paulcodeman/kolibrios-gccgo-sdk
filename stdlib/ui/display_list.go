package ui

import "kos"

type DisplayList struct {
	items        []renderItem
	rootClip     clipState
	scrollOffset int
}

func nodeNeedsFullDirtyPaint(node Node) bool {
	if node == nil {
		return false
	}
	switch current := node.(type) {
	case *Element:
		if current == nil {
			return false
		}
		style := current.effectiveStyle()
		if current.canUseDirtyClip(style) {
			return false
		}
		if current.isTextInput() {
			return true
		}
		if elementShowsDefaultFocusRing(current) {
			return true
		}
		if resolveBorderRadius(style).Active() {
			return true
		}
		if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
			return true
		}
		if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
			return true
		}
	case *DocumentView:
		if current == nil {
			return false
		}
		style := current.effectiveStyle()
		if resolveBorderRadius(style).Active() {
			return true
		}
		if shadow, ok := resolveShadow(style.shadow); ok && shadow != nil {
			return true
		}
		if opacity, ok := resolveOpacity(style.opacity); ok && opacity < 255 {
			return true
		}
	}
	return false
}

func (element *Element) canUseDirtyClip(style Style) bool {
	if element == nil || FastNoCache {
		return false
	}
	rect := element.layoutRect
	if rect.Empty() {
		rect = element.Bounds()
	}
	cacheable, _, _ := element.cacheInfo(style, rect)
	return cacheable
}

func (window *Window) scrollPaintOffset() int {
	if window == nil {
		return 0
	}
	if window.scrollEnabled() && window.scrollY != 0 {
		return -window.scrollY
	}
	return 0
}

func (window *Window) currentDisplayList() DisplayList {
	if window == nil {
		return DisplayList{}
	}
	return DisplayList{
		items:        window.renderList,
		rootClip:     window.rootClipState(),
		scrollOffset: window.scrollPaintOffset(),
	}
}

func (list DisplayList) Items() []renderItem {
	return list.items
}

func (list DisplayList) ScrollOffset() int {
	return list.scrollOffset
}

func (list DisplayList) Paint(window *Window, full bool, dirty Rect, stats *FrameStats) {
	if window == nil || window.canvas == nil {
		return
	}
	if list.rootClip.set {
		window.canvas.PushClip(list.rootClip.rect)
		defer window.canvas.PopClip()
	}
	for _, item := range list.items {
		if item.node == nil {
			continue
		}
		paint := item.paint
		if list.scrollOffset != 0 {
			paint.Y += list.scrollOffset
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
		clipSet := false
		clipRect := Rect{}
		useDirtyClip := !full && !dirty.Empty() && !nodeNeedsFullDirtyPaint(item.node)
		if useDirtyClip {
			clipRect = dirty
			clipSet = true
		}
		if item.clip.set {
			itemClip := item.clip.rect
			if list.scrollOffset != 0 {
				itemClip.Y += list.scrollOffset
			}
			if clipSet {
				clipRect = IntersectRect(clipRect, itemClip)
			} else {
				clipRect = itemClip
				clipSet = true
			}
		}
		if clipSet {
			if clipRect.Empty() {
				continue
			}
			window.canvas.PushClip(clipRect)
		}
		if stats != nil && !window.DisableNodeTiming {
			start := kos.UptimeNanoseconds()
			if list.scrollOffset != 0 {
				if offsetAware, ok := item.node.(OffsetDrawAware); ok {
					offsetAware.DrawToOffset(window.canvas, list.scrollOffset)
				} else if element != nil {
					window.drawElementWithOffset(element, list.scrollOffset)
				} else {
					item.node.DrawTo(window.canvas)
				}
			} else {
				item.node.DrawTo(window.canvas)
			}
			stats.NodesNs += kos.UptimeNanoseconds() - start
		} else if list.scrollOffset != 0 {
			if offsetAware, ok := item.node.(OffsetDrawAware); ok {
				offsetAware.DrawToOffset(window.canvas, list.scrollOffset)
			} else if element != nil {
				window.drawElementWithOffset(element, list.scrollOffset)
			} else {
				item.node.DrawTo(window.canvas)
			}
		} else {
			item.node.DrawTo(window.canvas)
		}
		if clipSet {
			window.canvas.PopClip()
		}
		if aware, ok := item.node.(DirtyAware); ok {
			aware.ClearDirty()
		}
	}
}

func (list DisplayList) Find(x int, y int) Node {
	for i := len(list.items) - 1; i >= 0; i-- {
		item := list.items[i]
		if item.node == nil {
			continue
		}
		paint := item.paint
		if list.scrollOffset != 0 {
			paint.Y += list.scrollOffset
		}
		if paint.Contains(x, y) {
			return item.node
		}
	}
	return nil
}
