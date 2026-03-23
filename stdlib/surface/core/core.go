package core

import (
	"unicode/utf8"
	"unsafe"

	"kos"
)

const (
	DefaultCharWidth  = 8
	DefaultFontHeight = 16

	windowClientLeft   = 5
	windowClientRight  = 4
	windowClientBottom = 4

	runtimeMemfill32MinWords = 64
	runtimeMemcpy32MinWords  = 32
)

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

func UnionRect(a Rect, b Rect) Rect {
	if a.Empty() {
		return b
	}
	if b.Empty() {
		return a
	}
	left := a.X
	if b.X < left {
		left = b.X
	}
	top := a.Y
	if b.Y < top {
		top = b.Y
	}
	right := a.X + a.Width
	if value := b.X + b.Width; value > right {
		right = value
	}
	bottom := a.Y + a.Height
	if value := b.Y + b.Height; value > bottom {
		bottom = value
	}
	return Rect{
		X:      left,
		Y:      top,
		Width:  right - left,
		Height: bottom - top,
	}
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
	width   int
	height  int
	data    []uint32
	alpha   bool
	clip    clipState
	stack   []clipState
	scratch []uint32
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
		buffer.scratch = nil
		return
	}
	area := int64(width) * int64(height)
	if area <= 0 || area > int64(^uint(0)>>1)-2 {
		buffer.width = 0
		buffer.height = 0
		buffer.data = nil
		buffer.clip = clipState{}
		buffer.stack = nil
		buffer.scratch = nil
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

func (buffer *Buffer) Clear(color uint32) {
	if buffer == nil || len(buffer.data) < 2 {
		return
	}
	fill32(buffer.data[2:], colorValue(color)|0xFF000000)
}

func (buffer *Buffer) FillRect(x int, y int, width int, height int, color uint32) {
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

func (buffer *Buffer) DrawText(x int, y int, color uint32, text string) {
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
			buffer.drawTextAlphaClipped(x, y, rgb, text, alpha, clip)
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
		buffer.drawTextAlpha(x, y, rgb, text, alpha)
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

func (buffer *Buffer) scratchPixels(size int) []uint32 {
	if buffer == nil || size <= 0 {
		return nil
	}
	if cap(buffer.scratch) < size {
		buffer.scratch = make([]uint32, size)
	} else {
		buffer.scratch = buffer.scratch[:size]
	}
	return buffer.scratch
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
	count := len(slice)
	if count == 0 {
		return
	}
	if count >= runtimeMemfill32MinWords {
		runtimeMemfill32(unsafe.Pointer(&slice[0]), value, uintptr(count))
		return
	}
	slice[0] = value
	for filled := 1; filled < count; {
		copyCount := filled
		if copyCount > count-filled {
			copyCount = count - filled
		}
		copy(slice[filled:filled+copyCount], slice[:copyCount])
		filled += copyCount
	}
}

func copy32(dst []uint32, src []uint32) {
	count := len(dst)
	if count == 0 {
		return
	}
	if len(src) < count {
		count = len(src)
		if count == 0 {
			return
		}
	}
	if count >= runtimeMemcpy32MinWords {
		runtimeMemcpy32(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), uintptr(count))
		return
	}
	copy(dst[:count], src[:count])
}

func move32(dst []uint32, src []uint32) {
	count := len(dst)
	if count == 0 {
		return
	}
	if len(src) < count {
		count = len(src)
		if count == 0 {
			return
		}
	}
	if count >= runtimeMemcpy32MinWords {
		runtimeMemmove32(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), uintptr(count))
		return
	}
	copy(dst[:count], src[:count])
}

func runtimeMemfill32(dst unsafe.Pointer, value uint32, count uintptr) __asm__("runtime.memfill32")
func runtimeMemcpy32(dst unsafe.Pointer, src unsafe.Pointer, count uintptr) __asm__("runtime.memcpy32")
func runtimeMemmove32(dst unsafe.Pointer, src unsafe.Pointer, count uintptr) __asm__("runtime.memmove32")

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
