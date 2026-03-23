package ui

import "kos"

func (window *Window) hoverStateNeedsDirtyNode(node Node) bool {
	if window == nil || node == nil {
		return false
	}
	switch node.(type) {
	case *Element, *DocumentView:
		return false
	default:
		return true
	}
}

func (window *Window) noteHoverStateChange(node Node) {
	if window == nil || node == nil {
		return
	}
	if window.hoverStateNeedsDirtyNode(node) {
		window.noteDirty(node)
	}
}

func (window *Window) activeStateNeedsDirtyNode(node Node) bool {
	if window == nil || node == nil {
		return false
	}
	switch node.(type) {
	case *Element, *DocumentView:
		return false
	default:
		return true
	}
}

func (window *Window) noteActiveStateChange(node Node) {
	if window == nil || node == nil {
		return
	}
	if window.activeStateNeedsDirtyNode(node) {
		window.noteDirty(node)
	}
}

func dispatchPointerCancelForNode(node Node, x int, y int, button MouseButton, buttons PointerButtons) bool {
	switch current := node.(type) {
	case *Element:
		return current.dispatchPointerCancelEvent(x, y, button, buttons)
	case *DocumentView:
		return current.dispatchPointerCancelEvent(x, y, button, buttons)
	default:
		return false
	}
}

