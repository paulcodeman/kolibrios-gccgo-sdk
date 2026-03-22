package duit

import (
	"fmt"
	"image"

	"9fans.net/go/draw"
)

const (
	BorderSize    = 1
	ScrollbarSize = 10
)

const (
	Button1 = 1 << iota
	Button2
	Button3
	Button4
	Button5
)

type Halign byte

const (
	HalignLeft Halign = iota
	HalignMiddle
	HalignRight
)

type Valign byte

const (
	ValignMiddle Valign = iota
	ValignTop
	ValignBottom
)

type Event struct {
	Consumed   bool
	NeedLayout bool
	NeedDraw   bool
}

type Result struct {
	Hit      UI
	Consumed bool
	Warp     *image.Point
}

type Colors struct {
	Text       *draw.Image `json:"-"`
	Background *draw.Image `json:"-"`
	Border     *draw.Image `json:"-"`
}

type Colorset struct {
	Normal Colors
	Hover  Colors
}

type InputType byte

const (
	InputMouse InputType = iota
	InputKey
	InputFunc
	InputResize
	InputError
)

type Input struct {
	Type  InputType
	Mouse draw.Mouse
	Key   rune
	Func  func()
	Error error
}

type State byte

const (
	Dirty State = iota
	DirtyKid
	Clean
)

type DUI struct {
	Inputs  chan Input
	Top     Kid
	Call    chan func()
	Error   chan error
	Display *draw.Display

	Disabled,
	Inverse,
	Selection,
	SelectionHover,
	Placeholder,
	Striped Colors

	Regular,
	Primary,
	Secondary,
	Success,
	Danger Colorset

	BackgroundColor draw.Color
	Background      *draw.Image

	ScrollBGNormal,
	ScrollBGHover,
	ScrollVisibleNormal,
	ScrollVisibleHover *draw.Image

	Gutter *draw.Image

	Debug       bool
	DebugDraw   int
	DebugLayout int
	DebugKids   bool

	stop        chan struct{}
	mousectl    *draw.Mousectl
	keyctl      *draw.Keyboardctl
	mouse       draw.Mouse
	origMouse   draw.Mouse
	lastMouseUI UI
	focusUI     UI
	closed      bool
}

type DUIOpts struct {
	FontName   string
	Dimensions string
}

func NewDUI(name string, opts *DUIOpts) (dui *DUI, err error) {
	if opts == nil {
		opts = &DUIOpts{}
	}
	if opts.Dimensions == "" {
		opts.Dimensions = "960x700"
	}
	display, err := draw.Init(make(chan error, 1), opts.FontName, name, opts.Dimensions)
	if err != nil {
		return nil, err
	}
	makeColor := func(v draw.Color) *draw.Image {
		img, _ := display.AllocImage(image.Rect(0, 0, 1, 1), draw.ARGB32, true, v)
		return img
	}
	dui = &DUI{
		Inputs:  make(chan Input, 32),
		Call:    make(chan func(), 8),
		Error:   make(chan error, 8),
		Display: display,
		stop:    make(chan struct{}),

		Disabled: Colors{
			Text:       makeColor(0x888888ff),
			Background: makeColor(0xf0f0f0ff),
			Border:     makeColor(0xe0e0e0ff),
		},
		Selection: Colors{
			Text:       makeColor(0xeeeeeeff),
			Background: makeColor(0xbbbbbbff),
			Border:     makeColor(0x666666ff),
		},
		SelectionHover: Colors{
			Text:       makeColor(0xeeeeeeff),
			Background: makeColor(0x3272dcff),
			Border:     makeColor(0x666666ff),
		},
		Placeholder: Colors{
			Text:       makeColor(0xaaaaaaff),
			Background: makeColor(0xf8f8f8ff),
			Border:     makeColor(0xbbbbbbff),
		},
		Regular: Colorset{
			Normal: Colors{
				Text:       makeColor(0x333333ff),
				Background: makeColor(0xf8f8f8ff),
				Border:     makeColor(0xbbbbbbff),
			},
			Hover: Colors{
				Text:       makeColor(0x222222ff),
				Background: makeColor(0xfafafaff),
				Border:     makeColor(0x3272dcff),
			},
		},
		Primary: Colorset{
			Normal: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x007bffff),
				Border:     makeColor(0x007bffff),
			},
			Hover: Colors{
				Text:       makeColor(0xffffffff),
				Background: makeColor(0x0062ccff),
				Border:     makeColor(0x0062ccff),
			},
		},
		BackgroundColor:     draw.Color(0xfcfcfcff),
		Background:          makeColor(0xfcfcfcff),
		ScrollBGNormal:      makeColor(0xf4f4f4ff),
		ScrollBGHover:       makeColor(0xf0f0f0ff),
		ScrollVisibleNormal: makeColor(0xbbbbbbff),
		ScrollVisibleHover:  makeColor(0x999999ff),
		Gutter:              makeColor(0xbbbbbbff),
		Debug:               true,
	}
	dui.mouse = display.CurrentMouse()
	return dui, nil
}

