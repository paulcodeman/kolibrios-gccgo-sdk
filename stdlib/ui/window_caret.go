package ui

import "kos"

func (window *Window) caretBlinkActive() bool {
	if WindowCaretBlinkMs == 0 {
		return false
	}
	if window == nil || window.focused == nil {
		return false
	}
	switch current := window.focused.(type) {
	case *Element:
		if current == nil || !current.isTextInput() {
			return false
		}
		return current.focused && !current.hasSelection()
	case *DocumentView:
		return current != nil && current.textInputBlinkActive()
	default:
		return false
	}
}

func (window *Window) caretBlinkTimeout() uint32 {
	if !window.caretBlinkActive() {
		return 0
	}
	half := WindowCaretBlinkMs / 2
	if half == 0 {
		half = 1
	}
	timeout := (half + 9) / 10
	if timeout == 0 {
		timeout = 1
	}
	return timeout
}

func (window *Window) caretBlinkVisible() bool {
	if WindowCaretBlinkMs == 0 {
		return true
	}
	if window == nil {
		return true
	}
	period := WindowCaretBlinkMs
	if period == 0 {
		return true
	}
	now := kos.UptimeCentiseconds()
	start := window.caretBlinkResetAt
	if start == 0 {
		start = now
	}
	elapsedMs := (now - start) * 10
	phase := elapsedMs % period
	return phase < (period / 2)
}

func (window *Window) caretBlinkNeedsRedraw() bool {
	if window == nil || !window.caretBlinkActive() {
		return false
	}
	if !window.caretBlinkVisibleSet {
		return false
	}
	return window.caretBlinkVisible() != window.caretBlinkVisibleCached
}

func (window *Window) noteCaretBlinkDrawn() {
	if window == nil {
		return
	}
	if !window.caretBlinkActive() {
		window.caretBlinkVisibleSet = false
		window.caretBlinkVisibleCached = false
		return
	}
	window.caretBlinkVisibleSet = true
	window.caretBlinkVisibleCached = window.caretBlinkVisible()
}

func (window *Window) noteCaretBlinkDirty() {
	if window == nil || window.focused == nil {
		return
	}
	switch current := window.focused.(type) {
	case *Element:
		if current.isTextInput() {
			style := current.effectiveStyle()
			rect := current.layoutRect
			if rect.Empty() {
				rect = current.Bounds()
			}
			if dirty := current.caretDirtyRect(rect, style); !dirty.Empty() {
				window.InvalidateVisualContent(dirty)
				return
			}
			current.MarkDirty()
		}
	case *DocumentView:
		if current != nil {
			if dirty := current.textInputCaretDirtyRect(); !dirty.Empty() {
				window.InvalidateVisual(dirty)
				return
			}
			current.MarkDirty()
		}
	}
}
