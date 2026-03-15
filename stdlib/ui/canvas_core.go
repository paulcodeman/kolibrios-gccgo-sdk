package ui

import "unsafe"

type Canvas struct {
	width     int
	height    int
	data      []uint32
	clip      clipState
	clipStack []clipState
	alpha     bool
}

type clipState struct {
	rect Rect
	set  bool
}

func NewCanvas(width int, height int) *Canvas {
	canvas := &Canvas{}
	canvas.Resize(width, height)
	return canvas
}

func NewCanvasAlpha(width int, height int) *Canvas {
	canvas := &Canvas{alpha: true}
	canvas.Resize(width, height)
	return canvas
}

func (canvas *Canvas) Width() int {
	if canvas == nil {
		return 0
	}
	return canvas.width
}

func (canvas *Canvas) Height() int {
	if canvas == nil {
		return 0
	}
	return canvas.height
}

func (canvas *Canvas) Resize(width int, height int) {
	if canvas == nil {
		return
	}
	if width <= 0 || height <= 0 {
		canvas.width = 0
		canvas.height = 0
		canvas.data = nil
		return
	}
	area := int64(width) * int64(height)
	if area <= 0 || area > int64(maxInt-2) {
		canvas.width = 0
		canvas.height = 0
		canvas.data = nil
		return
	}
	size := 2 + int(area)
	if canvas.data == nil || len(canvas.data) != size {
		canvas.data = make([]uint32, size)
	}
	canvas.width = width
	canvas.height = height
	if len(canvas.data) >= 2 {
		canvas.data[0] = uint32(width)
		canvas.data[1] = uint32(height)
	}
	canvas.clip = clipState{}
	canvas.clipStack = nil
}

func (canvas *Canvas) PushClip(rect Rect) {
	if canvas == nil {
		return
	}
	canvas.clipStack = append(canvas.clipStack, canvas.clip)
	if rect.Empty() {
		canvas.clip = clipState{rect: Rect{}, set: true}
		return
	}
	canvasRect := Rect{X: 0, Y: 0, Width: canvas.width, Height: canvas.height}
	clip := IntersectRect(rect, canvasRect)
	if canvas.clip.set {
		clip = IntersectRect(clip, canvas.clip.rect)
	}
	canvas.clip = clipState{rect: clip, set: true}
}

func (canvas *Canvas) PopClip() {
	if canvas == nil {
		return
	}
	if len(canvas.clipStack) == 0 {
		canvas.clip = clipState{}
		return
	}
	last := canvas.clipStack[len(canvas.clipStack)-1]
	canvas.clipStack = canvas.clipStack[:len(canvas.clipStack)-1]
	canvas.clip = last
}

func (canvas *Canvas) clampRect(x int, y int, width int, height int) (int, int, int, int, bool) {
	if canvas == nil || width <= 0 || height <= 0 {
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
	if x >= canvas.width || y >= canvas.height || width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	if x+width > canvas.width {
		width = canvas.width - x
	}
	if y+height > canvas.height {
		height = canvas.height - y
	}
	if width <= 0 || height <= 0 {
		return 0, 0, 0, 0, false
	}
	if canvas.clip.set {
		clip := canvas.clip.rect
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

func (canvas *Canvas) headerPtr() *byte {
	if canvas == nil || len(canvas.data) == 0 {
		return nil
	}
	return (*byte)(unsafe.Pointer(&canvas.data[0]))
}

func (canvas *Canvas) pixelPtr(offsetPixels int) *byte {
	if canvas == nil {
		return nil
	}
	index := 2 + offsetPixels
	if index < 2 || index >= len(canvas.data) {
		return nil
	}
	return (*byte)(unsafe.Pointer(&canvas.data[index]))
}
