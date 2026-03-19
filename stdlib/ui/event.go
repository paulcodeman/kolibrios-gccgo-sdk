package ui

import "kos"

type EventType int
type EventPhase uint8

const (
	EventClick EventType = 1
	EventMouseDown
	EventMouseUp
	EventMouseMove
	EventMouseEnter
	EventMouseLeave
	EventScroll
	EventFocus
	EventBlur
	EventKeyDown
	EventInput
	EventChange
)

const (
	EventPhaseNone EventPhase = iota
	EventPhaseTarget
	EventPhaseBubble
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
