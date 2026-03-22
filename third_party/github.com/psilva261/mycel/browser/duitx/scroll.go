package duitx

import (
	"fmt"
	"image"
	"time"

	"9fans.net/go/draw"
	"github.com/mjl-/duit"
	log "github.com/psilva261/mycel/logger"
)

const maxAge = time.Minute

// Scroll shows a part of its single child, typically a box, and lets you scroll the content.
type Scroll struct {
	Kid    duit.Kid
	Height int

	r             image.Rectangle
	barR          image.Rectangle
	barActiveR    image.Rectangle
	childR        image.Rectangle
	Offset        int
	img           *draw.Image
	scrollbarSize int
	lastMouseUI   duit.UI
	drawOffset    int

	tiles        []scrollTile
	tilesChanged bool
}

type scrollTile struct {
	index int
	img   *draw.Image
	last  time.Time
}

var _ duit.UI = &Scroll{}

// NewScroll returns a full-height scroll bar containing ui.
func NewScroll(dui *duit.DUI, ui duit.UI) *Scroll {
	s := &Scroll{
		Height: -1,
		Kid:    duit.Kid{UI: ui},
	}
	_ = dui
	return s
}

func (ui *Scroll) Free() {
	if ui.img != nil {
		ui.img.Free()
		ui.img = nil
	}
	for i := range ui.tiles {
		if ui.tiles[i].img != nil {
			ui.tiles[i].img.Free()
			ui.tiles[i].img = nil
		}
	}
	ui.tiles = ui.tiles[:0]
}

func (ui *Scroll) tileAt(index int) (pos int, tile *scrollTile) {
	for i := range ui.tiles {
		if ui.tiles[i].index == index {
			return i, &ui.tiles[i]
		}
	}
	return -1, nil
}

func (ui *Scroll) dropTile(index int, freeImg bool) {
	pos, tile := ui.tileAt(index)
	if pos < 0 {
		return
	}
	if freeImg && tile != nil && tile.img != nil {
		tile.img.Free()
	}
	last := len(ui.tiles) - 1
	copy(ui.tiles[pos:], ui.tiles[pos+1:])
	ui.tiles[last] = scrollTile{}
	ui.tiles = ui.tiles[:last]
}

func (ui *Scroll) setTile(index int, img *draw.Image, last time.Time) {
	pos, tile := ui.tileAt(index)
	if pos >= 0 {
		if tile.img != nil && tile.img != img {
			tile.img.Free()
		}
		ui.tiles[pos].img = img
		ui.tiles[pos].last = last
		return
	}
	ui.tiles = append(ui.tiles, scrollTile{
		index: index,
		img:   img,
		last:  last,
	})
}

