package surface

import "surface/core"

type Buffer struct {
	raw *core.Buffer
}

func NewBuffer(width int, height int) *Buffer {
	return &Buffer{raw: core.NewBuffer(width, height)}
}

func NewBufferAlpha(width int, height int) *Buffer {
	return &Buffer{raw: core.NewBufferAlpha(width, height)}
}

func rawBuffer(buffer *Buffer) *core.Buffer {
	if buffer == nil {
		return nil
	}
	return buffer.raw
}

func rawGradient(gradient Gradient) core.Gradient {
	return core.Gradient{
		From:      uint32(gradient.From),
		To:        uint32(gradient.To),
		Direction: gradient.Direction,
	}
}

func rawShadow(shadow Shadow) core.Shadow {
	return core.Shadow{
		OffsetX: shadow.OffsetX,
		OffsetY: shadow.OffsetY,
		Blur:    shadow.Blur,
		Color:   uint32(shadow.Color),
		Alpha:   shadow.Alpha,
	}
}

func (buffer *Buffer) Raw() *core.Buffer {
	return rawBuffer(buffer)
}

func (buffer *Buffer) Width() int {
	raw := rawBuffer(buffer)
	if raw == nil {
		return 0
	}
	return raw.Width()
}

func (buffer *Buffer) Height() int {
	raw := rawBuffer(buffer)
	if raw == nil {
		return 0
	}
	return raw.Height()
}

func (buffer *Buffer) Bounds() Rect {
	raw := rawBuffer(buffer)
	if raw == nil {
		return Rect{}
	}
	return raw.Bounds()
}

func (buffer *Buffer) Resize(width int, height int) {
	if buffer == nil {
		return
	}
	if buffer.raw == nil {
		buffer.raw = core.NewBuffer(width, height)
		return
	}
	buffer.raw.Resize(width, height)
}

func (buffer *Buffer) PushClip(rect Rect) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.PushClip(rect)
}

func (buffer *Buffer) PopClip() {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.PopClip()
}
