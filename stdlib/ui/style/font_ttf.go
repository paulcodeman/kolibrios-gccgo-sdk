package ui

import (
	"image"
	"sync"

	"golang.org/x/image/math/fixed"
	"surface"
)

type ttfFont struct {
	path    string
	size    int
	surface *surface.Font
	metrics fontMetrics
}

type fontKey struct {
	path string
	size int
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
	sf := surface.GetFont(path, size)
	if sf == nil {
		return nil
	}
	entry := &ttfFont{
		path:    path,
		size:    size,
		surface: sf,
		metrics: fontMetricsFromSurface(sf.Metrics()),
	}
	fontCache.Lock()
	fontCache.entries[key] = entry
	fontCache.Unlock()
	return entry
}

func fontMetricsFromSurface(metrics surface.FontMetrics) fontMetrics {
	return fontMetrics{
		width:  metrics.Width,
		height: metrics.Height,
		ascent: metrics.Ascent,
	}
}

func (font *ttfFont) available() bool {
	return font != nil && font.surface != nil
}

func (font *ttfFont) glyphAdvance(r rune) fixed.Int26_6 {
	if !font.available() {
		return 0
	}
	return font.surface.GlyphAdvance(r)
}

func (font *ttfFont) kern(left rune, right rune) fixed.Int26_6 {
	if !font.available() {
		return 0
	}
	return font.surface.Kern(left, right)
}

func (font *ttfFont) glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
	if !font.available() {
		return image.Rectangle{}, nil, image.Point{}, 0, false
	}
	return font.surface.Glyph(dot, r)
}

func (font *ttfFont) measureStringFixed(text string) fixed.Int26_6 {
	if !font.available() || text == "" {
		return 0
	}
	return font.surface.MeasureStringFixed(text)
}
