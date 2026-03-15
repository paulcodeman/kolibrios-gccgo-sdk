package ui

type EventType int

const (
	EventClick EventType = 1
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
