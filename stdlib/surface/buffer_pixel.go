package surface

import "kos"

func (buffer *Buffer) HasAlpha() bool {
	raw := rawBuffer(buffer)
	if raw == nil {
		return false
	}
	return raw.HasAlpha()
}

func (buffer *Buffer) PixelValue(x int, y int) uint32 {
	raw := rawBuffer(buffer)
	if raw == nil {
		return 0
	}
	return raw.PixelValue(x, y)
}

func (buffer *Buffer) SetPixelValue(x int, y int, value uint32) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.SetPixelValue(x, y, value)
}

func (buffer *Buffer) BlendPremultipliedPixelValue(x int, y int, value uint32) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.BlendPremultipliedPixelValue(x, y, value)
}

func (buffer *Buffer) SetPixel(x int, y int, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.SetPixel(x, y, uint32(color))
}

func (buffer *Buffer) SetPixelAlpha(x int, y int, color kos.Color, alpha uint8) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.SetPixelAlpha(x, y, uint32(color), alpha)
}

func (buffer *Buffer) DrawLine(x0 int, y0 int, x1 int, y1 int, color kos.Color) {
	raw := rawBuffer(buffer)
	if raw == nil {
		return
	}
	raw.DrawLine(x0, y0, x1, y1, uint32(color))
}