func tileDistance(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func (ui *Scroll) freeCur() {
	i, of := ui.pos()
	_, tile := ui.tileAt(i)
	_, tile1 := ui.tileAt(i + 1)
	if !ui.tilesChanged && (tile == nil || ui.sizeOk(tile.img)) && (of == 0 || tile1 == nil || ui.sizeOk(tile1.img)) {
		return
	}
	ui.dropTile(i, true)
	if of > 0 {
		ui.dropTile(i+1, true)
	}
	ui.tilesChanged = false
}

func (ui *Scroll) sizeOk(tl *draw.Image) bool {
	return tl != nil && tl.R.Dx() == ui.r.Dx() && tl.R.Dy() == ui.r.Dy()
}

func (ui *Scroll) ensure(dui *duit.DUI, i int) {
	log.Printf("ensure(dui, %v)", i)
	_, tile := ui.tileAt(i)
	if tile != nil && time.Since(tile.last) < maxAge && ui.sizeOk(tile.img) {
		return
	}

	log.Printf("ensure(dui, %v): draw", i)
	r := ui.r.Add(image.Point{X: 0, Y: i * ui.r.Dy()})
	img, err := dui.Display.AllocImage(r, draw.ARGB32, false, dui.BackgroundColor)
	if duitError(dui, err, "allocimage") {
		return
	}
	ui.Kid.UI.Draw(dui, &ui.Kid, img, image.ZP, draw.Mouse{}, true)

	log.Printf("ensure: ui.tiles[%d] = img(R=%+v, ...)", i, img.R)
	ui.setTile(i, img, time.Now())

	for idx := 0; idx < len(ui.tiles); {
		if tileDistance(i, ui.tiles[idx].index) > 5 {
			ui.dropTile(ui.tiles[idx].index, true)
			continue
		}
		idx++
	}
}

func (ui *Scroll) pos() (t, of int) {
	t = ui.Offset / ui.r.Dy()
	of = ui.Offset % ui.r.Dy()
	return
}

func (ui *Scroll) tlR(i int) (r image.Rectangle) {
	r.Min.X = ui.r.Min.X
	r.Max.X = ui.r.Max.X
	r.Min.Y = ui.r.Min.Y + i*ui.r.Dy()
	r.Max.Y = r.Min.Y + ui.r.Dy()
	return
}

func (ui *Scroll) Layout(dui *duit.DUI, self *duit.Kid, sizeAvail image.Point, force bool) {
	debugLayout(dui, self)

	if self.Layout == duit.Clean && !force {
		return
	}
	self.Layout = duit.Clean
	self.Draw = duit.Dirty

	ui.scrollbarSize = dui.Scale(duit.ScrollbarSize)
	scaledHeight := dui.Scale(ui.Height)
	if scaledHeight > 0 && scaledHeight < sizeAvail.Y {
		sizeAvail.Y = scaledHeight
	}
	ui.r = rect(sizeAvail)
	ui.childR = ui.r
	ui.barR = image.ZR

	ui.Kid.UI.Layout(dui, &ui.Kid, ui.childR.Size(), force)
	ui.Kid.Layout = duit.Clean
	ui.Kid.Draw = duit.Dirty

	kY := ui.Kid.R.Dy()
	if kY > ui.r.Dy() {
		ui.barR = ui.r
		ui.barR.Max.X = ui.barR.Min.X + ui.scrollbarSize
		ui.childR = ui.r
		ui.childR.Min.X = ui.barR.Max.X
		ui.Kid.UI.Layout(dui, &ui.Kid, image.Pt(ui.r.Dx()-ui.barR.Dx(), ui.r.Dy()), force)
		ui.Kid.Layout = duit.Clean
		ui.Kid.Draw = duit.Dirty
		kY = ui.Kid.R.Dy()
	}
	if ui.r.Dy() > kY && ui.Height == 0 {
		if !ui.barR.Empty() {
			ui.barR.Max.Y = kY
		}
		ui.r.Max.Y = kY
		ui.childR.Max.Y = kY
	}
	self.R = rect(ui.r.Size())
	ui.Free()
}

func (ui *Scroll) Draw(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	debugDraw(dui, self)

	if self.Draw == duit.Clean {
		return
	} else {
		log.Printf("Draw: self.Draw=%v is not clean, force=%v", self.Draw, force)
	}

	if ui.r.Empty() {
		self.Draw = duit.Clean
		return
	}

	ui.scroll(0)
	ui.drawBar(dui, self, img, orig, m, force)
	ui.drawChild(dui, self, img, orig, m, force)
	self.Draw = duit.Clean
}

func (ui *Scroll) drawBar(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	if ui.barR.Empty() {
		return
	}
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
		barY := ui.Offset * h / uih
		ui.barActiveR = ui.barR
		ui.barActiveR.Min.Y += barY
		ui.barActiveR.Max.Y = ui.barActiveR.Min.Y + barH
		barActiveR := ui.barActiveR.Add(orig)
		barActiveR.Max.X -= 1
		img.Draw(barActiveR, vis, nil, image.ZP)
	}
}

func (ui *Scroll) drawChild(dui *duit.DUI, self *duit.Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	if ui.childR.Empty() {
		return
	}

	if ui.img == nil || ui.img.R.Size() != ui.Kid.R.Size() {
		var err error
		if ui.img != nil {
			ui.img.Free()
			ui.img = nil
		}
		ui.Kid.Draw = duit.Dirty
		if ui.Kid.R.Dx() == 0 || ui.Kid.R.Dy() == 0 {
			return
		}
		ui.img, err = dui.Display.AllocImage(rect(ui.Kid.R.Size()), draw.ARGB32, false, dui.BackgroundColor)
		if duitError(dui, err, "allocimage") {
			return
		}
	}

	if force || ui.Kid.Draw != duit.Clean {
		log.Printf("drawChild: buffered refresh: ui.Kid.Draw=%v force=%v", ui.Kid.Draw, force)
		ui.img.Draw(ui.img.R, dui.Background, nil, image.ZP)
		nm := m
		nm.Point = nm.Point.Add(image.Pt(-ui.childR.Min.X, ui.Offset))
		ui.Kid.UI.Draw(dui, &ui.Kid, ui.img, image.ZP, nm, true)
		ui.Kid.Draw = duit.Clean
	}

	img.Draw(ui.childR.Add(orig), ui.img, nil, image.Pt(0, ui.Offset))
}

