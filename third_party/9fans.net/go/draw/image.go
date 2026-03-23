package draw

import (
	"image"
	"image/color"
	"math"

	core "surface/core"
)

const replClipExtent = 1 << 30

type Image struct {
	Display *Display
	R       Rectangle
	Clipr   Rectangle
	Pix     Pix
	Depth   int
	Repl    bool

	buffer *core.Buffer
}

func newImage(display *Display, r Rectangle, pix Pix, repl bool) *Image {
	size := r.Size()
	buffer := core.NewBuffer(size.X, size.Y)
	if pixHasAlpha(pix) {
		buffer = core.NewBufferAlpha(size.X, size.Y)
	}
	clipr := r
	if repl {
		clipr = image.Rect(-replClipExtent, -replClipExtent, replClipExtent, replClipExtent)
	}
	return &Image{
		Display: display,
		R:       r,
		Clipr:   clipr,
		Pix:     pix,
		Depth:   32,
		Repl:    repl,
		buffer:  buffer,
	}
}

func (img *Image) ColorModel() color.Model {
	return colorModel
}

func (img *Image) Bounds() Rectangle {
	if img == nil {
		return image.ZR
	}
	return img.Clipr
}

func (img *Image) At(x int, y int) color.Color {
	value, ok := img.pixelAtPlane(x, y)
	if !ok {
		return color.RGBA{}
	}
	a := uint8(value >> 24)
	r := uint8((value >> 16) & 0xFF)
	g := uint8((value >> 8) & 0xFF)
	b := uint8(value & 0xFF)
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func (img *Image) Set(x int, y int, c color.Color) {
	if img == nil {
		return
	}
	model := colorModel.Convert(c).(Color)
	img.setPixelPlaneValue(x, y, premultipliedARGB(model))
}

func (img *Image) fill(c Color) {
	if img == nil || img.buffer == nil {
		return
	}
	value := premultipliedARGB(c)
	if value>>24 == 0 && img.buffer.HasAlpha() {
		img.buffer.ClearTransparent()
		return
	}
	img.buffer.Clear(value)
}

func (img *Image) resizeRect(r Rectangle) {
	if img == nil {
		return
	}
	img.R = r
	img.Clipr = r
	if img.buffer != nil {
		img.buffer.Resize(r.Dx(), r.Dy())
	}
}

func (img *Image) Free() {
	if img == nil {
		return
	}
	img.buffer = nil
}

func (img *Image) ReplClipr(repl bool, clipr Rectangle) {
	if img == nil {
		return
	}
	img.Repl = repl
	img.Clipr = clipr
}

func (img *Image) Load(r Rectangle, data []byte) (int, error) {
	if img == nil || img.buffer == nil {
		return 0, nil
	}
	r = r.Intersect(img.R)
	if r.Empty() {
		return 0, nil
	}
	width := r.Dx()
	height := r.Dy()
	need := width * height * 4
	if len(data) < need {
		need = len(data) - len(data)%4
	}
	pixels := need / 4
	for i := 0; i < pixels; i++ {
		src := data[i*4:]
		x := r.Min.X + (i % width)
		y := r.Min.Y + (i / width)
		value := decodeLoadedPixel(img.Pix, src[0], src[1], src[2], src[3])
		img.setPixelPlaneValue(x, y, value)
	}
	return pixels * 4, nil
}

func decodeLoadedPixel(pix Pix, b0 byte, b1 byte, b2 byte, b3 byte) uint32 {
	switch pix {
	case ARGB32:
		return uint32(b3)<<24 | uint32(b2)<<16 | uint32(b1)<<8 | uint32(b0)
	case ABGR32:
		return premultipliedARGB(Color(uint32(b0)<<24 | uint32(b1)<<16 | uint32(b2)<<8 | uint32(b3)))
	case RGBA32:
		return premultipliedARGB(Color(uint32(b0)<<24 | uint32(b1)<<16 | uint32(b2)<<8 | uint32(b3)))
	default:
		return premultipliedARGB(Color(uint32(b0)<<24 | uint32(b1)<<16 | uint32(b2)<<8 | uint32(0xFF)))
	}
}

func premultipliedARGB(c Color) uint32 {
	r := uint32(c >> 24)
	g := uint32(c>>16) & 0xFF
	b := uint32(c>>8) & 0xFF
	a := uint32(c) & 0xFF
	return a<<24 | r<<16 | g<<8 | b
}

func (img *Image) planeToLocal(x int, y int) (int, int, bool) {
	if img == nil || img.buffer == nil {
		return 0, 0, false
	}
	if !image.Pt(x, y).In(img.Clipr) {
		return 0, 0, false
	}
	if img.Repl {
		if img.R.Dx() <= 0 || img.R.Dy() <= 0 {
			return 0, 0, false
		}
		x = wrapCoord(x, img.R.Min.X, img.R.Max.X)
		y = wrapCoord(y, img.R.Min.Y, img.R.Max.Y)
	} else if !image.Pt(x, y).In(img.R) {
		return 0, 0, false
	}
	return x - img.R.Min.X, y - img.R.Min.Y, true
}

func wrapCoord(value int, min int, max int) int {
	size := max - min
	if size <= 0 {
		return min
	}
	value -= min
	value %= size
	if value < 0 {
		value += size
	}
	return min + value
}

func (img *Image) pixelAtPlane(x int, y int) (uint32, bool) {
	lx, ly, ok := img.planeToLocal(x, y)
	if !ok {
		return 0, false
	}
	return img.buffer.PixelValue(lx, ly), true
}

func (img *Image) setPixelPlaneValue(x int, y int, value uint32) {
	if img == nil || img.buffer == nil || !image.Pt(x, y).In(img.R) {
		return
	}
	lx := x - img.R.Min.X
	ly := y - img.R.Min.Y
	if img.buffer.HasAlpha() {
		img.buffer.BlendPremultipliedPixelValue(lx, ly, value)
		return
	}
	alpha := uint8(value >> 24)
	if alpha >= 255 {
		img.buffer.SetPixelValue(lx, ly, 0xFF000000|(value&0xFFFFFF))
		return
	}
	img.buffer.BlendPremultipliedPixelValue(lx, ly, value)
}

func (dst *Image) Draw(r Rectangle, src *Image, mask *Image, p Point) {
	dst.DrawOp(r, src, mask, p, Over)
}

func (dst *Image) DrawOp(r Rectangle, src *Image, mask *Image, p Point, op Op) {
	_ = op
	if dst == nil || dst.buffer == nil {
		return
	}
	if src == nil {
		src = dst.Display.Black
	}
	if mask != nil {
		_ = mask
	}
	r = r.Intersect(dst.R).Intersect(dst.Clipr)
	if r.Empty() {
		return
	}
	if mask == nil && dst.fastDrawRect(r, src, p) {
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			sp := image.Pt(p.X+(x-r.Min.X), p.Y+(y-r.Min.Y))
			value, ok := src.pixelAtPlane(sp.X, sp.Y)
			if !ok {
				continue
			}
			dst.setPixelPlaneValue(x, y, value)
		}
	}
}

