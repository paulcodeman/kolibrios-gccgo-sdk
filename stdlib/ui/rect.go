package ui

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

func (rect Rect) Empty() bool {
	return rect.Width <= 0 || rect.Height <= 0
}

func (rect Rect) Contains(x int, y int) bool {
	if rect.Empty() {
		return false
	}
	return x >= rect.X && y >= rect.Y &&
		x < rect.X+rect.Width && y < rect.Y+rect.Height
}

func UnionRect(a Rect, b Rect) Rect {
	if a.Empty() {
		return b
	}
	if b.Empty() {
		return a
	}
	left := a.X
	if b.X < left {
		left = b.X
	}
	top := a.Y
	if b.Y < top {
		top = b.Y
	}
	right := a.X + a.Width
	if b.X+b.Width > right {
		right = b.X + b.Width
	}
	bottom := a.Y + a.Height
	if b.Y+b.Height > bottom {
		bottom = b.Y + b.Height
	}
	return Rect{
		X:      left,
		Y:      top,
		Width:  right - left,
		Height: bottom - top,
	}
}

func IntersectRect(a Rect, b Rect) Rect {
	if a.Empty() || b.Empty() {
		return Rect{}
	}
	left := a.X
	if b.X > left {
		left = b.X
	}
	top := a.Y
	if b.Y > top {
		top = b.Y
	}
	right := a.X + a.Width
	if b.X+b.Width < right {
		right = b.X + b.Width
	}
	bottom := a.Y + a.Height
	if b.Y+b.Height < bottom {
		bottom = b.Y + b.Height
	}
	if right <= left || bottom <= top {
		return Rect{}
	}
	return Rect{
		X:      left,
		Y:      top,
		Width:  right - left,
		Height: bottom - top,
	}
}