func (ui *Scroll) scroll(delta int) (changed bool) {
	o := ui.Offset
	ui.Offset += delta
	ui.Offset = maximum(0, ui.Offset)
	ui.Offset = minimum(ui.Offset, maximum(0, ui.Kid.R.Dy()-ui.childR.Dy()))
	return o != ui.Offset
}

func (ui *Scroll) scrollKey(k rune) (consumed bool) {
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

func (ui *Scroll) scrollMouse(m draw.Mouse, scrollOnly bool) (consumed bool) {
	switch m.Buttons {
	case duit.Button4:
		return ui.scroll(-m.Y / 4)
	case duit.Button5:
		return ui.scroll(m.Y / 4)
	}

	if scrollOnly {
		return false
	}
	switch m.Buttons {
	case duit.Button1:
		return ui.scroll(-m.Y)
	case duit.Button2:
		offset := m.Y * ui.Kid.R.Dy() / ui.barR.Dy()
		offsetMax := ui.Kid.R.Dy() - ui.childR.Dy()
		offset = maximum(0, minimum(offset, offsetMax))
		o := ui.Offset
		ui.Offset = offset
		return o != ui.Offset
	case duit.Button3:
		return ui.scroll(m.Y)
	}
	return false
}

func (ui *Scroll) result(dui *duit.DUI, self *duit.Kid, r *duit.Result, scrolled bool) {
	if ui.Kid.Layout != duit.Clean {
		ui.Kid.UI.Layout(dui, &ui.Kid, ui.childR.Size(), false)
		ui.Kid.Layout = duit.Clean
		ui.Kid.Draw = duit.Dirty
		self.Draw = duit.Dirty
		if r.Consumed && !scrolled {
			ui.tilesChanged = true
		}
	} else if ui.Kid.Draw != duit.Clean || scrolled {
		self.Draw = duit.Dirty
		if r.Consumed && !scrolled {
			ui.tilesChanged = true
		}
	}
}

func (ui *Scroll) Mouse(dui *duit.DUI, self *duit.Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r duit.Result) {
	nOrigM := origM
	nOrigM.Point = nOrigM.Point.Add(image.Pt(-ui.childR.Min.X, ui.Offset))
	nm := m
	nm.Point = nm.Point.Add(image.Pt(-ui.childR.Min.X, ui.Offset))

	if m.Buttons == 0 {
		ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, image.ZP)
		return
	}
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollMouse(m, false)
		self.Draw = duit.Dirty
		return
	} else if m.Point.In(ui.childR) {
		r.Consumed = ui.scrollMouse(m, true)
		if r.Consumed {
			self.Draw = duit.Dirty
			return
		}
		r = ui.Kid.UI.Mouse(dui, &ui.Kid, nm, nOrigM, image.ZP)
		if r.Consumed {
			self.Draw = duit.Dirty
			ui.tilesChanged = true
			log.Printf("Mouse: set ui.tilesChanged = true")
		}
	}
	return
}

func (ui *Scroll) Key(dui *duit.DUI, self *duit.Kid, k rune, m draw.Mouse, orig image.Point) (r duit.Result) {
	if m.Point.In(ui.barR) {
		r.Hit = ui
		r.Consumed = ui.scrollKey(k)
		if r.Consumed {
			self.Draw = duit.Dirty
		}
	}
	if m.Point.In(ui.childR) {
		log.Printf("Key: in ui.childR (self.Draw=%v)", self.Draw)
		m.Point = m.Point.Add(image.Pt(-ui.childR.Min.X, ui.Offset))
		scrolled := ui.scrollKey(k)
		if scrolled {
			self.Draw = duit.Dirty
			r.Consumed = scrolled
			return
		}
		r = ui.Kid.UI.Key(dui, &ui.Kid, k, m, image.ZP)
		ui.warpScroll(dui, self, r.Warp, orig)
		ui.result(dui, self, &r, scrolled)
		log.Printf("Key: in ui.childR (self.Draw'=%v)", self.Draw)
	}
	return
}

