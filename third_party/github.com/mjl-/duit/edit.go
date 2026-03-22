package duit

import (
	"image"
	"strings"
	"unicode/utf8"

	"9fans.net/go/draw"
)

type EditColors struct {
	Fg, Bg,
	SelFg, SelBg,
	ScrollVis, ScrollBg,
	HoverScrollVis, HoverScrollBg,
	CommandBorder, VisualBorder *draw.Image
}

type Cursor struct {
	Cur   int64
	Start int64
}

// Edit is a minimal multiline text editor compatible with the upstream API
// surface used by mycel. It intentionally implements only the subset needed by
// browser bring-up on KolibriOS.
type Edit struct {
	NoScrollbar  bool
	LastSearch   string
	Error        chan error
	Colors       *EditColors                                `json:"-"`
	Font         *draw.Font                                 `json:"-"`
	Keys         func(k rune, m draw.Mouse) (e Event)       `json:"-"`
	Click        func(m draw.Mouse, offset int64) (e Event) `json:"-"`
	DirtyChanged func(dirty bool)                           `json:"-"`

	text   string
	cursor int
	size   image.Point
	m      draw.Mouse
}

var _ UI = &Edit{}

func (ui *Edit) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Edit) padding(dui *DUI) image.Point {
	h := ui.font(dui).Height
	return image.Pt(h/4, h/4)
}

func (ui *Edit) lineHeight(dui *DUI) int {
	return ui.font(dui).Height + ui.padding(dui).Y/2
}

func (ui *Edit) lineStarts() []int {
	starts := []int{0}
	for i := 0; i < len(ui.text); i++ {
		if ui.text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func (ui *Edit) lineInfo(pos int) (line int, col int, starts []int) {
	starts = ui.lineStarts()
	if pos < 0 {
		pos = 0
	}
	if pos > len(ui.text) {
		pos = len(ui.text)
	}
	for i := 0; i < len(starts); i++ {
		start := starts[i]
		end := len(ui.text)
		if i+1 < len(starts) {
			end = starts[i+1] - 1
		}
		if start <= pos && pos <= end {
			return i, pos - start, starts
		}
	}
	last := len(starts) - 1
	return last, pos - starts[last], starts
}

func (ui *Edit) lineBounds(starts []int, line int) (start int, end int) {
	if len(starts) == 0 {
		return 0, 0
	}
	if line < 0 {
		line = 0
	}
	if line >= len(starts) {
		line = len(starts) - 1
	}
	start = starts[line]
	end = len(ui.text)
	if line+1 < len(starts) {
		end = starts[line+1] - 1
	}
	if end < start {
		end = start
	}
	return
}

func (ui *Edit) lineText(starts []int, line int) string {
	start, end := ui.lineBounds(starts, line)
	return ui.text[start:end]
}

func (ui *Edit) moveVertical(delta int) {
	line, col, starts := ui.lineInfo(ui.cursor)
	line += delta
	if line < 0 {
		line = 0
	}
	if line >= len(starts) {
		line = len(starts) - 1
	}
	start, end := ui.lineBounds(starts, line)
	ui.cursor = start + minimum(col, end-start)
}

func (ui *Edit) prevIndex(pos int) int {
	if pos <= 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(ui.text[:pos])
	if size <= 0 {
		size = 1
	}
	return maximum(0, pos-size)
}

func (ui *Edit) nextIndex(pos int) int {
	if pos >= len(ui.text) {
		return len(ui.text)
	}
	_, size := utf8.DecodeRuneInString(ui.text[pos:])
	if size <= 0 {
		size = 1
	}
	return minimum(len(ui.text), pos+size)
}

func (ui *Edit) insertText(s string) {
	if s == "" {
		return
	}
	ui.text = ui.text[:ui.cursor] + s + ui.text[ui.cursor:]
	ui.cursor += len(s)
}

func (ui *Edit) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	dui.debugLayout(self)
	pad := ui.padding(dui)
	lines := len(ui.lineStarts())
	if lines == 0 {
		lines = 1
	}
	h := lines*ui.lineHeight(dui) + 2*pad.Y + 2
	w := sizeAvail.X
	if w <= 0 {
		w = ui.font(dui).StringWidth(ui.text) + 2*pad.X + 2
	}
	if w <= 0 {
		w = dui.Scale(100)
	}
	ui.size = image.Pt(w, h)
	self.R = rect(ui.size)
}

func (ui *Edit) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	dui.debugDraw(self)
	r := rect(ui.size).Add(orig)
	hover := m.In(rect(ui.size))
	focused := dui.focused(ui)
	colors := dui.Regular.Normal
	if hover || focused {
		colors = dui.Regular.Hover
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)

	pad := ui.padding(dui)
	font := ui.font(dui)
	p := r.Min.Add(image.Pt(pad.X+1, pad.Y+1))
	lineHeight := ui.lineHeight(dui)
	starts := ui.lineStarts()
	for i := 0; i < len(starts); i++ {
		img.String(p.Add(image.Pt(0, i*lineHeight)), colors.Text, image.ZP, font, ui.lineText(starts, i))
	}
	if focused {
		line, col, starts := ui.lineInfo(ui.cursor)
		start, _ := ui.lineBounds(starts, line)
		x := font.StringWidth(ui.text[start : start+col])
		cp := p.Add(image.Pt(x, line*lineHeight))
		cp1 := cp
		cp1.Y += font.Height
		img.Line(cp, cp1, 1, 1, 0, dui.Regular.Hover.Border, image.ZP)
	}
}

