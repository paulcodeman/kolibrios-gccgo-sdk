package ui

import "kos"

func (window *Window) caretBlinkActive() bool {
	if WindowCaretBlinkMs == 0 {
		return false
	}
	if window == nil || window.focused == nil {
		return false
	}
	element, ok := window.focused.(*Element)
	if !ok || !element.isTextInput() {
		return false
	}
	return element.focused
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
	if element, ok := window.focused.(*Element); ok && element.isTextInput() {
		element.MarkDirty()
	}
}
