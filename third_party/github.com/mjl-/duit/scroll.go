package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

type Scroll struct {
	Kid    Kid
	Height int

	r             image.Rectangle
	barR          image.Rectangle
	barActiveR    image.Rectangle
	childR        image.Rectangle
	offset        int
	img           *draw.Image
	scrollbarSize int
	lastMouseUI   UI
}

var _ UI = &Scroll{}

func NewScroll(ui UI) *Scroll {
	return &Scroll{Height: -1, Kid: Kid{UI: ui}}
}

func (ui *Scroll) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	if self.Layout == Clean && !force {
		return
	}
	self.Layout = Clean
	self.Draw = Dirty
	ui.scrollbarSize = dui.Scale(ScrollbarSize)
	scaledHeight := dui.Scale(ui.Height)
	if scaledHeight > 0 && scaledHeight < sizeAvail.Y {
		sizeAvail.Y = scaledHeight
	}
	ui.r = rect(sizeAvail)
	ui.barR = ui.r
	ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
	ui.childR = ui.r
	ui.childR.Min.X = ui.barR.Max.X
	ui.Kid.UI.Layout(dui, &ui.Kid, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy()), force)
	ui.Kid.Layout = Clean
	ui.Kid.Draw = Dirty
	if ui.r.Dy() > ui.Kid.R.Dy() && ui.Height == 0 {
		ui.barR.Max.Y = ui.Kid.R.Dy()
		ui.r.Max.Y = ui.Kid.R.Dy()
		ui.childR.Max.Y = ui.Kid.R.Dy()
	}
	self.R = rect(ui.r.Size())
}

func (ui *Scroll) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)
	if self.Draw == Clean {
		return
	}
	drawState := self.Draw
	self.Draw = Clean
	if ui.r.Empty() {
		return
	}
	ui.scroll(0)
	barHover := m.In(ui.barR)
	bg := dui.ScrollBGNormal
	vis := dui.ScrollVisibleNormal
	if barHover {
		bg = dui.ScrollBGHover
		vis = dui.ScrollVisibleHover
	}
	h := ui.r.Dy()
	uih := ui.Kid.R.Dy()
	if uih > h {
		barR := ui.barR.Add(orig)
		img.Draw(barR, bg, nil, image.ZP)
		barH := h * h / uih
		barY := ui.offset * h / uih
		ui.barActiveR = ui.barR
		ui.barActiveR.Min.Y += barY
		ui.barActiveR.Max.Y = ui.barActiveR.Min.Y + barH
		barActiveR := ui.barActiveR.Add(orig)
		barActiveR.Max.X--
		img.Draw(barActiveR, vis, nil, image.ZP)
	}
	if ui.childR.Empty() {
		return
	}
	if ui.img == nil || ui.Kid.R.Size() != ui.img.R.Size() {
		var err error
		if ui.img != nil {
			ui.img.Free()
		}
		ui.Kid.Draw = Dirty
		if ui.Kid.R.Dx() == 0 || ui.Kid.R.Dy() == 0 {
			return
		}
		ui.img, err = dui.Display.AllocImage(ui.Kid.R, draw.ARGB32, false, dui.BackgroundColor)
		if dui.error(err, "allocimage") {
			return
		}
	} else if ui.Kid.Draw == Dirty {
		ui.img.Draw(ui.img.R, dui.Background, nil, image.ZP)
	}
	m.Point = m.Point.Add(image.Pt(-ui.childR.Min.X, ui.offset))
	childDraw := ui.Kid.Draw
	childDirty := image.ZR
	if childDraw == DirtyKid {
		childDirty, _ = dirtyBounds(&ui.Kid, image.ZP)
	}
	if ui.Kid.Draw != Clean {
		if force {
			ui.Kid.Draw = Dirty
		}
		ui.Kid.UI.Draw(dui, &ui.Kid, ui.img, image.ZP, m, ui.Kid.Draw == Dirty)
		ui.Kid.Draw = Clean
	}
	if !force && drawState == DirtyKid && childDraw == DirtyKid {
		view := childDirty.Intersect(image.Rect(0, ui.offset, ui.childR.Dx(), ui.offset+ui.childR.Dy()))
		if !view.Empty() {
			target := view.Add(orig.Add(ui.childR.Min).Sub(image.Pt(0, ui.offset)))
			img.Draw(target, ui.img, nil, view.Min)
		}
		return
	}
	img.Draw(ui.childR.Add(orig), ui.img, nil, image.Pt(0, ui.offset))
}

func (ui *Scroll) scroll(delta int) bool {
	prev := ui.offset
	ui.offset += delta
	ui.offset = maximum(0, ui.offset)
	ui.offset = minimum(ui.offset, maximum(0, ui.Kid.R.Dy()-ui.childR.Dy()))
	return prev != ui.offset
}

func (ui *Scroll) scrollKey(k rune) bool {
	switch k {
	case draw.KeyUp:
		return ui.scroll(-50)
	case draw.KeyDown:
		return ui.scroll(50)
	case draw.KeyPageUp:
		return ui.scroll(-200)
	case draw.KeyPageDown:
		return ui.scroll(200)
	}
	return false
}