func (ui *Edit) cursorFromPoint(dui *DUI, p image.Point) int {
	pad := ui.padding(dui)
	font := ui.font(dui)
	lineHeight := ui.lineHeight(dui)
	x := p.X - pad.X - 1
	y := p.Y - pad.Y - 1
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	starts := ui.lineStarts()
	line := minimum(len(starts)-1, y/maximum(1, lineHeight))
	start, end := ui.lineBounds(starts, line)
	pos := start
	prevWidth := 0
	for pos < end {
		next := ui.nextIndex(pos)
		nextWidth := font.StringWidth(ui.text[start:next])
		if x < (prevWidth+nextWidth)/2 {
			return pos
		}
		prevWidth = nextWidth
		pos = next
	}
	return end
}

func (ui *Edit) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	ui.m = m
	if !m.In(rect(ui.size)) {
		return
	}
	r.Hit = ui
	if ui.Click != nil && ui.m.Buttons&Button1 == 0 && m.Buttons&Button1 == Button1 {
		e := ui.Click(m, int64(ui.cursor))
		propagateEvent(self, &r, e)
		if r.Consumed {
			return
		}
	}
	if origM.Buttons&Button1 == 0 && m.Buttons&Button1 == Button1 {
		ui.cursor = ui.cursorFromPoint(dui, m.Point)
		self.Draw = Dirty
		r.Consumed = true
	}
	return
}

func (ui *Edit) Text() ([]byte, error) {
	return []byte(ui.text), nil
}

func (ui *Edit) Append(buf []byte) {
	ui.text += string(buf)
	ui.cursor = len(ui.text)
}

func (ui *Edit) Cursor() Cursor {
	return Cursor{Cur: int64(ui.cursor), Start: int64(ui.cursor)}
}

func (ui *Edit) SetCursor(c Cursor) {
	pos := int(c.Cur)
	if pos < 0 {
		pos = 0
	}
	if pos > len(ui.text) {
		pos = len(ui.text)
	}
	ui.cursor = pos
}

func (ui *Edit) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	r.Hit = ui
	if ui.Keys != nil {
		e := ui.Keys(k, m)
		propagateEvent(self, &r, e)
		if r.Consumed {
			return
		}
	}
	beforeLines := strings.Count(ui.text, "\n")
	changed := false
	switch k {
	case draw.KeyLeft:
		ui.cursor = ui.prevIndex(ui.cursor)
		r.Consumed = true
	case draw.KeyRight:
		ui.cursor = ui.nextIndex(ui.cursor)
		r.Consumed = true
	case draw.KeyUp:
		ui.moveVertical(-1)
		r.Consumed = true
	case draw.KeyDown:
		ui.moveVertical(1)
		r.Consumed = true
	case draw.KeyHome:
		line, _, starts := ui.lineInfo(ui.cursor)
		start, _ := ui.lineBounds(starts, line)
		ui.cursor = start
		r.Consumed = true
	case draw.KeyEnd:
		line, _, starts := ui.lineInfo(ui.cursor)
		_, end := ui.lineBounds(starts, line)
		ui.cursor = end
		r.Consumed = true
	case draw.KeyBackspace:
		if ui.cursor > 0 {
			prev := ui.prevIndex(ui.cursor)
			ui.text = ui.text[:prev] + ui.text[ui.cursor:]
			ui.cursor = prev
			changed = true
			r.Consumed = true
		}
	case draw.KeyDelete:
		if ui.cursor < len(ui.text) {
			next := ui.nextIndex(ui.cursor)
			ui.text = ui.text[:ui.cursor] + ui.text[next:]
			changed = true
			r.Consumed = true
		}
	case '\n', '\r':
		ui.insertText("\n")
		changed = true
		r.Consumed = true
	case '\t':
		ui.insertText("\t")
		changed = true
		r.Consumed = true
	default:
		if k >= ' ' && k != 0x7f && k < draw.KeyCmd {
			ui.insertText(string(k))
			changed = true
			r.Consumed = true
		}
	}
	if changed || r.Consumed {
		self.Draw = Dirty
		if strings.Count(ui.text, "\n") != beforeLines {
			self.Layout = Dirty
		}
		if ui.DirtyChanged != nil && changed {
			ui.DirtyChanged(true)
		}
	}
	return
}

func (ui *Edit) FirstFocus(dui *DUI, self *Kid) (warp *image.Point) {
	pad := ui.padding(dui)
	p := image.Pt(pad.X+1, pad.Y+1)
	return &p
}

func (ui *Edit) Focus(dui *DUI, self *Kid, o UI) (warp *image.Point) {
	if o != ui {
		return nil
	}
	pad := ui.padding(dui)
	line, col, starts := ui.lineInfo(ui.cursor)
	start, _ := ui.lineBounds(starts, line)
	x := ui.font(dui).StringWidth(ui.text[start : start+col])
	p := image.Pt(pad.X+1+x, pad.Y+1+line*ui.lineHeight(dui))
	return &p
}

func (ui *Edit) Mark(self *Kid, o UI, forLayout bool) (marked bool) {
	return self.Mark(o, forLayout)
}

func (ui *Edit) Print(self *Kid, indent int) {
	PrintUI("Edit", self, indent)
}
