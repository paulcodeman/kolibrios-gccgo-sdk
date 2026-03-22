package duit

import (
	"image"
	"strings"
	"unicode/utf8"

	"9fans.net/go/draw"
)

type Field struct {
	Text            string
	Placeholder     string
	Disabled        bool
	Cursor1         int
	SelectionStart1 int
	Password        bool
	Font            *draw.Font                       `json:"-"`
	Changed         func(text string) Event          `json:"-"`
	Keys            func(k rune, m draw.Mouse) Event `json:"-"`

	size image.Point
	m    draw.Mouse
}

var _ UI = &Field{}

func (ui *Field) font(dui *DUI) *draw.Font {
	return dui.Font(ui.Font)
}

func (ui *Field) padding(dui *DUI) image.Point {
	fontHeight := ui.font(dui).Height
	return image.Pt(fontHeight/4, fontHeight/4)
}

func (ui *Field) space(dui *DUI) image.Point {
	return ui.padding(dui).Add(pt(1))
}

func (ui *Field) cursor0() int {
	if ui.Cursor1 <= 0 || ui.Cursor1-1 > len(ui.Text) {
		return len(ui.Text)
	}
	return ui.Cursor1 - 1
}

func (ui *Field) setCursor0(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(ui.Text) {
		pos = len(ui.Text)
	}
	ui.Cursor1 = pos + 1
	ui.SelectionStart1 = 0
}

func (ui *Field) setSelectionStart0(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos > len(ui.Text) {
		pos = len(ui.Text)
	}
	ui.SelectionStart1 = pos + 1
}

func (ui *Field) selectionStart0() int {
	if ui.SelectionStart1 <= 0 || ui.SelectionStart1-1 > len(ui.Text) {
		return ui.cursor0()
	}
	return ui.SelectionStart1 - 1
}

func (ui *Field) selectionRange0() (start int, end int, ok bool) {
	if ui.SelectionStart1 == 0 {
		return 0, 0, false
	}
	start = ui.selectionStart0()
	end = ui.cursor0()
	if start == end {
		return 0, 0, false
	}
	if start > end {
		start, end = end, start
	}
	return start, end, true
}

func (ui *Field) clearSelection() {
	ui.SelectionStart1 = 0
}

func (ui *Field) Layout(dui *DUI, self *Kid, sizeAvail image.Point, force bool) {
	ui.size = image.Point{X: sizeAvail.X, Y: ui.font(dui).Height + 2*ui.space(dui).Y}
	self.R = rect(ui.size)
}

func (ui *Field) displayText() string {
	if ui.Password && ui.Text != "" {
		return strings.Repeat("•", utf8.RuneCountInString(ui.Text))
	}
	if ui.Text == "" {
		return ui.Placeholder
	}
	return ui.Text
}

func (ui *Field) contentText() string {
	if ui.Password && ui.Text != "" {
		return strings.Repeat("•", utf8.RuneCountInString(ui.Text))
	}
	return ui.Text
}

func (ui *Field) displayPrefix(pos int) string {
	if pos <= 0 {
		return ""
	}
	if pos > len(ui.Text) {
		pos = len(ui.Text)
	}
	if ui.Password {
		return strings.Repeat("•", utf8.RuneCountInString(ui.Text[:pos]))
	}
	return ui.Text[:pos]
}

func (ui *Field) displaySegment(start int, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(ui.Text) {
		end = len(ui.Text)
	}
	if start >= end {
		return ""
	}
	if ui.Password {
		return strings.Repeat("•", utf8.RuneCountInString(ui.Text[start:end]))
	}
	return ui.Text[start:end]
}

func (ui *Field) prevCursor0(pos int) int {
	if pos <= 0 {
		return 0
	}
	_, size := utf8.DecodeLastRuneInString(ui.Text[:pos])
	if size <= 0 {
		size = 1
	}
	return maximum(0, pos-size)
}

func (ui *Field) nextCursor0(pos int) int {
	if pos >= len(ui.Text) {
		return len(ui.Text)
	}
	_, size := utf8.DecodeRuneInString(ui.Text[pos:])
	if size <= 0 {
		size = 1
	}
	return minimum(len(ui.Text), pos+size)
}

