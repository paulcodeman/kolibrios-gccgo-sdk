package duit

import (
	"image"

	"9fans.net/go/draw"
)

func NewBox(uis ...UI) *Box {
	return &Box{Kids: NewKids(uis...)}
}

type Box struct {
	Kids       []*Kid
	Reverse    bool
	Margin     image.Point
	Padding    Space
	Valign     Valign
	Width      int
	Height     int
	MaxWidth   int
	Background *draw.Image `json:"-"`

	size image.Point
}

var _ UI = &Box{}

func (ui *Box) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	if KidsLayout(dui, self, ui.Kids, force) {
		return
	}
	padding := dui.ScaleSpace(ui.Padding)
	margin := image.Pt(dui.Scale(ui.Margin.X), dui.Scale(ui.Margin.Y))
	if ui.Width > 0 && dui.Scale(ui.Width) < sizeAvail.X {
		sizeAvail.X = dui.Scale(ui.Width)
	} else if ui.MaxWidth > 0 && dui.Scale(ui.MaxWidth) < sizeAvail.X {
		sizeAvail.X = dui.Scale(ui.MaxWidth)
	}
	if ui.Height > 0 {
		sizeAvail.Y = dui.Scale(ui.Height)
	}
	content := sizeAvail.Sub(padding.Size())
	cur := image.ZP
	xmax := 0
	lineY := 0
	lineStart := 0
	flushLine := func(end int) {
		if end-lineStart < 2 {
			return
		}
		for _, k := range ui.Kids[lineStart:end] {
			switch ui.Valign {
			case ValignMiddle:
				k.R = k.R.Add(image.Pt(0, (lineY-k.R.Dy())/2))
			case ValignBottom:
				k.R = k.R.Add(image.Pt(0, lineY-k.R.Dy()))
			}
		}
	}
	for i, k := range ui.Kids {
		k.UI.Layout(dui, k, content.Sub(image.Pt(0, cur.Y+lineY)), true)
		childSize := k.R.Size()
		if cur.X > 0 && cur.X+childSize.X > content.X {
			flushLine(i)
			cur.X = 0
			cur.Y += lineY + margin.Y
			lineY = 0
			lineStart = i
		}
		k.R = rect(childSize).Add(cur).Add(padding.Topleft())
		cur.X += childSize.X + margin.X
		lineY = maximum(lineY, childSize.Y)
		xmax = maximum(xmax, cur.X)
	}
	flushLine(len(ui.Kids))
	cur.Y += lineY
	ui.size = image.Pt(maximum(0, xmax-margin.X), cur.Y).Add(padding.Size())
	if ui.Width < 0 && ui.size.X < sizeAvail.X {
		ui.size.X = sizeAvail.X
	}
	if ui.Height < 0 && ui.size.Y < sizeAvail.Y {
		ui.size.Y = sizeAvail.Y
	}
	self.R = rect(ui.size)
}

func (ui *Box) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	KidsDraw(dui, self, ui.Kids, ui.size, ui.Background, img, orig, m, force)
}
func (ui *Box) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) Result {
	return KidsMouse(dui, self, ui.Kids, m, origM, orig)
}
func (ui *Box) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) Result {
	return KidsKey(dui, self, ui.Kids, k, m, orig)
}
func (ui *Box) FirstFocus(dui *DUI, self *Kid) *image.Point { return KidsFirstFocus(dui, self, ui.Kids) }
func (ui *Box) Focus(dui *DUI, self *Kid, o UI) *image.Point { return KidsFocus(dui, self, ui.Kids, o) }
func (ui *Box) Mark(self *Kid, o UI, forLayout bool) bool    { return KidsMark(self, ui.Kids, o, forLayout) }
func (ui *Box) Print(self *Kid, indent int)                  { PrintUI("Box", self, indent); KidsPrint(ui.Kids, indent+1) }
