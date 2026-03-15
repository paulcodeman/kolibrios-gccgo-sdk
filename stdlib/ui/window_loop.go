package ui

import (
	"kos"
	"runtime"
)

func (window *Window) Start() {
	if window == nil {
		return
	}
	if registerWindowStart() {
		window.primary = true
		window.startLoop(true)
		return
	}
	window.startThreadedNoRegister(0)
}

// StartThreaded launches the window event loop on a new runtime-managed OS thread.
// Avoid concurrent access to the window from other threads.
func (window *Window) StartThreaded() (tid uint32, ok bool) {
	return window.StartThreadedWithStack(0)
}

func (window *Window) StartThreadedWithStack(stackSize int) (tid uint32, ok bool) {
	if window == nil {
		return 0, false
	}
	window.primary = false
	registerWindowStart()
	return window.startThreadedNoRegister(stackSize)
}

func registerWindowStart() bool {
	if windowStartCount == 0 {
		windowStartCount = 1
		return true
	}
	windowStartCount++
	return false
}

func (window *Window) startThreadedNoRegister(stackSize int) (tid uint32, ok bool) {
	if window == nil {
		return 0, false
	}
	return kos.CreateRuntimeThread(func() { window.startLoop(false) }, stackSize)
}

func (window *Window) startLoop(pollGC bool) {
	if window == nil {
		return
	}
	runtime.LockOSThread()
	window.running = true
	window.syncThreadSlot(true)
	if kos.ThreadDebug != nil {
		tid, _ := kos.CurrentThreadID()
		kos.ThreadDebug(kos.ThreadDebugEvent{
			Stage:    "window_start",
			ThreadID: tid,
		})
	}
	if FastNoFontSmoothing {
		kos.SetFontSmoothing(kos.FontSmoothingOff)
	}
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse | kos.EventMaskMouseActiveWindowOnly)
	// Ensure the window is created and painted at least once.
	window.Redraw()
	window.lastEventAt = kos.UptimeCentiseconds()
	window.syncThreadSlot(true)
	if window.threadSlotSet {
		kos.FocusWindowSlot(window.threadSlot)
	}
	if DebugStartHook != nil {
		DebugStartHook(window)
	}
	for window.running {
		redrawn := false
		needsRedraw := false
		event := window.nextEvent(pollGC)
		switch event {
		case kos.EventNone:
		case kos.EventRedraw:
			window.Redraw()
			redrawn = true
		case kos.EventMouse:
			window.drainMouseEvents()
			if window.handleMouse() {
				needsRedraw = true
			}
			if MouseEventThrottleMs > 0 && !window.lastMouseInteractive && window.pendingEvent == kos.EventNone {
				kos.SleepMilliseconds(MouseEventThrottleMs)
			}
		case kos.EventKey:
			if window.handleKey() {
				needsRedraw = true
			}
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				window.Close()
			}
		}
		if !window.running {
			break
		}
		if window.scrollRedraw {
			window.scrollRedraw = false
			needsRedraw = true
		}
		if needsRedraw {
			window.RedrawContent()
			redrawn = true
		}
		if !redrawn && window.caretBlinkNeedsRedraw() {
			window.noteCaretBlinkDirty()
			window.RedrawContent()
		}
		if pollGC && WindowPollRuntimeGC && event != kos.EventNone && WindowGCPollActiveIntervalMs > 0 {
			window.pollRuntimeGCWithInterval(WindowGCPollActiveIntervalMs)
		}
	}
	if window.primary {
		kos.Exit()
	} else {
		kos.ExitRuntimeThread()
	}
}

func (window *Window) nextEvent(pollGC bool) kos.EventType {
	if window == nil {
		return kos.EventNone
	}
	if window.pendingEvent != kos.EventNone {
		event := window.pendingEvent
		window.pendingEvent = kos.EventNone
		window.noteEvent(event)
		return event
	}
	event := window.waitEvent(pollGC)
	window.noteEvent(event)
	return event
}

func (window *Window) waitEvent(pollGC bool) kos.EventType {
	timeout := window.caretBlinkTimeout()
	if !window.primary && timeout == 0 {
		timeout = 1
	}
	if !pollGC || !WindowPollRuntimeGC {
		if timeout == 0 {
			return kos.EventType(kos.Event())
		}
		kos.PollRuntimeWorldStopRaw()
		event := kos.EventType(kos.CheckEvent())
		if event != kos.EventNone {
			return event
		}
		return kos.EventType(kos.WaitEventTimeout(timeout))
	}
	event := kos.EventType(kos.CheckEvent())
	if event != kos.EventNone {
		return event
	}
	if window.gcIdleReady() {
		window.pollRuntimeGC()
	}
	if timeout == 0 {
		return kos.EventType(kos.Event())
	}
	kos.PollRuntimeWorldStopRaw()
	return kos.EventType(kos.WaitEventTimeout(timeout))
}

func (window *Window) noteEvent(event kos.EventType) {
	if window == nil || event == kos.EventNone {
		return
	}
	switch event {
	case kos.EventMouse, kos.EventKey, kos.EventButton, kos.EventDesktop:
		window.lastEventAt = kos.UptimeCentiseconds()
	}
}

func (window *Window) gcIdleReady() bool {
	if WindowGCPollIdleMs == 0 {
		return true
	}
	if window == nil {
		return false
	}
	now := kos.UptimeCentiseconds()
	last := window.lastEventAt
	if last == 0 {
		window.lastEventAt = now
		return false
	}
	idleMs := (now - last) * 10
	return idleMs >= WindowGCPollIdleMs
}

func (window *Window) pollRuntimeGC() {
	window.pollRuntimeGCWithInterval(WindowGCPollIntervalMs)
}

func (window *Window) pollRuntimeGCWithInterval(intervalMs uint32) {
	if intervalMs == 0 {
		kos.PollRuntimeGCRaw()
		window.lastGCPollAt = kos.UptimeCentiseconds()
		return
	}
	interval := (intervalMs + 9) / 10
	if interval == 0 {
		interval = 1
	}
	now := kos.UptimeCentiseconds()
	if window.lastGCPollAt == 0 || now-window.lastGCPollAt >= interval {
		kos.PollRuntimeGCRaw()
		window.lastGCPollAt = now
	}
}

func (window *Window) syncThreadSlot(force bool) {
	if window == nil || window.threadSlotSet {
		return
	}
	if !force && window.threadSlotRetry > 0 {
		window.threadSlotRetry--
		return
	}
	slot, ok := kos.CurrentThreadSlotIndex()
	if ok {
		window.threadSlot = slot
		window.threadSlotSet = true
		window.threadSlotRetry = 0
		return
	}
	if !force {
		window.threadSlotRetry = 8
	}
}

func (window *Window) isActiveWindow() bool {
	if window == nil {
		return true
	}
	window.syncThreadSlot(false)
	if !window.threadSlotSet {
		return true
	}
	return kos.ActiveWindowSlot() == window.threadSlot
}