func (ui *Field) cursorFromPoint(dui *DUI, p image.Point) int {
	textX := p.X - ui.space(dui).X
	if textX <= 0 || ui.Text == "" {
		return 0
	}
	font := ui.font(dui)
	prevWidth := 0
	for pos := 0; pos < len(ui.Text); {
		next := ui.nextCursor0(pos)
		nextWidth := font.StringWidth(ui.displayPrefix(next))
		if textX < (prevWidth+nextWidth)/2 {
			return pos
		}
		prevWidth = nextWidth
		pos = next
	}
	return len(ui.Text)
}

func (ui *Field) deleteSelection() bool {
	start, end, ok := ui.selectionRange0()
	if !ok {
		return false
	}
	ui.Text = ui.Text[:start] + ui.Text[end:]
	ui.setCursor0(start)
	return true
}

func (ui *Field) selectedText() string {
	start, end, ok := ui.selectionRange0()
	if !ok {
		return ""
	}
	return ui.Text[start:end]
}

func (ui *Field) Draw(dui *DUI, self *Kid, img *draw.Image, orig image.Point, m draw.Mouse, force bool) {
	r := rect(ui.size).Add(orig)
	hover := m.In(rect(ui.size))
	focused := dui.focused(ui)
	colors := dui.Regular.Normal
	if ui.Disabled {
		colors = dui.Disabled
	} else if hover || focused {
		colors = dui.Regular.Hover
	}
	if ui.Text == "" && !ui.Disabled {
		colors = dui.Placeholder
		if hover || focused {
			colors.Border = dui.Regular.Hover.Border
		}
	}
	img.Draw(r, colors.Background, nil, image.ZP)
	drawRoundedBorder(img, r, colors.Border)
	p := r.Min.Add(ui.space(dui))
	font := ui.font(dui)
	if ui.Text == "" {
		img.String(p, colors.Text, image.ZP, font, ui.displayText())
	} else if start, end, ok := ui.selectionRange0(); ok && focused {
		before := ui.displayPrefix(start)
		selected := ui.displaySegment(start, end)
		after := ui.displaySegment(end, len(ui.Text))
		beforeWidth := font.StringWidth(before)
		selectedWidth := font.StringWidth(selected)
		if selectedWidth > 0 {
			selMin := p.Add(image.Pt(beforeWidth, 0))
			selMax := selMin.Add(image.Pt(selectedWidth, font.Height))
			img.Draw(image.Rectangle{Min: selMin, Max: selMax}, dui.Selection.Background, nil, image.ZP)
			img.String(selMin, dui.Selection.Text, image.ZP, font, selected)
		}
		img.String(p, colors.Text, image.ZP, font, before)
		img.String(p.Add(image.Pt(beforeWidth+selectedWidth, 0)), colors.Text, image.ZP, font, after)
	} else {
		img.String(p, colors.Text, image.ZP, font, ui.contentText())
	}
	if focused && !ui.Disabled {
		cursorText := ui.displayPrefix(ui.cursor0())
		cp := p.Add(image.Pt(font.StringWidth(cursorText), 0))
		cp1 := cp
		cp1.Y += font.Height
		img.Line(cp, cp1, 1, 1, 0, dui.Regular.Hover.Border, image.ZP)
	}
}

func (ui *Field) insertText(s string) {
	if ui.deleteSelection() {
	}
	cursor := ui.cursor0()
	ui.Text = ui.Text[:cursor] + s + ui.Text[cursor:]
	ui.setCursor0(cursor + len(s))
}

func (ui *Field) deleteBackward() {
	if ui.deleteSelection() {
		return
	}
	cursor := ui.cursor0()
	if cursor <= 0 {
		return
	}
	prev := ui.prevCursor0(cursor)
	ui.Text = ui.Text[:prev] + ui.Text[cursor:]
	ui.setCursor0(prev)
}

func (ui *Field) deleteForward() {
	if ui.deleteSelection() {
		return
	}
	cursor := ui.cursor0()
	if cursor >= len(ui.Text) {
		return
	}
	next := ui.nextCursor0(cursor)
	ui.Text = ui.Text[:cursor] + ui.Text[next:]
	ui.setCursor0(cursor)
}

func (ui *Field) notifyChanged(self *Kid, r *Result) {
	if ui.Changed != nil {
		e := ui.Changed(ui.Text)
		propagateEvent(self, r, e)
	} else {
		self.Draw = Dirty
		r.Consumed = true
	}
}