func (ui *Scroll) warpScroll(dui *duit.DUI, self *duit.Kid, warp *image.Point, orig image.Point) {
	if warp == nil {
		return
	}

	offset := ui.Offset
	if warp.Y < ui.Offset {
		ui.Offset = maximum(0, warp.Y-dui.Scale(40))
	} else if warp.Y > ui.Offset+ui.r.Dy() {
		ui.Offset = minimum(ui.Kid.R.Dy()-ui.r.Dy(), warp.Y+dui.Scale(40)-ui.r.Dy())
	}
	if offset != ui.Offset {
		if self != nil {
			self.Draw = duit.Dirty
		} else {
			dui.MarkDraw(ui)
		}
	}
	warp.Y -= ui.Offset
	warp.X += orig.X + ui.childR.Min.X
	warp.Y += orig.Y
}

func (ui *Scroll) focus(dui *duit.DUI, p *image.Point) *image.Point {
	if p == nil {
		return nil
	}
	pp := p.Add(ui.childR.Min)
	p = &pp
	ui.warpScroll(dui, nil, p, image.ZP)
	return p
}

func (ui *Scroll) FirstFocus(dui *duit.DUI, self *duit.Kid) *image.Point {
	p := ui.Kid.UI.FirstFocus(dui, &ui.Kid)
	return ui.focus(dui, p)
}

func (ui *Scroll) Focus(dui *duit.DUI, self *duit.Kid, o duit.UI) *image.Point {
	if o == ui {
		p := image.Pt(minimum(maximum(1, ui.childR.Min.X)/2, ui.r.Dx()), minimum(maximum(1, ui.scrollbarSize)/2, ui.r.Dy()))
		return &p
	}
	p := ui.Kid.UI.Focus(dui, &ui.Kid, o)
	return ui.focus(dui, p)
}

func (ui *Scroll) Mark(self *duit.Kid, o duit.UI, forLayout bool) (marked bool) {
	if self.Mark(o, forLayout) {
		return true
	}
	marked = ui.Kid.UI.Mark(&ui.Kid, o, forLayout)
	if marked {
		if forLayout {
			if self.Layout == duit.Clean {
				self.Layout = duit.DirtyKid
			}
		} else {
			if self.Layout == duit.Clean {
				self.Draw = duit.DirtyKid
			}
		}
	}
	return
}

func (ui *Scroll) Print(self *duit.Kid, indent int) {
	what := fmt.Sprintf("Scroll Offset=%d childR=%v", ui.Offset, ui.childR)
	duit.PrintUI(what, self, indent)
	ui.Kid.UI.Print(&ui.Kid, indent+1)
}

func pt(v int) image.Point {
	return image.Point{v, v}
}

func rect(p image.Point) image.Rectangle {
	return image.Rectangle{image.ZP, p}
}

func extendY(r image.Rectangle, dy int) image.Rectangle {
	r.Max.Y += dy
	return r
}

func insetPt(r image.Rectangle, pad image.Point) image.Rectangle {
	r.Min = r.Min.Add(pad)
	r.Max = r.Max.Sub(pad)
	return r
}

func outsetPt(r image.Rectangle, pad image.Point) image.Rectangle {
	r.Min = r.Min.Sub(pad)
	r.Max = r.Max.Add(pad)
	return r
}

func minimum64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maximum64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minimum(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maximum(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func debugLayout(d *duit.DUI, self *duit.Kid) {
	if d.DebugLayout > 0 {
		log.Printf("duit: Layout %T %s layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func debugDraw(d *duit.DUI, self *duit.Kid) {
	if d.DebugDraw > 0 {
		log.Printf("duit: Draw %T %s layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func duitError(d *duit.DUI, err error, msg string) bool {
	if err == nil {
		return false
	}
	go func() {
		d.Error <- fmt.Errorf("%s: %s", msg, err)
	}()
	return true
}