func (ui *Scroll) scrollMouse(m draw.Mouse, scrollOnly bool) bool {
	switch m.Buttons {
	case Button4:
		return ui.scroll(-m.Y / 4)
	case Button5:
		return ui.scroll(m.Y / 4)
	}
	if scrollOnly {
		return false
	}
	switch m.Buttons {
	case Button1:
		return ui.scroll(-m.Y)
	case Button2:
		offset := m.Y * ui.Kid.R.Dy() / ui.barR.Dy()
		offset = maximum(0, minimum(offset, ui.Kid.R.Dy()-ui.childR.Dy()))
		prev := ui.offset
		ui.offset = offset
		return prev != ui.offset
	case Button3:
		return ui.scroll(m.Y)
	}
	return false
}

func (ui *Scroll) result(dui *DUI, self *Kid, r *Result, scrolled bool) {
	if ui.Kid.Layout != Clean {
		nk := ui.Kid
		ui.Kid.UI.Layout(dui, &nk, ui.childR.Size(), false)
		if nk.R.Size() != ui.Kid.R.Size() {
			self.Layout = Dirty
			self.Draw = Dirty
			return
		}
		ui.Kid.Layout = Clean
		ui.Kid.Draw = Dirty
		if self.Draw == Clean {
			self.Draw = DirtyKid
		}
		return
	}
	if scrolled {
		self.Draw = Dirty
		return
	}
	if ui.Kid.Draw != Clean && self.Draw == Clean {
		self.Draw = DirtyKid
	}
}

func (ui *Scroll) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollMouse(m, false)
		self.Draw = Dirty
		return
	}
	if m.Point.In(ui.childR) {
		nOrigM := origM
		nOrigM.Point = nOrigM.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		nm := m
		nm.Point = nm.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r = ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, image.ZP)
		ui.warpScroll(dui, self, r.Warp, orig)
		scrolled := false
		if !r.Consumed {
			scrolled = ui.scrollMouse(m, true)
			r.Consumed = scrolled
		}
		ui.result(dui, self, &r, scrolled)
		if r.Hit != ui.lastMouseUI {
			if r.Hit != nil {
				ui.Mark(self, r.Hit, false)
			}
			if ui.lastMouseUI != nil {
				ui.Mark(self, ui.lastMouseUI, false)
			}
		}
		ui.lastMouseUI = r.Hit
	}
	return
}

func (ui *Scroll) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollKey(k)
		if r.Consumed {
			self.Draw = Dirty
		}
	}
	if m.Point.In(ui.childR) {
		m.Point = m.Point.Add(image.Pt(-ui.scrollbarSize, ui.offset))
		r = ui.Kid.UI.Key(dui, &ui.Kid, k, m, image.ZP)
		ui.warpScroll(dui, self, r.Warp, orig)
		scrolled := false
		if !r.Consumed {
			scrolled = ui.scrollKey(k)
			r.Consumed = scrolled
		}
		ui.result(dui, self, &r, scrolled)
	}
	return
}

func (ui *Scroll) warpScroll(dui *DUI, self *Kid, warp *image.Point, orig image.Point) {
	if warp == nil {
		return
	}
	prev := ui.offset
	if warp.Y < ui.offset {
		ui.offset = maximum(0, warp.Y-dui.Scale(40))
	} else if warp.Y > ui.offset+ui.r.Dy() {
		ui.offset = minimum(ui.Kid.R.Dy()-ui.r.Dy(), warp.Y+dui.Scale(40)-ui.r.Dy())
	}
	if prev != ui.offset {
		if self != nil {
			self.Draw = Dirty
		} else {
			dui.MarkDraw(ui)
		}
	}
	warp.Y -= ui.offset
	warp.X += orig.X + ui.scrollbarSize
	warp.Y += orig.Y
}

func (ui *Scroll) focusPoint(p *image.Point) *image.Point {
	if p == nil {
		return nil
	}
	pp := p.Add(ui.childR.Min)
	return &pp
}

func (ui *Scroll) FirstFocus(dui *DUI, self *Kid) *image.Point {
	return ui.focusPoint(ui.Kid.UI.FirstFocus(dui, &ui.Kid))
}

func (ui *Scroll) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o == ui {
		p := image.Pt(minimum(ui.scrollbarSize/2, ui.r.Dx()), minimum(ui.scrollbarSize/2, ui.r.Dy()))
		return &p
	}
	return ui.focusPoint(ui.Kid.UI.Focus(dui, &ui.Kid, o))
}

func (ui *Scroll) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	if self.Mark(o, forLayout) {
		return true
	}
	marked = ui.Kid.UI.Mark(&ui.Kid, o, forLayout)
	if marked {
		if forLayout {
			if self.Layout == Clean {
				self.Layout = DirtyKid
			}
		} else if self.Draw == Clean {
			self.Draw = DirtyKid
		}
	}
	return
}

func (ui *Scroll) Print(self *Kid, indent int) {
	PrintUI(fmt.Sprintf("Scroll offset=%d childR=%v", ui.offset, ui.childR), self, indent)
	ui.Kid.UI.Print(&ui.Kid, indent+1)
}
