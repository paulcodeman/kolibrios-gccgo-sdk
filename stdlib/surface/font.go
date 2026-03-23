package surface

import (
	"image"
	"os"
	"sync"
	"unicode/utf8"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type FontMetrics struct {
	Width  int
	Height int
	Ascent int
}

type Font struct {
	path    string
	size    int
	face    xfont.Face
	metrics FontMetrics

	mu       sync.Mutex
	advances map[rune]fixed.Int26_6
	kerns    map[uint64]fixed.Int26_6
	glyphs   map[glyphCacheKey]cachedGlyph
}

type fontKey struct {
	path string
	size int
}

type glyphCacheKey struct {
	r     rune
	phase uint8
}

type cachedGlyph struct {
	rect    image.Rectangle
	mask    image.Image
	maskp   image.Point
	advance fixed.Int26_6
	ok      bool
}

var fontCache = struct {
	sync.Mutex
	entries map[fontKey]*Font
}{
	entries: map[fontKey]*Font{},
}

func GetFont(path string, size int) *Font {
	if path == "" {
		return nil
	}
	if size <= 0 {
		size = DefaultFontHeight
	}
	key := fontKey{path: path, size: size}
	fontCache.Lock()
	if entry, ok := fontCache.entries[key]; ok {
		fontCache.Unlock()
		return entry
	}
	fontCache.Unlock()
	entry := loadFont(path, size)
	fontCache.Lock()
	fontCache.entries[key] = entry
	fontCache.Unlock()
	return entry
}

func loadFont(path string, size int) *Font {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	parsed, err := opentype.Parse(data)
	if err != nil {
		return nil
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    float64(size),
		DPI:     72,
		Hinting: xfont.HintingFull,
	})
	if err != nil {
		return nil
	}
	return &Font{
		path:     path,
		size:     size,
		face:     face,
		metrics:  computeFontMetrics(face),
		advances: map[rune]fixed.Int26_6{},
		kerns:    map[uint64]fixed.Int26_6{},
		glyphs:   map[glyphCacheKey]cachedGlyph{},
	}
}

func DefaultFontMetrics() FontMetrics {
	return DefaultFontMetricsForSize(DefaultFontHeight)
}

func DefaultFontMetricsForSize(size int) FontMetrics {
	if size <= 0 {
		size = DefaultFontHeight
	}
	width := size / 2
	if width < DefaultCharWidth {
		width = DefaultCharWidth
	}
	ascent := size * 3 / 4
	if ascent <= 0 {
		ascent = 1
	}
	return FontMetrics{
		Width:  width,
		Height: size,
		Ascent: ascent,
	}
}

func computeFontMetrics(face xfont.Face) FontMetrics {
	if face == nil {
		return DefaultFontMetrics()
	}
	m := face.Metrics()
	height := m.Height.Ceil()
	if height <= 0 {
		height = DefaultFontHeight
	}
	ascent := m.Ascent.Ceil()
	if ascent <= 0 {
		ascent = height * 3 / 4
	}
	if ascent > height {
		ascent = height
	}
	width := glyphAdvancePixels(face, 'M')
	if width <= 0 {
		width = glyphAdvancePixels(face, '0')
	}
	if width <= 0 {
		width = glyphAdvancePixels(face, ' ')
	}
	if width <= 0 {
		width = DefaultCharWidth
	}
	return FontMetrics{Width: width, Height: height, Ascent: ascent}
}

func glyphAdvancePixels(face xfont.Face, r rune) int {
	if face == nil {
		return 0
	}
	advance, ok := face.GlyphAdvance(r)
	if !ok {
		return 0
	}
	width := advance.Ceil()
	if width < 0 {
		width = -width
	}
	return width
}

func glyphPairKey(left rune, right rune) uint64 {
	return uint64(uint32(left))<<32 | uint64(uint32(right))
}

func fixedFloor(value fixed.Int26_6) int {
	return int(value >> 6)
}

func fixedPhase(value fixed.Int26_6) uint8 {
	return uint8(int(value) & 63)
}