func (ui *Field) Mouse(dui *DUI, self *Kid, m draw.Mouse, origM draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	if m.In(rect(ui.size)) && ui.m.Buttons&Button1 == 0 && m.Buttons&Button1 == Button1 {
		cursor := ui.cursorFromPoint(dui, m.Point)
		ui.setCursor0(cursor)
		ui.setSelectionStart0(cursor)
		self.Draw = Dirty
		r.Hit = ui
		r.Consumed = true
	} else if ui.m.Buttons&Button1 == Button1 && m.Buttons&Button1 == Button1 {
		cursor := ui.cursorFromPoint(dui, m.Point)
		if cursor != ui.cursor0() {
			ui.Cursor1 = cursor + 1
			self.Draw = Dirty
		}
		r.Hit = ui
		r.Consumed = true
	} else if ui.m.Buttons&Button1 == Button1 && m.Buttons&Button1 == 0 {
		cursor := ui.cursorFromPoint(dui, m.Point)
		if cursor != ui.cursor0() {
			ui.Cursor1 = cursor + 1
			self.Draw = Dirty
		}
		if ui.SelectionStart1 != 0 && ui.selectionStart0() == ui.cursor0() {
			ui.clearSelection()
		}
		self.Draw = Dirty
		r.Hit = ui
		r.Consumed = true
	}
	ui.m = m
	return
}

func (ui *Field) Key(dui *DUI, self *Kid, k rune, m draw.Mouse, orig image.Point) (r Result) {
	if ui.Disabled {
		return
	}
	r.Hit = ui
	if ui.Keys != nil {
		e := ui.Keys(k, m)
		propagateEvent(self, &r, e)
		if r.Consumed {
			return
		}
	}
	switch k {
	case draw.KeyLeft:
		if start, _, ok := ui.selectionRange0(); ok {
			ui.setCursor0(start)
		} else {
			ui.setCursor0(ui.prevCursor0(ui.cursor0()))
		}
	case draw.KeyRight:
		if _, end, ok := ui.selectionRange0(); ok {
			ui.setCursor0(end)
		} else {
			ui.setCursor0(ui.nextCursor0(ui.cursor0()))
		}
	case draw.KeyHome:
		ui.setCursor0(0)
	case draw.KeyEnd:
		ui.setCursor0(len(ui.Text))
	case draw.KeyBackspace:
		ui.deleteBackward()
		ui.notifyChanged(self, &r)
	case draw.KeyDelete:
		ui.deleteForward()
		ui.notifyChanged(self, &r)
	case draw.KeyCmd + 'a':
		ui.setCursor0(len(ui.Text))
		ui.setSelectionStart0(0)
		r.Consumed = true
	case draw.KeyCmd + 'c':
		text := ui.selectedText()
		if text == "" {
			text = ui.Text
		}
		if text != "" {
			dui.WriteSnarf([]byte(text))
			r.Consumed = true
		}
	case draw.KeyCmd + 'x':
		text := ui.selectedText()
		if text == "" {
			text = ui.Text
		}
		if text != "" {
			dui.WriteSnarf([]byte(text))
			if !ui.deleteSelection() {
				ui.Text = ""
				ui.setCursor0(0)
			}
			ui.notifyChanged(self, &r)
		}
	case draw.KeyCmd + 'v':
		if buf, ok := dui.ReadSnarf(); ok && len(buf) > 0 {
			ui.insertText(string(buf))
			ui.notifyChanged(self, &r)
		}
	default:
		if k >= 32 && k != 127 {
			ui.insertText(string(k))
			ui.notifyChanged(self, &r)
		}
	}
	self.Draw = Dirty
	return
}

func (ui *Field) FirstFocus(dui *DUI, self *Kid) *image.Point {
	p := ui.space(dui)
	return &p
}

func (ui *Field) Focus(dui *DUI, self *Kid, o UI) *image.Point {
	if o != ui {
		return nil
	}
	return ui.FirstFocus(dui, self)
}

func (ui *Field) Mark(self *Kid, o UI, forLayout bool) bool { return self.Mark(o, forLayout) }
func (ui *Field) Print(self *Kid, indent int)               { PrintUI("Field", self, indent) }
