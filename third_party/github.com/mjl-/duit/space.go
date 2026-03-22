package duit

import "image"

type Space struct {
	Top, Right, Bottom, Left int
}

func (s Space) Dx() int {
	return s.Left + s.Right
}

func (s Space) Dy() int {
	return s.Top + s.Bottom
}

func (s Space) Size() image.Point {
	return image.Pt(s.Dx(), s.Dy())
}

func (s Space) Mul(n int) Space {
	s.Top *= n
	s.Right *= n
	s.Bottom *= n
	s.Left *= n
	return s
}

func (s Space) Topleft() image.Point {
	return image.Pt(s.Left, s.Top)
}

func (s Space) Inset(r image.Rectangle) image.Rectangle {
	return image.Rect(r.Min.X+s.Left, r.Min.Y+s.Top, r.Max.X-s.Right, r.Max.Y-s.Bottom)
}

func SpaceXY(x int, y int) Space {
	return Space{Top: y, Right: x, Bottom: y, Left: x}
}

func SpacePt(p image.Point) Space {
	return Space{Top: p.Y, Right: p.X, Bottom: p.Y, Left: p.X}
}

func NSpace(n int, space Space) []Space {
	values := make([]Space, n)
	for i := range values {
		values[i] = space
	}
	return values
}
