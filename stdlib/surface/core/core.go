package core

import (
	"image"
	"os"
	"sync"
	"unicode/utf8"
	"unsafe"

	xfont "golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	"kos"
)

const (
	DefaultCharWidth  = 8
	DefaultFontHeight = 16

	windowClientLeft   = 5
	windowClientRight  = 4
	windowClientBottom = 4
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

func (font *Font) MeasureString(text string) int {
	if font == nil || text == "" {
		return 0
	}
	if font.face == nil {
		return textColumnCount(text) * font.metrics.Width
	}
	width := fixed.Int26_6(0)
	prev := rune(-1)
	for _, r := range text {
		if prev >= 0 {
			width += font.kern(prev, r)
		}
		width += font.glyphAdvance(r)
		prev = r
	}
	return width.Ceil()
}

func (font *Font) glyphAdvance(r rune) fixed.Int26_6 {
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

func (font *Font) kern(left rune, right rune) fixed.Int26_6 {
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

func (font *Font) glyph(dot fixed.Point26_6, r rune) (image.Rectangle, image.Image, image.Point, fixed.Int26_6, bool) {
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

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type clipState struct {
	rect Rect
	set  bool
}

func (rect Rect) Empty() bool {
	return rect.Width <= 0 || rect.Height <= 0
}

func (rect Rect) Contains(x int, y int) bool {
	return !rect.Empty() &&
		x >= rect.X && y >= rect.Y &&
		x < rect.X+rect.Width && y < rect.Y+rect.Height
}

func IntersectRect(a Rect, b Rect) Rect {
	if a.Empty() || b.Empty() {
		return Rect{}
	}
	left := a.X
	if b.X > left {
		left = b.X
	}
	top := a.Y
	if b.Y > top {
		top = b.Y
	}
	right := a.X + a.Width
	if value := b.X + b.Width; value < right {
		right = value
	}
	bottom := a.Y + a.Height
	if value := b.Y + b.Height; value < bottom {
		bottom = value
	}
	if right <= left || bottom <= top {
		return Rect{}
	}
	return Rect{
		X:      left,
		Y:      top,
		Width:  right - left,
		Height: bottom - top,
	}
}

type Buffer struct {
	width  int
	height int
	data   []uint32
	alpha  bool
	clip   clipState
	stack  []clipState
}

func NewBuffer(width int, height int) *Buffer {
	buffer := &Buffer{}
	buffer.Resize(width, height)
	return buffer
}

func NewBufferAlpha(width int, height int) *Buffer {
	buffer := &Buffer{alpha: true}
	buffer.Resize(width, height)
	return buffer
}

func (buffer *Buffer) Width() int {
	if buffer == nil {
		return 0
	}
	return buffer.width
}

func (buffer *Buffer) Height() int {
	if buffer == nil {
		return 0
	}
	return buffer.height
}

func (buffer *Buffer) Bounds() Rect {
	if buffer == nil {
		return Rect{}
	}
	return Rect{Width: buffer.width, Height: buffer.height}
}

func (buffer *Buffer) Resize(width int, height int) {
	if buffer == nil {
		return
	}
	if width <= 0 || height <= 0 {
		buffer.width = 0
		buffer.height = 0
		buffer.data = nil
		buffer.clip = clipState{}
		buffer.stack = nil
		return
	}
	area := int64(width) * int64(height)
	if area <= 0 || area > int64(^uint(0)>>1)-2 {
		buffer.width = 0
		buffer.height = 0
		buffer.data = nil
		buffer.clip = clipState{}
		buffer.stack = nil
		return
	}
	size := 2 + int(area)
	if len(buffer.data) != size {
		buffer.data = make([]uint32, size)
	}
	buffer.width = width
	buffer.height = height
	buffer.data[0] = uint32(width)
	buffer.data[1] = uint32(height)
	buffer.clip = clipState{}
	buffer.stack = nil
}

func (buffer *Buffer) PushClip(rect Rect) {
	if buffer == nil {
		return
	}
	buffer.stack = append(buffer.stack, buffer.clip)
	if rect.Empty() {
		buffer.clip = clipState{rect: Rect{}, set: true}
		return
	}
	clip := IntersectRect(rect, buffer.Bounds())
	if buffer.clip.set {
		clip = IntersectRect(clip, buffer.clip.rect)
	}
	buffer.clip = clipState{rect: clip, set: true}
}

func (buffer *Buffer) PopClip() {
	if buffer == nil {
		return
	}
	if len(buffer.stack) == 0 {
		buffer.clip = clipState{}
		return
	}
	last := buffer.stack[len(buffer.stack)-1]
	buffer.stack = buffer.stack[:len(buffer.stack)-1]
	buffer.clip = last
}

func (buffer *Buffer) Clear(color kos.Color) {
	if buffer == nil || len(buffer.data) < 2 {
		return
	}
	fill32(buffer.data[2:], colorValue(color)|0xFF000000)
}

func (buffer *Buffer) FillRect(x int, y int, width int, height int, color kos.Color) {
	if buffer == nil {
		return
	}
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	value := colorValue(color) | 0xFF000000
	rowStart := 2 + y*buffer.width + x
	if x == 0 && width == buffer.width {
		fill32(buffer.data[rowStart:rowStart+width*height], value)
		return
	}
	for row := 0; row < height; row++ {
		index := rowStart + row*buffer.width
		fill32(buffer.data[index:index+width], value)
	}
}

func (buffer *Buffer) fillRectValue(x int, y int, width int, height int, value uint32) {
	x, y, width, height, ok := buffer.clampRect(x, y, width, height)
	if !ok {
		return
	}
	rowStart := 2 + y*buffer.width + x
	if x == 0 && width == buffer.width {
		fill32(buffer.data[rowStart:rowStart+width*height], value)
		return
	}
	for row := 0; row < height; row++ {
		index := rowStart + row*buffer.width
		fill32(buffer.data[index:index+width], value)
	}
}

func (buffer *Buffer) ClearTransparent() {
	if buffer == nil || len(buffer.data) < 2 {
		return
	}
	fill32(buffer.data[2:], 0)
}

func (buffer *Buffer) DrawText(x int, y int, color kos.Color, text string) {
	if buffer == nil || text == "" || buffer.width <= 0 || buffer.height <= 0 {
		return
	}
	columns := textColumnCount(text)
	if columns == 0 {
		return
	}
	rgb, alpha := colorValueAndAlpha(color)
	if alpha == 0 {
		return
	}
	if buffer.clip.set {
		clip := buffer.clip.rect
		if clip.Empty() {
			return
		}
		if y+DefaultFontHeight <= clip.Y || y >= clip.Y+clip.Height {
			return
		}
		if x >= clip.X+clip.Width {
			return
		}
		if x+columns*DefaultCharWidth <= clip.X {
			return
		}
		if x < clip.X {
			skip := (clip.X - x + DefaultCharWidth - 1) / DefaultCharWidth
			if skip >= columns {
				return
			}
			text = textSliceColumns(text, skip, columns)
			columns -= skip
			x += skip * DefaultCharWidth
		}
		maxWidth := clip.X + clip.Width - x
		if maxWidth <= 0 {
			return
		}
		maxChars := maxWidth / DefaultCharWidth
		if maxChars <= 0 {
			return
		}
		if columns > maxChars {
			text = textSliceColumns(text, 0, maxChars)
			columns = maxChars
		}
		if text == "" {
			return
		}
		partialY := clip.Y > y || clip.Y+clip.Height < y+DefaultFontHeight
		if partialY || buffer.alpha || alpha < 255 {
			buffer.drawTextAlphaClipped(x, y, kos.Color(rgb), text, alpha, clip)
			return
		}
	}
	if x < 0 || y < 0 || x >= buffer.width || y >= buffer.height {
		return
	}
	if y+DefaultFontHeight > buffer.height {
		return
	}
	if buffer.alpha || alpha < 255 {
		buffer.drawTextAlpha(x, y, kos.Color(rgb), text, alpha)
		return
	}
	maxChars := (buffer.width - x) / DefaultCharWidth
	if maxChars <= 0 {
		return
	}
	if columns > maxChars {
		text = textSliceColumns(text, 0, maxChars)
		if text == "" {
			return
		}
	}
	kos.DrawTextBuffer(x, y, kos.Color(rgb), text, buffer.headerPtr())
}

func (buffer *Buffer) BlitToWindow(x int, y int) {
	if buffer == nil || buffer.width <= 0 || buffer.height <= 0 {
		return
	}
	ptr := buffer.pixelPtr(0)
	if ptr == nil {
		return
	}
	kos.PutImage32(ptr, buffer.width, buffer.height, x, y)
}

func (buffer *Buffer) BlitRectToWindow(rect Rect, x int, y int) {
	if buffer == nil || rect.Empty() {
		return
	}
	cx, cy, cw, ch, ok := buffer.clampRect(rect.X, rect.Y, rect.Width, rect.Height)
	if !ok {
		return
	}
	x += cx - rect.X
	y += cy - rect.Y
	ptr := buffer.pixelPtr(cy*buffer.width + cx)
	if ptr == nil {
		return
	}
	rowOffset := (buffer.width - cw) * 4
	kos.PutPaletteImage(ptr, cw, ch, x, y, 32, nil, rowOffset)
}

func (buffer *Buffer) clampRect(x int, y int, width int, height int) (int, int, int, int, bool) {
	if buffer == nil || width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	if x < 0 {
		width += x
		x = 0
	}
	if y < 0 {
		height += y
		y = 0
	}
	if x >= buffer.width || y >= buffer.height {
		return 0, 0, 0, 0, false
	}
	if x+width > buffer.width {
		width = buffer.width - x
	}
	if y+height > buffer.height {
		height = buffer.height - y
	}
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	if buffer.clip.set {
		clip := buffer.clip.rect
		if clip.Empty() {
			return 0, 0, 0, 0, false
		}
		inter := IntersectRect(Rect{X: x, Y: y, Width: width, Height: height}, clip)
		if inter.Empty() {
			return 0, 0, 0, 0, false
		}
		x = inter.X
		y = inter.Y
		width = inter.Width
		height = inter.Height
	}
	return x, y, width, height, true
}

func (buffer *Buffer) headerPtr() *byte {
	if buffer == nil || len(buffer.data) == 0 {
		return nil
	}
	return (*byte)(unsafe.Pointer(&buffer.data[0]))
}

func (buffer *Buffer) pixelPtr(offsetPixels int) *byte {
	if buffer == nil {
		return nil
	}
	index := 2 + offsetPixels
	if index < 2 || index >= len(buffer.data) {
		return nil
	}
	return (*byte)(unsafe.Pointer(&buffer.data[index]))
}

type Presenter struct {
	X      int
	Y      int
	Width  int
	Height int
	Title  string
	Client Rect
}

func NewPresenter(x int, y int, width int, height int, title string) Presenter {
	return Presenter{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
		Title:  title,
		Client: WindowClientRect(width, height),
	}
}

func (presenter *Presenter) SetTitle(title string) {
	if presenter == nil {
		return
	}
	presenter.Title = title
}

func (presenter *Presenter) SetSize(width int, height int) {
	if presenter == nil {
		return
	}
	presenter.Width = width
	presenter.Height = height
	presenter.Client = WindowClientRect(width, height)
}

func (presenter *Presenter) SetClientRect(rect Rect) {
	if presenter == nil {
		return
	}
	if rect.Width < 0 {
		rect.Width = 0
	}
	if rect.Height < 0 {
		rect.Height = 0
	}
	presenter.Client = rect
}

func (presenter Presenter) PresentFull(buffer *Buffer) {
	kos.BeginRedraw()
	kos.OpenWindow(presenter.X, presenter.Y, presenter.Width, presenter.Height, presenter.Title)
	if buffer != nil {
		buffer.BlitToWindow(presenter.Client.X, presenter.Client.Y)
	}
	kos.EndRedraw()
}

func (presenter Presenter) PresentClient(buffer *Buffer) {
	if buffer == nil {
		return
	}
	buffer.BlitToWindow(presenter.Client.X, presenter.Client.Y)
}

func (presenter Presenter) PresentRect(buffer *Buffer, rect Rect) {
	if buffer == nil || rect.Empty() {
		return
	}
	buffer.BlitRectToWindow(rect, presenter.Client.X+rect.X, presenter.Client.Y+rect.Y)
}

func WindowClientRect(width int, height int) Rect {
	skin := kos.SkinHeight()
	if skin < 0 {
		skin = 0
	}
	x := windowClientLeft
	y := skin
	w := width - windowClientLeft - windowClientRight
	h := height - skin - windowClientBottom
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: x, Y: y, Width: w, Height: h}
}

func fill32(slice []uint32, value uint32) {
	for i := range slice {
		slice[i] = value
	}
}

func textColumnCount(value string) int {
	if value == "" {
		return 0
	}
	if isASCIIString(value) {
		return len(value)
	}
	return utf8.RuneCountInString(value)
}

func textSliceColumns(value string, startCol int, endCol int) string {
	if value == "" {
		return value
	}
	if startCol < 0 {
		startCol = 0
	}
	if endCol < startCol {
		endCol = startCol
	}
	if isASCIIString(value) {
		if startCol > len(value) {
			startCol = len(value)
		}
		if endCol > len(value) {
			endCol = len(value)
		}
		return value[startCol:endCol]
	}
	start := textByteIndexForColumn(value, startCol)
	end := textByteIndexForColumn(value, endCol)
	if start > end {
		start = end
	}
	return value[start:end]
}

func textByteIndexForColumn(value string, col int) int {
	if col <= 0 {
		return 0
	}
	if isASCIIString(value) {
		if col >= len(value) {
			return len(value)
		}
		return col
	}
	count := 0
	for i := 0; i < len(value); {
		if count >= col {
			return i
		}
		_, size := utf8.DecodeRuneInString(value[i:])
		if size <= 0 {
			return i
		}
		i += size
		count++
	}
	return len(value)
}

func isASCIIString(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
