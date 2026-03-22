package draw

import (
	"errors"
	"image"
	"os"
	"strconv"
	"sync"

	"kos"
	core "surface/core"
)

type Display struct {
	ScreenImage *Image
	DPI         int
	DefaultFont *Font

	White       *Image
	Black       *Image
	Opaque      *Image
	Transparent *Image

	mu        sync.Mutex
	title     string
	presenter core.Presenter
	stop      chan struct{}

	mouseMu     sync.Mutex
	mousectl    *Mousectl
	mouseCh     chan Mouse
	resizeCh    chan bool
	keyboardctl *Keyboardctl
	keyboardCh  chan rune
	loopStarted bool
	closed      bool
	fullRedraw  bool
}

type EventType int

const (
	EventNone EventType = iota
	EventMouseInput
	EventKeyInput
	EventResize
	EventClose
)

type Event struct {
	Type  EventType
	Mouse Mouse
	Key   rune
}

func Init(errch chan<- error, fontName string, label string, size string) (*Display, error) {
	width, height, err := parseDimensions(size)
	if err != nil {
		return nil, err
	}
	client := core.WindowClientRect(width, height)
	screen := newImage(nil, image.Rect(0, 0, client.Width, client.Height), ARGB32, false)
	display := &Display{
		DPI:        DefaultDPI,
		title:      label,
		presenter:  core.NewPresenter(48, 48, width, height, label),
		stop:       make(chan struct{}),
		fullRedraw: true,
	}
	screen.Display = display
	display.ScreenImage = screen
	display.White, _ = display.AllocImage(image.Rect(0, 0, 1, 1), ARGB32, true, White)
	display.Black, _ = display.AllocImage(image.Rect(0, 0, 1, 1), ARGB32, true, Black)
	display.Opaque = display.White
	display.Transparent, _ = display.AllocImage(image.Rect(0, 0, 1, 1), ARGB32, true, Transparent)
	display.DefaultFont = newFallbackFont(display, "*default*")
	if fontName != "" {
		if font, err := display.OpenFont(fontName); err == nil && font != nil {
			display.DefaultFont = font
		}
	} else if path := defaultFontPath(); path != "" {
		if font, err := display.OpenFont(path); err == nil && font != nil {
			display.DefaultFont = font
		}
	}
	display.Flush()
	_ = display.Attach(0)
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse | kos.EventMaskMouseActiveWindowOnly)
	if errch != nil {
		_ = errch
	}
	return display, nil
}

func (d *Display) HiDPI() bool {
	return d != nil && d.DPI > DefaultDPI
}

func (d *Display) Scale(n int) int {
	if d == nil || d.DPI <= DefaultDPI {
		return n
	}
	return (d.DPI / DefaultDPI) * n
}

func (d *Display) SetDebug(bool) {}

func (d *Display) AllocImage(r Rectangle, pix Pix, repl bool, color Color) (*Image, error) {
	if d == nil {
		return nil, errors.New("nil display")
	}
	img := newImage(d, r, pix, repl)
	img.fill(color)
	return img, nil
}

func (d *Display) AllocImageMix(color1 Color, color3 Color) (*Image, error) {
	_ = color3
	return d.AllocImage(image.Rect(0, 0, 1, 1), ARGB32, true, color1)
}

func (d *Display) OpenFont(name string) (*Font, error) {
	if d == nil {
		return nil, errors.New("nil display")
	}
	if name == "" {
		if d.DefaultFont != nil {
			return d.DefaultFont, nil
		}
		return newFallbackFont(d, "*default*"), nil
	}
	path, size := parseFontRequest(name)
	if _, err := os.Stat(path); err == nil {
		if font := loadFontFile(d, name); font != nil {
			return font, nil
		}
	}
	if path := defaultFontPath(); path != "" {
		request := path
		if size > 0 {
			request = path + "@" + strconv.Itoa(size)
		}
		if font := loadFontFile(d, request); font != nil {
			font.Name = name
			return font, nil
		}
	}
	return newFallbackFont(d, name), nil
}

func (d *Display) InitMouse() *Mousectl {
	if d == nil {
		return nil
	}
	d.mouseMu.Lock()
	defer d.mouseMu.Unlock()
	if d.mousectl != nil {
		return d.mousectl
	}
	d.mouseCh = make(chan Mouse, 32)
	d.resizeCh = make(chan bool, 8)
	d.mousectl = &Mousectl{
		C:       d.mouseCh,
		Resize:  d.resizeCh,
		Display: d,
	}
	d.mouseCh <- d.currentMouse()
	d.startLoop()
	return d.mousectl
}