func (dst *Image) fastDrawRect(r Rectangle, src *Image, p Point) bool {
	if dst == nil || dst.buffer == nil || src == nil || src.buffer == nil {
		return false
	}
	if src.Repl && src.R.Dx() == 1 && src.R.Dy() == 1 {
		value, ok := src.pixelAtPlane(p.X, p.Y)
		if !ok {
			return true
		}
		if value>>24 == 0xFF {
			dst.buffer.FillRect(r.Min.X-dst.R.Min.X, r.Min.Y-dst.R.Min.Y, r.Dx(), r.Dy(), value)
			return true
		}
	}
	if src.Repl {
		return false
	}
	srcRect := image.Rect(p.X, p.Y, p.X+r.Dx(), p.Y+r.Dy())
	visible := srcRect.Intersect(src.R).Intersect(src.Clipr)
	if visible.Empty() {
		return true
	}
	dstRect := r
	dstRect.Min.X += visible.Min.X - srcRect.Min.X
	dstRect.Min.Y += visible.Min.Y - srcRect.Min.Y
	dstRect.Max = dstRect.Min.Add(visible.Size())
	if dstRect.Empty() {
		return true
	}
	dst.buffer.BlitFrom(
		src.buffer,
		core.Rect{
			X:      visible.Min.X - src.R.Min.X,
			Y:      visible.Min.Y - src.R.Min.Y,
			Width:  visible.Dx(),
			Height: visible.Dy(),
		},
		dstRect.Min.X-dst.R.Min.X,
		dstRect.Min.Y-dst.R.Min.Y,
	)
	return true
}

