package ui

import (
	"image"
	"os"
	"sync"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type ttfFont struct {
	path    string
	size    int
	face    xfont.Face
	metrics fontMetrics

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
	entries map[fontKey]*ttfFont
}{
	entries: map[fontKey]*ttfFont{},
}

func fontForStyle(style Style) *ttfFont {
	path, ok := resolveFontPath(style.fontPath)
	if !ok {
		return nil
	}
	size := defaultFontHeight
	if value, ok := resolveFontSize(style.fontSize); ok {
		size = value
	}
	return getTTFFont(path, size)
}

func getTTFFont(path string, size int) *ttfFont {
	if path == "" {
		return nil
	}
	if size <= 0 {
		size = defaultFontHeight
	}
	key := fontKey{path: path, size: size}
	fontCache.Lock()
	if entry, ok := fontCache.entries[key]; ok {
		fontCache.Unlock()
		return entry
	}
	fontCache.Unlock()

	entry := loadTTFFont(path, size)
	fontCache.Lock()
	fontCache.entries[key] = entry
	fontCache.Unlock()
	return entry
}

func loadTTFFont(path string, size int) *ttfFont {
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
	metrics := computeFontMetrics(face)
	return &ttfFont{
		path:     path,
		size:     size,
		face:     face,
		metrics:  metrics,
		advances: map[rune]fixed.Int26_6{},
		kerns:    map[uint64]fixed.Int26_6{},
		glyphs:   map[glyphCacheKey]cachedGlyph{},
	}
}

func computeFontMetrics(face xfont.Face) fontMetrics {
	if face == nil {
		return defaultFontMetrics()
	}
	m := face.Metrics()
	height := m.Height.Ceil()
	if height <= 0 {
		height = defaultFontHeight
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
		width = defaultCharWidth
	}
	return fontMetrics{
		width:  width,
		height: height,
		ascent: ascent,
	}
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

func (font *ttfFont) glyphAdvance(r rune) fixed.Int26_6 {
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

func (font *ttfFont) kern(left rune, right rune) fixed.Int26_6 {
	if font == nil || font.face == nil || left < 0 || right < 0 {
		return 0
	}
	key := glyphPairKey(left, right)
	font.mu.Lock()
	if kern, ok := font.kerns[key]; ok {
		font.mu.Unlock()
		return kern
	}
	font.mu.Unlock()
	kern := font.face.Kern(left, right)
	font.mu.Lock()
	font.kerns[key] = kern
	font.mu.Unlock()
	return kern
}

func (font *ttfFont) glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
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

	cacheDot := fixed.Point26_6{
		X: fixed.Int26_6(key.phase),
		Y: 0,
	}
	rect, mask, maskp, advance, ok := font.face.Glyph(cacheDot, r)
	glyph := cachedGlyph{
		rect:    rect,
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

func (font *ttfFont) measureStringFixed(text string) fixed.Int26_6 {
	if font == nil || font.face == nil || text == "" {
		return 0
	}
	var width fixed.Int26_6
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			width += font.kern(prev, r)
		}
		width += font.glyphAdvance(r)
		prev = r
	}
	if width < 0 {
		return -width
	}
	return width
}