func (d *DUI) Render() {
	d.Layout()
	d.Draw()
}

func (d *DUI) Layout() {
	if d.Top.Layout == Clean || d.Top.UI == nil {
		return
	}
	d.Top.UI.Layout(d, &d.Top, d.Display.ScreenImage.R.Size(), d.Top.Layout == Dirty)
	d.Top.Layout = Clean
}

func (d *DUI) Draw() {
	if d.Top.Draw == Clean || d.Top.UI == nil {
		return
	}
	fullRedraw := d.Top.Draw == Dirty
	dirtyRect, partialPresent := image.ZR, false
	if !fullRedraw {
		dirtyRect, partialPresent = dirtyBounds(&d.Top, image.ZP)
	}
	if fullRedraw && d.Background != nil {
		d.Display.ScreenImage.Draw(d.Display.ScreenImage.R, d.Background, nil, image.ZP)
	}
	d.Top.UI.Draw(d, &d.Top, d.Display.ScreenImage, image.ZP, d.mouse, fullRedraw)
	d.Top.Draw = Clean
	if partialPresent && !dirtyRect.Empty() {
		d.Display.FlushRect(dirtyRect)
		return
	}
	d.Display.Flush()
}

func (d *DUI) MarkLayout(ui UI) {
	if ui == nil {
		d.Top.Layout = Dirty
		return
	}
	d.Top.UI.Mark(&d.Top, ui, true)
}

func (d *DUI) MarkDraw(ui UI) {
	if ui == nil {
		d.Top.Draw = Dirty
		return
	}
	d.Top.UI.Mark(&d.Top, ui, false)
}

func (d *DUI) apply(r Result) {
	warped := false
	if r.Warp != nil {
		warped = true
		_ = d.Display.MoveTo(*r.Warp)
		d.mouse.Point = *r.Warp
		d.mouse.Buttons = 0
		d.origMouse = d.mouse
		r = d.Top.UI.Mouse(d, &d.Top, d.mouse, d.origMouse, image.ZP)
		if r.Hit != nil {
			d.setFocus(r.Hit)
		}
	}
	hitChanged := r.Hit != d.lastMouseUI
	if r.Hit != d.lastMouseUI {
		if r.Hit != nil {
			d.MarkDraw(r.Hit)
		}
		if d.lastMouseUI != nil {
			d.MarkDraw(d.lastMouseUI)
		}
	}
	d.lastMouseUI = r.Hit
	if warped || hitChanged || d.Top.Layout != Clean || d.Top.Draw != Clean {
		d.Render()
	}
}

func (d *DUI) setFocus(ui UI) bool {
	if d == nil || d.focusUI == ui {
		return false
	}
	prev := d.focusUI
	d.focusUI = ui
	if prev != nil {
		d.MarkDraw(prev)
	}
	if ui != nil {
		d.MarkDraw(ui)
	}
	return true
}

func (d *DUI) focused(ui UI) bool {
	return d != nil && ui != nil && d.focusUI == ui
}

func (d *DUI) Mouse(m draw.Mouse) {
	origMouse := d.origMouse
	if origMouse.Buttons == 0 {
		d.origMouse = m
		origMouse = m
	}
	d.mouse = m
	r := d.Top.UI.Mouse(d, &d.Top, m, origMouse, image.ZP)
	if m.Buttons != 0 || origMouse.Buttons != 0 {
		_ = d.setFocus(r.Hit)
	}
	d.apply(r)
	if m.Buttons == 0 {
		d.origMouse = m
	}
}

