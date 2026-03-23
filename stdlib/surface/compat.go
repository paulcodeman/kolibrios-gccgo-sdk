package surface

import (
	"kos"
	"surface/core"
)

const (
	DefaultCharWidth  = core.DefaultCharWidth
	DefaultFontHeight = core.DefaultFontHeight
)

const (
	GradientVertical   = core.GradientVertical
	GradientHorizontal = core.GradientHorizontal
)

type (
	Rect              = core.Rect
	GradientDirection = core.GradientDirection
	CornerRadii       = core.CornerRadii
)

type Gradient struct {
	From      kos.Color
	To        kos.Color
	Direction GradientDirection
}

type Shadow struct {
	OffsetX int
	OffsetY int
	Blur    int
	Color   kos.Color
	Alpha   uint8
}

type Presenter struct {
	core.Presenter
}

func NewPresenter(x int, y int, width int, height int, title string) Presenter {
	return Presenter{Presenter: core.NewPresenter(x, y, width, height, title)}
}

func (presenter Presenter) PresentFull(buffer *Buffer) {
	presenter.Presenter.PresentFull(rawBuffer(buffer))
}

func (presenter Presenter) PresentClient(buffer *Buffer) {
	presenter.Presenter.PresentClient(rawBuffer(buffer))
}

func (presenter Presenter) PresentRect(buffer *Buffer, rect Rect) {
	presenter.Presenter.PresentRect(rawBuffer(buffer), rect)
}

func WindowClientRect(width int, height int) Rect {
	return core.WindowClientRect(width, height)
}

func UnionRect(a Rect, b Rect) Rect {
	return core.UnionRect(a, b)
}

func IntersectRect(a Rect, b Rect) Rect {
	return core.IntersectRect(a, b)
}
