package duitx

import (
	"image"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

// NewBox returns a box containing all uis in its Kids field.
func NewBox(uis ...duit.UI) *Box {
	kids := make([]*duit.Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &duit.Kid{UI: ui}
	}
	return &Box{Kids: kids}
}

// NewReverseBox returns a box containing all uis in original order in its Kids field, with the Reverse field set.
func NewReverseBox(uis ...duit.UI) *Box {
	kids := make([]*duit.Kid, len(uis))
	for i, ui := range uis {
		kids[i] = &duit.Kid{UI: ui}
	}
	return &Box{Kids: kids, Reverse: true}
}

type Display int

const (
	InlineBlock = iota
	Block
	Inline
	Flex
)

type Dir int

const (
	Row = iota + 1
	Column
)

type Boxable interface {
	Display() Display
	FlexDir() Dir
}

// Box keeps elements on a line as long as they fit, then moves on to the next line.
type Box struct {
	Kids            []*duit.Kid
	Reverse         bool
	Margin          duit.Space
	Padding         duit.Space
	AutoMarginLeft  bool
	AutoMarginRight bool
	Valign          duit.Valign
	Width           int
	Height          int
	MaxWidth        int
	ContentBox      bool
	Disp            Display
	Dir             Dir
	Background      *draw.Image `json:"-"`

	size image.Point
}

var _ duit.UI = &Box{}
var _ Boxable = &Box{}

func (ui *Box) Display() Display {
	return ui.Disp
}

func (ui *Box) FlexDir() Dir {
	return ui.Dir
}

func (ui *Box) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)
	if duit.KidsLayout(dui, self, ui.Kids, force) {
		return
	}

	if ui.Width < 0 && ui.MaxWidth > 0 {
		panic("combination ui.Width < 0 and ui.MaxWidth > 0 invalid")
	}

	padding := dui.ScaleSpace(ui.Padding)
	margin := dui.ScaleSpace(ui.Margin)

	bbw := dui.Scale(ui.Width)
	bbmaxw := dui.Scale(ui.MaxWidth)
	bbh := dui.Scale(ui.Height)

	if ui.Disp == Inline {
		bbw = 0
		bbmaxw = 0
		bbh = 0
	}

	if ui.ContentBox {
		bbw += margin.Dx() + padding.Dx()
		bbmaxw += margin.Dx() + padding.Dx()
		bbh += margin.Dy() + padding.Dy()
	}

	osize := sizeAvail
	if ui.Width > 0 && bbw < sizeAvail.X {
		sizeAvail.X = bbw
	} else if ui.MaxWidth > 0 && bbmaxw < sizeAvail.X {
		sizeAvail.X = bbmaxw
	}
	if ui.Height > 0 {
		sizeAvail.Y = bbh
	}
	sizeAvail = sizeAvail.Sub(padding.Size()).Sub(margin.Size())
	nx := 0

	cur := image.ZP
	xmax := 0
	lineY := 0

	fixValign := func(kids []*duit.Kid) {
		if len(kids) < 2 {
			return
		}
		for _, k := range kids {
			switch ui.Valign {
			case duit.ValignTop:
			case duit.ValignMiddle:
				k.R = k.R.Add(image.Pt(0, (lineY-k.R.Dy())/2))
			case duit.ValignBottom:
				k.R = k.R.Add(image.Pt(0, lineY-k.R.Dy()))
			}
		}
	}

	autoMarginOffset := func(availWidth int, childWidth int, autoLeft bool, autoRight bool) int {
		if availWidth <= childWidth {
			return 0
		}
		switch {
		case autoLeft && autoRight:
			return (availWidth - childWidth) / 2
		case autoLeft:
			return availWidth - childWidth
		default:
			return 0
		}
	}

	for i, k := range ui.Kids {
		k.UI.Layout(dui, k, sizeAvail.Sub(image.Pt(0, cur.Y+lineY)), true)
		childSize := k.R.Size()
		var kr image.Rectangle
		var shouldCol bool
		if ui.Disp == Flex {
			shouldCol = ui.Dir == Column
		} else if display(k) == Block {
			shouldCol = true
		}
		if (nx == 0 || cur.X+childSize.X <= sizeAvail.X) && !shouldCol {
			kr = rect(childSize).Add(cur).Add(padding.Topleft())
			cur.X += childSize.X
			lineY = maximum(lineY, childSize.Y)
			nx += 1
		} else {
			if nx > 0 {
				fixValign(ui.Kids[i-nx : i])
				cur.X = 0
				cur.Y += lineY + margin.Topleft().Y
			}
			kr = rect(childSize).Add(cur).Add(padding.Topleft())
			nx = 1
			cur.X = childSize.X
			lineY = childSize.Y
		}
		if auto, ok := k.UI.(interface{ AutoMargin() (bool, bool) }); ok {
			autoLeft, autoRight := auto.AutoMargin()
			if autoLeft || autoRight {
				kr = kr.Add(image.Pt(autoMarginOffset(sizeAvail.X, childSize.X, autoLeft, autoRight), 0))
			}
		}
		k.R = kr
		occupiedX := kr.Max.X - padding.Left
		if xmax < occupiedX {
			xmax = occupiedX
		}
	}
	fixValign(ui.Kids[len(ui.Kids)-nx : len(ui.Kids)])
	cur.Y += lineY

	if ui.Reverse {
		bottomY := cur.Y + padding.Dy()
		for _, k := range ui.Kids {
			y1 := bottomY - k.R.Min.Y
			y0 := y1 - k.R.Dy()
			k.R = image.Rect(k.R.Min.X, y0, k.R.Max.X, y1)
		}
	}

	ui.size = image.Pt(xmax, cur.Y).Add(padding.Size())
	if ui.Width < 0 {
		ui.size.X = osize.X
	} else if ui.Disp == Block || ui.Disp == Flex {
		if minWidth := sizeAvail.X + padding.Dx(); ui.size.X < minWidth {
			ui.size.X = minWidth
		}
	}
	if ui.Height < 0 && ui.size.Y < osize.Y {
		ui.size.Y = osize.Y
	}
	self.R = rect(ui.size.Add(margin.Size()))
}