func (d *Display) InitKeyboard() *Keyboardctl {
	if d == nil {
		return nil
	}
	d.mouseMu.Lock()
	defer d.mouseMu.Unlock()
	if d.keyboardctl != nil {
		return d.keyboardctl
	}
	d.keyboardCh = make(chan rune, 32)
	d.keyboardctl = &Keyboardctl{C: d.keyboardCh}
	d.startLoop()
	return d.keyboardctl
}

func (d *Display) startLoop() {
	if d.loopStarted {
		return
	}
	d.loopStarted = true
	go d.eventLoop()
}

func (d *Display) eventLoop() {
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse | kos.EventMaskMouseActiveWindowOnly)
	for {
		select {
		case <-d.stop:
			return
		default:
		}
		switch kos.WaitEvent() {
		case kos.EventMouse:
			m := d.currentMouse()
			if d.mouseCh != nil {
				select {
				case d.mouseCh <- m:
				default:
				}
			}
		case kos.EventKey:
			if d.keyboardCh != nil {
				if key := mapKeyEvent(kos.ReadKey()); key != 0 {
					select {
					case d.keyboardCh <- key:
					default:
					}
				}
			} else {
				_ = kos.ReadKey()
			}
		case kos.EventRedraw:
			d.fullRedraw = true
			if d.resizeCh != nil {
				select {
				case d.resizeCh <- true:
				default:
				}
			}
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				d.Close()
				return
			}
		}
	}
}

func (d *Display) currentMouse() Mouse {
	pos := kos.MouseWindowPosition()
	pos.X -= d.presenter.Client.X
	pos.Y -= d.presenter.Client.Y
	buttons := mouseButtons(kos.MouseHeldButtons(), kos.MouseButtons())
	return Mouse{
		Point:   image.Pt(pos.X, pos.Y),
		Buttons: buttons,
		Msec:    kos.UptimeCentiseconds() * 10,
	}
}

func mouseButtons(held kos.MouseButtonInfo, changed kos.MouseButtonInfo) int {
	buttons := 0
	if held.LeftHeld {
		buttons |= 1
	}
	if held.MiddleHeld {
		buttons |= 2
	}
	if held.RightHeld {
		buttons |= 4
	}
	if changed.VerticalScroll {
		delta := kos.MouseScrollDelta()
		if delta.Y < 0 {
			buttons = 8
		} else if delta.Y > 0 {
			buttons = 16
		}
	}
	return buttons
}

func mapKeyEvent(event kos.KeyEvent) rune {
	if event.Empty {
		return 0
	}
	if event.Hotkey {
		switch event.ScanCode {
		case 59, 60, 61, 62, 63, 64, 65, 66, 67:
			return KeyFn + rune(event.ScanCode-58)
		case 68:
			return KeyFn + 10
		case 71:
			return KeyHome
		case 72:
			return KeyUp
		case 73:
			return KeyPageUp
		case 75:
			return KeyLeft
		case 77:
			return KeyRight
		case 79:
			return KeyEnd
		case 80:
			return KeyDown
		case 81:
			return KeyPageDown
		case 82:
			return KeyInsert
		case 83:
			return KeyDelete
		}
	}
	if event.Code == 0 {
		return 0
	}
	key := rune(event.Code)
	controls := kos.ControlKeysStatus()
	if controls.Ctrl() && key >= 32 && key < 127 {
		if key >= 'A' && key <= 'Z' {
			key += 'a' - 'A'
		}
		return KeyCmd + key
	}
	switch key {
	case '\r':
		return '\n'
	case '\b':
		return KeyBackspace
	default:
		return key
	}
}

func (d *Display) MoveTo(p Point) error {
	info, _, ok := kos.ReadCurrentThreadInfo()
	if !ok {
		return errors.New("thread info unavailable")
	}
	kos.SetMousePointerPosition(info.ClientPosition.X+p.X, info.ClientPosition.Y+p.Y)
	return nil
}

func (d *Display) SwitchCursor(*Cursor) error {
	return nil
}

