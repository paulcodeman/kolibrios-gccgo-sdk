package draw

import (
	"fmt"
	"image"
	"image/color"
)

type Point = image.Point
type Rectangle = image.Rectangle

var ZP = image.ZP

func Pt(x int, y int) Point {
	return image.Pt(x, y)
}

func Rect(x0 int, y0 int, x1 int, y1 int) Rectangle {
	return image.Rect(x0, y0, x1, y1)
}

const DefaultDPI = 100

type Color uint32

func (c Color) RGBA() (r uint32, g uint32, b uint32, a uint32) {
	r = uint32(c >> 24)
	g = uint32(c>>16) & 0xFF
	b = uint32(c>>8) & 0xFF
	a = uint32(c) & 0xFF
	return r | r<<8, g | g<<8, b | b<<8, a | a<<8
}

func (c Color) WithAlpha(alpha uint8) Color {
	r := uint32(c >> 24)
	g := uint32(c>>16) & 0xFF
	b := uint32(c>>8) & 0xFF
	r = (r * uint32(alpha)) / 255
	g = (g * uint32(alpha)) / 255
	b = (b * uint32(alpha)) / 255
	return Color(r<<24 | g<<16 | b<<8 | uint32(alpha))
}

const (
	Transparent   Color = 0x00000000
	Opaque        Color = 0xFFFFFFFF
	Black         Color = 0x000000FF
	White         Color = 0xFFFFFFFF
	Red           Color = 0xFF0000FF
	Green         Color = 0x00FF00FF
	Blue          Color = 0x0000FFFF
	Cyan          Color = 0x00FFFFFF
	Magenta       Color = 0xFF00FFFF
	Yellow        Color = 0xFFFF00FF
	PaleYellow    Color = 0xFFFFAAFF
	DarkYellow    Color = 0xEEEE9EFF
	DarkGreen     Color = 0x448844FF
	PaleGreen     Color = 0xAAFFAAFF
	MedGreen      Color = 0x88CC88FF
	DarkBlue      Color = 0x000055FF
	PaleBlueGreen Color = 0xAAFFFFFF
	PaleBlue      Color = 0x0000BBFF
	BlueGreen     Color = 0x008888FF
	GreyBlue      Color = 0x005DBBFF
	PurpleBlue    Color = 0x8888CCFF
	NoFill        Color = 0xFFFFFF00
)

type Pix uint32

const (
	CRed = iota
	CGreen
	CBlue
	CGrey
	CAlpha
	CMap
	CIgnore
)

func MakePix(list ...int) Pix {
	var p Pix
	for _, x := range list {
		p <<= 4
		p |= Pix(x)
	}
	return p
}

var (
	GREY1  Pix = MakePix(CGrey, 1)
	GREY2  Pix = MakePix(CGrey, 2)
	GREY4  Pix = MakePix(CGrey, 4)
	GREY8  Pix = MakePix(CGrey, 8)
	CMAP8  Pix = MakePix(CMap, 8)
	RGB24      = MakePix(CRed, 8, CGreen, 8, CBlue, 8)
	BGR24      = MakePix(CBlue, 8, CGreen, 8, CRed, 8)
	RGBA32     = MakePix(CRed, 8, CGreen, 8, CBlue, 8, CAlpha, 8)
	ARGB32     = MakePix(CAlpha, 8, CRed, 8, CGreen, 8, CBlue, 8)
	ABGR32     = MakePix(CAlpha, 8, CBlue, 8, CGreen, 8, CRed, 8)
	XRGB32     = MakePix(CIgnore, 8, CRed, 8, CGreen, 8, CBlue, 8)
	XBGR32     = MakePix(CIgnore, 8, CBlue, 8, CGreen, 8, CRed, 8)
)

type Op int

const (
	Clear Op = 0
	SinD  Op = 8
	DinS  Op = 4
	SoutD Op = 2
	DoutS Op = 1

	S      = SinD | SoutD
	SoverD = SinD | SoutD | DoutS
	SatopD = SinD | DoutS
	SxorD  = SoutD | DoutS
)

const (
	Over Op = SoverD
	Src  Op = S
)

const (
	Refnone   = 1
	Refmesg   = 2
	RefMesg   = Refmesg
	Refbackup = 0
)

const (
	KeyFn        = '\uF000'
	KeyHome      = KeyFn | 0x0D
	KeyUp        = KeyFn | 0x0E
	KeyPageUp    = KeyFn | 0x0F
	KeyPrint     = KeyFn | 0x10
	KeyLeft      = KeyFn | 0x11
	KeyRight     = KeyFn | 0x12
	KeyDown      = 0x80
	KeyView      = 0x80
	KeyPageDown  = KeyFn | 0x13
	KeyInsert    = KeyFn | 0x14
	KeyEnd       = KeyFn | 0x18
	KeyAlt       = KeyFn | 0x15
	KeyShift     = KeyFn | 0x16
	KeyCtl       = KeyFn | 0x17
	KeyBackspace = 0x08
	KeyDelete    = 0x7F
	KeyEscape    = 0x1B
	KeyEOF       = 0x04
	KeyCmd       = 0xF100
)

var colorModel = color.ModelFunc(func(value color.Color) color.Color {
	if c, ok := value.(Color); ok {
		return c
	}
	r, g, b, a := value.RGBA()
	if a == 0 {
		return Transparent
	}
	return Color(uint32(uint8(r>>8))<<24 |
		uint32(uint8(g>>8))<<16 |
		uint32(uint8(b>>8))<<8 |
		uint32(uint8(a>>8)))
})

func pixHasAlpha(p Pix) bool {
	switch p {
	case RGBA32, ARGB32, ABGR32:
		return true
	default:
		return false
	}
}

func parseDimensions(size string) (int, int, error) {
	if size == "" {
		return 800, 600, nil
	}
	var width, height int
	if _, err := fmt.Sscanf(size, "%dx%d", &width, &height); err != nil {
		return 0, 0, err
	}
	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("invalid size %q", size)
	}
	return width, height, nil
}
