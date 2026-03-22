package duitx

import (
	"fmt"
	"image"
	"math"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
)

var (
	selectedBg *draw.Image
)

// Label draws multiline text in a single font.
//
// Keys:
// cmd-c, copy text
// \n, like button1 click, calls the Click function
type Label struct {
	Text     string     // Text to draw, wrapped at glyph boundary.
	Font     *draw.Font `json:"-"`
	LineH    int
	Click    func() (e duit.Event) `json:"-"`
	Selected bool

	orig image.Point
	size image.Point
	m    draw.Mouse
}

var _ duit.UI = &Label{}

func (ui *Label) font(dui *duit.DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Label) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)
	if ui.Text == "" || ui.size != image.ZP {
		return
	}

	font := ui.font(dui)
	ui.size = image.Pt(font.StringWidth(ui.Text), ui.lineHeight(font))
	self.R = rect(ui.size)
}

func (ui *Label) lineHeight(font *draw.Font) int {
	if ui.LineH > 0 {
		return ui.LineH
	}
	return int(math.Ceil(float64(font.Height) * 1.2))
}

func (ui *Label) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	debugDraw(dui, self)

	if selectedBg == nil {
		var err error
		selectedBg, err = dui.Display.AllocImage(image.Rect(0, 0, 10, 10), draw.ARGB32, true, 0x9acd32ff)
		if err != nil {
			panic(fmt.Errorf("%v", err))
		}
	}

	font := ui.font(dui)
	if ui.Selected {
		img.StringBg(orig, dui.Regular.Normal.Text, image.ZP, font, ui.Text, selectedBg, image.ZP)
	} else {
		img.String(orig, dui.Regular.Normal.Text, image.ZP, font, ui.Text)
	}
	ui.orig = orig
}

func (ui *Label) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	if m.In(rect(ui.size)) && ui.m.Buttons == 0 && m.Buttons == duit.Button1 && ui.Click != nil {
		e := ui.Click()
		propagateEvent(self, &r, e)
	}
	ui.m = m
	return
}

func (ui *Label) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
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

func (ui *Label) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	return nil
}

func (ui *Label) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	if ui != o {
		return nil
	}
	return &image.ZP
}

func (ui *Label) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Label) Print(self *duit.Kid, indent int) {
	duit.PrintUI("Label", self, indent)
}

func propagateEvent(self *duit.Kid, r *duit.Result, e duit.Event) {
	if e.NeedLayout {
		self.Layout = duit.Dirty
	}
	if e.NeedDraw {
		self.Draw = duit.Dirty
	}
	r.Consumed = e.Consumed || r.Consumed
}

func (ui *Label) Rect() draw.Rectangle {
	if ui == nil {
		return draw.Rectangle{}
	}
	return draw.Rectangle{
		ui.orig,
		ui.orig.Add(ui.size),
	}
}
