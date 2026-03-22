package duit

import (
	"image"

	"9fans.net/go/draw"
)

type Label struct {
	Text  string
	Font  *draw.Font `json:"-"`
	Click func() Event `json:"-"`

	lines []string
	size  image.Point
	m     draw.Mouse
}

var _ UI = &Label{}

func (ui *Label) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Label) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	font := ui.font(dui)
	ui.lines = []string{}
	start := 0
	width := 0
	maxWidth := 0
	for i, r := range ui.Text {
		if r == '\n' {
			ui.lines = append(ui.lines, ui.Text[start:i])
			maxWidth = maximum(maxWidth, width)
			start = i + 1
			width = 0
			continue
		}
		dx := font.StringWidth(string(r))
		if width+dx > sizeAvail.X && width > 0 {
			ui.lines = append(ui.lines, ui.Text[start:i])
			maxWidth = maximum(maxWidth, width)
			start = i
			width = 0
		}
		width += dx
	}
	if start <= len(ui.Text) {
		ui.lines = append(ui.lines, ui.Text[start:])
		maxWidth = maximum(maxWidth, width)
	}
	ui.size = image.Pt(maxWidth, maximum(1, len(ui.lines))*font.Height)
	self.R = rect(ui.size)
}

func (ui *Label) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)
	font := ui.font(dui)
	p := orig
	for _, line := range ui.lines {
		img.String(p, dui.Regular.Normal.Text, image.ZP, font, line)
		p.Y += font.Height
	}
}

func propagateEvent(self *Kid, r *Result, e Event) {
	if e.NeedLayout {
		self.Layout = Dirty
	}
	if e.NeedDraw {
		self.Draw = Dirty
	}
	r.Consumed = r.Consumed || e.Consumed
}

func (ui *Label) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if m.In(rect(ui.size)) && ui.m.Buttons == 0 && m.Buttons == Button1 && ui.Click != nil {
		e := ui.Click()
		propagateEvent(self, &r, e)
	}
	ui.m = m
	return
}

func (ui *Label) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	switch k {
	case '\n':
		if ui.Click != nil {
			e := ui.Click()
			propagateEvent(self, &r, e)
		}
	case draw.KeyCmd + 'c':
		dui.WriteSnarf([]byte(ui.Text))
		r.Consumed = true
	}
	return
}

func (ui *Label) FirstFocus(dui *DUI, self *Kid) *image.Point { return nil }
func (ui *Label) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return &image.ZP
}
func (ui *Label) Mark(self *Kid, o UI, forLayout bool) bool { return self.Mark(o, forLayout) }
func (ui *Label) Print(self *Kid, indent int)               { PrintUI("Label", self, indent) }
