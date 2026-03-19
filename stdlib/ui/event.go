package ui

import "kos"

type EventType int

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

type MouseButton int

const (
	MouseLeft MouseButton = 1
)

type Event struct {
	Type   EventType
	X      int
	Y      int
	DeltaX int
	DeltaY int
	Button MouseButton
	Key    kos.KeyEvent
	Target Node
}
