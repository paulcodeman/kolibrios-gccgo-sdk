package ui

import surfacepkg "surface"

type Canvas struct {
	buffer    *surfacepkg.Buffer
	clip      clipState
	clipStack []clipState
	alpha     bool
}

type clipState struct {
	rect Rect
	set  bool
}

func NewCanvas(width int, height int) *Canvas {
	return &Canvas{buffer: surfacepkg.NewBuffer(width, height)}
}

func NewCanvasAlpha(width int, height int) *Canvas {
	return &Canvas{
		buffer: surfacepkg.NewBufferAlpha(width, height),
		alpha:  true,
	}
}

func surfaceBuffer(canvas *Canvas) *surfacepkg.Buffer {
	if canvas == nil {
		return nil
	}
	return canvas.buffer
}

func (canvas *Canvas) Width() int {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return 0
	}
	return raw.Width()
}

func (canvas *Canvas) Height() int {
	raw := surfaceBuffer(canvas)
	if raw == nil {
		return 0
	}
	return raw.Height()
}

func (canvas *Canvas) HasAlpha() bool {
	if canvas == nil {
		return false
	}
	if canvas.buffer != nil {
		return canvas.buffer.HasAlpha()
	}
	return canvas.alpha
}

func (canvas *Canvas) ClipRect() (Rect, bool) {
	if canvas == nil || !canvas.clip.set {
		return Rect{}, false
	}
	return canvas.clip.rect, true
}

func (canvas *Canvas) Resize(width int, height int) {
	if canvas == nil {
		return
	}
	if canvas.buffer == nil {
		if canvas.alpha {
			canvas.buffer = surfacepkg.NewBufferAlpha(width, height)
		} else {
			canvas.buffer = surfacepkg.NewBuffer(width, height)
		}
	} else {
		canvas.buffer.Resize(width, height)
	}
	canvas.alpha = canvas.buffer != nil && canvas.buffer.HasAlpha()
	canvas.clip = clipState{}
	canvas.clipStack = nil
}

func (canvas *Canvas) PushClip(rect Rect) {
	if canvas == nil {
		return
	}
	raw := surfaceBuffer(canvas)
	if raw != nil {
		raw.PushClip(rect)
	}
	current := canvas.clip
	canvas.clipStack = append(canvas.clipStack, canvas.clip)
	if rect.Empty() {
		canvas.clip = clipState{rect: Rect{}, set: true}
		return
	}
	canvasRect := Rect{X: 0, Y: 0, Width: canvas.Width(), Height: canvas.Height()}
	clip := IntersectRect(rect, canvasRect)
	if current.set {
		clip = IntersectRect(clip, current.rect)
	}
	canvas.clip = clipState{rect: clip, set: true}
}

func (canvas *Canvas) PopClip() {
	if canvas == nil {
		return
	}
	raw := surfaceBuffer(canvas)
	if raw != nil {
		raw.PopClip()
	}
	if len(canvas.clipStack) == 0 {
		canvas.clip = clipState{}
		return
	}
	last := canvas.clipStack[len(canvas.clipStack)-1]
	canvas.clipStack = canvas.clipStack[:len(canvas.clipStack)-1]
	canvas.clip = last
}
