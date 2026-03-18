package ui

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
)

type MouseButton int

const (
	MouseLeft MouseButton = 1
)

type Event struct {
	Type   EventType
	X      int
	Y      int
	Button MouseButton
	Target Node
}
