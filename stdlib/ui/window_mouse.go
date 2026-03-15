package ui

import "kos"

func (window *Window) handleMouse() bool {
	if window == nil || window.client.Empty() {
		return false
	}
	if !window.isActiveWindow() {
		needsRedraw := false
		window.prevMouseButtons = kos.MouseButtonInfo{}
		window.hoverDirty = true
		window.lastMouseValid = false
		if window.focused != nil {
			if aware, ok := window.focused.(FocusAware); ok {
				if aware.SetFocus(false) {
					needsRedraw = true
					window.noteDirty(window.focused)
				}
			}
			window.focused = nil
			window.caretBlinkResetAt = 0
		}
		if window.mouseDown != nil {
			if aware, ok := window.mouseDown.(ActiveAware); ok {
				if aware.SetActive(false) {
					needsRedraw = true
					window.noteDirty(window.mouseDown)
				}
			}
			window.mouseDown = nil
		}
		if window.mouseHover != nil {
			if aware, ok := window.mouseHover.(HoverAware); ok {
				if aware.SetHover(false) {
					needsRedraw = true
					window.noteDirty(window.mouseHover)
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
	leftHeld := held.LeftHeld
	window.lastMouseInteractive = leftHeld || buttons.VerticalScroll || buttons.HorizontalScroll
	leftPressed := buttons.LeftPressed || (leftHeld && !window.prevMouseButtons.LeftHeld)
	leftReleased := buttons.LeftReleased || (!leftHeld && window.prevMouseButtons.LeftHeld)
	window.prevMouseButtons = held
	if leftPressed {
		window.awaitingPress = false
	}
	if window.scrollDragActive {
		needsRedraw := false
		if leftHeld {
			if window.handleWindowScrollbarDrag(y) {
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
	scrollbarHit := window.windowScrollbarHit(x, y)

	hover := window.mouseHover
	inside := x >= 0 && y >= 0 && x < window.client.Width && y < window.client.Height
	if !inside {
		hover = nil
		window.lastMouseX = x
		window.lastMouseY = y
		window.lastMouseValid = false
		window.hoverDirty = false
	} else {
		if window.hoverDirty || !window.lastMouseValid || window.lastMouseX != x || window.lastMouseY != y {
			hover = window.hitTest(x, y)
		}
		if scrollbarHit {
			hover = nil
		}
		window.lastMouseX = x
		window.lastMouseY = y
		window.lastMouseValid = true
		window.hoverDirty = false
	}
	needsRedraw := false
	if hover != window.mouseHover {
		if aware, ok := window.mouseHover.(HoverAware); ok {
			if aware.SetHover(false) {
				needsRedraw = true
				window.noteDirty(window.mouseHover)
			}
		}
		window.mouseHover = hover
		if aware, ok := hover.(HoverAware); ok {
			if aware.SetHover(true) {
				needsRedraw = true
				window.noteDirty(hover)
			}
		}
	}

	if leftPressed {
		if scrollbarHit {
			if window.handleWindowScrollbarMouseDown(x, y) {
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
				window.noteDirty(window.mouseDown)
			}
		}
		if element, ok := window.mouseDown.(*Element); ok && element.isTextInput() {
			if element.handleTextMouseDown(eventX, eventY) {
				needsRedraw = true
				window.noteDirty(element)
			}
		}
	} else if leftHeld && window.mouseDown != nil {
		if aware, ok := window.mouseDown.(ActiveAware); ok {
			if aware.SetActive(window.mouseDown == hover) {
				needsRedraw = true
				window.noteDirty(window.mouseDown)
			}
		}
	}
	if leftHeld && window.mouseDown != nil {
		if element, ok := window.mouseDown.(*Element); ok && element.isTextInput() {
			if element.handleTextMouseDrag(eventX, eventY) {
				needsRedraw = true
				window.noteDirty(element)
			}
		}
	}
	if leftReleased {
		if element, ok := window.mouseDown.(*Element); ok && element.isTextInput() {
			if element.handleTextMouseUp() {
				needsRedraw = true
				window.noteDirty(element)
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
				Type:   EventClick,
				X:      eventX,
				Y:      eventY,
				Button: MouseLeft,
				Target: hover,
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
				window.noteDirty(window.mouseDown)
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
				Type:   EventClick,
				X:      eventX,
				Y:      eventY,
				Button: MouseLeft,
				Target: target,
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
		if aware, ok := window.mouseDown.(ActiveAware); ok {
			if aware.SetActive(false) {
				needsRedraw = true
				window.noteDirty(window.mouseDown)
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
