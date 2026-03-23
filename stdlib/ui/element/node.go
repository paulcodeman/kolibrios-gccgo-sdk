package ui

import "kos"

type Node interface {
	DrawTo(*Canvas)
	Bounds() Rect
	Handle(Event) bool
}

type windowAware interface {
	setWindow(*Window)
}

type OffsetDrawAware interface {
	DrawToOffset(*Canvas, int)
}

type DirtyAware interface {
	Dirty() bool
	ClearDirty()
}

type LayoutAware interface {
	Layout(*Canvas)
}

type LayoutContextAware interface {
	LayoutWithContext(LayoutContext)
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

type MouseMoveAware interface {
	HandleMouseMove(x int, y int, buttons PointerButtons) bool
}

type MouseDownAware interface {
	HandleMouseDown(x int, y int, button MouseButton, buttons PointerButtons) bool
}

type MouseUpAware interface {
	HandleMouseUp(x int, y int, button MouseButton, buttons PointerButtons) bool
}

type FocusAware interface {
	SetFocus(bool) bool
	Focused() bool
}

type TabAware interface {
	HandleTab(shift bool) bool
}

type KeyAware interface {
	HandleKey(kos.KeyEvent) bool
}

type ScrollAware interface {
	HandleScroll(deltaX int, deltaY int) bool
}