func (dst *Image) String(p Point, src *Image, sp Point, f *Font, s string) Point {
	_ = sp
	return dst.drawString(p, src, f, s, nil)
}

func (dst *Image) StringBg(p Point, src *Image, sp Point, f *Font, s string, bg *Image, bgp Point) Point {
	_ = sp
	_ = bgp
	return dst.drawString(p, src, f, s, bg)
}

func (dst *Image) drawString(p Point, src *Image, f *Font, s string, bg *Image) Point {
	if dst == nil || dst.buffer == nil || s == "" {
		return p
	}
	if f == nil {
		f = dst.Display.DefaultFont
	}
	if src == nil {
		src = dst.Display.Black
	}
	if bg != nil {
		width := f.StringWidth(s)
		height := f.Height
		bgRect := image.Rect(p.X, p.Y, p.X+width, p.Y+height)
		dst.Draw(bgRect, bg, nil, bg.R.Min)
	}
	colorValue := colorAtImage(src)
	if f.surface != nil {
		dst.buffer.DrawTextFont(p.X, p.Y, colorValue, s, f.surface)
	} else {
		dst.buffer.DrawText(p.X, p.Y, colorValue, s)
	}
	p.X += f.StringWidth(s)
	return p
}

func colorAtImage(img *Image) uint32 {
	if img == nil {
		return premultipliedARGB(Black)
	}
	value, ok := img.pixelAtPlane(img.R.Min.X, img.R.Min.Y)
	if !ok {
		return premultipliedARGB(Black)
	}
	return value
}

func (dst *Image) drawLineColor(p0 Point, p1 Point, thick int, value uint32) {
	if dst == nil || dst.buffer == nil {
		return
	}
	if thick <= 0 {
		thick = 1
	}
	if thick == 1 {
		dst.buffer.DrawLine(p0.X-dst.R.Min.X, p0.Y-dst.R.Min.Y, p1.X-dst.R.Min.X, p1.Y-dst.R.Min.Y, value)
		return
	}
	half := thick / 2
	for dy := -half; dy <= half; dy++ {
		dst.buffer.DrawLine(p0.X-dst.R.Min.X, p0.Y-dst.R.Min.Y+dy, p1.X-dst.R.Min.X, p1.Y-dst.R.Min.Y+dy, value)
	}
}

func (dst *Image) Line(p0 Point, p1 Point, _ int, _ int, thick int, src *Image, _ Point) {
	dst.drawLineColor(p0, p1, thick, colorAtImage(src))
}

func (dst *Image) Arc(center Point, rx int, ry int, _ int, src *Image, _ Point, alpha int, phi int) {
	if dst == nil || dst.buffer == nil || rx <= 0 || ry <= 0 {
		return
	}
	start := float64(alpha) * math.Pi / 180
	sweep := float64(phi) * math.Pi / 180
	steps := int(math.Max(float64(rx+ry), 24))
	if steps < 1 {
		steps = 1
	}
	prev := image.Pt(
		center.X+int(math.Round(float64(rx)*math.Cos(start))),
		center.Y-int(math.Round(float64(ry)*math.Sin(start))),
	)
	colorValue := colorAtImage(src)
	for i := 1; i <= steps; i++ {
		t := start + sweep*float64(i)/float64(steps)
		next := image.Pt(
			center.X+int(math.Round(float64(rx)*math.Cos(t))),
			center.Y-int(math.Round(float64(ry)*math.Sin(t))),
		)
		dst.drawLineColor(prev, next, 1, colorValue)
		prev = next
	}
}
