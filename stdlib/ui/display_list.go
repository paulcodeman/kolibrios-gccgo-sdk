package ui

import "kos"

type DisplayList struct {
	items        []renderItem
	rootClip     clipState
	scrollOffset int
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
	if !full && !dirty.Empty() {
		window.canvas.PushClip(dirty)
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
		if item.clip.set {
			clipRect := item.clip.rect
			if list.scrollOffset != 0 {
				clipRect.Y += list.scrollOffset
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
		if item.clip.set {
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
