package ui

import "kos"

type Node interface {
	DrawTo(*Canvas)
	Bounds() Rect
	Handle(Event) bool
}

type DirtyAware interface {
	Dirty() bool
	ClearDirty()
}

type LayoutAware interface {
	Layout(*Canvas)
}

type LayoutDirtyAware interface {
	LayoutDirty() bool
}

type VisualBoundsAware interface {
	VisualBounds() Rect
}

type HoverAware interface {
	SetHover(bool) bool
}

type ActiveAware interface {
	SetActive(bool) bool
}

type FocusAware interface {
	SetFocus(bool) bool
	Focused() bool
}

type KeyAware interface {
	HandleKey(kos.KeyEvent) bool
}

type ScrollAware interface {
	HandleScroll(deltaX int, deltaY int) bool
}