func (d *Display) Attach(int) error {
	if d == nil {
		return errors.New("nil display")
	}
	info, _, ok := kos.ReadCurrentThreadInfo()
	if !ok {
		return nil
	}
	if info.WindowSize.X > 0 && info.WindowSize.Y > 0 {
		d.presenter.X = info.WindowPosition.X
		d.presenter.Y = info.WindowPosition.Y
		d.presenter.SetSize(info.WindowSize.X, info.WindowSize.Y)
	}
	if d.presenter.Client.Width > 0 && d.presenter.Client.Height > 0 {
		d.ScreenImage.resizeRect(image.Rect(0, 0, d.presenter.Client.Width, d.presenter.Client.Height))
	}
	d.fullRedraw = true
	return nil
}

func (d *Display) Flush() {
	if d == nil || d.ScreenImage == nil {
		return
	}
	if d.fullRedraw {
		d.presenter.PresentFull(d.ScreenImage.buffer)
		d.fullRedraw = false
		return
	}
	d.presenter.PresentClient(d.ScreenImage.buffer)
}

func (d *Display) FlushRect(r Rectangle) {
	if d == nil || d.ScreenImage == nil {
		return
	}
	if d.fullRedraw {
		d.Flush()
		return
	}
	r = r.Intersect(d.ScreenImage.R)
	if r.Empty() {
		return
	}
	d.presenter.PresentRect(d.ScreenImage.buffer, core.Rect{
		X:      r.Min.X - d.ScreenImage.R.Min.X,
		Y:      r.Min.Y - d.ScreenImage.R.Min.Y,
		Width:  r.Dx(),
		Height: r.Dy(),
	})
}

func (d *Display) CurrentMouse() Mouse {
	if d == nil {
		return Mouse{}
	}
	return d.currentMouse()
}

func (d *Display) PollInput() Event {
	if d == nil {
		return Event{}
	}
	switch kos.PollEvent() {
	case kos.EventNone:
		return Event{Type: EventNone}
	case kos.EventMouse:
		return Event{Type: EventMouseInput, Mouse: d.currentMouse()}
	case kos.EventKey:
		if key := mapKeyEvent(kos.ReadKey()); key != 0 {
			return Event{Type: EventKeyInput, Key: key}
		}
		return Event{Type: EventNone}
	case kos.EventRedraw:
		d.fullRedraw = true
		return Event{Type: EventResize}
	case kos.EventButton:
		if kos.CurrentButtonID() == 1 {
			return Event{Type: EventClose}
		}
	}
	return Event{Type: EventNone}
}

func (d *Display) WaitInput() Event {
	if d == nil {
		return Event{}
	}
	for {
		switch kos.WaitEvent() {
		case kos.EventMouse:
			return Event{Type: EventMouseInput, Mouse: d.currentMouse()}
		case kos.EventKey:
			if key := mapKeyEvent(kos.ReadKey()); key != 0 {
				return Event{Type: EventKeyInput, Key: key}
			}
		case kos.EventRedraw:
			d.fullRedraw = true
			return Event{Type: EventResize}
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 {
				return Event{Type: EventClose}
			}
		}
	}
}

func (d *Display) WaitInputFor(timeoutCentiseconds int) Event {
	if d == nil {
		return Event{}
	}
	if timeoutCentiseconds <= 0 {
		return d.WaitInput()
	}
	switch kos.WaitEventFor(uint32(timeoutCentiseconds)) {
	case kos.EventNone:
		return Event{Type: EventNone}
	case kos.EventMouse:
		return Event{Type: EventMouseInput, Mouse: d.currentMouse()}
	case kos.EventKey:
		if key := mapKeyEvent(kos.ReadKey()); key != 0 {
			return Event{Type: EventKeyInput, Key: key}
		}
		return Event{Type: EventNone}
	case kos.EventRedraw:
		d.fullRedraw = true
		return Event{Type: EventResize}
	case kos.EventButton:
		if kos.CurrentButtonID() == 1 {
			return Event{Type: EventClose}
		}
	}
	return Event{Type: EventNone}
}

func (d *Display) Close() {
	if d == nil || d.closed {
		return
	}
	d.closed = true
	close(d.stop)
}

func (d *Display) WriteSnarf(data []byte) error {
	if status := kos.ClipboardCopyText(string(data)); status != kos.ClipboardOK {
		return errors.New("clipboard write failed")
	}
	return nil
}

func (d *Display) ReadSnarf([]byte) (int, int, error) {
	return 0, 0, errors.New("clipboard read is not implemented")
}
