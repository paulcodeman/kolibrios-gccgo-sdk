package surface

import "kos"

type GradientDirection int

const (
	GradientVertical GradientDirection = iota
	GradientHorizontal
)

func (value GradientDirection) String() string {
	switch value {
	case GradientVertical:
		return "vertical"
	case GradientHorizontal:
		return "horizontal"
	default:
		return ""
	}
}

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
