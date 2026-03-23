package ui

import "kos"

type EventType int
type EventPhase uint8
type PointerType uint8
type PointerButtons uint8

const (
	EventClick EventType = iota + 1
	EventMouseDown
	EventMouseUp
	EventMouseMove
	EventMouseEnter
	EventMouseLeave
	EventPointerDown
	EventPointerUp
	EventPointerMove
	EventPointerEnter
	EventPointerLeave
	EventPointerCancel
	EventScroll
	EventFocus
	EventBlur
	EventFocusIn
	EventFocusOut
	EventKeyDown
	EventInput
	EventChange
)

const (
	PointerTypeUnknown PointerType = iota
	PointerTypeMouse
	PointerTypePen
	PointerTypeTouch
)

const (
	EventPhaseNone EventPhase = iota
	EventPhaseCapture
	EventPhaseTarget
	EventPhaseBubble
)

const (
	PointerButtonsNone    PointerButtons = 0
	PointerButtonsPrimary PointerButtons = 1 << iota
	PointerButtonsSecondary
	PointerButtonsAuxiliary
	PointerButtonsBack
	PointerButtonsForward
)

type MouseButton int

const (
	MouseLeft MouseButton = 1
)

type Event struct {
	Type          EventType
	Phase         EventPhase
	X             int
	Y             int
	DeltaX        int
	DeltaY        int
	Button        MouseButton
	Buttons       PointerButtons
	PointerID     int
	PointerType   PointerType
	IsPrimary     bool
	Key           kos.KeyEvent
	Target        Node
	CurrentTarget Node
	Bubbles       bool
	Cancelable    bool

	defaultPrevented   bool
	propagationStopped bool
}

func (event *Event) PreventDefault() {
	if event == nil || !event.Cancelable {
		return
	}
	event.defaultPrevented = true
}

func (event *Event) DefaultPrevented() bool {
	if event == nil {
		return false
	}
	return event.defaultPrevented
}

func (event *Event) StopPropagation() {
	if event == nil {
		return
	}
	event.propagationStopped = true
}

func (event *Event) PropagationStopped() bool {
	if event == nil {
		return false
	}
	return event.propagationStopped
}

func pointerButtonsFromMouseInfo(info kos.MouseButtonInfo) PointerButtons {
	var buttons PointerButtons
	if info.LeftHeld {
		buttons |= PointerButtonsPrimary
	}
	if info.RightHeld {
		buttons |= PointerButtonsSecondary
	}
	if info.MiddleHeld {
		buttons |= PointerButtonsAuxiliary
	}
	if info.Button4Held {
		buttons |= PointerButtonsBack
	}
	if info.Button5Held {
		buttons |= PointerButtonsForward
	}
	return buttons
}