func (d *DUI) Resize() {
	_ = d.Display.Attach(draw.Refmesg)
	d.Top.Layout = Dirty
	d.Top.Draw = Dirty
	d.Render()
}

func (d *DUI) Key(k rune) {
	if d.focusUI == nil {
		if k == '\t' {
			var r Result
			if first := d.Top.UI.FirstFocus(d, &d.Top); first != nil {
				r.Warp = first
				r.Consumed = true
			}
			d.apply(r)
		}
		return
	}
	keyMouse := d.mouse
	if p := d.Top.UI.Focus(d, &d.Top, d.focusUI); p != nil {
		keyMouse.Point = *p
	}
	r := d.Top.UI.Key(d, &d.Top, k, keyMouse, image.ZP)
	if !r.Consumed && k == '\t' {
		if first := d.Top.UI.FirstFocus(d, &d.Top); first != nil {
			r.Warp = first
			r.Consumed = true
		}
	}
	if r.Hit != nil {
		_ = d.setFocus(r.Hit)
	}
	d.apply(r)
}

func (d *DUI) Scale(n int) int {
	if d == nil {
		return n
	}
	return d.Display.Scale(n)
}

func (d *DUI) ScaleSpace(s Space) Space {
	return Space{Top: d.Scale(s.Top), Right: d.Scale(s.Right), Bottom: d.Scale(s.Bottom), Left: d.Scale(s.Left)}
}

func (d *DUI) Font(font *draw.Font) *draw.Font {
	if font != nil {
		return font
	}
	return d.Display.DefaultFont
}

func (d *DUI) Input(e Input) {
	switch e.Type {
	case InputMouse:
		d.Mouse(e.Mouse)
	case InputKey:
		d.Key(e.Key)
	case InputResize:
		d.Resize()
	case InputFunc:
		if e.Func != nil {
			e.Func()
		}
		d.Render()
	case InputError:
		if e.Error != nil {
			d.Error <- e.Error
		}
	}
}

func (d *DUI) Close() {
	if d == nil || d.closed {
		return
	}
	d.closed = true
	close(d.stop)
	d.Display.Close()
}

func (d *DUI) Step() bool {
	if d == nil || d.Display == nil {
		return false
	}
	return d.stepEvent(d.Display.WaitInput())
}

func (d *DUI) StepFor(timeoutCentiseconds int) bool {
	if d == nil || d.Display == nil {
		return false
	}
	return d.stepEvent(d.Display.WaitInputFor(timeoutCentiseconds))
}

func (d *DUI) StepPoll() bool {
	if d == nil || d.Display == nil {
		return false
	}
	return d.stepEvent(d.Display.PollInput())
}

func (d *DUI) stepEvent(e draw.Event) bool {
	switch e.Type {
	case draw.EventMouseInput:
		d.Mouse(e.Mouse)
	case draw.EventKeyInput:
		d.Key(e.Key)
	case draw.EventResize:
		d.Resize()
	case draw.EventClose:
		d.Close()
		return false
	}
	return !d.closed
}

func (d *DUI) Run() {
	for d.Step() {
	}
}

func (d *DUI) WriteSnarf(buf []byte) {
	if err := d.Display.WriteSnarf(buf); err != nil {
		fmt.Printf("duit: writesnarf: %v\n", err)
	}
}

func (d *DUI) ReadSnarf() ([]byte, bool) {
	return nil, false
}

func (d *DUI) debugLayout(self *Kid) {
	if d.DebugLayout > 0 {
		fmt.Printf("duit: Layout %T %v layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func (d *DUI) debugDraw(self *Kid) {
	if d.DebugDraw > 0 {
		fmt.Printf("duit: Draw %T %v layout=%d draw=%d\n", self.UI, self.R, self.Layout, self.Draw)
	}
}

func (d *DUI) error(err error, msg string) bool {
	if err == nil {
		return false
	}
	select {
	case d.Error <- fmt.Errorf("%s: %v", msg, err):
	default:
	}
	return true
}