func cloneGlyphMask(mask image.Image) image.Image {
	if mask == nil {
		return nil
	}
	if alpha, ok := mask.(*image.Alpha); ok {
		cloned := image.NewAlpha(alpha.Rect)
		for y := alpha.Rect.Min.Y; y < alpha.Rect.Max.Y; y++ {
			src := (y - alpha.Rect.Min.Y) * alpha.Stride
			dst := (y - cloned.Rect.Min.Y) * cloned.Stride
			copy(cloned.Pix[dst:dst+cloned.Rect.Dx()], alpha.Pix[src:src+alpha.Rect.Dx()])
		}
		return cloned
	}
	bounds := mask.Bounds()
	cloned := image.NewAlpha(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		row := (y - bounds.Min.Y) * cloned.Stride
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := mask.At(x, y).RGBA()
			cloned.Pix[row+(x-bounds.Min.X)] = uint8(a >> 8)
		}
	}
	return cloned
}

func (font *Font) Metrics() FontMetrics {
	if font == nil {
		return DefaultFontMetrics()
	}
	return font.metrics
}

func (font *Font) LineHeight() int {
	return font.Metrics().Height
}

func (font *Font) Ascent() int {
	return font.Metrics().Ascent
}

func (font *Font) MeasureString(text string) int {
	width := font.MeasureStringFixed(text).Ceil()
	if width < 0 {
		return -width
	}
	return width
}

func (font *Font) MeasureStringFixed(text string) fixed.Int26_6 {
	if font == nil || text == "" {
		return 0
	}
	if font.face == nil {
		return fixed.I(utf8.RuneCountInString(text) * font.metrics.Width)
	}
	width := fixed.Int26_6(0)
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			width += font.Kern(prev, r)
		}
		width += font.GlyphAdvance(r)
		prev = r
	}
	return width
}

func (font *Font) GlyphAdvance(r rune) fixed.Int26_6 {
	if font == nil || font.face == nil {
		return 0
	}
	font.mu.Lock()
	if advance, ok := font.advances[r]; ok {
		font.mu.Unlock()
		return advance
	}
	font.mu.Unlock()
	advance, _ := font.face.GlyphAdvance(r)
	font.mu.Lock()
	font.advances[r] = advance
	font.mu.Unlock()
	return advance
}

func (font *Font) Kern(left rune, right rune) fixed.Int26_6 {
	if font == nil || font.face == nil || left < 0 || right < 0 {
		return 0
	}
	key := glyphPairKey(left, right)
	font.mu.Lock()
	if value, ok := font.kerns[key]; ok {
		font.mu.Unlock()
		return value
	}
	font.mu.Unlock()
	value := font.face.Kern(left, right)
	font.mu.Lock()
	font.kerns[key] = value
	font.mu.Unlock()
	return value
}

func (font *Font) Glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	if font == nil || font.face == nil {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	key := glyphCacheKey{r: r, phase: fixedPhase(dot.X)}
	font.mu.Lock()
	if glyph, ok := font.glyphs[key]; ok {
		font.mu.Unlock()
		return glyph.rect.Add(image.Point{X: fixedFloor(dot.X), Y: fixedFloor(dot.Y)}), glyph.mask, glyph.maskp, glyph.advance, glyph.ok
	}
	font.mu.Unlock()
	dr, mask, maskp, advance, ok := font.face.Glyph(dot, r)
	glyph := cachedGlyph{
		rect:    dr.Sub(image.Point{X: fixedFloor(dot.X), Y: fixedFloor(dot.Y)}),
		mask:    cloneGlyphMask(mask),
		maskp:   maskp,
		advance: advance,
		ok:      ok,
	}
	font.mu.Lock()
	font.glyphs[key] = glyph
	font.mu.Unlock()
	return glyph.rect.Add(image.Point{X: fixedFloor(dot.X), Y: fixedFloor(dot.Y)}), glyph.mask, glyph.maskp, glyph.advance, glyph.ok
}
