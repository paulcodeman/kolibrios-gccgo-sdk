//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kolibrios

import (
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/internal/driver"
	"fyne.io/fyne/v2/internal/painter"
	paintsoft "fyne.io/fyne/v2/internal/painter/software"
	"kos"
)

type driverState struct {
	device  device
	painter softwarePainter
	windows []fyne.Window
	active  *window
	done    bool
}

var _ fyne.Driver = (*driverState)(nil)
var _ desktop.Driver = (*driverState)(nil)

func NewDriver() fyne.Driver {
	return &driverState{painter: paintsoft.NewPainter()}
}

func (d *driverState) AbsolutePositionForObject(obj fyne.CanvasObject) fyne.Position {
	c := d.CanvasForObject(obj)
	if c == nil {
		return fyne.NewPos(0, 0)
	}
	kCanvas, ok := c.(*canvas)
	if !ok {
		return fyne.NewPos(0, 0)
	}
	return driver.AbsolutePositionForObject(obj, kCanvas.objectTrees())
}

func (d *driverState) AllWindows() []fyne.Window {
	windows := make([]fyne.Window, len(d.windows))
	copy(windows, d.windows)
	return windows
}

func (d *driverState) CanvasForObject(fyne.CanvasObject) fyne.Canvas {
	if d.active != nil {
		return d.active.Canvas()
	}
	if len(d.windows) == 0 {
		return nil
	}
	return d.windows[len(d.windows)-1].Canvas()
}

func (d *driverState) CreateSplashWindow() fyne.Window {
	window := d.CreateWindow("")
	window.SetPadded(false)
	return window
}

func (d *driverState) CreateWindow(title string) fyne.Window {
	window := &window{
		title:     title,
		canvas:    newCanvas(d.painter),
		clipboard: &clipboard{},
		driver:    d,
		x:         40,
		y:         40,
	}
	d.windows = append(d.windows, window)
	if d.active == nil {
		d.active = window
	}
	return window
}

func (d *driverState) Device() fyne.Device {
	return &d.device
}

func (d *driverState) Quit() {
	d.done = true
}

func (d *driverState) RenderedTextSize(text string, fontSize float32, style fyne.TextStyle) (fyne.Size, float32) {
	return painter.RenderedTextSize(text, fontSize, style)
}

func (d *driverState) Run() {
	runtime.LockOSThread()
	kos.SwapEventMask(kos.DefaultEventMask | kos.EventMaskMouse | kos.EventMaskMouseActiveWindowOnly)
	if d.active != nil {
		d.active.needsFullRedraw = true
	}
	d.renderActive()

	for !d.done {
		if d.active == nil {
			return
		}
		switch kos.WaitEvent() {
		case kos.EventNone:
		case kos.EventRedraw:
			if d.active != nil {
				d.active.needsFullRedraw = true
			}
			d.renderActive()
		case kos.EventMouse:
			if d.active.handleMouse() {
				d.renderActive()
			}
		case kos.EventKey:
			if d.active.handleKey() {
				d.renderActive()
			}
		case kos.EventButton:
			if kos.CurrentButtonID() == 1 && d.active != nil {
				if d.active.requestClose() {
					d.renderActive()
				}
			}
		}
	}
}

func (d *driverState) StartAnimation(a *fyne.Animation) {
	if a == nil || a.Tick == nil {
		return
	}
	a.Tick(1.0)
	d.renderActive()
}

func (d *driverState) StopAnimation(*fyne.Animation) {
}

func (d *driverState) activate(window *window) {
	if window == nil {
		return
	}
	d.active = window
}

func (d *driverState) removeWindow(target *window) {
	if target == nil {
		return
	}
	index := -1
	for i, window := range d.windows {
		if window == target {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	copy(d.windows[index:], d.windows[index+1:])
	d.windows[len(d.windows)-1] = nil
	d.windows = d.windows[:len(d.windows)-1]

	if target.master || len(d.windows) == 0 {
		d.active = nil
		d.done = true
		return
	}
	if d.active == target {
		d.active = nil
		for i := len(d.windows) - 1; i >= 0; i-- {
			if candidate, ok := d.windows[i].(*window); ok && candidate.shown {
				d.active = candidate
				break
			}
		}
		if d.active == nil {
			if candidate, ok := d.windows[len(d.windows)-1].(*window); ok {
				d.active = candidate
			}
		}
	}
}

func (d *driverState) renderActive() {
	if d.active == nil || !d.active.shown {
		return
	}
	d.active.render()
}
