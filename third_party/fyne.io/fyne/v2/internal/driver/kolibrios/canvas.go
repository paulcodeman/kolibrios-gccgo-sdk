//go:build kolibrios && gccgo
// +build kolibrios,gccgo

package kolibrios

import (
	"image"
	draw "image/draw"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/internal"
	intapp "fyne.io/fyne/v2/internal/app"
	"fyne.io/fyne/v2/theme"
)

type softwarePainter interface {
	Paint(fyne.Canvas) image.Image
}

type canvas struct {
	size  fyne.Size
	scale float32

	content  fyne.CanvasObject
	overlays *internal.OverlayStack
	focusMgr *intapp.FocusManager
	hovered  desktop.Hoverable
	padded   bool

	onTypedRune func(rune)
	onTypedKey  func(*fyne.KeyEvent)

	fyne.ShortcutHandler
	painter      softwarePainter
	propertyLock sync.RWMutex
}

func newCanvas(painter softwarePainter) *canvas {
	c := &canvas{
		focusMgr: intapp.NewFocusManager(nil),
		padded:   true,
		painter:  painter,
		scale:    1.0,
		size:     fyne.NewSize(10, 10),
	}
	c.overlays = &internal.OverlayStack{Canvas: c}
	return c
}

func (c *canvas) Capture() image.Image {
	bounds := image.Rect(0, 0, internal.ScaleInt(c, c.Size().Width), internal.ScaleInt(c, c.Size().Height))
	img := image.NewNRGBA(bounds)
	draw.Draw(img, bounds, image.NewUniform(theme.BackgroundColor()), image.Point{}, draw.Src)
	if c.painter != nil {
		draw.Draw(img, bounds, c.painter.Paint(c), image.Point{}, draw.Over)
	}
	return img
}

func (c *canvas) Content() fyne.CanvasObject {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.content
}

func (c *canvas) Focus(obj fyne.Focusable) {
	c.focusManager().Focus(obj)
}

func (c *canvas) FocusNext() {
	c.focusManager().FocusNext()
}

func (c *canvas) FocusPrevious() {
	c.focusManager().FocusPrevious()
}

func (c *canvas) Focused() fyne.Focusable {
	return c.focusManager().Focused()
}

func (c *canvas) InteractiveArea() (fyne.Position, fyne.Size) {
	return fyne.Position{}, c.Size()
}

func (c *canvas) MinSize() fyne.Size {
	c.propertyLock.RLock()
	content := c.content
	padded := c.padded
	c.propertyLock.RUnlock()
	if content == nil {
		return fyne.NewSize(1, 1)
	}
	min := content.MinSize()
	if padded {
		min = min.Add(fyne.NewSize(theme.Padding()*2, theme.Padding()*2))
	}
	return min
}

func (c *canvas) OnTypedKey() func(*fyne.KeyEvent) {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.onTypedKey
}

func (c *canvas) OnTypedRune() func(rune) {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.onTypedRune
}

func (c *canvas) Overlays() fyne.OverlayStack {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.overlays
}

func (c *canvas) Padded() bool {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.padded
}

func (c *canvas) PixelCoordinateForPosition(pos fyne.Position) (int, int) {
	return int(pos.X * c.Scale()), int(pos.Y * c.Scale())
}

func (c *canvas) Refresh(fyne.CanvasObject) {
}

func (c *canvas) Resize(size fyne.Size) {
	c.propertyLock.Lock()
	content := c.content
	overlays := c.overlays
	padded := c.padded
	c.size = size
	c.propertyLock.Unlock()

	if content == nil {
		return
	}

	for _, overlay := range overlays.List() {
		type popupWidget interface {
			fyne.CanvasObject
			ShowAtPosition(fyne.Position)
		}
		if p, ok := overlay.(popupWidget); ok {
			p.Refresh()
		} else {
			overlay.Resize(size)
		}
	}

	if padded {
		content.Resize(size.Subtract(fyne.NewSize(theme.Padding()*2, theme.Padding()*2)))
		content.Move(fyne.NewPos(theme.Padding(), theme.Padding()))
	} else {
		content.Resize(size)
		content.Move(fyne.NewPos(0, 0))
	}
}

func (c *canvas) Scale() float32 {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.scale
}

func (c *canvas) SetContent(content fyne.CanvasObject) {
	c.propertyLock.Lock()
	c.content = content
	c.focusMgr = intapp.NewFocusManager(c.content)
	c.propertyLock.Unlock()

	if content == nil {
		return
	}

	padding := fyne.NewSize(0, 0)
	if c.Padded() {
		padding = fyne.NewSize(theme.Padding()*2, theme.Padding()*2)
	}
	c.Resize(content.MinSize().Add(padding))
}

func (c *canvas) SetOnTypedKey(handler func(*fyne.KeyEvent)) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	c.onTypedKey = handler
}

func (c *canvas) SetOnTypedRune(handler func(rune)) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	c.onTypedRune = handler
}

func (c *canvas) SetPadded(padded bool) {
	c.propertyLock.Lock()
	c.padded = padded
	c.propertyLock.Unlock()
	c.Resize(c.Size())
}

func (c *canvas) SetScale(scale float32) {
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	c.scale = scale
}

func (c *canvas) Size() fyne.Size {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	return c.size
}

func (c *canvas) Unfocus() {
	c.focusManager().Focus(nil)
}

func (c *canvas) focusManager() *intapp.FocusManager {
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	if focusMgr := c.overlays.TopFocusManager(); focusMgr != nil {
		return focusMgr
	}
	return c.focusMgr
}

func (c *canvas) objectTrees() []fyne.CanvasObject {
	trees := make([]fyne.CanvasObject, 0, len(c.Overlays().List())+1)
	if c.content != nil {
		trees = append(trees, c.content)
	}
	trees = append(trees, c.Overlays().List()...)
	return trees
}