func display(k *duit.Kid) (d Display) {
	if b, ok := k.UI.(Boxable); ok {
		return b.Display()
	}
	return InlineBlock
}

func (ui *Box) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	margin := dui.ScaleSpace(ui.Margin)
	orig = orig.Add(margin.Topleft())
	duit.KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}

func (ui *Box) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	margin := dui.ScaleSpace(ui.Margin)
	origM.Point = origM.Point.Sub(margin.Topleft())
	m.Point = m.Point.Sub(margin.Topleft())
	return duit.KidsMouse(dui, self, ui.Kids, m, origM, orig)
}

func (ui *Box) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	if dui != nil {
		margin := dui.ScaleSpace(ui.Margin)
		m.Point = m.Point.Sub(margin.Topleft())
	}
	return duit.KidsKey(dui, self, ui.orderedKids(), k, m, orig)
}

func (ui *Box) orderedKids() []*duit.Kid {
	if !ui.Reverse {
		return ui.Kids
	}
	n := len(ui.Kids)
	kids := make([]*duit.Kid, n)
	for i := range ui.Kids {
		kids[i] = ui.Kids[n-1-i]
	}
	return kids
}

func (ui *Box) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	return duit.KidsFirstFocus(dui, self, ui.orderedKids())
}

func (ui *Box) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	return duit.KidsFocus(dui, self, ui.orderedKids(), o)
}

func (ui *Box) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	return duit.KidsMark(self, ui.Kids, o, forLayout)
}

func (ui *Box) Print(self *duit.Kid, indent int) {
	duit.PrintUI("Box", self, indent)
	for _, k := range ui.Kids {
		k.UI.Print(k, indent+1)
	}
}