func (window *Window) handleMouse() bool {
	if window == nil || window.client.Empty() {
		return false
	}
	if !window.isActiveWindow() {
		needsRedraw := false
		cancelX := window.lastMouseX
		cancelY := window.lastMouseY
		if !window.lastMouseValid {
			cancelX = 0
			cancelY = 0
		}
		window.prevMouseButtons = kos.MouseButtonInfo{}
		window.hoverDirty = true
		window.lastMouseValid = false
		if window.focused != nil {
			if aware, ok := window.focused.(FocusAware); ok {
				if aware.SetFocus(false) {
					needsRedraw = true
					window.noteFocusStateChange(window.focused)
				}
			}
			window.focused = nil
			window.caretBlinkResetAt = 0
		}
		if window.mouseDown != nil {
			if dispatchPointerCancelForNode(window.mouseDown, cancelX, cancelY, MouseLeft, PointerButtonsNone) {
				needsRedraw = true
				window.noteDirty(window.mouseDown)
			}
			if aware, ok := window.mouseDown.(ActiveAware); ok {
				if aware.SetActive(false) {
					needsRedraw = true
					window.noteActiveStateChange(window.mouseDown)
				}
			}
			window.mouseDown = nil
		}
		if window.mouseHover != nil {
			if aware, ok := window.mouseHover.(HoverAware); ok {
				if aware.SetHover(false) {
					needsRedraw = true
					window.noteHoverStateChange(window.mouseHover)
				}
			}
			window.mouseHover = nil
		}
		return needsRedraw
	}
	pos := kos.MouseWindowPosition()
	x := pos.X - window.client.X
	y := pos.Y - window.client.Y
	eventX := x
	eventY := y
	if window.scrollEnabled() && window.scrollY != 0 {
		eventY = y + window.scrollY
	}
	buttons := kos.MouseButtons()
	held := kos.MouseHeldButtons()
	pointerButtons := pointerButtonsFromMouseInfo(held)
	leftHeld := held.LeftHeld
	window.lastMouseInteractive = leftHeld || buttons.VerticalScroll || buttons.HorizontalScroll
	leftPressed := buttons.LeftPressed || (leftHeld && !window.prevMouseButtons.LeftHeld)
	leftReleased := buttons.LeftReleased || (!leftHeld && window.prevMouseButtons.LeftHeld)
	window.prevMouseButtons = held
	if leftPressed {
		window.awaitingPress = false
	}
	if window.scrollDragActive {
		scrollState := window.windowScrollbarState()
		needsRedraw := false
		if leftHeld {
			if window.handleWindowScrollbarDragWithState(scrollState, y) {
				needsRedraw = true
			}
		}
		if leftReleased || !leftHeld {
			window.scrollDragActive = false
		}
		if DebugMouseHook != nil {
			DebugMouseHook(window, MouseDebugEvent{
				X:            x,
				Y:            y,
				Buttons:      buttons,
				Held:         held,
				LeftPressed:  leftPressed,
				LeftReleased: leftReleased,
				LeftHeld:     leftHeld,
				Hovered:      false,
				MouseDown:    false,
			})
		}
		return needsRedraw
	}

	hover := window.mouseHover
	inside := x >= 0 && y >= 0 && x < window.client.Width && y < window.client.Height
	var scrollState windowScrollPropertyState
	scrollbarHit := false
	if inside {
		scrollState = window.windowScrollbarState()
		scrollbarHit = windowScrollbarHitWithState(scrollState, x, y)
	}
	mouseMoved := inside && (window.hoverDirty || !window.lastMouseValid || window.lastMouseX != x || window.lastMouseY != y)
	if !inside {
		hover = nil
		window.lastMouseX = x
		window.lastMouseY = y
		window.lastMouseValid = false
		window.hoverDirty = false
	} else {
		if mouseMoved {
			if scrollbarHit {
				hover = nil
			} else {
				hover = window.hitTest(x, y)
			}
		}
		window.lastMouseX = x
		window.lastMouseY = y
		window.lastMouseValid = true
		window.hoverDirty = false
	}
	needsRedraw := false
	if hover != window.mouseHover {
		if element, ok := window.mouseHover.(*Element); ok && element != nil {
			window.noteHandlerMayMutate(element)
			if element.dispatchMouseLeaveEvent(eventX, eventY) {
				needsRedraw = true
				window.noteDirty(element)
			}
		}
		if aware, ok := window.mouseHover.(HoverAware); ok {
			if aware.SetHover(false) {
				needsRedraw = true
				window.noteHoverStateChange(window.mouseHover)
			}
		}
		window.mouseHover = hover
		if aware, ok := hover.(HoverAware); ok {
			if aware.SetHover(true) {
				needsRedraw = true
				window.noteHoverStateChange(hover)
			}
		}
		if element, ok := hover.(*Element); ok && element != nil {
			window.noteHandlerMayMutate(element)
			if element.dispatchMouseEnterEvent(eventX, eventY) {
				needsRedraw = true
				window.noteDirty(element)
			}
		}
	}
	if mouseMoved {
		if aware, ok := hover.(MouseMoveAware); ok {
			window.noteHandlerMayMutate(hover)
			if aware.HandleMouseMove(eventX, eventY, pointerButtons) {
				needsRedraw = true
				if _, isDocumentView := hover.(*DocumentView); !isDocumentView {
					window.noteDirty(hover)
				}
			}
		}
	}
	if leftHeld && window.mouseDown != nil && window.mouseDown != hover {
		if aware, ok := window.mouseDown.(MouseMoveAware); ok {
			window.noteHandlerMayMutate(window.mouseDown)
			if aware.HandleMouseMove(eventX, eventY, pointerButtons) {
				needsRedraw = true
				if _, isDocumentView := window.mouseDown.(*DocumentView); !isDocumentView {
					window.noteDirty(window.mouseDown)
				}
			}
		}
	}

	if leftPressed {
		if scrollbarHit {
			if window.handleWindowScrollbarMouseDownWithState(scrollState, x, y) {
				needsRedraw = true
				if DebugMouseHook != nil {
					DebugMouseHook(window, MouseDebugEvent{
						X:            x,
						Y:            y,
						Buttons:      buttons,
						Held:         held,
						LeftPressed:  leftPressed,
						LeftReleased: leftReleased,
						LeftHeld:     leftHeld,
						Hovered:      hover != nil,
						MouseDown:    window.mouseDown != nil,
					})
				}
				return needsRedraw
			}
		}
		window.mouseDown = hover
		if aware, ok := window.mouseDown.(ActiveAware); ok {
			if aware.SetActive(true) {
				needsRedraw = true
				window.noteActiveStateChange(window.mouseDown)
			}
		}
		if aware, ok := window.mouseDown.(MouseDownAware); ok {
			window.noteHandlerMayMutate(window.mouseDown)
			if aware.HandleMouseDown(eventX, eventY, MouseLeft, pointerButtons) {
				needsRedraw = true
				window.noteDirty(window.mouseDown)
			}
		}
	} else if leftHeld && window.mouseDown != nil {
		if aware, ok := window.mouseDown.(ActiveAware); ok {
			if aware.SetActive(window.mouseDown == hover) {
				needsRedraw = true
				window.noteActiveStateChange(window.mouseDown)
			}
		}
	}
	if leftReleased {
		mouseDown := window.mouseDown
		if aware, ok := mouseDown.(MouseUpAware); ok {
			window.noteHandlerMayMutate(mouseDown)
			if aware.HandleMouseUp(eventX, eventY, MouseLeft, pointerButtons) {
				needsRedraw = true
				window.noteDirty(mouseDown)
			}
		}
		focusTarget := hover
		if element, ok := window.mouseDown.(*Element); ok && element.isTextInput() {
			focusTarget = element
		}
		if window.setFocus(focusTarget) {
			needsRedraw = true
		}
		if window.mouseDown == nil && hover != nil && window.awaitingPress {
			window.awaitingPress = false
			window.noteHandlerMayMutate(hover)
			handled := hover.Handle(Event{
				Type:    EventClick,
				X:       eventX,
				Y:       eventY,
				Button:  MouseLeft,
				Buttons: pointerButtons,
				Target:  hover,
			})
			if handled {
				window.noteDirty(hover)
			}
			if DebugMouseHook != nil {
				DebugMouseHook(window, MouseDebugEvent{
					X:            x,
					Y:            y,
					Buttons:      buttons,
					Held:         held,
					LeftPressed:  leftPressed,
					LeftReleased: leftReleased,
					LeftHeld:     leftHeld,
					Hovered:      hover != nil,
					MouseDown:    window.mouseDown != nil,
				})
			}
			return handled || needsRedraw
		}
		window.awaitingPress = false
		if aware, ok := window.mouseDown.(ActiveAware); ok {
			if aware.SetActive(false) {
				needsRedraw = true
				window.noteActiveStateChange(window.mouseDown)
			}
		}
		target := hover
		if target != nil && target == window.mouseDown {
			if element, ok := target.(*Element); ok && element.isTextInput() && element.dragMoved {
				window.mouseDown = nil
				if DebugMouseHook != nil {
					DebugMouseHook(window, MouseDebugEvent{
						X:            x,
						Y:            y,
						Buttons:      buttons,
						Held:         held,
						LeftPressed:  leftPressed,
						LeftReleased: leftReleased,
						LeftHeld:     leftHeld,
						Hovered:      hover != nil,
						MouseDown:    window.mouseDown != nil,
					})
				}
				return needsRedraw
			}
			window.noteHandlerMayMutate(target)
			handled := target.Handle(Event{
				Type:    EventClick,
				X:       eventX,
				Y:       eventY,
				Button:  MouseLeft,
				Buttons: pointerButtons,
				Target:  target,
			})
			if handled {
				window.noteDirty(target)
			}
			window.mouseDown = nil
			if DebugMouseHook != nil {
				DebugMouseHook(window, MouseDebugEvent{
					X:            x,
					Y:            y,
					Buttons:      buttons,
					Held:         held,
					LeftPressed:  leftPressed,
					LeftReleased: leftReleased,
					LeftHeld:     leftHeld,
					Hovered:      hover != nil,
					MouseDown:    window.mouseDown != nil,
				})
			}
			return handled || needsRedraw
		}
		window.mouseDown = nil
	} else if window.mouseDown != nil && !leftHeld {
		if dispatchPointerCancelForNode(window.mouseDown, eventX, eventY, MouseLeft, PointerButtonsNone) {
			needsRedraw = true
			window.noteDirty(window.mouseDown)
		}
		if aware, ok := window.mouseDown.(ActiveAware); ok {
			if aware.SetActive(false) {
				needsRedraw = true
				window.noteActiveStateChange(window.mouseDown)
			}
		}
		window.mouseDown = nil
	}

	delta := kos.MouseScrollDelta()
	if delta.X != 0 || delta.Y != 0 {
		target := hover
		if target == nil {
			target = window.focused
		}
		if scrollbarHit {
			target = nil
		}
		handled := false
		if target != nil {
			if aware, ok := target.(ScrollAware); ok {
				window.noteHandlerMayMutate(target)
				if aware.HandleScroll(delta.X, delta.Y) {
					handled = true
					needsRedraw = true
					window.noteDirty(target)
				}
			}
		}
		if !handled && delta.Y != 0 {
			if window.scrollWindowBy(delta.Y) {
				needsRedraw = true
			}
		}
	}

	if DebugMouseHook != nil {
		DebugMouseHook(window, MouseDebugEvent{
			X:            x,
			Y:            y,
			Buttons:      buttons,
			Held:         held,
			LeftPressed:  leftPressed,
			LeftReleased: leftReleased,
			LeftHeld:     leftHeld,
			Hovered:      hover != nil,
			MouseDown:    window.mouseDown != nil,
		})
	}
	return needsRedraw
}

func (window *Window) drainMouseEvents() {
	if window == nil {
		return
	}
	for {
		event := kos.EventType(kos.CheckEvent())
		if event == kos.EventNone {
			return
		}
		if event == kos.EventMouse {
			continue
		}
		window.pendingEvent = event
		return
	}
}
