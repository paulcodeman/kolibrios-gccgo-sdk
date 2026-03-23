package core

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
	From      uint32
	To        uint32
	Direction GradientDirection
}

type Shadow struct {
	OffsetX int
	OffsetY int
	Blur    int
	Color   uint32
	Alpha   uint8
}
